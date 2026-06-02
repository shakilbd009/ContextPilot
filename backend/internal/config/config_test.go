package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear all env vars we touch so the test is hermetic.
	for _, k := range []string{
		"SERVER_PORT", "DATABASE_URL", "REDIS_URL", "ENVIRONMENT",
		"RATE_LIMIT_IMPORT_MAX_REQUESTS", "RATE_LIMIT_IMPORT_WINDOW_SECS",
		"RATE_LIMIT_UPLOAD_MAX_REQUESTS", "RATE_LIMIT_UPLOAD_WINDOW_SECS",
		"CSRF_ALLOWED_ORIGINS",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	cfg := Load()
	if cfg.ServerPort != "3000" {
		t.Errorf("ServerPort default = %q, want %q", cfg.ServerPort, "3000")
	}
	if cfg.RateLimitImportMaxRequests != 100 {
		t.Errorf("RateLimitImportMaxRequests default = %d, want %d", cfg.RateLimitImportMaxRequests, 100)
	}
	if cfg.CSRFAllowedOrigins != nil {
		t.Errorf("CSRFAllowedOrigins default = %v, want nil", cfg.CSRFAllowedOrigins)
	}
}

func TestLoad_CSRFOrigins_SingleValue(t *testing.T) {
	t.Setenv("CSRF_ALLOWED_ORIGINS", "http://localhost:5173")
	cfg := Load()
	want := []string{"http://localhost:5173"}
	if !reflect.DeepEqual(cfg.CSRFAllowedOrigins, want) {
		t.Errorf("CSRFAllowedOrigins = %v, want %v", cfg.CSRFAllowedOrigins, want)
	}
}

func TestLoad_CSRFOrigins_CommaSeparated(t *testing.T) {
	t.Setenv("CSRF_ALLOWED_ORIGINS", "http://localhost:5173, http://localhost:8080 ,https://app.contextpilot.com")
	cfg := Load()
	want := []string{
		"http://localhost:5173",
		"http://localhost:8080",
		"https://app.contextpilot.com",
	}
	if !reflect.DeepEqual(cfg.CSRFAllowedOrigins, want) {
		t.Errorf("CSRFAllowedOrigins = %v, want %v", cfg.CSRFAllowedOrigins, want)
	}
}

func TestLoad_CSRFOrigins_EmptyEntriesDropped(t *testing.T) {
	t.Setenv("CSRF_ALLOWED_ORIGINS", "http://a.example.com,,, http://b.example.com ,")
	cfg := Load()
	want := []string{"http://a.example.com", "http://b.example.com"}
	if !reflect.DeepEqual(cfg.CSRFAllowedOrigins, want) {
		t.Errorf("CSRFAllowedOrigins = %v, want %v", cfg.CSRFAllowedOrigins, want)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{",", nil},
		{"   ", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ,c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}
	for _, tt := range tests {
		got := splitCSV(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
