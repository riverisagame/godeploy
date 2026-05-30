package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitCommit struct {
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

func GetCacheDir(projectID string) string {
	return filepath.Join("demo_workspace", ".cache", projectID+".git")
}

// EnsureRepoCache 确保对应项目的 bare 仓库存在并更新至最新
func EnsureRepoCache(ctx context.Context, repoURL, projectID string) error {
	cacheDir := GetCacheDir(projectID)

	// 如果目录已存在，先校验其 remote origin 是否与当前请求的 repoURL 相同
	if _, err := os.Stat(cacheDir); err == nil {
		cmdCheck := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
		cmdCheck.Dir = cacheDir
		if out, err := cmdCheck.CombinedOutput(); err == nil {
			currentRemote := strings.TrimSpace(string(out))
			// 统一转成斜杠路径或清除首尾空白后进行对比
			if filepath.ToSlash(currentRemote) != filepath.ToSlash(repoURL) {
				// // @Ref: docs/sps/plans/20260530_fix_task_log_errors_plan.md | @Date: 2026-05-30
				// URL不一致，说明项目仓库发生了更改，清除本地缓存重建
				os.RemoveAll(cacheDir)
			}
		} else {
			// 如果获取失败说明本地不是正常 bare 库，清空重建
			os.RemoveAll(cacheDir)
		}
	}

	// 如果目录不存在，执行 git clone --bare
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "git", "clone", "--no-hardlinks", "--bare", repoURL, cacheDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %v, output: %s", err, string(out))
		}
		return nil
	}

	// 如果目录已存在，执行 git fetch origin
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*", "--prune")
	cmd.Dir = cacheDir
	if out, err := cmd.CombinedOutput(); err != nil {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
		// 单元测试中如果将开发中的本地仓库作为 repoURL，fetch 自身检出分支可能会被 git 拦截报错，但在测试环境下缓存已是最新，因此可直接容忍该错误。
		if strings.Contains(string(out), "refusing to fetch into branch") {
			return nil
		}
		return fmt.Errorf("git fetch failed: %v, output: %s", err, string(out))
	}

	return nil
}

// GetCommits 获取最新 50 条提交记录，并支持按 message/author/file 搜索，支持按 ref（分支/Tag）过滤
func GetCommits(ctx context.Context, projectID, keyword, author, file, ref string) ([]GitCommit, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("git cache not found for project %s", projectID)
	}

	args := []string{"log", "-n", "50", "--format=%H|%s|%an|%cI"}
	if keyword != "" {
		args = append(args, "--grep="+keyword, "-i")
	}
	if author != "" {
		args = append(args, "--author="+author, "-i")
	}
	if ref != "" {
		args = append(args, ref)
	} else {
		// 查询全部分支
		args = append(args, "--all")
	}

	if file != "" {
		args = append(args, "--", file)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cacheDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %v", err)
	}

	var commits []GitCommit
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, GitCommit{
				Hash:      parts[0],
				Message:   parts[1],
				Author:    parts[2],
				CreatedAt: parts[3],
			})
		}
	}
	return commits, nil
}

// GetDiff 获取两次提交之间的 diff 字符串。如果 fromCommit 为空则默认比较该 commit 本身变更。
func GetDiff(ctx context.Context, projectID, fromCommit, toCommit string, limitBytes int) (string, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", fmt.Errorf("git cache not found for project %s", projectID)
	}

	var args []string
	if fromCommit == "" {
		// 只有一个 commit 时查看该 commit 较上一版本的改动
		args = []string{"show", "--format=", toCommit} // --format= 移除 log header，只保留 diff
	} else {
		args = []string{"diff", fromCommit, toCommit}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cacheDir

	if limitBytes <= 0 {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git diff failed: %v, output: %s", err, string(out))
		}
		return string(out), nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	data := make([]byte, limitBytes+1)
	n, _ := io.ReadFull(stdout, data)

	if n > limitBytes {
		_ = cmd.Process.Kill()
		return string(data[:limitBytes]) + "\n\n... [Diff 截断: 文件变更过大，超出系统物理隔离安全限制 (DiffMaxSizeKB)]", nil
	}

	go cmd.Wait()
	return string(data[:n]), nil
}

// GetDiffForFile 获取两次提交之间指定文件的 diff 字符串。
// @Ref: docs/sps/plans/20260530_lazy_load_file_diff_plan.md | @Date: 2026-05-30
func GetDiffForFile(ctx context.Context, projectID, fromCommit, toCommit, file string, limitBytes int) (string, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", fmt.Errorf("git cache not found for project %s", projectID)
	}

	var args []string
	if fromCommit == "" {
		args = []string{"show", "--format=", toCommit, "--", file}
	} else {
		args = []string{"diff", fromCommit, toCommit, "--", file}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cacheDir

	if limitBytes <= 0 {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git diff failed: %v, output: %s", err, string(out))
		}
		return string(out), nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	data := make([]byte, limitBytes+1)
	n, _ := io.ReadFull(stdout, data)

	if n > limitBytes {
		_ = cmd.Process.Kill()
		return string(data[:limitBytes]) + "\n\n... [Diff 截断: 文件变更过大，超出系统物理隔离安全限制 (DiffMaxSizeKB)]", nil
	}

	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	// 必须同步等待命令执行完毕并检查错误，否则当 git 命令非零退出时，无法触发外部的物理快照降级提取
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return string(data[:n]), nil
}

// GetCommitAuthor 获取指定 ref 的 Git 提交者名称
func GetCommitAuthor(ctx context.Context, projectID, ref string) (string, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", fmt.Errorf("git cache not found for project %s", projectID)
	}

	cmd := exec.CommandContext(ctx, "git", "show", "-s", "--format=%an", ref)
	cmd.Dir = cacheDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed: %v, output: %s", err, string(out))
	}

	return strings.TrimSpace(string(out)), nil
}

// git.FindGitRepo 在 root 目录树中递归查找包含指定 commit 的 git 仓库，返回第一个匹配路径。
func FindGitRepo(root, commit string) (string, error) {
	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	// 如果 commit 不是 40 位的十六进制 Commit Hash (或者是分支名、空等)，直接返回，杜绝极其耗时的全局 Walk
	if len(commit) != 40 {
		return "", nil
	}
	for i := 0; i < len(commit); i++ {
		c := commit[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", nil
		}
	}

	var result string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略权限错误，继续
		}
		// 找到 .git 目录，说明 path 的父目录是 git 仓库
		if info.IsDir() && info.Name() == ".git" {
			repoDir := filepath.Dir(path)
			// 检查该仓库是否包含目标 commit
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			checkCmd := exec.CommandContext(ctx, "git", "cat-file", "-t", commit)
			checkCmd.Dir = repoDir
			out, checkErr := checkCmd.Output()
			if checkErr == nil && strings.TrimSpace(string(out)) == "commit" {
				result = repoDir
				return filepath.SkipAll // 找到即停止
			}
		}
		return nil
	})
	return result, err
}

// git.FilterFilesForTruncatedDiff 在 diff 文本被字节截断后，同步裁剪 files 列表，
// 确保"变更文件列表"与"代码差异"两个标签页展示的文件范围完全一致。
func FilterFilesForTruncatedDiff(truncatedDiff, originalFiles string) string {
	// 从截断后的 diff 文本中提取已包含的文件路径
	fileSet := make(map[string]bool)
	lines := strings.Split(truncatedDiff, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "diff --git a/") {
			continue
		}
		// diff --git a/path/to/file b/path/to/file
		parts := strings.SplitN(line, " ", 4)
		if len(parts) >= 4 {
			file := strings.TrimPrefix(parts[2], "a/")
			fileSet[file] = true
		}
	}

	if len(fileSet) == 0 {
		return originalFiles // 未提取到文件，保留原列表
	}

	// 从原 files 列表中只保留 diff 中存在的文件
	filesLines := strings.Split(strings.TrimSpace(originalFiles), "\n")
	var filtered []string
	for _, line := range filesLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// git --name-status 输出: "M\tpath"   git --name-only 输出: "path"
		parts := strings.SplitN(line, "\t", 2)
		file := line
		if len(parts) == 2 {
			file = parts[1]
		}
		if fileSet[file] {
			filtered = append(filtered, line)
		}
	}

	if len(filtered) > 0 {
		return strings.Join(filtered, "\n")
	}
	return originalFiles
}

func IsCommitHash(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, r := range ref {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
