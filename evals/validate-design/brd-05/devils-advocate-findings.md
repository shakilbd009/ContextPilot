# Devil's Advocate Analysis — BRD-05 Manual Upcoming Meeting Creation

## Area: BRD-04 /ready Endpoint Dependency Reporting
**Spec claim:** AC-19 and FR (GET /ready) state that BRD-04 queue degradation is visible in readiness details "where app conventions allow," but must not make BRD-05 unavailable when trigger failure can be safely skipped or surfaced.
**Devil's advocate analysis:** The /ready endpoint spec uses deliberately vague language—"where app conventions allow" and "degraded-but-available" signals without a defined contract. If BRD-04 queue is down, the /ready response could either (a) report partial degradation and still return 200, or (b) return 503 and block meeting creation entirely. The spec does not define which behavior is correct, leaving an ambiguity that could cause false failures in production. A load balancer or orchestrator reading /ready could interpret a 200 with a "BRD-04 degraded" detail as fully healthy and route traffic normally, while downstream monitoring interprets a 503 as unhealthy and pulls the instance. The two interpretations are mutually exclusive and both plausible.
**Severity:** High
**Recommendation:** Define explicit /ready contract: return 200 whenever upcoming_meetings storage is healthy, regardless of BRD-04 queue state. Include a JSON body detail field `brd04_trigger: "available"|"degraded"|"unavailable"` so callers can emit alerts without confusing availability with health.

---

## Area: Edit Window Clock Reference
**Spec claim:** FR-9 allows editing until 15 minutes after scheduled_start. FR-8 validates scheduled_start must be no earlier than 15 minutes before server current time. AC-8/AC-9 test with "controlled clock."
**Devil's advocate analysis:** FR-9 says "until 15 minutes after scheduled_start"—but this could mean either (a) 15 min after the originally saved scheduled_start (if the meeting was rescheduled), or (b) 15 min after the current server time when the edit request arrives, measured against the scheduled_start stored in the record. If a user creates a meeting at 10:00, scheduled_start = 11:00, the edit window is 11:15. If the user then edits to reschedule to 12:00 at 11:30, does the window reset to 12:15, or does it stay at 11:15? The spec is silent. AC-8 tests "users can edit until 15 minutes after scheduled_start" but never clarifies whether rescheduling reopens or shifts the window. Implementation ambiguity here could cause legitimate edits to be unexpectedly rejected.
**Severity:** High
**Recommendation:** Clarify FR-9: edit window is anchored to the stored scheduled_start value at time of edit evaluation, and rescheduling to a new time re-anchors the window to 15 min after the new scheduled_start. Include this behavior in AC-8 test cases.

---

## Area: Normalized Title Definition for Duplicate Detection
**Spec claim:** FR-24 says UI warns on duplicate "normalized title and scheduled_start." No definition of normalized title is provided.
**Devil's advocate analysis:** "Normalized" is undefined. Ambiguities: lowercase vs original case? Trim leading/trailing whitespace? Collapse internal whitespace ("Team  Sync" → "Team Sync")? Strip diacritics? Ignore punctuation? Consider a meeting titled "Q1 Planning" and one titled "Q1  planning" (double space, lowercase)—are these flagged as duplicates? If normalization is too aggressive, false positives annoy users. If too conservative, duplicates slip through. If two users in the same org both create "Q1 Planning" at 10:00, only the second sees a warning—but the spec never addresses cross-user vs same-user scope. Also, what is the time window for "same scheduled_start"? Exact minute? Within 5 minutes?
**Severity:** Medium
**Recommendation:** Define "normalized title" explicitly in FR-24 or a referenced section: trim leading/trailing whitespace, collapse internal whitespace, lowercase, strip common punctuation. Scope duplicate detection to the same authenticated user (not all users). Define time window as same scheduled_start minute (±0 minutes) or specify a tolerance.

---

## Area: Participant Email as Identity — Email Change Breakage
**Spec claim:** FR-4 and FR-6 use email as a structured participant field supporting BRD-04 matching. FR-16 defines meaningful edit to include participant email changes. No mention of email-as-identity stability over time.
**Devil's advocate analysis:** If a participant changes their email address (e.g., user renames from john@old.com to john@new.com), BRD-04 matching that relies on email as a stable identifier breaks silently. FR-16 treats changing a participant's email as a normal meaningful edit, but the BRD-04 matching logic that depends on that email for historical meeting correlation is not addressed. When a user's corporate email is rotated during offboarding/onboarding, BRD-04 may lose all prior meeting context for that participant. The spec also does not address whether email uniqueness is enforced—if two participants somehow have the same email, is that allowed, or does it cause ambiguity in matching?
**Severity:** Medium
**Recommendation:** Add a note to FR-4 or FR-16 acknowledging that email is used as a matching identifier and email changes may affect BRD-04 matching continuity. Consider whether a stable opaque participant ID (not email) should be the primary matching handle, with email as a mutable attribute.

---

## Area: Soft Cancellation Permanence and Uncancel
**Spec claim:** FR-10: cancellation sets status=cancelled, prevents future briefing generation, hides from default views, preserves history. No mention of undo/uncancel capability.
**Devil's advocate analysis:** "Preserves history" is ambiguous—does it mean the record is immutable forever (soft delete semantics), or that cancellation is a state that could theoretically be reversed (soft disable semantics)? If a user accidentally cancels a meeting, can they "uncancel" it by changing status back to scheduled? The Risks table mentions "soft-cancelled records accumulate without purge policy" suggesting they persist indefinitely. But if uncancel is not supported, a user who cancels then realizes they need the briefing has no recourse except creating a new meeting record (with a new ID, breaking briefing continuity). The spec neither allows nor prohibits uncancel, leaving the door open to scope creep in either direction.
**Severity:** Medium
**Recommendation:** Explicitly state in FR-10 that cancellation is permanent and status cannot be reverted to scheduled. If accidental cancellation recovery is a future requirement, flag it as a deferred could-have in FR deferred list.

---

## Area: Meaningful Edit vs Data Migration
**Spec claim:** FR-16 defines meaningful edit classes: title, scheduled_start, description, client/organization, participant email, display_name, organization. All are treated equivalently for stale-briefing triggering.
**Devil's advocate analysis:** Editing participant A to have participant B's values (swapping display_name and email between two participant rows) is classified as a "meaningful edit" but is arguably data migration rather than intentional change. More importantly, changing participant email from A to B could be an error (wrong row selected in UI) that the user doesn't notice until briefings start referencing the wrong person. The spec treats all meaningful edit classes identically, but in practice, a display_name-only change is much less likely to affect briefing quality than an email change. Treating them identically for stale-briefing purposes may cause excessive regeneration triggering for trivial edits (e.g., typo fixes) while undertriggering for semantic changes that the matching algorithm doesn't weight appropriately. The spec also doesn't address whether adding a participant counts as meaningful (it doesn't appear in FR-16's explicit list).
**Severity:** Low
**Recommendation:** Add participant add/remove to FR-16 meaningful edit classes. Consider whether trivial typo fixes to display_name should be weighted differently for regeneration urgency, or document that all meaningful edits trigger full regeneration regardless of scope.

---

## Area: Privacy-Safe Diagnostics — Hash Algorithm Unspecified
**Spec claim:** FR-20: operator diagnostics use "opaque or hashed identifiers only," must not include raw private content. Log events use `user_id_hash` and `upcoming_meeting_id_hash`. No hash algorithm specified.
**Devil's advocate analysis:** If the hash algorithm is weak (e.g., MD5 of a predictable user ID), it may be vulnerable to collision or rainbow table attacks—especially if the identifier space is small or predictable (sequential UUIDs, incrementing integers). FR-20 explicitly bans raw user IDs, meeting titles, emails from logs, but the replacement hash must be computationally irreversible to count as privacy-safe. A naive CRC32 or truncated MD5 of a sequential UUID is not privacy-safe. The spec also does not specify whether the same underlying ID always produces the same hash (for correlation across log lines) or whether salts are used per-session. If salts are used, log correlation for debugging becomes impossible. If no salt, the hash must be slow (bcrypt/argon2) or at minimum collision-resistant (SHA-256).
**Severity:** High
**Recommendation:** Specify in FR-20 or the Observability section which hash algorithm is used (SHA-256 truncated to fixed length is acceptable; bcrypt if slow is better). Document whether hash is deterministic per-instance (no salt) to allow log correlation, and justify the tradeoff.

---

## Area: 50-Participant Limit — Hard DB Limit or Soft Target
**Spec claim:** NFR Scale: "MVP supports at least 100 active upcoming meetings per user and up to 50 participants per upcoming meeting without degrading p95 targets."
**Devil's advocate analysis:** "Up to 50 participants" is stated as a target in fixture tests but is not enforced as a hard constraint in the data model (FR-6 has no maximum cardinality check) or validation rules (FR-82 has no `too_many_participants` validation class). A user could submit a meeting with 200 participants and the spec does not explicitly say this fails. The NFR uses "at least" and "up to" suggesting these are floor/ceiling targets, not hard invariants. If the 50-participant limit is a performance target rather than a business rule, it should be enforced at the validation layer with a specific error (FR-82's `too_many_participants` is listed but never tied to this limit). Without enforcement, an attacker or buggy client could submit a 500-participant meeting, bypass the validation class, and degrade the server.
**Severity:** Medium
**Recommendation:** Decide whether 50 participants is a hard validation limit (enforced with `too_many_participants` rejection) or a performance target. If hard, add explicit max constraint to FR-6 data model and FR-82 validation class. If soft target, document in NFR that exceeding it may degrade performance with no guaranteed SL.

---

## Area: Symmetry of Edit and Cancellation Windows
**Spec claim:** FR-9 (edit window) and FR-10 (cancel window) are both "until 15 minutes after scheduled_start." The Risks table does not flag this as intentional design symmetry.
**Devil's advocate analysis:** The identical 15-minute window for both edit and cancel may be intentional symmetry (clean mental model) or coincidental (independent decisions that happen to align). If intentional, it should be documented as a design principle. If coincidental, future requirements might decouple them—for example, a could-have requirement to allow cancellation only up to the meeting start time (no post-meeting cancellation). The current spec gives no guidance on which scenario applies. More critically, if the edit window and cancellation window are meant to be the same boundary, the implementation should use a single shared function to compute the window deadline rather than two independent implementations that could drift apart.
**Severity:** Low
**Recommendation:** State explicitly in FR-9/FR-10 or the design rationale that the 15-minute post-window is intentional symmetry. Require a shared `is_within_edit_window(scheduled_start)` utility used by both edit and cancel enforcement.

---

## Area: AC-14 "Stale" Briefing Meaning
**Spec claim:** AC-14: "meaningful edits queue BRD-04 regeneration or mark active briefing stale per BRD-04 contract while keeping latest completed briefing visible." FR-18 echoes this.
**Devil's advocate analysis:** "Stale" is used but not defined in BRD-05. Does stale mean (a) the briefing is inaccessible (403/404 when accessed), (b) the briefing is visible but clearly labeled "stale, regenerating" with a warning banner, or (c) the briefing is hidden from default view but retrievable via direct URL? AC-14 says "keeping latest completed briefing visible" which implies option (b)—visible but outdated. But if the stale briefing is visible while regeneration runs, a user could read it before the new one completes and act on outdated information. The spec never defines the user-facing behavior when a briefing transitions to stale, the UX label applied, or whether the new briefing must complete before the stale one is hidden. Also, what triggers regeneration to start—any meaningful edit, or only edits that affect briefing-relevant fields (title, participants, scheduled_start)? Editing a meeting description might not affect briefing quality but would still trigger regeneration under a literal reading of FR-16.
**Severity:** High
**Recommendation:** Define "stale" behavior explicitly: stale briefings remain visible with a visible "outdated—regenerating" status indicator, and the new briefing must complete before the stale is fully superseded. Consider narrowing "meaningful edit" for briefing regeneration purposes to only fields that affect briefing quality (participant changes, title changes, scheduled_start changes)—not description changes.