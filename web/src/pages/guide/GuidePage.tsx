import { Card, Row, Col, Typography, Steps, Divider, Tag, theme } from 'antd'
import {
  MessageOutlined,
  DashboardOutlined,
  BellOutlined,
  RobotOutlined,
  ApiOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  ClusterOutlined,
  RocketOutlined,
  BulbOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../../stores/auth'
import { useNavigate } from 'react-router-dom'

const { Title, Paragraph, Text } = Typography

const features = [
  {
    icon: <ApiOutlined style={{ fontSize: 36, color: '#1677ff' }} />,
    title: '多渠道数据接入',
    desc: '通过 HTTP API 插件自动拉取外部平台的用户反馈，支持分页、字段映射、Cookie 解密；同时支持 MySQL 数据库同步，按已存在 ID 早停避免重复入库。',
    color: '#e6f4ff',
  },
  {
    icon: <RobotOutlined style={{ fontSize: 36, color: '#722ed1' }} />,
    title: 'LLM 智能分类归因',
    desc: '多 Provider 并发分类 + 熔断器 + 指数退避重试。可配置 Prompt 模板，自动解析 LLM 输出为结构化标签。',
    color: '#f9f0ff',
  },
  {
    icon: <DashboardOutlined style={{ fontSize: 36, color: '#13c2c2' }} />,
    title: '可视化仪表盘',
    desc: '拖拽式网格布局编辑，支持数值面板嵌套筛选条件与预聚合求和查询，实时展示反馈趋势与分布。',
    color: '#e6fffb',
  },
  {
    icon: <BellOutlined style={{ fontSize: 36, color: '#fa8c16' }} />,
    title: '智能告警通知',
    desc: '周期性规则评估 + Webhook 推送，支持企业微信、飞书、钉钉。多实例部署通过 MySQL GET_LOCK 互斥锁保证单次触发。',
    color: '#fff7e6',
  },
  {
    icon: <SettingOutlined style={{ fontSize: 36, color: '#52c41a' }} />,
    title: '灵活配置管理',
    desc: '枚举字段、分类器参数、LLM Provider、指标维度均可通过界面配置，无需重启服务即可生效。',
    color: '#f6ffed',
  },
  {
    icon: <ThunderboltOutlined style={{ fontSize: 36, color: '#eb2f96' }} />,
    title: '单进程一体化',
    desc: '单个 Go 进程同时运行 API 服务器 + 调度器 + 分类器 + 告警引擎，部署运维简单。',
    color: '#fff0f6',
  },
]

const archLayers = [
  { label: 'HTTP / Gin', desc: 'Handler 层', color: '#1677ff', items: ['JWT 认证', 'RBAC 鉴权', '审计日志', 'CORS'] },
  { label: 'Service', desc: '业务逻辑层', color: '#722ed1', items: ['数据源编排', '分类调度', '反馈查询', '告警评估'] },
  { label: 'Repository', desc: '数据访问层', color: '#13c2c2', items: ['GORM 动态查询', 'Upsert', 'Ngram 全文检索'] },
  { label: 'Infrastructure', desc: '基础设施', color: '#fa8c16', items: ['MySQL', 'Cron 调度', 'Worker Pool', 'AES-GCM'] },
]

export default function GuidePage() {
  const role = useAuthStore((s) => s.role)
  const navigate = useNavigate()
  const { token } = theme.useToken()

  return (
    <div style={{ maxWidth: 960, margin: '0 auto' }}>
      {/* Hero */}
      <div style={{ textAlign: 'center', marginBottom: 48 }}>
        <Title level={2} style={{ marginBottom: 8 }}>
          <BulbOutlined style={{ marginRight: 12, color: token.colorPrimary }} />
          倾听用户反馈
        </Title>
        <Paragraph type="secondary" style={{ fontSize: 15, maxWidth: 600, margin: '0 auto' }}>
          自动采集 · 智能分类 · 可视化洞察 · 实时告警 —— 让每一条反馈驱动产品改进
        </Paragraph>
      </div>

      {/* 核心功能 */}
      <Title level={4} style={{ marginBottom: 20 }}>
        <ThunderboltOutlined style={{ marginRight: 8 }} />
        核心功能
      </Title>
      <Row gutter={[16, 16]} style={{ marginBottom: 48 }}>
        {features.map((f) => (
          <Col xs={24} sm={12} md={8} key={f.title}>
            <Card
              hoverable
              style={{ height: '100%', borderRadius: 8, borderTop: `3px solid ${f.icon.props.style.color}` }}
              styles={{ body: { padding: 24 } }}
            >
              <div
                style={{
                  width: 64,
                  height: 64,
                  borderRadius: 16,
                  background: f.color,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginBottom: 16,
                }}
              >
                {f.icon}
              </div>
              <Title level={5} style={{ marginBottom: 8 }}>{f.title}</Title>
              <Paragraph type="secondary" style={{ fontSize: 13, marginBottom: 0, lineHeight: 1.7 }}>
                {f.desc}
              </Paragraph>
            </Card>
          </Col>
        ))}
      </Row>

      <Divider />

      {/* 快速上手 */}
      <Title level={4} style={{ marginBottom: 20 }}>
        <RocketOutlined style={{ marginRight: 8 }} />
        快速上手
      </Title>
      <Card style={{ marginBottom: 48, borderRadius: 8 }}>
        <Steps
          direction="vertical"
          current={-1}
          items={[
            {
              title: '配置数据源',
              description: (
                <span>
                  在「<a onClick={() => navigate('/datasources')}>数据源管理</a>」中添加 HTTP API 数据源，填写接口地址、分页参数、字段映射及 Cron 同步表达式，保存后即可手动触发首次拉取或等待定时执行。
                </span>
              ),
            },
            {
              title: '查看反馈列表',
              description: (
                <span>
                  同步完成后进入「<a onClick={() => navigate('/feedback')}>反馈列表</a>」查看采集到的反馈，支持关键词搜索、分类筛选、情感标签过滤，点击行可展开详情抽屉。
                </span>
              ),
            },
            {
              title: '启用智能分类',
              description: (
                <span>
                  在「<a onClick={() => navigate('/classifier-configs')}>智能配置</a>」中创建分类器，选择 LLM Provider、配置 Prompt 模板与分类维度，系统将自动对未分类反馈进行批量标注。
                </span>
              ),
            },
            {
              title: '搭建仪表盘',
              description: (
                <span>
                  进入「<a onClick={() => navigate('/dashboards')}>仪表盘</a>」创建 Dashboard，添加数值面板并配置筛选条件与聚合维度，拖拽调整布局后即可获得实时数据看板。
                </span>
              ),
            },
            {
              title: '设置告警规则',
              description: (
                <span>
                  前往「<a onClick={() => navigate('/alert')}>告警管理</a>」创建规则，定义评估周期与触发条件，绑定「<a onClick={() => navigate('/notification-channels')}>通知渠道</a>」后系统将在条件满足时自动推送消息。
                </span>
              ),
            },
          ]}
        />
      </Card>

      <Divider />

      {/* 系统架构 */}
      <Title level={4} style={{ marginBottom: 20 }}>
        <ClusterOutlined style={{ marginRight: 8 }} />
        系统架构
      </Title>

      {/* 架构分层图 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        {archLayers.map((layer) => (
          <Col xs={24} sm={12} md={6} key={layer.label}>
            <Card
              size="small"
              style={{ borderRadius: 8, borderTop: `3px solid ${layer.color}` }}
              styles={{ body: { padding: '16px 12px' } }}
            >
              <Text strong style={{ color: layer.color, fontSize: 13 }}>{layer.label}</Text>
              <div style={{ fontSize: 12, color: token.colorTextSecondary, marginBottom: 8 }}>{layer.desc}</div>
              {layer.items.map((item) => (
                <Tag key={item} style={{ marginBottom: 4, fontSize: 11 }}>{item}</Tag>
              ))}
            </Card>
          </Col>
        ))}
      </Row>

      {/* 架构流程示意 */}
      <Card style={{ marginBottom: 48, borderRadius: 8 }} styles={{ body: { padding: 32 } }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexWrap: 'wrap',
            gap: 8,
            fontSize: 13,
          }}
        >
          {[
            { label: '外部 API', icon: '🌐', color: '#e6f4ff' },
            { label: '数据源插件', icon: '📥', color: '#fff7e6' },
            { label: 'GORM / MySQL', icon: '💾', color: '#f6ffed' },
            { label: 'LLM 分类器', icon: '🤖', color: '#f9f0ff' },
            { label: '告警引擎', icon: '🔔', color: '#fff0f6' },
            { label: 'Webhook', icon: '📤', color: '#e6fffb' },
          ].map((step, i, arr) => (
            <div key={step.label} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div
                style={{
                  background: step.color,
                  padding: '10px 16px',
                  borderRadius: 8,
                  textAlign: 'center',
                  whiteSpace: 'nowrap',
                }}
              >
                <div style={{ fontSize: 20 }}>{step.icon}</div>
                <Text style={{ fontSize: 12 }}>{step.label}</Text>
              </div>
              {i < arr.length - 1 && (
                <Text type="secondary" style={{ fontSize: 18 }}>→</Text>
              )}
            </div>
          ))}
        </div>
      </Card>

      {/* 技术栈 */}
      <Title level={5} style={{ marginBottom: 12 }}>
        技术栈
      </Title>
      <Row gutter={[16, 16]} style={{ marginBottom: 48 }}>
        {[
          { label: '后端', tags: ['Go', 'Gin', 'GORM', 'MySQL', 'robfig/cron'] },
          { label: '前端', tags: ['React 18', 'TypeScript', 'Ant Design 5', 'ECharts', 'Zustand'] },
          { label: 'AI', tags: ['多 LLM Provider', '熔断器', '指数退避', '结构化 Prompt'] },
        ].map((stack) => (
          <Col xs={24} sm={8} key={stack.label}>
            <Card size="small" title={stack.label} style={{ borderRadius: 8 }}>
              {stack.tags.map((t) => (
                <Tag key={t} color="blue" style={{ marginBottom: 4 }}>{t}</Tag>
              ))}
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  )
}
