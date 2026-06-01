-- Migration: 000006_idempotency_token_user_scope
-- Description: Add userid column to idempotency_tokens and scope tokens to (userid, token) tuple
--              to fix cross-user idempotency collision (F2 from pen test).
-- Fixes: Global idempotency token scope — two different users using the same UUID token
--        were getting 409 Conflicts for each other's meetings.

BEGIN;

-- Add userid column (NOT NULL, no default — will be backfilled from meetings table)
ALTER TABLE idempotency_tokens ADD COLUMN IF NOT EXISTS userid UUID NOT NULL;

-- Backfill existing rows: join to meetings to get the creator's userid
UPDATE idempotency_tokens
SET userid = m.createdby
FROM meetings m
WHERE idempotency_tokens.meetingid = m.id;

-- Drop the old primary key (token alone) and replace with composite unique constraint
-- First drop the existing primary key constraint (PostgreSQL auto-names it)
ALTER TABLE idempotency_tokens DROP CONSTRAINT IF EXISTS idempotency_tokens_pkey;

-- Re-create as composite on (userid, token) — still prevents duplicate tokens per user
ALTER TABLE idempotency_tokens ADD PRIMARY KEY (userid, token);

-- Add index for expiry cleanup queries (keeps the expiresat index for TTL sweeps)
-- The composite PK already covers (userid, token) lookups efficiently.

-- Add index for userid-only lookups (e.g., listing all tokens for a user)
CREATE INDEX IF NOT EXISTS idx_idempotency_tokens_userid ON idempotency_tokens(userid);

-- Keep the expiresat index for the TTL cleanup background job
CREATE INDEX IF NOT EXISTS idx_idempotency_tokens_expiresat ON idempotency_tokens(expiresat);

COMMENT ON COLUMN idempotency_tokens.userid IS 'User who created the idempotency token — tokens are scoped per-user, not global';

COMMIT;