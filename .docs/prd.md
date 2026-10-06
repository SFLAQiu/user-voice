# 倾听用户反馈 PRD

## 1. 产品概述

### 1.1 产品名称
倾听用户反馈（Feedback Intelligence Platform）

### 1.2 产品定位
面向内部运营团队的用户反馈数据智能管理与分析平台，提供反馈数据的自动采集、智能分类、多维度可视化分析及告警通知能力。

### 1.3 背景与痛点
| 痛点 | 描述 |
|------|------|
| 分类缺失 | 当前仅展示原始反馈列表，无法对反馈内容按问题类型精确归类 |
| 分析不足 | 无多维度图表展示，无法直观了解反馈趋势和分布 |
| 告警缺失 | 无法对关注指标设置阈值和告警通知 |
| 数据孤岛 | 仅支持单一接口数据源，缺乏可扩展的多数据源支持 |

### 1.4 目标用户
- 产品运营人员
- 产品经理
- 客服团队
- 技术支持团队

---

## 2. 功能需求

### 2.1 用户认证模块

#### 2.1.1 账号密码登录
- 支持用户名 + 密码登录
- 密码加密存储（bcrypt）
- 登录态基于 JWT Token，有效期可配置
- 登录失败次数限制（5次/5分钟，基于内存或数据库计数）
- 支持退出登录

#### 2.1.2 用户管理
- 管理员可创建/禁用账号
- 支持角色权限：管理员、普通用户（只读）

---

### 2.2 数据源管理模块

#### 2.2.1 多数据源支持
- 支持配置多个数据源，每个数据源独立配置拉取策略
- 数据源类型可扩展，当前需支持：**HTTP API 数据源**

#### 2.2.2 HTTP API 数据源
- 配置项：
  - 数据源名称
  - 请求 URL（支持分页参数模板）
  - 请求方法（GET/POST）
  - 请求头（Headers）
  - Cookie
  - 数据字段映射（将接口字段映射到统一元数据结构）
- 支持定时拉取（cron 表达式配置）
- 增量同步：基于时间戳或 ID 去重，仅同步新增数据

#### 2.2.3 当前数据源实例
- URL: `https://demo.example.com/api/feedback_list`
- 参数：`is_ajax=1&app_id={app_id}&platform_id={platform_id}&to_page={page}`
- 认证方式：Cookie
- 分页遍历直到无新增数据

---

### 2.3 反馈元数据管理模块

#### 2.3.1 统一元数据结构

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 反馈唯一ID |
| source | string | 数据来源标识 |
| app_id | int | 应用ID |
| app_name | string | 应用名称 |
| platform | string | 平台（iOS/Android/HarmonyOS） |
| user_id | string | 用户ID |
| user_name | string | 用户昵称 |
| user_mode | int | 用户模式（游客/注册/VIP 等，按业务自定义） |
| content | text | 反馈内容 |
| images | json | 图片URL列表 |
| phone_model | string | 手机型号 |
| app_version | string | APP版本 |
| channel_id | string | 渠道ID |
| category | string | 智能分类结果 |
| category_confidence | float | 分类置信度 |
| sentiment | string | 情感倾向（正面/中性/负面） |
| created_at | datetime | 反馈时间 |
| synced_at | datetime | 同步入库时间 |

#### 2.3.2 智能分类（LLM）

**分类维度：**
- 功能问题（Bug/闪退/功能异常）
- 体验问题（交互不便/UI问题）
- 内容问题（广告过多/内容质量）
- 商业化问题（收费/会员/红包）
- 数据问题（数据丢失/不准确）
- 兼容性问题（系统兼容/设备适配）
- 账号问题（登录/注册/账号安全）
- 建议需求（功能建议/优化建议）
- 其他

**实现方案：**
- 新增反馈入库后异步触发 LLM 分类
- 调用大模型 API（支持配置 OpenAI/Claude/国产大模型）
- Prompt 模板可配置，支持自定义分类体系
- 分类结果写入元数据 category 字段
- 同时输出情感倾向（sentiment）
- 分类失败时标记为"待分类"，支持人工修正

---

### 2.4 反馈列表管理模块

#### 2.4.1 列表展示字段
- 反馈ID、用户昵称、反馈内容（截断展示）、智能分类、情感倾向、平台、APP版本、手机型号、反馈时间

#### 2.4.2 筛选条件
| 筛选项 | 类型 | 说明 |
|--------|------|------|
| 时间范围 | date range | 反馈创建时间 |
| 应用 | select | app_id 对应应用 |
| 平台 | multi-select | iOS/Android/HarmonyOS |
| 智能分类 | multi-select | 分类结果 |
| 情感倾向 | select | 正面/中性/负面 |
| APP版本 | select | 版本号 |
| 用户模式 | multi-select | 游客/注册用户/VIP 用户 |
| 关键词 | text | 全文搜索反馈内容 |

#### 2.4.3 排序支持
- 反馈时间（默认倒序）
- 反馈ID

#### 2.4.4 详情查看
- 点击列表项查看完整反馈内容
- 展示图片附件
- 展示分类详情与置信度
- 支持人工修正分类

---

### 2.5 数据可视化 Dashboard 模块

#### 2.5.1 Dashboard 管理
- 支持创建多个 Dashboard
- 每个 Dashboard 可自定义名称和描述
- 支持 Dashboard 布局编辑（拖拽排列图表面板）

#### 2.5.2 图表面板（Panel）
- 每个 Panel 可独立配置：
  - 图表类型：折线图、柱状图、饼图、表格、数字卡片
  - 数据查询：选择维度和指标
  - 时间范围：相对时间（最近1小时/24小时/7天/30天）或绝对时间
  - 刷新频率：自动刷新间隔

#### 2.5.3 预置图表维度

| 图表 | 类型 | 维度 | 指标 |
|------|------|------|------|
| 反馈趋势 | 折线图 | 时间（小时/天） | 反馈数量 |
| 分类分布 | 饼图 | 智能分类 | 反馈数量占比 |
| 平台分布 | 柱状图 | 平台 | 反馈数量 |
| 版本分布 | 柱状图 | APP版本 | 反馈数量 |
| 情感趋势 | 堆叠面积图 | 时间 | 各情感类型数量 |
| 分类趋势 | 堆叠柱状图 | 时间 | 各分类数量 |
| Top问题 | 表格 | 分类 | 数量/占比 |
| 反馈总量 | 数字卡片 | - | 总量/日增量 |

#### 2.5.4 自定义图表
- 用户可新增自定义 Panel
- 选择 X 轴维度、Y 轴指标、分组维度
- 支持过滤条件叠加

---

### 2.6 告警模块

#### 2.6.1 告警规则配置
- 基于图表指标设置阈值告警
- 告警规则配置项：
  - 规则名称
  - 关联指标（如：某分类反馈数/总反馈数/负面反馈数）
  - 时间窗口（如：最近1小时/最近24小时）
  - 触发条件（大于/小于/等于 阈值）
  - 告警级别（警告/严重/紧急）
  - 沉默期（同一规则触发后多长时间内不重复告警）

#### 2.6.2 告警通知渠道
- 支持配置企业通讯 APP 群机器人：
  - 企业微信群机器人（Webhook）
  - 飞书群机器人（Webhook）
  - 钉钉群机器人（Webhook）
- 每条告警规则可绑定多个通知渠道

#### 2.6.3 告警文案模板
- 支持自定义告警文案模板
- 模板变量：
  - `{{rule_name}}` - 规则名称
  - `{{metric_name}}` - 指标名称
  - `{{current_value}}` - 当前值
  - `{{threshold}}` - 阈值
  - `{{level}}` - 告警级别
  - `{{time_window}}` - 时间窗口
  - `{{trigger_time}}` - 触发时间
  - `{{detail_url}}` - 详情链接

**默认告警模板示例：**
```
⚠️ 【{{level}}】{{rule_name}}
指标：{{metric_name}}
当前值：{{current_value}}（阈值：{{threshold}}）
时间窗口：{{time_window}}
触发时间：{{trigger_time}}
详情：{{detail_url}}
```

#### 2.6.4 告警策略
- 连续触发升级：同一规则连续触发 N 次自动升级告警级别
- 告警恢复通知：指标恢复正常时发送恢复通知
- 告警静默：支持手动静默指定时间段

---

## 3. 非功能需求

### 3.1 性能
- 列表查询响应 < 500ms（万级数据量）
- Dashboard 图表加载 < 2s
- 数据同步延迟 < 5min（从数据源产生到入库）

### 3.2 可用性
- 系统可用性 > 99%
- 数据不丢失（同步失败自动重试）

### 3.3 安全
- 接口鉴权（JWT）
- 敏感数据脱敏展示（用户QQ等）
- 操作日志审计
- 数据源凭证加密存储

### 3.4 可扩展性
- 数据源类型可插件化扩展
- 分类模型可替换
- 告警渠道可扩展

---

## 4. 技术方案概要

### 4.1 技术选型建议

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端 | Go (Gin/Echo) | 高性能 HTTP 服务 |
| 数据库 | MySQL 8.0 | 元数据存储 |
| 定时任务 | cron | 数据拉取调度 |
| LLM | OpenAI API / 兼容接口 | 智能分类 |
| 前端 | React + Ant Design + ECharts | 管理后台与图表 |
| 告警 | Webhook | 企业通讯机器人推送 |

### 4.2 系统架构

```
┌─────────────────────────────────────────────────────┐
│                   前端 (React)                        │
│   登录 │ 反馈列表 │ Dashboard │ 告警配置 │ 数据源配置  │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP API
┌──────────────────────▼──────────────────────────────┐
│                  后端 API 服务 (Go)                    │
│  Auth │ Feedback │ Dashboard │ Alert │ DataSource     │
└──┬───────────┬──────────┬───────────┬───────────────┘
   │           │          │           │
   ▼           ▼          ▼           ▼
┌────────┐  ┌────────┐ ┌──────────────┐
│MySQL 8.0│  │LLM API │ │ 数据源 (HTTP) │
└────────┘  └────────┘ └──────────────┘
                                       │
                          ┌────────────▼────────────┐
                          │   定时任务 (Cron Worker)   │
                          │  增量拉取 → 入库 → 分类    │
                          └─────────────────────────┘
                                       │
                          ┌────────────▼────────────┐
                          │     告警引擎 (Alert)      │
                          │  指标计算 → 阈值判断 → 通知 │
                          └─────────────────────────┘
```

---

## 5. 数据模型

### 5.1 核心表

```sql
-- 用户表
users (id, username, password_hash, role, status, created_at)

-- 数据源配置表
data_sources (id, name, type, config_json, sync_cron, last_sync_at, status, created_at)

-- 反馈元数据表
feedbacks (id, source_id, original_id, app_id, app_name, platform, user_id, user_name, user_mode, content, images, phone_model, app_version, channel_id, category, category_confidence, sentiment, original_created_at, synced_at, created_at)

-- Dashboard 表
dashboards (id, name, description, layout_json, created_by, created_at, updated_at)

-- 图表面板表
panels (id, dashboard_id, name, chart_type, query_config_json, position_json, created_at)

-- 告警规则表
alert_rules (id, name, metric_query, time_window, condition, threshold, level, silence_minutes, status, created_at)

-- 告警通知渠道表
alert_channels (id, name, type, webhook_url, created_at)

-- 告警规则-渠道关联表
alert_rule_channels (rule_id, channel_id)

-- 告警模板表
alert_templates (id, name, content_template, created_at)

-- 告警记录表
alert_records (id, rule_id, trigger_value, level, notified_channels, triggered_at, resolved_at)
```

---

## 6. 接口设计（核心）

### 6.1 认证
- `POST /api/auth/login` - 登录
- `POST /api/auth/logout` - 退出
- `GET /api/auth/me` - 获取当前用户信息

### 6.2 反馈管理
- `GET /api/feedbacks` - 反馈列表（分页、筛选、排序）
- `GET /api/feedbacks/:id` - 反馈详情
- `PUT /api/feedbacks/:id/category` - 人工修正分类

### 6.3 Dashboard
- `GET /api/dashboards` - Dashboard 列表
- `POST /api/dashboards` - 创建 Dashboard
- `PUT /api/dashboards/:id` - 更新 Dashboard
- `DELETE /api/dashboards/:id` - 删除 Dashboard
- `GET /api/dashboards/:id/panels` - 获取面板列表
- `POST /api/dashboards/:id/panels` - 添加面板
- `PUT /api/panels/:id` - 更新面板
- `DELETE /api/panels/:id` - 删除面板
- `POST /api/panels/:id/query` - 查询面板数据

### 6.4 告警
- `GET /api/alert-rules` - 告警规则列表
- `POST /api/alert-rules` - 创建告警规则
- `PUT /api/alert-rules/:id` - 更新规则
- `DELETE /api/alert-rules/:id` - 删除规则
- `GET /api/alert-channels` - 通知渠道列表
- `POST /api/alert-channels` - 添加通知渠道
- `GET /api/alert-records` - 告警记录

### 6.5 数据源
- `GET /api/data-sources` - 数据源列表
- `POST /api/data-sources` - 添加数据源
- `PUT /api/data-sources/:id` - 更新数据源
- `POST /api/data-sources/:id/sync` - 手动触发同步

---

## 7. 里程碑计划

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| P0 | 用户登录 + 数据源配置 + 数据拉取入库 + 反馈列表 | 高 |
| P1 | LLM 智能分类 + 分类筛选 | 高 |
| P2 | Dashboard 可视化（预置图表） | 中 |
| P3 | 自定义图表 + 告警规则配置 + 通知推送 | 中 |
| P4 | 多数据源扩展 + 告警策略增强 | 低 |

---

## 8. 风险与依赖

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| LLM API 不稳定 | 分类延迟或失败 | 异步重试 + 人工兜底 |
| 数据源接口变更 | 同步失败 | 监控 + 告警 + 字段映射可配置 |
| Cookie 过期 | 数据拉取中断 | 监控同步状态 + 支持手动更新凭证 |
| 反馈量暴增 | 分类积压 | 队列削峰 + 批量分类 |
