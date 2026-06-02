package upcoming

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// These tests exercise the production Handler() against a real PostgreSQL
// database. They are designed to catch the class of bug where a handler
// forgets to install r.Use(WithRepository(pool)) in its chi router — the
// buildTestRouter helper masks that bug by pre-injecting a repository into
// the request context, so the production middleware chain is never exercised
// by unit tests.
//
// The tests skip silently when DATABASE_URL is unset (CI without DB).
// Locally they run against the trust-auth dev DB on localhost:5432.

// requirePool returns a real *pgxpool.Pool or skips the test. It cleans up
// the connection when the test finishes.
func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test that needs a real PostgreSQL connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New(%q): %v", dsn, err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	return pool
}

// TestHandler_InstallsWithRepositoryMiddleware is the regression test for
// the /api/v1/upcoming 500 reported by the ethical-hacker subagent
// (task t_42c0144b). The previous handler created a chi router but never
// called r.Use(WithRepository(pool)), so getRepository(r) returned nil
// and every request returned 500 with log "no repository in request
// context". This test calls Handler() directly (no test wrapper) with a
// real *pgxpool.Pool and asserts the request returns 200, not 500.
func TestHandler_InstallsWithRepositoryMiddleware(t *testing.T) {
	pool := requirePool(t)

	// Enable the feature flag for this request. Use the package-level
	// testFFValue override so we don't have to mutate process env.
	origFF := testFFValue
	testFFValue = "true"
	t.Cleanup(func() { testFFValue = origFF })

	logger := zerolog.New(os.Stderr).Level(zerolog.WarnLevel)

	// Real handler, real pool, no test wrapper. This is exactly the
	// configuration main.go uses at cmd/server/main.go:109.
	router := Handler(&logger, pool)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", uuid.New().String()) // fresh user → no rows
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body=%q. expected the production "+
			"handler to inject a real repository into the request context "+
			"via r.Use(WithRepository(pool)). If this returns 500 with "+
			"log 'no repository in request context', the middleware "+
			"install line in Handler() is missing.",
			w.Code, http.StatusOK, w.Body.String())
	}

	// Empty result list should serialise as a JSON array.
	var got []UpcomingMeeting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response body: %v (body=%q)", err, w.Body.String())
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %d meetings: %+v", len(got), got)
	}
}
