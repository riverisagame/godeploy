package godeployer

import (
	"context"
	"fmt"
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
func GetDiff(ctx context.Context, projectID, fromCommit, toCommit string) (string, error) {
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
	out, err := cmd.CombinedOutput() // 包含 stderr 在内，如果失败便于排查
	if err != nil {
		return "", fmt.Errorf("git diff failed: %v, output: %s", err, string(out))
	}

	return string(out), nil
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
