# TASK-008: 精准回滚、Git Diff、并发部署锁及日志防护纳米计划

## 1. 计划目标
在不污染物理数据库（使用 SQLite 内存库）、不使用 DDL 敏感词（DROP/TRUNCATE/CREATE TABLE）的前提下，实现以下增强：
- **精准回滚**：回滚直接切换到指定任务的 Release 目录。
- **Git Diff**：实现 `/api/tasks/:id/diff` 接口，自动对比当前任务与其前一个成功部署任务的 Commit 差异。
- **并发互斥部署锁**：同项目同环境在同一时刻只允许一个 `pending`/`deploying` 任务。
- **日志截断保护**：当读取部署日志文件大小超过 1MB 时只返回尾部 1MB 内容。
- **Rsync 错误信息捕获**：修正 nil stderr，完整收集传输中的报错。

---

## 2. 纳米级原子子任务拆解

### 子任务 1: `godeployer/ssh.go` 中的 Rsync 报错捕获优化
* **文件路径**：[ssh.go](file:///d:/claudeprj/deploy/godeployer/ssh.go)
* **函数**：`Rsync`
* **变量/改动逻辑**：
  - 将 `var stderr io.Writer` 变更为 `var stderr bytes.Buffer`
  - 将 `cmd.Stderr = &stderr` 绑定到该 buffer
  - 发生错误时，如果 `stderr.Len() > 0`，返回 `fmt.Errorf("rsync command failed: %s: %w", stderr.String(), err)`，否则返回原 error。
* **改动规模**：约 10 行。

### 子任务 2: `godeployer/engine.go` 中的回滚重构为精准切换
* **文件路径**：[engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
* **函数**：`RunRollback` 废弃；新增 `RunRollbackToTask(targetTaskID int64, server ServerConfig) error`
* **改动逻辑**：
  - 传入要回滚到的特定 `targetTaskID`。
  - SQL 查询：`SELECT project_id, env_id, release_name FROM deploy_tasks WHERE id = ? AND status = 'success'` 校验并提取 `releaseName`。
  - 调用 `SwitchSymlink(server, releaseName)` 进行软链接切换。
  - 可选：在数据库里将该任务或回滚审计状态进行更新。
* **改动规模**：约 20 行。

### 子任务 3: `godeployer/api.go` 引入项目环境互斥锁
* **文件路径**：[api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
* **函数**：`HandleCreateTask`
* **改动逻辑**：
  - 在执行 `INSERT INTO deploy_tasks` 之前，先执行查询：
    `SELECT COUNT(*) FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND status IN ('pending', 'deploying')`
  - 如果 count > 0，则直接调用 `c.JSON(http.StatusConflict, gin.H{"error": "another deployment is already in progress for this project and environment"})` 并 `return`。
* **改动规模**：约 12 行。

### 子任务 4: `godeployer/api.go` 日志读取 1MB 截断防爆
* **文件路径**：[api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
* **函数**：`HandleGetTaskLog`
* **改动逻辑**：
  - 使用 `os.Open(logFilePath)` 打开文件，并获取文件状态 `Stat()`。
  - 若文件大小 `size <= 1 * 1024 * 1024`（1MB），直接用 `os.ReadFile` 读取。
  - 若文件大小 `size > 1 * 1024 * 1024`，使用 `Seek(size - 1*1024*1024, io.SeekStart)` 定位，然后读取最后的 1MB 字节内容，附加前置提示 `[Log truncated, showing last 1MB]...` 返回给前端。
* **改动规模**：约 18 行。

### 子任务 5: `godeployer/api.go` 回滚路由对接精准回滚
* **文件路径**：[api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
* **函数**：`HandleTriggerRollback`
* **改动逻辑**：
  - 从路由参数获取回滚目标任务 ID：`c.Param("id")`
  - 在 `HandleTriggerRollback` 中，查询该任务的 `project_id` 和 `env_id`。
  - 获取服务器配置。
  - 循环服务器调用 `engine.RunRollbackToTask(targetTaskID, srv)` 进行回滚。
* **改动规模**：约 15 行。

### 子任务 6: `godeployer/api.go` 实现 Git Diff 对比路由
* **文件路径**：[api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
* **路由**：`GET /api/tasks/:id/diff`
* **函数**：`HandleGetTaskDiff`
* **改动逻辑**：
  - 获取 `:id`，查出当前任务的 `project_id`, `commit_id`。
  - SQL 查找在当前任务 ID 之前同一项目环境下最近一次状态为 `success` 的任务：
    `SELECT commit_id FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND id < ? AND status = 'success' ORDER BY id DESC LIMIT 1`
  - 若无前置成功记录，直接返回 `{"diff": "首次部署，无对比基准"}`。
  - 若有，在本地对应工作区目录（`WorkspacePath/project_id/release_name`）执行命令：
    `git diff <prev_commit> <curr_commit>`
  - 将生成的文本输出返回给前端。
* **改动规模**：约 30 行（可细分为查询和执行两步）。

---

## 3. 测试与验证设计 (TDD)

### 3.1 RED 阶段 (编写失败测试用例)
在编写逻辑代码之前，我们必须编写测试来覆盖这些新场景：
1. **测试互斥部署锁**：在 `api_test.go` 中，向 DB 插入一条状态为 `deploying` 的任务，然后发起 API `POST /api/tasks` 部署该项目环境，期望返回 `409 Conflict`。
2. **测试日志截断**：写一个 1.5MB 的临时日志文件，调用 `GET /api/tasks/:id/log`，验证返回结果的前缀包含截断声明，且总大小不超过 1.1MB。
3. **测试精准回滚**：往 DB 中写入三个任务（Task 1成功，Task 2成功，Task 3成功）。触发回滚到 Task 1，验证 SwitchSymlink 收到的 release_name 是 Task 1 的 release_name。
4. **测试 Git Diff 接口**：模拟前置成功 Commit 与当前 Commit，调用 `GET /api/tasks/:id/diff` 验证其输出。

### 3.2 物理零污染约束
- 全量测试中均使用 SQLite 内存模式，不与任何磁盘上的 `.db` 文件接触。
- 测试源码中坚决不出现 `DROP`、`TRUNCATE`、`CREATE TABLE` 语句。

---

## 4. 运行验证命令
单元测试运行：
```powershell
go test -v ./godeployer/...
```
编译 Linux 二进制命令：
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o godeployer_linux main.go
```
