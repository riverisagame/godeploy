package application

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
)

type PipelineRunner struct {
	sshClient SSHClient
	serverSvc *ServerService
}

func NewPipelineRunner(sshClient SSHClient, serverSvc *ServerService) *PipelineRunner {
	return &PipelineRunner{
		sshClient: sshClient,
		serverSvc: serverSvc,
	}
}

func (r *PipelineRunner) Run(ctx context.Context, p *domain.Pipeline, workspacePath, releaseName string, env *domain.Environment, deployID uint, logger func(string)) error {
	for _, stage := range p.Stages {
		logger(fmt.Sprintf(">>> [STAGE] %s\n", stage))
		var tasks []string
		for tName, tConfig := range p.Tasks {
			if tConfig.Stage == stage {
				tasks = append(tasks, tName)
			}
		}
		sort.Strings(tasks)

		for _, tName := range tasks {
			tConfig := p.Tasks[tName]
			if err := r.runTask(ctx, tName, tConfig, workspacePath, releaseName, env, deployID, logger); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PipelineRunner) runTask(ctx context.Context, taskName string, config *domain.TaskConfig, workspacePath, releaseName string, env *domain.Environment, deployID uint, logger func(string)) error {
	logger(fmt.Sprintf("    -> [TASK] %s\n", taskName))

	if config.Type == "sync" {
		return r.handleSyncTask(ctx, workspacePath, releaseName, env, logger)
	}

	if config.Type == "script" {
		if config.RunOn == "local" {
			return r.handleLocalScript(ctx, config, workspacePath, logger)
		} else if config.RunOn == "remote" {
			return r.handleRemoteScript(ctx, config, releaseName, env, logger)
		}
	}

	logger(fmt.Sprintf("    -> WARN: Unknown task type/run_on combo: type=%s, run_on=%s\n", config.Type, config.RunOn))
	return nil
}

func (r *PipelineRunner) handleSyncTask(ctx context.Context, workspacePath, releaseName string, env *domain.Environment, logger func(string)) error {
	logChan := make(chan string, 100)
	defer close(logChan)
	go func() {
		for msg := range logChan {
			logger(msg)
		}
	}()

	for _, srvID := range env.ServerIDs {
		var srv *domain.Server
		var err error
		if r.serverSvc != nil {
			srv, err = r.serverSvc.GetServerByID(srvID)
			if err != nil || srv == nil {
				logger(fmt.Sprintf("ERROR: Server %d not found.\n", srvID))
				continue
			}
		} else {
			srv = &domain.Server{ID: srvID, Name: fmt.Sprintf("TestServer-%d", srvID)}
		}

		logger(fmt.Sprintf(">>> 同步代码到服务器 %s...\n", srv.Name))
		remoteReleasePath := fmt.Sprintf("%s/releases/%s", env.DeployPath, releaseName)

		if r.sshClient != nil {
			err = r.sshClient.SyncFiles(srv, workspacePath, remoteReleasePath, "", logChan)
			if err != nil {
				logger(fmt.Sprintf("ERROR: Sync failed: %v\n", err))
				continue
			}

			// Handle symlink
			if env.DeployType == "symlink" {
				currentLink := fmt.Sprintf("%s/current", env.DeployPath)
				tmpLink := fmt.Sprintf("%s/current_tmp_%d", env.DeployPath, time.Now().UnixNano())
				symlinkCmd := fmt.Sprintf("ln -sfn %s %s && mv -Tf %s %s", remoteReleasePath, tmpLink, tmpLink, currentLink)
				_ = r.sshClient.RunCommand(srv, symlinkCmd, logChan)
			}
		} else {
			logger("Test mode: skipping ssh sync.\n")
		}
	}
	return nil
}

func (r *PipelineRunner) handleLocalScript(ctx context.Context, config *domain.TaskConfig, workspacePath string, logger func(string)) error {
	for _, cmdStr := range config.Script {
		logger(fmt.Sprintf("      $ %s\n", cmdStr))
		
		// Basic parsing
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}
		
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		cmd.Dir = workspacePath
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			logger(string(out) + "\n")
		}
		if err != nil {
			logger(fmt.Sprintf("ERROR: Local script failed: %v\n", err))
			return err
		}
	}
	return nil
}

func (r *PipelineRunner) handleRemoteScript(ctx context.Context, config *domain.TaskConfig, releaseName string, env *domain.Environment, logger func(string)) error {
	logChan := make(chan string, 100)
	defer close(logChan)
	go func() {
		for msg := range logChan {
			logger(msg)
		}
	}()

	currentLink := fmt.Sprintf("%s/current", env.DeployPath)

	for _, srvID := range env.ServerIDs {
		var srv *domain.Server
		var err error
		if r.serverSvc != nil {
			srv, err = r.serverSvc.GetServerByID(srvID)
			if err != nil || srv == nil {
				continue
			}
		} else {
			srv = &domain.Server{ID: srvID, Name: fmt.Sprintf("TestServer-%d", srvID)}
		}

		if r.sshClient == nil {
			logger("Test mode: skipping remote script.\n")
			continue
		}

		for _, cmdStr := range config.Script {
			logger(fmt.Sprintf("      [%s] $ %s\n", srv.Name, cmdStr))
			
			// Always run in current dir context
			fullCmd := fmt.Sprintf("cd %s && %s", currentLink, cmdStr)
			err = r.sshClient.RunCommand(srv, fullCmd, logChan)
			if err != nil {
				logger(fmt.Sprintf("ERROR: Remote script failed: %v\n", err))
				return err
			}
		}
	}
	return nil
}
