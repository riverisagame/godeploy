package api

import (
	"encoding/json"
	"net/http"

	"github.com/riverisagame/godeploy/internal/application"
)

type WebhookHandler struct {
	projectSvc   *application.ProjectService
	deploySvc    *application.DeployService
	deployEngine *application.DeployEngine
}

func NewWebhookHandler(projectSvc *application.ProjectService, deploySvc *application.DeployService, deployEngine *application.DeployEngine) *WebhookHandler {
	return &WebhookHandler{
		projectSvc:   projectSvc,
		deploySvc:    deploySvc,
		deployEngine: deployEngine,
	}
}

func (h *WebhookHandler) HandleGitHubPush(w http.ResponseWriter, r *http.Request) {
	// TODO: Full webhook implementation with signature verification and auto-deploy
	// This is a stub to satisfy the router definition.
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "webhook received"})
}
