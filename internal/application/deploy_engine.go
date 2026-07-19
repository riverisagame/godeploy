package application

import (
	"fmt"
	"pdeploy/internal/domain"
	"strings"
	"sync"
	"time"
)

// DeployEngine 部署引擎：真实执行 Git Clone -> Rsync -> Symlink 流程
// @Ref: docs/sps/plans/20260719_p0_deploy_gaps_plan.md S3-S6 | @Date: 2026-07-19
type DeployEngine struct {
	sshClient  SSHClient
	gitClient  GitClient
	serverRepo domain.ServerRepository
	deploySvc  *DeployService

	// Pub/Sub for SSE logs. Map of DeploymentID to channels.
	subscribers map[uint][]chan string
	mu          sync.RWMutex

	// 并发部署锁：同一环境同时只允许一个部署
	deployLocks map[string]*sync.Mutex
	locksMu     sync.Mutex
}

// NewDeployEngine 创建部署引擎
func NewDeployEngine(sshClient SSHClient, gitClient GitClient, serverRepo domain.ServerRepository, deploySvc *DeployService) *DeployEngine {
	return &DeployEngine{
		sshClient:   sshClient,
		gitClient:   gitClient,
		serverRepo:  serverRepo,
		deploySvc:   deploySvc,
		subscribers: make(map[uint][]chan string),
		deployLocks: make(map[string]*sync.Mutex),
	}
}

// getDeployLock 获取环境级别的部署锁，防止并发部署同一环境
func (e *DeployEngine) getDeployLock(envKey string) *sync.Mutex {
	e.locksMu.Lock()
	defer e.locksMu.Unlock()
	if _, ok := e.deployLocks[envKey]; !ok {
		e.deployLocks[envKey] = &sync.Mutex{}
	}
	return e.deployLocks[envKey]
}

// Subscribe 订阅部署日志流
func (e *DeployEngine) Subscribe(deploymentID uint) chan string {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan string, 100)
	e.subscribers[deploymentID] = append(e.subscribers[deploymentID], ch)
	return ch
}

// Publish 发布日志到所有订阅者
func (e *DeployEngine) Publish(deploymentID uint, msg string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	subs := e.subscribers[deploymentID]
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Non-blocking if channel is full
		}
	}
}

// CloseSubscribers 关闭并清理指定部署的所有订阅
func (e *DeployEngine) CloseSubscribers(deploymentID uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	subs := e.subscribers[deploymentID]
	for _, ch := range subs {
		close(ch)
	}
	delete(e.subscribers, deploymentID)
}

// StartDeploy 异步执行完整部署流程：Clone -> Sync -> Symlink -> Hooks
func (e *DeployEngine) StartDeploy(deployment *domain.Deployment, project *domain.Project, env *domain.Environment) {
	go func() {
		defer e.CloseSubscribers(deployment.ID)

		// 环境级并发锁
		envKey := fmt.Sprintf("%d-%s", project.ID, env.Name)
		lock := e.getDeployLock(envKey)
		if !lock.TryLock() {
			e.Publish(deployment.ID, "ERROR: 该环境正在部署中，请等待当前部署完成。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, "环境并发部署冲突")
			return
		}
		defer lock.Unlock()

		var logLines []string
		appendLog := func(msg string) {
			logLines = append(logLines, msg)
			e.Publish(deployment.ID, msg)
		}

		deployFailed := false

		appendLog(fmt.Sprintf(">>> 启动部署 #%d | 项目: %s | 环境: %s | 分支: %s\n",
			deployment.ID, project.Name, env.Name, env.Branch))

		// ========== 阶段 1: Git Clone/Pull ==========
		appendLog(">>> [1/5] 拉取代码...\n")
		deployment.SetPhase("clone")

		logChan := make(chan string, 100)
		go func() {
			for msg := range logChan {
				appendLog(msg)
			}
		}()

		workspacePath, err := e.gitClient.CloneOrPull(project.RepoURL, env.Branch, project.Name, logChan)
		close(logChan)
		if err != nil {
			appendLog(fmt.Sprintf("ERROR: Git 拉取失败: %v\n", err))
			e.deploySvc.CompleteDeploy(deployment.ID, false, strings.Join(logLines, ""))
			return
		}
		appendLog(fmt.Sprintf(">>> 代码已拉取到: %s\n", workspacePath))

		// ========== 阶段 2-5: 逐台服务器部署 ==========
		if len(env.ServerIDs) == 0 {
			appendLog("ERROR: 该环境没有配置服务器。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, strings.Join(logLines, ""))
			return
		}

		releaseTimestamp := time.Now().Format("20060102_150405")

		for _, srvID := range env.ServerIDs {
			srv, err := e.serverRepo.FindByID(srvID)
			if err != nil || srv == nil {
				appendLog(fmt.Sprintf("ERROR: 服务器 %d 未找到。\n", srvID))
				deployFailed = true
				continue
			}

			appendLog(fmt.Sprintf(">>> 开始部署到服务器 %s (%s@%s:%d)...\n", srv.Name, srv.User, srv.IP, srv.Port))

			// 阶段 2: 创建 release 目录并同步文件
			appendLog(">>> [2/5] 同步文件到服务器...\n")
			deployment.SetPhase("sync")

			releaseDir := fmt.Sprintf("%s/releases/%s", env.DeployPath, releaseTimestamp)
			syncLogChan := make(chan string, 50)
			go func() {
				for msg := range syncLogChan {
					appendLog(msg)
				}
			}()

			// 创建远程目录
			err = e.sshClient.RunCommand(srv, fmt.Sprintf("mkdir -p %s", releaseDir), syncLogChan)
			if err != nil {
				appendLog(fmt.Sprintf("ERROR: 创建远程目录失败: %v\n", err))
				deployFailed = true
				close(syncLogChan)
				continue
			}

			// Rsync 同步
			err = e.sshClient.SyncFiles(srv, workspacePath, releaseDir, syncLogChan)
			close(syncLogChan)
			if err != nil {
				appendLog(fmt.Sprintf("ERROR: 文件同步失败: %v\n", err))
				deployFailed = true
				continue
			}

			// 阶段 3: Pre-deploy hook
			if env.PreDeploy != "" {
				appendLog(">>> [3/5] 执行前置脚本...\n")
				deployment.SetPhase("pre_deploy")
				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						appendLog(msg)
					}
				}()
				err = e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", releaseDir, env.PreDeploy), hookLogChan)
				close(hookLogChan)
				if err != nil {
					appendLog(fmt.Sprintf("ERROR: 前置脚本失败: %v\n", err))
					deployFailed = true
					continue
				}
			} else {
				appendLog(">>> [3/5] 无前置脚本，跳过。\n")
			}

			// 阶段 4: Symlink 原子切换
			appendLog(">>> [4/5] 切换 Symlink...\n")
			deployment.SetPhase("symlink")
			currentLink := fmt.Sprintf("%s/current", env.DeployPath)
			tmpLink := fmt.Sprintf("%s/current_tmp_%d", env.DeployPath, time.Now().UnixNano())
			symlinkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", releaseDir, tmpLink, tmpLink, currentLink)
			symlinkLogChan := make(chan string, 50)
			go func() {
				for msg := range symlinkLogChan {
					appendLog(msg)
				}
			}()
			err = e.sshClient.RunCommand(srv, symlinkCmd, symlinkLogChan)
			close(symlinkLogChan)
			if err != nil {
				appendLog(fmt.Sprintf("ERROR: Symlink 切换失败: %v\n", err))
				deployFailed = true
				continue
			}

			// 阶段 5: Post-deploy hook
			if env.PostDeploy != "" {
				appendLog(">>> [5/5] 执行后置脚本...\n")
				deployment.SetPhase("post_deploy")
				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						appendLog(msg)
					}
				}()
				err = e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", currentLink, env.PostDeploy), hookLogChan)
				close(hookLogChan)
				if err != nil {
					appendLog(fmt.Sprintf("ERROR: 后置脚本失败: %v\n", err))
					deployFailed = true
					continue
				}
			} else {
				appendLog(">>> [5/5] 无后置脚本，跳过。\n")
			}

			// 清理旧版本
			if project.KeepReleases > 0 {
				cleanCmd := fmt.Sprintf("cd %s/releases && ls -1dt */ 2>/dev/null | tail -n +%d | xargs rm -rf 2>/dev/null || true",
					env.DeployPath, project.KeepReleases+1)
				cleanLogChan := make(chan string, 10)
				go func() {
					for msg := range cleanLogChan {
						appendLog(msg)
					}
				}()
				e.sshClient.RunCommand(srv, cleanCmd, cleanLogChan)
				close(cleanLogChan)
			}

			appendLog(fmt.Sprintf(">>> 服务器 %s 部署成功。\n", srv.Name))
		}

		// 最终状态回写
		allLogs := strings.Join(logLines, "")
		if deployFailed {
			appendLog(">>> 部署完成，但部分服务器失败。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, allLogs)
		} else {
			appendLog(">>> 所有服务器部署成功！\n")
			e.deploySvc.CompleteDeploy(deployment.ID, true, allLogs)
		}

		// 等待日志 flush
		time.Sleep(300 * time.Millisecond)
	}()
}

// Rollback 回滚到指定 release 版本
func (e *DeployEngine) Rollback(deployment *domain.Deployment, env *domain.Environment, targetRelease string) {
	go func() {
		defer e.CloseSubscribers(deployment.ID)

		var logLines []string
		appendLog := func(msg string) {
			logLines = append(logLines, msg)
			e.Publish(deployment.ID, msg)
		}

		appendLog(fmt.Sprintf(">>> 回滚部署 #%d -> release: %s\n", deployment.ID, targetRelease))

		for _, srvID := range env.ServerIDs {
			srv, err := e.serverRepo.FindByID(srvID)
			if err != nil || srv == nil {
				appendLog(fmt.Sprintf("ERROR: 服务器 %d 未找到。\n", srvID))
				continue
			}

			appendLog(fmt.Sprintf(">>> 回滚服务器 %s...\n", srv.Name))

			// Symlink 回切
			releaseDir := fmt.Sprintf("%s/releases/%s", env.DeployPath, targetRelease)
			currentLink := fmt.Sprintf("%s/current", env.DeployPath)
			tmpLink := fmt.Sprintf("%s/current_tmp_%d", env.DeployPath, time.Now().UnixNano())
			symlinkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", releaseDir, tmpLink, tmpLink, currentLink)

			rollbackLogChan := make(chan string, 50)
			go func() {
				for msg := range rollbackLogChan {
					appendLog(msg)
				}
			}()
			err = e.sshClient.RunCommand(srv, symlinkCmd, rollbackLogChan)
			close(rollbackLogChan)
			if err != nil {
				appendLog(fmt.Sprintf("ERROR: 回滚 Symlink 切换失败: %v\n", err))
				continue
			}

			// Post-deploy hook
			if env.PostDeploy != "" {
				appendLog(">>> 执行后置脚本...\n")
				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						appendLog(msg)
					}
				}()
				e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", currentLink, env.PostDeploy), hookLogChan)
				close(hookLogChan)
			}

			appendLog(fmt.Sprintf(">>> 服务器 %s 回滚成功。\n", srv.Name))
		}

		allLogs := strings.Join(logLines, "")
		e.deploySvc.CompleteDeploy(deployment.ID, true, allLogs)
		appendLog(">>> 回滚完成。\n")

		time.Sleep(300 * time.Millisecond)
	}()
}
