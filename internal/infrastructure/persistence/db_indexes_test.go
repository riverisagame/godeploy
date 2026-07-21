package persistence_test

import (
	"github.com/riverisagame/godeploy/internal/infrastructure/persistence"
	"testing"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseIndexes_UniqueConstraints(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	err = db.AutoMigrate(
		&persistence.ProjectModel{},
		&persistence.EnvironmentModel{},
		&persistence.ServerModel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 1. Project Name Uniqueness
	p1 := &persistence.ProjectModel{Name: "Proj1", RepoURL: "url"}
	p2 := &persistence.ProjectModel{Name: "Proj1", RepoURL: "url2"}
	if err := db.Create(p1).Error; err != nil {
		t.Fatalf("Expected success, got %v", err)
	}
	if err := db.Create(p2).Error; err == nil {
		t.Error("Expected error for duplicate project name, got nil")
	}

	// 2. Environment Name Uniqueness within Project
	e1 := &persistence.EnvironmentModel{ProjectID: p1.ID, Name: "prod"}
	e2 := &persistence.EnvironmentModel{ProjectID: p1.ID, Name: "prod"}
	if err := db.Create(e1).Error; err != nil {
		t.Fatalf("Expected success, got %v", err)
	}
	if err := db.Create(e2).Error; err == nil {
		t.Error("Expected error for duplicate environment name in same project, got nil")
	}

	// 3. Server Name Uniqueness
	s1 := &persistence.ServerModel{Name: "Server1", IP: "1.1.1.1"}
	s2 := &persistence.ServerModel{Name: "Server1", IP: "2.2.2.2"}
	if err := db.Create(s1).Error; err != nil {
		t.Fatalf("Expected success, got %v", err)
	}
	if err := db.Create(s2).Error; err == nil {
		t.Error("Expected error for duplicate server name, got nil")
	}
}
