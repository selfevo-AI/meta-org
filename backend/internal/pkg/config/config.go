package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort                int
	DatabaseURL               string
	JWTSecret                 string
	ModelSecretKey            string
	CorsOrigins               []string
	MigrationsPath            string
	MetaOrgMode               string
	PlatformAdminEmail        string
	PlatformAdminPasswordHash string
}

func Load() *Config {
	mode := strings.ToLower(strings.TrimSpace(getEnv("META_ORG_MODE", "single_org")))
	if mode != "saas" {
		mode = "single_org"
	}
	return &Config{
		ServerPort:                getEnvInt("SERVER_PORT", 8080),
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/meta_org?sslmode=disable"),
		JWTSecret:                 getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		ModelSecretKey:            getEnv("MODEL_SECRET_KEY", "dev-model-secret-key-32-bytes!!!"),
		CorsOrigins:               getEnvSlice("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		MigrationsPath:            getEnv("MIGRATIONS_PATH", "migrations"),
		MetaOrgMode:               mode,
		PlatformAdminEmail:        strings.ToLower(strings.TrimSpace(getEnv("META_ORG_PLATFORM_ADMIN_EMAIL", ""))),
		PlatformAdminPasswordHash: getEnv("META_ORG_PLATFORM_ADMIN_PASSWORD_HASH", ""),
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
