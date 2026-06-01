#!/usr/bin/env bash
# scripts/ci-local-dry-run.sh — Mirror of the blocking CI gates in
# .github/workflows/eval.yml. Run this locally to verify the same checks
# CI runs without pushing to GitHub.
#
# Exit codes:
#   0 — all blocking gates passed
#   1 — at least one blocking gate failed (message printed with --verbose
#       or directly to stderr; this script uses set -euo pipefail)
#   2 — required tool missing (e.g. docker, pnpm, go)
#
# Usage:
#   scripts/ci-local-dry-run.sh         # run every blocking gate
#   scripts/ci-local-dry-run.sh --skip-e2e   # run only non-environment gates
#
# This intentionally does NOT include:
#   - Security (gosec + trufflehog) — these install extra tools and may
#     take minutes. Use `make ci-security` once `gosec` is on PATH.
#     CI runs them as the `security` job.
#   - E2E (Playwright) — requires docker-compose up. Use
#     `scripts/ci-e2e.sh` (or the `make ci-e2e` target) after starting
#     services. CI runs them as the `e2e` job.
#
# Why split? These two gates need real services / extra installs. We
# keep the main `ci-local` fast (< 2 min on a clean tree) and run the
# heavy ones on demand.

set -euo pipefail

# When not running on a TTY (e.g. CI runner, kanban worker, redirected
# output), pnpm refuses to mutate node_modules/ in some flows. Pin CI=1
# so pnpm install --frozen-lockfile and pnpm build behave non-interactively.
if [ -t 0 ] && [ -t 1 ]; then
  : # interactive — leave CI unset
else
  export CI=1
fi

SKIP_E2E=0
for arg in "$@"; do
  case "$arg" in
    --skip-e2e) SKIP_E2E=1 ;;
    -h|--help)
      sed -n '2,28p' "$0"
      exit 0
      ;;
  esac
done

# Resolve project root (script lives in scripts/, project root is ..)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

# Tool preflight
missing=()
command -v go    >/dev/null 2>&1 || missing+=("go")
command -v pnpm  >/dev/null 2>&1 || missing+=("pnpm")
command -v node  >/dev/null 2>&1 || missing+=("node")
if [ "$SKIP_E2E" -eq 0 ]; then
  command -v docker >/dev/null 2>&1 || missing+=("docker")
fi
if [ "${#missing[@]}" -gt 0 ]; then
  echo "ERROR: missing required tools: ${missing[*]}" >&2
  echo "Run 'make doctor' to see installation status." >&2
  exit 2
fi

PASS=0
FAIL=0
run() {
  local name="$1"; shift
  echo ""
  echo "=== $name ==="
  if "$@"; then
    echo "✅ $name"
    PASS=$((PASS + 1))
  else
    echo "❌ $name"
    FAIL=$((FAIL + 1))
  fi
}

# Gate 1: Architecture (mirrors CI: architecture job)
run "architecture: evals/architecture/check-*.sh" \
  bash -c 'for f in evals/architecture/check-*.sh; do [ -f "$f" ] || continue; bash "$f" || exit 1; done'

# Gate 2: Sync check (mirrors CI: sync-check job)
run "sync-check: scripts/check-status-sync.sh" \
  bash scripts/check-status-sync.sh

# Gate 3: Backend (mirrors CI: backend job)
run "backend: go vet"    bash -c 'cd backend && go vet ./...'
run "backend: go test"   bash -c 'cd backend && go test ./...'

# Gate 4: Frontend (mirrors CI: frontend job — typecheck + build + unit tests)
run "frontend: pnpm install" bash -c 'cd frontend && pnpm install --frozen-lockfile'
run "frontend: svelte-check" bash -c 'cd frontend && pnpm exec svelte-check --tsconfig ./tsconfig.json'
run "frontend: pnpm test"    bash -c 'cd frontend && pnpm test'
run "frontend: pnpm build"   bash -c 'cd frontend && pnpm build'

# Gate 5: E2E (mirrors CI: e2e job — only when not skipped)
if [ "$SKIP_E2E" -eq 0 ]; then
  run "e2e: playwright" bash scripts/ci-e2e.sh
else
  echo ""
  echo "=== e2e: SKIPPED (--skip-e2e) ==="
fi

echo ""
echo "==============================================="
echo "ci-local dry-run: ${PASS} passed, ${FAIL} failed"
echo "==============================================="
[ "$FAIL" -eq 0 ]
