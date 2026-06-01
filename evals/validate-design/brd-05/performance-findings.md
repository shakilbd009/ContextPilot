# Performance Findings: brd-05-manual-upcoming-meeting-creation

**Validated:** 2026-05-24
**Validator:** performance
**Target:** specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md
**BRDs reviewed in context:** brd-04 (trigger integration), brd-06 (retention/deferral)

---

## Verdict: NEEDS_ATTENTION

BRD-05 has well-defined latency targets and a clean scale envelope. Three findings require clarification or fix before implementation proceeds: two Medium (histogram bucket gaps, create-at-scale with BRD-04 trigger not tested) and one Low (trigger latency measurement scope ambiguity).

---

## 1. Create/Update/Cancel Latency — p95 < 500ms (NFR line 100)

**Spec requirement:** "Create/update/cancel latency: p95 < 500ms for BRD-05 server action completion, excluding async BRD-04 generation work."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Target (500ms), scope exclusion (async BRD-04), and measurement point (server action completion) are all stated |
| Metric | PASS | `cp_upcoming_meeting_action_duration_ms` histogram with `action` label (create/update/cancel/list/detail/calendar) and `result` label (success/failure) covers all CRUD operations |
| Histogram buckets | ADVISORY | BRD does not enumerate explicit bucket boundaries. Without defined upper bounds (e.g., 100ms, 250ms, 500ms, 1000ms), P95 calculation depends on implementation defaults. Must be specified before performance eval can verify SLO compliance. |
| BRD-04 exclusion clarity | PASS | Spec explicitly says "excluding async BRD-04 generation." The create endpoint returns as soon as the meeting is persisted and the trigger is enqueued (or safely skipped), not when briefing generation completes. |

**Note:** The 500ms target covers the full synchronous create/update/cancel handler — persistence + authorization + BRD-04 trigger check (but not async job execution). This is correctly scoped.

---

## 2. List/Detail/Calendar View Latency — p95 < 500ms (NFR line 101)

**Spec requirement:** "List/detail/calendar view latency: p95 < 500ms for current user's upcoming meeting data retrieval and render path, excluding client/network variability."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Target and exclusion (client/network variability) clearly stated |
| Metric | PASS | `cp_upcoming_meeting_action_duration_ms` covers list/detail/calendar actions |
| Histogram buckets | ADVISORY | Same bucket definition gap as Finding 1 — no explicit bucket boundaries in BRD |
| Scale test coverage | OPEN | NFR line 106 requires "MVP supports at least 100 active upcoming meetings per user... without degrading p95 targets in fixture tests." No list pagination or offset/limit controls are defined in the spec. At 100 meetings with default sort (nearest scheduled_start), query is a simple indexed range scan on `created_by = user_id AND status = 'scheduled' ORDER BY scheduled_start ASC`. This is efficient with a compound index. However, the p95 target for list has not been verified against the full 100-meeting fixture. |
| Index coverage | PASS | BRD data model (FR-5) implies index on `(created_by, status, scheduled_start)` for the default view. Implementation must verify this index exists. |

---

## 3. Briefing Trigger Enqueue Latency — p95 < 100ms (NFR line 102)

**Spec requirement:** "Briefing trigger enqueue latency: p95 < 100ms to enqueue or safely skip the BRD-04 briefing trigger after successful create or meaningful edit."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Target (100ms) and measurement point (enqueue or safely skip) clearly stated |
| Metric | PASS | `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` histogram with `trigger` (create/meaningful_edit) and `result` (queued/skipped/failed) labels |
| BRD-04 integration path | PASS | Trigger logic: (1) check both flags enabled, (2) check meeting is scheduled (not cancelled), (3) check within eligible window, (4) enqueue async job. Each step is a simple conditional with no blocking I/O. |
| Measurement scope | ADVISORY | `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` measures only the trigger enqueue/skip operation, NOT the full create handler. Per Finding 1, create/update/cancel target is 500ms. The 100ms target is a sub-path budget. This is correctly scoped in the spec, but the two metrics measure different things. |
| Trigger types | PASS | Both `create` and `meaningful_edit` trigger paths defined with distinct metric labels. FR-16 defines meaningful edit classes clearly. |

---

## 4. Scale: 100 Meetings + 50 Participants Per User (NFR line 106)

**Spec requirement:** "MVP supports at least 100 active upcoming meetings per user and up to 50 participants per upcoming meeting without degrading p95 targets in fixture tests."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Participant limit enforcement | PASS | FR-26 names `too_many_participants` validation class; integration eval tests 51-participant rejection. Limit enforced at API validation layer. |
| Meeting count limit | ABSENT | BRD-05 does not define a per-user meeting creation limit. There is no `max_meetings_per_user` validation. At 100 active meetings per user (NFR scale target), no mechanism prevents a user from creating a 101st meeting. This is an operational risk if the NFR is a hard ceiling rather than an expected load. |
| Create + 50 participants + BRD-04 trigger | OPEN | The 50-participant path has not been performance-tested with BRD-04 trigger enabled. Creating a meeting with 50 participants generates 50 participant rows in `upcoming_meeting_participants` plus 1 meeting row. If the trigger enqueues synchronously and BRD-04 must inspect participant data (for matching), the enqueue phase could involve participant loading. The 100ms enqueue target applies to trigger dispatch, not to participant FK integrity. No issue identified if trigger is fire-and-forget queue INSERT, but this must be verified in implementation. |
| List with 100 meetings | PASS (conditional) | Default sorted list (nearest scheduled_start) over 100 meetings per user requires `(created_by, status, scheduled_start)` index. Efficient if indexed. No pagination defined, so a user with 100 meetings gets all 100 in a single response. 100-row response is acceptable at < 500ms p95 for an indexed query. |
| Edit with 50 participants | PASS (conditional) | PATCH updates participant rows via FK. If edit adds/removes participants, trigger evaluates meaningful-edit criteria. With proper DB indexes on `upcoming_meeting_id`, participant updates are O(n) where n = participant changes, not total participants. |

---

## 5. Histogram Bucket Coverage

**Spec requirement:** `cp_upcoming_meeting_action_duration_ms` histogram (action: create/update/cancel/list/detail/calendar, result: success/failure) — bucket boundaries not specified in BRD.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Metric definition | PASS | Metric name, type (histogram), labels (action, result), and all label values are defined in BRD lines 129 |
| Histogram buckets | ADVISORY | BRD does not enumerate bucket boundaries. Without defined buckets, the histogram's P95 calculation is implementation-dependent. For a 500ms target, typical buckets would be: 50ms, 100ms, 250ms, 500ms, 1000ms, 5000ms. The eval file (evals/perf) is expected to define specific buckets, but the BRD itself must state them as the canonical contract. |
| `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` | SAME | Same bucket gap applies. For a 100ms target, buckets might be: 10ms, 25ms, 50ms, 100ms, 250ms, 500ms. |

---

## 6. Limit Enforcement — 50 Participants (FR-26)

**Spec requirement:** "Low-cardinality validation classes: ... `too_many_participants`..."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Enforcement in validation layer | PASS | Unit eval (lines 57–61) tests 50 (accepted) and 51 (rejected with `too_many_participants`). Integration eval (lines 117–129) confirms the 51-participant rejection path. |
| Enforcement location | PASS | Validation is at API handler layer, not at database level. This is acceptable for MVP since the limit is a business rule (scale ceiling), not a data integrity constraint. Database FK from `upcoming_meeting_participants.upcoming_meeting_id` enforces referential integrity only. |
| Participant ergonomic at scale | OPEN | FR-25: "creation/edit form supports adding, removing, and editing multiple participant rows without losing entered values during validation errors." At 50 participants with form echo on error (FR-15), the server must echo back all 50 participant rows in a validation error response. With ~500 byte/participant, that's ~25KB of echoed data. This is acceptable but not performance-tested. |

---

## 7. Availability — BRD-04 Degradation Isolation (NFR line 103)

**Spec requirement:** "Upcoming meeting create/list/detail/edit/cancel follow app-baseline availability; BRD-04 generation degradation must not prevent BRD-05 meeting creation when BRD-05 is healthy."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Async trigger isolation | PASS | FR-17/FR-18: creation returns after enqueue or safe skip; does not wait for BRD-04 generation. |
| Trigger failure handling | PASS | FR-19: trigger is "skipped safely and observably" when BRD-04 disabled, dependencies unavailable, or meeting is cancelled. Counter `cp_upcoming_meeting_briefing_trigger_skipped_total` with `reason` label tracks all skip cases. |
| `/ready` endpoint | PASS | BRD lines 158–163: `/ready` reflects upcoming meeting storage health. BRD-04 queue degradation must not make BRD-05 unavailable. Correctly scoped to storage-only dependency. |
| BRD-04 trigger flag dual-gate | PASS | Both `FF_ENABLE_UPCOMING_MEETINGS` AND `FF_ENABLE_PRE_CALL_BRIEFING` must be true for trigger. Either disabled skips safely via `pre_call_briefing_disabled` or `dependency_unavailable` reason on the skipped counter. |

---

## Summary Table

| ID | Severity | Area | Finding |
|----|----------|------|---------|
| 1 | Medium | Histogram buckets | `cp_upcoming_meeting_action_duration_ms` has no defined bucket boundaries in BRD; P95 verification depends on implementation defaults |
| 2 | Medium | Scale | Create + 50 participants + BRD-04 trigger path not performance-tested; trigger may involve participant loading which is not scoped |
| 3 | Low | Limit enforcement | No per-user meeting count limit defined; 100-meeting NFR target has no hard ceiling enforcement |
| 4 | Low | Trigger latency scope | `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` measures only trigger sub-path, not full create latency; ambiguity if create fails to meet 500ms due to trigger path |
| 5 | Low | Participant form echo | 50-participant form echo on validation error not performance-tested; ~25KB response at scale acceptable but unverified |

---

## Positive Findings

- Latency targets are clearly stated with measurement scope and exclusions (client/network variability, async BRD-04 generation)
- Metric labels are low-cardinality enums; no raw user content (titles, emails, participant names) in metric labels
- Trigger is correctly async; creation does not block on BRD-04 generation
- Trigger skip logic is well-defined with observable counters for all failure modes
- Scale envelope (100 meetings × 50 participants) is stated and testable
- List query sort is on `scheduled_start` which is efficiently indexed (B-tree)
- Edit window (15 min after scheduled_start) is enforced consistently for both edit and cancel
- Owner-only authorization enforced at API layer; no data leakage to non-owners
- `/ready` correctly scoped to storage-only; BRD-04 queue degradation does not affect readiness
- Participant identity validation (FR-6) prevents access confusion at the matching layer

---

## Recommendations

1. **Add histogram bucket definitions to BRD-05 observability section.** Define explicit bucket boundaries for `cp_upcoming_meeting_action_duration_ms` (e.g., 25ms, 50ms, 100ms, 250ms, 500ms, 1000ms) and for `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` (e.g., 10ms, 25ms, 50ms, 100ms, 250ms, 500ms).

2. **Clarify whether 100-meeting limit is a hard ceiling or expected load.** If hard ceiling, add `max_upcoming_meetings_per_user` validation. If expected load, document acceptable degradation at 100+.

3. **Performance test the 50-participant + BRD-04 trigger path** before considering BRD-05 performance validation complete. Verify trigger enqueue latency (p95 < 100ms) and full create latency (p95 < 500ms) under this scenario.

4. **Specify participant echo response size budget** in NFR or implementation guidance. 50-participant form echo on validation error produces ~25KB of JSON; ensure this does not cause request/response size issues in the SLO path.