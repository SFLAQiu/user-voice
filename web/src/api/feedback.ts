import http from './http'

export interface Feedback {
  id: number
  source_id: number
  original_id: string
  app_id: number
  app_name?: string
  platform?: string
  platform_id?: number | null
  user_id: string
  user_name: string
  user_mode: number | null
  content: string
  images: string[] | null
  videos: string[] | null
  phone_model: string
  app_version: string
  channel_id: string
  qq: string
  file_url: string
  category: string
  business_module: string
  category_confidence: number | null
  category_status: number
  sentiment: string
  classified_at: string | null
  original_created_at: string
  synced_at: string
  created_at: string
  updated_at: string
}

export interface FeedbackQuery {
  page?: number
  page_size?: number
  start_time?: string
  end_time?: string
  app_id?: string
  platform_id?: string
  app_version?: string
  user_mode?: string
  user_id?: string
  category?: string
  business_module?: string
  sentiment?: string
  keyword?: string
  order_by?: string
  order_dir?: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export const listFeedbacks = (q: FeedbackQuery) =>
  http
    .get<{ data: PageResult<Feedback> }>('/feedbacks', { params: q })
    .then((r) => r.data.data)

export const getFeedback = (id: number) =>
  http.get<{ data: Feedback }>(`/feedbacks/${id}`).then((r) => r.data.data)

export const updateCategory = (id: number, category: string, businessModule?: string) =>
  http.put(`/feedbacks/${id}/category`, { category, business_module: businessModule ?? '' })