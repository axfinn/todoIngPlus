# TodoIng 变更日志

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
