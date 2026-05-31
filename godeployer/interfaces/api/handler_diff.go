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

	if status == string(domain.StatusDeploying) || status == string(domain.StatusPending) {
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

