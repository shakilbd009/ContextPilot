package briefing

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WorkerConfig configures the background briefing processing worker.
type WorkerConfig struct {
	PollInterval   time.Duration
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffMult    float64
	JitterFraction float64
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

// BriefingService generates briefing content from source meeting memories.
type BriefingService interface {
	Generate(ctx context.Context, input BriefingGeneratorInput) (*BriefingContent, BriefingResult, []uuid.UUID, error)
}

// BriefingGeneratorInput is the input to the briefing generator.
type BriefingGeneratorInput struct {
	UpcomingMeetingID   uuid.UUID
	UpcomingTitle       string
	UpcomingScheduledStart time.Time
	SourceMeetingIDs    []uuid.UUID
	ExcludedSourceIDs   []uuid.UUID
}

// Worker is the background queue consumer that processes briefing generation jobs.
type Worker struct {
	repo      *Repository
	service  BriefingService
	log      *zerolog.Logger
	cfg      WorkerConfig
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewWorker returns a new background briefing worker.
func NewWorker(repo *Repository, service BriefingService, log *zerolog.Logger, cfg WorkerConfig) *Worker {
	return &Worker{
		repo:     repo,
		service:  service,
		log:      log,
		cfg:      cfg,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start launches the worker in a background goroutine.
func (w *Worker) Start(ctx context.Context) {
	w.log.Info().Msg("briefing worker starting")
	go w.run(ctx)
}

// Stop signals the worker to shut down gracefully.
func (w *Worker) Stop() {
	close(w.stopCh)
	<-w.doneCh
	w.log.Info().Msg("briefing worker stopped")
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
		w.log.Error().Err(err).Msg("failed to fetch queued briefing jobs")
		return
	}

	for _, job := range jobs {
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job BriefingProcessingJob) {
	logger := w.log.With().
		Str("job_id", job.ID.String()).
		Str("upcoming_meeting_id", job.UpcomingMeetingID.String()).
		Str("trigger_type", string(job.TriggerType)).
		Logger()

	logger.Info().
		Str("upcoming_meeting_id", job.UpcomingMeetingID.String()).
		Str("trigger_type", string(job.TriggerType)).
		Int("source_count_expected", 0). // actual count derived inside service; 0 is placeholder until then
		Msg("briefing.generation.started")

	// Claim the job atomically
	claimed, err := w.repo.PickJob(ctx, job.ID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to pick briefing job")
		return
	}
	if claimed == nil {
		// Another worker picked it
		return
	}

	start := time.Now()

	// Build generator input
	input := BriefingGeneratorInput{
		UpcomingMeetingID: job.UpcomingMeetingID,
	}

	// Call service to generate briefing content
	content, result, sourceVersionIDs, genErr := w.service.Generate(ctx, input)

	var prepStatus PreparationStatus
	switch result {
	case BriefingResultReady:
		prepStatus = PreparationStatusReady
	case BriefingResultReadyCaveats:
		prepStatus = PreparationStatusReadyCaveats
	case BriefingResultNoPriorMemory:
		prepStatus = PreparationStatusNoPriorMemory
	case BriefingResultFailed:
		prepStatus = PreparationStatusFailed
	}

	if genErr != nil {
		w.handleFailure(ctx, claimed, genErr.Error(), logger, result, prepStatus, start)
		return
	}

	// Create briefing version
	versionID, versionNum, err := w.repo.CreateBriefingVersion(
		ctx,
		job.UpcomingMeetingID,
		job.TriggerType,
		result,
		prepStatus,
		*content,
		len(sourceVersionIDs),
		sourceVersionIDs,
		job.PreviousVersionID,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create briefing version")
		w.handleFailure(ctx, claimed, "failed to persist briefing: "+err.Error(), logger, result, prepStatus, start)
		return
	}

	// Mark job complete
	if err := w.repo.UpdateJobCompleted(ctx, job.ID, versionID, nil); err != nil {
		logger.Error().Err(err).Msg("failed to mark briefing job completed")
	}

	elapsed := time.Since(start)
	BriefingGenerationDurationMs.WithLabelValues(string(result), string(FailureClassNone), fmt.Sprintf("%d", len(sourceVersionIDs)), string(job.TriggerType)).Observe(float64(elapsed.Milliseconds()))
	BriefingProcessingCompletedTotal.WithLabelValues(string(result), string(FailureClassNone)).Inc()

	logger.Info().
		Str("upcoming_meeting_id", job.UpcomingMeetingID.String()).
		Str("status", string(result)).
		Int("version_number", versionNum).
		Int("source_count_used", len(sourceVersionIDs)).
		Int64("duration_ms", elapsed.Milliseconds()).
		Msg("briefing.generation.completed")

	if result == BriefingResultNoPriorMemory {
		logger.Info().
			Str("upcoming_meeting_id", job.UpcomingMeetingID.String()).
			Msg("briefing.no_prior_memory_shell.generated")
	}
}

func (w *Worker) handleFailure(ctx context.Context, job *BriefingProcessingJob, reason string, logger zerolog.Logger, result BriefingResult, prepStatus PreparationStatus, start time.Time) {
	failureClass := classifyBriefingError(reason)

	if failureClass == FailureClassTimeout || failureClass == FailureClassUpstreamError {
		retryCount := job.RetryCount + 1
		if retryCount >= job.MaxRetries {
			if err := w.repo.UpdateJobFailed(ctx, job.ID, reason); err != nil {
				logger.Error().Err(err).Msg("failed to mark briefing job retry_exhausted")
			}
			BriefingProcessingRetryExhaustedTotal.WithLabelValues(string(job.TriggerType)).Inc()
			logger.Warn().Int("retry_count", retryCount).Msg("briefing job retry exhausted")
		} else {
			delay := w.computeBackoff(retryCount)
			nextRetryAt := time.Now().Add(delay)
			if err := w.repo.UpdateJobRetrying(ctx, job.ID, retryCount, nextRetryAt, reason); err != nil {
				logger.Error().Err(err).Msg("failed to update briefing job to retrying")
			}
			BriefingProcessingRetriedTotal.WithLabelValues(string(job.TriggerType)).Inc()
			logger.Info().
				Int("retry_count", retryCount).
				Time("next_retry_at", nextRetryAt).
				Msg("briefing job scheduled for retry")
		}
		BriefingProcessingFailedTotal.WithLabelValues(string(job.TriggerType), string(failureClass)).Inc()
		return
	}

	// Permanent failure: create a failed briefing version, mark job failed
	if job.PreviousVersionID != nil {
		// Keep previous version active; mark job failed but don't create new version
		if err := w.repo.UpdateJobFailed(ctx, job.ID, reason); err != nil {
			logger.Error().Err(err).Msg("failed to mark briefing job failed")
		}
	} else {
		// No previous version; create a failed version
		_, _, err := w.repo.CreateBriefingVersion(
			ctx,
			job.UpcomingMeetingID,
			job.TriggerType,
			BriefingResultFailed,
			PreparationStatusFailed,
			BriefingContent{},
			0,
			[]uuid.UUID{},
			nil,
		)
		if err != nil {
			logger.Error().Err(err).Msg("failed to create failed briefing version")
		}
		if err := w.repo.UpdateJobFailed(ctx, job.ID, reason); err != nil {
			logger.Error().Err(err).Msg("failed to mark briefing job failed")
		}
	}

	BriefingProcessingFailedTotal.WithLabelValues(string(job.TriggerType), string(failureClass)).Inc()
	BriefingGenerationDurationMs.WithLabelValues(string(result), string(failureClass), "0", string(job.TriggerType)).Observe(float64(time.Since(start).Milliseconds()))
	logger.Error().
		Str("failure_class", string(failureClass)).
		Str("upcoming_meeting_id", job.UpcomingMeetingID.String()).
		Bool("is_retry", failureClass == FailureClassTimeout || failureClass == FailureClassUpstreamError).
		Msg("briefing.generation.failed")
	logger.Error().
		Str("failure_class", string(failureClass)).
		Msg("briefing job permanently failed")
}

func (w *Worker) computeBackoff(retryCount int) time.Duration {
	delay := float64(w.cfg.InitialBackoff)
	for i := 0; i < retryCount; i++ {
		delay *= w.cfg.BackoffMult
	}
	if delay > float64(w.cfg.MaxBackoff) {
		delay = float64(w.cfg.MaxBackoff)
	}
	jitter := delay * w.cfg.JitterFraction * (2*rand.Float64() - 1)
	delay = delay + jitter
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay)
}

func classifyBriefingError(errStr string) FailureClass {
	switch {
	case contains(errStr, "connection refused"), contains(errStr, "timeout"),
		contains(errStr, "503"), contains(errStr, "502"), contains(errStr, "504"),
		contains(errStr, "network"), contains(errStr, "i/o"):
		return FailureClassUpstreamError
	case contains(errStr, "context deadline"), contains(errStr, "context canceled"):
		return FailureClassTimeout
	default:
		return FailureClassInternalError
	}
}

func contains(s, substr string) bool {
	return bytes.Index([]byte(s), []byte(substr)) >= 0
}