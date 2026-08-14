#!/usr/bin/env bash
# TC-040 (test-plan.md AC test matrix; T-E40-F05-014 task spec Test Cases).
#
# Drives bench/scripts/verify-fixture-py-base.sh directly (AC-020's owning
# script), proving it independently catches a red test at base_sha even
# when that test would never be visible through any one scenario package's
# narrower final_predicate.p2p_selection -- the exact gap admission check
# (b) alone leaves open (ADR-F05-10's absolute-P2P premise).
#
# Cases:
#   (positive) the real, frozen fixture.base_sha shared by all four
#              committed scenario packages: verify-fixture-py-base.sh must
#              report success ("CLEAN" + a positive pass count).
#   (fail)     a real commit on top of the frozen base that breaks a fixture
#              test's assertion: verify-fixture-py-base.sh must reject,
#              naming the broken test.
#   (skip)     a real commit on top of the frozen base that adds a *new*,
#              undocumented @pytest.mark.skip on a test that used to pass:
#              verify-fixture-py-base.sh must reject, naming that test --
#              proving the script does not blanket-tolerate skips, only the
#              one named, permanent py-feature-recurring-tasks placeholder.
#
# Mirrors TC-033's make_broken_fixture_sha pattern: a real commit added via
# `git worktree add --detach` (never touching bench/fixture-py's checked-out
# HEAD/working tree) plus one throwaway branch ref (so checkout-scenario-
# fixture.sh's `git clone` of the submodule can fetch it), removed again in
# this script's own cleanup trap.
#
# Caller-Path Contract (test-plan.md TC-040): the real
# verify-fixture-py-base.sh entrypoint driving real checkout-scenario-
# fixture.sh / adapter.sh subprocesses against real checkouts. No
# --include filter is ever passed by this script (that is
# verify-fixture-py-base.sh's own job); the broken/skip states come from
# real fixture commits, not hand-authored JSON standing in for adapter
# output.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

VERIFY_SCRIPT="$SCRIPTS_DIR/verify-fixture-py-base.sh"
FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"
PACKAGES_DIR="$BENCH_DIR/scenarios/packages"

fail() {
	echo "TC-040 FAIL: $1" >&2
	exit 1
}

[[ -x "$VERIFY_SCRIPT" ]] || fail "verify-fixture-py-base.sh missing or not executable"
[[ -e "$FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
SCRATCH_BRANCHES_FILE="$WORKDIR/scratch-branches.txt"
: >"$SCRATCH_BRANCHES_FILE"

cleanup() {
	local branch
	if [[ -f "$SCRATCH_BRANCHES_FILE" ]]; then
		while IFS= read -r branch; do
			[[ -n "$branch" ]] || continue
			git -C "$FIXTURE_SUBMODULE" branch -D "$branch" >/dev/null 2>&1 || true
		done <"$SCRATCH_BRANCHES_FILE"
	fi
	rm -rf "$WORKDIR"
}
trap cleanup EXIT

# Surface residue from a previously killed TC-040 run without false-
# positiving on this run's own branches (same reasoning as TC-033: at this
# point in the script this run has not created any yet).
stale_branches="$(git -C "$FIXTURE_SUBMODULE" for-each-ref --format='%(refname:short)' 'refs/heads/tc040-scratch-*')"
[[ -z "$stale_branches" ]] || fail "stale tc040-scratch-* branch(es) found in bench/fixture-py from a previous run that did not clean up (delete manually with 'git -C bench/fixture-py branch -D <name>' and re-run): $stale_branches"

# All four committed packages must already agree on one frozen base_sha --
# this test's own precondition, not something it derives independently per
# package.
FROZEN_BASE_SHA="$(python3 - "$PACKAGES_DIR" <<'PYEOF'
import glob
import os
import sys

import yaml

packages_dir = sys.argv[1]
shas = {}
for path in sorted(glob.glob(os.path.join(packages_dir, "*", "package.yaml"))):
    with open(path) as f:
        data = yaml.safe_load(f)
    shas[os.path.basename(os.path.dirname(path))] = data["fixture"]["base_sha"]

unique = set(shas.values())
if len(unique) != 1:
    sys.exit(f"scenario packages do not share one frozen base_sha: {shas}")
print(next(iter(unique)))
PYEOF
)" || fail "could not derive one shared frozen base_sha from bench/scenarios/packages/*/package.yaml"
[[ -n "$FROZEN_BASE_SHA" ]] || fail "empty frozen base_sha derived from scenario packages"

FIXTURE_STATE_BEFORE="$(git -C "$FIXTURE_SUBMODULE" rev-parse HEAD) $(git -C "$FIXTURE_SUBMODULE" status --porcelain)"

# -----------------------------------------------------------------------------
# make_fixture_sha <label> <python-mutator-heredoc-file>
#
# Same helper as TC-033's make_broken_fixture_sha, renamed since this test
# uses it for both a genuinely-broken commit and a skip-injecting commit,
# neither of which is "broken" in TC-033's build-failure sense.
# -----------------------------------------------------------------------------
make_fixture_sha() {
	local label="$1" mutator="$2"
	local wt_dir="$WORKDIR/wt-$label"
	local branch="tc040-scratch-$label-$$"

	git -C "$FIXTURE_SUBMODULE" worktree add --quiet --detach "$wt_dir" "$FROZEN_BASE_SHA" || fail "$label: git worktree add failed"
	python3 "$mutator" "$wt_dir" || fail "$label: mutator script failed"
	git -C "$wt_dir" add -A
	git -C "$wt_dir" -c user.email=tc040@shark-bench.local -c user.name="TC-040" commit --quiet -m "TC-040 scratch: $label (never pushed, deleted by this test's own cleanup)"
	local new_sha
	new_sha="$(git -C "$wt_dir" rev-parse HEAD)"
	git -C "$FIXTURE_SUBMODULE" branch --quiet "$branch" "$new_sha"
	git -C "$FIXTURE_SUBMODULE" worktree remove --force "$wt_dir"
	echo "$branch" >>"$SCRATCH_BRANCHES_FILE"
	echo "$new_sha"
}

echo "TC-040: (positive) the real frozen base_sha is reported CLEAN"

pos_out="$WORKDIR/pos.out"
pos_err="$WORKDIR/pos.err"
pos_code=0
"$VERIFY_SCRIPT" "$FROZEN_BASE_SHA" >"$pos_out" 2>"$pos_err" || pos_code=$?
[[ "$pos_code" -eq 0 ]] || fail "positive control: verify-fixture-py-base.sh exited $pos_code (expected 0); stderr: $(cat "$pos_err")"
grep -q '^CLEAN$' "$pos_out" || fail "positive control: expected a 'CLEAN' line in stdout, got: $(cat "$pos_out")"
pass_line="$(grep '^pass_count=' "$pos_out" || true)"
[[ -n "$pass_line" ]] || fail "positive control: expected a 'pass_count=<N>' line in stdout, got: $(cat "$pos_out")"
pass_n="${pass_line#pass_count=}"
[[ "$pass_n" =~ ^[0-9]+$ && "$pass_n" -gt 0 ]] || fail "positive control: pass_count is not a positive integer: $pass_line"
echo "TC-040: positive control CLEAN with pass_count=$pass_n"

echo "TC-040: (fail) a real commit breaking a fixture test's assertion is rejected, naming the broken test"

cat >"$WORKDIR/mutator-fail.py" <<'PYEOF'
import sys

worktree = sys.argv[1]
path = f"{worktree}/taskmanager/due_date.py"
with open(path) as f:
    content = f.read()
old = "    return parse_due_date(due_date) < today"
new = "    return False  # TC-040 scratch: intentionally broken"
if old not in content:
    sys.exit("mutator-fail: due_date.py is_overdue body did not match the expected shape -- mutation target moved")
content = content.replace(old, new, 1)
with open(path, "w") as f:
    f.write(content)
PYEOF
BROKEN_SHA="$(make_fixture_sha fail "$WORKDIR/mutator-fail.py")"

fail_out="$WORKDIR/fail.out"
fail_err="$WORKDIR/fail.err"
fail_code=0
"$VERIFY_SCRIPT" "$BROKEN_SHA" >"$fail_out" 2>"$fail_err" || fail_code=$?
[[ "$fail_code" -eq 1 ]] || fail "(fail) case: verify-fixture-py-base.sh exited $fail_code (expected 1); stdout: $(cat "$fail_out"); stderr: $(cat "$fail_err")"
grep -q 'tests.test_due_date::test_is_overdue_true_for_past_date' "$fail_err" \
	|| fail "(fail) case: stderr did not name the broken test: $(cat "$fail_err")"
echo "TC-040: (fail) case rejected, naming the broken test as expected"

echo "TC-040: (skip) a real commit adding an undocumented new skip is rejected, naming that test"

cat >"$WORKDIR/mutator-skip.py" <<'PYEOF'
import sys

worktree = sys.argv[1]
path = f"{worktree}/tests/test_validation.py"
with open(path) as f:
    content = f.read()
old = "def test_non_empty_title_rejects_blank():"
new = (
    "@pytest.mark.skip(reason=\"TC-040 scratch: undocumented new skip\")\n"
    "def test_non_empty_title_rejects_blank():"
)
if old not in content:
    sys.exit("mutator-skip: test_validation.py did not match the expected shape -- mutation target moved")
content = content.replace(old, new, 1)
with open(path, "w") as f:
    f.write(content)
PYEOF
SKIP_SHA="$(make_fixture_sha skip "$WORKDIR/mutator-skip.py")"

skip_out="$WORKDIR/skip.out"
skip_err="$WORKDIR/skip.err"
skip_code=0
"$VERIFY_SCRIPT" "$SKIP_SHA" >"$skip_out" 2>"$skip_err" || skip_code=$?
[[ "$skip_code" -eq 1 ]] || fail "(skip) case: verify-fixture-py-base.sh exited $skip_code (expected 1); stdout: $(cat "$skip_out"); stderr: $(cat "$skip_err")"
grep -q 'tests.test_validation::test_non_empty_title_rejects_blank' "$skip_err" \
	|| fail "(skip) case: stderr did not name the newly-skipped test: $(cat "$skip_err")"
echo "TC-040: (skip) case rejected, naming the newly-skipped test as expected"

echo "TC-040: verify bench/fixture-py's checked-out state is unchanged by this test's scratch branches"

FIXTURE_STATE_AFTER="$(git -C "$FIXTURE_SUBMODULE" rev-parse HEAD) $(git -C "$FIXTURE_SUBMODULE" status --porcelain)"
[[ "$FIXTURE_STATE_BEFORE" == "$FIXTURE_STATE_AFTER" ]] || fail "bench/fixture-py's checked-out HEAD/working tree changed by this test (before: $FIXTURE_STATE_BEFORE; after: $FIXTURE_STATE_AFTER)"

echo "TC-040: PASS"
