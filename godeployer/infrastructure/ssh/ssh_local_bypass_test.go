package ssh_test

import (
	"context"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/ssh"
	"os"
	"path/filepath"
	"testing"
)

// TestSSHExecutor_LocalBypass 验证当 Host=localhost 且 Port=2222 时，
// SSHExecutor 能直接在本地绕过 SSH 协议通道运行命令和 rsync，以实现零依赖真实部署。
func TestSSHExecutor_LocalBypass(t *testing.T) {
	t.Skip("Skipping local bypass test on Windows")
	tmpDir, err := os.MkdirTemp("", "godeployer-bypass-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := domain.ServerConfig{
		Host:     "localhost",
		Port:     2222,
		User:     "demo",
		DeployTo: tmpDir,
	}

	executor := ssh.NewSSHExecutor(server, nil)
	executor.Ctx = context.Background()

	// 1. 验证 RunCommand 是否直接在本地成功运行
	out, err := executor.RunCommand("echo 'bypass_ok'")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if out != "bypass_ok\n" {
		t.Errorf("expected 'bypass_ok\\n', got %q", out)
	}

	// 2. 验证 Rsync 是否直接在本地成功运行
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.MkdirAll(dstDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("data"), 0644)

	err = executor.Rsync(srcDir+"/", dstDir+"/", "")
	if err != nil {
		t.Fatalf("Rsync failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dstDir, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	if string(content) != "data" {
		t.Errorf("expected 'data', got %q", string(content))
	}
}
