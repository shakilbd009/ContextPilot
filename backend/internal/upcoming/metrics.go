package upcoming

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CreateRequestedTotal counts incoming create requests by source and participant presence.
	CreateRequestedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_create_requested_total",
			Help: "Total incoming POST /upcoming requests.",
		},
		[]string{"source", "has_participants"},
	)

	// CreateSucceededTotal counts successful creations by participant count bucket and client/org presence.
	CreateSucceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_created_total",
			Help: "Total successfully created upcoming meetings.",
		},
		[]string{"participant_count_bucket", "has_client_or_org"},
	)

	// CreateFailedTotal counts failed creation attempts by failure class.
	CreateFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_create_failed_total",
			Help: "Total failed upcoming meeting creation attempts.",
		},
		[]string{"failure_class"},
	)

	// ValidationFailedTotal counts validation failures by validation class.
	ValidationFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_validation_failed_total",
			Help: "Total validation failures by class.",
		},
		[]string{"validation_class"},
	)

	// UpdatedTotal counts successful updates by changed field class.
	UpdatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_updated_total",
			Help: "Total successful upcoming meeting updates.",
		},
		[]string{"changed_field_class"},
	)

	// UpdateFailedTotal counts failed update attempts by failure class.
	UpdateFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_update_failed_total",
			Help: "Total failed upcoming meeting update attempts.",
		},
		[]string{"failure_class"},
	)

	// CancelledTotal counts successful cancellations by prior briefing status.
	CancelledTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_cancelled_total",
			Help: "Total successful upcoming meeting cancellations.",
		},
		[]string{"had_briefing_status"},
	)

	// CancelFailedTotal counts failed cancellation attempts by failure class.
	CancelFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_cancel_failed_total",
			Help: "Total failed cancellation attempts.",
		},
		[]string{"failure_class"},
	)

	// ListLoadedTotal counts list loads by view and result count bucket.
	ListLoadedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_list_loaded_total",
			Help: "Total upcoming meeting list/calendar/dashboard loads.",
		},
		[]string{"view", "result_count_bucket"},
	)

	// ViewedTotal counts detail views by status and briefing status.
	ViewedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_viewed_total",
			Help: "Total upcoming meeting detail view loads.",
		},
		[]string{"status", "briefing_status"},
	)

	// ActionDurationMs histograms for each CRUD action.
	ActionDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cp_upcoming_meeting_action_duration_ms",
			Help:    "Duration of upcoming meeting actions in milliseconds.",
			Buckets: []float64{25, 50, 100, 250, 500, 1000},
		},
		[]string{"action", "result"},
	)

	// BriefingTriggerQueuedTotal counts queued briefing triggers by trigger type.
	BriefingTriggerQueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_briefing_trigger_queued_total",
			Help: "Total queued BRD-04 briefing triggers.",
		},
		[]string{"trigger"},
	)

	// BriefingTriggerSkippedTotal counts skipped triggers by reason.
	BriefingTriggerSkippedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_briefing_trigger_skipped_total",
			Help: "Total skipped briefing triggers.",
		},
		[]string{"reason"},
	)

	// BriefingTriggerFailedTotal counts failed trigger enqueues by class.
	BriefingTriggerFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_briefing_trigger_failed_total",
			Help: "Total failed briefing trigger enqueue attempts.",
		},
		[]string{"failure_class"},
	)

	// BriefingTriggerEnqueueDurationMs histograms for trigger enqueue latency.
	BriefingTriggerEnqueueDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms",
			Help:    "Duration of briefing trigger enqueue in milliseconds.",
			Buckets: []float64{10, 25, 50, 100, 250, 500},
		},
		[]string{"trigger", "result"},
	)

	// FlagMismatchTotal counts flag mismatch detections: frontend enabled a feature
	// but the backend rejected it because the backend flag is disabled. This is a
	// safe, correct response — it indicates frontend/backend flag desync.
	// direction: frontend-disabled means frontend FF is on but backend FF is off.
	// result: feature_disabled (safe response; NOT a 500).
	FlagMismatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_upcoming_meeting_flag_mismatch_total",
			Help: "Total flag mismatch detections. Emitted when a request arrives for a feature disabled on the backend while the frontend is attempting to use it.",
		},
		[]string{"flag_name", "direction", "result"},
	)
)

// ParticipantCountBucket maps a participant count to its bucket label.
func ParticipantCountBucket(n int) string {
	switch {
	case n == 0:
		return "0"
	case n == 1:
		return "1"
	case n >= 2 && n <= 5:
		return "2_5"
	case n >= 6 && n <= 10:
		return "6_10"
	default:
		return "11_50"
	}
}

// ResultCountBucket maps a meeting list result count to its bucket label.
func ResultCountBucket(n int) string {
	switch {
	case n == 0:
		return "0"
	case n == 1:
		return "1"
	case n >= 2 && n <= 5:
		return "2_5"
	case n >= 6 && n <= 20:
		return "6_20"
	default:
		return "21_plus"
	}
}