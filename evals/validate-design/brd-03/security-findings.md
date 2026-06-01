# Security Findings — BRD-03 Meeting Memory Processing

**Validated:** 2026-05-21
**Validator:** security
**Target:** specs/domain/brd-03-meeting-memory-processing.md
**Note:** curated version does not exist; validating against specs/domain/brd-03-meeting-memory-processing.md

---

## Verdict: NEEDS_ATTENTION

---

## Critical Issues (Must Fix)

None identified at Critical severity.

---

## High Priority Issues (Should Fix)

### [H1] Authentication mechanism not specified for memory processing endpoints

**Source:** BRD-03 FRs (US-1 through US-8) — no authentication/authorization detail
**BRD section:** Functional Requirements, all 8 user stories

BRD-03 says "Authenticated user" for every user story but does not specify how authentication is enforced for the memory processing, retry/reprocess, conflict review, and conflict resolution endpoints. The BRD defers auth to BRD-02, but BRD-02 covers manual meeting import — not the memory processing API surface.

**Evidence:** BRD-03 Overview states "authenticated user" without specifying session/JWT/API-key mechanism. No auth-related AC exists for memory endpoints (AC-1 through AC-24 cover feature behavior, not auth).

**Impact:** Unauthenticated users could trigger memory processing, access conflict queues, or resolve conflicts.

**Fix required:** Either add auth requirements to BRD-03 or explicitly reference a future auth BRD that will cover these endpoints. At minimum, specify which endpoints require authentication and which role/permission gates retry and conflict resolution actions.

---

### [H2] Audit trail for prior-memory input changes (FR-25) lacks protected storage specification

**Source:** BRD-03 Should Have, FR-25
**BRD section:** "Preserve a safe audit trail of user include/exclude changes to prior-memory inputs without logging sensitive content"

The audit trail requirement is stated but not specified: where is it stored, who can read it, what rotation/retention applies, and how is it protected from tampering? The phrase "safe audit trail" implies integrity protection (append-only, tamper-evident) but no mechanism is described.

**Impact:** If the audit trail is stored in a regular table without immutability controls, a malicious actor with DB access could modify or delete prior-memory exclusion records, masking manipulation of processing inputs.

**Fix required:** Specify audit trail storage (e.g., append-only table with no UPDATE/DELETE, or a separate audit log), who can read it (operators only, not users), and that it contains counts/correlations only — no meeting titles or memory content.

---

## Medium Priority Issues

### [M1] Flag misconfiguration detection (server=false, browser=true) emits "safe logs/metrics" — what exactly?

**Source:** BRD-03 Feature Flag table, row: `false | true`
**BRD section:** Feature Flag

When `FF_ENABLE_MEETING_MEMORY_PROCESSING=false` (server) but `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=true` (browser), the BRD says "this is a misconfiguration and should be surfaced through safe logs/metrics." The metric name and label are not specified.

**Impact:** Implementers may emit a metric with meeting_id or correlation_id that reveals a configuration state, or may not emit anything at all.

**Fix required:** Add the specific metric name (e.g., `cp_meeting_memory_flag_misconfiguration_total`) to the Observability section and the Feature Flag section.

---

### [M2] No rate limiting specified for memory processing endpoints

**Source:** BRD-03 NFR table — no rate limiting entry
**BRD section:** Non-Functional Requirements

BRD-02 had no rate limiting specification (flagged in BRD-02 security findings). BRD-03 doubles the API surface (retry, reprocess, conflict resolution, prior-memory include/exclude, review queue). No rate limiting is specified.

**Fix required:** Add rate limiting to NFR table: max N concurrent processing jobs per user, max N retries per hour, max N conflict resolutions per hour.

---

### [M3] ADR-0008 Status is "Proposed" — privacy enforcement not agreed

**Source:** ADR-0008 header
**ADR:** ADR-0008 Observability & Privacy Architecture

ADR-0008 defines the privacy architecture (allowed/prohibited fields, Redactor utility, ProcessingResult struct as the enforcement boundary). It is marked "Proposed" not "Accepted." Until accepted, the privacy enforcement design is not committed.

**Fix required:** Accept ADR-0008 before Phase 2 implementation begins. The ProcessingResult struct as the sole emission boundary is the correct pattern, but it must be agreed upon.

---

## Low Priority Issues

### [L1] `correlation_id` in logs could theoretically be correlated across sessions

**Source:** ADR-0008 Observability & Privacy Architecture, Allowed Fields
**ADR:** ADR-0008

`correlation_id` is described as "opaque, no business meaning." A correlation_id is stable across a processing job's lifetime but could theoretically be used to track that the same job was retried multiple times, revealing retry patterns over time.

**Severity:** Low — correlation_id is UUID v4 and carries no content. Mitigated by the fact that operators need it for job tracing.

**Recommendation:** Document this as an accepted trade-off in the ADR, or add a note that correlation_ids are rotated on each retry attempt if traceability is a concern.

---

## Positive Findings

- Privacy requirements are thorough and explicit: raw transcript, notes, evidence snippets, participant PII, stakeholder note content, generated memory content, and resolution text are all banned from logs/metrics (FR-25)
- Dual-namespace feature flag correctly specced with server-authoritative enforcement
- Observability events well-defined with safe field names (correlation_id, safe failure_class, etc.)
- `Redactor` utility design (SHA-256, first 8 hex chars) is a sound privacy pattern
- ProcessingResult struct as the sole emission boundary is architecturally correct
- Conflict review queue is isolated from briefing eligibility until resolved (FR-13, FR-14)
- Retry policy (3 retries, exponential backoff) is explicitly specified

---

## Summary Table

| ID | Severity | Issue |
|----|----------|-------|
| H1 | High | Auth mechanism not specified for memory processing endpoints |
| H2 | High | Audit trail for prior-memory input changes lacks protected storage spec |
| M1 | Medium | Flag misconfiguration metric name not specified |
| M2 | Medium | No rate limiting specified for memory processing endpoints |
| M3 | Medium | ADR-0008 privacy architecture is "Proposed" not "Accepted" |
| L1 | Low | correlation_id stability could theoretically reveal retry patterns |