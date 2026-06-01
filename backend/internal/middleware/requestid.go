package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// requestIDKey is a private type to avoid context key collisions.
type requestIDKey struct{}

// WithRequestID injects a UUID request ID into the context.
// It also sets the X-Request-ID response header if not already present.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			b := make([]byte, 16)
			rand.Read(b)
			reqID = hex.EncodeToString(b)
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID returns the request ID from a context.Context.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}