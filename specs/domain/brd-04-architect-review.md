# BRD-04 Pre-Call Briefing — Architect Review
## t_a1202254 — systematic-refinement: BRD-04 Pre-Call Briefing (architect)

**Author:** architect (run 273)
**Date:** 2026-05-23
**Status:** Design review complete — findings for refiner and PM review
**Workspaces:** t_a1202254 (scratch)
**Inputs:** BRD-04 (curated), BRD-02 (curated), BRD-03 (curated), BRD-04 decision-record.md

---

## 1. Executive Summary

BRD-04 Pre-Call Briefing is well-structured and builds correctly on BRD-02 and BRD-03. The async generation model, versioned briefing records, source exclusion mechanism, and evidence policy are all coherent. Four significant design issues are identified requiring decisions before implementation: (A) upcoming meetings data model dependency, (B) async job queue architecture reuse vs. new mechanism, (C) briefing staleness event subscription vs. polling, and (D) briefing generation trigger coupling with memory processing state. Two gaps in BRD-03's API contract are also flagged.

This review does NOT recommend blocking — all issues are addressable within the existing BRD framework. ADRs are proposed for the four significant decisions.

---

## 2. Cross-BRD Dependency Analysis

### 2.1 Hard Dependencies

| Dependency | BRD-04 FR | Integration Point | Status |
|------------|-----------|-------------------|--------|
| BRD-02 meetings table | FR-4, FR-19 | Source meeting lookup for related-meeting matching; ACL enforcement | BRD-02 implemented and graduated |
| BRD-02 participants table | FR-4 | Email-based participant matching (same email signal) | Same |
| BRD-03 memory_versions | FR-4, FR-11 | Briefing aggregates memory content from qualifying prior meetings; quality_status inference | BRD-03 in progress |
| BRD-03 memory_conflicts | FR-11 | Conflicting evidence exclusion from briefing advice | Same |
| BRD-03 briefing_readiness | FR-5, FR-20 | Briefing only generated from meetings with `ready`/`ready_with_weak`; status surfaced in preparation_status | Same |
| BRD-03 evidence statuses | FR-11 | Inherited `strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence` conventions | Same |

### 2.2 Critical Observation: Source ACL Enforcement Chain

**FR-19 (Authorization Boundary)** states: *"Briefings and source details are visible only to users authorized to view the upcoming meeting and all included source meetings. Unauthorized source meetings must not be selected, displayed, logged, or used in generation."*

This creates an **implicit dependency on BRD-02's meeting ACL model**. BRD-02 does not define an explicit meeting-level ACL — it only establishes `createdBy` as the owner. For BRD-04 to enforce FR-19, the ACL model must be extended to cover:
- "Authorized to view" — does this mean meeting owner only, or participants, or org-visible?
- Cross-user source meeting visibility — can user A's briefing include a source meeting owned by user B?

**ADR-0009 is recommended:** Define meeting ACL semantics (owner-only vs. participant-based vs. org-wide) before BRD-04 implementation.

### 2.3 Missing `upcoming_meetings` Table

BRD-04 FR-2 references "upcoming meeting creation" and the data model includes `upcoming_meetings(id)` as a foreign key in `briefing_versions` and `briefing_source_exclusions`. **The `upcoming_meetings` table is never defined in any BRD-02, BRD-03, or BRD-04 document.** The implementation-readiness.md correctly flags this.

**BRD-05 FR-5 defines the table:** BRD-05 (approved) now defines the `upcoming_meetings` table with full CRUD surface (create, list, detail, edit, soft-cancel). BRD-04 can rely on BRD-05 FR-5 for the schema. Key fields: `id` (UUID PK), `title`, `scheduled_start`, `created_by`, `status`, plus indexes. BRD-04 implementation should reference BRD-05 FR-5 for the table definition.

---

## 3. API Contract Assessment — BRD-04

### 3.1 Briefing Endpoints (BRD-04 API Contract)

| Method | Path | Assessment |
|--------|------|------------|
| GET | `/upcoming/{meetingId}/briefing` | OK — active/latest version |
| GET | `/upcoming/{meetingId}/briefing/versions` | OK — version history |
| GET | `/upcoming/{meetingId}/briefing/versions/{versionNumber}` | OK — specific version |
| POST | `/upcoming/{meetingId}/briefing/regenerate` | OK — manual regeneration |
| POST | `/upcoming/{meetingId}/briefing/sources/{sourceId}/exclude` | OK — source exclusion |
| POST | `/upcoming/{meetingId}/briefing/sources/{sourceId}/restore` | OK — exclusion undo |

**Completeness: Sufficient for MVP.** All primary workflows are covered.

### 3.2 BRD-03 Missing Endpoints (Gap Found)

BRD-03 FR-2 (Automatic Processing Queue) specifies that a `memory_processing_jobs` row is inserted after meeting save. However, BRD-03 does not define an endpoint for **listing memory processing job status**, nor does it define an endpoint to **subscribe to job completion events**. BRD-04's briefing generation relies on source meetings being processed — but there is no documented way for the briefing worker to know when a source meeting's processing is complete.

**Gap:** No `GET /meetings/{id}/memory/jobs` or `GET /meetings/{id}/memory/state` endpoint appears in BRD-03's API contract. BRD-04 FR-2's async trigger works for the immediate meeting, but if BRD-04 needs to query whether *other* meetings in the qualifying set are fully processed, that API is not documented.

**Recommendation:** This is likely an implementation detail (the briefing job queue worker can poll `GET /meetings/{id}/memory/state`), but it should be explicitly noted in the integration spec.

### 3.3 API Contract Gap: Briefing Generation Trigger

BRD-04 FR-2 (Automatic Async Generation on Upcoming Meeting Creation) says the system queues a briefing generation job when an upcoming meeting is created. **There is no POST endpoint for this** — the trigger is automatic server-side behavior (side effect of upcoming meeting creation). This is intentional and consistent with BRD-03 (memory processing is also triggered as a side effect of meeting save). No gap — just noting the implicit vs. explicit API distinction.

---

## 4. Data Model Evaluation

### 4.1 briefing_versions Table

| Assessment | Notes |
|-----------|-------|
| Structure | Clean — UUID, FK to upcoming_meetings, monotonically increasing version_number, active/superseded status |
| is_active constraint | Exactly one active per upcoming meeting — good for query performance |
| result field | `ready`, `ready_with_caveats`, `no_prior_memory`, `failed` — maps to FR-20 preparation_status states |
| content JSONB | Full structured output schema defined — includes concise_summary, detailed_sections, sources array |
| trigger_type | `auto` vs `manual_regenerate` — enables analytics on generation source mix |
| Missing: stale_at field | FR-15 says "affected briefing versions are marked stale" but the table has no `stale_at` or `is_stale` column. Detection is likely via source memory version comparison (consistent with BRD-03 FR-10 stale detection), but the staleness *state* on the briefing record is not persisted. If the briefing is stale, how is that flag stored? The `content` JSONB doesn't include a stale flag, and the `status` field only has `active/superseded`. **This is a gap.** |

**ADR-0011 is recommended:** Add `is_stale BOOLEAN DEFAULT FALSE` and `stale_at TIMESTAMPTZ` to `briefing_versions`, or document that stale detection is purely a derived state computed from source memory versions at read time.

### 4.2 briefing_source_exclusions Table

| Assessment | Notes |
|-----------|-------|
| Structure | Clean — per upcoming meeting, per excluded source meeting |
| UNIQUE constraint | Prevents double-exclusion |
| ON DELETE CASCADE | Proper cleanup when upcoming meeting is deleted |
| Missing: restored_at / active boolean | FR-17 says users can undo exclusions. The table has no column to track whether an exclusion is currently active or has been undone. An undo likely means a row delete (since there's a restore endpoint), but this means audit history of exclusions is lost after undo. **Design question for PM/refiner:** Should undos be soft deletes (add `restored_at` column) or hard deletes? |

### 4.3 Briefing JSON Content Schema Assessment

The schema is well-structured with clear separation between `concise_summary` (readable without source clutter) and `detailed_sections` (expandable with full sourcing). Key observations:

1. **Evidence quality propagation:** Quality status values in briefing content are inherited from BRD-03 (`strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence`). BRD-04 FR-11 specifies how these map to display (strong = normal, weak = caveat, insufficient = not a fact, conflicting = excluded from advice). This is coherent.

2. **sources array:** Each source in the `sources` array includes `source_meeting_id`, `relatedness_reasons[]` (human-readable strings), and `quality_status`. This matches FR-6 (relatedness explanations) and FR-12 (source annotations behind details).

3. **no_prior_memory_shell:** When generated, this is a separate top-level key rather than a special case of `concise_summary`. This is the right design — it avoids ambiguity about whether content is real or fabricated.

4. **Missing: version metadata in content.** The `content` JSONB does not include which source memory versions were used. FR-15 (stale detection) says source memory reprocessing makes briefings stale — but if the content doesn't record the source memory version IDs, stale detection must query external state. This is acceptable but slightly opaque for debugging.

---

## 5. Non-Functional Requirements Assessment

### 5.1 Performance

| NFR | Target | Assessment |
|-----|--------|------------|
| Generation latency | P95 < 60s for up to 3 qualifying prior meetings | Consistent with BRD-03 (30s processing) + briefing aggregation overhead. 60s is reasonable. |
| Cached view latency | P95 < 500ms | Tight — requires indexed query on `(upcoming_meeting_id, is_active)` with prepared statement. Achievable. |

### 5.2 Scalability

| NFR | Assessment |
|-----|------------|
| MVP supports up to 3 qualifying source meetings | Bounded scope — no scale concern for MVP |
| Source exclusion state per upcoming meeting | State is per-upcoming-meeting, not global. Scales with upcoming meetings, not source meetings. Reasonable. |

### 5.3 Availability

| NFR | Assessment |
|-----|------------|
| Graceful degradation — cached/latest available during failures | Correctly specified. Consistent with BRD-03 availability model. |
| Regeneration continuity — latest remains readable during regeneration | Correctly specified. No user-facing downtime concern. |

### 5.4 Privacy / Observability

| NFR | Assessment |
|-----|------------|
| No raw transcript/notes/briefing text/PII in logs/metrics | Correctly specified. Same strict redaction as BRD-03. |
| Authorization at meeting ACL level | Depends on ACL model (see ADR-0009 above). |

---

## 6. Identified Design Decisions — ADRs

### ADR-0009: Meeting ACL Semantics for Cross-Meeting Briefing Authorization

**Status:** Proposed
**Component:** Authorization / Cross-BRD Integration

**Problem:** BRD-04 FR-19 requires that briefings and source details are visible only to users authorized to view the upcoming meeting AND all included source meetings. BRD-02 establishes `createdBy` as the meeting owner but does not define a generalized meeting ACL model. The question is: does "authorized to view" mean meeting owner only, or does it include participants, or is it org-wide?

**Options considered:**
- **Option A (Owner-only):** Only the meeting creator (`createdBy`) can access the meeting and its memory/briefing. Simple, consistent with BRD-02. Excludes cross-user briefings where one user's meeting is a source for another user's briefing.
- **Option B (Participant-based):** Meeting participants (from `meeting_participants`) can access the meeting's memory and briefing. Allows cross-user briefings when participants overlap.
- **Option C (Org-wide):** Any authenticated user in the same organization can access any meeting. Simplest for org use cases but most permissive.

**Trade-offs:**
- Option A is safest for privacy but limits briefing to same-user meeting chains.
- Option B requires maintaining a participant→meeting relationship that's meaningful for access control.
- Option C is simplest but may be too permissive for the trust model.

**Recommendation:** Option A (owner-only) for MVP. Cross-user briefings can be addressed in a future BRD when sharing semantics are better understood. FR-19's current phrasing ("authorized to view the upcoming meeting and all included source meetings") is consistent with Option A semantics.

---

### BRD-05 Defines the `upcoming_meetings` Table

**Status:** Resolved
**Component:** Data Model / Cross-BRD Integration

BRD-04 data model references `upcoming_meetings(id)` as a foreign key in `briefing_versions` and `briefing_source_exclusions`. The `upcoming_meetings` table is now defined in **BRD-05 FR-5** — approved and in BRD-05's curated package. BRD-04 implementation should reference BRD-05 FR-5 for the schema definition.

**Minimal required fields for BRD-04 to function (from BRD-05 FR-5):**
- `id` (UUID PK)
- `title` (TEXT)
- `scheduled_start` (TIMESTAMPTZ)
- `created_by` (UUID FK to users)
- `status` enum (`scheduled`, `cancelled`)
- `description` (TEXT, nullable)
- `client_or_organization` (TEXT, nullable)
- `created_at`, `updated_at` (TIMESTAMPTZ)
- Indexes: `idx_upcoming_meetings_owner_status_start` on `(created_by, status, scheduled_start ASC)`, `idx_upcoming_meetings_scheduled_start` on `scheduled_start`

**Authorization:** Follows ADR-0009 owner-only semantics. `created_by` is the owner reference; participants have no access rights.

**Resolved:** The table exists and is defined in BRD-05 FR-5. No further ADR needed for the table definition.

---

### ADR-0011: Briefing Staleness Persistence

**Status:** Proposed
**Component:** Data Model / FR-15 Implementation

**Problem:** BRD-04 FR-15 says "If source meeting memory is reprocessed or changes after a briefing is generated, affected briefing versions are marked stale and regeneration is offered." The `briefing_versions` table has no `is_stale` or `stale_at` column. The staleness detection mechanism (comparing source memory versions) is described conceptually but not persisted in the table schema.

**Options considered:**
- **Option A (Derived at read time):** Staleness is computed at query time by comparing source memory `updated_at` against `briefing_versions.created_at`. No column needed. Trade-off: stale state can't be queried directly, and the briefing content itself doesn't reflect staleness without a read-time computation.
- **Option B (Persist is_stale flag):** Add `is_stale BOOLEAN DEFAULT FALSE` and `stale_at TIMESTAMPTZ` to `briefing_versions`. The briefing worker sets this to TRUE when it detects a source memory version change. Trade-off: requires the worker to re-check staleness after any source memory update.
- **Option C (Event-driven staleness):** BRD-03 emits `memory.source.stale` events when source memory is reprocessed. A briefing staleness subscriber listens for these events and marks affected briefings stale. Requires event subscription infrastructure.

**Recommendation:** Option A (derived at read time) for MVP simplicity, with documentation that the stale indicator is computed, not stored. This is consistent with BRD-03 FR-10 which also derives staleness from `updated_at` comparison. If performance or complexity concerns arise in implementation, Option B can be adopted via ADR revision.

**Important:** The `content` JSONB should include a `generated_from_memory_versions` array (list of source memory version IDs) to make read-time staleness computation deterministic. This is not currently in the schema.

---

### ADR-0012: Source Exclusion Undo — Soft Delete vs. Hard Delete

**Status:** Proposed
**Component:** Data Model / FR-17 Implementation

**Problem:** BRD-04 FR-17 says users can undo an exclusion before regeneration, and the `restore` endpoint (`POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore`) exists. The `briefing_source_exclusions` table has no `restored_at` or `is_active` column — an undo is likely implemented as a row DELETE. This means the audit history of exclusions is permanently lost after undo.

**Options considered:**
- **Option A (Hard delete on undo):** `briefing_source_exclusions` row is deleted on restore. Simple. Audit trail of exclusions is lost after undo.
- **Option B (Soft delete with restored_at):** Add `restored_at TIMESTAMPTZ` column. NULL = active exclusion; set = restored by user. Audit trail preserved. Trade-off: queries must filter `WHERE restored_at IS NULL` to get active exclusions.
- **Option C (History table):** A separate `briefing_source_exclusion_history` table tracks all exclusion/restore events. Cleanest separation but most complex.

**Recommendation:** Option B — add `restored_at TIMESTAMPTZ NULL` to `briefing_source_exclusions`. The unique constraint stays on `(upcoming_meeting_id, excluded_source_meeting_id)` since a source can only be excluded once per upcoming meeting. `restored_at IS NULL` indicates active exclusion; non-NULL indicates restored.

---

## 7. Gaps, Contradictions, and Areas Needing Refiner Attention

### 7.1 Gap: No `upcoming_meetings` Table Definition

The most significant gap. BRD-04 cannot be implemented without the upcoming meetings table. The implementation-readiness.md correctly identifies this. **Action:** PM/PM lead to confirm whether upcoming meetings table is being created in a separate BRD (BRD-05?) or if it exists as part of a different feature.

### 7.2 Gap: Briefing Worker Trigger — When Does It Poll Source Meeting Readiness?

BRD-04 FR-2 queues a briefing generation job when an upcoming meeting is created. The job must:
1. Find qualifying prior meetings (per FR-4 signal matching)
2. Verify each qualifying meeting has memory that is `ready` or `ready_with_weak` (briefing-ready per BRD-03 FR-15)
3. If a source meeting's memory is still processing/queued, what happens?

The BRD does not specify whether:
- Briefing generation waits for all source meetings to complete processing
- Briefing generation proceeds with whatever is ready, marking affected sections as `insufficient_evidence`
- Briefing generation fails and retries if too few source meetings are ready

**This is a significant edge case** that could lead to inconsistent briefing quality. Recommendation: Add a "source readiness timeout" or "minimum ready sources threshold" to FR-4 or FR-5.

### 7.3 Gap: Briefing Generation Cannot Start Until Source Memory Is Ready

This is related to the above. BRD-03's memory processing is async with no guaranteed completion time (P95 = 30s but variable). If BRD-04 relies on memory content from qualifying prior meetings, the briefing job may need to poll or wait for those memories to become ready. The BRD does not specify this dependency graph explicitly.

**Recommendation:** Document a "briefing readiness dependency" — briefing generation for a given upcoming meeting can proceed when all qualifying source meetings have `briefing_readiness.core_categories_ready = true`. If a source meeting is still processing, the briefing can either (a) wait with a timeout, or (b) proceed with reduced context.

### 7.4 Minor Gap: No Briefing Failure Classification

BRD-04 FR-18 specifies safe user-facing failure reasons and retry/regenerate options, but unlike BRD-03, it does not define failure class categories for observability. The NFR mentions metrics and logs for "generation pipeline health" but FR-18 does not specify what those failure classes are.

**Recommendation:** Add FR-18a: Define failure classes for briefing generation (e.g., `source_not_found`, `source_not_ready`, `processing_timeout`, `generation_error`, `partial_failure`). Consistent with BRD-03's `failure_class` label pattern.

### 7.5 Minor Gap: OpenAPI Contract Not Updated

BRD-04 API endpoints are not yet in `contracts/openapi.yaml`. This is explicitly noted as a Phase 1 blocker in BRD-02 and would also block BRD-04 implementation. The OpenAPI update must include:
- `Briefing` schema
- `BriefingVersion` schema
- `BriefingSource` schema
- All 6 briefing endpoints
- `upcoming_meeting` schema (if not already present)

### 7.6 Inconsistency: BRD-04 FR-4 Chronology Signal vs. BRD-03 FR-5

BRD-04 FR-4 defines "close chronology" as *"The prior meeting completed within 90 days before the upcoming meeting's scheduled start."*

BRD-03 FR-5 defines "close chronology" as *"Prior meeting `completed_at` is within 90 days before or 7 days after the current meeting's `completed_at`."*

These are structurally similar but use different anchor dates (upcoming meeting's scheduled start vs. current meeting's completed_at). This is **intentional** — BRD-04 is finding prior context for an *upcoming* meeting, while BRD-03 is finding prior context for a *completed* meeting being processed. However, the terminology overlap is confusing and could lead to implementation errors. **Recommendation:** Add a clarifying note in BRD-04 that the chronology signal is anchored to the upcoming meeting's `scheduled_start`, not the source meeting's `completed_at`.

---

## 8. Summary of Recommendations

| # | Category | Finding | Action |
|---|----------|---------|--------|
| 1 | Authorization | Meeting ACL semantics not defined | ADR-0009: adopt owner-only model for MVP |
| 2 | Data Model | `upcoming_meetings` table undefined | **RESOLVED** — BRD-05 FR-5 defines the table |
| 3 | Staleness | No `is_stale` column in `briefing_versions` | ADR-0011: derive at read-time; document in schema |
| 4 | Source Exclusion | Undo behavior undefined (soft vs hard delete) | ADR-0012: add `restored_at` column for soft delete |
| 5 | Briefing Trigger | Source readiness dependency not documented | Add "source readiness minimum threshold" to FR-5 |
| 6 | Observability | FR-18 missing failure class definitions | Add FR-18a with failure class taxonomy |
| 7 | OpenAPI | Briefing endpoints not in openapi.yaml | Backend: add before Phase 1 implementation |
| 8 | Terminology | BRD-04 vs BRD-03 chronology signal naming | Add clarifying note in BRD-04 FR-4 |

---

## 9. Integration Readiness Verdict

**Overall:** BRD-04 is implementable contingent on three prerequisites being resolved:

1. **`upcoming_meetings` table exists** — confirm or create via separate BRD
2. **BRD-03 memory processing is operational** — briefing generation depends on source memory and briefing-readiness signals
3. **OpenAPI contract updated** — add Briefing schemas and endpoints

The four proposed ADRs are addressable within the existing BRD framework and do not require changes to BRD-04's core architecture. The briefing generation model (async, versioned, source-exclusion-aware) is sound and consistent with BRD-03 patterns.

**No blocking issues.** All findings are design decisions or gaps requiring PM/refiner attention — not structural problems with the BRD.

---

*Architect review completed 2026-05-23. ADRs 0009-0012 proposed for acceptance. Findings routed to refiner and PM for review.*