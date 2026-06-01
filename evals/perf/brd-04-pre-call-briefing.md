# Performance Eval: brd-04-pre-call-briefing

> 🔴 Failing — implementation pending

## Scope

Performance benchmarks for Pre-Call Briefing (BRD-04): briefing generation latency, relatedness scoring throughput, concurrent briefing request handling, version table growth, source exclusion/restore latency, and memory footprint under load.

Source of truth: `specs/curated/brd-04-pre-call-briefing.md`

---

## Briefing Generation Latency

> **NFR Target:** Async briefing generation completes within P95 < 60s for up to 3 qualifying prior meetings under normal operating conditions. Measured from job start to terminal success/failure; excludes queue wait time.
>
> **NFR Target:** Cached briefing view renders within P95 < 500ms excluding network variability outside the app.

### Generation Latency — Async Job Duration

| Metric | Target | Method |
|--------|--------|--------|
| Generation job duration P50 | < 20,000ms for up to 3 sources | `cp_briefing_generation_duration_ms` histogram at p50 |
| Generation job duration P95 | < 60,000ms for up to 3 sources | `cp_briefing_generation_duration_ms` histogram at p95 |
| Generation job duration P99 | < 90,000ms for up to 3 sources | `cp_briefing_generation_duration_ms` histogram at p99 |
| Generation with 1 qualifying source | < 30,000ms | Single source reduces scoring load |
| Generation with 0 qualifying sources (shell) | < 10,000ms | No relatedness scoring; shell generation only |

**Method:** Enqueue 100 briefing generation jobs using fixtures with up to 3 qualifying prior meetings. Measure from `briefing.job.started` to `briefing.job.completed` in the processing pipeline. Exclude queue wait time.

### Cached Briefing View Latency

| Metric | Target | Method |
|--------|--------|--------|
| GET /upcoming/{id}/briefing (cached, latest) | < 500ms P95 | Load latest active version from `briefing_versions`; measure round-trip |
| GET /upcoming/{id}/briefing/versions | < 200ms P95 | List versions for upcoming meeting; index on `(upcoming_meeting_id, version_number)` |
| GET /upcoming/{id}/briefing/versions/{n} | < 300ms P95 | Fetch specific version by `(upcoming_meeting_id, version_number)` |

**Method:** With `FF_ENABLE_PRE_CALL_BRIEFING=true`, generate and cache a briefing. Measure repeated GET requests to the latest active version. Cold DB fetch (uncached) should still meet P95 < 500ms target.

---

## Relatedness Scoring Throughput

> **NFR alignment:** Signal matching (2-signal threshold) and relatedness scoring run during async generation. Throughput must scale with signal count without degrading generation latency below NFR targets.

### Signal Count vs Latency

| Fixture | Input | Target | Notes |
|--------|-------|--------|-------|
| 3 qualifying sources, 4 signals each | 12 total signals | < 60,000ms total job duration | Full load: 3 sources with all 4 signals |
| 3 qualifying sources, 2 signals each | 6 total signals | < 50,000ms | Minimum qualifying load |
| 1 qualifying source, 4 signals | 4 signals | < 35,000ms | Single source, max signals |
| 1 qualifying source, 2 signals | 2 signals | < 30,000ms | Single source, minimum signals |
| 0 qualifying sources (shell) | 0 signals | < 10,000ms | No matching; shell generation only |
| 3 sources + 1 excluded (restore) | 2 active signals | < 40,000ms | After restore, one fewer source |

**Method:** Enqueue briefing jobs with fixtures varying signal counts. Measure total job duration. Score correlation between signal count and latency to confirm linear scaling.

### Relatedness Scoring Order Determinism

| Scenario | Expected |
|----------|----------|
| Source A has 4 signals, Source B has 2 signals | A ranked first |
| Both have 3 signals, different completion dates | More recent first |
| Both have 3 signals, same completion date | Stronger title similarity wins |
| Signal count tie + date tie + title tie | Stable meeting ID as tie-breaker; deterministic across runs |
| Opaque internal scores | Not exposed in API; ordering explained by signal count + date + title similarity |

**Method:** Generate briefings for fixtures with deliberate tie-breaking scenarios. Verify ordering is stable across 5 repeated runs.

---

## Concurrent Briefing Request Handling

### Queue and Worker Capacity

| Metric | Target | Method |
|--------|--------|--------|
| Concurrent briefing jobs processing | 5+ jobs simultaneously | With 5+ queued jobs, observe 5+ in `generating` state |
| Queue depth under burst | No unbounded growth at 100 concurrent requests | Monitor `briefing_jobs` queue depth |
| Connection pool under load | No exhaustion at 20 concurrent briefing API calls | All requests complete within timeout |
| Briefing request backlog clears | 50 queued jobs process to completion within 10 minutes | Sustained throughput check |

**Method:** Enqueue 50 concurrent briefing generation requests. Measure time for all to reach `ready`/`failed` terminal state. Verify no requests timeout or deadlock.

### Queue Backlog Behavior

| Scenario | Expected |
|----------|----------|
| Worker is slow/down | Queue depth grows; requests wait but do not fail silently |
| Worker recovers | Queue drains; backlog clears within 5 minutes |
| Queue at 1000 jobs | New requests still enqueued; oldest jobs processed first |
| Queue saturation | System emits warning metric `cp_briefing_queue_saturation` |

**Method:** With queue at 1000 depth, enqueue 10 new jobs. Verify they are visible in queue state and process to completion.

---

## briefing_versions Table Growth

### Version Pruning Strategy

> **NFR alignment:** Retention is deferred to BRD-06. For MVP, version growth must not cause unbounded table size without a defined pruning strategy.

| Metric | Target | Method |
|--------|--------|--------|
| Versions per upcoming meeting | Distinct version per generation/regeneration | Each `POST /briefing/regenerate` creates new version with monotonically increasing `version_number` |
| Active version constraint | Exactly one `is_active=true` per upcoming meeting | On new version: `UPDATE ... SET is_active=false WHERE upcoming_meeting_id=X; INSERT new version with is_active=true` |
| Superseded versions remain | `status='superseded'` versions accessible via GET /versions | Never deleted without BRD-06 retention policy |
| Index size for 100 versions/meeting | `(upcoming_meeting_id, is_active)` index stays efficient | Verify index scan on latest-version lookup remains < 10ms |

**Method:** Generate 10 regeneration requests for a single upcoming meeting. Verify 10 versions exist, exactly 1 is `is_active=true`, all have distinct `version_number`. Check index scan plan for latest-version query.

### Version Growth Under Load

| Metric | Target | Method |
|--------|--------|--------|
| 1,000 upcoming meetings × 5 versions each | 5,000 rows in `briefing_versions` | Generate 5 versions per meeting; measure table size |
| Latest version lookup with 5 versions | < 50ms | Index on `(upcoming_meeting_id, is_active)` WHERE `is_active=TRUE` |
| Version list query with 50 versions | < 200ms | `SELECT * FROM briefing_versions WHERE upcoming_meeting_id=? ORDER BY version_number` |

---

## Source Exclusion and Restore Latency Impact

### Exclusion Operation Latency

| Metric | Target | Method |
|--------|--------|--------|
| POST /briefing/sources/{id}/exclude | < 500ms to persist | Insert into `briefing_source_exclusions`; verify persisted |
| Exclude + regenerate sequence | Regeneration uses remaining sources only | Exclude one source; trigger regenerate; verify excluded source not in new version |
| Undo exclude before regenerate | < 500ms | Delete from `briefing_source_exclusions`; source reappears as candidate |

**Method:** With an active briefing using 3 sources, exclude one via POST. Verify it does not reappear in next regeneration. Undo the exclusion and verify it reappears as candidate.

### Exclusion State Persistence

| Scenario | Expected |
|----------|----------|
| Exclude + regenerate + exclude again | Same source can be re-excluded per upcoming meeting |
| Exclude A, regenerate, exclude B | Both A and B are excluded; new version uses remaining |
| Exclude all qualifying sources | Generation produces no-prior-memory shell |
| Exclude during active generation | Current generation completes with pre-exclusion source set; next regeneration uses exclusions |
| Cross-upcoming exclusion | Exclusion for meeting M1 does not affect meeting M2 |

**Method:** Exclude all 3 sources for an upcoming meeting. Trigger regeneration. Verify shell is generated. Restore one source, regenerate. Verify shell no longer appears.

---

## Memory Footprint Under Load

### Relatedness Scoring Memory Usage

| Metric | Target | Method |
|--------|--------|--------|
| Memory per briefing job (1 source) | < 50MB RSS | Baseline: single source scoring |
| Memory per briefing job (3 sources) | < 150MB RSS | All 3 sources loaded for signal matching |
| Memory growth with signal count | Sublinear; scoring is CPU-bound | Confirm RSS does not double when signals increase 4x |
| Peak memory during generation | < 200MB per worker | Monitor RSS during max-signal generation |

**Method:** With memory profiling enabled, generate briefings with 1 and 3 sources. Measure RSS at job start, during scoring, and at completion. Verify no unbounded growth.

### Concurrent Load Memory Behavior

| Metric | Target | Method |
|--------|--------|--------|
| 10 concurrent briefing jobs | < 500MB total system memory | All 10 jobs in `generating` state simultaneously |
| Worker memory recovery | Memory returned after job completion | RSS drops back to baseline within 30s of job completion |
| Memory under queue buildup | Memory stays bounded as queue grows | With 100 queued jobs, system memory does not exceed 1GB |

---

## Running

```bash
# Requires services up and briefing worker running
make eval-perf

# Manual measurement
# Enqueue briefing generation and measure duration
curl -X POST http://localhost:3000/upcoming/<meetingId>/briefing/regenerate \
  -H "Authorization: Bearer ***" \
  -d '{}'

# Poll state until completed
curl http://localhost:3000/upcoming/<meetingId>/briefing \
  -H "Authorization: Bearer ***"

# Check metrics
curl http://localhost:9090/metrics | grep cp_briefing_generation_duration_ms
```

---

## NFR Summary

| Requirement | Target | Source |
|------------|--------|--------|
| Generation latency | P95 < 60s for up to 3 qualifying sources | BRD-04 NFR |
| Cached view latency | P95 < 500ms | BRD-04 NFR |
| Availability | Graceful degradation; cached briefing available during failures | BRD-04 NFR |
| Regeneration continuity | Latest completed briefing readable during regeneration | BRD-04 NFR |
| Privacy | No raw content, PII, snippets in logs/metrics | BRD-04 NFR |
| Authorization | Only authorized meetings used in generation | BRD-04 NFR |