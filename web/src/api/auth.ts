import http from './http'

export interface LoginReq {
  username: string
  password: string
}

export interface LoginResp {
  token: string
  user: { id: number; username: string; role: string }
}

export const login = (data: LoginReq) =>
  http.post<{ data: LoginResp }>('/auth/login', data).then((r) => r.data.data)

export const logout = () => http.post('/auth/logout')

export const getMe = () =>
  http.get<{ data: { id: number; username: string; role: string } }>('/auth/me').then((r) => r.data.data)

export const getMyPermissions = () =>
  http.get<{ data: { permissions: Record<string, string[]> } }>('/auth/permissions').then((r) => r.data.data)
