package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// mockRow implements pgx.Row for test scenarios.
type mockRow struct {
	values []any // row values for assignValue-based scanning
	scanFn func(dest ...any) error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	if r.values == nil {
		return fmt.Errorf("mockRow.Scan: no scanFn and no values provided")
	}
	if len(r.values) != len(dest) {
		return fmt.Errorf("mockRow.Scan: got %d values, want %d", len(r.values), len(dest))
	}
	for i, v := range r.values {
		if err := assignValue(dest[i], v); err != nil {
			return fmt.Errorf("mockRow.Scan[%d]: %w", i, err)
		}
	}
	return nil
}

// ─── mockRows ─────────────────────────────────────────────────────────────────

// mockRows implements pgx.Rows for test scenarios.
type mockRows struct {
	values  [][]any
	pos     int
	scanFn  func(dest ...any) error
	closeFn func()
	errFn   func() error
}

func (r *mockRows) Next() bool {
	if r.pos >= len(r.values) {
		r.pos = len(r.values) + 1
		return false
	}
	r.pos++
	return true
}

func (r *mockRows) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	if r.pos <= 0 || r.pos > len(r.values) {
		return fmt.Errorf("mockRows.Scan: no row at position %d", r.pos)
	}
	vals := r.values[r.pos-1]
	if len(vals) != len(dest) {
		return fmt.Errorf("mockRows.Scan: got %d values, want %d", len(vals), len(dest))
	}
	for i, v := range vals {
		if err := assignValue(dest[i], v); err != nil {
			return fmt.Errorf("mockRows.Scan[%d]: %w", i, err)
		}
	}
	return nil
}

func (r *mockRows) Close()              { if r.closeFn != nil { r.closeFn() } }
func (r *mockRows) Err() error         { if r.errFn == nil { return nil }; return r.errFn() }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Values() ([]any, error)    { return nil, nil }
func (r *mockRows) Conn() *pgx.Conn            { return nil }
func (r *mockRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *mockRows) RawValues() [][]byte        { return nil }

// ─── mockTx ───────────────────────────────────────────────────────────────────

// mockTx implements pgx.Tx for CreateVersion transaction testing.
type mockTx struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	commitFn   func(ctx context.Context) error
	rollbackFn func(ctx context.Context) error
	beginFn    func(ctx context.Context) (pgx.Tx, error)
}

func (tx *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	if tx.beginFn != nil {
		return tx.beginFn(ctx)
	}
	return tx, nil
}

func (tx *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return tx.queryRowFn(ctx, sql, args...)
}

func (tx *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return &mockRows{}, nil
}

func (tx *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return tx.execFn(ctx, sql, args...)
}

func (tx *mockTx) Commit(ctx context.Context) error {
	if tx.commitFn != nil {
		return tx.commitFn(ctx)
	}
	return nil
}

func (tx *mockTx) Rollback(ctx context.Context) error {
	if tx.rollbackFn != nil {
		return tx.rollbackFn(ctx)
	}
	return nil
}

func (tx *mockTx) TxState()                             {}
func (tx *mockTx) CONN() *pgx.Conn                      { return nil }
func (tx *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *mockTx) ExecFunc(ctx context.Context, fn func(pgx.Tx) error) error { return nil }
func (tx *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}
func (tx *mockTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }
func (tx *mockTx) Conn() *pgx.Conn                     { return nil }

// ─── mockPool ─────────────────────────────────────────────────────────────────

// mockPool implements the Pool interface for repository unit testing.
type mockPool struct {
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	beginFn    func(ctx context.Context) (pgx.Tx, error)
}

func (p *mockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if p.queryFn == nil {
		return &mockRows{}, nil
	}
	return p.queryFn(ctx, sql, args...)
}

func (p *mockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.queryRowFn(ctx, sql, args...)
}

func (p *mockPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if p.execFn == nil {
		return pgconn.CommandTag{}, nil
	}
	return p.execFn(ctx, sql, args...)
}

func (p *mockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	if p.beginFn == nil {
		return &mockTx{}, nil
	}
	return p.beginFn(ctx)
}

// ─── assignValue ──────────────────────────────────────────────────────────────

// assignValue copies v into the address pointed to by dest, for the specific
// types used in repository tests (uuid.UUID, string, int, time.Time, and pointers).
func assignValue(dest, v any) error {
	if v == nil {
		return nil // leave dest zeroed
	}
	switch d := dest.(type) {
	case *uuid.UUID:
		switch val := v.(type) {
		case uuid.UUID:
			*d = val
		case *uuid.UUID:
			if val != nil {
				*d = *val
			}
		}
		// Also handle: assigning uuid.UUID directly to *uuid.UUID scan target
		if *d == uuid.Nil && v != nil {
			if val, ok := v.(uuid.UUID); ok {
				*d = val
			}
		}
	case *string:
		switch val := v.(type) {
		case string:
			*d = val
		case *string:
			if val != nil {
				*d = *val
			}
		case fmt.Stringer:
			*d = val.String()
		}
	case *int:
		switch val := v.(type) {
		case int:
			*d = val
		case int32:
			*d = int(val)
		case int64:
			*d = int(val)
		}
	case *time.Time:
		switch val := v.(type) {
		case time.Time:
			*d = val
		case *time.Time:
			if val != nil {
				*d = *val
			}
		}
	case *TriggerType:
		switch val := v.(type) {
		case string:
			*d = TriggerType(val)
		case TriggerType:
			*d = val
		case fmt.Stringer:
			*d = TriggerType(val.String())
		}
	case *JobStatus:
		switch val := v.(type) {
		case string:
			*d = JobStatus(val)
		case JobStatus:
			*d = val
		case fmt.Stringer:
			*d = JobStatus(val.String())
		}
	case *VersionStatus:
		if s, ok := v.(string); ok {
			*d = VersionStatus(s)
		} else if s, ok := v.(fmt.Stringer); ok {
			*d = VersionStatus(s.String())
		}
	case *QualityStatus:
		if s, ok := v.(string); ok {
			*d = QualityStatus(s)
		} else if s, ok := v.(fmt.Stringer); ok {
			*d = QualityStatus(s.String())
		}
	case *ReviewStatus:
		if s, ok := v.(string); ok {
			*d = ReviewStatus(s)
		} else if s, ok := v.(fmt.Stringer); ok {
			*d = ReviewStatus(s.String())
		} else if rs, ok := v.(ReviewStatus); ok {
			*d = rs
		}
	case *MatchConfidence:
		switch val := v.(type) {
		case MatchConfidence:
			*d = val
		case string:
			*d = MatchConfidence(val)
		case fmt.Stringer:
			*d = MatchConfidence(val.String())
		}
	case **uuid.UUID:
		if v == nil {
			*d = nil
			return nil
		}
		switch val := v.(type) {
		case *uuid.UUID:
			*d = val
		case uuid.UUID:
			*d = &val
		}
	case **string:
		if v == nil {
			*d = nil
			return nil
		}
		if val, ok := v.(*string); ok {
			*d = val
		} else if val, ok := v.(string); ok {
			*d = &val
		}
	case **time.Time:
		if v == nil {
			*d = nil
			return nil
		}
		if val, ok := v.(*time.Time); ok {
			*d = val
		} else if val, ok := v.(time.Time); ok {
			*d = &val
		}
	case **bool:
		if v == nil {
			*d = nil
			return nil
		}
		if val, ok := v.(*bool); ok {
			*d = val
		} else if val, ok := v.(bool); ok {
			*d = &val
		}
	case **MatchConfidence:
		if v == nil {
			*d = nil
			return nil
		}
		if val, ok := v.(*MatchConfidence); ok {
			*d = val
		} else if val, ok := v.(MatchConfidence); ok {
			*d = &val
		}
	case *bool:
		if val, ok := v.(bool); ok {
			*d = val
		}
	case *[]byte:
		if val, ok := v.([]byte); ok {
			*d = val
		}
	case *ConflictPair:
		switch val := v.(type) {
		case ConflictPair:
			*d = val
		case []byte:
			if err := json.Unmarshal(val, d); err != nil {
				return fmt.Errorf("assignValue: unmarshal ConflictPair from []byte: %w", err)
			}
		}
	default:
		return fmt.Errorf("assignValue: unhandled type %T", dest)
	}
	return nil
}