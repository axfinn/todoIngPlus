# 🐳 TodoIng Docker 部署

当前版本仅包含 Golang 单体后端与前端 (Vite + Nginx)。所有旧的 Node.js / 微服务草案已移除，文档与脚本已同步精简。

## ⚡ 快速开始

```bash
# 启动 (后端 + 前端 + Mongo + Redis + 可选监控)
./deploy.sh golang up --build

# 停止
./deploy.sh golang down
```

访问:

- 前端: <http://localhost> (或 <http://localhost:3000> 开发热更场景)
- API: <http://localhost:5004/api>
- Swagger: <http://localhost:5004/swagger/>

## 📁 主要 Compose 文件

| 文件 | 场景 | 说明 |
|------|------|------|
| docker-compose.golang.yml | 标准运行 | 后端 + 前端 + (可选) gRPC / 监控 |
| docker-compose.dev-full.yml | 开发 | 热重载 / 调试工具 / Mongo Express / Redis Commander / MailHog / docs |
| docker-compose.prod.yml | 生产 | 单实例后端 + 前端 + (可选) 副本 / 监控 |
| docker-compose.local.yml | 本地精简 | 仅快速本地运行（Mongo + 后端 + 前端） |

## 🧩 Profiles (可选)

- grpc: 启动独立 gRPC 目标镜像（当前与主二进制功能一致）
- monitoring: Prometheus + Grafana
- replica (prod): 启用 Mongo/Redis 副本相关服务（当前默认关闭）
- testing (dev-full): 提供独立测试 Mongo 实例

示例:

```bash
# 启动含监控
./deploy.sh golang up --profile monitoring
# 开发模式仅启 Go 后端与前端
./deploy.sh dev up --profile golang
```

## 🔧 环境变量

复制 `docker/.env.example` 为 `.env`。关键变量:

- MONGO_ROOT_PASSWORD / JWT_SECRET / DEFAULT_PASSWORD: 生产务必修改
- ENABLE_CAPTCHA / ENABLE_EMAIL_VERIFICATION / DISABLE_REGISTRATION: 功能开关
- EMAIL_*: 邮件发送 (未配置则相关邮件功能不可用)
- GRAFANA_PASSWORD: 监控面板密码 (monitoring profile)

## 🗃️ 数据持久化

默认使用命名卷：

- Mongo: mongodb_data_*
- Redis: redis_data_*
- Prometheus / Grafana: `prometheus_data_*` / `grafana_data_*`

日志映射至 `docker/logs/*` (golang compose)。

## 🛠️ 常用命令

```bash
# 查看状态
./deploy.sh golang ps
# 追踪日志
./deploy.sh golang logs backend-golang --follow
# 重建并启动
./deploy.sh golang up --build -d
# 开发热重载环境 (Go + 前端)
./deploy.sh dev up --profile golang
```

## 🛡️ 生产建议

- 使用固定版本标签，不用 latest
- 设定强随机 JWT_SECRET (>=32 chars)
- 按需去除对外暴露的 Mongo/Redis 端口 (删除 ports 映射仅内部网络访问)
- 配置 CORS_ORIGINS 为实际域名
- 通过反向代理 (如云 WAF / LB) 提供 TLS
- 定期备份数据库卷 (mongodump)

## 🧪 健康检查

后端: `GET /health` -> 200 ok

可在编排系统 (Kubernetes 等) 中分别配置 `/health` (liveness) 与未来可扩展的 `/ready` (readiness)。

## 🐳 发布镜像

使用 `docker/publish.sh`：

```bash
# 构建本机架构
./docker/publish.sh build
# 多架构并推送
PLATFORMS=linux/amd64,linux/arm64 ./docker/publish.sh release
# 自定义版本
VERSION=1.2.3 ./docker/publish.sh release
```

支持前端构建参数:

```bash
BUILD_ARGS_FRONTEND="VITE_API_BASE_URL=/api VITE_APP_VERSION=1.2.3" ./docker/publish.sh build
```

## ♻️ 清理

```bash
docker system prune -f       # 清理悬挂层(谨慎)
docker volume rm <name>      # 删除不再需要的卷
```

## 🔄 升级注意

- 修改环境变量后需要: `./deploy.sh golang up --build` (或重启后端容器)
- 更新前端仅需重建前端镜像 (publish.sh 或 compose build frontend)

## ✅ 当前不再包含

- 旧 Node.js 后端
- 微服务草案 (Consul / Kong / RabbitMQ 等)
- 多余第三方云存储占位变量

如需引入微服务或扩展基础设施，请在独立分支重新设计并添加对应 `services/*` 代码与配置，再更新本文档。

