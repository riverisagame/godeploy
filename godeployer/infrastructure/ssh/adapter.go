// ============================================================
// 文件：adapter.go
// 作用：🔌 SSH 适配器——把 SSH 操作包装成统一接口！
//
// 适配器（Adapter）设计模式：
// 就像电源转换器！🔌 中国的两脚插头→转换器→美国的插座
// 这里的 NodeAdapter 把"SSH 的具体操作"转换成
// domain.NodeExecutor 接口（这个接口领域层定义好了）。
//
// 这样，领域层只需要说"执行命令"（RunCommand），
// 不用管是通过 SSH、本地直接执行、还是通过 API 调用。
// NodeAdapter 负责把它们翻译成真正的 SSH 操作！
//
// 这个方法实现了 3 个操作：
// 1. RunCommand：在远程服务器上执行命令
// 2. SwitchSymlink：切换软链接
// 3. Rsync：同步文件
// ============================================================

// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31

package ssh

import (
	"context"       // 📡 上下文
	"fmt"           // ✏️ 格式化
	"path/filepath" // 📁 路径处理
	"strings"       // 📏 字符串处理
	"sync"          // 🔒 并发锁

	"deploy/godeployer/domain" // 📋 领域实体和接口

	gossh "golang.org/x/crypto/ssh" // 🔑 标准 SSH 库
)

// ============================================================
// 🎯 NodeAdapter：节点适配器
//
// 它实现了 domain.NodeExecutor 接口（在 deployment_service.go 中定义）。
// 负责把"对远程服务器操作"的抽象接口翻译成具体的 SSH 命令。
//
// 它内部管理着多个 SSH 连接池（一个服务器一个池子）。
// 就像管着一个"电话总机"——每个服务器分机有自己的电话线。
// ============================================================

// NodeAdapter 包装 SSH 连接池映射，将底层 SSH 能力适配为 domain.NodeExecutor 接口。
type NodeAdapter struct {
	mu    sync.Mutex                // 🔒 保护 pools 的锁
	pools map[string]*SSHPool       // 📦 连接池映射表（key = "host:port"）
}

// NewNodeAdapter 创建空的 NodeAdapter 实例，池按需创建。
func NewNodeAdapter() *NodeAdapter {
	return &NodeAdapter{
		pools: make(map[string]*SSHPool), // 🆕 初始化空的连接池映射表
	}
}

// getPool 获取或创建到指定服务器的连接池
func (a *NodeAdapter) getPool(node domain.ServerConfig) *SSHPool {
	// 用 "host:port" 作为唯一标识
	key := fmt.Sprintf("%s:%d", node.Host, node.Port)

	a.mu.Lock()
	defer a.mu.Unlock()

	// 如果已经有这个服务器的池子，直接复用
	if pool, exists := a.pools[key]; exists {
		return pool
	}

	// 没有就新建一个（最多 10 个连接）
	pool := NewSSHPool(node, 10)
	a.pools[key] = pool
	return pool
}

// ============================================================
// 🖥️ RunCommand：在远程服务器上执行一条命令
//
// 流程：
// 1. 从连接池获取一个 SSH 连接
// 2. 创建一个新的"会话"（Session）
// 3. 在会话中执行命令
// 4. 获取命令的输出
// 5. 归还连接回池子
//
// 就像：拿起电话（连接）→ 说话（执行命令）→ 听回复（获取输出）→ 挂好电话（归还）
// ============================================================

// RunCommand 在目标节点上执行任意命令，返回标准输出。
func (a *NodeAdapter) RunCommand(ctx context.Context, node domain.ServerConfig, cmd string) (string, error) {
	// 获取连接池（或者创建）
	pool := a.getPool(node)
	// 从池子里拿一个 SSH 连接
	client, err := pool.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("runcommand: %w", err)
	}
	// 用完了记得归还
	defer pool.Put(client, err)

	// 创建一个 SSH 会话（Session）
	// 每个命令都在独立的 Session 中执行
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("runcommand: new session: %w", err)
	}
	defer session.Close() // 用完就关

	// 在远程服务器上执行命令，并获取 stdout + stderr 的合并输出
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("runcommand: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// ============================================================
// 🔗 SwitchSymlink：在远程服务器上执行"原子软链接切换"
//
// 这是实现"零停机发布"的关键！
//
// 假设：
// - /var/www/myapp/releases/20260601_120000/  ← 新版本
// - /var/www/myapp/current → 指向旧版本的软链接
//
// 切换命令（摘选自代码）：
//   temp_link="/var/www/myapp/current.0"
//   ln -sfn "/var/www/myapp/releases/20260601_120000" "$temp_link"
//   mv -fT "$temp_link" "/var/www/myapp/current"
//
// 为什么这是"原子操作"？
// 在 Linux 上，mv 命令（在同一个文件系统内）是原子的！
// 一瞬间完成，不存在"半中间状态"。
// 而且用临时链接避免了"先删后建"导致的空窗期！
// ============================================================

// SwitchSymlink 在目标节点上执行原子软链接切换。
func (a *NodeAdapter) SwitchSymlink(ctx context.Context, node domain.ServerConfig, releaseName string) error {
	// 计算路径
	releasesDir := filepath.ToSlash(filepath.Join(node.DeployTo, "releases"))   // .../releases/
	targetPath := filepath.ToSlash(filepath.Join(releasesDir, releaseName))     // .../releases/xxx/
	currentPath := filepath.ToSlash(filepath.Join(node.DeployTo, "current"))    // .../current

	// 原子切换命令！
	// 1. ln -sfn：创建指向新版本的临时软链接
	// 2. mv -fT：用临时链接原子替换 current
	cmd := fmt.Sprintf(
		"temp_link=\"%s.%d\" && ln -sfn \"%s\" \"$temp_link\" && mv -fT \"$temp_link\" \"%s\"",
		currentPath, 0, targetPath, currentPath,
	)

	_, err := a.RunCommand(ctx, node, cmd)
	return err
}

// ============================================================
// 📤 Rsync：通过 rsync 同步文件到远程服务器
//
// rsync 是 Linux 上一个超好用的文件同步工具！
// 特点：
// 1. 只传输变化的文件（增量传输）
// 2. 支持压缩传输（-z 参数）
// 3. 支持排除模式（--exclude=node_modules）
// 4. 支持硬链接（--link-dest）
//
// 这里的 rsync 命令是通过 SSH 传输的（-e 'ssh -p 22 -i key'）
// 相当于：先建立 SSH 连接，然后通过这个连接传文件~
// ============================================================

// Rsync 将构建产物通过 rsync 同步到目标节点，支持排除列表和硬链接参考。
func (a *NodeAdapter) Rsync(ctx context.Context, node domain.ServerConfig, localPath, remotePath, linkDest string, excludes []string) error {
	pool := a.getPool(node)
	client, err := pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("rsync: %w", err)
	}
	defer pool.Put(client, err)

	// 构造排除参数
	// 比如 ["node_modules", ".git"] → --exclude='node_modules' --exclude='.git'
	excludeArgs := ""
	for _, p := range excludes {
		excludeArgs += fmt.Sprintf(" --exclude='%s'", p)
	}

	// 构造 --link-dest 参数（如果提供了）
	linkDestArg := ""
	if linkDest != "" {
		linkDestArg = fmt.Sprintf(" --link-dest='%s'", linkDest)
	}

	// 完整的 rsync 命令
	// 例如：rsync -avz --delete --exclude='node_modules' -e 'ssh -p 22 -i key' ./build/ user@host:/var/www/releases/xxx/
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

// ============================================================
// 🧹 Close：释放所有连接池的资源
// ============================================================

// Close 释放所有连接池资源。
func (a *NodeAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var lastErr error
	// 关闭所有连接池
	for key, pool := range a.pools {
		if err := pool.Close(); err != nil {
			lastErr = fmt.Errorf("close pool %s: %w", key, err)
		}
	}
	return lastErr
}

// ============================================================
// ✅ 编译时检查：确保 NodeAdapter 实现了 NodeExecutor 接口
//
// var _ domain.NodeExecutor = (*NodeAdapter)(nil)
// 这行代码的意思是：
// 把 nil 转成 *NodeAdapter，然后赋值给 NodeExecutor 类型的变量 _
// 如果 NodeAdapter 没完全实现 NodeExecutor 接口，编译就会报错！
// 这是一种在编译期就检查接口实现是否完整的小技巧~
// ============================================================

// compile-time check
var _ domain.NodeExecutor = (*NodeAdapter)(nil)

// Ensure gossh import is used (for session type compatibility)
var _ = gossh.Client{}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 适配器（Adapter）模式是什么？
//    A: 把一种接口转成另一种接口。就像电源转换器——
//       把 SSH 的具体操作转换成 NodeExecutor 定义的统一接口~
//
// 2. Q: 什么是"原子操作"？
//    A: 要么全部完成，要么什么都没发生！没有"正在切换中"的状态。
//       Linux 的 mv 命令在同一个磁盘上就是原子的~
//
// 中级（面试常考）：
// 3. Q: SwitchSymlink 为什么先用临时链接再 mv？
//    A: 防止"空窗期"！如果直接 rm old_link && ln -s new_link，
//       在 rm 之后 ln 之前有一瞬间 current 不存在，用户会访问不到！
//       用 mv 替换保证任何时候 current 都指向一个有效目录~
//
// 4. Q: 编译时检查 var _ domain.NodeExecutor = (*NodeAdapter)(nil) 是干什么的？
//    A: 确保 NodeAdapter 实现了 NodeExecutor 接口的所有方法！
//       如果某个方法没实现或者签名变了，编译时会立刻报错，
//       而不是等到运行时才发现~
//
// 高级（架构师级别）：
// 5. Q: 为什么 RunCommand 在 adapter 里执行，而不是在 executor.go 里？
//    A: executor.go 是"本地直接执行"的版本（给 demo 用的），
//       adapter.go 是"通过 SSH 远程执行"的版本（给生产用的）。
//       两种实现都满足 NodeExecutor 接口，可以随时切换~
// ============================================================
