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
#      and clones/checks out both registered fixtures (`py`, `go`) at the
#      requested SHA.
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
GO_FIXTURE_SUBMODULE="$BENCH_DIR/fixture-repo"
PY_FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"

fail() {
	echo "TC-038 FAIL: $1" >&2
	exit 1
}

[[ -f "$CHECKOUT_FIXTURE_SCRIPT" ]] || fail "checkout-fixture.sh missing: $CHECKOUT_FIXTURE_SCRIPT"
[[ -x "$CHECKOUT_SCENARIO_FIXTURE_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable: $CHECKOUT_SCENARIO_FIXTURE_SCRIPT"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml missing: $SCENARIOS_YAML"
[[ -e "$GO_FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-repo submodule not initialized; run 'git submodule update --init'"
[[ -e "$PY_FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"
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
# from scenarios.yaml and clones/checks out both registered fixtures.
# ---------------------------------------------------------------------------
echo "TC-038: part 2 - checkout-scenario-fixture.sh resolves both registered fixtures (AC-T1)"

GO_BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$GO_BASE_SHA" ]] || fail "could not derive base_sha from $CORPUS_YAML"

PY_BASE_SHA="$(git -C "$PY_FIXTURE_SUBMODULE" rev-parse HEAD)"
[[ -n "$PY_BASE_SHA" ]] || fail "could not derive HEAD sha from $PY_FIXTURE_SUBMODULE"

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

check_fixture_checkout py "$PY_BASE_SHA" "$PY_FIXTURE_SUBMODULE"
check_fixture_checkout go "$GO_BASE_SHA" "$GO_FIXTURE_SUBMODULE"

echo "TC-038(part 2: both registered fixtures resolve via scenarios.yaml and clone/checkout at the requested SHA) PASS"

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
