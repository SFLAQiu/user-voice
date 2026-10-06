import { useState, useEffect } from 'react'
import { Spin } from 'antd'
import FilterGroupEditor from './FilterGroupEditor'
import { listMetricDimensions, type MetricDimensionConfig } from '../api/metricDimension'
import { FilterNode, isGroup, createEmptyGroup } from '../types/filterNode'
import { FALLBACK_FIELDS } from './DashboardFilterEditor'

interface FieldOption {
  key: string
  label: string
}

interface PanelFilterTreeEditorProps {
  value?: FilterNode | null
  onChange?: (node: FilterNode | null) => void
}

export default function PanelFilterTreeEditor({ value, onChange }: PanelFilterTreeEditorProps) {
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
