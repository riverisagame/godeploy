// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31
package domain

// DeployStatus 部署任务状态值对象。
// 预定义合法状态，禁止外部直接构造无效状态字符串。
type DeployStatus string

const (
	StatusPending            DeployStatus = "pending"
	StatusDeploying          DeployStatus = "deploying"
	StatusSuccess            DeployStatus = "success"
	StatusFailed             DeployStatus = "failed"
	StatusAborted            DeployStatus = "aborted"
	StatusRolledBack         DeployStatus = "rolled_back"
	StatusFailedLockRejected DeployStatus = "failed_lock_rejected"
	StatusCriticalBrainSplit DeployStatus = "critical_brain_split"
)

// Valid 判断当前状态是否为预定义的合法值。
func (s DeployStatus) Valid() bool {
	switch s {
	case StatusPending, StatusDeploying, StatusSuccess, StatusFailed,
		StatusAborted, StatusRolledBack, StatusFailedLockRejected, StatusCriticalBrainSplit:
		return true
	}
	return false
}

// IsTerminal 判断当前状态是否为终态（不会再变迁）。
func (s DeployStatus) IsTerminal() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusAborted
}

// IsRunnable 判断当前状态是否允许启动部署。
func (s DeployStatus) IsRunnable() bool {
	return s == StatusPending
}
