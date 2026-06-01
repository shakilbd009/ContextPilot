# BRD-02 Manual Meeting Import

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-02` |
| Title | Manual Meeting Import |
| Author | pm |
| Created | 2026-05-20 |
| Status | Draft |
| Priority | P1 |
| Phase | Phase 1 |

---

## Overview

Manual Meeting Import lets authenticated users create completed meeting records without connecting a calendar, conferencing provider, or email account. It is the foundational data entry path for ContextPilot: users can supply a meeting title, completed date/time, structured meeting participants, and either transcript text or structured notes so later features can process the meeting into durable memory.

This feature matters because provider integrations, future meeting creation, and AI memory processing are intentionally out of scope for the first domain capture workflow. ContextPilot must still prove the meeting memory loop with explicit, user-controlled input before automated imports exist.

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
- Require a meeting title before a meeting can be saved.
- Require a completed meeting date/time before a meeting can be saved.
- Require at least one structured meeting participant before a meeting can be saved.
- Require each participant to have a display name.
- Allow optional participant fields for email, organization/company, and role/title behind an advanced affordance so the default form remains lightweight.
- Store participants as structured meeting participant records attached to the meeting.
- Require at least one content source: transcript text or notes text.
- Enforce a combined maximum of 50,000 characters across transcript text plus notes text.
- Preserve user-entered form content when validation fails.
- Store the saved meeting as a first-class meeting record that can be listed and opened through the meeting list and meeting detail routes defined by BRD-01.
- Clearly distinguish transcript input from notes input so downstream memory processing can reason about source quality.
- Redirect the user to the saved meeting detail page after a successful save.
- Gate all manual import server behavior behind `FF_ENABLE_MANUAL_MEETING_IMPORT` with default `false`.
- Gate all browser-visible manual import UI behind `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT` with default `false`.
- When the feature flag is disabled, hide the manual import entry point from navigation and prevent direct access to `/meetings/new`.
- Emit observability metrics and log events for started, validation failed, completed, and failed import attempts.
- Keep future/upcoming meeting creation out of scope for this BRD; upcoming meeting creation belongs to BRD-04 Pre-Call Briefing.
- Keep global contacts, participant deduplication, CRM-style relationship history, and identity resolution out of scope for this BRD.
- Keep AI memory processing, summary generation, decision extraction, action item extraction, and pre-call briefing generation out of scope for this BRD.

### Should Have

- Allow users to save a meeting when both transcript and notes are present, preserving both inputs separately.
- Support multiline transcript and notes content without destructive whitespace normalization.
- Display an explicit success indication during redirect or on the saved meeting detail page so users understand the meeting was saved.
- Display a clear zero-state explanation when manual import is disabled or unavailable.
- Explain the 50,000 character combined transcript-plus-notes limit before or during validation.
- Capture content-source metadata: transcript only, notes only, or both.
- Prevent duplicate submission from repeated save clicks during an in-flight save.
- Return field-level validation messages that are specific enough for E2E evals to assert.

### Could Have

- Offer participant entry helpers such as comma-separated parsing or add/remove chips.
- Offer a draft warning before navigating away with unsaved content.
- Provide a lightweight paste-cleanup affordance for transcript text copied from conferencing tools.
- Show an estimated character count or processing readiness hint for downstream memory processing.
- Allow optional participant fields to expand/collapse per participant or as a single advanced section.

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Form interactivity | Field input, participant row updates, advanced participant affordance expansion, validation feedback, character-limit feedback, and save button state update without noticeable delay for content up to 50,000 combined transcript-plus-notes characters |
| Save latency | Successful save returns within 1 second for transcript plus notes content up to the 50,000 character product limit |
| Availability | Manual import path follows the app shell availability target from BRD-01 |
| Accessibility | Form controls, errors, and success states meet WCAG 2.1 AA expectations inherited from BRD-01 |
| Data retention | Imported meeting content follows the project retention policy once defined; until then, records are retained as user-owned application data |
| Privacy | Transcript text, notes text, participant names, participant emails, participant organizations, participant roles, and meeting titles are treated as sensitive meeting data and must not be logged in raw form |
| Browser support | Same browser support baseline as BRD-01 App Shell |

---

## Observability

### Metrics to emit

- `cp_manual_meeting_import_started_total` — counter for manual import save attempts that pass initial client intent.
- `cp_manual_meeting_import_validation_failed_total` — counter for validation failures by field category, including title, date/time, participants, participant display name, content source, and content size limit.
- `cp_manual_meeting_import_completed_total` — counter for successful meeting record creation.
- `cp_manual_meeting_import_failed_total` — counter for server-side failures before a meeting is saved.
- `cp_manual_meeting_import_save_duration_ms` — histogram for save request duration.
- `cp_manual_meeting_import_content_size_chars` — histogram for submitted transcript plus notes character count, bucketed without storing raw content and with buckets that identify submissions near or above the 50,000 character limit.
- `cp_manual_meeting_import_flag_eval_duration_ms` — histogram for server-side feature flag evaluation latency.

Required labels should be limited to non-sensitive values such as `result`, `field`, `content_source`, `route`, and `flag`. Labels must not include meeting titles, participant names, email addresses, transcript text, notes text, or raw identifiers.

### Log events

- `meeting.import.started` — emitted when a save attempt begins; include correlation ID, route, flag state, and content-source category only.
- `meeting.import.validation_failed` — emitted when user-correctable validation blocks save; include correlation ID and field category only.
- `meeting.import.completed` — emitted after a meeting record is saved; include correlation ID, meeting record identifier, and content-source category.
- `meeting.import.failed` — emitted when save fails unexpectedly; include correlation ID, safe error class, and retryability.

Raw transcript text, raw notes text, participant values, and meeting titles must not be written to logs.

### Health/readiness endpoints

- `GET /ready` — returns 200 only when the app can evaluate the manual import flag and the meeting persistence dependency is reachable.
- `GET /live` — returns 200 when the process is alive.

---

## Feature Flag

- **Flag name:** `ff_enable_manual_meeting_import`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_MANUAL_MEETING_IMPORT`
- **Browser env:** `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT`

Required behavior:

| Server env | Browser env | Expected behavior |
|------------|-------------|-------------------|
| `false` | `false` | Manual import navigation is hidden, direct access is blocked, and no meeting can be created through this path |
| `true` | `true` | Authenticated users can access `/meetings/new` and create completed manual meeting records |
| `true` | `false` | Server may accept authorized requests, but browser UI remains hidden; this state is allowed only for controlled server-side testing |
| `false` | `true` | Server denies creation even if browser UI is visible; this state should be treated as a misconfiguration and surfaced in logs/metrics |

The server flag is authoritative for data mutation. Browser behavior must not bypass server-side gating.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | `/meetings/new` is not reachable when `FF_ENABLE_MANUAL_MEETING_IMPORT` is `false` | Feature flag E2E scenario verifies hidden navigation and blocked direct route |
| AC-2 | Authenticated users can open `/meetings/new` when both manual import flags are enabled | Playwright authenticated route test |
| AC-3 | Save is blocked when title is missing | Form validation E2E test with field-level assertion |
| AC-4 | Save is blocked when completed meeting date/time is missing | Form validation E2E test with field-level assertion |
| AC-5 | Save is blocked when no participant is present | Form validation E2E test with participant-level assertion |
| AC-6 | Save is blocked when any participant is missing display name | Form validation E2E test with participant display-name assertion |
| AC-7 | Optional participant email, organization/company, and role/title are available behind an advanced affordance and are not required for save | E2E test expands advanced participant fields, saves with and without optional values, and verifies behavior |
| AC-8 | Save is blocked when both transcript and notes are empty | Form validation E2E test with content-source assertion |
| AC-9 | Save is blocked when transcript plus notes exceed 50,000 characters | E2E or integration validation test submits oversized content and asserts limit messaging |
| AC-10 | Save succeeds with title, completed date/time, at least one participant display name, and transcript text | E2E happy-path test verifies redirect to saved meeting detail and meeting list visibility |
| AC-11 | Save succeeds with title, completed date/time, at least one participant display name, and notes text when transcript is absent | E2E notes-only test verifies source metadata and saved record visibility |
| AC-12 | If transcript and notes are both supplied within the combined 50,000 character limit, both are preserved separately | Integration or API-level eval verifies saved record fields without content loss |
| AC-13 | Structured participants are preserved as meeting participant records attached to the saved meeting | Integration or API-level eval verifies participant display name and optional fields are stored separately from raw transcript/notes |
| AC-14 | Successful save redirects directly to the saved meeting detail page | E2E happy-path test asserts final route and visible meeting title on detail page |
| AC-15 | Validation failure preserves entered form values | E2E test mutates fields, triggers failure, and asserts values remain present |
| AC-16 | Double-clicking save does not create duplicate meeting records | E2E or integration test submits repeated clicks and asserts one persisted record |
| AC-17 | Logs and metrics emit for started, validation failed, completed, and failed outcomes without raw meeting content or participant PII | Observability eval checks emitted event names and redaction rules |
| AC-18 | Manual import route inherits BRD-01 app shell accessibility expectations | Accessibility eval verifies labels, error announcements, keyboard navigation, focus behavior, and advanced participant affordance behavior |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Users paste sensitive transcript content and expect privacy protections | High | High | Treat content as sensitive data, never log raw content, and surface clear ownership/retention expectations |
| Oversized transcripts degrade save performance or storage reliability | Medium | Medium | Enforce a 50,000 character combined transcript-plus-notes product limit, show user-facing limit messaging, and track safe content-size buckets |
| Manual import creates low-quality or incomplete data for BRD-03 processing | Medium | Medium | Require title, completed date/time, at least one participant display name, and at least one content source; preserve content-source metadata for downstream quality handling |
| Feature flag namespace drift exposes UI without server support | Medium | High | Keep server and browser flags registered together; server flag remains authoritative; emit misconfiguration logs/metrics |
| Duplicate submissions create duplicate meeting records | Medium | Medium | Disable or debounce save while in flight and verify duplicate-click behavior with eval coverage |
| Participants may include personally identifiable information | High | Medium | Avoid participant values in logs and metrics; treat participant data as sensitive application data |
| Provider import assumptions leak into manual flow | Low | Medium | Explicitly keep calendar/provider integrations out of scope and document that manual import is the foundation path |
| Manual import scope expands into contact management | Medium | Medium | Store structured meeting participants only; keep global contacts, deduplication, relationship history, and identity resolution out of scope |
| Users confuse completed meeting import with future meeting preparation | Medium | Medium | Use completed-meeting copy in the manual import flow and keep upcoming meeting creation in BRD-04 Pre-Call Briefing |

---

## Relations

- **Parent feature:** BRD-01 App Shell
- **Blocked by:** BRD-01 App Shell, including authenticated route shell, `/meetings`, `/meetings/new`, and `/meetings/[id]` route structure
- **Blocks:** BRD-03 Meeting Memory Processing
- **Related specs:** `specs/curated/brd-01-app-shell.md`, `specs/feature-flags.md`

---

## Open Questions

- None. Product decisions resolved for Phase 1: manual import is limited to completed meetings, participant records are structured with display name required and optional fields behind an advanced affordance, transcript plus notes have a combined 50,000 character limit, successful save redirects to the saved meeting detail page, and future/upcoming meeting creation belongs to BRD-04 Pre-Call Briefing.
