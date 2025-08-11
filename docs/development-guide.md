# TodoIng 开发指南

## 快速开始

### 环境要求

#### 系统要求

- Linux / macOS / Windows
- 最少 4GB RAM，10GB 可用磁盘

#### 软件依赖

| 软件 | 版本 | 说明 |
|------|------|------|
| Go | 1.23+ | 后端开发 |
| Node.js | 18+ | 前端开发 (Vite) |
| MongoDB | 7.0+ | 数据库 |
| Docker | 24+ | 可选：容器化本地/部署 |
| Git | 2.30+ | 版本控制 |

### 克隆与分支

```bash
git clone git@github.com:axfinn/todoIngPlus.git
cd todoIngPlus
git checkout dev
```

## 后端开发 (Go)

### 初始化

```bash
cd backend-go
cp .env.example .env
# 编辑 .env 后：
go mod download
```

关键环境变量（完整列表见 `docs/configuration.md`）：

- MONGO_URI
- JWT_SECRET
- PORT (默认 5004)
- DEFAULT_USERNAME / DEFAULT_PASSWORD / DEFAULT_EMAIL
- ENABLE_CAPTCHA / ENABLE_EMAIL_VERIFICATION / DISABLE_REGISTRATION

### 运行

```bash
# 直接运行
go run ./cmd/api/main.go

# 或使用 Docker 本地依赖（简化：只起 Mongo）
docker run -d --name todoing-mongo -p 27017:27017 mongo:7

# 构建二进制
go build -o bin/todoing ./cmd/api/main.go
./bin/todoing
```

(旧文档中的 `cmd/server/main.go` 已弃用，现入口统一为 `cmd/api/main.go`。)

### 项目结构（精简实际）

```text
backend-go/
├── cmd/
│   ├── api/              # 主 HTTP 入口
│   │   └── main.go
│   └── grpc/             # 备用 gRPC 入口（当前非主路径）
├── internal/
│   ├── api/              # 路由 + 处理器 (auth, tasks, events, reminders, unified, notifications...)
│   ├── auth/             # JWT 等认证
│   ├── captcha/          # 图片验证码
│   ├── email/            # 邮件发送 (验证码 + 通知)
│   ├── notifications/    # SSE Hub / 推送
│   ├── services/         # 聚合 & 业务组合 (unified, dashboard, reminder 调度等)
│   ├── models/           # 数据模型 (Mongo 映射 + DTO)
│   ├── observability/    # 日志 / 追踪占位
│   └── ... 其它工具 (convert, handlers legacy 等)
├── pkg/                  # 计划复用/公共封装（当前较少）
├── scripts/              # 脚本（生成 swagger / 测试）
├── test/                 # 测试帮助 & 集成测试
└── docs/                 # 后端相关文档
```

说明：已不再使用典型 repository/service 深层分层，直接在 handler 中组合 Mongo 操作或通过精简 service。旧文档的 `handler/service/repository/model` 四层示例已移除。

### 主要模块概览

| 模块 | 目录 | 说明 |
|------|------|------|
| 授权认证 | `internal/auth` + `internal/api/auth_handlers.go` | JWT 登录 / 默认用户创建 / 邮箱验证码 / 图片验证码 |
| 任务 & 事件 | `internal/api/tasks_handlers.go` / `event_routes.go` | 基础 CRUD |
| 提醒系统 | `internal/api/reminder_handlers.go` + scheduler | 定时扫描 `reminders` 集合 + 即时测试发送 |
| 通知 (SSE) | `internal/api/notification_handlers.go` + `internal/notifications` | Server-Sent Events 推送结构化消息 |
| 统一聚合 | `internal/api/unified_handlers.go` + `services/unified_service.go` | 合并任务 / 事件 / 提醒的日历 & upcoming 视图 |
| 仪表盘 | `internal/api/dashboard_handlers.go` | 汇总统计 |
| 邮件 | `internal/email` | 验证码发送 + 通知/提醒邮件 (REMINDER_EMAIL_* 覆盖) |

### 路由与中间件

- 统一 Router：`internal/api/middleware.go` 中 `NewRouter` + Logging / Recover / Auth
- SSE 通过自定义 ResponseWriter 保持 `Flush()` 可用
- Auth 支持 `Authorization: Bearer <token>` 或 `?token=`（便于 SSE）

### 提醒调度（摘要）

- 定时 Ticker（1m）查询条件：`is_active=true AND next_send <= now`
- 发送邮件 / SSE 后更新下一次 `next_send`
- 测试即时提醒：`POST /api/reminders/test`（使用临时事件）

详情参见 `docs/reminder-module.md`。

### 统一聚合 (Unified)

- Upcoming：`GET /api/unified/upcoming?hours=72&sources=task,event,reminder&limit=50`
- Calendar：按月聚合，支持按源过滤与 limit 截断
- `debug=1` 返回调试结构（`UnifiedUpcomingDebug`）

### API 文档

```bash
# 生成 swagger (脚本依据注释)
./scripts/generate_api_docs.sh
```

Swagger 诊断见 `SWAGGER_DIAGNOSIS.md`。

### 测试

```bash
# 单元 + 逻辑测试
go test ./...

# 指定包
go test ./internal/services -run TestReminder

# 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | head
```

测试公共工具：`test/testutil`（Mongo 临时 URI / 跳过逻辑）。

### 日志与调试

- `DEBUG=true` 输出调试级别日志：见 `internal/observability/logger.go`
- SSE 连接可查看响应头 `X-Instance` 确认实例

## 前端开发 (React + TS)

### 初始化（前端）

```bash
cd frontend
npm install
cp .env.example .env
```

关键变量（详见 config 文档）：`VITE_API_BASE_URL`、`VITE_ENABLE_CAPTCHA`、`VITE_ENABLE_EMAIL_VERIFICATION`、`VITE_DISABLE_REGISTRATION`。

### 常用脚本

```bash
npm run dev       # 开发
npm run build     # 构建
npm run preview   # 预览
npm run test      # 测试 (Vitest)
npm run lint      # ESLint
npm run type-check
```

### 结构（精简实际目录）

```text
frontend/src/
├── app/            # store 入口等
├── features/       # 按领域划分 slices / 逻辑 (tasks, reminders, auth ...)
├── components/     # UI 组件
├── pages/          # 页面级组件
├── hooks/          # 自定义 hooks
├── locales/        # i18n 文本
├── utils/          # 工具函数 (calendar utils 等)
└── styles/         # 样式
```

### 与后端交互要点

- Axios 拦截器自动注入 token
- SSE：前端可使用 `EventSource(/api/notifications/stream?token=JWT)` 接收通知
- 下载 ICS：`calendarUtils.ts` 辅助生成

## Docker / Compose

提供多份 compose（见 `docker/`）。典型前后端 + Mongo 开发：

```bash
docker compose -f docker/docker-compose.dev.yml up -d
```

若仅需 Mongo，可直接 `docker run mongo:7`。

## 部署简述

- 构建后端镜像：`docker build -t todoing/backend:latest backend-go`（或使用现有 Dockerfile）
- 运行：挂载 `.env` 或传入变量；暴露 5004
- 前端构建后由 Nginx / 静态服务器托管；Nginx 反向代理 `/api/` -> 后端
- 设置 `INSTANCE_ID` 便于多副本诊断

## 代码规范（简要）

### Go

- 包名小写；文件聚焦单一责任
- 处理器内快速校验参数，调用 service 或直接数据库操作
- 错误：返回 `http.Error` 或统一 JSON（后续可抽象）
- 避免过度抽象：当前未引入 repository 层

### TypeScript

- 使用显式接口定义 API 返回结构
- RTK createSlice 管理状态；避免在组件内直接发起重复请求
- 组件保持展示职责，数据逻辑放 slice / hooks

## 常见问题 (FAQ 精简)

### 登录验证码/邮箱不生效

- 检查后端是否设置 `ENABLE_CAPTCHA` / `ENABLE_EMAIL_VERIFICATION`
- 邮箱：确认 `EMAIL_HOST/USER/PASS`；若使用提醒专用，确保 `REMINDER_EMAIL_*` 三元完整

### Mongo 连接失败

```bash
docker ps | grep mongo
mongosh $MONGO_URI --eval 'db.runCommand({ping:1})'
```

### SSE 无推送

- 确认使用 GET 并保持连接（不要被代理提前关闭）
- Header 含 `Cache-Control: no-cache`
- Token 方式：`?token=JWT` 或 Bearer Header

### 统一聚合为空

- 检查 hours 参数是否过小
- 数据未来时间是否超出窗口
- 打开 `debug=1` 查看返回调试字段

## 已移除或暂不实现项

| 旧文档内容 | 状态 | 备注 |
|------------|------|------|
| repository 深层分层示例 | 移除 | 直接简化为 handler + service / 直连 DB |
| `cmd/server/main.go` 入口 | 移除 | 使用 `cmd/api/main.go` |
| OpenAI / AI 相关配置 | 暂未实现 | 删除说明，待集成再补充 |
| Redis / 消息队列 | 暂未实现 | 未来性能/异步需求再加入 |
| 复杂团队/统计集合 | 暂未实现 | 参考 `database-design.md` 精简版 |

---

本指南聚焦当前真实实现，新增模块或变量后请同步更新此文件与相关文档。
