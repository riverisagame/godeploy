package notifier

import (
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	EventDeployStart   EventType = "deploy:start"
	EventDeploySuccess EventType = "deploy:success"
	EventDeployFailed  EventType = "deploy:failed"
)

type DeployEvent struct {
	Type     EventType `json:"type"`
	TaskId   int64     `json:"task_id"`
	Project  string    `json:"project"`
	Env      string    `json:"env"`
	Operator string    `json:"operator"`
	Commit   string    `json:"commit"`
}

type Notifier interface {
	Send(event *DeployEvent) error
}

type EventBus struct {
	mu        sync.RWMutex
	notifiers []Notifier
	ch        chan *DeployEvent
	wg        sync.WaitGroup
	closed    bool
}

func NewEventBus() *EventBus {
	return &EventBus{
		notifiers: make([]Notifier, 0),
		ch:        make(chan *DeployEvent, 1000), // 缓冲区大小 1000
	}
}

func (b *EventBus) Register(n Notifier) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.notifiers = append(b.notifiers, n)
}

func (b *EventBus) Publish(event *DeployEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	select {
	case b.ch <- event:
	default:
		// channel full, drop event
	}
}

// StartEventConsumer 启动消费者协程，从 Channel 消费事件并触发所有通知器。
func (b *EventBus) StartEventConsumer(workers int) {
	b.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer b.wg.Done()
			for event := range b.ch {
				b.mu.RLock()
				notifiers := make([]Notifier, len(b.notifiers))
				copy(notifiers, b.notifiers)
				b.mu.RUnlock()

				for _, n := range notifiers {
					// @Ref: docs/sps/plans/20260527_m3_notifier_ir.md | @Date: 2026-05-27
					err := n.Send(event)
					if err != nil {
						// 局部重试
						time.Sleep(1 * time.Second)
						_ = n.Send(event)
					}
				}
			}
		}()
	}
}

// Close 优雅停机
func (b *EventBus) Close(timeout time.Duration) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	close(b.ch)
	b.mu.Unlock()

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for event bus to close")
	}
}
