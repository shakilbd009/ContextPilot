package appshell

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/contextpilot/backend/internal/featureflag"
	"github.com/contextpilot/backend/internal/problem"
)

// FeatureFlagEnv is the server-side env var for the app shell flag.
const FeatureFlagEnv = "FF_ENABLE_APP_SHELL"

// Handler returns a chi router with all app-shell routes registered.
// All routes are gated by FF_ENABLE_APP_SHELL (defaults false).
// Protected routes redirect unauthenticated users to /login.
// Public routes are /, /login, /signup, and /404.
//
// The log parameter is used for the not-found redirect handler only; pass
// log.Logger from the main package. In Phase 2+ this will be removed in favor
// of a proper structured logger injected via request context.
func Handler(log *zerolog.Logger) http.Handler {
	r := chi.NewRouter()

	// --- Middleware scoped to app-shell routes ---

	// Correlation ID injection for all app-shell requests.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = NewCorrelationID()
			}
			// Always set both headers so clients can always read the ID from response.
			w.Header().Set("X-Request-ID", reqID)
			w.Header().Set("X-Correlation-ID", reqID)
			next.ServeHTTP(w, r)
		})
	})

	// Feature flag gate — if disabled, every route returns 503 with a problem body.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			enabled := featureflag.IsEnabled(FeatureFlagEnv)
			elapsed := time.Since(start).Microseconds()

			GlobalMetrics.RecordFlagEval(FeatureFlagEnv, boolStr(enabled), elapsed)

			if !enabled {
				problem := problem.Problem{
					Type:    "https://docs.contextpilot.ai/errors/app-shell-disabled",
					Title:   "App Shell Unavailable",
					Status:  http.StatusServiceUnavailable,
					Detail:  "The app shell feature flag is not enabled. Set FF_ENABLE_APP_SHELL=true to activate.",
					Instance: r.RequestURI,
				}
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusServiceUnavailable)
				// #nosec G705 -- shell body with internal id interpolation
				fmt.Fprintf(w, `{"type":"%s","title":"%s","status":%d,"detail":"%s","instance":"%s"}`,
					problem.Type, problem.Title, problem.Status, problem.Detail, problem.Instance)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Auth guard — redirect unauthenticated users on protected routes.
	// Phase 1: auth is a header presence check (X-User-ID).
	// In Phase 2 this will validate a real session/JWT.
	r.Group(func(protected chi.Router) {
		protected.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !isAuthenticated(r) {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		// Dashboard — root when authenticated. Redirect to /dashboard to avoid
		// a loop when the root is also the entry point for unauthenticated users.
		protected.Get("/", func(w http.ResponseWriter, r *http.Request) {
			GlobalMetrics.RecordNav("/dashboard")
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		})

		// Meeting list
		protected.Get("/meetings", func(w http.ResponseWriter, r *http.Request) {
			GlobalMetrics.RecordNav("/meetings")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"meeting-list","route":"/meetings"}`))
		})

		// New meeting
		protected.Get("/meetings/new", func(w http.ResponseWriter, r *http.Request) {
			GlobalMetrics.RecordNav("/meetings/new")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"meeting-new","route":"/meetings/new"}`))
		})

		// Meeting detail
		protected.Get("/meetings/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			GlobalMetrics.RecordNav("/meetings/{id}")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// #nosec G705 -- shell JSON body, internal id from chi URL param;
			// JSON encoding prevents any taint-flow misinterpretation.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"shell": "meeting-detail",
				"route": "/meetings/" + id,
			})
		})

		// Pre-call briefing
		protected.Get("/meetings/{id}/briefing", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			GlobalMetrics.RecordNav("/meetings/{id}/briefing")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// #nosec G705 -- shell JSON body, internal id from chi URL param;
			// JSON encoding prevents any taint-flow misinterpretation.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"shell": "meeting-briefing",
				"route": "/meetings/" + id + "/briefing",
			})
		})

		// Settings
		protected.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			GlobalMetrics.RecordNav("/settings")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"settings","route":"/settings"}`))
		})
	})

	// Public routes — no auth required.
	r.Group(func(public chi.Router) {
		// Landing page (unauthenticated root)
		public.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"landing","route":"/"}`))
		})

		// Login
		public.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"login","route":"/login"}`))
		})

		// Signup
		public.Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"shell":"signup","route":"/signup"}`))
		})

		// 404 — friendly not-found
		public.Get("/404", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"shell":"not-found","route":"/404"}`))
		})
	})

	// Catch-all 404 — rewrite to /404 so the route renders the friendly page.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// Avoid redirect loop for /404 itself.
		if r.RequestURI == "/404" {
			problem.NotFound("route not found").JSON(w)
			return
		}
		log.Debug().Str("uri", r.RequestURI).Msg("app-shell 404")
		http.Redirect(w, r, "/404", http.StatusFound)
	})

	return r
}

// isAuthenticated returns true if the request carries a valid auth identity.
// Phase 1: checks for X-User-ID header. Phase 2+ will validate JWT/session.
func isAuthenticated(r *http.Request) bool {
	// Defensive: empty user ID is unauthenticated.
	return strings.TrimSpace(r.Header.Get("X-User-ID")) != ""
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}