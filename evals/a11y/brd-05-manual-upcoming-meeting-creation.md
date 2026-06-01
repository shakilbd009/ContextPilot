# Accessibility Eval: brd-05-manual-upcoming-meeting-creation

> 🔴 Failing — implementation pending

## Scope

WCAG 2.1 AA accessibility scenarios for Manual Upcoming Meeting Creation (BRD-05).

Source of truth: `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` Accessibility requirement (line 109).

BRD-05 accessibility requirements (WCAG 2.1 AA):
- Keyboard navigation path
- Focus to first error on validation failure
- aria-describedby on form fields
- 50-participant row limit warning
- Duplicate warning in role="alert"
- Form values preserved across validation errors

---

## Scenario: Keyboard navigation — meeting creation form

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is authenticated and on the `/upcoming/new` page
- Keyboard focus is at the page's first focusable element (e.g., skip link or page heading)

### When
User tabs through the form using the keyboard

### Then
- All interactive form fields are reachable via Tab key in a logical order: title → scheduled_start → description → client_or_organization → participant rows → submit
- Tab moves focus forward through all fields and actions
- Shift+Tab moves focus backward through all fields and actions
- Focus indicator is visibly styled (not invisible) on all focused elements
- The submit button is the last focusable element before the form submits (or before validation errors re-focus)

---

## Scenario: Focus management on validation error — focus moves to first invalid field

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is on `/upcoming/new` with an empty form (or with a form that will fail validation)

### When
User submits the form with `title` empty and `scheduled_start` empty (both missing required fields)

### Then
- Form submission returns a validation error response
- Focus is moved to the first invalid field (`title`) so the user can correct without having to tab back through all fields
- The field with the validation error is programmatically associated with its error message (via `aria-describedby`)

### When
User fixes `title` but leaves `scheduled_start` empty and resubmits

### Then
- Focus moves to `scheduled_start` as the next invalid field

---

## Scenario: aria-describedby on form fields — error association

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is on `/upcoming/new`

### When
User submits with a missing title and an invalid email in a participant row

### Then
- The `title` input element has an `aria-describedby` attribute referencing the id of its error message element
- The participant email input element has an `aria-describedby` attribute referencing the id of its error message element
- When the field has an error, the associated error message element has `role="alert"` or is otherwise surfaced to assistive technology

### When
User submits with a valid form

### Then
- All `aria-describedby` attributes continue to point to valid, non-empty description or hint text (not broken references)

---

## Scenario: Screen reader announcement of duplicate warning (role="alert")

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- Authenticated user has an existing upcoming meeting with title "Weekly Sync" scheduled at "2026-06-01 09:00"
- User is on `/upcoming/new`

### When
User enters title "Weekly Sync" and `scheduled_start` "2026-06-01 09:00" (exact duplicate within the same minute) and focuses away from the scheduled_start field (or prior to form submission)

### Then
- A warning is displayed in a live region with `role="alert"` announcing the duplicate meeting
- The duplicate warning does not block form submission; it is an advisory only

### When
User ignores the warning and submits

### Then
- Form submission succeeds
- A new meeting record is created

---

## Scenario: Screen reader announcement of participant limit warning (50 rows)

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is on `/upcoming/new`

### When
User adds 49 participant rows to the form

### Then
- No warning is shown (threshold not yet reached)

### When
User adds the 50th participant row

### Then
- A warning is announced to screen readers in a live region (e.g., `role="alert"`) indicating the 50-participant limit has been reached
- The warning is visible and programmatically exposed

### When
User attempts to add a 51st participant row

### Then
- Form submission is rejected with `too_many_participants` validation class
- Error message is associated with the participant section via `aria-describedby`
- The 50-participant warning remains visible/announced

---

## Scenario: Form state preservation after failed submission

### Given
- `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`
- User is on `/upcoming/new`

### When
User fills in:
- title: "Q3 Kickoff"
- scheduled_start: "2026-07-01 10:00"
- description: "Initial planning session for Q3 objectives"
- client_or_organization: "Acme Corp"
- participants:
  - display_name: "Alice", email: "alice@example.com"
  - display_name: "Bob", email: "bob@example.com"

And submits the form with an intentionally invalid field (e.g., `scheduled_start` set to a time in the past to trigger a validation error)

### Then
- After validation failure, all form fields retain their submitted values:
  - `title` field shows "Q3 Kickoff"
  - `scheduled_start` field shows "2026-07-01 10:00"
  - `description` field shows "Initial planning session for Q3 objectives"
  - `client_or_organization` field shows "Acme Corp"
  - Both participant rows are present with correct display names and emails
- The user does not need to re-enter any valid data to correct the invalid field

### When
User corrects the invalid field (sets `scheduled_start` to a valid future time) and resubmits

### Then
- Form submission succeeds
- Meeting detail page shows all originally entered values (including preserved form state from the failed attempt)