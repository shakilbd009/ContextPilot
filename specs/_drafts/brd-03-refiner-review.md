# Refiner Review — BRD-03 Meeting Memory Processing

**Reviewer:** refiner
**Source:** `specs/domain/brd-03-meeting-memory-processing.md`
**Date:** 2026-05-21
**Task:** t_a68bc12e

---

## Summary

BRD-03 Meeting Memory Processing is structurally complete and has strong coverage across most sections. The evidence-grounding model, quality status taxonomy, version semantics, and briefing-safety isolation are well-defined. However, seven issues prevent acceptance: two HIGH (US-5 "safely" is undefined, US-6 conflict resolution UX is underspecified), three MEDIUM (NFR latency is vague, missing API contract, missing data model), and two LOW (ambiguous retry backoff language, AC eval methods not independently evaluable). A systematic ambiguity audit against the critique iron law surfaces five additional "Could Fix" items: the "provider-agnostic" escape hatch, stale detection mechanism, conflict resolution atomicity, prior-memory eligibility criteria, and evidence-granularity specification.

---

## Section Verification

| Section | Present | Content | Issues |
|---------|---------|---------|--------|
| Metadata | Yes | Complete | None |
| Overview | Yes | 3 paragraphs, clear | None |
| User Stories | Yes | 8 stories | US-5 "safely matched" is undefined; US-6 resolution UX is underspecified |
| Functional Requirements | Yes | Must/Should/Could organized | Missing API contract; missing data model |
| Non-Functional Requirements | Yes | 9 entries | Processing latency is "architecture-defined" — vague for implementation |
| Observability | Yes | 9 metrics, 9 log events, 2 health endpoints | Good coverage; missing queue depth limits and retry backoff specification |
| Feature Flag | Yes | Dual namespace, 4-state table | Correct |
| Acceptance Criteria | Yes | 24 criteria | AC-5, AC-9, AC-21 use vague eval language |
| Risks & Mitigations | Yes | 8 risks | Adequate |
| Relations | Yes | Parent, blocked-by, blocks, related specs | Correct |
| Open Questions | Yes | 5 entries | Adequate; no empty "None" claim |

---

## Issues

### HIGH — US-5 "safely matched" is undefined

**Location:** User Story US-5, line 35

**Problem:** "safely matched prior processed meeting memories" — the word "safely" is never defined. What makes a match safe vs unsafe? The FR adds constraints (line 70: "constrained signals such as same participants, similar title, and close chronology") but does not explain what an unsafe match would look like, what consequences an unsafe match would have, or what mechanism prevents unsafe matches. A reader cannot determine what "safe" means from this document.

Additionally, line 72 says users can "exclude incorrectly matched prior memories" — but if matching is done automatically and the user must correct it after the fact, the matching is not "safe" in a meaningful sense; it is tolerant of post-hoc correction.

**Fix:** Define "safe" as a property of the matching output, not just the matching inputs. Add to FR:

> "Safe automatic matching means: (1) matching is constrained to narrow, verifiable signals — same participant email, identical title string, or timestamp within N hours — where N is defined during curation; (2) every match is surfaced to the user before processing uses it as input; (3) no match is used as processing input without explicit user acknowledgment or a user-configured default that was set before the match occurred. Unsafe matches are those that rely on opaque semantic similarity, cross-system relationship inference, or participant-name-only fuzzy matching."

Then remove the word "safely" from US-5 and replace with "constrained, user-visible prior-memory matching."

---

### HIGH — US-6 conflict resolution UX is underspecified

**Location:** User Story US-6, FR lines 76–80, AC-18–AC-20

**Problem:** US-6 says users can "resolve conflicts" but provides no detail on what resolution looks like. Questions the BRD does not answer:

1. **What is the UI?** A side panel? A separate route? An inline edit? The conflict appears in a "review queue" but no queue UI is described.
2. **What does "evidence-backed resolution note" mean practically?** Must the user select from the two conflicting sources? Can they write free text? Must the note cite specific source snippets?
3. **What happens to the memory during resolution?** Is the briefing blocked while the conflict is in queue? Can users see the memory and its conflicts simultaneously?
4. **What is the atomic unit of resolution?** One conflict at a time? Batch resolution? All conflicts for one meeting at once?
5. **What triggers "reviewed state"?** Does resolving one conflict in a meeting mark the whole memory as reviewed, or only that conflict?

AC-19 tests that resolution works but does not test the UX behavior, the briefing-blockage behavior, or the atomicity of resolution.

**Fix:** Add a "Conflict Resolution UX" subsection under FR Must Have:

> **Conflict resolution UI** — Conflicts appear in a dedicated `/meetings/[id]/memory/review` sub-route accessible from the meeting memory view. Each conflict shows: (1) the conflicting evidence snippets with source attribution, (2) the affected memory category, (3) a free-text resolution note field, and (4) a "Mark Reviewed" button. The resolution note may reference specific evidence by source ID but is not required to select a winner — the user may also state that evidence is insufficient to resolve. Resolving a conflict emits a `meeting_memory.conflict.reviewed` log event and creates a new reviewed memory version. A meeting memory is briefing-ready only when all conflicts in the needs-review queue have been resolved or explicitly marked insufficient.

Also update AC-18, AC-19, AC-20 to include UX behavior testing and briefing-blockage verification.

---

### HIGH — NFR Processing latency is "architecture-defined" with no target

**Location:** NFR Table, line 121

**Problem:** "Processing latency — Architecture-defined during curation; processing must be asynchronous, observable, and non-blocking for meetings up to the BRD-02 50,000 character transcript-plus-notes limit"

This is not a requirement; it is a deferral. An implementor reading this NFR has no latency target to design toward, no SLO to measure against, and no basis for determining whether a processing run is "fast enough." The phrase "architecture-defined during curation" shifts responsibility to a future phase without a commitment to a specific range.

**Fix:** Provide an interim target with explicit deferral:

> **Processing latency** — Processing for a 50,000-character transcript-plus-notes input completes within 30 seconds at the 95th percentile under normal load. This target is provisional and will be updated to a provider-specific SLO during curation when the processing worker and model runtime are selected. Processing must be asynchronous, observable, and non-blocking for meetings up to the BRD-02 limit.

---

### MEDIUM — No API contract

**Location:** Functional Requirements (implied by lines 46–88)

**Problem:** The FR specifies queueing behavior, retry policy, lifecycle states, and observability but never describes the API surface. The document does not specify:
- Which endpoint queues a processing job (POST, internal queue, event trigger?)
- What the processing worker reads from (meeting ID + fields, or webhook payload?)
- How the processing result is written back (callback, mutation, event?)
- What the prior-memory include/exclude action does (API call, state mutation, event?)

An implementor cannot build the integration without guessing at the contract.

**Fix:** Add an API Contract subsection under FR Must Have. At minimum, define:
1. Internal queueing trigger (event on meeting save, or explicit job creation call?)
2. Processing worker read contract (GET /meetings/[id]/processing-input or message queue payload?)
3. Memory version write-back (PATCH /meetings/[id]/memory or callback URL?)
4. Prior-memory include/exclude action (PATCH /meetings/[id]/memory/inputs with body `{"include": [...], "exclude": [...]}`?)

---

### MEDIUM — No data model for memory versions

**Location:** Functional Requirements (implied by lines 63–66)

**Problem:** FR states that each processing run creates a new memory version, the latest successful version is active by default, and failed runs do not replace the active version. No schema is provided for: memory version records, category records, evidence records, conflict records, or resolution records. Field names, types, relationships, and ID generation are all unspecified.

**Fix:** Add a Data Model subsection under FR Must Have:

> **Memory version record:**
> | Field | Type | Required | Note |
> |-------|------|----------|------|
> | `id` | UUID | Yes | Primary key |
> | `meetingId` | FK → Meeting.id | Yes | Foreign key |
> | `versionNumber` | integer | Yes | Monotonically increasing per meeting |
> | `status` | enum(processing, completed, completed_with_insufficient_evidence, failed, retry_exhausted) | Yes | — |
> | `isActive` | boolean | Yes | Only one active per meeting |
> | `processedAt` | timestamp | No | Set on completion |
> | `sourceVersionId` | string | Yes | Source meeting version at time of processing |
> | `priorMemoryInputs` | JSON array | No | List of prior memory IDs used |
>
> **Memory category record:**
> | Field | Type | Required | Note |
> |-------|------|----------|------|
> | `id` | UUID | Yes | Primary key |
> | `memoryVersionId` | FK → MemoryVersion.id | Yes | Foreign key |
> | `category` | enum(summary, decisions, action_items, risks_blockers, open_questions, stakeholder_notes, next_recommended_focus) | Yes | — |
> | `qualityStatus` | enum(strong_evidence, weak_evidence, insufficient_evidence, conflicting_evidence) | Yes | — |
> | `content` | JSON or text | No | Structured content per category type |
>
> **Evidence record:**
> | Field | Type | Required | Note |
> |-------|------|----------|------|
> | `id` | UUID | Yes | Primary key |
> | `categoryId` | FK → MemoryCategory.id | Yes | Foreign key |
> | `snippet` | string | Yes | Source text excerpt |
> | `sourceType` | enum(transcript, notes, prior_memory) | Yes | — |
> | `sourceLocation` | string | Yes | Stable locator (format TBD during curation) |
>
> **Conflict record:**
> | Field | Type | Required | Note |
> |-------|------|----------|------|
> | `id` | UUID | Yes | Primary key |
> | `memoryVersionId` | FK → MemoryVersion.id | Yes | Foreign key |
> | `categoryId` | FK → MemoryCategory.id | Yes | Which category has the conflict |
> | `currentEvidenceId` | FK → Evidence.id | Yes | Evidence from current processing |
> | `priorEvidenceId` | FK → Evidence.id | Yes | Evidence from prior memory |
> | `resolutionNote` | text | No | User-provided resolution text |
> | `resolvedAt` | timestamp | No | Set when user resolves |
> | `resolvedBy` | userId | No | User who resolved |

---

### MEDIUM — AC-5 eval method is not independently evaluable

**Location:** Acceptance Criteria, line 207

**Problem:** AC-5 reads "Evidence-grounding eval checks item/evidence consistency for representative fixtures." This does not specify how an evaluator would determine that evidence actually supports the item. The eval method name "evidence-grounding eval" does not exist in the eval conventions, and "item/evidence consistency" is not defined operationally.

**Fix:** Rewrite AC-5:
> "AC-5 — A structured review eval with at least 10 fixture pairs (item text + evidence snippet) is conducted manually or via LLM-assisted review, where each fixture is labeled as: (a) evidence fully supports item, (b) evidence partially supports item, or (c) evidence does not support item. The eval reports the distribution of labels. All fixtures must be labeled (a) or (c) with zero (b) for the criterion to pass."

---

### LOW — Retry backoff is described but not specified

**Location:** FR line 85, NFR Retry behavior line 124

**Problem:** Both the FR ("exponential backoff") and the NFR ("exponential backoff") describe the retry strategy but do not specify: initial delay, maximum delay, jitter, or total timeout before retry exhaustion. An implementor could choose initial delay of 1ms or 1 minute and both would satisfy "exponential backoff."

**Fix:** Add to NFR Retry behavior:

> **Retry behavior** — Transient processing failures retry up to 3 times with exponential backoff (initial delay 1 second, multiplier 2x, max delay 30 seconds, ±10% jitter). After 3 failed retries the job enters retry-exhausted state and requires manual retry. Total time from first failure to retry-exhausted under this formula is approximately 1 + 2 + 4 = 7 seconds plus jitter, not counting processing time.

---

### LOW — AC-9 eval method lacks specificity

**Location:** AC-9, line 210

**Problem:** "Fixture eval verifies change-aware decision behavior with and without prior memory input" — does not specify what "change-aware decision behavior" means operationally. How does an evaluator determine that a decision is correctly marked standalone vs confirms vs changes vs reverses?

**Fix:** Expand AC-9:
> "AC-9 — Integration eval creates two meetings: one with no prior memories and one where the current meeting's decisions are semantically identical to a prior meeting's decisions. For each meeting, processing produces decisions with explicit status. The eval verifies: (a) decisions from the no-prior-memory meeting are marked standalone, and (b) decisions from the matching-prior meeting are marked confirms. A second scenario with semantically contradictory decisions verifies the changes/reverses states."

---

## Systematic Ambiguity Audit (Critique Framework)

Per the critique skill iron law — "stop when you've exhausted every angle, not when it feels thorough" — I audited for multi-dimensional ambiguities beyond the structured issues above:

### Could Fix — "Provider-agnostic" as undefined escape hatch

**Location:** FR line 88, NFR line 130

The document repeatedly says the system is "provider-agnostic" and "must not depend on a specific provider." This is a constraint, not a specification. The gap: if the processing worker fails or the provider changes behavior, what is the guaranteed minimum behavior? The BRD provides no fallback path when the provider is unavailable other than retry-exhaustion. A true provider-agnostic spec would define the minimum behavioral contract that any provider must satisfy — not just what the product doesn't depend on.

### Could Fix — Stale detection has no mechanism specified

**Location:** FR line 67, AC-13

"Automatically queue reprocessing when meeting transcript, notes, participants, title, completed date/time, or other source metadata changes" — but how does the system detect these changes? An event on the meeting record? Polling? Comparison of a stored source version hash? The mechanism is absent. Without it, implementors will guess, and different implementations may have different detection latencies.

### Could Fix — Conflict resolution atomicity undefined

**Location:** FR lines 78–80

"User can resolve a conflict with an evidence-backed resolution note that cites current/prior evidence or explicitly states that evidence is insufficient. Original conflicting sources and resolution note are preserved for audit."

What happens if the user starts resolving a conflict, loses their session, and returns? Is partial resolution allowed? Is there a draft state? What if they resolve partially, close the browser, and return? The document assumes a single atomic resolution action with no failure recovery.

### Could Fix — Prior-memory eligibility for decisions is underconstrained

**Location:** FR line 58

"Mark a decision as confirming, changing, or reversing a prior decision only when prior memory is eligible input and the current meeting evidence supports that relationship."

"Eligible input" is not defined. Is any prior memory eligible? Only memories from the same participants? Memories within a certain time window? Memories that passed a quality threshold? The ambiguity affects both decision status semantics and the broader prior-memory matching system.

### Could Fix — Evidence snippet and source location format is deferred

**Location:** FR line 53, Open Questions line 257

"Stable source location chosen during curation" — this defers a critical implementation detail. For a transcript stored as plain text, character offsets are stable within a version but shift if the transcript is edited. For a transcript stored with line numbers, offsets are more robust but not universal. The choice affects evidence-granularity and item/evidence consistency verification. Deferring it to curation without a provisional default means implementors will choose arbitrarily and the eval may not be reproducible across teams.

---

## Verification Checklist

| Requirement | Status | Notes |
|------------|--------|-------|
| All 9 BRD sections present | Pass | No missing sections |
| No TBD or empty dashes | Pass | All sections have content |
| Observability has >= 1 metric | Pass | 9 metrics defined |
| Observability has >= 1 log event | Pass | 9 log events defined |
| Feature flag: server env defined | Pass | `FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| Feature flag: browser env defined | Pass | `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` |
| Feature flag: default false | Pass | Explicitly stated |
| Feature flag: 4-state table correct | Pass | Matches AGENTS.md dual namespace rule |
| Open questions noted | Pass | 5 entries, none empty |
| NFRs have specific numeric targets | **Partial** | Latency is "architecture-defined" — deferral not a target |
| ACs are specific and eval-able | **Partial** | AC-5 and AC-9 are vague |
| API contract specified | **Fail** | No endpoint shapes or data mutation contracts |
| Data model specified | **Fail** | No schema for memory versions, categories, evidence, conflicts |
| Retry backoff formula specified | **Fail** | "Exponential backoff" with no numeric parameters |
| "Safely" defined in US-5 | **Fail** | Term is used but never defined |
| Conflict resolution UX defined | **Fail** | No UI description, atomicity, or queue behavior |

---

## Recommendations

**Must fix before PM acceptance (HIGH):**

1. **Define "safely" in US-5** — replace vague term with explicit matching constraints, signal requirements, and user-visibility requirements.
2. **Specify conflict resolution UX** — add UI description, queue behavior, atomicity model, and briefing-blockage semantics.
3. **Provide interim processing latency target** — 30 seconds at P95 with explicit deferral to curation for provider-specific SLO.

**Should fix before Phase 2 implementation (MEDIUM):**

4. **Add API contract** — internal queue trigger, processing input read, result write-back, prior-memory include/exclude action.
5. **Add data model** — memory version, category, evidence, conflict, resolution records at minimum.
6. **Fix AC-5 eval method** — make independently evaluable with concrete fixture requirements.
7. **Fix AC-9 eval method** — add explicit two-scenario structure with expected outcomes.

**Nice to have for production quality (LOW):**

8. Specify retry backoff parameters (initial delay, multiplier, max, jitter).
9. Define stale detection mechanism (event, polling, or hash comparison).
10. Add conflict resolution atomicity/failure-recovery model.
11. Define prior-memory eligibility criteria for decision status.
12. Choose provisional evidence snippet/location format with explicit deferral.

---

## Prior Work

This review is the first gap analysis for BRD-03. It is independent of prior work on BRD-02 (t_10b6141a, t_3be30df3). The review methodology follows the same structure: structured issues by severity, systematic ambiguity audit per critique skill, and verification checklist.