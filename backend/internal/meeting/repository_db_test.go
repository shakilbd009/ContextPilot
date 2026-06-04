package meeting

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

// ─── Real-DB integration tests ──────────────────────────────────────────────
//
// These tests exercise the real *pgxpool.Pool against the live database. They
// are the only tests in the meeting package that can catch schema/SQL
// mismatches like the PATCH 500 caused by UpdateMeeting referencing a
// `meetings.updatedat` column that did not exist (CWE-209, fixed in
// migration 000007). Mock-based tests cannot reproduce this class of bug.
//
// Skipped automatically when DATABASE_URL is not set so `go test ./...`
// passes in environments without a running database (CI gates run with DB
// available; the test surfaces the real bug, not a mock happy-path).

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping real-DB test")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return pool
}

// TestRepository_UpdateMeeting_RealDB is the regression test for the PATCH 500
// reported by ethical-hacker task t_6af3f7c8. Before the fix, UpdateMeeting
// referenced meetings.updatedat which did not exist in the schema; this test
// exercises the real SQL path and asserts (1) no error, (2) updatedat moved.
func TestRepository_UpdateMeeting_RealDB(t *testing.T) {
	pool := openTestPool(t)
	repo := NewRepository(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Insert a meeting directly via SQL so we control the seed state. We
	// deliberately bypass CreateMeeting to avoid coupling this regression
	// test to unrelated helpers; the goal is to assert the UPDATE path
	// works against the real schema, not to retest CreateMeeting.
	ownerID := uuid.New()
	completedAt := time.Now().UTC().Truncate(time.Second)
	meetingID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO meetings
		    (id, title, completedat, transcript, notes, contentsource, createdat, createdby, displayorder)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, 0)
	`, meetingID, "original title", completedAt, "original transcript", "original notes", "transcript", ownerID)
	if err != nil {
		t.Fatalf("insert seed meeting: %v", err)
	}
	t.Cleanup(func() {
		// Best-effort cleanup; ignore errors so the test result isn't masked.
		_, _ = pool.Exec(context.Background(), `DELETE FROM meetings WHERE id = $1`, meetingID)
	})

	// Capture updatedat before
	var beforeUpdatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updatedat FROM meetings WHERE id = $1`, meetingID,
	).Scan(&beforeUpdatedAt); err != nil {
		t.Fatalf("read updatedat before: %v", err)
	}

	// Small sleep guarantees the new updatedat will be strictly greater
	// than the seeded value (PostgreSQL NOW() has microsecond resolution
	// but timestamptz comparisons collapse to microsecond precision;
	// sleeping a few ms is the reliable cross-platform approach).
	time.Sleep(10 * time.Millisecond)

	// Perform the update that triggered the original 500
	newTitle := "patched title"
	hasChanges, err := repo.UpdateMeeting(ctx, meetingID, &newTitle, nil, nil, nil, nil)
	if err != nil {
		// Wrap the error so the message survives t.Fatalf cleanly.
		t.Fatalf("UpdateMeeting returned error (this is the original 500 bug): %v", err)
	}
	if !hasChanges {
		t.Errorf("hasChanges = false, want true (title changed from %q to %q)", "original title", newTitle)
	}

	// Assert title was updated
	var gotTitle string
	if err := pool.QueryRow(ctx,
		`SELECT title FROM meetings WHERE id = $1`, meetingID,
	).Scan(&gotTitle); err != nil {
		t.Fatalf("read title after: %v", err)
	}
	if gotTitle != newTitle {
		t.Errorf("title = %q, want %q", gotTitle, newTitle)
	}

	// Assert updatedat moved (this is the BRD-02 FR-10 / BRD-04 FR-15
	// contract that IsMemoryStale depends on)
	var afterUpdatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updatedat FROM meetings WHERE id = $1`, meetingID,
	).Scan(&afterUpdatedAt); err != nil {
		t.Fatalf("read updatedat after: %v", err)
	}
	if !afterUpdatedAt.After(beforeUpdatedAt) {
		t.Errorf("updatedat did not advance: before=%v after=%v", beforeUpdatedAt, afterUpdatedAt)
	}
}

// TestRepository_UpdateMeeting_NoChanges_RealDB asserts the no-op path
// (hasChanges=false when nothing changed) doesn't write updatedat. This is
// the contract IsMemoryStale relies on: a PATCH that doesn't change source
// content must not be treated as a source mutation.
func TestRepository_UpdateMeeting_NoChanges_RealDB(t *testing.T) {
	pool := openTestPool(t)
	repo := NewRepository(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerID := uuid.New()
	completedAt := time.Now().UTC().Truncate(time.Second)
	meetingID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO meetings
		    (id, title, completedat, transcript, notes, contentsource, createdat, createdby, displayorder)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, 0)
	`, meetingID, "stable title", completedAt, "stable transcript", "stable notes", "transcript", ownerID)
	if err != nil {
		t.Fatalf("insert seed meeting: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM meetings WHERE id = $1`, meetingID)
	})

	var beforeUpdatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updatedat FROM meetings WHERE id = $1`, meetingID,
	).Scan(&beforeUpdatedAt); err != nil {
		t.Fatalf("read updatedat before: %v", err)
	}

	// PATCH with the same values — repo should return hasChanges=false
	sameTitle := "stable title"
	hasChanges, err := repo.UpdateMeeting(ctx, meetingID, &sameTitle, &completedAt, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateMeeting unexpected error: %v", err)
	}
	if hasChanges {
		t.Errorf("hasChanges = true, want false (no source field actually changed)")
	}

	var afterUpdatedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updatedat FROM meetings WHERE id = $1`, meetingID,
	).Scan(&afterUpdatedAt); err != nil {
		t.Fatalf("read updatedat after: %v", err)
	}
	if !afterUpdatedAt.Equal(beforeUpdatedAt) {
		t.Errorf("updatedat changed despite no source mutation: before=%v after=%v", beforeUpdatedAt, afterUpdatedAt)
	}
}

// ─── SanitizeDBError unit tests (no DB required) ──────────────────────────

// TestSanitizeDBError_StripsSchemaDetails is the regression test for the
// server-log schema disclosure (CWE-209). It asserts that sanitizeDBError
// strips the raw pgx error message (column name, table name, SQL fragment)
// and returns only a stable, safe representation (SQLSTATE).
func TestSanitizeDBError_StripsSchemaDetails(t *testing.T) {
	// Construct a *pgconn.PgError that mimics the original PATCH 500 log
	// line: "ERROR: column \"updatedat\" of relation \"meetings\" does not
	// exist (SQLSTATE 42703)". Without sanitizeDBError, calling
	// err.Error() in zerolog would emit that whole string — including
	// the column name "updatedat" and table name "meetings" — into the
	// server log.
	pgErr := &pgconn.PgError{
		Message:    `column "updatedat" of relation "meetings" does not exist`,
		Code:       "42703",
		ColumnName: "updatedat",
		TableName:  "meetings",
	}

	got := sanitizeDBError(pgErr)
	if got != `pgx.PgError(sqlstate=42703)` {
		t.Errorf("sanitizeDBError = %q, want %q", got, `pgx.PgError(sqlstate=42703)`)
	}
	// Negative assertions: the redacted string must NOT contain schema
	// details that would have leaked before the fix.
	for _, leak := range []string{"updatedat", "meetings", "column", "relation", "does not exist"} {
		if strings.Contains(got, leak) {
			t.Errorf("sanitizeDBError still leaks %q in %q", leak, got)
		}
	}
}

// TestSanitizeDBError_NonPgError verifies non-pg errors fall back to %T so
// callers can see the Go error type without leaking internal message bodies.
func TestSanitizeDBError_NonPgError(t *testing.T) {
	plain := errors.New("some internal failure")
	got := sanitizeDBError(plain)
	if !strings.Contains(got, "errorString") {
		t.Errorf("sanitizeDBError for plain error = %q, want it to include the Go type name (e.g. *errors.errorString)", got)
	}
	if strings.Contains(got, "internal failure") {
		t.Errorf("sanitizeDBError leaked message body in %q", got)
	}
}

// TestSanitizeDBError_Nil is a smoke test: nil error returns empty string.
func TestSanitizeDBError_Nil(t *testing.T) {
	if got := sanitizeDBError(nil); got != "" {
		t.Errorf("sanitizeDBError(nil) = %q, want \"\"", got)
	}
}
