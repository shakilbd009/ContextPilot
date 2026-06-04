package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog"
)

// CSRFConfig configures the CSRF/Origin-check middleware.
//
// CSRF defense-in-depth is implemented as an Origin/Referer allowlist
// applied to state-changing methods (POST, PUT, PATCH, DELETE). This is
// the recommended baseline for SPA + cookie-authenticated APIs because
// SameSite=Lax (the default in most browsers) still permits top-level
// form navigations, which means a malicious site can auto-submit a form
// targeting our endpoints and the browser will attach the session cookie.
//
// Per the ContextPilot security baseline (docs/security-baseline.md §
// "Authentication & Sessions"), state-changing operations must be
// protected by an Origin/Referer allowlist at minimum; the double-submit
// token pattern is enforced in the SvelteKit hook layer.
type CSRFConfig struct {
	// AllowedOrigins is the set of fully-qualified origins that are
	// permitted to issue state-changing requests. Comparison is exact
	// string match against the Origin header (preferred) or the origin
	// derived from the Referer header (fallback for legacy clients that
	// only send Referer).
	//
	// Examples:
	//   http://localhost:5173   (SvelteKit dev)
	//   http://localhost:8080   (Go backend direct)
	//   https://app.contextpilot.com
	AllowedOrigins []string

	// Methods is the set of HTTP methods considered "state-changing"
	// and therefore subject to Origin/Referer enforcement. Defaults to
	// POST, PUT, PATCH, DELETE.
	Methods []string

	// Enabled allows callers to disable the middleware (returns the
	// next handler unchanged) without removing it from the chain.
	Enabled bool
}

// IsStateChanging reports whether the given method is one of the
// configured state-changing methods. The default method set is
// {POST, PUT, PATCH, DELETE} when cfg.Methods is empty.
func (c CSRFConfig) IsStateChanging(method string) bool {
	methods := c.Methods
	if len(methods) == 0 {
		methods = []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	for _, m := range methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// allowedSet is a small helper that returns the set of allowed origins
// for O(1) lookup. Treats the slice as authoritative; empty values are
// skipped.
func (c CSRFConfig) allowedSet() map[string]struct{} {
	set := make(map[string]struct{}, len(c.AllowedOrigins))
	for _, o := range c.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		set[o] = struct{}{}
	}
	return set
}

// RequestOrigin returns the effective origin of the request. It
// prefers the Origin header (sent by fetch/XHR/most modern clients)
// and falls back to the origin derived from the Referer header for
// legacy clients that don't send Origin.
//
// Returns "" when neither header is present or the Referer is not a
// valid URL — in which case the caller should treat the request as
// unverified.
func RequestOrigin(r *http.Request) string {
	if o := strings.TrimSpace(r.Header.Get("Origin")); o != "" {
		return o
	}
	if ref := strings.TrimSpace(r.Header.Get("Referer")); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			return strings.TrimSpace(u.Scheme + "://" + u.Host)
		}
	}
	return ""
}

// NewCSRF returns a chi middleware that enforces the CSRF Origin/Referer
// allowlist on state-changing methods. GET, HEAD, OPTIONS, and any
// method not in the configured set are passed through untouched.
//
// The middleware is defense-in-depth: it is expected that the SvelteKit
// /api/* layer applies the same check at the SvelteKit hooks layer
// (see frontend/src/hooks.server.ts). If the SvelteKit check is
// bypassed (e.g. an attacker talks directly to the Go backend), this
// middleware is the second line of defense.
//
// On rejection the middleware responds with 403 Forbidden and a
// problem+json body, logs at Warn level, and does NOT call the next
// handler. Safe methods always pass through with no logging.
func NewCSRF(log zerolog.Logger, cfg CSRFConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	allowed := cfg.allowedSet()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.IsStateChanging(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			origin := RequestOrigin(r)
			if origin == "" {
				log.Warn().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Str("remote_addr", r.RemoteAddr).
					Msg("csrf: missing Origin and Referer on state-changing request")
				writeCSRFProblem(w, "Missing Origin and Referer headers on state-changing request.")
				return
			}
			if _, ok := allowed[origin]; !ok {
				log.Warn().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Str("origin", origin).
					Str("remote_addr", r.RemoteAddr).
					Msg("csrf: origin not allowed")
				writeCSRFProblem(w, "Origin not allowed for this endpoint.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writeCSRFProblem writes a 403 problem+json body.
func writeCSRFProblem(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusForbidden)
	body := `{"type":"about:blank","title":"Forbidden","status":403,"detail":"` + detail + `"}`
	_, _ = w.Write([]byte(body))
}
