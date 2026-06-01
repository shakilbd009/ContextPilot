# BRD-04: Pre-Call Briefing

> **Status:** Approved

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-04` |
| Title | Pre-Call Briefing |
| Author | brainstormer |
| Created | 2026-05-23 |
| Status | Approved |
| Priority | P1 |
| Phase | Phase 2 |
| Feature flag | `ff_enable_pre_call_briefing` |
| Spec path | `specs/domain/brd-04-pre-call-briefing.md` |

---

## Overview

Pre-Call Briefing helps users prepare for upcoming meetings by generating a concise, traceable briefing from related prior meeting memory. It is the flagship ContextPilot experience: users should quickly understand what happened before, what remains unresolved, what to focus on, and what questions to ask without manually reviewing old notes.

The MVP includes simple, explainable automatic related-meeting selection so the product can prove the core promise — never walk into a meeting cold again — while preserving user trust through source details, evidence caveats, source exclusions, and versioned briefing history.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | Authenticated user | A briefing to generate automatically for an upcoming meeting | I can prepare without remembering to manually search old meetings |
| US-2 | Authenticated user | A concise summary with objective, preparation status, recommended focus, prior context, open actions, and risks/questions | I can understand what matters in under 60 seconds |
| US-3 | Authenticated user | The system to find related prior meetings using explainable matching | I get useful prior context without manually selecting every source |
| US-4 | Authenticated user | To inspect source details and relatedness reasons behind briefing content | I can verify why information was included before relying on it |
| US-5 | Authenticated user | To exclude an irrelevant source meeting and regenerate the briefing | I can correct bad automatic matches without losing control |
| US-6 | Authenticated user | The latest completed briefing to remain visible while regeneration runs | I am not blocked if a new briefing takes time or fails |
| US-7 | Authenticated user | Weak-evidence content to be caveated and unresolved conflicts excluded | I can trust that briefing advice does not overstate uncertain memory |
| US-8 | Authenticated user | A limited preparation shell when no prior memory exists | I still get useful prep guidance without fake continuity |

---

## Functional Requirements

### Must Have

- **FR-1: Upcoming meeting briefing entry point.** The feature provides a pre-call briefing entry point from an upcoming meeting detail page. The briefing entry point is visible only when `ff_enable_pre_call_briefing` is enabled and the user is authorized to view the meeting.
- **FR-2: Automatic async generation on upcoming meeting creation.** When an upcoming meeting is created and `FF_ENABLE_PRE_CALL_BRIEFING=true`, the system queues an asynchronous briefing generation job. The user can leave and return while generation runs.
- **FR-3: Manual regenerate.** Users can manually request regeneration from the upcoming meeting detail page. Regeneration runs asynchronously and does not remove the latest completed briefing while a new version is pending.
- **FR-4: Balanced related-meeting matching.** The MVP automatically selects related prior meetings when at least two of the following explainable signals match: same participant, similar title, same organization/client, close chronology. The MVP signal definitions are:
  - **Same participant:** at least one participant in the upcoming meeting and prior meeting shares an exact email address. If email is unavailable, this signal does not pass; display-name-only matching is deferred to advanced related-meeting detection.
  - **Similar title:** normalized titles have Levenshtein distance of 3 or less, or share a common token sequence of 4 or more consecutive words, matching the constrained BRD-03 prior-memory convention.
  - **Same organization/client:** at least one participant organization/company value matches exactly after normalization, or the upcoming meeting metadata explicitly names the same organization/client as a prior meeting.
  - **Close chronology:** the prior meeting completed within 90 days before the upcoming meeting's scheduled start. Future-dated source meetings must not qualify as prior context.
- **FR-5: Source scope.** Briefing generation uses up to the top 3 highest-ranked qualifying prior meetings. If fewer than 3 prior meetings qualify, generation uses the available qualifying meetings. If none qualify, the system generates a no-prior-memory preparation shell.
- **FR-6: Relatedness explanation.** Each selected source meeting has visible relatedness reasons available behind an expand/details affordance, such as matched participant, similar title, same organization/client, or close chronology.
- **FR-7: Progressive briefing format.** The briefing uses a progressive format: a concise summary first, followed by expandable detailed sections.
- **FR-8: Concise summary contents.** The concise summary always includes meeting objective, preparation status, recommended focus, top prior context, open actions, and risks/questions. If prior memory is weak or missing, affected sections show explicit insufficient-context states instead of invented content.
- **FR-9: Detailed sections.** Expandable detailed sections include previous relevant context, important prior decisions, open action items and commitments, unresolved risks/blockers, open questions, stakeholder notes when allowed, suggested questions, suggested agenda, source details, and source meeting list.
- **FR-10: No-prior-memory preparation shell.** If no related prior memory is found, the system generates a limited shell from upcoming meeting title/description/metadata only, clearly marked `No prior memory found`. It may include objective, recommended focus, and suggested prep questions derived from the upcoming meeting metadata, but it must not imply prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks.
- **FR-11: Evidence policy.** The briefing follows BRD-03 evidence status conventions: `strong_evidence`, `weak_evidence`, `insufficient_evidence`, and `conflicting_evidence`. Strong-evidence items may appear normally. Weak-evidence items may appear with clear caveats. Insufficient-evidence items must not be presented as facts. Unresolved conflicting items are excluded from briefing advice.
- **FR-12: Source annotations behind details.** Every briefing item derived from prior memory is traceable to source memory and source meeting details behind an expand/details affordance. The concise summary remains readable without inline source clutter.
- **FR-13: Stakeholder notes handling.** Stakeholder notes are allowed only in expandable details, never in the concise summary. They must be professional, meeting-relevant, evidence-backed, and caveated when weak. Sensitive personal attributes, unsupported personality inference, private/irrelevant details, and unresolved conflicting stakeholder notes are excluded.
- **FR-14: Versioned briefing records.** Every generated or regenerated briefing is persisted as a distinct version. The active/latest completed version is shown by default. Prior versions remain available for audit/history.
- **FR-15: Stale briefing behavior.** If source meeting memory is reprocessed or changes after a briefing is generated, affected briefing versions are marked stale and regeneration is offered. Stale briefings remain readable with a visible stale indicator until replaced or dismissed.
- **FR-16: Source exclusion.** Users can exclude an automatically selected source meeting before regeneration. Exclusions persist at the upcoming-meeting level and apply to future regenerations for that upcoming meeting only.
- **FR-17: Exclusion undo and visibility.** Users can undo an upcoming-meeting-level source exclusion before regeneration. Excluded source meetings remain visible in a collapsed excluded-sources area with their original relatedness reasons.
- **FR-18: Failure and retry.** Failed generation shows a safe user-facing failure reason, keeps the latest completed briefing available when one exists, and offers retry/regenerate. Operator diagnostics must not expose raw meeting content or participant PII.
- **FR-19: Authorization boundary.** Briefings and source details are visible only to users authorized to view the upcoming meeting and all included source meetings. Unauthorized source meetings must not be selected, displayed, logged, or used in generation.

### Should Have

- **FR-20: Preparation status states.** The briefing should display preparation status values such as `generating`, `ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, and `regenerating`.
- **FR-21: Cached briefing view.** Viewing the latest completed briefing should use persisted/cached briefing data rather than forcing generation on every page view.
- **FR-22: Low-cardinality observability.** Metrics and logs should use controlled enum labels for status, failure class, source count bucket, and generation trigger.
- **FR-23: Product usefulness events.** The system should emit product-usefulness events for viewing a briefing, expanding details/source panels, requesting regeneration, excluding/restoring a source, and generating a no-prior-memory shell.
- **FR-24: Human-readable explanations.** Relatedness and source explanations should be understandable to a non-technical user and should not expose implementation scoring internals.

### Could Have

- **FR-25: Manual addition of non-matched meetings.** A future enhancement may allow users to manually add prior meetings that were not automatically matched.
- **FR-26: Advanced relatedness detection.** A future BRD may define recurring-series detection, topic embeddings, project graph matching, or learned ranking.
- **FR-27: Delivery channels.** A future BRD may send briefings through email, calendar, Teams, Slack, browser notifications, or mobile push.
- **FR-28: Hard briefing length controls.** A future enhancement may define strict character, section, or reading-time limits beyond the MVP progressive structure.

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Generation latency | Async briefing generation completes within p95 < 60s for up to 3 qualifying prior meetings under normal operating conditions |
| Cached view latency | Latest completed briefing view renders within p95 < 500ms excluding network variability outside the app |
| Availability | Briefing pages follow app-baseline availability; generation failures degrade gracefully and do not take down upcoming meeting detail pages |
| Regeneration continuity | Latest completed briefing remains readable while regeneration is queued/running/failed |
| Traceability | 100% of briefing items are traceable either to qualifying source memory or upcoming meeting metadata |
| Evidence safety | 0 unresolved conflicting memory items appear as briefing advice; weak-evidence items are visibly caveated |
| Privacy | Logs and metric labels contain no raw transcript text, notes text, briefing text, source snippets, participant PII, stakeholder note content, or private meeting content |
| Authorization | Briefing generation and viewing use only meetings and memory the authenticated user is authorized to access |
| Accessibility | Briefing UI, generating/failure/stale states, and expand/details controls meet WCAG 2.1 AA expectations |
| Scale | MVP supports generation from up to 3 qualifying source meetings per upcoming meeting and preserves source exclusion state per upcoming meeting |
| Retention | Briefing versions inherit retention/deletion behavior from the associated upcoming meeting until BRD-06 defines final privacy and retention controls |

---

## Observability

### Metrics to emit

- `cp_pre_call_briefing_job_queued_total` — counter labeled by `trigger` (`auto`, `manual_regenerate`) and `source_count_bucket` (`0`, `1`, `2`, `3`).
- `cp_pre_call_briefing_job_started_total` — counter labeled by `trigger`.
- `cp_pre_call_briefing_job_completed_total` — counter labeled by `result` (`ready`, `ready_with_caveats`, `no_prior_memory`) and `source_count_bucket`.
- `cp_pre_call_briefing_job_failed_total` — counter labeled by `failure_class` (`source_unavailable`, `generation_timeout`, `provider_error`, `authorization_denied`, `validation_error`, `unknown`).
- `cp_pre_call_briefing_job_retried_total` — counter labeled by `failure_class` and `attempt_bucket` (`1`, `2`, `3`, `exhausted`).
- `cp_pre_call_briefing_generation_duration_ms` — histogram from job start to terminal success/failure.
- `cp_pre_call_briefing_view_duration_ms` — histogram for rendering/fetching latest completed briefing view.
- `cp_pre_call_briefing_viewed_total` — counter labeled by `status` (`ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, `generating`).
- `cp_pre_call_briefing_regenerate_requested_total` — counter labeled by `has_active_briefing` (`true`, `false`).
- `cp_pre_call_briefing_details_expanded_total` — counter labeled by `panel` (`sources`, `evidence`, `stakeholders`, `agenda`, `questions`, `history`).
- `cp_pre_call_briefing_source_excluded_total` — counter labeled by `reason_count_bucket` (`1`, `2`, `3`, `4`).
- `cp_pre_call_briefing_source_restored_total` — counter labeled by `reason_count_bucket`.
- `cp_pre_call_briefing_no_prior_memory_shell_total` — counter labeled by `trigger` (`auto`, `manual_regenerate`).

Metric labels must be low-cardinality controlled enums. The feature must not emit user IDs, meeting IDs, participant identifiers, titles, raw content, snippets, or generated text in labels.

### Log events

- `pre_call_briefing.generation_queued` — emitted when a job is queued; fields: `meeting_id_hash`, `trigger`, `source_count`, `excluded_source_count`.
- `pre_call_briefing.generation_started` — emitted when processing begins; fields: `job_id_hash`, `meeting_id_hash`, `trigger`, `source_count`.
- `pre_call_briefing.generation_completed` — emitted when a briefing version is persisted; fields: `job_id_hash`, `meeting_id_hash`, `briefing_version`, `result`, `source_count`, `duration_ms`.
- `pre_call_briefing.generation_failed` — emitted when generation fails; fields: `job_id_hash`, `meeting_id_hash`, `failure_class`, `retryable`, `attempt`, `duration_ms`.
- `pre_call_briefing.regeneration_requested` — emitted when a user requests regeneration; fields: `meeting_id_hash`, `has_active_briefing`, `excluded_source_count`.
- `pre_call_briefing.viewed` — emitted when a user views a briefing page; fields: `meeting_id_hash`, `status`, `active_version_present`.
- `pre_call_briefing.details_expanded` — emitted when a user expands a details panel; fields: `meeting_id_hash`, `panel`.
- `pre_call_briefing.source_excluded` — emitted when a user excludes a source meeting; fields: `meeting_id_hash`, `source_meeting_id_hash`, `relatedness_reason_count`.
- `pre_call_briefing.source_restored` — emitted when a user restores an excluded source; fields: `meeting_id_hash`, `source_meeting_id_hash`, `relatedness_reason_count`.
- `pre_call_briefing.no_prior_memory_shell_generated` — emitted when the limited shell is generated; fields: `meeting_id_hash`, `trigger`.
- `pre_call_briefing.marked_stale` — emitted when source memory changes and an affected briefing is marked stale; fields: `meeting_id_hash`, `briefing_version`, `source_meeting_id_hash`.

Log events may include hashed or opaque identifiers only. Logs must not include raw meeting titles, transcript text, notes text, generated briefing text, source snippets, participant PII, stakeholder note content, or user-entered private content.

### Health/readiness endpoints

- `GET /live` — unchanged process liveness check; returns 200 when the service process is alive.
- `GET /ready` — reflects dependencies needed to serve briefing views and queue generation work according to app convention. Existing completed briefing views should degrade gracefully if generation dependencies are temporarily unavailable. Readiness should make generation dependency degradation visible without unnecessarily blocking unrelated app functionality.

---

## Feature Flag

- **Flag name:** `ff_enable_pre_call_briefing`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_PRE_CALL_BRIEFING`
- **Browser env:** `VITE_FF_ENABLE_PRE_CALL_BRIEFING`
- **Scope:** Backend generation/API behavior and frontend briefing UI/actions.
- **Disabled behavior:** The frontend hides briefing entry points and generation/regeneration actions. The backend rejects generation requests for this feature with a safe feature-disabled response and does not auto-queue briefing jobs on upcoming meeting creation.
- **Registry:** Already registered in `specs/feature-flags.md` under Domain Layer as Planned.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | With `FF_ENABLE_PRE_CALL_BRIEFING=false` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING=false`, briefing UI entry points are hidden and backend generation/regeneration requests are rejected with a safe feature-disabled response. | E2E + integration flag eval |
| AC-2 | Creating an upcoming meeting with the flag enabled queues an async briefing generation job without blocking access to the upcoming meeting detail page. | Integration eval |
| AC-3 | Manual regenerate queues a new async job while the latest completed briefing remains viewable. | E2E + integration eval |
| AC-4 | Automatic related-meeting matching selects only prior meetings with at least two matching signals among same participant, similar title, same organization/client, and close chronology. | Unit + integration eval with fixture meetings |
| AC-5 | Briefing generation uses no more than the top 3 qualifying prior meetings and uses fewer when fewer qualify. | Integration eval |
| AC-6 | If no qualifying prior memory exists, the system generates a clearly marked no-prior-memory preparation shell without implying prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks. | E2E + content safety eval |
| AC-7 | The concise summary includes objective, preparation status, recommended focus, top prior context, open actions, and risks/questions, with explicit insufficient-context states where content is missing. | E2E eval |
| AC-8 | A reviewer can understand the recommended meeting focus from the concise summary in under 60 seconds using the prepared fixture scenario. | Usability/e2e review eval |
| AC-9 | Every briefing item is traceable to qualifying source memory or upcoming meeting metadata through expand/details source information. | E2E + integration traceability eval |
| AC-10 | Source details show human-readable relatedness reasons for each selected source meeting. | E2E eval |
| AC-11 | Weak-evidence items appear only with clear caveats; insufficient-evidence items are not presented as facts. | Unit + content safety eval |
| AC-12 | Unresolved conflicting memory items are excluded from briefing advice and stakeholder note details. | Unit + integration eval using conflicting-memory fixture |
| AC-13 | Stakeholder notes appear only in expandable details, never in the concise summary, and only when professional, meeting-relevant, evidence-backed, and non-conflicting. | E2E + content safety eval |
| AC-14 | Excluding a selected source meeting persists for the upcoming meeting across future regenerations and does not create a global exclusion rule. | Integration eval |
| AC-15 | Users can undo a source exclusion before regeneration, and excluded sources remain visible in a collapsed excluded-sources area. | E2E eval |
| AC-16 | Each successful generation/regeneration persists a distinct briefing version; the active/latest completed version is shown by default and prior versions remain accessible for history/audit. | Integration eval |
| AC-17 | When source memory changes or is reprocessed, affected briefing versions are marked stale and regeneration is offered. | Integration eval |
| AC-18 | Failed generation shows a safe user-facing failure reason, offers retry/regenerate, and does not remove the latest completed briefing. | E2E + integration eval |
| AC-19 | Generation completes within p95 < 60s for up to 3 qualifying prior meetings in the performance fixture. | Performance eval |
| AC-20 | Latest completed briefing view renders within p95 < 500ms in the performance fixture. | Performance eval |
| AC-21 | Metrics and logs cover generation pipeline health and product usefulness events without raw content, participant PII, source snippets, or generated briefing text. | Observability/security eval |
| AC-22 | Authorization checks prevent unauthorized source meetings from being selected, displayed, logged, or used in generation. | Security integration eval |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Automatic matching selects irrelevant prior meetings | Medium | High | Require at least two explainable signals, limit to top 3, show relatedness reasons, allow meeting-level source exclusions and undo |
| Briefing becomes too long or hard to scan | Medium | High | Use progressive format with concise summary first and expandable details below |
| Weak-evidence content reduces trust | Medium | High | Caveat weak-evidence items, show source details, exclude insufficient/conflicting items from advice |
| Unresolved conflicting memory influences user decisions | Medium | High | Exclude unresolved conflicts from briefing advice and stakeholder notes; rely on BRD-03 conflict status |
| Async generation finishes too late for pre-call use | Medium | Medium | Queue automatically on upcoming meeting creation, show generating state, allow user to leave/return, keep latest completed briefing visible |
| Generation failure blocks meeting preparation | Medium | Medium | Graceful failure state, retry/regenerate, cached/latest completed briefing remains available |
| Stakeholder notes expose sensitive or unsupported information | Medium | High | Keep stakeholder notes out of concise summary, require professional/evidence-backed content, exclude sensitive personal attributes and unsupported inference |
| Source memory reprocessing makes briefings stale | Medium | Medium | Mark affected briefing versions stale and offer regeneration |
| Observability leaks meeting content | Low | High | Ban raw content, snippets, briefing text, participant PII, stakeholder notes, and meeting titles from logs/metric labels; use hashed/opaque IDs only |
| Feature flag mismatch exposes partial functionality | Low | Medium | Use dual namespace flags, server-authoritative enforcement, and flag parity evals |

---

## Relations

- **Parent feature:** ContextPilot meeting intelligence / pre-call preparation loop.
- **Depends on:** BRD-02 Manual Meeting Import for source meeting data foundation; BRD-03 Meeting Memory Processing for source memory, evidence statuses, conflict statuses, briefing-readiness signals, and source memory versions.
- **Related specs:** `specs/curated/brd-02-manual-meeting-import.md`, `specs/curated/brd-03-meeting-memory-processing.md`, `specs/feature-flags.md`, `contextPilot.md`.
- **Blocks:** Future dashboard preparation states, advanced related-meeting detection, continuity/what-changed views, briefing delivery integrations, and provider connector workflows.
- **Future BRDs:** Advanced related meeting detection may still be split into BRD-08 when matching needs to go beyond the MVP balanced signal rule.

---

## Open Questions

The prior draft had open matching/scoping questions. Brainstormer restart disposition:

| Question | Disposition | Resolution |
|----------|-------------|------------|
| Similar title threshold | Resolved for curation | Use the constrained BRD-03 title rule: normalized titles have Levenshtein distance of 3 or less, or share a common token sequence of 4 or more consecutive words. This stays explainable and avoids opaque semantic matching. |
| Close chronology window | Resolved for curation | A source meeting qualifies as close chronology when it completed within 90 days before the upcoming meeting's scheduled start. Future-dated meetings do not qualify as prior context. |
| Ranking weights for top 3 source meetings | Resolved for curation | Use deterministic explainable ordering: highest number of matched signals first, then most recent completed meeting, then stronger title similarity, then stable meeting ID as tie-breaker. Do not expose opaque scores to users. |
| Manual addition of non-matched prior meetings | Deferred to future BRD | Keep BRD-04 MVP limited to exclude/restore of automatically selected sources. Manual addition of non-matched meetings belongs in a future related-meeting detection or correction workflow BRD if user testing shows automatic matching is insufficient. |

No Shakil decision is required before curation unless he wants display-name-only participant matching or manual source addition in BRD-04 MVP.
