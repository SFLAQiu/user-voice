import http from './http'

export interface PermissionGroup {
  id: number
  name: string
  description?: string
  permissions: Record<string, string[]>
  created_at: string
  updated_at: string
}

export interface UserPermissionOverride {
  user_id: number
  page_key: string
  perm_key: string
  grant_type: 0 | 1
}

export interface UserPermissions {
  groups: PermissionGroup[]
  effective: Record<string, string[]>
  overrides: UserPermissionOverride[]
}

// 权限组 CRUD
export const listPermissionGroups = () =>
  http.get<{ data: PermissionGroup[] }>('/permission-groups').then((r) => r.data.data)

export const getPermissionGroup = (id: number) =>
  http.get<{ data: PermissionGroup }>(`/permission-groups/${id}`).then((r) => r.data.data)

export const createPermissionGroup = (data: {
  name: string
  description?: string
  permissions: Record<string, string[]>
}) =>
  http.post<{ data: PermissionGroup }>('/permission-groups', data).then((r) => r.data.data)

export const updatePermissionGroup = (
  id: number,
  data: { name: string; description?: string; permissions: Record<string, string[]> },
) =>
  http.put<{ data: PermissionGroup }>(`/permission-groups/${id}`, data).then((r) => r.data.data)

export const deletePermissionGroup = (id: number) =>
  http.delete(`/permission-groups/${id}`)

// 用户权限管理
export const getUserPermissions = (userId: number) =>
  http.get<{ data: UserPermissions }>(`/users/${userId}/permissions`).then((r) => r.data.data)

export const getUserGroups = (userId: number) =>
  http.get<{ data: PermissionGroup[] }>(`/users/${userId}/groups`).then((r) => r.data.data)

export const setUserGroups = (userId: number, groupIds: number[]) =>
  http.put(`/users/${userId}/groups`, { group_ids: groupIds })

export const getUserOverrides = (userId: number) =>
  http.get<{ data: UserPermissionOverride[] }>(`/users/${userId}/overrides`).then((r) => r.data.data)

export const setUserOverrides = (userId: number, overrides: UserPermissionOverride[]) =>
  http.put(`/users/${userId}/overrides`, { overrides })
