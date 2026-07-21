package persistence

import (
	"github.com/riverisagame/godeploy/internal/domain"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestEnvironmentServerIDsPersistence 验证 ServerIDs 和 DeployPath 在读写中不丢失
// RED 阶段：当前 EnvironmentModel 没有这两个字段，读出来一定为空/默认值
func TestEnvironmentServerIDsPersistence(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&ProjectModel{}, &EnvironmentModel{})

	repo := NewSqliteProjectRepository(db)

	// 构造带 ServerIDs 和 DeployPath 的项目
	p := &domain.Project{
		Name:         "test-project",
		RepoURL:      "https://github.com/test/repo.git",
		KeepReleases: 5,
		Environments: []*domain.Environment{
			{
				Name:       "production",
				Branch:     "main",
				DeployType: "symlink",
				PreDeploy:  "npm run build",
				PostDeploy: "systemctl restart app",
				ServerIDs:  []uint{10, 20},
				DeployPath: "/opt/app/production",
			},
		},
	}

	err = repo.Save(p)
	if err != nil {
		t.Fatal("Save failed:", err)
	}

	// 从 DB 重新读取
	loaded, err := repo.FindByID(p.ID)
	if err != nil {
		t.Fatal("FindByID failed:", err)
	}

	if len(loaded.Environments) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(loaded.Environments))
	}

	env := loaded.Environments[0]

	// 验证 ServerIDs 不丢失 —— 当前应该失败
	if len(env.ServerIDs) != 2 {
		t.Fatalf("expected 2 ServerIDs, got %d (ServerIDs: %v)", len(env.ServerIDs), env.ServerIDs)
	}
	if env.ServerIDs[0] != 10 || env.ServerIDs[1] != 20 {
		t.Errorf("expected ServerIDs [10, 20], got %v", env.ServerIDs)
	}

	// 验证 DeployPath 不丢失 —— 当前应该失败
	if env.DeployPath != "/opt/app/production" {
		t.Errorf("expected DeployPath '/opt/app/production', got '%s'", env.DeployPath)
	}
}

// TestEnvironmentEmptyServerIDs 验证空 ServerIDs 不会 panic
func TestEnvironmentEmptyServerIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&ProjectModel{}, &EnvironmentModel{})

	repo := NewSqliteProjectRepository(db)

	p := &domain.Project{
		Name:         "empty-env-project",
		RepoURL:      "https://github.com/test/empty.git",
		KeepReleases: 3,
		Environments: []*domain.Environment{
			{
				Name:       "staging",
				Branch:     "develop",
				DeployType: "symlink",
				ServerIDs:  []uint{},
				DeployPath: "/var/www/staging",
			},
		},
	}

	err = repo.Save(p)
	if err != nil {
		t.Fatal("Save failed:", err)
	}

	loaded, err := repo.FindByID(p.ID)
	if err != nil {
		t.Fatal("FindByID failed:", err)
	}

	env := loaded.Environments[0]

	// 空的 ServerIDs 应该是空切片 —— 当前应该失败（返回 nil）
	if env.ServerIDs == nil {
		t.Error("ServerIDs should not be nil, should be empty slice")
	}

	// DeployPath 应该保留
	if env.DeployPath != "/var/www/staging" {
		t.Errorf("expected DeployPath '/var/www/staging', got '%s'", env.DeployPath)
	}
}
