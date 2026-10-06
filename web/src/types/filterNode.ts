/**
 * FilterNode — shared filter condition tree type.
 * Supports recursive AND/OR nesting for filter conditions.
 * Leaf node: { field, op, value }
 * Group node: { type: "and" | "or", children: [...] }
 *
 * Used by: dashboard filters, feedback list queries, and any future filter UIs.
 */
export interface FilterNode {
  type?: 'and' | 'or'
  field?: string
  op?: string
  value?: string | string[]
  children?: FilterNode[]
}

/** Returns true if the node is a leaf (field+op+value). */
export function isLeaf(node: FilterNode): boolean {
  // A node is considered leaf as long as it is not an AND/OR group.
  return !isGroup(node)
}

/** Returns true if the node is a group (AND/OR with children). */
export function isGroup(node: FilterNode): boolean {
  return node.type === 'and' || node.type === 'or'
}

/** Normalize raw filter data (legacy flat array or tree) into a FilterNode tree. */
export function normalizeFilterTree(raw: unknown): FilterNode | null {
  if (!raw) return null

  // Already a tree
  if (typeof raw === 'object' && raw !== null) {
    const obj = raw as Record<string, unknown>
    if (obj.type === 'and' || obj.type === 'or') {
      return obj as FilterNode
    }

    // Legacy flat array: [{field, op, value}]
    if (Array.isArray(raw)) {
      const arr = raw as Array<Record<string, unknown>>
      if (arr.length === 0) return null
      return {
        type: 'and',
        children: arr.map((item) => ({
          field: String(item.field ?? ''),
          op: String(item.op ?? '='),
          value: item.value as string | string[],
        })),
      }
    }
  }

  return null
}

/** Create an empty AND group root node. */
export function createEmptyGroup(): FilterNode {
  return { type: 'and', children: [] }
}

/** Create a leaf node. */
export function createLeaf(field: string = '', op: string = '=', value: string | string[] = ''): FilterNode {
  return { field, op, value }
}