import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  message,
  Popconfirm,
  Tooltip,
  Collapse,
  Segmented,
} from 'antd'
import {
  PlusOutlined,
  SyncOutlined,
  EditOutlined,
  DeleteOutlined,
  UnorderedListOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import {
  listDataSources,
  createDataSource,
  updateDataSource,
  deleteDataSource,
  syncDataSource,
  listSyncLogs,
  type DataSource,
  type SyncLog,
  type DataSourceType,
  type HttpApiConfig,
  DATA_SOURCE_TYPE_OPTIONS,
  DEFAULT_MYSQL_CONFIG_JSON,
} from '../../api/datasource'
import { SYNC_STATUS_MAP } from '../../constants/enums'
import JsonEditor from '../../components/JsonEditor'
import HttpApiFormConfig from './HttpApiFormConfig'

const STATUS_MAP: Record<number, { label: string; color: string }> = {
  0: { label: '空闲', color: 'default' },
  1: { label: '同步中', color: 'processing' },
  2: { label: '成功', color: 'success' },
  3: { label: '失败', color: 'error' },
}

const DEFAULT_CONFIG: HttpApiConfig = {
  endpoint: '',
  method: 'GET',
  headers: {},
  params_template: {},
  data_path: 'data.list',
  page_paginate: { param: 'page', start: 1 },
  field_mapping: { original_id: 'id', content: 'content' },
  time_layout: '2006-01-02 15:04:05',
  max_pages: 50,
  stop_when_seen: false,
}

const DEFAULT_CONFIG_JSON = JSON.stringify(DEFAULT_CONFIG, null, 2)

const FIELD_REFERENCE = [
  { field: 'original_id', required: true, desc: '反馈原始 ID，用于增量同步判断是否已存在' },
  { field: 'content', required: true, desc: '反馈正文内容' },
  { field: 'app_id', desc: '应用 ID（如 1=AppA, 2=AppB），枚举映射在「枚举配置」中管理' },
  { field: 'app_name', desc: '应用名称字符串，如"AppA"' },
  { field: 'platform', desc: '平台标识（如 ios/android），枚举映射在「枚举配置」中管理' },
  { field: 'platform_id', desc: '平台数字 ID' },
  { field: 'user_id', desc: '用户 ID' },
  { field: 'user_name', desc: '用户昵称/显示名' },
  { field: 'user_mode', desc: '用户模式（如 1=游客），枚举映射在「枚举配置」中管理' },
  { field: 'images', desc: '图片 URL 列表，源 API 可以是数组或单个字符串' },
  { field: 'videos', desc: '视频 URL 列表，源 API 可以是数组或单个字符串' },
  { field: 'phone_model', desc: '手机型号（如 iPhone16,1）' },
  { field: 'app_version', desc: '应用版本号' },
  { field: 'channel_id', desc: '渠道 ID' },
  { field: 'qq', desc: 'QQ 号' },
  { field: 'file_url', desc: '附件文件 URL' },
  { field: 'original_created_at', desc: '反馈原始创建时间（用户提交时间）' },
]

const CONFIG_REFERENCE = [
  { key: 'endpoint', required: true, desc: '外部 API 地址' },
  { key: 'method', desc: 'HTTP 方法，默认 GET' },
  { key: 'headers', desc: '请求头（cookie/authorization 会加密存储）' },
  { key: 'cookie', desc: 'Cookie 字符串，保存时自动加密，显示时掩码为 ***' },
  { key: 'params_template', desc: '固定查询参数模板，支持 {{变量}} 渲染' },
  { key: 'data_path', desc: 'API 响应中反馈列表的 JSON 路径（如 data.feedback_question_list）' },
  { key: 'page_paginate', desc: '分页参数配置：{ param: "page参数名", start: 起始页码 }' },
  { key: 'field_mapping', required: true, desc: '源字段 → 系统字段映射。original_id 和 content 必填' },
  { key: 'time_layout', desc: '时间格式，默认 "2006-01-02 15:04:05"（Go 格式）' },
  { key: 'max_pages', desc: '最大拉取页数，默认 50' },
  { key: 'stop_when_seen', desc: '遇到已存在 ID 时停止翻页，默认 false' },
  { key: 'iterate', desc: '变量组列表，用于多参数组合批量拉取' },
]

type ConfigMode = 'json' | 'form'

export default function DataSourceList() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<DataSource[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DataSource | null>(null)
  const [syncing, setSyncing] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [syncLogs, setSyncLogs] = useState<SyncLog[]>([])
  const [logSourceId, setLogSourceId] = useState<number | null>(null)
  const [logsLoading, setLogsLoading] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)

  const [configMode, setConfigMode] = useState<ConfigMode>('json')
  const [configStr, setConfigStr] = useState(DEFAULT_CONFIG_JSON)
  const [formConfig, setFormConfig] = useState<HttpApiConfig>(DEFAULT_CONFIG)
  const [dsType, setDsType] = useState<DataSourceType>('http_api')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDataSources()
      setData(res ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const parseConfigStr = useCallback((): HttpApiConfig | null => {
    try {
      return JSON.parse(configStr) as HttpApiConfig
    } catch {
      return null
    }
  }, [configStr])

  const switchToFormMode = useCallback(() => {
    const parsed = parseConfigStr()
    if (!parsed) {
      message.error('JSON 格式错误，无法切换到表单模式')
      return
    }
    setFormConfig(parsed)
    setConfigMode('form')
  }, [parseConfigStr])

  const switchToJsonMode = useCallback(() => {
    setConfigStr(JSON.stringify(formConfig, null, 2))
    setConfigMode('json')
  }, [formConfig])

  const openCreate = () => {
    setEditing(null)
    setDsType('http_api')
    form.resetFields()
    setConfigStr(DEFAULT_CONFIG_JSON)
    setFormConfig(DEFAULT_CONFIG)
    setConfigMode('json')
    setModalOpen(true)
  }

  // 切换类型时载入对应类型的默认配置模板
  const handleTypeChange = (v: DataSourceType) => {
    setDsType(v)
    if (editing) return // 编辑态不改类型
    if (v === 'mysql') {
      setConfigStr(DEFAULT_MYSQL_CONFIG_JSON)
      setConfigMode('json')
    } else {
      setConfigStr(DEFAULT_CONFIG_JSON)
      setConfigMode('json')
    }
  }

  const openEdit = (row: DataSource) => {
    setEditing(row)
    setDsType(row.type as DataSourceType)
    form.setFieldsValue({
      name: row.name,
      sync_cron: row.sync_cron,
    })
    setConfigStr(JSON.stringify(row.config, null, 2))
    setFormConfig(row.config as unknown as HttpApiConfig)
    setConfigMode('json')
    setModalOpen(true)
  }

  const handleSubmit = async (values: { name: string; sync_cron: string }) => {
    let config: Record<string, unknown>
    if (configMode === 'json') {
      const parsed = parseConfigStr()
      if (!parsed) {
        message.error('配置 JSON 格式错误')
        return
      }
      config = parsed as unknown as Record<string, unknown>
    } else {
      config = formConfig as unknown as Record<string, unknown>
    }
    const payload = { name: values.name, type: dsType, sync_cron: values.sync_cron, config }
    if (editing) {
      await updateDataSource(editing.id, payload)
      message.success('已更新')
    } else {
      await createDataSource(payload)
      message.success('已创建')
    }
    setModalOpen(false)
    fetchData()
  }

  const handleDelete = async (id: number) => {
    await deleteDataSource(id)
    message.success('已删除')
    fetchData()
  }

  const handleSync = async (id: number) => {
    setSyncing(id)
    try {
      const res = await syncDataSource(id)
      message.success(`同步完成，新增 ${res.inserted} 条`)
      fetchData()
    } finally {
      setSyncing(null)
    }
  }

  const fetchSyncLogs = async (sourceId: number) => {
    setLogSourceId(sourceId)
    setLogsLoading(true)
    try {
      const logs = await listSyncLogs(sourceId, 20)
      setSyncLogs(logs ?? [])
    } finally {
      setLogsLoading(false)
    }
  }

  const columns: ColumnsType<DataSource> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '名称', dataIndex: 'name', width: 150, ellipsis: true },
    { title: '类型', dataIndex: 'type', width: 100 },
    { title: 'Cron', dataIndex: 'sync_cron', width: 130 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (v) => {
        const s = STATUS_MAP[v] ?? { label: String(v), color: 'default' }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    {
      title: '上次同步',
      dataIndex: 'last_sync_at',
      width: 160,
      render: (v) => (v ? dayjs(v).format('MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      width: 160,
      render: (_, row) => (
        <Space size="small">
          <Tooltip title="立即同步">
            <Button
              icon={<SyncOutlined spin={syncing === row.id} />}
              size="small"
              onClick={() => handleSync(row.id)}
            />
          </Tooltip>
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(row)} />
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(row.id)}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
          <Tooltip title="同步日志">
            <Button icon={<UnorderedListOutlined />} size="small" onClick={() => fetchSyncLogs(row.id)} />
          </Tooltip>
        </Space>
      ),
    },
  ]

  const logColumns: ColumnsType<SyncLog> = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (v) => {
        const s = SYNC_STATUS_MAP[v] ?? { label: v, color: 'default' }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    { title: '开始时间', dataIndex: 'started_at', width: 160, render: (v) => dayjs(v).format('MM-DD HH:mm:ss') },
    { title: '结束时间', dataIndex: 'finished_at', width: 160, render: (v) => v ? dayjs(v).format('MM-DD HH:mm:ss') : '-' },
    { title: '获取数', dataIndex: 'fetched_count', width: 80 },
    { title: '入库数', dataIndex: 'inserted_count', width: 80 },
    { title: '错误信息', dataIndex: 'error_message', ellipsis: true },
  ]

  const logSource = data.find((d) => d.id === logSourceId)

  const typeOptions = DATA_SOURCE_TYPE_OPTIONS.map((opt) => ({
    value: opt.value,
    label: opt.disabled ? (
      <Tooltip title={opt.tooltip}>
        <span style={{ color: '#bbb' }}>{opt.label}</span>
      </Tooltip>
    ) : opt.label,
    disabled: opt.disabled,
  }))

  return (
    <Card
      title="数据源管理"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建数据源
        </Button>
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} scroll={{ x: 'max-content' }} />

      <Modal
        title={editing ? '编辑数据源' : '新建数据源'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        width={640}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>

          <Form.Item label="数据源类型">
            <Select
              value={dsType}
              onChange={handleTypeChange}
              options={typeOptions}
              disabled={editing !== null}
            />
          </Form.Item>

          <Form.Item
            label="Cron 表达式"
            name="sync_cron"
            rules={[{ required: true }]}
            extra="例如：*/5 * * * * 表示每 5 分钟（5 字段）；0 */30 * * * * 表示每 30 分钟（6 字段，含秒）"
          >
            <Input placeholder="0 */30 * * * *" />
          </Form.Item>

          {dsType === 'http_api' && (
            <>
              <div style={{ marginBottom: 8 }}>
                <Segmented
                  value={configMode}
                  onChange={(v) => {
                    if (v === 'form') switchToFormMode()
                    else if (v === 'json') switchToJsonMode()
                  }}
                  options={[
                    { value: 'json', label: 'JSON 配置' },
                    { value: 'form', label: '表单配置' },
                  ]}
                />
              </div>

              {configMode === 'json' ? (
                <Form.Item
                  label="配置 (JSON)"
                  extra={
                    <Button size="small" style={{ marginTop: 8 }} icon={<QuestionCircleOutlined />} onClick={() => setHelpOpen(true)}>
                      字段说明
                    </Button>
                  }
                >
                  <JsonEditor value={configStr} onChange={setConfigStr} />
                </Form.Item>
              ) : (
                <HttpApiFormConfig config={formConfig} onChange={setFormConfig} />
              )}
            </>
          )}

          {dsType === 'mysql' && (
            <Form.Item
              label="配置 (JSON)"
              extra={
                <div>
                  <div style={{ marginBottom: 8 }}>
                    需配置：host/port/username/password/database/table/time_field（增量时间列）、
                    field_mapping（original_id / content / original_created_at 必填）。password 保存时自动加密。
                  </div>
                  <Button size="small" icon={<QuestionCircleOutlined />} onClick={() => setHelpOpen(true)}>
                    字段说明
                  </Button>
                </div>
              }
            >
              <JsonEditor value={configStr} onChange={setConfigStr} />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        title="配置字段说明"
        open={helpOpen}
        onCancel={() => setHelpOpen(false)}
        footer={null}
        width={700}
      >
        <Collapse
          items={[
            {
              key: 'config',
              label: '配置项说明',
              children: (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>
                      <th style={{ textAlign: 'left', borderBottom: '1px solid #eee', padding: 8 }}>配置键</th>
                      <th style={{ borderBottom: '1px solid #eee', padding: 8 }}>必填</th>
                      <th style={{ textAlign: 'left', borderBottom: '1px solid #eee', padding: 8 }}>说明</th>
                    </tr>
                  </thead>
                  <tbody>
                    {CONFIG_REFERENCE.map((r) => (
                      <tr key={r.key}>
                        <td style={{ padding: 6, fontFamily: 'monospace' }}>{r.key}</td>
                        <td style={{ padding: 6 }}>{r.required ? '是' : '否'}</td>
                        <td style={{ padding: 6 }}>{r.desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ),
            },
            {
              key: 'fields',
              label: 'field_mapping 字段说明',
              children: (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>
                      <th style={{ textAlign: 'left', borderBottom: '1px solid #eee', padding: 8 }}>系统字段</th>
                      <th style={{ borderBottom: '1px solid #eee', padding: 8 }}>必填</th>
                      <th style={{ textAlign: 'left', borderBottom: '1px solid #eee', padding: 8 }}>说明</th>
                    </tr>
                  </thead>
                  <tbody>
                    {FIELD_REFERENCE.map((r) => (
                      <tr key={r.field}>
                        <td style={{ padding: 6, fontFamily: 'monospace' }}>{r.field}</td>
                        <td style={{ padding: 6 }}>{r.required ? "是" : "否"}</td>
                        <td style={{ padding: 6 }}>{r.desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ),
            },
          ]}
        />
      </Modal>

      <Modal
        title={`同步日志 - ${logSource?.name ?? ''}`}
        open={logSourceId !== null}
        onCancel={() => setLogSourceId(null)}
        footer={null}
        width={800}
      >
        <Table
          rowKey="id"
          loading={logsLoading}
          columns={logColumns}
          dataSource={syncLogs}
          scroll={{ x: 'max-content' }}
        />
      </Modal>
    </Card>
  )
}