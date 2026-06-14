package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort     int
	DatabaseURL    string
	JWTSecret      string
	CorsOrigins    []string
	MigrationsPath string
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnvInt("SERVER_PORT", 8080),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/harness_org?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		CorsOrigins:    getEnvSlice("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvSlice(key, fallback string) []string {
	v := getEnv(key, fallback)
	parts := strings.Split(v, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}
