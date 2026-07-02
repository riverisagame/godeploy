// ============================================================
// 文件：deploy_service.go
// 作用：🏭 GoDeploy 的"核心工厂"——DeployEngine（部署引擎）！
//
// 这个文件是整个系统最核心的部分！
// 它负责执行真正的"部署流水线"，包括：
//
// 1. 接收部署任务（SubmitJob）
// 2. 启动调度器（StartDispatcher）——多个工人并发处理任务
// 3. 执行部署流水线（RunDeploy）：
//    a. Clone 仓库代码
//    b. Checkout 指定提交
//    c. 执行本地构建命令
//    d. Phase 1: 并发 Rsync 到所有目标服务器
//    e. Phase 2: 原子切换 Symlink
//    f. 失败时自动集群回滚
// 4. 回滚（RunRollback / RunRollbackToTask）
// 5. 优雅关闭（Close）
//
// 给初二小白的比喻：
// 这就像一家"部署工厂"🏭：
// - 工厂有一份订单列表（队列）
// - 几个工人同时工作（调度器）
// - 每笔订单的流程都是一样的（流水线）
// - 出错了还有应急预案（回滚）
// ============================================================

package application

import (
	"context"
	"deploy/godeployer/infrastructure/git"
	"deploy/godeployer/infrastructure/sys"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/ssh"
)

// ============================================================
// ❌ 自定义错误：让错误信息更清晰
// ============================================================

var (
	ErrQueueFull    = errors.New("deploy queue is full")       // 📥 队列满了，稍后再试
	ErrEngineClosed = errors.New("deploy engine is closed")    // 🚫 引擎已关闭，不接受新任务
)

// ============================================================
// 🏭 DeployEngine：部署引擎——整个系统的"心脏"！
//
// 它管理着：
// - 任务仓库（taskRepo）：记录部署任务的状态
// - 节点执行器（executor）：操作远程服务器（SSH）
// - 领域服务（deploySvc）：部署的业务规则
// - SSH 连接池（pools）：复用 SSH 连接，不用每次重新建立
// - 任务队列（queue）：待处理的部署任务列表
// - 项目锁（projectLocks）：同一项目同一时间只能部署一次
// ============================================================

// DeployJob is now domain.DeployJob

type DeployEngine struct {
	taskRepo  domain.TaskRepository      // 📋 任务仓库：读写部署任务的数据库
	executor  domain.NodeExecutor         // 🖥️ 节点执行器：操作远程服务器
	deploySvc *domain.DeploymentService   // 🧠 部署领域服务：部署的业务规则

	pools  map[string]*ssh.SSHPool        // 🔌 SSH 连接池（key = "host:port"）
	poolMu sync.Mutex                     // 🔒 连接池的锁（确保线程安全）

	mu     sync.Mutex                     // 🔒 引擎自己的锁
	queue  chan *domain.DeployJob         // 📥 任务队列（缓冲 50 个）
	wg     sync.WaitGroup                 // 👥 等待所有工人干完活
	closed bool                           // 🚫 是否已关闭

	projectLocks sync.Map                 // 🔐 项目锁：防止同一项目同时部署多次
}

// NewDeployEngine 创建部署引擎实例。
// 就像开一家工厂，需要准备好：
// - 仓库（taskRepo）：用来记账
// - 工具（executor）：用来干活
// - 规则书（deploySvc）：告诉工人怎么干
func NewDeployEngine(taskRepo domain.TaskRepository, executor domain.NodeExecutor, deploySvc *domain.DeploymentService) *DeployEngine {
	return &DeployEngine{
		taskRepo:  taskRepo,
		executor:  executor,
		deploySvc: deploySvc,
		queue:     make(chan *domain.DeployJob, 50), // 📥 最多排队 50 个任务
		pools:     make(map[string]*ssh.SSHPool),
	}
}

// getPool 获取或创建到指定服务器的 SSH 连接池。
// 如果之前已经连接过同一台服务器，就复用已有的连接池。
// 就像：之前已经打过电话的朋友，下次直接拨号就行，不用重新查号码~
func (e *DeployEngine) getPool(server domain.ServerConfig) *ssh.SSHPool {
	e.poolMu.Lock()
	defer e.poolMu.Unlock()

	// 用 "host:port" 作为连接池的唯一标识
	key := fmt.Sprintf("%s:%d", server.Host, server.Port)

	// 如果已经有这个服务器的连接池，直接返回
	if p, ok := e.pools[key]; ok {
		return p
	}

	// 没有的话，新建一个连接池（最大 10 个连接）
	p := ssh.NewSSHPool(server, 10)
	e.pools[key] = p
	return p
}

// ============================================================
// 🏗️ RunLocalBuild：在本地执行构建命令
//
// 什么是"构建"？
// 就是把源代码变成可以运行的程序！
// 比如：
// - npm install && npm run build  → 编译前端
// - go build .                   → 编译 Go 程序
// - composer install             → 安装 PHP 依赖
//
// 这些命令在项目配置的 build.before_sync 中定义。
// ============================================================

// RunLocalBuild 在指定的构建工作区中依次执行前置构建命令。
func (e *DeployEngine) RunLocalBuild(ctx context.Context, proj domain.ProjectConfig, buildPath string) error {
	// 遍历配置文件中定义的构建命令列表
	// 按顺序一条一条执行
	for _, rawCmd := range proj.Build.BeforeSync {
		if rawCmd == "" {
			continue // 跳过空命令
		}

		// 根据操作系统选择命令行解释器
		// Windows 用 cmd /C，其他用 sh -c
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", rawCmd)
		} else {
			cmd = exec.Command("sh", "-c", rawCmd)
		}

		cmd.Dir = buildPath // 在构建目录中执行命令

		// 执行命令并捕获输出
		output, err := runCmd(ctx, cmd)
		if err != nil {
			// ❌ 命令失败了！返回错误，终止部署
			return fmt.Errorf("command %q failed (output: %s): %w", rawCmd, string(output), err)
		}
	}
	return nil // ✅ 所有命令都执行成功
}

// ============================================================
// 🔗 SwitchSymlink：在远程服务器上切换软链接
//
// 什么是"软链接"（Symlink）？
// 就像 Windows 的"快捷方式"——它指向另一个文件或目录。
//
// 我们的部署策略是：
// 1. 每次部署都创建一个新目录：releases/20260601_123456/
// 2. 一个名为 current 的软链接指向最新的目录
// 3. 用户访问时走 current 这个链接
//
// 切换软链接 = 把 current 从旧目录指向新目录
// 这个操作是"原子的"——一瞬间完成，用户无感知！
// ============================================================

// SwitchSymlink 对目标服务器执行无空窗期的原子软链接切换。
func (e *DeployEngine) SwitchSymlink(server domain.ServerConfig, releaseName string) error {
	if e.executor == nil {
		return fmt.Errorf("node executor is not initialized")
	}
	return e.executor.SwitchSymlink(context.Background(), server, releaseName)
}

// ============================================================
// ↩️ 回滚操作
//
// 回滚就是"退回到上一个成功的版本"。
// 如果新版本有问题，马上把 current 软链接指回旧版本。
// ============================================================

// RunRollbackToTask 将指定项目和环境的目标服务器回滚到指定的任务 ID 对应的 Release 版本。
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (e *DeployEngine) RunRollbackToTask(targetTaskID int64, server domain.ServerConfig) error {
	if e.taskRepo == nil {
		return fmt.Errorf("task repository is required for rollback")
	}

	// 1. 查询要回滚到的目标任务
	task, err := e.taskRepo.GetTaskByID(int(targetTaskID))
	if err != nil {
		return fmt.Errorf("failed to query rollback version: %w", err)
	}
	if task == nil || task.Status != domain.StatusSuccess {
		return fmt.Errorf("specified task is not a successful release or does not exist")
	}

	// 2. 拿到那个版本的 release 名称
	releaseName := task.ReleaseName

	// 3. 把目标服务器的软链接切回那个版本
	if err := e.SwitchSymlink(server, releaseName); err != nil {
		return fmt.Errorf("rollback symlink switch failed: %w", err)
	}

	// 4. 在数据库里标记这个任务已经被回滚了
	if err := e.taskRepo.UpdateTaskStatus(int(targetTaskID), domain.StatusRolledBack); err != nil {
		return fmt.Errorf("database update failed but symlink rollback succeeded: %w", err)
	}

	return nil
}

// RunRollback 将指定项目和环境的目标服务器回滚到上一个成功的 Release 版本。
// 它会自动找到最近一次成功的部署，然后退回到那个版本。
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (e *DeployEngine) RunRollback(projectID, envID string, server domain.ServerConfig) error {
	if e.taskRepo == nil {
		return fmt.Errorf("task repository is required for rollback")
	}

	// 查询该项目环境最近的 10 条任务
	tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 10)
	if err != nil {
		return fmt.Errorf("failed to query rollback version: %w", err)
	}

	// 找到倒数第二个成功的版本
	// （第一个成功的版本是当前版本，我们要回退到上一个）
	var prevTaskID int64
	var successCount int
	for _, t := range tasks {
		if t.Status == domain.StatusSuccess {
			successCount++
			if successCount == 2 {
				prevTaskID = int64(t.ID)
				break
			}
		}
	}
	if prevTaskID == 0 {
		return fmt.Errorf("no previous successful release found to rollback to")
	}

	return e.RunRollbackToTask(prevTaskID, server)
}

// ============================================================
// 📥 SubmitJob：提交部署任务到队列
//
// 当用户在网页上点击"部署"按钮时，这个函数被调用。
// 任务会被放进一个"队列"（先进先出），
// 由工作协程（worker）从队列中取出并执行。
// 如果队列满了（超过 50 个），直接返回错误。
// ============================================================

// SubmitJob 提交部署任务到调度队列。如果队列满则立即返回 ErrQueueFull
func (e *DeployEngine) SubmitJob(job *domain.DeployJob) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 如果引擎已关闭，不接受新任务
	if e.closed {
		return ErrEngineClosed
	}

	// 尝试将任务放入队列
	// 如果队列满了，走 default 分支返回错误
	select {
	case e.queue <- job:
		return nil    // ✅ 成功入队
	default:
		return ErrQueueFull // ❌ 队列已满
	}
}

// ============================================================
// 🏃 StartDispatcher：启动后台部署调度器
//
// 调度器 = 多个"工人"（goroutine）从队列中取任务执行。
// 参数 workers = 同时工作的工人数量。
//
// 每个工人做的事情：
// 1. 从队列中取出一个任务（for job := range e.queue）
// 2. 执行 RunDeploy（真正的部署流水线）
// 3. 捕获 panic（防止一个任务的崩溃影响其他任务）
// ============================================================

// StartDispatcher 启动后台部署调度器
func (e *DeployEngine) StartDispatcher(workers int) {
	e.wg.Add(workers)
	// 启动 N 个工作协程
	for i := 0; i < workers; i++ {
		go func() {
			defer e.wg.Done()
			// 不断从队列中取任务
			for job := range e.queue {
				func() {
					// 如果任务有取消函数，确保最后调用
					if job.Cancel != nil {
						defer job.Cancel()
					}
					// 捕获潜在的 panic，防止其他任务受影响
					defer func() {
						if r := recover(); r != nil {
							e.UpdateTaskStatus(job.TaskID, domain.StatusFailed)
							log.Printf("Deployment panic for task %d: %v", job.TaskID, r)
						}
					}()
					// 执行真正的部署流水线
					e.RunDeploy(job.Ctx, job.TaskID, job.Config, job.LogFilePath)
				}()
			}
		}()
	}
}

// ============================================================
// 🛑 Close：优雅关闭引擎
//
// 关闭时不会立即停止所有任务，而是：
// 1. 停止接收新任务（close(e.queue)）
// 2. 等待正在执行的任务完成
// 3. 如果超过 timeout 还没完成，就超时退出
// ============================================================

// Close 优雅停机，等待所有队列中的部署任务完成
func (e *DeployEngine) Close(timeout time.Duration) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil // 已经关闭了，没事
	}
	e.closed = true
	close(e.queue) // 1️⃣ 关闭队列，不再接收新任务
	e.mu.Unlock()

	// 2️⃣ 等待所有工作协程完成当前任务
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	// 3️⃣ 等待完成，或者超时
	select {
	case <-done:
		return nil // ✅ 所有任务完成
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for DeployEngine to close") // ⏰ 超时了
	}
}

// ============================================================
// 🏗️ RunDeploy：核心部署流水线！
//
// 这是整个系统最重要的函数！
// 它按顺序执行以下步骤：
//
// 第 1 步：从缓存中 Clone 仓库到本地
// 第 2 步：Checkout 指定分支/提交
// 第 3 步：执行本地构建（npm build 等）
// 第 4 步（Phase 1）：并发 Rsync 到所有目标服务器
// 第 5 步（Phase 2）：原子切换 Symlink + 后置 Hook
// 第 6 步（如果失败）：集群回滚
//
// 整个过程中，每一步出错都会：
// - 记录日志
// - 更新任务状态为失败
// - 终止部署（不会继续执行后面的步骤）
// ============================================================

// RunDeploy 触发完整的部署流水线（用于后台异步运行）
func (e *DeployEngine) RunDeploy(ctx context.Context, taskID int64, config *domain.Config, logFilePath string) {
	// 确保执行器在使用完毕后关闭
	if closer, ok := e.executor.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	// ---------- 第 0 步：准备 ----------
	// 从数据库查询任务详情
	task, err := e.taskRepo.GetTaskByID(int(taskID))
	if err != nil || task == nil {
		log.Printf("[Task %d] Failed to query task: %v", taskID, err)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// 提取任务的各项参数
	projectID := task.ProjectID       // 📁 项目 ID
	envID := task.EnvID               // 🌍 环境 ID
	commitID := task.CommitID         // 🔖 Git 提交 SHA
	releaseName := task.ReleaseName    // 📦 发布版本名
	extraExclude := task.ExtraExclude  // 🚫 额外排除规则

	if e.executor == nil {
		log.Printf("[Task %d] Error: node executor is not initialized", taskID)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// 🔐 项目锁：同一项目+环境同一时间只能有一个部署任务
	lockKey := fmt.Sprintf("%s:%s", projectID, envID)
	if _, loaded := e.projectLocks.LoadOrStore(lockKey, struct{}{}); loaded {
		log.Printf("[Task %d] Concurrent deployment lock rejected for %s", taskID, lockKey)
		e.UpdateTaskStatus(taskID, domain.StatusFailedLockRejected)
		return
	}
	defer e.projectLocks.Delete(lockKey) // 解锁

	// 标记任务状态为"部署中"
	e.UpdateTaskStatus(taskID, domain.StatusDeploying)

	// ---------- 日志设置 ----------
	// 创建日志文件
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[Task %d] Failed to create log file: %v", taskID, err)
	} else {
		defer logFile.Close()
	}

	// writeLog 是一个辅助函数：同时向终端和日志文件输出
	var logMu sync.Mutex
	writeLog := func(format string, args ...interface{}) {
		logMu.Lock()
		defer logMu.Unlock()
		msg := fmt.Sprintf("[%s] "+format+"\n", append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, args...)...)
		log.Print(msg)           // 🖥️ 终端输出
		if logFile != nil {
			_, _ = logFile.WriteString(msg) // 📝 写入文件
		}
	}

	// 查找项目配置
	proj, exists := config.Projects[projectID]
	if !exists {
		writeLog("Error: project config %s not found", projectID)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// 查找环境配置
	var targetEnv *domain.EnvironmentConfig
	for _, env := range proj.Environments {
		if env.ID == envID {
			targetEnv = &env
			break
		}
	}
	if targetEnv == nil {
		writeLog("Error: environment config %s not found", envID)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// ============================================================
	// 📥 第 1 步：Clone 仓库到本地工作区
	// ============================================================
	buildPath := filepath.Join(config.Global.WorkspacePath, projectID, releaseName)

	// 先更新本地 bare 缓存（裸仓库）到最新
	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	writeLog("Updating local bare repo cache...")
	if cacheErr := git.EnsureRepoCache(ctx, proj.Repo, projectID); cacheErr != nil {
		writeLog("Warning: failed to update bare repo cache: %v", cacheErr)
	}

	// 创建工作区目录
	if err := os.MkdirAll(filepath.Dir(buildPath), 0755); err != nil {
		writeLog("Error: failed to create workspace dir: %v", err)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// 从本地 bare 缓存 clone（速度比从远程 clone 快 100 倍！）
	cacheDir := git.GetCacheDir(projectID)
	var cloneCmd *exec.Cmd
	if _, statErr := os.Stat(cacheDir); statErr == nil {
		writeLog("Step 1: Cloning repository locally from cache %s into %s...", cacheDir, buildPath)
		cloneCmd = exec.Command("git", "clone", "--no-hardlinks", cacheDir, buildPath)
	} else {
		writeLog("Step 1: Cloning repository from remote URL %s into %s...", proj.Repo, buildPath)
		cloneCmd = exec.Command("git", "clone", proj.Repo, buildPath)
	}
	if output, err := runCmd(ctx, cloneCmd); err != nil {
		writeLog("Error: git clone failed: %v (output: %s)", err, string(output))
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// ============================================================
	// 🔖 第 2 步：Checkout 到指定提交/分支/标签
	// ============================================================
	writeLog("Step 2: Checking out target commit/branch: %s...", commitID)
	// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
	checkoutCmd := exec.Command("git", "checkout", commitID)
	checkoutCmd.Dir = buildPath
	if output, err := runCmd(ctx, checkoutCmd); err != nil {
		writeLog("Error: git checkout failed: %v (output: %s)", err, string(output))
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// ============================================================
	// 🏗️ 第 3 步：执行本地构建
	// ============================================================
	writeLog("Step 3: Executing local build hooks...")
	if err := e.RunLocalBuild(ctx, proj, buildPath); err != nil {
		writeLog("Error: local build hooks failed: %v", err)
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// ============================================================
	// 📤 第 4 步（Phase 1）：并发 Rsync 到所有目标服务器
	// ============================================================
	// @Ref: docs/sps/plans/20260527_m5_multinode_ir.md | @Date: 2026-05-27
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)     // 🚦 信号量：最多同时 10 个 rsync
	var rsyncMu sync.Mutex
	rsyncFailed := false

	// 查询上一个成功版本，用于 Rsync --link-dest 加速
	// --link-dest 让 rsync 和上一个版本对比，
	// 没变化的文件直接创建硬链接（不复制），大大加快速度！
	var prevReleaseName string
	tasks, _ := e.taskRepo.GetTasksByEnv(projectID, envID, 5)
	for _, t := range tasks {
		if t.Status == domain.StatusSuccess {
			prevReleaseName = t.ReleaseName
			break
		}
	}

	// 并发向所有服务器推送代码
	for _, srv := range targetEnv.Servers {
		wg.Add(1)
		go func(srv domain.ServerConfig) {
			defer wg.Done()
			sem <- struct{}{}       // 🚦 获取信号量（占位）
			defer func() { <-sem }() // 🚦 释放信号量

			writeLog("Step 4 [Phase1]: Synchronizing files to remote server %s:%d...", srv.Host, srv.Port)

			// 合并静态排除规则（项目配置）和动态排除规则（任务传过来的）
			var totalExcludes []string
			totalExcludes = append(totalExcludes, proj.Exclude...)
			if extraExclude != "" {
				for _, part := range strings.Split(extraExclude, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						totalExcludes = append(totalExcludes, part)
					}
				}
			}

			// 💀 安全检查：过滤掉可能包含 shell 注入的排除规则
			var safeExcludes []string
			for _, ex := range totalExcludes {
				if strings.ContainsAny(ex, ";|&`$<>") {
					writeLog("Warning: Dropped suspicious exclude pattern to prevent shell injection: %s", ex)
					continue
				}
				safeExcludes = append(safeExcludes, ex)
			}

			// 确保远程服务器上有 releases 目录
			releasesDir := filepath.ToSlash(filepath.Join(srv.DeployTo, "releases"))
			mkCmd := fmt.Sprintf("mkdir -p %s", releasesDir)
			if _, err := e.executor.RunCommand(ctx, srv, mkCmd); err != nil {
				writeLog("Error: failed to create remote releases directory on %s: %v", srv.Host, err)
				rsyncMu.Lock()
				rsyncFailed = true
				rsyncMu.Unlock()
				return
			}

			// 计算 --link-dest 的绝对路径
			var absoluteLinkDest string
			if prevReleaseName != "" {
				absoluteLinkDest = filepath.ToSlash(filepath.Join(releasesDir, prevReleaseName))
			}

			remoteReleasePath := filepath.ToSlash(filepath.Join(releasesDir, releaseName)) + "/"
			localBuildDir := buildPath + "/"

			// 执行 rsync 同步！
			if err := e.executor.Rsync(ctx, srv, localBuildDir, remoteReleasePath, absoluteLinkDest, safeExcludes); err != nil {
				writeLog("Error: Rsync failed on %s: %v", srv.Host, err)
				rsyncMu.Lock()
				rsyncFailed = true
				rsyncMu.Unlock()
				return
			}
		}(srv)
	}

	wg.Wait() // 等待所有服务器同步完成

	// 如果有任何服务器 rsync 失败，终止部署
	if rsyncFailed {
		writeLog("Error: Phase 1 Rsync failed on one or more nodes. Halting deployment.")
		e.UpdateTaskStatus(taskID, domain.StatusFailed)
		return
	}

	// ============================================================
	// 🔗 第 5 步（Phase 2）：并发 Symlink 切换 + 后置 Hook
	// ============================================================
	// @Ref: docs/sps/plans/20260527_m5_multinode_ir.md | @Date: 2026-05-27
	var symlinkMu sync.Mutex
	shouldRollback := false

	for _, srv := range targetEnv.Servers {
		wg.Add(1)
		go func(srv domain.ServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// 原子切换软链接！
			writeLog("Step 5 [Phase2]: Switching active symlink on %s:%d...", srv.Host, srv.Port)
			if err := e.SwitchSymlink(srv, releaseName); err != nil {
				writeLog("Error: atomic symlink switch failed on %s: %v", srv.Host, err)
				symlinkMu.Lock()
				shouldRollback = true // 🔴 标记需要回滚
				symlinkMu.Unlock()
				return
			}

			// 执行后置 hook（比如重启服务、清缓存等）
			if len(targetEnv.AfterSymlink) > 0 {
				writeLog("Step 6: Executing after_symlink remote hooks on %s...", srv.Host)
				for _, hook := range targetEnv.AfterSymlink {
					if hook == "" {
						continue
					}
					hookCmd := fmt.Sprintf("cd %s && %s", filepath.ToSlash(filepath.Join(srv.DeployTo, "current")), hook)
					if out, err := e.executor.RunCommand(ctx, srv, hookCmd); err != nil {
						writeLog("Warning: after_symlink hook %q failed on %s (output: %s): %v", hook, srv.Host, out, err)
					}
				}
			}
		}(srv)
	}

	wg.Wait()

	// ============================================================
	// 🚨 Phase 3：分布式并发回滚保护
	//
	// 如果 Phase 2 中任何节点的 symlink 切换失败了，
	// 我们要把所有节点都回滚到上一个版本！
	//
	// 这叫"集群脑裂保护"——防止部分服务器在新版本、
	// 部分在旧版本的"分裂"状态。
	// ============================================================
	if shouldRollback {
		writeLog("Error: Phase 2 Symlink switch failed. Triggering cluster-wide Rollback...")
		var rbMu sync.Mutex
		rollbackFailed := false

		// 并发回滚所有服务器
		for _, srv := range targetEnv.Servers {
			wg.Add(1)
			go func(srv domain.ServerConfig) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// 找到上一个成功的版本
				var rbReleaseName string
				tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 5)
				for _, t := range tasks {
					if t.Status == domain.StatusSuccess {
						rbReleaseName = t.ReleaseName
						break
					}
				}
				if rbReleaseName == "" {
					if err != nil {
						writeLog("Rollback Error: failed to query last success release for %s: %v", srv.Host, err)
						rbMu.Lock()
						rollbackFailed = true
						rbMu.Unlock()
					}
					return
				}

				// 切换 symlink 到上一个版本
				if err := e.SwitchSymlink(srv, rbReleaseName); err != nil {
					writeLog("Rollback Error: failed to rollback symlink on %s: %v", srv.Host, err)
					rbMu.Lock()
					rollbackFailed = true
					rbMu.Unlock()
				}
			}(srv)
		}
		wg.Wait()

		// 判断回滚结果
		if rollbackFailed {
			writeLog("CRITICAL: Rollback failed on one or more nodes! Brain Split detected!")
			e.UpdateTaskStatus(taskID, domain.StatusCriticalBrainSplit) // ☠️ 脑裂！
		} else {
			writeLog("Rollback successful. Marking task as failed.")
			e.UpdateTaskStatus(taskID, domain.StatusFailed)
		}
		return
	}

	// ============================================================
	// ✅ 部署成功！
	// ============================================================
	writeLog("Deployment completed successfully!")
	e.UpdateTaskStatus(taskID, domain.StatusSuccess)

	// 🎯 异步生成 diff 快照（记录这次改了哪些代码）
	// @Ref: docs/sps/decisions/20260529_diff_ux_loading_scan.md | @Date: 2026-05-29
	go e.cacheTaskDiff(taskID, projectID, envID, commitID, releaseName, config, logFilePath)
}

// UpdateTaskStatus 更新任务状态并写入数据库
func (e *DeployEngine) UpdateTaskStatus(taskID int64, status domain.DeployStatus) {
	err := e.taskRepo.UpdateTaskStatus(int(taskID), status)
	if err != nil {
		log.Printf("Failed to update task status in DB: %v", err)
	}
}

// ============================================================
// 📸 cacheTaskDiff：生成并缓存代码差异快照
//
// 部署成功后，异步计算这次部署和前一次部署的代码差异（diff）。
// 这样用户在网页上可以直观地看到"这次改了什么"。
//
// 因为这是"锦上添花"的功能，所以：
// - 用 goroutine 异步执行，不影响部署结果
// - 失败了只打日志，不影响主流程
// ============================================================

// @Ref: docs/sps/decisions/20260529_diff_ux_loading_scan.md | @Date: 2026-05-29
func (e *DeployEngine) cacheTaskDiff(taskID int64, projectID, envID, commitID, releaseName string, config *domain.Config, logFilePath string) {
	// 捕获可能的 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Task %d] Diff cache gen panic (recovered): %v", taskID, r)
		}
	}()

	// 找到上一次成功部署的 commit_id
	var prevCommit, targetType string
	task, err := e.taskRepo.GetTaskByID(int(taskID))
	if task != nil {
		targetType = task.TargetType
	}
	tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 100)
	if err == nil {
		for _, t := range tasks {
			if t.ID < int(taskID) && t.Status == domain.StatusSuccess {
				prevCommit = t.CommitID
				break
			}
		}
	}
	if prevCommit == "" {
		log.Printf("[Task %d] Diff cache skipped: no previous successful deploy for %s/%s", taskID, projectID, envID)
		return
	}

	// 找到 git 仓库目录
	buildPath := filepath.Join(config.Global.WorkspacePath, projectID, releaseName)
	gitRepoPath := buildPath
	if _, statErr := os.Stat(filepath.Join(buildPath, ".git")); os.IsNotExist(statErr) {
		cacheDir := git.GetCacheDir(projectID)
		if _, cacheErr := os.Stat(cacheDir); cacheErr == nil {
			gitRepoPath = cacheDir
		} else {
			found, walkErr := git.FindGitRepo(config.Global.WorkspacePath, commitID)
			if walkErr != nil || found == "" {
				log.Printf("[Task %d] Diff cache skipped: git repo not found", taskID)
				return
			}
			gitRepoPath = found
		}
	}

	// @Ref: docs/sps/plans/20260530_dual_diff_persistence_plan.md | @Date: 2026-05-30
	// 生成 diff
	var liveDiffStr, gitLogDiffStr, filesStr string
	if targetType == "commit" {
		liveDiffStr, filesStr = generateTaskDiff(prevCommit, commitID, gitRepoPath, 60*time.Second)
		if strings.TrimSpace(liveDiffStr) == "" {
			liveDiffStr = fmt.Sprintf("两次提交内容完全相同（%s → %s），无代码变更。", prevCommit[:8], commitID[:8])
		}
		gitLogDiffStr, _ = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
	} else {
		gitLogDiffStr, filesStr = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
		liveDiffStr = ""
	}

	// 截断过大的 diff（防止爆磁盘）
	limitBytes := config.Global.DiffMaxSizeKB * 1024
	if limitBytes <= 0 {
		limitBytes = 5120 * 1024
	}
	if len(liveDiffStr) > limitBytes {
		truncatedDiff := liveDiffStr[:limitBytes]
		filesStr = git.FilterFilesForTruncatedDiff(truncatedDiff, filesStr)
		liveDiffStr = truncatedDiff + "\n\n... [DIFF OUT OF LIMIT, TRUNCATED FOR SAFETY]"
	}
	if len(gitLogDiffStr) > limitBytes {
		gitLogDiffStr = gitLogDiffStr[:limitBytes] + "\n\n... [DIFF OUT OF LIMIT, TRUNCATED FOR SAFETY]"
	}

	// 写入缓存文件
	var createdTime string
	if task != nil {
		createdTime = task.CreatedAt.Format("2006-01-02 15:04:05")
	}
	createdYM := "default"
	if len(createdTime) >= 7 {
		createdYM = strings.ReplaceAll(createdTime[:7], "-", "")
	}
	diffCacheDir := filepath.Join(config.Global.LogPath, "diffs", "projects", projectID, createdYM)
	if sys.GetFreeDiskSpaceMB(config.Global.LogPath) >= config.Global.DiskMinSpaceMB {
		_ = os.MkdirAll(diffCacheDir, 0755)
		diffCacheFile := filepath.Join(diffCacheDir, fmt.Sprintf("task_%d_diff.log", taskID))
		cacheMap := map[string]string{
			"files":        filesStr,
			"diff":         liveDiffStr,
			"git_log_diff": gitLogDiffStr,
		}
		if cacheBytes, err := json.Marshal(cacheMap); err == nil {
			_ = os.WriteFile(diffCacheFile, cacheBytes, 0644)
			log.Printf("[Task %d] Diff cache written: %s", taskID, diffCacheFile)
		}
	}
}

// generateTaskDiff 执行 git diff 与 git diff --name-status，返回 (diff 文本, files 列表)。
func generateTaskDiff(prevCommit, currentCommit, gitRepoPath string, timeout time.Duration) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	diffCmd := exec.CommandContext(ctx, "git", "diff", prevCommit, currentCommit)
	diffCmd.Dir = gitRepoPath
	diffOutput, diffErr := diffCmd.CombinedOutput()
	diffStr := string(diffOutput)
	if diffErr != nil {
		log.Printf("Diff cache: git diff failed: %v", diffErr)
	}

	var filesStr string
	filesCmd := exec.CommandContext(ctx, "git", "diff", "--name-status", prevCommit, currentCommit)
	filesCmd.Dir = gitRepoPath
	if filesOut, filesErr := filesCmd.CombinedOutput(); filesErr == nil {
		filesStr = string(filesOut)
	}

	return diffStr, filesStr
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 什么是"部署"？
//    A: 把写好的代码放到服务器上，让用户能访问到！
//       就像写完作文贴到公告栏让大家看~
//
// 2. Q: 为什么要用队列（channel）来管理任务？
//    A: 防止同时太多部署任务把系统搞崩！
//       队列就像一个等候区，先来后到，一个一个处理~
//
// 3. Q: --link-dest 是做什么的？
//    A: rsync 的一个优化选项！文件没变就不复制，直接建个"快捷方式"指向旧文件，
//       超级省时间！就像你抄作业只抄改动的部分~
//
// 中级（面试常考）：
// 4. Q: 为什么需要"项目锁"（projectLocks）？
//    A: 防止同一项目同时部署多次！如果两个人同时部署同一个项目，
//       后面的 rsync 会覆盖前面的，导致混乱。用锁确保一次只部署一个~
//
// 5. Q: Phase 1 和 Phase 2 为什么要分开？
//    A: 两阶段提交！Phase 1 传输文件即使失败也不会影响当前服务（因为还没切换），
//       Phase 2 才是真正的切换。如果 Phase 2 失败，可以回滚 Phase 1 的传输~
//
// 6. Q: 什么是"脑裂"（Brain Split）？
//    A: 集群中部分服务器在新版本、部分在旧版本，无法达成一致。
//       就像一群人中有人说"向左走"有人说"向右走"，队伍就分裂了！
//
// 高级（架构师级别）：
// 7. Q: 为什么 diff 快照是异步生成的？
//    A: 生成 diff 需要执行 git diff，如果仓库很大可能会很慢。
//       异步执行确保不影响部署主流程的响应速度，
//       用户不用等 diff 生成完就能看到"部署成功"的结果~
//
// 8. Q: sem（信号量）chan struct{} 的用途？
//    A: 限制并发数！虽然所有服务器同时传输，但 rsync 很耗费带宽。
//       用信号量限制最多 10 个 rsync 同时运行，防止带宽被占满~
// ============================================================
