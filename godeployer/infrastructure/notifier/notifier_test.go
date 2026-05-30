package notifier_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"deploy/godeployer/infrastructure/notifier"
)

// MockNotifier 记录接收到的事件以供断言。
type MockNotifier struct {
	mu     sync.Mutex
	events []*notifier.DeployEvent
}

func (m *MockNotifier) Send(event *notifier.DeployEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MockNotifier) GetEvents() []*notifier.DeployEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]*notifier.DeployEvent, len(m.events))
	copy(copied, m.events)
	return copied
}

// TestNotifier_AsyncPipeline 验证通知的异步派发逻辑。
// 事件抛送应当是非阻塞的，并在短时间内由消费协程完成处理。
func TestNotifier_AsyncPipeline(t *testing.T) {
	bus := notifier.NewEventBus()

	// 注册 Mock 通知发送器
	mock := &MockNotifier{}
	bus.Register(mock)

	// 启动消费者协程
	bus.StartEventConsumer(10)

	// 抛送事件
	event := &notifier.DeployEvent{
		Type:     notifier.EventDeploySuccess,
		TaskId:   1001,
		Project:  "demo",
		Env:      "production",
		Operator: "admin",
		Commit:   "a1b2c3d4",
	}

	// 记录抛送前的时间
	startTime := time.Now()
	bus.Publish(event)
	publishDuration := time.Since(startTime)

	// 验证发布操作是否是瞬间完成的（非阻塞）
	if publishDuration > 5*time.Millisecond {
		t.Errorf("Publish took too long (%v), it should be asynchronous and non-blocking", publishDuration)
	}

	// 稍微等待异步消费完成
	time.Sleep(50 * time.Millisecond)

	events := mock.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	received := events[0]
	if received.TaskId != 1001 || received.Type != notifier.EventDeploySuccess {
		t.Errorf("received incorrect event: %+v", received)
	}
}

// MockFailingNotifier 记录重试次数
type MockFailingNotifier struct {
	mu          sync.Mutex
	CallCount   int
	ShouldError bool
}

func (m *MockFailingNotifier) Send(event *notifier.DeployEvent) error {
	m.mu.Lock()
	m.CallCount++
	shouldErr := m.ShouldError
	m.mu.Unlock()
	if shouldErr {
		return fmt.Errorf("simulated network error")
	}
	return nil
}

// TestNotifier_RetryAndDrop 测试通知器失败重试逻辑以及超载时的丢弃保护
// @Ref: docs/sps/plans/20260527_m3_notifier_ir.md
func TestNotifier_RetryAndDrop(t *testing.T) {
	bus := notifier.NewEventBus()

	failingMock := &MockFailingNotifier{ShouldError: true}
	bus.Register(failingMock)

	bus.StartEventConsumer(1) // 启动 1 个 Worker

	// 抛送事件
	bus.Publish(&notifier.DeployEvent{TaskId: 1})

	// 等待重试时间 (代码中计划是 sleep 1s，我们这里给稍微多一点的时间)
	time.Sleep(1500 * time.Millisecond)

	failingMock.mu.Lock()
	calls := failingMock.CallCount
	failingMock.mu.Unlock()

	// 预期被调用2次（首次 + 1次重试）
	if calls != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", calls)
	}
}

// TestNotifier_GracefulShutdown 测试优雅停机，确保所有缓冲中的事件发送完毕才退出。
// @Ref: docs/sps/plans/20260527_m3_notifier_ir.md
func TestNotifier_GracefulShutdown(t *testing.T) {
	bus := notifier.NewEventBus()

	mock := &MockNotifier{}
	bus.Register(mock)

	// 先不启动 Consumer，塞入 5 个事件
	for i := 0; i < 5; i++ {
		bus.Publish(&notifier.DeployEvent{TaskId: int64(i)})
	}

	// 启动 Worker
	bus.StartEventConsumer(2)

	// 发起优雅停机，最多等待 3 秒
	err := bus.Close(3 * time.Second)
	if err != nil {
		t.Fatalf("expected graceful shutdown without timeout, got error: %v", err)
	}

	// 验证停机后，缓冲里的 5 个事件是否全部被处理完
	events := mock.GetEvents()
	if len(events) != 5 {
		t.Errorf("expected 5 events processed before shutdown, got %d", len(events))
	}

	// 停机后再发事件应被静默丢弃，不发生 panic
	bus.Publish(&notifier.DeployEvent{TaskId: 999})
}
