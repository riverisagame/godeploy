package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"github.com/riverisagame/godeploy/internal/infrastructure/git"
	"testing"
)

func TestClient_CloneOrPullWorktree(t *testing.T) {
	workspace := t.TempDir()
	client := git.NewClient(workspace)

	// Create remote mock repo
	remoteRepo := t.TempDir()
	createMockRepo(t, remoteRepo)

	projectName := "test-worktree"
	deployID1 := uint(100)
	deployID2 := uint(101)
	
	logChan1 := make(chan string, 100)
	logChan2 := make(chan string, 100)

	// Deploy 1
	path1, err := client.CloneForDeploy(remoteRepo, "main", projectName, deployID1, logChan1)
	if err != nil {
		t.Fatalf("CloneForDeploy 1 failed: %v", err)
	}
	
	if _, err := os.Stat(filepath.Join(path1, "file1.txt")); os.IsNotExist(err) {
		t.Errorf("expected file1.txt in worktree 1")
	}

	// Deploy 2 (concurrent simulation)
	path2, err := client.CloneForDeploy(remoteRepo, "main", projectName, deployID2, logChan2)
	if err != nil {
		t.Fatalf("CloneForDeploy 2 failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(path2, "file2.txt")); os.IsNotExist(err) {
		t.Errorf("expected file2.txt in worktree 2")
	}
	
	if path1 == path2 {
		t.Errorf("expected different paths for concurrent deploys, got %s", path1)
	}

	// Cleanup
	err = client.CleanupDeploy(projectName, deployID1, path1)
	if err != nil {
		t.Errorf("CleanupDeploy 1 failed: %v", err)
	}
	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Errorf("expected path1 to be removed")
	}
}

func TestClient_FetchAndGetCommits(t *testing.T) {
	// 这是一个集成测试，需要真实的或模拟的 git 环境
	// 为了零污染和不依赖外部，我们可以在本地临时创建一个 Git 仓库
	workspace := t.TempDir()
	client := git.NewClient(workspace)

	// 创建远程 mock repo
	remoteRepo := t.TempDir()
	createMockRepo(t, remoteRepo)

	projectName := "test-diff-project"

	// 测试首次拉取 (由于没有 fromCommit，应该返回所有或最近10条)
	commits, err := client.FetchAndGetCommits(remoteRepo, "main", projectName, "")
	if err != nil {
		t.Fatalf("FetchAndGetCommits failed: %v", err)
	}

	if len(commits) == 0 {
		t.Errorf("Expected commits, got 0")
	}

	// 测试带 fromCommit (获取第二条以后的)
	// 假设我们取第一个返回的 hash 作为 target，看看能不能获取增量
	if len(commits) >= 2 {
		fromCommit := commits[len(commits)-1].Hash // 最老的 commit
		diffCommits, err := client.FetchAndGetCommits(remoteRepo, "main", projectName, fromCommit)
		if err != nil {
			t.Fatalf("FetchAndGetCommits with fromCommit failed: %v", err)
		}
		if len(diffCommits) == 0 {
			t.Errorf("Expected diff commits, got 0")
		}
	}
}

func createMockRepo(t *testing.T, path string) {
	runCmd(t, path, "git", "init", "-b", "main")
	runCmd(t, path, "git", "config", "user.name", "TestUser")
	runCmd(t, path, "git", "config", "user.email", "test@test.com")
	
	// Create first commit
	_ = os.WriteFile(path+"/file1.txt", []byte("hello"), 0644)
	runCmd(t, path, "git", "add", ".")
	runCmd(t, path, "git", "commit", "-m", "first commit")
	
	// Create second commit
	_ = os.WriteFile(path+"/file2.txt", []byte("world"), 0644)
	runCmd(t, path, "git", "add", ".")
	runCmd(t, path, "git", "commit", "-m", "second commit")
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runCmd %s %v failed: %v, output: %s", name, args, err, string(out))
	}
}
