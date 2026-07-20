package git_test

import (
	"os"
	"os/exec"
	"pdeploy/internal/infrastructure/git"
	"testing"
)

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
	os.WriteFile(path+"/file1.txt", []byte("hello"), 0644)
	runCmd(t, path, "git", "add", ".")
	runCmd(t, path, "git", "commit", "-m", "first commit")
	
	// Create second commit
	os.WriteFile(path+"/file2.txt", []byte("world"), 0644)
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
