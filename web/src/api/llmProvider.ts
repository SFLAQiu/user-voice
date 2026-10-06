import http from './http'

export interface LLMProvider {
  id: number
  name: string
  base_url: string
  api_key: string // always "***" in responses
  model: string
  timeout_ms: number
  priority: number
  status: number
  created_at: string
  updated_at: string
}

export interface CBStatus {
  provider_id: number
  provider_name: string
  state: 'closed' | 'open' | 'half_open'
  consecutive_failures: number
  total_failures: number
  total_successes: number
  requests: number
}

export interface LLMConfig {
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

export interface CircuitBreakerConfig {
  failure_threshold: number
  failure_rate_threshold: number
  window_seconds: number
  cooldown_seconds: number
  min_requests: number
}

export interface RetryConfig {
  max_retries: number
  strategy: string
  timeout_ms: number
}

export interface TestConnectionInput {
  base_url: string
  api_key: string
  model: string
  timeout_ms?: number
}

export interface TestConnectionResult {
  success: boolean
  message: string
  latency_ms: number
  model: string
}

export const listLLMProviders = () =>
  http.get<{ data: LLMProvider[] }>('/llm-providers').then((r) => r.data.data)

export const getLLMProvider = (id: number) =>
  http.get<{ data: LLMProvider }>(`/llm-providers/${id}`).then((r) => r.data.data)

export const createLLMProvider = (data: Partial<LLMProvider>) =>
  http.post<{ data: LLMProvider }>('/llm-providers', data).then((r) => r.data.data)

export const updateLLMProvider = (id: number, data: Partial<LLMProvider>) =>
  http.put<{ data: LLMProvider }>(`/llm-providers/${id}`, data).then((r) => r.data.data)

export const deleteLLMProvider = (id: number) =>
  http.delete(`/llm-providers/${id}`)

export const getCircuitBreakerStatus = () =>
  http.get<{ data: CBStatus[] }>('/llm-providers/status').then((r) => r.data.data)

export const listLLMConfigs = () =>
  http.get<{ data: LLMConfig[] }>('/llm-configs').then((r) => r.data.data)

export const createLLMConfig = (data: Partial<LLMConfig>) =>
  http.post<{ data: LLMConfig }>('/llm-configs', data).then((r) => r.data.data)

export const updateLLMConfig = (id: number, data: Partial<LLMConfig>) =>
  http.put<{ data: LLMConfig }>(`/llm-configs/${id}`, data).then((r) => r.data.data)

export const deleteLLMConfig = (id: number) =>
  http.delete(`/llm-configs/${id}`)

export const getCircuitBreakerConfig = () =>
  http.get<{ data: CircuitBreakerConfig }>('/llm-configs/circuit-breaker').then((r) => r.data.data)

export const getRetryConfig = () =>
  http.get<{ data: RetryConfig }>('/llm-configs/retry').then((r) => r.data.data)

export const testNewLLMProvider = (data: TestConnectionInput) =>
  http.post<{ data: TestConnectionResult }>('/llm-providers/test', data).then((r) => r.data.data)

export const testExistingLLMProvider = (id: number) =>
  http.post<{ data: TestConnectionResult }>(`/llm-providers/${id}/test`).then((r) => r.data.data)