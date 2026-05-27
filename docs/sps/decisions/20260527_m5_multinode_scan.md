# Milestone 5: 多节点集群部署 (Multi-node Cluster Deployment) SCAN 审计报告

## 1. 需求深挖与对齐 (Requirement Alignment)
**目标**：重构 `DeployEngine.RunDeploy`，将现有的【串行单节点部署】升级为【高并发的多节点集群部署】，并保证集群状态的“最终一致性”（即要么全部成功，要么全部失败/回滚）。

### 现状痛点
当前 `engine.go` (line 323) 的逻辑是 `for _, srv := range targetEnv.Servers`：
1. **串行阻塞**：10 台机器部署需要 10 倍时间。
2. **状态机撕裂**：如果第 1 台机器部署且软链接切换成功，但第 2 台机器 Rsync 失败直接 `return` 并标记 `failed`，此时集群版本已经不一致（Split-Brain），没有自动的跨节点状态补偿！

## 2. 硬核审计 (Hardcore Audit)

### 2.1 实现架构提案 (两阶段提交 / 2PC 变体)
为了避免“脑裂”，多机部署必须引入类似两阶段提交（Two-Phase Commit）的逻辑：
- **Phase 1: 并发分发 (Prepare)**
  - 使用 `errgroup` 或 `sync.WaitGroup` + 错误通道，**并发**向所有 Server 执行 `Rsync`（上传到 `releases/{release_name}`）。
  - 若任何一台失败，则触发集群级 Abort，清理已经上传但未激活的 Release 目录，直接标记 `failed`。
- **Phase 2: 原子激活 (Commit)**
  - Phase 1 全通过后，**并发**向所有 Server 执行 `SwitchSymlink`。
  - 若有机器切换失败，进入灾难恢复模式。

### 2.2 自我攻击 (Adversarial Critique)
- **攻击1：Phase 2 切换时的网络中断**。如果在并发执行 `SwitchSymlink` 时，一半成功了，另一半因为 SSH 断开失败了，怎么办？
  - *防御*：此时必须触发**集群级强回滚**（Distributed Rollback）。读取上一个成功的 `release_name`，对所有当前尝试的 Server 执行强制回退，并在数据库标记为 `rollback_failed` 或 `failed`，同时调用 Notifier 触发 P0 级报警。
- **攻击2：并发资源的雪崩**。如果一个项目配置了 100 台服务器，`RunDeploy` 同时拉起 100 个 `SSHExecutor.Rsync`，会瞬间榨干宿主机的文件描述符（FD）和带宽，并可能导致内存 OOM。
  - *防御*：必须在 `RunDeploy` 内部引入 `WorkerPool` 或信号量（Semaphore，如 `make(chan struct{}, 10)`）对单次任务的下发并发度进行限流。
- **攻击3：多节点后置 Hook 的不可靠性**。`AfterSymlink` 钩子如果在 10 台机器上执行，部分成功部分失败，此时算成功还是失败？回滚软链接吗？
  - *防御*：后置 Hook 失败通常被认为是“部署成功但环境异常”，软链接已不可逆向简单回退（可能已经产生了数据库脏数据）。需明确定义：Symlink 成功即算作版本更新成功，Hook 失败只触发 Warning 通知，不回滚集群软链接。

### 2.3 性能对冲 (Performance Hedging)
- **Rsync 带宽复用**：并发分发相同的文件给多台机器属于 IO 密集型。需要在 `SSHExecutor.Rsync` 的执行上下文中，严格控制超时时间（例如 context timeout），防止僵尸进程。
- **并发状态竞争**：多机部署的日志输出必须加锁（`writeLog` 已有互斥锁，但多线程并发写入时可能导致日志交错乱序），需要给日志打上前缀 `[Host: 10.0.0.1]` 以供人类溯源。

## 3. 对现有功能的影响 (Impact Analysis)
- `engine.go: RunDeploy`: 将发生彻底的骨架级别重构。由原来的扁平循环变成复杂的并发等待和补偿回滚逻辑。
- `engine.go: RunRollbackToTask`: 同样需要支持并发并发回滚，否则回滚慢也无法忍受。
- `engine_test.go`: 需要编写多 Server 并发部署的 Mock 测试，模拟部分节点失败、网络断开等边缘场景。无 DDL 变更。

## 4. 需要您确认的核心问题
1. **并发限流**：单次部署任务内部，向多台机器分发的最大并发数（Batch Size）建议设置为 `5` 还是 `10`？
2. **后置Hook失败容忍度**：如果在 Phase 2 (Symlink) 全部成功后，部分机器的 `AfterSymlink` (例如 `php artisan cache:clear`) 失败了。我们是将整个任务标记为 `success` 附带警告，还是标记为 `failed`（但不回滚软链接）？
3. **Rollback 容错**：如果在灾难恢复（自动回退上一个版本）时再次发生部分节点断网，是否接受标记任务为 `critical_brain_split`（脑裂，需人工介入）？
