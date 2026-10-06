import { useState, useCallback, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Space,
  Popconfirm,
  Typography,
  Tag,
  message,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  listEnumConfigs,
  createEnumConfig,
  updateEnumConfig,
  deleteEnumConfig,
  type EnumConfig,
} from '../../api/enumConfig'

export default function EnumConfigList() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<EnumConfig[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<EnumConfig | null>(null)
  const [form] = Form.useForm()
  const [labelRows, setLabelRows] = useState<{ key: string; value: string }[]>([])
  const [addKey, setAddKey] = useState('')
  const [addValue, setAddValue] = useState('')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const rows = await listEnumConfigs()
      setData(rows ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setLabelRows([])
    setModalOpen(true)
  }

  const openEdit = (row: EnumConfig) => {
    setEditing(row)
    form.setFieldsValue({ field: row.field })
    setLabelRows(
      Object.entries(row.labels).map(([k, v]) => ({ key: k, value: v })),
    )
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    await deleteEnumConfig(id)
    message.success('已删除')
    fetchData()
  }

  const handleSubmit = async (values: { field: string }) => {
    const labels: Record<string, string> = {}
    for (const r of labelRows) {
      if (r.key && r.value) {
        labels[r.key] = r.value
      }
    }
    const payload = { field: values.field, labels }
    if (editing) {
      await updateEnumConfig(editing.id, payload)
      message.success('已更新')
    } else {
      await createEnumConfig(payload)
      message.success('已创建')
    }
    setModalOpen(false)
    fetchData()
  }

  const removeLabelRow = (index: number) => {
    setLabelRows((prev) => prev.filter((_, i) => i !== index))
  }

  const addLabelRow = () => {
    if (!addKey || !addValue) return
    setLabelRows((prev) => [...prev, { key: addKey, value: addValue }])
    setAddKey('')
    setAddValue('')
  }

  const updateLabelRow = (index: number, field: 'key' | 'value', val: string) => {
    setLabelRows((prev) =>
      prev.map((r, i) => (i === index ? { ...r, [field]: val } : r)),
    )
  }

  const columns: ColumnsType<EnumConfig> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '字段名',
      dataIndex: 'field',
      width: 150,
      ellipsis: true,
      render: (v) => <Typography.Text strong>{v}</Typography.Text>,
    },
    {
      title: '映射值',
      dataIndex: 'labels',
      render: (labels: Record<string, string>) => (
        <Space wrap size={4}>
          {Object.entries(labels).map(([k, v]) => (
            <Tag key={k}>
              <Typography.Text type="secondary">{k}</Typography.Text> → {v}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '操作',
      width: 120,
      render: (_, row) => (
        <Space size="small">
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(row)} />
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(row.id)}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Card title="枚举配置" extra={<Button icon={<PlusOutlined />} onClick={openCreate}>新增</Button>}>
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 'max-content' }} />

      <Modal
        title={editing ? '编辑枚举' : '新增枚举'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} onFinish={handleSubmit} layout="vertical">
          <Form.Item label="字段名" name="field" rules={[{ required: true }]}>
            <Input placeholder="e.g. app_id, platform_id, user_mode" disabled={editing != null} />
          </Form.Item>
        </Form>

        <Typography.Text strong style={{ marginBottom: 8 }}>映射值</Typography.Text>

        <div style={{ marginBottom: 8 }}>
          {labelRows.map((r, i) => (
            <Space key={i} style={{ marginBottom: 4 }}>
              <Input
                value={r.key}
                onChange={(e) => updateLabelRow(i, 'key', e.target.value)}
                placeholder="值 (如 1)"
                style={{ width: 120 }}
              />
              <Input
                value={r.value}
                onChange={(e) => updateLabelRow(i, 'value', e.target.value)}
                placeholder="标签 (如 AppA)"
                style={{ width: 150 }}
              />
              <Button size="small" danger onClick={() => removeLabelRow(i)}>删除</Button>
            </Space>
          ))}
        </div>

        <Space>
          <Input
            value={addKey}
            onChange={(e) => setAddKey(e.target.value)}
            placeholder="值"
            style={{ width: 120 }}
          />
          <Input
            value={addValue}
            onChange={(e) => setAddValue(e.target.value)}
            placeholder="标签"
            style={{ width: 150 }}
          />
          <Button size="small" type="dashed" onClick={addLabelRow}>添加</Button>
        </Space>
      </Modal>
    </Card>
  )
}