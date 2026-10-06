import http from './http'

export interface EnumConfig {
  id: number
  field: string
  labels: Record<string, string>
  created_at: string
  updated_at: string
}

export interface EnumConfigForm {
  field: string
  labels: Record<string, string>
}

export const listEnumConfigs = () =>
  http.get<{ data: EnumConfig[] }>('/enum-configs').then((r) => r.data.data)

export const createEnumConfig = (data: EnumConfigForm) =>
  http.post<{ data: EnumConfig }>('/enum-configs', data).then((r) => r.data.data)

export const updateEnumConfig = (id: number, data: Partial<EnumConfigForm>) =>
  http.put<{ data: EnumConfig }>(`/enum-configs/${id}`, data).then((r) => r.data.data)

export const deleteEnumConfig = (id: number) =>
  http.delete(`/enum-configs/${id}`)

export const aggregatedEnums = () =>
  http.get<{ data: Record<string, Record<string, string>> }>('/enum-configs/enums').then((r) => r.data.data)