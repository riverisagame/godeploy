package application

import (
	"context"
	"fmt"
	"github.com/riverisagame/godeploy/internal/domain"
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

	cancelFuncs map[uint]context.CancelFunc
	cancelMu    sync.Mutex
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
		cancelFuncs: make(map[uint]context.CancelFunc),
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
