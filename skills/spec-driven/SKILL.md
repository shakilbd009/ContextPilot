---
name: spec-driven
description: Spec-driven development cycle: read STATUS.md → locate BRD → audit existing implementation → confirm with user → write failing eval → TDD → run evals → update STATUS.md. Use when building features from a BRD.
---

# Spec-Driven Development

## Overview

Use this skill when you have a Business Requirements Document (BRD) and need to implement a feature against it. The cycle enforces discipline: audit before coding, failing eval before implementation, STATUS.md as the single source of truth.

## Full Cycle

### Step 0 — Pre-flight
- Verify you are authenticated: `gh auth status` or equivalent
- Confirm the kanban board is accessible: `hermes kanban list`
- Confirm you are in the correct project directory

### Step 1 — Read STATUS.md, locate the BRD
1. Read `STATUS.md` in the project root — it lists all features, their phase, and implementation status
2. Read `specs/feature-flags.md` — feature flag registry with lifecycle stages
3. Find the BRD: `specs/<domain>/brd-XX-*.md` (the spec file for this feature)
4. Read the BRD fully before touching any code

### Step 2 — Audit Existing Implementation
Inspect the codebase BEFORE writing any code. Report what already exists vs. what is missing.

Audit these in order:
- **Source folders**: `backend/` and `frontend/` — check for relevant packages/modules
- **Routes**: `frontend/src/routes/` — which routes exist, which are gated by feature flags
- **Migrations**: `backend/migrations/` — schema changes already applied
- **Handler / Service / Repository / Model files**: trace the full call chain for this feature
- **API contracts**: `backend/internal/api/` — REST or gRPC definitions
- **Eval files**: `evals/e2e/` — which evals already exist for this feature

Produce an audit report:
- ✅ **Already done**: files that match the BRD spec
- � partial **Partial**: files that need modification
- ❌ **Missing**: files that need to be created
- ⚠️ **Contract violations**: where the code diverges from the BRD

### Step 3 — Confirm with User
Show the audit report. Wait for confirmation before proceeding. Specifically confirm:
1. The plan is correct (existing code reuse, new file locations)
2. The implementation approach is acceptable
3. Any partial work identified will be handled

### Step 4 — Write Failing Eval First
Before writing any implementation code, write an eval that fails.

Why: the eval encodes why the behavior matters and documents what fails now. It serves as a regression contract.

Location: `evals/e2e/<domain>/brd-XX-*.md`

The eval must:
- Describe the desired behavior (from BRD acceptance criteria)
- Name the concrete failure that exists today
- Assert the success criteria in a way that can be run and measured

### Step 5 — TDD Implementation
RED-GREEN-REFACTOR cycle:
1. **RED**: Write the minimum implementation that makes the eval compile/run (even if it fails)
2. **GREEN**: Implement just enough to pass the failing eval
3. **REFACTOR**: Clean up without breaking the eval

Rules during TDD:
- Never skip from RED to a fully-implemented feature
- Run the eval after each micro-step to stay on track
- Keep BRD acceptance criteria in focus — do not add scope

### Step 6 — Run Eval Suite
```bash
cd evals/integration && go test -v ./...
```
Or the equivalent for your eval runner.

All evals must pass. If a different eval breaks, investigate and fix or quarantine with a note.

### Step 7 — Update STATUS.md
Mark the feature as `done` (or the appropriate phase). Record:
- Implementation decisions made during the audit
- Trade-offs accepted
- Any partial work deferred to a follow-up task

## Eval-Author / Implementer Separation Rule

**The person who writes the failing eval should not be the primary implementer of the feature.** The eval encodes what "good" looks like — an honest outsider is more likely to catch edge cases and missing requirements. If you must serve both roles, at minimum:
- Write the complete eval before touching any implementation code
- Treat the eval as a fixed contract, not a sketch

## File Naming Conventions

| Artifact | Pattern |
|---|---|
| BRD | `specs/<domain>/brd-XX-short-name.md` |
| Eval | `evals/e2e/<domain>/brd-XX-short-name.md` |
| Feature flag | `FF_ENABLE_<FEATURE_NAME>` in `specs/feature-flags.md` and `.env.example` |
| Backend handler | `backend/internal/<domain>/handler.go` |
| Backend service | `backend/internal/<domain>/service.go` |
| Backend repository | `backend/internal/<domain>/repository.go` |
| Backend model | `backend/internal/<domain>/models.go` |
| Frontend route | `frontend/src/routes/<domain>/+page.svelte` |

## Test Structure

Evals live in `evals/integration/` (Go) or `evals/e2e/` (markdown). Each BRD gets its own eval file. Evals are NOT unit tests — they test the full system behavior described in the BRD acceptance criteria.

Unit tests for handler/service/repository live alongside the code: `*_test.go` files in the same package.

## Common Pitfalls

1. **Skip the audit** — implementing to the BRD without checking what already exists duplicates work and creates contract violations
2. **Write implementation before the failing eval** — this skips the most important discipline; without the failing eval, you have no regression contract
3. **Skip user confirmation** — the audit report is the shared contract; proceeding without it risks building the wrong thing
4. **Update STATUS.md at the end of the project, not continuously** — STATUS.md is the kanban board's source of truth; it must be kept current at every phase boundary
5. **Forgetting the feature flag** — new features must be gated; if `FF_ENABLE_*` is missing or set to `false` in `.env`, the feature will not activate in most environments