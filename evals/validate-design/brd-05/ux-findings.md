# UX Findings: BRD-05 Manual Upcoming Meeting Creation

> Validator: Hermes Agent (validator profile)
> Source: `specs/domain/brd-05-manual-upcoming-meeting-creation.md`
> Date: 2026-05-24
> Focus: WCAG 2.1 AA, form preservation, participant ergonomics, duplicate awareness, validation error class messaging

---

## 1. Missing BRD-05 Creation Form — No Upcoming Meeting UI Exists

**Spec requirement:**
> FR-2: Full upcoming meetings section. The feature provides... `/upcoming/new` creation flow... (line 50)
> FR-15: Validation and form preservation. Submitted form values are preserved and echoed back after validation failures. (line 64)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Frontend | **Critical** | The page at `frontend/src/routes/meetings/new/+page.svelte` renders `ManualMeetingImportForm`, a component for **BRD-02** (completed meeting import). There is no `frontend/src/routes/upcoming/new/+page.svelte` or any `/upcoming/*` route in the codebase. The BRD-05 creation form does not exist. |
| Integration | **Critical** | AC-02 (authenticated user can create an upcoming meeting via `/upcoming/new`) and AC-03 (all optional fields) cannot be evaluated end-to-end because no BRD-05 form exists. |
| E2E | **Critical** | `evals/e2e/brd-05-manual-upcoming-meeting-creation.md` scenarios AC-02/AC-03 are blocked — no target UI to exercise. |

**Risk:** Without a BRD-05-specific form, FR-15 form preservation, FR-25 participant ergonomics, and FR-24 duplicate awareness cannot be implemented or tested against BRD-05.

---

## 2. WCAG 2.1 AA — Spec Does Not Define Accessibility Targets

**Spec requirement:**
> NFR Accessibility: Dashboard entry, list, calendar-style view, detail, form validation, edit, and cancel flows meet WCAG 2.1 AA expectations. (line 103)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | **Medium** | The spec states a WCAG 2.1 AA goal but provides no concrete criteria: no color contrast ratios, no keyboard navigation path, no ARIA role/state definitions, no screen reader UX description for any flow. |
| Input.svelte | **Positive** | Correctly uses `aria-invalid`, `aria-describedby` pointing to error span, and `role="alert"` on error messages. Focus ring on `:focus-visible` present. `autocomplete="off"` on all inputs. |
| ManualMeetingImportForm | **Positive** | `aria-live="polite"` on character count. `aria-expanded` on advanced toggle. `aria-label` on remove participant button. `<fieldset>/<legend>` for participant section and meeting content section. |
| Color contrast | **Unknown** | No CSS variable audit performed. Spec does not define minimum contrast targets. |
| Keyboard nav | **Unknown** | No documented tab order for the participant row add/remove pattern. `addParticipant` and `removeParticipant` are keyboard-accessible via button click but no explicit `kbd` guidance. |
| Focus management on error | **Missing** | On server validation error, focus is not programmatically moved to the first error field. User must discover errors visually. WCAG 2.1 AA 3.3.1 (Error Identification) recommends errors be associated with form controls and surfaced programmatically. |
| Screen reader — duplicate warning | **Unknown** | FR-24 duplicate awareness UI (warning on same title+scheduled_start) is unimplemented. No ARIA live region exists for this scenario. |
| Screen reader — cancelled affordance | **Unknown** | FR-23 low-prominence cancelled/history access pattern is unimplemented. No accessible disclosure mechanism specified. |

**Recommendation:** The spec should enumerate specific WCAG 2.1 AA success criteria per flow (e.g., 1.4.3 Contrast, 2.1.1 Keyboard, 3.3.1 Error Identification, 4.1.2 Name/Role/Value). The component's existing ARIA groundwork is solid but focus management on error and screen reader coverage for duplicate/cancelled flows need explicit spec and implementation.

---

## 3. Form Preservation on Validation Error (FR-15)

**Spec requirement:**
> FR-15: Validation is server-authoritative. Client-side validation may provide helper feedback, but the server remains the final gate. Submitted form values are preserved and echoed back after validation failures. (line 64)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD-02 form (ManualMeetingImportForm) | **Positive** | Uses Svelte `$state` for all form fields (`title`, `completedAt`, `participants`, etc.). On server error (status 400 with `data.errors`), the handler populates `serverErrors` without touching reactive state — form values are preserved automatically since `$state` is not reset. (lines 29–30, 107–114) |
| BRD-05 form | **Critical** | Does not exist (see Finding 1). FR-15 compliance cannot be assessed for BRD-05. |
| Server echo contract | **Unknown** | The spec does not define the JSON shape of server validation error responses. The unit eval (line 122–125) assumes an echo contract but no backend handler implementation is present in the codebase to verify. |

**Positive note for BRD-02:** The pattern of clearing `serverErrors` on new submission while leaving form `$state` untouched is correct and meets FR-15 intent.

---

## 4. Participant Input Ergonomics (FR-25)

**Spec requirement:**
> FR-25: The creation/edit form should support adding, removing, and editing multiple participant rows without losing entered values during validation errors. (line 77)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD-02 form (ManualMeetingImportForm) | **Positive** | Participant rows are managed via `participants = $state<Participant[]>` array. `addParticipant()` appends a new `defaultParticipant()`. `removeParticipant(index)` filters by index. Both operations preserve sibling row state because Svelte reactivity updates only what changes. The remove button is disabled when only 1 participant remains (line 59). |
| BRD-05 form | **Critical** | Does not exist (see Finding 1). FR-25 compliance cannot be assessed. |
| Edit flow ergonomics | **Unknown** | No edit route (`/upcoming/{id}/edit`) or form exists in the codebase to evaluate participant in-edit scenarios. |
| 50-participant limit UX | **Unknown** | FR-26 / unit eval AC specifies a 50-participant cap. No UI affordance warns users as they approach the limit (no counter or disabling of add button at 49). |

---

## 5. Duplicate Awareness UI (FR-24)

**Spec requirement:**
> FR-24: The UI should warn when a user creates another upcoming meeting with the same normalized title and scheduled start within a short time window, while still allowing creation if the user proceeds. (line 76)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD-02 form (ManualMeetingImportForm) | **Positive** | Handles duplicate response (HTTP 409) by redirecting to original meeting (lines 115–118). This implements BRD-02 duplicate awareness but not FR-24's warning-before-proceed pattern. |
| BRD-05 form | **Critical** | Does not exist (see Finding 1). |
| FR-24 warning pattern | **Missing** | No BRD-05 UI shows an inline warning (e.g., "You already have an upcoming meeting titled 'Q3 Planning' at 2pm on June 3 — proceed anyway?"). The spec's "warn when...within a short time window" suggests a soft warning, not a blocking error. This pattern is not implemented in any existing component. |
| ARIA for duplicate warning | **Missing** | No `role="alert"` or `aria-live="polite"` region exists for a pre-submission duplicate warning in any meeting creation form. |

---

## 6. Validation Error Class Messaging (FR-26)

**Spec requirement:**
> FR-26: Validation failures should be categorized into controlled classes such as `missing_title`, `invalid_scheduled_start`, `participant_missing_identity`, `too_many_participants`, `invalid_email`, and `unknown`. (line 78)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Backend metrics | **Positive** | `cp_upcoming_meeting_validation_failed_total` metric is defined with label `validation_class` using the exact enum values from FR-26 (line 115). Log event `upcoming_meeting.create_failed` includes `validation_class` (line 134). |
| Frontend error display | **Medium** | `ManualMeetingImportForm` renders server errors as `serverErrors[field]` with `role="alert"` per-field (e.g., line 216: `error={serverErrors[`participants[${i}].displayName`]}`). However, errors are displayed as free-text strings returned by the server — they are not annotated with their `validation_class` in the UI. A user sees "Display name is required" but not `missing_title` or `participant_missing_identity`. The FR-26 class is in the protocol but not surfaced in UX. |
| Error class → user message mapping | **Missing** | No `validationClassToMessage` mapping exists in the frontend. The component relies entirely on server-provided text. If the server returns a raw string for `unknown`, the UI has no fallback classification to display a user-friendly message. |
| Privacy-safe diagnostics | **Positive** | FR-20 states validation messages must not expose private content. The metric label uses low-cardinality enums only, not raw user input (line 128). |

---

## 7. Cancelled / History Affordance (FR-23)

**Spec requirement:**
> FR-23: Users should have a low-prominence way to find cancelled upcoming meetings for audit/debugging without cluttering default preparation views. (line 75)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | **Medium** | FR-23 describes the affordance as "low-prominence" but does not define what surface (filter toggle, separate tab, archive link, admin-only). The term is ambiguous. |
| UI implementation | **Critical** | No cancelled meeting access UI exists in any route or component reviewed. No filter on `/upcoming` list, no "Show cancelled" toggle, no history section. AC-07 (list excludes cancelled) is specified but AC-23 (accessing cancelled) has no acceptance criteria. |
| Screen reader for cancelled | **Unknown** | With no visible affordance, there is no ARIA pattern to evaluate. |

---

## Summary

| # | Finding | Severity |
|---|---------|----------|
| 1 | BRD-05 creation form does not exist; `/upcoming/*` routes are entirely absent from the codebase | **Critical** |
| 2 | WCAG 2.1 AA is stated as a goal but no concrete per-criterion spec exists; focus management on error is missing | **High** |
| 3 | Form preservation for BRD-05 cannot be assessed (form missing); BRD-02 form uses correct pattern | **High** |
| 4 | Participant ergonomics for BRD-05 cannot be assessed (form missing) | **High** |
| 5 | FR-24 duplicate warning ("proceed anyway?" pattern) is not implemented in any existing form | **High** |
| 6 | FR-26 validation classes are in backend metrics but not surfaced to users via UI messaging | **Medium** |
| 7 | FR-23 cancelled/history affordance has no UI implementation | **Medium** |

**Root cause:** The BRD-05 implementation has not started. The frontend codebase contains only the BRD-02 ManualMeetingImportForm at `meetings/new`. There are no `/upcoming/*` routes, no upcoming meeting list, calendar, detail, or edit views. All BRD-05 UX findings are blocked pending implementation.