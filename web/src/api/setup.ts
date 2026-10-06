import http from './http'

export interface SetupStatus {
  initialized: boolean
}

export interface DbReq {
  host: string
  port: number
  user: string
  password: string
  name: string
}

export interface FinishReq extends DbReq {
  jwt_secret: string
  encryption_key: string
  admin_username: string
  admin_password: string
}

export const getSetupStatus = () =>
  http.get<{ data: SetupStatus }>('/setup/status').then((r) => r.data.data)

export const testDb = (data: DbReq) =>
  http.post<{ data: { ok: boolean; message: string } }>('/setup/test-db', data).then((r) => r.data.data)

export const genSecrets = () =>
  http.get<{ data: { jwt_secret: string; encryption_key: string } }>('/setup/secrets').then((r) => r.data.data)

export const finishSetup = (data: FinishReq) =>
  http.post<{ data: { config_path: string; message: string } }>('/setup/finish', data).then((r) => r.data.data)
