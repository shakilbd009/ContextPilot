# Devil's Advocate: brd-04-pre-call-briefing

**Validator:** validate-design (orchestrator — critique framework)
**Date:** 2026-05-23
**Target:** specs/curated/brd-04-pre-call-briefing/brd.md

---

## Blind Spots the BRD Team May Have Missed

### Assumption: "Phase 2" means the feature is fully scoped

BRD-04 is labeled Phase 2 and blocks future dashboard preparation states, advanced related-meeting detection, continuity/what-changed views, briefing delivery integrations, and provider connector workflows. This creates a large and poorly defined Phase 2 scope. The "Pre-Implementation Approval Gate" (lines 367–377) references 7 OQ resolutions but they are embedded in prose, not tracked as formal open issues.

**What if:** The 7 OQ resolutions represent significant scope decisions that should have been separate ADRs or at minimum tracked with explicit acceptance criteria in the spec rather than buried in a gate section?

**Recommendation:** Extract the 7 OQ resolutions into explicit acceptance checkboxes in the spec with owner and disposition clearly marked. Do not let them be "resolved" by prose — confirm each has a concrete test.

---

### Assumption: Future-dated source meetings are excluded only by "close chronology" rule

FR-4: "Future-dated source meetings must not qualify as prior context." This means if a source meeting is scheduled in the future (e.g., a recurring series), it is disqualified from being a briefing source. However, the spec does not address what happens if a user has a series of meetings where some are past and some are future-dated.

**What if:** A user has a recurring weekly team meeting. The last 3 occurrences are completed (past). The next occurrence is scheduled (future). FR-4 says the future-dated one must not qualify. But what about the close chronology signal? If the upcoming meeting is scheduled for March 15 and the last completed occurrence was March 1, the 90-day signal still passes. Is the meeting excluded because it's "future-dated" or included because it passes the chronology threshold?

**The spec says:** "Future-dated source meetings must not qualify as prior context." The test is a single binary check — is the source meeting's `completed_at` in the future? If yes, disqualify regardless of other signals.

**The gap:** This rule only works when `completed_at` is populated. If a user creates a meeting record with a future `completed_at` date (indicating it's a scheduled future meeting not yet held), it would be disqualified. But BRD-02 requires `completedAt` (past tense) for manual meeting import. So how would a future-dated meeting ever exist as a source? If the answer is "they wouldn't via BRD-02, but calendar-synced meetings might in the future", this creates an inconsistency in the signal evaluation logic.

**Recommendation:** Clarify in BRD-04 FR-4 what `completed_at` means for scheduled (not-yet-occurred) meetings and how future-dated source disqualification interacts with calendar-sync scenarios.

---

### Assumption: Staleness is derived at read time with no operator visibility

ADR-0011 adopts Option A (derived at read time). The stale state is computed fresh on each briefing read by comparing `source_memory_version_ids` in `_meta` against current active memory version `updated_at`. This means:

- There is no persisted `is_stale` column
- There is no way to query "show me all stale briefings" without scanning every briefing
- The stale indicator is always accurate but only visible when someone loads the briefing page

**What if:** An operator needs to proactively identify and reach out to users whose briefings are stale. They cannot do this without loading every briefing page individually. This is a significant gap for operational visibility.

**This is acknowledged in ADR-0011:** "If direct stale-query capability is needed (e.g., an operator dashboard showing all stale briefings), Option B (add `is_stale BOOLEAN`) can be adopted via ADR revision."

**Recommendation:** Add a low-priority metric or background job that emits `briefing.stale` events when staleness is detected at read time. This gives operators signal without needing a persisted column. Track as a should-have for Phase 2.

---

### Assumption: The "owner-only" ACL from ADR-0009 is sufficient for MVP

ADR-0009 adopts owner-only semantics (only `createdBy` can access meeting, memory, and briefing). This is simple but creates a specific gap: if an executive assistant manages a manager's calendar and creates upcoming meetings on their behalf, who owns the briefing? If the EA creates the upcoming meeting, they own it — but the briefing is about the manager's meeting, not the EA's.

**What if:** A user wants to share a briefing with a meeting participant who is not the owner. There is no mechanism. The ADR explicitly defers this: "Users cannot share briefings with meeting participants who are not owners."

**This is not a bug — it's an accepted limitation.** But it means the MVP is limited to single-owner use cases. If the product's target users are executives with EAs or team coordinators who manage others' meetings, this limitation will surface quickly.

**Recommendation:** Confirm with PM whether the Phase 2 target users are single-owner (personal productivity) or team-coordinator (assistant managing executive's briefings) before heavily investing in Phase 2 implementation. If team-coordinator is a target, this needs a separate BRD for ACL extension before Phase 2 is locked.

---

### Assumption: "Regeneration continuity" (NFR-4) is satisfied by returning the latest completed briefing

NFR-4: "Latest completed briefing remains readable while regeneration is queued/running/failed."

The spec says: "The latest completed briefing remains visible while regeneration runs." This is satisfied by showing the latest completed version during `regenerating` state.

**What if:** The first-ever briefing for an upcoming meeting is requested before generation completes? The integration eval tests `generating` status returning latest (possibly null or in-progress content). But if the first generation is taking a long time and the user navigates away and returns, they see `generating` status. That's fine.

**What if:** The user triggers manual regenerate. The latest completed briefing (v1) remains visible. A new job is queued. The user navigates away. The job fails. What do they see? According to AC-18, they should see "safe user-facing failure reason" and "latest completed briefing (v1) still available." So v1 remains visible — correct.

**The gap:** There's no defined behavior for what happens if the user triggers regenerate and the job is still queued/running when they return. The integration eval says "GET /briefing returns latest completed briefing with `preparation_status = 'regenerating'`" during regeneration. But if v1 is the "latest completed" and we're in `regenerating` state, do we return v1 or do we return nothing until v2 succeeds? The spec suggests v1 remains visible, which is the correct behavior.

This is handled correctly. No finding.

---

### Assumption: Evidence policy in BRD-04 matches BRD-03 exactly

BRD-04 FR-11 references BRD-03 evidence status conventions. BRD-03 FR-4a specifies HTML escaping for evidence snippets displayed in the browser. BRD-04 does not mention HTML escaping for briefing content.

BRD-04's briefing content includes `top_prior_context`, `open_actions`, `risks_questions`, `stakeholder_notes`, and other fields that could contain content copied from source meeting transcripts or notes. The same injection risk that BRD-03 addresses with FR-4a (evidence snippet display sanitization) exists in BRD-04 briefing content rendering.

**What if:** A source meeting's `summary` field in BRD-03 memory contains `<script>alert(1)</script>`. This gets pulled into BRD-04's `top_prior_context` in the briefing JSON. The briefing is rendered in the browser. No HTML escaping is specified in BRD-04 FR-11 or FR-13.

**The spec says:** "Weak-evidence items appear only with clear caveats; insufficient-evidence items are not presented as facts." It does not say "and all text content is HTML-escaped before rendering."

**BRD-03 FR-4a** explicitly requires: "Before any evidence snippet is rendered in the memory view or in any API response that exposes it for client-side display, HTML special characters — specifically `<`, `>`, `&`, double quotes (`\"`), and single quotes (`'`) — must be escaped."

**Recommendation:** BRD-04 should either (a) adopt BRD-03's HTML escaping requirement for all briefing content fields, or (b) explicitly delegate to BRD-03's FR-4a as the governing sanitization standard for any briefing content derived from memory. This is a security gap if briefing content can carry unescaped HTML from source transcripts.

---

## Summary

The devil's advocate review surfaces three findings not captured in the main validation:

| Finding | Severity | Type |
|---------|----------|------|
| OQ resolutions embedded in prose, not tracked as formal acceptance criteria | Medium | Process |
| Future-dated source disqualification logic unclear for calendar-sync scenarios | Medium | Ambiguity |
| BRD-04 briefing content HTML escaping not specified (delegates to BRD-03 FR-4a?) | Medium | Security |

All three are Medium — the spec is well-structured and the critical gaps (OpenAPI, upcoming_meetings table) are the real blockers.

---

*Devil's advocate findings — apply critique framework against BRD-04 spec directly. Not a specialist validator; no sub-agent dispatched.*