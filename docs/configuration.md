# 配置管理

系统通过环境变量进行配置。以下列出当前代码实际使用或支持的变量，移除未实现/不再使用的旧项。

## 后端配置

### 基础服务

| 变量名 | 描述 | 默认值 | 示例 |
|--------|------|--------|------|
| `PORT` | HTTP 服务端口 | `5004` | `5004` |
| `MONGO_URI` | MongoDB 连接字符串 | (必填) | `mongodb://localhost:27017/todoing` |
| `JWT_SECRET` | JWT 签发密钥 | (必填) | `change_me_long_secret` |
| `GRPC_PORT` | （可选）gRPC 端口（目前默认未启用主流程） | `9000` / 未使用 | `9000` |
| `INSTANCE_ID` | 实例标识（用于在响应头 `X-Instance` 中回显） | 空 | `todoing-api-1` |
| `DEBUG` | 是否输出调试日志（影响内部 LogDebug） | `false` | `true` |

说明：代码中使用的是 `MONGO_URI`（不是 `MONGODB_URI`）。文档旧版本出现的 `MONGODB_URI` 已移除。

### 功能 / 安全控制

| 变量名 | 描述 | 默认值 | 示例 |
|--------|------|--------|------|
| `DISABLE_REGISTRATION` | 禁用注册（true 则仅允许已有账号登录） | `false` | `true` |
| `ENABLE_CAPTCHA` | 启用图片验证码登录流程 | `false` | `true` |
| `ENABLE_EMAIL_VERIFICATION` | 启用邮箱验证码（注册/登录） | `false` | `true` |

### 邮件发送配置

用于两类功能：1) 邮箱验证码（Send）；2) 提醒 / 通知邮件（SendGeneric）。

系统支持定义一组专用的提醒邮箱（`REMINDER_EMAIL_*`）。如果该组中核心字段缺失（host/user/pass），将自动回退使用通用 `EMAIL_*` 配置。

| 变量名 | 描述 | 默认值 | 示例 |
|--------|------|--------|------|
| `EMAIL_HOST` | 通用 SMTP 主机 | (必填*) | `smtp.gmail.com` |
| `EMAIL_PORT` | 通用 SMTP 端口 | `587` | `587` |
| `EMAIL_USER` | 通用 SMTP 用户名 | (必填*) | `noreply@example.com` |
| `EMAIL_PASS` | 通用 SMTP 密码 / 应用专用密码 | (必填*) | `app_password` |
| `EMAIL_FROM` | 通用发件人（未设则用 USER） | 空 | `TodoIng <noreply@example.com>` |
| `REMINDER_EMAIL_HOST` | 提醒专用 SMTP 主机 | 回退到 EMAIL_HOST | `smtp.qq.com` |
| `REMINDER_EMAIL_PORT` | 提醒专用端口 | `587` | `465` |
| `REMINDER_EMAIL_USER` | 提醒专用用户名 | 回退到 EMAIL_USER | `reminder@example.com` |
| `REMINDER_EMAIL_PASS` | 提醒专用密码 | 回退到 EMAIL_PASS | `xxx` |
| `REMINDER_EMAIL_FROM` | 提醒专用发件人 | 回退到 REMINDER_EMAIL_USER 或 EMAIL_FROM | `TodoIng Reminder <reminder@example.com>` |

(* 邮箱验证码或提醒需要发送邮件时，至少 EMAIL_HOST / EMAIL_USER / EMAIL_PASS 需要配置；端口缺省默认为 587。代码未使用 `EMAIL_SECURE`，因此删除该项。)

### 默认初始用户

首次启动如果数据库中尚无用户，会读取以下变量创建一个初始账户。

| 变量名 | 描述 | 默认值 | 示例 |
|--------|------|--------|------|
| `DEFAULT_USERNAME` | 初始用户名 | `admin` | `admin` |
| `DEFAULT_PASSWORD` | 初始密码 | `admin123` | `S3cureP@ss` |
| `DEFAULT_EMAIL` | 初始邮箱 | `admin@example.com` | `admin@example.com` |

(旧文档中的 `DEFAULT_ADMIN_EMAIL / PASSWORD / NAME` 已废弃，名称以代码实际变量为准。)

### 其他 / 运行时

暂无 Redis / 消息队列配置（旧文档提及内容已移除）。性能指标、追踪等后续加入时再补充。

## 前端配置

前端使用 Vite，环境变量需以 `VITE_` 前缀。仅保留当前功能实际使用的变量。

| 变量名 | 描述 | 默认值 | 示例 |
|--------|------|--------|------|
| `VITE_API_BASE_URL` | 后端 API 基础路径 | `/api` | `http://localhost:5004/api` |
| `VITE_DISABLE_REGISTRATION` | 是否禁用注册 UI | `false` | `true` |
| `VITE_ENABLE_CAPTCHA` | 启用验证码 UI | `false` | `true` |
| `VITE_ENABLE_EMAIL_VERIFICATION` | 启用邮箱验证码 UI | `false` | `true` |

## 配置示例

### 后端 .env 示例

```env
# 基础
PORT=5004
MONGO_URI=mongodb://localhost:27017/todoing
JWT_SECRET=change_me_long_secret
INSTANCE_ID=todoing-api-1
DEBUG=false

# 功能控制
DISABLE_REGISTRATION=false
ENABLE_CAPTCHA=true
ENABLE_EMAIL_VERIFICATION=true

# 邮件（通用）
EMAIL_HOST=smtp.gmail.com
EMAIL_PORT=587
EMAIL_USER=noreply@example.com
EMAIL_PASS=app_password
EMAIL_FROM=TodoIng <noreply@example.com>

# （可选）提醒专用邮箱；如不完整则自动回退到上面通用配置
REMINDER_EMAIL_HOST=smtp.qq.com
REMINDER_EMAIL_PORT=587
REMINDER_EMAIL_USER=reminder@example.com
REMINDER_EMAIL_PASS=reminder_password
REMINDER_EMAIL_FROM=TodoIng Reminder <reminder@example.com>

# 默认初始用户
DEFAULT_USERNAME=admin
DEFAULT_PASSWORD=admin123
DEFAULT_EMAIL=admin@example.com
```

### 前端 .env 示例

```env
VITE_API_BASE_URL=http://localhost:5004/api
VITE_DISABLE_REGISTRATION=false
VITE_ENABLE_CAPTCHA=true
VITE_ENABLE_EMAIL_VERIFICATION=true
```

## 关键说明

### 注册开关

`DISABLE_REGISTRATION=true` 时，注册接口返回 403；前端若同时设置 `VITE_DISABLE_REGISTRATION=true` 会隐藏注册入口。

### 图片验证码

启用后（前后端都为 true）登录流程需先获取并输入验证码。

### 邮箱验证码

启用后支持：

1. 注册校验邮箱验证码。
2. 无密码邮箱验证码登录（切换登录方式后获取验证码）。

需要确保邮件配置完整。若配置了 `REMINDER_EMAIL_*` 且完整，将用它发送提醒类邮件；否则继续使用通用 `EMAIL_*`。

### 调试日志

`DEBUG=true` 时 `internal/observability/logger.go` 中的调试日志会输出，有助于排查复杂查询/聚合行为。

### 实例标识

设置 `INSTANCE_ID` 后，认证中间件会在响应头写入 `X-Instance`，方便多副本部署调试流量分布。

---

以上配置列表与当前代码匹配，如有新增变量请同步更新本文件。