package api

import (
	"deploy/godeployer/application"
	"deploy/godeployer/infrastructure/git"
	"deploy/godeployer/infrastructure/ssh"
	"deploy/godeployer/infrastructure/sys"

	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"deploy/godeployer/domain"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
// 全局并发限流阀门，限制最大同时执行 5 个并发 Git Diff 比对
var diffSemaphore = make(chan struct{}, 5)

type APIHandler struct {
	config   *domain.Config
	db       *sql.DB
	executor ssh.RemoteExecutor
	engine   *application.DeployEngine
}

func SetupRoutes(config *domain.Config, db *sql.DB, engine *application.DeployEngine) *gin.Engine {
	return SetupRoutesWithExecutor(config, db, nil, engine)
}

// SetupRoutesWithExecutor 允许传入模拟 Executor 以支持测试驱动开发 (TDD)
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func SetupRoutesWithExecutor(config *domain.Config, db *sql.DB, executor ssh.RemoteExecutor, engine *application.DeployEngine) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if config.Global.DiffMaxSizeKB <= 0 {
		config.Global.DiffMaxSizeKB = 5120
	}
	if config.Global.DiskMinSpaceMB <= 0 {
		config.Global.DiskMinSpaceMB = 500
	}

	handler := &APIHandler{
		config:   config,
		db:       db,
		executor: executor,
		engine:   engine,
	}

	// 开放接口
	r.POST("/api/login", handler.HandleLogin)
	r.POST("/api/webhooks/github/:project_id/:env_id", handler.HandleGithubWebhook)

	// WebSocket 路由，内部自带鉴权 (Token-based authentication on first payload)
	r.GET("/api/ws/tasks/:id/log", handler.HandleWSLog)

	// 受保护接口（要求 JWT 认证）
	protected := r.Group("/api")
	protected.Use(application.AuthMiddleware(config.Global.JWTSecret))
	{
		// 基础只读权限：所有角色均可访问
		viewerGrp := protected.Group("/")
		viewerGrp.Use(application.RoleMiddleware("admin", "deployer", "viewer"))
		{
			viewerGrp.GET("/projects", handler.HandleGetProjects)
			viewerGrp.GET("/projects/:id/refs", handler.HandleGetProjectRefs)
			viewerGrp.GET("/projects/:id/commits", handler.HandleGetProjectCommits)
			viewerGrp.GET("/projects/:id/preview_diff", handler.HandleGetProjectPreviewDiff)
			viewerGrp.GET("/tasks", handler.HandleGetTasks)
			viewerGrp.GET("/tasks/:id", handler.HandleGetTaskDetail)
			viewerGrp.GET("/tasks/:id/log", handler.HandleGetTaskLog)
			viewerGrp.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)
		}

		// 管理员权限操作
		adminGrp := protected.Group("/")
		adminGrp.Use(application.RoleMiddleware("admin"))
		{
			adminGrp.GET("/users", handler.HandleGetUsers)
			adminGrp.POST("/users", handler.HandleCreateUser)
			adminGrp.PUT("/users/:username", handler.HandleUpdateUser)
			adminGrp.DELETE("/users/:username", handler.HandleDeleteUser)
			adminGrp.GET("/users/:username/git_binding", handler.HandleGetUserGitBinding)
			adminGrp.PUT("/users/:username/git_binding", handler.HandleUpdateUserGitBinding)
			adminGrp.PUT("/users/:username/permissions", handler.HandleUpdateUserPermissions)
			// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
			adminGrp.POST("/system/prune", handler.HandleSystemPrune)
		}

		// 部署操作权限：仅 admin 和 deployer 可访问
		deployerGrp := protected.Group("/")
		deployerGrp.Use(application.RoleMiddleware("admin", "deployer"))
		{
			deployerGrp.POST("/tasks", handler.HandleCreateTask)
			deployerGrp.POST("/tasks/:id/rollback", handler.HandleTriggerRollback)
		}
	}

	return r
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *APIHandler) HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从数据库查询用户
	var passwordHash string
	var role string
	err := h.db.QueryRow("SELECT password_hash, role FROM users WHERE username = ?", req.Username).Scan(&passwordHash, &role)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 比对密码
	if !application.CheckPasswordHash(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 签发 Token (有效 24 小时)
	token, err := application.GenerateToken(req.Username, role, h.config.Global.JWTSecret, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": req.Username,
		"role":     role,
	})
}

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

func (h *APIHandler) HandleGetProjectRefs(c *gin.Context) {
	projectID := c.Param("id")
	proj, ok := h.config.Projects[projectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	cmd := exec.CommandContext(c.Request.Context(), "git", "ls-remote", "--heads", "--tags", proj.Repo)
	out, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get refs: %v", err)})
		return
	}

	type GitRef struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Hash string `json:"hash"`
	}
	var refs []GitRef

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		hash := parts[0]
		refPath := parts[1]

		if strings.HasPrefix(refPath, "refs/heads/") {
			name := strings.TrimPrefix(refPath, "refs/heads/")
			refs = append(refs, GitRef{Name: name, Type: "branch", Hash: hash})
		} else if strings.HasPrefix(refPath, "refs/tags/") {
			name := strings.TrimPrefix(refPath, "refs/tags/")
			if strings.HasSuffix(name, "^{}") {
				continue
			}
			refs = append(refs, GitRef{Name: name, Type: "tag", Hash: hash})
		}
	}

	c.JSON(http.StatusOK, refs)
}

func (h *APIHandler) HandleGetProjectCommits(c *gin.Context) {
	projectID := c.Param("id")
	proj, ok := h.config.Projects[projectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	keyword := c.Query("q")
	author := c.Query("author")
	file := c.Query("file")
	ref := c.Query("ref")

	// 这里按需触发 cache 更新
	if err := git.EnsureRepoCache(c.Request.Context(), proj.Repo, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update repo cache: %v", err)})
		return
	}

	commits, err := git.GetCommits(c.Request.Context(), projectID, keyword, author, file, ref)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get commits: %v", err)})
		return
	}

	c.JSON(http.StatusOK, commits)
}

func (h *APIHandler) HandleGetProjectPreviewDiff(c *gin.Context) {
	// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
	// 进程并发安全限流，排队 3 秒超时退化，杜绝雪崩卡死
	select {
	case diffSemaphore <- struct{}{}:
		defer func() { <-diffSemaphore }()
	case <-time.After(3 * time.Second):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "系统繁忙，差异比对排队中，请稍后再试"})
		return
	}

	projectID := c.Param("id")
	proj, ok := h.config.Projects[projectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	fromCommit := c.Query("from")
	toCommit := c.Query("to")
	if toCommit == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to commit is required"})
		return
	}

	envID := c.Query("env_id")
	if fromCommit == "" && envID != "" {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
		// 自动查询该项目在此环境上最近一次成功部署的 commit_id 作为 Live Diff 对比基准
		_ = h.db.QueryRow("SELECT commit_id FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND status = 'success' ORDER BY id DESC LIMIT 1", projectID, envID).Scan(&fromCommit)
	}

	diffType := c.DefaultQuery("diff_type", "live")
	fileParam := c.Query("file")
	if fileParam != "" {
		// @Ref: docs/sps/plans/20260530_lazy_load_file_diff_plan.md | @Date: 2026-05-30
		// 只获取单个文件的 diff，直接读取本地 bare 缓存库，免去不必要的 git.EnsureRepoCache 网络请求以极速响应
		limitBytes := h.config.Global.DiffMaxSizeKB * 1024
		baseCommit := fromCommit
		if diffType == "git_log" {
			baseCommit = toCommit + "^"
		}
		diffText, err := git.GetDiffForFile(c.Request.Context(), projectID, baseCommit, toCommit, fileParam, limitBytes)
		if err != nil {
			diffText = "无法获取该文件的差异对比文本。"
		}
		c.JSON(http.StatusOK, gin.H{
			"diff":  diffText,
			"files": []string{},
		})
		return
	}

	if err := git.EnsureRepoCache(c.Request.Context(), proj.Repo, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update repo cache: %v", err)})
		return
	}

	targetType := c.Query("target_type")
	if targetType == "" {
		if git.IsCommitHash(toCommit) {
			targetType = "commit"
		} else {
			targetType = "branch"
		}
	}

	// 如果没有传入 file 参数，我们根据发布类型返回全量或变更文件列表，避开全量大 Diff 的拉取，避免 OOM 并极大提升弹框响应速度
	gitCacheDir := git.GetCacheDir(projectID)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if targetType == "commit" {
		cmd = exec.CommandContext(ctx, "git", "diff", "--name-only", fromCommit, toCommit, "--")
	} else {
		cmd = exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", toCommit, "--")
	}
	cmd.Dir = gitCacheDir
	filesOutput, filesErr := cmd.CombinedOutput()
	fileList := make([]string, 0)
	if filesErr == nil {
		lines := strings.Split(string(filesOutput), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				fileList = append(fileList, trimmed)
			}
		}
	}

	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	// 限制返回的最大文件树长度（例如最多 2000 个文件），防止前端 Element Plus 树节点过多渲染时挂起
	const maxFilesLimit = 2000
	if len(fileList) > maxFilesLimit {
		fileList = fileList[:maxFilesLimit]
		fileList = append(fileList, "注意：全量文件数过多已进行截断展示，请在本地 Git 中查看完整目录树")
	}

	c.JSON(http.StatusOK, gin.H{
		"diff":  "",
		"files": fileList,
	})
}

type UpdateGitBindingRequest struct {
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
}

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

type CreateTaskRequest struct {
	ProjectID    string `json:"project_id" binding:"required"`
	EnvID        string `json:"env_id" binding:"required"`
	CommitID     string `json:"commit_id" binding:"required"`
	TargetType   string `json:"target_type"`
	Description  string `json:"description"`
	ExtraExclude string `json:"extra_exclude"`
}

func (h *APIHandler) HandleCreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查项目是否存在
	proj, exists := h.config.Projects[req.ProjectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 检查环境是否存在
	envExists := false
	for _, env := range proj.Environments {
		if env.ID == req.EnvID {
			envExists = true
			break
		}
	}
	if !envExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	username := usernameVal.(string)
	if !h.checkProjectAccess(username, req.ProjectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
	// 环境部署互斥锁
	var activeCount int
	lockSQL := `
		SELECT COUNT(*) 
		FROM deploy_tasks 
		WHERE project_id = ? AND env_id = ? AND status IN ('pending', 'deploying')`
	_ = h.db.QueryRow(lockSQL, req.ProjectID, req.EnvID).Scan(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "another deployment is already in progress for this project and environment"})
		return
	}

	// 获取用户信息
	var userID int64
	var boundAuthors string
	var restrict bool
	_ = h.db.QueryRow("SELECT id, COALESCE(bound_git_authors, ''), COALESCE(restrict_git_authors, 0) FROM users WHERE username = ?", username).Scan(&userID, &boundAuthors, &restrict)

	if restrict {
		if err := git.EnsureRepoCache(c.Request.Context(), proj.Repo, req.ProjectID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update repo cache for auth check"})
			return
		}

		author, err := git.GetCommitAuthor(c.Request.Context(), req.ProjectID, req.CommitID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve commit author"})
			return
		}

		allowed := false
		for _, a := range strings.Split(boundAuthors, ",") {
			if strings.TrimSpace(a) == author {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Access Denied: you are not allowed to deploy commits authored by '%s'", author)})
			return
		}
	}

	releaseName := time.Now().Format("20060102150405")

	targetType := req.TargetType
	if targetType == "" {
		if git.IsCommitHash(req.CommitID) {
			targetType = "commit"
		} else {
			targetType = "branch"
		}
	}

	// 插入任务记录（初始状态为 pending）
	insertSQL := `
		INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, description, extra_exclude, target_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := h.db.Exec(insertSQL, req.ProjectID, req.EnvID, req.CommitID, "pending", releaseName, userID, username, "{}", req.Description, req.ExtraExclude, targetType, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	taskID, _ := res.LastInsertId()

	// 创建日志目录和路径
	logDir := h.config.Global.LogPath
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, fmt.Sprintf("task_%d.log", taskID))

	// 创建带超时的上下文，交由调度器管理生命周期
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)

	job := &domain.DeployJob{
		Ctx:         ctx,
		Cancel:      cancel,
		TaskID:      taskID,
		Config:      h.config,
		LogFilePath: logFilePath,
	}

	err = h.engine.SubmitJob(job)
	if err != nil {
		cancel()
		if err == application.ErrQueueFull {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "deployment queue is full"})
			h.engine.UpdateTaskStatus(taskID, "failed")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit deploy task"})
		return
	}

	// 返回 201 Created 且携带审计人
	c.JSON(http.StatusCreated, gin.H{
		"id":           taskID,
		"task_id":      taskID,
		"project_id":   req.ProjectID,
		"project_name": proj.Name,
		"env_id":       req.EnvID,
		"commit_id":    req.CommitID,
		"status":       "pending",
		"username":     username,
		"created_at":   time.Now().Format(time.RFC3339),
	})
}

// HandleGetTasks 返回历史任务记录列表
func (h *APIHandler) HandleGetTasks(c *gin.Context) {
	projectID := c.Query("project_id")
	envID := c.Query("env_id")

	query := `SELECT id, project_id, env_id, commit_id, status, release_name, username, COALESCE(description, ''), COALESCE(extra_exclude, ''), created_at FROM deploy_tasks`
	var args []interface{}

	if projectID != "" && envID != "" {
		query += ` WHERE project_id = ? AND env_id = ?`
		args = append(args, projectID, envID)
	}
	query += ` ORDER BY id DESC LIMIT 50`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type TaskRes struct {
		ID           int64     `json:"id"`
		ProjectID    string    `json:"project_id"`
		EnvID        string    `json:"env_id"`
		CommitID     string    `json:"commit_id"`
		Status       string    `json:"status"`
		ReleaseName  string    `json:"release_name"`
		Username     string    `json:"username"`
		Description  string    `json:"description"`
		ExtraExclude string    `json:"extra_exclude"`
		CreatedAt    time.Time `json:"created_at"`
	}

	tasks := make([]TaskRes, 0)
	for rows.Next() {
		var t TaskRes
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.EnvID, &t.CommitID, &t.Status, &t.ReleaseName, &t.Username, &t.Description, &t.ExtraExclude, &t.CreatedAt); err == nil {
			tasks = append(tasks, t)
		}
	}

	c.JSON(http.StatusOK, tasks)
}

// HandleGetTaskDetail 获取任务详情
func (h *APIHandler) HandleGetTaskDetail(c *gin.Context) {
	id := c.Param("id")
	var status, projectID string
	err := h.db.QueryRow("SELECT status, project_id FROM deploy_tasks WHERE id = ?", id).Scan(&status, &projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": status})
}

// HandleGetTaskLog 获取部署日志文件的文本内容 (带 1MB 截断防爆保护)
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleGetTaskLog(c *gin.Context) {
	id := c.Param("id")

	var projectID string
	err := h.db.QueryRow("SELECT project_id FROM deploy_tasks WHERE id = ?", id).Scan(&projectID)
	if err == nil {
		usernameVal, _ := c.Get("username")
		if !h.checkProjectAccess(usernameVal.(string), projectID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
			return
		}
	}

	logFilePath := filepath.Join(h.config.Global.LogPath, fmt.Sprintf("task_%s.log", id))

	file, err := os.Open(logFilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "log file not found"})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stat log file"})
		return
	}

	var data []byte
	const maxLogSize = 1 * 1024 * 1024 // 1MB 限额

	if stat.Size() <= maxLogSize {
		data, err = os.ReadFile(logFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 大于 1MB 进行截断读取，指针移动到倒数 1MB 处
		_, err = file.Seek(stat.Size()-maxLogSize, io.SeekStart)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to seek log file"})
			return
		}
		buf := make([]byte, maxLogSize)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read truncated log"})
			return
		}
		data = []byte("[Log truncated, showing last 1MB]...\n" + string(buf[:n]))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":  id,
		"log": string(data),
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 支持跨域
	},
}

// HandleWSLog WebSocket 流式推送日志
// @Ref: docs/sps/plans/20260527_m6_frontend_ir.md | @Date: 2026-05-27
func (h *APIHandler) HandleWSLog(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. 首包鉴权 (Token-based authentication on first payload)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	err = conn.ReadJSON(&authMsg)
	if err != nil || authMsg.Type != "auth" {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "missing or invalid auth payload"))
		return
	}

	username, _, err := application.ParseToken(authMsg.Token, h.config.Global.JWTSecret)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"))
		return
	}

	// 重置读超时，之后只推日志，无需读客户端
	conn.SetReadDeadline(time.Time{})

	id := c.Param("id")

	var projectID string
	err = h.db.QueryRow("SELECT project_id FROM deploy_tasks WHERE id = ?", id).Scan(&projectID)
	if err == nil && !h.checkProjectAccess(username, projectID) {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "access denied for this project"))
		return
	}
	logFilePath := filepath.Join(h.config.Global.LogPath, fmt.Sprintf("task_%s.log", id))

	// 简单的轮询推送日志 delta (类似 tail -f)
	var lastPos int64 = 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return // 客户端主动断开
		case <-ticker.C:
			file, err := os.Open(logFilePath)
			if err != nil {
				// 文件可能还没创建，等待
				continue
			}

			stat, err := file.Stat()
			if err != nil {
				file.Close()
				continue
			}

			currentSize := stat.Size()
			if currentSize < lastPos {
				// 文件被截断或重新创建
				lastPos = 0
			}

			if currentSize > lastPos {
				file.Seek(lastPos, io.SeekStart)
				buf := make([]byte, currentSize-lastPos)
				n, err := file.Read(buf)
				if err == nil && n > 0 {
					err = conn.WriteMessage(websocket.TextMessage, buf[:n])
					if err != nil {
						file.Close()
						return // 发送失败，断开连接
					}
					lastPos += int64(n)
				}
			}
			file.Close()
		}
	}
}

// HandleTriggerRollback 触发版本回滚到特定历史任务版本
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleTriggerRollback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	// 查出任务相关参数以获取对应环境的 server 配置
	var projectID, envID string
	err = h.db.QueryRow("SELECT project_id, env_id FROM deploy_tasks WHERE id = ?", id).Scan(&projectID, &envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	proj, exists := h.config.Projects[projectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	var targetEnv *domain.EnvironmentConfig
	for _, env := range proj.Environments {
		if env.ID == envID {
			targetEnv = &env
			break
		}
	}

	if targetEnv == nil || len(targetEnv.Servers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target servers not configured"})
		return
	}

	// 异步调用回滚（精准切换，同时支持 Mock 注入）
	engine := application.NewDeployEngine(h.db, h.executor)

	// 这里支持为每个服务器逐一精准回滚到目标 task
	for _, srv := range targetEnv.Servers {
		if err := engine.RunRollbackToTask(id, srv); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "rollback completed"})
}

// HandleGetTaskDiff 获取当前任务与其前一个成功部署版本之间的 Git Diff
func (h *APIHandler) HandleGetTaskDiff(c *gin.Context) {
	// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
	// 进程并发安全限流，排队 3 秒超时退化，杜绝雪崩卡死
	select {
	case diffSemaphore <- struct{}{}:
		defer func() { <-diffSemaphore }()
	case <-time.After(3 * time.Second):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "系统繁忙，差异比对排队中，请稍后再试"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	// 1. 获取当前任务的 commit_id, project_id, release_name, status, created_at 及 target_type
	var projectID, envID, currentCommit, releaseName, status, createdAt, targetType string
	err = h.db.QueryRow("SELECT project_id, env_id, commit_id, release_name, status, created_at, target_type FROM deploy_tasks WHERE id = ?", id).
		Scan(&projectID, &envID, &currentCommit, &releaseName, &status, &createdAt, &targetType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	if status == "deploying" || status == "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "task is still deploying, diff not ready"})
		return
	}

	// 2. 查出同一项目同环境下在此任务之前的最近一次成功发布的 commit_id
	querySQL := `
		SELECT commit_id 
		FROM deploy_tasks 
		WHERE project_id = ? AND env_id = ? AND id < ? AND status = 'success' 
		ORDER BY id DESC LIMIT 1`

	var prevCommit string
	err = h.db.QueryRow(querySQL, projectID, envID, id).Scan(&prevCommit)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"diff": "首次部署，暂无对比基准。"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query previous task: " + err.Error()})
		return
	}

	diffType := c.DefaultQuery("diff_type", "live")
	fileParam := c.Query("file")

	createdYM := "default"
	if len(createdAt) >= 7 {
		createdYM = strings.ReplaceAll(createdAt[:7], "-", "")
	}
	diffCacheDir := filepath.Join(h.config.Global.LogPath, "diffs", "projects", projectID, createdYM)
	diffCacheFile := filepath.Join(diffCacheDir, fmt.Sprintf("task_%d_diff.log", id))

	// 3. 确定执行 git diff 的工作目录
	buildPath := filepath.Join(h.config.Global.WorkspacePath, projectID, releaseName)
	gitRepoPath := buildPath
	if _, statErr := os.Stat(filepath.Join(buildPath, ".git")); os.IsNotExist(statErr) {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
		// 优先使用本地项目的 bare 缓存目录，其常驻且包含完整引用，避免直接触发 walk 全局搜索
		cacheDir := git.GetCacheDir(projectID)
		if _, cacheErr := os.Stat(cacheDir); cacheErr == nil {
			gitRepoPath = cacheDir
		} else {
			found, walkErr := git.FindGitRepo(h.config.Global.WorkspacePath, currentCommit)
			if walkErr == nil && found != "" {
				gitRepoPath = found
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 4. 如果有 file 参数，只获取特定文件的单文件 diff，跳过缓存读取
	if fileParam != "" {
		// @Ref: docs/sps/plans/20260530_lazy_load_file_diff_plan.md | @Date: 2026-05-30
		limitBytes := h.config.Global.DiffMaxSizeKB * 1024
		baseCommit := prevCommit
		if diffType == "git_log" {
			baseCommit = currentCommit + "^"
		}
		var diffText string
		var err error
		if diffType == "live" && (targetType == "branch" || targetType == "tag") {
			diffText = "提示：全量部署任务，未归档与线上对比快照。请在右上方切换为「本地变更 (Git Log Diff)」查看文件修改。"
		} else {
			diffText, err = git.GetDiffForFile(ctx, projectID, baseCommit, currentCommit, fileParam, limitBytes)
			if err != nil {
				// 降级尝试：从物理 JSON 快照中做文本正则切片提取
				if data, readErr := os.ReadFile(diffCacheFile); readErr == nil {
					var cacheObj struct {
						Diff       string `json:"diff"`
						GitLogDiff string `json:"git_log_diff"`
					}
					if jsonErr := json.Unmarshal(data, &cacheObj); jsonErr == nil {
						targetFullDiff := cacheObj.Diff
						isFullReleaseCache := false
						if diffType == "git_log" && cacheObj.GitLogDiff != "" {
							targetFullDiff = cacheObj.GitLogDiff
						} else if diffType == "live" && cacheObj.Diff == "" && cacheObj.GitLogDiff != "" {
							isFullReleaseCache = true
						}

						if isFullReleaseCache {
							diffText = "提示：全量部署任务，未归档与线上对比快照。请在右上方切换为「本地变更(Git Log Diff)」查看文件修改。"
							err = nil
						} else {
							diffText = extractFileDiffFromLog(targetFullDiff, fileParam)
							err = nil
						}
					}
				}
			}
		}
		if err != nil {
			diffText = "无法获取该文件的差异对比文本。"
		}
		c.JSON(http.StatusOK, gin.H{
			"files": "",
			"diff":  diffText,
		})
		return
	}

	// 5. 如果没有 file 参数，仅获取文件列表（避开全量差异读取）
	// 优先尝试读取持久化缓存
	if data, readErr := os.ReadFile(diffCacheFile); readErr == nil {
		var cacheObj struct {
			Files      string `json:"files"`
			Diff       string `json:"diff"`
			GitLogDiff string `json:"git_log_diff"`
		}
		if jsonErr := json.Unmarshal(data, &cacheObj); jsonErr == nil {
			c.JSON(http.StatusOK, gin.H{
				"files": cacheObj.Files,
				"diff":  "", // 懒加载，在此处为空
			})
			return
		}
	}

	// 获取变更文件状态列表 (e.g. M src/App.vue)
	var filesListStr string
	filesCmd := exec.CommandContext(ctx, "git", "diff", "--name-status", prevCommit, currentCommit)
	filesCmd.Dir = gitRepoPath
	if filesOut, filesErr := filesCmd.CombinedOutput(); filesErr == nil {
		filesListStr = string(filesOut)
	} else {
		filesListStr = "获取变更文件列表失败"
	}

	// 写入缓存文件（仅记录文件列表，将 diff 设为空白，彻底杜绝大 diff 在硬盘和内存的无谓堆积）
	if sys.GetFreeDiskSpaceMB(h.config.Global.LogPath) >= h.config.Global.DiskMinSpaceMB {
		_ = os.MkdirAll(diffCacheDir, 0755)
		cacheMap := map[string]string{
			"files": filesListStr,
			"diff":  "",
		}
		if cacheBytes, marshalErr := json.Marshal(cacheMap); marshalErr == nil {
			_ = os.WriteFile(diffCacheFile, cacheBytes, 0644)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files": filesListStr,
		"diff":  "",
	})
}

// @Ref: docs/sps/decisions/20260529_diff_ux_loading_scan.md | @Date: 2026-05-29

// ComputeGithubSignature 计算 Github Webhook 签名
func ComputeGithubSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// HandleGithubWebhook 处理 Github Push 事件，进行防抖与自动部署
func (h *APIHandler) HandleGithubWebhook(c *gin.Context) {
	projectID := c.Param("project_id")
	envID := c.Param("env_id")

	proj, exists := h.config.Projects[projectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 1. 验证签名
	signatureHeader := c.GetHeader("X-Hub-Signature-256")
	if signatureHeader == "" || proj.WebhookSecret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature or secret not configured"})
		return
	}
	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature format"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	expectedMac := ComputeGithubSignature(body, proj.WebhookSecret)
	if !hmac.Equal([]byte(parts[1]), []byte(expectedMac)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
		return
	}

	// 2. 解析分支信息
	var payload struct {
		Ref   string `json:"ref"`
		After string `json:"after"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return
	}

	expectedBranch := proj.Branch
	if expectedBranch == "" {
		expectedBranch = "main" // fallback
	}

	if !strings.HasSuffix(payload.Ref, "/"+expectedBranch) {
		c.JSON(http.StatusOK, gin.H{"message": "ignored push to different branch"})
		return
	}

	// 3. 防抖逻辑 (Thundering Herd Defense)
	var activeCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND status IN ('pending', 'deploying')", projectID, envID).Scan(&activeCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error checking task status"})
		return
	}
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "a deployment is already running for this project and environment"})
		return
	}

	// 4. 插入任务并异步触发部署
	commitID := payload.After
	if commitID == "" {
		commitID = "HEAD"
	}

	releaseName := time.Now().Format("20060102150405")
	username := "github-webhook"
	var userID int64 = 0 // 系统或特殊的Webhook用户ID

	insertSQL := `
		INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := h.db.Exec(insertSQL, projectID, envID, commitID, "pending", releaseName, userID, username, "{}", time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	taskID, _ := res.LastInsertId()

	// 创建日志目录和路径
	logDir := h.config.Global.LogPath
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, fmt.Sprintf("task_%d.log", taskID))

	// 异步调用部署引擎
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)

	job := &domain.DeployJob{
		Ctx:         ctx,
		Cancel:      cancel,
		TaskID:      taskID,
		Config:      h.config,
		LogFilePath: logFilePath,
	}

	err = h.engine.SubmitJob(job)
	if err != nil {
		cancel()
		if err == application.ErrQueueFull {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "deployment queue is full"})
			h.engine.UpdateTaskStatus(taskID, "failed")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit deploy task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "deployment triggered", "task_id": taskID})
}

// HandleSystemPrune 手动系统清理与脏数据自愈
// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
func (h *APIHandler) HandleSystemPrune(c *gin.Context) {
	roleVal, _ := c.Get("role")
	if roleVal.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	var prunedTasksCount, prunedOrphansCount int
	var freedBytes int64

	// 1. 主动老化清理
	var idsToPrune []int64
	var taskMap = make(map[int64][2]string) // taskID -> [projectID, createdAt]

	// 基于天数老化
	if h.config.Global.TaskRetainDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -h.config.Global.TaskRetainDays)
		rows, err := h.db.Query(`
			SELECT id, project_id, created_at 
			FROM deploy_tasks 
			WHERE status NOT IN ('pending', 'deploying') AND created_at < ?`, cutoffTime)
		if err == nil {
			for rows.Next() {
				var id int64
				var pid, createdAt string
				if err := rows.Scan(&id, &pid, &createdAt); err == nil {
					idsToPrune = append(idsToPrune, id)
					taskMap[id] = [2]string{pid, createdAt}
				}
			}
			rows.Close()
		}
	}

	// 基于数量限额老化
	if h.config.Global.TaskRetainMax > 0 {
		var totalCount int
		_ = h.db.QueryRow("SELECT COUNT(*) FROM deploy_tasks").Scan(&totalCount)
		if totalCount > h.config.Global.TaskRetainMax {
			excess := totalCount - h.config.Global.TaskRetainMax
			rows, err := h.db.Query(`
				SELECT id, project_id, created_at 
				FROM deploy_tasks 
				WHERE status NOT IN ('pending', 'deploying') 
				ORDER BY id ASC LIMIT ?`, excess)
			if err == nil {
				for rows.Next() {
					var id int64
					var pid, createdAt string
					if err := rows.Scan(&id, &pid, &createdAt); err == nil {
						// 避免重复
						if _, exists := taskMap[id]; !exists {
							idsToPrune = append(idsToPrune, id)
							taskMap[id] = [2]string{pid, createdAt}
						}
					}
				}
				rows.Close()
			}
		}
	}

	// 执行“先库后盘”第一步：从数据库删除
	if len(idsToPrune) > 0 {
		for _, id := range idsToPrune {
			_, err := h.db.Exec("DELETE FROM deploy_tasks WHERE id = ?", id)
			if err == nil {
				prunedTasksCount++
			}
		}
	}

	// 执行“先库后盘”第二步：删除对应的物理文件，并累计释放大小
	logDir := h.config.Global.LogPath
	for _, id := range idsToPrune {
		// 清理运行日志
		logPath := filepath.Join(logDir, fmt.Sprintf("task_%d.log", id))
		if fi, err := os.Stat(logPath); err == nil {
			freedBytes += fi.Size()
			_ = os.Remove(logPath)
		}

		// 清理 diff 快照
		meta := taskMap[id]
		createdYM := "default"
		if len(meta[1]) >= 7 {
			createdYM = strings.ReplaceAll(meta[1][:7], "-", "")
		}
		diffFile := filepath.Join(logDir, "diffs", "projects", meta[0], createdYM, fmt.Sprintf("task_%d_diff.log", id))
		if fi, err := os.Stat(diffFile); err == nil {
			freedBytes += fi.Size()
			_ = os.Remove(diffFile)
		}
	}

	// 2. 脏数据/孤儿文件物理自愈
	// 遍历 LogPath 根目录清除孤儿日志文件
	if files, err := os.ReadDir(logDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasPrefix(file.Name(), "task_") && strings.HasSuffix(file.Name(), ".log") {
				var id int64
				_, scanErr := fmt.Sscanf(file.Name(), "task_%d.log", &id)
				if scanErr == nil {
					var exists int
					err := h.db.QueryRow("SELECT COUNT(*) FROM deploy_tasks WHERE id = ?", id).Scan(&exists)
					if err == nil && exists == 0 {
						filePath := filepath.Join(logDir, file.Name())
						if fi, statErr := os.Stat(filePath); statErr == nil {
							freedBytes += fi.Size()
							_ = os.Remove(filePath)
							prunedOrphansCount++
						}
					}
				}
			}
		}
	}

	// 遍历 LogPath/diffs/projects 清除孤儿 diff 快照
	diffsRoot := filepath.Join(logDir, "diffs", "projects")
	_ = filepath.Walk(diffsRoot, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasPrefix(info.Name(), "task_") && strings.HasSuffix(info.Name(), "_diff.log") {
			var id int64
			_, scanErr := fmt.Sscanf(info.Name(), "task_%d_diff.log", &id)
			if scanErr == nil {
				var exists int
				err := h.db.QueryRow("SELECT COUNT(*) FROM deploy_tasks WHERE id = ?", id).Scan(&exists)
				if err == nil && exists == 0 {
					freedBytes += info.Size()
					_ = os.Remove(path)
					prunedOrphansCount++
				}
			}
		}
		return nil
	})

	c.JSON(http.StatusOK, gin.H{
		"message":              "system prune and self-healing completed",
		"pruned_tasks_count":   prunedTasksCount,
		"pruned_orphans_count": prunedOrphansCount,
		"freed_bytes":          freedBytes,
	})
}

// @Ref: docs/sps/plans/20260530_goal_perfect_diff_plan.md | @Date: 2026-05-30

// @Ref: docs/sps/plans/20260530_goal_perfect_diff_plan.md | @Date: 2026-05-30
func extractFileDiffFromLog(fullDiff, filePath string) string {
	lines := strings.Split(fullDiff, "\n")
	var result []string
	recording := false
	targetHeader := fmt.Sprintf("diff --git a/%s b/%s", filePath, filePath)
	targetHeaderAlternative := fmt.Sprintf("diff --git a/%s ", filePath)

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if strings.HasPrefix(line, targetHeader) || strings.Contains(line, targetHeaderAlternative) {
				recording = true
				result = append(result, line)
			} else {
				if recording {
					break
				}
			}
		} else if recording {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return "该文件无代码变更差异。"
	}
	return strings.Join(result, "\n")
}
