// ============================================================
// 文件：git.go
// 作用：🌿 Git 操作工具箱——管理代码仓库！
//
// Git 是一个非常强大的"版本控制"工具。
// 简单理解：就像玩游戏的"存档"功能——
// 你可以看到每次改了什么、谁改的、什么时候改的。
//
// 这个文件负责所有 Git 相关的操作：
// 1. 维护本地 bare 仓库缓存（加速克隆速度）
// 2. 查询提交记录（commit log）
// 3. 获取代码差异（diff——看看改了什么）
// 4. 查找仓库、验证提交等辅助功能
//
// 给初二小白的解释：
// - bare 仓库 = 一个"大仓库"，没有工作目录，专门用来 clone
// - commit = 一次"存档"，记录了当时代码的样子
// - diff = 两个版本之间的"找不同"
// ============================================================

package git

import (
	"context"       // 📡 上下文：控制超时
	"fmt"           // ✏️ 格式化
	"io"            // 📥 输入输出：读取 diff 流
	"os"            // 💻 文件系统操作
	"os/exec"       // 🖥️ 执行 Git 命令
	"path/filepath" // 📁 路径处理
	"strings"       // 📏 字符串处理
	"time"          // ⏰ 超时控制
)

// GitCommit 一次 Git 提交的信息
// 就像作业本上的一次批改记录——谁改的、改了什么内容、什么时候改的
type GitCommit struct {
	Hash      string `json:"hash"`       // 🆔 提交的唯一 ID（40 位十六进制）
	Message   string `json:"message"`    // 📝 提交信息（程序员写的"这次改了啥"）
	Author    string `json:"author"`     // 👤 作者名字
	CreatedAt string `json:"created_at"` // 📅 提交时间（ISO 格式）
}

// GetCacheDir 返回项目 bare 仓库的缓存目录路径
// bare 仓库 = 只有 git 历史没有工作文件的"轻量版仓库"
// 用它来 clone 比从远程仓库 clone 快 100 倍！
func GetCacheDir(projectID string) string {
	return filepath.Join("demo_workspace", ".cache", projectID+".git")
}

// ============================================================
// 🗃️ EnsureRepoCache：确保 bare 仓库缓存存在且最新
//
// 这个函数的逻辑像一个"智能缓存"：
// 1. 检查缓存目录是否存在
// 2. 如果存在，检查 remote URL 是否匹配（如果换了仓库地址就重建）
// 3. 如果不存在，从远程仓库 clone 一份 bare 仓库
// 4. 如果已存在，执行 git fetch 更新到最新
// ============================================================

// EnsureRepoCache 确保对应项目的 bare 仓库存在并更新至最新
func EnsureRepoCache(ctx context.Context, repoURL, projectID string) error {
	cacheDir := GetCacheDir(projectID)

	// --- 检查已有缓存 ---
	if _, err := os.Stat(cacheDir); err == nil {
		// ✅ 缓存目录存在
		// 检查它的 remote origin 是否跟我们要的一致
		cmdCheck := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
		cmdCheck.Dir = cacheDir
		if out, err := cmdCheck.CombinedOutput(); err == nil {
			currentRemote := strings.TrimSpace(string(out))
			// 如果仓库地址变了，说明项目移动到新仓库了，删除旧缓存重建
			if filepath.ToSlash(currentRemote) != filepath.ToSlash(repoURL) {
				os.RemoveAll(cacheDir) // 🗑️ 删除旧缓存
			}
		} else {
			// 获取 remote 失败，说明本地缓存有问题，删了重建
			os.RemoveAll(cacheDir)
		}
	}

	// --- 如果缓存不存在，从远程 clone ---
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
			return err
		}
		// git clone --bare：只克隆 git 历史，不创建工作文件
		// --no-hardlinks：不使用硬链接（兼容不同文件系统）
		cmd := exec.CommandContext(ctx, "git", "clone", "--no-hardlinks", "--bare", repoURL, cacheDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %v, output: %s", err, string(out))
		}
		return nil
	}

	// --- 如果缓存已存在，执行 git fetch 更新 ---
	// +refs/heads/*:refs/heads/* 表示强制更新所有分支
	// --prune 删除远程已经删掉的分支
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin",
		"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*", "--prune")
	cmd.Dir = cacheDir
	if out, err := cmd.CombinedOutput(); err != nil {
		// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
		// 如果是本地仓库作为 repoURL，fetch 本身可能会因为分支检出问题报错
		if strings.Contains(string(out), "refusing to fetch into branch") {
			return nil // 容忍这个错误，缓存已经是最新的了
		}
		return fmt.Errorf("git fetch failed: %v, output: %s", err, string(out))
	}

	return nil
}

// ============================================================
// 📜 GetCommits：获取最新的提交记录
//
// 支持按以下条件搜索：
// - keyword：在提交信息中搜索关键词
// - author：按作者搜索
// - file：按涉及的文件搜索
// - ref：指定分支/标签（不指定就查所有分支）
// ============================================================

// GetCommits 获取最新 50 条提交记录，并支持按 message/author/file 搜索，支持按 ref（分支/Tag）过滤
func GetCommits(ctx context.Context, projectID, keyword, author, file, ref string) ([]GitCommit, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("git cache not found for project %s", projectID)
	}

	// 构建 git log 命令参数
	// --format=%H|%s|%an|%cI 自定义输出格式：hash|消息|作者|时间
	args := []string{"log", "-n", "50", "--format=%H|%s|%an|%cI"}

	if keyword != "" {
		args = append(args, "--grep="+keyword, "-i") // -i 不区分大小写
	}
	if author != "" {
		args = append(args, "--author="+author, "-i")
	}
	if ref != "" {
		args = append(args, ref) // 只查某个分支/标签
	} else {
		args = append(args, "--all") // 查全部分支
	}
	if file != "" {
		args = append(args, "--", file) // 只查涉及某个文件的提交
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cacheDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %v", err)
	}

	// 解析输出：每行用 | 分隔 → 分成 Hash/Message/Author/CreatedAt
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

// ============================================================
// 🔍 GetDiff：获取两次提交之间的代码差异
//
// 如果 fromCommit 为空，就查看 toCommit 相对于父提交的变更。
// limitBytes 参数可以限制 diff 的大小（防止把内存撑爆）。
// ============================================================

// GetDiff 获取两次提交之间的 diff 字符串。如果 fromCommit 为空则默认比较该 commit 本身变更。
func GetDiff(ctx context.Context, projectID, fromCommit, toCommit string, limitBytes int) (string, error) {
	cacheDir := GetCacheDir(projectID)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", fmt.Errorf("git cache not found for project %s", projectID)
	}

	var args []string
	if fromCommit == "" {
		args = []string{"show", "--format=", toCommit} // --format= 去掉 log 头，只保留 diff
	} else {
		args = []string{"diff", fromCommit, toCommit}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cacheDir

	// 如果不限制大小，直接获取完整 diff
	if limitBytes <= 0 {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git diff failed: %v, output: %s", err, string(out))
		}
		return string(out), nil
	}

	// 限制大小：通过管道流式读取，超过限制就截断
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
		_ = cmd.Process.Kill() // 超出限制，杀掉进程
		return string(data[:limitBytes]) + "\n\n... [Diff 截断: 文件变更过大]", nil
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
		return string(data[:limitBytes]) + "\n\n... [Diff 截断: 文件变更过大]", nil
	}

	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
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

// FindGitRepo 在 root 目录树中递归查找包含指定 commit 的 git 仓库
// 就像在一堆文件夹里找包含某次"存档"的游戏存档目录
func FindGitRepo(root, commit string) (string, error) {
	// @Ref: docs/sps/plans/20260530_fix_branch_deploy_diff_freeze_plan.md | @Date: 2026-05-30
	// 如果传入的不是 40 位 SHA（比如分支名），直接返回
	// 防止因为全局搜索太慢把系统搞崩
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
			return nil // 忽略权限错误
		}
		if info.IsDir() && info.Name() == ".git" {
			repoDir := filepath.Dir(path)
			// 检查这个仓库里有没有指定的 commit
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			checkCmd := exec.CommandContext(ctx, "git", "cat-file", "-t", commit)
			checkCmd.Dir = repoDir
			out, checkErr := checkCmd.Output()
			if checkErr == nil && strings.TrimSpace(string(out)) == "commit" {
				result = repoDir
				return filepath.SkipAll // 找到了，停止搜索
			}
		}
		return nil
	})
	return result, err
}

// FilterFilesForTruncatedDiff diff 被截断后，同步裁剪文件列表
// 确保 diff 展示的文件跟实际 diff 内容一致
func FilterFilesForTruncatedDiff(truncatedDiff, originalFiles string) string {
	fileSet := make(map[string]bool)
	lines := strings.Split(truncatedDiff, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "diff --git a/") {
			continue
		}
		parts := strings.SplitN(line, " ", 4)
		if len(parts) >= 4 {
			file := strings.TrimPrefix(parts[2], "a/")
			fileSet[file] = true
		}
	}

	if len(fileSet) == 0 {
		return originalFiles
	}

	filesLines := strings.Split(strings.TrimSpace(originalFiles), "\n")
	var filtered []string
	for _, line := range filesLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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

// IsCommitHash 检查字符串是不是合法的 40 位 Git 提交哈希
// Git 的 commit hash 是 40 位十六进制数（0-9, a-f）
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

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: bare 仓库是什么？
//    A: 只有 git 历史，没有工作目录。就像一个"存档专用文件夹"~
//
// 2. Q: git diff 和 git show 有什么区别？
//    A: show 显示单次提交的变更，diff 显示两个提交之间的差异~
//
// 中级：
// 3. Q: 为什么不用直接 clone 远程仓库，而是先建 bare cache？
//    A: 从本地 bare 仓库 clone 比从远程 clone 快几十倍！
//       而且 bare 仓库只更新 fetch，不用每次重新下载全部历史~
//
// 4. Q: FindGitRepo 为什么先检查 commit 长度？
//    A: 如果传进来的是分支名（比如 "main"），
//       全局搜索会非常耗时。40 位 SHA 检查是个快速过滤器~
//
// 高级：
// 5. Q: io.ReadFull 在 limitBytes 场景下的作用？
//    A: 只会读取指定数量的字节，超过就丢弃。
//       git diff 可能产生非常巨大的输出（比如几百 MB），
//       用 ReadFull 确保内存不会被撑爆~
// ============================================================
