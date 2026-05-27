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
	"testing"

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

	executor := NewSSHExecutor(serverCfg)
	executor.Ctx = context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.RunCommand("echo test")
		if err != nil {
			b.Fatalf("RunCommand failed: %v", err)
		}
	}
}
