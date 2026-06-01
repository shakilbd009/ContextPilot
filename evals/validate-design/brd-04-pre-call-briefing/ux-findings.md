# UX Findings: brd-04-pre-call-briefing

## Overview
UX validation of BRD-04 Pre-Call Briefing. Accessibility, WCAG 2.1 AA, error states, focus management, and disabled-flag zero-state reviewed against brd.md.

---

## 1. WCAG 2.1 AA Accessibility (NFR line 269)

**Spec requirement:** "Briefing UI, generating/failure/stale states, and expand/details controls meet WCAG 2.1 AA expectations."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | WCAG 2.1 AA commitment stated in NFR |
| AC-8 usability eval | PASS | AC-8: "A reviewer can understand the recommended meeting focus from the concise summary in under 60 seconds" — usability test for readability |
| BRD-01 component inventory | UNVERIFIED | NFR references "Specific criteria must be verified against BRD-01 component inventory" — BRD-01 is the App Shell BRD; no explicit cross-reference to component accessibility compliance |
| ARIA live regions | NOT SPECIFIED | Generating state, stale indicator, and failure state do not specify ARIA live region behavior for screen readers |

**Required action:** BRD-04 UX validation should reference the specific BRD-01 component accessibility checklist. ARIA live region behavior for dynamic states (generating, stale, failed) should be specified in the UX eval.

---

## 2. Disabled-Flag Zero-State

**Spec requirement:** AC-1: "With both flags false, briefing UI entry points are hidden."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Flag behavior table (brd.md lines 286–292) defines all 4 flag state combinations |
| E2E eval | PASS | AC-1 scenarios in evals/e2e lines 13–45 cover entry point visibility and backend rejection |
| Zero-state content | PASS | FR-10 (no-prior-memory shell) and AC-6 define behavior when no prior memory exists |

---

## 3. Error Messages and Failure States (FR-18)

**Spec requirement:** "Failed generation shows a safe user-facing failure reason, offers retry/regenerate, and does not remove the latest completed briefing."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | FR-18 fully defines failure behavior |
| AC-18 | PASS | AC-18 is explicit about retry/regenerate behavior |
| E2E eval | PASS | Failed generation scenarios covered in evals/e2e |
| "Safe" failure reason definition | MEDIUM | BRD does not define what constitutes a "safe" failure reason vs. an unsafe one (e.g., should technical errors be shown to users?) |

---

## Summary

| Finding | Severity | File:Line | Description |
|---------|----------|-----------|-------------|
| ARIA live regions for dynamic states not specified | Medium | brd.md NFR line 269 | Generating, stale, and failed states should specify ARIA live region behavior |
| BRD-01 component inventory cross-reference not verified | Medium | brd.md NFR line 269 | "Specific criteria must be verified against BRD-01" — no verification evidence in BRD-04 |
| "Safe" failure reason definition vague | Low | brd.md FR-18 | No guidance on what constitutes safe vs unsafe user-facing error content |

**Verdict: NEEDS_ATTENTION** — 2 medium, 1 low.
