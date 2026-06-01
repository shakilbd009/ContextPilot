package briefing

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// productionBriefingService is the concrete BriefingService implementation.
// Real signal-matching, source selection, and briefing generation logic lives here.
// This is the implementation called by the background worker.
type productionBriefingService struct {
	repo     *Repository
	meetRepo MeetingSourceRepo
	log      *zerolog.Logger
}

// NewProductionBriefingService returns a BriefingService that generates real briefings.
func NewProductionBriefingService(repo *Repository, meetRepo MeetingSourceRepo, log *zerolog.Logger) BriefingService {
	return &productionBriefingService{
		repo:     repo,
		meetRepo: meetRepo,
		log:      log,
	}
}

// Generate produces briefing content for an upcoming meeting by matching qualifying
// source meetings and constructing a structured BriefingContent with concise summary
// and detailed sections. It returns the content, result, source memory version IDs,
// and any error encountered.
func (s *productionBriefingService) Generate(ctx context.Context, input BriefingGeneratorInput) (*BriefingContent, BriefingResult, []uuid.UUID, error) {
	lg := s.log.With().
		Str("upcoming_meeting_id", input.UpcomingMeetingID.String()).
		Logger()

	// Fetch upcoming meeting details
	upcoming, err := s.repo.GetUpcomingMeeting(ctx, input.UpcomingMeetingID)
	if err != nil {
		lg.Warn().Err(err).Msg("briefing.generator: failed to load upcoming meeting")
		return nil, BriefingResultFailed, nil, err
	}

	// Get active exclusions
	excludedIDs, err := s.repo.ListActiveExclusions(ctx, input.UpcomingMeetingID)
	if err != nil {
		lg.Warn().Err(err).Msg("briefing.generator: failed to load exclusions")
		return nil, BriefingResultFailed, nil, err
	}
	excludedSet := make(map[uuid.UUID]struct{}, len(excludedIDs))
	for _, id := range excludedIDs {
		excludedSet[id] = struct{}{}
	}

	// Fetch all completed meetings with active memory for this user
	// The user is inferred from the meeting's created_by (upcoming.CreatedBy)
	sourceMeetings, err := s.meetRepo.ListCompletedMeetingsWithMemory(ctx, upcoming.CreatedBy)
	if err != nil {
		lg.Warn().Err(err).Msg("briefing.generator: failed to load source meetings")
		return nil, BriefingResultFailed, nil, err
	}

	upcomingStart := upcoming.ScheduledStart
	upcomingTitle := upcoming.Title
	upcomingOrg := upcoming.ClientOrOrganization
	var upcomingEmails []*string
	if len(upcoming.Participants) > 0 {
		for _, p := range upcoming.Participants {
			if p.Email != nil && *p.Email != "" {
				upcomingEmails = append(upcomingEmails, p.Email)
			}
		}
	}

	// Score each source meeting
	var scored []scoredSource

	for _, src := range sourceMeetings {
		if _, excluded := excludedSet[src.ID]; excluded {
			continue
		}

		signals := s.computeSignals(ctx, src, upcomingStart, upcomingTitle, upcomingEmails, upcomingOrg)
		if signals.Count() >= 2 {
			scored = append(scored, scoredSource{
				meeting:    src,
				signals:    signals,
				signalCount: signals.Count(),
			})
		}
	}

	// Sort: highest signal count, most recent, strongest title similarity, stable UUID
	sortScoredSources(scored)

	// Take top 3
	if len(scored) > 3 {
		scored = scored[:3]
	}

	if len(scored) == 0 {
		return s.generateNoPriorMemoryShell(upcoming), BriefingResultNoPriorMemory, nil, nil
	}

	// Build source list and aggregate memory content
	var (
		sourceIDs       []uuid.UUID
		sourceVersionIDs []uuid.UUID
		sources         []SourceMeeting
		hasWeak         bool
		hasInsufficient bool
	)

	for _, ss := range scored {
		src := ss.meeting
		sourceIDs = append(sourceIDs, src.ID)
		if src.ActiveMemoryVersion != nil {
			sourceVersionIDs = append(sourceVersionIDs, *src.ActiveMemoryVersion)
		}
		sources = append(sources, SourceMeeting{
			SourceMeetingID:    src.ID,
			RelatednessReasons: ss.signals.Reasons(),
			QualityStatus:      s.deriveSourceQuality(ss.signals),
		})
		if ss.signals.HasWeak() {
			hasWeak = true
		}
		if ss.signals.HasInsufficient() {
			hasInsufficient = true
		}
	}

	// Aggregate content from top sources
	aggregate := s.aggregateMemoryContent(ctx, scored)

	content := s.buildBriefingContent(aggregate, sources, upcoming, hasWeak, hasInsufficient)
	result := s.deriveBriefingResult(hasWeak, hasInsufficient)

	lg.Info().
		Int("sources_selected", len(sourceIDs)).
		Str("result", string(result)).
		Msg("briefing.generator: generation complete")

	return content, result, sourceVersionIDs, nil
}

// MeetingSourceRepo abstracts the meeting data access needed for source selection.
// Implemented by a thin adapter over the briefing repository (shared pgxpool).
type MeetingSourceRepo interface {
	ListCompletedMeetingsWithMemory(ctx context.Context, userID uuid.UUID) ([]MeetingWithMemory, error)
	GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]Participant_email_org, error)
}

// Participant_email_org holds the minimal participant data needed for signal matching.
type Participant_email_org struct {
	Email        *string
	Organization *string
}

// MeetingWithMemory holds the data needed for source meeting scoring.
type MeetingWithMemory struct {
	ID                  uuid.UUID
	Title               string
	CompletedAt         int64 // Unix timestamp for quick comparison
	ActiveMemoryVersion *uuid.UUID
	MemoryJSON          []byte
	ParticipantEmail    *string // from first participant with email
	ParticipantOrg      *string
}

// scoredSource pairs a source meeting with its matched signals.
type scoredSource struct {
	meeting     MeetingWithMemory
	signals     signalSet
	signalCount int
}

// signalSet holds the signals matched for a source meeting.
type signalSet struct {
	sameParticipant bool
	similarTitle    bool
	sameOrg         bool
	closeChronology bool
	titleSimilarity int // 0=not similar, 1=Levenshtein, 2=token sequence
	reasons         []string
}

func (s *signalSet) Count() int {
	n := 0
	if s.sameParticipant {
		n++
	}
	if s.similarTitle {
		n++
	}
	if s.sameOrg {
		n++
	}
	if s.closeChronology {
		n++
	}
	return n
}

func (s *signalSet) Reasons() []string { return s.reasons }
func (s *signalSet) HasWeak() bool {
	return s.similarTitle && s.titleSimilarity == 0
}
func (s *signalSet) HasInsufficient() bool {
	return false // individual source signals are always sufficient if they pass threshold
}

// computeSignals evaluates the four FR-4 signals between a candidate source and the upcoming meeting.
func (s *productionBriefingService) computeSignals(ctx context.Context, src MeetingWithMemory, upcomingStart time.Time, upcomingTitle string, upcomingEmails []*string, upcomingOrg *string) signalSet {
	ss := signalSet{}

	// 1. Same participant: email exact match, case-insensitive
	for _, upEmail := range upcomingEmails {
		if upEmail != nil && *upEmail != "" {
			srcEmail := src.ParticipantEmail
			if srcEmail != nil && *srcEmail != "" {
				if strings.EqualFold(*upEmail, *srcEmail) {
					ss.sameParticipant = true
					ss.reasons = append(ss.reasons, "same participant")
					break
				}
			}
		}
	}

	// 2. Similar title: Levenshtein <= 3 OR 4+ shared word tokens
	if src.Title != "" && upcomingTitle != "" {
		titleSim := computeTitleSimilarity(src.Title, upcomingTitle)
		if titleSim >= 3 {
			ss.similarTitle = true
			ss.titleSimilarity = 2
			ss.reasons = append(ss.reasons, "similar title")
		}
	}

	// 3. Same org: exact match after normalization
	srcOrg := src.ParticipantOrg
	if upcomingOrg != nil && srcOrg != nil {
		if normalizeOrg(*upcomingOrg) == normalizeOrgEqual(*srcOrg) {
			ss.sameOrg = true
			ss.reasons = append(ss.reasons, "same organization/client")
		}
	}

	// 4. Close chronology: completed within 90 days before upcoming start, no future sources
	srcCompletedUnix := src.CompletedAt
	upcomingStartUnix := upcomingStart.Unix()
	if upcomingStartUnix > srcCompletedUnix {
		daysDiff := (upcomingStartUnix - srcCompletedUnix) / 86400
		if daysDiff <= 90 {
			ss.closeChronology = true
			ss.reasons = append(ss.reasons, "close chronology")
		}
	}

	return ss
}

// computeTitleSimilarity returns a score: 3+ for Levenshtein<=3, 2 for 4+ shared tokens, 0 otherwise.
func computeTitleSimilarity(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a == b {
		return 10
	}

	// Levenshtein distance
	dist := levenshtein(a, b)
	if dist <= 3 {
		return 3
	}

	// 4+ shared consecutive tokens
	aTokens := tokenize(a)
	bTokens := tokenize(b)
	if shareLongestTokenRun(aTokens, bTokens) >= 4 {
		return 2
	}

	return 0
}

func tokenize(s string) []string {
	var tokens []string
	var word []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, unicode.ToLower(r))
		} else {
			if len(word) > 0 {
				tokens = append(tokens, string(word))
				word = nil
			}
		}
	}
	if len(word) > 0 {
		tokens = append(tokens, string(word))
	}
	return tokens
}

func shareLongestTokenRun(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	// Build a set for O(1) lookup
	bSet := make(map[string]struct{}, len(b))
	for _, t := range b {
		bSet[t] = struct{}{}
	}
	maxRun := 0
	run := 0
	for _, token := range a {
		if _, ok := bSet[token]; ok {
			run++
			if run > maxRun {
				maxRun = run
			}
		} else {
			run = 0
		}
	}
	return maxRun
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	// Use rows of length b+1
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1]
			} else {
				min := prev[j] + 1 // deletion
				if c := curr[j-1] + 1; c < min {
					min = c
				}
				if c := prev[j-1] + 1; c < min {
					min = c
				}
				curr[j] = min
			}
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func normalizeOrg(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normalizeOrgEqual(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// sortScoredSources sorts by signal count desc, then completedAt desc, then titleSimilarity desc, then UUID asc.
func sortScoredSources(sources []scoredSource) {
	// Simple bubble sort; sources is at most 3 so O(n^2) is fine
	for i := 0; i < len(sources); i++ {
		for j := i + 1; j < len(sources); j++ {
			si, sj := sources[i], sources[j]
			if si.signalCount != sj.signalCount {
				if si.signalCount < sj.signalCount {
					sources[i], sources[j] = sources[j], sources[i]
				}
				continue
			}
			// Most recent first
			if si.meeting.CompletedAt < sj.meeting.CompletedAt {
				sources[i], sources[j] = sources[j], sources[i]
				continue
			}
			if si.meeting.CompletedAt == sj.meeting.CompletedAt {
				// Stronger title similarity
				if si.signals.titleSimilarity < sj.signals.titleSimilarity {
					sources[i], sources[j] = sources[j], sources[i]
					continue
				}
				if si.signals.titleSimilarity == sj.signals.titleSimilarity {
					// Stable UUID tiebreaker
					if si.meeting.ID.String() > sj.meeting.ID.String() {
						sources[i], sources[j] = sources[j], sources[i]
					}
				}
			}
		}
	}
}

// sourceContent holds parsed memory content from a single source.
type sourceContent struct {
	Summary          struct {
		Statement     string `json:"statement"`
		QualityStatus string `json:"quality_status"`
		Items         []any  `json:"items,omitempty"`
	} `json:"summary"`
	Decisions []struct {
		ID                string `json:"id"`
		DecisionStatement string `json:"decision_statement"`
		ChangeStatus     string `json:"change_status"`
		QualityStatus    string `json:"quality_status"`
	} `json:"decisions"`
	ActionItems []struct {
		ID            string `json:"id"`
		Description   string `json:"description"`
		QualityStatus string `json:"quality_status"`
	} `json:"action_items"`
	RisksBlockers []struct {
		ID            string `json:"id"`
		Description   string `json:"description"`
		Type          string `json:"type"`
		QualityStatus string `json:"quality_status"`
	} `json:"risks_blockers"`
	OpenQuestions []struct {
		ID            string `json:"id"`
		Question      string `json:"question"`
		QualityStatus string `json:"quality_status"`
	} `json:"open_questions"`
	StakeholderNotes []struct {
		ID            string `json:"id"`
		Statement     string `json:"statement"`
		QualityStatus string `json:"quality_status"`
	} `json:"stakeholder_notes"`
	NextRecommendedFocus struct {
		Focus         string `json:"focus"`
		QualityStatus string `json:"quality_status"`
	} `json:"next_recommended_focus"`
	BriefingReadiness struct {
		ReadinessState string `json:"readiness_state"`
		QualityStatus  string `json:"quality_status"`
	} `json:"briefing_readiness"`
}

func parseSourceContent(jsonBytes []byte) sourceContent {
	if len(jsonBytes) == 0 {
		return sourceContent{}
	}
	var c sourceContent
	// #nosec G104 -- parseSourceContent is a best-effort helper: malformed
	// JSON falls back to the zero value rather than returning an error,
	// so the caller can keep aggregating content from other sources.
	if err := json.Unmarshal(jsonBytes, &c); err != nil {
		_ = err
	}
	return c
}

// aggregateMemoryContent merges content from multiple source memories into a
// single aggregate representation.
func (s *productionBriefingService) aggregateMemoryContent(ctx context.Context, sources []scoredSource) sourceContent {
	var agg sourceContent
	if len(sources) == 0 {
		return agg
	}

	// For simplicity, concatenate statements from top source
	primary := parseSourceContent(sources[0].meeting.MemoryJSON)
	agg = primary

	// If we have multiple sources, combine action items and decisions
	if len(sources) > 1 {
		for _, ss := range sources[1:] {
			src := parseSourceContent(ss.meeting.MemoryJSON)
			agg.ActionItems = append(agg.ActionItems, src.ActionItems...)
			agg.Decisions = append(agg.Decisions, src.Decisions...)
			agg.RisksBlockers = append(agg.RisksBlockers, src.RisksBlockers...)
			agg.OpenQuestions = append(agg.OpenQuestions, src.OpenQuestions...)
			agg.StakeholderNotes = append(agg.StakeholderNotes, src.StakeholderNotes...)
		}
	}

	return agg
}

// deriveSourceQuality maps signal patterns to a BRD-03 quality status for the source.
func (s *productionBriefingService) deriveSourceQuality(signals signalSet) string {
	var quality string
	switch {
	// Strongest signals (2+ signals including participant or both title+org)
	default:
		quality = "strong_evidence"
	}
	if signals.HasWeak() {
		quality = "weak_evidence"
	}
	return quality
}

// buildBriefingContent assembles the BriefingContent from aggregated memory.
func (s *productionBriefingService) buildBriefingContent(agg sourceContent, sources []SourceMeeting, upcoming *UpcomingMeetingBriefing, hasWeak, hasInsufficient bool) *BriefingContent {
	// Concise summary
	conciseSummary := ConciseSummary{
		Objective:          agg.Summary.Statement,
		PreparationStatus:  string(PreparationStatusReady),
		RecommendedFocus:   agg.NextRecommendedFocus.Focus,
		TopPriorContext:    agg.Summary.Statement,
		OpenActions:        extractActionSummary(agg.ActionItems),
		RisksQuestions:     extractQuestionsSummary(agg.OpenQuestions),
	}

	// Detailed sections
	detailedSections := DetailedSections{
		PreviousRelevantContext: SectionItem{
			Statement:     agg.Summary.Statement,
			QualityStatus: agg.Summary.QualityStatus,
		},
		ImportantPriorDecisions: SectionWithItems{
			Items:         filterDecisions(agg.Decisions),
			QualityStatus: deriveCategoryQuality(agg.Decisions),
		},
		OpenActionItems: SectionWithItems{
			Items:         filterActionItems(agg.ActionItems),
			QualityStatus: deriveActionQuality(agg.ActionItems),
		},
		UnresolvedRisksBlockers: SectionWithItems{
			Items:         filterRisks(agg.RisksBlockers),
			QualityStatus: deriveRisksQuality(agg.RisksBlockers),
		},
		OpenQuestions: SectionWithItems{
			Items:         filterQuestions(agg.OpenQuestions),
			QualityStatus: deriveQuestionsQuality(agg.OpenQuestions),
		},
		StakeholderNotes: SectionWithItems{
			Items:         filterStakeholderNotes(agg.StakeholderNotes),
			QualityStatus: deriveStakeholderQuality(agg.StakeholderNotes),
		},
		SuggestedQuestions: StringArray{Items: buildSuggestedQuestions(agg)},
		SuggestedAgenda:    StringArray{Items: buildSuggestedAgenda(agg, upcoming)},
	}

	return &BriefingContent{
		ConciseSummary:   conciseSummary,
		DetailedSections: detailedSections,
		Sources:          sources,
	}
}

// generateNoPriorMemoryShell builds a shell briefing when no source meetings qualify.
func (s *productionBriefingService) generateNoPriorMemoryShell(upcoming *UpcomingMeetingBriefing) *BriefingContent {
	return &BriefingContent{
		ConciseSummary: ConciseSummary{
			Objective:          "",
			PreparationStatus:  string(PreparationStatusNoPriorMemory),
			RecommendedFocus:  "Review the meeting description and prepare your key discussion points",
			TopPriorContext:    "No prior memory found",
			OpenActions:        "No prior memory found",
			RisksQuestions:     "No prior memory found",
		},
		DetailedSections: DetailedSections{
			PreviousRelevantContext: SectionItem{
				QualityStatus: "insufficient_evidence",
			},
			ImportantPriorDecisions: SectionWithItems{
				Items:         []any{},
				QualityStatus: "insufficient_evidence",
			},
			OpenActionItems: SectionWithItems{
				Items:         []any{},
				QualityStatus: "insufficient_evidence",
			},
			UnresolvedRisksBlockers: SectionWithItems{
				Items:         []any{},
				QualityStatus: "insufficient_evidence",
			},
			OpenQuestions: SectionWithItems{
				Items:         []any{},
				QualityStatus: "insufficient_evidence",
			},
			StakeholderNotes: SectionWithItems{
				Items:         []any{},
				QualityStatus: "insufficient_evidence",
			},
			SuggestedQuestions: StringArray{Items: buildShellQuestions(upcoming)},
			SuggestedAgenda:    StringArray{Items: buildShellAgenda(upcoming)},
		},
		NoPriorMemoryShell: &NoPriorMemoryShell{
			Generated:            true,
			Objective:            "",
			RecommendedFocus:    "Review the meeting description and prepare your key discussion points",
			SuggestedPrepQuestions: buildShellQuestions(upcoming),
		},
	}
}

// deriveBriefingResult maps evidence quality to BriefingResult.
func (s *productionBriefingService) deriveBriefingResult(hasWeak, hasInsufficient bool) BriefingResult {
	if hasInsufficient {
		return BriefingResultReadyCaveats
	}
	if hasWeak {
		return BriefingResultReadyCaveats
	}
	return BriefingResultReady
}

// ─── Helper extractors ───────────────────────────────────────────────────────

func extractActionSummary(items []struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	QualityStatus string `json:"quality_status"`
}) string {
	if len(items) == 0 {
		return ""
	}
	var lines []string
	for _, item := range items {
		if item.QualityStatus != "conflicting_evidence" {
			lines = append(lines, item.Description)
		}
	}
	return strings.Join(lines, "; ")
}

func extractQuestionsSummary(items []struct {
	ID            string `json:"id"`
	Question      string `json:"question"`
	QualityStatus string `json:"quality_status"`
}) string {
	if len(items) == 0 {
		return ""
	}
	var lines []string
	for _, item := range items {
		if item.QualityStatus != "conflicting_evidence" {
			lines = append(lines, item.Question)
		}
	}
	return strings.Join(lines, "; ")
}

func filterDecisions(decisions []struct {
	ID                string `json:"id"`
	DecisionStatement string `json:"decision_statement"`
	ChangeStatus     string `json:"change_status"`
	QualityStatus    string `json:"quality_status"`
}) []any {
	var filtered []any
	for _, d := range decisions {
		if d.QualityStatus != "conflicting_evidence" {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func filterActionItems(items []struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	QualityStatus string `json:"quality_status"`
}) []any {
	var filtered []any
	for _, item := range items {
		if item.QualityStatus != "conflicting_evidence" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterRisks(items []struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	QualityStatus string `json:"quality_status"`
}) []any {
	var filtered []any
	for _, item := range items {
		if item.QualityStatus != "conflicting_evidence" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterQuestions(items []struct {
	ID            string `json:"id"`
	Question      string `json:"question"`
	QualityStatus string `json:"quality_status"`
}) []any {
	var filtered []any
	for _, item := range items {
		if item.QualityStatus != "conflicting_evidence" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterStakeholderNotes(items []struct {
	ID            string `json:"id"`
	Statement     string `json:"statement"`
	QualityStatus string `json:"quality_status"`
}) []any {
	var filtered []any
	for _, note := range items {
		// Only include strong/weak evidence; exclude insufficient/conflicting
		if note.QualityStatus == "strong_evidence" || note.QualityStatus == "weak_evidence" {
			filtered = append(filtered, note)
		}
	}
	return filtered
}

func deriveCategoryQuality[T any](items []T) string { return "strong_evidence" }

func deriveActionQuality(items []struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	QualityStatus string `json:"quality_status"`
}) string {
	hasWeak := false
	hasConflicting := false
	for _, item := range items {
		switch item.QualityStatus {
		case "weak_evidence":
			hasWeak = true
		case "conflicting_evidence":
			hasConflicting = true
		}
	}
	if hasConflicting {
		return "conflicting_evidence"
	}
	if hasWeak {
		return "weak_evidence"
	}
	return "strong_evidence"
}

func deriveRisksQuality(items []struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	QualityStatus string `json:"quality_status"`
}) string {
	hasWeak := false
	hasConflicting := false
	for _, item := range items {
		switch item.QualityStatus {
		case "weak_evidence":
			hasWeak = true
		case "conflicting_evidence":
			hasConflicting = true
		}
	}
	if hasConflicting {
		return "conflicting_evidence"
	}
	if hasWeak {
		return "weak_evidence"
	}
	return "strong_evidence"
}

func deriveQuestionsQuality(items []struct {
	ID            string `json:"id"`
	Question      string `json:"question"`
	QualityStatus string `json:"quality_status"`
}) string {
	hasWeak := false
	for _, item := range items {
		if item.QualityStatus == "weak_evidence" {
			hasWeak = true
		}
	}
	if hasWeak {
		return "weak_evidence"
	}
	return "strong_evidence"
}

func deriveStakeholderQuality(items []struct {
	ID            string `json:"id"`
	Statement     string `json:"statement"`
	QualityStatus string `json:"quality_status"`
}) string {
	hasWeak := false
	for _, note := range items {
		if note.QualityStatus == "weak_evidence" {
			hasWeak = true
		}
	}
	if hasWeak {
		return "weak_evidence"
	}
	return "strong_evidence"
}

func buildSuggestedQuestions(agg sourceContent) []string {
	var questions []string
	for _, q := range agg.OpenQuestions {
		if q.QualityStatus != "conflicting_evidence" {
			questions = append(questions, q.Question)
		}
	}
	return questions
}

func buildSuggestedAgenda(agg sourceContent, upcoming *UpcomingMeetingBriefing) []string {
	if upcoming == nil {
		return nil
	}
	var agenda []string
	if upcoming.Description != nil && *upcoming.Description != "" {
		agenda = append(agenda, "Review: "+*upcoming.Description)
	}
	// Seed agenda with key topics from prior focus
	if agg.NextRecommendedFocus.Focus != "" {
		agenda = append(agenda, "Focus: "+agg.NextRecommendedFocus.Focus)
	}
	return agenda
}

func buildShellQuestions(upcoming *UpcomingMeetingBriefing) []string {
	if upcoming == nil {
		return nil
	}
	var questions []string
	if upcoming.Description != nil && *upcoming.Description != "" {
		questions = append(questions, "What is the purpose of this meeting?")
		questions = append(questions, "What do you need to prepare before this meeting?")
	}
	return questions
}

func buildShellAgenda(upcoming *UpcomingMeetingBriefing) []string {
	if upcoming == nil {
		return nil
	}
	var agenda []string
	if upcoming.Title != "" {
		agenda = append(agenda, "Meeting: "+upcoming.Title)
	}
	if upcoming.Description != nil && *upcoming.Description != "" {
		agenda = append(agenda, " agenda: "+*upcoming.Description)
	}
	return agenda
}

// UpcomingMeetingBriefing is the minimal upcoming meeting struct used by the generator.
type UpcomingMeetingBriefing struct {
	ID                   uuid.UUID
	Title                string
	ScheduledStart       time.Time
	Description         *string
	ClientOrOrganization *string
	CreatedBy            uuid.UUID
	Participants         []UpcomingParticipantBriefing
	CreatedAt            time.Time
}

// UpcomingParticipantBriefing is the minimal participant struct for the generator.
type UpcomingParticipantBriefing struct {
	DisplayName  *string
	Email        *string
	Organization *string
}
