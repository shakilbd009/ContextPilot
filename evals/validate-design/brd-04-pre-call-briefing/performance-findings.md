# Performance Findings: brd-04-pre-call-briefing

## Overview
Performance validation of BRD-04 Pre-Call Briefing. NFR latency targets, throughput, limit enforcement, and histogram buckets reviewed against brd.md.

---

## 1. Generation Latency NFR (NFR table line 261)

**Spec requirement:** "Async briefing generation completes within P95 < 60s for up to 3 qualifying prior meetings under normal operating conditions."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Latency target, measurement method (job start to terminal), and exclusion (queue wait time) all stated |
| Performance eval | PASS | `cp_briefing_generation_duration_ms` histogram with P50/P95/P99 targets defined (evals/perf lines 23–25) |
| Histogram buckets | ADVISORY | Performance eval defines P50/P95/P99 buckets but BRD does not enumerate explicit histogram bucket boundaries. BRD FR-22 only states "controlled enum labels" without naming buckets. |

---

## 2. Cached View Latency NFR (NFR table line 262)

**Spec requirement:** "Latest completed briefing view renders within P95 < 500ms excluding network variability outside the app."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Target and measurement scope clearly defined |
| Performance eval | PASS | GET /upcoming/{id}/briefing (cached, latest) < 500ms P95 specified (evals/perf line 35) |
| Index coverage | PASS | BRD data model specifies index on `(upcoming_meeting_id, is_active)` WHERE `is_active = TRUE` (brd.md line 200) |

---

## 3. Concurrent Request Handling (Performance eval lines 74–96)

**Spec requirement:** BRD NFR table does not specify concurrent user limits. Performance eval tests 5+ concurrent jobs, 50-job burst, 1000 queue depth.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec gap | MEDIUM | No concurrent user or job limits stated in NFR table. "Normal operating conditions" is undefined for concurrency. |
| Performance eval | PASS | Queue depth, worker capacity, and backlog clearing tested (evals/perf lines 76–96) |

---

## 4. briefing_versions Table Growth and Pruning

**Spec requirement:** "Retention: Briefing versions inherit retention/deletion behavior from the associated upcoming meeting until BRD-06 defines final privacy and retention controls" (NFR table line 271).

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Retention deferred to BRD-06; no pruning logic defined in BRD-04 |
| Performance eval | PASS | Table growth and pruning behavior tested (evals/perf lines 100–103, 196–203) |
| ADR alignment | PASS | ADR-0011 (staleness derived at read time) avoids need for background staleness cleanup worker |

---

## Summary

| Finding | Severity | File:Line | Description |
|---------|----------|-----------|-------------|
| No concurrent user/queue depth limits in NFR | Medium | brd.md NFR table | "Normal operating conditions" undefined; queue depth and concurrent job limits only in performance eval |
| Histogram bucket boundaries not in BRD | Low | brd.md FR-22 | Only "controlled enum labels" stated; bucket edges (e.g., 5000ms, 10000ms, 30000ms, 60000ms) are in eval but not spec |

**Verdict: NEEDS_ATTENTION** — 1 medium finding, 1 low finding.
