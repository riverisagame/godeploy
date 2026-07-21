package config

import (
	"errors"
	"os"
)

type Config struct {
	Port         string
	DBPath       string
	WorkspaceDir string
	JWTSecret    string
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		DBPath:       getEnv("DB_PATH", "pdeploy.db"),
		WorkspaceDir: getEnv("WORKSPACE_DIR", "./workspace"),
		JWTSecret:    getEnv("JWT_SECRET", "secret-key"),
	}
}

// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 1.1 | @Date: 2026-07-21
func (c *Config) Validate() error {
	if len(c.JWTSecret) < 16 {
		return errors.New("JWT_SECRET must be at least 16 characters long for production security")
	}
	return nil
}
