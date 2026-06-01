package upcoming

import (
	"time"

	"github.com/google/uuid"
)

// ─── Status ─────────────────────────────────────────────────────────────────

// Status represents the lifecycle state of an upcoming meeting.
type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusCancelled Status = "cancelled"
)

// ─── Core Models ─────────────────────────────────────────────────────────────

// UpcomingMeeting represents a manually created upcoming meeting (BRD-05 FR-5).
type UpcomingMeeting struct {
	ID                   uuid.UUID                    `json:"id"`
	Title                string                       `json:"title"`
	ScheduledStart       time.Time                    `json:"scheduledStart"`
	Description          *string                      `json:"description,omitempty"`
	ClientOrOrganization *string                      `json:"clientOrOrganization,omitempty"`
	Status               Status                       `json:"status"`
	CreatedBy            uuid.UUID                    `json:"createdBy"`
	Participants         []UpcomingMeetingParticipant `json:"participants,omitempty"`
	BriefingStatus       *string                      `json:"briefingStatus,omitempty"`
	IsBriefingStale      bool                         `json:"isBriefingStale"`
	Editable             bool                         `json:"editable"`
	Cancellable          bool                         `json:"cancellable"`
	CreatedAt            time.Time                    `json:"createdAt"`
	UpdatedAt            time.Time                    `json:"updatedAt"`
}

// GetScheduledStart implements ScheduledStartProvider.
func (m *UpcomingMeeting) GetScheduledStart() time.Time {
	return m.ScheduledStart
}

// UpcomingMeetingParticipant represents a structured participant (BRD-05 FR-6).
type UpcomingMeetingParticipant struct {
	ID           uuid.UUID `json:"id"`
	DisplayName  *string   `json:"displayName,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Organization *string   `json:"organization,omitempty"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

// ─── Request Bodies ─────────────────────────────────────────────────────────

// CreateUpcomingMeetingInput is the request body for POST /upcoming.
type CreateUpcomingMeetingInput struct {
	Title                string             `json:"title"`
	ScheduledStart       string             `json:"scheduledStart"`
	Description         *string            `json:"description,omitempty"`
	ClientOrOrganization *string            `json:"clientOrOrganization,omitempty"`
	Participants        []ParticipantInput `json:"participants,omitempty"`
}

// ParticipantInput is a participant from the create/update request body.
type ParticipantInput struct {
	DisplayName  *string `json:"displayName,omitempty"`
	Email        *string `json:"email,omitempty"`
	Organization *string `json:"organization,omitempty"`
}

// UpdateUpcomingMeetingInput is the request body for PATCH /upcoming/{id}.
type UpdateUpcomingMeetingInput struct {
	Title                *string            `json:"title,omitempty"`
	ScheduledStart       *string            `json:"scheduledStart,omitempty"`
	Description         *string            `json:"description,omitempty"`
	ClientOrOrganization *string            `json:"clientOrOrganization,omitempty"`
	Participants        []ParticipantInput `json:"participants,omitempty"`
}

// ─── Response Bodies ─────────────────────────────────────────────────────────

// CreateUpcomingMeetingOutput is the 201 response body.
type CreateUpcomingMeetingOutput struct {
	ID       string `json:"id"`
	Redirect string `json:"redirect"`
}

// FieldError represents a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationErrorBody is the 400 response body.
type ValidationErrorBody struct {
	Errors []FieldError   `json:"errors"`
	Values map[string]any `json:"values,omitempty"`
}

// UpdateSuccessBody is the 200 response body for PATCH /upcoming/{id}.
type UpdateSuccessBody struct {
	UpcomingMeeting *UpcomingMeeting `json:"upcomingMeeting"`
	MeaningfulEdit  bool             `json:"meaningfulEdit"`
}