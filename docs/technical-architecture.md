# TodoIng 技术架构文档

## 概述

TodoIng 是一个基于现代技术栈构建的任务管理系统，采用前后端分离架构，使用 Golang 作为后端主要技术栈，React + TypeScript 作为前端技术栈。

## 技术架构

### 整体架构设计

```text
┌─────────────────┐     HTTP/HTTPS     ┌─────────────────┐
│                 │◄──────────────────►│                 │
│   React 前端    │    RESTful API     │  Golang 后端   │
│   (TypeScript)  │                    │                 │
│                 │◄──────────────────►│                 │
└─────────────────┘     WebSocket      └─────────────────┘
         │                                       │
         │                                       │
         ▼                                       ▼
┌─────────────────┐                    ┌─────────────────┐
│                 │                    │                 │
│  静态资源存储   │                    │   MongoDB       │
│   (可选)        │                    │   数据库        │
│                 │                    │                 │
└─────────────────┘                    └─────────────────┘
                                                │
                                                │
                                                ▼
                                       ┌─────────────────┐
                                       │                 │
                                       │   第三方服务    │
                                       │ • OpenAI API    │
                                       │ • 邮件服务      │
                                       │ • 文件存储      │
                                       └─────────────────┘
```

## 前端架构

### 技术栈

| 技术 | 版本 | 描述 | 用途 |
|------|------|------|------|
| **React** | 18+ | 现代化UI框架 | 用户界面构建 |
| **TypeScript** | 5+ | 静态类型检查 | 代码质量保证 |
| **Redux Toolkit** | 1.9+ | 状态管理 | 全局状态管理 |
| **React Router** | v6 | 前端路由 | SPA路由管理 |
| **Bootstrap** | 5+ | UI组件库 | 界面样式 |
| **Vite** | 4+ | 构建工具 | 快速开发构建 |
| **Axios** | 1.4+ | HTTP客户端 | API请求 |
| **i18next** | 22+ | 国际化 | 多语言支持 |

### 目录结构

```text
frontend/
├── public/                 # 静态资源
├── src/
│   ├── components/         # 可复用组件
│   ├── pages/             # 页面组件
│   ├── store/             # Redux store
│   ├── services/          # API服务
│   ├── hooks/             # 自定义hooks
│   ├── utils/             # 工具函数
│   ├── types/             # TypeScript类型定义
│   ├── i18n/              # 国际化配置
│   └── assets/            # 静态资源
├── package.json
├── tsconfig.json
├── vite.config.ts
└── Dockerfile
```

### 状态管理

使用 Redux Toolkit 进行状态管理：

- **Auth Store**: 用户认证状态
- **Todo Store**: 任务管理状态  
- **UI Store**: 界面状态（主题、语言等）
- **Report Store**: 报告数据状态

## 后端架构 (Golang)

### 技术栈

| 技术 | 版本 | 描述 | 用途 |
|------|------|------|------|
| **Go** | 1.23+ | 编程语言 | 后端主要开发语言 |
| **Gin/Gorilla Mux** | - | Web框架 | HTTP路由和中间件 |
| **MongoDB Driver** | 1.15+ | 数据库驱动 | 数据库连接和操作 |
| **JWT-Go** | 5.2+ | JWT处理 | 用户认证 |
| **gRPC** | 1.74+ | RPC框架 | 微服务通信 (可选) |
| **Swagger** | 1.16+ | API文档 | 自动生成API文档 |
| **Zap** | 1.27+ | 日志库 | 结构化日志 |
| **Viper** | 1.18+ | 配置管理 | 配置文件处理 |

### 目录结构

```text
backend-go/
├── cmd/                   # 应用程序入口
│   └── server/
├── internal/              # 私有应用程序代码
│   ├── handler/           # HTTP处理器
│   ├── service/           # 业务逻辑
│   ├── repository/        # 数据访问层
│   ├── model/            # 数据模型
│   ├── middleware/       # 中间件
│   └── config/           # 配置
├── pkg/                  # 可重用的包
│   ├── auth/             # 认证相关
│   ├── email/            # 邮件服务
│   ├── logger/           # 日志
│   └── validator/        # 验证器
├── api/                  # API定义
│   ├── rest/             # REST API
│   └── grpc/             # gRPC定义 (可选)
├── docs/                 # Swagger文档
├── scripts/              # 构建脚本
├── test/                 # 测试文件
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

### 架构层次

```text
┌─────────────────────────────────────────┐
│              HTTP Layer                 │
│  (Gin/Gorilla Mux + Middleware)       │
└─────────────────────────────────────────┘
                     │
┌─────────────────────────────────────────┐
│             Handler Layer               │
│        (REST API Controllers)          │
└─────────────────────────────────────────┘
                     │
┌─────────────────────────────────────────┐
│             Service Layer               │
│          (Business Logic)              │
└─────────────────────────────────────────┘
                     │
┌─────────────────────────────────────────┐
│           Repository Layer              │
│         (Data Access Layer)            │
└─────────────────────────────────────────┘
                     │
┌─────────────────────────────────────────┐
│            Database Layer               │
│              (MongoDB)                 │
└─────────────────────────────────────────┘
```

## 数据库设计

### MongoDB 集合设计

#### Users 集合
```javascript
{
  "_id": ObjectId,
  "username": String,           // 用户名
  "email": String,             // 邮箱 (唯一)
  "password": String,          // 加密密码
  "role": String,              // 用户角色
  "avatar": String,            // 头像URL
  "settings": {                // 用户设置
    "language": String,
    "theme": String,
    "timezone": String
  },
  "email_verified": Boolean,   // 邮箱验证状态
  "created_at": Date,
  "updated_at": Date
}
```

#### Todos 集合
```javascript
{
  "_id": ObjectId,
  "title": String,             // 任务标题
  "description": String,       // 任务描述
  "status": String,            // 状态: pending/in_progress/completed
  "priority": String,          // 优先级: low/medium/high
  "tags": [String],           // 标签数组
  "user_id": ObjectId,        // 关联用户ID
  "due_date": Date,           // 截止日期
  "completed_at": Date,       // 完成时间
  "history": [{               // 变更历史
    "action": String,         // 操作类型
    "changes": Object,        // 变更内容
    "timestamp": Date,
    "user_id": ObjectId
  }],
  "created_at": Date,
  "updated_at": Date
}
```

#### Reports 集合
```javascript
{
  "_id": ObjectId,
  "user_id": ObjectId,        // 关联用户ID
  "type": String,             // 报告类型: daily/weekly/monthly
  "period": {                 // 报告周期
    "start": Date,
    "end": Date
  },
  "content": String,          // 报告内容
  "ai_enhanced": Boolean,     // 是否AI增强
  "statistics": {             // 统计数据
    "total_tasks": Number,
    "completed_tasks": Number,
    "pending_tasks": Number
  },
  "created_at": Date
}
```

## 安全设计

### 认证与授权

1. **JWT Token认证**
   - 使用HS256算法签名
   - Token过期时间: 24小时
   - 支持Token刷新机制

2. **密码安全**
   - 使用bcrypt加密
   - 密码强度验证
   - 支持密码重置

3. **API安全**
   - CORS跨域保护
   - Rate Limiting防止API滥用
   - 输入验证和SQL注入防护

### 数据保护

1. **数据加密**
   - 敏感数据加密存储
   - HTTPS传输加密
   - 数据库连接加密

2. **访问控制**
   - 基于角色的访问控制(RBAC)
   - 资源级权限控制
   - 审计日志记录

## 部署架构

### Docker容器化

```yaml
# docker-compose.golang.yml
services:
  backend:
    image: todoing-backend-go
    ports:
      - "5004:5004"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/todoing
      - JWT_SECRET=${JWT_SECRET}
    
  frontend:
    image: todoing-frontend
    ports:
      - "80:80"
    depends_on:
      - backend
      
  mongodb:
    image: mongo:7
    volumes:
      - mongodb_data:/data/db
```

### 生产环境部署

1. **负载均衡**: Nginx反向代理
2. **数据库**: MongoDB集群
3. **缓存**: Redis (可选)
4. **监控**: Prometheus + Grafana
5. **日志**: ELK Stack

## 开发工作流

### 环境配置

1. **开发环境**
   ```bash
   # 后端开发
   cd backend-go
   go run cmd/server/main.go
   
   # 前端开发  
   cd frontend
   npm run dev
   ```

2. **测试环境**
   ```bash
   # 单元测试
   go test ./...
   npm test
   
   # 集成测试
   docker-compose -f docker-compose.dev.yml up
   ```

3. **生产环境**
   ```bash
   # 构建部署
   docker-compose -f docker-compose.golang.yml up -d
   ```

### 代码规范

1. **Go代码规范**
   - 使用gofmt格式化
   - 使用golint检查
   - 遵循Go最佳实践

2. **TypeScript代码规范**
   - 使用ESLint + Prettier
   - 遵循React最佳实践
   - 使用TypeScript严格模式

## 性能优化

### 后端优化

1. **数据库优化**
   - 合理设计索引
   - 查询优化
   - 连接池管理

2. **缓存策略**
   - Redis缓存
   - 内存缓存
   - CDN缓存

### 前端优化

1. **构建优化**
   - 代码分割
   - 懒加载
   - 资源压缩

2. **运行时优化**
   - 虚拟滚动
   - 防抖节流
   - 内存优化

## 监控与日志

### 应用监控

1. **性能监控**
   - API响应时间
   - 数据库查询性能
   - 内存和CPU使用率

2. **业务监控**
   - 用户行为分析
   - 错误率统计
   - 功能使用情况

### 日志管理

1. **结构化日志**
   - 使用Zap记录结构化日志
   - 日志级别管理
   - 日志轮转和归档

2. **错误追踪**
   - 错误堆栈记录
   - 用户操作链路追踪
   - 异常告警机制

---

*该文档将随着项目发展持续更新，确保架构设计与实际实现保持一致。*
