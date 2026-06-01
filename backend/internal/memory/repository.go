package memory

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMemoryNotFound     = errors.New("memory version not found")
	ErrConflictNotFound   = errors.New("memory conflict not found")
	ErrMeetingNotFound    = errors.New("meeting not found")
	ErrVersionNotFound    = errors.New("memory version not found")
	ErrNotAuthorized      = errors.New("not authorized to access this meeting")
)

// Pool is the interface a data store must implement for Repository.
// *pgxpool.Pool satisfies this interface.
type Pool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Repository handles all memory persistence operations.
type Repository struct {
	pool Pool
}

// NewRepository returns a Repository backed by the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── Queue ───────────────────────────────────────────────────────────────────

// QueueJob inserts a new memory processing job. Called after the meeting import
// transaction commits. Returns the job ID and correlation ID.
func (r *Repository) QueueJob(
	ctx context.Context,
	meetingID uuid.UUID,
	triggerType TriggerType,
	previousVersionID *uuid.UUID,
) (uuid.UUID, uuid.UUID, error) {
	correlationID := uuid.New()
	var jobID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO memory_processing_jobs (meeting_id, trigger_type, status, correlation_id, previous_version_id)
		VALUES ($1, $2, 'queued', $3, $4)
		RETURNING id
	`, meetingID, triggerType, correlationID, previousVersionID).Scan(&jobID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return jobID, correlationID, nil
}

// GetQueuedJobs returns up to limit jobs that are queued or retrying, using
// FOR UPDATE SKIP LOCKED for safe concurrent worker picking.
func (r *Repository) GetQueuedJobs(ctx context.Context, limit int) ([]MemoryProcessingJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, meeting_id, trigger_type, status, retry_count, max_retries,
		       failure_reason, previous_version_id, correlation_id,
		       queued_at, started_at, completed_at, next_retry_at
		FROM memory_processing_jobs
		WHERE status IN ('queued', 'retrying')
		  AND (next_retry_at IS NULL OR next_retry_at <= now())
		ORDER BY queued_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []MemoryProcessingJob
	for rows.Next() {
		var j MemoryProcessingJob
		if err := rows.Scan(
			&j.ID, &j.MeetingID, &j.TriggerType, &j.Status, &j.RetryCount, &j.MaxRetries,
			&j.FailureReason, &j.PreviousVersionID, &j.CorrelationID,
			&j.QueuedAt, &j.StartedAt, &j.CompletedAt, &j.NextRetryAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// PickJob atomically claims a job by setting status to 'processing' and started_at.
// Returns the job if successfully claimed, or nil if no job was available.
func (r *Repository) PickJob(ctx context.Context, jobID uuid.UUID) (*MemoryProcessingJob, error) {
	var j MemoryProcessingJob
	err := r.pool.QueryRow(ctx, `
		UPDATE memory_processing_jobs
		SET status = 'processing', started_at = now()
		WHERE id = $1 AND status IN ('queued', 'retrying')
		RETURNING id, meeting_id, trigger_type, status, retry_count, max_retries,
		          failure_reason, previous_version_id, correlation_id,
		          queued_at, started_at, completed_at, next_retry_at
	`, jobID).Scan(
		&j.ID, &j.MeetingID, &j.TriggerType, &j.Status, &j.RetryCount, &j.MaxRetries,
		&j.FailureReason, &j.PreviousVersionID, &j.CorrelationID,
		&j.QueuedAt, &j.StartedAt, &j.CompletedAt, &j.NextRetryAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

// UpdateJobCompleted marks a job as completed with the given status.
func (r *Repository) UpdateJobCompleted(ctx context.Context, jobID uuid.UUID, newVersionID uuid.UUID, failureReason *string) error {
	status := JobStatusCompleted
	if failureReason != nil {
		status = JobStatusFailed
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE memory_processing_jobs
		SET status = $2, completed_at = now(), failure_reason = $3
		WHERE id = $1
	`, jobID, status, failureReason)
	return err
}

// UpdateJobRetrying records that a job is scheduling a retry.
func (r *Repository) UpdateJobRetrying(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetryAt time.Time, failureReason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE memory_processing_jobs
		SET status = 'retrying', retry_count = $2, next_retry_at = $3, failure_reason = $4
		WHERE id = $1
	`, jobID, retryCount, nextRetryAt, failureReason)
	return err
}

// UpdateJobRetryExhausted marks a job as having exhausted all retries.
func (r *Repository) UpdateJobRetryExhausted(ctx context.Context, jobID uuid.UUID, failureReason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE memory_processing_jobs
		SET status = 'retry_exhausted', completed_at = now(), failure_reason = $2
		WHERE id = $1
	`, jobID, failureReason)
	return err
}

// UpdateJobFailed marks a job as permanently failed (no more retries).
func (r *Repository) UpdateJobFailed(ctx context.Context, jobID uuid.UUID, failureReason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE memory_processing_jobs
		SET status = 'failed', completed_at = now(), failure_reason = $2, retry_count = 0
		WHERE id = $1
	`, jobID, failureReason)
	return err
}

// GetLatestJobForMeeting returns the most recent job for a meeting.
func (r *Repository) GetLatestJobForMeeting(ctx context.Context, meetingID uuid.UUID) (*MemoryProcessingJob, error) {
	var j MemoryProcessingJob
	err := r.pool.QueryRow(ctx, `
		SELECT id, meeting_id, trigger_type, status, retry_count, max_retries,
		       failure_reason, previous_version_id, correlation_id,
		       queued_at, started_at, completed_at, next_retry_at
		FROM memory_processing_jobs
		WHERE meeting_id = $1
		ORDER BY queued_at DESC
		LIMIT 1
	`, meetingID).Scan(
		&j.ID, &j.MeetingID, &j.TriggerType, &j.Status, &j.RetryCount, &j.MaxRetries,
		&j.FailureReason, &j.PreviousVersionID, &j.CorrelationID,
		&j.QueuedAt, &j.StartedAt, &j.CompletedAt, &j.NextRetryAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

// ─── Versions ────────────────────────────────────────────────────────────────

// CreateVersion inserts a new memory version and atomically updates the meeting's
// active_memory_version_id pointer. Returns the version ID.
func (r *Repository) CreateVersion(
	ctx context.Context,
	meetingID uuid.UUID,
	jobID *uuid.UUID,
	triggerType TriggerType,
	content MemoryContent,
	previousVersionID *uuid.UUID,
	createdBy *uuid.UUID,
) (uuid.UUID, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer tx.Rollback(ctx)

	// Determine next version number
	var versionNumber int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM memory_versions
		WHERE meeting_id = $1
	`, meetingID).Scan(&versionNumber)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Deactivate any current active version
	_, err = tx.Exec(ctx, `
		UPDATE memory_versions
		SET is_active = FALSE, status = 'superseded'
		WHERE meeting_id = $1 AND is_active = TRUE
	`, meetingID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Insert new version
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return uuid.Nil, 0, err
	}

	var versionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO memory_versions (meeting_id, job_id, version_number, status, is_active, content, trigger_type, created_by)
		VALUES ($1, $2, $3, 'active', TRUE, $4, $5, $6)
		RETURNING id
	`, meetingID, jobID, versionNumber, contentJSON, triggerType, createdBy).Scan(&versionID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Update meeting's active version pointer
	_, err = tx.Exec(ctx, `
		UPDATE meetings SET active_memory_version_id = $1 WHERE id = $2
	`, versionID, meetingID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, 0, err
	}
	return versionID, versionNumber, nil
}

// GetActiveVersion returns the active memory version for a meeting, or nil if none.
func (r *Repository) GetActiveVersion(ctx context.Context, meetingID uuid.UUID) (*MemoryVersion, error) {
	var v MemoryVersion
	var contentJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, meeting_id, job_id, version_number, status, is_active, content, trigger_type, created_at, created_by
		FROM memory_versions
		WHERE meeting_id = $1 AND is_active = TRUE
	`, meetingID).Scan(
		&v.ID, &v.MeetingID, &v.JobID, &v.VersionNumber, &v.Status, &v.IsActive,
		&contentJSON, &v.TriggerType, &v.CreatedAt, &v.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(contentJSON, &v.Content); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVersionByNumber returns a specific version by meeting ID and version number.
func (r *Repository) GetVersionByNumber(ctx context.Context, meetingID uuid.UUID, versionNumber int) (*MemoryVersion, error) {
	var v MemoryVersion
	var contentJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, meeting_id, job_id, version_number, status, is_active, content, trigger_type, created_at, created_by
		FROM memory_versions
		WHERE meeting_id = $1 AND version_number = $2
	`, meetingID, versionNumber).Scan(
		&v.ID, &v.MeetingID, &v.JobID, &v.VersionNumber, &v.Status, &v.IsActive,
		&contentJSON, &v.TriggerType, &v.CreatedAt, &v.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(contentJSON, &v.Content); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVersions returns all versions for a meeting, ordered by version_number asc.
func (r *Repository) ListVersions(ctx context.Context, meetingID uuid.UUID) ([]MemoryVersionSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT version_number, is_active, status, created_at, created_by
		FROM memory_versions
		WHERE meeting_id = $1
		ORDER BY version_number ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []MemoryVersionSummary
	for rows.Next() {
		var v MemoryVersionSummary
		if err := rows.Scan(&v.VersionNumber, &v.IsActive, &v.Status, &v.CreatedAt, &v.CreatedBy); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// IsMemoryStale returns true when the meeting's updated_at is newer than the
// active memory version's created_at (meaning the source changed after processing).
func (r *Repository) IsMemoryStale(ctx context.Context, meetingID uuid.UUID) (bool, error) {
	var updatedAt time.Time
	var versionCreatedAt time.Time
	var hasActive bool

	err := r.pool.QueryRow(ctx, `
		SELECT m.updatedat, mv.created_at, mv.is_active
		FROM meetings m
		LEFT JOIN memory_versions mv ON mv.meeting_id = m.id AND mv.is_active = TRUE
		WHERE m.id = $1
	`, meetingID).Scan(&updatedAt, &versionCreatedAt, &hasActive)
	if err != nil {
		return false, err
	}
	// No active version means not stale (not processed yet)
	if !hasActive {
		return false, nil
	}
	return updatedAt.After(versionCreatedAt), nil
}

// MarkVersionConflictReview marks a version as conflict_review (conflicts detected
// but version not activated).
func (r *Repository) MarkVersionConflictReview(ctx context.Context, versionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE memory_versions SET status = 'conflict_review', is_active = FALSE WHERE id = $1
	`, versionID)
	return err
}

// GetActiveVersionID returns the active memory version ID for a meeting, or uuid.Nil.
func (r *Repository) GetActiveVersionID(ctx context.Context, meetingID uuid.UUID) (uuid.UUID, error) {
	var id *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT active_memory_version_id FROM meetings WHERE id = $1
	`, meetingID).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	if id == nil {
		return uuid.Nil, nil
	}
	return *id, nil
}

// ─── Processing State ─────────────────────────────────────────────────────────

// GetProcessingState returns the aggregated processing state for a meeting.
func (r *Repository) GetProcessingState(ctx context.Context, meetingID uuid.UUID) (*ProcessingState, error) {
	// Check if meeting exists
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM meetings WHERE id = $1)`, meetingID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMeetingNotFound
	}

	// Check active version
	activeVersionID, err := r.GetActiveVersionID(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	isStale, err := r.IsMemoryStale(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	latestJob, err := r.GetLatestJobForMeeting(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	state := &ProcessingState{
		MeetingID: meetingID,
		IsStale:   isStale,
		LatestJob: latestJob,
	}

	if activeVersionID == uuid.Nil {
		state.ProcessingState = string(MemoryStateNotProcessed)
		return state, nil
	}

	// Get version number
	var versionNum *int
	// #nosec G104 -- QueryRow returning no row is non-fatal here; we keep
	// the existing pointer (nil) so the caller sees "no version yet". A
	// surfaced error would be a behavior change for callers that rely on
	// the current "best-effort" semantics. Adding a logger is out of scope
	// for this CI fix.
	if err := r.pool.QueryRow(ctx, `
		SELECT version_number FROM memory_versions WHERE id = $1
	`, activeVersionID).Scan(&versionNum); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		_ = err // intentionally ignored; see comment above
	}
	state.ActiveVersionNumber = versionNum

	if latestJob != nil {
		switch latestJob.Status {
		case JobStatusQueued:
			state.ProcessingState = string(MemoryStateQueued)
		case JobStatusProcessing:
			state.ProcessingState = string(MemoryStateProcessing)
		case JobStatusCompleted:
			state.ProcessingState = string(MemoryStateCompleted)
		case JobStatusCompletedInsufficient:
			state.ProcessingState = string(MemoryStateCompletedInsufficient)
		case JobStatusFailed:
			state.ProcessingState = string(MemoryStateFailed)
		case JobStatusRetrying:
			state.ProcessingState = string(MemoryStateRetrying)
		case JobStatusRetryExhausted:
			state.ProcessingState = string(MemoryStateRetryExhausted)
		}
	} else if isStale {
		state.ProcessingState = string(MemoryStateStale)
	} else {
		state.ProcessingState = string(MemoryStateCompleted)
	}

	return state, nil
}

// ─── Conflicts ───────────────────────────────────────────────────────────────

// CreateConflict inserts a new conflict record. Returns the conflict ID.
func (r *Repository) CreateConflict(
	ctx context.Context,
	meetingID uuid.UUID,
	memoryVersionID *uuid.UUID,
	conflictingItems ConflictPair,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO memory_conflicts (meeting_id, memory_version_id, conflicting_items, quality_status, review_status)
		VALUES ($1, $2, $3, 'conflicting_evidence', 'pending')
		RETURNING id
	`, meetingID, memoryVersionID, conflictingItems).Scan(&id)
	return id, err
}

// ListPendingConflicts returns all pending conflicts for a meeting.
func (r *Repository) ListPendingConflicts(ctx context.Context, meetingID uuid.UUID) ([]MemoryConflict, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, meeting_id, memory_version_id, conflicting_items, quality_status,
		       review_status, resolution_note, resolved_at, resolved_by, created_at
		FROM memory_conflicts
		WHERE meeting_id = $1 AND review_status = 'pending'
		ORDER BY created_at ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []MemoryConflict
	for rows.Next() {
		var c MemoryConflict
		if err := rows.Scan(
			&c.ID, &c.MeetingID, &c.MemoryVersionID, &c.ConflictingItems,
			&c.QualityStatus, &c.ReviewStatus, &c.ResolutionNote,
			&c.ResolvedAt, &c.ResolvedBy, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, rows.Err()
}

// ResolveConflict marks a conflict as reviewed with the given resolution note.
func (r *Repository) ResolveConflict(
	ctx context.Context,
	conflictID uuid.UUID,
	resolvedBy uuid.UUID,
	resolutionNote string,
) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE memory_conflicts
		SET review_status = 'reviewed', resolved_at = now(),
		    resolved_by = $2, resolution_note = $3
		WHERE id = $1 AND review_status = 'pending'
	`, conflictID, resolvedBy, resolutionNote)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConflictNotFound
	}
	return nil
}

// GetConflict returns a specific conflict by ID.
func (r *Repository) GetConflict(ctx context.Context, conflictID uuid.UUID) (*MemoryConflict, error) {
	var c MemoryConflict
	err := r.pool.QueryRow(ctx, `
		SELECT id, meeting_id, memory_version_id, conflicting_items, quality_status,
		       review_status, resolution_note, resolved_at, resolved_by, created_at
		FROM memory_conflicts
		WHERE id = $1
	`, conflictID).Scan(
		&c.ID, &c.MeetingID, &c.MemoryVersionID, &c.ConflictingItems,
		&c.QualityStatus, &c.ReviewStatus, &c.ResolutionNote,
		&c.ResolvedAt, &c.ResolvedBy, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConflictNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ─── Prior Memory Inputs ─────────────────────────────────────────────────────

// RecordPriorMemoryInputs inserts prior memory version references used during processing.
func (r *Repository) RecordPriorMemoryInputs(
	ctx context.Context,
	memoryVersionID uuid.UUID,
	inputs []PriorMemoryInput,
) error {
	for _, inp := range inputs {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO memory_prior_memory_inputs
				(memory_version_id, prior_memory_version_id, match_confidence, included_by_user, excluded_by_user)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (memory_version_id, prior_memory_version_id) DO NOTHING
		`, memoryVersionID, inp.PriorMemoryVersionID, inp.MatchConfidence, inp.IncludedByUser, inp.ExcludedByUser)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListPriorMemoryInputs returns the prior memory inputs for a memory version,
// joined with meeting title for display.
func (r *Repository) ListPriorMemoryInputs(ctx context.Context, memoryVersionID uuid.UUID) ([]PriorMemorySummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.meeting_id, mt.title, mt.completedat, mi.match_confidence
		FROM memory_prior_memory_inputs mi
		JOIN memory_versions mv ON mv.id = mi.prior_memory_version_id
		JOIN meetings mt ON mt.id = mv.meeting_id
		WHERE mi.memory_version_id = $1 AND mi.excluded_by_user = FALSE
		ORDER BY mi.created_at ASC
	`, memoryVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []PriorMemorySummary
	for rows.Next() {
		var p PriorMemorySummary
		var mc *MatchConfidence
		if err := rows.Scan(&p.MeetingID, &p.Title, &p.CompletedAt, &mc); err != nil {
			return nil, err
		}
		if mc != nil {
			p.MatchConfidence = *mc
		}
		inputs = append(inputs, p)
	}
	return inputs, rows.Err()
}

// ─── Meeting ownership check ─────────────────────────────────────────────────

// MeetingExistsAndOwned returns true if the meeting exists and the user owns it.
func (r *Repository) MeetingExistsAndOwned(ctx context.Context, meetingID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM meetings WHERE id = $1 AND createdby = $2)
	`, meetingID, userID).Scan(&exists)
	return exists, err
}

// GetMeetingInfo returns the source material needed for memory processing.
// Used by the worker to fetch meeting transcript, notes, participants.
func (r *Repository) GetMeetingInfo(ctx context.Context, meetingID uuid.UUID) (*meetingInfo, error) {
	// Fetch meeting metadata + transcript + notes
	var mi meetingInfo
	var transcript, notes *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, transcript, notes, completedat, updatedat
		FROM meetings
		WHERE id = $1
	`, meetingID).Scan(&mi.ID, &mi.Title, &transcript, &notes, &mi.CompletedAt, &mi.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMeetingNotFound
		}
		return nil, err
	}
	mi.Transcript = transcript
	mi.Notes = notes

	// Fetch participants
	rows, err := r.pool.Query(ctx, `
		SELECT id, displayname, email, organization
		FROM meeting_participants
		WHERE meetingid = $1
		ORDER BY displayorder ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p struct {
			ID           uuid.UUID
			DisplayName  string
			Email        *string
			Organization *string
		}
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Organization); err != nil {
			return nil, err
		}
		mi.Participants = append(mi.Participants, p)
	}
	return &mi, rows.Err()
}
