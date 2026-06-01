---
name: init-feature
description: Quick-start for creating a new feature spec from template, registering it, scaffolding evals, and preparing for implementation — WITHOUT writing any implementation code.
---

# Init Feature

## Overview

Use this skill when a user asks you to create, register, and prepare a new feature for implementation. This skill does **NOT write implementation code** — it only creates the spec, registers the feature flag, and scaffolds the eval scaffold. The implementation itself is done via the `spec-driven` skill.

## Step-by-Step

### Step 1 — Gather Feature Info
Collect from the user:
- **Feature name**: short, kebab-case-friendly (e.g., "manual meeting import")
- **Domain**: which part of the system (e.g., `meetings`, `auth`, `billing`)
- **One-line description**: what it does
- **Acceptance criteria**: what "done" looks like — be specific
- **Priority / phase**: when it should ship

### Step 2 — Detect Conventions
Before writing anything, read:
- `STATUS.md` — to understand the current format and existing feature entries
- `specs/_template.md` — the BRD template (copy from here when creating new specs)
- `specs/feature-flags.md` — to understand the flag registry format
- `scripts/init-eval.sh` — to understand how evals are scaffolded

### Step 3 — Determine BRD ID
Look at existing BRDs in `specs/<domain>/`. The ID is the next sequential number. For example, if `brd-03-meeting-export.md` exists, yours is `brd-04-meeting-import.md`.

### Step 4 — Create Spec from Template
Copy `specs/_template.md` to `specs/<domain>/brd-XX-<feature-name>.md`.

Fill in every section:
- **Background**: why this feature exists
- **Functional Requirements**: concrete, testable behaviors (not UI descriptions)
- **Non-Functional Requirements**: performance, security, availability constraints
- **Acceptance Criteria**: the checklist that proves the feature is done
- **Out of Scope**: explicitly what this feature does NOT do
- **Dependencies**: other features or systems this one requires
- **Open Questions**: anything unresolved that needs a decision

### Step 5 — Register Feature Flag
Add an entry to `specs/feature-flags.md`:

```markdown
| Feature | Flag | Phase | Status | Notes |
|---|---|---|---|---|
| Manual Meeting Import | `FF_ENABLE_MANUAL_MEETING_IMPORT` | 1 | Planned | Import meetings via CSV upload |
```

Use `status: Planned` for newly registered features.

### Step 6 — Add FF_ENABLE_* to .env.example
In both `backend/.env.example` and `frontend/.env.example`, add:

```
# Manual Meeting Import (BRD-04)
FF_ENABLE_MANUAL_MEETING_IMPORT=false
```

Default must be `false` until the feature is verified complete.

### Step 7 — Add Feature Row to STATUS.md Backlog
In the appropriate phase section of `STATUS.md`, add:

```markdown
| BRD-04 | Manual Meeting Import | meetings | Phase 1 | Planned | - |
```

If the feature spans phases, note that in the Notes column.

### Step 8 — Scaffold Evals
Run the eval scaffolding script:
```bash
./scripts/init-eval.sh <domain> <brd-id> <feature-name>
```
For example:
```bash
./scripts/init-eval.sh meetings 04 "manual-meeting-import"
```

This creates `evals/e2e/meetings/brd-04-manual-meeting-import.md` with the eval structure (placeholders for test cases that you will fill in later via `spec-driven`).

### Step 9 — Verify Clean
Run the status sync check:
```bash
./scripts/check-status-sync.sh
```

This verifies that STATUS.md, feature-flags.md, and .env.example are in sync. Fix any discrepancies before reporting completion.

### Step 10 — Report Completion
Report to the user:
1. The BRD file created and its path
2. The feature flag registered and its flag name
3. The eval scaffold created and its path
4. Any open questions from Step 1 that need decisions before implementation
5. That the feature is ready for implementation via `spec-driven`

## IMPORTANT: Implementation Code Is Out of Scope

This skill stops at the spec, flag registration, and eval scaffold. Do NOT write:
- Handler, service, or repository code
- Frontend route or component code
- Database migrations
- API contracts or schema definitions

All of the above is done via the `spec-driven` skill after the user confirms the spec.

## Audit Reminder

Before any coding activity (handler, service, repository, frontend), the `spec-driven` skill requires an audit step. The audit step is non-negotiable — it catches existing work, partial implementations, and contract violations before any code is written.