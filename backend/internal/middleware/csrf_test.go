package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// --- CSRFConfig tests ---

func TestCSRFConfig_IsStateChanging_DefaultsToUnsafeMethods(t *testing.T) {
	cfg := CSRFConfig{}
	unsafe := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range unsafe {
		if !cfg.IsStateChanging(m) {
			t.Errorf("default config should treat %s as state-changing", m)
		}
	}
	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions, "TRACE"}
	for _, m := range safe {
		if cfg.IsStateChanging(m) {
			t.Errorf("default config should NOT treat %s as state-changing", m)
		}
	}
}

func TestCSRFConfig_IsStateChanging_CaseInsensitive(t *testing.T) {
	cfg := CSRFConfig{}
	if !cfg.IsStateChanging("post") {
		t.Error("lowercase 'post' should be detected")
	}
	if !cfg.IsStateChanging("Post") {
		t.Error("mixed-case 'Post' should be detected")
	}
	if !cfg.IsStateChanging("  DELETE  ") {
		t.Error("padded '  DELETE  ' should be detected")
	}
}

func TestCSRFConfig_IsStateChanging_CustomMethods(t *testing.T) {
	cfg := CSRFConfig{Methods: []string{"PROPFIND"}}
	if !cfg.IsStateChanging("PROPFIND") {
		t.Error("custom method should be detected")
	}
	if cfg.IsStateChanging(http.MethodPost) {
		t.Error("POST should not be state-changing when only PROPFIND is configured")
	}
}

func TestCSRFConfig_allowedSet_TrimsAndSkipsEmpty(t *testing.T) {
	cfg := CSRFConfig{
		AllowedOrigins: []string{
			"  http://localhost:5173  ",
			"",
			"https://app.contextpilot.com",
		},
	}
	set := cfg.allowedSet()
	if _, ok := set["http://localhost:5173"]; !ok {
		t.Error("expected trimmed localhost:5173 to be in set")
	}
	if _, ok := set["https://app.contextpilot.com"]; !ok {
		t.Error("expected app.contextpilot.com to be in set")
	}
	if _, ok := set[""]; ok {
		t.Error("empty origin should not be in set")
	}
}

// --- RequestOrigin tests ---

func TestRequestOrigin_PrefersOriginHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	r.Header.Set("Origin", "https://app.contextpilot.com")
	r.Header.Set("Referer", "https://malicious.example.com/page")
	if got := RequestOrigin(r); got != "https://app.contextpilot.com" {
		t.Errorf("Origin header should win, got %q", got)
	}
}

func TestRequestOrigin_FallsBackToReferer(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	r.Header.Set("Referer", "https://app.contextpilot.com/some/page?q=1")
	if got := RequestOrigin(r); got != "https://app.contextpilot.com" {
		t.Errorf("Referer-derived origin = %q, want %q", got, "https://app.contextpilot.com")
	}
}

func TestRequestOrigin_MissingBothReturnsEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	if got := RequestOrigin(r); got != "" {
		t.Errorf("expected empty origin, got %q", got)
	}
}

func TestRequestOrigin_MalformedRefererReturnsEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	r.Header.Set("Referer", "://not-a-url")
	if got := RequestOrigin(r); got != "" {
		t.Errorf("malformed Referer should yield empty origin, got %q", got)
	}
}

// --- NewCSRF middleware tests ---

func newCSRFHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestNewCSRF_DisabledIsNoop(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{Enabled: false, AllowedOrigins: []string{"http://allowed.test"}}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if !called {
		t.Error("disabled middleware should pass through")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (disabled middleware)", w.Code, http.StatusOK)
	}
}

func TestNewCSRF_SafeMethodsAlwaysPass(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	mw := NewCSRF(log, cfg)

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		called := false
		req := httptest.NewRequest(method, "/", nil)
		w := httptest.NewRecorder()
		mw(newCSRFHandler(&called)).ServeHTTP(w, req)
		if !called {
			t.Errorf("%s should always pass", method)
		}
		if w.Code != http.StatusOK {
			t.Errorf("%s status = %d, want %d", method, w.Code, http.StatusOK)
		}
	}
}

func TestNewCSRF_AllowsConfiguredOrigin_POST(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:8080"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called for allowed origin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewCSRF_RejectsEvilOrigin_POST(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if called {
		t.Error("handler must NOT be called for evil origin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
	}
	if !strings.Contains(w.Body.String(), "Origin not allowed") {
		t.Errorf("body should explain rejection; got %q", w.Body.String())
	}
}

func TestNewCSRF_RejectsEvilOrigin_PUT_PATCH_DELETE(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	mw := NewCSRF(log, cfg)

	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		called := false
		req := httptest.NewRequest(method, "/api/v1/meetings/abc", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()
		mw(newCSRFHandler(&called)).ServeHTTP(w, req)
		if called {
			t.Errorf("%s must NOT call handler for evil origin", method)
		}
		if w.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d", method, w.Code, http.StatusForbidden)
		}
	}
}

func TestNewCSRF_FallsBackToRefererWhenNoOrigin(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.contextpilot.com"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Referer", "https://app.contextpilot.com/some/page")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called when Referer-derived origin is allowed")
	}
}

func TestNewCSRF_RejectsEvilRefererWhenNoOrigin(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.contextpilot.com"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Referer", "https://evil.com/exploit")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if called {
		t.Error("handler must NOT be called for evil Referer")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestNewCSRF_RejectsWhenBothOriginAndRefererMissing(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if called {
		t.Error("handler must NOT be called when both Origin and Referer are missing")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), "Missing Origin and Referer") {
		t.Errorf("body should explain missing-header rejection; got %q", w.Body.String())
	}
}

func TestNewCSRF_AllowsSecondConfiguredOrigin(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:8080"},
	}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called for the second configured origin")
	}
}

func TestNewCSRF_OriginMatchIsExact(t *testing.T) {
	// Substring, scheme-less, and trailing-path forms must NOT match.
	// This protects against the classic "evil.com?allowed=foo" bypass.
	log := zerolog.New(nil)
	cfg := CSRFConfig{
		Enabled:        true,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	mw := NewCSRF(log, cfg)

	bad := []string{
		"http://localhost:5173.evil.com",
		"http://evil.com?x=http://localhost:5173",
		"localhost:5173",
		"http://localhost:5173/",
		"http://localhost:5173/a",
		"https://localhost:5173",
	}
	for _, origin := range bad {
		called := false
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		mw(newCSRFHandler(&called)).ServeHTTP(w, req)
		if called {
			t.Errorf("origin %q must NOT match (substring/scheme/path confusion)", origin)
		}
		if w.Code != http.StatusForbidden {
			t.Errorf("origin %q: status = %d, want %d", origin, w.Code, http.StatusForbidden)
		}
	}
}

func TestNewCSRF_EmptyAllowedListRejectsAll(t *testing.T) {
	log := zerolog.New(nil)
	cfg := CSRFConfig{Enabled: true, AllowedOrigins: nil}
	mw := NewCSRF(log, cfg)

	called := false
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "http://anything.test")
	w := httptest.NewRecorder()
	mw(newCSRFHandler(&called)).ServeHTTP(w, req)

	if called {
		t.Error("empty allowlist must reject all POSTs")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
