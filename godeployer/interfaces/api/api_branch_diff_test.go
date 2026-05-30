package api

import (
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/sqlite"
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

// TestAPI_BranchDeployDiff_Fallback 验证分支部署在历史记录中获取单文件 diff 时的避让和极速退化逻辑
func TestAPI_BranchDeployDiff_Fallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer_branch_diff_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := sqlite.InitDB(fmt.Sprintf("file:mem_branchdiff_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}
	defer db.Close()

	_, _ = db.Exec(`DELETE FROM deploy_tasks`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (701, "test-branch-app", "prod", "20260530130000", "master", 1, "admin", "success", "{}", "2026-05-30T13:00:00Z", "branch")
	`)

	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (700, "test-branch-app", "prod", "20260530120000", "old-branch-commit", 1, "admin", "success", "{}", "2026-05-30T12:00:00Z", "branch")
	`)

	diffCacheDir := filepath.Join(tmpDir, "diffs", "projects", "test-branch-app", "202605")
	_ = os.MkdirAll(diffCacheDir, 0755)
	diffCacheFile := filepath.Join(diffCacheDir, "task_701_diff.log")

	cacheMap := map[string]string{
		"files":        "M main.go",
		"diff":         "",
		"git_log_diff": "diff --git a/main.go b/main.go\n- old\n+ new",
	}
	cacheBytes, _ := json.Marshal(cacheMap)
	_ = os.WriteFile(diffCacheFile, cacheBytes, 0644)

	cfg := &domain.Config{
		Projects: map[string]domain.ProjectConfig{
			"test-branch-app": {
				ID:   "test-branch-app",
				Repo: "https://github.com/mock/repo.git",
			},
		},
		Global: domain.GlobalConfig{
			LogPath:        tmpDir,
			WorkspacePath:  filepath.Join(tmpDir, "workspace"),
			DiffMaxSizeKB:  512,
			DiskMinSpaceMB: 0,
		},
	}

	handler := &APIHandler{
		db:     db,
		config: cfg,
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("username", "admin")
		c.Next()
	})
	router.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)

	req, _ := http.NewRequest("GET", "/tasks/701/diff?file=main.go&diff_type=git_log", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var res gin.H
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	diffVal, ok := res["diff"].(string)
	if !ok || diffVal == "" {
		t.Errorf("expected non-empty diff text, got %v", res["diff"])
	}
}

// TestAPI_BranchDeployDiff_NoCache_Fallback 验证在没有持久化缓存文件且 build 目录不存在时，
// 接口在定位 Git 仓库时不会进入耗时极长的 git.FindGitRepo 磁盘 Walk，而是能被快速安全拦截并优雅降级。
func TestAPI_BranchDeployDiff_NoCache_Fallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer_nocache_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := sqlite.InitDB(fmt.Sprintf("file:mem_nocache_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}
	defer db.Close()

	_, _ = db.Exec(`DELETE FROM deploy_tasks`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (800, "test-branch-app", "prod", "20260530130000", "old-commit", 1, "admin", "success", "{}", "2026-05-30T13:00:00Z", "branch")
	`)
	_, _ = db.Exec(`
		INSERT INTO deploy_tasks (id, project_id, env_id, release_name, commit_id, user_id, username, status, config_snapshot, created_at, target_type)
		VALUES (801, "test-branch-app", "prod", "20260530140000", "master", 1, "admin", "success", "{}", "2026-05-30T14:00:00Z", "branch")
	`)

	cfg := &domain.Config{
		Projects: map[string]domain.ProjectConfig{
			"test-branch-app": {
				ID:   "test-branch-app",
				Repo: "https://github.com/mock/repo.git",
			},
		},
		Global: domain.GlobalConfig{
			LogPath:        tmpDir,
			WorkspacePath:  filepath.Join(tmpDir, "workspace"),
			DiffMaxSizeKB:  512,
			DiskMinSpaceMB: 0,
		},
	}

	handler := &APIHandler{
		db:     db,
		config: cfg,
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("username", "admin")
		c.Next()
	})
	router.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)

	// 在 workspace 中创建很多无关的目录以测试如果走 git.FindGitRepo 会产生 Walk。
	// 这里通过验证响应时间来证明是否被快速安全拦截，或者验证其是否未执行缓慢的扫描进程。
	// 在未修复时，由于没有 cache 且 buildPath 不存在，且 commit_id 是 master（非Hash），
	// 系统会去走 git.FindGitRepo，对 Workspace 进行 Walk 并查找是否有包含 master 的 git 仓库，这在空目录下可能不会卡死，但我们可以测出结果为“无法获取”。
	req, _ := http.NewRequest("GET", "/tasks/801/diff?file=main.go&diff_type=git_log", nil)

	startTime := time.Now()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	duration := time.Since(startTime)

	t.Logf("Response code: %d, time: %v, body: %s", w.Code, duration, w.Body.String())

	// 此时应该返回 200，但 diff 可能是 fallback 的提示或空白（因为无 git 仓库），关键是不能长时间挂起超时。
	// 我们希望它的响应是毫秒级的。
	if duration > 200*time.Millisecond {
		t.Errorf("expected response to be quick (under 200ms), but took %v", duration)
	}
}
