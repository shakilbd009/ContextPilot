# Production Readiness Checklist — BRD-04 Pre-Call Briefing

Generated from BRD-04 curated docs and eval contracts on 2026-05-23.
Validator task: t_c1b841b7 (all blocking/high/medium findings resolved).
Production-checklist task: t_3b1a4172.

---

## 1. Infrastructure & Deployment

- [ ] **Go API deployed to hosting target** (Cloud Run or equivalent; confirm runtime supports async job queue)
- [ ] **upcoming_meetings table exists** — BRD-05 schema (FR-5) must be implemented before or alongside BRD-04. Required for `briefing_versions.upcoming_meeting_id` and `briefing_source_exclusions.upcoming_meeting_id` FK references
- [ ] **briefing_versions table created** with all required fields: `id` (UUID PK), `upcoming_meeting_id` (UUID FK), `version_number` (integer, monotonically increasing), `status` (active/superseded), `is_active` (boolean), `result` (ready/ready_with_caveats/no_prior_memory/failed), `preparation_status` (generating/ready/ready_with_caveats/no_prior_memory/stale/failed/regenerating), `content` (JSONB), `source_count` (integer), `created_at`, `trigger_type` (auto/manual_regenerate)
- [ ] **UNIQUE constraint** on `(upcoming_meeting_id, version_number)` in `briefing_versions`
- [ ] **Index** on `(upcoming_meeting_id, is_active) WHERE is_active = TRUE`
- [ ] **briefing_source_exclusions table created** with: `id` (UUID PK), `upcoming_meeting_id` (UUID FK), `excluded_source_meeting_id` (UUID FK to meetings.id), `created_at`, `restored_at` (TIMESTAMPTZ NULL — soft-delete per ADR-0012)
- [ ] **UNIQUE constraint** on `(upcoming_meeting_id, excluded_source_meeting_id)` in `briefing_source_exclusions`
- [ ] **Active exclusions** queried via `WHERE upcoming_meeting_id = $id AND restored_at IS NULL`
- [ ] **Async job queue** mechanism for briefing generation jobs (retry behavior, failure handling); architecture assumed consistent with BRD-03 memory processing queue pattern
- [ ] **FF_ENABLE_PRE_CALL_BRIEFING=true** set in server environment
- [ ] **VITE_FF_ENABLE_PRE_CALL_BRIEFING=true** set in frontend environment
- [ ] **Database migrations** for all new tables run successfully against production schema before flag enablement

---

## 2. Security & Authorization

- [ ] **All 6 briefing endpoints require authenticated session** — confirm 401 on no session cookie for all paths:
  - `GET /upcoming/{meetingId}/briefing`
  - `GET /upcoming/{meetingId}/briefing/versions`
  - `GET /upcoming/{meetingId}/briefing/versions/{versionNumber}`
  - `POST /upcoming/{meetingId}/briefing/regenerate`
  - `POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude`
  - `POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore`
- [ ] **ACL enforcement on upcoming meeting** — 403 for authenticated users who do not own the upcoming meeting and are not in its ACL (per ADR-0009)
- [ ] **Source meeting authorization enforced** — unauthorized source meetings must not be selected, displayed, logged, or used in generation
- [ ] **Unauthorized source in GET response** — GET `/briefing` and GET `/briefing/versions/{n}` must return 403 or redact unauthorized source content; no raw PII or meeting content from unauthorized sources in response
- [ ] **Regenerate/restore with unauthorized source in superseded version history** — unauthorized source must not appear in new briefing content, logs, or be restored; restoration of an unauthorized source must return 403 with no mutation
- [ ] **Flag mismatch detection** — when server=false and browser=true, server returns 403; `cp_pre_call_briefing_flag_misconfiguration_total` counter incremented; `briefing.flag_misconfiguration` log event emitted
- [ ] **Feature flag server-authoritative** — browser flag cannot bypass server enforcement for data mutations
- [ ] **Input validation: UUID format** — meetingId path parameter rejects non-UUID input with 400 Bad Request
- [ ] **Input validation: integer version** — versionNumber path parameter rejects non-integer with 400 Bad Request
- [ ] **Safe failure reasons** — user-facing errors must not include meeting titles, participant names/emails, transcript snippets, notes content, generated briefing text, or any PII. Operator logs carry correlation ID linking safe user message to detailed diagnostic channel
- [ ] **No raw content in logs/metrics** — `cp_briefing_generation_duration_ms` histogram labels use only low-cardinality enums (status, failure_class, source_count_bucket, trigger_type); raw titles, PII, snippets, briefing text, and participant attributes forbidden from all metric labels and log event fields

---

## 3. Data & Migrations

- [ ] **upcoming_meetings table** created per BRD-05 FR-5 (schema: id, title, scheduled_start, created_by, created_at, updated_at)
- [ ] **briefing_versions table** created with all columns and constraints per BRD-04 Data Model
- [ ] **briefing_source_exclusions table** created with `restored_at` soft-delete column per ADR-0012
- [ ] **Source exclusion undo mechanism** — rows with `restored_at IS NOT NULL` remain visible in collapsed excluded-sources area (FR-17)
- [ ] **Briefing version immutability** — once created, a briefing version is never modified in place; regeneration creates a new distinct version
- [ ] **briefing_versions.is_active** — exactly one active version per upcoming meeting; on regeneration, previous active is marked superseded before new active row is created
- [ ] **Failed generation creates superseded row** — failed generation always creates a `briefing_versions` row with `status=superseded` and `result=failed`; previous active briefing left readable as superseded version
- [ ] **Stale indicator at read time** — staleness is derived on GET `/briefing` (not at generation time) by comparing `source_memory_version_ids` captured at generation against current source memory versions
- [ ] **Regeneration race condition** — regeneration job reads source memory state at job start time, not trigger time; newer source versions used if changed between trigger and start
- [ ] **Retention policy** — briefing versions inherit retention/deletion from associated upcoming meeting; BRD-06 privacy and retention controls deferred

---

## 4. Observability

- [ ] **`cp_briefing_generation_duration_ms` histogram** deployed with labels: `status` (success/failure), `failure_class` (none/upstream_error/timeout/internal_error), `source_count_bucket` (0/1/2/3), `trigger_type` (auto/manual_regenerate)
- [ ] **`cp_briefing_queue_saturation` gauge** deployed with label: `queue_name`
- [ ] **15+ log events implemented** with low-cardinality fields only:
  - `briefing.generation.started` — upcoming_meeting_id, trigger_type, source_count_expected
  - `briefing.generation.completed` — upcoming_meeting_id, status, version_number, source_count_used, duration_ms
  - `briefing.generation.failed` — upcoming_meeting_id, failure_class, is_retry
  - `briefing.viewed` — upcoming_meeting_id, version_number, has_prior_memory
  - `briefing.source.excluded` — upcoming_meeting_id, source_meeting_id
  - `briefing.source.restored` — upcoming_meeting_id, source_meeting_id
  - `briefing.regenerate.requested` — upcoming_meeting_id, source_count
  - `briefing.no_prior_memory_shell.generated` — upcoming_meeting_id
  - `briefing.stale.detected` — upcoming_meeting_id, version_number, stale_source_count
  - `briefing.flag_misconfiguration` — server_flag_value, browser_flag_value, endpoint
- [ ] **No high-cardinality content in metric labels** — confirmed by security eval: no raw titles, PII, snippets, or briefing text in any label
- [ ] **Operator diagnostic correlation** — safe user-facing failure messages linked via correlation ID to detailed non-interactive operator logs (stack trace, pipeline stage, upstream response)
- [ ] **`cp_pre_call_briefing_flag_misconfiguration_total`** counter incremented on flag mismatch detection

---

## 5. Performance

- [ ] **Generation latency P95 < 60,000ms** for up to 3 qualifying prior meetings, measured job-start to terminal state (excludes queue wait)
- [ ] **Generation latency P99 < 90,000ms** for up to 3 qualifying prior meetings
- [ ] **Shell generation (no qualifying sources) < 10,000ms** P95
- [ ] **Cached briefing view P95 < 500ms** — measured client-visible render time for latest completed briefing
- [ ] **Version list GET P95 < 200ms** — leveraging index on `(upcoming_meeting_id, version_number)`
- [ ] **Specific version GET P95 < 300ms**
- [ ] **Concurrency target: 10 concurrent generation jobs** at normal-operating capacity before degradation
- [ ] **Queue depth degradation threshold: 50 pending jobs** — latency targets apply under normal load; confirm queue saturation alerting at cp_briefing_queue_saturation threshold
- [ ] **Signal count scaling** — relatedness scoring confirmed to scale linearly with signal count; max 3 sources × 4 signals = 12 total signals within P95 budget
- [ ] **Progressive format rendering** — concise summary readable in under 60 seconds per usability acceptance criterion AC-8

---

## 6. Accessibility

- [ ] **WCAG 2.1 AA compliance verified** against BRD-01 component inventory before implementation (BRD-04 NFR requirement)
- [ ] **ARIA live region: `generating` state** — role=status, aria-live=polite, aria-atomic=true; announces "Briefing generation in progress" on entry
- [ ] **ARIA live region: `stale` state** — role=status, aria-live=polite, aria-atomic=true; announces "Briefing may be outdated; regeneration available" when staleness is first computed
- [ ] **ARIA live region: `failed` state** — role=alert, aria-live=assertive, aria-atomic=true; announces failure reason immediately; retry controls are focusable
- [ ] **ARIA live region: `regenerating` state** — role=status, aria-live=polite, aria-atomic=true; announces "Regeneration in progress"
- [ ] **Focus management on `failed`** — focus moves to failure message region or retry affordance on transition to failed
- [ ] **Stale indicator visibility** — stale indicator appears on briefing card/header on page render without requiring user action
- [ ] **Expand/details controls** — relatedness reasons and source details behind expand/details affordance; concise summary remains readable without inline source clutter (FR-6, FR-12)
- [ ] **Progressive format** — concise summary renders first and remains scannable without expanding detailed sections

---

## 7. Eval & Testing

- [ ] **All 5 BRD-04 eval contracts pass** before production go-live:
  - `evals/e2e/brd-04-pre-call-briefing.md` — E2E scenarios for AC-1 through AC-22
  - `evals/unit/brd-04-pre-call-briefing.md` — signal matching unit tests, preparation status state machine
  - `evals/integration/brd-04-pre-call-briefing.md` — API contract and data model integration tests
  - `evals/security/brd-04-pre-call-briefing.md` — authorization, input validation, flag mismatch, PII redaction
  - `evals/perf/brd-04-pre-call-briefing.md` — generation latency, cached view latency, concurrency benchmarks
- [ ] **`evals/architecture/check-feature-flags.sh` passes** — `FF_ENABLE_PRE_CALL_BRIEFING` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING` are registered in `specs/feature-flags.md` and present in `.env.example`
- [ ] **Flag parity eval passes** — AC-1: both flags false hides entry points and rejects requests; AC-22: authorization checks block unauthorized source meetings
- [ ] **Signal matching two-signal threshold** — confirmed by unit eval: qualifying requires ≥2 of same participant, similar title, same org/client, close chronology; future-dated sources excluded; email-display-name-only matching excluded
- [ ] **No-prior-memory shell** — integration eval confirms shell is clearly marked "No prior memory found" and does not imply prior continuity, decisions, actions, stakeholder memory, or historical risks
- [ ] **Evidence policy compliance** — unit and content safety evals confirm: weak-evidence items appear with clear caveats; insufficient-evidence items not presented as facts; unresolved conflicting items excluded from briefing advice and stakeholder notes
- [ ] **Observability security eval passes** — no raw transcript text, notes text, briefing text, source snippets, participant PII, stakeholder note content, or meeting titles in metric labels or log event fields
- [ ] **check-status-sync.sh passes** after all checks complete

---

## 8. Rollout & Feature Flags

- [ ] **FF_ENABLE_PRE_CALL_BRIEFING=true** in server environment (backend controls all data mutations)
- [ ] **VITE_FF_ENABLE_PRE_CALL_BRIEFING=true** in frontend environment (browser build-time embedding)
- [ ] **Flag mismatch alerting active** — `briefing.flag_misconfiguration` log event and `cp_pre_call_briefing_flag_misconfiguration_total` counter operational
- [ ] **Server flag is authoritative** — browser flag cannot be used to bypass server-side enforcement for generation, regeneration, exclude, or restore operations
- [ ] **Staged rollout consideration** — MVP supports up to 10 concurrent generation jobs; if pre-call usage is initially high, queue depth monitoring should be active before full user rollout
- [ ] **BRD-02 and BRD-03 must be implemented and functional** — BRD-04 briefing generation depends on source meeting data (BRD-02) and source memory/evidence/conflict statuses (BRD-03). These are hard prerequisites; do not enable FF_ENABLE_PRE_CALL_BRIEFING in production before BRD-02/BRD-03 are live.
- [ ] **BRD-05 upcoming_meetings table must exist** — required before briefing generation can be triggered on upcoming meeting creation

---

## 9. Rollback & Operations

- [ ] **Rollback procedure documented:**
  1. Set `FF_ENABLE_PRE_CALL_BRIEFING=false` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING=false`
  2. Server rejects all briefing generation/regeneration requests with safe feature-disabled response
  3. Existing briefing versions remain in `briefing_versions` but are inaccessible (no active feature flag gates access)
  4. `briefing_source_exclusions` rows remain in DB with no effect on generation
  5. No user data loss — source meetings and memory are unaffected by rollback
- [ ] **Briefing generation failure runbook** — safe user message ("Briefing generation failed — please try again"), correlation ID for operator diagnostics, operator logs must not expose raw meeting content or participant PII
- [ ] **Stale briefing runbook** — staleness detected at read time; visible stale indicator shown; regeneration offered but not auto-triggered; user agency preserved
- [ ] **Queue saturation runbook** — `cp_briefing_queue_saturation` gauge at or near 100% of normal-operating capacity (50 jobs) triggers operator alert; degradation begins above this threshold; evacuation or scale-up procedure documented
- [ ] **Regeneration continuity confirmed** — latest completed briefing remains readable during queued/running/failed regeneration; no user-facing downtime during regeneration cycle (NFR confirmed)
- [ ] **No raw content in operator logs** — operator diagnostic channel (linked via correlation ID to safe user message) must redact all meeting content, PII, snippets, and briefing text

---

## 10. Deferred Items (Implementation-Addressable)

These items were identified as low-priority deferrals by the validator. They do not block production go-live but should be tracked for future sprints:

- [ ] **OQ-4: Progressive format sufficiency threshold** — judgment call during implementation: whether the progressive format alone is sufficient for length management, or whether a hard section cap is needed. Tracked as a future enhancement (FR-28).
- [ ] **OQ-5: Aggregation rule example** — aggregation rule documentation (e.g., how items from multiple source meetings are merged in detailed sections) is nice-to-have but not blocking.
- [ ] **Optimistic locking on exclusion/restore** — UNIQUE constraint currently sufficient per validator; re-evaluate if concurrent exclusion/restore operations on the same source meeting become a problem.
- [ ] **BRD-01 component inventory verification** — WCAG 2.1 AA compliance verification against BRD-01 component inventory is required before implementation (NFR). BRD-01 implementation task should verify component inventory completeness when that task runs.

---

## Verification Commands

```bash
# Sync check must pass
./scripts/check-status-sync.sh

# Architecture fitness checks
./evals/architecture/check-feature-flags.sh
./evals/architecture/check-no-panic.sh
./evals/architecture/check-no-background-context.sh

# All 5 BRD-04 evals must pass (🔴 → 🟢 for implementation to be complete)
# Integration eval from subdirectory (has its own go.mod):
cd evals/integration && go test -v ./...
```

---

*Production checklist generated by ops profile. downstream gate: PM/orchestrator after checklist completion.*