# BRD-04 Pre-Call Briefing — Brainstormer Handoff

Status: Fresh brainstormer restart handoff
Source BRD: `specs/domain/brd-04-pre-call-briefing.md`
Date: 2026-05-23

## Starting point

The existing raw BRD at `specs/domain/brd-04-pre-call-briefing.md` is the correct starting point for the restarted BRD-04 pipeline. It already carries the intended brainstormer-authored scope, full BRD section coverage, a registered dual-namespace feature flag, concrete observability, NFRs, and evaluable acceptance criteria.

## Context read

- `AGENTS.md`: ContextPilot remains spec-driven, eval-driven, and feature-flagged; no implementation without BRD/eval/fitness checks.
- `STATUS.md`: BRD-04 is listed as Ready for Curation; STATUS contains some stale/inconsistent wording around BRD-03, but the task context states BRD-02 and BRD-03 are fully shipped.
- `specs/curated/brd-02-manual-meeting-import/brd.md`: BRD-02 provides the completed-meeting data foundation, structured participants, title/date/content source, and completed-meeting detail routes.
- `specs/curated/brd-03-meeting-memory-processing/brd.md`: BRD-03 provides evidence statuses, conflict exclusion, memory categories, briefing-readiness signals, versioning, prior-memory visibility, and stale/reprocess behavior used by BRD-04.
- Brainstorming reference: `references/contextpilot-brd-04-pre-call-briefing-decisions.md` confirms the intended Option A scope.

## Approaches considered

1. Manual-only source selection for BRD-04
   - Pros: simplest matching; maximum user control.
   - Cons: weak flagship value; users still have to remember old meetings.

2. Automatic balanced matching with progressive briefing
   - Pros: proves the core promise while staying explainable; uses BRD-03 evidence/readiness safeguards; keeps scope bounded to top 3 prior meetings.
   - Cons: matching quality depends on simple signals and may miss some valid context.

3. Full advanced related-meeting detection in BRD-04
   - Pros: richer relevance detection from the start.
   - Cons: over-expands BRD-04 and delays the value proof; better split into a later BRD.

Recommendation: keep Option 2. It is the fastest trustworthy proof of “never walk into a meeting cold again” without hiding complex relatedness inside the MVP.

## Scope confirmation

Keep BRD-04 scope as:

- Upcoming meeting briefing entry point and async generation.
- Automatic explainable related-meeting selection using at least 2 of 4 simple signals.
- Progressive briefing: concise summary first, detailed expandable sections below.
- BRD-03-aligned evidence policy: weak evidence caveated, insufficient evidence not stated as fact, unresolved conflicts excluded.
- Source details and relatedness reasons behind details affordances.
- No-prior-memory preparation shell.
- Versioned briefing records, stale marking, retry/regenerate, latest completed version continuity.
- Source exclusion/undo at upcoming-meeting scope only.
- Stakeholder notes only in expandable details with strict safety constraints.

Do not add manual source addition, advanced embeddings, provider delivery channels, strict length controls, privacy/retention policy, or correction UX to this MVP.

## Open question disposition

The draft currently had 4 open questions, not 7. The task body’s “7 OQs” appears stale or inherited from an earlier artifact. The 4 current draft OQs were resolved/deferred in the BRD:

- Similar title threshold: resolved using BRD-03 constrained title rule.
- Close chronology: resolved as prior meeting completed within 90 days before upcoming meeting start.
- Ranking weights: resolved as deterministic explainable ordering by matched signal count, recency, title similarity, stable ID tie-breaker.
- Manual addition of non-matched prior meetings: deferred to a future related-meeting detection/correction BRD.

No remaining product decision blocks curation. Shakil may still override if he wants display-name-only participant matching or manual source addition in BRD-04 MVP.

## Changes made during restart

- Updated BRD-04 FR-4 with concrete MVP signal definitions for same participant, similar title, same organization/client, and close chronology.
- Replaced BRD-04 Open Questions bullets with a disposition table resolving/deferring the current questions.
- Produced this fresh brainstormer handoff note for the reset pipeline.

## Coverage check

- Overview: present, concrete.
- User Stories: present, 8 stories.
- Functional Requirements: present, Must/Should/Could split.
- NFRs: present with latency, availability, traceability, privacy, authorization, accessibility, scale, retention.
- Observability: present with metrics, log events, health/readiness behavior, and privacy restrictions.
- Feature Flag: present, default false, server/browser envs, disabled behavior.
- Acceptance Criteria: present and eval-able, 22 criteria.
- Risks: present with mitigations.
- Open Questions: no blocking unresolved questions; one future-scope deferral remains documented.

## Curation gate

Ask Shakil to approve BRD-04 for curation before routing the pipeline. Do not implement.

Recommended next route after approval:

BRD approved and ready for curation. Orchestrator should route: systematic-refinement -> subagent-driven-development -> curating-artifacts -> completeness-score -> validate-design -> production-checklist -> implementation.
