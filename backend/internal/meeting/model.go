package meeting

import (
	"time"

	"github.com/google/uuid"
)

// Meeting represents a completed meeting record (BRD-02).
type Meeting struct {
	ID            uuid.UUID     `json:"id"`
	Title         string        `json:"title"`
	CompletedAt   time.Time     `json:"completedAt"`
	Transcript    *string       `json:"transcript,omitempty"`
	Notes         *string       `json:"notes,omitempty"`
	ContentSource string        `json:"contentSource"`
	CreatedAt     time.Time     `json:"createdAt"`
	CreatedBy     uuid.UUID     `json:"createdBy"`
	DisplayOrder  int           `json:"displayOrder"`
	Participants  []Participant `json:"meeting_participants"`
}

// Participant represents a structured meeting participant (BRD-02).
type Participant struct {
	ID           uuid.UUID `json:"id"`
	MeetingID    uuid.UUID `json:"meetingId,omitempty"`
	DisplayName  string    `json:"displayName"`
	Email        *string   `json:"email,omitempty"`
	Organization *string   `json:"organization,omitempty"`
	Role         *string   `json:"role,omitempty"`
	DisplayOrder int       `json:"displayOrder"`
}

// IdempotencyToken stores consumed tokens with 24h TTL.
type IdempotencyToken struct {
	Token     uuid.UUID `json:"token"`
	MeetingID uuid.UUID `json:"meetingId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// CreateMeetingInput is the request body for POST /meetings.
type CreateMeetingInput struct {
	Title            string             `json:"title"`
	CompletedAt      string             `json:"completedAt"`
	Participants     []ParticipantInput `json:"participants"`
	Transcript       *string            `json:"transcript,omitempty"`
	Notes            *string            `json:"notes,omitempty"`
	IdempotencyToken string             `json:"idempotencyToken"`
}

// ParticipantInput is a participant from the request body.
type ParticipantInput struct {
	DisplayName  string  `json:"displayName"`
	Email        *string `json:"email,omitempty"`
	Organization *string `json:"organization,omitempty"`
	Role         *string `json:"role,omitempty"`
}

// CreateMeetingOutput is the 201 response body.
type CreateMeetingOutput struct {
	ID       string `json:"id"`
	Redirect string `json:"redirect"`
}

// FieldError represents a single field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorBody is the 400 response body.
type ValidationErrorBody struct {
	Errors []FieldError `json:"errors"`
	Values map[string]any `json:"values"`
}

// DuplicateErrorBody is the 409 response body.
type DuplicateErrorBody struct {
	Error          string `json:"error"`
	MeetingID      string `json:"meetingId"`
	OriginalOutcome string `json:"originalOutcome"`
}

// UpdateMeetingInput is the request body for PATCH /meetings/{id}.
type UpdateMeetingInput struct {
	Title        *string            `json:"title,omitempty"`
	CompletedAt  *string            `json:"completedAt,omitempty"`
	Transcript   *string            `json:"transcript,omitempty"`
	Notes        *string            `json:"notes,omitempty"`
	Participants []ParticipantInput `json:"participants,omitempty"`
}

// InternalErrorBody is the 500 response body.
type InternalErrorBody struct {
	Error         string `json:"error"`
	CorrelationID string `json:"correlationId,omitempty"`
}