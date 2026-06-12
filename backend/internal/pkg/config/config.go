package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    int
	DatabaseURL   string
	JWTSecret     string
	CorsOrigins   string
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnvInt("SERVER_PORT", 8080),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/harness_org?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		CorsOrigins:   getEnv("CORS_ORIGINS", "http://localhost:3000"),
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
