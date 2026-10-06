import http from './http'

export interface DataSource {
  id: number
  name: string
  type: string
  sync_cron: string
  status: number
  last_sync_at: string | null
  last_sync_error: string | null
  config: Record<string, unknown>
}

export interface DataSourceForm {
  name: string
  type: string
  sync_cron: string
  config: Record<string, unknown>
}

export type DataSourceType = 'http_api' | 'mysql'

export interface HttpApiConfig {
  endpoint: string
  method?: string
  headers?: Record<string, string>
  cookie?: string
  params_template?: Record<string, string>
  data_path?: string
  page_paginate?: { param: string; start: number }
  field_mapping?: Record<string, string>
  time_layout?: string
  max_pages?: number
  stop_when_seen?: boolean
  iterate?: unknown[]
  enums?: Record<string, Record<string, string>>
}

export const DATA_SOURCE_TYPE_OPTIONS: { value: DataSourceType; label: string; disabled: boolean; tooltip?: string }[] = [
  { value: 'http_api', label: '从 API 同步', disabled: false },
  { value: 'mysql', label: '从 MySQL 同步', disabled: false },
]

export interface MysqlConfig {
  host: string
  port?: number
  username: string
  password: string
  database: string
  table: string
  time_field: string
  batch_limit?: number
  field_mapping: Record<string, string>
  enums?: Record<string, Record<string, string>>
}

export const DEFAULT_MYSQL_CONFIG: MysqlConfig = {
  host: '',
  port: 3306,
  username: '',
  password: '',
  database: '',
  table: '',
  time_field: 'created_at',
  batch_limit: 1000,
  field_mapping: {
    original_id: '',
    content: '',
    original_created_at: '',
    user_id: '',
    user_name: '',
    phone_model: '',
    app_version: '',
  },
}

export const DEFAULT_MYSQL_CONFIG_JSON = JSON.stringify(DEFAULT_MYSQL_CONFIG, null, 2)

export interface SyncLog {
  id: number
  source_id: number
  status: string
  started_at: string
  finished_at: string | null
  fetched_count: number
  inserted_count: number
  error_message: string | null
  created_at: string
}

export const listDataSources = () =>
  http.get<{ data: DataSource[] }>('/data-sources').then((r) => r.data.data)

export const getDataSource = (id: number) =>
  http.get<{ data: DataSource }>(`/data-sources/${id}`).then((r) => r.data.data)

export const createDataSource = (data: DataSourceForm) =>
  http.post<{ data: DataSource }>('/data-sources', data).then((r) => r.data.data)

export const updateDataSource = (id: number, data: DataSourceForm) =>
  http.put<{ data: DataSource }>(`/data-sources/${id}`, data).then((r) => r.data.data)

export const deleteDataSource = (id: number) =>
  http.delete(`/data-sources/${id}`)

export const syncDataSource = (id: number) =>
  http.post<{ data: { inserted: number } }>(`/data-sources/${id}/sync`).then((r) => r.data.data)

export const listSyncLogs = (id: number, limit = 20) =>
  http.get<{ data: SyncLog[] }>(`/data-sources/${id}/sync-logs`, { params: { limit } }).then((r) => r.data.data)

