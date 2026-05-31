// @Ref: docs/sps/plans/20260531-ddd-full-tactical-plan.md | @Date: 2026-05-31
package application

import (
	"bytes"
	"context"
	"os/exec"

	"deploy/godeployer/infrastructure/sys"
)

// runCmd 本地命令执行包裹，支持 Context 超时下彻底清退子进程树。
// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
func runCmd(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	sys.SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		sys.KillProcessGroup(cmd)
		<-done
		return buf.Bytes(), ctx.Err()
	case err := <-done:
		return buf.Bytes(), err
	}
}
