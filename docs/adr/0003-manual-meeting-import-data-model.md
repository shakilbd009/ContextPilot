# ADR-003: Manual Meeting Import — Data Model

> Status: Accepted

## Context

BRD-02 requires storing a meeting record with structured participants attached. The BRD does not provide a database schema. Key decisions needed:
- How to model the meeting record
- Whether participants are embedded objects or separate related records
- Whether to store `transcript` and `notes` as separate columns or as a single content blob
- How to handle the combined 50,000 character limit (enforced at write time or at the application layer)
- Whether a `contentSource` field is needed and how to derive it

## Decision

We will store meetings and participants as **two separate tables** with a foreign key relationship, and store `transcript` and `notes` as **separate, nullable columns** in the `meetings` table.

### Schema

```sql
CREATE TABLE meetings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    transcript  TEXT,
    notes       TEXT,
    content_source TEXT NOT NULL CHECK (content_source IN ('transcript', 'notes', 'both')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID NOT NULL
);

CREATE TABLE meeting_participants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id  UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    email       TEXT,
    organization TEXT,
    role        TEXT
);

-- Combined character limit enforced at the application layer on write
-- content_source derived: 'transcript' if transcript present and notes empty,
--                         'notes' if notes present and transcript empty,
--                         'both' if both present
```

## Rationale

**Separate tables vs embedded JSON array**: The BRD explicitly requires participants as "structured meeting participant records attached to the meeting" (FR-52). A separate `meeting_participants` table with a foreign key:
- Enables participant-level queries (list all meetings for a given participant) in the future without schema changes
- Avoids JSON column limitations in PostgreSQL for indexable, queryable fields
- Aligns with the BRD's explicit wording of "records"

**Separate columns for transcript and notes** (vs a single content column with a type tag):
- The BRD requires preserving both when both are present (FR-69: "Allow users to save a meeting when both transcript and notes are present, preserving both inputs separately")
- The BRD requires distinguishing transcript from notes for downstream memory processing quality reasoning (FR-57: "Clearly distinguish transcript input from notes input so downstream memory processing can reason about source quality")
- Separate nullable columns are the simplest representation that satisfies both requirements

**Content source as a derived column**: Since `content_source` can be derived deterministically from the presence/absence of transcript and notes, it should be set by the application on write rather than accepted as user input. This avoids an invalid state where content_source says "transcript only" but notes is populated.

**50,000 character limit at application layer**: The limit is enforced in the Go write handler before inserting. PostgreSQL TEXT columns have no practical size limit that would conflict with 50K chars. Enforcing at the DB layer via a CHECK constraint would produce a catchable DB error; enforcing at the application layer produces a cleaner 400 with a field-level message.

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| Simplicity of single-table | Participants in a separate table require a JOIN or preload for listing meetings with participants |
| JSON participant storage | A JSON column would simplify writes (single INSERT); separate table requires two INSERT statements or a transaction |
| Schema flexibility | Adding new participant fields requires a migration; JSON allows schema-less extension |

The separate-table approach is intentionally more structured than a JSON blob because the BRD's Phase 2 goals (downstream memory processing, briefing generation) will need to query and join participants. A structured schema is the correct foundation.

## Consequences

1. Write operations must INSERT into both tables within a transaction.
2. Read operations for meeting list/detail must JOIN or use a preload to fetch participants.
3. The application layer sets `content_source` based on which of `transcript`/`notes` are non-null at write time.
4. The 50,000 character combined limit check happens in the Go handler before any DB write.
5. Deleting a meeting cascades to delete its participants (ON DELETE CASCADE).

## Alternatives Considered

### Single JSONB array for participants
Rejected because: The BRD uses "structured meeting participant records" language suggesting discrete records, and future queries (meetings by participant, participant counts) benefit from normalized storage. JSONB is reserved for truly schemaless, variable-shape data — participant fields are fixed by the BRD spec.

### Single content column with type discriminator
Rejected because: Cannot preserve both transcript and notes when both are present (FR-69) without losing the distinction the BRD requires for downstream quality reasoning (FR-57).

### CHECK constraint for 50K limit
Considered but rejected in favor of application-layer enforcement because: a DB-level CHECK violation produces an untyped SQL error rather than a clean 400 with field attribution. Application-layer enforcement yields a better API error shape.

## Date

2026-05-20