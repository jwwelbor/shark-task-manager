#!/usr/bin/env bash
# TC-056 (test-plan.md AC test matrix row AC-009; Caller-Path Contracts row
# tc056; T-E40-F07-006 task spec Acceptance Criteria AC-T1/AC-T2/AC-T3).
#
# Proves REQ-F-012's bulk-disclosure guard -- the second invocation shape
# T-E40-F07-004 reserved on verify-replay-isolation.sh (dispatched on
# argument COUNT: three positional args, not one):
#   verify-replay-isolation.sh <bundle_path> <fixture_checkout> <scratch_project>
#
# Four independently-asserted cases (test-plan.md AC-009 row):
#   (a) the replay bundle itself, planted under its own committed filename
#       inside the fixture checkout, fails bundle_bulk_disclosure naming the
#       agent_fixture_checkout root and the exact planted path.
#   (b) a COPY of the bundle under a DIFFERENT filename but the SAME content
#       digest, planted inside the scratch Shark project, independently
#       fails the same way, naming the scratch_shark_project root and the
#       digest-matched (not filename-matched) path -- proving a guard that
#       compares filenames alone cannot pass this case (test-plan.md tc056
#       row, "Negative").
#   (c) both roots left clean while a single supplied entry response sits in
#       the scratch project (the resolver's own working directory, REQ-F-015)
#       -- entry-at-a-time disclosure is the permitted path (ADR-F07-07) and
#       MUST NOT trip the guard.
#   (d) case (a) repeated with a PATH-stubbed dispatcher on PATH, proving the
#       guard's failing exit happens with ZERO dispatcher invocations -- the
#       check runs before any dispatch, not merely before a successful one.
#
# Caller-Path Contract (test-plan.md tc056 row): a REAL filesystem walk of
# two live roots and REAL content-digest computation for the renamed-copy
# case. The planted file must actually exist on disk at the path under
# test -- never a metadata flag standing in for a real planted file. The one
# permitted mock is the PATH-stubbed dispatcher binary used only to prove
# zero invocations in case (d).
#
# Fixture note: this task (T-E40-F07-006) depends only on T-E40-F07-004 and
# its own Brownfield Context says "do not implement the resolver
# (replay-answer.sh, T-E40-F07-005) ... here." Case (c) therefore does not
# shell out to the real resolver (which may not have landed yet on a
# concurrently-developed branch) -- it plants a single response-shaped file
# directly, matching the resolver's end STATE (one entry's content sitting
# in the scratch project) without depending on the resolver's own
# implementation. spec.md's Integration Contracts row for this task confirms
# this slice is meant to read only the bundle path and the two roots.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

VERIFY_ISOLATION="$SCRIPTS_DIR/verify-replay-isolation.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
ROOTS_DIR="$SCRIPTS_DIR/testdata/replay/roots"
BUNDLE="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json"

fail() {
	echo "TC-056 FAIL: $1" >&2
	exit 1
}

[[ -x "$VERIFY_ISOLATION" ]] || fail "verify-replay-isolation.sh missing or not executable: $VERIFY_ISOLATION"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -f "$BUNDLE" ]] || fail "reference bundle fixture missing: $BUNDLE"
[[ -d "$ROOTS_DIR/clean-fixture-checkout" ]] || fail "clean-fixture-checkout fixture missing"
[[ -d "$ROOTS_DIR/clean-scratch-project" ]] || fail "clean-scratch-project fixture missing"
[[ -d "$ROOTS_DIR/planted-fixture-checkout" ]] || fail "planted-fixture-checkout fixture missing"
[[ -d "$ROOTS_DIR/planted-scratch-project" ]] || fail "planted-scratch-project fixture missing"
[[ -f "$ROOTS_DIR/planted-fixture-checkout/leaked-reference-bundle.json" ]] || fail "planted-fixture-checkout has no planted bundle copy"
[[ -f "$ROOTS_DIR/planted-scratch-project/archive/backup-notes.txt" ]] || fail "planted-scratch-project has no renamed bundle copy"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# fresh_roots <case_name> <fixture_checkout_src> <scratch_project_src> --
# copies the two named fixture root trees into a fresh pair of working
# directories under WORKDIR, so no test case mutates the committed fixtures
# and no case's working state leaks into another's.
fresh_roots() {
	local case_name="$1" fc_src="$2" sp_src="$3"
	local fc_dst="$WORKDIR/$case_name/fixture-checkout"
	local sp_dst="$WORKDIR/$case_name/scratch-project"
	mkdir -p "$fc_dst" "$sp_dst"
	cp -r "$ROOTS_DIR/$fc_src/." "$fc_dst/"
	cp -r "$ROOTS_DIR/$sp_src/." "$sp_dst/"
	echo "$fc_dst" "$sp_dst"
}

# ---------------------------------------------------------------------------
# Case (a): the bundle itself, planted inside the fixture checkout, fails
# bundle_bulk_disclosure naming the agent_fixture_checkout root and the
# exact path.
# ---------------------------------------------------------------------------
echo "TC-056: case (a) - bundle planted in fixture checkout fails, naming root and path"

read -r FC_A SP_A < <(fresh_roots "case-a" "planted-fixture-checkout" "clean-scratch-project")
PLANTED_A="$FC_A/leaked-reference-bundle.json"
[[ -f "$PLANTED_A" ]] || fail "case (a): test setup bug -- planted bundle copy not present at $PLANTED_A"

out="$WORKDIR/a.out"
err="$WORKDIR/a.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_A" "$SP_A" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (a): verify-replay-isolation.sh exited $code with the bundle planted in the fixture checkout, want 1: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (a): failure message does not name bundle_bulk_disclosure: $(cat "$err")"
grep -q "root=agent_fixture_checkout" "$err" || fail "case (a): failure message does not name the agent_fixture_checkout root: $(cat "$err")"
grep -qF "path=$PLANTED_A" "$err" || fail "case (a): failure message does not name the exact planted path $PLANTED_A: $(cat "$err")"

echo "TC-056(case a: bundle planted in fixture checkout -> bundle_bulk_disclosure naming root=agent_fixture_checkout and the exact path) PASS"

# ---------------------------------------------------------------------------
# Case (b): a content-digest-identical copy under a DIFFERENT filename,
# planted inside the scratch project, independently fails -- proving a
# filename-only guard would miss it.
# ---------------------------------------------------------------------------
echo "TC-056: case (b) - digest-matched renamed copy planted in scratch project fails, naming root and path"

read -r FC_B SP_B < <(fresh_roots "case-b" "clean-fixture-checkout" "planted-scratch-project")
PLANTED_B="$SP_B/archive/backup-notes.txt"
[[ -f "$PLANTED_B" ]] || fail "case (b): test setup bug -- renamed bundle copy not present at $PLANTED_B"
[[ "$(basename "$PLANTED_B")" != "$(basename "$BUNDLE")" ]] || fail "case (b): test setup bug -- renamed copy shares the bundle's own filename, does not exercise the renamed-copy path"
cmp -s "$BUNDLE" "$PLANTED_B" || fail "case (b): test setup bug -- renamed copy is not byte-identical to the real bundle"

out="$WORKDIR/b.out"
err="$WORKDIR/b.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_B" "$SP_B" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (b): verify-replay-isolation.sh exited $code with a digest-matched renamed copy in the scratch project, want 1: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (b): failure message does not name bundle_bulk_disclosure: $(cat "$err")"
grep -q "root=scratch_shark_project" "$err" || fail "case (b): failure message does not name the scratch_shark_project root: $(cat "$err")"
grep -qF "path=$PLANTED_B" "$err" || fail "case (b): failure message does not name the exact digest-matched path $PLANTED_B (a filename-only guard would miss this renamed copy): $(cat "$err")"

echo "TC-056(case b: digest-matched renamed copy in scratch project -> bundle_bulk_disclosure naming root=scratch_shark_project and the exact path, despite the different filename) PASS"

# ---------------------------------------------------------------------------
# Case (c): both roots clean, with a single supplied-entry response present
# in the scratch project (the working directory, REQ-F-015) -- entry-at-a-
# time disclosure is permitted and MUST NOT trip the guard.
# ---------------------------------------------------------------------------
echo "TC-056: case (c) - clean roots with one supplied entry in place exit 0"

read -r FC_C SP_C < <(fresh_roots "case-c" "clean-fixture-checkout" "clean-scratch-project")
SUPPLIED_ENTRY="$SP_C/docs/product/D01-problem-statement.md"
[[ -f "$SUPPLIED_ENTRY" ]] || fail "case (c): test setup bug -- single supplied-entry fixture not present at $SUPPLIED_ENTRY"
cmp -s "$BUNDLE" "$SUPPLIED_ENTRY" && fail "case (c): test setup bug -- supplied-entry fixture is byte-identical to the whole bundle, does not exercise the single-entry exemption"

out="$WORKDIR/c.out"
err="$WORKDIR/c.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_C" "$SP_C" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (c): verify-replay-isolation.sh exited $code on clean roots with one supplied entry in place, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "case (c): stdout was $(cat "$out"), want CLEAN"

echo "TC-056(case c: clean roots + one supplied entry in place -> exit 0, CLEAN -- entry-at-a-time disclosure permitted) PASS"

# ---------------------------------------------------------------------------
# Case (d): case (a) repeated with a PATH-stubbed dispatcher, proving the
# guard's failing exit happens with ZERO dispatcher invocations -- the check
# runs before any dispatch.
# ---------------------------------------------------------------------------
echo "TC-056: case (d) - case (a) repeated under a PATH-stubbed dispatcher records zero invocations"

read -r FC_D SP_D < <(fresh_roots "case-d" "planted-fixture-checkout" "clean-scratch-project")
LOG_D="$WORKDIR/d-claude-invocations.jsonl"
rm -f "$LOG_D"
err="$WORKDIR/d.err"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_D" "$VERIFY_ISOLATION" "$BUNDLE" "$FC_D" "$SP_D" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (d): verify-replay-isolation.sh exited $code with the bundle planted in the fixture checkout (PATH-stubbed dispatcher present), want 1: $(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (d): failure message does not name bundle_bulk_disclosure: $(cat "$err")"
[[ ! -s "$LOG_D" ]] || fail "case (d): stub claude recorded $(wc -l <"$LOG_D") invocation(s) on the failing bulk-disclosure path, want zero (the check must run before any dispatch): $(cat "$LOG_D")"

echo "TC-056(case d: bulk-disclosure failure occurs with zero PATH-stubbed dispatcher invocations, proving the check runs before dispatch) PASS"

echo "TC-056: PASS"
