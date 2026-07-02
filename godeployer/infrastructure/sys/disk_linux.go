// ============================================================
// 文件：disk_linux.go
// 作用：💿 检查 Linux 磁盘空间还剩多少！
//
// 在写 diff 缓存文件之前，先检查磁盘空间够不够。
// 如果磁盘快满了，就不写 diff 了，防止磁盘爆掉。
//
// 这个函数只会在 Linux 上编译（//go:build !windows）。
// 它调用底层的 Linux 系统 API（syscall.Statfs）来获取磁盘信息。
// ============================================================

//go:build !windows
// +build !windows

package sys

import "syscall"

// GetFreeSpaceMB 获取指定路径所在磁盘的可用空间（单位：兆字节 MB）。
// 比如 path = "/var/log"，它返回 /var 所在磁盘分区的剩余空间。
//
// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
func GetFreeDiskSpaceMB(path string) int {
	var stat syscall.Statfs_t

	// 如果没传路径，用当前目录 "." 代替
	if path == "" {
		path = "."
	}

	// 调用 Linux 的 statfs 系统调用获取磁盘信息
	err := syscall.Statfs(path, &stat)
	if err != nil {
		// 如果获取失败（比如路径不存在），返回 1GB 作为安全默认值
		return 1024
	}

	// 计算可用空间（字节 → 兆字节）
	// Bavail = 可用块数（非 root 用户可用的）
	// Bsize = 每块的大小（字节）
	// 相乘得到字节数，除以 1024 两次得到 MB
	return int((stat.Bavail * uint64(stat.Bsize)) / 1024 / 1024)
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 为什么写文件前要检查磁盘空间？
//    A: 防止磁盘写满导致系统崩溃！
//       就像提前看汽油表，不够就不开车了~
//
// 中级：
// 2. Q: statfs 是什么？
//    A: Linux 的系统调用，用来获取文件系统的信息——
//       总大小、已用空间、可用空间等~
//
// 3. Q: 为什么获取失败时返回 1024（1GB）？
//    A: 保守策略！返回 1GB 意味着"空间充足"
//       让程序继续运行而不是直接报错~
//
// 高级：
// 4. Q: //go:build !windows 是在哪一层发挥作用？
//    A: 编译时！Go 在编译期根据操作系统选择包含哪些文件。
//       Windows 上看不到这个文件，Linux 上看不到 disk_windows.go~
// ============================================================
