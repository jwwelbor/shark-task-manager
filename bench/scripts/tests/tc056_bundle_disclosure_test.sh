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
cleanup() {
	chmod -R u+rwX "$WORKDIR" 2>/dev/null || true
	rm -rf "$WORKDIR"
}
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

# Permission failures must be exercised as an unprivileged reader when the
# suite itself runs as root (where chmod 000 would otherwise remain readable).
run_as_scan_user() {
	if [[ "$(id -u)" -eq 0 ]] && command -v runuser >/dev/null 2>&1 && id nobody >/dev/null 2>&1; then
		chmod 755 "$WORKDIR"
		runuser -u nobody -- "$@"
	else
		"$@"
	fi
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

# ---------------------------------------------------------------------------
# Case (e)/(f): Code Review Kickback Round 2 (2026-08-16), Finding A -- a
# bundle copy reached ONLY through a SYMLINKED DIRECTORY inside a guarded
# root must still fail bundle_bulk_disclosure. This mirrors the exact
# session-realistic exploit the reviewer reproduced: run-prelude.sh exports
# REPLAY_BUNDLE_PATH (the real bundle's own path) into the dispatched
# session's environment and pins its cwd inside a guarded root, so
# `ln -s "$(dirname "$REPLAY_BUNDLE_PATH")" ./notes` alone defeated a guard
# that walked with no followlinks -- no bulk copy of bytes required. Content
# digest (not filename) still does the matching here, exactly like case (b),
# because the symlinked-directory target legitimately shares the bundle's
# own filename in the real exploit shape.
# ---------------------------------------------------------------------------
echo "TC-056: case (e) - bundle reached via a symlinked directory in fixture checkout fails"

read -r FC_E SP_E < <(fresh_roots "case-e" "clean-fixture-checkout" "clean-scratch-project")
EXTERNAL_E="$WORKDIR/case-e/external-leak"
mkdir -p "$EXTERNAL_E"
cp "$BUNDLE" "$EXTERNAL_E/$(basename "$BUNDLE")"
ln -s "$EXTERNAL_E" "$FC_E/notes"
[[ -L "$FC_E/notes" ]] || fail "case (e): test setup bug -- $FC_E/notes is not a symlink"
PLANTED_E="$FC_E/notes/$(basename "$BUNDLE")"
[[ -f "$PLANTED_E" ]] || fail "case (e): test setup bug -- planted bundle copy not reachable at $PLANTED_E via the symlinked directory"

out="$WORKDIR/e.out"
err="$WORKDIR/e.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_E" "$SP_E" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (e): verify-replay-isolation.sh exited $code with the bundle reachable via a symlinked directory in the fixture checkout, want 1 (Code Review Kickback Round 2, Finding A): stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (e): failure message does not name bundle_bulk_disclosure: $(cat "$err")"
grep -q "root=agent_fixture_checkout" "$err" || fail "case (e): failure message does not name the agent_fixture_checkout root: $(cat "$err")"
grep -qF "path=$PLANTED_E" "$err" || fail "case (e): failure message does not name the exact planted path (reached through the symlinked directory) $PLANTED_E: $(cat "$err")"

echo "TC-056(case e: bundle reached via a symlinked directory in fixture checkout -> bundle_bulk_disclosure naming root=agent_fixture_checkout and the exact path, Code Review Kickback Round 2 Finding A) PASS"

echo "TC-056: case (f) - bundle reached via a symlinked directory in scratch project fails"

read -r FC_F SP_F < <(fresh_roots "case-f" "clean-fixture-checkout" "clean-scratch-project")
EXTERNAL_F="$WORKDIR/case-f/external-leak"
mkdir -p "$EXTERNAL_F"
cp "$BUNDLE" "$EXTERNAL_F/$(basename "$BUNDLE")"
ln -s "$EXTERNAL_F" "$SP_F/notes"
[[ -L "$SP_F/notes" ]] || fail "case (f): test setup bug -- $SP_F/notes is not a symlink"
PLANTED_F="$SP_F/notes/$(basename "$BUNDLE")"
[[ -f "$PLANTED_F" ]] || fail "case (f): test setup bug -- planted bundle copy not reachable at $PLANTED_F via the symlinked directory"

out="$WORKDIR/f.out"
err="$WORKDIR/f.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_F" "$SP_F" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (f): verify-replay-isolation.sh exited $code with the bundle reachable via a symlinked directory in the scratch project, want 1 (Code Review Kickback Round 2, Finding A): stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (f): failure message does not name bundle_bulk_disclosure: $(cat "$err")"
grep -q "root=scratch_shark_project" "$err" || fail "case (f): failure message does not name the scratch_shark_project root: $(cat "$err")"
grep -qF "path=$PLANTED_F" "$err" || fail "case (f): failure message does not name the exact planted path (reached through the symlinked directory) $PLANTED_F: $(cat "$err")"

echo "TC-056(case f: bundle reached via a symlinked directory in scratch project -> bundle_bulk_disclosure naming root=scratch_shark_project and the exact path, Code Review Kickback Round 2 Finding A) PASS"

# ---------------------------------------------------------------------------
# Case (g): a self-referential symlinked directory (one pointing back at one
# of its own ancestors) must not hang the guard. The kickback's own CAUTION:
# a bare os.walk(root, followlinks=True) has no cycle protection and can
# hang on exactly this shape -- the fix needs a visited-directory guard.
# ---------------------------------------------------------------------------
echo "TC-056: case (g) - a self-referential symlinked directory does not hang the guard"

read -r FC_G SP_G < <(fresh_roots "case-g" "clean-fixture-checkout" "clean-scratch-project")
ln -s "$FC_G" "$FC_G/self-loop"
[[ -L "$FC_G/self-loop" ]] || fail "case (g): test setup bug -- $FC_G/self-loop is not a symlink"

out="$WORKDIR/g.out"
err="$WORKDIR/g.err"
set +e
timeout 15 "$VERIFY_ISOLATION" "$BUNDLE" "$FC_G" "$SP_G" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 124 ]] || fail "case (g): verify-replay-isolation.sh timed out (hung) walking a self-referential symlinked directory -- the followlinks fix needs a visited-directory cycle guard"
[[ "$code" -eq 0 ]] || fail "case (g): verify-replay-isolation.sh exited $code (want 0/CLEAN) on clean roots with only a self-referential symlink cycle present, no bundle copy planted: stdout=$(cat "$out") stderr=$(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "case (g): stdout was $(cat "$out"), want CLEAN"

echo "TC-056(case g: self-referential symlinked directory -> guard terminates and reports CLEAN, no infinite loop) PASS"

# ---------------------------------------------------------------------------
# Case (h): a bundle copy hidden below .git must still be found. The guard
# scans the complete reachable tree; .git is not a trust boundary.
# ---------------------------------------------------------------------------
echo "TC-056: case (h) - bundle hidden below .git fails closed as disclosure"

read -r FC_H SP_H < <(fresh_roots "case-h" "clean-fixture-checkout" "clean-scratch-project")
mkdir -p "$FC_H/.git/objects/pack"
PLANTED_H="$FC_H/.git/objects/pack/hidden-copy"
cp "$BUNDLE" "$PLANTED_H"

out="$WORKDIR/h.out"
err="$WORKDIR/h.err"
set +e
"$VERIFY_ISOLATION" "$BUNDLE" "$FC_H" "$SP_H" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (h): verifier exited $code with the bundle hidden below .git, want 1: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (h): .git-hidden disclosure was not reported: $(cat "$err")"
grep -qF "path=$PLANTED_H" "$err" || fail "case (h): failure did not name the .git-hidden copy $PLANTED_H: $(cat "$err")"

echo "TC-056(case h: .git-hidden bundle copy -> bundle_bulk_disclosure naming exact path) PASS"

# ---------------------------------------------------------------------------
# Case (i): a deep tree must be scanned iteratively and still find the same
# planted copy. This protects the round-2 RecursionError follow-up.
# ---------------------------------------------------------------------------
echo "TC-056: case (i) - deep tree remains scannable without recursion failure"

read -r FC_I SP_I < <(fresh_roots "case-i" "clean-fixture-checkout" "clean-scratch-project")
DEEP_I="$FC_I"
for _ in $(seq 1 1100); do
	DEEP_I="$DEEP_I/d"
	mkdir "$DEEP_I"
done
PLANTED_I="$DEEP_I/deep-copy"
cp "$BUNDLE" "$PLANTED_I"

out="$WORKDIR/i.out"
err="$WORKDIR/i.err"
set +e
timeout 15 "$VERIFY_ISOLATION" "$BUNDLE" "$FC_I" "$SP_I" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 124 ]] || fail "case (i): verifier timed out walking the deep tree"
[[ "$code" -eq 1 ]] || fail "case (i): verifier exited $code on a deep planted copy, want 1: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "bundle_bulk_disclosure" "$err" || fail "case (i): deep-tree disclosure was not reported: $(cat "$err")"
grep -qF "path=$PLANTED_I" "$err" || fail "case (i): failure did not name the deep planted copy: $(cat "$err")"

echo "TC-056(case i: deep planted copy -> bundle_bulk_disclosure without RecursionError) PASS"

# ---------------------------------------------------------------------------
# Case (j)/(k): unreadable directories and files are scan errors, not CLEAN.
# Run these as nobody when necessary so the permission denial is real even
# when the test runner is root.
# ---------------------------------------------------------------------------
echo "TC-056: case (j) - unreadable directory fails closed instead of CLEAN"

read -r FC_J SP_J < <(fresh_roots "case-j" "clean-fixture-checkout" "clean-scratch-project")
mkdir -p "$FC_J/blocked"
cp "$BUNDLE" "$FC_J/blocked/hidden-copy"
chmod 000 "$FC_J/blocked"

out="$WORKDIR/j.out"
err="$WORKDIR/j.err"
set +e
run_as_scan_user "$VERIFY_ISOLATION" "$BUNDLE" "$FC_J" "$SP_J" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 2 ]] || fail "case (j): verifier exited $code for an unreadable directory, want 2: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "cannot list scan directory" "$err" || fail "case (j): unreadable directory did not produce a scan error: $(cat "$err")"
[[ "$(cat "$out")" != "CLEAN" ]] || fail "case (j): unreadable directory was silently reported CLEAN"

echo "TC-056(case j: unreadable directory -> scan error, never CLEAN) PASS"

echo "TC-056: case (k) - unreadable file fails closed instead of CLEAN"

read -r FC_K SP_K < <(fresh_roots "case-k" "clean-fixture-checkout" "clean-scratch-project")
PLANTED_K="$FC_K/unreadable-copy"
cp "$BUNDLE" "$PLANTED_K"
chmod 000 "$PLANTED_K"

out="$WORKDIR/k.out"
err="$WORKDIR/k.err"
set +e
run_as_scan_user "$VERIFY_ISOLATION" "$BUNDLE" "$FC_K" "$SP_K" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 2 ]] || fail "case (k): verifier exited $code for an unreadable file, want 2: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -q "cannot read scan file" "$err" || fail "case (k): unreadable file did not produce a scan error: $(cat "$err")"
[[ "$(cat "$out")" != "CLEAN" ]] || fail "case (k): unreadable file was silently reported CLEAN"

echo "TC-056(case k: unreadable file -> scan error, never CLEAN) PASS"

echo "TC-056: case (l) - inaccessible symlink target fails closed instead of CLEAN"

read -r FC_L SP_L < <(fresh_roots "case-l" "clean-fixture-checkout" "clean-scratch-project")
TARGET_L="$WORKDIR/case-l/blocked-target"
mkdir -p "$TARGET_L"
ln -s "$TARGET_L" "$FC_L/inaccessible-target"
chmod 000 "$TARGET_L"

out="$WORKDIR/l.out"
err="$WORKDIR/l.err"
set +e
run_as_scan_user "$VERIFY_ISOLATION" "$BUNDLE" "$FC_L" "$SP_L" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 2 ]] || fail "case (l): verifier exited $code for an inaccessible symlink target, want 2: stdout=$(cat "$out") stderr=$(cat "$err")"
grep -Eq "cannot (inspect scan path|list scan directory)" "$err" || fail "case (l): inaccessible symlink target did not produce a scan error: $(cat "$err")"
[[ "$(cat "$out")" != "CLEAN" ]] || fail "case (l): inaccessible symlink target was silently reported CLEAN"

echo "TC-056(case l: inaccessible symlink target -> scan error, never CLEAN) PASS"

echo "TC-056: PASS"
