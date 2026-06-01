# Validation Summary: brd-04-pre-call-briefing

**Validated:** 2026-05-23
**Validators run:** security, performance, architecture, ux, devils-advocate
**Gate:** findings only — report to PM gate review (do not approve/reject)

---

## Verdict

| Validator | Verdict | Critical | High | Medium | Low |
|-----------|---------|----------|------|--------|-----|
| Security | NEEDS_ATTENTION | 0 | 1 | 1 | 0 |
| Performance | NEEDS_ATTENTION | 0 | 0 | 1 | 1 |
| Architecture | BLOCKING | 1 | 1 | 1 | 0 |
| UX | NEEDS_ATTENTION | 0 | 0 | 2 | 1 |
| Devil's Advocate | NEEDS_ATTENTION | 0 | 0 | 7 | 2 |

**Overall:** BLOCKING (architecture critical finding)

---

## Critical Issues (Must Fix Before Implementation)

### ARCH-1: briefing_source_exclusions missing restored_at column
**Source:** brd.md lines 202–212 vs ADR-0012
**Description:** ADR-0012 (Source Exclusion Undo — Soft Delete with restored_at Column) specifies `restored_at TIMESTAMPTZ NULL` column on `briefing_source_exclusions`. The BRD-04 data model does NOT include this column. FR-17 requires excluded sources remain visible in a collapsed excluded-sources area after undo — only possible with soft delete. ADR-0012 decision is NOT incorporated in the BRD data model.
**Required fix:** Add `restored_at TIMESTAMPTZ NULL` column to `briefing_source_exclusions` table schema in brd.md.

---

## High Priority Issues (Should Fix)

### ARCH-2: upcoming_meetings table now defined in BRD-05
**Source:** BRD-05 FR-5
**Description:** `upcoming_meetings` table is referenced by BRD-04 FK constraints. BRD-05 now defines the table, resolving this dependency.
**Status:** RESOLVED — BRD-05 FR-5 defines `upcoming_meetings` table.

### SEC-1: Source meeting ACL not tested on superseded version reads
**Source:** evals/security/brd-04-pre-call-briefing.md lines 35–42
**Description:** GET /briefing/versions/{n} with source from unauthorized meeting is tested, but POST regenerate and POST restore on superseded versions with unauthorized source access is not covered in security eval scenarios.
**Required fix:** Add security test scenarios for regenerate/restore operations accessing superseded briefing versions with unauthorized source meetings.

---

## Medium Priority Issues

| ID | Validator | Description |
|----|-----------|-------------|
| ARCH-3 | Architecture | OpenAPI contract not updated with 6 briefing endpoints (contracts/openapi.yaml) |
| SEC-2 | Security | Metric and log event names not enumerated in brd.md — only in eval files |
| PERF-1 | Performance | No concurrent user/queue depth limits in NFR table; "normal operating conditions" undefined |
| UX-1 | UX | ARIA live regions for dynamic states (generating, stale, failed) not specified |
| UX-2 | UX | BRD-01 component inventory cross-reference not verified for WCAG compliance |
| DA-1 | Devil's Advocate | OQ-4 threshold undefined: progressive format sufficiency has no trigger criteria |
| DA-2 | Devil's Advocate | No retention/archival policy for briefing versions (deferred to BRD-06 but storage implications unstated) |
| DA-3 | Devil's Advocate | Staleness first-computed-on trigger not defined (ADR-0011 read-time algorithm) |
| DA-4 | Devil's Advocate | Regeneration + source memory change race condition not specified |
| DA-5 | Devil's Advocate | preparation_status to briefing_versions.status mapping incomplete for failed generation lifecycle |

---

## Low Priority Issues

| ID | Validator | Description |
|----|-----------|-------------|
| PERF-2 | Performance | Histogram bucket boundaries not in BRD spec (only in eval) |
| UX-3 | UX | "Safe" failure reason definition vague (FR-18) |
| DA-6 | Devil's Advocate | OQ-5 aggregation rule not exemplified with mixed-quality scenario |
| DA-7 | Devil's Advocate | No optimistic locking on exclusion/restore concurrent operations |
| DA-8 | Devil's Advocate | Cross-participant briefing access not explicitly called out as out of scope |

---

## Positive Findings

- 8 US and 22 AC all have eval methods assigned; no ambiguous criteria
- API contract: 6 endpoints clearly defined with method/path/summary
- Observability: FR-22 low-cardinality enum commitment; privacy NFR well-specified
- Feature flags: dual namespace (FF_/VITE_FF_) correctly defined with server-authoritative enforcement
- BRD-03 evidence status conventions correctly adopted throughout
- ADR-0009 (owner-only ACL) and ADR-0011 (staleness derived at read time) properly incorporated
- All 5 required eval files present and non-empty
- check-status-sync.sh passes

---

## Next Steps

1. **ARCH-1 (Critical):** Patch brd.md `briefing_source_exclusions` data model to include `restored_at TIMESTAMPTZ NULL` per ADR-0012
2. **ARCH-2 (High):** Create BRD-05 to define `upcoming_meetings` minimal schema
3. **SEC-1 (High):** Add security eval scenarios for regenerate/restore on superseded versions with unauthorized sources
4. **ARCH-3 (Medium):** Update `contracts/openapi.yaml` with 6 briefing endpoint schemas before implementation
5. **DA-1 through DA-5 (Medium):** Address specification gaps in OQ resolution, retention policy, staleness trigger, and race conditions — consider whether these require BRD amendments or are implementation-level concerns
6. **UX-1, UX-2 (Medium):** Add ARIA live region specifications and verify BRD-01 component inventory cross-reference

---

## Files Produced

- `evals/validate-design/brd-04-pre-call-briefing/security-findings.md`
- `evals/validate-design/brd-04-pre-call-briefing/performance-findings.md`
- `evals/validate-design/brd-04-pre-call-briefing/architecture-findings.md`
- `evals/validate-design/brd-04-pre-call-briefing/ux-findings.md`
- `evals/validate-design/brd-04-pre-call-briefing/devils-advocate-findings.md`