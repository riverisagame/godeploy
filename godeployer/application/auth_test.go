package application

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestAuth_PasswordHashing 验证密码的单向哈希与比对。
func TestAuth_PasswordHashing(t *testing.T) {
	password := "securePass123"

	// 生成 Hash
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == password {
		t.Error("hash should not be equal to plain password")
	}

	// 比对正确密码
	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash failed for correct password")
	}

	// 比对错误密码
	if CheckPasswordHash("wrongpass", hash) {
		t.Error("CheckPasswordHash succeeded for incorrect password")
	}
}

// TestAuth_TokenLifecycle 验证 JWT Token 的签发、解密与过期。
func TestAuth_TokenLifecycle(t *testing.T) {
	secret := "test-jwt-secret-key"
	username := "testuser"

	// 签发 Token (有效期 1 秒)
	token, err := GenerateToken(username, "admin", secret, 1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// 验证 Token
	parsedUser, parsedRole, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if parsedUser != username {
		t.Errorf("expected parsed username to be %q, got %q", username, parsedUser)
	}
	if parsedRole != "admin" {
		t.Errorf("expected parsed role to be 'admin', got %q", parsedRole)
	}

	// 等待过期并校验
	time.Sleep(1200 * time.Millisecond)
	_, _, err = ParseToken(token, secret)
	if err == nil {
		t.Error("expected token to expire, but ParseToken succeeded")
	}
}

// TestAuth_Middleware 验证 JWT 拦截中间件。
func TestAuth_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "middleware-secret"

	r := gin.New()
	r.Use(AuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		// 从上下文获取用户名
		user, exists := c.Get("username")
		if !exists {
			c.String(http.StatusInternalServerError, "username not in context")
			return
		}
		c.String(http.StatusOK, "welcome "+user.(string))
	})

	// 1. 无 Authorization Header 测试
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// 2. 有效 Token 测试
	token, err := GenerateToken("john_doe", "admin", secret, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestAuth_RoleMiddleware 验证基于 RBAC 角色拦截的中间件
func TestAuth_RoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "rbac-secret"

	r := gin.New()
	r.Use(AuthMiddleware(secret))
	r.Use(RoleMiddleware("admin", "deployer"))
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "access granted")
	})

	// 1. 测试 viewer 角色（应被拒绝）
	viewerToken, _ := GenerateToken("viewer_user", "viewer", secret, 5*time.Second)
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for viewer, got %d", w.Code)
	}

	// 2. 测试 deployer 角色（应通过）
	deployerToken, _ := GenerateToken("deploy_user", "deployer", secret, 5*time.Second)
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+deployerToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK for deployer, got %d", w2.Code)
	}
}
