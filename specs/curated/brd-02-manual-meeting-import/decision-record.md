# BRD-02 Manual Meeting Import — Decision Record

---
brd_id: brd-02
title: Manual Meeting Import
gate: Graduation Evidence Package
date: 2026-05-22
approver: pm
source_tasks:
  - t_50949d54  (completeness-score)
  - t_c1d0e43c  (validate-design)
  - t_6a94a3fe  (PM gate after completeness-score)
  - t_489d033b  (PM gate after validate-design)
---

## Gate Verdicts

| Gate | Task | Verdict | Date | Notes |
|------|------|---------|------|-------|
| Completeness-score | t_50949d54 | PASS (20/20) | 2026-05-20 | 7/7 OQs resolved; 1 TBD deferred to BRD-06 |
| PM gate (completeness-score) | t_6a94a3fe | approve | 2026-05-22 | All open items resolved or deferred with rationale |
| validate-design | t_c1d0e43c | PASS (9/9 checks) | 2026-05-20 | 2 refiner HIGH, 2 architect critical, 4 architect HIGH resolved |
| PM gate (validate-design) | t_489d033b | approve | 2026-05-22 | All findings resolved or explicitly deferred |

---

## Open Question Resolutions

| OQ | Question | Disposition | Rationale | Source |
|----|----------|-------------|-----------|--------|
| OQ-1 | Idempotency token semantics not specified | Resolved — Must Have | Elevated from unspecified to Must Have; UUID per form load, server-side dedupe with 24h TTL, 409 Conflict with original meeting ID | t_c1d0e43c run 64; t_33b749d9 |
| OQ-2 | Meeting ID URL encoding | Resolved | UUID format confirmed: `/meetings/{meetingId: uuid}` in route paths | t_33b749d9 |
| OQ-3 | Client-side validation implementation detail | Deferred to code review | ADR-002 specifies dual-layer validation; shared constants package is a Phase 1 implementation detail | t_489d033b metadata |
| OQ-4 | Character count display threshold | Resolved | Warning at 45,000; hard block at 50,001; promoted from Could Have to Should Have | t_33b749d9 |
| OQ-5 | Transaction isolation level | Resolved | READ COMMITTED (PostgreSQL default); changing requires ADR approval | t_33b749d9 |
| OQ-6 | Participant ordering | Resolved | `displayOrder INTEGER NOT NULL DEFAULT 0` added to schema | t_33b749d9 |
| OQ-7 | title max length | Resolved | 500 characters; enforced via CHECK constraint | t_33b749d9 |

---

## Findings and Dispositions

### From completeness-score (t_50949d54)

No blocking findings. The single open TBD (data retention, line 167 of curated BRD) is explicitly deferred to BRD-06 and documented as acceptable for Phase 1.

| Finding | Severity | Disposition | Rationale |
|---------|----------|-------------|-----------|
| Data retention policy not defined | TBD | Deferred to BRD-06 | User-owned application data retained until retention policy exists; acceptable for Phase 1 per PM gate t_6a94a3fe |

### From validate-design (t_c1d0e43c)

All 9 validation criteria checked. All HIGH and critical findings resolved.

**Refiner findings:**

| Finding | Severity | Disposition | Rationale |
|---------|----------|-------------|-----------|
| Open Questions "None" was factually inaccurate | HIGH | Repaired | Replaced with explicit OQ resolution table in curated BRD |
| NFR contradiction between "noticeable delay" and 1-second save latency | HIGH | Repaired | Separated into form interactivity (<100ms) and save latency (1s); each covers distinct concerns |

**Architect critical findings:**

| Finding | Severity | Disposition | Rationale |
|---------|----------|-------------|-----------|
| Idempotency token not in BRD | Critical | Repaired | Added as Must Have with 409 Conflict semantics, 24h TTL, originalOutcome in response body |
| OpenAPI schema mismatch | Critical | Flagged as Phase 1 blocker | New `Meeting` schema required in `contracts/openapi.yaml`; documented as prerequisite in BRD Pre-Phase 1 section and decision-record |

**Architect HIGH findings:**

| Finding | Severity | Disposition | Rationale |
|---------|----------|-------------|-----------|
| OQ-2 Meeting ID URL encoding | HIGH | Repaired | UUID format confirmed in route paths |
| OQ-3 client-side validation implementation | HIGH | Deferred to code review | ADR-002 specifies dual-layer; shared validation constants package is Phase 1 implementation detail |
| OQ-7 title max length | HIGH | Repaired | 500 chars with CHECK constraint added to schema |
| Flag misconfiguration observability gap | HIGH | Repaired | `cp_manual_meeting_import_flag_misconfiguration_total` metric and `meeting.import.flag_misconfiguration` log event added |

---

## Deferred Items

| Item | Deferred To | Rationale |
|------|-------------|-----------|
| Participant normalization (name inversion, capitalization) | BRD-03 | Display name stored as-entered with whitespace trimmed; normalization logic belongs to BRD-03 participant handling |
| Advanced participant deduplication | BRD-03 | Global contacts and deduplication are out of scope for BRD-02 |
| Shared validation constants package between SvelteKit and Go | Code review | Phase 1 implementation detail; ADR-002 provides dual-layer validation strategy, specific constants package structure to be resolved during code review |
| Data retention policy | BRD-06 | Acceptable for Phase 1; user-owned application data retained until policy defined |

---

## Phase 1 Prerequisites (Implementation Blockers)

These items are tracked as implementation prerequisites, not BRD graduation blockers. BRD-02 is approved for graduation; implementation must not begin until these are resolved.

| Item | Owner | Status | Source |
|------|-------|--------|-------|
| OpenAPI contract update — new `Meeting` schema in `contracts/openapi.yaml` | backend | Open | t_c1d0e43c Phase 1 blocker |
| Idempotency token mechanism implementation | backend | Open | t_c1d0e43c Phase 1 blocker |
| Flag misconfiguration observability implementation | backend | Open | t_c1d0e43c Phase 1 blocker |

---

## Trade-offs

| Decision | Trade-off | Rationale |
|----------|-----------|-----------|
| Idempotency token as Must Have | Adds implementation complexity; requires token lifecycle management | Prevents duplicate submissions from double-clicks and retry storms; necessary for data integrity |
| Data retention deferred to BRD-06 | Records retained indefinitely until policy exists | Acceptable short-term for Phase 1 user-owned application data; explicit deferral documents the gap |
| Client-side validation constants deferred to code review | Shared constants duplication risk between SvelteKit and Go | ADR-002 provides dual-layer strategy; specific package structure can be resolved by implementors with code review oversight |
| NFR split into form interactivity (<100ms) and save latency (1s) | Two separate NFR targets instead of one | Form interactivity is client-side only; save latency is server round-trip; different measurement domains require separate targets |

---

## Approval

PM gate after completeness-score (t_6a94a3fe): **verdict: approve** — 2026-05-22
PM gate after validate-design (t_489d033b): **verdict: approve** — 2026-05-22

BRD-02 Manual Meeting Import is approved for graduation. Implementation tasks may be created subject to Phase 1 prerequisite resolution.