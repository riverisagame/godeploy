package api

import (
	"deploy/godeployer/domain"
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// checkProjectAccess checks if a user is permitted to access a specific project.
func (h *APIHandler) checkProjectAccess(username string, targetProjectID string) bool {
	var permittedProjectsStr string
	err := h.db.QueryRow("SELECT COALESCE(permitted_projects, '*') FROM users WHERE username = ?", username).Scan(&permittedProjectsStr)
	if err != nil {
		return false
	}

	permittedList := strings.Split(permittedProjectsStr, ",")
	for _, p := range permittedList {
		p = strings.TrimSpace(p)
		if p == "*" || p == targetProjectID {
			return true
		}
	}
	return false
}

func (h *APIHandler) HandleGetProjects(c *gin.Context) {
	usernameVal, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	username := usernameVal.(string)

	var permittedProjectsStr string
	err := h.db.QueryRow("SELECT COALESCE(permitted_projects, '*') FROM users WHERE username = ?", username).Scan(&permittedProjectsStr)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user permissions"})
		return
	}

	permittedList := strings.Split(permittedProjectsStr, ",")
	permittedMap := make(map[string]bool)
	for _, p := range permittedList {
		p = strings.TrimSpace(p)
		if p != "" {
			permittedMap[p] = true
		}
	}

	projects := make([]domain.ProjectConfig, 0)
	for _, p := range h.config.Projects {
		if permittedMap["*"] || permittedMap[p.ID] {
			projects = append(projects, p)
		}
	}
	c.JSON(http.StatusOK, projects)
}

type UpdatePermissionsRequest struct {
	PermittedProjects string `json:"permitted_projects"`
}

func (h *APIHandler) HandleUpdateUserPermissions(c *gin.Context) {
	username := c.Param("username")
	var req UpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count); err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	_, err := h.db.Exec("UPDATE users SET permitted_projects = ? WHERE username = ?", req.PermittedProjects, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "permissions updated successfully"})
}
