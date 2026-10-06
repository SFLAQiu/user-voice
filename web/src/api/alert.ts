import http from './http'

export interface AlertRule {
  id: number
  name: string
  panel_id?: number | null
  panel_name?: string
  dashboard_id?: number
  dashboard_name?: string
  metric_query: Record<string, unknown>
  time_window_sec: number
  condition_op: string
  threshold: number
  level: string
  silence_minutes: number
  eval_interval_sec: number
  pending_duration_sec: number
  reduce_mode: string
  channel_ids: number[]
  status: number
  last_eval_at: string | null
  last_state: string
  consecutive_fires: number
}


export interface AlertRecord {
  id: number
  rule_id: number
  trigger_value: number
  threshold: number
  level: string
  state: string
  notified_channels: number[]
  notify_status: string
  triggered_at: string
  metric_timestamp?: string | null
  resolved_at: string | null
}

export const listAlertRules = (params?: { type?: 'custom' | 'panel' }) =>
  http.get<{ data: AlertRule[] }>('/alert/rules', { params }).then((r) => r.data.data)

export const createAlertRule = (data: Partial<AlertRule>) =>
  http.post<{ data: AlertRule }>('/alert/rules', data).then((r) => r.data.data)

export const updateAlertRule = (id: number, data: Partial<AlertRule>) =>
  http.put<{ data: AlertRule }>(`/alert/rules/${id}`, data).then((r) => r.data.data)

export const toggleAlertRuleStatus = (id: number, status: number) =>
  http.put<{ data: AlertRule }>(`/alert/rules/${id}/status`, { status }).then((r) => r.data.data)

export const deleteAlertRule = (id: number) => http.delete(`/alert/rules/${id}`)

export const setRuleChannels = (id: number, channelIds: number[]) =>
  http.put(`/alert/rules/${id}/channels`, { channel_ids: channelIds })


export const listAlertRecords = (params?: { rule_id?: number; panel_id?: number; limit?: number; start_at?: string; end_at?: string }) =>
  http.get<{ data: AlertRecord[] }>('/alert/records', { params }).then((r) => r.data.data)