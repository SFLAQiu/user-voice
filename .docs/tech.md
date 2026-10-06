# 倾听用户反馈 技术方案

## 1. 技术栈总览

| 层级 | 技术选型 | 版本 | 说明 |
|------|---------|------|------|
| 后端语言 | Go | 1.22+ | 主开发语言 |
| Web 框架 | Gin | v1.10+ | HTTP 路由与中间件 |
| ORM | GORM | v2 | MySQL 操作 |
| 数据库 | MySQL | 8.0 | 元数据存储（含 JSON 字段） |
| 配置管理 | Viper | v1 | 多源配置（yaml/env） |
| 日志 | zap | v1 | 结构化日志 |
| 定时任务 | robfig/cron | v3 | 数据拉取调度 |
| 任务队列 | 内置 channel + worker pool | - | 异步分类、告警评估 |
| 认证 | golang-jwt/jwt | v5 | JWT Token |
| 密码哈希 | bcrypt | - | 密码存储 |
| 参数校验 | go-playground/validator | v10 | 入参校验 |
| HTTP 客户端 | resty | v2 | 数据源拉取 |
| LLM SDK | sashabaranov/go-openai | - | OpenAI 兼容协议 |
| 前端框架 | React | 18 | UI 框架 |
| 前端构建 | Vite | 5 | 构建工具 |
| UI 组件库 | Ant Design | 5 | 后台组件 |
| 图表库 | ECharts (echarts-for-react) | 5 | Dashboard 图表 |
| 路由 | React Router | 6 | 前端路由 |
| 状态管理 | Zustand | - | 轻量状态管理 |
| HTTP 请求 | axios | - | API 调用 |
| 容器化 | Docker + docker-compose | - | 本地/部署 |

---

## 2. 项目结构

### 2.1 后端目录结构（单仓库模块化）

```
feedback/
├── cmd/
│   ├── api/                      # API 服务入口
│   │   └── main.go
│   └── worker/                   # 定时任务/告警引擎入口（可与 api 同进程或分离）
│       └── main.go
├── internal/
│   ├── config/                   # 配置加载
│   ├── model/                    # GORM 实体定义
│   │   ├── user.go
│   │   ├── feedback.go
│   │   ├── data_source.go
│   │   ├── dashboard.go
│   │   ├── panel.go
│   │   ├── alert.go
│   │   └── audit_log.go
│   ├── repository/               # 数据访问层
│   │   ├── user_repo.go
│   │   ├── feedback_repo.go
│   │   └── ...
│   ├── service/                  # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── feedback_service.go
│   │   ├── dashboard_service.go
│   │   ├── alert_service.go
│   │   └── datasource_service.go
│   ├── handler/                  # HTTP Handler
│   │   ├── auth_handler.go
│   │   ├── feedback_handler.go
│   │   └── ...
│   ├── middleware/               # 中间件
│   │   ├── jwt.go
│   │   ├── rbac.go
│   │   ├── rate_limit.go
│   │   ├── audit.go
│   │   └── recovery.go
│   ├── router/                   # 路由注册
│   │   └── router.go
│   ├── datasource/               # 数据源插件
│   │   ├── plugin.go             # 插件接口
│   │   ├── registry.go           # 插件注册表
│   │   └── http_api/             # HTTP API 数据源实现
│   │       ├── plugin.go
│   │       └── parser.go
│   ├── classifier/               # 智能分类
│   │   ├── classifier.go         # 分类器接口
│   │   ├── llm_classifier.go     # LLM 实现
│   │   └── prompt.go
│   ├── alert/                    # 告警引擎
│   │   ├── engine.go             # 规则评估引擎
│   │   ├── evaluator.go          # 指标查询求值
│   │   ├── template.go           # 文案渲染
│   │   └── notifier/             # 通知渠道
│   │       ├── notifier.go       # 接口
│   │       ├── wecom.go          # 企业微信
│   │       ├── feishu.go         # 飞书
│   │       └── dingtalk.go       # 钉钉
│   ├── scheduler/                # 调度器
│   │   ├── cron.go               # cron 调度
│   │   └── job.go                # 任务定义
│   ├── queue/                    # 进程内任务队列
│   │   └── worker_pool.go
│   ├── query/                    # Panel 数据查询引擎
│   │   ├── builder.go            # 动态 SQL 构建
│   │   └── aggregator.go
│   ├── crypto/                   # 凭证加密（AES-GCM）
│   │   └── aes.go
│   └── pkg/                      # 通用工具
│       ├── response/             # 统一响应格式
│       ├── errors/               # 错误定义
│       └── logger/
├── migrations/                   # SQL 迁移文件
│   ├── 0001_init.up.sql
│   └── 0001_init.down.sql
├── configs/
│   ├── config.yaml
│   └── config.example.yaml
├── scripts/
├── web/                          # 前端项目
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── Makefile
```

### 2.2 前端目录结构

```
web/
├── src/
│   ├── api/                      # API 封装
│   ├── components/               # 通用组件
│   ├── layouts/                  # 布局
│   ├── pages/
│   │   ├── login/
│   │   ├── feedback/             # 反馈列表+详情
│   │   ├── dashboard/            # Dashboard 列表+编辑+查看
│   │   ├── alert/                # 告警规则+渠道+记录
│   │   ├── datasource/           # 数据源管理
│   │   └── user/                 # 用户管理
│   ├── stores/                   # Zustand stores
│   ├── router/
│   ├── utils/
│   └── main.tsx
├── package.json
└── vite.config.ts
```

---

## 3. 数据库设计（MySQL 8.0）

### 3.1 建表 DDL

> 默认字符集 `utf8mb4`，排序规则 `utf8mb4_0900_ai_ci`，引擎 InnoDB。

```sql
-- 用户表
CREATE TABLE users (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username        VARCHAR(64) NOT NULL UNIQUE,
  password_hash   VARCHAR(255) NOT NULL,
  role            VARCHAR(16) NOT NULL DEFAULT 'viewer'  COMMENT 'admin | viewer',
  status          TINYINT NOT NULL DEFAULT 1             COMMENT '1 启用 0 禁用',
  last_login_at   DATETIME NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 登录失败计数（替代 Redis 限流）
CREATE TABLE login_attempts (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  identifier   VARCHAR(128) NOT NULL COMMENT 'username 或 ip',
  attempted_at DATETIME NOT NULL,
  success      TINYINT NOT NULL,
  INDEX idx_identifier_time (identifier, attempted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 数据源配置
CREATE TABLE data_sources (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(128) NOT NULL,
  type            VARCHAR(32)  NOT NULL                   COMMENT 'http_api 等',
  config_json     JSON NOT NULL                            COMMENT '插件特定配置；敏感字段（cookie/headers）加密',
  sync_cron       VARCHAR(64) NOT NULL DEFAULT '*/5 * * * *',
  status          TINYINT NOT NULL DEFAULT 1               COMMENT '1 启用 0 禁用',
  last_sync_at    DATETIME NULL,
  last_sync_status VARCHAR(32) NULL                        COMMENT 'success | failed | running',
  last_sync_error TEXT NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_name (name),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 反馈元数据
CREATE TABLE feedbacks (
  id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  source_id             BIGINT UNSIGNED NOT NULL,
  original_id           VARCHAR(64)  NOT NULL                COMMENT '数据源原始ID',
  app_id                INT NOT NULL DEFAULT 0,
  app_name              VARCHAR(64) NULL,
  platform              VARCHAR(32) NULL                     COMMENT 'iOS/Android/HarmonyOS',
  platform_id           INT NULL,
  user_id               VARCHAR(64) NULL,
  user_name             VARCHAR(128) NULL,
  user_mode             INT NULL,
  content               TEXT NOT NULL,
  images                JSON NULL,
  phone_model           VARCHAR(64) NULL,
  app_version           VARCHAR(32) NULL,
  channel_id            VARCHAR(32) NULL,
  qq                    VARCHAR(32) NULL,
  file_url              TEXT NULL,
  raw_json              JSON NULL                            COMMENT '原始数据快照',
  category              VARCHAR(64) NULL                     COMMENT '分类结果',
  category_confidence   DECIMAL(4,3) NULL,
  category_status       TINYINT NOT NULL DEFAULT 0           COMMENT '0 待分类 1 已分类 2 失败 3 人工修正',
  sentiment             VARCHAR(16) NULL                     COMMENT 'positive/neutral/negative',
  classified_at         DATETIME NULL,
  original_created_at   DATETIME NOT NULL,
  synced_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_source_origid (source_id, original_id),
  INDEX idx_original_created (original_created_at),
  INDEX idx_app_platform_time (app_id, platform_id, original_created_at),
  INDEX idx_category (category),
  INDEX idx_sentiment (sentiment),
  INDEX idx_category_status (category_status),
  FULLTEXT INDEX ft_content (content) WITH PARSER ngram
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```
> 全文检索使用 MySQL 8.0 `ngram` 解析器以支持中文，最小词长 2。

```sql
-- Dashboard
CREATE TABLE dashboards (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(128) NOT NULL,
  description   VARCHAR(512) NULL,
  layout_json   JSON NULL                       COMMENT '布局信息（栅格）',
  created_by    BIGINT UNSIGNED NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 图表面板
CREATE TABLE panels (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  dashboard_id      BIGINT UNSIGNED NOT NULL,
  name              VARCHAR(128) NOT NULL,
  chart_type        VARCHAR(32) NOT NULL          COMMENT 'line/bar/pie/table/stat/stacked_bar/stacked_area',
  query_config_json JSON NOT NULL                 COMMENT '查询配置（维度/指标/过滤/时间）',
  position_json     JSON NULL                     COMMENT '位置 {x,y,w,h}',
  refresh_seconds   INT NOT NULL DEFAULT 60,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_dashboard (dashboard_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 告警规则
CREATE TABLE alert_rules (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name              VARCHAR(128) NOT NULL,
  panel_id          BIGINT UNSIGNED NULL          COMMENT '可绑定 Panel 复用其查询',
  metric_query_json JSON NOT NULL                 COMMENT '查询定义（同 Panel 查询）',
  time_window_sec   INT NOT NULL                  COMMENT '时间窗口秒数',
  condition_op      VARCHAR(8) NOT NULL           COMMENT '> < >= <= == !=',
  threshold         DECIMAL(20,4) NOT NULL,
  level             VARCHAR(16) NOT NULL          COMMENT 'warning/critical/emergency',
  silence_minutes   INT NOT NULL DEFAULT 30,
  eval_interval_sec INT NOT NULL DEFAULT 60,
  template_id       BIGINT UNSIGNED NULL,
  status            TINYINT NOT NULL DEFAULT 1,
  last_eval_at      DATETIME NULL,
  last_state        VARCHAR(16) NULL              COMMENT 'ok/firing',
  consecutive_fires INT NOT NULL DEFAULT 0,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 通知渠道
CREATE TABLE alert_channels (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(128) NOT NULL,
  type          VARCHAR(32) NOT NULL              COMMENT 'wecom/feishu/dingtalk',
  webhook_url   VARCHAR(512) NOT NULL             COMMENT '加密存储',
  secret        VARCHAR(255) NULL                 COMMENT '签名密钥（飞书/钉钉），加密',
  status        TINYINT NOT NULL DEFAULT 1,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 规则-渠道关联
CREATE TABLE alert_rule_channels (
  rule_id    BIGINT UNSIGNED NOT NULL,
  channel_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (rule_id, channel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 文案模板
CREATE TABLE alert_templates (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name             VARCHAR(128) NOT NULL,
  content_template TEXT NOT NULL,
  is_default       TINYINT NOT NULL DEFAULT 0,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 告警记录
CREATE TABLE alert_records (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  rule_id             BIGINT UNSIGNED NOT NULL,
  trigger_value       DECIMAL(20,4) NOT NULL,
  threshold           DECIMAL(20,4) NOT NULL,
  level               VARCHAR(16) NOT NULL,
  state               VARCHAR(16) NOT NULL          COMMENT 'firing/resolved',
  notified_channels   JSON NULL,
  notify_status       VARCHAR(32) NULL              COMMENT 'success/partial/failed',
  notify_detail_json  JSON NULL,
  triggered_at        DATETIME NOT NULL,
  resolved_at         DATETIME NULL,
  INDEX idx_rule_time (rule_id, triggered_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 同步水位线（增量同步游标）
CREATE TABLE sync_cursors (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  source_id     BIGINT UNSIGNED NOT NULL,
  cursor_key    VARCHAR(64) NOT NULL                COMMENT '如 app_id+platform_id 维度',
  last_original_id VARCHAR(64) NULL,
  last_created_at  DATETIME NULL,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_source_key (source_id, cursor_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 操作审计
CREATE TABLE audit_logs (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT UNSIGNED NULL,
  username    VARCHAR(64) NULL,
  action      VARCHAR(64) NOT NULL,
  resource    VARCHAR(64) NULL,
  resource_id VARCHAR(64) NULL,
  ip          VARCHAR(64) NULL,
  detail_json JSON NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_time (user_id, created_at),
  INDEX idx_action_time (action, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.2 索引设计要点
- `feedbacks` 复合索引覆盖最常见查询：按 `app_id + platform_id + 时间` 范围筛选。
- `ngram` 全文索引支持中文关键词搜索（MySQL 8.0+）。
- `alert_rules.status` 索引便于调度器扫描启用规则。
- 高基数字段（`user_id`、`phone_model`）不建独立索引，避免写放大。

---

## 4. 核心模块设计

### 4.1 认证与权限

#### JWT 流程
```
POST /api/auth/login → 校验 username/password → 校验失败计数 → 签发 JWT (HS256)
                    → 返回 access_token (默认 8h) + 用户信息
Header: Authorization: Bearer <token>
中间件解析 → 注入 user_id/role 到 context
```

#### 限流（无 Redis）
- 进程内 `sync.Map<identifier, *bucket>` 滑动窗口；
- 多实例部署时，回退到 `login_attempts` 表 5 分钟窗口计数；
- 任务后台清理表中过期记录（每 10 分钟）。

#### RBAC
- 中间件 `RequireRole("admin")` 守卫写操作；
- viewer 可访问 GET 类接口；
- admin 可访问全部接口。

### 4.2 数据源插件化

#### 插件接口
```go
type DataSourcePlugin interface {
    Type() string
    Validate(config json.RawMessage) error
    Pull(ctx context.Context, cfg json.RawMessage, cursor *SyncCursor) (
        items []FeedbackItem, nextCursor *SyncCursor, err error,
    )
}

type FeedbackItem struct {
    OriginalID        string
    AppID             int
    Platform          string
    PlatformID        int
    UserID, UserName  string
    Content           string
    Images            []string
    PhoneModel        string
    AppVersion        string
    ChannelID         string
    OriginalCreatedAt time.Time
    Raw               map[string]any
}
```

#### 注册表
- 启动时所有插件注册到 `datasource.Registry`；
- 调度器根据 `data_sources.type` 取插件实例执行 `Pull`。

#### HTTP API 插件配置示例
```json
{
  "endpoint": "https://demo.example.com/api/feedback_list",
  "method": "GET",
  "headers": {"accept": "application/json"},
  "cookie": "ENC:....",
  "params_template": {
    "is_ajax": "1",
    "app_id": "{{app_id}}",
    "platform_id": "{{platform_id}}",
    "to_page": "{{page}}"
  },
  "iterate": [
    {"app_id": 1, "platform_id": 2},
    {"app_id": 1, "platform_id": 3},
    {"app_id": 1, "platform_id": 8}
  ],
  "data_path": "data.feedback_question_list",
  "page_param": "to_page",
  "page_start": 1,
  "stop_when_empty": true,
  "field_mapping": {
    "original_id":         "id",
    "app_id":              "app_id",
    "platform_id":         "platform_id",
    "user_id":             "user_id",
    "user_name":           "screen_name",
    "user_mode":           "user_mode",
    "content":             "content",
    "images":              "image",
    "phone_model":         "phone_mode",
    "app_version":         "version_name",
    "channel_id":          "channel_id",
    "original_created_at": "created_at"
  }
}
```

#### 增量同步策略
- 翻页直至：遇到已存在 `original_id`（按 `uk_source_origid` 判断）或返回为空；
- 写库使用 `INSERT ... ON DUPLICATE KEY UPDATE` 防重；
- 更新 `sync_cursors.last_original_id` / `last_created_at`；
- 失败重试：指数退避，最多 3 次，最终写入 `last_sync_error`。

### 4.3 智能分类

#### 流程
```
新反馈入库 → 投递到 classify_queue
worker 取出 → 构造 Prompt → 调用 LLM → 解析 JSON → 写回 feedbacks
失败 → 重试 3 次（带退避）→ category_status=2
```

#### Prompt 模板（可配置）
```
你是产品反馈分类助手。请根据反馈内容输出严格 JSON：
{
  "category": "<分类>",
  "confidence": <0~1>,
  "sentiment": "positive|neutral|negative"
}
可选分类：功能问题、体验问题、内容问题、商业化问题、数据问题、兼容性问题、账号问题、建议需求、其他

反馈内容：{{content}}
```

#### 接口抽象
```go
type Classifier interface {
    Classify(ctx context.Context, content string) (Result, error)
}
type Result struct {
    Category   string
    Confidence float64
    Sentiment  string
}
```
实现：`LLMClassifier`（OpenAI 兼容），扩展点 `KeywordClassifier`（兜底/离线）。

#### 批量优化
- 单条调用成本高时，支持 batch（一次性传入 N 条，结果按下标返回）；
- worker pool 大小可配置，默认 4；
- 速率限制：每分钟最大调用次数可配置，避免触发 LLM 限流。

### 4.4 Dashboard 查询引擎

#### Panel 查询配置 JSON Schema
```json
{
  "metric": "count",                       // count | count_distinct
  "metric_field": "id",                    // count_distinct 时使用
  "x_dimension": {"field": "original_created_at", "bucket": "1h"},
  "group_by": ["category"],                // 可选
  "filters": [
    {"field": "app_id", "op": "=", "value": 1},
    {"field": "platform_id", "op": "in", "value": [2,3]},
    {"field": "category", "op": "in", "value": ["功能问题","数据问题"]}
  ],
  "time_range": {"type": "relative", "value": "7d"},
  "order_by": [{"field": "x", "dir": "asc"}],
  "limit": 1000
}
```

#### SQL 构建器
- 字段白名单（防 SQL 注入）：仅允许 `feedbacks` 表预定义列；
- 时间分桶：基于 MySQL `DATE_FORMAT(original_created_at, '%Y-%m-%d %H:00:00')` 或 `FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(...)/N)*N)`；
- 操作符白名单：`=, !=, >, >=, <, <=, in, not in, like`；
- 所有过滤参数作为 `?` 占位符传入，避免拼接；
- 限制 `LIMIT` 上限（默认 10000）；
- 全文搜索使用 `MATCH(content) AGAINST(? IN BOOLEAN MODE)`。

#### 预置 Panel 模板
后端提供 `GET /api/panels/templates`，前端拖拽即可创建。

### 4.5 告警引擎

#### 调度
- `cron` 每 `min(eval_interval_sec)` 秒触发扫描启用规则；
- 每个规则按 `last_eval_at + eval_interval_sec <= now` 判断是否求值。

#### 求值流程
```
取规则 → 用 query 引擎计算指标值 → 与 threshold 比较
  触发：state=firing
    - 若上次 firing 且未到 silence_minutes → 不发送
    - 否则 → 渲染模板 → 并行发送各渠道 → 写 alert_records
    - consecutive_fires++；达到阈值（如 3 次）→ 升级 level
  未触发：
    - 上次 firing → 发恢复通知 + resolved 记录
    - state=ok；consecutive_fires=0
```

#### 模板渲染
- 使用 Go `text/template`；
- 变量见 PRD §2.6.3；
- 模板预校验：保存时尝试用样例数据渲染。

#### Notifier 接口
```go
type Notifier interface {
    Type() string
    Send(ctx context.Context, channel *AlertChannel, message string) error
}
```
- `wecom`：POST JSON `{"msgtype":"markdown","markdown":{"content":"..."}}`；
- `feishu`：POST `{"msg_type":"text","content":{"text":"..."}}`，支持签名；
- `dingtalk`：POST `{"msgtype":"markdown",...}`，支持 sign。

### 4.6 调度器

- 使用 `robfig/cron/v3` 启动单个 scheduler；
- 启动时从 `data_sources` 加载启用项，注册同步 Job；
- 提供热加载：增删数据源后调用 `scheduler.Reload()`；
- 同步 Job 内部加分布式锁（基于 MySQL `GET_LOCK('sync_src_<id>', 0)`），防多实例并发；
- 告警 Job 类似处理。

### 4.7 异步任务队列

- 进程内 `chan Task` + worker pool；
- 失败任务持久化到 `task_retry` 表（轻量），重启可恢复；
  - 或简单方案：分类任务直接基于 `category_status=0` 扫表，重启即恢复，无需额外表。
- 当前 MVP 采用扫表方案：调度器每 30s 扫描 `category_status=0` 的反馈批量分类。

---

## 5. API 规范

### 5.1 统一响应
```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```
错误码：
- `0` 成功
- `40001` 参数错误
- `40100` 未认证
- `40300` 无权限
- `40400` 资源不存在
- `42900` 频率限制
- `50000` 服务器错误
- `50200` 外部依赖错误（LLM/数据源/Webhook）

### 5.2 分页参数
- `page`（默认 1）、`page_size`（默认 20，最大 100）；
- 响应：
```json
{
  "list": [...],
  "total": 1234,
  "page": 1,
  "page_size": 20
}
```

### 5.3 反馈列表查询参数
```
GET /api/feedbacks
  ?start_time=2026-05-01 00:00:00
  &end_time=2026-05-28 23:59:59
  &app_id=1
  &platform=iOS,Android
  &category=功能问题,数据问题
  &sentiment=negative
  &app_version=9.06.0
  &user_mode=2,3
  &keyword=闪退
  &order_by=original_created_at
  &order_dir=desc
  &page=1
  &page_size=20
```

---

## 6. 安全设计

### 6.1 密码与凭证
- 用户密码：`bcrypt` cost=10；
- 数据源敏感配置（cookie/headers/webhook_url）：AES-256-GCM，主密钥从环境变量 `APP_ENCRYPTION_KEY`（32 字节 base64）加载；
- 前端展示时脱敏（`********`），编辑时单独"修改"按钮。

### 6.2 输入校验
- 所有 Handler 入参使用 `validator` 标签校验；
- Panel 查询字段白名单；
- 自定义模板渲染禁用危险函数。

### 6.3 SQL 注入防护
- 全部使用 GORM 占位符或预处理；
- 动态构建查询时字段名白名单匹配。

### 6.4 XSS / CSRF
- API 全部返回 JSON，前端 React 默认转义；
- JWT 携带在 Authorization Header 而非 Cookie，避免 CSRF；
- CORS 仅放行前端域名。

### 6.5 敏感数据脱敏
- 列表返回 `qq` 字段：`123***42`；
- 用户手机号、设备 ID 类似处理。

### 6.6 操作审计
- 中间件记录所有写操作到 `audit_logs`；
- 包含 user_id、action、resource、ip、变更摘要。

### 6.7 外部请求
- 数据源拉取设置超时（30s）；
- LLM 调用设置超时（30s）；
- Webhook 调用设置超时（10s）+ 重试 3 次。

---

## 7. 配置示例（config.yaml）

```yaml
server:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

database:
  dsn: "feedback:password@tcp(127.0.0.1:3306)/feedback?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h

jwt:
  secret: "${JWT_SECRET}"
  expire: 8h

encryption:
  key: "${APP_ENCRYPTION_KEY}"   # 32 字节 base64

llm:
  provider: openai_compatible
  base_url: "https://api.openai.com/v1"
  api_key: "${LLM_API_KEY}"
  model: "gpt-4o-mini"
  timeout: 30s
  rate_limit_per_minute: 60
  worker_count: 4

scheduler:
  alert_eval_interval: 30s
  classify_scan_interval: 30s

log:
  level: info
  file: ./logs/app.log
```

---

## 8. 部署

### 8.1 Docker Compose

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: feedback
      MYSQL_USER: feedback
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_0900_ai_ci
      - --ngram_token_size=2
    volumes:
      - mysql_data:/var/lib/mysql
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "3306:3306"

  api:
    build: .
    environment:
      JWT_SECRET: ${JWT_SECRET}
      APP_ENCRYPTION_KEY: ${APP_ENCRYPTION_KEY}
      LLM_API_KEY: ${LLM_API_KEY}
    depends_on:
      - mysql
    ports:
      - "8080:8080"

  web:
    build: ./web
    ports:
      - "80:80"
    depends_on:
      - api

volumes:
  mysql_data:
```

### 8.2 部署模式
- **单进程**：api + scheduler + worker 合并启动（MVP 默认）；
- **分离部署**：高负载时可独立运行 `cmd/worker`，多实例通过 MySQL `GET_LOCK` 互斥避免重复拉取/告警。

---

## 9. 性能保障

| 场景 | 措施 |
|------|------|
| 列表查询 | 复合索引；分页强制；时间范围必选（默认最近 7 天） |
| 全文搜索 | ngram 全文索引；关键词长度 ≥ 2 |
| Dashboard 查询 | 进程内 LRU 缓存（key=query 指纹，TTL=30s）替代 Redis |
| 数据源拉取 | 单源串行、多源并行；遇到已存在 ID 即停止翻页 |
| 分类 | worker pool 限速；批量调用 |
| 告警求值 | 复用 Panel 查询缓存 |

---

## 10. 监控与可观测

- 健康检查：`GET /healthz`（DB ping）；
- Prometheus 指标（可选）：HTTP 请求数/延迟、同步成功率、分类成功率、告警触发次数；
- 日志：zap 结构化 JSON，按天滚动；
- 关键错误回写 `data_sources.last_sync_error` / `alert_records.notify_detail_json`，便于前端展示。

---

## 11. 实施计划与对应里程碑

| 里程碑 | 主要技术任务 |
|--------|------------|
| P0 | 项目脚手架、配置/日志/DB；用户/JWT/RBAC；数据源 CRUD + HTTP 插件 + 调度器 + 增量同步；反馈列表 API + 前端列表页 |
| P1 | LLM Classifier + 扫表调度 + 人工修正接口；列表筛选增加 category/sentiment |
| P2 | Dashboard/Panel CRUD；查询引擎；前端 Dashboard 编辑器 + ECharts 渲染；预置模板 |
| P3 | 告警规则/渠道/模板 CRUD；告警引擎 + 三类 Notifier；告警记录与恢复 |
| P4 | 多数据源插件扩展点示例；告警升级/静默；审计日志 UI；监控指标 |

---

## 12. 风险与对应技术决策

| 风险 | 技术决策 |
|------|---------|
| 无 Redis 影响限流 | 进程内滑动窗口 + 数据库兜底表 |
| LLM 成本/延迟 | 批量调用 + worker 限速 + 失败可人工修正 |
| 数据源 Cookie 过期 | 同步失败写错误信息 + 触发预置告警（系统级） |
| 多实例并发同步 | MySQL `GET_LOCK` 互斥 |
| 反馈表数据膨胀 | 按月分表预留扩展（MVP 单表 + 充分索引即可） |
