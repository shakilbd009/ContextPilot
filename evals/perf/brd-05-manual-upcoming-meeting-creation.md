# Performance Eval: brd-05-manual-upcoming-meeting-creation

> 🔴 Failing — implementation pending

## Scope

Performance benchmarks for Manual Upcoming Meeting Creation (BRD-05): create/update/cancel latency, list/detail/calendar view latency, briefing trigger enqueue latency, and scale targets.

Source of truth: `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md`

---

## Create / Update / Cancel Latency

> **NFR Target:** p95 < 500ms for BRD-05 server action completion, excluding async BRD-04 generation work.
>
> **NFR Target:** List/detail/calendar view latency p95 < 500ms, excluding client/network variability.

### Create Latency

| Metric | Target | Method |
|--------|--------|--------|
| POST /upcoming (create) — cold DB | < 500ms P95 | Measure end-to-end round-trip with no cached state; exclude briefing trigger |
| POST /upcoming with participants (0) | < 300ms P95 | Create with no participants; baseline |
| POST /upcoming with participants (1–5) | < 400ms P95 | Create with up to 5 participants |
| POST /upcoming with participants (6–10) | < 500ms P95 | Create with up to 10 participants |
| POST /upcoming with 50 participants | < 500ms P95 | At participant hard limit; insert all within target |

**Method:** Authenticate as a test user. Issue POST /upcoming with valid title + scheduled_start. Vary participant count. Measure round-trip to response body receipt. Exclude async briefing trigger work (triggered after response sent). Run 100 iterations per fixture; report p50, p95, p99.

### Update Latency

| Metric | Target | Method |
|--------|--------|--------|
| PATCH /upcoming/{id} (metadata only) | < 500ms P95 | Update title, description, client/org only |
| PATCH /upcoming/{id} (schedule change) | < 500ms P95 | Update scheduled_start; re-anchors edit window |
| PATCH /upcoming/{id} (add participant) | < 500ms P95 | Add 1 participant to existing meeting |
| PATCH /upcoming/{id} (add 10 participants) | < 500ms P95 | Add 10 participants at once |
| PATCH /upcoming/{id} (remove participant) | < 500ms P95 | Remove 1 participant |
| PATCH /upcoming/{id} (all changes) | < 500ms P95 | Change title, scheduled_start, description, org, + add/remove participants |

**Method:** Pre-create an upcoming meeting with the test user. Issue PATCH /upcoming/{id} with field variations. Measure round-trip. Exclude async briefing trigger work. Run 100 iterations per fixture; report p50, p95, p99.

### Cancel Latency

| Metric | Target | Method |
|--------|--------|--------|
| POST /upcoming/{id}/cancel | < 500ms P95 | Soft cancel within editable window |
| Cancel when briefing is queued | < 500ms P95 | Cancel meeting that has a queued briefing trigger |
| Cancel when briefing is generating | < 500ms P95 | Cancel meeting during active async generation |
| Cancel after editable window (reject) | < 200ms P95 | Rejection response for out-of-window cancel attempt |

**Method:** Pre-create upcoming meetings in various states. Issue cancel requests. Measure round-trip. Run 100 iterations; report p50, p95, p99.

---

## List / Detail / Calendar View Latency

### List View Latency

| Metric | Target | Method |
|--------|--------|--------|
| GET /upcoming (no filters) | < 500ms P95 | List user's non-cancelled upcoming meetings; up to 100 meetings |
| GET /upcoming (empty result) | < 100ms P95 | Authenticated user with no meetings |
| GET /upcoming (10 meetings) | < 300ms P95 | 10 scheduled meetings |
| GET /upcoming (50 meetings) | < 400ms P95 | 50 scheduled meetings |
| GET /upcoming (100 meetings) | < 500ms P95 | 100 scheduled meetings (MVP scale target) |

**Method:** Seed test user with N meetings. Issue GET /upcoming. Measure round-trip. Run 100 iterations per fixture; report p50, p95, p99.

### Detail View Latency

| Metric | Target | Method |
|--------|--------|--------|
| GET /upcoming/{id} | < 500ms P95 | Load meeting detail with participant metadata |
| GET /upcoming/{id} with 50 participants | < 500ms P95 | Max-participant fixture |
| GET /upcoming/{id} (cancelled) | < 300ms P95 | Detail of cancelled meeting (visible to owner) |

**Method:** Pre-create meetings with varying participant counts. Issue GET /upcoming/{id}. Measure round-trip. Run 100 iterations per fixture; report p50, p95, p99.

### Calendar View Latency

| Metric | Target | Method |
|--------|--------|--------|
| GET /upcoming/calendar | < 500ms P95 | Calendar-style view; up to 100 meetings |
| GET /upcoming/calendar (empty) | < 100ms P95 | No upcoming meetings |
| GET /upcoming/calendar (50 meetings) | < 400ms P95 | 50 meetings |

**Method:** Seed test user with N meetings. Issue GET /upcoming/calendar. Measure round-trip. Run 100 iterations per fixture; report p50, p95, p99.

---

## Briefing Trigger Enqueue Latency

> **NFR Target:** p95 < 100ms to enqueue or safely skip the BRD-04 briefing trigger after successful create or meaningful edit.

### Trigger Enqueue — Create Path

| Metric | Target | Method |
|--------|--------|--------|
| Enqueue on create (BRD-04 enabled) | < 100ms P95 | Both flags enabled; trigger enqueued after meeting persisted |
| Enqueue on create (BRD-04 disabled) | < 50ms P95 | Skip path; trigger skipped and observable via metric |
| Enqueue on create (meeting cancelled) | < 50ms P95 | Skip because cancelled; no enqueue attempted |
| Enqueue on create (outside eligible window) | < 50ms P95 | Skip because not eligible |

**Method:** Measure `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` histogram for `trigger=create` across 100 iterations per scenario. Exclude async generation time.

### Trigger Enqueue — Meaningful Edit Path

| Metric | Target | Method |
|--------|--------|--------|
| Enqueue on meaningful edit (BRD-04 enabled) | < 100ms P95 | Title change triggers regeneration |
| Enqueue on schedule change | < 100ms P95 | scheduled_start change triggers regeneration |
| Enqueue on participant add | < 100ms P95 | Adding participant is a meaningful edit |
| Enqueue on non-meaningful edit | < 50ms P95 | No-op edit; trigger skipped, no enqueue |
| Enqueue on edit (BRD-04 disabled) | < 50ms P95 | Skip path; trigger skipped |

**Method:** Pre-create meetings. Apply meaningful and non-meaningful edits. Measure `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` histogram for `trigger=meaningful_edit` across 100 iterations per scenario.

---

## Scale Targets

> **NFR Target:** MVP supports at least 100 active upcoming meetings per user and up to 50 participants per upcoming meeting without degrading p95 targets.

### Per-User Meeting Count Scaling

| Metric | Target | Method |
|--------|--------|--------|
| List with 100 meetings | < 500ms P95 | Seed 100 meetings; list and verify all returned |
| Detail with 100 meetings | < 500ms P95 | Each detail view joins on participants within target |
| Calendar with 100 meetings | < 500ms P95 | Calendar-style view with full meeting set |
| Create with 100 existing meetings | < 500ms P95 | Create 101st meeting; no performance degradation |

### Per-Meeting Participant Scaling

| Metric | Target | Method |
|--------|--------|--------|
| Create with 50 participants | < 500ms P95 | At hard limit; all 50 persisted |
| Update: add participant to 50-limit meeting | < 500ms P95 | Reject with `too_many_participants` within target |
| Detail with 50 participants | < 500ms P95 | All 50 participants returned |
| List with 50-participant meetings | < 500ms P95 | Does not slow list view |

**Method:** Seed fixtures at scale boundaries. Measure round-trip latency at p95 against target thresholds. Run 100 iterations.

---

## Running

```bash
# Requires services up (database, redis, app server)
make eval-perf

# Manual measurement
# Authenticate
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login   -H "Content-Type: application/json"   -d '{"email":"test@example.com","password":"test"}' | jq -r .token)

# Create benchmark
curl -X POST http://localhost:8080/upcoming   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"title":"perf-test","scheduled_start":"2026-06-01T10:00:00Z"}'   -w "
Total: %{time_total}s
"

# List benchmark
curl http://localhost:8080/upcoming   -H "Authorization: Bearer $TOKEN"   -w "
Total: %{time_total}s
"

# Check metrics
curl http://localhost:9090/metrics | grep cp_upcoming_meeting_action_duration_ms
curl http://localhost:9090/metrics | grep cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms
```

---

## NFR Summary

| Requirement | Target | Source |
|------------|--------|--------|
| Create/update/cancel latency | P95 < 500ms | BRD-05 NFR / AC-16 |
| List/detail/calendar view latency | P95 < 500ms | BRD-05 NFR / AC-16 |
| Briefing trigger enqueue latency | P95 < 100ms | BRD-05 NFR / AC-17 |
| Scale: 100 meetings per user | p95 targets met | BRD-05 NFR |
| Scale: 50 participants per meeting | p95 targets met | BRD-05 NFR / FR-6 |
| Availability | BRD-04 degradation must not block BRD-05 create | BRD-05 NFR |
| Privacy | No raw meeting titles, participant names, emails, org values in metrics/logs | BRD-05 NFR / FR-20 |
