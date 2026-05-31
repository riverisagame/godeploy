package api

import (
	"database/sql"
	"deploy/godeployer/application"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)


func (h *APIHandler) HandleGetUserGitBinding(c *gin.Context) {
	username := c.Param("username")
	var req UpdateGitBindingRequest
	err := h.db.QueryRow("SELECT bound_git_authors, restrict_git_authors FROM users WHERE username = ?", username).Scan(&req.BoundGitAuthors, &req.RestrictGitAuthors)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		}
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *APIHandler) HandleUpdateUserGitBinding(c *gin.Context) {
	username := c.Param("username")
	var req UpdateGitBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.db.Exec("UPDATE users SET bound_git_authors = ?, restrict_git_authors = ? WHERE username = ?", req.BoundGitAuthors, req.RestrictGitAuthors, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type UserResponse struct {
	ID                 int       `json:"id"`
	Username           string    `json:"username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	BoundGitAuthors    string    `json:"bound_git_authors"`
	RestrictGitAuthors bool      `json:"restrict_git_authors"`
	PermittedProjects  string    `json:"permitted_projects"`
}

func (h *APIHandler) HandleGetUsers(c *gin.Context) {
	rows, err := h.db.Query("SELECT id, username, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query users"})
		return
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.BoundGitAuthors, &u.RestrictGitAuthors, &u.PermittedProjects); err == nil {
			users = append(users, u)
		}
	}
	c.JSON(http.StatusOK, users)
}

type CreateUserRequest struct {
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
	Role               string `json:"role" binding:"required"`
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
	PermittedProjects  string `json:"permitted_projects"`
}

func (h *APIHandler) HandleCreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := application.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if req.PermittedProjects == "" {
		req.PermittedProjects = "*"
	}

	_, err = h.db.Exec("INSERT INTO users (username, password_hash, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Username, hash, req.Role, time.Now(), req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user, username might exist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

type UpdateUserRequest struct {
	Password           string `json:"password"`
	Role               string `json:"role" binding:"required"`
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
	PermittedProjects  string `json:"permitted_projects"`
}

func (h *APIHandler) HandleUpdateUser(c *gin.Context) {
	username := c.Param("username")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Password != "" {
		hash, _ := application.HashPassword(req.Password)
		_, err := h.db.Exec("UPDATE users SET password_hash = ?, role = ?, bound_git_authors = ?, restrict_git_authors = ?, permitted_projects = ? WHERE username = ?",
			hash, req.Role, req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects, username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	} else {
		_, err := h.db.Exec("UPDATE users SET role = ?, bound_git_authors = ?, restrict_git_authors = ?, permitted_projects = ? WHERE username = ?",
			req.Role, req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects, username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (h *APIHandler) HandleDeleteUser(c *gin.Context) {
	username := c.Param("username")
	if username == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete default admin"})
		return
	}
	_, err := h.db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}
