import { useState, useEffect, useCallback } from 'react'
import { Segmented, Select, Input, Tooltip, Spin } from 'antd'
import { QuestionCircleOutlined, LockOutlined } from '@ant-design/icons'
import { listMetricDimensions, type MetricDimensionConfig } from '../api/metricDimension'
import PanelFilterTreeEditor from './PanelFilterTreeEditor'
import { FilterNode } from '../types/filterNode'

// ── 常量 ──

const METRIC_OPTIONS = [
  { value: 'count', label: '计数' },
  { value: 'sum', label: '求和' },
]

const BUCKET_OPTIONS = [
  { value: '1m', label: '按分钟 (1m)' },
  { value: '1h', label: '按小时 (1h)' },
  { value: '1d', label: '按天 (1d)' },
]

const DEFAULT_QUERY_CONFIG: Record<string, unknown> = {
  metric: 'count',
  x_dimension: { field: 'original_created_at', bucket: '1m' },
  time_range: { type: 'relative', value: '30d' },
  panel_filter_tree: null,
}

const MODE_OPTIONS = [
  { value: 'form', label: '表单配置' },
  { value: 'json', label: 'JSON 编辑' },
]

const HINTS: Record<string, string> = {
  metric: '计数(count)基于预聚合时间桶查询时间序列，求和(sum)汇总总数',
  x_dimension_field: '时间轴固定为反馈提交时间，可选分钟或小时粒度',
  bucket: '分钟粒度适合短时间范围，小时粒度适合长时间范围趋势',
  display_field: '选择饼图的分组维度字段，数据将按此维度聚合展示',
  pie_time_locked: '饼图时间维度已锁定为最小粒度 (1m)，无需手动选择',
  number_time_locked: '数值面板时间维度已锁定为最小粒度 (1m)，对 feedback_count 求和',
  panel_filter: '数值面板支持 AND/OR 嵌套条件筛选，字段来自预聚合维度',
}

function HintLabel({ field, text }: { field: string; text: string }) {
  const hint = HINTS[field]
  return hint ? (
    <Tooltip title={hint}>
      {text} <QuestionCircleOutlined style={{ color: '#999', fontSize: 11, marginLeft: 2, cursor: 'help' }} />
    </Tooltip>
  ) : text
}

// ── 类型 ──

interface QueryConfigEditorProps {
  value?: Record<string, unknown>
  onChange?: (v: Record<string, unknown>) => void
  chartType?: string
}

type EditMode = 'form' | 'json'

// ── 组件 ──

export default function QueryConfigEditor({ value, onChange, chartType }: QueryConfigEditorProps) {
  const [mode, setMode] = useState<EditMode>('form')
  const [jsonText, setJsonText] = useState<string>('')
  const [jsonError, setJsonError] = useState<string>('')
  const [dimensionConfigs, setDimensionConfigs] = useState<MetricDimensionConfig[]>([])
  const [dimensionsLoading, setDimensionsLoading] = useState(false)

  const isPie = chartType === 'pie'
  const isNumber = chartType === 'number'

  // 加载维度配置
  useEffect(() => {
    if (!isPie) return
    setDimensionsLoading(true)
    listMetricDimensions()
      .then((configs) => setDimensionConfigs(configs ?? []))
      .catch(() => setDimensionConfigs([]))
      .finally(() => setDimensionsLoading(false))
  }, [isPie])

  // 从外部 value 初始化内部状态
  useEffect(() => {
    if (value) {
      setJsonText(JSON.stringify(value, null, 2))
      setJsonError('')
    }
  }, [value])

  // chartType 变化时自动调整 query_config
  useEffect(() => {
    if (!value) return
    const patch: Record<string, unknown> = {}
    if (isPie) {
      // 饼图锁定 1m 粒度
      const currentBucket = (value?.x_dimension as { bucket?: string })?.bucket
      if (currentBucket !== '1m') {
        patch.x_dimension = { field: 'original_created_at', bucket: '1m' }
      }
      // 确保 group_by 存在
      if (!value?.group_by) {
        patch.group_by = []
      }
    } else if (isNumber) {
      // 数值面板锁定 1m 粒度 + metric=sum
      const currentBucket = (value?.x_dimension as { bucket?: string })?.bucket
      if (currentBucket !== '1m') {
        patch.x_dimension = { field: 'original_created_at', bucket: '1m' }
      }
      if (value?.metric !== 'sum') {
        patch.metric = 'sum'
      }
    } else {
      // 非饼图/数值时清理 group_by
      if (value?.group_by && Array.isArray(value.group_by) && value.group_by.length > 0) {
        patch.group_by = []
      }
    }
    if (Object.keys(patch).length > 0) {
      onChange?.({ ...value, ...patch })
    }
  }, [isPie, isNumber]) // eslint-disable-line react-hooks/exhaustive-deps

  // 表单修改 → 同步到 JSON 文本
  const handleFormChange = useCallback(
    (patch: Record<string, unknown>) => {
      if (!value) return
      const next = { ...value, ...patch }
      onChange?.(next)
      setJsonText(JSON.stringify(next, null, 2))
      setJsonError('')
    },
    [value, onChange],
  )

  // JSON 文本修改 → 解析后同步到表单
  const handleJsonChange = useCallback(
    (text: string) => {
      setJsonText(text)
      try {
        const parsed = JSON.parse(text)
        if (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) {
          onChange?.(parsed as Record<string, unknown>)
          setJsonError('')
        } else {
          setJsonError('JSON 必须是对象格式')
        }
      } catch {
        setJsonError('JSON 格式错误')
      }
    },
    [onChange],
  )

  // ── 安全取值辅助 ──

  // ── 样式 ──

  const rowStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    marginBottom: 8,
  }

  return (
    <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, padding: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <span style={{ fontWeight: 500 }}>查询配置</span>
        <Segmented options={MODE_OPTIONS} value={mode} onChange={(v) => setMode(v as EditMode)} />
      </div>

      {mode === 'form' ? (
        <>
          {/* 指标 */}
          <div style={rowStyle}>
            <HintLabel field="metric" text="指标" />
            {isNumber ? (
              <span style={{ fontSize: 13 }}>
                总数求和 (sum)
                <LockOutlined style={{ fontSize: 11, marginLeft: 4, color: '#999' }} />
              </span>
            ) : (
              <span style={{ fontSize: 13 }}>总数计数 (count)</span>
            )}
          </div>

          {/* 饼图：展示字段配置 */}
          {isPie && (
            <div style={rowStyle}>
              <HintLabel field="display_field" text="展示字段" />
              {dimensionsLoading ? (
                <Spin size="small" />
              ) : (
                <Select
                  style={{ width: 200 }}
                  placeholder="选择分组维度"
                  value={((value?.group_by as string[]) ?? [])[0] ?? undefined}
                  options={dimensionConfigs.map((dc) => ({ value: dc.field, label: dc.label }))}
                  onChange={(field) => handleFormChange({ group_by: field ? [field] : [] })}
                  allowClear
                />
              )}
            </div>
          )}

          {/* 数值面板：筛选条件 AND/OR 组合 */}
          {isNumber && (
            <div style={{ marginTop: 8 }}>
              <HintLabel field="panel_filter" text="筛选条件" />
              <div style={{ marginTop: 4 }}>
                <PanelFilterTreeEditor
                  value={(value?.panel_filter_tree as FilterNode) ?? null}
                  onChange={(node) => handleFormChange({ panel_filter_tree: node ?? undefined })}
                />
              </div>
            </div>
          )}

          {/* 时间维度 */}
          <div style={rowStyle}>
            <HintLabel field={isPie ? 'pie_time_locked' : isNumber ? 'number_time_locked' : 'x_dimension_field'} text="时间维度" />
            {isPie || isNumber ? (
              <span style={{ fontSize: 13 }}>
                反馈提交时间
                <LockOutlined style={{ fontSize: 11, marginLeft: 4, color: '#999' }} />
                <span style={{ fontSize: 12, color: '#999', marginLeft: 4 }}>1m (最小粒度)</span>
              </span>
            ) : (
              <>
                <span style={{ fontSize: 13 }}>反馈提交时间</span>
                <HintLabel field="bucket" text="粒度" />
                <Select
                  style={{ width: 140 }}
                  options={BUCKET_OPTIONS}
                  value={(value?.x_dimension as { bucket?: string })?.bucket || '1m'}
                  onChange={(v) => handleFormChange({ x_dimension: { field: 'original_created_at', bucket: v } })}
                />
              </>
            )}
          </div>
        </>
      ) : (
        <>
          <Input.TextArea
            rows={12}
            value={jsonText}
            onChange={(e) => handleJsonChange(e.target.value)}
            style={{ fontFamily: 'monospace', fontSize: 12 }}
            status={jsonError ? 'error' : undefined}
          />
          {jsonError && <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>{jsonError}</div>}
        </>
      )}
    </div>
  )
}

export { DEFAULT_QUERY_CONFIG }