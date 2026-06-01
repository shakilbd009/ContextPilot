# BRD-05 Manual Upcoming Meeting Creation — Decision Record

---
brd_id: brd-05
title: Manual Upcoming Meeting Creation Decision Record
curated: 2026-05-23
updated: 2026-05-25
approval: 2026-05-24 — status changed from Deferred/PM HOLD to Approved per user confirmation (t_5be35a0e)
pm_approver: pm profile
approver_notes: All critical/high gate blockers resolved or accepted as implementation-phase work; no unresolved OQs remain
source: specs/domain/brd-05-manual-upcoming-meeting-creation.md (approved raw BRD, brainstormer, 2026-05-23)
references:
  - docs/adr/0009-meeting-acl-semantics.md
  - specs/domain/brd-04-pre-call-briefing.md
  - specs/security/brd-06-privacy-retention-controls.md

---

## D-1: Matching-Aware Manual Creation

**Decision:** Users manually enter upcoming meeting data with optional structured participant metadata (display name, email, organization) and optional meeting-level client/organization to support BRD-04 briefing matching.

**Rationale:** BRD-04 briefing quality depends on accurate participant-signal matching against prior meeting memory. Participant email and organization fields are the primary matching signals. Meeting-level client/organization provides additional context for title-agnostic matching. Making these fields optional avoids blocking creation when users lack full metadata, while ensuring that when entered, they are persisted and available to the BRD-04 matching pipeline.

**Source:** BRD-05 FR-3 (manual creation fields), FR-4 (structured participant metadata), BRD-04 FR-4 (matching signals).

**Disposition:** FR-3, FR-4 in raw BRD; no further ADR needed.

---

## D-2: Owner-Only MVP Per ADR-0009

**Decision:** Only the `created_by` owner can create, view, list, edit, or cancel their upcoming meetings. Participants do not receive access rights. Participant metadata exists solely to support BRD-04 briefing matching — it carries no authorization weight.

**Rationale:** ADR-0009 adopts Option A (Owner-only) for MVP, citing: simplicity, auditable single-predicate checks (`WHERE created_by = $session_user_id`), and avoidance of cross-user data leakage risk. BRD-05 FR-7 and FR-11 encode this. Participant-based access (Option B in ADR-0009) would require extending BRD-02's participant model with ACL semantics, which is explicitly out of Phase 1 scope.

**Source:** ADR-0009 (docs/adr/0009-meeting-acl-semantics.md), BRD-05 FR-7 (status model), FR-11 (owner-only authorization), NFR-Authorization row.

**Disposition:** Enforced via FR-11 acceptance criteria (AC-11, AC-12); no further ADR needed for MVP.

---

## D-3: Soft Cancellation

**Decision:** Cancellation sets `status=cancelled`, hides the record from default active upcoming views, preserves the meeting and briefing history, and prevents future BRD-04 briefing trigger attempts. No hard delete exists in BRD-05.

**Rationale:** Soft cancellation (FR-10) preserves audit and debug context while removing cancelled meetings from active preparation surfaces. This is consistent with the BRD-03 memory preservation principle — historical records are retained, not erased. Hard deletion and purge behavior are explicitly deferred to BRD-06 (FR-31). Cancellation does not remove associated briefing versions; those remain queryable for historical/debug purposes.

**Source:** BRD-05 FR-10 (soft cancellation), NFR-Retention row ("inherits from BRD-06"), Risks table (soft-cancelled record accumulation mitigation).

**Disposition:** FR-10 in raw BRD; hard delete deferred to BRD-06.

---

## D-4: BRD-04 Trigger Integration (FR-17 / FR-18)

**Decision:** BRD-05 creation and meaningful edits trigger BRD-04 briefing generation asynchronously and non-blocking. FR-17 queues generation on successful scheduled upcoming meeting creation when both feature flags are enabled. FR-18 queues regeneration or marks the active briefing stale on meaningful edits. Trigger failures are handled safely and observably — they do not prevent BRD-05 operations from completing.

**Rationale:** Decoupling upcoming meeting management from briefing generation keeps BRD-05 responsive (p95 < 500ms for create/update/cancel excluding async work) and avoids blocking the user on generation latency. Async enqueue target is p95 < 100ms. The "meaningful edit" definition (FR-16) is explicit — changes to title, scheduled_start, description, client/org, or any participant field — to avoid spurious regeneration triggers. Failed trigger attempts emit `briefing_trigger_failed` metrics and logs without rolling back the meeting operation.

**Source:** BRD-05 FR-17, FR-18, FR-19, NFR-Create/Update/Cancel latency, NFR-Briefing trigger enqueue latency; BRD-04 FR-2 and FR-15 (stale behavior).

**Disposition:** FR-17, FR-18, FR-19 in raw BRD; FR-16 defines meaningful edit surface; BRD-04 owns regeneration/staleness contract.

---

## D-5: BRD-06 Retention Inheritance

**Decision:** Hard deletion, purge behavior, retention policy enforcement, and data lifecycle controls are explicitly deferred to BRD-06 Privacy & Retention Controls. BRD-05 defines only soft cancellation (`status=cancelled`) as the cancellation mechanism.

**Rationale:** BRD-05 NFR-Retention row states: "Upcoming meeting retention, purge, export, and hard delete behavior inherit final policy from BRD-06 Privacy & Retention Controls." BRD-05 FR-31 (could-have) defers hard delete. BRD-06 is the appropriate owner for retention policy, purge triggers, and any hard-delete authorization that may be needed in a future phase. BRD-05's soft cancellation produces `cancelled` records that remain queryable until BRD-06 defines what happens next.

**Source:** BRD-05 NFR-Retention, FR-31, Risks table (soft-cancelled record accumulation); BRD-06 (specs/security/brd-06-privacy-retention-controls.md — in draft at time of BRD-05 curation).

**Disposition:** BRD-06 owns retention lifecycle; BRD-05 produces only soft-cancelled state.

---

## D-6: Scope Boundaries

**Decision:** The following are explicitly out of BRD-05 scope and deferred to future BRDs:

| Deferred capability | Reason for deferral |
|--------------------|--------------------|
| Provider sync / external calendar import | Requires separate provider connectors BRD; out of scope for matching-aware manual creation MVP |
| Recurrence and series management | Complexity beyond MVP; requires separate recurrence BRD |
| Reminders and delivery (email, push, calendar) | Delivery channel integration out of scope; deferred to future delivery BRD |
| Participant-based access control | Owner-only MVP per ADR-0009; participant ACL extension deferred to future BRD |
| Hard delete and purge | Deferred to BRD-06 retention controls |
| Availability scheduling, finding times, RSVP | Out of scope for manual meeting entry MVP |

**Source:** BRD-05 FR-13 (calendar-style view non-goals), FR-27–FR-32 (could-have deferred items), Non-Goals section.

**Disposition:** Documented in raw BRD as explicit scope boundaries; not implemented in BRD-05 MVP.

---

## D-7: upcoming_meetings Table Defined for BRD-05

**Decision:** The `upcoming_meetings` table was undefined — it appeared in BRD-04 data model references but was never defined in any published BRD. BRD-05 FR-5 now defines the table with full CRUD surface (create, list, detail, edit, soft-cancel) sufficient to close the BRD-04 dependency and support the briefing trigger contract.

**Authorization:** Follows ADR-0009 owner-only semantics. `created_by` is the owner reference; participants have no access rights.

**Source:** BRD-05 FR-5 (data model).

**Disposition:** FR-5 defines `upcoming_meetings` table schema; FR-6 defines `upcoming_meeting_participants`. Resolved — the table is now defined in BRD-05.

---

## D-8: Status Change — Deferred/PM HOLD → Approved

**Decision:** BRD-05 curated package status updated from Deferred/PM HOLD to Approved. Curated package now matches raw/domain BRD status (Approved) and user confirmation that BRD-05 is good to continue.

**Rationale:** Raw/domain BRD (`specs/domain/brd-05-manual-upcoming-meeting-creation.md`) shows Approved status. User explicitly confirmed BRD-05 is approved to continue. The curated package's Deferred/PM HOLD state was a downstream artifact of a prior hold that has since been released. Decision-record.md `pm_hold` field (added 2026-05-24 per t_1e69d59b) is now cleared and replaced with approval timestamp.

**Source:** t_5be35a0e (this task); parent PM task t_4f06c938.

**Disposition:** Curated brd.md, decision-record.md, validator-findings.md, and implementation-readiness.md all updated to Approved.

---

## D-9: PM Gate Chronology and Post-Repair Re-Gate Approval

**Decision:** BRD-05 passed PM implementation gate after a repair-and-revalidation cycle. All critical/high spec blockers from the initial validate-design run (t_3f0ab4f1) were resolved or explicitly dispositioned before the re-gate (t_1d1ab3a7) returned APPROVE.

**Gate chronology:**

| Date | Task | Event |
|------|------|-------|
| 2026-05-24 11:56 | t_d15bf873 | completeness-score PASS — 20/20, 0 open items |
| 2026-05-24 12:15 | t_3f0ab4f1 | validate-design NEEDS_WORK — 4 critical, 12 high |
| 2026-05-24 12:17 | t_87437820 | PM gate verdict: REPAIR (validate-design NEEDS_WORK; security timed out) |
| 2026-05-24 12:22 | t_a9a1d8bb | spec-writer: 11 PM-required repairs applied to brd.md |
| 2026-05-24 12:28 | t_f52036ac | ops: OpenAPI contract repair — all 5 /upcoming endpoints + 4 schemas |
| 2026-05-24 12:28 | t_1aba6add | validator: post-repair revalidation PASS — all 5 validators, security re-run |
| 2026-05-24 12:41 | t_1d1ab3a7 | PM re-gate: APPROVE |
| 2026-05-24 12:41 | t_87437820 | original gate completed with APPROVE; released t_07744ac5 and t_e94c6b27 |

**Source:** t_87437820 (PM gate), t_1d1ab3a7 (PM re-gate), t_a9a1d8bb (spec repair), t_f52036ac (OpenAPI repair), t_1aba6add (revalidation), t_3f0ab4f1 (initial validate-design).

**Disposition:** Gate history documented for audit trail; no further action required.

---

## D-10: Accepted Non-Blocking Items from PM Gate t_87437820

**Decision:** The following items were identified during validate-design re-validation and explicitly accepted by PM as non-blocking for BRD-05 activation. They require future decisions or implementation-phase work but do not block the graduation package.

| Item | Owner | Disposition | Rationale |
|------|-------|-------------|-----------|
| Observability stack implementation | Implementation team | Implementation phase | BRD now specifies all metrics, histograms, and log events; implement per production-checklist |
| Feature flag enforcement implementation | Implementation team | Implementation phase | Registry is complete; enforcement code is implementation-phase work |
| CSRF explicit spec reference | PM/spec-writer | Carried as implementation/spec note | App-level CSRF middleware is shared infrastructure; BRD-05 endpoints must respect it |
| Per-user meeting count hard ceiling | PM | PM follow-up | NFR Scale target is 100/user but no hard ceiling defined; PM decision needed |
| Unicode normalization for title (FR-24) | PM | PM follow-up | NFC vs NFD vs NFKC not yet specified; Unicode Standard Annex #15 |
| BRD-04 participant email continuity | BRD-04 owner | BRD-04 owner follow-up | Email used as BRD-04 matching identity; email change after creation may break continuity |

**Source:** t_87437820 PM re-gate comment, t_1d1ab3a7 metadata.

**Disposition:** All six items are documented in the Open Questions table below with owning parties. None block the graduation package.

---

## Summary Table

| ID | Decision | Deferred to |
|----|----------|-------------|
| D-1 | Matching-aware manual creation with participant/org metadata | BRD-04 matching pipeline |
| D-2 | Owner-only MVP (ADR-0009 Option A) | Future BRD for participant ACL |
| D-3 | Soft cancellation (status=cancelled, no hard delete) | BRD-06 |
| D-4 | Async non-blocking BRD-04 trigger on create/edit | BRD-04 regeneration/staleness contract |
| D-5 | Retention and hard delete deferred to BRD-06 | BRD-06 |
| D-6 | Provider sync, recurrence, reminders, participant access, hard delete, availability scheduling explicitly out of scope | Respective future BRDs |
| D-7 | upcoming_meetings table defined for BRD-05 | None — defined in BRD-05 FR-5 |
| D-8 | Status change Deferred/PM HOLD to Approved | None |
| D-9 | PM gate chronology: repair cycle → re-gate APPROVE | None — audit trail |
| D-10 | 6 non-blocking items accepted by PM at re-gate | Per-item owner (PM, BRD-04 owner, implementation team) |

---

## Open Questions

All primary design decisions are resolved. The following implementation-phase items were identified during validate-design re-validation (2026-05-24) and explicitly accepted by PM as non-blocking at the re-gate (t_87437820/t_1d1ab3a7). They require decisions or actions from the identified owners during or after implementation, not during spec curation.

| Item | Owner | PM Gate Disposition | Status |
|------|-------|---------------------|--------|
| Per-user meeting count hard ceiling | PM | Accepted as PM follow-up; NFR Scale 100/user is a soft target | Open — PM decision needed |
| Unicode normalization for title (FR-24) | PM | Accepted as PM follow-up; NFC/NFD/NFKC not yet specified | Open — PM decision needed |
| Participant email change → BRD-04 matching continuity | BRD-04 owner | Accepted as BRD-04 owner follow-up; email is BRD-04 primary matching signal | Open — BRD-04 owner action required |
| CSRF explicit spec reference | PM/spec-writer | Carried as implementation/spec note; app-level CSRF middleware is shared infrastructure | Deferred to implementation |
| Observability stack implementation | Implementation team | Deferred to activation; spec defines all metrics/histograms/logs | Deferred to implementation |
| Feature flag enforcement implementation | Implementation team | Deferred to activation; registry is complete | Deferred to implementation |

These items do not block the graduation package. They were reviewed and accepted as non-blocking by PM at the implementation-readiness gate.

**Source:** t_87437820 PM re-gate metadata (accepted_non_blocking_items list), t_1d1ab3a7 completion metadata.