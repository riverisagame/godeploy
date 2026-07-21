package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
)

type ServerHandler struct {
	svc *application.ServerService
}

func NewServerHandler(svc *application.ServerService) *ServerHandler {
	return &ServerHandler{svc: svc}
}

type CreateServerReq struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	User    string `json:"user"`
	KeyPath string `json:"key_path"`
}

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.svc.ListServers()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if servers == nil {
		servers = make([]*domain.Server, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	RespondJSON(w, servers)
}

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateServerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s, err := h.svc.CreateServer(req.Name, req.IP, req.Port, req.User, req.KeyPath)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, s)
}

func (h *ServerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		RespondError(w, http.StatusBadRequest, "missing server id")
		return
	}

	var id uint
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "invalid server id")
		return
	}

	if err := h.svc.DeleteServer(id); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Ref: docs/sps/plans/20260721_project_server_edit_ir.md | @Date: 2026-07-21
func (h *ServerHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "missing server id", http.StatusBadRequest)
		return
	}
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "invalid server id", http.StatusBadRequest)
		return
	}

	var req CreateServerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s, err := h.svc.UpdateServer(id, req.Name, req.IP, req.Port, req.User, req.KeyPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	RespondJSON(w, s)
}
