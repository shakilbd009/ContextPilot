package briefing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Tests for HTTP response helpers
func TestForbidden(t *testing.T) {
	rr := httptest.NewRecorder()
	forbidden(rr, "test detail")
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["detail"] != "test detail" {
		t.Errorf("detail = %v, want %q", resp["detail"], "test detail")
	}
}

func TestNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	notFound(rr, "not found")
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestUnauthorized(t *testing.T) {
	rr := httptest.NewRecorder()
	unauthorized(rr)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestInternalError(t *testing.T) {
	rr := httptest.NewRecorder()
	internalError(rr)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestConflict(t *testing.T) {
	rr := httptest.NewRecorder()
	conflict(rr, "already running")
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["detail"] != "already running" {
		t.Errorf("detail = %v, want %q", resp["detail"], "already running")
	}
}

func TestConflict_EscapesHTML(t *testing.T) {
	rr := httptest.NewRecorder()
	conflict(rr, "<script>alert('xss')</script>")
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	// detail must be escaped to prevent XSS in UI rendering
	if resp["detail"] != "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" {
		t.Errorf("detail = %v, want escaped HTML", resp["detail"])
	}
}

// Tests for getUserID header parsing
func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		header string
		wantID uuid.UUID
		wantOk bool
	}{
		{
			name:   "valid uuid",
			header: "550e8400-e29b-41d4-a716-446655440000",
			wantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantOk: true,
		},
		{
			name:   "valid with spaces",
			header: "  550e8400-e29b-41d4-a716-446655440000  ",
			wantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantOk: true,
		},
		{
			name:   "empty",
			header: "",
			wantID: uuid.Nil,
			wantOk: false,
		},
		{
			name:   "invalid uuid",
			header: "not-a-uuid",
			wantID: uuid.Nil,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-User-ID", tt.header)
			gotID, gotOk := getUserID(req)
			if gotID != tt.wantID || gotOk != tt.wantOk {
				t.Errorf("getUserID() = (%v, %v), want (%v, %v)", gotID, gotOk, tt.wantID, tt.wantOk)
			}
		})
	}
}

// Tests for ExclusionAccepted JSON round-trip
func TestExclusionAccepted_JSON(t *testing.T) {
	now := time.Now()
	ea := ExclusionAccepted{
		SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		RestoredAt:      &now,
	}

	encoded, err := json.Marshal(ea)
	if err != nil {
		t.Fatalf("json.Marshal(ea) = %v, want nil", err)
	}

	var decoded ExclusionAccepted
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.SourceMeetingID != ea.SourceMeetingID {
		t.Errorf("SourceMeetingID = %v, want %v", decoded.SourceMeetingID, ea.SourceMeetingID)
	}
	if decoded.RestoredAt == nil {
		t.Error("RestoredAt = nil, want non-nil")
	}
}

func TestExclusionAccepted_NilRestoredAt(t *testing.T) {
	ea := ExclusionAccepted{
		SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		RestoredAt:      nil,
	}

	encoded, err := json.Marshal(ea)
	if err != nil {
		t.Fatalf("json.Marshal(ea) = %v, want nil", err)
	}

	var decoded ExclusionAccepted
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.RestoredAt != nil {
		t.Errorf("RestoredAt = %v, want nil", decoded.RestoredAt)
	}
}

// Tests for BriefingProcessingJob fields
func TestBriefingProcessingJob_Fields(t *testing.T) {
	now := time.Now()
	job := BriefingProcessingJob{
		ID:                uuid.New(),
		UpcomingMeetingID: uuid.New(),
		TriggerType:      BriefingTriggerAuto,
		Status:           JobStatusQueued,
		RetryCount:       0,
		MaxRetries:       3,
		CorrelationID:    uuid.New(),
		QueuedAt:         now,
	}

	if job.TriggerType != BriefingTriggerAuto {
		t.Errorf("TriggerType = %v, want %v", job.TriggerType, BriefingTriggerAuto)
	}
	if job.Status != JobStatusQueued {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusQueued)
	}
	if job.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", job.RetryCount)
	}
}

// Tests for BriefingResponse JSON round-trip
func TestBriefingResponse_JSON(t *testing.T) {
	now := time.Now()
	content := BriefingContent{
		ConciseSummary: ConciseSummary{
			Objective:          "Test objective",
			PreparationStatus: "ready",
			RecommendedFocus:  "Focus area",
			TopPriorContext:    "Prior context",
			OpenActions:       "Action items",
			RisksQuestions:   "Risks",
		},
	}

	resp := BriefingResponse{
		MeetingID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		VersionNumber:      1,
		Status:             "active",
		PreparationStatus:  PreparationStatusReady,
		IsStale:             false,
		Content:            &content,
		SourceCount:        2,
		CreatedAt:          now,
		TriggerType:        BriefingTriggerAuto,
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(resp) = %v, want nil", err)
	}

	var decoded BriefingResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.MeetingID != resp.MeetingID {
		t.Errorf("MeetingID = %v, want %v", decoded.MeetingID, resp.MeetingID)
	}
	if decoded.VersionNumber != resp.VersionNumber {
		t.Errorf("VersionNumber = %d, want %d", decoded.VersionNumber, resp.VersionNumber)
	}
	if decoded.IsStale {
		t.Error("IsStale = true, want false")
	}
	if decoded.Content == nil {
		t.Error("Content = nil, want non-nil")
	} else if decoded.Content.ConciseSummary.Objective != "Test objective" {
		t.Errorf("Content.ConciseSummary.Objective = %q, want %q", decoded.Content.ConciseSummary.Objective, "Test objective")
	}
}