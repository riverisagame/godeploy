package godeployer_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_UserManagement_CRUD(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	// Login as admin
	loginJSON := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	body, _ := json.Marshal(loginJSON)
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	token := loginResp.Token

	// 1. GET /api/users
	t.Run("Get Users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
	})

	// 2. POST /api/users
	t.Run("Create User", func(t *testing.T) {
		newUser := map[string]interface{}{
			"username": "new_user",
			"password": "new_password",
			"role":     "developer",
		}
		body, _ := json.Marshal(newUser)
		req, _ := http.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("Expected 200 or 201, got %d", w.Code)
		}
	})

	// 3. PUT /api/users/:username
	t.Run("Update User", func(t *testing.T) {
		updateUser := map[string]interface{}{
			"role": "viewer",
			"permitted_projects": "test-app",
		}
		body, _ := json.Marshal(updateUser)
		req, _ := http.NewRequest("PUT", "/api/users/new_user", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
	})

	// 4. DELETE /api/users/:username
	t.Run("Delete User", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/users/new_user", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
	})
}
