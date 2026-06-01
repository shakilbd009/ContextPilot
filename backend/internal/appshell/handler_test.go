package appshell

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestIsAuthenticated(t *testing.T) {
	// Expose isAuthenticated for testing via a minimal handler wrapper
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := isAuthenticated(r)
		if got {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"valid user ID", "user-123", http.StatusOK},
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", http.StatusOK},
		{"empty", "", http.StatusUnauthorized},
		{"whitespace only", "   ", http.StatusUnauthorized},
		{"spaces around UUID", "  550e8400-e29b-41d4-a716-446655440000  ", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				r.Header.Set("X-User-ID", tt.header)
			}
			w := httptest.NewRecorder()
			testHandler.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Errorf("isAuthenticated() with header %q: got %d, want %d", tt.header, w.Code, tt.want)
			}
		})
	}
}

func TestHandler_FlagDisabled(t *testing.T) {
	// Feature flag disabled (default): all routes return 503
	log := zerolog.New(io.Discard)
	router := Handler(&log)

	// Any route, even public, is blocked by flag middleware
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Errorf("expected application/problem+json, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "App Shell Unavailable") {
		t.Errorf("expected 'App Shell Unavailable' in body, got %q", body)
	}
}

func TestHandler_FlagEnabled_CorrelationIDGenerated(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// Request without X-Request-ID: handler generates one
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Error("expected X-Request-ID header to be set")
	}
	if got := w.Header().Get("X-Correlation-ID"); got == "" {
		t.Error("expected X-Correlation-ID header to be set")
	}
	if w.Header().Get("X-Request-ID") != w.Header().Get("X-Correlation-ID") {
		t.Error("X-Request-ID and X-Correlation-ID should match")
	}
}

func TestHandler_FlagEnabled_CorrelationIDEchoed(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// Request with X-Request-ID: handler echoes it
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("X-Request-ID", "my-request-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "my-request-id" {
		t.Errorf("expected X-Request-ID 'my-request-id', got %q", got)
	}
	if got := w.Header().Get("X-Correlation-ID"); got != "my-request-id" {
		t.Errorf("expected X-Correlation-ID 'my-request-id', got %q", got)
	}
}

func TestHandler_FlagEnabled_PublicRoutes(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	tests := []struct {
		path     string
		wantCode int
		wantBody string
	}{
		{"/", http.StatusOK, `"shell":"landing"`},
		{"/login", http.StatusOK, `"shell":"login"`},
		{"/signup", http.StatusOK, `"shell":"signup"`},
		{"/404", http.StatusNotFound, `"shell":"not-found"`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("GET %s: expected %d, got %d", tt.path, tt.wantCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("GET %s: expected %q in body, got %q", tt.path, tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestHandler_FlagEnabled_AuthRedirect(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// Unauthenticated request to a protected route → redirect to /login
	req := httptest.NewRequest(http.MethodGet, "/meetings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("expected redirect to /login, got %q", loc)
	}
}

func TestHandler_NotFound_RedirectTo404(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// Unknown route → redirect to /404 (not itself a protected route)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The not-found handler issues a redirect to /404; the client must
	// then request /404 to complete the rendering. This tests the first hop.
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 to /404, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/404" {
		t.Errorf("expected redirect to /404, got %q", loc)
	}
}

func TestHandler_NotFound_LoopPrevention(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// /404 itself must not redirect (avoid loop)
	req := httptest.NewRequest(http.MethodGet, "/404", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("/404: expected 404, got %d", w.Code)
	}
}

func TestHandler_AuthenticatedMeetingRoutes(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	tests := []struct {
		path         string
		wantCode     int
		wantBodyFrag string
	}{
		{"/meetings", http.StatusOK, `"shell":"meeting-list"`},
		{"/meetings/new", http.StatusOK, `"shell":"meeting-new"`},
		{"/meetings/abc-123", http.StatusOK, `"shell":"meeting-detail"`},
		{"/meetings/abc-123/briefing", http.StatusOK, `"shell":"meeting-briefing"`},
		{"/settings", http.StatusOK, `"shell":"settings"`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("X-User-ID", "user-123")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("GET %s: expected %d, got %d", tt.path, tt.wantCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tt.wantBodyFrag) {
				t.Errorf("GET %s: expected %q in body, got %q", tt.path, tt.wantBodyFrag, w.Body.String())
			}
		})
	}
}

func TestHandler_UnauthenticatedMeetingRoutes(t *testing.T) {
	log := zerolog.New(io.Discard)
	restore := setAppShellFlag("true")
	defer restore()

	router := Handler(&log)

	// All protected routes redirect to /login when unauthenticated
	for _, path := range []string{"/meetings", "/meetings/new", "/meetings/abc-123", "/settings"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusFound {
				t.Errorf("GET %s: expected 302, got %d", path, w.Code)
			}
			if loc := w.Header().Get("Location"); loc != "/login" {
				t.Errorf("GET %s: expected redirect to /login, got %q", path, loc)
			}
		})
	}
}

// setAppShellFlag sets FF_ENABLE_APP_SHELL and returns a restore function.
func setAppShellFlag(val string) func() {
	orig := os.Getenv("FF_ENABLE_APP_SHELL")
	if val == "" {
		os.Unsetenv("FF_ENABLE_APP_SHELL")
	} else {
		os.Setenv("FF_ENABLE_APP_SHELL", val)
	}
	return func() {
		if orig == "" {
			os.Unsetenv("FF_ENABLE_APP_SHELL")
		} else {
			os.Setenv("FF_ENABLE_APP_SHELL", orig)
		}
	}
}