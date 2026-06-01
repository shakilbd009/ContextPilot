# check-brd-open-items

> Architecture fitness function: no unresolved open items in BRDs.

## Purpose

Before a feature can advance from design to implementation, all Open Questions, TBDs, and Unresolved flags in its BRD must be answered. This eval enforces that gate in CI — if any open item is still unresolved when a BRD is merged or a feature is promoted, this script blocks the pipeline.

## What It Checks

```
specs/**/*.md  →  looks for ## Open Questions / ## Open Items / ## TBD / ## Unresolved sections
                   parses the status column in each table row
                   exits 0 only when ALL items are Resolved/Closed/Done
```

Markers treated as **unresolved**:

| Marker | Example |
|--------|---------|
| `Open` | `**Open**`, `Open`, `[Open]` |
| `TBD` | `**TBD**`, `TBD`, `[TBD]` |
| `Unresolved` | `**Unresolved**`, `Unresolved` |

Markers treated as **resolved** (ignored):

| Marker | Example |
|--------|---------|
| `Resolved` | `**Resolved**`, `Resolved`, `[Resolved]` |
| `Closed` | `**Closed**`, `Closed` |
| `Done` | `**Done**`, `Done` |

## Output Format

```
# No open items — pipeline proceeds
OK: no unresolved open items in BRDs

# Open items found — pipeline blocked
OPEN: ui/brd-01-app-shell.md:306: OQ-3
OPEN: ui/brd-02-meeting-import.md:42: OQ-7

FAIL: 2 unresolved open item(s) found in BRDs
```

Each `OPEN:` line shows `file:line: question_id` for easy lookup and resolution.

## Phase 0 Behavior

Exits 0 when `specs/` is empty or contains no `.md` files (Phase 0 governance-only projects).

## Escape Hatch

None — open items are a deliberate human decision, not a code defect. Either resolve the item or explicitly leave it open with a documented reason.

## CI Integration

```
# Run standalone
bash evals/architecture/check-brd-open-items.sh

# Run via make (alongside all architecture evals)
make eval-arch

# Run via Makefile target
make check-brd-open-items
```

Add to CI when a BRD reaches "In Review" status:

```yaml
- name: Check BRD open items
  run: bash evals/architecture/check-brd-open-items.sh
```