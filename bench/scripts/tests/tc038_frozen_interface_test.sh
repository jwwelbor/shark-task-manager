#!/usr/bin/env bash
# TC-038 (test-plan.md AC test matrix; T-E40-F05-006 task spec Test Cases).
#
# Exercises AC-017 / AC-T2 / AC-T3 (REQ-NF-006; ADR-F05-06): proves
# multi-fixture checkout was added as a sibling script, never as a change to
# the frozen `checkout-fixture.sh` interface.
#
# Three parts:
#   1. AC-T2 -- `git diff <pre-feature-ref> -- bench/scripts/checkout-fixture.sh`
#      is empty: the original I-01 script is byte-unchanged by this feature.
#      Pre-feature ref is `git merge-base HEAD origin/main` (this branch
#      never merged back before F05 started, mirroring tc020's own
#      base-ref reasoning) -- an unresolvable base is a skip, not a silent
#      pass (same "wrong comparison is worse than no check" principle
#      tc020 documents).
#   2. AC-T1 -- `checkout-scenario-fixture.sh <fixture_id> <base_sha>
#      <dest_dir>` resolves `submodule_path` from `bench/scenarios/scenarios.yaml`
#      and clones/checks out every registered fixture at the requested SHA.
#      It also exercises each explicit guard: wrong arity, unregistered id,
#      and a destination that already exists.
#   3. AC-T3 -- F01's own TC-004, TC-006, TC-007 still pass UNMODIFIED,
#      proving `admit.sh`/`build-ledgers.sh` were not silently repointed at
#      the new script (a repointed caller would only surface here, not in a
#      fresh assertion invented for this feature -- test-plan.md's own
#      framing for TC-038's Caller-Path Contract).
#
# Caller-Path Contract (test-plan.md TC-038): real git diff; real script
# execution against real submodule checkouts; real, unmodified re-run of
# F01's committed TC-004/TC-006/TC-007 -- none of the three are hand-edited
# to accommodate this feature.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

CHECKOUT_FIXTURE_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CHECKOUT_SCENARIO_FIXTURE_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"

fail() {
	echo "TC-038 FAIL: $1" >&2
	exit 1
}

[[ -f "$CHECKOUT_FIXTURE_SCRIPT" ]] || fail "checkout-fixture.sh missing: $CHECKOUT_FIXTURE_SCRIPT"
[[ -x "$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable: $CHECKOUT_SCENARIO_FIXTURE_SCRIPT"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"
command -v git >/dev/null 2>&1 || fail "git not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Part 1 -- AC-T2: checkout-fixture.sh is byte-unchanged since the
# pre-feature ref. Same unresolvable-base skip discipline as tc020: a
# vacuous pass is worse than no check, so a base that fails to resolve is
# logged and skipped, never silently treated as "unchanged".
# ---------------------------------------------------------------------------
echo "TC-038: part 1 - checkout-fixture.sh byte-unchanged since pre-feature ref (AC-T2)"

base=""
if ! base="$(cd "$REPO_ROOT" && git merge-base HEAD origin/main 2>/dev/null)"; then
	base=""
fi

if [[ -z "$base" ]]; then
	echo "TC-038(part 1: no merge-base resolved against origin/main -- skipped, logged not silently passed) SKIP" >&2
else
	diff_output="$(cd "$REPO_ROOT" && git diff "$base" -- bench/scripts/checkout-fixture.sh)"
	[[ -z "$diff_output" ]] || fail "checkout-fixture.sh differs from pre-feature ref $base (must be byte-unchanged, REQ-NF-006):
$diff_output"
	echo "TC-038(part 1: checkout-fixture.sh byte-unchanged since $base) PASS"
fi

# ---------------------------------------------------------------------------
# Part 2 -- AC-T1: checkout-scenario-fixture.sh resolves submodule_path
# from scenarios.yaml and clones/checks out every registered fixture. Parse
# the live registry here instead of duplicating fixture ids/paths in test
# constants: a hardcoded lookup table could pass if the script stopped
# consulting scenarios.yaml.
# ---------------------------------------------------------------------------
echo "TC-038: part 2 - checkout-scenario-fixture.sh resolves both registered fixtures (AC-T1)"

check_fixture_checkout() {
	# check_fixture_checkout <fixture_id> <base_sha> <expected_submodule_dir>
	local fixture_id="$1" sha="$2" expected_submodule="$3"
	local dest="$WORKDIR/checkout-$fixture_id"

	"$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" "$fixture_id" "$sha" "$dest" >"$WORKDIR/$fixture_id.out" 2>"$WORKDIR/$fixture_id.err" \
		|| fail "checkout-scenario-fixture.sh $fixture_id $sha $dest exited non-zero: $(cat "$WORKDIR/$fixture_id.err")"

	[[ -e "$dest/.git" ]] || fail "checkout-scenario-fixture.sh $fixture_id: dest_dir has no .git after checkout: $dest"

	local actual_head
	actual_head="$(git -C "$dest" rev-parse HEAD)"
	[[ "$actual_head" == "$sha" ]] || fail "checkout-scenario-fixture.sh $fixture_id: dest checked out $actual_head, want $sha"

	local origin_url
	origin_url="$(git -C "$dest" remote get-url origin)"
	[[ "$origin_url" == "$expected_submodule" ]] || fail "checkout-scenario-fixture.sh $fixture_id: cloned from $origin_url, want submodule_path-resolved $expected_submodule (scenarios.yaml resolution not honored)"

	echo "TC-038: $fixture_id fixture resolved via scenarios.yaml and checked out at $sha"
}

registry_output="$WORKDIR/fixture-registry.tsv"
if ! python3 - "$SCENARIOS_YAML" "$CORPUS_YAML" "$REPO_ROOT" >"$registry_output" <<'PYEOF'
import os
import subprocess
import sys
import yaml

scenarios_path, corpus_path, repo_root = sys.argv[1:]
with open(scenarios_path) as f:
    scenarios = yaml.safe_load(f) or {}
with open(corpus_path) as f:
    corpus = yaml.safe_load(f) or {}

repo_root = os.path.realpath(repo_root)
for fixture_id, entry in sorted((scenarios.get("fixtures") or {}).items()):
    rel = (entry or {}).get("submodule_path")
    if not rel:
        raise SystemExit(f"fixture {fixture_id!r} has no submodule_path")
    submodule = os.path.realpath(os.path.join(repo_root, rel))
    if os.path.commonpath((repo_root, submodule)) != repo_root:
        raise SystemExit(f"fixture {fixture_id!r} has an invalid submodule_path: {rel!r}")
    # I-01's Go fixture keeps its immutable base SHA in corpus.yaml; other
    # registered fixtures are pinned by their checked-out submodule HEAD.
    if fixture_id == "go":
        base_sha = ((corpus.get("fixture") or {}).get("base_sha"))
    else:
        base_sha = subprocess.check_output(["git", "-C", submodule, "rev-parse", "HEAD"], text=True).strip()
    if not base_sha:
        raise SystemExit(f"fixture {fixture_id!r} has no resolvable base SHA")
    print(f"{fixture_id}\t{base_sha}\t{submodule}")
PYEOF
then
	fail "could not read complete fixture registry from scenarios.yaml"
fi

while IFS=$'\t' read -r fixture_id base_sha submodule; do
	[[ -n "$fixture_id" && -n "$base_sha" && -n "$submodule" ]] || fail "scenarios.yaml yielded an incomplete fixture registration"
	[[ -e "$submodule/.git" ]] || fail "$fixture_id fixture submodule not initialized: $submodule (run 'git submodule update --init')"
	check_fixture_checkout "$fixture_id" "$base_sha" "$submodule"
done < "$registry_output"

expect_rejected() {
	# expect_rejected <label> <stderr-token> <command...>
	local label="$1" token="$2"
	shift 2
	if "$@" >"$WORKDIR/$label.out" 2>"$WORKDIR/$label.err"; then
		fail "$label: expected checkout-scenario-fixture.sh to reject the invocation"
	fi
	grep -qi -- "$token" "$WORKDIR/$label.err" || fail "$label: rejection did not name $token: $(cat "$WORKDIR/$label.err")"
	echo "TC-038: $label guard rejected as expected"
}

expect_rejected wrong-arg-count usage "$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" py
expect_rejected unregistered-fixture fixture_id "$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" does-not-exist 0000000000000000000000000000000000000000 "$WORKDIR/unregistered"
mkdir "$WORKDIR/existing-destination"
expect_rejected pre-existing-destination dest_dir "$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" py 0000000000000000000000000000000000000000 "$WORKDIR/existing-destination"

echo "TC-038(part 2: registered fixtures resolve via scenarios.yaml, clone/checkout at requested SHAs, and all guards reject) PASS"

# ---------------------------------------------------------------------------
# Part 3 -- AC-T3: F01's own TC-004, TC-006, TC-007 still pass UNMODIFIED.
# A silently repointed admit.sh/build-ledgers.sh would only surface here.
# ---------------------------------------------------------------------------
echo "TC-038: part 3 - F01's TC-004/TC-006/TC-007 re-run unmodified (AC-T3)"

for tc in tc004_admit_full_set_test.sh tc006_admit_reproducibility_test.sh tc007_ledger_reproducibility_test.sh; do
	tc_script="$SCRIPT_DIR/$tc"
	[[ -x "$tc_script" ]] || fail "F01 test script missing or not executable: $tc_script"
	echo "TC-038: re-running $tc"
	"$tc_script" || fail "$tc failed after this feature landed -- admit.sh/build-ledgers.sh may have been silently repointed at checkout-scenario-fixture.sh"
	echo "TC-038: $tc PASS"
done

echo "TC-038(part 3: F01's TC-004/TC-006/TC-007 pass unmodified -- I-01 callers unaffected) PASS"

echo "TC-038: PASS"
