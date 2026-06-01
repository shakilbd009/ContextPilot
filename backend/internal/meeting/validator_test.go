package meeting

import (
	"strings"
	"testing"
)

func TestValidate_Title(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "title required",
			input: CreateMeetingInput{
				Title:            "",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "title",
		},
		{
			name: "title max 500 chars",
			input: CreateMeetingInput{
				Title:            strings.Repeat("a", 501),
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "title",
		},
		{
			name: "title valid at 500 chars",
			input: CreateMeetingInput{
				Title:            strings.Repeat("a", 500),
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "title whitespace trimmed valid",
			input: CreateMeetingInput{
				Title:            "  My Meeting  ",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "title empty after trim",
			input: CreateMeetingInput{
				Title:            "   ",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
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

func TestValidate_CompletedAt(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "completedAt required",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:       "",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "completedAt",
		},
		{
			name: "completedAt invalid format",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:       "not-a-date",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "completedAt",
		},
		{
			name: "completedAt valid ISO 8601",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:       "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			}
		})
	}
}

func TestValidate_Participants(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "participants required",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "participants",
		},
		{
			name: "displayName required",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: ""}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "participants[0].displayName",
		},
		{
			name: "displayName whitespace trimmed",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "  Alice  "}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "displayName empty after trim",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "   "}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "participants[0].displayName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			}
		})
	}
}

func TestValidate_Content(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "transcript or notes required",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       nil,
				Notes:            nil,
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "content",
		},
		{
			name: "transcript only",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("meeting transcript"),
				Notes:            nil,
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "notes only",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       nil,
				Notes:            strPtr("meeting notes"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "combined exactly 50000 chars",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr(strings.Repeat("x", 25000)),
				Notes:            strPtr(strings.Repeat("y", 25000)),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "combined 50001 chars blocked",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr(strings.Repeat("x", 25001)),
				Notes:            strPtr(strings.Repeat("y", 25000)),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr:  true,
			errField: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			}
		})
	}
}

func TestValidate_IdempotencyToken(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name: "token required",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "",
			},
			wantErr:  true,
			errField: "idempotencyToken",
		},
		{
			name: "malformed token",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "not-a-uuid",
			},
			wantErr:  true,
			errField: "idempotencyToken",
		},
		{
			name: "valid UUID token",
			input: CreateMeetingInput{
				Title:            "My Meeting",
				CompletedAt:      "2026-05-20T14:00:00Z",
				Participants:     []ParticipantInput{{DisplayName: "Alice"}},
				Transcript:       strPtr("transcript"),
				IdempotencyToken: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.input)
			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, result.Errors)
				}
			}
		})
	}
}

func TestContentSource(t *testing.T) {
	tests := []struct {
		name       string
		transcript *string
		notes      *string
		want       string
	}{
		{"transcript only", strPtr("hello"), nil, "transcript"},
		{"notes only", nil, strPtr("hello"), "notes"},
		{"both", strPtr("a"), strPtr("b"), "both"},
		{"empty transcript", strPtr(""), strPtr("notes"), "notes"},
		{"nil transcript", nil, strPtr("notes"), "notes"},
		{"empty notes", strPtr("transcript"), strPtr(""), "transcript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContentSource(tt.transcript, tt.notes)
			if got != tt.want {
				t.Errorf("ContentSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCombinedContentLen(t *testing.T) {
	tests := []struct {
		name       string
		transcript *string
		notes      *string
		want       int
	}{
		{"both present", strPtr("hello"), strPtr("world"), 10},
		{"transcript only", strPtr("hello"), nil, 5},
		{"notes only", nil, strPtr("world"), 5},
		{"empty strings", strPtr(""), strPtr(""), 0},
		{"nil both", nil, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CombinedContentLen(tt.transcript, tt.notes)
			if got != tt.want {
				t.Errorf("CombinedContentLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestValidateUpdate_Title(t *testing.T) {
	tests := []struct {
		name     string
		input    UpdateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name:     "nil title is valid",
			input:    UpdateMeetingInput{Title: nil},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "valid title",
			input:    UpdateMeetingInput{Title: strPtr("Valid Title")},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "empty title rejected",
			input:    UpdateMeetingInput{Title: strPtr("")},
			wantErr:  true,
			errField: "title",
		},
		{
			name:     "whitespace-only title rejected",
			input:    UpdateMeetingInput{Title: strPtr("   ")},
			wantErr:  true,
			errField: "title",
		},
		{
			name:     "title at max length (500 chars) valid",
			input:    UpdateMeetingInput{Title: strPtr(strings.Repeat("a", 500))},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "title over max length rejected",
			input:    UpdateMeetingInput{Title: strPtr(strings.Repeat("a", 501))},
			wantErr:  true,
			errField: "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateUpdate(tt.input)
			if tt.wantErr {
				if len(errors) == 0 {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, errors)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected errors: %+v", errors)
				}
			}
		})
	}
}

func TestValidateUpdate_CompletedAt(t *testing.T) {
	tests := []struct {
		name     string
		input    UpdateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name:     "nil completedAt is valid",
			input:    UpdateMeetingInput{CompletedAt: nil},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "valid ISO 8601 completedAt",
			input:    UpdateMeetingInput{CompletedAt: strPtr("2026-05-20T14:00:00Z")},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "invalid completedAt format rejected",
			input:    UpdateMeetingInput{CompletedAt: strPtr("not-a-date")},
			wantErr:  true,
			errField: "completedAt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateUpdate(tt.input)
			if tt.wantErr {
				if len(errors) == 0 {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, errors)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected errors: %+v", errors)
				}
			}
		})
	}
}

func TestValidateUpdate_Participants(t *testing.T) {
	tests := []struct {
		name     string
		input    UpdateMeetingInput
		wantErr  bool
		errField string
	}{
		{
			name:     "nil participants is valid",
			input:    UpdateMeetingInput{Participants: nil},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "empty participants list is valid",
			input:    UpdateMeetingInput{Participants: []ParticipantInput{}},
			wantErr:  false,
			errField: "",
		},
		{
			name:     "valid participants within limit",
			input:    UpdateMeetingInput{Participants: []ParticipantInput{{DisplayName: "Alice"}, {DisplayName: "Bob"}}},
			wantErr:  false,
			errField: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateUpdate(tt.input)
			if tt.wantErr {
				if len(errors) == 0 {
					t.Fatalf("expected errors, got none")
				}
				found := false
				for _, e := range errors {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error field %q, got errors: %+v", tt.errField, errors)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected errors: %+v", errors)
				}
			}
		})
	}
}