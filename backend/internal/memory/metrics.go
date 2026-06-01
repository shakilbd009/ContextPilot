package memory

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ProcessingQueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_queued_total",
			Help: "Incremented when a memory processing job row is inserted.",
		},
		[]string{"trigger_type"},
	)

	ProcessingStartedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_started_total",
			Help: "Incremented when worker picks up a job.",
		},
		[]string{"trigger_type"},
	)

	ProcessingCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_completed_total",
			Help: "Incremented on job completion.",
		},
		[]string{"trigger_type", "status"},
	)

	ProcessingFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_failed_total",
			Help: "Incremented on permanent failure.",
		},
		[]string{"trigger_type", "failure_class"},
	)

	ProcessingRetriedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_retried_total",
			Help: "Incremented on automatic retry.",
		},
		[]string{"trigger_type"},
	)

	ProcessingRetryExhaustedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_processing_retry_exhausted_total",
			Help: "Incremented when max retries reached.",
		},
		[]string{"trigger_type"},
	)

	ProcessingDurationMs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_meeting_memory_processing_duration_ms",
			Help:    "Processing job duration in milliseconds (job pick-up to completion).",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 30000, 60000, 120000},
		},
	)

	ProcessingQueueWaitMs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_meeting_memory_processing_queue_wait_ms",
			Help:    "Time from job creation to worker pick-up in milliseconds.",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 30000, 60000},
		},
	)

	ActiveVersionChangedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_active_version_changed_total",
			Help: "Incremented when active version pointer changes.",
		},
		[]string{"reason"},
	)

	ConflictsDetectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_conflicts_detected_total",
			Help: "Incremented when a conflict is placed in the review queue.",
		},
		[]string{"meeting_id"},
	)

	FlagEvaluationMs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_meeting_memory_flag_evaluation_ms",
			Help:    "Feature flag evaluation latency in milliseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100},
		},
	)

	FlagMisconfigurationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_meeting_memory_flag_misconfiguration_total",
			Help: "Incremented when server flag is false and browser flag is true.",
		},
		[]string{"flag"},
	)
)