# Unit Eval: brd-02-manual-meeting-import

> 🟢 Passing — all unit scenarios validated via `go test ./internal/meeting/...` (60/60 tests pass, 2026-05-30)

## Scope

Unit tests for Manual Meeting Import (BRD-02): validation logic, idempotency token handling, character count enforcement, form field constraints, and content-source derivation.

---

## Validation Tests

### Title Validation

| Scenario | Input | Expected |
|----------|-------|----------|
| Title required | `title: ""` | 400, `field: "title"` |
| Title max 500 chars | `title: "a" × 501` | 400, `field: "title"` |
| Title valid at 500 chars | `title: "a" × 500` | 201 |
| Title whitespace trimmed | `title: "  My Meeting  "` | 201, stored as "My Meeting" |
| Title empty after trim | `title: "   "` | 400, `field: "title"` |

### Date/Time Validation

| Scenario | Input | Expected |
|----------|-------|----------|
| completedAt required | `completedAt: ""` | 400, `field: "completedAt"` |
| completedAt invalid format | `completedAt: "not-a-date"` | 400, `field: "completedAt"` |
| completedAt valid ISO 8601 | `completedAt: "2026-05-20T14:00:00Z"` | 201 |
| completedAt in future | `completedAt: "2099-12-31T23:59:59Z"` | 201 (no restriction in BRD-02) |

### Participant Validation

| Scenario | Input | Expected |
|----------|-------|----------|
| Participants required | `participants: []` | 400, `field: "participants"` |
| displayName required | `participants: [{ displayName: "" }]` | 400, `field: "participants[0].displayName"` |
| displayName whitespace trimmed | `participants: [{ displayName: "  Alice  " }]` | 201, stored as "Alice" |
| displayName empty after trim | `participants: [{ displayName: "   " }]` | 400, `field: "participants[0].displayName"` |
| Valid optional email | `participants: [{ displayName: "Alice", email: "alice@example.com" }]` | 201 |
| Valid optional organization | `participants: [{ displayName: "Alice", organization: "ACME" }]` | 201 |
| Valid optional role | `participants: [{ displayName: "Alice", role: "Engineer" }]` | 201 |
| All optional fields populated | `participants: [{ displayName: "Alice", email: "alice@example.com", organization: "ACME", role: "Engineer" }]` | 201 |

### Content Validation

| Scenario | Input | Expected |
|----------|-------|----------|
| Transcript or notes required | `transcript: "", notes: ""` | 400, `field: "content"` |
| Transcript only | `transcript: "meeting notes here", notes: ""` | 201 |
| Notes only | `transcript: "", notes: "meeting notes here"` | 201 |
| Both present within limit | `transcript: "A" × 25000, notes: "B" × 25000` | 201 |
| Combined exactly 50,000 chars | `transcript: "X" × 25000, notes: "Y" × 25000` | 201 |
| Combined 50,001 chars | `transcript: "X" × 25001, notes: "Y" × 25000` | 400, `field: "content"` |

### Content Source Derivation

| Scenario | Input | Expected `contentSource` |
|----------|-------|--------------------------|
| Transcript only | `transcript: "hello", notes: ""` | `"transcript"` |
| Notes only | `transcript: "", notes: "hello"` | `"notes"` |
| Both present | `transcript: "hello", notes: "also hello"` | `"both"` |

---

## Idempotency Token Tests

| Scenario | Input | Expected |
|----------|-------|----------|
| Token required | `idempotencyToken: ""` | 400 |
| Valid UUID token | `idempotencyToken: "550e8400-e29b-41d4-a716-446655440000"` | 201 |
| First submission with token | Valid payload + new token | 201, meeting created |
| Duplicate token within 24h | Same token + valid payload | 409, `error: "duplicate"`, original meeting ID returned |
| Token older than 24h | Same token + payload, token TTL exceeded | 201, new meeting created (token expired) |
| Malformed token | `idempotencyToken: "not-a-uuid"` | 400 |

---

## Character Count Tests

| Scenario | Input | Expected |
|----------|-------|----------|
| 0 combined chars | `transcript: "", notes: ""` | blocked by content-required check |
| 1 char transcript | `transcript: "a", notes: ""` | 201 |
| 44,999 combined chars | transcript 44999, notes empty | 201, no warning |
| 45,000 combined chars | transcript 22500, notes 22500 | 201, warning triggered (≥45,000) |
| 50,000 combined chars | transcript 25000, notes 25000 | 201, at limit |
| 50,001 combined chars | transcript 25001, notes 25000 | 400, `field: "content"` |

---

## Running

```bash
# From project root
go test ./internal/validator/... -v
go test ./internal/handlers/... -v
```