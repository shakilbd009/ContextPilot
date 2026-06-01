# BRD-03 Architect Review — Meeting Memory Processing

> Status: In Architect Review
> BRD: brd-03-meeting-memory-processing
> Author: architect
> Date: 2026-05-21

---

## Overview

This document is the architect review for BRD-03 Meeting Memory Processing, produced during the design-refinement phase. It covers the component inventory, API surface, feature flag configuration, constraints, non-goals, and integration points. It is intended to be read alongside the four ADRs produced in parallel:

- [ADR-0005: Queue Architecture](./docs/adr/0005-meeting-memory-processing-queue-architecture.md)
- [ADR-0006: Data Model](./docs/adr/0006-meeting-memory-data-model.md)
- [ADR-0007: Prior Memory Matching & Conflict Detection](./docs/adr/0007-prior-memory-matching-conflict-detection.md)
- [ADR-0008: Observability & Privacy](./docs/adr/0008-observability-privacy-architecture.md)

---

## Component Inventory

### 1. Memory Processing Queue Worker

**Responsibility**: Poll for queued memory processing jobs, acquire them via PostgreSQL advisory locks, dispatch to the memory processor, handle retries and failure states, and emit observability events.

**Location**: `backend/internal/jobs/memory_processor_worker.go` (Phase 2)

**Inputs**:
- `memory_processing_jobs` table (queues, status)
- Feature flag `FF_ENABLE_MEETING_MEMORY_PROCESSING`

**Outputs**:
- `memory_versions` rows (active version atomically updated)
- `memory_evidence` rows
- `memory_conflicts` rows
- `memory_prior_memory_inputs` rows
- Observability events (via `ProcessingResult` struct — no direct log emission from processor)

**Key behaviors**:
- Polls `memory_processing_jobs` where `status = 'queued'` using `SELECT ... FOR UPDATE SKIP LOCKED`
- On acquisition: updates `status = 'processing'`, `started_at = now()`
- Calls `MemoryProcessor.Process(ctx, job)` with the job's meeting context
- On success: inserts new `memory_version`, atomically sets `is_active = TRUE`, updates `memory_processing_jobs.status = 'completed'` or `'completed_with_insufficient_evidence'`
- On transient failure: increments `retry_count`, sets `status = 'retrying'`, computes `next_retry_at`
- On permanent failure or retry exhaustion: sets `status = 'failed'` or `'retry_exhausted'`, sets `failure_reason` (user-safe string, no raw content)
- Does not have direct access to the logger or metrics registry — emits `ProcessingResult` to the queue worker harness which handles emission

**Failure modes**:
- Worker crash mid-processing: job stays `processing` with `started_at` set but no `completed_at`; stale detection (separate goroutine/cron) resets `processing` jobs with `started_at` older than a threshold (e.g., 30 minutes) back to `queued`
- Database unavailable: job stays `queued`; worker retries on next poll cycle
- Provider unavailable: treated as transient failure, enters retry loop

---

### 2. Memory Processor

**Responsibility**: The core business logic — takes a meeting and optional matched prior memories, extracts and categorizes memory items with evidence, detects conflicts, and produces a structured `MemoryResult`.

**Location**: `backend/internal/memory/processor.go` (interface: `MemoryProcessor`)

**Interface**:
```go
type MemoryProcessor interface {
    Process(ctx context.Context, job MemoryProcessingJob) (*ProcessingResult, error)
}

type MemoryProcessingJob struct {
    Meeting          *Meeting
    PriorMemories    []*MemoryVersion
    CorrelationID    uuid.UUID
    TriggerType      string  // 'import' | 'reprocess' | 'stale_reprocess' | 'manual_retry'
}

type ProcessingResult struct {
    CorrelationID               uuid.UUID
    MeetingID                   uuid.UUID
    TriggerType                 string
    DurationMs                  int64
    RetryCount                  int
    Status                      JobStatus
    FailureClass                string
    ActiveVersionChangeReason    string
    ProcessingVersionNumber      int
    CategoryCount               int
    QualityDistribution         map[string]int
    FlagEvaluationDurationMs    int64
    MemoryContent               json.RawMessage  // the content JSONB; not emitted to logs/metrics
}
```

**Pipeline stages**:
1. **Flag evaluation**: Check `FF_ENABLE_MEETING_MEMORY_PROCESSING`; if false, return early with `Status =skipped` (no-op, no error)
2. **Prior memory matching**: Call `PriorMemoryMatcher.FindMatches()` — returns eligible prior memories with match confidence (`safe_match` or `uncertain`)
3. **Evidence extraction**: For each category, extract items from transcript and notes; attach evidence snippets and source locations; assign quality status
4. **Conflict detection**: Run `ConflictDetector.Detect()` against matched prior memories; write `memory_conflicts` for unresolved conflicts
5. **Quality assignment**: At item level and category level, set `quality_status` based on evidence strength
6. **Next recommended focus synthesis**: Synthesize across strong/weak evidence categories; mark `Insufficient evidence` if basis is insufficient
7. **Briefing readiness**: Compute `briefing_readiness` signal — `ready`, `ready with weak/insufficient sections`, or `needs review`
8. **Result assembly**: Return `ProcessingResult` with safe fields for observability and `MemoryContent` for storage

**Failure modes**:
- Provider timeout: return `error` with `FailureClass = timeout`; worker treats as transient failure and retries
- Invalid input (meeting has no transcript or notes): return `error` with `FailureClass = validation_error`; worker marks job failed (not retryable)
- Provider returns 5xx: return `error`; worker treats as transient provider error

---

### 3. Prior Memory Matcher

**Responsibility**: Given a current meeting, find eligible prior memories using constrained signals (same participant email, similar title, close chronology). Implements the safe match definition from ADR-0007.

**Location**: `backend/internal/memory/prior_memory_matcher.go`

**Interface**:
```go
type PriorMemoryMatcher interface {
    FindMatches(ctx context.Context, meeting *Meeting) ([]PriorMemoryMatch, error)
}

type PriorMemoryMatch struct {
    MemoryVersion  *MemoryVersion
    MatchConfidence string  // 'safe_match' | 'uncertain'
    MatchedOn      MatchSignals
}

type MatchSignals struct {
    SameParticipant   bool
    TitleSimilarity   float32  // 0.0–1.0
    ChronologyDays    int       // days between meetings (negative = prior)
}
```

**Key behaviors**:
- AND of three signals: same email participant + similar title (Levenshtein distance <= 3 or shared 4+ word token sequence) + chronology within [-90, +7] days
- Returns both `safe_match` and `uncertain` candidates; uncertain matches are shown to the user for confirmation before being used as processing input
- User can include/exclude matches — `memory_prior_memory_inputs.included_by_user` or `excluded_by_user` flags trigger reprocessing

---

### 4. Conflict Resolver

**Responsibility**: Manages the conflict review queue and resolution workflow.

**Location**: `backend/internal/memory/conflict_resolver.go`

**Interface**:
```go
type ConflictResolver interface {
    DetectConflicts(ctx context.Context, current *MemoryVersion, priors []*MemoryVersion) ([]Conflict, error)
    ResolveConflict(ctx context.Context, conflictID uuid.UUID, resolutionNote string, resolvedBy uuid.UUID) (*MemoryVersion, error)
}
```

**DetectConflicts**:
- Compares current meeting items against matched prior memory items per ADR-0007 conflict detection rules
- Both items must have strong or weak evidence to trigger a conflict
- Returns `[]Conflict` — caller writes these to `memory_conflicts` table

**ResolveConflict**:
- Validates conflict exists and is `pending`
- Updates `memory_conflicts` with `review_status = 'reviewed'`, `resolution_note`, `resolved_at`, `resolved_by`
- Triggers reprocessing (queues a new job with `trigger_type = 'manual_retry'`)
- New memory version carries the resolution note; original sources are preserved for audit

---

### 5. Version Manager

**Responsibility**: Manages the memory version lifecycle — creating new versions, atomically activating the latest successful version, deactivating the previous active, detecting stale memory, and exposing version history.

**Location**: `backend/internal/memory/version_manager.go`

**Interface**:
```go
type VersionManager interface {
    CreateVersion(ctx context.Context, meetingID uuid.UUID, jobID uuid.UUID, content json.RawMessage) (*MemoryVersion, error)
    ActivateVersion(ctx context.Context, versionID uuid.UUID, reason VersionChangeReason) error
    GetActiveVersion(ctx context.Context, meetingID uuid.UUID) (*MemoryVersion, error)
    GetVersionHistory(ctx context.Context, meetingID uuid.UUID) ([]*MemoryVersion, error)
    MarkStale(ctx context.Context, meetingID uuid.UUID) error
    IsStale(ctx context.Context, meetingID uuid.UUID) (bool, error)
}
```

**Key behaviors**:
- `CreateVersion`: inserts new row with `version_number = COALESCE(MAX(version_number), 0) + 1` for the meeting, `is_active = FALSE` initially
- `ActivateVersion`: within a transaction, sets `is_active = FALSE` for all versions of this meeting, then sets the target version `is_active = TRUE` and `status = 'active'`
- Stale detection: when `meetings.updated_at` advances past the `memory_versions.created_at` of the active version, the active version is marked stale and a reprocess job is queued
- Failed/retrying/retry-exhausted jobs do not call `ActivateVersion` — the previous active version remains active

---

## API Endpoints

The following API endpoints are required for BRD-03. The BRD-02 `POST /meetings` write path must be updated to queue a memory processing job on successful import.

### New Endpoints (BRD-03)

#### `GET /meetings/:meetingId/memory`
Returns the active memory version for a meeting.

**Response**: `200 OK`
```json
{
  "meeting_id": "uuid",
  "version_number": 3,
  "status": "active",
  "processing_state": "completed",
  "is_stale": false,
  "is_briefing_ready": true,
  "briefing_readiness": {
    "ready": true,
    "core_categories_ready": true,
    "blocking_conflicts": [],
    "insufficient_categories": ["stakeholder_notes"]
  },
  "content": { /* Memory JSON schema per ADR-0006 */ },
  "prior_memories_used": [
    { "meeting_id": "uuid", "title": "...", "completed_at": "...", "match_confidence": "safe_match" }
  ],
  "created_at": "ISO8601"
}
```

#### `GET /meetings/:meetingId/memory/versions`
Returns version history for a meeting.

**Response**: `200 OK`
```json
{
  "meeting_id": "uuid",
  "versions": [
    { "version_number": 3, "is_active": true, "status": "active", "created_at": "ISO8601", "created_by": "uuid or null" },
    { "version_number": 2, "is_active": false, "status": "superseded", "created_at": "ISO8601", "created_by": "uuid or null" },
    { "version_number": 1, "is_active": false, "status": "superseded", "created_at": "ISO8601", "created_by": "uuid or null" }
  ]
}
```

#### `GET /meetings/:meetingId/memory/versions/:versionNumber`
Returns a specific memory version.

#### `POST /meetings/:meetingId/memory/reprocess`
Queues a manual reprocess (triggered by user edit to prior memory inclusions/exclusions, source correction, or manual retry).

**Request**: (optional body)
```json
{
  "excluded_prior_memory_ids": ["uuid"],
  "included_prior_memory_ids": ["uuid"],
  "reason": "user_corrected_includes"
}
```

**Response**: `202 Accepted` with `{ "job_id": "uuid", "correlation_id": "uuid" }`

#### `GET /meetings/:meetingId/memory/state`
Returns the current processing lifecycle state for the meeting's memory.

**Response**: `200 OK`
```json
{
  "meeting_id": "uuid",
  "processing_state": "queued",
  "active_version_number": 2,
  "is_stale": false,
  "latest_job": {
    "job_id": "uuid",
    "status": "queued",
    "retry_count": 0,
    "failure_reason": null,
    "correlation_id": "uuid",
    "queued_at": "ISO8601"
  }
}
```

#### `GET /meetings/:meetingId/memory/conflicts`
Returns pending conflicts for review.

**Response**: `200 OK`
```json
{
  "meeting_id": "uuid",
  "conflicts": [
    {
      "id": "uuid",
      "category": "decisions",
      "current_item": { "id": "...", "statement": "...", "quality_status": "strong_evidence" },
      "prior_item": { "memory_version_id": "uuid", "statement": "...", "quality_status": "strong_evidence" },
      "created_at": "ISO8601"
    }
  ]
}
```

#### `POST /meetings/:meetingId/memory/conflicts/:conflictId/resolve`
Resolves a conflict with an evidence-backed resolution note.

**Request**:
```json
{
  "resolution_note": "After reviewing both sources, the Q2 decision was superseded by the Q3 reprioritization."
}
```

**Response**: `200 OK` with the new memory version (same shape as `GET /meetings/:meetingId/memory`)

### Updated Endpoints (BRD-02)

#### `POST /meetings` (updated)
After the Phase 1 BRD-02 implementation, this endpoint must be updated to:
1. Insert `memory_processing_jobs` row with `trigger_type = 'import'` after the meeting INSERT transaction commits
2. Return `202 Accepted` or `201 Created` with the meeting body plus `{ "memory_job_id": "uuid", "correlation_id": "uuid" }` in a `_meta` or header field

The redirect after `POST /meetings` must not wait for memory processing completion (per AC-2).

---

## Feature Flag Configuration

BRD-03 requires a dual-namespace feature flag (per AGENTS.md and the BRD):

| Flag | Type | Default | Scope |
|------|------|---------|-------|
| `FF_ENABLE_MEETING_MEMORY_PROCESSING` | server (Go) | `false` | Gates queue INSERT on import; gates worker processing; gates all memory API endpoints |
| `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` | browser (SvelteKit) | `false` | Gates memory UI, memory state indicators, retry/reprocess button visibility, review queue access |

**OQ-1 (Split flag vs single dual-namespace flag)**: This architect review recommends **a single dual-namespace flag** for Phase 2 initial implementation, with a clear path to splitting if later rollout requirements demand it. Rationale:
- Phase 2 MVP scope is coherent: either the entire memory processing feature is on or off.
- Separate flags add complexity (OQ-1 is explicitly deferred to curation) with no demonstrated need in Phase 2.
- The interface is designed so that each flag can be promoted to independent control later without changing the API surface or data model.

**Flag evaluation timing**: `FF_ENABLE_MEETING_MEMORY_PROCESSING` is evaluated server-side at two points:
1. In the `POST /meetings` handler: if false, the memory processing job is NOT queued
2. In the queue worker: if false, the worker skips the job (sets `status = 'failed'`, `failure_reason = 'feature_disabled'`)

**OQ-1 resolution needed before Phase 2 implementation**: The curator must decide before Phase 2 implementation begins whether automatic processing, memory UI, and review/reprocess actions need independent rollout flags.

---

## Constraints and Non-Goals

### Constraints (from BRD-03)

1. **Provider agnosticism** (FR-17): The system must not depend on a specific processing provider, model, or runtime. All architectural decisions (ADR-0005 through ADR-0008) are designed to this constraint — the `MemoryProcessor` interface is the provider-agnostic seam.

2. **Privacy** (FR-20, FR-23, NFR): Raw meeting content, evidence snippets, participant PII, stakeholder notes, generated memory, prior memory content, and resolution notes must not appear in logs or metrics. The `ProcessingResult` struct and `Redactor` utility architecturally enforce this (ADR-0008).

3. **Non-blocking import** (FR-1, NFR on import path latency): Manual meeting import redirect must not wait for memory processing. The fire-and-forget `memory_processing_jobs` INSERT after the transaction commits enforces this (ADR-0005).

4. **Retention deferred** (NFR on retention): Memory versions and evidence are retained until BRD-06 defines final policy. No deletion logic is implemented in Phase 2.

5. **Combined transcript+notes limit** (BRD-02 constraint inherited): The 50,000 character limit is enforced at the BRD-02 import handler. The memory processor receives pre-validated content.

### Non-Goals (from BRD-03)

The following are explicitly out of scope and must not be introduced into the Phase 2 implementation:

1. **Future meeting creation** — handled by BRD-04 Pre-Call Briefing
2. **Global contacts, participant deduplication, CRM-style relationship history, identity resolution** — deferred to future BRDs
3. **Broad opaque matching for prior-memory matching** (embedding similarity, cross-system relatedness) — deferred to future Related Meeting Detection BRD
4. **Provider-based relationship inference and cross-system relatedness** — deferred
5. **Sensitive personal attributes, personality profiling, psychological inference, unsupported influence assumptions in stakeholder notes** — excluded by FR-7
6. **Specific model, vendor, or runtime** — BRD is provider-agnostic (FR-17)
7. **Automated conflict resolution** — conflicts require user resolution (FR-13: "cannot be reconciled automatically")

---

## Integration Points

### With BRD-02 (Manual Meeting Import)

- BRD-03 depends on BRD-02's `meetings` table, `meeting_participants` table, and `POST /meetings` endpoint
- The `POST /meetings` handler is updated to insert a `memory_processing_jobs` row after the meeting transaction commits
- Meeting detail page (`GET /meetings/:id`) must display memory processing state alongside meeting data
- Feature flag registration must include `ff_enable_meeting_memory_processing` in `specs/feature-flags.md`

### With BRD-04 (Pre-Call Briefing) — blocks

- BRD-04 requires source-grounded, conflict-free memory from BRD-03 as input
- Briefing eligibility is determined by `briefing_readiness.ready = true` and `briefing_readiness.blocking_conflicts = []` per AC-20
- The `active_memory_version_id` FK on the meeting is the primary briefing input pointer

### With BRD-06 (Privacy & Retention) — soft

- BRD-06 will define retention, deletion, export, and redaction policy for memory versions and evidence
- Phase 2 retains all versions until BRD-06 ships; no cleanup jobs are implemented

### With BRD-07 (Manual Memory Correction) — soft

- BRD-07 will extend conflict resolution workflows
- Phase 2 conflict resolution (AC-19) is the minimal viable version: evidence-backed resolution note + new version

---

## Open Questions (OQ-1 through OQ-5)

| ID | Question | Owner | Status for Architect Review |
|----|----------|-------|----------------------------|
| OQ-1 | Split feature flag or single dual-namespace flag? | Curator/Architect | **Recommendation**: single flag for Phase 2; design supports split later |
| OQ-2 | Exact processing latency SLO | Architect | **Deferred**: requires provider/runtime selection; use adaptive SLO tracking in Phase 2 |
| OQ-3 | Exact source-location format | Architect | **Deferred to implementation**: ADR-0006 schema accommodates char offsets, line refs, token ranges |
| OQ-4 | Quality distribution metrics for MVP ops? | Curator | **Recommendation**: emit as log field (ADR-0008), not metric labels; collect via log aggregation |
| OQ-5 | Provider/runtime selection | Architect | **Out of scope for BRD-03 architect review**: the interface is provider-agnostic; concrete selection is a separate decision |

---

## Verification Checklist

Before Phase 2 implementation begins, the following must be confirmed:

- [ ] `FF_ENABLE_MEETING_MEMORY_PROCESSING` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` are registered in `specs/feature-flags.md` with lifecycle `Planned → In Dev`
- [ ] `specs/feature-flags.md` is updated with the Phase 2 flag entry before any code is written
- [ ] OpenAPI contract (`contracts/openapi.yaml`) is updated with the new BRD-03 endpoints listed above
- [ ] Event contracts (`contracts/events.md`) are updated if any new async events are introduced (e.g., `memory.version.activated`)
- [ ] OQ-1 is resolved by the curator before Phase 2 implementation begins
- [ ] The architecture evals (`check-feature-flags.sh`, `check-no-panic.sh`, `check-no-background-context.sh`) are updated to cover the new Phase 2 source files
- [ ] `scripts/check-status-sync.sh` passes with the new BRD-03 artifacts
