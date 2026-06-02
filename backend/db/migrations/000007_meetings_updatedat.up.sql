-- Migration: 000007_meetings_updatedat
-- Description: Add `updatedat` column to `meetings` to support BRD-02 FR-10 stale
--              detection (drives BRD-04 FR-15 `IsMemoryStale` join) and to fix the
--              PATCH /api/v1/meetings/{id} 500 caused by UpdateMeeting setting
--              `updatedat = now()` against a column that did not exist.
--
-- Column name follows the existing `meetings.*` lowercase-no-underscore convention
-- (cf. createdat, completedat, contentsource, displayorder, createdby). It is the
-- same name the application code already references in
--   - backend/internal/meeting/repository.go (UpdateMeeting)
--   - backend/internal/memory/repository.go (IsMemoryStale, GetMeetingInfo)
-- so no Go-side changes are required for column naming. The new column is
-- populated from `createdat` on backfill so stale detection does not spuriously
-- fire for meetings processed before this migration.
--
-- Fixes: PATCH 500 with body `column "updatedat" of relation "meetings" does not
--        exist (SQLSTATE 42703)` and the resulting server-log schema disclosure
--        (CWE-209, MEDIUM). Found by ethical-hacker task t_6af3f7c8 (run 1941,
--        2026-06-01).

BEGIN;

-- Add column. NOT NULL with DEFAULT NOW() so:
--   - existing rows get a sensible default at ALTER time
--   - future inserts always have a value even if the writer forgets
--   - PATCH queries (`SET updatedat = now()`) can rely on the column existing
ALTER TABLE meetings
    ADD COLUMN IF NOT EXISTS updatedat TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Backfill: existing meetings should have updatedat == createdat so that
-- IsMemoryStale (which compares meetings.updatedat > memory_versions.created_at)
-- does not flag every pre-migration meeting as stale right after deploy.
-- The ALTER above set updatedat = NOW() for existing rows, but createdat is in
-- the past for those rows, so updatedat would otherwise be > createdat and the
-- stale check would fire spuriously for already-processed pre-migration meetings.
UPDATE meetings
SET updatedat = createdat;

COMMENT ON COLUMN meetings.updatedat IS
    'Last source mutation time. Updated by PATCH /api/v1/meetings/{id}. '
    'Drives BRD-02 FR-10 / BRD-04 FR-15 stale-memory detection via '
    'memory.Repository.IsMemoryStale.';

COMMIT;
