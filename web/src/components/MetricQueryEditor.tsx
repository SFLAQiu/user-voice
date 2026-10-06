import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Segmented,
  Select,
  Input,
  InputNumber,
  Radio,
  DatePicker,
  Space,
  Button,
  Tooltip,
  message,
} from 'antd'
import { MinusCircleOutlined, PlusOutlined, QuestionCircleOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import JsonEditor from './JsonEditor'
import { aggregatedEnums } from '../api/enumConfig'
import { getClassifierEnums } from '../api/classifierConfig'
import { listMetricDimensions, type MetricDimensionConfig } from '../api/metricDimension'
import { SENTIMENTS } from '../constants/enums'

type DynamicEnums = Record<string, Record<string, string>>

// ── Constants ──

const METRIC_OPTIONS = [
  { value: 'count', label: '总数 (count)' },
]

const RELATIVE_TIME_OPTIONS = [
  { value: '1m', label: '1 分钟' },
  { value: '5m', label: '5 分钟' },
  { value: '10m', label: '10 分钟' },
  { value: '30m', label: '30 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '3h', label: '3 小时' },
  { value: '6h', label: '6 小时' },
  { value: '12h', label: '12 小时' },
  { value: '1d', label: '1 天' },
  { value: '7d', label: '7 天' },
  { value: '30d', label: '30 天' },
]

// 兜底字段列表（维度配置 API 不可用时）
const FALLBACK_queryableFields = [
  { value: 'app_id', label: '应用ID (app_id)' },
  { value: 'platform_id', label: '平台ID (platform_id)' },
  { value: 'app_version', label: '版本号 (app_version)' },
  { value: 'category', label: '分类 (category)' },
  { value: 'business_module', label: '业务模块 (business_module)' },
  { value: 'sentiment', label: '情感 (sentiment)' },
  { value: 'category_status', label: '分类状态 (category_status)' },
]

const ENUM_FIELDS = new Set([
  'category', 'business_module', 'sentiment', 'category_status',
  'platform_id', 'app_id', 'app_version',
])

const FILTER_OP_OPTIONS = [
  { value: '=', label: '=' },
  { value: '!=', label: '!=' },
  { value: '>', label: '>' },
  { value: '>=', label: '>=' },
  { value: '<', label: '<' },
  { value: '<=', label: '<=' },
  { value: 'in', label: 'in' },
  { value: 'not in', label: 'not in' },
  { value: 'like', label: 'like' },
]


const BUCKET_OPTIONS = [
  { value: '1m', label: '按分钟 (1m)' },
  { value: '1h', label: '按小时 (1h)' },
  { value: '1d', label: '按天 (1d)' },
]


const FIELD_HINTS: Record<string, string> = {
  metric: '仅支持总数计数，基于预聚合时间桶查询',
  time_range: '限定查询的时间窗口。相对时间如 1h/7d/30d，绝对时间指定起止日期',
  filters: '按桶维度字段值筛选，支持多种比较操作。in/not in 支持多选值',
  group_by: '按桶维度字段对结果分组聚合',
  x_dimension: '时间轴固定为反馈提交时间，可选分钟或小时粒度',
  limit: '限制返回的最大记录数，默认 1000',
}

// ── Types ──

interface FilterItem {
  field: string
  op: string
  value: string | string[]
}

interface MetricQueryFormState {
  metric: string
  time_range_type: string
  time_range_value?: string
  time_range_start?: string
  time_range_end?: string
  filters: FilterItem[]
  group_by: string[]
  bucket?: string
  limit?: number
}

interface MetricQueryEditorProps {
  value?: string
  onChange?: (v: string) => void
}

// ── Conversion ──

function formStateToJson(state: MetricQueryFormState): string {
  const obj: Record<string, unknown> = { metric: 'count' }
  if (state.time_range_type === 'relative' && state.time_range_value) {
    obj.time_range = { type: 'relative', value: state.time_range_value }
  } else if (state.time_range_type === 'absolute' && state.time_range_start) {
    obj.time_range = { type: 'absolute', start: state.time_range_start, end: state.time_range_end || '' }
  }
  if (state.filters.length > 0) {
    obj.filters = state.filters
      .filter((f) => f.field && f.op)
      .map((f) => {
        const opLower = f.op.toLowerCase()
        if (opLower === 'in' || opLower === 'not in') {
          return { field: f.field, op: f.op, value: Array.isArray(f.value) ? f.value : [f.value] }
        }
        return { field: f.field, op: f.op, value: f.value }
      })
  }
  if (state.group_by.length > 0) {
    obj.group_by = state.group_by
  }
  // x_dimension 固定为反馈提交时间，bucket 由用户选择
  obj.x_dimension = { field: 'original_created_at', bucket: state.bucket || '1m' }
  if (state.limit && state.limit > 0) {
    obj.limit = state.limit
  }
  return JSON.stringify(obj, null, 2)
}

function jsonToFormState(jsonStr: string): MetricQueryFormState | null {
  try {
    const obj = JSON.parse(jsonStr)
    const filters = (obj.filters ?? []) as { field: string; op: string; value: unknown }[]
    const tr = obj.time_range ?? {}
    return {
      metric: 'count',
      time_range_type: tr.type ?? 'relative',
      time_range_value: tr.type === 'relative' ? tr.value : undefined,
      time_range_start: tr.type === 'absolute' ? tr.start : undefined,
      time_range_end: tr.type === 'absolute' ? tr.end : undefined,
      filters: filters.map((f) => {
        const opLower = (f.op ?? '=').toLowerCase()
        const rawVal = f.value ?? ''
        if (opLower === 'in' || opLower === 'not in') {
          return {
            field: f.field ?? '',
            op: f.op ?? 'in',
            value: Array.isArray(rawVal) ? rawVal.map(String) : [String(rawVal)],
          }
        }
        return { field: f.field ?? '', op: f.op ?? '=', value: String(rawVal) }
      }),
      group_by: obj.group_by ?? [],
      bucket: obj.x_dimension?.bucket ?? '1m',
      limit: obj.limit,
    }
  } catch {
    return null
  }
}

const DEFAULT_FORM_STATE: MetricQueryFormState = {
  metric: 'count',
  time_range_type: 'relative',
  time_range_value: '1h',
  filters: [],
  group_by: [],
  bucket: '1m',
}

// ── Label with Tooltip helper ──

function FieldLabel({ field, text }: { field: string; text: string }) {
  const hint = FIELD_HINTS[field]
  return (
    <div style={{ marginBottom: 4, fontSize: 12, color: '#666' }}>
      {hint ? (
        <Tooltip title={hint}>
          <span style={{ cursor: 'help' }}>{text} <QuestionCircleOutlined style={{ color: '#999', fontSize: 11 }} /></span>
        </Tooltip>
      ) : text}
    </div>
  )
}

// ── Component ──

export default function MetricQueryEditor({ value, onChange }: MetricQueryEditorProps) {
  const [mode, setMode] = useState<'form' | 'json'>('form')
  const [formState, setFormState] = useState<MetricQueryFormState>(DEFAULT_FORM_STATE)
  const [enums, setEnums] = useState<DynamicEnums>({})
  const [queryableFields, setQueryableFields] = useState(FALLBACK_queryableFields)
  const selfUpdateRef = useRef(false)

  // 加载维度配置（动态字段列表）
  useEffect(() => {
    listMetricDimensions()
      .then((configs: MetricDimensionConfig[] | null) => {
        if (configs && configs.length > 0) {
          setQueryableFields(configs.map((dc) => ({ value: dc.field, label: `${dc.label} (${dc.field})` })))
        }
      })
      .catch(() => {})
  }, [])

  // Load enum values on mount (merge enum_configs + classifier enums)
  useEffect(() => {
    Promise.all([
      aggregatedEnums().catch(() => null),
      getClassifierEnums().catch(() => null),
    ]).then(([enumRes, clfRes]) => {
      const merged: DynamicEnums = {}
      if (enumRes && Object.keys(enumRes).length > 0) {
        Object.assign(merged, enumRes)
      }
      // Classifier enums come as arrays; convert to self-mapping { value: value }
      if (clfRes) {
        if (clfRes.categories?.length) {
          merged.category = Object.fromEntries(clfRes.categories.map((c) => [c, c]))
        }
        if (clfRes.business_modules?.length) {
          merged.business_module = Object.fromEntries(clfRes.business_modules.map((m) => [m, m]))
        }
      }
      // System-level sentiment enum (hardcoded, not in enum_configs)
      merged.sentiment = Object.fromEntries(SENTIMENTS.map((s) => [s.value, s.label]))
      if (Object.keys(merged).length > 0) setEnums(merged)
    })
  }, [])

  // Sync: when value changes externally (e.g. form reset), update formState
  // Skip when the change was triggered by our own updateForm to avoid flash
  useEffect(() => {
    if (selfUpdateRef.current) {
      selfUpdateRef.current = false
      return
    }
    if (mode === 'form' && value) {
      const parsed = jsonToFormState(value)
      if (parsed) {
        setFormState(parsed)
      }
    }
  }, [value, mode])

  const updateForm = useCallback(
    (next: MetricQueryFormState) => {
      selfUpdateRef.current = true
      setFormState(next)
      onChange?.(formStateToJson(next))
    },
    [onChange],
  )

  const handleModeChange = (newMode: 'form' | 'json') => {
    if (newMode === 'json') {
      setMode('json')
    } else {
      const parsed = jsonToFormState(value ?? '')
      if (!parsed) {
        message.error('JSON 格式错误，无法切换到表单模式')
        return
      }
      setFormState(parsed)
      setMode('form')
    }
  }

  // Build enum options for a specific field
  const enumOptions = (field: string): { value: string; label: string }[] => {
    const map = enums[field]
    if (!map) return []
    return Object.entries(map).map(([k, v]) => ({ value: k, label: String(v) }))
  }

  const isInOp = (op: string) => op.toLowerCase() === 'in' || op.toLowerCase() === 'not in'

  // ── Render ──

  return (
    <div>
      <div style={{ marginBottom: 12 }}>
        <Segmented
          options={[
            { value: 'form', label: '表单配置' },
            { value: 'json', label: 'JSON 配置' },
          ]}
          value={mode}
          onChange={handleModeChange}
        />
      </div>

      {mode === 'json' && <JsonEditor value={value ?? '{}'} onChange={(v) => onChange?.(v)} />}

      {mode === 'form' && (
        <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, padding: 16 }}>
          {/* Metric */}
          <div style={{ flex: 1 }}>
            <FieldLabel field="metric" text="聚合指标" />
            <Select
              style={{ width: 240 }}
              options={METRIC_OPTIONS}
              value={formState.metric}
              onChange={(v) => updateForm({ ...formState, metric: v })}
            />
          </div>

          {/* Time Range */}
          <div style={{ marginTop: 16 }}>
            <FieldLabel field="time_range" text="时间范围" />
            <Space>
              <Radio.Group
                value={formState.time_range_type}
                onChange={(e) => updateForm({ ...formState, time_range_type: e.target.value })}
                options={[
                  { value: 'relative', label: '相对时间' },
                  { value: 'absolute', label: '绝对时间' },
                ]}
              />
              {formState.time_range_type === 'relative' && (
                <Select
                  style={{ width: 120 }}
                  options={RELATIVE_TIME_OPTIONS}
                  value={formState.time_range_value}
                  onChange={(v) => updateForm({ ...formState, time_range_value: v })}
                />
              )}
              {formState.time_range_type === 'absolute' && (
                <DatePicker.RangePicker
                  showTime
                  style={{ width: 360 }}
                  value={
                    formState.time_range_start && formState.time_range_end
                      ? [dayjs(formState.time_range_start), dayjs(formState.time_range_end)]
                      : undefined
                  }
                  onChange={(dates) => {
                    updateForm({
                      ...formState,
                      time_range_start: dates?.[0]?.format('YYYY-MM-DD HH:mm:ss') ?? '',
                      time_range_end: dates?.[1]?.format('YYYY-MM-DD HH:mm:ss') ?? '',
                    })
                  }}
                />
              )}
            </Space>
          </div>

          {/* Filters */}
          <div style={{ marginTop: 16 }}>
            <FieldLabel field="filters" text="过滤条件" />
            {formState.filters.map((f, i) => (
              <Space key={i} style={{ marginBottom: 8, display: 'flex' }} align="baseline">
                <Select
                  style={{ width: 160 }}
                  options={queryableFields}
                  placeholder="字段"
                  value={f.field}
                  onChange={(v) => {
                    // Reset value when field changes (different field may need different input type)
                    const next = { ...formState, filters: formState.filters.map((item, idx) => idx === i ? { ...item, field: v, value: isInOp(item.op) ? [] : '' } : item) }
                    updateForm(next)
                  }}
                />
                <Select
                  style={{ width: 90 }}
                  options={FILTER_OP_OPTIONS}
                  placeholder="操作"
                  value={f.op}
                  onChange={(v) => {
                    // Convert value type when op switches to/from in
                    let newValue = f.value
                    if (isInOp(v) && !Array.isArray(f.value)) {
                      newValue = f.value ? [f.value] : []
                    } else if (!isInOp(v) && Array.isArray(f.value)) {
                      newValue = f.value.length > 0 ? f.value[0] : ''
                    }
                    const next = { ...formState, filters: formState.filters.map((item, idx) => idx === i ? { ...item, op: v, value: newValue } : item) }
                    updateForm(next)
                  }}
                />
                {/* Value input: enum select or plain input, single or multi based on op */}
                {ENUM_FIELDS.has(f.field) && enumOptions(f.field).length > 0 ? (
                  isInOp(f.op) ? (
                    <Select
                      mode="multiple"
                      style={{ width: 180 }}
                      options={enumOptions(f.field)}
                      placeholder="选择值"
                      value={Array.isArray(f.value) ? f.value : []}
                      onChange={(v) => {
                        const next = { ...formState, filters: formState.filters.map((item, idx) => idx === i ? { ...item, value: v } : item) }
                        updateForm(next)
                      }}
                    />
                  ) : (
                    <Select
                      style={{ width: 180 }}
                      options={enumOptions(f.field)}
                      placeholder="选择值"
                      allowClear
                      value={Array.isArray(f.value) ? f.value[0] ?? '' : f.value}
                      onChange={(v) => {
                        const next = { ...formState, filters: formState.filters.map((item, idx) => idx === i ? { ...item, value: v ?? '' } : item) }
                        updateForm(next)
                      }}
                    />
                  )
                ) : (
                  <Input
                    style={{ width: 180 }}
                    placeholder="值"
                    value={Array.isArray(f.value) ? f.value.join(', ') : f.value}
                    onChange={(e) => {
                      const next = { ...formState, filters: formState.filters.map((item, idx) => idx === i ? { ...item, value: e.target.value } : item) }
                      updateForm(next)
                    }}
                  />
                )}
                <Button
                  icon={<MinusCircleOutlined />}
                  type="text"
                  danger
                  onClick={() => updateForm({ ...formState, filters: formState.filters.filter((_, idx) => idx !== i) })}
                />
              </Space>
            ))}
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              size="small"
              onClick={() => updateForm({ ...formState, filters: [...formState.filters, { field: '', op: '=', value: '' }] })}
            >
              添加条件
            </Button>
          </div>

          {/* Group By */}
          <div style={{ marginTop: 16 }}>
            <FieldLabel field="group_by" text="分组字段" />
            <Select
              mode="multiple"
              style={{ width: '100%' }}
              options={queryableFields}
              placeholder="选择分组字段"
              value={formState.group_by}
              onChange={(v) => updateForm({ ...formState, group_by: v })}
            />
          </div>

          {/* X Dimension — 时间维度选择 */}
          <div style={{ marginTop: 16 }}>
            <FieldLabel field="x_dimension" text="时间维度" />
            <Space>
              <span style={{ fontSize: 13 }}>反馈提交时间</span>
              <Select
                style={{ width: 140 }}
                options={BUCKET_OPTIONS}
                value={formState.bucket || '1m'}
                onChange={(v) => updateForm({ ...formState, bucket: v })}
              />
            </Space>
          </div>

          {/* Limit */}
          <div style={{ marginTop: 16 }}>
            <FieldLabel field="limit" text="结果上限" />
            <InputNumber
              min={1}
              max={10000}
              placeholder="默认 1000"
              style={{ width: 160 }}
              value={formState.limit}
              onChange={(v) => updateForm({ ...formState, limit: v ?? undefined })}
            />
          </div>
        </div>
      )}
    </div>
  )
}