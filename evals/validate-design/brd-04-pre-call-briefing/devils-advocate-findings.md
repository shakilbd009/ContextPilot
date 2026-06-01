# Devil's Advocate Findings: brd-04-pre-call-briefing

## Overview
Critical analysis of BRD-04 Pre-Call Briefing using Multi-Dimensional Analysis and Assumption Archaeology frameworks.

---

## 1. OQ Resolution Completeness

**Claim:** "No unresolved OQ/TBD language" — brd.md lines 373–379 show OQ-1 through OQ-7 as "resolved."

**Analysis:**

| OQ | Claimed Resolution | Devil's Advocate Finding |
|----|-------------------|--------------------------|
| OQ-4 | "briefing max length/section cap managed through progressive format" | Progressive format (concise summary + expandable details) is the stated solution, but FR-28 explicitly defers "strict character, section, or reading-time limits" to a future BRD. Who decides when the progressive format is insufficient? No threshold defined. |
| OQ-5 | "ready_with_caveats derived from BRD-03 when any category has weak evidence or insufficient evidence" | BRD-03 evidence statuses apply per-category. BRD-04 AC-7 requires "explicit insufficient-context states where content is missing." The cross-category aggregation rule (any category triggers `ready_with_caveats`) is stated but not exemplified. What if only `open_actions` is weak but everything else is strong? Still `ready_with_caveats`. Is this the intended behavior? |
| OQ-7 | "briefing versions are immutable once created; on-demand regeneration creates a new version" | Immutable briefing versions are specified, but no retention policy is defined (deferred to BRD-06). If a user regenerates 100 times, 100 versions persist. No cleanup, no archival. What is the storage implication? |

---

## 2. Edge Case: Regeneration + Source Change Race Condition

**Scenario:** User A has briefing v1 (sources: S1, S2). While v1 is displayed, source S2 memory is reprocessed (new version). User then clicks Regenerate. 

**Questions:**
1. Does the new regeneration use the updated S2 memory, or the S2 version that was current when regeneration was triggered?
2. Is the existing v1 marked stale before regeneration completes, or only after?
3. If regeneration fails, v1 is still displayed with stale S2 — but the stale indicator depends on read-time derivation (ADR-0011). Does the UI show "stale" immediately or on next refresh?

**Spec gap:** The staleness derivation algorithm in ADR-0011 describes read-time comparison but does not address when staleness is first computed — on page load, or on first API call?

---

## 3. Edge Case: Exclude + Restore + Regenerate Timing

**Scenario:** User excludes S1, then restores S1, then immediately clicks Regenerate before the restore transaction commits.

**Spec gap:** No optimistic locking or transaction isolation level specified for the exclusion/restore flow. If two concurrent restore requests arrive for the same exclusion, UNIQUE constraint on `(upcoming_meeting_id, excluded_source_meeting_id)` catches the second one — but the user experience is not defined.

---

## 4. Assumption: Owner-Only ACL Is Sufficient

**Assumption:** ADR-0009's owner-only semantics are sufficient for MVP. The briefing depends on BRD-02 meetings and BRD-03 memory — but both are also owner-only.

**Blind spot:** What happens when a user who is a PARTICIPANT (not owner) in a meeting tries to access its briefing? BRD-02's ACL model is owner-only. BRD-04 FR-19 says "authorized to view the upcoming meeting" — but if the participant wasn't invited via the app (external calendar invite), they have no access. This is accepted MVP behavior, but no explicit "out of scope for MVP" callout exists for cross-participant briefing sharing.

---

## 5. Scope Creep Risk: "Preparation Status States"

**Assumption:** 7 preparation status states (FR-20) are unambiguous: generating, ready, ready_with_caveats, no_prior_memory, stale, failed, regenerating.

**Question:** What happens when a briefing is `generating` and the user navigates away, comes back 30 minutes later, and the job failed? Status is `failed`. Latest completed briefing available. User clicks Regenerate — now state is `regenerating`. But a previous `failed` version also exists. Is the failed version accessible in version history?

**Spec gap:** The relationship between `preparation_status` (runtime state) and `status` in `briefing_versions` (`active`/`superseded`) is not explicitly mapped for all state transitions. E.g., does a failed generation create a `briefing_versions` row with `status=superseded`? Or is it just an ephemeral state?

---

## 6. Observability: Missing Metric Names in BRD

**Assumption:** All metrics and log events are enumeratable.

**Finding:** BRD FR-22 (Low-Cardinality Observability) and FR-23 (Product Usefulness Events) describe the categories of events but do NOT enumerate specific metric names or log event names in the spec itself (only in eval files):
- `cp_briefing_generation_duration_ms` — in perf eval only
- `cp_briefing_queue_saturation` — in perf eval only  
- `briefing.flag_misconfiguration` — in BRD feature flag table (line 291)
- Product usefulness events (viewing, expanding, regenerating, exclude/restore, no-prior-memory shell) — event names not specified

**Risk:** When implementation begins, engineers will need to look in eval files to find metric names instead of the spec. This is a documentation clarity issue.

---

## Summary

| Finding | Severity | Description |
|---------|----------|-------------|
| OQ-4 threshold undefined: progressive format sufficiency has no defined trigger | Medium | No criteria define when "progressive format proves insufficient" (BRD non-goal line 351) |
| OQ-5 aggregation rule not exemplified | Low | Any-category-triggers-ready_with_caveats stated but not illustrated with mixed-quality scenario |
| No retention/archival policy for briefing versions | Medium | 100 regenerations = 100 persisted versions; deferred to BRD-06 but storage implications unstated |
| Staleness first-computed-on trigger not defined | Medium | ADR-0011 read-time algorithm; when does UI first show stale badge? |
| Regeneration + source memory change race condition | Medium | No specified handling for concurrent source memory updates during/after generation |
| No optimistic locking on exclusion/restore | Low | UNIQUE constraint prevents duplicates but concurrent restore UX not defined |
| Cross-participant briefing access not explicitly out of scope | Low | Owner-only ACL assumed but participant ACL edge case not called out |
| preparation_status to briefing_versions.status mapping incomplete | Medium | Failed generation lifecycle — does it create a version row? |
| Metric/log event names not in BRD spec | Medium | Only in eval files; engineers must check multiple sources |

**Verdict: NEEDS_ATTENTION** — 7 medium, 2 low findings.
No blocking issues, but multiple specification gaps that could cause implementation ambiguity.
