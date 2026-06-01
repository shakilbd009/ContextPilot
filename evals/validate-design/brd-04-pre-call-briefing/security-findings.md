# Security Findings: brd-04-pre-call-briefing

## Overview
Security validation of BRD-04 Pre-Call Briefing. Auth/authorization, input validation, flag mismatch, PII exposure, and timing risks reviewed against brd.md.

---

## 1. Authorization Boundary (FR-19)

**Spec requirement:** "Briefings and source details are visible only to users authorized to view the upcoming meeting AND all included source meetings. Unauthorized source meetings must not be selected, displayed, logged, or used in generation." (brd.md line 127–128)

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD-04 spec | PASS | FR-19 clearly states owner-only ACL; ADR-0009 adopted owner-only semantics |
| BRD-02 data model | PASS | `createdBy` is the sole identity; `meeting_participants` is for content matching only |
| BRD-03 memory access | PASS | Authorization enforced at meeting ACL level per BRD-03 |
| Implementation risk | MEDIUM | No `check-authorization.sh` eval exists in security eval. Source meeting ACL enforcement for briefing versions (including superseded versions) not explicitly covered in security eval scenarios |

**Required action:** Security eval (evals/security/brd-04-pre-call-briefing.md lines 35–42) covers GET briefing revealing unauthorized sources, but POST regenerate and POST restore on superseded versions with unauthorized source access is not tested.

---

## 2. Feature Flag Mismatch Attack Surface (FR Feature Flag table)

**Spec requirement:** `false`/`true` mismatch emits metric `cp_pre_call_briefing_flag_misconfiguration_total` and log event `briefing.flag_misconfiguration` (brd.md lines 290–291).

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Metric name and log event both named explicitly |
| Security eval | PASS | Flag misconfiguration scenario covered at brd.md evals/security lines 50–51 |
| Implementation gap | LOW | Metric name is specified; no histogram buckets or label cardinality limits stated for this metric |

---

## 3. Input Validation / Injection Resistance

**Spec requirement:** All 6 endpoints validate path parameters and reject malformed input (brd.md lines 299–322 AC table).

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Security eval | PASS | UUID validation, integer version validation, body parsing, and self-referential UUID rejection all specified (evals/security lines 57–98) |
| BRD coverage | PASS | AC-22 covers authorization checks for unauthorized source meetings |

---

## 4. PII Exposure in Briefing Content (FR-18, NFR Privacy)

**Spec requirement:** "Operator diagnostics must not expose raw meeting content or participant PII" (brd.md line 123). "Logs and metric labels contain no raw transcript text, notes text, briefing text, source snippets, participant PII" (brd.md line 267).

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | Privacy NFR and FR-18 both ban PII from operator-facing outputs |
| Security eval | PASS | No briefing text, source snippets, or participant PII in metrics/labels scenario covered (evals/integration line 29) |
| Observability gap | MEDIUM | Metric names and log event names for pipeline health are not enumerated in brd.md. Only `cp_pre_call_briefing_flag_misconfiguration_total` and `briefing.flag_misconfiguration` are named. No `cp_briefing_generation_duration_ms` histogram, no product-usefulness event names are listed in the BRD spec itself — they appear only in the performance and e2e eval files. |

---

## 5. Preparation Status State Machine Authorization

**Spec requirement:** Only valid preparation status transitions per BRD-04 spec (generating, ready, ready_with_caveats, no_prior_memory, stale, failed, regenerating). AC-18 requires failed generation not remove latest completed briefing.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| BRD spec | PASS | FR-20 defines all 7 states; AC-18 covers failure continuity |
| Security eval | PASS | No scenario tests what happens if a user manually sets status via direct DB write or API manipulation — this is acceptable as DB-level integrity is assumed |

---

## Summary

| Finding | Severity | File:Line | Description |
|---------|----------|-----------|-------------|
| Source meeting ACL not tested on superseded version reads | High | evals/security/brd-04-pre-call-briefing.md | GET /briefing/versions/{n} with source from unauthorized meeting — only one scenario covered; regenerate/restore on superseded versions not tested |
| Metric and log event names not enumerated in brd.md | Medium | brd.md (no line ref) | Only `cp_pre_call_briefing_flag_misconfiguration_total` and `briefing.flag_misconfiguration` are named in the spec; other metric names appear only in eval files |

**Verdict: NEEDS_ATTENTION** — 1 high finding, 1 medium finding.
