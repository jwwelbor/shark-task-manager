#!/usr/bin/env bash
# TC-088 / T-E40-F10-010 (aggregator half): dimension separation and
# noise-band reporting (spec.md REQ-F-007, REQ-F-008, REQ-F-016,
# ADR-F10-12; test-plan.md TC-088 full body, target (i); AC-011, AC-T1,
# AC-T2).
#
# This is the AGGREGATOR half only. Report-lifecycle.sh's "no detectable
# effect" rendering and target (ii)'s report-template static scan are
# T-E40-F10-011's slice (same test filename, report half). This test proves
# the aggregator-owned mechanism report-lifecycle.sh will render from:
#   - noise_bands[] published per (scenario, metric) with the reused
#     min/median/max/spread_abs/acceptance_interval formula;
#   - insufficient_reps: true when a scenario's rep count falls below the
#     batch policy's declared /identity/min_reps (AC-T1), never a silently
#     published band;
#   - comparisons[] republishing compare-lifecycle-evaluations.sh's verdict
#     verbatim;
#   - invalid[] carrying every invalidity_reasons entry -- both
#     batch-dispatch-failure rows (invalid/index.jsonl) and retained-but-
#     ineligible rows, the latter copied verbatim from the SAME eligibility
#     object already published in scenarios[] (AC-T2: no recomputation,
#     override, or softening);
#   - target (i): a static scan of bench/reports/lifecycle-baseline-
#     schema.yaml's field/property definitions proving zero property names
#     match a forbidden_composite_fields term, excluding the schema's own
#     list declaration;
#   - no composite/weighted/blended score field anywhere in the actual
#     aggregate.json output either (REQ-F-016).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
SCHEMA="$BENCH_DIR/reports/lifecycle-baseline-schema.yaml"

fail() {
	echo "TC-088 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
[[ -f "$SCHEMA" ]] || fail "bench/reports/lifecycle-baseline-schema.yaml missing"
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


def build_pair(scenario_id, rep, elapsed, cost, eligibility, confirmed_findings="unavailable"):
    pair_dir = os.path.join(dest_root, "scenarios", scenario_id, str(rep))
    package_yaml = os.path.join(pair_dir, "package.yaml")
    os.makedirs(pair_dir, exist_ok=True)
    with open(package_yaml, "w", encoding="utf-8") as f:
        f.write(
            'schema_version: "1.0"\n'
            f'scenario_id: "{scenario_id}"\n'
            'scenario_version: "1"\n'
            f'entity_family: "family-{scenario_id}"\n'
        )
    lc = {
        "identity": {
            "schema_version": "1.0", "run_id": f"run-{scenario_id}-{rep}", "scenario_id": scenario_id,
            "scenario_version": "1", "fixture_id": f"fixture-{scenario_id}", "fixture_digest": "c" * 64,
            "adapter_id": "fixture-adapter", "adapter_version": "1",
            "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
        },
        "entity_graph": {}, "dispatches": [], "stages": [base_stage()],
        "workflow_policy": {}, "review_gates": [], "questions": [],
        "limits": {
            "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
            "observed_cost_usd": cost, "observed_wall_clock_seconds": elapsed,
            "observed_generated_tasks": 1, "first_exceeded": None,
        },
        "outcome": {"terminal": "complete", "reason": f"{scenario_id} fixture", "partial_evidence": False, "publication_eligible": True},
    }
    write_json(os.path.join(pair_dir, "lifecycle.jsonl"), lc)

    quality_metrics = {
        "confirmed_findings": (
            {"value": confirmed_findings, "available": True}
            if confirmed_findings != "unavailable"
            else {"value": None, "available": False}
        ),
        "unconfirmed_findings": {"value": 0, "available": True},
        "precision": {"value": None, "available": False},
        "recall": {"value": None, "available": False},
        "truth_set_status": "truth-set-unavailable",
        "review_measures": [],
        "aggregate_eligible": {"value": None, "available": False, "detail": "eligibility is reported separately"},
    }

    ev = {
        "schema_version": "1.0", "evaluation_id": f"eval-{scenario_id}-{rep}", "identity": {}, "source_artifacts": {},
        "structural": {}, "judge": {}, "execution_oracle": {},
        "eligibility": eligibility,
        "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
        "metrics": {
            "elapsed_time": {"value": elapsed, "available": True},
            "provider_cost": {"value": cost, "available": True},
            "rework": {"value": 0, "available": True},
            "quality": quality_metrics,
        },
    }
    write_json(os.path.join(pair_dir, "evaluation.jsonl"), ev)

    manifest = {
        "scenario_id": scenario_id, "rep": rep,
        "artifacts": {
            "package.yaml": {"source_path": package_yaml, "sha256": sha(package_yaml)},
            "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
            "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
        },
    }
    write_json(os.path.join(pair_dir, "manifest.json"), manifest)
    return pair_dir


ELIGIBLE = {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []}

# ---------------------------------------------------------------------------
# scenario-suff: two reps (>= batch min_reps=2) -- proves the normal
# noise-band mechanism: min/median/max/spread_abs/acceptance_interval and
# insufficient_reps: false.
# ---------------------------------------------------------------------------
pair_dir_1 = build_pair("scenario-suff", 1, elapsed=10.0, cost=1.0, eligibility=ELIGIBLE, confirmed_findings=2)
build_pair("scenario-suff", 2, elapsed=12.0, cost=1.2, eligibility=ELIGIBLE, confirmed_findings=4)

# rep 1 of scenario-suff also carries an OPTIONAL comparison.json --
# compare-lifecycle-evaluations.sh's own output shape, republished verbatim.
comparison_record = {
    "schema_version": "1.0",
    "mode": "independent_frozen_candidate",
    "left_evaluation_id": "eval-scenario-suff-1",
    "right_evaluation_id": "eval-candidate-x",
    "accepted": True,
    "comparison": {
        "accepted": True, "quality_delta": None, "causal_claim": None,
        "newly_confirmed_findings": [], "intervening_candidates": [],
    },
    "divergences": [],
}
write_json(os.path.join(pair_dir_1, "comparison.json"), comparison_record)

# ---------------------------------------------------------------------------
# scenario-insuff: ONE rep, below the batch's declared min_reps=2 --
# AC-T1: insufficient_reps must be true, never a silently published band.
# ---------------------------------------------------------------------------
build_pair("scenario-insuff", 1, elapsed=5.0, cost=0.5, eligibility=ELIGIBLE)

# ---------------------------------------------------------------------------
# scenario-ineligible: retained (has evaluation.jsonl, appears in
# scenarios[]) but I-08 itself flagged it ineligible -- AC-T2: this
# eligibility object must be copied VERBATIM into invalid[], no
# recomputation, override, or softening.
# ---------------------------------------------------------------------------
INELIGIBLE = {
    "aggregate_eligible": False,
    "publication_eligible": False,
    "invalidity_reasons": [{"code": "missing_oracle", "path": "/execution_oracle", "detail": "oracle did not produce a result"}],
}
build_pair("scenario-ineligible", 1, elapsed=3.0, cost=0.3, eligibility=INELIGIBLE)

# ---------------------------------------------------------------------------
# scenario-failed: a pair that NEVER reached a retained evaluation.jsonl at
# all -- run-lifecycle-batch.sh's own invalid/index.jsonl dispatch-failure
# ledger (source a), mirrored verbatim, never retained under scenarios/.
# ---------------------------------------------------------------------------
os.makedirs(os.path.join(dest_root, "invalid"), exist_ok=True)
with open(os.path.join(dest_root, "invalid", "index.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps({
        "scenario_id": "scenario-failed",
        "rep": 1,
        "retention_path": "scenarios/scenario-failed/1",
        "invalidity_reasons": ["lifecycle_run_failed"],
    }) + "\n")

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc088", "mode": "baseline", "min_reps": 2,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
write_json(os.path.join(dest_root, "batch.json"), batch)
PYEOF

OUT="$("$AGGREGATOR" --retention-root "$ROOT" 2>"$WORKDIR/err.log")" || fail "aggregator exited non-zero; stderr: $(cat "$WORKDIR/err.log")"

python3 - "$OUT" <<'PYEOF'
import json
import sys

agg = json.loads(sys.argv[1])


def fail(msg):
    print(f"TC-088 FAIL (python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


# ---------------------------------------------------------------------------
# noise_bands[]: three metrics per scenario (elapsed_time, provider_cost,
# confirmed_findings), one row per (scenario, metric).
# ---------------------------------------------------------------------------
bands = {(b["scenario_id"], b["metric"]): b for b in agg["noise_bands"]}
expected_metrics = {"elapsed_time", "provider_cost", "confirmed_findings"}
scenario_ids = {sid for sid, _ in bands}
if scenario_ids != {"scenario-suff", "scenario-insuff", "scenario-ineligible"}:
    fail(f"noise_bands scenario_id set = {sorted(scenario_ids)}, expected suff/insuff/ineligible")
for sid in scenario_ids:
    metrics_present = {m for s, m in bands if s == sid}
    if metrics_present != expected_metrics:
        fail(f"{sid}: noise_bands metrics = {sorted(metrics_present)}, expected {sorted(expected_metrics)}")

# --- AC-011 / basic mechanism: scenario-suff (2 reps == min_reps) publishes
# a real band, never insufficient_reps.
elapsed_band = bands[("scenario-suff", "elapsed_time")]
if elapsed_band["insufficient_reps"] is not False:
    fail(f"scenario-suff elapsed_time insufficient_reps = {elapsed_band['insufficient_reps']!r}, expected False")
if elapsed_band["rep_count"] != 2:
    fail(f"scenario-suff elapsed_time rep_count = {elapsed_band['rep_count']!r}, expected 2")
if elapsed_band["min"] != 10.0 or elapsed_band["max"] != 12.0 or elapsed_band["median"] != 11.0:
    fail(f"scenario-suff elapsed_time min/median/max = {elapsed_band['min']}/{elapsed_band['median']}/{elapsed_band['max']}, expected 10.0/11.0/12.0")
if elapsed_band["spread_abs"] != 2.0:
    fail(f"scenario-suff elapsed_time spread_abs = {elapsed_band['spread_abs']!r}, expected 2.0")
# acceptance interval: spread_abs (2.0) > 0 -> r_eff = 2.0 -> [8.0, 14.0].
interval = elapsed_band["acceptance_interval"]
if interval["lower_bound"] != 8.0 or interval["upper_bound"] != 14.0:
    fail(f"scenario-suff elapsed_time acceptance_interval = {interval}, expected lower_bound=8.0 upper_bound=14.0")
if elapsed_band["derivation_rule"] != "min_median_max_spread_accept_interval":
    fail(f"scenario-suff elapsed_time derivation_rule = {elapsed_band['derivation_rule']!r}, expected the schema-owned name")

cost_band = bands[("scenario-suff", "provider_cost")]
if cost_band["insufficient_reps"] is not False:
    fail(f"scenario-suff provider_cost insufficient_reps = {cost_band['insufficient_reps']!r}, expected False")
if cost_band["rep_count"] != 2:
    fail(f"scenario-suff provider_cost rep_count = {cost_band['rep_count']!r}, expected 2")

# --- AC-T1: scenario-insuff has 1 rep, below the batch's declared
# min_reps=2 -- insufficient_reps MUST be true, never a silently published
# band as though the data were sufficient.
for metric_name in ("elapsed_time", "provider_cost"):
    band = bands[("scenario-insuff", metric_name)]
    if band["insufficient_reps"] is not True:
        fail(f"scenario-insuff {metric_name} insufficient_reps = {band['insufficient_reps']!r}, expected True (AC-T1)")
    if band["rep_count"] != 1:
        fail(f"scenario-insuff {metric_name} rep_count = {band['rep_count']!r}, expected 1")

# --- confirmed_findings (quality dimension) is unavailable for
# scenario-insuff/scenario-ineligible (fixture never set it) -- renders
# 'unavailable', never a fabricated zero-width band (ADR-F10-10).
for sid in ("scenario-insuff", "scenario-ineligible"):
    band = bands[(sid, "confirmed_findings")]
    if band["rep_count"] != 0:
        fail(f"{sid} confirmed_findings rep_count = {band['rep_count']!r}, expected 0 (no fixture value set)")
    if band["insufficient_reps"] is not True:
        fail(f"{sid} confirmed_findings insufficient_reps = {band['insufficient_reps']!r}, expected True")
    for field in ("min", "median", "max", "spread_abs"):
        if band[field] != "unavailable":
            fail(f"{sid} confirmed_findings.{field} = {band[field]!r}, expected the literal string 'unavailable'")
    if band["acceptance_interval"]["lower_bound"] != "unavailable" or band["acceptance_interval"]["upper_bound"] != "unavailable":
        fail(f"{sid} confirmed_findings acceptance_interval = {band['acceptance_interval']}, expected both bounds 'unavailable'")

# scenario-suff DOES have both confirmed_findings values (2, 4) set.
suff_confirmed = bands[("scenario-suff", "confirmed_findings")]
if suff_confirmed["rep_count"] != 2 or suff_confirmed["insufficient_reps"] is not False:
    fail(f"scenario-suff confirmed_findings rep_count/insufficient_reps = {suff_confirmed['rep_count']}/{suff_confirmed['insufficient_reps']}, expected 2/False")
if suff_confirmed["min"] != 2 or suff_confirmed["max"] != 4 or suff_confirmed["median"] != 3:
    fail(f"scenario-suff confirmed_findings min/median/max = {suff_confirmed['min']}/{suff_confirmed['median']}/{suff_confirmed['max']}, expected 2/3/4")

# ---------------------------------------------------------------------------
# comparisons[]: republished verbatim from the retained comparison.json.
# ---------------------------------------------------------------------------
if len(agg["comparisons"]) != 1:
    fail(f"comparisons has {len(agg['comparisons'])} entries, expected exactly 1")
comparison_entry = agg["comparisons"][0]
if comparison_entry["scenario_id"] != "scenario-suff" or comparison_entry["rep"] != 1:
    fail(f"comparisons[0] scenario_id/rep = {comparison_entry['scenario_id']!r}/{comparison_entry['rep']!r}, expected scenario-suff/1")
if comparison_entry["mode"] != "independent_frozen_candidate":
    fail(f"comparisons[0].mode = {comparison_entry['mode']!r}, expected independent_frozen_candidate")
if comparison_entry["left_evaluation_id"] != "eval-scenario-suff-1" or comparison_entry["right_evaluation_id"] != "eval-candidate-x":
    fail(f"comparisons[0] evaluation ids = {comparison_entry['left_evaluation_id']!r}/{comparison_entry['right_evaluation_id']!r}, expected the fixture's verbatim ids")
if comparison_entry["accepted"] is not True:
    fail(f"comparisons[0].accepted = {comparison_entry['accepted']!r}, expected True (verbatim from comparison.json)")
if comparison_entry["divergences"] != []:
    fail(f"comparisons[0].divergences = {comparison_entry['divergences']!r}, expected [] (verbatim)")

# ---------------------------------------------------------------------------
# invalid[]: both sources present, AC-T2 verbatim eligibility copy.
# ---------------------------------------------------------------------------
invalid_by_scenario = {row["scenario_id"]: row for row in agg["invalid"]}
if set(invalid_by_scenario) != {"scenario-failed", "scenario-ineligible"}:
    fail(f"invalid[] scenario_id set = {sorted(invalid_by_scenario)}, expected scenario-failed/scenario-ineligible")

failed_row = invalid_by_scenario["scenario-failed"]
if failed_row["source"] != "batch_dispatch_failure":
    fail(f"scenario-failed invalid row source = {failed_row['source']!r}, expected batch_dispatch_failure")
if failed_row["invalidity_reasons"] != ["lifecycle_run_failed"]:
    fail(f"scenario-failed invalid row invalidity_reasons = {failed_row['invalidity_reasons']!r}, expected ['lifecycle_run_failed'] (mirrored verbatim)")
if failed_row["retention_path"] != "scenarios/scenario-failed/1":
    fail(f"scenario-failed invalid row retention_path = {failed_row['retention_path']!r}, expected scenarios/scenario-failed/1")

ineligible_row = invalid_by_scenario["scenario-ineligible"]
if ineligible_row["source"] != "i08_eligibility":
    fail(f"scenario-ineligible invalid row source = {ineligible_row['source']!r}, expected i08_eligibility")

# AC-T2: the invalid[] eligibility object must be byte-for-byte identical
# (no recomputation, override, or softening) to the SAME pair's
# scenarios[].eligibility -- fetched independently and compared, not
# assumed equal by construction.
scenario_row = next(s for s in agg["scenarios"] if s["scenario_id"] == "scenario-ineligible")
if ineligible_row["eligibility"] != scenario_row["eligibility"]:
    fail(
        f"invalid[] eligibility {ineligible_row['eligibility']!r} != scenarios[].eligibility "
        f"{scenario_row['eligibility']!r} -- AC-T2 requires a verbatim copy"
    )
if ineligible_row["eligibility"]["aggregate_eligible"] is not False:
    fail(f"scenario-ineligible eligibility.aggregate_eligible = {ineligible_row['eligibility']['aggregate_eligible']!r}, expected False")
if ineligible_row["eligibility"]["publication_eligible"] is not False:
    fail(f"scenario-ineligible eligibility.publication_eligible = {ineligible_row['eligibility']['publication_eligible']!r}, expected False")
if ineligible_row["eligibility"]["invalidity_reasons"] != [{"code": "missing_oracle", "path": "/execution_oracle", "detail": "oracle did not produce a result"}]:
    fail(f"scenario-ineligible eligibility.invalidity_reasons = {ineligible_row['eligibility']['invalidity_reasons']!r}, not copied verbatim")

# scenario-suff and scenario-insuff are eligible -- must NOT appear in
# invalid[] at all (no over-inclusion).
if "scenario-suff" in invalid_by_scenario or "scenario-insuff" in invalid_by_scenario:
    fail("an eligible scenario appears in invalid[] -- over-inclusion")

# ---------------------------------------------------------------------------
# REQ-F-016: no composite/weighted/blended score field anywhere in the
# actual aggregate.json OUTPUT (not just the schema, checked separately
# below).
# ---------------------------------------------------------------------------
FORBIDDEN_TERMS = ("efficiency", "value_score", "roi", "composite", "weighted_score", "blended")


def walk_keys(obj):
    if isinstance(obj, dict):
        for key, value in obj.items():
            yield key
            yield from walk_keys(value)
    elif isinstance(obj, list):
        for item in obj:
            yield from walk_keys(item)


for key in walk_keys(agg):
    lowered = key.lower()
    for term in FORBIDDEN_TERMS:
        if term in lowered:
            fail(f"aggregate.json output field {key!r} matches forbidden_composite_fields term {term!r}")

print(
    "TC-088 (aggregator half): noise_bands publishes a real "
    "min/median/max/spread_abs/acceptance_interval band per (scenario, "
    "metric) with insufficient_reps=false when reps meet the batch's "
    "declared minimum, and insufficient_reps=true (AC-T1) when a scenario's "
    "rep count falls below it; comparisons[] republishes the comparator's "
    "verdict verbatim; invalid[] carries every invalidity_reasons entry "
    "from both the batch dispatch-failure ledger and retained-but-"
    "ineligible pairs, the latter an exact verbatim copy of "
    "scenarios[].eligibility (AC-T2); the aggregate output carries no "
    "composite/weighted/blended score field"
)
PYEOF

# ---------------------------------------------------------------------------
# Target (i): static scan of bench/reports/lifecycle-baseline-schema.yaml's
# field/property definitions (aggregate_required_fields,
# aggregate_properties) for forbidden_composite_fields term matches,
# excluding the schema's own forbidden_composite_fields list declaration
# (which necessarily names the terms it forbids).
# ---------------------------------------------------------------------------
python3 - "$SCHEMA" <<'PYEOF'
import sys

import yaml

schema_path = sys.argv[1]
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f)


def fail(msg):
    print(f"TC-088 FAIL (schema static scan): {msg}", file=sys.stderr)
    raise SystemExit(1)


forbidden = schema.get("forbidden_composite_fields")
if not forbidden:
    fail("schema forbidden_composite_fields list is empty or missing")

required_fields = schema.get("aggregate_required_fields") or []
properties = schema.get("aggregate_properties") or {}
if not required_fields:
    fail("schema aggregate_required_fields is empty or missing")


def segments(pointer):
    parts = pointer.strip("/").split("/")
    return [p[:-2] if p.endswith("[]") else p for p in parts if p]


# Field/property definitions ONLY -- aggregate_required_fields and
# aggregate_properties pointers -- deliberately excludes the
# forbidden_composite_fields list declaration itself (this scan never even
# reads that key's own values as candidate property names).
violations = []
for pointer in list(required_fields) + list(properties):
    for segment in segments(pointer):
        for term in forbidden:
            if term.lower() in segment.lower():
                violations.append((pointer, segment, term))

if violations:
    fail(f"schema field/property definitions contain forbidden_composite_fields matches: {violations}")

print(
    f"TC-088 (schema static scan, target i): zero property-name matches "
    f"against {len(forbidden)} forbidden_composite_fields terms across "
    f"{len(required_fields)} required-field pointers and {len(properties)} "
    f"typed properties (schema's own list declaration excluded)"
)
PYEOF

echo "TC-088 (aggregator half): PASS"
