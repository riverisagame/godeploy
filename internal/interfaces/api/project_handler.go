package api

import (
	"encoding/json"
	"net/http"
	"pdeploy/internal/application"
	"pdeploy/internal/domain"
	"strconv"
	"strings"
)

type ProjectHandler struct {
	svc *application.ProjectService
}

func NewProjectHandler(svc *application.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type CreateProjectReq struct {
	Name    string `json:"name"`
	RepoURL string `json:"repo_url"`
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := h.svc.CreateProject(req.Name, req.RepoURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, p)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if projects == nil {
		projects = make([]*domain.Project, 0) // return [] instead of null
	}

	w.Header().Set("Content-Type", "application/json")
	RespondJSON(w, projects)
}

type AddEnvReq struct {
	Name       string `json:"name"`
	Branch     string `json:"branch"`
	DeployType string `json:"deploy_type"`
}

func (h *ProjectHandler) AddEnvironment(w http.ResponseWriter, r *http.Request) {
	// Parse ID from URL path: /api/projects/{id}/environments
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(parts[3])

	var req AddEnvReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := h.svc.AddEnvironment(uint(id), req.Name, req.Branch, req.DeployType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, p)
}

type UpdateEnvReq struct {
	PreDeploy  string `json:"pre_deploy"`
	PostDeploy string `json:"post_deploy"`
	ServerIDs  []uint          `json:"server_ids"`
	DeployPath string          `json:"deploy_path"`
	EnvVars    []domain.EnvVar `json:"env_vars"`
}

func (h *ProjectHandler) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	// Parse URL path: /api/projects/{id}/environments/{name}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(parts[3])
	envName := parts[5]

	var req UpdateEnvReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := h.svc.UpdateEnvironment(uint(id), envName, req.PreDeploy, req.PostDeploy, req.DeployPath, req.ServerIDs, req.EnvVars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	RespondJSON(w, p)
}
