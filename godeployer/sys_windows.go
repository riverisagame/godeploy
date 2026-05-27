//go:build windows

package godeployer

import (
	"fmt"
	"os/exec"
)

func setProcessGroup(cmd *exec.Cmd) {
	// On Windows, taskkill /T can kill the entire tree without process group settings
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Windows 平台通过 taskkill /F /T 级联强制清退整个子进程树以防对冲僵尸进程
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	return killCmd.Run()
}
