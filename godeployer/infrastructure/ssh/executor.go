package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"deploy/godeployer/domain"

	"golang.org/x/crypto/ssh"
)

type RemoteExecutor interface {
	RunCommand(cmd string) (string, error)
	Rsync(local, remote string, linkDest string) error
	Close() error
}

type SSHExecutor struct {
	Server      domain.ServerConfig
	Ctx         context.Context // 增加可选的 Context 字段以实现超时控制对冲
	ExcludeList []string        // 动态过滤排除列表

	pool *SSHPool
}

func NewSSHExecutor(server domain.ServerConfig, pool *SSHPool) *SSHExecutor {
	return &SSHExecutor{Server: server, pool: pool}
}

// getClient 和 Close 方法已废弃，其职责由 SSHPool 接管

func (s *SSHExecutor) Close() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

func (s *SSHExecutor) RunCommand(cmd string) (string, error) {
	// @Ref: docs/sps/plans/20260530_demo_script_optimization_plan.md | @Date: 2026-05-30
	// 针对 demo 场景的 2222 端口本地免 SSH 旁路处理
	if (s.Server.Host == "localhost" || s.Server.Host == "127.0.0.1") && s.Server.Port == 2222 {
		var execCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			execCmd = exec.Command("cmd", "/C", cmd)
		} else {
			execCmd = exec.Command("sh", "-c", cmd)
		}
		output, err := execCmd.CombinedOutput()
		return string(output), err
	}

	if s.pool == nil {
		return "", fmt.Errorf("SSH pool is not initialized")
	}

	client, err := s.pool.Get(s.Ctx)
	if err != nil {
		return "", err
	}
	// 执行完成后释放连接回池，如果有严重错误则传递 err，这里我们只捕获 session 错误
	var connErr error
	defer func() {
		s.pool.Put(client, connErr)
	}()

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
		// 根据 ssh 库，如果在连接断开时会报 EOF 或类似网络错误
		// 为保险起见，若是命令本身的错误 (ExitError)，不废弃连接
		if _, isExitErr := err.(*ssh.ExitError); !isExitErr {
			connErr = err
		}
		return string(stdout), fmt.Errorf("failed to run command %q: %w", cmd, err)
	}

	return string(stdout), nil
}

func (s *SSHExecutor) Rsync(local, remote string, linkDest string) error {
	// @Ref: docs/sps/plans/20260530_demo_script_optimization_plan.md | @Date: 2026-05-30
	// 针对 demo 场景的 2222 端口本地免 SSH 旁路处理
	if (s.Server.Host == "localhost" || s.Server.Host == "127.0.0.1") && s.Server.Port == 2222 {
		var args []string
		args = append(args, "-rlptz", "--delete")
		if linkDest != "" {
			args = append(args, fmt.Sprintf("--link-dest=%s", linkDest))
		}
		for _, pattern := range s.ExcludeList {
			trimmed := strings.TrimSpace(pattern)
			if trimmed != "" {
				args = append(args, fmt.Sprintf("--exclude=%s", trimmed))
			}
		}
		args = append(args, local, remote)

		ctx := s.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		cmd := exec.CommandContext(ctx, "rsync", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("local bypass rsync failed: %s: %w", stderr.String(), err)
			}
			return fmt.Errorf("local bypass rsync failed: %w", err)
		}
		return nil
	}

	// 拼接本地 rsync 命令。在 Linux/WSL/MacOS 环境下可用，在 Windows 平台需要安装并配置 rsync。
	// --link-dest 需要传入目标机相对于 releases/new_release 目录的相对路径，或者绝对路径。
	sshCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d -i %s", s.Server.Port, s.Server.SSHKeyPath)

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

	for _, pattern := range s.ExcludeList {
		trimmed := strings.TrimSpace(pattern)
		if trimmed != "" {
			args = append(args, fmt.Sprintf("--exclude=%s", trimmed))
		}
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
