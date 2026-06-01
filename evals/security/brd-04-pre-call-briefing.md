# Security Eval: brd-04-pre-call-briefing

> 🔴 Failing — implementation pending

## Scope

Security checks for Pre-Call Briefing (BRD-04): RBAC on briefing access, input validation on all 6 endpoints, feature flag mismatch attack surface, PII exposure in briefing content, timing/side-channel risks in relatedness scoring, and preparation_status state machine authorization.

---

## Authentication & Authorization

### All Briefing Endpoints Require Auth

| Endpoint | Method | Expected Without Session |
|----------|--------|--------------------------|
| GET /upcoming/{meetingId}/briefing | No cookie | 401 Unauthorized |
| GET /upcoming/{meetingId}/briefing/versions | No cookie | 401 Unauthorized |
| GET /upcoming/{meetingId}/briefing/versions/{versionNumber} | No cookie | 401 Unauthorized |
| POST /upcoming/{meetingId}/briefing/regenerate | No cookie | 401 Unauthorized |
| POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude | No cookie | 401 Unauthorized |
| POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore | No cookie | 401 Unauthorized |

### Authorization: Owner or ACL on Upcoming Meeting

| Scenario | Expected |
|----------|----------|
| Authenticated, not owner, not in ACL | 403 Forbidden on all briefing endpoints |
| Authenticated, owner of upcoming meeting | 200 on GET briefing |
| Authenticated, upcoming meeting visible via ACL | 200 on GET briefing |
| Regenerate by non-owner | 403 Forbidden |
| Exclude source by non-owner | 403 Forbidden |
| Restore source by non-owner | 403 Forbidden |

### Authorization: Source Meeting Access Enforcement

| Scenario | Expected |
|----------|----------|
| User authorized for upcoming meeting but not for a source meeting | Source meeting must not be selected, displayed, logged, or used in briefing generation |
| GET briefing reveals source from unauthorized meeting | 403 or source omitted from response; no PII from unauthorized source in response |
| GET /briefing/versions/{n} with source from unauthorized meeting | 403 or unauthorized source content redacted |

### Authorization: Superseded Version Regenerate / Restore with Unauthorized Sources

| Scenario | Input | Expected |
|----------|-------|----------|
| Regenerate briefing using superseded v1 which includes an unauthorized source meeting | POST /briefing/regenerate while prior version(s) contain source from unauthorized meeting | Unauthorized source must not be selected, displayed, logged, restored, regenerated from, or used in new briefing generation |
| Regenerate using superseded version where unauthorized source was previously excluded | POST /briefing/regenerate referencing superseded version with unauthorized source in history | Unauthorized source must not appear in new briefing; regeneration succeeds with only authorized sources |
| Restore source on upcoming meeting where superseded versions reference unauthorized source meetings | POST /sources/{sourceId}/restore while superseded briefing versions include unauthorized meetings | Restoration completes safely; unauthorized source meetings remain excluded from briefing content and logs |
| Restore source that is itself unauthorized (user lacks ACL) | POST /sources/{sourceId}/restore where sourceId belongs to a meeting the user cannot access | 403 Forbidden; no mutation of source visibility or briefing content |
| GET /briefing/versions/{n} reveals unauthorized source was used in generation | GET /briefing/versions/1 | Unauthorized source content redacted or 403; no raw PII or content from unauthorized source in response |
| Log event emitted during regenerate with superseded version containing unauthorized sources | POST /briefing/regenerate with superseded version referencing unauthorized meeting | Log event contains no raw content, participant PII, or source meeting title from the unauthorized source |

### Feature Flag Gate

| Scenario | Expected |
|----------|----------|
| Server flag `false`, authenticated, POST /briefing/regenerate | 403 Forbidden with safe feature-disabled response |
| Server flag `false`, authenticated, GET /briefing | 403 Forbidden with safe feature-disabled response |
| Server flag `true`, browser flag `false` | Server accepts authorized operations; browser UI hidden |
| Flag misconfiguration (server=false, browser=true) | Server returns 403; `cp_pre_call_briefing_flag_misconfiguration_total` incremented; `briefing.flag_misconfiguration` log event emitted |
| Feature flag not registered | `check-feature-flags.sh` fails until flag is registered in `specs/feature-flags.md` |

---

## Input Validation / Injection Resistance

### GET /upcoming/{meetingId}/briefing — Path Parameter

| Scenario | Input | Expected |
|----------|-------|----------|
| Invalid UUID in meetingId | `meetingId: "not-a-uuid"` | 400 Bad Request |
| UUID of non-existent meeting | Valid UUID, meeting does not exist | 404 Not Found |
| UUID of another user's meeting | Valid UUID, user not authorized | 403 Forbidden |

### GET /upcoming/{meetingId}/briefing/versions — Path Parameter

| Scenario | Input | Expected |
|----------|-------|----------|
| Invalid UUID in meetingId | `meetingId: "////"` | 400 Bad Request |
| Valid UUID, no briefing versions exist | Well-formed UUID, 0 versions | 200 with empty versions array |

### GET /upcoming/{meetingId}/briefing/versions/{versionNumber} — Path Parameter

| Scenario | Input | Expected |
|----------|-------|----------|
| Non-integer versionNumber | `/versions/abc` | 400 Bad Request |
| Negative versionNumber | `/versions/-1` | 400 Bad Request |
| versionNumber exceeds latest version | `/versions/999999` | 404 Not Found |
| versionNumber is zero | `/versions/0` | 400 Bad Request |

### POST /upcoming/{meetingId}/briefing/regenerate — Request Body

| Scenario | Input | Expected |
|----------|-------|----------|
| Empty body (no content) | `{}` | 202 Accepted (async job queued) or safe validation |
| Malformed JSON | `{invalid}` | 400 Bad Request |
| Extra unknown fields in body | `{"extra_field": "value"}` | Safe ignore or 400; no crash |
| Null or missing meetingId in path | Path traversal attempt | 400 or 404; request rejected |

### POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude — Path Parameters

| Scenario | Input | Expected |
|----------|-------|----------|
| Invalid sourceId UUID | `sourceId: "not-a-uuid"` | 400 Bad Request |
| sourceId is a valid UUID but not a source meeting | Valid UUID, no associated source | 400 or 404 |
| sourceId belongs to a meeting not in the briefing's source list | Valid UUID, unauthorized source | 403 Forbidden |
| sourceId is the same as upcomingMeetingId | Self-referential UUID | 400 Bad Request |

### POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore — Path Parameters

| Scenario | Input | Expected |
|----------|-------|----------|
| Invalid sourceId UUID | `sourceId: "////"` | 400 Bad Request |
| sourceId not in exclusion list | Valid UUID, not previously excluded | 400 or safe no-op |
| Restore already-active source | Valid UUID, source already active | Safe idempotent response; 200 |

### XSS / Injection in Briefing Content Retrieval

| Scenario | Input | Expected |
|----------|-------|----------|
| Briefing content contains script tag | `<script>alert(1)</script>` in any content field | Content returned as raw text; no script execution in browser |
| Briefing content contains SVG payload | `<svg onload=alert(1)>` in stakeholder_notes | Content returned as raw text |
| Briefing content contains nested iframe | `<iframe src="evil">` in recommendations | Content returned as raw text |
| Very large briefing content field | Content exceeds reasonable size | 200 with full content; no truncation to bypass limits without warning |
| Null byte in briefing content retrieval | Content contains `\x00` | Null byte stripped or content returned safely |

---

## Feature Flag Mismatch Attack Surface

### Dual Namespace Flag Parity

| Scenario | Input | Expected |
|----------|-------|----------|
| Server `FF_ENABLE_PRE_CALL_BRIEFING=false`, client `VITE_FF_ENABLE_PRE_CALL_BRIEFING=true` | Misconfiguration state | Server returns 403 on all briefing mutations; metric incremented; misconfiguration logged; browser UI visible but non-functional |
| Server `FF_ENABLE_PRE_CALL_BRIEFING=true`, client `VITE_FF_ENABLE_PRE_CALL_BRIEFING=false` | Misconfiguration state | Server accepts operations; briefing entry points hidden in browser; server-side testing works |
| Both flags `false` | Normal disabled state | UI hidden; backend returns safe 403 response; no metric spam |

### Flag Evaluation Consistency

| Scenario | Expected |
|----------|----------|
| Flag evaluated at request time, not at session start | Changing flag mid-session does not cause inconsistent state |
| Flag value cached for async job | Async generation job reads flag value at queue time, not at execution time |
| Flag toggled from false to true while regeneration pending | Pending job completes using old flag state; new requests use new flag state |

---

## Data Exposure via Briefing Content (PII)

### Briefing Concise Summary

| Scenario | Expected |
|----------|----------|
| Participant email in `top_prior_context` | Email addresses redacted or omitted; `participant@example.com` not visible in summary |
| Participant name in `open_actions` | Names redacted if not necessary for context; initials or role used instead |
| PII in `risks_questions` | Raw PII redacted; no unredacted phone numbers, addresses, SSNs in briefing text |
| Stakeholder note contains personal detail | Sensitive personal attributes excluded per BRD-04 FR-13 |
| Conflicting evidence appears in `risks_questions` | Conflicting items excluded from advice per BRD-04 FR-11 |

### Briefing Detailed Sections

| Scenario | Expected |
|----------|----------|
| Source meeting contains raw transcript text | Transcript snippets not included in `previous_relevant_context`; only evidence-backed summaries |
| Evidence snippet contains PII | PII redacted from evidence snippets; `quality_status: "weak_evidence"` shown |
| Stakeholder note contains unsupported personality inference | Not stored; filtered out per BRD-04 FR-13 |
| Resolution note from BRD-03 conflict appears in briefing | Conflict resolution notes not exposed in briefing (resolution is internal) |
| Source meeting title contains PII | Title used for source identification; no raw PII in generated summary text |

### PII in Logs and Metrics

| Forbidden Field | Must Not Appear In |
|----------------|---------------------|
| Briefing generated text | Any log event |
| Participant names/emails from source meetings | Any log event |
| Evidence snippets | Any log event |
| Source meeting titles | Metric labels |
| `briefing_content` field | Prometheus labels |
| `upcoming_meeting_id` (UUID) — allowed | Log events, metric labels |

---

## Timing / Side-Channel Risks in Relatedness Scoring

### Signal Matching Timing

| Scenario | Expected |
|----------|----------|
| Relatedness scoring completes in reasonable time | No timing oracle that reveals match quality via response latency |
| Signal comparison uses constant-time equality check | No timing leak that reveals whether email matched vs title matched |
| Two-signal threshold not inferable from timing | Response time does not reveal which two signals matched |

### Related-Meeting Selection Timing

| Scenario | Expected |
|----------|----------|
| Briefing generation latency | P95 < 60s per BRD-04 NFR; no denial-of-service from concurrent regeneration requests |
| Regeneration does not block upcoming meeting page | Generation is async; user can leave and return |

### Side-Channel in Source Exclusion

| Scenario | Expected |
|----------|----------|
| Exclude then restore a source | Timing of restore response does not reveal whether exclusion was active |
| Exclude a source not in qualifying set | Safe no-op; no error that reveals the full qualifying set |

---

## preparation_status State Machine Authorization

### Status Transition Authorization

| Scenario | Current Status | Requested Action | Expected |
|----------|----------------|------------------|----------|
| User calls regenerate when status is `generating` | `generating` | POST /briefing/regenerate | 409 Conflict or 202 Accepted (queued, not running twice) |
| User calls regenerate when status is `regenerating` | `regenerating` | POST /briefing/regenerate | 409 Conflict or idempotent 202 |
| User excludes source when status is `generating` | `generating` | POST /sources/{id}/exclude | Safe operation; does not affect in-progress generation |
| User restores source when status is `generating` | `generating` | POST /sources/{id}/restore | Safe operation; does not affect in-progress generation |
| User views briefing during `regenerating` status | `regenerating` | GET /briefing | Returns latest completed briefing; does not block |

### preparation_status Values — Correct Enumeration

| Scenario | Expected |
|----------|----------|
| Status field contains unrecognized value | Treated as `failed` or safe error; no panic |
| Status is `null` in database | Defaults to `generating` or safe error state |
| Status transitions to `stale` after source memory reprocess | BRD-04 FR-15 enforced; user offered regeneration |
| Status `ready_with_caveats` shown correctly | User sees clear caveat indication; not treated as `ready` |

### Status Visibility

| Scenario | Expected |
|----------|----------|
| `failed` status does not expose raw error details | Safe user-facing message only; raw exception not in response body |
| `generating` status shown to non-owner | Status visible only to authorized users |
| `stale` status persists across sessions | Stale indicator remains until regeneration or dismissal |

---

## Observability Security

### Metrics Labels

| Forbidden Label Value | Metric |
|----------------------|--------|
| Briefing generated text | Any `cp_pre_call_briefing_*` metric |
| Source meeting title | Any `cp_pre_call_briefing_*` metric |
| Participant name/email | Any `cp_pre_call_briefing_*` metric |
| Evidence snippet | Any label |
| `upcoming_meeting_id` (raw UUID) | Forbidden — use hashed identifier |

### Allowed Observability Fields

| Field | Where Allowed |
|-------|---------------|
| `correlation_id` | All log events |
| `upcoming_meeting_id` (opaque) | Log events, metric labels |
| `trigger_type` | Job log events (`auto`, `manual_regenerate`) |
| `preparation_status` | Log events, metric labels (enum values only) |
| `failure_class` | Failed job log events |
| `source_count_bucket` | `cp_pre_call_briefing_generation_started_total` |
| `generation_duration_ms` | `briefing.job.completed` |

### Log Events

| Event | Required Fields | Forbidden Fields |
|-------|-----------------|-----------------|
| `briefing.generation.started` | `correlation_id`, `trigger_type`, `source_count` | Raw briefing content, participant PII |
| `briefing.generation.completed` | `correlation_id`, `duration_ms`, `preparation_status` | Raw briefing text, evidence snippets |
| `briefing.generation.failed` | `correlation_id`, `failure_class` | Raw meeting content, source meeting titles |
| `briefing.source.excluded` | `correlation_id`, `source_id` | Source meeting content, participant names |
| `briefing.source.restored` | `correlation_id`, `source_id` | Source meeting content |
| `briefing.flag_misconfiguration` | `server_flag`, `browser_flag` | None |

---

## Secure Defaults

| Check | Method | Expected |
|-------|--------|----------|
| Error responses safe | POST with malformed JSON to any briefing endpoint | 400 body contains no raw input reflected, no stack traces |
| Session cookie httpOnly | Inspect Set-Cookie on auth | `HttpOnly` flag present |
| Session cookie SameSite | Inspect Set-Cookie | `SameSite=Lax` or `SameSite=Strict` |
| GET /briefing returns latest active only | Non-owner requests | 403; no version enumeration |
| Version history not exposed to unauthorized | GET /versions with valid UUID, unauthorized user | 403; no version list returned |
| Feature-disabled response is safe | FF=false, GET /briefing | Safe 403; does not reveal existence of briefing |
| Regenerate without auth | No session, POST /briefing/regenerate | 401; no information leakage |

---

## Threat Model: Briefing Content Integrity

| Scenario | Expected |
|----------|----------|
| Weak-evidence item appears as fact | System marks `quality_status: "weak_evidence"`; item caveated in UI |
| Insufficient-evidence item appears as fact | Item not presented as fact; `insufficient_evidence` status applied |
| Conflicting evidence appears in briefing advice | Conflicting item excluded per BRD-04 FR-11 |
| No-prior-memory shell implies continuity | Shell does not include prior decisions, prior actions, or stakeholder memory |
| Briefing source count mismatch | `source_count` field in response matches actual sources used |

---

## Running

```bash
# Requires app running
make eval-security

# Verify flag registration
./evals/architecture/check-feature-flags.sh

# Verify status sync
./scripts/check-status-sync.sh
```