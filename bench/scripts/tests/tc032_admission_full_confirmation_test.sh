#!/usr/bin/env bash
# TC-032 (test-plan.md AC test matrix; T-E40-F05-015 task spec Test Cases).
#
# Exercises AC-006 / AC-T2: bench/scripts/admit-scenario.sh over all four
# REQ-F-014 seed packages (one candidate per invocation -- admit-scenario.sh
# takes exactly one package.yaml, never a corpus-wide run), each against a
# real checkout-scenario-fixture.sh checkout of the frozen base_sha --
#   - every seed is admitted (status "admitted", failing_check null, every
#     one of the six named checks true)
#   - every seed records base_outcome: false and reference_outcome: true
#   - the four admitted scenario_ids cover entity_family
#     {bug, change_card, tech_debt, feature} exactly once each -- one seed
#     per family, no duplicate, no family missing
#
# "One admitted scenario per family" is this test's own cross-seed
# assertion (admit-scenario.sh itself only ever emits one verdict per
# invocation) -- built by iterating the four scenario_ids registered in
# bench/scenarios/scenarios.yaml, never a hardcoded id list, so a fifth
# registered scenario would be picked up automatically.
#
# Caller-Path Contract (test-plan.md TC-032): real git clone/checkout of the
# submodule per candidate (admit-scenario.sh provisions its own fresh
# checkout-scenario-fixture.sh checkout internally); real adapter subprocess
# execution for build/test/lint. No capability output is stubbed and no
# per-candidate verdict is hardcoded here.
#
# Runs against a scratch COPY of each package directory (mirroring TC-033's
# mk_candidate), never the live committed package.yaml -- admit-scenario.sh
# writes its admission: block in place on an admitted verdict, and this test
# must not mutate tracked files as a side effect of merely re-verifying them.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"

fail() {
	echo "TC-032 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# List of package.yaml directory names, relative to bench/scenarios/, in the
# order scenarios.yaml itself lists them -- never a hand-authored id list.
mapfile -t package_rel_dirs < <(python3 - "$SCENARIOS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for rel in data["scenarios"]:
    print(rel)
PYEOF
)
[[ "${#package_rel_dirs[@]}" -eq 4 ]] || fail "expected exactly 4 registered scenarios in scenarios.yaml, found ${#package_rel_dirs[@]}: ${package_rel_dirs[*]}"

verdicts_file="$WORKDIR/verdicts.jsonl"
: >"$verdicts_file"

for rel_dir in "${package_rel_dirs[@]}"; do
	real_package_dir="$BENCH_DIR/scenarios/$rel_dir"
	[[ -f "$real_package_dir/package.yaml" ]] || fail "package.yaml missing for $rel_dir"

	scenario_label="$(basename "$rel_dir")"
	candidate_dir="$WORKDIR/candidate-$scenario_label"
	cp -r "$real_package_dir" "$candidate_dir"

	out_file="$WORKDIR/verdict-$scenario_label.json"
	err_file="$WORKDIR/verdict-$scenario_label.err"
	code=0
	"$ADMIT_SCRIPT" "$candidate_dir/package.yaml" >"$out_file" 2>"$err_file" || code=$?

	[[ "$code" -eq 0 ]] || fail "$scenario_label: admit-scenario.sh exited $code (expected 0, admitted); stderr: $(cat "$err_file")"

	cat "$out_file" >>"$verdicts_file"
	echo "TC-032: $scenario_label admitted"
done

python3 - "$verdicts_file" "$SCENARIOS_YAML" "$WORKDIR" <<'PYEOF'
import json
import os
import sys

import yaml

verdicts_path, scenarios_yaml_path, workdir = sys.argv[1:4]

with open(verdicts_path) as f:
    verdicts = [json.loads(line) for line in f.read().splitlines() if line]

if len(verdicts) != 4:
    sys.exit(f"TC-032 FAIL: expected 4 verdicts, got {len(verdicts)}")

families = []
for v in verdicts:
    scenario_id = v["scenario_id"]
    if v["status"] != "admitted":
        sys.exit(f"TC-032 FAIL: {scenario_id}: status is {v['status']!r}, expected 'admitted'")
    if v["failing_check"] is not None:
        sys.exit(f"TC-032 FAIL: {scenario_id}: failing_check is {v['failing_check']!r}, expected None")
    for check_name, result in v["checks"].items():
        if result is not True:
            sys.exit(f"TC-032 FAIL: {scenario_id}: check {check_name!r} was {result!r}, want True")
    if v["base_outcome"] is not False:
        sys.exit(f"TC-032 FAIL: {scenario_id}: base_outcome is {v['base_outcome']!r}, expected False")
    if v["reference_outcome"] is not True:
        sys.exit(f"TC-032 FAIL: {scenario_id}: reference_outcome is {v['reference_outcome']!r}, expected True")

    # Read entity_family from the package.yaml admit-scenario.sh evaluated
    # (admit-scenario.sh's own verdict JSON does not carry entity_family) --
    # the candidate copy is still on disk under workdir at this point.
    package_yaml = os.path.join(workdir, f"candidate-{scenario_id}", "package.yaml")
    with open(package_yaml) as f:
        package = yaml.safe_load(f)
    if package.get("scenario_id") != scenario_id:
        sys.exit(f"TC-032 FAIL: candidate dir for {scenario_id!r} did not resolve to a package.yaml naming that scenario_id")
    families.append(package["entity_family"])

expected = {"bug", "change_card", "tech_debt", "feature"}
actual = set(families)
if actual != expected:
    sys.exit(f"TC-032 FAIL: admitted entity_family set is {sorted(actual)}, expected exactly {sorted(expected)}")
if len(families) != len(set(families)):
    sys.exit(f"TC-032 FAIL: a family appears more than once among the four admitted seeds: {families}")

print(f"TC-032: all four seeds admitted, one per family: {sorted(zip(families, [v['scenario_id'] for v in verdicts]))}")
PYEOF

echo "TC-032: PASS"
