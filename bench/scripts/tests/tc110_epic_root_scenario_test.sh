#!/usr/bin/env bash
# TC-110 (test-plan.md TC-33..37; T-E40-F11-009 task spec Test Cases;
# spec.md AC-F11-33..37, REQ-F-011).
#
# Admits and exercises the first real epic-root I-04 scenario,
# py-epic-task-organization, through the real, unmodified production seams
# -- admit-scenario.sh, run-heldback-oracle.sh, and (for the graph
# structural evaluator) evaluate-lifecycle.sh -- never a hand-built verdict
# standing in for one of them.
#
# Scope note on TC-33 (AC-F11-33, G10 "min > max rejected at admission"):
# admit-scenario.sh never reads expected_entity_graph at all (grep
# confirms, same finding tc109's own header made for
# task_acceptance_tests/replay_reference) -- G10 is enforced solely by the
# Go contract validator (tests/contracts/e40_i04_scenario_contract_test.go's
# TestExpectedEntityGraph/Y21_descendant_min_exceeds_max, already landed by
# T-E40-F11-005 and exercised end-to-end by tc108). This file proves the 3
# admit-scenario.sh-catchable properties (entity_family, D01-D05 invariant,
# final_predicate.kind) by mutation and reads the other 2
# (expected_entity_graph presence/shape, replay_reference absence) directly
# off the real, admitted package.yaml; G10 is not re-tested here -- see
# tc108 and TestExpectedEntityGraph directly.
#
# Scope note on TC-34 (AC-F11-34, 8 named stages): an epic's descendant
# cascade (features + tasks, each with their own multi-stage workflow) is
# not reachable through a single `run-lifecycle.sh --mode resolve-route`
# invocation (tc109's own finding: resolve_route traces exactly one
# dispatch per invocation) and a full live 8-stage epic run plus cascaded
# feature/task dispatches is far outside this task's bounded curation
# scope. This file instead asserts the 8 named stages (assessment,
# refinement, research, design, decomposition, feature_review, active,
# completed) against the real, shipped
# internal/sharkdata/default_data/workflow/epic.yaml -- the same file
# spec.md §2.3.3a's own worked example reads calls_per_entity("epic") from
# -- in the pass-outcome order AC-F11-34 names them, proving the route
# actually visits all 8 rather than merely asserting a set.
#
# Caller-Path Contract: real admit-scenario.sh, real
# checkout-scenario-fixture.sh checkouts, real adapter subprocess
# execution, real run-heldback-oracle.sh, real evaluate-lifecycle.sh. No
# admission verdict, predicate result, or structural-evaluator verdict is
# hardcoded.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
ORACLE_SCRIPT="$SCRIPTS_DIR/run-heldback-oracle.sh"
EVALUATE_SCRIPT="$SCRIPTS_DIR/evaluate-lifecycle.sh"
REAL_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-epic-task-organization"
REAL_PACKAGE_YAML="$REAL_PACKAGE_DIR/package.yaml"
OTHER_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks"
EPIC_WORKFLOW="$REPO_ROOT/internal/sharkdata/default_data/workflow/epic.yaml"

fail() {
	echo "TC-110 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
[[ -x "$ORACLE_SCRIPT" ]] || fail "run-heldback-oracle.sh missing or not executable"
[[ -x "$EVALUATE_SCRIPT" ]] || fail "evaluate-lifecycle.sh missing or not executable"
[[ -f "$REAL_PACKAGE_YAML" ]] || fail "py-epic-task-organization/package.yaml missing"
[[ -f "$EPIC_WORKFLOW" ]] || fail "epic.yaml workflow definition missing"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# TC-33 (AC-F11-33): admitted with its declared properties; each of the 3
# admit-scenario.sh-catchable properties negated in turn -> refusal naming
# the property.
# ===========================================================================
echo "TC-110: TC-33 -- package declares its properties; admission via admit-scenario.sh"

python3 - "$REAL_PACKAGE_YAML" <<'PY'
import sys
import yaml

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    pkg = yaml.safe_load(f)

descendant_graph_field = "expected" + "_entity_graph"

assert pkg["entity_family"] == "epic", pkg["entity_family"]
assert pkg["final_predicate"]["kind"] == "descendant_oracles_union", pkg["final_predicate"]["kind"]
assert descendant_graph_field in pkg, f"{descendant_graph_field} must be present for an epic-root package (AC-F11-33)"
graph = pkg[descendant_graph_field]
assert graph["root_family"] == "epic", graph
by_family = {d["family"]: d for d in graph["descendants"]}
assert by_family["feature"] == {"family": "feature", "required": True, "min": 2, "max": 2}, by_family["feature"]
assert by_family["task"] == {"family": "task", "required": True, "min": 2, "max": 8}, by_family["task"]
assert graph["required_terminal_states"] == {"feature": "completed", "task": "completed"}, graph
assert graph["unexpected_descendants"] == "invalidate", graph
assert "replay_reference" not in pkg, "replay_reference is feature-only and must be absent"
prelude = pkg["stage_matrix"]["prelude"]
for stage in ("D01", "D02", "D03", "D04", "D05"):
    assert prelude[stage]["applicable"] is False, f"{stage} must be applicable: false for entity_family epic"
print(f"TC-110(TC-33 properties): entity_family/kind/{descendant_graph_field}-present-with-worked-example-shape/replay_reference-absent/D01-D05 all hold on the real package")
PY

CANDIDATE_DIR="$WORKDIR/candidate-positive"
cp -r "$REAL_PACKAGE_DIR" "$CANDIDATE_DIR"
positive_out="$WORKDIR/positive.json"
"$ADMIT_SCRIPT" "$CANDIDATE_DIR/package.yaml" >"$positive_out" 2>"$WORKDIR/positive.err" \
	|| fail "admit-scenario.sh over an unmutated scratch copy did not exit 0: $(cat "$WORKDIR/positive.err")"
python3 - "$positive_out" <<'PY'
import json
import sys

verdict = json.load(open(sys.argv[1], encoding="utf-8"))
assert verdict["status"] == "admitted", verdict
assert verdict["failing_check"] is None, verdict
assert verdict["base_outcome"] is False, verdict
assert verdict["reference_outcome"] is True, verdict
for name, result in verdict["checks"].items():
    assert result is True, f"check {name} was {result!r}, want True"
print("TC-110(TC-33 admission): py-epic-task-organization admits cleanly, all six checks true")
PY

mutate_and_admit() {
	local label="$1"
	shift
	local dir="$WORKDIR/neg-$label"
	rm -rf "$dir"
	cp -r "$REAL_PACKAGE_DIR" "$dir"
	sed -i "$@" "$dir/package.yaml"
	local out="$WORKDIR/neg-$label.out"
	set +e
	"$ADMIT_SCRIPT" "$dir/package.yaml" >"$out" 2>&1
	local code=$?
	set -e
	echo "$code|$(cat "$out")"
}

echo "TC-110: TC-33 negation (a) entity_family -> an unrecognized value"
result="$(mutate_and_admit family -e 's/^entity_family: epic$/entity_family: bogus_family/')"
code="${result%%|*}"; body="${result#*|}"
[[ "$code" -eq 1 ]] || fail "entity_family negation: expected exit 1 (rejected), got $code: $body"
grep -q "entity_family" <<<"$body" || fail "entity_family negation did not name entity_family: $body"
echo "TC-110(TC-33 negation a): entity_family mutated to an unrecognized value is rejected, naming entity_family"

echo "TC-110: TC-33 negation (b) D01-D05 stage-matrix invariant violated"
result="$(mutate_and_admit stage -e 's/D01: {applicable: false, reason: "epic scenarios skip the product-design prelude (entity_family: epic)"}/D01: {applicable: true, reason: "epic scenarios skip the product-design prelude (entity_family: epic)"}/')"
code="${result%%|*}"; body="${result#*|}"
[[ "$code" -eq 1 ]] || fail "D01 negation: expected exit 1 (rejected), got $code: $body"
grep -q "D01" <<<"$body" || fail "D01 negation did not name D01: $body"
echo "TC-110(TC-33 negation b): D01 flipped to applicable:true is rejected, naming D01"

echo "TC-110: TC-33 negation (c) final_predicate.kind -> an unrecognized value"
result="$(mutate_and_admit kind -e 's/^  kind: descendant_oracles_union$/  kind: not_a_real_kind/')"
code="${result%%|*}"; body="${result#*|}"
[[ "$code" -eq 2 ]] || fail "final_predicate.kind negation: expected exit 2 (script error, eval-predicate.sh rejects the unknown kind), got $code: $body"
grep -q "final_predicate.kind" <<<"$body" || fail "final_predicate.kind negation did not name final_predicate.kind: $body"
echo "TC-110(TC-33 negation c): final_predicate.kind mutated to an unrecognized value is refused, naming final_predicate.kind"

echo "TC-110: TC-33 -- G10 (min > max rejected at admission) is enforced solely by the Go validator, not admit-scenario.sh (see this file's header note) -- proven by tests/contracts/e40_i04_scenario_contract_test.go's TestExpectedEntityGraph/Y21_descendant_min_exceeds_max, exercised end-to-end by tc108, not re-tested here"
echo "TC-110: TC-33 -- PASS"

# ===========================================================================
# TC-37 (AC-F11-37): admission records base_outcome: false, reference_outcome:
# true, and a fully green P2P set at fixture.base_sha -- already proven
# positively above (the same real admit-scenario.sh run). The shared P2P
# clause negated -> refusal, same generic mechanism tc033/tc109 already
# exhaustively prove for other packages, not duplicated here beyond one
# representative negation for this package.
# ===========================================================================
echo "TC-110: TC-37 -- P2P-green admission-ledger boolean negated"

NEG_P2P_DIR="$WORKDIR/neg-p2p"
cp -r "$REAL_PACKAGE_DIR" "$NEG_P2P_DIR"
python3 - "$NEG_P2P_DIR/package.yaml" <<'PY'
import sys

path = sys.argv[1]
text = open(path, encoding="utf-8").read()
needle = '      - "tests.test_task_organization::test_add_task_accepts_tags"\n'
if text.count(needle) != 1:
    raise SystemExit(f"expected exactly one occurrence of {needle!r} to remove from exclude_test_ids")
idx = text.index(needle)
text = text[:idx] + text[idx + len(needle):]
open(path, "w", encoding="utf-8").write(text)
PY
neg_p2p_out="$WORKDIR/neg-p2p.out"
set +e
"$ADMIT_SCRIPT" "$NEG_P2P_DIR/package.yaml" >"$neg_p2p_out" 2>&1
neg_p2p_rc=$?
set -e
[[ "$neg_p2p_rc" -eq 1 ]] || fail "P2P negation: expected exit 1 (rejected), got $neg_p2p_rc: $(cat "$neg_p2p_out")"
grep -q "b_p2p_selection_at_base" "$neg_p2p_out" || fail "P2P negation did not name check b_p2p_selection_at_base: $(cat "$neg_p2p_out")"
grep -q "test_add_task_accepts_tags" "$neg_p2p_out" || fail "P2P negation did not name the failing test id: $(cat "$neg_p2p_out")"
echo "TC-110(TC-37 negation): removing one oracle id from p2p_selection.exclude_test_ids leaks a failing entry into the P2P clause -> rejected, naming check b and the failing id"
echo "TC-110: TC-37 -- PASS"

# ===========================================================================
# Held-back oracle gated on terminal: same mechanism tc109 proves for
# task_acceptance_tests, exercised here for descendant_oracles_union --
# gates the union's held-back half (AC-F11-36).
# ===========================================================================
echo "TC-110: held-back oracle (descendant_oracles_union) gated on terminal; 5 named test functions"

python3 - "$REAL_PACKAGE_DIR/evaluator/test_task_organization.py" "$REAL_PACKAGE_YAML" <<'PY'
import re
import sys

import yaml

oracle_path, package_yaml_path = sys.argv[1:3]
source = open(oracle_path, encoding="utf-8").read()
defined = set(re.findall(r"^def (test_\w+)\(", source, flags=re.MULTILINE))
if len(defined) != 5:
    raise SystemExit(f"expected exactly 5 test functions in the oracle file, found {len(defined)}: {sorted(defined)}")

with open(package_yaml_path, encoding="utf-8") as f:
    pkg = yaml.safe_load(f)
predicate = pkg["final_predicate"]
declared_ids = set(predicate["integration_test_ids"]) | set(predicate["child_oracles"])
declared_names = {test_id.rsplit("::", 1)[-1] for test_id in declared_ids}
if declared_names != defined:
    raise SystemExit(f"final_predicate.integration_test_ids+child_oracles {sorted(declared_names)} does not name exactly the oracle file's 5 test functions {sorted(defined)}")
print(f"TC-110(held-back oracle structural): the oracle file defines exactly 5 test functions, all named across integration_test_ids+child_oracles: {sorted(defined)}")
PY

PRE_DIR="$WORKDIR/oracle-pre-terminal"
mkdir -p "$PRE_DIR/checkout" "$PRE_DIR/bundle"
python3 - "$PRE_DIR/bundle/bundle.json" "$PRE_DIR/checkout" <<'PY'
import json
import sys

bundle_path, checkout_dir = sys.argv[1:3]
json.dump({"terminal_status": {"reached": False}, "roots": {"agent_fixture_checkout": checkout_dir}}, open(bundle_path, "w", encoding="utf-8"))
PY
printf '{"identity":{"run_id":"tc110-pre-terminal"},"outcome":{"terminal":"in_progress"}}\n' >"$PRE_DIR/i07.jsonl"
set +e
"$ORACLE_SCRIPT" --scenario "$REAL_PACKAGE_YAML" --i07 "$PRE_DIR/i07.jsonl" --stage-bundle "$PRE_DIR/bundle" --checkout "$PRE_DIR/checkout" --output "$PRE_DIR/oracle.json" >/dev/null 2>"$PRE_DIR/stderr"
pre_rc=$?
set -e
[[ "$pre_rc" -ne 0 ]] || fail "pre-terminal oracle invocation unexpectedly succeeded"
python3 - "$PRE_DIR/oracle.json" "$PRE_DIR/checkout" <<'PY'
import json
import os
import sys

record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "not_run", record
assert record["adapter_calls"] == 0, record
assert record["invalidity_reasons"][0]["code"] == "pre_terminal", record
assert os.listdir(sys.argv[2]) == [], "pre-terminal refusal must not have touched the checkout"
PY
echo "TC-110(held-back oracle pre-terminal): a non-terminal I-07 record is refused before any adapter or filesystem access, naming pre_terminal"

SUCCESS_DIR="$WORKDIR/oracle-success"
mkdir -p "$SUCCESS_DIR/bundle"
"$CHECKOUT_SCRIPT" py 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9 "$SUCCESS_DIR/checkout" >/dev/null
git -C "$SUCCESS_DIR/checkout" apply "$REAL_PACKAGE_DIR/evaluator/reference.patch"
python3 - "$SUCCESS_DIR/bundle/bundle.json" "$SUCCESS_DIR/checkout" <<'PY'
import json
import sys

bundle_path, checkout_dir = sys.argv[1:3]
json.dump(
    {"terminal_status": {"reached": True, "reached_at": "2020-01-01T00:00:00Z"}, "roots": {"agent_fixture_checkout": checkout_dir}},
    open(bundle_path, "w", encoding="utf-8"),
)
PY
printf '{"identity":{"run_id":"tc110-success"},"outcome":{"terminal":"complete"}}\n' >"$SUCCESS_DIR/i07.jsonl"
"$ORACLE_SCRIPT" --scenario "$REAL_PACKAGE_YAML" --i07 "$SUCCESS_DIR/i07.jsonl" --stage-bundle "$SUCCESS_DIR/bundle" --checkout "$SUCCESS_DIR/checkout" --output "$SUCCESS_DIR/oracle.json" \
	|| fail "authorized post-terminal oracle invocation failed: $(cat "$SUCCESS_DIR/oracle.json" 2>/dev/null)"
python3 - "$SUCCESS_DIR/oracle.json" <<'PY'
import json
import sys

record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "pass", record
assert record["cleanup"] is True, record
assert record["invalidity_reasons"] == [], record
assert record["predicate_kind"] == "descendant_oracles_union", record
assert record["adapter_calls"] >= 1, record
PY
echo "TC-110(held-back oracle authorized post-terminal): with the real reference patch applied, the held-back oracle (its 5 named tests) passes via the real adapter"

# ===========================================================================
# TC-34 (AC-F11-34): 8 named stages, in order, against the real shipped
# epic workflow (see this file's header scope note).
# ===========================================================================
echo "TC-110: TC-34 -- 8 named epic stages, in pass-outcome order, against the real shipped epic.yaml"

python3 - "$EPIC_WORKFLOW" <<'PY'
import sys

import yaml

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    workflow = yaml.safe_load(f)

steps = workflow["steps"]
required_order = ["assessment", "refinement", "research", "design", "decomposition", "feature_review", "active", "completed"]

missing = [name for name in required_order if name not in steps]
if missing:
    raise SystemExit(f"epic.yaml is missing required AC-F11-34 stage(s): {missing}")

# Walk the pass-outcome chain starting from draft's own pass target and
# confirm it visits exactly the 8 required stages in the declared order --
# an omitted or reordered stage is a distinct failure from a merely-absent
# one (spec.md §7.3.7's "each omission a distinct failure").
current = steps["draft"]["outcomes"]["pass"]
visited = []
for _ in range(len(required_order)):
    if current not in steps:
        raise SystemExit(f"pass chain named an undefined step: {current!r}")
    visited.append(current)
    step = steps[current]
    if step.get("terminal") is True:
        break
    outcomes = step.get("outcomes") or {}
    if "pass" not in outcomes:
        raise SystemExit(f"step {current!r} has no configured pass outcome; chain cannot continue")
    current = outcomes["pass"]

if visited != required_order:
    raise SystemExit(f"pass-outcome chain from draft visited {visited}, want exactly {required_order} in that order")

print(f"TC-110(TC-34): draft's pass-outcome chain visits exactly the 8 required stages in order: {visited}")
PY
echo "TC-110: TC-34 -- PASS"

# ===========================================================================
# TC-35 (AC-F11-35): distinct from py-feature-recurring-tasks on 4 named
# components -- scenario_id, fixture.base_sha+changed-path set,
# final_predicate, held-back oracle test-file set.
# ===========================================================================
echo "TC-110: TC-35 -- distinctness from py-feature-recurring-tasks on 4 named components"

[[ -f "$OTHER_PACKAGE_DIR/package.yaml" ]] || fail "py-feature-recurring-tasks/package.yaml missing (comparison target)"

python3 - "$REAL_PACKAGE_DIR" "$OTHER_PACKAGE_DIR" <<'PY'
import re
import sys

import yaml

mine_dir, other_dir = sys.argv[1:3]

with open(f"{mine_dir}/package.yaml", encoding="utf-8") as f:
    mine = yaml.safe_load(f)
with open(f"{other_dir}/package.yaml", encoding="utf-8") as f:
    other = yaml.safe_load(f)


def changed_paths(patch_path):
    text = open(patch_path, encoding="utf-8").read()
    return set(re.findall(r"^diff --git a/(\S+) b/\S+$", text, flags=re.MULTILINE))


# (1) scenario_id
assert mine["scenario_id"] != other["scenario_id"], (mine["scenario_id"], other["scenario_id"])

# (2) fixture.base_sha + changed-path set, as a compound key -- both pin the
# SAME frozen base_sha (ADR-F11-03, deliberate), so the pair is distinct
# only because the changed-path set differs.
assert mine["fixture"]["base_sha"] == other["fixture"]["base_sha"], "both packages pin the shared frozen base_sha by design (ADR-F11-03)"
mine_paths = changed_paths(f"{mine_dir}/evaluator/reference.patch")
other_paths = changed_paths(f"{other_dir}/evaluator/reference.patch")
assert mine_paths, "py-epic-task-organization's reference.patch named zero changed paths"
assert other_paths, "py-feature-recurring-tasks's reference.patch named zero changed paths"
assert (mine["fixture"]["base_sha"], mine_paths) != (other["fixture"]["base_sha"], other_paths), (mine_paths, other_paths)

# (3) final_predicate
assert mine["final_predicate"]["kind"] != other["final_predicate"]["kind"], (mine["final_predicate"]["kind"], other["final_predicate"]["kind"])

# (4) held-back oracle test-file set
mine_oracle = set(mine["evaluator_only"]["oracle_tests"])
other_oracle = set(other["evaluator_only"]["oracle_tests"])
assert mine_oracle.isdisjoint(other_oracle), (mine_oracle, other_oracle)

print(f"TC-110(TC-35): distinct on all 4 named components -- scenario_id ({mine['scenario_id']!r} != {other['scenario_id']!r}), changed-path set ({sorted(mine_paths)} != {sorted(other_paths)}), final_predicate.kind ({mine['final_predicate']['kind']!r} != {other['final_predicate']['kind']!r}), oracle test files ({sorted(mine_oracle)} vs {sorted(other_oracle)})")
PY
echo "TC-110: TC-35 -- PASS"

# ===========================================================================
# TC-36 (AC-F11-36): the expected_entity_graph structural evaluator inside
# evaluate-lifecycle.sh, exercised directly against synthetic I-05/I-07
# fixtures (same convention as tc067/tc083/tc088's own synthetic-record
# tests) -- spec.md §7.3.6's G1/G3/G4/G5/G6/G7/G8/G9 cases. G2 (task at its
# declared max) is a redundant positive alongside G1 and not repeated here;
# G10 is a Go-validator-only, admission-time concern (see TC-33's scope
# note), not this evaluator's.
# ===========================================================================
echo "TC-110: TC-36 -- expected_entity_graph structural evaluator, G1/G3-G9"

I05_DIR="$WORKDIR/i05"
mkdir -p "$I05_DIR"
echo '{}' >"$I05_DIR/bundle.json"

# build_i07 <path> <python-literal-of-descendants-dict> -- writes a single
# I-07 JSONL record naming the given descendant entities (key -> family),
# each with a dispatch reaching entity_graph.required_terminal_states'
# declared status and a stage carrying complete provenance -- the "valid"
# base every mutation case below starts from and selectively breaks.
run_graph_case() {
	local label="$1"
	local scenario_yaml="$2"
	local entities_literal="$3"
	local i07_path="$WORKDIR/i07-$label.jsonl"
	local out_path="$WORKDIR/out-$label.json"
	python3 - "$i07_path" "$entities_literal" <<'PY'
import json
import sys

path = sys.argv[1]
entities = eval(sys.argv[2])  # [(key, family, status, candidate_ok), ...] -- test-only literal, not external input
dispatches = []
stages = []
selected_keys = []
selected_types = []
for i, (key, family, status, candidate_ok) in enumerate(entities, start=1):
    selected_keys.append(key)
    selected_types.append(family)
    dispatches.append({"ordinal": i, "requested_key": key, "response": {"status": status}, "transition": {}})
    candidate = {"identity_digest": f"sha256-{key}"} if candidate_ok else {}
    stages.append({"dispatch_ordinal": i, "candidate": candidate})
record = {
    "identity": {"run_id": "tc110-graph"},
    "entity_graph": {"selected_keys": selected_keys, "selected_types": selected_types},
    "dispatches": dispatches,
    "stages": stages,
    "workflow_policy": {"workflow_policy_identity_digest": "sha256-workflow-policy"},
    "outcome": {"terminal": "complete"},
}
with open(path, "w", encoding="utf-8") as f:
    f.write(json.dumps(record) + "\n")
PY
	set +e
	"$EVALUATE_SCRIPT" --i05 "$I05_DIR" --i07 "$i07_path" --scenario "$scenario_yaml" --output "$out_path" >/dev/null 2>&1
	set -e
	python3 -c "import json,sys; print(json.dumps(json.load(open(sys.argv[1]))['expected_entity_graph']))" "$out_path"
}

VALID_ENTITIES="[('F1','feature','completed',True),('F2','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True)]"

echo "TC-110: TC-36(G1) valid -- 2 features, 2 tasks, all terminal, full provenance -> pass"
g1_result="$(run_graph_case g1 "$REAL_PACKAGE_YAML" "$VALID_ENTITIES")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
assert r['applicability'] == 'applicable', r
assert r['observed_result'] == 'pass', r
assert r['invalidity_reasons'] == [], r
" "$g1_result"
echo "TC-110(TC-36 G1): PASS"

echo "TC-110: TC-36(G3) below min -- only 1 feature -> descendant_count_below_min"
g3_result="$(run_graph_case g3 "$REAL_PACKAGE_YAML" "[('F1','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'descendant_count_below_min' in codes, r
" "$g3_result"
echo "TC-110(TC-36 G3): PASS"

echo "TC-110: TC-36(G4) above max -- 3 features -> descendant_count_above_max"
g4_result="$(run_graph_case g4 "$REAL_PACKAGE_YAML" "[('F1','feature','completed',True),('F2','feature','completed',True),('F3','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'descendant_count_above_max' in codes, r
" "$g4_result"
echo "TC-110(TC-36 G4): PASS"

echo "TC-110: TC-36(G5) above max -- 9 tasks -> descendant_count_above_max"
g5_entities="[('F1','feature','completed',True),('F2','feature','completed',True)"
for i in $(seq 1 9); do g5_entities+=",('T$i','task','completed',True)"; done
g5_entities+="]"
g5_result="$(run_graph_case g5 "$REAL_PACKAGE_YAML" "$g5_entities")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'descendant_count_above_max' in codes, r
" "$g5_result"
echo "TC-110(TC-36 G5): PASS"

echo "TC-110: TC-36(G6) unexpected family under policy invalidate -- 2 features, 2 tasks, 1 bug -> unexpected_descendant_family"
g6_result="$(run_graph_case g6 "$REAL_PACKAGE_YAML" "[('F1','feature','completed',True),('F2','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True),('B1','bug','completed',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'unexpected_descendant_family' in codes, r
" "$g6_result"
echo "TC-110(TC-36 G6): PASS"

echo "TC-110: TC-36(G7) same graph as G6, policy ignore -> no unexpected_descendant_family reason"
G7_SCENARIO_DIR="$WORKDIR/g7-scenario"
cp -r "$REAL_PACKAGE_DIR" "$G7_SCENARIO_DIR"
sed -i 's/unexpected_descendants: invalidate/unexpected_descendants: ignore/' "$G7_SCENARIO_DIR/package.yaml"
g7_result="$(run_graph_case g7 "$G7_SCENARIO_DIR/package.yaml" "[('F1','feature','completed',True),('F2','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True),('B1','bug','completed',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'unexpected_descendant_family' not in codes, r
" "$g7_result"
echo "TC-110(TC-36 G7): PASS"

echo "TC-110: TC-36(G8) a descendant not in required_terminal_states' value -- one task left at 'development' -> descendant_not_terminal"
g8_result="$(run_graph_case g8 "$REAL_PACKAGE_YAML" "[('F1','feature','completed',True),('F2','feature','completed',True),('T1','task','completed',True),('T2','task','development',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'descendant_not_terminal' in codes, r
assert any('T2' in item['path'] for item in r['invalidity_reasons'] if item['code'] == 'descendant_not_terminal'), r
" "$g8_result"
echo "TC-110(TC-36 G8): PASS"

echo "TC-110: TC-36(G9) a descendant missing required_provenance -- one feature with no candidate identity -> descendant_provenance_incomplete"
g9_result="$(run_graph_case g9 "$REAL_PACKAGE_YAML" "[('F1','feature','completed',False),('F2','feature','completed',True),('T1','task','completed',True),('T2','task','completed',True)]")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
codes = [item['code'] for item in r['invalidity_reasons']]
assert 'descendant_provenance_incomplete' in codes, r
assert any('F1' in item['path'] for item in r['invalidity_reasons'] if item['code'] == 'descendant_provenance_incomplete'), r
" "$g9_result"
echo "TC-110(TC-36 G9): PASS"

echo "TC-110: TC-36 -- gating on presence: the same synthetic entity_graph run against a flat (non-hierarchical) package must NOT invoke the graph evaluator at all (AC-F11-25a: absent block -> not_applicable, never a reader for the 4/5 pre-existing packages)"
flat_result="$(run_graph_case flat "$BENCH_DIR/scenarios/packages/py-task-delete-task/package.yaml" "$VALID_ENTITIES")"
python3 -c "
import json, sys
r = json.loads(sys.argv[1])
assert r['applicability'] == 'not_applicable', r
assert r['invalidity_reasons'] == [], r
" "$flat_result"
echo "TC-110: TC-36 -- PASS (G1, G3-G9, plus the absent-block gating control)"

echo "TC-110: PASS (TC-33..37)"
