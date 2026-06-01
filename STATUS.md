# ContextPilot — Status

> Working memory. Human-edited. Do not generate placeholder entries.

---

## Project Phase

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 0 — Governance Scaffold | ● Done | All tasks complete |
| Phase 1 — Backend + Frontend Skeleton | ● In Progress | App Shell shipped; dev server running |
| Phase 2 — BRD Implementation | ◻ Todo | Pending Phase 1 |

---

## Backlog

| ID | Title | Spec Path | Priority | Status |
|----|-------|-----------|----------|--------|
| BRD-01 | App Shell | `specs/ui/brd-01-app-shell.md` | P0 | Done |
| BRD-02 | Manual Meeting Import | `specs/curated/brd-02-manual-meeting-import.md` | P1 | Done |
| BRD-03 | Meeting Memory Processing | `specs/curated/brd-03-meeting-memory-processing.md` | P1 | Done |
| BRD-04 | Pre-Call Briefing | `specs/curated/brd-04-pre-call-briefing/brd.md` | P1 | Done |
| BRD-05 | Manual Upcoming Meeting Creation | `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` | P1 | Done |

| BRD-06 | Privacy & Retention Controls | `specs/security/brd-06-privacy-retention-controls.md` | P1 | Todo |
| BRD-07 | Manual Memory Correction | `specs/domain/brd-07-manual-memory-correction.md` | P2 | Todo |

---

## Feature Flags

| Flag | Phase | Status |
|------|-------|--------|
| `FF_ENABLE_APP_SHELL` | UI | Active |
| `FF_ENABLE_MANUAL_MEETING_IMPORT` | Domain | Planned |
| `FF_ENABLE_MEETING_MEMORY_PROCESSING` | Domain | Planned |
| `FF_ENABLE_PRE_CALL_BRIEFING` | Domain | Planned |
| `FF_ENABLE_UPCOMING_MEETINGS` | Domain | Active |
| `FF_ENABLE_PROVIDER_CONNECTORS` | Infra | Planned |
| `FF_ENABLE_PRIVACY_RETENTION_CONTROLS` | Security | Planned |
| `FF_ENABLE_MANUAL_MEMORY_CORRECTION` | Domain | Planned |

Full registry: `specs/feature-flags.md`

---

## Decision Log

### D-001: Stack Selection — Go + SvelteKit
- **Decision:** Use Go/Echo for the backend REST API and SvelteKit for the frontend.
- **Rationale:** Go provides strong typing, fast compilation, and excellent concurrency primitives for a meeting intelligence workload. SvelteKit offers SSR, file-based routing, and a component model well-suited to briefing generation. Both are production-proven and have strong ecosystem support.
- **Trade-off:** Go requires more boilerplate than Python/FastAPI for simple CRUD; SvelteKit has a smaller market share than Next.js.
- **Mitigation:** Use gofiber or echo for ergonomic route handlers; lean on SvelteKit's form actions and+page.server.ts loaders for data flow.

### D-002: Package Manager — pnpm
- **Decision:** Use `pnpm` as the frontend package manager with version pinned to 10.32.1 in CI.
- **Rationale:** pnpm has faster install times, better disk utilization, and strict dependency hoisting compared to npm/yarn. Pinning in CI prevents surprising lockfile changes.
- **Trade-off:** pnpm is not as widely known as npm; onboarding may require a brief explanation for contributors unfamiliar with it.
- **Mitigation:** Document the why in AGENTS.md; add `corepack enable` / `enable` to setup steps so pnpm is available without a global install.

### D-003: Phase 0 Scope — Governance Only
- **Decision:** Phase 0 delivers governance, specs, contracts, evals, infra config, scripts, and CI. No source code.
- **Rationale:** Establishing governance before code prevents architectural drift and gives every subsequent agent clear operating rules. BRDs, evals, and feature flags are the contracts between PM, engineering, and QA.
- **Trade-off:** Phase 0 has no running software. Value delivery starts at Phase 1.
- **Mitigation:** Docker Compose Phase 0 infra (PostgreSQL, Redis, Mailpit) is included so the stack is ready to run on Day 1 of Phase 1.

### D-004: First BRD — App Shell (UI-First)
- **Decision:** The first BRD is the App Shell UI — layout, routing, design tokens, core pages, and shared components.
- **Rationale:** Interview/demo-oriented scaffolds benefit from a visual, demonstrable first deliverable. Backend error contracts, auth, and rate limiting are secondary to showing a working UI. App Shell also establishes the feature flag `ff_enable_app_shell` as the gating mechanism for the entire UI layer.
- **Trade-off:** App Shell BRD may feel premature if the team expects backend-first. However, the product spec (contextPilot.md) is user-facing and UI-first by nature.
- **Mitigation:** Backend contracts (RFC 7807, cache/session keyspace, auth) are deferred to BRD-02 and later domain specs.

### D-005: BRD-02 Validation Strategy — Dual-Layer (Client + Server)
- **Decision:** Dual-layer validation for manual meeting import: client-side for immediate UX feedback, server-side as the authoritative gate. Client-side validation does not block submission.
- **Rationale:** Client-side-only is bypassable. Server-only wastes round-trips for obvious errors. Dual-layer provides UX without sacrificing security.
- **ADR:** [ADR-002](docs/adr/0002-manual-meeting-import-validation-strategy.md)

### D-006: BRD-02 Data Model — Normalized Tables (meetings + meeting_participants)
- **Decision:** Separate `meetings` and `meeting_participants` tables with a foreign key. Transcript and notes stored as separate nullable columns. Content source derived at write time.
- **Rationale:** BRD requires structured participant records, separate transcript/notes preservation, and content-source metadata for downstream processing. Normalized tables support future participant queries.
- **ADR:** [ADR-003](docs/adr/0003-manual-meeting-import-data-model.md)

### D-007: BRD-02 Form Content Preservation — SvelteKit use:enhance
- **Decision:** Use SvelteKit form enhancement (`use:enhance`) with server echo of submitted values on validation failure. SessionStorage as fallback for catastrophic failures only.
- **Rationale:** Idiomatic SvelteKit approach, no full-page reload on validation failure, progressive enhancement degrades gracefully without JS.
- **ADR:** [ADR-004](docs/adr/0004-manual-meeting-import-form-content-preservation.md)

### D-008: Backend Code Review Gate
- **Decision:** Add `backend-reviewer` as a mandatory review gate between backend implementation and QA.
- **Rationale:** QA verifies behavior and eval compliance, but production code quality also needs independent review for business logic, TDD evidence, test quality, DRY/SOLID fit, backend Core Rules compliance, security, error handling, context propagation, feature flags, observability, and maintainability.
- **Mitigation:** `REQUEST_CHANGES` or `BLOCK` verdicts create/link backend repair work before QA starts, preventing QA from validating code that is passing tests but not production-quality.
- **Process audit:** [PA-0001](docs/process-audit/0001-brd03-phase2-missed-backend-reviewer-gate.md) — BRD-03 Phase 2 (2026-05-21) shipped without the gate; QA passed, no customer impact found; gate now enforced in AGENTS.md Rule 7 and kanban-orchestrator.

---

## Completed Deliverables

| BRD | Tasks | Key Files |
|-----|-------|-----------|
| BRD-01 App Shell | 17 tasks done | `frontend/src/` (9 UI components, 10 routes, tokens in app.css), `backend/` skeleton, `hooks.server.ts` auth guard, `/ready` + `/live` health endpoints |

---

## Active Runs

No active tasks. All pipelines clear.

---

## Notes for Next Session

- BRD-01 App Shell is complete and deployed to local dev server (`http://localhost:5173/`).
- Feature flags are dual-namespace: `FF_ENABLE_*` (server) and `VITE_FF_ENABLE_*` (browser). Both are registered in `.env.example`.
- Next BRD in queue: BRD-04 Pre-Call Briefing. BRD-02 Manual Meeting Import and BRD-03 Meeting Memory Processing are approved and ready for curation.
- When adding BRDs for domain specs, create corresponding eval files in `evals/e2e/`, `evals/unit/`, `evals/integration/` with the BRD number in the filename.
- The impeccable polish task (t_549b9b25) completed but the frontend-eng agent could not load the `impeccable` skill — skill availability for that profile needs investigation before the next polish pass.
