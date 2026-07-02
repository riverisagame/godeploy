// ============================================================
// 文件：sys_windows.go
// 作用：🪟 Windows 系统的进程管理工具！
//
// 这个文件只在 Windows 系统上编译（//go:build windows）。
// 跟 sys_unix.go 实现同样的功能，但用 Windows 特有的方式。
//
// Windows 没有 Linux 的"进程组"概念，
// 所以用 taskkill 命令来杀死整个进程树。
// ============================================================

//go:build windows

package sys

import (
	"fmt"
	"os/exec"
)

// SetProcessGroup 在 Windows 上不需要设置进程组
// Windows 上通过 taskkill /T 参数就可以级联杀死子进程
func SetProcessGroup(cmd *exec.Cmd) {
	// On Windows, taskkill /T can kill the entire tree without process group settings
}

// KillProcessGroup 在 Windows 上用 taskkill 强制杀死整个进程树
// /F = 强制终止，/T = 终止所有子进程
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Windows 平台通过 taskkill /F /T 级联强制清退整个子进程树
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	return killCmd.Run()
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: Windows 怎么杀死"进程全家"？
//    A: taskkill /F /T /PID xxx
//       /F = Force（强制），/T = Tree（连坐）~
//
// 中级：
// 2. Q: 为什么 SetProcessGroup 在 Windows 上是空的？
//    A: Windows 用 Job Object 管理进程组，不需要设置 Setpgid。
//       taskkill /T 已经能实现"杀全家"的效果了~
//
// 3. Q: //go:build windows 和 //go:build !windows 是什么关系？
//    A: 二选一！Windows 编译时只包含 sys_windows.go，
//       Linux/Mac 编译时只包含 sys_unix.go，互斥的~
// ============================================================
