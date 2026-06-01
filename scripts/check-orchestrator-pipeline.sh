#!/bin/bash
# check-orchestrator-pipeline.sh — deterministic Kanban pipeline graph checker
#
# Usage:
#   BRD_ID=brd-03 HERMES_KANBAN_BOARD=contextpilot scripts/check-orchestrator-pipeline.sh --phase implementation
#
# Phases:
#   validation       require pre-validation artifacts and curation/refinement chain
#   implementation   require full pre-implementation gate chain (default)
#   qa               require implementation tasks and QA children
#   release          require implementation + QA signoff/repair loop closed + done-auditor reality check
#
# This script checks task presence, status, dependency links, gate verdicts,
# open/blocker language, UI two-task pattern, and QA child-task enforcement.
set -euo pipefail

BOARD="${HERMES_KANBAN_BOARD:-contextpilot}"
BRD_ID="${BRD_ID:-}"
PHASE="implementation"
STRICT="${STRICT:-1}"
TMP_JSON="$(mktemp)"
trap 'rm -f "$TMP_JSON"' EXIT

usage() {
  sed -n '1,28p' "$0"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --phase)
      if [ "$#" -lt 2 ]; then echo "ERROR: --phase requires a value"; exit 2; fi
      PHASE="$2"; shift 2 ;;
    --board)
      if [ "$#" -lt 2 ]; then echo "ERROR: --board requires a value"; exit 2; fi
      BOARD="$2"; shift 2 ;;
    --help|-h)
      usage; exit 0 ;;
    *)
      echo "ERROR: unknown argument: $1"; usage; exit 2 ;;
  esac
done

case "$PHASE" in
  validation|implementation|qa|release) ;;
  *) echo "ERROR: unsupported --phase '$PHASE'"; exit 2 ;;
esac

if [ -z "$BRD_ID" ]; then
  echo "ERROR: BRD_ID is required, e.g. BRD_ID=brd-03 HERMES_KANBAN_BOARD=$BOARD $0 --phase implementation"
  exit 2
fi

BRD_ID_LOWER="$(printf '%s' "$BRD_ID" | tr '[:upper:]' '[:lower:]')"

echo "=== ContextPilot Orchestrator Pipeline Check v2 ==="
echo "Board: $BOARD"
echo "BRD:   $BRD_ID_LOWER"
echo "Phase: $PHASE"
echo ""

hermes kanban --board "$BOARD" list --json > "$TMP_JSON"

python3 - "$TMP_JSON" "$BRD_ID_LOWER" "$BOARD" "$PHASE" "$STRICT" <<'PY'
import glob
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from typing import Callable, Iterable

json_path, brd_id, board, phase, strict_s = sys.argv[1:6]
strict = strict_s not in ("0", "false", "False", "no")
with open(json_path, encoding="utf-8") as f:
    raw_tasks = json.load(f)

errors: list[str] = []
warnings: list[str] = []
_show_cache: dict[str, str] = {}

APPROVE_TERMS = (
    "verdict: approve", "pm verdict: approve", "approve_phase", "approve phase",
    "approved to proceed", "approve to proceed", "approved phase", "proceed to implementation",
    "proceed with implementation", "approved_raw_brd", "approved raw brd",
)
REPAIR_TERMS = (
    "needs_repair", "needs repair", "verdict: repair", "pm verdict: repair",
    "needs_attention", "needs attention", "not approved", "cannot approve",
    "do not proceed", "blocked", "blocker", "critical blocker",
)
OPEN_ITEM_TERMS = (
    "open_items", "open items", "action_items", "action items", "unresolved",
    "needs_repair", "needs repair", "needs_attention", "needs attention",
    "blocker", "blocked", "critical:", "critical blocker", "phase 1 blocker",
    "cannot approve", "not approved", "do not proceed",
)
OPEN_ITEM_CLEAR_TERMS = (
    "open_items: none", "open items: none", "no open items", "open_items=[]",
    '"open_items": []', "no blockers", "no unresolved", "no action items",
)
DONE_AUDIT_PASS_TERMS = ("status: trustworthy", "status: mostly trustworthy", "**status:** trustworthy", "**status:** mostly trustworthy", "trustworthy", "mostly trustworthy")
DONE_AUDIT_FAIL_TERMS = ("status: mixed", "status: risky", "status: not done", "**status:** mixed", "**status:** risky", "**status:** not done", "not done", "p0", "p1 blocker", "critical blocker")
LOAD_RE = re.compile(r"load\s+([a-z0-9_-]+)\s+skill", re.I)
TASK_ID_RE = re.compile(r"t_[a-f0-9]+")

@dataclass
class Task:
    data: dict
    show: str = ""
    parents: list[str] = field(default_factory=list)

    @property
    def id(self) -> str:
        return str(self.data.get("id") or "")

    @property
    def title(self) -> str:
        return str(self.data.get("title") or "")

    @property
    def body(self) -> str:
        return str(self.data.get("body") or "")

    @property
    def result(self) -> str:
        return str(self.data.get("result") or "")

    @property
    def assignee(self) -> str:
        return str(self.data.get("assignee") or "")

    @property
    def status(self) -> str:
        return str(self.data.get("status") or "")

    @property
    def skills(self) -> list[str]:
        v = self.data.get("skills") or []
        return [str(x) for x in v] if isinstance(v, list) else []

    @property
    def lower(self) -> str:
        return "\n".join([self.title, self.body, self.result, self.assignee, " ".join(self.skills)]).lower()

    @property
    def full_lower(self) -> str:
        return "\n".join([self.lower, self.show_text()]).lower()

    def show_text(self) -> str:
        if not self.id:
            return ""
        if self.id not in _show_cache:
            try:
                out = subprocess.check_output(
                    ["hermes", "kanban", "--board", board, "show", self.id],
                    text=True,
                    stderr=subprocess.DEVNULL,
                )
            except Exception:
                out = ""
            _show_cache[self.id] = out
        self.show = _show_cache[self.id]
        if not self.parents:
            self.parents = parse_parents(self.show)
        return self.show

    def summary_lower(self) -> str:
        s = self.show_text().lower()
        if "latest summary:" in s:
            s = s.split("latest summary:", 1)[1]
            for marker in ("\ncomments (", "\nevents (", "\nruns ("):
                if marker in s:
                    s = s.split(marker, 1)[0]
        return "\n".join([self.result.lower(), s])

    def one_line(self) -> str:
        first = self.title.splitlines()[0] if self.title else ""
        return f"{self.id} {self.status} {self.assignee} | {first[:110]}"


def parse_parents(show: str) -> list[str]:
    for line in show.splitlines():
        stripped = line.strip().lower()
        if stripped.startswith("parents:"):
            return TASK_ID_RE.findall(stripped)
    return []


tasks = [Task(t) for t in raw_tasks]
by_id = {t.id: t for t in tasks}

def has(t: Task, *terms: str, full: bool = False) -> bool:
    text = t.full_lower if full else t.lower
    return all(term.lower() in text for term in terms)


def any_of(t: Task, terms: Iterable[str], full: bool = False) -> bool:
    text = t.full_lower if full else t.lower
    return any(term.lower() in text for term in terms)


def primary_text(t: Task) -> str:
    """Text used to classify a task's primary BRD/stage.

    Avoid comments/show output here: those often mention parent or previous BRDs and
    can make a BRD-03 task look like a BRD-02 task.
    """
    return "\n".join([t.title, t.body]).lower()


def primary_title(t: Task) -> str:
    # Kanban titles sometimes contain the entire task body after newlines.
    # The first line is the actual title/objective and should define the primary BRD.
    return (t.title.splitlines()[0] if t.title else "").lower()


def brd_task(t: Task, full: bool = False) -> bool:
    text = primary_text(t)
    title_text = primary_title(t)
    title_brds = re.findall(r"brd-[0-9]+", title_text)
    if title_brds:
        return brd_id in title_brds
    text_brds = re.findall(r"brd-[0-9]+", text)
    if text_brds:
        return brd_id in text_brds
    if full:
        show_brds = re.findall(r"brd-[0-9]+", t.full_lower)
        return bool(show_brds) and brd_id in show_brds
    return False


def done(t: Task) -> bool:
    return t.status == "done"


def active_or_done(t: Task) -> bool:
    return t.status in {"ready", "running", "blocked", "done"}


def find_tasks(pred: Callable[[Task], bool], *, full: bool = False) -> list[Task]:
    hits = []
    for t in tasks:
        if full:
            t.show_text()
        try:
            if pred(t):
                hits.append(t)
        except Exception:
            continue
    return hits


def done_hits(name: str, pred: Callable[[Task], bool], *, full: bool = False, required: bool = True) -> list[Task]:
    hits = find_tasks(lambda t: done(t) and pred(t), full=full)
    if hits:
        print(f"  ✓ {name}: {hits[-1].one_line()}")
    else:
        msg = f"{name}: no done task found"
        if required:
            fail(msg)
        else:
            warn(msg)
    return hits


def fail(msg: str) -> None:
    errors.append(msg)
    print(f"  ✗ {msg}")


def warn(msg: str) -> None:
    warnings.append(msg)
    print(f"  ! {msg}")


def ok(msg: str) -> None:
    print(f"  ✓ {msg}")


def first_line(t: Task) -> str:
    src = t.body or t.title
    return src.strip().splitlines()[0] if src and src.strip() else ""


def load_skills(t: Task) -> list[str]:
    src = "\n".join([t.body, t.title])
    return [m.group(1).lower() for m in LOAD_RE.finditer(src)]


def approval_text(t: Task) -> str:
    return t.summary_lower()


def is_approval(t: Task) -> bool:
    text = approval_text(t)
    return any(term in text for term in APPROVE_TERMS) and not any(term in text for term in REPAIR_TERMS)


def has_open_items(t: Task) -> bool:
    text = t.summary_lower()
    if not text.strip():
        return False
    if any(clear in text for clear in OPEN_ITEM_CLEAR_TERMS):
        # Still fail if explicit repair/blocker appears alongside the clearing phrase.
        hard = ("needs_repair", "needs repair", "cannot approve", "not approved", "do not proceed")
        return any(term in text for term in hard)
    return any(term in text for term in OPEN_ITEM_TERMS)


def require_parent(child: Task, parents: list[Task], label: str) -> None:
    if not child.id or not parents:
        return
    child.show_text()
    parent_ids = {p.id for p in parents if p.id}
    if child.parents and parent_ids.intersection(child.parents):
        ok(f"dependency linked: {label} ({child.id}) has parent {sorted(parent_ids.intersection(child.parents))[0]}")
    elif child.parents:
        fail(f"dependency missing: {label} ({child.id}) parents={child.parents}, expected one of {sorted(parent_ids)}")
    else:
        fail(f"dependency missing: {label} ({child.id}) has no parent link, expected one of {sorted(parent_ids)}")


def require_static_artifacts() -> None:
    print("\n[1] Static artifacts and repo parity")
    missing = []
    for eval_type in ("e2e", "unit", "integration"):
        if not glob.glob(f"evals/{eval_type}/{brd_id}-*.md"):
            missing.append(f"evals/{eval_type}/{brd_id}-*.md")
    if missing:
        fail("missing eval contracts: " + " ".join(missing))
    else:
        ok("required eval contracts exist")

    if os.path.exists("scripts/check-status-sync.sh"):
        status = subprocess.run(["bash", "scripts/check-status-sync.sh"], stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)
        if status.returncode == 0:
            ok("scripts/check-status-sync.sh passes")
        else:
            fail("scripts/check-status-sync.sh fails")
    else:
        fail("scripts/check-status-sync.sh missing")


def require_skill_directives(candidates: list[Task]) -> None:
    print("\n[2] Task skill directive hygiene")
    checked = 0
    for t in candidates:
        loads = load_skills(t)
        if not loads:
            continue
        checked += 1
        fl = first_line(t).lower()
        if not LOAD_RE.search(fl):
            fail(f"{t.id} skill directive is not first line: {t.one_line()}")
        if len(set(loads)) > 1:
            fail(f"{t.id} stacks multiple skills {loads}: {t.one_line()}")
    if checked:
        ok(f"checked {checked} skill-bearing task(s)")
    else:
        warn("no skill-bearing tasks found for BRD; future tasks should start with 'Load <skill> skill. Then...'")


def require_project_scaffold() -> tuple[list[Task], list[Task]]:
    print("\n[3] Project scaffold gates")
    scaffold = done_hits(
        "project scaffold",
        lambda t: ("scaffold" in primary_title(t) and "review" not in primary_title(t) and t.assignee in {"pm", "ops"}),
        required=True,
    )
    review = done_hits(
        "scaffold-review by validator",
        lambda t: ("scaffold" in primary_title(t) and "review" in primary_title(t) and t.assignee == "validator"),
        required=True,
    )
    bad_self = find_tasks(lambda t: "scaffold" in primary_title(t) and "review" in primary_title(t) and t.assignee == "orchestrator")
    for t in bad_self:
        fail(f"scaffold-review self-assigned to orchestrator: {t.one_line()}")
    if scaffold and review:
        require_parent(review[-1], scaffold, "scaffold-review")
    return scaffold, review


def require_brd_chain() -> dict[str, list[Task]]:
    print("\n[4] BRD pipeline task chain")
    chain: dict[str, list[Task]] = {}
    chain["brainstorming"] = done_hits(
        "brainstorming/raw BRD",
        lambda t: brd_task(t) and ("brainstorming" in primary_title(t) or "raw brd" in primary_title(t) or "brainstorm" in primary_title(t)) and t.assignee in {"brainstormer", "pm", "spec-writer"},
        full=False,
        required=True,
    )
    approval = find_tasks(
        lambda t: done(t) and brd_task(t, full=True) and any(term in t.full_lower for term in ("approved_raw_brd", "approved raw brd", "shakil approved", "approved by shakil")),
        full=True,
    )
    if approval:
        ok("raw BRD approval recorded: " + approval[-1].one_line())
    else:
        fail("raw BRD approval not recorded with deterministic phrase (expected APPROVED_RAW_BRD / Shakil approved)")
    chain["raw_approval"] = approval

    chain["curation"] = done_hits(
        "PM curation / curating-artifacts",
        lambda t: brd_task(t) and t.assignee == "pm" and ("curating-artifacts" in primary_title(t) or "curation" in primary_title(t) or "curated" in primary_title(t) or "curate" in primary_title(t)),
        required=True,
    )
    chain["refinement"] = done_hits(
        "systematic-refinement/design refinement",
        lambda t: brd_task(t) and t.assignee in {"architect", "refiner"} and ("systematic-refinement" in primary_title(t) or "design-refinement" in primary_title(t) or "refinement" in primary_title(t)),
        required=True,
    )
    chain["subagent"] = done_hits(
        "subagent-driven-development planning",
        lambda t: brd_task(t) and t.assignee in {"architect", "refiner"} and "subagent-driven-development" in primary_title(t),
        required=True,
    )
    chain["eval_contracts"] = done_hits(
        "eval contract task",
        lambda t: brd_task(t) and t.assignee == "qa" and ("eval" in primary_title(t) and ("contract" in primary_title(t) or "e2e" in primary_title(t) or "unit" in primary_title(t) or "integration" in primary_title(t))),
        required=True,
    )
    chain["flag_parity"] = done_hits(
        "flag/contract parity task",
        lambda t: brd_task(t) and t.assignee == "ops" and (("flag" in primary_title(t) and "parity" in primary_title(t)) or ("contract" in primary_title(t) and "parity" in primary_title(t)) or "sync parity" in primary_title(t)),
        required=True,
    )
    chain["completeness"] = done_hits(
        "completeness-score",
        lambda t: brd_task(t) and "completeness-score" in primary_title(t),
        required=True,
    )
    chain["pm_after_completeness"] = done_hits(
        "PM approval after completeness-score",
        lambda t: brd_task(t, full=True) and t.assignee == "pm" and "completeness-score" in t.full_lower and is_approval(t),
        full=True,
        required=True,
    )
    chain["validate"] = done_hits(
        "validate-design",
        lambda t: brd_task(t) and "validate-design" in primary_title(t),
        required=True,
    )
    chain["pm_after_validate"] = done_hits(
        "PM approval after validate-design",
        lambda t: brd_task(t, full=True) and t.assignee == "pm" and "validate-design" in t.full_lower and is_approval(t),
        full=True,
        required=True,
    )
    chain["production"] = done_hits(
        "production-checklist",
        lambda t: brd_task(t) and "production-checklist" in primary_title(t),
        required=True,
    )
    return chain


def require_dependency_chain(chain: dict[str, list[Task]]) -> None:
    print("\n[5] Dependency link checks")
    pairs = [
        ("curation", "brainstorming"),
        ("refinement", "curation"),
        ("subagent", "refinement"),
        ("eval_contracts", "curation"),
        ("flag_parity", "curation"),
        ("completeness", "curation"),
        ("completeness", "eval_contracts"),
        ("completeness", "flag_parity"),
        ("pm_after_completeness", "completeness"),
        ("validate", "pm_after_completeness"),
        ("pm_after_validate", "validate"),
        ("production", "pm_after_validate"),
    ]
    for child_key, parent_key in pairs:
        if chain.get(child_key) and chain.get(parent_key):
            require_parent(chain[child_key][-1], chain[parent_key], child_key)


def require_no_open_gate_items(chain: dict[str, list[Task]]) -> None:
    print("\n[6] Gate open-item/blocker scan")
    gate_keys = ["completeness", "pm_after_completeness", "validate", "pm_after_validate", "production"]
    checked = 0
    for key in gate_keys:
        for t in chain.get(key, []):
            checked += 1
            if has_open_items(t):
                fail(f"{key} has unresolved/open/blocker language in latest summary: {t.one_line()}")
    if checked:
        ok(f"checked {checked} gate summary/summaries for blockers")


def implementation_tasks() -> list[Task]:
    return find_tasks(lambda t: brd_task(t) and t.assignee in {"backend", "frontend-eng"} and active_or_done(t) and any(term in primary_title(t) for term in ("implement", "implementation", "build", "spec-driven", "svelte-frontend-design")))


def require_no_early_implementation(chain: dict[str, list[Task]]) -> None:
    print("\n[7] Downstream implementation guard")
    impls = implementation_tasks()
    required_keys = ["completeness", "pm_after_completeness", "validate", "pm_after_validate", "production"]
    gates_passed = all(chain.get(k) for k in required_keys)
    if impls and not gates_passed:
        for t in impls:
            fail(f"implementation exists before all gates passed: {t.one_line()}")
    elif impls:
        ok(f"implementation task(s) exist and pre-implementation gates are present: {len(impls)}")
    else:
        ok("no implementation tasks exist yet")
    if gates_passed:
        for impl in impls:
            require_parent(impl, chain["production"], "implementation")


def require_ui_pattern() -> None:
    print("\n[8] UI two-task pattern")
    frontend = find_tasks(lambda t: brd_task(t) and t.assignee == "frontend-eng" and active_or_done(t))
    if not frontend:
        ok("no frontend implementation task detected for this BRD")
        return
    svelte = [t for t in frontend if "svelte-frontend-design" in load_skills(t)]
    polish = [t for t in frontend if "impeccable" in load_skills(t)]
    if svelte:
        ok("svelte-frontend-design implementation task exists: " + svelte[-1].one_line())
    else:
        fail("frontend implementation exists but no svelte-frontend-design task found")
    if polish:
        ok("impeccable polish task exists: " + polish[-1].one_line())
    else:
        fail("frontend implementation exists but no separate impeccable polish task found")
    for t in frontend:
        loads = set(load_skills(t))
        if "svelte-frontend-design" in loads and "impeccable" in loads:
            fail(f"frontend task stacks svelte-frontend-design and impeccable in one task: {t.one_line()}")
    if svelte and polish:
        if svelte[-1].id == polish[-1].id:
            fail("svelte implementation and impeccable polish resolve to same task; must be separate tasks")
        else:
            ok("frontend implementation and polish are separate tasks")


def require_qa_children(release: bool = False) -> None:
    print("\n[9] QA child-task enforcement")
    impls = implementation_tasks()
    if not impls:
        fail("no implementation tasks found; cannot verify QA children")
        return
    qa_tasks = find_tasks(lambda t: brd_task(t) and t.assignee == "qa" and active_or_done(t), full=False)
    for impl in impls:
        impl.show_text()
        kind = "frontend" if impl.assignee == "frontend-eng" else "backend"
        required = [f"test coverage ({kind})", f"brd compliance ({kind})"]
        for label in required:
            hits = []
            for q in qa_tasks:
                q.show_text()
                qtext = q.full_lower
                if "qa" in qtext and label in qtext and impl.id in q.parents:
                    hits.append(q)
            if hits:
                ok(f"QA child exists for {impl.id}: {label} -> {hits[-1].id} ({hits[-1].status})")
                if release:
                    if hits[-1].status != "done":
                        fail(f"release requires QA done: {hits[-1].one_line()}")
                    elif has_open_items(hits[-1]):
                        fail(f"QA task has unresolved/open/blocker language: {hits[-1].one_line()}")
            else:
                fail(f"missing QA child for {impl.id}: {label}")


def require_done_auditor_gate() -> None:
    print("\n[10] Definition-of-done audit gate")
    audits = find_tasks(
        lambda t: brd_task(t, full=True)
        and t.assignee == "done-auditor"
        and active_or_done(t)
        and ("definition-of-done" in t.full_lower or "done audit" in t.full_lower or "done-auditor" in t.full_lower),
        full=True,
    )
    if not audits:
        fail("release requires done-auditor definition-of-done audit task")
        return
    audit = audits[-1]
    if audit.status != "done":
        fail(f"release requires done-auditor audit done: {audit.one_line()}")
        return
    summary = audit.summary_lower()
    if any(term in summary for term in DONE_AUDIT_FAIL_TERMS):
        fail(f"done-auditor audit reports blocker/failing verdict: {audit.one_line()}")
    elif any(term in summary for term in DONE_AUDIT_PASS_TERMS) and not has_open_items(audit):
        ok("done-auditor audit passed: " + audit.one_line())
    else:
        fail(f"done-auditor audit lacks explicit Trustworthy/Mostly trustworthy blocker-free verdict: {audit.one_line()}")


# Candidate skill-bearing tasks: all BRD tasks + project scaffold/review.
brd_candidates = find_tasks(lambda t: brd_task(t))
project_candidates = find_tasks(lambda t: "scaffold" in primary_title(t) or "review" in primary_title(t))
require_static_artifacts()
require_skill_directives(brd_candidates + project_candidates)
scaffold, scaffold_review = require_project_scaffold()
chain = require_brd_chain()
require_dependency_chain(chain)
require_no_open_gate_items(chain)

if phase in {"implementation", "qa", "release"}:
    require_no_early_implementation(chain)
    require_ui_pattern()
if phase in {"qa", "release"}:
    require_qa_children(release=(phase == "release"))
if phase == "release":
    require_done_auditor_gate()

print("\n=== Result ===")
if warnings:
    print(f"WARNINGS: {len(warnings)}")
    for msg in warnings:
        print(f"  - {msg}")
if errors:
    print(f"BLOCKED: {len(errors)} orchestrator pipeline check(s) failed for {brd_id}")
    for msg in errors:
        print(f"  - {msg}")
    sys.exit(1)
print(f"OK: orchestrator pipeline is valid for {brd_id} at phase={phase}")
PY
