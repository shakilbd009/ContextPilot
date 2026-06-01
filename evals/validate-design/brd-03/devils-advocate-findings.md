# Devil's Advocate Findings — BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validator:** devils-advocate (via validate-design orchestrator)
**Target:** specs/domain/brd-03-meeting-memory-processing.md

---

## Verdict: NEEDS_ATTENTION

---

## Challenged Assumptions

### [A1] "Provider-agnostic" is stated but the BRD still implicitly assumes a single-pass extraction-then-synthesis model

**Assumption challenged:** FR-17 says the BRD "must not depend on a specific processing provider, model, or runtime." AC-24 confirms provider portability. However, the BRD's memory structure (7 categories, evidence-grounded items, change-aware decisions) implies a specific processing architecture: extract memory items from the current meeting, optionally augment with prior memories, detect conflicts, produce a structured JSON output. This is a single-pass or two-pass LLM pipeline. The BRD never questions whether a different processing paradigm could achieve better results.

**Blind spot:** What if the "right" architecture is a multi-pass process where different passes handle different categories? Or a hierarchical model where high-level synthesis and category-specific extraction happen in separate stages? The BRD assumes the architecture rather than the outcome. This locks in an implementation pattern before the provider is selected.

**Question to resolve:** Is the single-pass extraction architecture a firm requirement or an assumption? If it is an assumption, the BRD should describe "what" (outcomes, evidence requirements, category structure) not "how" (extraction pipeline). This would allow multi-pass or novel architectures.

---

### [A2] Evidence-grounding eval (AC-7) requires item/evidence consistency but has no defined pass threshold

**Assumption challenged:** AC-7 says "Evidence snippets and source locations support the extracted item they are attached to." ADR-0008 consequence 5 says "Evidence-grounding evals (AC-7) verify that items in `content` have corresponding records in `memory_evidence` where `quality_status IN ('strong_evidence', 'weak_evidence')`." But AC-7 only checks that evidence exists — not that the evidence actually supports the item's claim.

**Blind spot:** A processor could attach evidence snippets that are adjacent to the item in the transcript but do not actually support the item's claim. The BRD says "Prevent extracted items from being accepted as supported unless their evidence snippets and locations actually support the item" (FR-4). But there is no mechanism to evaluate whether evidence "actually supports" the claim — only whether it exists. This is the difference between syntactic presence and semantic support.

**Question to resolve:** Is AC-7 checking syntactic presence (evidence exists) or semantic support (evidence proves the item)? The BRD uses "actually support" which implies semantic support. The eval method must match.

---

### [A3] Semantic similarity threshold for conflict detection is a critical parameter left as "to be determined"

**Assumption challenged:** ADR-0007 says conflict detection uses "semantic similarity above a threshold" but does not specify the threshold. The BRD says "strict evidence eval" for change-aware decisions (FR-6). The combination of vague thresholds and high "strict evidence" requirements creates a gap between the stated requirements and the implementable spec.

**Blind spot:** What if the semantic similarity threshold is set incorrectly? If the threshold is too low (e.g., 0.6 cosine similarity), many items that should conflict are merged silently. If the threshold is too high (e.g., 0.95), almost everything becomes a conflict. The BRD makes strong promises about conflict detection accuracy but leaves the key parameter undefined.

**Question to resolve:** Specify the similarity threshold with empirical justification (e.g., "threshold of 0.85 corresponds to human inter-annotator agreement of X% on a held-out evaluation set"). This is a prerequisite for implementing AC-9.

---

### [A4] "Safe automatic matching" of prior memories is defined by algorithm, not by accuracy measurement

**Assumption challenged:** FR-12 describes the constrained matching signals (same participant, similar title, close chronology). ADR-0007 defines the exact criteria (Levenshtein distance <= 3, 90 days / 7 days). But there is no accuracy metric defined: what is the expected precision and recall of this matching algorithm? How many false positives (wrong prior memories matched) and false negatives (missed relevant prior memories) are acceptable?

**Blind spot:** "Safe automatic matching" is a qualitative claim. Without a quantitative target, the implementation team has no target to hit and no way to know when the matching is "safe enough." The BRD's deferral of broader matching to a future BRD suggests the team recognizes the limitations of the constrained approach — but it does not define what "safe enough" means for Phase 2.

**Question to resolve:** Define a target accuracy for prior-memory matching (e.g., precision >= 0.9, recall >= 0.7 on a representative evaluation set). Add this as an acceptance criterion.

---

### [A5] Processing is "non-blocking" for import, but queue depth under load is not evaluated

**Assumption challenged:** AC-2 verifies that import redirect completes before processing starts. The NFR says "processing must be asynchronous, observable, and non-blocking for meetings up to the BRD-02 50,000 character transcript-plus-notes limit." This is a per-meeting latency requirement. But what happens when the queue depth grows beyond worker capacity? The system degrades "gracefully" (queue/retry/retry-exhausted), but no bound is given on how long a job can wait.

**Blind spot:** If processing takes 30 seconds per meeting and 1000 meetings are imported per day, the queue will process 2880 seconds worth of jobs per day. If jobs arrive faster than workers can drain them, queue depth grows indefinitely until workers add capacity. There is no backpressure mechanism described — no way to signal to the import path that processing is saturated.

**Question to resolve:** Define queue depth limits and backpressure behavior. What happens when the queue exceeds a depth threshold? Does import fail? Does processing get deferred? Add to NFR table.

---

### [A6] "Briefing-readiness" signal excludes stakeholder notes and next recommended focus without justification

**Assumption challenged:** FR-15 says "Allow stakeholder notes and next recommended focus to be insufficient without automatically blocking core briefing-readiness." The rationale is that stakeholder notes and next recommended focus are "nice to have" rather than essential for a briefing. But this decision is not justified in the BRD.

**Blind spot:** Why are stakeholder notes not core to a briefing? If the purpose of ContextPilot is to help users prepare for meetings by surfacing relevant context from previous conversations, stakeholder notes about a person's preferences, concerns, and commitments seem highly relevant to a pre-call briefing. Excluding them from briefing-readiness may produce briefings that lack critical relationship context.

**Question to resolve:** Add explicit rationale for why stakeholder notes and next recommended focus are excluded from briefing-readiness blocking criteria. If the rationale is "they enhance the briefing but aren't essential," document it. If the rationale is "their evidence quality is less reliable," document that too.

---

### [A7] Active version activation on reprocessing success — what if the new version is worse?

**Assumption challenged:** FR-9 says "Make the latest successful memory version active by default." ADR-0006 describes atomic activation: deactivate prior, insert new, activate. FR-31 says "Failed, retrying, or retry-exhausted runs do not replace the previous active successful memory version." This is correct — but what about a reprocess that succeeds but produces a lower-quality memory (more insufficient-evidence categories, more conflicts)?

**Blind spot:** The activation logic only checks that the run "succeeded" (status = completed, completed_with_insufficient_evidence) — not that it improved over the prior version. A reprocess that succeeds but degrades memory quality would still replace the active version. There is no mechanism for the user to compare versions and rollback to a better one (rollback is listed in Could Have but not specced).

**Question to resolve:** Either specify that version comparison/rollback is out of scope for BRD-03 (accepted risk), or add it as a required feature. AC-10 does not cover this scenario.

---

### [A8] The "no guessing" principle (FR-3) is stated as absolute but has no defined boundary

**Assumption challenged:** FR-3 says "The system must not fill weak categories with guesses." The BRD says insufficient evidence categories should be marked "Insufficient evidence" rather than omitted or guessed. But what counts as a "guess"? Is a summary generated from a single sentence of transcript a "guess"? Is an "open questions" list generated from a meeting with no explicit questions a "guess"?

**Blind spot:** The "no guessing" principle is stated absolutely but has no defined boundary. An aggressive processor could generate confident-sounding summaries from minimal evidence and claim they are not "guesses" because every statement is technically extractable from the transcript. The BRD's quality statuses (Strong evidence / Weak evidence / Insufficient evidence) are the right framework, but the line between "weak evidence" and "guessing" is not defined.

**Question to resolve:** Define the minimum evidence threshold for each quality status. E.g., "Weak evidence: at least one relevant snippet with direct textual support. Insufficient evidence: no relevant snippets with direct textual support." Without this, processors cannot be evaluated consistently.

---

### [A9] Processing SLO is "architecture-defined during curation" — Phase 2 implementation cannot proceed without it

**Assumption challenged:** The NFR table and OQ-2 both say the processing latency target is "deferred to curation." But the BRD contains 24 ACs, 7 user stories, 4 ADRs, and a full observability spec — all of which are implemented assuming the processing SLO exists. Deferring the SLO to curation means the entire BRD is built on a missing foundation.

**Blind spot:** AC-2 (non-blocking import) is verifiable without a processing SLO. AC-22 (degradation behavior) is verifiable without a processing SLO. But AC-21 (transient failure retry up to 3 times) is time-dependent: how long between retries? The exponential backoff (2^retry_count seconds) is defined, but the initial retry delay is implied by the backoff formula. More importantly, without a processing SLO, no histogram bucket definitions can be committed, no performance tests can be written, and no SLO alerting thresholds can be set.

**Question to resolve:** This is a curation blocker, not a validation gap. Surface to PM as a required decision before Phase 2 implementation begins. OQ-2 must be resolved.

---

### [A10] BRD-03 blocks BRD-04 (Pre-Call Briefing) but BRD-04's briefing format is not specified

**Assumption challenged:** BRD-03 Relations section says "Blocks: BRD-04 Pre-Call Briefing because briefings require trusted, source-grounded memory." BRD-04 is listed in the backlog as "Todo." The BRD-04 spec does not exist yet (only in _drafts). The BRD-03 memory structure (7 categories, evidence, quality status, version history) is designed to power briefings — but the briefing output format is not defined.

**Blind spot:** What if the briefing format requires a different memory structure than what BRD-03 produces? If BRD-04 needs a different aggregation of memory items (e.g., a timeline view vs. a category view), the BRD-03 data model may need to change. Without BRD-04 drafted, BRD-03 is designing to an unknown specification.

**Question to resolve:** Draft BRD-04 before finalizing BRD-03 data model, or add a note to BRD-03 specifying that the memory JSON schema may need to evolve as BRD-04 briefing format is defined.

---

## Summary Table

| ID | Assumption | Challenge | Risk | Severity |
|----|-----------|-----------|------|----------|
| A1 | Single-pass extraction architecture | Locks in implementation pattern before provider selected | May exclude better architectures | Medium |
| A2 | AC-7 checks semantic support | Only checks syntactic evidence presence | Hallucination undetected | High |
| A3 | Semantic similarity threshold TBD | Key conflict detection parameter undefined | False positive/negative explosions | High |
| A4 | "Safe" matching is qualitative | No precision/recall target defined | Unknown false match rate | Medium |
| A5 | Non-blocking implies bounded wait | Queue depth unbounded, no backpressure | System degrades silently | Medium |
| A6 | Stakeholder notes excluded from briefing-readiness | No explicit rationale for exclusion | Briefings may lack critical context | Low |
| A7 | "Successful" reprocess activates | No quality comparison before activation | Worse version becomes active | Medium |
| A8 | "No guessing" is absolute | No minimum evidence threshold defined | Hallucination by another name | High |
| A9 | Processing SLO deferred | Foundation missing for all implementation | Phase 2 cannot proceed | High |
| A10 | BRD-03 blocks BRD-04 | Memory model designed to unknown briefing format | Data model may need rework | Medium |

---

## Recommendations

1. **A2, A8 (High):** Define minimum evidence thresholds for each quality status. AC-7 eval must check semantic support, not just syntactic presence.
2. **A3 (High):** Specify the semantic similarity threshold with empirical justification before AC-9 implementation.
3. **A9 (High):** Resolve OQ-2 (processing SLO) as a prerequisite for Phase 2 implementation. This is a curation decision, not an architectural one.
4. **A4 (Medium):** Add a precision/recall target for prior-memory matching accuracy.
5. **A5 (Medium):** Define queue depth limits and backpressure behavior for the NFR table.
6. **A7 (Medium):** Consider adding version quality comparison before activation, or explicitly accept this as a risk for Phase 2.
7. **A10 (Medium):** Draft BRD-04 before finalizing BRD-03 data model, or add schema evolution caveat.