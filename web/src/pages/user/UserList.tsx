import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Tag,
  Select,
  message,
  Switch,
} from 'antd'
import { PlusOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { listUsers, createUser, setUserStatus, type User } from '../../api/user'
import { listPermissionGroups, type PermissionGroup } from '../../api/permission'
import { useAuthStore } from '../../stores/auth'
import UserPermissionModal from './UserPermissionModal'

export default function UserList() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<User[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [permModalUser, setPermModalUser] = useState<{ id: number; username: string } | null>(null)
  const [groups, setGroups] = useState<PermissionGroup[]>([])
  const [form] = Form.useForm()
  const currentUsername = useAuthStore((s) => s.username)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listUsers()
      setData(res ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchGroups = useCallback(async () => {
    try {
      const gs = await listPermissionGroups()
      setGroups(gs)
    } catch {}
  }, [])

  useEffect(() => {
    fetchData()
    fetchGroups()
  }, [fetchData, fetchGroups])

  const handleCreate = async (values: { username: string; password: string; role: string; group_ids?: number[] }) => {
    await createUser(values)
    message.success('用户已创建')
    setModalOpen(false)
    form.resetFields()
    fetchData()
  }

  const handleToggleStatus = async (user: User, checked: boolean) => {
    await setUserStatus(user.id, checked ? 1 : 0)
    message.success('状态已更新')
    fetchData()
  }

  const openCreate = () => {
    form.resetFields()
    form.setFieldsValue({ role: 'viewer', group_ids: [] })
    setModalOpen(true)
  }

  const columns: ColumnsType<User> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '用户名', dataIndex: 'username', width: 150, ellipsis: true },
    {
      title: '角色',
      dataIndex: 'role',
      width: 100,
      render: (v) => (
        <Tag color={v === 'admin' ? 'red' : 'blue'}>{v === 'admin' ? '管理员' : '观察者'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (v, row) => (
        <Switch
          checked={v === 1}
          disabled={row.username === currentUsername}
          onChange={(checked) => handleToggleStatus(row, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '最后登录',
      dataIndex: 'last_login_at',
      width: 160,
      render: (v) => (v ? dayjs(v).format('MM-DD HH:mm') : '-'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v) => dayjs(v).format('YYYY-MM-DD'),
    },
    {
      title: '操作',
      width: 80,
      render: (_, row) => (
        <Button
          icon={<SafetyCertificateOutlined />}
          size="small"
          onClick={() => setPermModalUser({ id: row.id, username: row.username })}
        >
          权限
        </Button>
      ),
    },
  ]

  return (
    <Card
      title="用户管理"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建用户
        </Button>
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 'max-content' }} />

      <Modal
        title="新建用户"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); form.resetFields() }}
        onOk={() => form.submit()}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item label="用户名" name="username" rules={[{ required: true, min: 2 }]}>
            <Input />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, min: 6 }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item label="角色" name="role" initialValue="viewer" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="admin">管理员</Select.Option>
              <Select.Option value="viewer">观察者</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="权限组" name="group_ids">
            <Select mode="multiple" placeholder="选择权限组（可选）" allowClear>
              {groups.map((g) => (
                <Select.Option key={g.id} value={g.id}>{g.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {permModalUser && (
        <UserPermissionModal
          userId={permModalUser.id}
          username={permModalUser.username}
          open={!!permModalUser}
          onClose={() => setPermModalUser(null)}
        />
      )}
    </Card>
  )
}
