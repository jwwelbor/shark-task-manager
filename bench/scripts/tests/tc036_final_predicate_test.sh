#!/usr/bin/env bash
# TC-036 (test-plan.md AC test matrix; T-E40-F05-009 task spec Test Cases,
# AC-T4). Shared script: this task authors the `f2p_p2p` subtest, using
# T-E40-F05-008's repro test and reference patch (py-bug-due-date-boundary).
# T-E40-F05-010, -011, and -013 each append their own family's subtest
# function call at the bottom, following the same
# provision/capture/eval-predicate.sh/assert pattern the helpers below
# establish -- no new script, no twin harness.
#
# Drives the real bench/adapters/python/adapter.sh and the real
# bench/scripts/eval-predicate.sh against three real states of a checkout
# for the `f2p_p2p` kind (AC-014):
#   base              -- repro test fails, all real fixture P2P tests pass
#                         -> final_predicate.result == false
#   reference-applied -- reference.patch applied: repro passes, P2P entries
#                         still pass -> result == true
#   P2P-regression    -- reference.patch applied AND one P2P selection entry
#                         (outside the named f2p_test_ids) deliberately
#                         broken, repro still passes -> result == false,
#                         proving eval-predicate.sh's shared p2p_selection
#                         clause is actually enforced, not merely the named
#                         id
#
# Caller-Path Contract (test-plan.md TC-036): real `test`/`lint` capability
# JSON as produced by the real adapter against a real checkout; the P2P-
# regression state comes from a real source edit, never a hand-authored
# JSON document standing in for adapter output.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
EVAL_PREDICATE_SCRIPT="$SCRIPTS_DIR/eval-predicate.sh"
SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"
BUG_PACKAGE_YAML="$BENCH_DIR/scenarios/packages/py-bug-due-date-boundary/package.yaml"
CHANGE_PACKAGE_YAML="$BENCH_DIR/scenarios/packages/py-change-priority-scale/package.yaml"
FEATURE_PACKAGE_YAML="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/package.yaml"
TECHDEBT_PACKAGE_YAML="$BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation/package.yaml"

fail() {
	echo "TC-036 FAIL: $1" >&2
	exit 1
}

[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
[[ -x "$EVAL_PREDICATE_SCRIPT" ]] || fail "eval-predicate.sh missing or not executable"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"
[[ -f "$BUG_PACKAGE_YAML" ]] || fail "py-bug-due-date-boundary/package.yaml missing"
[[ -f "$CHANGE_PACKAGE_YAML" ]] || fail "py-change-priority-scale/package.yaml missing"
[[ -f "$FEATURE_PACKAGE_YAML" ]] || fail "py-feature-recurring-tasks/package.yaml missing"
[[ -f "$TECHDEBT_PACKAGE_YAML" ]] || fail "py-techdebt-consolidate-validation/package.yaml missing"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# -----------------------------------------------------------------------------
# Reusable helpers -- generic over package.yaml, not hardcoded to bug/python,
# so a future family's subtest (T-E40-F05-010/011/013) can reuse them as-is.
# -----------------------------------------------------------------------------

resolve_adapter_script() {
	local package_yaml="$1"
	python3 - "$package_yaml" "$SCENARIOS_YAML" "$BENCH_DIR" <<'PYEOF'
import sys
import yaml

package_yaml_path, scenarios_yaml_path, bench_dir = sys.argv[1:4]

with open(package_yaml_path) as f:
    package = yaml.safe_load(f)
adapter_name = package["adapter"]["name"]

with open(scenarios_yaml_path) as f:
    scenarios = yaml.safe_load(f)
entry = (scenarios.get("adapters") or {}).get(adapter_name)
if not entry or not entry.get("path"):
    sys.exit(f"adapter.name not registered in scenarios.yaml: {adapter_name!r}")

import os
print(os.path.join(bench_dir, "..", entry["path"], "adapter.sh"))
PYEOF
}

package_field() {
	# package_field <package_yaml> <dotted.path>
	local package_yaml="$1" dotted="$2"
	python3 - "$package_yaml" "$dotted" <<'PYEOF'
import json
import sys

import yaml

package_yaml_path, dotted = sys.argv[1:3]
with open(package_yaml_path) as f:
    data = yaml.safe_load(f)
node = data
for part in dotted.split("."):
    node = node[part]
print(node if isinstance(node, str) else json.dumps(node))
PYEOF
}

provision_checkout() {
	# provision_checkout <package_yaml> <dest_dir>
	local package_yaml="$1" dest_dir="$2"
	local fixture_id base_sha
	fixture_id="$(package_field "$package_yaml" fixture.fixture_id)"
	base_sha="$(package_field "$package_yaml" fixture.base_sha)"
	"$CHECKOUT_SCRIPT" "$fixture_id" "$base_sha" "$dest_dir" >/dev/null
}

inject_oracle_tests() {
	# inject_oracle_tests <adapter_script> <package_yaml> <checkout_dir>
	local adapter_script="$1" package_yaml="$2" checkout_dir="$3"
	local package_dir oracle_tests_json
	package_dir="$(dirname "$package_yaml")"
	oracle_tests_json="$(package_field "$package_yaml" evaluator_only.oracle_tests)"
	local -a files=()
	while IFS= read -r rel; do
		[[ -n "$rel" ]] || continue
		files+=("$package_dir/$rel")
	done < <(python3 -c "import json,sys; print('\n'.join(json.loads(sys.argv[1])))" "$oracle_tests_json")
	[[ ${#files[@]} -gt 0 ]] || return 0
	"$adapter_script" inject-tests --checkout "$checkout_dir" --files "${files[@]}" >/dev/null
}

# capture_state <adapter_script> <package_yaml> <checkout_dir> <tag>
# Writes $WORKDIR/test-<tag>.json (merged p2p_selection + named-id entries)
# and $WORKDIR/lint-<tag>.json, both from real adapter.sh invocations against
# the checkout's CURRENT on-disk state.
capture_state() {
	local adapter_script="$1" package_yaml="$2" checkout_dir="$3" tag="$4"
	python3 - "$adapter_script" "$package_yaml" "$checkout_dir" "$WORKDIR" "$tag" <<'PYEOF'
import json
import subprocess
import sys

import yaml

adapter_script, package_yaml_path, checkout_dir, workdir, tag = sys.argv[1:6]

with open(package_yaml_path) as f:
    package = yaml.safe_load(f)
predicate = package["final_predicate"]
kind = predicate["kind"]


def named_ids_for(kind, predicate):
    if kind == "f2p_p2p":
        return list(predicate.get("f2p_test_ids") or [])
    if kind == "acceptance_tests":
        return list(predicate.get("acceptance_test_ids") or [])
    if kind == "child_oracles_union":
        return list(predicate.get("integration_test_ids") or []) + list(predicate.get("child_oracles") or [])
    return []


def run_adapter(capability, extra_args=None):
    cmd = [adapter_script, capability, "--checkout", checkout_dir] + list(extra_args or [])
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        sys.exit(f"adapter.sh {capability} failed (exit {proc.returncode}): {proc.stderr.strip()}")
    return json.loads(proc.stdout)


p2p_selection = predicate.get("p2p_selection") or {}
include = list(p2p_selection.get("include") or [])
exclude_ids = list(p2p_selection.get("exclude_test_ids") or [])
test_args = ["--include"] + include
if exclude_ids:
    test_args += ["--exclude-id"] + exclude_ids
p2p_doc = run_adapter("test", test_args)

named_ids = named_ids_for(kind, predicate)
if named_ids:
    named_doc = run_adapter("test", ["--only-id"] + named_ids)
else:
    named_doc = {"entries": []}

seen = set()
entries = []
for doc in (p2p_doc, named_doc):
    for entry in doc.get("entries", []):
        if entry["id"] in seen:
            continue
        seen.add(entry["id"])
        entries.append(entry)

lint_doc = run_adapter("lint")

with open(f"{workdir}/test-{tag}.json", "w") as f:
    json.dump({"entries": entries}, f)
with open(f"{workdir}/lint-{tag}.json", "w") as f:
    json.dump(lint_doc, f)
PYEOF
}

# eval_and_assert <package_yaml> <tag> <expected_bool> <label>
eval_and_assert() {
	local package_yaml="$1" tag="$2" expected="$3" label="$4"
	local out
	out="$("$EVAL_PREDICATE_SCRIPT" "$package_yaml" "$WORKDIR/test-$tag.json" "$WORKDIR/lint-$tag.json")" \
		|| fail "$label: eval-predicate.sh failed"
	python3 - "$label" "$out" "$expected" <<'PYEOF'
import json
import sys

label, out, expected = sys.argv[1:4]
doc = json.loads(out)
want = expected == "true"
if doc["result"] != want:
    sys.exit(f"TC-036 FAIL: {label}: expected result={want}, got {doc}")
print(f"TC-036: {label}: result={doc['result']} as expected ({doc})")
PYEOF
}

# -----------------------------------------------------------------------------
# f2p_p2p subtest (T-E40-F05-009 own scope) -- py-bug-due-date-boundary,
# using T-E40-F05-008's repro test (evaluator/test_due_date_boundary.py) and
# reference patch (evaluator/reference.patch).
# -----------------------------------------------------------------------------
echo "TC-036: f2p_p2p (py-bug-due-date-boundary)"

kind="$(package_field "$BUG_PACKAGE_YAML" final_predicate.kind)"
[[ "$kind" == "f2p_p2p" ]] || fail "py-bug-due-date-boundary's final_predicate.kind is not f2p_p2p: $kind"

ADAPTER_SCRIPT="$(resolve_adapter_script "$BUG_PACKAGE_YAML")"
[[ -x "$ADAPTER_SCRIPT" ]] || fail "resolved adapter script missing or not executable: $ADAPTER_SCRIPT"

CHECKOUT_DIR="$WORKDIR/f2p_p2p-checkout"
provision_checkout "$BUG_PACKAGE_YAML" "$CHECKOUT_DIR"
inject_oracle_tests "$ADAPTER_SCRIPT" "$BUG_PACKAGE_YAML" "$CHECKOUT_DIR"

echo "TC-036: f2p_p2p - base state (repro fails, false)"
capture_state "$ADAPTER_SCRIPT" "$BUG_PACKAGE_YAML" "$CHECKOUT_DIR" base
eval_and_assert "$BUG_PACKAGE_YAML" base false "f2p_p2p base"

echo "TC-036: f2p_p2p - reference-applied state (repro passes + p2p_selection green, true)"
REFERENCE_PATCH="$(dirname "$BUG_PACKAGE_YAML")/$(package_field "$BUG_PACKAGE_YAML" evaluator_only.reference_solution)"
[[ -f "$REFERENCE_PATCH" ]] || fail "reference solution patch not found: $REFERENCE_PATCH"
git -C "$CHECKOUT_DIR" apply "$REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly"
capture_state "$ADAPTER_SCRIPT" "$BUG_PACKAGE_YAML" "$CHECKOUT_DIR" reference
eval_and_assert "$BUG_PACKAGE_YAML" reference true "f2p_p2p reference-applied"

echo "TC-036: f2p_p2p - P2P-regression state (repro passes, one p2p_selection entry broken, false)"
# taskmanager/manager.py's overdue_tasks() filter is a real P2P entry
# (tests.test_manager::test_overdue_tasks_excludes_completed_tasks /
# test_overdue_tasks_includes_incomplete_past_due_task, both in
# p2p_selection.include and NOT in exclude_test_ids) that never touches
# taskmanager/due_date.py, so this mutation cannot itself affect the named
# f2p repro test -- isolating the P2P clause from the named-id clause.
MANAGER_PY="$CHECKOUT_DIR/taskmanager/manager.py"
[[ -f "$MANAGER_PY" ]] || fail "taskmanager/manager.py not found in checkout"
grep -qF 'if t.due_date is not None and not t.is_done() and is_overdue(t.due_date, today)' "$MANAGER_PY" \
	|| fail "overdue_tasks() filter line did not match the expected shape -- mutation target moved"
sed -i 's/if t\.due_date is not None and not t\.is_done() and is_overdue(t\.due_date, today)/if t.due_date is not None and t.is_done() and is_overdue(t.due_date, today)/' "$MANAGER_PY"
capture_state "$ADAPTER_SCRIPT" "$BUG_PACKAGE_YAML" "$CHECKOUT_DIR" p2p_regression
eval_and_assert "$BUG_PACKAGE_YAML" p2p_regression false "f2p_p2p P2P-regression"

# Confirm the P2P-regression state's repro test itself is genuinely still
# passing -- i.e. the false result above comes from the shared p2p_selection
# clause, not from the named f2p id also having broken by coincidence.
python3 - "$WORKDIR/test-p2p_regression.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
repro_id = "tests.test_due_date_boundary::test_is_overdue_true_for_task_due_today"
outcome = next((e["outcome"] for e in doc["entries"] if e["id"] == repro_id), None)
if outcome != "pass":
    sys.exit(f"TC-036 FAIL: f2p_p2p P2P-regression: expected the named repro test to still pass, got {outcome!r}")
print("TC-036: f2p_p2p P2P-regression: named repro test still passes (false result is from the shared p2p_selection clause)")
PYEOF

# -----------------------------------------------------------------------------
# acceptance_tests subtest (T-E40-F05-010 own scope) --
# py-change-priority-scale, using T-E40-F05-010's three acceptance tests
# (evaluator/test_priority_scale_acceptance.py) and reference patch
# (evaluator/reference.patch).
#
# AC-T4 requires exactly four states, each from a fresh checkout so the
# mutations below never accumulate onto one another:
#   base                        -- all three acceptance tests fail (the
#                                   fixture still uses the legacy string
#                                   scale), all real fixture P2P tests pass
#                                   -> result == false
#   reference-applied           -- reference.patch applied: all three
#                                   acceptance tests pass, P2P entries still
#                                   pass -> result == true
#   one acceptance test failing -- reference.patch applied, then
#                                   LEGACY_PRIORITY_MAP["low"] is mutated to
#                                   an out-of-range value so only
#                                   test_legacy_records_converted_to_new_scale
#                                   fails while the other two acceptance
#                                   tests and P2P stay green -> result ==
#                                   false
#   P2P-regression               -- reference.patch applied AND one P2P
#                                   selection entry (outside the named
#                                   acceptance_test_ids) deliberately broken,
#                                   all three acceptance tests still pass
#                                   -> result == false, proving
#                                   eval-predicate.sh's shared p2p_selection
#                                   clause is actually enforced, not merely
#                                   the named ids
# -----------------------------------------------------------------------------
echo "TC-036: acceptance_tests (py-change-priority-scale)"

kind="$(package_field "$CHANGE_PACKAGE_YAML" final_predicate.kind)"
[[ "$kind" == "acceptance_tests" ]] || fail "py-change-priority-scale's final_predicate.kind is not acceptance_tests: $kind"

CHANGE_ADAPTER_SCRIPT="$(resolve_adapter_script "$CHANGE_PACKAGE_YAML")"
[[ -x "$CHANGE_ADAPTER_SCRIPT" ]] || fail "resolved adapter script missing or not executable: $CHANGE_ADAPTER_SCRIPT"
CHANGE_REFERENCE_PATCH="$(dirname "$CHANGE_PACKAGE_YAML")/$(package_field "$CHANGE_PACKAGE_YAML" evaluator_only.reference_solution)"
[[ -f "$CHANGE_REFERENCE_PATCH" ]] || fail "reference solution patch not found: $CHANGE_REFERENCE_PATCH"

# -- base ----------------------------------------------------------------------
echo "TC-036: acceptance_tests - base state (all three acceptance tests fail, false)"
BASE_CHECKOUT="$WORKDIR/acceptance_tests-checkout-base"
provision_checkout "$CHANGE_PACKAGE_YAML" "$BASE_CHECKOUT"
inject_oracle_tests "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$BASE_CHECKOUT"
capture_state "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$BASE_CHECKOUT" base
eval_and_assert "$CHANGE_PACKAGE_YAML" base false "acceptance_tests base"

# -- reference-applied -----------------------------------------------------------
echo "TC-036: acceptance_tests - reference-applied state (all three acceptance tests + p2p_selection green, true)"
REF_CHECKOUT="$WORKDIR/acceptance_tests-checkout-reference"
provision_checkout "$CHANGE_PACKAGE_YAML" "$REF_CHECKOUT"
inject_oracle_tests "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$REF_CHECKOUT"
git -C "$REF_CHECKOUT" apply "$CHANGE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly"
capture_state "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$REF_CHECKOUT" reference
eval_and_assert "$CHANGE_PACKAGE_YAML" reference true "acceptance_tests reference-applied"

# -- one acceptance test still failing post-reference -----------------------------
echo "TC-036: acceptance_tests - one acceptance test still failing post-reference (false)"
ONEFAIL_CHECKOUT="$WORKDIR/acceptance_tests-checkout-onefail"
provision_checkout "$CHANGE_PACKAGE_YAML" "$ONEFAIL_CHECKOUT"
inject_oracle_tests "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$ONEFAIL_CHECKOUT"
git -C "$ONEFAIL_CHECKOUT" apply "$CHANGE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (onefail checkout)"
PRIORITY_PY_ONEFAIL="$ONEFAIL_CHECKOUT/taskmanager/priority.py"
[[ -f "$PRIORITY_PY_ONEFAIL" ]] || fail "taskmanager/priority.py not found in checkout after applying reference patch"
grep -qF '"low": 4,' "$PRIORITY_PY_ONEFAIL" \
	|| fail "LEGACY_PRIORITY_MAP's \"low\" entry did not match the expected shape -- mutation target moved"
sed -i 's/"low": 4,/"low": 999,/' "$PRIORITY_PY_ONEFAIL"
capture_state "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$ONEFAIL_CHECKOUT" onefail
eval_and_assert "$CHANGE_PACKAGE_YAML" onefail false "acceptance_tests one-test-failing"

# Confirm the mutation broke exactly the intended acceptance test and left
# the other two (plus P2P) green -- i.e. the false result above is a genuine
# single-test failure, not an accidental collection-wide break.
python3 - "$WORKDIR/test-onefail.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
by_id = {e["id"]: e["outcome"] for e in doc["entries"]}
broken = "tests.test_priority_scale_acceptance::test_legacy_records_converted_to_new_scale"
still_green = [
    "tests.test_priority_scale_acceptance::test_new_scale_labels_and_range",
    "tests.test_priority_scale_acceptance::test_validate_priority_rejects_out_of_range",
]
if by_id.get(broken) == "pass":
    sys.exit(f"TC-036 FAIL: acceptance_tests one-test-failing: expected {broken} to fail, got pass")
bad = {tid: by_id.get(tid) for tid in still_green if by_id.get(tid) != "pass"}
if bad:
    sys.exit(f"TC-036 FAIL: acceptance_tests one-test-failing: expected the other two acceptance tests to still pass, got {bad}")
print("TC-036: acceptance_tests one-test-failing: exactly the mutated acceptance test failed, the other two still pass")
PYEOF

# -- P2P-regression --------------------------------------------------------------
echo "TC-036: acceptance_tests - P2P-regression state (all three acceptance tests pass, one p2p_selection entry broken, false)"
P2PREG_CHECKOUT="$WORKDIR/acceptance_tests-checkout-p2pregression"
provision_checkout "$CHANGE_PACKAGE_YAML" "$P2PREG_CHECKOUT"
inject_oracle_tests "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$P2PREG_CHECKOUT"
git -C "$P2PREG_CHECKOUT" apply "$CHANGE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (p2p-regression checkout)"
MANAGER_PY_P2P="$P2PREG_CHECKOUT/taskmanager/manager.py"
[[ -f "$MANAGER_PY_P2P" ]] || fail "taskmanager/manager.py not found in checkout after applying reference patch"
grep -qF 'if t.due_date is not None and not t.is_done() and is_overdue(t.due_date, today)' "$MANAGER_PY_P2P" \
	|| fail "overdue_tasks() filter line did not match the expected shape -- mutation target moved"
sed -i 's/if t\.due_date is not None and not t\.is_done() and is_overdue(t\.due_date, today)/if t.due_date is not None and t.is_done() and is_overdue(t.due_date, today)/' "$MANAGER_PY_P2P"
capture_state "$CHANGE_ADAPTER_SCRIPT" "$CHANGE_PACKAGE_YAML" "$P2PREG_CHECKOUT" p2pregression
eval_and_assert "$CHANGE_PACKAGE_YAML" p2pregression false "acceptance_tests P2P-regression"

# Confirm every acceptance test is genuinely still passing under this
# mutation -- i.e. the false result above comes from the shared
# p2p_selection clause alone, not from a named id also breaking.
python3 - "$WORKDIR/test-p2pregression.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
named_ids = [
    "tests.test_priority_scale_acceptance::test_new_scale_labels_and_range",
    "tests.test_priority_scale_acceptance::test_legacy_records_converted_to_new_scale",
    "tests.test_priority_scale_acceptance::test_validate_priority_rejects_out_of_range",
]
by_id = {e["id"]: e["outcome"] for e in doc["entries"]}
bad = {nid: by_id.get(nid) for nid in named_ids if by_id.get(nid) != "pass"}
if bad:
    sys.exit(f"TC-036 FAIL: acceptance_tests P2P-regression: expected every named acceptance test to still pass, got {bad}")
print("TC-036: acceptance_tests P2P-regression: every acceptance test still passes (false result is from the shared p2p_selection clause)")
PYEOF

# -----------------------------------------------------------------------------
# child_oracles_union subtest (T-E40-F05-013 own scope) --
# py-feature-recurring-tasks, using T-E40-F05-013's integration test plus
# child oracles (evaluator/test_recurring.py) and reference patch
# (evaluator/reference.patch).
#
# AC-T5 requires exactly four states, each from a fresh checkout so the
# mutations below never accumulate onto one another:
#   all green          -- reference.patch applied, integration test and all
#                          three child oracles pass -> result == true
#   child-oracle-failed -- reference.patch applied, then the "weekly"
#                          recurrence interval is mutated 7 -> 8 days: the
#                          exact-offset child oracle
#                          (test_generate_next_occurrence_advances_by_rule_interval)
#                          fails, but the integration test's looser 14-day
#                          window still contains the (wrongly-dated)
#                          occurrence, and the other two child oracles never
#                          touch the interval calculation -> result == false
#   integration-failed  -- reference.patch applied, then complete()'s call to
#                          generate_next_occurrence() is disabled: the
#                          integration test (which relies on that wiring)
#                          fails, but every child oracle calls its own
#                          capability directly (add_task's recurrence_rule
#                          field, generate_next_occurrence() called without
#                          going through complete(), upcoming_occurrences()
#                          on manually-constructed tasks) and is unaffected
#                          -> result == false
#   P2P-regression      -- reference.patch applied, integration test and all
#                          child oracles still pass, but one unrelated
#                          p2p_selection entry (manager.py's overdue_tasks()
#                          filter, the same real P2P entry
#                          py-bug-due-date-boundary's own P2P-regression
#                          state mutates) is deliberately broken -> result ==
#                          false, proving the shared p2p_selection clause is
#                          enforced independently of this kind's own named
#                          ids, not only f2p_p2p's
# -----------------------------------------------------------------------------
echo "TC-036: child_oracles_union (py-feature-recurring-tasks)"

kind="$(package_field "$FEATURE_PACKAGE_YAML" final_predicate.kind)"
[[ "$kind" == "child_oracles_union" ]] || fail "py-feature-recurring-tasks's final_predicate.kind is not child_oracles_union: $kind"

FEATURE_ADAPTER_SCRIPT="$(resolve_adapter_script "$FEATURE_PACKAGE_YAML")"
[[ -x "$FEATURE_ADAPTER_SCRIPT" ]] || fail "resolved adapter script missing or not executable: $FEATURE_ADAPTER_SCRIPT"

FEATURE_REFERENCE_PATCH="$(dirname "$FEATURE_PACKAGE_YAML")/$(package_field "$FEATURE_PACKAGE_YAML" evaluator_only.reference_solution)"
[[ -f "$FEATURE_REFERENCE_PATCH" ]] || fail "reference solution patch not found: $FEATURE_REFERENCE_PATCH"

# -- all green --------------------------------------------------------------
echo "TC-036: child_oracles_union - all green (integration test + child oracles pass, true)"
ALLGREEN_CHECKOUT="$WORKDIR/child_oracles_union-checkout-allgreen"
provision_checkout "$FEATURE_PACKAGE_YAML" "$ALLGREEN_CHECKOUT"
inject_oracle_tests "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$ALLGREEN_CHECKOUT"
git -C "$ALLGREEN_CHECKOUT" apply "$FEATURE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (all-green checkout)"
capture_state "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$ALLGREEN_CHECKOUT" allgreen
eval_and_assert "$FEATURE_PACKAGE_YAML" allgreen true "child_oracles_union all-green"

# -- child-oracle-failed ------------------------------------------------------
echo "TC-036: child_oracles_union - child-oracle-failed (weekly interval off by one day, false)"
COFAIL_CHECKOUT="$WORKDIR/child_oracles_union-checkout-cofail"
provision_checkout "$FEATURE_PACKAGE_YAML" "$COFAIL_CHECKOUT"
inject_oracle_tests "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$COFAIL_CHECKOUT"
git -C "$COFAIL_CHECKOUT" apply "$FEATURE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (cofail checkout)"
RECURRENCE_PY="$COFAIL_CHECKOUT/taskmanager/recurrence.py"
[[ -f "$RECURRENCE_PY" ]] || fail "taskmanager/recurrence.py not found in checkout after applying reference patch"
grep -qF '"weekly": timedelta(days=7),' "$RECURRENCE_PY" \
	|| fail "recurrence.py's weekly interval did not match the expected shape -- mutation target moved"
sed -i 's/"weekly": timedelta(days=7),/"weekly": timedelta(days=8),/' "$RECURRENCE_PY"
capture_state "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$COFAIL_CHECKOUT" cofail
eval_and_assert "$FEATURE_PACKAGE_YAML" cofail false "child_oracles_union child-oracle-failed"

# Confirm the integration test itself is genuinely still passing under this
# mutation -- i.e. the false result above is carried by the one broken child
# oracle, not by the whole union collapsing.
python3 - "$WORKDIR/test-cofail.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
integration_id = "tests.test_recurring::test_completing_recurring_task_schedules_next_occurrence_into_upcoming_list"
outcome = next((e["outcome"] for e in doc["entries"] if e["id"] == integration_id), None)
if outcome != "pass":
    sys.exit(f"TC-036 FAIL: child_oracles_union child-oracle-failed: expected the integration test to still pass, got {outcome!r}")
print("TC-036: child_oracles_union child-oracle-failed: integration test still passes (false result is from the one broken child oracle)")
PYEOF

# -- integration-failed -------------------------------------------------------
echo "TC-036: child_oracles_union - integration-failed (complete() wiring disabled, child oracles green, false)"
INTFAIL_CHECKOUT="$WORKDIR/child_oracles_union-checkout-intfail"
provision_checkout "$FEATURE_PACKAGE_YAML" "$INTFAIL_CHECKOUT"
inject_oracle_tests "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$INTFAIL_CHECKOUT"
git -C "$INTFAIL_CHECKOUT" apply "$FEATURE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (intfail checkout)"
MANAGER_PY_INTFAIL="$INTFAIL_CHECKOUT/taskmanager/manager.py"
[[ -f "$MANAGER_PY_INTFAIL" ]] || fail "taskmanager/manager.py not found in checkout after applying reference patch"
grep -qF '            self.generate_next_occurrence(task)' "$MANAGER_PY_INTFAIL" \
	|| fail "complete()'s generate_next_occurrence() call did not match the expected shape -- mutation target moved"
sed -i 's/            self\.generate_next_occurrence(task)/            pass  # TC-036 mutation: wiring intentionally disabled/' "$MANAGER_PY_INTFAIL"
capture_state "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$INTFAIL_CHECKOUT" intfail
eval_and_assert "$FEATURE_PACKAGE_YAML" intfail false "child_oracles_union integration-failed"

# Confirm every child oracle is genuinely still passing under this mutation
# -- i.e. the false result above is carried by the integration test alone.
python3 - "$WORKDIR/test-intfail.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
child_oracle_ids = [
    "tests.test_recurring::test_add_task_accepts_recurrence_rule",
    "tests.test_recurring::test_generate_next_occurrence_advances_by_rule_interval",
    "tests.test_recurring::test_upcoming_occurrences_filters_to_window_and_excludes_done",
]
by_id = {e["id"]: e["outcome"] for e in doc["entries"]}
bad = {cid: by_id.get(cid) for cid in child_oracle_ids if by_id.get(cid) != "pass"}
if bad:
    sys.exit(f"TC-036 FAIL: child_oracles_union integration-failed: expected every child oracle to still pass, got {bad}")
print("TC-036: child_oracles_union integration-failed: every child oracle still passes (false result is from the integration test alone)")
PYEOF

# -- P2P-regression -----------------------------------------------------------
echo "TC-036: child_oracles_union - P2P-regression (integration test + child oracles green, one p2p_selection entry broken, false)"
P2PREG_CHECKOUT="$WORKDIR/child_oracles_union-checkout-p2pregression"
provision_checkout "$FEATURE_PACKAGE_YAML" "$P2PREG_CHECKOUT"
inject_oracle_tests "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$P2PREG_CHECKOUT"
git -C "$P2PREG_CHECKOUT" apply "$FEATURE_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (p2p-regression checkout)"
MANAGER_PY_P2P="$P2PREG_CHECKOUT/taskmanager/manager.py"
[[ -f "$MANAGER_PY_P2P" ]] || fail "taskmanager/manager.py not found in checkout after applying reference patch"
grep -qF 'if t.due_date is not None and not t.is_done() and is_overdue(t.due_date, today)' "$MANAGER_PY_P2P" \
	|| fail "overdue_tasks() filter line did not match the expected shape -- mutation target moved"
sed -i 's/if t\.due_date is not None and not t\.is_done() and is_overdue(t\.due_date, today)/if t.due_date is not None and t.is_done() and is_overdue(t.due_date, today)/' "$MANAGER_PY_P2P"
capture_state "$FEATURE_ADAPTER_SCRIPT" "$FEATURE_PACKAGE_YAML" "$P2PREG_CHECKOUT" p2pregression
eval_and_assert "$FEATURE_PACKAGE_YAML" p2pregression false "child_oracles_union P2P-regression"

# Confirm the integration test and every child oracle are genuinely still
# passing under this mutation -- i.e. the false result above comes from the
# shared p2p_selection clause alone, not from the named ids also breaking.
python3 - "$WORKDIR/test-p2pregression.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
named_ids = [
    "tests.test_recurring::test_add_task_accepts_recurrence_rule",
    "tests.test_recurring::test_generate_next_occurrence_advances_by_rule_interval",
    "tests.test_recurring::test_upcoming_occurrences_filters_to_window_and_excludes_done",
    "tests.test_recurring::test_completing_recurring_task_schedules_next_occurrence_into_upcoming_list",
]
by_id = {e["id"]: e["outcome"] for e in doc["entries"]}
bad = {nid: by_id.get(nid) for nid in named_ids if by_id.get(nid) != "pass"}
if bad:
    sys.exit(f"TC-036 FAIL: child_oracles_union P2P-regression: expected every named id to still pass, got {bad}")
print("TC-036: child_oracles_union P2P-regression: integration test and every child oracle still pass (false result is from the shared p2p_selection clause)")
PYEOF

# -----------------------------------------------------------------------------
# p2p_plus_rule_drop subtest (T-E40-F05-011 own scope) -- py-techdebt-
# consolidate-validation, using this task's own fixture-py commit (three
# modules duplicating taskmanager/validation.py's non_empty_title inline,
# one of them -- taskmanager/bulk_import.py -- also shadowing its own
# import of non_empty_title, which ruff's F811 flags) and reference patch
# (evaluator/reference.patch, consolidating all three onto the shared
# validator).
#
# Three real states of a checkout (AC-T4; no sub-floor "over-satisfied"
# state is asserted, since max_remaining: 0 is already the floor):
#   count = 1 (base)               -- F811 fires once (bulk_import.py's
#                                      shadowed import), every p2p_selection
#                                      entry (including the injected
#                                      behavior-preserving oracle) passes
#                                      -> final_predicate.result == false
#                                      (rule count 1 > max_remaining 0)
#   count = 0 (reference-applied)  -- reference.patch applied: F811 count
#                                      drops to 0, p2p_selection stays green
#                                      -> result == true
#   P2P-regression (at count = 0)  -- reference.patch applied AND one
#                                      p2p_selection entry in an unrelated
#                                      module (taskmanager/priority.py)
#                                      deliberately broken -> result ==
#                                      false, proving eval-predicate.sh's
#                                      shared p2p_selection clause is
#                                      actually enforced even when the
#                                      lint clause alone is already
#                                      satisfied
# -----------------------------------------------------------------------------
echo "TC-036: p2p_plus_rule_drop (py-techdebt-consolidate-validation)"

kind="$(package_field "$TECHDEBT_PACKAGE_YAML" final_predicate.kind)"
[[ "$kind" == "p2p_plus_rule_drop" ]] || fail "py-techdebt-consolidate-validation's final_predicate.kind is not p2p_plus_rule_drop: $kind"

TECHDEBT_ADAPTER_SCRIPT="$(resolve_adapter_script "$TECHDEBT_PACKAGE_YAML")"
[[ -x "$TECHDEBT_ADAPTER_SCRIPT" ]] || fail "resolved adapter script missing or not executable: $TECHDEBT_ADAPTER_SCRIPT"
TECHDEBT_REFERENCE_PATCH="$(dirname "$TECHDEBT_PACKAGE_YAML")/$(package_field "$TECHDEBT_PACKAGE_YAML" evaluator_only.reference_solution)"
[[ -f "$TECHDEBT_REFERENCE_PATCH" ]] || fail "reference solution patch not found: $TECHDEBT_REFERENCE_PATCH"

# assert_lint_rule_count <tag> <expected_count> <label>
# Re-invokes the real eval-predicate.sh against the same test/lint JSON
# capture_state already wrote for <tag>, reading its own reported
# lint_rule_count.count -- the single named owner of this arithmetic
# (eval-predicate.sh header) -- rather than re-deriving a count here.
assert_lint_rule_count() {
	local tag="$1" expected="$2" label="$3"
	local out
	out="$("$EVAL_PREDICATE_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$WORKDIR/test-$tag.json" "$WORKDIR/lint-$tag.json")" \
		|| fail "$label: eval-predicate.sh failed"
	python3 - "$label" "$out" "$expected" <<'PYEOF'
import json
import sys

label, out, expected = sys.argv[1:4]
doc = json.loads(out)
got = doc["lint_rule_count"]["count"]
want = int(expected)
if got != want:
    sys.exit(f"TC-036 FAIL: {label}: expected lint_rule_count.count={want}, got {got} ({doc})")
print(f"TC-036: {label}: lint_rule_count.count={got} as expected")
PYEOF
}

# -- base (count = 1) ---------------------------------------------------------
echo "TC-036: p2p_plus_rule_drop - base state (F811 count = 1, false)"
BASE_CHECKOUT="$WORKDIR/p2p_plus_rule_drop-checkout-base"
provision_checkout "$TECHDEBT_PACKAGE_YAML" "$BASE_CHECKOUT"
inject_oracle_tests "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$BASE_CHECKOUT"
capture_state "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$BASE_CHECKOUT" base
eval_and_assert "$TECHDEBT_PACKAGE_YAML" base false "p2p_plus_rule_drop base"
assert_lint_rule_count base 1 "p2p_plus_rule_drop base"

# -- reference-applied (count = 0) --------------------------------------------
echo "TC-036: p2p_plus_rule_drop - reference-applied state (F811 count = 0, p2p_selection green, true)"
REF_CHECKOUT="$WORKDIR/p2p_plus_rule_drop-checkout-reference"
provision_checkout "$TECHDEBT_PACKAGE_YAML" "$REF_CHECKOUT"
inject_oracle_tests "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$REF_CHECKOUT"
git -C "$REF_CHECKOUT" apply "$TECHDEBT_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly"
capture_state "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$REF_CHECKOUT" reference
eval_and_assert "$TECHDEBT_PACKAGE_YAML" reference true "p2p_plus_rule_drop reference-applied"
assert_lint_rule_count reference 0 "p2p_plus_rule_drop reference-applied"

# -- P2P-regression (at count = 0) --------------------------------------------
echo "TC-036: p2p_plus_rule_drop - P2P-regression state (F811 count = 0, one p2p_selection entry broken, false)"
P2PREG_CHECKOUT="$WORKDIR/p2p_plus_rule_drop-checkout-p2pregression"
provision_checkout "$TECHDEBT_PACKAGE_YAML" "$P2PREG_CHECKOUT"
inject_oracle_tests "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$P2PREG_CHECKOUT"
git -C "$P2PREG_CHECKOUT" apply "$TECHDEBT_REFERENCE_PATCH" || fail "reference solution patch did not apply cleanly (p2p-regression checkout)"
PRIORITY_PY_P2P="$P2PREG_CHECKOUT/taskmanager/priority.py"
[[ -f "$PRIORITY_PY_P2P" ]] || fail "taskmanager/priority.py not found in checkout after applying reference patch"
# Adding "urgent" to the closed set of accepted priority levels breaks
# exactly test_priority.py::test_validate_priority_rejects_unknown_level
# (it stops raising for "urgent") -- a real, non-empty p2p_selection entry,
# with no cascade into any test relying on the "high"/"medium"/"low" values
# themselves (e.g. add_task's own "medium" default) and no effect on the
# F811 lint count this predicate's own rule-drop clause tracks.
grep -qF 'PRIORITY_LEVELS = ("high", "medium", "low")' "$PRIORITY_PY_P2P" \
	|| fail "PRIORITY_LEVELS tuple did not match the expected shape -- mutation target moved"
sed -i 's/PRIORITY_LEVELS = ("high", "medium", "low")/PRIORITY_LEVELS = ("high", "medium", "low", "urgent")/' "$PRIORITY_PY_P2P"
capture_state "$TECHDEBT_ADAPTER_SCRIPT" "$TECHDEBT_PACKAGE_YAML" "$P2PREG_CHECKOUT" p2pregression
eval_and_assert "$TECHDEBT_PACKAGE_YAML" p2pregression false "p2p_plus_rule_drop P2P-regression"
assert_lint_rule_count p2pregression 0 "p2p_plus_rule_drop P2P-regression"

# Confirm the mutation broke a real, unrelated test (proving the false
# result above comes from the shared p2p_selection clause, not from the
# F811 count having crept back up by coincidence).
python3 - "$WORKDIR/test-p2pregression.json" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
by_id = {e["id"]: e["outcome"] for e in doc["entries"]}
broken = sorted(tid for tid, outcome in by_id.items() if "test_priority" in tid and outcome != "pass")
if not broken:
    sys.exit("TC-036 FAIL: p2p_plus_rule_drop P2P-regression: expected at least one test_priority entry to fail, got none")
print(f"TC-036: p2p_plus_rule_drop P2P-regression: {broken} broken (false result is from the shared p2p_selection clause, lint clause alone is satisfied)")
PYEOF

echo "TC-036: PASS"
