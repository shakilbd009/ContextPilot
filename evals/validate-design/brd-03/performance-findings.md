# Performance Findings — BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validator:** performance
**Target:** specs/domain/brd-03-meeting-memory-processing.md
**ADRs reviewed:** ADR-0005 (queue), ADR-0006 (data model), ADR-0008 (observability)

---

## Verdict: PASS

BRD-03 specifies a well-designed asynchronous processing architecture. The non-blocking import path, PostgreSQL advisory lock-based queue, and versioned memory model are all architecturally sound for the performance requirements. No Must Fix or High severity issues identified.

---

## No Critical Issues

---

## No High Priority Issues

---

## Medium Priority Issues

### [M1] `cp_meeting_memory_processing_duration_ms` histogram has no bucket definitions

**Source:** BRD-03 Observability, Metrics section
**BRD section:** Observability / Metrics to emit

The `cp_meeting_memory_processing_duration_ms` histogram metric is defined but no bucket boundaries (Prometheus `bucket` labels) are specified. Without bucket definitions, the p50/p95/p99 percentiles cannot be computed or compared against the processing latency SLO (OQ-2: deferred to curation).

**Fix required:** Add histogram bucket definitions to the observability section once the processing latency SLO is committed. Suggested initial buckets (to be refined once SLO is known): 1s, 5s, 10s, 30s, 60s, 120s, 300s.

---

### [M2] Queue wait time histogram defined but no target specified

**Source:** BRD-03 Observability / Metrics
**BRD section:** `cp_meeting_memory_processing_queue_wait_ms`

The `cp_meeting_memory_processing_queue_wait_ms` histogram tracks time from job creation to processing start. This is a valuable latency metric, but no target or SLO is defined. Queue wait time is a function of worker capacity and job throughput.

**Fix required:** Once the processing latency SLO is set (OQ-2), derive a queue wait time SLO from the overall end-to-end processing latency budget.

---

### [M3] PostgreSQL advisory lock contention risk not evaluated at scale

**Source:** ADR-0005 Trade-offs section
**ADR:** ADR-0005

ADR-0005 acknowledges "Multiple Go processes compete correctly but the DB is the bottleneck at scale." No evaluation has been done of how many concurrent workers can run before PostgreSQL advisory lock contention becomes a bottleneck. At 1000+ meetings per day with processing times of 10-30 seconds, the queue could grow faster than workers can drain it.

**Fix required:** Add a note to ADR-0005 about expected queue depth at target load, and specify when the team should re-evaluate migrating to a dedicated queue (BullMQ/Redis). Alternatively, document the decision to accept PostgreSQL advisory locks at all scales as the Phase 2 approach.

---

## Low Priority Issues

### [L1] No limit on memory_versions rows per meeting over time

**Source:** ADR-0006 Data Model
**ADR:** ADR-0006

Each processing/reprocessing run creates a new `memory_versions` row. With frequent source edits, stale reprocessing, and conflict resolutions, a meeting could accumulate many versions. No retention policy is specified (deferred to BRD-06). This is acceptable given BRD-06 is in the backlog, but the storage growth is unbounded until then.

**Note:** This is a known deferred issue (BRD-06) and not a blocking concern for BRD-03 validation.

---

## Positive Findings

- Import path is correctly non-blocking: `POST /meetings` returns after transaction commit; memory processing INSERT is fire-and-forget (AC-2)
- Advisory lock-based queue correctly prevents double-processing: `FOR UPDATE SKIP LOCKED` is the right primitive for concurrent worker acquisition
- Exponential backoff (2^retry_count seconds) is a sound retry strategy for transient provider failures
- Queue wait time and processing duration are tracked separately — enables distinguishing slow queue vs. slow processing
- `meeting_memory_processing_queue_wait_ms` histogram correctly measures time-from-queue to processing-start (not just total processing time)
- Retry scheduler loop (moves `retrying` jobs back to `queued` when next_retry_at <= now()) is correctly described
- Graceful degradation: existing active memory remains available when processing worker/provider is unavailable (FR-26)
- Processing is explicitly bounded by BRD-02's 50,000 character transcript-plus-notes limit (NFR table)
- Worker count and job polling interval are not hardcoded in the ADR — deferred to implementation configuration

---

## Summary Table

| ID | Severity | Issue |
|----|----------|-------|
| M1 | Medium | Processing duration histogram has no bucket definitions (SLO deferred) |
| M2 | Medium | Queue wait time histogram has no target SLO specified |
| M3 | Medium | Advisory lock contention risk not evaluated at scale |
| L1 | Low | No limit on memory_versions rows per meeting (deferred to BRD-06) |