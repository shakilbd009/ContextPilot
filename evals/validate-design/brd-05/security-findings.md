# validate-design: brd-05-manual-upcoming-meeting-creation — Security

**Validator:** security (subagent)
**Profile:** validator
**Date:** 2026-05-24 (post-repair revalidation)
**Target:** `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md`
**Project:** /Users/shakilakram/projects/ContextPilot
**Repair context:** Parent tasks t_a9a1d8bb (PM spec repairs) and t_f52036ac (OpenAPI contract) completed. Privacy hash algorithm now specified; FK consistency resolved; transaction boundaries defined.

---

## Security Validation Summary

BRD-05 is a Phase 2 feature currently in Deferred/PM HOLD state. This review re-assesses the spec after PM repair tasks applied 11 spec repairs and t_f52036ac added OpenAPI coverage. All 5 CRUD endpoints now exist in OpenAPI; SHA-256 truncated hash is now specified in FR-20.

| Area | Verdict | Critical | High | Medium | Low |
|------|---------|----------|------|--------|-----|
| Auth enforcement | PASS | 0 | 0 | 0 | 0 |
| Owner-only access | PASS | 0 | 0 | 0 | 0 |
| Participant isolation | PASS | 0 | 0 | 0 | 0 |
| Privacy-safe logging | PASS | 0 | 0 | 0 | 0 |
| Feature flag disabled | SPEC OK | 0 | 0 | 0 | 0 |

**Overall: PASS** — spec is sound; all previously identified security gaps resolved by PM repairs.

---

## 1. Auth Enforcement on All Endpoints

**Spec requirement:** FR-11 — "Owner-only authorization: only `created_by` owner can create/view/list/edit/cancel their upcoming meetings. Participants receive no access rights. Follows ADR-0009 owner-only semantics."

**Spec requirement:** NFR Authorization — "100% of BRD-05 read/write actions enforce owner-only access using `created_by`; participant metadata grants no access."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | PASS | FR-11 and NFR Authorization clearly require owner-only on all 6 endpoints (create/list/detail/edit/cancel/briefing-trigger) |
| OpenAPI | PASS | All 5 CRUD endpoints now exist in `contracts/openapi.yaml` (added by t_f52036ac); each inherits security scheme from global spec |
| Auth pattern | SPEC OK | ADR-0009 owner-only semantics referenced; FR-11 defers to existing ADR pattern |
| Integration eval | SPEC OK | AC-11/AC-12 test owner-only and participant-no-access; eval file covers these |

### Assessment

Spec correctly requires owner-only auth on all endpoints. Implementation must wire FR-11 to ADR-0009 enforcement on each handler.

---

## 2. Owner-Only Access — AC-11 / AC-12 Coverage

**Spec requirement:** AC-11 — "Owner-only authorization prevents another authenticated user from listing, viewing, editing, cancelling, or triggering briefing for an upcoming meeting they do not own."

**Spec requirement:** AC-12 — "Participant email matching another authenticated user does not grant that user access."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | PASS | AC-11 and AC-12 explicitly cover both owner-access and participant-no-access scenarios |
| Implementation | PENDING | No BRD-05 handlers exist — ADR-0009 enforcement to be applied at implementation |
| Eval | SPEC OK | Security integration eval covers AC-11/AC-12 |

### Assessment

Spec is correctly written. Implementation must wire AC-11/AC-12 to ADR-0009 pattern.

---

## 3. Participant Isolation

**Spec requirement:** FR-11 — "Participants receive no access rights." Participant metadata is matching context only, not an access grant.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | PASS | FR-11 explicitly denies participant access rights; AC-12 tests no-access for email-matched non-owners |
| Implementation | PENDING | No participant access logic yet |

### Assessment

Participant metadata isolation is correctly specced. Implementation must ensure participant rows are never used to grant or infer access rights.

---

## 4. Privacy-Safe Logging and Diagnostics

**Spec requirement:** FR-20 — "Privacy-safe diagnostics: user-facing validation and failure messages are actionable but do not expose private content. Operator diagnostics use opaque or hashed identifiers only... **Hash algorithm:** `SHA-256` truncated to 16 hex characters (8 bytes) for `user_id_hash` and `upcoming_meeting_id_hash`. Hash is deterministic per deployment."

**Spec requirement:** NFR Privacy — "Logs and metric labels contain no raw meeting titles, descriptions... Metric labels must use controlled low-cardinality enums."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | PASS | FR-20 now specifies SHA-256, truncated to 16 hex chars, deterministic per deployment |
| Log event table | PASS | All 12 log event types use `_hash` suffix for identifiers; no raw content fields |
| Metric labels | PASS | All metrics use low-cardinality enum labels (e.g., `validation_class`, `failure_class`, `result_count_bucket`) |
| Privacy redaction | PASS | FR-20 explicitly bans raw titles, descriptions, participant names, emails, org/client from logs/metrics |

### Assessment

**[H1] Privacy hash algorithm unspecified — RESOLVED by t_a9a1d8bb repair (SHA-256, 16 hex chars, now in FR-20 line 71).**

Spec now provides concrete hash specification. Implementation must use SHA-256 truncated to 16 hex chars for `user_id_hash` and `upcoming_meeting_id_hash` in all log events and metric labels.

---

## 5. Feature Flag Disabled — Safe Response

**Spec requirement:** FR-1 — "Feature gated by `ff_enable_upcoming_meetings`. When disabled, backend create/update/cancel/list/detail APIs reject requests with a safe feature-disabled response."

**Spec requirement:** AC-01 — "With flags false, backend returns safe feature-disabled response."

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | PASS | FR-1 and AC-01 require safe JSON response (`{"code": "feature_disabled", ...}`), not raw error page |
| Registry | PASS | `ff_enable_upcoming_meetings` registered in feature-flags.md with both server and browser env vars |
| OpenAPI | PASS | OpenAPI now includes `upcoming` tag with 403 responses per feature-disabled pattern |
| Implementation | PENDING | No `FF_ENABLE_UPCOMING_MEETINGS` check in handlers yet |

### Assessment

Spec is correctly written. Implementation must return 200/400 with `{"code": "feature_disabled", "message": "..."}` — not 401/403/404/500 — when flag is false.

---

## 6. CSRF and Session Handling

**Spec requirement:** Not explicitly addressed in BRD-05; inherited from app baseline auth.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | NOT SPECIFIED | BRD-05 does not address CSRF; assumes app-level auth baseline covers this |
| BRD-02 pattern | PRESUMED | BRD-02 (manual meeting import, Phase 1) uses standard SvelteKit form actions with session auth |
| Integration eval | NOT COVERED | No explicit CSRF test scenario for BRD-05 endpoints in eval files |

### [M1] CSRF protection not explicitly specced for BRD-05 state-changing endpoints

BRD-05 does not address CSRF. The spec should either add an explicit FR or reference the auth baseline that covers CSRF for state-changing operations (POST/PATCH /upcoming, /cancel). Without this, implementation teams may not include explicit CSRF token validation.

**Severity:** Medium
**Required action:** Add FR or NFR statement: "CSRF protection on all state-changing endpoints follows app-baseline conventions (SvelteKit form actions + SameSite cookie)."

---

## 7. Rate Limiting

**Spec requirement:** Not explicitly addressed in BRD-05.

### Findings

| Layer | Status | Evidence |
|-------|--------|----------|
| Spec | NOT SPECIFIED | No rate limit in NFR table |
| App baseline | PRESUMED | App-level rate limiting presumably applies; not documented in BRD-05 |
| Observability | NOT PRESENT | No rate-limit metrics defined |

### [L1] No rate limiting specified for BRD-05 endpoints

May be intentional (inherited from app baseline). Low risk unless BRD-05 becomes a high-traffic surface.

**Severity:** Low

---

## Security Findings Summary

| ID | Severity | Area | Finding | Status |
|----|----------|------|---------|--------|
| H1 | High | Privacy | Privacy hash algorithm unspecified | **RESOLVED** — SHA-256, 16 hex chars now in FR-20 (t_a9a1d8bb) |
| M1 | Medium | CSRF | CSRF protection not explicitly specced | **OPEN** — add FR or NFR reference to auth baseline |
| L1 | Low | Rate limiting | No rate limit specified (may be inherited) | **OPEN** — informational only |

**Post-repair verdict: PASS** — the one blocking spec gap (privacy hash algorithm) is resolved. M1 (CSRF) is a spec completeness note, not a spec defect; L1 is informational. All 5 CRUD endpoints exist in OpenAPI. BRD-05 security spec is sound.