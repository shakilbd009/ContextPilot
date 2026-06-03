#!/bin/bash
# check-no-fmt-printf.sh — no fmt.Printf/Println/Print in handler production code
# CWE-209: prevent debug strings and error detail leaks from reaching stderr
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

  # Only scan handler files under internal/*/handler*.go
  if ! echo "$file" | grep -qE 'internal/.+/handler.*\.go$'; then
    continue
  fi

  # Find fmt.Print* / fmt.Println / fmt.Print occurrences with line numbers
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

    echo "VIOLATION: $file:${line_num}: fmt.Print* found — use zerolog instead"
    violations=$((violations + 1))
  done < <(grep -nE 'fmt\.(Printf|Println|Print)\(' "$file" 2>/dev/null) || true

done < <(find "$SOURCE_DIR" -name '*.go' -type f 2>/dev/null) || true

if [ "$violations" -gt 0 ]; then
  echo "FAIL: $violations fmt.Print* violation(s) found in handler production code"
  exit 1
fi

echo "OK: no fmt.Printf/Println/Print in handler production code"
exit 0