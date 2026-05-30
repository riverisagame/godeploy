package godeployer_test

import (
	"os"
	"path/filepath"
	"testing"

	"deploy/godeployer"
)

// TestMain_AppBootstrapVerify 验证主程序的生命周期装载。
// 检查配置文件载入、数据库迁移、以及嵌入式静态资源的读取连贯性。
func TestMain_AppBootstrapVerify(t *testing.T) {
	// 创建临时主配置 yaml
	tmpDir, err := os.MkdirTemp("", "godeployer-main-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configYAML := `
global:
  sqlite_path: "file::memory:?cache=shared"
  log_path: "` + filepath.ToSlash(filepath.Join(tmpDir, "logs")) + `"
  workspace_path: "` + filepath.ToSlash(filepath.Join(tmpDir, "workspace")) + `"
  server_port: 9999
  jwt_secret: "bootstrap-secret"
project_config_dir: ""
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// 启动应用引导逻辑 (在 RED 阶段尚未实现，预计编译或调用失败)
	config, db, _, err := godeployer.BootstrapApp(configPath)
	if err != nil {
		t.Fatalf("BootstrapApp failed: %v", err)
	}
	defer db.Close()

	if config.Global.ServerPort != 9999 {
		t.Errorf("expected server port to be 9999, got %d", config.Global.ServerPort)
	}

	// 验证内嵌静态资源是否正常工作
	htmlContent, err := godeployer.GetEmbeddedAsset("dist/index.html")
	if err != nil {
		t.Fatalf("failed to read embedded index.html: %v", err)
	}

	if len(htmlContent) == 0 {
		t.Error("embedded index.html content is empty")
	}
}
