# ADR-0005: Meeting Memory Processing — Job Queue Architecture

> Status: Accepted

## Context

BRD-03 FR-1 requires that successful manual meeting imports automatically queue a memory processing job without blocking the import redirect response. FR-11, FR-16, and the NFR on import path latency impose a strict non-blocking requirement: the `POST /meetings` response must return before memory processing begins, and existing meeting detail access must continue even when the processing worker is unavailable. Failed processing must retry up to 3 times with exponential backoff, preserve the existing active memory version, and expose a user-safe retry state.

Key forces:
- The Go/Echo backend currently has no background job infrastructure.
- BRD-03 is provider-agnostic (FR-17): the processing runtime (LLM API call, local model, etc.) is not specified, meaning the job queue must be provider-agnostic.
- The BRD does not specify BullMQ, PostgreSQL advisory locks, inline goroutines, or any other backend.
- The system must degrade gracefully: queue for short outages, retry, then expose retry-exhausted state.
- Processing is Phase 2; the architecture must not require changes to the Phase 1 data model.

## Decision

We will use a **database-backed job queue using PostgreSQL advisory locks** for the initial implementation, with a defined interface that allows a future migration to a dedicated queue system (e.g., BullMQ, Redis, Temporal) without changing the processing logic.

### Queue Table Schema

```sql
CREATE TABLE memory_processing_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id      UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN ('import', 'reprocess', 'stale_reprocess', 'manual_retry', 'conflict_resolution')),
    status          TEXT NOT NULL CHECK (status IN (
                        'queued', 'processing', 'completed',
                        'completed_with_insufficient_evidence',
                        'failed', 'retrying', 'retry_exhausted'
                    )) DEFAULT 'queued',
    retry_count     INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    failure_reason  TEXT,  -- user-safe, never contains PII or raw content
    previous_version_id  UUID REFERENCES memory_versions(id) ON DELETE SET NULL,
    correlation_id  UUID NOT NULL DEFAULT gen_random_uuid(),
    queued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    next_retry_at   TIMESTAMPTZ
);

CREATE INDEX idx_memory_processing_jobs_meeting_id ON memory_processing_jobs(meeting_id);
CREATE INDEX idx_memory_processing_jobs_status ON memory_processing_jobs(status) WHERE status IN ('queued', 'retrying');
```

### Non-Blocking Trigger Flow

1. `POST /meetings` handler completes the meeting INSERT in a transaction.
2. After the transaction commits, the handler inserts a `memory_processing_jobs` row with `status='queued'` and `trigger_type='import'`.
3. The handler returns HTTP 201 with the meeting representation — **no await on processing**.
4. A background worker (polling or event-driven) picks up `queued` jobs.

The INSERT into `memory_processing_jobs` is fire-and-forget from the HTTP response path. The import response is not blocked.

### Worker Acquisition via PostgreSQL Advisory Locks

The background worker acquires a job using `SELECT ... FOR UPDATE SKIP LOCKED`:

```sql
BEGIN;
SELECT id, meeting_id, trigger_type, correlation_id
FROM memory_processing_jobs
WHERE status = 'queued'
ORDER BY queued_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED;
-- If a row is returned, the worker updates status='processing', started_at=now(), releases the lock on commit.
COMMIT;
```

`FOR UPDATE SKIP LOCKED` ensures:
- At most one worker acquires any given job.
- Multiple workers can run concurrently without deadlocking.
- Jobs that are already locked by a crashed worker are automatically skipped.

### Retry with Exponential Backoff

On processing failure:
1. Worker increments `retry_count`.
2. If `retry_count < max_retries`: set `status='retrying'`, compute `next_retry_at = now() + (2 ^ retry_count) seconds`, release lock.
3. If `retry_count >= max_retries`: set `status='retry_exhausted'`, set `failure_reason` to a user-safe message, release lock.

A separate retry scheduler loop (or cron) moves `retrying` jobs back to `queued` when `next_retry_at <= now()`.

### Graceful Degradation

If the processing provider/worker is unavailable:
- The job stays `queued` — no failure, no retry exhaustion.
- Existing meetings with active memory are accessible (their `active_memory_version_id` points to successful output from prior runs).
- The meeting detail page shows `queued` state.
- No user content is lost; no false failure state is shown.

## Rationale

**PostgreSQL advisory locks over BullMQ**: The system is in Phase 0/1 — introducing Redis or BullMQ adds an operational dependency not present in the current Docker Compose setup. PostgreSQL is already provisioned. Advisory locks provide at-least-once delivery with minimal new infrastructure.

**PostgreSQL advisory locks over inline goroutines**: Inline goroutines (fire-and-forget in the HTTP handler) cannot survive process restarts, cannot be retried after a crash, and have no observability surface. They are unsuitable for a feature with explicit retry requirements and NFRs on failure behavior.

**Interface-based queue abstraction**: The `MemoryProcessor` interface is the seam between the queue and the actual processing logic:

```go
type MemoryProcessor interface {
    Process(ctx context.Context, job MemoryProcessingJob) (*MemoryResult, error)
}
```

This allows the same logic to run from the PostgreSQL-backed queue in Phase 2, or be migrated to BullMQ/Temporal later by swapping the queue implementation while keeping `MemoryProcessor` unchanged.

**Non-blocking import path**: The job INSERT happens after the transaction commits, ensuring the import redirect is never delayed by queue contention or processing latency. This directly satisfies AC-2 (import redirect completes before processing done) and the NFR on import path latency.

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| BullMQ/Redis throughput | PostgreSQL advisory locks are slower than a dedicated queue for high-volume processing; acceptable for Phase 2 meeting-by-meeting cadence |
| Horizontal worker scaling across processes | `FOR UPDATE SKIP LOCKED` works within a single DB; multiple Go processes compete correctly but the DB is the bottleneck at scale |
| Job visibility UI | BullMQ/Redis have rich dashboards; PostgreSQL requires custom queries for visibility |
| Crash recovery for in-flight jobs | If a worker crashes mid-process after reporting completion but before the memory version is marked active, the job is done but its output may not be visible; mitigated by transactionally writing the memory version before updating job status |

## Consequences

1. The `memory_processing_jobs` table is created in a Phase 2 migration.
2. A `MemoryProcessor` interface is defined in the backend; the concrete implementation is injected.
3. The job worker runs as a separate goroutine or process; it is not in the HTTP request path.
4. Retries are managed by the queue layer, not the processor.
5. Feature flag `FF_ENABLE_MEETING_MEMORY_PROCESSING` gates both the queue INSERT (on import) and the worker execution.
6. Observability (per ADR-0008) emits `memory.job.queued`, `memory.job.started`, `memory.job.completed`, `memory.job.failed`, `memory.job.retrying` events with correlation IDs, meeting IDs (safe identifier), and retry counts — never raw content.

## Alternatives Considered

### Inline goroutine (fire-and-forget in HTTP handler)
Rejected because: No retry after crash, no observability, blocks if the goroutine panics, no way to query job state from the UI.

### Redis/BullMQ from day one
Rejected because: Adds operational dependency in Phase 2 when PostgreSQL is sufficient. Can be introduced later via the `JobQueue` interface abstraction.

### PostgreSQL-based queue with polling only (no advisory locks)
Polling without `FOR UPDATE SKIP LOCKED` risks double-processing or lost jobs under concurrent workers. Advisory locks are the correct primitive.

## Date

2026-05-21