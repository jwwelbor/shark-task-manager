#!/usr/bin/env bash
# TC-045 (test-plan.md AC test matrix row AC-005; Caller-Path Contracts row
# tc045; T-E40-F06-005 task spec Test Cases, AC-T1 through AC-T3).
#
# Drives bench/scripts/verify-stage-evidence.sh (the candidate-identity
# portion T-E40-F06-005 implements, same entrypoint as tc044) against real
# bundle fixtures under bench/scripts/testdata/evidence/candidate/ --
# REQ-F-006, REQ-F-007, ADR-009, ADR-F06-05:
#
#   AC-T1: a `code` snapshot missing any one of `tree_digest`,
#     `binary_diff_digest`, `changed_path_digest`, `dirty_untracked_manifest`,
#     or `test_suite_digest` -> rejected naming that exact field. Five
#     independent fixtures, one per missing field.
#   AC-T2: two snapshots sharing one `base_commit` value but differing in
#     `tree_digest` -> reported as DISTINCT candidates (different
#     `candidate_identity_digest`), never merged/deduplicated -- proving
#     `base_commit` alone never establishes identity (ADR-009).
#   AC-T3: `test_suite_digest` is validated only for presence, like every
#     other digest field -- the guard never recomputes it, never inspects
#     test file contents, and contains no fixture-language branching
#     token. The repo-wide mechanical sweep across EVERY generic evidence
#     script is AC-019/T-E40-F06-011's job; this case only proves the
#     property for THIS script, at THIS task's point in the build order.
#
# Caller-Path Contract (test-plan.md tc045): the guard reads the actual
# fixture JSON fields on disk; this test never stubs `base_commit`
# comparison logic or hand-computes the expected `candidate_identity_digest`
# separately from the guard's own output -- AC-T2 compares the guard's two
# live outputs against EACH OTHER, and a fixture self-check (mirroring
# tc044's AC-T4 style) confirms the two source fixtures on disk genuinely
# share one base_commit and differ only in tree_digest, so a fixture typo
# cannot make the comparison pass vacuously.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
FIXTURES="$SCRIPTS_DIR/testdata/evidence/candidate"

fail() {
	echo "TC-045 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $GUARD"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# AC-T1: five independent missing-field cases.
# ---------------------------------------------------------------------------
declare -A MISSING_FIELD_FIXTURES=(
	["missing-tree-digest"]="tree_digest"
	["missing-binary-diff-digest"]="binary_diff_digest"
	["missing-changed-path-digest"]="changed_path_digest"
	["missing-dirty-untracked-manifest"]="dirty_untracked_manifest"
	["missing-test-suite-digest"]="test_suite_digest"
)

for fixture in "${!MISSING_FIELD_FIXTURES[@]}"; do
	field="${MISSING_FIELD_FIXTURES[$fixture]}"
	echo "TC-045: AC-T1 - $fixture (missing candidate.$field) -> rejected naming that field"

	err="$WORKDIR/$fixture.err"
	set +e
	"$GUARD" "$FIXTURES/$fixture" >/dev/null 2>"$err"
	code=$?
	set -e
	[[ "$code" -eq 1 ]] || fail "$fixture: guard exited $code on a candidate block missing '$field', want 1: $(cat "$err")"
	grep -q "candidate_field_missing" "$err" || fail "$fixture: stderr does not name candidate_field_missing: $(cat "$err")"
	grep -q "field=$field" "$err" || fail "$fixture: stderr does not name the missing field ($field): $(cat "$err")"

	echo "TC-045(AC-T1: $fixture missing candidate.$field -> rejected naming field=$field) PASS"
done

# ---------------------------------------------------------------------------
# AC-T2: two snapshots sharing base_commit but differing tree_digest are
# reported as distinct candidates, never merged.
# ---------------------------------------------------------------------------
echo "TC-045: AC-T2 - shared base_commit, differing tree_digest -> distinct candidates"

# Fixture self-check first: confirm the two source snapshots on disk
# genuinely share one base_commit and differ ONLY in tree_digest, so the
# guard-output comparison below cannot pass vacuously on a fixture typo
# (mirrors tc044's AC-T4 fixture self-check style).
python3 -c "
import json
with open('$FIXTURES/distinct-a/stages/1-develop.json') as f:
    snap_a = json.load(f)
with open('$FIXTURES/distinct-b/stages/1-develop.json') as f:
    snap_b = json.load(f)
cand_a = snap_a['candidate']
cand_b = snap_b['candidate']
assert cand_a['base_commit'] == cand_b['base_commit'], (
    f'fixture bug: base_commit differs ({cand_a[\"base_commit\"]!r} vs {cand_b[\"base_commit\"]!r}), '
    'AC-T2 requires the two fixtures to share one base_commit value'
)
assert cand_a['tree_digest'] != cand_b['tree_digest'], (
    'fixture bug: tree_digest is identical between distinct-a and distinct-b, '
    'AC-T2 requires them to differ'
)
for field in ('binary_diff_digest', 'changed_path_digest', 'dirty_untracked_manifest', 'test_suite_digest'):
    assert cand_a[field] == cand_b[field], (
        f'fixture bug: {field} differs between distinct-a and distinct-b -- AC-T2 isolates '
        'tree_digest as the ONLY differing field so the regression it catches is unambiguous'
    )
" || fail "AC-T2: fixture self-check failed (see message above)"

out_a="$WORKDIR/distinct-a.out"
out_b="$WORKDIR/distinct-b.out"
err_a="$WORKDIR/distinct-a.err"
err_b="$WORKDIR/distinct-b.err"
set +e
"$GUARD" "$FIXTURES/distinct-a" >"$out_a" 2>"$err_a"
code_a=$?
"$GUARD" "$FIXTURES/distinct-b" >"$out_b" 2>"$err_b"
code_b=$?
set -e
[[ "$code_a" -eq 0 ]] || fail "AC-T2: guard exited $code_a on distinct-a (complete candidate), want 0: $(cat "$err_a")"
[[ "$code_b" -eq 0 ]] || fail "AC-T2: guard exited $code_b on distinct-b (complete candidate), want 0: $(cat "$err_b")"

# The comparison itself: read straight from the guard's OWN two live
# outputs, never a hand-computed expected digest -- the guard performs the
# hashing; this test only checks the guard agrees with itself never that a
# parallel implementation agrees with the guard (Caller-Path Contract).
python3 -c "
import json
with open('$out_a') as f:
    result_a = json.load(f)
with open('$out_b') as f:
    result_b = json.load(f)
cand_a = result_a['stages'][0]['candidate']
cand_b = result_b['stages'][0]['candidate']
assert cand_a['base_commit'] == cand_b['base_commit'], (
    f'guard output bug: reported base_commit differs ({cand_a[\"base_commit\"]!r} vs {cand_b[\"base_commit\"]!r}) '
    'even though both fixtures declare the same base_commit'
)
assert cand_a['candidate_identity_digest'] != cand_b['candidate_identity_digest'], (
    'AC-T2 VIOLATION: two snapshots sharing base_commit but differing only in tree_digest '
    'produced the SAME candidate_identity_digest -- base_commit (or some field subset '
    'excluding tree_digest) is being used as identity, exactly the regression ADR-009 forbids'
)
" || fail "AC-T2: distinct-candidate identity assertion failed (see message above)"

echo "TC-045(AC-T2: distinct-a/distinct-b share base_commit, differ only in tree_digest, report distinct candidate_identity_digest values -> never merged) PASS"

# ---------------------------------------------------------------------------
# AC-T3: test_suite_digest is validated as an opaque presence-only field --
# the guard contains no fixture-language branching token. The repo-wide
# sweep across every generic evidence script is AC-019/task 011's job; this
# is the single-script proof for verify-stage-evidence.sh, using the exact
# word-boundary pattern tc031 established for AC-012 (word-boundary so a
# script's own legitimate `python3` interpreter invocation for JSON/YAML
# processing does not trip the check).
# ---------------------------------------------------------------------------
echo "TC-045: AC-T3 - verify-stage-evidence.sh contains no fixture-language branching token"

FORBIDDEN_TOKENS='\<python\>|\<pytest\>|\<pip\>|\<go[[:space:]]+test\>|\<golangci-lint\>|\<go[[:space:]]+build\>'
if hits="$(grep -nE "$FORBIDDEN_TOKENS" "$GUARD")"; then
	fail "AC-T3: forbidden fixture-language token found in $GUARD (REQ-F-007 opacity discipline, ADR-F06-05):
$hits"
fi

echo "TC-045(AC-T3: verify-stage-evidence.sh contains no forbidden fixture-language token; test_suite_digest validated opaque -- repo-wide sweep is AC-019/task 011) PASS"

echo "TC-045: PASS"
