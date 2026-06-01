#!/bin/bash
# check-pipeline-gates.sh — BRD-specific pipeline gate check before implementation
#
# Usage:
#   BRD_ID=brd-03 HERMES_KANBAN_BOARD=contextpilot scripts/check-pipeline-gates.sh
#
# This is stricter than check-status-sync.sh. Sync check proves static repo parity;
# this script proves the Kanban pipeline is at the right phase for a BRD.
set -euo pipefail

BOARD="${HERMES_KANBAN_BOARD:-contextpilot}"
BRD_ID="${BRD_ID:-}"
TMP_JSON="$(mktemp)"
trap 'rm -f "$TMP_JSON"' EXIT

echo "=== ContextPilot Pipeline Gate Check ==="
echo "Board: $BOARD"

if [ -z "$BRD_ID" ]; then
  echo "ERROR: BRD_ID is required, e.g. BRD_ID=brd-03 HERMES_KANBAN_BOARD=$BOARD scripts/check-pipeline-gates.sh"
  exit 2
fi

BRD_ID_LOWER="$(printf '%s' "$BRD_ID" | tr '[:upper:]' '[:lower:]')"
echo "BRD: $BRD_ID_LOWER"

hermes kanban --board "$BOARD" list --json > "$TMP_JSON"

python3 - "$TMP_JSON" "$BRD_ID_LOWER" "$BOARD" <<'PY'
import glob, json, subprocess, sys
from pathlib import Path

path, brd_id, board = sys.argv[1:4]
with open(path) as f:
    tasks = json.load(f)

errors = 0

def title(t):
    return str(t.get("title") or "").lower()

def body(t):
    return str(t.get("body") or "").lower()

def result(t):
    return str(t.get("result") or "").lower()

_show_cache = {}
def show_text(t):
    task_id = t.get("id")
    if not task_id:
        return ""
    if task_id not in _show_cache:
        try:
            out = subprocess.check_output(
                ["hermes", "kanban", "--board", board, "show", task_id],
                text=True,
                stderr=subprocess.DEVNULL,
            )
        except Exception:
            out = ""
        _show_cache[task_id] = out.lower()
    return _show_cache[task_id]

def task_result_text(t):
    # list --json can omit latest summary; show includes it. Only inspect
    # latest summary/result, not the task body where "approve" and "repair"
    # can appear as decision options.
    shown = show_text(t)
    if "latest summary:" in shown:
        shown = shown.split("latest summary:", 1)[1]
        for marker in ("\ncomments (", "\nevents (", "\nruns ("):
            if marker in shown:
                shown = shown.split(marker, 1)[0]
    return "\n".join([result(t), shown])

def all_text(t):
    return "\n".join([title(t), body(t), result(t), str(t.get("assignee") or "").lower()])

def brd_task(t):
    return brd_id in all_text(t)

def one_line(t):
    return f"{t.get('id')} {t.get('status')} {t.get('assignee')} | {str(t.get('title') or '').splitlines()[0]}"

def pass_(msg):
    print(f"  ✓ {msg}")

def fail(msg):
    global errors
    errors += 1
    print(f"  ✗ {msg}")

def done_title_contains(keyword):
    for t in tasks:
        if brd_task(t) and t.get("status") == "done" and keyword in title(t):
            return t
    return None

def pm_approval_after(label):
    # PM must explicitly approve/proceed in the RESULT/SUMMARY, not merely list
    # "approve" as one possible option in the task body.
    positive = ("approve_phase" in label or "approved to proceed" in label or
                "approve to proceed" in label or "verdict: approve" in label or
                "pm verdict: approve" in label or "approved phase" in label)
    negative = ("needs_repair" in label or "needs repair" in label or
                "verdict: repair" in label or "pm verdict: repair" in label or
                "not approve" in label)
    return positive and not negative

print("\n[1/7] Checking eval contracts exist before implementation...")
missing = []
for eval_type in ("e2e", "unit", "integration"):
    if not glob.glob(f"evals/{eval_type}/{brd_id}-*.md"):
        missing.append(f"evals/{eval_type}/{brd_id}-*.md")
if missing:
    fail("Missing required eval contracts: " + " ".join(missing))
else:
    pass_("Required eval contracts exist")

print("\n[2/7] Checking static sync parity...")
status = subprocess.run(["bash", "scripts/check-status-sync.sh"], stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)
if status.returncode == 0:
    pass_("scripts/check-status-sync.sh passes")
else:
    fail("scripts/check-status-sync.sh fails")

print("\n[3/7] Checking completeness-score task is done...")
t = done_title_contains("completeness-score")
if t:
    pass_("completeness-score done: " + one_line(t))
else:
    fail(f"No done completeness-score task found in title for {brd_id}")

print("\n[4/7] Checking PM review after completeness-score approved proceed...")
pm_hits = [t for t in tasks if brd_task(t) and t.get("status") == "done" and t.get("assignee") == "pm" and pm_approval_after(task_result_text(t))]
if pm_hits:
    pass_("PM approval/proceed verdict: " + one_line(pm_hits[-1]))
else:
    fail(f"No done PM task with explicit approve/proceed result for {brd_id}")

print("\n[5/7] Checking validate-design task is done...")
t = done_title_contains("validate-design")
if t:
    pass_("validate-design done: " + one_line(t))
else:
    fail(f"No done validate-design task found in title for {brd_id}")

print("\n[6/7] Checking latest PM/validator readiness verdict approved proceed...")
# Allows either PM explicit approval or a validator readiness re-run with APPROVE_PHASE_2,
# but orchestrator SOUL still requires a PM review before dispatching new implementation.
readiness_hits = [t for t in tasks if brd_task(t) and t.get("status") == "done" and (t.get("assignee") in ("pm", "validator")) and pm_approval_after(task_result_text(t))]
if readiness_hits:
    pass_("Readiness approval/proceed verdict: " + one_line(readiness_hits[-1]))
else:
    fail(f"No approved PM/validator readiness verdict found for {brd_id}")

print("\n[7/7] Checking production-checklist is done before implementation...")
t = done_title_contains("production-checklist")
if t:
    pass_("production-checklist done: " + one_line(t))
else:
    fail(f"No done production-checklist task found in title for {brd_id}")

print("")
if errors:
    print(f"BLOCKED: {errors} pipeline gate(s) not satisfied for {brd_id}")
    sys.exit(1)
print(f"OK: all pipeline gates passed for {brd_id}")
PY
