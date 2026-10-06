import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from './stores/auth'
import MainLayout from './layouts/MainLayout'
import LoginPage from './pages/login/Login'
import SetupWizard from './pages/setup/SetupWizard'
import FeedbackList from './pages/feedback/FeedbackList'
import DataSourceList from './pages/datasource/DataSourceList'
import EnumConfigList from './pages/enum-config/EnumConfigList'
import ClassifierConfigList from './pages/classifier-config/ClassifierConfigList'
import UserList from './pages/user/UserList'
import DashboardList from './pages/dashboard/DashboardList'
import DashboardView from './pages/dashboard/DashboardView'
import AlertPage from './pages/alert/AlertPage'
import NotificationChannelList from './pages/notification-channel/NotificationChannelList'
import MetricDimensionConfigPage from './pages/admin/MetricDimensionConfig'
import GuidePage from './pages/guide/GuidePage'
import PermissionGroupList from './pages/permission/PermissionGroupList'
import { getSetupStatus } from './api/setup'

// 未初始化时强制进入 /setup 向导（已在 /setup 则直接渲染，避免 Navigate 对相同路径无限循环导致白屏）
function RequireSetup({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<'loading' | 'uninit' | 'ready'>('loading')
  const location = useLocation()
  useEffect(() => {
    getSetupStatus()
      .then((s) => setState(s.initialized ? 'ready' : 'uninit'))
      .catch(() => setState('ready')) // 接口异常不阻塞使用（如后端暂不可达）
  }, [])
  // 状态未知时不渲染业务树：若放行，带 token 的页面会挂载并发请求，
  // setup 模式后端无业务路由 → 404 吐司。多一帧白屏换取不闪错误页面。
  if (state === 'loading') return null
  if (state === 'uninit' && location.pathname !== '/setup') {
    return <Navigate to="/setup" replace />
  }
  return <>{children}</>
}

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

interface RequirePagePermissionProps {
  pageKey: string
  children: React.ReactNode
}

function RequirePagePermission({ pageKey, children }: RequirePagePermissionProps) {
  const role = useAuthStore((s) => s.role)
  const permissions = useAuthStore((s) => s.permissions)
  // 超管直接放行
  if (role === 'admin') return <>{children}</>
  // 权限已加载则按权限校验
  if (permissions) {
    if (permissions[pageKey]?.includes('view')) return <>{children}</>
    return <Navigate to="/feedback" replace />
  }
  // 权限未加载时放行（兼容加载中状态）
  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      {/* 未初始化时任何路径都引导到 /setup 完成配置 */}
      <RequireSetup>
        <Routes>
          <Route path="/setup" element={<SetupWizard />} />
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <RequireAuth>
                <MainLayout />
              </RequireAuth>
            }
          >
            <Route index element={<Navigate to="/feedback" replace />} />
            <Route path="feedback" element={<RequirePagePermission pageKey="feedback"><FeedbackList /></RequirePagePermission>} />
            <Route path="dashboards" element={<RequirePagePermission pageKey="dashboard"><DashboardList /></RequirePagePermission>} />
            <Route path="dashboards/:id" element={<RequirePagePermission pageKey="dashboard"><DashboardView /></RequirePagePermission>} />
            <Route path="alert" element={<RequirePagePermission pageKey="alert"><AlertPage /></RequirePagePermission>} />
            <Route path="notification-channels" element={<RequirePagePermission pageKey="notification_channels"><NotificationChannelList /></RequirePagePermission>} />
            <Route path="datasources" element={<RequirePagePermission pageKey="datasources"><DataSourceList /></RequirePagePermission>} />
            <Route path="enum-configs" element={<RequirePagePermission pageKey="enum_configs"><EnumConfigList /></RequirePagePermission>} />
            <Route path="classifier-configs" element={<RequirePagePermission pageKey="classifier_configs"><ClassifierConfigList /></RequirePagePermission>} />
            <Route path="metric-dimensions" element={<RequirePagePermission pageKey="metric_dimensions"><MetricDimensionConfigPage /></RequirePagePermission>} />
            <Route path="permission-groups" element={<RequirePagePermission pageKey="users"><PermissionGroupList /></RequirePagePermission>} />
            <Route path="users" element={<RequirePagePermission pageKey="users"><UserList /></RequirePagePermission>} />
            <Route path="guide" element={<RequirePagePermission pageKey="guide"><GuidePage /></RequirePagePermission>} />
          </Route>
        </Routes>
      </RequireSetup>
    </BrowserRouter>
  )
}
