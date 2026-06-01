package briefing

import (
	"time"

	"github.com/google/uuid"
)

// ─── Preparation Status (preparation_status field) ───────────────────────────

// PreparationStatus is the runtime state of a briefing version.
type PreparationStatus string

const (
	PreparationStatusGenerating    PreparationStatus = "generating"
	PreparationStatusReady         PreparationStatus = "ready"
	PreparationStatusReadyCaveats  PreparationStatus = "ready_with_caveats"
	PreparationStatusNoPriorMemory PreparationStatus = "no_prior_memory"
	PreparationStatusStale         PreparationStatus = "stale"
	PreparationStatusFailed       PreparationStatus = "failed"
	PreparationStatusRegenerating PreparationStatus = "regenerating"
)

// ─── Briefing Result (result field) ─────────────────────────────────────────

// BriefingResult is the terminal result of a briefing generation attempt.
type BriefingResult string

const (
	BriefingResultReady         BriefingResult = "ready"
	BriefingResultReadyCaveats  BriefingResult = "ready_with_caveats"
	BriefingResultNoPriorMemory BriefingResult = "no_prior_memory"
	BriefingResultFailed        BriefingResult = "failed"
)

// ─── Briefing Version Status ───────────────────────────────────────────────────

// BriefingVersionStatus is the lifecycle status of a briefing version.
type BriefingVersionStatus string

const (
	BriefingVersionStatusActive    BriefingVersionStatus = "active"
	BriefingVersionStatusSuperseded BriefingVersionStatus = "superseded"
)

// ─── Trigger Type ─────────────────────────────────────────────────────────────

// BriefingTriggerType is what caused the briefing to be generated.
type BriefingTriggerType string

const (
	BriefingTriggerAuto           BriefingTriggerType = "auto"
	BriefingTriggerManualRegenerate BriefingTriggerType = "manual_regenerate"
)

// ─── Briefing Processing Job ─────────────────────────────────────────────────

// BriefingProcessingJob represents a queued/processing briefing generation job.
type BriefingProcessingJob struct {
	ID               uuid.UUID           `json:"id"`
	UpcomingMeetingID uuid.UUID           `json:"upcomingMeetingId"`
	TriggerType      BriefingTriggerType `json:"triggerType"`
	Status           JobStatus           `json:"status"`
	RetryCount       int                 `json:"retryCount"`
	MaxRetries       int                 `json:"maxRetries"`
	FailureReason    *string             `json:"failureReason,omitempty"`
	PreviousVersionID *uuid.UUID          `json:"previousVersionId,omitempty"`
	CorrelationID    uuid.UUID           `json:"correlationId"`
	QueuedAt         time.Time           `json:"queuedAt"`
	StartedAt        *time.Time          `json:"startedAt,omitempty"`
	CompletedAt      *time.Time          `json:"completedAt,omitempty"`
	NextRetryAt      *time.Time          `json:"nextRetryAt,omitempty"`
}

// JobStatus represents the processing lifecycle state of a briefing job.
type JobStatus string

const (
	JobStatusQueued         JobStatus = "queued"
	JobStatusProcessing    JobStatus = "processing"
	JobStatusCompleted     JobStatus = "completed"
	JobStatusFailed        JobStatus = "failed"
	JobStatusRetrying      JobStatus = "retrying"
	JobStatusRetryExhausted JobStatus = "retry_exhausted"
)

// FailureClass classifies failures for metrics.
type FailureClass string

const (
	FailureClassNone                    FailureClass = "none"
	FailureClassUpstreamError           FailureClass = "upstream_error"
	FailureClassTimeout                 FailureClass = "timeout"
	FailureClassInternalError           FailureClass = "internal_error"
)

// ─── Briefing Version ──────────────────────────────────────────────────────────

// BriefingVersion represents a single briefing version.
type BriefingVersion struct {
	ID                     uuid.UUID           `json:"id"`
	UpcomingMeetingID     uuid.UUID           `json:"upcomingMeetingId"`
	VersionNumber         int                `json:"versionNumber"`
	Status               BriefingVersionStatus `json:"status"`
	IsActive              bool               `json:"isActive"`
	Result               BriefingResult     `json:"result"`
	PreparationStatus     PreparationStatus  `json:"preparationStatus"`
	Content              BriefingContent    `json:"content"`
	SourceCount          int                `json:"sourceCount"`
	SourceMemoryVersionIDs []uuid.UUID       `json:"sourceMemoryVersionIds"`
	CreatedAt            time.Time          `json:"createdAt"`
	TriggerType          BriefingTriggerType `json:"triggerType"`
}

// ─── Briefing Content (JSONB) ──────────────────────────────────────────────────

// BriefingContent is the full structured briefing output stored in briefing_versions.content.
type BriefingContent struct {
	ConciseSummary    ConciseSummary         `json:"concise_summary"`
	DetailedSections DetailedSections       `json:"detailed_sections"`
	Sources          []SourceMeeting         `json:"sources"`
	NoPriorMemoryShell *NoPriorMemoryShell  `json:"no_prior_memory_shell,omitempty"`
}

// ConciseSummary is the top-level scannable briefing summary.
type ConciseSummary struct {
	Objective          string `json:"objective"`
	PreparationStatus string `json:"preparation_status"`
	RecommendedFocus  string `json:"recommended_focus"`
	TopPriorContext   string `json:"top_prior_context"`
	OpenActions       string `json:"open_actions"`
	RisksQuestions    string `json:"risks_questions"`
}

// DetailedSections are the expandable detail sections.
type DetailedSections struct {
	PreviousRelevantContext SectionItem `json:"previous_relevant_context"`
	ImportantPriorDecisions SectionWithItems `json:"important_prior_decisions"`
	OpenActionItems         SectionWithItems `json:"open_action_items"`
	UnresolvedRisksBlockers SectionWithItems `json:"unresolved_risks_blockers"`
	OpenQuestions           SectionWithItems `json:"open_questions"`
	StakeholderNotes       SectionWithItems `json:"stakeholder_notes"`
	SuggestedQuestions     StringArray      `json:"suggested_questions"`
	SuggestedAgenda        StringArray      `json:"suggested_agenda"`
}

// SectionItem is a briefing section with a statement and quality status.
type SectionItem struct {
	Statement     string `json:"statement,omitempty"`
	QualityStatus string `json:"quality_status"`
}

// SectionWithItems is a briefing section with a list of items and quality status.
type SectionWithItems struct {
	Items []interface{} `json:"items"`
	QualityStatus string `json:"quality_status"`
}

// StringArray is a simple string array wrapper for JSONB serialization.
type StringArray struct {
	Items []string `json:"items"`
}

// SourceMeeting is a source meeting with relatedness explanation.
type SourceMeeting struct {
	SourceMeetingID    uuid.UUID `json:"source_meeting_id"`
	RelatednessReasons []string `json:"relatedness_reasons"`
	QualityStatus      string   `json:"quality_status"`
}

// NoPriorMemoryShell is the briefing generated when no qualifying sources exist.
type NoPriorMemoryShell struct {
	Generated            bool     `json:"generated"`
	Objective            string   `json:"objective"`
	RecommendedFocus    string   `json:"recommended_focus"`
	SuggestedPrepQuestions []string `json:"suggested_prep_questions"`
}

// ─── Briefing Source Exclusion ─────────────────────────────────────────────────

// BriefingSourceExclusion represents an active or restored source exclusion.
type BriefingSourceExclusion struct {
	ID                     uuid.UUID  `json:"id"`
	UpcomingMeetingID     uuid.UUID  `json:"upcomingMeetingId"`
	ExcludedSourceMeetingID uuid.UUID `json:"excludedSourceMeetingId"`
	CreatedAt             time.Time  `json:"createdAt"`
	RestoredAt            *time.Time `json:"restoredAt,omitempty"`
}

// ─── API Request/Response Types ──────────────────────────────────────────────

// BriefingResponse is the GET /upcoming/{meetingId}/briefing response.
type BriefingResponse struct {
	MeetingID          uuid.UUID          `json:"meetingId"`
	VersionNumber      int                `json:"versionNumber"`
	Status             string             `json:"status"`
	PreparationStatus  PreparationStatus  `json:"preparationStatus"`
	IsStale            bool               `json:"isStale"`
	Content            *BriefingContent   `json:"content,omitempty"`
	SourceCount        int                `json:"sourceCount"`
	CreatedAt          time.Time          `json:"createdAt"`
	TriggerType        BriefingTriggerType `json:"triggerType"`
}

// BriefingVersionSummary is the minimal info shown in version list.
type BriefingVersionSummary struct {
	VersionNumber int                   `json:"version_number"`
	IsActive      bool                  `json:"isActive"`
	Status        string                `json:"status"`
	Result        string                `json:"result"`
	CreatedAt     time.Time             `json:"createdAt"`
	TriggerType   BriefingTriggerType   `json:"triggerType"`
}

// BriefingVersionList is the GET /upcoming/{meetingId}/briefing/versions response.
type BriefingVersionList struct {
	MeetingID uuid.UUID                `json:"meetingId"`
	Versions  []BriefingVersionSummary `json:"versions"`
}

// SourceExclusionResponse is the GET /upcoming/{meetingId}/briefing/sources response.
type SourceExclusionResponse struct {
	UpcomingMeetingID uuid.UUID                `json:"upcomingMeetingId"`
	Excluded          []ExclusionEntry        `json:"excluded"`
	Restored          []ExclusionEntry        `json:"restored"`
}

// ExclusionEntry is a single exclusion entry in the response.
type ExclusionEntry struct {
	SourceMeetingID uuid.UUID  `json:"sourceMeetingId"`
	CreatedAt       time.Time `json:"createdAt"`
	RestoredAt      *time.Time `json:"restoredAt,omitempty"`
}

// RegenerateAccepted is the 202 response to a regenerate request.
type RegenerateAccepted struct {
	JobID         uuid.UUID `json:"jobId"`
	CorrelationID uuid.UUID `json:"correlationId"`
}

// ExclusionAccepted is the response to an exclude/restore request.
type ExclusionAccepted struct {
	SourceMeetingID uuid.UUID  `json:"sourceMeetingId"`
	RestoredAt     *time.Time `json:"restoredAt,omitempty"`
}