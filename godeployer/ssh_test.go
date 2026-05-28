package godeployer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// setupMockSSHServer 启动一个极其简单的内存 SSH 服务用于压测，并返回其运行地址、私钥路径及关闭函数。
func setupMockSSHServer(t testing.TB) (string, string, func()) {
	// 生成服务端及客户端测试密钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}
	
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "id_rsa_test")
	err = os.WriteFile(keyPath, pem.EncodeToMemory(&privBlock), 0600)
	if err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// 允许所有公钥登录
			return nil, nil
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on TCP: %v", err)
	}

	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(netConn net.Conn) {
				_, chans, reqs, err := ssh.NewServerConn(netConn, config)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)

				for newChannel := range chans {
					if newChannel.ChannelType() != "session" {
						newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
						continue
					}
					channel, requests, err := newChannel.Accept()
					if err != nil {
						continue
					}

					go func(in <-chan *ssh.Request) {
						for req := range in {
							if req.Type == "exec" {
								cmdStr := string(req.Payload)
								if strings.Contains(cmdStr, "sleep") {
									time.Sleep(50 * time.Millisecond)
								}
								req.Reply(true, nil)
								channel.Write([]byte("mock_output\n"))
								channel.SendRequest("exit-status", false, ssh.Marshal(struct{ uint32 }{0}))
								channel.Close()
							} else {
								req.Reply(false, nil)
							}
						}
					}(requests)
				}
			}(nConn)
		}
	}()

	return listener.Addr().String(), keyPath, func() {
		listener.Close()
	}
}

// BenchmarkSSHExecutor_RunCommand 验证 SSH 连接的重复执行开销
func BenchmarkSSHExecutor_RunCommand(b *testing.B) {
	addr, keyPath, cleanup := setupMockSSHServer(b)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	serverCfg := ServerConfig{
		Host:       host,
		Port:       port,
		User:       "testuser",
		SSHKeyPath: keyPath,
	}

	pool := NewSSHPool(serverCfg, 1)
	defer pool.Close()
	executor := NewSSHExecutor(serverCfg, pool)
	executor.Ctx = context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.RunCommand("echo test")
		if err != nil {
			b.Fatalf("RunCommand failed: %v", err)
		}
	}
}

// TestSSHPool_AcquireRelease 验证连接池的获取、释放和超时机制
func TestSSHPool_AcquireRelease(t *testing.T) {
	addr, keyPath, cleanup := setupMockSSHServer(t)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	serverCfg := ServerConfig{
		Host:       host,
		Port:       port,
		User:       "testuser",
		SSHKeyPath: keyPath,
	}

	pool := NewSSHPool(serverCfg, 2)
	defer pool.Close()

	ctx := context.Background()

	// 1. 获取两个连接
	client1, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get client1: %v", err)
	}
	client2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get client2: %v", err)
	}

	// 2. 第三次获取应该阻塞直到超时
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, err = pool.Get(ctxTimeout)
	if err == nil {
		t.Fatalf("Expected timeout error when pool is exhausted, got nil")
	}

	// 3. 释放一个连接后，应该能获取新连接
	pool.Put(client1, nil)
	client3, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get client3 after release: %v", err)
	}
	
	pool.Put(client2, nil)
	pool.Put(client3, nil)
}

// BenchmarkSSHPool_Concurrent 压测多 goroutine 争抢连接
func BenchmarkSSHPool_Concurrent(b *testing.B) {
	addr, keyPath, cleanup := setupMockSSHServer(b)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	serverCfg := ServerConfig{
		Host:       host,
		Port:       port,
		User:       "testuser",
		SSHKeyPath: keyPath,
	}

	// 限制为 10 个并发连接
	pool := NewSSHPool(serverCfg, 10)
	defer pool.Close()

	executor := &SSHExecutor{
		Server: serverCfg,
		Ctx:    context.Background(),
		pool:   pool,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := executor.RunCommand("echo test")
			if err != nil {
				b.Errorf("RunCommand in parallel failed: %v", err)
			}
		}
	})
}

// TestSSHExecutor_Timeout 验证 SSH 执行器的超时保护
func TestSSHExecutor_Timeout(t *testing.T) {
	addr, keyPath, cleanup := setupMockSSHServer(t)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	serverCfg := ServerConfig{
		Host:       host,
		Port:       port,
		User:       "testuser",
		SSHKeyPath: keyPath,
	}

	pool := NewSSHPool(serverCfg, 1)
	defer pool.Close()

	executor := NewSSHExecutor(serverCfg, pool)
	
	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	executor.Ctx = ctx

	_, err := executor.RunCommand("sleep 5")
	if err == nil {
		t.Errorf("Expected timeout error, got nil")
	} else if err.Error() != "context deadline exceeded" && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context deadline exceeded or timeout error, got: %v", err)
	}
}
