package application_test

import (
	"pdeploy/internal/application"
	"pdeploy/internal/domain"
	"testing"
	"time"
)

type mockSSHClient struct {
	cmds   []string
	synced bool
}

func (m *mockSSHClient) RunCommand(srv *domain.Server, cmd string, logChan chan<- string) error {
	m.cmds = append(m.cmds, cmd)
	return nil
}
func (m *mockSSHClient) SyncFiles(server *domain.Server, localPath, remotePath, linkDest string, logChan chan<- string) error {
	m.synced = true
	return nil
}

type mockGitClient struct{}

func (m *mockGitClient) CloneForDeploy(r, b, p string, id uint, l chan<- string) (string, error) {
	return "/tmp/mock", nil
}

func (m *mockGitClient) CleanupDeploy(p string, id uint, dp string) error {
	return nil
}

func (m *mockGitClient) FetchAndGetCommits(r, b, p, f string) ([]domain.CommitInfo, error) {
	return []domain.CommitInfo{{Hash: "123"}}, nil
}

func TestDeployEngine_DependencyInversion(t *testing.T) {
	sshClient := &mockSSHClient{}
	gitClient := &mockGitClient{}
	
	// Should accept interfaces and ServerService, not concrete types or Repo directly
	_ = application.NewDeployEngine(sshClient, gitClient, nil, nil)
}

func TestDeployEngine_Subscribe_ReceivesHistory(t *testing.T) {
	// Task 1.3 RED Test
	deploySvc := application.NewDeployService(nil, nil, nil)
	engine := application.NewDeployEngine(nil, nil, nil, deploySvc)

	depID := uint(999)

	// Simulate engine publishing logs BEFORE subscriber connects
	// (Note: Since we are testing missing functionality, this might not work on the raw engine
	// without logHistory, but we can simulate the internal Publish if it was public.
	// Currently Publish is public!)
	engine.Publish(depID, "log 1\n")
	engine.Publish(depID, "log 2\n")

	// Now client connects
	ch := engine.Subscribe(depID)

	// We expect the channel to immediately receive the history
	receivedLogs := []string{}
	
	// Read with a short timeout
	done := make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case msg := <-ch:
				receivedLogs = append(receivedLogs, msg)
			case <-time.After(100 * time.Millisecond):
				done <- true
				return
			}
		}
		done <- true
	}()

	<-done

	if len(receivedLogs) != 2 {
		t.Fatalf("expected 2 historical log lines upon subscribe, got %d", len(receivedLogs))
	}
	if receivedLogs[0] != "log 1\n" || receivedLogs[1] != "log 2\n" {
		t.Errorf("history mismatch: %v", receivedLogs)
	}
}
