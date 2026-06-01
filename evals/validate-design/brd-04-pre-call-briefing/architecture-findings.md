# Architecture Findings: brd-04-pre-call-briefing

## Overview
Architecture validation of BRD-04 Pre-Call Briefing. Data model, API contract, ADR alignment, and cross-BRD consistency reviewed against brd.md.

---

## 1. Data Model: briefing_source_exclusions missing restored_at Column

**Spec requirement:** ADR-0012 (Source Exclusion Undo — Soft Delete with restored_at Column) specifies `restored_at TIMESTAMPTZ NULL` column on `briefing_source_exclusions`. BRD-04 FR-17 requires excluded sources remain visible in collapsed excluded-sources area with original relatedness reasons after undo.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| ADR-0012 spec | PASS | ADR-0012 lines 20–28 define `restored_at` column and soft-delete behavior |
| BRD-04 data model | FAIL | brd.md lines 202–212 define `briefing_source_exclusions` WITHOUT `restored_at` column |
| BRD FR-17 alignment | FAIL | FR-17 ("excluded source meetings remain visible in a collapsed excluded-sources area") requires the row persist after undo — only possible with soft delete column |
| Security eval | PASS | Restore endpoint behavior tested (evals/security lines 99–106) |

**Required fix:** BRD-04 data model (brd.md lines 202–212) must add `restored_at TIMESTAMPTZ NULL` column to `briefing_source_exclusions` table schema to match ADR-0012 decision.

---

## 2. Data Model: upcoming_meetings Table Dependency

**Spec requirement:** BRD-05 FR-5 defines minimal `upcoming_meetings` schema (id, title, scheduled_start, created_by, status). BRD-04 data model references `upcoming_meetings(id)` as FK.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD-05 FR-5 (upcoming_meetings) | PASS | Minimal schema defined with owner-only ACL |
| BRD-04 FK references | PASS | `briefing_versions.upcoming_meeting_id` and `briefing_source_exclusions.upcoming_meeting_id` both FK to `upcoming_meetings(id)` (brd.md lines 189, 207) |
| upcoming_meetings BRD | PASS | BRD-05 now defines `upcoming_meetings` table (FR-5); dependency resolved |

**Status:** RESOLVED — BRD-05 FR-5 defines `upcoming_meetings` table.

## 3. API Contract: No OpenAPI Specification

**Spec requirement:** brd.md API Contract section (lines 167–179) lists 6 endpoints with method, path, and summary only. No request/response body schemas, no parameter types, no error responses defined.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | 6 endpoints listed; scope clearly defined |
| OpenAPI spec | NOT PRESENT | `contracts/openapi.yaml` does not define briefing endpoints |
| Eval coverage | PASS | Integration eval defines expected request/response shapes for all 6 endpoints |

**Note:** Per AGENTS.md, OpenAPI contracts are defined in `contracts/openapi.yaml`. The BRD lists the API surface but does not include full schema definitions. This is acceptable as the eval files define the contract. However, `contracts/openapi.yaml` should be updated before implementation begins.

---

## 4. ADR Alignment: ADRs 0009–0012 Status

**Spec requirement:** "ADR alignment: ADRs 0009-0012 (meeting ACL, upcoming_meetings table, staleness derivation, source exclusion undo) incorporated or flagged as outstanding."

### Findings

| ADR | Status | Evidence |
|-----|--------|----------|
| ADR-0009 (meeting ACL) | PARTIAL | FR-19 enforces owner-only semantics; `meeting_participants` used only for content matching; BUT `briefing_source_exclusions` data model missing `restored_at` per ADR-0012 conflict |
| BRD-05 FR-5 (upcoming_meetings) | PASS | RESOLVED — BRD-05 FR-5 now defines `upcoming_meetings` table |
| ADR-0011 (staleness) | PASS | FR-15 and NFR aligned; staleness derived at read time via source_memory_version_ids in `_meta` — BRD does not specify `_meta` field explicitly but FR-15 is consistent |
| ADR-0012 (source exclusion undo) | FAIL | ADR-0012 decision NOT incorporated in BRD-04 data model (see Finding #1 above) |

---

## 5. Cross-BRD Consistency: BRD-02 and BRD-03

**Spec requirement:** BRD-04 "Depends on: BRD-02 Manual Meeting Import for source meeting data foundation; BRD-03 Meeting Memory Processing for source memory, evidence statuses, conflict statuses, briefing-readiness signals, and source memory versions" (brd.md line 360).

### Findings

| Dependency | Status | Evidence |
|-------------|--------|----------|
| BRD-02 source meetings | PASS | BRD-04 FR-4 signal matching uses BRD-02's `meetings` and `meeting_participants` tables; no schema gaps |
| BRD-03 evidence statuses | PASS | BRD-04 FR-11 adopts BRD-03 evidence status conventions (`strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence`) |
| BRD-03 conflict statuses | PASS | BRD-04 FR-7 and FR-11 correctly exclude unresolved conflicts from briefing advice |
| BRD-03 source memory versions | PASS | ADR-0011 leverages BRD-03 source memory version tracking for staleness derivation |

---

## Summary

| Finding | Severity | File:Line | Description |
|---------|----------|-----------|-------------|
| `briefing_source_exclusions` missing `restored_at` column | Critical | brd.md lines 202–212 vs ADR-0012 | ADR-0012 decision not incorporated in BRD data model |
| `upcoming_meetings` table defined in BRD-05 | High | BRD-05 FR-5 | RESOLVED — BRD-05 defines `upcoming_meetings` table |
| OpenAPI contract not updated with briefing endpoints | Medium | contracts/openapi.yaml | 6 endpoints listed in BRD but not in OpenAPI spec |

**Verdict: BLOCKING** — 1 critical, 1 high, 1 medium.
Critical and high findings must be resolved before implementation proceeds.
