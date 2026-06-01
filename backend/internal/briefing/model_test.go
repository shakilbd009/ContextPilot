package briefing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPreparationStatus_Values(t *testing.T) {
	valid := []PreparationStatus{
		PreparationStatusGenerating,
		PreparationStatusReady,
		PreparationStatusReadyCaveats,
		PreparationStatusNoPriorMemory,
		PreparationStatusStale,
		PreparationStatusFailed,
		PreparationStatusRegenerating,
	}
	for _, s := range valid {
		if s == "" {
			t.Errorf("PreparationStatus value must not be empty")
		}
	}
}

func TestBriefingResult_Values(t *testing.T) {
	valid := []BriefingResult{
		BriefingResultReady,
		BriefingResultReadyCaveats,
		BriefingResultNoPriorMemory,
		BriefingResultFailed,
	}
	for _, r := range valid {
		if r == "" {
			t.Errorf("BriefingResult value must not be empty")
		}
	}
}

func TestBriefingVersionStatus_Values(t *testing.T) {
	valid := []BriefingVersionStatus{
		BriefingVersionStatusActive,
		BriefingVersionStatusSuperseded,
	}
	for _, s := range valid {
		if s == "" {
			t.Errorf("BriefingVersionStatus value must not be empty")
		}
	}
}

func TestBriefingTriggerType_Values(t *testing.T) {
	valid := []BriefingTriggerType{
		BriefingTriggerAuto,
		BriefingTriggerManualRegenerate,
	}
	for _, bt := range valid {
		if bt == "" {
			t.Error("BriefingTriggerType value must not be empty")
		}
	}
}

func TestBriefingContent_JSON(t *testing.T) {
	content := BriefingContent{
		ConciseSummary: ConciseSummary{
			Objective:          "Discuss Q3 roadmap",
			PreparationStatus: "ready",
			RecommendedFocus:  "Budget approval",
			TopPriorContext:    "Last discussed in June meeting",
			OpenActions:       "3 items pending",
			RisksQuestions:    "Resource allocation risk",
		},
		DetailedSections: DetailedSections{
			PreviousRelevantContext: SectionItem{Statement: "Prior context here", QualityStatus: "strong_evidence"},
			ImportantPriorDecisions: SectionWithItems{Items: []interface{}{"Decision 1", "Decision 2"}, QualityStatus: "strong_evidence"},
			OpenActionItems:         SectionWithItems{Items: []interface{}{"Action 1"}, QualityStatus: "weak_evidence"},
			UnresolvedRisksBlockers: SectionWithItems{Items: []interface{}{}, QualityStatus: "insufficient_evidence"},
			OpenQuestions:           SectionWithItems{Items: []interface{}{"Question 1"}, QualityStatus: "weak_evidence"},
			StakeholderNotes:        SectionWithItems{Items: []interface{}{}, QualityStatus: "insufficient_evidence"},
			SuggestedQuestions:      StringArray{Items: []string{"What is the timeline?"}},
			SuggestedAgenda:         StringArray{Items: []string{"Intro", "Budget", "Close"}},
		},
		Sources: []SourceMeeting{
			{
				SourceMeetingID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
				RelatednessReasons: []string{"same participant", "similar title"},
				QualityStatus:      "strong_evidence",
			},
		},
	}

	encoded, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("json.Marshal(content) = %v, want nil", err)
	}

	var decoded BriefingContent
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.ConciseSummary.Objective != content.ConciseSummary.Objective {
		t.Errorf("ConciseSummary.Objective = %q, want %q", decoded.ConciseSummary.Objective, content.ConciseSummary.Objective)
	}
	if len(decoded.Sources) != len(content.Sources) {
		t.Errorf("len(Sources) = %d, want %d", len(decoded.Sources), len(content.Sources))
	}
	if decoded.Sources[0].SourceMeetingID != content.Sources[0].SourceMeetingID {
		t.Errorf("Sources[0].SourceMeetingID = %v, want %v", decoded.Sources[0].SourceMeetingID, content.Sources[0].SourceMeetingID)
	}
}

func TestBriefingSourceExclusion_JSON(t *testing.T) {
	now := time.Now()
	e := BriefingSourceExclusion{
		ID:                     uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		UpcomingMeetingID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
		ExcludedSourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
		CreatedAt:             now,
		RestoredAt:            nil,
	}

	encoded, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal(e) = %v, want nil", err)
	}

	var decoded BriefingSourceExclusion
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.ID != e.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, e.ID)
	}
	if decoded.RestoredAt != nil {
		t.Errorf("RestoredAt = %v, want nil", decoded.RestoredAt)
	}
}

func TestExclusionEntry_JSON(t *testing.T) {
	now := time.Now()
	ee := ExclusionEntry{
		SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		CreatedAt:       now,
		RestoredAt:      nil,
	}

	encoded, err := json.Marshal(ee)
	if err != nil {
		t.Fatalf("json.Marshal(ee) = %v, want nil", err)
	}

	var decoded ExclusionEntry
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.SourceMeetingID != ee.SourceMeetingID {
		t.Errorf("SourceMeetingID = %v, want %v", decoded.SourceMeetingID, ee.SourceMeetingID)
	}
}

func TestSourceExclusionResponse_JSON(t *testing.T) {
	resp := SourceExclusionResponse{
		UpcomingMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Excluded: []ExclusionEntry{
			{
				SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
				CreatedAt:       time.Now(),
				RestoredAt:      nil,
			},
		},
		Restored: []ExclusionEntry{},
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(resp) = %v, want nil", err)
	}

	var decoded SourceExclusionResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.UpcomingMeetingID != resp.UpcomingMeetingID {
		t.Errorf("UpcomingMeetingID = %v, want %v", decoded.UpcomingMeetingID, resp.UpcomingMeetingID)
	}
	if len(decoded.Excluded) != 1 {
		t.Errorf("len(Excluded) = %d, want 1", len(decoded.Excluded))
	}
}

func TestRegenerateAccepted_JSON(t *testing.T) {
	jobID := uuid.New()
	correlationID := uuid.New()
	ra := RegenerateAccepted{
		JobID:         jobID,
		CorrelationID: correlationID,
	}

	encoded, err := json.Marshal(ra)
	if err != nil {
		t.Fatalf("json.Marshal(ra) = %v, want nil", err)
	}

	var decoded RegenerateAccepted
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.JobID != jobID {
		t.Errorf("JobID = %v, want %v", decoded.JobID, jobID)
	}
}

func TestNoPriorMemoryShell_JSON(t *testing.T) {
	shell := NoPriorMemoryShell{
		Generated:              true,
		Objective:              "Prepare for upcoming meeting",
		RecommendedFocus:      "Review action items",
		SuggestedPrepQuestions: []string{"What was decided last time?", "Any blockers?"},
	}

	encoded, err := json.Marshal(shell)
	if err != nil {
		t.Fatalf("json.Marshal(shell) = %v, want nil", err)
	}

	var decoded NoPriorMemoryShell
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.Generated != shell.Generated {
		t.Errorf("Generated = %v, want %v", decoded.Generated, shell.Generated)
	}
	if decoded.Objective != shell.Objective {
		t.Errorf("Objective = %q, want %q", decoded.Objective, shell.Objective)
	}
}

func TestBriefingVersionSummary_AllFields(t *testing.T) {
	now := time.Now()
	vs := BriefingVersionSummary{
		VersionNumber: 3,
		IsActive:      true,
		Status:        "active",
		Result:        "ready",
		CreatedAt:     now,
		TriggerType:   BriefingTriggerAuto,
	}

	encoded, err := json.Marshal(vs)
	if err != nil {
		t.Fatalf("json.Marshal(vs) = %v, want nil", err)
	}

	var decoded BriefingVersionSummary
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.VersionNumber != vs.VersionNumber {
		t.Errorf("VersionNumber = %d, want %d", decoded.VersionNumber, vs.VersionNumber)
	}
	if decoded.IsActive != vs.IsActive {
		t.Errorf("IsActive = %v, want %v", decoded.IsActive, vs.IsActive)
	}
}

func TestJobStatus_Values(t *testing.T) {
	valid := []JobStatus{
		JobStatusQueued,
		JobStatusProcessing,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusRetrying,
		JobStatusRetryExhausted,
	}
	for _, s := range valid {
		if s == "" {
			t.Errorf("JobStatus value must not be empty")
		}
	}
}

func TestFailureClass_Values(t *testing.T) {
	valid := []FailureClass{
		FailureClassNone,
		FailureClassUpstreamError,
		FailureClassTimeout,
		FailureClassInternalError,
	}
	for _, f := range valid {
		if f == "" {
			t.Errorf("FailureClass value must not be empty")
		}
	}
}