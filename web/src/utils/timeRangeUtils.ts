import dayjs from 'dayjs'
import isoWeek from 'dayjs/plugin/isoWeek'

dayjs.extend(isoWeek)

export interface TimeRangeOverride {
  type: 'relative' | 'absolute'
  value?: string    // '1d', '5m', etc.
  start?: string    // ISO8601 for absolute
  end?: string
}

// ── 时间范围持续时间计算 ──

const DURATION_UNITS: Record<string, number> = {
  s: 1000,
  m: 60 * 1000,
  h: 60 * 60 * 1000,
  d: 24 * 60 * 60 * 1000,
  w: 7 * 24 * 60 * 60 * 1000,
}

/** 计算相对时间范围的毫秒数 */
function parseRelativeMs(value: string): number {
  if (!value || value.length < 2) return 24 * 60 * 60 * 1000 // 默认1天
  const n = parseInt(value.slice(0, -1), 10)
  const unit = value.slice(-1)
  const ms = DURATION_UNITS[unit] ?? DURATION_UNITS.d
  return n * ms
}

/** 计算时间范围的总毫秒数 */
export function timeRangeDuration(tr: TimeRangeOverride): number {
  if (tr.type === 'relative' && tr.value) {
    return parseRelativeMs(tr.value)
  }
  if (tr.type === 'absolute' && tr.start && tr.end) {
    return dayjs(tr.end).diff(dayjs(tr.start))
  }
  return 24 * 60 * 60 * 1000 // 默认1天
}

// ── 动态分桶粒度 ──

/** 根据时间范围长度自动选择最优 bucket，与后端 AutoBucket 保持一致 */
export function autoBucket(tr: TimeRangeOverride | undefined): string {
  if (!tr) return '1d'
  const ms = timeRangeDuration(tr)

  if (ms <= 5 * 60 * 1000) return '1s'         // ≤5分钟 → 秒级
  if (ms <= 30 * 60 * 1000) return '1m'         // ≤30分钟 → 分钟级
  if (ms <= 6 * 60 * 60 * 1000) return '1m'     // ≤6小时 → 分钟级
  if (ms <= 24 * 60 * 60 * 1000) return '1h'    // ≤1天 → 小时级
  if (ms <= 7 * 24 * 60 * 60 * 1000) return '1h' // ≤7天 → 小时级
  if (ms <= 30 * 24 * 60 * 60 * 1000) return '1d' // ≤30天 → 天级
  return '1w'                                     // >30天 → 周级
}

// ── 时间格式映射 ──

const BUCKET_DAYJS_FORMAT: Record<string, string> = {
  '1s': 'YYYY-MM-DD HH:mm:ss',
  '1m': 'YYYY-MM-DD HH:mm',
  '1h': 'YYYY-MM-DD HH',
  '1d': 'YYYY-MM-DD',
  '1w': 'YYYY-[W]WW',
}

/** 从 bucket 推导 dayjs 格式字符串 */
export function bucketToDayjsFormat(bucket: string): string {
  return BUCKET_DAYJS_FORMAT[bucket] ?? 'YYYY-MM-DD'
}

// ── MySQL DATE_FORMAT 兼容的时间格式化 ──

/**
 * 将时间戳格式化为与后端 MySQL DATE_FORMAT 完全匹配的字符串。
 * ECharts category 轴数据来自 SQL DATE_FORMAT，告警 markLine xAxis 值必须精确匹配才能定位。
 *
 * MySQL DATE_FORMAT 对照：
 * - 1s: %Y-%m-%d %H:%i:%s → "2024-01-15 14:30:00"
 * - 1m: %Y-%m-%d %H:%i:00 → "2024-01-15 14:30:00"  (尾部 :00)
 * - 1h: %Y-%m-%d %H:00:00 → "2024-01-15 14:00:00"  (尾部 :00:00)
 * - 1d: %Y-%m-%d          → "2024-01-15"
 * - 1w: %Y-%u              → "2024-3" (ISO 周号)
 */
export function formatTimeForBucket(ts: string | Date, bucket: string): string {
  const d = dayjs(ts)
  switch (bucket) {
    case '1s':
      return d.format('YYYY-MM-DD HH:mm:ss')
    case '1m':
      // MySQL %Y-%m-%d %H:%i:00 — 分钟精度但尾部带 :00
      return d.format('YYYY-MM-DD HH:mm') + ':00'
    case '1h':
      // MySQL %Y-%m-%d %H:00:00 — 小时精度但尾部带 :00:00
      return d.format('YYYY-MM-DD HH') + ':00:00'
    case '1d':
      return d.format('YYYY-MM-DD')
    case '1w':
      // MySQL %Y-%u — 年份 + ISO 周号（无补零）
      return `${d.year()}-${d.isoWeek()}`
    default:
      return d.format('YYYY-MM-DD')
  }
}

// ── MySQL 格式字符串 → 时间戳 ──

/**
 * 将后端 MySQL DATE_FORMAT 输出的时间字符串解析为毫秒时间戳。
 * 用于 ECharts time 轴的数据点定位和告警标注定位。
 */
export function parseBucketTimeString(xStr: string, bucket: string): number {
  if (!xStr) return 0
  switch (bucket) {
    case '1s':
    case '1m':
    case '1h':
      // MySQL 输出含完整日期+时间部分："2024-01-15 14:30:00" / "2024-01-15 14:00:00"
      return dayjs(xStr).valueOf()
    case '1d':
      // MySQL 输出仅日期："2024-01-15"
      return dayjs(xStr).valueOf()
    case '1w':
      // MySQL %Y-%u 输出："2024-3" → 需手动解析 year + isoWeek
      const parts = xStr.split('-')
      const year = parseInt(parts[0], 10)
      const week = parseInt(parts[1], 10)
      return dayjs().year(year).isoWeek(week).startOf('isoWeek').valueOf()
    default:
      return dayjs(xStr).valueOf()
  }
}

// ── 时间轴标签格式化 ──

/**
 * 根据 bucket 粒度格式化时间轴标签（接收时间戳，返回精简显示文本）。
 * 参考 Grafana：粒度越细显示越详细，去掉与同图表上下文重复的部分。
 */
export function formatTimeAxisLabel(ts: number, bucket: string): string {
  if (!ts) return ''
  const d = dayjs(ts)
  switch (bucket) {
    case '1s':
      return d.format('HH:mm:ss')
    case '1m':
      return d.format('HH:mm')
    case '1h':
      return d.format('MM-DD HH:mm')
    case '1d':
      return d.format('MM-DD')
    case '1w':
      return `${d.isoWeek()}周`
    default:
      return d.format('MM-DD')
  }
}

export const QUICK_TIME_OPTIONS = [
  { value: '5m', label: '5分钟' },
  { value: '15m', label: '15分钟' },
  { value: '30m', label: '30分钟' },
  { value: '1h', label: '1小时' },
  { value: '6h', label: '6小时' },
  { value: '12h', label: '12小时' },
  { value: '1d', label: '1天' },
  { value: '7d', label: '7天' },
  { value: '30d', label: '30天' },
]

// ── X 轴标签格式化 ──

/** 根据 bucket 粒度智能格式化 X 轴时间标签，去掉冗余部分 */
export function formatXAxisLabel(value: string, bucket: string): string {
  if (!value) return ''
  switch (bucket) {
    case '1s':
    case '1m':
      // "2024-01-15 14:30" → "14:30"（同日只显示时间）
      return value.length > 10 ? value.slice(11) : value
    case '1h':
      // "2024-01-15 14" → "01-15 14:00"
      return value.length > 10 ? value.slice(5) : value
    case '1d':
      // "2024-01-15" → "01-15"
      return value.length > 7 ? value.slice(5) : value
    case '1w':
      return value
    default:
      return value.length > 10 ? value.slice(5) : value
  }
}

/** 将 TimeRangeOverride 展平为 query params */
export function flattenTimeRange(tr: TimeRangeOverride): Record<string, string> {
  if (tr.type === 'relative') {
    return { time_range_type: 'relative', time_range_value: tr.value ?? '1d' }
  }
  return {
    time_range_type: 'absolute',
    time_range_start: tr.start ?? '',
    time_range_end: tr.end ?? '',
  }
}

/** 从 TimeRangeOverride 计算出本地时间区间（start/end），供告警记录查询和图表标注使用。
 *  输出格式为 YYYY-MM-DD HH:mm:ss（不带时区后缀），与后端 ParseAbsoluteTime 兼容。 */
export function effectiveTimeRangeToInterval(tr: TimeRangeOverride): { start: string; end: string } {
  if (tr.type === 'relative' && tr.value) {
    const ms = parseRelativeMs(tr.value)
    const end = dayjs()
    const start = end.subtract(ms, 'ms')
    return { start: start.format('YYYY-MM-DD HH:mm:ss'), end: end.format('YYYY-MM-DD HH:mm:ss') }
  }
  if (tr.type === 'absolute' && tr.start && tr.end) {
    // absolute 类型可能已为本地格式，确保格式一致
    return { start: dayjs(tr.start).format('YYYY-MM-DD HH:mm:ss'), end: dayjs(tr.end).format('YYYY-MM-DD HH:mm:ss') }
  }
  const end = dayjs()
  return { start: end.subtract(24 * 60 * 60 * 1000, 'ms').format('YYYY-MM-DD HH:mm:ss'), end: end.format('YYYY-MM-DD HH:mm:ss') }
}

// ── Bucket 毫秒间隔 ──

const BUCKET_MS: Record<string, number> = {
  '1s': 1000,
  '1m': 60 * 1000,
  '1h': 60 * 60 * 1000,
  '1d': 24 * 60 * 60 * 1000,
  '1w': 7 * 24 * 60 * 60 * 1000,
}

/** 将 bucket 字符串转换为毫秒间隔 */
export function parseBucketMs(bucket: string): number {
  return BUCKET_MS[bucket] ?? 60 * 60 * 1000
}

// ── 时间对齐辅助（拖拽选区扩展到 bucket 边界）──

/** 将时间戳向下对齐到 bucket 起始边界（如 04:43 → 04:00 for '1h'） */
export function alignToBucketStart(timeMs: number, bucket: string): number {
  const d = dayjs(timeMs)
  switch (bucket) {
    case '1s':
      return d.startOf('second').valueOf()
    case '1m':
      return d.startOf('minute').valueOf()
    case '1h':
      return d.startOf('hour').valueOf()
    case '1d':
      return d.startOf('day').valueOf()
    case '1w':
      return d.startOf('isoWeek').valueOf()
    default:
      return d.startOf('hour').valueOf()
  }
}

/** 将时间戳向上对齐到 bucket 结束边界（如 16:10 → 17:00 for '1h'） */
export function alignToBucketEnd(timeMs: number, bucket: string): number {
  const d = dayjs(timeMs)
  switch (bucket) {
    case '1s':
      return d.endOf('second').valueOf()
    case '1m':
      return d.endOf('minute').valueOf()
    case '1h':
      return d.endOf('hour').valueOf()
    case '1d':
      return d.endOf('day').valueOf()
    case '1w':
      return d.endOf('isoWeek').valueOf()
    default:
      return d.endOf('hour').valueOf()
  }
}

