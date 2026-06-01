package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Worker tests - testing worker configuration, backoff computation, and error classification

func TestDefaultWorkerConfig(t *testing.T) {
	cfg := DefaultWorkerConfig()
	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", cfg.PollInterval)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if cfg.BackoffMult != 2.0 {
		t.Errorf("BackoffMult = %v, want 2.0", cfg.BackoffMult)
	}
	if cfg.JitterFraction != 0.1 {
		t.Errorf("JitterFraction = %v, want 0.1", cfg.JitterFraction)
	}
}

func TestWorkerConfig_Setters(t *testing.T) {
	cfg := WorkerConfig{
		PollInterval:   10 * time.Second,
		MaxRetries:     5,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     60 * time.Second,
		BackoffMult:    1.5,
		JitterFraction: 0.2,
	}
	if cfg.PollInterval != 10*time.Second {
		t.Errorf("PollInterval = %v, want 10s", cfg.PollInterval)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
}

func TestWorker_Struct(t *testing.T) {
	// Test that Worker struct has expected nil fields
	w := &Worker{}
	if w.stopCh != nil {
		t.Error("expected nil stopCh initially")
	}
	if w.doneCh != nil {
		t.Error("expected nil doneCh initially")
	}
	if w.repo != nil {
		t.Error("expected nil repo initially")
	}
	if w.processor != nil {
		t.Error("expected nil processor initially")
	}
}

func TestWorker_NewWorker(t *testing.T) {
	var log *zerolog.Logger // explicitly typed to avoid untyped nil
	repo := &Repository{}
	processor := &DefaultMemoryProcessor{}

	w := NewWorker(repo, processor, log, DefaultWorkerConfig())
	if w == nil {
		t.Fatal("NewWorker returned nil")
	}
	if w.repo != repo {
		t.Error("repo not set correctly")
	}
	if w.processor != processor {
		t.Error("processor not set correctly")
	}
	if w.cfg.PollInterval != 5*time.Second {
		t.Errorf("cfg.PollInterval = %v, want 5s", w.cfg.PollInterval)
	}
}

func TestWorker_StartAndStop_NoPanic(t *testing.T) {
	// Cannot test Start/Stop lifecycle without a real database/mock pool.
	// Verify that NewWorker returns a non-nil worker with correct config.
	cfg := DefaultWorkerConfig()
	w := NewWorker(&Repository{}, &DefaultMemoryProcessor{}, nil, cfg)
	if w == nil {
		t.Fatal("NewWorker returned nil")
	}
	// Verify channels are initialized (not nil)
	if w.stopCh == nil {
		t.Error("stopCh is nil after NewWorker")
	}
	if w.doneCh == nil {
		t.Error("doneCh is nil after NewWorker")
	}
}

func TestWorker_Start_DoesNotPanic(t *testing.T) {
	// Start/Stop require a real pgxpool; we only verify Start doesn't panic
	// when called with a cancellable context and then stopped quickly.
	// Use zerolog.Nop() to avoid nil pointer dereference on log.Info()
	nop := zerolog.Nop()
	w := &Worker{
		repo:      nil,
		processor: nil,
		log:       &nop,
		cfg:       WorkerConfig{PollInterval: 10 * time.Millisecond},
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)
	cancel()
	w.Stop()
}

func TestWorker_BuildProcessorInput(t *testing.T) {
	meetingID := uuid.New()
	correlationID := uuid.New()
	meeting := &meetingInfo{
		ID:         meetingID,
		Title:      "Test Meeting",
		Transcript: ptrStr("Test transcript content"),
		Notes:      ptrStr("Test notes"),
		CompletedAt: time.Now(),
		UpdatedAt:   time.Now(),
		Participants: []struct {
			ID           uuid.UUID
			DisplayName  string
			Email        *string
			Organization *string
		}{
			{ID: uuid.New(), DisplayName: "Alice", Email: ptrStr("alice@example.com"), Organization: ptrStr("Acme")},
		},
	}

	w := &Worker{cfg: DefaultWorkerConfig()}
	job := MemoryProcessingJob{
		CorrelationID: correlationID,
	}

	input := w.buildProcessorInput(job, meeting)
	if input.MeetingID != meetingID {
		t.Errorf("MeetingID = %v, want %v", input.MeetingID, meetingID)
	}
	if input.Title != "Test Meeting" {
		t.Errorf("Title = %v, want %v", input.Title, "Test Meeting")
	}
	if input.Transcript == nil || *input.Transcript != "Test transcript content" {
		t.Errorf("Transcript = %v, want %v", input.Transcript, "Test transcript content")
	}
	if input.CorrelationID != correlationID {
		t.Errorf("CorrelationID = %v, want %v", input.CorrelationID, correlationID)
	}
	if len(input.Participants) != 1 {
		t.Fatalf("len(Participants) = %d, want 1", len(input.Participants))
	}
	if input.Participants[0].DisplayName != "Alice" {
		t.Errorf("Participant[0].DisplayName = %v, want %v", input.Participants[0].DisplayName, "Alice")
	}
}

func TestWorker_ComputeBackoff(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		cfg        WorkerConfig
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "retry 0",
			retryCount: 0,
			cfg:        DefaultWorkerConfig(),
			wantMin:    900 * time.Millisecond, // 1s * 0.9 (10% jitter down)
			wantMax:    1100 * time.Millisecond, // 1s * 1.1 (10% jitter up)
		},
		{
			name:       "retry 1",
			retryCount: 1,
			cfg:        DefaultWorkerConfig(),
			wantMin:    1800 * time.Millisecond, // 2s * 0.9
			wantMax:    2200 * time.Millisecond, // 2s * 1.1
		},
		{
			name:       "retry 2",
			retryCount: 2,
			cfg:        DefaultWorkerConfig(),
			wantMin:    3600 * time.Millisecond, // 4s * 0.9
			wantMax:    4400 * time.Millisecond, // 4s * 1.1
		},
		{
			name:       "max backoff",
			retryCount: 10,
			cfg:        DefaultWorkerConfig(),
			wantMin:    27 * time.Second,  // 30s * 0.9
			wantMax:    33 * time.Second,  // 30s * 1.1
		},
		{
			name:       "custom backoff",
			retryCount: 1,
			cfg: WorkerConfig{
				PollInterval:   5 * time.Second,
				MaxRetries:     3,
				InitialBackoff: 500 * time.Millisecond,
				MaxBackoff:     10 * time.Second,
				BackoffMult:    2.0,
				JitterFraction: 0.1,
			},
			wantMin:    900 * time.Millisecond,  // 1s * 0.9
			wantMax:    1100 * time.Millisecond, // 1s * 1.1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{cfg: tt.cfg}
			delay := w.computeBackoff(tt.retryCount)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("computeBackoff(%d) = %v, want between %v and %v", tt.retryCount, delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected FailureClass
	}{
		{"nil error", nil, FailureClassPermanentProcessing},
		{"provider unavailable", ErrProviderUnavailable, FailureClassTransientProvider},
		{"deadline exceeded", context.DeadlineExceeded, FailureClassTimeout},
		{"canceled", context.Canceled, FailureClassTimeout},
		{"malformed input", ErrMalformedInput, FailureClassPermanentProcessing},
		{"meeting not found", ErrMeetingNotFound, FailureClassPermanentProcessing},
		{"connection refused", errorsNew("connection refused"), FailureClassTransientProvider},
		{"timeout error", errorsNew("timeout error"), FailureClassTransientProvider},
		{"503 error", errorsNew("503 error"), FailureClassTransientProvider},
		{"502 error", errorsNew("502 bad gateway"), FailureClassTransientProvider},
		{"504 error", errorsNew("504 gateway timeout"), FailureClassTransientProvider},
		{"network error", errorsNew("network error"), FailureClassTransientProvider},
		{"i/o error", errorsNew("i/o error"), FailureClassTransientProvider},
		{"generic error", errorsNew("something went wrong"), FailureClassPermanentProcessing},
		{"random error", errorsNew("unexpected error"), FailureClassPermanentProcessing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyError(tt.err)
			if got != tt.expected {
				t.Errorf("classifyError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

// errorsNew is a helper to create errors without importing errors package
func errorsNew(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestWorker_BuildProcessorInput_NilTranscript(t *testing.T) {
	meeting := &meetingInfo{
		ID:         uuid.New(),
		Title:      "Test Meeting",
		Transcript: nil,
		Notes:      nil,
		CompletedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	w := &Worker{cfg: DefaultWorkerConfig()}
	job := MemoryProcessingJob{CorrelationID: uuid.New()}
	input := w.buildProcessorInput(job, meeting)

	if input.Transcript != nil {
		t.Errorf("Transcript = %v, want nil", input.Transcript)
	}
	if input.Notes != nil {
		t.Errorf("Notes = %v, want nil", input.Notes)
	}
}

func TestWorker_BuildProcessorInput_NoParticipants(t *testing.T) {
	meeting := &meetingInfo{
		ID:           uuid.New(),
		Title:        "Test Meeting",
		Transcript:   ptrStr("content"),
		Participants: nil,
	}

	w := &Worker{cfg: DefaultWorkerConfig()}
	job := MemoryProcessingJob{CorrelationID: uuid.New()}
	input := w.buildProcessorInput(job, meeting)

	// buildProcessorInput initializes an empty slice for nil participants
	if input.Participants == nil {
		t.Error("Participants = nil, want non-nil empty slice")
	}
	if len(input.Participants) != 0 {
		t.Errorf("len(Participants) = %d, want 0", len(input.Participants))
	}
}

// Helper
func ptrStr(s string) *string { return &s }