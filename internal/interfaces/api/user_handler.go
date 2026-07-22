package api

import (
	"encoding/json"
	"net/http"

	"github.com/riverisagame/godeploy/internal/application"
)

type UserHandler struct {
	userSvc *application.UserService
}

func NewUserHandler(userSvc *application.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// @Ref: docs/sps/plans/20260721_v2.5_refactoring_ir.md Task 3.2 | @Date: 2026-07-22
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		RespondError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	if req.Role == "" {
		req.Role = "developer" // Default role
	}

	if err := h.userSvc.CreateUser(req.Username, req.Password, req.Role); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, map[string]string{"status": "created"})
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userSvc.ListUsers()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	RespondJSON(w, users)
}
