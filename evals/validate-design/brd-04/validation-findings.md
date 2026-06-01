# validate-design: brd-04-pre-call-briefing

**Validator:** validate-design (orchestrator)
**Profile:** validator
**Run:** 297
**Date:** 2026-05-23
**Target:** specs/curated/brd-04-pre-call-briefing/brd.md
**Project:** /Users/shakilakram/projects/ContextPilot

---

## Validation Summary

| Area | Verdict | Critical | High | Medium | Low |
|------|---------|----------|------|--------|-----|
| API Contract | **BLOCKING** | 1 | 0 | 0 | 0 |
| Data Model | **NEEDS_WORK** | 1 | 1 | 0 | 0 |
| Observability | PASS | 0 | 0 | 0 | 0 |
| Feature Flags | PASS | 0 | 0 | 0 | 0 |
| Cross-BRD Consistency | **NEEDS_WORK** | 1 | 0 | 0 | 0 |
| ADR Alignment | **NEEDS_WORK** | 0 | 1 | 1 | 0 |
| Eval Coverage | PASS | 0 | 0 | 0 | 0 |
| US/AC Testability | PASS | 0 | 0 | 0 | 0 |

**Overall: NEEDS_WORK**

---

## Critical Issues (Must Fix Before Implementation)

### [C1] API Contract: 6 briefing endpoints absent from OpenAPI

**Spec reference:** BRD-04 API Contract (lines 169–179)

BRD-04 defines 6 briefing endpoints. None are present in `contracts/openapi.yaml`. The OpenAPI spec contains only BRD-02 (meetings) and BRD-03 (memory) endpoints. No `upcoming`, `briefing`, `briefing_versions`, or `briefing_source_exclusions` paths exist.

```
GET  /upcoming/{meetingId}/briefing              — not in OpenAPI
GET  /upcoming/{meetingId}/briefing/versions     — not in OpenAPI
GET  /upcoming/{meetingId}/briefing/versions/{v} — not in OpenAPI
POST /upcoming/{meetingId}/briefing/regenerate   — not in OpenAPI
POST /upcoming/{meetingId}/briefing/sources/{id}/exclude  — not in OpenAPI
POST /upcoming/{meetingId}/briefing/sources/{id}/restore  — not in OpenAPI
```

**Required action:** Add all 6 briefing endpoints to `contracts/openapi.yaml` with request/response schemas matching BRD-04's data model (briefing_versions, briefing_source_exclusions, Briefing JSON Content Schema). This was a Phase 1 blocker for BRD-02; it is equally a blocker for BRD-04.

---

### [C2] Data Model: upcoming_meetings table undefined

**Spec reference:** BRD-04 Data Model (lines 182–212)

BRD-04's `briefing_versions` and `briefing_source_exclusions` tables both reference `upcoming_meetings(id)`. BRD-05 now defines the `upcoming_meetings` table (FR-5), which resolves this gap.

**Status:** RESOLVED — BRD-05 defines `upcoming_meetings` table per FR-5.

---

## High Priority Issues (Should Fix)

### [H1] Data Model: briefing_source_exclusions.restored_at not in BRD-04 schema

**Spec reference:** ADR-0012, BRD-04 Data Model lines 202–211

ADR-0012 (status: Proposed) adds `restored_at TIMESTAMPTZ NULL` to `briefing_source_exclusions` to support FR-17's "excluded source meetings remain visible in a collapsed excluded-sources area after undo." The published BRD-04 schema (lines 202–211) does not include this column. The BRD-04 data model section was not updated to reflect this ADR decision.

If ADR-0012 is Accepted, the `briefing_source_exclusions` table schema in BRD-04 must be updated to include `restored_at`. If the ADR is not yet Accepted, this is a schema gap.

**Required action:** Confirm ADR-0012 status. If Accepted, update BRD-04 data model schema to include `restored_at`. If not yet Accepted, flag this as a pending schema dependency.

---

### [H2] ADR Alignment: ADRs 0009–0012 all status "Proposed"

**Spec reference:** docs/adr/0009–0012

All four ADRs relevant to BRD-04 are marked "Proposed" in their ADR headers. The parent PM gate task (t_a23b6e3d) reports "PM approves ADR-0009 through ADR-0012 posture for validate-design to proceed" — but the ADR documents themselves have not been formally Accepted/Decided. "Proposed" status means they are still under review, not locked.

The BRD-04 header states `adrs: none (PM/OQ decisions incorporated directly)`, which is inconsistent with ADRs 0009–0012 being referenced as the authoritative decision record for ACL semantics (0009), upcoming_meetings dependency (0010), staleness derivation (0011), and source exclusion undo (0012).

**Required action:** If these ADRs are intended to govern BRD-04 implementation, they must be formally Accepted (status changed from "Proposed"). If PM decisions are incorporated directly into the BRD without ADR authority, the ADRs should be marked "Superseded" or absorbed into BRD-04 directly. Do not proceed to implementation with 4 governing ADRs in "Proposed" state.

---

## Medium Priority Issues

### [M1] Cross-BRD: BRD-04 signals and BRD-03 prior-memory matching use different signal names

**Spec reference:** BRD-04 FR-4 vs BRD-03 FR-5

BRD-04 FR-4 uses "Same organization/client" signal with normalization. BRD-03 FR-5 uses "Same organization/client" for prior-memory matching with identical normalization definition. Both are consistent in practice, but the naming is slightly different from the BRD-03 framing.

BRD-04 correctly constrains the signal to exact-match after normalization (not broad matching), which aligns with BRD-03's constrained rule. No inconsistency — noted for awareness.

---

## Positive Findings (No Changes Needed)

- **Feature flags:** `FF_ENABLE_PRE_CALL_BRIEFING` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING` correctly registered in `specs/feature-flags.md` as Phase 2, Planned, with correct default false and dual-namespace behavior documented.
- **Observability:** Metrics and log events referenced in FR-22, FR-23, and NFR privacy requirements. Low-cardinality enums specified; high-cardinality content (raw titles, PII, snippets) banned. Product usefulness events enumerated for all key user actions.
- **US/AC testability:** All 8 US and 22 AC are present with specific eval method annotations. No TBD language in US or AC rows. OQ resolutions (7 items) documented in Pre-Implementation Approval Gate section.
- **Eval files:** All 5 required eval files present: e2e, unit, integration, security, perf — confirmed present by check-status-sync.sh.
- **Signal matching consistency:** BRD-04's 4-signal framework (same participant, similar title, same org/client, close chronology) is consistent with BRD-03's prior-memory matching signal definitions.
- **Evidence policy:** BRD-04 correctly adopts BRD-03's quality status conventions (strong_evidence, weak_evidence, insufficient_evidence, conflicting_evidence) and conflict exclusion requirements.
- **No TBD:** Pre-Implementation Approval Gate (lines 367–377) confirms 7 OQ resolutions are documented. No unresolved OQ in the spec body.

---

## Dependencies and Blockers

| Blocker | Severity | Source | Must Resolve Before |
|---------|----------|--------|---------------------|
| OpenAPI briefing endpoints missing | Critical | BRD-04 API Contract | Phase 1 implementation |
| upcoming_meetings table undefined | Critical | BRD-04 Data Model | BRD-04 implementation start — RESOLVED by BRD-05 FR-5 |
| ADR 0009–0012 "Proposed" status | High | docs/adr/0009–0012 | Implementation gate |
| briefing_source_exclusions.restored_at missing from BRD schema | High | ADR-0012 vs BRD-04 Data Model | Implementation start |

---

## Findings for PM Gate Review

1. **OpenAPI gap is a Phase 1 blocker** for BRD-04, same as it was for BRD-02. Backend cannot implement briefing endpoints without OpenAPI contract.

2. **upcoming_meetings table is an architectural gap.** BRD-04 is the first BRD to reference it. BRD-05 now defines the table (FR-5) — gap resolved.

3. **ADR status inconsistency:** BRD-04 header says "adrs: none" but implementation references ADRs 0009–0012 as the authoritative decisions. ADRs are "Proposed" not "Accepted."

4. **Eval files are complete** — check-status-sync.sh confirms all 5 eval files present.

5. **Spec quality is high** — 8 US, 22 AC, all testable, no TBD, feature flags correct, cross-BRD consistency maintained.

---

*Validator findings only — do not approve/reject. Report to PM gate review.*