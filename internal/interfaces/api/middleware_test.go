package api_test

import (
	"context"
	"github.com/riverisagame/godeploy/internal/interfaces/api"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	secret := []byte("test-secret")
	middleware := api.NewAuthMiddleware(string(secret))

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		userID := r.Context().Value(api.ContextKeyUserID)
		if userID != float64(123) {
			t.Errorf("expected UserID 123 in context, got %v", userID)
		}
	}))

	// Case 1: Missing token
	req1 := httptest.NewRequest("GET", "/protected", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", w1.Code)
	}

	// Case 2: Invalid token
	req2 := httptest.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer invalidtoken")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", w2.Code)
	}

	// Case 3: Valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 123,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(secret)

	req3 := httptest.NewRequest("GET", "/protected", nil)
	req3.Header.Set("Authorization", "Bearer "+tokenString)
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", w3.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	middleware := api.RecoveryMiddleware{}
	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for panic, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected json content type, got %s", contentType)
	}

	expectedBody := `{"code":500,"error":"Internal Server Error"}`
	if w.Body.String() != expectedBody+"\n" {
		t.Errorf("expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAdminMiddleware(t *testing.T) {
	handler := api.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: No Role in Context (Not Admin)
	req1 := httptest.NewRequest("GET", "/admin", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", w1.Code)
	}

	// Case 2: Role is developer (Not Admin)
	req2 := httptest.NewRequest("GET", "/admin", nil)
	ctx2 := req2.Context()
	ctx2 = context.WithValue(ctx2, api.ContextKeyRole, "developer")
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for developer, got %d", w2.Code)
	}

	// Case 3: Role is admin
	req3 := httptest.NewRequest("GET", "/admin", nil)
	ctx3 := req3.Context()
	ctx3 = context.WithValue(ctx3, api.ContextKeyRole, "admin")
	req3 = req3.WithContext(ctx3)
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin, got %d", w3.Code)
	}
}
