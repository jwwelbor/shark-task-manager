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
# post/ contract (T-E40-F02-002's slice; ALL of post/ is optional -- a run
# directory with no post/ at all, or no post/toolchain-guard.json, simply
# gets no oracle/quality/loc blocks on its record, never a fabricated one).
# This is this task's own pinned extension of spec.md's Interface contracts
# post/ layout, since neither spec.md nor test-plan.md pins the exact
# machine-readable shape of the post-run check outputs (only the raw-output
# filenames fmt.txt/vet.txt/test.txt/numstat.txt are named there) -- the
# driver side that WRITES these files is T-E40-F02-003/004's scope, so this
# is the read contract they must produce against:
#
#   post/toolchain-guard.json   {"status": "pass"} |
#                                {"status": "fail", "mismatches": [<str>, ...]}
#                                REQ-F-015 step 1: absent entirely means no
#                                post-run phase ran at all (e.g. a timed-out
#                                run) -- oracle/quality/loc all stay absent.
#                                "fail" means the guard aborted the WHOLE
#                                post-run phase (REQ-F-015): quality gets
#                                only `toolchain_guard`, oracle and loc are
#                                both absent, and errors[] gets one
#                                postrun_check_aborted entry naming the
#                                mismatched axis/axes (ADR-F02-06: this
#                                script copies diff-ledgers.sh's own
#                                mismatch text, never re-derives it).
#   post/test-diff.json         diff-ledgers.sh --kind=test stdout, verbatim
#                                (regressions/regressions_count/removed/
#                                removed_count) -> oracle.p2p_regressions*/
#                                .removed* (AC-09: copied, not recomputed).
#   post/lint-diff.json         diff-ledgers.sh --kind=lint stdout, verbatim
#                                (new_issues/new_issues_count) ->
#                                quality.lint_new_issues* (AC-09).
#   post/quality.json           {"fmt": {"status": "pass"|"fail"|null,
#                                         "reason": <str, present iff null>},
#                                 "vet": {...same shape...},
#                                 "tests": {...same shape...}}
#                                -> quality.fmt_clean/.vet_ok/.tests_pass
#                                (REQ-F-016: null means the gate could not
#                                be executed, never silently a pass).
#   post/f2p.json                {"f2p_resolved": <bool|null>,
#                                 "repro_confirmed": <bool|null>}  (the
#                                latter present only for bug items, per
#                                spec.md's data model) -> oracle.f2p_resolved/
#                                .repro_confirmed.
#   post/numstat.txt            `git diff --numstat` text, tab-separated
#                                "<added>\t<deleted>\t<path>" per line, `-`
#                                for both counts on a binary file -> loc.*
#                                (REQ-F-017): a `_test.go` path's counts go
#                                to test_added/test_deleted, every other
#                                path's counts go to prod_added/prod_deleted;
#                                files_touched counts every line including
#                                binary ones.
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
# Scope of THIS file as of T-E40-F02-002 (added on top of the above):
#   - rejections.by_gate/.rework_loops: inferred purely from RunResult's
#     stages[] -- a stage whose `status` re-enters a status already seen at
#     an earlier stage is one rejection, attributed to the immediately
#     preceding stage's status (the "gate" whose outcome sent work back)
#     (REQ-F-014).
#   - rejections.crosscheck: only attempted when meta.json supplies
#     scratch_db_path + entity_key + item_type (older/unrelated synthetic
#     fixtures that predate this task's DB fixture convention keep working
#     with no crosscheck sub-object at all, never a fabricated one).
#     Resolves entity_key -> entity_id against the scratch DB's tasks/bugs
#     table first (ADR-F02-09); a resolution failure is its own named error
#     (crosscheck_resolution_error -- a documented extension of spec.md's
#     six-value errors[].kind set this task's AC-19 negative case requires,
#     since crosscheck_disagreement itself only covers a resolved
#     disagreement, not an unresolvable key). A resolved id's backward-
#     transition count (same "re-entry into an already-seen value" measure,
#     applied to entity_history.to_status in changed_at order) disagreeing
#     with the RunResult-inferred rework_loops is crosscheck_disagreement,
#     never silently resolved either way (REQ-F-014, AC-19).
#     work_session_outcomes is a supplementary count (work_sessions rows
#     with outcome='blocked' -- the only SessionOutcome value adjacent to a
#     mid-session rejection) and does not itself participate in the
#     agree/disagree comparison.
#   - oracle.*/quality.*/loc.*: copied from post/*, per the post/ contract
#     above -- never recomputed (ADR-F02-06, AC-09), never fabricated when
#     the source file is absent (REQ-N-005).
#
# Scope of THIS file as of T-E40-F02-007 (added on top of the above): parses
# each spawn_agent stage's transcript envelope (the Envelope extraction rule
# in spec.md's Interface contracts: split on the FIRST "\n---STDOUT---\n"
# and the LAST "\n---STDERR---\n"; the text between is the agent's raw
# stdout, one JSON object) and extracts stages[].usage using the field
# names T-E40-F02-006 confirmed from a real capture (task note,
# 2026-08-07): modelUsage (camelCase, object keyed by canonical model ID),
# num_turns and duration_api_ms (top-level, snake_case), plus the
# pre-existing flat usage sub-object and total_cost_usd. Every expected
# field is checked independently -- a missing/renamed field is its own
# named envelope_parse_error and is simply absent from stages[].usage,
# never a zero (REQ-F-012/AC-05). manifest.model_ids/model_id_source are
# derived from modelUsage's own object keys, deduplicated and sorted across
# every stage that resolved one; NO fallback to a top-level `model` field is
# implemented -- T-E40-F02-006's real capture found no such field in any
# observed envelope, so a modelUsage-absent envelope is its own fail-loud
# error, never a reach for an unconfirmed field (spec.md REQ-F-021,
# amended; E40-F02 decision note, 2026-08-07). This closes Q003.
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
# "seeded_keys" is T-E40-F02-003's own addition (driver core): the driver
# writes meta.json's seeded_keys object (the assigned epic/feature/task or
# bug key(s), captured verbatim from each `shark create ... --json`
# response -- AC-16), and this pass-through is what puts it on the record
# without inventing a value; a meta.json without the key (older/synthetic
# fixtures) simply omits it here too, per the same optional-field guard the
# rest of this loop already uses.
#
# Code-review round 1 kickback (Finding B-1, BLOCKER): fixture_base_sha/
# corpus_schema_version/p2p_set/variant_bundle_sha256/shark_version/
# shark_binary_sha256 are the G7-reproducibility manifest fields
# architecture.md#metric-collection-and-artifact-schema ("fixture and
# bundle SHAs ... exact model IDs, timeout cap"), E40-interaction-map.md's
# I-02 row, and Phase-1-exit-gating uat-plan.md UAT-07 ("pinned fixture
# SHA ... same variant bundle ... The report regenerates from the artifact
# directory alone, with no state outside it" -- ADR-002) all name. T-E40-
# F02-003's rework (2026-08-06) writes all six to meta.json; this pass-
# through is what puts them on the record, via the exact same optional-
# field guard, so an older/synthetic fixture predating that rework keeps
# working with the fields simply absent, never a fabricated placeholder.
manifest = {}
for key in (
    "item_id",
    "item_type",
    "variant_id",
    "rep",
    "timeout_cap_s",
    "seeded_keys",
    "fixture_base_sha",
    "corpus_schema_version",
    "p2p_set",
    "variant_bundle_sha256",
    "shark_version",
    "shark_binary_sha256",
):
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
sources = {}


def add_error(kind, detail, stage_index=None, path=None):
    entry = {"kind": kind, "detail": detail}
    if stage_index is not None:
        entry["stage_index"] = stage_index
    if path is not None:
        entry["path"] = path
    errors.append(entry)


TRANSCRIPT_NAME_RE = re.compile(r"^(\d+)-(.+)-([^-]+)\.log$")

STDOUT_MARKER = "\n---STDOUT---\n"
STDERR_MARKER = "\n---STDERR---\n"

# T-E40-F02-006's confirmed field names (task note, 2026-08-07): top-level
# scalars, snake_case, read verbatim off the envelope.
ENVELOPE_SCALAR_FIELDS = (
    ("num_turns", "num_turns"),
    ("duration_api_ms", "duration_api_ms"),
    ("total_cost_usd", "total_cost_usd"),
)
# The pre-existing flat `usage` sub-object's own sub-fields (snake_case,
# distinct from modelUsage's camelCase per-model fields).
ENVELOPE_USAGE_SUBFIELDS = (
    "input_tokens",
    "output_tokens",
    "cache_read_input_tokens",
    "cache_creation_input_tokens",
)

# Deduplicated model IDs across every spawn_agent stage that resolved one
# (manifest.model_ids, REQ-F-021) -- accumulated as reconcile_transcripts
# parses each stage's envelope below.
ALL_MODEL_IDS = set()


def extract_envelope(path):
    """spec.md "Envelope extraction rule": split the transcript on the
    FIRST "\n---STDOUT---\n" and the LAST "\n---STDERR---\n"; the text
    between is the agent's raw stdout, parsed as one JSON object. A
    transcript that does not match this shape is a named parse error, never
    a skipped stage. Returns (envelope_dict_or_None, error_detail_or_None).
    """
    with open(path) as f:
        content = f.read()
    stderr_idx = content.rfind(STDERR_MARKER)
    if stderr_idx == -1:
        return None, "transcript has no '---STDERR---' marker"
    head = content[:stderr_idx]
    stdout_idx = head.find(STDOUT_MARKER)
    if stdout_idx == -1:
        return None, "transcript has no '---STDOUT---' marker"
    raw = head[stdout_idx + len(STDOUT_MARKER):]
    try:
        envelope = json.loads(raw)
    except ValueError as exc:
        return None, "transcript ---STDOUT--- block is not valid JSON: %s" % exc
    if not isinstance(envelope, dict):
        return None, "transcript ---STDOUT--- block did not decode to a JSON object"
    return envelope, None


def extract_stage_usage(envelope, stage_index, rel_path):
    """REQ-F-011/012/REQ-F-021: extracts a spawn_agent stage's `usage`
    sub-object from its parsed envelope. Every expected field is checked
    independently -- a missing field produces its own named
    envelope_parse_error and is simply absent from the returned dict, never
    a zero, empty string, or fabricated value (REQ-N-005). No fallback to a
    top-level `model` field is implemented: T-E40-F02-006's real capture
    found no such field in any observed envelope (decision note,
    2026-08-07), so spec.md's REQ-F-021 fallback sentence is amended to
    match -- a modelUsage-absent envelope is its own fail-loud error, never
    a reach for an unobserved field.
    """
    usage = {}
    for src_key, out_key in ENVELOPE_SCALAR_FIELDS:
        if src_key in envelope:
            usage[out_key] = envelope[src_key]
        else:
            add_error(
                "envelope_parse_error",
                "transcript envelope missing expected field %r" % src_key,
                stage_index=stage_index,
                path=rel_path,
            )

    usage_obj = envelope.get("usage")
    if not isinstance(usage_obj, dict):
        add_error(
            "envelope_parse_error",
            "transcript envelope missing expected field 'usage'",
            stage_index=stage_index,
            path=rel_path,
        )
    else:
        for sub_key in ENVELOPE_USAGE_SUBFIELDS:
            if sub_key in usage_obj:
                usage[sub_key] = usage_obj[sub_key]
            else:
                add_error(
                    "envelope_parse_error",
                    "transcript envelope missing expected field 'usage.%s'" % sub_key,
                    stage_index=stage_index,
                    path=rel_path,
                )

    model_usage = envelope.get("modelUsage")
    if not isinstance(model_usage, dict):
        add_error(
            "envelope_parse_error",
            "transcript envelope missing expected field 'modelUsage'",
            stage_index=stage_index,
            path=rel_path,
        )
    else:
        model_ids = sorted(model_usage.keys())
        usage["model_ids"] = model_ids
        ALL_MODEL_IDS.update(model_ids)

    return usage


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
            continue
        stages_out[idx - 1]["transcript_path"] = rel_path
        envelope, err = extract_envelope(path)
        if err is not None:
            add_error("envelope_parse_error", err, stage_index=idx, path=rel_path)
        else:
            usage = extract_stage_usage(envelope, idx, rel_path)
            if usage:
                stages_out[idx - 1]["usage"] = usage

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


def infer_rejections_from_stages(stages):
    """REQ-F-014, RunResult-only half: walking `stages[]` in array order, a
    stage whose `status` re-enters a status already seen at an EARLIER stage
    is one rejection, attributed to the status of the stage immediately
    preceding the re-entry (the "gate" -- whatever that stage evaluated is
    what sent work back to an already-visited status). Returns (by_gate,
    rework_loops).
    """
    seen = set()
    by_gate = {}
    for idx, stage in enumerate(stages):
        status = stage.get("status", "")
        if idx > 0 and status in seen:
            gate = stages[idx - 1].get("status", "")
            by_gate[gate] = by_gate.get(gate, 0) + 1
        seen.add(status)
    rework_loops = sum(by_gate.values())
    return by_gate, rework_loops


def resolve_entity_id(conn, item_type, entity_key):
    table = "tasks" if item_type == "task" else "bugs"
    row = conn.execute(
        "SELECT id FROM %s WHERE key = ?" % table, (entity_key,)
    ).fetchone()
    return row[0] if row else None


def count_backward_transitions(conn, item_type, entity_id):
    """ADR-F02-09's DB-derived half of REQ-F-014: entity_history rows for
    this entity, ordered by changed_at, applying the SAME "re-entry into an
    already-seen value" measure as infer_rejections_from_stages above, but
    over `to_status` instead of stage `status` -- the same thing, measured
    by a different means, which is the entire point of a crosscheck.
    """
    rows = conn.execute(
        "SELECT to_status FROM entity_history "
        "WHERE entity_type = ? AND entity_id = ? "
        "ORDER BY changed_at ASC, id ASC",
        (item_type, entity_id),
    ).fetchall()
    seen = set()
    backward = 0
    for (to_status,) in rows:
        if to_status in seen:
            backward += 1
        else:
            seen.add(to_status)
    return backward


def count_blocked_work_sessions(conn, item_type, entity_key):
    row = conn.execute(
        "SELECT COUNT(*) FROM work_sessions "
        "WHERE entity_type = ? AND entity_key = ? AND outcome = 'blocked'",
        (item_type, entity_key),
    ).fetchone()
    return row[0] if row else 0


def compute_rejections(run_result):
    """REQ-F-014/ADR-F02-09/AC-19. The RunResult-only half (by_gate/
    rework_loops) always runs when a RunResult was delivered. The DB
    crosscheck half only runs when meta.json supplies scratch_db_path +
    entity_key + item_type -- synthetic fixtures that predate this task's DB
    fixture convention keep passing with no crosscheck sub-object, never a
    fabricated one (REQ-N-005).
    """
    by_gate, rework_loops = infer_rejections_from_stages(run_result.get("stages") or [])
    rejections = {"by_gate": by_gate, "rework_loops": rework_loops}

    scratch_db_path = meta.get("scratch_db_path")
    entity_key = meta.get("entity_key")
    item_type = meta.get("item_type")
    if scratch_db_path and entity_key and item_type:
        conn = sqlite3.connect(resolve(scratch_db_path))
        try:
            entity_id = resolve_entity_id(conn, item_type, entity_key)
            if entity_id is None:
                add_error(
                    "crosscheck_resolution_error",
                    "entity_key %r did not resolve to any %s.id in the scratch DB"
                    % (entity_key, "tasks" if item_type == "task" else "bugs"),
                )
            else:
                entity_history_derived = count_backward_transitions(conn, item_type, entity_id)
                work_session_outcomes = count_blocked_work_sessions(conn, item_type, entity_key)
                agrees = entity_history_derived == rework_loops
                rejections["crosscheck"] = {
                    "entity_history_backward_transitions": entity_history_derived,
                    "work_session_outcomes": work_session_outcomes,
                    "agrees": agrees,
                }
                if not agrees:
                    add_error(
                        "crosscheck_disagreement",
                        "runresult_inferred=%d entity_history_derived=%d"
                        % (rework_loops, entity_history_derived),
                    )
        finally:
            conn.close()

    return rejections


QUALITY_GATE_STATUS = {"pass": True, "fail": False, None: None}


def parse_numstat(path):
    """REQ-F-017: `git diff --numstat` text, tab-separated. A `_test.go`
    path's counts go to test_added/test_deleted, every other path's counts
    go to prod_added/prod_deleted. A binary file (`-`/`-` counts) never
    contributes to either line-count total but still counts toward
    files_touched.
    """
    prod_added = prod_deleted = test_added = test_deleted = 0
    files_touched = 0
    with open(path) as f:
        for raw in f:
            line = raw.rstrip("\n")
            if not line:
                continue
            parts = line.split("\t")
            if len(parts) != 3:
                fail("post/numstat.txt: malformed line (want 3 tab-separated fields): %r" % line)
            added_s, deleted_s, path_field = parts
            files_touched += 1
            if added_s == "-" or deleted_s == "-":
                continue
            try:
                added = int(added_s)
                deleted = int(deleted_s)
            except ValueError:
                fail("post/numstat.txt: non-integer line counts: %r" % line)
            if path_field.endswith("_test.go"):
                test_added += added
                test_deleted += deleted
            else:
                prod_added += added
                prod_deleted += deleted
    return {
        "prod_added": prod_added,
        "prod_deleted": prod_deleted,
        "test_added": test_added,
        "test_deleted": test_deleted,
        "files_touched": files_touched,
    }


def read_post_json(*parts):
    path = rd("post", *parts)
    if not os.path.isfile(path):
        return None
    with open(path) as f:
        return json.load(f)


def compute_postrun():
    """REQ-F-015 pinned order/ADR-F02-06/ADR-F02-11: oracle.*/quality.*/
    loc.* are copied from post/*, never recomputed. Absence of
    post/toolchain-guard.json means no post-run phase ran at all for this
    run (e.g. a timed-out run) -- every family stays absent, not fabricated.
    A failed guard aborts the WHOLE post-run phase (REQ-F-015): quality
    carries only `toolchain_guard`, oracle and loc stay entirely absent.
    """
    guard = read_post_json("toolchain-guard.json")
    if guard is None:
        return

    if guard.get("status") == "fail":
        mismatches = guard.get("mismatches") or []
        detail = "; ".join(mismatches) if mismatches else "toolchain mismatch (no axis detail recorded)"
        add_error("postrun_check_aborted", detail)
        record["quality"] = {"toolchain_guard": detail}
        sources["quality"] = "postrun"
        return

    quality = {"toolchain_guard": "pass"}
    oracle = {}

    lint_diff = read_post_json("lint-diff.json")
    if lint_diff is not None:
        if "new_issues" not in lint_diff or "new_issues_count" not in lint_diff:
            fail("post/lint-diff.json missing expected diff-ledgers.sh field(s)")
        quality["lint_new_issues"] = lint_diff["new_issues"]
        quality["lint_new_issues_count"] = lint_diff["new_issues_count"]

    test_diff = read_post_json("test-diff.json")
    if test_diff is not None:
        for field in ("regressions", "regressions_count", "removed", "removed_count"):
            if field not in test_diff:
                fail("post/test-diff.json missing expected diff-ledgers.sh field: %s" % field)
        oracle["p2p_regressions"] = test_diff["regressions"]
        oracle["p2p_regressions_count"] = test_diff["regressions_count"]
        oracle["removed"] = test_diff["removed"]
        oracle["removed_count"] = test_diff["removed_count"]

    quality_gates = read_post_json("quality.json")
    if quality_gates is not None:
        for gate_key, record_key in (("fmt", "fmt_clean"), ("vet", "vet_ok"), ("tests", "tests_pass")):
            gate = quality_gates.get(gate_key)
            if gate is None:
                continue
            gate_status = gate.get("status")
            if gate_status not in QUALITY_GATE_STATUS:
                fail("post/quality.json: unrecognized %s status %r" % (gate_key, gate_status))
            quality[record_key] = QUALITY_GATE_STATUS[gate_status]
    else:
        # A raw gate-output file with no post/quality.json sidecar is a
        # producer-contract violation, not "gate not run": the driver wrote
        # evidence of having executed the gate but no pass/fail/null verdict
        # to read it back as (REQ-N-005 -- refuse rather than silently drop
        # the family, the same posture the diff-ledgers.sh field checks
        # above take on a malformed producer file).
        for raw_name in ("fmt.txt", "vet.txt", "test.txt"):
            if os.path.isfile(rd("post", raw_name)):
                fail(
                    "post/%s is present but post/quality.json (the gate "
                    "status sidecar) is absent -- cannot determine pass/"
                    "fail/null without fabricating one" % raw_name
                )

    f2p = read_post_json("f2p.json")
    if f2p is not None:
        if "f2p_resolved" in f2p:
            oracle["f2p_resolved"] = f2p["f2p_resolved"]
        if "repro_confirmed" in f2p:
            oracle["repro_confirmed"] = f2p["repro_confirmed"]

    if oracle:
        record["oracle"] = oracle
        sources["oracle"] = "postrun"
    if quality:
        record["quality"] = quality
        sources["quality"] = "postrun"

    numstat_path = rd("post", "numstat.txt")
    if os.path.isfile(numstat_path):
        record["loc"] = parse_numstat(numstat_path)
        sources["loc"] = "postrun"


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
    if ALL_MODEL_IDS:
        manifest["model_ids"] = sorted(ALL_MODEL_IDS)
        manifest["model_id_source"] = "modelUsage"
    record["rejections"] = compute_rejections(run_result)
elif isinstance(exit_status, dict) and exit_status.get("timed_out") is True:
    record["outcome"] = "timeout"
    detail, source = resolve_timeout_detail()
    record["timeout_detail"] = detail
    sources["stalled_stage"] = source
else:
    fail(
        "run/stdout.json missing/unparseable and exit_status does not "
        "indicate a timeout -- cannot determine outcome (REQ-N-005: refusing "
        "to fabricate one)"
    )

compute_postrun()

if sources:
    record["sources"] = sources

# Stable sort: entries sharing a (kind, stage_index, path) key -- e.g.
# multiple envelope_parse_error entries for one stage -- retain their
# original insertion order, which is the fixed field-check order
# ENVELOPE_SCALAR_FIELDS -> ENVELOPE_USAGE_SUBFIELDS -> modelUsage. That is
# the determinism guarantee for a multi-error stage (REQ-N-004).
errors.sort(key=lambda e: (e["kind"], e.get("stage_index", -1), e.get("path", "")))
record["errors"] = errors

print(json.dumps(record, sort_keys=True, separators=(",", ":")))
PYEOF
