package application

import (
	"bytes"
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

var (
	ErrQueueFull    = errors.New("deploy queue is full")
	ErrEngineClosed = errors.New("deploy engine is closed")
)

// DeployJob is now domain.DeployJob

type DeployEngine struct {
	taskRepo domain.TaskRepository
	executor ssh.RemoteExecutor

	pools  map[string]*ssh.SSHPool
	poolMu sync.Mutex

	mu     sync.Mutex
	queue  chan *domain.DeployJob
	wg     sync.WaitGroup
	closed bool

	projectLocks sync.Map
}

func NewDeployEngine(taskRepo domain.TaskRepository, executor ssh.RemoteExecutor) *DeployEngine {
	return &DeployEngine{
		taskRepo: taskRepo,
		executor: executor,
		queue:    make(chan *domain.DeployJob, 50),
		pools:    make(map[string]*ssh.SSHPool),
	}
}

func (e *DeployEngine) getPool(server domain.ServerConfig) *ssh.SSHPool {
	e.poolMu.Lock()
	defer e.poolMu.Unlock()
	if e.pools == nil {
		e.pools = make(map[string]*ssh.SSHPool)
	}
	key := fmt.Sprintf("%s:%d", server.Host, server.Port)
	if p, ok := e.pools[key]; ok {
		return p
	}
	p := ssh.NewSSHPool(server, 10)
	e.pools[key] = p
	return p
}

// RunLocalBuild 在指定的构建工作区中依次执行前置构建命令。
func (e *DeployEngine) RunLocalBuild(ctx context.Context, proj domain.ProjectConfig, buildPath string) error {
	for _, rawCmd := range proj.Build.BeforeSync {
		if rawCmd == "" {
			continue
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", rawCmd)
		} else {
			cmd = exec.Command("sh", "-c", rawCmd)
		}

		cmd.Dir = buildPath
		output, err := runCmd(ctx, cmd)
		if err != nil {
			return fmt.Errorf("command %q failed (output: %s): %w", rawCmd, string(output), err)
		}
	}
	return nil
}

// SwitchSymlink 对目标服务器执行无空窗期的原子软链接切换。
func (e *DeployEngine) SwitchSymlink(server domain.ServerConfig, releaseName string) error {
	executor := e.executor
	if executor == nil {
		executor = ssh.NewSSHExecutor(server, e.getPool(server))
	}

	releasesDir := filepath.ToSlash(filepath.Join(server.DeployTo, "releases"))
	newReleasePath := filepath.ToSlash(filepath.Join(releasesDir, releaseName))
	tempSymlinkPath := filepath.ToSlash(filepath.Join(server.DeployTo, "current_temp"))
	currentSymlinkPath := filepath.ToSlash(filepath.Join(server.DeployTo, "current"))

	// 1 & 2. 创建临时软链接并原子重命名覆盖 current 链接
	linkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", newReleasePath, tempSymlinkPath, tempSymlinkPath, currentSymlinkPath)
	if _, err := executor.RunCommand(linkCmd); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	return nil
}

// RunRollbackToTask 将指定项目和环境的目标服务器回滚到指定的任务 ID 对应的 Release 版本。
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (e *DeployEngine) RunRollbackToTask(targetTaskID int64, server domain.ServerConfig) error {
	if e.taskRepo == nil {
		return fmt.Errorf("task repository is required for rollback")
	}

	task, err := e.taskRepo.GetTaskByID(int(targetTaskID))
	if err != nil {
		return fmt.Errorf("failed to query rollback version: %w", err)
	}
	if task == nil || task.Status != "success" {
		return fmt.Errorf("specified task is not a successful release or does not exist")
	}
	releaseName := task.ReleaseName

	// 将目标服务器软链接切换到对应版本
	if err := e.SwitchSymlink(server, releaseName); err != nil {
		return fmt.Errorf("rollback symlink switch failed: %w", err)
	}

	// 更新目标任务的状态为已回滚 (仅作标记)
	if err := e.taskRepo.UpdateTaskStatus(int(targetTaskID), "rolled_back"); err != nil {
		return fmt.Errorf("database update failed but symlink rollback succeeded: %w", err)
	}

	return nil
}

// RunRollback 将指定项目和环境的目标服务器回滚到上一个成功的 Release 版本。
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (e *DeployEngine) RunRollback(projectID, envID string, server domain.ServerConfig) error {
	if e.taskRepo == nil {
		return fmt.Errorf("task repository is required for rollback")
	}

	tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 10)
	if err != nil {
		return fmt.Errorf("failed to query rollback version: %w", err)
	}
	
	var prevTaskID int64
	var successCount int
	for _, t := range tasks {
		if t.Status == "success" {
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

// SubmitJob 提交部署任务到调度队列。如果队列满则立即返回 ErrQueueFull
func (e *DeployEngine) SubmitJob(job *domain.DeployJob) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ErrEngineClosed
	}

	select {
	case e.queue <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// StartDispatcher 启动后台部署调度器
func (e *DeployEngine) StartDispatcher(workers int) {
	e.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer e.wg.Done()
			for job := range e.queue {
				// 同步执行具体的部署流水线并捕获 Panic
				func() {
					if job.Cancel != nil {
						defer job.Cancel()
					}
					defer func() {
						if r := recover(); r != nil {
							e.UpdateTaskStatus(job.TaskID, "failed")
							log.Printf("Deployment panic for task %d: %v", job.TaskID, r)
						}
					}()
					e.RunDeploy(job.Ctx, job.TaskID, job.Config, job.LogFilePath)
				}()
			}
		}()
	}
}

// Close 优雅停机，等待所有队列中的部署任务完成
func (e *DeployEngine) Close(timeout time.Duration) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	close(e.queue)
	e.mu.Unlock()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for DeployEngine to close")
	}
}

// RunDeploy 触发完整的部署流水线（用于后台异步运行）
func (e *DeployEngine) RunDeploy(ctx context.Context, taskID int64, config *domain.Config, logFilePath string) {
	if closer, ok := e.executor.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	// 从数据库查询任务信息
	task, err := e.taskRepo.GetTaskByID(int(taskID))
	if err != nil || task == nil {
		log.Printf("[Task %d] Failed to query task: %v", taskID, err)
		e.UpdateTaskStatus(taskID, "failed")
		return
	}
	projectID := task.ProjectID
	envID := task.EnvID
	commitID := task.CommitID
	releaseName := task.ReleaseName
	extraExclude := task.ExtraExclude

	lockKey := fmt.Sprintf("%s:%s", projectID, envID)
	if _, loaded := e.projectLocks.LoadOrStore(lockKey, struct{}{}); loaded {
		log.Printf("[Task %d] Concurrent deployment lock rejected for %s", taskID, lockKey)
		e.UpdateTaskStatus(taskID, "failed_lock_rejected")
		return
	}
	defer e.projectLocks.Delete(lockKey)

	e.UpdateTaskStatus(taskID, "deploying")

	// 打开日志文件用于输出构建细节
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[Task %d] Failed to create log file: %v", taskID, err)
	} else {
		defer logFile.Close()
	}

	var logMu sync.Mutex
	writeLog := func(format string, args ...interface{}) {
		logMu.Lock()
		defer logMu.Unlock()
		msg := fmt.Sprintf("[%s] "+format+"\n", append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, args...)...)
		log.Print(msg)
		if logFile != nil {
			_, _ = logFile.WriteString(msg)
		}
	}

	proj, exists := config.Projects[projectID]
	if !exists {
		writeLog("Error: project config %s not found", projectID)
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	var targetEnv *domain.EnvironmentConfig
	for _, env := range proj.Environments {
		if env.ID == envID {
			targetEnv = &env
			break
		}
	}

	if targetEnv == nil {
		writeLog("Error: environment config %s not found", envID)
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 1. 本地检出目录建立
	buildPath := filepath.Join(config.Global.WorkspacePath, projectID, releaseName)
	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	// 部署前先确保本地 bare 仓库缓存已更新到最新
	writeLog("Updating local bare repo cache...")
	if cacheErr := git.EnsureRepoCache(ctx, proj.Repo, projectID); cacheErr != nil {
		writeLog("Warning: failed to update bare repo cache: %v", cacheErr)
	}

	if err := os.MkdirAll(filepath.Dir(buildPath), 0755); err != nil {
		writeLog("Error: failed to create workspace dir: %v", err)
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 优先从本地 Bare 缓存库中进行 clone，极大加快部署速度，彻底解决外网拉取极慢的问题
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
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 切换到指定的分支/Commit
	writeLog("Step 2: Checking out target commit/branch: %s...", commitID)
	// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
	checkoutCmd := exec.Command("git", "checkout", commitID)
	checkoutCmd.Dir = buildPath
	if output, err := runCmd(ctx, checkoutCmd); err != nil {
		writeLog("Error: git checkout failed: %v (output: %s)", err, string(output))
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 2. 本地构建
	writeLog("Step 3: Executing local build hooks...")
	if err := e.RunLocalBuild(ctx, proj, buildPath); err != nil {
		writeLog("Error: local build hooks failed: %v", err)
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 3. Phase 1: Rsync并发同步
	// @Ref: docs/sps/plans/20260527_m5_multinode_ir.md | @Date: 2026-05-27
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	var rsyncMu sync.Mutex
	rsyncFailed := false

	// 获取上一个成功版本，用作 Rsync link-dest 硬链接参考
	var prevReleaseName string
	tasks, _ := e.taskRepo.GetTasksByEnv(projectID, envID, 5)
	for _, t := range tasks {
		if t.Status == "success" {
			prevReleaseName = t.ReleaseName
			break
		}
	}

	for _, srv := range targetEnv.Servers {
		wg.Add(1)
		go func(srv domain.ServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			writeLog("Step 4 [Phase1]: Synchronizing files to remote server %s:%d...", srv.Host, srv.Port)
			executor := e.executor
			if executor == nil {
				executor = ssh.NewSSHExecutor(srv, e.getPool(srv))
				// We attach Ctx directly to the struct if needed, but NewSSHExecutor returns a pointer, so we can't easily chain unless we mutate
				if sshExec, ok := executor.(*ssh.SSHExecutor); ok {
					sshExec.Ctx = ctx
				}
			}

			// 合并静态与动态排除规则，注入到 executor 的 ExcludeList 中
			// @Ref: docs/sps/plans/20260529_deploy_enhancements_plan.md | @Date: 2026-05-29
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

			// 过滤可疑注入字符
			var safeExcludes []string
			for _, ex := range totalExcludes {
				if strings.ContainsAny(ex, ";|&`$<>") {
					writeLog("Warning: Dropped suspicious exclude pattern to prevent shell injection: %s", ex)
					continue
				}
				safeExcludes = append(safeExcludes, ex)
			}

			if sshExec, ok := executor.(*ssh.SSHExecutor); ok {
				sshExec.ExcludeList = safeExcludes
			}

			// 检查目标机 releases 目录是否存在
			releasesDir := filepath.ToSlash(filepath.Join(srv.DeployTo, "releases"))
			mkCmd := fmt.Sprintf("mkdir -p %s", releasesDir)
			if _, err := executor.RunCommand(mkCmd); err != nil {
				writeLog("Error: failed to create remote releases directory on %s: %v", srv.Host, err)
				rsyncMu.Lock()
				rsyncFailed = true
				rsyncMu.Unlock()
				return
			}

			var absoluteLinkDest string
			if prevReleaseName != "" {
				absoluteLinkDest = filepath.ToSlash(filepath.Join(releasesDir, prevReleaseName))
			}

			remoteReleasePath := filepath.ToSlash(filepath.Join(releasesDir, releaseName)) + "/"
			localBuildDir := buildPath + "/"

			// 调用 Rsync
			if err := executor.Rsync(localBuildDir, remoteReleasePath, absoluteLinkDest); err != nil {
				writeLog("Error: Rsync failed on %s: %v", srv.Host, err)
				rsyncMu.Lock()
				rsyncFailed = true
				rsyncMu.Unlock()
				return
			}
		}(srv)
	}

	wg.Wait()
	if rsyncFailed {
		writeLog("Error: Phase 1 Rsync failed on one or more nodes. Halting deployment.")
		e.UpdateTaskStatus(taskID, "failed")
		return
	}

	// 4. Phase 2: Symlink并发切换与Hooks
	// @Ref: docs/sps/plans/20260527_m5_multinode_ir.md | @Date: 2026-05-27
	var symlinkMu sync.Mutex
	shouldRollback := false

	for _, srv := range targetEnv.Servers {
		wg.Add(1)
		go func(srv domain.ServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			writeLog("Step 5 [Phase2]: Switching active symlink on %s:%d...", srv.Host, srv.Port)
			if err := e.SwitchSymlink(srv, releaseName); err != nil {
				writeLog("Error: atomic symlink switch failed on %s: %v", srv.Host, err)
				symlinkMu.Lock()
				shouldRollback = true
				symlinkMu.Unlock()
				return
			}

			// 5. 执行后置 hook (不影响主流程)
			if len(targetEnv.AfterSymlink) > 0 {
				writeLog("Step 6: Executing after_symlink remote hooks on %s...", srv.Host)
				executor := e.executor
				if executor == nil {
					executor = ssh.NewSSHExecutor(srv, e.getPool(srv))
					if sshExec, ok := executor.(*ssh.SSHExecutor); ok {
						sshExec.Ctx = ctx
					}
				}
				for _, hook := range targetEnv.AfterSymlink {
					if hook == "" {
						continue
					}
					hookCmd := fmt.Sprintf("cd %s && %s", filepath.ToSlash(filepath.Join(srv.DeployTo, "current")), hook)
					if out, err := executor.RunCommand(hookCmd); err != nil {
						writeLog("Warning: after_symlink hook %q failed on %s (output: %s): %v", hook, srv.Host, out, err)
					}
				}
			}
		}(srv)
	}

	wg.Wait()

	// 5. Phase 3: 分布式并发回滚保护
	if shouldRollback {
		writeLog("Error: Phase 2 Symlink switch failed. Triggering cluster-wide Rollback...")
		var rbMu sync.Mutex
		rollbackFailed := false

		for _, srv := range targetEnv.Servers {
			wg.Add(1)
			go func(srv domain.ServerConfig) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				var rbReleaseName string
				tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 5)
				for _, t := range tasks {
					if t.Status == "success" {
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

				if err := e.SwitchSymlink(srv, rbReleaseName); err != nil {
					writeLog("Rollback Error: failed to rollback symlink on %s: %v", srv.Host, err)
					rbMu.Lock()
					rollbackFailed = true
					rbMu.Unlock()
				}
			}(srv)
		}
		wg.Wait()

		if rollbackFailed {
			writeLog("CRITICAL: Rollback failed on one or more nodes! Brain Split detected!")
			e.UpdateTaskStatus(taskID, "critical_brain_split")
		} else {
			writeLog("Rollback successful. Marking task as failed.")
			e.UpdateTaskStatus(taskID, "failed")
		}
		return
	}

	writeLog("Deployment completed successfully!")
	e.UpdateTaskStatus(taskID, "success")

	// @Ref: docs/sps/decisions/20260529_diff_ux_loading_scan.md | @Date: 2026-05-29
	// 异步生成持久化 diff 快照，确保即使 git 仓库被清理后仍可查看
	go e.cacheTaskDiff(taskID, projectID, envID, commitID, releaseName, config, logFilePath)
}

func (e *DeployEngine) UpdateTaskStatus(taskID int64, status string) {
	err := e.taskRepo.UpdateTaskStatus(int(taskID), status)
	if err != nil {
		log.Printf("Failed to update task status in DB: %v", err)
	}
}

// @Ref: docs/sps/decisions/20260529_diff_ux_loading_scan.md | @Date: 2026-05-29
// cacheTaskDiff 在部署成功后异步生成持久化 diff 快照。
// 策略文档要求的"异步落盘"能力，确保即使后续 git 仓库被清理，diff 依然可用。
func (e *DeployEngine) cacheTaskDiff(taskID int64, projectID, envID, commitID, releaseName string, config *domain.Config, logFilePath string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Task %d] Diff cache gen panic (recovered): %v", taskID, r)
		}
	}()

	// 查询前置成功部署的 commit_id
	var prevCommit, targetType string
	task, err := e.taskRepo.GetTaskByID(int(taskID))
	if task != nil {
		targetType = task.TargetType
	}
	tasks, err := e.taskRepo.GetTasksByEnv(projectID, envID, 100)
	if err == nil {
		for _, t := range tasks {
			if t.ID < int(taskID) && t.Status == "success" {
				prevCommit = t.CommitID
				break
			}
		}
	}
	if prevCommit == "" {
		log.Printf("[Task %d] Diff cache skipped: no previous successful deploy for %s/%s", taskID, projectID, envID)
		return
	}

	// 定位 git 仓库目录（优先构建目录，降级为 workspace 搜索）
	buildPath := filepath.Join(config.Global.WorkspacePath, projectID, releaseName)
	gitRepoPath := buildPath
	if _, statErr := os.Stat(filepath.Join(buildPath, ".git")); os.IsNotExist(statErr) {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
		// 优先使用本地常驻 Bare 缓存目录，避免直接退化到全局 Walk 搜索
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
	// 增量上线（commit）两者都要保留；全量上线（branch/tag）仅保留 git log diff 即可。
	var liveDiffStr, gitLogDiffStr, filesStr string
	if targetType == "commit" {
		liveDiffStr, filesStr = generateTaskDiff(prevCommit, commitID, gitRepoPath, 60*time.Second)
		if strings.TrimSpace(liveDiffStr) == "" {
			liveDiffStr = fmt.Sprintf("两次提交内容完全相同（%s → %s），无代码变更。", prevCommit[:8], commitID[:8])
		}
		gitLogDiffStr, _ = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
	} else {
		gitLogDiffStr, filesStr = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
		liveDiffStr = "" // 全量上线不保留线上对比快照
	}

	// 应用全局限额，并确保 files 与 diff 按文件粒度一致截断
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

	// 写入缓存文件（含磁盘预留检查）
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

// runCmd 是一个安全的本地命令执行包裹函数，解决 Windows/Linux 等平台在 Context 超时时无法彻底清退整个子进程树的问题。
// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
func runCmd(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	sys.SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		sys.KillProcessGroup(cmd)
		<-done // 确保资源完全释放
		return buf.Bytes(), ctx.Err()
	case err := <-done:
		return buf.Bytes(), err
	}
}
