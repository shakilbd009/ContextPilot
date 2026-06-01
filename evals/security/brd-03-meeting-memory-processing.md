# Security Eval: brd-03-meeting-memory-processing

> 🔴 Failing — implementation pending

## Scope

Security checks for Meeting Memory Processing (BRD-03): input sanitization, authentication and authorization on all 7 memory endpoints, flag-gate enforcement, sensitive data exclusion from logs/metrics, and PII handling in evidence.

---

## Authentication & Authorization

### All Memory Endpoints Require Auth

| Endpoint | Method | Expected Without Session |
|----------|--------|--------------------------|
| GET /meetings/{id}/memory | No cookie | 401 Unauthorized |
| GET /meetings/{id}/memory/versions | No cookie | 401 Unauthorized |
| GET /meetings/{id}/memory/versions/{n} | No cookie | 401 Unauthorized |
| POST /meetings/{id}/memory/reprocess | No cookie | 401 Unauthorized |
| GET /meetings/{id}/memory/state | No cookie | 401 Unauthorized |
| GET /meetings/{id}/memory/conflicts | No cookie | 401 Unauthorized |
| POST /meetings/{id}/memory/conflicts/{id}/resolve | No cookie | 401 Unauthorized |

### Authorization: Owner or ACL

| Scenario | Expected |
|----------|----------|
| Authenticated, not owner, not in ACL | 403 Forbidden on all memory endpoints |
| Authenticated, owner of meeting | 200 on GET memory |
| Authenticated, meeting visible via ACL | 200 on GET memory |
| Reprocess by non-owner | 403 Forbidden |
| Resolve conflict by non-owner | 403 Forbidden |

### Feature Flag Gate

| Scenario | Expected |
|----------|----------|
| Server flag `false`, authenticated | 403 on all memory endpoints |
| Server flag `true`, browser flag `false` | Server accepts memory operations; browser UI hidden |
| Flag misconfiguration (server=false, browser=true) | Server returns 403; `cp_meeting_memory_flag_misconfiguration_total` incremented; `memory.flag_misconfiguration` log event emitted |

---

## Input Sanitization

### Evidence Snippet Injection

| Scenario | Input | Expected |
|----------|-------|----------|
| XSS in evidence snippet | Memory item with `snippet: "<script>alert(1)</script>"` | Stored as-is or sanitized; GET response does not execute script |
| HTML in evidence snippet | `snippet: "<h1>Big</h1>"` | Stored as-is; GET returns raw text, not rendered HTML |
| Null bytes in snippet | `snippet: "Test\x00Meeting"` | Null byte stripped or request rejected |
| Very long evidence snippet | `snippet` exceeds reasonable length | Rejected or truncated with validation error |

### Stakeholder Notes Input

| Scenario | Input | Expected |
|----------|-------|----------|
| XSS in stakeholder note field | `preferences: "<script>evil()</script>"` | Stored sanitized; GET does not execute |
| SQL injection in stakeholder note | `concerns: "'; DROP TABLE memory_conflicts; --"` | Stored as literal string; no table deletion |
| HTML tags in stakeholder note | `commitments: "<b>bold</b>"` | Stored as-is or sanitized; GET returns raw text |
| Sensitive personal profiling language | Unsupported psychological inference | Rejected or excluded; BRD-03 FR-7 constrains stakeholder notes to meeting-relevant business context |

### Resolution Note Input

| Scenario | Input | Expected |
|----------|-------|----------|
| XSS in resolution note | `resolution_note: "<img src=x onerror=alert(1)>"` | Stored sanitized; no script execution |
| Resolution note max length | `resolution_note` exceeds 10,000 chars | Rejected or truncated |
| HTML in resolution note | `resolution_note: "<script>evil()</script>"` | Stored as sanitized text, not executed |

### Prior Memory Input

| Scenario | Input | Expected |
|----------|-------|----------|
| Invalid UUID for excluded_prior_memory_ids | `excluded_prior_memory_ids: ["not-a-uuid"]` | 400 Bad Request |
| UUID for non-existent prior memory | Valid UUID but not found | 400 or processed without effect (graceful handling) |
| Attempt to exclude another user's memory | UUID belonging to different user's meeting | 403 Forbidden |

---

## Sensitive Data Exclusion from Observability

### Logs Must Not Contain

| Forbidden Field | Must Not Appear In |
|-----------------|---------------------|
| Raw transcript text | `memory.job.completed`, `memory.job.failed`, any log event |
| Raw notes text | Any log event |
| Evidence snippets | Any log event |
| Participant names | Any log event |
| Participant emails | Any log event |
| Memory content (generated) | Any log event |
| Prior memory content | Any log event |
| Resolution note text | Any log event |
| Source locations (char offsets) | Any log event |
| `memory_evidence.id` | Any log event (correlation risk) |

### Metrics Labels Must Not Contain

| Forbidden Label Value | Metric |
|----------------------|--------|
| Meeting title | Any `cp_meeting_memory_*` counter/histogram |
| Transcript text | Any histogram bucket or label |
| Evidence snippet | Any label |
| Participant name | Any label |
| Memory content | Any label |
| Resolution note | Any label |

### Verification Method

| Check | Method |
|-------|--------|
| Logs excluded | During E2E or integration eval: grep log output for 10+ forbidden strings; none should match |
| Metrics excluded | During eval: inspect Prometheus labels for `cp_meeting_memory_*` metrics; no content labels present |
| Source locations excluded | During eval: search all log lines for `char_offset` patterns; none should appear |

### Allowed Observability Fields

| Field | Where Allowed |
|-------|--------------|
| `correlation_id` | All log events |
| `meeting_id` (UUID) | All log events, metric labels |
| `trigger_type` | All job log events, `cp_meeting_memory_processing_queued_total` label |
| `failure_class` | `memory.job.failed`, `cp_meeting_memory_processing_failed_total` |
| `retry_count` | `memory.job.retrying`, `memory.job.retry_exhausted` |
| `duration_ms` | `memory.job.completed`, `memory.job.failed` |
| `processing_version_number` | `memory.version.activated` |
| `category` | `memory.conflict.detected` |
| `conflict_id` | `memory.conflict.detected`, `memory.conflict.resolved` |
| `reason` | Various events (reprocessing reason, conflict resolution reason) |
| `quality_distribution` | `memory.job.completed` (log only, not Prometheus label) |

---

## Privacy: Prior Memory Matching Constraints

| Scenario | Expected |
|----------|----------|
| Matching constrained to same email | Cross-meeting match without shared email not attempted |
| Matching constrained to similar title | Levenshtein ≤ 3 or 4+ token match required |
| Matching constrained to 90-day window | Prior meeting outside ±90 day window not matched |
| Uncertain match requires confirmation | `match_confidence='uncertain'` displayed to user before use |
| Broad opaque matching excluded | No embedding similarity, no cross-system relationship inference |
| Prior memory content not exposed | Prior memory visible as title/date/match_confidence only |

---

## Secure Defaults

| Check | Method | Expected |
|-------|--------|----------|
| Error responses safe | Memory endpoint with bad input | 4xx body contains no raw input reflected, no stack traces |
| Session cookie is httpOnly | Inspect Set-Cookie on auth | `HttpOnly` flag present |
| Session cookie SameSite | Inspect Set-Cookie | `SameSite=Lax` or `SameSite=Strict` |
| Conflict resolution notes private | User A resolves conflict | User B cannot see User A's resolution note |
| Version history not exposed to unauthorized | Non-owner GET versions | 403 Forbidden |
| Flag evaluation errors are safe | Misconfigured flag | No crash; safe error response; `flag_evaluation_errors_total` incremented |

---

## Threat Model: Hallucination Prevention

| Scenario | Expected |
|----------|----------|
| Insufficient evidence never filled with guess | Processing with empty transcript/notes: all categories `insufficient_evidence` |
| Tempting but unsupported claims rejected | Evals inject tempting-unsupported fixtures; system marks `insufficient_evidence` |
| Evidence-grounding verified | At least 10 fixture pairs verified: evidence snippet actually supports claimed item |
| Stakeholder notes stay in scope | Notes do not include personality profiling, psychological inference |

---

## Running

```bash
# Requires app running
make eval-security
```
