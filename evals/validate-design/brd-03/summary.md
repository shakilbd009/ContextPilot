# Validation Summary: BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validators run:** security, architecture, performance, ux, devils-advocate
**Target:** specs/domain/brd-03-meeting-memory-processing.md
**Curated version:** does not exist — validated against specs/domain/brd-03-meeting-memory-processing.md

---

## Verdict: NEEDS_ATTENTION

| Validator | Verdict | Critical | High | Medium | Low |
|-----------|---------|----------|------|--------|-----|
| Security | NEEDS_ATTENTION | 0 | 2 | 3 | 1 |
| Architecture | NEEDS_ATTENTION | 1 | 4 | 4 | 2 |
| Performance | PASS | 0 | 0 | 3 | 1 |
| UX | NEEDS_ATTENTION | 0 | 3 | 4 | 1 |
| Devils Advocate | NEEDS_ATTENTION | 0 | 4 | 6 | 0 |
| **Total** | **NEEDS_ATTENTION** | **1** | **13** | **20** | **5** |

**Overall:** NEEDS_ATTENTION — 1 critical, 13 high, 20 medium, 5 low findings.

---

## Critical Issues (Must Fix)

### [C1] Processing latency target is deferred to curation — BRD-03 cannot be implemented without a concrete SLO

**Source:** architecture-findings.md [C1]
**Validator:** architect

The NFR table says "Processing latency | Architecture-defined during curation." Open Question 2 (OQ-2) says "curation must set a concrete processing latency target after model/provider/runtime architecture is selected." This means the BRD does not contain an implementable latency requirement. Implementation cannot begin without a concrete SLO.

**Fix required:** PM must make a latency SLO decision before Phase 2 implementation begins. This is a curation blocker, not an implementation gap.

---

## High Priority Issues (Should Fix)

**Security:**
- [H1] Authentication mechanism not specified for memory processing endpoints
- [H2] Audit trail for prior-memory input changes lacks protected storage specification

**Architecture:**
- [H1] `memory_conflicts.conflicting_items` JSONB may store raw memory content — redaction hazard
- [H2] `source_location` format not committed — multiple incompatible formats possible
- [H3] `check-no-sensitive-content.sh` architecture eval does not exist
- [H4] All four BRD-03 ADRs (ADR-0005, ADR-0006, ADR-0007, ADR-0008) are "Proposed" not "Accepted"

**UX:**
- [H1] Memory state UI labels and interactions not specced
- [H2] Evidence inspection component not specced
- [H3] Briefing-readiness signal location not specified

**Devils Advocate:**
- [A2] AC-7 checks syntactic evidence presence, not semantic support — hallucination undetected
- [A3] Semantic similarity threshold for conflict detection not specified
- [A8] "No guessing" principle is stated as absolute but has no minimum evidence threshold
- [A9] Processing SLO is deferred to curation — Phase 2 cannot proceed without it

---

## Medium Priority Issues

**Security:**
- [M1] Flag misconfiguration metric name not specified
- [M2] No rate limiting specified for memory processing endpoints
- [M3] ADR-0008 privacy architecture is "Proposed" not "Accepted"

**Architecture:**
- [M1] `blocking_conflicts` structure underspecified for multi-category conflicts
- [M2] Semantic similarity threshold for conflict detection not specified
- [M3] No OpenAPI endpoint definitions for memory processing/retry/conflict APIs
- [M4] `is_active` + `status` dual-column has no atomicity constraint

**Performance:**
- [M1] Processing duration histogram has no bucket definitions (SLO deferred)
- [M2] Queue wait time histogram has no target SLO
- [M3] Advisory lock contention risk not evaluated at scale

**UX:**
- [M1] Insufficient evidence copy not specified
- [M2] Conflict review queue access path not specced
- [M3] Prior-memory include/exclude flow not specced
- [M4] Version diff decision (ship vs defer) not made

**Devils Advocate:**
- [A1] Single-pass extraction architecture assumed — locks in implementation pattern
- [A4] "Safe" matching has no precision/recall target
- [A5] Queue depth unbounded, no backpressure mechanism
- [A6] Stakeholder notes excluded from briefing-readiness without explicit rationale
- [A7] "Successful" reprocess activates without quality comparison
- [A10] BRD-03 data model designed to unknown BRD-04 briefing format

---

## Phase 2 Implementation Prerequisites

Before Phase 2 implementation begins, the following must be resolved:

1. **C1 + A9: Processing latency SLO** — PM curation decision on OQ-2
2. **H4: Accept all four ADRs** — ADR-0005, ADR-0006, ADR-0007, ADR-0008 status updated from "Proposed" to "Accepted"
3. **Security H1: Auth mechanism** — specify authentication for memory processing endpoints, or reference a future auth BRD
4. **Architecture H2: source_location format** — commit to character offsets, line references, or another stable locator
5. **Architecture H3: check-no-sensitive-content.sh** — write the architecture eval before production enablement
6. **Architecture M3: OpenAPI endpoints** — define API contracts for memory processing, retry, conflict review, conflict resolution
7. **A3 + M2: Semantic similarity threshold** — specify with empirical justification
8. **A2 + A8: Evidence threshold definitions** — minimum evidence for each quality status

---

## Positive Findings

- Non-blocking import path correctly specced: fire-and-forget queue INSERT after transaction commit
- PostgreSQL advisory lock queue is Phase-appropriate (no new infra dependency)
- `MemoryProcessor` and `PriorMemoryMatcher` interfaces allow architecture evolution
- Privacy requirements are explicit and comprehensive (FR-25, FR-26)
- Dual-namespace feature flag correctly specced with server-authoritative enforcement
- Observability events and metrics well-defined with safe field names
- `ProcessingResult` struct as the sole emission boundary is architecturally correct
- Conflict isolation in separate table with review queue is a clean design
- Evidence-grounding requirement (FR-4, FR-5) is correctly strict
- Quality status enumeration is consistent and user-comprehensible
- Brief readiness signal design correctly excludes stakeholder notes and next recommended focus from blocking criteria

---

## Eval Coverage Assessment

**check-status-sync.sh result:** PASS

| Eval type | Status |
|-----------|--------|
| e2e | MISSING — no brd-03 eval file |
| unit | MISSING — no brd-03 eval file |
| integration | MISSING — no brd-03 eval file |
| perf | MISSING — no brd-03 eval file |
| security | MISSING — no brd-03 eval file |
| architecture | EXISTS — check-no-panic, check-no-background-context, check-feature-flags; check-no-sensitive-content.sh missing (H3 above) |

**Summary:** No BRD-03 eval files exist in evals/e2e/, evals/unit/, evals/integration/, evals/perf/, or evals/security/. The check-status-sync.sh does not validate eval file existence for BRDs outside the curated folder — but the task body explicitly requires eval coverage for all 8 user stories. This is a MUST FIX before Phase 2 QA execution.

---

## Recommendations

1. PM resolves OQ-2 (processing SLO) — prerequisite for all implementation
2. Architect accepts ADRs 0005-0008 before Phase 2 begins
3. Security review specifies auth mechanism for memory processing endpoints
4. Architecture commits source_location format
5. QA creates eval files for all 8 user stories before Phase 2 implementation (see eval coverage gap above)
6. Ops writes check-no-sensitive-content.sh before production enablement

---

## Next Steps

- [ ] Resolve [C1] — processing latency SLO decision (PM)
- [ ] Accept ADRs 0005-0008 (architect)
- [ ] Specify auth mechanism for memory processing endpoints (security)
- [ ] Commit source_location format (architect)
- [ ] Write check-no-sensitive-content.sh eval (ops)
- [ ] Add OpenAPI endpoint definitions for memory APIs (architect)
- [ ] Create eval files for all BRD-03 ACs (qa)
- [ ] Define semantic similarity threshold (architect)
- [ ] Define minimum evidence thresholds per quality status (architect)
- [ ] Specced UX states, evidence component, briefing-readiness signal location (ux)