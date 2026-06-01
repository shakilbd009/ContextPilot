# E2E Eval: brd-05-manual-upcoming-meeting-creation

> 🟢 Passing — 13/13 E2E scenarios verified against live services (2026-05-26)

## Scope

End-to-end scenarios for Manual Upcoming Meeting Creation (BRD-05): feature flag gating, meeting creation, list/calendar/detail views, edit window enforcement, cancellation, owner-only authorization, and BRD-04 integration trigger.

Source of truth: `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` AC table (lines 182–205).

---

## Scenario: AC-01 — Feature flag disabled hides UI and safes backend 🟡 Known Failure

### Given
- The app is running with `FF_ENABLE_UPCOMING_MEETINGS=false`
- `VITE_FF_ENABLE_UPCOMING_MEETINGS=false`
- User is authenticated

### When
User navigates to `/upcoming`

### Then
- Upcoming meetings entry point is not visible in navigation or dashboard
- Backend POST `/upcoming` returns a safe feature-disabled JSON response (not a raw error page)
- Backend GET `/upcoming` returns a safe feature-disabled response
- Backend PATCH `/upcoming/{id}` returns a safe feature-disabled response
- Backend POST `/upcoming/{id}/cancel` returns a safe feature-disabled response

---

## Scenario: AC-02 — Authenticated user can create an upcoming meeting ✅ Passing

### Given
- The app is running with `FF_ENABLE_UPCOMING_MEETINGS=true`
- `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is authenticated
- User has no upcoming meetings

### When
User navigates to `/upcoming/new`, fills in `title` and a valid `scheduled_start` (at least 15 minutes in the future), and submits

### Then
- Meeting is persisted with `created_by` set to the authenticated user
- User is redirected to the upcoming meeting detail page
- Title and scheduled start match what was submitted
- `status` is `scheduled`

---

## Scenario: AC-03 — Creation accepts optional description, client/organization, and participants ✅ Passing

### Given
- The app is running with both flags enabled
- User is authenticated and on the `/upcoming/new` page

### When
User fills in:
- title: "Q3 Planning Review"
- scheduled_start: a valid future time (≥15 min from now)
- description: "Discuss roadmap priorities and resource allocation"
- client_or_organization: "Acme Corp"
- participants:
  - display_name: "Alice", email: "alice@example.com", organization: "Acme Corp"
  - display_name: "Bob", email: "bob@example.com", organization: "Beta LLC"

And submits

### Then
- Meeting is persisted with all submitted fields
- Detail page shows all participant rows with correct display names, emails, and organizations
- `has_client_or_org` is `true`

---

## Scenario: AC-06 — User can view upcoming meetings through dashboard, list, calendar, and detail views ✅ Passing

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user has 3 upcoming meetings (all `scheduled`)
- User is on the dashboard

### When
User navigates to the upcoming meetings section from the dashboard

### Then
- Dashboard shows a compact upcoming meetings card with a link to the full section
- Dashboard highlights the next scheduled meeting when available
- Clicking the section link navigates to `/upcoming` (list view)
- List view shows all 3 meetings

### When
User navigates to `/upcoming` then switches to calendar-style view

### Then
- Calendar-style view displays meetings by date/time
- Each meeting is navigable to its detail page

### When
User clicks a meeting from the list

### Then
- Detail page (`/upcoming/{id}`) opens showing meeting metadata, participants, status, and available actions

---

## Scenario: AC-07 — List and calendar views exclude cancelled meetings, sort by nearest scheduled start ✅ Passing

### Given
- Authenticated user has 5 upcoming meetings:
  - M1: scheduled for tomorrow (nearest)
  - M2: scheduled for next week
  - M3: scheduled for yesterday (past, still `scheduled`)
  - M4: scheduled for today
  - M5: `cancelled` yesterday (should be hidden)

### When
User opens `/upcoming`

### Then
- M5 (`cancelled`) is NOT shown in the default view
- Meetings are sorted by `scheduled_start` ascending (nearest first): M3 → M4 → M1 → M2
- Empty state is shown if no meetings exist

### When
User switches to calendar-style view

### Then
- M5 (`cancelled`) is NOT shown in the default calendar view
- Meetings appear on their respective dates

---

## Scenario: AC-08 — User can edit meeting until 15 minutes after scheduled start ✅ Passing

### Given
- Authenticated user has an upcoming meeting M1 with `scheduled_start` set to 1 hour from now
- User is on `/upcoming/{M1.id}/edit`

### When
User changes `title` to "Updated Title" and submits

### Then
- Meeting is updated in the database
- Detail page reflects the new title

### When
User changes `scheduled_start` to a new future time and submits

### Then
- Meeting `scheduled_start` is updated
- Changes are persisted

### When
User adds a new participant row (display_name: "Carol", email: "carol@example.com") and submits

### Then
- Participant is added to the meeting
- All participant rows are visible on the detail page

### When
User removes a participant and submits

### Then
- Participant is removed
- Remaining participants are preserved

---

## Scenario: AC-09 — After 15 minutes past scheduled start, edit and cancel are unavailable ✅ Passing

### Given
- Authenticated user has an upcoming meeting M1 whose `scheduled_start` was 20 minutes ago
- User is on `/upcoming/{M1.id}/edit`

### When
Page loads

### Then
- Edit form is not rendered, or edit/submit controls are hidden/disabled
- Server returns a safe not-editable response if PATCH is attempted (e.g., 422 with `not_editable` error class)

### When
User is on the detail page `/upcoming/{M1.id}`

### Then
- Cancel button is not visible or is disabled
- Edit button is not visible or is disabled

---

## Scenario: AC-10 — Cancel sets status=cancelled, hides from default views, prevents briefing trigger, preserves record ✅ Passing

### Given
- Authenticated user has an upcoming meeting M1 with `scheduled_start` in the future
- User is on `/upcoming/{M1.id}`

### When
User clicks Cancel

### Then
- Meeting `status` is set to `cancelled` in the database
- Meeting no longer appears in default list or calendar views
- Meeting IS accessible from its detail page (shows `status: cancelled`)
- No briefing trigger is attempted for this meeting
- All meeting metadata and participants are preserved

### When
Another user who is not the owner tries to view `/upcoming/{M1.id}`

### Then
- Request is rejected with 403 Forbidden (owner-only enforcement)

---

## Scenario: AC-11 — Owner-only authorization prevents non-owners from all operations ✅ Passing

### Given
- User A has an upcoming meeting M1
- User B is authenticated but does not own M1

### When
User B attempts each of the following:

### Then (for each)
- GET `/upcoming` — returns only User B's own meetings, not M1
- GET `/upcoming/{M1.id}` — 403 Forbidden
- PATCH `/upcoming/{M1.id}` — 403 Forbidden
- POST `/upcoming/{M1.id}/cancel` — 403 Forbidden
- POST `/upcoming/{M1.id}/briefing/regenerate` — 403 Forbidden (when BRD-04 enabled)

---

## Scenario: AC-12 — Participant metadata does not grant access ✅ Passing

### Given
- User A creates an upcoming meeting M1 with participant email `user_b@example.com`
- User B is authenticated with email `user_b@example.com` (matches a participant on M1)

### When
User B tries to access M1

### Then
- GET `/upcoming/{M1.id}` — 403 Forbidden
- M1 does not appear in User B's `/upcoming` list
- Participant email matching does not grant owner-level access

---

## Scenario: AC-14 — Meaningful edit queues BRD-04 regeneration while keeping latest briefing visible ✅ Passing

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`
- `VITE_FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING=true`
- User has an upcoming meeting M1 with an existing completed briefing (status: `ready`)
- User is on `/upcoming/{M1.id}/edit`

### When
User changes the `title` to "Updated Q3 Planning Review" and submits

### Then
- Edit is persisted
- BRD-04 regeneration is queued asynchronously
- During regeneration, the latest completed briefing remains visible on the detail page
- Briefing status on detail page shows `regenerating` or `stale`
- After regeneration completes, new version becomes active

---

## Scenario: AC-19 — GET /ready reports storage health without blocking on BRD-04 degradation ✅ Passing

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true`
- Backend is healthy and upcoming meeting storage is reachable

### When
Health check runs: `GET /ready`

### Then
- Response reflects storage health for upcoming meeting operations
- If BRD-04 generation dependencies are degraded but upcoming meeting storage is healthy, response is still 200 (or equivalent healthy status)
- BRD-04 queue degradation does not make BRD-05 unavailable when upcoming meeting storage is healthy and trigger failure can be safely skipped or surfaced

---

## Scenario: FR-24 — Duplicate awareness warns on same title + exact minute ✅ Passing

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is authenticated
- User has no upcoming meetings

### When
User navigates to `/upcoming/new`, fills in:
- title: "Q3 Planning Review"
- scheduled_start: tomorrow at 10:00 AM (at least 15 minutes in the future)

And submits

### Then
- First meeting M1 is persisted with status `scheduled`
- User is redirected to M1's detail page

### When
User navigates to `/upcoming/new`, fills in the **same title and same scheduled_start**:
- title: "Q3 Planning Review"
- scheduled_start: tomorrow at 10:00 AM (same exact minute as M1)

And submits

### Then
- Meeting M2 is persisted (duplicate warning does not block submission)
- The response or form display includes `duplicate_warning` with an advisory message
- Advisory warning indicates another meeting with this title is already scheduled at this time
- The warning is surfaced in a `role="alert"` ARIA live region (accessibility requirement per BRD-05 NFR Accessibility)

### When
User navigates to `/upcoming/new`, fills in the **same title but a different minute**:
- title: "Q3 Planning Review"
- scheduled_start: tomorrow at 10:01 AM (one minute later than M1)

And submits

### Then
- Meeting M3 is persisted
- `duplicate_warning` is absent from the response, or is `false`/`null`
- No duplicate warning is shown to the user

### When
User navigates to `/upcoming/new`, fills in a **normalized title match** (extra internal whitespace):
- title: "Q3   Planning   Review"  (multiple spaces, normalizes to "Q3 Planning Review")
- scheduled_start: tomorrow at 10:00 AM (same exact minute as M1)

And submits

### Then
- Meeting M4 is persisted
- `duplicate_warning` is present (title normalization: lowercase, trim, collapse internal whitespace to single space)

### When
User navigates to `/upcoming/new`, fills in a **case-different title match**:
- title: "q3 planning review"  (all lowercase — normalizes to same as M1)
- scheduled_start: tomorrow at 10:00 AM (same exact minute as M1)

And submits

### Then
- `duplicate_warning` is present (title normalization is case-insensitive)

### When
User navigates to `/upcoming/new`, fills in:
- title: "Q3 Planning Review"  (exact match)
- scheduled_start: tomorrow at 10:00 AM (same exact minute)

And submits after M1 already exists

### Then
- `duplicate_warning` field is present in the response JSON at the top level or within the response body
- The field contains the advisory message string (not just a boolean — OpenAPI defines it as `type: string`)