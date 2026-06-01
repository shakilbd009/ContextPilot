# ADR-0011: Briefing Staleness — Derived State vs. Persisted Flag

**Status:** Accepted
**Date:** 2026-05-23
**BRD:** brd-04
**Component:** Data Model / FR-15 Implementation

---

## Context

BRD-04 FR-15 states: *"If source meeting memory is reprocessed or changes after a briefing is generated, affected briefing versions are marked stale and regeneration is offered."* The `briefing_versions` table has no `is_stale` or `stale_at` column. The mechanism for staleness detection is described conceptually in BRD-03 FR-10 (source version hash comparison) but is not mapped to a specific database column or query pattern for briefings.

---

## Decision

**Option A (Derived at read time) adopted for MVP.**

Staleness is computed at query time using a deterministic derived-state algorithm:

1. When a briefing version is generated, store the `source_memory_version_ids[]` in the `content` JSONB or in a `briefing_source_versions` sidecar table.
2. On each read of the briefing, compare the current active memory version IDs of each source meeting against the stored `source_memory_version_ids`.
3. If any source memory version has changed (i.e., `updated_at` of source meeting's active memory version > `briefing_versions.created_at`), mark the briefing as stale in the UI.

The `content` JSONB schema is updated to include:

```json
{
  "concise_summary": { ... },
  "detailed_sections": { ... },
  "sources": [ ... ],
  "no_prior_memory_shell": { ... },
  "_meta": {
    "source_memory_version_ids": ["uuid", "uuid", ...]
  }
}
```

The `_meta` field is internal and not user-visible.

---

## Rationale

- Consistent with BRD-03 FR-10, which also derives staleness from `updated_at` comparison rather than a persisted flag.
- Avoids the complexity of an event subscription system (Option C) for MVP.
- Avoids adding a column that must be kept in sync by a background worker.
- The stale indicator is a UI concern (display "stale" badge + offer regeneration) — it does not need to be a persisted database column for the MVP.

---

## Consequences

- **Positive:** Simple. No additional columns or event infrastructure needed. Stale state is always accurate because it is computed fresh on each read.
- **Negative:** Stale state cannot be queried directly (e.g., "show me all stale briefings"). The UI must compute staleness on page load. This is acceptable for MVP where the user is viewing one upcoming meeting at a time.
- **Negative:** If the briefing page is loaded while source meetings are being reprocessed, the stale badge could appear mid-session. This is acceptable — it reflects the actual state.
- **Neutral:** The `_meta.source_memory_version_ids` field must be populated by the briefing generation worker. This is a required field, not optional.

---

## Future Consideration

If direct stale-query capability is needed (e.g., an operator dashboard showing all stale briefings), Option B (add `is_stale BOOLEAN`) can be adopted via ADR revision. The cost is a background stale-detection job that updates the flag whenever source memories change. This is a premature optimization for MVP.

---

## Implementation Note

The briefing generation worker must query the active memory versions for each qualifying source meeting at generation time and record their IDs in `_meta.source_memory_version_ids`. This is done at write time, not read time, to avoid needing source memory read access at briefing view time.