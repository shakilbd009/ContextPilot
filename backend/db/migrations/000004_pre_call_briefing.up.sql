-- Migration: 000004_pre_call_briefing
-- Description: Create briefing_versions, briefing_source_exclusions, and briefing_processing_jobs
--              tables for BRD-04 Pre-Call Briefing.
--              All three tables use standalone UUID primary keys (id) and reference
--              upcoming_meeting_id as a plain column — NOT as a foreign key.
--              This is intentional: upcoming_meetings is a
--              BRD-05 artifact (ff_enable_upcoming_meetings) whose table may not exist when
--              BRD-04 migrations run. Briefing tables are self-contained and do not require
--              the upcoming_meetings table to be present.
-- Depends: 000002_meeting_import (meetings, meeting_participants tables must exist)

BEGIN;

-- ─── briefing_versions ────────────────────────────────────────────────────────
-- Stores the structured briefing output per generation run; immutable once created.
-- Uses standalone UUID primary key; upcoming_meeting_id is a plain indexed column.
CREATE TABLE IF NOT EXISTS briefing_versions (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upcoming_meeting_id      UUID        NOT NULL,
    version_number           INTEGER     NOT NULL,
    status                   TEXT        NOT NULL CHECK (status IN ('active', 'superseded')) DEFAULT 'active',
    is_active                BOOLEAN     NOT NULL DEFAULT FALSE,
    result                   TEXT        NOT NULL CHECK (result IN (
                                       'ready', 'ready_with_caveats', 'no_prior_memory', 'failed'
                                   )) DEFAULT 'ready',
    preparation_status       TEXT        NOT NULL CHECK (preparation_status IN (
                                       'generating', 'ready', 'ready_with_caveats',
                                       'no_prior_memory', 'stale', 'failed', 'regenerating'
                                   )),
    content                  JSONB       NOT NULL DEFAULT '{}',
    source_count             INTEGER     NOT NULL DEFAULT 0,
    -- source_memory_version_ids: UUIDs of source memory versions captured at generation time
    -- Used at read time to detect staleness by comparing against current source versions
    source_memory_version_ids UUID[]     NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    trigger_type             TEXT        NOT NULL CHECK (trigger_type IN ('auto', 'manual_regenerate')),

    UNIQUE (upcoming_meeting_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_briefing_versions_meeting_id
    ON briefing_versions(upcoming_meeting_id);
CREATE INDEX IF NOT EXISTS idx_briefing_versions_active
    ON briefing_versions(upcoming_meeting_id, is_active)
    WHERE is_active = TRUE;

COMMENT ON TABLE briefing_versions IS
    'Per-version structured briefing output; exactly one is_active=true per upcoming meeting';

-- ─── briefing_source_exclusions ──────────────────────────────────────────────
-- Per-upcoming-meeting source exclusions with soft-delete (restored_at) per ADR-0012.
-- Uses standalone UUID primary key; upcoming_meeting_id is a plain indexed column.
CREATE TABLE IF NOT EXISTS briefing_source_exclusions (
    id                         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upcoming_meeting_id        UUID        NOT NULL,
    excluded_source_meeting_id UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- restored_at: NULL = active exclusion; SET = user restored this exclusion (soft delete per ADR-0012)
    restored_at                 TIMESTAMPTZ,

    UNIQUE (upcoming_meeting_id, excluded_source_meeting_id)
);

CREATE INDEX IF NOT EXISTS idx_briefing_source_exclusions_upcoming
    ON briefing_source_exclusions(upcoming_meeting_id)
    WHERE restored_at IS NULL;

COMMENT ON TABLE briefing_source_exclusions IS
    'Per-upcoming-meeting source exclusions; restored_at IS NULL means active exclusion (ADR-0012)';

-- ─── briefing_processing_jobs ─────────────────────────────────────────────────
-- Database-backed job queue for briefing generation (mirrors memory_processing_jobs pattern).
-- Uses standalone UUID primary key; upcoming_meeting_id is a plain indexed column.
CREATE TABLE IF NOT EXISTS briefing_processing_jobs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upcoming_meeting_id  UUID        NOT NULL,
    trigger_type         TEXT        NOT NULL CHECK (trigger_type IN ('auto', 'manual_regenerate')),
    status               TEXT        NOT NULL CHECK (status IN (
                                   'queued', 'processing', 'completed', 'failed', 'retrying', 'retry_exhausted'
                               )) DEFAULT 'queued',
    retry_count          INTEGER     NOT NULL DEFAULT 0,
    max_retries          INTEGER     NOT NULL DEFAULT 3,
    failure_reason       TEXT,
    previous_version_id  UUID,
    correlation_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    queued_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    next_retry_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_briefing_processing_jobs_meeting_id
    ON briefing_processing_jobs(upcoming_meeting_id);
CREATE INDEX IF NOT EXISTS idx_briefing_processing_jobs_status
    ON briefing_processing_jobs(status)
    WHERE status IN ('queued', 'retrying');
CREATE INDEX IF NOT EXISTS idx_briefing_processing_jobs_correlation_id
    ON briefing_processing_jobs(correlation_id);

COMMENT ON TABLE briefing_processing_jobs IS
    'Job queue for briefing generation; picked up by background worker via FOR UPDATE SKIP LOCKED';

COMMIT;