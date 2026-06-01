package meeting

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxTitleLen     = 500
	maxContentLen   = 50000
	warnContentLen  = 45000
	participantLimit = 100 // sanity limit
)

// ValidationResult holds all field errors and submitted values for form re-render.
type ValidationResult struct {
	Errors []FieldError
	Values map[string]any
}

// Validate validates CreateMeetingInput and returns field errors with submitted values.
func Validate(in CreateMeetingInput) ValidationResult {
	result := ValidationResult{
		Errors: []FieldError{},
		Values: make(map[string]any),
	}

	// Store all submitted values for form re-render (BRD-02 AC-15)
	result.Values["title"] = in.Title
	result.Values["completedAt"] = in.CompletedAt
	result.Values["participants"] = in.Participants
	result.Values["transcript"] = in.Transcript
	result.Values["notes"] = in.Notes
	result.Values["idempotencyToken"] = in.IdempotencyToken

	// Title: required, max 500 chars, trimmed
	title := strings.TrimSpace(in.Title)
	if title == "" {
		result.Errors = append(result.Errors, FieldError{
			Field:   "title",
			Message: "Title is required and must be at most 500 characters.",
		})
	} else if utf8.RuneCountInString(title) > maxTitleLen {
		result.Errors = append(result.Errors, FieldError{
			Field:   "title",
			Message: "Title must be at most 500 characters.",
		})
	}

	// completedAt: required, valid ISO 8601
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(in.CompletedAt))
	if err != nil {
		result.Errors = append(result.Errors, FieldError{
			Field:   "completedAt",
			Message: "Completed date/time is required and must be a valid ISO 8601 timestamp.",
		})
	}

	// Participants: required, at least one
	if len(in.Participants) == 0 {
		result.Errors = append(result.Errors, FieldError{
			Field:   "participants",
			Message: "At least one participant is required.",
		})
	} else if len(in.Participants) > participantLimit {
		result.Errors = append(result.Errors, FieldError{
			Field:   "participants",
			Message: fmt.Sprintf("Too many participants (max %d).", participantLimit),
		})
	} else {
		// Validate each participant's displayName
		for i, p := range in.Participants {
			dn := strings.TrimSpace(p.DisplayName)
			if dn == "" {
				result.Errors = append(result.Errors, FieldError{
					Field:   fmt.Sprintf("participants[%d].displayName", i),
					Message: "Participant display name is required.",
				})
			}
		}
	}

	// Content: transcript or notes (at least one non-empty)
	transcriptLen := 0
	notesLen := 0
	if in.Transcript != nil {
		transcriptLen = utf8.RuneCountInString(*in.Transcript)
	}
	if in.Notes != nil {
		notesLen = utf8.RuneCountInString(*in.Notes)
	}
	if transcriptLen == 0 && notesLen == 0 {
		result.Errors = append(result.Errors, FieldError{
			Field:   "content",
			Message: "Either transcript or notes (or both) is required.",
		})
	}

	// Combined character limit (50,000)
	if transcriptLen+notesLen > maxContentLen {
		result.Errors = append(result.Errors, FieldError{
			Field:   "content",
			Message: fmt.Sprintf("Combined transcript and notes must be at most %d characters.", maxContentLen),
		})
	}

	// Idempotency token: required, valid UUID
	token := strings.TrimSpace(in.IdempotencyToken)
	if token == "" {
		result.Errors = append(result.Errors, FieldError{
			Field:   "idempotencyToken",
			Message: "Idempotency token is required.",
		})
	} else if _, err := uuid.Parse(token); err != nil {
		result.Errors = append(result.Errors, FieldError{
			Field:   "idempotencyToken",
			Message: "Idempotency token must be a valid UUID.",
		})
	}

	return result
}

// HasErrors returns true if any validation errors were found.
func (v ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

// ValidateUpdate validates an UpdateMeetingInput.
// Returns field errors; empty errors means the input is valid.
func ValidateUpdate(in UpdateMeetingInput) []FieldError {
	var errors []FieldError

	// Title: optional, but if present must be non-empty and max 500 chars
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			errors = append(errors, FieldError{
				Field:   "title",
				Message: "Title cannot be empty.",
			})
		} else if utf8.RuneCountInString(title) > maxTitleLen {
			errors = append(errors, FieldError{
				Field:   "title",
				Message: "Title must be at most 500 characters.",
			})
		}
	}

	// completedAt: optional, but if present must be valid ISO 8601
	if in.CompletedAt != nil {
		_, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.CompletedAt))
		if err != nil {
			errors = append(errors, FieldError{
				Field:   "completedAt",
				Message: "Completed date/time must be a valid ISO 8601 timestamp.",
			})
		}
	}

	// Participants: optional, but if present must not exceed limit
	if in.Participants != nil && len(in.Participants) > participantLimit {
		errors = append(errors, FieldError{
			Field:   "participants",
			Message: fmt.Sprintf("Too many participants (max %d).", participantLimit),
		})
	}

	return errors
}

// ParseCompletedAt parses and returns the completedAt time.Time.
func ParseCompletedAt(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(s))
}

// ContentSource derives the contentSource string from transcript/notes presence.
// Returns "transcript", "notes", or "both".
func ContentSource(transcript, notes *string) string {
	hasTranscript := transcript != nil && *transcript != ""
	hasNotes := notes != nil && *notes != ""
	if hasTranscript && hasNotes {
		return "both"
	}
	if hasTranscript {
		return "transcript"
	}
	return "notes"
}

// TrimString returns nil if s is empty after trim, otherwise returns trimmed string.
func TrimString(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// CombinedContentLen returns the combined rune count of transcript + notes.
func CombinedContentLen(transcript, notes *string) int {
	var total int
	if transcript != nil {
		total += utf8.RuneCountInString(*transcript)
	}
	if notes != nil {
		total += utf8.RuneCountInString(*notes)
	}
	return total
}