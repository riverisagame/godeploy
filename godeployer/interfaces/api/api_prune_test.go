package api

import (
	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/sqlite"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAPI_SystemPrune_Permissions 验证系统清理接口的管理员权限校验
func TestAPI_SystemPrune_Permissions(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	// 1. 无 Token 访问，预期 401
	req, _ := http.NewRequest("POST", "/api/system/prune", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous access, got %d", w.Code)
	}

	// 2. Viewer 角色访问，预期 403
	req2, _ := http.NewRequest("POST", "/api/system/prune", nil)
	viewerToken, _ := application.GenerateToken("viewer", "viewer", "test-secret-key-12345", 5*time.Second)
	req2.Header.Set("Authorization", "Bearer "+viewerToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer user, got %d", w2.Code)
	}
}

// TestAPI_SystemPrune_OrphanCleanup 验证手动清理孤儿日志和差异文件的逻辑
func TestAPI_SystemPrune_OrphanCleanup(t *testing.T) {
	// 创建测试用例专属临时目录以防止污染真实磁盘
	tmpLogDir, err := os.MkdirTemp("", "godeployer_prune_test_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpLogDir)

	// 自定义路由，将 LogPath 指向临时目录
	db, err := sqlite.InitDB("file:mem_prune_1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	// 插入一个正常任务
	_, err = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, username, status, user_id, config_snapshot, created_at)
		VALUES (101, 'test-app', 'testing', '20260529120000', 'abcdef123456', 'admin', 'success', 1, '{}', '2026-05-29T18:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	// 物理生成对应的文件
	// 1. 正常任务 101 的日志文件
	task101Log := filepath.Join(tmpLogDir, "task_101.log")
	_ = os.WriteFile(task101Log, []byte("normal task 101 log"), 0644)

	// 2. 正常任务 101 的 diff 缓存文件 (CRM/YYYYMM/task_id_diff.log 目录结构)
	diffDir101 := filepath.Join(tmpLogDir, "diffs", "projects", "test-app", "202605")
	_ = os.MkdirAll(diffDir101, 0755)
	task101Diff := filepath.Join(diffDir101, "task_101_diff.log")
	_ = os.WriteFile(task101Diff, []byte("normal diff content"), 0644)

	// 3. 磁盘孤儿任务 999 的日志文件 (数据库中没有 999 号任务)
	task999Log := filepath.Join(tmpLogDir, "task_999.log")
	_ = os.WriteFile(task999Log, []byte("orphaned task 999 log"), 0644)

	// 4. 磁盘孤儿任务 999 的 diff 缓存文件
	diffDir999 := filepath.Join(tmpLogDir, "diffs", "projects", "test-app", "202605")
	task999Diff := filepath.Join(diffDir999, "task_999_diff.log")
	_ = os.WriteFile(task999Diff, []byte("orphaned diff content"), 0644)

	// 运行清理
	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:  "test-secret-key-12345",
			LogPath:    tmpLogDir,
			SSHKeyPath: "./test_keys/id_rsa",
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {ID: "test-app", Name: "Mock Test App"},
		},
	}

	engine := application.NewDeployEngine(db, nil)
	r := SetupRoutes(mockConfig, db, engine)

	req, _ := http.NewRequest("POST", "/api/system/prune", nil)
	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 验证返回结果
	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res["pruned_orphans_count"] == nil {
		t.Errorf("expected pruned_orphans_count in response, got %v", res)
	}

	// 验证文件存在性：
	// 正常的 101 文件必须保留
	if _, err := os.Stat(task101Log); os.IsNotExist(err) {
		t.Error("expected task_101.log to be preserved, but it was deleted")
	}
	if _, err := os.Stat(task101Diff); os.IsNotExist(err) {
		t.Error("expected task_101_diff.log to be preserved, but it was deleted")
	}

	// 孤儿的 999 文件必须已被物理清理
	if _, err := os.Stat(task999Log); !os.IsNotExist(err) {
		t.Error("expected task_999.log (orphan) to be physically pruned, but it still exists")
	}
	if _, err := os.Stat(task999Diff); !os.IsNotExist(err) {
		t.Error("expected task_999_diff.log (orphan) to be physically pruned, but it still exists")
	}
}

// TestAPI_DiffCache_MaxSizeLimit 验证获取 Diff 时支持大文件配置截断及持久化缓存读取
func TestAPI_DiffCache_MaxSizeLimit(t *testing.T) {
	tmpLogDir, err := os.MkdirTemp("", "godeployer_diff_test_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpLogDir)

	db, err := sqlite.InitDB("file:mem_prune_2?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	// 插入测试任务
	_, err = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, username, status, user_id, config_snapshot, created_at)
		VALUES (201, 'test-app', 'testing', '20260529120000', 'abcdef123456', 'admin', 'success', 1, '{}', '2026-05-29T18:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert task 201: %v", err)
	}
	// 插入其前置成功任务作为基准
	_, err = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, username, status, user_id, config_snapshot, created_at)
		VALUES (200, 'test-app', 'testing', '20260529110000', 'aaaaaa111111', 'admin', 'success', 1, '{}', '2026-05-29T17:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert task 200: %v", err)
	}

	// 在临时目录提前植入 diff 缓存快照
	diffDir := filepath.Join(tmpLogDir, "diffs", "projects", "test-app", "202605")
	_ = os.MkdirAll(diffDir, 0755)
	cachedDiffPath := filepath.Join(diffDir, "task_201_diff.log")

	// 模拟写入一个极其巨大的 diff 数据 (超限截断测试)
	largeDiffContent := make([]byte, 5000)
	for i := range largeDiffContent {
		largeDiffContent[i] = 'A'
	}
	_ = os.WriteFile(cachedDiffPath, largeDiffContent, 0644)

	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:      "test-secret-key-12345",
			LogPath:        tmpLogDir,
			SSHKeyPath:     "./test_keys/id_rsa",
			DiffMaxSizeKB:  3, // 配置最大 3KB 限制（即 3072 字节）
			DiskMinSpaceMB: 1,
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {ID: "test-app", Name: "Mock Test App"},
		},
	}

	engine := application.NewDeployEngine(db, nil)
	r := SetupRoutes(mockConfig, db, engine)

	req, _ := http.NewRequest("GET", "/api/tasks/201/diff", nil)
	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	diffOutput := res["diff"]

	// 验证是否包含缓存中的内容，并且由于超过 3KB 的限制而被截断
	if len(diffOutput) > 3072+200 { // 允许有一些溢出指示字符的额外长度
		t.Errorf("expected diff to be truncated around 3072 bytes, but got length %d", len(diffOutput))
	}
}
