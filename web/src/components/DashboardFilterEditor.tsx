import { useState, useEffect } from 'react'
import { Spin } from 'antd'
import FilterGroupEditor from './FilterGroupEditor'
import { listMetricDimensions, type MetricDimensionConfig } from '../api/metricDimension'
import { FilterNode, normalizeFilterTree, createEmptyGroup } from '../types/filterNode'

export type { FilterNode } from '../types/filterNode'

interface FieldOption {
  key: string
  label: string
}

// 维度配置加载失败时的兜底字段（不含 user_mode 等非维度字段）
const FALLBACK_FIELDS: FieldOption[] = [
  { key: 'app_id', label: '应用' },
  { key: 'platform_id', label: '平台' },
  { key: 'app_version', label: '版本号' },
  { key: 'category', label: '分类' },
  { key: 'sentiment', label: '情感倾向' },
  { key: 'business_module', label: '业务模块' },
  { key: 'category_status', label: '分类状态' },
]

interface DashboardFilterEditorProps {
  value?: FilterNode | null
  onChange?: (node: FilterNode | null) => void
}

export default function DashboardFilterEditor({ value, onChange }: DashboardFilterEditorProps) {
  const [fields, setFields] = useState<FieldOption[]>(FALLBACK_FIELDS)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listMetricDimensions()
      .then((configs: MetricDimensionConfig[] | null) => {
        if (configs && configs.length > 0) {
          setFields(configs.map((dc) => ({ key: dc.field, label: dc.label })))
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const rootNode = value ?? createEmptyGroup()

  const handleChange = (node: FilterNode) => {
    if (isGroup(node) && (node.children ?? []).length === 0) {
      onChange?.(null)
    } else {
      onChange?.(node)
    }
  }

  if (loading) {
    return <Spin size="small" />
  }

  return (
    <FilterGroupEditor
      value={rootNode}
      onChange={handleChange}
      availableFields={fields}
    />
  )
}

function isGroup(node: FilterNode): boolean {
  return node.type === 'and' || node.type === 'or'
}

export { FALLBACK_FIELDS, normalizeFilterTree, createEmptyGroup }
