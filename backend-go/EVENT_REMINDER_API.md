# 事件与提醒系统 API 文档

## 概述

本系统实现了事件记录和智能提醒功能，包含以下主要组件：

1. **事件管理** - 支持创建、查看、编辑日程事件
2. **智能提醒** - 基于事件的自动提醒系统  
3. **仪表板** - 智能任务排序和数据聚合

## API 端点

### 事件管理 (`/api/events`)

#### 创建事件
- **POST** `/api/events`
- **认证**: 需要 Bearer Token
- **请求体**:
```json
{
  "title": "会议标题",
  "description": "会议描述",
  "date": "2024-01-15T10:00:00Z",
  "location": "会议室A",
  "recurrence": {
    "enabled": true,
    "pattern": "weekly",
    "interval": 1,
    "end_date": "2024-12-31T00:00:00Z"
  }
}
```

#### 获取事件列表
- **GET** `/api/events`
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `page`: 页码 (默认: 1)
  - `limit`: 每页数量 (默认: 20)
  - `start_date`: 开始日期过滤
  - `end_date`: 结束日期过滤

#### 获取单个事件
- **GET** `/api/events/{id}`
- **认证**: 需要 Bearer Token

#### 更新事件  
- **PUT** `/api/events/{id}`
- **认证**: 需要 Bearer Token

#### 删除事件
- **DELETE** `/api/events/{id}`
- **认证**: 需要 Bearer Token

#### 获取即将到来的事件
- **GET** `/api/events/upcoming`
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `days`: 未来天数 (默认: 7)

#### 搜索事件
- **GET** `/api/events/search`
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `q`: 搜索关键词

#### 获取日历视图
- **GET** `/api/events/calendar`
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `year`: 年份
  - `month`: 月份

### 提醒管理 (`/api/reminders`)

#### 创建提醒
- **POST** `/api/reminders`
- **认证**: 需要 Bearer Token
- **请求体**:
```json
{
  "event_id": "64f123...",
  "type": "email",
  "send_time": "2024-01-15T09:00:00Z",
  "message": "自定义提醒消息",
  "is_enabled": true
}
```

#### 获取提醒列表
- **GET** `/api/reminders`
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `page`: 页码
  - `limit`: 每页数量
  - `event_id`: 事件ID过滤

#### 获取单个提醒
- **GET** `/api/reminders/{id}`
- **认证**: 需要 Bearer Token

#### 更新提醒
- **PUT** `/api/reminders/{id}`
- **认证**: 需要 Bearer Token

#### 删除提醒
- **DELETE** `/api/reminders/{id}`
- **认证**: 需要 Bearer Token

#### 获取即将发送的提醒
- **GET** `/api/reminders/upcoming`
- **认证**: 需要 Bearer Token

#### 暂停提醒
- **POST** `/api/reminders/{id}/snooze`
- **认证**: 需要 Bearer Token
- **请求体**:
```json
{
  "minutes": 30
}
```

### 仪表板 (`/api/dashboard`)

#### 获取仪表板数据
- **GET** `/api/dashboard`
- **认证**: 需要 Bearer Token
- **响应**:
```json
{
  "upcoming_events": [...],
  "priority_tasks": [...],
  "pending_reminders": [...],
  "stats": {
    "total_events": 10,
    "total_tasks": 25,
    "pending_reminders": 5
  }
}
```

#### 获取优先任务
- **GET** `/api/dashboard/tasks`
- **认证**: 需要 Bearer Token

#### 获取任务排序配置
- **GET** `/api/dashboard/config`
- **认证**: 需要 Bearer Token

#### 更新任务排序配置
- **PUT** `/api/dashboard/config`
- **认证**: 需要 Bearer Token
- **请求体**:
```json
{
  "priority_days": 7,
  "max_display_count": 20,
  "weight_urgent": 0.7,
  "weight_important": 0.3
}
```

## 认证

所有 API 端点都需要 JWT Bearer Token 认证：

```
Authorization: Bearer <your-jwt-token>
```

## 响应格式

### 成功响应
- **状态码**: 200, 201
- **Content-Type**: `application/json`

### 错误响应
- **状态码**: 400, 401, 404, 500
- **Content-Type**: `application/json`
- **格式**:
```json
{
  "error": "错误描述信息"
}
```

## 数据模型

### Event 事件模型
```json
{
  "id": "64f123...",
  "user_id": "64f456...",
  "title": "事件标题",
  "description": "事件描述",
  "date": "2024-01-15T10:00:00Z",
  "location": "地点",
  "recurrence": {
    "enabled": true,
    "pattern": "weekly",
    "interval": 1,
    "end_date": "2024-12-31T00:00:00Z"
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Reminder 提醒模型
```json
{
  "id": "64f789...",
  "user_id": "64f456...",
  "event_id": "64f123...",
  "type": "email",
  "send_time": "2024-01-15T09:00:00Z",
  "message": "提醒消息",
  "is_enabled": true,
  "is_sent": false,
  "sent_at": null,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### TaskSortConfig 任务排序配置模型
```json
{
  "user_id": "64f456...",
  "priority_days": 7,
  "max_display_count": 20,
  "weight_urgent": 0.7,
  "weight_important": 0.3,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

## 使用示例

### 创建事件和提醒的完整流程

1. **创建事件**:
```bash
curl -X POST http://localhost:5001/api/events \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "项目会议",
    "description": "讨论Q1计划",
    "date": "2024-01-15T10:00:00Z",
    "location": "会议室A"
  }'
```

2. **为事件创建提醒**:
```bash
curl -X POST http://localhost:5001/api/reminders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "<事件ID>",
    "type": "email",
    "send_time": "2024-01-15T09:30:00Z",
    "message": "项目会议将在30分钟后开始"
  }'
```

3. **查看仪表板数据**:
```bash
curl -X GET http://localhost:5001/api/dashboard \
  -H "Authorization: Bearer <token>"
```

## 部署和启动

### 环境变量配置
```bash
MONGO_URI=mongodb://localhost:27017
JWT_SECRET=your-secret-key
PORT=5001
```

### 启动服务
```bash
cd backend-go
go run cmd/api/main.go
```

### Docker 部署
```bash
docker build -t todoing-backend .
docker run -p 5001:5001 --env-file .env todoing-backend
```

## 技术特性

- **智能提醒调度**: 基于时间的自动提醒发送
- **事件循环支持**: 支持日、周、月、年度重复事件
- **任务优先级算法**: 智能排序基于紧急度和重要性
- **分页和过滤**: 高效的数据查询和展示
- **JWT 认证**: 安全的用户身份验证
- **MongoDB 存储**: NoSQL 数据库支持
- **RESTful API**: 标准化的 HTTP 接口设计
