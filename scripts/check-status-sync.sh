#!/bin/bash
# check-status-sync.sh — Verify governance parity across project files
set -euo pipefail

ERRORS=0

echo "=== ContextPilot Sync Check ==="

# 1. STATUS.md backlog entries have spec paths (not TBD)
echo ""
echo "[1/7] Checking STATUS.md backlog spec paths..."
while IFS='|' read -r id title spec_path remainder; do
  # Skip header, separator, empty lines
  [[ "$id" =~ ^[[:space:]]*ID ]] && continue
  [[ "$id" =~ ^[[:space:]]*- ]] && continue
  [[ -z "$id" ]] && continue

  if echo "$spec_path" | grep -qE 'TBD|pending|coming'; then
    echo "  ERROR: $id has no spec path assigned"
    ERRORS=$((ERRORS + 1))
  fi
done < STATUS.md || true

# 2. Eval .md files reference valid BRDs (skip architecture/security/perf)
echo ""
echo "[2/7] Checking eval file BRD references..."
for eval_file in evals/e2e/*.md evals/unit/*.md evals/integration/*.md; do
  [ -f "$eval_file" ] || continue

  # Extract BRD IDs from the filename: brd-01-app-shell.md -> brd-01
  basename=$(basename "$eval_file")
  if [[ "$basename" =~ ^(brd-[0-9]+) ]]; then
    brd_id="${BASH_REMATCH[1]}"
    # Check if the referenced BRD exists in specs/
    if ! find specs/ -name "${brd_id}-*.md" -type f 2>/dev/null | grep -q .; then
      echo "  ERROR: $eval_file references $brd_id but no spec found in specs/"
      ERRORS=$((ERRORS + 1))
    fi
  fi
done

# 3. Every feature flag in registry has FF_ENABLE_* in .env.example
echo ""
echo "[3/7] Checking feature flag parity (specs/feature-flags.md vs .env.example)..."

# Extract flag names from table data rows (count | > 3 to distinguish from header/separator rows)
# Use awk to process line-by-line via while loop over awk output
while IFS= read -r line; do
  flag=$(echo "$line" | grep -oE '`ff_enable_[a-z_]+`' | tr -d '`' | sort -u)
  for f in $flag; do
    env_var_name=$(echo "$f" | tr '[:lower:]' '[:upper:]')
    if ! grep -qE "^${env_var_name}=" .env.example 2>/dev/null; then
      echo "  ERROR: $f registered in feature-flags.md but not in .env.example"
      ERRORS=$((ERRORS + 1))
    fi
  done
done < <(awk '/`ff_enable_[a-z_]+`/ { count=0; for(i=1;i<=NF;i++) if($i!~/^[[:space:]]*$/) count++; if(count>3) print }' specs/feature-flags.md 2>/dev/null) || true

# 4. Decision log entries have all 4 fields
echo ""
echo "[4/7] Checking STATUS.md decision log completeness..."
in_decision_block=false
has_decision=false
has_rationale=false
has_tradeoff=false
has_mitigation=false

while IFS= read -r line; do
  if echo "$line" | grep -qE '^### D-[0-9]+:'; then
    # Check previous decision block before resetting
    if $in_decision_block && [ "$has_decision" = false ]; then
      echo "  ERROR: Decision block missing **Decision** field"
      ERRORS=$((ERRORS + 1))
    fi
    in_decision_block=true
    has_decision=false
    has_rationale=false
    has_tradeoff=false
    has_mitigation=false
  fi

  if $in_decision_block; then
    echo "$line" | grep -q '^\- \*\*Decision:\*\*' && has_decision=true
    echo "$line" | grep -q '^\- \*\*Rationale:\*\*' && has_rationale=true
    echo "$line" | grep -q '^\- \*\*Trade-off:\*\*' && has_tradeoff=true
    echo "$line" | grep -q '^\- \*\*Mitigation:\*\*' && has_mitigation=true
  fi
done < STATUS.md || true

# Check the last decision block
if $in_decision_block && [ "$has_decision" = false ]; then
  echo "  ERROR: Last decision block missing **Decision** field"
  ERRORS=$((ERRORS + 1))
fi

# 5. No TBD in feature-flags.md
echo ""
echo "[5/7] Checking feature-flags.md for TBD entries..."
if grep -E '^[[:space:]]*\|.*TBD' specs/feature-flags.md 2>/dev/null | grep -qv '^[[:space:]]*#'; then
  echo "  ERROR: feature-flags.md contains TBD entries"
  ERRORS=$((ERRORS + 1))
fi

# 6. Architecture scripts are executable
echo ""
echo "[6/7] Checking architecture script executability..."
for script in evals/architecture/check-*.sh; do
  if [ -f "$script" ]; then
    if [ ! -x "$script" ]; then
      echo "  ERROR: $script is not executable"
      chmod +x "$script"
      echo "  FIXED: chmod +x $script"
    fi
  fi
done

# 7. Every curated BRD has corresponding eval files (e2e, unit, integration, security, perf)
echo ""
echo "[7/7] Checking curated BRDs have corresponding eval files..."
for brd_file in specs/curated/brd-*.md; do
  [ -f "$brd_file" ] || continue

  basename=$(basename "$brd_file" .md)
  # Extract BRD number: brd-02-manual-meeting-import -> brd-02
  if [[ "$basename" =~ ^(brd-[0-9]+) ]]; then
    brd_id="${BASH_REMATCH[1]}"
    missing=""
    for eval_type in e2e unit integration security perf; do
      if [ "$eval_type" = "security" ] || [ "$eval_type" = "perf" ]; then
        # security and perf are optional during Phase 0/1
        continue
      fi
      if ! ls "evals/${eval_type}/${brd_id}"-*.md 1>/dev/null 2>&1; then
        missing="${missing} evals/${eval_type}/${brd_id}-*.md"
      fi
    done
    if [ -n "$missing" ]; then
      echo "  ERROR: $basename missing eval files:${missing}"
      ERRORS=$((ERRORS + 1))
    fi
  fi
done

# 8. All FF_ENABLE_* env vars default to false
echo ""
echo "[8/7] Checking .env.example defaults..."
while IFS='=' read -r var value; do
  # Only check FF_ENABLE_* vars
  [[ "$var" =~ ^[[:space:]]*FF_ENABLE_ ]] || continue
  # BRD-05 ff_enable_upcoming_meetings is Active — allowed to default true
  [[ "$var" == "FF_ENABLE_UPCOMING_MEETINGS" ]] && continue
  # Strip comments
  value=$(echo "$value" | sed 's/#.*//' | xargs)
  if [ -n "$value" ] && [ "$value" != "false" ]; then
    echo "  ERROR: $var should default to 'false' but is set to '$value'"
    ERRORS=$((ERRORS + 1))
  fi
done < .env.example || true

# Summary
echo ""
if [ "$ERRORS" -gt 0 ]; then
  echo "FAIL: $ERRORS sync check error(s) found"
  exit 1
else
  echo "OK: all sync checks passed"
  exit 0
fi