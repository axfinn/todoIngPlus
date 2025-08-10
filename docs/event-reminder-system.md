# 事件记录与提醒系统技术方案

## 1. 系统概述

事件记录与提醒系统是 TodoIng 项目的扩展功能，旨在提供：
- **事件管理**：创建和管理纪念日、重要事件等
- **智能提醒**：多种提醒方式和频率设置
- **任务优先级**：基于时间的智能任务排序
- **循环事件**：支持年度、月度等重复事件

## 2. 功能需求

### 2.1 事件管理
- [x] 创建事件（纪念日、生日、节日等）
- [x] 设置事件类型和重要程度
- [x] 支持一次性和循环事件
- [x] 事件描述和备注
- [x] 事件分类和标签

### 2.2 提醒系统
- [x] 多种提醒时间设置（提前N天、当天、多次提醒）
- [x] 提醒方式：应用内通知、邮件提醒
- [x] 自定义提醒文案
- [x] 暂停/恢复提醒功能

### 2.3 智能任务排序
- [x] 近期任务优先显示（7天内）
- [x] 基于截止时间的动态排序
- [x] 重要程度权重计算
- [x] 用户自定义显示数量

## 3. 技术架构

### 3.1 系统架构图

```mermaid
graph TB
    subgraph "前端层"
        A[React 组件]
        B[Redux Store]
        C[事件日历组件]
        D[提醒通知组件]
    end
    
    subgraph "后端层"
        E[REST API]
        F[事件服务]
        G[提醒服务]
        H[任务排序服务]
    end
    
    subgraph "数据层"
        I[MongoDB]
        J[事件集合]
        K[提醒集合]
        L[任务集合]
    end
    
    subgraph "定时任务"
        M[Cron 调度器]
        N[提醒检查任务]
        O[事件更新任务]
    end
    
    A --> E
    B --> A
    C --> B
    D --> B
    
    E --> F
    E --> G
    E --> H
    
    F --> I
    G --> I
    H --> I
    
    I --> J
    I --> K
    I --> L
    
    M --> N
    M --> O
    N --> G
    O --> F
```

### 3.2 数据流程图

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as 前端
    participant A as API网关
    participant E as 事件服务
    participant R as 提醒服务
    participant S as 排序服务
    participant D as 数据库
    participant C as 定时任务
    
    Note over U,D: 创建事件流程
    U->>F: 创建事件
    F->>A: POST /api/events
    A->>E: 处理事件创建
    E->>D: 保存事件数据
    E->>R: 创建提醒规则
    R->>D: 保存提醒设置
    
    Note over C,U: 提醒检查流程
    C->>R: 定时检查提醒
    R->>D: 查询待提醒事件
    R->>F: 发送实时通知
    R->>U: 邮件提醒(可选)
    
    Note over U,S: 任务排序流程
    U->>F: 请求任务列表
    F->>A: GET /api/tasks
    A->>S: 智能排序处理
    S->>D: 查询任务和事件
    S->>F: 返回排序结果
```

## 4. 数据库设计

### 4.1 事件数据模型

```mermaid
erDiagram
    EVENTS {
        string id PK
        string user_id FK
        string title
        string description
        string event_type
        datetime event_date
        string recurrence_type
        json recurrence_config
        int importance_level
        string[] tags
        datetime created_at
        datetime updated_at
        boolean is_active
    }
    
    REMINDERS {
        string id PK
        string event_id FK
        string user_id FK
        int advance_days
        string[] reminder_times
        string reminder_type
        string custom_message
        boolean is_active
        datetime last_sent
        datetime next_send
    }
    
    TASKS {
        string id PK
        string user_id FK
        string title
        string description
        datetime due_date
        int priority_score
        string status
        datetime created_at
        datetime updated_at
    }
    
    EVENTS ||--o{ REMINDERS : "has"
    USERS ||--o{ EVENTS : "creates"
    USERS ||--o{ TASKS : "owns"
```

### 4.2 Golang 数据结构

```go
// 事件模型
type Event struct {
    ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID           primitive.ObjectID `bson:"user_id" json:"user_id"`
    Title            string            `bson:"title" json:"title"`
    Description      string            `bson:"description" json:"description"`
    EventType        string            `bson:"event_type" json:"event_type"` // birthday, anniversary, holiday, custom
    EventDate        time.Time         `bson:"event_date" json:"event_date"`
    RecurrenceType   string            `bson:"recurrence_type" json:"recurrence_type"` // none, yearly, monthly, weekly
    RecurrenceConfig map[string]interface{} `bson:"recurrence_config" json:"recurrence_config"`
    ImportanceLevel  int               `bson:"importance_level" json:"importance_level"` // 1-5
    Tags             []string          `bson:"tags" json:"tags"`
    CreatedAt        time.Time         `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time         `bson:"updated_at" json:"updated_at"`
    IsActive         bool              `bson:"is_active" json:"is_active"`
}

// 提醒模型
type Reminder struct {
    ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    EventID       primitive.ObjectID `bson:"event_id" json:"event_id"`
    UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
    AdvanceDays   int               `bson:"advance_days" json:"advance_days"`
    ReminderTimes []string          `bson:"reminder_times" json:"reminder_times"` // ["09:00", "18:00"]
    ReminderType  string            `bson:"reminder_type" json:"reminder_type"` // app, email, both
    CustomMessage string            `bson:"custom_message" json:"custom_message"`
    IsActive      bool              `bson:"is_active" json:"is_active"`
    LastSent      *time.Time        `bson:"last_sent" json:"last_sent"`
    NextSend      *time.Time        `bson:"next_send" json:"next_send"`
}

// 任务排序配置
type TaskSortConfig struct {
    PriorityDays    int     `json:"priority_days"`    // 优先显示天数
    MaxDisplayCount int     `json:"max_display_count"` // 最大显示数量
    WeightUrgent    float64 `json:"weight_urgent"`    // 紧急权重
    WeightImportant float64 `json:"weight_important"` // 重要权重
}
```

## 5. API 设计

### 5.1 事件管理 API

```mermaid
graph LR
    subgraph "事件 API"
        A[POST /api/events] --> A1[创建事件]
        B[GET /api/events] --> B1[获取事件列表]
        C[GET /api/events/:id] --> C1[获取事件详情]
        D[PUT /api/events/:id] --> D1[更新事件]
        E[DELETE /api/events/:id] --> E1[删除事件]
        F[GET /api/events/calendar] --> F1[获取日历视图]
    end
```

### 5.2 提醒管理 API

```mermaid
graph LR
    subgraph "提醒 API"
        A[POST /api/reminders] --> A1[创建提醒]
        B[GET /api/reminders] --> B1[获取提醒列表]
        C[PUT /api/reminders/:id] --> C1[更新提醒]
        D[DELETE /api/reminders/:id] --> D1[删除提醒]
        E[POST /api/reminders/:id/snooze] --> E1[暂停提醒]
        F[GET /api/reminders/upcoming] --> F1[获取即将到来的提醒]
    end
```

### 5.3 智能排序 API

```mermaid
graph LR
    subgraph "排序 API"
        A[GET /api/tasks/smart] --> A1[智能排序任务]
        B[GET /api/tasks/priority] --> B1[优先级任务]
        C[POST /api/tasks/sort-config] --> C1[保存排序配置]
        D[GET /api/dashboard/upcoming] --> D1[获取看板数据]
    end
```

## 6. 前端组件设计

### 6.1 组件架构

```mermaid
graph TB
    subgraph "页面组件"
        A[EventsPage] --> B[EventList]
        A --> C[EventCalendar]
        A --> D[CreateEventModal]
        
        E[RemindersPage] --> F[ReminderList]
        E --> G[ReminderSettings]
        
        H[DashboardPage] --> I[UpcomingEvents]
        H --> J[PriorityTasks]
        H --> K[EventSummary]
    end
    
    subgraph "共享组件"
        L[EventCard]
        M[ReminderBadge]
        N[PriorityIndicator]
        O[DatePicker]
        P[RecurrenceSelector]
    end
    
    B --> L
    C --> L
    F --> M
    J --> N
    D --> O
    D --> P
```

### 6.2 状态管理

```mermaid
graph TB
    subgraph "Redux Store"
        A[eventsSlice] --> A1[events: Event[]]
        A --> A2[loading: boolean]
        A --> A3[selectedEvent: Event]
        
        B[remindersSlice] --> B1[reminders: Reminder[]]
        B --> B2[upcomingCount: number]
        
        C[tasksSlice] --> C1[priorityTasks: Task[]]
        C --> C2[sortConfig: TaskSortConfig]
        
        D[uiSlice] --> D1[showCreateModal: boolean]
        D --> D2[notifications: Notification[]]
    end
```

## 7. 实现计划

### 7.1 开发阶段

```mermaid
gantt
    title 事件提醒系统开发计划
    dateFormat  YYYY-MM-DD
    section 后端开发
    数据模型设计      :done, des1, 2025-08-10, 1d
    API 接口开发      :active, api1, 2025-08-11, 3d
    提醒服务开发      :reminder1, after api1, 2d
    排序算法实现      :sort1, after reminder1, 2d
    定时任务开发      :cron1, after sort1, 1d
    
    section 前端开发
    组件设计          :front1, 2025-08-14, 2d
    页面开发          :front2, after front1, 3d
    状态管理          :front3, after front2, 2d
    API 集成          :front4, after front3, 2d
    
    section 测试部署
    单元测试          :test1, after front4, 2d
    集成测试          :test2, after test1, 1d
    部署上线          :deploy1, after test2, 1d
```

### 7.2 技术要点

#### 7.2.1 循环事件处理
- 年度循环：生日、纪念日
- 月度循环：月度检查、还款提醒
- 自定义循环：自定义间隔天数

#### 7.2.2 智能排序算法
```go
// 任务优先级计算
func CalculateTaskPriority(task Task, config TaskSortConfig) float64 {
    now := time.Now()
    daysUntilDue := task.DueDate.Sub(now).Hours() / 24
    
    // 紧急程度 (越接近截止时间分数越高)
    urgencyScore := math.Max(0, (config.PriorityDays - daysUntilDue) / config.PriorityDays)
    
    // 重要程度 (用户设定的重要性)
    importanceScore := float64(task.ImportanceLevel) / 5.0
    
    // 综合分数
    totalScore := urgencyScore * config.WeightUrgent + importanceScore * config.WeightImportant
    
    return totalScore
}
```

#### 7.2.3 提醒调度
- 使用 Golang 的 `time.Ticker` 进行定时检查
- 支持多时间点提醒（早上、中午、晚上）
- 避免重复提醒的去重机制

## 8. 性能考虑

### 8.1 数据库优化
- 事件表按用户ID和日期建立复合索引
- 提醒表按下次发送时间建立索引
- 任务表按截止时间和优先级建立索引

### 8.2 缓存策略
- Redis 缓存用户的排序配置
- 缓存近期事件和提醒数据
- 前端本地缓存事件列表

### 8.3 实时通知
- WebSocket 连接用于实时推送提醒
- 支持离线消息队列
- 移动端推送通知支持

## 9. 安全考虑

- 用户只能访问自己的事件和提醒
- API 请求需要 JWT 认证
- 敏感信息加密存储
- 防止暴力创建大量事件

## 10. 扩展功能

- [ ] 事件分享功能
- [ ] 团队共享事件
- [ ] 事件导入/导出
- [ ] 第三方日历集成
- [ ] 移动端 App 支持
- [ ] 语音提醒功能

---

**技术方案总结：**
这个事件记录与提醒系统将显著提升 TodoIng 的用户体验，通过智能化的任务排序和多样化的提醒方式，帮助用户更好地管理时间和重要事件。系统采用模块化设计，易于扩展和维护。
