# ADR-0007: Meeting Memory Processing — Prior Memory Matching & Conflict Detection

> Status: Accepted

## Context

BRD-03 FR-12 defines safe automatic matching of eligible prior memories using constrained signals (same participants, similar title, close chronology). FR-13 defines quality statuses including `Conflicting evidence`. FR-6 requires that conflicting evidence be placed in a review queue, excluded from normal memory categories and from downstream briefing eligibility. FR-14 defines conflict resolution: user writes an evidence-backed resolution note, original sources are preserved, a new memory version is created.

Key decisions needed:
1. What constitutes a "safe match" — exact criteria for the constrained signals
2. How conflict detection works — what triggers the `Conflicting evidence` quality status
3. How decision status (standalone/confirms/changes/reverses) is determined
4. How the review queue is populated and excluded from briefing eligibility

## Decision

### Prior Memory Matching (Safe Match)

A prior memory is eligible for use as input to current processing if **all** of the following hold:

| Signal | Criterion |
|--------|-----------|
| Same participant | At least one participant in the current meeting shares an exact `email` with a participant in the prior meeting |
| Similar title | Levenshtein distance between normalized titles is <= 3, or titles share a common token sequence of >= 4 consecutive words |
| Close chronology | Prior meeting `completed_at` is within 90 days before or 7 days after the current meeting's `completed_at` |

These three signals are ANDed — a match requires all three to be true.

**Match confidence**:
- `safe_match`: all three signals pass at the full criteria above
- `uncertain`: only `same_participant` and `close_chronology` pass (title similarity fails) — shown to the user for confirmation before use

**Exclusion/inclusion by user**: The user can exclude a system-matched prior memory or add an un-matched prior memory (FR-5, AC-16). These user actions set `included_by_user = TRUE` or `excluded_by_user = TRUE` in `memory_prior_memory_inputs` and trigger reprocessing.

**Out of scope per BRD**: Broad opaque matching (embedding similarity, provider-based relationship inference, cross-system relatedness) is deferred to the future Related Meeting Detection BRD.

### Conflict Detection

A **conflict** is detected when all of the following hold:
1. The current processing run extracts a memory item (in `decisions`, `action_items`, `risks/blockers`, or `summary`) in a category that also has an item in the matched prior memory.
2. Both the current item and the prior item have `quality_status` of `strong_evidence` or `weak_evidence` (neither is `insufficient_evidence`).
3. The item content is semantically different — specifically:
   - For `decisions`: the decision statements disagree (one says do X, the other says do not X, or they reach different conclusions from the same evidence)
   - For `action_items`: the action descriptions refer to the same subject but differ in owner, due date, or stated status in a way that cannot be merged
   - For `risks_blockers`: the risk statements describe the same risk but with opposite polarity or materially different severity assessment
   - For `summary`: the summary statements cover the same topic but attribute different outcomes or states

The conflict detection algorithm:
1. Process the current meeting with matched prior memories as input.
2. For each extracted item, check whether the prior memory (referenced via `memory_prior_memory_inputs`) has a corresponding item in the same category that covers the same subject (exact match on entity/topic is not required — semantic similarity above a threshold triggers conflict evaluation).
3. If both have strong/weak evidence and semantic similarity is below the merge threshold, flag as `Conflicting evidence`.

**Note**: The BRD says "strict evidence eval" (FR-18) — the system must not generate a conflict when the prior memory item has `Insufficient evidence`. An item with insufficient evidence cannot participate in a conflict because it has no authoritative claim.

**Decision status derivation** (FR-6):
- `standalone`: current meeting evidence stands on its own; no prior memory item in the same category with overlapping subject
- `confirms prior decision`: current item agrees with prior item (semantic similarity above threshold, same polarity)
- `changes prior decision`: current item disagrees on the outcome but both have evidence; triggers `Conflicting evidence` if the disagreement is material
- `reverses prior decision`: current item directly contradicts prior item (explicit negation detected); triggers `Conflicting evidence`

### Conflict Review Queue

When conflict is detected:
1. The conflicting items from both current and prior memory are written to `memory_conflicts` with `review_status = 'pending'`.
2. The items are **excluded** from the normal `memory_versions.content` categories — they appear only in the conflict review queue.
3. The memory version's `status` is set to `active` (it is still usable), but the `briefing_readiness` signal in `content` is updated to reflect the unresolved conflict (FR-15).
4. The meeting's briefing eligibility is blocked for the conflicting categories only — non-conflicting categories remain briefing-eligible.

### Conflict Resolution

When a user resolves a conflict (FR-14):
1. User writes a resolution note (evidence-backed or explicitly states evidence is insufficient).
2. `memory_conflicts.review_status` is set to `reviewed`, `resolution_note` is stored, `resolved_at` and `resolved_by` are recorded.
3. A new memory version is created (`version_number = previous + 1`, `status = 'active'`, `is_active = TRUE`).
4. The new version's content includes the conflict resolution as a `conflict_resolution_note` field on the affected items, and the `memory_conflicts` record is preserved for audit.
5. The original conflicting sources are never deleted — they are preserved alongside the resolution note.

## Rationale

**Constrained signals only (no opaque matching)**: The BRD explicitly limits prior-memory matching to same-participant, similar-title, close-chronology (FR-12). This is a deliberate product decision to avoid the trust and privacy issues that come with broad embedding-based relatedness detection. The three-signal AND approach is stricter than any single signal alone, reducing false positive matches that could contaminate processing.

**Email-based participant matching**: Email is the stable identifier for a participant across meetings (BRD-02's participant model uses `email` as an optional field). Using email for matching avoids name disambiguation complexity while satisfying the "same participants" signal.

**Semantic similarity for conflict detection**: Literal string matching would miss paraphrased disagreements. Semantic similarity (e.g., embedding cosine distance) detects disagreements even when the wording differs. We use a threshold approach: above the merge threshold = agree; below the merge threshold but both have evidence = conflict.

**Conflict isolation in separate table**: Storing conflicts in `memory_conflicts` (ADR-0006) rather than inline in the memory JSON achieves three things: (1) clean exclusion from briefing eligibility via `review_status = 'pending'` filter, (2) auditability (original sources preserved), (3) the review queue is a simple query, not JSON parsing.

**Resolution creates a new version**: FR-14 and AC-19 require a new memory version after conflict resolution. This is consistent with the version-per-processing-run model (ADR-0006) and ensures the audit trail includes the resolution note and its author.

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| Richer matching signals | Embedding similarity, CRM-style relationship history, cross-system relatedness are out of scope — deferred to future Related Meeting Detection BRD |
| Automated conflict resolution | The system flags conflicts but does not attempt automated resolution; this is intentional per FR-13 ("cannot be reconciled automatically") |
| Real-time conflict detection | Conflict detection happens at processing time; a conflict arising from a later edit to a prior memory is caught only when the affected meeting is reprocessed |
| Title similarity granularity | Levenshtein distance of 3 is a simple heuristic; shared-n-gram or embedding similarity could be more accurate but adds complexity and is deferred |

## Consequences

1. The matching algorithm is implemented in the `PriorMemoryMatcher` interface:
   ```go
   type PriorMemoryMatcher interface {
       FindMatches(ctx context.Context, currentMeeting *Meeting) ([]PriorMemoryMatch, error)
   }
   ```
2. The conflict detector runs as part of the `MemoryProcessor` pipeline (ADR-0005), after extraction but before the memory version is finalized.
3. The review queue endpoint (`GET /meetings/:id/memory/conflicts`) queries `memory_conflicts` where `meeting_id = ? AND review_status = 'pending'`.
4. The conflict resolution endpoint (`POST /meetings/:id/memory/conflicts/:conflict_id/resolve`) updates the conflict record and triggers reprocessing.
5. Briefing eligibility is computed as: meeting has `active_memory_version_id` pointing to a version where `briefing_readiness.core_categories_ready = true` and `briefing_readiness.blocking_conflicts = []` for the requested categories.
6. Observability (ADR-0008) emits `memory.conflict.detected` with safe identifiers — meeting IDs, category, prior memory IDs — never the actual conflicting content.

## Alternatives Considered

### Embedding-based prior memory matching
Rejected because: The BRD explicitly scopes matching to same participants, similar title, close chronology (FR-12). Embedding similarity is "broad opaque matching" deferred to a future BRD. Additionally, embedding-based matching of meeting content to find related memories would require processing that content — which itself may contain sensitive decisions that should not be used for cross-meeting inference without explicit consent.

### Semantic similarity threshold — provider-calibrated
The semantic similarity threshold used for conflict detection is **defined by the processing provider's semantic similarity model** — the `MemoryProcessor` implementation compares item embeddings (cosine similarity or equivalent) and returns `Conflicting evidence` when similarity falls below the merge threshold for item pairs that both have `strong_evidence` or `weak_evidence` in the same category.

The exact threshold (e.g., cosine similarity < 0.78) is provider-specific and will be calibrated during Phase 2 eval. The BRD-03 architecture defines the mechanism — the threshold is an implementation calibration, not a product requirement.

### Automated conflict resolution
Rejected because: FR-13 says "cannot be reconciled automatically" when both sides have source support. Automated resolution would require the system to make an authoritative judgment about which evidence is better — outside the scope of BRD-03.

### String-match conflict detection
Rejected because: Two statements that disagree semantically can be worded identically while having different implied commitments. String matching produces false negatives (misses conflicts) and false positives (flags identical statements as conflicts when they refer to different situations). Semantic similarity is the appropriate primitive.

## Date

2026-05-21