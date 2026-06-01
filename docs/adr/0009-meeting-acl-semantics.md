# ADR-0009: Meeting ACL Semantics for Cross-Meeting Briefing Authorization

**Status:** Accepted
**Date:** 2026-05-23
**BRD:** brd-04
**Component:** Authorization / Cross-BRD Integration

---

## Context

BRD-04 FR-19 (Authorization Boundary) requires that briefings and source details are visible only to users authorized to view the upcoming meeting AND all included source meetings. BRD-02 establishes `createdBy` as the meeting owner but does not define a generalized meeting ACL model. Before BRD-04 can implement authorization checks, the semantics of "authorized to view" must be defined.

---

## Decision

**Option A (Owner-only) adopted for MVP.** Only the meeting creator (`createdBy`) can access the meeting, its memory, and its briefing. A user can only generate a briefing for an upcoming meeting they own, and can only include source meetings they own in that briefing.

---

## Rationale

- Owner-only semantics are the simplest and safest for MVP. They are consistent with BRD-02's data model where `createdBy` is the sole identity attached to a meeting record.
- Cross-user briefings (where meeting A owned by user X is a source for a briefing owned by user Y) can be addressed in a future BRD when sharing semantics and trust boundaries are better understood.
- FR-19's current phrasing ("authorized to view the upcoming meeting and all included source meetings") is compatible with owner-only semantics.
- Participant-based access (Option B) would require BRD-02's participant model to be extended with access-control semantics, which is out of scope for Phase 1.

---

## Consequences

- **Positive:** Simple, auditable, no cross-user data leakage risk. Authorization checks are a single `WHERE created_by = $session_user_id` predicate.
- **Negative:** Users cannot share briefings with meeting participants who are not owners. This may be limiting for org use cases where an executive assistant manages briefings for a manager. This is an accepted MVP limitation.
- **Neutral:** The `meeting_participants` table in BRD-02 is used for content matching (participant email signals) but not for authorization.

---

## Future Consideration

If org-wide meeting visibility is needed in a future phase, a new BRD should define the ACL model extension. The briefing exclusion mechanism (FR-16/FR-17) provides a partial workaround: users who own source meetings can exclude them from others' briefings, but this is not the same as controlling access.