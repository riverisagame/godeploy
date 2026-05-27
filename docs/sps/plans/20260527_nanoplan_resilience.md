# TASK-009: 启动状态自愈与超时防护纳米计划

## 1. 计划目标
在保证测试无损（不含 DROP/TRUNCATE/CREATE TABLE）的前提下，实现以下自愈和安全机制：
- **启动自愈**：每次主服务冷启动时，自动扫描数据库表，将卡锁在 `pending` 或 `deploying` 状态的历史任务更新为 `aborted` 并写入日志，消除僵尸部署锁。
- **超时防护**：为部署协程引入 Context 超时机制，确保本地 Hook、Git 操作以及 Rsync 同步全部在 Context 取消或超时（默认 15 分钟）时被强杀，防止僵尸进程。

---

## 2. 纳米级原子子任务拆解

### 子任务 1: `godeployer/db.go` 增加 RepairStalledTasks 自愈函数
* **文件路径**：[db.go](file:///d:/claudeprj/deploy/godeployer/db.go)
* **新增函数**：`RepairStalledTasks(db *sql.DB) error`
* **改动逻辑**：
  - 增加该全局函数。
  - 执行：`UPDATE deploy_tasks SET status = 'aborted' WHERE status IN ('pending', 'deploying')`。
* **代码规模**：约 8 行。

### 子任务 2: `godeployer/db.go` 的 InitDB 自动触发自愈
* **文件路径**：[db.go](file:///d:/claudeprj/deploy/godeployer/db.go)
* **改动位置**：`InitDB`
* **改动逻辑**：
  - 在 `InitDB` 返回 `db, nil` 前，调用 `RepairStalledTasks(db)`。
  - 返回相应的 err（若有）。
* **代码规模**：约 5 行。

### 子任务 3: `godeployer/ssh.go` 的 SSHExecutor 支持 Ctx 及 Rsync 超时
* **文件路径**：[ssh.go](file:///d:/claudeprj/deploy/godeployer/ssh.go)
* **改动位置**：`SSHExecutor` 结构体定义，`Rsync` 方法
* **改动逻辑**：
  - 在 `SSHExecutor` 结构体中新增 `Ctx context.Context` 字段。
  - 在 `Rsync` 中，将 `exec.Command("rsync", args...)` 变更为 `exec.CommandContext(s.Ctx, "rsync", args...)`（如果 `s.Ctx` 为 nil 则使用 `context.Background()`）。
* **代码规模**：约 10 行。

### 子任务 4: `godeployer/engine.go` 本地构建与 Git 操作支持 Context，并为 SSHExecutor 注入 Context
* **文件路径**：[engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
* **改动位置**：`RunDeploy`、`RunLocalBuild`
* **改动逻辑**：
  - 在 `RunLocalBuild` 中，将 `exec.Command` 改为 `exec.CommandContext(ctx, ...)`。
  - 在 `RunDeploy` 中，克隆和切换命令由 `exec.Command` 改为 `exec.CommandContext(ctx, ...)`。
  - 在 `RunDeploy` 中实例化 `SSHExecutor` 时注入 Context：`executor = &SSHExecutor{Server: srv, Ctx: ctx}`。
* **代码规模**：约 15 行。

### 子任务 5: `godeployer/api.go` 修复损坏的异步调用并传导 Context
* **文件路径**：[api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
* **改动位置**：`HandleCreateTask` 第 168-171 行
* **改动逻辑**：
  - 实例化 `DeployEngine`。
  - 获取全局超时设置（默认 15 分钟）：`ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)`。
  - 启动 goroutine 异步运行部署，并在结束时调用 `cancel()`：
    ```go
    engine := NewDeployEngine(h.db, h.executor)
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
    go func() {
        defer cancel()
        engine.RunDeploy(ctx, taskID, h.config, logFilePath)
    }()
    ```
* **代码规模**：约 10 行。

---

## 3. 测试与验证设计 (TDD)

### 3.1 RED 阶段 (编写失败测试用例)
在修改任何业务代码之前，在 `godeployer/db_test.go` 和 `godeployer/engine_test.go` 中：
1. **测试冷启动自愈**：
   - 使用内存数据库，手动插入一条 `deploying` 状态任务。
   - 调用 `RepairStalledTasks(db)`。
   - 校验原任务状态是否已变为 `aborted`。
2. **测试超时强杀**：
   - 编写 `TestEngine_DeployTimeout`。
   - 构造一个死循环的 Local Build hook（如 `sleep 5`）。
   - 将 Context 超时设置极短（如 `50 * time.Millisecond`）。
   - 触发 `RunDeploy`，验证其是否在 50ms 内超时终止。

### 3.2 物理零污染约束
- 全量测试连接 SQLite 内存库，对物理表零侵入。
- 测试源码中绝对禁止出现 `DROP`、`TRUNCATE`、`CREATE TABLE` 敏感词。

---

## 4. 运行验证命令
```powershell
go test -v ./godeployer/...
```
