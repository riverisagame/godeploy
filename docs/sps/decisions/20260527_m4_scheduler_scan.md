# Milestone 4: Scheduler (调度器) 架构分析与对抗性拷问 (SCAN Phase)

## 1. 需求分析与现状剖析

在完成 M3 高并发通知重构后，系统在“部署后事件广播”方面已具备高并发抗压能力。然而，针对**“部署任务本身”**的调度，目前的架构仍然存在单点和失控的隐患。

**现状分析 (engine.go & api.go)**：
- API 接收到部署请求后，直接通过 `go engine.RunDeploy(ctx, taskID, ...)` 触发执行。
- `RunDeploy` 内部会调用外部 PHP Deployer 进程。如果同时涌入 50 个部署请求（不论是人工触发还是 Webhook 洪峰触发），后台将会瞬间拉起 50 个 PHP 进程，直接挤爆 CPU 与内存。
- 缺少任务排队、最大并发限制（Concurrency Limit）、延时执行（Delayed/Cron Schedule）的支持。

## 2. 深度对抗性拷问 (Adversarial Critique)

在你（用户）简单回复“继续”并要求实施 M4 调度时，我作为主控编译器必须提出以下硬核质疑：

1. **调度的真实意图是什么？**
   - **选项 A (防洪排队 Worker Pool)**：你仅仅是希望避免 API 并发触发导致宿主机 OOM，所以需要一个针对部署任务的“内存级排队+限定并发度（例如同一时间最多只允许跑 3 个部署任务）”的机制？
   - **选项 B (Cron 定时任务)**：你需要像 Jenkins 那样支持 `0 0 * * *` 定时自动拉取代码并部署的定时引擎？
   - **选项 C (持久化队列池)**：系统重启时，尚未执行完毕的任务不能丢失，必须通过 SQLite 持久化队列表（Pending 状态），重启后继续按顺序调度？

2. **性能与副作用对冲分析**：
   - 如果仅仅是内存 Worker Pool（选项 A），实现简单，无须 DML，但应用重启会丢失队列中的 Pending 任务。
   - 如果引入 SQLite 轮询（选项 C），则必须引入轮询机制（或 SELECT FOR UPDATE 悲观锁机制），由于 SQLite 处理高并发锁能力有限，多节点部署时将极容易遇到 `database is locked`。

3. **测试环境的零污染约束**：
   - 如果调度器涉及时间，测试中将不可避免需要 `time.Sleep` 或 Mock 虚拟时钟。
   - 如果涉及物理表排队，如何在单体测试内存库中防竞态？

## 3. 架构师方案推荐 (Recommendation)

考虑到 GoDeployer 当前的定位和 SQLite 存储的局限性，我推荐 **“持久化状态辅助 + 内存信号量控制”** 结合方案：
- **不引入重量级 Cron**：除非你有硬性要求，否则保持职责单一。
- **并发锁 (Semaphore / Worker Pool)**：在 `DeployEngine` 内部启动全局唯一的调度队列（比如允许 `concurrency = 3`）。
- **状态同步**：收到 API 请求后，在 DB 插入 `Status = pending`，然后将 `TaskID` 推入带缓冲的 Channel 中。`Engine` 后台的 `DeploymentWorker` 会从 Channel 拿任务执行，执行完成后捞取下一个。

---
请针对以上质疑进行回复与明确，只有 100% 对齐我们才能进入 IR 纳米级计划编写。
