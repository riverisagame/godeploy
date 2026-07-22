package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/riverisagame/godeploy/internal/application"
)

type contextKey string

const ContextKeyUserID contextKey = "user_id"
const ContextKeyRole contextKey = "role"

type AuthMiddleware struct {
	secret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{secret: []byte(secret)}
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// @Ref: docs/sps/plans/20260721_phase4_ir.md Task 4.4 | @Date: 2026-07-21
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fallback to query parameter for SSE
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return m.secret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userIDFloat, ok := claims["user_id"].(float64); ok {
				ctx := context.WithValue(r.Context(), ContextKeyUserID, userIDFloat)
				if roleStr, ok := claims["role"].(string); ok {
					ctx = context.WithValue(ctx, ContextKeyRole, roleStr)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// RequireAdmin middleware ensures the user has admin role
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(ContextKeyRole).(string)
		if !ok || role != "admin" {
			RespondError(w, http.StatusForbidden, "Forbidden: Admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware records request method, path and latency
type LoggingMiddleware struct{}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{}
}

func (m *LoggingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We use a custom ResponseWriter to capture the status code
		rw := &responseWriter{w, http.StatusOK}

		// Wait, instead of implementing full responseWriter, we can just do basic log
		log.Printf("[REQ] %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(rw, r)
		log.Printf("[RES] %d %s %s", rw.statusCode, r.Method, r.URL.Path)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RecoveryMiddleware 捕获所有的 panic 并返回 500 标准 JSON 错误
type RecoveryMiddleware struct{}

func NewRecoveryMiddleware() *RecoveryMiddleware {
	return &RecoveryMiddleware{}
}

func (m *RecoveryMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// TODO: 记录日志
				RespondError(w, http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RespondError 提供统一的标准错误响应格式
func RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(`{"code":` + fmt.Sprintf("%d", code) + `,"error":"` + message + `"}` + "\n")); err != nil {
		log.Printf("Failed to write error response: %v\n", err)
	}
}

type AuditMiddleware struct {
	svc application.AuditService
}

func NewAuditMiddleware(svc application.AuditService) *AuditMiddleware {
	return &AuditMiddleware{svc: svc}
}

func (m *AuditMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isMutating := r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE"

		if !isMutating {
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)

		go func(method, path string, ctx context.Context) {
			userID, _ := ctx.Value(ContextKeyUserID).(float64)
			role, _ := ctx.Value(ContextKeyRole).(string)
			_ = m.svc.RecordAction(context.Background(), int64(userID), role, method, path, "")
		}(r.Method, r.URL.Path, r.Context())
	})
}
