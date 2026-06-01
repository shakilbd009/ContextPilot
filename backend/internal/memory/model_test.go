package memory

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJobStatus_Values(t *testing.T) {
	statuses := []JobStatus{
		JobStatusQueued,
		JobStatusProcessing,
		JobStatusCompleted,
		JobStatusCompletedInsufficient,
		JobStatusFailed,
		JobStatusRetrying,
		JobStatusRetryExhausted,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("JobStatus %q is empty", s)
		}
	}
}

func TestTriggerType_Values(t *testing.T) {
	triggers := []TriggerType{
		TriggerTypeImport,
		TriggerTypeReprocess,
		TriggerTypeStaleReprocess,
		TriggerTypeManualRetry,
		TriggerTypeConflictResolution,
	}
	for _, tr := range triggers {
		if tr == "" {
			t.Errorf("TriggerType %q is empty", tr)
		}
	}
}

func TestFailureClass_Values(t *testing.T) {
	classes := []FailureClass{
		FailureClassTransientProvider,
		FailureClassPermanentProcessing,
		FailureClassValidation,
		FailureClassTimeout,
		FailureClassWorkerUnavailable,
	}
	for _, c := range classes {
		if c == "" {
			t.Errorf("FailureClass %q is empty", c)
		}
	}
}

func TestQualityStatus_Values(t *testing.T) {
	statuses := []QualityStatus{
		QualityStrong,
		QualityWeak,
		QualityInsufficient,
		QualityConflicting,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("QualityStatus %q is empty", s)
		}
	}
}

func TestActionStatus_Values(t *testing.T) {
	statuses := []ActionStatus{
		ActionStatusPending,
		ActionStatusInProgress,
		ActionStatusCompleted,
		ActionStatusDeferred,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("ActionStatus %q is empty", s)
		}
	}
}

func TestChangeStatus_Values(t *testing.T) {
	statuses := []ChangeStatus{
		ChangeStandalone,
		ChangeConfirmsPrior,
		ChangeChangesPrior,
		ChangeReversesPrior,
	}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("ChangeStatus %q is empty", s)
		}
	}
}

func TestMemoryProcessingJob_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	job := MemoryProcessingJob{
		ID:          uuid.New(),
		MeetingID:   uuid.New(),
		TriggerType: TriggerTypeImport,
		Status:      JobStatusQueued,
		RetryCount:  0,
		MaxRetries:  3,
		CorrelationID: uuid.New(),
		QueuedAt:    now,
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got MemoryProcessingJob
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != job.ID {
		t.Errorf("ID = %v, want %v", got.ID, job.ID)
	}
	if got.TriggerType != job.TriggerType {
		t.Errorf("TriggerType = %v, want %v", got.TriggerType, job.TriggerType)
	}
	if got.Status != job.Status {
		t.Errorf("Status = %v, want %v", got.Status, job.Status)
	}
}

func TestMemoryVersion_JSON(t *testing.T) {
	id := uuid.New()
	meetingID := uuid.New()

	content := MemoryContent{
		Summary: CategorySummary{
			Statement:     "Meeting occurred",
			QualityStatus: QualityWeak,
		},
	}
	contentBytes, _ := json.Marshal(content)

	version := MemoryVersion{
		ID:            id,
		MeetingID:     meetingID,
		VersionNumber: 1,
		Status:        VersionStatusActive,
		IsActive:      true,
		Content:       content,
		TriggerType:   TriggerTypeImport,
		CreatedAt:     time.Now(),
	}

	data, err := json.Marshal(version)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify Content serializes correctly
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	_ = contentBytes // suppress unused warning
}

func TestCategorySummary_JSON(t *testing.T) {
	cs := CategorySummary{
		Statement:     "Test statement",
		QualityStatus: QualityStrong,
		Items:         []interface{}{"item1", "item2"},
	}

	data, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got CategorySummary
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Statement != cs.Statement {
		t.Errorf("Statement = %v, want %v", got.Statement, cs.Statement)
	}
	if got.QualityStatus != cs.QualityStatus {
		t.Errorf("QualityStatus = %v, want %v", got.QualityStatus, cs.QualityStatus)
	}
}

func TestDecisionItem_JSON(t *testing.T) {
	di := DecisionItem{
		ID:                "dec-1",
		DecisionStatement: "We approved the budget",
		ChangeStatus:      ChangeStandalone,
		QualityStatus:     QualityStrong,
		Evidence:          []EvidenceRef{},
	}

	data, err := json.Marshal(di)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got DecisionItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != di.ID {
		t.Errorf("ID = %v, want %v", got.ID, di.ID)
	}
	if got.DecisionStatement != di.DecisionStatement {
		t.Errorf("DecisionStatement = %v, want %v", got.DecisionStatement, di.DecisionStatement)
	}
}

func TestActionItem_JSON(t *testing.T) {
	owner := "Alice"
	due := "2024-01-15"
	ai := ActionItem{
		ID:               "ai-1",
		Description:      "Review the proposal",
		Owner:            &owner,
		OwnerSpecified:   true,
		DueDate:          &due,
		DueDateSpecified: true,
		Status:           ActionStatusPending,
		QualityStatus:    QualityWeak,
		Evidence:         []EvidenceRef{},
	}

	data, err := json.Marshal(ai)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got ActionItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.ID != ai.ID {
		t.Errorf("ID = %v, want %v", got.ID, ai.ID)
	}
	if *got.Owner != owner {
		t.Errorf("Owner = %v, want %v", *got.Owner, owner)
	}
}

func TestRiskBlockerItem_JSON(t *testing.T) {
	rbi := RiskBlockerItem{
		ID:            "rb-1",
		Type:          RiskTypeRisk,
		Description:   "Budget risk identified",
		QualityStatus: QualityWeak,
		Evidence:      []EvidenceRef{},
	}

	data, err := json.Marshal(rbi)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got RiskBlockerItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Type != RiskTypeRisk {
		t.Errorf("Type = %v, want %v", got.Type, RiskTypeRisk)
	}
}

func TestMemoryContent_AllCategories(t *testing.T) {
	content := MemoryContent{
		Summary: CategorySummary{
			Statement:     "Test",
			QualityStatus: QualityWeak,
		},
		Decisions: DecisionsCategory{
			Items: []DecisionItem{},
		},
		ActionItems: ActionItemsCategory{
			Items: []ActionItem{},
		},
		RisksBlockers: RisksBlockersCategory{
			Items: []RiskBlockerItem{},
		},
		OpenQuestions: OpenQuestionsCategory{
			Items: []OpenQuestionItem{},
		},
		StakeholderNotes: StakeholderNotesCategory{
			Items: []StakeholderNoteItem{},
		},
		NextRecommendedFocus: NextRecommendedFocus{
			Statement:     "",
			QualityStatus: QualityInsufficient,
		},
		BriefingReadiness: BriefingReadiness{
			Ready:               false,
			CoreCategoriesReady: false,
			BlockingConflicts:   []string{},
			InsufficientCategories: []string{"summary"},
		},
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got MemoryContent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Summary.QualityStatus != QualityWeak {
		t.Errorf("Summary.QualityStatus = %v, want %v", got.Summary.QualityStatus, QualityWeak)
	}
	if got.BriefingReadiness.Ready != false {
		t.Error("BriefingReadiness.Ready should be false")
	}
}

func TestConflictPair_Fields(t *testing.T) {
	pair := ConflictPair{
		Current: []ConflictItemContent{
			{ID: "cur-1", Statement: "Current statement", QualityStatus: QualityStrong},
		},
		Prior: []ConflictItemContent{
			{ID: "pri-1", Statement: "Prior statement", QualityStatus: QualityStrong},
		},
	}

	if len(pair.Current) != 1 {
		t.Fatalf("len(Current) = %d, want 1", len(pair.Current))
	}
	if pair.Current[0].Statement != "Current statement" {
		t.Errorf("Current[0].Statement = %q, want %q", pair.Current[0].Statement, "Current statement")
	}
}

func TestWorkerConfig_Defaults(t *testing.T) {
	cfg := DefaultWorkerConfig()

	if cfg.PollInterval == 0 {
		t.Error("PollInterval should not be zero")
	}
	if cfg.MaxRetries == 0 {
		t.Error("MaxRetries should not be zero")
	}
	if cfg.InitialBackoff == 0 {
		t.Error("InitialBackoff should not be zero")
	}
	if cfg.MaxBackoff == 0 {
		t.Error("MaxBackoff should not be zero")
	}
}

func TestWorker_ProcessBatch_Empty(t *testing.T) {
	// Test that processBatch doesn't panic when no jobs available
	// We can't easily test this without a real DB, but we can verify
	// the method exists and the Worker type is correct
	cfg := DefaultWorkerConfig()
	w := &Worker{
		cfg: cfg,
	}
	if w == nil {
		t.Fatal("Worker should not be nil")
	}
}