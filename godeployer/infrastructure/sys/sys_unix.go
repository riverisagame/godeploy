// ============================================================
// 文件：sys_unix.go
// 作用：🖥️ Linux/Unix 系统的进程管理工具！
//
// 这个文件只在 Linux/Mac 系统上编译（//go:build !windows）。
// 它提供了两个关键功能：
//
// 1. SetProcessGroup：把命令设置成一个"进程组"
//    就像把一大家子人放在同一个户口本上
//
// 2. KillProcessGroup：杀死整个进程组
//    如果要关掉命令，连带它的子子孙孙一起杀掉！
//
// 为什么需要这个？
// 当执行 npm build 时，npm 启动 webpack，webpack 又启动其他工具……
// 如果只杀掉 npm，webpack 还会继续运行（变成"孤儿进程"）。
// 通过进程组机制，我们可以一次性干掉整个家族！
// ============================================================

//go:build !windows

package sys

import (
	"os/exec" // 🖥️ 执行外部命令
	"syscall" // ⚙️ 系统调用（Linux 底层 API）
)

// SetProcessGroup 为命令设置进程组。
// 这样以后 KillProcessGroup 可以把整个进程树一起杀掉。
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		// 初始化系统进程属性
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Setpgid = true：创建一个新的进程组
	// 进程组 ID = 当前进程的 ID
	cmd.SysProcAttr.Setpgid = true
}

// KillProcessGroup 杀死整个进程组（包括所有子进程）。
// 负数 PID 表示要杀死整个组，而不是单个进程。
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil // 进程还没启动，不用杀
	}
	// -cmd.Process.Pid：负号表示"整个进程组"
	// SIGKILL：强制杀死（无法被捕获/忽略）
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是"孤儿进程"？
//    A: 父进程死了，子进程还活着——没人管的孩子。
//       如果只杀 npm 不杀 webpack，webpack 就成了孤儿进程~
//
// 中级：
// 2. Q: Setpgid = true 是做什么的？
//    A: 创建一个新的"进程组"并让命令成为组长。
//       以后该命令创建的子进程都会自动加入这个组~
//
// 3. Q: 为什么用负 PID 来杀进程组？
//    A: Linux 中，kill(-pid, sig) 表示"杀死整个进程组"，
//       kill(pid, sig) 只杀单个进程。负号 = 全组~
//
// 高级：
// 4. Q: 这个文件为什么有 //go:build !windows 标记？
//    A: Windows 的进程管理机制完全不同！
//       它没有 POSIX 的进程组概念，需要用其他方式（如 Job Object）实现类似功能~
// ============================================================
