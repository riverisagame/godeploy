package application_test

import (
	"context"
	"testing"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPipelineRunner_Run_Success(t *testing.T) {
	// Setup mocks
	sshClient := &mockSSHClient{}
	
	// Server service with dummy repo
	serverRepo := &mockServerRepo{servers: map[uint]*domain.Server{
		1: {ID: 1, Name: "Test Server"},
	}}
	serverSvc := application.NewServerService(serverRepo, nil)
	
	runner := application.NewPipelineRunner(sshClient, serverSvc)

	p := &domain.Pipeline{
		Version: "1.0",
		Stages:  []string{"deploy"},
		Tasks: map[string]*domain.TaskConfig{
			"sync_code": {
				Stage: "deploy",
				Type:  "sync",
			},
		},
	}
	
	env := &domain.Environment{
		ID: 1,
		Name: "prod",
		ServerIDs: []uint{1},
		DeployPath: "/var/www/html",
	}

	var logs []string
	logger := func(msg string) {
		logs = append(logs, msg)
	}

	err := runner.Run(context.Background(), p, "/tmp/workspace", "release_123", env, 1, logger)
	
	assert.NoError(t, err)
	assert.True(t, sshClient.synced)
	assert.NotEmpty(t, logs)
}

type mockServerRepo struct {
	servers map[uint]*domain.Server
}
func (m *mockServerRepo) Save(s *domain.Server) error { return nil }
func (m *mockServerRepo) FindAll() ([]*domain.Server, error) { return nil, nil }
func (m *mockServerRepo) FindByID(id uint) (*domain.Server, error) {
	return m.servers[id], nil
}
func (m *mockServerRepo) FindByProjectID(pid uint) ([]*domain.Server, error) { return nil, nil }
func (m *mockServerRepo) Delete(id uint) error { return nil }
func (m *mockServerRepo) Update(s *domain.Server) error { return nil }
