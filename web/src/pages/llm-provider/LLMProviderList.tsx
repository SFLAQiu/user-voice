import { useEffect, useState, useCallback } from 'react'
import {
  Card,
  Tabs,
  Table,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  Space,
  Tag,
  message,
  Typography,
  Slider,
  Tooltip,
} from 'antd'
import { PlusOutlined, ReloadOutlined, QuestionCircleOutlined, ApiOutlined } from '@ant-design/icons'
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

const TAB_TIPS: Record<string, string> = {
  providers: '管理 LLM 服务提供商（如 OpenAI），配置连接参数、超时和优先级',
  status: '实时监控各 Provider 的熔断器状态与请求统计（按统计窗口周期刷新，每 10 秒自动更新）',
  config: '配置熔断策略（何时触发熔断）、重试机制（尝试次数与策略）及请求超时',
}

export default function LLMProviderList() {
  const [providers, setProviders] = useState<LLMProvider[]>([])
  const [cbStatus, setCBStatus] = useState<CBStatus[]>([])
  const [cbConfig, setCBConfig] = useState<CircuitBreakerConfig | null>(null)
  const [retryConfig, setRetryConfig] = useState<RetryConfig | null>(null)
  const [loading, setLoading] = useState(false)
  const [testing, setTesting] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editItem, setEditItem] = useState<LLMProvider | null>(null)
  const [activeTab, setActiveTab] = useState('providers')
  const [form] = Form.useForm()

  const fetchProviders = useCallback(async () => {
    setLoading(true)
    try {
      const rows = await listLLMProviders()
      setProviders(rows ?? [])
    } finally {
      setLoading(false)
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

  const fetchConfigs = useCallback(async () => {
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
    fetchProviders()
    fetchCBStatus()
    fetchConfigs()
  }, [fetchProviders, fetchCBStatus, fetchConfigs])

  // Auto-refresh CB status every 10s
  useEffect(() => {
    if (activeTab !== 'status') return
    const timer = setInterval(fetchCBStatus, 10000)
    return () => clearInterval(timer)
  }, [activeTab, fetchCBStatus])

  const openCreate = () => {
    setEditItem(null)
    form.resetFields()
    form.setFieldsValue({ priority: 1, timeout_ms: 30000, status: true })
    setModalOpen(true)
  }

  const openEdit = (item: LLMProvider) => {
    setEditItem(item)
    form.setFieldsValue({
      name: item.name,
      base_url: item.base_url,
      api_key: '',
      model: item.model,
      timeout_ms: item.timeout_ms,
      priority: item.priority,
      status: item.status === 1,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    const values = await form.validateFields()
    const payload = {
      ...values,
      status: values.status ? 1 : 0,
    }
    if (editItem) {
      // Only send api_key if user entered a new value
      if (!payload.api_key || payload.api_key === '***') {
        delete payload.api_key
      }
      await updateLLMProvider(editItem.id, payload)
      message.success('Provider 已更新')
    } else {
      await createLLMProvider(payload)
      message.success('Provider 已创建')
    }
    setModalOpen(false)
    fetchProviders()
  }

  const handleDelete = async (id: number) => {
    await deleteLLMProvider(id)
    message.success('Provider 已删除')
    fetchProviders()
  }

  const handleTest = async (id: number) => {
    setTesting(true)
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
      setTesting(false)
    }
  }

  const handleModalTest = async () => {
    const values = await form.validateFields()
    if (editItem) {
      // 编辑模式：使用已保存的配置测试
      setTesting(true)
      try {
        const result = await testExistingLLMProvider(editItem.id)
        if (result.success) {
          message.success(`连接成功 (延迟: ${result.latency_ms}ms, 模型: ${result.model})`)
        } else {
          message.error(result.message)
        }
      } catch (e: unknown) {
        message.error('测试请求失败: ' + (e instanceof Error ? e.message : String(e)))
      } finally {
        setTesting(false)
      }
    } else {
      // 新增模式：使用表单中的配置测试
      if (!values.api_key) {
        message.warning('请先填写 API Key')
        return
      }
      setTesting(true)
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
        setTesting(false)
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

  const providerColumns = [
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
      width: 120,
      render: (_: unknown, row: LLMProvider) => (
        <Space>
          <Button type="link" size="small" icon={<ApiOutlined />} loading={testing} onClick={() => handleTest(row.id)}>测试</Button>
          <Button type="link" size="small" onClick={() => openEdit(row)}>编辑</Button>
          <Button type="link" size="small" danger onClick={() => handleDelete(row.id)}>删除</Button>
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
        onChange={setActiveTab}
        items={[
          {
            key: 'providers',
            label: (
              <Space size={4}>
                LLM Provider 管理
                <Tooltip title={TAB_TIPS.providers}>
                  <QuestionCircleOutlined style={{ color: '#999', fontSize: 12 }} />
                </Tooltip>
              </Space>
            ),
          },
          {
            key: 'status',
            label: (
              <Space size={4}>
                熔断状态监控
                <Tooltip title={TAB_TIPS.status}>
                  <QuestionCircleOutlined style={{ color: '#999', fontSize: 12 }} />
                </Tooltip>
              </Space>
            ),
          },
          {
            key: 'config',
            label: (
              <Space size={4}>
                配置管理
                <Tooltip title={TAB_TIPS.config}>
                  <QuestionCircleOutlined style={{ color: '#999', fontSize: 12 }} />
                </Tooltip>
              </Space>
            ),
          },
        ]}
      />

      {activeTab === 'providers' && (
        <>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} style={{ marginBottom: 16 }}>
            新增 Provider
          </Button>
          <Table rowKey="id" loading={loading} columns={providerColumns} dataSource={providers} pagination={false} scroll={{ x: 'max-content' }} />
        </>
      )}

      {activeTab === 'status' && (
        <>
          <Typography.Text type="secondary" style={{ marginBottom: 16, display: 'block' }}>
            统计数据按「统计窗口」（默认 60 秒）周期刷新，窗口到期后自动归零重新计数；页面每 10 秒自动刷新
          </Typography.Text>
          <Button icon={<ReloadOutlined />} onClick={fetchCBStatus} style={{ marginBottom: 12 }}>
            刷新状态
          </Button>
          <Table rowKey="provider_id" columns={statusColumns} dataSource={cbStatus} pagination={false} scroll={{ x: 'max-content' }} />
        </>
      )}

      {activeTab === 'config' && (
        <>
          <Card title="熔断策略配置" style={{ marginBottom: 24 }}>
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
          <Card title="重试机制配置">
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

      <Modal
        title={editItem ? '编辑 Provider' : '新增 Provider'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
        footer={[
          <Button key="test" icon={<ApiOutlined />} loading={testing} onClick={handleModalTest}>
            测试连通性
          </Button>,
          <Button key="cancel" onClick={() => setModalOpen(false)}>
            取消
          </Button>,
          <Button key="submit" type="primary" onClick={handleSubmit}>
            {editItem ? '保存' : '创建'}
          </Button>,
        ]}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, max: 128 }]}>
            <Input placeholder="如: openai-gpt4" />
          </Form.Item>
          <Form.Item name="base_url" label="API Base URL" rules={[{ required: true }]}>
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="api_key" label="API Key" rules={editItem ? [] : [{ required: true }]}>
            <Input placeholder={editItem ? '留空保留原密钥，填写则更新' : '输入 API Key'} />
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