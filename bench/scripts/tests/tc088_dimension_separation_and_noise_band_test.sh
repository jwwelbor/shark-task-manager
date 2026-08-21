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

# ---------------------------------------------------------------------------
# T-E40-F10-011 (report half), target (ii): static scan of the enumerated
# F10 report-template file set (same list as TC-086's Input) for
# forbidden_composite_fields matches, ANYWHERE in the file, with NO
# exclusion (unlike target (i)'s schema-declaration exclusion). The single
# short single-token term (`roi`) is matched on a word boundary -- a plain
# substring scan for "roi" false-positives on ordinary English words with
# no connection to the forbidden concept (e.g. a prose sentence using
# "prior"); the remaining terms are distinctive multi-character identifiers
# safe to match as plain case-insensitive substrings.
# ---------------------------------------------------------------------------
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
python3 - "$BENCH_DIR" "$SCHEMA" <<'PYEOF'
import glob
import os
import re
import sys

import yaml

bench_dir, schema_path = sys.argv[1], sys.argv[2]
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f)


def fail(msg):
    print(f"TC-088 FAIL (report-template static scan, target ii): {msg}", file=sys.stderr)
    raise SystemExit(1)


forbidden = schema.get("forbidden_composite_fields")
if not forbidden:
    fail("schema forbidden_composite_fields list is empty or missing")

# Same enumerated file set as TC-086's Input. bench/reports/templates/* and
# bench/reports/*.md both contribute zero files today (report-lifecycle.sh
# renders inline, T-E40-F10-011's Scope; no bench/reports/*.md is committed
# yet) -- a valid, vacuously-passing scan target, not a silently-skipped one.
files = [
    f"{bench_dir}/scripts/run-lifecycle-batch.sh",
    f"{bench_dir}/scripts/run-review-comparison.sh",
    f"{bench_dir}/scripts/pilot-ledger.sh",
    f"{bench_dir}/scripts/verify-retention-root.sh",
    f"{bench_dir}/scripts/aggregate-lifecycle.sh",
    f"{bench_dir}/scripts/report-lifecycle.sh",
    f"{bench_dir}/scripts/lib/spend-gate.sh",
]
files += sorted(glob.glob(f"{bench_dir}/reports/*.md"))
files += sorted(glob.glob(f"{bench_dir}/reports/templates/*"))

for f in files:
    if not os.path.isfile(f):
        fail(f"enumerated file missing: {f}")

violations = []
for file_path in files:
    with open(file_path, encoding="utf-8") as fh:
        lowered = fh.read().lower()
    for term in forbidden:
        term_lower = term.lower()
        if len(term_lower) <= 4 and " " not in term_lower and "_" not in term_lower and "-" not in term_lower:
            if re.search(r"\b" + re.escape(term_lower) + r"\b", lowered):
                violations.append((file_path, term))
        else:
            if term_lower in lowered:
                violations.append((file_path, term))

if violations:
    fail(f"forbidden_composite_fields matches found (no exclusion, target ii): {violations}")

print(
    f"TC-088 (report-template static scan, target ii): zero "
    f"forbidden_composite_fields matches across {len(files)} enumerated F10 "
    f"files (no exclusion) against {len(forbidden)} closed terms"
)
PYEOF

echo "TC-088 (report half, target ii): PASS"

# ---------------------------------------------------------------------------
# T-E40-F10-011 (report half): "no detectable effect" paired-delta
# rendering (task Goal, REQ-F-016). report-lifecycle.sh is a pure function
# of one aggregate.json (ADR-F10-06), so a hand-built aggregate is a
# legitimate input to prove this rendering branch -- upstream
# compare-lifecycle-evaluations.sh hardcodes quality_delta to null today
# (checked: line 191), so a real aggregator run can never exercise the
# inside/outside-band classification; this is the only way to prove the
# branch live rather than leaving it dead code.
# ---------------------------------------------------------------------------
REPORTER="$SCRIPTS_DIR/report-lifecycle.sh"
[[ -x "$REPORTER" ]] || fail "bench/scripts/report-lifecycle.sh missing or not executable"

HANDBUILT="$WORKDIR/handbuilt-aggregate.json"
python3 - "$HANDBUILT" <<'PYEOF'
import json
import sys


def band(scenario_id, metric, lo, hi):
    return {
        "scenario_id": scenario_id, "metric": metric,
        "min": lo, "median": (lo + hi) / 2, "max": hi, "spread_abs": hi - lo,
        "acceptance_interval": {"lower_bound": lo, "upper_bound": hi},
        "derivation_rule": "min_median_max_spread_accept_interval",
        "rep_count": 2, "insufficient_reps": False,
    }


def scenario(scenario_id):
    return {
        "scenario_id": scenario_id, "scenario_version": "1", "family": f"family-{scenario_id}",
        "rep": 1, "retention_path": f"scenarios/{scenario_id}/1",
        "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
        "outcome": {"terminal": "complete"}, "source_digests": {},
    }


def comparison(scenario_id, quality_delta):
    return {
        "scenario_id": scenario_id, "rep": 1, "retention_path": f"scenarios/{scenario_id}/1",
        "mode": "independent_frozen_candidate",
        "left_evaluation_id": f"eval-{scenario_id}-left", "right_evaluation_id": f"eval-{scenario_id}-right",
        "accepted": True,
        "comparison": {"accepted": True, "quality_delta": quality_delta, "causal_claim": None,
                       "newly_confirmed_findings": [], "intervening_candidates": []},
        "divergences": [],
    }


agg = {
    "identity": {
        "schema_version": "1.0", "batch_id": "batch-tc088-handbuilt", "phase": "lifecycle_v2",
        "retention_root_digest": "a" * 64, "batch_policy_digest": "b" * 64,
        "ceilings": {"max_cost_usd": 100, "max_wall_clock_seconds": 3600, "max_generated_tasks": 20},
        "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
        "min_reps": 2,
    },
    "scenarios": [scenario("scenario-inside-band"), scenario("scenario-outside-band")],
    "time": {"lifecycle_wall_seconds": 0, "stage_category": {"unattributed": 0}, "interval_category": {"unattributed": 0}, "share_partition": {"unattributed": 0}},
    "cost": {"stage_category": {"unattributed": 0}, "interval_category": {"unattributed": 0}, "share_partition": {"unattributed": 0}, "ceiling_consumption": {"observed_cost_usd": 0, "max_cost_usd": 100}},
    "quality": {"by_scenario": [], "first_pass_yield": "unavailable"},
    "review_value": {"gates": []},
    "artifact_use": {"produced_count": 0, "consumed_count": 0, "reused_count": 0, "orphan_count": 0, "edges": [],
                      "replayed_interaction_proxy": {"label": "replayed_interaction_proxy: a replayed proxy",
                                                      "request_count": "unavailable", "response_count": "unavailable",
                                                      "payload_size_bytes": "unavailable", "revision_count": "unavailable",
                                                      "unresolved_gate_count": "unavailable"}},
    "noise_bands": [
        band("scenario-inside-band", "confirmed_findings", 1, 5),
        band("scenario-inside-band", "elapsed_time", 1, 5),
        band("scenario-inside-band", "provider_cost", 1, 5),
        band("scenario-outside-band", "confirmed_findings", 1, 5),
        band("scenario-outside-band", "elapsed_time", 1, 5),
        band("scenario-outside-band", "provider_cost", 1, 5),
    ],
    # scenario-inside-band's quality_delta (3) is INSIDE [1, 5] -> "no
    # detectable effect". scenario-outside-band's quality_delta (9) is
    # OUTSIDE [1, 5] -> reported as clearing the band, never "no detectable
    # effect" and never an unstated improvement/regression claim.
    "comparisons": [comparison("scenario-inside-band", 3), comparison("scenario-outside-band", 9)],
    "invalid": [],
}
with open(sys.argv[1], "w", encoding="utf-8") as f:
    json.dump(agg, f, sort_keys=True)
PYEOF

REPORT_OUT="$WORKDIR/handbuilt-headline.md"
"$REPORTER" --aggregate "$HANDBUILT" --view headline >"$REPORT_OUT" 2>"$WORKDIR/handbuilt.err" || fail "reporter exited non-zero on hand-built aggregate; stderr: $(cat "$WORKDIR/handbuilt.err")"

python3 - "$REPORT_OUT" <<'PYEOF'
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    report = f.read()


def fail(msg):
    print(f"TC-088 FAIL (no-detectable-effect python check): {msg}", file=sys.stderr)
    raise SystemExit(1)


dimension_heading = "## Dimension separation and paired deltas"
if dimension_heading not in report:
    fail("headline report has no 'Dimension separation and paired deltas' section")
dimension_block = report[report.index(dimension_heading):]


def section_for(scenario_id):
    marker = f"### `{scenario_id}`"
    if marker not in dimension_block:
        fail(f"dimension-separation section has no sub-section for {scenario_id!r}")
    start = dimension_block.index(marker)
    rest = dimension_block[start + len(marker):]
    next_marker = rest.find("\n### `")
    return rest if next_marker == -1 else rest[:next_marker]


inside = section_for("scenario-inside-band")
if "Paired delta: 3 -- no detectable effect" not in inside:
    fail("scenario-inside-band: in-band quality delta did not render exactly 'no detectable effect'")

outside = section_for("scenario-outside-band")
if "no detectable effect" in outside:
    fail("scenario-outside-band: an out-of-band delta must never render 'no detectable effect'")
if "Paired delta: 9" not in outside:
    fail("scenario-outside-band: the out-of-band delta value (9) is not printed")
if "outside this scenario's published confirmed_findings noise band" not in outside:
    fail("scenario-outside-band: headline does not name the delta as clearing the published noise band")

# Time/cost dimensions carry no upstream per-dimension delta field --
# rendered "unavailable", never a value this script derives itself.
for section in (inside, outside):
    if section.count("Paired delta: unavailable") != 2:
        fail("time and cost dimensions must both render 'Paired delta: unavailable' (no upstream delta field)")

print(
    "TC-088 (no-detectable-effect check): an in-band quality paired delta "
    "renders exactly 'no detectable effect'; an out-of-band delta renders "
    "its value without that phrase and without an improvement/regression "
    "claim; time and cost dimensions render 'unavailable' (no upstream "
    "per-dimension delta field)"
)
PYEOF

echo "TC-088 (no-detectable-effect check): PASS"

# ===========================================================================
# T-E40-F10-010 rework (code-review-2026-08-20T2138-E40-F10.md findings 2
# and 8; kickback on this task): chains a REAL `run-review-comparison.sh
# --mode pilot` run into a REAL `aggregate-lifecycle.sh` run, end to end.
#
# Finding 2's original defect: run-review-comparison.sh retained gate
# evidence under gate-named directories (`scenarios/<id>/qa/`,
# `scenarios/<id>/deep_review/`) and a scenario-level comparison.json, which
# aggregate-lifecycle.sh's `int(rep)` pair loop could never find and, worse,
# crashed on ("rep 'qa' is not an integer") whenever a comparison shared a
# retention root with a batch run. Finding 8: no committed test chained a
# real run-review-comparison.sh retention output into a real
# aggregate-lifecycle.sh run, which is exactly how findings 1 and 2 shipped
# undetected through two rework rounds.
#
# T-E40-F10-005's rework (commit 63a7605a, `bench/scripts/lib/retain_pair`)
# fixed the producer: `retain_gate()` now writes through the SAME shared
# manifest builder `run-lifecycle-batch.sh`'s `retain_pair()` uses, at the
# SAME (scenario_id, rep) layout -- qa=rep 1, deep_review=rep 2
# (`gate_rep()`), both real integers. This section proves the fixed
# producer's real output is correctly consumed by the aggregator:
#   1. `run-review-comparison.sh` is driven for REAL (real retain_gate()/
#      lib/retain_pair path -- both gates dispatch from scratch, never the
#      skipped_complete shortcut, so retain_gate() actually executes; real,
#      unstubbed compare-lifecycle-evaluations.sh). Only
#      RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN are stubbed, matching this
#      suite's existing convention (tc082's (a2), tc087).
#   2. The resulting retention root also carries a hand-built batch pair
#      (rep 3) of the SAME scenario_id -- reproducing finding 2's exact
#      co-location scenario ("if a comparison is ever run against the same
#      --retention-root as a batch").
#   3. `aggregate-lifecycle.sh` runs for real, unstubbed, over the combined
#      root and must not crash; its `/comparisons` block must carry the
#      real published comparison.json verbatim.
# ===========================================================================
COMPARISON="$SCRIPTS_DIR/run-review-comparison.sh"
[[ -x "$COMPARISON" ]] || fail "bench/scripts/run-review-comparison.sh missing or not executable"

E2E_WORKDIR="$WORKDIR/e2e-comparison-chain"
E2E_ROOT="$E2E_WORKDIR/root"
mkdir -p "$E2E_ROOT"
E2E_SCENARIO_ID="scenario-e2e-comparison"

# --- candidate.yaml resolution inputs: a real scenario index/package -------
E2E_INDEX_DIR="$E2E_WORKDIR/index"
mkdir -p "$E2E_INDEX_DIR/packages/$E2E_SCENARIO_ID"
cat >"$E2E_INDEX_DIR/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$E2E_SCENARIO_ID
EOF
cat >"$E2E_INDEX_DIR/packages/$E2E_SCENARIO_ID/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$E2E_SCENARIO_ID"
scenario_version: "1"
entity_family: "family-e2e-comparison"
EOF

E2E_SCRATCH="$E2E_WORKDIR/scratch-template"
mkdir -p "$E2E_SCRATCH"
echo "placeholder scratch project (never mutated -- the driver copies it)" >"$E2E_SCRATCH/marker.txt"

E2E_I05_BUNDLE="$E2E_WORKDIR/i05-bundle"
mkdir -p "$E2E_I05_BUNDLE/transcripts"
echo '{"stage": "code", "note": "tc088 e2e comparison-chain evidence"}' >"$E2E_I05_BUNDLE/stage.json"
echo "tc088 e2e comparison-chain transcript" >"$E2E_I05_BUNDLE/transcripts/stage.txt"

cat >"$E2E_WORKDIR/candidate.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$E2E_SCENARIO_ID"
scenario_index: "$E2E_INDEX_DIR/scenarios.yaml"
gates:
  qa:
    root_key: "ROOT-E2E"
    scratch_root: "$E2E_SCRATCH"
    i05_bundle_dir: "$E2E_I05_BUNDLE"
  deep_review:
    root_key: "ROOT-E2E"
    scratch_root: "$E2E_SCRATCH"
    i05_bundle_dir: "$E2E_I05_BUNDLE"
EOF

# --- Fixture content the RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN stubs
# write to whatever --output path the real dispatch_gate() hands them.
# Identical evaluation content on both gates (same candidate identity, same
# workflow_policy) so the REAL compare-lifecycle-evaluations.sh genuinely
# ACCEPTS the pair -- an accepted comparison is the only kind ever
# published, and publication is exactly the path finding 2 broke.
E2E_LC_FIXTURE="$E2E_WORKDIR/lifecycle-fixture.jsonl"
E2E_EV_FIXTURE="$E2E_WORKDIR/evaluation-fixture.jsonl"
python3 - "$E2E_SCENARIO_ID" "$E2E_LC_FIXTURE" "$E2E_EV_FIXTURE" <<'PYEOF'
import hashlib
import json
import sys

scenario_id, lc_path, ev_path = sys.argv[1:4]
digest = "a" * 64

base_stage = {
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
        "schema_version": "1.0", "run_id": "run-e2e", "scenario_id": scenario_id,
        "scenario_version": "1", "fixture_id": "fixture-e2e", "fixture_digest": digest,
        "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": digest, "shark_content_digest": digest, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": [base_stage],
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {
        "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
        "observed_cost_usd": 1.0, "observed_wall_clock_seconds": 10.0,
        "observed_generated_tasks": 1, "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "e2e comparison-chain fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(lc_path, "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")


def candidate_digest_fields(candidate):
    return {key: candidate[key] for key in ("base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest")}


def with_identity_digest(candidate):
    candidate = dict(candidate)
    candidate["identity_digest"] = hashlib.sha256(json.dumps(candidate_digest_fields(candidate), sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    return candidate


base_candidate = with_identity_digest({
    "base_commit": "b" * 40, "tree_digest": digest, "binary_diff_digest": digest,
    "changed_path_digest": digest, "dirty_untracked_manifest": digest,
    "test_suite_digest": digest, "snapshot_digest": digest,
})

identity = {
    "scenario_id": scenario_id, "scenario_version": "1", "fixture_id": "f", "fixture_digest": digest,
    "adapter_id": "a", "adapter_version": "1", "toolchain_identity": [{"key": "go", "value": "1"}],
    "shark_binary_digest": digest, "shark_content_digest": digest, "rendered_prompt_digest": digest,
    "rendered_prompt_digests": [digest], "provider": "fixture", "model": "m", "effort": "low",
    "provider_identity": [{"stage": "qa", "provider": "fixture", "model": "m", "effort": "low"}],
    "judge_digest": digest, "judge_identity": {"model": "j", "configuration": "c"},
    "reference_digest": digest, "reference_digests": [digest], "resource_policy_digest": digest,
}

policy = {
    "enabled_gates": ["qa", "deep_review"], "gate_order": ["qa", "deep_review"],
    "reviewer": {"provider": "fixture", "model": "m", "effort": "low"},
    "prompt_digest": digest, "rendered_prompt_digest": digest,
    "review_bundle_digest": digest, "deep_review_bundle_digest": digest,
    "fixes_allowed_between_gates": False, "fix_policy": "none",
}
policy["workflow_policy_identity_digest"] = hashlib.sha256(
    json.dumps(policy, sort_keys=True, separators=(",", ":")).encode()
).hexdigest()

ev = {
    "schema_version": "1.0",
    "evaluation_id": "eval-e2e",
    "identity": identity,
    "workflow_policy": policy,
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [{"stage": "qa", "candidate": base_candidate}],
    "review_findings": {"normalized_findings": [], "derived_counts": {}},
    "structural": {"applicability": "applicable", "checks": [], "observed_result": "pass"},
    "judge": {"applicability": "not_applicable", "observed_result": "not_applicable", "invalidity_reasons": []},
    "execution_oracle": {"predicate_kind": "acceptance_tests", "observed_result": "pass"},
    "source_artifacts": {},
    "comparison": {},
    "metrics": {
        "elapsed_time": {"value": 10.0, "available": True},
        "provider_cost": {"value": 1.0, "available": True},
        "rework": {"value": 0, "available": True},
    },
}
with open(ev_path, "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")
PYEOF

E2E_RUN_STUB="$E2E_WORKDIR/run-lifecycle-stub.sh"
cat >"$E2E_RUN_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
cp "$E2E_LC_FIXTURE" "\$output"
EOF
chmod +x "$E2E_RUN_STUB"

E2E_EVAL_STUB="$E2E_WORKDIR/evaluate-lifecycle-stub.sh"
cat >"$E2E_EVAL_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
cp "$E2E_EV_FIXTURE" "\$output"
EOF
chmod +x "$E2E_EVAL_STUB"

# --- Co-located batch pair (rep 3), hand-built (this task's own file
# already hand-builds batch pairs above; batch.json's real writer is
# T-E40-F10-008's scope, not this driver's) -- SAME scenario_id as the
# comparison gates, reproducing finding 2's "comparison co-located with a
# batch run in the same retention root" scenario literally.
# ---------------------------------------------------------------------------
python3 - "$E2E_ROOT" "$E2E_SCENARIO_ID" <<'PYEOF'
import hashlib
import json
import os
import sys

dest_root, scenario_id = sys.argv[1:3]
rep = 3
pair_dir = os.path.join(dest_root, "scenarios", scenario_id, str(rep))
os.makedirs(pair_dir, exist_ok=True)


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def write_json(path, obj):
    with open(path, "w", encoding="utf-8") as f:
        f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")


package_yaml = os.path.join(pair_dir, "package.yaml")
with open(package_yaml, "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{scenario_id}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-e2e-comparison"\n'
    )

base_stage = {
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
        "schema_version": "1.0", "run_id": f"run-{scenario_id}-{rep}", "scenario_id": scenario_id,
        "scenario_version": "1", "fixture_id": f"fixture-{scenario_id}", "fixture_digest": "c" * 64,
        "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": [base_stage],
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {
        "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
        "observed_cost_usd": 0.9, "observed_wall_clock_seconds": 9.0,
        "observed_generated_tasks": 1, "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "batch co-location fixture", "partial_evidence": False, "publication_eligible": True},
}
write_json(os.path.join(pair_dir, "lifecycle.jsonl"), lc)

ev = {
    "schema_version": "1.0", "evaluation_id": f"eval-{scenario_id}-{rep}", "identity": {}, "source_artifacts": {},
    "structural": {}, "judge": {}, "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
    "metrics": {
        "elapsed_time": {"value": 9.0, "available": True},
        "provider_cost": {"value": 0.9, "available": True},
        "rework": {"value": 0, "available": True},
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
print("rep-3 batch-co-location pair written", file=sys.stderr)
PYEOF

python3 - "$E2E_ROOT" <<'PYEOF'
import json
import sys

dest_root = sys.argv[1]
batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc088-e2e", "mode": "baseline", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(f"{dest_root}/batch.json", "w", encoding="utf-8") as f:
    f.write(json.dumps(batch, sort_keys=True, separators=(",", ":")) + "\n")
PYEOF

# --- Drive the REAL run-review-comparison.sh --mode pilot. Neither gate's
# evaluation.jsonl exists yet under $E2E_ROOT, so this is the real dispatch
# path -- retain_gate()/lib/retain_pair genuinely executes for both gates,
# not the skipped_complete shortcut. ---------------------------------------
E2E_ACK_FLAGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10)
e2e_comparison_rc=0
RUN_LIFECYCLE_BIN="$E2E_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$E2E_EVAL_STUB" \
	"$COMPARISON" --candidate "$E2E_WORKDIR/candidate.yaml" --retention-root "$E2E_ROOT" \
	--mode pilot --comparison-mode independent_frozen_candidate "${E2E_ACK_FLAGS[@]}" \
	>"$E2E_WORKDIR/comparison.stdout" 2>"$E2E_WORKDIR/comparison.stderr" || e2e_comparison_rc=$?
[[ "$e2e_comparison_rc" -eq 0 ]] || fail "T-E40-F10-010 rework: real run-review-comparison.sh dispatch/accept failed: exit $e2e_comparison_rc; stdout: $(cat "$E2E_WORKDIR/comparison.stdout"); stderr: $(cat "$E2E_WORKDIR/comparison.stderr")"

E2E_QA_MANIFEST="$E2E_ROOT/scenarios/$E2E_SCENARIO_ID/1/manifest.json"
E2E_DR_MANIFEST="$E2E_ROOT/scenarios/$E2E_SCENARIO_ID/2/manifest.json"
[[ -f "$E2E_QA_MANIFEST" ]] || fail "T-E40-F10-010 rework: qa gate's real retain_gate()/lib/retain_pair path did not retain rep 1"
[[ -f "$E2E_DR_MANIFEST" ]] || fail "T-E40-F10-010 rework: deep_review gate's real retain_gate()/lib/retain_pair path did not retain rep 2"

E2E_COMPARISON_JSON="$E2E_ROOT/scenarios/$E2E_SCENARIO_ID/2/comparison.json"
[[ -f "$E2E_COMPARISON_JSON" ]] || fail "T-E40-F10-010 rework: run-review-comparison.sh did not publish comparison.json at the real (scenario, rep) pair path scenarios/$E2E_SCENARIO_ID/2/comparison.json"

# --- Chain into a REAL, unstubbed aggregate-lifecycle.sh run over the
# combined root (2 real comparison-driven pairs + 1 hand-built batch pair,
# same scenario_id). Before the T-E40-F10-005 fix this crashed with
# "retention directory rep 'qa' is not an integer"; it must not crash now.
E2E_AGG_OUT="$("$AGGREGATOR" --retention-root "$E2E_ROOT" 2>"$E2E_WORKDIR/aggregate.stderr")" \
	|| fail "T-E40-F10-010 rework: real aggregate-lifecycle.sh crashed over a retention root co-locating a real run-review-comparison.sh output with a batch pair of the SAME scenario_id (finding 2's exact scenario); stderr: $(cat "$E2E_WORKDIR/aggregate.stderr")"
[[ ! -s "$E2E_WORKDIR/aggregate.stderr" ]] || fail "T-E40-F10-010 rework: aggregate-lifecycle.sh produced unexpected stderr diagnostics over the chained root: $(cat "$E2E_WORKDIR/aggregate.stderr")"

python3 - "$E2E_AGG_OUT" "$E2E_SCENARIO_ID" "$E2E_COMPARISON_JSON" <<'PYEOF'
import json
import sys

agg_json, scenario_id, comparison_json_path = sys.argv[1:4]
agg = json.loads(agg_json)


def fail(msg):
    print(f"TC-088 FAIL (e2e comparison-chain check): {msg}", file=sys.stderr)
    raise SystemExit(1)


# scenarios[]: rep 1 (qa), rep 2 (deep_review), rep 3 (co-located batch
# pair) all present for the SAME scenario_id -- proves the now-real
# integer reps a real run-review-comparison.sh writes coexist with an
# ordinary batch pair under one scenario without the pair loop's
# int(rep) check (aggregate-lifecycle.sh:709-712) ever firing.
reps = sorted(s["rep"] for s in agg["scenarios"] if s["scenario_id"] == scenario_id)
if reps != [1, 2, 3]:
    fail(f"scenario {scenario_id!r} reps = {reps}, expected [1, 2, 3] (qa, deep_review, co-located batch pair)")

# comparisons[]: exactly the ONE real comparison.json run-review-comparison.sh
# published, republished verbatim -- read directly off disk and compared,
# not assumed equal by construction.
with open(comparison_json_path, encoding="utf-8") as fh:
    on_disk = json.load(fh)

matches = [c for c in agg["comparisons"] if c["scenario_id"] == scenario_id]
if len(matches) != 1:
    fail(f"comparisons[] has {len(matches)} entries for {scenario_id!r}, expected exactly 1")
entry = matches[0]
if entry["rep"] != 2:
    fail(f"comparisons[0].rep = {entry['rep']!r}, expected 2 (deep_review, per gate_rep())")
for key in ("mode", "left_evaluation_id", "right_evaluation_id", "accepted", "comparison", "divergences"):
    if entry[key] != on_disk[key]:
        fail(f"comparisons[0].{key} = {entry[key]!r} != the real published comparison.json's {key} = {on_disk[key]!r} (not republished verbatim)")
if entry["accepted"] is not True:
    fail(f"comparisons[0].accepted = {entry['accepted']!r}, expected True (the real comparator accepted the identity-compatible pair)")

print(
    "TC-088 (T-E40-F10-010 rework, real comparison-chain end-to-end): a "
    "real run-review-comparison.sh --mode pilot run (real retain_gate()/"
    "lib/retain_pair dispatch path, real compare-lifecycle-evaluations.sh) "
    "co-located with a batch-produced pair of the SAME scenario_id chains "
    "cleanly into a real aggregate-lifecycle.sh run -- no crash, and the "
    "published comparison.json reaches /comparisons verbatim"
)
PYEOF

echo "TC-088 (T-E40-F10-010 rework, real comparison-chain end-to-end): PASS"
