package middleware

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// Recover catches panics and logs them with stack trace.
// It writes a minimal 500 problem response so the client sees a clean error.
func Recover(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					log.Error().
						Interface("panic", err).
						Str("stack", string(stack)).
						Msg("panic recovered")
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// StructuredLogging returns middleware that logs every request.
// The actual logging happens via the chi router's logger middleware in main.go.
func StructuredLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// WithContext is a no-op placeholder for Phase 1.
// In Phase 2+ it will provide typed context accessors.
func WithContext(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}