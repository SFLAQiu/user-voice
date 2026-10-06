import { useEffect, useState, useCallback } from 'react'
import { Table, Card, Button, Space, Modal, Form, Input, Checkbox, message, Popconfirm, Tag } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import {
  listPermissionGroups,
  createPermissionGroup,
  updatePermissionGroup,
  deletePermissionGroup,
  type PermissionGroup,
} from '../../api/permission'

const PAGE_LABELS: Record<string, string> = {
  feedback: '反馈列表',
  dashboard: '仪表盘',
  alert: '告警管理',
  notification_channels: '通知渠道',
  datasources: '数据源管理',
  enum_configs: '枚举配置',
  classifier_configs: '智能配置',
  metric_dimensions: '维度配置',
  llm_providers: 'LLM 提供商',
  users: '用户管理',
  guide: '功能指南',
}

const PERM_LABELS: Record<string, string> = {
  view: '查看',
  edit: '编辑',
}

const PAGE_KEYS = Object.keys(PAGE_LABELS)

export default function PermissionGroupList() {
  const [data, setData] = useState<PermissionGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PermissionGroup | null>(null)
  const [form] = Form.useForm()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const rows = await listPermissionGroups()
      setData(rows)
    } catch {
      // error handled by interceptor
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
    form.setFieldsValue({
      permissions: PAGE_KEYS.reduce(
        (acc, k) => ({ ...acc, [k]: k === 'guide' ? ['view'] : [] }),
        {} as Record<string, string[]>,
      ),
    })
    setModalOpen(true)
  }

  const openEdit = (item: PermissionGroup) => {
    setEditing(item)
    form.setFieldsValue({
      name: item.name,
      description: item.description || '',
      permissions: item.permissions,
    })
    setModalOpen(true)
  }

  const handleSubmit = async (values: { name: string; description?: string; permissions: Record<string, string[]> }) => {
    try {
      if (editing) {
        await updatePermissionGroup(editing.id, values)
        message.success('已更新')
      } else {
        await createPermissionGroup(values)
        message.success('已创建')
      }
      setModalOpen(false)
      fetchData()
    } catch {
      // error handled by interceptor
    }
  }

  const handleDelete = async (id: number) => {
    await deletePermissionGroup(id)
    message.success('已删除')
    fetchData()
  }

  const columns: ColumnsType<PermissionGroup> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '名称', dataIndex: 'name', width: 120 },
    {
      title: '描述',
      dataIndex: 'description',
      width: 200,
      ellipsis: true,
      render: (v: string | undefined) => v || '-',
    },
    {
      title: '权限摘要',
      dataIndex: 'permissions',
      width: 400,
      render: (perms: Record<string, string[]>) => {
        const tags: React.ReactNode[] = []
        for (const page of PAGE_KEYS) {
          const keys = perms[page]
          if (!keys || keys.length === 0) continue
          const label = PAGE_LABELS[page] || page
          const permText = keys.map((k) => PERM_LABELS[k] || k).join('/')
          tags.push(
            <Tag key={page} color={keys.includes('edit') ? 'blue' : 'green'}>
              {label}:{permText}
            </Tag>,
          )
        }
        return <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>{tags}</div>
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 120,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD'),
    },
    {
      title: '操作',
      width: 140,
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
    <Card
      title="权限组管理"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建权限组
        </Button>
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 'max-content' }} />

      <Modal
        title={editing ? '编辑权限组' : '新建权限组'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        destroyOnClose
        width={720}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入权限组名称' }]}>
            <Input maxLength={64} placeholder="如：产品运营" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea maxLength={255} rows={2} placeholder="可选，说明该权限组的用途" />
          </Form.Item>
          <Form.Item label="权限配置" required>
            <div style={{ maxHeight: 360, overflow: 'auto', border: '1px solid #d9d9d9', borderRadius: 6, padding: 16 }}>
              {PAGE_KEYS.map((page) => {
                const pageLabel = PAGE_LABELS[page] || page
                const perms = page === 'guide' ? ['view'] : ['view', 'edit']
                return (
                  <div key={page} style={{ marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid #f0f0f0' }}>
                    <div style={{ fontWeight: 600, marginBottom: 4 }}>{pageLabel}</div>
                    <Form.Item name={['permissions', page]} noStyle>
                      <Checkbox.Group
                        options={perms.map((k) => ({
                          label: PERM_LABELS[k] || k,
                          value: k,
                        }))}
                      />
                    </Form.Item>
                  </div>
                )
              })}
            </div>
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
