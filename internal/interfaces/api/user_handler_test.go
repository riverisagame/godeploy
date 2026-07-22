package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/riverisagame/godeploy/internal/interfaces/api"
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

func (m *mockUserRepo) FindAll() ([]*domain.User, error) {
	var list []*domain.User
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func TestUserHandler_CreateAndList(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*domain.User)}
	userSvc := application.NewUserService(repo)
	handler := api.NewUserHandler(userSvc)

	req := httptest.NewRequest("GET", "/api/users", nil)
	rr := httptest.NewRecorder()
	
	handler.List(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for List, got %d", rr.Code)
	}

	body := []byte(`{"username":"dev1", "password":"123", "role":"developer"}`)
	req = httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	
	handler.Create(rr, req)
	
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for Create, got %d", rr.Code)
	}
}
