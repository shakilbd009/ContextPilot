package config

import (
	"os"
	"strconv"
)

// Config holds all environment-backed configuration.
type Config struct {
	ServerPort  string
	DatabaseURL string
	RedisURL    string
	Environment string

	// Rate limiting — import endpoints (POST /meetings)
	RateLimitImportMaxRequests int
	RateLimitImportWindowSecs  int
	// Rate limiting — upload endpoints (future; reserved)
	RateLimitUploadMaxRequests int
	RateLimitUploadWindowSecs  int
}

// Load reads env vars and returns a Config with defaults applied.
func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		Environment: getEnv("ENVIRONMENT", "development"),

		RateLimitImportMaxRequests: GetInt("RATE_LIMIT_IMPORT_MAX_REQUESTS", 100),
		RateLimitImportWindowSecs:  GetInt("RATE_LIMIT_IMPORT_WINDOW_SECS", 60),
		RateLimitUploadMaxRequests: GetInt("RATE_LIMIT_UPLOAD_MAX_REQUESTS", 10),
		RateLimitUploadWindowSecs:  GetInt("RATE_LIMIT_UPLOAD_WINDOW_SECS", 60),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetInt is a helper for integer env vars with a fallback default.
func GetInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}