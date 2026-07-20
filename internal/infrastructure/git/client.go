package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pdeploy/internal/domain"
	"strings"
	"sync"
)

// Client 封装本地 git 命令行操作
type Client struct {
	workspaceBase string // 工作区根目录，如 "./workspace"
	projectLocks  sync.Map
}

// NewClient 创建 Git 客户端
// @Ref: docs/sps/plans/20260719_p0_deploy_gaps_plan.md S3 | @Date: 2026-07-19
func NewClient(workspaceBase string) *Client {
	return &Client{workspaceBase: workspaceBase}
}

func (c *Client) getLock(projectName string) *sync.Mutex {
	lock, _ := c.projectLocks.LoadOrStore(projectName, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// CloneOrPull 拉取或更新代码到本地 workspace
// 如果 workspace 已存在则 fetch + reset，否则 clone
func (c *Client) CloneOrPull(repoURL, branch, projectName string, logChan chan<- string) (string, error) {
	lock := c.getLock(projectName)
	lock.Lock()
	defer lock.Unlock()

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
		// @Ref: docs/sps/plans/20260720_pre_deploy_diff_ir.md | @Date: 2026-07-20
		// 移除 --depth=1 以拉取全量历史
		if err := c.runGit("", logChan, "clone", "-b", branch, repoURL, workspacePath); err != nil {
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

// FetchAndGetCommits 拉取最新代码并获取从 fromCommit 到最新分支的提交记录
// @Ref: docs/sps/plans/20260720_pre_deploy_diff_ir.md | @Date: 2026-07-20
func (c *Client) FetchAndGetCommits(repoURL, branch, projectName, fromCommit string) ([]domain.CommitInfo, error) {
	lock := c.getLock(projectName)
	lock.Lock()
	defer lock.Unlock()

	workspacePath := filepath.Join(c.workspaceBase, projectName)

	if _, err := os.Stat(filepath.Join(workspacePath, ".git")); err != nil {
		// 不存在则克隆
		if err := os.MkdirAll(c.workspaceBase, 0755); err != nil {
			return nil, fmt.Errorf("mkdir workspace failed: %w", err)
		}
		cmd := exec.Command("git", "clone", "-b", branch, repoURL, workspacePath)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		// 存在则 fetch
		cmd := exec.Command("git", "fetch", "-q", "origin", branch)
		cmd.Dir = workspacePath
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("git fetch failed: %w", err)
		}
	}

	var args []string
	if fromCommit != "" {
		args = []string{"log", fmt.Sprintf("%s..origin/%s", fromCommit, branch), "--pretty=format:%H|%s|%an|%ad", "--date=iso"}
	} else {
		args = []string{"log", fmt.Sprintf("origin/%s", branch), "-n", "10", "--pretty=format:%H|%s|%an|%ad", "--date=iso"}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = workspacePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w, %s", err, string(out))
	}

	var commits []domain.CommitInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, domain.CommitInfo{
				Hash:    parts[0],
				Message: parts[1],
				Author:  parts[2],
				Date:    parts[3],
			})
		}
	}

	return commits, nil
}
