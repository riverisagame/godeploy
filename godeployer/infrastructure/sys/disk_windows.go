// ============================================================
// 文件：disk_windows.go
// 作用：💿 检查 Windows 磁盘空间还剩多少！
//
// 跟 disk_linux.go 功能一样，但用 Windows 的 API。
// Windows 通过 kernel32.dll 的 GetDiskFreeSpaceExW 函数
// 来获取磁盘可用空间。
// ============================================================

//go:build windows
// +build windows

package sys

import (
	"syscall"
	"unsafe"
)

// GetFreeDiskSpaceMB 获取 Windows 系统上指定路径的磁盘可用空间（MB）
// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
func GetFreeDiskSpaceMB(path string) int {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	// 加载 Windows 的 kernel32.dll（核心系统库）
	h, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return 1024 // 加载失败，返回 1GB 安全值
	}

	// 找到 GetDiskFreeSpaceExW 函数（Windows 的磁盘空间查询 API）
	c, err := h.FindProc("GetDiskFreeSpaceExW")
	if err != nil {
		return 1024
	}

	if path == "" {
		path = "."
	}

	// 把路径转成 Windows 需要的 UTF16 格式
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 1024
	}

	// 调用 Windows API
	_, _, _ = c.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	// 字节 → MB（除以 1024 两次）
	return int(freeBytesAvailable / 1024 / 1024)
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么 DLL？
//    A: Dynamic Link Library（动态链接库），Windows 上的共享代码库。
//       kernel32.dll 是 Windows 最核心的库之一~
//
// 中级：
// 2. Q: 为什么 syscall.LoadDLL 和 FindProc 要分开？
//    A: 先加载 DLL 文件，再从 DLL 里找到具体函数。
//       就像先拿起工具箱（DLL），再从里面拿出螺丝刀（函数）~
//
// 3. Q: unsafe.Pointer 是做什么的？
//    A: Go 是安全语言，不允许直接操作内存。
//       unsafe.Pointer 是"不安全指针"，用来跟 C 语言 API 交互~
// ============================================================
