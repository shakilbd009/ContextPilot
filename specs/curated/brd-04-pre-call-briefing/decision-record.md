# BRD-04 Pre-Call Briefing — Decision Record

---
brd_id: brd-04
title: Pre-Call Briefing Decision Record
curated: 2026-05-23
source: specs/domain/brd-04-pre-call-briefing.md (approved raw BRD, brainstormer, 2026-05-23)

---

## OQ Decisions

The following seven open questions were resolved during BRD-04 authoring and review. Each is documented with its resolution source, rationale, and disposition for traceability.

---

### OQ-1: Minimum Bar for Recommended Objective When No Prior Memory

**Question:** What constitutes an acceptable no-prior-memory preparation shell? Must the system fabricate prior context or can it limit to what the upcoming meeting itself provides?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — Open Questions section, similar title threshold and ranking weights were resolved as part of the same brainstorming session.

**Decision:** When no qualifying prior memory exists, the system generates a limited preparation shell from upcoming meeting title/description/metadata only, clearly marked `No prior memory found`. The shell includes objective, recommended focus, and suggested prep questions derived from upcoming meeting metadata. The shell must not imply prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks.

**Rationale:** Fabricating prior context would undermine user trust and contradict BRD-03's evidence grounding principles. Limiting the shell to upcoming-meeting-derived content is honest, useful, and avoids the risk of users acting on false premises. User testing can later determine whether the limited shell is sufficient or whether manual source addition is needed.

**Incorporated in:** BRD-04 FR-10 (No-Prior-Memory Preparation Shell)

---

### OQ-2: Multi-Meeting Briefing Support

**Question:** Does the MVP support briefing from multiple prior meetings simultaneously, or is it strictly limited to a single prior meeting?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-5 and Open Questions section.

**Decision:** The MVP supports briefing from up to 3 qualifying prior meetings simultaneously. If fewer than 3 qualify, briefing uses the available qualifying meetings. Briefing content aggregates relevant context across all qualifying sources.

**Rationale:** A single-meeting briefing would often be insufficient given that meetings frequently reference decisions and actions from multiple prior sessions. Limiting to 3 prevents coherence degradation and keeps MVP scope bounded while proving the multi-meeting aggregation concept.

**Incorporated in:** BRD-04 FR-5 (Source Scope), API Contract, Data Model

---

### OQ-3: Max Selectable Sources Limit

**Question:** What is the maximum number of qualifying prior meetings that can be selected for a single briefing?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-5 (source scope).

**Decision:** Maximum is 3 qualifying prior meetings per upcoming meeting.

**Rationale:** 3 provides sufficient context for most pre-call scenarios without overwhelming the user or diluting quality. This also bounds generation latency (P95 < 60s), keeps the data model simple, and aligns with the performance target for MVP. If user testing shows 3 is insufficient, a future enhancement can raise the limit with explicit justification.

**Incorporated in:** BRD-04 FR-5, Data Model (source_count field), NFR (Scale)

---

### OQ-4: Briefing Max Length/Section Cap

**Question:** Should the MVP impose hard character, section, or reading-time limits on briefing output?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-7 (progressive format) and FR-9 (detailed sections), with the understanding that hard limits are deferred to a future enhancement (FR-28).

**Decision:** The MVP uses a progressive format structure (concise summary first, then expandable details) to manage length naturally. Hard character/section/reading-time limits are out of scope for MVP and may be addressed in a future enhancement (FR-28) if the progressive format proves insufficient for length management.

**Rationale:** Hard limits risk truncating useful context mid-section and complicate eval definitions. The progressive format gives users control: they get a scannable summary first and can opt into full detail. This is a WHAT-level decision about briefing structure, not an implementation constraint.

**Incorporated in:** BRD-04 FR-7, FR-9, Non-Goals

---

### OQ-5: Minimum Quality Threshold for Whole-Briefing Low-Confidence Signal

**Question:** At what point does the briefing overall signal that its content may not be reliable?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-8 (concise summary contents), FR-11 (evidence policy), FR-20 (preparation status states), and NFR (Evidence safety).

**Decision:** The briefing signals low confidence at the whole-briefing level when any included category has weak evidence or insufficient evidence, mapped to the `ready_with_caveats` preparation status. This is derived from BRD-03's category-level quality statuses aggregated to the briefing level. The specific threshold (any weak/insufficient vs. only insufficient) is determined by BRD-03's category aggregation rules: a category has the worst quality status of any item within it, and briefing readiness follows BRD-03 FR-15.

**Rationale:** `ready_with_caveats` communicates to users that the briefing is functional but some underlying evidence is weaker than ideal. The status is surfaced in the preparation status field and in the concise summary's preparation status line. This is consistent with BRD-03's evidence status conventions and avoids a binary ready/not-ready signal that would be less actionable.

**Incorporated in:** BRD-04 FR-8, FR-11, FR-20, NFR (Evidence safety)

---

### OQ-6: Upcoming Meeting Creation UX

**Question:** When and how does an upcoming meeting get created, and how does that trigger briefing generation?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-1 (upcoming meeting briefing entry point) and FR-2 (automatic async generation on upcoming meeting creation).

**Decision:** Upcoming meeting creation is handled separately from briefing generation. When an upcoming meeting is created and the feature flag is enabled, the system queues an asynchronous briefing generation job. The user lands on the upcoming meeting detail page immediately without waiting for generation. The briefing entry point on the upcoming meeting detail page reflects the generation state (generating, ready, failed, etc.). No calendar, email, or conferencing provider integrations are required for MVP.

**Rationale:** Decoupling upcoming meeting creation from briefing generation keeps BRD-04 MVP scope bounded and avoids blocking UI on generation latency. The async pattern is consistent with BRD-03's memory processing architecture. Future integrations (calendar sync, etc.) can be addressed in future BRDs.

**Incorporated in:** BRD-04 FR-1, FR-2, Non-Goals

---

### OQ-7: Immutability vs. On-Demand Regeneration

**Question:** Can an existing briefing version be edited or updated in place, or does regeneration always create a new version?

**Resolution source:** `specs/domain/brd-04-pre-call-briefing.md` — FR-14 (versioned briefing records), FR-15 (stale briefing behavior), FR-16 (source exclusion), and FR-17 (exclusion undo and visibility).

**Decision:** Briefing versions are immutable once created. On-demand regeneration creates a new distinct version. Briefings become stale when source memory changes; staleness triggers a regeneration offer but does not auto-regenerate or modify the existing version. Source exclusions are upcoming-meeting-scoped and persist across regenerations for that specific upcoming meeting only.

**Rationale:** Immutability provides a reliable audit trail and allows users to compare versions. Staleness detection (rather than auto-regeneration) respects user agency: the user decides when to regenerate. Per-upcoming-meeting source exclusions prevent global exclusion rules from having unintended side effects across other upcoming meetings. This model is consistent with BRD-03 version management and memory staleness handling.

**Incorporated in:** BRD-04 FR-14, FR-15, FR-16, FR-17, Data Model (briefing_versions.is_active)

---

## Summary Table

| OQ | Subject | Decision | Key Constraint |
|----|---------|----------|-----------------|
| OQ-1 | No-prior-memory minimum bar | Shell from upcoming meeting metadata only, no fabricated continuity | Marked `No prior memory found`; no implied prior decisions/actions |
| OQ-2 | Multi-meeting support | Up to 3 qualifying prior meetings | Aggregation across all qualifying sources |
| OQ-3 | Max selectable sources | 3 | Bounds generation latency, prevents coherence degradation |
| OQ-4 | Briefing max length/section cap | Progressive format (concise summary + expandable details) | Hard limits deferred to future BRD (FR-28) |
| OQ-5 | Whole-briefing low-confidence signal | `ready_with_caveats` when any category has weak/insufficient evidence | Derived from BRD-03 category aggregation |
| OQ-6 | Upcoming meeting creation UX | Async non-blocking generation queued on upcoming meeting creation | No calendar/provider integrations required for MVP |
| OQ-7 | Immutability vs. regeneration | Immutable versions; regeneration creates new version; staleness offers regeneration | Audit trail preserved; user agency respected |

---

## Other Resolved Questions

The following items from the raw BRD's Open Questions section were resolved during the original brainstorming session and are incorporated directly into BRD-04:

| Question | Resolution | Where incorporated |
|----------|------------|-------------------|
| Similar title threshold | Levenshtein distance 3 or less, or 4+ consecutive shared tokens | BRD-04 FR-4 |
| Close chronology window | Source meeting completed within 90 days before upcoming meeting start | BRD-04 FR-4 |
| Ranking weights for top 3 | Highest signal count first, then most recent, then stronger title similarity, then stable ID | BRD-04 FR-5 |
| Manual addition of non-matched prior meetings | Deferred to future BRD | BRD-04 Non-Goals |

These were resolved in the raw BRD and are not repeated as OQ decisions in this record.