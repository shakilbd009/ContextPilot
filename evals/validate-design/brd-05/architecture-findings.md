# validate-design: brd-05-manual-upcoming-meeting-creation

**Validator:** validate-design (orchestrator)
**Profile:** validator
**Run:** (this run)
**Date:** 2026-05-24
**Target:** specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md
**Project:** /Users/shakilakram/projects/ContextPilot

---

## Validation Summary

BRD-05 is a Phase 2 feature currently in Deferred state. The spec is well-formed with 8 US, 20 AC, complete feature-flag definition, and full observability matrix. However, no implementation exists — the `upcoming_meetings` and `upcoming_meeting_participants` tables are not created, all API endpoints are absent from OpenAPI, no metrics are emitted, and no feature-flag enforcement code exists in either backend or frontend. This review assesses the architecture as specified versus what would need to be built.

| Area | Verdict | Critical | High | Medium | Low |
|------|---------|----------|------|--------|-----|
| Data Model | **NEEDS_WORK** | 2 | 0 | 0 | 0 |
| API Contract | **BLOCKING** | 1 | 0 | 0 | 0 |
| Feature Flags | PASS | 0 | 0 | 0 | 0 |
| Observability | **NEEDS_WORK** | 0 | 1 | 0 | 0 |
| Transaction Atomicity | **NEEDS_WORK** | 0 | 1 | 0 | 0 |
| Index Coverage | **NEEDS_WORK** | 0 | 1 | 0 | 0 |
| Cross-BRD Consistency | PASS | 0 | 0 | 0 | 0 |

**Overall: NEEDS_WORK** — BRD-05 spec is solid but implementation is zero. Findings below are the architecture checklist that implementation must satisfy.

---

## 1. Data Model vs BRD

**Spec requirement:** FR-5 `upcoming_meetings(id UUID PK, title, scheduled_start, description, client_or_organization, status, created_by UUID, created_at, updated_at)`, FR-6 `upcoming_meeting_participants(id UUID PK, upcoming_meeting_id FK, display_name, email, organization, created_at, updated_at)`

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Migration | **NOT CREATED** | No migration creates `upcoming_meetings` or `upcoming_meeting_participants`. Migration 000005 is a no-op marker resolving a BRD-04/BRD-05 ordering conflict — it does not create the table. |
| Go model | **MISSING** | No Go struct for `UpcomingMeeting` or `UpcomingMeetingParticipant` found in `backend/internal/models/` or equivalent. |
| Repository | **MISSING** | No repository methods for CRUD on `upcoming_meetings` found in codebase. |
| OpenAPI schema | **MISSING** | `contracts/openapi.yaml` contains no `UpcomingMeeting` or `UpcomingMeetingParticipant` schema and no `/upcoming` paths. |
| FK enforcement | **NOT ENFORCED — RESOLVED** | `upcoming_meeting_participants.upcoming_meeting_id` is a plain UUID column with application-level referential integrity, not a DB FK constraint. FR-6 in brd.md now explicitly states this, resolving the prior spec inconsistency. |

### [C1] FR-6 vs plain UUID column — resolved

BRD-05 FR-6: "Data model — `upcoming_meeting_participants` table: `id` UUID PK, `upcoming_meeting_id` UUID (plain column, application-level referential integrity; no DB FK constraint)..."

migration 000004: `upcoming_meeting_id UUID NOT NULL` (plain column, no FK constraint).

**Resolution:** FR-6 now explicitly states "plain UUID column, application-level referential integrity; no DB FK constraint" — consistent with the BRD-05 FR-6 decision. Finding closed.

---

## 2. Transaction Atomicity

**Spec requirement:** FR-17 (BRD-04 trigger on create), FR-18 (BRD-04 trigger on meaningful edit), and the general BRD-05 create/edit flows must atomically persist meeting + participants + trigger enqueue (or safely skip).

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Create transaction scope | **NOT SPECIFIED** | BRD-05 does not explicitly define the transaction boundary for create: is it (meeting + participants) in one tx, with briefing trigger enqueue inside or outside? FR-17 says "queues asynchronously" which implies outside, but the atomicity guarantee is not stated. |
| Edit transaction scope | **NOT SPECIFIED** | FR-18 on meaningful edit: same ambiguity — is the briefing trigger part of the edit transaction or a separate async step? |
| Participant batch insert | **NOT SPECIFIED** | FR-6 says participants are a separate table. Whether create/edit of meeting + all participants is one transaction or multiple is not specified in FRs. |
| Rollback on trigger failure | **NOT SPECIFIED** | If the briefing job enqueue fails (e.g., queue unavailable), FR-17 says "creation does not wait for generation" — does the meeting creation itself roll back, or does the meeting persist and the trigger fail observably? |

### [H1] Transaction boundaries for meeting+participant+trigger not specified

BRD-05 defines the data model but not the transaction semantics. The create flow spans: (1) insert `upcoming_meetings`, (2) insert `upcoming_meeting_participants` batch, (3) enqueue `briefing_processing_jobs` (or skip if conditions not met). These could be one transaction, two transactions, or three separate operations.

**Required action:** Add explicit transaction boundary specification to BRD-05 FRs or an ADR:
- Recommended: meeting + participants in one DB transaction; trigger enqueue as separate async step with its own error handling.
- If trigger enqueue fails, meeting/participants persist; error is logged and `cp_upcoming_meeting_briefing_trigger_failed_total` is incremented.

---

## 3. Flag Parity (Server + Browser Dual Namespace)

**Spec requirement:** FR-1, Feature Flag section (lines 166–178): `ff_enable_upcoming_meetings` with server env `FF_ENABLE_UPCOMING_MEETINGS` and browser env `VITE_FF_ENABLE_UPCOMING_MEETINGS`.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Registry | **PASS** | `specs/feature-flags.md` line 24: `ff_enable_upcoming_meetings` registered with correct server and browser env vars, Phase 2, Deferred. |
| .env.example | **PASS** | `.env.example` lines 26 and 36: both `FF_ENABLE_UPCOMING_MEETINGS=false` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=false` present. |
| Go flag evaluation | **MISSING** | No `FF_ENABLE_UPCOMING_MEETINGS` evaluation found in backend code. No middleware or handler gating on this flag. |
| SvelteKit flag access | **MISSING** | No `$env/static/public` or `$env/dynamic/public` usage of `VITE_FF_ENABLE_UPCOMING_MEETINGS` found in frontend routes or components. |
| Flag-consistent behavior | **NOT ENFORCED** | FR-1 says backend rejects requests with "safe feature-disabled response" when flag is false — no enforcement code exists. |

### [H2] No flag evaluation code exists

Both flag namespaces are registered and present in env files, but no code reads or enforces them. This is expected given BRD-05 is Deferred — but the architecture requires:
- Backend: middleware/handler checks `FF_ENABLE_UPCOMING_MEETINGS` before processing any `/upcoming` request, returns `{"code":"feature_disabled","message":"Upcoming meetings are not available"}` with 200/400 (not 401/403/404/500).
- Frontend: SvelteKit load functions and page components check `VITE_FF_ENABLE_UPCOMING_MEETINGS` to hide/show UI surfaces.

**Required action:** When BRD-05 moves from Deferred to In Dev, implement flag checks in both layers following the pattern used by `FF_ENABLE_MANUAL_MEETING_IMPORT` and `FF_ENABLE_MEETING_MEMORY_PROCESSING` (existing Phase 1 flags).

---

## 4. Index Coverage

**Spec requirement:** FR-5 and FR-6 define the schema; no explicit index requirements are stated in BRD-05 FRs. However, NFR Scale (100 active meetings/user, 50 participants/meeting) implies query performance requirements.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Required indexes | **NOT SPECIFIED** | BRD-05 does not define required indexes. By analogy to BRD-02 (meetings table), `upcoming_meetings` would need: PK on `id`; index on `created_by` for owner-list queries; index on `(created_by, status, scheduled_start)` for default list view sorted by nearest scheduled start. |
| Participants indexes | **NOT SPECIFIED** | `upcoming_meeting_participants` would need: PK on `id`; index on `upcoming_meeting_id` for participant lookup on detail view; composite index on `(upcoming_meeting_id, email)` if participant-email search is needed. |
| Briefing join index | **NOT SPECIFIED** | BRD-04 briefing tables use plain `upcoming_meeting_id` column. For join performance, an index on `briefing_versions(upcoming_meeting_id)` exists (migration 000004 line 41). If `upcoming_meetings` is created, a corresponding index is needed. |

### [H3] Index requirements not defined in BRD-05

BRD-05 FR-5 and FR-6 specify column definitions but not indexes. Given the scale requirements (100 meetings/user, 50 participants/meeting) and the list view sort order (nearest `scheduled_start` first, filtered by `created_by` and `status=scheduled`), the following indexes are architecturally required and should be documented either in BRD-05 or an ADR:

```sql
-- upcoming_meetings
CREATE INDEX idx_upcoming_meetings_owner_status_scheduled
    ON upcoming_meetings(created_by, status, scheduled_start ASC)
    WHERE status = 'scheduled';

-- upcoming_meeting_participants  
CREATE INDEX idx_upcoming_meeting_participants_meeting_id
    ON upcoming_meeting_participants(upcoming_meeting_id);
```

**Required action:** Add index specification to BRD-05 data model section before implementation begins.

---

## 5. API Contract Compliance

**Spec requirement:** FR-2 (full UI surface), FR-11 (owner-only auth), API: POST/GET/PATCH `/upcoming`, GET/PATCH `/upcoming/{id}`, POST `/upcoming/{id}/cancel`. All gated by `ff_enable_upcoming_meetings`.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| OpenAPI | **MISSING** | `contracts/openapi.yaml` contains no `/upcoming` paths, no `UpcomingMeeting` or `UpcomingMeetingParticipant` schemas, no `upcoming_meeting_participants` schema. |
| Backend handlers | **MISSING** | No Go handlers for any `/upcoming` route found in `backend/`. |
| Frontend routes | **PARTIAL** | `frontend/src/routes/` has no `/upcoming`, `/upcoming/new`, `/upcoming/{id}`, `/upcoming/{id}/edit` routes. The existing `frontend/src/routes/api/upcoming/{meetingId}/briefing/*` routes are BRD-04 briefing routes, not BRD-05 upcoming-meeting CRUD. |
| Auth/owner check | **NOT SPECIFIED** | FR-11 says owner-only per ADR-0009. The ADR-0009 pattern (meeting ACL semantics) is defined, but BRD-05 does not specify the exact authorization check implementation (e.g., `UpcomingMeetingExistsAndOwned` query pattern). |

### [C2] All /upcoming endpoints absent from OpenAPI

BRD-05 defines 6 API surface actions (create POST /upcoming, list GET /upcoming, detail GET /upcoming/{id}, update PATCH /upcoming/{id}, cancel POST /upcoming/{id}/cancel, plus the /upcoming/new form route). None exist in `contracts/openapi.yaml`.

By comparison: BRD-04 (also deferred but in validation stage) has the same gap for its 6 briefing endpoints. BRD-02 (Phase 1, in implementation) has full OpenAPI coverage.

**Required action:** Add to `contracts/openapi.yaml` before implementation:
```yaml
/upcoming:
  post:
    operationId: createUpcomingMeeting
    # ... FR-1 flag gating, FR-3 fields, FR-5/FR-6 schema refs
  get:
    operationId: listUpcomingMeetings
/upcoming/{id}:
  get:
    operationId: getUpcomingMeeting
  patch:
    operationId: updateUpcomingMeeting
/upcoming/{id}/cancel:
  post:
    operationId: cancelUpcomingMeeting
```

---

## 6. Observability

**Spec requirement:** Full metrics matrix (lines 117–133) and log events (lines 139–153) covering create/update/cancel/list/detail/briefing-trigger actions with privacy-safe labels.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Metrics | **MISSING** | No `cp_upcoming_meeting_*` metrics found in `backend/internal/briefing/metrics.go` or any other backend metrics file. BRD-04 has `cp_briefing_*` metrics; BRD-05 metrics are not implemented. |
| Histogram | **MISSING** | `cp_upcoming_meeting_action_duration_ms` histogram (FR-29) with actions `create/update/cancel/list/detail/calendar` is not implemented. |
| Briefing trigger metrics | **MISSING** | `cp_upcoming_meeting_briefing_trigger_queued_total`, `cp_upcoming_meeting_briefing_trigger_skipped_total`, `cp_upcoming_meeting_briefing_trigger_failed_total`, and enqueue duration histogram are not implemented. |
| Log events | **MISSING** | No `upcoming_meeting.*` log events found in codebase. |
| Privacy enforcement | **NOT VERIFIABLE** | No implementation exists to verify that metric labels and log fields contain no raw meeting titles, descriptions, participant PII, or organization values (FR-20/NFR privacy). |

### [H4] Full observability stack not implemented

BRD-05's observability matrix specifies 14 counter metrics, 1 histogram, 12 log event types, and privacy requirements — none of which are implemented. This is expected for a Deferred feature but is a significant implementation requirement when BRD-05 activates.

**Required action:** When BRD-05 moves to In Dev, implement full metrics using the pattern from BRD-04 (`backend/internal/briefing/metrics.go` shows `cp_briefing_*` metrics — BRD-05 should follow the same `cp_upcoming_meeting_*` naming convention). Key implementation requirements:
- All 14 counters with controlled low-cardinality enum labels (no raw PII)
- Histogram with `action` and `result` labels
- 12 log event types with hashed/opaque identifiers
- Privacy audit: confirm metrics/logs contain zero raw user content

---

## Positive Findings (No Changes Needed)

| Area | Finding |
|------|---------|
| **Flag registry** | `ff_enable_upcoming_meetings` correctly registered in `specs/feature-flags.md` with both `FF_ENABLE_UPCOMING_MEETINGS` and `VITE_FF_ENABLE_UPCOMING_MEETINGS`, default false, Phase 2, Approved. |
| **.env.example** | Both flag namespaces present in `.env.example` lines 26 and 36. |
| **ADR alignment** | Application-level referential integrity correctly applied. Plain UUID column with no DB FK constraint per FR-6. Briefing tables self-contained regardless of BRD-05 state. |
| **Cross-BRD dependency model** | BRD-05 correctly documents that BRD-04 trigger requires both `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true` (FR-17, FR-18, FR-19). The dual-flag dependency is explicit. |
| **Feature flag default** | Flag defaults false in all locations — registry, .env.example, spec. Consistent with AGENTS.md dual-namespace rule. |
| **Status enum** | FR-7 specifies `scheduled` and `cancelled` as minimum status values. This is sufficient for MVP. |
| **Scale NFR** | NFR scale target (100 active/user, 50 participants/meeting) is clearly stated, enabling index and capacity planning. |
| **Eval files** | All three eval files (e2e, unit, integration) exist and are marked 🔴 Failing with implementation-pending status. Eval coverage is complete. |
| **Spec completeness** | 8 US, 20 AC, 32 FRs, full NFR table, complete feature flag spec, full observability matrix, explicit scope boundaries, and explicit deferred items list. |

---

## Summary of Required Actions

| ID | Severity | Area | Action |
|----|----------|------|--------|
| C1 | Informational | Data Model | **RESOLVED** — FR-6 now states "plain UUID column, application-level referential integrity; no DB FK constraint". BRD-05 FR-6 decision correctly applied. |
| C2 | Critical | API Contract | Add all `/upcoming` CRUD endpoints to `contracts/openapi.yaml` before implementation (same gap as BRD-04 had). |
| H1 | High | Transaction Atomicity | Specify transaction boundary for meeting+participants+trigger in FRs or ADR (recommended: meeting+participants in one DB tx, trigger as separate async step). |
| H2 | High | Flag Parity | Implement `FF_ENABLE_UPCOMING_MEETINGS` checks in backend handlers and `VITE_FF_ENABLE_UPCOMING_MEETINGS` in SvelteKit components when BRD-05 activates (following existing Phase 1 flag patterns). |
| H3 | High | Index Coverage | Add explicit index specification to BRD-05 data model section (at minimum: `created_by`-filtered scheduled_start index, `upcoming_meeting_id` index on participants). |
| H4 | High | Observability | Implement full `cp_upcoming_meeting_*` metrics and `upcoming_meeting.*` log events when BRD-05 activates (14 counters, 1 histogram, 12 log types, privacy audit). |

---

## Dependencies and Blockers

| Blocker | Severity | Must Resolve Before |
|---------|----------|---------------------|
| OpenAPI /upcoming endpoints missing | Critical | BRD-05 implementation start |
| FR-6 FK/plain UUID inconsistency | **RESOLVED** | BRD-05 implementation start |
| Index specification missing | High | BRD-05 implementation start |
| Transaction boundary not specified | High | BRD-05 implementation start |
| Observability stack not implemented | High | BRD-05 implementation start |
| Flag enforcement code missing | High | BRD-05 implementation start (when Deferred → In Dev) |

---

*Architecture findings only — implementation pending. BRD-05 spec quality is high; findings represent the checklist implementation must satisfy.*
