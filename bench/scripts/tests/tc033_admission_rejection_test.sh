#!/usr/bin/env bash
# TC-033 (test-plan.md AC test matrix; T-E40-F05-009 task spec Test Cases,
# AC-T1/AC-T2).
#
# Drives bench/scripts/admit-scenario.sh against seven candidates, each a
# copy of the real, committed py-bug-due-date-boundary scenario mutated at
# test time for exactly one REQ-F-012 trigger:
#   (a)       adapter build capability fails at base_sha
#   (b-empty) p2p_selection resolves to an empty entry set at base
#   (b-fail)  p2p_selection resolves to a non-empty set with a failing entry
#   (c)       stage_matrix violates the REQ-F-004 family invariant
#   (d)       final_predicate is already true at base
#   (e)       final_predicate stays false after the reference solution
#   (f)       resource_policy.max_cost_usd is non-positive
#
# Each must be rejected naming EXACTLY its own check -- (b-empty) and
# (b-fail) both name check (b) despite differing triggers -- and none may
# ever be admitted (AC-T2): a candidate that accidentally passes every check
# fails this test loudly rather than silently.
#
# Candidates (a) and (b-fail) need a REAL broken commit in bench/fixture-py
# (a build-breaking syntax error / a test-breaking source edit) -- no
# generic script may mutate an in-flight ephemeral checkout admit-scenario.sh
# owns internally, so the only way to give admit-scenario.sh's own fresh
# checkout-scenario-fixture.sh checkout a genuinely broken base_sha is a real
# commit it can clone. make_broken_fixture_sha() below adds that commit via
# `git worktree add --detach` (never touching bench/fixture-py's checked-out
# HEAD or working tree) plus one throwaway branch ref (so a `git clone` of
# the submodule transfers the commit), and removes the branch ref again in
# this script's own cleanup trap -- the submodule's checked-out state is
# byte-identical before and after this test runs (verified below).
#
# Also runs one positive control -- an unmutated copy of the real seed --
# through the same admit-scenario.sh code path so this file is the one
# registered test that actually exercises the *admitted* branch (status
# "admitted", every check true, the admission: block written into the
# copy's package.yaml, and a second run producing a byte-identical file).
# Without it, every candidate below being "rejected" would be equally true
# of a gate that always rejects -- this positive control is what makes the
# seven rejections meaningful evidence rather than a one-sided test.
#
# Caller-Path Contract (test-plan.md TC-033): the real admit-scenario.sh
# entrypoint driving real checkout-scenario-fixture.sh / adapter.sh /
# eval-predicate.sh subprocesses against real checkouts. No check's
# subprocess result is stubbed, no per-candidate boolean is hardcoded, and no
# candidate is special-cased to force its expected rejection message.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit-scenario.sh"
REAL_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-bug-due-date-boundary"
FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"

fail() {
	echo "TC-033 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit-scenario.sh missing or not executable"
[[ -f "$REAL_PACKAGE_DIR/package.yaml" ]] || fail "py-bug-due-date-boundary/package.yaml missing"
[[ -e "$FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"

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

# Surface residue from a previously killed TC-033 run (SIGKILL, or any exit
# this script's own EXIT trap didn't get to run for) without false-positiving
# on this run's own branches -- at this point in the script this run has not
# created any yet, so any tc033-scratch-* branch already present is, by
# construction, leftover from another run, never this one's own (per-run
# branch names are suffixed with that run's own $$).
stale_branches="$(git -C "$FIXTURE_SUBMODULE" for-each-ref --format='%(refname:short)' 'refs/heads/tc033-scratch-*')"
[[ -z "$stale_branches" ]] || fail "stale tc033-scratch-* branch(es) found in bench/fixture-py from a previous run that did not clean up (delete manually with 'git -C bench/fixture-py branch -D <name>' and re-run): $stale_branches"

BASE_SHA="$(python3 - "$REAL_PACKAGE_DIR/package.yaml" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    print(yaml.safe_load(f)["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$BASE_SHA" ]] || fail "could not derive base_sha from the real package.yaml"

FIXTURE_STATE_BEFORE="$(git -C "$FIXTURE_SUBMODULE" rev-parse HEAD) $(git -C "$FIXTURE_SUBMODULE" status --porcelain)"

# -----------------------------------------------------------------------------
# make_broken_fixture_sha <label> <python-mutator-heredoc-file>
#
# Adds a commit on top of BASE_SHA in a linked worktree (never touching
# bench/fixture-py's own checked-out HEAD/working tree), applying the given
# python script (invoked as `python3 <mutator> <worktree_dir>`) to corrupt one
# file, then creates a throwaway branch ref pointing at the new commit so a
# subsequent `git clone` of the submodule can fetch it. Echoes the new SHA.
# The branch name is appended to SCRATCH_BRANCHES_FILE (not an in-memory
# array: this function is always invoked via `$(...)` command substitution,
# which runs it in a subshell, so an in-memory array mutation here would
# never be visible to this script's own cleanup trap) for this script's
# cleanup trap to delete.
# -----------------------------------------------------------------------------
make_broken_fixture_sha() {
	local label="$1" mutator="$2"
	local wt_dir="$WORKDIR/wt-$label"
	local branch="tc033-scratch-$label-$$"

	git -C "$FIXTURE_SUBMODULE" worktree add --quiet --detach "$wt_dir" "$BASE_SHA" || fail "$label: git worktree add failed"
	python3 "$mutator" "$wt_dir" || fail "$label: mutator script failed"
	git -C "$wt_dir" add -A
	git -C "$wt_dir" -c user.email=tc033@shark-bench.local -c user.name="TC-033" commit --quiet -m "TC-033 scratch: $label (never pushed, deleted by this test's own cleanup)"
	local broken_sha
	broken_sha="$(git -C "$wt_dir" rev-parse HEAD)"
	git -C "$FIXTURE_SUBMODULE" branch --quiet "$branch" "$broken_sha"
	git -C "$FIXTURE_SUBMODULE" worktree remove --force "$wt_dir"
	echo "$branch" >>"$SCRATCH_BRANCHES_FILE"
	echo "$broken_sha"
}

# mk_candidate <label> -- copies the real, committed package into a scratch
# directory this test is free to mutate.
mk_candidate() {
	local label="$1"
	local dest="$WORKDIR/candidate-$label"
	cp -r "$REAL_PACKAGE_DIR" "$dest"
	echo "$dest/package.yaml"
}

set_base_sha() {
	local package_yaml="$1" new_sha="$2"
	python3 - "$package_yaml" "$new_sha" <<'PYEOF'
import sys
import yaml

path, new_sha = sys.argv[1:3]
with open(path) as f:
    data = yaml.safe_load(f)
data["fixture"]["base_sha"] = new_sha
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
}

# assert_rejected <label> <package_yaml> <expected_failing_check>
assert_rejected() {
	local label="$1" package_yaml="$2" expected_check="$3"
	local out_file="$WORKDIR/verdict-${label}.json"
	local code=0
	"$ADMIT_SCRIPT" "$package_yaml" >"$out_file" 2>"$WORKDIR/verdict-${label}.err" || code=$?

	[[ "$code" -eq 1 ]] || fail "$label: admit-scenario.sh exited $code (expected 1 for a rejected candidate); stderr: $(cat "$WORKDIR/verdict-${label}.err")"

	python3 - "$label" "$out_file" "$expected_check" <<'PYEOF'
import json
import sys

label, verdict_path, expected_check = sys.argv[1:4]

with open(verdict_path) as f:
    line = f.read().strip()

try:
    verdict = json.loads(line)
except json.JSONDecodeError as exc:
    sys.exit(f"TC-033 FAIL: {label}: admit-scenario.sh did not print a single JSON verdict line: {line!r} ({exc})")

if verdict["status"] != "rejected":
    sys.exit(f"TC-033 FAIL: {label}: expected status 'rejected' (AC-T2: never admitted), got {verdict['status']!r}: {line}")
if verdict["failing_check"] != expected_check:
    sys.exit(
        f"TC-033 FAIL: {label}: expected failing_check {expected_check!r}, "
        f"got {verdict['failing_check']!r}: {line}"
    )
print(f"TC-033: {label} rejected naming check {expected_check!r} as expected")
PYEOF
}

# assert_containment_rejected <label> <package_yaml> <expected_message_substring>
#
# UAT Finding 1 rework (T-E40-F05-009 kickback): a package-declared relative
# path (evaluator_only.oracle_tests[], evaluator_only.reference_solution,
# final_predicate.p2p_selection.include[]) that escapes its required
# containment root -- absolute as declared, a `../` traversal, or a symlink
# resolving outside it -- is a script/toolchain precondition failure (exit
# 2, no verdict printed), not a normal per-candidate check verdict. Distinct
# from assert_rejected above, which expects exit 1 and a JSON "rejected"
# verdict naming one of the six named checks.
assert_containment_rejected() {
	local label="$1" package_yaml="$2" expected_substring="$3"
	local out_file="$WORKDIR/verdict-${label}.json"
	local err_file="$WORKDIR/verdict-${label}.err"
	local code=0
	"$ADMIT_SCRIPT" "$package_yaml" >"$out_file" 2>"$err_file" || code=$?

	[[ "$code" -eq 2 ]] || fail "$label: admit-scenario.sh exited $code (expected 2 for a containment violation); stdout: $(cat "$out_file"); stderr: $(cat "$err_file")"
	[[ ! -s "$out_file" ]] || fail "$label: admit-scenario.sh printed a verdict to stdout despite a containment violation (expected none): $(cat "$out_file")"
	grep -qF -- "$expected_substring" "$err_file" || fail "$label: stderr did not mention $expected_substring: $(cat "$err_file")"
	echo "TC-033: $label rejected as a containment violation (exit 2), stderr names it"
}

CHECK_A="a_runnable_base_fixture"
CHECK_B="b_p2p_selection_at_base"
CHECK_C="c_stage_matrix_invariant"
CHECK_D="d_predicate_false_at_base"
CHECK_E="e_predicate_true_after_reference"
CHECK_F="f_resource_policy"

echo "TC-033: (a) adapter build capability fails at base_sha"

cat >"$WORKDIR/mutator-a.py" <<'PYEOF'
import sys

worktree = sys.argv[1]
with open(f"{worktree}/taskmanager/manager.py", "a") as f:
    f.write("\ndef _tc033_broken_build_syntax_error(:\n")
PYEOF
BROKEN_SHA_A="$(make_broken_fixture_sha a "$WORKDIR/mutator-a.py")"
CAND_A="$(mk_candidate a)"
set_base_sha "$CAND_A" "$BROKEN_SHA_A"
assert_rejected "(a) build fails" "$CAND_A" "$CHECK_A"

echo "TC-033: (b-empty) p2p_selection.include resolves to an empty entry set"

CAND_B_EMPTY="$(mk_candidate b_empty)"
python3 - "$CAND_B_EMPTY" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# "taskmanager" is a real path under the checkout with zero test_*.py files --
# pytest legitimately collects nothing (adapter.sh's own documented "no tests
# ran" -> empty {"entries": []} case), not an include path that fails to
# resolve at all.
data["final_predicate"]["p2p_selection"]["include"] = ["taskmanager"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_rejected "(b-empty) p2p_selection empty at base" "$CAND_B_EMPTY" "$CHECK_B"

echo "TC-033: (b-fail) p2p_selection resolves to a non-empty set with a failing entry at base"

cat >"$WORKDIR/mutator-b-fail.py" <<'PYEOF'
import sys

worktree = sys.argv[1]
path = f"{worktree}/taskmanager/priority.py"
with open(path) as f:
    content = f.read()
# Adding "urgent" to the closed set of accepted priority levels breaks
# exactly test_priority.py::test_validate_priority_rejects_unknown_level
# (it stops raising for "urgent") -- a real, non-empty p2p_selection entry,
# with no cascade into any test relying on the "high"/"medium"/"low" values
# themselves (e.g. add_task's own "medium" default) and no effect on lint
# or any other predicate operand.
old = 'PRIORITY_LEVELS = ("high", "medium", "low")'
new = 'PRIORITY_LEVELS = ("high", "medium", "low", "urgent")'
if old not in content:
    sys.exit("mutator-b-fail: priority.py PRIORITY_LEVELS tuple did not match the expected shape -- mutation target moved")
content = content.replace(old, new, 1)
with open(path, "w") as f:
    f.write(content)
PYEOF
BROKEN_SHA_B_FAIL="$(make_broken_fixture_sha b-fail "$WORKDIR/mutator-b-fail.py")"
CAND_B_FAIL="$(mk_candidate b_fail)"
set_base_sha "$CAND_B_FAIL" "$BROKEN_SHA_B_FAIL"
assert_rejected "(b-fail) p2p_selection has a failing entry at base" "$CAND_B_FAIL" "$CHECK_B"

echo "TC-033: (c) stage_matrix violates the REQ-F-004 family invariant"

CAND_C="$(mk_candidate c)"
python3 - "$CAND_C" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# entity_family is "bug" -- REQ-F-004 requires every prelude stage false.
# Flip D02 to true (with its 'reason' left in place) to violate the invariant.
data["stage_matrix"]["prelude"]["D02"]["applicable"] = True
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_rejected "(c) stage_matrix family invariant violated" "$CAND_C" "$CHECK_C"

echo "TC-033: (d) final_predicate operands loosened so the predicate is already true at base"

CAND_D="$(mk_candidate d)"
python3 - "$CAND_D" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# Repoint f2p_test_ids at a real fixture test that already passes at base
# (unrelated to the due-date boundary bug), instead of the held-back repro --
# named_ok is then true at base with no fixture change at all.
data["final_predicate"]["f2p_test_ids"] = ["tests.test_due_date::test_is_overdue_true_for_past_date"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_rejected "(d) final_predicate already true at base" "$CAND_D" "$CHECK_D"

echo "TC-033: (e) final_predicate stays false after the reference solution is applied"

CAND_E="$(mk_candidate e)"
CAND_E_DIR="$(dirname "$CAND_E")"
# Corrupt the reference patch so it still applies cleanly (real context lines,
# real hunk) but touches only the docstring -- never the strict "<" -> "<="
# comparison the bug actually requires -- so the repro test the predicate
# names is still red after "the reference solution" is applied.
cat >"$CAND_E_DIR/evaluator/reference.patch" <<'PATCHEOF'
diff --git a/taskmanager/due_date.py b/taskmanager/due_date.py
index 8b9dcbc..1111111 100644
--- a/taskmanager/due_date.py
+++ b/taskmanager/due_date.py
@@ -13,7 +13,7 @@ def parse_due_date(value: str) -> date:


 def is_overdue(due_date: str, today: date) -> bool:
-    """Return True if due_date is strictly before today."""
+    """Return True if due_date is strictly before today (TC-033 corrupted reference: docstring-only edit, behavior unchanged)."""
     return parse_due_date(due_date) < today


PATCHEOF
assert_rejected "(e) final_predicate stays false after reference" "$CAND_E" "$CHECK_E"

echo "TC-033: (f) resource_policy.max_cost_usd is non-positive"

CAND_F="$(mk_candidate f)"
python3 - "$CAND_F" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["resource_policy"]["max_cost_usd"] = 0
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_rejected "(f) resource_policy.max_cost_usd non-positive" "$CAND_F" "$CHECK_F"

# -----------------------------------------------------------------------------
# UAT Finding 1 rework (T-E40-F05-009 kickback): path-containment gap at the
# evaluator-isolation boundary. admit-scenario.sh joined package-declared
# relative paths (evaluator_only.oracle_tests[], evaluator_only.reference_
# solution) directly with the package directory and fed the result into
# inject-tests / `git apply` with no check that the resolved path stayed
# inside the package's own evaluator/ subtree or the repository tree at all.
# Candidates (g)-(n) below each mutate exactly one package-declared relative-
# path field to attempt an escape (absolute path, `../` traversal, or a
# symlink pointing outside the required root) and must be rejected as a
# containment violation (exit 2) BEFORE the offending path is ever read,
# copied, or applied.
#
# Candidate (o) (round-2 code-review kickback, T-E40-F05-009): a distinct
# defect in resolve_scoped's subtree check, found in the (g)-(n) rework
# itself. When a package-declared "evaluator" subtree collapses onto the
# package root (e.g. a same-named file/symlink at the root resolving back
# onto base_real), the old check treated root_real == base_real as
# "nothing further to check" instead of a rejection -- silently downgrading
# the evaluator-subtree restriction to a package-root restriction, so any
# file anywhere in the package became reachable through it.
# -----------------------------------------------------------------------------

echo "TC-033: (g) evaluator_only.oracle_tests[] declares an absolute path"

CAND_G="$(mk_candidate g)"
python3 - "$CAND_G" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["evaluator_only"]["oracle_tests"] = ["/etc/passwd"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(g) oracle_tests absolute path" "$CAND_G" "absolute path not allowed"

echo "TC-033: (h) evaluator_only.oracle_tests[] traverses out of the package root entirely"

CAND_H="$(mk_candidate h)"
python3 - "$CAND_H" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["evaluator_only"]["oracle_tests"] = ["../../../../../../../../../../etc/passwd"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(h) oracle_tests traversal escapes package root" "$CAND_H" "resolved path escapes"

echo "TC-033: (i) evaluator_only.oracle_tests[] resolves inside the package but outside evaluator/"

CAND_I="$(mk_candidate i)"
python3 - "$CAND_I" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# input/prompt.md is a real file in this package -- just not under evaluator/.
data["evaluator_only"]["oracle_tests"] = ["input/prompt.md"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(i) oracle_tests outside evaluator subtree" "$CAND_I" "resolved path escapes"

echo "TC-033: (j) evaluator_only.reference_solution declares an absolute path"

CAND_J="$(mk_candidate j)"
python3 - "$CAND_J" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["evaluator_only"]["reference_solution"] = "/etc/passwd"
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(j) reference_solution absolute path" "$CAND_J" "absolute path not allowed"

echo "TC-033: (k) evaluator_only.reference_solution traverses out of the package root entirely"

CAND_K="$(mk_candidate k)"
python3 - "$CAND_K" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["evaluator_only"]["reference_solution"] = "../../../../../../../../../../etc/passwd"
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(k) reference_solution traversal escapes package root" "$CAND_K" "resolved path escapes"

echo "TC-033: (l) evaluator_only.reference_solution resolves inside the package but outside evaluator/"

CAND_L="$(mk_candidate l)"
python3 - "$CAND_L" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# input/prompt.md is a real file in this package -- just not under evaluator/.
data["evaluator_only"]["reference_solution"] = "input/prompt.md"
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(l) reference_solution outside evaluator subtree" "$CAND_L" "resolved path escapes"

echo "TC-033: (m) evaluator_only.oracle_tests[] names a symlink under evaluator/ that resolves outside the package"

CAND_M="$(mk_candidate m)"
CAND_M_DIR="$(dirname "$CAND_M")"
EXTERNAL_TARGET="$WORKDIR/tc033-m-external-secret.py"
echo "# not a real oracle test -- containment must reject this before it is ever read" >"$EXTERNAL_TARGET"
ln -s "$EXTERNAL_TARGET" "$CAND_M_DIR/evaluator/escape-link.py"
python3 - "$CAND_M" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["evaluator_only"]["oracle_tests"] = ["evaluator/escape-link.py"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(m) oracle_tests symlink escapes evaluator subtree" "$CAND_M" "resolved path escapes"

echo "TC-033: (n) final_predicate.p2p_selection.include[] traverses out of the checkout root"

CAND_N="$(mk_candidate n)"
python3 - "$CAND_N" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
data["final_predicate"]["p2p_selection"]["include"] = ["../../../../../../../../../../etc"]
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
assert_containment_rejected "(n) p2p_selection.include traversal escapes checkout root" "$CAND_N" "resolved path escapes"

echo "TC-033: (o) evaluator subtree collapses onto the package root via a same-named symlink (degenerate containment bypass)"

CAND_O="$(mk_candidate o)"
CAND_O_DIR="$(dirname "$CAND_O")"
# Relocate the real evaluator/ file pair to the package root as plain,
# non-symlinked copies first -- adapter.sh's inject-tests keys off
# basename only (bench/adapters/python/adapter.sh cmd_inject_tests), so
# relocating the source changes nothing about where it lands in the
# checkout -- then delete the real evaluator/ directory and replace it
# with a same-named symlink pointing at "." (the package root itself).
# os.path.realpath(package_dir/evaluator) then collapses to
# realpath(package_dir): the exact degenerate case resolve_scoped's
# subtree check must reject on its own, before the final containment
# comparison ever runs.
#
# The symlink is created LAST, strictly after every cp above it -- any
# earlier command reordered to run through "evaluator" would recurse
# evaluator -> . -> evaluator -> ... forever, since it is now a self-
# referential symlink; only the ordering below keeps this candidate safe
# for a `cp -r`/`find` traversal (this test's own cleanup `rm -rf` unlinks
# without descending, so cleanup is unaffected either way).
cp "$CAND_O_DIR/evaluator/reference.patch" "$CAND_O_DIR/reference.patch"
cp "$CAND_O_DIR/evaluator/test_due_date_boundary.py" "$CAND_O_DIR/test_due_date_boundary.py"
rm -rf "$CAND_O_DIR/evaluator"
ln -s . "$CAND_O_DIR/evaluator"
python3 - "$CAND_O" <<'PYEOF'
import sys
import yaml

path = sys.argv[1]
with open(path) as f:
    data = yaml.safe_load(f)
# Both evaluator_only.* fields are re-pointed through the collapsed
# "evaluator" symlink at content byte-identical to the real
# evaluator/reference.patch and evaluator/test_due_date_boundary.py --
# proving the bypass is about containment, not content: the same bytes
# that legitimately live in evaluator/ are now reachable via a path the
# subtree restriction was supposed to make unreachable from anywhere else
# in the package (e.g. package.yaml itself, or input/prompt.md).
data["evaluator_only"]["oracle_tests"] = ["evaluator/test_due_date_boundary.py"]
data["evaluator_only"]["reference_solution"] = "evaluator/reference.patch"
with open(path, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PYEOF
# oracle_tests[0] resolves first inside admit-scenario.sh's evaluate(), so
# the ScriptError fires on the "evaluator_only.oracle_tests[0]" label
# before reference_solution's own resolve_scoped call ever runs -- assert
# on the field-agnostic collapse phrase, not on either field name.
assert_containment_rejected "(o) evaluator subtree collapses onto package root via symlink" "$CAND_O" "collapses onto or escapes"

echo "TC-033: positive control - an unmutated copy is admitted (proves the seven rejections above are meaningful, not a gate that always rejects)"

CAND_BASELINE="$(mk_candidate baseline)"
baseline_out="$WORKDIR/verdict-baseline.json"
baseline_code=0
"$ADMIT_SCRIPT" "$CAND_BASELINE" >"$baseline_out" 2>"$WORKDIR/verdict-baseline.err" || baseline_code=$?
[[ "$baseline_code" -eq 0 ]] || fail "positive control: admit-scenario.sh exited $baseline_code (expected 0 for an admitted candidate); stderr: $(cat "$WORKDIR/verdict-baseline.err")"

python3 - "$baseline_out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    verdict = json.loads(f.read().strip())

if verdict["status"] != "admitted":
    sys.exit(f"TC-033 FAIL: positive control: expected status 'admitted', got {verdict['status']!r}: {verdict}")
if verdict["failing_check"] is not None:
    sys.exit(f"TC-033 FAIL: positive control: expected failing_check None, got {verdict['failing_check']!r}: {verdict}")
if not all(verdict["checks"].values()):
    sys.exit(f"TC-033 FAIL: positive control: expected every check true, got {verdict['checks']}: {verdict}")
if verdict["base_outcome"] is not False:
    sys.exit(f"TC-033 FAIL: positive control: expected base_outcome false, got {verdict['base_outcome']!r}: {verdict}")
if verdict["reference_outcome"] is not True:
    sys.exit(f"TC-033 FAIL: positive control: expected reference_outcome true, got {verdict['reference_outcome']!r}: {verdict}")
print(f"TC-033: positive control admitted with every check true, base_outcome=false, reference_outcome=true: {verdict}")
PYEOF

echo "TC-033: positive control - admission: block written into the copy, cross-encoding matches (AC-019), and a second run is byte-identical (idempotency, REQ-NF-004)"

cp "$CAND_BASELINE" "$WORKDIR/baseline-run1.yaml"
"$ADMIT_SCRIPT" "$CAND_BASELINE" >/dev/null

python3 - "$CAND_BASELINE" <<'PYEOF'
import sys

import yaml

path = sys.argv[1]
with open(path) as f:
    package = yaml.safe_load(f)

admission = package.get("admission")
if not isinstance(admission, dict):
    sys.exit("TC-033 FAIL: positive control: package.yaml carries no 'admission' block after an admitted run")
if admission.get("status") != "admitted":
    sys.exit(f"TC-033 FAIL: positive control: admission.status is not 'admitted': {admission}")
if package.get("toolchain_identity") != admission.get("toolchain_identity"):
    sys.exit(
        "TC-033 FAIL: positive control (AC-019): top-level toolchain_identity does not equal "
        f"admission.toolchain_identity element-for-element: {package.get('toolchain_identity')} != {admission.get('toolchain_identity')}"
    )
print("TC-033: positive control admission: block present, status admitted, toolchain_identity cross-encoding matches (AC-019)")
PYEOF

diff -q "$WORKDIR/baseline-run1.yaml" "$CAND_BASELINE" >/dev/null \
	|| fail "positive control: package.yaml differs after a second admit-scenario.sh run on an unchanged checkout/toolchain (REQ-NF-004 byte-identical idempotency violated)"
echo "TC-033: positive control: second run produced a byte-identical package.yaml (idempotent, REQ-NF-004)"

echo "TC-033: verify bench/fixture-py's checked-out state is unchanged by this test's scratch branches"

FIXTURE_STATE_AFTER="$(git -C "$FIXTURE_SUBMODULE" rev-parse HEAD) $(git -C "$FIXTURE_SUBMODULE" status --porcelain)"
[[ "$FIXTURE_STATE_BEFORE" == "$FIXTURE_STATE_AFTER" ]] || fail "bench/fixture-py's checked-out HEAD/working tree changed by this test (before: $FIXTURE_STATE_BEFORE; after: $FIXTURE_STATE_AFTER)"

# This run's own scratch branches are still present here by design (this
# script's own cleanup trap, registered above, deletes them on EXIT -- after
# this point, not before) -- the state check above is what actually matters:
# bench/fixture-py's checked-out HEAD and working tree were never touched.

echo "TC-033: PASS"
