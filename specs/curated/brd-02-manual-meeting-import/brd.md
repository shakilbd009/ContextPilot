# BRD-02 Manual Meeting Import — Curated

---
status: Curated
brd_id: brd-02
title: Manual Meeting Import
author: pm
created: 2026-05-20
curated: 2026-05-20
graduated: 2026-05-22
reviewers: refiner (t_10b6141a), architect (t_a2939a7d)
adrs: ADR-002 (validation), ADR-003 (data model), ADR-004 (form preservation)
parent: brd-01-app-shell
blocks: brd-03-meeting-memory-processing

---

## Overview

Manual Meeting Import lets authenticated users create completed meeting records without connecting a calendar, conferencing provider, or email account. Users supply a meeting title, completed date/time, structured participants, and either transcript text or notes so that later features can process the meeting into durable memory.

This feature is the foundational data entry path for ContextPilot: provider integrations, future meeting creation, and AI memory processing are intentionally out of scope for Phase 1. The meeting memory loop must be proven with explicit, user-controlled input before automated imports exist.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | Authenticated user | Manually create a completed meeting record with title, date/time, structured participants, and transcript or notes | I can start using ContextPilot before connecting any external provider |
| US-2 | Authenticated user | Paste or type meeting transcript text | The system has enough raw context for future memory processing |
| US-3 | Authenticated user | Enter notes when no transcript is available | I can preserve decisions and context from informal or offline meetings |
| US-4 | Authenticated user | Add participants with a required display name and optional advanced details | Future memory and briefing features can reason about people without requiring a full contacts system |
| US-5 | Authenticated user | Review validation errors before saving | I can correct missing or malformed fields without losing entered content |
| US-6 | Authenticated user | Land on the saved meeting detail page after successful save | I can confirm the import succeeded and continue to downstream workflows |
| US-7 | Authenticated user | Understand that future meeting creation is handled outside manual import | I do not confuse completed meeting capture with pre-call preparation |
| US-8 | Product/operator | Observe import attempts, failures, and saved record counts | I can detect user friction and data-quality issues early |

---

## Functional Requirements

### Must Have

- Provide a manual completed-meeting creation entry point at `/meetings/new` behind the app shell and authenticated route behavior defined by BRD-01.
- Require a meeting title before a meeting can be saved. Titles are limited to 500 characters.
- Require a completed meeting date/time before a meeting can be saved.
- Require at least one structured meeting participant before a meeting can be saved.
- Require each participant to have a display name. Display names are stored as-entered; leading and trailing whitespace is trimmed. Empty-only inputs after trimming are rejected as invalid. Name inversion, capitalization normalization, and deduplication are out of scope for BRD-02 and deferred to BRD-03 participant handling.
- Allow optional participant fields for email, organization/company, and role/title behind a collapsible "Advanced participant details" section. This section may be per-participant or shared at the bottom of the participant list, consistent with the disclosure pattern used elsewhere in BRD-01.
- Store participants as structured meeting participant records attached to the meeting via a foreign key relationship.
- Require at least one content source: transcript text or notes text.
- Enforce a combined maximum of 50,000 characters across transcript text plus notes text. The 50,000-character limit is a storage and performance sanity check derived from the product storage model; it is enforced at the application write layer, not at the database layer.
- Preserve user-entered form content when validation fails. The server returns field-level validation errors alongside submitted values in the error response body; SvelteKit's `use:enhance` intercepts the response and re-renders the form with fields pre-populated. As a fallback for catastrophic failures (server crash mid-submit, navigation away during an in-flight request), sessionStorage may store a draft flag `{ hasDraft: true, savedAt: timestamp }` with no raw form content.
- Store the saved meeting as a first-class meeting record that can be listed and opened through the meeting list and meeting detail routes defined by BRD-01.
- Clearly distinguish transcript input from notes input so downstream memory processing can reason about source quality.
- Redirect the user to the saved meeting detail page after a successful save. No query params or special state are added to the redirect URL.
- Gate all manual import server behavior behind `FF_ENABLE_MANUAL_MEETING_IMPORT` with default `false`.
- Gate all browser-visible manual import UI behind `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT` with default `false`.
- When the feature flag is disabled, hide the manual import entry point from navigation and prevent direct access to `/meetings/new`.
- Emit observability metrics and log events for started, validation failed, completed, and failed import attempts.
- **Idempotency token**: The form includes a hidden `idempotencyToken` field (UUID generated on page load). The server checks this token before saving; if the token has already been consumed, the server returns 409 Conflict with the original meeting ID in the response body. Tokens expire after 24 hours.
- Keep future/upcoming meeting creation out of scope for this BRD; upcoming meeting creation belongs to BRD-04 Pre-Call Briefing.
- Keep global contacts, participant deduplication, CRM-style relationship history, and identity resolution out of scope for this BRD.
- Keep AI memory processing, summary generation, decision extraction, action item extraction, and pre-call briefing generation out of scope for this BRD.

### Should Have

- Allow users to save a meeting when both transcript and notes are present, preserving both inputs separately.
- Support multiline transcript and notes content without destructive whitespace normalization.
- Display an explicit success indication during redirect or on the saved meeting detail page so users understand the meeting was saved.
- Display a clear zero-state explanation when manual import is disabled or unavailable.
- Display a live character count for transcript plus notes. Show a warning when the combined count reaches 45,000 characters (within 5,000 of the limit); block submission at 50,001 characters.
- Capture content-source metadata: transcript only, notes only, or both.
- Prevent duplicate submission from repeated save clicks during an in-flight save (button disabled client-side; server-side enforcement via idempotency token).
- Return field-level validation messages specific enough for E2E evals to assert.

### Could Have

- Offer participant entry helpers such as comma-separated parsing or add/remove chips.
- Offer a draft warning before navigating away with unsaved content.
- Provide a lightweight paste-cleanup affordance for transcript text copied from conferencing tools.
- Allow optional participant fields to expand/collapse per participant or as a single advanced section.

---

## API Contract

`POST /meetings` — gated by `FF_ENABLE_MANUAL_MEETING_IMPORT`

**Request body (application/json):**
```json
{
  "title": "string (required, max 500 chars)",
  "completedAt": "ISO 8601 timestamp (required)",
  "participants": [
    {
      "displayName": "string (required)",
      "email": "string (optional)",
      "organization": "string (optional)",
      "role": "string (optional)"
    }
  ],
  "transcript": "string (optional; notes must be present if absent)",
  "notes": "string (optional; transcript must be present if absent)",
  "idempotencyToken": "string (UUID, required)"
}
```

**Responses:**

| Status | Condition | Body |
|--------|-----------|------|
| 201 Created | Meeting saved | `{ "id": "<meetingId>", "redirect": "/meetings/<id>" }` |
| 400 Bad Request | Validation failure | `{ "errors": [{ "field": "<name>", "message": "<specific>" }], "values": { ... } }` |
| 401 Unauthorized | No valid session | — |
| 403 Forbidden | Feature flag disabled | — |
| 409 Conflict | Idempotency token already used | `{ "error": "duplicate", "meetingId": "<id>", "originalOutcome": "completed" }` |
| 500 Internal Server Error | Unexpected failure | `{ "error": "<safe class>", "correlationId": "<uuid>" }` |

**OpenAPI prerequisite (Phase 1 blocker):** `contracts/openapi.yaml` currently defines a `MeetingSummary` schema with `start_time`, `end_time`, `attendees[]` that is incompatible with the BRD-02 data model (`completedAt`, `transcript`, `notes`, `contentSource`). A new `Meeting` schema must be added to `openapi.yaml` before Phase 1 implementation begins.

---

## Data Model

### Meeting record

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key; generated server-side via `gen_random_uuid()` |
| `title` | TEXT | Yes | Max 500 characters; enforced via CHECK constraint |
| `completedAt` | TIMESTAMPTZ | Yes | Stored in UTC |
| `transcript` | TEXT | No | Null when notes-only |
| `notes` | TEXT | No | Null when transcript-only |
| `contentSource` | TEXT | Yes | Derived at save time: `transcript` if transcript present and notes empty, `notes` if notes present and transcript empty, `both` if both present |
| `createdAt` | TIMESTAMPTZ | Yes | Auto-set server-side |
| `createdBy` | UUID | Yes | Derived from authenticated session |
| `displayOrder` | INTEGER | Yes | Defaults to 0; ordering column for stable participant display across BRD-04 and later features |

### Meeting participant record

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `meetingId` | UUID | Yes | Foreign key to `meetings(id)` with ON DELETE CASCADE |
| `displayName` | TEXT | Yes | Leading/trailing whitespace trimmed; empty-after-trim rejected |
| `email` | TEXT | No | — |
| `organization` | TEXT | No | — |
| `role` | TEXT | No | — |

Combined transcript-plus-notes character limit (50,000) is enforced at the application write layer in the Go handler before any database INSERT. PostgreSQL TEXT columns have no practical size limit; a CHECK constraint is not used because it produces an untyped database error rather than a clean 400 response.

The `contentSource` field is set by the application on write, not accepted as user input, to prevent invalid state where `contentSource` says "transcript only" but `notes` is populated.

Transaction isolation level is READ COMMITTED (PostgreSQL default). Changing this requires ADR approval.

---

## Non-Functional Requirements

| Requirement | Target | Notes |
|-------------|--------|-------|
| Form interactivity | Field updates, participant row updates, advanced section expand/collapse, validation feedback, character-limit feedback, and save button state update respond in under 100ms client-side for content up to 50,000 combined characters | Client-side responsiveness only; independent of network round-trip |
| Save latency | Successful save returns within 1 second (server round-trip) for content up to the 50,000-character product limit | Measured client-visible: user clicks Save to receiving the redirect or error |
| Availability | Manual import path follows the app shell availability target from BRD-01 | BRD-01 defines the concrete SLA; `/ready` probe validates PostgreSQL connectivity |
| Accessibility | Form controls, errors, and success states meet WCAG 2.1 AA: explicit `for`/`id` association on labels, error announcements via ARIA live region, keyboard navigation (Tab through fields, Enter to submit, Escape to cancel), focus management on validation failure (focus moves to first invalid field), focus management on successful save (focus moves to meeting detail page heading), visible focus indicators | Specific criteria must be verified against BRD-01 component inventory |
| Data retention | Imported meeting content follows the project retention policy once defined (BRD-06); until then, records are retained as user-owned application data | Explicit TBD; acceptable for Phase 1; deferred to BRD-06 |
| Privacy | Transcript text, notes text, participant names, participant emails, participant organizations, participant roles, and meeting titles are treated as sensitive meeting data and must not be logged in raw form | See Observability section for redaction schema |
| Browser support | Same browser support baseline as BRD-01 App Shell | — |

---

## Observability

### Metrics

| Metric | Type | Labels | Note |
|--------|------|--------|------|
| `cp_manual_meeting_import_started_total` | Counter | `result`, `content_source`, `route` | Incremented when a save attempt passes initial client intent |
| `cp_manual_meeting_import_validation_failed_total` | Counter | `result`, `field`, `content_source` | Incremented per validation failure; `field` is the failing field name |
| `cp_manual_meeting_import_completed_total` | Counter | `result`, `content_source` | Incremented on successful meeting record creation |
| `cp_manual_meeting_import_failed_total` | Counter | `result`, `flag` | Incremented on server-side failures before a meeting is saved |
| `cp_manual_meeting_import_save_duration_ms` | Histogram | — | Buckets: 50, 100, 250, 500, 1000, 2500, 5000, 10000 |
| `cp_manual_meeting_import_content_size_chars` | Histogram | — | Buckets: 100, 1000, 5000, 10000, 25000, 40000, 50000, 60000, 100000, Inf |
| `cp_manual_meeting_import_flag_eval_duration_ms` | Histogram | — | Buckets: 1, 5, 10, 25, 50, 100 |
| `cp_manual_meeting_import_flag_misconfiguration_total` | Counter | `flag` | Incremented when server=false and browser=true |

### Log events

| Event | When emitted | Fields |
|-------|-------------|--------|
| `meeting.import.started` | Save attempt begins | `correlationId`, `route`, `flag`, `contentSource` |
| `meeting.import.validation_failed` | User-correctable validation blocks save | `correlationId`, `field` |
| `meeting.import.completed` | Meeting record is saved | `correlationId`, `meetingId`, `contentSource` |
| `meeting.import.failed` | Save fails unexpectedly | `correlationId`, `errorClass`, `retryable` |
| `meeting.import.flag_misconfiguration` | Flag misconfiguration detected on request | `correlationId`, `flag`, `serverEnv`, `browserEnv` |
| `meeting.import.duplicate_submission` | Idempotency token already consumed | `correlationId`, `idempotencyToken`, `originalMeetingId`, `originalOutcome` |

### Redaction schema

Allowed metric labels and log fields: `result`, `field`, `content_source`, `route`, `flag`, `errorClass`, `retryable`, `correlationId`.

Forbidden: meeting titles, participant names, email addresses, transcript text, notes text, raw user IDs, raw meeting IDs. Label cardinality must not exceed 20 unique values per label name across the fleet.

Raw transcript text, raw notes text, participant values, and meeting titles must never be written to logs.

### Health/readiness endpoints

- `GET /ready` — returns 200 only when the app can evaluate the manual import flag and the meeting persistence dependency (PostgreSQL) is reachable.
- `GET /live` — returns 200 when the process is alive.

---

## Feature Flag

- **Flag name:** `ff_enable_manual_meeting_import`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_MANUAL_MEETING_IMPORT`
- **Browser env:** `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT`

Dual namespace registration is confirmed in `specs/feature-flags.md` (Phase 1, Planned).

| Server env | Browser env | Expected behavior |
|------------|-------------|-------------------|
| `false` | `false` | Manual import navigation is hidden, direct access is blocked |
| `true` | `true` | Authenticated users can access `/meetings/new` and create records |
| `true` | `false` | Server accepts authorized requests; browser UI hidden; allowed for server-side testing |
| `false` | `true` | Server denies creation; misconfiguration state emits metric and log event |

The server flag is authoritative for all data mutations.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | `/meetings/new` is not reachable when `FF_ENABLE_MANUAL_MEETING_IMPORT` is `false` | Feature flag E2E scenario verifies hidden navigation and blocked direct route |
| AC-2 | Authenticated users can open `/meetings/new` when both flags are enabled | Playwright authenticated route test asserts HTTP 200 and form renders |
| AC-3 | Save is blocked when title is missing; error message references the title field | Form validation E2E test submits without title, asserts 400 response with `field: "title"` |
| AC-4 | Save is blocked when completed meeting date/time is missing; error message references the date/time field | Form validation E2E test submits without date/time, asserts 400 with `field: "completedAt"` |
| AC-5 | Save is blocked when no participant is present | Form validation E2E test submits with empty participants array, asserts 400 with `field: "participants"` |
| AC-6 | Save is blocked when any participant is missing display name | Form validation E2E test submits a participant with empty displayName, asserts 400 with `field: "participants[0].displayName"` |
| AC-7 | E2E test expands advanced participant fields, submits with optional email, organization, and role populated, asserts save succeeds and saved meeting record contains those fields | E2E test verifies both populated and empty optional fields paths |
| AC-8 | Save is blocked when both transcript and notes are empty | Form validation E2E test submits empty transcript and empty notes, asserts 400 with `field: "content"` |
| AC-9 | Save is blocked when transcript plus notes exceed 50,000 characters | Integration test submits 50,001-character combined payload, asserts 400 with `field: "content"` |
| AC-10 | Save succeeds with title, completed date/time, at least one participant display name, and transcript text; user lands on saved meeting detail page | E2E happy-path test clicks Save, waits for redirect to `/meetings/<id>`, asserts meeting title visible |
| AC-11 | Save succeeds with title, completed date/time, at least one participant display name, and notes text when transcript is absent | E2E notes-only test asserts save succeeds, meeting list shows record, detail page shows notes content |
| AC-12 | If transcript and notes are both supplied within the 50,000-character limit, both are preserved separately | Integration or API-level test submits both fields, retrieves saved meeting, asserts both are non-null and match submitted values |
| AC-13 | Structured participants are preserved as separate meeting participant records | Integration or API-level test saves a meeting with multiple participants, retrieves meeting, asserts participant records have correct values |
| AC-14 | Successful save redirects directly to the saved meeting detail page; no query params or special state | E2E happy-path test asserts final URL matches `/meetings/<id>` with no query string |
| AC-15 | Validation failure preserves all entered form values on re-render | E2E test fills form, triggers validation failure, asserts all previously entered values remain |
| AC-16 | Double-clicking save does not create duplicate meeting records | E2E or integration test clicks Save twice with same idempotencyToken, asserts exactly one meeting record is persisted and second submission returns 409 |
| AC-17 | Observability emits the correct event names and no forbidden values appear in metric labels or log fields | Observability eval runs full import flow, retrieves emitted metrics and log events, asserts event names match and no sensitive content appears |
| AC-18 | Manual import form meets WCAG 2.1 AA | Accessibility eval verifies each criterion using axe-core or equivalent |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Users paste sensitive transcript content and expect privacy protections | High | High | Treat content as sensitive data; never log raw content; surface clear ownership/retention expectations |
| Oversized transcripts degrade save performance or storage reliability | Medium | Medium | Enforce 50,000-character combined limit at application write layer; show user-facing limit and warning at 45,000 characters |
| Manual import creates low-quality or incomplete data for BRD-03 processing | Medium | Medium | Require title, completed date/time, at least one participant display name, and at least one content source; preserve content-source metadata for downstream quality handling |
| Feature flag namespace drift exposes UI without server support | Medium | High | Server flag is authoritative; misconfiguration state is emitted as a metric and log event; both flags must be registered together |
| Duplicate submissions create duplicate meeting records | Medium | Medium | Idempotency token (UUID per form load, server-side dedupe with 24h TTL) plus client-side button disable |
| Participants may include personally identifiable information | High | Medium | Avoid participant values in logs and metrics; treat participant data as sensitive application data |
| Provider import assumptions leak into manual flow | Low | Medium | Explicitly keep calendar/provider integrations out of scope; manual import is documented as the foundation path |
| Manual import scope expands into contact management | Medium | Medium | Store structured meeting participants only; keep global contacts, deduplication, relationship history, and identity resolution out of scope |
| Users confuse completed meeting import with future meeting preparation | Medium | Medium | Use completed-meeting copy in the manual import flow; keep upcoming meeting creation in BRD-04 Pre-Call Briefing |

---

## Open Questions

The following items were raised during review and are resolved or deferred as noted:

| Question | Disposition | Resolution |
|----------|-------------|------------|
| transcript/notes size limit derivation | Resolved | 50,000 characters; derived from storage/performance sanity check; enforced in FR |
| Participant normalization (name inversion, capitalization) | Deferred to BRD-03 | Display name stored as-entered with leading/trailing whitespace trimmed; empty-after-trim rejected; normalization out of scope for BRD-02 |
| Post-save navigation UX detail | Resolved | Redirect to `/meetings/<id>` with no query params; back button returns to `/meetings` |
| OQ-1: Idempotency token semantics | Resolved | Must Have — hidden UUID form field generated on page load; server checks token before saving; duplicate token returns 409 Conflict with original meeting ID and outcome; TTL 24 hours |
| OQ-2: Meeting ID URL encoding | Resolved | UUID format confirmed: `/meetings/{meetingId: uuid}` in route paths |
| OQ-3: Client-side validation implementation | Deferred to code review | ADR-002 specifies dual-layer validation; shared validation constants package is a Phase 1 implementation detail to be resolved in code review |
| OQ-4: Character count display threshold | Resolved | Warning at 45,000 characters (within 5,000 of limit); hard block at 50,001; moved from Could Have to Should Have |
| OQ-5: Transaction isolation level | Resolved | READ COMMITTED (PostgreSQL default); changing requires ADR approval |
| OQ-6: Participant ordering | Resolved | `displayOrder INTEGER NOT NULL DEFAULT 0` added to `meeting_participants` schema |
| OQ-7: title max length | Resolved | 500 characters; enforced via CHECK constraint |

---

## Relations

- **Parent feature:** BRD-01 App Shell
- **Blocked by:** BRD-01 App Shell (authenticated route shell, `/meetings`, `/meetings/new`, `/meetings/[id]`), OpenAPI contract update (Meeting schema addition)
- **Blocks:** BRD-03 Meeting Memory Processing
- **Related specs:** `specs/curated/brd-01-app-shell.md`, `specs/feature-flags.md`, `docs/adr/0002-manual-meeting-import-validation-strategy.md`, `docs/adr/0003-manual-meeting-import-data-model.md`, `docs/adr/0004-manual-meeting-import-form-content-preservation.md`

---

## Pre-Phase 1 Implementation Prerequisites

The following must be resolved before Phase 1 implementation begins:

1. **OpenAPI contract update (critical blocker):** `contracts/openapi.yaml` must be updated with a new `Meeting` schema for the `POST /meetings` write path. The existing `MeetingSummary` schema is incompatible with BRD-02's data model and must not be used for BRD-02 endpoints.
2. **Idempotency token mechanism:** The idempotency token must be implemented as specified (UUID per form load, server-side dedupe with 24h TTL, 409 Conflict response shape). This is a Must Have before AC-16 can be verified.
3. **Flag misconfiguration observability:** `cp_manual_meeting_import_flag_misconfiguration_total` metric and `meeting.import.flag_misconfiguration` log event must be implemented before the feature is enabled in any environment.