# Security Eval: brd-02-manual-meeting-import

> 🟡 Known Failure — 1 blocking finding: frontend POST /api/meetings is a STUB (returns fake UUID without calling Go backend); security eval cannot be fully exercised until t_a2c3d3d5 is resolved. SQL injection blocked by parameterized queries, flag bypass blocked, no forbidden field logging.

## Scope

Security checks for Manual Meeting Import (BRD-02): input sanitization, rate limiting, authentication enforcement, authorization, and sensitive data handling.

---

## Input Sanitization

| Check | Method | Expected |
|-------|--------|----------|
| XSS in title | POST with `title: "<script>alert(1)</script>"` | 201 or 400; GET /meetings/:id does not execute the script |
| XSS in transcript | POST with `transcript: "<img src=x onerror=alert(1)>"` | 201; GET response does not execute the payload |
| XSS in participant displayName | POST with `displayName: "<svg onload=alert(1)>"` | 201; GET response sanitized |
| XSS in participant email | POST with `email: "test<script>@example.com"` | 201 or 400 |
| SQL injection attempt in title | `title: "'; DROP TABLE meetings; --"` | 201 or 400; no table deletion |
| HTML in notes | POST with `notes: "<h1>Big Header</h1>"` | 201; GET returns raw text, not rendered HTML |
| Very long title (501+ chars) | `title: "a" × 501` | 400, `field: "title"` |
| Invalid UTF-8 in transcript | POST with malformed UTF-8 bytes | 400 or sanitized; no crash |
| Null bytes in title | `title: "Test\x00Meeting"` | 400 or null stripped before storage |

---

## Rate Limiting

| Check | Method | Expected |
|-------|--------|----------|
| Burst limit | 100+ rapid POST /meetings requests | 429 Too Many Requests after threshold |
| Rate limit header | After exceeding limit | `Retry-After` header present |
| Limit resets | Wait after `Retry-After` period, POST again | 201 again |

---

## Authentication & Authorization

| Check | Method | Expected |
|-------|--------|----------|
| Unauthenticated POST | No session cookie, POST /meetings | 401 Unauthorized |
| Unauthenticated GET /meetings | No session cookie | 401 Unauthorized |
| Unauthenticated GET /meetings/:id | No session cookie | 401 Unauthorized |
| Flag disabled blocks write | FF_ENABLE_MANUAL_MEETING_IMPORT=false, authenticated POST | 403 Forbidden |
| Flag disabled blocks read | FF_ENABLE_MANUAL_MEETING_IMPORT=false, authenticated GET /meetings | 403 Forbidden |
| CSRF on form submit | POST without CSRF token | 403 Forbidden (if CSRF protection enabled) |

---

## Sensitive Data Handling

| Check | Method | Expected |
|-------|--------|----------|
| Transcript not in GET /meetings list | POST with transcript, GET /meetings | Meeting list items do not include transcript text |
| Transcript not in metric labels | POST with transcript, inspect metrics | No transcript content in `cp_manual_meeting_import_completed_total` labels |
| Transcript not in logs | POST with transcript, inspect log output | No raw transcript text in `meeting.import.completed` log event |
| Participant names not in logs | POST with participants, inspect logs | No participant displayNames in log fields |
| Meeting title not in logs | POST with title, inspect logs | No title in log event fields |
| Flag misconfiguration detected | Server=false, Browser=true | `cp_manual_meeting_import_flag_misconfiguration_total` incremented; `meeting.import.flag_misconfiguration` log event emitted |

---

## Secure Defaults

| Check | Method | Expected |
|-------|--------|----------|
| No secret in idempotency token response | Duplicate token returns 409 | Response does not expose internal state or stack traces |
| Error responses are safe | POST with `{"title": "<script>"}` | 400 error body contains no raw input reflected |
| Session cookie is httpOnly | Inspect Set-Cookie header | `HttpOnly` flag present |
| Session cookie is SameSite | Inspect Set-Cookie header | `SameSite=Lax` or `SameSite=Strict` |

---

## Running

```bash
# Requires app running
make eval-security
```