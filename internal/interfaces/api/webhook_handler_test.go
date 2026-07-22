package api_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/riverisagame/godeploy/internal/interfaces/api"
	"github.com/stretchr/testify/assert"
)

type mockProjectRepoWebhook struct {
	projects map[uint]*domain.Project
}

func (m *mockProjectRepoWebhook) Save(p *domain.Project) error        { return nil }
func (m *mockProjectRepoWebhook) FindAll() ([]*domain.Project, error) { return nil, nil }
func (m *mockProjectRepoWebhook) FindByID(id uint) (*domain.Project, error) {
	return m.projects[id], nil
}
func (m *mockProjectRepoWebhook) Delete(id uint) error { return nil }
func (m *mockProjectRepoWebhook) FindProjectByEnvID(envID uint) (*domain.Project, error) { return nil, nil }

type mockDeployRepoWebhook struct{}

func (m *mockDeployRepoWebhook) Save(d *domain.Deployment) error { return nil }
func (m *mockDeployRepoWebhook) FindByProjectID(projectID uint) ([]*domain.Deployment, error) {
	return nil, nil
}
func (m *mockDeployRepoWebhook) FindByID(id uint) (*domain.Deployment, error) { return nil, nil }
func (m *mockDeployRepoWebhook) FindByEnvID(envID uint) ([]*domain.Deployment, error) {
	return nil, nil
}
func (m *mockDeployRepoWebhook) Update(d *domain.Deployment) error { return nil }
func (m *mockDeployRepoWebhook) FindByStatus(status string) ([]*domain.Deployment, error) { return nil, nil }

func TestWebhookHandler_HandleGitHubPush(t *testing.T) {
	// The WebhookHandler needs mock services.
	project := &domain.Project{
		ID:            1,
		Name:          "Test Project",
		WebhookSecret: "my-secret",
	}
	repo := &mockProjectRepoWebhook{projects: map[uint]*domain.Project{1: project}}
	prjSvc := application.NewProjectService(repo)
	deploySvc := application.NewDeployService(&mockDeployRepoWebhook{}, nil, nil)

	// We pass nil for DeployEngine because we won't test the actual background deploy here
	handler := api.NewWebhookHandler(prjSvc, deploySvc, nil)

	body := []byte(`{"ref":"refs/heads/main"}`)

	req := httptest.NewRequest("POST", "/api/webhook/github/1", bytes.NewBuffer(body))
	req.SetPathValue("project_id", "1")
	req.Header.Set("Content-Type", "application/json")

	// Test without signature
	rr := httptest.NewRecorder()
	handler.HandleGitHubPush(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without signature, got %d", rr.Code)
	}

	// Test with valid signature
	secret := "my-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	req = httptest.NewRequest("POST", "/api/webhook/github/1", bytes.NewBuffer(body))
	req.SetPathValue("project_id", "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	rr = httptest.NewRecorder()
	handler.HandleGitHubPush(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestWebhookHandler_GitLabAndGitee(t *testing.T) {
	project := &domain.Project{
		ID:            1,
		Name:          "Test Project",
		WebhookSecret: "my-secret",
	}
	repo := &mockProjectRepoWebhook{projects: map[uint]*domain.Project{1: project}}
	prjSvc := application.NewProjectService(repo)
	deploySvc := application.NewDeployService(&mockDeployRepoWebhook{}, nil, nil)
	handler := api.NewWebhookHandler(prjSvc, deploySvc, nil)
	body := []byte(`{"ref":"refs/heads/main"}`)

	// GitLab valid
	req := httptest.NewRequest("POST", "/api/webhook/github/1", bytes.NewBuffer(body))
	req.SetPathValue("project_id", "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitlab-Token", "my-secret")
	rr := httptest.NewRecorder()
	handler.HandleGitHubPush(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Gitee valid
	req = httptest.NewRequest("POST", "/api/webhook/github/1", bytes.NewBuffer(body))
	req.SetPathValue("project_id", "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitee-Token", "my-secret")
	rr = httptest.NewRecorder()
	handler.HandleGitHubPush(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Gitee invalid
	req = httptest.NewRequest("POST", "/api/webhook/github/1", bytes.NewBuffer(body))
	req.SetPathValue("project_id", "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitee-Token", "wrong-secret")
	rr = httptest.NewRecorder()
	handler.HandleGitHubPush(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
