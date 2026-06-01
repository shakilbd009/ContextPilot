#!/bin/bash
# check-feature-flags.sh — every ff_* in source must be registered in specs/feature-flags.md
set -euo pipefail

REGISTRY="specs/feature-flags.md"
SOURCE_DIRS="backend frontend"

# Exit 0 if no source exists yet (Phase 0 safe)
has_source=false
for dir in $SOURCE_DIRS; do
  if [ -d "$dir" ] && [ -n "$(find "$dir" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.svelte' \) 2>/dev/null)" ]; then
    has_source=true
    break
  fi
done

if [ "$has_source" = false ]; then
  echo "SKIP: no source to scan (Phase 0)"
  exit 0
fi

# Build set of registered flags from the feature-flags.md registry
# Only match table data rows (lines that start with | and contain ff_enable_)
# Skip header rows (no ff_enable_) and separator rows (only dashes/bars/pipes/spaces)
registered_flags=$(grep -E '^\|[^|]*ff_enable_[a-z_]+' "$REGISTRY" 2>/dev/null | grep -oE '`ff_enable_[a-z_]+`' | tr -d '`' | sort -u)

if [ -z "$registered_flags" ]; then
  echo "SKIP: no registered flags found in $REGISTRY"
  exit 0
fi

violations=0

# Scan source files for ff_* identifiers
for dir in $SOURCE_DIRS; do
  if [ ! -d "$dir" ]; then
    continue
  fi
  while IFS= read -r file; do
    # Extract all ff_* identifiers from the file
    # Use grep -oE for POSIX compatibility on macOS
    flags_found=$(grep -oE 'ff_[a-zA-Z0-9_]+' "$file" 2>/dev/null | sort -u)

    for flag in $flags_found; do
      # Check if this flag is registered (case-insensitive match)
      if ! echo "$registered_flags" | grep -iFx "$flag" >/dev/null 2>&1; then
        # Check for ARCH_OK escape hatch on the previous line
        line_num=$(grep -n "$flag" "$file" | head -1 | cut -d: -f1)
        if [ -n "$line_num" ] && [ "$line_num" -gt 1 ]; then
          prev_line=$((line_num - 1))
          if grep -q "ARCH_OK" "$file" 2>/dev/null | head -1 | grep -q "ARCH_OK"; then
            # escape hatch present
            continue
          fi
        fi

        echo "VIOLATION: $file: unregistered flag '$flag' (not in $REGISTRY)"
        violations=$((violations + 1))
      fi
    done
  done < <(find "$dir" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.svelte' \) -not -path '*/node_modules/*' 2>/dev/null) || true
done

if [ "$violations" -gt 0 ]; then
  echo "FAIL: $violations unregistered feature flag(s) found"
  exit 1
fi

echo "OK: all feature flags are registered"
exit 0