# Milestone 5: Multi-node Cluster Deployment - Implementation Plan

## 1. 目标
将 `engine.go` 中的串行部署重构为高并发、受控、支持最终一致性的多节点集群部署，并支持网络中断下的全局灾难回滚。

## 2. 涉及文件
- `godeployer/engine.go`
- `godeployer/engine_test.go`

## 3. 具体修改步骤 (Nano-plan)

### Step 1: 增强 `writeLog` 线程安全性 (原子操作, ~10行)
**目标文件**: `godeployer/engine.go` (`RunDeploy` 函数内部)
- **动作**: 引入 `var logMu sync.Mutex`。
- **动作**: 修改 `writeLog` 闭包，在向 `logFile.WriteString` 写入时增加加锁与解锁操作，防止多节点并发输出时产生日志交错乱序。

### Step 2: 引入 Phase 1 (Concurrent Rsync) 并发分发 (原子操作, ~30行)
**目标文件**: `godeployer/engine.go`
- **动作**: 移除原有的扁平 `for _, srv := range targetEnv.Servers` 串行逻辑。
- **动作**: 定义并发控制：`sem := make(chan struct{}, 10)` (用户确认的最大并发度为 10)。
- **动作**: 使用 `sync.WaitGroup` 和互斥锁 `var phase1Err error`。
- **动作**: 并发执行 `Rsync`。如果任何节点失败，使用 `logMu` 记录日志，并设置 `phase1Err`。
- **动作**: 阻塞等待 WaitGroup。如果 `phase1Err != nil`，直接更新状态为 `failed` 并 `return` (此时未触碰符号链接，无需回滚)。

### Step 3: 引入 Phase 2 (Concurrent Symlink & 灾难回滚) (原子操作, ~40行)
**目标文件**: `godeployer/engine.go`
- **动作**: 在 Phase 1 成功后，并发发起 Phase 2 (`SwitchSymlink`)。
- **动作**: 同理使用 WaitGroup 和 `sem` 并发控制，记录是否有任何节点 `SwitchSymlink` 失败。
- **动作**: **回滚机制**：若 Phase 2 发生任何失败，触发**全局灾难回滚** (Distributed Rollback)：
  - 从数据库提取 `prevReleaseName`。如果为空，则直接抛出，任务状态标记为 `failed`。
  - 如果存在，并发向所有节点执行 `e.SwitchSymlink(srv, prevReleaseName)`。
  - **脑裂判定**：如果回滚过程中又有节点失败，调用 `e.UpdateTaskStatus(taskID, "critical_brain_split")` 标记人工介入，并直接 `return`。
  - 回滚成功则更新状态为 `failed` 并 `return`。

### Step 4: 引入 Phase 3 (Concurrent AfterSymlink Hooks) (原子操作, ~20行)
**目标文件**: `godeployer/engine.go`
- **动作**: 如果 Phase 2 全部成功，并发执行 `AfterSymlink`。
- **动作**: 根据用户指令，Hook 失败只记录日志 (WriteLog)，**不触发回滚**，也不会中断主流程。
- **动作**: 最终统一调用 `e.UpdateTaskStatus(taskID, "success")` 并在 `EventBus` 中广播。

### Step 5: TDD 单元测试验证 (RED 阶段准备)
**目标文件**: `godeployer/engine_test.go`
- **动作**: 添加并发断网测试，模拟部分 Server `SwitchSymlink` 失败时，触发 `critical_brain_split`。
- **动作**: 添加并发测试，模拟部分 Server Rsync 失败时，触发 `failed`。
- **动作**: 添加并发测试，模拟 Hook 失败时，状态依然为 `success`。

## 4. 退出条件
通过 `go test -v .` 测试并发场景下的所有分支，并确认 `critical_brain_split` 等状态流转正确。
