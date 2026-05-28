# 用户管理可视化界面与后端API SCAN 分析

## 1. 需求背景与对齐
用户要求（使用 `/goal`）：开发完整的“用户管理”页面，实现之前所有权限相关功能的可视化管理，包含交互界面，且要求 TDD 并保证测试通过。

涉及功能点：
1. **基础用户 CRUD**：用户列表、添加用户、修改密码、删除用户。
2. **角色管理 (Role)**：分配管理员 (admin) / 观察者 (viewer) / 开发者 (developer) 角色。
3. **Git 代码提交人绑定与限制** (`bound_git_authors`, `restrict_git_authors`)。
4. **项目可见性控制** (`permitted_projects`)。

## 2. 影响面评估与自我攻击审计
### 当前实现状态
- **DB 层**：已有 `users` 表，并包含了 `username`, `password_hash`, `role`, `bound_git_authors`, `restrict_git_authors`, `permitted_projects` 字段。
- **API 层**：
  - 目前仅存在更新部分权限的特定接口（如 `HandleUpdateUserPermissions`, `HandleUpdateUserGitBinding`）。
  - **缺失**：`GET /users` (列表)、`POST /users` (创建)、`PUT /users/:username` (基础信息更新)、`DELETE /users/:username` (删除)。
- **前端层**：
  - 路由仅有 `/login` 和 `/` (Dashboard)。
  - 需要在主布局中增加一个“用户管理”导航入口（仅对 Admin 可见）。

### 影响范围
- **API `api.go`**：新增 User CRUD 路由并添加对应的 SQLite 查询。这不影响现有部署核心。
- **UI 布局 `Dashboard.vue`** 或新增统一 `Layout`：可能会修改当前的整体视图，抽出独立菜单。为了实现最小改动，建议在 `Dashboard.vue` 侧边栏菜单中添加 `/users` 路由链接，并创建 `views/UserManagement.vue`。
- **数据一致性**：删除用户是否会级联删除任务历史？当前 `deploy_tasks` 只记录 `username` 文本，无需强外键，因此对老数据无损。

### 性能对冲
- `GET /users` 可能因用户数量大而响应慢？由于内部系统用户量通常 < 100，无需复杂分页，直接全量返回或加简单分页即可。
- SQLite 并发：增加用户 CRUD 不会引发死锁，已有单连接控制，满足性能要求。

## 3. 测试对齐
1. **后端单元测试 (`api_test.go` 或新建 `user_test.go`)**：
   - Red: 创建、获取、更新、删除 API 均返回 404 或未实现错误。
   - Green: 实现数据库交互并保证测试通过。
2. **前端组件测试 (`UserManagement.spec.ts`)**：
   - Red: 列表不渲染，添加操作无触发。
   - Green: 列表正确加载数据，添加/编辑/删除事件及校验逻辑正常。

## 4. 下一步计划
与用户对齐确认后，即输出纳米级 IR 计划。
