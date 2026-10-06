import { useState, useEffect, useCallback, useRef } from 'react'
import { Select, Input, Space, Button, Tooltip } from 'antd'
import { MinusCircleOutlined, PlusOutlined, QuestionCircleOutlined } from '@ant-design/icons'
import { aggregatedEnums } from '../api/enumConfig'
import { getClassifierEnums } from '../api/classifierConfig'
import { listMetricDimensions, type MetricDimensionConfig } from '../api/metricDimension'
import { SENTIMENTS } from '../constants/enums'

type DynamicEnums = Record<string, Record<string, string>>

// 兜底字段列表（维度配置 API 不可用时）
const FALLBACK_QUERYABLE_FIELDS = [
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
  { value: 'in', label: 'in' },
  { value: 'not in', label: 'not in' },
]

interface FilterItem {
  field: string
  op: string
  value: string | string[]
}

interface AlertFilterEditorProps {
  value?: string
  onChange?: (v: string) => void
}

function parseFilters(jsonStr: string): FilterItem[] {
  try {
    const obj = JSON.parse(jsonStr)
    const filters = (obj.filters ?? []) as { field: string; op: string; value: unknown }[]
    return filters.map((f) => {
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
    })
  } catch {
    return []
  }
}

function filtersToJson(filters: FilterItem[]): string {
  const obj: Record<string, unknown> = {}
  if (filters.length > 0) {
    obj.filters = filters
      .filter((f) => f.field && f.op)
      .map((f) => {
        const opLower = f.op.toLowerCase()
        if (opLower === 'in' || opLower === 'not in') {
          return { field: f.field, op: f.op, value: Array.isArray(f.value) ? f.value : [f.value] }
        }
        return { field: f.field, op: f.op, value: f.value }
      })
  }
  return JSON.stringify(obj)
}

const isInOp = (op: string) => op.toLowerCase() === 'in' || op.toLowerCase() === 'not in'

export default function AlertFilterEditor({ value, onChange }: AlertFilterEditorProps) {
  const [filters, setFilters] = useState<FilterItem[]>([])
  const [enums, setEnums] = useState<DynamicEnums>({})
  const [queryableFields, setQueryableFields] = useState(FALLBACK_QUERYABLE_FIELDS)
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

  useEffect(() => {
    Promise.all([
      aggregatedEnums().catch(() => null),
      getClassifierEnums().catch(() => null),
    ]).then(([enumRes, clfRes]) => {
      const merged: DynamicEnums = {}
      if (enumRes && Object.keys(enumRes).length > 0) {
        Object.assign(merged, enumRes)
      }
      if (clfRes) {
        if (clfRes.categories?.length) {
          merged.category = Object.fromEntries(clfRes.categories.map((c) => [c, c]))
        }
        if (clfRes.business_modules?.length) {
          merged.business_module = Object.fromEntries(clfRes.business_modules.map((m) => [m, m]))
        }
      }
      merged.sentiment = Object.fromEntries(SENTIMENTS.map((s) => [s.value, s.label]))
      if (Object.keys(merged).length > 0) setEnums(merged)
    })
  }, [])

  useEffect(() => {
    if (selfUpdateRef.current) {
      selfUpdateRef.current = false
      return
    }
    if (value) {
      setFilters(parseFilters(value))
    }
  }, [value])

  const updateFilters = useCallback(
    (next: FilterItem[]) => {
      selfUpdateRef.current = true
      setFilters(next)
      onChange?.(filtersToJson(next))
    },
    [onChange],
  )

  const enumOptions = (field: string): { value: string; label: string }[] => {
    const map = enums[field]
    if (!map) return []
    return Object.entries(map).map(([k, v]) => ({ value: k, label: String(v) }))
  }

  return (
    <div>
      <div style={{ marginBottom: 4, fontSize: 12, color: '#666' }}>
        <Tooltip title="按桶维度字段值筛选告警数据源，支持等值和多值匹配。可选项，不填则查询全量数据">
          过滤条件 <QuestionCircleOutlined style={{ color: '#999', fontSize: 11 }} />
        </Tooltip>
      </div>
      {filters.map((f, i) => (
        <Space key={i} style={{ marginBottom: 8, display: 'flex' }} align="baseline">
          <Select
            style={{ width: 160 }}
            options={queryableFields}
            placeholder="字段"
            value={f.field}
            onChange={(v) => {
              const next = filters.map((item, idx) => idx === i ? { ...item, field: v, value: isInOp(item.op) ? [] : '' } : item)
              updateFilters(next)
            }}
          />
          <Select
            style={{ width: 90 }}
            options={FILTER_OP_OPTIONS}
            placeholder="操作"
            value={f.op}
            onChange={(v) => {
              let newValue = f.value
              if (isInOp(v) && !Array.isArray(f.value)) {
                newValue = f.value ? [f.value] : []
              } else if (!isInOp(v) && Array.isArray(f.value)) {
                newValue = f.value.length > 0 ? f.value[0] : ''
              }
              const next = filters.map((item, idx) => idx === i ? { ...item, op: v, value: newValue } : item)
              updateFilters(next)
            }}
          />
          {ENUM_FIELDS.has(f.field) && enumOptions(f.field).length > 0 ? (
            isInOp(f.op) ? (
              <Select
                mode="multiple"
                style={{ width: 180 }}
                options={enumOptions(f.field)}
                placeholder="选择值"
                value={Array.isArray(f.value) ? f.value : []}
                onChange={(v) => {
                  const next = filters.map((item, idx) => idx === i ? { ...item, value: v } : item)
                  updateFilters(next)
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
                  const next = filters.map((item, idx) => idx === i ? { ...item, value: v ?? '' } : item)
                  updateFilters(next)
                }}
              />
            )
          ) : (
            <Input
              style={{ width: 180 }}
              placeholder="值"
              value={Array.isArray(f.value) ? f.value.join(', ') : f.value}
              onChange={(e) => {
                const next = filters.map((item, idx) => idx === i ? { ...item, value: e.target.value } : item)
                updateFilters(next)
              }}
            />
          )}
          <Button
            icon={<MinusCircleOutlined />}
            type="text"
            danger
            onClick={() => updateFilters(filters.filter((_, idx) => idx !== i))}
          />
        </Space>
      ))}
      <Button
        type="dashed"
        icon={<PlusOutlined />}
        size="small"
        onClick={() => updateFilters([...filters, { field: '', op: '=', value: '' }])}
      >
        添加条件
      </Button>
    </div>
  )
}