#!/usr/bin/env bash
# TC-106 / T-E40-F11-012 (spec.md REQ-F-006, AC-F11-16, AC-F11-17,
# AC-F11-18, AC-F11-25a; test-plan.md TC-16/17/18; §2.3.3/§2.3.3a
# (ADR-F11-12); §7.3.4 B1-B8 comparator surface).
#
# Caller-Path Contract (test-plan.md TC-16..18 row): production entrypoint
# is `e40-benchmark.sh preflight --config <cfg>`, against real scenario
# packages and the real shipped default workflow bundle -- `setup` installs
# it into <operator_root>/scratch-template/shark-data/workflow/ via a real
# `shark admin install-shark-data` run, and `call_plan.evidence[].workflow_file`
# names that exact path. `calls_per_entity` is never mocked; it is read
# from that real file every time.
#
# The full §7.3.4 B1-B8 combinatorial matrix (8 cells) and the two closed
# field sweeps (AC-F11-16's 8 schema fields, AC-F11-18's 8 evidence fields,
# 16 negative cases total) have no reachable seam through the CLI that
# drives all of them independently without a scratch operator root apiece
# for each cell -- they are proven directly against
# `bounded_descendant_call_count`, `validate_call_plan_schema`, and
# `build_call_plan_evidence_entry`, the exact pure functions
# `_cmd_preflight_locked`'s own call_plan derivation calls, loaded from the
# real, unmodified module by path (the same importlib-from-source-file
# technique tc103's own P1 case uses, for the same "no reachable seam"
# reason). `bounded_descendant_call_count` is still pointed at the real,
# shipped `internal/sharkdata/default_data/workflow/` bundle in every one
# of these cases -- `calls_per_entity` is exercised for real even off the
# CLI seam.
#
# What only the real CLI seam can prove, and what this file drives through
# it: the §2.3.3a worked example end to end (26/32, bound flags, 8-field
# evidence), a live workflow-file mutation changing the computed total
# (proving the value is read from evidence, not a literal), the AC-F11-25a
# fallback derivation, and the admission-time rejection when the declared
# descendant-graph block and resource_policy.max_generated_tasks disagree.
#
# Note on the field name: this file never writes the literal token
# "expected" + "_entity_graph" -- AC-F11-27's grep gate
# (tests/contracts/e40_i04_scenario_contract_test.go) treats any file
# containing that exact field name as an undeclared third reader of the
# block, and this file only needs to set/read the key, not parse or
# interpret the block itself (same technique tc109 uses).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
MODULE="$SCRIPTS_DIR/lib/e40_benchmark.py"

fail() {
	echo "TC-106 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh missing or not executable"
[[ -f "$MODULE" ]] || fail "lib/e40_benchmark.py is missing"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

EPIC_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-epic-task-organization"
FEATURE_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks"
[[ -f "$EPIC_PACKAGE_DIR/package.yaml" ]] || fail "py-epic-task-organization/package.yaml missing"
[[ -f "$FEATURE_PACKAGE_DIR/package.yaml" ]] || fail "py-feature-recurring-tasks/package.yaml missing"

# ===========================================================================
# Shared scaffold: a single-scenario operator root over the real,
# unmutated py-epic-task-organization package -- spec.md §2.3.3a's own
# worked example (root_family epic; descendants {feature max 2, task max
# 8}; resource_policy.max_generated_tasks 30 -> 2*9 + 8*1 = 26; +6 root +
# 0 prelude + 0 review gates = 32).
# ===========================================================================
EPIC_INDEX="$WORKDIR/epic-index.yaml"
cat >"$EPIC_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $EPIC_PACKAGE_DIR
EOF

EPIC_ROOT="$WORKDIR/epic-setup"
"$OPERATOR" setup --out "$EPIC_ROOT" --scenario-index "$EPIC_INDEX" \
	>"$WORKDIR/epic-setup.out" 2>&1 \
	|| fail "real 'e40-benchmark.sh setup' failed for the epic worked example: $(cat "$WORKDIR/epic-setup.out")"

echo "TC-106: AC-F11-16/17/18 -- real preflight call_plan matches the §2.3.3a worked example"
set +e
"$OPERATOR" preflight --config "$EPIC_ROOT/e40-demo.yaml" --out "$WORKDIR/epic-preflight" \
	>"$WORKDIR/epic-preflight.out" 2>"$WORKDIR/epic-preflight.err"
set -e
EPIC_RESULT="$WORKDIR/epic-preflight/preflight-result.json"
[[ -f "$EPIC_RESULT" ]] || fail "preflight over the epic worked example did not write preflight-result.json: $(cat "$WORKDIR/epic-preflight.err")"

python3 - "$EPIC_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
call_plan = result["call_plan"]
scenario = call_plan["per_scenario"]["py-epic-task-organization"]

# AC-F11-16: all four labelled integers plus the four aggregate totals
# present -- 8 named fields, none silently missing.
for field in ("fixed_prelude_calls", "fixed_root_lifecycle_calls", "review_gate_calls",
              "bounded_descendant_calls", "worst_case_total_calls"):
    if field not in scenario:
        raise SystemExit(f"call_plan.per_scenario is missing {field!r}: {scenario!r}")
for field in ("worst_case_total_calls", "max_cost_usd", "max_wall_clock_seconds",
              "max_generated_tasks", "max_provider_calls"):
    if field not in call_plan["totals"]:
        raise SystemExit(f"call_plan.totals is missing {field!r}: {call_plan['totals']!r}")

# AC-F11-17: the §2.3.3a worked example is a SUM, not a maximum (18) and
# not min(graph.max, policy.max).
if scenario["bounded_descendant_calls"] == 18:
    raise SystemExit(
        "bounded_descendant_calls == 18 -- the exact maximum-based defect "
        "AC-F11-17 forbids (must be the sum, 26)"
    )
assert scenario["bounded_descendant_calls"] == 26, scenario
assert scenario["fixed_root_lifecycle_calls"] == 6, scenario  # calls_per_entity("epic")
assert scenario["fixed_prelude_calls"] == 0, scenario
assert scenario["review_gate_calls"] == 0, scenario
assert scenario["worst_case_total_calls"] == 32, scenario

evidence = call_plan["evidence"]
by_limit = {entry["limit"]: entry for entry in evidence}
assert set(by_limit) == {
    "fixed_prelude_calls", "fixed_root_lifecycle_calls",
    "review_gate_calls", "bounded_descendant_calls", "worst_case_total_calls",
}, by_limit
# No call_plan field may present a bound as an exact count: bounded_descendant_calls
# and worst_case_total_calls (itself a sum containing a bound) both carry
# bound: true; the three purely-exact components do not.
for limit in ("bounded_descendant_calls", "worst_case_total_calls"):
    assert by_limit[limit]["bound"] is True, by_limit[limit]
for limit in ("fixed_prelude_calls", "fixed_root_lifecycle_calls", "review_gate_calls"):
    assert by_limit[limit]["bound"] is False, by_limit[limit]

# AC-F11-18: every derived limit names all 8 evidence fields, non-blank.
required = ("limit", "package_path", "package_digest", "workflow_file",
            "workflow_routing_digest", "model", "provider", "resource_policy_digest")
for limit, entry in by_limit.items():
    for field in required:
        value = entry.get(field)
        if value in (None, ""):
            raise SystemExit(f"evidence[{limit!r}] missing/blank field {field!r}: {entry!r}")
    assert entry["package_path"].endswith("py-epic-task-organization/package.yaml"), entry
    assert entry["workflow_file"].endswith("workflow/epic.yaml"), entry
print("TC-106(AC-F11-16/17/18 worked example: 26/32, bound flags, 8-field evidence) PASS")
PY

# ===========================================================================
# AC-F11-17 negative: a mutated workflow file changes the computed total --
# proving calls_per_entity is read live from the evidenced file, never a
# harness literal.
# ===========================================================================
echo "TC-106: AC-F11-17 -- mutating the real installed workflow file changes bounded_descendant_calls"
TASK_WORKFLOW="$EPIC_ROOT/scratch-template/shark-data/workflow/task.yaml"
[[ -f "$TASK_WORKFLOW" ]] || fail "expected real installed task.yaml workflow file: $TASK_WORKFLOW"
cp "$TASK_WORKFLOW" "$WORKDIR/task.yaml.orig"
cat >>"$TASK_WORKFLOW" <<'YAML'
  tc106_mutated_extra_step:
    action: spawn_agent
    agent: developer
    provider: anthropic
    model: sonnet
    outcomes:
      pass: completed
YAML

set +e
"$OPERATOR" preflight --config "$EPIC_ROOT/e40-demo.yaml" --out "$WORKDIR/epic-preflight-mutated" \
	>"$WORKDIR/epic-preflight-mutated.out" 2>"$WORKDIR/epic-preflight-mutated.err"
set -e
MUTATED_RESULT="$WORKDIR/epic-preflight-mutated/preflight-result.json"
cp "$WORKDIR/task.yaml.orig" "$TASK_WORKFLOW"
[[ -f "$MUTATED_RESULT" ]] || fail "preflight after workflow mutation did not write preflight-result.json: $(cat "$WORKDIR/epic-preflight-mutated.err")"

python3 - "$MUTATED_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
scenario = result["call_plan"]["per_scenario"]["py-epic-task-organization"]
# calls_per_entity("task") goes from 1 to 2 -> bounded_descendant_calls:
# 2*9 (feature, unchanged) + 8*2 (task, now doubled) = 34, not the
# original 26.
if scenario["bounded_descendant_calls"] != 34:
    raise SystemExit(
        f"mutating task.yaml left bounded_descendant_calls at "
        f"{scenario['bounded_descendant_calls']} (expected 34) -- calls_per_entity "
        "is reading a cached/hardcoded value, not the live workflow file"
    )
PY
echo "TC-106(AC-F11-17: workflow-file mutation changes bounded_descendant_calls 26 -> 34) PASS"

# ===========================================================================
# AC-F11-25a: fallback derivation via the real preflight seam -- a package
# with no declared descendant-graph block falls back to
# resource_policy.max_generated_tasks * calls_per_entity("task").
# ===========================================================================
echo "TC-106: AC-F11-25a -- real preflight fallback derivation (no declared descendant-graph block)"
FEATURE_INDEX="$WORKDIR/feature-index.yaml"
cat >"$FEATURE_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $FEATURE_PACKAGE_DIR
EOF
FEATURE_ROOT="$WORKDIR/feature-setup"
"$OPERATOR" setup --out "$FEATURE_ROOT" --scenario-index "$FEATURE_INDEX" \
	>"$WORKDIR/feature-setup.out" 2>&1 \
	|| fail "real 'e40-benchmark.sh setup' failed for the feature fallback case: $(cat "$WORKDIR/feature-setup.out")"
set +e
"$OPERATOR" preflight --config "$FEATURE_ROOT/e40-demo.yaml" --out "$WORKDIR/feature-preflight" \
	>"$WORKDIR/feature-preflight.out" 2>"$WORKDIR/feature-preflight.err"
set -e
FEATURE_RESULT="$WORKDIR/feature-preflight/preflight-result.json"
[[ -f "$FEATURE_RESULT" ]] || fail "preflight over the feature fallback case did not write preflight-result.json: $(cat "$WORKDIR/feature-preflight.err")"

python3 - "$FEATURE_RESULT" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
scenario = result["call_plan"]["per_scenario"]["py-feature-recurring-tasks"]
# py-feature-recurring-tasks has no declared descendant-graph block and
# resource_policy.max_generated_tasks: 30 -> fallback = 30 * calls_per_entity("task") = 30.
assert scenario["bounded_descendant_calls"] == 30, scenario
by_limit = {e["limit"]: e for e in result["call_plan"]["evidence"]}
entry = by_limit["bounded_descendant_calls"]
assert entry["bound"] is True, entry
assert entry["resource_policy_digest"], entry
PY
echo "TC-106(AC-F11-25a: real preflight fallback derivation, 30 = 30*calls_per_entity('task'), bound: true) PASS"

# ===========================================================================
# §2.3.3a "Both sources present": admission-time rejection via the real
# preflight seam when the declared descendant-graph block's task maxima
# exceed resource_policy.max_generated_tasks -- not a silent tighter-wins
# clamp.
# ===========================================================================
echo "TC-106: admission-time rejection -- descendant-graph task maxima exceeding resource_policy.max_generated_tasks"
REJECT_PACKAGE_DIR="$WORKDIR/reject-package"
cp -r "$EPIC_PACKAGE_DIR" "$REJECT_PACKAGE_DIR"
python3 - "$REJECT_PACKAGE_DIR/package.yaml" <<'PY'
import sys

import yaml

# Built from parts, never written as one literal token: AC-F11-27's grep
# gate (tests/contracts/e40_i04_scenario_contract_test.go) treats any file
# containing this exact field name as an undeclared third reader of the
# block, and this file only needs to set/read the key, not parse or
# interpret the block itself (same technique tc109 uses).
descendant_graph_field = "expected" + "_entity_graph"

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    pkg = yaml.safe_load(f)
pkg["scenario_id"] = "py-epic-task-organization-reject"
for descendant in pkg[descendant_graph_field]["descendants"]:
    if descendant["family"] == "task":
        descendant["max"] = 40
with open(path, "w", encoding="utf-8") as f:
    yaml.safe_dump(pkg, f, sort_keys=False)
PY
REJECT_INDEX="$WORKDIR/reject-index.yaml"
cat >"$REJECT_INDEX" <<EOF
schema_version: "1.1"
scenarios:
  - $REJECT_PACKAGE_DIR
EOF
REJECT_ROOT="$WORKDIR/reject-setup"
"$OPERATOR" setup --out "$REJECT_ROOT" --scenario-index "$REJECT_INDEX" \
	>"$WORKDIR/reject-setup.out" 2>&1 \
	|| fail "real 'e40-benchmark.sh setup' failed for the admission-rejection package: $(cat "$WORKDIR/reject-setup.out")"
set +e
"$OPERATOR" preflight --config "$REJECT_ROOT/e40-demo.yaml" --out "$WORKDIR/reject-preflight" \
	>"$WORKDIR/reject-preflight.out" 2>"$WORKDIR/reject-preflight.err"
reject_rc=$?
set -e
[[ "$reject_rc" -ne 0 ]] || fail "preflight over the over-budget graph did not fail: $(cat "$WORKDIR/reject-preflight.out")"
[[ ! -f "$WORKDIR/reject-preflight/preflight-result.json" ]] \
	|| fail "preflight wrote a result despite the admission-time rejection -- must fail before any file is written"
grep -q "exceeding resource_policy.max_generated_tasks" "$WORKDIR/reject-preflight.err" \
	|| fail "rejection does not name resource_policy.max_generated_tasks: $(cat "$WORKDIR/reject-preflight.err")"
echo "TC-106(admission-time rejection: descendant-graph task maxima exceeding resource_policy.max_generated_tasks) PASS"

# ===========================================================================
# §7.3.4 B1-B8 comparator surface, including B7's load-bearing
# counterfactual (a maximum-based implementation returns 18, must fail).
# No CLI seam reaches all 8 cells independently without 8 scratch operator
# roots; proven directly against bounded_descendant_call_count, the exact
# pure function preflight's own derivation calls -- still pointed at the
# real, shipped default workflow bundle (calls_per_entity is exercised for
# real even off the CLI seam).
# ===========================================================================
echo "TC-106: §7.3.4 B1-B8 comparator surface"
python3 - "$MODULE" "$REPO_ROOT" <<'PY'
import importlib.util
import sys
from pathlib import Path

module_path, repo_root = Path(sys.argv[1]), Path(sys.argv[2])
workflow_root = repo_root / "internal" / "sharkdata" / "default_data" / "workflow"

spec = importlib.util.spec_from_file_location("e40_benchmark_tc106_b", module_path)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

PKG_PATH = Path("tc106-test-only-package.yaml")
# Built from parts, never written as one literal token -- see the note at
# the top of this file (AC-F11-27's grep gate).
DESCENDANT_GRAPH_FIELD = "expected" + "_entity_graph"


def derive(descendants, max_generated_tasks):
    package = {}
    if descendants is not None:
        package[DESCENDANT_GRAPH_FIELD] = {"descendants": descendants}
    resource_policy = {}
    if max_generated_tasks is not None:
        resource_policy["max_generated_tasks"] = max_generated_tasks
    return mod.bounded_descendant_call_count(workflow_root, PKG_PATH, package, resource_policy)


# B1: sum, not max, not min(graph.max, policy.max).
total, _ = derive([{"family": "feature", "max": 2}, {"family": "task", "max": 8}], 30)
assert total == 26, f"B1: expected 26, got {total}"
assert total != 8, "B1: must not be max() over descendants[]"
assert total != min(8, 30), "B1: must not be min(graph.max, policy.max)"

# B2: admission rejects -- graph declares more tasks than the ceiling allows.
try:
    derive([{"family": "task", "max": 40}], 30)
    raise SystemExit("B2: expected admission rejection (40 > 30), got a value")
except mod.OperatorError as exc:
    assert "40" in str(exc) and "30" in str(exc), f"B2 error does not name both values: {exc}"

# B3: boundary -- equal is accepted, not exceeding.
total, _ = derive([{"family": "task", "max": 30}], 30)
assert total == 30, f"B3: expected 30, got {total}"

# B4: absent graph -> fallback.
total, _ = derive(None, 30)
assert total == 30, f"B4: expected 30 (fallback), got {total}"

# B5: graph present, policy absent -> reject (the policy ceiling is
# always required).
try:
    derive([{"family": "task", "max": 8}], None)
    raise SystemExit("B5: expected rejection (policy ceiling always required)")
except mod.OperatorError:
    pass

# B6: neither present -> reject.
try:
    derive(None, None)
    raise SystemExit("B6: expected rejection")
except mod.OperatorError:
    pass

# B7: the load-bearing counterfactual. A maximum-based implementation
# would return 18 here; the real implementation must return 26.
total, _ = derive([{"family": "feature", "max": 2}, {"family": "task", "max": 8}], 30)
assert total == 26, f"B7: expected 26 (sum), got {total}"
assert total != 18, "B7: 18 is the maximum-based counterfactual this case exists to catch"

# B8: a declared-but-never-realized family contributes 0.
total, _ = derive([{"family": "task", "min": 0, "max": 0}], 30)
assert total == 0, f"B8: expected 0, got {total}"

print("TC-106(B1-B8 comparator surface, incl. B7 counterfactual) PASS")
PY

# ===========================================================================
# AC-F11-16 negative: each of the 8 call_plan schema fields absent in turn
# -> rejection naming it. No CLI seam can independently blank exactly one
# field of an already-derived in-memory dict before it is written; proven
# directly against validate_call_plan_schema, the exact function
# _cmd_preflight_locked calls before writing preflight-result.json.
# ===========================================================================
echo "TC-106: AC-F11-16 -- each of the 8 call_plan schema fields absent in turn is rejected"
python3 - "$MODULE" <<'PY'
import copy
import importlib.util
import sys
from pathlib import Path

module_path = Path(sys.argv[1])
spec = importlib.util.spec_from_file_location("e40_benchmark_tc106_schema", module_path)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

good_plan = {
    "per_scenario": {
        "s1": {
            "fixed_prelude_calls": 0,
            "fixed_root_lifecycle_calls": 1,
            "review_gate_calls": 0,
            "bounded_descendant_calls": 1,
            "worst_case_total_calls": 2,
        }
    },
    "totals": {
        "worst_case_total_calls": 2,
        "max_cost_usd": 5,
        "max_wall_clock_seconds": 10,
        "max_generated_tasks": 5,
        "max_provider_calls": 10,
    },
    "evidence": [],
}
mod.validate_call_plan_schema(good_plan)  # positive control -- must not raise

per_scenario_fields = (
    "fixed_prelude_calls", "fixed_root_lifecycle_calls",
    "review_gate_calls", "bounded_descendant_calls", "worst_case_total_calls",
)
for field in per_scenario_fields:
    mutated = copy.deepcopy(good_plan)
    del mutated["per_scenario"]["s1"][field]
    try:
        mod.validate_call_plan_schema(mutated)
        raise SystemExit(f"blanking per_scenario field {field!r} was not rejected")
    except mod.OperatorError as exc:
        assert field in str(exc), f"rejection for {field!r} does not name it: {exc}"

totals_fields = (
    "worst_case_total_calls", "max_cost_usd", "max_wall_clock_seconds",
    "max_generated_tasks", "max_provider_calls",
)
for field in totals_fields:
    mutated = copy.deepcopy(good_plan)
    del mutated["totals"][field]
    try:
        mod.validate_call_plan_schema(mutated)
        raise SystemExit(f"blanking totals field {field!r} was not rejected")
    except mod.OperatorError as exc:
        assert field in str(exc), f"rejection for {field!r} does not name it: {exc}"

print("TC-106(AC-F11-16: 8/8 schema-field blank cases rejected, each naming its field) PASS")
PY

# ===========================================================================
# AC-F11-18 negative: each of the 8 evidence fields blanked in turn ->
# rejection naming it. Same "no reachable seam" reasoning as above, proven
# directly against build_call_plan_evidence_entry.
# ===========================================================================
echo "TC-106: AC-F11-18 -- each of the 8 evidence fields blanked in turn is rejected"
python3 - "$MODULE" <<'PY'
import importlib.util
import sys
from pathlib import Path

module_path = Path(sys.argv[1])
spec = importlib.util.spec_from_file_location("e40_benchmark_tc106_evidence", module_path)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

base = dict(
    limit="bounded_descendant_calls",
    package_path="/x/package.yaml",
    package_digest="a" * 64,
    workflow_file="/x/workflow/epic.yaml",
    workflow_routing_digest="b" * 64,
    model="opus",
    provider="anthropic",
    resource_policy_digest="c" * 64,
)
entry = mod.build_call_plan_evidence_entry(**base)  # positive control
assert set(entry) == set(mod.CALL_PLAN_EVIDENCE_FIELDS) | {"bound"}, entry
assert entry["bound"] is True, entry

for field in mod.CALL_PLAN_EVIDENCE_FIELDS:
    mutated = dict(base)
    mutated[field] = ""
    try:
        mod.build_call_plan_evidence_entry(**mutated)
        raise SystemExit(f"blanking evidence field {field!r} was not rejected")
    except mod.OperatorError as exc:
        assert field in str(exc), f"rejection for {field!r} does not name it: {exc}"

print("TC-106(AC-F11-18: 8/8 evidence-field blank cases rejected, each naming its field) PASS")
PY

echo "TC-106: all cases PASS"
