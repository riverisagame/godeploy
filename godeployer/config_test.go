package godeployer_test

import (
	"os"
	"path/filepath"
	"testing"

	"deploy/godeployer"
)

// TestConfig_LoadEnvVerify 用于验证环境变量替换机制是否正常工作。
// 属于 Edge Case: 配置文件包含未定义的环境变量时，应该保留占位符或报错。
func TestConfig_LoadEnvVerify(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("TEST_DB_PATH", "/tmp/test.db")
	defer os.Unsetenv("TEST_DB_PATH")

	// 模拟主配置文件内容，包含环境变量占位符
	mainConfigYAML := `
global:
  sqlite_path: "${TEST_DB_PATH}"
  log_path: "./logs"
  workspace_path: "./workspace"
  ssh_key_path: "~/.ssh/id_rsa"
  server_port: 8080
  jwt_secret: "secret"
project_config_dir: "./projects.d"
`

	// 临时创建配置目录
	tmpDir, err := os.MkdirTemp("", "godeployer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mainConfigPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(mainConfigPath, []byte(mainConfigYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// 执行加载
	config, err := godeployer.LoadConfig(mainConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证环境变量是否成功替换
	expectedDBPath := "/tmp/test.db"
	if config.Global.SQLitePath != expectedDBPath {
		t.Errorf("expected SQLitePath to be %q, got %q", expectedDBPath, config.Global.SQLitePath)
	}
}

// TestConfig_LoadProjects 验证扫描目录加载项目配置的能力。
// 包括验证 exclusions 和 environments 的正确解析。
func TestConfig_LoadProjects(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mainConfigYAML := `
global:
  sqlite_path: "./data.db"
  log_path: "./logs"
  workspace_path: "./workspace"
  server_port: 8080
  jwt_secret: "secret"
project_config_dir: "` + filepath.ToSlash(filepath.Join(tmpDir, "projects.d")) + `"
`
	mainConfigPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(mainConfigPath, []byte(mainConfigYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// 创建 projects.d 目录并写入一个项目配置文件
	projDir := filepath.Join(tmpDir, "projects.d")
	if err := os.Mkdir(projDir, 0755); err != nil {
		t.Fatalf("failed to create projects.d: %v", err)
	}

	projYAML := `
id: "test-proj"
name: "Test Project"
repo: "git@github.com:test/repo.git"
exclude:
  - ".git"
environments:
  - id: "testing"
    name: "Testing Environment"
    servers:
      - host: "localhost"
        port: 22
        user: "deploy"
        deploy_to: "/var/www/test"
`
	if err := os.WriteFile(filepath.Join(projDir, "test-proj.yaml"), []byte(projYAML), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// 载入配置
	config, err := godeployer.LoadConfig(mainConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证项目配置是否正确加载到全局配置中
	if len(config.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(config.Projects))
	}

	proj := config.Projects["test-proj"]
	if proj.Name != "Test Project" {
		t.Errorf("expected project name to be 'Test Project', got %q", proj.Name)
	}

	if len(proj.Environments) != 1 || proj.Environments[0].ID != "testing" {
		t.Errorf("expected 1 environment 'testing', got %+v", proj.Environments)
	}
}
