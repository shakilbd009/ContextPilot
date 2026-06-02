// Package auth implements the demo login and signup endpoints.
//
// Security properties (F4 from t_793ea842):
//   - session_id cookie is HttpOnly (XSS defense)
//   - session_id cookie is Secure (downgrade defense)
//   - session_id cookie carries SameSite=Strict (CSRF defense)
//   - X-User-ID is derived server-side from the session token, not from
//     a client-controlled input. Previously the client derived
//     X-User-ID as SHA-256(email) with no salt — anyone who knew the
//     email could recompute the "auth token" and impersonate the user.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/contextpilot/backend/internal/problem"
)

const (
	// CookieSessionID is the opaque server-issued session token.
	CookieSessionID = "session_id"
	// CookieUserID is the derived UUID identifying the authenticated user.
	CookieUserID = "X-User-ID"
)

// sessionCookieMaxAge is the demo session lifetime (8 hours).
const sessionCookieMaxAge = 8 * 60 * 60

// FeatureFlagEnv is the server-side env var for the auth endpoints.
// Auth is a core capability of the app shell, so it is always-on
// when the app shell is enabled.
const FeatureFlagEnv = "FF_ENABLE_APP_SHELL"

// loginRequest is the body shape accepted by /login and /signup.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// loginResponse is the body shape returned by /login and /signup.
type loginResponse struct {
	UserID string `json:"userId"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email"`
}

// Handler returns an http.Handler with /login, /signup, and /logout
// routes registered.
func Handler(log *zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(w, r, log)
	})
	mux.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		handleSignup(w, r, log)
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r, log)
	})
	return mux
}

func handleLogin(w http.ResponseWriter, r *http.Request, log *zerolog.Logger) {
	if !flagEnabled() {
		refuseDisabled(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "Invalid JSON body")
		return
	}
	if !validEmail(req.Email) || strings.TrimSpace(req.Password) == "" {
		badRequest(w, "Email and password are required")
		return
	}
	issueSession(w, r, log, req.Email, strings.TrimSpace(req.Name))
}

func handleSignup(w http.ResponseWriter, r *http.Request, log *zerolog.Logger) {
	if !flagEnabled() {
		refuseDisabled(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "Invalid JSON body")
		return
	}
	if !validEmail(req.Email) || strings.TrimSpace(req.Password) == "" {
		badRequest(w, "Email and password are required")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		badRequest(w, "Name is required for signup")
		return
	}
	issueSession(w, r, log, req.Email, name)
}

func handleLogout(w http.ResponseWriter, r *http.Request, log *zerolog.Logger) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	expireCookie(w, CookieSessionID)
	expireCookie(w, CookieUserID)
	w.WriteHeader(http.StatusNoContent)
	if log != nil {
		log.Debug().Str("path", r.URL.Path).Msg("auth: session cleared")
	}
}

// issueSession builds the Set-Cookie headers and the JSON response.
// Called by both /login and /signup.
func issueSession(w http.ResponseWriter, r *http.Request, log *zerolog.Logger, email, name string) {
	// Opaque session token from crypto/rand. Never derive from the
	// email; never use time.Now() as the seed.
	sessionToken, err := newOpaqueToken()
	if err != nil {
		internalError(w, "Failed to generate session token")
		return
	}

	// X-User-ID is derived from the session token, not from the
	// email. This is the change that closes F4.
	userID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("contextpilot:auth:"+sessionToken)).String()

	// Set-Cookie: session_id=<opaque>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800
	// Set-Cookie: X-User-ID=<derived-uuid>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800
	setSessionCookie(w, CookieSessionID, sessionToken, sessionCookieMaxAge)
	setSessionCookie(w, CookieUserID, userID, sessionCookieMaxAge)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginResponse{
		UserID: userID,
		Email:  email,
		Name:   name,
	})

	if log != nil {
		log.Info().
			Str("path", r.URL.Path).
			Str("user_id", userID).
			Msg("auth: session issued")
	}
}

// setSessionCookie writes a single Set-Cookie header with the
// production hardening attributes. SameSite=Strict (CSRF defense),
// HttpOnly (XSS defense), Secure (downgrade defense), Path=/ (whole
// origin).
func setSessionCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
}

// expireCookie clears a previously-issued cookie. We mirror the same
// path / SameSite / HttpOnly / Secure attributes so the browser
// matches the existing cookie.
func expireCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// newOpaqueToken returns 32 bytes of random hex (64 characters).
// 32 bytes is well above the 128-bit security floor.
func newOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// validEmail is a deliberately conservative check.
func validEmail(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 3 || len(s) > 254 {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	if strings.IndexByte(s[at+1:], '.') < 0 {
		return false
	}
	return true
}

// flagEnabled mirrors the pattern used by the other handlers.
func flagEnabled() bool {
	v := os.Getenv(FeatureFlagEnv)
	return v == "true" || v == "1"
}

// ── Error helpers ─────────────────────────────────────────────────

func badRequest(w http.ResponseWriter, detail string) {
	problem.Problem{
		Type:   "about:blank",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: detail,
	}.JSON(w)
}

func internalError(w http.ResponseWriter, detail string) {
	problem.Problem{
		Type:   "about:blank",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: detail,
	}.JSON(w)
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	problem.Problem{
		Type:   "about:blank",
		Title:  "Method Not Allowed",
		Status: http.StatusMethodNotAllowed,
		Detail: "Use " + allow,
	}.JSON(w)
}

func refuseDisabled(w http.ResponseWriter, r *http.Request) {
	problem.Problem{
		Type:     "https://docs.contextpilot.ai/errors/auth-disabled",
		Title:    "Auth Unavailable",
		Status:   http.StatusServiceUnavailable,
		Detail:   "The app shell feature flag is not enabled. Set FF_ENABLE_APP_SHELL=true to activate auth.",
		Instance: r.RequestURI,
	}.JSON(w)
}
