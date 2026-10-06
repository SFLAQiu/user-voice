import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Table,
  Space,
  Modal,
  Form,
  Input,
  message,
  Tag,
  Divider,
} from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, BarChartOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import {
  listDashboards,
  createDashboard,
  updateDashboard,
  deleteDashboard,
  getDeleteInfo,
  hasPanelAlertRules,
  type Dashboard,
} from '../../api/dashboard'
import DashboardFilterEditor from '../../components/DashboardFilterEditor'
import { FilterNode, normalizeFilterTree, createEmptyGroup } from '../../types/filterNode'
import { useDashboardEnums } from '../../hooks/useDashboardEnums'

const FIELD_LABELS: Record<string, string> = {
  platform_id: '平台',
  app_id: '应用',
  app_version: '版本号',
  category: '分类',
  sentiment: '情感倾向',
  business_module: '业务模块',
  category_status: '分类状态',
}

export default function DashboardList() {
  const { resolveFilterNodeLabels } = useDashboardEnums()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [data, setData] = useState<Dashboard[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Dashboard | null>(null)
  const [filters, setFilters] = useState<FilterNode | null>(null)
  const [form] = Form.useForm()
  const navigate = useNavigate()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDashboards()
      setData(res ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const openCreate = () => {
    setEditing(null)
    setFilters(null)
    setModalOpen(true)
    form.resetFields()
  }

  const openEdit = (row: Dashboard) => {
    setEditing(row)
    // Normalize filters (supports both old flat array and new tree format)
    setFilters(normalizeFilterTree(row.filters) ?? createEmptyGroup())
    setModalOpen(true)
    form.setFieldsValue({ name: row.name, description: row.description })
  }

  // 深比较筛选条件是否变更
  const filtersChanged = useCallback((old: FilterNode | null | undefined, newFilters: FilterNode | null): boolean => {
    const normOld = normalizeFilterTree(old)
    const normNew = normalizeFilterTree(newFilters)
    return JSON.stringify(normOld) !== JSON.stringify(normNew)
  }, [])

  const handleSubmit = async (values: { name: string; description: string }) => {
    const payload: Partial<Dashboard> & { rebuild_alerts?: boolean } = {
      ...values,
      filters: filters ?? undefined,
    }
    // 编辑模式下筛选条件变更时，需检查是否有面板告警规则并二次确认
    if (editing && filtersChanged(editing.filters, filters)) {
      try {
        const hasAlerts = await hasPanelAlertRules(editing.id)
        if (hasAlerts) {
          // 需二次确认
          Modal.confirm({
            title: '告警规则将重建',
            content: '修改仪表筛选条件，面板告警规则将重建，原来的面板告警日志将清空。是否继续？',
            okText: '确认修改',
            cancelText: '取消',
            onOk: async () => {
              setSubmitting(true)
              try {
                payload.rebuild_alerts = true
                await updateDashboard(editing.id, payload)
                message.success('已更新，告警规则已重建')
                setModalOpen(false)
                await fetchData()
              } catch {
                // Global axios interceptor already shows backend error message.
              } finally {
                setSubmitting(false)
              }
            },
          })
          return // 等待用户确认，不直接提交
        }
      } catch {
        // 预检失败时不阻断流程，直接提交
      }
    }
    setSubmitting(true)
    try {
      if (editing) {
        await updateDashboard(editing.id, payload)
        message.success('已更新')
      } else {
        await createDashboard(payload)
        message.success('已创建')
      }
      setModalOpen(false)
      await fetchData()
    } catch {
      // Global axios interceptor already shows backend error message.
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = (row: Dashboard) => {
    getDeleteInfo(row.id).then((info) => {
      const lines = [`当前仪表盘下有 ${info.panel_count} 个面板`]
      if (info.alert_rule_count > 0) {
        lines.push(`有 ${info.alert_rule_count} 条告警规则（删除后将同步清除）`)
      }
      Modal.confirm({
        title: '确认删除仪表盘？',
        content: lines.join('，') + '。是否确认删除？',
        okText: '确认删除',
        okButtonProps: { danger: true },
        cancelText: '取消',
        onOk: async () => {
          await deleteDashboard(row.id)
          message.success('已删除')
          fetchData()
        },
      })
    }).catch(() => {
      // 预检失败时仍弹出简单确认
      Modal.confirm({
        title: '确认删除？',
        onOk: async () => {
          await deleteDashboard(row.id)
          message.success('已删除')
          fetchData()
        },
      })
    })
  }

  const renderFilterSummary = (filters?: FilterNode) => {
    const normalized = normalizeFilterTree(filters)
    if (!normalized) return <span style={{ color: '#999' }}>无筛选</span>
    const label = resolveFilterNodeLabels(normalized, FIELD_LABELS)
    if (!label) return <span style={{ color: '#999' }}>无筛选</span>
    return <Tag color="blue">{label}</Tag>
  }

  const columns: ColumnsType<Dashboard> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true },
    {
      title: '筛选条件',
      dataIndex: 'filters',
      width: 240,
      render: (_, row) => renderFilterSummary(row.filters),
    },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      width: 120,
      render: (_, row) => (
        <Space>
          <Button icon={<BarChartOutlined />} size="small" type="primary" ghost onClick={() => navigate(`/dashboards/${row.id}`)}>
            查看
          </Button>
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(row)} />
          <Button icon={<DeleteOutlined />} size="small" danger onClick={() => handleDelete(row)} />
        </Space>
      ),
    },
  ]

  return (
    <Card
      title="仪表盘"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建仪表盘
        </Button>
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 'max-content' }} />

      <Modal
        title={editing ? '编辑仪表盘' : '新建仪表盘'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        width={640}
        destroyOnHidden
        forceRender
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Divider>筛选条件</Divider>
          <DashboardFilterEditor value={filters} onChange={setFilters} />
        </Form>
      </Modal>
    </Card>
  )
}