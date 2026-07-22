package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/riverisagame/godeploy/internal/application"
)

type AuditHandler struct {
	svc application.AuditService
}

func NewAuditHandler(svc application.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 10
	}

	logs, total, err := h.svc.GetLogs(r.Context(), page, pageSize)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve audit logs")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  logs,
		"total": total,
	})
}
