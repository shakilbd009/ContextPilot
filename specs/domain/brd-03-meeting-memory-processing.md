# BRD-03 Meeting Memory Processing

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-03` |
| Title | Meeting Memory Processing |
| Author | pm |
| Created | 2026-05-20 |
| Status | Accepted |
| Priority | P1 |
| Phase | Phase 2 |

---

## Overview

Meeting Memory Processing transforms a saved meeting into durable, source-grounded memory that can power future pre-call briefings. It processes transcript text, notes, meeting metadata, participants, and safely matched prior memories into structured categories with evidence, quality states, version history, and briefing-readiness signals.

This feature matters because ContextPilot's core promise depends on trusted memory, not merely generated summaries. Users must be able to verify what the system remembered, understand when evidence is weak or conflicting, and prevent unsupported or unresolved information from flowing into future briefings.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | Authenticated user | My imported meeting to be automatically processed into structured memory | I can get value from a saved meeting without remembering a separate processing step |
| US-2 | Authenticated user | Every memory item to include source evidence | I can verify that ContextPilot is not inventing unsupported facts |
| US-3 | Authenticated user | A complete memory structure even when evidence is weak | I can see which categories are useful and which are insufficient instead of wondering whether data is missing |
| US-4 | Authenticated user | The memory view to show briefing-readiness | I can understand whether this meeting can safely power a future pre-call briefing |
| US-5 | Authenticated user | Prior memories used during processing to be visible and editable | I can remove wrong matches, add missing context, and reprocess with the right inputs |
| US-6 | Authenticated user | Conflicting evidence to appear in a review queue | I can resolve conflicts before they influence future briefings |
| US-7 | Authenticated user | Failed, stale, or retry-exhausted processing to be retryable | I can recover from transient processing problems or source edits |
| US-8 | Product/operator | Observe processing lifecycle, retries, failures, and active version changes | I can diagnose system reliability without exposing sensitive meeting content |

---

## Functional Requirements

### Must Have

- Automatically queue meeting memory processing after a successful manual meeting import from BRD-02 when Meeting Memory Processing is enabled.
- Provide a manual retry/reprocess action for failed, retry-exhausted, stale, or user-corrected processing inputs.
- Keep meeting import non-blocking: manual meeting save and meeting detail navigation must not wait for memory processing completion.
- Process the following inputs when available and authorized: meeting title, completed date/time, participants, transcript text, notes text, and safely matched prior processed meeting memories.
- Preserve a complete memory output structure with these categories for every completed processing result: summary, decisions, action items, risks/blockers, open questions, stakeholder notes, and next recommended focus.
- Mark a category as `Insufficient evidence` when the source material does not support useful content for that category; the system must not fill weak categories with guesses.
- Require strict source evidence for every extracted memory item and every synthesized next recommended focus.
- Preserve evidence as both a supporting snippet and a stable source location. Source location uses character offsets: a JSON object `{ "type": "char_offset", "start": integer, "end": integer }` where start and end are zero-based rune indices into the stored transcript or notes text. This format is committed in ADR-0006 and replaces the "chosen during curation" deferral. Source locations must never appear in logs or metrics.
- Prevent extracted items from being accepted as supported unless their evidence snippets and locations actually support the item.
- Represent action items with description, owner when explicitly stated, due date when explicitly stated, status, source evidence, and quality status.
- Leave action item owner and due date unset or marked unspecified when the source does not explicitly support them.
- Represent decisions as change-aware memory items with decision statement, evidence, and a status indicating standalone, confirms prior decision, changes prior decision, or reverses prior decision.
- Mark a decision as confirming, changing, or reversing a prior decision only when prior memory is eligible input and the current meeting evidence supports that relationship.
- Represent stakeholder notes as relationship-oriented business context with person/participant, preferences, concerns, commitments, influence/stake, evidence, and quality status.
- Limit stakeholder notes to meeting-relevant, source-supported business context; sensitive personal attributes, personality profiling, psychological inference, or unsupported influence assumptions are out of scope.
- Generate next recommended focus by synthesizing across supported memory categories while preserving evidence links to source snippets and/or source-grounded memory items.
- Mark next recommended focus as `Insufficient evidence` when supported memory categories do not provide enough basis for a recommendation.
- Create a new memory version for each processing or reprocessing run.
- Make the latest successful memory version active by default, except that unresolved conflicting evidence remains outside normal memory categories until reviewed.
- Preserve prior successful memory versions for audit, rollback, comparison, and debugging until retention rules are finalized by BRD-06.
- Ensure failed processing runs do not replace the active successful memory version.
- Automatically queue reprocessing when meeting transcript, notes, participants, title, completed date/time, or other source metadata changes after memory has been processed.
- Show stale memory state when active memory is based on an older source version while reprocessing is queued or running.
- Expose the processing lifecycle states: not processed, queued, processing, completed, completed with insufficient evidence, failed, stale, reprocessing, retrying, and retry exhausted.
- Allow safe automatic matching of eligible prior memories using constrained signals such as same participants, similar title, and close chronology.
- Show users which prior memories were used as processing input.
- Allow users to exclude incorrectly matched prior memories, include missing relevant prior memories, and reprocess.
- Keep prior-memory matching narrower than the future Related Meeting Detection BRD; broad opaque matching, provider-based relationship inference, and cross-system relatedness are out of scope.
- Apply quality statuses at both item level and category level: `Strong evidence`, `Weak evidence`, `Insufficient evidence`, and `Conflicting evidence`.
- Mark current/prior memory disagreements as `Conflicting evidence` when both sides have source support but cannot be reconciled automatically.
- Place unresolved conflicting evidence in a separate needs-review queue rather than normal memory categories.
- Exclude unresolved conflicting evidence from downstream briefing input.
- Allow users to resolve a conflict by writing an evidence-backed resolution note that cites current/prior evidence or explicitly states that evidence is insufficient.
- Preserve original conflicting sources and the resolution note for audit.
- Treat resolved conflicts as part of a new reviewed memory version or reviewed state defined during curation.
- Emphasize briefing-readiness in the primary memory view.
- Mark memory as briefing-ready only when summary, decisions, action items, risks/blockers, and open questions have strong or weak evidence and no unresolved conflict blocks those core categories.
- Allow stakeholder notes and next recommended focus to be insufficient without automatically blocking core briefing-readiness.
- Provide full job failure behavior: queued, retrying, failed, retry exhausted, user-safe failure reason, operator diagnostics, and manual retry.
- Retry transient processing failures up to 3 times with exponential backoff before marking retry exhausted.
- Support hybrid degradation when the processing worker or provider is unavailable: queue for short outages, retry according to policy, then show retry-exhausted or unavailable state if recovery does not occur.
- Keep existing active memory available when reprocessing is pending, running, failed, or retry exhausted.
- Remain provider-agnostic: this BRD defines product behavior, evidence requirements, state model, eval expectations, privacy constraints, and observability, not a specific model, vendor, or runtime.
- Require strict unsupported-claim evals with fixtures where tempting but unsupported claims must be omitted or marked insufficient evidence.
- Gate processing behavior behind feature flags with defaults `false`; minimum expected server flag is `FF_ENABLE_MEETING_MEMORY_PROCESSING`.
- Gate browser-visible memory UI, retry/reprocess actions, review queue, and memory state indicators behind a browser-visible flag if those behaviors ship with this BRD.
- Treat transcript text, notes text, evidence snippets, generated memory, stakeholder notes, participant values, prior-memory references, and resolution notes as sensitive meeting-derived data.
- Prevent raw meeting content, evidence snippets, participant PII, stakeholder note content, generated memory content, and resolution note text from being written to logs or metrics.

### Should Have

- Display memory version metadata, including active version, source version, processing status, and whether prior memories were used.
- Display user-safe failure reasons that explain whether the user should retry later, edit source content, or contact support.
- Emit operator diagnostics with safe error classes and correlation IDs for failed, retried, and retry-exhausted processing jobs.
- Show when processing used only the current meeting versus current meeting plus prior memories.
- Make evidence easy to inspect from each memory item without forcing users to read the full transcript or notes.
- Provide clear copy for `Insufficient evidence` categories so users understand that the system intentionally avoided guessing.
- Provide clear copy for briefing-readiness states: ready for briefing, ready with weak/insufficient sections, needs review, processing/reprocessing, failed/retry exhausted, and stale.
- Preserve a safe audit trail of user include/exclude changes to prior-memory inputs without logging sensitive content.
- Allow curation to add quality distribution metrics if needed for trust, evaluation, or operations, provided they are content-safe.

### Could Have

- Show a compact diff between memory versions for user review.
- Provide rollback to a previous successful memory version if a later reprocess is worse.
- Offer richer conflict-resolution workflows in BRD-07 Manual Memory Correction.
- Offer broader related meeting detection in a future Related Meeting Detection BRD.
- Add advanced quality analytics, such as category-level quality trends over time, if curation determines they are operationally necessary and privacy-safe.

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Processing latency | **Provisional: 30 seconds at P95** for a 50,000-character transcript-plus-notes input under normal load, measured from job pick-up to completion. This target is provisional and will be updated to a provider-specific SLO when the processing worker and model runtime are selected. Processing must be asynchronous, observable, and non-blocking for meetings up to the BRD-02 limit. Queue wait time is tracked separately via `cp_meeting_memory_processing_queue_wait_ms`. |
| Import path latency | Successful manual meeting import and redirect must not wait on memory processing completion |
| Availability | Manual import and existing meeting detail access continue when memory processing is unavailable; processing degradation follows queued/retry/retry-exhausted behavior |
| Retry behavior | Up to 3 automatic retries with exponential backoff for transient processing failures, followed by retry-exhausted state and manual retry |
| Evidence integrity | Every supported item must have evidence snippet plus stable source location; unsupported claims must be omitted or marked insufficient evidence |
| Privacy | Raw transcript text, raw notes text, evidence snippets, participant PII, stakeholder note content, generated memory content, and resolution note text must not appear in logs or metric labels |
| Retention | Memory versions and evidence snippets are retained with the source meeting until BRD-06 defines final retention, deletion, export, and redaction policy |
| Briefing safety | Unresolved conflicting evidence must not be eligible for downstream briefing input |
| Accessibility | Memory status, evidence access, retry/reprocess actions, and review queue interactions inherit BRD-01 WCAG 2.1 AA expectations |
| Provider portability | Product behavior and eval contracts must not depend on a specific processing provider, model, or runtime |

---

## Observability

### Metrics to emit

- `cp_meeting_memory_processing_queued_total` — counter for processing attempts queued, labeled by safe trigger type such as `import`, `source_edit`, `manual_retry`, or `prior_memory_input_change`.
- `cp_meeting_memory_processing_started_total` — counter for processing attempts that begin execution.
- `cp_meeting_memory_processing_completed_total` — counter for successful processing attempts, labeled by safe completion state such as `completed` or `completed_with_insufficient_evidence`.
- `cp_meeting_memory_processing_failed_total` — counter for failed processing attempts, labeled by safe failure class and retryability.
- `cp_meeting_memory_processing_retried_total` — counter for automatic retry attempts, labeled by retry number and safe failure class.
- `cp_meeting_memory_processing_retry_exhausted_total` — counter for attempts that exhaust automatic retries.
- `cp_meeting_memory_processing_duration_ms` — histogram for processing attempt duration.
- `cp_meeting_memory_processing_queue_wait_ms` — histogram for time spent queued before processing starts.
- `cp_meeting_memory_active_version_changed_total` — counter for active memory version changes, labeled by safe reason such as `new_successful_run`, `manual_reprocess`, or `conflict_resolution`.
- `cp_meeting_memory_flag_eval_duration_ms` — histogram for server-side feature flag evaluation latency.

Quality distribution metrics are deferred to curation. If added, they must use safe aggregate labels only, such as category and quality status, and must not include meeting content, evidence snippets, participant PII, raw identifiers, or generated memory text.

### Log events

- `meeting_memory.processing.queued` — emitted when a processing attempt is queued; include correlation ID, meeting identifier, safe trigger type, flag state, and source version identifier.
- `meeting_memory.processing.started` — emitted when execution begins; include correlation ID, processing attempt identifier, memory version candidate identifier, and whether prior memories are included.
- `meeting_memory.processing.completed` — emitted when a processing attempt succeeds; include correlation ID, memory version identifier, completion state, active-version change status, and safe category status counts if approved during curation.
- `meeting_memory.processing.failed` — emitted when a processing attempt fails; include correlation ID, safe failure class, retryability, and attempt number.
- `meeting_memory.processing.retrying` — emitted before an automatic retry; include correlation ID, attempt number, safe failure class, and backoff class, not exact sensitive payload details.
- `meeting_memory.processing.retry_exhausted` — emitted when automatic retries are exhausted; include correlation ID, safe failure class, and manual retry availability.
- `meeting_memory.version.activated` — emitted when a successful or reviewed memory version becomes active; include correlation ID, prior active version identifier, new active version identifier, and safe activation reason.
- `meeting_memory.source.stale` — emitted when source meeting edits make active memory stale; include correlation ID, meeting identifier, source version identifier, and safe edited field category.
- `meeting_memory.prior_inputs.updated` — emitted when a user includes or excludes prior memories for reprocessing; include correlation ID and counts only, not prior meeting titles or memory content.
- `meeting_memory.conflict.reviewed` — emitted when a user submits an evidence-backed conflict resolution; include correlation ID, conflict identifier, and resolution status without resolution text.

Raw transcript text, raw notes text, evidence snippets, participant names, participant emails, stakeholder note text, generated memory text, prior memory content, and resolution note text must not be written to logs.

### Health/readiness endpoints

- `GET /ready` — returns 200 for core app readiness only when the app can evaluate the Meeting Memory Processing flag and reach required persistence dependencies; if the processing worker/provider is unavailable but the app can queue jobs and serve existing memory, readiness behavior should be defined during curation to avoid taking the whole app down for a degraded async feature.
- `GET /live` — returns 200 when the process is alive.
- Worker or provider health should be observable separately if architecture curation introduces a separate processing worker, queue, or external provider dependency.

---

## Feature Flag

- **Flag name:** `ff_enable_meeting_memory_processing`
- **Type:** boolean
- **Default:** `false`
- **Minimum server env:** `FF_ENABLE_MEETING_MEMORY_PROCESSING`
- **Browser env if UI/actions ship in this BRD:** `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING`

Required behavior:

| Server env | Browser env | Expected behavior |
|------------|-------------|-------------------|
| `false` | `false` | Processing is not queued or executed; browser-visible memory UI/actions are hidden or read-only unavailable states are shown |
| `true` | `true` | Processing, memory UI, retry/reprocess actions, prior-memory include/exclude, review queue, and briefing-readiness states may be available according to user permissions |
| `true` | `false` | Server may process jobs for controlled testing, but browser-visible actions remain hidden; this state is allowed only for controlled rollout/testing |
| `false` | `true` | Browser may show no-op or unavailable state, but server does not queue or execute processing; this is a misconfiguration and should be surfaced through safe logs/metrics |

Curation may split automatic processing, memory UI, and review/reprocess actions into separate flags if that improves rollout safety. Any split flags must default to `false`, be registered in `specs/feature-flags.md`, and preserve server-side authority for processing and data mutation.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | When Meeting Memory Processing is disabled, successful manual import does not queue or execute memory processing | Feature flag integration or E2E eval verifies no processing job is created when `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` |
| AC-2 | When enabled, successful manual import queues memory processing without blocking the import redirect or meeting detail access | E2E eval imports a meeting, asserts redirect succeeds, and verifies queued processing state appears separately |
| AC-3 | A completed processing result includes summary, decisions, action items, risks/blockers, open questions, stakeholder notes, and next recommended focus sections | Integration eval verifies complete memory structure for a fixture meeting |
| AC-4 | Categories without enough evidence are shown as `Insufficient evidence` rather than omitted or guessed | Fixture eval with sparse notes verifies complete structure and insufficient-evidence category statuses |
| AC-5 | Every supported memory item includes evidence snippet, source type, and stable source location | Schema/integration eval verifies evidence fields on all extracted items |
| AC-6 | Unsupported or tempting claims are omitted or marked `Insufficient evidence` | Strict hallucination eval uses fixtures with tempting unsupported claims and verifies no unsupported supported items are produced |
| AC-7 | Evidence snippets and source locations support the extracted item they are attached to | Evidence-grounding eval checks item/evidence consistency for representative fixtures |
| AC-8 | Action items preserve description, explicitly stated owner, explicitly stated due date, status, evidence, and quality status without inferring missing owner/due date | Fixture eval includes stated and unstated owners/dates and verifies extraction behavior |
| AC-9 | Decisions are marked standalone, confirms, changes, or reverses only when current evidence and eligible prior memory support that status | Fixture eval verifies change-aware decision behavior with and without prior memory input |
| AC-10 | Stakeholder notes are limited to meeting-relevant business context and exclude unsupported personal profiling or sensitive personal attributes | Safety eval verifies stakeholder-note guardrails on representative fixtures |
| AC-11 | Each processing or reprocessing attempt creates a new memory version, and the latest successful version becomes active by default | Integration eval verifies version creation and active-version pointer behavior |
| AC-12 | Failed, retrying, or retry-exhausted runs do not replace the previous active successful memory version | Integration eval simulates failure after an active version exists and verifies active memory remains available |
| AC-13 | Editing source meeting transcript, notes, participants, title, or date/time marks active memory stale and queues reprocessing when enabled | Integration or E2E eval edits source fields and verifies stale/reprocessing state transition |
| AC-14 | User-visible lifecycle states include not processed, queued, processing, completed, completed with insufficient evidence, failed, stale, reprocessing, retrying, and retry exhausted | State-machine eval verifies allowed transitions and user-visible labels |
| AC-15 | Prior memories used for processing are visible to the user | E2E or integration eval verifies used prior-memory references are displayed without exposing unsafe content in logs |
| AC-16 | User can exclude an incorrect prior memory, include a missing relevant prior memory, and reprocess to create a new version | E2E eval exercises include/exclude flow and verifies new version creation |
| AC-17 | Item-level and category-level quality statuses use only `Strong evidence`, `Weak evidence`, `Insufficient evidence`, or `Conflicting evidence` | Schema/unit eval verifies status enum and aggregate category status behavior |
| AC-18 | Conflicting evidence appears in a needs-review queue and is excluded from normal memory categories and downstream briefing input until resolved | E2E/integration eval verifies conflict placement and downstream eligibility flag |
| AC-19 | User can resolve a conflict with an evidence-backed resolution note, preserving original conflicting sources for audit | E2E eval submits conflict resolution with evidence reference and verifies reviewed state/version behavior |
| AC-20 | Briefing-readiness is ready only when summary, decisions, action items, risks/blockers, and open questions have strong or weak evidence and no unresolved conflict blocks those core categories | Readiness eval verifies ready, weak/insufficient, needs-review, stale, processing, and failed cases |
| AC-21 | Transient failures retry up to 3 times with exponential backoff before retry-exhausted state and manual retry availability | Job lifecycle integration eval simulates retryable failure and verifies retry count/state transitions |
| AC-22 | Worker/provider unavailability does not break manual import or existing active memory access | Degradation eval verifies queue/retry/retry-exhausted behavior while existing memory remains viewable |
| AC-23 | Logs and metrics emit for queued, started, completed, failed, retried, retry exhausted, duration, and active version changes without sensitive content | Observability eval checks event/metric names and redaction rules |
| AC-24 | Provider/model selection is not hard-coded into product acceptance behavior | Architecture review verifies BRD/evals define behavior and contracts rather than mandating a specific provider |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Processing invents unsupported memory | High | High | Require strict evidence for every item, insufficient-evidence states, and unsupported-claim eval fixtures |
| Evidence snippets appear trustworthy but do not actually support the item | Medium | High | Require evidence-grounding evals that verify item/evidence consistency |
| Stakeholder notes become sensitive profiling | Medium | High | Limit stakeholder notes to source-supported business context and exclude sensitive attributes, personality profiling, and unsupported inference |
| Safe automatic prior-memory matching chooses the wrong context | Medium | High | Show prior memories used, allow include/exclude and reprocess, keep matching constrained, and defer broad relatedness to a later BRD |
| Conflicting evidence blocks useful briefing output | Medium | Medium | Isolate conflicts in a review queue, allow evidence-backed resolution, and let non-conflicting memory remain active |
| Provider or worker outages prevent timely processing | Medium | Medium | Keep import non-blocking, queue/retry/retry-exhaust, preserve active memory, and expose user-safe retry states |
| Version history grows without clear retention policy | Medium | Medium | Retain with source meeting for now and defer final retention/deletion/export policy to BRD-06 |
| Logs or metrics leak sensitive meeting-derived data | Medium | High | Explicitly ban raw content, evidence snippets, PII, generated memory, and resolution text from logs/metrics; verify through observability evals |
| Architecture-defined latency becomes too vague for implementation | Medium | Medium | Require curation to set processing SLO based on provider/runtime choice before implementation |
| Browser/server flag drift exposes UI without processing support | Medium | Medium | Server flag remains authoritative, browser flag gates visible actions, and misconfiguration emits safe observability |

---

## Relations

- **Parent feature:** Meeting intelligence domain memory foundation.
- **Blocked by:** BRD-02 Manual Meeting Import for saved meetings, transcript/notes preservation, structured participants, and meeting detail route.
- **Blocks:** BRD-04 Pre-Call Briefing because briefings require trusted, source-grounded memory.
- **Related specs:** `specs/domain/brd-02-manual-meeting-import.md`, `specs/curated/brd-02-manual-meeting-import.md`, `specs/feature-flags.md`, `specs/brd-sequencing-roadmap.md`.
- **Future related BRDs:** BRD-06 Privacy & Retention Controls, BRD-07 Manual Memory Correction, Related Meeting Detection, and Meeting Continuity / What Changed Since Last Time.

---

## Open Questions

The following items were raised during review and curation and are resolved or deferred as noted:

| ID | Question | Disposition | Resolution |
|----|----------|-------------|------------|
| OQ-1 | Split feature flag (auto processing / memory UI / review actions) or single dual-namespace flag? | Deferred to PM | Recommendation: single flag for Phase 2 MVP; interface supports split later if rollout requires it. Must confirm before Phase 2 implementation. |
| OQ-2 | Exact processing latency SLO | Partially resolved | **Provisional target: 30 seconds at P95** for 50,000-character input. Provider-specific SLO deferred until runtime selected. Queue wait tracked separately. |
| OQ-3 | Exact source-location format (char offsets, line references, token ranges) | **Resolved** | Character offsets committed. Schema: `{ "type": "char_offset", "start": integer, "end": integer }`. See ADR-0006. |
| OQ-4 | Quality distribution observability for MVP ops? | Deferred to PM | Recommendation: emit as log event field per ADR-0008, not Prometheus metric labels. |
| OQ-5 | Provider/runtime selection | Out of scope | Provider-agnostic interface defined in ADRs; concrete selection is a separate decision. |
