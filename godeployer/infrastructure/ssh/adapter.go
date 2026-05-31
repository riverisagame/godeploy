// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31
package ssh

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"deploy/godeployer/domain"

	gossh "golang.org/x/crypto/ssh"
)

// NodeAdapter 包装 SSH 连接池映射，将底层 SSH 能力适配为 domain.NodeExecutor 接口。
type NodeAdapter struct {
	mu    sync.Mutex
	pools map[string]*SSHPool
}

// NewNodeAdapter 创建空的 NodeAdapter 实例，池按需创建。
func NewNodeAdapter() *NodeAdapter {
	return &NodeAdapter{pools: make(map[string]*SSHPool)}
}

func (a *NodeAdapter) getPool(node domain.ServerConfig) *SSHPool {
	key := fmt.Sprintf("%s:%d", node.Host, node.Port)
	a.mu.Lock()
	defer a.mu.Unlock()
	if pool, exists := a.pools[key]; exists {
		return pool
	}
	pool := NewSSHPool(node, 10)
	a.pools[key] = pool
	return pool
}

// RunCommand 在目标节点上执行任意命令，返回标准输出。
func (a *NodeAdapter) RunCommand(ctx context.Context, node domain.ServerConfig, cmd string) (string, error) {
	pool := a.getPool(node)
	client, err := pool.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("runcommand: %w", err)
	}
	defer pool.Put(client, err)

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("runcommand: new session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("runcommand: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// SwitchSymlink 在目标节点上执行原子软链接切换。
func (a *NodeAdapter) SwitchSymlink(ctx context.Context, node domain.ServerConfig, releaseName string) error {
	releasesDir := filepath.ToSlash(filepath.Join(node.DeployTo, "releases"))
	targetPath := filepath.ToSlash(filepath.Join(releasesDir, releaseName))
	currentPath := filepath.ToSlash(filepath.Join(node.DeployTo, "current"))

	cmd := fmt.Sprintf(
		"temp_link=\"%s.%d\" && ln -sfn \"%s\" \"$temp_link\" && mv -fT \"$temp_link\" \"%s\"",
		currentPath, 0, targetPath, currentPath,
	)

	_, err := a.RunCommand(ctx, node, cmd)
	return err
}

// Rsync 将构建产物通过 rsync 同步到目标节点，支持排除列表和硬链接参考。
func (a *NodeAdapter) Rsync(ctx context.Context, node domain.ServerConfig, localPath, remotePath, linkDest string, excludes []string) error {
	pool := a.getPool(node)
	client, err := pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("rsync: %w", err)
	}
	defer pool.Put(client, err)

	// 构造 rsync 命令
	excludeArgs := ""
	for _, p := range excludes {
		excludeArgs += fmt.Sprintf(" --exclude='%s'", p)
	}
	linkDestArg := ""
	if linkDest != "" {
		linkDestArg = fmt.Sprintf(" --link-dest='%s'", linkDest)
	}

	remoteHost := fmt.Sprintf("%s@%s", node.User, node.Host)
	rsyncCmd := fmt.Sprintf(
		"rsync -avz --delete%s%s -e 'ssh -p %d -i %s' %s %s:%s",
		excludeArgs, linkDestArg, node.Port, node.SSHKeyPath, localPath, remoteHost, remotePath,
	)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("rsync: new session: %w", err)
	}
	defer session.Close()

	return session.Run(rsyncCmd)
}

// Close 释放所有连接池资源。
func (a *NodeAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var lastErr error
	for key, pool := range a.pools {
		if err := pool.Close(); err != nil {
			lastErr = fmt.Errorf("close pool %s: %w", key, err)
		}
	}
	return lastErr
}

// compile-time check
var _ domain.NodeExecutor = (*NodeAdapter)(nil)

// Ensure gossh import is used (for session type compatibility)
var _ = gossh.Client{}
