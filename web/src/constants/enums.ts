// System-level enums (fixed, not configurable per data source)
export const CATEGORY_STATUS_MAP: Record<number, { label: string; color: string }> = {
  0: { label: '待分类', color: 'default' },
  1: { label: '已分类', color: 'success' },
  2: { label: '分类失败', color: 'error' },
  3: { label: '手动分类', color: 'warning' },
}

export const SENTIMENTS = [
  { value: 'positive', label: '正向', color: 'green' },
  { value: 'neutral', label: '中性', color: 'blue' },
  { value: 'negative', label: '负向', color: 'red' },
]

export const SYNC_STATUS_MAP: Record<string, { label: string; color: string }> = {
  running: { label: '运行中', color: 'processing' },
  success: { label: '成功', color: 'success' },
  failed: { label: '失败', color: 'error' },
}

export const CATEGORIES = ['功能建议', '问题报告', '体验反馈', '其他']

export const BUSINESS_MODULES = ['账号体系', '消息通知', '支付订单', '内容浏览', '上传存储', '数据统计', '社交互动', '设置中心', '其他']

// Classification log status
export const LOG_STATUS_MAP: Record<number, { label: string; color: string }> = {
  0: { label: '待处理', color: 'default' },
  1: { label: '成功', color: 'success' },
  2: { label: '失败', color: 'error' },
  3: { label: '超时', color: 'warning' },
}

// Dynamic enums: loaded from enum_configs table via aggregatedEnums API
// Fields: app_id, platform, user_mode, platform_id, channel_id, etc.
export type DynamicEnums = Record<string, Record<string, string>>