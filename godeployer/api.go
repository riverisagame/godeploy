package godeployer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
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

type APIHandler struct {
	config   *Config
	db       *sql.DB
	executor RemoteExecutor
	engine   *DeployEngine
}

func SetupRoutes(config *Config, db *sql.DB, engine *DeployEngine) *gin.Engine {
	return SetupRoutesWithExecutor(config, db, nil, engine)
}

// SetupRoutesWithExecutor 允许传入模拟 Executor 以支持测试驱动开发 (TDD)
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func SetupRoutesWithExecutor(config *Config, db *sql.DB, executor RemoteExecutor, engine *DeployEngine) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

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
	protected.Use(AuthMiddleware(config.Global.JWTSecret))
	{
		// 基础只读权限：所有角色均可访问
		viewerGrp := protected.Group("/")
		viewerGrp.Use(RoleMiddleware("admin", "deployer", "viewer"))
		{
			viewerGrp.GET("/projects", handler.HandleGetProjects)
			viewerGrp.GET("/tasks", handler.HandleGetTasks)
			viewerGrp.GET("/tasks/:id", handler.HandleGetTaskDetail)
			viewerGrp.GET("/tasks/:id/log", handler.HandleGetTaskLog)
			viewerGrp.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)
		}

		// 部署操作权限：仅 admin 和 deployer 可访问
		deployerGrp := protected.Group("/")
		deployerGrp.Use(RoleMiddleware("admin", "deployer"))
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
	if !CheckPasswordHash(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 签发 Token (有效 24 小时)
	token, err := GenerateToken(req.Username, role, h.config.Global.JWTSecret, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": req.Username,
	})
}

func (h *APIHandler) HandleGetProjects(c *gin.Context) {
	projects := make([]ProjectConfig, 0, len(h.config.Projects))
	for _, p := range h.config.Projects {
		projects = append(projects, p)
	}
	c.JSON(http.StatusOK, projects)
}

type CreateTaskRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
	EnvID     string `json:"env_id" binding:"required"`
	CommitID  string `json:"commit_id" binding:"required"`
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

	// 从中间件中解析出审计用户名
	usernameVal, _ := c.Get("username")
	username := usernameVal.(string)

	// 获取用户 ID
	var userID int64
	_ = h.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)

	releaseName := time.Now().Format("20060102150405")

	// 插入任务记录（初始状态为 pending）
	insertSQL := `
		INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	res, err := h.db.Exec(insertSQL, req.ProjectID, req.EnvID, req.CommitID, "pending", releaseName, userID, username, "{}", time.Now())
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

	job := &DeployJob{
		Ctx:         ctx,
		Cancel:      cancel,
		TaskID:      taskID,
		Config:      h.config,
		LogFilePath: logFilePath,
	}

	err = h.engine.SubmitJob(job)
	if err != nil {
		cancel()
		if err == ErrQueueFull {
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

	query := `SELECT id, project_id, env_id, commit_id, status, release_name, username, created_at FROM deploy_tasks`
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
		ID          int64     `json:"id"`
		ProjectID   string    `json:"project_id"`
		EnvID       string    `json:"env_id"`
		CommitID    string    `json:"commit_id"`
		Status      string    `json:"status"`
		ReleaseName string    `json:"release_name"`
		Username    string    `json:"username"`
		CreatedAt   time.Time `json:"created_at"`
	}

	tasks := make([]TaskRes, 0)
	for rows.Next() {
		var t TaskRes
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.EnvID, &t.CommitID, &t.Status, &t.ReleaseName, &t.Username, &t.CreatedAt); err == nil {
			tasks = append(tasks, t)
		}
	}

	c.JSON(http.StatusOK, tasks)
}

// HandleGetTaskDetail 获取任务详情
func (h *APIHandler) HandleGetTaskDetail(c *gin.Context) {
	id := c.Param("id")
	var status string
	err := h.db.QueryRow("SELECT status FROM deploy_tasks WHERE id = ?", id).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": status})
}

// HandleGetTaskLog 获取部署日志文件的文本内容 (带 1MB 截断防爆保护)
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleGetTaskLog(c *gin.Context) {
	id := c.Param("id")
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
	
	_, _, err = ParseToken(authMsg.Token, h.config.Global.JWTSecret)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"))
		return
	}

	// 重置读超时，之后只推日志，无需读客户端
	conn.SetReadDeadline(time.Time{})

	id := c.Param("id")
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

	var targetEnv *EnvironmentConfig
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
	engine := NewDeployEngine(h.db, h.executor)
	
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
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleGetTaskDiff(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	// 1. 获取当前任务的 commit_id, project_id, release_name 及 status
	var projectID, envID, currentCommit, releaseName, status string
	err = h.db.QueryRow("SELECT project_id, env_id, commit_id, release_name, status FROM deploy_tasks WHERE id = ?", id).
		Scan(&projectID, &envID, &currentCommit, &releaseName, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
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

	// 3. 在本地克隆工作区（WorkspacePath/project_id/release_name）执行 git diff
	buildPath := filepath.Join(h.config.Global.WorkspacePath, projectID, releaseName)

	// 带超时上限执行命令，对冲 150ms 性能限额，防止命令卡死
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", prevCommit, currentCommit)
	cmd.Dir = buildPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 优雅退化容错：如果目录已被清理或尚无本地仓库
		c.JSON(http.StatusOK, gin.H{"diff": fmt.Sprintf("git diff 执行失败 (可能本地构建目录已被清理): %s\n输出: %s", err.Error(), string(output))})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"diff": string(output),
	})
}

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

	job := &DeployJob{
		Ctx:         ctx,
		Cancel:      cancel,
		TaskID:      taskID,
		Config:      h.config,
		LogFilePath: logFilePath,
	}

	err = h.engine.SubmitJob(job)
	if err != nil {
		cancel()
		if err == ErrQueueFull {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "deployment queue is full"})
			h.engine.UpdateTaskStatus(taskID, "failed")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit deploy task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "deployment triggered", "task_id": taskID})
}
