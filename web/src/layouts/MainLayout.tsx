import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Button, Avatar, Dropdown, Typography, theme } from 'antd'
import {
  MessageOutlined,
  ApiOutlined,
  UserOutlined,
  LogoutOutlined,
  DashboardOutlined,
  BellOutlined,
  SendOutlined,
  SettingOutlined,
  RobotOutlined,
  AppstoreOutlined,
  BookOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/auth'
import { logout } from '../api/auth'

const { Sider, Header, Content } = Layout

interface NavItem {
  key: string
  icon: React.ReactNode
  label: string
  permissionKey?: string
}

const menuItems: NavItem[] = [
  { key: '/feedback', icon: <MessageOutlined />, label: '反馈列表', permissionKey: 'feedback' },
  { key: '/dashboards', icon: <DashboardOutlined />, label: '仪表盘', permissionKey: 'dashboard' },
  { key: '/alert', icon: <BellOutlined />, label: '告警管理', permissionKey: 'alert' },
  { key: '/notification-channels', icon: <SendOutlined />, label: '通知渠道', permissionKey: 'notification_channels' },
  { key: '/datasources', icon: <ApiOutlined />, label: '数据源管理', permissionKey: 'datasources' },
  { key: '/classifier-configs', icon: <RobotOutlined />, label: '智能配置', permissionKey: 'classifier_configs' },
  { key: '/enum-configs', icon: <SettingOutlined />, label: '枚举配置', permissionKey: 'enum_configs' },
  { key: '/metric-dimensions', icon: <AppstoreOutlined />, label: '维度配置', permissionKey: 'metric_dimensions' },
  { key: '/permission-groups', icon: <SafetyCertificateOutlined />, label: '权限组管理', permissionKey: 'users' },
  { key: '/users', icon: <UserOutlined />, label: '用户管理', permissionKey: 'users' },
  { key: '/guide', icon: <BookOutlined />, label: '功能指南', permissionKey: 'guide' },
]

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { username, role, permissions, logout: storeLogout } = useAuthStore()
  const { token } = theme.useToken()

  const handleLogout = async () => {
    await logout().catch(() => {})
    storeLogout()
    navigate('/login')
  }

  // 基于 permissions 过滤菜单；若未加载则 fallback 到 role 检查
  const visibleMenu = (() => {
    if (permissions) {
      return menuItems.filter((m) => {
        if (!m.permissionKey) return true
        return permissions[m.permissionKey]?.includes('view')
      })
    }
    // fallback: admin 可见全部，viewer 不可见用户管理
    return role === 'admin' ? menuItems : menuItems.filter((m) => m.key !== '/users')
  })()

  return (
    <Layout style={{ height: '100vh', overflow: 'hidden' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed} theme="dark">
        <div
          style={{
            height: 48,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '0 16px',
            gap: 8,
          }}
        >
          <img
            src="/public/logo.svg"
            alt="logo"
            style={{ width: 28, height: 28, flexShrink: 0 }}
          />
          {!collapsed && (
            <Typography.Text
              strong
              style={{ color: '#fff', fontSize: 14, whiteSpace: 'nowrap', overflow: 'hidden' }}
            >
              倾听反馈
            </Typography.Text>
          )}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={visibleMenu as any}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout style={{ overflow: 'hidden' }}>
        <Header
          style={{
            background: token.colorBgContainer,
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 12,
            boxShadow: '0 1px 4px rgba(0,0,0,.08)',
            position: 'sticky',
            top: 0,
            zIndex: 1,
            flexShrink: 0,
          }}
        >
          <Dropdown
            menu={{
              items: [
                {
                  key: 'logout',
                  icon: <LogoutOutlined />,
                  label: '退出登录',
                  onClick: handleLogout,
                },
              ],
            }}
          >
            <Button type="text" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <span>{username}</span>
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, overflow: 'auto', flex: 1, minHeight: 0 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
