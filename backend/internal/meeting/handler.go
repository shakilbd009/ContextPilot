package meeting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/contextpilot/backend/internal/memory"
)

// FeatureFlagEnv is the server-side env var for the manual meeting import flag.
const FeatureFlagEnv = "FF_ENABLE_MANUAL_MEETING_IMPORT"

// ContextKey is an exported type to avoid context key collisions.
// Exported for use in test files that inject mock repositories.
type ContextKey struct{}

// WithRepository injects the meeting repository into the request context.
func WithRepository(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	repo := NewRepository(pool)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ContextKey{}, repo)
			ctx = context.WithValue(ctx, poolKey{}, pool)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type poolKey struct{}

// MeetingRepository is the interface for meeting persistence operations.
// Exported for testability — both *Repository and test mocks implement it.
type MeetingRepository interface {
	CheckIdempotencyToken(ctx context.Context, userID, token uuid.UUID) (uuid.UUID, string, error)
	CreateMeeting(ctx context.Context, title string, completedAt time.Time, transcript, notes *string, contentSource string, createdBy uuid.UUID, participants []ParticipantInput, idempotencyToken uuid.UUID) (uuid.UUID, error)
	ListMeetings(ctx context.Context, createdBy uuid.UUID, limit, offset int) ([]Meeting, error)
	GetMeeting(ctx context.Context, id uuid.UUID) (*Meeting, error)
	UpdateMeeting(ctx context.Context, meetingID uuid.UUID, title *string, completedAt *time.Time, transcript, notes *string, participants []ParticipantInput) (bool, error)
}

func getRepository(r *http.Request) MeetingRepository {
	if v := r.Context().Value(ContextKey{}); v != nil {
		if repo, ok := v.(MeetingRepository); ok {
			return repo
		}
	}
	return nil
}

func getPool(r *http.Request) *pgxpool.Pool {
	if v := r.Context().Value(poolKey{}); v != nil {
		if p, ok := v.(*pgxpool.Pool); ok {
			return p
		}
	}
	return nil
}

func getCorrelationID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return uuid.New().String()
}

// Handler returns a chi router with meeting CRUD routes.
// The rateLimitMiddleware is applied to POST /meetings (import) to prevent abuse.
// Pass nil to disable rate limiting on this handler.
func Handler(log *zerolog.Logger, pool *pgxpool.Pool, rateLimitMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()

	// Inject repository into context
	r.Use(WithRepository(pool))

	// Feature flag gate — all routes behind FF_ENABLE_MANUAL_MEETING_IMPORT
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				serverEnv := os.Getenv(FeatureFlagEnv)
				enabled := isFeatureFlagEnabled(serverEnv)
				elapsed := time.Since(start).Microseconds()
				FlagEvalDurationMs.Observe(float64(elapsed) / 1000.0)
				if !enabled {
					// Flag misconfiguration: server disabled, browser claims enabled
					if DetectFlagMisconfiguration(r, serverEnv) {
						EmitFlagMisconfiguration(log, r, FeatureFlagEnv, serverEnv)
					}
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"detail":"Manual meeting import is not enabled. Set FF_ENABLE_MANUAL_MEETING_IMPORT=true to activate."}`))
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		// POST /meetings — create meeting (rate limited)
		if rateLimitMiddleware != nil {
			r.With(rateLimitMiddleware).Post("/", handleCreateMeeting(log))
		} else {
			r.Post("/", handleCreateMeeting(log))
		}

		// GET /meetings — list meetings
		r.Get("/", handleListMeetings(log))

		// GET /meetings/{id} — get meeting detail
		r.Get("/{id}", handleGetMeeting(log))

		// PATCH /meetings/{id} — update meeting (FR-10 stale detection trigger)
		r.Patch("/{id}", handleUpdateMeeting(log))
	})

	return r
}

func isFeatureFlagEnabled(envVar string) bool {
	v := os.Getenv(envVar)
	part := strings.TrimSpace(strings.SplitN(v, ",", 2)[0])
	return strings.ToLower(part) == "true"
}

// handleCreateMeeting handles POST /meetings
func handleCreateMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		contentSource := "unknown"

		// Auth guard — require X-User-ID header
		userIDStr := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if userIDStr == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		// Parse request body
		var in CreateMeetingInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		// Derive content source early for metrics
		contentSource = ContentSource(in.Transcript, in.Notes)

		// Emit started metric
		StartedTotal.WithLabelValues("started", contentSource, r.URL.Path).Inc()

		// Validate
		validation := Validate(in)
		if validation.HasErrors() {
			for _, fe := range validation.Errors {
				ValidationFailedTotal.WithLabelValues("validation_failed", fe.Field, contentSource).Inc()
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{
				Errors: validation.Errors,
				Values: validation.Values,
			})
			return
		}

		// Idempotency check
		token, err := uuid.Parse(strings.TrimSpace(in.IdempotencyToken))
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid idempotency token"}`, http.StatusBadRequest)
			return
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			FailedTotal.WithLabelValues("internal_error", "false").Inc()
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// Check idempotency token
		existingID, originalOutcome, err := repo.CheckIdempotencyToken(r.Context(), userID, token)
		if err != nil && err != ErrTokenNotFound {
			log.Error().Err(err).Str("correlationId", getCorrelationID(r)).Msg("idempotency check failed")
			FailedTotal.WithLabelValues("internal_error", "false").Inc()
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}
		if err == nil {
			// Token already used — return 409
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(DuplicateErrorBody{
				Error:          "duplicate",
				MeetingID:      existingID.String(),
				OriginalOutcome: originalOutcome,
			})
			return
		}

		// Parse completedAt
		completedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(in.CompletedAt))
		if err != nil {
			validation.Errors = append(validation.Errors, FieldError{
				Field:   "completedAt",
				Message: "Completed date/time must be a valid ISO 8601 timestamp.",
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{
				Errors: validation.Errors,
				Values: validation.Values,
			})
			return
		}

		// Trim title and participant display names
		title := strings.TrimSpace(in.Title)
		participants := make([]ParticipantInput, len(in.Participants))
		for i, p := range in.Participants {
			participants[i] = ParticipantInput{
				DisplayName:  strings.TrimSpace(p.DisplayName),
				Email:        p.Email,
				Organization: p.Organization,
				Role:         p.Role,
			}
		}

		// Create meeting
		meetingID, err := repo.CreateMeeting(
			r.Context(),
			title,
			completedAt,
			in.Transcript,
			in.Notes,
			contentSource,
			userID,
			participants,
			token,
		)
		if err != nil {
			log.Error().Err(err).Str("correlationId", getCorrelationID(r)).Msg("failed to create meeting")
			FailedTotal.WithLabelValues("internal_error", "false").Inc()
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// Record metrics
		elapsed := time.Since(start).Milliseconds()
		SaveDurationMs.Observe(float64(elapsed))
		ContentSizeChars.Observe(float64(CombinedContentLen(in.Transcript, in.Notes)))
		CompletedTotal.WithLabelValues("completed", contentSource).Inc()

		// Queue memory processing job after meeting transaction commits (FR-1, ADR-0005)
		// This is fire-and-forget from the HTTP response path; we don't await processing.
		if memory.IsFeatureFlagEnabled() {
			memRepo := memory.NewRepository(getPool(r))
			_, _, _ = memRepo.QueueJob(r.Context(), meetingID, memory.TriggerTypeImport, nil)
			memory.ProcessingQueuedTotal.WithLabelValues(string(memory.TriggerTypeImport)).Inc()
		}

		// Return 201 with redirect
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateMeetingOutput{
			ID:       meetingID.String(),
			Redirect: fmt.Sprintf("/meetings/%s", meetingID.String()),
		})
	}
}

// handleListMeetings handles GET /meetings
func handleListMeetings(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if userIDStr == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// Parse page query param (1-indexed), default to page 1
		page := 1
		if pageStr := strings.TrimSpace(r.URL.Query().Get("page")); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		limit := 20
		offset := (page - 1) * limit

		meetings, err := repo.ListMeetings(r.Context(), userID, limit, offset)
		if err != nil {
			log.Error().Err(err).Msg("failed to list meetings")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		if meetings == nil {
			meetings = []Meeting{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"meetings": meetings})
	}
}

// handleGetMeeting handles GET /meetings/{id}
func handleGetMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if userIDStr == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Invalid meeting ID"}`, http.StatusNotFound)
			return
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		meeting, err := repo.GetMeeting(r.Context(), id)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
				return
			}
			log.Error().Err(err).Msg("failed to get meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(meeting)
	}
}

// handleUpdateMeeting handles PATCH /meetings/{id} — FR-10 stale detection trigger.
// When source fields (transcript, notes, participants, title, completed_at) change,
// checks if active memory is stale and queues a stale_reprocess job if so.
func handleUpdateMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if userIDStr == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Invalid meeting ID"}`, http.StatusNotFound)
			return
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// Check the meeting exists and belongs to this user
		existing, err := repo.GetMeeting(r.Context(), id)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
				return
			}
			log.Error().Err(err).Msg("failed to get meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}
		if existing.CreatedBy != userID {
			http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
			return
		}

		// Parse update body
		var in UpdateMeetingInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		// Validate update input
		validationErrors := ValidateUpdate(in)
		if len(validationErrors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{
				Errors: validationErrors,
				Values: map[string]any{
					"title":        in.Title,
					"completedAt":  in.CompletedAt,
					"transcript":   in.Transcript,
					"notes":        in.Notes,
					"participants": in.Participants,
				},
			})
			return
		}

		// Parse completedAt if provided
		var completedAt *time.Time
		if in.CompletedAt != nil {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.CompletedAt))
			if err != nil {
				http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid completedAt format"}`, http.StatusBadRequest)
				return
			}
			completedAt = &parsed
		}

		// Perform update; hasChanges is true if watched source fields changed
		hasChanges, err := repo.UpdateMeeting(r.Context(), id, in.Title, completedAt, in.Transcript, in.Notes, in.Participants)
		if err != nil {
			log.Error().Err(err).Msg("failed to update meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// Return 200 with updated meeting; don't wait for memory processing
		updated, err := repo.GetMeeting(r.Context(), id)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch updated meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		// FR-10: stale detection — after source mutation, check memory staleness
		// Only relevant when the memory processing flag is enabled
		if hasChanges && memory.IsFeatureFlagEnabled() {
			memRepo := memory.NewRepository(getPool(r))
			// IsMemoryStale checks if meeting.updated_at > active_version.created_at
			isStale, err := memRepo.IsMemoryStale(r.Context(), id)
			if err == nil && isStale {
				// Queue stale_reprocess job; fire-and-forget from response path
				_, _, _ = memRepo.QueueJob(r.Context(), id, memory.TriggerTypeStaleReprocess, nil)
				memory.ProcessingQueuedTotal.WithLabelValues(string(memory.TriggerTypeStaleReprocess)).Inc()
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(updated)
	}
}