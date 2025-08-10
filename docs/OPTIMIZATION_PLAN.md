# TodoIng 优化计划 (Draft)

## 1. 已完成的改动快照 (Current Baseline)

| 类别 | 已完成 | 说明 / 价值 |
|------|--------|-------------|
| 聚合层 | Unified Upcoming & Calendar 支持 sources / limit 过滤 | 去除重复前端调用，服务端统一聚合逻辑，便于后续缓存与推送 |
| 日历 | Calendar 聚合包含 reminders | 减少用户切换视图；数据完整性提升 |
| 过滤 | upcoming / calendar 后端与 handler 参数解析 | 为后续 UI 添加筛选控件打基础 |
| 深链路 | ?focus= 直接定位并高亮 | 提升分享/跳转效率 |
| 严重度 | SeverityBadge + computeSeverity 统一评分 | 视觉优先级标准化，可扩展到通知/排序 |
| 前端去重 | 移除 events/reminders 独立 upcoming 逻辑 | 降低维护成本，避免状态不一致 |
| 聚合响应 | upcoming / calendar 增加 server_timestamp | 客户端可计算数据延迟、未来缓存新鲜度判断 |
| 全局反馈 | ToastProvider / GlobalErrorBoundary / SkeletonList 基础组件 | 统一错误提示/异常捕获与加载占位体验 |
| i18n | 新增 unified.severity 等键值 | 多语言基础延展，尚未统一 common.* |
| 测试基础 | 引入 Vitest + Testing Library + 初始组件 & 评分边界测试 | 建立质量基线，后续可覆盖服务与切片逻辑 |
| DevOps | docker/publish.sh 脚本 + NO_PULL 模式 | 加速本地/离线构建，统一镜像版本策略 |
| 杂项清理 | 移除 captcha & 未实现的 loadUser 残余、修正类型 | 降低噪音错误，便于继续演进 |

## 2. 当前主要痛点 (Gap Analysis)

| 维度 | 问题 | 影响 |
|------|------|------|
| UX/可用性 | 列表信息密集、缺少筛选 / 搜索 / 排序交互 | 用户感知“难用”主因之一 |
| 反馈机制 | 缺统一 Toast / Error Boundary / Skeleton | 操作结果与加载状态不直观 |
| 数据层 | 没有缓存 / SWR / RTK Query，重复请求 | 网络冗余 & 首屏延迟 |
| 实时性 | 缺 SSE/WebSocket，需手动刷新 | 协作与多端同步弱 |
| 安全/正确性 | 普通任务查询 (GetUpcoming) 代码疑似缺 userID 过滤 | 可能泄露跨用户任务 (高风险) |
| 测试覆盖 | 仅组件 + 少量函数测试，后端 0 单测 | 重构/优化风险高 |
| 性能 | 聚合每次全量 DB 访问，无短期缓存 | QPS 上升时响应时间增加 |
| 可观测性 | 缺统一结构化日志 / 指标 / Trace | 线下问题定位慢 |
| i18n 规范 | 键分布散乱，没有 lint 校验 | 多语言易不一致 |
| 可访问性 | 未评估对比度 / ARIA / 键盘导航 | 潜在用户群体验差 |
| 文档 | 架构 & 模块职责未成体系 | 新成员或二次维护成本高 |

## 3. 优化目标 (Objectives & Key Results)

| 目标 | 量化指标 (KR) |
|------|---------------|
| 首屏体验提升 | 首次聚合加载 <= 1.5s (本地缓存 & 并发) |
| 可用性 | 任务/事件检索 3 秒内完成（含筛选）；核心操作成功率 > 99% |
| 稳定性 | 覆盖率：前端关键逻辑 >= 40%，后端 Service/Handler >= 60% |
| 实时性 | 数据变更 2 秒内客户端 UI 更新 |
| 易维护性 | PR 引入的回归缺陷一周内 <= 1 个 |
| 可观测性 | 90% 请求具备 traceID & 指标上报 |

## 4. 分阶段实施路线 (Roadmap)

### Phase 0 (快速赢, 1~2 天)

- (DONE) 修复：UnifiedService 普通任务查询缺少 userID 过滤 (安全)
- (DONE) 全局 Toast / ErrorBoundary / Skeleton 基础
- (DONE) 聚合接口添加 server_timestamp 字段
- (DONE) README 增加“当前缺陷与计划”板块


### Phase 1 后端可靠性 (3~4 天)

- 为 unified_service: GetUpcoming / GetCalendar 编写单元测试 (mock Mongo layer 或引入 in-memory server)
- 增加 Mongo 索引：tasks.deadline, tasks.scheduledDate, events.eventDate, reminders.reminderAt
- 参数校验：hours / limit / sources 白名单集中封装
- 返回结构增加 version 字段 (向前兼容未来 schema)

### Phase 2 前端数据层与交互 (4~5 天)

- 引入 RTK Query：unifiedUpcomingApi / unifiedCalendarApi
- 缓存策略：staleTime 30s + onFocus refetch
- 全局 UI：ToastProvider, ErrorBoundary, SkeletonList, LoadingOverlay
- 过滤 UI：source 多选、时间窗口 hours 可调、limit 输入、severity 级别筛选
- 搜索：前端过滤 title 模糊匹配 (后端后续补 server-side)

### Phase 3 UX / 可用性增强 (5~7 天)

- Dashboard 重排：顶栏 (关键指标) + 3 列 (高严重度 / 即将到期 / Calendar 微缩)
- Inline Edit：任务/事件标题快速修改
- 快捷操作：键盘 n 新增任务；/ 聚焦搜索
- 主题/响应式：移动端折叠侧栏 + 卡片布局

### Phase 4 性能与实时 (5~7 天)

- 服务端短期缓存：Redis key: unified:upcoming:{user}:{hash(params)} TTL 15s
- SSE / WebSocket：发布任务/事件/提醒变更；客户端合并更新 state
- 批处理：后台定时预计算高频用户 next 15 分钟窗口
- 前端乐观更新：创建/编辑后本地立即插入并标记 pending

### Phase 5 可观测性 & 运维 (4~5 天)

- 结构化日志 (zap 或 logrus) + requestID Middleware
- Prometheus 指标：http_request_duration_seconds, upcoming_cache_hit_total
- 简易 tracing (header 透传 X-Request-ID)；后期可接入 OpenTelemetry
- 前端埋点：页面切换、聚合请求耗时、错误计数 (console + window.onerror)

### Phase 6 国际化 & 可访问性 (2~3 天)

- i18n key 整理：common.*, forms.*, severity.*
- 脚本校验未翻译键 & 未使用键
- A11y：语义化标签、跳转链接、颜色对比校验、Tab 顺序

### Phase 7 文档与移交流程 (2 天)

- docs/ARCHITECTURE.md：模块边界、数据流、聚合时序
- docs/SEVERITY.md：评分模型 & 可调参数
- CONTRIBUTING.md：分支、测试、提交规范、发布流程
- API 文档补齐 /api/unified/* (Swagger 注释)

## 5. 任务拆分清单 (Backlog Board)

| ID | 描述 | Phase | Priority | 预估 | 状态 |
|----|------|-------|----------|------|------|
| B01 | 普通任务查询添加 userID 过滤 | 0 | P0 | 0.5d | DONE |
| F01 | 全局 Toast / ErrorBoundary | 0 | P0 | 0.5d | DONE |
| F02 | Skeleton 组件 & 首屏占位 | 0 | P1 | 0.5d | DONE |
| F03 | README 缺陷与计划板块 | 0 | P1 | 0.5d | DONE |
| T01 | GetUpcoming 单元测试 (sources/limit/排序) (已覆盖纯逻辑, 待补 Mongo/mock) | 1 | P0 | 1d | IN_PROGRESS |
| T02 | GetCalendar 单元测试 (过滤/limit) | 1 | P1 | 0.5d | TODO |
| DB01 | 添加索引 (deadline 等) | 1 | P0 | 0.5d | TODO |
| API01 | 参数校验统一封装 | 1 | P1 | 0.5d | TODO |
| FE01 | RTK Query 集成 upcoming (api+store+Board+Events/Reminders 预热) | 2 | P0 | 1d | IN_PROGRESS |
| FE02 | 过滤控件 (sources/hours/limit/severity) (Board 初版完成) | 2 | P0 | 1d | IN_PROGRESS |
| FE03 | 搜索框 & 快捷键 / | 3 | P1 | 0.5d | TODO |
| UX01 | Dashboard 重构 | 3 | P0 | 2d | TODO |
| RT01 | Redis 缓存 unified | 4 | P0 | 1d | TODO |
| RT02 | SSE 推送实现 | 4 | P0 | 2d | TODO |
| OBS01 | Prometheus 指标 | 5 | P0 | 1d | TODO |
| OBS02 | 结构化日志 + requestID | 5 | P0 | 1d | TODO |
| I18N1 | i18n key 重构 | 6 | P1 | 1d | TODO |
| DOC1 | Architecture 文档 | 7 | P0 | 0.5d | TODO |
| DOC2 | Severity 文档 | 7 | P1 | 0.5d | TODO |

## 6. 技术决策摘要

| 主题 | 决策 | 依据 |
|------|------|------|
| 数据获取 | RTK Query 取代手写 thunk | 缓存 + 请求去重 + 自动重试 |
| 实时 | SSE 起步，后续可升 WebSocket | 实现简单，后向兼容 |
| 缓存策略 | Redis 短 TTL + ETag | 热数据重复查询成本高 |
| 测试策略 | Service 层与 Handler 分离 mock | 控制依赖，快速反馈 |
| 指标 | Prometheus + 自定义中间件 | 容器化兼容性好 |
| 严重度模型 | 外置配置 (JSON/ENV 可调权重) | 支持后期动态调优 |

## 7. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 聚合缓存与实时推送一致性 | 用户看到过期数据 | TTL 短 + 推送后主动失效缓存 |
| 任务查询安全遗漏 | 数据泄露 | 立即修复 + 增加单测覆盖 |
| SSE 断线 | 数据不同步 | 心跳 + 重连回退全量拉取 |
| 复杂度上升 | 维护成本 | 分阶段提交 + 文档先行 |

## 8. 验收标准 (Definition of Done)

- 每阶段结束：
  1. CI 通过 (lint + test + build)
  2. README / docs 更新
  3. 关键指标或日志验证点具备
  4. 回归测试 (手动脚本或 checklist)

## 9. 时间轴 (粗略)

| 周 | 主要里程碑 |
|----|------------|
| 第 1 周 | Phase 0 + Phase 1 完成 |
| 第 2 周 | Phase 2 大部分, 开始 Phase 3 |
| 第 3 周 | 完成 Phase 3 & 4 |
| 第 4 周 | Phase 5~7 收尾 + 调优 |

## 10. 下一步 (Immediate Next Steps)

1. (FE01) 将 UnifiedBoardPage 改为使用 useGetUpcomingQuery 并验证缓存 / 重复请求去重
2. (FE02) 抽象过滤栏组件（复用到 Events / Reminders / Dashboard），接入真实 severity 计算而非 roughScore
3. (T01) 扩展 GetUpcoming 测试：使用 in-memory Mongo / mock collection 验证 sources/limit/hours 边界 & 去重逻辑
4. (API01) 抽取 hours/limit/sources 参数校验辅助函数至独立模块并在 handler 中引用
5. (Docs) 更新 README：标记 RTK Query 引入（试点），新增一小节解释 buildUpcomingItems 抽象与测试覆盖范围
6. (FE01) 增加 RTK Query 配置：refetchOnFocus、keepUnusedDataFor=30，提供 stale 缓存策略说明
7. (FE02) Severity 过滤：后端返回或前端统一 computeSeverity 值，替换临时 roughScore，支持区间筛选

> 注：若 FE01 页面迁移确认稳定，可逐步淘汰 thunk fetchUnifiedUpcoming，减少重复数据源。

---
(本文件为草案，可随着执行进度更新。)
