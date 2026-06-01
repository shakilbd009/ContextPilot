package briefing

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BriefingGenerationDurationMs tracks end-to-end briefing generation time.
	BriefingGenerationDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "cp",
			Name:      "briefing_generation_duration_ms",
			Help:      "End-to-end briefing generation time from job start to terminal state",
			Buckets:   []float64{1000, 5000, 10000, 30000, 60000},
		},
		[]string{"status", "failure_class", "source_count_bucket", "trigger_type"},
	)

	// BriefingQueueSaturation tracks current queue depth as percentage of normal-operating capacity.
	BriefingQueueSaturation = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "cp",
			Name:      "briefing_queue_saturation",
			Help:      "Current briefing queue depth as percentage of normal-operating capacity",
		},
		[]string{"queue_name"},
	)

	// BriefingProcessingQueuedTotal counts queued briefing jobs.
	BriefingProcessingQueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_processing_queued_total",
			Help:      "Total number of queued briefing processing jobs",
		},
		[]string{"trigger_type"},
	)

	// BriefingProcessingCompletedTotal counts completed briefing jobs.
	BriefingProcessingCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_processing_completed_total",
			Help:      "Total number of completed briefing processing jobs",
		},
		[]string{"status", "failure_class"},
	)

	// BriefingProcessingFailedTotal counts failed briefing jobs.
	BriefingProcessingFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_processing_failed_total",
			Help:      "Total number of failed briefing processing jobs",
		},
		[]string{"trigger_type", "failure_class"},
	)

	// BriefingProcessingRetriedTotal counts retried briefing jobs.
	BriefingProcessingRetriedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_processing_retried_total",
			Help:      "Total number of retried briefing processing jobs",
		},
		[]string{"trigger_type"},
	)

	// BriefingProcessingRetryExhaustedTotal counts jobs that exhausted retries.
	BriefingProcessingRetryExhaustedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_processing_retry_exhausted_total",
			Help:      "Total number of briefing jobs that exhausted retries",
		},
		[]string{"trigger_type"},
	)

	// FlagMisconfigurationTotal counts flag mismatch detections.
	FlagMisconfigurationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "pre_call_briefing_flag_misconfiguration_total",
			Help:      "Total flag misconfiguration detections for pre-call briefing",
		},
		[]string{"server_flag_value", "browser_flag_value", "endpoint"},
	)

	// BriefingViewedTotal counts briefing view events.
	// Uses a plain Counter (no labels) — adding an upcoming_meeting_id label would
	// cause unbounded cardinality growth and violate the BRD-04 privacy NFR.
	BriefingViewedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_viewed_total",
			Help:      "Total number of briefing view events",
		},
	)

	// BriefingSourceExcludedTotal counts source exclusion events.
	// Uses a plain Counter (no labels) — adding an upcoming_meeting_id label would
	// cause unbounded cardinality growth and violate the BRD-04 privacy NFR.
	BriefingSourceExcludedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_source_excluded_total",
			Help:      "Total number of source exclusion events",
		},
	)

	// BriefingSourceRestoredTotal counts source restoration events.
	// Uses a plain Counter (no labels) — adding an upcoming_meeting_id label would
	// cause unbounded cardinality growth and violate the BRD-04 privacy NFR.
	BriefingSourceRestoredTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "cp",
			Name:      "briefing_source_restored_total",
			Help:      "Total number of source restoration events",
		},
	)
)