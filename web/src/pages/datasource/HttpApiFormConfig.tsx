import { useState, useCallback } from 'react'
import {
  Form,
  Input,
  Select,
  Switch,
  InputNumber,
  Button,
  Tooltip,
  Space,
  Collapse,
  message,
} from 'antd'
import { PlusOutlined, DeleteOutlined, QuestionCircleOutlined } from '@ant-design/icons'
import type { HttpApiConfig } from '../../api/datasource'

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

interface FieldMappingItem {
  systemField: string
  sourceField: string
}

interface KeyValueItem {
  key: string
  value: string
}

interface Props {
  config: HttpApiConfig
  onChange: (config: HttpApiConfig) => void
}

function buildInitialMappings(fieldMapping?: Record<string, string>): FieldMappingItem[] {
  if (!fieldMapping || Object.keys(fieldMapping).length === 0) {
    return [
      { systemField: 'original_id', sourceField: '' },
      { systemField: 'content', sourceField: '' },
    ]
  }
  return Object.entries(fieldMapping).map(([systemField, sourceField]) => ({
    systemField,
    sourceField,
  }))
}

function buildInitialKvList(obj?: Record<string, string>): KeyValueItem[] {
  if (!obj || Object.keys(obj).length === 0) return []
  return Object.entries(obj).map(([key, value]) => ({ key, value }))
}

function mappingsToObj(items: FieldMappingItem[]): Record<string, string> {
  const obj: Record<string, string> = {}
  for (const m of items) {
    if (m.systemField && m.sourceField) {
      obj[m.systemField] = m.sourceField
    }
  }
  return obj
}

function kvListToObj(items: KeyValueItem[]): Record<string, string> {
  const obj: Record<string, string> = {}
  for (const item of items) {
    if (item.key && item.value) {
      obj[item.key] = item.value
    }
  }
  return obj
}

function fieldLabel(label: string, tooltip: string) {
  return (
    <Space size={4}>
      {label}
      <Tooltip title={tooltip}>
        <QuestionCircleOutlined style={{ color: '#999', cursor: 'help' }} />
      </Tooltip>
    </Space>
  )
}

export default function HttpApiFormConfig({ config, onChange }: Props) {
  const [mappings, setMappings] = useState<FieldMappingItem[]>(() => buildInitialMappings(config.field_mapping))
  const [headers, setHeaders] = useState<KeyValueItem[]>(() => buildInitialKvList(config.headers))
  const [paramsTemplate, setParamsTemplate] = useState<KeyValueItem[]>(() => buildInitialKvList(config.params_template))

  const updateField = useCallback(
    <K extends keyof HttpApiConfig>(key: K, value: HttpApiConfig[K]) => {
      onChange({ ...config, [key]: value })
    },
    [config, onChange],
  )

  const updateMapping = useCallback(
    (index: number, field: 'systemField' | 'sourceField', value: string) => {
      const updated = mappings.map((m, i) => (i === index ? { ...m, [field]: value } : m))
      setMappings(updated)
      onChange({ ...config, field_mapping: mappingsToObj(updated) })
    },
    [config, onChange, mappings],
  )

  const addMapping = useCallback(() => {
    const next = [...mappings, { systemField: '', sourceField: '' }]
    setMappings(next)
    onChange({ ...config, field_mapping: mappingsToObj(next) })
  }, [config, onChange, mappings])

  const removeMapping = useCallback(
    (index: number) => {
      const ref = FIELD_REFERENCE.find((r) => r.field === mappings[index]?.systemField)
      if (ref?.required) {
        message.warning(`${ref.field} 为必填字段，不可删除`)
        return
      }
      const next = mappings.filter((_, i) => i !== index)
      setMappings(next)
      onChange({ ...config, field_mapping: mappingsToObj(next) })
    },
    [config, onChange, mappings],
  )

  const updateKvList = useCallback(
    (
      items: KeyValueItem[],
      setItems: (v: KeyValueItem[]) => void,
      index: number,
      field: 'key' | 'value',
      value: string,
      configKey: 'headers' | 'params_template',
    ) => {
      const updated = items.map((item, i) => (i === index ? { ...item, [field]: value } : item))
      setItems(updated)
      onChange({ ...config, [configKey]: kvListToObj(updated) })
    },
    [config, onChange],
  )

  const addKvItem = useCallback(
    (
      items: KeyValueItem[],
      setItems: (v: KeyValueItem[]) => void,
      configKey: 'headers' | 'params_template',
    ) => {
      const next = [...items, { key: '', value: '' }]
      setItems(next)
      onChange({ ...config, [configKey]: kvListToObj(next) })
    },
    [config, onChange],
  )

  const removeKvItem = useCallback(
    (
      items: KeyValueItem[],
      setItems: (v: KeyValueItem[]) => void,
      index: number,
      configKey: 'headers' | 'params_template',
    ) => {
      const next = items.filter((_, i) => i !== index)
      setItems(next)
      onChange({ ...config, [configKey]: kvListToObj(next) })
    },
    [config, onChange],
  )

  return (
    <Form layout="vertical" size="small">
      {/* 基础配置 */}
      <Form.Item label={fieldLabel('API 地址', '外部 API 地址（必填），用于拉取反馈数据的接口地址')}>
        <Input
          value={config.endpoint ?? ''}
          onChange={(e) => updateField('endpoint', e.target.value)}
          placeholder="https://api.example.com/feedback/list"
        />
      </Form.Item>

      <Form.Item label={fieldLabel('HTTP 方法', 'HTTP 请求方法，默认 GET')}>
        <Select
          value={config.method ?? 'GET'}
          onChange={(v) => updateField('method', v)}
          options={[
            { value: 'GET', label: 'GET' },
            { value: 'POST', label: 'POST' },
          ]}
          style={{ width: 120 }}
        />
      </Form.Item>

      <Form.Item label={fieldLabel('Cookie', 'Cookie 字符串，保存时自动加密存储，显示时掩码为 ***')}>
        <Input
          value={config.cookie ?? ''}
          onChange={(e) => updateField('cookie', e.target.value)}
          placeholder="可选，部分接口需要认证 Cookie"
        />
      </Form.Item>

      <Form.Item label={fieldLabel('数据路径', 'API 响应 JSON 中反馈列表的路径，如 data.feedback_question_list')}>
        <Input
          value={config.data_path ?? ''}
          onChange={(e) => updateField('data_path', e.target.value)}
          placeholder="data.list"
        />
      </Form.Item>

      {/* 分页配置 */}
      <Collapse
        size="small"
        items={[
          {
            key: 'paginate',
            label: '分页配置',
            children: (
              <Space direction="vertical" style={{ width: '100%' }}>
                <Form.Item label={fieldLabel('分页参数名', 'URL 中分页页码对应的查询参数名，如 page 或 to_page')} style={{ marginBottom: 8 }}>
                  <Input
                    value={config.page_paginate?.param ?? ''}
                    onChange={(e) =>
                      updateField('page_paginate', {
                        param: e.target.value,
                        start: config.page_paginate?.start ?? 1,
                      })
                    }
                    placeholder="page"
                  />
                </Form.Item>
                <Form.Item label={fieldLabel('起始页码', '分页起始页码，默认 1')} style={{ marginBottom: 0 }}>
                  <InputNumber
                    value={config.page_paginate?.start ?? 1}
                    onChange={(v) =>
                      updateField('page_paginate', {
                        param: config.page_paginate?.param ?? 'page',
                        start: v ?? 1,
                      })
                    }
                    min={0}
                    style={{ width: 120 }}
                  />
                </Form.Item>
              </Space>
            ),
          },
        ]}
      />

      {/* 时间与限制 */}
      <Space style={{ width: '100%', marginTop: 12 }} direction="vertical">
        <Form.Item label={fieldLabel('时间格式', '时间字段解析格式，默认 Go 格式 "2006-01-02 15:04:05"')}>
          <Input
            value={config.time_layout ?? ''}
            onChange={(e) => updateField('time_layout', e.target.value)}
            placeholder="2006-01-02 15:04:05"
          />
        </Form.Item>

        <Form.Item label={fieldLabel('最大页数', '单次同步最大拉取页数，防止无限翻页，默认 50')}>
          <InputNumber
            value={config.max_pages ?? 50}
            onChange={(v) => updateField('max_pages', v ?? 50)}
            min={1}
            max={1000}
            style={{ width: 120 }}
          />
        </Form.Item>

        <Form.Item label={fieldLabel('遇已存在停止', '翻页时遇到已导入的 original_id 即停止，默认关闭')} valuePropName="checked">
          <Switch
            checked={config.stop_when_seen ?? false}
            onChange={(v) => updateField('stop_when_seen', v)}
          />
        </Form.Item>
      </Space>

      {/* 字段映射 */}
      <div style={{ marginTop: 16 }}>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>
          <Space size={4}>
            字段映射
            <Tooltip title="将源 API 响应字段映射到系统内部字段。original_id 和 content 为必填映射">
              <QuestionCircleOutlined style={{ color: '#999', cursor: 'help' }} />
            </Tooltip>
          </Space>
        </div>

        {mappings.map((m, index) => {
          const ref = FIELD_REFERENCE.find((r) => r.field === m.systemField)
          return (
            <Space key={index} style={{ display: 'flex', marginBottom: 4 }} align="baseline">
              <Select
                value={m.systemField}
                onChange={(v) => updateMapping(index, 'systemField', v)}
                placeholder="系统字段"
                style={{ width: 160 }}
                options={FIELD_REFERENCE.map((r) => ({
                  value: r.field,
                  label: r.required ? `${r.field} *` : r.field,
                }))}
              />
              {ref && (
                <Tooltip title={ref.desc}>
                  <QuestionCircleOutlined style={{ color: '#999', cursor: 'help' }} />
                </Tooltip>
              )}
              <span>→</span>
              <Input
                value={m.sourceField}
                onChange={(e) => updateMapping(index, 'sourceField', e.target.value)}
                placeholder="源字段名"
                style={{ width: 160 }}
              />
              <Button
                icon={<DeleteOutlined />}
                size="small"
                danger
                onClick={() => removeMapping(index)}
                disabled={ref?.required}
              />
            </Space>
          )
        })}
        <Button icon={<PlusOutlined />} size="small" onClick={addMapping} style={{ marginTop: 4 }}>
          添加映射
        </Button>
      </div>

      {/* 高级配置 */}
      <Collapse
        size="small"
        style={{ marginTop: 16 }}
        items={[
          {
            key: 'headers',
            label: '请求头配置',
            children: (
              <>
                {headers.map((h, index) => (
                  <Space key={index} style={{ display: 'flex', marginBottom: 4 }} align="baseline">
                    <Input
                      value={h.key}
                      onChange={(e) => updateKvList(headers, setHeaders, index, 'key', e.target.value, 'headers')}
                      placeholder="Header 名称"
                      style={{ width: 150 }}
                    />
                    <Input
                      value={h.value}
                      onChange={(e) => updateKvList(headers, setHeaders, index, 'value', e.target.value, 'headers')}
                      placeholder="Header 值"
                      style={{ width: 200 }}
                    />
                    <Button icon={<DeleteOutlined />} size="small" danger onClick={() => removeKvItem(headers, setHeaders, index, 'headers')} />
                  </Space>
                ))}
                <Button icon={<PlusOutlined />} size="small" onClick={() => addKvItem(headers, setHeaders, 'headers')}>
                  添加请求头
                </Button>
              </>
            ),
          },
          {
            key: 'params',
            label: '查询参数模板',
            children: (
              <>
                {paramsTemplate.map((p, index) => (
                  <Space key={index} style={{ display: 'flex', marginBottom: 4 }} align="baseline">
                    <Input
                      value={p.key}
                      onChange={(e) => updateKvList(paramsTemplate, setParamsTemplate, index, 'key', e.target.value, 'params_template')}
                      placeholder="参数名"
                      style={{ width: 150 }}
                    />
                    <Input
                      value={p.value}
                      onChange={(e) => updateKvList(paramsTemplate, setParamsTemplate, index, 'value', e.target.value, 'params_template')}
                      placeholder="参数值（支持 {{变量}}）"
                      style={{ width: 200 }}
                    />
                    <Button icon={<DeleteOutlined />} size="small" danger onClick={() => removeKvItem(paramsTemplate, setParamsTemplate, index, 'params_template')} />
                  </Space>
                ))}
                <Button icon={<PlusOutlined />} size="small" onClick={() => addKvItem(paramsTemplate, setParamsTemplate, 'params_template')}>
                  添加参数
                </Button>
              </>
            ),
          },
          {
            key: 'iterate',
            label: '变量组（iterate）',
            children: (
              <div style={{ color: '#999', fontSize: 13 }}>
                变量组配置较复杂，请使用 JSON 配置模式进行编辑
              </div>
            ),
          },
        ]}
      />
    </Form>
  )
}