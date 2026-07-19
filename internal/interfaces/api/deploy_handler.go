package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pdeploy/internal/application"
	"pdeploy/internal/domain"
	"strconv"
	"strings"
)

type DeployHandler struct {
	svc    *application.DeployService
	engine *application.DeployEngine
	prjSvc *application.ProjectService
}

func NewDeployHandler(svc *application.DeployService, engine *application.DeployEngine, prjSvc *application.ProjectService) *DeployHandler {
	return &DeployHandler{
		svc:    svc,
		engine: engine,
		prjSvc: prjSvc,
	}
}

type StartDeployReq struct {
	ProjectID  uint   `json:"project_id"`
	EnvName    string `json:"env_name"`
	CommitHash string `json:"commit_hash"`
}

func (h *DeployHandler) StartDeploy(w http.ResponseWriter, r *http.Request) {
	var req StartDeployReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In real DDD we'd have a use-case for "TriggerDeploy" that encapsulates this.
	// We'll simulate fetching the Project and Environment
	projects, err := h.prjSvc.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	var targetProject *domain.Project
	var env *domain.Environment
	for _, prj := range projects {
		if prj.ID == req.ProjectID {
			for _, e := range prj.Environments {
				if e.Name == req.EnvName {
					targetProject = prj
					env = e
					break
				}
			}
			break
		}
	}
	
	if env == nil || targetProject == nil {
		http.Error(w, "environment or project not found", http.StatusNotFound)
		return
	}

	deployment, err := h.svc.TriggerDeploy(req.ProjectID, 1, req.CommitHash) // 1 is mock UserID
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start deployment asynchronously in the engine
	h.engine.StartDeploy(deployment, targetProject, env)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(deployment)
}

// SSEEndpoint handling realtime logs
func (h *DeployHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	deployID, _ := strconv.Atoi(parts[3]) // /api/deployments/{id}/logs

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// CORS headers if needed
	
	logChan := h.engine.Subscribe(uint(deployID))
	
	// Ensure we handle client disconnect
	ctx := r.Context()
	
	for {
		select {
		case msg, ok := <-logChan:
			if !ok {
				fmt.Fprintf(w, "data: [EOF]\n\n")
				flusher.Flush()
				return
			}
			
			// Format as SSE event
			// JSON escape or just pass string since SSE handles raw strings, but multiline needs care.
			// Let's replace newlines if needed, or send line by line.
			lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
			for _, line := range lines {
				fmt.Fprintf(w, "data: %s\n\n", line)
			}
			flusher.Flush()
			
		case <-ctx.Done():
			// Client disconnected
			return
		}
	}
}

type RollbackReq struct {
	ProjectID     uint   `json:"project_id"`
	EnvName       string `json:"env_name"`
	TargetRelease string `json:"target_release"`
}

func (h *DeployHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	deployIDStr := parts[3]
	_, err := strconv.ParseUint(deployIDStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid deployment ID", http.StatusBadRequest)
		return
	}

	var req RollbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	projects, err := h.prjSvc.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var targetProject *domain.Project
	var env *domain.Environment
	for _, prj := range projects {
		if prj.ID == req.ProjectID {
			for _, e := range prj.Environments {
				if e.Name == req.EnvName {
					targetProject = prj
					env = e
					break
				}
			}
			break
		}
	}

	if env == nil || targetProject == nil {
		http.Error(w, "environment or project not found", http.StatusNotFound)
		return
	}

	// Trigger a new deployment record for the rollback
	deployment, err := h.svc.TriggerDeploy(req.ProjectID, 1, "ROLLBACK_TO_"+req.TargetRelease)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.Rollback(deployment, env, req.TargetRelease)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(deployment)
}

func (h *DeployHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	envIDStr := r.URL.Query().Get("env_id")
	if envIDStr == "" {
		http.Error(w, "env_id query parameter is required", http.StatusBadRequest)
		return
	}
	envID, err := strconv.ParseUint(envIDStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid env_id", http.StatusBadRequest)
		return
	}

	deployments, err := h.svc.GetDeploymentsByEnv(uint(envID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}
