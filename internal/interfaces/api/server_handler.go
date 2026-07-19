package api

import (
	"encoding/json"
	"net/http"
	"pdeploy/internal/domain"
)

type ServerHandler struct {
	repo domain.ServerRepository
}

func NewServerHandler(repo domain.ServerRepository) *ServerHandler {
	return &ServerHandler{repo: repo}
}

type CreateServerReq struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.repo.FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if servers == nil {
		servers = make([]*domain.Server, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateServerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s, err := domain.NewServer(req.Name, req.IP, req.Port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Save(s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}
