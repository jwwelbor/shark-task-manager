#!/usr/bin/env bash
# TC-034 (test-plan.md AC test matrix; T-E40-F05-015 task spec Test Cases).
#
# Exercises AC-008 / REQ-NF-004 ("Determinism boundary" section of
# test-plan.md): bench/scripts/admit-scenario.sh over all four REQ-F-014
# seed packages, run twice, each round against an independently provisioned
# scratch copy of every package -- admit-scenario.sh itself provisions a
# fresh checkout-scenario-fixture.sh checkout per candidate internally, so
# no checkout is shared between rounds, and no candidate copy is reused
# between rounds either (a memoizing/caching implementation must not be
# able to pass by reading round 1's own output back).
#
# Asserts byte-identical stdout (the JSON verdict line) and exit code per
# candidate across both rounds, and, per the Determinism boundary section's
# own explicit normalization requirement, separately confirms neither
# adapter's `test`/`lint` capability output smuggles a wall-clock/duration
# field into the compared verdict -- rather than merely assuming it doesn't.
#
# Caller-Path Contract (test-plan.md TC-034): the real admit-scenario.sh
# entrypoint driving real checkout-scenario-fixture.sh / adapter.sh /
# eval-predicate.sh subprocesses against real checkouts, twice, from a
# clean scratch copy each time. No verdict is memoized or cached between
# the two invocations under test.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
PYTHON_ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"

fail() {
	echo "TC-034 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -x "$PYTHON_ADAPTER" ]] || fail "python adapter.sh missing or not executable"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

mapfile -t package_rel_dirs < <(python3 - "$SCENARIOS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for rel in sorted(data["scenarios"]):
    print(rel)
PYEOF
)
[[ "${#package_rel_dirs[@]}" -eq 4 ]] || fail "expected exactly 4 registered scenarios in scenarios.yaml, found ${#package_rel_dirs[@]}"

# run_round <round_dir> -- admits all four seeds from a freshly-copied set of
# candidate package dirs, writing one stdout/exit file pair per scenario_id.
# Called twice below with entirely independent WORKDIR subtrees, so no
# filesystem state is shared between rounds.
run_round() {
	local round_dir="$1"
	mkdir -p "$round_dir"

	local rel_dir scenario_label candidate_dir
	for rel_dir in "${package_rel_dirs[@]}"; do
		scenario_label="$(basename "$rel_dir")"
		candidate_dir="$round_dir/candidate-$scenario_label"
		cp -r "$BENCH_DIR/scenarios/$rel_dir" "$candidate_dir"

		set +e
		"$ADMIT_SCRIPT" "$candidate_dir/package.yaml" >"$round_dir/$scenario_label.out" 2>"$round_dir/$scenario_label.err"
		echo $? >"$round_dir/$scenario_label.exit"
		set -e
	done
}

echo "TC-034: round 1 - independently provisioned candidate copies and checkouts"
run_round "$WORKDIR/round1"

echo "TC-034: round 2 - independently provisioned candidate copies and checkouts"
run_round "$WORKDIR/round2"

echo "TC-034: comparing round 1 vs round 2 for byte-identical verdict output"

compare_file() {
	local label="$1" rel="$2"
	local f1="$WORKDIR/round1/$rel" f2="$WORKDIR/round2/$rel"
	[[ -f "$f1" ]] || fail "$label: round 1 output missing: $f1"
	[[ -f "$f2" ]] || fail "$label: round 2 output missing: $f2"
	if ! cmp -s "$f1" "$f2"; then
		fail "$label: round 1 and round 2 output differ (not byte-identical): $(diff "$f1" "$f2" | head -20)"
	fi
}

for rel_dir in "${package_rel_dirs[@]}"; do
	scenario_label="$(basename "$rel_dir")"
	compare_file "$scenario_label (stdout verdict)" "$scenario_label.out"
	compare_file "$scenario_label (exit code)" "$scenario_label.exit"
	[[ "$(cat "$WORKDIR/round1/$scenario_label.exit")" -eq 0 ]] || fail "$scenario_label did not exit 0 (admitted) in round 1: $(cat "$WORKDIR/round1/$scenario_label.err")"
	echo "TC-034: $scenario_label byte-identical across both rounds"
done

echo "TC-034: confirming adapter test/lint output carries no wall-clock/duration field (Determinism boundary normalization)"

# Real adapter invocation, not a stub: proves the *shipped* adapter output
# genuinely has nothing timing-shaped to normalize, rather than merely
# assuming so. Uses round 1's own already-provisioned checkout indirectly
# via a fresh, cheap checkout of the bug seed's fixture.
base_sha="$(python3 - "$BENCH_DIR/scenarios/packages/py-bug-due-date-boundary/package.yaml" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    print(yaml.safe_load(f)["fixture"]["base_sha"])
PYEOF
)"
timing_checkout="$WORKDIR/timing-checkout"
"$SCRIPTS_DIR/checkout-scenario-fixture.sh" py "$base_sha" "$timing_checkout" >/dev/null

test_doc="$("$PYTHON_ADAPTER" test --checkout "$timing_checkout")"
lint_doc="$("$PYTHON_ADAPTER" lint --checkout "$timing_checkout")"

python3 - "$test_doc" "$lint_doc" <<'PYEOF'
import json
import sys

test_doc, lint_doc = (json.loads(s) for s in sys.argv[1:3])

TIMING_KEY_SUBSTRINGS = ("duration", "elapsed", "timestamp", "time_ms", "wall_clock", "started_at", "ended_at")

def check(doc, label):
    for entry_list_key in ("entries", "issues"):
        for entry in doc.get(entry_list_key, []):
            for key in entry:
                lowered = key.lower()
                if any(sub in lowered for sub in TIMING_KEY_SUBSTRINGS):
                    sys.exit(f"TC-034 FAIL: {label} entry carries a timing-shaped field {key!r}: {entry}")

check(test_doc, "test")
check(lint_doc, "lint")
print("TC-034: adapter test/lint output carries no wall-clock/duration field")
PYEOF

echo "TC-034: PASS"
