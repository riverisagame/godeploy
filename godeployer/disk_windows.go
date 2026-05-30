// +build windows

package godeployer

import (
	"syscall"
	"unsafe"
)

// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
func getFreeDiskSpaceMB(path string) int {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64
	h, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return 1024 // 失败默认 1GB
	}
	c, err := h.FindProc("GetDiskFreeSpaceExW")
	if err != nil {
		return 1024
	}
	
	// 若 path 为空，默认检查当前盘符
	if path == "" {
		path = "."
	}
	
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 1024
	}

	_, _, _ = c.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	return int(freeBytesAvailable / 1024 / 1024)
}
