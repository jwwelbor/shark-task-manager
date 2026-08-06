#!/usr/bin/env bash
# TC-015 (test-plan.md AC test matrix; T-E40-F02-001 task spec Test Cases).
#
# This task's slice: sub-cases a (minus `usage`), b, d, e, f, j, m. The
# remaining sub-cases (c, g, h, i, l) are added by T-E40-F02-002/007, which
# extend this same file per their own task specs' Scope.
#
# Caller-Path Contract (test-plan.md TC-015): real subprocess invocation of
# `bench/scripts/collect-run.sh --run-dir <dir>` against committed synthetic
# run directories under testdata/run/ -- nothing in the collector is stubbed;
# only a real sqlite database (materialized at test time from a committed
# .sql fixture) backs the TC-015b DB-status-fallback sub-case.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

COLLECT_SCRIPT="$SCRIPTS_DIR/collect-run.sh"
FIXTURES_DIR="$SCRIPTS_DIR/testdata/run"

fail() {
	echo "TC-015 FAIL: $1" >&2
	exit 1
}

[[ -x "$COLLECT_SCRIPT" ]] || fail "collect-run.sh missing or not executable"
[[ -d "$FIXTURES_DIR" ]] || fail "testdata/run fixtures directory missing: $FIXTURES_DIR"
command -v sqlite3 >/dev/null 2>&1 || fail "sqlite3 not found on PATH (needed to materialize TC-015b's DB fixture)"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# run_collect <run-dir> <out-file> -- invokes collect-run.sh, failing loudly
# on a non-zero exit (every fixture in this file represents a well-formed,
# fully-captured run directory; a non-zero exit always means a bug).
run_collect() {
	local run_dir="$1" out_file="$2"
	"$COLLECT_SCRIPT" --run-dir "$run_dir" >"$out_file" ||
		fail "collect-run.sh exited non-zero for $run_dir: $(cat "$out_file")"
}

# ---------------------------------------------------------------------------
# TC-015a (AC-08, minus `usage` -- T-E40-F02-007's scope): outcome
# equivalence partitioning across the five RunResult values (timeout is
# TC-015b, below). Every clean-* fixture has zero errors[] and a `runresult`
# block copying RunResult verbatim.
# ---------------------------------------------------------------------------
test_a() {
	local out="$WORKDIR/a.out"
	local fixture outcome
	for fixture in clean-completed clean-paused-question clean-already-terminal clean-no-action clean-failed; do
		run_collect "$FIXTURES_DIR/$fixture" "$out"
		python3 - "$out" "$fixture" <<'PYEOF'
import json
import sys

path, fixture = sys.argv[1], sys.argv[2]
with open(path) as f:
    record = json.load(f)

expected = {
    "clean-completed": "completed",
    "clean-paused-question": "paused",
    "clean-already-terminal": "already_terminal",
    "clean-no-action": "no_action",
    "clean-failed": "failed",
}[fixture]

if record.get("outcome") != expected:
    sys.exit("TC-015a FAIL: %s: outcome=%r, want %r" % (fixture, record.get("outcome"), expected))
if record.get("errors") != []:
    sys.exit("TC-015a FAIL: %s: errors=%r, want []" % (fixture, record.get("errors")))
if "runresult" not in record:
    sys.exit("TC-015a FAIL: %s: record has no runresult block" % fixture)
if "timeout_detail" in record:
    sys.exit("TC-015a FAIL: %s: a non-timeout record must never carry timeout_detail" % fixture)

if fixture == "clean-paused-question":
    qb = record["runresult"].get("question_block")
    want = {
        "question_key": "Q100",
        "summary": "Confirm rounding mode before implementing the discount calculator.",
        "resolution_owner": "product",
        "current_responder": "product",
    }
    if qb != want:
        sys.exit("TC-015a FAIL: clean-paused-question: question_block=%r, want verbatim %r" % (qb, want))
    # AC-08 negative: a paused outcome must never also read as failed/timeout
    # anywhere in the record -- checked structurally (no such field exists to
    # contaminate) rather than by inventing a rollup field this task's schema
    # slice does not declare.
    if record["outcome"] in ("failed", "timeout"):
        sys.exit("TC-015a FAIL: clean-paused-question: outcome misclassified as %r" % record["outcome"])

if fixture == "clean-failed":
    if not record["runresult"].get("error"):
        sys.exit("TC-015a FAIL: clean-failed: runresult.error is empty, want a populated message")

print("TC-015a: %s -> outcome=%s PASS" % (fixture, expected))
PYEOF
	done
}

# ---------------------------------------------------------------------------
# TC-015b (AC-03, REQ-F-019): timeout path, no RunResult ever delivered.
# Sub-case 1: stalled stage recovered from the liveness stream.
# Sub-case 2: no liveness stream at all -> scratch-DB status fallback.
# ---------------------------------------------------------------------------
test_b() {
	local out="$WORKDIR/b1.out"
	run_collect "$FIXTURES_DIR/timeout-with-liveness" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record.get("outcome") != "timeout":
    sys.exit("TC-015b FAIL: liveness sub-case: outcome=%r, want 'timeout'" % record.get("outcome"))
if "runresult" in record:
    sys.exit("TC-015b FAIL: liveness sub-case: runresult must be genuinely absent on the timeout path, not null-filled")
detail = record.get("timeout_detail")
want = {
    "stage_index": 1,
    "status": "in_development",
    "action": "spawn_agent",
    "agent_type": "developer",
    "provider": "anthropic",
    "source": "liveness_stream",
}
if detail != want:
    sys.exit("TC-015b FAIL: liveness sub-case: timeout_detail=%r, want %r" % (detail, want))
if record.get("sources", {}).get("stalled_stage") != "liveness":
    sys.exit("TC-015b FAIL: liveness sub-case: sources.stalled_stage=%r, want 'liveness'" % record.get("sources"))
print("TC-015b: liveness sub-case PASS")
PYEOF

	# Sub-case 2 needs a real sqlite file materialized from the committed
	# .sql fixture -- copy the fixture (its committed form has no .db file)
	# into a scratch workdir first (test-plan.md TC-015: "real sqlite read...
	# do not stub SQLite reads").
	local db_case_dir="$WORKDIR/timeout-no-liveness-db-fallback"
	cp -r "$FIXTURES_DIR/timeout-no-liveness-db-fallback" "$db_case_dir"
	sqlite3 "$db_case_dir/scratch/shark-tasks.db" <"$db_case_dir/scratch/scratch_db.sql"

	local out2="$WORKDIR/b2.out"
	run_collect "$db_case_dir" "$out2"
	python3 - "$out2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record.get("outcome") != "timeout":
    sys.exit("TC-015b FAIL: DB-fallback sub-case: outcome=%r, want 'timeout'" % record.get("outcome"))
detail = record.get("timeout_detail")
want = {
    "stage_index": None,
    "status": "in_development",
    "action": None,
    "agent_type": None,
    "provider": None,
    "source": "scratch_db_status_fallback",
}
if detail != want:
    sys.exit("TC-015b FAIL: DB-fallback sub-case: timeout_detail=%r, want %r" % (detail, want))
if record.get("sources", {}).get("stalled_stage") != "scratch_db":
    sys.exit("TC-015b FAIL: DB-fallback sub-case: sources.stalled_stage=%r, want 'scratch_db'" % record.get("sources"))
# The run_id fallback (AC-20's mechanism) also fires here, since there is no
# liveness stream at all to resolve run_id from either.
if record["manifest"].get("run_id_source") != "fallback_newest_dir":
    sys.exit("TC-015b FAIL: DB-fallback sub-case: manifest.run_id_source=%r, want 'fallback_newest_dir'"
              % record["manifest"].get("run_id_source"))
print("TC-015b: scratch-DB status fallback sub-case PASS")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015d (AC-06): the only stage is advance_status -- zero envelope_parse_error
# and zero transcript_missing entries. Negative: a stray transcript file at
# the advance_status stage's own array index must be ignored, not
# misattributed (built into a temp copy so the committed positive fixture
# stays clean).
# ---------------------------------------------------------------------------
test_d() {
	local out="$WORKDIR/d.out"
	run_collect "$FIXTURES_DIR/advance-status-only" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

kinds = {e["kind"] for e in record.get("errors", [])}
if "envelope_parse_error" in kinds or "transcript_missing" in kinds:
    sys.exit("TC-015d FAIL: advance_status-only stage produced %r, want neither" % kinds)
if record.get("errors") != []:
    sys.exit("TC-015d FAIL: errors=%r, want [] (no stage_join_error either, since zero spawn_agent stages are expected)" % record.get("errors"))
print("TC-015d: advance_status-only positive PASS (zero envelope/transcript errors)")
PYEOF

	# Negative: stray transcript "1-<status>-<provider>.log" at the
	# advance_status stage's own index (1) must be ignored, not misattributed.
	local stray_dir="$WORKDIR/advance-status-only-stray"
	cp -r "$FIXTURES_DIR/advance-status-only" "$stray_dir"
	printf 'COMMAND: some-unrelated-command\nEXIT: 0\nDURATION: 10ms\n---STDOUT---\n{}\n---STDERR---\n' \
		>"$stray_dir/run/transcripts/1-ready_for_qa-anthropic.log"

	local out_stray="$WORKDIR/d_stray.out"
	run_collect "$stray_dir" "$out_stray"
	python3 - "$out_stray" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record.get("errors") != []:
    sys.exit("TC-015d FAIL: stray-transcript negative: errors=%r, want [] (stray file must be ignored)" % record.get("errors"))
stage = record["stages"][0]
if "transcript_path" in stage:
    sys.exit("TC-015d FAIL: stray-transcript negative: the advance_status stage must never gain a transcript_path")
print("TC-015d: stray-transcript negative PASS (ignored, not misattributed)")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015e (AC-07): spawn_agent/transcript reconciliation. Sub-case 1
# (committed fixture): aggregate count mismatch, one transcript missing.
# Sub-case 2 (dynamic): count equal, but a filename component (provider)
# mismatches the corresponding stage's recorded provider.
# ---------------------------------------------------------------------------
test_e() {
	local out="$WORKDIR/e1.out"
	run_collect "$FIXTURES_DIR/missing-transcript-count-mismatch" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

errors = record.get("errors", [])
if len(errors) != 1 or errors[0]["kind"] != "stage_join_error":
    sys.exit("TC-015e FAIL: count-mismatch sub-case: errors=%r, want exactly one stage_join_error" % errors)
detail = errors[0]["detail"]
if "expected 2" not in detail or "found 1" not in detail:
    sys.exit("TC-015e FAIL: count-mismatch sub-case: detail=%r does not name expected=2/observed=1" % detail)
print("TC-015e: count-mismatch sub-case PASS")
PYEOF

	# Component-mismatch sub-case: reuse the count-mismatch fixture (which is
	# missing stage 3's transcript entirely) and add stage 3's transcript
	# back, but with the wrong provider component -- count becomes equal (2
	# expected, 2 present-at-matching-indices), isolating the component check.
	local mismatch_dir="$WORKDIR/component-mismatch"
	cp -r "$FIXTURES_DIR/missing-transcript-count-mismatch" "$mismatch_dir"
	printf 'COMMAND: claude --output-format json\nEXIT: 0\nDURATION: 1960ms\n---STDOUT---\n{}\n---STDERR---\n' \
		>"$mismatch_dir/run/transcripts/3-in_qa-codex.log"

	local out2="$WORKDIR/e2.out"
	run_collect "$mismatch_dir" "$out2"
	python3 - "$out2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

errors = record.get("errors", [])
join_errors = [e for e in errors if e["kind"] == "stage_join_error"]
if len(join_errors) != 1:
    sys.exit("TC-015e FAIL: component-mismatch sub-case: stage_join_error entries=%r, want exactly 1" % join_errors)
entry = join_errors[0]
if entry.get("stage_index") != 3:
    sys.exit("TC-015e FAIL: component-mismatch sub-case: stage_index=%r, want 3" % entry.get("stage_index"))
detail = entry["detail"]
if "anthropic" not in detail or "codex" not in detail:
    sys.exit("TC-015e FAIL: component-mismatch sub-case: detail=%r does not name both providers" % detail)
print("TC-015e: component-mismatch sub-case PASS")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015f (AC-20): no liveness stream -> run_id resolved by newest-directory
# fallback. Plus: two-candidate mtime race (fixture mtimes are not
# git-preserved, so this sub-case is built fresh in a temp dir), and the
# negative that a present stream must never read the fallback value even
# when a stale extra .shark/runs/ directory also exists.
# ---------------------------------------------------------------------------
test_f() {
	local out="$WORKDIR/f1.out"
	run_collect "$FIXTURES_DIR/no-liveness-stream" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record["manifest"].get("run_id") != "71f0a000-0000-4000-8000-000000000007":
    sys.exit("TC-015f FAIL: run_id=%r, want the single candidate directory's name" % record["manifest"].get("run_id"))
if record["manifest"].get("run_id_source") != "fallback_newest_dir":
    sys.exit("TC-015f FAIL: run_id_source=%r, want 'fallback_newest_dir'" % record["manifest"].get("run_id_source"))
print("TC-015f: single-candidate fallback PASS")
PYEOF

	# Two candidates, deterministic mtime race (git does not preserve mtimes,
	# so this is built fresh rather than committed).
	local race_dir="$WORKDIR/race"
	cp -r "$FIXTURES_DIR/no-liveness-stream" "$race_dir"
	local runs_dir="$race_dir/scratch/.shark/runs"
	rm -rf "${runs_dir:?}"/*
	mkdir -p "$runs_dir/older-candidate" "$runs_dir/newer-candidate"
	: >"$runs_dir/older-candidate/run.log"
	: >"$runs_dir/newer-candidate/run.log"
	touch -d '2026-01-01T00:00:00' "$runs_dir/older-candidate"
	touch -d '2026-01-02T00:00:00' "$runs_dir/newer-candidate"

	local out_race="$WORKDIR/f_race.out"
	run_collect "$race_dir" "$out_race"
	python3 - "$out_race" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record["manifest"].get("run_id") != "newer-candidate":
    sys.exit("TC-015f FAIL: mtime race: run_id=%r, want the newer of the two candidates" % record["manifest"].get("run_id"))
if record["manifest"].get("run_id_source") != "fallback_newest_dir":
    sys.exit("TC-015f FAIL: mtime race: run_id_source=%r, want 'fallback_newest_dir' (still a fallback, not a false stream claim)"
              % record["manifest"].get("run_id_source"))
print("TC-015f: two-candidate mtime race PASS (newest wins)")
PYEOF

	# Negative: a genuinely present stream must never read the fallback value,
	# even when a stale extra run directory also exists.
	local stream_present_dir="$WORKDIR/stream-present-with-stale-dir"
	cp -r "$FIXTURES_DIR/clean-completed" "$stream_present_dir"
	python3 - "$stream_present_dir/meta.json" <<'PYEOF'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    meta = json.load(f)
meta["scratch_root"] = "scratch"
with open(path, "w") as f:
    json.dump(meta, f)
PYEOF
	mkdir -p "$stream_present_dir/scratch/.shark/runs/stale-unrelated-dir"
	: >"$stream_present_dir/scratch/.shark/runs/stale-unrelated-dir/run.log"

	local out_negative="$WORKDIR/f_negative.out"
	run_collect "$stream_present_dir" "$out_negative"
	python3 - "$out_negative" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if record["manifest"].get("run_id_source") != "liveness_stream":
    sys.exit("TC-015f FAIL: negative: run_id_source=%r, want 'liveness_stream' (a present stream must win over a stale fallback candidate)"
              % record["manifest"].get("run_id_source"))
if record["manifest"].get("run_id") == "stale-unrelated-dir":
    sys.exit("TC-015f FAIL: negative: run_id resolved to the stale fallback candidate instead of the real stream value")
print("TC-015f: negative PASS (stream takes priority over a stale fallback candidate)")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015j (AC-14, REQ-N-004): two independent collect-run.sh invocations over
# the same run directory produce byte-identical stdout.
# ---------------------------------------------------------------------------
test_j() {
	local out1="$WORKDIR/j1.out"
	local out2="$WORKDIR/j2.out"
	run_collect "$FIXTURES_DIR/clean-completed" "$out1"
	run_collect "$FIXTURES_DIR/clean-completed" "$out2"
	cmp -s "$out1" "$out2" ||
		fail "TC-015j: two independent invocations over the same run directory produced different output"
	# The record must be exactly one line (one JSON object, per REQ-F-018).
	local lines
	lines="$(wc -l <"$out1" | tr -d ' ')"
	[[ "$lines" == "1" ]] || fail "TC-015j: record is $lines lines, want exactly 1"
	echo "TC-015j: PASS (byte-identical across two independent invocations, exactly one line)"
}

# ---------------------------------------------------------------------------
# TC-015m (errors[].kind schema completeness, this task's Ambiguity Finding
# resolution): a genuinely zero-byte transcript at an otherwise count-/
# component-correct position is transcript_missing, never stage_join_error.
# ---------------------------------------------------------------------------
test_m() {
	local out="$WORKDIR/m.out"
	run_collect "$FIXTURES_DIR/missing-transcript-zero-byte" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

errors = record.get("errors", [])
if len(errors) != 1 or errors[0]["kind"] != "transcript_missing":
    sys.exit("TC-015m FAIL: errors=%r, want exactly one transcript_missing entry" % errors)
entry = errors[0]
if entry.get("stage_index") != 3:
    sys.exit("TC-015m FAIL: stage_index=%r, want 3" % entry.get("stage_index"))
if entry.get("path") != "run/transcripts/3-in_qa-anthropic.log":
    sys.exit("TC-015m FAIL: path=%r, want the zero-byte file's relative path" % entry.get("path"))
if any(e["kind"] == "stage_join_error" for e in errors):
    sys.exit("TC-015m FAIL: a zero-byte-but-otherwise-correct transcript must never also raise stage_join_error")
# The first (non-zero-byte) stage must still resolve its transcript_path
# normally -- proving the zero-byte failure is scoped to stage 3 alone.
stage1 = record["stages"][0]
if "transcript_path" not in stage1:
    sys.exit("TC-015m FAIL: stage 1's genuinely non-empty transcript lost its transcript_path")
print("TC-015m: PASS (transcript_missing, not stage_join_error)")
PYEOF
}

test_a
test_b
test_d
test_e
test_f
test_j
test_m

echo "TC-015: PASS"
