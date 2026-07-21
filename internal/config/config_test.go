package config_test

import (
	"github.com/riverisagame/godeploy/internal/config"
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Unset environment variables to test defaults
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("DB_PATH")
	_ = os.Unsetenv("WORKSPACE_DIR")
	_ = os.Unsetenv("JWT_SECRET")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected default Port 8080, got %s", cfg.Port)
	}
	if cfg.DBPath != "pdeploy.db" {
		t.Errorf("Expected default DB_PATH pdeploy.db, got %s", cfg.DBPath)
	}
	if cfg.WorkspaceDir != "./workspace" {
		t.Errorf("Expected default WORKSPACE_DIR ./workspace, got %s", cfg.WorkspaceDir)
	}
	if cfg.JWTSecret != "secret-key" {
		t.Errorf("Expected default JWTSecret secret-key, got %s", cfg.JWTSecret)
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	_ = os.Setenv("PORT", "9090")
	_ = os.Setenv("DB_PATH", "custom.db")
	_ = os.Setenv("WORKSPACE_DIR", "/tmp/ws")
	_ = os.Setenv("JWT_SECRET", "super-secret")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("DB_PATH")
		os.Unsetenv("WORKSPACE_DIR")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("Expected Port 9090, got %s", cfg.Port)
	}
	if cfg.DBPath != "custom.db" {
		t.Errorf("Expected DB_PATH custom.db, got %s", cfg.DBPath)
	}
	if cfg.WorkspaceDir != "/tmp/ws" {
		t.Errorf("Expected WORKSPACE_DIR /tmp/ws, got %s", cfg.WorkspaceDir)
	}
	if cfg.JWTSecret != "super-secret" {
		t.Errorf("Expected JWTSecret super-secret, got %s", cfg.JWTSecret)
	}
}
