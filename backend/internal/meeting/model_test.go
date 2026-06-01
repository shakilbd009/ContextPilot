package meeting

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMeeting_JSON(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	transcript := "Hello world"
	notes := "Action items"
	createdBy := uuid.New()

	meeting := Meeting{
		ID:            id,
		Title:         "Test Meeting",
		CompletedAt:   now,
		Transcript:    &transcript,
		Notes:         &notes,
		ContentSource: "both",
		CreatedAt:     now,
		CreatedBy:     createdBy,
		DisplayOrder:  0,
		Participants: []Participant{
			{
				ID:           uuid.New(),
				MeetingID:    id,
				DisplayName:  "Alice",
				Email:        strPtr("alice@example.com"),
				Organization: strPtr("Acme"),
				Role:         strPtr("host"),
				DisplayOrder: 0,
			},
		},
	}

	data, err := json.Marshal(meeting)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got Meeting
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
	if got.Title != meeting.Title {
		t.Errorf("Title = %v, want %v", got.Title, meeting.Title)
	}
	if got.ContentSource != meeting.ContentSource {
		t.Errorf("ContentSource = %v, want %v", got.ContentSource, meeting.ContentSource)
	}
	if got.DisplayOrder != meeting.DisplayOrder {
		t.Errorf("DisplayOrder = %v, want %v", got.DisplayOrder, meeting.DisplayOrder)
	}
	if len(got.Participants) != 1 {
		t.Fatalf("len(Participants) = %d, want 1", len(got.Participants))
	}
	if got.Participants[0].DisplayName != "Alice" {
		t.Errorf("Participants[0].DisplayName = %v, want Alice", got.Participants[0].DisplayName)
	}
}

func TestParticipant_JSONTags(t *testing.T) {
	id := uuid.New()
	email := "bob@example.com"
	org := "Org"
	role := "attendee"

	p := Participant{
		ID:           id,
		DisplayName:  "Bob",
		Email:        &email,
		Organization: &org,
		Role:         &role,
		DisplayOrder: 1,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Check JSON field names
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map error: %v", err)
	}

	wantFields := []string{"id", "meetingId", "displayName", "email", "organization", "role", "displayOrder"}
	for _, f := range wantFields {
		if _, ok := m[f]; !ok {
			t.Errorf("missing JSON field %q", f)
		}
	}
}

func TestCreateMeetingInput_JSON(t *testing.T) {
	transcript := "Transcript text"
	in := CreateMeetingInput{
		Title:            "Planning",
		CompletedAt:      "2024-01-15T10:00:00Z",
		Participants:     []ParticipantInput{{DisplayName: "Carol", Email: strPtr("carol@example.com")}},
		Transcript:       &transcript,
		Notes:            nil,
		IdempotencyToken: uuid.New().String(),
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got CreateMeetingInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Title != in.Title {
		t.Errorf("Title = %v, want %v", got.Title, in.Title)
	}
	if got.CompletedAt != in.CompletedAt {
		t.Errorf("CompletedAt = %v, want %v", got.CompletedAt, in.CompletedAt)
	}
	if len(got.Participants) != 1 {
		t.Fatalf("len(Participants) = %d, want 1", len(got.Participants))
	}
	if *got.Transcript != transcript {
		t.Errorf("Transcript = %v, want %v", *got.Transcript, transcript)
	}
	if got.Notes != nil {
		t.Errorf("Notes = %v, want nil", got.Notes)
	}
}

func TestParticipantInput_JSONTags(t *testing.T) {
	email := "dave@example.com"
	org := "Acme Corp"
	role := "engineer"
	p := ParticipantInput{
		DisplayName:  "Dave",
		Email:        &email,
		Organization: &org,
		Role:         &role,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	wantFields := []string{"displayName", "email", "organization", "role"}
	for _, f := range wantFields {
		if _, ok := m[f]; !ok {
			t.Errorf("missing JSON field %q", f)
		}
	}
}

func TestCreateMeetingOutput_JSON(t *testing.T) {
	id := uuid.New()
	out := CreateMeetingOutput{
		ID:       id.String(),
		Redirect: "/meetings/" + id.String(),
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got CreateMeetingOutput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != out.ID {
		t.Errorf("ID = %v, want %v", got.ID, out.ID)
	}
	if got.Redirect != out.Redirect {
		t.Errorf("Redirect = %v, want %v", got.Redirect, out.Redirect)
	}
}

func TestFieldError_JSON(t *testing.T) {
	fe := FieldError{
		Field:   "title",
		Message: "Title is required.",
	}

	data, err := json.Marshal(fe)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got FieldError
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Field != fe.Field {
		t.Errorf("Field = %v, want %v", got.Field, fe.Field)
	}
	if got.Message != fe.Message {
		t.Errorf("Message = %v, want %v", got.Message, fe.Message)
	}
}

func TestValidationErrorBody_JSON(t *testing.T) {
	body := ValidationErrorBody{
		Errors: []FieldError{
			{Field: "title", Message: "required"},
			{Field: "completedAt", Message: "invalid format"},
		},
		Values: map[string]any{
			"title": "Test",
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got ValidationErrorBody
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(got.Errors) != 2 {
		t.Errorf("len(Errors) = %d, want 2", len(got.Errors))
	}
	if got.Errors[0].Field != "title" {
		t.Errorf("Errors[0].Field = %v, want title", got.Errors[0].Field)
	}
	if _, ok := got.Values["title"]; !ok {
		t.Errorf("Values missing title key")
	}
}

func TestDuplicateErrorBody_JSON(t *testing.T) {
	meetingID := uuid.New().String()
	body := DuplicateErrorBody{
		Error:           "duplicate",
		MeetingID:       meetingID,
		OriginalOutcome: "created",
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got DuplicateErrorBody
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Error != body.Error {
		t.Errorf("Error = %v, want %v", got.Error, body.Error)
	}
	if got.MeetingID != body.MeetingID {
		t.Errorf("MeetingID = %v, want %v", got.MeetingID, body.MeetingID)
	}
	if got.OriginalOutcome != body.OriginalOutcome {
		t.Errorf("OriginalOutcome = %v, want %v", got.OriginalOutcome, body.OriginalOutcome)
	}
}

func TestInternalErrorBody_JSON(t *testing.T) {
	body := InternalErrorBody{
		Error:         "internal_error",
		CorrelationID: "req-123",
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got InternalErrorBody
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Error != body.Error {
		t.Errorf("Error = %v, want %v", got.Error, body.Error)
	}
	if got.CorrelationID != body.CorrelationID {
		t.Errorf("CorrelationID = %v, want %v", got.CorrelationID, body.CorrelationID)
	}
}

func TestIdempotencyToken_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	token := IdempotencyToken{
		Token:     uuid.New(),
		MeetingID: uuid.New(),
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got IdempotencyToken
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Token != token.Token {
		t.Errorf("Token = %v, want %v", got.Token, token.Token)
	}
	if got.MeetingID != token.MeetingID {
		t.Errorf("MeetingID = %v, want %v", got.MeetingID, token.MeetingID)
	}
}