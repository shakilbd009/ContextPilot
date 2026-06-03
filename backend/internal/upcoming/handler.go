package upcoming

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/contextpilot/backend/internal/briefing"
)

const featureFlagEnv = "FF_ENABLE_UPCOMING_MEETINGS"

// BrowserFlagHeader is the request header the browser client sends to indicate
// the state of its local upcoming-meetings feature flag. The server uses it to
// detect misconfiguration (browser enabled but server disabled) so operators
// can observe the deployment drift via metric + log.
const BrowserFlagHeader = "X-Browser-FF-Upcoming-Meetings"

type contextKey struct{}
type poolKey     struct{}
type authContextKey struct{}

// WithRepository injects the upcoming repository into the request context.
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

func getRepository(r *http.Request) *Repository {
	if v := r.Context().Value(contextKey{}); v != nil {
		if repo, ok := v.(*Repository); ok {
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

func isFeatureFlagEnabled(ctx context.Context) bool {
	if testFFValue != "" {
		part := strings.SplitN(testFFValue, ",", 2)[0]
		return strings.ToLower(part) == "true"
	}
	v := strings.TrimSpace(os.Getenv(featureFlagEnv))
	part := strings.SplitN(v, ",", 2)[0]
	return strings.ToLower(part) == "true"
}

// testFFValue is a package-level override for testing only.
// It is set by buildTestRouter and read by isFeatureFlagEnabled
// to bypass the deferred os.Setenv restore timing issue.
var testFFValue string

// hashID returns the first 16 hex chars of SHA-256 of the stringified UUID.
// Used for privacy-safe log correlation (FR-20).
func hashID(id uuid.UUID) string {
	h := sha256.Sum256([]byte(id.String()))
	return hex.EncodeToString(h[:8])
}

// hashTitle returns the first 16 hex chars of SHA-256 of the title string.
// Used for privacy-safe log correlation (FR-20). No raw titles in logs.
func hashTitle(title string) string {
	h := sha256.Sum256([]byte(title))
	return hex.EncodeToString(h[:8])
}

// detectFlagMisconfiguration returns true if the browser is sending a request
// claiming the upcoming-meetings feature is enabled while the server flag is
// disabled. Mirrors the pattern used by the memory and briefing handlers so
// the operator signal reflects only real deployment drift, not random hits to
// a disabled-feature path.
func detectFlagMisconfiguration(r *http.Request) bool {
	browserFlag := r.Header.Get(BrowserFlagHeader)
	if browserFlag == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(browserFlag), "true")
}

// Handler returns a chi router with upcoming meeting CRUD routes.
func Handler(log *zerolog.Logger, pool *pgxpool.Pool) http.Handler {
	return HandlerWithSubRoute(log, pool, nil)
}

// HandlerWithSubRoute returns a chi router with upcoming meeting CRUD routes.
// If mountSubRoute is non-nil, it is called with the router after all upcoming
// routes are registered, allowing the caller to mount sub-routes that share the
// /api/v1/upcoming prefix without being shadowed. Specifically, the briefing
// handler (which registers /upcoming/{meetingId}/briefing/...) is mounted here
// so it receives requests before the flag-mismatch guard in upcoming's group
// runs. The pattern solves chi's longest-prefix mount shadowing: if briefing
// were mounted at /api/v1 (outside and below upcoming's mount), every request
// to /api/v1/upcoming/{meetingId}/briefing/... would be captured by upcoming's
// mount and return 404 with no chance for briefing's handlers to run.
func HandlerWithSubRoute(log *zerolog.Logger, pool *pgxpool.Pool, mountSubRoute func(chi.Router)) http.Handler {
	r := chi.NewRouter()

	// Inject repository into context so every route can call getRepository(r).
	// Without this middleware, getRepository(r) returns nil and routes that
	// touch the DB return 500 with log "no repository in request context".
	// Mirrors the pattern in internal/meeting/handler.go:84 and
	// internal/briefing/handler.go:112. Fix for task t_42c0144b (ethical-hacker
	// finding F-A: /api/v1/upcoming returns 500).
	r.Use(WithRepository(pool))

	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !isFeatureFlagEnabled(r.Context()) {
					// Only emit the flag-mismatch signal when the browser
					// actually sent the feature header. Otherwise the metric
					// would fire for any random GET/POST to a disabled
					// feature path and pollute the operator signal.
					if detectFlagMisconfiguration(r) {
						FlagMismatchTotal.WithLabelValues("upcoming_meetings", "frontend-disabled", "feature_disabled").Inc()

						userID, _ := getUserID(r)
						log.Info().
							Str("request_id", getCorrelationID(r)).
							Str("user_id_hash", hashID(userID)).
							Str("flag_name", "upcoming_meetings").
							Str("direction", "frontend-disabled").
							Msg("upcoming_meeting.flag_mismatch")
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"code":"feature_disabled","message":"Upcoming meetings are not enabled. Set FF_ENABLE_UPCOMING_MEETINGS=true to activate."}`))
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		g.Post("/", handleCreateUpcomingMeeting(log))
		g.Get("/", handleListUpcomingMeetings(log))
		g.Get("/{id}", handleGetUpcomingMeeting(log))
		g.Patch("/{id}", handleUpdateUpcomingMeeting(log))
		g.Post("/{id}/cancel", handleCancelUpcomingMeeting(log))
	})

	if mountSubRoute != nil {
		mountSubRoute(r)
	}

	return r
}

func handleCreateUpcomingMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := getUserID(r)
		if !ok {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		var in CreateUpcomingMeetingInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		source := "direct"
		switch {
		case strings.Contains(r.Referer(), "dashboard"):
			source = "dashboard"
		case strings.Contains(r.Referer(), "calendar"):
			source = "calendar"
		case strings.Contains(r.Referer(), "list"):
			source = "list"
		}
		hasParticipants := len(in.Participants) > 0
		CreateRequestedTotal.WithLabelValues(source, strconv.FormatBool(hasParticipants)).Inc()

		validation := ValidateCreate(in, time.Now())
		if validation.HasErrors() {
			for _, fe := range validation.Errors {
				ValidationFailedTotal.WithLabelValues(fe.Code).Inc()
			}
			CreateFailedTotal.WithLabelValues("validation").Inc()

			// LOG-3: upcoming_meeting.create_failed — validation error
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("failure_class", "validation").
				Str("validation_class", validation.Errors[0].Code).
				Bool("retryable", false).
				Msg("upcoming_meeting.create_failed")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{Errors: validation.Errors, Values: validation.Values})
			return
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			CreateFailedTotal.WithLabelValues("internal_error").Inc()
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		scheduledStart, err := time.Parse(time.RFC3339, strings.TrimSpace(in.ScheduledStart))
		if err != nil {
			validation.Errors = append(validation.Errors, FieldError{
				Field:   "scheduledStart",
				Message: "Scheduled start must be a valid ISO 8601 timestamp.",
				Code:    "invalid_scheduled_start",
			})
			CreateFailedTotal.WithLabelValues("validation").Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{Errors: validation.Errors, Values: validation.Values})
			return
		}

		title := strings.TrimSpace(in.Title)

		// LOG-1: upcoming_meeting.create_started
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("title", hashTitle(title)).
			Msg("upcoming_meeting.create_started")

		participants := make([]CreateParticipantInput, len(in.Participants))
		for i, p := range in.Participants {
			dn := strings.TrimSpace(nullString(p.DisplayName))
			em := strings.TrimSpace(nullString(p.Email))
			var dnp, emp *string
			if dn != "" {
				dnp = &dn
			}
			if em != "" {
				emp = &em
			}
			participants[i] = CreateParticipantInput{DisplayName: dnp, Email: emp, Organization: p.Organization}
		}

		meetingID, err := repo.CreateMeeting(r.Context(), CreateMeetingInput{
			Title:                title,
			ScheduledStart:       scheduledStart,
			Description:         in.Description,
			ClientOrOrganization: in.ClientOrOrganization,
			CreatedBy:           userID,
			Participants:        participants,
		})
if err != nil {
			log.Error().Err(err).Str("correlationId", getCorrelationID(r)).Msg("failed to create upcoming meeting")
			CreateFailedTotal.WithLabelValues("storage_error").Inc()

			// LOG-3: upcoming_meeting.create_failed — storage error
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("failure_class", "storage_error").
				Str("validation_class", "").
				Bool("retryable", true).
				Msg("upcoming_meeting.create_failed")

			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		elapsed := time.Since(start).Milliseconds()
		ActionDurationMs.WithLabelValues("create", "success").Observe(float64(elapsed))

		hasClientOrOrg := in.ClientOrOrganization != nil && strings.TrimSpace(*in.ClientOrOrganization) != ""
		CreateSucceededTotal.WithLabelValues(ParticipantCountBucket(len(participants)), strconv.FormatBool(hasClientOrOrg)).Inc()

		// LOG-2: upcoming_meeting.create_completed
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(meetingID)).
			Int("participant_count", len(participants)).
			Msg("upcoming_meeting.create_completed")

		triggerBriefing(log, r, meetingID, "create")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateUpcomingMeetingOutput{
			ID:       meetingID.String(),
			Redirect: "/api/v1/upcoming/" + meetingID.String(),
		})
	}
}

func handleListUpcomingMeetings(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := getUserID(r)
		if !ok {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
			return
		}

		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if status == "" {
			status = "scheduled"
		}

		view := "list"
		if strings.Contains(r.URL.Path, "calendar") {
			view = "calendar"
		} else if strings.Contains(r.Referer(), "dashboard") {
			view = "dashboard"
		}

		repo := getRepository(r)
		if repo == nil {
			log.Error().Msg("no repository in request context")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		meetings, err := repo.ListMeetings(r.Context(), userID, status)
		if err != nil {
			log.Error().Err(err).Msg("failed to list upcoming meetings")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		if meetings == nil {
			meetings = []UpcomingMeeting{}
		}

		now := time.Now()
		for i := range meetings {
			meetings[i].Editable = !EditWindowExpired(meetings[i].ScheduledStart, now)
			meetings[i].Cancellable = !CancelWindowExpired(meetings[i].ScheduledStart, now) && meetings[i].Status == StatusScheduled
		}

		elapsed := time.Since(start).Milliseconds()
		ActionDurationMs.WithLabelValues("list", "success").Observe(float64(elapsed))
		ListLoadedTotal.WithLabelValues(view, ResultCountBucket(len(meetings))).Inc()

		// LOG-9: upcoming_meeting.list_loaded
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Int("result_count", len(meetings)).
			Str("view", view).
			Msg("upcoming_meeting.list_loaded")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(meetings)
	}
}

func handleGetUpcomingMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := getUserID(r)
		if !ok {
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

		meeting, err := repo.GetMeetingOwned(r.Context(), id, userID)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
				return
			}
			log.Error().Err(err).Msg("failed to get upcoming meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		now := time.Now()
		meeting.Editable = !EditWindowExpired(meeting.ScheduledStart, now)
		meeting.Cancellable = !CancelWindowExpired(meeting.ScheduledStart, now) && meeting.Status == StatusScheduled

		briefingStatus := determineBriefingStatus(r, meeting.ID, log)
		meeting.BriefingStatus = &briefingStatus
		if briefingStatus == "stale" {
			meeting.IsBriefingStale = true
		}

		elapsed := time.Since(start).Milliseconds()
		ActionDurationMs.WithLabelValues("detail", "success").Observe(float64(elapsed))
		ViewedTotal.WithLabelValues(string(meeting.Status), briefingStatus).Inc()

		// LOG-10: upcoming_meeting.viewed
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(meeting.ID)).
			Str("status", string(meeting.Status)).
			Str("briefing_status", briefingStatus).
			Msg("upcoming_meeting.viewed")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(meeting)
	}
}

func handleUpdateUpcomingMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := getUserID(r)
		if !ok {
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

		meeting, err := repo.GetMeetingOwned(r.Context(), id, userID)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
				return
			}
			log.Error().Err(err).Msg("failed to get upcoming meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		now := time.Now()
		if EditWindowExpired(meeting.ScheduledStart, now) {
			UpdateFailedTotal.WithLabelValues("not_editable").Inc()

			// LOG-6: upcoming_meeting.update_failed — not editable
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("upcoming_meeting_id_hash", hashID(id)).
				Str("failure_class", "not_editable").
				Str("reason", "").
				Msg("upcoming_meeting.update_failed")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{
					"field":   "_",
					"message": "Edit window has expired. Meetings can only be edited within 15 minutes of their scheduled start time.",
					"code":    "not_editable",
				}},
			})
			return
		}

		var in UpdateUpcomingMeetingInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		validation := ValidateUpdate(in, meeting, now)
		if validation.HasErrors() {
			for _, fe := range validation.Errors {
				ValidationFailedTotal.WithLabelValues(fe.Code).Inc()
			}
			UpdateFailedTotal.WithLabelValues("validation").Inc()

			// LOG-6: upcoming_meeting.update_failed — validation error
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("upcoming_meeting_id_hash", hashID(id)).
				Str("failure_class", "validation").
				Str("reason", "").
				Msg("upcoming_meeting.update_failed")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ValidationErrorBody{Errors: validation.Errors, Values: validation.Values})
			return
		}

		updateIn := UpdateMeetingInput{}
		if in.Title != nil {
			t := strings.TrimSpace(*in.Title)
			updateIn.Title = &t
		}
		if in.ScheduledStart != nil {
			ss, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.ScheduledStart))
			if err == nil {
				updateIn.ScheduledStart = &ss
			}
		}
		if in.Description != nil {
			updateIn.Description = in.Description
		}
		if in.ClientOrOrganization != nil {
			updateIn.ClientOrOrganization = in.ClientOrOrganization
		}
		if in.Participants != nil {
			participants := make([]CreateParticipantInput, len(in.Participants))
			for i, p := range in.Participants {
				dn := strings.TrimSpace(nullString(p.DisplayName))
				em := strings.TrimSpace(nullString(p.Email))
				var dnp, emp *string
				if dn != "" {
					dnp = &dn
				}
				if em != "" {
					emp = &em
				}
				participants[i] = CreateParticipantInput{DisplayName: dnp, Email: emp, Organization: p.Organization}
			}
			updateIn.Participants = participants
		}

		// LOG-4: upcoming_meeting.update_started
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(id)).
			Msg("upcoming_meeting.update_started")

		err = repo.UpdateMeeting(r.Context(), id, updateIn)
		if err != nil {
			log.Error().Err(err).Msg("failed to update upcoming meeting")
			UpdateFailedTotal.WithLabelValues("storage_error").Inc()

			// LOG-6: upcoming_meeting.update_failed — storage error
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("upcoming_meeting_id_hash", hashID(id)).
				Str("failure_class", "storage_error").
				Str("reason", "").
				Msg("upcoming_meeting.update_failed")

			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		meaningfulEdit := IsMeaningfulEdit(meeting, in)
		if meaningfulEdit {
			UpdatedTotal.WithLabelValues(classifyChange(in)).Inc()
		}

		// LOG-5: upcoming_meeting.update_completed
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(id)).
			Str("changed_field_class", classifyChange(in)).
			Bool("meaningful_edit", meaningfulEdit).
			Msg("upcoming_meeting.update_completed")

		updated, err := repo.GetMeetingOwned(r.Context(), id, userID)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch updated meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		updated.Editable = !EditWindowExpired(updated.ScheduledStart, now)
		updated.Cancellable = !CancelWindowExpired(updated.ScheduledStart, now) && updated.Status == StatusScheduled

		elapsed := time.Since(start).Milliseconds()
		ActionDurationMs.WithLabelValues("update", "success").Observe(float64(elapsed))

		if meaningfulEdit {
			triggerBriefing(log, r, id, "meaningful_edit")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(UpdateSuccessBody{UpcomingMeeting: updated, MeaningfulEdit: meaningfulEdit})
	}
}

func handleCancelUpcomingMeeting(log *zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := getUserID(r)
		if !ok {
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

		meeting, err := repo.GetMeetingOwned(r.Context(), id, userID)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Meeting not found"}`, http.StatusNotFound)
				return
			}
			log.Error().Err(err).Msg("failed to get upcoming meeting")
			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		now := time.Now()
		if CancelWindowExpired(meeting.ScheduledStart, now) || meeting.Status == StatusCancelled {
			CancelFailedTotal.WithLabelValues("not_cancellable").Inc()

			// LOG-8: upcoming_meeting.cancel_failed — not cancellable
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("upcoming_meeting_id_hash", hashID(id)).
				Str("failure_class", "not_cancellable").
				Str("reason", "").
				Msg("upcoming_meeting.cancel_failed")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "not_cancellable",
				"message": "Cancellation is no longer available after the meeting window.",
			})
			return
		}

		err = repo.CancelMeeting(r.Context(), id)
		if err != nil {
			log.Error().Err(err).Msg("failed to cancel upcoming meeting")
			CancelFailedTotal.WithLabelValues("storage_error").Inc()

			// LOG-8: upcoming_meeting.cancel_failed — storage error
			log.Info().
				Str("request_id", getCorrelationID(r)).
				Str("user_id_hash", hashID(userID)).
				Str("upcoming_meeting_id_hash", hashID(id)).
				Str("failure_class", "storage_error").
				Str("reason", "").
				Msg("upcoming_meeting.cancel_failed")

			http.Error(w, `{"type":"about:blank","title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			return
		}

		hadBriefingStatus := determineBriefingStatus(r, id, log)

		elapsed := time.Since(start).Milliseconds()
		ActionDurationMs.WithLabelValues("cancel", "success").Observe(float64(elapsed))
		CancelledTotal.WithLabelValues(hadBriefingStatus).Inc()

		// LOG-7: upcoming_meeting.cancel_completed
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(id)).
			Str("had_briefing_status", "none").
			Msg("upcoming_meeting.cancel_completed")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
	}
}

// ─── BRD-04 Briefing Trigger ───────────────────────────────────────────────

func triggerBriefing(log *zerolog.Logger, r *http.Request, meetingID uuid.UUID, trigger string) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start).Milliseconds()
		BriefingTriggerEnqueueDurationMs.WithLabelValues(trigger, "queued").Observe(float64(elapsed))
	}()

	if !IsFeatureFlagEnabled() || !briefing.IsFeatureFlagEnabled() {
		BriefingTriggerSkippedTotal.WithLabelValues("pre_call_briefing_disabled").Inc()
		BriefingTriggerEnqueueDurationMs.WithLabelValues(trigger, "skipped").Observe(float64(time.Since(start).Milliseconds()))

		// LOG-12: upcoming_meeting.briefing_trigger_skipped — pre_call_briefing disabled
		userID, _ := getUserID(r)
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(meetingID)).
			Str("reason", "pre_call_briefing_disabled").
			Msg("upcoming_meeting.briefing_trigger_skipped")

		return
	}

	pool := getPool(r)
	if pool == nil {
		BriefingTriggerFailedTotal.WithLabelValues("queue_unavailable").Inc()

		// LOG-13: upcoming_meeting.briefing_trigger_failed — queue unavailable
		userID, _ := getUserID(r)
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(meetingID)).
			Str("failure_class", "queue_unavailable").
			Bool("retryable", true).
			Msg("upcoming_meeting.briefing_trigger_failed")

		return
	}

	briefingRepo := briefing.NewRepository(pool)
	_, _, err := briefingRepo.QueueBriefingJob(r.Context(), meetingID, briefing.BriefingTriggerAuto, nil)
	if err != nil {
		BriefingTriggerFailedTotal.WithLabelValues("unknown").Inc()

		// LOG-13: upcoming_meeting.briefing_trigger_failed — unknown error
		userID, _ := getUserID(r)
		log.Info().
			Str("request_id", getCorrelationID(r)).
			Str("user_id_hash", hashID(userID)).
			Str("upcoming_meeting_id_hash", hashID(meetingID)).
			Str("failure_class", "unknown").
			Bool("retryable", true).
			Msg("upcoming_meeting.briefing_trigger_failed")

		return
	}

	BriefingTriggerQueuedTotal.WithLabelValues(trigger).Inc()

	// LOG-11: upcoming_meeting.briefing_trigger_queued
	userID, _ := getUserID(r)
	log.Info().
		Str("request_id", getCorrelationID(r)).
		Str("user_id_hash", hashID(userID)).
		Str("upcoming_meeting_id_hash", hashID(meetingID)).
		Str("trigger", trigger).
		Msg("upcoming_meeting.briefing_trigger_queued")
}

func determineBriefingStatus(r *http.Request, meetingID uuid.UUID, log *zerolog.Logger) string {
	if !briefing.IsFeatureFlagEnabled() {
		return "disabled"
	}

	pool := getPool(r)
	if pool == nil {
		return "unavailable"
	}

	briefingRepo := briefing.NewRepository(pool)
	active, err := briefingRepo.GetActiveBriefingVersion(r.Context(), meetingID)
	if err != nil || active == nil {
		return "unavailable"
	}

	switch active.PreparationStatus {
	case briefing.PreparationStatusGenerating:
		return "generating"
	case briefing.PreparationStatusReady, briefing.PreparationStatusReadyCaveats:
		return "ready"
	case briefing.PreparationStatusNoPriorMemory:
		return "unavailable"
	case briefing.PreparationStatusStale, briefing.PreparationStatusRegenerating:
		return "stale"
	case briefing.PreparationStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func classifyChange(in UpdateUpcomingMeetingInput) string {
	fields := 0
	if in.Title != nil {
		fields++
	}
	if in.ScheduledStart != nil {
		fields++
	}
	if in.Description != nil {
		fields++
	}
	if in.ClientOrOrganization != nil {
		fields++
	}
	if in.Participants != nil {
		fields++
	}

	if fields == 0 || fields > 1 {
		return "multiple"
	}
	if in.ScheduledStart != nil {
		return "schedule"
	}
	if in.Participants != nil {
		return "participants"
	}
	return "metadata"
}

func nullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}