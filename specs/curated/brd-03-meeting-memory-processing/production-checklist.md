# Production Readiness Checklist — BRD-03 Meeting Memory Processing

Generated from BRD-03 curated docs and implementation on 2026-05-25.
Source: implementation-readiness.md (t_b08a63fd).

---

## 1. Infrastructure & Deployment

- [ ] **Go API deployed to hosting target** (Cloud Run or equivalent; confirm runtime supports async job queue with PostgreSQL advisory lock pattern per ADR-0005)
- [ ] **memory_processing_jobs table exists** — job queue with fields: id, meeting_id, trigger_type, status, retry_count, max_retries, failure_reason, previous_version_id, correlation_id, queued_at, started_at, completed_at, next_retry_at
- [ ] **Indexes on memory_processing_jobs**: (meeting_id), (status) WHERE status IN ('queued', 'retrying'), (correlation_id)
- [ ] **memory_versions table exists** — version history with: id, meeting_id, job_id, version_number, status, is_active, content (JSONB), trigger_type, created_at, created_by
- [ ] **UNIQUE constraint** on `(meeting_id, version_number)` in memory_versions
- [ ] **Index** on `(meeting_id, is_active) WHERE is_active = TRUE`
- [ ] **meetings.active_memory_version_id column added** — FK to memory_versions(id) ON DELETE SET NULL
- [ ] **memory_evidence table exists** — normalized evidence with: id, memory_version_id, category, item_id, source_type, source_location (JSONB char_offset), evidence_snippet, quality_status, created_at
- [ ] **Index** on memory_evidence(memory_version_id)
- [ ] **memory_conflicts table exists** — conflict records with: id, meeting_id, memory_version_id, conflicting_items (JSONB), quality_status, review_status, resolution_note, resolved_at, resolved_by, created_at
- [ ] **Indexes on memory_conflicts**: (meeting_id), (review_status) WHERE review_status = 'pending'
- [ ] **memory_prior_memory_inputs table exists** — prior-memory tracking with: id, memory_version_id, prior_memory_version_id, match_confidence, included_by_user, excluded_by_user, created_at
- [ ] **UNIQUE constraint** on `(memory_version_id, prior_memory_version_id)` in memory_prior_memory_inputs
- [ ] **Async job queue** mechanism for memory processing jobs (retry behavior: 3 retries, exponential backoff 1s→30s, jitter ±10%; transient vs non-transient failure handling); uses PostgreSQL advisory locks with FOR UPDATE SKIP LOCKED per ADR-0005
- [ ] **FF_ENABLE_MEETING_MEMORY_PROCESSING=false** in server environment (default false; set true only after all checklist items verified)
- [ ] **VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false** in frontend environment (default false; set true only after all checklist items verified)
- [ ] **Database migrations** for all memory tables run successfully against production schema before flag enablement

---

## 2. Security & Authorization

- [ ] **All 7 memory endpoints require authenticated session** — confirm 401 on no session cookie for all paths:
  - `GET /meetings/{id}/memory`
  - `GET /meetings/{id}/memory/versions`
  - `GET /meetings/{id}/memory/versions/{versionNumber}`
  - `POST /meetings/{id}/memory/reprocess`
  - `GET /meetings/{id}/memory/state`
  - `GET /meetings/{id}/memory/conflicts`
  - `POST /meetings/{id}/memory/conflicts/{conflictId}/resolve`
- [ ] **ACL enforcement on meeting** — authenticated user must own or have ACL access to the meeting; 403 otherwise
- [ ] **FF_ENFORCEMENT: when FF_ENABLE_MEETING_MEMORY_PROCESSING=false, all 7 endpoints return 403** with safe error message (no raw meeting content, no PII)
- [ ] **Input validation: UUID format** — meetingId path parameter rejects non-UUID input with 400 Bad Request
- [ ] **Input validation: integer version** — versionNumber path parameter rejects non-integer with 400 Bad Request
- [ ] **Flag mismatch detection** — when server=false and browser=true, server returns 403; `cp_meeting_memory_flag_misconfiguration_total` counter incremented with `flag` label
- [ ] **Feature flag server-authoritative** — browser flag cannot bypass server enforcement for data mutations
- [ ] **Safe failure reasons** — user-facing errors must not include meeting titles, participant names/emails, transcript snippets, notes content, generated memory text, or any PII. Operator logs carry correlation ID linking safe user message to detailed diagnostic channel
- [ ] **No raw content in logs/metrics** — all metric label cardinalities confirmed low: trigger_type, status, failure_class, reason. Forbidden: raw transcript text, raw notes text, evidence snippets, participant PII, stakeholder note content, generated memory text, prior memory content, resolution note text, source locations
- [ ] **check-no-sensitive-content.sh passes** — confirms no hardcoded sensitive meeting-derived content in logs/metrics

---

## 3. Data & Migrations

- [ ] **memory_processing_jobs trigger_type values enforced** — CHECK constraint: ('import', 'reprocess', 'stale_reprocess', 'manual_retry', 'conflict_resolution')
- [ ] **memory_processing_jobs status values enforced** — CHECK constraint: ('queued', 'processing', 'completed', 'completed_with_insufficient_evidence', 'failed', 'retrying', 'retry_exhausted')
- [ ] **memory_versions status values enforced** — CHECK constraint: ('active', 'superseded', 'conflict_review')
- [ ] **memory_evidence source_type values enforced** — CHECK constraint: ('transcript', 'notes', 'prior_memory_reference')
- [ ] **memory_evidence quality_status values enforced** — CHECK constraint: ('strong_evidence', 'weak_evidence', 'insufficient_evidence')
- [ ] **memory_conflicts review_status values enforced** — CHECK constraint: ('pending', 'reviewed')
- [ ] **memory_prior_memory_inputs match_confidence values enforced** — CHECK constraint: ('safe_match', 'uncertain')
- [ ] **Exactly one is_active=true per meeting** — enforced by application logic; migration creates index but does not add DB-level constraint
- [ ] **Previous active version preserved on reprocess** — failed processing does not replace active successful memory version; `previous_version_id` set on job so app can restore
- [ ] **Stale detection** — source edit (transcript, notes, participants, title, completed_at, other metadata) marks active memory version stale and queues reprocessing; staleness derived by comparing meeting's updated_at against memory_versions.created_at of active version
- [ ] **Conflict atomic unit** — resolving one conflict resolves that conflict only; atomic per conflict, not all conflicts for a meeting
- [ ] **Conflict exclusion from briefing** — unresolved conflicting items excluded from normal memory categories and from downstream briefing eligibility until resolved
- [ ] **Version immutability** — once created, a memory version row is never modified in place; new processing creates new version
- [ ] **Retention policy** — memory versions and evidence snippets retained with source meeting until BRD-06 defines final retention, deletion, export, and redaction policy (deferred)

---

## 4. Observability

- [ ] **`cp_meeting_memory_processing_queued_total` counter** deployed with label: `trigger_type`
- [ ] **`cp_meeting_memory_processing_started_total` counter** deployed with label: `trigger_type`
- [ ] **`cp_meeting_memory_processing_completed_total` counter** deployed with labels: `trigger_type`, `status`
- [ ] **`cp_meeting_memory_processing_failed_total` counter** deployed with labels: `trigger_type`, `failure_class`
- [ ] **`cp_meeting_memory_processing_retried_total` counter** deployed with label: `trigger_type`
- [ ] **`cp_meeting_memory_processing_retry_exhausted_total` counter** deployed with label: `trigger_type`
- [ ] **`cp_meeting_memory_processing_duration_ms` histogram** deployed with buckets: 100, 500, 1000, 5000, 10000, 30000, 60000, 120000 (no labels)
- [ ] **`cp_meeting_memory_processing_queue_wait_ms` histogram** deployed with buckets: 100, 500, 1000, 5000, 10000, 30000, 60000 (no labels)
- [ ] **`cp_meeting_memory_active_version_changed_total` counter** deployed with label: `reason`
- [ ] **`cp_meeting_memory_conflicts_detected_total` counter** deployed with label: `meeting_id` (safe UUID, not content)
- [ ] **`cp_meeting_memory_flag_evaluation_ms` histogram** deployed with buckets: 1, 5, 10, 25, 50, 100 (no labels)
- [ ] **`cp_meeting_memory_flag_misconfiguration_total` counter** deployed with label: `flag`
- [ ] **15 log events implemented** with low-cardinality fields only:
  - `memory.job.queued` — correlation_id, meeting_id, trigger_type
  - `memory.job.started` — correlation_id, meeting_id, trigger_type
  - `memory.job.completed` — all allowed fields
  - `memory.job.failed` — correlation_id, meeting_id, trigger_type, failure_class, retry_count, duration_ms
  - `memory.job.retrying` — correlation_id, meeting_id, trigger_type, retry_count, duration_ms, next_retry_at
  - `memory.job.retry_exhausted` — correlation_id, meeting_id, trigger_type, retry_count, failure_class
  - `memory.version.activated` — correlation_id, meeting_id, processing_version_number, active_version_change_reason
  - `memory.conflict.detected` — correlation_id, meeting_id, category, conflict_id
  - `memory.conflict.resolved` — correlation_id, meeting_id, conflict_id, resolved_by
  - `memory.source.stale` — correlation_id, meeting_id, edited_field
  - `memory.prior_inputs.updated` — correlation_id, meeting_id, included_count, excluded_count
- [ ] **Health/readiness endpoints** — `GET /ready` returns 200 when feature flag evaluator and PostgreSQL are reachable (independent of worker/provider); `GET /live` returns 200 when process alive
- [ ] **No high-cardinality content in metric labels** — confirmed by check-no-sensitive-content.sh; no raw titles, PII, snippets, evidence text in any label
- [ ] **Operator diagnostic correlation** — safe user-facing failure messages linked via correlation ID to detailed non-interactive operator logs

---

## 5. Performance

- [ ] **Processing latency P95 < 30,000ms** for 50,000-character transcript-plus-notes input under normal load, measured job pick-up to completion (excludes queue wait)
- [ ] **Queue wait tracked separately** via `cp_meeting_memory_processing_queue_wait_ms` histogram
- [ ] **Import path latency** — successful manual meeting import and redirect must not wait on memory processing completion; job queued after transaction commits; meeting detail shows memory in `queued` state immediately
- [ ] **Concurrency target: N concurrent jobs** at normal-operating capacity — confirm queue depth alerting if job saturation exceeds threshold
- [ ] **Version list GET P95 < 200ms** — leveraging index on `(meeting_id, version_number)`
- [ ] **Specific version GET P95 < 300ms**
- [ ] **Conflict resolution P95 < 500ms**
- [ ] **Stale detection at read time** — staleness derived on GET `/meetings/{id}/memory` not at processing time; no staleness check blocking response

---

## 6. Accessibility

- [ ] **WCAG 2.1 AA compliance** verified against BRD-01 component inventory before implementation (BRD-03 NFR requirement)
- [ ] **Memory state indicators** — briefing-readiness state visible on meeting detail page; `queued`, `processing`, `completed`, `failed`, `stale`, `retry_exhausted` states communicated without exposing raw meeting content
- [ ] **Retry/reprocess affordances** — accessible retry and reprocess controls for `failed`, `retry_exhausted`, `stale` states; keyboard accessible and screen-reader announced
- [ ] **Conflict review queue accessible** — conflict list, evidence snippets, resolution note field, and "Mark Reviewed" action all accessible; ARIA live regions for state changes
- [ ] **Evidence snippet display** — HTML-escaped evidence snippets rendered safely; FR-4a and FR-13a sanitization confirmed (no raw HTML from source text or user-authored resolution notes executed in browser)

---

## 7. Eval & Testing

- [ ] **All 5 BRD-03 eval contracts pass** before production go-live:
  - `evals/e2e/brd-03-meeting-memory-processing.md` — E2E scenarios for US-1 through US-8
  - `evals/unit/brd-03-meeting-memory-processing.md` — processing state machine, evidence grounding, quality status assignment, prior-memory matching, conflict detection
  - `evals/integration/brd-03-meeting-memory-processing.md` — API contract and data model integration tests
  - `evals/security/brd-03-meeting-memory-processing.md` — authorization, input validation, flag mismatch, PII redaction, evidence snippet and resolution note HTML sanitization
  - `evals/perf/brd-03-meeting-memory-processing.md` — processing latency, queue wait, import path latency, concurrent job capacity
- [ ] **`evals/architecture/check-feature-flags.sh` passes** — `ff_enable_meeting_memory_processing` registered in `specs/feature-flags.md` as Phase 2 In Implementation; both `FF_ENABLE_MEETING_MEMORY_PROCESSING` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` present in `.env.example` with default `false`
- [ ] **`evals/architecture/check-no-panic.sh` passes** — no `panic()` in production code
- [ ] **`evals/architecture/check-no-background-context.sh` passes** — no `context.Background()` in production code
- [ ] **`evals/architecture/check-no-sensitive-content.sh` passes** — no hardcoded sensitive meeting-derived content in logs/metrics
- [ ] **Flag parity eval passes** — server=false/browse=false blocks all 7 endpoints with 403; server=true/browse=true enables full access; server=true/browse=false hides browser UI but server accepts authenticated requests
- [ ] **Strict hallucination eval passes** — tempting but unsupported claims are omitted or marked `Insufficient evidence`; every supported item has evidence that actually supports the item
- [ ] **check-status-sync.sh passes** after all checks complete

---

## 8. Rollout & Feature Flags

- [ ] **FF_ENABLE_MEETING_MEMORY_PROCESSING=false** in server environment before production go-live (backend controls all data mutations; default stays false until all items verified)
- [ ] **VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false** in frontend environment before production go-live
- [ ] **Flag mismatch alerting active** — `cp_meeting_memory_flag_misconfiguration_total` counter with `flag` label incremented on mismatch; visible in Prometheus
- [ ] **Server flag is authoritative** — browser flag cannot be used to bypass server-side enforcement for data mutations
- [ ] **Staged rollout consideration** — MVP supports N concurrent processing jobs; if usage is initially high, queue depth monitoring should be active before full user rollout
- [ ] **BRD-02 (manual meeting import) must be implemented and functional** — BRD-03 processing is triggered after BRD-02 meeting save; BRD-02 is a hard prerequisite; do not enable FF_ENABLE_MEETING_MEMORY_PROCESSING in production before BRD-02 is live
- [ ] **No prior-memory processing until BRD-03 is complete** — do not enable processing before all checklist items verified; prior-memory matching depends on full BRD-03 infrastructure

---

## 9. Rollback & Operations

- [ ] **Rollback procedure documented:**
  1. Set `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false`
  2. Server rejects all memory processing requests with safe feature-disabled response (403)
  3. Existing memory versions remain in `memory_versions` but are inaccessible (no active feature flag gates access)
  4. `memory_conflicts` rows remain in DB with no effect on generation
  5. No user data loss — source meetings and memory are unaffected by rollback
  6. Queue jobs remain `queued` until worker picks them up and evaluates the flag; flag=false stops new jobs from being queued
- [ ] **Processing worker failure runbook** — jobs remain `queued`; existing active memory continues to serve; meeting detail shows `queued` state; no user content lost
- [ ] **Provider outage runbook** — `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` disables processing; manual import continues without memory processing; existing memory remains accessible
- [ ] **Bad memory version runbook** — failed processing does not replace active successful memory version; user can manually reprocess via FR-21
- [ ] **Stale source runbook** — source edit detection marks memory `stale` and queues reprocessing; previous active version serves briefing until new version succeeds
- [ ] **Data corruption runbook** — `memory_versions` entries are immutable; new processing creates new version; `memory_conflicts` audit trail preserved
- [ ] **No raw content in operator logs** — operator diagnostic channel (linked via correlation ID to safe user message) must redact all meeting content, PII, snippets, and memory text

---

## 10. Deferred Items (Implementation-Addressable)

These items do not block production go-live but are tracked for future sprints:

| Item | Deferred To | Rationale |
|------|-------------|-----------|
| Provider-specific latency SLO calibration | Phase 2 (post-provider-selection) | Cannot calibrate without concrete provider; 30s P95 is planning estimate |
| Semantic similarity threshold for conflict detection | Phase 2 eval | Provider-specific; calibrated during Phase 2 |
| Quality distribution metric expansion criteria | Phase 2 eval | Decision trigger: if Phase 2 eval shows unexplained skew, separate BRD addresses it |
| Retention, deletion, export, redaction policy | BRD-06 | Memory versions retained until BRD-06 defines policy |
| Full allowlist sanitization (DOMPurify) | Future BRD if rich text rendering introduced | FR-4a + FR-13a provide minimum HTML-escaping; Non-Goal documented in BRD-03 |

---

## Verification Commands

```bash
# Sync check must pass
./scripts/check-status-sync.sh

# Architecture fitness checks
./evals/architecture/check-feature-flags.sh
./evals/architecture/check-no-panic.sh
./evals/architecture/check-no-background-context.sh
./evals/architecture/check-no-sensitive-content.sh

# All 5 BRD-03 evals must pass (🔴 → 🟢 for implementation to be complete)
# Integration eval from subdirectory (has its own go.mod):
cd evals/integration && go test -v ./...
```

---

*Production checklist generated by ops profile. Downstream gate: PM/orchestrator after checklist completion. Gate: check-status-sync.sh must exit 0 after the file is created.*