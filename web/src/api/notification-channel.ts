import http from './http'

export interface NotificationChannel {
  id: number
  name: string
  type: string
  webhook_url: string
  secret?: string
  status: number
}

export const listNotificationChannels = () =>
  http.get<{ data: NotificationChannel[] }>('/notification-channels').then((r) => r.data.data)

export const createNotificationChannel = (data: Partial<NotificationChannel>) =>
  http.post<{ data: NotificationChannel }>('/notification-channels', data).then((r) => r.data.data)

export const updateNotificationChannel = (id: number, data: Partial<NotificationChannel>) =>
  http.put<{ data: NotificationChannel }>(`/notification-channels/${id}`, data).then((r) => r.data.data)

export const deleteNotificationChannel = (id: number) => http.delete(`/notification-channels/${id}`)

export const testNotificationChannel = (id: number, msg?: string) =>
  http.post(`/notification-channels/${id}/test`, { message: msg })