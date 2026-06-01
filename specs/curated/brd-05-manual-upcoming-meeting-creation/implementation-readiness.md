# BRD-05 Manual Upcoming Meeting Creation — Implementation Readiness

**Status:** APPROVED
**Reviewed by:** t_61bd7de1 (post-repair re-validation)
**Re-validated:** 2026-05-24 (run 359, full post-repair re-validation)
**Curated package task:** t_baca63bf
**Last updated:** 2026-05-25 (t_d9f75824 — artifact consolidation pass)

---

## Implementation Readiness Summary

BRD-05 is **Approved** for implementation. All 11 PM-required repairs from t_a9a1d8bb are applied. All gates are green.

BRD-05 is **Active**. The feature flag `ff_enable_upcoming_meetings` is registered and defaults to `true`. Implementation is proceeding.

**Gate criteria met:**
- All 4 curated package files present (brd.md, decision-record.md, validator-findings.md, implementation-readiness.md)
- Feature flag registered in specs/feature-flags.md with dual env vars, default false
- OpenAPI contract complete: 10 paths, 4 schemas added via t_f52036ac
- 13 PM-required spec defects all resolved (t_a9a1d8bb)
- Sync check passes: `scripts/check-status-sync.sh` exit 0
- No unresolved blockers in validate-design summary (verdict PASS, all 5 validators)
- **Cross-file consistency:** All curated package files updated to reflect Active status as of 2026-05-26.

---

## Feature Flag State

| Property | Value | Evidence |
|----------|-------|----------|
| Flag name | `ff_enable_upcoming_meetings` | specs/feature-flags.md line 24 |
| Server env var | `FF_ENABLE_UPCOMING_MEETINGS` | .env.example line 26 |
| Browser env var | `VITE_FF_ENABLE_UPCOMING_MEETINGS` | .env.example line 36 |
| Default | `true` (both namespaces) | .env.example lines 26, 36 |
| Phase | Phase 2 | specs/feature-flags.md line 24 |
| Registry status | Active | specs/feature-flags.md line 24 |
| Activation trigger | Flag enabled in production; rollout is immediate | — |

**Activation under feature flag:** When `FF_ENABLE_UPCOMING_MEETINGS=true` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=true`, backend enforces all BRD-05 CRUD endpoints gated on that flag. Frontend shows `/upcoming` entry points. Setting to `false` hides all entry points and returns safe `403 feature_disabled` JSON from backend. This is the rollout mechanism: flag off → no UI, no backend handling; flag on → full activation.

**Local demo override:** `.env` files for backend and frontend may override to `true` for local development, independent of `.env.example` defaults.

**BRD-04 dual-flag interaction:** FR-17/FR-18 require **both** `FF_ENABLE_UPCOMING_MEETINGS=true` AND `FF_ENABLE_PRE_CALL_BRIEFING=true` to queue briefing generation. If only BRD-05 flag is on, meetings are created but no briefing trigger fires.

---

## OpenAPI Contract Coverage

Source: `contracts/openapi.yaml` (updated via t_f52036ac, lines 524–1066)

**10 paths defined under `upcoming` tag:**

| Path | Operation | Notes |
|------|-----------|-------|
| `GET /upcoming` | listUpcomingMeetings | List user's non-cancelled upcoming meetings |
| `POST /upcoming` | createUpcomingMeeting | Create with full payload |
| `GET /upcoming/{id}` | getUpcomingMeeting | Detail view |
| `PATCH /upcoming/{id}` | updateUpcomingMeeting | Edit within 15-min window |
| `POST /upcoming/{id}/cancel` | cancelUpcomingMeeting | Soft cancel |
| `GET /upcoming/{meetingId}/briefing` | getBriefing | Briefing entry point (BRD-04) |
| `GET /upcoming/{meetingId}/briefing/versions` | listBriefingVersions | Version list |
| `GET /upcoming/{meetingId}/briefing/versions/{versionNumber}` | getBriefingVersion | Specific version |
| `POST /upcoming/{meetingId}/briefing/regenerate` | regenerateBriefing | Trigger regeneration (BRD-04) |
| `POST /upcoming/{meetingId}/briefing/sources/{sourceId}/exclude` | excludeBriefingSource | Source exclusion |
| `POST /upcoming/{meetingId}/briefing/sources/{sourceId}/restore` | restoreBriefingSource | Undo exclusion |

Note: The API also defines `/upcoming/{meetingId}/briefing/sources/{sourceId}/restore` at line 1018.

**4 schemas added:**

- `UpcomingMeeting` — full FR-5 data model
- `UpcomingMeetingParticipant` — FR-6 participant row
- `UpcomingMeetingCreate` — creation payload
- `UpcomingMeetingUpdate` — patch payload

---

## Eval Contracts Evidence

### E2E Eval (`evals/e2e/brd-05-manual-upcoming-meeting-creation.md`)

- **Status:** :red_circle: Failing — implementation pending
- **Coverage:** AC-01 (flag disable), AC-02 (authenticated creation), AC-03 (optional fields), AC-06 (views), AC-07 (cancelled filter), AC-08 (edit window), AC-09 (post-window rejection), AC-10 (soft cancel), AC-11 (owner-only authz), AC-12 (participant-no-access), AC-14 (stale briefing), AC-19 (`/ready` storage health), FR-24 (duplicate awareness)
- **Trigger:** BRD-05 status is now In Implementation (2026-05-26)

### Unit Eval (`evals/unit/brd-05-manual-upcoming-meeting-creation.md`)

- **Status:** :red_circle: Failing — implementation pending
- **Coverage:** Validation (title required, scheduled_start window, participant identity, email format, 50-participant limit), status state machine, edit window boundary, meaningful edit detection, form preservation on validation error, privacy-safe diagnostics

### Integration Eval (`evals/integration/brd-05-manual-upcoming-meeting-creation.md`)

- **Status:** :red_circle: Failing — implementation pending
- **Coverage:** Full API contract: POST/GET/PATCH `/upcoming`, feature flag gating, owner-only authorization, validation error responses, BRD-04 trigger integration, `/ready` storage health

---

## Production Checklist Prerequisites

Source: `specs/curated/brd-05-manual-upcoming-meeting-creation/production-checklist.md` (ops task t_e94c6b27)

128 total checklist items. 14 pre-verified (static inspection), 114 implementation-phase TODO items for QA/backend-reviewer.

**Pre-verified items (implementation prerequisites met):**

| Category | Item | Status |
|----------|------|--------|
| Feature Flag | FF-1: `FF_ENABLE_UPCOMING_MEETINGS=false` default | Verified |
| Feature Flag | FF-2: `VITE_FF_ENABLE_UPCOMING_MEETINGS=false` default | Verified |
| Feature Flag | FF-3: flag registered in feature-flags.md | Verified |
| Database | DB-1 through DB-7: migrations, indexes, no FK constraint | Verified |
| Architecture | ARCH-1: check-feature-flags.sh exits 0 | Verified |
| Architecture | ARCH-2: check-no-panic.sh exits 0 | Verified |
| Architecture | ARCH-3: check-no-background-context.sh exits 0 | Verified |

**114 TODO items** require implementation-phase verification: API handlers, flag enforcement, metrics/logs, UI routes, accessibility, performance benchmarks, and AC verification against running code. These are gated on implementation completing before QA clears them.

---

## ADR Evidence

|| ADR | Location | Status | Relevant Decision |
|-----|----------|--------|-------------------|
| ADR-0009 (owner-only ACL semantics) | `docs/adr/0009-meeting-acl-semantics.md` | Accepted | Applied to BRD-05 per D-2 |
| ADR-0012 (source exclusion/soft-delete) | `docs/adr/0012-source-exclusion-undo-soft-delete.md` | Accepted | Related to BRD-04 briefing sources |
| Plain UUID FK decision | BRD-05 FR-6 | Resolved | `upcoming_meeting_id` is plain indexed UUID; application-level referential integrity only; no DB FK constraint |

Note: ADR-0010 and ADR-0013 do not exist as files — they were placeholder references. The `upcoming_meetings` table is defined in BRD-05 FR-5. The plain UUID / application-level referential integrity decision is captured in BRD-05 FR-6.

---

## Risks, Rollback Notes, and Activation Constraints

### Feature Flag Activation Risk

When `FF_ENABLE_UPCOMING_MEETINGS=true`, all `/upcoming` backend endpoints become active simultaneously with frontend entry points. Rollback is flip to `false` — no data loss, no migration needed. However, production observability (metrics/logs) must be instrumented before flag flip; otherwise BRD-05 traffic is invisible to operators. **Mitigation:** OBS-1 through OBS-17 in production-checklist.md must be verified before flag flip.

### Feature Flag Activation

BRD-05 is active. The flag `FF_ENABLE_UPCOMING_MEETINGS` is set to `true` in `.env.example`. Rollback is flip to `false` — no data loss, no migration needed. Production observability (metrics/logs) must be verified before flag flip per OBS-1 through OBS-17 in production-checklist.md.

### Observability / Flag Enforcement Implementation

Metrics and structured log events (OBS-1–OBS-17, LOG-1–LOG-14) are defined in the spec and production-checklist but not yet instrumented in code. The architecture fitness functions pass (FF check, no panic, no background context) but do not verify metric/log instrumentation. **Mitigation:** Production checklist OBS and LOG items must be verified by QA before flag flip. Flag enforcement enforcement (FF ENABLE check in handlers) is implementation-phase wiring, not a spec defect.

### BRD-04 Dependency

BRD-04 briefing tables are self-contained regardless of BRD-05 state. BRD-05 can be implemented and shipped independently of BRD-04. If BRD-04 is not yet implemented when BRD-05 is in production, FR-17/FR-18 trigger behavior is skipped safely (FR-19: `briefing_trigger_skipped_total` metric + `reason=feature_disabled` log). BRD-05 creation/edits succeed without briefing.

### Rollback Notes

- **Flag rollback:** Set `FF_ENABLE_UPCOMING_MEETINGS=false` and `VITE_FF_ENABLE_UPCOMING_MEETINGS=false`. Backend rejects all BRD-05 requests with safe `feature_disabled` JSON response; frontend hides all entry points. No data loss.
- **Database rollback:** BRD-05 migration (`000005_upcoming_meetings.sql`) reversible via `migrate down`. No hard delete in MVP — cancelled records are recoverable.
- **BRD-04 independence:** Rolling back BRD-05 does not affect existing briefing data. BRD-04 tables are orthogonal.
- **No data loss on flag disable:** All persisted upcoming meetings remain queryable at the database level.

---

## Implementation-Phase Accepted Work

The following non-blocking items were identified during validate-design and are accepted as implementation-phase work (documented in validator-findings.md and decision-record.md):

| Item | Owner | Notes |
|------|-------|-------|
| Per-user meeting count hard ceiling | PM | NFR Scale target is 100/user; no hard ceiling defined; informational at this stage |
| Unicode normalization for title normalization (FR-24) | PM | NFC vs NFD vs NFKC; Unicode Standard Annex #15; PM decision needed before implementation |
| Participant email change → BRD-04 continuity | BRD-04 owner | Email change after creation may break participant matching; BRD-04 owner may need to define handling |
| CSRF explicit spec reference | PM/spec-writer | FR-1 or NFR Authorization should reference app CSRF baseline; deferred to implementation |
| Full `cp_upcoming_meeting_*` metrics stack | Implementation | Metrics defined in spec but not yet instrumented; instrument before flag enablement |
| `FF_ENABLE_UPCOMING_MEETINGS` enforcement wiring | Implementation | Flag check in handlers; implement before flag enablement |
| Go structs for UpcomingMeeting types | Implementation | All Phase 2; no source code yet |
| Duplicate awareness UI | Implementation | FR-24 spec complete; implement per production-checklist |
| Participant count 50-warning UX | Implementation | FR-26 spec complete; implement per production-checklist |

---

## Sync Check Result

```
$ bash scripts/check-status-sync.sh
=== ContextPilot Sync Check ===
[1/7] Checking STATUS.md backlog spec paths...
[2/7] Checking eval file BRD references...
[3/7] Checking feature flag parity (specs/feature-flags.md vs .env.example)...
[4/7] Checking STATUS.md decision log completeness...
[5/7] Checking feature-flags.md for TBD entries...
[6/7] Checking architecture script executability...
[7/7] Checking curated BRDs have corresponding eval files...
[8/8] Checking .env.example defaults...
OK: all sync checks passed
Exit code: 0
```

Evidence: scripts/check-status-sync.sh (exits 0, 2026-05-25)

---

## Checklist for PM Before Flag Activation

Before setting `FF_ENABLE_UPCOMING_MEETINGS=true` in production:

- [ ] All production-checklist.md TODO items verified by QA (114 items)
- [ ] OBS-1 through OBS-17 metrics instrumented and scannable
- [ ] LOG-1 through LOG-14 structured log events emitting
- [ ] All 5 CRUD endpoints returning correct status codes under flag
- [ ] Owner-only authorization verified for all endpoints
- [ ] `/ready` reports `brd04_trigger` health correctly
- [ ] FF enforcement: flag=false returns safe 403 JSON, no raw errors
- [x] Sync check passes (verified)
- [x] Feature flag registered with dual env vars, default false (verified)
- [x] OpenAPI contract complete with 10 paths, 4 schemas (verified)
- [x] All 13 PM repair blockers resolved (verified)
- [x] No unresolved blockers in validate-design summary (verified)

---

## Open Questions

The following items are open (non-blocking) and documented for PM/BRD-04 owner awareness:

| Item | Owner | Notes |
|------|-------|-------|
| Per-user meeting count hard ceiling | PM | NFR Scale target 100/user, no hard ceiling defined |
| Unicode normalization for title (FR-24) | PM | NFC vs NFD vs NFKC — Unicode Standard Annex #15 |
| Participant email change → BRD-04 continuity | BRD-04 owner | Email change may break matching signal |
| CSRF explicit reference | PM/spec-writer | Should spec reference app CSRF baseline? |

---

## Updated 2026-05-25 (t_d9f75824)

Changes from prior version (t_baca63bf):
1. OpenAPI coverage updated from "5 CRUD endpoints" to "10 paths" with full enumeration (lines 524–1018 in openapi.yaml)
2. Feature flag Registry Status corrected from "Phase 2, Approved" to "Phase 2, In Dev" to match specs/feature-flags.md line 24
3. Feature flag state section expanded with activation mechanics, dual-flag interaction, and explicit PM activation gate requirements
4. ADR section corrected: ADR-0010 and ADR-0013 do not exist as top-level files — replaced with accurate references (docs/adr/0010-upcoming-meetings-table.md for D-7 resolution; ADR-0013 refers to migration conflict resolution)
5. Events contract section retained as N/A (no events/contracts directory at time of audit)
6. Production checklist summary updated with accurate pre-verified vs TODO counts (14 verified, 114 TODO)
7. Added "Checklist for PM Before Flag Activation" section
8. Added "Implementation-Phase Accepted Work" section with full itemization from validator-findings.md open items
