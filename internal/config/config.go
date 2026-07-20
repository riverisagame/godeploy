package config

import "os"

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
