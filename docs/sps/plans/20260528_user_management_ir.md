# 用户管理可视化界面纳米级执行计划 (IR)

## 背景
需要为 admin 用户提供完整的可视化“用户管理”页面。这涉及扩充现有 User CRUD API，以及在前端实现对应页面并注册路由。我们将按照 TDD 流程进行。

## 第一阶段：API 测试与接口定义 (Red Phase)
**目标**：编写失败的单元测试，定义用户管理的 CRUD 接口。
- **文件**：`godeployer/api_user_test.go` (新建)
- **动作**：
  1. 编写 `TestAPI_GetUsers`：测试 `GET /api/admin/users`。
  2. 编写 `TestAPI_CreateUser`：测试 `POST /api/admin/users`。
  3. 编写 `TestAPI_UpdateUser`：测试 `PUT /api/admin/users/:username`。
  4. 编写 `TestAPI_DeleteUser`：测试 `DELETE /api/admin/users/:username`。
- **预期**：所有测试运行失败（404 Not Found 或未通过）。

## 第二阶段：实现后端 API (Green Phase)
**目标**：在 `api.go` 实现路由与处理函数，使测试通过。
- **文件**：`godeployer/api.go`
- **动作**：
  1. 在 `SetupRoutes` 的 `adminGrp` 组中注册四个路由。
  2. 实现 `HandleGetUsers`：`SELECT id, username, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects FROM users`。
  3. 实现 `HandleCreateUser`：解析请求、bcrypt 密码、INSERT 入库。
  4. 实现 `HandleUpdateUser`：解析请求、更新 role, permitted_projects 等。若有 password 则同时更新 hash。
  5. 实现 `HandleDeleteUser`：`DELETE FROM users WHERE username = ?`（限制不能删除自身或唯一admin）。

## 第三阶段：前端测试先行 (Red Phase)
**目标**：定义 `UserManagement.vue` 的单元测试。
- **文件**：`web/src/__tests__/UserManagement.spec.ts` (新建)
- **动作**：
  1. 挂载 `UserManagement.vue`（此时文件不存在）。
  2. 测试其能够调用 `GET /api/admin/users` 并在 `el-table` 渲染用户列表。
  3. 测试点击“新增用户”按钮能打开弹窗。
- **预期**：测试运行失败。

## 第四阶段：实现前端页面 (Green Phase)
**目标**：实现页面代码和路由并使前端测试通过。
- **文件**：
  - `web/src/views/UserManagement.vue` (新建)
  - `web/src/router.ts`
  - `web/src/views/Dashboard.vue`
- **动作**：
  1. 编写 `UserManagement.vue`：使用 Element Plus 实现表格（用户名、角色、允许的项目等）与“添加/编辑”对话框（支持密码输入与权限选择）。
  2. `router.ts`：增加 `/users` 路由映射。
  3. `Dashboard.vue`：在 el-menu 侧边栏增加 `<el-menu-item index="/users">用户管理</el-menu-item>`（可设为仅限 `role === 'admin'` 时可见）。

## 第五阶段：全链路验收
- **目标**：合并、构建并手动跑一遍页面。
- **动作**：通过 `npm run test` 和 `go test` 验证。
