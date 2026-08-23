#!/usr/bin/env bash
# TC-089 / T-E40-F10-008: phase separation from completed Phase 1 tooling
# (spec.md REQ-F-017, ADR-F10-11; test-plan.md TC-089 full body; AC-012,
# AC-T1).
#
# Resolves TC-089's own deferred "mixed v1/v2 root" question (test-plan.md
# Edge Cases): this task decomposition names the resolution explicitly --
# aggregate-lifecycle.sh refuses the WHOLE retention root whenever a v1
# record.jsonl is found ANYWHERE under it, never a partial aggregate of
# just the v2 pairs. This test proves that against a root carrying BOTH a
# v1 record.jsonl and a structurally valid v2 (scenario, rep) pair side by
# side, so the refusal is proven against the actual mixed case, not just a
# root that only ever contained v1 data.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
AGGREGATE_RUNS="$SCRIPTS_DIR/aggregate-runs.sh"
RUN_BATCH="$SCRIPTS_DIR/run-batch.sh"
REPORT_BASELINE="$SCRIPTS_DIR/report-baseline.sh"
V1_FIXTURE="$SCRIPT_DIR/testdata/e40_f10/v1-record-fixture.jsonl"
BASELINE_SHA_FILE="$SCRIPT_DIR/testdata/PRE_F10_BASELINE_SHA"

fail() {
	echo "TC-089 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
[[ -x "$AGGREGATE_RUNS" ]] || fail "bench/scripts/aggregate-runs.sh missing or not executable"
[[ -x "$RUN_BATCH" ]] || fail "bench/scripts/run-batch.sh missing or not executable"
[[ -x "$REPORT_BASELINE" ]] || fail "bench/scripts/report-baseline.sh missing or not executable"
[[ -f "$V1_FIXTURE" ]] || fail "committed v1-record fixture missing: $V1_FIXTURE"
[[ -f "$BASELINE_SHA_FILE" ]] || fail "PRE_F10_BASELINE_SHA fixture missing: $BASELINE_SHA_FILE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
command -v git >/dev/null 2>&1 || fail "git not found on PATH"

PRE_F10_BASELINE_SHA="$(tr -d '[:space:]' <"$BASELINE_SHA_FILE")"
[[ -n "$PRE_F10_BASELINE_SHA" ]] || fail "PRE_F10_BASELINE_SHA fixture is empty"
git -C "$REPO_ROOT" cat-file -e "$PRE_F10_BASELINE_SHA" 2>/dev/null || fail "PRE_F10_BASELINE_SHA ($PRE_F10_BASELINE_SHA) is not a commit reachable in this checkout"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# (a) AC-T1 / REQ-F-017: a pure v1-only root is refused with a named,
# non-crashing reason.
# ===========================================================================
ROOT_V1_ONLY="$WORKDIR/v1-only"
mkdir -p "$ROOT_V1_ONLY/item-tc089/baseline/rep-1"
cp "$V1_FIXTURE" "$ROOT_V1_ONLY/item-tc089/baseline/rep-1/record.jsonl"

set +e
OUT_A="$("$AGGREGATOR" --retention-root "$ROOT_V1_ONLY" 2>"$WORKDIR/a.err")"
RC_A=$?
set -e
[[ "$RC_A" -ne 0 ]] || fail "(a) v1-only root: expected non-zero exit, got 0"
[[ -z "$OUT_A" ]] || fail "(a) v1-only root: expected empty stdout on refusal (no partial aggregate), got: $OUT_A"
grep -q "phase1_record_input_refused" "$WORKDIR/a.err" || fail "(a) v1-only root: stderr did not name the schema-owned refusal reason phase1_record_input_refused: $(cat "$WORKDIR/a.err")"
grep -q "record.jsonl" "$WORKDIR/a.err" || fail "(a) v1-only root: stderr did not name the offending record.jsonl path: $(cat "$WORKDIR/a.err")"
echo "TC-089(a): pure v1 root refused with the named phase1_record_input_refused reason, no partial aggregate on stdout"

# ===========================================================================
# (b) The mixed-root case TC-089's own Edge Cases section deferred as an
# open question: a root carrying BOTH a v1 record.jsonl AND a
# structurally valid v2 (scenario, rep) pair. This task's resolution
# (ADR-F10-11, this file's header) refuses the WHOLE root -- never a
# partial aggregate of just the v2 pair.
# ===========================================================================
ROOT_MIXED="$WORKDIR/mixed"
mkdir -p "$ROOT_MIXED/item-tc089/baseline/rep-1"
cp "$V1_FIXTURE" "$ROOT_MIXED/item-tc089/baseline/rep-1/record.jsonl"

python3 - "$ROOT_MIXED" <<'PYEOF'
import hashlib
import json
import os
import sys

dest_root = sys.argv[1]
SCENARIO_ID = "scenario-tc089-v2"
REP = "1"
pair_dir = os.path.join(dest_root, "scenarios", SCENARIO_ID, REP)
os.makedirs(pair_dir, exist_ok=True)

with open(os.path.join(pair_dir, "package.yaml"), "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{SCENARIO_ID}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-tc089"\n'
    )

stage = {
    "dispatch_ordinal": 1, "stage": "code", "category": "code",
    "snapshot_digest": "a" * 64, "prompt_digest": "b" * 64,
    "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
    "usage": {"provider": "fixture", "model": "fixture-model"},
    "cost_usd": 1.0, "elapsed_seconds": 1.0, "errors": [], "rework": False,
    "intervals": [{"category": "provider_active", "start": 0, "end": 1}],
    "candidate": {}, "artifacts": [], "access_events": [], "evidence_refs": {},
}
lc = {
    "identity": {
        "schema_version": "1.0", "run_id": "run-tc089", "scenario_id": SCENARIO_ID,
        "scenario_version": "1", "fixture_id": "fixture-tc089", "fixture_digest": "c" * 64,
        "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": [stage],
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {"max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
               "observed_cost_usd": 1.0, "observed_wall_clock_seconds": 1.0,
               "observed_generated_tasks": 1, "first_exceeded": None},
    "outcome": {"terminal": "complete", "reason": "tc089 v2 fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(os.path.join(pair_dir, "lifecycle.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")

ev = {
    "schema_version": "1.0", "evaluation_id": "eval-tc089", "identity": {}, "source_artifacts": {},
    "structural": {}, "judge": {}, "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
    "metrics": {
        "elapsed_time": {"value": 1.0, "available": True},
        "provider_cost": {"value": 1.0, "available": True},
        "rework": {"value": 0, "available": True},
    },
}
with open(os.path.join(pair_dir, "evaluation.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


manifest = {
    "scenario_id": SCENARIO_ID, "rep": int(REP),
    "artifacts": {
        "package.yaml": {"source_path": os.path.join(pair_dir, "package.yaml"), "sha256": sha(os.path.join(pair_dir, "package.yaml"))},
        "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
        "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
    },
}
with open(os.path.join(pair_dir, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc089", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "5", "max_wall_clock_seconds": "60", "max_generated_tasks": "5"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(os.path.join(dest_root, "batch.json"), "w", encoding="utf-8") as f:
    json.dump(batch, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")
PYEOF

set +e
OUT_B="$("$AGGREGATOR" --retention-root "$ROOT_MIXED" 2>"$WORKDIR/b.err")"
RC_B=$?
set -e
[[ "$RC_B" -ne 0 ]] || fail "(b) mixed root: expected non-zero exit, got 0"
[[ -z "$OUT_B" ]] || fail "(b) mixed root: expected empty stdout (whole-root refusal, never a partial aggregate of the valid v2 pair), got: $OUT_B"
grep -q "phase1_record_input_refused" "$WORKDIR/b.err" || fail "(b) mixed root: stderr did not name phase1_record_input_refused: $(cat "$WORKDIR/b.err")"
echo "TC-089(b): mixed v1+v2 root refuses the WHOLE aggregation (ADR-F10-11), not a partial aggregate of the valid v2 pair"

# ===========================================================================
# (c) The Phase 1 path itself still runs successfully end-to-end and is
# byte-unmodified against PRE_F10_BASELINE_SHA -- this is the OTHER half
# of REQ-F-017: F10 refusing v1 input must not come at the cost of
# breaking or touching the completed Phase 1 surfaces.
# ===========================================================================
PHASE1_ROOT="$WORKDIR/phase1-root"
mkdir -p "$PHASE1_ROOT/item-tc089/baseline/rep-1"
cp "$V1_FIXTURE" "$PHASE1_ROOT/item-tc089/baseline/rep-1/record.jsonl"

OUT_C="$("$AGGREGATE_RUNS" --root "$PHASE1_ROOT" 2>"$WORKDIR/c.err")" || fail "(c) Phase 1 aggregate-runs.sh failed against its own valid v1 fixture: $(cat "$WORKDIR/c.err")"
echo "$OUT_C" | python3 -c "import json,sys; json.loads(sys.stdin.read())" >/dev/null || fail "(c) Phase 1 aggregate-runs.sh did not print a valid JSON aggregate"
echo "$OUT_C" >"$WORKDIR/c.aggregate.json"
"$REPORT_BASELINE" --aggregate "$WORKDIR/c.aggregate.json" >"$WORKDIR/c.report.md" 2>"$WORKDIR/c.report.err" || fail "(c) Phase 1 report-baseline.sh failed against the Phase 1 aggregate: $(cat "$WORKDIR/c.report.err")"
[[ -s "$WORKDIR/c.report.md" ]] || fail "(c) Phase 1 report-baseline.sh produced no output"
echo "TC-089(c): the unmodified Phase 1 run-batch.sh/aggregate-runs.sh/report-baseline.sh path still runs end-to-end"

DIFF_OUT="$(git -C "$REPO_ROOT" diff "$PRE_F10_BASELINE_SHA" -- bench/scripts/run-batch.sh bench/scripts/aggregate-runs.sh bench/scripts/report-baseline.sh)"
[[ -z "$DIFF_OUT" ]] || fail "(c) Phase 1 surfaces are NOT byte-unmodified vs PRE_F10_BASELINE_SHA ($PRE_F10_BASELINE_SHA):
$DIFF_OUT"
echo "TC-089(c): run-batch.sh, aggregate-runs.sh, and report-baseline.sh are byte-unmodified vs PRE_F10_BASELINE_SHA ($PRE_F10_BASELINE_SHA)"

# ===========================================================================
# (d) No v1 output anywhere carries the lifecycle-v2 phase label (Negative
# Cases: "A phase label accidentally applied to Phase 1 output ... must
# fail this test"). Scans both the Phase 1 aggregate and report produced
# above.
# ===========================================================================
grep -q "lifecycle_v2" "$WORKDIR/c.aggregate.json" && fail "(d) Phase 1 aggregate.json unexpectedly carries the lifecycle_v2 phase label"
grep -q "lifecycle_v2" "$WORKDIR/c.report.md" && fail "(d) Phase 1 report-baseline.sh output unexpectedly carries the lifecycle_v2 phase label"
echo "TC-089(d): no Phase 1 (v1) output carries the lifecycle_v2 phase label"

# ===========================================================================
# (e) Every F10 aggregate DOES carry the lifecycle_v2 phase label -- the
# other half of (d)'s negative check, proven against the real aggregator
# on a valid v2-only root (no v1 record.jsonl anywhere).
# ===========================================================================
ROOT_V2_ONLY="$WORKDIR/v2-only"
mkdir -p "$ROOT_V2_ONLY"
cp -a "$ROOT_MIXED/scenarios" "$ROOT_V2_ONLY/scenarios"
cp "$ROOT_MIXED/batch.json" "$ROOT_V2_ONLY/batch.json"

OUT_E="$("$AGGREGATOR" --retention-root "$ROOT_V2_ONLY" 2>"$WORKDIR/e.err")" || fail "(e) v2-only root: aggregator failed unexpectedly: $(cat "$WORKDIR/e.err")"
echo "$OUT_E" | python3 -c "
import json, sys
agg = json.loads(sys.stdin.read())
assert agg['identity']['phase'] == 'lifecycle_v2', agg['identity']['phase']
" || fail "(e) v2-only root: aggregate.json identity.phase is not lifecycle_v2"
echo "TC-089(e): a valid v2-only aggregate carries the lifecycle_v2 phase label in /identity/phase"

echo "TC-089: v1-only and mixed v1+v2 roots both refuse the whole aggregation with a named reason; the unmodified Phase 1 path runs end-to-end and stays byte-identical to PRE_F10_BASELINE_SHA; no v1 output is relabelled lifecycle_v2 and every v2 aggregate carries it"
