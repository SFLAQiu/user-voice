import http from './http'
import { FilterNode } from '../types/filterNode'
import type { TimeRangeOverride } from '../utils/timeRangeUtils'
import { flattenTimeRange } from '../utils/timeRangeUtils'

export interface PanelAlertConfig {
  threshold: number
  condition_op: string
  level: string
  silence_minutes: number
  eval_interval_sec: number
  pending_duration_sec: number
  time_window_sec: number
  reduce_mode: string // max | avg | last | sum，默认 max
  channel_ids: number[]
}

export interface Panel {
  id: number
  dashboard_id: number
  name: string
  chart_type: string // line | bar | pie | number
  query_config: Record<string, unknown>
  position: Record<string, unknown>
  refresh_seconds: number
  alert_config?: PanelAlertConfig | null
}

export interface AlertRuleStateBrief {
  rule_id: number
  last_state: string
  threshold: number
  condition_op: string
  level: string
  pending_duration_sec: number
  last_eval_at: string | null
  consecutive_fires: number
  reduce_mode: string // max | avg | last | sum
}

export interface PanelWithAlertState extends Panel {
  alert_state?: AlertRuleStateBrief | null
}

export interface Dashboard {
  id: number
  name: string
  description: string
  layout: Record<string, unknown>
  filters?: FilterNode
  created_by: number
  created_at: string
  panels?: Panel[]
}

export const listDashboards = () =>
  http.get<{ data: Dashboard[] }>('/dashboards').then((r) => r.data.data)

export const getDashboard = (id: number) =>
  http.get<{ data: Dashboard }>(`/dashboards/${id}`).then((r) => r.data.data)

export const createDashboard = (data: Partial<Dashboard>) =>
  http.post<{ data: Dashboard }>('/dashboards', data).then((r) => r.data.data)

export const updateDashboard = (id: number, data: Partial<Dashboard> & { rebuild_alerts?: boolean }) =>
  http.put<{ data: Dashboard }>(`/dashboards/${id}`, data).then((r) => r.data.data)

export const deleteDashboard = (id: number) => http.delete(`/dashboards/${id}`)

export const getDeleteInfo = (id: number) =>
  http.get<{ data: { panel_count: number; alert_rule_count: number } }>(`/dashboards/${id}/delete-info`).then((r) => r.data.data)

export const hasPanelAlertRules = (id: number) =>
  http.get<{ data: { has_alert_rules: boolean } }>(`/dashboards/${id}/has-alert-rules`).then((r) => r.data.data.has_alert_rules)

export const createPanel = (dashboardId: number, data: Partial<Panel>) =>
  http.post<{ data: Panel }>(`/dashboards/${dashboardId}/panels`, data).then((r) => r.data.data)

export const updatePanel = (dashboardId: number, panelId: number, data: Partial<Panel>) =>
  http.put<{ data: Panel }>(`/dashboards/${dashboardId}/panels/${panelId}`, data).then((r) => r.data.data)

export const deletePanel = (dashboardId: number, panelId: number) =>
  http.delete(`/dashboards/${dashboardId}/panels/${panelId}`)

export const queryPanel = (panelId: number, timeRangeOverride?: TimeRangeOverride) =>
  http
    .get<{ data: Array<Record<string, unknown>> }>(`/panels/${panelId}/query`, {
      params: timeRangeOverride ? flattenTimeRange(timeRangeOverride) : undefined,
    })
    .then((r) => r.data.data)

export const getPanelTemplates = () =>
  http.get<{ data: Array<Record<string, unknown>> }>('/panels/templates').then((r) => r.data.data)

export const getPanelAlertState = (panelId: number) =>
  http.get<{ data: PanelWithAlertState }>(`/panels/${panelId}/alert-state`).then((r) => r.data.data)