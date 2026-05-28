# 用户项目可见性与权限限制

根据需求，用户需要有仓库权限的限制，只能看到并部署自己有权限的项目（仓库）。

## Proposed Changes

### 1. 数据库修改 (db.go)
- 在 `InitDB` 中为 `users` 表增加新字段 `permitted_projects TEXT DEFAULT '*'`。
- `*` 表示拥有所有项目权限（默认管理员），逗号分隔的字符串（如 `proj1,proj2`）表示具体拥有的项目 ID 列表。

### 2. API 接口更新 (api.go)
- 修改 `HandleGetProjects` 接口：
  - 从 `gin.Context` 获取当前 `username`。
  - 查询数据库获取该用户的 `permitted_projects`。
  - 遍历 `h.config.Projects`，如果权限为 `*` 则全部返回；否则只过滤出 `permitted_projects` 包含的项目列表返回。
- 部署相关的接口（如 `HandleCreateTask`）同样需要加入拦截：
  - 如果用户没有该 `ProjectID` 的权限，则返回 `403 Forbidden`，防止 API 绕过。
- 新增/修改用户权限配置接口：
  - 新增 `GET /api/users` 获取所有用户列表（供管理面板使用）。
  - 修改或复用 `PUT /api/users/:username/git_binding`（可重命名为权限 API），支持传入 `permitted_projects` 字段进行修改。

### 3. 前端界面优化 (Dashboard.vue)
- 由于之前新增了**账号配置**弹窗，目前仅能修改自己的配置。
- **改动设计**：
  - 考虑到只有管理员（admin）能修改权限，我们在弹窗内增加一个 **项目可见性 (Permitted Projects)** 的多选框或文本框。
  - 为了能够配置 **其他用户**，我们将“账号配置”升级为“**用户管理与权限配置**”，在弹窗顶部增加一个选择器，拉取系统内所有用户并切换配置对象。
  - 普通用户登录时隐藏该入口或只读。

## Open Questions

> [!WARNING]
> **是否需要一个完整的“用户管理”页面？**
> 目前系统只有固定的 demo 用户（admin，deployer，viewer）。我计划在前端弹窗里加一个“选择用户”的下拉框来完成对各个角色的权限配置。这是否符合您的期望？还是您希望我在左侧栏单独增加一个「用户管理」菜单栏？

> [!IMPORTANT]
> **默认权限策略**
> 新字段增加后，现存用户的默认权限我会设置为 `*`（即能看到所有项目）以免打断现有流程，管理员可按需将 `*` 更改为具体的项目 ID。是否可以？
