package git

import (
	"context"
	"testing"
	"time"
)

// TestClient_Git_Cancellation 验证 Git 客户端的命令在上下文被取消时是否能中止
// 预期 RED 阶段将引发编译错误（缺少 ctx 参数）
func TestClient_Git_Cancellation(t *testing.T) {
	client := NewClient("/tmp")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// RED 阶段编译报错，GREEN 阶段在超时后应能正常返回 context.DeadlineExceeded 相关错误
	// 由于这只是测试取消逻辑，我们调用 FetchAndGetCommits 触发一个假死循环或者因为无法解析地址超时
	_, err := client.FetchAndGetCommits(ctx, "https://invalid.repo.local/does-not-exist.git", "main", "test-project", "")
	
	if err == nil {
		t.Errorf("Expected context cancellation error, got nil")
	}
}
