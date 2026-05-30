package api

import (
	"bytes"
	"database/sql"
	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/db"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupEnhanceTestRouter 初始化 Gin 测试路由，并将 repo 设为本地当前路径
func SetupEnhanceTestRouter(t *testing.T) (*gin.Engine, *sql.DB, func()) {
	gin.SetMode(gin.TestMode)

	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_enhance_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}

	repoDir := t.TempDir()
	_ = exec.Command("git", "init", repoDir).Run()
	func() {
		cmd := exec.Command("git", "config", "user.email", "test@test.com")
		cmd.Dir = repoDir
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = repoDir
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "--allow-empty", "-m", "init")
		cmd.Dir = repoDir
		_ = cmd.Run()
	}()

	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:  "test-secret-key-12345",
			LogPath:    "./test_logs",
			SSHKeyPath: "./test_keys/id_rsa",
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {
				ID:            "test-app",
				Name:          "Mock Test App",
				Repo:          repoDir, // 设为本地仓库路径，确保 git clone 本地能够成功
				WebhookSecret: "secret123",
				Branch:        "main",
				Environments: []domain.EnvironmentConfig{
					{
						ID:   "testing",
						Name: "Test Env",
						Servers: []domain.ServerConfig{
							{
								Host:       "127.0.0.1",
								Port:       22,
								User:       "mock-user",
								DeployTo:   "/tmp/mock-deploy",
								SSHKeyPath: "./test_keys/id_rsa",
							},
						},
					},
				},
			},
		},
	}

	engine := application.NewDeployEngine(taskRepo, nil)
	engine.StartDispatcher(1)
	r := SetupRoutes(mockConfig, db, taskRepo, engine)

	cleanup := func() {
		engine.Close(2 * time.Second)
		db.Close()
		_ = os.RemoveAll("demo_workspace") // 清理测试产生的临时缓存
	}

	return r, db, cleanup
}

// TestAPI_CreateTaskWithDescription 验证创建部署任务时，支持传入描述/备注字段并正确持久化
func TestAPI_CreateTaskWithDescription(t *testing.T) {
	r, db, cleanup := SetupEnhanceTestRouter(t)
	defer cleanup()

	// 构造创建任务的参数，带 description 和 extra_exclude
	reqBody := map[string]interface{}{
		"project_id":    "test-app",
		"env_id":        "testing",
		"commit_id":     "HEAD",
		"description":   "这是一个高级测试部署，包含了系统自愈和容量限制说明",
		"extra_exclude": "src/config.dev.ts,tests/*",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 201/200, got %d, body: %s", w.Code, w.Body.String())
	}

	var createdTask map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &createdTask)
	taskID := int64(createdTask["task_id"].(float64))

	// 直接从内存数据库查询，验证 description 和 extra_exclude 是否成功写入
	var desc, extraExclude string
	err := db.QueryRow("SELECT description, extra_exclude FROM deploy_tasks WHERE id = ?", taskID).Scan(&desc, &extraExclude)
	if err != nil {
		t.Fatalf("failed to query new fields from deploy_tasks: %v", err)
	}

	if desc != "这是一个高级测试部署，包含了系统自愈 and 容量限制说明" && desc != "这是一个高级测试部署，包含了系统自愈和容量限制说明" {
		t.Errorf("expected description to be saved, got '%s'", desc)
	}
	if extraExclude != "src/config.dev.ts,tests/*" {
		t.Errorf("expected extra_exclude to be saved, got '%s'", extraExclude)
	}
}

// TestAPI_PreviewDiffWithFileList 验证预览接口能够同时返回文件变更列表
func TestAPI_PreviewDiffWithFileList(t *testing.T) {
	r, _, cleanup := SetupEnhanceTestRouter(t)
	defer cleanup()

	// 触发预览 GET /api/projects/:id/preview_diff?from=HEAD~1&to=HEAD
	req, _ := http.NewRequest("GET", "/api/projects/test-app/preview_diff?from=HEAD~1&to=HEAD", nil)
	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	// 断言必须包含 files 字段列表
	filesVal, ok := res["files"]
	if !ok {
		t.Error("expected 'files' list in preview_diff response, but it was missing")
	}

	filesList, ok := filesVal.([]interface{})
	if !ok {
		t.Errorf("expected 'files' to be an array, got %T", filesVal)
	}

	// 至少包含一些本地 Git 变更文件，或者即使为空也应当存在 files 字段
	if len(filesList) == 0 {
		t.Log("Warning: files list is empty")
	}
}

// TestAPI_JSON_ChangesCache 验证新版 JSON 结构化快照获取与兼容性
func TestAPI_JSON_ChangesCache(t *testing.T) {
	tmpLogDir, err := os.MkdirTemp("", "godeployer_json_changes_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpLogDir)

	// 使用 InitDB 替代裸 sql.Open，防止因没有初始化权限表导致 403 Access Denied
	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_cache_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}
	defer db.Close()

	// 清理多余的默认插入，并插入指定任务记录
	_, _ = db.Exec(`DELETE FROM deploy_tasks`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at)
		VALUES (501, 'test-app', 'testing', '20260529120000', 'abcdef123456', 1, 'admin', 'success', '{}', '2026-05-29T18:00:00Z')
	`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at)
		VALUES (500, 'test-app', 'testing', '20260529110000', 'aaaaaa111111', 1, 'admin', 'success', '{}', '2026-05-29T17:00:00Z')
	`)

	// 植入高内聚 JSON 快照
	diffDir := filepath.Join(tmpLogDir, "diffs", "projects", "test-app", "202605")
	_ = os.MkdirAll(diffDir, 0755)
	jsonPath := filepath.Join(diffDir, "task_501_diff.log") // 保持原后缀以便无感升级

	mockChanges := map[string]string{
		"files": "M src/App.vue\nA src/components/TreeView.vue",
		"diff":  "diff --git a/src/App.vue b/src/App.vue\n...",
	}
	jsonData, _ := json.Marshal(mockChanges)
	_ = os.WriteFile(jsonPath, jsonData, 0644)

	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:      "test-secret-key-12345",
			LogPath:        tmpLogDir,
			SSHKeyPath:     "./test_keys/id_rsa",
			DiffMaxSizeKB:  2048,
			DiskMinSpaceMB: 1,
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {ID: "test-app", Name: "Mock Test App"},
		},
	}

	engine := application.NewDeployEngine(taskRepo, nil)
	r := SetupRoutes(mockConfig, db, taskRepo, engine)

	req, _ := http.NewRequest("GET", "/api/tasks/501/diff", nil)
	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	// 此时因为是懒加载模式，不传 file 时 diff 字段必为空
	if res["files"] != "M src/App.vue\nA src/components/TreeView.vue" {
		t.Errorf("expected files list match, got '%v'", res["files"])
	}
	if res["diff"] != "" {
		t.Errorf("expected empty diff under lazy-loading mode, got '%v'", res["diff"])
	}

	// 接下来发起带 file 的请求，测试单文件正则切片懒加载提取
	reqFile, _ := http.NewRequest("GET", "/api/tasks/501/diff?file=src/App.vue", nil)
	reqFile.Header.Set("Authorization", "Bearer "+adminToken)
	wFile := httptest.NewRecorder()
	r.ServeHTTP(wFile, reqFile)

	if wFile.Code != http.StatusOK {
		t.Fatalf("expected 200 for single file diff, got %d", wFile.Code)
	}

	var resFile map[string]interface{}
	_ = json.Unmarshal(wFile.Body.Bytes(), &resFile)
	expectedSubDiff := "diff --git a/src/App.vue b/src/App.vue\n..."
	if resFile["diff"] != expectedSubDiff {
		t.Errorf("expected single file diff match, got '%v'", resFile["diff"])
	}
}

// TestAPI_DualDiff_PersistenceAndFallback 验证全量部署在获取 Live Diff 时的友好降级提示
func TestAPI_DualDiff_PersistenceAndFallback(t *testing.T) {
	tmpLogDir, err := os.MkdirTemp("", "godeployer_dualdiff_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpLogDir)

	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_dualdiff_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}
	defer db.Close()

	_, _ = db.Exec(`DELETE FROM deploy_tasks`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (600, "test-app", "testing", "20260530110000", "aaaaaa111111", 1, "admin", "success", "{}", "2026-05-30T17:00:00Z", "branch")
	`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (601, "test-app", "testing", "20260530120000", "abcdef123456", 1, "admin", "success", "{}", "2026-05-30T18:00:00Z", "branch")
	`)

	// 模拟全量部署的快照：没有 live diff ("diff": ""), 只有 git_log_diff
	diffDir := filepath.Join(tmpLogDir, "diffs", "projects", "test-app", "202605")
	_ = os.MkdirAll(diffDir, 0755)
	jsonPath := filepath.Join(diffDir, "task_601_diff.log")

	mockChanges := map[string]string{
		"files":        "M src/App.vue",
		"diff":         "", // 线上对比为空
		"git_log_diff": "diff --git a/src/App.vue b/src/App.vue\n- old line\n+ new line",
	}
	jsonData, _ := json.Marshal(mockChanges)
	_ = os.WriteFile(jsonPath, jsonData, 0644)

	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:      "test-secret-key-12345",
			LogPath:        tmpLogDir,
			SSHKeyPath:     "./test_keys/id_rsa",
			DiffMaxSizeKB:  2048,
			DiskMinSpaceMB: 1,
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {ID: "test-app", Name: "Mock Test App"},
		},
	}

	engine := application.NewDeployEngine(taskRepo, nil)
	r := SetupRoutes(mockConfig, db, taskRepo, engine)

	adminToken, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	// 1. 请求 diff_type = live 时的单文件 diff，应该返回降级友好提示
	reqLive, _ := http.NewRequest("GET", "/api/tasks/601/diff?file=src/App.vue&diff_type=live", nil)
	reqLive.Header.Set("Authorization", "Bearer "+adminToken)
	wLive := httptest.NewRecorder()
	r.ServeHTTP(wLive, reqLive)

	if wLive.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wLive.Code)
	}
	if !bytes.Contains(wLive.Body.Bytes(), []byte("全量部署任务，未归档与线上对比快照")) {
		t.Errorf("expected fallback hint in live diff, got: %s", wLive.Body.String())
	}

	// 2. 请求 diff_type = git_log 时的单文件 diff，应该能正常解析并返回对应的变更
	reqLog, _ := http.NewRequest("GET", "/api/tasks/601/diff?file=src/App.vue&diff_type=git_log", nil)
	reqLog.Header.Set("Authorization", "Bearer "+adminToken)
	wLog := httptest.NewRecorder()
	r.ServeHTTP(wLog, reqLog)

	if !bytes.Contains(wLog.Body.Bytes(), []byte("new line")) {
		t.Errorf("expected actual git log diff, got: %s", wLog.Body.String())
	}
}
