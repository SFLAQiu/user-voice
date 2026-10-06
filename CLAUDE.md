# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

倾听用户反馈（Feedback Intelligence Platform）— 基于 Go + React 的系统，自动从外部 HTTP API 采集用户反馈，通过 LLM 智能分类归因，提供可视化 Dashboard 和告警通知。目标用户：内部运营/产品团队。

## 构建与运行命令

```bash
# 后端
make build          # go build -o bin/feedback-api ./cmd/api
make run            # go run ./cmd/api（默认加载 configs/config.yaml）
make test           # go test -race -cover ./...
make lint           # gofmt 检查 + go vet
make tidy           # go mod tidy

# 前端（web/）
cd web && npm run dev      # Vite 开发服务器 :5173，代理 /api → :8080
cd web && npm run build    # tsc + vite build → web/dist/

# 本地全栈运行
docker compose up -d       # MySQL + API；前端通过 Vite dev 或 Go 静态托管 web/dist
```

单个 Go 进程同时运行 API 服务器 + 调度器 + 分类器 + 告警引擎。

**环境配置**：配置默认加载 `configs/config.yaml`（首次启动无此文件时进入网页初始化向导，`make setup` 可命令行初始化），可用 `-config <路径>` 指定。环境变量覆盖使用 `FEEDBACK_<SECTION>_<KEY>`（如 `FEEDBACK_DATABASE_DSN`）。本地含敏感信息的 `configs/config*.yaml` 已加入 `.gitignore`，仅 `configs/config.example.yaml` 模板入库。

YAML 配置仅包含基础设施（server、database、jwt、encryption、admin、log、cors）。LLM 相关配置（provider、熔断器、重试策略、分类器调度参数）全部从 DB `llm_providers` / `llm_configs` 表读取，不再在 YAML 中配置。

## 架构

**后端**（`internal/`）：分层 Gin 应用 — handler → service → repository → GORM/MySQL。

- `cmd/api/main.go`：组装所有依赖，启动调度器、分类器、告警引擎、HTTP 服务器
- `handler/`：Gin HTTP 处理器；入参解析，响应包装使用 `pkg/response`
- `service/`：业务逻辑层。`datasource_service.go` 编排插件 Pull + upsert + 游标更新；`classifier_service.go` 扫描 `category_status=0` 的记录并分发到 worker pool；`feedback_service.go` 处理列表/详情/分类修改
- `repository/`：GORM 数据访问。`feedback_repo.go` 构建动态查询（白名单安全谓词 + ngram 全文检索）；使用 `ON DUPLICATE KEY UPDATE` 做 upsert
- `model/`：GORM 实体 + 常量（category_status、sentiment、排序白名单）
- `datasource/plugin.go`：`Plugin` 接口（`Type`、`Validate`、`Pull`）；`Registry` 插件注册表
- `datasource/httpapi/plugin.go`：HTTP API 插件 — 分页拉取、字段映射、Cookie 解密、已存在 ID 早停
- `datasource/mysql/plugin.go`：MySQL 表插件 — 时间游标增量（`WHERE time_field > cursor`）、`batch_limit` 单批上限、表名/列名防注入（`quoteIdent`）、密码解密
- `classifier/`：`Classifier` 接口；`MultiLLMClassifier`（多 provider + 熔断 + 重试）；可配置 Prompt 模板；`parseResult` 解析 LLM 输出
- `scheduler/cron.go`：robfig/cron + MySQL `GET_LOCK` 互斥锁（多实例安全）
- `alert/engine.go`：周期性规则评估 + Webhook 通知（企业微信/飞书/钉钉）
- `queue/worker_pool.go`：进程内有界 worker pool，用于分类任务
- `crypto/aes.go`：AES-256-GCM 加密敏感配置字段（`datasource_service.encryptSensitive` 按插件类型处理：httpapi 的 cookie/headers、mysql 的 password，密文带 `ENC:` 前缀）
- `middleware/`：JWT 认证、RBAC（`RequireRole("admin")`）、审计日志、CORS、异常恢复

**前端**（`web/src/`）：React 18 + Ant Design 5 + ECharts + Zustand + React Router 6。

- `api/`：axios 封装；`http.ts` 自动添加 JWT header
- `pages/feedback/FeedbackList.tsx`：表格 + 抽屉详情 + 分类修改 + 搜索表单
- `pages/datasource/DataSourceList.tsx`：CRUD 模态框 + 手动触发同步
- `stores/auth.ts`：Zustand store 存储 JWT token + 用户信息
- `layouts/MainLayout.tsx`：侧边栏导航
- `App.tsx`：路由定义，`RequireAuth` 守卫

**API 响应规范**：所有接口使用 `{code, message, data}` 统一包装。错误使用 `apperr.AppError`，错误码：0=成功, 40001=参数错误, 40100=未认证, 40300=无权限, 40400=不存在, 42900=限频, 50000=内部错误, 50200=外部依赖失败。

## 约束与规范

项目约束文档放在 `docs/ai/` 下，涉及对应领域时按需读取对应文件。不遵守 `constraints/` 下的规范会导致 bug：

| 触发场景 | 必读文件 |
|---|---|
| 时间解析/格式化/DB 时间查询/趋势图 | `docs/ai/constraints/timezone.md` |
| DDL 变更、新增 GORM model | `docs/ai/constraints/database.md` |
| 提交代码 | `docs/ai/constraints/git.md` |
| 涉及架构决策、理解系统设计意图 | `docs/ai/decisions/architecture.md` |

以下规范始终生效（写入 `docs/ai/conventions/chinese.md`）：文档与 AI 输出使用中文、面向用户的提示使用中文、代码注释优先中文。

## 运行测试

```bash
make test                              # 全部包
go test -race -cover ./internal/crypto # 单个包
go test -run TestEncrypt ./internal/crypto # 单个测试
```

前端暂无测试。集成测试需要 MySQL 运行。
