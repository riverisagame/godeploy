package application_test

import (
	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"testing"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func (m *mockUserRepo) Save(u *domain.User) error {
	m.users[u.Username] = u
	u.ID = uint(len(m.users))
	return nil
}

func (m *mockUserRepo) FindByUsername(username string) (*domain.User, error) {
	return m.users[username], nil
}

func TestAuthService_Login(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*domain.User)}
	svc := application.NewAuthService(repo, "secret-key")

	// Pre-create user "admin" with password "admin123"
	// Hash logic would normally happen on creation, but we test the service wrapper
	hashedPwd, _ := application.HashPassword("admin123")
	_ = repo.Save(&domain.User{
		Username:     "admin",
		PasswordHash: hashedPwd,
	})

	// Test valid login
	token, err := svc.Login("admin", "admin123")
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected a valid JWT token, got empty string")
	}

	// Test invalid password
	_, err = svc.Login("admin", "wrongpass")
	if err == nil {
		t.Fatalf("expected error on invalid password, got none")
	}

	// Test non-existent user
	_, err = svc.Login("nobody", "admin123")
	if err == nil {
		t.Fatalf("expected error on non-existent user, got none")
	}
}
