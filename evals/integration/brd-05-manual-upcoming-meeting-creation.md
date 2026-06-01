# Integration Eval: brd-05-manual-upcoming-meeting-creation

> 🔴 Failing — implementation pending

## Scope

API contract integration tests for Manual Upcoming Meeting Creation (BRD-05): all CRUD endpoints, feature flag gating, owner-only authorization, validation error responses, BRD-04 trigger integration, form echo on error, and /ready storage health.

Source of truth: `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` FR and AC tables.

---

## API Contract Tests

### POST /upcoming — Feature flag disabled returns safe response

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=false`
- Authenticated request with valid meeting payload

#### When
`POST /upcoming` is called with:
```json
{
  "title": "Q3 Planning",
  "scheduled_start": "2099-01-01T12:00:00Z"
}
```

#### Then
- Response status is `200 OK` or `400 Bad Request` (not `401`, `403`, `404`, or `500`)
- Response body is valid JSON
- Body contains `{"code": "feature_disabled", "message": "Upcoming meetings are not available"}` or equivalent safe structure
- No upcoming meeting record was created

---

### POST /upcoming — Successful creation returns 201

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U with session cookie/token
- Valid payload: `title` (non-empty), `scheduled_start` (≥15 min in future)

#### When
`POST /upcoming` with:
```json
{
  "title": "Q3 Planning Review",
  "scheduled_start": "<valid future ISO timestamp>",
  "description": "Discuss roadmap priorities",
  "client_or_organization": "Acme Corp",
  "participants": [
    {"display_name": "Alice", "email": "alice@example.com", "organization": "Acme Corp"},
    {"display_name": "Bob", "email": "bob@example.com", "organization": "Beta LLC"}
  ]
}
```

#### Then
- Response status `201 Created`
- Response body contains `id` (UUID), `title`, `scheduled_start`, `status: "scheduled"`, `created_by` (matches authenticated user)
- `client_or_organization` is "Acme Corp"
- `participants` array has 2 entries with correct display_name/email/organization
- `created_at` and `updated_at` are set

---

### POST /upcoming — Missing title returns 400 with form echo

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U
- Payload: `title: ""`, valid `scheduled_start`, valid participants

#### When
`POST /upcoming` with empty title

#### Then
- Response status `400 Bad Request`
- `code` or `validation_class` is `missing_title`
- Response body echoes back `scheduled_start` and `participants` (server-authoritative form preservation per FR-15)

---

### POST /upcoming — scheduled_start too soon returns 400

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U

#### When
`POST /upcoming` with `scheduled_start` = server time + 5 minutes

#### Then
- Response status `400 Bad Request`
- `code` or `validation_class` is `invalid_scheduled_start`
- Submitted form values are echoed back in response

---

### POST /upcoming — Participant without identity returns 400

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U

#### When
`POST /upcoming` with participant row: `{"display_name": null, "email": null}`

#### Then
- Response status `400 Bad Request`
- `code` or `validation_class` is `participant_missing_identity`

---

### POST /upcoming — Too many participants returns 400

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U

#### When
`POST /upcoming` with 51 participant rows, all with valid identity (display_name or email)

#### Then
- Response status `400 Bad Request`
- `code` or `validation_class` is `too_many_participants`

---

### GET /upcoming — Returns only authenticated user's meetings

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user U owns meetings M1, M2
- Another user V owns meeting M3

#### When
User U calls `GET /upcoming` with their session

#### Then
- Response status `200 OK`
- Response body is a JSON array
- M1 and M2 are present; M3 is NOT present
- Meetings are sorted by `scheduled_start` ascending (nearest first)
- Cancelled meetings are excluded by default

---

### GET /upcoming/{id} — Owner can view their meeting

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1 (UUID known)

#### When
User U calls `GET /upcoming/{M1.id}`

#### Then
- Response status `200 OK`
- Response body contains full meeting: `id`, `title`, `scheduled_start`, `status`, `description`, `client_or_organization`, `participants`, `created_by`, `created_at`, `updated_at`

---

### GET /upcoming/{id} — Non-owner returns 403

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1
- User V is authenticated but does not own M1

#### When
User V calls `GET /upcoming/{M1.id}`

#### Then
- Response status `403 Forbidden` or `404 Not Found`
- No meeting details are leaked in the response

---

### PATCH /upcoming/{id} — Within edit window allows update

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1 with `scheduled_start` set to now + 30 minutes
- Edit is within the 15-minute post-scheduled_start window

#### When
User U calls `PATCH /upcoming/{M1.id}` with `{"title": "Updated Title"}`

#### Then
- Response status `200 OK`
- `title` is updated to "Updated Title"
- `updated_at` is refreshed

---

### PATCH /upcoming/{id} — After edit window returns 422

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1 with `scheduled_start` set to 20 minutes ago
- Edit window (15 min after scheduled_start) has expired

#### When
User U calls `PATCH /upcoming/{M1.id}` with `{"title": "Updated Title"}`

#### Then
- Response status `422 Unprocessable Entity` or `400 Bad Request`
- `code` or `validation_class` is `not_editable`

---

### POST /upcoming/{id}/cancel — Within cancel window succeeds

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1 with `scheduled_start` set to now + 30 minutes

#### When
User U calls `POST /upcoming/{M1.id}/cancel`

#### Then
- Response status `200 OK`
- Meeting `status` is `cancelled` in database
- Meeting is excluded from default `GET /upcoming` results

---

### POST /upcoming/{id}/cancel — After cancel window returns 422

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1 with `scheduled_start` set to 20 minutes ago

#### When
User U calls `POST /upcoming/{M1.id}/cancel`

#### Then
- Response status `422 Unprocessable Entity` or `400 Bad Request`
- `code` or `validation_class` is `not_cancellable`

---

### POST /upcoming/{id}/cancel — Non-owner returns 403

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- User U owns meeting M1
- User V is authenticated but does not own M1

#### When
User V calls `POST /upcoming/{M1.id}/cancel`

#### Then
- Response status `403 Forbidden` or `404 Not Found`

---

### AC-13 — BRD-04 trigger on creation when both flags enabled

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`
- User U calls `POST /upcoming` with a valid meeting

#### When
Creation succeeds with `201 Created`

#### Then
- `POST /upcoming` response is returned within 500ms (briefing trigger is async, does not block creation)
- A BRD-04 briefing generation job is queued asynchronously
- No error related to briefing generation appears in the creation response

---

### AC-15 — BRD-04 disabled does not block BRD-05 creation

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=false`
- User U calls `POST /upcoming` with a valid meeting

#### When
Creation request is made

#### Then
- Response status `201 Created` (briefing trigger skipped safely)
- No error about BRD-04 being disabled appears in response
- Logs indicate briefing trigger was skipped (low-cardinality `skipped_reason: "feature_disabled"` or similar)

---

### AC-19 — GET /ready reports storage health for upcoming meetings

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Upcoming meeting storage (PostgreSQL) is healthy

#### When
`GET /ready` is called

#### Then
- Response status is `200 OK`
- Response includes upcoming meeting storage health indicator
- If BRD-04 generation dependencies are degraded but upcoming meeting storage is healthy, response is still `200 OK` (not `503`)
- No raw meeting titles, participant names, or other private content appears in `/ready` response

---

### Authenticated request without session returns 401

#### Given
- No authenticated session cookie/token

#### When
Any of the following are called:
- `POST /upcoming`
- `GET /upcoming`
- `GET /upcoming/{some-uuid}`
- `PATCH /upcoming/{some-uuid}`
- `POST /upcoming/{some-uuid}/cancel`

#### Then
- Response status `401 Unauthorized`
- No internal system details are leaked in the response

---

### Participant identity validation (FR-06)

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user

#### When
`POST /upcoming` is called with a participant row that has:
- `display_name: ""` (empty string) and `email: null`

#### Then
- Response status `400 Bad Request`
- `validation_class` is `participant_missing_identity`

#### When
`POST /upcoming` is called with a participant row:
- `display_name: "Alice"` only (no email)

#### Then
- Response status `201 Created` (display_name is sufficient identity per FR-6)

#### When
`POST /upcoming` is called with a participant row:
- `email: "alice@example.com"` only (no display_name)

#### Then
- Response status `201 Created` (email is sufficient identity per FR-6)

---

### Invalid email format validation

#### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user

#### When
`POST /upcoming` is called with participant: `{"display_name": "Alice", "email": "not-a-valid-email"}`

#### Then
- Response status `400 Bad Request`
- `validation_class` is `invalid_email`