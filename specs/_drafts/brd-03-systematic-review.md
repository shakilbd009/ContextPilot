# BRD-03 Systematic Design Review — Meeting Memory Processing

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | brd-03 |
| Review type | Systematic deep design review |
| Reviewer | architect |
| Date | 2026-05-21 |
| Status | Draft |
| Task | t_a5b098df |
| Parent | t_021bf175 (ADR creation, still in-progress) |

---

## Review Scope

Source documents:
- `specs/domain/brd-03-meeting-memory-processing.md` (BRD-03, Accepted)
- `specs/curated/brd-02-manual-meeting-import.md` (BRD-02, Curated — API surface and data model context)
- `docs/adr/0002-manual-meeting-import-validation-strategy.md`
- `docs/adr/0003-manual-meeting-import-data-model.md`
- `docs/adr/0004-manual-meeting-import-form-content-preservation.md`

ADRs 0005 and 0006 (meeting-memory-processing-architecture and memory-data-model) are listed as outputs of the parent task t_021bf175, which is currently running. This review identifies what those ADRs must cover and flags a dependency risk if they are not yet finalized.

---

## Focus Area 1: End-to-End Processing Flow

### Flow: Import → Queue → Processor → Storage → Briefing Readiness

```
BRD-02 POST /meetings
        │
        ▼
┌───────────────────────┐
│ Meeting record saved   │
│ (transcript + notes + │
│  participants +      │
│  completedAt)         │
└───────────────────────┘
        │
        ▼ (if FF_ENABLE_MEETING_MEMORY_PROCESSING=true)
┌───────────────────────┐
│ Processing job queued │
│ (non-blocking)        │
│ correlation_id issued │
└───────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────┐
│ Queue worker picks up job                    │
│ Inputs: meeting record + source_version_id  │
│         + safely matched prior memories     │
└─────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────┐
│ Memory processor                            │
│ - Extracts structured memory categories     │
│ - Evaluates evidence quality per item        │
│ - Detects conflicts with prior memory       │
│ - Synthesizes next recommended focus        │
│ - Emits processing lifecycle log events     │
└─────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────┐
│ New memory version written (candidate)       │
│ - Active pointer updated if successful       │
│ - Failed runs leave prior active intact     │
│ - Conflict items placed in review queue     │
└─────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────┐
│ Briefing readiness evaluated                │
│ - Core categories (summary, decisions,      │
│   action_items, risks, open_questions)     │
│ - Stakeholder notes + next focus optional   │
│ - Unresolved conflicts block readiness     │
└─────────────────────────────────────────────┘
```

### Non-blocking Guarantee

The BRD requires that "manual meeting save and meeting detail navigation must not wait for memory processing completion." The import flow must:
1. Write the meeting record to PostgreSQL
2. Enqueue a processing job (e.g., via PostgreSQL advisory lock + enqueue, or an external queue)
3. Return the redirect to `/meetings/<id>` immediately

The queue worker is a separate goroutine or worker process that processes jobs asynchronously. Existing active memory remains viewable even when reprocessing is pending/running/failed/retry-exhausted (BRD-03, AC-22, AC-12).

### Idempotency

Each processing run creates a new memory version with a unique `memory_version_id`. The job itself must be idempotent: if a job is retried, it creates a new version (not a duplicate). The job key should incorporate `meeting_id + source_version_id + attempted_at` or equivalent to prevent double-processing on retry exhaustion.

### What could go wrong
- **Job enqueue failure**: If the queue write fails, the import transaction must not also fail — the job enqueue must be its own database transaction or use an outbox pattern.
- **Queue worker crash mid-processing**: The job record stays in "processing" state. A recovery mechanism (staleness detection, max processing time) must mark it as failed and allow manual retry.
- **Source meeting deleted while job queued**: The processor must check that the meeting still exists before processing; if deleted, mark the job as permanently failed with a clear reason.

---

## Focus Area 2: JSON Memory Structure — All 7 Categories

### Top-Level Memory Document

```json
{
  "id": "<uuid>",
  "meetingId": "<uuid>",
  "version": "<integer>",
  "status": "completed | completed_with_insufficient_evidence | failed | stale | reprocessing",
  "briefingReady": true | false,
  "briefingReadyDetail": {
    "coreCategoriesReady": true | false,
    "blockingConflicts": ["<conflict_id>"],
    "insufficientCoreCategories": ["<category>"]
  },
  "sourceVersion": "<string>",
  "processedAt": "<ISO8601>",
  "processingDurationMs": "<integer>",
  "priorMemoryInputs": [
    {
      "memoryId": "<uuid>",
      "meetingId": "<uuid>",
      "memoryVersion": "<integer>",
      "included": true,
      "excluded": false,
      "excludedBy": null,
      "excludedAt": null
    }
  ],
  "categories": {
    "summary": { ... },
    "decisions": { ... },
    "actionItems": { ... },
    "risksBlockers": { ... },
    "openQuestions": { ... },
    "stakeholderNotes": { ... },
    "nextRecommendedFocus": { ... }
  },
  "conflictQueue": [
    {
      "conflictId": "<uuid>",
      "category": "<string>",
      "priorItemRef": "<uuid>",
      "currentItemRef": "<uuid>",
      "priorEvidence": { ... },
      "currentEvidence": { ... },
      "status": "pending | resolved",
      "resolutionNote": null,
      "resolvedAt": null,
      "resolvedBy": null
    }
  ]
}
```

### Category Schemas

#### summary
```json
{
  "summary": {
    "text": "<string>",
    "evidence": [
      {
        "snippet": "<string>",
        "sourceType": "transcript | notes | prior_memory",
        "sourceLocator": { "type": "char_offset | line_ref", "value": "<string>" },
        "sourceMeetingId": "<uuid>"
      }
    ],
    "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence"
  }
}
```

#### decisions
```json
{
  "decisions": [
    {
      "id": "<uuid>",
      "statement": "<string>",
      "changeStatus": "standalone | confirms_prior | changes_prior | reverses_prior",
      "priorDecisionRef": "<uuid | null>",
      "evidence": [ ... ],
      "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence | conflicting_evidence",
      "conflictingWith": "<uuid | null>"
    }
  ]
}
```

#### actionItems
```json
{
  "actionItems": [
    {
      "id": "<uuid>",
      "description": "<string>",
      "owner": "<string | null>",
      "ownerSpecified": true | false,
      "dueDate": "<ISO8601 date | null>",
      "dueDateSpecified": true | false,
      "status": "pending | in_progress | completed | deferred",
      "evidence": [ ... ],
      "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence | conflicting_evidence"
    }
  ]
}
```

#### risksBlockers
```json
{
  "risksBlockers": [
    {
      "id": "<uuid>",
      "type": "risk | blocker",
      "description": "<string>",
      "evidence": [ ... ],
      "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence"
    }
  ]
}
```

#### openQuestions
```json
{
  "openQuestions": [
    {
      "id": "<uuid>",
      "question": "<string>",
      "evidence": [ ... ],
      "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence"
    }
  ]
}
```

#### stakeholderNotes
```json
{
  "stakeholderNotes": [
    {
      "id": "<uuid>",
      "participantRef": "<uuid>",
      "personDisplayName": "<string>",
      "relationship": "<string>",
      "preferences": "<string | null>",
      "concerns": "<string | null>",
      "commitments": "<string | null>",
      "influenceStake": "<string | null>",
      "evidence": [ ... ],
      "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence"
    }
  ]
}
```

Constraints per BRD-03:
- Stakeholder notes are **limited to meeting-relevant business context** — no personality profiling, psychological inference, or unsupported influence assumptions.
- The `participantRef` links to a `meeting_participant` record from BRD-02.

#### nextRecommendedFocus
```json
{
  "nextRecommendedFocus": {
    "text": "<string>",
    "evidence": [
      {
        "snippet": "<string>",
        "sourceType": "transcript | notes | prior_memory",
        "sourceLocator": { "type": "char_offset | line_ref", "value": "<string>" },
        "sourceMeetingId": "<uuid>",
        "sourceMemoryItemRef": "<uuid>"
      }
    ],
    "qualityStatus": "strong_evidence | weak_evidence | insufficient_evidence"
  }
}
```

### Insufficient Evidence Rule

Per BRD-03 FR: "Mark a category as `Insufficient evidence` when the source material does not support useful content for that category; the system must not fill weak categories with guesses."

This means every category **must always be present** in the output — even if it is `{ "qualityStatus": "insufficient_evidence", "items": [] }`. The system must not omit a category entirely when the source doesn't support it.

### What could go wrong
- **Processor hallucinates content**: Mitigated by requiring evidence for every item, insufficient_evidence fallback, and hallucination eval fixtures (BRD-03 AC-6).
- **Evidence snippets don't actually support the item**: Mitigated by evidence-grounding evals (AC-7).
- **Sensitive stakeholder profiling**: Guardrail that stakeholder notes exclude personal attributes, personality profiling, psychological inference — must be enforced by eval (AC-10).

---

## Focus Area 3: Version History Model

### Version Creation Triggers

A new memory version is created on:
1. First successful processing of a meeting
2. Every subsequent processing or reprocessing run (BRD-03 FR: "Create a new memory version for each processing or reprocessing run")
3. Conflict resolution by user (creates a "reviewed" version)

### Version Lifecycle

```
memory_versions
├── id: uuid (PK)
├── meeting_id: uuid (FK)
├── version_number: integer (sequential per meeting)
├── status: processing | completed | completed_with_insufficient_evidence | failed | retrying | retry_exhausted | stale
├── is_active: boolean (only one true at a time per meeting)
├── source_version: string (hash of meeting fields at processing time)
├── processing_trigger: import | source_edit | manual_retry | prior_memory_input_change | conflict_resolution
├── processed_at: timestamptz
├── processing_duration_ms: integer
├── error_class: string (if failed)
├── error_message_safe: string (user-safe, not containing transcript/notes)
├── retry_count: integer
├── correlation_id: uuid
└── created_at: timestamptz
```

### Active Version Pointer

- Exactly one version per meeting has `is_active = true` at any time.
- Failed, retrying, retry_exhausted, or stale versions are never active.
- On successful completion of a processing run, the new version becomes active (AC-11).
- Failed runs do not replace the active version (AC-12).
- When source edits are detected, the active version is marked `stale` and a reprocessing run is queued; the old active version remains active until the new run succeeds.

### Source Version Tracking

`source_version` must change whenever any of the following change:
- `transcript` text
- `notes` text
- `participants` (any participant add/remove/modify)
- `title`
- `completedAt`

This is used to detect staleness. The source version can be a hash (SHA-256 of canonicalized fields) stored as a hex string.

### What changes between versions

Not all fields change — version N+1 may have:
- Different extracted items in categories (new items, removed items, changed qualityStatus)
- Different evidence snippets (processor found better or different evidence)
- New/changed/resolved conflicts
- Different `nextRecommendedFocus`

What does **not** change: the `meetingId`, the sequential `version_number`, the `source_version` that was the input.

### Version Comparison (Could Have)

BRD-03 Could Have: "Show a compact diff between memory versions for user review." This is not MVP scope but the data model supports it — two versions can be diffed by comparing their category items and qualityStatus fields.

### What could go wrong
- **Version number collision**: Must use a database sequence or atomic increment, not application-level counter.
- **Active pointer corruption**: Race condition if two successful runs complete nearly simultaneously. Mitigated by using a database transaction with `SELECT FOR UPDATE` on the meeting row, or an atomic compare-and-swap on `is_active`.
- **Retention unbounded**: Versions grow with every edit. BRD-06 will define retention policy. Until then, versions are retained with the source meeting.

---

## Focus Area 4: Retry/Reprocess Logic

### Retry Triggers

| Trigger | Cause | Automatic? |
|---------|-------|------------|
| `import` | New meeting saved via BRD-02 | Automatic |
| `source_edit` | Meeting transcript/notes/participants/title/completedAt changed | Automatic |
| `manual_retry` | User clicks retry on failed/stale/retry-exhausted | Manual |
| `prior_memory_input_change` | User includes/excludes a prior memory | Automatic |
| `conflict_resolution` | User resolves a conflict | Automatic |

### Retry Policy

- **Max retries**: 3 automatic retries for transient processing failures
- **Backoff**: Exponential backoff — delay doubles each retry (e.g., 10s, 20s, 40s)
- **After 3 failures**: State transitions to `retry_exhausted`; manual retry available
- **Non-transient failures**: Failures that are not retryable (e.g., malformed input, missing meeting) should fail immediately without retrying

### Idempotency in Retry

Each processing run must be idempotent at the meeting level. If a job is picked up while a prior run for the same meeting is still in-flight:
- If the prior run is `processing` and exceeds a staleness threshold (e.g., 10 minutes), it can be marked `failed` and the new run proceeds.
- The new run creates a new version (never overwrites the in-flight version).

### Reprocess vs Retry

| Term | Meaning |
|------|---------|
| Retry | Re-attempt a failed job for the same source version |
| Reprocess | Process after a source edit or prior-memory input change (new source version) |

### State Machine

```
not_processed
    │ (import with FF=true)
    ▼
queued
    │
    ▼
processing ─────────────────────────────────────────┐
    │                                             │
    ├──► completed ────────────────────────────────►│ (becomes active)
    │                                                 │
    │                                             ▼
    ├──► completed_with_insufficient_evidence     active (stable)
    │
    ├──► failed ──► retrying ──► processing (up to 3x)
    │                   │
    │                   └──► retry_exhausted ──► manual_retry ──► processing
    │
    └──► stale ──► reprocessing ──► processing
```

### What could go wrong
- **Retry storm**: If the processing provider is down, retries pile up. Queue depth must be monitored. Exponential backoff prevents thundering herd.
- **Infinite retry on non-transient error**: The processor must classify errors as retryable vs non-retryable; only retryable errors count against the retry budget.
- **Stale active memory during long outage**: If the processor is unavailable for extended periods, meetings remain in `queued` or `stale` state. This is acceptable per BRD-03 AC-22 — existing memory remains viewable.

---

## Focus Area 5: Conflict Queue

### What Enters the Conflict Queue

A conflict is created when:
1. Current processing extracts a memory item (decision, action item, etc.)
2. Prior memory (used as input) contains a related item
3. Both items have source evidence
4. The items **cannot be automatically reconciled** — they are factually contradictory

Example: Prior memory says "Q3 launch date is September 15." Current meeting transcript says "We're pushing the launch to October 1." Both have evidence snippets supporting each date.

### Conflict Data Model

```json
{
  "conflictId": "<uuid>",
  "category": "decisions | actionItems | risksBlockers | openQuestions",
  "itemRefCurrent": "<uuid in current memory version>",
  "itemRefPrior": "<uuid in prior memory version>",
  "conflictType": "contradiction | change | reversal",
  "priorEvidence": [ ... ],
  "currentEvidence": [ ... ],
  "status": "pending | resolved",
  "resolutionNote": "<string | null>",
  "resolvedBy": "<uuid | null>",
  "resolvedAt": "<ISO8601 | null>"
}
```

### Conflict Resolution Flow

1. User sees conflicts in the review queue (BRD-03 US-6)
2. User reads both items and their evidence
3. User submits a resolution:
   - **Evidence-backed resolution**: User writes a note citing current/prior evidence
   - **Insufficient evidence resolution**: User explicitly states evidence is insufficient to resolve
4. System creates a new memory version with `processing_trigger: conflict_resolution`
5. The conflict item is resolved; original conflicting sources are preserved for audit
6. `meeting_memory.conflict.reviewed` log event is emitted

### After Resolution

- The new version may have an updated item that reflects the resolution, or the item may be removed if the user determined evidence was insufficient.
- The conflict record is retained (not deleted) for audit.
- If resolution produced a new active version, briefing readiness is re-evaluated.

### What Could Go Wrong
- **False conflict detection**: Processor flags items as conflicting when they're actually complementary. Mitigation: conservative detection — only flag as conflict when evidence directly contradicts (not just differs).
- **Conflict never resolved**: Items remain in conflict queue indefinitely. They are excluded from briefing input but don't block other memory categories.
- **Resolution note contains sensitive content**: Resolution notes are user-generated content — they must be treated as sensitive data and not logged.

---

## Focus Area 6: Prior Memory Matching

### Safe Match Criteria (per BRD-03 FR)

Prior-memory matching is **constrained** to narrow, unambiguous signals:
- **Same participant**: At least one participant in the current meeting shares a displayName (or email, if available) with a participant in the prior meeting
- **Similar title**: Title similarity (e.g., exact match on normalized title, or high token overlap) — exact string match is safest for MVP
- **Close chronology**: Prior meeting completed within a configurable time window of current meeting (e.g., ±30 days)

### What Matching Must NOT Do (Out of Scope)

Per BRD-03: "Keep prior-memory matching narrower than the future Related Meeting Detection BRD; broad opaque matching, provider-based relationship inference, and cross-system relatedness are out of scope."

This means:
- No semantic embedding similarity
- No cross-system relationship inference (e.g., "same email domain")
- No opaque ML-based relatedness scores

### Matching Output

For each prior memory used as input:
```json
{
  "priorMemoryInputs": [
    {
      "memoryId": "<uuid>",
      "meetingId": "<uuid>",
      "memoryVersion": "<integer>",
      "matchSignal": "same_participant | similar_title | close_chronology | combination",
      "matchConfidence": "high | medium | low",
      "included": true,
      "excluded": false,
      "excludedBy": null,
      "excludedAt": null
    }
  ]
}
```

### User Override

Users can:
1. **Exclude** an incorrectly matched prior memory — emits `meeting_memory.prior_inputs.updated` log (counts only, no content)
2. **Include** a missing relevant prior memory — same log event
3. **Reprocess** with the updated inputs

### What Could Go Wrong
- **Wrong match gets included**: User sees the prior memory in the processing inputs list and can exclude it. This is the safety valve.
- **Matching too narrow**: Some relevant prior memories are missed. This is a recall problem, not a safety problem — users can manually include missing memories.
- **Participant name ambiguity**: "John" in meeting A and "John" in meeting B might be different people. The match signal should be surfaced (not just "matched"), and users can exclude.

---

## Focus Area 7: Briefing-Readiness Signal

### Definition of Briefing Ready

Per BRD-03 FR: "Mark memory as briefing-ready only when summary, decisions, action items, risks/blockers, and open questions have strong or weak evidence and no unresolved conflict blocks those core categories."

```
briefingReady = true WHEN:
  summary.qualityStatus IN (strong_evidence, weak_evidence)
  AND decisions.qualityStatus IN (strong_evidence, weak_evidence)
  AND actionItems.qualityStatus IN (strong_evidence, weak_evidence)
  AND risksBlockers.qualityStatus IN (strong_evidence, weak_evidence)
  AND openQuestions.qualityStatus IN (strong_evidence, weak_evidence)
  AND conflictQueue WHERE status=pending AND blocking=true IS EMPTY
```

Note: `stakeholderNotes` and `nextRecommendedFocus` being insufficient does **not** block core briefing readiness (BRD-03 FR: "Allow stakeholder notes and next recommended focus to be insufficient without automatically blocking core briefing-readiness").

### Briefing Ready States (User-Facing)

| State | Condition |
|-------|-----------|
| `ready` | All core categories strong/weak, no blocking conflicts |
| `ready_with_weak` | All core categories strong/weak, at least one is weak, no blocking conflicts |
| `ready_with_insufficient` | Stakeholder notes and/or next recommended focus insufficient; core categories ready |
| `needs_review` | At least one unresolved conflict in conflictQueue that blocks core categories |
| `processing` | Currently running |
| `reprocessing` | Source edited, reprocessing queued/running |
| `failed` | Processing failed and exhausted retries |
| `retry_exhausted` | Processing exhausted retries, manual retry available |
| `stale` | Source edited, reprocessing not yet complete |

### Briefing Input Eligibility

A memory item (or meeting) is eligible for downstream briefing generation (BRD-04 Pre-Call Briefing) when:
1. `briefingReady = true`
2. The item itself has `qualityStatus` of `strong_evidence` or `weak_evidence`
3. No unresolved conflicts reference that item

Unresolved conflicts are **excluded from briefing input** (BRD-03 FR: "Exclude unresolved conflicting evidence from downstream briefing input").

### What Could Go Wrong
- **Briefing includes weak evidence without flagging it**: BRD-04 must surface qualityStatus of items it uses in briefings so users can calibrate trust.
- **Briefing ready but quality is actually poor**: The qualityStatus system is the mitigation — `weak_evidence` items are included but flagged. Users must see the quality status in the briefing UI.
- **Stale memory fed to briefing**: `briefingReady = false` when `status = stale`. BRD-04 must check this flag before using a meeting's memory.

---

## Cross-Cutting: Privacy and Observability

### Privacy Requirements (Non-Negotiable)

The following must **never** appear in logs or metrics:
- Raw transcript text
- Raw notes text
- Evidence snippets
- Participant PII (names, emails)
- Stakeholder note content
- Generated memory text
- Prior memory content
- Resolution note text

Correlation IDs (UUIDs) are safe to log. Meeting IDs, memory version IDs are safe to log as they are internal identifiers — not user-facing PII.

### Observability Summary

| Metric/Event | Purpose |
|---|---|
| `cp_meeting_memory_processing_queued_total` | Monitor queue depth and trigger types |
| `cp_meeting_memory_processing_started_total` | Processing throughput |
| `cp_meeting_memory_processing_completed_total` | Success rate |
| `cp_meeting_memory_processing_failed_total` | Failure rate by safe class |
| `cp_meeting_memory_processing_retried_total` | Retry pressure |
| `cp_meeting_memory_processing_retry_exhausted_total` | Exhaustion events |
| `cp_meeting_memory_processing_duration_ms` | Latency histogram |
| `cp_meeting_memory_processing_queue_wait_ms` | Queue wait histogram |
| `cp_meeting_memory_active_version_changed_total` | Version churn |
| `meeting_memory.processing.queued` | Correlation ID + safe trigger + source version |
| `meeting_memory.processing.started` | Correlation ID + attempt ID + version candidate ID |
| `meeting_memory.processing.completed` | Correlation ID + version ID + completion state |
| `meeting_memory.processing.failed` | Correlation ID + safe failure class + retryability |
| `meeting_memory.processing.retrying` | Correlation ID + attempt number + backoff class |
| `meeting_memory.processing.retry_exhausted` | Correlation ID + safe failure class |
| `meeting_memory.version.activated` | Correlation ID + prior version ID + new version ID |
| `meeting_memory.source.stale` | Correlation ID + meeting ID + source version + edited field |
| `meeting_memory.prior_inputs.updated` | Correlation ID + counts only (no content) |
| `meeting_memory.conflict.reviewed` | Correlation ID + conflict ID + resolution status (no text) |

---

## Component Inventory

| Component | Responsibility | New API Endpoints |
|-----------|---------------|-------------------|
| **Queue enqueuer** | On meeting save/edit/manual-retry, writes job to queue | None (internal) |
| **Queue worker** | Picks up jobs, coordinates processing | None (internal) |
| **Memory processor** | Extracts categories, evaluates evidence, detects conflicts | None (internal) |
| **Version manager** | Creates/manages memory versions, active pointer | None (internal) |
| **Conflict resolver** | Stores conflicts, processes resolutions | None (internal) |
| **Prior memory matcher** | Finds eligible prior memories using constrained signals | None (internal) |
| **Readiness evaluator** | Computes briefing-ready state | None (internal) |
| **Memory API** | Serves memory to client, surfaces conflict queue | `GET /meetings/{id}/memory`, `GET /meetings/{id}/memory/versions`, `POST /meetings/{id}/memory/retry`, `POST /meetings/{id}/memory/conflicts/{id}/resolve` |
| **Flag evaluator** | Server-side flag evaluation for processing gate | None (internal) |

### API Surface (New Endpoints Beyond BRD-02)

| Method | Path | Description | FF Gate |
|--------|------|-------------|---------|
| GET | `/meetings/{id}/memory` | Get active memory version | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| GET | `/meetings/{id}/memory/versions` | List all memory versions | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| GET | `/meetings/{id}/memory/versions/{version}` | Get specific version | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| POST | `/meetings/{id}/memory/retry` | Manual retry/reprocess | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| POST | `/meetings/{id}/memory/prior-memories` | Update prior-memory inputs + reprocess | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| GET | `/meetings/{id}/memory/conflicts` | Get conflict queue | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| POST | `/meetings/{id}/memory/conflicts/{id}/resolve` | Submit conflict resolution | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |

---

## Feature Flag

Dual-namespace flag: `FF_ENABLE_MEETING_MEMORY_PROCESSING` (server) / `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` (browser).

Server flag is authoritative for all data mutations and processing. Browser flag gates visible UI (memory view, retry button, conflict queue, prior-memory include/exclude).

Flag behavior table matches BRD-03 FR exactly (see BRD-03 Feature Flag section, lines 176–191).

Note from Open Questions: Curation may split into separate flags for automatic processing, memory UI, and review/reprocess actions. This must be resolved before implementation.

---

## Constraints and Non-Goals (from BRD-03)

### In Scope
- Async processing after BRD-02 import
- All 7 memory categories with evidence
- Version history
- Retry/reprocess with idempotency
- Conflict queue with user resolution
- Constrained prior-memory matching
- Briefing-readiness signal
- Observability without sensitive content
- Provider-agnostic product behavior

### Out of Scope
- Synchronous processing
- Opaque ML-based related-meeting detection (deferred to future BRD)
- Stakeholder note personality profiling
- Global contacts/CRM
- Upcoming meeting creation
- Pre-call briefing generation (BRD-04)
- Retention/deletion (BRD-06)
- Manual memory correction beyond conflict resolution (BRD-07)

---

## Open Issues Found in Review

| # | Issue | Severity | Recommendation |
|---|-------|----------|----------------|
| 1 | ADRs 0005 and 0006 (architecture, data model) are listed as parent task outputs but are not yet finalized — this systematic review depends on them | High | Block on their completion; if they diverge from the patterns described here, this review must be updated |
| 2 | Exact source-locator format (char_offset vs line_ref) is unresolved — impacts evidence schema | Medium | Resolve in ADR-0006 curation before implementation |
| 3 | Exact processing SLO is TBD — architecture cannot give hard latency guarantees until provider/runtime selected | Medium | ADR-0005 must define degradation behavior independent of specific latency targets |
| 4 | Feature flag split (automatic processing vs UI vs review) is unresolved — impacts API surface and rollout safety | Medium | Resolve in feature-flags.md curation; assume single flag for MVP scope |
| 5 | Quality distribution metrics (category-level quality trend analytics) are deferred to curation — data model should not preclude adding them later | Low | Store `qualityStatus` as enum on each item; aggregate counts are derived, not stored separately |
| 6 | Health/readiness probe behavior when worker/provider is degraded but app is healthy needs explicit definition — risk of whole-app outage for async feature | Medium | ADR-0005 must define `/ready` behavior: app returns 200 if it can queue jobs and serve existing memory, regardless of worker/provider availability |

---

## Red Flags Checklist

| Check | Status |
|-------|--------|
| Solution bias (describing HOW not WHAT) | None — BRD defines WHAT; this review maps HOW |
| Only happy paths | Covered — failure, stale, retry-exhausted all documented |
| God object / unclear separation | None — components have clear boundaries |
| Hand-waving complexity | None — each component has defined responsibility |
| Missing privacy boundary | None — privacy requirements enumerated explicitly |
| Undocumented edge cases | Covered — hallucination, evidence mismatch, conflict false positives, stale active memory |
| Figure-it-out-later on critical decisions | Flagged as Open Issues — 6 items require curation decisions before implementation |

---

## Dependencies

- **Blocking**: Parent task t_021bf175 (ADR creation) must complete ADRs 0005 and 0006
- **Prerequisite**: BRD-02 meeting record, participants, and source storage (complete)
- **Blocked by**: BRD-03 does not block any other work in this review cycle, but it is a prerequisite for BRD-04 Pre-Call Briefing

---

*End of systematic review. This document is a working draft until ADRs 0005/0006 are confirmed and any divergences are resolved.*
