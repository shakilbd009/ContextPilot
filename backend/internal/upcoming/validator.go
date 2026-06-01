package upcoming

import (
	"net/mail"
	"strings"
	"time"
)

// ValidationResult holds validation errors and echoed form values.
type ValidationResult struct {
	Errors []FieldError
	Values map[string]any
}

// HasErrors returns true if any validation errors were collected.
func (v ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

// ValidateCreate validates a POST /upcoming request body. The caller passes
// the reference "now" so that the 15-minute scheduling floor is deterministic
// and testable (boundary checks at exactly T+15min do not depend on the gap
// between two time.Now() calls in the test and the validator).
func ValidateCreate(in CreateUpcomingMeetingInput, now time.Time) ValidationResult {
	result := ValidationResult{
		Values: echoValues(in),
	}

	// Title required (FR-3, AC-02)
	title := strings.TrimSpace(in.Title)
	if title == "" {
		result.Errors = append(result.Errors, FieldError{
			Field:   "title",
			Message: "Title is required.",
			Code:    "missing_title",
		})
	}

	// scheduled_start required + 15-minute floor (FR-8, AC-05)
	if in.ScheduledStart == "" {
		result.Errors = append(result.Errors, FieldError{
			Field:   "scheduledStart",
			Message: "Scheduled start time is required and must be at least 15 minutes in the future.",
			Code:    "invalid_scheduled_start",
		})
	} else {
		scheduled, err := time.Parse(time.RFC3339, strings.TrimSpace(in.ScheduledStart))
		if err != nil {
			result.Errors = append(result.Errors, FieldError{
				Field:   "scheduledStart",
				Message: "Scheduled start must be a valid ISO 8601 timestamp.",
				Code:    "invalid_scheduled_start",
			})
		} else {
			// 15-minute floor: scheduled_start must be >= now + 15 minutes.
			// `now` is provided by the caller so tests can pin the reference
			// instant and avoid time-of-day drift between the two reads.
			floor := now.Add(15 * time.Minute)
			if scheduled.Before(floor) {
				result.Errors = append(result.Errors, FieldError{
					Field:   "scheduledStart",
					Message: "Scheduled start must be at least 15 minutes in the future.",
					Code:    "invalid_scheduled_start",
				})
			}
		}
	}

	// Participants validation (FR-6, AC-04, FR-26)
	if len(in.Participants) > 50 {
		result.Errors = append(result.Errors, FieldError{
			Field:   "participants",
			Message: "Maximum 50 participants allowed.",
			Code:    "too_many_participants",
		})
	}
	for i, p := range in.Participants {
		hasName := strings.TrimSpace(nullString(p.DisplayName)) != ""
		hasEmail := strings.TrimSpace(nullString(p.Email)) != ""
		if !hasName && !hasEmail {
			result.Errors = append(result.Errors, FieldError{
				Field:   "participants",
				Message: "Each participant must have at least a display name or email.",
				Code:    "participant_missing_identity",
			})
		}
		// Email format validation if provided
		if hasEmail {
			email := strings.TrimSpace(nullString(p.Email))
			if _, err := mail.ParseAddress(email); err != nil {
				result.Errors = append(result.Errors, FieldError{
					Field:   "participants",
					Message: "Invalid email format for participant.",
					Code:    "invalid_email",
				})
			}
		}
		_ = i // suppress unused warning
	}

	return result
}

// ValidateUpdate validates a PATCH /upcoming/{id} request body.
func ValidateUpdate(in UpdateUpcomingMeetingInput, meeting ScheduledStartProvider, now time.Time) ValidationResult {
	result := ValidationResult{
		Values: echoUpdateValues(in),
	}

	// Title optional but if provided must not be empty
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			result.Errors = append(result.Errors, FieldError{
				Field:   "title",
				Message: "Title cannot be empty if provided.",
				Code:    "missing_title",
			})
		}
	}

	// scheduled_start validation if provided — must be >= 15 min from now (not meeting's stored time for edit)
	if in.ScheduledStart != nil {
		schedStr := strings.TrimSpace(*in.ScheduledStart)
		if schedStr != "" {
			scheduled, err := time.Parse(time.RFC3339, schedStr)
			if err != nil {
				result.Errors = append(result.Errors, FieldError{
					Field:   "scheduledStart",
					Message: "Scheduled start must be a valid ISO 8601 timestamp.",
					Code:    "invalid_scheduled_start",
				})
			} else {
				floor := now.Add(15 * time.Minute)
				if scheduled.Before(floor) {
					result.Errors = append(result.Errors, FieldError{
						Field:   "scheduledStart",
						Message: "Scheduled start must be at least 15 minutes in the future.",
						Code:    "invalid_scheduled_start",
					})
				}
			}
		}
	}

	// Participants: if provided, validate same rules
	if len(in.Participants) > 50 {
		result.Errors = append(result.Errors, FieldError{
			Field:   "participants",
			Message: "Maximum 50 participants allowed.",
			Code:    "too_many_participants",
		})
	}
	for i, p := range in.Participants {
		hasName := strings.TrimSpace(nullString(p.DisplayName)) != ""
		hasEmail := strings.TrimSpace(nullString(p.Email)) != ""
		if !hasName && !hasEmail {
			result.Errors = append(result.Errors, FieldError{
				Field:   "participants",
				Message: "Each participant must have at least a display name or email.",
				Code:    "participant_missing_identity",
			})
		}
		if hasEmail {
			email := strings.TrimSpace(nullString(p.Email))
			if _, err := mail.ParseAddress(email); err != nil {
				result.Errors = append(result.Errors, FieldError{
					Field:   "participants",
					Message: "Invalid email format for participant.",
					Code:    "invalid_email",
				})
			}
		}
		_ = i
	}

	return result
}

// ScheduledStartProvider abstracts access to the meeting's scheduled_start.
type ScheduledStartProvider interface {
	GetScheduledStart() time.Time
}

// EditWindowExpired returns true when the edit window has passed (15 min after scheduled_start).
func EditWindowExpired(scheduledStart time.Time, now time.Time) bool {
	cutoff := scheduledStart.Add(15 * time.Minute)
	return now.After(cutoff)
}

// CancelWindowExpired returns true when the cancel window has passed.
func CancelWindowExpired(scheduledStart time.Time, now time.Time) bool {
	cutoff := scheduledStart.Add(15 * time.Minute)
	return now.After(cutoff)
}

// IsMeaningfulEdit returns true when the update changed title, scheduled_start,
// description, client_or_organization, or any participant field (FR-16).
func IsMeaningfulEdit(before *UpcomingMeeting, after UpdateUpcomingMeetingInput) bool {
	// Title changed
	if after.Title != nil && strings.TrimSpace(*after.Title) != before.Title {
		return true
	}
	// scheduled_start changed
	if after.ScheduledStart != nil {
		afterStr := strings.TrimSpace(*after.ScheduledStart)
		if afterStr != "" {
			afterTime, err := time.Parse(time.RFC3339, afterStr)
			if err == nil && !afterTime.Equal(before.ScheduledStart) {
				return true
			}
		}
	}
	// Description changed
	if after.Description != nil {
		beforeDesc := ""
		if before.Description != nil {
			beforeDesc = *before.Description
		}
		if strings.TrimSpace(nullString(after.Description)) != strings.TrimSpace(beforeDesc) {
			return true
		}
	}
	// Client/org changed
	if after.ClientOrOrganization != nil {
		beforeOrg := ""
		if before.ClientOrOrganization != nil {
			beforeOrg = *before.ClientOrOrganization
		}
		if strings.TrimSpace(nullString(after.ClientOrOrganization)) != strings.TrimSpace(beforeOrg) {
			return true
		}
	}
	// Participants changed — any addition/removal counts as meaningful
	if after.Participants != nil {
		if len(after.Participants) != len(before.Participants) {
			return true
		}
		// Compare each participant's identity fields
		for i, ap := range after.Participants {
			if i >= len(before.Participants) {
				return true
			}
			bp := before.Participants[i]
			if nullString(ap.DisplayName) != nullString(bp.DisplayName) {
				return true
			}
			if nullString(ap.Email) != nullString(bp.Email) {
				return true
			}
			if nullString(ap.Organization) != nullString(bp.Organization) {
				return true
			}
		}
	}
	return false
}

// echoValues returns the submitted form values for preservation on error.
func echoValues(in CreateUpcomingMeetingInput) map[string]any {
	m := map[string]any{
		"title":           in.Title,
		"scheduledStart":  in.ScheduledStart,
		"description":     in.Description,
		"clientOrOrganization": in.ClientOrOrganization,
	}
	if len(in.Participants) > 0 {
		m["participants"] = in.Participants
	}
	return m
}

// echoUpdateValues returns the submitted form values for PATCH echo.
func echoUpdateValues(in UpdateUpcomingMeetingInput) map[string]any {
	m := map[string]any{}
	if in.Title != nil {
		m["title"] = *in.Title
	}
	if in.ScheduledStart != nil {
		m["scheduledStart"] = *in.ScheduledStart
	}
	if in.Description != nil {
		m["description"] = *in.Description
	}
	if in.ClientOrOrganization != nil {
		m["clientOrOrganization"] = *in.ClientOrOrganization
	}
	if in.Participants != nil {
		m["participants"] = in.Participants
	}
	return m
}