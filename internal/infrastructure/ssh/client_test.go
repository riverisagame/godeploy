package ssh

import (
	"context"
	"testing"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
)

func TestClient_RunCommand_Cancellation(t *testing.T) {
	client := NewClient()

	srv := &domain.Server{
		ID:   1,
		IP:   "127.0.0.1",
		Port: 2222,
		User: "test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	logChan := make(chan string, 10)
	
	// 期望这里由于传入 ctx 发生编译错误 (RED 阶段)
	// 在 GREEN 阶段修复后，期望由于 context 提前取消而返回错误，而不是一直阻塞
	err := client.RunCommand(ctx, srv, "sleep 10", logChan)
	if err == nil {
		t.Errorf("Expected error due to context cancellation, got nil")
	}
}
