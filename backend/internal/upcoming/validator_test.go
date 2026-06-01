package upcoming

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr(s string) *string { return &s }

func TestValidateCreate_Title(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateUpcomingMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "missing title",
			input: CreateUpcomingMeetingInput{
				Title:          "",
				ScheduledStart: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
			},
			wantErr:  true,
			errField: "title",
		},
		{
			name: "whitespace-only title",
			input: CreateUpcomingMeetingInput{
				Title:          "   ",
				ScheduledStart: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
			},
			wantErr:  true,
			errField: "title",
		},
		{
			name: "valid title",
			input: CreateUpcomingMeetingInput{
				Title:          "Q3 Planning",
				ScheduledStart: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCreate(tt.input, time.Now())
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						if !strings.Contains(e.Code, "missing_title") {
							t.Errorf("expected missing_title code, got %q", e.Code)
						}
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("unexpected errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidateCreate_ScheduledStart(t *testing.T) {
	// Truncate to second precision: the validator parses ScheduledStart with
	// time.RFC3339, which discards sub-second components. Pinning `now` to
	// the same precision prevents the validator's "now" from being a few
	// nanoseconds later than the formatted input's reference, which would
	// make the exactly-15-minutes case spuriously fail.
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name     string
		input    CreateUpcomingMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "exactly 15 minutes from now",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: now.Add(15 * time.Minute).Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "16 minutes from now",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: now.Add(16 * time.Minute).Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "14 minutes from now",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: now.Add(14 * time.Minute).Format(time.RFC3339),
			},
			wantErr:  true,
			errField: "scheduledStart",
		},
		{
			name: "1 minute from now",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: now.Add(1 * time.Minute).Format(time.RFC3339),
			},
			wantErr:  true,
			errField: "scheduledStart",
		},
		{
			name: "in the past",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			wantErr:  true,
			errField: "scheduledStart",
		},
		{
			name: "empty scheduled_start",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: "",
			},
			wantErr:  true,
			errField: "scheduledStart",
		},
		{
			name: "invalid format",
			input: CreateUpcomingMeetingInput{
				Title:          "Valid Meeting",
				ScheduledStart: "not-a-date",
			},
			wantErr:  true,
			errField: "scheduledStart",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the `now` captured at table construction so the 15-minute
			// boundary check is deterministic. Without this, the validator's
			// own time.Now() call would drift past `now + 15min` and the
			// exactly-15-minutes case would fail intermittently.
			result := ValidateCreate(tt.input, now)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						if !strings.Contains(e.Code, "invalid_scheduled_start") {
							t.Errorf("expected invalid_scheduled_start code, got %q", e.Code)
						}
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("unexpected errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidateCreate_Participants(t *testing.T) {
	future := time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	validEmail := "alice@example.com"
	invalidEmail := "aliceexample.com"

	tests := []struct {
		name     string
		input    CreateUpcomingMeetingInput
		wantErr  bool
		errField string
		errCode  string
	}{
		{
			name: "display_name only is valid",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("Alice")}},
			},
			wantErr: false,
		},
		{
			name: "email only is valid",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{Email: ptr(validEmail)}},
			},
			wantErr: false,
		},
		{
			name: "both name and email is valid",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("Alice"), Email: ptr(validEmail)}},
			},
			wantErr: false,
		},
		{
			name: "neither name nor email",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr(""), Email: ptr("")}},
			},
			wantErr:  true,
			errField: "participants",
			errCode:  "participant_missing_identity",
		},
		{
			name: "whitespace only name with valid email",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("   "), Email: ptr(validEmail)}},
			},
			wantErr: false,
		},
		{
			name: "invalid email format missing @",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("Alice"), Email: ptr(invalidEmail)}},
			},
			wantErr:  true,
			errField: "participants",
			errCode:  "invalid_email",
		},
		{
			name: "invalid email format missing domain",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("Alice"), Email: ptr("alice@")}},
			},
			wantErr:  true,
			errField: "participants",
			errCode:  "invalid_email",
		},
		{
			name: "null email is optional",
			input: CreateUpcomingMeetingInput{
				Title:          "Meeting",
				ScheduledStart: future,
				Participants:   []ParticipantInput{{DisplayName: ptr("Alice"), Email: nil}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCreate(tt.input, time.Now())
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField && e.Code == tt.errCode {
						found = true
					}
				}
				if !found {
					t.Errorf("expected error field=%q code=%q, got errors: %+v", tt.errField, tt.errCode, result.Errors)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("unexpected errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestValidateCreate_TooManyParticipants(t *testing.T) {
	future := time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	participants := make([]ParticipantInput, 51)
	for i := range participants {
		participants[i] = ParticipantInput{DisplayName: ptr("Participant")}
	}

	result := ValidateCreate(CreateUpcomingMeetingInput{
		Title:          "Meeting",
		ScheduledStart: future,
		Participants:   participants,
	}, time.Now())

	if !result.HasErrors() {
		t.Fatal("expected too_many_participants error, got none")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "too_many_participants" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected too_many_participants error, got: %+v", result.Errors)
	}
}

func TestValidateCreate_FormEcho(t *testing.T) {
	future := time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	in := CreateUpcomingMeetingInput{
		Title:                "",
		ScheduledStart:       future,
		Description:         ptr("my description"),
		ClientOrOrganization: ptr("Acme"),
		Participants:         []ParticipantInput{{DisplayName: ptr("Alice"), Email: ptr("alice@example.com")}},
	}
	result := ValidateCreate(in, time.Now())
	if !result.HasErrors() {
		t.Fatal("expected validation errors")
	}
	if result.Values == nil {
		t.Fatal("expected form values to be echoed")
	}
	if result.Values["scheduledStart"] != future {
		t.Errorf("echoed scheduledStart = %q, want %q", result.Values["scheduledStart"], future)
	}
}

func TestEditWindowExpired(t *testing.T) {
	scheduled := time.Now()
	tests := []struct {
		name        string
		offsetMins  int
		wantExpired bool
	}{
		{"at scheduled start", 0, false},
		{"14 minutes after", 14, false},
		{"exactly 16 minutes after", 16, true},
		{"1 hour after", 60, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := scheduled.Add(time.Duration(tt.offsetMins) * time.Minute)
			got := EditWindowExpired(scheduled, now)
			if got != tt.wantExpired {
				t.Errorf("EditWindowExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestCancelWindowExpired(t *testing.T) {
	scheduled := time.Now()
	tests := []struct {
		name        string
		offsetMins  int
		wantExpired bool
	}{
		{"at scheduled start", 0, false},
		{"14 minutes after", 14, false},
		{"exactly 16 minutes after", 16, true},
		{"1 hour after", 60, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := scheduled.Add(time.Duration(tt.offsetMins) * time.Minute)
			got := CancelWindowExpired(scheduled, now)
			if got != tt.wantExpired {
				t.Errorf("CancelWindowExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestIsMeaningfulEdit(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	existing := &UpcomingMeeting{
		ID:                   uuid.New(),
		Title:                "Q3 Planning",
		ScheduledStart:       future,
		Description:         ptr("old desc"),
		ClientOrOrganization: ptr("Acme"),
		Status:               StatusScheduled,
		Participants: []UpcomingMeetingParticipant{
			{ID: uuid.New(), DisplayName: ptr("Alice"), Email: ptr("alice@example.com"), Organization: ptr("Acme")},
		},
	}

	tests := []struct {
		name   string
		input  UpdateUpcomingMeetingInput
		before *UpcomingMeeting
		want   bool
	}{
		{
			name:   "change title",
			input:  UpdateUpcomingMeetingInput{Title: ptr("Q3 Planning Review")},
			before: existing,
			want:   true,
		},
		{
			name:   "change scheduled_start",
			input:  UpdateUpcomingMeetingInput{ScheduledStart: ptr(future.Add(1 * time.Hour).Format(time.RFC3339))},
			before: existing,
			want:   true,
		},
		{
			name:   "change description",
			input:  UpdateUpcomingMeetingInput{Description: ptr("new desc")},
			before: existing,
			want:   true,
		},
		{
			name:   "change client_or_organization",
			input:  UpdateUpcomingMeetingInput{ClientOrOrganization: ptr("Beta")},
			before: existing,
			want:   true,
		},
		{
			name: "add participant row",
			input: UpdateUpcomingMeetingInput{
				Participants: []ParticipantInput{
					{DisplayName: ptr("Alice"), Email: ptr("alice@example.com")},
					{DisplayName: ptr("Bob"), Email: ptr("bob@example.com")},
				},
			},
			before: existing,
			want:   true,
		},
		{
			name: "remove participant row",
			input: UpdateUpcomingMeetingInput{
				Participants: []ParticipantInput{},
			},
			before: existing,
			want:   true,
		},
		{
			name: "change participant display_name",
			input: UpdateUpcomingMeetingInput{
				Participants: []ParticipantInput{
					{DisplayName: ptr("Alice Smith"), Email: ptr("alice@example.com"), Organization: ptr("Acme")},
				},
			},
			before: existing,
			want:   true,
		},
		{
			name: "change participant email",
			input: UpdateUpcomingMeetingInput{
				Participants: []ParticipantInput{
					{DisplayName: ptr("Alice"), Email: ptr("alice.smith@example.com"), Organization: ptr("Acme")},
				},
			},
			before: existing,
			want:   true,
		},
		{
			name: "change participant organization",
			input: UpdateUpcomingMeetingInput{
				Participants: []ParticipantInput{
					{DisplayName: ptr("Alice"), Email: ptr("alice@example.com"), Organization: ptr("Beta")},
				},
			},
			before: existing,
			want:   true,
		},
		{
			name:   "no field changed",
			input:  UpdateUpcomingMeetingInput{},
			before: existing,
			want:   false,
		},
		{
			name:   "identical title",
			input:  UpdateUpcomingMeetingInput{Title: ptr("Q3 Planning")},
			before: existing,
			want:   false,
		},
		{
			name:   "identical description",
			input:  UpdateUpcomingMeetingInput{Description: ptr("old desc")},
			before: existing,
			want:   false,
		},
		{
			name:   "identical client_or_organization",
			input:  UpdateUpcomingMeetingInput{ClientOrOrganization: ptr("Acme")},
			before: existing,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMeaningfulEdit(tt.before, tt.input)
			if got != tt.want {
				t.Errorf("IsMeaningfulEdit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateUpdate_Title(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	meeting := &UpcomingMeeting{ID: uuid.New(), Title: "Original", ScheduledStart: future, Status: StatusScheduled}

	tests := []struct {
		name     string
		input    UpdateUpcomingMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name:  "nil title is valid",
			input: UpdateUpcomingMeetingInput{Title: nil},
		},
		{
			name:     "empty title rejected",
			input:    UpdateUpcomingMeetingInput{Title: ptr("")},
			wantErr:  true,
			errField: "title",
		},
		{
			name:     "whitespace-only title rejected",
			input:    UpdateUpcomingMeetingInput{Title: ptr("   ")},
			wantErr:  true,
			errField: "title",
		},
		{
			name:  "valid title",
			input: UpdateUpcomingMeetingInput{Title: ptr("Updated Title")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUpdate(tt.input, meeting, time.Now())
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("unexpected errors: %+v", result.Errors)
				}
			}
		})
	}
}

func TestStatusValues(t *testing.T) {
	if StatusScheduled != "scheduled" {
		t.Errorf("StatusScheduled = %q, want %q", StatusScheduled, "scheduled")
	}
	if StatusCancelled != "cancelled" {
		t.Errorf("StatusCancelled = %q, want %q", StatusCancelled, "cancelled")
	}
}

func TestUpcomingMeeting_GetScheduledStart(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	m := &UpcomingMeeting{ScheduledStart: future}
	if got := m.GetScheduledStart(); !got.Equal(future) {
		t.Errorf("GetScheduledStart() = %v, want %v", got, future)
	}
}