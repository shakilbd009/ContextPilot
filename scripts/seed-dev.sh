#!/bin/bash
# seed-dev.sh — Load deterministic seed data for local development
set -euo pipefail

# Phase 0: No backend, skip gracefully
if [ ! -f "backend/main.go" ]; then
  echo "SKIP: backend/main.go not found — seed-dev runs after Phase 1 backend is scaffolded"
  echo "      To start infrastructure: docker compose up -d"
  exit 0
fi

echo "Loading seed data..."
echo "(Full implementation in Phase 1)"