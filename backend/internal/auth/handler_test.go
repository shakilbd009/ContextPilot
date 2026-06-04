// Tests for the auth handler. These cover the security properties
// fixed in F4 from t_793ea842:
//
//  1. The Set-Cookie header for session_id carries HttpOnly, Secure,
//     and SameSite=Strict.
//  2. The X-User-ID is derived server-side from the session token,
//     NOT from the email. Two different logins with the same email
//     receive two different X-User-IDs.
//  3. The response never embeds the session token in the JSON body
//     (otherwise the JS layer would see it).
//  4. Logout clears both cookies.
//  5. The handler refuses to issue a session when the feature flag
//     is disabled.
//  6. Method validation: GET on /login → 405.
//  7. Body validation: empty email or password → 400.
package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// newTestHandler returns an auth Handler with the feature flag set so
// the happy-path tests can run. Tests that exercise the disabled path
// override the env via t.Setenv.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv(FeatureFlagEnv, "true")
	// log.Logger is fine for tests — we just discard the output.
	return Handler(defaultTestLogger())
}

// defaultTestLogger returns a *zerolog.Logger that discards output.
// Tests don't assert on logs; they only need a non-nil pointer so
// Handler's signature is satisfied.
func defaultTestLogger() *zerolog.Logger {
	logger := zerolog.New(io.Discard)
	return &logger
}

// TestLoginIssuesHttpOnlySessionCookie is the headline F4 test: the
// Set-Cookie header for session_id must carry HttpOnly, Secure, and
// SameSite=Strict. This is the change that closes the XSS / downgrade
// / CSRF exposure described in the F4 finding.
func TestLoginIssuesHttpOnlySessionCookie(t *testing.T) {
	h := newTestHandler(t)

	body := strings.NewReader(`{"email":"alice@example.com","password":"hunter2"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body=%q)", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 Set-Cookie headers (session_id + X-User-ID), got %d", len(cookies))
	}

	// Build a name -> cookie map for easy assertion.
	byName := make(map[string]*http.Cookie, len(cookies))
	for _, c := range cookies {
		byName[c.Name] = c
	}

	session, ok := byName[CookieSessionID]
	if !ok {
		t.Fatalf("expected Set-Cookie for %s, got headers: %v",
			CookieSessionID, cookieNames(cookies))
	}
	if !session.HttpOnly {
		t.Errorf("%s cookie is missing HttpOnly (XSS defense)", CookieSessionID)
	}
	if !session.Secure {
		t.Errorf("%s cookie is missing Secure (downgrade defense)", CookieSessionID)
	}
	if session.SameSite != http.SameSiteStrictMode {
		t.Errorf("%s cookie SameSite = %v, want %v (CSRF defense)",
			CookieSessionID, session.SameSite, http.SameSiteStrictMode)
	}
	if session.Value == "" {
		t.Errorf("%s cookie value is empty", CookieSessionID)
	}
	// And — critically — the value must be the opaque random token,
	// never the literal string "demo-session". The placeholder was
	// the whole point of the finding.
	if session.Value == "demo-session" {
		t.Errorf("%s cookie still carries the placeholder value 'demo-session'", CookieSessionID)
	}
	if session.Path != "/" {
		t.Errorf("%s cookie Path = %q, want %q", CookieSessionID, session.Path, "/")
	}

	user, ok := byName[CookieUserID]
	if !ok {
		t.Fatalf("expected Set-Cookie for %s, got headers: %v",
			CookieUserID, cookieNames(cookies))
	}
	if !user.HttpOnly {
		t.Errorf("%s cookie is missing HttpOnly", CookieUserID)
	}
	if !user.Secure {
		t.Errorf("%s cookie is missing Secure", CookieUserID)
	}
	if user.SameSite != http.SameSiteStrictMode {
		t.Errorf("%s cookie SameSite = %v, want %v",
			CookieUserID, user.SameSite, http.SameSiteStrictMode)
	}
	// The X-User-ID is a UUID, not the email and not a SHA-256 of the
	// email. The original F4 demo used SHA-256(email) which is fully
	// derivable from a known email.
	if user.Value == "" {
		t.Errorf("%s cookie value is empty", CookieUserID)
	}
	if !looksLikeUUID(user.Value) {
		t.Errorf("%s cookie value = %q, want a UUID", CookieUserID, user.Value)
	}
	if user.Value == "alice@example.com" {
		t.Errorf("%s cookie value is the raw email — F4 not fixed", CookieUserID)
	}
}

// TestXUserIDIsDerivedFromSessionNotEmail proves that two logins with
// the same email receive different X-User-IDs (because the session
// token is fresh random each time, and the user ID is derived from
// the session token, not the email).
func TestXUserIDIsDerivedFromSessionNotEmail(t *testing.T) {
	h := newTestHandler(t)

	doLogin := func() string {
		body := strings.NewReader(`{"email":"same@example.com","password":"hunter2"}`)
		req := httptest.NewRequest(http.MethodPost, "/login", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("login failed: %d %q", rec.Code, rec.Body.String())
		}
		for _, c := range rec.Result().Cookies() {
			if c.Name == CookieUserID {
				return c.Value
			}
		}
		t.Fatalf("no %s cookie in response", CookieUserID)
		return ""
	}

	first := doLogin()
	second := doLogin()
	if first == second {
		t.Errorf("expected different X-User-IDs for two logins of the same email, both = %q", first)
	}
}

// TestResponseBodyDoesNotLeakSessionToken guards against a regression
// where a future maintainer might add the session token to the JSON
// response for "convenience". The whole point of HttpOnly is that the
// JS layer never sees the token, so the response body must not
// include it either.
func TestResponseBodyDoesNotLeakSessionToken(t *testing.T) {
	h := newTestHandler(t)

	body := strings.NewReader(`{"email":"bob@example.com","password":"hunter2"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// The response body is JSON; decode and assert the shape.
	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%q)", err, rec.Body.String())
	}
	if resp.UserID == "" {
		t.Errorf("response.userId is empty")
	}
	// The response must not include the session_id. (It includes
	// userId — the derived UUID, not the session token.)
	bodyStr := rec.Body.String()
	if strings.Contains(strings.ToLower(bodyStr), "session_id") {
		t.Errorf("response body mentions 'session_id' (must not leak token): %q", bodyStr)
	}
	if strings.Contains(bodyStr, "demo-session") {
		t.Errorf("response body contains the placeholder 'demo-session': %q", bodyStr)
	}
}

// TestLogoutClearsCookies verifies the /logout path emits Set-Cookie
// with Max-Age=-1 for both session_id and X-User-ID, so the browser
// removes them.
func TestLogoutClearsCookies(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("logout: expected 204, got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("logout: expected 2 Set-Cookie headers, got %d", len(cookies))
	}
	for _, c := range cookies {
		if c.Name != CookieSessionID && c.Name != CookieUserID {
			t.Errorf("logout: unexpected cookie %q", c.Name)
		}
		if c.MaxAge >= 0 {
			t.Errorf("logout: cookie %q MaxAge = %d, want <0", c.Name, c.MaxAge)
		}
		if c.Value != "" {
			t.Errorf("logout: cookie %q value = %q, want empty", c.Name, c.Value)
		}
	}
}

// TestLoginRefusedWhenFlagDisabled proves the handler is gated by
// the feature flag — if FF_ENABLE_APP_SHELL is off, no session is
// issued. This is the safety valve for the "ship the same binary in
// production and dev" pattern.
func TestLoginRefusedWhenFlagDisabled(t *testing.T) {
	t.Setenv(FeatureFlagEnv, "false")
	h := Handler(defaultTestLogger())

	body := strings.NewReader(`{"email":"alice@example.com","password":"hunter2"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when feature flag off, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("expected problem+json content type, got %q", got)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("expected no Set-Cookie when feature flag is off, got %d",
			len(rec.Result().Cookies()))
	}
}

// TestLoginRejectsNonPOST makes sure GET (or any other method) on
// /login returns 405 with an Allow header, so a misconfigured client
// gets a clear error instead of a silent failure.
func TestLoginRejectsNonPOST(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /login, got %d", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "POST" {
		t.Errorf("Allow header = %q, want POST", got)
	}
}

func TestHandlerWorksWhenMountedUnderAPIPrefix(t *testing.T) {
	t.Setenv(FeatureFlagEnv, "true")

	root := chi.NewRouter()
	root.Mount("/api/v1/auth", Handler(defaultTestLogger()))

	body := strings.NewReader(`{"email":"test@example.com","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("mounted login status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("mounted login did not issue session cookies")
	}
}

// TestLoginRejectsMissingFields guards the body-validation branch —
// an empty email or password must produce 400, not a 200 with a
// junk session.
func TestLoginRejectsMissingFields(t *testing.T) {
	h := newTestHandler(t)

	cases := []struct {
		name string
		body string
	}{
		{"empty email", `{"email":"","password":"hunter2"}`},
		{"empty password", `{"email":"alice@example.com","password":""}`},
		{"missing email field", `{"password":"hunter2"}`},
		{"malformed email", `{"email":"not-an-email","password":"hunter2"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %s, got %d (body=%q)",
					tc.name, rec.Code, rec.Body.String())
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Errorf("expected no Set-Cookie on validation failure, got %d",
					len(rec.Result().Cookies()))
			}
		})
	}
}

// TestSignupRequiresName enforces the signup-only contract: name is
// mandatory on /signup but optional on /login.
func TestSignupRequiresName(t *testing.T) {
	h := newTestHandler(t)

	body := strings.NewReader(`{"email":"new@example.com","password":"hunter2"}`)
	req := httptest.NewRequest(http.MethodPost, "/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for signup without name, got %d (body=%q)",
			rec.Code, rec.Body.String())
	}
}

// TestSignupWithNameIssuesSession is the happy path for /signup —
// the name flows in and back out, both cookies are set.
func TestSignupWithNameIssuesSession(t *testing.T) {
	h := newTestHandler(t)

	body := strings.NewReader(`{"email":"new@example.com","password":"hunter2","name":"New User"}`)
	req := httptest.NewRequest(http.MethodPost, "/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%q)", rec.Code, rec.Body.String())
	}
	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if resp.Name != "New User" {
		t.Errorf("response.name = %q, want %q", resp.Name, "New User")
	}
	if !looksLikeUUID(resp.UserID) {
		t.Errorf("response.userId = %q, want UUID", resp.UserID)
	}
}

// ── helpers ───────────────────────────────────────────────────────

// cookieNames returns just the names of the given cookies. Used in
// test failure messages so we don't dump the full cookie struct.
func cookieNames(cs []*http.Cookie) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}

// looksLikeUUID accepts a canonical 8-4-4-4-12 hex UUID. The server
// issues uuid.NewSHA1 outputs, which match this pattern.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !isHex(r) {
				return false
			}
		}
	}
	return true
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
