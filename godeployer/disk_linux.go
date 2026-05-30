// +build !windows

package godeployer

import "syscall"

// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
func getFreeDiskSpaceMB(path string) int {
	var stat syscall.Statfs_t
	// 若 path 为空，默认检查当前盘符
	if path == "" {
		path = "."
	}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 1024 // 失败默认 1GB 兜底
	}
	return int((stat.Bavail * uint64(stat.Bsize)) / 1024 / 1024)
}
