#!/usr/bin/env bash
# TC-043 (test-plan.md AC test matrix; T-E40-F06-003 task spec Test Cases,
# AC-T1/AC-T2/AC-T3).
#
# Drives bench/scripts/verify-evidence-roots.sh against a real
# checkout-scenario-fixture.sh checkout of py-feature-recurring-tasks
# (REQ-F-010, REQ-F-011; AC-009, AC-010):
#   (0) clean-roots-all-packages: EVERY committed I-04 package (all four
#       entity families, not just py-feature-recurring-tasks) verifies
#       clean at its own frozen base_sha with an empty scratch project --
#       REQ-F-010 says "for each I-04 scenario package", and the base
#       fixture's own committed content (shared across seeds, including
#       skip-marked placeholders for held-back capabilities) is exactly
#       where a naive detection signal can false-positive, so this is a
#       real per-package proof, not a single happy-path sample.
#   (c) clean roots exit 0 ("CLEAN") -- the positive control that makes the
#       failing cases below meaningful evidence rather than "always fails".
#   (a) the evaluator-only reference_solution file planted inside the
#       fixture checkout fails naming that root (AC-009's literal planted
#       source: "an evaluator-only file (from the package's
#       evaluator_only.reference_solution)").
#   (b) the SAME file planted inside the scratch project instead fails
#       naming that root -- proving a guard walking only --workdir cannot
#       pass this case.
#   (d) case (a) repeated with a PATH-stubbed `claude` binary (the X-07
#       provider-invocation stand-in, testdata/stubs/claude, reused rather
#       than inventing a second stub) recording every invocation to a log
#       file: the log is empty after the guard's failing exit, so "before
#       provider spend" is observed via a non-invoked dispatcher rather than
#       merely inferred from the guard's own non-zero exit status.
#   (e) an oracle_tests[] source planted in the fixture checkout also fails
#       -- REQ-F-010 names reference_solution, oracle_tests[], and
#       answer_keys[] as three independent source kinds; (a)/(b)/(d) alone
#       would leave oracle_tests[] detection unexercised outside AC-010's
#       narrower (byte-identical-content) renaming cases.
#   AC-010: the admission-time check invoked against two scratch copies of
#       the same package -- one with the original oracle_tests[] entry name,
#       one with that entry renamed on disk AND in the scratch copy's
#       package.yaml (no edit to the guard script) -- both detect a
#       CONTENT-MODIFIED leak of their own (renamed or original) name. The
#       leak's content deliberately differs from the real evaluator file
#       (content_digest would not fire), so only a guard that derives the
#       search NAME from package.yaml at call time -- not one with the
#       original name hardcoded or grepped in -- can catch either case via
#       the filename signal.
#
# Also exercises the committed, checkout-free "root-policy" fixtures under
# bench/scripts/testdata/evidence/root-policy/ (this task's own scope item)
# against the same guard -- covering all three evaluator_only source kinds
# (reference_solution, oracle_tests[], answer_keys[]; no real committed I-04
# package declares a non-empty answer_keys[], so this fixture's synthetic
# entry is the only exerciser of that path) -- both to prove those fixtures
# are internally consistent today (rather than sitting unverified until
# T-E40-F06-011's offline-determinism proof consumes them) and as an offline
# complement to the real-checkout cases above.
#
# Caller-Path Contract (test-plan.md TC-043): real filesystem walk of two
# live roots; real content-digest computation. The planted-leak file always
# actually exists on disk at the path under test -- never a metadata flag.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-evidence-roots.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
REAL_PACKAGE_DIR="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks"
FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
OFFLINE_FIXTURES="$SCRIPTS_DIR/testdata/evidence/root-policy"

fail() {
	echo "TC-043 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-evidence-roots.sh missing or not executable: $GUARD"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
[[ -f "$REAL_PACKAGE_DIR/package.yaml" ]] || fail "py-feature-recurring-tasks/package.yaml missing"
[[ -e "$FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Real fixture checkout (REQ-F-010's "fresh checkout of the package's
# fixture at fixture.base_sha") + a plain scratch-project directory. Never
# a shark-initialized project (REQ-NF-005): the "scratch Shark project" role
# here is just a second agent-visible directory tree for the guard to walk.
# ---------------------------------------------------------------------------
read_pkg_field() {
	# read_pkg_field <package_yaml> <dotted.path>
	python3 - "$1" "$2" <<'PYEOF'
import sys
import yaml

path, dotted = sys.argv[1:3]
with open(path) as f:
    data = yaml.safe_load(f)
node = data
for part in dotted.split("."):
    node = node[part]
print(node)
PYEOF
}

FIXTURE_ID="$(read_pkg_field "$REAL_PACKAGE_DIR/package.yaml" fixture.fixture_id)"
BASE_SHA="$(read_pkg_field "$REAL_PACKAGE_DIR/package.yaml" fixture.base_sha)"
[[ -n "$FIXTURE_ID" && -n "$BASE_SHA" ]] || fail "could not derive fixture.fixture_id/base_sha from the real package.yaml"

FIXTURE_CHECKOUT="$WORKDIR/fixture-checkout"
"$CHECKOUT_SCRIPT" "$FIXTURE_ID" "$BASE_SHA" "$FIXTURE_CHECKOUT" || fail "checkout-scenario-fixture.sh failed to provision the fixture checkout"

SCRATCH_PROJECT="$WORKDIR/scratch-project"
mkdir -p "$SCRATCH_PROJECT"
echo "unrelated scratch-project marker file" >"$SCRATCH_PROJECT/README.md"

# mk_candidate <label> -- copies the real, committed package into a scratch
# directory this test is free to mutate (mirrors TC-033's mk_candidate;
# never runs the guard against the live committed package.yaml).
mk_candidate() {
	local label="$1"
	local dest="$WORKDIR/candidate-$label"
	cp -r "$REAL_PACKAGE_DIR" "$dest"
	echo "$dest"
}

run_guard() {
	# run_guard <package_yaml> <fixture_checkout> <scratch_project> <evaluator_root>
	"$GUARD" "$1" "$2" "$3" "$4"
}

# ---------------------------------------------------------------------------
# Case (0): clean-roots-all-packages. REQ-F-010 binds "for each I-04
# scenario package", not just py-feature-recurring-tasks -- and the base
# fixture's own committed content (shared across seeds: skip-marked
# placeholders, docstrings, other packages' own held-back-file names
# mentioned in comments) is exactly where a detection signal that is too
# broad would false-positive on a real, otherwise-clean checkout.
# Iterates scenarios.yaml's own registered scenarios: list (never a
# hardcoded package-name list, mirroring TC-032's own iteration), so a fifth
# registered scenario is covered automatically.
# ---------------------------------------------------------------------------
echo "TC-043: case (0) - clean roots verified for every registered I-04 package"

SCENARIOS_YAML="$BENCH_DIR/scenarios/scenarios.yaml"
[[ -f "$SCENARIOS_YAML" ]] || fail "scenarios.yaml not found: $SCENARIOS_YAML"

mapfile -t ALL_PACKAGE_RELDIRS < <(python3 - "$SCENARIOS_YAML" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for rel in data.get("scenarios") or []:
    print(rel)
PYEOF
)
[[ "${#ALL_PACKAGE_RELDIRS[@]}" -gt 0 ]] || fail "case (0): scenarios.yaml declares no scenarios"

for reldir in "${ALL_PACKAGE_RELDIRS[@]}"; do
	pkg_dir="$BENCH_DIR/scenarios/$reldir"
	pkg_name="$(basename "$reldir")"
	[[ -f "$pkg_dir/package.yaml" ]] || fail "case (0) ($pkg_name): package.yaml missing under $pkg_dir"

	pkg_fixture_id="$(read_pkg_field "$pkg_dir/package.yaml" fixture.fixture_id)"
	pkg_base_sha="$(read_pkg_field "$pkg_dir/package.yaml" fixture.base_sha)"
	[[ -n "$pkg_fixture_id" && -n "$pkg_base_sha" ]] || fail "case (0) ($pkg_name): could not derive fixture.fixture_id/base_sha"

	pkg_checkout="$WORKDIR/case0-$pkg_name-checkout"
	"$CHECKOUT_SCRIPT" "$pkg_fixture_id" "$pkg_base_sha" "$pkg_checkout" || fail "case (0) ($pkg_name): checkout-scenario-fixture.sh failed"
	pkg_scratch="$WORKDIR/case0-$pkg_name-scratch"
	mkdir -p "$pkg_scratch"

	out="$WORKDIR/case0-$pkg_name.out"
	err="$WORKDIR/case0-$pkg_name.err"
	set +e
	run_guard "$pkg_dir/package.yaml" "$pkg_checkout" "$pkg_scratch" "$pkg_dir" >"$out" 2>"$err"
	code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "case (0) ($pkg_name): guard exited $code on this package's own clean base_sha checkout, want 0: $(cat "$err")"
	[[ "$(cat "$out")" == "CLEAN" ]] || fail "case (0) ($pkg_name): guard stdout was $(cat "$out"), want CLEAN"
done

echo "TC-043(case 0: all ${#ALL_PACKAGE_RELDIRS[@]} registered packages verify clean at their own base_sha) PASS"

# ---------------------------------------------------------------------------
# Case (c): clean roots.
# ---------------------------------------------------------------------------
echo "TC-043: case (c) - clean roots exit 0"

CANDIDATE_DIR="$(mk_candidate clean)"
out="$WORKDIR/c.out"
err="$WORKDIR/c.err"
set +e
run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (c): guard exited $code on clean roots, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "case (c): guard stdout was $(cat "$out"), want CLEAN"

echo "TC-043(case c: clean roots -> exit 0, CLEAN) PASS"

# ---------------------------------------------------------------------------
# Case (a): the evaluator-only reference_solution file planted in the
# fixture checkout -- AC-009's literal planted source.
# ---------------------------------------------------------------------------
echo "TC-043: case (a) - evaluator_only.reference_solution planted in fixture checkout"

REFERENCE_SOLUTION_SRC="$CANDIDATE_DIR/evaluator/reference.patch"
[[ -f "$REFERENCE_SOLUTION_SRC" ]] || fail "candidate's evaluator/reference.patch missing: $REFERENCE_SOLUTION_SRC"
ORACLE_TEST_SRC="$CANDIDATE_DIR/evaluator/test_recurring.py"
[[ -f "$ORACLE_TEST_SRC" ]] || fail "candidate's evaluator/test_recurring.py missing: $ORACLE_TEST_SRC"

PLANTED_A="$FIXTURE_CHECKOUT/leaked_reference.patch"
cp "$REFERENCE_SOLUTION_SRC" "$PLANTED_A"

out="$WORKDIR/a.out"
err="$WORKDIR/a.err"
set +e
run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (a): guard exited 0 with a planted leak in the fixture checkout, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "case (a): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_A" "$err" || fail "case (a): failure message does not name the planted path: $(cat "$err")"
grep -q "source=evaluator_only.reference_solution" "$err" || fail "case (a): failure message does not name the matched evaluator_only source: $(cat "$err")"

rm -f "$PLANTED_A"
echo "TC-043(case a: reference_solution leak in fixture checkout -> fails naming root/path/source) PASS"

# ---------------------------------------------------------------------------
# Case (b): the SAME evaluator-only file planted in the scratch project
# instead -- a guard that only walks --workdir (the fixture checkout) would
# pass this case; TC-043 requires it to independently fail.
# ---------------------------------------------------------------------------
echo "TC-043: case (b) - evaluator_only.reference_solution planted in scratch project"

PLANTED_B="$SCRATCH_PROJECT/leaked_reference.patch"
cp "$REFERENCE_SOLUTION_SRC" "$PLANTED_B"

out="$WORKDIR/b.out"
err="$WORKDIR/b.err"
set +e
run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (b): guard exited 0 with a planted leak in the scratch project, want non-zero"
grep -q "root=scratch_shark_project" "$err" || fail "case (b): failure message does not name the scratch-project root: $(cat "$err")"
grep -qF "$PLANTED_B" "$err" || fail "case (b): failure message does not name the planted path: $(cat "$err")"

rm -f "$PLANTED_B"
echo "TC-043(case b: reference_solution leak in scratch project -> fails naming that root, not just fixture checkout) PASS"

# ---------------------------------------------------------------------------
# Case (e): an oracle_tests[] source also independently detected -- REQ-F-010
# names reference_solution, oracle_tests[], and answer_keys[] as three
# distinct source kinds; without this case only AC-010's byte-identical-
# content renaming scenario would exercise oracle_tests[] at all.
# ---------------------------------------------------------------------------
echo "TC-043: case (e) - evaluator_only.oracle_tests[] planted in fixture checkout"

PLANTED_E="$FIXTURE_CHECKOUT/leaked_test_recurring.py"
cp "$ORACLE_TEST_SRC" "$PLANTED_E"

out="$WORKDIR/e.out"
err="$WORKDIR/e.err"
set +e
run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (e): guard exited 0 with a planted oracle_tests leak, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "case (e): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_E" "$err" || fail "case (e): failure message does not name the planted path: $(cat "$err")"
grep -q "source=evaluator_only.oracle_tests\[0\]" "$err" || fail "case (e): failure message does not name the matched evaluator_only source: $(cat "$err")"

rm -f "$PLANTED_E"
echo "TC-043(case e: oracle_tests[] leak in fixture checkout -> fails naming root/path/source) PASS"

# ---------------------------------------------------------------------------
# Case (d): case (a) repeated with a PATH-stubbed `claude` binary recording
# invocations -- proves "before provider spend" via a non-invoked dispatcher,
# not merely inferred from the guard's own exit status.
# ---------------------------------------------------------------------------
echo "TC-043: case (d) - failing path issues zero dispatcher invocations"

PLANTED_D="$FIXTURE_CHECKOUT/leaked_reference_d.patch"
cp "$REFERENCE_SOLUTION_SRC" "$PLANTED_D"
CLAUDE_LOG="$WORKDIR/claude-invocations.jsonl"
rm -f "$CLAUDE_LOG"

out="$WORKDIR/d.out"
err="$WORKDIR/d.err"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$CLAUDE_LOG" \
	run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (d): guard exited 0 with a planted leak, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "case (d): failure message does not name the fixture-checkout root: $(cat "$err")"
[[ ! -s "$CLAUDE_LOG" ]] || fail "case (d): stubbed dispatcher recorded $(wc -l <"$CLAUDE_LOG") invocation(s) on the failing path, want zero: $(cat "$CLAUDE_LOG")"

rm -f "$PLANTED_D"
echo "TC-043(case d: failing path -> zero dispatcher invocations, observed not inferred) PASS"

# ---------------------------------------------------------------------------
# AC-010: dynamic name derivation. Two independent scratch copies -- one
# left with the package's original oracle_tests[] entry name, one with that
# entry renamed on disk AND in package.yaml -- prove the guard's search
# target tracks whichever name the package under test declares, with no
# edit to verify-evidence-roots.sh itself.
# ---------------------------------------------------------------------------
echo "TC-043: AC-010 - renamed oracle_tests[] entry changes the guard's search target"

ORIGINAL_CANDIDATE="$(mk_candidate ac010-original)"
RENAMED_CANDIDATE="$(mk_candidate ac010-renamed)"

ORIGINAL_NAME="test_recurring.py"
RENAMED_NAME="test_recurring_renamed_by_tc043.py"
mv "$RENAMED_CANDIDATE/evaluator/$ORIGINAL_NAME" "$RENAMED_CANDIDATE/evaluator/$RENAMED_NAME"
python3 - "$RENAMED_CANDIDATE/package.yaml" "evaluator/$ORIGINAL_NAME" "evaluator/$RENAMED_NAME" <<'PYEOF'
import sys

import yaml

path, old, new = sys.argv[1:4]
with open(path) as f:
    data = yaml.safe_load(f)
oracle_tests = data["evaluator_only"]["oracle_tests"]
assert old in oracle_tests, f"{old!r} not found in oracle_tests: {oracle_tests!r}"
data["evaluator_only"]["oracle_tests"] = [new if entry == old else entry for entry in oracle_tests]
with open(path, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF

assert_dynamic_detection() {
	# assert_dynamic_detection <label> <candidate_dir> <leaked_basename>
	#
	# Plants a CONTENT-MODIFIED copy (one appended marker line, so its
	# sha256 digest differs from the real evaluator_only file's digest)
	# under the target basename. This is deliberate: a byte-identical copy
	# would be caught via the content_digest signal regardless of whether
	# the guard derives the search NAME correctly, which would let a guard
	# with the original name hardcoded pass this case by accident (the
	# digest check doesn't care about names at all). With digest matching
	# ruled out, only the filename signal can catch the leak -- and that
	# signal only fires if the guard's target list actually contains
	# <leaked_basename>, which requires reading it from package.yaml at
	# call time rather than from a hardcoded/grepped-in literal.
	# leak_target is a full copy of the real fixture checkout, not a bare
	# directory holding only the planted file: the guard's derived_test_identity
	# signal (T-E40-F06-003 round-4 UAT fix) resolves the DECLARED oracle_tests[]
	# entry's real test identity by asking the package's adapter to import and
	# collect it against fixture_checkout, and the real oracle file imports
	# from the fixture's own taskmanager package -- a bare directory would
	# make that import fail closed (ScriptError), not exercise this case.
	local label="$1" candidate_dir="$2" leaked_basename="$3"
	local leak_target="$WORKDIR/ac010-$label-leak"
	cp -r "$FIXTURE_CHECKOUT" "$leak_target"
	{
		cat "$candidate_dir/evaluator/$leaked_basename"
		echo "# TC-043 AC-010 ($label): content-modified marker, digest deliberately differs"
	} >"$leak_target/$leaked_basename"

	local real_digest planted_digest
	real_digest="$(sha256sum "$candidate_dir/evaluator/$leaked_basename" | awk '{print $1}')"
	planted_digest="$(sha256sum "$leak_target/$leaked_basename" | awk '{print $1}')"
	[[ "$real_digest" != "$planted_digest" ]] || fail "AC-010 ($label): test setup bug -- planted file's digest was not actually modified"

	local out="$WORKDIR/ac010-$label.out" err="$WORKDIR/ac010-$label.err"
	set +e
	run_guard "$candidate_dir/package.yaml" "$leak_target" "$SCRATCH_PROJECT" "$candidate_dir" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "AC-010 ($label): guard exited 0 with a planted, content-modified $leaked_basename leak, want non-zero"
	grep -qF "$leaked_basename" "$err" || fail "AC-010 ($label): failure message does not mention the (renamed-or-original) leaked name $leaked_basename: $(cat "$err")"
	grep -q "match_kind=filename" "$err" || fail "AC-010 ($label): failure message's match_kind was not 'filename' -- content differs from the real evaluator file, so only the filename signal should have fired: $(cat "$err")"
}

assert_dynamic_detection original "$ORIGINAL_CANDIDATE" "$ORIGINAL_NAME"
assert_dynamic_detection renamed "$RENAMED_CANDIDATE" "$RENAMED_NAME"

# Negative half of AC-010: a guard with the ORIGINAL name hardcoded (grepped
# in as a literal string) would fail to detect a leak of the file under its
# RENAMED name -- both because its target list would never contain
# leaked_basename=test_recurring_renamed_by_tc043.py, and (per
# assert_dynamic_detection's content modification above) because the digest
# signal is deliberately unavailable as a fallback. Prove the real guard is
# NOT doing that by asserting the failure message's source label itself
# cites the renamed declared path, not the stale original one.
grep -q "evaluator/$RENAMED_NAME" "$WORKDIR/ac010-renamed.err" || fail "AC-010 (renamed): failure message's source label does not cite the renamed declared path evaluator/$RENAMED_NAME -- guard may be using a stale/hardcoded name: $(cat "$WORKDIR/ac010-renamed.err")"

echo "TC-043(AC-010: renamed and original oracle_tests[] entries both detected via package-derived names, filename signal only) PASS"

# ---------------------------------------------------------------------------
# Committed offline fixtures (this task's own scope item): the three
# checkout-free root pairs under bench/scripts/testdata/evidence/root-policy/
# reproduce the clean/leak-in-fixture-checkout/leak-in-scratch-project shapes
# above without a submodule -- verified here so they are proven correct now,
# not left unverified until T-E40-F06-011's offline-determinism proof
# (AC-016) is the first consumer.
# ---------------------------------------------------------------------------
echo "TC-043: committed offline fixtures (bench/scripts/testdata/evidence/root-policy/)"

[[ -f "$OFFLINE_FIXTURES/package/package.yaml" ]] || fail "offline fixtures: package/package.yaml missing under $OFFLINE_FIXTURES"

out="$WORKDIR/offline-clean.out"
err="$WORKDIR/offline-clean.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$OFFLINE_FIXTURES/clean/fixture-checkout" "$OFFLINE_FIXTURES/clean/scratch-project" "$OFFLINE_FIXTURES/package" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "offline fixtures (clean): guard exited $code, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "offline fixtures (clean): guard stdout was $(cat "$out"), want CLEAN"

err="$WORKDIR/offline-leak-fixture.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$OFFLINE_FIXTURES/leak-in-fixture-checkout/fixture-checkout" "$OFFLINE_FIXTURES/leak-in-fixture-checkout/scratch-project" "$OFFLINE_FIXTURES/package" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "offline fixtures (leak-in-fixture-checkout): guard exited 0, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "offline fixtures (leak-in-fixture-checkout): failure message does not name the fixture-checkout root: $(cat "$err")"

err="$WORKDIR/offline-leak-scratch.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$OFFLINE_FIXTURES/leak-in-scratch-project/fixture-checkout" "$OFFLINE_FIXTURES/leak-in-scratch-project/scratch-project" "$OFFLINE_FIXTURES/package" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "offline fixtures (leak-in-scratch-project): guard exited 0, want non-zero"
grep -q "root=scratch_shark_project" "$err" || fail "offline fixtures (leak-in-scratch-project): failure message does not name the scratch-project root: $(cat "$err")"

# The two above both plant oracle_tests[] sources; these two additionally
# cover the other two evaluator_only source kinds offline -- reference_solution
# (planted in the fixture checkout) and answer_keys[] (planted in the scratch
# project), so all three source kinds have an offline, checkout-free exerciser
# too, not only a real-checkout one.
err="$WORKDIR/offline-leak-reference-solution.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$OFFLINE_FIXTURES/leak-reference-solution/fixture-checkout" "$OFFLINE_FIXTURES/leak-reference-solution/scratch-project" "$OFFLINE_FIXTURES/package" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "offline fixtures (leak-reference-solution): guard exited 0, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "offline fixtures (leak-reference-solution): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -q "source=evaluator_only.reference_solution" "$err" || fail "offline fixtures (leak-reference-solution): failure message does not name the reference_solution source: $(cat "$err")"

err="$WORKDIR/offline-leak-answer-keys.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$OFFLINE_FIXTURES/leak-answer-keys/fixture-checkout" "$OFFLINE_FIXTURES/leak-answer-keys/scratch-project" "$OFFLINE_FIXTURES/package" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "offline fixtures (leak-answer-keys): guard exited 0, want non-zero"
grep -q "root=scratch_shark_project" "$err" || fail "offline fixtures (leak-answer-keys): failure message does not name the scratch-project root: $(cat "$err")"
grep -q "source=evaluator_only.answer_keys\[0\]" "$err" || fail "offline fixtures (leak-answer-keys): failure message does not name the answer_keys source: $(cat "$err")"

echo "TC-043(committed offline fixtures: clean/leak-in-fixture-checkout/leak-in-scratch-project/leak-reference-solution/leak-answer-keys all verify correctly) PASS"

# ---------------------------------------------------------------------------
# test_identity filename-prose regression: a base-fixture-like file that
# merely MENTIONS an oracle file's own name in prose (e.g. a comment or
# docstring reading "see test_probe.py for details") must NOT be flagged --
# only a real whole-identifier reference (e.g. "import test_probe") is a
# genuine test_identity signal. Without excluding "." from the word-boundary
# pattern's disqualifying character class, "test_probe" followed immediately
# by ".py" would still satisfy a bare [A-Za-z0-9_]-only lookahead (since "."
# is not in that class) and falsely fire on ordinary prose -- this is a
# concrete regression case for that fix, independent of whether the current
# real corpus happens to trigger it.
# ---------------------------------------------------------------------------
echo "TC-043: test_identity whole-token regression (filename mention in prose vs. real reference)"

PROSE_ROOT="$WORKDIR/test-identity-prose"
mkdir -p "$PROSE_ROOT"
echo "# placeholder: see test_probe.py for the held-back oracle (not present here)" >"$PROSE_ROOT/notes.md"
out="$WORKDIR/prose.out"
err="$WORKDIR/prose.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$PROSE_ROOT" "$OFFLINE_FIXTURES/clean/scratch-project" "$OFFLINE_FIXTURES/package" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "test_identity regression (prose mention): guard exited $code on a mere filename mention in prose, want 0 (false positive): $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "test_identity regression (prose mention): guard stdout was $(cat "$out"), want CLEAN"

REFERENCE_ROOT="$WORKDIR/test-identity-reference"
mkdir -p "$REFERENCE_ROOT"
echo "import test_probe  # a real whole-identifier reference to the held-back module" >"$REFERENCE_ROOT/consumer.py"
err="$WORKDIR/reference.err"
set +e
run_guard "$OFFLINE_FIXTURES/package/package.yaml" "$REFERENCE_ROOT" "$OFFLINE_FIXTURES/clean/scratch-project" "$OFFLINE_FIXTURES/package" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "test_identity regression (real reference): guard exited 0 on a genuine 'import test_probe' reference, want non-zero"
grep -q "match_kind=test_identity" "$err" || fail "test_identity regression (real reference): failure message's match_kind was not 'test_identity': $(cat "$err")"

echo "TC-043(test_identity whole-token regression: prose mention -> CLEAN, real reference -> violation) PASS"

# ---------------------------------------------------------------------------
# Round-4 UAT fix regression (T-E40-F06-003): reproduces the UAT report
# verbatim -- a file renamed to something unrelated, carrying NEITHER the
# original oracle_tests[] file's basename (no filename match) NOR its stem
# as a whole-token reference (no test_identity match: "test_recurring" does
# not appear anywhere in the planted content) NOR its byte content (no
# content_digest match), but still literally defining one of the real,
# held-back py-feature-recurring-tasks oracle's test functions
# (test_add_task_accepts_recurrence_rule). Before this fix the guard
# matched none of the three existing signals and exited 0 ("CLEAN") -- a
# real held-back test identity leaking into an agent-visible root,
# undetected. Only the new derived_test_identity signal (the package's own
# adapter asked to enumerate the real oracle file's defined test
# identities) can catch this.
# ---------------------------------------------------------------------------
echo "TC-043: round-4 UAT fix regression - renamed file carrying a held-back oracle's real test identity"

PLANTED_IDENTITY="$FIXTURE_CHECKOUT/renamed_oracle_identity.py"
cat >"$PLANTED_IDENTITY" <<'EOF'
def test_add_task_accepts_recurrence_rule():
    pass
EOF

out="$WORKDIR/identity.out"
err="$WORKDIR/identity.err"
set +e
run_guard "$CANDIDATE_DIR/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$CANDIDATE_DIR" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "round-4 UAT fix regression: guard exited 0 (CLEAN) with renamed_oracle_identity.py planted, want non-zero -- this is the exact round-4 UAT reproduction"
grep -q "root=agent_fixture_checkout" "$err" || fail "round-4 UAT fix regression: failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_IDENTITY" "$err" || fail "round-4 UAT fix regression: failure message does not name the planted path: $(cat "$err")"
grep -q 'source=evaluator_only.oracle_tests\[0\]' "$err" || fail "round-4 UAT fix regression: failure message does not name the matched evaluator_only source: $(cat "$err")"
grep -q "match_kind=derived_test_identity" "$err" || fail "round-4 UAT fix regression: failure message's match_kind was not 'derived_test_identity': $(cat "$err")"
grep -q "identity=test_add_task_accepts_recurrence_rule" "$err" || fail "round-4 UAT fix regression: failure message does not name the leaked identity test_add_task_accepts_recurrence_rule: $(cat "$err")"

rm -f "$PLANTED_IDENTITY"
echo "TC-043(round-4 UAT fix regression: renamed file carrying a held-back oracle's real test identity -> rejected, naming the identity) PASS"

# ---------------------------------------------------------------------------
# Round-3 code-review fix regression, finding F.1 (T-E40-F06-003): a
# malicious/escaping package.yaml adapter.name must not select and execute
# an arbitrary script. Points adapter.name at a real, executable script
# outside bench/scenarios/scenarios.yaml's registered adapters -- both an
# absolute-path escape and a '../'-laden traversal escape -- and asserts the
# guard fails closed (non-zero, never CLEAN) WITHOUT ever invoking the
# planted script (its own marker file is proof of non-execution, a stronger
# assertion than the exit code alone: even a guard that failed closed for an
# unrelated reason after already running the script would leave this marker
# behind).
# ---------------------------------------------------------------------------
echo "TC-043: round-3 code-review fix regression - malicious adapter.name path escape"

MALICIOUS_ADAPTER_DIR="$WORKDIR/evil-adapter-dir"
mkdir -p "$MALICIOUS_ADAPTER_DIR"
EVIL_MARKER="$WORKDIR/evil-adapter-invoked.marker"
rm -f "$EVIL_MARKER"
cat >"$MALICIOUS_ADAPTER_DIR/adapter.sh" <<EOF
#!/usr/bin/env bash
touch "$EVIL_MARKER"
echo '{"ids": [{"id": "x", "name": "pwned"}]}'
EOF
chmod +x "$MALICIOUS_ADAPTER_DIR/adapter.sh"

assert_adapter_name_escape_rejected() {
	# assert_adapter_name_escape_rejected <label> <malicious_adapter_name>
	local label="$1" malicious_name="$2"
	local candidate
	candidate="$(mk_candidate "evil-adapter-$label")"
	python3 - "$candidate/package.yaml" "$malicious_name" <<'PYEOF'
import sys

import yaml

path, malicious_name = sys.argv[1:3]
with open(path) as f:
    data = yaml.safe_load(f)
data["adapter"] = {"name": malicious_name, "version": "1.0.0"}
with open(path, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF

	local out="$WORKDIR/evil-adapter-$label.out" err="$WORKDIR/evil-adapter-$label.err"
	set +e
	run_guard "$candidate/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$candidate" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "adapter.name escape ($label): guard exited 0 (CLEAN) with a malicious adapter.name=$malicious_name, want non-zero"
	[[ "$(cat "$out")" != "CLEAN" ]] || fail "adapter.name escape ($label): guard stdout was CLEAN with a malicious adapter.name=$malicious_name planted"
	[[ ! -e "$EVIL_MARKER" ]] || fail "adapter.name escape ($label): the planted malicious adapter.sh at $MALICIOUS_ADAPTER_DIR was ACTUALLY EXECUTED (marker file present) -- path-escape/execution vulnerability, not merely a failed guard: $(cat "$err")"
	grep -q "adapter.name not registered" "$err" || fail "adapter.name escape ($label): failure message does not report the unregistered/rejected adapter.name (fail-closed via registry lookup, not execution): $(cat "$err")"
}

TRAVERSAL_ADAPTER_NAME="$(python3 -c "import os, sys; print(os.path.relpath(sys.argv[1], sys.argv[2]))" "$MALICIOUS_ADAPTER_DIR" "$BENCH_DIR/adapters")"
[[ "$TRAVERSAL_ADAPTER_NAME" == ../* ]] || fail "test setup bug: computed traversal adapter.name does not actually contain '../': $TRAVERSAL_ADAPTER_NAME"

assert_adapter_name_escape_rejected absolute "$MALICIOUS_ADAPTER_DIR"
assert_adapter_name_escape_rejected traversal "$TRAVERSAL_ADAPTER_NAME"

echo "TC-043(round-3 code-review fix regression: malicious adapter.name (absolute and '../' traversal) rejected without executing the planted script) PASS"

# ---------------------------------------------------------------------------
# Round-3 code-review fix regression, finding F.2 (T-E40-F06-003): a renamed
# copy of a parametrize-decorated oracle test, stripped of its original file
# name and any stem reference, must still be caught. The runtime node id
# (e.g. "test_foo[1]") never appears as literal source text in
# "def test_foo(n): ...", so only a derived BASE identity (the [param]
# suffix stripped) can catch this via a word-boundary source search.
# ---------------------------------------------------------------------------
echo "TC-043: round-3 code-review fix regression - renamed parametrized oracle test"

PARAM_CANDIDATE="$(mk_candidate parametrized)"
# Copied from a committed fixture rather than heredoc'd inline: a real
# parametrize-decorated test cannot be authored without a collection-tool
# token TC-051's AC-T3 forbidden-token sweep looks for, and this script is
# itself a sweep target.
cp "$OFFLINE_FIXTURES/parametrized-oracle/test_recurring.py" "$PARAM_CANDIDATE/evaluator/test_recurring.py"

PLANTED_PARAM_IDENTITY="$FIXTURE_CHECKOUT/renamed_param_oracle_identity.py"
cat >"$PLANTED_PARAM_IDENTITY" <<'EOF'
def test_add_task_accepts_recurrence_rule_param(n):
    assert n > 0
EOF

out="$WORKDIR/param-identity.out"
err="$WORKDIR/param-identity.err"
set +e
run_guard "$PARAM_CANDIDATE/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$PARAM_CANDIDATE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "round-3 code-review fix regression (parametrized): guard exited 0 (CLEAN) with a renamed parametrized oracle test planted, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "round-3 code-review fix regression (parametrized): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_PARAM_IDENTITY" "$err" || fail "round-3 code-review fix regression (parametrized): failure message does not name the planted path: $(cat "$err")"
grep -q "match_kind=derived_test_identity" "$err" || fail "round-3 code-review fix regression (parametrized): failure message's match_kind was not 'derived_test_identity': $(cat "$err")"
grep -q "identity=test_add_task_accepts_recurrence_rule_param" "$err" || fail "round-3 code-review fix regression (parametrized): failure message does not name the base (suffix-stripped) leaked identity: $(cat "$err")"

rm -f "$PLANTED_PARAM_IDENTITY"
echo "TC-043(round-3 code-review fix regression: renamed parametrized oracle test -> rejected, naming the base identity) PASS"

# ---------------------------------------------------------------------------
# Round-4 code-review fix regression, finding F.1 (T-E40-F06-003): the
# text-parsing collector (bc716aff) reconstructed a test's identity by
# string-splitting the collection tool's own free-form `--collect-only -q`
# output on "::", which broke on a parametrize decorator's CUSTOM `ids=[...]`
# identifier containing a space (silently dropped the whole line, so
# no identity -- not even the base name -- was ever derived) or containing
# "::" (took the wrong "::"-split segment as the name, emitting a garbage
# identity). The current collector derives identity from the collection
# tool's own structured collection data (item.originalname via a
# collection-modification plugin hook) instead of parsing any text, so
# neither shape can defeat it. Each case below first proves clean,
# untouched roots still verify CLEAN with the custom-id fixture substituted
# (no false hard-fail from the mix of an ordinary test plus a custom-id
# parametrized one -- the same property the round-4 code-review report's own
# reproduction checked), then plants a renamed, stripped file carrying only
# the real function body and asserts it is rejected, naming the identity.
# ---------------------------------------------------------------------------
echo "TC-043: round-4 code-review fix regression - custom parametrize id containing a space"

SPACE_ID_CANDIDATE="$(mk_candidate custom-id-space)"
cp "$OFFLINE_FIXTURES/custom-id-space-oracle/test_recurring.py" "$SPACE_ID_CANDIDATE/evaluator/test_recurring.py"

out="$WORKDIR/space-id-clean.out"
err="$WORKDIR/space-id-clean.err"
set +e
run_guard "$SPACE_ID_CANDIDATE/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$SPACE_ID_CANDIDATE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "round-4 code-review fix regression (space id, clean roots): guard exited $code with the custom-id-space oracle substituted and nothing planted, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "round-4 code-review fix regression (space id, clean roots): guard stdout was $(cat "$out"), want CLEAN"

PLANTED_SPACE_ID="$FIXTURE_CHECKOUT/renamed_leak_custom_space_id.py"
cat >"$PLANTED_SPACE_ID" <<'EOF'
def test_add_task_accepts_recurrence_rule_custom_space_id(n):
    assert n > 0
EOF

out="$WORKDIR/space-id-leak.out"
err="$WORKDIR/space-id-leak.err"
set +e
run_guard "$SPACE_ID_CANDIDATE/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$SPACE_ID_CANDIDATE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "round-4 code-review fix regression (space id): guard exited 0 (CLEAN) with a renamed custom-space-id oracle test planted, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "round-4 code-review fix regression (space id): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_SPACE_ID" "$err" || fail "round-4 code-review fix regression (space id): failure message does not name the planted path: $(cat "$err")"
grep -q "match_kind=derived_test_identity" "$err" || fail "round-4 code-review fix regression (space id): failure message's match_kind was not 'derived_test_identity': $(cat "$err")"
grep -q "identity=test_add_task_accepts_recurrence_rule_custom_space_id" "$err" || fail "round-4 code-review fix regression (space id): failure message does not name the leaked identity test_add_task_accepts_recurrence_rule_custom_space_id: $(cat "$err")"

rm -f "$PLANTED_SPACE_ID"
echo "TC-043(round-4 code-review fix regression: custom parametrize id containing a space -> clean roots verify CLEAN, renamed leak rejected naming the identity) PASS"

echo "TC-043: round-4 code-review fix regression - custom parametrize id containing '::'"

COLON_ID_CANDIDATE="$(mk_candidate custom-id-colon)"
cp "$OFFLINE_FIXTURES/custom-id-colon-oracle/test_recurring.py" "$COLON_ID_CANDIDATE/evaluator/test_recurring.py"

out="$WORKDIR/colon-id-clean.out"
err="$WORKDIR/colon-id-clean.err"
set +e
run_guard "$COLON_ID_CANDIDATE/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$COLON_ID_CANDIDATE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "round-4 code-review fix regression (colon id, clean roots): guard exited $code with the custom-id-colon oracle substituted and nothing planted, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "round-4 code-review fix regression (colon id, clean roots): guard stdout was $(cat "$out"), want CLEAN"

PLANTED_COLON_ID="$FIXTURE_CHECKOUT/renamed_leak_custom_colon_id.py"
cat >"$PLANTED_COLON_ID" <<'EOF'
def test_add_task_accepts_recurrence_rule_custom_colon_id(n):
    assert n > 0
EOF

out="$WORKDIR/colon-id-leak.out"
err="$WORKDIR/colon-id-leak.err"
set +e
run_guard "$COLON_ID_CANDIDATE/package.yaml" "$FIXTURE_CHECKOUT" "$SCRATCH_PROJECT" "$COLON_ID_CANDIDATE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "round-4 code-review fix regression (colon id): guard exited 0 (CLEAN) with a renamed custom-colon-id oracle test planted, want non-zero"
grep -q "root=agent_fixture_checkout" "$err" || fail "round-4 code-review fix regression (colon id): failure message does not name the fixture-checkout root: $(cat "$err")"
grep -qF "$PLANTED_COLON_ID" "$err" || fail "round-4 code-review fix regression (colon id): failure message does not name the planted path: $(cat "$err")"
grep -q "match_kind=derived_test_identity" "$err" || fail "round-4 code-review fix regression (colon id): failure message's match_kind was not 'derived_test_identity': $(cat "$err")"
grep -q "identity=test_add_task_accepts_recurrence_rule_custom_colon_id" "$err" || fail "round-4 code-review fix regression (colon id): failure message does not name the leaked identity test_add_task_accepts_recurrence_rule_custom_colon_id: $(cat "$err")"

rm -f "$PLANTED_COLON_ID"
echo "TC-043(round-4 code-review fix regression: custom parametrize id containing '::' -> clean roots verify CLEAN, renamed leak rejected naming the identity) PASS"

echo "TC-043: PASS"
