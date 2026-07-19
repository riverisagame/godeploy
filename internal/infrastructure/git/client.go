package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Client 封装本地 git 命令行操作
type Client struct {
	workspaceBase string // 工作区根目录，如 "./workspace"
}

// NewClient 创建 Git 客户端
// @Ref: docs/sps/plans/20260719_p0_deploy_gaps_plan.md S3 | @Date: 2026-07-19
func NewClient(workspaceBase string) *Client {
	return &Client{workspaceBase: workspaceBase}
}

// CloneOrPull 拉取或更新代码到本地 workspace
// 如果 workspace 已存在则 fetch + reset，否则 clone
func (c *Client) CloneOrPull(repoURL, branch, projectName string, logChan chan<- string) (string, error) {
	workspacePath := filepath.Join(c.workspaceBase, projectName)

	if _, err := os.Stat(filepath.Join(workspacePath, ".git")); err == nil {
		// 目录已存在，执行 fetch + checkout + reset
		logChan <- fmt.Sprintf("[Git] Fetching %s (branch: %s)...\n", projectName, branch)
		if err := c.runGit(workspacePath, logChan, "fetch", "origin", branch); err != nil {
			return "", fmt.Errorf("git fetch failed: %w", err)
		}
		if err := c.runGit(workspacePath, logChan, "checkout", branch); err != nil {
			return "", fmt.Errorf("git checkout failed: %w", err)
		}
		if err := c.runGit(workspacePath, logChan, "reset", "--hard", "origin/"+branch); err != nil {
			return "", fmt.Errorf("git reset failed: %w", err)
		}
		logChan <- fmt.Sprintf("[Git] Updated %s to latest origin/%s\n", projectName, branch)
	} else {
		// 目录不存在，执行 clone
		logChan <- fmt.Sprintf("[Git] Cloning %s (branch: %s)...\n", repoURL, branch)
		if err := os.MkdirAll(c.workspaceBase, 0755); err != nil {
			return "", fmt.Errorf("mkdir workspace failed: %w", err)
		}
		if err := c.runGit("", logChan, "clone", "--depth=1", "-b", branch, repoURL, workspacePath); err != nil {
			return "", fmt.Errorf("git clone failed: %w", err)
		}
		logChan <- fmt.Sprintf("[Git] Cloned %s successfully\n", projectName)
	}

	return workspacePath, nil
}

// runGit 执行 git 命令并流式输出到 logChan
func (c *Client) runGit(dir string, logChan chan<- string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// 合并 stdout 和 stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("[Git] Failed to start: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		logChan <- "[Git] " + scanner.Text() + "\n"
	}

	return cmd.Wait()
}
