package godeployer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHPool 提供并发安全的 SSH 连接池管理
type SSHPool struct {
	server   ServerConfig
	maxConns int
	idle     chan *ssh.Client
	mu       sync.Mutex
	active   int
}

// NewSSHPool 创建一个固定容量的 SSH 连接池
func NewSSHPool(server ServerConfig, maxConns int) *SSHPool {
	if maxConns <= 0 {
		maxConns = 10
	}
	return &SSHPool{
		server:   server,
		maxConns: maxConns,
		idle:     make(chan *ssh.Client, maxConns),
	}
}

func (p *SSHPool) createClient() (*ssh.Client, error) {
	path := p.server.SSHKeyPath
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: p.server.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", p.server.Host, p.server.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect SSH server: %w", err)
	}
	return client, nil
}

// Get 获取一个 SSH 客户端，支持 ctx 超时机制
func (p *SSHPool) Get(ctx context.Context) (*ssh.Client, error) {
	// 1. 尝试从空闲池非阻塞获取
	select {
	case client := <-p.idle:
		return client, nil
	default:
	}

	// 2. 检查是否可以新建连接
	p.mu.Lock()
	if p.active < p.maxConns {
		p.active++
		p.mu.Unlock()

		client, err := p.createClient()
		if err != nil {
			p.mu.Lock()
			p.active--
			p.mu.Unlock()
			return nil, err
		}
		return client, nil
	}
	p.mu.Unlock()

	// 3. 超过最大并发限制，阻塞等待其他请求归还或直到超时
	select {
	case client := <-p.idle:
		return client, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Put 归还连接或将其废弃
func (p *SSHPool) Put(client *ssh.Client, err error) {
	if err != nil {
		_ = client.Close()
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
		return
	}

	// 无报错，放入空闲池
	select {
	case p.idle <- client:
	default:
		// 理论上不会发生，因为 maxConns = cap(idle)
		_ = client.Close()
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
	}
}

// Close 销毁池中所有空闲连接
func (p *SSHPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for {
		select {
		case client := <-p.idle:
			if err := client.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		default:
			p.active = 0
			return firstErr
		}
	}
}
