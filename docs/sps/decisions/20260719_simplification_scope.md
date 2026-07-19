# 功能大精简范围分析 (2026-07-19)

## 核心保留 (确保上线功能可靠友好)
1. **项目配置与管理 (Projects)**：读取或管理要发布的项目配置（目标服务器、目录、构建脚本）。
2. **多节点部署与原子发布引擎 (Deployment Engine)**：拉取代码、构建、Rsync传输、SSH 执行 `ln -sfn` 软链接切换。
3. **部署记录与实时日志 (WebSocket Log)**：查看部署状态、查看实时标准输出、以及发布失败自动回滚机制。

## 提议精简剔除清单 (Technical Debt / Over-engineering)
1. **复杂的用户角色体系 (RBAC)**：删除 `Role`, `User`, `ProjectPermission`。改为单机模式（单用户或免密认证）。
2. **Webhook 自动触发部署 (Webhooks)**：移除 GitHub webhook 签名认证和自动触发功能，仅保留控制台手动一键部署，减少暴露风险。
3. **空间清理与垃圾回收 (Prune)**：旧版本历史可以保留固定个数，删除专门复杂的手动空间清理 API 逻辑。
4. **分支对比 (Branch Diff)**：删除为了实现 UI 分支对比而写的 `api_branch_diff`，属于过度设计。
5. **项目内联配置复杂功能**：精简 API 接口，只留下必要的 CRUD。

## 影响范围
- **接口层 (interfaces/api)**: 删除 `handler_auth.go`, `handler_diff.go`, `handler_prune.go`, `handler_user.go`, `handler_webhook.go`。修改 `handler_project.go` 和 `handler_deploy.go`，剥离与 User 相关的权限鉴权代码。
- **领域层 (domain)**: 大量删除 Entity 中的 User, Permission, Webhook 相关结构体，简化 `Task` 中的认证字段。
- **基础设施层 (infrastructure/db)**: 移除 Users 等表的建表与迁移逻辑，移除复杂的数据库级联查询。
- **前端页面 (web)**: 需要大面积移除登录页面、用户管理页面、分支对比页面，让主界面直接成为 Dashboard 和部署任务流。
