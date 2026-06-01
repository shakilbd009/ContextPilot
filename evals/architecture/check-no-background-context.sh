#!/bin/bash
# check-no-background-context.sh — no context.Background() in production Go code
set -euo pipefail

SOURCE_DIR="backend"

# Exit 0 if no Go source exists yet (Phase 0 safe)
if [ ! -d "$SOURCE_DIR" ] || [ -z "$(find "$SOURCE_DIR" -name '*.go' -type f 2>/dev/null)" ]; then
  echo "SKIP: no Go source to scan (Phase 0)"
  exit 0
fi

violations=0

while IFS= read -r file; do
  # Skip test files
  if echo "$file" | grep -q '_test\.go$'; then
    continue
  fi

  # Find context.Background() occurrences with line numbers
  while IFS= read -r line; do
    line_num=$(echo "$line" | cut -d: -f1)
    content=$(echo "$line" | cut -d: -f2-)

    # Check for ARCH_OK escape hatch on the line immediately above
    if [ "$line_num" -gt 1 ]; then
      prev_line=$((line_num - 1))
      prev_content=$(sed -n "${prev_line}p" "$file" 2>/dev/null || echo "")
      if echo "$prev_content" | grep -q "ARCH_OK"; then
        continue
      fi
    fi

    echo "VIOLATION: $file:${line_num}: context.Background() found — use request context or context.WithTimeout"
    violations=$((violations + 1))
  done < <(grep -nE 'context\.Background\(\)' "$file" 2>/dev/null) || true

done < <(find "$SOURCE_DIR" -name '*.go' -type f 2>/dev/null) || true

if [ "$violations" -gt 0 ]; then
  echo "FAIL: $violations context.Background() violation(s) found in production code"
  exit 1
fi

echo "OK: no context.Background() in production code"
exit 0