# ADR-0006: Meeting Memory Processing — Data Model

> Status: Accepted

## Context

BRD-03 defines a memory structure with seven categories (summary, decisions, action items, risks/blockers, open questions, stakeholder notes, next recommended focus), each containing items with evidence, quality status, and source tracking. The BRD requires version history (FR-9), stale detection and reprocessing (FR-10), conflict isolation (FR-6, FR-13, FR-14), and briefing-readiness signals (FR-15). The BRD does not specify whether to use JSON columns, structured tables, or a hybrid approach.

Key decisions needed:
1. Schema for the memory output per category — JSON or normalized?
2. How to model version history without explosion of tables
3. How to model evidence (snippet + source location + source type)
4. How to model conflicts and their review queue
5. **source_location format — committed below**

## Decision

We will use a **hybrid approach**: a `memory_versions` table with a `content` JSONB column storing the structured memory output, combined with normalized tables for evidence references, conflict records, and job/version linkage. This gives us the schema flexibility of JSON for the variable-shape memory categories while preserving the queryability and referential integrity of normalized tables for version lifecycle, conflict management, and audit.

### Schema

```sql
-- Core memory version record
CREATE TABLE memory_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id          UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    job_id              UUID REFERENCES memory_processing_jobs(id) ON DELETE SET NULL,
    version_number      INTEGER NOT NULL,
    status              TEXT NOT NULL CHECK (status IN (
                            'active', 'superseded', 'conflict_review'
                        )) DEFAULT 'active',
    is_active           BOOLEAN NOT NULL DEFAULT FALSE,
    -- content stores the full structured memory as JSONB
    -- see Memory JSON Schema section below
    content             JSONB NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by          UUID REFERENCES users(id),

    UNIQUE (meeting_id, version_number)
);

CREATE INDEX idx_memory_versions_meeting_id ON memory_versions(meeting_id);
CREATE INDEX idx_memory_versions_active ON memory_versions(meeting_id, is_active) WHERE is_active = TRUE;

-- Evidence records: each item can have one or more evidence citations
CREATE TABLE memory_evidence (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_version_id   UUID NOT NULL REFERENCES memory_versions(id) ON DELETE CASCADE,
    -- category and item_id reference into the memory content JSON
    category            TEXT NOT NULL,  -- 'summary', 'decisions', 'action_items', etc.
    item_id             TEXT NOT NULL,  -- stable ID within the JSON content
    source_type         TEXT NOT NULL CHECK (source_type IN ('transcript', 'notes', 'prior_memory_reference')),
    -- source_location stores the stable locator; format is flexible (char offset, line ref, etc.)
    -- application layer validates format per source_type
    source_location     JSONB NOT NULL, -- e.g., {"type": "char_offset", "start": 1234, "end": 5678}
    evidence_snippet    TEXT NOT NULL,
    quality_status      TEXT NOT NULL CHECK (quality_status IN (
                            'strong_evidence', 'weak_evidence', 'insufficient_evidence'
                        )),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_memory_evidence_version_id ON memory_evidence(memory_version_id);

-- Conflict records: isolated from normal memory until resolved
CREATE TABLE memory_conflicts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id              UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    memory_version_id       UUID REFERENCES memory_versions(id) ON DELETE CASCADE,
    -- JSON blob storing conflicting items from both sides
    conflicting_items       JSONB NOT NULL,  -- {current: [...], prior: [...]}
    quality_status          TEXT NOT NULL DEFAULT 'conflicting_evidence',
    review_status           TEXT NOT NULL CHECK (review_status IN ('pending', 'reviewed')) DEFAULT 'pending',
    resolution_note         TEXT,
    resolved_at             TIMESTAMPTZ,
    resolved_by            UUID REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_memory_conflicts_meeting_id ON memory_conflicts(meeting_id);
CREATE INDEX idx_memory_conflicts_pending ON memory_conflicts(review_status) WHERE review_status = 'pending';

-- Link table: which prior memories were used as input to a given memory version
CREATE TABLE memory_prior_memory_inputs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_version_id   UUID NOT NULL REFERENCES memory_versions(id) ON DELETE CASCADE,
    prior_memory_version_id UUID NOT NULL REFERENCES memory_versions(id) ON DELETE RESTRICT,
    match_confidence    TEXT CHECK (match_confidence IN ('safe_match', 'uncertain')),
    included_by_user    BOOLEAN NOT NULL DEFAULT FALSE,  -- FALSE = system-matched, TRUE = user-added
    excluded_by_user     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (memory_version_id, prior_memory_version_id)
);
```

### Memory JSON Content Schema (JSONB column)

The `content` JSONB column follows this structure:

```json
{
  "summary": {
    "statement": "...",
    "quality_status": "strong_evidence",
    "items": [...]
  },
  "decisions": {
    "items": [
      {
        "id": "dec-001",
        "decision_statement": "We will use PostgreSQL advisory locks for the job queue",
        "status": "standalone",
        "quality_status": "strong_evidence",
        "prior_memory_reference_id": null
      }
    ]
  },
  "action_items": {
    "items": [
      {
        "id": "ai-001",
        "description": "Write ADR-0005",
        "owner": "architect",
        "due_date": "2026-05-22",
        "status": "open",
        "quality_status": "strong_evidence"
      }
    ]
  },
  "risks_blockers": {
    "items": [...]
  },
  "open_questions": {
    "items": [...]
  },
  "stakeholder_notes": {
    "items": [
      {
        "id": "sn-001",
        "participant_id": "uuid",
        "preferences": [...],
        "concerns": [...],
        "commitments": [...],
        "influence_stake": "high",
        "quality_status": "weak_evidence"
      }
    ]
  },
  "next_recommended_focus": {
    "statement": "...",
    "quality_status": "strong_evidence",
    "supporting_item_ids": ["dec-001", "ai-002"]
  },
  "briefing_readiness": {
    "ready": true,
    "core_categories_ready": true,
    "blocking_conflicts": [],
    "insufficient_categories": ["stakeholder_notes"]
  }
}
```

Each category follows a consistent pattern: `items[]` with a stable `id`, category-specific fields, `quality_status`, and `evidence` references via `memory_evidence` table.

### source_location Format — Committed

**Decision**: Use **character offsets** (byte-indexed) as the primary `source_location` format for transcript and notes text.

**Schema**:
```json
{ "type": "char_offset", "start": 1234, "end": 5678 }
```

**Rationale**:
- Character offsets are stable across re-processing runs that use the same transcript version (the offset points into the stored text, not a line number that could shift with text normalization).
- Line references are fragile: transcript pasted from different tools may have different line-breaking behavior. Character offsets are robust to whitespace normalization.
- Byte offsets are unambiguous in UTF-8 stored text. The application layer validates that `start` and `end` are valid rune indices within the source text.
- Token-range or embedding-based references are deferred — they require a tokenization library and add provider coupling.

**Constraints**:
- `start` and `end` are integers; `0 <= start < end <= len(source_text)`
- The offset is computed against the **stored** transcript/notes text, not the original pasted input (which may have been normalized on save per BRD-02 form preservation rules).
- When transcript or notes are edited after initial save, the character offsets in existing evidence records may become invalid — the system marks affected memory versions as stale and queues reprocessing (per BRD-03 FR-10).
- Source locations must never appear in logs or metrics (per ADR-0008 privacy requirements).

**Out of scope for this ADR**:
- Line references as an alternative format (can be added as `{ "type": "line_ref", "start_line": 12, "end_line": 15 }` if future tools require it — the schema is extensible).
- Token ranges (requires tokenizer alignment with the processing provider).
- Byte vs rune offset for non-ASCII text — implementation must use rune indices, not raw bytes, for Unicode correctness.

## Rationale

**JSONB content column with normalized evidence/conflict tables**:
- The BRD's seven categories and variable item counts per meeting are naturally modeled as JSON. Forcing a normalized table-per-category creates a schema explosion and makes additive category additions require migrations.
- Evidence citations and conflict records are queryable and joinable via normalized tables without parsing JSON.
- The `content` JSONB is immutable per version (new version = new row); no updates to existing rows, only new inserts. This makes version comparison and audit trivial.
- PostgreSQL JSONB supports GIN indexes for content searches if needed in the future (e.g., find all meetings with `summary.quality_status = 'weak_evidence'`).

**Separate conflict table**:
- Unresolved conflicts are excluded from briefing input per FR-13 and AC-18. Keeping them in a separate table with `review_status='pending'` makes the exclusion queryable without parsing memory content.
- Conflicts are tied to a specific `memory_version_id`; when a new version is created after resolution, it carries the resolution note but the conflict record is preserved for audit (FR-14).

**`is_active` flag as the source of truth**:
- FR-9 requires the latest successful version to be active by default. A Boolean flag (instead of querying by `MAX(version_number)` on every read) lets the meeting detail query do a single-indexed lookup: `WHERE meeting_id = ? AND is_active = TRUE`.
- The previous active version is atomically deactivated when a new version is activated: `UPDATE memory_versions SET is_active = FALSE WHERE meeting_id = ? AND is_active = TRUE; INSERT new version with is_active = TRUE;` all within a transaction.

**Prior memory input linking**:
- `memory_prior_memory_inputs` records which prior memories were used, whether matched automatically (system) or added by the user, enabling FR-5 (visibility of prior memories used) and FR-12 (user can include/exclude and reprocess).

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| Full normalization | Evidence and item content are split between JSONB and the evidence table; a join is needed to reconstruct the full item+evidence picture |
| Schema flexibility of pure JSON | Normalized tables for evidence and conflicts add migration complexity for new fields |
| Query simplicity | Category-level queries require JSONB operators; not as straightforward as `SELECT * FROM decisions` |
| Storage efficiency | Duplicating item IDs as strings in both `content` JSONB and `memory_evidence` table (for joinability) uses marginal extra storage |

The hybrid model is the correct balance for a Phase 2 feature with explicit version history, conflict review, and evidence-grounding requirements that will persist into BRD-04 (Pre-Call Briefing) and BRD-06 (Retention).

## Consequences

1. A Phase 2 migration creates `memory_versions`, `memory_evidence`, `memory_conflicts`, and `memory_prior_memory_inputs` tables.
2. The `meetings` table gains an `active_memory_version_id` column (nullable FK to `memory_versions.id`) updated atomically when a new successful version is activated.
3. The `memory_versions.content` JSONB column is validated by the application layer on write (expected structure), not by a PostgreSQL JSON Schema validator.
4. `source_location` JSONB structure is defined by the application layer and supports multiple formats; OQ-3 (exact format) is deferred to implementation but the schema accommodates character offsets, line references, and token ranges.
5. Evidence-grounding evals (AC-7) verify that items in `content` have corresponding records in `memory_evidence` where `quality_status IN ('strong_evidence', 'weak_evidence')`.
6. Stale detection: when the meeting's `updated_at` advances past the `memory_versions.created_at` of the active version, the active memory is marked `stale` and reprocessing is queued.

## Alternatives Considered

### Pure JSONB (no normalized evidence/conflict tables)
Rejected because: Evidence citations and conflicts need to be queryable and joinable without parsing large JSON blobs. A pure JSON approach makes it hard to efficiently query "all unresolved conflicts across all meetings" or "all evidence for a specific item" without full table scans.

### Fully normalized (tables for every category)
Rejected because: Seven category tables plus item tables creates a schema explosion. Categories are additive in future BRDs (e.g., new memory categories). The JSONB content column provides the flexibility to add categories without migrations while normalized tables handle the structural concerns (versions, evidence, conflicts, inputs).

### EAV (entity-attribute-value) pattern
Rejected because: EAV makes querying and integrity constraints complex. The hybrid model avoids EAV while retaining the flexibility of schemaless content storage.

## Date

2026-05-21