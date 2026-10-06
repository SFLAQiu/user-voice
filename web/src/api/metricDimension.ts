import http from './http'

export interface MetricDimensionConfig {
  id: number
  field: string
  label: string
  data_type: string
  default_value: string
  status: number
  sort_order: number
  created_at: string
  updated_at: string
}

export interface AddDimensionInput {
  field: string
  label: string
  data_type: string
  default_value?: string
}

export interface RebuildStatus {
  state: 'idle' | 'running' | 'completed' | 'failed'
  version: number
  started_at?: string | null
  message?: string
}

export const listMetricDimensions = () =>
  http.get<{ data: MetricDimensionConfig[] }>('/metric-dimensions').then((r) => r.data.data)

export const addMetricDimension = (data: AddDimensionInput) =>
  http.post<{ data: { message: string } }>('/metric-dimensions', data).then((r) => r.data.data)

export const removeMetricDimension = (field: string) =>
  http.delete<{ data: { message: string } }>(`/metric-dimensions/${field}`).then((r) => r.data.data)

export const getRebuildStatus = () =>
  http.get<{ data: RebuildStatus }>('/metric-dimensions/rebuild-status').then((r) => r.data.data)

export const triggerBackfill = () =>
  http.post<{ data: { message: string } }>('/metric-dimensions/backfill').then((r) => r.data.data)

export interface MetricBucketUpdateLog {
  id: number
  op_type: string
  version: number
  affected_rows: number
  duration_ms: number
  status: string
  message?: string
  triggered_by?: string
  created_at: string
}

export const listBucketUpdateLogs = (limit?: number) =>
  http.get<{ data: MetricBucketUpdateLog[] }>('/metric-dimensions/update-logs', { params: { limit } }).then((r) => r.data.data)