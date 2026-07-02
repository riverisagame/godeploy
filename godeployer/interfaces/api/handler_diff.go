// ============================================================
// 文件：handler_diff.go
// 作用：🔍 代码差异（Diff）查看 API！
//
// 这个文件提供了"查看代码变更"的各种 API：
// 1. HandleGetTaskDiff：查看某次部署改了什么
// 2. HandleGetProjectPreviewDiff：部署前预览要改什么
// 3. HandleGetProjectRefs：获取分支/标签列表
// 4. HandleGetProjectCommits：获取提交记录
//
// 什么是 Diff？
// Diff = 两次提交之间的"找不同"。
// 绿色+号 = 新增的代码，红色-号 = 删除的代码。
//
// 给初二小白的比喻：
// 就像比较两份作业📝：
// - 第一份（旧版本）："今天天气真好"
// - 第二份（新版本）："今天天气真好啊"
// - Diff 结果：在"好"后面加了"啊"！
// ============================================================

package api

import (
	"deploy/godeployer/infrastructure/git"
	"deploy/godeployer/infrastructure/sys"

	"deploy/godeployer/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 📋 HandleGetTaskDiff：查看某次部署的代码差异
//
// 用户可以在部署历史页面中查看每次部署改了什么代码。
// 支持两种 diff 类型：
// - live：线上对比（当前版本 vs 上一个成功版本）
// - git_log：Git 提交对比（基于 git 历史）
// 还支持查看单个文件的 diff（懒加载）。
// ============================================================

// HandleGetTaskDiff 获取当前任务与其前一个成功部署版本之间的 Git Diff
func (h *APIHandler) HandleGetTaskDiff(c *gin.Context) {
	// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
	// 🚦 并发限流：同时只能有 5 个 diff 请求，超时 3 秒
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

	// 1️⃣ 查询当前任务信息
	var projectID, envID, currentCommit, releaseName, status, createdAt, targetType string
	err = h.db.QueryRow(
		"SELECT project_id, env_id, commit_id, release_name, status, created_at, target_type FROM deploy_tasks WHERE id = ?", id,
	).Scan(&projectID, &envID, &currentCommit, &releaseName, &status, &createdAt, &targetType)
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

	// 正在部署中的任务，diff 还没准备好
	if status == string(domain.StatusDeploying) || status == string(domain.StatusPending) {
		c.JSON(http.StatusConflict, gin.H{"error": "task is still deploying, diff not ready"})
		return
	}

	// 2️⃣ 查询上一个成功版本的 commit
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

	// 计算 diff 缓存的目录路径
	createdYM := "default"
	if len(createdAt) >= 7 {
		createdYM = strings.ReplaceAll(createdAt[:7], "-", "")
	}
	diffCacheDir := filepath.Join(h.config.Global.LogPath, "diffs", "projects", projectID, createdYM)
	diffCacheFile := filepath.Join(diffCacheDir, fmt.Sprintf("task_%d_diff.log", id))

	// 3️⃣ 确定 git 仓库目录
	buildPath := filepath.Join(h.config.Global.WorkspacePath, projectID, releaseName)
	gitRepoPath := buildPath
	if _, statErr := os.Stat(filepath.Join(buildPath, ".git")); os.IsNotExist(statErr) {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
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

	// 4️⃣ 如果指定了 file 参数，只获取单个文件的 diff
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
				// 如果 git diff 失败，尝试从缓存的 JSON 快照中提取
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

	// 5️⃣ 没有 file 参数，只返回文件列表（diff 内容懒加载）
	// 先尝试读取缓存
	if data, readErr := os.ReadFile(diffCacheFile); readErr == nil {
		var cacheObj struct {
			Files      string `json:"files"`
			Diff       string `json:"diff"`
			GitLogDiff string `json:"git_log_diff"`
		}
		if jsonErr := json.Unmarshal(data, &cacheObj); jsonErr == nil {
			c.JSON(http.StatusOK, gin.H{
				"files": cacheObj.Files,
				"diff":  "",
			})
			return
		}
	}

	// 缓存没有，直接 git diff --name-status 查询
	var filesListStr string
	filesCmd := exec.CommandContext(ctx, "git", "diff", "--name-status", prevCommit, currentCommit)
	filesCmd.Dir = gitRepoPath
	if filesOut, filesErr := filesCmd.CombinedOutput(); filesErr == nil {
		filesListStr = string(filesOut)
	} else {
		filesListStr = "获取变更文件列表失败"
	}

	// 写入缓存文件（只记录文件列表，diff 留空——真正的懒加载）
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

// HandleGetProjectPreviewDiff 部署前预览代码变更
// 用户在选择要部署的版本时，可以预览改了什么
func (h *APIHandler) HandleGetProjectPreviewDiff(c *gin.Context) {
	// 🚦 并发限流
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

	// 权限检查
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
		// 自动查询上次成功部署的 commit 作为对比基准
		_ = h.db.QueryRow(
			"SELECT commit_id FROM deploy_tasks WHERE project_id = ? AND env_id = ? AND status = 'success' ORDER BY id DESC LIMIT 1",
			projectID, envID,
		).Scan(&fromCommit)
	}

	diffType := c.DefaultQuery("diff_type", "live")
	fileParam := c.Query("file")
	if fileParam != "" {
		// 获取单个文件的 diff
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

	// 更新缓存
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

	// 只返回文件列表，不返回 diff 内容（懒加载）
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
	// 限制最多 2000 个文件，防止前端渲染卡死
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

// HandleGetProjectRefs 获取项目的 Git 分支和标签列表
// 前端部署页面需要展示"可选分支/标签"
func (h *APIHandler) HandleGetProjectRefs(c *gin.Context) {
	projectID := c.Param("id")
	proj, ok := h.config.Projects[projectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 权限检查
	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	// git ls-remote 列出远程仓库的所有分支和标签
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
				continue // 跳过注释标签
			}
			refs = append(refs, GitRef{Name: name, Type: "tag", Hash: hash})
		}
	}

	c.JSON(http.StatusOK, refs)
}

// HandleGetProjectCommits 获取项目的提交记录列表
// 前端部署页面需要展示"可选版本"
func (h *APIHandler) HandleGetProjectCommits(c *gin.Context) {
	projectID := c.Param("id")
	proj, ok := h.config.Projects[projectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// 权限检查
	usernameVal, _ := c.Get("username")
	if !h.checkProjectAccess(usernameVal.(string), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied for this project"})
		return
	}

	keyword := c.Query("q")
	author := c.Query("author")
	file := c.Query("file")
	ref := c.Query("ref")

	// 更新缓存
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

// extractFileDiffFromLog 从完整的 diff 文本中提取某个文件的 diff 片段
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

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: diff 类型 live 和 git_log 有什么区别？
//    A: live = 跟线上当前版本对比（看实际改了什么）
//       git_log = 跟 git 上一个提交对比（看这个提交改了什么）
//       比如连续部署了 3 个版本，live 显示跟最新版本的差异~
//
// 中级：
// 2. Q: 为什么要"懒加载"diff 内容？
//    A: diff 内容可能非常巨大（比如改了几百个文件）！
//       先只返回文件列表，用户点击某个文件时再加载那个文件的 diff~
//
// 3. Q: 并发限流 diffSemaphore 有什么用？
//    A: 防止太多人同时看 diff 把服务器搞崩！
//       git diff 是很消耗 CPU 的操作，限制同时只有 5 个人能用~
// ============================================================
