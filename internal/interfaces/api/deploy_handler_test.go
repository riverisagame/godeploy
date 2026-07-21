package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"pdeploy/internal/application"
	"pdeploy/internal/domain"
	"pdeploy/internal/interfaces/api"
	"testing"
)

type mockDeployRepo struct {
	savedDeployments []*domain.Deployment
}

func (m *mockDeployRepo) Save(d *domain.Deployment) error {
	m.savedDeployments = append(m.savedDeployments, d)
	d.ID = uint(len(m.savedDeployments))
	return nil
}
func (m *mockDeployRepo) FindByID(id uint) (*domain.Deployment, error) { return nil, nil }
func (m *mockDeployRepo) FindByEnvID(envID uint) ([]*domain.Deployment, error) { return nil, nil }
func (m *mockDeployRepo) Update(d *domain.Deployment) error { return nil }

type mockProjectRepo struct{}
func (m *mockProjectRepo) Save(p *domain.Project) error { return nil }
func (m *mockProjectRepo) FindByID(id uint) (*domain.Project, error) { return nil, nil }
func (m *mockProjectRepo) FindAll() ([]*domain.Project, error) {
	return []*domain.Project{
		{
			ID: 100, // ProjectID = 100
			Environments: []*domain.Environment{
				{
					ID: 200, // EnvID = 200
					Name: "prod",
				},
			},
		},
	}, nil
}
func (m *mockProjectRepo) Delete(id uint) error { return nil }

func TestDeployHandler_StartDeploy_UsesEnvID(t *testing.T) {
	deployRepo := &mockDeployRepo{}
	prjRepo := &mockProjectRepo{}
	deploySvc := application.NewDeployService(deployRepo, nil, nil)
	prjSvc := application.NewProjectService(prjRepo)

	// deployEngine is not mocked easily here since it's a concrete struct,
	// but we just pass nil or uninitialized because StartDeploy happens asynchronously
	// and we only care about what TriggerDeploy saves to the repository before calling engine.
	// We'll skip the engine start by letting it panic in goroutine or just ignoring it for now.
	// Actually, StartDeploy calls h.engine.StartDeploy which will panic if engine is nil.
	// So we create a dummy engine.
	engine := application.NewDeployEngine(nil, nil, nil, deploySvc)

	handler := api.NewDeployHandler(deploySvc, engine, prjSvc)

	reqBody := map[string]interface{}{
		"project_id":  100,
		"env_name":    "prod",
		"commit_hash": "abcdef",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	ctx := context.WithValue(req.Context(), api.ContextKeyUserID, float64(123))
	ctx = context.WithValue(ctx, api.ContextKeyRole, "admin")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.StartDeploy(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	if len(deployRepo.savedDeployments) == 0 {
		t.Fatalf("expected a deployment to be saved")
	}

	saved := deployRepo.savedDeployments[0]
	if saved.EnvID != 200 {
		t.Errorf("expected EnvID to be 200, but got %d (probably ProjectID was used instead)", saved.EnvID)
	}
}
