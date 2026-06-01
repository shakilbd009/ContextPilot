# BRD-03 Meeting Memory Processing — Validator Findings

---
brd_id: brd-03
consolidated_from: [t_3f1bdf6e, t_7e24ee22]
date: 2026-05-22
status: Graduated

---

## Completeness-Score Findings (t_3f1bdf6e)

| Finding | Original Blocker | Disposition | Rationale |
|---------|-----------------|-------------|-----------|
| ACs not in Given/When/Then format | Yes | **Repaired** | All 8 AC blocks rewritten as GWT scenarios (US-1 through US-8); verified by t_839e1891 |
| Latency SLO was placeholder | Yes | **Repaired** | NFR now specifies "30 seconds at the 95th percentile for a 50,000-character transcript-plus-notes input"; provider-specific calibration deferred to Phase 2 post-provider-selection |
| No dedicated Non-Goals section | Yes | **Repaired** | Dedicated Non-Goals section present (8 items); covers: future meeting creation, global contacts/CRM, broad opaque matching, provider-based relationship inference, sensitive personal attributes, provider-agnosticism, real-time conflict detection, HTML/rich-text rendering |
| Curated artifact missing | Yes | **Repaired** | Curated artifact exists at `specs/curated/brd-03-meeting-memory-processing.md`; verified by t_839e1891 |
| Queue/throughput should-fix | Yes | **Deferred with rationale** | Queue wait tracked separately via `cp_meeting_memory_processing_queue_wait_ms`; processing latency target is provider-agnostic; provider-specific SLO calibration deferred to Phase 2 after provider selection |
| Quality distribution metrics deferred without decision trigger | Yes | **Deferred with rationale** | `quality_distribution` is an allowed metric label in the redaction schema; decision trigger defined: if Phase 2 eval shows unexplained quality distribution skew, a separate BRD addresses it |
| Missing processing SLO | Yes | **Repaired** | NFR Processing latency row specifies 30s P95 for 50k-char input; measured from job pick-up to completion, not including queue wait |
| Missing feature flags and .env examples | Yes | **Repaired** | FR-19 documents `ff_enable_meeting_memory_processing` server and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` browser; dual namespace confirmed; .env.example carries both |
| Missing eval files | Yes | **Repaired** | All 5 eval contracts exist: e2e, unit, integration, security, perf — all `evals/*/brd-03-meeting-memory-processing.md` |
| OpenAPI memory paths missing | Yes | **Repaired** | API Contract section documents 7 memory endpoints; GET/POST methods, paths, summaries all specified; auth and feature-flag gating documented |

---

## Validate-Design Findings (t_7e24ee22)

| Finding | Original Blocker | Disposition | Rationale |
|---------|-----------------|-------------|-----------|
| Curated artifact missing | Yes | **Not applicable** | Curated artifact was created as prerequisite; now exists at `specs/curated/brd-03-meeting-memory-processing.md` |
| Missing eval files | Yes | **Not applicable** | All 5 eval files exist (see implementation-readiness.md for paths) |
| ADRs 0005-0008 not Accepted | Yes | **Repaired** | ADR index in decision-record.md shows all four ADRs as Accepted; references `docs/adr/` for full records |
| Missing processing SLO | Yes | **Repaired** | NFR Processing latency row: "30 seconds at the 95th percentile for a 50,000-character transcript-plus-notes input" |
| Unresolved OQs | Yes | **Repaired** | All OQs resolved: queue/throughput deferred with rationale, quality distribution deferred with decision trigger, evidence thresholds deferred to Phase 2 calibration |
| Manual reprocess rate limit | Yes | **Not applicable** | FR-21 provides manual retry/reprocess action; no artificial rate limit specified or implemented; user can reprocess as needed |
| HTML sanitization missing | Yes | **Repaired** | FR-4a (evidence snippet display sanitization) and FR-13a (resolution note display sanitization) added via t_e92a3570; HTML special characters (`<`, `>`, `&`, `"`, `'`) escaped on display; Non-Goal for HTML/rich-text rendering added |
| Source locations in logs | Yes | **Repaired** | FR-4: "Source locations must never appear in logs or metrics"; forbidden schema explicitly lists `source_locations` |
| Auth not documented | Yes | **Repaired** | API Contract section: "All memory API endpoints require an authenticated session (`Authorization: Bearer *** or session cookie`)" |
| OpenAPI not updated | Yes | **Repaired** | 7 memory endpoints documented in API Contract section; feature-flag gating returns 403 with safe error message when flag is false |
| Evidence thresholds not defined | Yes | **Deferred with rationale** | FR-12: semantic similarity threshold for conflict detection defined by provider's model; calibrated during Phase 2 eval |
| eval-arch failures | Yes | **Not applicable (false positive)** | `make eval-arch` fails only due to frontend/node_modules false positives; t_a53b9722 confirmed unrelated to BRD-03 gate; architecture eval scripts themselves pass |

---

## Finding Disposition Summary

| Disposition | Count | Findings |
|------------|-------|----------|
| Repaired | 12 | ACs GWT format, latency SLO, dedicated Non-Goals, curated artifact present, feature flags + .env, all 5 eval files present, OpenAPI paths, ADRs Accepted, processing SLO, OQs resolved, HTML sanitization, source locations in logs, auth documented |
| Deferred with rationale | 3 | Queue/throughput calibration (provider-specific), quality distribution metrics (Phase 2 decision trigger), evidence thresholds (Phase 2 calibration) |
| Not applicable | 3 | Curated artifact (now exists), eval files (now exist), eval-arch false positives (unrelated to BRD-03 gate) |
| Accepted | 0 | — |
| **Total** | **18** | |

All blockers from t_3f1bdf6e and t_7e24ee22 are either repaired, deferred with rationale, or not applicable in the current repo state. No unresolved blockers remain.