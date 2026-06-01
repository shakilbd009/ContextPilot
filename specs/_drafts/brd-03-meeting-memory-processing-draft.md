# BRD-03 Meeting Memory Processing — Curated Draft

---
status: Curated (Draft)
brd_id: brd-03
title: Meeting Memory Processing
author: pm
created: 2026-05-20
curated: 2026-05-21
reviewers: (pending architect review)
adrs: (to be produced by architect)
parent: brd-02-manual-meeting-import
blocks: brd-04-pre-call-briefing, brd-06-privacy-retention, brd-07-manual-memory-correction
feature_flag: ff_enable_meeting_memory_processing
phase: Phase 2

---

## Overview

Meeting Memory Processing transforms a saved meeting into durable, source-grounded memory that can power future pre-call briefings. It processes transcript text, notes, meeting metadata, participants, and safely matched prior memories into structured categories with evidence, quality states, version history, and briefing-readiness signals.

This feature matters because ContextPilot's core promise depends on trusted memory, not merely generated summaries. Users must be able to verify what the system remembered, understand when evidence is weak or conflicting, and prevent unsupported or unresolved information from flowing into future briefings.

---

## User Stories

| ID | As a... | I want to... | So that... |
|---|---|---|---|
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

**FR-1: Automatic Processing Queue**

Automatically queue meeting memory processing after a successful manual meeting import from BRD-02 when Meeting Memory Processing is enabled. Keep meeting import non-blocking: manual meeting save and meeting detail navigation must not wait for memory processing completion.

**FR-2: Memory Categories**

Preserve a complete memory output structure with these categories for every completed processing result: `summary`, `decisions`, `action items`, `risks/blockers`, `open questions`, `stakeholder notes`, and `next recommended focus`.

**FR-3: Insufficient Evidence Handling**

Mark a category as `Insufficient evidence` when the source material does not support useful content for that category; the system must not fill weak categories with guesses.

**FR-4: Evidence Requirements**

Require strict source evidence for every extracted memory item and every synthesized next recommended focus. Preserve evidence as both a supporting snippet and a stable source location, including source type (`transcript`, `notes`, or prior memory reference) and a source position such as character offset, line reference, or equivalent stable locator chosen during curation. Prevent extracted items from being accepted as supported unless their evidence snippets and locations actually support the item.

**FR-5: Action Items**

Represent action items with `description`, `owner` when explicitly stated, `due date` when explicitly stated, `status`, `source evidence`, and `quality status`. Leave action item owner and due date unset or marked unspecified when the source does not explicitly support them.

**FR-6: Decision Tracking**

Represent decisions as change-aware memory items with `decision statement`, `evidence`, and a `status` indicating `standalone`, `confirms prior decision`, `changes prior decision`, or `reverses prior decision`. Mark a decision as confirming, changing, or reversing a prior decision only when prior memory is eligible input and the current meeting evidence supports that relationship.

**FR-7: Stakeholder Notes**

Represent stakeholder notes as relationship-oriented business context with `person/participant`, `preferences`, `concerns`, `commitments`, `influence/stake`, `evidence`, and `quality status`. Limit stakeholder notes to meeting-relevant, source-supported business context. Sensitive personal attributes, personality profiling, psychological inference, and unsupported influence assumptions are out of scope.

**FR-8: Next Recommended Focus**

Generate next recommended focus by synthesizing across supported memory categories while preserving evidence links to source snippets and/or source-grounded memory items. Mark next recommended focus as `Insufficient evidence` when supported memory categories do not provide enough basis for a recommendation.

**FR-9: Version Management**

Create a new memory version for each processing or reprocessing run. Make the latest successful memory version active by default, except that unresolved conflicting evidence remains outside normal memory categories until reviewed. Preserve prior successful memory versions for audit, rollback, comparison, and debugging until retention rules are finalized by BRD-06. Failed processing runs do not replace the active successful memory version.

**FR-10: Stale Detection and Reprocessing**

Automatically queue reprocessing when meeting transcript, notes, participants, title, completed date/time, or other source metadata changes after memory has been processed. Show stale memory state when active memory is based on an older source version while reprocessing is queued or running.

**FR-11: Processing Lifecycle States**

Expose the processing lifecycle states: `not processed`, `queued`, `processing`, `completed`, `completed with insufficient evidence`, `failed`, `stale`, `reprocessing`, `retrying`, and `retry exhausted`.

**FR-12: Prior Memory Matching**

Allow safe automatic matching of eligible prior memories using constrained signals such as same participants, similar title, and close chronology. Show users which prior memories were used as processing input. Allow users to exclude incorrectly matched prior memories, include missing relevant prior memories, and reprocess. Keep prior-memory matching narrower than the future Related Meeting Detection BRD; broad opaque matching, provider-based relationship inference, and cross-system relatedness are out of scope.

**FR-13: Quality Statuses**

Apply quality statuses at both item level and category level: `Strong evidence`, `Weak evidence`, `Insufficient evidence`, and `Conflicting evidence`. Mark current/prior memory disagreements as `Conflicting evidence` when both sides have source support but cannot be reconciled automatically. Place unresolved conflicting evidence in a separate needs-review queue rather than normal memory categories. Exclude unresolved conflicting evidence from downstream briefing input.

**FR-14: Conflict Resolution**

Allow users to resolve a conflict by writing an evidence-backed resolution note that cites current/prior evidence or explicitly states that evidence is insufficient. Preserve original conflicting sources and the resolution note for audit. Treat resolved conflicts as part of a new reviewed memory version.

**FR-15: Briefing Readiness**

Emphasize briefing-readiness in the primary memory view. Mark memory as briefing-ready only when summary, decisions, action items, risks/blockers, and open questions have strong or weak evidence and no unresolved conflict blocks those core categories. Stakeholder notes and next recommended focus may be insufficient without automatically blocking core briefing-readiness.

**FR-16: Failure Behavior**

Provide full job failure behavior: queued, retrying, failed, retry exhausted, user-safe failure reason, operator diagnostics, and manual retry. Retry transient processing failures up to 3 times with exponential backoff before marking retry exhausted. Support hybrid degradation when the processing worker or provider is unavailable: queue for short outages, retry according to policy, then show retry-exhausted or unavailable state if recovery does not occur. Keep existing active memory available when reprocessing is pending, running, failed, or retry exhausted.

**FR-17: Provider Agnosticism**

Remain provider-agnostic: this BRD defines product behavior, evidence requirements, state model, eval expectations, privacy constraints, and observability, not a specific model, vendor, or runtime.

**FR-18: Strict Evidence Eval**

Require strict unsupported-claim evals with fixtures where tempting but unsupported claims must be omitted or marked insufficient evidence.

**FR-19: Feature Flags**

Gate processing behavior behind feature flags with defaults `false`; minimum expected server flag is `FF_ENABLE_MEETING_MEMORY_PROCESSING`. Gate browser-visible memory UI, retry/reprocess actions, review queue, and memory state indicators behind a browser-visible flag if those behaviors ship with this BRD.

**FR-20: Privacy**

Treat transcript text, notes text, evidence snippets, generated memory, stakeholder notes, participant values, prior-memory references, and resolution notes as sensitive meeting-derived data. Prevent raw meeting content, evidence snippets, participant PII, stakeholder note content, generated memory content, and resolution note text from being written to logs or metrics.

**FR-21: Manual Retry**

Provide a manual retry/reprocess action for failed, retry-exhausted, stale, or user-corrected processing inputs.

---

## Non-Functional Requirements

| Type | Requirement | Measurement |
|---|---|---|
| **Processing latency** | Asynchronous, observable, non-blocking for meetings up to the BRD-02 50,000 character transcript-plus-notes limit; SLO defined during curation after provider/runtime selection | TBD during architect review |
| **Import path latency** | Successful manual meeting import and redirect must not wait on memory processing completion | E2E eval: import → redirect completes before processing done |
| **Availability** | Manual import and existing meeting detail access continue when memory processing is unavailable; processing degradation follows queued/retry/retry-exhausted behavior | Degradation eval |
| **Retry behavior** | Up to 3 automatic retries with exponential backoff for transient processing failures, followed by retry-exhausted state and manual retry | Job lifecycle integration eval |
| **Evidence integrity** | Every supported item must have evidence snippet plus stable source location; unsupported claims must be omitted or marked insufficient evidence | Strict hallucination eval |
| **Privacy** | Raw transcript text, raw notes text, evidence snippets, participant PII, stakeholder note content, generated memory content, and resolution note text must not appear in logs or metric labels | Observability eval |
| **Retention** | Memory versions and evidence snippets are retained with the source meeting until BRD-06 defines final retention, deletion, export, and redaction policy | Deferred to BRD-06 |
| **Briefing safety** | Unresolved conflicting evidence must not be eligible for downstream briefing input | E2E eval |
| **Accessibility** | Memory status, evidence access, retry/reprocess actions, and review queue interactions inherit BRD-01 WCAG 2.1 AA expectations | Accessibility eval |
| **Provider portability** | Product behavior and eval contracts must not depend on a specific processing provider, model, or runtime | Architecture review |

---

## Non-Goals

The following are explicitly out of scope for BRD-03:

- **Future meeting creation** — handled outside manual import; belongs to BRD-04 Pre-Call Briefing
- **Global contacts, participant deduplication, CRM-style relationship history, and identity resolution** — deferred to future BRDs
- **Broad opaque matching for prior-memory matching** — deferred to future Related Meeting Detection BRD
- **Provider-based relationship inference and cross-system relatedness** — deferred to future BRDs
- **Sensitive personal attributes, personality profiling, psychological inference, or unsupported influence assumptions in stakeholder notes** — excluded by FR-7
- **Specific model, vendor, or runtime** — BRD is provider-agnostic per FR-17

---

## User Story Acceptance Criteria (Given/When/Then)

### US-1: Automatic Processing

**Given** a user has successfully imported a meeting via BRD-02 with transcript or notes content present,
**When** `FF_ENABLE_MEETING_MEMORY_PROCESSING` is `true`,
**Then** the system queues a memory processing job without blocking the import redirect response,
**And** the meeting detail page shows `queued` state for memory processing.

**Given** a user has successfully imported a meeting with `FF_ENABLE_MEETING_MEMORY_PROCESSING` set to `false`,
**When** the import completes,
**Then** no memory processing job is queued,
**And** the meeting remains in `not processed` state.

---

### US-2: Evidence on Every Memory Item

**Given** a processing run produces a memory item in any category,
**When** the item is extracted,
**Then** the item includes a non-empty `evidenceSnippet`, a `sourceType` from `{transcript, notes, priorMemoryReference}`, and a `sourceLocation` with a stable locator format defined during curation.

**Given** a memory item is produced without accompanying evidence,
**When** the item is retrieved,
**Then** the item quality status is set to `Insufficient evidence`.

---

### US-3: Complete Structure with Weak Categories

**Given** a meeting has sparse source content covering only summary and action items,
**When** processing completes,
**Then** the memory includes all seven categories: summary, decisions, action items, risks/blockers, open questions, stakeholder notes, and next recommended focus,
**And** categories without sufficient evidence show `Insufficient evidence` status rather than being omitted.

---

### US-4: Briefing-Readiness Signal

**Given** a completed memory has strong or weak evidence on all five core categories (summary, decisions, action items, risks/blockers, open questions) and no unresolved conflicts,
**When** the memory is viewed,
**Then** the briefing-readiness signal shows `ready for briefing`.

**Given** a completed memory has `Insufficient evidence` on one or more core categories,
**When** the memory is viewed,
**Then** the briefing-readiness signal shows `ready with weak/insufficient sections` and flags which categories are insufficient,
**But** the memory is still eligible for downstream briefing input.

**Given** a memory has unresolved conflicting evidence on a core category,
**When** the memory is viewed,
**Then** the briefing-readiness signal shows `needs review` and the conflicting items are excluded from briefing eligibility.

---

### US-5: Prior Memory Visibility and Edit

**Given** processing uses one or more prior memories as input,
**When** the memory version is displayed,
**Then** the prior memories used are listed by safe reference (meeting ID, title, date) without exposing raw content in UI or logs.

**Given** a user reviews a memory that used prior memories,
**When** the user identifies an incorrectly matched prior memory,
**Then** the user can exclude it and trigger reprocessing, creating a new memory version.

**Given** a user identifies a relevant prior memory that was not used,
**When** the user includes it and triggers reprocessing,
**Then** the system processes with the additional input and creates a new memory version.

---

### US-6: Conflict Review Queue

**Given** processing detects conflicting evidence between current meeting and prior memory where both sides have source support,
**When** the conflict cannot be reconciled automatically,
**Then** the conflicting items are placed in a separate `needs_review` queue,
**And** the items are excluded from normal memory categories and from downstream briefing eligibility.

**Given** a user submits a conflict resolution with an evidence-backed resolution note,
**When** the resolution is saved,
**Then** the conflict is marked `reviewed`, the resolution note and original conflicting sources are preserved for audit,
**And** a new memory version is created with the reviewed state.

---

### US-7: Retry for Failed/Stale Processing

**Given** a processing job has reached `retry exhausted` state,
**When** the user views the meeting memory,
**Then** the UI shows the failure state with a user-safe failure reason and a manual retry action.

**Given** the source meeting has been edited (transcript, notes, participants, title, or date/time),
**When** the edited meeting is saved,
**Then** the active memory is marked `stale`, reprocessing is queued, and the meeting memory view shows `stale` and `reprocessing` state.

**Given** a user corrects source content or includes/excludes prior memories,
**When** the user triggers manual reprocess,
**Then** a new processing job is queued and a new memory version is created on success.

---

### US-8: Observability Without Sensitive Content

**Given** processing jobs run for multiple meetings,
**When** metrics and logs are emitted,
**Then** the following appear in metrics and logs: correlation ID, meeting ID (safe identifier), safe trigger type, failure class, retry count, duration, active version change reason, flag evaluation duration,
**But** raw transcript text, raw notes text, evidence snippets, participant names, participant emails, stakeholder note content, generated memory text, prior memory content, and resolution note text do not appear in any label or log field.

---

## Security Considerations

- Transcript text, notes text, evidence snippets, participant values, stakeholder notes, generated memory content, prior-memory references, and resolution notes are treated as sensitive meeting-derived data
- Raw meeting content and evidence snippets must not be written to logs or metric labels
- Processing worker authentication uses service account tokens; no cross-meeting data leakage
- Feature flag evaluation is server-side authoritative; browser-visible flags gate UI actions only
- Conflict resolution notes are stored per-user and not exposed to other meeting participants
- Prior-memory matching is constrained to same-participant, similar-title, close-chronology signals; no broad opaque matching that could infer cross-meeting relationships

---

## Threat Model

| Threat | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Processing invents unsupported memory (hallucination) | High | High | Require strict evidence for every item; `Insufficient evidence` states; strict hallucination eval fixtures |
| Evidence snippets appear trustworthy but do not actually support the item | Medium | High | Evidence-grounding evals that verify item/evidence consistency |
| Stakeholder notes become sensitive profiling | Medium | High | Limit to source-supported meeting-relevant business context; exclude sensitive attributes, personality profiling, unsupported inference |
| Safe automatic prior-memory matching chooses wrong context | Medium | High | Show prior memories used; allow include/exclude and reprocess; keep matching constrained; defer broad relatedness to future BRD |
| Conflicting evidence blocks useful briefing output | Medium | Medium | Isolate conflicts in review queue; allow evidence-backed resolution; non-conflicting memory remains active |
| Provider or worker outages prevent timely processing | Medium | Medium | Keep import non-blocking; queue/retry/retry-exhaust; preserve active memory; expose user-safe retry states |
| Version history grows without retention policy | Medium | Medium | Retain with source meeting; defer final retention/deletion/export to BRD-06 |
| Logs or metrics leak sensitive meeting-derived data | Medium | High | Explicitly ban raw content, evidence snippets, PII, generated memory, resolution text from logs/metrics; verify through observability evals |
| Browser/server flag drift exposes UI without processing support | Medium | Medium | Server flag is authoritative; misconfiguration emits safe observability; browser flag gates visible actions only |

---

## Dependencies

| Dependency | Type | Notes |
|---|---|---|
| BRD-02 Manual Meeting Import | Hard | Saved meetings, transcript/notes preservation, structured participants, meeting detail route, idempotency token |
| OpenAPI contract (Meeting schema) | Hard | `POST /meetings` write path must be updated before Phase 1 implementation |
| Feature flags registry | Hard | `ff_enable_meeting_memory_processing` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` must be registered |
| BRD-04 Pre-Call Briefing | Blocks | Briefings require trusted, source-grounded memory from BRD-03 |
| BRD-06 Privacy & Retention Controls | Soft | Final retention/deletion/export policy for memory versions and evidence |
| BRD-07 Manual Memory Correction | Soft | Richer conflict-resolution workflows |
| Future: Related Meeting Detection | Soft | Broad prior-memory relatedness |

---

## Open Questions

| ID | Question | Owner | Resolution |
|---|---|---|---|
| OQ-1 | Split feature flag or single dual-namespace flag? | Curator/Architect | Curation must decide whether automatic processing, memory UI, and review/reprocess actions need separate rollout flags |
| OQ-2 | Exact processing latency SLO | Architect | Must set concrete target after provider/runtime selection |
| OQ-3 | Exact source-location format | Architect | Must choose character offsets, line references, or stable locator format for transcript and notes |
| OQ-4 | Quality distribution metrics for MVP ops? | Curator | Must decide whether aggregate strong/weak/insufficient/conflicting counts are required |
| OQ-5 | Provider/runtime selection | Architect | Must compare architecture options while preserving provider-agnostic product requirements |

---

## Acceptance Criteria Checklist

Before marking this BRD as completed:

- [ ] All functional requirements (FR-1 through FR-21) defined with acceptance criteria
- [ ] All 8 user stories have Given/When/Then ACs
- [ ] All NFRs separated into type/target/measurement format
- [ ] Non-goals explicitly listed and distinguished from deferred items
- [ ] Feature flag assigned: `FF_ENABLE_MEETING_MEMORY_PROCESSING` (server) and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` (browser)
- [ ] Phase assigned: Phase 2
- [ ] All 5 open questions documented with owner and TBD resolution
- [ ] Security considerations and threat model complete
- [ ] Dependencies mapped; BRD-02 hard dependency confirmed; BRD-04 blocking relationship explicit
- [ ] Observability section complete with metric names, log event names, redaction rules
- [ ] 24 acceptance criteria (AC-1 through AC-24) defined with eval methods
- [ ] Feature flag registered in `specs/feature-flags.md`
- [ ] API contracts updated in `openapi.yaml` (if new endpoints introduced)
- [ ] Event contracts updated in `events.md` (if async events introduced)
- [ ] `scripts/check-status-sync.sh` exits 0

---

## Acceptance Criteria Detail

| ID | Criterion | Eval method |
|---|---|---|
| AC-1 | When disabled, successful manual import does not queue memory processing | Feature flag integration or E2E eval |
| AC-2 | When enabled, successful manual import queues processing without blocking redirect or meeting detail access | E2E eval |
| AC-3 | Completed processing result includes all 7 categories | Integration eval |
| AC-4 | Categories without evidence show `Insufficient evidence` not omission or guess | Fixture eval with sparse notes |
| AC-5 | Every supported item includes evidence snippet, source type, stable source location | Schema/integration eval |
| AC-6 | Unsupported or tempting claims are omitted or marked insufficient | Strict hallucination eval with tempting-unsupported fixtures |
| AC-7 | Evidence snippets actually support the attached item | Evidence-grounding eval |
| AC-8 | Action items preserve description, explicitly stated owner/due date; unstated fields remain unspecified | Fixture eval |
| AC-9 | Decisions are marked standalone/confirms/changes/reverses only with supporting evidence and eligible prior memory | Fixture eval |
| AC-10 | Stakeholder notes exclude sensitive personal profiling or unsupported attributes | Safety eval |
| AC-11 | Each processing attempt creates a new version; latest successful becomes active | Integration eval |
| AC-12 | Failed/retrying/retry-exhausted runs do not replace previous active memory version | Integration eval |
| AC-13 | Source meeting edits mark active memory stale and queue reprocessing | Integration or E2E eval |
| AC-14 | User-visible lifecycle states include all 10 states: not processed, queued, processing, completed, completed with insufficient evidence, failed, stale, reprocessing, retrying, retry exhausted | State-machine eval |
| AC-15 | Prior memories used are visible without exposing content in logs | E2E or integration eval |
| AC-16 | User can exclude incorrect prior memory, include missing one, and reprocess to create new version | E2E eval |
| AC-17 | Quality statuses use only `Strong evidence`, `Weak evidence`, `Insufficient evidence`, `Conflicting evidence` at item and category level | Schema/unit eval |
| AC-18 | Conflicting evidence appears in review queue, excluded from normal categories and briefing until resolved | E2E/integration eval |
| AC-19 | User can resolve conflict with evidence-backed resolution note, preserving original sources for audit | E2E eval |
| AC-20 | Briefing-readiness ready only when core categories have strong/weak evidence and no unresolved conflict | Readiness eval |
| AC-21 | Transient failures retry up to 3 times with exponential backoff before retry-exhausted | Job lifecycle integration eval |
| AC-22 | Worker/provider unavailability does not break manual import or existing active memory access | Degradation eval |
| AC-23 | Logs and metrics emit without sensitive content (no transcript, notes, evidence snippets, PII, memory content, resolution text) | Observability eval |
| AC-24 | Provider/model selection is not hard-coded into product acceptance behavior | Architecture review |

---

## Relations

- **Parent feature:** BRD-02 Manual Meeting Import (meeting records, transcript/notes, structured participants, meeting detail route)
- **Blocked by:** BRD-02, OpenAPI contract update (if new endpoints needed), feature flag registration
- **Blocks:** BRD-04 Pre-Call Briefing, BRD-06 Privacy & Retention Controls, BRD-07 Manual Memory Correction, future Related Meeting Detection
- **Related specs:** `specs/domain/brd-02-manual-meeting-import.md`, `specs/curated/brd-02-manual-meeting-import.md`, `specs/feature-flags.md`, `specs/brd-sequencing-roadmap.md`