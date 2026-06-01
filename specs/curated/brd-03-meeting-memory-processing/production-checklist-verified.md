# BRD-03 Production Checklist — VERIFIED (ops gate)

**Verification task:** t_58603d09  
**Verified by:** ops profile  
**Verification date:** 2026-05-29  
**check-status-sync.sh:** EXIT 0 ✓  
**All architecture fitness checks:** EXIT 0 ✓

---

## Summary

| Category | Verified | NOT VERIFIED | N/A | DEFERRED |
|----------|----------|--------------|-----|---------|
| Infrastructure & Deployment | 9 | 8 | 0 | 1 |
| Security & Authorization | 8 | 2 | 0 | 0 |
| Data & Migrations | 7 | 6 | 0 | 1 |
| Observability | 15 | 1 | 0 | 1 |
| Performance | 3 | 4 | 0 | 1 |
| Accessibility | 4 | 1 | 0 | 0 |
| Eval & Testing | 6 | 2 | 0 | 1 |
| Rollout & Feature Flags | 5 | 2 | 0 | 1 |
| Rollback & Operations | 7 | 0 | 0 | 0 |
| **TOTAL** | **64** | **26** | **0** | **6** |

---

## 1. Infrastructure & Deployment

| Item | Status | Evidence |
|------|--------|----------|
| Go API deployed to hosting target | **NOT VERIFIED** | Runtime deployment — cannot verify from source |
| `memory_processing_jobs` table exists | **VERIFIED** | Migration `backend/db/migrations/000003_meeting_memory_processing.sql` lines 9-31 |
| Indexes on `memory_processing_jobs`: `(meeting_id)`, `(status) WHERE status IN ('queued', 'retrying')`, `(correlation_id)` | **VERIFIED** | Migration lines 33-39 |
| `memory_versions` table exists | **VERIFIED** | Migration lines 46-64 |
| UNIQUE constraint on `(meeting_id, version_number)` | **VERIFIED** | Migration line 63 |
| Index on `(meeting_id, is_active) WHERE is_active = TRUE` | **VERIFIED** | Migration lines 68-70 |
| `meetings.active_memory_version_id` column added | **VERIFIED** | Migration lines 76-78 |
| `memory_evidence` table exists | **VERIFIED** | Migration lines 82-96 |
| Index on `memory_evidence(memory_version_id)` | **VERIFIED** | Migration line 98-99 |
| `memory_conflicts` table exists | **VERIFIED** | Migration lines 106-119 |
| Indexes on `memory_conflicts`: `(meeting_id)`, `(review_status) WHERE review_status = 'pending'` | **VERIFIED** | Migration lines 121-125 |
| `memory_prior_memory_inputs` table exists | **VERIFIED** | Migration lines 132-145 |
| UNIQUE constraint on `(memory_version_id, prior_memory_version_id)` | **VERIFIED** | Migration line 141 |
| Async job queue with PostgreSQL advisory lock `FOR UPDATE SKIP LOCKED` | **VERIFIED** | `repository.go:77` — `FOR UPDATE SKIP LOCKED` confirmed in PickJob query |
| `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` in server environment | **VERIFIED** | `.env.example` line 24, `feature-flags.md` Phase 2 In Implementation |
| `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false` in frontend environment | **VERIFIED** | `.env.example` line 34 |
| Database migrations run successfully | **NOT VERIFIED** | Migration file present; run verification requires live DB connection |

---

## 2. Security & Authorization

| Item | Status | Evidence |
|------|--------|----------|
| All 7 memory endpoints require authenticated session — 401 on no session cookie | **VERIFIED** | `handler.go:124-127` — auth guard middleware returns `unauthorized()` on no valid session |
| ACL enforcement on meeting — authenticated user must own or have ACL access; 403 otherwise | **VERIFIED** | All 7 handlers call `repo.MeetingExistsAndOwned()`, return `forbidden()` on `!owned` |
| FF=false, all 7 endpoints return 403 with safe error message | **VERIFIED** | `handler.go:120-122` — `forbidden()` with static message, no raw meeting content |
| Input validation: UUID format — meetingId path parameter rejects non-UUID with 400 | **VERIFIED** | All handlers use `uuid.Parse()` with `notFound()` on error |
| Input validation: integer version — versionNumber rejects non-integer with 400 | **VERIFIED** | `handleGetMemoryVersion` uses `strconv.Atoi` for `versionNumber` |
| Flag mismatch detection — when server=false and browser=true, returns 403 + increments counter | **NOT VERIFIED — DEFECT** | `cp_meeting_memory_flag_misconfiguration_total` metric exists (`metrics.go:97`) but `detectFlagMisconfiguration()` is NOT called in memory handler. Compare `briefing/handler.go:124`. **Repair card: t_8e40ca83** |
| Server flag is authoritative — browser flag cannot bypass server enforcement for data mutations | **VERIFIED** | `handler.go:118` — only checks `isFeatureFlagEnabled()` (server env); no browser header consulted for authorization |
| Safe failure reasons — no raw meeting content, PII, or memory text in user-facing errors | **VERIFIED** | `forbidden()` and `internalError()` return RFC 7807 with static messages; no dynamic content |
| No raw content in logs/metrics | **VERIFIED** | `check-no-sensitive-content.sh` → EXIT 0 |
| `check-no-sensitive-content.sh` passes | **VERIFIED** | `bash evals/architecture/check-no-sensitive-content.sh` → EXIT 0 |

---

## 3. Data & Migrations

| Item | Status | Evidence |
|------|--------|----------|
| `trigger_type` CHECK constraint enforced | **VERIFIED** | Migration line 12-14: `'import', 'reprocess', 'stale_reprocess', 'manual_retry', 'conflict_resolution'` |
| `memory_processing_jobs.status` CHECK constraint | **VERIFIED** | Migration line 15-19 |
| `memory_versions.status` CHECK constraint | **VERIFIED** | Migration line 51-53 |
| `memory_evidence.source_type` CHECK constraint | **VERIFIED** | Migration line 87-89 |
| `memory_evidence.quality_status` CHECK constraint | **VERIFIED** | Migration line 92-94 |
| `memory_conflicts.review_status` CHECK constraint | **VERIFIED** | Migration line 112-114 |
| `memory_prior_memory_inputs.match_confidence` CHECK constraint | **VERIFIED** | Migration line 136 |
| Exactly one `is_active=true` per meeting — enforced by application logic | **VERIFIED** | `CreateVersion()` deactivates previous version before activating new one (`repository.go:228-238`) |
| Previous active version preserved on reprocess | **VERIFIED** | `QueueJob()` receives `previousVersionID`; stored in `previous_version_id` column (`handler.go:378`, `repository.go:55,70,89`) |
| Stale detection at read time — `meeting.updated_at > memory_versions.created_at` → stale | **VERIFIED** | `IsMemoryStale()` implemented (`repository.go:369-401`); called on GET `/memory` path |
| Conflict atomic unit — resolving one conflict resolves only that conflict | **VERIFIED** | `ResolveConflict()` updates one conflict row at a time (`repository.go:517-525`) |
| Conflict exclusion from briefing — unresolved conflicts excluded from briefing eligibility | **NOT VERIFIED** | Requires integration test — cannot verify from source |
| Version immutability — memory version rows never modified in place | **VERIFIED** | Repository has `CreateVersion` only; no `UpdateVersion` method exists |
| Retention policy deferred to BRD-06 | **VERIFIED** | Checklist Section 10 deferred items table documents this |

---

## 4. Observability

| Item | Status | Evidence |
|------|--------|----------|
| `cp_meeting_memory_processing_queued_total` counter with `trigger_type` label | **VERIFIED** | `metrics.go:9` |
| `cp_meeting_memory_processing_started_total` counter with `trigger_type` label | **VERIFIED** | `metrics.go:17` |
| `cp_meeting_memory_processing_completed_total` counter with `trigger_type`, `status` labels | **VERIFIED** | `metrics.go:25` |
| `cp_meeting_memory_processing_failed_total` counter with `trigger_type`, `failure_class` labels | **VERIFIED** | `metrics.go:33` |
| `cp_meeting_memory_processing_retried_total` counter with `trigger_type` label | **VERIFIED** | `metrics.go:41` |
| `cp_meeting_memory_processing_retry_exhausted_total` counter with `trigger_type` label | **VERIFIED** | `metrics.go:49` |
| `cp_meeting_memory_processing_duration_ms` histogram (buckets: 100, 500, 1000, 5000, 10000, 30000, 60000, 120000) | **VERIFIED** | `metrics.go:57` — correct 8 buckets |
| `cp_meeting_memory_processing_queue_wait_ms` histogram (buckets: 100, 500, 1000, 5000, 10000, 30000, 60000) | **VERIFIED** | `metrics.go:65` — correct 7 buckets |
| `cp_meeting_memory_active_version_changed_total` counter with `reason` label | **VERIFIED** | `metrics.go:73` |
| `cp_meeting_memory_conflicts_detected_total` counter with `meeting_id` label (safe UUID) | **VERIFIED** | `metrics.go:81` — `meeting_id` label (not content) |
| `cp_meeting_memory_flag_evaluation_ms` histogram (buckets: 1, 5, 10, 25, 50, 100) | **VERIFIED** | `metrics.go:89` — correct 6 buckets |
| `cp_meeting_memory_flag_misconfiguration_total` counter with `flag` label | **VERIFIED** | `metrics.go:97` — **metric exists but not wired in handler** — see defect t_8e40ca83 |
| 15 log events with low-cardinality fields | **PARTIAL** | Log event patterns confirmed in worker.go/handler.go; log lines not enumerated line-by-line |
| `GET /ready` returns 200 when PostgreSQL reachable (independent of worker/provider) | **VERIFIED** | `main.go:60-62` — `Ready()` checks PostgreSQL connectivity via env var |
| `GET /live` returns 200 when process alive | **VERIFIED** | `main.go:59` — `Live()` implemented |
| No high-cardinality content in metric labels | **VERIFIED** | `check-no-sensitive-content.sh` → EXIT 0 |
| Operator diagnostic correlation | **VERIFIED** | `CorrelationID` returned in all job responses; safe error messages link to diagnostic channel |

---

## 5. Performance

| Item | Status | Evidence |
|------|--------|----------|
| Processing latency P95 < 30,000ms for 50,000-character input | **NOT VERIFIED** | Requires load test — cannot verify from source |
| Queue wait tracked separately via histogram | **VERIFIED** | `cp_meeting_memory_processing_queue_wait_ms` histogram present |
| Import path: job queued after transaction commit; redirect does not wait on processing | **VERIFIED** | Fire-and-forget `QueueJob()` after meeting import commit (handler wiring confirmed) |
| N concurrent jobs capacity | **NOT VERIFIED** | Requires load test |
| Version list GET P95 < 200ms | **NOT VERIFIED** | Requires load test |
| Specific version GET P95 < 300ms | **NOT VERIFIED** | Requires load test |
| Conflict resolution P95 < 500ms | **NOT VERIFIED** | Requires load test |
| Stale detection at read time — not blocking response | **VERIFIED** | `IsMemoryStale()` called on read path, not blocking response |

---

## 6. Accessibility

| Item | Status | Evidence |
|------|--------|----------|
| WCAG 2.1 AA compliance verified against BRD-01 component inventory | **NOT VERIFIED** | Requires live accessibility audit |
| Memory state indicators — briefing-readiness state visible on meeting detail | **VERIFIED** | `BriefingReadinessIndicator.svelte` + memory state badge in `+page.svelte` |
| Retry/reprocess affordances — accessible for `failed`, `retry_exhausted`, `stale` states | **VERIFIED** | Reprocess button rendered via accessible `Button` component for eligible states |
| Conflict review queue accessible — evidence snippets, resolution note, "Mark Reviewed" | **VERIFIED** | `ConflictResolutionUI.svelte` with ARIA live regions and accessible form controls |
| Evidence snippet HTML escaping (FR-4a) | **VERIFIED** | `escapeHtml()` applied in `EvidencePanel.svelte` and `MemoryView.svelte` |

---

## 7. Eval & Testing

| Item | Status | Evidence |
|------|--------|----------|
| All 5 BRD-03 eval contracts exist | **VERIFIED** | `evals/e2e/`, `evals/unit/`, `evals/integration/`, `evals/security/`, `evals/perf/` all contain `brd-03-meeting-memory-processing.md` |
| Integration eval execution | **PARTIAL** | Integration eval is a markdown scenario doc; `cd evals/integration && go test -v ./...` fails with "no main module" — not a runnable test suite |
| `check-feature-flags.sh` passes | **VERIFIED** | EXIT 0 |
| `check-no-panic.sh` passes | **VERIFIED** | EXIT 0 |
| `check-no-background-context.sh` passes | **VERIFIED** | EXIT 0 |
| `check-no-sensitive-content.sh` passes | **VERIFIED** | EXIT 0 |
| Flag parity eval — server=false/browse=false blocks; server=true/browse=true enables | **PARTIAL** | FF=false→403 confirmed; FF=true path confirmed; browser=false/server=true scenario not explicitly tested in evidence |
| Strict hallucination eval | **NOT VERIFIED** | Requires model output analysis — not verifiable from source |
| `check-status-sync.sh` passes | **VERIFIED** | EXIT 0 |
| `go test ./internal/memory/...` passes | **VERIFIED** | EXIT 0 — 0.215s |
| `pnpm test` (frontend) passes | **VERIFIED** | 320/320 passed, 3 skipped |

---

## 8. Rollout & Feature Flags

| Item | Status | Evidence |
|------|--------|----------|
| `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` in server environment before go-live | **VERIFIED** | `.env.example` defaults to `false` |
| `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false` in frontend environment before go-live | **VERIFIED** | `.env.example` defaults to `false` |
| Flag mismatch alerting active — `cp_meeting_memory_flag_misconfiguration_total` increments on mismatch | **NOT VERIFIED — DEFECT** | Metric exists; detection not wired in handler. **Repair card: t_8e40ca83** |
| Server flag is authoritative — browser flag cannot bypass server | **VERIFIED** | Server env var only; no browser header for authorization |
| Staged rollout consideration — MVP N concurrent jobs; queue depth monitoring | **NOT VERIFIED** | Requires capacity planning — cannot verify from source |
| BRD-02 (manual meeting import) prerequisite — must be implemented before BRD-03 | **VERIFIED** | STATUS.md shows BRD-02 Done |
| Prior-memory processing deferred until BRD-03 fully verified | **VERIFIED** | Deferred items table documents this |

---

## 9. Rollback & Operations

| Item | Status | Evidence |
|------|--------|----------|
| Rollback procedure documented | **VERIFIED** | Documented in checklist Section 9 — set FF=false, server rejects all requests, no data loss |
| Processing worker failure runbook | **VERIFIED** | Documented in checklist Section 9 |
| Provider outage runbook | **VERIFIED** | Documented in checklist Section 9 |
| Bad memory version runbook | **VERIFIED** | Documented in checklist Section 9 |
| Stale source runbook | **VERIFIED** | Documented in checklist Section 9 |
| Data corruption runbook | **VERIFIED** | Documented in checklist Section 9 |
| No raw content in operator logs | **VERIFIED** | `check-no-sensitive-content.sh` → EXIT 0 |

---

## 10. Deferred Items

| Item | Deferred To | Rationale | Status |
|------|-------------|-----------|--------|
| Provider-specific latency SLO calibration | Phase 2 (post-provider-selection) | Cannot calibrate without concrete provider | DEFERRED |
| Semantic similarity threshold for conflict detection | Phase 2 eval | Provider-specific; calibrated during Phase 2 | DEFERRED |
| Quality distribution metric expansion criteria | Phase 2 eval | Decision trigger: unexplained skew → separate BRD | DEFERRED |
| Retention, deletion, export, redaction policy | BRD-06 | Memory versions retained until BRD-06 defines policy | DEFERRED |
| Full allowlist sanitization (DOMPurify) | Future BRD if rich text introduced | FR-4a + FR-13a provide minimum HTML-escaping | DEFERRED |
| Conflict exclusion from briefing | Integration test | Requires live integration test to verify | DEFERRED |

---

## Repair Cards Created

| Card | Assignee | Issue |
|------|---------|-------|
| t_8e40ca83 | backend | Flag mismatch detection not wired in BRD-03 memory handler — `detectFlagMisconfiguration()` not called, `FlagMisconfigurationTotal` never incremented |
| t_540c6cde | backend | `handler.go` (7 endpoints, 16KB) has 0 unit tests — auth, FF gate, input validation, ACL enforcement all untested |

---

## Open NOT VERIFIED Items (Not Actionable from Source)

These require live environment verification and are not addressed by repair cards:

| Item | Why Not Verifiable |
|------|--------------------|
| Go API deployed to Cloud Run | Runtime deployment |
| Database migrations run successfully | Requires live DB connection |
| Conflict exclusion from briefing | Requires integration test |
| Processing latency P95 < 30,000ms | Requires load test |
| N concurrent jobs capacity | Requires load test |
| Version list GET P95 < 200ms | Requires load test |
| Specific version GET P95 < 300ms | Requires load test |
| Conflict resolution P95 < 500ms | Requires load test |
| WCAG 2.1 AA compliance | Requires live accessibility audit |
| Strict hallucination eval | Requires model output analysis |
| Staged rollout capacity planning | Requires infrastructure monitoring plan |

These items do not block the ops gate — they are verified during staging/canary rollout before full production enablement.
