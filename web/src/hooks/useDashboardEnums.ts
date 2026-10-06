import { useState, useEffect } from 'react'
import { aggregatedEnums } from '../api/enumConfig'
import { getClassifierEnums } from '../api/classifierConfig'
import { SENTIMENTS } from '../constants/enums'
import { FilterNode, isLeaf, isGroup } from '../types/filterNode'

type DynamicEnums = Record<string, Record<string, string>>

const SYSTEM_ENUMS: Record<string, Record<string, string>> = {
  sentiment: Object.fromEntries(SENTIMENTS.map((s) => [s.value, s.label])),
}

/**
 * Loads merged enum maps (enum_configs + classifier enums + system enums)
 * and provides a resolver to convert raw enum values to display labels.
 */
export function useDashboardEnums() {
  const [enums, setEnums] = useState<DynamicEnums>({})

  useEffect(() => {
    Promise.all([
      aggregatedEnums().catch(() => null),
      getClassifierEnums().catch(() => null),
    ]).then(([enumRes, clfRes]) => {
      const merged: DynamicEnums = { ...SYSTEM_ENUMS }
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
      setEnums(merged)
    })
  }, [])

  /** Convert raw enum values to their display labels for a given field. */
  const resolveEnumLabels = (field: string, values: string[]): string => {
    const map = enums[field]
    if (!map) return values.join(', ')
    return values.map((v) => map[v] ?? v).join(', ')
  }

  /** Build Select options for a given enum field. */
  const enumOptions = (field: string): { value: string; label: string }[] => {
    const map = enums[field]
    if (!map) return []
    return Object.entries(map).map(([k, v]) => ({ value: k, label: String(v) }))
  }

  /** Recursively convert a FilterNode tree into a human-readable label string. */
  const resolveFilterNodeLabels = (
    node: FilterNode | null | undefined,
    fieldLabels?: Record<string, string>,
  ): string => {
    if (!node) return ''
    if (isLeaf(node)) {
      const label = fieldLabels?.[node.field ?? ''] ?? node.field ?? ''
      const values = Array.isArray(node.value) ? node.value : [String(node.value ?? '')]
      const displayValues = resolveEnumLabels(node.field ?? '', values)
      return `${label} ${node.op ?? '='} ${displayValues}`
    }
    if (isGroup(node) && node.children?.length) {
      const joiner = node.type === 'or' ? ' OR ' : ' AND '
      const childLabels = node.children
        .map((c) => resolveFilterNodeLabels(c, fieldLabels))
        .filter(Boolean)
      if (childLabels.length === 0) return ''
      if (childLabels.length === 1) return childLabels[0]
      return '(' + childLabels.join(joiner) + ')'
    }
    return ''
  }

  return { enums, resolveEnumLabels, resolveFilterNodeLabels, enumOptions }
}