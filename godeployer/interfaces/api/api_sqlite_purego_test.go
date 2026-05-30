package api

import (
	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/db"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAPI_SqlitePureGo_Connection 验证纯 Go 驱动在 CGO_ENABLED=0 模式下的连接与基础查询能力
func TestAPI_SqlitePureGo_Connection(t *testing.T) {
	// 初始化内存数据库，使用我们 InitDB 封装
	dsn := fmt.Sprintf("file:mem_purego_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, _, err := db.InitTestDB(dsn)
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	defer db.Close()

	// 执行一次基本查询以验证驱动活跃状态
	var dummy int
	err = db.QueryRow("SELECT 1").Scan(&dummy)
	if err != nil {
		t.Fatalf("basic query failed: %v", err)
	}

	if dummy != 1 {
		t.Errorf("expected 1, got %d", dummy)
	}
}

// TestAPI_DiffSemaphoreThrottling 验证并发 Diff 请求的限流控制机制 (RED 阶段期望失败)
func TestAPI_DiffSemaphoreThrottling(t *testing.T) {
	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_sem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	defer db.Close()

	_, _ = db.Exec(`DELETE FROM deploy_tasks`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (901, "test-sem-app", "testing", "20260530150000", "aaaaaa111111", 1, "admin", "success", "{}", "2026-05-30T15:00:00Z", "commit")
	`)

	cfg := &domain.Config{
		Projects: map[string]domain.ProjectConfig{
			"test-sem-app": {ID: "test-sem-app", Repo: "mock-repo"},
		},
		Global: domain.GlobalConfig{
			JWTSecret: "sem-secret-key-12345",
			LogPath:   t.TempDir(),
		},
	}

	engine := application.NewDeployEngine(taskRepo, nil)
	r := SetupRoutes(cfg, db, taskRepo, engine)

	adminToken, _ := application.GenerateToken("admin", "admin", "sem-secret-key-12345", 5*time.Second)

	// 1. 提前手动占满信号量 (容量为 5)
	for i := 0; i < 5; i++ {
		diffSemaphore <- struct{}{}
	}
	defer func() {
		// 退出前必须清空释放，防止干扰其他正常单元测试
		for i := 0; i < 5; i++ {
			<-diffSemaphore
		}
	}()

	// 2. 发起第 6 个请求，断言必须因为信号量满而触发 3 秒排队超时，从而快速退化返回 429
	req, _ := http.NewRequest("GET", "/api/tasks/901/diff?file=main.go&diff_type=git_log", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	// 在测试中我们为了加快测试执行，不希望干等 3 秒排队超时，所以我们在测试前可以临时将 diff 接口的排队逻辑做测试级超时映射，
	// 但此处可以直接通过 ServeHTTP 发送，因为超时为 3 秒在合理范围。
	startTime := time.Now()
	r.ServeHTTP(w, req)
	duration := time.Since(startTime)

	t.Logf("Response code: %d, time taken: %v, body: %s", w.Code, duration, w.Body.String())

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429 (Too Many Requests) when semaphore is full, but got %d", w.Code)
	}
	if duration < 3*time.Second {
		t.Errorf("expected at least 3 seconds timeout delay before returning 429, but took only %v", duration)
	}
}
