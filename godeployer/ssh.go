package godeployer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type RemoteExecutor interface {
	RunCommand(cmd string) (string, error)
	Rsync(local, remote string, linkDest string) error
	Close() error
}

type SSHExecutor struct {
	Server ServerConfig
	Ctx    context.Context // 增加可选的 Context 字段以实现超时控制对冲

	client *ssh.Client
	mu     sync.Mutex
}

func NewSSHExecutor(server ServerConfig) *SSHExecutor {
	return &SSHExecutor{Server: server}
}

func (s *SSHExecutor) getClient() (*ssh.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		// 简单起见，这里假设连接是活的。生产环境可以引入 keepalive 检查
		return s.client, nil
	}

	// 读取私钥文件
	key, err := os.ReadFile(s.Server.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: s.Server.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", s.Server.Host, s.Server.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect SSH server: %w", err)
	}

	s.client = client
	return s.client, nil
}

func (s *SSHExecutor) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		return err
	}
	return nil
}

func (s *SSHExecutor) RunCommand(cmd string) (string, error) {
	client, err := s.getClient()
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if s.Ctx != nil {
		doneChan := make(chan struct{})
		defer close(doneChan)
		go func() {
			select {
			case <-s.Ctx.Done():
				session.Close()
			case <-doneChan:
			}
		}()
	}

	stdout, err := session.Output(cmd)
	if err != nil {
		if s.Ctx != nil && s.Ctx.Err() != nil {
			return string(stdout), fmt.Errorf("SSH command context canceled: %w", s.Ctx.Err())
		}
		return string(stdout), fmt.Errorf("failed to run command %q: %w", cmd, err)
	}

	return string(stdout), nil
}

func (s *SSHExecutor) Rsync(local, remote string, linkDest string) error {
	// 拼接本地 rsync 命令。在 Linux/WSL/MacOS 环境下可用，在 Windows 平台需要安装并配置 rsync。
	// --link-dest 需要传入目标机相对于 releases/new_release 目录的相对路径，或者绝对路径。
	sshCmd := fmt.Sprintf("ssh -p %d -i %s", s.Server.Port, s.Server.SSHKeyPath)
	
	args := []string{
		"-rlptz",
		"--no-owner",
		"--no-group",
		"--delete",
		"-e", sshCmd,
	}

	if linkDest != "" {
		args = append(args, fmt.Sprintf("--link-dest=%s", linkDest))
	}

	// 拼接本地目录与目标服务器远程目录。例如: /local/path/ deploy@host:/var/www/my-app/releases/xxx/
	remoteTarget := fmt.Sprintf("%s@%s:%s", s.Server.User, s.Server.Host, remote)
	args = append(args, local, remoteTarget)

	ctx := s.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "rsync", args...)
	
	// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
	// 重定向 stderr 用于报错
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("rsync command failed: %s: %w", stderr.String(), err)
		}
		return fmt.Errorf("rsync command failed: %w", err)
	}

	return nil
}
