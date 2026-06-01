# Performance Eval: brd-03-meeting-memory-processing

> 🔴 Failing — implementation pending

## Scope

Performance benchmarks for Meeting Memory Processing (BRD-03): processing latency targets, queue throughput, retry backoff behavior, import non-blocking guarantee, and concurrent load handling.

---

## Processing Latency

> **NFR Target:** 30 seconds at P95 for 50,000-character transcript-plus-notes input under normal load. Measured from job pick-up to completion, not including queue wait time.

| Metric | Target | Method |
|--------|--------|--------|
| Job duration P95 | < 30,000ms for 50k-char input | `cp_meeting_memory_processing_duration_ms` histogram at p95 |
| Job duration P99 | < 60,000ms for 50k-char input | `cp_meeting_memory_processing_duration_ms` histogram at p99 |
| Job duration at 10k chars | < 10,000ms | Smaller input should complete faster |
| Job duration at 1k chars | < 5,000ms | Minimal input processed quickly |

**Method:** Enqueue 100 processing jobs with 50,000-character combined transcript+notes fixtures. Measure from `memory.job.started` to `memory.job.completed`. Collect `cp_meeting_memory_processing_duration_ms` histogram.

---

## Queue Wait Time

> **Measurement target:** Track queue wait time separately via `cp_meeting_memory_processing_queue_wait_ms` histogram. No fixed SLO committed; target is < 5s under normal load for provisional evaluation.

| Metric | Target | Method |
|--------|--------|--------|
| Queue wait P95 | < 5,000ms (provisional) | `cp_meeting_memory_processing_queue_wait_ms` histogram at p95 |
| Queue depth at steady state | < 200 jobs | Monitor queue depth during sustained import load |
| Queue latency under load | Queue wait degrades gracefully | When worker is slow, queue grows but wait stays bounded |

---

## Import Non-Blocking Guarantee

> **NFR Target:** Successful manual meeting import and redirect must not wait on memory processing completion.

| Metric | Target | Method |
|--------|--------|--------|
| Import redirect time | < 2,000ms from POST to redirect | E2E: POST /meetings → redirect to detail page completes |
| Redirect does not wait for processing | POST /meetings → 201 → redirect | Processing job may still be `queued` when detail page loads |
| Meeting detail loads while queued | GET /meetings/{id}/memory/state | Returns `state='queued'` without waiting for job completion |
| Meeting detail with active memory | GET /meetings/{id}/memory | Returns active memory even when new job is `queued` |

**Method:** Import a meeting with `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`. Measure time from Save click to meeting detail page render. Assert redirect completes within 2s regardless of processing state.

---

## Retry Backoff Behavior

| Scenario | Expected |
|----------|----------|
| First retry after ~1s | Measured from `memory.job.failed` to `memory.job.started` on retry #1: ~1,000ms ±10% |
| Second retry after ~2s | Measured from retry #1 start to retry #2 start: ~2,000ms ±10% |
| Third retry after ~4s | Measured from retry #2 start to retry #3 start: ~4,000ms ±10% |
| Backoff cap at 30s | Any retry delay capped at 30,000ms including jitter |
| Jitter prevents thundering herd | Multiple jobs failing simultaneously: retry times distributed within ±10% band |

**Method:** Inject transient-failure processing for 10 jobs. Measure actual delays between retry attempts. Verify mean delay matches exponential curve within ±10%.

---

## Concurrent Processing

| Metric | Target | Method |
|--------|--------|--------|
| Concurrent job processing | 5+ jobs processing simultaneously | With 5+ queued jobs, observe 5+ in `processing` state |
| Worker unavailability does not block import | Import continues when worker is down | With worker stopped, POST /meetings still succeeds with redirect < 2s |
| Queue depth ceiling | No memory explosion at 1,000 queued jobs | Queue table size remains bounded; old jobs not lost |
| Processing throughput | 10 jobs/minute under normal load | Measure jobs completing per minute at steady state |

---

## Memory API Latency

| Endpoint | Target | Method |
|----------|--------|--------|
| GET /meetings/{id}/memory (cached) | < 100ms | With active memory present, measure round-trip |
| GET /meetings/{id}/memory (uncached) | < 500ms | Cold fetch from DB |
| GET /meetings/{id}/memory/versions | < 200ms | List versions for meeting with 10 versions |
| GET /meetings/{id}/memory/state | < 50ms | State endpoint is lightweight |
| GET /meetings/{id}/memory/conflicts | < 200ms | Pending conflicts query with index |
| POST /meetings/{id}/memory/reprocess | < 500ms to queue job | Job insert is fast; processing async |
| POST conflict resolve | < 500ms to complete | Resolve + version create + emit event |

---

## Load Handling

| Metric | Target | Method |
|--------|--------|--------|
| Memory API under load | 50 concurrent GET /memory requests | p95 latency < 1,000ms |
| Reprocess under load | 10 concurrent POST /reprocess | All queued within 5s |
| Conflict resolution under load | 10 concurrent resolve requests | All resolve within 5s |
| Processing queue under import burst | 100 meetings imported rapidly | Queue depth grows and processes; no jobs lost |
| Database connection pool | No exhaustion at 20 concurrent memory API calls | All requests complete within timeout |

---

## Processing Latency: Provider-Agnostic Measurement

| Fixture | Input | Target | Notes |
|---------|-------|--------|-------|
| Minimal transcript | 100 chars | < 5,000ms | Baseline small input |
| Standard meeting | 20,000 chars transcript + notes | < 20,000ms | Typical BRD-02 meeting |
| Large meeting | 50,000 chars transcript + notes | < 30,000ms | Upper bound per NFR |
| Sparse content | 1 participant, empty transcript + notes | < 5,000ms | Processes quickly despite little content |
| Multiple prior memories | 3 matched prior meetings as input | < 40,000ms | Prior matching adds processing time |

> **Provider portability note:** These targets apply to the overall job duration regardless of which `MemoryProcessor` implementation is injected. Provider-specific SLOs will be defined after runtime selection during Phase 2.

---

## Running

```bash
# Requires services up and processing worker running
make eval-perf

# Manual measurement
# Enqueue job and measure duration
curl -X POST http://localhost:3000/meetings/<id>/memory/reprocess \
  -H "Authorization: Bearer <token>" \
  -d '{}'

# Poll state until completed
curl http://localhost:3000/meetings/<id>/memory/state \
  -H "Authorization: Bearer <token>"

# Check metrics
curl http://localhost:9090/metrics | grep cp_meeting_memory_processing_duration_ms
```
