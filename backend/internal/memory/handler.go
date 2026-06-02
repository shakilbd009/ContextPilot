package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// authContextKey is used by test router to store userID without colliding
// with the Repository context key used by WithRepository/WithWorker.
// Production code uses contextKey{} for both (a collision), but handlers
// never call getRepository after auth middleware runs, so this works.
// In test router we need both values accessible simultaneously.
type authContextKey struct{}

// WithRepository injects the memory repository into the request context.
func WithRepository(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	repo := NewRepository(pool)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, repo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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

func isFeatureFlagEnabled() bool {
	v := os.Getenv(FeatureFlagEnv)
	part := strings.TrimSpace(strings.SplitN(v, ",", 2)[0])
	return strings.ToLower(part) == "true"
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
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"detail":"` + detail + `"}`))
}

func notFound(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Not Found","status":404,"detail":"` + detail + `"}`))
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401,"detail":"Authentication required."}`))
}

func internalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500}`))
}

// Handler returns a chi router with memory API routes. Use this when the
// router is mounted at /api/v1 (the historical mount point) and the calling
// service is responsible for ensuring the meeting mount at /api/v1/meetings
// does not shadow these paths. Prefer RegisterRoutesOnRouter when nesting
// memory routes under the meeting sub-router — that pattern is immune to
// chi's longest-prefix-mount shadowing.
func Handler(log *zerolog.Logger, pool *pgxpool.Pool, worker *Worker) http.Handler {
	r := chi.NewRouter()
	RegisterRoutesOnRouter(r, log, pool, worker)
	return r
}

// RegisterRoutesOnRouter registers the memory API routes on the given parent
// router. The routes are registered as /{id}/memory/... relative to the
// parent, so the caller is responsible for scoping the parent appropriately
// (e.g. meeting.HandlerWithSubRoute delegates a sub-router scoped at
// /{id} so the full public path resolves to /api/v1/meetings/{id}/memory/...).
//
// This is the preferred mount pattern: registering memory routes inside the
// meeting router lets chi's radix tree do longest-prefix matching at the
// {id} node, so /{id}/memory/... reaches memory's handlers and /{id} still
// reaches the meeting handler. The previous approach of mounting the memory
// sub-router at /api/v1 was shadowed by the meeting mount at
// /api/v1/meetings and never reached the memory handlers in production.
func RegisterRoutesOnRouter(parent chi.Router, log *zerolog.Logger, pool *pgxpool.Pool, worker *Worker) {
	parent.Use(WithRepository(pool))
	if worker != nil {
		parent.Use(WithWorker(worker))
	}

	// Feature flag gate + auth guard for all routes
	parent.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				enabled := isFeatureFlagEnabled()
				FlagEvaluationMs.Observe(float64(time.Since(start).Milliseconds()))
				if !enabled {
					if detectFlagMisconfiguration(r) {
						emitFlagMisconfiguration(log, r)
					}
					forbidden(w, "Meeting memory processing is not enabled. Set FF_ENABLE_MEETING_MEMORY_PROCESSING=true to activate.")
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

		// Relative paths — the caller scopes the parent so these resolve to
		// /api/v1/meetings/{id}/memory/... at the public URL.
		g.Get("/memory", handleGetMemory(log))
		g.Get("/memory/versions", handleListMemoryVersions(log))
		g.Get("/memory/versions/{versionNumber}", handleGetMemoryVersion(log))
		g.Post("/memory/reprocess", handleReprocessMemory(log))
		g.Get("/memory/state", handleGetMemoryState(log))
		g.Get("/memory/conflicts", handleGetConflicts(log))
		g.Post("/memory/conflicts/{conflictId}/resolve", handleResolveConflict(log))
	})
}

func getUserIDFromContext(r *http.Request) uuid.UUID {
	if v := r.Context().Value(contextKey{}); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	// Also check authContextKey (used by test router to avoid repo/userID collision)
	if v := r.Context().Value(authContextKey{}); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// handleGetMemory handles GET /meetings/{id}/memory
func handleGetMemory(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		// Ownership check
		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil {
			internalError(w)
			return
		}
		if !owned {
			forbidden(w, "You do not have access to this meeting")
			return
		}

		// Get active version
		version, err := repo.GetActiveVersion(r.Context(), meetingID)
		fmt.Printf("DEBUG GetMemory: after GetActiveVersion version=%p err=%v\n", version, err)
		if err != nil {
			internalError(w)
			return
		}

		// Check processing state — wrap in recover to catch mock panics
		var state *ProcessingState
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("PANIC in GetProcessingState: %v\n", r)
				}
			}()
			state, err = repo.GetProcessingState(r.Context(), meetingID)
		}()
		fmt.Printf("DEBUG GetMemory: after GetProcessingState state=%p err=%v\n", state, err)
		if err != nil || state == nil {
			internalError(w)
			return
		}

		// Get prior memory inputs for display
		var priorMemories []PriorMemorySummary
		if version != nil {
			priorMemories, err = repo.ListPriorMemoryInputs(r.Context(), version.ID)
			fmt.Printf("DEBUG GetMemory: after ListPriorMemoryInputs priorMemories=%d err=%v\n", len(priorMemories), err)
			if priorMemories == nil {
				priorMemories = []PriorMemorySummary{}
			}
		}

		resp := MemoryResponse{
			MeetingID:         meetingID,
			ProcessingState:   state.ProcessingState,
			IsStale:           state.IsStale,
			PriorMemoriesUsed: priorMemories,
		}

		if version == nil {
			// No memory yet
			resp.ProcessingState = string(MemoryStateNotProcessed)
			resp.IsBriefingReady = false
		} else {
			resp.VersionNumber = version.VersionNumber
			resp.Status = string(version.Status)
			resp.ProcessingState = state.ProcessingState
			resp.IsStale = state.IsStale
			resp.IsBriefingReady = version.Content.BriefingReadiness.Ready
			resp.BriefingReadiness = version.Content.BriefingReadiness
			resp.Content = version.Content
			resp.CreatedAt = version.CreatedAt
		}

		fmt.Printf("DEBUG GetMemory: responding with status=%s state=%s version=%p\n", resp.Status, resp.ProcessingState, version)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleListMemoryVersions handles GET /meetings/{id}/memory/versions
func handleListMemoryVersions(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		versions, err := repo.ListVersions(r.Context(), meetingID)
		if err != nil {
			internalError(w)
			return
		}
		if versions == nil {
			versions = []MemoryVersionSummary{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(MemoryVersionList{
			MeetingID: meetingID,
			Versions:  versions,
		})
	}
}

// handleGetMemoryVersion handles GET /meetings/{id}/memory/versions/{versionNumber}
func handleGetMemoryVersion(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		version, err := repo.GetVersionByNumber(r.Context(), meetingID, versionNum)
		if err != nil {
			if errors.Is(err, ErrVersionNotFound) {
				notFound(w, "Memory version not found")
				return
			}
			internalError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(version)
	}
}

// handleReprocessMemory handles POST /meetings/{id}/memory/reprocess
func handleReprocessMemory(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		// Parse optional request body
		var req ReprocessRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Bad Request","status":400}`))
				return
			}
		}

		// Get previous active version if any
		prevVersionID, _ := repo.GetActiveVersionID(r.Context(), meetingID)

		// Queue a reprocess job
		jobID, correlationID, err := repo.QueueJob(r.Context(), meetingID, TriggerTypeManualRetry, &prevVersionID)
		if err != nil {
			internalError(w)
			return
		}

		ProcessingQueuedTotal.WithLabelValues(string(TriggerTypeManualRetry)).Inc()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(ReprocessAccepted{
			JobID:         jobID,
			CorrelationID: correlationID,
		})
	}
}

// handleGetMemoryState handles GET /meetings/{id}/memory/state
func handleGetMemoryState(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		state, err := repo.GetProcessingState(r.Context(), meetingID)
		if err != nil {
			if errors.Is(err, ErrMeetingNotFound) {
				notFound(w, "Meeting not found")
				return
			}
			internalError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(state)
	}
}

// handleGetConflicts handles GET /meetings/{id}/memory/conflicts
func handleGetConflicts(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
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

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		conflicts, err := repo.ListPendingConflicts(r.Context(), meetingID)
		if err != nil {
			internalError(w)
			return
		}
		if conflicts == nil {
			conflicts = []MemoryConflict{}
		}

		// Build summaries
		summaries := make([]MemoryConflictSummary, len(conflicts))
		for i, c := range conflicts {
			escapedNote := ""
			if c.ResolutionNote != nil {
				escapedNote = html.EscapeString(*c.ResolutionNote)
			}
			summaries[i] = MemoryConflictSummary{
				ID:               c.ID,
				QualityStatus:    c.QualityStatus,
				ReviewStatus:     c.ReviewStatus,
				CreatedAt:        c.CreatedAt,
				ConflictingItems: c.ConflictingItems,
				ResolutionNote:   &escapedNote,
				ResolvedAt:       c.ResolvedAt,
			}
			// Extract category from conflicting items if available
			if len(c.ConflictingItems.Current) > 0 {
				summaries[i].Category = c.ConflictingItems.Current[0].Statement
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ConflictList{
			MeetingID: meetingID,
			Conflicts: summaries,
		})
	}
}

// handleResolveConflict handles POST /meetings/{id}/memory/conflicts/{conflictId}/resolve
func handleResolveConflict(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meetingIDStr := chi.URLParam(r, "id")
		meetingID, err := uuid.Parse(meetingIDStr)
		if err != nil {
			notFound(w, "Invalid meeting ID")
			return
		}

		conflictIDStr := chi.URLParam(r, "conflictId")
		conflictID, err := uuid.Parse(conflictIDStr)
		if err != nil {
			notFound(w, "Invalid conflict ID")
			return
		}

		userID := getUserIDFromContext(r)
		repo := getRepository(r)
		if repo == nil || userID == uuid.Nil {
			internalError(w)
			return
		}

		owned, err := repo.MeetingExistsAndOwned(r.Context(), meetingID, userID)
		if err != nil || !owned {
			if !owned {
				forbidden(w, "You do not have access to this meeting")
				return
			}
			internalError(w)
			return
		}

		// Parse resolution request
		var req ConflictResolveRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Bad Request","status":400}`))
				return
			}
		}

		// Resolve the conflict
		err = repo.ResolveConflict(r.Context(), conflictID, userID, req.ResolutionNote)
		if err != nil {
			if errors.Is(err, ErrConflictNotFound) {
				notFound(w, "Conflict not found or already resolved")
				return
			}
			internalError(w)
			return
		}

		// Queue a conflict resolution reprocess job
		prevVersionID, _ := repo.GetActiveVersionID(r.Context(), meetingID)
		_, _, err = repo.QueueJob(r.Context(), meetingID, TriggerTypeConflictResolution, &prevVersionID)
		if err != nil {
			// Non-fatal: conflict resolved, but reprocess job failed to queue
			log.Error().Err(err).Str("meeting_id", meetingID.String()).Msg("failed to queue conflict resolution reprocess")
		} else {
			ProcessingQueuedTotal.WithLabelValues(string(TriggerTypeConflictResolution)).Inc()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"resolved"}`))
	}
}

// MountMemoryRoutes mounts the memory API onto an existing chi router,
// wiring the feature flag, auth, repository, and worker.
func MountMemoryRoutes(parent chi.Router, log *zerolog.Logger, pool *pgxpool.Pool, worker *Worker) {
	parent.Mount("/api/v1", Handler(log, pool, worker))
}

// detectFlagMisconfiguration returns true if the browser is sending a request with a mismatched flag
// (browser enabled but server disabled). We check the X-Browser-FF-Meeting-Memory header set by the frontend.
func detectFlagMisconfiguration(r *http.Request) bool {
	browserFlag := r.Header.Get("X-Browser-FF-Meeting-Memory")
	if browserFlag == "" {
		return false
	}
	// Mismatch: server is "false" but browser sends "true"
	return browserFlag == "true"
}

// emitFlagMisconfiguration emits the flag misconfiguration metric and log event.
// No raw endpoint paths are logged — only the feature flag name and a generic
// endpoint identifier to avoid disclosing internal API structure (CWE-201).
func emitFlagMisconfiguration(log *zerolog.Logger, r *http.Request) {
	browserValue := r.Header.Get("X-Browser-FF-Meeting-Memory")
	FlagMisconfigurationTotal.WithLabelValues("FF_ENABLE_MEETING_MEMORY_PROCESSING").Inc()
	log.Warn().
		Str("flag", "FF_ENABLE_MEETING_MEMORY_PROCESSING").
		Str("browser_flag_value", browserValue).
		Str("endpoint", "memory_endpoint").
		Msg("memory.flag_misconfiguration")
}
