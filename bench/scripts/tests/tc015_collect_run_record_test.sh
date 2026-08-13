#!/usr/bin/env bash
# TC-015 (test-plan.md AC test matrix; T-E40-F02-001 task spec Test Cases).
#
# T-E40-F02-001's slice: sub-cases a (minus `usage`), b, d, e, f, j, m.
# T-E40-F02-002 extends with g, h, i, l -- the rejections crosscheck and the
# oracle/quality/loc post-run blocks. T-E40-F02-007 (this extension) adds
# sub-case c (envelope parse error) and completes a's `usage`/
# `model_id_source` assertions, once Q003/REQ-F-021's real-transcript
# capture (T-E40-F02-006) confirmed the field names to test against.
#
# Caller-Path Contract (test-plan.md TC-015): real subprocess invocation of
# `bench/scripts/collect-run.sh --run-dir <dir>` against committed synthetic
# run directories under testdata/run/ -- nothing in the collector is stubbed;
# real sqlite databases (materialized at test time from committed .sql
# fixtures) back the TC-015b DB-status-fallback sub-case and TC-015g's
# rejection crosscheck; TC-015h's post/test-diff.json and post/lint-diff.json
# fixtures were produced by a real `bench/scripts/diff-ledgers.sh` invocation
# over hand-authored base/post ledgers (see test_h's comment below for the
# exact ledgers and command), not hand-typed to match the collector's own
# arithmetic.
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

if fixture == "clean-completed":
    # T-E40-F02-007: stages[].usage extraction, scoped to spawn_agent stages,
    # using the field names T-E40-F02-006 confirmed from a real capture
    # (num_turns, duration_api_ms, modelUsage -- camelCase, keyed by model
    # ID -- plus the pre-existing flat usage.* sub-object and total_cost_usd).
    want_usage_1 = {
        "num_turns": 3,
        "duration_api_ms": 1200,
        "total_cost_usd": 0.0142,
        "input_tokens": 80,
        "output_tokens": 95,
        "cache_read_input_tokens": 400,
        "cache_creation_input_tokens": 20,
        "model_ids": ["claude-sonnet-5"],
    }
    stage1 = record["stages"][0]
    if stage1.get("usage") != want_usage_1:
        sys.exit("TC-015a FAIL: clean-completed: stages[0].usage=%r, want %r" % (stage1.get("usage"), want_usage_1))

    want_usage_3 = {
        "num_turns": 5,
        "duration_api_ms": 1800,
        "total_cost_usd": 0.0231,
        "input_tokens": 150,
        "output_tokens": 210,
        "cache_read_input_tokens": 900,
        "cache_creation_input_tokens": 40,
        "model_ids": ["claude-sonnet-5"],
    }
    stage3 = record["stages"][2]
    if stage3.get("usage") != want_usage_3:
        sys.exit("TC-015a FAIL: clean-completed: stages[2].usage=%r, want %r" % (stage3.get("usage"), want_usage_3))

    # The advance_status stage (index 2) is not a spawn_agent stage and must
    # never gain a usage sub-object (AC-06's scoping, applied to usage too).
    if "usage" in record["stages"][1]:
        sys.exit("TC-015a FAIL: clean-completed: the advance_status stage must never gain a usage sub-object")

    if record["manifest"].get("model_ids") != ["claude-sonnet-5"]:
        sys.exit("TC-015a FAIL: clean-completed: manifest.model_ids=%r, want ['claude-sonnet-5']" % record["manifest"].get("model_ids"))
    if record["manifest"].get("model_id_source") != "modelUsage":
        sys.exit("TC-015a FAIL: clean-completed: manifest.model_id_source=%r, want 'modelUsage'" % record["manifest"].get("model_id_source"))

    # Code-review round 1 kickback (Finding B-1, BLOCKER): the six
    # G7-reproducibility manifest fields architecture.md#metric-collection-
    # and-artifact-schema/E40-interaction-map.md/uat-plan.md UAT-07 all name
    # (fixture and bundle SHAs, exact shark binary identity) must round-trip
    # from meta.json (T-E40-F02-003's rework) onto the record verbatim, via
    # the same optional-field copy pattern as item_id/variant_id/etc.
    want_g7 = {
        "fixture_base_sha": "4c24986844b09122e2d516f9bc1ec470b155b441",
        "corpus_schema_version": "1.0",
        "p2p_set": "default",
        "variant_bundle_sha256": "04fca7add5d7569e81c1beb1677f224c3fe6f149f8e231eb2b19219b25ef26bd",
        "shark_version": "shark version dev (7f188a20) built 2026-08-07",
        "shark_binary_sha256": "929d3cf370db03acac7e97c214cd85afb79d951363343f69a078afc413b9890d",
    }
    for key, want_value in want_g7.items():
        got = record["manifest"].get(key)
        if got != want_value:
            sys.exit("TC-015a FAIL: clean-completed: manifest.%s=%r, want %r (G7 reproducibility field silently dropped)"
                      % (key, got, want_value))

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
# TC-015c (AC-05, REQ-F-012, REQ-F-021): envelope parse error, one axis at a
# time. Each missing-envelope-field/<axis>/ fixture (T-E40-F02-006's
# per-axis fixtures) omits exactly one of modelUsage/num_turns/
# duration_api_ms from an otherwise complete, confirmed-shape envelope.
# Negative: the fully-populated clean-completed fixture (test_a/test_h)
# already proves zero envelope_parse_error entries on an all-fields-present
# transcript -- reasserted here directly against this TC's own axis list.
# ---------------------------------------------------------------------------
test_c() {
	local axis
	for axis in modelUsage num_turns duration_api_ms; do
		local out="$WORKDIR/c-$axis.out"
		run_collect "$FIXTURES_DIR/missing-envelope-field/$axis" "$out"
		python3 - "$out" "$axis" <<'PYEOF'
import json
import sys

path, axis = sys.argv[1], sys.argv[2]
with open(path) as f:
    record = json.load(f)

errors = record.get("errors", [])
matches = [e for e in errors if e["kind"] == "envelope_parse_error" and axis in e["detail"]]
if len(matches) != 1:
    sys.exit("TC-015c FAIL: %s: errors=%r, want exactly one envelope_parse_error naming %r" % (axis, errors, axis))
entry = matches[0]
if entry.get("stage_index") != 1:
    sys.exit("TC-015c FAIL: %s: stage_index=%r, want 1" % (axis, entry.get("stage_index")))
if not (entry.get("path") or "").endswith("1-in_development-anthropic.log"):
    sys.exit("TC-015c FAIL: %s: path=%r does not name the transcript" % (axis, entry.get("path")))

stage = record["stages"][0]
usage = stage.get("usage", {})
missing_key = {"modelUsage": "model_ids", "num_turns": "num_turns", "duration_api_ms": "duration_api_ms"}[axis]
if missing_key in usage:
    sys.exit("TC-015c FAIL: %s: usage.%s present, want absent (never a zero/empty masquerading as a measurement)"
              % (axis, missing_key))

# Every OTHER expected field must still be present -- proving the check is
# per-field, not "drop the whole usage block on any single miss" (AC-05's
# "the corresponding metric is absent, not zero" -- only that one metric).
all_fields = {
    "model_ids", "num_turns", "duration_api_ms", "total_cost_usd",
    "input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens",
}
present_others = sorted(all_fields - {missing_key})
missing_others = [k for k in present_others if k not in usage]
if missing_others:
    sys.exit("TC-015c FAIL: %s: usage missing unrelated fields %r (only %r should be absent): usage=%r"
              % (axis, missing_others, missing_key, usage))

if axis == "modelUsage":
    # The sharpest form of "never a silent reach for a nonexistent field"
    # (REQ-F-021's amendment, T-E40-F02-006's decision note): when the only
    # spawn_agent stage's envelope never resolves a model ID, manifest.
    # model_ids/model_id_source must be genuinely ABSENT -- never "" and
    # never a fabricated "model" fallback value, since no observed envelope
    # has ever carried a top-level `model` field to fall back to.
    if "model_ids" in record["manifest"]:
        sys.exit("TC-015c FAIL: modelUsage: manifest.model_ids=%r, want absent (no model resolved)"
                  % record["manifest"].get("model_ids"))
    if "model_id_source" in record["manifest"]:
        sys.exit("TC-015c FAIL: modelUsage: manifest.model_id_source=%r, want absent, never a 'model' fallback"
                  % record["manifest"].get("model_id_source"))

print("TC-015c: %s axis PASS (named error, metric absent, other fields intact)" % axis)
PYEOF
	done

	# Negative: an all-fields-present transcript produces zero
	# envelope_parse_error entries -- proves the check isn't unconditionally
	# firing (AC-05 negative).
	local out_negative="$WORKDIR/c-negative.out"
	run_collect "$FIXTURES_DIR/clean-completed" "$out_negative"
	python3 - "$out_negative" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if any(e["kind"] == "envelope_parse_error" for e in record.get("errors", [])):
    sys.exit("TC-015c FAIL: negative: clean-completed produced an envelope_parse_error, want none")
print("TC-015c: negative PASS (all-fields-present transcript produces zero envelope_parse_error entries)")
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
	mkdir -p "$stray_dir/run/transcripts"
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

# ---------------------------------------------------------------------------
# TC-015g (AC-19, ADR-F02-09): rejection crosscheck. Sub-case 1 (committed
# fixture): RunResult implies rework_loops=1 (one status re-entry after the
# "in_qa" gate); the committed .sql fixture seeds entity_history with 2
# backward transitions for the resolved entity -- a deliberate disagreement.
# Sub-case 2 (dynamic): both sources agree at zero rejections (the boundary
# AC-19 calls out: zero must not be special-cased as "skip the check").
# Sub-case 3 (dynamic, negative): an entity_key that fails to resolve to any
# row in the fixture DB must produce a distinct, separately named error, not
# a fabricated "0 backward transitions" agreement.
# ---------------------------------------------------------------------------
test_g() {
	local db_dir="$WORKDIR/db-crosscheck-disagree"
	cp -r "$FIXTURES_DIR/db-crosscheck-disagree" "$db_dir"
	sqlite3 "$db_dir/scratch/shark-tasks.db" <"$db_dir/scratch/scratch_db.sql"

	local out="$WORKDIR/g1.out"
	run_collect "$db_dir" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

rejections = record.get("rejections")
if rejections != {
    "by_gate": {"in_qa": 1},
    "rework_loops": 1,
    "crosscheck": {
        "entity_history_backward_transitions": 2,
        "work_session_outcomes": 1,
        "agrees": False,
    },
}:
    sys.exit("TC-015g FAIL: disagree sub-case: rejections=%r" % rejections)

errors = record.get("errors", [])
if len(errors) != 1 or errors[0]["kind"] != "crosscheck_disagreement":
    sys.exit("TC-015g FAIL: disagree sub-case: errors=%r, want exactly one crosscheck_disagreement" % errors)
detail = errors[0]["detail"]
if "runresult_inferred=1" not in detail or "entity_history_derived=2" not in detail:
    sys.exit("TC-015g FAIL: disagree sub-case: detail=%r does not name both counts" % detail)
print("TC-015g: disagree sub-case PASS")
PYEOF

	# Sub-case 2: both sources agree at zero rejections (dynamic -- a fresh
	# 2-stage RunResult with no re-entry, and a fresh DB with 0 backward
	# transitions, built in a temp copy so the committed disagree fixture
	# stays a clean, single-purpose fixture).
	local zero_dir="$WORKDIR/db-crosscheck-zero-agree"
	cp -r "$FIXTURES_DIR/db-crosscheck-disagree" "$zero_dir"
	rm -f "$zero_dir/run/transcripts/3-in_development-anthropic.log"
	python3 - "$zero_dir/run/stdout.json" <<'PYEOF'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    data = json.load(f)
data["stages"] = data["stages"][:2]
data["stages_completed"] = 2
data["total_duration_ns"] = sum(s["duration_ns"] for s in data["stages"])
with open(path, "w") as f:
    json.dump(data, f)
PYEOF
	sqlite3 "$zero_dir/scratch/shark-tasks.db" <<'SQL'
CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL UNIQUE);
CREATE TABLE entity_history (id INTEGER PRIMARY KEY AUTOINCREMENT, entity_type TEXT NOT NULL, entity_id INTEGER NOT NULL, from_status TEXT, to_status TEXT NOT NULL, changed_at DATETIME NOT NULL);
CREATE TABLE work_sessions (id INTEGER PRIMARY KEY AUTOINCREMENT, entity_type TEXT NOT NULL, entity_key TEXT, outcome TEXT, started_at TIMESTAMP NOT NULL, ended_at TIMESTAMP);
INSERT INTO tasks (id, key) VALUES (42, 'T-BENCH-DEMO-020');
INSERT INTO entity_history (entity_type, entity_id, from_status, to_status, changed_at) VALUES
    ('task', 42, NULL, 'in_development', '2026-08-06T11:00:00Z'),
    ('task', 42, 'in_development', 'in_qa', '2026-08-06T11:01:30Z');
SQL

	local out_zero="$WORKDIR/g2.out"
	run_collect "$zero_dir" "$out_zero"
	python3 - "$out_zero" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

crosscheck = record.get("rejections", {}).get("crosscheck")
if crosscheck != {"entity_history_backward_transitions": 0, "work_session_outcomes": 0, "agrees": True}:
    sys.exit("TC-015g FAIL: zero-agreement sub-case: crosscheck=%r" % crosscheck)
if any(e["kind"] == "crosscheck_disagreement" for e in record.get("errors", [])):
    sys.exit("TC-015g FAIL: zero-agreement sub-case: unexpected crosscheck_disagreement (zero must not be special-cased)")
print("TC-015g: zero-agreement boundary sub-case PASS")
PYEOF

	# Sub-case 3 (negative): entity_key does not resolve to any row.
	local unresolved_dir="$WORKDIR/db-crosscheck-unresolved"
	cp -r "$FIXTURES_DIR/db-crosscheck-disagree" "$unresolved_dir"
	sqlite3 "$unresolved_dir/scratch/shark-tasks.db" <"$unresolved_dir/scratch/scratch_db.sql"
	python3 - "$unresolved_dir/meta.json" <<'PYEOF'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    meta = json.load(f)
meta["entity_key"] = "T-BENCH-DEMO-999"
with open(path, "w") as f:
    json.dump(meta, f)
PYEOF

	local out_unresolved="$WORKDIR/g3.out"
	run_collect "$unresolved_dir" "$out_unresolved"
	python3 - "$out_unresolved" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

errors = record.get("errors", [])
resolution_errors = [e for e in errors if e["kind"] == "crosscheck_resolution_error"]
if len(resolution_errors) != 1:
    sys.exit("TC-015g FAIL: unresolved-key sub-case: errors=%r, want exactly one crosscheck_resolution_error" % errors)
if "T-BENCH-DEMO-999" not in resolution_errors[0]["detail"]:
    sys.exit("TC-015g FAIL: unresolved-key sub-case: detail=%r does not name the entity_key" % resolution_errors[0]["detail"])
if any(e["kind"] == "crosscheck_disagreement" for e in errors):
    sys.exit("TC-015g FAIL: unresolved-key sub-case: must never also fabricate a crosscheck_disagreement")
if "crosscheck" in record.get("rejections", {}):
    sys.exit("TC-015g FAIL: unresolved-key sub-case: rejections.crosscheck must be absent, not a fabricated '0 backward transitions' agreement")
print("TC-015g: unresolved-entity_key negative PASS")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015h (AC-02, AC-09): oracle.*/quality.* copied verbatim from post/*.
# The clean-completed fixture's post/test-diff.json and post/lint-diff.json
# were produced by a REAL `bench/scripts/diff-ledgers.sh` invocation (not
# hand-typed) over these hand-authored ledgers:
#
#   base-test.json entries: TestDiscountRounding/pass, TestDiscountFloor/pass,
#     TestDiscountLegacyFlag/pass
#   post-test.json entries: TestDiscountRounding/pass, TestDiscountFloor/fail
#   -> `diff-ledgers.sh --kind=test --base=base-test.json --post=post-test.json`
#      produced test-diff.json (1 regression: TestDiscountFloor pass->fail;
#      1 removed: TestDiscountLegacyFlag absent at post)
#
#   base-lint.json entries: staticcheck/pkg/pricing/pricing.go/SA4006
#   post-lint.json entries: the same staticcheck entry, plus
#     ineffassign/pkg/pricing/discount.go/"ineffectual assignment to err"
#   -> `diff-ledgers.sh --kind=lint --base=base-lint.json --post=post-lint.json`
#      produced lint-diff.json (1 new issue: the ineffassign entry)
#
# post/quality.json and post/f2p.json are hand-authored directly (there is
# no producing script for them to invoke yet -- T-E40-F02-003/004's scope);
# quality.json deliberately exercises all three of true/false/null across
# fmt/vet/tests so REQ-F-016's "null means could not execute, never a pass"
# posture has a test case, not just the true/false cases.
# ---------------------------------------------------------------------------
test_h() {
	local out="$WORKDIR/h.out"
	run_collect "$FIXTURES_DIR/clean-completed" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

oracle = record.get("oracle")
want_oracle = {
    "f2p_resolved": True,
    "p2p_regressions": [{"identity": "pkg/pricing::TestDiscountFloor"}],
    "p2p_regressions_count": 1,
    "removed": [{"identity": "pkg/pricing::TestDiscountLegacyFlag"}],
    "removed_count": 1,
}
if oracle != want_oracle:
    sys.exit("TC-015h FAIL: oracle=%r, want %r" % (oracle, want_oracle))

quality = record.get("quality")
want_quality = {
    "toolchain_guard": "pass",
    "fmt_clean": True,
    "vet_ok": False,
    "tests_pass": None,
    "gate_reasons": {"tests_pass": "go test binary timed out mid-suite"},
    "lint_new_issues": [
        {"from_linter": "ineffassign", "path": "pkg/pricing/discount.go", "text": "ineffectual assignment to err"}
    ],
    "lint_new_issues_count": 1,
}
if quality != want_quality:
    sys.exit("TC-015h FAIL: quality=%r, want %r" % (quality, want_quality))

sources = record.get("sources", {})
if sources.get("oracle") != "postrun":
    sys.exit("TC-015h FAIL: sources.oracle=%r, want 'postrun' (never a value implying independent recomputation)" % sources.get("oracle"))
if sources.get("quality") != "postrun":
    sys.exit("TC-015h FAIL: sources.quality=%r, want 'postrun'" % sources.get("quality"))
print("TC-015h: oracle/quality byte-identical-copy PASS")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015i (AC-10): toolchain-guard abort. Sub-case 1 (committed fixture): the
# guard failed, aborting the whole post-run phase before any diff would run
# -- errors[] names the axis, oracle/loc stay entirely absent. Sub-case 2
# (negative, reusing clean-completed's already-passing-guard fixture): a
# passing guard never produces postrun_check_aborted, even though the diff
# it gates later reports non-zero findings (clean-completed's own h fixture
# above already proves "non-zero" doesn't trip it; this asserts the absence
# directly).
# ---------------------------------------------------------------------------
test_i() {
	local out="$WORKDIR/i1.out"
	run_collect "$FIXTURES_DIR/toolchain-mismatch" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

errors = record.get("errors", [])
aborted = [e for e in errors if e["kind"] == "postrun_check_aborted"]
if len(aborted) != 1:
    sys.exit("TC-015i FAIL: abort sub-case: errors=%r, want exactly one postrun_check_aborted" % errors)
if "go_version" not in aborted[0]["detail"]:
    sys.exit("TC-015i FAIL: abort sub-case: detail=%r does not name the mismatched axis" % aborted[0]["detail"])
if "oracle" in record:
    sys.exit("TC-015i FAIL: abort sub-case: record.oracle must be absent, never a diff computed under the wrong toolchain")
if "loc" in record:
    sys.exit("TC-015i FAIL: abort sub-case: record.loc must be absent -- REQ-F-015's pinned order means LOC never ran either")
quality = record.get("quality")
if quality != {"toolchain_guard": aborted[0]["detail"]}:
    sys.exit("TC-015i FAIL: abort sub-case: quality=%r, want only the toolchain_guard detail" % quality)
print("TC-015i: toolchain-guard abort sub-case PASS")
PYEOF

	local out_negative="$WORKDIR/i2.out"
	run_collect "$FIXTURES_DIR/clean-completed" "$out_negative"
	python3 - "$out_negative" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

if any(e["kind"] == "postrun_check_aborted" for e in record.get("errors", [])):
    sys.exit("TC-015i FAIL: negative: a passing toolchain guard must never produce postrun_check_aborted, even with non-zero diff findings")
print("TC-015i: negative PASS (passing guard never aborts, regardless of diff findings)")
PYEOF
}

# ---------------------------------------------------------------------------
# TC-015l (AC-02, AC-18 arithmetic half): loc.* prod/test-split arithmetic
# over a synthetic post/numstat.txt with concrete, non-zero, mixed values --
# two production files, two _test.go files, and one binary file (`-`/`-`
# counts, contributing to files_touched only). Asserting the exact split
# values is itself the negative case this TC's description calls for: a
# `_test.go` file's lines miscounted into prod_added would fail this
# assertion by construction.
# ---------------------------------------------------------------------------
test_l() {
	local out="$WORKDIR/l.out"
	run_collect "$FIXTURES_DIR/loc-numstat-mixed" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    record = json.load(f)

loc = record.get("loc")
want = {"prod_added": 17, "prod_deleted": 4, "test_added": 42, "test_deleted": 2, "files_touched": 5}
if loc != want:
    sys.exit("TC-015l FAIL: loc=%r, want %r" % (loc, want))
if record.get("sources", {}).get("loc") != "postrun":
    sys.exit("TC-015l FAIL: sources.loc=%r, want 'postrun'" % record.get("sources", {}).get("loc"))
print("TC-015l: PASS (prod/test-split arithmetic over mixed numstat lines)")
PYEOF
}

# TC-015m: a cap-fired marker takes precedence over a RunResult that happened
# to arrive during the kill grace period, and a null quality gate retains its
# reason instead of becoming an unexplained missing measurement.
test_m() {
	local raced="$WORKDIR/m-raced" out="$WORKDIR/m-raced.out"
	cp -R "$FIXTURES_DIR/timeout-with-liveness" "$raced"
	cp "$FIXTURES_DIR/clean-completed/run/stdout.json" "$raced/run/stdout.json"
	python3 - "$raced/run/exit_status" <<'PYEOF'
import json
import sys
with open(sys.argv[1], "w") as f:
    json.dump({"exit_code": 0, "signaled": False, "timed_out": True}, f)
PYEOF
	run_collect "$raced" "$out"
	python3 - "$out" <<'PYEOF'
import json
import sys
record = json.load(open(sys.argv[1]))
if record.get("outcome") != "timeout":
    sys.exit("TC-015m FAIL: cap-fired record outcome=%r, want timeout" % record.get("outcome"))
if "runresult" in record or "oracle" in record or "quality" in record or "loc" in record:
    sys.exit("TC-015m FAIL: cap-fired record must not retain RunResult/post-run families: %r" % record)
print("TC-015m: cap-fired timeout precedence PASS")
PYEOF

	local unexecutable="$WORKDIR/m-unexecutable" quality_out="$WORKDIR/m-unexecutable.out"
	cp -R "$FIXTURES_DIR/clean-completed" "$unexecutable"
	python3 - "$unexecutable/post/quality.json" <<'PYEOF'
import json
import sys
with open(sys.argv[1], "w") as f:
    json.dump({
        "fmt": {"status": None, "reason": "gofmt could not execute (exit code 127)"},
        "vet": {"status": "pass"},
        "tests": {"status": "fail"},
    }, f)
PYEOF
	run_collect "$unexecutable" "$quality_out"
	python3 - "$quality_out" <<'PYEOF'
import json
import sys
quality = json.load(open(sys.argv[1])).get("quality", {})
if quality.get("fmt_clean") is not None:
    sys.exit("TC-015m FAIL: fmt_clean=%r, want null" % quality.get("fmt_clean"))
if quality.get("gate_reasons", {}).get("fmt_clean") != "gofmt could not execute (exit code 127)":
    sys.exit("TC-015m FAIL: missing null-gate reason: %r" % quality)
print("TC-015m: unexecutable gate reason PASS")
PYEOF

	local aborted="$WORKDIR/m-aborted" abort_out="$WORKDIR/m-aborted.out"
	cp -R "$FIXTURES_DIR/clean-completed" "$aborted"
	printf '%s\n' '{"step":"build_ledgers","detail":"fixture ledger builder failed"}' >"$aborted/post/postrun-abort.json"
	run_collect "$aborted" "$abort_out"
	python3 - "$abort_out" <<'PYEOF'
import json
import sys
record = json.load(open(sys.argv[1]))
errors = [e for e in record.get("errors", []) if e.get("kind") == "postrun_check_aborted"]
if len(errors) != 1 or "build_ledgers" not in errors[0].get("detail", ""):
    sys.exit("TC-015m FAIL: post-run abort was not recorded: %r" % record.get("errors"))
if "oracle" in record or "loc" in record or record.get("quality", {}).get("postrun_abort") != "build_ledgers":
    sys.exit("TC-015m FAIL: abort record retained invalid post-run metrics: %r" % record)
print("TC-015m: post-run abort record PASS")
PYEOF

	printf '%s\n' '{"step":"unknown","detail":"bad fixture"}' >"$aborted/post/postrun-abort.json"
	if "$COLLECT_SCRIPT" --run-dir "$aborted" >"$WORKDIR/m-malformed.out" 2>"$WORKDIR/m-malformed.err"; then
		fail "TC-015m: malformed postrun-abort marker unexpectedly succeeded"
	fi
	grep -q 'unrecognized step' "$WORKDIR/m-malformed.err" ||
		fail "TC-015m: malformed marker error did not name the contract violation"
	echo "TC-015m: malformed abort marker rejection PASS"
}

test_a
test_b
test_c
test_d
test_e
test_f
test_g
test_h
test_i
test_j
test_l
test_m

echo "TC-015: PASS"
