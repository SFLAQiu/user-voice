import http from './http'

export interface User {
  id: number
  username: string
  role: string
  status: number
  last_login_at: string | null
  created_at: string
}

export interface CreateUserForm {
  username: string
  password: string
  role: string
  group_ids?: number[]
}

export const listUsers = () =>
  http.get<{ data: User[] }>('/users').then((r) => r.data.data)

export const createUser = (data: CreateUserForm) =>
  http.post<{ data: User }>('/users', data).then((r) => r.data.data)

export const setUserStatus = (id: number, status: number) =>
  http.put(`/users/${id}/status`, { status })

export const getPermissions = (userId: number) =>
  http.get<{ data: any }>(`/users/${userId}/permissions`).then((r) => r.data.data)

export const getGroups = (userId: number) =>
  http.get<{ data: any[] }>(`/users/${userId}/groups`).then((r) => r.data.data)

export const setGroups = (userId: number, groupIds: number[]) =>
  http.put(`/users/${userId}/groups`, { group_ids: groupIds })

export const getOverrides = (userId: number) =>
  http.get<{ data: any[] }>(`/users/${userId}/overrides`).then((r) => r.data.data)

export const setOverrides = (userId: number, overrides: any[]) =>
  http.put(`/users/${userId}/overrides`, { overrides })
