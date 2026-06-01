package memory

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_ProcessingQueuedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_meeting_memory_processing_queued_total",
			Help: "Test counter",
		},
		[]string{"trigger_type"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues(string(TriggerTypeImport)).Inc()
	c.WithLabelValues(string(TriggerTypeReprocess)).Inc()
	c.WithLabelValues(string(TriggerTypeImport)).Inc()

	importCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeImport)))
	reprocessCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeReprocess)))

	if importCount != 2 {
		t.Errorf("import count = %v, want 2", importCount)
	}
	if reprocessCount != 1 {
		t.Errorf("reprocess count = %v, want 1", reprocessCount)
	}
}

func TestMetrics_ProcessingCompletedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_completed_total",
			Help: "Test counter",
		},
		[]string{"trigger_type", "status"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues(string(TriggerTypeImport), "completed").Inc()
	c.WithLabelValues(string(TriggerTypeManualRetry), "completed").Inc()
	c.WithLabelValues(string(TriggerTypeImport), "failed").Inc()

	completedCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeImport), "completed"))
	failedCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeImport), "failed"))
	manualRetryCompleted := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeManualRetry), "completed"))

	if completedCount != 1 {
		t.Errorf("completed count = %v, want 1", completedCount)
	}
	if failedCount != 1 {
		t.Errorf("failed count = %v, want 1", failedCount)
	}
	if manualRetryCompleted != 1 {
		t.Errorf("manual retry completed = %v, want 1", manualRetryCompleted)
	}
}

func TestMetrics_ProcessingFailedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_failed_total",
			Help: "Test counter",
		},
		[]string{"trigger_type", "failure_class"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues(string(TriggerTypeImport), string(FailureClassTransientProvider)).Inc()
	c.WithLabelValues(string(TriggerTypeImport), string(FailureClassTransientProvider)).Inc()
	c.WithLabelValues(string(TriggerTypeReprocess), string(FailureClassPermanentProcessing)).Inc()

	transientCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeImport), string(FailureClassTransientProvider)))
	permanentCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeReprocess), string(FailureClassPermanentProcessing)))

	if transientCount != 2 {
		t.Errorf("transient count = %v, want 2", transientCount)
	}
	if permanentCount != 1 {
		t.Errorf("permanent count = %v, want 1", permanentCount)
	}
}

func TestMetrics_ProcessingRetriedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_retried_total",
			Help: "Test counter",
		},
		[]string{"trigger_type"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues(string(TriggerTypeImport)).Inc()
	c.WithLabelValues(string(TriggerTypeStaleReprocess)).Inc()

	importCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeImport)))
	staleCount := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeStaleReprocess)))

	if importCount != 1 {
		t.Errorf("import retry count = %v, want 1", importCount)
	}
	if staleCount != 1 {
		t.Errorf("stale reprocess retry count = %v, want 1", staleCount)
	}
}

func TestMetrics_ProcessingRetryExhaustedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_retry_exhausted_total",
			Help: "Test counter",
		},
		[]string{"trigger_type"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues(string(TriggerTypeManualRetry)).Inc()

	count := testutil.ToFloat64(c.WithLabelValues(string(TriggerTypeManualRetry)))
	if count != 1 {
		t.Errorf("retry exhausted count = %v, want 1", count)
	}
}

func TestMetrics_ProcessingDurationMs(t *testing.T) {
	h := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "test_cp_duration_ms",
			Help:    "Test histogram",
			Buckets: []float64{100, 500, 1000, 5000},
		},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(h)

	h.Observe(50)
	h.Observe(150)
	h.Observe(600)
	h.Observe(3000)
}

func TestMetrics_ActiveVersionChangedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_active_version_changed_total",
			Help: "Test counter",
		},
		[]string{"reason"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues("reprocess").Inc()
	c.WithLabelValues("conflict_resolution").Inc()
	c.WithLabelValues("reprocess").Inc()

	reprocessCount := testutil.ToFloat64(c.WithLabelValues("reprocess"))
	conflictCount := testutil.ToFloat64(c.WithLabelValues("conflict_resolution"))

	if reprocessCount != 2 {
		t.Errorf("reprocess count = %v, want 2", reprocessCount)
	}
	if conflictCount != 1 {
		t.Errorf("conflict_resolution count = %v, want 1", conflictCount)
	}
}

func TestMetrics_ConflictsDetectedTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_conflicts_detected_total",
			Help: "Test counter",
		},
		[]string{"meeting_id"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues("meeting-1").Inc()
	c.WithLabelValues("meeting-2").Inc()
	c.WithLabelValues("meeting-1").Inc()

	m1Count := testutil.ToFloat64(c.WithLabelValues("meeting-1"))
	m2Count := testutil.ToFloat64(c.WithLabelValues("meeting-2"))

	if m1Count != 2 {
		t.Errorf("meeting-1 count = %v, want 2", m1Count)
	}
	if m2Count != 1 {
		t.Errorf("meeting-2 count = %v, want 1", m2Count)
	}
}

func TestMetrics_FlagEvaluationMs(t *testing.T) {
	h := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "test_cp_flag_evaluation_ms",
			Help:    "Test histogram",
			Buckets: []float64{1, 5, 10, 25},
		},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(h)

	h.Observe(2)
	h.Observe(8)
}

func TestMetrics_FlagMisconfigurationTotal(t *testing.T) {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_cp_flag_misconfiguration_total",
			Help: "Test counter",
		},
		[]string{"flag"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	c.WithLabelValues("FF_ENABLE_MEETING_MEMORY_PROCESSING").Inc()

	count := testutil.ToFloat64(c.WithLabelValues("FF_ENABLE_MEETING_MEMORY_PROCESSING"))
	if count != 1 {
		t.Errorf("flag misconfiguration count = %v, want 1", count)
	}
}

func TestMetrics_RegisteredMetricsNotNil(t *testing.T) {
	if ProcessingQueuedTotal == nil {
		t.Error("ProcessingQueuedTotal is nil")
	}
	if ProcessingStartedTotal == nil {
		t.Error("ProcessingStartedTotal is nil")
	}
	if ProcessingCompletedTotal == nil {
		t.Error("ProcessingCompletedTotal is nil")
	}
	if ProcessingFailedTotal == nil {
		t.Error("ProcessingFailedTotal is nil")
	}
	if ProcessingRetriedTotal == nil {
		t.Error("ProcessingRetriedTotal is nil")
	}
	if ProcessingRetryExhaustedTotal == nil {
		t.Error("ProcessingRetryExhaustedTotal is nil")
	}
	if ProcessingDurationMs == nil {
		t.Error("ProcessingDurationMs is nil")
	}
	if ProcessingQueueWaitMs == nil {
		t.Error("ProcessingQueueWaitMs is nil")
	}
	if ActiveVersionChangedTotal == nil {
		t.Error("ActiveVersionChangedTotal is nil")
	}
	if ConflictsDetectedTotal == nil {
		t.Error("ConflictsDetectedTotal is nil")
	}
	if FlagEvaluationMs == nil {
		t.Error("FlagEvaluationMs is nil")
	}
	if FlagMisconfigurationTotal == nil {
		t.Error("FlagMisconfigurationTotal is nil")
	}
}

func TestMetrics_IncrementOperations(t *testing.T) {
	ProcessingQueuedTotal.WithLabelValues(string(TriggerTypeImport)).Inc()
	ProcessingStartedTotal.WithLabelValues(string(TriggerTypeReprocess)).Inc()
	ProcessingCompletedTotal.WithLabelValues(string(TriggerTypeImport), "completed").Inc()
	ProcessingFailedTotal.WithLabelValues(string(TriggerTypeImport), string(FailureClassTransientProvider)).Inc()
	ProcessingRetriedTotal.WithLabelValues(string(TriggerTypeManualRetry)).Inc()
	ProcessingRetryExhaustedTotal.WithLabelValues(string(TriggerTypeStaleReprocess)).Inc()
	ProcessingDurationMs.Observe(500)
	ProcessingQueueWaitMs.Observe(200)
	ActiveVersionChangedTotal.WithLabelValues("reprocess").Inc()
	ConflictsDetectedTotal.WithLabelValues("meeting-abc").Inc()
	FlagEvaluationMs.Observe(3)
	FlagMisconfigurationTotal.WithLabelValues("FF_ENABLE_MEETING_MEMORY_PROCESSING").Inc()
}