package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"

	"github.com/gin-gonic/gin"
)

type UpdateGitBindingRequest struct {
	BoundGitAuthors    string `json:"bound_git_authors"`
	RestrictGitAuthors bool   `json:"restrict_git_authors"`
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
	res, err := h.db.Exec(insertSQL, projectID, envID, commitID, string(domain.StatusPending), releaseName, userID, username, "{}", time.Now())
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
			h.engine.UpdateTaskStatus(taskID, domain.StatusFailed)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit deploy task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "deployment triggered", "task_id": taskID})
}
