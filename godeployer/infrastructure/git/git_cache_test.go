package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureRepoCacheRemoteChange 验证当项目仓库 URL 改变时，本地 Bare 缓存能自动清理旧缓存并对齐新 URL
// // @Ref: docs/sps/plans/20260530_fix_task_log_errors_plan.md | @Date: 2026-05-30
func TestEnsureRepoCacheRemoteChange(t *testing.T) {
	ctx := context.Background()

	// 1. 创建临时的工作目录，并确保物理无污染
	tmpDir, err := os.MkdirTemp("", "godeploy_git_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectID := "test-cache-proj-temp"
	cacheDir := GetCacheDir(projectID)
	// 测试结束清理测试缓存目录
	defer os.RemoveAll(cacheDir)

	// 2. 初始化模拟仓库 A
	repoADir := filepath.Join(tmpDir, "repoA")
	if err := os.MkdirAll(repoADir, 0755); err != nil {
		t.Fatalf("failed to create repoA: %v", err)
	}
	runGit(t, repoADir, "init")
	runGit(t, repoADir, "config", "user.name", "Test A")
	runGit(t, repoADir, "config", "user.email", "a@test.com")
	if err := os.WriteFile(filepath.Join(repoADir, "file.txt"), []byte("content A"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoADir, "add", ".")
	runGit(t, repoADir, "commit", "-m", "feat: init A")

	// 3. 初始化模拟仓库 B
	repoBDir := filepath.Join(tmpDir, "repoB")
	if err := os.MkdirAll(repoBDir, 0755); err != nil {
		t.Fatalf("failed to create repoB: %v", err)
	}
	runGit(t, repoBDir, "init")
	runGit(t, repoBDir, "config", "user.name", "Test B")
	runGit(t, repoBDir, "config", "user.email", "b@test.com")
	if err := os.WriteFile(filepath.Join(repoBDir, "file.txt"), []byte("content B"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoBDir, "add", ".")
	runGit(t, repoBDir, "commit", "-m", "feat: init B")

	// 4. 第一阶段：对 repoA 执行缓存
	t.Logf("Caching repoA: %s", repoADir)
	if err := EnsureRepoCache(ctx, repoADir, projectID); err != nil {
		t.Fatalf("first EnsureRepoCache failed: %v", err)
	}

	// 验证 origin 应该为 repoA 的绝对路径
	outA, err := exec.Command("git", "-C", cacheDir, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("get remote URL failed: %v, output: %s", err, string(outA))
	}
	actualA := strings.TrimSpace(string(outA))
	// 统一转成 slash 来比对路径以适配 Windows/Linux
	if filepath.ToSlash(actualA) != filepath.ToSlash(repoADir) {
		t.Fatalf("expected origin URL to be %q, got %q", repoADir, actualA)
	}

	// 5. 第二阶段：在不删除缓存的情况下，将 repo 变更为 repoB 的路径，触发 EnsureRepoCache
	t.Logf("Caching repoB (remote changed): %s", repoBDir)
	if err := EnsureRepoCache(ctx, repoBDir, projectID); err != nil {
		t.Fatalf("second EnsureRepoCache failed: %v", err)
	}

	// 验证 origin 必须被更新为 repoB 的路径
	outB, err := exec.Command("git", "-C", cacheDir, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("get remote URL B failed: %v, output: %s", err, string(outB))
	}
	actualB := strings.TrimSpace(string(outB))
	if filepath.ToSlash(actualB) != filepath.ToSlash(repoBDir) {
		t.Errorf("expected origin URL B to be %q, got %q (URL mismatched, test fails in RED phase)", repoBDir, actualB)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed in %s: %v, output: %s", args, dir, err, string(out))
	}
}
