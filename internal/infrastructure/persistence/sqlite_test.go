package persistence

import (
	"pdeploy/internal/domain"
	"testing"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	
	// Create table for ProjectModel and EnvironmentModel
	err = db.AutoMigrate(&ProjectModel{}, &EnvironmentModel{})
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}
	
	return db
}

func TestSqliteProjectRepository_SaveAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteProjectRepository(db)

	p, _ := domain.NewProject("test-infra", "git@git")
	p.AddEnvironment("prod", "main", "symlink")
	
	err := repo.Save(p)
	if err != nil {
		t.Fatalf("Expected no error on save, got %v", err)
	}
	if p.ID == 0 {
		t.Fatalf("Expected ID to be populated")
	}

	found, err := repo.FindByID(p.ID)
	if err != nil {
		t.Fatalf("Expected no error on find, got %v", err)
	}
	if found == nil {
		t.Fatalf("Expected to find project")
	}
	if found.Name != "test-infra" {
		t.Errorf("Expected name 'test-infra', got '%s'", found.Name)
	}
	if len(found.Environments) != 1 {
		t.Fatalf("Expected 1 env, got %d", len(found.Environments))
	}
	if found.Environments[0].Name != "prod" {
		t.Errorf("Expected env name 'prod', got '%s'", found.Environments[0].Name)
	}
}
