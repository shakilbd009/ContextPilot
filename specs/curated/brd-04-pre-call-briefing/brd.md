# BRD-04 Pre-Call Briefing — Curated

---
status: Curated
brd_id: brd-04
title: Pre-Call Briefing
author: brainstormer
created: 2026-05-23
curated: 2026-05-23
reviewers: spec-writer (t_e859ba50)
adrs: none (PM/OQ decisions incorporated directly)
parent: brd-03-meeting-memory-processing
blocks: future dashboard preparation states, advanced related-meeting detection, continuity/what-changed views, briefing delivery integrations, provider connector workflows
feature_flag: ff_enable_pre_call_briefing
phase: Phase 2

---

## Overview

Pre-Call Briefing helps users prepare for upcoming meetings by generating a concise, traceable briefing from related prior meeting memory. It is the flagship ContextPilot experience: users should quickly understand what happened before, what remains unresolved, what to focus on, and what questions to ask without manually reviewing old notes.

The MVP includes explainable automatic related-meeting selection, source details with relatedness explanations, evidence caveats, source exclusions, and versioned briefing history. Source details, evidence caveats, source exclusions, and versioned history preserve user trust in generated briefings.

---

## User Stories

|| ID | As a | I want | So that |
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

**FR-1: Upcoming Meeting Briefing Entry Point**

The feature provides a pre-call briefing entry point from an upcoming meeting detail page. The briefing entry point is visible only when `ff_enable_pre_call_briefing` is enabled and the user is authorized to view the meeting.

**FR-2: Automatic Async Generation on Upcoming Meeting Creation**

When an upcoming meeting is created and `FF_ENABLE_PRE_CALL_BRIEFING=true`, the system queues an asynchronous briefing generation job without blocking access to the upcoming meeting detail page. The user can leave and return while generation runs.

**FR-3: Manual Regenerate**

Users can manually request regeneration from the upcoming meeting detail page. Regeneration runs asynchronously and does not remove the latest completed briefing while a new version is pending.

**FR-4: Balanced Related-Meeting Matching**

The MVP automatically selects related prior meetings when at least two of the following explainable signals match: same participant, similar title, same organization/client, close chronology. The MVP signal definitions are:

- **Same participant:** At least one participant in the upcoming meeting and prior meeting shares an exact email address. If email is unavailable, this signal does not pass. Display-name-only matching is deferred to advanced related-meeting detection.
- **Similar title:** Normalized titles have Levenshtein distance of 3 or less, or share a common token sequence of 4 or more consecutive words, matching the constrained BRD-03 prior-memory convention.
- **Same organization/client:** At least one participant organization/company value matches exactly after normalization, or the upcoming meeting metadata explicitly names the same organization/client as a prior meeting.
- **Close chronology:** The prior meeting completed within 90 days before the upcoming meeting's scheduled start. Future-dated source meetings must not qualify as prior context.

**FR-5: Source Scope**

Briefing generation uses up to the top 3 highest-ranked qualifying prior meetings. If fewer than 3 prior meetings qualify, generation uses the available qualifying meetings. If none qualify, the system generates a no-prior-memory preparation shell.

Ordering among qualifying meetings is deterministic and explainable: highest number of matched signals first, then most recent completed meeting, then stronger title similarity, then stable meeting ID as tie-breaker. Opaque internal scores are not exposed to users.

**FR-6: Relatedness Explanation**

Each selected source meeting has visible relatedness reasons available behind an expand/details affordance, such as matched participant, similar title, same organization/client, or close chronology. Explanations are human-readable and do not expose implementation scoring internals.

**FR-7: Progressive Briefing Format**

The briefing uses a progressive format: a concise summary first, followed by expandable detailed sections. The concise summary remains readable without inline source clutter.

**FR-8: Concise Summary Contents**

The concise summary always includes: meeting objective, preparation status, recommended focus, top prior context, open actions, and risks/questions. If prior memory is weak or missing, affected sections show explicit insufficient-context states instead of invented content.

**FR-9: Detailed Sections**

Expandable detailed sections include: previous relevant context, important prior decisions, open action items and commitments, unresolved risks/blockers, open questions, stakeholder notes (when allowed), suggested questions, suggested agenda, source details, and source meeting list.

**FR-10: No-Prior-Memory Preparation Shell**

When no related prior memory is found, the system generates a limited shell from upcoming meeting title/description/metadata only, clearly marked `No prior memory found`. The shell may include objective, recommended focus, and suggested prep questions derived from upcoming meeting metadata. The shell must not imply prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks.

**FR-11: Evidence Policy**

The briefing follows BRD-03 evidence status conventions: `strong_evidence`, `weak_evidence`, `insufficient_evidence`, and `conflicting_evidence`. Strong-evidence items may appear normally. Weak-evidence items may appear with clear caveats. Insufficient-evidence items must not be presented as facts. Unresolved conflicting items are excluded from briefing advice.

**FR-12: Source Annotations Behind Details**

Every briefing item derived from prior memory is traceable to source memory and source meeting details behind an expand/details affordance. The concise summary remains readable without inline source clutter.

**FR-13: Stakeholder Notes Handling**

Stakeholder notes are allowed only in expandable details, never in the concise summary. They must be professional, meeting-relevant, evidence-backed, and caveated when weak. Sensitive personal attributes, unsupported personality inference, private/irrelevant details, and unresolved conflicting stakeholder notes are excluded.

**FR-14: Versioned Briefing Records**

Every generated or regenerated briefing is persisted as a distinct version. The active/latest completed version is shown by default. Prior versions remain available for audit/history.

**FR-15: Stale Briefing Behavior**

If source meeting memory is reprocessed or changes after a briefing is generated, affected briefing versions are marked stale and regeneration is offered. Stale briefings remain readable with a visible stale indicator until replaced or dismissed.

**FR-16: Source Exclusion**

Users can exclude an automatically selected source meeting before regeneration. Exclusions persist at the upcoming-meeting level and apply to future regenerations for that upcoming meeting only, not as a global exclusion rule.

**FR-17: Exclusion Undo and Visibility**

Users can undo an upcoming-meeting-level source exclusion before regeneration. Excluded source meetings remain visible in a collapsed excluded-sources area with their original relatedness reasons.

**FR-18: Failure and Retry**

Failed generation shows a safe user-facing failure reason, keeps the latest completed briefing available when one exists, and offers retry/regenerate. Operator diagnostics must not expose raw meeting content or participant PII.

**FR-19: Authorization Boundary**

Briefings and source details are visible only to users authorized to view the upcoming meeting and all included source meetings. Unauthorized source meetings must not be selected, displayed, logged, or used in generation.

### Should Have

**FR-20: Preparation Status States**

The briefing displays preparation status values: `generating`, `ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, `regenerating`.

**FR-21: Cached Briefing View**

Viewing the latest completed briefing uses persisted/cached briefing data rather than forcing generation on every page view.

**FR-22: Low-Cardinality Observability**

Metrics and logs use controlled enum labels for status, failure class, source count bucket, and generation trigger. High-cardinality content (raw titles, PII, snippets, generated text) is forbidden from metric labels.

**FR-23: Product Usefulness Events**

The system emits product-usefulness events for: viewing a briefing, expanding details/source panels, requesting regeneration, excluding/restoring a source, and generating a no-prior-memory shell.

### Could Have

**FR-25: Manual Addition of Non-Matched Meetings**

A future enhancement may allow users to manually add prior meetings that were not automatically matched. This is deferred to a future related-meeting detection or correction workflow BRD if user testing shows automatic matching is insufficient.

**FR-26: Advanced Relatedness Detection**

A future BRD may define recurring-series detection, topic embeddings, project graph matching, or learned ranking.

**FR-27: Delivery Channels**

A future BRD may send briefings through email, calendar, Teams, Slack, browser notifications, or mobile push.

**FR-28: Hard Briefing Length Controls**

A future enhancement may define strict character, section, or reading-time limits beyond the MVP progressive structure.

---

## API Contract

All briefing endpoints require an authenticated session. A user may only access briefings for upcoming meetings they are authorized to view and for source meetings they are authorized to access.

|| Method | Path | Summary |
|--------|------|---------|
| GET | `/upcoming/{meetingId}/briefing` | Active/latest briefing version for an upcoming meeting |
| GET | `/upcoming/{meetingId}/briefing/versions` | Version history list |
| GET | `/upcoming/{meetingId}/briefing/versions/{versionNumber}` | Specific briefing version |
| POST | `/upcoming/{meetingId}/briefing/regenerate` | Manual regeneration request |
| POST | `/upcoming/{meetingId}/briefing/sources/{sourceId}/exclude` | Exclude a source meeting |
| POST | `/upcoming/{meetingId}/briefing/sources/{sourceId}/restore` | Restore an excluded source meeting |

---

## Data Model

### briefing_versions

|| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `upcoming_meeting_id` | UUID FK | Yes | References `upcoming_meetings(id)` ON DELETE CASCADE |
| `version_number` | integer | Yes | Monotonically increasing per upcoming meeting |
| `status` | text | Yes | One of: `active`, `superseded` |
| `is_active` | boolean | Yes | Exactly one active version per upcoming meeting |
| `result` | text | Yes | One of: `ready`, `ready_with_caveats`, `no_prior_memory`, `failed` |
| `preparation_status` | text | Yes | One of: `generating`, `ready`, `ready_with_caveats`, `no_prior_memory`, `stale`, `failed`, `regenerating` |
| `content` | JSONB | Yes | Full structured briefing output (see Briefing JSON Content Schema below) |
| `source_count` | integer | Yes | Number of qualifying source meetings used |
| `created_at` | timestamptz | Yes | Default now() |
| `trigger_type` | text | Yes | One of: `auto`, `manual_regenerate` |

UNIQUE constraint on `(upcoming_meeting_id, version_number)`. Index on `(upcoming_meeting_id, is_active)` WHERE `is_active = TRUE`.

### briefing_source_exclusions

| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `upcoming_meeting_id` | UUID FK | Yes | References `upcoming_meetings(id)` ON DELETE CASCADE |
| `excluded_source_meeting_id` | UUID FK | Yes | References `meetings(id)` ON DELETE CASCADE |
| `created_at` | timestamptz | Yes | Default now() |
| `restored_at` | timestamptz | No | NULL = active exclusion; set = restored by user (soft delete per ADR-0012) |

UNIQUE constraint on `(upcoming_meeting_id, excluded_source_meeting_id)`. Active exclusions queried via `WHERE upcoming_meeting_id = $id AND restored_at IS NULL`. Rows with `restored_at IS NOT NULL` are visible in the collapsed excluded-sources area per FR-17.

### Briefing JSON Content Schema

The `content` JSONB column stores the structured briefing output per version:

```json
{
  "concise_summary": {
    "objective": "string",
    "preparation_status": "string",
    "recommended_focus": "string",
    "top_prior_context": "string",
    "open_actions": "string",
    "risks_questions": "string"
  },
  "detailed_sections": {
    "previous_relevant_context": { "statement": "string", "quality_status": "string" },
    "important_prior_decisions": { "items": [], "quality_status": "string" },
    "open_action_items": { "items": [], "quality_status": "string" },
    "unresolved_risks_blockers": { "items": [], "quality_status": "string" },
    "open_questions": { "items": [], "quality_status": "string" },
    "stakeholder_notes": { "items": [], "quality_status": "string" },
    "suggested_questions": { "items": [] },
    "suggested_agenda": { "items": [] }
  },
  "sources": [
    {
      "source_meeting_id": "uuid",
      "relatedness_reasons": ["string"],
      "quality_status": "string"
    }
  ],
  "no_prior_memory_shell": {
    "generated": "boolean",
    "objective": "string",
    "recommended_focus": "string",
    "suggested_prep_questions": ["string"]
  }
}
```

Quality status values in the briefing content schema use the BRD-03 conventions: `strong_evidence`, `weak_evidence`, `insufficient_evidence`, `conflicting_evidence`.

---

## Non-Functional Requirements

|| Requirement | Target | Notes |
|-------------|--------|-------|
| Generation latency | Async briefing generation completes within P95 < 60s for up to 3 qualifying prior meetings under normal operating conditions | Measured from job start to terminal success/failure; excludes queue wait time |
| Cached view latency | Latest completed briefing view renders within P95 < 500ms excluding network variability outside the app | Measured client-visible render time |
| Availability | Briefing pages follow app-baseline availability; generation failures degrade gracefully and do not take down upcoming meeting detail pages | Cached/latest completed briefing remains available during generation failures |
| Regeneration continuity | Latest completed briefing remains readable while regeneration is queued/running/failed | No user-facing downtime during regeneration |
| Traceability | 100% of briefing items are traceable either to qualifying source memory or upcoming meeting metadata | Through expand/details source information |
| Evidence safety | 0 unresolved conflicting memory items appear as briefing advice; weak-evidence items are visibly caveated | Per BRD-03 evidence status conventions |
| Privacy | Logs and metric labels contain no raw transcript text, notes text, briefing text, source snippets, participant PII, stakeholder note content, or private meeting content | Hashed or opaque identifiers only |
| Authorization | Briefing generation and viewing use only meetings and memory the authenticated user is authorized to access | Enforced at the meeting ACL level |
| Accessibility | Briefing UI, generating/failure/stale states, and expand/details controls meet WCAG 2.1 AA expectations | Specific criteria must be verified against BRD-01 component inventory |
| Scale | MVP supports generation from up to 3 qualifying source meetings per upcoming meeting and preserves source exclusion state per upcoming meeting | Source exclusion state is upcoming-meeting-scoped, not global |
| Retention | Briefing versions inherit retention/deletion behavior from the associated upcoming meeting until BRD-06 defines final privacy and retention controls | Deferred to BRD-06 |
| Normal operating concurrency | MVP targets up to 10 concurrent generation jobs and a queue depth of up to 50 pending jobs before degradation begins; latency targets apply under these load conditions | P95 generation latency < 60s measured at job start; queue wait time is excluded |
| Cross-participant briefing sharing | Not in scope for MVP; briefings are accessible only to the owner of the upcoming meeting as defined by ADR-0009 | Cross-participant access deferred to future BRD unless an existing approved BRD specifies otherwise |

---

## Observability

All metrics and log events use low-cardinality enum labels. High-cardinality content (raw titles, PII, snippets, generated text) is forbidden from metric labels and log event fields.

### Metrics

| Metric name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `cp_briefing_generation_duration_ms` | Histogram | `status`, `failure_class`, `source_count_bucket`, `trigger_type` | End-to-end generation time from job start to terminal state; excludes queue wait |
| `cp_briefing_queue_saturation` | Gauge | `queue_name` | Current queue depth as percentage of normal-operating capacity |

`source_count_bucket` values: `0`, `1`, `2`, `3` (max). `failure_class` values: `none`, `upstream_error`, `timeout`, `internal_error`. `trigger_type` values: `auto`, `manual_regenerate`. `status` values: `success`, `failure`.

### Log Events

| Event name | Trigger | Fields (low-cardinality only) |
|------------|---------|-------------------------------|
| `briefing.generation.started` | Async job queued | `upcoming_meeting_id`, `trigger_type`, `source_count_expected` |
| `briefing.generation.completed` | Job reached terminal state | `upcoming_meeting_id`, `status`, `version_number`, `source_count_used`, `duration_ms` |
| `briefing.generation.failed` | Job reached failure terminal state | `upcoming_meeting_id`, `failure_class`, `is_retry` |
| `briefing.viewed` | User loads briefing | `upcoming_meeting_id`, `version_number`, `has_prior_memory` |
| `briefing.source.excluded` | User excludes a source | `upcoming_meeting_id`, `source_meeting_id` |
| `briefing.source.restored` | User restores an exclusion | `upcoming_meeting_id`, `source_meeting_id` |
| `briefing.regenerate.requested` | User requests manual regeneration | `upcoming_meeting_id`, `source_count` |
| `briefing.no_prior_memory_shell.generated` | Shell generated for upcoming meeting with no qualifying sources | `upcoming_meeting_id` |
| `briefing.stale.detected` | Read-time staleness derived for a briefing version | `upcoming_meeting_id`, `version_number`, `stale_source_count` |
| `briefing.flag_misconfiguration` | Server and browser flag mismatch detected | `server_flag_value`, `browser_flag_value`, `endpoint` |

No log event may include raw transcript text, notes text, briefing text, source snippets, participant PII, stakeholder note content, or private meeting content.

---

## Accessibility & State Lifecycle

### ARIA Live Region Expectations

The briefing UI uses ARIA live regions to communicate dynamic states to screen readers. The following states require explicit live region behavior:

| State | ARIA role | aria-live | aria-atomic | Content expectations |
|-------|-----------|-----------|-------------|----------------------|
| `generating` | `status` | `polite` | `true` | Announces "Briefing generation in progress" on entry; updates on progress milestones if available |
| `stale` | `status` | `polite` | `true` | Announces "Briefing may be outdated; regeneration available" when staleness is first computed and displayed |
| `failed` | `alert` | `assertive` | `true` | Announces failure reason immediately on error; retry controls are focusable |
| `regenerating` | `status` | `polite` | `true` | Announces "Regeneration in progress" |

Focus management: On transition to `failed`, focus moves to the failure message region or the retry affordance. On transition to `stale`, the stale indicator is placed in the reading order but does not interrupt current focus.

WCAG 2.1 AA compliance for briefing UI requires these live region behaviors. Verification against BRD-01 component inventory is required before implementation.

### Safe Failure Reason Semantics (FR-18)

A "safe" user-facing failure reason satisfies the following constraints:

1. **No raw content exposure** — failure reasons must not include meeting titles, participant names, transcript snippets, notes content, or generated briefing text.
2. **No PII exposure** — failure reasons must not include email addresses, phone numbers, or other participant attributes.
3. **Actionable** — the user is told what happened in terms of the feature's behavior (e.g., "Briefing generation failed — please try again") rather than technical internals (e.g., "null pointer in pipeline").
4. **Operator-diagnostic correlation** — a non-interactive operator log or error tracking system carries the detailed technical diagnosis (stack trace, pipeline stage, upstream service response) linked to the safe reason via a correlation ID; this diagnostic channel is not exposed to the user.

Safe failure reason examples:
- "Briefing generation failed — please try again"
- "Briefing could not be generated at this time — your previous briefing is still available"
- "Generation timed out — please try again or contact support if this persists"

Unsafe failure reason examples (prohibited):
- "Failed because meeting 'Q3 Budget Review' had no memory"
- "Failed due to upstream timeout on participant alice@example.com"
- "Pipeline error: memory service returned 500"

### Staleness Lifecycle

Staleness is derived at read time per ADR-0011. The following governs when and how staleness first appears in the UI:

1. **First appearance:** Staleness is computed when a user loads the briefing view page (`GET /upcoming/{meetingId}/briefing`). The computation compares the `source_memory_version_ids` captured in the briefing at generation time against the current source memory versions. If any source has a newer version, the briefing version is marked stale.
2. **Stale indicator visibility:** The stale indicator appears on the briefing card/header immediately upon page render, without requiring user action.
3. **Regeneration offered:** When a stale briefing is displayed, a regeneration affordance is prominently visible. The user is not blocked from viewing the stale content.
4. **Source-memory race condition resolution:** If the user clicks Regenerate while source memory is being reprocessed:
   - The regeneration job reads the source memory state at job start time (not at trigger time).
   - If a source memory version changes between trigger and start, the newer version is used.
   - The resulting briefing version may or may not be stale depending on whether the reprocessing completed before the job read the sources.
   - Staleness is re-derived on the next read, not immediately after regeneration completion.

### preparation_status to briefing_versions Mapping

The runtime `preparation_status` values map to `briefing_versions` fields as follows:

| preparation_status | briefing_versions row exists | status | result | Notes |
|--------------------|------------------------------|--------|--------|-------|
| `generating` | Yes (pending) | `active` | — | Row created when job is queued; `result` is unset until terminal |
| `regenerating` | Yes (pending) | `active` | — | New version row created; previous `active` row remains until new version completes |
| `ready` | Yes | `active` | `ready` | Terminal success |
| `ready_with_caveats` | Yes | `active` | `ready_with_caveats` | Terminal success with evidence caveats |
| `no_prior_memory` | Yes | `active` | `no_prior_memory` | Shell generated; terminal |
| `failed` | Yes | `superseded` | `failed` | Row created with `failed` result; replaces previous `active` row if one existed |
| `stale` | Yes | `active` | unchanged | Staleness is a computed property at read time, not a status field mutation |

A failed generation always creates a `briefing_versions` row with `status=superseded` and `result=failed`. The previous `active` briefing (if any) is left readable as a superseded version. The user can access failed versions via the version history endpoint.

---

## Feature Flag

- **Flag name:** `ff_enable_pre_call_briefing`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_PRE_CALL_BRIEFING`
- **Browser env:** `VITE_FF_ENABLE_PRE_CALL_BRIEFING`
- **Scope:** Backend generation/API behavior and frontend briefing UI/actions

Dual namespace registration is confirmed in `specs/feature-flags.md` (Phase 2, Planned).

|| Server env | Browser env | Expected behavior |
|------------|-------------|-------------------|
| `false` | `false` | Briefing entry points are hidden; backend rejects generation/regeneration requests with safe feature-disabled response |
| `true` | `true` | Authenticated users can access briefing entry points and trigger generation |
| `true` | `false` | Server accepts authorized requests; browser UI hidden; valid for server-side testing |
| `false` | `true` | Server denies requests; misconfiguration state emits metric and log event |

The server flag is authoritative for all data mutations.

---

## Acceptance Criteria

|| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | With both flags `false`, briefing UI entry points are hidden and backend generation/regeneration requests are rejected with a safe feature-disabled response | E2E + integration flag eval |
| AC-2 | Creating an upcoming meeting with the flag enabled queues an async briefing generation job without blocking access to the upcoming meeting detail page | Integration eval |
| AC-3 | Manual regenerate queues a new async job while the latest completed briefing remains viewable | E2E + integration eval |
| AC-4 | Automatic related-meeting matching selects only prior meetings with at least two matching signals among same participant, similar title, same organization/client, and close chronology | Unit + integration eval with fixture meetings |
| AC-5 | Briefing generation uses no more than the top 3 qualifying prior meetings and uses fewer when fewer qualify | Integration eval |
| AC-6 | If no qualifying prior memory exists, the system generates a clearly marked no-prior-memory preparation shell without implying prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks | E2E + content safety eval |
| AC-7 | The concise summary includes objective, preparation status, recommended focus, top prior context, open actions, and risks/questions, with explicit insufficient-context states where content is missing | E2E eval |
| AC-8 | A reviewer can understand the recommended meeting focus from the concise summary in under 60 seconds using the prepared fixture scenario | Usability/e2e review eval |
| AC-9 | Every briefing item is traceable to qualifying source memory or upcoming meeting metadata through expand/details source information | E2E + integration traceability eval |
| AC-10 | Source details show human-readable relatedness reasons for each selected source meeting | E2E eval |
| AC-11 | Weak-evidence items appear only with clear caveats; insufficient-evidence items are not presented as facts | Unit + content safety eval |
| AC-12 | Unresolved conflicting memory items are excluded from briefing advice and stakeholder note details | Unit + integration eval using conflicting-memory fixture |
| AC-13 | Stakeholder notes appear only in expandable details, never in the concise summary, and only when professional, meeting-relevant, evidence-backed, and non-conflicting | E2E + content safety eval |
| AC-14 | Excluding a selected source meeting persists for the upcoming meeting across future regenerations and does not create a global exclusion rule | Integration eval |
| AC-15 | Users can undo a source exclusion before regeneration, and excluded sources remain visible in a collapsed excluded-sources area | E2E eval |
| AC-16 | Each successful generation/regeneration persists a distinct briefing version; the active/latest completed version is shown by default and prior versions remain accessible for history/audit | Integration eval |
| AC-17 | When source memory changes or is reprocessed, affected briefing versions are marked stale and regeneration is offered | Integration eval |
| AC-18 | Failed generation shows a safe user-facing failure reason, offers retry/regenerate, and does not remove the latest completed briefing | E2E + integration eval |
| AC-19 | Generation completes within P95 < 60s for up to 3 qualifying prior meetings in the performance fixture | Performance eval |
| AC-20 | Latest completed briefing view renders within P95 < 500ms in the performance fixture | Performance eval |
| AC-21 | Metrics and logs cover generation pipeline health and product usefulness events without raw content, participant PII, source snippets, or generated briefing text | Observability/security eval |
| AC-22 | Authorization checks prevent unauthorized source meetings from being selected, displayed, logged, or used in generation | Security integration eval |

---

## Risks & Mitigations

|| Risk | Likelihood | Impact | Mitigation |
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

## Non-Goals

The following are explicitly out of scope for BRD-04 MVP:

- **Calendar, email, or conferencing provider integrations for upcoming meeting creation** — BRD-04 MVP creates briefings only for manually created upcoming meetings; provider integrations are deferred to future BRDs
- **Manual addition of non-matched prior meetings** — deferred to a future related-meeting detection or correction workflow BRD if user testing shows automatic matching is insufficient
- **Broad opaque matching (embedding similarity, topic embeddings, project graphs, learned ranking)** — deferred to a future advanced related-meeting detection BRD
- **Delivery channels (email, calendar, Teams, Slack, browser notifications, mobile push)** — deferred to a future delivery integrations BRD
- **Strict character, section, or reading-time limits** — future enhancement if MVP progressive format proves insufficient for length management
- **Global source exclusion rules** — source exclusions are upcoming-meeting-scoped only
- **Provider-specific or model-specific behavior** — BRD is provider-agnostic; concrete processing implementation is injected

---

## Relations

- **Parent feature:** ContextPilot meeting intelligence / pre-call preparation loop
- **Depends on:** BRD-02 Manual Meeting Import for source meeting data foundation; BRD-03 Meeting Memory Processing for source memory, evidence statuses, conflict statuses, briefing-readiness signals, and source memory versions
- **Related specs:** `specs/curated/brd-02-manual-meeting-import.md`, `specs/curated/brd-03-meeting-memory-processing.md`, `specs/feature-flags.md`, `contextPilot.md`
- **Blocks:** Future dashboard preparation states, advanced related-meeting detection, continuity/what-changed views, briefing delivery integrations, provider connector workflows
- **Future BRDs:** Advanced related meeting detection may still be split into BRD-08 when matching needs to go beyond the MVP balanced signal rule

---

## Pre-Implementation Approval Gate

**Implementation tasks derived from this BRD are blocked pending written approval from Shakil.**

The following must be confirmed before any implementation work begins:

1. OQ-1 resolved: no-prior-memory shell minimum bar confirmed as objective + recommended focus + suggested prep questions from upcoming meeting metadata only, no implied continuity
2. OQ-2 resolved: multi-meeting briefing support confirmed with up to 3 qualifying sources
3. OQ-3 resolved: max selectable sources limit confirmed as 3
4. OQ-4 resolved: briefing max length/section cap managed through progressive format (concise summary + expandable details); hard limits deferred to future BRD
5. OQ-5 resolved: minimum quality threshold for whole-briefing low-confidence signal derived from BRD-03 `ready_with_caveats` when any category has weak evidence or insufficient evidence
6. OQ-6 resolved: upcoming meeting creation UX queues async generation job on creation, non-blocking
7. OQ-7 resolved: briefing versions are immutable once created; on-demand regeneration creates a new version; staleness is detected when source memory changes