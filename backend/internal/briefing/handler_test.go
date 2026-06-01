package briefing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"
)

// Tests for HTTP response helpers
func TestForbidden(t *testing.T) {
	rr := httptest.NewRecorder()
	forbidden(rr, "test detail")
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["detail"] != "test detail" {
		t.Errorf("detail = %v, want %q", resp["detail"], "test detail")
	}
}

func TestNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	notFound(rr, "not found")
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestUnauthorized(t *testing.T) {
	rr := httptest.NewRecorder()
	unauthorized(rr)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestInternalError(t *testing.T) {
	rr := httptest.NewRecorder()
	internalError(rr)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestConflict(t *testing.T) {
	rr := httptest.NewRecorder()
	conflict(rr, "already running")
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["detail"] != "already running" {
		t.Errorf("detail = %v, want %q", resp["detail"], "already running")
	}
}

func TestConflict_EscapesHTML(t *testing.T) {
	rr := httptest.NewRecorder()
	conflict(rr, "<script>alert('xss')</script>")
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	// detail must be escaped to prevent XSS in UI rendering
	if resp["detail"] != "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" {
		t.Errorf("detail = %v, want escaped HTML", resp["detail"])
	}
}

// Tests for getUserID header parsing
func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		header string
		wantID uuid.UUID
		wantOk bool
	}{
		{
			name:   "valid uuid",
			header: "550e8400-e29b-41d4-a716-446655440000",
			wantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantOk: true,
		},
		{
			name:   "valid with spaces",
			header: "  550e8400-e29b-41d4-a716-446655440000  ",
			wantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantOk: true,
		},
		{
			name:   "empty",
			header: "",
			wantID: uuid.Nil,
			wantOk: false,
		},
		{
			name:   "invalid uuid",
			header: "not-a-uuid",
			wantID: uuid.Nil,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-User-ID", tt.header)
			gotID, gotOk := getUserID(req)
			if gotID != tt.wantID || gotOk != tt.wantOk {
				t.Errorf("getUserID() = (%v, %v), want (%v, %v)", gotID, gotOk, tt.wantID, tt.wantOk)
			}
		})
	}
}

// Tests for ExclusionAccepted JSON round-trip
func TestExclusionAccepted_JSON(t *testing.T) {
	now := time.Now()
	ea := ExclusionAccepted{
		SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		RestoredAt:      &now,
	}

	encoded, err := json.Marshal(ea)
	if err != nil {
		t.Fatalf("json.Marshal(ea) = %v, want nil", err)
	}

	var decoded ExclusionAccepted
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.SourceMeetingID != ea.SourceMeetingID {
		t.Errorf("SourceMeetingID = %v, want %v", decoded.SourceMeetingID, ea.SourceMeetingID)
	}
	if decoded.RestoredAt == nil {
		t.Error("RestoredAt = nil, want non-nil")
	}
}

func TestExclusionAccepted_NilRestoredAt(t *testing.T) {
	ea := ExclusionAccepted{
		SourceMeetingID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		RestoredAt:      nil,
	}

	encoded, err := json.Marshal(ea)
	if err != nil {
		t.Fatalf("json.Marshal(ea) = %v, want nil", err)
	}

	var decoded ExclusionAccepted
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.RestoredAt != nil {
		t.Errorf("RestoredAt = %v, want nil", decoded.RestoredAt)
	}
}

// Tests for BriefingProcessingJob fields
func TestBriefingProcessingJob_Fields(t *testing.T) {
	now := time.Now()
	job := BriefingProcessingJob{
		ID:                uuid.New(),
		UpcomingMeetingID: uuid.New(),
		TriggerType:      BriefingTriggerAuto,
		Status:           JobStatusQueued,
		RetryCount:       0,
		MaxRetries:       3,
		CorrelationID:    uuid.New(),
		QueuedAt:         now,
	}

	if job.TriggerType != BriefingTriggerAuto {
		t.Errorf("TriggerType = %v, want %v", job.TriggerType, BriefingTriggerAuto)
	}
	if job.Status != JobStatusQueued {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusQueued)
	}
	if job.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", job.RetryCount)
	}
}

// Tests for BriefingResponse JSON round-trip
func TestBriefingResponse_JSON(t *testing.T) {
	now := time.Now()
	content := BriefingContent{
		ConciseSummary: ConciseSummary{
			Objective:          "Test objective",
			PreparationStatus: "ready",
			RecommendedFocus:  "Focus area",
			TopPriorContext:    "Prior context",
			OpenActions:       "Action items",
			RisksQuestions:   "Risks",
		},
	}

	resp := BriefingResponse{
		MeetingID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		VersionNumber:      1,
		Status:             "active",
		PreparationStatus:  PreparationStatusReady,
		IsStale:             false,
		Content:            &content,
		SourceCount:        2,
		CreatedAt:          now,
		TriggerType:        BriefingTriggerAuto,
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(resp) = %v, want nil", err)
	}

	var decoded BriefingResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(encoded, &decoded) = %v, want nil", err)
	}

	if decoded.MeetingID != resp.MeetingID {
		t.Errorf("MeetingID = %v, want %v", decoded.MeetingID, resp.MeetingID)
	}
	if decoded.VersionNumber != resp.VersionNumber {
		t.Errorf("VersionNumber = %d, want %d", decoded.VersionNumber, resp.VersionNumber)
	}
	if decoded.IsStale {
		t.Error("IsStale = true, want false")
	}
	if decoded.Content == nil {
		t.Error("Content = nil, want non-nil")
	} else if decoded.Content.ConciseSummary.Objective != "Test objective" {
		t.Errorf("Content.ConciseSummary.Objective = %q, want %q", decoded.Content.ConciseSummary.Objective, "Test objective")
	}
}

// ─── FF-guard regression tests ────────────────────────────────────────────────
// These tests prove the briefing feature-flag guard middleware actually fires
// for briefing routes. Prior to this fix the routes were registered on the
// outer chi router (r.Get/Post) instead of the FF-guarded group (g.Get/Post),
// so the guard never executed and the production contract — "when
// FF_ENABLE_PRE_CALL_BRIEFING=false the server returns 403" — was silently
// violated. The tests below would have failed with the bug in place: the
// disabled-flag request would have reached the underlying handler (which then
// either 200s, 401s on missing X-User-ID, or 500s on nil pool) instead of 403.

// briefingFFDisabledRoutes enumerates every route the FF guard must cover.
// Each entry is a (method, path) pair. Path templating uses chi-style {param}
// — we substitute valid UUIDs before serving the request.
var briefingFFDisabledRoutes = []struct {
	name   string
	method string
	path   string
}{
	{"GetBriefing", http.MethodGet, "/upcoming/{mid}/briefing"},
	{"ListVersions", http.MethodGet, "/upcoming/{mid}/briefing/versions"},
	{"GetVersion", http.MethodGet, "/upcoming/{mid}/briefing/versions/{n}"},
	{"Regenerate", http.MethodPost, "/upcoming/{mid}/briefing/regenerate"},
	{"GetSources", http.MethodGet, "/upcoming/{mid}/briefing/sources"},
	{"ExcludeSource", http.MethodPost, "/upcoming/{mid}/briefing/sources/{sid}/exclude"},
	{"RestoreSource", http.MethodPost, "/upcoming/{mid}/briefing/sources/{sid}/restore"},
}

func substituteBriefingPathParams(t *testing.T, tmpl string) string {
	t.Helper()
	mid := uuid.New().String()
	sid := uuid.New().String()
	out := strings.ReplaceAll(tmpl, "{mid}", mid)
	out = strings.ReplaceAll(out, "{sid}", sid)
	out = strings.ReplaceAll(out, "{n}", "1")
	return out
}

// buildBriefingRouter builds the production handler with a nil pool/worker.
// The FF guard must short-circuit before any pool/worker call, so passing
// nil is safe for the disabled-flag tests.
func buildBriefingRouter(t *testing.T) http.Handler {
	t.Helper()
	// Silence logger output during tests.
	logger := zerolog.New(os.Stdout).Level(zerolog.Disabled)
	return Handler(&logger, nil, nil)
}

func TestHandler_FeatureFlagDisabled_AllRoutes(t *testing.T) {
	// Production default: feature flag unset (or explicitly false). Either
	// way the guard must fire and return 403. t.Setenv handles cleanup.
	t.Setenv(FeatureFlagEnv, "false")
	// Sanity: the FF helper itself must report disabled for this test to be
	// meaningful. If this assertion fails, the test setup is broken and the
	// regression coverage is meaningless.
	if IsFeatureFlagEnabled() {
		t.Fatalf("IsFeatureFlagEnabled() = true with env=%q; test setup is broken", "false")
	}

	router := buildBriefingRouter(t)

	for _, tc := range briefingFFDisabledRoutes {
		t.Run(tc.name, func(t *testing.T) {
			path := substituteBriefingPathParams(t, tc.path)
			req := httptest.NewRequest(tc.method, path, nil)
			// Even with a valid X-User-ID the guard must short-circuit
			// before user/auth checks. Include a UUID to prove the guard
			// fires regardless of auth state.
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d (FF guard did not fire for %s %s)",
					w.Code, http.StatusForbidden, tc.method, tc.path)
			}
			body := w.Body.String()
			if !strings.Contains(body, "Pre-call briefing is not enabled") {
				t.Errorf("body = %q, want to contain %q", body, "Pre-call briefing is not enabled")
			}
			// Content-Type must be the problem+json envelope used elsewhere
			// in this package — guards against silent response-shape changes.
			if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
			}
		})
	}
}

func TestHandler_FeatureFlagDisabled_UnsetEnv(t *testing.T) {
	// Production default: env var unset. The FF helper treats unset the same
	// as "false". Without the routing fix, requests on this code path would
	// bypass the guard and reach the handlers (which would then 500 on the
	// nil pool we pass). The guard must fire and return 403.
	// Note: t.Setenv("") explicitly sets it to empty rather than unsetting,
	// which is functionally equivalent for IsFeatureFlagEnabled (both yield
	// "false"). We use t.Setenv with the empty string so the test does not
	// depend on the host's actual env state.
	t.Setenv(FeatureFlagEnv, "")

	router := buildBriefingRouter(t)
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/upcoming/"+meetingID+"/briefing", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (FF guard must fire when env is unset)",
			w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), "Pre-call briefing is not enabled") {
		t.Errorf("body = %q, want to contain %q", w.Body.String(), "Pre-call briefing is not enabled")
	}
}

// TestHandler_FlagMismatch_MetricEmitted proves the FF-guard middleware
// increments FlagMisconfigurationTotal when the browser header disagrees
// with the server-side env var. This is the production observability
// signal that catches "browser thinks feature is on, server says off"
// misconfigurations before users start seeing 403s in the wild.
//
// The test uses a custom prometheus.Registry so the promauto-registered
// metric is observable via testutil. The delta-before/after assertion
// keeps the test stable even when other tests in the package have
// already touched the same counter.
func TestHandler_FlagMismatch_MetricEmitted(t *testing.T) {
	// Swap prometheus.DefaultRegisterer to a private registry so the
	// metric in this test starts from a clean slate for the labels
	// we care about. Restore the original at the end.
	origReg := prometheus.DefaultRegisterer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	t.Cleanup(func() { prometheus.DefaultRegisterer = origReg })

	// Server says feature is OFF. Browser claims it's ON.
	t.Setenv(FeatureFlagEnv, "false")

	logger := zerolog.New(os.Stdout).Level(zerolog.Disabled)
	router := Handler(&logger, nil, nil)

	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/upcoming/"+meetingID+"/briefing", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-Browser-FF-Pre-Call-Briefing", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Guard must still short-circuit with 403 even when the browser
	// header is present.
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (FF guard must still fire on mismatch)",
			w.Code, http.StatusForbidden)
	}

	// Metric must be incremented. Labels match emitFlagMisconfiguration
	// signature: (server_flag_value, browser_flag_value, endpoint).
	const serverVal, browserVal, endpoint = "false", "true", "briefing_endpoint"
	before := testutil.ToFloat64(FlagMisconfigurationTotal.WithLabelValues(serverVal, browserVal, endpoint))
	// Re-issue the request to capture the increment delta — the previous
	// call already incremented once (we can't observe "before" retroactively
	// for a request that already fired). Take a second request.
	req2 := httptest.NewRequest(http.MethodGet, "/upcoming/"+uuid.New().String()+"/briefing", nil)
	req2.Header.Set("X-User-ID", uuid.New().String())
	req2.Header.Set("X-Browser-FF-Pre-Call-Briefing", "true")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	after := testutil.ToFloat64(FlagMisconfigurationTotal.WithLabelValues(serverVal, browserVal, endpoint))

	if after-before != 1 {
		t.Errorf("FlagMisconfigurationTotal delta = %v, want 1 (before=%v after=%v)",
			after-before, before, after)
	}
}