package meeting

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics

	StartedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_manual_meeting_import_started_total",
			Help: "Incremented when a save attempt passes initial client intent.",
		},
		[]string{"result", "content_source", "route"},
	)

	ValidationFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_manual_meeting_import_validation_failed_total",
			Help: "Incremented per validation failure.",
		},
		[]string{"result", "field", "content_source"},
	)

	CompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_manual_meeting_import_completed_total",
			Help: "Incremented on successful meeting record creation.",
		},
		[]string{"result", "content_source"},
	)

	FailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_manual_meeting_import_failed_total",
			Help: "Incremented on server-side failures before a meeting is saved.",
		},
		[]string{"result", "flag"},
	)

	SaveDurationMs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_manual_meeting_import_save_duration_ms",
			Help:    "Save operation latency in milliseconds.",
			Buckets: []float64{50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
	)

	ContentSizeChars = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_manual_meeting_import_content_size_chars",
			Help:    "Combined transcript + notes character count.",
			Buckets: []float64{100, 1000, 5000, 10000, 25000, 40000, 50000, 60000, 100000},
		},
	)

	FlagEvalDurationMs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cp_manual_meeting_import_flag_eval_duration_ms",
			Help:    "Feature flag evaluation latency in microseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100},
		},
	)

	FlagMisconfigurationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_manual_meeting_import_flag_misconfiguration_total",
			Help: "Incremented when server flag is false and browser flag is true.",
		},
		[]string{"flag"},
	)
)