//go:build !windows

package sys

import (
	"os/exec"
	"syscall"
)

func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Kill the entire process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
