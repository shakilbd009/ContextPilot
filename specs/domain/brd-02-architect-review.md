# Architect Review — BRD-02 Manual Meeting Import

**Reviewer:** architect
**Source:** `specs/domain/brd-02-manual-meeting-import.md`
**ADR sources:** `docs/adr/0002-*.md`, `docs/adr/0003-*.md`, `docs/adr/0004-*.md`
**Refiner review:** `specs/domain/brd-02-refiner-review.md`
**Date:** 2026-05-20
**Task:** t_a2939a7d

---

## 1. Data Flows and API Contract Analysis

### 1.1 Primary Save Flow

```
User fills form → SvelteKit form action (?/create)
  → Go handler [POST /meetings]
    → Feature flag evaluation (FF_ENABLE_MANUAL_MEETING_IMPORT)
    → Session validation (auth check)
    → Idempotency token check (dedupe)
    → Server-side validation (authoritative gate)
      → Validation failure → 400 + field errors + echo values → use:enhance → form repopulated
      → Validation pass → PostgreSQL transaction
        → INSERT meetings (title, completed_at, transcript, notes, content_source, created_by)
        → INSERT meeting_participants (per participant row)
        → COMMIT
      → Success → 201 + redirect URL → SvelteKit redirect to /meetings/[id]
```

**ADR grounding:**
- ADR-002: server-side validation is the authoritative gate; client-side is non-blocking UX feedback
- ADR-003: two-table schema (meetings + meeting_participants) with foreign key cascade delete
- ADR-004: `use:enhance` for progressive enhancement; server echo for form preservation; sessionStorage draft flag as fallback only

### 1.2 Inferred API Contract

The BRD does not contain an explicit API contract. The refiner review (MEDIUM finding) drafted one; it is confirmed necessary and is restated here for cross-reference:

```
POST /meetings
Feature flag: FF_ENABLE_MANUAL_MEETING_IMPORT (server-side, authoritative)

Request body (application/json):
{
  "title": "string (required)",
  "completedAt": "ISO 8601 timestamp (required)",
  "participants": [
    {
      "displayName": "string (required)",
      "email": "string (optional)",
      "organization": "string (optional)",
      "role": "string (optional)"
    }
  ],
  "transcript": "string (optional; notes must be present if absent)",
  "notes": "string (optional; transcript must be present if absent)",
  "idempotencyToken": "string (UUID, hidden form field)"  — not in BRD, required by ADR-004
}

Responses:
201 Created      { "id": "<meetingId>", "redirect": "/meetings/<id>" }
400 Bad Request  { "errors": [{ "field": "<name>", "message": "<specific>" }], "values": { ... } }
401 Unauthorized Session invalid or absent
403 Forbidden    FF_ENABLE_MANUAL_MEETING_IMPORT=false
409 Conflict     Idempotency token already used (duplicate submission)
500 Internal     { "error": "<safe class>", "correlationId": "<uuid>" }
```

**OpenAPI gap:** The `contracts/openapi.yaml` currently defines only a `MeetingSummary` schema (with `start_time`, `end_time`, `attendees[]`) that does not match the BRD-02 data model (`completedAt`, `transcript`, `notes`, `content_source`, separate `meeting_participants` table). The BRD-02 `POST /meetings` contract must be added to `contracts/openapi.yaml` before Phase 1 implementation begins. The existing `MeetingSummary` schema should be renamed or scoped to the BRD-01 meeting list response, and a new `Meeting` schema (or `ManualMeetingImport` variant) should be added for the BRD-02 write path.

### 1.3 Read Flows

```
GET /meetings         → list all meetings for authenticated user (with participant JOIN)
GET /meetings/[id]    → meeting detail with participants loaded
```

These are deferred to BRD-01's route structure. The BRD-02 data model (ADR-003) defines the schema that these routes will query.

### 1.4 Feature Flag Gating Flow

```
Server path:
  FF_ENABLE_MANUAL_MEETING_IMPORT=false → 403 on POST /meetings; /meetings/new hidden at route level

Browser path:
  VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=false → nav link hidden; direct URL access blocked by server guard

Misconfiguration state (FF=true, VITE_FF=false): allowed for server-side testing
Misconfiguration state (FF=false, VITE_FF=true): logged as misconfiguration; server denies
```

The refiner review correctly identifies that the server flag is authoritative. The BRD does not specify the logging mechanism for misconfiguration detection — see Observability Gaps (Section 4).

### 1.5 Form Content Preservation Flow

```
User submits → server validates → failure → 400 with { errors, values }
  → use:enhance intercepts → form re-renders with echoed values pre-populated
  → Catastrophic failure (server crash mid-submit) → sessionStorage draft flag checked on page load → prompt user
```

ADR-004 specifies sessionStorage stores only `{ hasDraft: true, savedAt: timestamp }` to stay under the ~5MB limit. This is a design decision, not a BRD requirement — the gap is the BRD does not reference this behavior.

---

## 2. Failure Mode Identification

### 2.1 Network and Transport Failures

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Save request times out (>10s) | Client-side timeout; no 201 received | Idempotency token prevents duplicate on retry; user sees error and can resubmit |
| Partial response (server died after DB commit, before sending 201) | Same as above | Idempotency token + possibility of orphaned meeting (see 2.4) |
| Client loses connectivity mid-submit | XHR/fetch error | Form content preserved via use:enhance DOM state; user can resubmit |

### 2.2 Validation Failures

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Any required field missing | 400 response with field-level errors | use:enhance re-renders form with echoed values; user corrects and resubmits |
| Transcript + notes > 50,000 chars | 400 with content size error | Echoed values preserved; user truncates |
| All participants removed by concurrent edit | 400 with participant error | Same as above |

### 2.3 Authentication and Authorization

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Session expired mid-edit | 401 on submit | SvelteKit auth guard redirects to login; form content lost (acceptable — no sensitive data in transit) |
| Feature flag disabled between page load and submit | 403 on submit | User sees disabled state; no data loss |
| FF/server=false, VITE_FF/server=true (misconfiguration) | Server log/metric emitted; 403 returned | Operator notices misconfiguration metric |

### 2.4 Idempotency and Duplicate Submission

| Failure | Detection | Recovery |
|---------|-----------|----------|
| User double-clicks save | Server checks idempotency token; duplicate → 409 | No duplicate created; client shows "already saved" |
| User resubmits after network timeout (unaware of success) | Token already consumed → 409 | Client retrieves created meeting from 409 response body |

**Architectural gap (not addressed in BRD or ADRs):** The idempotency token is mentioned in ADR-004 but the BRD does not require it. AC-16 (double-click prevention) is listed as an E2E test requirement but the server-side enforcement mechanism is not specified. This must be resolved before Phase 1 — either the idempotency token goes into the BRD as a Must Have or a different dedupe strategy is specified.

### 2.5 Database Failures

| Failure | Detection | Recovery |
|---------|-----------|----------|
| PostgreSQL unreachable | Go handler returns 500; /ready probe fails | User sees generic error; retry after /ready recovers |
| Transaction failure mid-INSERT | PostgreSQL error → transaction rollback | No partial record created; user resubmits |
| Participant INSERT fails (FK violation) | Transaction rollback | No partial meeting; user resubmits |
| Meeting created but participant INSERT fails twice | ON DELETE CASCADE removes meeting | No orphaned participant record |

### 2.6 Form State and Storage

| Failure | Detection | Recovery |
|---------|-----------|----------|
| sessionStorage unavailable (private browsing, quota exceeded) | try/catch on setItem | Silently skipped; no draft recovery available |
| sessionStorage draft from tab A recovered in tab B | Draft token checked against in-flight server state | Clear stale draft on successful save |
| Browser back-button during in-flight save | use:enhance cancels fetch on navigation | Request may or may not reach server; idempotency token prevents double-create |

### 2.7 Content Size Edge Cases

| Failure | Detection | Recovery |
|---------|-----------|----------|
| 50,001-char submission | Server 400 | Echoed; user must truncate |
| Empty transcript AND empty notes | Server 400 | Echoed; user adds content |
| content_source derivation conflict (both null) | Server-side CHECK rejects | Never reaches DB; 400 returned |

### 2.8 Feature Flag Race Conditions

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Flag disabled between page load and submit | 403 on submit | User sees disabled UX; no data loss |
| Flag enabled between page load and submit | 201 on submit | Normal success path |

---

## 3. Non-Functional Requirements Check

### 3.1 Form Interactivity — CLIENT-SIDE (< 100ms target)

**BRD claim (line 92):** "without noticeable delay for content up to 50,000 combined transcript-plus-notes characters"

**Refiner finding (HIGH):** "noticeable delay" is subjective; the BRD contradicts itself by referencing the same content in Save latency as "1 second."

**Architect verdict:** The refiner's proposed fix (<100ms client-side feedback) is reasonable. However, the BRD should specify what "client-side interactivity" means in concrete terms:
- Keystroke response in a 50K-character textarea: native browser behavior is sufficient (browser handles own rendering)
- Participant row add/remove: must be <100ms DOM update
- Advanced affordance expand/collapse: must be <100ms CSS transition
- Validation feedback on submit: client-side check should return error messages <100ms
- Save button state (disabled during in-flight): immediate (no network)

**Concern:** A pure-Svelte textarea with 50K characters of initial value (after server echo) could cause layout thrashing on repaint. This should be verified with a prototype before Phase 1 concludes. A textarea with 50K chars is ~100KB of DOM — manageable but not trivial on low-end devices.

### 3.2 Save Latency — SERVER ROUND-TRIP (1 second target)

**BRD claim (line 93):** "Successful save returns within 1 second for transcript plus notes content up to the 50,000 character product limit"

**Architect verdict:** 1 second is achievable for a PostgreSQL transaction writing two tables with 50K text columns, assuming:
- PostgreSQL and Go server on same LAN (local dev / Docker Compose)
- No replica lag (single-node PostgreSQL in Phase 1)
- 50K TEXT column write is ~50KB; PostgreSQL can serialize this in <100ms
- Go JSON encoding of a 50K response body adds negligible overhead

**Scale risk (10x):** At 10x concurrent saves (10 users simultaneously saving 50K transcripts), PostgreSQL write throughput becomes a factor. Phase 1 is single-node; no horizontal write scaling planned yet. The NFR target is achievable in Phase 1 but should be re-evaluated before production with real load.

**Gap:** The BRD does not specify whether "within 1 second" includes the network round-trip (client → server → client) or only server-side processing. Assuming client-visible round-trip. For a 50K POST, the dominant cost is the network transfer (~50KB upload on a typical broadband connection is <100ms).

### 3.3 Availability

**BRD claim (line 94):** "Manual import path follows the app shell availability target from BRD-01"

**Architect finding:** BRD-01 availability target is not enumerated in the BRD-02 text. BRD-01 must define a concrete SLA (e.g., "99.5% monthly uptime") for this requirement to be verifiable. For Phase 1, a `/ready` endpoint that checks PostgreSQL connectivity is the minimum bar.

### 3.4 Data Retention

**BRD claim (line 96):** "Imported meeting content follows the project retention policy once defined; until then, records are retained as user-owned application data"

**Architect finding:** This is an explicit TBD. No retention policy exists. This is acceptable for Phase 1 but must be resolved before production. BRD-06 (Privacy & Retention Controls) is listed as P1 and will address this. Until then, no data deletion capability is required.

### 3.5 Privacy

**BRD claim (line 97):** Transcript, notes, participant PII must not be logged in raw form.

**Architect verdict:** Well-specified. The Observability section (3 log events, 4 log events) explicitly enumerates forbidden fields. ADR-003's CHECK constraint approach (application-layer 50K enforcement rather than DB-level) avoids DB error messages that could log content. The risk of inadvertent logging is in Go handler error messages (e.g., `%v` on a struct containing transcript) — a code review checklist item, not a design gap.

### 3.6 Accessibility

**BRD claim (line 95):** "WCAG 2.1 AA inherited from BRD-01"

**Architect finding:** BRD-01 does not enumerate specific WCAG criteria. The refiner review (LOW) recommends expanding AC-18 with specific expectations. This is valid — "inherited from BRD-01" without enumeration is not actionable for implementors or evaluators.

---

## 4. Observability Gaps

### 4.1 Metric Coverage Assessment

The BRD defines 7 metrics. Analysis against actual failure modes:

| Metric | Covers Failure Mode? | Gap |
|--------|---------------------|-----|
| `cp_manual_meeting_import_started_total` | Save attempt begins | OK |
| `cp_manual_meeting_import_validation_failed_total` | Validation failure | OK |
| `cp_manual_meeting_import_completed_total` | Successful save | OK |
| `cp_manual_meeting_import_failed_total` | Server-side failures | OK |
| `cp_manual_meeting_import_save_duration_ms` | Performance regression | OK |
| `cp_manual_meeting_import_content_size_chars` | Content size distribution | OK |
| `cp_manual_meeting_import_flag_eval_duration_ms` | Flag eval latency | OK |

**Missing metric:** `cp_manual_meeting_import_flag_misconfiguration_total` — for the case where `FF=false, VITE_FF=true`. The BRD (line 147) says this "should be surfaced in logs/metrics" but no metric is defined. This should be added.

### 4.2 Log Event Coverage Assessment

| Log Event | Covers Failure Mode? | Gap |
|-----------|---------------------|-----|
| `meeting.import.started` | Save attempt | OK |
| `meeting.import.validation_failed` | Validation failure | OK |
| `meeting.import.completed` | Success | OK |
| `meeting.import.failed` | Unexpected failure | OK |

**Missing log event:** `meeting.import.flag_misconfiguration` — for the `FF=false, VITE_FF=true` case. Currently no log event exists for this state.

### 4.3 Histogram Bucket Specification Gap

`cp_manual_meeting_import_content_size_chars` is described as "bucketed without storing raw content and with buckets that identify submissions near or above the 50,000 character limit." The BRD does not specify the actual bucket boundaries.

**Proposed bucket specification (architect recommendation):**
```
Buckets: [100, 1000, 5000, 10000, 25000, 40000, 50000, 60000, 100000, Inf]
Rationale:
  - 50000 is the limit boundary (distinguishes at-limit vs over-limit)
  - 60000 catches submissions just over the limit
  - 100000 catches bulk-import attempts from tooling
  - Below 100: likely notes-only, short transcripts
```

### 4.4 Idempotency Token Logging Gap

ADR-004 introduces an idempotency token (hidden form field) for duplicate submission prevention. When a 409 Conflict is returned (token already used), the log event should include whether the original request succeeded (meeting was created) or failed (it didn't). The current log event schema has no field for this.

**Proposed additional log field for 409 responses:**
```
{
  "event": "meeting.import.duplicate_submission",
  "correlationId": "<id>",
  "idempotencyToken": "<token>",
  "originalMeetingId": "<id or null>",
  "originalOutcome": "completed | failed | in_progress"
}
```

### 4.5 Label Cardinality — Confirming Refiner Finding

The refiner review (LOW) identified missing cardinality guidance. The architect confirms this is a production risk. Labels must not include high-cardinality values. The proposed fix (20 unique values per label across the fleet) is reasonable and should be added to the Observability section.

### 4.6 sessionStorage Draft Instrumentation Gap

ADR-004 introduces sessionStorage draft recovery as a fallback. This behavior is not specified in the BRD and has no observability. If draft recovery is attempted or successful, there is no metric or log event. Acceptable for Phase 1 (this is a UX enhancement), but should be tracked in Phase 2.

---

## 5. Open Architectural Questions

### OQ-1: Idempotency Token Requirement (HIGH — blocking Phase 1)

The idempotency token is referenced in ADR-004 but has no BRD requirement. AC-16 (double-click prevention) cannot be properly evaluated without knowing whether the dedupe mechanism is:
a) A server-side idempotency token (UUID per form load, stored in DB with meeting ID)
b) A client-side button-disable flag (insufficient — doesn't cover network-retry case)
c) A combination (client disables + server validates token)

**Recommendation:** Add idempotency token as a Must Have in FR. Specify it as a hidden UUID form field generated on page load, stored server-side with a TTL. This is the minimum needed for safe duplicate prevention.

### OQ-2: Meeting ID URL Encoding (MEDIUM — should resolve before Phase 1)

The BRD-01 route structure defines `/meetings/[id]` but does not specify the ID format in the URL. The ADR-003 data model uses UUIDs (`gen_random_uuid()`). In a URL: `/meetings/550e8400-e29b-41d4-a716-446655440000` is valid.

**Recommendation:** Explicitly confirm UUID format for meeting IDs in URLs. This affects OpenAPI path parameter typing (`/meetings/{meetingId: uuid}` in Echo).

### OQ-3: Client-Side Validation Implementation Detail (MEDIUM — before Phase 1)

ADR-002 says client-side validation provides "immediate feedback without a network round-trip" but does not specify the implementation. Specifically:
- Is client-side validation JS-based (vanilla or a library) or native HTML5 constraint validation?
- Are the client-side rules a mirror of server-side rules (risk of drift)?
- If using a shared validation package, how is it shared between SvelteKit and Go?

**Recommendation:** Document that a shared validation constants package should be explored if client-side and server-side rule drift becomes a maintenance burden. For Phase 1, a comment in both codebases noting the other is the source of truth is acceptable.

### OQ-4: Character Count Display Threshold (LOW — before Phase 1)

The refiner review (LOW) recommends promoting character count display from "Could Have" to "Should Have." The architect agrees — preventing user friction near the 50K limit is more cost-effective than handling validation failures.

**Additional concern not in refiner review:** The BRD does not specify when the character count warning should appear. Architect recommendation: show a warning when within 5,000 characters of the limit (i.e., at 45,000+ characters), with a hard block at 50,001.

### OQ-5: Transaction Isolation Level (MEDIUM — before Phase 1)

ADR-003 specifies a PostgreSQL transaction for the two INSERTs but does not specify the isolation level. Default PostgreSQL isolation is READ COMMITTED, which is sufficient for this use case (two sequential inserts with no concurrent modification of the same meeting). However, if BRD-03 or later features add concurrent participant updates, this may need to be REPEATABLE READ.

**Recommendation:** Add a comment in the Go handler specifying READ COMMITTED as the current isolation level and a note that changing it requires ADR approval.

### OQ-6: meeting_participants Ordering (LOW — before Phase 1)

The data model has no `display_order` or `index` column for participants. When displayed in the meeting detail UI, the order of participants is implementation-defined (likely insertion order). If two users add participants in different orders, the display may differ. For Phase 1 this is acceptable. BRD-03 may need to reason about participant roles in a canonical order.

**Recommendation:** Add `display_order INTEGER NOT NULL DEFAULT 0` to `meeting_participants` schema. This is a trivial migration and avoids a future schema change.

### OQ-7: title Max Length (MEDIUM — should resolve before Phase 1)

ADR-003 notes "Max length not specified in BRD; implementors should set a reasonable limit." PostgreSQL TEXT has no practical limit, but API request body size and UI rendering are concerns. A 500-character meeting title is reasonable; 5,000 is not.

**Recommendation:** Add `title TEXT NOT NULL CHECK (char_length(title) <= 500)` to the meetings table schema. 500 characters covers all realistic meeting title lengths while preventing abuse.

---

## 6. Summary of Findings

### Critical (must resolve before Phase 1 implementation)

1. **OQ-1: Idempotency token not in BRD** — AC-16 (duplicate prevention) cannot be implemented correctly without a server-side mechanism specified in the BRD.

### High (should resolve before Phase 1 implementation)

2. **OQ-2: Meeting ID URL encoding** — UUID format should be explicitly confirmed for OpenAPI and route contracts.
3. **OQ-3: Client-side validation implementation** — Need clarity on whether client-side rules are derived from or independent of server-side rules.
4. **OQ-7: title max length** — No upper bound specified; needs a reasonable CHECK constraint.
5. **Missing observability: flag misconfiguration metric and log event** — The BRD says this state "should be surfaced" but defines no metric or log event for it.
6. **OpenAPI schema mismatch** — `MeetingSummary` in `openapi.yaml` uses `start_time`/`end_time`/`attendees[]` which is incompatible with BRD-02's `completedAt`/`transcript`/`notes` data model. The BRD-02 contract must be added before Phase 1.

### Medium (should address in Phase 1 or before production)

7. **OQ-4: Character count display promotion** — Move from Could Have to Should Have; add warning threshold specification.
8. **OQ-5: Transaction isolation level** — Document READ COMMITTED in handler.
9. **OQ-6: participant display_order** — Add ordering column to avoid future migration.
10. **Histogram bucket specification** — 7 metrics defined but bucket boundaries for `content_size_chars` are not specified.
11. **Refiner HIGH: Open Questions accuracy** — The "None" statement is factually inaccurate; 3 PM items are absent.
12. **Refiner HIGH: NFR contradiction** — "noticeable delay" vs "1 second" need numeric separation.

### Low (nice to have before Phase 1)

13. **Refiner LOW items** (8 of them) — advanced affordance definition, AC-18 specificity, character count promotion, cardinality guidance, AC-7 eval method, redaction schema definition, participant normalization whitespace rule.

---

## 7. ADRs Required

No new ADRs are required for BRD-02 at this time. The existing three ADRs (validation strategy, data model, form content preservation) adequately cover the major architectural decisions. The gaps above are implementation details that should be resolved in code review, not new ADRs.

---

## 8. Interaction with Other BRDs

| BRD | Interaction | Concern |
|-----|-------------|---------|
| BRD-01 | App Shell provides `/meetings/new` route, auth guard, design tokens | Must confirm route is server-guarded by FF_ENABLE_MANUAL_MEETING_IMPORT, not just hidden in nav |
| BRD-03 | Meeting Memory Processing reads meetings + participants | content_source enum (transcript/notes/both) is the interface; any schema change to meetings table is a breaking change for BRD-03 |
| BRD-04 | Pre-Call Briefing may reference meeting participants | display_order (OQ-6) affects ordering stability |
| BRD-06 | Privacy & Retention Controls | 50K transcript storage + participant PII must be in scope for retention policy |
| OpenAPI | `contracts/openapi.yaml` | MeetingSummary schema mismatch must be resolved before Phase 1 endpoint implementation |

---

*Review complete. All findings documented above. Open questions OQ-1 through OQ-7 require answers before Phase 1 implementation begins.*
