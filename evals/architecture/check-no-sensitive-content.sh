#!/bin/bash
# check-no-sensitive-content.sh — no hardcoded sensitive meeting-derived content in logs/metrics
# BRD-03: Meeting Memory Processing — Architecture H3
set -euo pipefail

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

violations=0

# Helper: search a file for a pattern and emit a violation if found,
# but skip test files and lines with ARCH_OK escape hatch.
scan_file() {
  local file="$1"
  local pattern="$2"
  local label="$3"

  # Skip test files
  if echo "$file" | grep -qE '_test\.(go|ts|svelte)$'; then
    return 0
  fi

  while IFS= read -r line; do
    line_num=$(echo "$line" | cut -d: -f1)
    content=$(echo "$line" | cut -d: -f2-)

    # ARCH_OK escape hatch: line immediately above or same-line prefix
    if [ "$line_num" -gt 1 ]; then
      prev_line=$((line_num - 1))
      prev_content=$(sed -n "${prev_line}p" "$file" 2>/dev/null || echo "")
      if echo "$prev_content" | grep -q "ARCH_OK"; then
        continue
      fi
    fi
    if echo "$content" | grep -q "ARCH_OK"; then
      continue
    fi

    echo "VIOLATION: $file:${line_num}: ${label}"
    violations=$((violations + 1))
  done < <(grep -nE "$pattern" "$file" 2>/dev/null) || true
}

# ─── Go files ───────────────────────────────────────────────────────────────
if [ -d "backend" ]; then
  while IFS= read -r file; do

    # log.* calls containing sensitive field names in the argument list
    scan_file "$file" 'log\.[a-zA-Z]+\([^)]+(transcript|notes|evidenceSnippet|memory_text|generated_memory|prior_memory|stakeholder_note|conflict_note|participant_pii)[^)]*' \
      "hardcoded sensitive content in log call"

    # metrics calls with sensitive label names or values
    scan_file "$file" 'metrics\.(Counter|Gauge|Histogram|Record|WithLabel|Observe)[^;]*(transcript|notes|evidence|memory_text|participant_pii|stakeholder|conflict_resolution)' \
      "sensitive content in metric label/value"

    # structured log field keys (e.g. log.Info("msg", "transcript", value))
    scan_file "$file" 'log\.[a-zA-Z]+\([^)]*"(transcript|notes|evidence|memory_text|participant_pii|stakeholder_note)":' \
      "sensitive field key in structured log"

    # strings.Contains / ContainsAny on sensitive fields
    scan_file "$file" 'strings\.(Contains|ContainsAny)[^;]*(transcript|notes|evidence)[^;]*Get\(' \
      "accessing transcript/notes/evidence via string search"

    # fmt.Sprintf embedding sensitive field values (not just mentioning field names in messages)
    # Only fires when a transcript/notes/evidence/memory field is actually accessed as a value
    scan_file "$file" 'fmt\.Sprintf[^;]*(transcript|notes|evidenceSnippet|memory_text)\.(Get|Text|Content|String\(\))' \
      "embedding sensitive field value in Sprintf"

    # zap logger with sensitive field keys
    scan_file "$file" 'zap\.(Sugared)?Logger\.[a-z]+\([^)]*"(transcript|notes|evidenceSnippet|memory_text|participant_pii)":' \
      "sensitive field key in zap logger"

    # logrus with sensitive field keys
    scan_file "$file" 'logrus\.[a-z]+\([^)]*"(transcript|notes|evidence|memory_text|participant_pii)":' \
      "sensitive field key in logrus"

    # logging raw json.Marshal output (caller must ensure no sensitive fields)
    scan_file "$file" 'log\.[a-zA-Z]+\([^)]+json\.Marshal[^)]+\)' \
      "json marshaled struct in log (verify no sensitive fields)"

  done < <(find backend -name '*.go' -type f 2>/dev/null) || true
fi

# ─── TypeScript / Svelte files ──────────────────────────────────────────────
if [ -d "frontend" ]; then
  while IFS= read -r file; do

    # console.log/error/warn with sensitive field names
    scan_file "$file" 'console\.(log|error|warn|debug|info)\([^)]*(transcript|notes|evidenceSnippet|memory_text|participantPII|stakeholderNote|conflictResolutionNote)[^)]*' \
      "hardcoded sensitive content in console call"

    # logger with sensitive field keys
    scan_file "$file" 'logger\.[a-z]+\([^)]*"(transcript|notes|evidenceSnippet|memory_text)":' \
      "sensitive field key in logger"

    # analytics / telemetry calls with sensitive field keys
    scan_file "$file" '(track|analytics|metric|report)\([^)]*"(transcript|notes|evidence|memory_text|participant_pii|stakeholder_note)":' \
      "sensitive content in analytics/telemetry call"

    # winston / pino / tracer with sensitive field keys
    scan_file "$file" '(winston|pino| tracer|debug)\.[a-z]+\([^)]*"(transcript|notes|evidenceSnippet|memory_text|participantPII)":' \
      "sensitive field key in JS logger"

    # console.table / console.dir with sensitive objects
    scan_file "$file" 'console\.(table|dir)\([^)]*(transcript|notes|memory_text|evidence)' \
      "sensitive data passed to console.table/dir"

  done < <(find frontend -type f \( -name '*.ts' -o -name '*.svelte' \) 2>/dev/null) || true
fi

if [ "$violations" -gt 0 ]; then
  echo "FAIL: $violations hardcoded sensitive content violation(s) found"
  exit 1
fi

echo "OK: no hardcoded sensitive meeting-derived content in logs/metrics"
exit 0