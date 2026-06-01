package upcoming

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("upcoming meeting not found")

// Pool abstracts pgxpool.Pool for repository testing.
type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	Pool Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{Pool: pool}
}

// ─── Create ─────────────────────────────────────────────────────────────────

type CreateMeetingInput struct {
	Title                string
	ScheduledStart       time.Time
	Description         *string
	ClientOrOrganization *string
	CreatedBy            uuid.UUID
	Participants        []CreateParticipantInput
}

type CreateParticipantInput struct {
	DisplayName  *string
	Email        *string
	Organization *string
}

func (r *Repository) CreateMeeting(ctx context.Context, in CreateMeetingInput) (uuid.UUID, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var meetingID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO upcoming_meetings (title, scheduled_start, description, client_or_organization, status, created_by)
		VALUES ($1, $2, $3, $4, 'scheduled', $5)
		RETURNING id
	`, in.Title, in.ScheduledStart, in.Description, in.ClientOrOrganization, in.CreatedBy).Scan(&meetingID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, p := range in.Participants {
		_, err = tx.Exec(ctx, `
			INSERT INTO upcoming_meeting_participants (upcoming_meeting_id, display_name, email, organization)
			VALUES ($1, $2, $3, $4)
		`, meetingID, strPtr(p.DisplayName), strPtr(p.Email), strPtr(p.Organization))
		if err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return meetingID, nil
}

// ─── Read ──────────────────────────────────────────────────────────────────

func (r *Repository) GetMeeting(ctx context.Context, id uuid.UUID) (*UpcomingMeeting, error) {
	m := &UpcomingMeeting{}
	err := r.Pool.QueryRow(ctx, `
		SELECT id, title, scheduled_start, description, client_or_organization, status, created_by, created_at, updated_at
		FROM upcoming_meetings WHERE id = $1
	`, id).Scan(&m.ID, &m.Title, &m.ScheduledStart, &m.Description, &m.ClientOrOrganization, &m.Status, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	participants, err := r.getParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Participants = participants
	return m, nil
}

func (r *Repository) ListMeetings(ctx context.Context, createdBy uuid.UUID, status string) ([]UpcomingMeeting, error) {
	query := `
		SELECT id, title, scheduled_start, description, client_or_organization, status, created_by, created_at, updated_at
		FROM upcoming_meetings WHERE created_by = $1`
	args := []any{createdBy}

	switch status {
	case "all":
		// no additional filter
	case "cancelled":
		query += " AND status = 'cancelled'"
	default:
		query += " AND status = 'scheduled'"
	}
	query += " ORDER BY scheduled_start ASC"

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []UpcomingMeeting
	for rows.Next() {
		var m UpcomingMeeting
		err := rows.Scan(&m.ID, &m.Title, &m.ScheduledStart, &m.Description, &m.ClientOrOrganization, &m.Status, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		participants, err := r.getParticipants(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		m.Participants = participants
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

func (r *Repository) GetMeetingOwned(ctx context.Context, id, userID uuid.UUID) (*UpcomingMeeting, error) {
	m := &UpcomingMeeting{}
	err := r.Pool.QueryRow(ctx, `
		SELECT id, title, scheduled_start, description, client_or_organization, status, created_by, created_at, updated_at
		FROM upcoming_meetings WHERE id = $1 AND created_by = $2
	`, id, userID).Scan(&m.ID, &m.Title, &m.ScheduledStart, &m.Description, &m.ClientOrOrganization, &m.Status, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	participants, err := r.getParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Participants = participants
	return m, nil
}

// ─── Update ─────────────────────────────────────────────────────────────────

type UpdateMeetingInput struct {
	Title                *string
	ScheduledStart       *time.Time
	Description         *string
	ClientOrOrganization *string
	Participants        []CreateParticipantInput
}

func (r *Repository) UpdateMeeting(ctx context.Context, meetingID uuid.UUID, in UpdateMeetingInput) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if in.Title != nil || in.ScheduledStart != nil || in.Description != nil || in.ClientOrOrganization != nil {
		_, err = tx.Exec(ctx, `
			UPDATE upcoming_meetings
			SET title = COALESCE($1, title),
			    scheduled_start = COALESCE($2, scheduled_start),
			    description = COALESCE($3, description),
			    client_or_organization = COALESCE($4, client_or_organization),
			    updated_at = now()
			WHERE id = $5
		`, in.Title, in.ScheduledStart, in.Description, in.ClientOrOrganization, meetingID)
		if err != nil {
			return err
		}
	}

	if in.Participants != nil {
		_, err = tx.Exec(ctx, "DELETE FROM upcoming_meeting_participants WHERE upcoming_meeting_id = $1", meetingID)
		if err != nil {
			return err
		}
		for _, p := range in.Participants {
			_, err = tx.Exec(ctx, `
				INSERT INTO upcoming_meeting_participants (upcoming_meeting_id, display_name, email, organization)
				VALUES ($1, $2, $3, $4)
			`, meetingID, strPtr(p.DisplayName), strPtr(p.Email), strPtr(p.Organization))
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// ─── Cancel ─────────────────────────────────────────────────────────────────

func (r *Repository) CancelMeeting(ctx context.Context, meetingID uuid.UUID) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE upcoming_meetings SET status = 'cancelled', updated_at = now()
		WHERE id = $1 AND status = 'scheduled'
	`, meetingID)
	return err
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (r *Repository) getParticipants(ctx context.Context, meetingID uuid.UUID) ([]UpcomingMeetingParticipant, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, display_name, email, organization, created_at, updated_at
		FROM upcoming_meeting_participants WHERE upcoming_meeting_id = $1 ORDER BY created_at ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []UpcomingMeetingParticipant
	for rows.Next() {
		var p UpcomingMeetingParticipant
		err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Organization, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}