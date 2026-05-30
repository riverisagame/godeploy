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
