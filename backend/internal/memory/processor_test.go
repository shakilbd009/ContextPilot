package memory

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestDefaultMemoryProcessor_Name(t *testing.T) {
	p := DefaultMemoryProcessor{}
	if got := p.Name(); got != "default-stub" {
		t.Errorf("Name() = %q, want %q", got, "default-stub")
	}
}

func TestDefaultMemoryProcessor_Process_NilTranscriptAndNotes(t *testing.T) {
	p := DefaultMemoryProcessor{}
	input := MemoryProcessorInput{
		MeetingID: uuid.New(),
		Title:     "Test Meeting",
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// With nil transcript and notes, summary quality should be insufficient
	if out.Content.Summary.QualityStatus != QualityInsufficient {
		t.Errorf("Summary.QualityStatus = %v, want %v", out.Content.Summary.QualityStatus, QualityInsufficient)
	}
	if out.Content.Summary.Statement == "" {
		t.Error("Summary.Statement should not be empty when no content")
	}
}

func TestDefaultMemoryProcessor_Process_WithTranscript(t *testing.T) {
	p := DefaultMemoryProcessor{}
	transcript := "This is a test transcript for the meeting."
	input := MemoryProcessorInput{
		MeetingID:  uuid.New(),
		Title:      "Test Meeting",
		Transcript: &transcript,
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// With transcript, summary should be weak quality
	if out.Content.Summary.QualityStatus != QualityWeak {
		t.Errorf("Summary.QualityStatus = %v, want %v", out.Content.Summary.QualityStatus, QualityWeak)
	}

	// Should have at least one evidence record from transcript
	if len(out.Evidence) == 0 {
		t.Error("expected at least one evidence record with transcript")
	}
}

func TestDefaultMemoryProcessor_Process_WithNotes(t *testing.T) {
	p := DefaultMemoryProcessor{}
	notes := "These are meeting notes."
	input := MemoryProcessorInput{
		MeetingID: uuid.New(),
		Title:     "Test Meeting",
		Notes:     &notes,
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if out.Content.Summary.QualityStatus != QualityWeak {
		t.Errorf("Summary.QualityStatus = %v, want %v", out.Content.Summary.QualityStatus, QualityWeak)
	}
}

func TestDefaultMemoryProcessor_Process_TranscriptTruncation(t *testing.T) {
	p := DefaultMemoryProcessor{}
	// Transcript longer than 200 chars
	bytes := make([]byte, 300)
	for i := range bytes {
		bytes[i] = 'a'
	}
	transcript := string(bytes)
	input := MemoryProcessorInput{
		MeetingID:  uuid.New(),
		Title:      "Test",
		Transcript: &transcript,
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Evidence snippet should be 200 chars max
	if len(out.Evidence) > 0 {
		if len(out.Evidence[0].EvidenceSnippet) > 200 {
			t.Errorf("evidence snippet length = %d, want <= 200", len(out.Evidence[0].EvidenceSnippet))
		}
	}
}

func TestDefaultMemoryProcessor_Process_BriefingReadiness(t *testing.T) {
	p := DefaultMemoryProcessor{}
	transcript := "Meeting content"
	input := MemoryProcessorInput{
		MeetingID:  uuid.New(),
		Title:      "Test",
		Transcript: &transcript,
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Default processor always sets Ready=false
	if out.Content.BriefingReadiness.Ready {
		t.Error("BriefingReadiness.Ready should be false for default processor")
	}
	if len(out.Content.BriefingReadiness.InsufficientCategories) == 0 {
		t.Error("expected insufficient categories list to be populated")
	}
}

func TestDefaultMemoryProcessor_Process_NoConflicts(t *testing.T) {
	p := DefaultMemoryProcessor{}
	input := MemoryProcessorInput{
		MeetingID:  uuid.New(),
		Title:      "Test",
		Transcript: strPtr("content"),
	}

	out, err := p.Process(context.Background(), input)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Default processor does not compute conflicts
	if len(out.Conflicts) != 0 {
		t.Errorf("Conflicts = %d, want 0", len(out.Conflicts))
	}
}

func TestNewConflictDetector(t *testing.T) {
	d := NewConflictDetector()
	if d == nil {
		t.Fatal("expected non-nil ConflictDetector")
	}
}

func TestConflictDetector_DetectConflicts_NoPriorItems(t *testing.T) {
	d := NewConflictDetector()
	current := []PriorMemoryItem{
		{ID: "1", Statement: "We will proceed with Phase 1", QualityStatus: QualityStrong},
	}

	conflicts := d.DetectConflicts(current, nil, "decisions")
	if len(conflicts) != 0 {
		t.Errorf("DetectConflicts with nil prior: got %d, want 0", len(conflicts))
	}
}

func TestConflictDetector_DetectConflicts_AllInsufficient(t *testing.T) {
	d := NewConflictDetector()
	current := []PriorMemoryItem{
		{ID: "1", Statement: "We will do something", QualityStatus: QualityInsufficient},
	}
	prior := []PriorMemoryItem{
		{ID: "2", Statement: "We will not do something", QualityStatus: QualityInsufficient},
	}

	conflicts := d.DetectConflicts(current, prior, "decisions")
	if len(conflicts) != 0 {
		t.Errorf("DetectConflicts with insufficient quality: got %d, want 0", len(conflicts))
	}
}

func TestConflictDetector_DetectConflicts_SameStatement(t *testing.T) {
	d := NewConflictDetector()
	current := []PriorMemoryItem{
		{ID: "1", Statement: "We will approve the budget", QualityStatus: QualityStrong},
	}
	prior := []PriorMemoryItem{
		{ID: "2", Statement: "We will approve the budget", QualityStatus: QualityStrong},
	}

	conflicts := d.DetectConflicts(current, prior, "decisions")
	if len(conflicts) != 0 {
		t.Errorf("DetectConflicts with identical statements: got %d, want 0", len(conflicts))
	}
}

func TestConflictDetector_DetectConflicts_DifferentStatements(t *testing.T) {
	d := NewConflictDetector()
	current := []PriorMemoryItem{
		{ID: "1", Statement: "We will approve the budget", QualityStatus: QualityStrong},
	}
	prior := []PriorMemoryItem{
		{ID: "2", Statement: "We will not approve the budget", QualityStatus: QualityStrong},
	}

	conflicts := d.DetectConflicts(current, prior, "decisions")
	if len(conflicts) != 1 {
		t.Errorf("DetectConflicts with opposing statements: got %d, want 1", len(conflicts))
	}
}

func TestConflictDetector_itemsConflict_ApproveVsReject(t *testing.T) {
	d := &ConflictDetector{}

	tests := []struct {
		a         string
		b         string
		conflict  bool
	}{
		// approve/reject are NOT explicit negations, so these should NOT conflict
		{"We will approve the budget", "We will reject the budget", false},
		// negation patterns that do conflict
		{"We will not approve the budget", "We will approve the budget", true},
		// cancellation patterns do conflict
		{"We will not proceed", "We will proceed", true},
		{"Project cancelled", "Project approved", true},
		// unrelated statements don't conflict
		{"We will start Phase 1", "We will start Phase 2", false},
		{"We will do X", "We will do Y", false},
	}

	for _, tt := range tests {
		got := d.itemsConflict(tt.a, tt.b)
		if got != tt.conflict {
			t.Errorf("itemsConflict(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.conflict)
		}
	}
}

func TestMemoryProcessorInput_Fields(t *testing.T) {
	meetingID := uuid.New()
	transcript := "transcript"
	notes := "notes"
	correlationID := uuid.New()

	input := MemoryProcessorInput{
		MeetingID:     meetingID,
		Transcript:    &transcript,
		Notes:         &notes,
		Title:         "Test",
		Participants:  []ParticipantInfo{},
		PriorMemories: []PriorMemoryInfo{},
		CorrelationID: correlationID,
	}

	if input.MeetingID != meetingID {
		t.Errorf("MeetingID = %v, want %v", input.MeetingID, meetingID)
	}
	if *input.Transcript != transcript {
		t.Errorf("Transcript = %v, want %v", *input.Transcript, transcript)
	}
	if *input.Notes != notes {
		t.Errorf("Notes = %v, want %v", *input.Notes, notes)
	}
	if input.CorrelationID != correlationID {
		t.Errorf("CorrelationID = %v, want %v", input.CorrelationID, correlationID)
	}
}

func TestParticipantInfo_Fields(t *testing.T) {
	id := uuid.New()
	email := "alice@example.com"
	org := "Acme"
	p := ParticipantInfo{
		ID:           id,
		DisplayName:  "Alice",
		Email:        &email,
		Organization: &org,
	}

	if p.ID != id {
		t.Errorf("ID = %v, want %v", p.ID, id)
	}
	if *p.Email != email {
		t.Errorf("Email = %v, want %v", *p.Email, email)
	}
}

func TestPriorMemoryInfo_Fields(t *testing.T) {
	memID := uuid.New()
	meetingID := uuid.New()
	summary := "Summary text"
	decision := PriorMemoryItem{
		ID:            "dec-1",
		Statement:     "We decided X",
		QualityStatus: QualityStrong,
		Category:      "decisions",
	}

	pmi := PriorMemoryInfo{
		MemoryVersionID: memID,
		MeetingID:       meetingID,
		Title:           "Prior Meeting",
		CompletedAt:     "2024-01-01",
		Decisions:      []PriorMemoryItem{decision},
		ActionItems:    []PriorMemoryItem{},
		RisksBlockers:   []PriorMemoryItem{},
		Summary:         &summary,
	}

	if pmi.MemoryVersionID != memID {
		t.Errorf("MemoryVersionID = %v, want %v", pmi.MemoryVersionID, memID)
	}
	if len(pmi.Decisions) != 1 {
		t.Fatalf("len(Decisions) = %d, want 1", len(pmi.Decisions))
	}
	if pmi.Decisions[0].Statement != "We decided X" {
		t.Errorf("Decisions[0].Statement = %q, want %q", pmi.Decisions[0].Statement, "We decided X")
	}
}

func TestMemoryProcessorOutput_Fields(t *testing.T) {
	content := MemoryContent{
		Summary: CategorySummary{
			Statement:     "Meeting summary",
			QualityStatus: QualityWeak,
		},
	}
	evidence := []MemoryEvidence{
		{
			ID:              uuid.New(),
			SourceType:      EvidenceSourceTranscript,
			EvidenceSnippet: "transcript snippet",
			QualityStatus:   QualityWeak,
		},
	}

	out := MemoryProcessorOutput{
		Content:         content,
		Evidence:        evidence,
		Conflicts:       nil,
		ProcessingNotes: "test note",
	}

	if out.Content.Summary.Statement != "Meeting summary" {
		t.Errorf("Content.Summary.Statement = %q, want %q", out.Content.Summary.Statement, "Meeting summary")
	}
	if len(out.Evidence) != 1 {
		t.Fatalf("len(Evidence) = %d, want 1", len(out.Evidence))
	}
	if out.Evidence[0].EvidenceSnippet != "transcript snippet" {
		t.Errorf("Evidence[0].Snippet = %q, want %q", out.Evidence[0].EvidenceSnippet, "transcript snippet")
	}
}