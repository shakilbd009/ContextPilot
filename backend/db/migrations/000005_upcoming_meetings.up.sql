-- Migration: 000005_upcoming_meetings
-- Description: Create upcoming_meetings and upcoming_meeting_participants tables for BRD-05
-- Depends: 000004_pre_call_briefing (briefing_versions/briefing_source_exclusions/briefing_processing_jobs must exist first,
--              and their upcoming_meeting_id columns are plain UUID — no FK to upcoming_meetings)

BEGIN;

-- ─── upcoming_meetings ────────────────────────────────────────────────────────
-- Stores manually created upcoming meeting anchors for BRD-05 pre-call prep.
CREATE TABLE IF NOT EXISTS upcoming_meetings (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title                   TEXT        NOT NULL,
    scheduled_start         TIMESTAMPTZ NOT NULL,
    description             TEXT,
    client_or_organization  TEXT,
    status                  TEXT        NOT NULL CHECK (status IN ('scheduled', 'cancelled')) DEFAULT 'scheduled',
    created_by              UUID        NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Compound index for owner-list queries: filter by owner + status, sort by scheduled_start
CREATE INDEX IF NOT EXISTS idx_upcoming_meetings_owner_status_start
    ON upcoming_meetings(created_by, status, scheduled_start ASC);

-- UNIQUE constraint: prevent same owner from creating two meetings with identical title + start time
ALTER TABLE upcoming_meetings ADD CONSTRAINT uq_upcoming_meetings_owner_title_start
    UNIQUE (created_by, title, scheduled_start);

-- Index on scheduled_start for time-based queries (e.g. "upcoming within 30 days")
CREATE INDEX IF NOT EXISTS idx_upcoming_meetings_scheduled_start
    ON upcoming_meetings(scheduled_start);

COMMENT ON TABLE upcoming_meetings IS
    'Manually created upcoming meeting anchors for pre-call preparation (BRD-05)';

-- ─── upcoming_meeting_participants ───────────────────────────────────────────
-- Structured participants for upcoming meetings; plain column (no DB FK constraint)
-- per BRD-05 FR-6 decision: application-level referential integrity only.
CREATE TABLE IF NOT EXISTS upcoming_meeting_participants (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upcoming_meeting_id   UUID        NOT NULL,
    display_name          TEXT,
    email                 TEXT,
    organization          TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Participant must have at least one of display_name or email
    CHECK (
        (display_name IS NOT NULL AND display_name <> '')
        OR
        (email IS NOT NULL AND email <> '')
    )
);

-- Index for participant lookup by upcoming meeting
CREATE INDEX IF NOT EXISTS idx_upcoming_meeting_participants_meeting_id
    ON upcoming_meeting_participants(upcoming_meeting_id);

COMMENT ON TABLE upcoming_meeting_participants IS
    'Structured participants for BRD-05 upcoming meetings; application-level referential integrity (per BRD-05 FR-6 decision)';

COMMIT;