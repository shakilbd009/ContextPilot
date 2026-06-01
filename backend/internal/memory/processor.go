package memory

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// ErrInsufficientEvidence is returned by the processor when a category
// cannot be populated due to missing source material.
var ErrInsufficientEvidence = errors.New("insufficient evidence for this category")

// ErrProviderUnavailable is a transient error indicating the processing provider is down.
var ErrProviderUnavailable = errors.New("processing provider unavailable")

// ErrMalformedInput is a permanent failure for invalid source content.
var ErrMalformedInput = errors.New("malformed meeting input")

// MemoryProcessor is the interface for the meeting memory processing engine.
// Concrete implementations are injected (ADR-0005, FR-17).
type MemoryProcessor interface {
	// Process extracts structured memory from meeting source material and prior memories.
	// Returns the structured MemoryContent or an error. Errors are classified as
	// transient (retryable) or permanent (fail immediately).
	Process(ctx context.Context, input MemoryProcessorInput) (*MemoryProcessorOutput, error)
	// Name returns the provider name for observability.
	Name() string
}

// MemoryProcessorInput contains everything the processor needs to generate memory.
type MemoryProcessorInput struct {
	MeetingID        uuid.UUID
	Transcript       *string
	Notes            *string
	Title            string
	Participants     []ParticipantInfo
	PriorMemories    []PriorMemoryInfo
	CorrelationID    uuid.UUID
}

// ParticipantInfo is the subset of participant data exposed to the processor.
type ParticipantInfo struct {
	ID           uuid.UUID
	DisplayName  string
	Email        *string
	Organization *string
}

// PriorMemoryInfo is the minimal prior memory content needed for matching/conflict detection.
type PriorMemoryInfo struct {
	MemoryVersionID uuid.UUID
	MeetingID       uuid.UUID
	Title           string
	CompletedAt     string
	Decisions       []PriorMemoryItem
	ActionItems     []PriorMemoryItem
	RisksBlockers   []PriorMemoryItem
	Summary         *string
}

// PriorMemoryItem is a single item from a prior memory for conflict detection.
type PriorMemoryItem struct {
	ID             string
	Statement      string
	QualityStatus  QualityStatus
	Category       string
}

// MemoryProcessorOutput is what the processor returns for storage.
type MemoryProcessorOutput struct {
	Content          MemoryContent
	Evidence         []MemoryEvidence
	Conflicts        []ConflictPair
	ProcessingNotes  string // internal debugging, never exposed externally
}

// DefaultMemoryProcessor is the stub implementation used when no real AI provider is configured.
// It generates a minimal structured memory with insufficient_evidence for all categories
// and logs the processing event. This lets the queue/worker/repo stack be exercised
// end-to-end while the AI provider is developed separately.
type DefaultMemoryProcessor struct{}

// Name implements MemoryProcessor.
func (DefaultMemoryProcessor) Name() string { return "default-stub" }

// Process implements MemoryProcessor.
// It produces a minimal memory with insufficient_evidence for all categories
// to demonstrate the full pipeline without requiring a real AI provider.
func (DefaultMemoryProcessor) Process(ctx context.Context, input MemoryProcessorInput) (*MemoryProcessorOutput, error) {
	// Build evidence from transcript (first 200 chars as snippet if available)
	var summaryStatement string = "Insufficient meeting content for analysis."
	var summaryQuality QualityStatus = QualityInsufficient
	var snippetText string

	if input.Transcript != nil && len(*input.Transcript) > 0 {
		runes := []rune(*input.Transcript)
		if len(runes) > 200 {
			snippetText = string(runes[:200])
		} else {
			snippetText = *input.Transcript
		}
		summaryStatement = "Meeting processed. Transcript available for analysis."
		summaryQuality = QualityWeak
	} else if input.Notes != nil && len(*input.Notes) > 0 {
		runes := []rune(*input.Notes)
		if len(runes) > 200 {
			snippetText = string(runes[:200])
		} else {
			snippetText = *input.Notes
		}
		summaryStatement = "Meeting processed. Notes available for analysis."
		summaryQuality = QualityWeak
	}

	// Build evidence record if we have content
	var evidence []MemoryEvidence
	if snippetText != "" {
		evidence = []MemoryEvidence{
			{
				ID:              uuid.New(),
				SourceType:      EvidenceSourceTranscript,
				SourceLocation:  SourceLocation{Type: "char_offset", Start: 0, End: len(snippetText)},
				EvidenceSnippet: snippetText,
				QualityStatus:   summaryQuality,
			},
		}
	}

	content := MemoryContent{
		Summary: CategorySummary{
			Statement:     summaryStatement,
			QualityStatus: summaryQuality,
			Items:         []interface{}{},
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
			InsufficientCategories: []string{
				"summary", "decisions", "action_items", "risks_blockers", "open_questions",
			},
		},
	}

	// Encode notes into evidence
	if input.Notes != nil && len(*input.Notes) > 0 {
		runes := []rune(*input.Notes)
		end := len(runes)
		if end > 200 {
			end = 200
		}
		evidence = append(evidence, MemoryEvidence{
			ID:              uuid.New(),
			SourceType:      EvidenceSourceNotes,
			SourceLocation:  SourceLocation{Type: "char_offset", Start: 0, End: end},
			EvidenceSnippet: string(runes[:end]),
			QualityStatus:   QualityWeak,
		})
	}

	return &MemoryProcessorOutput{
		Content:  content,
		Evidence:  evidence,
		Conflicts: nil,
	}, nil
}

// ConflictDetector checks whether two memory items in the same category conflict.
// Used by the service layer to detect conflicts before calling the processor.
type ConflictDetector struct{}

// NewConflictDetector returns a new ConflictDetector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflicts compares current and prior memory items and returns
// conflicting pairs where both items have strong or weak evidence and
// are semantically contradictory.
func (d *ConflictDetector) DetectConflicts(
	currentItems []PriorMemoryItem,
	priorItems []PriorMemoryItem,
	category string,
) []ConflictPair {
	var pairs []ConflictPair

	for _, cur := range currentItems {
		if cur.QualityStatus == QualityInsufficient {
			continue
		}
		for _, pri := range priorItems {
			if pri.QualityStatus == QualityInsufficient {
				continue
			}
			if d.itemsConflict(cur.Statement, pri.Statement) {
				pairs = append(pairs, ConflictPair{
					Current: []ConflictItemContent{
						{ID: cur.ID, Statement: cur.Statement, QualityStatus: cur.QualityStatus},
					},
					Prior: []ConflictItemContent{
						{ID: pri.ID, Statement: pri.Statement, QualityStatus: pri.QualityStatus},
					},
				})
			}
		}
	}
	return pairs
}

// itemsConflict returns true when two statements are semantically contradictory.
// This is a stub: in production this would call a semantic similarity model.
// For the stub, we flag simple negation patterns and opposing action words.
func (d *ConflictDetector) itemsConflict(a, b string) bool {
	// Stub conflict detection: check for explicit negation patterns.
	// In production, this would use the provider's semantic similarity API.
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)
	if aLower == bLower {
		return false
	}
	// Check for shared negation word (both contain the same strong negation marker)
	for _, neg := range negationWords {
		if strings.Contains(aLower, neg) && strings.Contains(bLower, neg) {
			return true
		}
	}
	// Check for opposing patterns: presence vs absence of a key action word
	// e.g. "not proceed" vs "proceed" — "cancelled" vs "approved"
	for _, opp := range oppositionPairs {
		hasA := strings.Contains(aLower, opp.negative) || strings.Contains(aLower, opp.positive)
		hasB := strings.Contains(bLower, opp.negative) || strings.Contains(bLower, opp.positive)
		if hasA && hasB {
			// One has the negative form, the other has the positive form
			aHasNeg := strings.Contains(aLower, opp.negative)
			bHasNeg := strings.Contains(bLower, opp.negative)
			if aHasNeg != bHasNeg {
				return true
			}
		}
	}
	return false
}

var negationWords = []string{
	"not proceed", "won't proceed", "will not proceed",
	"cancel", "cancelled", "termination",
}

type oppositionPair struct {
	negative string // "not proceed", "cancelled"
	positive string // "proceed", "approved"
}

var oppositionPairs = []oppositionPair{
	{"not proceed", "proceed"},
	{"won't proceed", "will proceed"},
	{"cancelled", "approved"},
	{"not approve", "approve"},
	{"won't approve", "will approve"},
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}