package briefing

import (

	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// contextKey is a private type to avoid context key collisions.
type contextKey struct{}

// WithRepository injects the briefing repository into the request context.
func WithRepository(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	repo := NewRepository(pool)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, repo)
			ctx = context.WithValue(ctx, poolKey{}, pool)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type poolKey struct{}

// WithWorker injects the background worker into the request context.
func WithWorker(w *Worker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, w)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getRepository(r *http.Request) *Repository {
	if v := r.Context().Value(contextKey{}); v != nil {
		if repo, ok := v.(*Repository); ok {
			return repo
		}
	}
	return nil
}

func getWorker(r *http.Request) *Worker {
	if v := r.Context().Value(contextKey{}); v != nil {
		if w, ok := v.(*Worker); ok {
			return w
		}
	}
	return nil
}

func getUserID(r *http.Request) (uuid.UUID, bool) {
	userIDStr := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userIDStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func forbidden(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"detail":"` + html.EscapeString(detail) + `"}`))
}

func notFound(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"type":"about:blank","title":"Not Found","status":404,"detail":"` + html.EscapeString(detail) + `"}`))
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401,"detail":"Authentication required."}`))
}

func conflict(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	w.Write([]byte(`{"type":"about:blank","title":"Conflict","status":409,"detail":"` + html.EscapeString(detail) + `"}`))
}

func internalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500}`))
}

// Handler returns a chi router with briefing API routes.
func Handler(log *zerolog.Logger, pool *pgxpool.Pool, worker *Worker) http.Handler {
	r := chi.NewRouter()

	r.Use(WithRepository(pool))
	if worker != nil {
		r.Use(WithWorker(worker))
	}

	// Feature flag gate + auth guard for all routes
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverEnv := os.Getenv(FeatureFlagEnv)
				enabled := IsFeatureFlagEnabled()
				if !enabled {
					if detectFlagMisconfiguration(r, serverEnv) {
						emitFlagMisconfiguration(log, r, FeatureFlagEnv, serverEnv)
					}
					forbidden(w, "Pre-call briefing is not enabled. Set FF_ENABLE_PRE_CALL_BRIEFING=true to activate.")
					return
				}
				userID, ok := getUserID(r)
				if !ok {
					unauthorized(w)
					return
				}
				ctx := context.WithValue(r.Context(), contextKey{}, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		// GET /upcoming/{meetingId}/briefing
		r.Get("/upcoming/{meetingId}/briefing", handleGetBriefing(log))
		// GET /upcoming/{meetingId}/briefing/versions
		r.Get("/upcoming/{meetingId}/briefing/versions", handleListBriefingVersions(log))
		// GET /upcoming/{meetingId}/briefing/versions/{versionNumber}
		r.Get("/upcoming/{meetingId}/briefing/versions/{versionNumber}", handleGetBriefingVersion(log))
		// POST /upcoming/{meetingId}/briefing/regenerate
		r.Post("/upcoming/{meetingId}/briefing/regenerate", handleRegenerateBriefing(log))
		// GET /upcoming/{meetingId}/briefing/sources — FR-17: excluded/restored source visibility
		r.Get("/upcoming/{meetingId}/briefing/sources", handleGetBriefingSources(log))
		// POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude
		r.Post("/upcoming/{meetingId}/briefing/sources/{sourceId}/exclude", handleExcludeSource(log))
		// POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore
		r.Post("/upcoming/{meetingId}/briefing/sources/{sourceId}/restore", handleRestoreSource(log))
	})

	return r
}

// handleGetBriefingSources handles GET /upcoming/{meetingId}/briefing/sources
// FR-17: returns excluded and restored source meetings for the briefing.
func handleGetBriefingSources(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		exclusions, err := repo.ListAllExclusions(r.Context(), meetingID)
		if err != nil {
			internalError(w)
			return
		}

		var excluded, restored []ExclusionEntry
		for _, e := range exclusions {
			entry := ExclusionEntry{
				SourceMeetingID: e.ExcludedSourceMeetingID,
				CreatedAt:       e.CreatedAt,
				RestoredAt:      e.RestoredAt,
			}
			if e.RestoredAt == nil {
				excluded = append(excluded, entry)
			} else {
				restored = append(restored, entry)
			}
		}
		if excluded == nil {
			excluded = []ExclusionEntry{}
		}
		if restored == nil {
			restored = []ExclusionEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SourceExclusionResponse{
			UpcomingMeetingID: meetingID,
			Excluded:          excluded,
			Restored:          restored,
		})
	}
}

func getUserIDFromContext(r *http.Request) uuid.UUID {
	if v := r.Context().Value(contextKey{}); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// handleGetBriefing handles GET /upcoming/{meetingId}/briefing
func handleGetBriefing(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		// ACL check
		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil {
			internalError(w)
			return
		}
		if !owned {
			forbidden(w, "You do not have access to this upcoming meeting")
			return
		}

		// Get active version
		version, err := repo.GetActiveBriefingVersion(r.Context(), meetingID)
		if err != nil {
			if errors.Is(err, ErrBriefingNotFound) {
				notFound(w, "No briefing found")
				return
			}
			internalError(w)
			return
}

		// Stale detection: compare source_memory_version_ids at generation vs current versions
		isStale := false
		if len(version.SourceMemoryVersionIDs) > 0 {
			// Build unique source meeting IDs from the captured memory versions
			// We need the source meeting IDs to look up current versions
			isStale = detectStaleness(r.Context(), repo, version)
		}

		// Emit flag misconfiguration if browser claims disabled while server is enabled
		// (handled at middleware level; done here for visibility)
		BriefingViewedTotal.Inc()

		log.Info().
			Str("upcoming_meeting_id", meetingID.String()).
			Int("version_number", version.VersionNumber).
			Bool("has_prior_memory", len(version.SourceMemoryVersionIDs) > 0).
			Msg("briefing.viewed")

		var prepStatus PreparationStatus
		if isStale {
			prepStatus = PreparationStatusStale
			log.Info().
				Str("upcoming_meeting_id", meetingID.String()).
				Int("version_number", version.VersionNumber).
				Int("stale_source_count", len(version.SourceMemoryVersionIDs)).
				Msg("briefing.stale.detected")
		} else {
			prepStatus = version.PreparationStatus
		}

		// Marshal content
		contentJSON, _ := json.Marshal(version.Content)
		var content *BriefingContent
		if len(contentJSON) > 0 {
			c := BriefingContent{}
			json.Unmarshal(contentJSON, &c)
			content = &c
		}

		resp := BriefingResponse{
			MeetingID:          meetingID,
			VersionNumber:      version.VersionNumber,
			Status:             string(version.Status),
			PreparationStatus:  prepStatus,
			IsStale:             isStale,
			Content:            content,
			SourceCount:        version.SourceCount,
			CreatedAt:          version.CreatedAt,
			TriggerType:        version.TriggerType,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// handleListBriefingVersions handles GET /upcoming/{meetingId}/briefing/versions
func handleListBriefingVersions(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		versions, err := repo.ListBriefingVersions(r.Context(), meetingID)
		if err != nil {
			internalError(w)
			return
		}
		if versions == nil {
			versions = []BriefingVersionSummary{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BriefingVersionList{
			MeetingID: meetingID,
			Versions:  versions,
		})
	}
}

// handleGetBriefingVersion handles GET /upcoming/{meetingId}/briefing/versions/{versionNumber}
func handleGetBriefingVersion(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		versionNumStr := chi.URLParam(r, "versionNumber")
		versionNum, err := strconv.Atoi(versionNumStr)
		if err != nil {
			notFound(w, "Invalid version number")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		version, err := repo.GetBriefingVersionByNumber(r.Context(), meetingID, versionNum)
		if err != nil {
			if errors.Is(err, ErrBriefingNotFound) || errors.Is(err, ErrVersionNotFound) {
				notFound(w, "Briefing version not found")
				return
			}
			internalError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(version)
	}
}

// handleRegenerateBriefing handles POST /upcoming/{meetingId}/briefing/regenerate
func handleRegenerateBriefing(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		// Check for active job
		hasActive, err := repo.HasActiveJob(r.Context(), meetingID)
		if err != nil {
			internalError(w)
			return
		}
		if hasActive {
			conflict(w, "A briefing generation job is already in progress")
			return
		}

		// Get previous active version if any
		var prevVersionID *uuid.UUID
		prevVersion, err := repo.GetActiveBriefingVersion(r.Context(), meetingID)
		if err == nil && prevVersion != nil {
			prevVersionID = &prevVersion.ID
		} else if err != nil && err != ErrBriefingNotFound {
			internalError(w)
			return
		}

		// Queue job
		jobID, correlationID, err := repo.QueueBriefingJob(r.Context(), meetingID, BriefingTriggerManualRegenerate, prevVersionID)
		if err != nil {
			internalError(w)
			return
		}

		BriefingProcessingQueuedTotal.WithLabelValues(string(BriefingTriggerManualRegenerate)).Inc()

		var sourceCount int
		if prevVersion != nil {
			sourceCount = prevVersion.SourceCount
		}
		log.Info().
			Str("upcoming_meeting_id", meetingID.String()).
			Int("source_count", sourceCount).
			Msg("briefing.regenerate.requested")

		log.Info().
			Str("job_id", jobID.String()).
			Str("upcoming_meeting_id", meetingID.String()).
			Msg("briefing regeneration queued")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(RegenerateAccepted{
			JobID:         jobID,
			CorrelationID: correlationID,
		})
	}
}

// handleExcludeSource handles POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude
func handleExcludeSource(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		sourceIDStr := chi.URLParam(r, "sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			notFound(w, "Invalid source meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		// ACL: user must own the upcoming meeting
		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		// ACL: user must also be able to access the source meeting
		canAccess, err := repo.CanAccessMeeting(r.Context(), sourceID, userID)
		if err != nil || !canAccess {
			if !canAccess {
				forbidden(w, "You do not have access to this source meeting")
				return
			}
			internalError(w)
			return
		}

		// Upsert exclusion (idempotent)
		if err := repo.UpsertExclusion(r.Context(), meetingID, sourceID); err != nil {
			internalError(w)
			return
		}

		BriefingSourceExcludedTotal.Inc()
		log.Info().
			Str("upcoming_meeting_id", meetingID.String()).
			Str("source_meeting_id", sourceID.String()).
			Msg("briefing.source.excluded")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ExclusionAccepted{
			SourceMeetingID: sourceID,
		})
	}
}

// handleRestoreSource handles POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore
func handleRestoreSource(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "meetingId")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		sourceIDStr := chi.URLParam(r, "sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			notFound(w, "Invalid source meeting ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.UpcomingMeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this upcoming meeting")
				return
			}
			internalError(w)
			return
		}

		if err := repo.RestoreExclusion(r.Context(), meetingID, sourceID); err != nil {
			if errors.Is(err, ErrExclusionNotFound) {
				notFound(w, "No active exclusion found for this source meeting")
				return
			}
			internalError(w)
			return
		}

		restoredAt := time.Now()
		BriefingSourceRestoredTotal.Inc()
		log.Info().
			Str("upcoming_meeting_id", meetingID.String()).
			Str("source_meeting_id", sourceID.String()).
			Msg("briefing.source.restored")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ExclusionAccepted{
			SourceMeetingID: sourceID,
			RestoredAt:      &restoredAt,
		})
	}
}

// detectStaleness compares the source_memory_version_ids captured at generation time
// against the current source memory versions. Returns true if any source has changed.
// Per ADR-0011: staleness is derived at read time by comparing the captured memory
// version IDs against current active memory version IDs for each source meeting.
func detectStaleness(ctx context.Context, repo *Repository, version *BriefingVersion) bool {
	if len(version.SourceMemoryVersionIDs) == 0 {
		return false
	}
	// Build a set of captured version IDs for fast lookup
	captured := make(map[uuid.UUID]struct{}, len(version.SourceMemoryVersionIDs))
	for _, id := range version.SourceMemoryVersionIDs {
		captured[id] = struct{}{}
	}
	// Look up current versions by the captured IDs themselves
	// (source_memory_version_ids stores memory version UUIDs directly)
	currentVersions, err := repo.GetCurrentSourceMemoryVersionIDsByMemoryID(ctx, version.SourceMemoryVersionIDs)
	if err != nil {
		// If we can't check, assume not stale (graceful degradation)
		return false
	}
	// Compare: any captured ID that is no longer current means stale
	for _, id := range version.SourceMemoryVersionIDs {
		currentID, ok := currentVersions[id]
		if !ok {
			// Source memory version no longer exists or was superseded — stale
			return true
		}
		// If the current active version differs from what was captured, it's stale
		if currentID != id {
			return true
		}
	}
	return false
}

// detectFlagMisconfiguration returns true if the browser is sending a request with a mismatched flag
// (browser enabled but server disabled, or vice versa). We check the X-Forwarded-Proto or
// a custom header set by the frontend.
func detectFlagMisconfiguration(r *http.Request, serverFlagValue string) bool {
	browserFlag := r.Header.Get("X-Browser-FF-Pre-Call-Briefing")
	if browserFlag == "" {
		return false
	}
	// Mismatch: server is "false" but browser sends "true", or server is "true" but browser sends "false"
	return serverFlagValue != browserFlag
}

// emitFlagMisconfiguration emits the flag misconfiguration metric and log event.
// No raw endpoint paths are logged — only the feature flag name, server/browser
// values, and a generic endpoint identifier to avoid disclosing internal API
// structure (CWE-201).
func emitFlagMisconfiguration(log *zerolog.Logger, r *http.Request, flagName, serverValue string) {
	browserValue := r.Header.Get("X-Browser-FF-Pre-Call-Briefing")
	FlagMisconfigurationTotal.WithLabelValues(serverValue, browserValue, "briefing_endpoint").Inc()
	log.Warn().
		Str("flag", flagName).
		Str("server_flag_value", serverValue).
		Str("browser_flag_value", browserValue).
		Str("endpoint", "briefing_endpoint").
		Msg("briefing.flag_misconfiguration")
}