# BRD-03 Meeting Memory Processing — Implementation Readiness

---
brd_id: brd-03
date: 2026-05-22
status: Graduated

---

## Eval Contracts

All five mandatory eval contracts exist under `evals/` and reference BRD-03 acceptance criteria:

| Eval Type | Path | Status |
|-----------|------|--------|
| E2E | `evals/e2e/brd-03-meeting-memory-processing.md` | Present |
| Unit | `evals/unit/brd-03-meeting-memory-processing.md` | Present |
| Integration | `evals/integration/brd-03-meeting-memory-processing.md` | Present |
| Security | `evals/security/brd-03-meeting-memory-processing.md` | Present |
| Performance | `evals/perf/brd-03-meeting-memory-processing.md` | Present |

---

## Feature Flag Status

| Flag | Type | Default | Server Env | Browser Env | Status |
|------|------|---------|------------|-------------|--------|
| `ff_enable_meeting_memory_processing` | boolean | `false` | `FF_ENABLE_MEETING_MEMORY_PROCESSING` | `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` | Planned |

Dual namespace registration confirmed in `specs/feature-flags.md` (Phase 2, Planned).

---

## Architecture Fitness Checks

### scripts/check-status-sync.sh

```
=== ContextPilot Sync Check ===
[1/7] Checking STATUS.md backlog spec paths...
[2/7] Checking eval file BRD references...
[3/7] Checking feature flag parity (specs/feature-flags.md vs .env.example)...
[4/7] Checking STATUS.md decision log completeness...
[5/7] Checking feature-flags.md for TBD entries...
[6/7] Checking architecture script executability...
[7/7] Checking curated BRDs have corresponding eval files...
[8/7] Checking .env.example defaults...
OK: all sync checks passed
```

**Result: PASS**

### evals/architecture/check-no-sensitive-content.sh

```
OK: no hardcoded sensitive meeting-derived content in logs/metrics
```

**Result: PASS**

### make eval-arch

`make eval-arch` fails only due to `frontend/node_modules` false positives unrelated to BRD-03 gate. Per t_a53b9722 (PM gate), this is a pre-existing issue outside the scope of the BRD-03 graduation gate. The architecture eval scripts themselves pass independently.

---

## Production Checklist Prerequisites

| Prerequisite | Status | Notes |
|--------------|--------|-------|
| Curated BRD artifact exists | ✅ | `specs/curated/brd-03-meeting-memory-processing.md` |
| GWT Acceptance Criteria | ✅ | 8 GWT blocks (US-1 through US-8) |
| Dedicated Non-Goals section | ✅ | 8 items covering out-of-scope areas |
| Latency SLO defined | ✅ | 30s P95 for 50k-character transcript-plus-notes input |
| Feature flags registered | ✅ | Dual namespace in specs/feature-flags.md |
| .env.example defaults | ✅ | Both `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` and `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=false` |
| Five eval contracts present | ✅ | e2e, unit, integration, security, perf |
| OpenAPI memory paths | ✅ | 7 endpoints documented in API Contract section |
| ADR-0005 through ADR-0008 | ✅ | All Accepted |
| Sync check passes | ✅ | scripts/check-status-sync.sh exits 0 |
| Sensitive content check passes | ✅ | evals/architecture/check-no-sensitive-content.sh exits 0 |

---

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Provider-specific calibration required for latency SLO | Medium | Phase 2 eval will calibrate; target is provider-agnostic with 30s P95 as planning estimate |
| Provider-specific semantic similarity threshold for conflict detection | Medium | FR-12 defines the model interface; exact threshold calibrated during Phase 2 eval |
| Quality distribution metrics may need expansion | Low | Decision trigger: if Phase 2 eval shows unexplained skew, separate BRD addresses it |
| Retention/deletion policy not yet defined | Medium | Deferred to BRD-06; memory versions retained until BRD-06 defines policy |

---

## Rollback Notes

| Scenario | Rollback Procedure |
|----------|--------------------|
| Processing worker failure | Jobs remain `queued`; existing active memory continues to serve; meeting detail shows `queued` state; no user content lost |
| Provider outage | `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` disables processing; manual import continues without memory processing; existing memory remains accessible |
| Bad memory version | Failed processing does not replace active successful memory version; user can manually reprocess via FR-21 |
| Stale source | Source edit detection marks memory `stale` and queues reprocessing; previous active version serves briefing until new version succeeds |
| Data corruption | `memory_versions` entries are immutable; new processing creates new version; `memory_conflicts` audit trail preserved |

---

## Deferred Items

| Item | Deferred To | Rationale |
|------|-------------|-----------|
| Provider-specific latency SLO calibration | Phase 2 (post-provider-selection) | Cannot calibrate without concrete provider |
| Semantic similarity threshold for conflict detection | Phase 2 eval | Provider-specific; calibrated during Phase 2 |
| Quality distribution metric expansion criteria | Phase 2 eval | Decision trigger: unexplained skew triggers separate BRD |
| Retention, deletion, export, redaction policy | BRD-06 | Memory versions retained until BRD-06 defines policy |
| Full allowlist sanitization (DOMPurify) | Future BRD if rich text rendering introduced | FR-4a + FR-13a provide minimum HTML-escaping; Non-Goal documented |