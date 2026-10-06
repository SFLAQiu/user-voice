import { useEffect, useState } from 'react'
import { Modal, Tabs, Select, Button, message, Space, Tag, Spin } from 'antd'
import {
  getUserPermissions,
  listPermissionGroups,
  setUserGroups,
  setUserOverrides,
  type PermissionGroup,
  type UserPermissionOverride,
} from '../../api/permission'

const PAGE_LABELS: Record<string, string> = {
  feedback: '反馈列表',
  dashboard: '仪表盘',
  alert: '告警管理',
  notification_channels: '通知渠道',
  datasources: '数据源管理',
  enum_configs: '枚举配置',
  classifier_configs: '智能配置',
  metric_dimensions: '维度配置',
  llm_providers: 'LLM 提供商',
  users: '用户管理',
  guide: '功能指南',
}

const PERM_LABELS: Record<string, string> = {
  view: '查看',
  edit: '编辑',
}

const PAGE_KEYS = Object.keys(PAGE_LABELS)

interface Props {
  userId: number
  username: string
  open: boolean
  onClose: () => void
}

export default function UserPermissionModal({ userId, username, open, onClose }: Props) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [allGroups, setAllGroups] = useState<PermissionGroup[]>([])
  const [assignedGroupIds, setAssignedGroupIds] = useState<number[]>([])
  const [effective, setEffective] = useState<Record<string, string[]>>({})
  const [overrides, setOverrides] = useState<UserPermissionOverride[]>([])

  useEffect(() => {
    if (!open) return
    setLoading(true)
    Promise.all([
      listPermissionGroups(),
      getUserPermissions(userId),
    ])
      .then(([groups, perms]) => {
        setAllGroups(groups)
        setAssignedGroupIds(perms.groups.map((g) => g.id))
        setEffective(perms.effective)
        setOverrides(perms.overrides)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [userId, open])

  const handleSaveGroups = async () => {
    setSaving(true)
    try {
      await setUserGroups(userId, assignedGroupIds)
      message.success('分组已更新')
      // 重新加载有效权限
      const perms = await getUserPermissions(userId)
      setEffective(perms.effective)
    } catch {
      // error handled by interceptor
    } finally {
      setSaving(false)
    }
  }

  const handleOverrideChange = (pageKey: string, permKey: string, grantType: 0 | 1 | null) => {
    setOverrides((prev) => {
      const filtered = prev.filter((o) => !(o.page_key === pageKey && o.perm_key === permKey))
      if (grantType !== null) {
        return [...filtered, { user_id: userId, page_key: pageKey, perm_key: permKey, grant_type: grantType }]
      }
      return filtered
    })
  }

  const handleSaveOverrides = async () => {
    setSaving(true)
    try {
      await setUserOverrides(userId, overrides)
      message.success('覆盖已更新')
      const perms = await getUserPermissions(userId)
      setEffective(perms.effective)
    } catch {
      // error handled by interceptor
    } finally {
      setSaving(false)
    }
  }

  const getOverrideState = (pageKey: string, permKey: string): 0 | 1 | null => {
    const found = overrides.find((o) => o.page_key === pageKey && o.perm_key === permKey)
    return found ? (found.grant_type === 1 ? 1 : 0) : null
  }

  const permissionTab = (
    <div style={{ maxHeight: 400, overflow: 'auto' }}>
      {PAGE_KEYS.map((page) => {
        const pageLabel = PAGE_LABELS[page] || page
        const hasView = effective[page]?.includes('view')
        const hasEdit = effective[page]?.includes('edit')
        const perms = page === 'guide' ? ['view'] : ['view', 'edit']

        return (
          <div
            key={page}
            style={{
              marginBottom: 6,
              padding: '8px 12px',
              border: '1px solid #f0f0f0',
              borderRadius: 6,
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 6 }}>{pageLabel}</div>
            <Space size="large">
              {perms.map((pk) => {
                const state = getOverrideState(page, pk)
                const inherited = effective[page]?.includes(pk) && state === null
                const overridden = state !== null
                return (
                  <div key={pk} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 13, color: '#666', width: 32 }}>{PERM_LABELS[pk] || pk}</span>
                    <Select
                      size="small"
                      style={{ width: 100 }}
                      value={overridden ? (state === 1 ? 'grant' : 'deny') : 'inherit'}
                      onChange={(val) => {
                        if (val === 'inherit') {
                          handleOverrideChange(page, pk, null)
                        } else if (val === 'grant') {
                          handleOverrideChange(page, pk, 1)
                        } else {
                          handleOverrideChange(page, pk, 0)
                        }
                      }}
                    >
                      <Select.Option value="inherit">
                        {inherited ? '继承(有)' : '继承(无)'}
                      </Select.Option>
                      <Select.Option value="grant">强制授权</Select.Option>
                      <Select.Option value="deny">强制拒绝</Select.Option>
                    </Select>
                    {overridden && (
                      <Tag color={state === 1 ? 'green' : 'red'} style={{ margin: 0 }}>
                        {state === 1 ? '授权' : '拒绝'}
                      </Tag>
                    )}
                  </div>
                )
              })}
            </Space>
          </div>
        )
      })}
    </div>
  )

  return (
    <Modal
      title={`权限管理 - ${username}`}
      open={open}
      onCancel={onClose}
      footer={null}
      width={680}
      destroyOnClose
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}>
          <Spin />
        </div>
      ) : (
        <Tabs
          items={[
            {
              key: 'groups',
              label: '分组分配',
              children: (
                <div>
                  <Select
                    mode="multiple"
                    style={{ width: '100%' }}
                    placeholder="选择权限组"
                    value={assignedGroupIds}
                    onChange={setAssignedGroupIds}
                    options={allGroups.map((g) => ({
                      label: g.name,
                      value: g.id,
                    }))}
                  />
                  <Button
                    type="primary"
                    size="small"
                    loading={saving}
                    onClick={handleSaveGroups}
                    style={{ marginTop: 12 }}
                  >
                    保存分组
                  </Button>
                </div>
              ),
            },
            {
              key: 'overrides',
              label: '权限覆盖',
              children: (
                <div>
                  {permissionTab}
                  <Button
                    type="primary"
                    size="small"
                    loading={saving}
                    onClick={handleSaveOverrides}
                    style={{ marginTop: 12 }}
                  >
                    保存覆盖
                  </Button>
                </div>
              ),
            },
          ]}
        />
      )}
    </Modal>
  )
}
