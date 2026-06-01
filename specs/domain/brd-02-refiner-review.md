# Refiner Review — BRD-02 Manual Meeting Import

**Reviewer:** refiner
**Source:** `specs/domain/brd-02-manual-meeting-import.md`
**Date:** 2026-05-20
**Task:** t_10b6141a

---

## Summary

BRD-02 is structurally complete across all 9 required sections with no empty TBD placeholders. The feature flag contract is correct with dual namespace, the observability section has strong coverage (7 metrics, 4 log events, 2 health endpoints), and acceptance criteria cover the main happy-path and validation scenarios. Two HIGH issues prevent acceptance: the Open Questions section claims "None" but 3 PM-resolved items are absent from it, making the statement factually inaccurate; and the NFR section contains an internal contradiction between Form interactivity (undefined "noticeable delay") and Save latency ("within 1 second") that reference the same 50,000-character content. Four MEDIUM issues add blocking completeness gaps: missing API contract, missing data model, vague AC eval methods, and undefined redaction schema. The document is close to production-ready but needs the HIGH issues resolved before the PM can accept it.

---

## Section Verification

| Section | Present | Content | Issues |
|---------|---------|---------|--------|
| Metadata | Yes | Complete | None |
| Overview | Yes | 3 paragraphs, clear | None |
| User Stories | Yes | 8 stories | US-7 is informational rather than user-facing; acceptable |
| Functional Requirements | Yes | Must/Should/Could organized | No POST endpoint shape; no data model |
| Non-Functional Requirements | Yes | 7 entries | 1 internal contradiction; "noticeable delay" is not numeric |
| Observability | Yes | 7 metrics, 4 log events, 2 health endpoints | Redaction schema not enumerated; no cardinality guidance |
| Feature Flag | Yes | Dual namespace, 4-state table | Correct |
| Acceptance Criteria | Yes | 18 criteria | AC-7, AC-17, AC-18 use vague eval language |
| Risks & Mitigations | Yes | 9 risks | Minor: "redaction rules" referenced in AC-17 but not defined |
| Relations | Yes | Parent, blocked-by, blocks, related specs | Correct |
| Open Questions | Yes | States "None" | **HIGH: 3 PM-resolved items are absent; the "None" statement is factually inaccurate** |

---

## Issues

### HIGH — Open Questions section is factually inaccurate

**Location:** Line 205, Open Questions section

**Problem:** The section reads "None. Product decisions resolved for Phase 1." However, the PM handoff task explicitly listed 3 open questions:

1. **Exact transcript/notes size limit** — appears as 50,000 in FR line 54 but was listed as open by PM with no derivation explained
2. **Participant normalization** — handling "John Doe" vs "John  Doe" vs "Doe, John" vs "JOHN DOE" vs email-used-as-displayName is entirely unaddressed in FR
3. **Post-save navigation UX detail** — AC-14 covers redirect to `/meetings/[id]` but query params, back-button behavior, and success state are unspecified

The size limit IS present in FR so it is de facto resolved — but the BRD presents it as a settled product fact without explaining how 50,000 was derived. Participant normalization is flagged in the PM handoff but absent from the BRD. The Open Questions "None" statement is therefore inaccurate.

**Fix:** Replace the single "None" paragraph with an explicit resolution table:

```
| Question | PM Resolution | Note |
|----------|---------------|------|
| transcript/notes size limit | 50,000 characters | Derived from storage/performance sanity check; enforced in FR |
| participant normalization | Out of scope for BRD-02 | Defer to BRD-03 participant handling; displayName stored as-entered with leading/trailing whitespace trimmed |
| post-save navigation | Redirect to /meetings/[id] | AC-14; no query params or special state; back button returns to /meetings |
```

---

### HIGH — NFR internal contradiction: Form interactivity vs Save latency

**Location:** Lines 92–93

**Problem:** Two NFR entries describe latency for the same content scenario:

- Line 92 (Form interactivity): "without **noticeable delay** for content up to 50,000 combined transcript-plus-notes characters"
- Line 93 (Save latency): "within **1 second** for transcript plus notes content up to the 50,000 character product limit"

These measure different things (client-side feedback vs server round-trip) but both reference 50,000 characters and the first uses "noticeable delay" — an undefined, subjective term. A reader could interpret "noticeable delay" as contradicting or superseding the 1-second target, conflating client interactivity with server latency.

**Fix:** Give Form interactivity a specific numeric threshold and explicitly separate it from Save latency:

> **Form interactivity** — Field updates and validation feedback respond in <100ms client-side for content up to 50,000 characters. This covers field input, participant row updates, advanced participant affordance expansion, validation feedback, character-limit feedback, and save button state.
>
> **Save latency** — Successful save returns within 1 second server round-trip for content up to the 50,000 character product limit.

---

### MEDIUM — No API contract

**Location:** Functional Requirements (implied by lines 46–62)

**Problem:** The FR specifies what must be true to save a meeting but never describes the API contract: no HTTP method, no path, no request body schema, no response codes, no error response shape. An implementor reading only this BRD cannot write the API handler.

**Fix:** Add a new subsection under Functional Requirements:

```
### API Contract

`POST /meetings` — gated by `FF_ENABLE_MANUAL_MEETING_IMPORT`

**Request body (application/json):**
```json
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
  "transcript": "string (optional, requires notes if absent)",
  "notes": "string (optional, requires transcript if absent)"
}
```

**Responses:**
| Status | Condition | Body |
|--------|-----------|------|
| 201 Created | Meeting saved | `{ "id": "<meetingId>", "redirect": "/meetings/<id>" }` |
| 400 Bad Request | Validation failure | `{ "errors": [{ "field": "<name>", "message": "<specific>" }] }` |
| 401 Unauthorized | No valid session | — |
| 403 Forbidden | Feature flag disabled | — |
| 500 Internal Server Error | Unexpected failure | `{ "error": "<safe error class>", "correlationId": "<cp-uuid-timestamp>" }` |
```

---

### MEDIUM — No data model for meeting or participant records

**Location:** Functional Requirements (implied by lines 50–56)

**Problem:** FR states participants must be stored as "structured meeting participant records attached to the meeting" and that the meeting must be a "first-class meeting record" listable through the meeting list route. No schema is provided for either record type. Implementors must invent field names, types, relationships, and ID generation strategy.

**Fix:** Add a Data Model subsection under Functional Requirements:

```
### Data Model

**Meeting record:**
| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key; generated server-side |
| `title` | string | Yes | Max length not specified in BRD; implementors should set a reasonable limit |
| `completedAt` | timestamp | Yes | Stored in UTC |
| `transcript` | text | No | Null when notes-only |
| `notes` | text | No | Null when transcript-only |
| `contentSource` | enum(transcript\|notes\|both) | Yes | Captured at save time |
| `createdAt` | timestamp | Yes | Auto-set server-side |
| `createdBy` | userId | Yes | Derived from authenticated session |

**Meeting participant record:**
| Field | Type | Required | Note |
|-------|------|----------|------|
| `id` | UUID | Yes | Primary key |
| `meetingId` | FK → Meeting.id | Yes | Foreign key |
| `displayName` | string | Yes | Leading/trailing whitespace trimmed; empty-after-trim rejected |
| `email` | string | No | — |
| `organization` | string | No | — |
| `role` | string | No | — |
```

---

### MEDIUM — AC-7 eval method is not evaluable

**Location:** Line 163

**Problem:** AC-7 reads: "E2E test expands advanced participant fields, saves with and without optional values, and verifies behavior." The phrase "verifies behavior" does not specify what is expected. Does saving without optional values succeed? Does saving with them succeed? What is asserted?

**Fix:** Rewrite AC-7 as:
> "E2E test expands advanced participant fields, submits the form with optional email, organization, and role fields populated, and asserts save succeeds and the saved meeting record contains those fields with correct values; separately submits the form with all optional fields empty and asserts save also succeeds."

---

### MEDIUM — AC-17 references undefined "redaction rules"

**Location:** Line 173

**Problem:** AC-17 states "Observability eval checks emitted event names and redaction rules." No redaction rules are defined in the Observability section or elsewhere in the BRD. Without explicit rules, an implementor does not know what constitutes compliant redaction.

**Fix:** Add a Redaction Schema entry to the Observability section, Log events subsection:

> **Redaction schema** — Allowed labels: `result`, `field`, `content_source`, `route`, `flag`. Forbidden values: meeting titles, participant names, email addresses, transcript text, notes text, raw identifiers (user IDs, meeting IDs). Implementations must strip or hash any field that could contain forbidden values before emitting to metrics or logs. Label cardinality must not exceed 20 unique values per label name across the fleet.

Then update AC-17 to read: "Observability eval checks emitted event names match the defined log event schema and that no forbidden values appear in metric labels or log fields."

---

### MEDIUM — Participant normalization is unaddressed

**Location:** Parent task notes; absent from BRD Open Questions

**Problem:** The PM handoff flagged participant normalization as an open question. The BRD requires displayName but says nothing about how to handle "John  Doe" (double space), "Doe, John" (inverted), "JOHN DOE" (all caps), or "john.doe@example.com" (email used as displayName). These variations affect downstream memory quality and participant matching.

**Fix:** Add to FR participant requirements:

> "Display names are stored as-entered. Leading and trailing whitespace is trimmed. Empty-only inputs after trimming are rejected as invalid. Name inversion (Last, First) and capitalization normalization are out of scope for BRD-02; deferred to BRD-03 participant handling."

---

### LOW — "Advanced affordance" is undefined

**Location:** Lines 51, 71, 163, 174

**Problem:** "Advanced affordance" appears 4 times but is never defined. Is it a disclosure triangle? An expandable section? A separate route? No BRD-01 component is referenced for this progressive disclosure pattern.

**Fix:** Add a parenthetical to FR line 51:
> "Allow optional participant fields for email, organization/company, and role/title behind an advanced affordance (e.g., a collapsible 'Show advanced participant details' section per participant row, or a shared collapsible 'Advanced' section at the bottom of the participant list that expands/collapses all optional fields per participant)"

---

### LOW — AC-18 inherits BRD-01 accessibility without enumerating specific expectations

**Location:** Line 175

**Problem:** AC-18 reads "Manual import route inherits BRD-01 app shell accessibility expectations." BRD-01 is a large document; referencing it as a whole does not give implementors or evaluators specific things to check.

**Fix:** Expand AC-18 to enumerate specific expectations:
> "Manual import route inherits BRD-01 app shell accessibility expectations, specifically: field labels with explicit `for`/`id` association, error announcements via ARIA live region (`role="alert"` or `aria-live="polite"`), keyboard navigation (Tab through fields in DOM order, Enter to submit, Escape to cancel/close), focus management on validation failure (focus moves to first invalid field), focus management on successful save (focus moves to meeting detail page heading), and visible focus indicators (2px outline using `--color-primary` with appropriate offset)."

---

### LOW — Character count display is only in Could Have

**Location:** Line 83 vs FR lines 54 and 73

**Problem:** FR enforces a 50,000 character combined limit (line 54). FR line 73 (Should Have) mentions explaining the limit before or during validation. A live character count display is only in "Could Have" (line 83). Users pasting large transcripts have no way to know proximity to the limit until validation fails, creating friction on the primary content path.

**Fix:** Move character count display from "Could Have" to "Should Have" under Functional Requirements. Rationale: it directly supports the 50,000 character limit enforcement and prevents user friction from discovering the limit only at validation time.

---

### LOW — No label cardinality guidance in Observability

**Location:** Observability section, lines 104–123

**Problem:** Metrics use labels (`result`, `field`, `content_source`, `route`, `flag`) with no cardinality guidance. High-cardinality labels (e.g., user IDs, meeting IDs) could be emitted without violating the letter of the spec while violating its intent.

**Fix:** Add to Observability section:
> "Label cardinality must not exceed 20 unique values per label name across the fleet to prevent metric cardinality explosion in production monitoring systems."

---

## Verification Checklist

| Requirement | Status | Notes |
|------------|--------|-------|
| All 9 BRD sections present | Pass | No missing sections |
| No TBD or empty dashes | Pass | All sections have content |
| Observability has >= 1 metric | Pass | 7 metrics defined |
| Observability has >= 1 log event | Pass | 4 log events defined |
| Feature flag: server env defined | Pass | `FF_ENABLE_MANUAL_MEETING_IMPORT` |
| Feature flag: browser env defined | Pass | `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT` |
| Feature flag: default false | Pass | Explicitly stated |
| Feature flag: 4-state table correct | Pass | Matches AGENTS.md dual namespace rule |
| Open questions noted or marked None | **Fail** | States None but 3 PM items are absent — factually inaccurate |
| NFRs have specific numeric targets | **Fail** | "noticeable delay" is not numeric; contradicts 1-second save latency |
| ACs are specific and eval-able | **Partial** | AC-7, AC-17, AC-18 are vague; AC-17 references undefined redaction schema |
| API contract specified | **Fail** | No POST endpoint, request body, or response codes |
| Data model specified | **Fail** | No meeting or participant record schema |
| Redaction schema defined | **Fail** | AC-17 references undefined rules |

---

## Recommendations

**Must fix before PM acceptance (HIGH):**

1. **Open Questions accuracy** — Add the explicit resolution table for the 3 PM items. The current "None" statement is factually inaccurate.
2. **NFR numeric separation** — Give Form interactivity a numeric client-side threshold (<100ms) and clearly separate it from the server-side Save latency NFR (1 second).

**Should fix before Phase 1 implementation (MEDIUM):**

3. **Add API contract** — POST path, request body shape, and response codes are blocking for backend implementation.
4. **Add data model** — Meeting and participant record field names and types at minimum.
5. **Fix AC-7 eval method** — rewrite to match the specificity of AC-1 through AC-6.
6. **Define redaction schema** — add to Observability and update AC-17 accordingly.
7. **Address participant normalization** — add whitespace trimming rule and defer advanced normalization to BRD-03.

**Nice to have for UX quality (LOW):**

8. Define "advanced affordance" explicitly in FR.
9. Enumerate AC-18 accessibility specifics rather than referencing BRD-01 as a whole.
10. Promote character count display from Could Have to Should Have.
11. Add label cardinality guidance to Observability.

---

## Prior Work

This review supersedes `specs/domain/brd-02-refiner-review.md` (task t_3be30df3). The following issues from the prior review are carried forward with refined fixes:

| Issue ID | Title | Status in this review |
|----------|-------|----------------------|
| OQ-MISMATCH | Open Questions section misrepresents state | Carried forward as HIGH; fix expanded |
| NFR-CONTRADICTION | NFR internal contradiction | Carried forward as HIGH; fix expanded |
| NO-API-CONTRACT | No POST endpoint shape | Carried forward as MEDIUM |
| NO-DATA-MODEL | No data model | Carried forward as MEDIUM |
| AC7-VAGUE | AC-7 eval method not evaluable | Carried forward as MEDIUM; fix rewritten |
| AC17-UNDEFINED-REDACTION | AC-17 references undefined redaction rules | Carried forward as MEDIUM; fix expanded |
| PARTICIPANT-NORMALIZATION | Participant normalization unaddressed | New MEDIUM (parent task note) |
| ADVANCED-AFFORDANCE-UNDEFINED | "Advanced affordance" undefined | Carried forward as LOW |
| AC18-VAGUE | AC-18 inherits BRD-01 without specifics | Carried forward as LOW; fix expanded |
| CHAR-COUNT-COULD-HAVE | Character count display only in Could Have | Carried forward as LOW |
| NO-CARDINALITY-GUIDANCE | No label cardinality guidance | Carried forward as LOW |
