#!/usr/bin/env bash
# TC-049 (test-plan.md AC test matrix rows AC-012, AC-013; Caller-Path
# Contracts row tc049; T-E40-F06-010 task spec Test Cases, AC-T1
# through AC-T4).
#
# Drives bench/scripts/replay-stage-evidence.sh (this task's own new
# entrypoint -- "the replay script IS the entrypoint", test-plan.md tc049
# Caller-Path Contract) against a real stored-bundle fixture and its named
# checkout root under bench/scripts/testdata/evidence/replay/ -- REQ-F-013,
# REQ-F-015:
#
#   AC-T1: an edited tracked file -> tracked_file_changed naming the path;
#     an added untracked file -> untracked_file_changed naming the path.
#   AC-T2: a changed adapter normalized test-id set -> test_suite_changed
#     naming the differing id; a deleted artifact `consumers` key ->
#     artifact_consumption_record_missing naming the artifact.
#   AC-T3: a PATH-stubbed provider (bench/scripts/testdata/stubs/claude,
#     the same X-07 provider-invocation stand-in TC-043 reuses) records
#     ZERO invocations across every one of the four replays above, plus the
#     two AC-013 replays below -- proving "no worker rerun and no provider
#     call" empirically (the replay script never execs the stub at all;
#     this is the observation that a hypothetical implementation which
#     silently reruns the worker "to be sure" would fail).
#   AC-T4: recomputing `snapshot_digest` over an UNMODIFIED stored snapshot
#     reproduces the recorded value (case (i)); a one-byte edit to
#     `rework_count` yields `snapshot_mutated` naming the stage (case (ii)).
#
# Caller-Path Contract (test-plan.md tc049): real stored-bundle JSON, real
# digest recomputation, real filesystem diff of the checkout root against
# the recorded `candidate.dirty_untracked_manifest`, and a real
# `bench/adapters/python/adapter.sh test` subprocess invocation for the
# test-suite-drift case -- never stubbed (I-04's real `test` capability,
# already covered read-only by TC-030 per this task's Brownfield Context).
#
# Every case works on a FRESH COPY of the committed fixtures under a
# per-case WORKDIR subdirectory (mirroring tc048's new_bundle/new_checkout
# style) -- never against the committed fixture in place. Running
# adapter.sh test writes `.pytest_cache`/`__pycache__` side effects into
# whatever checkout it is pointed at; running it against the committed
# fixture directory would dirty the repo on every test run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/replay-stage-evidence.sh"
ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
FIXTURE="$SCRIPTS_DIR/testdata/evidence/replay"
BUNDLE_SRC="$FIXTURE/bundle"
CHECKOUT_SRC="$FIXTURE/checkout"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"

fail() {
	echo "TC-049 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "replay-stage-evidence.sh missing or not executable: $GUARD"
[[ -x "$ADAPTER" ]] || fail "python adapter missing or not executable: $ADAPTER"
[[ -f "$BUNDLE_SRC/bundle.json" ]] || fail "fixture bundle.json missing: $BUNDLE_SRC/bundle.json"
[[ -f "$CHECKOUT_SRC/tracked_file.txt" ]] || fail "fixture checkout missing tracked_file.txt: $CHECKOUT_SRC"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

new_bundle() {
	# new_bundle <name> -- fresh copy of the committed stored bundle.
	local name="$1"
	local dest="$WORKDIR/$name-bundle"
	cp -r "$BUNDLE_SRC" "$dest"
	echo "$dest"
}

new_checkout() {
	# new_checkout <name> -- fresh copy of the committed checkout root.
	local name="$1"
	local dest="$WORKDIR/$name-checkout"
	cp -r "$CHECKOUT_SRC" "$dest"
	echo "$dest"
}

# run_replay <bundle_dir> <checkout_dir> <out_file> <err_file> <log_file>
# Runs the guard with a PATH-stubbed `claude` ahead of the real PATH and a
# fresh (nonexistent-until-written) invocation log, so a genuinely
# unexercised stub is distinguishable from a stub that was invoked and
# happened to log nothing.
run_replay() {
	local bundle_dir="$1" checkout_dir="$2" out="$3" err="$4" log="$5"
	rm -f "$log"
	set +e
	PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$log" STUB_CLAUDE_EXIT=1 \
		"$GUARD" "$bundle_dir" --checkout "$checkout_dir" --adapter "$ADAPTER" \
		>"$out" 2>"$err"
	local code=$?
	set -e
	echo "$code"
}

assert_zero_invocations() {
	# assert_zero_invocations <label> <log_file>
	local label="$1" log="$2"
	[[ ! -s "$log" ]] || fail "$label: stubbed provider recorded $(wc -l <"$log") invocation(s), want zero (REQ-F-013 'no worker rerun and no provider call'): $(cat "$log")"
}

# ---------------------------------------------------------------------------
# Baseline: unmutated bundle + unmutated checkout -> clean replay. The
# positive control every AC-T1/AC-T2 mutation case below is a one-field
# delta from.
# ---------------------------------------------------------------------------
echo "TC-049: baseline - unmutated bundle and checkout replay clean"

bundle_base="$(new_bundle baseline)"
checkout_base="$(new_checkout baseline)"
out="$WORKDIR/baseline.out"
err="$WORKDIR/baseline.err"
log="$WORKDIR/baseline.log"
code="$(run_replay "$bundle_base" "$checkout_base" "$out" "$err" "$log")"
[[ "$code" -eq 0 ]] || fail "baseline: guard exited $code on an unmutated bundle/checkout, want 0: $(cat "$err")"
grep -q '"result": "clean"' "$out" || fail "baseline: stdout does not report a clean result: $(cat "$out")"
assert_zero_invocations baseline "$log"

echo "TC-049(baseline: unmutated bundle/checkout -> clean, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T1 (a): an edited tracked file.
# ---------------------------------------------------------------------------
echo "TC-049: AC-T1(a) - tracked file edited -> tracked_file_changed"

bundle_a="$(new_bundle case-a)"
checkout_a="$(new_checkout case-a)"
echo "tc049 mutation: tracked file edited after the snapshot was taken" >>"$checkout_a/tracked_file.txt"
out="$WORKDIR/case-a.out"
err="$WORKDIR/case-a.err"
log="$WORKDIR/case-a.log"
code="$(run_replay "$bundle_a" "$checkout_a" "$out" "$err" "$log")"
[[ "$code" -eq 1 ]] || fail "case (a): guard exited $code, want 1 (drift detected): $(cat "$err")"
grep -q "tracked_file_changed" "$err" || fail "case (a): stderr does not name tracked_file_changed: $(cat "$err")"
grep -q "path=tracked_file.txt" "$err" || fail "case (a): stderr does not name the edited path: $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (a): guard printed stdout on a drift verdict: $(cat "$out")"
assert_zero_invocations "case (a)" "$log"

echo "TC-049(AC-T1(a): edited tracked file -> tracked_file_changed naming the path, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T1 (b): an added untracked file.
# ---------------------------------------------------------------------------
echo "TC-049: AC-T1(b) - untracked file added -> untracked_file_changed"

bundle_b="$(new_bundle case-b)"
checkout_b="$(new_checkout case-b)"
echo "tc049 mutation: a brand new untracked file, absent from dirty_untracked_manifest" >"$checkout_b/leaked_note.txt"
out="$WORKDIR/case-b.out"
err="$WORKDIR/case-b.err"
log="$WORKDIR/case-b.log"
code="$(run_replay "$bundle_b" "$checkout_b" "$out" "$err" "$log")"
[[ "$code" -eq 1 ]] || fail "case (b): guard exited $code, want 1 (drift detected): $(cat "$err")"
grep -q "untracked_file_changed" "$err" || fail "case (b): stderr does not name untracked_file_changed: $(cat "$err")"
grep -q "path=leaked_note.txt" "$err" || fail "case (b): stderr does not name the added path: $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (b): guard printed stdout on a drift verdict: $(cat "$out")"
assert_zero_invocations "case (b)" "$log"

echo "TC-049(AC-T1(b): added untracked file -> untracked_file_changed naming the path, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T2 (c): the adapter's normalized test-id set changed (one id added).
# ---------------------------------------------------------------------------
echo "TC-049: AC-T2(c) - adapter test-id set changed -> test_suite_changed"

bundle_c="$(new_bundle case-c)"
checkout_c="$(new_checkout case-c)"
cat >>"$checkout_c/tests/test_sample.py" <<'PYEOF'


def test_gamma():
    assert 3 + 3 == 6
PYEOF
out="$WORKDIR/case-c.out"
err="$WORKDIR/case-c.err"
log="$WORKDIR/case-c.log"
code="$(run_replay "$bundle_c" "$checkout_c" "$out" "$err" "$log")"
[[ "$code" -eq 1 ]] || fail "case (c): guard exited $code, want 1 (drift detected): $(cat "$err")"
grep -q "test_suite_changed" "$err" || fail "case (c): stderr does not name test_suite_changed: $(cat "$err")"
grep -q "test_id=tests.test_sample::test_gamma" "$err" || fail "case (c): stderr does not name the differing test id: $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (c): guard printed stdout on a drift verdict: $(cat "$out")"
assert_zero_invocations "case (c)" "$log"

echo "TC-049(AC-T2(c): changed adapter test-id set -> test_suite_changed naming the differing id, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T2 (d): one artifact's `consumers` key deleted from the stored
# snapshot.
# ---------------------------------------------------------------------------
echo "TC-049: AC-T2(d) - artifact consumers key deleted -> artifact_consumption_record_missing"

bundle_d="$(new_bundle case-d)"
checkout_d="$(new_checkout case-d)"
python3 -c "
import json
path = '$bundle_d/stages/1-develop.json'
with open(path) as f:
    snapshot = json.load(f)
assert 'consumers' in snapshot['artifacts'][0], 'fixture bug: artifacts[0] has no consumers key to delete'
del snapshot['artifacts'][0]['consumers']
with open(path, 'w') as f:
    json.dump(snapshot, f, indent=2)
"
out="$WORKDIR/case-d.out"
err="$WORKDIR/case-d.err"
log="$WORKDIR/case-d.log"
code="$(run_replay "$bundle_d" "$checkout_d" "$out" "$err" "$log")"
[[ "$code" -eq 1 ]] || fail "case (d): guard exited $code, want 1 (drift detected): $(cat "$err")"
grep -q "artifact_consumption_record_missing" "$err" || fail "case (d): stderr does not name artifact_consumption_record_missing: $(cat "$err")"
grep -q "artifact=artifacts/develop.diff" "$err" || fail "case (d): stderr does not name the offending artifact: $(cat "$err")"
# Deleting the consumers key IS a snapshot content edit, so the generic
# snapshot_mutated verdict (AC-013: "a one-byte edit to ANY snapshot field
# yields snapshot_mutated") must ALSO be present, never replaced by the
# more specific artifact_consumption_record_missing verdict above.
grep -q "snapshot_mutated" "$err" || fail "case (d): stderr does not also name snapshot_mutated (a consumers-key deletion is a snapshot edit like any other): $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (d): guard printed stdout on a drift verdict: $(cat "$out")"
assert_zero_invocations "case (d)" "$log"

echo "TC-049(AC-T2(d): deleted consumers key -> artifact_consumption_record_missing AND snapshot_mutated, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T4 / AC-013 case (i): recomputing snapshot_digest over an UNMODIFIED
# stored snapshot reproduces the recorded value. This is the exact
# regression the task's Notes for Agent name: an implementation that folds
# `snapshot_digest` itself into the recomputation would spuriously fail
# this case even though nothing was mutated.
# ---------------------------------------------------------------------------
echo "TC-049: AC-013(i) - unmodified snapshot reproduces its recorded snapshot_digest"

bundle_i="$(new_bundle case-i)"
checkout_i="$(new_checkout case-i)"
out="$WORKDIR/case-i.out"
err="$WORKDIR/case-i.err"
log="$WORKDIR/case-i.log"
code="$(run_replay "$bundle_i" "$checkout_i" "$out" "$err" "$log")"
[[ "$code" -eq 0 ]] || fail "case (i): guard exited $code on an unmodified snapshot, want 0 (digest must reproduce): $(cat "$err")"
grep -q '"result": "clean"' "$out" || fail "case (i): stdout does not report a clean result: $(cat "$out")"
assert_zero_invocations "case (i)" "$log"

echo "TC-049(AC-013(i): unmodified snapshot's recomputed digest reproduces the recorded value, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# AC-T4 / AC-013 case (ii): a one-byte edit to `rework_count` yields
# snapshot_mutated naming the stage.
# ---------------------------------------------------------------------------
echo "TC-049: AC-013(ii) - one-byte edit to rework_count -> snapshot_mutated"

bundle_ii="$(new_bundle case-ii)"
checkout_ii="$(new_checkout case-ii)"
python3 -c "
import json
path = '$bundle_ii/stages/1-develop.json'
with open(path) as f:
    snapshot = json.load(f)
before = snapshot['rework_count']
snapshot['rework_count'] = before + 1
assert snapshot['snapshot_digest'], 'fixture bug: snapshot has no recorded snapshot_digest to leave stale'
with open(path, 'w') as f:
    json.dump(snapshot, f, indent=2)
"
out="$WORKDIR/case-ii.out"
err="$WORKDIR/case-ii.err"
log="$WORKDIR/case-ii.log"
code="$(run_replay "$bundle_ii" "$checkout_ii" "$out" "$err" "$log")"
[[ "$code" -eq 1 ]] || fail "case (ii): guard exited $code, want 1 (snapshot_mutated): $(cat "$err")"
grep -q "snapshot_mutated" "$err" || fail "case (ii): stderr does not name snapshot_mutated: $(cat "$err")"
grep -q "stage=develop" "$err" || fail "case (ii): stderr does not name the stage: $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (ii): guard printed stdout on a drift verdict: $(cat "$out")"
assert_zero_invocations "case (ii)" "$log"

echo "TC-049(AC-013(ii): one-byte rework_count edit -> snapshot_mutated naming the stage, zero provider invocations) PASS"

# ---------------------------------------------------------------------------
# REQ-F-007 opacity discipline (ADR-F06-05): replay-stage-evidence.sh itself
# contains no fixture-language branching token, the same single-script proof
# tc031/tc045 established for their own guards at their point in the build
# order. Scoped to $GUARD only -- never this test script, which legitimately
# names bench/adapters/python/adapter.sh to drive a REAL adapter, exactly as
# tc048 already does. The repo-wide sweep across every generic evidence
# script is AC-019/T-E40-F06-011's job.
# ---------------------------------------------------------------------------
echo "TC-049: replay-stage-evidence.sh contains no fixture-language branching token"

FORBIDDEN_TOKENS='\<python\>|\<pytest\>|\<pip\>|\<go[[:space:]]+test\>|\<golangci-lint\>|\<go[[:space:]]+build\>'
if hits="$(grep -nE "$FORBIDDEN_TOKENS" "$GUARD")"; then
	fail "forbidden fixture-language token found in $GUARD (REQ-F-007 opacity discipline, ADR-F06-05):
$hits"
fi

echo "TC-049(replay-stage-evidence.sh contains no forbidden fixture-language token) PASS"

echo "TC-049: PASS"
