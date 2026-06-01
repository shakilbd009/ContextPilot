# BRD-04 Pre-Call Briefing — Implementation Readiness

---
brd_id: brd-04
title: Pre-Call Briefing Implementation Readiness
curated: 2026-05-23
status: Pending Implementation Readiness Assessment

---

## Implementation Readiness Status

Implementation readiness assessment has not yet been run for the BRD-04 Pre-Call Briefing graduation package.

This file will be updated with readiness findings once the implementation readiness evaluation has been executed, including eval contract evidence, production readiness prerequisites, risks, and rollback notes.

---

## Known Inputs and Risks from Raw BRD

The following inputs and risks are documented in the raw BRD and will inform the readiness assessment once it is run:

### Dependencies (Hard)

- **BRD-02 Manual Meeting Import:** Source meeting data foundation required for briefing source selection. BRD-02 must be implemented and functional before BRD-04 can be fully verified.
- **BRD-03 Meeting Memory Processing:** Source memory, evidence statuses, conflict statuses, briefing-readiness signals, and source memory versions required for briefing generation. BRD-03 must be implemented and memory processing must be operational.
- **Feature flag registration:** `FF_ENABLE_PRE_CALL_BRIEFING` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING` must be registered in `specs/feature-flags.md` before implementation begins.
- **OpenAPI contract:** Briefing API endpoints must be defined in `contracts/openapi.yaml` before backend implementation begins.

### Upcoming Meeting Data Model

BRD-04 MVP assumes an `upcoming_meetings` table exists and that users can create upcoming meetings. The raw BRD's upcoming meeting creation UX (FR-6) is handled via async job queue on upcoming meeting creation, but the upcoming meeting data model itself (existence of `upcoming_meetings` table, authorization, etc.) is assumed to already exist or be provided by a separate upcoming-meeting-creation feature. If this assumption is not yet implemented, the upcoming meeting data model is a prerequisite.

### Async Job Queue

BRD-04 briefing generation requires an async job queue mechanism. The raw BRD specifies async generation jobs with retry behavior, failure handling, and observability. The job queue architecture is assumed to follow the pattern established in BRD-03 (memory processing queue via `memory_processing_jobs`). If a new queue mechanism is needed for briefing jobs, this is an architectural prerequisite.

### Observability Requirements

The raw BRD specifies 20+ metrics and 15+ log events. The implementation readiness assessment will need to verify that the observability contract is implementable given the current logging infrastructure.

### Feature Flag Dual Namespace

Both `FF_ENABLE_PRE_CALL_BRIEFING` (server) and `VITE_FF_ENABLE_PRE_CALL_BRIEFING` (browser) must be registered together and kept in sync. Flag parity evals must pass. If the dual-namespace flag infrastructure is not yet in place from prior BRDs, this is a prerequisite.

### Retention Policy

Briefing versions inherit retention/deletion behavior from the associated upcoming meeting until BRD-06 defines final privacy and retention controls. No explicit retention policy is specified in BRD-04.

### Known Risks from Raw BRD

- Automatic matching may select irrelevant prior meetings (mitigated by two-signal threshold, relatedness explanations, and source exclusions)
- Briefing may become too long or hard to scan (mitigated by progressive format)
- Weak-evidence content may reduce trust (mitigated by caveat policy and evidence exclusion rules)
- Async generation may finish too late for pre-call use (mitigated by non-blocking queue and cached view)
- Generation failure may block meeting preparation (mitigated by cached/latest available and retry option)
- Observability may leak meeting content (mitigated by strict redaction rules)
- Feature flag mismatch may expose partial functionality (mitigated by server-authoritative enforcement)

---

## Rollback Plan

If BRD-04 implementation must be rolled back:

1. Disable `FF_ENABLE_PRE_CALL_BRIEFING` and `VITE_FF_ENABLE_PRE_CALL_BRIEFING` feature flags
2. Reject all briefing API requests with safe feature-disabled response
3. Existing briefing versions remain in the database but are inaccessible (no active feature flag)
4. Source exclusions (`briefing_source_exclusions` rows) remain but have no effect without active briefing generation
5. No user data loss beyond briefing versions themselves (source meetings and memory are unaffected)

---

*Readiness assessment will be recorded here after implementation readiness evaluation.*