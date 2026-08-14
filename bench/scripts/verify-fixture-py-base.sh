#!/usr/bin/env bash
# verify-fixture-py-base.sh <base_sha>
#
# The AC-020/TC-040 owner: proves REQ-F-016's "full test suite MUST be
# green at base_sha" for the whole controlled Python fixture, independent
# of any single scenario package's own narrower final_predicate.p2p_selection
# (ADR-F05-10's absolute-P2P premise rests on the *fixture's* base being
# green, not on any one package's selection happening to cover it).
# admit-scenario.sh's own check (b) only resolves one package's
# p2p_selection -- a subset of the fixture's tests -- so passing check (b)
# for all four packages does not by itself prove this claim; this script is
# the named, independent owner of the broader claim (test-plan.md "Test
# infrastructure gaps").
#
# Fixture-specific by design (unlike checkout-scenario-fixture.sh, which is
# generic over fixture_id): this script's whole job is AC-020's literal
# text, "a fresh clone of bench/fixture-py at base_sha", so it hardcodes
# fixture_id "py" and bench/adapters/python/adapter.sh rather than taking a
# fixture_id argument. This does not reintroduce a REQ-F-007 language branch
# into any *generic* admission/checkout script -- this file is the one
# place a Python-only verification is expected to live.
#
# Checks, in order:
#   1. checkout-scenario-fixture.sh py <base_sha> <tmp_dir> (fresh clone)
#   2. pyproject.toml carries a dependency section ([project].dependencies),
#      a lint section ([tool.ruff]), and a formatter section ([tool.black])
#   3. no path under <tmp_dir> matches bench/scenarios/** or is/contains an
#      evaluator/ directory -- the fixture's working tree is fixture code
#      only (REQ-F-016)
#   4. bench/adapters/python/adapter.sh test --checkout <tmp_dir> with NO
#      --include/--exclude-id filter (the full suite, not a p2p_selection
#      subset -- TC-040's own counter-factual)
#   5. every entry's outcome is "pass", with exactly one named exception:
#      tests.test_manager::test_recurring_task_generates_next_occurrence,
#      the py-feature-recurring-tasks seed's own permanently-skipped
#      placeholder (committed at the fixture's very first commit, proving
#      the recurring-task capability's deliberate absence at base -- see
#      that test's own @pytest.mark.skip reason and every other package's
#      p2p_selection.exclude_test_ids, which excludes this same id for the
#      identical reason). Any *other* skip -- or any fail -- rejects, naming
#      the offending id: a newly introduced, undocumented skip on a test
#      that used to pass must fail this check exactly like a genuine
#      regression would (this is the check that makes AC-020 non-vacuous;
#      a blanket "skip is always fine" reading would let that class of
#      regression through unnoticed).
#
# Prints "CLEAN" plus the full-suite pass count on success (test-plan.md
# "Observability design" table). Exit 0 on success, 1 on a failed check
# (the fixture is not green / not clean at base_sha), 2 on a usage or
# script/toolchain error.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CHECKOUT_SCRIPT="$SCRIPT_DIR/checkout-scenario-fixture.sh"
ADAPTER_SCRIPT="$BENCH_DIR/adapters/python/adapter.sh"

# The one id permitted to be "skip" instead of "pass" -- see header comment.
ALLOWED_SKIP_ID="tests.test_manager::test_recurring_task_generates_next_occurrence"

usage() {
	echo "usage: verify-fixture-py-base.sh <base_sha>" >&2
	exit 2
}

fail() {
	echo "verify-fixture-py-base: $1" >&2
	exit 1
}

[[ $# -eq 1 ]] || usage
base_sha="$1"

[[ -x "$CHECKOUT_SCRIPT" ]] || {
	echo "verify-fixture-py-base: checkout-scenario-fixture.sh missing or not executable: $CHECKOUT_SCRIPT" >&2
	exit 2
}
[[ -x "$ADAPTER_SCRIPT" ]] || {
	echo "verify-fixture-py-base: python adapter missing or not executable: $ADAPTER_SCRIPT" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-fixture-py-base: python3 not found on PATH" >&2
	exit 2
}

WORKDIR="$(mktemp -d)"
CHECKOUT_DIR="$WORKDIR/checkout"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

checkout_err="$WORKDIR/checkout.err"
if ! "$CHECKOUT_SCRIPT" py "$base_sha" "$CHECKOUT_DIR" >"$WORKDIR/checkout.out" 2>"$checkout_err"; then
	fail "fresh checkout of bench/fixture-py at $base_sha failed: $(cat "$checkout_err")"
fi

# --- pyproject.toml: dependency / lint / formatter sections present -------
pyproject_path="$CHECKOUT_DIR/pyproject.toml"
[[ -f "$pyproject_path" ]] || fail "pyproject.toml not found in fresh checkout: $pyproject_path"

pyproject_check_err="$WORKDIR/pyproject-check.err"
if ! python3 - "$pyproject_path" 2>"$pyproject_check_err" <<'PYEOF'
import sys
import tomllib

path = sys.argv[1]
with open(path, "rb") as f:
    data = tomllib.load(f)

missing = []
if "dependencies" not in (data.get("project") or {}):
    missing.append("[project].dependencies (dependency section)")
tool = data.get("tool") or {}
if "ruff" not in tool:
    missing.append("[tool.ruff] (lint section)")
if "black" not in tool:
    missing.append("[tool.black] (formatter section)")

if missing:
    sys.exit("pyproject.toml missing required section(s): " + "; ".join(missing))
PYEOF
then
	fail "$(cat "$pyproject_check_err")"
fi

# --- no scenario/evaluator-only material leaked into the fixture tree -----
leak="$(find "$CHECKOUT_DIR" -type d \( -path '*/bench/scenarios/*' -o -name 'evaluator' \) -print -quit)"
[[ -z "$leak" ]] || fail "fixture checkout contains scenario/evaluator-only material: $leak"

# --- full, unfiltered test suite -------------------------------------------
test_out="$WORKDIR/test.json"
test_err="$WORKDIR/test.err"
if ! "$ADAPTER_SCRIPT" test --checkout "$CHECKOUT_DIR" >"$test_out" 2>"$test_err"; then
	fail "adapter.sh test failed: $(cat "$test_err")"
fi

result="$(python3 - "$test_out" "$ALLOWED_SKIP_ID" 2>&1 <<'PYEOF'
import json
import sys

test_json_path, allowed_skip_id = sys.argv[1:3]

with open(test_json_path) as f:
    doc = json.load(f)

entries = doc.get("entries") or []
if not entries:
    sys.exit("adapter.sh test (unfiltered) resolved to an empty entry set at base_sha")

failing = [e["id"] for e in entries if e["outcome"] == "fail"]
if failing:
    sys.exit(f"{len(failing)} failing test(s) at base_sha: {sorted(failing)}")

unexpected_skips = sorted(e["id"] for e in entries if e["outcome"] == "skip" and e["id"] != allowed_skip_id)
if unexpected_skips:
    sys.exit(f"{len(unexpected_skips)} unexpected skipped test(s) at base_sha (only {allowed_skip_id!r} is allowed to skip): {unexpected_skips}")

pass_count = sum(1 for e in entries if e["outcome"] == "pass")
print(pass_count)
PYEOF
)" || fail "$result"

echo "CLEAN"
echo "pass_count=$result"
