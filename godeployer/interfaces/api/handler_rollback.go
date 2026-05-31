package api

import (
	"net/http"
	"strconv"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"

	"github.com/gin-gonic/gin"
)

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
	engine := application.NewDeployEngine(h.taskRepo, h.executor, nil)

	// 这里支持为每个服务器逐一精准回滚到目标 task
	for _, srv := range targetEnv.Servers {
		if err := engine.RunRollbackToTask(id, srv); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "rollback completed"})
}
