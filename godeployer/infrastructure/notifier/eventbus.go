// ============================================================
// 文件：eventbus.go
// 作用：📢 事件总线——当有事情发生时，通知大家！
//
// 什么是"事件总线"（Event Bus）？
// 就像学校的广播系统！🎤
// - 校长室（发布者）说："下午停电！"
// - 喇叭（事件总线）播出通知
// - 每个教室（订阅者）都收到消息
//
// 在 GoDeploy 中，部署任务状态变化时（开始、成功、失败），
// 通过事件总线广播，WebSocket 收到后推送到前端页面，
// 这样用户就能实时看到部署进度了！
//
// 设计模式：发布-订阅（Pub/Sub）
// - Publish（发布）：发出一条消息
// - Notifier（通知器）：订阅消息，收到后执行操作
// - Channel（通道）：消息的"缓冲区"
// ============================================================

package notifier

import (
	"fmt"   // ✏️ 格式化
	"sync"  // 🔒 并发控制
	"time"  // ⏰ 超时控制
)

// ============================================================
// 📋 事件类型和事件结构
// ============================================================

// EventType 事件类型的字符串别名
type EventType string

// 🎯 预定义的事件类型常量
const (
	EventDeployStart   EventType = "deploy:start"   // 🚀 部署开始了
	EventDeploySuccess EventType = "deploy:success" // ✅ 部署成功了
	EventDeployFailed  EventType = "deploy:failed"  // ❌ 部署失败了
)

// DeployEvent 部署事件的结构体——一条完整的"通知"
type DeployEvent struct {
	Type     EventType `json:"type"`     // 📋 事件类型（开始/成功/失败）
	TaskId   int64     `json:"task_id"`    // 🔖 任务 ID
	Project  string    `json:"project"`   // 📁 项目名称
	Env      string    `json:"env"`        // 🌍 环境名称
	Operator string    `json:"operator"`   // 👤 操作人
	Commit   string    `json:"commit"`     // 🔖 Git 提交
}

// ============================================================
// 📡 Notifier：通知器接口
//
// 任何想要"收到事件通知"的东西，实现这个接口就行。
// 比如：WebSocket 推送器、日志记录器、邮件通知器……
// 这叫"面向接口编程"——不依赖具体实现！
// ============================================================

type Notifier interface {
	Send(event *DeployEvent) error // 发送事件
}

// ============================================================
// 🚌 EventBus：事件总线
//
// 核心组件！它管理着：
// - notifiers：所有订阅者列表
// - ch：事件通道（缓冲区 1000 条，满了就丢弃，防止阻塞）
// - workers：消费事件的"工人"数量
//
// 工作流程：
// 1. Publish（发布事件）→ 放入通道
// 2. StartEventConsumer（消费者）→ 从通道取出事件
// 3. 逐个通知所有订阅者（notifiers）
// 4. 如果某个订阅者失败，重试一次
// ============================================================

type EventBus struct {
	mu        sync.RWMutex        // 🔒 读写锁：保护 notifiers 和 closed
	notifiers []Notifier          // 📋 所有已注册的通知器
	ch        chan *DeployEvent   // 📥 事件通道（缓冲区 1000）
	wg        sync.WaitGroup      // 👥 等待所有消费者完成
	closed    bool                // 🚫 是否已关闭
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		notifiers: make([]Notifier, 0),
		ch: make(chan *DeployEvent, 1000), // 📥 缓冲区 1000 条
	}
}

// Register 注册一个新的通知器
// 就像教室新装了一个喇叭，安装到广播系统里
func (b *EventBus) Register(n Notifier) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.notifiers = append(b.notifiers, n)
}

// Publish 发布一条事件到总线
// 如果通道满了（缓冲区耗尽），直接丢弃事件，不阻塞！
// 因为部署日志很重要，但通知丢了总比系统卡住好~
func (b *EventBus) Publish(event *DeployEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return // 已经关闭了，不收新消息
	}

	select {
	case b.ch <- event: // ✅ 放入通道
	default:
		// ❌ 通道满了，丢弃事件（防止阻塞发布者）
	}
}

// StartEventConsumer 启动消费者协程，从 Channel 消费事件并触发所有通知器。
func (b *EventBus) StartEventConsumer(workers int) {
	b.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer b.wg.Done()
			// 不断从通道中取出事件
			for event := range b.ch {
				// 把 notifiers 列表复制一份（防止遍历过程中被修改）
				b.mu.RLock()
				notifiers := make([]Notifier, len(b.notifiers))
				copy(notifiers, b.notifiers)
				b.mu.RUnlock()

				// 逐个通知所有订阅者
				for _, n := range notifiers {
					// @Ref: docs/sps/plans/20260527_m3_notifier_ir.md | @Date: 2026-05-27
					err := n.Send(event)
					if err != nil {
						// 失败了？重试一次！（局部重试策略）
						time.Sleep(1 * time.Second) // 等 1 秒再试
						_ = n.Send(event)            // 忽略重试错误
					}
				}
			}
		}()
	}
}

// Close 优雅关闭事件总线
func (b *EventBus) Close(timeout time.Duration) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	close(b.ch) // 关闭通道——消费者会退出循环
	b.mu.Unlock()

	// 等待所有消费者处理完当前事件
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil // ✅ 正常关闭
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for event bus to close") // ⏰ 超时
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是"发布-订阅"模式？
//    A: 发布者只管发消息，不用管谁在收。
//       订阅者只管收消息，不用管谁发的。
//       就像公众号：作者发文章，你收到推送~
//
// 2. Q: 事件通道的缓冲区为什么是 1000？
//    A: 如果突然有很多事件同时产生（比如同时部署 50 个项目），
//       缓冲区可以防止事件丢失。1000 是一个安全阀值~
//
// 中级：
// 3. Q: 为什么通道满了要丢弃事件而不是阻塞？
//    A: 发布者（部署引擎）需要尽快完成状态更新，
//       如果因为通知发不出去而卡住，会影响部署主流程。
//       丢几个通知比让部署卡住好！
//
// 4. Q: 为什么要复制 notifiers 列表再遍历？
//    A: 防止"并发修改"问题！如果在遍历过程中有人注册了新通知器，
//       直接遍历原列表会导致 panic。复制一份可以安全遍历~
//
// 高级：
// 5. Q: 重试策略为什么只重试一次？
//    A: 简单的"尽力而为"策略。部署事件通知失败一次可能是瞬时错误，
//       如果重试还失败，说明通知器本身有问题，重试更多次也无意义~
// ============================================================
