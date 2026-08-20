#!/usr/bin/env bash
# TC-084 / T-E40-F10-008 (aggregator half) + T-E40-F10-012 (report half):
# stage-category / interval-category time reconciliation and the REQ-F-011
# six-share partition (spec.md REQ-F-009, REQ-F-010, REQ-F-011, ADR-F10-07;
# test-plan.md TC-084 full body; AC-007, AC-T1, AC-T2).
#
# Exercises the REAL aggregate-lifecycle.sh and report-lifecycle.sh binaries
# over a hand-built retention root covering the full 8 stage_category x 6
# interval_category matrix (TC-084 Preconditions: "a full 8x6 fixture
# matrix, not a handful of representative cells"), including two `code`
# stages -- one `rework: true`, one `rework: false` -- so the REQ-F-011
# rework/first_pass_code split is exercised on real data, never a stub.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
REPORTER="$SCRIPTS_DIR/report-lifecycle.sh"

fail() {
	echo "TC-084 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
[[ -x "$REPORTER" ]] || fail "bench/scripts/report-lifecycle.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# build_root <dest_root> <rework_metric_value_or_auto> <rework_flag_missing:0|1>
#
# Builds one retention root with ONE scenario ("scenario-tc084", rep 1)
# whose I-07 stages[] cover the full 8x6 (stage_category, interval_category)
# matrix -- 8 stage categories, plus a second `code` stage so both
# `rework: true` and `rework: false` are exercised -- each stage carrying
# all 6 interval_category entries with fixed, hand-computed durations so
# every partition total below is independently verifiable by arithmetic,
# not just "the script agrees with itself".
#
# Per-stage interval durations (seconds), identical across all 9 stages:
#   provider_active=10 tool_and_test=5 queue_or_claim_wait=3
#   replay_or_human_gate_wait=2 retry_or_backoff=1 unclassified=4
#   => elapsed_seconds=25 per stage, cost_usd=10 per stage.
# 9 stages * 25s = 225s total lifecycle wall time (set as
# metrics.elapsed_time.value verbatim, so stage/interval/share TIME
# partitions all reconcile to a computed, independently-checkable total
# with an EXACT ZERO unattributed residual -- TC-084's Edge Case: "the
# unattributed line is printed even when it is zero" without needing a
# schema-invalid cell to force a nonzero value).
# 9 stages * $10 = $90 stage cost; metrics.provider_cost.value is set to
# $95 (a $5 judge cost on top of stage costs, mirroring
# evaluate-lifecycle.sh's own derive_metrics(), which folds judge cost_usd
# into provider_cost) so the COST partition instead demonstrates a REAL,
# non-zero, well-understood unattributed residual: judge cost is not
# attributable to any (stage_category, interval_category) cell.
# ===========================================================================
build_root() {
	local dest_root="$1" rework_metric_override="$2" rework_flag_missing="$3"
	python3 - "$dest_root" "$rework_metric_override" "$rework_flag_missing" <<'PYEOF'
import json
import os
import sys

dest_root, rework_override, rework_missing = sys.argv[1:4]
rework_missing = rework_missing == "1"

SCENARIO_ID = "scenario-tc084"
REP = "1"
pair_dir = os.path.join(dest_root, "scenarios", SCENARIO_ID, REP)
os.makedirs(pair_dir, exist_ok=True)

with open(os.path.join(pair_dir, "package.yaml"), "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{SCENARIO_ID}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-tc084"\n'
    )

INTERVALS = [
    ("provider_active", 10),
    ("tool_and_test", 5),
    ("queue_or_claim_wait", 3),
    ("replay_or_human_gate_wait", 2),
    ("retry_or_backoff", 1),
    ("unclassified", 4),
]
STAGE_SECONDS = sum(d for _, d in INTERVALS)  # 25
STAGE_COST = 10.0


def make_intervals():
    out = []
    cursor = 0
    for category, duration in INTERVALS:
        out.append({"category": category, "start": cursor, "end": cursor + duration})
        cursor += duration
    return out


def make_stage(ordinal, category, rework, drop_rework=False):
    stage = {
        "dispatch_ordinal": ordinal,
        "stage": category,
        "category": category,
        "snapshot_digest": "a" * 64,
        "prompt_digest": "b" * 64,
        "input_lineage": [],
        "replay_lineage": [],
        "output_paths": [],
        "output_digests": [],
        "usage": {"provider": "fixture", "model": "fixture-model"},
        "cost_usd": STAGE_COST,
        "elapsed_seconds": float(STAGE_SECONDS),
        "errors": [],
        "intervals": make_intervals(),
        "candidate": {},
        "artifacts": [],
        "access_events": [],
        "evidence_refs": {},
    }
    if not drop_rework:
        stage["rework"] = rework
    return stage


STAGE_CATEGORIES = [
    "discovery", "specification", "planning", "review", "qa", "uat", "shipping",
]
stages = []
ordinal = 1
for category in STAGE_CATEGORIES:
    stages.append(make_stage(ordinal, category, rework=False))
    ordinal += 1
# Two `code` stages: one rework=false (first_pass_code), one rework=true
# (rework share). The rework=true stage is the one whose `rework` field
# is dropped entirely for the missing-flag negative case.
stages.append(make_stage(ordinal, "code", rework=False))
ordinal += 1
stages.append(make_stage(ordinal, "code", rework=True, drop_rework=rework_missing))
ordinal += 1

total_elapsed = float(STAGE_SECONDS * len(stages))  # 225.0
total_stage_cost = STAGE_COST * len(stages)  # 90.0
judge_cost = 5.0
total_provider_cost = total_stage_cost + judge_cost  # 95.0

lc = {
    "identity": {
        "schema_version": "1.0", "run_id": "run-tc084", "scenario_id": SCENARIO_ID,
        "scenario_version": "1", "fixture_id": "fixture-tc084",
        "fixture_digest": "c" * 64, "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": stages,
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {
        "max_cost_usd": 100.0, "max_wall_clock_seconds": 3600, "max_generated_tasks": 20,
        "observed_cost_usd": total_provider_cost, "observed_wall_clock_seconds": total_elapsed,
        "observed_generated_tasks": 1, "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "tc084 fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(os.path.join(pair_dir, "lifecycle.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")

if rework_override == "auto":
    rework_metric_value = 1  # exactly one contributing rework=true code stage
else:
    rework_metric_value = int(rework_override)

ev = {
    "schema_version": "1.0", "evaluation_id": "eval-tc084", "identity": {}, "source_artifacts": {},
    "structural": {}, "judge": {}, "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
    "metrics": {
        "elapsed_time": {"value": total_elapsed, "available": True, "detail": "sum of retained stage elapsed_seconds"},
        "provider_cost": {"value": total_provider_cost, "available": True, "detail": "sum of retained stage and judge cost_usd"},
        "rework": {"value": rework_metric_value, "available": True, "detail": "count of retained stages marked rework"},
    },
}
with open(os.path.join(pair_dir, "evaluation.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")

import hashlib


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
    "phase": "lifecycle_v2", "batch_id": "batch-tc084", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(os.path.join(dest_root, "batch.json"), "w", encoding="utf-8") as f:
    json.dump(batch, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")

print(json.dumps({"total_elapsed": total_elapsed, "total_provider_cost": total_provider_cost, "judge_cost": judge_cost}))
PYEOF
}

# ===========================================================================
# (a) Happy path: full 8x6 matrix, rework metric matches. TIME partitions
# reconcile to exactly zero unattributed; COST partitions show a real,
# non-zero unattributed equal to the judge cost.
# ===========================================================================
ROOT_A="$WORKDIR/root-a"
mkdir -p "$ROOT_A"
build_root "$ROOT_A" "auto" "0" >"$WORKDIR/a.meta.json"

OUT_A="$("$AGGREGATOR" --retention-root "$ROOT_A" 2>"$WORKDIR/a.err")" || fail "(a) happy path: aggregator exited non-zero; stderr: $(cat "$WORKDIR/a.err")"
[[ ! -s "$WORKDIR/a.err" ]] || fail "(a) happy path: expected no stderr diagnostics, got: $(cat "$WORKDIR/a.err")"

python3 - "$OUT_A" "$WORKDIR/a.meta.json" <<'PYEOF'
import json
import sys

out_raw, meta_path = sys.argv[1:3]
agg = json.loads(out_raw)
with open(meta_path, encoding="utf-8") as f:
    meta = json.load(f)

STAGE_CATEGORIES = ["discovery", "specification", "planning", "code", "review", "qa", "uat", "shipping"]
INTERVAL_CATEGORIES = [
    "provider_active", "tool_and_test", "queue_or_claim_wait",
    "replay_or_human_gate_wait", "retry_or_backoff", "unclassified",
]
SHARE_CELLS = ["pre_code", "review", "rework", "first_pass_code", "wait", "shipping"]


def fail(msg):
    print(f"TC-084 FAIL (python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


time_block = agg["time"]
total = time_block["lifecycle_wall_seconds"]
if total != meta["total_elapsed"]:
    fail(f"lifecycle_wall_seconds {total} != expected {meta['total_elapsed']}")

for partition_name, categories, expected_unattributed in (
    ("stage_category", STAGE_CATEGORIES, 0.0),
    ("interval_category", INTERVAL_CATEGORIES, 0.0),
    ("share_partition", SHARE_CELLS, 0.0),
):
    partition = time_block[partition_name]
    for cat in categories:
        if cat not in partition:
            fail(f"time.{partition_name} missing required cell {cat!r}")
    if "unattributed" not in partition:
        fail(f"time.{partition_name} missing the required 'unattributed' line")
    matched = sum(partition[cat] for cat in categories)
    reconciled = round(matched + partition["unattributed"], 6)
    if reconciled != total:
        fail(f"time.{partition_name} does not reconcile to lifecycle wall time: {matched} + {partition['unattributed']} = {reconciled} != {total}")
    if partition["unattributed"] != expected_unattributed:
        fail(f"time.{partition_name}.unattributed = {partition['unattributed']}, expected {expected_unattributed} (AC-007: printed even at zero)")

# REQ-F-011 exact numeric shares, independently hand-computed (see this
# script's header comment): 9 stages, 25s each, 6s of every stage's time is
# "wait" (queue_or_claim_wait+replay_or_human_gate_wait+retry_or_backoff =
# 3+2+1), 19s is non-wait.
share = time_block["share_partition"]
if share["wait"] != 9 * 6:
    fail(f"share.wait = {share['wait']}, expected {9*6}")
if share["pre_code"] != 3 * 19:
    fail(f"share.pre_code = {share['pre_code']}, expected {3*19}")
if share["review"] != 3 * 19:
    fail(f"share.review = {share['review']}, expected {3*19}")
if share["shipping"] != 1 * 19:
    fail(f"share.shipping = {share['shipping']}, expected {1*19}")
if share["rework"] != 1 * 19:
    fail(f"share.rework = {share['rework']}, expected {1*19} (only the rework=true code stage's non-wait cells)")
if share["first_pass_code"] != 1 * 19:
    fail(f"share.first_pass_code = {share['first_pass_code']}, expected {1*19} (only the rework=false code stage's non-wait cells)")

stage_cat = time_block["stage_category"]
if stage_cat["code"] != 2 * 25:
    fail(f"time.stage_category.code = {stage_cat['code']}, expected {2*25} (two code stages)")
for cat in ("discovery", "specification", "planning", "review", "qa", "uat", "shipping"):
    if stage_cat[cat] != 25:
        fail(f"time.stage_category.{cat} = {stage_cat[cat]}, expected 25")

interval_cat = time_block["interval_category"]
expected_interval = {
    "provider_active": 9 * 10, "tool_and_test": 9 * 5, "queue_or_claim_wait": 9 * 3,
    "replay_or_human_gate_wait": 9 * 2, "retry_or_backoff": 9 * 1, "unclassified": 9 * 4,
}
for cat, expected in expected_interval.items():
    if interval_cat[cat] != expected:
        fail(f"time.interval_category.{cat} = {interval_cat[cat]}, expected {expected}")

# COST block: same structural shape, but a REAL non-zero unattributed
# (the $5 judge cost, unattributable to any cell -- design decision 2/3 in
# aggregate-lifecycle.sh's own header comment).
cost_block = agg["cost"]
for partition_name, categories in (
    ("stage_category", STAGE_CATEGORIES),
    ("interval_category", INTERVAL_CATEGORIES),
    ("share_partition", SHARE_CELLS),
):
    partition = cost_block[partition_name]
    for cat in categories:
        if cat not in partition:
            fail(f"cost.{partition_name} missing required cell {cat!r}")
    if "unattributed" not in partition:
        fail(f"cost.{partition_name} missing the required 'unattributed' line")
    matched = sum(partition[cat] for cat in categories)
    reconciled = round(matched + partition["unattributed"], 6)
    if reconciled != meta["total_provider_cost"]:
        fail(f"cost.{partition_name} does not reconcile to total provider cost: {reconciled} != {meta['total_provider_cost']}")
    if partition["unattributed"] != meta["judge_cost"]:
        fail(f"cost.{partition_name}.unattributed = {partition['unattributed']}, expected the judge cost {meta['judge_cost']}")

# Cost's wait share must be exactly $0 (design decision 3: cost is never
# attributed to a wait interval).
if cost_block["share_partition"]["wait"] != 0.0:
    fail(f"cost.share_partition.wait = {cost_block['share_partition']['wait']}, expected 0.0 (cost never accrues during wait)")

print("TC-084(a): happy path -- full 8x6 matrix reconciles for time (zero unattributed) and cost (judge-cost unattributed), REQ-F-011 shares numerically exact")
PYEOF

# ===========================================================================
# (b) Negative Case: rework metric mismatch is reported as an upstream
# contract defect (REQ-F-008), not silently resolved -- and does NOT abort
# the aggregation (a data-quality diagnostic, not a refusal-vocabulary
# reason; the schema's refusal_reason list intentionally has no
# rework-mismatch entry).
# ===========================================================================
ROOT_B="$WORKDIR/root-b"
mkdir -p "$ROOT_B"
build_root "$ROOT_B" "99" "0" >/dev/null

set +e
OUT_B="$("$AGGREGATOR" --retention-root "$ROOT_B" 2>"$WORKDIR/b.err")"
RC_B=$?
set -e
[[ "$RC_B" -eq 0 ]] || fail "(b) rework mismatch: expected the aggregation to still succeed (exit 0), got $RC_B; stderr: $(cat "$WORKDIR/b.err")"
grep -q "UPSTREAM CONTRACT DEFECT" "$WORKDIR/b.err" || fail "(b) rework mismatch: stderr did not name an upstream contract defect: $(cat "$WORKDIR/b.err")"
grep -q "rework stage count" "$WORKDIR/b.err" || fail "(b) rework mismatch: stderr did not name the rework stage count reconciliation: $(cat "$WORKDIR/b.err")"
grep -q "REQ-F-008" "$WORKDIR/b.err" || fail "(b) rework mismatch: stderr did not cite REQ-F-008"
echo "$OUT_B" | python3 -c "import json,sys; json.loads(sys.stdin.read())" >/dev/null || fail "(b) rework mismatch: stdout is not valid JSON despite the non-fatal diagnostic"
echo "TC-084(b): rework/I-08 metrics.rework mismatch reported as an upstream contract defect (REQ-F-008), aggregation still completes"

# ===========================================================================
# (c) Negative Case: a `code` stage with the `rework` flag entirely absent
# (schema violation per i07-schema.yaml's required /stages[]/rework) must
# fail loudly -- never silently default to false (ADR-F10-08).
# ===========================================================================
ROOT_C="$WORKDIR/root-c"
mkdir -p "$ROOT_C"
build_root "$ROOT_C" "auto" "1" >/dev/null

set +e
OUT_C="$("$AGGREGATOR" --retention-root "$ROOT_C" 2>"$WORKDIR/c.err")"
RC_C=$?
set -e
[[ "$RC_C" -ne 0 ]] || fail "(c) missing rework flag: expected non-zero exit, got 0"
[[ -z "$OUT_C" ]] || fail "(c) missing rework flag: expected empty stdout on failure, got: $OUT_C"
grep -q "rework" "$WORKDIR/c.err" || fail "(c) missing rework flag: stderr did not mention the missing rework field: $(cat "$WORKDIR/c.err")"
echo "TC-084(c): a code stage with the rework flag entirely absent fails loudly, never silently defaulted"

# ===========================================================================
# (d) Report half / T-E40-F10-012: report-lifecycle.sh --view stage_diagnostic
# (spec.md REQ-F-009, REQ-F-010, ADR-F10-07; AC-T1, AC-T2). Reuses ROOT_A's
# aggregate.json -- the exact same full 8x6 fixture already reconciled by
# the aggregator-half checks above -- so the report half is checked against
# numbers already independently verified, never a second fixture.
# ===========================================================================
AGG_A="$WORKDIR/aggregate-a.json"
printf '%s' "$OUT_A" >"$AGG_A"

HEADLINE_A="$WORKDIR/headline-a.md"
"$REPORTER" --aggregate "$AGG_A" --view headline >"$HEADLINE_A" 2>"$WORKDIR/headline-a.err" ||
	fail "(d) headline view over ROOT_A: reporter exited non-zero; stderr: $(cat "$WORKDIR/headline-a.err")"

STAGE_DIAG_A="$WORKDIR/stage-diagnostic-a.md"
"$REPORTER" --aggregate "$AGG_A" --view stage_diagnostic >"$STAGE_DIAG_A" 2>"$WORKDIR/stage-diagnostic-a.err" ||
	fail "(d) stage_diagnostic view over ROOT_A: reporter exited non-zero; stderr: $(cat "$WORKDIR/stage-diagnostic-a.err")"

python3 - "$HEADLINE_A" "$STAGE_DIAG_A" <<'PYEOF'
import sys

headline_path, stage_path = sys.argv[1:3]

with open(headline_path, encoding="utf-8") as f:
    headline = f.read()
with open(stage_path, encoding="utf-8") as f:
    stage = f.read()


def fail(msg):
    print(f"TC-084 FAIL (python check, report half): {msg}", file=sys.stderr)
    raise SystemExit(1)


# AC-T1: the stage_diagnostic view carries the SAME headline verdict text in
# its own header (ADR-F10-07 "structurally subordinate"), never re-derived.
if "VERDICT: correct" not in headline:
    fail("headline view: expected scenario-tc084 to render VERDICT: correct (fixture eligibility is publication_eligible=True)")
if "VERDICT: correct" not in stage:
    fail("stage_diagnostic view: did not carry the headline VERDICT: correct in its own header")
if "carried from the headline view" not in stage:
    fail("stage_diagnostic view: verdict line does not say it was carried from the headline view, not re-derived")

# The verdict must appear within the document's HEADER block -- before the
# first "##" section heading -- so it is genuinely "in its header"
# (AC-T1), not buried in a body section.
title_idx = stage.index("# Shark Bench Lifecycle Stage Diagnostic Report")
first_section_idx = stage.index("\n## ")
verdict_idx = stage.index("VERDICT: correct")
if not (title_idx < verdict_idx < first_section_idx):
    fail("stage_diagnostic view: verdict is not printed within the document header block (before the first ## section)")

# AC-T2: both stage-category and interval-category partitions print an
# explicit `unattributed` line even at zero. This fixture's TIME partitions
# reconcile to EXACTLY zero unattributed (see this file's build_root header
# comment), so this is the true zero-value assertion AC-T2 requires.
unattributed_zero_count = stage.count("`unattributed`: 0")
if unattributed_zero_count < 2:
    fail(
        "stage_diagnostic view: expected at least 2 '`unattributed`: 0' lines "
        f"(time stage-category + interval-category), got {unattributed_zero_count}\n---\n{stage}"
    )

# stage_diagnostic is explicitly scoped to stage-category/interval-category
# only (task Goal) -- it must NOT duplicate the headline view's own
# REQ-F-011 share_partition block.
if "REQ-F-011 share partition" in stage:
    fail("stage_diagnostic view: must not duplicate the headline view's REQ-F-011 share partition block")

# Every stage_category/interval_category value already verified by the
# aggregator-half check above must appear verbatim in the stage_diagnostic
# report too -- same numbers, no new computation (pure formatting).
for needle in ("`code`: 50", "`discovery`: 25", "`provider_active`: 90", "`unclassified`: 36"):
    if needle not in stage:
        fail(f"stage_diagnostic view: expected time partition value {needle!r} not found\n---\n{stage}")

print("TC-084(d): stage_diagnostic view carries the headline verdict in its own header and prints stage/interval time-cost partitions with explicit unattributed lines")
PYEOF
echo "TC-084(d): report half (stage_diagnostic view happy path) PASS"

# ===========================================================================
# (e) AC-T1 negative case: the headline eligibility verdict is UNAVAILABLE
# (I-08 eligibility.publication_eligible is entirely ABSENT from the
# scenario, not merely false) -- stage_diagnostic MUST refuse to render,
# printing nothing to stdout. The headline view is unaffected by the same
# input, proving the extra gate is stage_diagnostic-specific.
# ===========================================================================
AGG_NO_VERDICT="$WORKDIR/aggregate-a-no-verdict.json"
python3 - "$AGG_A" "$AGG_NO_VERDICT" <<'PYEOF'
import json
import sys

src, dst = sys.argv[1:3]
with open(src, encoding="utf-8") as f:
    agg = json.load(f)
for s in agg["scenarios"]:
    del s["eligibility"]["publication_eligible"]
with open(dst, "w", encoding="utf-8") as f:
    json.dump(agg, f, sort_keys=True, separators=(",", ":"))
PYEOF

set +e
STAGE_DIAG_NO_VERDICT_OUT="$("$REPORTER" --aggregate "$AGG_NO_VERDICT" --view stage_diagnostic 2>"$WORKDIR/no-verdict.err")"
RC_NO_VERDICT=$?
set -e
[[ "$RC_NO_VERDICT" -ne 0 ]] || fail "(e) stage_diagnostic must refuse when the headline eligibility verdict is unavailable, got exit 0"
[[ -z "$STAGE_DIAG_NO_VERDICT_OUT" ]] || fail "(e) stage_diagnostic must print nothing to stdout on refusal, got: $STAGE_DIAG_NO_VERDICT_OUT"
grep -qi "eligibility" "$WORKDIR/no-verdict.err" || fail "(e) refusal reason did not mention eligibility: $(cat "$WORKDIR/no-verdict.err")"
grep -q "scenario-tc084" "$WORKDIR/no-verdict.err" || fail "(e) refusal reason did not name the affected scenario: $(cat "$WORKDIR/no-verdict.err")"

"$REPORTER" --aggregate "$AGG_NO_VERDICT" --view headline >"$WORKDIR/headline-no-verdict.md" 2>"$WORKDIR/headline-no-verdict.err" ||
	fail "(e) headline view must still render when only stage_diagnostic's own extra gate would refuse; stderr: $(cat "$WORKDIR/headline-no-verdict.err")"

echo "TC-084(e): stage_diagnostic refuses to render (empty stdout) when the headline eligibility verdict is unavailable for a scenario; headline view is unaffected"

# ===========================================================================
# (f) AC-T1 edge case: an aggregate with zero scenarios has no headline
# verdict to carry at all -- stage_diagnostic must refuse this too.
# ===========================================================================
EMPTY_AGG="$WORKDIR/aggregate-empty.json"
python3 - "$EMPTY_AGG" <<'PYEOF'
import json
import sys

dst = sys.argv[1]
agg = {
    "identity": {
        "schema_version": "1.0", "batch_id": "batch-empty", "phase": "lifecycle_v2",
        "retention_root_digest": "x" * 64, "batch_policy_digest": "y" * 64, "min_reps": 1,
    },
    "scenarios": [],
    "time": {}, "cost": {}, "quality": {}, "review_value": {}, "artifact_use": {},
    "noise_bands": [], "comparisons": [], "invalid": {},
}
with open(dst, "w", encoding="utf-8") as f:
    json.dump(agg, f, sort_keys=True, separators=(",", ":"))
PYEOF

set +e
EMPTY_OUT="$("$REPORTER" --aggregate "$EMPTY_AGG" --view stage_diagnostic 2>"$WORKDIR/empty.err")"
RC_EMPTY=$?
set -e
[[ "$RC_EMPTY" -ne 0 ]] || fail "(f) stage_diagnostic must refuse when the aggregate has no scenarios at all (no headline verdict to carry), got exit 0"
[[ -z "$EMPTY_OUT" ]] || fail "(f) stage_diagnostic must print nothing to stdout on refusal, got: $EMPTY_OUT"
echo "TC-084(f): stage_diagnostic refuses when the aggregate has zero scenarios (nothing to carry a headline verdict for)"

echo "TC-084: stage/interval/share time reconciliation, cost partition with a real judge-cost unattributed residual, both aggregator-half negative cases, the stage_diagnostic report half (headline-verdict carry, explicit unattributed lines), and both AC-T1 refusal cases pass"
