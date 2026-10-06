import { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Spin,
  Empty,
  Statistic,
  message,
  Popconfirm,
  Tag,
  Space,
  DatePicker,
  Segmented,
} from 'antd'
import { PlusOutlined, DeleteOutlined, ReloadOutlined, EditOutlined, ClockCircleOutlined, LockOutlined, HolderOutlined, SaveOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { WidthProvider, ReactGridLayout } from 'react-grid-layout/legacy'
import type { Layout } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import dayjs from 'dayjs'
import {
  getDashboard,
  createPanel,
  updatePanel,
  deletePanel,
  queryPanel,
  getPanelTemplates,
  getPanelAlertState,
  type Dashboard,
  type Panel,
  type PanelAlertConfig,
  type AlertRuleStateBrief,
} from '../../api/dashboard'
import { listAlertRecords, deleteAlertRule, type AlertRecord } from '../../api/alert'
import { listNotificationChannels, type NotificationChannel } from '../../api/notification-channel'
import { useDashboardEnums } from '../../hooks/useDashboardEnums'
import { normalizeFilterTree } from '../../types/filterNode'
import PanelAlertConfigEditor from '../../components/PanelAlertConfigEditor'
import QueryConfigEditor, { DEFAULT_QUERY_CONFIG } from '../../components/QueryConfigEditor'
import { buildAlertAnnotations, isAlertableChartType } from '../../utils/alertChartAnnotation'
import {
  type TimeRangeOverride,
  autoBucket,
  formatTimeAxisLabel,
  parseBucketTimeString,
  effectiveTimeRangeToInterval,
  QUICK_TIME_OPTIONS,
  alignToBucketStart,
  alignToBucketEnd,
} from '../../utils/timeRangeUtils'

const FIELD_LABELS: Record<string, string> = {
  platform_id: '平台',
  app_id: '应用',
  app_version: '版本号',
  category: '分类',
  sentiment: '情感倾向',
  business_module: '业务模块',
  category_status: '分类状态',
}

const GRID_COLS = 4
const GRID_ROW_HEIGHT = 60
const GRID_MARGIN: [number, number] = [12, 12]
const PANEL_FALLBACK_HEIGHT = 260


interface PanelData {
  rows: Array<Record<string, unknown>>
  loading: boolean
}

// 右键拖拽选区：避免与 dataZoom 左键拖动冲突
// 长按拖拽选区：触控板/触屏按住 500ms 后拖动即可选区，与左键快速拖动（dataZoom）不冲突
type SelectionPhase = 'idle' | 'waiting' | 'active'

const LONG_PRESS_DELAY = 500
const MOVE_THRESHOLD = 5

function ChartWithDragSelect({
  option,
  onTimeRangeSelect,
  style,
}: {
  option: Record<string, unknown>
  onTimeRangeSelect: (timeRange: TimeRangeOverride) => void
  style?: React.CSSProperties
}) {
  const chartRef = useRef<any>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const dragStartXRef = useRef(0)
  const dragCurrentXRef = useRef(0)
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [phase, setPhase] = useState<SelectionPhase>('idle')
  const [overlayPos, setOverlayPos] = useState<{ left: number; width: number } | null>(null)

  const getContainerRect = () => containerRef.current?.getBoundingClientRect() ?? new DOMRect()

  const clientXFromEvent = (e: MouseEvent | TouchEvent): number => {
    if ('touches' in e) return e.touches.length > 0 ? e.touches[0].clientX : (e as TouchEvent).changedTouches[0].clientX
    return e.clientX
  }

  const completeSelection = () => {
    const startX = dragStartXRef.current
    const endX = dragCurrentXRef.current
    if (Math.abs(endX - startX) < 10) {
      setPhase('idle')
      setOverlayPos(null)
      return
    }

    const echartsInstance = chartRef.current?.getEchartsInstance?.()
    if (!echartsInstance) { setPhase('idle'); setOverlayPos(null); return }

    const echartsDom = echartsInstance.getDom()
    const echartsRect = echartsDom.getBoundingClientRect()
    const containerRect = getContainerRect()
    const offsetX = echartsRect.left - containerRect.left

    const leftX = Math.min(startX, endX) - offsetX
    const rightX = Math.max(startX, endX) - offsetX

    const startCoord = echartsInstance.convertFromPixel('grid', [leftX, 0])
    const endCoord = echartsInstance.convertFromPixel('grid', [rightX, 0])

    if (!startCoord || !endCoord) { setPhase('idle'); setOverlayPos(null); return }

    if (Math.abs(endCoord[0] - startCoord[0]) < 60_000) {
      setPhase('idle')
      setOverlayPos(null)
      return
    }

    const startTimeMs = Math.min(startCoord[0], endCoord[0])
    const endTimeMs = Math.max(startCoord[0], endCoord[0])

    onTimeRangeSelect({
      type: 'absolute',
      start: dayjs(alignToBucketStart(startTimeMs, '1m')).format('YYYY-MM-DD HH:mm:ss'),
      end: dayjs(alignToBucketEnd(endTimeMs, '1m')).format('YYYY-MM-DD HH:mm:ss'),
    })

    setPhase('idle')
    setOverlayPos(null)
  }

  // 右键拖拽（原有逻辑）
  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault()
    if (longPressTimerRef.current) { clearTimeout(longPressTimerRef.current); longPressTimerRef.current = null }
    const rect = getContainerRect()
    const x = e.clientX - rect.left
    dragStartXRef.current = x
    dragCurrentXRef.current = x
    setPhase('active')
    setOverlayPos({ left: x, width: 0 })
  }

  // 长按启动：mousedown / touchstart 后延迟判定
  const handlePressStart = (clientX: number) => {
    const rect = getContainerRect()
    const x = clientX - rect.left
    dragStartXRef.current = x
    dragCurrentXRef.current = x
    setPhase('waiting')
    longPressTimerRef.current = setTimeout(() => {
      longPressTimerRef.current = null
      setPhase('active')
      setOverlayPos({ left: x, width: 0 })
    }, LONG_PRESS_DELAY)
  }

  const cancelLongPress = () => {
    if (longPressTimerRef.current) { clearTimeout(longPressTimerRef.current); longPressTimerRef.current = null }
    setPhase('idle')
  }

  // 鼠标长按
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return // 仅左键
    handlePressStart(e.clientX)
  }

  // 触屏长按
  const handleTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length !== 1) return
    handlePressStart(e.touches[0].clientX)
  }

  // ── 拖动 & 释放 ──

  useEffect(() => {
    if (phase === 'idle') return

    const handleMove = (e: MouseEvent | TouchEvent) => {
      const rect = getContainerRect()
      const x = Math.max(0, Math.min(clientXFromEvent(e) - rect.left, rect.width))

      if (phase === 'waiting') {
        // 等待期移动超过阈值 → 取消长按，让 ECharts 正常处理
        if (Math.abs(x - dragStartXRef.current) > MOVE_THRESHOLD) {
          cancelLongPress()
        }
        return
      }

      // active 阶段：更新选区覆盖层
      dragCurrentXRef.current = x
      const left = Math.min(dragStartXRef.current, x)
      const width = Math.abs(x - dragStartXRef.current)
      setOverlayPos({ left, width })

      // 阻止触屏滚动
      if ('touches' in e) e.preventDefault()
    }

    const handleUp = (e: MouseEvent | TouchEvent) => {
      if (phase === 'waiting') {
        cancelLongPress()
        return
      }
      if (phase === 'active') {
        // 更新最终位置（触屏 touchend 用 changedTouches）
        if ('changedTouches' in e) {
          const rect = getContainerRect()
          dragCurrentXRef.current = Math.max(0, Math.min(e.changedTouches[0].clientX - rect.left, rect.width))
        }
        completeSelection()
      }
    }

    // 长按激活期间使用 capture 阶段拦截，防止 ECharts 接收到事件
    const useCapture = phase === 'active'
    document.addEventListener('mousemove', handleMove, useCapture)
    document.addEventListener('mouseup', handleUp, useCapture)
    document.addEventListener('touchmove', handleMove, { capture: useCapture, passive: false })
    document.addEventListener('touchend', handleUp, useCapture)
    return () => {
      document.removeEventListener('mousemove', handleMove, useCapture)
      document.removeEventListener('mouseup', handleUp, useCapture)
      document.removeEventListener('touchmove', handleMove, useCapture)
      document.removeEventListener('touchend', handleUp, useCapture)
    }
  }, [phase]) // eslint-disable-line react-hooks/exhaustive-deps

  // 长按等待中：显示半透明提示，让用户知道正在识别长按
  const showWaitingHint = phase === 'waiting'

  // 选区激活中：全屏覆盖拦截 ECharts 交互
  const overlayActive = phase === 'active' && overlayPos !== null

  return (
    <div
      ref={containerRef}
      style={{ position: 'relative', ...(overlayActive || showWaitingHint ? { userSelect: 'none' } : {}) }}
      onContextMenu={handleContextMenu}
      onMouseDown={handleMouseDown}
      onTouchStart={handleTouchStart}
    >
      {/* 长按等待提示 */}
      {showWaitingHint && (
        <div style={{
          position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(24, 144, 255, 0.05)', zIndex: 8, pointerEvents: 'none',
          borderRadius: 4,
        }} />
      )}
      {/* 选区激活覆盖：拦截所有交互防止 ECharts dataZoom 冲突 */}
      {overlayActive && (
        <div style={{
          position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0, 0, 0, 0.2)', zIndex: 9,
        }} />
      )}
      <ReactECharts
        ref={chartRef}
        option={option}
        notMerge={false}
        style={style}
      />
      {overlayPos && phase === 'active' && (
        <div style={{
          position: 'absolute', top: 0, bottom: 0,
          left: overlayPos.left,
          width: overlayPos.width,
          borderLeft: '2px solid #1890ff', borderRight: '2px solid #1890ff',
          background: 'rgba(24, 144, 255, 0.1)',
          zIndex: 11, pointerEvents: 'none',
        }} />
      )}
    </div>
  )
}

const GridLayout = WidthProvider(ReactGridLayout)

// 从 panel position 字段解析 react-grid-layout Layout
// 无效/不合理 position 时自动计算默认布局：
// - null / 缺少字段
// - 旧 12 列格式（w > 4）
// - 明显错误的垂直堆叠（所有面板 x=0, y 间隔相同）
function isReasonablePosition(pos: Record<string, number> | undefined, chartType: string): boolean {
  if (!pos || pos.x == null || pos.y == null || pos.w == null || pos.h == null) return false
  if (pos.w > 4 || pos.h < getPanelMinH(chartType)) return false
  if (pos.w < getPanelMinW(chartType)) return false
  if (pos.x < 0 || pos.y < 0) return false
  if (pos.x + pos.w > 4) return false
  return true
}

function getPanelMinW(chartType: string): number {
  return 1
}

function getPanelMinH(chartType: string): number {
  return chartType === 'number' ? 3 : 4
}

function normalizeLayoutItemByChartType(item: Layout[number], chartType: string): Layout[number] {
  const minH = getPanelMinH(chartType)
  const minW = getPanelMinW(chartType)
  const normalizedW = Math.max(item.w, minW)
  return {
    ...item,
    x: Math.max(0, Math.min(item.x, 4 - normalizedW)),
    h: Math.max(item.h, minH),
    w: normalizedW,
    minH,
    maxH: 12,
    minW,
    maxW: 4,
  }
}

function getPanelBodyHeight(layoutItem: Layout[number] | undefined): number {
  if (!layoutItem) return PANEL_FALLBACK_HEIGHT
  return layoutItem.h * GRID_ROW_HEIGHT + Math.max(0, layoutItem.h - 1) * GRID_MARGIN[1]
}

function getChartAreaHeight(layoutItem: Layout[number] | undefined, chartType: string): number {
  const panelBodyHeight = getPanelBodyHeight(layoutItem)
  const reservedHeight = chartType === 'number' ? 120 : 110
  const minChartHeight = chartType === 'number' ? 72 : 120
  return Math.max(minChartHeight, panelBodyHeight - reservedHeight)
}

function buildDefaultLayout(panels: Panel[]): Layout {
  let curX = 0
  let curY = 0
  const MAX_COLS = GRID_COLS
  return panels.map((panel) => {
    const pos = panel.position as Record<string, number> | undefined
    const minH = getPanelMinH(panel.chart_type)
    const minW = getPanelMinW(panel.chart_type)
    if (isReasonablePosition(pos, panel.chart_type)) {
      return { i: String(panel.id), x: pos!.x, y: pos!.y, w: Math.max(pos!.w, minW), h: Math.max(pos!.h, minH), minW, maxW: 4, minH, maxH: 12 }
    }
    if (pos && pos.x != null && pos.y != null && pos.w != null && pos.h != null) {
      return normalizeLayoutItemByChartType({
        i: String(panel.id),
        x: Math.max(0, pos.x),
        y: Math.max(0, pos.y),
        w: pos.w,
        h: pos.h,
        minW,
        maxW: 4,
        minH,
        maxH: 12,
      }, panel.chart_type)
    }
    const w = minW
    if (curX + w > MAX_COLS) {
      curX = 0
      curY++
    }
    const item = { i: String(panel.id), x: curX, y: curY, w, h: Math.max(5, minH), minW, maxW: 4, minH, maxH: 12 }
    curX += w
    if (curX >= MAX_COLS) {
      curX = 0
      curY++
    }
    return item
  })
}

function niceRound(value: number): number {
  if (value <= 0) return 1
  const exp = Math.floor(Math.log10(value))
  const fraction = value / Math.pow(10, exp)
  let nice: number
  if (fraction <= 1) nice = 1
  else if (fraction <= 2) nice = 2
  else if (fraction <= 5) nice = 5
  else nice = 10
  return nice * Math.pow(10, exp)
}

export default function DashboardView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { resolveFilterNodeLabels, enums } = useDashboardEnums()
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [panelData, setPanelData] = useState<Record<number, PanelData>>({})
  const [alertStates, setAlertStates] = useState<Record<number, AlertRuleStateBrief | null>>({})
  const [alertRecords, setAlertRecords] = useState<Record<number, AlertRecord[]>>({})
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingPanel, setEditingPanel] = useState<Panel | null>(null)
  const [templates, setTemplates] = useState<Array<Record<string, unknown>>>([])
  // 网格布局状态
  const [gridLayout, setGridLayout] = useState<Layout>([])
  // 上次保存到后端的布局快照，用于取消时恢复
  const lastSavedRef = useRef<Layout>([])
  // 编辑模式状态
  const [isEditing, setIsEditing] = useState(false)
  const [editLayout, setEditLayout] = useState<Layout>([])
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  // 监听图表类型变化，驱动 QueryConfigEditor 的 pie/非 pie 模式切换
  const createChartType = Form.useWatch('chart_type', form)
  const editChartType = Form.useWatch('chart_type', editForm)
  // 告警配置状态（独立于 form，因为 PanelAlertConfigEditor 有自己的 onChange）
  const [createAlertConfig, setCreateAlertConfig] = useState<PanelAlertConfig | null>(null)
  const [editAlertConfig, setEditAlertConfig] = useState<PanelAlertConfig | null>(null)
  // 查询配置状态（独立于 form，因为 QueryConfigEditor 有自己的 onChange）
  const [createQueryConfig, setCreateQueryConfig] = useState<Record<string, unknown>>(DEFAULT_QUERY_CONFIG)
  const [editQueryConfig, setEditQueryConfig] = useState<Record<string, unknown>>({})
  // 全局时间范围（默认最近 1 天）
  const [dashboardTimeRange, setDashboardTimeRange] = useState<TimeRangeOverride>({ type: 'relative', value: '1d' })
  // 面板独立时间覆盖：null 表示跟随仪表盘
  const [panelTimeOverrides, setPanelTimeOverrides] = useState<Record<number, TimeRangeOverride | null>>({})
  // 自定义绝对时间范围选择器开关
  const [absolutePickerOpen, setAbsolutePickerOpen] = useState(false)
  // 面板级自定义时间范围选择器（panelId → 是否打开）
  const [panelPickerOpen, setPanelPickerOpen] = useState<Record<number, boolean>>({})

  // 面板生效时间范围：优先面板独立覆盖，否则跟随仪表盘
  const getEffectiveTimeRange = useCallback((panelId: number): TimeRangeOverride => {
    return panelTimeOverrides[panelId] ?? dashboardTimeRange
  }, [panelTimeOverrides, dashboardTimeRange])

  const fetchDashboard = useCallback(async () => {
    if (!id) return
    const d = await getDashboard(Number(id))
    setDashboard(d)
    // 从 panel position 字段初始化网格布局
    if (d.panels) {
      const layout = buildDefaultLayout(d.panels)
      setGridLayout(layout)
      lastSavedRef.current = [...layout]
    }
    return d
  }, [id])

  // 编辑模式下拖拽/resize 完成后更新本地暂存布局（不立即保存）
  const handleEditLayoutChange = useCallback((newLayout: Layout) => {
    setEditLayout(newLayout)
  }, [])

  // 进入编辑模式：复制当前布局到暂存
  const startEdit = () => {
    setEditLayout([...gridLayout])
    setIsEditing(true)
  }

  // 取消编辑：丢弃暂存布局，恢复原布局
  const cancelEdit = () => {
    setEditLayout([])
    setIsEditing(false)
  }

  // 保存编辑：将暂存布局写入后端，退出编辑模式
  const saveEdit = async () => {
    if (!dashboard) return
    try {
      for (const item of editLayout) {
        const panel = dashboard.panels?.find((p) => String(p.id) === item.i)
        if (!panel) continue
        const normalized = normalizeLayoutItemByChartType(item, panel.chart_type)
        await updatePanel(dashboard.id, panel.id, {
          name: panel.name,
          chart_type: panel.chart_type,
          query_config: panel.query_config,
          position: { x: normalized.x, y: normalized.y, w: normalized.w, h: normalized.h },
          refresh_seconds: panel.refresh_seconds,
          alert_config: panel.alert_config,
        })
      }
      const normalizedLayout = editLayout.map((item) => {
        const panel = dashboard.panels?.find((p) => String(p.id) === item.i)
        return panel ? normalizeLayoutItemByChartType(item, panel.chart_type) : item
      })
      setGridLayout(normalizedLayout)
      lastSavedRef.current = normalizedLayout
      setEditLayout([])
      setIsEditing(false)
      message.success('布局已保存')
    } catch {
      message.error('布局保存失败')
    }
  }

  const loadPanelData = useCallback(async (panel: Panel, timeOverride?: TimeRangeOverride) => {
    // 刷新时保留旧数据避免图表闪动；初次加载（无旧数据）时才清空显示 Spin
    setPanelData((prev) => ({
      ...prev,
      [panel.id]: { rows: prev[panel.id]?.rows ?? [], loading: true },
    }))
    try {
      const rows = await queryPanel(panel.id, timeOverride)
      setPanelData((prev) => ({ ...prev, [panel.id]: { rows: rows ?? [], loading: false } }))
    } catch {
      setPanelData((prev) => ({
        ...prev,
        [panel.id]: { rows: prev[panel.id]?.rows ?? [], loading: false },
      }))
    }
  }, [])

  const loadAlertState = useCallback(async (panel: Panel, timeOverride?: TimeRangeOverride) => {
    if (!panel.alert_config || !isAlertableChartType(panel.chart_type)) {
      setAlertStates((prev) => ({ ...prev, [panel.id]: null }))
      setAlertRecords((prev) => ({ ...prev, [panel.id]: [] }))
      return
    }
    try {
      const state = await getPanelAlertState(panel.id)
      setAlertStates((prev) => ({ ...prev, [panel.id]: state.alert_state ?? null }))
      // 加载告警记录 — 按当前生效时间范围过滤
      if (state.alert_state) {
        const tr = timeOverride ?? dashboardTimeRange
        const { start, end } = effectiveTimeRangeToInterval(tr)
        const recs = await listAlertRecords({ rule_id: state.alert_state.rule_id, limit: 50, start_at: start, end_at: end })
        setAlertRecords((prev) => ({ ...prev, [panel.id]: recs ?? [] }))
      } else {
        setAlertRecords((prev) => ({ ...prev, [panel.id]: [] }))
      }
    } catch {
      setAlertStates((prev) => ({ ...prev, [panel.id]: null }))
      setAlertRecords((prev) => ({ ...prev, [panel.id]: [] }))
    }
  }, [dashboardTimeRange])

  // 刷新所有面板数据（使用当前生效的时间范围）
  const refreshAllPanels = useCallback(() => {
    if (!dashboard?.panels) return
    dashboard.panels.forEach((p) => loadPanelData(p, dashboardTimeRange))
  }, [dashboard, dashboardTimeRange, loadPanelData])

  // 刷新所有面板告警状态
  const refreshAllAlertStates = useCallback(() => {
    if (!dashboard?.panels) return
    dashboard.panels.forEach((p) => loadAlertState(p, dashboardTimeRange))
  }, [dashboard, dashboardTimeRange, loadAlertState])

  // 防止 StrictMode 双执行导致重复请求
  const initialLoadDoneRef = useRef(false)

  // 初始加载：仅获取仪表盘元数据，数据由 time range effect 驱动
  useEffect(() => {
    if (initialLoadDoneRef.current) return
    initialLoadDoneRef.current = true
    fetchDashboard()
    getPanelTemplates().then(setTemplates).catch(() => {})
    listNotificationChannels().then(setChannels).catch(() => {})
  }, [fetchDashboard]) // eslint-disable-line react-hooks/exhaustive-deps

  // 全局时间范围变化或 dashboard 加载完成后：清除面板独立覆盖，刷新所有数据
  useEffect(() => {
    setPanelTimeOverrides({})
    refreshAllPanels()
    refreshAllAlertStates()
  }, [dashboardTimeRange, refreshAllPanels, refreshAllAlertStates])

  // ── 创建面板 ──

  const handleCreatePanel = async (values: {
    name: string
    chart_type: string
  }) => {
    // 计算新面板的默认位置：追加到现有布局末尾
    const panels = dashboard?.panels ?? []
    const defaultW = values.chart_type === 'number' ? 1 : 2
    const existingLayout = buildDefaultLayout(panels)
    const maxY = existingLayout.reduce((max, item) => Math.max(max, item.y + item.h), 0)
    const newPosition = { x: 0, y: maxY, w: defaultW, h: Math.max(5, getPanelMinH(values.chart_type)) }
    await createPanel(Number(id), {
      name: values.name,
      chart_type: values.chart_type,
      query_config: createQueryConfig,
      alert_config: createAlertConfig,
      position: newPosition,
    })
    message.success('Panel 已创建')
    setModalOpen(false)
    form.resetFields()
    setCreateAlertConfig(null)
    setCreateQueryConfig(DEFAULT_QUERY_CONFIG)
    const d = await fetchDashboard()
    if (d?.panels) {
      const newPanel = d.panels[d.panels.length - 1]
      if (newPanel) {
        loadPanelData(newPanel, dashboardTimeRange)
        loadAlertState(newPanel, dashboardTimeRange)
      }
    }
  }

  // ── 编辑面板 ──

  const openEditPanel = (panel: Panel) => {
    setEditingPanel(panel)
    editForm.setFieldsValue({
      name: panel.name,
      chart_type: panel.chart_type,
    })
    setEditQueryConfig(panel.query_config)
    setEditAlertConfig(panel.alert_config ?? null)
    setEditModalOpen(true)
  }

  const BUCKET_LABELS: Record<string, string> = { '1m': '按分钟 (1m)', '1h': '按小时 (1h)', '1d': '按天 (1d)' }

  const handleEditPanel = async (values: {
    name: string
    chart_type: string
  }) => {
    // 检测粒度变更
    const oldBucket = ((editingPanel!.query_config as Record<string, unknown>)?.x_dimension as { bucket?: string })?.bucket || '1m'
    const newBucket = ((editQueryConfig as Record<string, unknown>)?.x_dimension as { bucket?: string })?.bucket || '1m'
    const bucketChanged = oldBucket !== newBucket

    // 粒度变更且有告警配置时，需确认关闭告警
    if (bucketChanged && editingPanel!.alert_config) {
      // 保留当前布局位置
      const currentLayoutItem = gridLayout.find(item => item.i === String(editingPanel!.id))
      Modal.confirm({
        title: '告警规则将因粒度变更而关闭',
        content: `时间粒度已从「${BUCKET_LABELS[oldBucket] || oldBucket}」变为「${BUCKET_LABELS[newBucket] || newBucket}」，当前告警规则基于旧粒度设置，继续保存将自动关闭告警。重新开启告警时会创建新的告警规则。`,
        okText: '确认保存',
        cancelText: '取消',
        onOk: async () => {
          // 删除旧告警规则
          const as = alertStates[editingPanel!.id]
          if (as?.rule_id) {
            await deleteAlertRule(as.rule_id)
          }
          await updatePanel(Number(id), editingPanel!.id, {
            name: values.name,
            chart_type: values.chart_type,
            query_config: editQueryConfig,
            alert_config: null,
            position: currentLayoutItem
              ? { x: currentLayoutItem.x, y: currentLayoutItem.y, w: currentLayoutItem.w, h: currentLayoutItem.h }
              : undefined,
          })
          message.success('Panel 已更新，告警已关闭')
          setEditModalOpen(false)
          editForm.resetFields()
          setEditAlertConfig(null)
          setEditQueryConfig({})
          const panelId = editingPanel!.id
          setEditingPanel(null)
          // 清除告警状态
          setAlertStates(prev => ({ ...prev, [panelId]: null }))
          setAlertRecords(prev => ({ ...prev, [panelId]: [] }))
          const d = await fetchDashboard()
          if (d?.panels) {
            const updated = d.panels.find(p => p.id === panelId)
            if (updated) {
              loadPanelData(updated, dashboardTimeRange)
            }
          }
        },
      })
      return
    }

    // 保留当前布局位置
    const currentLayoutItem = gridLayout.find(item => item.i === String(editingPanel!.id))
    await updatePanel(Number(id), editingPanel!.id, {
      name: values.name,
      chart_type: values.chart_type,
      query_config: editQueryConfig,
      alert_config: editAlertConfig,
      position: currentLayoutItem
        ? { x: currentLayoutItem.x, y: currentLayoutItem.y, w: currentLayoutItem.w, h: currentLayoutItem.h }
        : undefined,
    })
    message.success('Panel 已更新')
    setEditModalOpen(false)
    editForm.resetFields()
    setEditAlertConfig(null)
    setEditQueryConfig({})
    setEditingPanel(null)
    const d = await fetchDashboard()
    if (d?.panels) {
      const updated = d.panels.find((p) => p.id === editingPanel!.id)
      if (updated) {
        loadPanelData(updated, dashboardTimeRange)
        loadAlertState(updated, dashboardTimeRange)
      }
    }
  }

  const handleDeletePanel = async (dashId: number, panelId: number) => {
    await deletePanel(dashId, panelId)
    message.success('已删除')
    await fetchDashboard()
  }

  const applyTemplate = (tplIdx: string) => {
    const tpl = templates[Number(tplIdx)]
    if (!tpl) return
    form.setFieldsValue({
      name: tpl.name,
      chart_type: tpl.chart_type,
    })
    setCreateQueryConfig(tpl.query_config as Record<string, unknown>)
  }

  // ── 图表构建 ──

  const buildChartOption = (
    panel: Panel,
    rows: Array<Record<string, unknown>>,
    timeRange: TimeRangeOverride,
    alertState?: AlertRuleStateBrief | null,
    records?: AlertRecord[],
  ) => {
    if (!rows || rows.length === 0) return {}
    const hasX = rows.some((r) => r.x != null)

    // 从面板配置读取粒度，替代启发式判断
    const cfgBucket = (panel.query_config as Record<string, unknown>)?.x_dimension as { bucket?: string } | undefined
    const bucket = cfgBucket?.bucket || '1m'

    // 告警标注（仅 line/bar）— 计算 Y 轴封顶值
    const { start: trStart, end: trEnd } = effectiveTimeRangeToInterval(timeRange)
    const xAxisMin = alignToBucketStart(dayjs(trStart).valueOf(), bucket)
    const xAxisMax = alignToBucketEnd(dayjs(trEnd).valueOf(), bucket)
    const chartEndTime = dayjs().valueOf()

    // 阈值与数据范围对比，决定 Y 轴封顶策略
    const threshold = alertState?.threshold ?? 0
    const RATIO_LIMIT = 3
    const isExtremeRatio = threshold > Math.max((rows.reduce((m, r) => Math.max(m, Number(r.value ?? 0)), 0)) * RATIO_LIMIT, 10)
    const yAxisMax = isAlertableChartType(panel.chart_type) && threshold > 0
      ? (isExtremeRatio
        ? niceRound(Math.max(rows.reduce((m, r) => Math.max(m, Number(r.value ?? 0)), 0) * 1.5, 5))
        : niceRound(Math.max(rows.reduce((m, r) => Math.max(m, Number(r.value ?? 0)), 0) * 1.2, threshold * 1.05)))
      : undefined

    const annotations = isAlertableChartType(panel.chart_type) && alertState
      ? buildAlertAnnotations(alertState, records ?? [], bucket, chartEndTime, yAxisMax)
      : {}

    const axisLabelFormatter = (val: number) => formatTimeAxisLabel(val, bucket)

    if (panel.chart_type === 'pie') {
      // 获取饼图的分组维度字段（从 query_config.group_by）
      const groupByFields = (panel.query_config as Record<string, unknown>)?.group_by as string[] | undefined
      const groupByField = groupByFields?.[0] ?? ''
      // 对应字段的枚举映射
      const enumMap = enums[groupByField]
      const groups = Object.entries(
        rows.reduce((acc: Record<string, number>, r) => {
          const rawKey = String(r[groupByField] ?? r[Object.keys(r).find((k) => k !== 'value') ?? 'x'] ?? '未知')
          // 用枚举中文标签展示，没有枚举则展示原始值
          const displayKey = enumMap ? (enumMap[String(rawKey)] ?? String(rawKey)) : String(rawKey)
          acc[displayKey] = (acc[displayKey] ?? 0) + Number(r.value ?? 0)
          return acc
        }, {}),
      )
      return {
        tooltip: { trigger: 'item' },
        series: [{ type: 'pie', data: groups.map(([name, value]) => ({ name, value })), radius: '60%' }],
      }
    }

    // line / bar — 有时间维度时使用 time 轴
    if (hasX) {
      // 1m 粒度数据：前端按 1m 最小粒度自动补零，确保每个分钟都有图表节点
      let chartData: Array<[number, number]>
      if (bucket === '1m') {
        // 构建已有数据索引（毫秒时间戳 → 值）
        const dataByMinute = new Map<number, number>()
        for (const r of rows) {
          const ts = parseBucketTimeString(String(r.x ?? ''), '1m')
          dataByMinute.set(ts, Number(r.value ?? 0))
        }
        // 从 xAxisMin 到 xAxisMax 逐分钟生成完整节点序列
        chartData = []
        for (let ms = xAxisMin; ms <= xAxisMax; ms += 60 * 1000) {
          chartData.push([ms, dataByMinute.get(ms) ?? 0])
        }
      } else {
        chartData = rows.map((r) => [
          parseBucketTimeString(String(r.x ?? ''), bucket),
          Number(r.value ?? 0),
        ])
      }

      // markLine 配置：移除 silent，启用 tooltip，限制选中时的线粗度
      const markLineOpt = annotations.markLine
        ? {
            symbol: 'none',
            data: annotations.markLine.data,
            tooltip: { show: true },
            label: { show: false },
            emphasis: {
              lineStyle: { width: 1.5 },
            },
          }
        : undefined

      // markArea 配置：粗粒度下使用半透明色块，隐藏色块内文字标签
      const markAreaOpt = annotations.markArea
        ? { silent: false, data: annotations.markArea.data, tooltip: { show: true }, label: { show: false } }
        : undefined

      // tooltip formatter：解析告警标注的增强 name 格式，展示完整告警信息
      const tooltipFormatter = (params: any) => {
        const items = Array.isArray(params) ? params : [params]
        // 检测 markLine/markArea 数据项
        const markItems = items.filter((p: any) =>
          p.componentType === 'markLine' || p.componentType === 'markArea'
        )
        if (markItems.length > 0) {
          return markItems.map((m: any) => {
            const nameStr = m.name ?? ''
            const parts = nameStr.split('|')
            const prefix = parts[0]

            if (prefix === 'threshold_capped') {
              // 封顶阈值线：threshold_capped|op|actualValue|cappedY|level
              const op = parts[1] ?? ''
              const actualValue = parts[2] ?? ''
              const level = parts[4] ?? ''
              const levelLabel = level === 'critical' ? '严重' : level === 'warning' ? '预警' : '信息'
              return `<b>预警阈值</b><br/>条件：${op} ${actualValue}<br/>告警级别：${levelLabel}<br/><span style="color:#999">（阈值超出图表显示范围）</span>`
            }

            if (prefix === 'threshold') {
              // 阈值水平线：threshold|op|value|level
              const op = parts[1] ?? ''
              const value = parts[2] ?? ''
              const level = parts[3] ?? ''
              const levelLabel = level === 'critical' ? '严重' : level === 'warning' ? '预警' : '信息'
              return `<b>预警阈值</b><br/>条件：${op} ${value}<br/>告警级别：${levelLabel}`
            }

            if (prefix === 'threshold_zone') {
              // 告警区：threshold_zone|actualThreshold|level
              const actualThreshold = parts[1] ?? ''
              const level = parts[2] ?? ''
              const levelLabel = level === 'critical' ? '严重' : level === 'warning' ? '预警' : '信息'
              return `<b>阈值超出范围区域</b><br/>阈值：${actualThreshold}<br/>告警级别：${levelLabel}`
            }

            if (prefix === 'event') {
              // 告警竖线：event|label|ts|triggerValue|threshold|level
              const label = parts[1] ?? ''
              const ts = Number(parts[2] ?? 0)
              const triggerValue = Number(parts[3] ?? 0)
              const threshold = Number(parts[4] ?? 0)
              const level = parts[5] ?? ''
              const levelLabel = level === 'critical' ? '严重' : level === 'warning' ? '预警' : '信息'
              let html = `<b>${label}</b><br/>时间：${dayjs(ts).format('YYYY-MM-DD HH:mm:ss')}`
              if (triggerValue > 0) html += `<br/>触发值：${triggerValue}`
              if (threshold > 0) html += `<br/>阈值：${threshold}`
              if (level) html += `<br/>告警级别：${levelLabel}`
              return html
            }

            if (prefix === 'period') {
              // 告警色块：period|label|startTs|endTs|level
              const label = parts[1] ?? ''
              const startTs = Number(parts[2] ?? 0)
              const endTs = Number(parts[3] ?? 0)
              const level = parts[4] ?? ''
              const durationMs = endTs - startTs
              const duration = durationMs >= 3600000
                ? `${Math.round(durationMs / 3600000)}小时`
                : `${Math.round(durationMs / 60000)}分钟`
              const levelLabel = level === 'critical' ? '严重' : level === 'warning' ? '预警' : '信息'
              let html = `<b>${label}</b><br/>开始：${dayjs(startTs).format('YYYY-MM-DD HH:mm')}<br/>结束：${dayjs(endTs).format('YYYY-MM-DD HH:mm')}`
              if (durationMs > 0) html += `<br/>持续时间：${duration}`
              if (level) html += `<br/>告警级别：${levelLabel}`
              return html
            }

            // 兜底：无法识别的格式直接显示原文
            return `<b>${nameStr}</b>`
          }).join('<br/><br/>')
        }
        // 正常数据 tooltip
        const dataItems = items.filter((p: any) => p.seriesType === 'line' || p.seriesType === 'bar')
        return dataItems.map((p: any) =>
          `${p.marker} ${dayjs(p.value[0]).format('YYYY-MM-DD HH:mm')}: ${p.value[1]}`
        ).join('<br/>')
      }

      // 1m 数据量大时启用 dataZoom 滑块导航
      const needDataZoom = bucket === '1m' && chartData.length > 120
      const allZero = chartData.every((d) => d[1] === 0)

      // 有告警标注时使用 item 触发：悬停 markLine/markArea 直接展示告警信息
      // 无告警标注时使用 axis 触发：保留流畅的轴十字线数据探索体验
      const hasAlertAnnotations = !!markLineOpt || !!markAreaOpt
      const tooltipTrigger = hasAlertAnnotations ? 'item' : 'axis'

      return {
        tooltip: {
          trigger: tooltipTrigger,
          formatter: tooltipFormatter,
          confine: true,
          ...(hasAlertAnnotations ? { axisPointer: { type: 'cross', snap: true } } : {}),
        },
        // 告警标注动画：立即显示，不随数据逐点动画
        animationDuration: hasAlertAnnotations ? 200 : 800,
        animationDurationUpdate: hasAlertAnnotations ? 200 : 300,
        animationEasing: 'cubicOut',
        dataZoom: needDataZoom ? [
          { type: 'inside', start: 0, end: 100, minValueSpan: 5 * 60 * 1000 },
          { type: 'slider', start: 0, end: 100, minValueSpan: 5 * 60 * 1000, height: 20, bottom: 5 },
        ] : undefined,
        xAxis: {
          type: 'time',
          min: xAxisMin,
          max: xAxisMax,
          axisLabel: {
            formatter: axisLabelFormatter,
            hideOverlap: true,
          },
        },
        yAxis: {
          type: 'value',
          min: 0,
          ...(allZero ? { max: 1 } : yAxisMax ? { max: yAxisMax, ...(isExtremeRatio ? { splitNumber: 5 } : {}) } : {}),
        },
        series: [{
          type: panel.chart_type === 'bar' ? 'bar' : 'line',
          data: chartData,
          smooth: panel.chart_type === 'line' && bucket !== '1m',
          connectNulls: panel.chart_type === 'line',
          showSymbol: bucket !== '1m' || chartData.length <= 60,
          symbolSize: bucket === '1m' ? 2 : (chartData.length <= 10 ? 8 : 4),
          lineStyle: { width: bucket === '1m' ? 1 : (chartData.length <= 10 ? 3 : 2) },
          barMaxWidth: 32,
          markLine: markLineOpt,
          markArea: markAreaOpt,
        }],
      }
    }

    // bar 无时间维度（group_by 场景）
    const xData = rows.map((r) => String(Object.values(r)[1] ?? ''))
    return {
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: xData },
      yAxis: { type: 'value' },
      series: [{
        type: 'bar',
        data: rows.map((r) => Number(r.value ?? 0)),
      }],
    }
  }

  if (!dashboard) return <Spin style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>{dashboard.name}</h2>
        <Space>
          {isEditing ? (
            <>
              <Button onClick={cancelEdit}>取消</Button>
              <Button type="primary" icon={<SaveOutlined />} onClick={saveEdit}>保存布局</Button>
            </>
          ) : (
            <>
              <Button icon={<EditOutlined />} onClick={startEdit}>编辑布局</Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>添加 Panel</Button>
            </>
          )}
        </Space>
      </div>

      {/* 全局时间范围选择器 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space size={8} wrap align="center">
          <ClockCircleOutlined style={{ color: '#666' }} />
          <span style={{ fontSize: 13, fontWeight: 500 }}>时间范围：</span>
          <Segmented
            value={dashboardTimeRange.type === 'relative' ? dashboardTimeRange.value ?? '1d' : '__absolute__'}
            options={[
              ...QUICK_TIME_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
              { value: '__absolute__', label: '自定义范围' },
            ]}
            onChange={(val) => {
              if (val === '__absolute__') {
                setAbsolutePickerOpen(true)
              } else {
                setDashboardTimeRange({ type: 'relative', value: String(val) })
              }
            }}
          />
          {absolutePickerOpen && (
            <DatePicker.RangePicker
              showTime
              size="small"
              style={{ width: 360 }}
              defaultValue={
                dashboardTimeRange.type === 'absolute' && dashboardTimeRange.start && dashboardTimeRange.end
                  ? [dayjs(dashboardTimeRange.start), dayjs(dashboardTimeRange.end)]
                  : undefined
              }
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  setDashboardTimeRange({
                    type: 'absolute',
                    start: dates[0].format('YYYY-MM-DD HH:mm:ss'),
                    end: dates[1].format('YYYY-MM-DD HH:mm:ss'),
                  })
                }
                setAbsolutePickerOpen(false)
              }}
              onOpenChange={(open) => { if (!open) setAbsolutePickerOpen(false) }}
            />
          )}
          {dashboardTimeRange.type === 'absolute' && dashboardTimeRange.start && dashboardTimeRange.end && !absolutePickerOpen && (
            <Tag
              color="blue"
              closable
              style={{ cursor: 'pointer' }}
              onClose={(e) => { e.preventDefault(); setDashboardTimeRange({ type: 'relative', value: '1d' }) }}
              onClick={() => setAbsolutePickerOpen(true)}
            >
              {dayjs(dashboardTimeRange.start).format('YYYY-MM-DD HH:mm')} ~ {dayjs(dashboardTimeRange.end).format('YYYY-MM-DD HH:mm')}
            </Tag>
          )}
        </Space>
      </Card>

      {dashboard.filters && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Space size={4} wrap>
            <span style={{ fontSize: 13, color: '#666' }}>数据范围：</span>
            <Tag color="blue">
              {resolveFilterNodeLabels(normalizeFilterTree(dashboard.filters), FIELD_LABELS)}
            </Tag>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => navigate('/dashboards')}
              style={{ marginLeft: 8 }}
            >
              修改筛选
            </Button>
          </Space>
        </Card>
      )}

      {(!dashboard.panels || dashboard.panels.length === 0) && (
        <Card>
          <Empty description="暂无 Panel，点击「添加 Panel」创建第一个图表" />
        </Card>
      )}

      {dashboard.panels && dashboard.panels.length > 0 && (
        <GridLayout
          className={`dashboard-grid${isEditing ? ' editing' : ''}`}
          layout={isEditing ? editLayout.map(item => ({ ...item })) : gridLayout.map(item => ({ ...item }))}
          cols={GRID_COLS}
          rowHeight={GRID_ROW_HEIGHT}
          isDraggable={isEditing}
          isResizable={isEditing}
          resizeHandles={['se']}
          draggableHandle=".ant-card-head"
          draggableCancel=".ant-btn,.ant-select,.ant-popover,.ant-picker,.ant-tag,.ant-card-extra,.ant-card-extra *"
          compactType="vertical"
          margin={GRID_MARGIN}
          onDragStop={isEditing ? handleEditLayoutChange : undefined}
          onResizeStop={isEditing ? handleEditLayoutChange : undefined}
        >
          {(dashboard.panels ?? []).map((panel) => {
            const currentLayoutItem = (isEditing ? editLayout : gridLayout).find((item) => item.i === String(panel.id))
            const chartAreaHeight = getChartAreaHeight(currentLayoutItem, panel.chart_type)
            const pd = panelData[panel.id]
            const rows = pd?.rows ?? []
            const as = alertStates[panel.id]
            const ar = alertRecords[panel.id] ?? []
            const effectiveTimeRange = getEffectiveTimeRange(panel.id)
            const cfgBucket = (panel.query_config as Record<string, unknown>)?.x_dimension as { bucket?: string } | undefined
            const currentBucket = cfgBucket?.bucket || '1m'
            const panelOverride = panelTimeOverrides[panel.id]
            const alertTag = panel.alert_config && as
              ? <Tag color={as.last_state === 'firing' ? 'red' : as.last_state === 'pending' ? 'gold' : as.last_state === 'resolved' ? 'green' : 'green'} style={{ marginLeft: 4 }}>
                  {as.last_state === 'firing' ? '告警中' : as.last_state === 'pending' ? '预警中' : as.last_state === 'resolved' ? '已恢复' : '正常'}
                </Tag>
              : null
            return (
              <div key={panel.id} style={isEditing ? { position: 'relative' } : undefined}>
                <Card
                  title={<span style={{ cursor: isEditing ? 'grab' : 'default' }}>{isEditing && <HolderOutlined style={{ marginRight: 6, fontSize: 12, color: '#999' }} />}{panel.name}{alertTag}</span>}
                  size="small"
                  style={{ height: '100%', overflow: 'hidden' }}
                  styles={{ body: { height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' } }}
                  extra={
                    <span style={{ display: 'flex', gap: 8 }}>
                      {!isEditing && (
                        <Button
                          size="small"
                          icon={<EditOutlined />}
                          onClick={() => openEditPanel(panel)}
                        />
                      )}
                      {!isEditing && (
                        <Button
                          size="small"
                          icon={<ReloadOutlined />}
                          onClick={() => { loadPanelData(panel, effectiveTimeRange); loadAlertState(panel, effectiveTimeRange) }}
                        />
                      )}
                      <Popconfirm
                        title="确认删除此 Panel？"
                        onConfirm={() => handleDeletePanel(dashboard.id, panel.id)}
                      >
                        <Button size="small" icon={<DeleteOutlined />} danger />
                      </Popconfirm>
                    </span>
                  }
                >
                {/* 面板独立时间范围选择器 */}
                {(panel.chart_type !== 'number') && (
                  <div style={{ marginBottom: 8, fontSize: 12, display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'wrap' }}>
                    <ClockCircleOutlined style={{ color: '#999', fontSize: 11 }} />
                    <Select
                      size="small"
                      variant="borderless"
                      value={panelOverride === null || panelOverride === undefined ? '__follow__' : panelOverride.type === 'relative' ? panelOverride.value ?? '1d' : '__absolute__'}
                      style={{ minWidth: 90 }}
                      options={[
                        { value: '__follow__', label: '跟随仪表盘' },
                        ...QUICK_TIME_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
                        { value: '__absolute__', label: '自定义' },
                      ]}
                      onChange={(val) => {
                        if (val === '__follow__') {
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: null }))
                          loadPanelData(panel, dashboardTimeRange)
                          loadAlertState(panel, dashboardTimeRange)
                        } else if (val === '__absolute__') {
                          // 切到自定义时不立即变更，等拖拽选区或手动输入
                        } else {
                          const newOverride: TimeRangeOverride = { type: 'relative', value: String(val) }
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: newOverride }))
                          loadPanelData(panel, newOverride)
                          loadAlertState(panel, newOverride)
                        }
                      }}
                    />
                    {panelOverride && panelOverride.type === 'absolute' && panelOverride.start && !panelPickerOpen[panel.id] && (
                      <Tag
                        color="blue"
                        closable
                        style={{ cursor: 'pointer' }}
                        onClose={(e) => {
                          e.preventDefault()
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: null }))
                          loadPanelData(panel, dashboardTimeRange)
                          loadAlertState(panel, dashboardTimeRange)
                        }}
                        onClick={() => setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: true }))}
                      >
                        {dayjs(panelOverride.start).format('MM-DD HH:mm')}~{dayjs(panelOverride.end ?? '').format('MM-DD HH:mm')}
                      </Tag>
                    )}
                    {panelPickerOpen[panel.id] && (
                      <DatePicker.RangePicker
                        showTime
                        size="small"
                        style={{ width: 280 }}
                        defaultValue={
                          panelOverride && panelOverride.type === 'absolute' && panelOverride.start && panelOverride.end
                            ? [dayjs(panelOverride.start), dayjs(panelOverride.end)]
                            : undefined
                        }
                        onChange={(dates) => {
                          if (dates && dates[0] && dates[1]) {
                            const newOverride: TimeRangeOverride = { type: 'absolute', start: dates[0].format('YYYY-MM-DD HH:mm:ss'), end: dates[1].format('YYYY-MM-DD HH:mm:ss') }
                            setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: newOverride }))
                            loadPanelData(panel, newOverride)
                            loadAlertState(panel, newOverride)
                          }
                          setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: false }))
                        }}
                        onOpenChange={(open) => { if (!open) setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: false })) }}
                      />
                    )}
                    <span style={{ color: '#999' }}>粒度: {currentBucket}</span>
                  </div>
                )}
                {/* 数值面板时间范围选择器 + 筛选条件 */}
                {panel.chart_type === 'number' && (
                  <div style={{ fontSize: 12, display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'wrap', marginBottom: 8 }}>
                    <ClockCircleOutlined style={{ color: '#999', fontSize: 11 }} />
                    <Select
                      size="small"
                      variant="borderless"
                      value={panelOverride === null || panelOverride === undefined ? '__follow__' : panelOverride.type === 'relative' ? panelOverride.value ?? '1d' : '__absolute__'}
                      style={{ minWidth: 80 }}
                      options={[
                        { value: '__follow__', label: '跟随仪表盘' },
                        ...QUICK_TIME_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
                        { value: '__absolute__', label: '自定义' },
                      ]}
                      onChange={(val) => {
                        if (val === '__follow__') {
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: null }))
                          loadPanelData(panel, dashboardTimeRange)
                        } else if (val === '__absolute__') {
                          // 切到自定义时不立即变更
                        } else {
                          const newOverride: TimeRangeOverride = { type: 'relative', value: String(val) }
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: newOverride }))
                          loadPanelData(panel, newOverride)
                        }
                      }}
                    />
                    {panelOverride && panelOverride.type === 'absolute' && panelOverride.start && !panelPickerOpen[panel.id] && (
                      <Tag
                        color="blue"
                        closable
                        style={{ cursor: 'pointer' }}
                        onClose={(e) => {
                          e.preventDefault()
                          setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: null }))
                          loadPanelData(panel, dashboardTimeRange)
                        }}
                        onClick={() => setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: true }))}
                      >
                        {dayjs(panelOverride.start).format('MM-DD HH:mm')}~{dayjs(panelOverride.end ?? '').format('MM-DD HH:mm')}
                      </Tag>
                    )}
                    {panelPickerOpen[panel.id] && (
                      <DatePicker.RangePicker
                        showTime
                        size="small"
                        style={{ width: 280 }}
                        defaultValue={
                          panelOverride && panelOverride.type === 'absolute' && panelOverride.start && panelOverride.end
                            ? [dayjs(panelOverride.start), dayjs(panelOverride.end)]
                            : undefined
                        }
                        onChange={(dates) => {
                          if (dates && dates[0] && dates[1]) {
                            const newOverride: TimeRangeOverride = { type: 'absolute', start: dates[0].format('YYYY-MM-DD HH:mm:ss'), end: dates[1].format('YYYY-MM-DD HH:mm:ss') }
                            setPanelTimeOverrides((prev) => ({ ...prev, [panel.id]: newOverride }))
                            loadPanelData(panel, newOverride)
                          }
                          setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: false }))
                        }}
                        onOpenChange={(open) => { if (!open) setPanelPickerOpen((prev) => ({ ...prev, [panel.id]: false })) }}
                      />
                    )}
                    <span style={{ color: '#999' }}>
                      <LockOutlined style={{ fontSize: 11 }} /> 1m
                    </span>
                  </div>
                )}
                {/* 数值面板：筛选条件展示（独立行，允许文本换行） */}
                {panel.chart_type === 'number' && (() => {
                  const rawTree = (panel.query_config as Record<string, unknown>)?.panel_filter_tree
                  const filterTree = rawTree ? normalizeFilterTree(rawTree) : null
                  if (!filterTree) return null
                  return (
                    <Tag color="blue" style={{ fontSize: 11, marginBottom: 8, whiteSpace: 'normal', wordBreak: 'break-word', maxWidth: '100%', display: 'block' }}>
                      {resolveFilterNodeLabels(filterTree, FIELD_LABELS)}
                    </Tag>
                  )
                })()}
                {/* 仅初次加载（无旧数据）时显示 Spin，刷新时保留图表避免闪动 */}
                {pd?.loading && rows.length === 0 && (
                  <div style={{ height: chartAreaHeight, display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 0 }}>
                    <Spin />
                  </div>
                )}
                {panel.chart_type === 'number' && rows.length > 0 && !pd?.loading && (
                  <div style={{ textAlign: 'center', padding: '8px 0', minHeight: chartAreaHeight, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <Statistic value={rows[0]?.value != null ? Number(rows[0].value) : '-'} valueStyle={{ fontSize: 32 }} />
                  </div>
                )}
                {panel.chart_type !== 'number' && rows.length > 0 && (
                  <div style={{ position: 'relative', minHeight: 0 }}>
                    {(panel.chart_type === 'line' || panel.chart_type === 'bar') ? (
                      <ChartWithDragSelect
                        option={buildChartOption(panel, rows, effectiveTimeRange, as, ar)}
                        onTimeRangeSelect={(timeRange) => {
                          setDashboardTimeRange(timeRange)
                        }}
                        style={{ height: chartAreaHeight }}
                      />
                    ) : (
                      <ReactECharts
                        option={buildChartOption(panel, rows, effectiveTimeRange, as, ar)}
                        notMerge={false}
                        style={{ height: chartAreaHeight }}
                      />
                    )}
                    {pd?.loading && (
                      <div style={{ position: 'absolute', top: 8, right: 8 }}>
                        <Spin size="small" />
                      </div>
                    )}
                  </div>
                )}
                {!pd?.loading && rows.length === 0 && (
                  <div style={{ height: chartAreaHeight, display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 0 }}>
                    <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                  </div>
                )}
              </Card>
            </div>
          )
        })}
      </GridLayout>
      )}

      {/* 创建面板 Modal */}
      <Modal
        title="添加 Panel"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); form.resetFields(); setCreateAlertConfig(null); setCreateQueryConfig(DEFAULT_QUERY_CONFIG) }}
        onOk={() => form.submit()}
        width={640}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleCreatePanel}>
          {templates.length > 0 && (
            <Form.Item label="从模板创建（可选）">
              <Select placeholder="选择预置模板" allowClear onChange={applyTemplate} style={{ width: '100%' }}>
                {templates.map((t, i) => (
                  <Select.Option key={i} value={String(i)}>
                    {String(t.name)}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
          )}
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="图表类型" name="chart_type" initialValue="line" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="line">折线图</Select.Option>
              <Select.Option value="bar">柱状图</Select.Option>
              <Select.Option value="pie">饼图</Select.Option>
              <Select.Option value="number">数值</Select.Option>
            </Select>
          </Form.Item>
          <QueryConfigEditor
            value={createQueryConfig}
            onChange={setCreateQueryConfig}
            chartType={createChartType}
          />
          <PanelAlertConfigEditor
            chartType={form.getFieldValue('chart_type') ?? 'line'}
            value={createAlertConfig}
            onChange={setCreateAlertConfig}
            channels={channels}
          />
        </Form>
      </Modal>

      {/* 编辑面板 Modal */}
      <Modal
        title="编辑 Panel"
        open={editModalOpen}
        onCancel={() => { setEditModalOpen(false); editForm.resetFields(); setEditAlertConfig(null); setEditQueryConfig({}); setEditingPanel(null) }}
        onOk={() => editForm.submit()}
        width={640}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" onFinish={handleEditPanel}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="图表类型" name="chart_type" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="line">折线图</Select.Option>
              <Select.Option value="bar">柱状图</Select.Option>
              <Select.Option value="pie">饼图</Select.Option>
              <Select.Option value="number">数值</Select.Option>
            </Select>
          </Form.Item>
          <QueryConfigEditor
            value={editQueryConfig}
            onChange={setEditQueryConfig}
            chartType={editChartType}
          />
          <PanelAlertConfigEditor
            chartType={editForm.getFieldValue('chart_type') ?? 'line'}
            value={editAlertConfig}
            onChange={setEditAlertConfig}
            channels={channels}
          />
        </Form>
      </Modal>
    </div>
  )
}