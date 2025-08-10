# 📐 TodoIng 系统架构设计

## 🏗️ 整体架构

```mermaid
graph TB
    subgraph "前端层 Frontend"
        A[React + TypeScript]
        B[Redux Toolkit]
        C[React Router]
        D[Axios HTTP Client]
    end
    
    subgraph "API 网关层 Gateway"
        E[Nginx Reverse Proxy]
        F[SSL/TLS 终端]
    end
    
    subgraph "后端服务层 Backend Services"
        G[Go HTTP Server]
        H[gRPC Server]
        I[JWT 认证中间件]
        J[邮件服务]
    end
    
    subgraph "数据层 Data Layer"
        K[MongoDB]
        L[Redis Cache]
        M[文件存储]
    end
    
    A --> E
    B --> A
    C --> A
    D --> A
    E --> G
    E --> H
    F --> E
    G --> I
    G --> J
    H --> I
    I --> K
    G --> L
    H --> K
    J --> M
```

## 🔄 数据流图

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as Frontend
    participant N as Nginx
    participant G as Go Backend
    participant M as MongoDB
    participant R as Redis
    
    U->>F: 用户操作
    F->>N: HTTP/HTTPS 请求
    N->>G: 转发请求
    G->>G: JWT 验证
    G->>R: 检查缓存
    alt 缓存命中
        R-->>G: 返回缓存数据
    else 缓存未命中
        G->>M: 查询数据库
        M-->>G: 返回数据
        G->>R: 更新缓存
    end
    G-->>N: 返回响应
    N-->>F: 转发响应
    F-->>U: 更新界面
```

## 🏢 服务架构

```mermaid
graph LR
    subgraph "微服务架构"
        subgraph "核心服务 Core Services"
            A1[用户服务 User Service]
            A2[任务服务 Task Service]
            A3[认证服务 Auth Service]
        end
        
        subgraph "扩展服务 Extended Services"
            B1[邮件服务 Email Service]
            B2[报告服务 Report Service]
            B3[AI 服务 AI Service]
        end
        
        subgraph "基础设施 Infrastructure"
            C1[数据库 MongoDB]
            C2[缓存 Redis]
            C3[消息队列 Message Queue]
        end
    end
    
    A1 --> C1
    A2 --> C1
    A3 --> C2
    B1 --> C3
    B2 --> A2
    B3 --> B2
```

## 📊 组件关系图

```mermaid
classDiagram
    class Frontend {
        +React Components
        +Redux Store
        +API Client
        +Router
    }
    
    class APIGateway {
        +Nginx Config
        +SSL Termination
        +Load Balancing
        +Rate Limiting
    }
    
    class GoBackend {
        +HTTP Handlers
        +gRPC Services
        +Middleware
        +Business Logic
    }
    
    class Database {
        +MongoDB Collections
        +Indexes
        +Aggregation Pipelines
    }
    
    class Cache {
        +Redis Sessions
        +API Cache
        +Rate Limit Store
    }
    
    Frontend --> APIGateway : HTTPS Requests
    APIGateway --> GoBackend : Proxy
    GoBackend --> Database : Data Operations
    GoBackend --> Cache : Session/Cache
```

## 🌐 部署架构

```mermaid
graph TB
    subgraph "生产环境 Production"
        subgraph "负载均衡层"
            LB[Load Balancer]
        end
        
        subgraph "Web 层"
            W1[Web Server 1]
            W2[Web Server 2]
        end
        
        subgraph "应用层"
            A1[App Server 1]
            A2[App Server 2]
            A3[App Server 3]
        end
        
        subgraph "数据层"
            DB1[MongoDB Primary]
            DB2[MongoDB Secondary]
            RD[Redis Cluster]
        end
    end
    
    LB --> W1
    LB --> W2
    W1 --> A1
    W1 --> A2
    W2 --> A2
    W2 --> A3
    A1 --> DB1
    A2 --> DB1
    A3 --> DB1
    DB1 --> DB2
    A1 --> RD
    A2 --> RD
    A3 --> RD
```

## 🔐 安全架构

```mermaid
graph TD
    subgraph "安全层级 Security Layers"
        L1[网络安全 Network Security]
        L2[应用安全 Application Security]
        L3[数据安全 Data Security]
        L4[访问控制 Access Control]
    end
    
    subgraph "安全组件 Security Components"
        S1[SSL/TLS 加密]
        S2[JWT 认证]
        S3[验证码系统]
        S4[邮箱验证]
        S5[密码加密]
        S6[数据库加密]
        S7[API 限流]
        S8[CORS 配置]
    end
    
    L1 --> S1
    L1 --> S7
    L1 --> S8
    L2 --> S2
    L2 --> S3
    L2 --> S4
    L3 --> S5
    L3 --> S6
    L4 --> S2
```

## 📦 模块依赖图

```mermaid
graph TD
    subgraph "前端模块 Frontend Modules"
        F1[认证模块]
        F2[任务管理模块]
        F3[报告模块]
        F4[用户设置模块]
        F5[公共组件模块]
    end
    
    subgraph "后端模块 Backend Modules"
        B1[认证中间件]
        B2[用户服务]
        B3[任务服务]
        B4[邮件服务]
        B5[报告服务]
        B6[公共工具模块]
    end
    
    F1 --> B1
    F1 --> B2
    F2 --> B3
    F3 --> B5
    F4 --> B2
    F5 --> B6
    
    B1 --> B6
    B2 --> B6
    B3 --> B6
    B4 --> B6
    B5 --> B6
```

## 🔄 开发流程图

```mermaid
gitgraph
    commit id: "初始化项目"
    branch dev
    checkout dev
    commit id: "Golang 后端重构"
    commit id: "移除 Node.js 依赖"
    branch feature/auth
    checkout feature/auth
    commit id: "实现 JWT 认证"
    commit id: "添加邮箱验证"
    checkout dev
    merge feature/auth
    branch feature/tasks
    checkout feature/tasks
    commit id: "任务 CRUD 操作"
    commit id: "任务历史追踪"
    checkout dev
    merge feature/tasks
    branch feature/reports
    checkout feature/reports
    commit id: "报告生成功能"
    commit id: "AI 集成"
    checkout dev
    merge feature/reports
    checkout main
    merge dev
    commit id: "v2.0.0 发布"
```

## 🚀 技术选型理由

```mermaid
mindmap
  root((技术选型))
    (前端)
      React
        ::icon(fa fa-react)
        组件化开发
        生态成熟
        TypeScript 支持
      Redux Toolkit
        状态管理
        开发工具完善
        异步处理
      Vite
        快速构建
        热更新
        ES 模块
    (后端)
      Golang
        ::icon(fa fa-code)
        高性能
        并发处理
        类型安全
      Gin/Echo
        轻量级框架
        中间件丰富
        易于测试
      gRPC
        高性能通信
        类型安全
        跨语言支持
    (数据库)
      MongoDB
        ::icon(fa fa-database)
        文档型数据库
        水平扩展
        JSON 原生支持
      Redis
        高性能缓存
        数据结构丰富
        持久化支持
```

## 📈 性能指标

```mermaid
xychart-beta
    title "系统性能指标"
    x-axis [响应时间, 并发用户, 数据吞吐, 可用性]
    y-axis "指标值" 0 --> 100
    bar [95, 80, 90, 99.9]
```
