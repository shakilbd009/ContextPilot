# ContextPilot — Status

> Working memory. Human-edited. Do not generate placeholder entries.

> **State vocabulary (used throughout):** **Implemented** = code exists, **Gates Green** = committed code passes `make eval` on the committed ref, **Production-Ready** = `done-auditor` returned Trustworthy / Mostly trustworthy with no P0/P1 blockers.

---

## Recovery state (2026-06-01)

This section is the **ground truth** for what is and is not yet true on disk. Read it before believing any "Done" badge in the tables below.

**Recovery work delivered (in uncommitted working tree on `ops/restore-ci-baseline`):**
- `t_29c0c4a5` (ops): `make demo-check` now falls back `docker compose` v2 → `docker-compose` v1, exits non-zero on real config failure. Playwright chromium/headless-shell/ffmpeg installed. All 5 ContextPilot containers healthy.
- `t_b684f226` (ops): git repo initialized; CI workflow gates unblocked (`if: false` removed from backend/frontend/e2e/gosec); `make eval` split into blocking vs `make eval-report` non-blocking; `docker-compose` v1 wired into Makefile + `scripts/doctor.sh`. **No GitHub remote — Shakil must create/connect the repo and open a PR.**
- `t_dc615711` (QA): backend recovery verified — `go test ./...` + `go vet ./...` + `make eval-arch` (5/5) green on the working tree.
- `t_5d9f1c92` (backend): three Go test root causes fixed (chi route-on-parent typo, validator clock injection, FF metric test isolation).
- `t_e51a46af` (backend): briefing chi r-vs-g routing typo repaired in the working tree — `briefing/handler.go:141-153` now uses `g.Get` / `g.Post` inside the FF-middleware group, mirroring the pattern from `upcoming/handler.go:134-138`. `TestHandler_FeatureFlagDisabled_AllRoutes` passes all 7 subtests. **Code change is in the working tree but not yet committed.**
- `t_e925e18a` (spec-writer, this task): reconciled docs/status/feature-flag lifecycle. Also fixed a recovery-introduced Makefile bug: `eval-backend` and `eval-frontend` each had two separate `cd backend` (or `cd frontend`) invocations in different subshells, so the second `cd` ran from the directory the first `cd` left behind and failed with `cd: backend: No such file or directory`. Both targets now chain their two commands in a single subshell. **As of 2026-06-01, `make eval`, `make eval-backend`, `make eval-frontend`, `make eval-arch`, and `make sync-check` all exit 0 on the working tree.**
- `t_f6efb43c` (ops, this task): CI workflow hardened to match local gates. (1) Frontend job no longer calls `pnpm lint` (no `lint` script in `package.json`); the `lint` step is now a no-op SKIP that mirrors `make lint` behavior. (2) E2E job installs Node + pnpm + Playwright Chromium with system deps, then waits for backend `/healthz` and frontend root before running tests. (3) Security job installs `gosec` via `go install` and runs it blocking; TruffleHog runs via the official action and also blocks on findings. (4) `scripts/ci-local-dry-run.sh`, `scripts/ci-e2e.sh`, `scripts/ci-security.sh` plus `make ci-local`, `make ci-local-skip-e2e`, `make ci-e2e`, `make ci-security` give a local dry-run equivalent for every blocking CI gate. **Verified:** `make ci-local-skip-e2e` passes 8/8 gates on the working tree (architecture, sync-check, go vet, go test, pnpm install, svelte-check, pnpm test, pnpm build). E2E full-run still reproduces the 14 app-level failures flagged in `t_cf76b921` (out of recovery scope).
- `t_c37b703c` (ops, this task): source-control delivery prepared. Cleaned `.gitignore` (added `backend/server`, `backend/contextpilot-server`, `backend/*.test`, `backend/cover.out`, `frontend/test-results/`, `frontend/playwright-report/`). Patched `scripts/ci-local-dry-run.sh` to export `CI=1` when not on a TTY so `pnpm install --frozen-lockfile` and `pnpm build` behave non-interactively under the kanban worker / CI runner. Did **not** commit two unreferenced speculative files (`frontend/src/test-types.d.ts`, `frontend/tsconfig.test.json`) — they are documented in the commit message as deliberate holds pending a wiring decision. Did **not** commit the deleted root-level binaries (`backend/briefing.test`, `backend/contextpilot-server`, `backend/server`) — they are staged as `D` so the next commit removes them from history. **No remote, no push, no main-touch**; awaiting Shakil's remote-create decision (see review-required handoff comment for exact commands).

**Honest caveats (pre-commit — true at the moment of writing, will be re-stated post-commit):**
- **Recovery code is being committed by `t_c37b703c` to `ops/restore-ci-baseline`.** Prior to that commit, the working tree carries the uncommitted fixes referenced in items 14-20 above. On the committed ref `89f250f`, `go test ./...` and `go vet ./...` fail (briefing package has a build error; `internal/upcoming` has 3 failing tests — the same ones the recovery work was supposed to fix). CI on the committed ref would not pass.
- **No GitHub remote exists.** CI gates exist in `.github/workflows/eval.yml` but cannot execute until Shakil creates a GitHub repo and pushes.
- **E2E:** Playwright browsers are installed for the ops profile; 13 E2E passed and 14 failed in the parent run, all on app-level bugs in meeting-import and upcoming features. These are separate from the recovery scope.
- **Frontend:** `pnpm test` (vitest) passes 25 files / 336 passed / 3 skipped. `pnpm exec svelte-check` passes with 0 errors and 7 warnings (all unused CSS selectors). `pnpm build` succeeds. All three run green inside the `eval-frontend` Makefile target.

**Open recovery follow-ups (in priority order):**
1. Backend recovery code commit (currently uncommitted working tree needs `git add` + `git commit` on `ops/restore-ci-baseline`). The briefing r→g fix is part of this batch — not a separate work item.
2. Frontend recovery code commit (uncommitted working tree).
3. `t_cf76b921` (done-auditor) re-audit after recovery code is committed.
4. t_a43d9d1b flag-mismatch instrumentation (blocked on PM decision).
5. t_e51a46af cleanup fold-ins: delete `debug_test.go` / `ff_debug_test.go`; wire or delete dead `ffOverrideKey` / `withFFOverride` / `getFFOverride` code.
6. Resolve 14 E2E app-level failures (out of recovery scope).
7. No GitHub remote — Shakil must create/connect.

---

## Project Phase

| Phase | Impl | Gates Green | Prod-Ready |
|-------|------|-------------|------------|
| Phase 0 — Governance Scaffold | Yes | Yes (on `89f250f`) | No — pre-feature |
| Phase 1 — Backend + Frontend Skeleton | Yes | **No** — only on working tree, not on a commit | No |
| Phase 2 — BRD Implementation | Mostly (BRD-01..05) | **No** — recovery state caveat | No |

---

## Backlog

| ID | Title | Spec Path | Priority | State |
|----|-------|-----------|----------|-------|
| BRD-01 | App Shell | `specs/ui/brd-01-app-shell.md` | P0 | Implemented (frontend routes/UI built; backend/FF scaffolding in place); not Gates Green (recovery-state caveat) |
| BRD-02 | Manual Meeting Import | `specs/curated/brd-02-manual-meeting-import.md` | P1 | Implemented; not Gates Green |
| BRD-03 | Meeting Memory Processing | `specs/curated/brd-03-meeting-memory-processing.md` | P1 | Implemented; not Gates Green |
| BRD-04 | Pre-Call Briefing | `specs/curated/brd-04-pre-call-briefing/brd.md` | P1 | Implemented; r-vs-g routing typo repaired in working tree (`briefing/handler.go:141-153`); `TestHandler_FeatureFlagDisabled_AllRoutes` passes all 7 subtests; not Gates Green on commit |
| BRD-05 | Manual Upcoming Meeting Creation | `specs/curated/brd-05-manual-upcoming-meeting-creation/brd.md` | P1 | Spec **APPROVED** (2026-05-24 per D-8); code Implemented (chi r-vs-g typo fixed in working tree); not Gates Green on commit |
| BRD-06 | Privacy & Retention Controls | `specs/security/brd-06-privacy-retention-controls.md` | P1 | Spec only |
| BRD-07 | Manual Memory Correction | `specs/domain/brd-07-manual-memory-correction.md` | P2 | Spec only |

**Note on prior "Done" labels:** earlier versions of this table marked BRD-01..BRD-05 as `Done`. That conflates "spec approved" + "code written" with "passes every gate" + "auditor approved for production". This table now uses the three-state vocabulary. The curated BRD packages themselves (in `specs/curated/`) are still the authoritative spec artifacts.

---

## Feature Flags

Source of truth for flag **state** (lifecycle column) is `specs/feature-flags.md`. This table mirrors it and adds a **Gates Green** column. Disagreement = bug; fix the more stale file.

| Flag | Layer | Lifecycle | Gates Green (on commit) | Notes |
|------|-------|-----------|--------------------------|-------|
| `FF_ENABLE_APP_SHELL` | UI | **In Dev** (per `specs/feature-flags.md`; an earlier STATUS.md said "Active" — that was wrong and is fixed here) | No — recovery-state caveat | App shell, layout, routing, design tokens |
| `FF_ENABLE_MANUAL_MEETING_IMPORT` | Domain | **In Dev** (earlier STATUS.md said "Planned" — stale) | No | Manual meeting creation with transcript or notes import |
| `FF_ENABLE_MEETING_MEMORY_PROCESSING` | Domain | **In Implementation** (earlier STATUS.md said "Planned" — stale) | No | Meeting → memory extraction with version history, conflict review queue, evidence-grounding |
| `FF_ENABLE_PRE_CALL_BRIEFING` | Domain | **Planned** (matches `specs/feature-flags.md`) | No — r-vs-g fix is in working tree, not on a commit; ff is not in production yet so the production-bypass impact is latent | Briefing generation |
| `FF_ENABLE_UPCOMING_MEETINGS` | Domain | **Active** (matches `specs/feature-flags.md`) | No — recovery-state caveat; the r-vs-g fix is in working tree, not on a commit | Manual upcoming meeting creation — BRD-05 |
| `FF_ENABLE_PROVIDER_CONNECTORS` | Infra | Planned | n/a | Teams, Meet, Zoom, calendar integrations |
| `FF_ENABLE_PRIVACY_RETENTION_CONTROLS` | Security | Planned | n/a | PII masking, retention policies, data export |
| `FF_ENABLE_MANUAL_MEMORY_CORRECTION` | Domain | Planned | n/a | User corrections to meeting summaries |

Full registry: `specs/feature-flags.md`.

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
- **Superseded by:** D-009 (Phase 0 closed, Phase 1 skeleton shipped, recovery in progress).

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

### D-009: Recovery Sprint — Done-Audit-Driven Reconstruction (2026-05-31 → 2026-06-01)
- **Trigger:** done-auditor task t_7b66e171 returned verdict **Not done, 1/5**. The board was nearly all done but source control was absent, backend tests failed, frontend typecheck failed, E2E could not launch, CI masked/disabled core gates, runtime demo checks returned success despite docker compose failure, and docs/status/feature flags contradicted implementation state.
- **Decision:** Spawn a 9-task recovery graph (ops, backend, backend-reviewer, qa, frontend-eng, qa, ops-runtime, spec-writer, done-auditor re-gate). Each parent verifies the underlying state on disk before the next parent claims success.
- **Rationale:** Definition-of-done is a property of the project, not a label on a card. Claiming "Done" without gates-green is the exact failure mode the recovery is fixing.
- **Outcome as of 2026-06-01:** most recovery work is Implemented in the working tree but **not committed**. The honest state of the project on the latest commit is still red. See "Recovery state" at the top of this file.

---

## Completed Deliverables

| BRD | Tasks | Key Files |
|-----|-------|-----------|
| BRD-01 App Shell | 17 tasks done | `frontend/src/` (9 UI components, 10 routes, tokens in app.css), `backend/` skeleton, `hooks.server.ts` auth guard, `/ready` + `/live` health endpoints. **Caveat:** "Done" here means "task marked done on the board", not "production-ready". |

---

## Active Runs

- `t_e51a46af` (backend) — repair briefing chi r-vs-g routing typo (code change in working tree; awaiting commit as part of the uncommitted recovery batch)
- `t_e925e18a` (spec-writer) — reconcile docs and feature-flag status (this task, in progress)
- The current task dispatched you to do this reconciliation. Other tasks may appear here as the board moves.

---

## Notes for Next Session

- **Read "Recovery state" at the top of this file first.** It is the honest picture of what is and is not yet committed.
- BRD-01 App Shell code is in `frontend/src/` and `backend/`. Dev server runs on `http://localhost:5173/`. Whether it is "done" depends on which of the three states you mean.
- Feature flags are dual-namespace: `FF_ENABLE_*` (server) and `VITE_FF_ENABLE_*` (browser). Both are registered in `.env.example`.
- Next BRDs after recovery: BRD-04 Pre-Call Briefing (r→g fix is in working tree; commit is part of the uncommitted recovery batch), BRD-06 Privacy & Retention Controls (spec only), BRD-07 Manual Memory Correction (spec only).
- When adding BRDs for domain specs, create corresponding eval files in `evals/e2e/`, `evals/unit/`, `evals/integration/` with the BRD number in the filename.
- The impeccable polish task (t_549b9b25) completed but the frontend-eng agent could not load the `impeccable` skill — skill availability for that profile needs investigation before the next polish pass.
- **No GitHub remote is configured.** The `.github/workflows/eval.yml` gates exist but cannot run until Shakil creates/connects a GitHub repo and pushes.
