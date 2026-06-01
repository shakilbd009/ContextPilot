# UX Findings — BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validator:** ux
**Target:** specs/domain/brd-03-meeting-memory-processing.md
**Note:** AC-18 references WCAG 2.1 AA inherited from BRD-01; validating against BRD-03 spec and BRD-01 baseline

---

## Verdict: NEEDS_ATTENTION

---

## Critical Issues (Must Fix)

None identified at Critical severity for BRD-03 UX requirements. BRD-03 inherits WCAG 2.1 AA from BRD-01 AC-18 (referenced in NFR table). BRD-01 validated UX coverage is assumed.

---

## High Priority Issues (Should Fix)

### [H1] Memory status UI states are described in prose but not specced with explicit states and transitions

**Source:** BRD-03 Should Have, "Provide clear copy for briefing-readiness states"
**BRD section:** Should Have / User-facing copy requirements

BRD-03 lists user-facing copy requirements for briefing-readiness states: "ready for briefing, ready with weak/insufficient sections, needs review, processing/reprocessing, failed/retry exhausted, and stale." These are described in prose requirements but not as a defined state machine. AC-14 validates the state machine exists, but the UX team needs explicit state definitions with:
- What text/UI is shown in each state
- Which states are interactive (processing, reprocessing, failed/retry exhausted)
- How the user navigates to retry or view conflicts

**Impact:** Without explicit UX state speccing, developers will make ad-hoc UI decisions. The resulting states may be inconsistent or unclear.

**Fix required:** Add a "Memory States" UX section to BRD-03 specifying the exact UI copy and behavior for each of: `not_processed`, `queued`, `processing`, `completed`, `completed_with_insufficient_evidence`, `failed`, `stale`, `reprocessing`, `retrying`, `retry_exhausted`. At minimum, list the user-visible label and any available actions for each state.

---

### [H2] Evidence inspection interaction is described but not specced as a UX component

**Source:** BRD-03 Should Have, "Make evidence easy to inspect from each memory item without forcing users to read the full transcript or notes"
**BRD section:** Should Have / Evidence inspection

The requirement to make evidence "easy to inspect" is a UX description, not a UX spec. No component behavior is described: Does the user click an item to expand evidence? Is there a tooltip? A side panel? An icon? How is the source location (char offset or line reference) rendered in the UI?

**Fix required:** Define an EvidenceSnippet UI component or pattern: click-to-expand evidence panel showing source type, source location as human-readable (e.g., "Line 42-45 in transcript"), and the evidence snippet text. Reference this pattern by name in the ACs.

---

### [H3] Briefing-readiness signal location in UI not specified

**Source:** BRD-03 FR-15, "Emphasize briefing-readiness in the primary memory view"
**BRD section:** FR-15 / Briefing-readiness

FR-15 says "Emphasize briefing-readiness in the primary memory view" but does not specify where. Is it a banner at the top? A badge on the meeting card? A status indicator on the memory tab? The term "primary memory view" is not defined in the BRD.

**Fix required:** Specify the location of the briefing-readiness signal in the meeting detail UI. E.g., "A persistent banner at the top of the memory tab shows the briefing-readiness status."

---

## Medium Priority Issues

### [M1] Insufficient evidence copy is required by Should Have but not specced

**Source:** BRD-03 Should Have, "Provide clear copy for `Insufficient evidence` categories"
**BRD section:** Should Have

The Should Have section requires "clear copy for `Insufficient evidence` categories so users understand that the system intentionally avoided guessing." This is a UX requirement but no copy is specified.

**Fix required:** Add the exact copy to use for `Insufficient evidence` categories to the "User-Facing Copy" section or a UX appendix. Example: "This section has insufficient evidence to generate a reliable summary. Add more details to the meeting transcript or notes to improve this section."

---

### [M2] Conflict review queue UI location and access path not specced

**Source:** BRD-03 FR-6, AC-18
**BRD section:** FR-6, AC-18

AC-18 says "Conflicting evidence appears in a needs-review queue and is excluded from normal memory categories." The review queue is described functionally but not as a UX element. Where does the user access it? From the meeting detail? From a global review queue? Is it a notification/badge on the meeting card?

**Fix required:** Specify the access path and location of the conflict review queue in the UX section.

---

### [M3] Prior-memory include/exclude interaction not specced as a UI flow

**Source:** BRD-03 FR-5, FR-12, AC-15, AC-16
**BRD section:** FR-5, FR-12

AC-15 requires "Prior memories used for processing are visible to the user." AC-16 requires "User can exclude an incorrect prior memory, include a missing relevant prior memory, and reprocess." The UI flow for viewing, excluding, including, and triggering reprocessing is not specced as a sequence of steps.

**Fix required:** Add a "Prior Memory Selection" UX flow: which screen shows used prior memories, how the user excludes or includes them, and the confirmation/trigger for reprocessing.

---

### [M4] Memory version comparison (compact diff) is Could Have — clarify if it ships with this BRD

**Source:** BRD-03 Could Have
**BRD section:** Could Have / "Show a compact diff between memory versions"

The Could Have section lists "Show a compact diff between memory versions for user review" with no decision made. If this ships with BRD-03, it needs a UX spec. If not, it should be moved to a future BRD.

**Fix required:** Decide whether version diff ships with BRD-03 or is deferred to BRD-07 or another future BRD.

---

## Low Priority Issues

### [L1] User-safe failure reasons (FR-29) not specced with example copy

**Source:** BRD-03 Should Have, "Display user-safe failure reasons"
**BRD section:** Should Have / User-safe failure reasons

FR-29 says "Provide full job failure behavior: ... user-safe failure reason." The Should Have section adds "Display user-safe failure reasons that explain whether the user should retry later, edit source content, or contact support." Examples of user-safe failure copy are not provided.

**Fix required:** Add example failure reason copy to the UX section. E.g., "Processing failed due to a temporary issue. Your meeting memory is safe — tap Retry to try again." vs. "Processing failed due to a system error. Please try again later or contact support if the problem persists."

---

## Positive Findings

- Briefing-readiness exclusion logic (unresolved conflicts block core categories, stakeholder notes and next recommended focus don't block) is clearly specced and well-designed
- Quality status enumeration (`Strong evidence`, `Weak evidence`, `Insufficient evidence`, `Conflicting evidence`) is consistent and user-comprehensible
- State machine (AC-14) explicitly lists all user-visible lifecycle states with clear labels
- Evidence requirement (FR-3) explicitly says "system must not fill weak categories with guesses" — good user trust design
- Incomplete memory structure (all categories always present, some marked insufficient) is a strong UX pattern — users always know what to expect
- User include/exclude of prior memories (FR-12) is well-described as a user control
- Conflict resolution flow (FR-14) with evidence-backed resolution note preserves audit trail

---

## Summary Table

| ID | Severity | Issue |
|----|----------|-------|
| H1 | High | Memory state UI labels and interactions not specced |
| H2 | High | Evidence inspection component not specced |
| H3 | High | Briefing-readiness signal location not specified |
| M1 | Medium | Insufficient evidence copy not specified |
| M2 | Medium | Conflict review queue access path not specced |
| M3 | Medium | Prior-memory include/exclude flow not specced |
| M4 | Medium | Version diff decision (ship vs defer) not made |
| L1 | Low | User-safe failure reason copy not provided |