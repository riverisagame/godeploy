package domain

import "testing"

// TestNewServerWithAuth 验证 Server 能携带 SSH 认证信息
// RED 阶段：NewServer 签名还没有 user 参数，应该编译失败或字段不存在
func TestNewServerWithAuth(t *testing.T) {
	srv, err := NewServer("web-01", "192.168.1.100", 22, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// 验证默认用户名是 root
	if srv.User != "root" {
		t.Errorf("expected default User 'root', got '%s'", srv.User)
	}

	// 验证 KeyPath 默认为空
	if srv.KeyPath != "" {
		t.Errorf("expected empty KeyPath, got '%s'", srv.KeyPath)
	}
}

// TestServerCustomAuth 验证可以设定自定义 SSH 用户和密钥
func TestServerCustomAuth(t *testing.T) {
	srv, err := NewServer("db-01", "10.0.0.50", 2222, "deployer", "~/.ssh/custom_rsa")
	if err != nil {
		t.Fatal(err)
	}

	if srv.User != "deployer" {
		t.Errorf("expected User 'deployer', got '%s'", srv.User)
	}

	if srv.KeyPath != "~/.ssh/custom_rsa" {
		t.Errorf("expected KeyPath '~/.ssh/custom_rsa', got '%s'", srv.KeyPath)
	}
}
