#!/usr/bin/env bash
# TC-016 (test-plan.md AC test matrix; T-E40-F02-005 task spec Test Cases).
#
# Exercises AC-13 (REQ-F-020): the X-07 canary preflight
# (`bench/scripts/canary-runsurface.sh`) asserts the RunResult/StageLog JSON
# field set and the transcript byte format are unchanged, aborting and
# naming exactly the changed field on drift.
#
# Caller-Path Contract (test-plan.md TC-016a/b): real subprocess invocation
# of `bench/scripts/canary-runsurface.sh [--corpus <corpus.yaml>]`.
# TC-016a exercises the real `RunController`/`LivenessRecorder`/
# transcript-writer code path via a real `shark run` invocation in the
# canary's own throwaway scratch project (provisioned by the canary itself),
# with a PATH-stubbed `claude` binary only -- `shark run` itself is never
# stubbed. TC-016b points the canary (via CANARY_RUNSURFACE_RUNRESULT_FIXTURE,
# mirroring TC-011's DIFF_LEDGERS_GOLANGCI_CONFIG override technique) at a
# hand-authored fixture RunResult JSON with `stages_completed` renamed to
# `stagesCompleted`, so no scratch project or real run is needed for that
# sub-case.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

CANARY="$SCRIPTS_DIR/canary-runsurface.sh"
REAL_SHARK="$REPO_ROOT/bin/shark"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-016 FAIL: $1" >&2
	exit 1
}

[[ -x "$CANARY" ]] || fail "canary-runsurface.sh missing or not executable: $CANARY"
[[ -x "$REAL_SHARK" ]] || fail "real shark binary missing at $REAL_SHARK (run 'make shark' first)"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# TC-016a (AC-13, pass path): a real shark run invocation (throwaway scratch
# project provisioned by the canary itself, PATH-stubbed `claude` only)
# exits 0 and prints the confirmed field set. Also covers AC-13's own
# "reordered keys still pass" negative case, via the same fixture-override
# mechanism TC-016b uses (field presence/names are checked, not JSON key
# encoding order).
# ---------------------------------------------------------------------------
test_a() {
	local out="$WORKDIR/a.out" err="$WORKDIR/a.err"

	set +e
	"$CANARY" --corpus "$CORPUS_YAML" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "a: canary-runsurface.sh exited $code, want 0: $(cat "$err")"

	# Output contract (Observability Design table): the canary's own
	# messages -- including the final PASS sentinel -- go to stderr,
	# matching testdata/stubs/shark's established "failure text lands on
	# stderr" convention that run-one.sh's TC-014g already depends on.
	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == "PASS" ]] || fail "a: last stderr line was $(printf '%q' "$last_line"), want exactly PASS: $(cat "$err")"

	# AC-13: "printing the confirmed field set" -- derived from the live
	# invocation's actual JSON keys, not hardcoded from memory. Confirm the
	# known-required RunResult/StageLog fields actually appear somewhere in
	# the diagnostic output (proves it inspected real data, not a canned
	# string).
	grep -q "stages_completed" "$err" || fail "a: confirmed field set does not mention stages_completed: $(cat "$err")"
	grep -q "output_summary" "$err" || fail "a: confirmed field set does not mention output_summary: $(cat "$err")"

	# Provisioning marker must have appeared (a real scratch project was
	# used) and the scratch dir named in it must be gone after exit (AC-13:
	# "tears the scratch project down on exit regardless of outcome").
	local scratch_line scratch_dir
	scratch_line="$(grep -m1 '^canary-runsurface: scratch=' "$err" || true)"
	[[ -n "$scratch_line" ]] || fail "a: no scratch provisioning marker on stderr: $(cat "$err")"
	scratch_dir="${scratch_line#canary-runsurface: scratch=}"
	[[ ! -e "$scratch_dir" ]] || fail "a: scratch dir $scratch_dir still exists after canary-runsurface.sh exited"

	echo "TC-016a PASS"

	# --- Negative (AC-13): every field present but JSON keys reordered
	# must still pass -- the canary checks field presence and names, not
	# encoding order. ---
	local reordered_fixture="$WORKDIR/a-reordered-fixture.json"
	cat >"$reordered_fixture" <<'JSON'
{
  "outcome": "completed",
  "total_duration_ns": 123,
  "stages": [],
  "stages_completed": 0,
  "final_status": "completed",
  "entity_key": "T-E01-F01-001"
}
JSON
	local rout="$WORKDIR/a-reordered.out" rerr="$WORKDIR/a-reordered.err"
	set +e
	CANARY_RUNSURFACE_RUNRESULT_FIXTURE="$reordered_fixture" "$CANARY" >"$rout" 2>"$rerr"
	local rcode=$?
	set -e
	[[ "$rcode" -eq 0 ]] || fail "a(reordered): canary-runsurface.sh exited $rcode on a reordered-but-complete fixture, want 0: $(cat "$rerr")"
	[[ "$(tail -n1 "$rerr")" == "PASS" ]] || fail "a(reordered): last stderr line was not PASS: $(cat "$rerr")"
	# The fixture path never provisions a scratch project.
	! grep -q '^canary-runsurface: scratch=' "$rerr" || fail "a(reordered): scratch project was provisioned despite the fixture override"

	echo "TC-016a(reordered) PASS"
}

# ---------------------------------------------------------------------------
# TC-016b (AC-13, fail path): canary-runsurface.sh pointed at a fixture
# RunResult JSON with `stages_completed` renamed to `stagesCompleted` exits
# non-zero, naming exactly the changed field -- never a generic "shape
# mismatch" message. No scratch project or real run is needed for this path.
# ---------------------------------------------------------------------------
test_b() {
	local fixture="$WORKDIR/b-fixture.json"
	cat >"$fixture" <<'JSON'
{
  "entity_key": "T-E01-F01-001",
  "final_status": "completed",
  "stagesCompleted": 2,
  "stages": [
    {"status": "draft", "action": "advance_status", "duration_ns": 1, "exit_code": 0},
    {"status": "development", "action": "spawn_agent", "agent_type": "developer", "provider": "anthropic", "duration_ns": 1, "exit_code": 0, "output_summary": "{}"}
  ],
  "outcome": "completed",
  "total_duration_ns": 123
}
JSON

	local out="$WORKDIR/b.out" err="$WORKDIR/b.err"
	if CANARY_RUNSURFACE_RUNRESULT_FIXTURE="$fixture" "$CANARY" >"$out" 2>"$err"; then
		fail "b: canary-runsurface.sh exited 0 against a mutated fixture (stages_completed -> stagesCompleted)"
	fi

	grep -Eq 'stages_completed|stagesCompleted' "$err" || fail "b: neither stages_completed nor stagesCompleted named on stderr: $(cat "$err")"
	! grep -qi "shape mismatch" "$err" || fail "b: generic 'shape mismatch' message used instead of naming the field: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == FAIL:* ]] || fail "b: last stderr line was $(printf '%q' "$last_line"), want a 'FAIL: <field>' line"

	# No scratch project or real shark run for the fixture-override path.
	! grep -q '^canary-runsurface: scratch=' "$err" || fail "b: scratch project was provisioned despite the fixture override"

	echo "TC-016b PASS"
}

test_a
test_b

echo "TC-016 PASS"
