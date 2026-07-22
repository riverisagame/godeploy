package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
)

type DeployEngine struct {
	sshClient SSHClient
	gitClient GitClient
	serverSvc *ServerService
	deploySvc *DeployService

	// Real-time log broadcasting
	subscribers map[uint][]chan string
	subMu       sync.RWMutex

	// Log history
	logHistory map[uint][]string
	historyMu  sync.RWMutex

	// Concurrency control per environment
	deployLocks map[string]*sync.Mutex
	lockMu      sync.Mutex

	cancelFuncs map[uint]context.CancelFunc
	cancelMu    sync.Mutex

	dispatcher *WebhookDispatcher
	runner     *PipelineRunner
}

func NewDeployEngine(sshClient SSHClient, gitClient GitClient, serverSvc *ServerService, deploySvc *DeployService, dispatcher *WebhookDispatcher) *DeployEngine {
	runner := NewPipelineRunner(sshClient, serverSvc)
	return &DeployEngine{
		sshClient:   sshClient,
		gitClient:   gitClient,
		serverSvc:   serverSvc,
		deploySvc:   deploySvc,
		subscribers: make(map[uint][]chan string),
		logHistory:  make(map[uint][]string),
		deployLocks: make(map[string]*sync.Mutex),
		cancelFuncs: make(map[uint]context.CancelFunc),
		dispatcher:  dispatcher,
		runner:      runner,
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

func (e *DeployEngine) CancelDeploy(deployID uint) {
	e.cancelMu.Lock()
	defer e.cancelMu.Unlock()
	if cancel, exists := e.cancelFuncs[deployID]; exists {
		cancel()
		e.Publish(deployID, "WARN: Deploy cancelled by user.\n")
	}
}

func (e *DeployEngine) StartDeploy(deployment *domain.Deployment, project *domain.Project, env *domain.Environment) {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelMu.Lock()
	e.cancelFuncs[deployment.ID] = cancel
	e.cancelMu.Unlock()

	go func() {
		defer func() {
			e.cancelMu.Lock()
			delete(e.cancelFuncs, deployment.ID)
			e.cancelMu.Unlock()
			e.CloseSubscribers(deployment.ID)
		}()
		e.runDeploySteps(ctx, deployment, project, env)
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
				_ = e.sshClient.RunCommand(srv, fmt.Sprintf("cd %s && %s", currentLink, safeCmd), hookLogChan)
				close(hookLogChan)
			}
			e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 服务器 %s 回滚成功。\n", srv.Name))
		}

		_ = e.deploySvc.CompleteDeploy(deployment.ID, true, e.GetLogHistory(deployment.ID), "")
		e.broadcastLog(deployment.ID, ">>> 回滚完成。\n")
		time.Sleep(300 * time.Millisecond)
	}()
}

func (e *DeployEngine) runDeploySteps(ctx context.Context, deployment *domain.Deployment, project *domain.Project, env *domain.Environment) {
	lockKey := fmt.Sprintf("%d-%d", project.ID, env.ID)
	lock := e.getDeployLock(lockKey)
	lock.Lock()
	defer lock.Unlock()

	// @Ref: docs/sps/plans/20260721_v2.5_refactoring_ir.md Task 3.4 | @Date: 2026-07-22
	if DeployRunningCurrent != nil {
		DeployRunningCurrent.Inc()
	}
	startTime := time.Now()
	success := false
	defer func() {
		if DeployRunningCurrent != nil {
			DeployRunningCurrent.Dec()
		}
		status := "failed"
		if success {
			status = "success"
		}
		if DeployTotal != nil {
			DeployTotal.WithLabelValues(fmt.Sprintf("%d", project.ID), status).Inc()
			DeployDuration.WithLabelValues(fmt.Sprintf("%d", project.ID), status).Observe(time.Since(startTime).Seconds())
		}

		if env.NotifyWebhook != "" && e.dispatcher != nil {
			payload, _ := json.Marshal(map[string]interface{}{
				"project": project.Name,
				"env":     env.Name,
				"status":  status,
			})
			e.dispatcher.Dispatch(WebhookEvent{URL: env.NotifyWebhook, Payload: payload})
		}
	}()

	e.broadcastLog(deployment.ID, fmt.Sprintf(">>> 开始部署任务 #%d (环境: %s)...\n", deployment.ID, env.Name))

	logChan := make(chan string, 100)
	go func() {
		for msg := range logChan {
			e.broadcastLog(deployment.ID, msg)
		}
	}()

	// 1. Clone Repo
	var workspacePath string
	var err error
	if e.gitClient != nil {
		workspacePath, err = e.gitClient.CloneForDeploy(project.RepoURL, env.Branch, project.Name, deployment.ID, logChan)
		if err != nil {
			e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: Git clone failed: %v\n", err))
			if e.deploySvc != nil {
				_ = e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
			}
			return
		}
		defer func() { _ = e.gitClient.CleanupDeploy(project.Name, deployment.ID, workspacePath) }()
	} else {
		e.broadcastLog(deployment.ID, "Test mode: skipping git clone.\n")
		workspacePath = "/tmp/test-workspace"
	}

	// 2. Load Pipeline or fallback to default
	var pipeline *domain.Pipeline
	yamlFile := fmt.Sprintf("%s/.pdeploy.yml", workspacePath)
	data, readErr := os.ReadFile(yamlFile)
	if readErr == nil {
		pipeline, err = domain.ParsePipeline(data)
		if err != nil {
			e.broadcastLog(deployment.ID, fmt.Sprintf("ERROR: 解析 .pdeploy.yml 失败: %v\n", err))
			if e.deploySvc != nil {
				_ = e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
			}
			return
		}
		e.broadcastLog(deployment.ID, ">>> 检测到 .pdeploy.yml，使用声明式流水线...\n")
	} else {
		e.broadcastLog(deployment.ID, ">>> 未检测到 .pdeploy.yml，使用默认流水线...\n")
		// @Ref: docs/sps/plans/20260722_v3.0_pipeline_ir.md | @Date: 2026-07-22
		pipeline = &domain.Pipeline{
			Stages: []string{"deploy", "post_deploy"},
			Tasks: map[string]*domain.TaskConfig{
				"sync_code": {Stage: "deploy", Type: "sync"},
			},
		}
		if env.PostDeploy != "" {
			pipeline.Tasks["post_script"] = &domain.TaskConfig{
				Stage:  "post_deploy",
				Type:   "script",
				RunOn:  "remote",
				Script: []string{env.PostDeploy},
			}
		}
	}

	if ctx.Err() != nil {
		e.broadcastLog(deployment.ID, "ERROR: Deploy cancelled before sync.\n")
		if e.deploySvc != nil {
			_ = e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
		}
		return
	}

	releaseName := fmt.Sprintf("release_%d", time.Now().UnixNano())

	// 3. Run Pipeline
	err = e.runner.Run(ctx, pipeline, workspacePath, releaseName, env, deployment.ID, func(msg string) {
		e.broadcastLog(deployment.ID, msg)
	})

	if err != nil {
		if e.deploySvc != nil {
			_ = e.deploySvc.CompleteDeploy(deployment.ID, false, e.GetLogHistory(deployment.ID), "")
		}
		return
	}

	if e.deploySvc != nil {
		_ = e.deploySvc.CompleteDeploy(deployment.ID, true, e.GetLogHistory(deployment.ID), releaseName)
	}
	e.broadcastLog(deployment.ID, ">>> 部署成功完成。\n")
	success = true
}
