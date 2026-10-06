<div align="left">
  <img src="web/public/logo.svg" width="56" align="top" alt="logo" />
  <h1 style="display:inline;vertical-align:middle;margin-left:8px">user-voice</h1>
</div>

用户反馈平台（Go + React）— 自动从多种数据源采集用户反馈，通过 LLM 智能分类归因，提供可视化 Dashboard 和告警通知。

📖 [功能介绍与使用教程](docs/TUTORIAL.md)（含截图演示）

## 功能特性

- **多数据源采集**：HTTP API / MySQL 表增量拉取，Cron 定时 + 手动触发，游标式增量同步，支持字段映射与枚举翻译
- **LLM 智能分类**：多 Provider + 熔断 + 重试，自动归类业务模块与情感倾向
- **可视化分析**：时间网格预聚合仪表盘、自定义维度、趋势图
- **告警通知**：规则评估 + 企业微信/飞书/钉钉 Webhook

## 快速开始

前置：Go 1.25+、Node 18+、MySQL 8。

```bash
make run     # 首次启动无配置时自动进入初始化向导模式，浏览器访问 http://localhost:8080 完成初始化
```

前端开发：

```bash
cd web && npm install && npm run dev   # 或 make dev 一键起前后端
```

## Docker 方式

```bash
MYSQL_ROOT_PASSWORD=... JWT_SECRET=... APP_ENCRYPTION_KEY=... docker compose up -d
```

表结构由 API 进程启动时自动迁移，无需手工导入 SQL。

## 敏感配置

一律通过环境变量覆盖，不入库：

- `FEEDBACK_DATABASE_DSN`
- `FEEDBACK_JWT_SECRET`
- `FEEDBACK_ENCRYPTION_KEY`
- `FEEDBACK_ADMIN_USERNAME` / `FEEDBACK_ADMIN_PASSWORD`

模板见 `configs/config.example.yaml`；推荐用网页向导初始化（`make run` 后浏览器访问服务地址），命令行方式可用 `make setup`。

## 更换数据库

改配置文件里的 `database.dsn`（或用 `FEEDBACK_DATABASE_DSN` 环境变量覆盖），重启即可。新库只需建空库，表结构与种子数据启动时自动创建。注意 DSN 需保留 `charset=utf8mb4&parseTime=True&loc=Local` 参数；换库不迁移历史数据。不要通过删配置重走向导的方式换库——向导会重新生成加密密钥，旧库中已加密的数据（如 LLM API Key）将无法解密。

## 数据库约定

表结构只通过 `migrations/` 下的 SQL 维护（启动自动执行，记录在 `schema_migrations`）。新增表/字段必须新增迁移文件，不再使用 ORM AutoMigrate。

## 数据源配置

在「数据源管理」页面新建数据源，支持两种类型（JSON 配置，密码/Cookie 保存时自动加密）：

**MySQL 增量拉取**：指定外部库表的连接信息、增量时间列（`time_field`）与字段映射（`field_mapping`，`original_id` / `content` / `original_created_at` 必填），单批默认拉 1000 条，靠 Cron 逐批追平；增量游标持久化在 `sync_cursors` 表，不重不漏。

**HTTP API 拉取**：分页拉取外部接口，支持字段映射、枚举翻译、已存在 ID 早停等，详见页面内「字段说明」。
