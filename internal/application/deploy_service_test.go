package application

import (
	"pdeploy/internal/domain"
	"testing"
)

type mockDeployRepo struct {
	deployments map[uint]*domain.Deployment
	lastID      uint
}

func (m *mockDeployRepo) Save(d *domain.Deployment) error {
	if d.ID == 0 {
		m.lastID++
		d.ID = m.lastID
	}
	m.deployments[d.ID] = d
	return nil
}

func (m *mockDeployRepo) FindByID(id uint) (*domain.Deployment, error) {
	if d, ok := m.deployments[id]; ok {
		return d, nil
	}
	return nil, nil
}

func (m *mockDeployRepo) FindByEnvID(envID uint) ([]*domain.Deployment, error) {
	var res []*domain.Deployment
	for _, d := range m.deployments {
		if d.EnvID == envID {
			res = append(res, d)
		}
	}
	return res, nil
}

type mockDeployProjectRepo struct {
	projects map[uint]*domain.Project
}
func (m *mockDeployProjectRepo) Save(p *domain.Project) error { return nil }
func (m *mockDeployProjectRepo) FindByID(id uint) (*domain.Project, error) { return m.projects[id], nil }
func (m *mockDeployProjectRepo) FindAll() ([]*domain.Project, error) { return nil, nil }

type mockGitClient struct{}
func (m *mockGitClient) CloneOrPull(r, b, p string, l chan<- string) (string, error) { return "", nil }
func (m *mockGitClient) FetchAndGetCommits(r, b, p, f string) ([]domain.CommitInfo, error) {
	return []domain.CommitInfo{{Hash: "123"}}, nil
}

func TestDeployService_TriggerDeploy(t *testing.T) {
	repo := &mockDeployRepo{deployments: make(map[uint]*domain.Deployment)}
	projRepo := &mockDeployProjectRepo{projects: make(map[uint]*domain.Project)}
	gitClient := &mockGitClient{}
	svc := NewDeployService(repo, projRepo, gitClient)

	// Test invalid input
	_, err := svc.TriggerDeploy(0, 1, "hash123")
	if err == nil {
		t.Error("Expected error for empty envID, got nil")
	}

	// Test valid input
	d, err := svc.TriggerDeploy(1, 1, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if d.ID == 0 {
		t.Error("Expected deployment ID to be assigned")
	}
	if d.Status != "pending" {
		t.Error("Expected deployment status to be pending")
	}
}

func TestDeployService_CompleteDeploy(t *testing.T) {
	repo := &mockDeployRepo{deployments: make(map[uint]*domain.Deployment)}
	projRepo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	gitClient := &mockGitClient{}
	svc := NewDeployService(repo, projRepo, gitClient)
	
	d, _ := svc.TriggerDeploy(1, 1, "hash123")
	
	err := svc.CompleteDeploy(d.ID, true, "deployed successfully", "release_v1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	saved, _ := repo.FindByID(d.ID)
	if saved.Status != "success" {
		t.Errorf("Expected status success, got %s", saved.Status)
	}
	if saved.Log != "deployed successfully" {
		t.Errorf("Expected log 'deployed successfully', got '%s'", saved.Log)
	}
	if saved.ReleaseName != "release_v1" {
		t.Errorf("Expected release name 'release_v1', got '%s'", saved.ReleaseName)
	}
}
