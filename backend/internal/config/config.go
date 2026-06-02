package config

import (
	"os"
	"strconv"
	"strings"
)

// Defaults for the import rate limiter. Live as named constants so the
// config loader, the main gate, and the security baseline can all reference
// the same authoritative numbers (CWE-770 drift fix).
const (
	DefaultRateLimitImportMaxRequests = 100
	DefaultRateLimitImportWindowSecs  = 60
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

	// CSRF (CWE-352) — Origin/Referer allowlist applied to all
	// state-changing methods (POST/PUT/PATCH/DELETE) at the chi router
	// layer. Comma-separated env var. Defense-in-depth alongside the
	// SvelteKit /api/* hooks-layer check.
	//
	// When empty the CSRF middleware is disabled (legacy/dev behavior);
	// when set, requests whose Origin/Referer does not match one of
	// the configured origins are rejected with 403.
	CSRFAllowedOrigins []string
}

// Load reads env vars and returns a Config with defaults applied.
//
// SECURITY: RATE_LIMIT_IMPORT_MAX_REQUESTS=0 is treated as "use the
// documented default" (100/min), not "disable rate limiting". The previous
// behaviour silently skipped the middleware on 0, which is a CWE-770
// (Allocation of Resources Without Limits or Throttling) regression vector
// when an operator sets the env to 0 in compose/.env. To explicitly
// disable, set the value to a negative number.
func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		Environment: getEnv("ENVIRONMENT", "development"),

		RateLimitImportMaxRequests: rateLimitOrDefault("RATE_LIMIT_IMPORT_MAX_REQUESTS", DefaultRateLimitImportMaxRequests),
		RateLimitImportWindowSecs:  GetInt("RATE_LIMIT_IMPORT_WINDOW_SECS", DefaultRateLimitImportWindowSecs),
		RateLimitUploadMaxRequests: GetInt("RATE_LIMIT_UPLOAD_MAX_REQUESTS", 10),
		RateLimitUploadWindowSecs:  GetInt("RATE_LIMIT_UPLOAD_WINDOW_SECS", 60),

		CSRFAllowedOrigins: splitCSV(getEnv("CSRF_ALLOWED_ORIGINS", "")),
	}
}

// rateLimitOrDefault reads an integer env var. If unset, unparseable, or
// zero, it returns the supplied default. A negative value is passed through
// unchanged so callers can use a negative sentinel to mean "explicitly
// disabled" if they ever need that escape hatch.
func rateLimitOrDefault(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n == 0 {
		return def
	}
	return n
}

// splitCSV splits a comma-separated string into a clean slice, trimming
// whitespace and dropping empty entries. Returns nil for empty input.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
