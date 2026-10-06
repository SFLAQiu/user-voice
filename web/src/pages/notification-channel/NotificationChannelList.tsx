import { useEffect, useState, useCallback } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  message,
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  listNotificationChannels,
  createNotificationChannel,
  updateNotificationChannel,
  deleteNotificationChannel,
  testNotificationChannel,
  type NotificationChannel,
} from '../../api/notification-channel'

const TYPE_LABEL: Record<string, string> = {
  wecom: '企业微信',
  feishu: '飞书',
  dingtalk: '钉钉',
  webhook: 'Webhook',
}

export default function NotificationChannelList() {
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<NotificationChannel | null>(null)
  const [form] = Form.useForm()

  const fetchChannels = useCallback(async () => {
    setLoading(true)
    try { setChannels((await listNotificationChannels()) ?? []) } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchChannels() }, [fetchChannels])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ type: 'wecom' })
    setModalOpen(true)
  }

  const openEdit = (row: NotificationChannel) => {
    setEditing(row)
    form.setFieldsValue({ name: row.name, type: row.type, webhook_url: row.webhook_url, secret: row.secret })
    setModalOpen(true)
  }

  const handleSubmit = async (values: Record<string, unknown>) => {
    if (editing) { await updateNotificationChannel(editing.id, values); message.success('已更新') }
    else { await createNotificationChannel(values); message.success('已创建') }
    setModalOpen(false)
    fetchChannels()
  }

  const handleTest = async (id: number) => {
    await testNotificationChannel(id)
    message.success('测试消息已发送')
  }

  const columns: ColumnsType<NotificationChannel> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '名称', dataIndex: 'name', width: 150, ellipsis: true },
    { title: '类型', dataIndex: 'type', width: 90, render: (v: string) => TYPE_LABEL[v] ?? v },
    { title: 'Webhook URL', dataIndex: 'webhook_url', ellipsis: true, width: 200 },
    {
      title: '操作', width: 160, render: (_, row) => (
        <Space size="small">
          <Button icon={<CheckCircleOutlined />} size="small" onClick={() => handleTest(row.id)}>测试</Button>
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(row)} />
          <Popconfirm title="确认删除？" onConfirm={async () => { await deleteNotificationChannel(row.id); message.success('已删除'); fetchChannels() }}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Card title="通知渠道">
      <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} style={{ marginBottom: 16 }}>
        新建渠道
      </Button>
      <Table rowKey="id" loading={loading} columns={columns} dataSource={channels} scroll={{ x: 'max-content' }} />

      <Modal title={editing ? '编辑渠道' : '新建渠道'} open={modalOpen}
        onCancel={() => setModalOpen(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="类型" name="type" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="wecom">企业微信</Select.Option>
              <Select.Option value="feishu">飞书</Select.Option>
              <Select.Option value="dingtalk">钉钉</Select.Option>
              <Select.Option value="webhook">Webhook</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="Webhook URL" name="webhook_url" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Secret (可选)" name="secret"><Input /></Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}