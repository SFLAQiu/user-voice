import http from './http'

export interface ClassifierConfig {
  id: number
  type: string
  name: string
  content: string
  sort_order: number
  enabled: number
  is_default: number
  created_at: string
  updated_at: string
}

export interface ActiveConfig {
  prompt_template: string
  categories: string[]
  business_modules: { name: string; description: string }[]
  prompt_config_id: number
}

export interface ClassifierEnums {
  categories: string[]
  business_modules: string[]
}

export const listClassifierConfigs = () =>
  http.get<{ data: ClassifierConfig[] }>('/classifier-configs').then((r) => r.data.data)

export const getActiveConfig = () =>
  http.get<{ data: ActiveConfig }>('/classifier-configs/active').then((r) => r.data.data)

export const getClassifierEnums = () =>
  http.get<{ data: ClassifierEnums }>('/classifier-configs/enums').then((r) => r.data.data)

export const createClassifierConfig = (data: Partial<ClassifierConfig>) =>
  http.post<{ data: ClassifierConfig }>('/classifier-configs', data).then((r) => r.data.data)

export const updateClassifierConfig = (id: number, data: Partial<ClassifierConfig>) =>
  http.put<{ data: ClassifierConfig }>(`/classifier-configs/${id}`, data).then((r) => r.data.data)

export const deleteClassifierConfig = (id: number) =>
  http.delete(`/classifier-configs/${id}`)

export const listClassificationLogs = (feedbackId: number) =>
  http.get<{ data: ClassificationLog[] }>(`/feedbacks/${feedbackId}/classification-logs`).then((r) => r.data.data)

export interface ClassificationLog {
  id: number
  feedback_id: number
  attempt: number
  prompt_config_id: number | null
  provider_id: number | null
  provider_name: string | null
  prompt_used: string | null
  llm_raw_response: string | null
  parsed_result: string | null
  category_result: string | null
  module_result: string | null
  sentiment_result: string | null
  confidence_result: number | null
  error_message: string | null
  status: number
  duration_ms: number | null
  created_at: string
}