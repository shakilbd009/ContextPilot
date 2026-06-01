# Integration Eval: brd-02-manual-meeting-import

> 🟢 Passing — backend handler tests pass (`go test ./internal/meeting/...`), SvelteKit `/api/meetings` proxy correctly forwards POST/GET to Go backend

## Scope

Integration tests for Manual Meeting Import (BRD-02): API contract for `POST /meetings`, `GET /meetings`, `GET /meetings/:id`, idempotency enforcement, and data persistence across the full stack.

---

## API Contract Tests

### POST /meetings

| Scenario | Request | Expected |
|----------|---------|----------|
| Valid minimal meeting | title, completedAt, 1 participant, transcript, idempotencyToken | 201, `{"id": "<uuid>", "redirect": "/meetings/<id>"}` |
| Valid notes-only meeting | title, completedAt, 1 participant, notes only, idempotencyToken | 201 |
| Valid meeting with both content types | transcript + notes within 50k chars, idempotencyToken | 201, both fields preserved on retrieval |
| Unauthenticated request | No valid session cookie | 401 Unauthorized |
| Flag disabled server-side | FF_ENABLE_MANUAL_MEETING_IMPORT=false | 403 Forbidden |
| Missing idempotency token | Valid payload, no token field | 400 |
| Empty body | `{}` | 400, multiple field errors |

### GET /meetings

| Scenario | Expected |
|----------|----------|
| Empty state | `{"meetings": []}` when no meetings exist |
| With meetings | Meetings listed with id, title, completedAt, contentSource; no transcript/notes content in list items |
| Unauthenticated | 401 Unauthorized |

### GET /meetings/:id

| Scenario | Request | Expected |
|----------|---------|----------|
| Existing meeting | Authenticated, valid meeting ID | 200, full meeting object with participants array |
| Transcript + notes both preserved | POST with both fields, GET | Both `transcript` and `notes` are non-null and match submitted values |
| Participant fields preserved | POST with email, organization, role | GET returns participant records with all fields intact |
| Non-existent meeting | Valid UUID not in DB | 404 Not Found |
| Unauthenticated | No valid session | 401 Unauthorized |

---

## Data Persistence Tests

### Participant Record Structure

| Scenario | Method | Expected |
|----------|--------|----------|
| Single participant | POST meeting, then GET /meetings/:id | 1 participant record with correct `displayName` |
| Multiple participants | POST with 3 participants | GET returns 3 participant records in stable order |
| Participant optional fields null | POST without email/org/role | GET returns participant with null for those fields |

### Idempotency Enforcement

| Scenario | Method | Expected |
|----------|--------|----------|
| Same token submitted twice | POST with token T1 → 201; POST with same T1 → 409 | Second response has `{"error": "duplicate", "meetingId": "<original>"}` |
| Different token per submission | POST with T1 → 201; POST with T2 → 201 | Two distinct meeting records created |

### Content Source Derivation (API-level)

| Scenario | POST body | Expected in GET |
|----------|-----------|-----------------|
| Transcript only | transcript="hello", notes="" | `contentSource: "transcript"` |
| Notes only | transcript="", notes="hello" | `contentSource: "notes"` |
| Both | transcript="a", notes="b" | `contentSource: "both"` |

### Character Limit Enforcement (API-level)

| Scenario | Request | Expected |
|----------|---------|----------|
| Exactly 50,000 chars combined | transcript: 25000 "x", notes: 25000 "y" | 201 |
| 50,001 chars combined | transcript: 25001 "x", notes: 25000 "y" | 400, `field: "content"` |

---

## Running

```bash
# Requires backend and frontend both running
make eval-integration
# or
go test ./tests/integration/... -v
```