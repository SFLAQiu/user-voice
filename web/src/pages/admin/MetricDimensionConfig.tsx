import { useState, useCallback, useEffect } from 'react'
import dayjs from 'dayjs'
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Popconfirm,
  Tag,
  Typography,
  message,
  Spin,
  Tabs,
} from 'antd'
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  listMetricDimensions,
  addMetricDimension,
  removeMetricDimension,
  getRebuildStatus,
  triggerBackfill,
  listBucketUpdateLogs,
  type MetricDimensionConfig,
  type RebuildStatus,
  type MetricBucketUpdateLog,
} from '../../api/metricDimension'

const { Text } = Typography

const OP_LABEL: Record<string, string> = {
  backfill: '全量回填',
  increment: '增量更新',
  classify_move: '分类变更',
  manual_classify_move: '手动变更分类',
  version_switch: '版本切换',
  cleanup: '清理',
  reconcile: '对账补漏',
}
const OP_COLOR: Record<string, string> = {
  backfill: 'blue',
  increment: 'green',
  classify_move: 'orange',
  manual_classify_move: 'volcano',
  version_switch: 'purple',
  cleanup: 'default',
  reconcile: 'cyan',
}

const TRIGGER_LABEL: Record<string, string> = {
  manual: '手动触发',
  sync: '数据同步',
  classify: '分类器',
  system: '系统自动',
  dimension_change: '维度变更',
}

export default function MetricDimensionConfigPage() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<MetricDimensionConfig[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [rebuildStatus, setRebuildStatus] = useState<RebuildStatus>({ state: 'idle', version: 1 })
  const [refreshing, setRefreshing] = useState(false)
  const [logs, setLogs] = useState<MetricBucketUpdateLog[]>([])
  const [logsLoading, setLogsLoading] = useState(false)
  const [form] = Form.useForm()

  const isZeroTime = (t?: string | null) =>
    !t || t.startsWith('0001-01-01')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const rows = await listMetricDimensions()
      setData(rows ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchStatus = useCallback(async (showRefresh?: boolean) => {
    if (showRefresh) setRefreshing(true)
    try {
      const s = await getRebuildStatus()
      setRebuildStatus(s)
    } catch { /* ignore */ }
    if (showRefresh) setRefreshing(false)
  }, [])

  const fetchLogs = useCallback(async () => {
    setLogsLoading(true)
    try {
      setLogs((await listBucketUpdateLogs(100)) ?? [])
    } catch { /* ignore */ }
    setLogsLoading(false)
  }, [])

  useEffect(() => {
    fetchData()
    fetchStatus()
    fetchLogs()
  }, [fetchData, fetchStatus, fetchLogs])

  useEffect(() => {
    if (rebuildStatus.state !== 'running') return
    const timer = setInterval(() => { fetchStatus() }, 3000)
    return () => clearInterval(timer)
  }, [rebuildStatus.state, fetchStatus])

  const handleAdd = async (values: { field: string; label: string; data_type: string; default_value?: string }) => {
    try {
      await addMetricDimension(values)
      message.success('维度已添加，重建任务已启动')
      setModalOpen(false)
      form.resetFields()
      fetchData()
      fetchStatus()
    } catch (e: any) {
      message.error(e?.response?.data?.message || '添加失败')
    }
  }

  const handleRemove = async (field: string) => {
    try {
      await removeMetricDimension(field)
      message.success('维度已删除，重建任务已启动')
      fetchData()
      fetchStatus()
    } catch (e: any) {
      message.error(e?.response?.data?.message || '删除失败')
    }
  }

  const columns: ColumnsType<MetricDimensionConfig> = [
    { title: '字段名', dataIndex: 'field', key: 'field', width: 140, ellipsis: true },
    { title: '中文名', dataIndex: 'label', key: 'label', width: 120 },
    {
      title: '类型',
      dataIndex: 'data_type',
      key: 'data_type',
      width: 80,
      render: (v: string) => v === 'int' ? <Tag color="blue">int</Tag> : <Tag>string</Tag>,
    },
    { title: '默认值', dataIndex: 'default_value', key: 'default_value', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (v: number) => v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">禁用</Tag>,
    },
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm') },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, row) => (
        <Popconfirm
          title={`删除维度 "${row.label}" 将触发全量数据重建，确认删除？`}
          onConfirm={() => handleRemove(row.field)}
          okText="确认"
          cancelText="取消"
        >
          <Button type="text" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ]

  const statusColor = () => {
    switch (rebuildStatus.state) {
      case 'running': return 'processing'
      case 'completed': return 'success'
      case 'failed': return 'error'
      default: return 'default'
    }
  }

  const statusLabel = () => {
    switch (rebuildStatus.state) {
      case 'running': return '重建中'
      case 'completed': return '已完成'
      case 'failed': return '失败'
      default: return '空闲'
    }
  }

  const logColumns: ColumnsType<MetricBucketUpdateLog> = [
    {
      title: '操作类型', dataIndex: 'op_type', width: 100,
      render: (v: string) => <Tag color={OP_COLOR[v] ?? 'default'}>{OP_LABEL[v] ?? v}</Tag>,
    },
    { title: '版本', dataIndex: 'version', width: 70 },
    { title: '影响行数', dataIndex: 'affected_rows', width: 90 },
    {
      title: '耗时', dataIndex: 'duration_ms', width: 90,
      render: (v: number) => v > 0 ? `${v}ms` : '-',
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: (v: string) => v === 'success' ? <Tag color="green">成功</Tag> : <Tag color="red">失败</Tag>,
    },
    {
      title: '来源', dataIndex: 'triggered_by', width: 100,
      render: (v: string) => v ? <Tag>{TRIGGER_LABEL[v] ?? v}</Tag> : '-',
    },
    { title: '描述', dataIndex: 'message', width: 200, ellipsis: true },
    { title: '时间', dataIndex: 'created_at', width: 160, render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm') },
  ]

  return (
    <div>
      <Card
        title="重建状态"
        size="small"
        style={{ marginBottom: 16 }}
        extra={<Space size="small">
          <Popconfirm title="全量回填会从 feedbacks 表重新聚合所有历史数据到时间桶，确认执行？" onConfirm={async () => {
            try {
              await triggerBackfill()
              message.success('回填任务已启动')
              fetchStatus()
            } catch (e: any) { message.error(e?.response?.data?.message || '回填启动失败') }
          }} okText="确认" cancelText="取消">
            <Button size="small" disabled={rebuildStatus.state === 'running'}>全量回填</Button>
          </Popconfirm>
          <Button size="small" icon={<ReloadOutlined />} loading={refreshing} onClick={() => { fetchStatus(true); fetchLogs() }}>刷新</Button>
        </Space>}
      >
        <Tabs
          size="small"
          items={[
            {
              key: 'status',
              label: '状态',
              children: (
                <Space>
                  <Tag color={statusColor()}>{statusLabel()}</Tag>
                  <Text>版本 v{rebuildStatus.version}</Text>
                  {!isZeroTime(rebuildStatus.started_at) && <Text type="secondary">开始于 {dayjs(rebuildStatus.started_at).format('YYYY-MM-DD HH:mm:ss')}</Text>}
                  {rebuildStatus.message && <Text type="secondary">{rebuildStatus.message}</Text>}
                  {rebuildStatus.state === 'running' && <Spin size="small" />}
                </Space>
              ),
            },
            {
              key: 'logs',
              label: '更新日志',
              children: (
                <Table
                  rowKey="id"
                  columns={logColumns}
                  dataSource={logs}
                  loading={logsLoading}
                  pagination={{ pageSize: 20, size: 'small' }}
                  size="small"
                  scroll={{ x: 'max-content' }}
                />
              ),
            },
          ]}
        />
      </Card>

      <Card
        title="时间网格维度配置"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { form.resetFields(); setModalOpen(true) }}
            disabled={rebuildStatus.state === 'running'}
          >
            新增维度
          </Button>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          size="small"
          scroll={{ x: 'max-content' }}
        />
      </Card>

      <Modal
        title="新增维度"
        open={modalOpen}
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        okText="添加并重建"
        confirmLoading={rebuildStatus.state === 'running'}
      >
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="field" label="字段名" rules={[{ required: true, message: '请输入字段名' }]}
            extra="feedbacks 表中的字段名，如 app_version、channel_id">
            <Input placeholder="如 app_version" />
          </Form.Item>
          <Form.Item name="label" label="中文名" rules={[{ required: true, message: '请输入中文名' }]}>
            <Input placeholder="如 应用版本" />
          </Form.Item>
          <Form.Item name="data_type" label="数据类型" rules={[{ required: true }]}
            initialValue="string">
            <Select options={[
              { value: 'string', label: '字符串 (string)' },
              { value: 'int', label: '整数 (int)' },
            ]} />
          </Form.Item>
          <Form.Item name="default_value" label="默认值"
            extra="空值时的填充值，string 默认空串，int 默认 0">
            <Input placeholder="如 '' 或 0" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}