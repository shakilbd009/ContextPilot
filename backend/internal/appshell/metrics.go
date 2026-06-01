package appshell

import (
	"sync"
	"sync/atomic"
)

// Metrics holds app-shell metrics using atomics (prometheus client compatible labels).
// In production this would be a prometheus.Counter and prometheus.Histogram.
// This implementation uses atomics to avoid pulling in the prometheus library as a
// hard dependency for Phase 1.
type Metrics struct {
	errorTotal       int64
	flagEvalDuration int64 // microseconds, aggregated via rolling sum+count
	navTotal         int64

	mu            sync.RWMutex
	flagEvalSum   int64
	flagEvalCount int64
}

// RecordError increments the error counter with the given labels.
// Labels are accepted for interface compatibility; in the atomic impl they are
// logged as a no-op (prometheus would record them).
func (m *Metrics) RecordError(component, route string) {
	atomic.AddInt64(&m.errorTotal, 1)
	// In production: errorCounter.WithLabelValues(component, route).Inc()
}

// RecordFlagEval records a flag evaluation with the given labels and duration
// in microseconds.
func (m *Metrics) RecordFlagEval(flag, result string, us int64) {
	m.mu.Lock()
	atomic.AddInt64(&m.flagEvalSum, us)
	atomic.AddInt64(&m.flagEvalCount, 1)
	atomic.AddInt64(&m.flagEvalDuration, us)
	m.mu.Unlock()
	// In production: flagEvalDurationHistogram.WithLabelValues(flag, result).Observe(float64(us))
}

// RecordNav increments the navigation counter for the given route.
func (m *Metrics) RecordNav(route string) {
	atomic.AddInt64(&m.navTotal, 1)
	// In production: navCounter.WithLabelValues(route).Inc()
}

// NavTotal returns the current navigation count.
func (m *Metrics) NavTotal() int64 {
	return atomic.LoadInt64(&m.navTotal)
}

// ErrorTotal returns the current error count.
func (m *Metrics) ErrorTotal() int64 {
	return atomic.LoadInt64(&m.errorTotal)
}

// FlagEvalMicroseconds returns the most recent flag eval duration in microseconds
// and the running total count.
func (m *Metrics) FlagEvalMicroseconds() (int64, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return atomic.LoadInt64(&m.flagEvalDuration), atomic.LoadInt64(&m.flagEvalCount)
}

// GlobalMetrics is the process-wide app-shell metrics instance.
var GlobalMetrics = &Metrics{}