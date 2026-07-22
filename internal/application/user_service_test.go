package application_test

import (
	"testing"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
)

// Reusing mockUserRepo from auth_service_test.go
// Wait, they are in the same package (application_test), so we can reuse mockUserRepo.

func TestUserService_CreateAndListUsers(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*domain.User)}
	svc := application.NewUserService(repo)

	// Test CreateUser
	err := svc.CreateUser("dev1", "devpass", "developer")
	if err != nil {
		t.Fatalf("expected nil error on CreateUser, got %v", err)
	}

	// Validate DB state
	u, _ := repo.FindByUsername("dev1")
	if u == nil {
		t.Fatalf("expected user to be created")
	}
	if u.Role != "developer" {
		t.Fatalf("expected role developer, got %s", u.Role)
	}

	// Test ListUsers
	users, err := svc.ListUsers()
	if err != nil {
		t.Fatalf("expected nil error on ListUsers, got %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].PasswordHash != "" {
		t.Fatalf("expected PasswordHash to be sanitized in ListUsers, got %s", users[0].PasswordHash)
	}

	// Test duplicate user
	err = svc.CreateUser("dev1", "devpass2", "admin")
	if err == nil {
		t.Fatalf("expected error when creating duplicate user")
	}
}
