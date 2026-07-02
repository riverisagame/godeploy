// ============================================================
// 文件：handler_rollback.go
// 作用：↩️ 回滚 API——一键回到上个版本！
//
// 当新版本出问题时，管理员可以"回滚"到上一个成功的版本。
// 回滚不是重新部署旧代码，而是直接把 symlink 指回旧目录！
// 所以回滚非常快——毫秒级完成！
//
// 给初二小白的比喻：
// 就像玩游戏打 boss：
// - 新版本 = 你冲上去打 boss
// - 打不过 = 回滚到安全点
// - 回滚 = 读档（Loading……成功！）
// symlink 切换就像"读档"一样快！
// ============================================================

package api

import (
	"net/http"
	"strconv"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"

	"github.com/gin-gonic/gin"
)

// HandleTriggerRollback 触发版本回滚到特定历史任务版本
//
// 用户请求 POST /api/tasks/:id/rollback
// 其中 :id 是要"回滚到"的目标任务 ID（不是回滚当前任务！）
//
// 逻辑：
// 1. 根据任务 ID 找到项目和环境
// 2. 检查用户权限
// 3. 对每个目标服务器执行 RollbackToTask
// 4. RollbackToTask 把 symlink 切回目标版本
//
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func (h *APIHandler) HandleTriggerRollback(c *gin.Context) {
	// 从 URL 参数中取出任务 ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	// 查询任务的项目和环境
	var projectID, envID string
	err = h.db.QueryRow(
		"SELECT project_id, env_id FROM deploy_tasks WHERE id = ?", id,
	).Scan(&projectID, &envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 检查项目配置是否存在
	proj, exists := h.config.Projects[projectID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 👤 权限检查
	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	// 找到对应的环境配置
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

	// 🏃 执行回滚
	// 创建一个新的部署引擎（不经过队列，直接执行回滚）
	engine := application.NewDeployEngine(h.taskRepo, h.executor, nil)

	// 对每台服务器执行回滚
	for _, srv := range targetEnv.Servers {
		if err := engine.RunRollbackToTask(id, srv); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "rollback completed"})
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 回滚和重新部署有什么区别？
//    A: 回滚只是切换 symlink，毫秒级完成！
//       重新部署要 clone + build + rsync，可能要好几分钟。
//       回滚就像"读档"，重新部署就像"重新打一遍"~
//
// 2. Q: 回滚会丢失数据吗？
//    A: 不会！旧版本的代码和数据目录还在服务器上。
//       回滚只是把"快捷方式"指回去~
//
// 中级：
// 3. Q: 为什么回滚不走部署队列（SubmitJob）？
//    A: 回滚只需要切换 symlink，非常快且风险低。
//       不需要经过完整的部署流水线（clone-build-rsync）。
//       直接执行就像"快速通道"~
//
// 高级：
// 4. Q: 如果回滚的目标版本对应的 releases 目录被清理了怎么办？
//    A: 这是目前的一个限制！如果 keep_releases 设得很小，
//       旧版本被清理后回滚就无效了。需要确保保留足够的版本数~
// ============================================================
