# ADR-004: Manual Meeting Import — Form Content Preservation on Validation Failure

> Status: Accepted

## Context

BRD-02 FR-55 requires: "Preserve user-entered form content when validation fails." The BRD does not specify the mechanism. The refiner review identified this as needing a decision. Options include:
1. Server echo: re-render the form with the submitted values returned in the error response body
2. Client-side sessionStorage: store form state on submit, restore on re-render
3. SvelteKit's built-in form enhancement with `use:enhance`

## Decision

We will use **SvelteKit's native form enhancement (`use:enhance`) with progressive enhancement** — submitting via fetch, with the server returning field-level validation errors alongside the original submitted values in the error response body. The SvelteKit form action handles the re-render with populated fields.

For the `use:enhance` approach:
- The client-side form action submits via fetch (no full-page reload)
- On validation failure, the server returns a 400 with field-level errors AND the submitted values
- The form re-renders with the returned values pre-populating each field
- The `use:enhance` hook updates the DOM in place, preserving scroll position and partial user edits

SessionStorage is used only as a **fallback** for catastrophic failures (e.g., server crash mid-submit, navigation away during in-flight request) where the user might lose content — not as the primary mechanism.

## Rationale

SvelteKit's form actions are the idiomatic approach for this framework and provide:
- No full-page reload on submission (better UX)
- Built-in error and value binding to form fields
- Native CSRF protection via SvelteKit's origin checking
- Server-side handling of all validation (per ADR-002)
- Progressive enhancement — works without JavaScript (though without the fetch-based experience)

The error response body carries the submitted values back, so the server echo is clean: no extra storage needed, no stale session data.

SessionStorage as a fallback for the edge case of in-flight navigation or crash is low-cost and handles the "user accidentally hit back button" scenario mentioned in BRD-02 Could Have (line 82: "draft warning before navigating away with unsaved content").

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| No-JS baseline | Without JavaScript, `use:enhance` degrades to a full-page form POST/reload cycle — acceptable per BRD-01 browser support baseline |
| SessionStorage size limits | Browsers limit sessionStorage to ~5MB; the 50K character limit means transcript alone can approach this. We store only a draft flag + timestamp, not full content, to avoid this limit |
| Stale data on concurrent tabs | If a user has two `/meetings/new` tabs open, sessionStorage can carry stale data from a different tab's draft — mitigated by using a draft token per session |

## Consequences

1. The server error response shape must include submitted field values alongside field-level error messages so the form can re-populate.
2. SessionStorage stores only `{ hasDraft: true, savedAt: timestamp }` — not full form content — to stay well under the 5MB limit.
3. The `use:enhance` hook must handle the case where the server returns a redirect (success) vs re-render with errors.
4. The form's `action` URL must match the SvelteKit form action convention (`?/create`) for `use:enhance` to work correctly.
5. Duplicate submission prevention (AC-16) is handled by disabling the save button in the UI and by the server using an idempotency token — a hidden UUID generated on page load and included in the form.

## Alternatives Considered

### SessionStorage as primary
Rejected because: 50K characters in transcript alone could exceed sessionStorage limits on some browsers. Additionally, sessionStorage is per-origin and per-tab, so draft recovery across tabs is not straightforward.

### Full-page server render (no `use:enhance`)
Rejected because: Forces a full-page reload on every validation failure, which is poor UX especially with 50K-character inputs where the browser must re-render a large text area with all content.

## Date

2026-05-20