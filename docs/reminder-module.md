# 提醒模块使用与架构说明 (Repository + Preview/Test 更新)

> 本文档同步 dev：已引入 ReminderRepository（含 Preview / CreateImmediateTest / Pending 聚合），事件推进 (Advance) 异步触发相关提醒 next_send 重算；新增统一聚合对 reminders 的窗口视图支持。

## 1. 增量更新摘要

| 新能力 | 说明 |
|--------|------|
| Repository 下沉 | 所有 Mongo 聚合 / 计算（含 $lookup）集中到仓储层 |
| Preview 接口 | `/api/reminders/preview` 不落库计算 next_send + 文本描述 |
| Immediate Test | `/api/reminders/test` 立即或短延迟创建 + 可即时邮件尝试 |
| Advance 级联 | 事件推进后异步重算相关活跃提醒 next_send |
| 简化列表 ListSimple | 精简字段 + `$lookup` 事件标题，快速渲染 |
| Pending 优化 | 直接匹配 `is_active=true && next_send<=now` 聚合事件信息 |

## 2. Repository 方法

| 方法 | 用途 | 备注 |
|------|------|------|
| Insert | 创建时计算 next_send | 读取事件以派生 |
| GetWithEvent | `$lookup` 事件 | 用于详情/更新回显 |
| UpdateFields | 更新字段并可选重算 next_send | `recomputeNext=true` 时读取事件 |
| ListPagedWithEvent | 分页 + `$lookup` | 提供总页数 / 总数 |
| ListSimple | 精简列表 | 限制数量 (≤100) |
| Upcoming | 窗口内 next_send | 返回事件标题/重要级别/倒计时字段 |
| Pending | 调度扫描 | 不暴露给 HTTP 层 |
| ToggleActive | 反转 is_active | 更新 updated_at |
| Snooze | 推迟到当前时间 + minutes | 直接写 next_send |
| MarkSent | last_sent=now + 重算 next_send | 非循环事件多为 nil |
| CreateImmediateTest | 测试提醒 | next_send=now+delaySeconds |
| Preview | 计算 next_send + 文本 | 不落库 / 返回验证错误列表 |

## 3. Preview 输出结构

```jsonc
{
  "event": { "id": "...", "title": "...", "event_date": "2025-08-20T10:00:00Z" },
  "next": "2025-08-20T09:00:00Z",
  "text": "会议 @ 2025-08-20 10:00 | advance 0 day(s) | times [09:00] => next 2025-08-20 09:00",
  "validation_errs": []
}
```

## 4. Immediate Test 交互

请求：

```bash
POST /api/reminders/test
{
  "event_id": "<optional>",
  "message": "测试提醒",
  "delay_seconds": 5,
  "send_email": true
}
```

逻辑：缺 event_id 选最近未来事件 → 插入 reminder (advance_days=0, reminder_times=[HH:MM]) → 可即时邮件尝试 → 返回 `email_sent/email_error`。

## 5. 事件推进对提醒影响

`Advance(eventID)`：

1. 计算事件下一次日期或关闭
2. 写系统时间线 (event_comments)
3. 异步扫描关联 active reminders （absolute_times 为空的相对提醒）
4. 为每条重算 next_send → 更新

## 6. Unified 聚合中的提醒

- Upcoming：按窗口 (hours) 聚合，统一排序；字段 `ReminderAt` 作为统一 ScheduledAt
- Calendar：对 next_send 落在目标月的提醒进行日归档

## 7. 算法注意点

| 场景 | 行为 |
|------|------|
| advance_days > 0 | 提醒日期 = event_date - N 天（按日 UTC 计算） |
| 多 reminder_times | 取提醒日内第一个未来时间（相对当前 now） |
| 所有时间已过 | next_send = nil |
| Snooze | 强制写 next_send = now + minutes |
| Toggle 停用 | 不删除 next_send（可保留分析），扫描时被过滤 |
| 测试提醒 | 不依赖 CalculateNextSendTime |

## 8. 字段差异与命名统一

| 旧字段 | 新/现字段 | 说明 |
|--------|-----------|------|
| type | reminder_type | 明确语义 |
| send_time | next_send | 统一派生字段 |
| is_enabled | is_active | 布尔命名统一 |
| message | custom_message | 自定义提醒文本 |

## 9. 索引建议

```text
reminders: { is_active:1, next_send:1 }
reminders: { user_id:1, created_at:-1 }
```

## 10. 常见失败情况

| 情况 | 当前处理 | 计划改进 |
|------|----------|----------|
| 事件改期后提醒未重算 (手动更新日期) | 需调用 Advance 或未来实现批量重算 | 引入 UpdateEventReminders |
| 重复测试提醒堆积 | 正常调度扫描 | 标记 is_test + 定期清理 |
| 邮件发送失败 | 返回 email_error | 增加重试/队列 |
| 时区差异 | 统一 UTC 存储 | 前端本地化展示 |

## 11. API 清单（补充）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/reminders/preview | 预览 next_send |
| POST | /api/reminders/test | 创建测试提醒 |
| GET  | /api/reminders/upcoming | 即将到来窗口 |
| GET  | /api/reminders/simple | 简化列表 |
| POST | /api/reminders/{id}/snooze | 推迟 |

## 12. 未来改进

- 批量重算接口：事件批量移动 / 改期
- 更丰富循环（周模式 / 月第 N 个工作日 / 排除日期）
- 自然语言时间解析下沉后端
- Prometheus 指标：pending 数、发送耗时、失败率
- WebSocket 替换或并行 SSE

---
最后更新：含 Repository / Preview / ImmediateTest / Advance 级联版本
