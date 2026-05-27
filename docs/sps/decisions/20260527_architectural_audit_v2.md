# ARCH-005: 启动状态自愈与超时防护架构决策

* **日期**: 2026-05-27
* **状态**: 批准
* **发起人**: 资深架构工程师
* **受访者**: 用户

## 1. 决策背景

虽然系统已具备基本的排他锁与回滚，但在面对非正常停机（如主进程 Crash、容器驱逐、物理机断电）时，依然面临不可恢复的状态错误风险：
1. **僵尸部署锁**：正在部署（`pending`/`deploying`）的 Goroutine 被强制中止后，数据库状态残留为部署中，重启后由于互斥锁校验，新部署永久被拦截。
2. **流水线挂起**：Hook 脚本或网络 Rsync 若无超时上限，可能出现永久卡死，导致部署锁无法释放。

---

## 2. 解决方案设计

### 2.1 系统引导阶段状态自愈 (Startup Resilience)
* **实现点**: 在 Go 后端启动主流程 [main.go](file:///d:/claudeprj/deploy/main.go) 或初始化数据库 [db.go](file:///d:/claudeprj/deploy/godeployer/db.go) 之后，显式运行一次状态修剪：
  ```go
  func RepairStalledTasks(db *sql.DB) error {
      query := `UPDATE deploy_tasks SET status = 'aborted' WHERE status IN ('pending', 'deploying')`
      _, err := db.Exec(query)
      return err
  }
  ```
* **效益**: 保证每次进程冷启动后，都会自动清除上一次遭遇非正常退出的任务状态，安全释放部署排他锁。

### 2.2 部署流水线超时控制 (Deployment Timeout Context)
* **实现点**: 
  1. 在 `HandleCreateTask` 中，在发起异步 Goroutine 前创建带超时的 Context：
     `ctx, cancel := context.WithTimeout(context.Background(), 15 * time.Minute)`（最大部署时长限制为 15 分钟）。
  2. 修改 `RunDeploy` 签名以接收此 Context。
  3. 将该 Context 传导至所有的 `exec.CommandContext` 管道，如 `git clone`，`RunLocalBuild`，以及 `SSHExecutor`。
  4. 如果在 15 分钟内未完成，流水线自动强行 Kill 对应命令进程并更新状态为 `failed` (超时中断)。

---

## 3. 对现有功能影响与评估
* **无损性**: 零 DDL 操作，完全采用状态变更维护，对现有表结构和物理数据库无害。
* **开发流程**: 继续采用 TDD，编写失败用例（如在初始化前数据库有 active 任务，验证初始化后其状态是否自动变为了 aborted）。
