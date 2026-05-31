package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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

	// 执行"先库后盘"第一步：从数据库删除
	if len(idsToPrune) > 0 {
		for _, id := range idsToPrune {
			_, err := h.db.Exec("DELETE FROM deploy_tasks WHERE id = ?", id)
			if err == nil {
				prunedTasksCount++
			}
		}
	}

	// 执行"先库后盘"第二步：删除对应的物理文件，并累计释放大小
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
