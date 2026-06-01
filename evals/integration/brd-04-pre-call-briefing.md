# Integration Eval: brd-04-pre-call-briefing

> 🔴 Failing — implementation pending

## Scope

Integration tests for Pre-Call Briefing (BRD-04): briefing API contract (6 endpoints), data model integrity, async job queue integration, BRD-02/BRD-03 data model dependencies, and version/stale lifecycle across the stack.

---

## Briefing API Contract

### GET /upcoming/{meetingId}/briefing

> Active/latest briefing version for an upcoming meeting.

| Scenario | Request | Expected |
|----------|---------|----------|
| Authenticated, briefing ready | Valid session, `ff_enable_pre_call_briefing=true`, briefing exists and is completed | 200, `preparation_status` in `["ready", "ready_with_caveats", "no_prior_memory"]`, `content` with `concise_summary` and `sources` |
| Authenticated, generating in progress | Valid session, job still running | 200, `preparation_status = "generating"` or `"regenerating"`, `content` may be null or partial |
| Authenticated, failed generation | Valid session, prior job failed | 200, `preparation_status = "failed"`, latest completed briefing still returned if one exists |
| Authenticated, stale briefing | Source memory changed after briefing created | 200, `preparation_status = "stale"` with visible stale indicator |
| No briefing yet, not generated | Valid session, no briefing job has run | 404 Not Found, or 200 with `preparation_status` indicating not-generated depending on whether a job was ever triggered |
| Flag disabled server-side | `FF_ENABLE_PRE_CALL_BRIEFING=false`, authenticated | 403 Forbidden |
| Unauthenticated | No session cookie | 401 Unauthorized |
| Meeting does not exist | Valid UUID not in DB | 404 Not Found |
| User not authorized for meeting | Meeting ACL denies access | 403 Forbidden |
| No prior memory shell | No qualifying source meetings found | 200, `preparation_status = "no_prior_memory"`, `content.no_prior_memory_shell.generated = true` |
| Content not in logs/metrics | GET /briefing, inspect observability output | No briefing text, source snippets, or participant PII in metrics/labels |

---

### GET /upcoming/{meetingId}/briefing/versions

> Version history list.

| Scenario | Expected |
|----------|----------|
| Lists all versions | Array of version objects with `version_number`, `created_at`, `trigger_type`, `status` |
| Active version marked | One entry with `is_active: true` |
| Ordered by version number | Versions listed ascending |
| Unauthenticated | 401 Unauthorized |
| Flag disabled | 403 Forbidden |
| No briefing versions exist | 200, `[]` |
| Meeting does not exist | 404 Not Found |

---

### GET /upcoming/{meetingId}/briefing/versions/{versionNumber}

> Specific briefing version.

| Scenario | Expected |
|----------|----------|
| Specific version accessible | 200, full `content` for that version including `concise_summary` and `detailed_sections` |
| Active and superseded both accessible | Both return 200 with full content |
| Version not found | 404 Not Found |
| Invalid version number (non-integer) | 400 Bad Request |
| Meeting does not exist | 404 Not Found |
| Unauthenticated | 401 Unauthorized |

---

### POST /upcoming/{meetingId}/briefing/regenerate

> Manual regeneration request.

| Scenario | Request | Expected |
|----------|---------|----------|
| Triggers async job, returns immediately | Valid session, `ff_enable_pre_call_briefing=true` | 202 Accepted, `job_id` or equivalent, does not block on generation completion |
| Latest completed briefing still viewable during regeneration | After POST regenerate, before job completes | GET `/upcoming/{meetingId}/briefing` returns latest completed briefing with `preparation_status = "regenerating"` |
| New version created after regeneration | Job completes successfully | New version in GET `/briefing/versions`, new version becomes active |
| Regeneration with source exclusions applied | One or more sources excluded via POST `/sources/{id}/exclude` before regenerate | New briefing version generated without excluded sources |
| Flag disabled server-side | `FF_ENABLE_PRE_CALL_BRIEFING=false` | 403 Forbidden |
| Unauthenticated | No session | 401 Unauthorized |
| Meeting does not exist | Valid UUID not in DB | 404 Not Found |
| User not authorized for meeting | Meeting ACL denies access | 403 Forbidden |
| Regeneration while job already running | Second POST regenerate while first job still running | 409 Conflict or 422 Unprocessable Entity, latest briefing still accessible |
| Regeneration after exclusion undone | Source restored then regenerate requested | Restored source included in new briefing version |

---

### POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude

> Exclude a source meeting from future briefings for this upcoming meeting.

| Scenario | Request | Expected |
|----------|---------|----------|
| Exclusion persisted per upcoming meeting | Valid session, `ff_enable_pre_call_briefing=true` | 200 or 201, exclusion saved to `briefing_source_exclusions` table |
| Exclusion does not affect other upcoming meetings | Source excluded for Meeting A | Source still available for Meeting B's briefings |
| Subsequent regeneration excludes the source | POST exclude → POST regenerate | New briefing version generated without the excluded source |
| Excluding already-excluded source | Idempotent | 200, no duplicate row in `briefing_source_exclusions` |
| Flag disabled | `FF_ENABLE_PRE_CALL_BRIEFING=false` | 403 Forbidden |
| Unauthenticated | No session | 401 Unauthorized |
| Source meeting does not exist | Valid UUID not in DB | 404 Not Found |
| Source not in current qualifying set | Source ID not among selected sources | 404 or 422, exclusion only applies to qualifying sources |
| User not authorized for upcoming meeting | Meeting ACL denies access | 403 Forbidden |
| User not authorized for source meeting | Source meeting ACL denies access | 403 Forbidden |

---

### POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore

> Restore a previously excluded source meeting.

| Scenario | Request | Expected |
|----------|---------|----------|
| Restoration removes exclusion | Valid session, source previously excluded | 200, exclusion removed from `briefing_source_exclusions` |
| Restored source included in next regeneration | POST restore → POST regenerate | New briefing version includes the restored source |
| Restore non-excluded source | Source not currently excluded | 200, no-op, no change to exclusions |
| Flag disabled | `FF_ENABLE_PRE_CALL_BRIEFING=false` | 403 Forbidden |
| Unauthenticated | No session | 401 Unauthorized |
| User not authorized for upcoming meeting | Meeting ACL denies access | 403 Forbidden |

---

## Async Job Queue Integration

### Job Trigger on Upcoming Meeting Creation

| Scenario | Trigger | Expected |
|----------|---------|----------|
| Upcoming meeting creation queues async job | POST `/upcoming` with `ff_enable_pre_call_briefing=true` | After redirect to detail page, GET `/upcoming/{id}/briefing/status` returns `generating`; job queued, not blocking |
| Upcoming meeting creation does not queue when flag disabled | POST `/upcoming` with `ff_enable_pre_call_briefing=false` | No briefing job queued; GET `/upcoming/{id}/briefing` returns 404 or not-yet-generated state |
| Job insert after transaction commit | Create upcoming meeting → redirect → poll state | Briefing state is `queued` or `generating` only after meeting row is committed |
| Redirect does not wait for processing | POST `/upcoming` → redirect to detail | Redirect completes while generation still `queued` or `generating` |

### GET /upcoming/{meetingId}/briefing/status (preparation_status polling)

| Scenario | Expected |
|----------|----------|
| Returns preparation status | One of: `generating`, `ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, `regenerating` |
| Generating state transitions to terminal | Poll after job completes | Status moves to `ready`, `ready_with_caveats`, `no_prior_memory`, or `failed` |
| Retry_exhausted state surface | Job exhausted retries | `preparation_status = "failed"` |
| Stale state after source memory change | After BRD-03 reprocessing | `preparation_status = "stale"` |
| Flag disabled returns 403 | `FF_ENABLE_PRE_CALL_BRIEFING=false` | 403 Forbidden |
| Unauthenticated returns 401 | No session | 401 Unauthorized |

---

## Data Model Integrity

### briefing_versions

| Scenario | Expected |
|----------|----------|
| UNIQUE(upcoming_meeting_id, version_number) enforced | Two inserts with same upcoming meeting + version number: second fails or rejected |
| Exactly one is_active per upcoming meeting | After multiple versions: `SELECT is_active FROM briefing_versions WHERE upcoming_meeting_id=?` returns exactly 1 true |
| Index on (upcoming_meeting_id, is_active) used | Query by upcoming meeting + is_active performs index scan, not full table scan |
| ON DELETE CASCADE | Deleting upcoming meeting removes all its briefing_versions rows |
| JSONB content valid | All stored `content` values parse as valid Briefing JSON per BRD-04 schema |
| status values restricted | Insert with `status='pending'`: fails; valid values are `active`, `superseded` |
| preparation_status values restricted | Insert with `preparation_status='unknown'`: fails; valid values per BRD-04 FR-20 |
| trigger_type values restricted | Insert with `trigger_type='scheduled'`: fails; valid values are `auto`, `manual_regenerate` |

### briefing_source_exclusions

| Scenario | Expected |
|----------|----------|
| UNIQUE(upcoming_meeting_id, excluded_source_meeting_id) enforced | Same pair twice: second fails or rejected |
| ON DELETE CASCADE | Deleting upcoming meeting removes its exclusions |
| Exclusions scoped per upcoming meeting | Same source meeting excluded for Meeting A and Meeting B | Meeting A exclusion does not affect Meeting B |
| excluded_source_meeting_id references meetings(id) | FK constraint enforced, invalid UUID rejected |

---

## BRD-02 / BRD-03 Data Model Dependencies

### BRD-02 Import Dependency

| Scenario | Expected |
|----------|----------|
| Briefing requires source meetings to exist | Source meetings imported via BRD-02 POST `/meetings` | Meetings accessible and meet BRD-02 import requirements |
| Source meeting authorization enforced | User cannot access source meeting | Source not selected for briefing; authorization error not exposed to user |
| Source meeting deleted after briefing generated | Source meeting removed | Briefing source list remains stable with version; stale detection may trigger |
| Source meeting memory not yet processed | Source meeting imported but BRD-03 processing incomplete | Source not selected OR selected with `insufficient_evidence`; briefing may generate with `no_prior_memory` shell if no qualifying sources |

### BRD-03 Memory Processing Dependency

| Scenario | Expected |
|----------|----------|
| Briefing uses BRD-03 evidence status | Source memory has `quality_status` values | `strong_evidence` items appear normally; `weak_evidence` items appear with caveats; `insufficient_evidence` items excluded from advice; `conflicting_evidence` items excluded from briefing advice and stakeholder notes |
| Briefing uses BRD-03 conflict status | Source has unresolved conflicts | Conflicting memory items excluded from briefing advice |
| Briefing uses briefing-readiness signal | GET `/meetings/{id}/memory` shows readiness | `ready_with_weak` signals inform briefing `preparation_status = "ready_with_caveats"` |
| Source memory version included in briefing | Source meeting has multiple memory versions | Latest completed version used for briefing; version number traceable |
| Source memory marked stale after briefing | BRD-03 reprocessing triggers staleness | Briefing version marked `stale`; regeneration offered |

---

## Version Lifecycle

| Scenario | Trigger | Expected |
|----------|---------|----------|
| New version created on regenerate | POST `/briefing/regenerate` | New row in `briefing_versions`, incremented `version_number`, `is_active=true`, prior version `is_active=false` and `status='superseded'` |
| Active/latest completed version shown by default | GET `/briefing` | Returns version with `is_active=true` and highest `version_number` |
| Prior versions accessible | GET `/briefing/versions/{n}` | Returns superseded version content |
| Version history ordered ascending | GET `/briefing/versions` | First entry is version 1, ascending |
| Stale version still accessible | Source memory changed | Stale version still queryable at its version number; `preparation_status='stale'` on active version |
| Regeneration preserves failed state | Job fails, no new version succeeds | Active version remains previous successful version; `preparation_status='failed'` |

---

## Source Selection Integration

| Scenario | Trigger | Expected |
|----------|---------|----------|
| Related-meeting matching uses at least 2 signals | Source meeting + upcoming meeting share signals | Participant email match, title similarity, org match, or close chronology — minimum 2 required |
| Top 3 qualifying sources selected | More than 3 meetings qualify | Only top 3 by signal count then recency selected |
| Fewer than 3 qualifies uses available | Only 1 or 2 meetings qualify | Briefing uses available qualifying meetings |
| No qualifying meetings generates shell | No source meetings meet 2-signal threshold | `preparation_status='no_prior_memory'`, no-prior-memory shell generated |
| Close chronology: future-dated source excluded | Source meeting scheduled after upcoming meeting | Source not selected even if other signals match |
| Close chronology: 90-day window enforced | Source meeting completed > 90 days before upcoming start | Source not selected even if other signals match |

---

## Authorization Integration

| Scenario | Expected |
|----------|----------|
| User cannot access briefing for unauthorized upcoming meeting | Meeting ACL denies access | 403 Forbidden on GET `/briefing`, `/briefing/versions`, POST regenerate |
| User cannot access source meeting | Source meeting ACL denies access | Source not selected, not logged, not used in generation |
| Unauthorized source meetings excluded from source list | Source meeting not accessible to user | Excluded from `content.sources` array; count reflects accessible sources only |
| Exclude/restore requires upcoming meeting authorization | User can view but not modify | 403 Forbidden on POST exclude/restore |

---

## Observability Integration

| Scenario | Expected |
|----------|----------|
| Preparation status enum in metrics | GET /briefing/status or job state transitions | Metrics use controlled enum labels: `generating`, `ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, `regenerating` |
| Source count bucket in metrics | Briefing generated with N sources | Metrics labels use bucket: `0`, `1`, `2`, `3` |
| Generation trigger in metrics | Job created | Metrics label: `auto` or `manual_regenerate` |
| No raw content in metric labels | Inspect all labels | No meeting titles, participant names, briefing text, or source snippets in labels |
| No PII in logs | GET /briefing, POST regenerate, poll status | Logs contain only hashed/opaque identifiers |

---

## Running

```bash
# Requires backend and database running
make eval-integration
# or
go test ./tests/integration/... -v
```

---

## Acceptance Criteria Coverage

| ID | Criterion | Integration Eval Coverage |
|----|-----------|---------------------------|
| AC-1 | Flag disabled hides UI + rejects backend | Tested via 403 on all endpoints with flag disabled |
| AC-2 | Upcoming meeting creation queues async job non-blocking | Tested in Async Job Queue Integration section |
| AC-3 | Manual regenerate queues new job, latest briefing remains | Tested in POST /regenerate section |
| AC-4 | Related-meeting matching requires ≥2 signals | Tested in Source Selection Integration section |
| AC-5 | Top 3 qualifying sources used | Tested in Source Selection Integration section |
| AC-6 | No-prior-memory shell when no qualifying sources | Tested in GET /briefing section + Source Selection Integration |
| AC-9 | Every briefing item traceable to source | Tested via version history + source list in content |
| AC-12 | Unresolved conflicts excluded from briefing advice | Tested in BRD-03 Memory Processing Dependency section |
| AC-14 | Source exclusion scoped per upcoming meeting | Tested in Data Model Integrity section |
| AC-16 | Each generation creates distinct version | Tested in Version Lifecycle section |
| AC-17 | Source memory change marks stale | Tested in Version Lifecycle section |
| AC-18 | Failed generation keeps latest briefing available | Tested in GET /briefing section + POST /regenerate section |