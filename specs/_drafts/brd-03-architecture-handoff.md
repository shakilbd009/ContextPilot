# BRD-03 Architecture Handoff — For Spec-Writer

**From:** architect
**Date:** 2026-05-21
**Task:** t_d44c2133
**Sources:** ADRs 0005-0008 (all Accepted), architect review (brd-03-architect-review.md), refiner review, validate-design findings

---

## Summary of Decisions Made

This document records the architecture decisions that affect BRD-03 prose and must be incorporated into the curated BRD artifact. All ADRs are now **Accepted**. Decisions are final unless marked with explicit deferral.

---

## 1. source_location Format

**Committed in:** ADR-0006 (Accepted) — `source_location Format — Committed` section.

**BRD prose implication:**

In the Functional Requirements (FR line 53: evidence field) and the Evidence record section, replace the TBD language with:

> **source_location** — A JSON object of type `{ "type": "char_offset", "start": integer, "end": integer }`. `start` and `end` are zero-based rune indices into the stored transcript or notes text. Offsets are validated: `0 <= start < end <= len(source_text)`. Offsets are computed against the stored (possibly whitespace-normalized) text, not the raw pasted input. When the source meeting is edited, all evidence offsets for the affected meeting are invalidated — the memory is marked stale and reprocessing is queued automatically (per FR-10). Source locations must never appear in logs or metrics.

**Also update:**
- Open Questions: remove OQ-3 (source-location format) — resolved.
- NFR or FR wherever `source_location format TBD` appears: replace with the above.

---

## 2. Memory API OpenAPI Surface

**Committed in:** `contracts/openapi.yaml` (updated in-place).

**Endpoints committed:**

| Method | Path | Summary |
|--------|------|---------|
| GET | `/meetings/{id}/memory` | Active memory version with content, briefing readiness, prior-memory inputs |
| GET | `/meetings/{id}/memory/versions` | Version history list |
| GET | `/meetings/{id}/memory/versions/{versionNumber}` | Specific version |
| POST | `/meetings/{id}/memory/reprocess` | Queue reprocess (user include/exclude prior memories or manual retry) |
| GET | `/meetings/{id}/memory/state` | Current processing lifecycle state |
| GET | `/meetings/{id}/memory/conflicts` | Pending conflict review queue |
| POST | `/meetings/{id}/memory/conflicts/{conflictId}/resolve` | Submit resolution note, create new version |

**Auth:** Same BearerAuth/CookieAuth as existing meeting endpoints. No separate auth scheme.

**Feature flag gate:** All endpoints gated by `FF_ENABLE_MEETING_MEMORY_PROCESSING`. When `false`, returns 403 with safe error — no content leaked.

**BRD prose implication:**

Add a new **API Contract** section (or sub-section under Functional Requirements) with the above table. The refiner-review flagged this as a missing MEDIUM issue — add it now.

The `POST /meetings` BRD-02 endpoint description should note (in a footnote or integration note) that on successful save, the handler inserts a `memory_processing_jobs` row with `trigger_type='import'` after the transaction commits, and the redirect does not wait for processing.

---

## 3. Auth Mechanism for Memory Endpoints

**Committed in:** ADR-0008 (Accepted) — `Authentication for Memory Endpoints` section.

**BRD prose implication:**

Add to FR or the new API Contract section:

> All memory API endpoints require an authenticated session (`Authorization: Bearer <jwt>` or `session` cookie). A user may only access memory for meetings they own or that are visible via standard meeting ACL. The server-side queue worker operates with elevated privileges but memory read/write API calls are subject to per-user authorization. When `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`, all memory endpoints return 403 with a safe error message; no memory content is disclosed.

This resolves the validate-design finding "Security specifies auth mechanism for memory endpoints."

---

## 4. Processing Latency SLO

**Status:** Deferred — OQ-2. The refiner review (HIGH) correctly flagged this as a vague deferral. The architect review deferred it to provider/runtime selection (OQ-5).

**BRD prose implication:**

Replace the current NFR latency row:

> **Processing latency** — Processing must be asynchronous, observable, and non-blocking for meetings up to the 50,000-character transcript-plus-notes limit. **Provisional target: 30 seconds at the 95th percentile** for a 50,000-character input under normal load. This target is provisional and will be updated to a provider-specific SLO when the processing worker and model runtime are selected during Phase 2 implementation. The target is measured from job pick-up to completion (not including queue wait time).

**Update the Open Questions:** OQ-2 (processing latency SLO) remains open but with a committed interim target and explicit deferral. Remove the "vague" characterization from the NFR table.

**Note for PM:** The 30-second provisional target is based on typical LLM API latencies for documents of this size. The actual SLO must be confirmed when the processing provider is selected (OQ-5). Queue wait time is tracked separately via `cp_meeting_memory_processing_queue_wait_ms`.

---

## 5. Semantic Similarity Threshold (for Conflict Detection)

**Committed in:** ADR-0007 (Accepted) — semantic similarity threshold used to determine when two items "disagree."

**BRD prose implication:**

The conflict detection section of FR (or ADR-0007 reference in the BRD) should note:

> Conflict detection uses semantic similarity to determine whether two memory items in the same category disagree. The threshold is **defined by the processing provider's semantic similarity model** — the `MemoryProcessor` implementation returns `conflicting_evidence` when two items have strong/weak evidence but semantic similarity falls below the merge threshold. The exact cosine-similarity or embedding-distance threshold is provider-specific and will be calibrated during Phase 2 eval.

This is the threshold referenced in validate-design as "Architect defines semantic similarity threshold." It cannot be committed as a single number because it depends on the embedding model selected (OQ-5). The interim BRD prose should describe the mechanism, not lock in a number.

---

## 6. Evidence Thresholds (Quality Status)

**Status:** Defined in ADR-0006 and the BRD itself. No additional threshold number is needed — the quality statuses (`strong_evidence`, `weak_evidence`, `insufficient_evidence`) are the thresholds.

**Clarification added:** The quality status is assigned by the `MemoryProcessor` based on evidence strength signals (number of corroborating snippets, source type, snippet length, etc.). These are internal calibration details. The BRD defines the four statuses and their meanings:

- `Strong evidence`: Multiple independent evidence snippets from the source that fully support the item.
- `Weak evidence`: One snippet or ambiguous evidence that partially supports the item.
- `Insufficient evidence`: No meaningful evidence; item must not be generated.
- `Conflicting evidence`: Two items with strong/weak evidence cannot be reconciled automatically.

**BRD prose implication:** No change needed to BRD prose — the four statuses are already defined. The implementation-calibration details (what constitutes "strong" vs "weak") belong in the eval fixtures, not the BRD.

---

## 7. ADR Status Summary

| ADR | Title | Status |
|-----|-------|--------|
| ADR-0005 | Job Queue Architecture | **Accepted** |
| ADR-0006 | Data Model | **Accepted** (with source_location committed) |
| ADR-0007 | Prior Memory Matching & Conflict Detection | **Accepted** |
| ADR-0008 | Observability & Privacy | **Accepted** (with auth mechanism added) |

All four ADRs are now in an Accepted/curation-ready state. The validate-design finding "Accept ADRs 0005-0008" is resolved.

---

## 8. Remaining Open Questions (for PM/Curator)

| OQ | Question | Owner | Status |
|----|----------|-------|--------|
| OQ-1 | Split feature flag or single dual-namespace flag? | Curator | Single flag recommended; must confirm before Phase 2 |
| OQ-2 | Exact processing latency SLO | PM + Architect | Interim 30s P95 target committed; provider-specific SLO deferred |
| OQ-4 | Quality distribution metrics for MVP ops? | Curator | Recommendation: emit as log field, not metric label |
| OQ-5 | Provider/runtime selection | Architect | Provider-agnostic interface defined; concrete selection out of scope for BRD-03 |

OQ-3 (source_location format) is **resolved** — character offsets committed.

---

## 9. What This Means for the Curated BRD

The spec-writer should incorporate the following when producing `specs/curated/brd-03-meeting-memory-processing.md`:

1. **Add API Contract section** with the 7 endpoints from openapi.yaml.
2. **Update Open Questions table** — remove OQ-3, update OQ-2 with interim target.
3. **Add auth mechanism prose** referencing BearerAuth/CookieAuth + meeting-level ACL.
4. **Add source_location format** (char_offset with start/end) in the Evidence section or a new Data Model sub-section.
5. **Confirm OQ-1** with PM before Phase 2 implementation (single flag vs split).
6. **Do not add** semantic similarity threshold as a specific number — describe the mechanism only.

---

## Files Changed

| File | Change |
|------|--------|
| `docs/adr/0005-meeting-memory-processing-queue-architecture.md` | Status: Proposed → Accepted |
| `docs/adr/0006-meeting-memory-data-model.md` | Status: Proposed → Accepted; source_location format committed |
| `docs/adr/0007-prior-memory-matching-conflict-detection.md` | Status: Proposed → Accepted |
| `docs/adr/0008-observability-privacy-architecture.md` | Status: Proposed → Accepted; auth mechanism added |
| `contracts/openapi.yaml` | 7 memory endpoints added; Memory schemas added; tags added |
| `specs/domain/brd-03-meeting-memory-processing.md` | NFR processing latency interim target (30s P95) — in this handoff |

**No application code was written.** All changes are documentation/contract only.