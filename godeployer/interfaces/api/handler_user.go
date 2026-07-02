// ============================================================
// 文件：handler_user.go
// 作用：👤 用户管理 API——增删改查用户！
//
// 这个文件提供管理员操作用户的功能：
// - HandleGetUsers：获取所有用户列表
// - HandleCreateUser：创建新用户
// - HandleUpdateUser：更新用户信息
// - HandleDeleteUser：删除用户
// - HandleGetUserGitBinding：查看用户的 Git 作者绑定
// - HandleUpdateUserGitBinding：更新 Git 作者绑定
//
// 注意：这些 API 只有 admin 角色才能调用！
// ============================================================

package api

import (
	"database/sql"
	"deploy/godeployer/application"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 🔗 Git 作者绑定
//
// 可以限制用户只能部署"自己提交的代码"。
// 比如：张三的代码只能由张三来部署。
// 这是一种安全策略——防止随意部署别人的代码。
// ============================================================

// UpdateGitBindingRequest Git 作者绑定的请求体
type UpdateGitBindingRequest struct {
	BoundGitAuthors    string `json:"bound_git_authors"`    // 允许的 Git 作者（逗号分隔）
	RestrictGitAuthors bool   `json:"restrict_git_authors"` // 是否开启限制
}

// HandleGetUserGitBinding 获取用户的 Git 作者绑定配置
func (h *APIHandler) HandleGetUserGitBinding(c *gin.Context) {
	username := c.Param("username")
	var req UpdateGitBindingRequest
	err := h.db.QueryRow(
		"SELECT bound_git_authors, restrict_git_authors FROM users WHERE username = ?", username,
	).Scan(&req.BoundGitAuthors, &req.RestrictGitAuthors)
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

// HandleUpdateUserGitBinding 更新用户的 Git 作者绑定
func (h *APIHandler) HandleUpdateUserGitBinding(c *gin.Context) {
	username := c.Param("username")
	var req UpdateGitBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.db.Exec(
		"UPDATE users SET bound_git_authors = ?, restrict_git_authors = ? WHERE username = ?",
		req.BoundGitAuthors, req.RestrictGitAuthors, username,
	)
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

// ============================================================
// 👥 用户 CRUD（增删改查）
// ============================================================

// UserResponse 返回给前端的用户信息（不含密码哈希！）
type UserResponse struct {
	ID                 int       `json:"id"`
	Username           string    `json:"username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	BoundGitAuthors    string    `json:"bound_git_authors"`
	RestrictGitAuthors bool      `json:"restrict_git_authors"`
	PermittedProjects  string    `json:"permitted_projects"`
}

// HandleGetUsers 获取所有用户列表（管理员查看用户管理页面时调用）
func (h *APIHandler) HandleGetUsers(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, username, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects FROM users",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query users"})
		return
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt,
			&u.BoundGitAuthors, &u.RestrictGitAuthors, &u.PermittedProjects); err == nil {
			users = append(users, u)
		}
	}
	c.JSON(http.StatusOK, users)
}

// CreateUserRequest 创建用户的请求体
type CreateUserRequest struct {
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
	Role               string `json:"role" binding:"required"`
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
	PermittedProjects  string `json:"permitted_projects"`
}

// HandleCreateUser 创建新用户
func (h *APIHandler) HandleCreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 对密码进行哈希加密（绝不存明文！）
	hash, err := application.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 如果没指定权限，默认给全部项目
	if req.PermittedProjects == "" {
		req.PermittedProjects = "*"
	}

	_, err = h.db.Exec(
		"INSERT INTO users (username, password_hash, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Username, hash, req.Role, time.Now(), req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user, username might exist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

// UpdateUserRequest 更新用户的请求体
type UpdateUserRequest struct {
	Password           string `json:"password"`
	Role               string `json:"role" binding:"required"`
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
	PermittedProjects  string `json:"permitted_projects"`
}

// HandleUpdateUser 更新用户信息（角色、密码、权限等）
func (h *APIHandler) HandleUpdateUser(c *gin.Context) {
	username := c.Param("username")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Password != "" {
		// 如果传了新密码，一起更新
		hash, _ := application.HashPassword(req.Password)
		_, err := h.db.Exec(
			"UPDATE users SET password_hash = ?, role = ?, bound_git_authors = ?, restrict_git_authors = ?, permitted_projects = ? WHERE username = ?",
			hash, req.Role, req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects, username,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	} else {
		// 不更新密码
		_, err := h.db.Exec(
			"UPDATE users SET role = ?, bound_git_authors = ?, restrict_git_authors = ?, permitted_projects = ? WHERE username = ?",
			req.Role, req.BoundGitAuthors, req.RestrictGitAuthors, req.PermittedProjects, username,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

// HandleDeleteUser 删除用户（不能删除默认的 admin 账号）
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

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 为什么不能删除 admin 用户？
//    A: admin 是系统默认的管理员账号，删了就没法管理了！
//       就像不能删除 Windows 的 Administrator 账号一样~
//
// 中级：
// 2. Q: 为什么更新密码时分开处理（有密码/没密码）？
//    A: 防止默认值覆盖！如果用户只想改角色不改密码，
//       传空字符串时不会更新密码字段~
//
// 高级：
// 3. Q: Git 作者绑定解决了什么问题？
//    A: 审计安全！只允许用户部署自己提交的代码，
//       防止张三部署李四的代码导致责任不清~
// ============================================================
