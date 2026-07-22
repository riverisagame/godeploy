package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/stretchr/testify/assert"
)

type schedulerMockDeployRepo struct {
	deployments map[uint]*domain.Deployment
}
func (m *schedulerMockDeployRepo) Save(d *domain.Deployment) error { 
	m.deployments[d.ID] = d
	return nil 
}
func (m *schedulerMockDeployRepo) FindByID(id uint) (*domain.Deployment, error) {
	return m.deployments[id], nil
}
func (m *schedulerMockDeployRepo) FindByProjectID(pid uint) ([]*domain.Deployment, error) { return nil, nil }
func (m *schedulerMockDeployRepo) FindByEnvID(envID uint) ([]*domain.Deployment, error) { return nil, nil }
func (m *schedulerMockDeployRepo) FindProjectByEnvID(envID uint) (*domain.Project, error) { return nil, nil }
func (m *schedulerMockDeployRepo) FindAll() ([]*domain.Project, error) { return nil, nil }
func (m *schedulerMockDeployRepo) Delete(id uint) error { return nil }
func (m *schedulerMockDeployRepo) FindByStatus(status string) ([]*domain.Deployment, error) {
	var res []*domain.Deployment
	for _, d := range m.deployments {
		if d.Status == status {
			res = append(res, d)
		}
	}
	return res, nil
}

type schedulerMockProjectRepo struct {}
func (m *schedulerMockProjectRepo) Save(p *domain.Project) error { return nil }
func (m *schedulerMockProjectRepo) FindByID(id uint) (*domain.Project, error) { return nil, nil }
func (m *schedulerMockProjectRepo) FindAll() ([]*domain.Project, error) { return nil, nil }
func (m *schedulerMockProjectRepo) Delete(id uint) error { return nil }
func (m *schedulerMockProjectRepo) FindProjectByEnvID(envID uint) (*domain.Project, error) { return nil, nil }

func TestDeployScheduler_Recover(t *testing.T) {
	mockRepo := &schedulerMockDeployRepo{
		deployments: map[uint]*domain.Deployment{
			1: {ID: 1, Status: "running"},
			2: {ID: 2, Status: "running"},
			3: {ID: 3, Status: "success"},
		},
	}
	mockProjRepo := &schedulerMockProjectRepo{}
	scheduler := application.NewDeployScheduler(mockRepo, mockProjRepo, nil)

	scheduler.Recover()

	d1, _ := mockRepo.FindByID(1)
	assert.Equal(t, "failed", d1.Status)
	assert.Contains(t, d1.Log, "crashed")

	d2, _ := mockRepo.FindByID(2)
	assert.Equal(t, "failed", d2.Status)

	d3, _ := mockRepo.FindByID(3)
	assert.Equal(t, "success", d3.Status)
}

func TestDeployScheduler_Queue(t *testing.T) {
	mockRepo := &schedulerMockDeployRepo{
		deployments: make(map[uint]*domain.Deployment),
	}
	
	// Create a dummy deployment and project
	d := &domain.Deployment{ID: 10, EnvID: 20}
	mockRepo.deployments[10] = d
	
	mockProjRepo := &schedulerMockProjectRepo{}
	scheduler := application.NewDeployScheduler(mockRepo, mockProjRepo, nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)
	
	// Notify queue
	scheduler.Notify(10)
	
	// Wait a bit for goroutine
	time.Sleep(50 * time.Millisecond)
	cancel() // Stop the scheduler
	
	// Since projectRepo (mockRepo) returns nil project in our stub, it should mark as failed
	assert.Equal(t, "failed", d.Status)
	assert.Contains(t, d.Log, "project or environment not found")
}
