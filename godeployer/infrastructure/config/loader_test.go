package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"deploy/godeployer/infrastructure/config"
)

func TestConfig_LoadEnvVerify(t *testing.T) {
	os.Setenv("TEST_DB_PATH", "/tmp/test.db")
	defer os.Unsetenv("TEST_DB_PATH")

	mainYAML := `
global:
  sqlite_path: "$TEST_DB_PATH"
  log_path: /var/log
  workspace_path: /workspace
  server_port: 8080
  jwt_secret: test-secret
project_config_dir: ""
`

	tmpDir, err := os.MkdirTemp("", "godeployer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mainPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(mainPath, []byte(mainYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	got, err := config.LoadConfig(mainPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got.Global.SQLitePath != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %q", got.Global.SQLitePath)
	}
}

func TestConfig_LoadProjects(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projDir := filepath.Join(tmpDir, "projects")
	_ = os.MkdirAll(projDir, 0755)

	projectYAML := "id: test-proj\nname: Test Project\nrepo: git@github.com:test/repo.git\nbranch: main\nexclude:\n  - .git\n  - \"*.log\"\nenvironments:\n  - id: prod\n    name: Production\n    servers:\n      - host: 10.0.0.1\n        port: 22\n        user: deploy\n        deploy_to: /var/www/test-proj\n"
	if err := os.WriteFile(filepath.Join(projDir, "test-proj.yaml"), []byte(projectYAML), 0644); err != nil {
		t.Fatalf("failed to write project yaml: %v", err)
	}

	mainYAML := "global:\n  sqlite_path: \":memory:\"\n  log_path: /tmp/logs\n  workspace_path: /tmp/workspace\n  server_port: 8080\n  jwt_secret: test\nproject_config_dir: \"" + filepath.ToSlash(projDir) + "\"\n"
	mainPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(mainPath, []byte(mainYAML), 0644); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	got, err := config.LoadConfig(mainPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	proj, exists := got.Projects["test-proj"]
	if !exists {
		t.Fatal("expected test-proj to be loaded")
	}
	if proj.Name != "Test Project" {
		t.Errorf("expected 'Test Project', got %q", proj.Name)
	}
}
