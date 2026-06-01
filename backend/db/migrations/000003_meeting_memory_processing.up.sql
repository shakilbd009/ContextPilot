-- Migration: 000003_meeting_memory_processing
-- Description: Create memory processing tables for BRD-03 — job queue, version history, evidence, conflicts, prior-memory inputs
-- Depends: 000002_meeting_import (meetings table must exist)

BEGIN;

-- ─── memory_processing_jobs ───────────────────────────────────────────────────
-- Database-backed job queue using PostgreSQL advisory locks (ADR-0005)
CREATE TABLE IF NOT EXISTS memory_processing_jobs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id           UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    trigger_type         TEXT        NOT NULL CHECK (trigger_type IN (
                                     'import', 'reprocess', 'stale_reprocess', 'manual_retry', 'conflict_resolution'
                                 )),
    status               TEXT        NOT NULL CHECK (status IN (
                                     'queued', 'processing', 'completed',
                                     'completed_with_insufficient_evidence',
                                     'failed', 'retrying', 'retry_exhausted'
                                 )) DEFAULT 'queued',
    retry_count          INTEGER     NOT NULL DEFAULT 0,
    max_retries          INTEGER     NOT NULL DEFAULT 3,
    failure_reason       TEXT,
    -- previous_version_id: set when job replaces an active version (so we can restore it on failure)
    previous_version_id  UUID,
    correlation_id        UUID        NOT NULL DEFAULT gen_random_uuid(),
    queued_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    -- next_retry_at: set when status moves to 'retrying'; cleared when job is picked up again
    next_retry_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_memory_processing_jobs_meeting_id
    ON memory_processing_jobs(meeting_id);
CREATE INDEX IF NOT EXISTS idx_memory_processing_jobs_status
    ON memory_processing_jobs(status)
    WHERE status IN ('queued', 'retrying');
CREATE INDEX IF NOT EXISTS idx_memory_processing_jobs_correlation_id
    ON memory_processing_jobs(correlation_id);

COMMENT ON TABLE memory_processing_jobs IS
    'Job queue for meeting memory processing; picked up by background worker via FOR UPDATE SKIP LOCKED (ADR-0005)';

-- ─── memory_versions ───────────────────────────────────────────────────────────
-- Stores the structured memory output per processing run (ADR-0006)
CREATE TABLE IF NOT EXISTS memory_versions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id      UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    job_id          UUID        REFERENCES memory_processing_jobs(id) ON DELETE SET NULL,
    version_number  INTEGER     NOT NULL,
    status          TEXT        NOT NULL CHECK (status IN (
                                  'active', 'superseded', 'conflict_review'
                              )) DEFAULT 'active',
    is_active       BOOLEAN     NOT NULL DEFAULT FALSE,
    content         JSONB       NOT NULL,
    -- trigger_type records why this version was created
    trigger_type    TEXT        NOT NULL CHECK (trigger_type IN (
                                  'import', 'reprocess', 'stale_reprocess', 'manual_retry', 'conflict_resolution'
                              )),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by      UUID,

    UNIQUE (meeting_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_memory_versions_meeting_id
    ON memory_versions(meeting_id);
CREATE INDEX IF NOT EXISTS idx_memory_versions_active
    ON memory_versions(meeting_id, is_active)
    WHERE is_active = TRUE;

COMMENT ON TABLE memory_versions IS
    'Per-version structured memory output; exactly one is_active=true per meeting (ADR-0006)';

-- meetings.active_memory_version_id: FK set atomically when a version becomes active
ALTER TABLE meetings
    ADD COLUMN IF NOT EXISTS active_memory_version_id UUID
    REFERENCES memory_versions(id) ON DELETE SET NULL;

-- ─── memory_evidence ──────────────────────────────────────────────────────────
-- Normalized evidence records joined to memory content items (ADR-0006)
CREATE TABLE IF NOT EXISTS memory_evidence (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_version_id UUID       NOT NULL REFERENCES memory_versions(id) ON DELETE CASCADE,
    category         TEXT        NOT NULL,
    item_id          TEXT        NOT NULL,
    source_type      TEXT        NOT NULL CHECK (source_type IN (
                                  'transcript', 'notes', 'prior_memory_reference'
                              )),
    source_location  JSONB       NOT NULL,
    evidence_snippet TEXT        NOT NULL,
    quality_status   TEXT        NOT NULL CHECK (quality_status IN (
                                  'strong_evidence', 'weak_evidence', 'insufficient_evidence'
                              )),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memory_evidence_version_id
    ON memory_evidence(memory_version_id);

COMMENT ON TABLE memory_evidence IS
    'Evidence citations for memory content items; enables joinable evidence queries without parsing JSONB (ADR-0006)';

-- ─── memory_conflicts ──────────────────────────────────────────────────────────
-- Isolated conflict records excluded from normal memory until resolved (ADR-0006, FR-12)
CREATE TABLE IF NOT EXISTS memory_conflicts (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id        UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    memory_version_id UUID        REFERENCES memory_versions(id) ON DELETE CASCADE,
    conflicting_items JSONB       NOT NULL,
    quality_status    TEXT        NOT NULL DEFAULT 'conflicting_evidence',
    review_status     TEXT        NOT NULL CHECK (review_status IN (
                                  'pending', 'reviewed'
                              )) DEFAULT 'pending',
    resolution_note   TEXT,
    resolved_at       TIMESTAMPTZ,
    resolved_by       UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memory_conflicts_meeting_id
    ON memory_conflicts(meeting_id);
CREATE INDEX IF NOT EXISTS idx_memory_conflicts_pending
    ON memory_conflicts(review_status)
    WHERE review_status = 'pending';

COMMENT ON TABLE memory_conflicts IS
    'Pending and resolved conflicts isolated from normal memory categories; excluded from briefing until reviewed (ADR-0006)';

-- ─── memory_prior_memory_inputs ───────────────────────────────────────────────
-- Tracks which prior memories were used as input to a given version (ADR-0007)
CREATE TABLE IF NOT EXISTS memory_prior_memory_inputs (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_version_id         UUID        NOT NULL REFERENCES memory_versions(id) ON DELETE CASCADE,
    prior_memory_version_id   UUID        NOT NULL REFERENCES memory_versions(id) ON DELETE RESTRICT,
    match_confidence          TEXT        CHECK (match_confidence IN ('safe_match', 'uncertain')),
    included_by_user          BOOLEAN     NOT NULL DEFAULT FALSE,
    excluded_by_user          BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (memory_version_id, prior_memory_version_id)
);

CREATE INDEX IF NOT EXISTS idx_memory_prior_inputs_version_id
    ON memory_prior_memory_inputs(memory_version_id);

COMMENT ON TABLE memory_prior_memory_inputs IS
    'Which prior memories were used as processing input, and whether matched automatically or added by user (ADR-0007)';

COMMIT;
