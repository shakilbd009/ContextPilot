package meeting

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("meeting not found")
	ErrTokenUsed     = errors.New("idempotency token already used")
	ErrTokenNotFound = errors.New("idempotency token not found or expired")
)

// Repository handles meeting persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a Repository backed by the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateMeeting creates a meeting with participants in a transaction.
// It also records the idempotency token. Returns the created meeting ID.
func (r *Repository) CreateMeeting(
	ctx context.Context,
	title string,
	completedAt time.Time,
	transcript, notes *string,
	contentSource string,
	createdBy uuid.UUID,
	participants []ParticipantInput,
	idempotencyToken uuid.UUID,
) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	// Insert meeting
	var meetingID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO meetings (title, completedat, transcript, notes, contentsource, createdby, displayorder)
		VALUES ($1, $2, $3, $4, $5, $6, 0)
		RETURNING id
	`, title, completedAt, transcript, notes, contentSource, createdBy).Scan(&meetingID)
	if err != nil {
		return uuid.Nil, err
	}

	// Insert participants in display order
	for i, p := range participants {
		_, err = tx.Exec(ctx, `
			INSERT INTO meeting_participants (meetingid, displayname, email, organization, role, displayorder)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, meetingID, strings.TrimSpace(p.DisplayName), p.Email, p.Organization, p.Role, i)
		if err != nil {
			return uuid.Nil, err
		}
	}

	// Record idempotency token
	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_tokens (token, meetingid, userid, expiresat)
		VALUES ($1, $2, $3, NOW() + INTERVAL '24 hours')
	`, idempotencyToken, meetingID, createdBy)
	if err != nil {
		return uuid.Nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return meetingID, nil
}

// CheckIdempotencyToken checks if the token exists for the given user and is not expired.
// Returns the meeting ID and originalOutcome if found and not expired.
func (r *Repository) CheckIdempotencyToken(ctx context.Context, userID, token uuid.UUID) (uuid.UUID, string, error) {
	var meetingID uuid.UUID
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT meetingid, expiresat FROM idempotency_tokens
		WHERE token = $1 AND userid = $2 AND expiresat > NOW()
	`, token, userID).Scan(&meetingID, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, "", ErrTokenNotFound
		}
		return uuid.Nil, "", err
	}
	return meetingID, "completed", nil
}

// GetMeeting retrieves a meeting by ID with its participants.
func (r *Repository) GetMeeting(ctx context.Context, id uuid.UUID) (*Meeting, error) {
	var m Meeting
	var transcript, notes *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, completedat, transcript, notes, contentsource, createdat, createdby, displayorder
		FROM meetings
		WHERE id = $1
	`, id).Scan(&m.ID, &m.Title, &m.CompletedAt, &transcript, &notes, &m.ContentSource, &m.CreatedAt, &m.CreatedBy, &m.DisplayOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.Transcript = transcript
	m.Notes = notes

	rows, err := r.pool.Query(ctx, `
		SELECT id, displayname, email, organization, role, displayorder
		FROM meeting_participants
		WHERE meetingid = $1
		ORDER BY displayorder
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m.Participants = []Participant{}
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Organization, &p.Role, &p.DisplayOrder); err != nil {
			return nil, err
		}
		p.MeetingID = id
		m.Participants = append(m.Participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &m, nil
}

// GetMeetingOwned retrieves a meeting by ID only if it is owned by the given user.
// Used by handleGetMeeting to enforce ownership on the read path. Returns ErrNotFound
// when the meeting does not exist OR is owned by a different user; callers MUST NOT
// distinguish those cases to the client (return 404 either way to avoid existence leak).
// IDOR-FIX-CWE-639
func (r *Repository) GetMeetingOwned(ctx context.Context, id, userID uuid.UUID) (*Meeting, error) {
	var m Meeting
	var transcript, notes *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, completedat, transcript, notes, contentsource, createdat, createdby, displayorder
		FROM meetings
		WHERE id = $1 AND createdby = $2
	`, id, userID).Scan(&m.ID, &m.Title, &m.CompletedAt, &transcript, &notes, &m.ContentSource, &m.CreatedAt, &m.CreatedBy, &m.DisplayOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.Transcript = transcript
	m.Notes = notes

	rows, err := r.pool.Query(ctx, `
		SELECT id, displayname, email, organization, role, displayorder
		FROM meeting_participants
		WHERE meetingid = $1
		ORDER BY displayorder
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m.Participants = []Participant{}
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Organization, &p.Role, &p.DisplayOrder); err != nil {
			return nil, err
		}
		p.MeetingID = id
		m.Participants = append(m.Participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &m, nil
}

// ListMeetings returns meetings for a user with pagination.
// limit defaults to 20; offset is 0-based page index.
func (r *Repository) ListMeetings(ctx context.Context, createdBy uuid.UUID, limit, offset int) ([]Meeting, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, completedat, transcript, notes, contentsource, createdat, createdby, displayorder
		FROM meetings
		WHERE createdby = $1
		ORDER BY createdat DESC
		LIMIT $2 OFFSET $3
	`, createdBy, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		var transcript, notes *string
		if err := rows.Scan(&m.ID, &m.Title, &m.CompletedAt, &transcript, &notes, &m.ContentSource, &m.CreatedAt, &m.CreatedBy, &m.DisplayOrder); err != nil {
			return nil, err
		}
		m.Transcript = transcript
		m.Notes = notes
		meetings = append(meetings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return meetings, nil
}

// UpdateMeeting updates mutable fields of a meeting.
// Returns true if any of the watched source fields (transcript, notes, title, completed_at, participants) changed.
// The participants field being non-nil signals a participant list update.
func (r *Repository) UpdateMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	title *string,
	completedAt *time.Time,
	transcript, notes *string,
	participants []ParticipantInput,
) (bool, error) {
	// First, fetch current meeting to know what we're updating
	current, err := r.GetMeeting(ctx, meetingID)
	if err != nil {
		return false, err
	}

	// Determine which source fields are changing
	titleChanged := false
	completedAtChanged := false
	transcriptChanged := false
	notesChanged := false
	participantsChanged := participants != nil

	if title != nil && strings.TrimSpace(*title) != current.Title {
		titleChanged = true
	}
	if completedAt != nil && !completedAt.Equal(current.CompletedAt) {
		completedAtChanged = true
	}
	if transcript != nil {
		curTranscript := ""
		if current.Transcript != nil {
			curTranscript = *current.Transcript
		}
		if *transcript != curTranscript {
			transcriptChanged = true
		}
	}
	if notes != nil {
		curNotes := ""
		if current.Notes != nil {
			curNotes = *current.Notes
		}
		if *notes != curNotes {
			notesChanged = true
		}
	}

	hasUpdates := titleChanged || completedAtChanged || transcriptChanged || notesChanged || participantsChanged
	if !hasUpdates {
		return false, nil
	}

	// Update meeting row
	updatedTitle := current.Title
	if title != nil {
		updatedTitle = strings.TrimSpace(*title)
	}
	updatedCompletedAt := current.CompletedAt
	if completedAt != nil {
		updatedCompletedAt = *completedAt
	}
	updatedTranscript := current.Transcript
	if transcript != nil {
		updatedTranscript = transcript
	}
	updatedNotes := current.Notes
	if notes != nil {
		updatedNotes = notes
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE meetings
		SET title = $1, completedat = $2, transcript = $3, notes = $4, updatedat = now()
		WHERE id = $5
	`, updatedTitle, updatedCompletedAt, updatedTranscript, updatedNotes, meetingID)
	if err != nil {
		return false, err
	}

	// If participants are being updated, replace the participant list
	if participantsChanged {
		// Delete existing participants
		_, err = r.pool.Exec(ctx, `DELETE FROM meeting_participants WHERE meetingid = $1`, meetingID)
		if err != nil {
			return false, err
		}
		// Insert new participants
		for i, p := range participants {
			_, err = r.pool.Exec(ctx, `
				INSERT INTO meeting_participants (meetingid, displayname, email, organization, role, displayorder)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, meetingID, strings.TrimSpace(p.DisplayName), p.Email, p.Organization, p.Role, i)
			if err != nil {
				return false, err
			}
		}
	}

	return true, nil
}