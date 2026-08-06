#!/usr/bin/env bash
# collect-run.sh --run-dir <dir>
#
# The E40-F02 collector (ADR-F02-01): a PURE FUNCTION of one already-completed
# run directory -> one I-02 JSONL record, printed to stdout. Never invokes
# `shark run`; never invoked by anything other than run-one.sh (T-E40-F02-003)
# at the end of its own run, or a human/test replaying a stored run directory.
#
# Run directory layout this script reads (spec.md "Interface contracts"):
#
#   <run-dir>/
#   |-- meta.json               driver-captured manifest inputs (contract below)
#   |-- run/
#   |   |-- stdout.json         `shark run --json` stdout, verbatim (absent on
#   |   |                       a killed run -- no RunResult was ever delivered)
#   |   |-- stderr.ndjson       liveness stream, verbatim: one plain-text
#   |   |                       "run.log: <path>" announce line, then one
#   |   |                       NDJSON object per line (absent/empty when no
#   |   |                       liveness stream exists at all)
#   |   |-- exit_status         process exit status + whether the cap fired
#   |   `-- transcripts/        copied from <scratch>/.shark/runs/<run_id>/*.log
#   `-- post/                   post-run check outputs (T-E40-F02-002's scope;
#                               not read by this task's collector logic)
#
# meta.json contract (this task's slice -- T-E40-F02-002/007 extend it):
#   item_id           string   corpus item id
#   item_type         string   "task" | "bug"
#   variant_id        string   workflow variant id
#   rep               integer  1-based repetition number
#   timeout_cap_s     integer  the run's timeout cap, seconds
#   harness_wall_ns   integer  driver-measured wall-clock duration (t1 - t0),
#                              nanoseconds -- always known, timeout or not,
#                              since the driver observes both process-group
#                              start and teardown regardless of outcome.
#   entity_key        string   the benched entity's key (timeout DB-status
#                              fallback lookup only; REQ-F-019/ADR-F02-02)
#   scratch_root      string   path to the scratch project root (relative
#                              paths resolve against <run-dir>), consulted
#                              only for the run_id newest-directory fallback
#                              (REQ-F-008) -- its ".shark/runs/" subdirectory
#                              is inspected, never anything else under it.
#   scratch_db_path   string   path to the scratch project's sqlite db file
#                              (relative paths resolve against <run-dir>),
#                              consulted only for the timeout DB-status
#                              fallback (uat-plan.md UAT-05).
# scratch_root/scratch_db_path are optional: a completed (non-timeout) run
# whose liveness stream resolved run_id needs neither.
#
# run/exit_status contract (JSON object); this task reads only `timed_out`:
#   {"exit_code": <int|null>, "signaled": <bool>, "timed_out": <bool>}
#
# Scope of THIS file as of T-E40-F02-001 (mirrors internal/runner/liveness.go's
# "Scope of THIS file as of T-..." convention for staged construction):
#   - outcome resolution: the five RunResult values, plus harness-assigned
#     "timeout" when stdout.json never delivered a RunResult and exit_status
#     says the cap fired (REQ-F-009/010).
#   - timeout_detail (present only on the timeout path), resolved from the
#     I-03 liveness stream primarily, the scratch DB's entity status as a
#     named fallback (REQ-F-019, ADR-F02-02). Pinned key: sources.stalled_stage
#     ("liveness" | "scratch_db").
#   - manifest.run_id / manifest.run_id_source, resolved from the liveness
#     stream (NDJSON run_id field, then the plain-text run.log announce line),
#     falling back to the newest directory under scratch_root/.shark/runs/
#     (REQ-F-008, ADR-F02-02).
#   - stages[] (all StageLog fields except the envelope `usage` sub-object)
#     and the spawn_agent/transcript reconciliation: aggregate count and
#     per-position filename-component checks (both -> stage_join_error, the
#     test-plan.md Ambiguity Finding split), then per-file readability
#     (-> transcript_missing) (REQ-F-011 shape only/REQ-F-013).
#   - timing.harness_wall_ns, copied from meta.json.
#   - the deterministic, sorted-key, single-line JSON encoding (REQ-N-004).
#
# NOT in this file's scope yet (added by later, sequenced tasks -- their
# fields are OMITTED here, never null-filled placeholders, per REQ-N-005):
#   - rejections.*, oracle.*, quality.*, loc.*                (T-E40-F02-002)
#   - stages[].usage (tokens/cost/model IDs), manifest.model_ids/
#     model_id_source, envelope_parse_error                   (T-E40-F02-007)
#
# REQ-N-006: bash entry point, embedded python3 for JSON/sqlite work,
# machine-readable JSON to stdout, diagnostics to stderr.
set -euo pipefail

usage() {
	echo "usage: collect-run.sh --run-dir <dir>" >&2
	exit 2
}

run_dir=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--run-dir)
		[[ $# -ge 2 ]] || usage
		run_dir="$2"
		shift 2
		;;
	--run-dir=*)
		run_dir="${1#--run-dir=}"
		shift
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$run_dir" ]] || usage
[[ -d "$run_dir" ]] || {
	echo "collect-run: run directory not found: $run_dir" >&2
	exit 1
}

python3 - "$run_dir" <<'PYEOF'
import json
import os
import re
import sqlite3
import sys

SCHEMA_VERSION = "1.0"
VALID_OUTCOMES = {"completed", "paused", "failed", "already_terminal", "no_action"}


def fail(msg):
    print("collect-run: " + msg, file=sys.stderr)
    sys.exit(1)


run_dir = sys.argv[1]


def rd(*parts):
    return os.path.join(run_dir, *parts)


def resolve(path):
    """Resolve a meta.json path field against run_dir when relative."""
    return path if os.path.isabs(path) else rd(path)


def load_json(path, required=True):
    if not os.path.isfile(path):
        if required:
            fail("required file not found: %s" % path)
        return None
    with open(path) as f:
        return json.load(f)


meta = load_json(rd("meta.json"))
exit_status = load_json(rd("run", "exit_status"))

record = {"schema_version": SCHEMA_VERSION}

# --- manifest (this task's slice: identity fields + run_id resolution) ---
manifest = {}
for key in ("item_id", "item_type", "variant_id", "rep", "timeout_cap_s"):
    if meta.get(key) is not None:
        manifest[key] = meta[key]
if all(k in manifest for k in ("item_id", "variant_id", "rep")):
    manifest["run_key"] = "%s::%s::rep%s" % (
        manifest["item_id"],
        manifest["variant_id"],
        manifest["rep"],
    )


def parse_liveness_stream(path):
    """Parses the I-03 stderr stream (test-plan.md's consumer rule: the
    "run.log: <path>" announce line is plain text, not NDJSON, and is
    skipped for event parsing but still yields a run_id candidate from the
    path's parent directory name).

    Returns (run_id_or_None, events) where events is the in-order list of
    parsed NDJSON objects (stage_start/heartbeat/stage_end).
    """
    if not os.path.isfile(path):
        return None, []
    found_run_id = None
    events = []
    with open(path) as f:
        for raw in f:
            line = raw.rstrip("\n")
            if not line:
                continue
            if line.startswith("run.log: "):
                announced_path = line[len("run.log: ") :].strip()
                candidate = os.path.basename(os.path.dirname(announced_path))
                if candidate and found_run_id is None:
                    found_run_id = candidate
                continue
            try:
                obj = json.loads(line)
            except ValueError:
                continue
            if found_run_id is None and obj.get("run_id"):
                found_run_id = obj["run_id"]
            events.append(obj)
    return found_run_id, events


def resolve_run_id_fallback():
    scratch_root = meta.get("scratch_root")
    if not scratch_root:
        fail(
            "no liveness stream (run/stderr.ndjson) and no meta.json "
            "scratch_root to fall back to (REQ-F-008)"
        )
    runs_dir = os.path.join(resolve(scratch_root), ".shark", "runs")
    if not os.path.isdir(runs_dir):
        fail("run_id fallback: runs directory not found: %s" % runs_dir)
    candidates = [
        d
        for d in os.listdir(runs_dir)
        if os.path.isdir(os.path.join(runs_dir, d))
    ]
    if not candidates:
        fail("run_id fallback: no candidate run directories under %s" % runs_dir)
    candidates.sort(key=lambda d: os.path.getmtime(os.path.join(runs_dir, d)))
    return candidates[-1]


stream_run_id, liveness_events = parse_liveness_stream(rd("run", "stderr.ndjson"))
if stream_run_id:
    manifest["run_id"] = stream_run_id
    manifest["run_id_source"] = "liveness_stream"
else:
    manifest["run_id"] = resolve_run_id_fallback()
    manifest["run_id_source"] = "fallback_newest_dir"

record["manifest"] = manifest

# --- timing (driver-measured; independent of outcome/RunResult) ---
if meta.get("harness_wall_ns") is not None:
    record["timing"] = {"harness_wall_ns": meta["harness_wall_ns"]}

errors = []


def add_error(kind, detail, stage_index=None, path=None):
    entry = {"kind": kind, "detail": detail}
    if stage_index is not None:
        entry["stage_index"] = stage_index
    if path is not None:
        entry["path"] = path
    errors.append(entry)


TRANSCRIPT_NAME_RE = re.compile(r"^(\d+)-(.+)-([^-]+)\.log$")


def reconcile_transcripts(stages_in):
    """REQ-F-013: reconciles spawn_agent stages against run/transcripts/.

    Order matters (test-plan.md TC-015m's counter-factual): aggregate count,
    THEN per-position filename-component match, THEN per-file readability --
    a zero-byte file at an otherwise count-/component-correct position must
    read as transcript_missing, never as a spurious stage_join_error.

    Matching is keyed by the transcript filename's own numeric prefix
    against the stage's 1-based position in RunResult.Stages (that prefix
    IS the run loop's iteration counter -- internal/runner/controller.go's
    stageN -- which increments once per StageLog entry regardless of action,
    so it equals the stage's array position exactly). A file whose prefix
    does not correspond to any spawn_agent stage (e.g. a stray file at an
    advance_status stage's own index) is silently excluded, never
    misattributed to a different stage (test-plan.md AC-06 negative).
    """
    expected = {}
    stages_out = []
    for idx, stage in enumerate(stages_in, start=1):
        entry = {
            "index": idx,
            "status": stage.get("status", ""),
            "action": stage.get("action", ""),
            "duration_ns": stage.get("duration_ns", 0),
            "exit_code": stage.get("exit_code", 0),
        }
        if stage.get("agent_type"):
            entry["agent_type"] = stage["agent_type"]
        if stage.get("provider"):
            entry["provider"] = stage["provider"]
        if stage.get("action") == "spawn_agent":
            expected[idx] = (stage.get("status", ""), stage.get("provider", ""))
        stages_out.append(entry)

    transcripts_dir = rd("run", "transcripts")
    actual = {}
    if os.path.isdir(transcripts_dir):
        for name in os.listdir(transcripts_dir):
            m = TRANSCRIPT_NAME_RE.match(name)
            if not m:
                continue
            idx = int(m.group(1))
            actual[idx] = (m.group(2), m.group(3), os.path.join(transcripts_dir, name))

    matched = sorted(i for i in expected if i in actual)
    expected_count = len(expected)
    observed_count = len(matched)

    if observed_count != expected_count:
        add_error(
            "stage_join_error",
            "expected %d spawn_agent transcript(s), found %d"
            % (expected_count, observed_count),
        )
        return stages_out

    for idx in matched:
        exp_status, exp_provider = expected[idx]
        act_status, act_provider, path = actual[idx]
        if (act_status, act_provider) != (exp_status, exp_provider):
            add_error(
                "stage_join_error",
                "stage %d transcript filename mismatch: expected status=%s "
                "provider=%s, found status=%s provider=%s"
                % (idx, exp_status, exp_provider, act_status, act_provider),
                stage_index=idx,
            )
            continue
        rel_path = os.path.relpath(path, run_dir)
        if os.path.getsize(path) == 0:
            add_error(
                "transcript_missing",
                "stage %d transcript is present but empty/unreadable" % idx,
                stage_index=idx,
                path=rel_path,
            )
        else:
            stages_out[idx - 1]["transcript_path"] = rel_path

    return stages_out


def resolve_timeout_detail():
    """AC-03/REQ-F-019: the stalled stage, sourced from the liveness stream
    primarily, falling back to the scratch DB's entity status (UAT-05).
    """
    opened = {}
    closed = set()
    for ev in liveness_events:
        iteration = ev.get("iteration")
        event = ev.get("event")
        if event == "stage_start":
            opened[iteration] = ev
        elif event == "heartbeat" and iteration in opened:
            opened[iteration] = ev
        elif event == "stage_end":
            closed.add(iteration)

    stalled_iterations = [it for it in opened if it not in closed and it]
    if stalled_iterations:
        it = max(stalled_iterations)
        ev = opened[it]
        detail = {
            "stage_index": it,
            "status": ev.get("status", ""),
            "action": ev.get("action", ""),
            "agent_type": ev.get("agent_type", ""),
            "provider": ev.get("provider", ""),
            "source": "liveness_stream",
        }
        return detail, "liveness"

    entity_key = meta.get("entity_key")
    item_type = meta.get("item_type")
    scratch_db_path = meta.get("scratch_db_path")
    if not (entity_key and item_type and scratch_db_path):
        fail(
            "timeout with no liveness stalled-stage and no meta.json "
            "entity_key/item_type/scratch_db_path for the DB-status fallback"
        )
    table = "tasks" if item_type == "task" else "bugs"
    conn = sqlite3.connect(resolve(scratch_db_path))
    try:
        row = conn.execute(
            "SELECT status FROM %s WHERE key = ?" % table, (entity_key,)
        ).fetchone()
    finally:
        conn.close()
    if row is None:
        fail(
            "timeout DB-status fallback: entity %s not found in scratch DB %s table"
            % (entity_key, table)
        )
    detail = {
        "stage_index": None,
        "status": row[0],
        "action": None,
        "agent_type": None,
        "provider": None,
        "source": "scratch_db_status_fallback",
    }
    return detail, "scratch_db"


# --- outcome resolution (REQ-F-009/010) ---
stdout_path = rd("run", "stdout.json")
run_result = None
if os.path.isfile(stdout_path) and os.path.getsize(stdout_path) > 0:
    try:
        with open(stdout_path) as f:
            candidate = json.load(f)
        if isinstance(candidate, dict) and candidate.get("outcome") in VALID_OUTCOMES:
            run_result = candidate
    except ValueError:
        run_result = None

if run_result is not None:
    record["outcome"] = run_result["outcome"]

    runresult_block = {
        "final_status": run_result.get("final_status", ""),
        "stages_completed": run_result.get("stages_completed", 0),
        "total_duration_ns": run_result.get("total_duration_ns", 0),
    }
    if run_result.get("error"):
        runresult_block["error"] = run_result["error"]
    if run_result.get("question_block") is not None:
        runresult_block["question_block"] = run_result["question_block"]
    record["runresult"] = runresult_block

    record["stages"] = reconcile_transcripts(run_result.get("stages") or [])
elif isinstance(exit_status, dict) and exit_status.get("timed_out") is True:
    record["outcome"] = "timeout"
    detail, source = resolve_timeout_detail()
    record["timeout_detail"] = detail
    record["sources"] = {"stalled_stage": source}
else:
    fail(
        "run/stdout.json missing/unparseable and exit_status does not "
        "indicate a timeout -- cannot determine outcome (REQ-N-005: refusing "
        "to fabricate one)"
    )

errors.sort(key=lambda e: (e["kind"], e.get("stage_index", -1), e.get("path", "")))
record["errors"] = errors

print(json.dumps(record, sort_keys=True, separators=(",", ":")))
PYEOF
