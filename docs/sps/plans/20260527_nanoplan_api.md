# TASK-005: GoDeployer Web APIs 纳米计划

本计划旨在实现 GoDeployer Web 界面所需的 API 接口。包括身份验证、只读项目/环境配置查询、触发部署与回滚、流式日志查看、以及基于 Git 的多版本文件对比（Diff）。

---

## 1. 拟修改/创建文件清单

- **[NEW]** `godeployer/api.go` - API 路由注册与 HTTP 控制器实现

---

## 2. 纳米级任务细分 (代码量限制在 10-20 行/步)

### 2.1 任务 1: 用户登录与只读配置 API
*文件路径*: `godeployer/api.go`

- **[x] [Sub-Task 1.1]** 实现 `HandleLogin`，从 Body 接收用户名密码，比对数据库 bcrypt 哈希，生成 JWT 并返回。
- **[x] [Sub-Task 1.2]** 实现 `HandleGetProjects`，从全局内存缓存中获取当前的 `Config.Projects`，以 JSON 格式输出项目及其环境列表，供前端只读渲染。

### 2.2 任务 2: 任务管理与部署触发
*文件路径*: `godeployer/api.go`

- **[x] [Sub-Task 2.1]** 实现 `HandleCreateTask`，创建一个新的部署记录（状态为 `pending`），获取当前登录用户名并写入。
- **[x] [Sub-Task 2.2]** 编写异步部署 Goroutine。任务创建后，立即在新协程中执行：代码检出 -> 本地构建 -> Rsync -> 软链接切换，将实时 stdout 重定向到独立日志文件，并在成功/失败后，发布部署事件到通知总线。
- **[x] [Sub-Task 2.3]** 实现 `HandleGetTasks`，返回带有分页和过滤功能（如根据项目 ID 和环境 ID）的部署历史列表。

### 2.3 任务 3: 回滚触发与流式日志 API
*文件路径*: `godeployer/api.go`

- **[x] [Sub-Task 3.1]** 实现 `HandleTriggerRollback`，接收任务 ID，从 SQLite 中查询当时保存的配置快照，并在后台发起回滚操作。
- **[x] [Sub-Task 3.2]** 实现 `HandleGetTaskLog`，允许通过 Web 套接字（或简单的 HTTP Server-Sent Events / Chunked 传输）实时流式读取本地日志文件。

### 2.4 任务 4: 基于 Git 的多版本 Diff 对比 API
*文件路径*: `godeployer/api.go`

- **[x] [Sub-Task 4.1]** 实现 `HandleGetDiff`，接收项目 ID，从 Git 本地镜像是用 `git diff <commit_a> <commit_b>` 计算两次部署的变更细节，并以 Diff 文本结构返回，供前端做高亮对比展示。

---

## 3. 验证与单元测试规划 (物理零污染)

- **物理零污染与隔离**：
  - 测试同样使用内存数据库连接，通过注入 Gin 的 `httptest.ResponseRecorder` 模拟客户端 HTTP 请求，验证接口返回的状态码与 JSON 数据。
  - 测试的源码物理文件中字面绝不包含 `CREATE TABLE` / `DROP` / `TRUNCATE`。
- **单元测试覆盖**：
  - `TestAPI_Login`：提供正确与错误的用户名密码，断言登录接口的返回状态码与 JWT 存在性。
  - `TestAPI_GetProjects`：测试项目配置只读接口，断言返回的 JSON 与本地 yaml 配置树一致。
  - `TestAPI_CreateTaskAndGetLog`：模拟触发部署，验证部署记录正确写入 SQLite 审计表，且能通过 API 获取日志。
