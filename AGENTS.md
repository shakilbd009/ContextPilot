# ContextPilot — Agent Operating Rules

> Version 1.0.0 · Phase 0 Governance

---

## What This Project Is

ContextPilot is a meeting intelligence system that helps users prepare for meetings by surfacing relevant context from previous conversations. It is spec-driven, eval-driven, and feature-flagged. No source code ships without a spec, an eval, and a passing fitness check.

---

## Core Methodology

### Spec-Driven Development

Every feature begins with a BRD (Business Requirements Document) in `specs/<domain>/brd-XX-*.md`. The BRD defines:
- What the feature does and why it matters
- User stories and acceptance criteria
- Observability expectations
- Feature flag name (`ff_enable_*`)
- Constraints and non-goals

No feature flag in code may exist without a corresponding BRD entry in `specs/feature-flags.md`.

### Eval-Driven Development

Every feature has an eval contract in `evals/<type>/`. E2E scenarios are marked 🔴 **Failing** before implementation and 🟢 **Passing** after. Architecture fitness functions run on every push.

Three mandatory architecture fitness functions:
1. `evals/architecture/check-feature-flags.sh` — every `ff_*` in code is registered
2. `evals/architecture/check-no-panic.sh` — no `panic()` in production code
3. `evals/architecture/check-no-background-context.sh` — no `context.Background()` in production code

All three must exit 0 when source folders are empty (Phase 0 safe).

### Feature Flags

Every feature is gated behind an `ff_*` flag:
- Backend/server: `FF_ENABLE_<FEATURE>=true|false` in environment
- Browser/SvelteKit: `VITE_FF_ENABLE_<FEATURE>=true|false` in environment
- Defaults are always `false`
- Local demo override: `.env.local` or `FF_ENABLE_<FEATURE>=true make dev`
- Feature flags are registered in `specs/feature-flags.md` with lifecycle: Planned → In Dev → Active → Deprecated → Removed

---

## Agent Profiles

| Profile | Responsibility |
|---------|---------------|
| `pm` | Product decomposition, BRD scoping, prioritization |
| `ops` | Governance scaffold, CI/CD, infra, tooling |
| `backend` | Go server, DB migrations, API handlers |
| `backend-reviewer` | Senior backend code review for logic, TDD, DRY/SOLID, backend Core Rules compliance, security, observability, and production readiness |
| `frontend-eng` | SvelteKit UI, component library, browser integration |
| `validator` | Review BRD completeness, eval coverage, flag parity |
| `spec-writer` | Author BRDs and eval contracts |
| `architect` | Technical design, ADR decisions, non-functional requirements |
| `done-auditor` | Definition-of-done reality check before flag enablement, BRD/phase/release completion, recovery sprint closure, or starting the next phase after a long agent-built cycle |

---

## Operating Rules

### General

1. **Read before writing.** Load `project-scaffold` skill before Phase 0. Load relevant BRD before implementing a feature.
2. **Verify everything.** Package versions, CLI commands, API conventions — confirm they exist before using them. Never guess.
3. **No source code in Phase 0.** Backend/ and frontend/ stay empty. Governance, specs, contracts, evals, scripts, CI, and docs only.
4. **No placeholder metadata.** STATUS.md, feature-flags.md, and env vars must have real paths, real flag names, real defaults.
5. **Sync check must pass before commit.** Run `scripts/check-status-sync.sh` locally; CI runs it on every push.
6. **PR workflow only.** No direct pushes to `main`. Branch names include BRD ID (`brd-01/app-shell`). PR summaries include eval evidence, screenshots for UI, flag status, migration/security/observability notes, and rollback plan.
7. **Backend code review gate.** Every backend implementation task must be reviewed by `backend-reviewer` before QA starts. The reviewer checks production code quality: business logic, TDD evidence, test quality, DRY/SOLID fit, backend Core Rules compliance, security, error handling, context propagation, feature flags, observability, and maintainability. If `backend-reviewer` returns `REQUEST_CHANGES` or `BLOCK`, create/link repair work for `backend`; QA must wait until blocking findings are resolved or explicitly deferred by PM/product.
8. **QA review after implementation and code review.** Every implementation task (backend, frontend-eng) must be followed by two separate QA verification tasks — never combined into one task:
   - **QA: test coverage (backend)** → qa checks `go test ./...`; no stubs without assertions
   - **QA: test coverage (frontend)** → qa checks `pnpm test`; no stubs without assertions
   - **QA: BRD compliance (backend)** → qa checks acceptance criteria met, E2E scenarios pass
   - **QA: BRD compliance (frontend)** → qa checks acceptance criteria met, `pnpm test:e2e` (playwright) scenarios pass
   QA tasks are linked as children of the implementation task and cannot start until implementation and required code-review gates are done. QA findings must be resolved before the feature is considered complete.
9. **Definition-of-done audit gate.** After QA and post-QA ops readiness pass, create a `done-auditor` task that loads `definition-of-done-audit` and verifies source control, tests/build/CI, runtime when applicable, docs/status files, feature flags, security/observability expectations, and Kanban state. Do not enable feature flags, mark a BRD/phase/release complete, close a recovery sprint, or start the next phase after a long agent-built cycle unless the audit returns `Trustworthy` or `Mostly trustworthy` with no P0/P1 blockers.

### Shell / macOS Safety

- Use `grep -oE` with POSIX character classes — never `grep -oP`
- Use `[a-zA-Z0-9_]` instead of `\w` in grep/grep-E
- Use `[[:space:]]` instead of `\s`
- Append `|| true` to `while ... read` loops over pipes to prevent `set -e` pipe-break fatal exits
- Use shell builtins for path extraction instead of `xargs`
- Port conflicts are WARNs, not ERRORs — existing services on host ports are expected

### Docker

- No `version` key in `docker-compose.yml`
- Healthchecks on every active service
- Named volumes for persistence
- Phase 0: active services must not reference non-existent frontend/backend build contexts
- Phase 1 frontend: anonymous volume for `node_modules/` so host bind mounts don't clobber container deps

### Feature Flag Dual Namespace

When a feature has both server and browser behavior:
```
FF_ENABLE_APP_SHELL=false          # server evaluation
VITE_FF_ENABLE_APP_SHELL=false      # browser build-time embedding
```
Both namespaces must stay in sync. The `.env.example` and `.env` files carry both.

---

## Tooling Versions (Verified)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26.3 | Backend |
| Node | 22 | Frontend |
| pnpm | 10.32.1 | Package manager (pinned in CI) |
| Docker | latest | Container runtime |
| SvelteKit | latest | Frontend framework |

> Version pinning is confirmed via local verification or official docs. Do not invent versions.

---

## Directory Structure

```
.
├── AGENTS.md                      # This file
├── STATUS.md                      # Working memory — backlog, decisions, progress
├── README.md                      # Project overview, local dev, troubleshooting
├── Makefile                       # Orchestration targets
├── .gitignore                     # Standard exclusions
├── .env.example                   # All FF_ENABLE_* vars, defaults false
├── .github/
│   └── workflows/
│       └── eval.yml               # CI pipeline
├── backend/                       # Empty in Phase 0 (Go module in Phase 1)
├── frontend/                      # Empty in Phase 0 (SvelteKit scaffold in Phase 1)
├── contracts/
│   ├── openapi.yaml               # Metadata + common schemas only (no invented endpoints)
│   └── events.md                  # Async/event contract placeholder
├── docs/
│   ├── adr/
│   │   └── 0001-record-architecture-decisions.md
│   ├── security-baseline.md
│   └── observability.md
├── evals/
│   ├── CONVENTIONS.md
│   ├── architecture/
│   │   ├── check-feature-flags.md + .sh
│   │   ├── check-no-panic.md + .sh
│   │   └── check-no-background-context.md + .sh
│   ├── e2e/                      # Placeholder scenarios (🔴 Failing)
│   ├── unit/
│   ├── integration/
│   ├── security/
│   └── perf/
├── scripts/
│   ├── init-eval.sh
│   ├── check-status-sync.sh
│   ├── doctor.sh
│   └── seed-dev.sh
└── specs/
    ├── _template.md
    ├── feature-flags.md
    └── ui/
        └── brd-01-app-shell.md
```

---

## Quick Reference

```bash
# Verify governance
make sync-check       # runs check-status-sync.sh, must exit 0
make eval-arch       # runs all architecture evals (skip in Phase 0 if no source)

# Local dev
make dev             # docker compose up + service URLs
make doctor          # check Docker, CLIs, package manager, ports

# Phase 0 infra only
docker compose up -d
docker compose config  # must pass

# Feature flag demo
FF_ENABLE_APP_SHELL=true make dev   # enable app shell locally
```

---

## Decision Log Quick Index

| ID | Decision | Status |
|----|----------|--------|
| D-001 | Stack selection: Go + SvelteKit | Decided |
| D-002 | Package manager: pnpm | Decided |
| D-003 | Phase 0 scope: governance only | Decided |
| D-004 | First BRD: App Shell (UI-first) | Decided |

Full decisions in `STATUS.md` → Decision Log section.