package api

import (
	"bytes"
	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/db"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProjectPermissions(t *testing.T) {
	// 准备测试环境
	db, taskRepo, err := db.InitTestDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer db.Close()

	// 清空 users 表
	_, _ = db.Exec("DELETE FROM users")

	// 插入测试用户
	now := time.Now()
	// user1 拥有所有项目权限
	_, err = db.Exec("INSERT INTO users (username, password_hash, role, permitted_projects, created_at) VALUES (?, ?, ?, ?, ?)", "user1", "hash", "deployer", "*", now)
	if err != nil {
		t.Fatalf("failed to insert user1: %v", err)
	}
	// user2 只有部分项目权限
	_, err = db.Exec("INSERT INTO users (username, password_hash, role, permitted_projects, created_at) VALUES (?, ?, ?, ?, ?)", "user2", "hash", "deployer", "proj_a", now)
	if err != nil {
		t.Fatalf("failed to insert user2: %v", err)
	}
	// admin
	_, err = db.Exec("INSERT INTO users (username, password_hash, role, permitted_projects, created_at) VALUES (?, ?, ?, ?, ?)", "admin", "hash", "admin", "*", now)
	if err != nil {
		t.Fatalf("failed to insert admin: %v", err)
	}

	// 准备引擎与路由
	config := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret: "secret",
		},
		Projects: map[string]domain.ProjectConfig{
			"proj_a": {ID: "proj_a", Name: "Project A"},
			"proj_b": {ID: "proj_b", Name: "Project B"},
		},
	}
	engine := application.NewDeployEngine(taskRepo, nil)
	r := SetupRoutes(config, db, taskRepo, engine)

	t.Run("user1 sees all projects", func(t *testing.T) {
		token, _ := application.GenerateToken("user1", "deployer", "secret", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/api/projects", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp []domain.ProjectConfig
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Errorf("expected 2 projects for user1, got %d", len(resp))
		}
	})

	t.Run("user2 sees restricted projects", func(t *testing.T) {
		token, _ := application.GenerateToken("user2", "deployer", "secret", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/api/projects", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp []domain.ProjectConfig
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Errorf("expected 1 project for user2, got %d", len(resp))
		} else if resp[0].ID != "proj_a" {
			t.Errorf("expected proj_a, got %s", resp[0].ID)
		}
	})

	t.Run("admin updates user2 permissions", func(t *testing.T) {
		adminToken, _ := application.GenerateToken("admin", "admin", "secret", time.Hour)
		reqBody := `{"permitted_projects": "proj_a,proj_b"}`
		req, _ := http.NewRequest(http.MethodPut, "/api/users/user2/permissions", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
