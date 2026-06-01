# BRD-05: Manual Upcoming Meeting Creation — Canonical Build Spec

|||> **Status:** APPROVED
> **Source:** `specs/domain/brd-05-manual-upcoming-meeting-creation.md`

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-05` |
| Title | Manual Upcoming Meeting Creation |
| Author | brainstormer |
| Created | 2026-05-23 |
| Status | APPROVED |
| Priority | P1 |
| Phase | Phase 2 |
| Feature flag | `ff_enable_upcoming_meetings` |
| Spec path | `specs/domain/brd-05-manual-upcoming-meeting-creation.md` |

---

## Overview

Manual Upcoming Meeting Creation gives authenticated users a first-party way to create and manage future meeting records that ContextPilot can use as anchors for pre-call preparation. MVP is matching-aware but not calendar-integrated: users manually enter title, scheduled start, optional description/agenda, participants, and organization/client metadata, then manage records through dashboard, list, calendar-style, detail, edit, and cancel surfaces. Provider sync, recurrence, reminders, hard delete, and participant-based access control are explicitly deferred.

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

| ID | Requirement |
|----|-------------|
| FR-1 | Feature gated by `ff_enable_upcoming_meetings`. When disabled, UI entry points are hidden and backend create/update/cancel/list/detail APIs reject requests with a safe feature-disabled response. |
| FR-2 | Full upcoming meetings section: dashboard entry point, `/upcoming` list view, calendar-style view, `/upcoming/new` creation flow, `/upcoming/{id}` detail view, edit/cancel actions. |
| FR-3 | Manual creation fields: required `title` and `scheduled_start`; optional `description`/agenda, meeting-level `client_or_organization`, zero or more structured participants. |
| FR-4 | Structured participant metadata: `display_name`, `email`, optional `organization`. Supports BRD-04 matching; does not grant access. |
| FR-5 | Data model — `upcoming_meetings` table: `id` UUID PK, `title` text, `scheduled_start` timestamptz, `description` nullable, `client_or_organization` nullable, `status` enum/string, `created_by` UUID owner ref, `created_at` timestamptz, `updated_at` timestamptz. **Indexes:** PK on `id`; compound index on `(created_by, status, scheduled_start ASC)` for owner-list queries sorted by nearest scheduled start; index on `scheduled_start` for time-based queries. |
| FR-6 | Data model — `upcoming_meeting_participants` table: `id` UUID PK, `upcoming_meeting_id` UUID (plain column, application-level referential integrity; no DB FK constraint), `display_name` nullable, `email` nullable, `organization` nullable, `created_at` timestamptz, `updated_at` timestamptz. Participant must have at least one of display name or email. **Indexes:** PK on `id`; index on `upcoming_meeting_id` for participant lookup by meeting. |
| FR-7 | Status model: at least `scheduled` and `cancelled`. Scheduled appear in active views by default. Cancelled hidden from default views but available from detail/history. |
| FR-8 | Scheduled-start validation: `scheduled_start` must be no earlier than 15 minutes before server current time. Grace-window violations fail with field-specific feedback. |
| FR-9 | Edit window: users can edit title, scheduled start, description, client/organization, and participants until 15 minutes after the meeting's stored `scheduled_start` at time of evaluation. Evaluation uses server time against the stored `scheduled_start` value at request time. Rescheduling to a new `scheduled_start` re-anchors the edit window to 15 minutes after the new stored value (if the edit is accepted). After 15 minutes past the stored `scheduled_start`, the record is read-only for MVP. |
| FR-10 | Soft cancellation: users can cancel until 15 minutes after the meeting's stored `scheduled_start` at time of evaluation. Cancellation sets `status=cancelled`, prevents future briefing generation, hides from default active views, preserves history. Cancellation is permanent; status cannot be reverted to scheduled. |
| FR-11 | Owner-only authorization: only `created_by` owner can create/view/list/edit/cancel their upcoming meetings. Participants receive no access rights. Follows ADR-0009 owner-only semantics. |
| FR-12 | List view behavior: `/upcoming` shows current user's non-cancelled upcoming meetings sorted by nearest scheduled start first, with clear empty states and create affordance. |
| FR-13 | Calendar-style view behavior: read/navigation affordances for upcoming meetings by date/time. No provider sync, external calendar import, recurrence management, reminders, or availability scheduling in MVP. |
| FR-14 | Detail view behavior: shows meeting metadata, participant metadata, status, edit/cancel actions when allowed, briefing status/entry points only when BRD-04 is enabled. |
| FR-15 | Validation and form preservation: server is authoritative. Client-side validation may provide helper feedback. Submitted form values preserved and echoed back after validation failures. |
| FR-16 | Meaningful edit definition: any change to title, scheduled start, description, meeting-level organization/client, participant email, participant display name, or participant organization; adding or removing a participant is also a meaningful edit. |
| FR-17 | BRD-04 trigger on create: when `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`, successful creation of a scheduled upcoming meeting queues BRD-04 pre-call briefing generation asynchronously. Creation does not wait for generation. Trigger enqueue is outside the meeting+participants DB transaction; trigger failure must not roll back meeting persistence and must be observable via `cp_upcoming_meeting_briefing_trigger_failed_total`. |
| FR-18 | BRD-04 trigger on meaningful edit: when both flags enabled, meaningful edit queues BRD-04 regeneration or marks active briefing stale per BRD-04 contract. Latest completed briefing remains visible but marked outdated while regeneration runs. Trigger enqueue is outside the meeting+participants DB transaction; trigger failure must not roll back the edit and must be observable. |
| FR-18a | Stale briefing UX/semantics: when a briefing is marked stale (AC-14), the latest completed briefing remains visible with a visible "outdated — regenerating" status indicator. The stale briefing is not hidden, not inaccessible, and not replaced until the new briefing completes. No silent hidden stale state. New briefing must complete before the stale is fully superseded in the UI. Stale briefing is visible-but-outdated, not inaccessible. |
| FR-19 | BRD-04 trigger skipped states: if BRD-04 is disabled, dependencies unavailable, upcoming meeting is cancelled, or record is outside editable/eligible window, BRD-05 creation/edit follows its own rules but briefing trigger skipped safely and observably. |
| FR-20 | Privacy-safe diagnostics: user-facing validation and failure messages are actionable but do not expose private content. Operator diagnostics use opaque or hashed identifiers only and must not include meeting titles, descriptions, participant names, emails, or organization/client values. **Hash algorithm:** `SHA-256` truncated to 16 hex characters (8 bytes) for `user_id_hash` and `upcoming_meeting_id_hash`. Hash is deterministic per deployment (no per-session salt) to enable log correlation. Collision risk is negligible at this truncation length for UUID-based identifiers. |

### Should Have

| ID | Requirement |
|----|-------------|
| FR-21 | Dashboard preparation entry: compact upcoming meetings card/section linking to full section, highlighting next scheduled meeting when available. |
| FR-22 | Briefing trigger status: detail views show whether briefing is unavailable, queued, generating, ready, stale, failed, or skipped because BRD-04 is disabled, reusing BRD-04 status conventions where possible. |
| FR-23 | Cancelled/history affordance: low-prominence way to find cancelled upcoming meetings for audit/debugging without cluttering default preparation views. |
| FR-24 | Duplicate awareness: UI warns when the same authenticated user creates another upcoming meeting with the same normalized title and same `scheduled_start` (exact minute), within a short time window, while still allowing creation if the user proceeds. **Normalized title:** lowercase, trim leading/trailing whitespace, collapse internal whitespace (two or more consecutive spaces→single space). Scope is same authenticated user only. Time window is exact `scheduled_start` minute match (±0 minutes). Duplicate warning is a pre-submission advisory shown in a `role="alert"` ARIA live region; it does not block submission. |
| FR-25 | Participant input ergonomics: creation/edit form supports adding, removing, and editing multiple participant rows without losing entered values during validation errors. |
| FR-26 | Low-cardinality validation classes: `missing_title`, `invalid_scheduled_start`, `participant_missing_identity`, `too_many_participants`, `invalid_email`, `unknown`. |

### Could Have

| ID | Requirement |
|----|-------------|
| FR-27 | Provider/calendar sync: external calendar provider import, provider event IDs, sync conflict handling, webhook updates — deferred to future provider connectors BRD. |
| FR-28 | Recurrence: recurring meeting series, recurrence exceptions, series-level matching — deferred. |
| FR-29 | Reminders and delivery: email, Teams, Slack, browser notification, mobile push, or calendar reminder delivery — deferred. |
| FR-30 | Participant-based access: participant access rights, organization-wide sharing, cross-user upcoming meeting visibility — deferred beyond owner-only MVP. |
| FR-31 | Hard delete: hard deletion and purge behavior — deferred to BRD-06 Privacy & Retention Controls unless required by later security review. |
| FR-32 | Availability scheduling: finding times, RSVP state, availability overlays, meeting invitation workflows — out of scope. |

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Create/update/cancel latency | p95 < 500ms for BRD-05 server action completion, excluding async BRD-04 generation work |
| List/detail/calendar view latency | p95 < 500ms for current user's upcoming meeting data retrieval and render path, excluding client/network variability |
| Briefing trigger enqueue latency | p95 < 100ms to enqueue or safely skip the BRD-04 briefing trigger after successful create or meaningful edit |
| Availability | Upcoming meeting create/list/detail/edit/cancel follow app-baseline availability; BRD-04 generation degradation must not prevent BRD-05 meeting creation when BRD-05 is healthy |
| Authorization | 100% of BRD-05 read/write actions enforce owner-only access using `created_by`; participant metadata grants no access |
| Privacy | Logs and metric labels contain no raw meeting titles, descriptions, agenda text, participant names, emails, organization/client values, or user-entered private content |
| Scale | MVP supports at least 100 active upcoming meetings per user and up to 50 participants per upcoming meeting without degrading p95 targets in fixture tests. **50 participants is a hard validation limit** enforced with `too_many_participants` class; requests exceeding 50 participants are rejected with field-specific feedback. |
| Validation consistency | Server validation is authoritative for 100% of create/edit/cancel actions; client-side checks are advisory only |
| Accessibility | Dashboard entry, list, calendar-style view, detail, form validation, edit, and cancel flows meet WCAG 2.1 AA expectations. **Per-flow requirements (no CSS/code syntax):** keyboard path navigable from page entry to action completion; focus moves to first error field on validation failure; error text associated with fields via `aria-describedby`; 50-participant warning shown at threshold and error shown at limit with `too_many_participants` class; duplicate warning shown in `role="alert"` live region; form values preserved on all validation failures. |
| Retention | Upcoming meeting retention, purge, export, and hard delete inherit final policy from BRD-06 Privacy & Retention Controls; BRD-05 defines only soft cancellation |

---

## Observability

### Metrics to emit

| Metric | Type | Labels |
|--------|------|--------|
| `cp_upcoming_meeting_create_requested_total` | Counter | `source` (`dashboard`, `list`, `calendar`, `direct`), `has_participants` (`true`, `false`) |
| `cp_upcoming_meeting_created_total` | Counter | `participant_count_bucket` (`0`, `1`, `2_5`, `6_10`, `11_50`), `has_client_or_org` (`true`, `false`) |
| `cp_upcoming_meeting_create_failed_total` | Counter | `failure_class` (`validation`, `authorization`, `feature_disabled`, `storage_error`, `unknown`) |
| `cp_upcoming_meeting_validation_failed_total` | Counter | `validation_class` (`missing_title`, `invalid_scheduled_start`, `participant_missing_identity`, `too_many_participants`, `invalid_email`, `unknown`) |
| `cp_upcoming_meeting_updated_total` | Counter | `changed_field_class` (`metadata`, `schedule`, `participants`, `multiple`) |
| `cp_upcoming_meeting_update_failed_total` | Counter | `failure_class` (`validation`, `authorization`, `feature_disabled`, `not_editable`, `storage_error`, `unknown`) |
| `cp_upcoming_meeting_cancelled_total` | Counter | `had_briefing_status` (`none`, `queued`, `generating`, `ready`, `stale`, `failed`, `unknown`) |
| `cp_upcoming_meeting_cancel_failed_total` | Counter | `failure_class` (`authorization`, `feature_disabled`, `not_cancellable`, `storage_error`, `unknown`) |
| `cp_upcoming_meeting_list_loaded_total` | Counter | `view` (`list`, `calendar`, `dashboard`), `result_count_bucket` (`0`, `1`, `2_5`, `6_20`, `21_plus`) |
| `cp_upcoming_meeting_viewed_total` | Counter | `status` (`scheduled`, `cancelled`), `briefing_status` (`disabled`, `unavailable`, `queued`, `generating`, `ready`, `stale`, `failed`, `unknown`) |
| `cp_upcoming_meeting_action_duration_ms` | Histogram | `action` (`create`, `update`, `cancel`, `list`, `detail`, `calendar`), `result` (`success`, `failure`) — buckets: `25, 50, 100, 250, 500, 1000` ms |
| `cp_upcoming_meeting_briefing_trigger_queued_total` | Counter | `trigger` (`create`, `meaningful_edit`) |
| `cp_upcoming_meeting_briefing_trigger_skipped_total` | Counter | `reason` (`pre_call_briefing_disabled`, `cancelled`, `not_eligible`, `dependency_unavailable`, `no_meaningful_change`) |
| `cp_upcoming_meeting_briefing_trigger_failed_total` | Counter | `failure_class` (`queue_unavailable`, `authorization`, `validation`, `unknown`) |
| `cp_upcoming_meeting_briefing_trigger_enqueue_duration_ms` | Histogram | `trigger` (`create`, `meaningful_edit`), `result` (`queued`, `skipped`, `failed`) — buckets: `10, 25, 50, 100, 250, 500` ms |

Metric labels must use controlled low-cardinality enums. Metrics must not include user IDs, meeting IDs, titles, descriptions, participant names, emails, organization/client values, or private user-entered content.

### Log events

| Event | Emitted when | Fields |
|-------|--------------|--------|
| `upcoming_meeting.create_started` | Server-side creation begins | `request_id`, `user_id_hash`, `source` |
| `upcoming_meeting.create_completed` | After successful persistence | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `participant_count`, `has_client_or_org` |
| `upcoming_meeting.create_failed` | After failed creation | `request_id`, `user_id_hash`, `failure_class`, `validation_class`, `retryable` |
| `upcoming_meeting.update_started` | Server-side update begins | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `changed_field_class` |
| `upcoming_meeting.update_completed` | After successful update | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `changed_field_class`, `meaningful_edit` |
| `upcoming_meeting.update_failed` | After failed update | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `validation_class`, `retryable` |
| `upcoming_meeting.cancel_completed` | After soft cancellation | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `had_briefing_status` |
| `upcoming_meeting.cancel_failed` | After failed cancellation | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `retryable` |
| `upcoming_meeting.list_loaded` | List, calendar, or dashboard upcoming data loads | `request_id`, `user_id_hash`, `view`, `result_count_bucket` |
| `upcoming_meeting.viewed` | Detail view loads | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `status`, `briefing_status` |
| `upcoming_meeting.briefing_trigger_queued` | BRD-04 generation/regeneration queued | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `trigger` |
| `upcoming_meeting.briefing_trigger_skipped` | Trigger intentionally skipped | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `reason` |
| `upcoming_meeting.briefing_trigger_failed` | Trigger enqueue fails | `request_id`, `user_id_hash`, `upcoming_meeting_id_hash`, `failure_class`, `retryable` |

Logs may include hashed or opaque identifiers only. Logs must not include raw meeting titles, descriptions, agenda/preparation notes, participant names, emails, organization/client values, briefing text, source snippets, or other private content.

### Health/readiness endpoints

| Endpoint | Behavior |
|----------|----------|
| `GET /live` | Unchanged process liveness check; returns 200 when service process is alive. |
| `GET /ready` | Reflects dependencies needed to create, read, update, and cancel upcoming meetings, including primary storage. Returns HTTP 200 when upcoming meeting storage is healthy. Includes `brd04_trigger: "available"|"degraded"|"unavailable"` in response body diagnostics field; BRD-04 queue degradation is surfaced here without making BRD-05 unavailable when upcoming meeting storage is healthy and trigger failure can be safely skipped or surfaced. |

---

## Feature Flag

| Field | Value |
|-------|-------|
| Flag name | `ff_enable_upcoming_meetings` |
| Type | boolean |
| Default | `false` |
| Server env | `FF_ENABLE_UPCOMING_MEETINGS` |
| Browser env | `VITE_FF_ENABLE_UPCOMING_MEETINGS` |
| Scope | Backend upcoming meeting API/actions and frontend upcoming meeting dashboard/list/calendar/detail/create/edit/cancel UI |
| Disabled behavior | Frontend hides upcoming meeting entry points and actions. Backend rejects BRD-05 create/list/detail/edit/cancel requests with a safe feature-disabled response. |
| BRD-04 interaction | BRD-04 briefing trigger requires both `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`. Browser briefing entry/status affordances require both relevant browser flags when visible in BRD-05 surfaces. |
| Registry | Must be registered in `specs/feature-flags.md` under Domain Layer as Planned. |

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | With `FF_ENABLE_UPCOMING_MEETINGS=false` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=false`, upcoming meeting UI entry points are hidden and backend create/list/detail/edit/cancel requests return a safe feature-disabled response. | E2E + integration flag eval |
| AC-2 | Authenticated user can create an upcoming meeting with title and scheduled start; record is persisted with `created_by` set to that user. | E2E + integration eval |
| AC-3 | Creation accepts optional description/agenda, meeting-level client/organization, and structured participant rows with display name, email, optional organization. | E2E + integration eval |
| AC-4 | Participant row with neither display name nor email is rejected with field-specific validation feedback and submitted form values are preserved. | E2E + validation unit eval |
| AC-5 | Creation and edits reject `scheduled_start` values earlier than 15 minutes before server current time. | Unit + integration eval with controlled clock |
| AC-6 | Current user can view their upcoming meetings through dashboard, list, calendar-style, and detail views. | E2E eval |
| AC-7 | List and calendar-style views exclude cancelled meetings by default and sort/display scheduled meetings by date/time. | E2E eval |
| AC-8 | Users can edit title, scheduled start, description, client/organization, and participants until 15 minutes after scheduled start. | E2E + integration eval |
| AC-9 | After 15 minutes past scheduled start, edit and cancel actions are unavailable or rejected with a safe not-editable/not-cancellable response. | E2E + integration eval with controlled clock |
| AC-10 | Cancelling sets `status=cancelled`, hides from default active views, prevents future briefing trigger attempts, and preserves the record. | Integration + E2E eval |
| AC-11 | Owner-only authorization prevents another authenticated user from listing, viewing, editing, cancelling, or triggering briefing for an upcoming meeting they do not own. | Security integration eval |
| AC-12 | Participant metadata does not grant access; participant email matching another user account still cannot view the meeting unless they are the owner. | Security integration eval |
| AC-13 | When both `FF_ENABLE_UPCOMING_MEETINGS=true` and `FF_ENABLE_PRE_CALL_BRIEFING=true`, creating a scheduled upcoming meeting queues BRD-04 briefing generation asynchronously without blocking creation. | Integration eval |
| AC-14 | When both server flags enabled, meaningful edits queue BRD-04 regeneration or mark active briefing stale per BRD-04 contract while keeping latest completed briefing visible. | Integration eval |
| AC-15 | When BRD-04 is disabled, upcoming meeting creation and edits still succeed under BRD-05 rules, but briefing trigger is skipped safely and observably. | Integration + observability eval |
| AC-16 | p95 latency for create/update/cancel/list/detail/calendar actions is under 500ms in performance fixture, excluding async briefing generation. | Performance eval |
| AC-17 | p95 latency for briefing trigger enqueue or safe skip is under 100ms in performance fixture. | Performance eval |
| AC-18 | Metrics and logs cover CRUD, validation, view usage, and briefing trigger queued/skipped/failed without raw meeting titles, descriptions, participant names, emails, organization/client values, or private content. | Observability/security eval |
| AC-19 | `GET /ready` reports storage dependency health for upcoming meeting operations and does not make BRD-05 unavailable solely because BRD-04 generation dependencies are degraded when trigger can be safely skipped or surfaced. | Integration readiness eval |
| AC-20 | Feature flag `ff_enable_upcoming_meetings` is registered in `specs/feature-flags.md` with server and browser env vars defaulting false. | Architecture feature-flag eval |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Scope creeps into calendar provider sync or recurrence | Medium | High | Explicitly defer provider IDs, external import, recurrence, reminders, and scheduling workflows to future BRDs |
| Participant metadata is mistaken for access control | Medium | High | State owner-only authorization in FRs, NFRs, and acceptance criteria; participant fields support matching only |
| Calendar-style view expands into a full calendar product | Medium | Medium | Limit calendar-style view to read/navigation of manually created upcoming meetings |
| Meaningful edits cause stale or duplicate briefing jobs | Medium | Medium | Define meaningful edit classes, require async trigger idempotency in curation/implementation, keep latest completed briefing visible per BRD-04 |
| BRD-04 dependency degradation blocks meeting creation | Medium | Medium | Creation succeeds under BRD-05 rules; trigger enqueue is async and can be skipped/failed safely and observably |
| Feature flag mismatch exposes partial functionality | Medium | Medium | Use separate BRD-05 flag plus explicit BRD-04 dual-flag dependency; add flag parity acceptance criteria |
| Privacy leakage through logs or metrics | Low | High | Ban raw title, description, participant, email, organization/client, and private text from logs and metric labels; use opaque/hashes only |
| Matching metadata quality is inconsistent because users omit participants/orgs | Medium | Medium | Make metadata optional but visible; BRD-04 already supports no/weak prior memory behavior and degrades safely |
| Soft-cancelled records accumulate without purge policy | Medium | Low | Inherit retention/purge behavior from BRD-06; hide cancelled records from default active views |

---

## Relations

| Relation | Details |
|----------|---------|
| Parent feature | ContextPilot meeting intelligence / pre-call preparation loop |
| Depends on | App Shell for dashboard/navigation surfaces; ADR-0009 for owner-only access semantics; BRD-05 defines `upcoming_meetings` gap and minimum schema requirement |
| Related specs | `specs/domain/brd-04-pre-call-briefing.md`, `docs/adr/0009-meeting-acl-semantics.md`, `specs/feature-flags.md`, `specs/security/brd-06-privacy-retention-controls.md` |
| Blocks | BRD-04 implementation readiness for `upcoming_meetings` foreign key and automatic briefing trigger source |
| Future BRDs | Provider connectors/calendar sync, recurrence, reminders/delivery channels, participant-based access, organization sharing, hard-delete/retention controls |

---

## Explicit Scope Boundaries

**Deferred to future BRDs:**
- Provider/calendar sync (external calendar provider import, provider event IDs, sync conflict handling, webhook updates)
- Recurrence (recurring meeting series, recurrence exceptions, series-level matching)
- Reminders and delivery (email, Teams, Slack, browser notification, mobile push, calendar reminder)
- Participant-based access (participant access rights, organization-wide sharing, cross-user upcoming meeting visibility)
- Hard delete (hard deletion and purge — inherits from BRD-06 Privacy & Retention Controls)
- Availability scheduling (finding times, RSVP state, availability overlays, meeting invitation workflows)

---

## Open Questions

None.