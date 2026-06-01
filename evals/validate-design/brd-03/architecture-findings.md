# Architecture Findings — BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validator:** architecture
**Target:** specs/domain/brd-03-meeting-memory-processing.md
**ADRs reviewed:** ADR-0005 (queue), ADR-0006 (data model), ADR-0007 (prior memory matching + conflict detection), ADR-0008 (observability + privacy)

---

## Verdict: NEEDS_ATTENTION

---

## Critical Issues (Must Fix)

### [C1] Processing latency target is deferred to "curation" — BRD-03 cannot be implemented without a concrete SLO

**Source:** BRD-03 Open Question 2, NFR table
**BRD section:** NFR table row: "Processing latency | Architecture-defined during curation"

The NFR table says processing latency is "Architecture-defined during curation." Open Question 2 (OQ-2) says "curation must set a concrete processing latency target after model/provider/runtime architecture is selected." This means the BRD does not contain an implementable latency requirement. Implementation cannot begin until this is resolved.

**Impact:** Engineers cannot write performance tests, set histogram bucket boundaries, or define SLO thresholds without a latency target. The observability section defines `cp_meeting_memory_processing_duration_ms` histogram but has no bucket definitions.

**Fix required:** Before Phase 2 implementation begins, set a concrete processing latency target (e.g., p95 < 30s for 95th percentile of meetings under 50k characters). This requires a curation decision, not an architectural one.

**Note:** This is a curation blocker, not an implementation gap. The task body says "BRD-03 Meeting Memory Processing" is the target. This finding should be surfaced to the PM for a latency SLO decision.

---

## High Priority Issues (Should Fix)

### [H1] `memory_conflicts.conflicting_items` JSONB contains raw memory content — potential redaction hazard

**Source:** ADR-0006, memory_conflicts table schema
**ADR:** ADR-0006

The `memory_conflicts.conflicting_items` column is defined as `JSONB NOT NULL -- {current: [...], prior: [...]}`. The ADR does not specify whether these items contain the full memory text or just IDs/references to `memory_versions.content`. If this JSONB column stores actual decision statements, action item descriptions, or summary text, those would be "generated memory content" banned from logging — but this field is database storage, not logs.

**Impact:** If the conflicting_items JSONB stores full content, any DB read of `memory_conflicts` is equivalent to exposing generated memory text in a queryable table. This could violate the BRD's privacy requirements (FR-25) if the table is accessible to operators or exported in backups.

**Fix required:** Clarify in ADR-0006 whether `conflicting_items` stores full item content or stable item IDs that reference back to `memory_versions.content`. If it stores content, document the privacy rationale and access controls explicitly.

---

### [H2] `source_location` JSONB format not specified — multiple incompatible formats possible

**Source:** ADR-0006 schema, BRD-03 OQ-3
**ADR:** ADR-0006

The `source_location` column is defined as `JSONB NOT NULL -- e.g., {"type": "char_offset", "start": 1234, "end": 5678}` with the comment "format is flexible (char offset, line ref, etc.)". The BRD defers this to OQ-3 ("curation must choose character offsets, line references, or another stable locator"). ADR-0006 says the application layer validates format per source_type.

**Impact:** Without a committed format, two implementations of the processor could produce incompatible `source_location` values. Evals (AC-5, AC-7) cannot be written against a stable format. Integration tests and evidence-grounding evals require a deterministic format.

**Fix required:** Commit to a source location format in ADR-0006. OQ-3 is an open question, meaning this decision has not been made. Until it is, the data model is incomplete for implementation.

---

### [H3] `check-no-sensitive-content.sh` architecture eval referenced in ADR-0008 but does not exist

**Source:** ADR-0008 Consequences section, item 6
**ADR:** ADR-0008

ADR-0008 says: "The `check-no-sensitive-content.sh` architecture eval (to be written) validates that `Redactor` is called on all string fields from `transcript`, `notes`, `evidence_snippet`, `stakeholder_note`, `generated_memory_content`, `prior_memory_content`, and `resolution_note` before those fields can be used in log or metric emission."

This eval does not exist in `evals/architecture/`.

**Impact:** The privacy enforcement described in ADR-0008 has no automated verification. The architecture fitness function that would catch accidental logging of sensitive content is not written and is not in the eval coverage.

**Fix required:** Write `evals/architecture/check-no-sensitive-content.md` and `evals/architecture/check-no-sensitive-content.sh` as part of Phase 2 implementation readiness. This is a prerequisite before production enablement.

---

### [H4] ADR-0005 and ADR-0006 are "Proposed" not "Accepted" — queue and data model not committed

**Source:** ADR-0005, ADR-0006, ADR-0007, ADR-0008 headers
**ADR:** All four ADRs

All four ADRs for BRD-03 are marked "Proposed." For Phase 2 implementation to proceed, these must be accepted so the data model, queue architecture, matching strategy, and observability patterns are committed.

**Fix required:** Accept ADRs-0005 through 0008 before Phase 2 implementation begins. Each ADR should have its status updated from "Proposed" to "Accepted" with an acceptance date.

---

## Medium Priority Issues

### [M1] `briefing_readiness.core_categories_ready` logic is underspecified for multi-category conflicts

**Source:** ADR-0006 Memory JSON Schema, ADR-0007 Consequences section
**ADR:** ADR-0006, ADR-0007

The `briefing_readiness` object in the memory JSON has `blocking_conflicts: []`. The BRD (FR-15, AC-20) says unresolved conflicts block briefing readiness. ADR-0007 says "blocking conflicts" means the categories that have unresolved conflicts. But the logic for determining which categories are blocked is not defined in either ADR.

Specifically: if the `summary` category has a conflict, does that block all of briefing readiness, or only the `summary` portion of the briefing? AC-20 says "no unresolved conflict blocks those core categories" — but the schema structure in ADR-0006 has `blocking_conflicts` as a flat array with no category association.

**Fix required:** Specify in ADR-0006 or ADR-0007 how `blocking_conflicts` is structured and how it is computed. The schema should be updated to `[{category: string, conflict_id: uuid}]` or equivalent.

---

### [M2] Semantic similarity threshold for conflict detection is not specified

**Source:** ADR-0007 Conflict Detection section
**ADR:** ADR-0007

ADR-0007 says conflict detection uses "semantic similarity above a threshold" to determine whether items agree or conflict. The threshold is not specified. This is a key parameter that determines the false-positive and false-negative rate of conflict detection.

**Fix required:** Specify the similarity threshold (e.g., cosine distance < 0.85 = conflict, >= 0.85 = agree) or document it as a tunable runtime parameter with a recommended default.

---

### [M3] No OpenAPI endpoint definitions for memory processing, retry, reprocess, or conflict resolution

**Source:** BRD-03 FRs (US-1 through US-8) — endpoints implied but not defined
**BRD section:** Functional Requirements

BRD-03 describes the following user-facing actions: manual retry/reprocess (FR-2), prior-memory include/exclude and reprocess (FR-12, AC-16), conflict resolution (FR-14, AC-19), and review queue access (FR-6, AC-18). The API endpoints for these are not specified in the BRD. No route paths, request/response shapes, or HTTP status codes are defined.

**Impact:** Backend implementation cannot begin without API contracts. The OpenAPI contract gap here is analogous to the BRD-02 issue flagged as C1 in the previous validation.

**Fix required:** Add API endpoint definitions to BRD-03 or a companion spec: at minimum `POST /meetings/:id/memory/retry`, `POST /meetings/:id/memory/reprocess`, `GET /meetings/:id/memory/conflicts`, `POST /meetings/:id/memory/conflicts/:conflict_id/resolve`, `GET /meetings/:id/memory/versions`.

---

### [M4] `is_active` + `status` dual-column design has no defined atomicity constraint

**Source:** ADR-0006, `is_active` flag vs `status` column
**ADR:** ADR-0006

ADR-0006 uses both `is_active: BOOLEAN` and `status: TEXT IN ('active', 'superseded', 'conflict_review')`. The ADR says "the previous active version is atomically deactivated" using a transaction. But the unique constraint is only on `(meeting_id, version_number)`, not on `(meeting_id, is_active)`. This means two rows for the same meeting could have `is_active = TRUE` simultaneously if an application bug issues two concurrent UPDATE statements.

**Fix required:** Add a partial unique index `UNIQUE (meeting_id) WHERE is_active = TRUE` to prevent concurrent activation, or add an exclusion constraint `USING gist (meeting_id WITH =, is_active WITH =)` if supported. Document the atomic activation procedure in ADR-0006.

---

## Low Priority Issues

### [L1] `meeting_memory_processing_jobs` lacks a `lock_Acquired_at` field — crash recovery gap

**Source:** ADR-0005 Worker Acquisition section
**ADR:** ADR-0005

ADR-0005 describes `FOR UPDATE SKIP LOCKED` acquisition but does not specify what happens if a worker crashes after acquiring a lock but before completing processing. The job stays in `status='processing'` forever (no timeout). The ADR mentions "transactionally writing the memory version before updating job status" as mitigation, but there's no `locked_at` or `lock_expires_at` column to detect stale in-flight jobs.

**Fix required:** Add a `locked_at TIMESTAMPTZ` column and a timeout mechanism (e.g., `lock_expires_at = now() + 5 minutes`). A recovery process can then detect stale locks and return the job to `queued`.

---

### [L2] ADR-0007's `uncertain` match confidence allows user-included memories without title similarity

**Source:** ADR-0007 Prior Memory Matching section
**ADR:** ADR-0007

The matching algorithm defines `uncertain` match as: same_participant AND close_chronology pass, but title similarity fails. An `uncertain` match is shown to the user for confirmation. However, if the user confirms (includes) an uncertain match, it gets `included_by_user = TRUE` in `memory_prior_memory_inputs` and participates in processing.

**Issue:** FR-12 says matching must use "constrained signals" (same participants, similar title, close chronology). If a user can include a prior memory that failed the title similarity signal, the "constrained" constraint is bypassable at user discretion. This may be intentional but is worth explicitly documenting.

**Recommendation:** Document this as an intentional relaxation of the constrained matching signal, or remove `uncertain` as a separate confidence level and treat all user-inclusions as a distinct category.

---

## Positive Findings

- ADR-0005 queue architecture using PostgreSQL advisory locks is sound and Phase-appropriate (Phase 2, no new infra dependency)
- Non-blocking import trigger (fire-and-forget INSERT after transaction commit) correctly satisfies AC-2 and the NFR on import path latency
- Hybrid JSONB + normalized table data model is the right balance for versioned memory with queryable evidence/conflict records
- `MemoryProcessor` interface abstraction allows future migration to BullMQ/Temporal without changing processing logic
- `PriorMemoryMatcher` interface allows swapping matching strategies
- ADR-0007 constrained matching (three-signal AND) is correctly scoped to BRD-03 and defers broad opaque matching to a future BRD
- Conflict isolation in separate `memory_conflicts` table with `review_status='pending'` is a clean exclusion pattern
- Active version activation pattern (atomically deactivate prior + insert new) is correctly specified
- Stale detection logic (compare meeting.updated_at vs memory_versions.created_at) is correctly described
- ADR-0008 ProcessingResult struct as the sole emission boundary is architecturally correct

---

## Summary Table

| ID | Severity | Issue |
|----|----------|-------|
| C1 | Critical | Processing latency target deferred to curation — BRD unimplementable without SLO |
| H1 | High | memory_conflicts.conflicting_items JSONB may store raw memory content |
| H2 | High | source_location format not committed — multiple incompatible formats possible |
| H3 | High | check-no-sensitive-content.sh eval does not exist |
| H4 | High | All four BRD-03 ADRs are "Proposed" not "Accepted" |
| M1 | Medium | blocking_conflicts structure underspecified for multi-category conflicts |
| M2 | Medium | Semantic similarity threshold for conflict detection not specified |
| M3 | Medium | No OpenAPI endpoint definitions for memory processing/retry/conflict APIs |
| M4 | Medium | is_active + status dual-column has no atomicity constraint (concurrent activation possible) |
| L1 | Low | memory_processing_jobs lacks lock timeout for crash recovery |
| L2 | Low | uncertain match confidence allows bypassing constrained matching signal |