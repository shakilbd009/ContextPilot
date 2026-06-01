# BRD-05: Manual Upcoming Meeting Creation

> **Status:** Approved

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-05` |
| Title | Manual Upcoming Meeting Creation |
| Author | brainstormer |
| Created | 2026-05-23 |
| Status | Approved |
| Priority | P1 |
| Phase | Phase 2 |
| Feature flag | `ff_enable_upcoming_meetings` |
| Spec path | `specs/domain/brd-05-manual-upcoming-meeting-creation.md` |

---

## Overview

Manual Upcoming Meeting Creation gives authenticated users a first-party way to create and manage future meeting records that ContextPilot can use as anchors for pre-call preparation. It closes the BRD-04 dependency gap by defining the `upcoming_meetings` data model, user-facing creation workflow, matching metadata, and generation trigger contract for Pre-Call Briefing.

The MVP is matching-aware but not calendar-integrated: users can manually enter title, scheduled start, optional description/agenda, participants, and organization/client metadata, then manage those records through dashboard, list, calendar-style, detail, edit, and cancel surfaces. Provider sync, recurrence, reminders, hard delete, and participant-based access control are explicitly deferred.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | Authenticated user | To create an upcoming meeting with title, scheduled start, and optional preparation context | ContextPilot has a concrete future meeting anchor for pre-call preparation |
| US-2 | Authenticated user | To add participants and organization/client metadata to an upcoming meeting | Related prior meetings can be matched more accurately for BRD-04 briefings |
| US-3 | Authenticated user | To view upcoming meetings in dashboard, list, and calendar-style views | I can quickly see what meetings need preparation |
| US-4 | Authenticated user | To open an upcoming meeting detail page | I can inspect meeting metadata, briefing status, and available actions in one place |
| US-5 | Authenticated user | To edit an upcoming meeting before or shortly after its scheduled start | I can correct meeting details before they affect briefing quality |
| US-6 | Authenticated user | To cancel an upcoming meeting without deleting its history | I can remove it from active preparation views while preserving audit/debug context |
| US-7 | Authenticated user | Owner-only access to my upcoming meetings | Meeting preparation metadata remains private to the creator in the MVP |
| US-8 | Authenticated user | Pre-call briefing generation to queue automatically when eligible upcoming meetings are created or meaningfully edited | Preparation starts without requiring a separate manual action |

---

## Functional Requirements

### Must Have

- **FR-1: Upcoming meetings feature flag.** Manual Upcoming Meeting Creation is gated by `ff_enable_upcoming_meetings`. When disabled, upcoming meeting UI entry points are hidden and backend create/update/cancel/list/detail APIs reject requests with a safe feature-disabled response.
- **FR-2: Full upcoming meetings section.** The feature provides a dashboard entry point, `/upcoming` list view, calendar-style upcoming view, `/upcoming/new` creation flow, `/upcoming/{id}` detail view, and edit/cancel actions.
- **FR-3: Manual creation fields.** Users can create an upcoming meeting with required `title` and `scheduled_start`, plus optional `description` or agenda/preparation notes, optional meeting-level `client_or_organization`, and zero or more structured participants.
- **FR-4: Structured participant metadata.** Each manually entered participant supports `display_name`, `email`, and optional `organization`. Participant metadata supports BRD-04 matching but does not grant access to the upcoming meeting.
- **FR-5: Data model — upcoming_meetings.** The feature defines an `upcoming_meetings` table with at least: `id` UUID primary key, `title` text, `scheduled_start` timestamptz, `description` text nullable, `client_or_organization` text nullable, `status` enum/string, `created_by` UUID owner reference, `created_at` timestamptz, and `updated_at` timestamptz.
- **FR-6: Data model — upcoming_meeting_participants.** The feature defines an `upcoming_meeting_participants` table with at least: `id` UUID primary key, `upcoming_meeting_id` UUID foreign key, `display_name` text nullable, `email` text nullable, `organization` text nullable, `created_at` timestamptz, and `updated_at` timestamptz. A participant must have at least one of display name or email.
- **FR-7: Status model.** Upcoming meetings support at least `scheduled` and `cancelled` statuses. Scheduled meetings appear in active upcoming views by default. Cancelled meetings are hidden from default active views but remain available from detail/history surfaces where appropriate.
- **FR-8: Scheduled-start validation.** Creation and edits require `scheduled_start` to be no earlier than 15 minutes before the server's current time. Attempts outside that grace window fail validation with field-specific feedback.
- **FR-9: Edit window.** Users can edit title, scheduled start, description, meeting-level organization/client, and participants until 15 minutes after scheduled start. After that window, the upcoming meeting becomes read-only for BRD-05 MVP purposes.
- **FR-10: Soft cancellation.** Users can cancel an upcoming meeting until 15 minutes after scheduled start. Cancellation sets `status=cancelled`, prevents future briefing generation for that upcoming meeting, hides the record from default active upcoming views, and preserves the meeting and briefing history.
- **FR-11: Owner-only authorization.** Only the `created_by` owner can create, view, list, edit, or cancel their upcoming meetings. Participants do not receive access rights. This follows ADR-0009 owner-only semantics for MVP.
- **FR-12: List view behavior.** The `/upcoming` list shows the current user's non-cancelled upcoming meetings sorted by nearest scheduled start first, with clear empty states and access to create a new upcoming meeting.
- **FR-13: Calendar-style view behavior.** The calendar-style view provides read/navigation affordances for upcoming meetings by date/time. It does not include provider sync, external calendar import, recurrence management, reminders, or availability scheduling in BRD-05.
- **FR-14: Detail view behavior.** The detail view shows meeting metadata, participant metadata, status, edit/cancel actions when allowed, and briefing status/entry points only when BRD-04 is enabled and available.
- **FR-15: Validation and form preservation.** Validation is server-authoritative. Client-side validation may provide helper feedback, but the server remains the final gate. Submitted form values are preserved and echoed back after validation failures.
- **FR-16: Meaningful edit definition.** A meaningful edit is any change to title, scheduled start, description, meeting-level organization/client, participant email, participant display name, or participant organization.
- **FR-17: BRD-04 trigger on create.** When `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`, successful creation of a scheduled upcoming meeting queues BRD-04 pre-call briefing generation asynchronously. Creation does not wait for generation to complete.
- **FR-18: BRD-04 trigger on meaningful edit.** When both BRD-05 and BRD-04 server flags are enabled, a meaningful edit queues BRD-04 regeneration or marks the active briefing stale according to the BRD-04 generation contract. The latest completed briefing remains visible while regeneration runs.
- **FR-19: BRD-04 trigger skipped states.** If BRD-04 is disabled, generation dependencies are unavailable, the upcoming meeting is cancelled, or the record is outside the editable/eligible window, BRD-05 creation/edit still follows its own rules but briefing trigger behavior is skipped safely and observably.
- **FR-20: Privacy-safe diagnostics.** User-facing validation and failure messages are actionable but do not expose private content. Operator diagnostics use opaque or hashed identifiers only and must not include meeting titles, descriptions, participant names, emails, or organization/client values.

### Should Have

- **FR-21: Dashboard preparation entry.** The app dashboard should include a compact upcoming meetings card or section that links to the full upcoming meetings section and highlights the next scheduled meeting when available.
- **FR-22: Briefing trigger status.** Detail views should show whether a briefing is unavailable, queued, generating, ready, stale, failed, or skipped because BRD-04 is disabled, reusing BRD-04 status conventions where possible.
- **FR-23: Cancelled/history affordance.** Users should have a low-prominence way to find cancelled upcoming meetings for audit/debugging without cluttering default preparation views.
- **FR-24: Duplicate awareness.** The UI should warn when a user creates another upcoming meeting with the same normalized title and scheduled start within a short time window, while still allowing creation if the user proceeds.
- **FR-25: Participant input ergonomics.** The creation/edit form should support adding, removing, and editing multiple participant rows without losing entered values during validation errors.
- **FR-26: Low-cardinality validation classes.** Validation failures should be categorized into controlled classes such as `missing_title`, `invalid_scheduled_start`, `participant_missing_identity`, `too_many_participants`, `invalid_email`, and `unknown`.

### Could Have

- **FR-27: Provider/calendar sync.** External calendar provider import, provider event IDs, sync conflict handling, and webhook updates may be handled in a future provider connectors BRD.
- **FR-28: Recurrence.** Recurring meeting series, recurrence exceptions, and series-level matching are deferred.
- **FR-29: Reminders and delivery.** Email, Teams, Slack, browser notification, mobile push, or calendar reminder delivery is deferred.
- **FR-30: Participant-based access.** Participant access rights, organization-wide sharing, and cross-user upcoming meeting visibility are deferred beyond the owner-only MVP.
- **FR-31: Hard delete.** Hard deletion and purge behavior are deferred to BRD-06 Privacy & Retention Controls unless required by a later security review.
- **FR-32: Availability scheduling.** Finding times, RSVP state, availability overlays, and meeting invitation workflows are out of scope.

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Create/update/cancel latency | p95 < 500ms for BRD-05 server action completion, excluding asynchronous BRD-04 generation work |
| List/detail/calendar view latency | p95 < 500ms for current user's upcoming meeting data retrieval and render path, excluding client/network variability outside the app |
| Briefing trigger enqueue latency | p95 < 100ms to enqueue or safely skip the BRD-04 briefing trigger after successful create or meaningful edit |
| Availability | Upcoming meeting create/list/detail/edit/cancel follow app-baseline availability; BRD-04 generation degradation must not prevent BRD-05 meeting creation when BRD-05 itself is healthy |
| Authorization | 100% of BRD-05 read/write actions enforce owner-only access using `created_by`; participant metadata grants no access |
| Privacy | Logs and metric labels contain no raw meeting titles, descriptions, agenda text, participant names, participant emails, organization/client values, or user-entered private content |
| Scale | MVP supports at least 100 active upcoming meetings per user and up to 50 participants per upcoming meeting without degrading the p95 targets in fixture tests |
| Validation consistency | Server validation is authoritative for 100% of create/edit/cancel actions; client-side checks are advisory only |
| Accessibility | Dashboard entry, list, calendar-style view, detail, form validation, edit, and cancel flows meet WCAG 2.1 AA expectations |
| Retention | Upcoming meeting retention, purge, export, and hard delete behavior inherit final policy from BRD-06 Privacy & Retention Controls; BRD-05 defines only soft cancellation |

---

## Observability

### Metrics to emit

- `cp_upcoming_meeting_create_requested_total` — counter labeled by `source` (`dashboard`, `list`, `calendar`, `direct`) and `has_participants` (`true`, `false`).
- `cp_upcoming_meeting_created_total` — counter labeled by `participant_count_bucket` (`0`, `1`, `2_5`, `6_10`, `11_50`) and `has_client_or_org` (`true`, `false`).
- `cp_upcoming_meeting_create_failed_total` — counter labeled by `failure_class` (`validation`, `authorization`, `feature_disabled`, `storage_error`, `unknown`).
- `cp_upcoming_meeting_validation_failed_total` — counter labeled by `validation_class` (`missing_title`, `invalid_scheduled_start`, `participant_missing_identity`, `too_many_participants`, `invalid_email`, `unknown`).
- `cp_upcoming_meeting_updated_total` — counter labeled by `changed_field_class` (`metadata`, `schedule`, `participants`, `multiple`).
- `cp_upcoming_meeting_update_failed_total` — counter labeled by `failure_class` (`validation`, `authorization`, `feature_disabled`, `not_editable`, `storage_error`, `unknown`).
- `cp_upcoming_meeting_cancelled_total` — counter labeled by `had_briefing_status` (`none`, `queued`, `generating`, `ready`, `stale`, `failed`, `unknown`).
- `cp_upcoming_meeting_cancel_failed_total` — counter labeled by `failure_class` (`authorization`, `feature_disabled`, `not_cancellable`, `storage_error`, `unknown`).
- `cp_upcoming_meeting_list_loaded_total` — counter labeled by `view` (`list`, `calendar`, `dashboard`) and `result_count_bucket` (`0`, `1`, `2_5`, `6_20`, `21_plus`).
- `cp_upcoming_meeting_viewed_total` — counter labeled by `status` (`scheduled`, `cancelled`) and `briefing_status` (`disabled`, `unavailable`, `queued`, `generating`, `ready`, `stale`, `failed`, `unknown`).
- `cp_upcoming_meeting_action_duration_ms` — histogram labeled by `action` (`create`, `update`, `cancel`, `list`, `detail`, `calendar`) and `result` (`success`, `failure`).
- `cp_upcoming_meeting_briefing_trigger_queued_total` — counter labeled by `trigger` (`create`, `meaningful_edit`).
- `cp_upcoming_meeting_briefing_trigger_skipped_total` — counter labeled by `reason` (`pre_call_briefing_disabled`, `cancelled`, `not_eligible`, `dependency_unavailable`, `no_meaningful_change`).
- `cp_upcoming_meeting_briefing_trigger_failed_total` — counter labeled by `failure_class` (`queue_unavailable`, `authorization`, `validation`, `unknown`).
- `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` — histogram labeled by `trigger` (`create`, `meaningful_edit`) and `result` (`queued`, `skipped`, `failed`).

Metric labels must use controlled low-cardinality enums. Metrics must not include user IDs, meeting IDs, titles, descriptions, participant names, participant emails, organization/client values, or private user-entered content.

### Log events

- `upcoming_meeting.create_started` — emitted when server-side creation begins; fields: `request_id`, `user_id_hash`, `source`.
- `upcoming_meeting.create_completed` — emitted after successful persistence; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `participant_count`, `has_client_or_org`.
- `upcoming_meeting.create_failed` — emitted after failed creation; fields: `request_id`, `user_id_hash`, `failure_class`, `validation_class`, `retryable`.
- `upcoming_meeting.update_started` — emitted when server-side update begins; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `changed_field_class`.
- `upcoming_meeting.update_completed` — emitted after successful update; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `changed_field_class`, `meaningful_edit`.
- `upcoming_meeting.update_failed` — emitted after failed update; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `validation_class`, `retryable`.
- `upcoming_meeting.cancel_completed` — emitted after soft cancellation; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `had_briefing_status`.
- `upcoming_meeting.cancel_failed` — emitted after failed cancellation; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `retryable`.
- `upcoming_meeting.list_loaded` — emitted when list, calendar, or dashboard upcoming data loads; fields: `request_id`, `user_id_hash`, `view`, `result_count_bucket`.
- `upcoming_meeting.viewed` — emitted when a detail view loads; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `status`, `briefing_status`.
- `upcoming_meeting.briefing_trigger_queued` — emitted when BRD-04 generation/regeneration is queued; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `trigger`.
- `upcoming_meeting.briefing_trigger_skipped` — emitted when trigger is intentionally skipped; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `reason`.
- `upcoming_meeting.briefing_trigger_failed` — emitted when trigger enqueue fails; fields: `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `retryable`.

Logs may include hashed or opaque identifiers only. Logs must not include raw meeting titles, descriptions, agenda/preparation notes, participant names, participant emails, organization/client values, briefing text, source snippets, or other user-entered private content.

### Health/readiness endpoints

- `GET /live` — unchanged process liveness check; returns 200 when the service process is alive.
- `GET /ready` — reflects dependencies needed to create, read, update, and cancel upcoming meetings, including primary storage. BRD-04 queue degradation should be visible in readiness details where app conventions allow, but must not make BRD-05 unavailable when upcoming meeting storage is healthy and trigger failure can be safely skipped or surfaced.

---

## Feature Flag

- **Flag name:** `ff_enable_upcoming_meetings`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_UPCOMING_MEETINGS`
- **Browser env:** `VITE_FF_ENABLE_UPCOMING_MEETINGS`
- **Scope:** Backend upcoming meeting API/actions and frontend upcoming meeting dashboard/list/calendar/detail/create/edit/cancel UI.
- **Disabled behavior:** Frontend hides upcoming meeting entry points and actions. Backend rejects BRD-05 create/list/detail/edit/cancel requests with a safe feature-disabled response.
- **BRD-04 interaction:** BRD-04 briefing trigger behavior requires both `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`. Browser briefing entry/status affordances require both relevant browser flags when visible in BRD-05 surfaces.
- **Registry:** Must be registered in `specs/feature-flags.md` under Domain Layer as Planned.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | With `FF_ENABLE_UPCOMING_MEETINGS=false` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=false`, upcoming meeting UI entry points are hidden and backend create/list/detail/edit/cancel requests return a safe feature-disabled response. | E2E + integration flag eval |
| AC-2 | An authenticated user can create an upcoming meeting with title and scheduled start, and the record is persisted with `created_by` set to that user. | E2E + integration eval |
| AC-3 | Creation accepts optional description/agenda, meeting-level client/organization, and structured participant rows with display name, email, and optional organization. | E2E + integration eval |
| AC-4 | A participant row with neither display name nor email is rejected with field-specific validation feedback and submitted form values are preserved. | E2E + validation unit eval |
| AC-5 | Creation and edits reject scheduled_start values earlier than 15 minutes before server current time. | Unit + integration eval with controlled clock |
| AC-6 | The current user can view their upcoming meetings through dashboard, list, calendar-style, and detail views. | E2E eval |
| AC-7 | Upcoming meeting list and calendar-style views exclude cancelled meetings by default and sort or display scheduled meetings by date/time. | E2E eval |
| AC-8 | Users can edit title, scheduled start, description, client/organization, and participants until 15 minutes after scheduled start. | E2E + integration eval |
| AC-9 | After 15 minutes past scheduled start, edit and cancel actions are unavailable or rejected with a safe not-editable/not-cancellable response. | E2E + integration eval with controlled clock |
| AC-10 | Cancelling an upcoming meeting sets `status=cancelled`, hides it from default active views, prevents future briefing trigger attempts, and preserves the record. | Integration + E2E eval |
| AC-11 | Owner-only authorization prevents another authenticated user from listing, viewing, editing, cancelling, or triggering briefing behavior for an upcoming meeting they do not own. | Security integration eval |
| AC-12 | Participant metadata does not grant access to the upcoming meeting. A participant email matching another user account still cannot view the meeting unless they are the owner. | Security integration eval |
| AC-13 | When both `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`, creating a scheduled upcoming meeting queues BRD-04 briefing generation asynchronously without blocking creation. | Integration eval |
| AC-14 | When both server flags are enabled, meaningful edits queue BRD-04 regeneration or mark the active briefing stale according to the BRD-04 contract while keeping the latest completed briefing visible. | Integration eval |
| AC-15 | When BRD-04 is disabled, upcoming meeting creation and edits still succeed under BRD-05 rules, but briefing trigger behavior is skipped safely and observably. | Integration + observability eval |
| AC-16 | The p95 latency for create/update/cancel/list/detail/calendar actions is under 500ms in the performance fixture, excluding asynchronous briefing generation. | Performance eval |
| AC-17 | The p95 latency for briefing trigger enqueue or safe skip is under 100ms in the performance fixture. | Performance eval |
| AC-18 | Metrics and logs cover CRUD, validation, view usage, and briefing trigger queued/skipped/failed behavior without raw meeting titles, descriptions, participant names, emails, organization/client values, or private content. | Observability/security eval |
| AC-19 | `GET /ready` reports storage dependency health for upcoming meeting operations and does not make BRD-05 unavailable solely because BRD-04 generation dependencies are degraded when the trigger can be safely skipped or surfaced. | Integration readiness eval |
| AC-20 | The feature flag `ff_enable_upcoming_meetings` is registered in `specs/feature-flags.md` with server and browser env vars defaulting false. | Architecture feature-flag eval |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Scope creeps into calendar provider sync or recurrence | Medium | High | Explicitly defer provider IDs, external import, recurrence, reminders, and scheduling workflows to future BRDs |
| Participant metadata is mistaken for access control | Medium | High | State owner-only authorization in FRs, NFRs, and acceptance criteria; participant fields support matching only |
| Calendar-style view expands into a full calendar product | Medium | Medium | Limit calendar-style view to read/navigation of manually created upcoming meetings |
| Meaningful edits cause stale or duplicate briefing jobs | Medium | Medium | Define meaningful edit classes, require async trigger idempotency in curation/implementation, and keep latest completed briefing visible per BRD-04 |
| BRD-04 dependency degradation blocks meeting creation | Medium | Medium | Creation succeeds under BRD-05 rules; trigger enqueue is asynchronous and can be skipped/failed safely and observably |
| Feature flag mismatch exposes partial functionality | Medium | Medium | Use separate BRD-05 flag plus explicit BRD-04 dual-flag dependency; add flag parity acceptance criteria |
| Privacy leakage through logs or metrics | Low | High | Ban raw title, description, participant, email, organization/client, and private text from logs and metric labels; use opaque/hashes only |
| Matching metadata quality is inconsistent because users omit participants/orgs | Medium | Medium | Make metadata optional but visible; BRD-04 already supports no/weak prior memory behavior and should degrade safely |
| Soft-cancelled records accumulate without purge policy | Medium | Low | Inherit retention/purge behavior from BRD-06; hide cancelled records from default active views |

---

## Relations

- **Parent feature:** ContextPilot meeting intelligence / pre-call preparation loop.
- **Depends on:** App Shell for dashboard/navigation surfaces; ADR-0009 for owner-only access semantics.
- **Related specs:** `specs/domain/brd-04-pre-call-briefing.md`, `docs/adr/0009-meeting-acl-semantics.md`, `specs/feature-flags.md`, `specs/security/brd-06-privacy-retention-controls.md`.
- **Blocks:** BRD-04 implementation readiness for the `upcoming_meetings` foreign key and automatic briefing trigger source.
- **Future BRDs:** Provider connectors/calendar sync, recurrence, reminders/delivery channels, participant-based access, organization sharing, and hard-delete/retention controls.

---

## Open Questions

None. The current draft resolves MVP scope as matching-aware manual creation with a full upcoming meetings section, owner-only authorization, soft cancellation, BRD-04 trigger integration, privacy-safe observability, and BRD-06 retention inheritance.
