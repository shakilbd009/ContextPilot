#!/bin/bash
# check-brd-open-items.sh — block pipeline if any BRD has unresolved open items
# Scans all .md files under specs/ for Open Questions / TBD / Unresolved status markers.
# Exit 0 when all open items resolved. Exit 1 with listing when open items found.
set -euo pipefail

SPECS_DIR="specs"
UNRESOLVED_MARKERS="Open|TBD|Unresolved"
RESOLVED_MARKERS="Resolved|Closed|Done"

# Exit 0 if no spec files exist (Phase 0 safe)
if [ ! -d "$SPECS_DIR" ] || [ -z "$(find "$SPECS_DIR" -name '*.md' -type f 2>/dev/null)" ]; then
  echo "OK: no spec files found"
  exit 0
fi

# Collect all OPEN lines into a temp file (avoids subshell exit-code issues with -e)
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

find "$SPECS_DIR" -name '*.md' -type f 2>/dev/null | sort | while IFS= read -r file; do
  file_basename="${file#$SPECS_DIR/}"

  # shellcheck disable=SC2086
  awk \
    -v file_basename="$file_basename" \
    -v unresolved="$UNRESOLVED_MARKERS" \
    -v resolved="$RESOLVED_MARKERS" \
    '
    BEGIN {
      in_open_section = 0
      in_table = 0
      status_col = 0
    }

    /^##[[:space:]]+(Open|[Oo]pen [Qq]uestions|[Uu]nresolved|TBD)/ {
      in_open_section = 1
      in_table = 0
      status_col = 0
      next
    }

    /^##[[:space:]]/ {
      in_open_section = 0
      in_table = 0
      status_col = 0
      next
    }

    in_open_section && /^\|/ {
      if (!in_table) {
        n = split($0, cells, /\|/)
        for (i = 1; i <= n; i++) {
          cell = cells[i]
          gsub(/^[[:space:]]+|[[:space:]]+$/, "", cell)
          if (tolower(cell) ~ /^(status|state)$/) {
            status_col = i
            in_table = 1
            break
          }
        }
        next
      }

      if (status_col > 0) {
        n = split($0, cells, /\|/)
        if (n >= status_col) {
          status_val = cells[status_col]
          gsub(/^[[:space:]]+|[[:space:]]+$/, "", status_val)

          if (status_val ~ /^[-:]+$/) next
          if (tolower(status_val) ~ resolved) next
          if (status_val ~ /\*(Resolved|Closed|Done)\*/) next
          if (status_val ~ /\[Resolved\]/) next

          if (tolower(status_val) ~ unresolved || status_val ~ /\*(Open|TBD|Unresolved)\*/) {
            question = (n >= 2) ? cells[2] : ""
            gsub(/^[[:space:]]+|[[:space:]]+$/, "", question)
            printf "OPEN: %s:%d: %s\n", file_basename, NR, question
          }
        }
      }
      next
    }

    in_open_section && !/^\|/ {
      in_table = 0
      status_col = 0
    }
  ' "$file"
done >> "$tmp" || true

if [ -s "$tmp" ]; then
  cat "$tmp"
  echo ""
  echo "FAIL: $(wc -l < "$tmp") unresolved open item(s) found in BRDs"
  exit 1
fi

echo "OK: no unresolved open items in BRDs"
exit 0