import { useEffect, useState, useCallback } from 'react'
import {
  Card,
  Tabs,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  InputNumber,
  Space,
  Tag,
  message,
  Typography,
  Slider,
  Tooltip,
} from 'antd'
import { PlusOutlined, ReloadOutlined, QuestionCircleOutlined, DownOutlined, UpOutlined, ApiOutlined } from '@ant-design/icons'
import {
  listClassifierConfigs,
  createClassifierConfig,
  updateClassifierConfig,
  deleteClassifierConfig,
  type ClassifierConfig,
} from '../../api/classifierConfig'
import {
  listLLMProviders,
  createLLMProvider,
  updateLLMProvider,
  deleteLLMProvider,
  getCircuitBreakerStatus,
  getCircuitBreakerConfig,
  getRetryConfig,
  listLLMConfigs,
  updateLLMConfig,
  testNewLLMProvider,
  testExistingLLMProvider,
  type LLMProvider,
  type CBStatus,
  type CircuitBreakerConfig,
  type RetryConfig,
  type TestConnectionResult,
} from '../../api/llmProvider'

const { TextArea } = Input

const TYPE_LABELS: Record<string, string> = {
  prompt: '提示词模板',
  category: '智能分类',
  business_module: '智能归属业务',
}

const STATE_COLORS: Record<string, string> = {
  closed: 'success',
  half_open: 'warning',
  open: 'error',
}

const STATE_LABELS: Record<string, string> = {
  closed: '正常',
  half_open: '半开（探测中）',
  open: '熔断',
}

const TAB_KEYS = ['prompt', 'category', 'business_module', 'llm_provider'] as const
type TabKey = typeof TAB_KEYS[number]

const TAB_LABELS: Record<TabKey, string> = {
  prompt: '提示词模板',
  category: '智能分类',
  business_module: '智能归属业务',
  llm_provider: 'LLM Provider',
}

const TAB_TIPS: Record<TabKey, string> = {
  prompt: '管理 LLM 分类使用的提示词模板，支持变量替换',
  category: '定义反馈的分类选项，帮助 LLM 判断反馈类型',
  business_module: '定义业务归属模块及其描述，帮助 LLM 判断反馈归属',
  llm_provider: '管理 LLM 服务提供商、监控熔断状态、配置熔断与重试策略',
}

function CollapsibleText({ text, maxRows, id, expandedSet, toggle }: {
  text: string, maxRows: number, id: number, expandedSet: Set<number>, toggle: (id: number) => void
}) {
  const expanded = expandedSet.has(id)
  return (
    <div>
      <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', ...(expanded ? {} : { display: '-webkit-box', WebkitLineClamp: maxRows, WebkitBoxOrient: 'vertical', overflow: 'hidden' }) }}>
        {text}
      </div>
      <a onClick={() => toggle(id)} style={{ fontSize: 12 }}>{expanded ? '收起' : '展开'}</a>
    </div>
  )
}

export default function ClassifierConfigList() {
  const [activeTab, setActiveTab] = useState<TabKey>('prompt')

  // --- Classifier Config state ---
  const [configs, setConfigs] = useState<ClassifierConfig[]>([])
  const [clfLoading, setClfLoading] = useState(false)
  const [clfModalOpen, setClfModalOpen] = useState(false)
  const [editClfItem, setEditClfItem] = useState<ClassifierConfig | null>(null)
  const [clfForm] = Form.useForm()
  const [clfExpanded, setClfExpanded] = useState<Set<number>>(new Set())
  const toggleClfExpanded = (id: number) => setClfExpanded(prev => { const next = new Set(prev); next.has(id) ? next.delete(id) : next.add(id); return next })

  // --- LLM Provider state ---
  const [providers, setProviders] = useState<LLMProvider[]>([])
  const [provLoading, setProvLoading] = useState(false)
  const [provTestingId, setProvTestingId] = useState<number | null>(null)
  const [provModalOpen, setProvModalOpen] = useState(false)
  const [editProvItem, setEditProvItem] = useState<LLMProvider | null>(null)
  const [provForm] = Form.useForm()

  // --- CB Status & Config state ---
  const [cbStatus, setCBStatus] = useState<CBStatus[]>([])
  const [cbConfig, setCBConfig] = useState<CircuitBreakerConfig | null>(null)
  const [retryConfig, setRetryConfig] = useState<RetryConfig | null>(null)
  const [cbConfigCollapsed, setCBConfigCollapsed] = useState(true)

  // --- Fetch functions ---
  const fetchConfigs = useCallback(async () => {
    setClfLoading(true)
    try {
      const rows = await listClassifierConfigs()
      setConfigs(rows ?? [])
    } finally {
      setClfLoading(false)
    }
  }, [])

  const fetchProviders = useCallback(async () => {
    setProvLoading(true)
    try {
      const rows = await listLLMProviders()
      setProviders(rows ?? [])
    } finally {
      setProvLoading(false)
    }
  }, [])

  const fetchCBStatus = useCallback(async () => {
    try {
      const status = await getCircuitBreakerStatus()
      setCBStatus(status ?? [])
    } catch {
      setCBStatus([])
    }
  }, [])

  const fetchCBConfigs = useCallback(async () => {
    try {
      const cb = await getCircuitBreakerConfig()
      setCBConfig(cb ?? null)
      const retry = await getRetryConfig()
      setRetryConfig(retry ?? null)
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    fetchConfigs()
    fetchProviders()
    fetchCBStatus()
    fetchCBConfigs()
  }, [fetchConfigs, fetchProviders, fetchCBStatus, fetchCBConfigs])

  // Auto-refresh CB status every 10s when on llm_provider tab
  useEffect(() => {
    if (activeTab !== 'llm_provider') return
    const timer = setInterval(fetchCBStatus, 10000)
    return () => clearInterval(timer)
  }, [activeTab, fetchCBStatus])

  // --- Classifier Config handlers ---
  const filteredConfigs = configs.filter((c) => c.type === activeTab)

  const openCreateClf = () => {
    setEditClfItem(null)
    clfForm.resetFields()
    clfForm.setFieldsValue({ type: activeTab, sort_order: 0, enabled: true, is_default: false })
    setClfModalOpen(true)
  }

  const openEditClf = (item: ClassifierConfig) => {
    setEditClfItem(item)
    clfForm.setFieldsValue({
      type: item.type,
      name: item.name,
      content: item.content,
      sort_order: item.sort_order,
      enabled: item.enabled === 1,
      is_default: item.is_default === 1,
    })
    setClfModalOpen(true)
  }

  const handleClfSubmit = async () => {
    const values = await clfForm.validateFields()
    const payload = {
      ...values,
      enabled: values.enabled ? 1 : 0,
      is_default: values.is_default ? 1 : 0,
    }
    if (editClfItem) {
      await updateClassifierConfig(editClfItem.id, payload)
      message.success('配置已更新')
    } else {
      await createClassifierConfig(payload)
      message.success('配置已创建')
    }
    setClfModalOpen(false)
    fetchConfigs()
  }

  const handleDeleteClf = async (id: number) => {
    await deleteClassifierConfig(id)
    message.success('配置已删除')
    fetchConfigs()
  }

  const clfColumns = [
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true },
    ...(activeTab === 'prompt'
      ? [
          {
            title: '内容',
            dataIndex: 'content',
            width: 300,
            render: (t: string, row: ClassifierConfig) => (
              <CollapsibleText text={t} maxRows={2} id={row.id} expandedSet={clfExpanded} toggle={toggleClfExpanded} />
            ),
          },
        ]
      : activeTab === 'business_module'
      ? [{ title: '描述', dataIndex: 'content', width: 200 }]
      : []),
    { title: '排序', dataIndex: 'sort_order', width: 80 },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 80,
      render: (v: number) => v === 1 ? <Tag color="success">启用</Tag> : <Tag color="default">禁用</Tag>,
    },
    ...(activeTab === 'prompt'
      ? [
          {
            title: '默认',
            dataIndex: 'is_default',
            width: 80,
            render: (v: number) => v === 1 ? <Tag color="blue">默认</Tag> : <Tag>非默认</Tag>,
          },
        ]
      : []),
    {
      title: '操作',
      width: 120,
      render: (_: unknown, row: ClassifierConfig) => (
        <Space>
          <Button type="link" size="small" onClick={() => openEditClf(row)}>编辑</Button>
          <Button type="link" size="small" danger onClick={() => handleDeleteClf(row.id)}>删除</Button>
        </Space>
      ),
    },
  ]

  // --- LLM Provider handlers ---
  const openCreateProv = () => {
    setEditProvItem(null)
    provForm.resetFields()
    provForm.setFieldsValue({ priority: 1, timeout_ms: 30000, status: true })
    setProvModalOpen(true)
  }

  const openEditProv = (item: LLMProvider) => {
    setEditProvItem(item)
    provForm.setFieldsValue({
      name: item.name,
      base_url: item.base_url,
      api_key: '',
      model: item.model,
      timeout_ms: item.timeout_ms,
      priority: item.priority,
      status: item.status === 1,
    })
    setProvModalOpen(true)
  }

  const handleProvSubmit = async () => {
    const values = await provForm.validateFields()
    const payload = { ...values, status: values.status ? 1 : 0 }
    if (editProvItem) {
      if (!payload.api_key || payload.api_key === '***') {
        delete payload.api_key
      }
      await updateLLMProvider(editProvItem.id, payload)
      message.success('Provider 已更新')
    } else {
      await createLLMProvider(payload)
      message.success('Provider 已创建')
    }
    setProvModalOpen(false)
    fetchProviders()
  }

  const handleDeleteProv = async (id: number) => {
    await deleteLLMProvider(id)
    message.success('Provider 已删除')
    fetchProviders()
  }

  const handleTestProv = async (id: number) => {
    setProvTestingId(id)
    try {
      const result = await testExistingLLMProvider(id)
      if (result.success) {
        message.success(`连接成功 (延迟: ${result.latency_ms}ms, 模型: ${result.model})`)
      } else {
        message.error(result.message)
      }
    } catch (e: unknown) {
      message.error('测试请求失败: ' + (e instanceof Error ? e.message : String(e)))
    } finally {
      setProvTestingId(null)
    }
  }

  const handleModalTestProv = async () => {
    const values = await provForm.validateFields()
    if (editProvItem) {
      // 编辑模式：使用已保存的配置测试
      setProvTestingId(editProvItem.id)
      try {
        const result = await testExistingLLMProvider(editProvItem.id)
        if (result.success) {
          message.success(`连接成功 (延迟: ${result.latency_ms}ms, 模型: ${result.model})`)
        } else {
          message.error(result.message)
        }
      } catch (e: unknown) {
        message.error('测试请求失败: ' + (e instanceof Error ? e.message : String(e)))
      } finally {
        setProvTestingId(null)
      }
    } else {
      // 新增模式：使用表单中的配置测试
      if (!values.api_key) {
        message.warning('请先填写 API Key')
        return
      }
      setProvTestingId(-1) // 用 -1 表示新增模式的测试
      try {
        const result = await testNewLLMProvider({
          base_url: values.base_url,
          api_key: values.api_key,
          model: values.model,
          timeout_ms: values.timeout_ms,
        })
        if (result.success) {
          message.success(`连接成功 (延迟: ${result.latency_ms}ms, 模型: ${result.model})`)
        } else {
          message.error(result.message)
        }
      } catch (e: unknown) {
        message.error('测试请求失败: ' + (e instanceof Error ? e.message : String(e)))
      } finally {
        setProvTestingId(null)
      }
    }
  }

  const handleCBConfigSave = async () => {
    if (!cbConfig) return
    const configs = await listLLMConfigs()
    const cbRow = configs?.find((c) => c.type === 'circuit_breaker' && c.is_default === 1)
    if (cbRow) {
      await updateLLMConfig(cbRow.id, { content: JSON.stringify(cbConfig) })
      message.success('熔断配置已更新')
    }
  }

  const handleRetryConfigSave = async () => {
    if (!retryConfig) return
    const configs = await listLLMConfigs()
    const retryRow = configs?.find((c) => c.type === 'retry' && c.is_default === 1)
    if (retryRow) {
      await updateLLMConfig(retryRow.id, { content: JSON.stringify(retryConfig) })
      message.success('重试配置已更新')
    }
  }

  const provColumns = [
    { title: '名称', dataIndex: 'name', width: 140, ellipsis: true },
    { title: 'Base URL', dataIndex: 'base_url', width: 240, ellipsis: true },
    { title: '模型', dataIndex: 'model', width: 120 },
    { title: 'API Key', dataIndex: 'api_key', width: 80, render: () => '***' },
    { title: '超时(ms)', dataIndex: 'timeout_ms', width: 100 },
    { title: '优先级', dataIndex: 'priority', width: 80, render: (v: number) => <Tag color={v === 1 ? 'blue' : 'default'}>{v}</Tag> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (v: number) => v === 1 ? <Tag color="success">启用</Tag> : <Tag color="default">禁用</Tag>,
    },
    {
      title: '操作',
      width: 160,
      render: (_: unknown, row: LLMProvider) => (
        <Space>
          <Button type="link" size="small" icon={<ApiOutlined />} loading={provTestingId === row.id} onClick={() => handleTestProv(row.id)}>测试</Button>
          <Button type="link" size="small" onClick={() => openEditProv(row)}>编辑</Button>
          <Button type="link" size="small" danger onClick={() => handleDeleteProv(row.id)}>删除</Button>
        </Space>
      ),
    },
  ]

  const statusColumns = [
    { title: 'Provider', dataIndex: 'provider_name', width: 140 },
    {
      title: '状态',
      dataIndex: 'state',
      width: 140,
      render: (v: string) => <Tag color={STATE_COLORS[v] || 'default'}>{STATE_LABELS[v] || v}</Tag>,
    },
    { title: '连续失败', dataIndex: 'consecutive_failures', width: 100 },
    { title: '总失败', dataIndex: 'total_failures', width: 80 },
    { title: '总成功', dataIndex: 'total_successes', width: 80 },
    { title: '总请求', dataIndex: 'requests', width: 80 },
  ]

  return (
    <Card>
      <Tabs
        activeKey={activeTab}
        onChange={(k) => setActiveTab(k as TabKey)}
        items={TAB_KEYS.map((k) => ({
          key: k,
          label: (
            <Space size={4}>
              {TAB_LABELS[k]}
              <Tooltip title={TAB_TIPS[k]}>
                <QuestionCircleOutlined style={{ color: '#999', fontSize: 12 }} />
              </Tooltip>
            </Space>
          ),
        }))}
      />

      {/* Classifier Config tabs */}
      {(activeTab === 'prompt' || activeTab === 'category' || activeTab === 'business_module') && (
        <>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateClf} style={{ marginBottom: 16 }}>
            新增{TYPE_LABELS[activeTab]}
          </Button>
          <Table rowKey="id" loading={clfLoading} columns={clfColumns} dataSource={filteredConfigs} pagination={false} scroll={{ x: 'max-content' }} />
        </>
      )}

      {/* LLM Provider tab — 三张卡片 */}
      {activeTab === 'llm_provider' && (
        <>
          {/* LLM Provider 管理 */}
          <Card title="LLM Provider 管理" style={{ marginBottom: 24 }}>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreateProv} style={{ marginBottom: 16 }}>
              新增 Provider
            </Button>
            <Table rowKey="id" loading={provLoading} columns={provColumns} dataSource={providers} pagination={false} scroll={{ x: 'max-content' }} />
          </Card>

          {/* 熔断状态监控 */}
          <Card
            title="熔断状态监控"
            style={{ marginBottom: 24 }}
            extra={<Button icon={<ReloadOutlined />} onClick={fetchCBStatus} size="small">刷新</Button>}
          >
            <Typography.Text type="secondary" style={{ marginBottom: 12, display: 'block' }}>
              统计数据按「统计窗口」（默认 60 秒）周期刷新，窗口到期后自动归零重新计数；页面每 10 秒自动刷新
            </Typography.Text>
            <Table rowKey="provider_id" columns={statusColumns} dataSource={cbStatus} pagination={false} size="small" scroll={{ x: 'max-content' }} />
          </Card>

          {/* 熔断与重试配置 — 可折叠 */}
          <Card
            title={
              <Space style={{ cursor: 'pointer' }} onClick={() => setCBConfigCollapsed(!cbConfigCollapsed)}>
                熔断与重试配置
                {cbConfigCollapsed ? <DownOutlined style={{ fontSize: 12 }} /> : <UpOutlined style={{ fontSize: 12 }} />}
              </Space>
            }
            extra={
              <Button type="text" size="small" onClick={() => setCBConfigCollapsed(!cbConfigCollapsed)}>
                {cbConfigCollapsed ? '展开' : '折叠'}
              </Button>
            }
          >
            {!cbConfigCollapsed && (
              <>
                <Typography.Text type="secondary" style={{ marginBottom: 12, display: 'block' }}>
                  配置熔断策略（何时触发熔断）、重试机制（尝试次数与策略）及请求超时
                </Typography.Text>
                <Card title="熔断策略配置" style={{ marginBottom: 24 }} size="small">
                  <Typography.Text type="secondary" style={{ marginBottom: 12, display: 'block' }}>
                    熔断器保护系统免受持续失败的 Provider 影响：连续失败达到阈值或失败率过高时自动熔断，冷却期后允许探测恢复
                  </Typography.Text>
                  {cbConfig && (
                    <Form layout="vertical">
                      <Form.Item label="连续失败次数阈值">
                        <InputNumber min={1} max={50} value={cbConfig.failure_threshold} onChange={(v) => setCBConfig({ ...cbConfig, failure_threshold: v ?? 5 })} />
                      </Form.Item>
                      <Form.Item label="失败率阈值">
                        <Slider min={0} max={1} step={0.05} value={cbConfig.failure_rate_threshold} onChange={(v) => setCBConfig({ ...cbConfig, failure_rate_threshold: v })} marks={{ 0: '0%', 0.5: '50%', 1: '100%' }} />
                        <Typography.Text type="secondary">当前: {(cbConfig.failure_rate_threshold * 100).toFixed(0)}%</Typography.Text>
                      </Form.Item>
                      <Form.Item label="统计窗口（秒）">
                        <InputNumber min={10} max={300} value={cbConfig.window_seconds} onChange={(v) => setCBConfig({ ...cbConfig, window_seconds: v ?? 60 })} />
                        <Typography.Text type="secondary" style={{ marginLeft: 8 }}>统计数据在此周期内累计，窗口到期后自动归零</Typography.Text>
                      </Form.Item>
                      <Form.Item label="冷却期（秒）">
                        <InputNumber min={5} max={300} value={cbConfig.cooldown_seconds} onChange={(v) => setCBConfig({ ...cbConfig, cooldown_seconds: v ?? 30 })} />
                      </Form.Item>
                      <Form.Item label="最小请求数（率阈值生效前）">
                        <InputNumber min={1} max={20} value={cbConfig.min_requests} onChange={(v) => setCBConfig({ ...cbConfig, min_requests: v ?? 3 })} />
                      </Form.Item>
                      <Button type="primary" onClick={handleCBConfigSave}>保存熔断配置</Button>
                    </Form>
                  )}
                </Card>
                <Card title="重试机制配置" size="small">
                  <Typography.Text type="secondary" style={{ marginBottom: 12, display: 'block' }}>
                    重试机制决定分类失败后尝试其他 Provider 的行为；超时为单个 LLM 请求的最大等待时间
                  </Typography.Text>
                  {retryConfig && (
                    <Form layout="vertical">
                      <Form.Item label="请求超时(ms)">
                        <InputNumber min={5000} max={300000} step={1000} value={retryConfig.timeout_ms} onChange={(v) => setRetryConfig({ ...retryConfig, timeout_ms: v ?? 30000 })} style={{ width: 200 }} />
                        <Typography.Text type="secondary" style={{ marginLeft: 8 }}>单个 LLM 请求的最大等待时间，Provider 级别优先</Typography.Text>
                      </Form.Item>
                      <Form.Item label="最大重试次数">
                        <InputNumber min={1} max={10} value={retryConfig.max_retries} onChange={(v) => setRetryConfig({ ...retryConfig, max_retries: v ?? 3 })} />
                      </Form.Item>
                      <Form.Item label="重试策略">
                        <Space>
                          <Tag color={retryConfig.strategy === 'priority' ? 'blue' : 'default'} style={{ cursor: 'pointer' }} onClick={() => setRetryConfig({ ...retryConfig, strategy: 'priority' })}>按优先级</Tag>
                          <Tag color={retryConfig.strategy === 'random' ? 'blue' : 'default'} style={{ cursor: 'pointer' }} onClick={() => setRetryConfig({ ...retryConfig, strategy: 'random' })}>随机选择</Tag>
                        </Space>
                      </Form.Item>
                      <Button type="primary" onClick={handleRetryConfigSave}>保存重试配置</Button>
                    </Form>
                  )}
                </Card>
              </>
            )}
          </Card>
        </>
      )}

      {/* Classifier Config Modal */}
      <Modal
        title={editClfItem ? `编辑${TYPE_LABELS[activeTab]}` : `新增${TYPE_LABELS[activeTab]}`}
        open={clfModalOpen}
        onOk={handleClfSubmit}
        onCancel={() => setClfModalOpen(false)}
        destroyOnClose
      >
        <Form form={clfForm} layout="vertical">
          <Form.Item name="type" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true, max: 128 }]}>
            <Input />
          </Form.Item>
          {activeTab === 'prompt' && (
            <Form.Item name="content" label="提示词模板" rules={[{ required: true }]}>
              <TextArea rows={8} placeholder={`可用变量：{categories}、{business_modules}、{content}`} />
            </Form.Item>
          )}
          {activeTab === 'business_module' && (
            <Form.Item name="content" label="描述（帮助 LLM 判断归属）">
              <TextArea rows={3} placeholder="简要描述该模块涵盖的功能范围" />
            </Form.Item>
          )}
          <Form.Item name="sort_order" label="排序">
            <InputNumber min={0} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          {activeTab === 'prompt' && (
            <Form.Item name="is_default" label="默认模板" valuePropName="checked">
              <Switch />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* LLM Provider Modal */}
      <Modal
        title={editProvItem ? '编辑 Provider' : '新增 Provider'}
        open={provModalOpen}
        onOk={handleProvSubmit}
        onCancel={() => setProvModalOpen(false)}
        destroyOnClose
        footer={[
          <Button key="test" icon={<ApiOutlined />} loading={provTestingId !== null} onClick={handleModalTestProv}>
            测试连通性
          </Button>,
          <Button key="cancel" onClick={() => setProvModalOpen(false)}>
            取消
          </Button>,
          <Button key="submit" type="primary" onClick={handleProvSubmit}>
            {editProvItem ? '保存' : '创建'}
          </Button>,
        ]}
      >
        <Form form={provForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, max: 128 }]}>
            <Input placeholder="如: openai-gpt4" />
          </Form.Item>
          <Form.Item name="base_url" label="API Base URL" rules={[{ required: true }]}>
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="api_key" label="API Key" rules={editProvItem ? [] : [{ required: true }]}>
            <Input placeholder={editProvItem ? '留空保留原密钥，填写则更新' : '输入 API Key'} />
          </Form.Item>
          <Form.Item name="model" label="模型" rules={[{ required: true }]}>
            <Input placeholder="如: gpt-4o-mini" />
          </Form.Item>
          <Form.Item name="timeout_ms" label="超时(ms)">
            <InputNumber min={1000} max={120000} step={1000} />
          </Form.Item>
          <Form.Item name="priority" label="优先级（1=最高）">
            <InputNumber min={1} max={10} />
          </Form.Item>
          <Form.Item name="status" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}