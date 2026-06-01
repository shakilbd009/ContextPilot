package upcoming

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ─── mockPool ─────────────────────────────────────────────────────────────────

// mockPool implements the Pool interface for repository unit testing.
type mockPool struct {
	beginFn    func(ctx context.Context) (pgx.Tx, error)
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	// rowsValues is a default set of rows to return from Query when queryFn is nil
	rowsValues [][]any
}

func (p *mockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	if p.beginFn != nil {
		return p.beginFn(ctx)
	}
	return &mockTx{}, nil
}

func (p *mockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if p.queryFn != nil {
		rows, err := p.queryFn(ctx, sql, args...)
		// When queryFn returns a mockRows, we need to ensure each caller gets
		// a fresh instance to avoid state pollution between iterations.
		if mr, ok := rows.(*mockRows); ok {
			// Copy values into a fresh mockRows so each call starts at pos=0
			copied := make([][]any, len(mr.values))
			copy(copied, mr.values)
			var errFn func() error
			if mr.errFn != nil {
				errFn = mr.errFn
			}
			return &mockRows{values: copied, errFn: errFn}, err
		}
		return rows, err
	}
	if p.rowsValues != nil {
		return &mockRows{values: p.rowsValues}, nil
	}
	return &mockRows{}, nil
}

func (p *mockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if p.queryRowFn != nil {
		return p.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{}
}

func (p *mockPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if p.execFn != nil {
		return p.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

// ─── mockTx ───────────────────────────────────────────────────────────────────

// mockTx implements pgx.Tx for repository unit testing.
type mockTx struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	commitFn   func(ctx context.Context) error
	rollbackFn func(ctx context.Context) error
}

func (tx *mockTx) Begin(ctx context.Context) (pgx.Tx, error)                          { return tx, nil }
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
func (tx *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx.execFn != nil {
		return tx.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (tx *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)  { return &mockRows{}, nil }
func (tx *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx.queryRowFn != nil {
		return tx.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{}
}
func (tx *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) { return 0, nil }
func (tx *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults         { return nil }
func (tx *mockTx) LargeObjects() pgx.LargeObjects                                        { return pgx.LargeObjects{} }
func (tx *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) { return nil, nil }
func (tx *mockTx) Conn() *pgx.Conn                                                       { return nil }

// ─── mockRows ─────────────────────────────────────────────────────────────────

// mockRows implements pgx.Rows for repository unit testing.
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
		return errorsNew("mockRows.Scan: no row at position")
	}
	vals := r.values[r.pos-1]
	if len(vals) != len(dest) {
		return errorsNew("mockRows.Scan: value count mismatch")
	}
	for i, v := range vals {
		if err := assignValue(dest[i], v); err != nil {
			return err
		}
	}
	return nil
}

func (r *mockRows) Close()                    { if r.closeFn != nil { r.closeFn() } }
func (r *mockRows) Err() error               { if r.errFn == nil { return nil }; return r.errFn() }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Values() ([]any, error)                         { return nil, nil }
func (r *mockRows) Conn() *pgx.Conn                                 { return nil }
func (r *mockRows) CommandTag() pgconn.CommandTag                   { return pgconn.CommandTag{} }
func (r *mockRows) RawValues() [][]byte                             { return nil }

// ─── mockRow ──────────────────────────────────────────────────────────────────

// mockRow implements pgx.Row for repository unit testing.
type mockRow struct {
	values []any
	scanFn func(dest ...any) error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	if r.values == nil {
		return errorsNew("mockRow.Scan: no values provided")
	}
	if len(r.values) != len(dest) {
		return errorsNew("mockRow.Scan: value count mismatch")
	}
	for i, v := range r.values {
		if err := assignValue(dest[i], v); err != nil {
			return err
		}
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func errorsNew(msg string) error { return &mockErr{msg} }

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

// assignValue copies v into the address pointed to by dest, for the specific
// types used in repository tests (uuid.UUID, string, int, time.Time, Status, and pointers).
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
	case *int64:
		switch val := v.(type) {
		case int:
			*d = int64(val)
		case int64:
			*d = val
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
	case *Status:
		if s, ok := v.(string); ok {
			*d = Status(s)
		} else if se, ok := v.(fmt.Stringer); ok {
			*d = Status(se.String())
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
	case *bool:
		if val, ok := v.(bool); ok {
			*d = val
		}
	default:
		return errorsNew("assignValue: unhandled type")
	}
	return nil
}
