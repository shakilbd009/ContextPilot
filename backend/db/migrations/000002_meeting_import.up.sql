-- Migration: 000002_meeting_import
-- Description: Create meetings, meeting_participants, and idempotency_tokens tables for BRD-02

BEGIN;

-- Meetings table
CREATE TABLE IF NOT EXISTS meetings (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT        NOT NULL,
    completedat TIMESTAMPTZ NOT NULL,
    transcript  TEXT,
    notes       TEXT,
    contentsource TEXT      NOT NULL DEFAULT 'transcript',
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    createdby   UUID        NOT NULL,
    displayorder INTEGER     NOT NULL DEFAULT 0,
    CONSTRAINT  meetings_title_max_len CHECK (char_length(title) <= 500)
);

-- Meeting participants table
CREATE TABLE IF NOT EXISTS meeting_participants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    meetingid   UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    displayname TEXT        NOT NULL,
    email       TEXT,
    organization TEXT,
    role        TEXT,
    displayorder INTEGER     NOT NULL DEFAULT 0
);

-- Idempotency tokens table (24h TTL enforced on read)
CREATE TABLE IF NOT EXISTS idempotency_tokens (
    token       UUID        PRIMARY KEY,
    meetingid   UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiresat   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

-- Index for idempotency token lookups
CREATE INDEX IF NOT EXISTS idx_idempotency_tokens_expiresat ON idempotency_tokens(expiresat);

-- Index for meeting participant lookups
CREATE INDEX IF NOT EXISTS idx_meeting_participants_meetingid ON meeting_participants(meetingid);

-- Index for listing meetings by createdby
CREATE INDEX IF NOT EXISTS idx_meetings_createdby ON meetings(createdby);

COMMENT ON TABLE meetings IS 'Completed meeting records created via manual import (BRD-02)';
COMMENT ON TABLE meeting_participants IS 'Structured participants attached to a meeting record';
COMMENT ON TABLE idempotency_tokens IS 'Idempotency tokens for duplicate save prevention; tokens expire after 24 hours';

COMMIT;