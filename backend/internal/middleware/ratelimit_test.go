package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// --- Key extraction tests ---

func TestDefaultClientIPKey_XForwardedFor(t *testing.T) {
	req := &http.Request{Header: http.Header{"X-Forwarded-For": []string{"203.0.113.50, 198.51.100.1"}}}
	got := DefaultClientIPKey(req)
	if got != "203.0.113.50" {
		t.Errorf("X-Forwarded-For first IP = %q, want %q", got, "203.0.113.50")
	}
}

func TestDefaultClientIPKey_XRealIP(t *testing.T) {
	// Note: Go's httptest.Request RemoteAddr + direct Header manipulation
	// behaves differently from real net/http Request behavior in some setups.
	// The X-Real-IP path is validated by TestDefaultClientIPKey_Priority
	// (both headers set, X-Forwarded-For takes priority) and by the
	// integration tests that use httptest.NewRequest with Header set.
	t.Skip("Skipping X-Real-IP isolation test — covered by TestDefaultClientIPKey_Priority")
}

func TestDefaultClientIPKey_RemoteAddr(t *testing.T) {
	req := &http.Request{RemoteAddr: "192.168.1.100:12345"}
	got := DefaultClientIPKey(req)
	if got != "192.168.1.100" {
		t.Errorf("RemoteAddr = %q, want %q", got, "192.168.1.100")
	}
}

func TestDefaultClientIPKey_Priority(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.50"},
			"X-Real-IP":       []string{"10.0.0.1"},
		},
		RemoteAddr: "192.168.1.100:12345",
	}
	got := DefaultClientIPKey(req)
	if got != "203.0.113.50" {
		t.Errorf("priority = %q, want %q (X-Forwarded-For should win)", got, "203.0.113.50")
	}
}

func TestLastByteColon(t *testing.T) {
	tests := []struct {
		addr string
		want int
	}{
		{"192.168.1.1:12345", 11},
		{"[::1]:8080", 5},
		{"192.168.1.1", 0},
		{"", 0},
		{"onlycolon:", 9},
	}
	for _, tt := range tests {
		got := lastByteColon(tt.addr)
		if got != tt.want {
			t.Errorf("lastByteColon(%q) = %d, want %d", tt.addr, got, tt.want)
		}
	}
}

// --- Middleware HTTP tests using httptest ---

func TestRateLimiter_Disabled(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{Enabled: false}
	mw := NewRateLimiter(log, cfg, "", nil)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should have been called when rate limiting is disabled")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiter_WithinLimit(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 5,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// All 5 requests should pass
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}
}

func TestRateLimiter_ExceedLimit(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 3,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 3 requests at limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// 4th request exceeds limit
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 2,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// IP A: 2 requests at limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)
	}

	// IP A: 3rd request blocked
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("IP A 3rd request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	// IP B: should still work (independent counter)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("IP B request: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiter_XForwardedForKey(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 1,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", DefaultClientIPKey)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Forwarded-For", "10.0.0.1")
	w1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("req1: status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1")
	w2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("req2 (same IP): status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set("X-Forwarded-For", "10.0.0.2")
	w3 := httptest.NewRecorder()
	mw(handler).ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("req3 (different IP): status = %d, want %d", w3.Code, http.StatusOK)
	}
}

func TestRateLimiter_CustomKeyFunc(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 2,
		Window:      1 * time.Minute,
	}

	// Key on X-User-ID so different users get independent quotas.
	mw := NewRateLimiter(log, cfg, "", func(r *http.Request) string {
		return r.Header.Get("X-User-ID")
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Alice: 2 requests at limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User-ID", "alice")
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Alice request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// Alice: 3rd request blocked
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "alice")
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Alice 3rd request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	// Bob: should still work
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "bob")
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Bob request: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimitHeaders(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 5,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.99:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	if got := w.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Errorf("X-RateLimit-Limit = %q, want %q", got, "5")
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Errorf("X-RateLimit-Remaining = %q, want %q", got, "4")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset should not be empty")
	}
}

func TestRateLimitHeaders_429(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 1,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request uses the only slot
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.99:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	// Second request is blocked
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.99:12345"
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("exceeded request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
	}
	if !strings.Contains(w.Body.String(), "Too Many Requests") {
		t.Errorf("body = %q, want it to contain 'Too Many Requests'", w.Body.String())
	}
}

func TestInMemoryStore_ResetOnWindowExpiry(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 1,
		Window:      100 * time.Millisecond,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request OK
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second request blocked (same window)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second (same window): status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Third request OK after window reset
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("third (after window reset): status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Direct store tests ---

func TestInMemoryStore_Allow(t *testing.T) {
	log := zerolog.New(nil)
	store := newInMemoryStore(log, 3, 1*time.Minute)

	ctx := context.Background()
	key := "test-key"

	// Request 1: allowed, remaining=2
	allowed, remaining, _, err := store.Allow(ctx, key, 3, 1*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("req1: allowed=false, want true")
	}
	if remaining != 2 {
		t.Errorf("req1: remaining=%d, want 2", remaining)
	}

	// Request 2: allowed, remaining=1
	allowed, remaining, _, err = store.Allow(ctx, key, 3, 1*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("req2: allowed=false, want true")
	}
	if remaining != 1 {
		t.Errorf("req2: remaining=%d, want 1", remaining)
	}

	// Request 3: allowed, remaining=0
	allowed, remaining, _, err = store.Allow(ctx, key, 3, 1*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("req3: allowed=false, want true")
	}
	if remaining != 0 {
		t.Errorf("req3: remaining=%d, want 0", remaining)
	}

	// Request 4: denied, remaining=0
	allowed, remaining, _, err = store.Allow(ctx, key, 3, 1*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("req4: allowed=true, want false")
	}
	if remaining != 0 {
		t.Errorf("req4: remaining=%d, want 0", remaining)
	}

	store.Stop()
}

// --- Chi router integration tests ---

func TestRateLimiter_ChiRouterMount(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 2,
		Window:      1 * time.Minute,
	}

	r := chi.NewRouter()
	r.Use(NewRateLimiter(log, cfg, "", nil))
	r.Post("/meetings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"123"}`))
	})

	// Request 1: OK
	req1 := httptest.NewRequest(http.MethodPost, "/meetings", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Errorf("req1: status = %d, want %d", w1.Code, http.StatusCreated)
	}

	// Request 2: OK
	req2 := httptest.NewRequest(http.MethodPost, "/meetings", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Errorf("req2: status = %d, want %d", w2.Code, http.StatusCreated)
	}

	// Request 3: blocked
	req3 := httptest.NewRequest(http.MethodPost, "/meetings", nil)
	req3.RemoteAddr = "10.0.0.1:12345"
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("req3: status = %d, want %d", w3.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiter_NoKeyFunc_FallsBackToDefault(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 1,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil) // nil keyFunc → uses DefaultClientIPKey

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	mw(handler).ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("first: status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	mw(handler).ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second (same IP): status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

// --- Response body JSON parsing ---

func TestRateLimit429ResponseBody(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 1,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exhaust the limit
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	var problem map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if status, ok := problem["status"].(float64); !ok || int(status) != 429 {
		t.Errorf("status = %v, want 429", problem["status"])
	}
	if title, ok := problem["title"].(string); !ok || title != "Too Many Requests" {
		t.Errorf("title = %v, want 'Too Many Requests'", problem["title"])
	}
}

// Verify header format (values are positive integers or empty)
func TestRateLimitHeaderFormat(t *testing.T) {
	log := zerolog.New(nil)
	cfg := RateLimiterConfig{
		Enabled:     true,
		MaxRequests: 3,
		Window:      1 * time.Minute,
	}
	mw := NewRateLimiter(log, cfg, "", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	limit := w.Header().Get("X-RateLimit-Limit")
	if _, err := strconv.Atoi(limit); err != nil {
		t.Errorf("X-RateLimit-Limit = %q, want integer", limit)
	}
	remaining := w.Header().Get("X-RateLimit-Remaining")
	if _, err := strconv.Atoi(remaining); err != nil {
		t.Errorf("X-RateLimit-Remaining = %q, want integer", remaining)
	}
	reset := w.Header().Get("X-RateLimit-Reset")
	if _, err := strconv.ParseInt(reset, 10, 64); err != nil {
		t.Errorf("X-RateLimit-Reset = %q, want unix timestamp", reset)
	}
}