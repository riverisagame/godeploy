# [IR] 纳米级执行计划: Milestone 3 - Notifier 高并发与优雅停机改造

## 1. 设计目标与对齐
基于 [SCAN] 阶段共识：
- **模型**：固定 `Worker Pool` (默认 10 个) 处理事件，消除无界 `go func`。
- **重试**：针对单一 `Notifier` 失败时，执行最多 1 次重试 (间隔 1 秒)。
- **发布策略**：加大缓冲至 `1000`，`Publish` 采用非阻塞的 `select` 配合 `default` 丢弃并记录错误日志。
- **生命周期**：提供 `Close(timeout)` 方法，触发并等待所有内存通道中的事件消费完成或达到超时。

---

## 2. 纳米级执行计划 (Atomized Steps)

### Step 1: `EventBus` 结构体升级 (RED & GREEN)
**文件**: `godeployer/notifier.go`
**动作**:
- 给 `EventBus` 结构体新增：
  - `wg sync.WaitGroup`
  - `closed bool`
- 修改 `NewEventBus()`：`ch` 的缓冲区调整为 `1000`。

**测试预期 (RED)**：
- 编写 `TestNotifier_GracefulShutdown` 验证 `Close()`。

### Step 2: 实现 `Publish` 的非阻塞与安全关闭 (GREEN)
**文件**: `godeployer/notifier.go`
**动作**:
- 修改 `Publish(event *DeployEvent)`：
  - 加 `b.mu.RLock()` 校验 `b.closed`，如果关闭直接返回。
  - 用 `select` 结构尝试写入 `b.ch`。如果 `default` 分支命中，输出警告日志丢弃事件。

### Step 3: `StartEventConsumer` 重构为 Worker Pool (GREEN)
**文件**: `godeployer/notifier.go`
**动作**:
- `StartEventConsumer(workers int)` 接收参数 (默认用 10)。
- 内部 `b.wg.Add(workers)` 并循环启动 `workers` 数量的 `go func()`。
- Worker 内部：
  - `defer b.wg.Done()`
  - `for event := range b.ch` 循环获取事件（通道关闭且清空后自动退出）。
  - 内部获取注册的 notifiers。
  - 遍历 notifiers：
    - 同步执行 `err := n.Send(event)`。
    - 如果 `err != nil`，执行重试：`time.Sleep(1 * time.Second)` 然后再 `n.Send(event)`。若仍失败，输出错误日志。

### Step 4: 实现 `Close(timeout time.Duration)` 优雅停机 (GREEN)
**文件**: `godeployer/notifier.go`
**动作**:
- `Close()` 方法内部：
  - 加锁设置 `b.closed = true`，并执行 `close(b.ch)`。
  - 启动一个协程执行 `b.wg.Wait()`，通过通道通知。
  - 主协程 `select` 等待 `wg.Wait()` 完成或 `time.After(timeout)` 超时。

### Step 5: 全局停机钩子接入 (GREEN)
**文件**: `godeployer/main.go`
**动作**:
- 在现有的 HTTP Server 优雅停机流程中（如果存在）或应用退出前，增加 `eventBus.Close(5 * time.Second)` 的调用，保障进程退出时不丢失日志。

---

## 3. 约束检查
- [x] 所有修改集中在 `notifier.go`。
- [x] 不会破坏外层 `Publish` 的函数签名（保证无损过渡）。
- [x] 完美满足防内存泄漏与防并发爆炸要求。
