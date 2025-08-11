# 🚀 TodoIng Backend - Go 版本

<div align="center">

高性能的 Go 语言任务 / 事件 / 提醒 / 报表 后端服务

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![gRPC](https://img.shields.io/badge/gRPC-Gateway-green.svg)](https://grpc.io/)
[![MongoDB](https://img.shields.io/badge/MongoDB-5.0+-green.svg)](https://mongodb.com/)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://docker.com/)
[![API Docs](https://img.shields.io/badge/OpenAPI-Auto-orange.svg)](http://localhost:5004/swagger/)

</div>

## ✨ 功能概述

TodoIng Go 后端是原 Node 版本的性能 & 架构升级实现，现已包含完整的事件 & 提醒调度体系与统一聚合视图。

### 🎯 核心功能

- ✅ 用户认证（注册 / 登录 / 邮箱验证码 / 图形验证码）
- ✅ 任务管理（CRUD / 导入导出 / 优先级排序）
- ✅ 事件管理（循环推进 / 时间线系统记录 / 搜索 / 下拉选项 / 日历）
- ✅ 提醒系统（多时间点 / 提前天数 / Snooze / Toggle / Upcoming / 简化列表）
- ✅ 提醒预览 & 测试接口（/preview 不落库、/test 即时创建并可立刻尝试发送邮件）
- ✅ 报表生成（日报 / 周报 / 月报 + Markdown / CSV 导出 + 可扩展 AI 润色）
- ✅ 统一聚合 Upcoming & Calendar（任务 + 事件 + 提醒合并视图，支持调试窗口）
- ✅ SSE 通知推送（提醒发送 / 系统事件通知）
- ✅ gRPC + HTTP 双入口（Proto 单一真实来源，自动生成 OpenAPI）

### 🧱 新的 Repository 分层（部分落地）

当前已经对高复杂度领域实施仓储重构：

| 领域 | Repository | 内容 | 说明 |
|------|------------|------|------|
| Event | ✔ | 日期推进 / 循环 / 日历 / 搜索 / 系统时间线 | Service 纯组合，不含 bson |
| Reminder | ✔ | 聚合 Lookup / next_send 计算 / Pending / Preview / ImmediateTest | Service 仅做轻度校验 |
| Task / Report / Notification | ⏳ | 仍直接集合操作 | 后续按需迁移 |

> 目标：复杂 Mongo 聚合与级联更新全部下沉 repository，Service 保持“参数装配 + 调用 + 轻度校验”。

### 🔌 API 架构

- REST (gorilla/mux) + gRPC(gateway) 同步暴露
- Proto -> pb.go + gateway + swagger（`make proto-all`）
- OpenAPI 文档：`docs/swagger/todoing.swagger.json`
- 增强的分页 / 搜索 / 预览 / 调试输出

## 🏗️ 技术架构

```text
Handlers -> Services (薄) -> Repositories (Event/Reminder) -> Mongo
                |-> Scheduler (Reminders)
                |-> Unified Aggregator
                |-> Notifications (SSE)
```

关键点：

- Event/Reminder Service 已无直接 `bson` 依赖
- 事件推进 `Advance`：更新事件 -> 异步重算相关提醒 -> 写系统 Comment
- 提醒发送：Scheduler 使用仓储 `Pending()` + `MarkSent()`；支持 Snooze / Toggle / ImmediateTest / Preview

## 📂 项目结构（节选）

```text
backend-go/
  cmd/api / cmd/grpc          # 启动入口
  internal/
    api/                      # HTTP handlers
    services/                 # 薄业务 + unified + scheduler
    repository/               # Event / Reminder 仓储实现
    models/                   # 领域模型 + 计算逻辑
    notifications/            # SSE Hub
    email/                    # 邮件发送封装
    observability/            # 日志
  api/proto/v1/               # *.proto (SSOT)
  pkg/api/v1/                 # 生成的 gRPC + Gateway 代码
  docs/swagger/               # OpenAPI 输出
```

## ⏰ 提醒系统快速参考

| 能力 | 路径 | 说明 |
|------|------|------|
| 创建 | POST /api/reminders | advance_days + reminder_times |
| 预览 | POST /api/reminders/preview | 不落库，返回 next_send / 文本 |
| 测试 | POST /api/reminders/test | 立即或延迟创建 + 可即时邮件 |
| 即将到来 | GET /api/reminders/upcoming?hours=48 | 未来窗口 |
| 简化列表 | GET /api/reminders/simple | 精简字段，快速渲染 |
| Snooze | POST /api/reminders/{id}/snooze | 推迟 N 分钟 |
| Toggle | POST /api/reminders/{id}/toggle_active | 启用 / 停用 |

## 🔗 统一聚合 (Unified)

| 路径 | 说明 |
|------|------|
| GET /api/unified/upcoming | 聚合任务/事件/提醒；支持 ?sources=task,event,reminder&limit=50 |
| GET /api/unified/calendar | 按日合并 (任务有日期 / 事件 / 提醒下一次发送) |
| Debug (context flag) | 设置 unifiedDebug 返回内部窗口统计 |

## 📈 索引建议

```text
reminders: { is_active:1, next_send:1 }
events:    { user_id:1, event_date:1, is_active:1 }
```

## ♻️ 已更新内容概览

| 主题 | 状态 | 说明 |
|------|------|------|
| Event / Reminder Repository 重构 | ✔ | 复杂逻辑下沉 + Service 薄化 |
| Reminder Preview / Test | ✔ | Preview 不落库；Test 支持即时邮件 |
| Unified Debug 输出 | ✔ | 返回窗口过滤与统计信息 |
| Proto -> OpenAPI | ✔ | make proto-all 一键生成 |
| 事件推进级联提醒重算 | ✔ | Advance 异步重算相关提醒 next_send |
| gRPC 服务器 | ✔ | 与 HTTP 共享 Service / Repository |

## 版权 & License

详见根目录 `LICENSE`。若该文档与实际实现不符，请以代码为准并提交 Issue。

> 最后更新：Repository / Preview / Unified 重构同步版

