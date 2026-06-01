package memory

import (
	"time"

	"github.com/google/uuid"
)

// ─── Job statuses ─────────────────────────────────────────────────────────────

// JobStatus represents the processing lifecycle state of a memory job.
type JobStatus string

const (
	JobStatusQueued                   JobStatus = "queued"
	JobStatusProcessing               JobStatus = "processing"
	JobStatusCompleted                JobStatus = "completed"
	JobStatusCompletedInsufficient     JobStatus = "completed_with_insufficient_evidence"
	JobStatusFailed                   JobStatus = "failed"
	JobStatusRetrying                 JobStatus = "retrying"
	JobStatusRetryExhausted          JobStatus = "retry_exhausted"
)

// TriggerType represents what caused a processing job to be created.
type TriggerType string

const (
	TriggerTypeImport             TriggerType = "import"
	TriggerTypeReprocess          TriggerType = "reprocess"
	TriggerTypeStaleReprocess     TriggerType = "stale_reprocess"
	TriggerTypeManualRetry        TriggerType = "manual_retry"
	TriggerTypeConflictResolution TriggerType = "conflict_resolution"
)

// FailureClass classifies permanent vs transient failures for retry logic.
type FailureClass string

const (
	FailureClassTransientProvider  FailureClass = "transient_provider_error"
	FailureClassPermanentProcessing FailureClass = "permanent_processing_error"
	FailureClassValidation         FailureClass = "validation_error"
	FailureClassTimeout            FailureClass = "timeout"
	FailureClassWorkerUnavailable  FailureClass = "worker_unavailable"
)

// ─── Memory Processing Job ────────────────────────────────────────────────────

// MemoryProcessingJob represents a queued/processing memory job.
type MemoryProcessingJob struct {
	ID                uuid.UUID  `json:"id"`
	MeetingID         uuid.UUID  `json:"meetingId"`
	TriggerType       TriggerType `json:"triggerType"`
	Status            JobStatus  `json:"status"`
	RetryCount        int        `json:"retryCount"`
	MaxRetries        int        `json:"maxRetries"`
	FailureReason     *string    `json:"failureReason,omitempty"`
	PreviousVersionID *uuid.UUID `json:"previousVersionId,omitempty"`
	CorrelationID     uuid.UUID  `json:"correlationId"`
	QueuedAt          time.Time  `json:"queuedAt"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	NextRetryAt       *time.Time `json:"nextRetryAt,omitempty"`
}

// ─── Memory Version ───────────────────────────────────────────────────────────

// MemoryVersion represents a single version of processed meeting memory.
type MemoryVersion struct {
	ID            uuid.UUID     `json:"id"`
	MeetingID     uuid.UUID     `json:"meetingId"`
	JobID         *uuid.UUID    `json:"jobId,omitempty"`
	VersionNumber int           `json:"versionNumber"`
	Status        VersionStatus `json:"status"`
	IsActive      bool         `json:"isActive"`
	Content       MemoryContent `json:"content"`
	TriggerType   TriggerType  `json:"triggerType"`
	CreatedAt     time.Time    `json:"createdAt"`
	CreatedBy     *uuid.UUID   `json:"createdBy,omitempty"`
}

// VersionStatus is the status of a memory version within its lifecycle.
type VersionStatus string

const (
	VersionStatusActive        VersionStatus = "active"
	VersionStatusSuperseded   VersionStatus = "superseded"
	VersionStatusConflictReview VersionStatus = "conflict_review"
)

// ─── Memory Content (JSONB) ──────────────────────────────────────────────────

// MemoryContent is the structured memory output stored in memory_versions.content.
type MemoryContent struct {
	Summary               CategorySummary          `json:"summary"`
	Decisions            DecisionsCategory        `json:"decisions"`
	ActionItems          ActionItemsCategory      `json:"action_items"`
	RisksBlockers        RisksBlockersCategory    `json:"risks_blockers"`
	OpenQuestions        OpenQuestionsCategory    `json:"open_questions"`
	StakeholderNotes     StakeholderNotesCategory `json:"stakeholder_notes"`
	NextRecommendedFocus NextRecommendedFocus     `json:"next_recommended_focus"`
	BriefingReadiness    BriefingReadiness        `json:"briefing_readiness"`
}

// ─── Categories ──────────────────────────────────────────────────────────────

// QualityStatus represents evidence quality for a memory item or category.
type QualityStatus string

const (
	QualityStrong        QualityStatus = "strong_evidence"
	QualityWeak         QualityStatus = "weak_evidence"
	QualityInsufficient QualityStatus = "insufficient_evidence"
	QualityConflicting  QualityStatus = "conflicting_evidence"
)

// CategorySummary is the summary category.
type CategorySummary struct {
	Statement     string        `json:"statement,omitempty"`
	QualityStatus QualityStatus `json:"quality_status"`
	Items         []interface{} `json:"items,omitempty"`
}

// DecisionsCategory is the decisions category.
type DecisionsCategory struct {
	Items []DecisionItem `json:"items"`
}

// DecisionItem is a decision extracted from the meeting.
type DecisionItem struct {
	ID                string         `json:"id"`
	DecisionStatement string         `json:"decision_statement"`
	ChangeStatus     ChangeStatus   `json:"change_status"`
	PriorDecisionRef *uuid.UUID     `json:"prior_decision_ref,omitempty"`
	QualityStatus    QualityStatus  `json:"quality_status"`
	Evidence          []EvidenceRef `json:"evidence"`
}

// ChangeStatus describes how a decision relates to prior memory.
type ChangeStatus string

const (
	ChangeStandalone    ChangeStatus = "standalone"
	ChangeConfirmsPrior ChangeStatus = "confirms_prior"
	ChangeChangesPrior ChangeStatus = "changes_prior"
	ChangeReversesPrior ChangeStatus = "reverses_prior"
)

// ActionItemsCategory is the action items category.
type ActionItemsCategory struct {
	Items []ActionItem `json:"items"`
}

// ActionItem is an action item extracted from the meeting.
type ActionItem struct {
	ID               string        `json:"id"`
	Description      string        `json:"description"`
	Owner            *string       `json:"owner,omitempty"`
	OwnerSpecified   bool          `json:"owner_specified"`
	DueDate          *string       `json:"due_date,omitempty"`
	DueDateSpecified bool          `json:"due_date_specified"`
	Status           ActionStatus  `json:"status"`
	QualityStatus    QualityStatus `json:"quality_status"`
	Evidence         []EvidenceRef `json:"evidence"`
}

// ActionStatus is the completion status of an action item.
type ActionStatus string

const (
	ActionStatusPending    ActionStatus = "pending"
	ActionStatusInProgress ActionStatus = "in_progress"
	ActionStatusCompleted  ActionStatus = "completed"
	ActionStatusDeferred   ActionStatus = "deferred"
)

// RisksBlockersCategory is the risks/blockers category.
type RisksBlockersCategory struct {
	Items []RiskBlockerItem `json:"items"`
}

// RiskBlockerItem is a risk or blocker extracted from the meeting.
type RiskBlockerItem struct {
	ID            string        `json:"id"`
	Type          RiskType     `json:"type"`
	Description   string        `json:"description"`
	QualityStatus QualityStatus `json:"quality_status"`
	Evidence      []EvidenceRef `json:"evidence"`
}

// RiskType distinguishes risks from blockers.
type RiskType string

const (
	RiskTypeRisk    RiskType = "risk"
	RiskTypeBlocker RiskType = "blocker"
)

// OpenQuestionsCategory is the open questions category.
type OpenQuestionsCategory struct {
	Items []OpenQuestionItem `json:"items"`
}

// OpenQuestionItem is an open question extracted from the meeting.
type OpenQuestionItem struct {
	ID            string        `json:"id"`
	Question      string        `json:"question"`
	QualityStatus QualityStatus `json:"quality_status"`
	Evidence      []EvidenceRef `json:"evidence"`
}

// StakeholderNotesCategory is the stakeholder notes category.
type StakeholderNotesCategory struct {
	Items []StakeholderNoteItem `json:"items"`
}

// StakeholderNoteItem is a per-participant business-context note.
type StakeholderNoteItem struct {
	ID             string        `json:"id"`
	ParticipantID  uuid.UUID     `json:"participant_id"`
	Preferences    *string       `json:"preferences,omitempty"`
	Concerns       *string       `json:"concerns,omitempty"`
	Commitments    *string       `json:"commitments,omitempty"`
	InfluenceStake *string       `json:"influence_stake,omitempty"`
	QualityStatus  QualityStatus `json:"quality_status"`
	Evidence       []EvidenceRef `json:"evidence"`
}

// NextRecommendedFocus is the synthesized next-step recommendation.
type NextRecommendedFocus struct {
	Statement         string         `json:"statement,omitempty"`
	QualityStatus     QualityStatus  `json:"quality_status"`
	SupportingItemIDs []string       `json:"supporting_item_ids,omitempty"`
	Evidence          []EvidenceRef  `json:"evidence"`
}

// BriefingReadiness signals whether this memory can safely power a briefing.
type BriefingReadiness struct {
	Ready                  bool     `json:"ready"`
	CoreCategoriesReady    bool     `json:"core_categories_ready"`
	BlockingConflicts      []string `json:"blocking_conflicts,omitempty"`
	InsufficientCategories []string `json:"insufficient_categories,omitempty"`
}

// BriefingReadinessState is the briefing-readiness signal text.
type BriefingReadinessState string

const (
	BriefingReady             BriefingReadinessState = "ready"
	BriefingReadyWithWeak    BriefingReadinessState = "ready_with_weak"
	BriefingReadyInsufficient BriefingReadinessState = "ready_with_insufficient"
	BriefingNeedsReview      BriefingReadinessState = "needs_review"
	BriefingProcessing       BriefingReadinessState = "processing"
	BriefingReprocessing    BriefingReadinessState = "reprocessing"
	BriefingFailed           BriefingReadinessState = "failed"
	BriefingRetryExhausted   BriefingReadinessState = "retry_exhausted"
	BriefingStale            BriefingReadinessState = "stale"
)

// EvidenceRef is a reference to an evidence record in memory_evidence.
type EvidenceRef struct {
	Snippet         string             `json:"snippet"`
	SourceType      EvidenceSourceType `json:"source_type"`
	SourceLocation  SourceLocation     `json:"source_location"`
	SourceMeetingID uuid.UUID          `json:"source_meeting_id"`
}

// EvidenceSourceType is the origin of an evidence record.
type EvidenceSourceType string

const (
	EvidenceSourceTranscript          EvidenceSourceType = "transcript"
	EvidenceSourceNotes               EvidenceSourceType = "notes"
	EvidenceSourcePriorMemoryReference EvidenceSourceType = "prior_memory_reference"
)

// SourceLocation is the character-offset location within source text.
type SourceLocation struct {
	Type  string `json:"type"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// ─── Memory Evidence ─────────────────────────────────────────────────────────

// MemoryEvidence is a normalized evidence record joined to a memory item.
type MemoryEvidence struct {
	ID              uuid.UUID        `json:"id"`
	MemoryVersionID uuid.UUID        `json:"memoryVersionId"`
	Category        string           `json:"category"`
	ItemID          string           `json:"itemId"`
	SourceType      EvidenceSourceType `json:"sourceType"`
	SourceLocation  SourceLocation   `json:"sourceLocation"`
	EvidenceSnippet string           `json:"evidenceSnippet"`
	QualityStatus   QualityStatus    `json:"qualityStatus"`
	CreatedAt       time.Time        `json:"createdAt"`
}

// ─── Memory Conflict ─────────────────────────────────────────────────────────

// MemoryConflict is a conflict record pending review.
type MemoryConflict struct {
	ID               uuid.UUID     `json:"id"`
	MeetingID        uuid.UUID     `json:"meetingId"`
	MemoryVersionID  *uuid.UUID    `json:"memoryVersionId,omitempty"`
	ConflictingItems ConflictPair  `json:"conflicting_items"`
	QualityStatus    QualityStatus `json:"quality_status"`
	ReviewStatus     ReviewStatus  `json:"review_status"`
	ResolutionNote   *string       `json:"resolution_note,omitempty"`
	ResolvedAt       *time.Time    `json:"resolved_at,omitempty"`
	ResolvedBy       *uuid.UUID    `json:"resolved_by,omitempty"`
	CreatedAt        time.Time     `json:"createdAt"`
}

// ConflictPair holds the current and prior conflicting items.
type ConflictPair struct {
	Current []ConflictItemContent `json:"current"`
	Prior   []ConflictItemContent `json:"prior"`
}

// ConflictItemContent is the minimal content of a conflicting item shown in the review UI.
type ConflictItemContent struct {
	ID            string        `json:"id,omitempty"`
	Statement     string        `json:"statement,omitempty"`
	QualityStatus QualityStatus `json:"quality_status"`
}

// ReviewStatus is whether a conflict has been reviewed.
type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusReviewed ReviewStatus = "reviewed"
)

// ─── Prior Memory Input ──────────────────────────────────────────────────────

// PriorMemoryInput records which prior memories were used as processing input.
type PriorMemoryInput struct {
	ID                   uuid.UUID       `json:"id"`
	MemoryVersionID      uuid.UUID       `json:"memoryVersionId"`
	PriorMemoryVersionID  uuid.UUID       `json:"priorMemoryVersionId"`
	MatchConfidence      *MatchConfidence `json:"match_confidence,omitempty"`
	IncludedByUser       bool            `json:"included_by_user"`
	ExcludedByUser       bool            `json:"excluded_by_user"`
	CreatedAt            time.Time       `json:"createdAt"`
}

// MatchConfidence describes how a prior memory was matched.
type MatchConfidence string

const (
	MatchSafe      MatchConfidence = "safe_match"
	MatchUncertain MatchConfidence = "uncertain"
)

// ─── Processing State (GET /memory/state) ───────────────────────────────────

// ProcessingState is the aggregated processing state for a meeting.
type ProcessingState struct {
	MeetingID           uuid.UUID            `json:"meetingId"`
	ProcessingState     string               `json:"processingState"`
	ActiveVersionNumber *int                 `json:"activeVersionNumber,omitempty"`
	IsStale             bool                 `json:"isStale"`
	LatestJob           *MemoryProcessingJob `json:"latestJob,omitempty"`
}

// ─── Memory State enum ───────────────────────────────────────────────────────

// MemoryState represents the 10 visible processing lifecycle states.
type MemoryState string

const (
	MemoryStateNotProcessed        MemoryState = "not_processed"
	MemoryStateQueued              MemoryState = "queued"
	MemoryStateProcessing          MemoryState = "processing"
	MemoryStateCompleted           MemoryState = "completed"
	MemoryStateCompletedInsufficient MemoryState = "completed_with_insufficient_evidence"
	MemoryStateFailed              MemoryState = "failed"
	MemoryStateStale               MemoryState = "stale"
	MemoryStateReprocessing        MemoryState = "reprocessing"
	MemoryStateRetrying            MemoryState = "retrying"
	MemoryStateRetryExhausted      MemoryState = "retry_exhausted"
)

// ─── API Request/Response types ─────────────────────────────────────────────

// ReprocessRequest is the POST /meetings/{id}/memory/reprocess request body.
type ReprocessRequest struct {
	ExcludedPriorMemoryIDs []uuid.UUID `json:"excludedPriorMemoryIds,omitempty"`
	IncludedPriorMemoryIDs []uuid.UUID `json:"includedPriorMemoryIds,omitempty"`
	Reason                string      `json:"reason,omitempty"`
}

// ReprocessAccepted is the 202 response to a reprocess request.
type ReprocessAccepted struct {
	JobID         uuid.UUID `json:"jobId"`
	CorrelationID uuid.UUID `json:"correlationId"`
}

// ConflictResolveRequest is the POST /meetings/{id}/memory/conflicts/{id}/resolve body.
type ConflictResolveRequest struct {
	ResolutionNote string `json:"resolution_note"`
}

// MemoryVersionSummary is the minimal info shown in the version list.
type MemoryVersionSummary struct {
	VersionNumber int        `json:"version_number"`
	IsActive      bool      `json:"isActive"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     *uuid.UUID `json:"createdBy,omitempty"`
}

// MemoryVersionList is the GET /meetings/{id}/memory/versions response.
type MemoryVersionList struct {
	MeetingID uuid.UUID              `json:"meetingId"`
	Versions  []MemoryVersionSummary `json:"versions"`
}

// ConflictList is the GET /meetings/{id}/memory/conflicts response.
type ConflictList struct {
	MeetingID uuid.UUID                `json:"meetingId"`
	Conflicts []MemoryConflictSummary `json:"conflicts"`
}

// MemoryConflictSummary is the conflict info shown in the review queue.
type MemoryConflictSummary struct {
	ID               uuid.UUID     `json:"id"`
	Category         string        `json:"category"`
	QualityStatus    QualityStatus `json:"quality_status"`
	ReviewStatus     ReviewStatus  `json:"review_status"`
	CreatedAt        time.Time     `json:"createdAt"`
	ConflictingItems ConflictPair  `json:"conflicting_items"`
	ResolutionNote   *string       `json:"resolution_note,omitempty"`
	ResolvedAt       *time.Time    `json:"resolved_at,omitempty"`
}

// PriorMemorySummary is the minimal prior memory info shown to users.
type PriorMemorySummary struct {
	MeetingID       uuid.UUID       `json:"meetingId"`
	Title           string          `json:"title"`
	CompletedAt     time.Time       `json:"completedAt"`
	MatchConfidence MatchConfidence `json:"matchConfidence"`
}

// MemoryResponse is the GET /meetings/{id}/memory response.
type MemoryResponse struct {
	MeetingID         uuid.UUID          `json:"meetingId"`
	VersionNumber     int                `json:"versionNumber"`
	Status            string             `json:"status"`
	ProcessingState   string             `json:"processingState"`
	IsStale           bool               `json:"isStale"`
	IsBriefingReady   bool               `json:"isBriefingReady"`
	BriefingReadiness BriefingReadiness  `json:"briefing_readiness"`
	Content           MemoryContent      `json:"content"`
	PriorMemoriesUsed []PriorMemorySummary `json:"prior_memories_used,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
}
