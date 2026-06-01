#!/usr/bin/env bash
# scripts/ci-e2e.sh — Local equivalent of the CI `e2e` job.
# Assumes docker-compose is available (v1 or v2). Exits non-zero on any
# failure. Mirrors .github/workflows/eval.yml::jobs.e2e.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

# Decide docker-compose flavor
if command -v docker-compose >/dev/null 2>&1; then
  DC=(docker-compose)
elif docker compose version >/dev/null 2>&1; then
  DC=(docker compose)
else
  echo "ERROR: neither 'docker-compose' v1 nor 'docker compose' v2 found" >&2
  exit 2
fi

# Pick host ports that don't conflict with the dev env. CI uses 3000/5174;
# local dev defaults to 8080/5173. Reuse the dev defaults to keep behavior
# close to the demo, but allow override.
export APP_PORT="${APP_PORT:-3000}"
export FRONTEND_PORT="${FRONTEND_PORT:-5174}"

# Install Playwright Chromium (idempotent — pnpm is a no-op if cached)
echo "Installing Playwright chromium..."
(cd frontend && pnpm install --frozen-lockfile)
(cd frontend && pnpm exec playwright install --with-deps chromium)

# Start services
echo "Starting services via: ${DC[*]}"
"${DC[@]}" up -d

cleanup() {
  echo "Stopping services..."
  "${DC[@]}" down || true
}
trap cleanup EXIT

# Wait for backend
echo "Waiting for backend on port ${APP_PORT}..."
for i in $(seq 1 90); do
  if curl -fsS "http://localhost:${APP_PORT}/healthz" >/dev/null 2>&1; then
    echo "  backend healthy after ${i}s"
    break
  fi
  sleep 1
  if [ "$i" -eq 90 ]; then
    echo "  backend did NOT become healthy in 90s" >&2
    "${DC[@]}" ps || true
    exit 1
  fi
done

# Wait for frontend
echo "Waiting for frontend on port ${FRONTEND_PORT}..."
for i in $(seq 1 90); do
  if curl -fsS -o /dev/null "http://localhost:${FRONTEND_PORT}/"; then
    echo "  frontend ready after ${i}s"
    break
  fi
  sleep 1
  if [ "$i" -eq 90 ]; then
    echo "  frontend did NOT become ready in 90s" >&2
    "${DC[@]}" ps || true
    exit 1
  fi
done

# Run Playwright
echo "Running Playwright tests..."
(cd frontend && pnpm exec playwright test --reporter=list)
