# Unit Eval: brd-05-manual-upcoming-meeting-creation

> 🔴 Failing — implementation pending

## Scope

Unit tests for Manual Upcoming Meeting Creation (BRD-05): validation logic (title required, scheduled_start window, participant identity, email format), status state machine, edit window enforcement, meaningful edit detection, and form preservation on validation failure.

Source of truth: `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` FR and AC tables.

---

## Validation Tests

### AC-02/AC-04 — Title Required

|| Scenario | Input | Expected |
|----------|---------|--------|
| Missing title | `title: ""`, valid `scheduled_start` | 400, validation_class: `missing_title` |
| Null title | `title: null`, valid `scheduled_start` | 400, validation_class: `missing_title` |
| Whitespace-only title | `title: "   "`, valid `scheduled_start` | 400, validation_class: `missing_title` |
| Valid title | `title: "Q3 Planning"`, valid `scheduled_start` | 201 Created |

### AC-05 — Scheduled Start Must Be ≥15 Minutes in the Future

|| Scenario | Input | Expected |
|----------|---------|--------|
| Exactly 15 minutes from now | `scheduled_start` = server_now + 15min | 201 Created |
| 16 minutes from now | `scheduled_start` = server_now + 16min | 201 Created |
| 14 minutes from now | `scheduled_start` = server_now + 14min | 400, validation_class: `invalid_scheduled_start` |
| 1 minute from now | `scheduled_start` = server_now + 1min | 400, validation_class: `invalid_scheduled_start` |
| In the past | `scheduled_start` = server_now - 1hr | 400, validation_class: `invalid_scheduled_start` |
| Exactly now | `scheduled_start` = server_now | 400, validation_class: `invalid_scheduled_start` |

### AC-04 — Participant Must Have Display Name or Email

|| Scenario | Input | Expected |
|----------|---------|--------|
| Display name only | participant: `{display_name: "Alice", email: null}` | 201 Created |
| Email only | participant: `{display_name: null, email: "alice@example.com"}` | 201 Created |
| Both name and email | participant: `{display_name: "Alice", email: "alice@example.com"}` | 201 Created |
| Neither name nor email | participant: `{display_name: null, email: null}` | 400, validation_class: `participant_missing_identity` |
| Empty string both | participant: `{display_name: "", email: ""}` | 400, validation_class: `participant_missing_identity` |
| Whitespace only name | participant: `{display_name: "  ", email: "alice@example.com"}` | 201 Created (whitespace trimmed) |

### AC-04 — Email Format Validation

|| Scenario | Input | Expected |
|----------|---------|--------|
| Valid email | `email: "alice@example.com"` | 201 Created |
| Missing @ | `email: "aliceexample.com"` | 400, validation_class: `invalid_email` |
| Missing domain | `email: "alice@"` | 400, validation_class: `invalid_email` |
| Invalid characters | `email: "ali ce@example.com"` | 400, validation_class: `invalid_email` |
| Null email (optional) | `email: null` | 201 Created (email is optional for participants) |

### AC-04 / FR-26 — Too Many Participants

|| Scenario | Input | Expected |
|----------|---------|--------|
| 50 participants (limit) | 50 valid participant rows | 201 Created |
| 51 participants | 51 valid participant rows | 400, validation_class: `too_many_participants` |

---

## Status State Machine Tests

### FR-07 / AC-10 — Status Enum

|| Scenario | Input | Expected |
|----------|---------|--------|
| Created meeting has status `scheduled` | New record | `status = "scheduled"` |
| Cancelled meeting has status `cancelled` | After cancel action | `status = "cancelled"` |
| No other status values accepted | `status: "completed"` on create | 400 or ignored |

---

## Edit Window Tests

### AC-08 / AC-09 — Edit Window Boundary

|| Scenario | Clock Offset | Expected |
|----------|-------------|----------|
| Edit allowed at T+0 (at scheduled start) | `scheduled_start` is now | Edit allowed (within window) |
| Edit allowed at T+14min | `scheduled_start` was 14 min ago | Edit allowed (within 15-min window) |
| Edit blocked at T+15min | `scheduled_start` was 15 min ago | 422 or hidden controls, validation_class: `not_editable` |
| Edit blocked at T+1hr | `scheduled_start` was 1 hour ago | 422 or hidden controls |
| Cancel allowed at T+0 | `scheduled_start` is now | Cancel allowed |
| Cancel allowed at T+14min | `scheduled_start` was 14 min ago | Cancel allowed |
| Cancel blocked at T+15min | `scheduled_start` was 15 min ago | 422 or hidden controls, validation_class: `not_cancellable` |

---

## Meaningful Edit Tests

### AC-16 / FR-16 — Meaningful Edit Detection

A meaningful edit is any change to: `title`, `scheduled_start`, `description`, `client_or_organization`, participant email, participant display_name, or participant organization.

|| Scenario | Change | Meaningful |
|----------|---------|-----------|
| Change title | `"Q3 Planning"` → `"Q3 Planning Review"` | Yes |
| Change scheduled_start | one future time → another future time | Yes |
| Change description | `"old desc"` → `"new desc"` | Yes |
| Change client_or_organization | `"Acme"` → `"Beta"` | Yes |
| Add participant row | 0 → 1 participant | Yes |
| Remove participant row | 1 → 0 participants | Yes |
| Change participant display_name | `"Alice"` → `"Alice Smith"` | Yes |
| Change participant email | `"alice@example.com"` → `"alice.smith@example.com"` | Yes |
| Change participant organization | `"Acme"` → `"Beta"` | Yes |
| Reorder participants | Swap positions of two existing participants | No (data unchanged) |
| Add then remove participant (net zero) | +1 then -1 | No |
| No field changed | Submit identical values | No |

---

## Form Preservation on Validation Error

### AC-04 / AC-15 — Server-Authoritative Validation with Form Echo

|| Scenario | Input | Expected |
|----------|---------|--------|
| Title missing on create | Empty title, valid scheduled_start, valid participants | 400, response body includes submitted `scheduled_start` and participants echoed back |
| Participant identity missing | Valid title, valid scheduled_start, one participant with no name/email | 400, response body includes submitted `title` and `scheduled_start` echoed back |
| Invalid email format | Valid title, valid scheduled_start, participant with bad email | 400, response body includes all valid fields echoed back |
| scheduled_start in past | Valid title, past scheduled_start | 400, response body includes submitted `title` echoed back |

---

## Privacy-Safe Diagnostics Tests

### AC-20 / FR-20 — No Private Content in Validation Errors

|| Scenario | Expected |
|----------|----------|
| Validation failure message | Contains no raw meeting title, description, participant names, emails, or organization values |
| Error response | Uses low-cardinality validation class enums, not free-text content from user input |
| Log output on validation failure | No raw title, description, participant PII, or organization values |
| Metric labels on validation failure | Label values are low-cardinality enums only (e.g., `validation_class: "missing_title"`), never raw user content |