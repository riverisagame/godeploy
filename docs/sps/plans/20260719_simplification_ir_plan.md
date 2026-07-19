# 纳米级精简执行计划 (2026-07-19)

**目标**：精简繁杂的多用户/权限、Webhook体系；坚决保留核心四要素：**上线、回滚、日志、对比(Diff)**。确保系统处于极简且高可靠状态。

## Step 1: 领域层与底层数据库 (Data Model Pruning)
- **目标文件**: `deploy/godeployer/domain/entity.go`
  - **动作**: 删除 `User`、`ProjectPermission` 结构体。
  - **动作**: 修改 `Task` 结构体，移除或注释掉关联的 `UserID` (改为单纯记录执行者的字符串如 "admin" 即可)。
- **目标文件**: `deploy/godeployer/domain/repository.go`
  - **动作**: 删除 `User` 相关的仓储接口方法，例如 `FindUser`, `CreateUser` 等。
- **目标文件**: `deploy/godeployer/infrastructure/db/db.go`
  - **动作**: 注释或删除 `AutoMigrate` 中的 `User` 和 `ProjectPermission`。

## Step 2: 业务逻辑层权限剥离 (Service Decoupling)
- **目标文件**: `deploy/godeployer/application/deployment_service.go`
  - **动作**: 移除所有与用户强关联的检验逻辑（如检验某用户是否有某项目权限），所有部署动作直接放行或依赖全局级认证。

## Step 3: API 层清洗与接口删减 (Endpoint Purge)
- **目标文件**: (直接执行物理删除)
  - `[DELETE]` `deploy/godeployer/interfaces/api/handler_auth.go`
  - `[DELETE]` `deploy/godeployer/interfaces/api/handler_user.go`
  - `[DELETE]` `deploy/godeployer/interfaces/api/handler_webhook.go`
- **目标文件**: `deploy/godeployer/interfaces/api/routes.go`
  - **动作**: 移除对应的用户/鉴权/Webhook 路由注册。将原来的 JWT 多用户中间件简化为“全局白名单”或“静态统一 Token”中间件。
- **目标文件**: `deploy/godeployer/interfaces/api/handler_deploy.go` 及 `handler_project.go`
  - **动作**: 取消从 Context 获取 userID，替换为常量（如 "system"）以保证日志表不断裂。

## Step 4: 绿灯验证 (TDD & Validation)
- 运行针对保留四大功能的测试命令（确保物理静默与对现有数据的零侵入）：
  - `go test -v ./godeployer/interfaces/api -run TestDeploy` (上线)
  - `go test -v ./godeployer/interfaces/api -run TestRollback` (回滚)
  - `go test -v ./godeployer/interfaces/api -run TestDiff` (对比)
  - `go test -v ./godeployer/interfaces/api -run TestWS` (日志)
