# Security Eval: brd-05-manual-upcoming-meeting-creation

> 🔴 Failing — implementation pending

## Scope

Security checks for Manual Upcoming Meeting Creation (BRD-05): authentication on all `/upcoming` endpoints, owner-only ACL enforcement via `created_by = session_user_id` predicate, input validation on meeting/participant creation fields, feature flag bypass attack surface, PII handling in participant metadata (display_name, email, organization), and edit window race condition surface.

---

## Authentication & Authorization

### All Upcoming Meeting Endpoints Require Auth

| Endpoint | Method | Expected Without Session |
|----------|--------|--------------------------|
| GET /upcoming | No cookie | 401 Unauthorized |
| POST /upcoming | No cookie | 401 Unauthorized |
| GET /upcoming/{id} | No cookie | 401 Unauthorized |
| PATCH /upcoming/{id} | No cookie | 401 Unauthorized |
| POST /upcoming/{id}/cancel | No cookie | 401 Unauthorized |

### Owner-Only ACL

| Scenario | Expected |
|----------|----------|
| Authenticated, not owner of meeting | 403 Forbidden on GET, PATCH, POST cancel |
| Authenticated, owner | 200 on GET, PATCH, POST cancel |
| Owner edits own meeting | 200 + updated meeting |
| Non-owner attempts cancel | 403 Forbidden |
| Unauthenticated request | 401 Unauthorized |

### Feature Flag Access Control

| Scenario | Expected |
|----------|----------|
| FF=disabled, any request to /upcoming/* | 404 Not Found (flag-gated surface) |
| FF=enabled, authenticated owner | Normal 2xx response |
| FF=enabled, authenticated non-owner | 403 Forbidden |
| FF=enabled, unauthenticated | 401 Unauthorized |

---

## Input Validation

### Meeting Creation (POST /upcoming)

| Field | Test | Expected |
|-------|------|----------|
| title | Missing title | 400 Bad Request |
| title | Empty string | 400 Bad Request |
| title | Whitespace only | 400 Bad Request |
| scheduled_start | Missing | 400 Bad Request |
| scheduled_start | Not ISO 8601 | 400 Bad Request |
| scheduled_start | In the past | 400 Bad Request |
| scheduled_start | Not 15-minute floor (e.g. 10:07) | 400 Bad Request |
| scheduled_start | Valid future datetime | 201 Created |
| description | Valid string | 201 Created |
| client_or_organization | Valid string | 201 Created |
| client_or_organization | Empty | 201 Created (optional) |
| FF=disabled | POST /upcoming | 404 Not Found |

### Participant Input (POST /upcoming + nested participant)

| Field | Test | Expected |
|-------|------|----------|
| participants[].display_name | Missing + email absent | 400 Bad Request |
| participants[].email | Invalid email format | 400 Bad Request |
| participants[].email | Valid email | 201 Created |
| participants[].display_name | Valid + email absent | 201 Created |
| participants[].display_name | Valid + email valid | 201 Created |
| participants[].organization | Valid string | 201 Created |
| participants[].organization | Empty | 201 Created (optional) |

### Edit (PATCH /upcoming/{id})

| Field | Test | Expected |
|-------|------|----------|
| title | Empty string | 400 Bad Request |
| scheduled_start | Not 15-minute floor | 400 Bad Request |
| scheduled_start | Before edit window (now > 15min past scheduled_start) | 409 Conflict |
| scheduled_start | Within edit window | 200 OK |
| FF=disabled | PATCH /upcoming/{id} | 404 Not Found |

### Cancel (POST /upcoming/{id}/cancel)

| Scenario | Expected |
|----------|----------|
| Cancel scheduled meeting by owner | 200 OK, status=cancelled |
| Cancel already-cancelled meeting | 409 Conflict |
| FF=disabled | POST /upcoming/{id}/cancel | 404 Not Found |

---

## PII Handling

| Scenario | Expected |
|----------|----------|
| GET /upcoming returns other users' meetings | Response omits other users' meetings (owner-only list) |
| Participant email in response | Email returned only for meetings owned by requester |
| Participant display_name | Returned only for meetings owned by requester |
| Participant organization | Returned only for meetings owned by requester |
| Log entry contains participant email | Email must not appear in plaintext logs |
| Log entry contains meeting title | Title may appear in logs (non-PII) |

---

## Feature Flag Bypass

| Attack Vector | Expected |
|---------------|----------|
| Manipulate VITE_FF_ENABLE_UPCOMING_MEETINGS client-side | Server must still enforce FF — backend gate on all handlers |
| Direct API request to /upcoming/* when FF=disabled | 404 from server, not 200 |
| Replay authenticated request after flag disabled | 404, not data exposure |

---

## Race Conditions & Timing

| Scenario | Expected |
|----------|----------|
| Edit within edit window (15 min) | 200, edit succeeds |
| Edit after edit window expires | 409 Conflict |
| Cancel after edit window expires | 200 OK (cancel always allowed) |
| Concurrent edit by owner | Last write wins, no data corruption |
| Concurrent edit + cancel | Cancel wins, meeting cancelled |

---

## Edit Window

| Scenario | Expected |
|----------|----------|
| scheduled_start = now + 30 min, edit immediately | 200 OK |
| scheduled_start = now - 10 min, edit now | 409 Conflict (past edit window) |
| scheduled_start = now + 15 min, edit exactly at 15-min mark | 200 OK |
| scheduled_start = now + 16 min, edit exactly at 16-min mark | 409 Conflict |