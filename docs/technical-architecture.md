# TodoIng 技术架构（更新：Repository + Preview/Test + Proto SSOT）

> 本文件已再次同步当前 dev：引入 Event / Reminder Repository、提醒 Preview & Immediate Test、事件推进级联、统一聚合 Debug。旧内容中有关“无 repository 层”描述已修订。

## 1. 当前形态

- 单体 Go API (REST + gRPC Gateway) + React 前端 + MongoDB
- 事件 & 提醒采用 Repository 下沉复杂逻辑；其余领域逐步迁移
- Proto 为接口单一真实来源（生成 gRPC + Gateway + OpenAPI）

## 2. 更新后的后端结构

```text
internal/
  api/            # Handlers (auth, tasks, events, reminders, unified, reports, notifications, dashboard)
  services/       # 薄服务：EventService/ReminderService 调用仓储；Unified 聚合；Scheduler
  repository/     # EventRepository / ReminderRepository
  models/         # 数据模型 + 计算 (CalculateNextSendTime / GetNextOccurrence)
  notifications/  # SSE Hub
  email/          # 邮件发送封装
  observability/  # 日志
```

## 3. Repository 特性

| 仓储 | 关键方法 | 说明 |
|------|----------|------|
| EventRepository | Advance / ListUpcoming / CalendarRange / Search / ListStartingWindow / MarkTriggered | 推进+级联重算提醒+写系统时间线 |
| ReminderRepository | Preview / CreateImmediateTest / Pending / Upcoming / Snooze / ToggleActive / MarkSent | 预览不落库 + 测试提醒即时发送支持 |

## 4. 提醒扩展

- Preview: 计算下一次发送 (advance_days + reminder_times) → 返回文本与时间
- Immediate Test: 直接插入 reminder + 允许立刻邮件尝试
- Scheduler: 1m ticker → Pending() → 发送 → MarkSent() → 重算 next_send

## 5. 统一聚合改进

- Upcoming：新增 debug 模式，返回窗口过滤器与统计计数
- Calendar：聚合 events + reminders(next_send) + tasks(有日期)

## 6. 事件推进流程 (Advance)

1. 读取事件 → 判断 recurrence_type
2. 计算下一日期或关闭 (is_active=false)
3. 更新事件 + 写系统时间线 (event_comments)
4. 异步遍历相关 reminders：重算 next_send（仅相对事件型）

## 7. Proto 驱动 API

```bash
make proto-all   # 生成 pb + gateway + swagger
```

输出：`pkg/api/v1/*` + `docs/swagger/todoing.swagger.json`

## 8. 未迁移领域（后续）

| 领域 | 状态 | 计划 |
|------|------|------|
| Tasks | 直接 Mongo | 引入 TaskRepository（排序与筛选聚合） |
| Reports | 直接 Mongo | 导出/统计逻辑下沉 repository |
| Notifications | 直接 Mongo | 简单写入保留 |

## 9. 指标 & 优化计划

- 建议索引：`reminders(is_active,next_send)` / `events(user_id,event_date,is_active)`
- 计划：Prometheus 指标 + repository instrumentation + 缓存 upcoming 结果

## 10. 后续 Roadmap 摘要

- UserRepository (统一邮箱获取 / 用户查询)
- 复杂循环 (weekly pattern, monthly day rules, exclusion dates)
- Reminder 批量重算管道 (事件批量编辑)
- SSE -> WebSocket 可插拔
- Metrics / Tracing / Rate Limiter 中间件

---

最后更新：同步仓储/预览/统一聚合调试版本
