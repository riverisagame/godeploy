package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/git"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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

	res, err := h.db.Exec(insertSQL, req.ProjectID, req.EnvID, req.CommitID, string(domain.StatusPending), releaseName, userID, username, "{}", req.Description, req.ExtraExclude, targetType, time.Now())
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
			h.engine.UpdateTaskStatus(taskID, domain.StatusFailed)
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
		"status":       string(domain.StatusPending),
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
