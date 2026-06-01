# Eval Conventions

> How to write evals for ContextPilot.

---

## Types

| Type | Directory | When it runs |
|------|-----------|--------------|
| Architecture | `evals/architecture/` | Every push (pre-commit) |
| E2E | `evals/e2e/` | Phase 1+, services up |
| Unit | `evals/unit/` | Every PR |
| Integration | `evals/integration/` | Phase 1+, services up |
| Security | `evals/security/` | Weekly + on PR |
| Performance | `evals/perf/` | Pre-release |

---

## Naming

- Architecture: `check-<rule>.md` + `check-<rule>.sh`
- E2E: `brd-<XX>-<slug>.md`
- Unit: `brd-<XX>-<slug>.md`
- Integration: `brd-<XX>-<slug>.md`

---

## Status Markers

E2E and scenario eval files use these markers:

| Marker | Meaning |
|--------|---------|
| 🔴 **Failing** | Scenario defined but not yet implemented. Must be red before a feature ships. |
| 🟡 **Known Failure** | Bug filed, acceptable to ship with bug tracked |
| 🟢 **Passing** | Scenario passes in current code |

All `evals/e2e/*.md` scenarios start as 🔴 **Failing**. Do not write a scenario as 🟢 **Passing** unless it actually passes against implemented code.

---

## Architecture Fitness Functions

Three mandatory fitness functions:

1. **check-feature-flags** — every `ff_*` in code must be registered in `specs/feature-flags.md`
2. **check-no-panic** — no `panic()` in production code (allows `// ARCH_OK:` escape hatch)
3. **check-no-background-context** — no bare `context.Background()` in production code (allows `// ARCH_OK:` escape hatch)

Each fitness function has:
- A `.md` file explaining why the rule exists
- A `.sh` file enforcing it (exits 0 when source dirs are empty)

All three must:
- Exit 0 when `backend/` and `frontend/` are empty (Phase 0 safe)
- Be safe under `set -euo pipefail`
- Support `// ARCH_OK:` escape hatch per file
- Print `file:line:rule:remediation` for each violation

---

## Running Evals

```bash
# Architecture (pre-commit)
make eval-arch

# All evals
make eval

# E2E (requires services up)
make eval-e2e

# Sync check
make sync-check
```

---

## Writing an E2E Scenario

```markdown
## Scenario:riefing generation

### Given
- User has 2 completed meetings in the system
- The meetings share the same attendees

### When
User navigates to `/meetings/{id}/briefing`

### Then
- Briefing shows a summary of previous meeting
- Action items from previous meeting are listed
- Decisions are surfaced
- Open questions are carried forward
```

Mark as 🔴 **Failing** until implemented and verified.

---

## Escape Hatch

Files may opt out of a fitness function by adding a comment on the first line:

```go
// ARCH_OK: test fixture requires panic
func TestPanicHelper() {
    panic("test helper")
}
```

The escape hatch comment must be on the line immediately above the violation, or on the same line before the violation text.

---

## CI Integration

See `.github/workflows/eval.yml`. Architecture and sync-check jobs run on every push. E2E and integration jobs are gated with `if: false` until Phase 1 services exist.