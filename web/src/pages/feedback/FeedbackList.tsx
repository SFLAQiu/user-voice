import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Input,
  Select,
  DatePicker,
  Space,
  Tag,
  Button,
  Drawer,
  Descriptions,
  Typography,
  Form,
  Image,
  Collapse,
  message,
} from 'antd'
import { SearchOutlined, ReloadOutlined, ClockCircleOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import {
  listFeedbacks,
  getFeedback,
  updateCategory,
  type Feedback,
  type FeedbackQuery,
} from '../../api/feedback'
import { aggregatedEnums } from '../../api/enumConfig'
import { listClassificationLogs, type ClassificationLog } from '../../api/classifierConfig'
import { getClassifierEnums } from '../../api/classifierConfig'
import { CATEGORY_STATUS_MAP, LOG_STATUS_MAP, SENTIMENTS, CATEGORIES, BUSINESS_MODULES, type DynamicEnums } from '../../constants/enums'

const { RangePicker } = DatePicker

const FALLBACK_ENUMS: DynamicEnums = {
  app_id: { '1': 'AppA', '2': 'AppB' },
  platform_id: { '1': 'Android', '2': 'iOS' },
  user_mode: { '1': '游客', '2': '注册用户', '3': 'VIP用户' },
}

export default function FeedbackList() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<Feedback[]>([])
  const [total, setTotal] = useState(0)
  const [query, setQuery] = useState<FeedbackQuery>({ page: 1, page_size: 20 })
  const [selected, setSelected] = useState<Feedback | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [catLoading, setCatLoading] = useState(false)
  const [catForm] = Form.useForm()
  const [searchForm] = Form.useForm()
  const [enums, setEnums] = useState<DynamicEnums>(FALLBACK_ENUMS)
  const [clfCategories, setClfCategories] = useState<string[]>(CATEGORIES)
  const [clfModules, setClfModules] = useState<string[]>(BUSINESS_MODULES)
  const [clfLogs, setClfLogs] = useState<ClassificationLog[]>([])

  const loadEnums = useCallback(async () => {
    try {
      const res = await aggregatedEnums()
      if (res && Object.keys(res).length > 0) {
        setEnums(res)
      }
    } catch {
      // keep fallback enums
    }
  }, [])

  const loadClfEnums = useCallback(async () => {
    try {
      const res = await getClassifierEnums()
      if (res.categories?.length) setClfCategories(res.categories)
      if (res.business_modules?.length) setClfModules(res.business_modules)
    } catch {
      // keep fallback
    }
  }, [])

  useEffect(() => {
    loadEnums()
    loadClfEnums()
  }, [loadEnums, loadClfEnums])

  const fetchData = useCallback(async (q: FeedbackQuery) => {
    setLoading(true)
    try {
      const res = await listFeedbacks(q)
      setData(res.list ?? [])
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData(query)
  }, [query, fetchData])

  const openDetail = async (id: number) => {
    const f = await getFeedback(id)
    setSelected(f)
    catForm.setFieldsValue({ category: f.category, business_module: f.business_module })
    try {
      const logs = await listClassificationLogs(id)
      setClfLogs(logs ?? [])
    } catch {
      setClfLogs([])
    }
    setDrawerOpen(true)
  }

  const handleUpdateCategory = async (values: { category: string; business_module?: string }) => {
    if (!selected) return
    setCatLoading(true)
    try {
      await updateCategory(selected.id, values.category, values.business_module)
      message.success('分类已更新')
      setDrawerOpen(false)
      fetchData(query)
    } finally {
      setCatLoading(false)
    }
  }

  const enumLabel = (field: string, value: string | number | undefined | null): string => {
    if (value === undefined || value === null || value === '') return ''
    const map = enums[field]
    if (!map) return ''
    return map[String(value)] ?? ''
  }

  const resolveField = (
    enumKey: string, rawValue: string | number | undefined | null, storedName: string | undefined,
  ): string => {
    const label = enumLabel(enumKey, rawValue)
    if (label) return label
    if (storedName) return storedName
    if (rawValue != null && rawValue !== '') return String(rawValue)
    return '-'
  }

  const handleSearch = (values: Record<string, unknown>) => {
    const range = values.range as [dayjs.Dayjs, dayjs.Dayjs] | undefined
    setQuery({
      page: 1,
      page_size: 20,
      keyword: values.keyword as string,
      app_id: values.app_id as string,
      platform_id: (values.platform as string[])?.join(','),
      app_version: values.app_version as string,
      user_mode: (values.user_mode as number[])?.join(','),
      user_id: values.user_id as string,
      category: (values.category as string[])?.join(',') ?? values.category as string,
      business_module: (values.business_module as string[])?.join(','),
      sentiment: values.sentiment as string,
      start_time: range?.[0]?.toISOString(),
      end_time: range?.[1]?.toISOString(),
    })
  }

  const columns: ColumnsType<Feedback> = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: '内容',
      dataIndex: 'content',
      width: 200,
      render: (t) => (
        <Typography.Text ellipsis={{ tooltip: t }} style={{ maxWidth: 200 }}>
          {t}
        </Typography.Text>
      ),
    },
    {
      title: '用户',
      dataIndex: 'user_id',
      width: 80,
      render: (v, row) => v || row.user_name || '-',
    },
    {
      title: 'App',
      width: 80,
      render: (_, row) => resolveField('app_id', row.app_id, row.app_name),
    },
    {
      title: '平台',
      width: 60,
      render: (_, row) => resolveField('platform_id', row.platform_id, row.platform),
    },
    { title: '版本号', dataIndex: 'app_version', width: 80, render: (v) => v || '-' },
    {
      title: '情感',
      dataIndex: 'sentiment',
      width: 70,
      render: (v) => {
        const s = SENTIMENTS.find((x) => x.value === v)
        return s ? <Tag color={s.color}>{s.label}</Tag> : <Tag>{v || '-'}</Tag>
      },
    },
    { title: '分类', dataIndex: 'category', width: 100, render: (v) => v || '-' },
    { title: '业务模块', dataIndex: 'business_module', width: 100, render: (v) => (v ? v.replace(/\(.*?\)/, '') : '-') },
    {
      title: '分类状态',
      dataIndex: 'category_status',
      width: 80,
      render: (v) => {
        const s = CATEGORY_STATUS_MAP[v] ?? { label: String(v), color: 'default' }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    {
      title: '反馈时间',
      dataIndex: 'original_created_at',
      width: 120,
      render: (v) => v ? dayjs(v).format('MM-DD HH:mm') : '-',
    },
    {
      title: '操作',
      width: 60,
      render: (_, row) => (
        <Button type="link" size="small" onClick={() => openDetail(row.id)}>
          详情
        </Button>
      ),
    },
  ]

  const [quickRange, setQuickRange] = useState<string | undefined>()

  const QUICK_RANGES = [
    { value: '1h',  label: '最近 1 小时',  minutes: 60 },
    { value: '3h',  label: '最近 3 小时',  minutes: 180 },
    { value: '6h',  label: '最近 6 小时',  minutes: 360 },
    { value: '12h', label: '最近 12 小时', minutes: 720 },
    { value: '24h', label: '最近 24 小时', minutes: 1440 },
    { value: '3d',  label: '最近 3 天',    minutes: 4320 },
    { value: '7d',  label: '最近 7 天',    minutes: 10080 },
    { value: '30d', label: '最近 30 天',   minutes: 43200 },
  ]

  const handleQuickRangeChange = (value: string) => {
    setQuickRange(value)
    const opt = QUICK_RANGES.find((r) => r.value === value)
    if (!opt) return
    const now = dayjs()
    searchForm.setFieldsValue({ range: [now.subtract(opt.minutes, 'minute'), now] })
  }

  const handleRangeChange = (_dates: null | (dayjs.Dayjs | null)[], _formatStr: [string, string]) => {
    // 用户手动修改 RangePicker 时清除快捷选择标记
    if (_dates && _dates[0] && _dates[1]) {
      setQuickRange(undefined)
    }
  }

  const enumOptions = (field: string): { value: string; label: string }[] => {
    const map = enums[field]
    if (!map) return []
    return Object.entries(map).map(([k, v]) => ({ value: k, label: v }))
  }

  return (
    <Card>
      <Form layout="inline" onFinish={handleSearch} form={searchForm} style={{ marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <Form.Item name="keyword">
          <Input prefix={<SearchOutlined />} placeholder="关键词搜索" allowClear style={{ width: 200 }} />
        </Form.Item>
        <Form.Item name="app_id">
          <Select placeholder="应用" allowClear style={{ width: 120 }}>
            {enumOptions('app_id').map((o) => (
              <Select.Option key={o.value} value={o.value}>{o.label}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="platform">
          <Select placeholder="平台" allowClear mode="multiple" style={{ width: 160 }}>
            {enumOptions('platform_id').map((o) => (
              <Select.Option key={o.value} value={o.value}>{o.label}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="app_version">
          <Input placeholder="版本号" allowClear style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="user_id">
          <Input placeholder="用户ID" allowClear style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="user_mode">
          <Select placeholder="用户模式" allowClear style={{ width: 120 }}>
            {enumOptions('user_mode').map((o) => (
              <Select.Option key={o.value} value={o.value}>{o.label}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="category">
          <Select placeholder="分类" allowClear style={{ width: 130 }}>
            {clfCategories.map((c) => (
              <Select.Option key={c}>{c}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="business_module">
          <Select placeholder="业务模块" allowClear mode="multiple" style={{ width: 160 }}>
            {clfModules.map((m) => (
              <Select.Option key={m}>{m}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="sentiment">
          <Select placeholder="情感倾向" allowClear style={{ width: 120 }}>
            {SENTIMENTS.map((s) => (
              <Select.Option key={s.value}>{s.label}</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item>
          <Select
            placeholder="快捷时间"
            allowClear
            style={{ width: 130 }}
            value={quickRange}
            suffixIcon={<ClockCircleOutlined />}
            onChange={(v) => {
              if (!v) { setQuickRange(undefined); searchForm.setFieldsValue({ range: undefined }) }
              else handleQuickRangeChange(v)
            }}
            options={QUICK_RANGES.map((r) => ({ value: r.value, label: r.label }))}
          />
        </Form.Item>
        <Form.Item name="range">
          <RangePicker showTime onChange={handleRangeChange} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>搜索</Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                searchForm.resetFields()
                setQuickRange(undefined)
                setQuery({ page: 1, page_size: 20 })
              }}
            >
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={data}
        className="feedback-table"
        scroll={{ x: 'max-content' }}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
      />

      <Drawer
        title="反馈详情"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={820}
        destroyOnHidden
        styles={{ body: { overflowY: 'auto', maxHeight: 'calc(100vh - 55px)' } }}
        extra={
          <Button type="primary" loading={catLoading} onClick={() => catForm.submit()}>
            保存分类
          </Button>
        }
      >
        {selected && (
          <>
            <Descriptions
              column={2}
              bordered
              size="small"
              style={{ marginBottom: 24 }}
              styles={{ label: { width: 90, whiteSpace: 'nowrap' }, content: { maxWidth: 280, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }}
            >
              <Descriptions.Item label="ID">{selected.id}</Descriptions.Item>
              <Descriptions.Item label="App">{resolveField('app_id', selected.app_id, selected.app_name)}</Descriptions.Item>
              <Descriptions.Item label="平台">{resolveField('platform_id', selected.platform_id, selected.platform)}</Descriptions.Item>
              <Descriptions.Item label="版本">{selected.app_version || '-'}</Descriptions.Item>
              <Descriptions.Item label="用户模式">{resolveField('user_mode', selected.user_mode, undefined)}</Descriptions.Item>
              <Descriptions.Item label="渠道">{resolveField('channel_id', selected.channel_id, undefined)}</Descriptions.Item>
              <Descriptions.Item label="用户">{selected.user_id || selected.user_name || '-'}</Descriptions.Item>
              <Descriptions.Item label="情感">
                {SENTIMENTS.find((s) => s.value === selected.sentiment)?.label ?? selected.sentiment}
              </Descriptions.Item>
              <Descriptions.Item label="分类">{selected.category || '-'}</Descriptions.Item>
              <Descriptions.Item label="业务模块">{selected.business_module ? selected.business_module.replace(/\(.*?\)/, '') : '-'}</Descriptions.Item>
              <Descriptions.Item label="分类状态">
                {CATEGORY_STATUS_MAP[selected.category_status]?.label ?? selected.category_status}
              </Descriptions.Item>
              <Descriptions.Item label="反馈时间">
                {dayjs(selected.original_created_at).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="入库时间" span={2}>
                {dayjs(selected.created_at).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="内容" span={2} styles={{ content: { maxWidth: 'unset', whiteSpace: 'normal', overflow: 'hidden' } }}>
                <Typography.Paragraph ellipsis={{ rows: 3, expandable: 'collapsible', symbol: (collapsed: boolean) => collapsed ? '展开' : '收起' }} style={{ margin: 0 }}>{selected.content}</Typography.Paragraph>
              </Descriptions.Item>
              {selected.images && selected.images.length > 0 && (
                <Descriptions.Item label="图片" span={2} styles={{ content: { maxWidth: 'unset', whiteSpace: 'normal' } }}>
                  <Image.PreviewGroup>
                    <Space direction="vertical">
                      {selected.images.map((url, i) => <Image key={i} src={url} width={120} />)}
                    </Space>
                  </Image.PreviewGroup>
                </Descriptions.Item>
              )}
              {selected.videos && selected.videos.length > 0 && (
                <Descriptions.Item label="视频" span={2} styles={{ content: { maxWidth: 'unset', whiteSpace: 'normal' } }}>
                  <Space direction="vertical">
                    {selected.videos.map((url, i) => <video key={i} src={url} controls width={240}></video>)}
                  </Space>
                </Descriptions.Item>
              )}
            </Descriptions>
            <Form form={catForm} onFinish={handleUpdateCategory}>
              <Space>
                <Form.Item label="修改分类" name="category" rules={[{ required: true }]}>
                  <Select style={{ width: 160 }}>
                    {clfCategories.map((c) => (
                      <Select.Option key={c}>{c}</Select.Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item label="业务模块" name="business_module">
                  <Select style={{ width: 160 }} allowClear>
                    {clfModules.map((m) => (
                      <Select.Option key={m}>{m}</Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Space>
            </Form>

            {clfLogs.length > 0 && (
              <Collapse
                style={{ marginTop: 16 }}
                items={clfLogs.map((log) => ({
                  key: log.id,
                  label: (
                    <Space>
                      <Tag color={LOG_STATUS_MAP[log.status]?.color ?? 'default'}>
                        #{log.attempt} {LOG_STATUS_MAP[log.status]?.label ?? '未知'}
                      </Tag>
                      {log.provider_name && <Tag color="blue">{log.provider_name}</Tag>}
                      <Typography.Text type="secondary">
                        {dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss')}
                      </Typography.Text>
                      {log.duration_ms != null && (
                        <Typography.Text type="secondary">{log.duration_ms}ms</Typography.Text>
                      )}
                    </Space>
                  ),
                  children: (
                    <Descriptions column={1} bordered size="small" styles={{ label: { width: 120, whiteSpace: 'nowrap' }, content: { maxWidth: 'unset', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }}>
                      {log.category_result && (
                        <Descriptions.Item label="分类结果">{log.category_result}</Descriptions.Item>
                      )}
                      {log.module_result && (
                        <Descriptions.Item label="业务模块">{log.module_result.replace(/\(.*?\)/, '')}</Descriptions.Item>
                      )}
                      {log.sentiment_result && (
                        <Descriptions.Item label="情感倾向">{log.sentiment_result}</Descriptions.Item>
                      )}
                      {log.confidence_result != null && (
                        <Descriptions.Item label="置信度">{log.confidence_result}</Descriptions.Item>
                      )}
                      {log.error_message && (
                        <Descriptions.Item label="错误信息" styles={{ content: { maxWidth: 'unset', whiteSpace: 'normal' } }}>
                          <Typography.Text type="danger">{log.error_message}</Typography.Text>
                        </Descriptions.Item>
                      )}
                      {log.llm_raw_response && (
                        <Descriptions.Item label="LLM 原始输出" styles={{ content: { maxWidth: 'unset', whiteSpace: 'normal' } }}>
                          <Typography.Paragraph ellipsis={{ rows: 3, expandable: 'collapsible' }} style={{ margin: 0 }}>
                            {log.llm_raw_response}
                          </Typography.Paragraph>
                        </Descriptions.Item>
                      )}
                    </Descriptions>
                  ),
                }))}
              />
            )}
          </>
        )}
      </Drawer>
    </Card>
  )
}