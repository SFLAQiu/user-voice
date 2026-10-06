import { useEffect, useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import {
  Tabs,
  Table,
  Card,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  Tag,
  Popconfirm,
  message,
  Switch,
  Tooltip,
  Segmented,
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import AlertFilterEditor from '../../components/AlertFilterEditor'
import {
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  toggleAlertRuleStatus,
  deleteAlertRule,
  listAlertRecords,
  type AlertRule,
  type AlertRecord,
} from '../../api/alert'
import { listNotificationChannels, type NotificationChannel } from '../../api/notification-channel'

const DEFAULT_FILTERS = JSON.stringify({ filters: [] }, null, 2)

const LEVEL_LABEL: Record<string, string> = { info: '信息', warning: '警告', critical: '严重' }
const LEVEL_COLOR: Record<string, string> = { info: 'blue', warning: 'orange', critical: 'red' }
const STATE_LABEL: Record<string, string> = { ok: '正常', firing: '触发中', pending: '预警', resolved: '已恢复', '': 'N/A' }
const STATE_COLOR: Record<string, string> = { ok: 'green', firing: 'red', pending: 'gold', resolved: 'green', '': 'default' }

const CONDITION_OPTIONS = [
  { value: '>', label: '>' },
  { value: '>=', label: '≥' },
  { value: '<', label: '<' },
  { value: '<=', label: '≤' },
]

const LEVEL_OPTIONS = [
  { value: 'info', label: '信息' },
  { value: 'warning', label: '警告' },
  { value: 'critical', label: '严重' },
]

const REDUCE_MODE_OPTIONS = [
  { value: 'max', label: '峰值（最大桶值）' },
  { value: 'last', label: '最新（最近桶值）' },
]

const HINTS: Record<string, string> = {
  name: '告警规则的名称，用于区分不同告警场景',
  metric_query: '按桶维度字段值筛选告警数据源，可选项，不填则查询全量数据',
  condition_op: '比较操作符，将查询结果值与阈值对比',
  threshold: '触发告警的数值阈值',
  level: '告警严重程度',
  eval_interval_sec: '每隔多少秒重新评估一次规则，最小 10 秒',
  pending_duration_sec: '指标突破阈值后需持续高于阈值的时间（秒）。0=立即告警，>0 则先进入预警期，超过此时间仍高于阈值才正式告警',
  time_window_sec: '评估窗口时长（秒），决定查询 metric_time_buckets 的时间范围。窗口越长，搜索的 1m 桶越多',
  reduce_mode: '触发值基于时间网格桶的反馈数量。峰值=窗口内最大桶值，最新=最近桶值',
  silence_minutes: '告警触发后的静默期，避免短时间内重复通知',
  channel_ids: '告警触发时向哪些通知渠道发送消息',
}

function HintLabel({ field, text }: { field: string; text: string }) {
  const hint = HINTS[field]
  return hint ? (
    <Tooltip title={hint}>
      {text} <QuestionCircleOutlined style={{ color: '#999', fontSize: 11, marginLeft: 2, cursor: 'help' }} />
    </Tooltip>
  ) : text
}

const NOTIFY_STATUS_LABEL: Record<string, string> = { ok: '已通知', no_channels: '无渠道', partial_fail: '部分失败' }
const NOTIFY_STATUS_COLOR: Record<string, string> = { ok: 'green', no_channels: 'orange', partial_fail: 'red' }

// 两列网格字段样式
const fieldItemStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
}

export default function AlertPage() {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [records, setRecords] = useState<AlertRecord[]>([])
  const [ruleLoading, setRuleLoading] = useState(false)
  const [ruleModalOpen, setRuleModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [ruleForm] = Form.useForm()
  const [ruleFilter, setRuleFilter] = useState<'all' | 'custom' | 'panel'>('all')

  const fetchRules = useCallback(async () => {
    setRuleLoading(true)
    try {
      const params = ruleFilter !== 'all' ? { type: ruleFilter } : undefined
      setRules((await listAlertRules(params)) ?? [])
    } finally { setRuleLoading(false) }
  }, [ruleFilter])

  const fetchChannels = useCallback(async () => {
    setChannels((await listNotificationChannels()) ?? [])
  }, [])

  const fetchRecords = useCallback(async () => {
    setRecords((await listAlertRecords({ limit: 100 })) ?? [])
  }, [])

  useEffect(() => { fetchRules(); fetchChannels(); fetchRecords() }, [fetchRules, fetchChannels, fetchRecords])

  // ── 规则 CRUD ──
  const openCreateRule = () => {
    setEditingRule(null)
    ruleForm.resetFields()
    ruleForm.setFieldsValue({
      metric_query: DEFAULT_FILTERS,
      condition_op: '>',
      threshold: 0,
      level: 'warning',
      eval_interval_sec: 60,
      pending_duration_sec: 0,
      time_window_sec: 300,
      reduce_mode: 'max',
      silence_minutes: 30,
      channel_ids: channels.length > 0 ? [channels[0].id] : [],
    })
    setRuleModalOpen(true)
  }
  const openEditRule = (row: AlertRule) => {
    if (row.panel_id) {
      message.warning('面板告警规则需通过面板配置修改')
      return
    }
    setEditingRule(row)
    ruleForm.setFieldsValue({
      name: row.name,
      metric_query: JSON.stringify(row.metric_query, null, 2),
      condition_op: row.condition_op,
      threshold: row.threshold,
      level: row.level,
      eval_interval_sec: row.eval_interval_sec,
      pending_duration_sec: row.pending_duration_sec ?? 0,
      time_window_sec: row.time_window_sec ?? 300,
      reduce_mode: row.reduce_mode ?? 'max',
      silence_minutes: row.silence_minutes,
      channel_ids: row.channel_ids ?? [],
    })
    setRuleModalOpen(true)
  }
  const handleSubmitRule = async (values: Record<string, unknown>) => {
    let mq
    try { mq = JSON.parse(values.metric_query as string) } catch { message.error('过滤条件 JSON 格式错误'); return }
    const payload = { ...values, metric_query: mq }
    if (editingRule) { await updateAlertRule(editingRule.id, payload); message.success('已更新') }
    else { await createAlertRule(payload); message.success('已创建') }
    setRuleModalOpen(false)
    fetchRules()
  }

  const ruleColumns: ColumnsType<AlertRule> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '名称', dataIndex: 'name', width: 200, ellipsis: true },
    {
      title: '来源', width: 100,
      render: (_, r) => r.panel_id
        ? <Tag color="blue">面板告警</Tag>
        : <Tag>自定义规则</Tag>,
    },
    {
      title: '仪表盘', width: 120,
      render: (_, r) => r.dashboard_name
        ? <Link to={`/dashboards/${r.dashboard_id}`}>{r.dashboard_name}</Link>
        : '-',
    },
    { title: '条件', width: 120, render: (_, r) => `${r.condition_op} ${r.threshold}` },
    { title: '级别', dataIndex: 'level', width: 80, render: (v) => <Tag color={LEVEL_COLOR[v] ?? 'default'}>{LEVEL_LABEL[v] ?? v}</Tag> },
    { title: '最近状态', dataIndex: 'last_state', width: 100, render: (v) => <Tag color={STATE_COLOR[v ?? ''] ?? 'default'}>{STATE_LABEL[v ?? ''] ?? v ?? 'N/A'}</Tag> },
    { title: '通知渠道', dataIndex: 'channel_ids', width: 150, render: (ids: number[]) => {
      const names = (ids ?? []).map((cid) => channels.find((c) => c.id === cid)?.name ?? cid)
      return names.length > 0 ? names.map((n) => <Tag key={String(n)}>{n}</Tag>) : <Tag>未关联</Tag>
    }},
    { title: '启用', dataIndex: 'status', width: 70, render: (v, row) => (
      <Switch checked={v === 1} size="small" onChange={async (checked) => {
        await toggleAlertRuleStatus(row.id, checked ? 1 : 0); fetchRules()
      }} />
    )},
    {
      title: '操作', width: 110, render: (_, row) => (
        <Space size="small">
          {row.panel_id ? (
            <Tooltip title="面板告警规则需通过面板配置修改">
              <Button icon={<EditOutlined />} size="small" disabled />
            </Tooltip>
          ) : (
            <Button icon={<EditOutlined />} size="small" onClick={() => openEditRule(row)} />
          )}
          <Popconfirm title="确认删除？" onConfirm={async () => { await deleteAlertRule(row.id); message.success('已删除'); fetchRules() }}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const recordColumns: ColumnsType<AlertRecord> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '规则ID', dataIndex: 'rule_id', width: 80 },
    { title: '级别', dataIndex: 'level', width: 80, render: (v) => <Tag color={LEVEL_COLOR[v] ?? 'default'}>{LEVEL_LABEL[v] ?? v}</Tag> },
    { title: '状态', dataIndex: 'state', width: 80, render: (v) => <Tag color={STATE_COLOR[v ?? ''] ?? 'default'}>{STATE_LABEL[v ?? ''] ?? v ?? 'N/A'}</Tag> },
    { title: '触发值', dataIndex: 'trigger_value', width: 90 },
    { title: '阈值', dataIndex: 'threshold', width: 80 },
    { title: '通知渠道', dataIndex: 'notified_channels', width: 150, render: (ids: number[]) => {
      if (!ids || ids.length === 0) return <Tag color="orange">无</Tag>
      return ids.map((cid) => <Tag key={cid}>{channels.find((c) => c.id === cid)?.name ?? cid}</Tag>)
    }},
    { title: '通知状态', dataIndex: 'notify_status', width: 100, render: (v) => <Tag color={NOTIFY_STATUS_COLOR[v] ?? 'default'}>{NOTIFY_STATUS_LABEL[v] ?? v ?? 'N/A'}</Tag> },
    { title: '触发时间', dataIndex: 'triggered_at', width: 160, render: (v, record: AlertRecord) => dayjs(record.metric_timestamp ?? v).format('MM-DD HH:mm:ss') },
  ]

  return (
    <Card title="告警管理">
      <Tabs
        items={[
          {
            key: 'rules',
            label: '告警规则',
            children: (
              <>
                <Space style={{ marginBottom: 16 }}>
                  <Segmented
                    options={[
                      { value: 'all', label: '全部' },
                      { value: 'custom', label: '自定义规则' },
                      { value: 'panel', label: '面板告警' },
                    ]}
                    value={ruleFilter}
                    onChange={(v) => setRuleFilter(v as 'all' | 'custom' | 'panel')}
                  />
                  <Button type="primary" icon={<PlusOutlined />} onClick={openCreateRule}>
                    新建规则
                  </Button>
                </Space>
                <Table rowKey="id" loading={ruleLoading} columns={ruleColumns} dataSource={rules} scroll={{ x: 'max-content' }} />
              </>
            ),
          },
          {
            key: 'records',
            label: '告警记录',
            children: (
              <Table rowKey="id" columns={recordColumns} dataSource={records} scroll={{ x: 'max-content' }} />
            ),
          },
        ]}
      />

      {/* Rule Modal */}
      <Modal title={editingRule ? '编辑规则' : '新建规则'} open={ruleModalOpen}
        onCancel={() => setRuleModalOpen(false)} onOk={() => ruleForm.submit()} width={640} destroyOnClose>
        <Form form={ruleForm} layout="vertical" onFinish={handleSubmitRule}>
          <Form.Item label={<HintLabel field="name" text="名称" />} name="name" rules={[{ required: true }]}><Input /></Form.Item>

          {/* 过滤条件 */}
          <Form.Item label={<HintLabel field="metric_query" text="过滤条件" />} name="metric_query" rules={[{ required: true }]}>
            <AlertFilterEditor />
          </Form.Item>

          {/* 两列网格：条件 + 级别 | 阈值 + 评估间隔 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '12px 16px',
            marginBottom: 12,
          }}>
            <div style={fieldItemStyle}>
              <HintLabel field="condition_op" text="条件" />
              <Form.Item name="condition_op" noStyle rules={[{ required: true }]}>
                <Select style={{ width: 80 }} options={CONDITION_OPTIONS} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="level" text="级别" />
              <Form.Item name="level" noStyle rules={[{ required: true }]}>
                <Select style={{ width: 100 }} options={LEVEL_OPTIONS} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="threshold" text="阈值" />
              <Form.Item name="threshold" noStyle rules={[{ required: true }]}>
                <InputNumber style={{ width: 120 }} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="eval_interval_sec" text="评估间隔(s)" />
              <Form.Item name="eval_interval_sec" noStyle>
                <InputNumber style={{ width: 120 }} min={10} />
              </Form.Item>
            </div>
          </div>

          {/* 两列网格：待确认时长 + 评估窗口 | 缩减模式 + 静默时间 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '12px 16px',
            marginBottom: 12,
          }}>
            <div style={fieldItemStyle}>
              <HintLabel field="pending_duration_sec" text="待确认时长(s)" />
              <Form.Item name="pending_duration_sec" noStyle>
                <InputNumber style={{ width: 120 }} min={0} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="time_window_sec" text="评估窗口(s)" />
              <Form.Item name="time_window_sec" noStyle>
                <InputNumber style={{ width: 120 }} min={30} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="reduce_mode" text="缩减模式" />
              <Form.Item name="reduce_mode" noStyle>
                <Select style={{ width: 160 }} options={REDUCE_MODE_OPTIONS} />
              </Form.Item>
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="silence_minutes" text="静默时间(min)" />
              <Form.Item name="silence_minutes" noStyle>
                <InputNumber style={{ width: 120 }} min={0} />
              </Form.Item>
            </div>
          </div>

          {/* 通知渠道：独占一行 */}
          <div style={fieldItemStyle}>
            <HintLabel field="channel_ids" text="通知渠道" />
            <Form.Item name="channel_ids" noStyle rules={[{ required: true, message: '请至少选择一个通知渠道' }]}>
              <Select
                mode="multiple"
                style={{ flex: 1, minWidth: 200 }}
                placeholder="选择通知渠道"
                options={channels.map((c) => ({ value: c.id, label: `${c.name} (${c.type})` }))}
              />
            </Form.Item>
          </div>
        </Form>
      </Modal>

    </Card>
  )
}