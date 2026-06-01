# check-feature-flags

> Fitness function: every `ff_*` identifier in source code must be registered in `specs/feature-flags.md`.

## Why This Rule Exists

Feature flags are the contracts between PM, engineering, and QA. An unregistered flag means:
- PM did not approve it
- QA did not define acceptance criteria for it
- It cannot be disabled in production if it causes issues

Allowing ad-hoc flags to exist in code is the path to configuration sprawl and uncontrolled features.

## What This Checks

```bash
# Scans backend/ and frontend/ for ff_* identifiers (excludes node_modules)
grep -rE 'ff_[a-zA-Z0-9_]+' backend/ frontend/ \
  --include='*.go' --include='*.ts' --include='*.svelte' \
  --exclude-dir=node_modules
```

For every match:
1. Extract the flag name (e.g., `ff_enable_meeting_import`)
2. Check if it exists in `specs/feature-flags.md` under a feature flag table row
3. Flag as a violation if not registered

## Grace Period

Flags may exist in code before being registered only if:
1. The flag name follows the pattern `ff_enable_<feature>` (well-formed)
2. A BRD exists in draft form at `specs/<domain>/brd-XX-<slug>.md`
3. The flag is added to `specs/feature-flags.md` within the same sprint as first use

## Escape Hatch

Files may opt out by adding `// ARCH_OK:` on the line above the flag reference.

Example:
```go
// ARCH_OK: test fixture uses unregistered flag
fixture := NewTestHarness("ff_enable_unregistered_flag")
```

## Phase 0 Behavior

Exits 0 when `backend/` and `frontend/` are empty (no source to scan).