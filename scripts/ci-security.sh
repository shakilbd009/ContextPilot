#!/usr/bin/env bash
# scripts/ci-security.sh — Local equivalent of the CI `security` job.
# Installs gosec and runs it. TruffleHog is left to CI because the
# official action does a fresh install on every CI run; locally we'd
# either skip the scan or use `go install` for trufflehog too.
#
# Exit non-zero on findings.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

command -v go >/dev/null 2>&1 || { echo "ERROR: 'go' not on PATH" >&2; exit 2; }

# Install gosec if missing
if ! command -v gosec >/dev/null 2>&1; then
  echo "Installing gosec..."
  go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

echo "Running gosec (severity=low, confidence=low)..."
cd backend
gosec -severity=low -confidence=low ./...
