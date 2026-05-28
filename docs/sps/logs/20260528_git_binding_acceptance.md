# 验收报告：Git 部署作者限制与配置中心 (2026-05-28)

## 1. 需求回顾
用户要求增加 Git 提交者白名单过滤功能，即：“用户可以和某些指定的git提交者绑定，可选的可限制某个用户只能提交某些用户的git 提交”。
为了保证安全性和发布的规范性，系统要求拦截非白名单人员的非法提交流水线触发。

## 2. 核心架构修改
- **数据库层 (db.go)**：为 `users` 表安全地新增了 `bound_git_authors` (TEXT) 和 `restrict_git_authors` (BOOLEAN) 字段。
- **Git 服务层 (git.go)**：引入了根据部署参数（commit/branch/tag）动态获取对应 Commit 真实提交者的解析逻辑 (`GetCommitAuthor`)。
- **业务 API 层 (api.go)**：
  - 新增 `GET /api/users/:username/git_binding`
  - 新增 `PUT /api/users/:username/git_binding`
  - 核心拦截逻辑：在 `HandleCreateTask` 中插入白名单拦截。若拦截触发，抛出 403 错误。
- **前端视图层 (Dashboard.vue)**：
  - 引入了弹窗式配置组件，并与顶栏整合。
  - 完成与 API 接口的闭环联调。

## 3. 验收结果
- [x] **前端交互测试**：弹窗状态同步正常，更新和加载均能精准反映用户配置。
- [x] **后端 API 测试**：权限组 (Admin) 的路由挂载正常，无内存泄露和并发冲突问题。
- [x] **拦截机制测试**：开启 `restrict_git_authors` 后，尝试部署未授权的作者 Commit 立即抛出 HTTP 403 被阻断，通过。
- [x] **系统稳定性**：WSL 交叉编译测试通过，SQLite 架构向下兼容通过。

## 4. 后续维护建议
- 由于目前没有完备的 RBAC 角色体系，目前的配置 API 虽然在 Admin 分组，但最好之后增加基于 `Session User == targetUser` 或真正的超管才能修改他人配置的逻辑。
- 可选地，在部署历史表中增加目标部署代码作者的持久化，以便之后查询审计日志。

## 结论
`[BUILD_SUCCESS]` 所有验收指标均已达标，修改实现完全满足用户初始需求并保持了系统零污染原则。
