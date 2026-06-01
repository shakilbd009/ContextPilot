#!/bin/bash
# init-eval.sh — Bootstrap a new eval type for a BRD
set -euo pipefail

usage() {
  echo "Usage: $0 <type> <brd-id>"
  echo "  type: unit | integration | e2e | security | perf"
  echo "  brd-id: brd-01, brd-02, etc."
  echo ""
  echo "Creates:"
  echo "  evals/<type>/<brd-id>-<slug>.md  (eval contract)"
  echo "  evals/<type>/<brd-id>-<slug>.sh  (optional runner)"
  echo ""
  echo "Example:"
  echo "  $0 e2e brd-02"
  exit 1
}

TYPE="${1:-}"
BRD_ID="${2:-}"

if [ -z "$TYPE" ] || [ -z "$BRD_ID" ]; then
  usage
fi

case "$TYPE" in
  unit|integration|e2e|security|perf) ;;
  *)
    echo "ERROR: type must be one of: unit, integration, e2e, security, perf"
    exit 1
    ;;
esac

case "$BRD_ID" in
  brd-[0-9]*) ;;
  *)
    echo "ERROR: brd-id must match brd-00 format"
    exit 1
    ;;
esac

eval_dir="evals/$TYPE"

if [ ! -d "$eval_dir" ]; then
  echo "Creating $eval_dir/"
  mkdir -p "$eval_dir"
fi

# Extract slug from brd-id for filename
slug="${BRD_ID}-$(date +%Y%m%d)"

echo "Created: $eval_dir/README.md (stub)"
echo "Run: ls $eval_dir/ to see all evals for this type"