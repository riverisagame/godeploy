package application

import (
	"pdeploy/internal/domain"
	"testing"
)

type mockProjectRepo struct {
	projects map[uint]*domain.Project
	lastID   uint
}

func (m *mockProjectRepo) Save(p *domain.Project) error {
	if p.ID == 0 {
		m.lastID++
		p.ID = m.lastID
	}
	m.projects[p.ID] = p
	return nil
}
func (m *mockProjectRepo) FindByID(id uint) (*domain.Project, error) { return m.projects[id], nil }
func (m *mockProjectRepo) FindAll() ([]*domain.Project, error)       { return nil, nil }

func TestProjectService_CreateProject(t *testing.T) {
	repo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	svc := NewProjectService(repo)

	p, err := svc.CreateProject("test-proj", "git@github.com:test/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID == 0 {
		t.Errorf("expected project ID to be set")
	}

	if p.Name != "test-proj" {
		t.Errorf("expected name test-proj, got %s", p.Name)
	}
}

func TestProjectService_AddAndUpdateEnvironment(t *testing.T) {
	repo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	svc := NewProjectService(repo)

	p, _ := svc.CreateProject("test-proj", "git@github.com:test/repo.git")
	
	p2, err := svc.AddEnvironment(p.ID, "prod", "main", "symlink")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(p2.Environments) != 1 {
		t.Fatalf("expected 1 environment")
	}
	
	p3, err := svc.UpdateEnvironment(p.ID, "prod", "echo 'pre'", "echo 'post'", "/var/www/prod", []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if p3.Environments[0].PreDeploy != "echo 'pre'" {
		t.Errorf("expected pre-deploy hook to be updated")
	}
	if p3.Environments[0].DeployPath != "/var/www/prod" {
		t.Errorf("expected deploy path to be updated")
	}
	if len(p3.Environments[0].ServerIDs) != 2 {
		t.Errorf("expected 2 server IDs")
	}
}
