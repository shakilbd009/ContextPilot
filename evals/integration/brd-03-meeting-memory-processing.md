# Integration Eval: brd-03-meeting-memory-processing

> 🔴 Failing — implementation pending

## Scope

Integration tests for Meeting Memory Processing (BRD-03): BRD-02 import integration, full memory API contract (7 endpoints), data model integrity, and version lifecycle across the stack.

---

## BRD-02 Import Integration

### POST /meetings Triggers Memory Processing

| Scenario | Trigger | Expected |
|----------|---------|----------|
| Import queues processing when enabled | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`, POST `/meetings` succeeds | After redirect to `/meetings/{id}`, GET `/meetings/{id}/memory/state` returns `state='queued'` |
| Import does not queue when disabled | `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`, meeting saved | GET `/meetings/{id}/memory/state` returns 403 or job not queued |
| Queue insert after transaction commit | Import POST → 201 → GET `/meetings/{id}/memory/state` | State is `queued` only after meeting row is committed |
| Import redirect does not wait for processing | POST `/meetings` → redirect to detail | Redirect completes while processing still `queued` or `processing` |
| Import with minimal content queues | Transcript only, meeting saved | Processing queued; later processing handles sparse content |

---

## Memory API Contract

### GET /meetings/{id}/memory

| Scenario | Request | Expected |
|----------|---------|----------|
| Authenticated, completed memory | Valid session, meeting has completed processing | 200, `content` with all 7 categories, `briefing_readiness` signal |
| Unauthenticated | No session cookie | 401 Unauthorized |
| Flag disabled server-side | `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`, authenticated | 403 Forbidden |
| Meeting does not exist | Valid UUID not in DB | 404 Not Found |
| Memory not yet processed | Meeting imported, processing not started | 200, `state: 'queued'` or `state: 'not_processed'` |
| Memory in retry_exhausted | Job exhausted retries | 200, active memory still accessible, `state: 'retry_exhausted'` |
| Content not in logs/metrics | GET /memory, inspect observability output | No memory content, evidence snippets, or PII in metrics/labels |

### GET /meetings/{id}/memory/versions

| Scenario | Expected |
|----------|----------|
| Lists all versions | Array of version objects with `version_number`, `created_at`, `trigger_type`, `status` |
| Active version marked | One entry with `is_active: true` |
| Ordered by version number | Versions listed ascending |
| Unauthenticated | 401 Unauthorized |
| Flag disabled | 403 Forbidden |

### GET /meetings/{id}/memory/versions/{versionNumber}

| Scenario | Expected |
|----------|----------|
| Specific version accessible | 200, full `content` for that version |
| Active and superseded both accessible | Both return 200 with full content |
| Version not found | 404 Not Found |
| Invalid version number | 404 or 400 |

### POST /meetings/{id}/memory/reprocess

| Scenario | Request | Expected |
|----------|---------|----------|
| Reprocess on failed job | `trigger_type='manual_retry'` job queued | After completion, new version created |
| Reprocess excludes prior memory | Body: `{ "excluded_prior_memory_ids": ["<uuid>"] }` | New version processed without specified prior memory |
| Reprocess includes prior memory | Body: `{ "included_prior_memory_ids": ["<uuid>"] }` | New version includes specified prior memory |
| Reprocess from stale state | Memory marked `stale` | `trigger_type='stale_reprocess'` queued |
| Unauthenticated | No session | 401 Unauthorized |
| Flag disabled | `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` | 403 Forbidden |

### GET /meetings/{id}/memory/state

| Scenario | Expected |
|----------|----------|
| Returns current state | One of: `not_processed`, `queued`, `processing`, `completed`, `completed_with_insufficient_evidence`, `failed`, `stale`, `reprocessing`, `retrying`, `retry_exhausted` |
| State reflects retry progress | After transient failure: `retrying`; after max retries: `retry_exhausted` |
| Stale state after source edit | After meeting `updated_at` advances past active version `created_at`: `stale` |

### GET /meetings/{id}/memory/conflicts

| Scenario | Expected |
|----------|----------|
| Lists pending conflicts | Array of conflict objects with `conflicting_items`, `category`, `review_status` |
| Conflicting items from both sides | `conflicting_items.current` and `conflicting_items.prior` both present |
| Evidence snippets attributed | Each conflict shows source meeting ID for snippet attribution |
| Resolution note field present | `resolution_note` field present (may be null) |
| No conflicts returns empty array | 200, `[]` |
| Only pending listed | Resolved conflicts not returned |

### POST /meetings/{id}/memory/conflicts/{conflictId}/resolve

| Scenario | Request | Expected |
|----------|---------|----------|
| Resolve with note | `{ "resolution_note": "Chose current based on later timestamp" }` | 200, `review_status = 'reviewed'`, `resolved_at` set, `resolved_by` set |
| Resolve without note | `{}` | 200, `resolution_note` null |
| New version created | After resolve | New version with `trigger_type = 'conflict_resolution'` |
| Conflict resolved removed from queue | After resolve, GET conflicts | Conflict no longer in pending list |
| Event emitted | After resolve | `meeting_memory.conflict.reviewed` event emitted |
| Original sources preserved | After resolve, inspect new version | Original conflicting sources still accessible in version history |
| Unauthenticated | No session | 401 Unauthorized |
| Conflict already resolved | POST on `review_status='reviewed'` | 400 or 409 |

---

## Data Model Integrity

### memory_versions

| Scenario | Expected |
|----------|----------|
| UNIQUE(meeting_id, version_number) enforced | Two inserts with same meeting_id + version_number: second fails or rejected |
| Exactly one is_active per meeting | After multiple versions: `SELECT is_active FROM memory_versions WHERE meeting_id=?` returns exactly 1 true |
| Index on (meeting_id, is_active) used | Query by meeting + is_active performs index scan, not full table scan |
| ON DELETE CASCADE | Deleting meeting removes all its memory_versions rows |
| JSONB content valid | All stored `content` values parse as valid Memory JSON (schema per BRD-03) |

### memory_evidence

| Scenario | Expected |
|----------|----------|
| ON DELETE CASCADE | Deleting memory_version removes its evidence rows |
| Index on memory_version_id | Lookup by version uses index |
| Required fields enforced | Insert without `evidence_snippet`: fails |
| Quality status valid | Insert with `quality_status='unknown'`: fails |

### memory_conflicts

| Scenario | Expected |
|----------|----------|
| Pending conflicts index | Query `WHERE review_status='pending'` uses index |
| ON DELETE CASCADE | Deleting meeting removes its conflicts |
| Resolution fields set on resolve | After resolve: `resolved_at`, `resolved_by`, `review_status='reviewed'` all set |
| Atomic conflict resolution | Resolving one conflict does not affect others for same meeting |

### memory_prior_memory_inputs

| Scenario | Expected |
|----------|----------|
| UNIQUE(memory_version_id, prior_memory_version_id) enforced | Same pair twice: second fails |
| RESTRICT on prior_memory_version_id | Cannot delete a prior memory version while referenced |
| included_by_user defaults false | New row: `included_by_user = false` |
| excluded_by_user defaults false | New row: `excluded_by_user = false` |

---

## Stale Detection Integration

| Scenario | Trigger | Expected |
|----------|---------|----------|
| Transcript edit marks stale | Update meeting transcript after processing | Active memory version: `status='stale'`; new job queued with `trigger_type='stale_reprocess'` |
| Notes edit marks stale | Update meeting notes | Same as above |
| Participant change marks stale | Add/remove meeting participant | Same as above |
| Title edit marks stale | Update meeting title | Same as above |
| Prior version still accessible | After stale detected | Prior active version still queryable at its version number |
| Reprocess succeeds | Source edit + new processing | New version becomes active, `stale` state cleared |

---

## Authorization Integration

| Scenario | Expected |
|----------|----------|
| User cannot access other user's meeting memory | Meeting owned by User A; User B authenticated | 403 Forbidden on GET `/meetings/{id}/memory` |
| User cannot resolve conflict on other user's meeting | Conflict belongs to User A's meeting | 403 Forbidden on POST resolve |
| ACL from BRD-02 applies | Meeting visible via standard ACL | Memory endpoints respect same authorization as meeting endpoints |

---

## Running

```bash
# Requires backend and database running
make eval-integration
# or
go test ./tests/integration/... -v
```
