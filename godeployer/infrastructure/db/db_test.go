package db

import (
	"testing"

	"deploy/godeployer/domain"
)

func TestInitGORM(t *testing.T) {
	db, err := InitGORM("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	if !db.Migrator().HasTable(&domain.UserResponse{}) {
		t.Error("expected users table to exist")
	}
	if !db.Migrator().HasTable(&domain.DeployTask{}) {
		t.Error("expected deploy_tasks table to exist")
	}

	// Verify admin is created
	var admin domain.UserResponse
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Errorf("admin user should exist: %v", err)
	}
	if admin.Role != "admin" {
		t.Errorf("expected admin role, got %s", admin.Role)
	}
}
