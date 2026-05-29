package godeployer_test

import (
	"bytes"
	"database/sql"
	"deploy/godeployer"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupEnhanceTestRouter 初始化 Gin 测试路由，并将 repo 设为本地当前路径
func SetupEnhanceTestRouter(t *testing.T) (*gin.Engine, *sql.DB, func()) {
	gin.SetMode(gin.TestMode)

	db, err := godeployer.InitDB(fmt.Sprintf("file:mem_enhance_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}

	wd, err := os.Getwd()
	if err == nil {
		if filepath.Base(wd) == "godeployer" {
			wd = filepath.Dir(wd)
		}
	} else {
		wd = "."
	}

	mockConfig := &godeployer.Config{
		Global: godeployer.GlobalConfig{
			JWTSecret:  "test-secret-key-12345",
			LogPath:    "./test_logs",
			SSHKeyPath: "./test_keys/id_rsa",
		},
		Projects: map[string]godeployer.ProjectConfig{
			"test-app": {
				ID:            "test-app",
				Name:          "Mock Test App",
				Repo:          wd, // 设为本地仓库路径，确保 git clone 本地能够成功
				WebhookSecret: "secret123",
				Branch:        "main",
				Environments: []godeployer.EnvironmentConfig{
					{
						ID:   "testing",
						Name: "Test Env",
						Servers: []godeployer.ServerConfig{
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

	engine := godeployer.NewDeployEngine(db, nil)
	engine.StartDispatcher(1)
	r := godeployer.SetupRoutes(mockConfig, db, engine)

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
	
	adminToken, _ := godeployer.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
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
	adminToken, _ := godeployer.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
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
	db, err := godeployer.InitDB(fmt.Sprintf("file:mem_cache_%d?mode=memory&cache=shared", time.Now().UnixNano()))
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

	mockConfig := &godeployer.Config{
		Global: godeployer.GlobalConfig{
			JWTSecret:      "test-secret-key-12345",
			LogPath:        tmpLogDir,
			SSHKeyPath:     "./test_keys/id_rsa",
			DiffMaxSizeKB:  2048,
			DiskMinSpaceMB: 1,
		},
		Projects: map[string]godeployer.ProjectConfig{
			"test-app": {ID: "test-app", Name: "Mock Test App"},
		},
	}

	engine := godeployer.NewDeployEngine(db, nil)
	r := godeployer.SetupRoutes(mockConfig, db, engine)

	req, _ := http.NewRequest("GET", "/api/tasks/501/diff", nil)
	adminToken, _ := godeployer.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	// 断言：必须解析出了 files 和 diff 两部分
	if res["files"] != "M src/App.vue\nA src/components/TreeView.vue" {
		t.Errorf("expected files list match, got '%v'", res["files"])
	}
	if res["diff"] != "diff --git a/src/App.vue b/src/App.vue\n..." {
		t.Errorf("expected diff text match, got '%v'", res["diff"])
	}
}
