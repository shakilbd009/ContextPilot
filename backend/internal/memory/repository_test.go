package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ─── Queue ───────────────────────────────────────────────────────────────────

func TestRepository_QueueJob(t *testing.T) {
	meetingID := uuid.New()
	triggerType := TriggerTypeImport
	previousVersionID := uuid.New()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			// Verify correct SQL was issued
			if args[0] != meetingID || args[1] != triggerType {
				t.Errorf("QueueJob args mismatch: got %v, %v", args[0], args[1])
			}
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = uuid.New() // return a job ID
				return nil
			}}
		},
	}

	repo := &Repository{pool: pool}
	jobID, correlationID, err := repo.QueueJob(context.Background(), meetingID, triggerType, &previousVersionID)
	if err != nil {
		t.Errorf("QueueJob unexpected error: %v", err)
	}
	if jobID == uuid.Nil {
		t.Error("QueueJob returned nil jobID")
	}
	if correlationID == uuid.Nil {
		t.Error("QueueJob returned nil correlationID")
	}
}

func TestRepository_QueueJob_Error(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return errors.New("db error")
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, _, err := repo.QueueJob(context.Background(), uuid.New(), TriggerTypeImport, nil)
	if err == nil {
		t.Error("QueueJob expected error, got nil")
	}
}

func TestRepository_GetQueuedJobs(t *testing.T) {
	job1 := uuid.New()
	job2 := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{
				values: [][]any{
					{
						job1, uuid.New(), TriggerTypeImport, JobStatusQueued,
						0, 3, nil, uuid.New(), nil,
						now, nil, nil, nil,
					},
					{
						job2, uuid.New(), TriggerTypeReprocess, JobStatusRetrying,
						1, 3, strPtr("retrying"), uuid.New(), strPtr("retrying"),
						now, nil, nil, nil,
					},
				},
			}, nil
		},
	}

	repo := &Repository{pool: pool}
	jobs, err := repo.GetQueuedJobs(context.Background(), 10)
	if err != nil {
		t.Errorf("GetQueuedJobs unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("GetQueuedJobs returned %d jobs, want 2", len(jobs))
	}
	if jobs[0].ID != job1 {
		t.Errorf("jobs[0].ID = %v, want %v", jobs[0].ID, job1)
	}
	if jobs[1].Status != JobStatusRetrying {
		t.Errorf("jobs[1].Status = %v, want %v", jobs[1].Status, JobStatusRetrying)
	}
}

func TestRepository_GetQueuedJobs_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: nil}, nil
		},
	}
	repo := &Repository{pool: pool}
	jobs, err := repo.GetQueuedJobs(context.Background(), 10)
	if err != nil {
		t.Errorf("GetQueuedJobs unexpected error: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("GetQueuedJobs returned %d jobs, want 0", len(jobs))
	}
}

func TestRepository_GetQueuedJobs_Error(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, errors.New("query error")
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetQueuedJobs(context.Background(), 10)
	if err == nil {
		t.Error("GetQueuedJobs expected error, got nil")
	}
}

func TestRepository_PickJob(t *testing.T) {
	jobID := uuid.New()
	meetingID := uuid.New()
	correlationID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{
				values: []any{
					jobID, meetingID, TriggerTypeImport, JobStatusProcessing,
					0, 3, nil, nil, correlationID,
					now, now, nil, nil,
				},
			}
		},
	}

	repo := &Repository{pool: pool}
	job, err := repo.PickJob(context.Background(), jobID)
	if err != nil {
		t.Errorf("PickJob unexpected error: %v", err)
	}
	if job == nil {
		t.Fatal("PickJob returned nil job")
	}
	if job.ID != jobID {
		t.Errorf("job.ID = %v, want %v", job.ID, jobID)
	}
	if job.Status != JobStatusProcessing {
		t.Errorf("job.Status = %v, want %v", job.Status, JobStatusProcessing)
	}
	if job.MeetingID != meetingID {
		t.Errorf("job.MeetingID = %v, want %v", job.MeetingID, meetingID)
	}
}

func TestRepository_PickJob_NotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}
	repo := &Repository{pool: pool}
	job, err := repo.PickJob(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("PickJob unexpected error: %v", err)
	}
	if job != nil {
		t.Errorf("PickJob returned job, want nil for unclaimed job")
	}
}

func TestRepository_PickJob_Error(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return errors.New("db error")
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.PickJob(context.Background(), uuid.New())
	if err == nil {
		t.Error("PickJob expected error, got nil")
	}
}

func TestRepository_UpdateJobCompleted(t *testing.T) {
	jobID := uuid.New()

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			if args[0] != jobID {
				t.Errorf("UpdateJobCompleted args[0] = %v, want %v", args[0], jobID)
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobCompleted(context.Background(), jobID, uuid.New(), nil)
	if err != nil {
		t.Errorf("UpdateJobCompleted unexpected error: %v", err)
	}
}

func TestRepository_UpdateJobCompleted_WithFailure(t *testing.T) {
	jobID := uuid.New()
	reason := "test failure"

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobCompleted(context.Background(), jobID, uuid.New(), &reason)
	if err != nil {
		t.Errorf("UpdateJobCompleted unexpected error: %v", err)
	}
}

func TestRepository_UpdateJobCompleted_Error(t *testing.T) {
	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("exec error")
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobCompleted(context.Background(), uuid.New(), uuid.New(), nil)
	if err == nil {
		t.Error("UpdateJobCompleted expected error, got nil")
	}
}

func TestRepository_UpdateJobRetrying(t *testing.T) {
	jobID := uuid.New()
	retryCount := 2
	nextRetry := time.Now().Add(10 * time.Second)
	failureReason := "transient error"

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobRetrying(context.Background(), jobID, retryCount, nextRetry, failureReason)
	if err != nil {
		t.Errorf("UpdateJobRetrying unexpected error: %v", err)
	}
}

func TestRepository_UpdateJobRetryExhausted(t *testing.T) {
	jobID := uuid.New()

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobRetryExhausted(context.Background(), jobID, "max retries exceeded")
	if err != nil {
		t.Errorf("UpdateJobRetryExhausted unexpected error: %v", err)
	}
}

func TestRepository_UpdateJobFailed(t *testing.T) {
	jobID := uuid.New()

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.UpdateJobFailed(context.Background(), jobID, "permanent error")
	if err != nil {
		t.Errorf("UpdateJobFailed unexpected error: %v", err)
	}
}

func TestRepository_GetLatestJobForMeeting(t *testing.T) {
	jobID := uuid.New()
	meetingID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{values: []any{
				jobID, meetingID, TriggerTypeImport, JobStatusCompleted,
				0, 3, nil, nil, uuid.New(), now, nil, nil, nil,
			}}
		},
	}
	repo := &Repository{pool: pool}
	job, err := repo.GetLatestJobForMeeting(context.Background(), meetingID)
	if err != nil {
		t.Errorf("GetLatestJobForMeeting unexpected error: %v", err)
	}
	if job == nil {
		t.Fatal("GetLatestJobForMeeting returned nil")
	}
	if job.ID != jobID {
		t.Errorf("job.ID = %v, want %v", job.ID, jobID)
	}
}

func TestRepository_GetLatestJobForMeeting_NoRows(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}
	repo := &Repository{pool: pool}
	job, err := repo.GetLatestJobForMeeting(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("GetLatestJobForMeeting unexpected error: %v", err)
	}
	if job != nil {
		t.Errorf("GetLatestJobForMeeting returned job, want nil")
	}
}

// ─── Versions ────────────────────────────────────────────────────────────────

func TestRepository_GetActiveVersion(t *testing.T) {
	meetingID := uuid.New()
	versionID := uuid.New()
	now := time.Now()
	contentJSON := []byte(`{"summary":{"statement":"test"}}`)

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = versionID
				*dest[1].(*uuid.UUID) = meetingID
				*dest[2].(**uuid.UUID) = nil
				*dest[3].(*int) = 1
				*dest[4].(*VersionStatus) = VersionStatusActive
				*dest[5].(*bool) = true
				*dest[6].(*[]byte) = contentJSON
				*dest[7].(*TriggerType) = TriggerTypeImport
				*dest[8].(*time.Time) = now
				*dest[9].(**uuid.UUID) = nil
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	version, err := repo.GetActiveVersion(context.Background(), meetingID)
	if err != nil {
		t.Errorf("GetActiveVersion unexpected error: %v", err)
	}
	if version == nil {
		t.Fatal("GetActiveVersion returned nil")
	}
	if version.ID != versionID {
		t.Errorf("version.ID = %v, want %v", version.ID, versionID)
	}
	if version.VersionNumber != 1 {
		t.Errorf("version.VersionNumber = %d, want 1", version.VersionNumber)
	}
	if !version.IsActive {
		t.Error("version.IsActive = false, want true")
	}
}

func TestRepository_GetActiveVersion_NoRows(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return pgx.ErrNoRows
			}}
		},
	}
	repo := &Repository{pool: pool}
	version, err := repo.GetActiveVersion(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("GetActiveVersion unexpected error: %v", err)
	}
	if version != nil {
		t.Errorf("GetActiveVersion returned version, want nil")
	}
}

func TestRepository_GetActiveVersion_Error(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return errors.New("db error")
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetActiveVersion(context.Background(), uuid.New())
	if err == nil {
		t.Error("GetActiveVersion expected error, got nil")
	}
}

func TestRepository_GetVersionByNumber(t *testing.T) {
	meetingID := uuid.New()
	versionID := uuid.New()
	now := time.Now()
	contentJSON := []byte(`{"summary":{"statement":"test"}}`)

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			if args[1].(int) != 2 {
				t.Errorf("GetVersionByNumber args[1] = %v, want 2", args[1])
			}
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = versionID
				*dest[1].(*uuid.UUID) = meetingID
				*dest[2].(**uuid.UUID) = nil
				*dest[3].(*int) = 2
				*dest[4].(*VersionStatus) = VersionStatusActive
				*dest[5].(*bool) = true
				*dest[6].(*[]byte) = contentJSON
				*dest[7].(*TriggerType) = TriggerTypeReprocess
				*dest[8].(*time.Time) = now
				*dest[9].(**uuid.UUID) = nil
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	version, err := repo.GetVersionByNumber(context.Background(), meetingID, 2)
	if err != nil {
		t.Errorf("GetVersionByNumber unexpected error: %v", err)
	}
	if version == nil {
		t.Fatal("GetVersionByNumber returned nil")
	}
	if version.VersionNumber != 2 {
		t.Errorf("version.VersionNumber = %d, want 2", version.VersionNumber)
	}
}

func TestRepository_GetVersionByNumber_NotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return ErrVersionNotFound
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetVersionByNumber(context.Background(), uuid.New(), 99)
	if !errors.Is(err, ErrVersionNotFound) {
		t.Errorf("GetVersionByNumber error = %v, want ErrVersionNotFound", err)
	}
}

func TestRepository_ListVersions(t *testing.T) {
	meetingID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{
				values: [][]any{
					{1, true, VersionStatusActive, now, nil},
					{2, false, VersionStatusSuperseded, now, nil},
				},
			}, nil
		},
	}
	repo := &Repository{pool: pool}
	versions, err := repo.ListVersions(context.Background(), meetingID)
	if err != nil {
		t.Errorf("ListVersions unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("ListVersions returned %d versions, want 2", len(versions))
	}
	if versions[0].VersionNumber != 1 || !versions[0].IsActive {
		t.Errorf("versions[0] = %+v, want VersionNumber=1, IsActive=true", versions[0])
	}
	if versions[1].VersionNumber != 2 || versions[1].IsActive {
		t.Errorf("versions[1] = %+v, want VersionNumber=2, IsActive=false", versions[1])
	}
}

func TestRepository_ListVersions_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: nil}, nil
		},
	}
	repo := &Repository{pool: pool}
	versions, err := repo.ListVersions(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("ListVersions unexpected error: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("ListVersions returned %d versions, want 0", len(versions))
	}
}

// ─── Stale detection ─────────────────────────────────────────────────────────

func TestRepository_IsMemoryStale_NotProcessed(t *testing.T) {
	meetingID := uuid.New()
	now := time.Now()

	// hasActive=false → not stale
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*time.Time) = now
				*dest[1].(*time.Time) = now.Add(-1 * time.Hour)
				*dest[2].(*bool) = false // no active version
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	stale, err := repo.IsMemoryStale(context.Background(), meetingID)
	if err != nil {
		t.Errorf("IsMemoryStale unexpected error: %v", err)
	}
	if stale {
		t.Error("IsMemoryStale = true, want false when no active version")
	}
}

func TestRepository_IsMemoryStale_Fresh(t *testing.T) {
	meetingID := uuid.New()
	versionCreated := time.Now().Add(-1 * time.Hour)
	meetingUpdated := time.Now().Add(-30 * time.Minute)

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
// Scan dest order: updatedAt, versionCreatedAt, hasActive
	*dest[0].(*time.Time) = versionCreated // version.created_at (older)
	*dest[1].(*time.Time) = meetingUpdated // meeting.updated_at (newer)
				*dest[2].(*bool) = true
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	stale, err := repo.IsMemoryStale(context.Background(), meetingID)
	if err != nil {
		t.Errorf("IsMemoryStale unexpected error: %v", err)
	}
	if stale {
		t.Error("IsMemoryStale = true, want false (meeting updated before version created)")
	}
}

func TestRepository_IsMemoryStale_Stale(t *testing.T) {
	meetingID := uuid.New()
	versionCreated := time.Now().Add(-1 * time.Hour)
	meetingUpdated := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*time.Time) = meetingUpdated  // meeting.updated_at (newer)
				*dest[1].(*time.Time) = versionCreated // version.created_at (older)
				*dest[2].(*bool) = true
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	stale, err := repo.IsMemoryStale(context.Background(), meetingID)
	if err != nil {
		t.Errorf("IsMemoryStale unexpected error: %v", err)
	}
	if !stale {
		t.Error("IsMemoryStale = false, want true (meeting updated after version created)")
	}
}

func TestRepository_IsMemoryStale_Error(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return errors.New("db error")
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.IsMemoryStale(context.Background(), uuid.New())
	if err == nil {
		t.Error("IsMemoryStale expected error, got nil")
	}
}

// ─── Conflicts ───────────────────────────────────────────────────────────────

func TestRepository_CreateConflict(t *testing.T) {
	meetingID := uuid.New()
	memoryVersionID := uuid.New()
	conflictID := uuid.New()
	pair := ConflictPair{
		Current:  []ConflictItemContent{{Statement: "current statement", QualityStatus: QualityConflicting}},
		Prior:    []ConflictItemContent{{Statement: "prior statement", QualityStatus: QualityConflicting}},
	}

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = conflictID
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	id, err := repo.CreateConflict(context.Background(), meetingID, &memoryVersionID, pair)
	if err != nil {
		t.Errorf("CreateConflict unexpected error: %v", err)
	}
	if id != conflictID {
		t.Errorf("CreateConflict returned %v, want %v", id, conflictID)
	}
}

func TestRepository_ListPendingConflicts(t *testing.T) {
	meetingID := uuid.New()
	conflictID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{
				values: [][]any{
					{
						conflictID, meetingID, uuid.New(),
						ConflictPair{Current: []ConflictItemContent{{Statement: "a", QualityStatus: QualityConflicting}}, Prior: []ConflictItemContent{{Statement: "b", QualityStatus: QualityConflicting}}},
						QualityConflicting, ReviewStatusPending,
						nil, nil, nil, now,
					},
				},
			}, nil
		},
	}
	repo := &Repository{pool: pool}
	conflicts, err := repo.ListPendingConflicts(context.Background(), meetingID)
	if err != nil {
		t.Errorf("ListPendingConflicts unexpected error: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("ListPendingConflicts returned %d conflicts, want 1", len(conflicts))
	}
	if conflicts[0].ID != conflictID {
		t.Errorf("conflicts[0].ID = %v, want %v", conflicts[0].ID, conflictID)
	}
}

func TestRepository_ListPendingConflicts_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: nil}, nil
		},
	}
	repo := &Repository{pool: pool}
	conflicts, err := repo.ListPendingConflicts(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("ListPendingConflicts unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("ListPendingConflicts returned %d conflicts, want 0", len(conflicts))
	}
}

func TestRepository_ResolveConflict(t *testing.T) {
	conflictID := uuid.New()
	resolvedBy := uuid.New()
	note := "resolved by user"

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.ResolveConflict(context.Background(), conflictID, resolvedBy, note)
	if err != nil {
		t.Errorf("ResolveConflict unexpected error: %v", err)
	}
}

func TestRepository_ResolveConflict_NotFound(t *testing.T) {
	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.ResolveConflict(context.Background(), uuid.New(), uuid.New(), "note")
	if !errors.Is(err, ErrConflictNotFound) {
		t.Errorf("ResolveConflict error = %v, want ErrConflictNotFound", err)
	}
}

func TestRepository_GetConflict(t *testing.T) {
	conflictID := uuid.New()
	meetingID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = conflictID
				*dest[1].(*uuid.UUID) = meetingID
				*dest[2].(**uuid.UUID) = nil
				*dest[3].(*ConflictPair) = ConflictPair{
					Current:  []ConflictItemContent{{Statement: "current", QualityStatus: QualityConflicting}},
					Prior:    []ConflictItemContent{{Statement: "previous", QualityStatus: QualityConflicting}},
				}
				*dest[4].(*QualityStatus) = QualityConflicting
				*dest[5].(*ReviewStatus) = ReviewStatusPending
				*dest[6].(**string) = nil
				*dest[7].(**time.Time) = nil
				*dest[8].(**uuid.UUID) = nil
				*dest[9].(*time.Time) = now
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	conflict, err := repo.GetConflict(context.Background(), conflictID)
	if err != nil {
		t.Errorf("GetConflict unexpected error: %v", err)
	}
	if conflict == nil {
		t.Fatal("GetConflict returned nil")
	}
	if conflict.ID != conflictID {
		t.Errorf("conflict.ID = %v, want %v", conflict.ID, conflictID)
	}
}

func TestRepository_GetConflict_NotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return ErrConflictNotFound
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetConflict(context.Background(), uuid.New())
	if !errors.Is(err, ErrConflictNotFound) {
		t.Errorf("GetConflict error = %v, want ErrConflictNotFound", err)
	}
}

func TestRepository_MarkVersionConflictReview(t *testing.T) {
	versionID := uuid.New()

	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.MarkVersionConflictReview(context.Background(), versionID)
	if err != nil {
		t.Errorf("MarkVersionConflictReview unexpected error: %v", err)
	}
}

// ─── Prior Memory Inputs ─────────────────────────────────────────────────────

func TestRepository_RecordPriorMemoryInputs(t *testing.T) {
	memVersionID := uuid.New()
	priorVersionID := uuid.New()
	inputs := []PriorMemoryInput{
		{
			PriorMemoryVersionID: priorVersionID,
			MatchConfidence:      func() *MatchConfidence { mc := MatchSafe; return &mc }(),
			IncludedByUser:        false,
			ExcludedByUser:        false,
		},
	}

	called := false
	pool := &mockPool{
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			called = true
			if args[0].(uuid.UUID) != memVersionID {
				t.Errorf("args[0] = %v, want %v", args[0], memVersionID)
			}
			if args[1].(uuid.UUID) != priorVersionID {
				t.Errorf("args[1] = %v, want %v", args[1], priorVersionID)
			}
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	}
	repo := &Repository{pool: pool}
	err := repo.RecordPriorMemoryInputs(context.Background(), memVersionID, inputs)
	if err != nil {
		t.Errorf("RecordPriorMemoryInputs unexpected error: %v", err)
	}
	if !called {
		t.Error("RecordPriorMemoryInputs did not call pool.Exec")
	}
}

func TestRepository_ListPriorMemoryInputs(t *testing.T) {
	memVersionID := uuid.New()
	meetingID := uuid.New()
	now := time.Now()
	mc := MatchConfidence(MatchSafe)

	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if args[0].(uuid.UUID) != memVersionID {
				t.Errorf("ListPriorMemoryInputs args[0] = %v, want %v", args[0], memVersionID)
			}
			return &mockRows{
				values: [][]any{
					{meetingID, "Prior Meeting", now, mc},
				},
			}, nil
		},
	}
	repo := &Repository{pool: pool}
	inputs, err := repo.ListPriorMemoryInputs(context.Background(), memVersionID)
	if err != nil {
		t.Errorf("ListPriorMemoryInputs unexpected error: %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("ListPriorMemoryInputs returned %d inputs, want 1", len(inputs))
	}
	if inputs[0].MeetingID != meetingID {
		t.Errorf("inputs[0].MeetingID = %v, want %v", inputs[0].MeetingID, meetingID)
	}
	if inputs[0].Title != "Prior Meeting" {
		t.Errorf("inputs[0].Title = %q, want %q", inputs[0].Title, "Prior Meeting")
	}
}

func TestRepository_ListPriorMemoryInputs_Empty(t *testing.T) {
	pool := &mockPool{
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: nil}, nil
		},
	}
	repo := &Repository{pool: pool}
	inputs, err := repo.ListPriorMemoryInputs(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("ListPriorMemoryInputs unexpected error: %v", err)
	}
	if len(inputs) != 0 {
		t.Errorf("ListPriorMemoryInputs returned %d inputs, want 0", len(inputs))
	}
}

// ─── Meeting ownership ────────────────────────────────────────────────────────

func TestRepository_MeetingExistsAndOwned_True(t *testing.T) {
	meetingID := uuid.New()
	userID := uuid.New()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*bool) = true
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	exists, err := repo.MeetingExistsAndOwned(context.Background(), meetingID, userID)
	if err != nil {
		t.Errorf("MeetingExistsAndOwned unexpected error: %v", err)
	}
	if !exists {
		t.Error("MeetingExistsAndOwned = false, want true")
	}
}

func TestRepository_MeetingExistsAndOwned_False(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*dest[0].(*bool) = false
				return nil
			}}
		},
	}
	repo := &Repository{pool: pool}
	exists, err := repo.MeetingExistsAndOwned(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("MeetingExistsAndOwned unexpected error: %v", err)
	}
	if exists {
		t.Error("MeetingExistsAndOwned = true, want false")
	}
}

// ─── Meeting info ─────────────────────────────────────────────────────────────

func TestRepository_GetMeetingInfo(t *testing.T) {
	meetingID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{values: []any{
				meetingID, "Test Meeting", nil, nil, now, now,
			}}
		},
		// queryFn for participants
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{
				values: [][]any{
					{uuid.New(), "Alice", strPtr("alice@example.com"), strPtr("Org A")},
					{uuid.New(), "Bob", nil, nil},
				},
			}, nil
		},
	}
	repo := &Repository{pool: pool}
	info, err := repo.GetMeetingInfo(context.Background(), meetingID)
	if err != nil {
		t.Errorf("GetMeetingInfo unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("GetMeetingInfo returned nil")
	}
	if info.Title != "Test Meeting" {
		t.Errorf("info.Title = %q, want %q", info.Title, "Test Meeting")
	}
	if info.Participants == nil || len(info.Participants) != 2 {
		t.Errorf("info.Participants len = %d, want 2", len(info.Participants))
	}
	if info.Participants[0].DisplayName != "Alice" {
		t.Errorf("info.Participants[0].DisplayName = %q, want %q", info.Participants[0].DisplayName, "Alice")
	}
}

func TestRepository_GetMeetingInfo_NotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return ErrMeetingNotFound
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetMeetingInfo(context.Background(), uuid.New())
	if !errors.Is(err, ErrMeetingNotFound) {
		t.Errorf("GetMeetingInfo error = %v, want ErrMeetingNotFound", err)
	}
}

func TestRepository_GetMeetingInfo_NoParticipants(t *testing.T) {
	meetingID := uuid.New()
	now := time.Now()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{values: []any{
				meetingID, "Empty Meeting", nil, nil, now, now,
			}}
		},
		queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &mockRows{values: nil}, nil
		},
	}
	repo := &Repository{pool: pool}
	info, err := repo.GetMeetingInfo(context.Background(), meetingID)
	if err != nil {
		t.Errorf("GetMeetingInfo unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("GetMeetingInfo returned nil")
	}
	if len(info.Participants) != 0 {
		t.Errorf("info.Participants len = %d, want 0", len(info.Participants))
	}
}

// ─── Processing State ─────────────────────────────────────────────────────────

func TestRepository_GetProcessingState_NotProcessed(t *testing.T) {
	meetingID := uuid.New()

	// Call tracker so mock responds correctly to each SQL call:
	// 1) SELECT EXISTS(...) → true
	// 2) SELECT id FROM memory_versions WHERE meeting_id=$1 AND is_active=true → NULL
	// 3) IsMemoryStale → false
	// 4) GetLatestJobForMeeting → nil job
	call := 0
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			call++
			switch call {
			case 1:
				// EXISTS check → meeting exists
				return &mockRow{values: []any{true}}
			case 2:
				// GetActiveVersionID: no active version → nil pointer
				var nilID *uuid.UUID
				return &mockRow{values: []any{nilID}}
			case 3:
				// IsMemoryStale: updatedAt, versionCreatedAt, hasActive
				now := time.Now()
				return &mockRow{values: []any{now, now, false}} // hasActive=false → returns false
			case 4:
				// GetLatestJobForMeeting → no rows
				return &mockRow{scanFn: func(dest ...any) error {
					return pgx.ErrNoRows
				}}
			default:
				t.Fatalf("unexpected SQL call #%d: %s", call, sql)
				return nil
			}
		},
	}
	repo := &Repository{pool: pool}
	state, err := repo.GetProcessingState(context.Background(), meetingID)
	if err != nil {
		t.Errorf("GetProcessingState unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("GetProcessingState returned nil")
	}
	if state.ProcessingState != string(MemoryStateNotProcessed) {
		t.Errorf("ProcessingState = %q, want %q", state.ProcessingState, MemoryStateNotProcessed)
	}
}

func TestRepository_GetProcessingState_MeetingNotFound(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				return ErrMeetingNotFound
			}}
		},
	}
	repo := &Repository{pool: pool}
	_, err := repo.GetProcessingState(context.Background(), uuid.New())
	if !errors.Is(err, ErrMeetingNotFound) {
		t.Errorf("GetProcessingState error = %v, want ErrMeetingNotFound", err)
	}
}

// ─── GetActiveVersionID ─────────────────────────────────────────────────────

func TestRepository_GetActiveVersionID(t *testing.T) {
	versionID := uuid.New()

	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			// Return a pointer to uuid.UUID (pgx scans into *uuid.UUID)
			return &mockRow{values: []any{&versionID}}
		},
	}
	repo := &Repository{pool: pool}
	id, err := repo.GetActiveVersionID(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("GetActiveVersionID unexpected error: %v", err)
	}
	if id != versionID {
		t.Errorf("GetActiveVersionID = %v, want %v", id, versionID)
	}
}

func TestRepository_GetActiveVersionID_Nil(t *testing.T) {
	pool := &mockPool{
		queryRowFn: func(ctx context.Context, sql string, args ...any) pgx.Row {
			// Simulate NULL: pass nil pointer to uuid.UUID (pgx leaves it nil)
			var nilID *uuid.UUID
			return &mockRow{values: []any{nilID}}
		},
	}
	repo := &Repository{pool: pool}
	id, err := repo.GetActiveVersionID(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("GetActiveVersionID unexpected error: %v", err)
	}
	if id != uuid.Nil {
		t.Errorf("GetActiveVersionID = %v, want uuid.Nil", id)
	}
}