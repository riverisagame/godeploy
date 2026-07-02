// ============================================================
// 文件：handler_webhook.go
// 作用：🌐 GitHub Webhook 自动部署！
//
// 什么是 Webhook？
// 当你在 GitHub 上推送代码时，GitHub 可以自动通知一个 URL。
// 这个文件就是接收 GitHub 通知的"收件箱"。
//
// 流程：
// 1. GitHub 推送代码 → 发送 POST 请求到这里
// 2. 验证签名（确保是真实 GitHub 发的，不是伪造的！）
// 3. 检查是否匹配的推送分支
// 4. 检查是否有正在进行的部署（防抖）
// 5. 创建部署任务并提交到引擎
//
// 给初二小白的比喻：
// 就像你订了外卖：
// - GitHub 是厨师做好菜（推送代码）
// - Webhook 是外卖员按门铃（通知服务器）
// - 签名验证是看外卖员的工作证（防伪）
// - 防抖是看看厨房有没有在做别的菜（避免冲突）
// ============================================================

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

// ComputeGithubSignature 计算 GitHub Webhook 签名
// GitHub 在发送请求时，会用你的密钥对请求体进行签名，
// 我们这边用同样的方式计算签名，对比是否一致。
// 这个叫 HMAC-SHA256——一种安全的哈希算法。
func ComputeGithubSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// HandleGithubWebhook 处理 GitHub Push 事件，进行防抖与自动部署
func (h *APIHandler) HandleGithubWebhook(c *gin.Context) {
	projectID := c.Param("project_id")
	envID := c.Param("env_id")

	proj, exists := h.config.Projects[projectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// ============================================================
	// 🔐 1. 验证签名
	// GitHub 在请求头 X-Hub-Signature-256 中携带签名
	// 格式："sha256=xxxxx"
	// 我们用项目的 webhook_secret 计算签名，对比是否一致
	// ============================================================
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

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 计算期望的签名
	expectedMac := ComputeGithubSignature(body, proj.WebhookSecret)
	// 用 hmac.Equal 比对签名（防止"时序攻击"）
	if !hmac.Equal([]byte(parts[1]), []byte(expectedMac)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
		return
	}

	// ============================================================
	// 📦 2. 解析推送信息
	// GitHub 的推送事件 body 里包含：
	// - ref：推送的分支（如 "refs/heads/main"）
	// - after：最新提交的 SHA
	// ============================================================
	var payload struct {
		Ref   string `json:"ref"`
		After string `json:"after"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return
	}

	// 检查分支是否匹配
	expectedBranch := proj.Branch
	if expectedBranch == "" {
		expectedBranch = "main" // 默认 main 分支
	}

	// ref 的格式是 "refs/heads/main"
	// 所以用 HasSuffix 检查
	if !strings.HasSuffix(payload.Ref, "/"+expectedBranch) {
		c.JSON(http.StatusOK, gin.H{"message": "ignored push to different branch"})
		return
	}

	// ============================================================
	// 🔒 3. 防抖逻辑（Thundering Herd Defense）
	// 如果频繁推送，可能同时触发多个 webhook。
	// 检查是否有正在进行的部署，有就不重复创建了。
	// ============================================================
	var activeCount int
	err = h.db.QueryRow(
		"SELECT COUNT(*) FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND status IN ('pending', 'deploying')",
		projectID, envID,
	).Scan(&activeCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error checking task status"})
		return
	}
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "a deployment is already running for this project and environment"})
		return
	}

	// ============================================================
	// 🚀 4. 创建部署任务并提交到引擎
	// ============================================================
	commitID := payload.After
	if commitID == "" {
		commitID = "HEAD"
	}

	releaseName := time.Now().Format("20060102150405")
	username := "github-webhook"
	var userID int64 = 0 // 系统账号或特殊的 Webhook 用户 ID

	insertSQL := `
		INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := h.db.Exec(insertSQL, projectID, envID, commitID, string(domain.StatusPending), releaseName, userID, username, "{}", time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	taskID, _ := res.LastInsertId()

	// 创建日志目录
	logDir := h.config.Global.LogPath
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, fmt.Sprintf("task_%d.log", taskID))

	// 提交到部署引擎
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

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是 Webhook？
//    A: 别人家程序调用你家程序的接口！GitHub 推送代码时通知部署系统~
//
// 2. Q: 为什么要验证签名？
//    A: 防止别人伪造 GitHub 的请求！签名验证确保是真实的 GitHub 发来的~
//
// 中级：
// 3. Q: 什么是"时序攻击"（Timing Attack）？
//    A: 通过比较耗时来猜测密码。如果"== "比到第一位不同就返回，
//       攻击者可以通过计时猜到签名内容。hmac.Equal 固定时间比较防这个~
//
// 4. Q: 什么是"防抖"？
//    A: 防止短时间内多次触发。如果已经有一个部署在跑，新来的就忽略~
//
// 高级：
// 5. Q: 为什么 webhook 用户 ID 是 0？
//    A: 系统触发而非用户触发。0 表示"系统自动操作"，
//       和正常用户登录触发的部署区分开，方便审计~
// ============================================================
