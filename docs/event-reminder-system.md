# 事件记录与提醒系统（精简当前实现）

> 本文件已对原方案文档进行裁剪，仅保留已实现或正在实现的部分；高级循环、智能权重排序、团队/分享等内容暂未落地，转入 Roadmap。

## 1. 已实现功能概述

- 事件管理：基础创建/列出（`events` 集合，字段参考 `database-design.md`）。
- 提醒系统：`reminders` 集合 + 定时扫描 (`next_send` / `is_active`) + 邮件 / SSE 通知。
- 测试提醒：即时触发测试邮件（`POST /api/reminders/test`）。
- 提醒暂停/恢复：`is_active` 字段控制。
- 统一聚合：通过 `/api/unified/upcoming` & `/api/unified/calendar` 返回事件/任务/提醒组合视图。

## 2. 数据模型（当前核心字段）

```go
// Event (摘录)
type Event struct {
    ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
    Title      string             `bson:"title" json:"title"`
    EventDate  time.Time          `bson:"event_date" json:"event_date"`
    EventType  string             `bson:"event_type" json:"event_type"` // 可选：birthday/anniversary/custom
    IsActive   bool               `bson:"is_active" json:"is_active"`
    CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

// Reminder (摘录)
type Reminder struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
    EventID     *primitive.ObjectID `bson:"event_id,omitempty" json:"event_id,omitempty"`
    Title       string             `bson:"title" json:"title"`
    ReminderType string            `bson:"reminder_type" json:"reminder_type"` // email / app / both
    NextSend    *time.Time         `bson:"next_send" json:"next_send"`
    LastSent    *time.Time         `bson:"last_sent" json:"last_sent"`
    IsActive    bool               `bson:"is_active" json:"is_active"`
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
```

更多字段 / 索引建议：见 `database-design.md`。

## 3. 调度流程

1. Ticker（1 分钟）扫描：`{is_active: true, next_send: { $lte: now }}`
2. 对匹配提醒：发送邮件（优先 `REMINDER_EMAIL_*`）与 SSE（通知频道）
3. 更新 `last_sent` 并计算下一个 `next_send`（实现内含简单公式 / 重置逻辑）
4. 测试提醒：使用临时事件上下文发送，不持久化新的事件记录

## 4. API 摘要

| 功能 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 创建事件 | POST | `/api/events` | 基础事件创建 |
| 列出事件 | GET | `/api/events` | 当前用户事件列表 |
| 创建提醒 | POST | `/api/reminders` | 事件或独立提醒创建 |
| 列出提醒 | GET | `/api/reminders` | 当前用户提醒列表 |
| 测试提醒 | POST | `/api/reminders/test` | 立即测试提醒（邮件/SSE） |
| 即将提醒 | GET | `/api/reminders/upcoming` | 未来窗口提醒（已存在） |
| 暂停/恢复 | PATCH | `/api/reminders/:id` | 切换 `is_active` (实现以当前 handler 为准) |
| 统一 upcoming | GET | `/api/unified/upcoming` | 合并任务/事件/提醒 |
| 统一日历 | GET | `/api/unified/calendar` | 合并日历视图 |

> 更详细字段与查询参数：参考各 handler 文件 (`internal/api/*_handlers.go`)。

## 5. 通知通道

| 通道 | 说明 | 现状 |
|------|------|------|
| 邮件 | Send / SendGeneric | 已实现，支持专用与回退配置 |
| SSE  | `/api/notifications/stream` | 已实现，event: notification |
| WebSocket | 旧方案 | 未实现，SSE 足够当前需求 |

## 6. 已裁剪 / 未实现项

| 原方案条目 | 状态 | 说明 |
|------------|------|------|
| 多时间点提醒 (`reminder_times`) | 未实现 | 当前以单一 `next_send` 驱动 |
| advance_days 复杂组合 | 未实现 | 可在计算下一次发送时扩展 |
| 自定义权重智能排序 | 未实现 | 暂用简单时间排序（聚合服务） |
| 年/月/周循环 + recurrence_config | 部分/未实现 | 仅基础日期；复杂循环待定 |
| 任务排序服务独立模块 | 未实现 | 聚合统一服务中处理 |
| 团队共享 / 分享事件 | 未实现 | Roadmap |
| 外部日历集成 | 未实现 | Roadmap |
| 移动端推送 / 离线队列 | 未实现 | Roadmap |

## 7. Roadmap (优先级建议)

| 优先级 | 项 | 目标 |
|--------|----|------|
| 高 | 多时间点提醒 | 支持同日多时间发送（早/午/晚） |
| 高 | 循环事件配置 | yearly/monthly/weekly 简化策略 |
| 中 | 提醒发送监控 | 指标 + 失败重试策略 |
| 中 | SSE 横向扩展 | 多实例广播：Redis Pub/Sub |
| 中 | 智能排序 | 基于 due 时间 + 权重合成分值 |
| 低 | 外部日历集成 | iCal 订阅导出 / Google Calendar 同步 |
| 低 | 团队/分享 | 事件共享权限模型 |
| 低 | 移动端推送 | 推送通道 (FCM / APNs) |

## 8. 相关文档链接

- 架构概览：`architecture.md`
- 数据模型：`database-design.md`
- 提醒细节：`reminder-module.md`
- 聚合：代码 `internal/api/unified_handlers.go`

---
本文件保持与当前实现同步，扩展上线后再行更新。
