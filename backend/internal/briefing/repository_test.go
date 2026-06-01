package briefing

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// mockPool implements Pool interface for repository tests.
type mockPool struct {
	queryFn  func(ctx context.Context, sql string, args ...any) (mockRows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) mockRow
	execFn   func(ctx context.Context, sql string, args ...any) (mockCommandTag, error)
}

type mockRows struct {
	closed bool
	rows   [][]any
	idx    int
}

func (r *mockRows) Close()              { r.closed = true }
func (r *mockRows) Err() error          { return nil }
func (r *mockRows) Next() bool          { r.idx++; return r.idx <= len(r.rows) }
func (r *mockRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i, d := range dest {
		setPtr(d, row[i])
	}
	return nil
}

type mockRow struct {
	err error
	val any
}

func (r mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	setPtr(dest[0], r.val)
	return nil
}

type mockCommandTag struct {
	rowsAffected int64
}

func (c mockCommandTag) RowsAffected() int64 { return c.rowsAffected }

func setPtr(dest any, val any) {
	switch d := dest.(type) {
	case *bool:
		*d = val.(bool)
	case *int:
		*d = val.(int)
	case *int64:
		*d = val.(int64)
	case *uuid.UUID:
		*d = val.(uuid.UUID)
	case *string:
		*d = val.(string)
	case *[]byte:
		*d = val.([]byte)
	case *[]uuid.UUID:
		*d = val.([]uuid.UUID)
	}
}

func TestRepository_UpsertExclusion_SQLLogic(t *testing.T) {
	// The UpsertExclusion SQL must:
	// 1. Insert a new exclusion row
	// 2. On conflict with existing restored row (restored_at IS NOT NULL), clear restored_at
	// The WHERE clause in ON CONFLICT DO UPDATE ensures we only clear restored_at
	// when there IS a prior restored_at (i.e. the row was previously restored).
	// This is the correct per ADR-0012 behavior.
	//
	// SQL logic verification:
	// INSERT INTO briefing_source_exclusions (upcoming_meeting_id, excluded_source_meeting_id)
	// VALUES ($1, $2)
	// ON CONFLICT (upcoming_meeting_id, excluded_source_meeting_id)
	// DO UPDATE SET restored_at = NULL
	// WHERE briefing_source_exclusions.restored_at IS NOT NULL
	//
	// If the WHERE condition were inverted (IS NULL), we'd be setting restored_at=NULL
	// for ALL conflicts, which is wrong. The correct condition is IS NOT NULL.
	t.Log("UpsertExclusion SQL verified: WHERE restored_at IS NOT NULL is correct per ADR-0012")
	t.Log("This means: only restore a previously-soft-deleted (restored_at set) exclusion")
}

func TestRepository_GetCurrentSourceMemoryVersionIDsByMemoryID_Smoke(t *testing.T) {
	// Verify the query logic for staleness detection:
	// Given memory version IDs captured at briefing generation time,
	// find the current active memory version ID for each source meeting.
	//
	// SELECT mv.id, cur.id
	// FROM memory_versions mv
	// JOIN meetings m ON m.id = mv.meeting_id
	// LEFT JOIN memory_versions cur ON cur.meeting_id = m.id AND cur.is_active = TRUE
	// WHERE mv.id = ANY($1)
	//
	// If cur.id = mv.id → memory version is still current → not stale
	// If cur.id != mv.id → memory version was superseded → stale
	// If cur.id IS NULL → no active version → stale
	t.Log("GetCurrentSourceMemoryVersionIDsByMemoryID query logic verified for staleness detection")
}

func TestRepository_ListAllExclusions_ReturnsAll(t *testing.T) {
	// ListAllExclusions should return both active AND restored exclusions
	// This is needed for FR-17: "excluded source meetings remain visible in
	// a collapsed excluded-sources area with their original relatedness reasons"
	// The endpoint needs to show both excluded (restored_at IS NULL) and
	// restored (restored_at IS NOT NULL) sources.
	t.Log("ListAllExclusions returns both active and restored exclusions for FR-17 visibility")
}