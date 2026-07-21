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

func (m *mockProjectRepo) Delete(id uint) error {
	delete(m.projects, id)
	return nil
}

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

func TestProjectService_UpdateProject(t *testing.T) {
	// [RED] Edge Cases Test for UpdateProject
	repo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	svc := NewProjectService(repo)
	
	p, _ := svc.CreateProject("old-name", "git@github.com:test/old.git")
	
	p2, err := svc.UpdateProject(p.ID, "new-name", "git@github.com:test/new.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Name != "new-name" || p2.RepoURL != "git@github.com:test/new.git" {
		t.Errorf("expected project to be updated, got: %v", p2)
	}
	
	// Edge Case: Invalid project
	_, err = svc.UpdateProject(999, "name", "repo")
	if err == nil {
		t.Errorf("expected error for non-existent project")
	}
}

func TestProjectService_DeleteProject(t *testing.T) {
	// [RED] Edge Cases Test for DeleteProject
	repo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	svc := NewProjectService(repo)
	
	p, _ := svc.CreateProject("to-delete", "git@github.com:test/del.git")
	err := svc.DeleteProject(p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	pFound, _ := repo.FindByID(p.ID)
	if pFound != nil {
		t.Errorf("expected project to be deleted from repo")
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
	
	p3, err := svc.UpdateEnvironment(p.ID, "prod", "echo 'pre'", "echo 'post'", "/var/www/prod", []uint{1, 2}, nil)
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
