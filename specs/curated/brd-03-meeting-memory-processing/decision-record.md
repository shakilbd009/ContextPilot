# BRD-03 Meeting Memory Processing — Decision Record

---
brd_id: brd-03
created: 2026-05-22
approver: pm
status: Graduated

---

## Gate Decisions

| Gate Task | Validator | Verdict | Date | Key Evidence |
|-----------|-----------|---------|------|-------------|
| t_328a02ab (completeness-score) | completeness-score | **Approve** | 2026-05-21 | Curated artifact exists; 8 GWT AC blocks present; dedicated Non-Goals present; latency SLO 30s P95 for 50k-char input; feature flags and .env examples present; 5 eval contracts present; OpenAPI memory paths found |
| t_a53b9722 (PM gate) | pm | **Approve** | 2026-05-22 | Confirms t_328a02ab prerequisite approved; validate-design blockers from t_7e24ee22 resolved or explicitly deferred in current repo state |
| t_7e24ee22 (validate-design) | validator | **Approve** | 2026-05-22 | All blockers resolved or deferred; eval architecture checks pass; no unresolved OQs remain |

---

## Open Questions (OQ) Resolutions

No OQs remain open in the current repo state. All items originally flagged by completeness-score and validate-design have been addressed:

| OQ | Source | Resolution | Disposition |
|----|--------|------------|-------------|
| Queue/throughput SLO | t_3f1bdf6e | Specified as 30s P95 for 50,000-character input; provider-specific calibration deferred to Phase 2 post-provider-selection; queue wait tracked separately via `cp_meeting_memory_processing_queue_wait_ms` | **Deferred with rationale** — provider-specific calibration requires provider selection (Phase 2) |
| Quality distribution metrics | t_3f1bdf6e | `quality_distribution` tracked in redaction schema as allowed metric label; explicit decision trigger: if Phase 2 eval shows quality distribution skews unexplained by evidence quality, a separate BRD addresses it | **Deferred with rationale** — Phase 2 eval will determine if additional quality distribution metrics are needed |
| HTML sanitization | t_7e24ee22 | FR-4a (evidence snippet display sanitization) and FR-13a (resolution note display sanitization) added; Non-Goal for HTML/rich-text rendering added; minimum protection is HTML-escaping until allowlist sanitization is introduced | **Repaired** |
| Source locations in logs | t_7e24ee22 | FR-4: "Source locations must never appear in logs or metrics." Forbidden schema explicitly lists `source_locations` | **Repaired** |
| Auth/OpenAPI | t_7e24ee22 | API contract section documents auth requirement (`Authorization: Bearer or session cookie`); all 7 memory endpoints documented with method/path/summary; feature flag gating documented (403 when false) | **Repaired** |
| Manual reprocess rate limit | t_7e24ee22 | FR-21: manual retry/reprocess action provided for failed/retry-exhausted/stale states and user corrections; no artificial rate limit beyond standard auth | **Not applicable** — no rate limit specified; user can reprocess as needed |
| Evidence thresholds | t_7e24ee22 | FR-12 specifies conflict detection uses provider's semantic similarity model; exact cosine-similarity threshold calibrated during Phase 2 eval | **Deferred with rationale** — provider-specific calibration required |

---

## ADR References

| ADR | Title | Status | Relevance |
|-----|-------|--------|-----------|
| ADR-0005 | Queue Architecture | Accepted | Defines `memory_processing_jobs` table and async processing model |
| ADR-0006 | Data Model | Accepted | Defines `memory_versions`, `memory_evidence`, `memory_conflicts`, `memory_prior_memory_inputs` schema |
| ADR-0007 | Prior Memory Matching + Conflict Detection | Accepted | Defines constrained matching (FR-5), `changeStatus` field (FR-6), and conflict detection (FR-12) |
| ADR-0008 | Observability + Privacy | Accepted | Defines metrics, log events, and redaction schema (FR-20) |

---

## Trade-offs

| Decision | Trade-off | Rationale |
|----------|-----------|-----------|
| Provider-agnostic BRD vs. specific implementation | Harder to specify exact latency SLO without provider knowledge | FR-17; NFR states provider-agnostic 30s P95 target with Phase 2 calibration after provider selection |
| Constrained prior-memory matching (3-signal AND) vs. broad embedding similarity | Users may miss some relevant prior memories | Explicit Non-Goal: broad opaque matching deferred to future Related Meeting Detection BRD; 3-signal approach is verifiable and privacy-safe |
| Conflict detection via provider similarity threshold vs. deterministic rules | Exact conflict boundary is provider-specific | FR-12: threshold defined by provider's semantic similarity model, calibrated during Phase 2 eval |
| Character-offset source locations vs. token offsets | Offsets invalidated on whitespace normalization | FR-4: offsets computed against stored (possibly whitespace-normalized) text; stale detection via `updated_at` comparison ensures reprocessing when source changes |
| Evidence snippet HTML escaping vs. full allowlist sanitization | Full rich-text not yet supported | FR-4a + FR-13a: minimum HTML-escaping required; Non-Goal deferred to separate BRD if rich text rendering later introduced |

---

## Source Task References

- **t_328a02ab** (completeness-score approve): Parent of t_3f1bdf6e; verdict confirmed all 8 GWT AC blocks present, dedicated Non-Goals present, latency SLO specified, feature flags and eval contracts present
- **t_a53b9722** (PM gate approve): Parent PM gate confirming t_328a02ab approved and t_7e24ee22 blockers resolved/deferred
- **t_3f1bdf6e** (completeness-score original blockers): ACs not GWT, latency placeholder, no dedicated Non-Goals, curated artifact missing, queue/throughput should-fix, quality distribution metrics deferred without decision trigger — all repaired or deferred with rationale in current repo
- **t_7e24ee22** (validate-design original blockers): missing curated artifact, missing eval files, ADRs 0005-0008 not Accepted, missing processing SLO, unresolved OQs, manual reprocess rate limit, HTML sanitization/source-location/auth/OpenAPI/evidence thresholds — all resolved or deferred in current repo
- **t_839e1891** (repair task): Confirmed all 4 Must Fix items from completeness-score t_3f1bdf6e closed
- **t_e92a3570** (XSS sanitization repair): Added FR-4a, FR-13a, and Non-Goal for HTML/rich-text rendering
- **t_a16cb693** (XSS sanitization verify): Confirmed all 3 HTML special character handling requirements met

---

## Graduated Package

| File | Purpose |
|------|--------|
| `brd.md` | Canonical build spec — no unresolved OQs, TBDs, or blocker language |
| `decision-record.md` | This file — gate decisions, OQ resolutions, trade-offs, ADR status |
| `validator-findings.md` | Consolidated completeness-score and validate-design findings with dispositions |
| `implementation-readiness.md` | Eval/flag/contract evidence, production prerequisites, risks, rollback notes |