# Unit Eval: brd-03-meeting-memory-processing

> 🔴 Failing — implementation pending

## Scope

Unit tests for Meeting Memory Processing (BRD-03): async queue behavior, memory JSON schema, version history, retry backoff, conflict detection, quality status assignment, and evidence structure validation.

---

## Async Queue Tests

### Queue Insert Behavior

| Scenario | Input | Expected |
|----------|-------|----------|
| Flag disabled, import completes | `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`, meeting saved via BRD-02 | No `memory_processing_jobs` row inserted |
| Flag enabled, import completes | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`, meeting saved | `memory_processing_jobs` row inserted with `trigger_type='import'`, `state='queued'` |
| Queue row has required fields | After successful import | `job.meeting_id` set, `job.trigger_type='import'`, `job.state='queued'`, `job.created_at` set |
| Queue insert is transactionally after commit | Import transaction + queue insert | Queue row visible only after meeting transaction commits; rollback of import prevents queue row |
| Meeting detail accessible while queued | Meeting imported, job in `queued` | GET `/meetings/{id}/memory/state` returns `state='queued'`, no blocking |

### Retry Trigger Conditions

| Scenario | Input | Expected |
|----------|-------|----------|
| Transient failure queues retry | Provider returns 503 on first attempt | `retry_count=1`, `state='retrying'`, next retry scheduled |
| Max retries exhausts job | 3 consecutive transient failures | `state='retry_exhausted'`, no further automatic retry |
| Non-transient failure does not retry | Malformed transcript, missing required field | `state='failed'`, `retry_count=0`, no automatic retry |
| Manual retry from retry_exhausted | User calls POST `/meetings/{id}/memory/reprocess` on `retry_exhausted` job | New job row created with `trigger_type='manual_retry'`, `state='queued'` |
| Reprocess from stale state | Source edit marks memory `stale`, user reprocesses | New job with `trigger_type='manual_retry'` or `trigger_type='stale_reprocess'` |

---

## Memory Structure Tests

### JSON Schema Validation

| Scenario | Input | Expected |
|----------|-------|----------|
| Complete memory has all 7 categories | Valid processing output | `content` contains keys: `summary`, `decisions`, `action_items`, `risks_blockers`, `open_questions`, `stakeholder_notes`, `next_recommended_focus` |
| Summary structure valid | Any processing result | `summary.statement` is string, `summary.quality_status` in {strong_evidence, weak_evidence, insufficient_evidence}, `summary.items` is array |
| Decisions item structure | Any decision item | `items[].id`, `items[].decision_statement`, `items[].change_status` in {standalone, confirms_prior, changes_prior, reverses_prior}, `items[].quality_status`, `items[].evidence` array |
| Action item required fields | Any action item | `id`, `description`, `status` in {pending, in_progress, completed, deferred}, `quality_status`, `evidence` array |
| Action item optional fields | Owner not stated | `owner` is null, `owner_specified: false` |
| Action item owner specified | Owner "Alice" stated | `owner: "Alice"`, `owner_specified: true` |
| Action item due_date formats | ISO 8601 date string | `due_date: "2026-06-01"`, `due_date_specified: true` |
| Risks/blockers item structure | Any risk/blocker | `id`, `type` in {risk, blocker}, `description`, `quality_status`, `evidence` array |
| Open questions structure | Any question | `id`, `question`, `quality_status`, `evidence` array |
| Stakeholder notes structure | Any stakeholder note | `id`, `participant_id` (UUID), `quality_status`, `evidence` array; optional: `preferences`, `concerns`, `commitments`, `influence_stake` as string or null |
| Next recommended focus structure | Valid focus output | `statement`, `quality_status`, `supporting_item_ids` array, `evidence` array |
| Briefing readiness structure | Any completed processing | `ready` boolean, `core_categories_ready` boolean, `blocking_conflicts` array, `insufficient_categories` array |

### Insufficient Evidence Handling

| Scenario | Input | Expected |
|----------|-------|----------|
| Missing category gets insufficient_evidence | Transcript with no mention of risks | `risks_blockers.quality_status = 'insufficient_evidence'`, items array empty |
| Category marked insufficient not empty-object | No evidence for a category | Category present with `quality_status = 'insufficient_evidence'`, not omitted, not filled with guesses |
| Stakeholder notes missing from transcript | Transcript has no stakeholder-relevant content | `stakeholder_notes.quality_status = 'insufficient_evidence'` |
| Next recommended focus insufficient | Few categories have evidence | `next_recommended_focus.quality_status = 'insufficient_evidence'` |

### Evidence Record Validation

| Scenario | Evidence Record | Expected |
|----------|-----------------|----------|
| Evidence snippet non-empty | Any supported memory item | `evidence[].snippet` is non-empty string |
| Source type valid | Any evidence | `source_type` in {transcript, notes, prior_memory_reference} |
| Char offset structure | Transcript evidence | `source_location.type = "char_offset"`, `start` integer, `end` integer |
| Char offset valid range | Stored transcript "hello" (5 chars) | Valid: `{start: 0, end: 5}`; Invalid: `{start: -1, end: 5}` rejected; `{start: 0, end: 6}` rejected |
| Prior memory reference structure | Prior memory evidence | `source_meeting_id` present (UUID) |
| Quality status values | Any evidence | In {strong_evidence, weak_evidence, insufficient_evidence} |

---

## Version History Tests

| Scenario | Setup | Expected |
|----------|-------|----------|
| First processing creates version 1 | New meeting, processing completes | `memory_versions.version_number = 1`, `is_active = true`, `status = 'active'` |
| Second reprocess increments version | Reprocess same meeting | New row: `version_number = 2`, `is_active = true`; prior row: `is_active = false`, `status = 'superseded'` |
| Exactly one active per meeting | After 3 versions | Only one row with `is_active = true` |
| Version uniqueness constraint | Two rows with same `meeting_id` + `version_number` | Second insert fails or is rejected (database constraint) |
| Failed run does not replace active | Processing v2 fails | v1 remains `is_active = true`, no new active version created |
| Conflict review version excluded | Conflict detected during processing | New version has `status = 'conflict_review'`, `is_active = false` |
| Conflict resolved version becomes active | Resolution creates new version | New version `status = 'active'`, `is_active = true` |
| Version trigger types | Various reprocess scenarios | `trigger_type` in {import, reprocess, stale_reprocess, manual_retry, conflict_resolution} |
| Stale reprocess keeps prior active | Source edit detected, reprocessing | Prior version remains `is_active = true` until new version succeeds |

---

## Retry Logic Tests

### Backoff Calculation

| Scenario | Failure History | Expected Next Delay |
|----------|-----------------|---------------------|
| First retry delay | 0 prior retries | ~1s (±10%: 900ms–1100ms) |
| Second retry delay | 1 prior retry | ~2s (±10%: 1800ms–2200ms) |
| Third retry delay | 2 prior retries | ~4s (±10%: 3600ms–4400ms) |
| Delay does not exceed max | Any retry | Delay ≤ 30s (including jitter) |
| Jitter is applied | Multiple runs with same failure | Delay varies within ±10% band |
| Retry count tracked | After 1 retry | `retry_count = 1` on job row |
| Retry count at max | After 3 retries | `retry_count = 3`, job enters `retry_exhausted` |

### Transient vs Non-Transient Classification

| Scenario | Error Type | Expected Behavior |
|----------|-----------|-------------------|
| Provider 503 | Transient | Retry queued, retry_count incremented |
| Provider 500 | Transient | Retry queued (internal error) |
| Malformed input (missing transcript) | Non-transient | Immediate `failed`, retry_count = 0, no retry |
| Meeting not found | Non-transient | Immediate `failed`, no retry |
| Network timeout | Transient | Retry queued |
| Provider unavailable (connection refused) | Transient | Retry queued |

---

## Conflict Detection Tests

| Scenario | Input | Expected |
|----------|-------|----------|
| No conflict when no prior memory | Single meeting processing | No conflict record created |
| No conflict when prior is different category | Current: decision, prior: action item | No conflict |
| No conflict when evidence insufficient | Both items have `insufficient_evidence` | No conflict (AC-12) |
| Conflict when same category, contradictory, both strong | Decision A says "approve X", Decision B says "reject X" | Conflict created, `review_status = 'pending'` |
| Conflict when same category, both weak evidence | Both have `weak_evidence` but contradictory | Conflict created |
| Conflicting evidence excluded from categories | Meeting has pending conflict | Conflicting items NOT in `decisions`/`action_items`/`risks_blockers`/`summary` categories |
| Conflict resolution note stored | User resolves conflict | `resolution_note` field set, `resolved_at` set, `resolved_by` set |
| Single conflict resolution | Two conflicts exist, user resolves one | Second conflict remains `pending` |
| Conflict resolution creates new version | Resolve conflict | New version created with `trigger_type = 'conflict_resolution'` |

---

## Quality Status Tests

| Scenario | Evidence State | Expected |
|----------|---------------|----------|
| Item strong evidence | Source directly supports item | `quality_status = 'strong_evidence'` |
| Item weak evidence | Source partially supports item | `quality_status = 'weak_evidence'` |
| Item insufficient evidence | No source support | `quality_status = 'insufficient_evidence'` |
| Category worst-status rule | Category has 1 weak + 2 strong items | Category `quality_status = 'weak_evidence'` |
| Conflicting evidence status | Conflict detected | Item or category `quality_status = 'conflicting_evidence'` |
| Briefing readiness blocked by conflict | Conflict on `decisions`, all else strong | `core_categories_ready = false`, conflict in `blocking_conflicts` |
| Briefing readiness not blocked by non-core | `stakeholder_notes` insufficient only | `core_categories_ready = true` |

---

## Running

```bash
# From project root
go test ./internal/queue/... -v
go test ./internal/memory/... -v
go test ./internal/processor/... -v
```
