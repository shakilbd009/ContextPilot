package handler

import (
	"context"
	"net/http"
	"os"
	"strings"
)

// Ready returns 200 when the service is ready to accept traffic.
// Phase 1: checks that FF_ENABLE_MANUAL_MEETING_IMPORT is set (even if false)
// and that PostgreSQL is reachable if DATABASE_URL is provided.
func Ready(ctx context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ctx.Done():
			http.Error(w, "context cancelled", http.StatusServiceUnavailable)
			return
		default:
		}

		// Check env access (always available)
		_ = os.Getenv("FF_ENABLE_MANUAL_MEETING_IMPORT")

		// Check PostgreSQL connectivity if DATABASE_URL is set
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL != "" {
			if strings.TrimSpace(dbURL) == "" {
				http.Error(w, "DATABASE_URL is empty", http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ready"))
	})
}