# [SCAN] Milestone 3: Notifier 高并发改造选型对齐

## 1. 现状分析 (实现与审计)
当前的 `godeployer/notifier.go` 中：
```go
func (b *EventBus) StartEventConsumer() {
	for event := range b.ch {
		// ...
		for _, n := range notifiers {
			go func(notifier Notifier, ev *DeployEvent) {
				_ = notifier.Send(ev)
			}(n, event)
		}
	}
}
```
**自我攻击 & 性能对冲**：
1. **Goroutine 泄漏风险 (Unbounded Goroutines)**：每触发一个事件，针对每一个注册的 Notifier，系统都会启动一个新的 Goroutine。在极端高并发（例如集中 Push 引发大量 webhook 或大量批量部署）的情况下，Goroutine 数量可能激增，引发 OOM 或系统抖动。
2. **丢失事件 (No Graceful Shutdown)**：主程序退出时，若 `b.ch` 内仍有消息未消费，或者刚启动的 `go func` 还没有执行完毕，系统将直接杀掉它们，导致事件（如钉钉、Webhook 通知）丢失。
3. **无重试与熔断机制**：`_ = notifier.Send(ev)` 忽略了错误。如果外部 API 暂时不可用（网络抖动、Rate Limit），通知直接丢弃，不具备容错性。

## 2. 改造方案选型 (Adversarial Protocol)
为了彻底解决上述技术债务 (Technical Debt)，需明确以下需求的边界，特向用户发起提问：

- **Q1 [并发控制]**：我们是采用 **Worker Pool (协程池)** 来限制最大并发通知数（例如启动固定 10-50 个 Worker 从共享的 Channel 抢占任务），还是引入类似 `golang.org/x/sync/semaphore` 的信号量机制？
- **Q2 [优雅停机]**：是否需要引入 `context.Context` 和 `sync.WaitGroup` 实现 Graceful Shutdown（收到 SIGTERM 时，最多等待 N 秒将内存中剩余的通知发送完毕再退出）？
- **Q3 [重试机制]**：由于是内存态队列，当外部接口（如飞书/钉钉）超时或返回 5xx 时，是否需要引入简单的指数退避重试（Retry）机制？如果多次重试依然失败，直接丢弃还是记录到本地日志？
- **Q4 [持久化]**：目前通道 `b.ch` 仅有 100 缓冲且存在于内存中，一旦崩溃这 100 条即刻丢失。这对于当前阶段是否可以接受？还是必须升级为基于 SQLite 或 Redis 的持久化事件队列？（鉴于最小改动原则，推荐先维持内存通道+优雅停机的方案，若非必要不引入新组件）

## 3. 现有功能影响范围 (Impact Radius)
本次改造将触及以下文件：
- `godeployer/notifier.go` (核心改动区)
- `godeployer/notifier_test.go` (重写验证逻辑)
- `godeployer/main.go` (由于可能引入优雅停机，需在主退出流程注入 `eventBus.Stop()`)

影响范围极小，均为旁路系统，不会阻断核心的主部署引擎 `engine.go` 和 API 接口 `api.go` 的运行。
