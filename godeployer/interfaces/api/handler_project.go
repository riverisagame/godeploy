// ============================================================
// 文件：handler_project.go
// 作用：📁 项目相关的 API 处理函数！
//
// 这个文件处理"查看项目列表"和"检查权限"的逻辑。
//
// 核心功能：
// 1. checkProjectAccess：检查用户有没有权限操作某个项目
// 2. HandleGetProjects：获取用户能看到的所有项目列表
// 3. HandleUpdateUserPermissions：管理员修改用户的权限
// ============================================================

package api

import (
	"deploy/godeployer/domain"
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 🔐 checkProjectAccess：权限检查工具函数
//
// 每个用户都有一个"permitted_projects"字段，格式是逗号分隔的项目 ID。
// 比如 "myblog,thinkphp" 表示只能操作这两个项目。
// 如果是 "*" 表示所有项目都能操作。
// ============================================================

// checkProjectAccess checks if a user is permitted to access a specific project.
func (h *APIHandler) checkProjectAccess(username string, targetProjectID string) bool {
	// 从数据库查询用户的权限列表
	var permittedProjectsStr string
	err := h.db.QueryRow(
		"SELECT COALESCE(permitted_projects, '*') FROM users WHERE username = ?",
		username,
	).Scan(&permittedProjectsStr)
	if err != nil {
		return false // 找不到用户？不给访问
	}

	// 解析逗号分隔的权限列表
	permittedList := strings.Split(permittedProjectsStr, ",")
	for _, p := range permittedList {
		p = strings.TrimSpace(p)
		// "*" 表示所有项目，或者直接匹配项目 ID
		if p == "*" || p == targetProjectID {
			return true // ✅ 有权限！
		}
	}
	return false // ❌ 没权限
}

// HandleGetProjects 返回当前用户能看到的项目列表
func (h *APIHandler) HandleGetProjects(c *gin.Context) {
	usernameVal, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	username := usernameVal.(string)

	// 查询用户的权限
	var permittedProjectsStr string
	err := h.db.QueryRow(
		"SELECT COALESCE(permitted_projects, '*') FROM users WHERE username = ?",
		username,
	).Scan(&permittedProjectsStr)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user permissions"})
		return
	}

	// 解析权限列表
	permittedList := strings.Split(permittedProjectsStr, ",")
	permittedMap := make(map[string]bool)
	for _, p := range permittedList {
		p = strings.TrimSpace(p)
		if p != "" {
			permittedMap[p] = true
		}
	}

	// 过滤项目列表，只返回用户有权限的
	projects := make([]domain.ProjectConfig, 0)
	for _, p := range h.config.Projects {
		if permittedMap["*"] || permittedMap[p.ID] {
			projects = append(projects, p)
		}
	}
	c.JSON(http.StatusOK, projects)
}

// UpdatePermissionsRequest 更新权限的请求结构体
type UpdatePermissionsRequest struct {
	PermittedProjects string `json:"permitted_projects"` // 新权限列表
}

// HandleUpdateUserPermissions 更新用户的权限（管理员专用）
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

	_, err := h.db.Exec(
		"UPDATE users SET permitted_projects = ? WHERE username = ?",
		req.PermittedProjects, username,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "permissions updated successfully"})
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 权限控制是怎么实现的？
//    A: 每个用户有一个 permitted_projects 字段，用逗号分隔项目 ID。
//       "*" = 所有项目，否则只有能匹配上的才能操作~
//
// 中级：
// 2. Q: COALESCE(permitted_projects, '*') 是什么意思？
//    A: SQL 函数！如果 permitted_projects 为 NULL，就用 "*"（全部权限）。
//       这确保老用户升级后自动拥有全部权限，不会"没权限"~
// ============================================================
