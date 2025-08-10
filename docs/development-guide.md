# TodoIng 开发指南

## 快速开始

### 环境要求

#### 系统要求
- **操作系统**: Linux, macOS, Windows
- **内存**: 最少 4GB RAM
- **存储**: 最少 10GB 可用空间

#### 软件依赖

| 软件 | 版本要求 | 用途 |
|------|----------|------|
| **Go** | 1.23+ | 后端开发 |
| **Node.js** | 18+ | 前端开发 |
| **MongoDB** | 7.0+ | 数据库 |
| **Docker** | 24+ | 容器化部署 |
| **Git** | 2.30+ | 版本控制 |

### 项目克隆与设置

```bash
# 克隆项目
git clone git@github.com:axfinn/todoIngPlus.git
cd todoIngPlus

# 切换到开发分支
git checkout dev
```

## 后端开发 (Golang)

### 环境配置

```bash
# 进入后端目录
cd backend-go

# 安装依赖
go mod download

# 复制环境配置文件
cp .env.example .env

# 编辑环境变量
vim .env
```

### 环境变量配置

```bash
# .env 文件示例
MONGO_URI=mongodb://localhost:27017/todoing
JWT_SECRET=your-super-secret-jwt-key
PORT=5004

# 邮件服务配置
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
EMAIL_FROM=TodoIng <your-email@gmail.com>

# OpenAI配置 (可选)
OPENAI_API_KEY=your-openai-api-key

# 其他配置
LOG_LEVEL=debug
CORS_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 开发命令

```bash
# 开发模式运行 (热重载)
make dev

# 或者直接运行
go run cmd/server/main.go

# 编译项目
make build

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 生成 API 文档
make docs
```

### Makefile 常用命令

```makefile
# 查看所有可用命令
make help

# 开发相关
make dev          # 开发模式运行
make build        # 编译项目
make clean        # 清理构建文件

# 测试相关
make test         # 运行所有测试
make test-unit    # 运行单元测试
make test-integration # 运行集成测试
make coverage     # 生成测试覆盖率报告

# 代码质量
make fmt          # 格式化代码
make lint         # 代码静态检查
make vet          # Go代码检查

# 文档相关
make docs         # 生成API文档
make docs-serve   # 启动文档服务器

# Docker相关
make docker-build # 构建Docker镜像
make docker-run   # 运行Docker容器
```

### 项目结构说明

```text
backend-go/
├── cmd/
│   └── server/           # 应用程序入口
│       └── main.go
├── internal/             # 私有代码，不对外暴露
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   │   ├── auth.go     # 认证相关接口
│   │   ├── todo.go     # 任务管理接口
│   │   └── report.go   # 报告生成接口
│   ├── service/         # 业务逻辑层
│   ├── repository/      # 数据访问层
│   ├── model/          # 数据模型
│   └── middleware/     # 中间件
├── pkg/                 # 可重用的包
│   ├── auth/           # 认证工具
│   ├── email/          # 邮件服务
│   ├── logger/         # 日志工具
│   └── validator/      # 验证器
├── api/                # API定义
│   └── swagger/        # Swagger文档
├── test/               # 测试文件
├── scripts/            # 构建和部署脚本
└── docs/               # 项目文档
```

### API 开发规范

#### RESTful API 设计

```go
// 路由定义示例
func (h *TodoHandler) SetupRoutes(r *mux.Router) {
    // 任务相关路由
    todoRouter := r.PathPrefix("/api/todos").Subrouter()
    todoRouter.HandleFunc("", h.CreateTodo).Methods("POST")
    todoRouter.HandleFunc("", h.GetTodos).Methods("GET")
    todoRouter.HandleFunc("/{id}", h.GetTodo).Methods("GET")
    todoRouter.HandleFunc("/{id}", h.UpdateTodo).Methods("PUT")
    todoRouter.HandleFunc("/{id}", h.DeleteTodo).Methods("DELETE")
}
```

#### 统一响应格式

```go
// 成功响应
type SuccessResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Message string      `json:"message,omitempty"`
}

// 错误响应
type ErrorResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error"`
    Code    int    `json:"code"`
}
```

#### 错误处理

```go
// 自定义错误类型
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}

func (e *AppError) Error() string {
    return e.Message
}

// 错误处理中间件
func ErrorHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                logger.Error("Panic recovered", zap.Any("error", err))
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

## 前端开发 (React + TypeScript)

### 环境配置

```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 复制环境配置文件
cp .env.example .env

# 编辑环境变量
vim .env
```

### 环境变量配置

```bash
# .env 文件示例
VITE_API_BASE_URL=http://localhost:5004/api
VITE_APP_NAME=TodoIng
VITE_VERSION=2.0.0

# 开发环境配置
VITE_DEBUG=true
VITE_LOG_LEVEL=debug
```

### 开发命令

```bash
# 开发模式运行
npm run dev

# 构建项目
npm run build

# 预览构建结果
npm run preview

# 运行测试
npm run test

# 运行测试并观察变化
npm run test:watch

# 代码检查
npm run lint

# 代码格式化
npm run format

# 类型检查
npm run type-check
```

### 项目结构说明

```text
frontend/
├── public/              # 静态资源
│   ├── index.html
│   └── favicon.ico
├── src/
│   ├── components/      # 可复用组件
│   │   ├── common/     # 通用组件
│   │   ├── forms/      # 表单组件
│   │   └── layout/     # 布局组件
│   ├── pages/          # 页面组件
│   │   ├── auth/       # 认证页面
│   │   ├── dashboard/  # 仪表板
│   │   ├── todos/      # 任务管理
│   │   └── reports/    # 报告页面
│   ├── store/          # Redux状态管理
│   │   ├── slices/     # Redux Toolkit切片
│   │   └── index.ts    # Store配置
│   ├── services/       # API服务
│   │   ├── api.ts      # API客户端配置
│   │   ├── auth.ts     # 认证服务
│   │   ├── todos.ts    # 任务服务
│   │   └── reports.ts  # 报告服务
│   ├── hooks/          # 自定义Hooks
│   ├── utils/          # 工具函数
│   ├── types/          # TypeScript类型定义
│   ├── i18n/           # 国际化配置
│   └── assets/         # 静态资源
├── package.json
├── tsconfig.json
├── vite.config.ts
└── Dockerfile
```

### 组件开发规范

#### 函数组件模板

```typescript
import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../store';

interface Props {
  title: string;
  onSubmit: (data: any) => void;
}

const MyComponent: React.FC<Props> = ({ title, onSubmit }) => {
  const dispatch = useDispatch();
  const { user } = useSelector((state: RootState) => state.auth);

  const handleSubmit = (data: any) => {
    onSubmit(data);
  };

  return (
    <div className="my-component">
      <h2>{title}</h2>
      {/* 组件内容 */}
    </div>
  );
};

export default MyComponent;
```

#### API 服务封装

```typescript
// services/api.ts
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 10000,
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;
```

#### 状态管理

```typescript
// store/slices/todoSlice.ts
import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { todoAPI } from '../../services/todos';

interface TodoState {
  todos: Todo[];
  loading: boolean;
  error: string | null;
}

const initialState: TodoState = {
  todos: [],
  loading: false,
  error: null,
};

export const fetchTodos = createAsyncThunk(
  'todos/fetchTodos',
  async (_, { rejectWithValue }) => {
    try {
      const response = await todoAPI.getTodos();
      return response.data;
    } catch (error) {
      return rejectWithValue(error.message);
    }
  }
);

const todoSlice = createSlice({
  name: 'todos',
  initialState,
  reducers: {
    clearError: (state) => {
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTodos.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchTodos.fulfilled, (state, action) => {
        state.loading = false;
        state.todos = action.payload;
      })
      .addCase(fetchTodos.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      });
  },
});

export const { clearError } = todoSlice.actions;
export default todoSlice.reducer;
```

## 数据库管理

### MongoDB 开发环境

```bash
# 使用 Docker 启动 MongoDB
docker run -d \
  --name todoing-mongo \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=admin123 \
  -e MONGO_INITDB_DATABASE=todoing \
  -v mongo_data:/data/db \
  mongo:7

# 连接到 MongoDB
mongosh mongodb://admin:admin123@localhost:27017/todoing
```

### 数据库初始化

```javascript
// 创建索引
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "username": 1 }, { unique: true });
db.todos.createIndex({ "user_id": 1 });
db.todos.createIndex({ "status": 1 });
db.todos.createIndex({ "created_at": -1 });

// 创建管理员用户
db.users.insertOne({
  "username": "admin",
  "email": "admin@todoing.com",
  "password": "$2a$10$...", // bcrypt hash
  "role": "admin",
  "email_verified": true,
  "created_at": new Date(),
  "updated_at": new Date()
});
```

## 测试

### 后端测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试示例

```go
// internal/service/todo_test.go
func TestTodoService_CreateTodo(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    service := NewTodoService(db)
    
    todo := &model.Todo{
        Title:       "Test Todo",
        Description: "Test Description",
        UserID:      "user123",
        Status:      "pending",
    }
    
    createdTodo, err := service.CreateTodo(todo)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, createdTodo.ID)
    assert.Equal(t, "Test Todo", createdTodo.Title)
}
```

### 前端测试

```bash
# 运行所有测试
npm run test

# 运行测试并观察变化
npm run test:watch

# 生成覆盖率报告
npm run test:coverage
```

### 测试示例

```typescript
// components/TodoItem.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Provider } from 'react-redux';
import { store } from '../store';
import TodoItem from './TodoItem';

const mockTodo = {
  id: '1',
  title: 'Test Todo',
  description: 'Test Description',
  status: 'pending',
  createdAt: new Date().toISOString(),
};

test('renders todo item correctly', () => {
  render(
    <Provider store={store}>
      <TodoItem todo={mockTodo} />
    </Provider>
  );
  
  expect(screen.getByText('Test Todo')).toBeInTheDocument();
  expect(screen.getByText('Test Description')).toBeInTheDocument();
});

test('handles status change', () => {
  const onStatusChange = jest.fn();
  
  render(
    <Provider store={store}>
      <TodoItem todo={mockTodo} onStatusChange={onStatusChange} />
    </Provider>
  );
  
  const checkbox = screen.getByRole('checkbox');
  fireEvent.click(checkbox);
  
  expect(onStatusChange).toHaveBeenCalledWith('1', 'completed');
});
```

## Docker 开发

### 开发环境 Docker Compose

```yaml
# docker/docker-compose.dev.yml
version: '3.8'

services:
  backend:
    build:
      context: ../backend-go
      dockerfile: Dockerfile.dev
    ports:
      - "5004:5004"
    volumes:
      - ../backend-go:/app
    environment:
      - MONGO_URI=mongodb://mongodb:27017/todoing
      - JWT_SECRET=dev-secret
    depends_on:
      - mongodb

  frontend:
    build:
      context: ../frontend
      dockerfile: Dockerfile.dev
    ports:
      - "5173:5173"
    volumes:
      - ../frontend:/app
      - /app/node_modules
    environment:
      - VITE_API_BASE_URL=http://localhost:5004/api

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db
    environment:
      - MONGO_INITDB_DATABASE=todoing

volumes:
  mongodb_data:
```

### 开发命令

```bash
# 启动开发环境
docker-compose -f docker/docker-compose.dev.yml up -d

# 查看日志
docker-compose -f docker/docker-compose.dev.yml logs -f

# 停止服务
docker-compose -f docker/docker-compose.dev.yml down

# 重建镜像
docker-compose -f docker/docker-compose.dev.yml build
```

## 调试

### 后端调试

```bash
# 使用 Delve 调试器
dlv debug cmd/server/main.go

# 或者在 VS Code 中配置 launch.json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Go App",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend-go/cmd/server/main.go",
      "env": {
        "MONGO_URI": "mongodb://localhost:27017/todoing",
        "JWT_SECRET": "debug-secret"
      }
    }
  ]
}
```

### 前端调试

```typescript
// 在浏览器开发者工具中调试
console.log('Debug info:', data);

// 使用 React Developer Tools
// 安装浏览器扩展：React Developer Tools

// VS Code 调试配置
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug React App",
      "type": "node",
      "request": "launch",
      "cwd": "${workspaceFolder}/frontend",
      "runtimeExecutable": "npm",
      "runtimeArgs": ["run", "dev"]
    }
  ]
}
```

## 代码规范

### Go 代码规范

1. **命名规范**
   - 包名：小写，简洁
   - 函数名：驼峰命名
   - 常量：大写下划线

2. **代码组织**
   - 每个文件单一职责
   - 公共接口在包顶部
   - 错误处理要明确

3. **注释规范**
   ```go
   // Package service provides business logic for the application.
   package service
   
   // TodoService handles todo-related operations.
   type TodoService struct {
       repo repository.TodoRepository
   }
   
   // CreateTodo creates a new todo item.
   // It returns the created todo or an error if creation fails.
   func (s *TodoService) CreateTodo(todo *model.Todo) (*model.Todo, error) {
       // Implementation
   }
   ```

### TypeScript 代码规范

1. **类型定义**
   ```typescript
   // types/todo.ts
   export interface Todo {
     id: string;
     title: string;
     description?: string;
     status: TodoStatus;
     createdAt: string;
     updatedAt: string;
   }
   
   export type TodoStatus = 'pending' | 'in_progress' | 'completed';
   ```

2. **组件规范**
   ```typescript
   // 使用 React.FC 类型
   const TodoList: React.FC<TodoListProps> = ({ todos, onTodoChange }) => {
     // 组件实现
   };
   
   // 导出默认组件
   export default TodoList;
   ```

## 常见问题

### 后端问题

**Q: MongoDB 连接失败**
```bash
# 检查 MongoDB 是否运行
docker ps | grep mongo

# 检查网络连接
telnet localhost 27017

# 查看 MongoDB 日志
docker logs todoing-mongo
```

**Q: Go 模块依赖问题**
```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 整理依赖
go mod tidy
```

### 前端问题

**Q: npm 安装失败**
```bash
# 清理 npm 缓存
npm cache clean --force

# 删除 node_modules 重新安装
rm -rf node_modules package-lock.json
npm install
```

**Q: Vite 开发服务器启动失败**
```bash
# 检查端口占用
lsof -i :5173

# 使用不同端口
npm run dev -- --port 3001
```

### Docker 问题

**Q: Docker 构建失败**
```bash
# 清理 Docker 缓存
docker system prune -a

# 查看构建日志
docker-compose build --no-cache

# 检查 Dockerfile 语法
docker build -f Dockerfile .
```

## 部署指南

### 生产环境部署

```bash
# 克隆项目到服务器
git clone git@github.com:axfinn/todoIngPlus.git
cd todoIngPlus

# 配置生产环境变量
cp docker/.env.example docker/.env.prod
vim docker/.env.prod

# 启动生产环境
docker-compose -f docker/docker-compose.golang.yml up -d

# 查看服务状态
docker-compose -f docker/docker-compose.golang.yml ps
```

### 域名和 SSL 配置

```nginx
# nginx/golang.conf
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        root /var/www/html;
        try_files $uri $uri/ /index.html;
    }
    
    location /api/ {
        proxy_pass http://backend:5004;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

*该开发指南会随着项目发展持续更新，为开发团队提供最新的开发规范和最佳实践。*
