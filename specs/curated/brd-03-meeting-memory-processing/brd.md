# BRD-03 Meeting Memory Processing — Curated

---
status: Curated
brd_id: brd-03
title: Meeting Memory Processing
author: pm
created: 2026-05-20
curated: 2026-05-21
reviewers: refiner (t_a68bc12e), architect (t_a5b098df)
adrs: ADR-0005 (queue architecture), ADR-0006 (data model), ADR-0007 (prior memory matching + conflict detection), ADR-0008 (observability + privacy)
parent: brd-02-manual-meeting-import
blocks: brd-04-pre-call-briefing, brd-06-privacy-retention, brd-07-manual-memory-correction
feature_flag: ff_enable_meeting_memory_processing
phase: Phase 2

---

## Overview

Meeting Memory Processing transforms a saved meeting into durable, source-grounded memory that can power future pre-call briefings. It processes transcript text, notes, meeting metadata, participants, and safely matched prior memories into structured categories with evidence, quality states, version history, and briefing-readiness signals.

This feature matters because ContextPilot's core promise depends on trusted memory, not merely generated summaries. Users must be able to verify what the system remembered, understand when evidence is weak or conflicting, and prevent unsupported or unresolved information from flowing into future briefings.

Processing is asynchronous and non-blocking: manual meeting save and meeting detail navigation complete without waiting for memory processing. Existing active memory remains available during reprocessing, retry exhaustion, or worker unavailability.

---

## User Stories

| ID | As a... | I want to... | So that... |
|----|---------|--------------|------------|
| US-1 | Authenticated user | My imported meeting to be automatically processed into structured memory | I can get value from a saved meeting without remembering a separate processing step |
| US-2 | Authenticated user | Every memory item to include source evidence | I can verify that ContextPilot is not inventing unsupported facts |
| US-3 | Authenticated user | A complete memory structure even when evidence is weak | I can see which categories are useful and which are insufficient instead of wondering whether data is missing |
| US-4 | Authenticated user | The memory view to show briefing-readiness | I can understand whether this meeting can safely power a future pre-call briefing |
| US-5 | Authenticated user | Prior memories used during processing to be visible and editable | I can remove wrong matches, add missing context, and reprocess with the right inputs |
| US-6 | Authenticated user | Conflicting evidence to appear in a review queue | I can resolve conflicts before they influence future briefings |
| US-7 | Authenticated user | Failed, stale, or retry-exhausted processing to be retryable | I can recover from transient processing problems or source edits |
| US-8 | Product/operator | Observe processing lifecycle, retries, failures, and active version changes | I can diagnose system reliability without exposing sensitive meeting content |

---

## User Story Acceptance Criteria

> Each user story is verified through concrete Given/When/Then scenarios. These are the acceptance criteria for each user story.

### US-1: Automatic Processing

**Given** an authenticated user has saved a meeting via BRD-02 with `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`
**When** the `POST /meetings` response returns 201 Created
**Then** a memory processing job is inserted into `memory_processing_jobs` with `trigger_type='import'` and `status='queued'`; the redirect to `/meetings/{id}` completes without waiting for processing; and the meeting detail page shows memory in `queued` state

### US-2: Evidence on Every Item

**Given** an authenticated user owns a meeting with transcript text and notes, and memory processing completes successfully
**When** the user retrieves `GET /meetings/{id}/memory`
**Then** every memory item in every category contains at least one evidence record with a non-empty `evidenceSnippet`, a `sourceType` in `{transcript, notes, prior_memory_reference}`, and a `sourceLocation` of type `char_offset` with valid `start` and `end` integers

### US-3: All Categories Present with Weak Evidence

**Given** an authenticated user owns a meeting where the transcript contains fewer than 200 characters covering only a general topic
**When** memory processing completes
**Then** all 7 categories are present in the memory output; categories without supporting evidence show `quality_status: insufficient_evidence`; no category is omitted; and no category is filled with synthesized content unsupported by the source

### US-4: Briefing-Readiness Signal

**Given** an authenticated user owns a meeting with completed memory processing
**When** the user retrieves `GET /meetings/{id}/memory`
**Then** the response includes a `briefing_readiness` block with `ready: true|false`, `core_categories_ready`, `blocking_conflicts`, and `insufficient_categories`; and the briefing-readiness state matches the definition in FR-15 (ready only when core categories have evidence and no unresolved conflicts block them)

### US-5: Prior Memories Visible, Content Protected

**Given** an authenticated user owns a meeting that was processed with prior memories as input
**When** the user retrieves `GET /meetings/{id}/memory`
**Then** the response includes the `prior_memory_inputs` array showing each prior memory's meeting ID, match confidence, and user inclusion/exclusion status; no prior memory raw content (decision text, summary statements, or action items) appears in any log event or metric label

**And** the user can navigate to an include/exclude control for each prior memory input
**And** toggling exclusion or inclusion and submitting `POST /meetings/{id}/memory/reprocess` queues a new processing run that creates a new memory version

### US-6: Conflicts in Review Queue

**Given** an authenticated user owns two meetings, M1 and M2, where M2 was processed with M1 as a prior memory and a decision in M2 contradicts the corresponding decision in M1 with both having strong or weak evidence
**When** processing of M2 completes
**Then** a conflict record appears in `memory_conflicts` with `review_status: pending`; the conflict's `conflicting_items` contains both the current and prior item content with evidence; and the memory categories for M2 exclude the conflicting item while it remains unresolved

### US-7: Manual Retry Available

**Given** an authenticated user owns a meeting whose memory is in `failed`, `retry_exhausted`, or `stale` state
**When** the user clicks the retry or reprocess action on the meeting memory view
**Then** `POST /meetings/{id}/memory/reprocess` is called with the current prior-memory inputs; the meeting's memory state transitions to `queued`; and a new processing job appears in `memory_processing_jobs`

### US-8: Observability Without Sensitive Content

**Given** memory processing has run for a meeting
**When** an operator inspects emitted logs and metrics
**Then** `memory.job.queued`, `memory.job.started`, `memory.job.completed`, `memory.job.failed`, `memory.job.retrying`, `memory.job.retry_exhausted`, `memory.version.activated`, `memory.conflict.detected`, `memory.conflict.resolved`, `memory.source.stale`, and `memory.prior_inputs.updated` events are present; and raw transcript text, raw notes text, evidence snippets, participant PII, stakeholder note content, generated memory text, prior memory content, and resolution note text do not appear in any log field or metric label

---

## Functional Requirements

### Must Have

**FR-1: Automatic Processing Queue**

On successful manual meeting import via BRD-02 and when `FF_ENABLE_MEETING_MEMORY_PROCESSING` is `true`, the system inserts a memory processing job row after the meeting transaction commits. The `POST /meetings` response returns without waiting for processing completion. The meeting detail page shows the memory in `queued` state. This does not block import redirect or meeting detail access.

**FR-2: Memory Categories**

Every completed processing result preserves all seven categories in the memory output, even when evidence is insufficient for a category. Categories are: `summary`, `decisions`, `action_items`, `risks_blockers`, `open_questions`, `stakeholder_notes`, `next_recommended_focus`.

**FR-3: Insufficient Evidence Handling**

A category is marked `Insufficient evidence` when the source material does not support useful content for that category. The system does not fill weak categories with guesses or synthesized content not grounded in source evidence.

**FR-4: Evidence Requirements**

Every supported memory item includes an `evidence` array containing one or more evidence records. Each evidence record contains: a non-empty `evidenceSnippet`, a `sourceType` from `{transcript, notes, prior_memory_reference}`, and a `sourceLocation` object of type `{type: "char_offset", start: integer, end: integer}`. The `source_location` uses character offsets (rune indices) into the stored transcript or notes text. Offsets are validated: `0 <= start < end <= len(source_text)`. Offsets are computed against the stored (possibly whitespace-normalized) text, not the raw pasted input.

When a source meeting is edited, all evidence offsets for that meeting are invalidated: the active memory version is marked `stale`, and reprocessing is queued automatically (FR-10). Source locations must never appear in logs or metrics.

**FR-4a: Evidence Snippet Display Sanitization**

Evidence snippets extracted from transcript text and notes are stored as plain text. Before any evidence snippet is rendered in the memory view or in any API response that exposes it for client-side display, HTML special characters — specifically `<`, `>`, `&`, double quotes (`"`), and single quotes (`'`) — must be escaped. This prevents injected HTML or script content within source meeting text from executing in the browser. Allowlist sanitization via a maintained sanitizer such as DOMPurify, or an equivalent framework-provided HTML escaper, is required when any rich text rendering capability is later introduced. Until then, HTML escaping of evidence snippets is the minimum required protection.

**FR-5: Prior Memory Matching — Constrained, User-Visible**

Automatic prior-memory matching is constrained to three narrow, verifiable signals applied together (AND semantics):

- **Same participant**: At least one participant in the current meeting shares an exact email with a participant in the prior meeting.
- **Similar title**: Levenshtein distance between normalized titles is 3 or less, or titles share a common token sequence of 4 or more consecutive words.
- **Close chronology**: Prior meeting `completed_at` is within 90 days before or 7 days after the current meeting's `completed_at`.

The match confidence is `safe_match` when all three signals pass at full criteria. The match confidence is `uncertain` when only same-participant and close-chronology pass (title similarity fails) — uncertain matches are shown to the user for confirmation before use as processing input.

Users can exclude any system-matched prior memory or manually add an eligible prior memory. Excluded or manually added prior memories trigger reprocessing. The prior memories used as input are visible to the user without exposing raw content in UI or logs.

Broad opaque matching (embedding similarity, provider-based relationship inference, cross-system relatedness) is out of scope.

**FR-6: Decision Tracking**

Decisions include a `changeStatus` field with value from {`standalone`, `confirms_prior`, `changes_prior`, `reverses_prior`}. A decision is marked `confirms_prior`, `changes_prior`, or `reverses_prior` only when a prior memory is eligible input (per FR-5) and the current meeting evidence supports that relationship with the prior decision. `standalone` means no prior memory item in the same category with an overlapping subject.

**FR-7: Stakeholder Notes**

Stakeholder notes are limited to meeting-relevant business context. Each note includes `participant_id` (reference to a meeting participant from BRD-02), `preferences`, `concerns`, `commitments`, `influence_stake`, `evidence`, and `quality_status`. Sensitive personal attributes, personality profiling, psychological inference, and unsupported influence assumptions are out of scope.

**FR-8: Next Recommended Focus**

Next recommended focus is synthesized from supported memory categories and includes evidence links to source snippets or source-grounded memory items. It is marked `Insufficient evidence` when supported categories do not provide sufficient basis.

**FR-9: Version Management**

Each processing or reprocessing run creates a new memory version with a monotonically increasing version number per meeting. The latest successful version is `active`. Unresolved conflicting evidence is placed in a separate conflict review queue and excluded from normal memory categories and downstream briefing input. Failed processing runs do not replace the active successful memory version.

**FR-10: Stale Detection and Reprocessing**

When the meeting's `transcript`, `notes`, `participants`, `title`, `completed_at`, or other source metadata changes after memory has been processed, the active memory version is marked `stale` and reprocessing is queued. The previous active version remains accessible and serves as the briefing source until the new version succeeds. Stale detection uses a source version hash stored on each memory version; when the meeting's `updated_at` advances past the `memory_versions.created_at` of the active version, staleness is detected.

**FR-11: Processing Lifecycle States**

The following states are exposed: `not_processed`, `queued`, `processing`, `completed`, `completed_with_insufficient_evidence`, `failed`, `stale`, `reprocessing`, `retrying`, `retry_exhausted`.

**FR-12: Conflict Detection**

A conflict is detected when the current processing run extracts a memory item in `decisions`, `action_items`, `risks_blockers`, or `summary` where the matched prior memory contains a corresponding item in the same category, both items have `strong_evidence` or `weak_evidence`, and the items are semantically contradictory to a degree that cannot be reconciled automatically.

The semantic similarity threshold for conflict detection is defined by the processing provider's semantic similarity model. The `MemoryProcessor` implementation returns `Conflicting evidence` when similarity falls below the merge threshold for item pairs that both have `strong_evidence` or `weak_evidence` in the same category. The exact cosine-similarity or embedding-distance threshold is provider-specific and calibrated during Phase 2 eval.

Conflicting items are placed in a `memory_conflicts` table with `review_status = pending` and excluded from the normal memory categories and from downstream briefing eligibility. An item with `Insufficient evidence` cannot participate in a conflict.

**FR-13: Conflict Resolution**

A dedicated `/meetings/{id}/memory/conflicts` route shows pending conflicts. Each conflict shows: the conflicting evidence snippets with source attribution, the affected memory category, a free-text resolution note field, and a "Mark Reviewed" button. The resolution note may reference specific evidence by source ID but is not required to select a winner — the user may also state that evidence is insufficient to resolve.

Resolving emits `meeting_memory.conflict.reviewed` and creates a new memory version with `processing_trigger: conflict_resolution`. The original conflicting sources and resolution note are preserved for audit. Resolving one conflict resolves that conflict; the atomic unit is a single conflict, not all conflicts for a meeting.

**FR-13a: Resolution Note Display Sanitization**

Resolution notes are user-authored free text that may include content pasted from various sources. Before any resolution note is rendered in the memory view or in any API response that exposes it for client-side display, HTML special characters — specifically `<`, `>`, `&`, double quotes (`"`), and single quotes (`'`) — must be escaped. This prevents user-authored resolution notes from injecting HTML or script content into the page. Allowlist sanitization via a maintained sanitizer such as DOMPurify, or an equivalent framework-provided HTML escaper, is required when any rich text rendering capability is later introduced. Until then, HTML escaping of resolution notes is the minimum required protection.

**FR-14: Quality Statuses**

Quality statuses are applied at both item level and category level: `strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence`. A category has the worst quality status of any item within it.

**FR-15: Briefing Readiness**

Memory is briefing-ready when all five core categories (summary, decisions, action_items, risks_blockers, open_questions) have `strong_evidence` or `weak_evidence` and no unresolved conflict blocks those core categories. Stakeholder notes and next recommended focus being insufficient does not block core briefing readiness.

The briefing-readiness signal states are: `ready`, `ready_with_weak`, `ready_with_insufficient`, `needs_review`, `processing`, `reprocessing`, `failed`, `retry_exhausted`, `stale`.

**FR-16: Failure Behavior**

Transient processing failures retry up to 3 times with exponential backoff: initial delay 1 second, multiplier 2x, maximum delay 30 seconds, jitter of plus or minus 10 percent. After 3 failed retries the job enters `retry_exhausted` state. Non-transient failures (malformed input, missing meeting) fail immediately without retry.

When the processing worker or provider is unavailable, jobs stay `queued`; existing active memory remains accessible; the meeting detail shows `queued` state. No user content is lost and no false failure state is shown.

**FR-17: Provider Agnosticism**

This BRD defines product behavior, evidence requirements, state model, eval expectations, privacy constraints, and observability — not a specific model, vendor, or runtime. The processing worker uses a `MemoryProcessor` interface; concrete implementations are injected.

**FR-18: Strict Evidence Eval**

Eval fixtures verify that tempting but unsupported claims are omitted or marked `Insufficient evidence`. Every supported item must have evidence that actually supports the item; evidence-grounding eval verifies item/evidence consistency.

**FR-19: Feature Flags**

Gate processing behavior behind `FF_ENABLE_MEETING_MEMORY_PROCESSING` with default `false`. Gate browser-visible memory UI, retry/reprocess actions, review queue, and memory state indicators behind `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` with default `false`. The server flag is authoritative for all data mutations.

**FR-20: Privacy**

Raw transcript text, raw notes text, evidence snippets, participant PII, stakeholder note content, generated memory content, prior memory content, and resolution note text must not appear in logs or metric labels. Source locations must never appear in logs or metrics.

**FR-21: Manual Retry**

Provide a manual retry/reprocess action for meetings in `failed`, `retry_exhausted`, `stale` states or after user corrections to prior-memory inputs or source content.

---

## API Contract

All memory API endpoints require an authenticated session (`Authorization: Bearer ***` or `session` cookie). A user may only access memory for meetings they own or that are visible via standard meeting ACL. All endpoints are gated by `FF_ENABLE_MEETING_MEMORY_PROCESSING`; when `false`, endpoints return 403 with a safe error message.

| Method | Path | Summary |
|--------|------|---------|
| GET | `/meetings/{id}/memory` | Active memory version with content, briefing readiness, prior-memory inputs |
| GET | `/meetings/{id}/memory/versions` | Version history list |
| GET | `/meetings/{id}/memory/versions/{versionNumber}` | Specific version |
| POST | `/meetings/{id}/memory/reprocess` | Manual retry or reprocess with optional prior-memory include/exclude |
| GET | `/meetings/{id}/memory/state` | Current processing lifecycle state |
| GET | `/meetings/{id}/memory/conflicts` | Pending conflict review queue |
| POST | `/meetings/{id}/memory/conflicts/{conflictId}/resolve` | Submit resolution note, create new version |

### POST /meetings Integration Note

On successful manual meeting save (per BRD-02), the handler inserts a `memory_processing_jobs` row with `trigger_type='import'` after the transaction commits. The redirect does not wait for processing.

---

## Data Model

### memory_versions

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `meeting_id` | UUID FK | Yes | References `meetings(id)` ON DELETE CASCADE |
| `job_id` | UUID FK | No | References `memory_processing_jobs(id)` ON DELETE SET NULL |
| `version_number` | integer | Yes | Monotonically increasing per meeting |
| `status` | text | Yes | One of: `active`, `superseded`, `conflict_review` |
| `is_active` | boolean | Yes | Exactly one active version per meeting |
| `content` | JSONB | Yes | Full structured memory output (see Memory JSON Schema below) |
| `created_at` | timestamptz | Yes | Default now() |
| `created_by` | UUID | No | User who triggered processing |
| `trigger_type` | text | Yes | One of: `import`, `reprocess`, `stale_reprocess`, `manual_retry`, `conflict_resolution` |

UNIQUE constraint on `(meeting_id, version_number)`. Index on `(meeting_id, is_active)` WHERE `is_active = TRUE`.

### memory_evidence

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `memory_version_id` | UUID FK | Yes | References `memory_versions(id)` ON DELETE CASCADE |
| `category` | text | Yes | Category name: `summary`, `decisions`, etc. |
| `item_id` | text | Yes | Stable item ID within the JSON content |
| `source_type` | text | Yes | One of: `transcript`, `notes`, `prior_memory_reference` |
| `source_location` | JSONB | Yes | `{type: "char_offset", start: integer, end: integer}` |
| `evidence_snippet` | text | Yes | Source text excerpt |
| `quality_status` | text | Yes | One of: `strong_evidence`, `weak_evidence`, `insufficient_evidence` |

Index on `memory_version_id`.

### memory_conflicts

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `meeting_id` | UUID FK | Yes | References `meetings(id)` ON DELETE CASCADE |
| `memory_version_id` | UUID FK | No | References `memory_versions(id)` ON DELETE CASCADE |
| `conflicting_items` | JSONB | Yes | `{current: [...], prior: [...]}` — the conflicting item content from both sides |
| `quality_status` | text | Yes | Default `conflicting_evidence` |
| `review_status` | text | Yes | One of: `pending`, `reviewed`; default `pending` |
| `resolution_note` | text | No | User-provided resolution text |
| `resolved_at` | timestamptz | No | Set when `review_status` becomes `reviewed` |
| `resolved_by` | UUID | No | User who resolved |
| `created_at` | timestamptz | Yes | Default now() |

Index on `(meeting_id)`. Index on `(review_status)` WHERE `review_status = 'pending'`.

### memory_prior_memory_inputs

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `memory_version_id` | UUID FK | Yes | References `memory_versions(id)` ON DELETE CASCADE |
| `prior_memory_version_id` | UUID FK | Yes | References `memory_versions(id)` ON DELETE RESTRICT |
| `match_confidence` | text | No | One of: `safe_match`, `uncertain` |
| `included_by_user` | boolean | Yes | Default FALSE — TRUE when user manually added this prior memory |
| `excluded_by_user` | boolean | Yes | Default FALSE — TRUE when user excluded this prior memory |
| `created_at` | timestamptz | Yes | Default now() |

UNIQUE constraint on `(memory_version_id, prior_memory_version_id)`.

### Memory JSON Content Schema

The `content` JSONB column stores the structured memory output per version:

```json
{
  "summary": {
    "statement": "string",
    "quality_status": "strong_evidence | weak_evidence | insufficient_evidence",
    "items": []
  },
  "decisions": {
    "items": [
      {
        "id": "string",
        "decision_statement": "string",
        "change_status": "standalone | confirms_prior | changes_prior | reverses_prior",
        "prior_decision_ref": "uuid | null",
        "quality_status": "string",
        "evidence": []
      }
    ]
  },
  "action_items": {
    "items": [
      {
        "id": "string",
        "description": "string",
        "owner": "string | null",
        "owner_specified": true | false,
        "due_date": "ISO 8601 date | null",
        "due_date_specified": true | false,
        "status": "pending | in_progress | completed | deferred",
        "quality_status": "string",
        "evidence": []
      }
    ]
  },
  "risks_blockers": {
    "items": [
      {
        "id": "string",
        "type": "risk | blocker",
        "description": "string",
        "quality_status": "string",
        "evidence": []
      }
    ]
  },
  "open_questions": {
    "items": [
      {
        "id": "string",
        "question": "string",
        "quality_status": "string",
        "evidence": []
      }
    ]
  },
  "stakeholder_notes": {
    "items": [
      {
        "id": "string",
        "participant_id": "uuid",
        "preferences": "string | null",
        "concerns": "string | null",
        "commitments": "string | null",
        "influence_stake": "string | null",
        "quality_status": "string",
        "evidence": []
      }
    ]
  },
  "next_recommended_focus": {
    "statement": "string",
    "quality_status": "string",
    "supporting_item_ids": ["string"],
    "evidence": []
  },
  "briefing_readiness": {
    "ready": true | false,
    "core_categories_ready": true | false,
    "blocking_conflicts": [],
    "insufficient_categories": []
  }
}
```

Each `evidence` array entry contains:

```json
{
  "snippet": "string",
  "source_type": "transcript | notes | prior_memory_reference",
  "source_location": {"type": "char_offset", "start": 1234, "end": 5678},
  "source_meeting_id": "uuid"
}
```

---

## Non-Functional Requirements

| Type | Requirement | Measurement |
|------|-------------|--------------|
| Processing latency | **30 seconds at the 95th percentile** for a 50,000-character transcript-plus-notes input under normal load. This target is provider-agnostic; provider-specific SLO calibration will be updated after Phase 2 provider selection. Measured from job pick-up to completion, not including queue wait time. Queue wait time tracked separately via `cp_meeting_memory_processing_queue_wait_ms`. | Eval measures end-to-end job duration histogram |
| Import path latency | Successful manual meeting import and redirect must not wait on memory processing completion | E2E eval: import redirect completes before processing done |
| Availability | Manual import and existing meeting detail access continue when memory processing is unavailable; degradation follows queued/retry/retry-exhausted behavior | Degradation eval |
| Retry behavior | Up to 3 automatic retries with exponential backoff (initial delay 1s, multiplier 2x, max delay 30s, jitter ±10%) for transient failures, followed by retry-exhausted and manual retry | Job lifecycle integration eval |
| Evidence integrity | Every supported item has evidence snippet plus stable source location; unsupported claims omitted or marked insufficient evidence | Strict hallucination eval |
| Privacy | Raw transcript text, notes text, evidence snippets, participant PII, stakeholder note content, generated memory, prior memory content, and resolution note text must not appear in logs or metric labels | Observability eval |
| Retention | Memory versions and evidence snippets are retained with the source meeting until BRD-06 defines final retention, deletion, export, and redaction policy | Deferred to BRD-06 |
| Briefing safety | Unresolved conflicting evidence is excluded from downstream briefing input | E2E eval |
| Accessibility | Memory status, evidence access, retry/reprocess actions, and review queue interactions inherit BRD-01 WCAG 2.1 AA expectations | Accessibility eval |
| Provider portability | Product behavior and eval contracts do not depend on a specific processing provider, model, or runtime | Architecture review |

---

## Non-Goals

The following are explicitly out of scope for BRD-03:

- **Future meeting creation** — handled outside manual import; belongs to BRD-04 Pre-Call Briefing
- **Global contacts, participant deduplication, CRM-style relationship history, and identity resolution** — deferred to future BRDs
- **Broad opaque matching for prior-memory matching** — deferred to future Related Meeting Detection BRD; this includes embedding similarity, provider-based relationship inference, and cross-system relatedness
- **Provider-based relationship inference and cross-system relatedness** — deferred to future BRDs
- **Sensitive personal attributes, personality profiling, psychological inference, or unsupported influence assumptions in stakeholder notes** — excluded by FR-7
- **Specific model, vendor, or runtime** — BRD is provider-agnostic per FR-17
- **Real-time conflict detection from prior memory edits** — conflicts are detected at processing time; a conflict arising from a later edit to a prior memory is caught only when the affected meeting is reprocessed
- **HTML or rich-text rendering of evidence snippets or resolution notes** — evidence snippets are stored as plain text and escaped on display per FR-4a; resolution notes are escaped on display per FR-13a; full allowlist sanitization is deferred to a separate BRD if rich text rendering is later introduced

---

## Observability

### Metrics

| Metric | Type | Labels | Note |
|--------|------|--------|------|
| `cp_meeting_memory_processing_queued_total` | Counter | `trigger_type` | Incremented when a job row is inserted |
| `cp_meeting_memory_processing_started_total` | Counter | `trigger_type` | Incremented when worker picks up a job |
| `cp_meeting_memory_processing_completed_total` | Counter | `trigger_type`, `status` | Incremented on job completion |
| `cp_meeting_memory_processing_failed_total` | Counter | `trigger_type`, `failure_class` | Incremented on permanent failure |
| `cp_meeting_memory_processing_retried_total` | Counter | `trigger_type` | Incremented on automatic retry |
| `cp_meeting_memory_processing_retry_exhausted_total` | Counter | `trigger_type` | Incremented when max retries reached |
| `cp_meeting_memory_processing_duration_ms` | Histogram | — | Buckets: 100, 500, 1000, 5000, 10000, 30000, 60000, 120000 |
| `cp_meeting_memory_processing_queue_wait_ms` | Histogram | — | Time from job creation to worker pick-up |
| `cp_meeting_memory_active_version_changed_total` | Counter | `reason` | Incremented when active version pointer changes |
| `cp_meeting_memory_conflicts_detected_total` | Counter | `meeting_id` | meeting_id is the safe UUID, not content |
| `cp_meeting_memory_flag_evaluation_ms` | Histogram | — | Buckets: 1, 5, 10, 25, 50, 100 |

### Log events

| Event | When emitted | Allowed Fields |
|-------|-------------|---------------|
| `memory.job.queued` | Job row inserted | `correlation_id`, `meeting_id`, `trigger_type` |
| `memory.job.started` | Worker picks up job | `correlation_id`, `meeting_id`, `trigger_type` |
| `memory.job.completed` | Processing succeeds | All allowed fields |
| `memory.job.failed` | Processing returns permanent error | `correlation_id`, `meeting_id`, `trigger_type`, `failure_class`, `retry_count`, `duration_ms` |
| `memory.job.retrying` | Transient failure, retry scheduled | `correlation_id`, `meeting_id`, `trigger_type`, `retry_count`, `duration_ms`, `next_retry_at` |
| `memory.job.retry_exhausted` | Max retries reached | `correlation_id`, `meeting_id`, `trigger_type`, `retry_count`, `failure_class` |
| `memory.version.activated` | New active version set | `correlation_id`, `meeting_id`, `processing_version_number`, `active_version_change_reason` |
| `memory.conflict.detected` | Conflict placed in review queue | `correlation_id`, `meeting_id`, `category`, `conflict_id` |
| `memory.conflict.resolved` | User resolves conflict | `correlation_id`, `meeting_id`, `conflict_id`, `resolved_by` |
| `memory.source.stale` | Source edit detected | `correlation_id`, `meeting_id`, `edited_field` |
| `memory.prior_inputs.updated` | User includes/excludes prior memory | `correlation_id`, `meeting_id`, `included_count`, `excluded_count` |

### Redaction schema

Allowed log fields and metric labels: `correlation_id`, `meeting_id`, `trigger_type`, `failure_class`, `retry_count`, `duration_ms`, `processing_version_number`, `category_count`, `quality_distribution`, `active_version_change_reason`, `flag_evaluation_duration_ms`, `reason`.

Forbidden: raw transcript text, raw notes text, evidence snippets, participant names, participant emails, participant organizations, stakeholder note content, generated memory text, prior memory content, resolution note text, source locations, `id` fields from `memory_evidence`, `memory_conflicts`, or `memory_prior_memory_inputs` that could be correlated with content.

### Health/readiness

- `GET /ready` returns 200 when the app can evaluate feature flags and PostgreSQL is reachable, regardless of worker or provider availability.
- `GET /live` returns 200 when the process is alive.

---

## Feature Flag

- **Flag name:** `ff_enable_meeting_memory_processing`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_MEETING_MEMORY_PROCESSING`
- **Browser env:** `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING`

Dual namespace registration is confirmed in `specs/feature-flags.md` (Phase 2, Planned).

| Server env | Browser env | Expected behavior |
|------------|-------------|-------------------|
| `false` | `false` | Memory endpoints return 403; no processing queued on import |
| `true` | `true` | Authenticated users can access memory UI and processing is active |
| `true` | `false` | Server accepts authorized requests; browser UI remains hidden |
| `false` | `true` | Server denies all memory operations; this is a misconfiguration and is emitted as a metric and log event |

The server flag is authoritative for all data mutations. When disabled, all seven memory API endpoints return 403 with a safe error message; no memory content is disclosed.

---

## Security Considerations

- Transcript text, notes text, evidence snippets, participant values, stakeholder notes, generated memory content, prior-memory references, and resolution notes are treated as sensitive meeting-derived data.
- Raw meeting content and evidence snippets must not be written to logs or metric labels.
- Memory API endpoints use the same BearerAuth/CookieAuth as existing meeting endpoints; no separate auth scheme.
- Authorization is checked at the meeting level: a user may only access memory for meetings they own or that are visible via standard meeting ACL.
- Conflict resolution notes are stored per-user and not exposed to other meeting participants.
- Prior-memory matching is constrained to same-email, similar-title, close-chronology signals; no broad opaque matching that could infer cross-meeting relationships.

---

## Threat Model

| Threat | Likelihood | Impact | Mitigation |
|--------|------------|--------|------------|
| Processing invents unsupported memory (hallucination) | High | High | Require strict evidence for every item; `Insufficient evidence` states; strict hallucination eval fixtures |
| Evidence snippets appear trustworthy but do not support the item | Medium | High | Evidence-grounding eval that verifies item/evidence consistency for at least 10 fixture pairs |
| Stakeholder notes become sensitive profiling | Medium | High | Limit to source-supported meeting-relevant business context; exclude sensitive attributes, personality profiling, unsupported inference |
| Safe automatic prior-memory matching chooses wrong context | Medium | High | Show prior memories used; allow include/exclude and reprocess; keep matching constrained; uncertain matches require user confirmation |
| Conflicting evidence blocks useful briefing output | Medium | Medium | Isolate conflicts in review queue; allow evidence-backed resolution; non-conflicting memory remains active and briefing-eligible |
| Provider or worker outages prevent timely processing | Medium | Medium | Keep import non-blocking; queue/retry/retry-exhaust; preserve active memory; expose user-safe retry states |
| Version history grows without retention policy | Medium | Medium | Retain with source meeting; defer final retention/deletion/export to BRD-06 |
| Logs or metrics leak sensitive meeting-derived data | Medium | High | Explicitly ban raw content, evidence snippets, PII, generated memory, resolution text from logs/metrics; verify through observability eval |
| Browser/server flag drift exposes UI without processing support | Medium | Medium | Server flag is authoritative; misconfiguration emits safe observability; browser flag gates visible actions only |

---

## Dependencies

| Dependency | Type | Notes |
|------------|------|-------|
| BRD-02 Manual Meeting Import | Hard | Saved meetings, transcript/notes preservation, structured participants, meeting detail route, idempotency token |
| OpenAPI contract | Hard | `contracts/openapi.yaml` updated with 7 memory endpoints and Memory schemas |
| Feature flags registry | Hard | Both `FF_ENABLE_MEETING_MEMORY_PROCESSING` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` registered |
| BRD-04 Pre-Call Briefing | Blocks | Briefings require trusted, source-grounded memory from BRD-03 |
| BRD-06 Privacy & Retention Controls | Soft | Final retention/deletion/export policy for memory versions and evidence |
| BRD-07 Manual Memory Correction | Soft | Richer conflict-resolution workflows |
| Future: Related Meeting Detection | Soft | Broad prior-memory relatedness |

---

## Open Questions

| ID | Question | Owner | Resolution |
|----|----------|-------|------------|
| OQ-1 | Split feature flag or single dual-namespace flag? | Curator/PM | Single flag recommended; must confirm before Phase 2. Recommendation: single `FF_ENABLE_MEETING_MEMORY_PROCESSING` gates both server processing and browser UI, with the understanding that `true` server + `false` browser is a valid configuration for server-side testing. |
| OQ-2 | Exact processing latency SLO | Architect | **Resolved:** 30-second P95 target committed as provider-agnostic interim SLO. Provider-specific calibration deferred to Phase 2 implementation after provider/runtime selection (OQ-5). Queue wait time tracked separately via `cp_meeting_memory_processing_queue_wait_ms`. |
| OQ-4 | Quality distribution metrics for MVP ops? | Curator | Recommendation: emit `quality_distribution` as a log event field on `memory.job.completed`, not as Prometheus metric labels (avoids high-cardinality label explosion). Query via log aggregation. |
| OQ-5 | Provider/runtime selection | Architect | Provider-agnostic interface defined; concrete selection is Phase 2 implementation detail, out of scope for BRD-03. |

OQ-3 (source_location format) is **resolved**: character offsets committed — `{type: "char_offset", start: integer, end: integer}` with rune-index validation.

---

## Acceptance Criteria Checklist

Before marking this BRD as completed:

- [x] All functional requirements (FR-1 through FR-21) defined with acceptance criteria
- [x] All 8 user stories have Given/When/Then ACs
- [x] All NFRs separated into type/target/measurement format
- [x] Non-goals explicitly listed and distinguished from deferred items
- [x] Feature flag assigned: `FF_ENABLE_MEETING_MEMORY_PROCESSING` (server) and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` (browser)
- [x] Phase assigned: Phase 2
- [x] All open questions documented with owner and resolution status
- [x] Security considerations and threat model complete
- [x] Dependencies mapped; BRD-02 hard dependency confirmed; BRD-04 blocking relationship explicit
- [x] Observability section complete with metric names, log event names, redaction rules
- [x] All 24 acceptance criteria (AC-1 through AC-24) defined with independently evaluable methods
- [x] Feature flag registered in `specs/feature-flags.md`
- [ ] API contracts defined in `contracts/openapi.yaml` (7 endpoints, 10 schemas) — out of scope for curation; implementation task
- [x] ADRs 0005-0008 all Accepted and incorporated into BRD prose

---

## Acceptance Criteria Detail

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | When disabled (`FF_ENABLE_MEETING_MEMORY_PROCESSING=false`), successful manual import does not queue memory processing | Feature flag integration eval |
| AC-2 | When enabled, successful manual import queues processing without blocking redirect or meeting detail access | E2E eval: import redirect completes before processing done |
| AC-3 | Completed processing result includes all 7 categories with evidence or insufficient_evidence status | Integration eval with sparse-fixture input |
| AC-4 | Categories without evidence show `Insufficient evidence` not omission or guess | Fixture eval with sparse notes |
| AC-5 | Every supported item includes evidence snippet, source type, stable source location (char_offset format) | Schema/integration eval |
| AC-6 | Unsupported or tempting claims are omitted or marked insufficient | Strict hallucination eval with tempting-unsupported fixtures — at least 10 fixture pairs labeled as: (a) evidence fully supports item, (b) evidence partially supports item, (c) evidence does not support item. All fixtures must be labeled (a) or (c) with zero (b) for this criterion to pass |
| AC-7 | Evidence snippets actually support the attached item | Evidence-grounding eval: at least 10 fixture pairs (item text + evidence snippet) manually or LLM-assisted reviewed and labeled as fully supporting, partially supporting, or not supporting. Zero partially-supporting labels required to pass |
| AC-8 | Action items preserve description, explicitly stated owner/due date; unstated fields remain unspecified | Fixture eval with varied owner/due date presence |
| AC-9 | Decisions are marked standalone/confirms/changes/reverses only with supporting evidence and eligible prior memory | Integration eval with two meetings: (a) no prior memory produces standalone decisions, (b) semantically identical prior produces confirms status. Second scenario with contradictory decisions verifies changes/reverses + Conflicting evidence trigger |
| AC-10 | Stakeholder notes exclude sensitive personal profiling or unsupported attributes | Safety eval with explicit negative test cases for personality, psychological inference, unsupported influence |
| AC-11 | Each processing attempt creates a new version; latest successful becomes active | Integration eval |
| AC-12 | Failed/retrying/retry-exhausted runs do not replace previous active memory version | Integration eval |
| AC-13 | Source meeting edits mark active memory stale and queue reprocessing | Integration or E2E eval |
| AC-14 | User-visible lifecycle states include all 10 states: not processed, queued, processing, completed, completed with insufficient evidence, failed, stale, reprocessing, retrying, retry exhausted | State-machine eval |
| AC-15 | Prior memories used are visible without exposing content in logs | E2E or integration eval; log inspection |
| AC-16 | User can exclude incorrect prior memory, include missing one, and reprocess to create new version | E2E eval |
| AC-17 | Quality statuses use only `strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence` at item and category level | Schema/unit eval |
| AC-18 | Conflicting evidence appears in review queue, excluded from normal categories and briefing until resolved | E2E/integration eval |
| AC-19 | User can resolve conflict with evidence-backed resolution note, preserving original sources for audit | E2E eval |
| AC-20 | Briefing-readiness ready only when core categories have strong/weak evidence and no unresolved conflict | Readiness eval |
| AC-21 | Transient failures retry up to 3 times with exponential backoff (initial delay 1s, multiplier 2x, max delay 30s, jitter ±10%) before retry-exhausted | Job lifecycle integration eval |
| AC-22 | Worker/provider unavailability does not break manual import or existing active memory access | Degradation eval |
| AC-23 | Logs and metrics emit without sensitive content (no transcript, notes, evidence snippets, PII, memory content, resolution text) | Observability eval: grep log output and metric label names for prohibited fields |
| AC-24 | Provider/model selection is not hard-coded into product acceptance behavior | Architecture review |

---

## Relations

- **Parent feature:** BRD-02 Manual Meeting Import
- **Blocked by:** BRD-02, OpenAPI contract update (completed), feature flag registration
- **Blocks:** BRD-04 Pre-Call Briefing, BRD-06 Privacy & Retention Controls, BRD-07 Manual Memory Correction
- **Related specs:** `specs/domain/brd-02-manual-meeting-import.md`, `specs/curated/brd-02-manual-meeting-import.md`, `specs/feature-flags.md`, `docs/adr/0005-meeting-memory-processing-queue-architecture.md`, `docs/adr/0006-meeting-memory-data-model.md`, `docs/adr/0007-prior-memory-matching-conflict-detection.md`, `docs/adr/0008-observability-privacy-architecture.md`

---

## Pre-Implementation Approval Gate

**Implementation tasks derived from this BRD are blocked pending written approval from Shakil.**

The following must be confirmed before any implementation work begins:

1. Processing latency SLO: 30-second P95 target for Phase 2 (provider-agnostic); provider-specific calibration deferred to Phase 2 implementation after provider selection
2. Feature flag scope: single dual-namespace flag `FF_ENABLE_MEETING_MEMORY_PROCESSING` / `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` for both server and browser
3. OQ-1 resolved: single flag confirmed (gates both server processing and browser UI; `true` server + `false` browser is valid for server-side testing)
4. Source location format: character offsets (char_offsets) committed per ADR-0006

---

*Last curated: 2026-05-21. ADRs 0005-0008 all Accepted. Refiner HIGH issues resolved. Architect handoff incorporated.*