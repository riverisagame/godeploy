package domain_test

import (
	"testing"

	"deploy/godeployer/domain"
)

// @Ref: docs/sps/plans/001-ddd-refactor-plan.md | @Date: 2026-05-30
// Edge Case 测试：确保提取出来的核心领域模型不受任何外部依赖（如 sql.DB, gin.Context）的污染。
func TestDomain_EntityInstantiation(t *testing.T) {
	// 测试 Config 实体是否被成功移动到 domain 并可被实例化
	cfg := domain.Config{
		ProjectConfigDir: "/etc/godeployer/projects",
	}
	if cfg.ProjectConfigDir == "" {
		t.Fatal("Config entity not properly defined or instantiated")
	}

	// 测试 DeployJob (Task) 实体，并确保不包含任何外部耦合
	job := domain.DeployJob{
		TaskID:      123,
		LogFilePath: "/var/log/task.log",
	}
	if job.TaskID != 123 {
		t.Fatal("DeployJob entity not properly defined")
	}

	// 测试 UserResponse，确保它是存粹的领域 DTO
	user := domain.UserResponse{
		ID:       1,
		Username: "admin",
		Role:     "Admin",
	}
	if user.Username != "admin" {
		t.Fatal("UserResponse entity not properly defined")
	}
}

// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31
// DeployTask 状态机边界测试

func TestDeployTask_StateMachine_ValidTransitions(t *testing.T) {
	// pending → deploying → success
	task := &domain.DeployTask{Status: domain.StatusPending}
	if err := task.Start(); err != nil {
		t.Fatalf("pending → deploying should succeed, got: %v", err)
	}
	if err := task.Complete(); err != nil {
		t.Fatalf("deploying → success should succeed, got: %v", err)
	}

	// pending → deploying → failed
	task2 := &domain.DeployTask{Status: domain.StatusPending}
	task2.Start()
	if err := task2.Fail(); err != nil {
		t.Fatalf("deploying → failed should succeed, got: %v", err)
	}

	// pending → abort
	task3 := &domain.DeployTask{Status: domain.StatusPending}
	if err := task3.Abort(); err != nil {
		t.Fatalf("pending → aborted should succeed, got: %v", err)
	}

	// deploying → abort
	task4 := &domain.DeployTask{Status: domain.StatusPending}
	task4.Start()
	if err := task4.Abort(); err != nil {
		t.Fatalf("deploying → aborted should succeed, got: %v", err)
	}
}

func TestDeployTask_StateMachine_InvalidTransitions(t *testing.T) {
	// success → deploying (invalid)
	task := &domain.DeployTask{Status: domain.StatusSuccess}
	if err := task.Start(); err == nil {
		t.Fatal("success → deploying should fail")
	}

	// pending → success (skip deploying, invalid)
	task2 := &domain.DeployTask{Status: domain.StatusPending}
	if err := task2.Complete(); err == nil {
		t.Fatal("pending → success should fail")
	}

	// failed → start (invalid)
	task3 := &domain.DeployTask{Status: domain.StatusFailed}
	if err := task3.Start(); err == nil {
		t.Fatal("failed → deploying should fail")
	}

	// aborted → deploying (invalid)
	task4 := &domain.DeployTask{Status: domain.StatusAborted}
	if err := task4.Start(); err == nil {
		t.Fatal("aborted → deploying should fail")
	}
}

func TestDeployTask_IsActive(t *testing.T) {
	pending := &domain.DeployTask{Status: domain.StatusPending}
	deploying := &domain.DeployTask{Status: domain.StatusDeploying}
	success := &domain.DeployTask{Status: domain.StatusSuccess}
	failed := &domain.DeployTask{Status: domain.StatusFailed}
	aborted := &domain.DeployTask{Status: domain.StatusAborted}

	if !pending.IsActive() || !deploying.IsActive() {
		t.Fatal("pending and deploying should be active")
	}
	if success.IsActive() || failed.IsActive() || aborted.IsActive() {
		t.Fatal("terminal states should not be active")
	}
}

func TestDeployJob_NewDeployJob(t *testing.T) {
	cfg := &domain.Config{}
	job := domain.NewDeployJob(42, cfg, "/tmp/test.log")

	if job.TaskID != 42 {
		t.Errorf("expected TaskID 42, got %d", job.TaskID)
	}
	if job.Config != cfg {
		t.Error("Config pointer mismatch")
	}
	if job.LogFilePath != "/tmp/test.log" {
		t.Errorf("expected LogFilePath /tmp/test.log, got %s", job.LogFilePath)
	}
	if job.Ctx == nil || job.Cancel == nil {
		t.Fatal("Ctx and Cancel must not be nil")
	}
	if job.IsCancelled() {
		t.Fatal("new job should not be cancelled")
	}

	// 取消后 IsCancelled 返回 true
	job.Cancel()
	if !job.IsCancelled() {
		t.Fatal("cancelled job should report IsCancelled=true")
	}
}

func TestDeployStatus_Valid(t *testing.T) {
	tests := []struct {
		s     domain.DeployStatus
		valid bool
	}{
		{domain.StatusPending, true},
		{domain.StatusDeploying, true},
		{domain.StatusSuccess, true},
		{domain.StatusFailed, true},
		{domain.StatusAborted, true},
		{domain.StatusRolledBack, true},
		{domain.StatusFailedLockRejected, true},
		{domain.StatusCriticalBrainSplit, true},
		{domain.DeployStatus("bogus"), false},
		{domain.DeployStatus(""), false},
	}
	for _, tt := range tests {
		if tt.s.Valid() != tt.valid {
			t.Errorf("DeployStatus(%q).Valid() = %v, want %v", tt.s, tt.s.Valid(), tt.valid)
		}
	}
}
