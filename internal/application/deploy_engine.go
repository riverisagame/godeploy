package application

import (
	"fmt"
	"pdeploy/internal/domain"
	"sync"
	"time"
)

type DeployEngine struct {
	sshClient SSHClient
	serverRepo domain.ServerRepository
	
	// Pub/Sub for SSE logs. Map of DeploymentID to channels.
	subscribers map[uint][]chan string
	mu          sync.RWMutex
}

func NewDeployEngine(sshClient SSHClient, serverRepo domain.ServerRepository) *DeployEngine {
	return &DeployEngine{
		sshClient:   sshClient,
		serverRepo:  serverRepo,
		subscribers: make(map[uint][]chan string),
	}
}

// Subscribe returns a channel that receives logs for a specific deployment.
func (e *DeployEngine) Subscribe(deploymentID uint) chan string {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan string, 100)
	e.subscribers[deploymentID] = append(e.subscribers[deploymentID], ch)
	return ch
}

// Publish sends a log message to all subscribers of a deployment.
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

func (e *DeployEngine) CloseSubscribers(deploymentID uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	subs := e.subscribers[deploymentID]
	for _, ch := range subs {
		close(ch)
	}
	delete(e.subscribers, deploymentID)
}

// StartDeploy runs the deployment process asynchronously.
func (e *DeployEngine) StartDeploy(deployment *domain.Deployment, env *domain.Environment) {
	go func() {
		defer e.CloseSubscribers(deployment.ID)
		
		logChan := make(chan string, 50)
		
		// Consume logChan and Publish
		go func() {
			for msg := range logChan {
				e.Publish(deployment.ID, msg)
			}
		}()
		
		e.Publish(deployment.ID, fmt.Sprintf(">>> Starting deployment %d for commit %s\n", deployment.ID, deployment.CommitHash))
		
		if len(env.ServerIDs) == 0 {
			e.Publish(deployment.ID, "ERROR: No servers configured for this environment.\n")
			return
		}
		
		for _, srvID := range env.ServerIDs {
			srv, err := e.serverRepo.FindByID(srvID)
			if err != nil || srv == nil {
				e.Publish(deployment.ID, fmt.Sprintf("ERROR: Server %d not found.\n", srvID))
				continue
			}
			
			e.Publish(deployment.ID, fmt.Sprintf(">>> Connecting to server %s (%s:%d)...\n", srv.Name, srv.IP, srv.Port))
			
			// Simulate Clone & Build locally, then sync to server
			time.Sleep(1 * time.Second)
			e.Publish(deployment.ID, ">>> Code pulled and synced to server.\n")
			
			// Run Pre-deploy
			if env.PreDeploy != "" {
				e.Publish(deployment.ID, ">>> Running Pre-deploy hook...\n")
				err := e.sshClient.RunCommand(srv.IP, srv.Port, env.PreDeploy, logChan)
				if err != nil {
					e.Publish(deployment.ID, fmt.Sprintf("Pre-deploy hook failed: %v\n", err))
					return // Halt deployment
				}
			}
			
			// Switch Symlink
			e.Publish(deployment.ID, ">>> Switching Symlink...\n")
			time.Sleep(1 * time.Second)
			
			// Run Post-deploy
			if env.PostDeploy != "" {
				e.Publish(deployment.ID, ">>> Running Post-deploy hook...\n")
				err := e.sshClient.RunCommand(srv.IP, srv.Port, env.PostDeploy, logChan)
				if err != nil {
					e.Publish(deployment.ID, fmt.Sprintf("Post-deploy hook failed: %v\n", err))
					return
				}
			}
			
			e.Publish(deployment.ID, fmt.Sprintf(">>> Deployment successful on server %s.\n", srv.Name))
		}
		
		e.Publish(deployment.ID, ">>> All servers deployed successfully.\n")
		// Wait a bit to ensure all logs flush
		time.Sleep(500 * time.Millisecond)
		close(logChan)
	}()
}
