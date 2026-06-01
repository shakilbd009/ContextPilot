package memory

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WorkerConfig configures the background processing worker.
type WorkerConfig struct {
	PollInterval   time.Duration // How often to poll for queued jobs
	MaxRetries     int           // Max automatic retries per job (default 3)
	InitialBackoff time.Duration // Initial retry delay (default 1s)
	MaxBackoff     time.Duration // Maximum retry delay (default 30s)
	BackoffMult    float64       // Backoff multiplier (default 2.0)
	JitterFraction float64       // Jitter as fraction of delay (default 0.1 = ±10%)
}

// DefaultWorkerConfig returns a sensible default configuration.
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		PollInterval:   5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffMult:    2.0,
		JitterFraction: 0.1,
	}
}

// Worker is the background queue consumer that processes memory jobs.
type Worker struct {
	repo      *Repository
	processor MemoryProcessor
	log       *zerolog.Logger
	cfg       WorkerConfig
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewWorker returns a new background worker.
func NewWorker(repo *Repository, processor MemoryProcessor, log *zerolog.Logger, cfg WorkerConfig) *Worker {
	return &Worker{
		repo:      repo,
		processor: processor,
		log:       log,
		cfg:       cfg,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// Start launches the worker in a background goroutine.
func (w *Worker) Start(ctx context.Context) {
	w.log.Info().Msg("memory worker starting")
	go w.run(ctx)
}

// Stop signals the worker to shut down gracefully.
func (w *Worker) Stop() {
	close(w.stopCh)
	<-w.doneCh
	w.log.Info().Msg("memory worker stopped")
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	jobs, err := w.repo.GetQueuedJobs(ctx, 5)
	if err != nil {
		w.log.Error().Err(err).Msg("failed to fetch queued jobs")
		return
	}

	for _, job := range jobs {
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job MemoryProcessingJob) {
	logger := w.log.With().
		Str("job_id", job.ID.String()).
		Str("meeting_id", job.MeetingID.String()).
		Str("trigger_type", string(job.TriggerType)).
		Logger()

	// Claim the job atomically
	claimed, err := w.repo.PickJob(ctx, job.ID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to pick job")
		return
	}
	if claimed == nil {
		// Another worker picked it
		return
	}

	start := time.Now()

	// Fetch meeting source material
	meeting, err := w.getMeeting(ctx, job.MeetingID)
	if err != nil {
		w.handlePermanentFailure(ctx, claimed, "meeting not found: "+err.Error(), logger)
		return
	}

	// Build processor input
	processorInput := w.buildProcessorInput(job, meeting)

	// Call processor
	output, procErr := w.processor.Process(ctx, processorInput)

	if procErr != nil {
		w.handleProcessingError(ctx, claimed, procErr, logger)
		return
	}

	// Create new version
	versionID, versionNum, err := w.repo.CreateVersion(
		ctx,
		job.MeetingID,
		&job.ID,
		job.TriggerType,
		output.Content,
		job.PreviousVersionID,
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create memory version")
		w.handlePermanentFailure(ctx, claimed, "failed to persist memory: "+err.Error(), logger)
		return
	}

	// Record prior memory inputs
	if len(output.Evidence) > 0 {
		_ = output.Evidence // evidence stored separately if needed
	}

	// Create conflict records if any
	for _, conflict := range output.Conflicts {
		conflictID, err := w.repo.CreateConflict(ctx, job.MeetingID, &versionID, conflict)
		if err != nil {
			logger.Error().Err(err).Str("conflict_id", conflictID.String()).Msg("failed to record conflict")
		}
	}

	// Mark job complete
	if err := w.repo.UpdateJobCompleted(ctx, job.ID, versionID, nil); err != nil {
		logger.Error().Err(err).Msg("failed to mark job completed")
	}

	elapsed := time.Since(start)
	ProcessingDurationMs.Observe(float64(elapsed.Milliseconds()))
	ProcessingCompletedTotal.WithLabelValues(string(job.TriggerType), "completed").Inc()

	logger.Info().
		Int("version_number", versionNum).
		Int64("duration_ms", elapsed.Milliseconds()).
		Msg("memory job completed")
}

func (w *Worker) getMeeting(ctx context.Context, meetingID uuid.UUID) (*meetingInfo, error) {
	// This will call into the meeting repository.
	// We avoid an import cycle by fetching directly via the pool from the repo.
	// The repo exposes the pool getter indirectly through a helper.
	return w.repo.GetMeetingInfo(ctx, meetingID)
}

type meetingInfo struct {
	ID          uuid.UUID
	Title       string
	Transcript  *string
	Notes       *string
	CompletedAt time.Time
	UpdatedAt   time.Time
	Participants []struct {
		ID           uuid.UUID
		DisplayName  string
		Email        *string
		Organization *string
	}
}

func (w *Worker) buildProcessorInput(job MemoryProcessingJob, meeting *meetingInfo) MemoryProcessorInput {
	participants := make([]ParticipantInfo, len(meeting.Participants))
	for i, p := range meeting.Participants {
		participants[i] = ParticipantInfo{
			ID:           p.ID,
			DisplayName:  p.DisplayName,
			Email:        p.Email,
			Organization: p.Organization,
		}
	}

	return MemoryProcessorInput{
		MeetingID:     meeting.ID,
		Title:        meeting.Title,
		Transcript:   meeting.Transcript,
		Notes:        meeting.Notes,
		Participants: participants,
		CorrelationID: job.CorrelationID,
	}
}

func (w *Worker) handleProcessingError(ctx context.Context, job *MemoryProcessingJob, procErr error, logger zerolog.Logger) {
	// Classify error as transient or permanent
	failureClass := classifyError(procErr)
	var failureReason string

	if failureClass == FailureClassTransientProvider || failureClass == FailureClassTimeout {
		// Transient — schedule retry
		retryCount := job.RetryCount + 1
		if retryCount >= job.MaxRetries {
			// Exhausted retries
			if err := w.repo.UpdateJobRetryExhausted(ctx, job.ID, procErr.Error()); err != nil {
				logger.Error().Err(err).Msg("failed to mark job retry_exhausted")
			}
			ProcessingRetryExhaustedTotal.WithLabelValues(string(job.TriggerType)).Inc()
			logger.Warn().Int("retry_count", retryCount).Msg("job retry exhausted")
		} else {
			// Schedule next retry with exponential backoff
			delay := w.computeBackoff(retryCount)
			nextRetryAt := time.Now().Add(delay)
			if err := w.repo.UpdateJobRetrying(ctx, job.ID, retryCount, nextRetryAt, procErr.Error()); err != nil {
				logger.Error().Err(err).Msg("failed to update job to retrying")
			}
			ProcessingRetriedTotal.WithLabelValues(string(job.TriggerType)).Inc()
			logger.Info().
				Int("retry_count", retryCount).
				Time("next_retry_at", nextRetryAt).
				Msg("job scheduled for retry")
		}
		ProcessingFailedTotal.WithLabelValues(string(job.TriggerType), string(failureClass)).Inc()
		return
	}

	// Permanent failure
	failureReason = procErr.Error()
	if err := w.repo.UpdateJobFailed(ctx, job.ID, failureReason); err != nil {
		logger.Error().Err(err).Msg("failed to mark job failed")
	}
	ProcessingFailedTotal.WithLabelValues(string(job.TriggerType), string(failureClass)).Inc()
	logger.Error().Err(procErr).Msg("job permanently failed")
}

func (w *Worker) handlePermanentFailure(ctx context.Context, job *MemoryProcessingJob, reason string, logger zerolog.Logger) {
	if err := w.repo.UpdateJobFailed(ctx, job.ID, reason); err != nil {
		logger.Error().Err(err).Msg("failed to mark job failed")
	}
	ProcessingFailedTotal.WithLabelValues(string(job.TriggerType), string(FailureClassPermanentProcessing)).Inc()
}

// computeBackoff computes the retry delay with exponential backoff and jitter.
// Formula: min(initial * mult^retryCount, maxBackoff) * (1 ± jitterFraction)
func (w *Worker) computeBackoff(retryCount int) time.Duration {
	delay := float64(w.cfg.InitialBackoff)
	for i := 0; i < retryCount; i++ {
		delay *= w.cfg.BackoffMult
	}
	if delay > float64(w.cfg.MaxBackoff) {
		delay = float64(w.cfg.MaxBackoff)
	}
	// Apply jitter
	// #nosec G404 -- math/rand is the correct PRNG for non-security jitter; not a secret
	jitter := delay * w.cfg.JitterFraction * (2*rand.Float64() - 1)
	delay = delay + jitter
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay)
}

// classifyError determines whether an error is transient or permanent.
func classifyError(err error) FailureClass {
	if err == nil {
		return FailureClassPermanentProcessing
	}
	errStr := err.Error()
	switch {
	case err == ErrProviderUnavailable:
		return FailureClassTransientProvider
	case err == context.DeadlineExceeded, err == context.Canceled:
		return FailureClassTimeout
	case err == ErrMalformedInput, err == ErrMeetingNotFound:
		return FailureClassPermanentProcessing
	case contains(errStr, "connection refused"), contains(errStr, "timeout"),
		contains(errStr, "503"), contains(errStr, "502"), contains(errStr, "504"),
		contains(errStr, "network"), contains(errStr, "i/o"):
		return FailureClassTransientProvider
	default:
		return FailureClassPermanentProcessing
	}
}