# 提醒模块使用与架构说明 (当前实现版本)

> 本文档描述 dev 分支当前提醒系统（Reminders）真实代码实现，而不是旧版 `EVENT_REMINDER_API.md` 中的早期设计。涵盖数据模型、核心流程、API 端点、调度机制、即时测试提醒、通知/邮件发送、扩展点与改进建议。

## 1. 目标与特性概览

提醒模块围绕“事件 (Event)”提供多渠道、可配置、可预览的提醒能力：

- 支持基于事件日期 + 提前天数 (advance_days) + 一天内多个时间点 (reminder_times) 的组合调度。
- 支持提醒类型：`app`（站内通知）、`email`、`both`。
- 支持快速获取轻量列表 `/simple`，减少前端渲染等待。
- 支持即将到来的提醒查询 `/upcoming`。
- 支持暂停/激活、Snooze（推迟）、预览不落库。
- 支持“测试提醒”立即创建并（现在）即时尝试发送邮件：`/api/reminders/test`。
- 调度器后台每分钟轮询到期提醒发送，并推进下一次发送时间（支持未来扩展周期事件）。
- SSE 通知集成：发送提醒或邮件时创建 Notification 并广播。

## 2. 数据模型 (Mongo)

集合：`reminders`、`events`、`users`、`notifications`

Reminder 文档（字段与代码 `internal/models/reminder.go` 一致）：

```jsonc
{
  "_id": ObjectId,
  "event_id": ObjectId,         // 关联事件
  "user_id": ObjectId,
  "advance_days": 0,            // 提前 N 天 (0=事件当天)
  "reminder_times": ["09:00", "18:30"], // 当天多个时间点 (HH:MM 24h)
  "reminder_type": "app|email|both",
  "custom_message": "可选自定义消息",
  "is_active": true,
  "last_sent": ISODate|null,
  "next_send": ISODate|null,    // 下一次发送时间（已计算）
  "created_at": ISODate,
  "updated_at": ISODate
}
```

关键派生逻辑：
- `next_send = CalculateNextSendTime(event)`：基于事件实际发生时间 - advance_days 的日期，再在该日期内按顺序寻找首个未来的 reminder_time。
- 若事件已过期或当日之前则 `next_send = null`。
- 发送后通过 `MarkReminderSent` 重算下一次（目前对非循环事件会归零，可扩展到循环事件）。

SimpleReminderDTO（`ListSimpleReminders` 输出）：

```jsonc
{
  "id": "...",
  "event_id": "...",
  "event_title": "事件标题",
  "event_date": "2025-08-20T10:00:00Z",
  "advance_days": 0,
  "reminder_times": ["09:00"],
  "reminder_type": "app",
  "custom_message": "...",      // 可为空
  "is_active": true,
  "next_send": "2025-08-20T09:00:00Z",
  "last_sent": null,
  "created_at": "...",
  "updated_at": "..."
}
```

## 3. 核心组件

| 组件 | 位置 | 作用 |
|------|------|------|
| 模型 | `internal/models/reminder.go` | 定义结构与 `CalculateNextSendTime` 算法 |
| 服务 | `internal/services/reminder_service.go` | CRUD、列表、简单列表、预览、即将到来、待发送、测试提醒创建、状态切换、Snooze |
| 调度器 | `internal/services/reminder_scheduler.go` | 每分钟扫描 `next_send <= now` 发送并推进状态 |
| 处理器 | `internal/api/reminder_handlers.go` | HTTP 层请求解析、响应、权限与参数校验 |
| 邮件发送 | `internal/email/*` | `email.SendGeneric()` 统一发送封装 |
| 通知 | `internal/notifications` | 创建 Notification 并通过 Hub SSE 广播 |
| 用户邮箱查询 | 调度器 + 测试接口 | 直接查 `users` 集合 `email` 字段 |

## 4. 计算流程：`CalculateNextSendTime`

1. 若 `is_active=false` → 返回 `nil`。
2. 解析事件下一次发生时间：
  - 当前实现：若 `event.recurrence_type == "none"` 则只使用事件日期；未来可扩展循环（函数已预留 `GetNextOccurrence`）。
3. 计算提醒日期：`reminderDate = eventDate - advance_days`。
4. 若提醒日期已经早于今天（按天比较）→ `nil`。
5. 在 `reminder_times`（顺序）中挑出第一个 “当前时间之后” 点，构造 `reminderDate + HH:MM`。
6. 没有任何符合 → `nil`。

## 5. 调度器工作流

周期：`time.NewTicker(1 * time.Minute)`。

流程：

```
GetPendingReminders(): match { is_active:true, next_send <= now }
  -> for each reminder:
       sendReminder(reminder+event)
         - 根据 reminder_type: app / email / both
         - 生成消息: 自定义消息优先，否则模板（事件类型 + 标题 + 距离描述）
       MarkReminderSent(reminderID)
         - 写入 last_sent = now
         - 重新计算 next_send (对一次性事件多数情况下变为 nil)
```
错误处理：单条发送失败仅记录日志，继续后续。

## 6. 即时测试提醒 `/api/reminders/test`

当前逻辑（`CreateTestReminder`）：

1. 可选提交 `event_id`；若缺省则自动挑选该用户最近一个事件（按 `event_date` 升序）。
2. 参数：

```jsonc
{
  "event_id": "(可选)",
  "message": "(可选, 默认: 测试提醒)",
  "delay_seconds": 5,          // >=0, <=3600
  "send_email": true           // 可选；若缺省默认 true（即进行即时邮件尝试）
}
```
3. 调用 `CreateImmediateTestReminder`：
  - 直接插入一条 reminder：`advance_days=0`, `reminder_times=[now+delaySeconds]`, `next_send = now+delay`。
4. 现在新增：创建后**即时尝试发送邮件**（不等待调度器）：
  - 查询用户邮箱 → `email.SendGeneric()`。
  - 返回：`email_sent=true/false`, `email_error`（可选）。
5. 调度器仍会在到期时再扫描；若邮件已发，这条 reminder 在下一轮可能因已过 `next_send` 重新被匹配（后续可增加：测试提醒自动失效逻辑）。

响应示例：

```jsonc
{
  "reminder": { "id": "...", "next_send": "2025-08-10T08:20:05Z", ... },
  "event": { "id": "...", "title": "项目启动会" },
  "email_sent": true,
  "note": "Test reminder created and immediate email attempted"
}
```

## 7. API 端点一览

| 方法 | 路径 | 描述 | 备注 |
|------|------|------|------|
| GET | `/api/reminders/simple` | 轻量列表 | `?active_only=true&limit=50` |
| GET | `/api/reminders/upcoming` | 未来 N 小时（默认24）到期 | `?hours=48` |
| POST | `/api/reminders/test` | 创建测试提醒 & 尝试立即邮件 | `send_email` 可控 |
| GET | `/api/reminders` | 分页列表 | `page,page_size,active_only` |
| POST | `/api/reminders` | 创建提醒 | 提供 `event_id, advance_days, reminder_times, reminder_type, custom_message` |
| POST | `/api/reminders/preview` | 预览计划不落库 | 返回 `schedule_text`, `next_send` |
| GET | `/api/reminders/{id}` | 获取单个 | - |
| PUT | `/api/reminders/{id}` | 更新提醒 | 任意字段可选（指针） |
| DELETE | `/api/reminders/{id}` | 删除 | - |
| POST | `/api/reminders/{id}/snooze` | 推迟 N 分钟 | `{ "snooze_minutes": 90 }` |
| POST | `/api/reminders/{id}/toggle_active` | 启用/停用切换 | 返回最新状态 |

路由顺序：静态前缀 (`/simple`,`/upcoming`,`/test`) 先注册，再注册 `/{id}`，避免被通配捕获；`{id}` 添加 24 位十六进制正则确保只匹配合法 ObjectID。

## 8. 典型使用流程（创建→发送）
```
[前端] 用户选择事件 & 配置 -> POST /api/reminders
[服务] 校验 + Insert -> 计算 next_send
[调度器] 每分钟扫描 next_send<=now -> 发送 -> MarkReminderSent 重算
[前端] /simple 或 /upcoming 轮询/刷新视图
```

测试提醒：

```
[前端] 点击“测试提醒” -> POST /api/reminders/test (delay_seconds=5)
[服务] 插入 + 即时尝试邮件 -> 返回 email_sent 标志
[调度器] 5 秒后到期（若未即时邮件或需第二渠道）
```

## 9. 关键算法与注意点

| 场景 | 行为 |
|------|------|
| 事件已过期 | 新建/重算 next_send → nil（不会发送） |
| 多个时间点 | 取提醒日期当天首个未来时间 |
| 停用提醒 | `is_active=false` → next_send 不再计算/使用 |
| Snooze | 直接写入新 `next_send = now + snooze_minutes` |
| 测试提醒 | 直接写固定 next_send，不依赖计算函数 |
| 重发策略 | 当前不做重试；邮件失败仅日志 + API 标志 |
| 并发 | 调度器串行循环；未来高并发可改成批量+goroutine |

## 10. 安全与校验
- 所有端点经 Auth 中间件，`GetUserID` 解析 JWT。
- ObjectID 字段均校验 24 hex；路由层正则限制。
- Create/Update 像时间、类型等做基本字段检查。
- `CreateEvent` & `CreateReminder` 采用“严格解析失败→宽松 fallback”策略，提升兼容性同时记录警告日志。

## 11. 性能考量
- 常规列表使用聚合 `$lookup + $project`，分页加排序 + skip；未来可加复合索引：`{ user_id:1, created_at:-1 }`、`{ is_active:1, next_send:1 }`。
- `/simple` 限制最大 100 条，前端首次渲染快速。
- 调度器查询 `GetPendingReminders` 建议创建索引：`{ is_active:1, next_send:1 }`。
- 测试提醒路径避免复杂计算，仅一次插入。

## 12. 日志 & 可观测
- 关键操作（创建提醒 / 创建事件 / 严格解析失败）使用 `observability.LogInfo/LogWarn`。
- 调度器输出发送统计、失败原因。
- 可扩展：Prometheus 指标（待实现）。

## 13. 失败与边界情况
| 情况 | 当前结果 | 改进建议 |
|------|----------|----------|
| 用户无邮箱 | 即时邮件失败 `email_error` | 前端提示绑定邮箱 |
| 测试提醒发送后仍在调度器再次触发 | 可能重复渠道 | 增加标志 `is_test` 或自动停用 |
| 事件改期 | 现有提醒 next_send 未自动刷新 | 在事件更新逻辑中调用 `UpdateEventReminders` (TODO) |
| 时区差异 | 使用服务器本地/UTC 混合 | 统一存储 UTC，前端本地化显示 |
| 长期循环事件 | 暂未 fully 覆盖 | 扩展 `GetNextOccurrence` 并在重算链路调用 |

## 14. 未来改进路线图
1. `UpdateEventReminders` 实现：事件修改后批量重算相关提醒。
2. 支持复杂循环 (daily/weekly/monthly/annual + exclusion dates)。
3. 测试提醒标记与自动清理（定期删除 is_test=true 且过期）。
4. 增加邮件发送重试与队列（避免瞬时失败）。
5. SSE 统一事件类型枚举 + 前端实时刷新提醒列表。
6. 提醒时间自然语言解析下沉到后端（前端已有初步解析）。
7. 增加管理界面：查看调度器状态 & 强制执行（目前已有 /triggerOnce 仅内部使用）。
8. 指标：pending 数量、发送成功率、延迟、失败分布。
9. 权限：团队/共享事件的跨用户提醒权限模型。
10. 多渠道拓展：短信 / 推送通知。

## 15. API 示例

### 创建提醒

```bash
curl -X POST $BASE/api/reminders \
 -H "Authorization: Bearer $TOKEN" \
 -H "Content-Type: application/json" \
 -d '{
  "event_id": "64f123456789abcdef012345",
  "advance_days": 0,
  "reminder_times": ["09:00", "18:30"],
  "reminder_type": "both",
  "custom_message": "今日会议别忘了"
 }'
```

### 测试提醒 (5 秒后计划 + 即时邮件尝试)

```bash
curl -X POST $BASE/api/reminders/test \
 -H "Authorization: Bearer $TOKEN" \
 -H "Content-Type: application/json" \
 -d '{
   "event_id": "64f123456789abcdef012345",
   "message": "这是一条测试提醒",
   "delay_seconds": 5,
   "send_email": true
 }'
```

### 轻量列表

```bash
curl -H "Authorization: Bearer $TOKEN" "$BASE/api/reminders/simple?active_only=true&limit=30"
```

### Snooze 90 分钟

```bash
curl -X POST $BASE/api/reminders/64f123456789abcdef012345/snooze \
 -H "Authorization: Bearer $TOKEN" \
 -H "Content-Type: application/json" \
 -d '{"snooze_minutes":90}'
```

## 16. 与旧文档差异对照
| 旧文档字段 | 现实现 | 说明 |
|------------|--------|------|
| `type` | `reminder_type` | 统一 snake_case |
| `send_time` | (派生) `next_send` | 不再直接由客户端传送时间；通过规则计算 |
| `is_enabled` | `is_active` | 名称微调 |
| 直接单次时间 | `advance_days + reminder_times` | 支持组合与多时刻 |
| 无测试接口 | `/api/reminders/test` | 便于功能验证 |

## 17. 快速脑图（文本式）
```
ReminderHandlers
  -> ReminderService
       -> Mongo(reminders, events)
  -> NotificationService (+Hub)
  -> Email (SendGeneric)
Scheduler (1m ticker)
  -> ReminderService.GetPendingReminders
  -> sendReminder(app/email/both)
  -> MarkReminderSent -> Recompute next_send
Test Reminder Path
  -> CreateImmediateTestReminder (direct insert + next_send = now+delay)
  -> Immediate email attempt
```

---
**最后更新**: 2025-08-10
