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
	
	var env *domain.Environment
	for _, prj := range projects {
		if prj.ID == req.ProjectID {
			for _, e := range prj.Environments {
				if e.Name == req.EnvName {
					env = e
					break
				}
			}
			break
		}
	}
	
	if env == nil {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	deployment, err := h.svc.TriggerDeploy(req.ProjectID, 1, req.CommitHash) // 1 is mock UserID
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start deployment asynchronously in the engine
	h.engine.StartDeploy(deployment, env)

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
