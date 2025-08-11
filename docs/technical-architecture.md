# TodoIng 技术架构（当前实现版）

> 本文件已重写以与实际代码保持一致；旧版中提到的微服务、WebSocket、repository 深层分层、Redis、OpenAI、Zap、Viper 等尚未实现或已移除。更完整叙述参见：`architecture.md`、`database-design.md`、`reminder-module.md`。

## 总览

当前形态：单体 Go HTTP API + React 前端 + MongoDB。

- 后端：Go 1.23，`gorilla/mux` 路由，中间件 (Logging / Recover / Auth)；SSE 用于实时通知。
- 前端：React + TS + Redux Toolkit + Vite；使用 EventSource 订阅通知。
- 数据库：MongoDB（集合：users, tasks, events, reminders, reports, notifications）。
- 调度：内部 `time.Ticker` 每分钟扫描 reminders。
- 认证：JWT + 可选图片验证码 + 邮箱验证码。
- 邮件：通用 `EMAIL_*` + 可选 `REMINDER_EMAIL_*` 覆盖。

## 组件关系（简化）

```text
[ React (Vite) ] -- REST --> [ Go API (mux) ] -- Mongo Driver --> [ MongoDB ]
        |                             |
        '---- SSE (EventSource)  <-----'
```

## 后端目录实际结构

```text
cmd/api/main.go          # 入口
internal/api             # 处理器 (auth/tasks/events/reminders/unified/notifications/...)
internal/auth            # JWT / token 解析
internal/captcha         # 图片验证码
internal/email           # 邮件发送 & 覆盖逻辑
internal/notifications   # SSE hub
internal/services        # 组合 / 聚合逻辑 (unified, dashboard, reminder scheduler helper)
internal/models          # 数据模型
internal/observability   # 日志（轻量封装）
```

(旧文档的 `handler/service/repository` 典型分层与 `cmd/server` 入口已废弃。)

## 核心模块说明

| 模块 | 关键文件 | 说明 |
|------|----------|------|
| 认证 & 用户初始化 | `internal/api/auth_handlers.go` / `internal/auth/jwt.go` | 登录/注册/验证码，缺用户时创建默认账户 |
| 任务 & 事件 | `internal/api/tasks_handlers.go` / `internal/api/event_routes.go` | 基础 CRUD |
| 提醒系统 | `internal/api/reminder_handlers.go` | 创建/列出/测试；scheduler 周期扫描发送 |
| 调度逻辑 | scheduler（main 中初始化） | `next_send` 条件查询并更新 |
| 通知 (SSE) | `internal/api/notification_handlers.go` + `internal/notifications` | 服务器推事件，前端 EventSource 接收 |
| 统一聚合 | `internal/api/unified_handlers.go` + `internal/services/unified_service.go` | upcoming + calendar 聚合多个源 |
| 仪表盘 | `internal/api/dashboard_handlers.go` | 汇总统计 |
| 邮件发送 | `internal/email/email.go` | 验证码 & 通知邮件，支持双配置组 |

## 与旧架构文档差异对照

| 旧描述 | 现状 | 说明 |
|--------|------|------|
| WebSocket 实时 | 使用 SSE | `EventSource` 简化，无自定义协议握手需求 |
| repository 层 | 省略 | 直接 Mongo 调用 + 轻量 services |
| Redis 缓存 | 未实现 | 后续如需会在高频查询/排行榜加入 |
| OpenAI 集成功能 | 未实现 | 移除示例配置，规划阶段 |
| Zap / Viper | 未使用 | 自定义轻量日志 & 直接 `os.Getenv` |
| Rate limiting | 未实现 | 可后续在中间件补充 |
| WebSocket 离线队列 | 未实现 | SSE 不做离线；将来可用消息队列/Redis PubSub |
| gRPC 微服务通信 | 备用入口存在 | 非主路径；当前单体充分 |

## 数据模型（摘录）

> 全量详见 `database-design.md`。

```go
// Reminder 关键字段
type Reminder struct {
    ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    EventID    *primitive.ObjectID `bson:"event_id,omitempty" json:"event_id,omitempty"`
    UserID     primitive.ObjectID  `bson:"user_id" json:"user_id"`
    Title      string              `bson:"title" json:"title"`
    ReminderType string            `bson:"reminder_type" json:"reminder_type"`
    NextSend   *time.Time          `bson:"next_send" json:"next_send"`
    IsActive   bool                `bson:"is_active" json:"is_active"`
    CreatedAt  time.Time           `bson:"created_at" json:"created_at"`
    UpdatedAt  time.Time           `bson:"updated_at" json:"updated_at"`
}
```

## 通知机制 (SSE)

- 前端：`EventSource('/api/notifications/stream?token=JWT')`
- 服务端：保持长连接，定期/事件触发写入 `event: notification\ndata: {...}\n\n`
- 响应头：`Content-Type: text/event-stream`, `Cache-Control: no-cache`
- 中间件确保 `Flush()` 可用

## 提醒调度流程

1. Ticker 触发扫描：`is_active=true AND next_send <= now`
2. 生成邮件 / SSE 通知
3. 更新 reminder：`last_sent` & 计算下一个 `next_send`
4. 测试提醒：临时事件，不持久化主事件集合

## 聚合 (Unified) 简述

- upcoming：聚合 tasks/events/reminders 未来窗口内条目，按时间排序
- calendar：按日 bucket，支持源过滤与数量限制
- `debug=1` 返回窗口参数与范围

## 安全要点

| 方面 | 当前实现 |
|------|----------|
| 认证 | JWT (HS256) |
| 验证码 | 图片 + 邮箱（可开关） |
| 授权 | 用户资源隔离：查询按 `user_id` 过滤 |
| 速率限制 | 未实现 |
| RBAC | 暂无复杂角色；默认用户为 admin 用于初始管理 |

## 性能与扩展计划（Roadmap）

| 项目 | 状态 | 计划 |
|------|------|------|
| Reminder 扫描效率 | 基础 | 添加索引 (`is_active`, `next_send`) 已在设计中 |
| SSE 横向扩展 | 待做 | 借助 Redis Pub/Sub 或内存分片广播 |
| 缓存层 (Redis) | 待做 | 统一聚合结果短期缓存 |
| 指标监控 | 待做 | 暴露 `/metrics` Prometheus 指标 |
| Rate Limiting | 待做 | Token 桶中间件 |
| 任务/事件高级搜索 | 规划 | 建立文本索引/ES 集成 |

## 文件引用

- 架构：`docs/architecture.md`
- 数据库：`docs/database-design.md`
- 配置：`docs/configuration.md`
- 提醒：`docs/reminder-module.md`
- 统一聚合：源代码 `internal/api/unified_handlers.go`

---

本文件旨在取代旧版 `technical-architecture.md` 过时内容，随实现演进更新。
