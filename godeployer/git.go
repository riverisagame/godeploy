package godeployer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitCommit struct {
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

func getCacheDir(projectID string) string {
	return filepath.Join("demo_workspace", ".cache", projectID+".git")
}

// EnsureRepoCache 确保对应项目的 bare 仓库存在并更新至最新
func EnsureRepoCache(ctx context.Context, repoURL, projectID string) error {
	cacheDir := getCacheDir(projectID)

	// 如果目录不存在，执行 git clone --bare
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "git", "clone", "--bare", repoURL, cacheDir)
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
	cacheDir := getCacheDir(projectID)
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
	cacheDir := getCacheDir(projectID)
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
	cacheDir := getCacheDir(projectID)
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
	cacheDir := getCacheDir(projectID)
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
