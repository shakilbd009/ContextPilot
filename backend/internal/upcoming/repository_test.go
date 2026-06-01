package upcoming

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// repoWithPool builds a Repository backed by a mockPool.
func repoWithPool(pool *mockPool) *Repository {
	return &Repository{Pool: pool}
}

// ─── CreateMeeting ───────────────────────────────────────────────────────────

func TestRepository_CreateMeeting_HappyPath(t *testing.T) {
	owner := uuid.New()
	meetingID := uuid.New()
	scheduledStart := time.Now().Add(1 * time.Hour)
	desc := "Design review"
	org := "Acme Corp"
	participantEmail := "alice@example.com"
	participantName := "Alice"

	var committed bool
	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return &mockRow{values: []any{meetingID}}
				},
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("INSERT 1"), nil
				},
				commitFn: func(ctx context.Context) error {
					committed = true
					return nil
				},
			}, nil
		},
	}

	repo := repoWithPool(pool)
	id, err := repo.CreateMeeting(context.Background(), CreateMeetingInput{
		Title:                "Sprint Planning",
		ScheduledStart:       scheduledStart,
		Description:         &desc,
		ClientOrOrganization: &org,
		CreatedBy:            owner,
		Participants: []CreateParticipantInput{
			{DisplayName: &participantName, Email: &participantEmail},
		},
	})

	if err != nil {
		t.Fatalf("CreateMeeting() error = %v, want nil", err)
	}
	if id != meetingID {
		t.Errorf("CreateMeeting() id = %v, want %v", id, meetingID)
	}
	if !committed {
		t.Error("CreateMeeting() transaction was not committed")
	}
}

func TestRepository_CreateMeeting_ParticipantInsertFails(t *testing.T) {
	owner := uuid.New()
	meetingID := uuid.New()
	scheduledStart := time.Now().Add(1 * time.Hour)
	wantErr := errors.New("participant insert error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return &mockRow{values: []any{meetingID}}
				},
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, wantErr
				},
				rollbackFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.CreateMeeting(context.Background(), CreateMeetingInput{
		Title:          "Sprint Planning",
		ScheduledStart: scheduledStart,
		CreatedBy:      owner,
		Participants: []CreateParticipantInput{
			{DisplayName: ptr("Alice"), Email: ptr("alice@example.com")},
		},
	})

	if err == nil {
		t.Fatal("CreateMeeting() error = nil, want error")
	}
}

func TestRepository_CreateMeeting_CommitFails(t *testing.T) {
	owner := uuid.New()
	meetingID := uuid.New()
	scheduledStart := time.Now().Add(1 * time.Hour)
	wantErr := errors.New("commit error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
					return &mockRow{values: []any{meetingID}}
				},
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("INSERT 1"), nil
				},
				commitFn: func(ctx context.Context) error { return wantErr },
				rollbackFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.CreateMeeting(context.Background(), CreateMeetingInput{
		Title:          "Sprint Planning",
		ScheduledStart: scheduledStart,
		CreatedBy:      owner,
	})

	if err == nil {
		t.Fatal("CreateMeeting() error = nil, want error")
	}
}

func TestRepository_CreateMeeting_BeginFails(t *testing.T) {
	wantErr := errors.New("begin error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.CreateMeeting(context.Background(), CreateMeetingInput{
		Title:          "Sprint Planning",
		ScheduledStart: time.Now().Add(1 * time.Hour),
		CreatedBy:      uuid.New(),
	})

	if err == nil {
		t.Fatal("CreateMeeting() error = nil, want error")
	}
}

// ─── GetMeeting ──────────────────────────────────────────────────────────────

func TestRepository_GetMeeting_HappyPath(t *testing.T) {
	owner := uuid.New()
	meetingID := uuid.New()
	now := time.Now()
	desc := "Design review"
	org := "Acme"
	scheduledStart := now.Add(1 * time.Hour)
	participantID := uuid.New()
	participantName := "Alice"
	participantEmail := "alice@example.com"
	participantOrg := "Acme Corp"

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{values: []any{
				meetingID, "Sprint Planning", scheduledStart, &desc, &org,
				"scheduled", owner, now, now,
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: [][]any{
				{participantID, &participantName, &participantEmail, &participantOrg, now, now},
			}}, nil
		},
	}

	repo := repoWithPool(pool)
	m, err := repo.GetMeeting(context.Background(), meetingID)

	if err != nil {
		t.Fatalf("GetMeeting() error = %v, want nil", err)
	}
	if m.ID != meetingID {
		t.Errorf("GetMeeting().ID = %v, want %v", m.ID, meetingID)
	}
	if m.Title != "Sprint Planning" {
		t.Errorf("GetMeeting().Title = %q, want %q", m.Title, "Sprint Planning")
	}
	if m.Status != StatusScheduled {
		t.Errorf("GetMeeting().Status = %v, want %v", m.Status, StatusScheduled)
	}
	if len(m.Participants) != 1 {
		t.Fatalf("GetMeeting().Participants = %d, want 1", len(m.Participants))
	}
	if *m.Participants[0].DisplayName != participantName {
		t.Errorf("GetMeeting().Participants[0].DisplayName = %v, want %v", *m.Participants[0].DisplayName, participantName)
	}
}

func TestRepository_GetMeeting_NotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.GetMeeting(context.Background(), uuid.New())

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetMeeting() error = %v, want ErrNotFound", err)
	}
}

func TestRepository_GetMeeting_ParticipantQueryFails(t *testing.T) {
	wantErr := errors.New("participant query error")

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			now := time.Now()
			return &mockRow{values: []any{
				uuid.New(), "Sprint Planning", now.Add(1 * time.Hour), nil, nil,
				"scheduled", uuid.New(), now, now,
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.GetMeeting(context.Background(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("GetMeeting() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_GetMeeting_PoolQueryRowFails(t *testing.T) {
	wantErr := errors.New("query error")

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return wantErr }}
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.GetMeeting(context.Background(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("GetMeeting() error = %v, want %v", err, wantErr)
	}
}

// ─── ListMeetings ────────────────────────────────────────────────────────────

func TestRepository_ListMeetings_Scheduled(t *testing.T) {
	owner := uuid.New()
	id1, id2 := uuid.New(), uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if len(sql) > 180 {
				// Main meetings query — 9 columns: id, title, scheduled_start, description, client_or_organization, status, created_by, created_at, updated_at
				return &mockRows{values: [][]any{
					{id1, "Meeting A", now.Add(1 * time.Hour), nil, nil, "scheduled", owner, now, now},
					{id2, "Meeting B", now.Add(2 * time.Hour), nil, nil, "scheduled", owner, now, now},
				}}, nil
			}
			// getParticipants query — 6 columns: id, display_name, email, organization, created_at, updated_at
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	meetings, err := repo.ListMeetings(context.Background(), owner, "scheduled")

	if err != nil {
		t.Fatalf("ListMeetings() error = %v, want nil", err)
	}
	if len(meetings) != 2 {
		t.Fatalf("ListMeetings() returned %d meetings, want 2", len(meetings))
	}
	// Verify sorted by scheduled_start ASC
	if meetings[0].Title != "Meeting A" || meetings[1].Title != "Meeting B" {
		t.Errorf("ListMeetings() order wrong: [%v, %v]", meetings[0].Title, meetings[1].Title)
	}
}

func TestRepository_ListMeetings_AllStatus(t *testing.T) {
	owner := uuid.New()
	id1, id2 := uuid.New(), uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if len(sql) > 180 {
				// Main meetings query — 9 columns
				return &mockRows{values: [][]any{
					{id1, "Scheduled Meeting", now.Add(1 * time.Hour), nil, nil, "scheduled", owner, now, now},
					{id2, "Cancelled Meeting", now.Add(2 * time.Hour), nil, nil, "cancelled", owner, now, now},
				}}, nil
			}
			// getParticipants query — 6 columns
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	meetings, err := repo.ListMeetings(context.Background(), owner, "all")

	if err != nil {
		t.Fatalf("ListMeetings() error = %v, want nil", err)
	}
	if len(meetings) != 2 {
		t.Fatalf("ListMeetings() returned %d meetings, want 2", len(meetings))
	}
}

func TestRepository_ListMeetings_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	meetings, err := repo.ListMeetings(context.Background(), uuid.New(), "scheduled")

	if err != nil {
		t.Fatalf("ListMeetings() error = %v, want nil", err)
	}
	if len(meetings) != 0 {
		t.Errorf("ListMeetings() returned %d meetings, want 0", len(meetings))
	}
}

func TestRepository_ListMeetings_QueryFails(t *testing.T) {
	wantErr := errors.New("query error")

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.ListMeetings(context.Background(), uuid.New(), "scheduled")

	if !errors.Is(err, wantErr) {
		t.Errorf("ListMeetings() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_ListMeetings_RowsErr(t *testing.T) {
	wantErr := errors.New("rows error")
	now := time.Now()
	meetingID := uuid.New()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if len(sql) > 180 {
				// Main meetings query — 9 columns
				return &mockRows{
					values: [][]any{{meetingID, "Meeting", now, nil, nil, "scheduled", uuid.New(), now, now}},
					errFn:  func() error { return wantErr },
				}, nil
			}
			// getParticipants query — 6 columns
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.ListMeetings(context.Background(), uuid.New(), "scheduled")

	if !errors.Is(err, wantErr) {
		t.Errorf("ListMeetings() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_ListMeetings_Cancelled(t *testing.T) {
	owner := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if len(sql) > 180 {
				// Main meetings query — 9 columns
				return &mockRows{values: [][]any{
					{uuid.New(), "Cancelled Meeting", now.Add(1 * time.Hour), nil, nil, "cancelled", owner, now, now},
				}}, nil
			}
			// getParticipants query — 6 columns
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	meetings, err := repo.ListMeetings(context.Background(), owner, "cancelled")

	if err != nil {
		t.Fatalf("ListMeetings() error = %v, want nil", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("ListMeetings() returned %d meetings, want 1", len(meetings))
	}
	if meetings[0].Status != StatusCancelled {
		t.Errorf("ListMeetings()[0].Status = %v, want %v", meetings[0].Status, StatusCancelled)
	}
}

// ─── GetMeetingOwned ────────────────────────────────────────────────────────

func TestRepository_GetMeetingOwned_HappyPath(t *testing.T) {
	owner := uuid.New()
	meetingID := uuid.New()
	now := time.Now()
	scheduledStart := now.Add(1 * time.Hour)

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{values: []any{
				meetingID, "Owned Meeting", scheduledStart, nil, nil,
				"scheduled", owner, now, now,
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	m, err := repo.GetMeetingOwned(context.Background(), meetingID, owner)

	if err != nil {
		t.Fatalf("GetMeetingOwned() error = %v, want nil", err)
	}
	if m.ID != meetingID {
		t.Errorf("GetMeetingOwned().ID = %v, want %v", m.ID, meetingID)
	}
	if m.CreatedBy != owner {
		t.Errorf("GetMeetingOwned().CreatedBy = %v, want %v", m.CreatedBy, owner)
	}
}

func TestRepository_GetMeetingOwned_NotFound_WrongOwner(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{}, nil
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.GetMeetingOwned(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetMeetingOwned() error = %v, want ErrNotFound", err)
	}
}

func TestRepository_GetMeetingOwned_ParticipantQueryFails(t *testing.T) {
	wantErr := errors.New("participant error")

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			now := time.Now()
			return &mockRow{values: []any{
				uuid.New(), "Meeting", now.Add(1 * time.Hour), nil, nil,
				"scheduled", uuid.New(), now, now,
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.GetMeetingOwned(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("GetMeetingOwned() error = %v, want %v", err, wantErr)
	}
}

// ─── UpdateMeeting ──────────────────────────────────────────────────────────

func TestRepository_UpdateMeeting_AllFields(t *testing.T) {
	meetingID := uuid.New()
	newTitle := "Updated Title"
	newStart := time.Now().Add(2 * time.Hour)
	now := time.Now()

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("UPDATE 1"), nil
				},
				queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
					return &mockRows{values: [][]any{
						{uuid.New(), ptr("Alice"), ptr("alice@example.com"), nil, now, now},
					}}, nil
				},
				commitFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), meetingID, UpdateMeetingInput{
		Title:          &newTitle,
		ScheduledStart: &newStart,
		Participants: []CreateParticipantInput{
			{DisplayName: ptr("Alice"), Email: ptr("alice@example.com")},
		},
	})

	if err != nil {
		t.Fatalf("UpdateMeeting() error = %v, want nil", err)
	}
}

func TestRepository_UpdateMeeting_OnlyParticipants(t *testing.T) {
	meetingID := uuid.New()
	now := time.Now()
	participantDeleteCalled := false

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					participantDeleteCalled = true
					return pgconn.NewCommandTag("UPDATE 1"), nil
				},
				queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
					return &mockRows{values: [][]any{
						{uuid.New(), ptr("Bob"), ptr("bob@example.com"), nil, now, now},
					}}, nil
				},
				commitFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), meetingID, UpdateMeetingInput{
		Participants: []CreateParticipantInput{
			{DisplayName: ptr("Bob"), Email: ptr("bob@example.com")},
		},
	})

	if err != nil {
		t.Fatalf("UpdateMeeting() error = %v, want nil", err)
	}
	if !participantDeleteCalled {
		t.Error("UpdateMeeting() did not call exec for participants")
	}
}

func TestRepository_UpdateMeeting_UpdateFails(t *testing.T) {
	meetingID := uuid.New()
	wantErr := errors.New("update error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, wantErr
				},
				rollbackFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), meetingID, UpdateMeetingInput{
		Title: ptr("Updated"),
	})

	if !errors.Is(err, wantErr) {
		t.Errorf("UpdateMeeting() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_UpdateMeeting_CommitFails(t *testing.T) {
	meetingID := uuid.New()
	wantErr := errors.New("commit error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("UPDATE 1"), nil
				},
				commitFn:  func(ctx context.Context) error { return wantErr },
				rollbackFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), meetingID, UpdateMeetingInput{
		Title: ptr("Updated"),
	})

	if !errors.Is(err, wantErr) {
		t.Errorf("UpdateMeeting() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_UpdateMeeting_BeginFails(t *testing.T) {
	wantErr := errors.New("begin error")

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), uuid.New(), UpdateMeetingInput{
		Title: ptr("Updated"),
	})

	if !errors.Is(err, wantErr) {
		t.Errorf("UpdateMeeting() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_UpdateMeeting_ParticipantReinsertFails(t *testing.T) {
	meetingID := uuid.New()
	wantErr := errors.New("re-insert error")
	insertCount := 0

	pool := &mockPool{
		beginFn: func(ctx context.Context) (pgx.Tx, error) {
			return &mockTx{
				execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					if insertCount == 0 {
						insertCount++
						return pgconn.NewCommandTag("DELETE 1"), nil
					}
					return pgconn.CommandTag{}, wantErr
				},
				rollbackFn: func(ctx context.Context) error { return nil },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.UpdateMeeting(context.Background(), meetingID, UpdateMeetingInput{
		Participants: []CreateParticipantInput{
			{DisplayName: ptr("Alice")},
			{DisplayName: ptr("Bob")},
		},
	})

	if !errors.Is(err, wantErr) {
		t.Errorf("UpdateMeeting() error = %v, want %v", err, wantErr)
	}
}

// ─── CancelMeeting ───────────────────────────────────────────────────────────

func TestRepository_CancelMeeting_HappyPath(t *testing.T) {
	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.CancelMeeting(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("CancelMeeting() error = %v, want nil", err)
	}
}

func TestRepository_CancelMeeting_AlreadyCancelledOrNotFound(t *testing.T) {
	// CancelMeeting uses WHERE status='scheduled' so a non-existent or
	// already-cancelled meeting returns rowsAffected=0 but no error.
	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}

	repo := repoWithPool(pool)
	err := repo.CancelMeeting(context.Background(), uuid.New())

	// Repository does not treat 0 rows as an error
	if err != nil {
		t.Errorf("CancelMeeting() error = %v, want nil", err)
	}
}

func TestRepository_CancelMeeting_ExecError(t *testing.T) {
	wantErr := errors.New("exec error")

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, wantErr
		},
	}

	repo := repoWithPool(pool)
	err := repo.CancelMeeting(context.Background(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("CancelMeeting() error = %v, want %v", err, wantErr)
	}
}

// ─── getParticipants (internal) ──────────────────────────────────────────────

func TestRepository_getParticipants_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: [][]any{}}, nil
		},
	}

	repo := repoWithPool(pool)
	participants, err := repo.getParticipants(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("getParticipants() error = %v, want nil", err)
	}
	if len(participants) != 0 {
		t.Errorf("getParticipants() returned %d, want 0", len(participants))
	}
}

func TestRepository_getParticipants_QueryError(t *testing.T) {
	wantErr := errors.New("query error")

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, wantErr
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.getParticipants(context.Background(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("getParticipants() error = %v, want %v", err, wantErr)
	}
}

func TestRepository_getParticipants_RowsErr(t *testing.T) {
	wantErr := errors.New("rows error")
	now := time.Now()
	pid := uuid.New()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{
				values: [][]any{{pid, ptr("Alice"), ptr("alice@example.com"), nil, now, now}},
				errFn:  func() error { return wantErr },
			}, nil
		},
	}

	repo := repoWithPool(pool)
	_, err := repo.getParticipants(context.Background(), uuid.New())

	if !errors.Is(err, wantErr) {
		t.Errorf("getParticipants() error = %v, want %v", err, wantErr)
	}
}
