package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/contextpilot/backend/internal/briefing"
)

// briefingSourceAdapter implements briefing.MeetingSourceRepo against the
// shared pgxpool, bridging the briefing service's source-selection queries to
// the meetings + memory_versions tables.
type briefingSourceAdapter struct {
	pool *pgxpool.Pool
}

// ListCompletedMeetingsWithMemory returns completed meetings that have an active
// memory version for a given user. It enriches each meeting with its active memory
// content, participant email, and participant org for signal matching.
func (a *briefingSourceAdapter) ListCompletedMeetingsWithMemory(
	ctx context.Context,
	userID uuid.UUID,
) ([]briefing.MeetingWithMemory, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT
			m.id,
			m.title,
			EXTRACT(EPOCH FROM m.completedat)::bigint  AS completed_at,
			mv.id                                       AS active_memory_version_id,
			mv.content                                  AS memory_json,
			(SELECT mp.email
			 FROM meeting_participants mp
			 WHERE mp.meetingid = m.id AND mp.email IS NOT NULL AND mp.email != ''
			 LIMIT 1)                                   AS participant_email,
			(SELECT mp.organization
			 FROM meeting_participants mp
			 WHERE mp.meetingid = m.id AND mp.organization IS NOT NULL AND mp.organization != ''
			 LIMIT 1)                                   AS participant_org
		FROM meetings m
		LEFT JOIN memory_versions mv
		  ON mv.meeting_id = m.id AND mv.is_active = TRUE
		WHERE m.createdby = $1
		  AND mv.id IS NOT NULL
		ORDER BY m.completedat DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []briefing.MeetingWithMemory
	for rows.Next() {
		var m briefing.MeetingWithMemory
		if err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.CompletedAt,
			&m.ActiveMemoryVersion,
			&m.MemoryJSON,
			&m.ParticipantEmail,
			&m.ParticipantOrg,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	if result == nil {
		result = []briefing.MeetingWithMemory{}
	}
	return result, rows.Err()
}

// GetMeetingParticipants returns the email and organization for all participants
// of a given meeting.
func (a *briefingSourceAdapter) GetMeetingParticipants(
	ctx context.Context,
	meetingID uuid.UUID,
) ([]briefing.Participant_email_org, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT email, organization
		FROM meeting_participants
		WHERE meetingid = $1
		ORDER BY displayorder ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []briefing.Participant_email_org
	for rows.Next() {
		var p briefing.Participant_email_org
		if err := rows.Scan(&p.Email, &p.Organization); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if result == nil {
		result = []briefing.Participant_email_org{}
	}
	return result, rows.Err()
}