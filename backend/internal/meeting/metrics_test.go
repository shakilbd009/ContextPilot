package meeting

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics_Registration(t *testing.T) {
	// Use the default gatherer to check that package-level metrics are registered.
	// promauto registers to the default registry on package import, so we check
	// the default gatherer for the presence of our named metrics.
	gatherer := prometheus.DefaultGatherer

	mfs, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}

	// Build a set of metric names present in the default registry
	names := make(map[string]bool)
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}

	// Check each expected metric is present (promauto registered on import).
	// Only CounterVec and Histogram metrics that are actually registered in the
	// default registry are checked here. FailedTotal and FlagMisconfigurationTotal
	// are validated via their own per-test registries (TestFailedTotal_Labels,
	// TestFlagMisconfigurationTotal_Labels) to avoid promauto double-registration
	// issues.
	expected := []string{
		"cp_manual_meeting_import_started_total",
		"cp_manual_meeting_import_validation_failed_total",
		"cp_manual_meeting_import_completed_total",
		"cp_manual_meeting_import_save_duration_ms",
		"cp_manual_meeting_import_content_size_chars",
		"cp_manual_meeting_import_flag_eval_duration_ms",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected metric %q not found in default gatherer", name)
		}
	}
}

func TestStartedTotal_Labels(t *testing.T) {
	StartedTotal.WithLabelValues("started", "transcript", "/").Inc()
	registry := prometheus.NewRegistry()
	registry.MustRegister(StartedTotal)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	metric := mfs[0].Metric[0]
	if len(metric.Label) != 3 {
		t.Errorf("label count = %d, want 3", len(metric.Label))
	}
}

func TestValidationFailedTotal_Labels(t *testing.T) {
	ValidationFailedTotal.WithLabelValues("validation_failed", "title", "notes").Inc()
	registry := prometheus.NewRegistry()
	registry.MustRegister(ValidationFailedTotal)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	metric := mfs[0].Metric[0]
	if len(metric.Label) != 3 {
		t.Errorf("label count = %d, want 3", len(metric.Label))
	}
}

func TestCompletedTotal_Labels(t *testing.T) {
	CompletedTotal.WithLabelValues("completed", "transcript").Inc()
	registry := prometheus.NewRegistry()
	registry.MustRegister(CompletedTotal)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	metric := mfs[0].Metric[0]
	if len(metric.Label) != 2 {
		t.Errorf("label count = %d, want 2", len(metric.Label))
	}
}

func TestFailedTotal_Labels(t *testing.T) {
	FailedTotal.WithLabelValues("internal_error", "false").Inc()
	registry := prometheus.NewRegistry()
	registry.MustRegister(FailedTotal)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	metric := mfs[0].Metric[0]
	if len(metric.Label) != 2 {
		t.Errorf("label count = %d, want 2", len(metric.Label))
	}
}

func TestSaveDurationMs_Histogram(t *testing.T) {
	SaveDurationMs.Observe(500)
	registry := prometheus.NewRegistry()
	registry.MustRegister(SaveDurationMs)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	if mfs[0].Metric[0].Histogram == nil {
		t.Error("expected histogram type")
	}
}

func TestContentSizeChars_Histogram(t *testing.T) {
	ContentSizeChars.Observe(5000)
	registry := prometheus.NewRegistry()
	registry.MustRegister(ContentSizeChars)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	if mfs[0].Metric[0].Histogram == nil {
		t.Error("expected histogram type")
	}
}

func TestFlagEvalDurationMs_Histogram(t *testing.T) {
	FlagEvalDurationMs.Observe(10)
	registry := prometheus.NewRegistry()
	registry.MustRegister(FlagEvalDurationMs)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	if mfs[0].Metric[0].Histogram == nil {
		t.Error("expected histogram type")
	}
}

func TestFlagMisconfigurationTotal_Labels(t *testing.T) {
	FlagMisconfigurationTotal.WithLabelValues("test_flag").Inc()
	registry := prometheus.NewRegistry()
	registry.MustRegister(FlagMisconfigurationTotal)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather error: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("no metrics gathered")
	}
	metric := mfs[0].Metric[0]
	if len(metric.Label) != 1 {
		t.Errorf("label count = %d, want 1", len(metric.Label))
	}
}