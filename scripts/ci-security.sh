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

# Resolve GOPATH/bin so we can both install and invoke gosec without
# requiring the caller to manually extend PATH. This is the same
# directory `go install` writes to, so we don't depend on a previous
# `go env GOPATH` being on PATH.
GOBIN_DIR="$(go env GOPATH)/bin"
export PATH="$GOBIN_DIR:$PATH"

# Install gosec if missing
if ! command -v gosec >/dev/null 2>&1; then
  echo "Installing gosec into $GOBIN_DIR..."
  go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

echo "Running gosec (severity=low, confidence=low)..."
cd backend
gosec -severity=low -confidence=low ./...
