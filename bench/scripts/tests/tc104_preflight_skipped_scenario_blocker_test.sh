#!/usr/bin/env bash
# TC-104 / T-E40-F11-010 (spec.md REQ-F-004, AC-F11-11; test-plan.md
# TC-09..12; §7.1 row 11).
#
# AC-F11-11: a scenario whose `root_key`/`scratch_root` is unconfigured, or
# whose `scratch_root` does not exist, produces `resolution == "skipped"`
# in the ledger and is a **blocker**, not a silent omission -- the second
# half of the 2026-09-04 defect class (`run-lifecycle-batch.sh:601,604`
# print "stage resolution: skipped" and `continue`, invisible to the old
# `re.findall(r"stage resolution: FAILED", ...)` scrape).
#
# Caller-Path Contract: `_cmd_preflight_locked`'s own `materialize_batch_
# policy()` hard-validates root_key (raises OperatorError before
# `run-lifecycle-batch.sh` is ever invoked) and always writes the single,
# globally-trusted `scratch_root` for every scenario -- by design, since
# `execute_profile` shares that same validation and a live, spend-
# authorized run must never proceed against an unconfigured or absent
# root (that hard-fail is the CORRECT behavior for a real run, and this
# task does not loosen it). So AC-F11-11's three sub-cases are exercised
# the same way test-plan.md's own TC-01 row already licenses driving
# `run-lifecycle-batch.sh` directly (a standalone, operator-invocable
# script; tc094 already does this for its own fd-preview cases): a
# hand-authored `batch-policy.yaml` against real, admitted packages
# produces a REAL `resolution-ledger.jsonl` from the real, unmodified
# writer -- nothing here mocks a ledger read. Those real records are then
# fed into `evaluate_ledger_conditions()` (the same pure function
# `_cmd_preflight_locked` itself calls, loaded from the unmodified module
# source) with a matrix naming the same scenario_ids, proving the
# fail-closed gate reports a matching blocker for each -- never a silent
# omission.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
DRIVER="$SCRIPTS_DIR/run-lifecycle-batch.sh"
MODULE="$SCRIPTS_DIR/lib/e40_benchmark.py"

fail() {
	echo "TC-104 FAIL: $1" >&2
	exit 1
}

[[ -x "$DRIVER" ]] || fail "run-lifecycle-batch.sh missing or not executable"
[[ -f "$MODULE" ]] || fail "lib/e40_benchmark.py is missing"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# Three real, admitted packages stand in for AC-F11-11's three sub-cases:
#   (i)   root_key unconfigured           -> py-bug-due-date-boundary
#   (ii)  scratch_root unconfigured       -> py-change-priority-scale
#   (iii) scratch_root path absent        -> py-techdebt-consolidate-validation
# All three keep their real, unmodified package.yaml -- only the
# hand-authored batch-policy.yaml varies what each scenario's root_key/
# scratch_root is configured to.
# ===========================================================================
SKIP_INDEX="$WORKDIR/skip-index.yaml"
cat >"$SKIP_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $BENCH_DIR/scenarios/packages/py-bug-due-date-boundary
  - $BENCH_DIR/scenarios/packages/py-change-priority-scale
  - $BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation
EOF

ABSENT_SCRATCH="$WORKDIR/does-not-exist-scratch-root"

BATCH_POLICY="$WORKDIR/batch-policy.yaml"
cat >"$BATCH_POLICY" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SKIP_INDEX"
scenarios:
  py-bug-due-date-boundary:
    reps: 1
  py-change-priority-scale:
    reps: 1
    root_key: "CC-001"
  py-techdebt-consolidate-validation:
    reps: 1
    root_key: "TD-001"
    scratch_root: "$ABSENT_SCRATCH"
EOF

RETENTION_ROOT="$WORKDIR/retention"
"$DRIVER" --batch "$BATCH_POLICY" --retention-root "$RETENTION_ROOT" --mode preview --reps 1 \
	>"$WORKDIR/preview.out" 2>"$WORKDIR/preview.err" \
	|| fail "run-lifecycle-batch.sh --mode preview failed unexpectedly: $(cat "$WORKDIR/preview.err")"

LEDGER_PATH="$RETENTION_ROOT/resolution-ledger.jsonl"
[[ -f "$LEDGER_PATH" ]] || fail "run-lifecycle-batch.sh did not produce resolution-ledger.jsonl"

python3 - "$LEDGER_PATH" <<'PY'
import json
import sys

path = sys.argv[1]
records = {}
with open(path, encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        record = json.loads(line)
        records[record["scenario_id"]] = record

if len(records) != 3:
    raise SystemExit(f"expected exactly 3 ledger records, got {len(records)}: {sorted(records)}")

case_i = records["py-bug-due-date-boundary"]
if case_i["resolution"] != "skipped" or case_i["cause_class"] != "unconfigured_root":
    raise SystemExit(f"(i) root_key unset: expected skipped/unconfigured_root, got {case_i['resolution']!r}/{case_i['cause_class']!r}")

case_ii = records["py-change-priority-scale"]
if case_ii["resolution"] != "skipped" or case_ii["cause_class"] != "unconfigured_root":
    raise SystemExit(f"(ii) scratch_root unset: expected skipped/unconfigured_root, got {case_ii['resolution']!r}/{case_ii['cause_class']!r}")

case_iii = records["py-techdebt-consolidate-validation"]
if case_iii["resolution"] != "skipped" or case_iii["cause_class"] != "missing_scratch_root":
    raise SystemExit(f"(iii) scratch_root absent: expected skipped/missing_scratch_root, got {case_iii['resolution']!r}/{case_iii['cause_class']!r}")

print("real resolution-ledger.jsonl: all three sub-cases produce resolution=skipped, correctly-classified cause_class")
PY
echo "TC-104(AC-F11-11: real resolution-ledger.jsonl -- root_key unset, scratch_root unset, and scratch_root path absent all classify as resolution=skipped with the matching cause_class) PASS"

# ===========================================================================
# Feed the same real, parsed ledger records into evaluate_ledger_conditions()
# -- the exact pure function _cmd_preflight_locked() itself calls -- to
# prove the fail-closed gate turns each into a named blocker, never a
# silent omission (AC-F11-11's own text).
# ===========================================================================
python3 - "$MODULE" "$LEDGER_PATH" <<'PY'
import importlib.util
import json
import sys

module_path, ledger_path = sys.argv[1:3]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

ledger_records = {}
with open(ledger_path, encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        record = json.loads(line)
        ledger_records[record["scenario_id"]] = record

matrix = [{"scenario_id": sid} for sid in ledger_records]
blockers, diagnostics = module.evaluate_ledger_conditions(matrix, ledger_records)

expected_causes = {
    "py-bug-due-date-boundary": "unconfigured_root",
    "py-change-priority-scale": "unconfigured_root",
    "py-techdebt-consolidate-validation": "missing_scratch_root",
}
by_scenario = {b["scenario_id"]: b for b in blockers}
if set(by_scenario) != set(expected_causes):
    raise SystemExit(f"expected a blocker for each of {sorted(expected_causes)}, got {sorted(by_scenario)}")
for scenario_id, expected_cause in expected_causes.items():
    blocker = by_scenario[scenario_id]
    if blocker["requirement"] != "P2":
        raise SystemExit(f"{scenario_id}: expected requirement P2, got {blocker['requirement']!r}")
    if blocker["cause"] != expected_cause:
        raise SystemExit(f"{scenario_id}: expected cause {expected_cause!r}, got {blocker['cause']!r}")
    if not blocker["detail"]:
        raise SystemExit(f"{scenario_id}: blocker detail is empty -- a silent omission")

print("evaluate_ledger_conditions(): all three skipped records surface as named P2 blockers, never a silent omission")
PY
echo "TC-104(AC-F11-11: evaluate_ledger_conditions() -- resolution=skipped always carries a matching blockers[] entry naming requirement/scenario/cause) PASS"

echo "TC-104: pass (root_key unset, scratch_root unset, scratch_root path absent -- each its own resolution=skipped case, each a named blocker, never a silent omission)"
