package briefing

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
	ErrBriefingNotFound     = errors.New("briefing not found")
	ErrVersionNotFound      = errors.New("briefing version not found")
	ErrUpcomingNotFound     = errors.New("upcoming meeting not found")
	ErrNotAuthorized        = errors.New("not authorized to access this meeting")
	ErrExclusionNotFound    = errors.New("exclusion not found")
	ErrJobNotFound          = errors.New("job not found")
)

// Pool is the interface a data store must implement for Repository.
type Pool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Repository handles all briefing persistence operations.
type Repository struct {
	pool Pool
}

// NewRepository returns a Repository backed by the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── Upcoming Meeting ACL ───────────────────────────────────────────────────

// UpcomingMeetingExistsAndOwned checks if an upcoming meeting exists and the user owns it.
func (r *Repository) UpcomingMeetingExistsAndOwned(ctx context.Context, meetingID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM upcoming_meetings WHERE id = $1 AND created_by = $2
		)
	`, meetingID, userID).Scan(&exists)
	return exists, err
}

// UpcomingMeetingExists checks if an upcoming meeting exists at all.
func (r *Repository) UpcomingMeetingExists(ctx context.Context, meetingID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM upcoming_meetings WHERE id = $1)
	`, meetingID).Scan(&exists)
	return exists, err
}

// ─── Briefing Versions ───────────────────────────────────────────────────────

// GetActiveBriefingVersion returns the active briefing version for an upcoming meeting.
func (r *Repository) GetActiveBriefingVersion(ctx context.Context, upcomingMeetingID uuid.UUID) (*BriefingVersion, error) {
	query := `
		SELECT id, upcoming_meeting_id, version_number, status, is_active, result,
		       preparation_status, content, source_count, source_memory_version_ids,
		       created_at, trigger_type
		FROM briefing_versions
		WHERE upcoming_meeting_id = $1 AND is_active = TRUE
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, upcomingMeetingID)
	return r.scanBriefingVersion(row)
}

// GetBriefingVersionByNumber returns a specific briefing version by version number.
func (r *Repository) GetBriefingVersionByNumber(ctx context.Context, upcomingMeetingID uuid.UUID, versionNum int) (*BriefingVersion, error) {
	query := `
		SELECT id, upcoming_meeting_id, version_number, status, is_active, result,
		       preparation_status, content, source_count, source_memory_version_ids,
		       created_at, trigger_type
		FROM briefing_versions
		WHERE upcoming_meeting_id = $1 AND version_number = $2
	`
	row := r.pool.QueryRow(ctx, query, upcomingMeetingID, versionNum)
	return r.scanBriefingVersion(row)
}

// ListBriefingVersions returns all briefing versions for an upcoming meeting, ordered by version_number.
func (r *Repository) ListBriefingVersions(ctx context.Context, upcomingMeetingID uuid.UUID) ([]BriefingVersionSummary, error) {
	query := `
		SELECT version_number, is_active, status, result, created_at, trigger_type
		FROM briefing_versions
		WHERE upcoming_meeting_id = $1
		ORDER BY version_number ASC
	`
	rows, err := r.pool.Query(ctx, query, upcomingMeetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []BriefingVersionSummary
	for rows.Next() {
		var v BriefingVersionSummary
		var result *string
		if err := rows.Scan(&v.VersionNumber, &v.IsActive, &v.Status, &result, &v.CreatedAt, &v.TriggerType); err != nil {
			return nil, err
		}
		if result != nil {
			v.Result = *result
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// CreateBriefingVersion inserts a new briefing version and returns its ID and version number.
// It also deactivates any previously active version.
func (r *Repository) CreateBriefingVersion(
	ctx context.Context,
	upcomingMeetingID uuid.UUID,
	triggerType BriefingTriggerType,
	result BriefingResult,
	preparationStatus PreparationStatus,
	content BriefingContent,
	sourceCount int,
	sourceMemoryVersionIDs []uuid.UUID,
	previousVersionID *uuid.UUID,
) (uuid.UUID, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer tx.Rollback(ctx)

	// Deactivate any active version
	_, err = tx.Exec(ctx, `
		UPDATE briefing_versions
		SET is_active = FALSE, status = 'superseded'
		WHERE upcoming_meeting_id = $1 AND is_active = TRUE
	`, upcomingMeetingID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Get next version number
	var versionNumber int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM briefing_versions
		WHERE upcoming_meeting_id = $1
	`, upcomingMeetingID).Scan(&versionNumber)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Insert new version
	var versionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO briefing_versions
		  (upcoming_meeting_id, version_number, status, is_active, result,
		   preparation_status, content, source_count, source_memory_version_ids, trigger_type)
		VALUES ($1, $2, 'active', TRUE, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, upcomingMeetingID, versionNumber, result, preparationStatus, content, sourceCount, sourceMemoryVersionIDs, triggerType).
		Scan(&versionID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, 0, err
	}
	return versionID, versionNumber, nil
}

// GetCurrentSourceMemoryVersionIDs returns the current memory version IDs for a set of source meetings.
// Used for staleness detection: compare with the IDs captured at briefing generation time.
func (r *Repository) GetCurrentSourceMemoryVersionIDs(ctx context.Context, sourceMeetingIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if len(sourceMeetingIDs) == 0 {
		return make(map[uuid.UUID]uuid.UUID), nil
	}
	query := `
		SELECT m.id, mv.id
		FROM meetings m
		LEFT JOIN memory_versions mv ON mv.meeting_id = m.id AND mv.is_active = TRUE
		WHERE m.id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, query, sourceMeetingIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[uuid.UUID]uuid.UUID)
	for rows.Next() {
		var meetingID, versionID uuid.UUID
		if err := rows.Scan(&meetingID, &versionID); err != nil {
			return nil, err
		}
		versions[meetingID] = versionID
	}
	return versions, rows.Err()
}

// GetCurrentSourceMemoryVersionIDsByMemoryID returns the current active memory version ID
// for each memory version ID in the input list.
// This is used for staleness detection: given the memory version IDs stored in a briefing
// at generation time, find which ones are still current.
func (r *Repository) GetCurrentSourceMemoryVersionIDsByMemoryID(ctx context.Context, memoryVersionIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if len(memoryVersionIDs) == 0 {
		return make(map[uuid.UUID]uuid.UUID), nil
	}
	query := `
		SELECT mv.id, cur.id
		FROM memory_versions mv
		JOIN meetings m ON m.id = mv.meeting_id
		LEFT JOIN memory_versions cur ON cur.meeting_id = m.id AND cur.is_active = TRUE
		WHERE mv.id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, query, memoryVersionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[uuid.UUID]uuid.UUID)
	for rows.Next() {
		var historicalID, currentID uuid.UUID
		if err := rows.Scan(&historicalID, &currentID); err != nil {
			return nil, err
		}
		versions[historicalID] = currentID
	}
	return versions, rows.Err()
}

// ─── Briefing Source Exclusions ──────────────────────────────────────────────

// ListActiveExclusions returns the active (non-restored) source exclusions for an upcoming meeting.
func (r *Repository) ListActiveExclusions(ctx context.Context, upcomingMeetingID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT excluded_source_meeting_id
		FROM briefing_source_exclusions
		WHERE upcoming_meeting_id = $1 AND restored_at IS NULL
	`
	rows, err := r.pool.Query(ctx, query, upcomingMeetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListAllExclusions returns all exclusions (active and restored) for an upcoming meeting.
func (r *Repository) ListAllExclusions(ctx context.Context, upcomingMeetingID uuid.UUID) ([]BriefingSourceExclusion, error) {
	query := `
		SELECT id, upcoming_meeting_id, excluded_source_meeting_id, created_at, restored_at
		FROM briefing_source_exclusions
		WHERE upcoming_meeting_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, upcomingMeetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exclusions []BriefingSourceExclusion
	for rows.Next() {
		var e BriefingSourceExclusion
		if err := rows.Scan(&e.ID, &e.UpcomingMeetingID, &e.ExcludedSourceMeetingID, &e.CreatedAt, &e.RestoredAt); err != nil {
			return nil, err
		}
		exclusions = append(exclusions, e)
	}
	return exclusions, rows.Err()
}

// UpsertExclusion creates or keeps an exclusion; idempotent.
// Per ADR-0012: restores previously soft-deleted exclusion if restored_at IS NOT NULL.
func (r *Repository) UpsertExclusion(ctx context.Context, upcomingMeetingID, sourceMeetingID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO briefing_source_exclusions (upcoming_meeting_id, excluded_source_meeting_id)
		VALUES ($1, $2)
		ON CONFLICT (upcoming_meeting_id, excluded_source_meeting_id)
		DO UPDATE SET restored_at = NULL
		WHERE briefing_source_exclusions.restored_at IS NOT NULL
	`, upcomingMeetingID, sourceMeetingID)
	return err
}

// RestoreExclusion clears restored_at for an exclusion (undoes exclusion).
func (r *Repository) RestoreExclusion(ctx context.Context, upcomingMeetingID, sourceMeetingID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE briefing_source_exclusions
		SET restored_at = now()
		WHERE upcoming_meeting_id = $1 AND excluded_source_meeting_id = $2 AND restored_at IS NULL
	`, upcomingMeetingID, sourceMeetingID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrExclusionNotFound
	}
	return nil
}

// ─── Job Queue ────────────────────────────────────────────────────────────────

// QueueBriefingJob inserts a new briefing processing job.
func (r *Repository) QueueBriefingJob(
	ctx context.Context,
	upcomingMeetingID uuid.UUID,
	triggerType BriefingTriggerType,
	previousVersionID *uuid.UUID,
) (uuid.UUID, uuid.UUID, error) {
	var jobID, correlationID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO briefing_processing_jobs (upcoming_meeting_id, trigger_type, previous_version_id)
		VALUES ($1, $2, $3)
		RETURNING id, correlation_id
	`, upcomingMeetingID, triggerType, previousVersionID).Scan(&jobID, &correlationID)
	return jobID, correlationID, err
}

// GetQueuedJobs returns up to N queued jobs for processing.
func (r *Repository) GetQueuedJobs(ctx context.Context, limit int) ([]BriefingProcessingJob, error) {
	query := `
		SELECT id, upcoming_meeting_id, trigger_type, status, retry_count, max_retries,
		       failure_reason, previous_version_id, correlation_id, queued_at,
		       started_at, completed_at, next_retry_at
		FROM briefing_processing_jobs
		WHERE status = 'queued'
		ORDER BY queued_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []BriefingProcessingJob
	for rows.Next() {
		var j BriefingProcessingJob
		if err := rows.Scan(
			&j.ID, &j.UpcomingMeetingID, &j.TriggerType, &j.Status,
			&j.RetryCount, &j.MaxRetries, &j.FailureReason,
			&j.PreviousVersionID, &j.CorrelationID, &j.QueuedAt,
			&j.StartedAt, &j.CompletedAt, &j.NextRetryAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// PickJob atomically claims a job for processing.
func (r *Repository) PickJob(ctx context.Context, jobID uuid.UUID) (*BriefingProcessingJob, error) {
	query := `
		UPDATE briefing_processing_jobs
		SET status = 'processing', started_at = now()
		WHERE id = $1 AND status = 'queued'
		RETURNING id, upcoming_meeting_id, trigger_type, status, retry_count, max_retries,
		          failure_reason, previous_version_id, correlation_id, queued_at,
		          started_at, completed_at, next_retry_at
	`
	row := r.pool.QueryRow(ctx, query, jobID)
	var j BriefingProcessingJob
	err := row.Scan(
		&j.ID, &j.UpcomingMeetingID, &j.TriggerType, &j.Status,
		&j.RetryCount, &j.MaxRetries, &j.FailureReason,
		&j.PreviousVersionID, &j.CorrelationID, &j.QueuedAt,
		&j.StartedAt, &j.CompletedAt, &j.NextRetryAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// UpdateJobCompleted marks a job as completed.
func (r *Repository) UpdateJobCompleted(ctx context.Context, jobID uuid.UUID, versionID uuid.UUID, failureReason *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE briefing_processing_jobs
		SET status = 'completed', completed_at = now(), failure_reason = $2
		WHERE id = $1
	`, jobID, failureReason)
	return err
}

// UpdateJobFailed marks a job as permanently failed.
func (r *Repository) UpdateJobFailed(ctx context.Context, jobID uuid.UUID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE briefing_processing_jobs
		SET status = 'failed', failure_reason = $2, completed_at = now()
		WHERE id = $1
	`, jobID, reason)
	return err
}

// UpdateJobRetrying reschedules a failed job for retry.
func (r *Repository) UpdateJobRetrying(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetryAt time.Time, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE briefing_processing_jobs
		SET status = 'retrying', retry_count = $2, next_retry_at = $3, failure_reason = $4
		WHERE id = $1
	`, jobID, retryCount, nextRetryAt, reason)
	return err
}

// HasActiveJob returns true if there's a non-terminal job for the upcoming meeting.
func (r *Repository) HasActiveJob(ctx context.Context, upcomingMeetingID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM briefing_processing_jobs
			WHERE upcoming_meeting_id = $1 AND status IN ('queued', 'processing', 'retrying')
		)
	`, upcomingMeetingID).Scan(&exists)
	return exists, err
}

// ─── Source Meeting Authorization ────────────────────────────────────────────

// CanAccessMeeting checks if a user can access a completed meeting (source meeting ACL).
func (r *Repository) CanAccessMeeting(ctx context.Context, meetingID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM meetings WHERE id = $1 AND createdby = $2
		)
	`, meetingID, userID).Scan(&exists)
	return exists, err
}

// ─── Scan helper ─────────────────────────────────────────────────────────────

func (r *Repository) scanBriefingVersion(row pgx.Row) (*BriefingVersion, error) {
	var v BriefingVersion
	var contentJSON []byte
	err := row.Scan(
		&v.ID, &v.UpcomingMeetingID, &v.VersionNumber, &v.Status, &v.IsActive,
		&v.Result, &v.PreparationStatus, &contentJSON, &v.SourceCount,
		&v.SourceMemoryVersionIDs, &v.CreatedAt, &v.TriggerType,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBriefingNotFound
	}
	if err != nil {
		return nil, err
	}
	// JSONB content is scanned as []byte; unmarshal into BriefingVersion.Content
	if len(contentJSON) > 0 {
		if err := json.Unmarshal(contentJSON, &v.Content); err != nil {
			return nil, err
		}
	}
	return &v, nil
}

// GetUpcomingMeeting returns the upcoming meeting by ID.
func (r *Repository) GetUpcomingMeeting(ctx context.Context, id uuid.UUID) (*UpcomingMeetingBriefing, error) {
	var m UpcomingMeetingBriefing
	var desc *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, scheduled_start, description, client_or_organization,
		       created_by, created_at
		FROM upcoming_meetings WHERE id = $1
	`, id).Scan(&m.ID, &m.Title, &m.ScheduledStart, &desc, &m.ClientOrOrganization, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.Description = desc
	participants, err := r.getUpcomingParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Participants = participants
	return &m, nil
}

func (r *Repository) getUpcomingParticipants(ctx context.Context, meetingID uuid.UUID) ([]UpcomingParticipantBriefing, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT display_name, email, organization
		FROM upcoming_meeting_participants WHERE upcoming_meeting_id = $1
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var participants []UpcomingParticipantBriefing
	for rows.Next() {
		var p UpcomingParticipantBriefing
		if err := rows.Scan(&p.DisplayName, &p.Email, &p.Organization); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}