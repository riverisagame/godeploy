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

// CloneForDeploy prepares a deployment workspace using a git bare repo and worktree.
// This allows O(1) concurrent checkouts and caches the repo locally.
// @Ref: docs/sps/plans/20260721_v2.5_refactoring_ir.md | @Date: 2026-07-21
func (c *Client) CloneForDeploy(repoURL, branch, projectName string, deployID uint, logChan chan<- string) (string, error) {
	lock := c.getLock(projectName)
	lock.Lock()
	defer lock.Unlock()

	bareRepoPath := filepath.Join(c.workspaceBase, projectName+".git")
	deployPath := filepath.Join(c.workspaceBase, fmt.Sprintf("%s_deploy_%d", projectName, deployID))

	// 1. Ensure bare repo exists
	if _, err := os.Stat(bareRepoPath); err != nil {
		logChan <- fmt.Sprintf("[Git] Initialize bare repo %s...\n", repoURL)
		if err := os.MkdirAll(bareRepoPath, 0755); err != nil {
			return "", fmt.Errorf("mkdir bare repo failed: %w", err)
		}
		if err := c.runGit("", logChan, "clone", "--bare", repoURL, bareRepoPath); err != nil {
			return "", fmt.Errorf("git bare clone failed: %w", err)
		}
	} else {
		logChan <- fmt.Sprintf("[Git] Fetching latest from origin...\n")
		// 必须指定 refspec 来强制更新本地的引用，或者直接 fetch 默认
		if err := c.runGit(bareRepoPath, logChan, "fetch", "origin", "+refs/heads/*:refs/heads/*", "--prune"); err != nil {
			logChan <- fmt.Sprintf("[Git] Fetch failed, ignoring... %v\n", err)
		}
	}

	// 2. Remove existing worktree if left over
	if _, err := os.Stat(deployPath); err == nil {
		os.RemoveAll(deployPath)
		c.runGit(bareRepoPath, logChan, "worktree", "prune")
	}

	// 3. Create worktree detached at target branch
	logChan <- fmt.Sprintf("[Git] Checking out branch %s to worktree...\n", branch)
	if err := c.runGit(bareRepoPath, logChan, "worktree", "add", "--detach", deployPath, branch); err != nil {
		return "", fmt.Errorf("git worktree add failed: %w", err)
	}

	return deployPath, nil
}

// CleanupDeploy removes the temporary worktree after deployment finishes.
func (c *Client) CleanupDeploy(projectName string, deployID uint, deployPath string) error {
	bareRepoPath := filepath.Join(c.workspaceBase, projectName+".git")
	
	// Remove directory
	if err := os.RemoveAll(deployPath); err != nil {
		return err
	}
	
	// Prune worktree
	// We run it silently since logChan might be closed
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = bareRepoPath
	_ = cmd.Run()
	
	return nil
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
