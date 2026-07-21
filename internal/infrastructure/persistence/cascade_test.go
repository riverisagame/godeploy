package persistence_test

import (
	"github.com/glebarez/sqlite"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/riverisagame/godeploy/internal/infrastructure/persistence"
	"gorm.io/gorm"
	"testing"
)

func TestDatabaseCascadeDeletes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	// SQLite needs PRAGMA foreign_keys = ON for cascade to work
	db.Exec("PRAGMA foreign_keys = ON")

	err = db.AutoMigrate(
		&persistence.ProjectModel{},
		&persistence.EnvironmentModel{},
		&persistence.DeploymentModel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a project with environment
	projectRepo := persistence.NewSqliteProjectRepository(db)
	p, _ := domain.NewProject("CascadeProj", "url")
	_ = p.AddEnvironment("prod", "main", "symlink")
	err = projectRepo.Save(p)
	if err != nil {
		t.Fatalf("Failed to save project: %v", err)
	}

	found, _ := projectRepo.FindByID(p.ID)
	envID := found.Environments[0].ID

	// Create a deployment
	deployRepo := persistence.NewSqliteDeploymentRepository(db)
	d, err := domain.NewDeployment(envID, 1, "abc1234")
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}
	err = deployRepo.Save(d)
	if err != nil {
		t.Fatalf("Failed to save deployment: %v", err)
	}

	// Delete Project -> should cascade delete Environment -> should cascade delete Deployment
	// For cascade delete using GORM, we can either use Repo delete (if we implement it)
	// or directly delete the model via db
	db.Delete(&persistence.ProjectModel{}, p.ID)

	var count int64
	db.Model(&persistence.EnvironmentModel{}).Where("project_id = ?", p.ID).Count(&count)
	if count > 0 {
		t.Errorf("Expected Environments to be cascade deleted, but found %d", count)
	}

	db.Model(&persistence.DeploymentModel{}).Where("env_id = ?", envID).Count(&count)
	if count > 0 {
		t.Errorf("Expected Deployments to be cascade deleted, but found %d", count)
	}
}
