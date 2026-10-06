import { useState } from 'react'
import { Select, Input, Button, Space, Segmented } from 'antd'
import { PlusOutlined, MinusCircleOutlined } from '@ant-design/icons'
import { useDashboardEnums } from '../hooks/useDashboardEnums'
import { FilterNode, isLeaf, isGroup, createLeaf, createEmptyGroup } from '../types/filterNode'

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

const ENUM_FIELDS = new Set([
  'category', 'business_module', 'sentiment', 'category_status', 'platform',
  'platform_id', 'user_mode', 'app_version', 'app_id',
])

const isInOp = (op: string) => op.toLowerCase() === 'in' || op.toLowerCase() === 'not in'

interface FieldOption {
  key: string
  label: string
}

interface FilterGroupEditorProps {
  value: FilterNode
  onChange: (node: FilterNode) => void
  availableFields: FieldOption[]
  maxDepth?: number
}

/** Recursive tree-based filter condition editor supporting AND/OR nesting. */
export default function FilterGroupEditor({ value, onChange, availableFields, maxDepth = 3 }: FilterGroupEditorProps) {
  const { enumOptions } = useDashboardEnums()

  const updateNode = (updated: FilterNode) => {
    onChange(updated)
  }

  // Immutable update helper: replace a child at index
  const updateChild = (parent: FilterNode, index: number, newChild: FilterNode): FilterNode => {
    const children = [...(parent.children ?? [])]
    children[index] = newChild
    return { ...parent, children }
  }

  // Immutable update helper: remove a child at index
  const removeChild = (parent: FilterNode, index: number): FilterNode => {
    const children = [...(parent.children ?? [])]
    children.splice(index, 1)
    return { ...parent, children }
  }

  // Immutable update helper: add a child
  const addChild = (parent: FilterNode, child: FilterNode): FilterNode => {
    return { ...parent, children: [...(parent.children ?? []), child] }
  }

  const fieldOptions = availableFields.map((f) => ({ value: f.key, label: f.label }))

  return (
    <div>
      {/* Group header */}
      {isGroup(value) && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Segmented
            options={[
              { value: 'and', label: 'AND (且)' },
              { value: 'or', label: 'OR (或)' },
            ]}
            value={value.type}
            onChange={(v) => updateNode({ ...value, type: v as 'and' | 'or' })}
          />
          <Space>
            <Button
              type="dashed"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => updateNode(addChild(value, createLeaf()))}
            >
              添加条件
            </Button>
            {maxDepth > 1 && (
              <Button
                type="dashed"
                size="small"
                onClick={() => updateNode(addChild(value, createEmptyGroup()))}
              >
                添加组合
              </Button>
            )}
          </Space>
        </div>
      )}

      {/* Children */}
      {(value.children ?? []).map((child, i) => (
        <div key={i} style={{ marginBottom: 8 }}>
          {isLeaf(child) ? (
            <LeafEditor
              node={child}
              fieldOptions={fieldOptions}
              enumOpts={enumOptions}
              onChange={(updated) => updateNode(updateChild(value, i, updated))}
              onRemove={() => updateNode(removeChild(value, i))}
            />
          ) : isGroup(child) ? (
            <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, padding: 12, marginLeft: 0 }}>
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                <Button
                  size="small"
                  icon={<MinusCircleOutlined />}
                  danger
                  onClick={() => updateNode(removeChild(value, i))}
                />
              </div>
              <FilterGroupEditor
                value={child}
                onChange={(updated) => updateNode(updateChild(value, i, updated))}
                availableFields={availableFields}
                maxDepth={maxDepth - 1}
              />
            </div>
          ) : null}
        </div>
      ))}

      {/* Empty state for root */}
      {isGroup(value) && (value.children ?? []).length === 0 && (
        <div style={{ color: '#999', fontSize: 13, textAlign: 'center', padding: 12 }}>
          点击上方按钮添加筛选条件
        </div>
      )}
    </div>
  )
}

/** Leaf node editor: field → operator → value */
interface LeafEditorProps {
  node: FilterNode
  fieldOptions: { value: string; label: string }[]
  enumOpts: (field: string) => { value: string; label: string }[]
  onChange: (node: FilterNode) => void
  onRemove: () => void
}

function LeafEditor({ node, fieldOptions, enumOpts, onChange, onRemove }: LeafEditorProps) {
  const [field, setField] = useState(node.field ?? '')
  const [op, setOp] = useState(node.op ?? '=')
  const [val, setVal] = useState<string | string[]>(node.value ?? '')

  // Sync external changes
  const syncChange = (newField: string, newOp: string, newValue: string | string[]) => {
    onChange({ field: newField, op: newOp, value: newValue })
  }

  const handleFieldChange = (newField: string) => {
    setField(newField)
    // Reset value when field changes
    const isIn = isInOp(op)
    const newVal = isIn ? [] : ''
    setVal(newVal)
    syncChange(newField, op, newVal)
  }

  const handleOpChange = (newOp: string) => {
    const wasIn = isInOp(op)
    const nowIn = isInOp(newOp)
    let newVal = val
    if (nowIn && !wasIn) {
      newVal = val ? [String(val)] : []
    } else if (!nowIn && wasIn) {
      newVal = Array.isArray(val) && val.length > 0 ? val[0] : ''
    }
    setOp(newOp)
    setVal(newVal)
    syncChange(field, newOp, newVal)
  }

  const handleValueChange = (newVal: string | string[]) => {
    setVal(newVal)
    syncChange(field, op, newVal)
  }

  const hasEnum = ENUM_FIELDS.has(field) && enumOpts(field).length > 0

  return (
    <Space style={{ display: 'flex', flexWrap: 'wrap' }} align="baseline">
      <Select
        style={{ width: 140 }}
        options={fieldOptions}
        placeholder="选择字段"
        value={field}
        allowClear
        onChange={handleFieldChange}
      />
      {field && (
        <Select
          style={{ width: 100 }}
          options={FILTER_OP_OPTIONS}
          value={op}
          onChange={handleOpChange}
        />
      )}
      {field && op && (
        hasEnum ? (
          isInOp(op) ? (
            <Select
              mode="multiple"
              style={{ width: 200 }}
              options={enumOpts(field)}
              placeholder="选择值"
              value={Array.isArray(val) ? val : []}
              onChange={handleValueChange}
            />
          ) : (
            <Select
              style={{ width: 200 }}
              options={enumOpts(field)}
              placeholder="选择值"
              allowClear
              value={Array.isArray(val) ? (val[0] ?? '') : val}
              onChange={(v) => handleValueChange(v ?? '')}
            />
          )
        ) : (
          <Input
            style={{ width: 200 }}
            placeholder="输入值"
            value={Array.isArray(val) ? val.join(', ') : val}
            onChange={(e) => handleValueChange(e.target.value)}
          />
        )
      )}
      <Button icon={<MinusCircleOutlined />} type="text" danger onClick={onRemove} />
    </Space>
  )
}