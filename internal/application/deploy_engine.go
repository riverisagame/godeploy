package application

import (
	"fmt"
	"os"
	"os/exec"
	"pdeploy/internal/domain"
	"runtime"
	"strings"
	"sync"
	"time"
)

type DeployEngine struct {
	sshClient   SSHClient
	gitClient   GitClient
	serverSvc   *ServerService
	deploySvc   *DeployService
	
	// Real-time log broadcasting
	subscribers map[uint][]chan string
	subMu       sync.RWMutex
	
	// Log history
	logHistory  map[uint][]string
	historyMu   sync.RWMutex

	// Concurrency control per environment
	deployLocks map[string]*sync.Mutex
	lockMu      sync.Mutex
}

func NewDeployEngine(sshClient SSHClient, gitClient GitClient, serverSvc *ServerService, deploySvc *DeployService) *DeployEngine {
	return &DeployEngine{
		sshClient:   sshClient,
		gitClient:   gitClient,
		serverSvc:   serverSvc,
		deploySvc:   deploySvc,
		subscribers: make(map[uint][]chan string),
		logHistory:  make(map[uint][]string),
		deployLocks: make(map[string]*sync.Mutex),
	}
}

func (e *DeployEngine) getDeployLock(envKey string) *sync.Mutex {
	e.lockMu.Lock()
	defer e.lockMu.Unlock()
	if lock, exists := e.deployLocks[envKey]; exists {
		return lock
	}
	lock := &sync.Mutex{}
	e.deployLocks[envKey] = lock
	return lock
}

func (e *DeployEngine) Subscribe(deployID uint) chan string {
	e.subMu.Lock()
	defer e.subMu.Unlock()
	ch := make(chan string, 100)
	e.subscribers[deployID] = append(e.subscribers[deployID], ch)
	
	e.historyMu.RLock()
	defer e.historyMu.RUnlock()
	if history, ok := e.logHistory[deployID]; ok {
		for _, line := range history {
			ch <- line
		}
	}
	return ch
}

func (e *DeployEngine) Publish(deployID uint, msg string) {
	e.historyMu.Lock()
	e.logHistory[deployID] = append(e.logHistory[deployID], msg)
	e.historyMu.Unlock()

	e.subMu.RLock()
	defer e.subMu.RUnlock()
	for _, ch := range e.subscribers[deployID] {
		select {
		case ch <- msg:
		default: // skip if blocked
		}
	}
}

func (e *DeployEngine) CloseSubscribers(deployID uint) {
	e.subMu.Lock()
	for _, ch := range e.subscribers[deployID] {
		close(ch)
	}
	delete(e.subscribers, deployID)
	e.subMu.Unlock()

	// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 2.2 | @Date: 2026-07-21
	e.historyMu.Lock()
	delete(e.logHistory, deployID)
	e.historyMu.Unlock()
}

func (e *DeployEngine) GetLogHistory(deployID uint) string {
	e.historyMu.RLock()
	defer e.historyMu.RUnlock()
	return strings.Join(e.logHistory[deployID], "")
}

func (e *DeployEngine) broadcastLog(deployID uint, msg string) {
	e.Publish(deployID, msg)
}

func (e *DeployEngine) StartDeploy(deployment *domain.Deployment, project *domain.Project, env *domain.Environment) {
	go func() {
		defer e.CloseSubscribers(deployment.ID)

		envKey := fmt.Sprintf("%d-%s", project.ID, env.Name)
		lock := e.getDeployLock(envKey)
		if !lock.TryLock() {
			e.broadcastLog(deployment.ID, "ERROR: 该环境正在部署中，请等待当前部署完成。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, "环境并发部署冲突", "")
			return
		}
		defer lock.Unlock()

		deployFailed := false
		e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 启动部署 #%d | 项目: %s | 环境: %s | 分支: %s\n",
			deployment.ID, project.Name, env.Name, env.Branch))

		e.broadcastLog(deployment.ID, ">>> [1/5] 拉取代码...\n")
		deployment.SetPhase("clone")

		logChan := make(chan string, 100)
		go func() {
			for msg := range logChan {
				e.broadcastLog(deployment.ID, msg)
			}
		}()

		if e.gitClient == nil {
			e.broadcastLog(deployment.ID, "Test mode: skipping git clone.\n")
			return
		}

		workspacePath, err := e.gitClient.CloneOrPull(project.RepoURL, env.Branch, project.Name, logChan)
		close(logChan)
		if err != nil {
			e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: Git 拉取失败: %v\n", err))
			e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
			return
		}
		e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 代码已拉取到: %s\n", workspacePath))

		if len(env.EnvVars) > 0 {
			e.broadcastLog(deployment.ID, ">>> 注入环境变量 (.env)...\n")
			envContent := ""
			for _, v := range env.EnvVars {
				envContent += fmt.Sprintf("%s=%s\n", v.Key, v.Value)
			}
			err = os.WriteFile(fmt.Sprintf("%s/.env", workspacePath), []byte(envContent), 0644)
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("WARN: 无法写入 .env 文件: %v\n", err))
			} else {
				e.broadcastLog(deployment.ID, ">>> .env 文件生成成功。\n")
			}
		}

		if env.BuildCommand != "" {
			e.broadcastLog(deployment.ID, ">>> [1.5/5] 执行本地构建...\n")
			deployment.SetPhase("build")

			var buildCmd *exec.Cmd
			if runtime.GOOS == "windows" {
				// On Windows, use cmd /c. But for cross-platform support we can also just run it if it's a batch file.
				// For safety, assuming they type 'npm install && npm run build'
				buildCmd = exec.Command("cmd", "/c", env.BuildCommand)
			} else {
				buildCmd = exec.Command("sh", "-c", env.BuildCommand)
			}
			buildCmd.Dir = workspacePath
			
			output, err := buildCmd.CombinedOutput()
			if len(output) > 0 {
				e.broadcastLog(deployment.ID, string(output)+"\n")
			}
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 本地构建失败: %v\n", err))
				e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
				return
			}
			e.broadcastLog(deployment.ID, ">>> 本地构建完成。\n")
		}

		if len(env.ServerIDs) == 0 {
			e.broadcastLog(deployment.ID, "ERROR: 该环境没有配置服务器。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
			return
		}

		releaseTimestamp := time.Now().Format("20060102_150405")

		for _, srvID := range env.ServerIDs {
			srv, err := e.serverSvc.GetServerByID(srvID)
			if err != nil || srv == nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 服务器 %d 未找到。\n", srvID))
				deployFailed = true
				continue
			}

			e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 开始部署到服务器 %s (%s@%s:%d)...\n", srv.Name, srv.User, srv.IP, srv.Port))

			e.broadcastLog(deployment.ID, ">>> [2/5] 同步文件到服务器...\n")
			deployment.SetPhase("sync")
			releaseDir := fmt.Sprintf("%s/releases/%s", env.DeployPath, releaseTimestamp)
			syncLogChan := make(chan string, 50)
			go func() {
				for msg := range syncLogChan {
					e.broadcastLog(deployment.ID, msg)
				}
			}()

			err = e.sshClient.RunCommand(srv, fmt.Sprintf("mkdir -p %s", releaseDir), syncLogChan)
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 创建远程目录失败: %v\n", err))
				deployFailed = true
				close(syncLogChan)
				continue
			}

			currentLink := fmt.Sprintf("%s/current", env.DeployPath)
			err = e.sshClient.SyncFiles(srv, workspacePath, releaseDir, currentLink, syncLogChan)
			close(syncLogChan)
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 文件同步失败: %v\n", err))
				deployFailed = true
				continue
			}

			// Shared Setup
			if env.SharedDirs != "" || env.SharedFiles != "" {
				e.broadcastLog(deployment.ID, ">>> [2.5/5] 配置共享目录/文件 (Shared)...\n")
				
				sharedDirPaths := strings.Split(env.SharedDirs, "\n")
				sharedFilePaths := strings.Split(env.SharedFiles, "\n")
				
				sharedSetupCmd := ""
				
				for _, dir := range sharedDirPaths {
					dir = strings.TrimSpace(dir)
					if dir != "" {
						sharedPath := fmt.Sprintf("%s/shared/%s", env.DeployPath, dir)
						releasePath := fmt.Sprintf("%s/%s", releaseDir, dir)
						sharedSetupCmd += fmt.Sprintf("mkdir -p %s && rm -rf %s && mkdir -p $(dirname %s) && ln -sfn %s %s && ", sharedPath, releasePath, releasePath, sharedPath, releasePath)
					}
				}
				
				for _, file := range sharedFilePaths {
					file = strings.TrimSpace(file)
					if file != "" {
						sharedPath := fmt.Sprintf("%s/shared/%s", env.DeployPath, file)
						releasePath := fmt.Sprintf("%s/%s", releaseDir, file)
						sharedSetupCmd += fmt.Sprintf("mkdir -p $(dirname %s) && touch -a %s && rm -f %s && mkdir -p $(dirname %s) && ln -sfn %s %s && ", sharedPath, sharedPath, releasePath, releasePath, sharedPath, releasePath)
					}
				}
				
				if sharedSetupCmd != "" {
					sharedSetupCmd += "true"
					sharedLogChan := make(chan string, 50)
					go func() {
						for msg := range sharedLogChan {
							e.broadcastLog(deployment.ID, msg)
						}
					}()
					err = e.sshClient.RunCommand(srv, sharedSetupCmd, sharedLogChan)
					close(sharedLogChan)
					if err != nil {
						e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 共享目录/文件配置失败: %v\n", err))
						deployFailed = true
						continue
					}
				}
			}

			if env.PreDeploy != "" {
				e.broadcastLog(deployment.ID, ">>> [3/5] 执行前置脚本...\n")
				deployment.SetPhase("pre_deploy")
				
				safeCmd, validErr := ValidateAndFormat(env.PreDeploy, nil)
				if validErr != nil {
					e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 前置脚本不合法 (白名单拦截): %v\n", validErr))
					deployFailed = true
					continue
				}

				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						e.broadcastLog(deployment.ID, msg)
					}
				}()
				err = e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", releaseDir, safeCmd), hookLogChan)
				close(hookLogChan)
				if err != nil {
					e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 前置脚本失败: %v\n", err))
					deployFailed = true
					continue
				}
			} else {
				e.broadcastLog(deployment.ID, ">>> [3/5] 无前置脚本，跳过。\n")
			}

			e.broadcastLog(deployment.ID, ">>> [4/5] 切换 Symlink...\n")
			deployment.SetPhase("symlink")
			tmpLink := fmt.Sprintf("%s/current_tmp_%d", env.DeployPath, time.Now().UnixNano())
			symlinkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", releaseDir, tmpLink, tmpLink, currentLink)
			symlinkLogChan := make(chan string, 50)
			go func() {
				for msg := range symlinkLogChan {
					e.broadcastLog(deployment.ID, msg)
				}
			}()
			err = e.sshClient.RunCommand(srv, symlinkCmd, symlinkLogChan)
			close(symlinkLogChan)
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: Symlink 切换失败: %v\n", err))
				deployFailed = true
				continue
			}

			if env.PostDeploy != "" {
				e.broadcastLog(deployment.ID, ">>> [5/5] 执行后置脚本...\n")
				deployment.SetPhase("post_deploy")

				safeCmd, validErr := ValidateAndFormat(env.PostDeploy, nil)
				if validErr != nil {
					e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 后置脚本不合法 (白名单拦截): %v\n", validErr))
					deployFailed = true
					continue
				}

				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						e.broadcastLog(deployment.ID, msg)
					}
				}()
				err = e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", currentLink, safeCmd), hookLogChan)
				close(hookLogChan)
				if err != nil {
					e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 后置脚本失败: %v\n", err))
					deployFailed = true
					continue
				}
			} else {
				e.broadcastLog(deployment.ID, ">>> [5/5] 无后置脚本，跳过。\n")
			}

			if project.KeepReleases > 0 {
				cleanCmd := fmt.Sprintf("cd %s/releases && ls -1dt */ 2>/dev/null | tail -n +%d | xargs rm -rf 2>/dev/null || true",
					env.DeployPath, project.KeepReleases+1)
				cleanLogChan := make(chan string, 10)
				go func() {
					for msg := range cleanLogChan {
						e.broadcastLog(deployment.ID, msg)
					}
				}()
				e.sshClient.RunCommand(srv, cleanCmd, cleanLogChan)
				close(cleanLogChan)
			}
			e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 服务器 %s 部署成功。\n", srv.Name))
		}

		if deployFailed {
			e.broadcastLog(deployment.ID, ">>> 部署完成，但部分服务器失败。\n")
			e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
		} else {
			e.broadcastLog(deployment.ID, ">>> 所有服务器部署成功！\n")
			e.deploySvc.CompleteDeploy(deployment.ID, true, e.GetLogHistory(deployment.ID), "")
		}
		time.Sleep(300 * time.Millisecond)
	}()
}

func (e *DeployEngine) Rollback(deployment *domain.Deployment, env *domain.Environment, targetRelease string) {
	go func() {
		defer e.CloseSubscribers(deployment.ID)
		e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 回滚部署 #%d -> release: %s\n", deployment.ID, targetRelease))

		for _, srvID := range env.ServerIDs {
			srv, err := e.serverSvc.GetServerByID(srvID)
			if err != nil || srv == nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 服务器 %d 未找到。\n", srvID))
				continue
			}

			e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 回滚服务器 %s...\n", srv.Name))
			releaseDir := fmt.Sprintf("%s/releases/%s", env.DeployPath, targetRelease)
			currentLink := fmt.Sprintf("%s/current", env.DeployPath)
			tmpLink := fmt.Sprintf("%s/current_tmp_%d", env.DeployPath, time.Now().UnixNano())
			symlinkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", releaseDir, tmpLink, tmpLink, currentLink)

			if e.sshClient == nil {
				e.broadcastLog(deployment.ID, "Test mode: skipping ssh symlink.\n")
				continue
			}

			rollbackLogChan := make(chan string, 50)
			go func() {
				for msg := range rollbackLogChan {
					e.broadcastLog(deployment.ID, msg)
				}
			}()
			err = e.sshClient.RunCommand(srv, symlinkCmd, rollbackLogChan)
			close(rollbackLogChan)
			if err != nil {
				e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 回滚 Symlink 切换失败: %v\n", err))
				continue
			}

			if env.PostDeploy != "" {
				e.broadcastLog(deployment.ID, ">>> 执行后置脚本...\n")
				
				safeCmd, validErr := ValidateAndFormat(env.PostDeploy, nil)
				if validErr != nil {
					e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 后置脚本不合法 (白名单拦截): %v\n", validErr))
					continue
				}

				hookLogChan := make(chan string, 50)
				go func() {
					for msg := range hookLogChan {
						e.broadcastLog(deployment.ID, msg)
					}
				}()
				e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", currentLink, safeCmd), hookLogChan)
				close(hookLogChan)
			}
			e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 服务器 %s 回滚成功。\n", srv.Name))
		}

		e.deploySvc.CompleteDeploy(deployment.ID, true, e.GetLogHistory(deployment.ID), "")
		e.broadcastLog(deployment.ID, ">>> 回滚完成。\n")
		time.Sleep(300 * time.Millisecond)
	}()
}
