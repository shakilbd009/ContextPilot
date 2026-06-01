# ADR-002: Manual Meeting Import — Validation Strategy

> Status: Accepted

## Context

BRD-02 requires field-level validation (title, date/time, at least one participant with display name, at least one content source, combined transcript+notes ≤ 50,000 characters) before a meeting can be saved. The BRD does not specify whether validation runs client-side, server-side, or both, and whether client-side validation is a performance optimization or the authoritative gate. This decision affects API security posture, UX responsiveness, and implementation complexity.

## Decision

We will implement **dual-layer validation**: client-side validation for immediate UX feedback, and server-side validation as the authoritative gate. The server will never trust client-side validation alone.

### Client-side (SvelteKit form actions / Reactively)
- Real-time or on-submit client-side checks for: required fields (title, date/time, participant display name, content source), character count against the 50,000 limit, and participant count ≥ 1.
- Client-side validation provides immediate feedback without a network round-trip.
- Client-side validation errors do NOT block submission — the form is always submitted and server response handles errors.

### Server-side (Go handler)
- Full re-validation of all fields on every save attempt.
- Returns field-level 400 errors with enough detail for E2E assertions (e.g., `{ "field": "title", "message": "required" }`).
- No field is trusted from the client; all client data is re-evaluated.

## Rationale

Client-side-only validation is insufficient because:
- Browser DevTools can bypass JavaScript validation.
- API clients (non-browser) must have enforced rules.
- Feature flag misconfigurations (FF=false, VITE_FF=true) could expose a client-only gate that the server would bypass.

Server-side-only validation wastes a round-trip for preventable UX feedback. The dual approach provides:
- Immediate feedback for common mistakes (via client) without server load.
- Authoritative enforcement at the server (true security boundary).

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| Simplicity | Two validation sites instead of one means potential drift between client and server rules |
| Performance | Client validation avoids a round-trip for simple mistakes; we accept one extra server call for truly invalid submissions (rare) vs adding complexity for sync'd rules |
| Latency | Users get immediate feedback but also submit and wait for server rejection on edge cases the client misses |

## Consequences

1. Validation rule logic must be maintained in two places — when a rule changes, both the SvelteKit form action/validator and the Go handler must be updated.
2. To reduce drift risk, both layers should share a common validation rule definition (e.g., a shared constants/validation package in the monorepo) if the project structure allows it.
3. Server-side validation errors always return HTTP 400 with a structured body; the client uses this to re-populate the form with the user's original input.

## Alternatives Considered

### Server-only validation
Rejected because: Forces a full round-trip for every validation failure, degrading UX for the common case of empty or obviously wrong fields. Not acceptable for a 50,000-character text input where users need instant feedback.

### Client-only validation
Rejected because: Browser-based systems can always be bypassed. The server is the security boundary. Allowing saves with missing required fields because the client said no is not acceptable.

## Date

2026-05-20