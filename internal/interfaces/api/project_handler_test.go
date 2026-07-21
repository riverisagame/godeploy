package api

import (
	"bytes"
	"encoding/json"
	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"net/http"
	"net/http/httptest"
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

func TestProjectHandler_Create(t *testing.T) {
	repo := &mockProjectRepo{projects: make(map[uint]*domain.Project)}
	svc := application.NewProjectService(repo)
	handler := NewProjectHandler(svc)

	reqBody := `{"name":"test-api","repo_url":"git@test"}`
	req, _ := http.NewRequest("POST", "/api/projects", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["name"] != "test-api" {
		t.Errorf("Expected project name 'test-api', got %v", resp["name"])
	}
}
