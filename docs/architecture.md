# TodoIng 架构概览 (当前 dev 实现)

> 本文档同步至最新代码：引入了 Event / Reminder 专用 **Repository 层**，Service 变为薄封装；Reminder 支持 Preview / Immediate Test；统一聚合 (Unified) 增强调试输出；Proto 成为接口单一真实来源 (SSOT)。

## 1. 总览

```mermaid
flowchart LR
  subgraph Frontend
    FE[React + TS\nVite + RTK]
  end
  subgraph Backend(Golang Monolith)
    API[HTTP / gRPC Gateway]
    AUTH[Auth/JWT]
    TASK[Tasks]
    EVENT[Events\nRepo+Svc]
    REM[Reminders\nRepo+Scheduler]
    REP[Reports]
    UNI[Unified\nUpcoming/Calendar]
    NOTI[Notifications\nSSE Hub]
    OBS[Observability]
  end
  DB[(MongoDB)]
  EMAIL[(SMTP)]

  FE -->|REST / JSON| API
  FE <-->|gRPC (optional)| API
  API --> EVENT
  API --> REM
  API --> TASK
  API --> REP
  API --> UNI
  API --> NOTI
  EVENT --> DB
  REM --> DB
  TASK --> DB
  REP --> DB
  UNI --> DB
  AUTH --> DB
  REM --> EMAIL
  FE <-->|SSE| NOTI
```

## 2. 分层与职责

| 层/模块 | 状态 | 说明 |
|---------|------|------|
| Handlers (`internal/api`) | ✅ | 参数解析 / 鉴权 / 调用 Service / 输出 DTO |
| Services (`internal/services`) | ✅ 精简 | 事件 & 提醒：调用 Repository；其余仍直接 Mongo （逐步迁移） |
| Repositories (`internal/repository`) | ✅ 部分 | 已实现 `EventRepository`, `ReminderRepository`（含复杂推进 / 聚合 / 预览 / 级联逻辑） |
| Models (`internal/models`) | ✅ | 领域模型 + 计算函数 (eg. `CalculateNextSendTime`) |
| Scheduler (`reminder_scheduler.go`) | ✅ | 每分钟扫描 `next_send`；使用仓储重新计算/标记发送 |
| Unified (`unified_service.go`) | ✅ | 汇总 tasks/events/reminders；新增 debug 输出窗口过滤信息 |
| gRPC (`internal/grpc`) | ✅ | 与 HTTP 共用 Service/Repository；Proto 为单一真实来源 |

迁移策略：优先将“含复杂 Mongo 聚合 / 级联写”下沉至 Repository；简单 CRUD 可保持在 Service 直到需要扩展。

## 3. Repository 要点

### EventRepository

- 提供：`Insert / Find / UpdateFields / Delete / ListPaged / ListUpcoming / CalendarRange / Search`
- 拓展：`Advance` (推进循环+写系统时间线+异步重算相关 Reminders)、`ListStartingWindow`、`MarkTriggered`

### ReminderRepository

- 提供：`Insert / GetWithEvent / UpdateFields(recomputeNext) / Delete / ListPagedWithEvent / ListSimple / Upcoming / Pending`
- 调度辅助：`ToggleActive / Snooze / MarkSent`
- 特殊：`CreateImmediateTest`（测试提醒）、`Preview`（不落库计算 next_send + 文本）

Service 不再直接使用 `bson`，仅组装更新字段 map。

## 4. 接口单一真实来源 (SSOT)

- 所有对外 gRPC + HTTP (grpc-gateway) 接口由 `api/proto/v1/*.proto` 定义
- 生成：`make proto-all` => `pkg/api/v1/*` + `docs/swagger/todoing.swagger.json`
- OpenAPI 前端直接消费；手写 Swagger 片段已废弃

## 5. 统一聚合 (Unified)

功能：`/api/unified/upcoming` & `/api/unified/calendar`

- Upcoming：聚合 events (含循环下一次实例化)、reminders (active + 未来窗口)、tasks（带优先任务 + 普通任务日期解析）
- 支持 sources 过滤、limit 限制；context 里设置 `unifiedDebug` 时返回调试结构（窗口过滤器、统计）
- Calendar：按日 bucket events/reminders/tasks

## 6. 提醒扩展

新增能力：

- Preview：POST `/api/reminders/preview` 调用 Repository 计算 next_send 与描述文本
- Immediate Test：POST `/api/reminders/test` 创建临时提醒并可即时尝试发送邮件
- Scheduler：使用 `Pending()` + `MarkSent()`；发送后重算 next_send（循环事件未来扩展）

## 7. 时间线与推进

- 事件创建 / 推进写入 `event_comments` 系统记录（异步 goroutine）
- `Advance` 同时：推进日期或关闭事件 + 重算相关提醒 next_send + 写系统 comment

## 8. 渐进式重构现状

| 领域 | Mongo 直接访问 | Repository 封装 | 说明 |
|------|----------------|------------------|------|
| Events | ❌ | ✅ | 全量迁移完成 |
| Reminders | ❌ | ✅ | 全量迁移 + Preview/Test |
| Tasks | ✅ | ❌ | 后续按需引入（排序/统计逻辑较少） |
| Reports / Dashboard | ✅ | ❌ | 聚合简单，暂不抽象 |
| Notifications | ✅ | ❌ | 直接 collection 写入足够 |

## 9. 指标 / 观察点 (规划)

- 索引建议：`events(user_id,event_date,is_active)`, `reminders(is_active,next_send)`, `reminders(user_id,created_at)`
- 计划增加：Prometheus exporter / metrics 路由 / repository 层 instrumentation

## 10. 未来迭代

- 统一 UserRepository（当前 scheduler/测试提醒仍直接查 `users`）
- 任务 & 报表 Repository 化 / 查询缓存
- 更丰富的事件循环 (cron-like) + reminder next_send 智能批量重算
- SSE -> 可插拔 WebSocket 层

---
最后更新：自动化重写 (包含 repository+preview 变更)
