#!/usr/bin/env bash
# TC-083 / T-E40-F10-011: offline diagnosis of a completed-but-failed-oracle
# run (spec.md REQ-F-006; test-plan.md TC-083 full body; AC-006).
#
# Exercises the REAL aggregate-lifecycle.sh and report-lifecycle.sh binaries
# over hand-built retention roots, each with real evidence/transcripts/
# oracle.json artifacts on disk so this test can assert the headline
# report's printed evidence paths, resolved against the retention root,
# name real files -- reachability by direct file-path, never by
# re-invoking any script (TC-083 Observability Evidence).
#
# Four scenarios, one retention root:
#   scenario-notcorrect -- I-07 outcome.publication_eligible is TRUE (proves
#     the headline verdict is read from I-08, never I-07); I-08
#     execution_oracle.observed_result is `fail`; I-08
#     eligibility.publication_eligible is `false`. AC-006's core case: the
#     headline reports NOT correct, never a false pass inferred from
#     terminal status alone.
#   scenario-defect -- oracle fail AND (inconsistently) I-08
#     eligibility.publication_eligible TRUE. TC-083 Edge Case: the oracle-
#     fail evidence still surfaces, and the report names this an upstream
#     contract defect under REQ-F-008 rather than silently trusting either
#     field alone.
#   scenario-oracle-absent -- execution_oracle is `{}` (no observed_result
#     key at all). TC-083 Negative Case: must render distinctly from a
#     passed oracle.
#   scenario-oracle-passed -- execution_oracle.observed_result is `pass`.
#     Contrasts with scenario-oracle-absent.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
REPORTER="$SCRIPTS_DIR/report-lifecycle.sh"

fail() {
	echo "TC-083 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
[[ -x "$REPORTER" ]] || fail "bench/scripts/report-lifecycle.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

ROOT="$WORKDIR/root"
mkdir -p "$ROOT"

python3 - "$ROOT" <<'PYEOF'
import hashlib
import json
import os
import sys

dest_root = sys.argv[1]


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def write_json(path, obj):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")


def base_stage():
    return {
        "dispatch_ordinal": 1, "stage": "code", "category": "code",
        "snapshot_digest": "a" * 64, "prompt_digest": "b" * 64,
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": {"provider": "fixture", "model": "fixture-model"},
        "cost_usd": 1.0, "elapsed_seconds": 1.0, "errors": [], "rework": False,
        "intervals": [{"category": "provider_active", "start": 0, "end": 1}],
        "candidate": {}, "artifacts": [], "access_events": [], "evidence_refs": {},
    }


def build_pair(scenario_id, execution_oracle, eligibility, i07_outcome_publication_eligible):
    pair_dir = os.path.join(dest_root, "scenarios", scenario_id, "1")
    os.makedirs(pair_dir, exist_ok=True)
    os.makedirs(os.path.join(pair_dir, "evidence"), exist_ok=True)
    os.makedirs(os.path.join(pair_dir, "transcripts"), exist_ok=True)

    package_yaml = os.path.join(pair_dir, "package.yaml")
    with open(package_yaml, "w", encoding="utf-8") as f:
        f.write(
            'schema_version: "1.0"\n'
            f'scenario_id: "{scenario_id}"\n'
            'scenario_version: "1"\n'
            f'entity_family: "family-{scenario_id}"\n'
        )

    # Real per-stage evidence and transcript files on disk -- TC-083's
    # reachability assertion resolves the report's printed paths against
    # these, not against a mocked layout.
    with open(os.path.join(pair_dir, "evidence", "code.json"), "w", encoding="utf-8") as f:
        f.write(json.dumps({"stage": "code", "note": f"{scenario_id} evidence"}) + "\n")
    with open(os.path.join(pair_dir, "transcripts", "code.txt"), "w", encoding="utf-8") as f:
        f.write(f"{scenario_id} transcript\n")

    lc = {
        "identity": {
            "schema_version": "1.0", "run_id": f"run-{scenario_id}", "scenario_id": scenario_id,
            "scenario_version": "1", "fixture_id": f"fixture-{scenario_id}", "fixture_digest": "c" * 64,
            "adapter_id": "fixture-adapter", "adapter_version": "1",
            "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
        },
        "entity_graph": {}, "dispatches": [{"ordinal": 1, "note": "single dispatch"}], "stages": [base_stage()],
        "workflow_policy": {}, "review_gates": [], "questions": [],
        "limits": {
            "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
            "observed_cost_usd": 1.0, "observed_wall_clock_seconds": 1.0,
            "observed_generated_tasks": 1, "first_exceeded": None,
        },
        # A terminal COMPLETED workflow regardless of scenario -- TC-083's
        # premise is a workflow that reached completion while its held-back
        # oracle failed, never a workflow that failed outright.
        "outcome": {
            "terminal": "complete", "reason": f"{scenario_id} fixture",
            "partial_evidence": False,
            "publication_eligible": i07_outcome_publication_eligible,
        },
    }
    write_json(os.path.join(pair_dir, "lifecycle.jsonl"), lc)

    ev = {
        "schema_version": "1.0", "evaluation_id": f"eval-{scenario_id}", "identity": {}, "source_artifacts": {},
        "structural": {}, "judge": {}, "execution_oracle": execution_oracle,
        "eligibility": eligibility,
        "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
        "metrics": {
            "elapsed_time": {"value": 1.0, "available": True},
            "provider_cost": {"value": 1.0, "available": True},
            "rework": {"value": 0, "available": True},
        },
    }
    write_json(os.path.join(pair_dir, "evaluation.jsonl"), ev)

    oracle_json_path = os.path.join(pair_dir, "oracle.json")
    with open(oracle_json_path, "w", encoding="utf-8") as f:
        f.write(json.dumps({"held_back": True, "observed_result": execution_oracle.get("observed_result", "not_run")}, sort_keys=True) + "\n")

    manifest = {
        "scenario_id": scenario_id, "rep": 1,
        "artifacts": {
            "package.yaml": {"source_path": package_yaml, "sha256": sha(package_yaml)},
            "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
            "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
            "oracle.json": {"source_path": oracle_json_path, "sha256": sha(oracle_json_path)},
        },
    }
    write_json(os.path.join(pair_dir, "manifest.json"), manifest)
    return pair_dir


ELIGIBLE_FALSE_ORACLE_FAIL = {
    "aggregate_eligible": False, "publication_eligible": False,
    "invalidity_reasons": [{"code": "oracle_failure", "path": "/execution_oracle", "detail": "held-back oracle failed"}],
}
# The Edge Case fixture: an inconsistent upstream record where the oracle
# failed but I-08 nonetheless asserts eligibility. This is a contract
# violation upstream; F10 must still surface the oracle-fail evidence AND
# report the inconsistency rather than trust either field silently.
ELIGIBLE_TRUE_ORACLE_FAIL = {
    "aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": [],
}
ELIGIBLE_TRUE_ORACLE_PASS = {
    "aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": [],
}

build_pair(
    "scenario-notcorrect",
    execution_oracle={"observed_result": "fail", "invalidity_reasons": []},
    eligibility=ELIGIBLE_FALSE_ORACLE_FAIL,
    # I-07's OWN outcome.publication_eligible is TRUE -- proves the
    # headline verdict is read from I-08's eligibility, never from this
    # I-07 field or from outcome.terminal alone.
    i07_outcome_publication_eligible=True,
)

build_pair(
    "scenario-defect",
    execution_oracle={"observed_result": "fail", "invalidity_reasons": []},
    eligibility=ELIGIBLE_TRUE_ORACLE_FAIL,
    i07_outcome_publication_eligible=True,
)

build_pair(
    "scenario-oracle-absent",
    execution_oracle={},
    eligibility=ELIGIBLE_TRUE_ORACLE_PASS,
    i07_outcome_publication_eligible=True,
)

build_pair(
    "scenario-oracle-passed",
    execution_oracle={"observed_result": "pass", "invalidity_reasons": []},
    eligibility=ELIGIBLE_TRUE_ORACLE_PASS,
    i07_outcome_publication_eligible=True,
)

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc083", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
write_json(os.path.join(dest_root, "batch.json"), batch)
PYEOF

# ---------------------------------------------------------------------------
# Offline diagnosis: aggregate, then report --view headline, with no
# provider/network path reachable (PATH deliberately unmodified -- this
# script and its two invoked binaries make zero subprocess calls of their
# own; nothing here shells out to claude/codex/curl/etc).
# ---------------------------------------------------------------------------
AGG_OUT="$WORKDIR/aggregate.json"
"$AGGREGATOR" --retention-root "$ROOT" >"$AGG_OUT" 2>"$WORKDIR/agg.err" || fail "aggregator exited non-zero; stderr: $(cat "$WORKDIR/agg.err")"

REPORT_OUT="$WORKDIR/headline.md"
"$REPORTER" --aggregate "$AGG_OUT" --view headline >"$REPORT_OUT" 2>"$WORKDIR/report.err" || fail "reporter exited non-zero; stderr: $(cat "$WORKDIR/report.err")"

python3 - "$ROOT" "$REPORT_OUT" <<'PYEOF'
import os
import sys

root, report_path = sys.argv[1], sys.argv[2]

with open(report_path, encoding="utf-8") as f:
    report = f.read()


def fail(msg):
    print(f"TC-083 FAIL (python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


def section_for(scenario_id):
    marker = f"### `{scenario_id}` rep 1"
    if marker not in report:
        fail(f"headline report has no section for {scenario_id!r}")
    start = report.index(marker)
    rest = report[start + len(marker):]
    next_marker = rest.find("\n### `")
    return rest if next_marker == -1 else rest[:next_marker]


# ---------------------------------------------------------------------------
# AC-006 core case: a completed workflow with a failed held-back oracle
# renders as NOT correct -- never a false pass inferred from terminal
# status alone (I-07 outcome.publication_eligible was TRUE in the fixture).
# ---------------------------------------------------------------------------
notcorrect = section_for("scenario-notcorrect")
if "VERDICT: NOT CORRECT" not in notcorrect:
    fail("scenario-notcorrect: headline does not report the workflow as NOT correct")
if "VERDICT: correct" in notcorrect:
    fail("scenario-notcorrect: headline also printed a correct verdict -- ambiguous")
if "observed_result=fail" not in notcorrect:
    fail("scenario-notcorrect: execution oracle fail result is not visible in the headline")

# Reachability: every evidence path printed is the scenario's own retention
# path plus a fixed artifact name, and each must resolve to a REAL file/dir
# on disk -- reachable by direct file-path, never by rerunning the scenario.
for suffix in ("/oracle.json", "/lifecycle.jsonl", "/evidence/", "/transcripts/"):
    marker = f"scenarios/scenario-notcorrect/1{suffix}"
    if marker not in notcorrect:
        fail(f"scenario-notcorrect: evidence path {marker!r} not printed in headline")
    resolved = os.path.join(root, marker.rstrip("/"))
    if not os.path.exists(resolved):
        fail(f"scenario-notcorrect: printed evidence path {marker!r} does not resolve to a real file/dir at {resolved}")

# ---------------------------------------------------------------------------
# Edge Case: an inconsistent upstream fixture (oracle fail, I-08 eligibility
# claims true) still surfaces the oracle-fail evidence AND is named as an
# upstream contract defect under REQ-F-008 -- never silently trusted.
# ---------------------------------------------------------------------------
defect = section_for("scenario-defect")
if "observed_result=fail" not in defect:
    fail("scenario-defect: oracle-fail evidence must still surface even with an inconsistent eligibility fixture")
if "UPSTREAM CONTRACT DEFECT (REQ-F-008)" not in defect:
    fail("scenario-defect: inconsistent oracle-fail/eligibility-true fixture was not reported as an upstream contract defect")
if "VERDICT: correct" not in defect:
    fail("scenario-defect: this fixture's I-08 eligibility.publication_eligible is true, so the printed verdict must still read the field verbatim (correct), alongside the defect notice")

# ---------------------------------------------------------------------------
# Negative Case: a missing execution_oracle result must render distinctly
# from an oracle that ran and passed -- never collapsed into one rendering.
# ---------------------------------------------------------------------------
absent = section_for("scenario-oracle-absent")
passed = section_for("scenario-oracle-passed")
if "no observed_result present in this record" not in absent:
    fail("scenario-oracle-absent: missing execution_oracle result did not render its own distinct sentence")
if "observed_result=pass" not in passed:
    fail("scenario-oracle-passed: passed oracle result not rendered")
if "no observed_result present in this record" in passed:
    fail("scenario-oracle-passed must not render the absent-oracle sentence")
if "observed_result=pass" in absent:
    fail("scenario-oracle-absent must not render a passed-oracle result it never had")

print(
    "TC-083: a completed workflow with a failed held-back oracle renders NOT "
    "correct from I-08's own verbatim eligibility (never inferred from I-07 "
    "outcome.terminal alone); every evidence path printed resolves to a real "
    "file/dir under the retention root without rerunning the scenario; an "
    "inconsistent upstream fixture still surfaces the oracle-fail evidence "
    "and is named an upstream contract defect (REQ-F-008); a missing oracle "
    "result renders distinctly from a passed one"
)
PYEOF

echo "TC-083: PASS"
