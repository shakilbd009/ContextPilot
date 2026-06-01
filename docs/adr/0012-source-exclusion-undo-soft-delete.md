# ADR-0012: Source Exclusion Undo — Soft Delete with restored_at Column

**Status:** Accepted
**Date:** 2026-05-23
**BRD:** brd-04
**Component:** Data Model / FR-17 Implementation

---

## Context

BRD-04 FR-17 says users can undo an upcoming-meeting-level source exclusion before regeneration. The restore endpoint (`POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore`) exists in the API contract. The `briefing_source_exclusions` table has no column to track undo state — an undo is likely implemented as a row DELETE. This means the audit history of exclusion events is permanently lost after undo.

---

## Decision

**Option B (Soft delete with `restored_at`) adopted.**

Add `restored_at TIMESTAMPTZ NULL` column to `briefing_source_exclusions`:

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `upcoming_meeting_id` | UUID FK | Yes | References `upcoming_meetings(id)` ON DELETE CASCADE |
| `excluded_source_meeting_id` | UUID FK | Yes | References `meetings(id)` ON DELETE CASCADE |
| `created_at` | TIMESTAMPTZ | Yes | Default now() |
| `restored_at` | TIMESTAMPTZ | No | NULL = active exclusion; set = restored by user |

Behavior:
- **Exclude:** Insert row with `restored_at = NULL`.
- **Restore (undo):** Update row SET `restored_at = now()` WHERE `id = $exclusion_id AND restored_at IS NULL`.
- **Query active exclusions:** `WHERE upcoming_meeting_id = $id AND restored_at IS NULL`.
- **Query excluded-sources area (FR-17):** All rows for this upcoming meeting, with `restored_at IS NOT NULL` shown in collapsed excluded-sources area.

The UNIQUE constraint on `(upcoming_meeting_id, excluded_source_meeting_id)` is preserved — a source can only be excluded once per upcoming meeting.

---

## Rationale

- FR-17 requirement: "excluded source meetings remain visible in a collapsed excluded-sources area with their original relatedness reasons" — this implies the row must persist after undo so the UI can render it.
- Soft delete (restored_at) preserves the audit trail of what was excluded and when. Hard delete would lose this information.
- The RESTORE endpoint semantics ("restore an excluded source meeting") maps cleanly to an UPDATE setting `restored_at`.
- The alternative (hard delete + separate history table) adds complexity without MVP benefit.

---

## Consequences

- **Positive:** Audit trail preserved. FR-17 UI requirement satisfied. Undo is idempotent (restoring an already-restored row is a no-op given the WHERE clause).
- **Negative:** Queries for "active exclusions only" must include `restored_at IS NULL` predicate. Negligible performance cost given the per-upcoming-meeting scope.
- **Neutral:** The `briefing_source_exclusions` row is never truly deleted — it persists forever. This is appropriate for audit purposes. Garbage collection can be addressed in BRD-06 (retention/deletion).

---

## Alternates Considered

**Option A (Hard delete on undo):** Row deleted on restore. Simpler but violates FR-17's requirement to show excluded sources in a collapsed area. Rejected.

**Option C (Separate history table):** A `briefing_source_exclusion_history` table tracks all exclusion/restore events. Cleanest separation but adds a second table for a low-volume MVP feature. Rejected in favor of Option B.