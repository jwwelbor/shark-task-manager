#!/usr/bin/env bash
# TC-085 / T-E40-F10-009 (aggregator half): review-value reporting per gate
# (spec.md REQ-F-008, REQ-F-012, REQ-F-015, ADR-F10-04, ADR-F10-10; test-plan.md
# TC-085 full body; AC-008, AC-T1, AC-T2).
#
# Exercises the REAL aggregate-lifecycle.sh binary over a hand-built
# retention root with two pairs (TC-085 Preconditions): pair A carries a
# `findings`-state gate with mixed severity/defect class, a duplicate, a
# recurrent-eligible bucket, confirmed/unconfirmed findings, a downstream
# escape, a second-round `findings` gate, and a seeded truth set with
# numeric precision/recall; pair B carries a `zero_findings` gate, a
# `collection_failure` gate, a `not_reached` gate, and NO truth set. This
# is the aggregator half only -- report-lifecycle.sh --view headline
# rendering is T-E40-F10-011's slice.
#
# T-E40-F10-009 round 2 (feature-level tech-lead code-review kickback,
# REQ-F-008): per-gate elapsed_seconds/provider_cost_usd/resolution_cost_usd
# have no upstream join key in I-07/I-08 AT ALL -- a permanent contract gap,
# not a per-run absence -- so they MUST render a distinct, named
# upstream-contract-defect reason, never the bare "unavailable" string the
# legitimate no-truth-set case (precision/recall) uses.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
REPORTER="$SCRIPTS_DIR/report-lifecycle.sh"

fail() {
	echo "TC-085 FAIL: $1" >&2
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


def measure(gate, severity, defect_class, emitted, unique, duplicate, recurrent, confirmed, unconfirmed, downstream_escape):
    def m(v):
        return {"value": v, "available": True}
    return {
        "gate": gate, "severity": severity, "defect_class": defect_class,
        "measures": {
            "emitted": m(emitted), "normalized_unique": m(unique), "duplicate": m(duplicate),
            "recurrent": m(recurrent), "confirmed": m(confirmed), "unconfirmed": m(unconfirmed),
            "downstream_escape": m(downstream_escape),
        },
    }


def build_pair(scenario_id, review_gates, quality_metrics):
    pair_dir = os.path.join(dest_root, "scenarios", scenario_id, "1")
    write_json_yaml = os.path.join(pair_dir, "package.yaml")
    os.makedirs(pair_dir, exist_ok=True)
    with open(write_json_yaml, "w", encoding="utf-8") as f:
        f.write(
            'schema_version: "1.0"\n'
            f'scenario_id: "{scenario_id}"\n'
            'scenario_version: "1"\n'
            f'entity_family: "family-{scenario_id}"\n'
        )
    lc = {
        "identity": {
            "schema_version": "1.0", "run_id": f"run-{scenario_id}", "scenario_id": scenario_id,
            "scenario_version": "1", "fixture_id": f"fixture-{scenario_id}", "fixture_digest": "c" * 64,
            "adapter_id": "fixture-adapter", "adapter_version": "1",
            "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
        },
        "entity_graph": {}, "dispatches": [], "stages": [base_stage()],
        "workflow_policy": {}, "review_gates": review_gates, "questions": [],
        "limits": {
            "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
            "observed_cost_usd": 1.0, "observed_wall_clock_seconds": 1.0,
            "observed_generated_tasks": 1, "first_exceeded": None,
        },
        "outcome": {"terminal": "complete", "reason": f"{scenario_id} fixture", "partial_evidence": False, "publication_eligible": True},
    }
    write_json(os.path.join(pair_dir, "lifecycle.jsonl"), lc)

    ev = {
        "schema_version": "1.0", "evaluation_id": f"eval-{scenario_id}", "identity": {}, "source_artifacts": {},
        "structural": {}, "judge": {}, "execution_oracle": {},
        "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
        "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
        "metrics": {
            "elapsed_time": {"value": 1.0, "available": True},
            "provider_cost": {"value": 1.0, "available": True},
            "rework": {"value": 0, "available": True},
            "quality": quality_metrics,
        },
    }
    write_json(os.path.join(pair_dir, "evaluation.jsonl"), ev)

    manifest = {
        "scenario_id": scenario_id, "rep": 1,
        "artifacts": {
            "package.yaml": {"source_path": write_json_yaml, "sha256": sha(write_json_yaml)},
            "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
            "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
        },
    }
    write_json(os.path.join(pair_dir, "manifest.json"), manifest)


# ---------------------------------------------------------------------------
# Pair A: seeded truth set, one round-1 `findings` gate ("code_review") with
# three severity/defect-class buckets, one round-2 `findings` gate
# ("second_pass_review") -- proves first_pass_yield's denominator/numerator.
# ---------------------------------------------------------------------------
pair_a_gates = [
    {"gate_id": "code_review", "state": "findings", "round": 1, "candidate_ref": "candidate-a", "policy_ref": "policy-a", "findings": [{}, {}, {}, {}]},
    {"gate_id": "second_pass_review", "state": "findings", "round": 2, "candidate_ref": "candidate-a2", "policy_ref": "policy-a", "findings": [{}]},
]
pair_a_measures = [
    measure("code_review", "high", "correctness", emitted=2, unique=2, duplicate=0, recurrent=0, confirmed=1, unconfirmed=1, downstream_escape=1),
    measure("code_review", "low", "style", emitted=1, unique=0, duplicate=1, recurrent=0, confirmed=0, unconfirmed=0, downstream_escape=0),
    measure("code_review", "high", "security", emitted=1, unique=1, duplicate=0, recurrent=1, confirmed=0, unconfirmed=1, downstream_escape=0),
    measure("second_pass_review", "medium", "perf", emitted=1, unique=1, duplicate=0, recurrent=0, confirmed=1, unconfirmed=0, downstream_escape=0),
]
pair_a_quality = {
    "confirmed_findings": {"value": 2, "available": True},
    "unconfirmed_findings": {"value": 2, "available": True},
    "precision": {"value": 0.5, "available": True},
    "recall": {"value": 0.4, "available": True},
    "truth_set_status": "available",
    "review_measures": pair_a_measures,
    "aggregate_eligible": {"value": None, "available": False, "detail": "eligibility is reported separately"},
}
build_pair("scenario-tc085a", pair_a_gates, pair_a_quality)

# ---------------------------------------------------------------------------
# Pair B: no truth set, one `zero_findings` gate, one `collection_failure`
# gate, one `not_reached` gate (excluded from first_pass_yield).
# ---------------------------------------------------------------------------
pair_b_gates = [
    {"gate_id": "qa_zero", "state": "zero_findings", "round": 1, "candidate_ref": "candidate-b", "policy_ref": "policy-b", "findings": []},
    {"gate_id": "qa_failed", "state": "collection_failure", "round": 1, "candidate_ref": "candidate-b", "policy_ref": "policy-b", "findings": [], "collector_error": "timeout"},
    {"gate_id": "uat_gate", "state": "not_reached", "round": None, "candidate_ref": None, "policy_ref": "policy-b", "findings": []},
]
pair_b_quality = {
    "confirmed_findings": {"value": 0, "available": True},
    "unconfirmed_findings": {"value": 0, "available": True},
    "precision": {"value": None, "available": False},
    "recall": {"value": None, "available": False},
    "truth_set_status": "truth-set-unavailable",
    "review_measures": [],
    "aggregate_eligible": {"value": None, "available": False, "detail": "eligibility is reported separately"},
}
build_pair("scenario-tc085b", pair_b_gates, pair_b_quality)

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc085", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
write_json(os.path.join(dest_root, "batch.json"), batch)
PYEOF

AGG_OUT="$WORKDIR/aggregate.json"
"$AGGREGATOR" --retention-root "$ROOT" >"$AGG_OUT" 2>"$WORKDIR/err.log" || fail "aggregator exited non-zero; stderr: $(cat "$WORKDIR/err.log")"
OUT="$(cat "$AGG_OUT")"
LIFECYCLE_SCHEMA="$SCRIPTS_DIR/../reports/lifecycle-baseline-schema.yaml"
[[ -f "$LIFECYCLE_SCHEMA" ]] || fail "bench/reports/lifecycle-baseline-schema.yaml missing"

python3 - "$OUT" "$LIFECYCLE_SCHEMA" <<'PYEOF'
import json
import sys

import yaml

agg = json.loads(sys.argv[1])
with open(sys.argv[2], encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}


def fail(msg):
    print(f"TC-085 FAIL (python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


gates = {g["gate_id"]: g for g in agg["review_value"]["gates"]}
expected_ids = {"code_review", "second_pass_review", "qa_zero", "qa_failed", "uat_gate"}
if set(gates) != expected_ids:
    fail(f"gate_id set = {sorted(gates)}, expected {sorted(expected_ids)}")

# --- AC-T2: zero_findings vs collection_failure render as distinct states,
# never collapsed, and never inferred as "zero findings" for a failed gate.
if gates["qa_zero"]["state"] != "zero_findings":
    fail(f"qa_zero.state = {gates['qa_zero']['state']!r}, expected 'zero_findings'")
if gates["qa_failed"]["state"] != "collection_failure":
    fail(f"qa_failed.state = {gates['qa_failed']['state']!r}, expected 'collection_failure'")
if gates["qa_zero"]["state"] == gates["qa_failed"]["state"]:
    fail("zero_findings and collection_failure gates rendered with the same state -- AC-T2 violated")
if gates["uat_gate"]["state"] != "not_reached":
    fail(f"uat_gate.state = {gates['uat_gate']['state']!r}, expected 'not_reached'")

zero_counts = {"emitted": 0, "normalized_unique": 0, "duplicate": 0, "recurrent": 0, "confirmed": 0, "unconfirmed": 0, "downstream_escape": 0}
for gate_id in ("qa_zero", "qa_failed", "uat_gate"):
    if gates[gate_id]["counts"] != zero_counts:
        fail(f"{gate_id}.counts = {gates[gate_id]['counts']}, expected all-zero {zero_counts}")

# --- AC-T1: no seeded truth set -> precision/recall render literally
# "unavailable", never 0, an empty value, or an inferred substitute.
for gate_id in ("qa_zero", "qa_failed", "uat_gate"):
    g = gates[gate_id]
    if g["truth_set_available"] is not False:
        fail(f"{gate_id}.truth_set_available = {g['truth_set_available']!r}, expected False")
    if g["precision"] != "unavailable":
        fail(f"{gate_id}.precision = {g['precision']!r}, expected the literal string 'unavailable'")
    if g["recall"] != "unavailable":
        fail(f"{gate_id}.recall = {g['recall']!r}, expected the literal string 'unavailable'")
    # Edge case: zero findings AND no truth set must render BOTH facts
    # distinctly, never conflated into one ambiguous "no data" line.
    if gate_id == "qa_zero" and (g["counts"] != zero_counts or g["precision"] != "unavailable"):
        fail("qa_zero must show zero findings AND precision unavailable simultaneously, not conflated")

# --- Pair A: seeded truth set -> numeric precision/recall, never
# "unavailable", carried onto every gate that pair contributed.
for gate_id in ("code_review", "second_pass_review"):
    g = gates[gate_id]
    if g["truth_set_available"] is not True:
        fail(f"{gate_id}.truth_set_available = {g['truth_set_available']!r}, expected True")
    if g["precision"] != 0.5:
        fail(f"{gate_id}.precision = {g['precision']!r}, expected 0.5")
    if g["recall"] != 0.4:
        fail(f"{gate_id}.recall = {g['recall']!r}, expected 0.4")

# --- All seven finding measures render, broken out by severity and defect
# class, matching I-08's own measure-key names verbatim (normalized_unique,
# not "unique").
code_review = gates["code_review"]
expected_total = {"emitted": 4, "normalized_unique": 3, "duplicate": 1, "recurrent": 1, "confirmed": 1, "unconfirmed": 2, "downstream_escape": 1}
if code_review["counts"] != expected_total:
    fail(f"code_review.counts = {code_review['counts']}, expected {expected_total}")

expected_by_severity = {
    "high": {"emitted": 3, "normalized_unique": 3, "duplicate": 0, "recurrent": 1, "confirmed": 1, "unconfirmed": 2, "downstream_escape": 1},
    "low": {"emitted": 1, "normalized_unique": 0, "duplicate": 1, "recurrent": 0, "confirmed": 0, "unconfirmed": 0, "downstream_escape": 0},
}
if code_review["counts_by_severity"] != expected_by_severity:
    fail(f"code_review.counts_by_severity = {code_review['counts_by_severity']}, expected {expected_by_severity}")

expected_by_defect_class = {
    "correctness": {"emitted": 2, "normalized_unique": 2, "duplicate": 0, "recurrent": 0, "confirmed": 1, "unconfirmed": 1, "downstream_escape": 1},
    "style": {"emitted": 1, "normalized_unique": 0, "duplicate": 1, "recurrent": 0, "confirmed": 0, "unconfirmed": 0, "downstream_escape": 0},
    "security": {"emitted": 1, "normalized_unique": 1, "duplicate": 0, "recurrent": 1, "confirmed": 0, "unconfirmed": 1, "downstream_escape": 0},
}
if code_review["counts_by_defect_class"] != expected_by_defect_class:
    fail(f"code_review.counts_by_defect_class = {code_review['counts_by_defect_class']}, expected {expected_by_defect_class}")

# --- elapsed/cost per gate: I-08 carries no per-gate join key AT ALL (a
# permanent upstream contract gap, not a per-run absence like the
# no-truth-set case above) -- REQ-F-008 requires this render as a distinct,
# named upstream-contract-defect reason, never the bare "unavailable"
# string the legitimate no-truth-set case above uses, and never zero.
# REQ-F-018: read the schema-owned reason rather than duplicating the
# literal text privately in this test.
UPSTREAM_GAP_REASON = schema.get("review_value_gate_time_cost_upstream_gap_reason")
if not UPSTREAM_GAP_REASON:
    fail("schema review_value_gate_time_cost_upstream_gap_reason is empty or missing")
for gate_id in gates:
    g = gates[gate_id]
    for field in ("elapsed_seconds", "provider_cost_usd", "resolution_cost_usd"):
        if g[field] != UPSTREAM_GAP_REASON:
            fail(f"{gate_id}.{field} = {g[field]!r}, expected the distinct upstream-contract-defect reason {UPSTREAM_GAP_REASON!r}")
        if g[field] == "unavailable":
            fail(f"{gate_id}.{field} must NOT render the bare 'unavailable' string used for the legitimate no-truth-set case")

# --- quality.first_pass_yield: reached gates = code_review(r1),
# second_pass_review(r2), qa_zero(r1), qa_failed(r1) = 4; round==1 count = 3.
# not_reached is excluded from the denominator.
expected_yield = round(3 / 4, 6)
if agg["quality"]["first_pass_yield"] != expected_yield:
    fail(f"quality.first_pass_yield = {agg['quality']['first_pass_yield']!r}, expected {expected_yield}")

# --- quality.by_scenario carries verbatim structural/judge/execution_oracle
# per pair.
by_scenario = {(q["scenario_id"], q["rep"]): q for q in agg["quality"]["by_scenario"]}
if ("scenario-tc085a", 1) not in by_scenario or ("scenario-tc085b", 1) not in by_scenario:
    fail(f"quality.by_scenario missing expected pairs: {sorted(by_scenario)}")

print("TC-085: all seven finding measures render correctly broken out by severity/defect class; zero_findings and collection_failure render as distinct never-collapsed states; precision/recall render 'unavailable' (never 0) without a seeded truth set and numeric with one; per-gate elapsed/cost render the distinct upstream-contract-defect reason (never bare 'unavailable', never 0); first_pass_yield reconciles against retained gate rounds")
PYEOF

echo "TC-085 (aggregator half): PASS"

# ===========================================================================
# T-E40-F10-011 (report half): report-lifecycle.sh --view headline over the
# SAME aggregate, review_value block only (TC-085 full body, Input).
# ===========================================================================
REPORT_OUT="$WORKDIR/headline.md"
"$REPORTER" --aggregate "$AGG_OUT" --view headline >"$REPORT_OUT" 2>"$WORKDIR/report.err" || fail "reporter exited non-zero; stderr: $(cat "$WORKDIR/report.err")"

python3 - "$REPORT_OUT" "$LIFECYCLE_SCHEMA" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1], encoding="utf-8") as f:
    report = f.read()
with open(sys.argv[2], encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}


def fail(msg):
    print(f"TC-085 FAIL (report-half python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


def section_for(gate_id):
    marker = f"### Gate `{gate_id}`"
    if marker not in report:
        fail(f"headline report has no section for gate {gate_id!r}")
    start = report.index(marker)
    rest = report[start + len(marker):]
    next_marker = rest.find("\n### Gate `")
    return rest if next_marker == -1 else rest[:next_marker]


# --- AC-T2: no seeded truth set (pair B gates) -> precision/recall render
# EXACTLY "precision: unavailable" / "recall: unavailable", never 0, an
# empty value, or an inferred substitute (asserted by exact string match,
# not absence-of-output, per TC-085 Observability Evidence).
for gate_id in ("qa_zero", "qa_failed", "uat_gate"):
    section = section_for(gate_id)
    if "precision: unavailable" not in section:
        fail(f"{gate_id}: headline does not print the literal 'precision: unavailable'")
    if "recall: unavailable" not in section:
        fail(f"{gate_id}: headline does not print the literal 'recall: unavailable'")

# --- Pair A: seeded truth set -> numeric precision/recall, never
# "unavailable", rendered on every gate that pair contributed.
for gate_id in ("code_review", "second_pass_review"):
    section = section_for(gate_id)
    if "precision: 0.5" not in section:
        fail(f"{gate_id}: headline does not print numeric precision 0.5")
    if "recall: 0.4" not in section:
        fail(f"{gate_id}: headline does not print numeric recall 0.4")
    if "precision: unavailable" in section or "recall: unavailable" in section:
        fail(f"{gate_id}: headline must not print 'unavailable' for a gate with a seeded truth set")

# --- Zero-finding and collection-failure gates must render VISIBLY DISTINCT
# from each other -- never both collapsing into "zero findings" or any
# other single shared rendering (TC-085 Negative Cases: a collection
# failure must not silently render as a clean zero-finding pass).
zero_section = section_for("qa_zero")
failed_section = section_for("qa_failed")
if "reported zero findings" not in zero_section:
    fail("qa_zero: headline does not name this as a clean zero-finding pass")
if "COLLECTION FAILURE" not in failed_section:
    fail("qa_failed: headline does not flag this as a collection failure")
if "COLLECTION FAILURE" in zero_section:
    fail("qa_zero must not be flagged as a collection failure")
if "reported zero findings" in failed_section:
    fail("qa_failed must not render as a clean zero-finding pass")

# Edge case: qa_zero must show BOTH "zero findings" and "precision
# unavailable" simultaneously, not conflated into one ambiguous line.
if "reported zero findings" not in zero_section or "precision: unavailable" not in zero_section:
    fail("qa_zero must show zero findings AND precision unavailable simultaneously, not conflated")

# --- All seven finding measures, broken out by severity and defect class,
# render using I-08's own measure-key names verbatim (normalized_unique).
code_review_section = section_for("code_review")
for key in ("emitted=4", "normalized_unique=3", "duplicate=1", "recurrent=1", "confirmed=1", "unconfirmed=2", "downstream_escape=1"):
    if key not in code_review_section:
        fail(f"code_review: headline total counts missing {key!r}")
if "`high`:" not in code_review_section or "`low`:" not in code_review_section:
    fail("code_review: headline does not break counts out by severity (high/low)")
if "`correctness`:" not in code_review_section or "`style`:" not in code_review_section or "`security`:" not in code_review_section:
    fail("code_review: headline does not break counts out by defect class (correctness/style/security)")

# --- elapsed/cost per gate: I-08 carries no per-gate join key AT ALL (a
# permanent upstream contract gap) -- REQ-F-008 requires the distinct,
# named upstream-contract-defect reason, never the bare "unavailable"
# string the no-truth-set case above uses. REQ-F-018: read the schema-owned
# reason rather than duplicating the literal text privately in this test.
UPSTREAM_GAP_REASON = schema.get("review_value_gate_time_cost_upstream_gap_reason")
if not UPSTREAM_GAP_REASON:
    fail("schema review_value_gate_time_cost_upstream_gap_reason is empty or missing")
for gate_id in ("code_review", "second_pass_review", "qa_zero", "qa_failed", "uat_gate"):
    section = section_for(gate_id)
    for prefix in ("elapsed_seconds", "provider_cost_usd", "resolution_cost_usd"):
        field = f"{prefix}: {UPSTREAM_GAP_REASON}"
        if field not in section:
            fail(f"{gate_id}: headline missing {field!r}")
        if f"{prefix}: unavailable" in section:
            fail(f"{gate_id}: headline must not print the bare '{prefix}: unavailable' -- expected the distinct upstream-contract-defect reason")

print(
    "TC-085 (report half): review_value block renders all seven finding "
    "measures broken out by severity/defect class; zero_findings and "
    "collection_failure render visibly distinct; precision/recall render "
    "exactly 'unavailable' without a seeded truth set and numeric with one; "
    "per-gate elapsed/cost render the distinct upstream-contract-defect "
    "reason, never bare 'unavailable'"
)
PYEOF

echo "TC-085 (report half): PASS"
