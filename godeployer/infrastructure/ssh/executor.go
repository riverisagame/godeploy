// ============================================================
// 文件：executor.go
// 作用：🖥️ SSH 执行器 + 本地绕过——两种方式操作远程服务器！
//
// 这个文件实现了两种"执行命令"的方式：
//
// 1. 本地绕过（Local Bypass）
//    当目标服务器是 localhost:2222（Demo 场景）时，
//    直接在本机执行命令，不通过 SSH。
//    好处：Demo 不需要装 SSH 服务器！💪
//
// 2. SSH 远程执行（生产场景）
//    通过 SSH 连接池执行命令和 rsync 同步文件。
//
// 这个 SSHExecutor 实现了 NodeExecutor 接口中定义的：
// - RunCommand：执行命令
// - Rsync：同步文件
// - Close：关闭连接
//
// 给初二小白的比喻：
// 这就像一个"遥控器"📡——
// - 普通模式：通过 WiFi 遥控（SSH）
// - Demo 模式：直接用手按（本地绕过）
// ============================================================

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

// RemoteExecutor 定义远程执行器接口（已被 NodeExecutor 替代，保留兼容）
type RemoteExecutor interface {
	RunCommand(cmd string) (string, error)
	Rsync(local, remote string, linkDest string) error
	Close() error
}

// SSHExecutor 封装了对单台服务器的操作
// 它直接使用 ssh 包或者本地绕过方式执行命令
type SSHExecutor struct {
	Server      domain.ServerConfig // 🖥️ 目标服务器配置
	Ctx         context.Context     // 📡 上下文（用于超时控制）
	ExcludeList []string            // 🚫 排除列表（要过滤掉的文件模式）

	pool *SSHPool // 🔌 SSH 连接池
}

// NewSSHExecutor 创建一个新的 SSH 执行器
func NewSSHExecutor(server domain.ServerConfig, pool *SSHPool) *SSHExecutor {
	return &SSHExecutor{Server: server, pool: pool}
}

// Close 关闭 SSH 连接
func (s *SSHExecutor) Close() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

// RunCommand 在目标服务器上执行一条命令
//
// 如果是 localhost:2222（Demo 模式），直接本地执行。
// 否则通过 SSH 连接池远程执行。
func (s *SSHExecutor) RunCommand(cmd string) (string, error) {
	// @Ref: docs/sps/plans/20260530_demo_script_optimization_plan.md | @Date: 2026-05-30
	// 🏠 针对 Demo 场景的 localhost:2222 本地免 SSH 旁路处理
	if (s.Server.Host == "localhost" || s.Server.Host == "127.0.0.1") && s.Server.Port == 2222 {
		// 直接在本机执行命令
		var execCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			execCmd = exec.Command("cmd", "/C", cmd)
		} else {
			execCmd = exec.Command("sh", "-c", cmd)
		}
		output, err := execCmd.CombinedOutput()
		return string(output), err
	}

	// 🌐 生产环境：通过 SSH 连接池执行
	if s.pool == nil {
		return "", fmt.Errorf("SSH pool is not initialized")
	}

	client, err := s.pool.Get(s.Ctx)
	if err != nil {
		return "", err
	}

	// 执行完成后把连接放回池子
	var connErr error
	defer func() {
		s.pool.Put(client, connErr)
	}()

	// 创建 SSH 会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// 如果上下文有超时/取消，监听 ctx.Done() 主动关闭会话
	if s.Ctx != nil {
		doneChan := make(chan struct{})
		defer close(doneChan)
		go func() {
			select {
			case <-s.Ctx.Done():
				session.Close() // 🔴 超时了，关闭会话
			case <-doneChan:
			}
		}()
	}

	// 执行命令并获取输出
	stdout, err := session.Output(cmd)
	if err != nil {
		if s.Ctx != nil && s.Ctx.Err() != nil {
			return string(stdout), fmt.Errorf("SSH command context canceled: %w", s.Ctx.Err())
		}
		// 如果是命令本身的错误（比如命令返回非零），不废弃连接
		if _, isExitErr := err.(*ssh.ExitError); !isExitErr {
			connErr = err // 连接层面的错误，告诉池子这个连接可能坏了
		}
		return string(stdout), fmt.Errorf("failed to run command %q: %w", cmd, err)
	}

	return string(stdout), nil
}

// Rsync 通过 rsync 将本地目录同步到远程服务器
func (s *SSHExecutor) Rsync(local, remote string, linkDest string) error {
	// @Ref: docs/sps/plans/20260530_demo_script_optimization_plan.md | @Date: 2026-05-30
	// 🏠 Demo 模式：直接本地执行 rsync
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

	// 🌐 生产环境：通过 SSH + rsync 同步
	sshCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d -i %s",
		s.Server.Port, s.Server.SSHKeyPath)

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

	remoteTarget := fmt.Sprintf("%s@%s:%s", s.Server.User, s.Server.Host, remote)
	args = append(args, local, remoteTarget)

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
			return fmt.Errorf("rsync command failed: %s: %w", stderr.String(), err)
		}
		return fmt.Errorf("rsync command failed: %w", err)
	}

	return nil
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是"本地绕过"？
//    A: Demo 模式下不通过 SSH，直接在本机执行命令。
//       就像练车时不用真的上路，在操场练就行~
//
// 2. Q: rsync 是什么？
//    A: Linux 上强大的文件同步工具！
//       只会传输变更的部分，支持压缩、排除、断点续传~
//
// 中级：
// 3. Q: 为什么 localhost:2222 要特殊处理？
//    A: Demo 环境用 Docker 模拟远程服务器，需要本地执行。
//       这个特殊处理让 Demo 不需要真装 SSH 服务器就能跑起来~
//
// 4. Q: defer s.pool.Put(client, connErr) 中的 connErr 有什么用？
//    A: 如果连接出错了（connErr != nil），Put 会关闭这个连接
//       而不是放回池子。防止"坏连接"被重复使用~
//
// 高级：
// 5. Q: 为什么不直接用 adapter.go 而保留了 executor.go？
//    A: executor.go 是旧的实现，adapter.go 是 DDD 重构后的实现。
//       两者实现了不同的接口，executor 暂时保留用于兼容~
// ============================================================
