// ============================================================
// 文件：handler_deploy.go
// 作用：🎯 部署相关的 API 处理函数！
//
// 这个文件处理"用户点击部署按钮"后的所有操作：
// 1. HandleCreateTask：创建并提交部署任务
// 2. HandleGetTasks：查询历史任务列表
// 3. HandleGetTaskDetail：查看某个任务的详情
// 4. HandleGetTaskLog：查看部署日志
// 5. HandleWSLog：WebSocket 实时推送日志
//
// 给初二小白的比喻：
// 这就像一家餐厅的"点餐台"🍽️——
// - HandleCreateTask = 客人点菜（创建部署任务）
// - HandleGetTasks = 查看历史菜单
// - HandleGetTaskLog = 看厨房的做菜记录
// - HandleWSLog = 实时看厨师怎么做菜（直播！）
// ============================================================

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

// ============================================================
// 📋 CreateTaskRequest：创建任务时的请求体
//
// 用户通过 API 创建任务时需要提供：
// - project_id：部署哪个项目
// - env_id：部署到哪个环境
// - commit_id：部署哪个 Git 版本
// ============================================================

type CreateTaskRequest struct {
	ProjectID    string `json:"project_id" binding:"required"`    // 📁 项目 ID（必填）
	EnvID        string `json:"env_id" binding:"required"`         // 🌍 环境 ID（必填）
	CommitID     string `json:"commit_id" binding:"required"`      // 🔖 Git 版本（必填）
	TargetType   string `json:"target_type"`                       // 🎯 目标类型（commit/branch/tag）
	Description  string `json:"description"`                        // 💬 备注说明
	ExtraExclude string `json:"extra_exclude"`                      // 🚫 额外排除规则
}

// ============================================================
// 🚀 HandleCreateTask：创建部署任务！
//
// 这是最重要的 API 之一！用户点击"部署"按钮后调用它。
// 它做以下几件事：
// 1. 验证项目和配置是否存在
// 2. 检查用户有没有权限部署这个项目
// 3. 检查当前有没有其他部署正在进行
// 4. 如果限制了 Git 作者，验证提交者是否被允许
// 5. 在数据库创建一条部署任务记录
// 6. 把任务提交给部署引擎执行
// ============================================================

func (h *APIHandler) HandleCreateTask(c *gin.Context) {
	// 解析用户发来的 JSON 请求体
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 📁 检查项目是否存在
	proj, exists := h.config.Projects[req.ProjectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 🌍 检查环境是否存在
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

	// 👤 检查用户权限
	usernameVal, _ := c.Get("username")
	username := usernameVal.(string)
	if !h.checkProjectAccess(username, req.ProjectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	// 🔒 检查是否有正在进行的部署（同一个项目+环境不能同时部署两次）
	// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
	var activeCount int
	lockSQL := `
		SELECT COUNT(*) 
		FROM deploy_tasks 
		WHERE project_id = ? AND env_id = ? AND status IN ('pending', 'deploying')`
	_ = h.db.QueryRow(lockSQL, req.ProjectID, req.EnvID).Scan(&activeCount)
	if activeCount > 0 {
		// 🔴 已经有了，返回冲突错误
		c.JSON(http.StatusConflict, gin.H{
			"error": "another deployment is already in progress for this project and environment",
		})
		return
	}

	// 🆔 获取用户 ID 和绑定信息
	var userID int64
	var boundAuthors string
	var restrict bool
	_ = h.db.QueryRow(
		"SELECT id, COALESCE(bound_git_authors, ''), COALESCE(restrict_git_authors, 0) FROM users WHERE username = ?",
		username,
	).Scan(&userID, &boundAuthors, &restrict)

	// 🔐 如果用户开启了"Git 作者限制"，检查要部署的提交的作者是否在允许列表中
	if restrict {
		// 先更新缓存
		if err := git.EnsureRepoCache(c.Request.Context(), proj.Repo, req.ProjectID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update repo cache for auth check"})
			return
		}

		// 查询该提交的作者
		author, err := git.GetCommitAuthor(c.Request.Context(), req.ProjectID, req.CommitID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve commit author"})
			return
		}

		// 检查作者是否在允许列表中
		allowed := false
		for _, a := range strings.Split(boundAuthors, ",") {
			if strings.TrimSpace(a) == author {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Access Denied: you are not allowed to deploy commits authored by '%s'", author),
			})
			return
		}
	}

	// 📦 生成发布版本的名称（时间戳格式，精确到秒）
	releaseName := time.Now().Format("20060102150405") // 比如 20260601123456

	// 🎯 确定目标类型（是 commit 精确版本，还是分支/tag）
	targetType := req.TargetType
	if targetType == "" {
		if git.IsCommitHash(req.CommitID) {
			targetType = "commit" // 40 位 SHA → 是精确提交
		} else {
			targetType = "branch" // 否则是分支名/tag
		}
	}

	// 💾 在数据库中插入任务记录（初始状态 = pending）
	insertSQL := `
		INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, description, extra_exclude, target_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := h.db.Exec(insertSQL, req.ProjectID, req.EnvID, req.CommitID,
		string(domain.StatusPending), releaseName, userID, username, "{}",
		req.Description, req.ExtraExclude, targetType, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	taskID, _ := res.LastInsertId()

	// 📝 创建日志文件目录
	logDir := h.config.Global.LogPath
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, fmt.Sprintf("task_%d.log", taskID))

	// ⏰ 创建带 15 分钟超时的上下文
	// 如果部署超过 15 分钟还没完成，自动取消
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)

	// 📋 创建部署任务并提交到引擎
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

	// ✅ 返回创建成功的响应（HTTP 201 = Created）
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

	// 查询任务列表（最多 50 条）
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
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.EnvID, &t.CommitID, &t.Status,
			&t.ReleaseName, &t.Username, &t.Description, &t.ExtraExclude, &t.CreatedAt); err == nil {
			tasks = append(tasks, t)
		}
	}

	c.JSON(http.StatusOK, tasks)
}

// HandleGetTaskDetail 获取任务详情
func (h *APIHandler) HandleGetTaskDetail(c *gin.Context) {
	id := c.Param("id")
	var status, projectID string
	err := h.db.QueryRow(
		"SELECT status, project_id FROM deploy_tasks WHERE id = ?", id,
	).Scan(&status, &projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 权限检查
	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": status})
}

// HandleGetTaskLog 获取部署日志文件的内容
// 日志文件可能很大，所以做了 1MB 截断保护~
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleGetTaskLog(c *gin.Context) {
	id := c.Param("id")

	// 权限检查
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
	const maxLogSize = 1 * 1024 * 1024 // 1MB 上限

	if stat.Size() <= maxLogSize {
		// 文件不到 1MB，全部读取
		data, err = os.ReadFile(logFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 超过 1MB，只读取最后 1MB
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

// ============================================================
// 📡 WebSocket 实时日志推送
//
// WebSocket = 双向通信的"管道"
// 普通 HTTP：你问一句，服务器答一句（像发短信）
// WebSocket：建立连接后，服务端可以随时推送数据（像打电话）
//
// 这里的流程：
// 1. 客户端发起 WebSocket 连接
// 2. 客户端发送一个认证消息（包含 JWT token）
// 3. 服务端验证身份
// 4. 服务端每秒检查日志文件有没有新增内容
// 5. 有新内容就推送给客户端
// 6. 客户端实时看到部署日志！
// ============================================================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ✅ 允许跨域请求（开发环境）
	},
}

// HandleWSLog WebSocket 流式推送日志
// @Ref: docs/sps/plans/20260527_m6_frontend_ir.md | @Date: 2026-05-27
func (h *APIHandler) HandleWSLog(c *gin.Context) {
	// 升级 HTTP 连接为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 🔐 1. 认证：客户端必须先发送一个包含 token 的消息
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // 5 秒内必须发
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	err = conn.ReadJSON(&authMsg)
	if err != nil || authMsg.Type != "auth" {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation,
				"missing or invalid auth payload"))
		return
	}

	// 验证 token
	username, _, err := application.ParseToken(authMsg.Token, h.config.Global.JWTSecret)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"))
		return
	}

	// 重置读超时（后面只需要推送，不需要读了）
	conn.SetReadDeadline(time.Time{})

	id := c.Param("id")

	// 权限检查
	var projectID string
	err = h.db.QueryRow("SELECT project_id FROM deploy_tasks WHERE id = ?", id).Scan(&projectID)
	if err == nil && !h.checkProjectAccess(username, projectID) {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation,
				"access denied for this project"))
		return
	}

	logFilePath := filepath.Join(h.config.Global.LogPath, fmt.Sprintf("task_%s.log", id))

	// 📡 轮询推送：每秒检查一次日志文件有没有新内容
	// 就像 "tail -f" 命令的效果
	var lastPos int64 = 0
	ticker := time.NewTicker(1 * time.Second) // 每秒触发一次
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断开连接了
			return
		case <-ticker.C:
			// 每秒检查一次日志文件
			file, err := os.Open(logFilePath)
			if err != nil {
				// 日志文件可能还没创建（部署还没开始）
				continue
			}

			stat, err := file.Stat()
			if err != nil {
				file.Close()
				continue
			}

			currentSize := stat.Size()
			if currentSize < lastPos {
				// 文件被截断或重新创建了（比如新部署重新写日志）
				lastPos = 0
			}

			// 如果有新内容，读取并推送
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
					lastPos += int64(n) // 更新已读位置
				}
			}
			file.Close()
		}
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: WebSocket 和 HTTP 有什么区别？
//    A: HTTP 是"一问一答"（像发短信），WebSocket 是"随时说话"（像打电话）。
//       部署日志需要实时推送，WebSocket 比轮询更高效~
//
// 2. Q: 为什么日志要截断 1MB？
//    A: 防止日志文件太大把浏览器撑爆！只返回最后 1MB 就够看了~
//
// 中级：
// 3. Q: HandleCreateTask 中的并发锁机制是什么？
//    A: 先查数据库里有没有 pending/deploying 状态的任务。
//       如果有，说明正在部署中，不再创建新任务。防止同时部署冲突！
//
// 4. Q: 为什么要用 WebSocket 轮询日志文件而不是直接推送？
//    A: 部署引擎在另一个协程写日志文件，WebSocket 处理器没法直接获取。
//       每秒轮询文件变化是一种简单的"搭桥"方式~
//
// 高级：
// 5. Q: 为什么 WebSocket 要先发一个"auth"消息？
//    A: HTTP 升级为 WebSocket 时无法修改请求头！
//       所以不能在 URL 参数里带 token（会被日志记录），
//       必须在建立连接后通过第一条消息来验证身份~
// ============================================================
