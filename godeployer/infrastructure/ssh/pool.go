// ============================================================
// 文件：pool.go
// 作用：🔌 SSH 连接池——管理到远程服务器的"电话线"！
//
// 什么是"连接池"？
// 每次 SSH 连接就像打一次电话：
// - 拨号（建立连接）很慢，可能需要 1-2 秒
// - 挂断后下次又要重新拨号
//
// 连接池就像：打完电话不挂断，把电话放在一边。
// 下次需要时直接拿起来用，不用重新拨号！
// 这叫做"复用连接"——大大节省时间。
//
// 这个池子实现了：
// 1. 最多创建 N 个连接（默认 10 个）
// 2. 用完了放回池子，给别人用
// 3. 如果池子满了，新请求就要排队等待
// 4. 如果连不上或连接断了，自动清理
// ============================================================

package ssh

import (
	"context" // 📡 上下文：支持超时等待
	"fmt"     // ✏️ 格式化
	"os"      // 💻 读取 SSH 私钥文件
	"path/filepath" // 📁 路径处理
	"strings" // 📏 字符串处理
	"sync"    // 🔒 并发锁
	"time"    // ⏰ 超时设置

	"deploy/godeployer/domain" // 📋 领域层：使用 ServerConfig

	"golang.org/x/crypto/ssh" // 🔑 SSH 库：建立 SSH 连接
)

// ============================================================
// 📦 SSHPool：并发安全的 SSH 连接池
//
// 就像一个"电话交换机"☎️：
// - idle：空闲的电话（可以随时用）
// - active：正在被使用的电话数
// - maxConns：最多能装多少部电话
// ============================================================

// SSHPool 提供并发安全的 SSH 连接池管理
type SSHPool struct {
	server   domain.ServerConfig // 🖥️ 目标服务器信息（IP、端口、用户名）
	maxConns int                 // 📊 最大连接数（最多同时保持多少个 SSH 连接）
	idle     chan *ssh.Client    // 💤 空闲连接队列（缓冲 channel）
	mu       sync.Mutex          // 🔒 互斥锁：保护 active 计数器
	active   int                 // 🔢 当前活跃的连接数
}

// NewSSHPool 创建一个固定容量的 SSH 连接池
func NewSSHPool(server domain.ServerConfig, maxConns int) *SSHPool {
	// 如果没传最大连接数，默认 10 个
	if maxConns <= 0 {
		maxConns = 10
	}
	return &SSHPool{
		server:   server,                              // 🖥️ 目标服务器
		maxConns: maxConns,                             // 📊 最多多少连接
		idle:     make(chan *ssh.Client, maxConns),     // 💤 空闲队列（容量 = maxConns）
	}
}

// ============================================================
// 🔨 createClient：创建一个新的 SSH 连接
//
// 这个过程包括：
// 1. 读取 SSH 私钥文件（~/.ssh/id_rsa）
// 2. 解析私钥（用密码还是直接解析）
// 3. 配置连接参数
// 4. 建立 TCP 连接 + SSH 握手
// ============================================================

func (p *SSHPool) createClient() (*ssh.Client, error) {
	// 1️⃣ 处理 SSH 私钥路径
	// 如果路径以 ~ 开头，替换为用户的 home 目录
	// 比如 ~/.ssh/id_rsa → /home/username/.ssh/id_rsa
	path := p.server.SSHKeyPath
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// 2️⃣ 读取私钥文件
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// 3️⃣ 解析私钥（将文件内容解析成可用的签名器）
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 4️⃣ 配置 SSH 客户端
	config := &ssh.ClientConfig{
		User: p.server.User, // 👤 SSH 登录用户名
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer), // 🔑 用公钥认证方式登录
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ 开发环境：不验证主机密钥（生产环境应该用 KnownHosts）
		Timeout:         10 * time.Second,             // ⏰ 连接超时：最多等 10 秒
	}

	// 5️⃣ 建立连接！dial = 拨号
	addr := fmt.Sprintf("%s:%d", p.server.Host, p.server.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect SSH server: %w", err)
	}
	return client, nil // ✅ 连接成功！
}

// ============================================================
// 📞 Get：从池中获取一个 SSH 连接
//
// 逻辑（优先用空闲的，没有就新建，超了就等待）：
// 1. 先看有没有空闲的 → 有就直接拿
// 2. 没有空闲的，看还能不能新建 → 能就新建
// 3. 不能新建（满了）→ 阻塞等待别人归还
// 4. 如果 ctx 取消了（超时/手动取消）→ 返回错误
// ============================================================

// Get 获取一个 SSH 客户端，支持 ctx 超时机制
func (p *SSHPool) Get(ctx context.Context) (*ssh.Client, error) {
	// 1️⃣ 先从空闲池拿（非阻塞）
	select {
	case client := <-p.idle:
		return client, nil // ✅ 有现成的，直接用！
	default:
		// 没有空闲的，继续走下面的逻辑
	}

	// 2️⃣ 检查能不能新建连接
	p.mu.Lock()
	if p.active < p.maxConns {
		// 还没到上限，可以新建
		p.active++
		p.mu.Unlock()

		client, err := p.createClient()
		if err != nil {
			// 创建失败！要记得把计数器减回去
			p.mu.Lock()
			p.active--
			p.mu.Unlock()
			return nil, err
		}
		return client, nil // ✅ 新连接建立成功
	}
	p.mu.Unlock()

	// 3️⃣ 满了！阻塞等待别人归还，或者超时
	select {
	case client := <-p.idle:
		return client, nil // ✅ 终于等到别人归还了！
	case <-ctx.Done():
		return nil, ctx.Err() // ⏰ 等太久了，取消
	}
}

// ============================================================
// 🔄 Put：归还连接
//
// 用完了要放回去！用完不放就是"内存泄漏"！
// - 如果连接有错误，就关闭它（不要了），并减少 active 计数
// - 如果连接正常，放回空闲池，给别人用
// ============================================================

// Put 归还连接或将其废弃
func (p *SSHPool) Put(client *ssh.Client, err error) {
	if err != nil {
		// ❌ 连接有错误，关闭它不再使用
		_ = client.Close()
		p.mu.Lock()
		p.active-- // 活跃数减 1
		p.mu.Unlock()
		return
	}

	// ✅ 连接正常，放回空闲池
	select {
	case p.idle <- client: // 放入空闲队列
	default:
		// 理论上不会到这里，因为 maxConns = cap(idle)
		// 但如果发生了，关闭连接防止泄漏
		_ = client.Close()
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
	}
}

// ============================================================
// 🧹 Close：关闭所有空闲连接
//
// 程序退出前调用这个函数，清理所有 SSH 连接。
// 就像下班前把电话都挂好~ 📞
// ============================================================

// Close 销毁池中所有空闲连接
func (p *SSHPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	// 不断从空闲队列取连接，然后关闭
	for {
		select {
		case client := <-p.idle:
			if err := client.Close(); err != nil && firstErr == nil {
				firstErr = err // 记录第一个错误
			}
		default:
			// 队列空了，退出
			p.active = 0 // 重置活跃计数
			return firstErr
		}
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 为什么要有"连接池"？
//    A: 建立 SSH 连接很耗时（需要 TCP 握手 + SSH 认证）！
//       连接池复用已有连接，就像不挂断电话直接打下一个~
//
// 2. Q: idle 为什么是 chan *ssh.Client？
//    A: channel 天然是"先进先出"的队列，而且可以安全地被多个
//       goroutine 同时使用。非常适合做连接池的底层存储！
//
// 中级（面试常考）：
// 3. Q: Get 方法的三级获取策略是什么？
//    A: 1) 非阻塞取空闲连接；2) 没空闲就新建（未超上限）；
//       3) 超上限就阻塞等待。效率最高，且不会无限制创建连接~
//
// 4. Q: 为什么要用 InsecureIgnoreHostKey？
//    A: 方便开发环境快速使用。生产环境应该用 KnownHosts 回调
//       来验证服务器身份，防止中间人攻击~
//
// 高级（架构师级别）：
// 5. Q: Put 方法中，为什么如果 idle 放不进去就关闭连接？
//    A: 防止连接泄漏！正常情况下 idle 容量 = maxConns，不会满。
//       但如果业务层有 bug（比如调用 Put 比 Get 多），
//       兜底关闭连接可以防止积累僵尸连接耗尽系统资源~
// ============================================================
