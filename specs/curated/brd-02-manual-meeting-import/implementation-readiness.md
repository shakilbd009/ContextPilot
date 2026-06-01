# BRD-02 Manual Meeting Import — Implementation Readiness

---
brd_id: brd-02
title: Manual Meeting Import
date: 2026-05-22
---

## Eval Files

| Eval type | Path | AC coverage |
|-----------|------|-------------|
| E2E | `evals/e2e/brd-02-manual-meeting-import.md` | AC-1 through AC-18 (all 18 scenarios) |
| Unit | `evals/unit/brd-02-manual-meeting-import.md` | Validation, API contract, flag evaluation |
| Integration | `evals/integration/brd-02-manual-meeting-import.md` | Content limits, participant storage, transaction isolation |
| Security | `evals/security/brd-02-manual-meeting-import.md` | Flag bypass, injection, redaction schema verification |
| Performance | `evals/perf/brd-02-manual-meeting-import.md` | Save latency, content size histogram, flag eval duration |

Eval file paths are relative to the project root `/Users/shakilakram/projects/ContextPilot`.

---

## Feature Flag Status

| Flag | Server env | Browser env | Default | Status |
|------|-----------|-------------|---------|--------|
| `ff_enable_manual_meeting_import` | `FF_ENABLE_MANUAL_MEETING_IMPORT` | `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT` | `false` | Registered in `specs/feature-flags.md` (Phase 1, Planned) |

**Flag evaluation architecture check:** `make eval-arch` (or `scripts/check-feature-flags.sh`) must pass before the feature flag can be enabled. This verifies that every `ff_*` reference in source code corresponds to a registration in `specs/feature-flags.md`.

---

## Contract Evidence

### OpenAPI Contract

**File:** `contracts/openapi.yaml`

**Status:** Gap identified — `MeetingSummary` schema (with `start_time`, `end_time`, `attendees[]`) is incompatible with BRD-02's `completedAt`, `transcript`, `notes`, `contentSource` data model.

**Required action:** Add a new `Meeting` schema to `contracts/openapi.yaml` for the `POST /meetings` write path before Phase 1 implementation. This is a Phase 1 prerequisite listed in both `brd.md` Pre-Phase 1 Implementation Prerequisites and `decision-record.md`.

### ADR Files

| ADR | Path | Status |
|-----|------|--------|
| ADR-002 | `docs/adr/0002-manual-meeting-import-validation-strategy.md` | Present |
| ADR-003 | `docs/adr/0003-manual-meeting-import-data-model.md` | Present |
| ADR-004 | `docs/adr/0004-manual-meeting-import-form-content-preservation.md` | Present |

All ADRs confirmed present at time of completeness-score (t_50949d54).

---

## Production-Checklist Prerequisites

The following must be completed before the feature is enabled in any environment beyond local development:

| # | Prerequisite | Owner | Status |
|---|-------------|-------|--------|
| 1 | OpenAPI `Meeting` schema added to `contracts/openapi.yaml` | backend | Done | `contracts/openapi.yaml` lines 844–889: `createdAt` added to required array; `createdBy` and `displayOrder` added as properties; `MeetingCreate` schema covers POST body |
| 2 | Idempotency token mechanism implemented (UUID per form load, 24h TTL, 409 Conflict) | backend | Open |
| 3 | `cp_manual_meeting_import_flag_misconfiguration_total` metric implemented | backend | Open |
| 4 | `meeting.import.flag_misconfiguration` log event implemented | backend | Open |
| 5 | Dual flag namespaces (`FF_ENABLE_MANUAL_MEETING_IMPORT` and `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT`) registered and verified | backend, frontend-eng | Open |
| 6 | `make eval-arch` passes with all architecture fitness functions green | ops | Open |
| 7 | E2E eval scenarios pass (all 18 ACs verified as 🟢 Passing) | qa | Open |
| 8 | Security eval scenarios pass | qa | Open |
| 9 | Performance eval scenarios pass | qa | Open |

---

## Risks

| Risk | Likelihood | Impact | Notes |
|------|------------|--------|-------|
| OpenAPI schema gap causes integration failures | High | High | Pre-implementation prerequisite; must not be skipped |
| Duplicate meeting records from double-submit without idempotency token | Medium | Medium | Token mechanism is Must Have; AC-16 depends on it |
| Feature flag namespace drift exposes UI without server support | Medium | High | Server flag is authoritative; misconfiguration metric/log must be implemented first |
| Sensitive meeting content logged in observability pipeline | High | High | Redaction schema defined; implementation must respect it; forbidden fields list must not appear in any label or log field |
| Data retained indefinitely due to missing retention policy | Low | Medium | Deferred to BRD-06; acceptable for Phase 1 |

---

## Rollback Notes

| Scenario | Rollback approach |
|----------|------------------|
| Feature flag enabled in production causes issues | Set `FF_ENABLE_MANUAL_MEETING_IMPORT=false` and `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=false`; no database migration required for flag-only rollback |
| Duplicate meeting records discovered post-launch | Idempotency token dedup prevents new duplicates; existing duplicates require data remediation task |
| OpenAPI contract mismatch causes client errors | Revert `contracts/openapi.yaml` change; do not deploy new `Meeting` schema until backend and frontend are aligned |
| Observability redaction failure | Disable feature flag; audit logs for forbidden field presence; implement proper redaction before re-enabling |

---

## Deferred to Downstream BRDs

| Item | Deferred To | Rationale |
|------|-------------|-----------|
| Participant normalization (name inversion, capitalization) | BRD-03 | Display name stored as-entered; normalization is BRD-03 participant handling scope |
| Advanced participant deduplication | BRD-03 | Global contacts/deduplication out of BRD-02 scope |
| Data retention policy | BRD-06 | Acceptable for Phase 1; user-owned application data |

---

## Architecture Fitness Checks

Required `make eval-arch` targets for BRD-02:

| Check | Script | Expected result |
|-------|--------|-----------------|
| Feature flag parity | `evals/architecture/check-feature-flags.sh` | All `ff_*` refs in code match `specs/feature-flags.md` |
| No panic in production | `evals/architecture/check-no-panic.sh` | Zero `panic()` calls in backend |
| No background context in production | `evals/architecture/check-no-background-context.sh` | Zero `context.Background()` in backend |

These checks must exit 0 before Phase 1 implementation begins.