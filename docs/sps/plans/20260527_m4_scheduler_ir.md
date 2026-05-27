# Milestone 4: Scheduler 并发调度器 (IR Plan)

## 1. 针对“静默丢弃”的风险解答

你问到了“满载不阻塞直接静默丢弃会不会有问题”。
- **在 M3 (Notifier) 中**：事件通知（如钉钉、邮件）被丢弃是**可接受的降级策略**。当系统面临雪崩时，保主业务（部署）比发通知更重要，静默丢弃能有效防止内存爆满。
- **在 M4 (Scheduler) 中**：部署任务是核心业务，**绝对不能静默丢弃**！如果用户的部署任务被吃掉了，会导致状态永久卡在 `pending` 且不执行。
- **我们的策略 (针对 M4)**：如果调度队列满载，我们采取**同步拒绝 (Fail-Fast)**。向客户端返回 `HTTP 429 Too Many Requests`，并在数据库中将该任务直接标记为 `failed`，并附带日志“系统部署队列已满”。

## 2. 纳米级执行计划 (Option A: 内存排队 + 数据库状态辅佐)

### 阶段 1: `engine.go` 核心引擎改造
- **动作 1**：定义 `DeployJob` 结构体，包含 `TaskID`, `Config`, `LogFilePath`。
- **动作 2**：为 `DeployEngine` 增加字段：
  ```go
  queue  chan *DeployJob
  wg     sync.WaitGroup
  closed bool
  mu     sync.Mutex
  ```
- **动作 3**：在 `NewDeployEngine` 中初始化 `queue`，缓冲区设置为 `50`。
- **动作 4**：新增 `SubmitJob(job *DeployJob) error`。
  - 使用 `select { case e.queue <- job: return nil; default: return ErrQueueFull }`。
- **动作 5**：新增 `StartDispatcher(workers int)`。
  - 启动 `workers` 个常驻协程，`range e.queue`。
  - 获取到 job 后，同步调用原有的 `e.RunDeploy(...)`（无需再 `go`）。
- **动作 6**：新增 `Close(timeout)` 实现优雅停机，确保进行中的部署能走完流程或被安全中断。

### 阶段 2: `api.go` 接入调度器
- **动作**：修改 `HandleCreateTask` 和 `HandleGithubWebhook` 中的触发逻辑。
  - **旧逻辑**：`go engine.RunDeploy(...)`
  - **新逻辑**：
    ```go
    err := engine.SubmitJob(&DeployJob{...})
    if err != nil {
        engine.UpdateTaskStatus(taskID, "failed")
        // 记录日志：调度队列已满
        // API 响应 HTTP 429
    }
    ```

### 阶段 3: `main.go` 启动与优雅停机
- **动作**：
  - 在 `BootstrapApp` 后调用 `engine.StartDispatcher(3)` (默认允许同时跑 3 个并发部署)。
  - 在 `main.go` 停机钩子处追加 `engine.Close(30 * time.Second)`（部署需要较长宽限期）。

### 阶段 4: 测试覆盖 (TDD)
- **动作**：在 `engine_test.go` 和 `api_test.go` 中验证并发度限制（塞入 > 50 个任务时，触发 `ErrQueueFull` 429）。

---
**核对通过后，我们将进入 [RED 阶段]，优先编写测试。**
