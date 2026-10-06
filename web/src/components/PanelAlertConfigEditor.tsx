import { useState, useEffect } from 'react'
import { Switch, InputNumber, Select, Tooltip } from 'antd'
import { QuestionCircleOutlined } from '@ant-design/icons'
import type { PanelAlertConfig } from '../api/dashboard'
import type { NotificationChannel } from '../api/notification-channel'

const ALERTABLE_TYPES = new Set(['line', 'bar'])

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
  enabled: '仅折线图和柱状图支持告警配置',
  threshold: '触发告警的数值阈值',
  condition_op: '比较操作符，将查询结果值与阈值对比',
  level: '告警严重程度',
  silence_minutes: '告警触发后的静默期，避免短时间内重复通知',
  eval_interval_sec: '每隔多少秒重新评估一次规则',
  pending_duration_sec: '指标突破阈值后需持续高于阈值的时间（秒）。0=立即告警，>0 则先进入预警期，超过此时间仍高于阈值才正式告警',
  time_window_sec: '评估窗口时长（秒），决定查询 metric_time_buckets 的时间范围。窗口越长，搜索的 1m 桶越多',
  reduce_mode: '触发值基于时间网格桶的反馈数量，确保告警线与图表数据点精确对齐。峰值=窗口内最大桶值，最新=最近桶值。粒度（1m/1h）跟随面板时间维度配置',
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

interface PanelAlertConfigEditorProps {
  chartType: string
  value?: PanelAlertConfig | null
  onChange?: (v: PanelAlertConfig | null) => void
  channels: NotificationChannel[]
}

export default function PanelAlertConfigEditor({ chartType, value, onChange, channels }: PanelAlertConfigEditorProps) {
  const [enabled, setEnabled] = useState(value != null && ALERTABLE_TYPES.has(chartType))

  useEffect(() => {
    if (!ALERTABLE_TYPES.has(chartType) && enabled) {
      setEnabled(false)
      onChange?.(null)
    }
  }, [chartType])

  if (!ALERTABLE_TYPES.has(chartType)) {
    return <div style={{ color: '#999', fontSize: 13, marginTop: 8 }}>饼图和数值面板不支持告警配置</div>
  }

  const toggle = (checked: boolean) => {
    setEnabled(checked)
    if (!checked) {
      onChange?.(null)
    } else if (!value) {
      onChange?.({ threshold: 0, condition_op: '>', level: 'warning', silence_minutes: 30, eval_interval_sec: 60, pending_duration_sec: 0, time_window_sec: 300, reduce_mode: 'max', channel_ids: [] })
    }
  }

  const update = (patch: Partial<PanelAlertConfig>) => {
    if (!value) return
    onChange?.({ ...value, ...patch })
  }

  // 两列网格字段：每个字段 label + input 水平排列
  const fieldItemStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  }

  return (
    <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, padding: 16, marginTop: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: enabled ? 16 : 0 }}>
        <HintLabel field="enabled" text="启用告警" />
        <Switch checked={enabled} onChange={toggle} />
      </div>

      {enabled && value && (
        <>
          {/* 两列网格：条件 + 级别 | 阈值 + 评估间隔 */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '12px 16px',
            marginBottom: 12,
          }}>
            <div style={fieldItemStyle}>
              <HintLabel field="condition_op" text="条件" />
              <Select style={{ width: 80 }} options={CONDITION_OPTIONS} value={value.condition_op} onChange={(v) => update({ condition_op: v })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="level" text="级别" />
              <Select style={{ width: 100 }} options={LEVEL_OPTIONS} value={value.level} onChange={(v) => update({ level: v })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="threshold" text="阈值" />
              <InputNumber style={{ width: 120 }} value={value.threshold} onChange={(v) => update({ threshold: v ?? 0 })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="eval_interval_sec" text="评估间隔(s)" />
              <InputNumber style={{ width: 120 }} min={10} value={value.eval_interval_sec} onChange={(v) => update({ eval_interval_sec: v ?? 60 })} />
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
              <InputNumber style={{ width: 120 }} min={0} value={value.pending_duration_sec ?? 0} onChange={(v) => update({ pending_duration_sec: v ?? 0 })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="time_window_sec" text="评估窗口(s)" />
              <InputNumber style={{ width: 120 }} min={30} value={value.time_window_sec ?? 300} onChange={(v) => update({ time_window_sec: v ?? 300 })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="reduce_mode" text="缩减模式" />
              <Select style={{ width: 160 }} options={REDUCE_MODE_OPTIONS} value={value.reduce_mode ?? 'max'} onChange={(v) => update({ reduce_mode: v })} />
            </div>
            <div style={fieldItemStyle}>
              <HintLabel field="silence_minutes" text="静默时间(min)" />
              <InputNumber style={{ width: 120 }} min={0} value={value.silence_minutes} onChange={(v) => update({ silence_minutes: v ?? 30 })} />
            </div>
          </div>

          {/* 通知渠道：独占一行 */}
          <div style={fieldItemStyle}>
            <HintLabel field="channel_ids" text="通知渠道" />
            <Select
              mode="multiple"
              style={{ flex: 1, minWidth: 200 }}
              placeholder="选择通知渠道"
              options={channels.map((c) => ({ value: c.id, label: `${c.name} (${c.type})` }))}
              value={value.channel_ids}
              onChange={(v) => update({ channel_ids: v })}
            />
          </div>
        </>
      )}
    </div>
  )
}