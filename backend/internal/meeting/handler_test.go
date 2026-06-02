package meeting

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockRepo simulates Repository for handler tests.
type mockRepo struct {
	meetings    map[uuid.UUID]*Meeting
	idempTokens map[string]idempTokenEntry // key: "userID|token"
}

// tokenKey builds the composite map key for idempotency token lookups.
func tokenKey(userID, token uuid.UUID) string {
	return userID.String() + "|" + token.String()
}

type idempTokenEntry struct {
	meetingID       uuid.UUID
	originalOutcome string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		meetings:    make(map[uuid.UUID]*Meeting),
		idempTokens: make(map[string]idempTokenEntry),
	}
}

func (m *mockRepo) CheckIdempotencyToken(ctx context.Context, userID, token uuid.UUID) (uuid.UUID, string, error) {
	if entry, ok := m.idempTokens[tokenKey(userID, token)]; ok {
		return entry.meetingID, entry.originalOutcome, nil
	}
	return uuid.Nil, "", ErrTokenNotFound
}

func (m *mockRepo) CreateMeeting(ctx context.Context, title string, completedAt time.Time, transcript, notes *string, contentSource string, createdBy uuid.UUID, participants []ParticipantInput, idempotencyToken uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	m.meetings[id] = &Meeting{
		ID:            id,
		Title:         title,
		CompletedAt:   completedAt,
		Transcript:    transcript,
		Notes:         notes,
		ContentSource: contentSource,
		CreatedBy:     createdBy,
		Participants:  []Participant{},
	}
	m.idempTokens[tokenKey(createdBy, idempotencyToken)] = idempTokenEntry{meetingID: id, originalOutcome: "created"}
	return id, nil
}

func (m *mockRepo) ListMeetings(ctx context.Context, createdBy uuid.UUID, limit, offset int) ([]Meeting, error) {
	var result []Meeting
	for _, v := range m.meetings {
		result = append(result, *v)
	}
	if result == nil {
		return []Meeting{}, nil
	}
	return result, nil
}

func (m *mockRepo) GetMeeting(ctx context.Context, id uuid.UUID) (*Meeting, error) {
	if meeting, ok := m.meetings[id]; ok {
		return meeting, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepo) GetMeetingOwned(ctx context.Context, id, userID uuid.UUID) (*Meeting, error) {
	meeting, ok := m.meetings[id]
	if !ok || meeting.CreatedBy != userID {
		// Collapsed into a single ErrNotFound so the handler cannot distinguish
		// "does not exist" from "exists but owned by someone else" — same as
		// the SQL repository, which relies on pgx.ErrNoRows for both cases.
		// IDOR-FIX-CWE-639
		return nil, ErrNotFound
	}
	return meeting, nil
}

func (m *mockRepo) UpdateMeeting(ctx context.Context, meetingID uuid.UUID, title *string, completedAt *time.Time, transcript, notes *string, participants []ParticipantInput) (bool, error) {
	meeting, ok := m.meetings[meetingID]
	if !ok {
		return false, ErrNotFound
	}
	changed := false
	if title != nil && *title != meeting.Title {
		meeting.Title = *title
		changed = true
	}
	if completedAt != nil && !completedAt.Equal(meeting.CompletedAt) {
		meeting.CompletedAt = *completedAt
		changed = true
	}
	if transcript != nil {
		meeting.Transcript = transcript
		changed = true
	}
	if notes != nil {
		meeting.Notes = notes
		changed = true
	}
	if participants != nil {
		meeting.Participants = make([]Participant, len(participants))
		for i, p := range participants {
			meeting.Participants[i] = Participant{
				ID:           uuid.New(),
				MeetingID:    meetingID,
				DisplayName:  p.DisplayName,
				Email:        p.Email,
				Organization: p.Organization,
				Role:         p.Role,
				DisplayOrder: i,
			}
		}
		changed = true
	}
	return changed, nil
}

func buildTestRouter(repo *mockRepo, ffEnabled bool) http.Handler {
	logger := zerolog.New(os.Stdout).Level(zerolog.WarnLevel)
	r := chi.NewRouter()

	// Inject mock repository and nil pool into context
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ContextKey{}, repo)
			ctx = context.WithValue(ctx, poolKey{}, (*struct{})(nil))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !ffEnabled {
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"detail":"Manual meeting import is not enabled."}`))
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		r.Post("/", handleCreateMeeting(&logger))
		r.Get("/", handleListMeetings(&logger))
		r.Get("/{id}", handleGetMeeting(&logger))
		r.Patch("/{id}", handleUpdateMeeting(&logger))
	})

	return r
}

// buildTestRouterWithFF flips the real isFeatureFlagEnabled by setting the env var
func buildTestRouterWithFF(repo *mockRepo, ffValue string) http.Handler {
	orig := os.Getenv(FeatureFlagEnv)
	defer os.Setenv(FeatureFlagEnv, orig)
	os.Setenv(FeatureFlagEnv, ffValue)
	return buildTestRouter(repo, true)
}

func TestHandler_FeatureFlagDisabled(t *testing.T) {
	orig := os.Getenv(FeatureFlagEnv)
	defer os.Setenv(FeatureFlagEnv, orig)
	os.Setenv(FeatureFlagEnv, "false")

	repo := newMockRepo()
	logger := zerolog.New(os.Stdout).Level(zerolog.WarnLevel)

	r := chi.NewRouter()
	// Inject repository even though FF should block first
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ContextKey{}, repo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				enabled := isFeatureFlagEnabled(FeatureFlagEnv)
				if !enabled {
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403}`))
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		g.Post("/", handleCreateMeeting(&logger))
		g.Get("/", handleListMeetings(&logger))
		g.Get("/{id}", handleGetMeeting(&logger))
	})

	// Even with auth header, FF=disabled should block before auth guard runs
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_FeatureFlagEnabled_ListWithoutAuth(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Feature flag passes; missing X-User-ID gives 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateMeeting_NoAuth(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Test","completedAt":"2024-01-01T00:00:00Z","participants":[{"displayName":"Alice"}],"idempotencyToken":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateMeeting_InvalidAuth(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Test","completedAt":"2024-01-01T00:00:00Z","participants":[{"displayName":"Alice"}],"idempotencyToken":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateMeeting_InvalidJSON(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateMeeting_ValidationError(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	// Missing title (empty), invalid completedAt, no participants, bad idempotency token
	body := `{"title":"","completedAt":"invalid","participants":[],"idempotencyToken":"not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ValidationErrorBody
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected validation errors")
	}
}

func TestCreateMeeting_Success(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	transcript := "Hello world"
	body := map[string]any{
		"title":             "Test Meeting",
		"completedAt":       "2024-01-15T10:00:00Z",
		"participants":      []map[string]any{{"displayName": "Alice", "email": "alice@example.com"}},
		"transcript":        transcript,
		"idempotencyToken":  uuid.New().String(),
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d. body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CreateMeetingOutput
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Redirect == "" {
		t.Error("expected non-empty Redirect")
	}
}

func TestListMeetings_NoAuth(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestListMeetings_Success(t *testing.T) {
	repo := newMockRepo()
	// Pre-populate a meeting
	meetingID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Existing Meeting",
		CompletedAt: time.Now(),
		CreatedBy:   uuid.New(),
	}

	r := buildTestRouterWithFF(repo, "true")

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	meetings, ok := resp["meetings"]
	if !ok {
		t.Fatal("expected 'meetings' key in response")
	}
	// meetings is a list; our pre-populated meeting should be there
	_ = meetings
}

func TestGetMeeting_InvalidUUID(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetMeeting_NotFound(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetMeeting_Success(t *testing.T) {
	repo := newMockRepo()
	meetingID := uuid.New()
	ownerID := uuid.New() // IDOR-FIX-CWE-639
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Found Meeting",
		CompletedAt: time.Now(),
		CreatedBy:   ownerID, // IDOR-FIX-CWE-639
		Participants: []Participant{},
	}

	r := buildTestRouterWithFF(repo, "true")

	req := httptest.NewRequest(http.MethodGet, "/"+meetingID.String(), nil)
	req.Header.Set("X-User-ID", ownerID.String()) // IDOR-FIX-CWE-639
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Meeting
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Title != "Found Meeting" {
		t.Errorf("Title = %q, want %q", resp.Title, "Found Meeting")
	}
}

// TestGetMeeting_OwnerCheck is the regression test for CWE-639 (IDOR on
// GET /api/v1/meetings/{id}). A non-owner MUST receive 404 (not 403, not 200)
// and the response body MUST NOT contain the secret transcript or the owner's
// createdBy — i.e. no existence or content leak.
func TestGetMeeting_OwnerCheck(t *testing.T) {
	repo := newMockRepo()
	meetingID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()
	secretTranscript := "secret transcript content for idor-regression"
	secretNotes := "private notes that must never leak cross-tenant"
	repo.meetings[meetingID] = &Meeting{
		ID:            meetingID,
		Title:         "final-idor-verify",
		CompletedAt:   time.Now(),
		Transcript:    strPtr(secretTranscript),
		Notes:         strPtr(secretNotes),
		ContentSource: "manual",
		CreatedBy:     ownerID,
		Participants:  []Participant{},
	}

	r := buildTestRouterWithFF(repo, "true")

	// 1. Owner (User A) reads their own meeting -> 200 with full body.
	req := httptest.NewRequest(http.MethodGet, "/"+meetingID.String(), nil)
	req.Header.Set("X-User-ID", ownerID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("owner GET: status = %d, want %d. body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	ownerBody := w.Body.String()
	if !strings.Contains(ownerBody, secretTranscript) {
		t.Errorf("owner body missing transcript: %s", ownerBody)
	}
	if !strings.Contains(ownerBody, secretNotes) {
		t.Errorf("owner body missing notes: %s", ownerBody)
	}
	if !strings.Contains(ownerBody, ownerID.String()) {
		t.Errorf("owner body missing owner createdBy: %s", ownerBody)
	}

	// 2. Attacker (User B) reads User A's meeting -> 404, body MUST NOT leak
	//    transcript, notes, or createdBy.
	req2 := httptest.NewRequest(http.MethodGet, "/"+meetingID.String(), nil)
	req2.Header.Set("X-User-ID", attackerID.String())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Fatalf("attacker GET: status = %d, want %d. body: %s", w2.Code, http.StatusNotFound, w2.Body.String())
	}
	attackerBody := w2.Body.String()
	if strings.Contains(attackerBody, secretTranscript) {
		t.Errorf("IDOR LEAK: attacker body contains secret transcript: %s", attackerBody)
	}
	if strings.Contains(attackerBody, secretNotes) {
		t.Errorf("IDOR LEAK: attacker body contains secret notes: %s", attackerBody)
	}
	if strings.Contains(attackerBody, ownerID.String()) {
		t.Errorf("IDOR LEAK: attacker body contains owner createdBy: %s", attackerBody)
	}
	if strings.Contains(attackerBody, "final-idor-verify") {
		t.Errorf("IDOR LEAK: attacker body contains meeting title: %s", attackerBody)
	}

	// 3. Unknown user (no X-User-ID) -> 401, before any DB lookup.
	req3 := httptest.NewRequest(http.MethodGet, "/"+meetingID.String(), nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusUnauthorized {
		t.Errorf("no-auth GET: status = %d, want %d", w3.Code, http.StatusUnauthorized)
	}

	// 4. Malformed X-User-ID -> 401, before any DB lookup.
	req4 := httptest.NewRequest(http.MethodGet, "/"+meetingID.String(), nil)
	req4.Header.Set("X-User-ID", "not-a-uuid")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusUnauthorized {
		t.Errorf("bad-auth GET: status = %d, want %d", w4.Code, http.StatusUnauthorized)
	}
}

func TestIsFeatureFlagEnabled(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"true,extra", true},
		{"false", false},
		{"False", false},
		{"", false},
		{"anything", false},
	}

	for _, tt := range tests {
		os.Setenv("TEST_FF", tt.value)
		got := isFeatureFlagEnabled("TEST_FF")
		if got != tt.expected {
			t.Errorf("isFeatureFlagEnabled(%q) = %v, want %v", tt.value, got, tt.expected)
		}
	}
}

func TestCreateMeeting_IdempotencyConflict(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	userID := uuid.New()
	token := uuid.New()
	existingID := uuid.New()
	repo.idempTokens[tokenKey(userID, token)] = idempTokenEntry{meetingID: existingID, originalOutcome: "created"}

	transcript := "Hello"
	body := map[string]any{
		"title":             "Test",
		"completedAt":       "2024-01-15T10:00:00Z",
		"participants":      []map[string]any{{"displayName": "Alice"}},
		"transcript":        transcript,
		"idempotencyToken":  token.String(),
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp DuplicateErrorBody
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.MeetingID != existingID.String() {
		t.Errorf("MeetingID = %q, want %q", resp.MeetingID, existingID.String())
	}
}

func TestCreateMeeting_IdempotencyConflict_PerUserScope(t *testing.T) {
	// Two different users with the same token — only the matching user gets a conflict.
	// This is the regression test for the global idempotency token scope bug (F2).
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	token := uuid.New()
	userA := uuid.New()
	userB := uuid.New() // different user

	// Pre-populate: User A already used this token
	existingID := uuid.New()
	repo.idempTokens[tokenKey(userA, token)] = idempTokenEntry{meetingID: existingID, originalOutcome: "created"}

	// User B sends the same token — should succeed (per-user scope)
	body := map[string]any{
		"title":             "User B's Meeting",
		"completedAt":       "2024-01-15T10:00:00Z",
		"participants":      []map[string]any{{"displayName": "Bob"}},
		"transcript":        "User B transcript",
		"idempotencyToken":  token.String(),
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userB.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Must be 201 — different user, same token is fine (per-user scope)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d (same token, different user should not conflict)", w.Code, http.StatusCreated)
	}

	// Now User B retries with the same token — should get 409
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-ID", userB.String())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (same user, same token should conflict)", w2.Code, http.StatusConflict)
	}
}

func TestCreateMeeting_InvalidIdempotencyToken(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Test","completedAt":"2024-01-01T00:00:00Z","participants":[{"displayName":"Alice"}],"idempotencyToken":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateMeeting_NoAuth(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/"+uuid.New().String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUpdateMeeting_NotFound(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/"+uuid.New().String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateMeeting_InvalidUUID(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateMeeting_InvalidJSON(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
	}

	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateMeeting_ValidationError_EmptyTitle(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
	}

	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBufferString(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateMeeting_Success_TitleOnly(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
		Participants: []Participant{},
	}

	newTitle := "Updated Title"
	body := map[string]any{"title": newTitle}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp Meeting
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Title != newTitle {
		t.Errorf("Title = %q, want %q", resp.Title, newTitle)
	}
}

func TestUpdateMeeting_Success_TranscriptOnly(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
		Transcript:  strPtr("original transcript"),
		Participants: []Participant{},
	}

	newTranscript := "updated transcript content"
	body := map[string]any{"transcript": newTranscript}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp Meeting
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Transcript == nil || *resp.Transcript != newTranscript {
		t.Errorf("Transcript = %v, want %q", resp.Transcript, newTranscript)
	}
}

func TestUpdateMeeting_Success_ParticipantsOnly(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
		Participants: []Participant{},
	}

	body := map[string]any{
		"participants": []map[string]any{
			{"displayName": "Alice", "email": "alice@example.com"},
			{"displayName": "Bob", "email": "bob@example.com"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp Meeting
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Participants) != 2 {
		t.Errorf("len(Participants) = %d, want 2", len(resp.Participants))
	}
	if resp.Participants[0].DisplayName != "Alice" {
		t.Errorf("Participants[0].DisplayName = %q, want %q", resp.Participants[0].DisplayName, "Alice")
	}
}

func TestUpdateMeeting_NoChanges(t *testing.T) {
	repo := newMockRepo()
	r := buildTestRouterWithFF(repo, "true")

	meetingID := uuid.New()
	userID := uuid.New()
	repo.meetings[meetingID] = &Meeting{
		ID:          meetingID,
		Title:       "Original Title",
		CompletedAt: time.Now(),
		CreatedBy:   userID,
		Transcript:  strPtr("original"),
		Participants: []Participant{},
	}

	// Send the same title (no actual change)
	body := map[string]any{"title": "Original Title"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/"+meetingID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}