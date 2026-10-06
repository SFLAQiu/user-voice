import dayjs from 'dayjs'
import type { AlertRuleStateBrief } from '../api/dashboard'
import type { AlertRecord } from '../api/alert'

// ── 阈值标记标签样式 ──
const THRESHOLD_LABEL_STYLE = {
  show: true as const,
  position: 'insideEndTop' as const,
  color: '#faad14',
  fontSize: 12,
  fontWeight: 'bold' as const,
}

// ── 封顶阈值标签样式（超出图表范围时） ──
const CAPPED_THRESHOLD_LABEL_STYLE = {
  show: true as const,
  position: 'insideEndTop' as const,
  color: '#faad14',
  fontSize: 11,
  fontWeight: 'bold' as const,
}

// ── 告警标注颜色 — 三色区分 ──
const COLOR_THRESHOLD = '#faad14'   // 黄色 — 阈值水平线（预警色，表示"将触发告警的阈值"）
const COLOR_PENDING   = '#faad14'   // 黄色 — 预警线（Pending 状态）
const COLOR_FIRING    = '#ff4d4f'   // 红色 — 告警线（Firing 状态）
const COLOR_RESOLVED  = '#52c41a'   // 绿色 — 恢复正常线（Resolved 状态）

// ── 粒度分级 ──

type GranularityLevel = 'fine' | 'medium' | 'coarse'

function classifyGranularity(bucket: string): GranularityLevel {
  switch (bucket) {
    case '1s': case '1m': return 'fine'
    case '1h': return 'medium'
    case '1d': case '1w': return 'coarse'
    default: return 'medium'
  }
}

// ── 标注数据结构 ──

interface MarkLineItem {
  yAxis?: number
  xAxis?: number
  lineStyle: { type: string; color: string; width?: number }
  name: string
  label: { show: boolean; position?: string; formatter?: string; color?: string; fontSize?: number; fontWeight?: string }
}

// 告警事件竖线 — 增加 triggerValue/threshold/level 用于 tooltip 展示
interface AlertEventLine {
  ts: number
  color: string
  name: string
  triggerValue: number
  threshold: number
  level: string
}

interface MarkAreaRangeItem {
  0: { xAxis?: number; yAxis?: number; name: string; itemStyle: { color: string; opacity: number } }
  1: { xAxis?: number; yAxis?: number }
}

export interface AlertAnnotationResult {
  markLine?: { data: MarkLineItem[] }
  markArea?: { data: MarkAreaRangeItem[] }
}

/** 判断告警记录是否为恢复正常事件（兼容 state="ok" 和 state="resolved"） */
function isResolvedState(state: string): boolean {
  return state === 'resolved' || state === 'ok'
}

/** 获取告警事件定位时间戳 — 优先 metric_timestamp（数据桶时间），回退 triggered_at（评估时刻） */
function eventTimestamp(record: AlertRecord): string {
  return record.metric_timestamp ?? record.triggered_at
}

/** 获取恢复时间点：优先 resolved_at，回退到 triggered_at（state="ok" 无 resolved_at） */
function resolveTimestamp(record: AlertRecord): number {
  return record.resolved_at
    ? dayjs(record.resolved_at).valueOf()
    : dayjs(record.triggered_at).valueOf()
}

// 告警区间 — 增加 triggerValue/threshold/level 用于 tooltip 展示
interface AlertPeriod {
  startTs: number       // triggered_at 毫秒
  endTs: number | null  // resolved_at 毫秒，null 表示仍在进行中
  state: 'pending' | 'firing'
  triggerValue: number  // 触发时的实际值
  threshold: number     // 告警阈值
  level: string         // 告警级别
}

/**
 * 将告警记录按时间排序，配对 firing/resolved 形成告警区间。
 * 未配对的 firing 记录视为仍在进行中（endTs = null）。
 * pending 记录作为独立的预警区间。
 */
function buildAlertPeriods(records: AlertRecord[]): AlertPeriod[] {
  // 按时间升序排序
  const sorted = [...records].sort((a, b) =>
    dayjs(eventTimestamp(a)).unix() - dayjs(eventTimestamp(b)).unix()
  )

  const periods: AlertPeriod[] = []
  const pendingFirings: AlertRecord[] = []

  for (const record of sorted) {
    if (record.state === 'pending') {
      periods.push({
        startTs: dayjs(eventTimestamp(record)).valueOf(),
        endTs: null,
        state: 'pending',
        triggerValue: record.trigger_value,
        threshold: record.threshold,
        level: record.level,
      })
    } else if (record.state === 'firing') {
      pendingFirings.push(record)
    } else if (isResolvedState(record.state)) {
      // 尝试与最早的未配对 firing 匹配
      const matched = pendingFirings.shift()
      if (matched) {
        periods.push({
          startTs: dayjs(eventTimestamp(matched)).valueOf(),
          endTs: resolveTimestamp(record),
          state: 'firing',
          triggerValue: matched.trigger_value,
          threshold: matched.threshold,
          level: matched.level,
        })
      }
    }
  }

  // 未配对的 firing 记录视为仍在进行中
  for (const f of pendingFirings) {
    periods.push({
      startTs: dayjs(eventTimestamp(f)).valueOf(),
      endTs: null,
      state: 'firing',
      triggerValue: f.trigger_value,
      threshold: f.threshold,
      level: f.level,
    })
  }

  return periods
}

// ── 细粒度/中粒度渲染：markLine 竖线 ──

function buildMarkLineAnnotations(
  alertState: AlertRuleStateBrief,
  records: AlertRecord[],
  bucket: string,
  yAxisMax?: number,
): MarkLineItem[] {
  const result: MarkLineItem[] = []

  // 1. 阈值水平线
  if (alertState?.threshold != null) {
    const opLabel = alertState.condition_op ?? '>'
    const threshold = alertState.threshold
    // 判断是否需要封顶绘制：阈值超出 Y 轴范围
    const needsCap = yAxisMax != null && threshold > yAxisMax
    if (needsCap) {
      // Case B：阈值远超数据范围 — 封顶绘制在图表顶部
      const cappedY = yAxisMax * 0.85
      result.push({
        yAxis: cappedY,
        lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
        name: `threshold_capped|${opLabel}|${threshold}|${cappedY}|${alertState.level}`,
        label: { ...CAPPED_THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
      })
    } else {
      // Case A：阈值在合理范围内 — 在实际 Y 坐标位置绘制
      result.push({
        yAxis: threshold,
        lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
        name: `threshold|${opLabel}|${threshold}|${alertState.level}`,
        label: { ...THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
      })
    }
  }

  // 2. 告警记录竖线 — 三色区分
  // name 格式：event|label|ts|triggerValue|threshold|level
  const eventLines: AlertEventLine[] = []
  for (const record of records) {
    if (record.state === 'pending') {
      const ts = dayjs(eventTimestamp(record)).valueOf()
      eventLines.push({
        ts, color: COLOR_PENDING,
        name: `event|预警触发|${ts}|${record.trigger_value}|${record.threshold}|${record.level}`,
        triggerValue: record.trigger_value,
        threshold: record.threshold,
        level: record.level,
      })
    } else if (record.state === 'firing') {
      const ts = dayjs(eventTimestamp(record)).valueOf()
      eventLines.push({
        ts, color: COLOR_FIRING,
        name: `event|告警触发|${ts}|${record.trigger_value}|${record.threshold}|${record.level}`,
        triggerValue: record.trigger_value,
        threshold: record.threshold,
        level: record.level,
      })
    } else if (isResolvedState(record.state)) {
      const ts = resolveTimestamp(record)
      eventLines.push({
        ts, color: COLOR_RESOLVED,
        name: `event|恢复正常|${ts}|0|0|`,
        triggerValue: 0,
        threshold: 0,
        level: '',
      })
    }
  }

  // 根据时间粒度动态抽稀告警线，避免粗粒度下密集重叠
  const thinnedLines = thinAlertEventLines(eventLines, bucket)
  result.push(
    ...thinnedLines.map((line) => ({
      xAxis: line.ts,
      lineStyle: { type: 'dashed', color: line.color, width: 1 },
      name: line.name,
      label: { show: false as const },
    })),
  )

  return result
}

function getBucketWindowMs(bucket: string): number {
  switch (bucket) {
    case '1s':
      return 1000
    case '1m':
      return 60 * 1000
    case '1h':
      return 60 * 60 * 1000
    case '1d':
      return 24 * 60 * 60 * 1000
    case '1w':
      return 7 * 24 * 60 * 60 * 1000
    default:
      return 60 * 60 * 1000
  }
}

function getMaxAlertLineCount(bucket: string): number {
  switch (bucket) {
    case '1s':
      return 36
    case '1m':
      return 28
    case '1h':
      return 16
    case '1d':
      return 10
    case '1w':
      return 6
    default:
      return 16
  }
}

function thinAlertEventLines(lines: AlertEventLine[], bucket: string): AlertEventLine[] {
  if (lines.length <= 1) return lines
  const sorted = [...lines].sort((a, b) => a.ts - b.ts)
  const maxCount = getMaxAlertLineCount(bucket)
  const minGapMs = Math.max(getBucketWindowMs(bucket) / 4, 1) // 放宽最小间隔，允许更多恢复事件

  // 过滤时保留 resolved 事件（绿色线），只在同类型事件间去重
  const deduped: AlertEventLine[] = []
  for (const line of sorted) {
    const last = deduped[deduped.length - 1]
    // 恢复事件总是保留；其他类型事件需要满足最小时间间隔
    if (!last || line.color === COLOR_RESOLVED || line.ts - last.ts >= minGapMs) {
      deduped.push(line)
    }
  }

  if (deduped.length <= maxCount) {
    return deduped
  }

  // 采样策略：优先保留 firing 和 resolved 事件
  const resolved = deduped.filter((l) => l.color === COLOR_RESOLVED)
  const others = deduped.filter((l) => l.color !== COLOR_RESOLVED)
  const step = Math.max(Math.ceil(others.length / (maxCount - resolved.length)), 1)
  const sampledOthers: AlertEventLine[] = []
  for (let i = 0; i < others.length; i += step) {
    sampledOthers.push(others[i])
  }

  const combined = [...sampledOthers, ...resolved].sort((a, b) => a.ts - b.ts)
  return combined.slice(0, maxCount)
}

// ── 粗粒度渲染：markArea 色块区间 ──

function buildMarkAreaAnnotations(
  alertState: AlertRuleStateBrief,
  records: AlertRecord[],
  chartEndTime: number,
  yAxisMax?: number,
): { markLine: MarkLineItem[]; markArea: MarkAreaRangeItem[] } {
  const thresholdLines: MarkLineItem[] = []

  // 阈值水平线
  if (alertState?.threshold != null) {
    const opLabel = alertState.condition_op ?? '>'
    const threshold = alertState.threshold
    const needsCap = yAxisMax != null && threshold > yAxisMax
    if (needsCap) {
      const cappedY = yAxisMax * 0.85
      thresholdLines.push({
        yAxis: cappedY,
        lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
        name: `threshold_capped|${opLabel}|${threshold}|${cappedY}|${alertState.level}`,
        label: { ...CAPPED_THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
      })
    } else {
      thresholdLines.push({
        yAxis: threshold,
        lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
        name: `threshold|${opLabel}|${threshold}|${alertState.level}`,
        label: { ...THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
      })
    }
  }

  // 配对告警区间为色块
  // name 格式：period|label|startTs|endTs|level
  const periods = buildAlertPeriods(records)
  const areaItems: MarkAreaRangeItem[] = []

  for (const period of periods) {
    const endTs = period.endTs ?? chartEndTime
    const color = period.state === 'pending' ? COLOR_PENDING : COLOR_FIRING
    const opacity = period.state === 'pending' ? 0.12 : 0.15
    const label = period.state === 'pending' ? '预警期' : '告警期'

    areaItems.push({
      0: {
        xAxis: period.startTs,
        name: `period|${label}|${period.startTs}|${endTs}|${period.level}`,
        itemStyle: { color, opacity },
      },
      1: { xAxis: endTs },
    })
  }

  return { markLine: thresholdLines, markArea: areaItems }
}

// ── 主入口 ──

/**
 * 构建 ECharts 告警标注数据，根据时间粒度自动选择渲染模式。
 * - fine/medium → markLine 竖线（精确定位每个告警事件）
 * - coarse → markArea 半透明色块区间（展示告警持续范围，避免竖线重叠）
 *
 * name 字段编码格式（供 tooltip formatter 解析）：
 * - threshold|op|value|level  — 阈值水平线
 * - event|label|ts|triggerValue|threshold|level  — 告警竖线
 * - period|label|startTs|endTs|level  — 告警色块区间
 */
export function buildAlertAnnotations(
  alertState: AlertRuleStateBrief | null | undefined,
  records: AlertRecord[],
  bucket: string,
  chartEndTime?: number,
  yAxisMax?: number,
): AlertAnnotationResult {
  if (!alertState || !records || records.length === 0 && alertState.threshold == null) {
    // 仅阈值线无记录的情况
    if (alertState?.threshold != null) {
      const opLabel = alertState.condition_op ?? '>'
      const threshold = alertState.threshold
      const needsCap = yAxisMax != null && threshold > yAxisMax
      if (needsCap) {
        const cappedY = yAxisMax * 0.85
        const markLineItems: MarkLineItem[] = [{
          yAxis: cappedY,
          lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
          name: `threshold_capped|${opLabel}|${threshold}|${cappedY}|${alertState.level}`,
          label: { ...CAPPED_THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
        }]
        // 封顶时添加告警区
        const markAreaItems: MarkAreaRangeItem[] = [{
          0: {
            xAxis: 0,
            name: `threshold_zone|${threshold}|${alertState.level}`,
            itemStyle: { color: COLOR_THRESHOLD, opacity: 0.08 },
            yAxis: cappedY,
          },
          1: { yAxis: yAxisMax },
        }]
        return {
          markLine: { data: markLineItems },
          markArea: { data: markAreaItems },
        }
      }
      return {
        markLine: {
          data: [{
            yAxis: threshold,
            lineStyle: { type: 'dashed', color: COLOR_THRESHOLD, width: 2 },
            name: `threshold|${opLabel}|${threshold}|${alertState.level}`,
            label: { ...THRESHOLD_LABEL_STYLE, formatter: `阈值 ${opLabel} ${threshold}` },
          }],
        },
      }
    }
    return {}
  }

  const granularity = classifyGranularity(bucket)

  if (granularity === 'coarse') {
    // 粗粒度：markArea + markLine（阈值线）
    const endTime = chartEndTime ?? dayjs().valueOf()
    const { markLine, markArea } = buildMarkAreaAnnotations(alertState, records, endTime, yAxisMax)
    // 封顶阈值时追加告警区
    const needsCap = yAxisMax != null && alertState.threshold > yAxisMax
    let combinedMarkArea = markArea
    if (needsCap) {
      const cappedY = yAxisMax * 0.85
      const zoneItem: MarkAreaRangeItem = {
        0: {
          xAxis: 0,
          name: `threshold_zone|${alertState.threshold}|${alertState.level}`,
          itemStyle: { color: COLOR_THRESHOLD, opacity: 0.08 },
          yAxis: cappedY,
        },
        1: { yAxis: yAxisMax },
      }
      combinedMarkArea = markArea.length > 0 ? [...markArea, zoneItem] : [zoneItem]
    }
    return {
      markLine: markLine.length > 0 ? { data: markLine } : undefined,
      markArea: combinedMarkArea.length > 0 ? { data: combinedMarkArea } : undefined,
    }
  }

  // 细粒度/中粒度：纯 markLine
  const lineItems = buildMarkLineAnnotations(alertState, records, bucket, yAxisMax)
  // 封顶阈值时追加告警区
  const needsCap = yAxisMax != null && alertState.threshold > yAxisMax
  if (needsCap) {
    const cappedY = yAxisMax * 0.85
    const zoneItem: MarkAreaRangeItem = {
      0: {
        xAxis: 0,
        name: `threshold_zone|${alertState.threshold}|${alertState.level}`,
        itemStyle: { color: COLOR_THRESHOLD, opacity: 0.08 },
        yAxis: cappedY,
      },
      1: { yAxis: yAxisMax },
    }
    return {
      markLine: lineItems.length > 0 ? { data: lineItems } : undefined,
      markArea: { data: [zoneItem] },
    }
  }
  return lineItems.length > 0 ? { markLine: { data: lineItems } } : {}
}

/**
 * 仅 line 和 bar 类型面板支持告警配置。
 */
export function isAlertableChartType(chartType: string): boolean {
  return chartType === 'line' || chartType === 'bar'
}