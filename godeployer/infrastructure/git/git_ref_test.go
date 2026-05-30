package git

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestGetCommits_WithRef(t *testing.T) {
	// Create a temporary directory for the mock project cache
	cacheDir := GetCacheDir("test_ref_project")
	defer os.RemoveAll(cacheDir)

	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Initialize a bare git repository
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = cacheDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init bare repo: %s", string(out))
	}

	// Configure git for the test repo
	_ = exec.Command("git", "-C", cacheDir, "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "-C", cacheDir, "config", "user.email", "test@test.com").Run()

	// To create a commit in a bare repo without a working tree:
	// 1. Create an empty tree
	out, _ := exec.Command("git", "-C", cacheDir, "mktree").Output()
	treeHash := string(out[:len(out)-1]) // remove newline

	// 2. Commit the tree
	cmdCommit := exec.Command("git", "-C", cacheDir, "commit-tree", treeHash, "-m", "Initial commit")
	out, err = cmdCommit.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create commit: %s", string(out))
	}
	commitHash := string(out[:len(out)-1])

	// 3. Update refs/heads/master
	_ = exec.Command("git", "-C", cacheDir, "update-ref", "refs/heads/master", commitHash).Run()

	// Now call GetCommits
	_, err = GetCommits(context.Background(), "test_ref_project", "", "", "", "master")
	if err != nil {
		t.Errorf("GetCommits failed: %v", err)
	}
}
