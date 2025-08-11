# TodoIng 变更日志

## [0.0.2] - 2025-08-11

### 🚀 新增

- EventRepository / ReminderRepository 下沉所有 Mongo 访问与聚合
- Reminder Preview 接口：不落库计算 next_send 与描述
- Immediate Test Reminder：快速创建测试提醒 + 可选即时邮件
- Unified 聚合：基于 repository 的 upcoming / calendar 视图 + debug 输出

### ♻️ 重构

- Event / Reminder Service 去除直接 *mongo.Database 依赖，改为 interface 注入
- Advance 事件推进异步级联重算相关 active 相对提醒 next_send
- Pending / List / Search / Calendar / Advance 等复杂逻辑集中仓储层

### ✏️ 文档

- 更新 architecture.md / technical-architecture.md / backend-go README / reminder-module.md 描述新架构
- 强调 Protobuf 作为 API 单一事实来源（SSOT）
- 补充 Preview / Immediate Test / Advance 级联流程图与说明

### 🛠️ 调整

- 统一 UpdateFields 签名为 map[string]interface{}
- ReminderRepository interface 增补 Preview，解决编译缺口
- HTTP handlers 改为显式构造 repository 再注入 service

### 🧹 清理

- 移除服务层中残存的 bson / 直接集合访问
- 修复多处 Markdown 规范（标题空行 / fenced code 语言 / 表格对齐 / 去尾随空格）

### ⚠️ 未完成（Roadmap）

- UserRepository 抽象（当前 scheduler 仍直接查用户邮箱）
- Tasks / Reports 等其余集合仓储化
- Prometheus 指标 & Tracing
- 更丰富的循环 / 排除日 & 批量重算接口
- Repository 单元测试（Advance / Preview / Pending 边界）

---

## [0.0.1] - 2025-08-11

### ✨ 核心特性

- 周期事件推进：基于当前 event_date 向前滚动 +1 周期（年/月/周/日），若仍落后继续补齐到未来
- 相对提醒自动重排：事件推进后即时 recalculation（不影响 absolute times）
- 系统时间线审计：created / advanced 同步写入；空历史自动回填 created 条目
- 推进理由模态框：统一弹窗，可填写/可取消，替换旧 prompt 方式
- 多视图事件看板：卡片 / 列表 / 紧凑 / 时间线 四模式
- 评论 & 编辑窗口：用户评论 10 分钟内可编辑；系统条目不可修改
- 高亮与倒计时：全局 NowContext + 最近推进事件临时视觉高亮
- i18n 基础：中 / 英键值结构，可扩展
- Docker 一键体验：deploy.sh 脚本快速拉起 (golang 模式)
- README 精简重写并保留“请作者喝咖啡”赞助模块

### 🔄 关键流程

- 推进算法：按周期累进；one-off 推进后 inactive
- 时间线保真：创建/推进同步写入 system；首次空结果自动回填
- 提醒重排：仅相对提醒重新计算 next_send
- 评论策略：增 / 改 / 删 权限与时间窗口由服务层校验

### 🧱 技术栈基线

Frontend: React 18 · TypeScript 5 · Vite · RTK Query · i18next  
Backend: Go + Gorilla/Mux · MongoDB · Redis (提醒) · Swagger  
DevOps: Docker / Compose · Makefile · Nginx

### 📦 结构产物

- 统一事件/提醒/时间线服务逻辑与数据模型
- 初始测试用例覆盖核心 API 基础路径
- 生成 Swagger 文档入口 /swagger/

### 💖 赞助模块

保留支付宝 / 微信二维码（用于覆盖后续服务器与功能迭代成本）。

### 📝 说明

此为新仓库初始化发行版本（无历史迁移负担）。后续规划：过滤器、实时推送、统计分析、测试覆盖率提升。

---

## [2.0.0] - 2025-08-10

### � 重大重构

- **移除Node.js后端**
  - 删除所有Node.js后端相关代码和配置
  - 移除Node.js相关的文档和部署配置
  - 专注于Golang后端单一技术栈

### � Docker配置优化

- **Golang后端Docker化改进**
  - 修复Dockerfile多阶段构建配置，添加production和grpc目标
  - 统一端口配置为5004
  - 添加健康检查支持（curl）和安全配置
  
- **前端配置专业化**
  - 保留nginx-golang.conf专门用于Golang后端部署
  - 移除nginx-nodejs.conf配置
  - 修复服务间通信配置，确保正确的后端服务发现

### 🐛 环境变量与配置修复

- **邮件服务配置优化**
  - 修复EMAIL_FROM环境变量特殊字符问题
  - 完善邮件验证码功能的SMTP配置

### 🚀 部署验证

- **功能验证完成**
  - Golang版本：所有服务正常运行，API健康检查通过
  - 验证码显示问题已解决，环境变量正确生效

### 📝 文档与规范

- 完善Golang后端部署场景的Docker配置
- 优化服务间依赖关系和健康检查
- 统一容器命名规范

### 🔄 技术债务清理

- 彻底清理Node.js相关配置和文件
- 优化构建上下文和文件映射
- 统一环境变量命名和默认值

---

*本版本完成了TodoIng项目的重大重构，移除了Node.js后端，专注于Golang技术栈，简化了项目架构和维护复杂度。*
