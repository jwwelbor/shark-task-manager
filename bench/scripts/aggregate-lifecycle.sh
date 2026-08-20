#!/usr/bin/env bash
# aggregate-lifecycle.sh --retention-root <retention_root>
#
# T-E40-F10-008 (spec.md REQ-F-007 (partial), REQ-F-009 (partial),
# REQ-F-010, REQ-F-011, REQ-F-017, ADR-F10-08, ADR-F10-11, REQ-NF-003;
# test-plan.md TC-084, TC-089). Pure lifecycle-v2 aggregator, modelled on
# `aggregate-runs.sh`'s header contract: prints one `aggregate.json`
# document to stdout, diagnostics to stderr, consults no clock, invokes no
# subprocess/provider/network, and touches nothing outside the named
# `--retention-root`.
#
# THIS TASK'S SLICE ONLY (component-changes row, this task's Scope): the
# `identity`, `scenarios[]`, `time`, and `cost` blocks. `quality`,
# `review_value`, and `artifact_use` are T-E40-F10-009's; `noise_bands`,
# `comparisons`, and `invalid` are T-E40-F10-010's. This script's
# `aggregate.json` therefore carries only four of the ten top-level blocks
# `bench/reports/lifecycle-baseline-schema.yaml` eventually requires --
# later tasks extend this same document, they do not replace it.
#
# ---------------------------------------------------------------------------
# Design decisions made by this task (documented here because spec.md does
# not fix them at this level of detail; each is a defensible reading of
# REQ-F-010/REQ-F-011/ADR-F10-04, not an invented business rule):
#
#   1. Whole-root Phase 1 refusal (ADR-F10-11, AC-T1). A file named exactly
#      `record.jsonl` ANYWHERE under the retention root (not just under
#      scenarios/) is Phase 1's fixed record filename -- F10 never retains
#      a file by that name (its own records are `lifecycle.jsonl` /
#      `evaluation.jsonl`) -- so its mere presence is refused with the
#      schema-owned `phase1_record_input_refused` reason before any other
#      work happens, per TC-089's resolution of its own deferred mixed-root
#      question: refuse the WHOLE root, never a partial aggregate.
#
#   2. `time.lifecycle_wall_seconds` / total provider cost are read
#      VERBATIM from each pair's retained `evaluation.jsonl` --
#      `metrics.elapsed_time.value` and `metrics.provider_cost.value`
#      (evaluate-lifecycle.sh's own `derive_metrics()` already computes
#      these as `sum(stage.elapsed_seconds)` / `sum(stage.cost_usd +
#      judge.cost_usd)`). ADR-F10-04 lists `metrics` among the fields F10
#      copies verbatim rather than recomputes; recomputing elapsed/cost
#      independently here would be exactly the "second, divergent
#      implementation" ADR-F10-04 forbids. The per-cell PARTITIONS
#      (stage_category / interval_category / share_partition), which I-08
#      does not carry, are computed here directly from retained I-07
#      `/stages[]/intervals` and reconciled against that verbatim total,
#      with any residual reported as `unattributed` -- this is what makes
#      the reconciliation in TC-084 a real check rather than a tautology.
#      A meaningful non-zero cost `unattributed` case is: a scenario with
#      an evaluated judge cost, since judge cost contributes to
#      `metrics.provider_cost.value` but is not attributable to any
#      (stage_category, interval_category) cell.
#
#   3. Cost-per-cell attribution rule. I-07 does not decompose a stage's
#      `cost_usd` per interval the way it decomposes elapsed time (a
#      stage's `intervals[]` durations already reconcile to its
#      `elapsed_seconds`, but nothing analogous exists for cost). Rather
#      than invent a pro-rata split across a stage's wait/tool/unclassified
#      time -- which would misrepresent provider spend as accruing during
#      idle wait -- this script attributes a stage's entire `cost_usd` to
#      the `(stage.category, "provider_active")` cell: cost is modelled as
#      accruing where the provider was doing the work, never during a wait
#      interval. This is why the REQ-F-011 `wait` share is always $0 for
#      cost by construction, which matches the domain reading "waiting
#      costs nothing" rather than being a computation gap.
#
#   4. `cost.ceiling_consumption.max_cost_usd` is the BATCH-declared
#      ceiling from `/identity/ceilings/max_cost_usd` (the single
#      `--max-cost-usd` value the operator acknowledged for the whole
#      batch, REQ-F-002), not a sum of each pair's own I-07
#      `limits.max_cost_usd` -- those are per-run resource-policy ceilings
#      (`run-lifecycle.sh`'s own `parse_limits()`, sourced from each
#      scenario package's `resource_policy` when `--limits` is not passed)
#      and may legitimately differ per scenario. `observed_cost_usd` is
#      the sum of every pair's I-07 `limits.observed_cost_usd`, i.e. what
#      was actually spent against that declared batch ceiling.
#
#   5. `retention_root_digest` is a digest of digests, not a full-tree
#      content hash: sha256 over the canonical JSON of `batch.json`'s own
#      raw-byte digest plus the sorted list of every retained pair's
#      `manifest.json` raw-byte digest. Every retained artifact's own byte
#      integrity is already covered by that pair's `manifest.json` sha256
#      entries (verify-retention-root.sh is the dedicated byte-preservation
#      checker); re-hashing the full evidence/transcripts payloads here
#      would cost O(retention root size) for no additional integrity
#      signal and would work against REQ-NF-004's streaming/scale
#      requirement for no benefit -- this aggregator does not otherwise
#      need to touch a single byte of evidence/ or transcripts/ content.
#
#   6. No per-scenario `time`/`cost` breakdown array. The `aggregate.json`
#      blocks table describes `time`/`cost` as "per scenario and rolled
#      up", but `bench/reports/lifecycle-baseline-schema.yaml`'s
#      `aggregate_required_fields` only requires the ROLLED-UP pointers
#      (`/time/lifecycle_wall_seconds`, `/time/stage_category`, ...) --
#      unlike `/quality/by_scenario[]`, which IS required. This task keeps
#      to the schema-required (rolled-up) shape rather than inventing an
#      untested `by_scenario` array; a later task may add one without
#      breaking this shape.
# ---------------------------------------------------------------------------
#
# Known upstream gap checked and found NOT BLOCKING for this task: T-007's
# retained finding that `run-lifecycle-batch.sh`'s `retain_pair` only
# populates manifest.json entries for 5 of 7 non-manifest artifacts (no
# evidence/transcripts entries) does not affect this script, because the
# identity/scenarios/time/cost blocks built here never read evidence/ or
# transcripts/ content or their manifest entries -- only package.yaml,
# lifecycle.jsonl, evaluation.jsonl, and manifest.json's entries for THOSE
# three names, all of which retain_pair already populates correctly with
# real source digests.
#
# A SEPARATE, previously undocumented gap in the same file WAS blocking and
# is fixed by this task (see run-lifecycle-batch.sh's batch.json writer):
# batch.json never retained the batch policy's declared `min_reps`, so
# `/identity/min_reps` (a field owned by THIS task) had no retained source
# to read from `--retention-root` alone. Fixed by re-parsing the same
# `batch_policy_path` that writer already opens for `batch_policy_digest`.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

LIFECYCLE_SCHEMA="${LIFECYCLE_SCHEMA:-$BENCH_DIR/reports/lifecycle-baseline-schema.yaml}"
I05_SCHEMA="${I05_SCHEMA:-$BENCH_DIR/evidence/i05-schema.yaml}"

usage() {
	echo "usage: aggregate-lifecycle.sh --retention-root <retention_root>" >&2
	exit 2
}

retention_root=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		[[ $# -ge 2 ]] || usage
		retention_root="$2"
		shift 2
		;;
	--help)
		usage
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$retention_root" ]] || usage
[[ -d "$retention_root" ]] || {
	echo "aggregate-lifecycle: retention root not found: $retention_root" >&2
	exit 2
}
[[ -f "$LIFECYCLE_SCHEMA" ]] || {
	echo "aggregate-lifecycle: schema not found: $LIFECYCLE_SCHEMA" >&2
	exit 2
}
[[ -f "$I05_SCHEMA" ]] || {
	echo "aggregate-lifecycle: i05 schema not found: $I05_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "aggregate-lifecycle: python3 not found on PATH" >&2
	exit 2
}

RETENTION_ROOT_CANON="$(cd "$retention_root" && pwd)"

python3 - "$RETENTION_ROOT_CANON" "$LIFECYCLE_SCHEMA" "$I05_SCHEMA" <<'PYEOF'
import hashlib
import json
import os
import sys

import yaml

retention_root, schema_path, i05_schema_path = sys.argv[1:4]


def fail(msg):
    print("aggregate-lifecycle: " + msg, file=sys.stderr)
    sys.exit(1)


def usage_fail(msg):
    print("aggregate-lifecycle: " + msg, file=sys.stderr)
    sys.exit(2)


# ---------------------------------------------------------------------------
# Phase 0: whole-root Phase 1 v1-record refusal (ADR-F10-11, AC-T1, TC-089).
# A recursive walk, not the scenarios/*/*/ pinned layout -- "anywhere under
# it" per REQ-F-017/AC-T1, including a record.jsonl injected outside the
# scenarios/ tree. This is the ONLY thing this script does before any other
# read, so an accidental Phase 1 root never gets even a partial aggregate.
# ---------------------------------------------------------------------------
v1_records = []
for current, dirs, files in os.walk(retention_root):
    dirs.sort()
    if "record.jsonl" in files:
        v1_records.append(os.path.relpath(os.path.join(current, "record.jsonl"), retention_root))
if v1_records:
    for rel in sorted(v1_records):
        print(
            f"aggregate-lifecycle: refusing entire aggregation: phase1_record_input_refused: "
            f"Phase 1 record.jsonl found at {rel} (ADR-F10-11: whole-root refusal, not a partial aggregate)",
            file=sys.stderr,
        )
    sys.exit(1)

# ---------------------------------------------------------------------------
# Closed vocabularies, read from schema (REQ-F-018) rather than duplicated.
# ---------------------------------------------------------------------------
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}
with open(i05_schema_path, encoding="utf-8") as f:
    i05_schema = yaml.safe_load(f) or {}

SCHEMA_VERSION = schema.get("schema_version")
if not SCHEMA_VERSION:
    usage_fail(f"{schema_path}: missing schema_version")

PHASE_LABELS = schema.get("phase_label") or []
PHASE_LABEL = "lifecycle_v2"
if PHASE_LABEL not in PHASE_LABELS:
    usage_fail(f"{schema_path}: does not declare phase label {PHASE_LABEL!r}")

STAGE_CATEGORIES = i05_schema.get("stage_category") or []
INTERVAL_CATEGORIES = i05_schema.get("interval_category") or []
SHARE_CELLS = schema.get("share_partition_cell") or []
UNATTRIBUTED = schema.get("share_partition_residual", "unattributed")
if not STAGE_CATEGORIES or not INTERVAL_CATEGORIES or not SHARE_CELLS:
    usage_fail("schema vocabularies (stage_category/interval_category/share_partition_cell) are empty or missing")

# REQ-F-011's exact share-assignment rule.
WAIT_INTERVALS = {"queue_or_claim_wait", "replay_or_human_gate_wait", "retry_or_backoff"}
PRE_CODE_STAGES = {"discovery", "specification", "planning"}
REVIEW_STAGES = {"review", "qa", "uat"}


def share_for_cell(stage_category, interval_category, rework):
    if interval_category in WAIT_INTERVALS:
        return "wait"
    if stage_category in PRE_CODE_STAGES:
        return "pre_code"
    if stage_category in REVIEW_STAGES:
        return "review"
    if stage_category == "shipping":
        return "shipping"
    if stage_category == "code":
        return "rework" if rework else "first_pass_code"
    return None  # genuinely unmappable; see TC-084 edge-case note -- never
    # reached for the eight current stage_category values, but the
    # partition-building code below still routes it to unattributed rather
    # than silently dropping it, in case a future vocabulary addition ever
    # produces one.


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    with open(path, "rb") as fh:
        return sha256_bytes(fh.read())


def require(d, key, ctx):
    if not isinstance(d, dict) or key not in d or d[key] in (None, ""):
        fail(f"{ctx}: missing required field {key!r}")
    return d[key]


def read_one_line_json(path, ctx):
    try:
        with open(path, encoding="utf-8") as fh:
            lines = [ln for ln in (raw.strip() for raw in fh.readlines()) if ln]
    except OSError as exc:
        fail(f"{ctx}: cannot read {path}: {exc}")
    if len(lines) != 1:
        fail(f"{ctx}: {path}: expected exactly one JSON record line, got {len(lines)}")
    try:
        obj = json.loads(lines[0])
    except json.JSONDecodeError as exc:
        fail(f"{ctx}: {path}: unparseable JSON: {exc}")
    if not isinstance(obj, dict):
        fail(f"{ctx}: {path}: record is not a JSON object")
    return obj


def is_number(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool)


# ---------------------------------------------------------------------------
# /identity -- from batch.json, the sole retained identity source
# (REQ-NF-003: pure function of --retention-root alone).
# ---------------------------------------------------------------------------
batch_path = os.path.join(retention_root, "batch.json")
if not os.path.isfile(batch_path):
    fail("batch.json not found under retention root -- required for /identity")
with open(batch_path, "rb") as fh:
    batch_bytes = fh.read()
try:
    batch = json.loads(batch_bytes)
except json.JSONDecodeError as exc:
    fail(f"batch.json is not valid JSON: {exc}")

batch_id = require(batch, "batch_id", "batch.json")
batch_phase = require(batch, "phase", "batch.json")
if batch_phase != PHASE_LABEL:
    fail(f"batch.json declares phase {batch_phase!r}, expected {PHASE_LABEL!r}")
batch_policy_digest = require(batch, "batch_policy_digest", "batch.json")
acknowledgement_ref = require(batch, "acknowledgement_ref", "batch.json")

min_reps = batch.get("min_reps")
if not isinstance(min_reps, int) or isinstance(min_reps, bool) or min_reps < 1:
    fail(
        "batch.json missing a valid min_reps (declared batch minimum reps) -- "
        "see run-lifecycle-batch.sh's batch.json writer (T-E40-F10-008 fix)"
    )


def ceiling_number(raw_ceilings, name):
    value = raw_ceilings.get(name)
    if value is None:
        fail(f"batch.json ceilings.{name} is missing")
    try:
        text = str(value)
        if "." in text or "e" in text.lower():
            return float(value)
        return int(value)
    except (TypeError, ValueError):
        fail(f"batch.json ceilings.{name} is not numeric: {value!r}")


raw_ceilings = batch.get("ceilings") or {}
ceilings = {
    "max_cost_usd": ceiling_number(raw_ceilings, "max_cost_usd"),
    "max_wall_clock_seconds": ceiling_number(raw_ceilings, "max_wall_clock_seconds"),
    "max_generated_tasks": ceiling_number(raw_ceilings, "max_generated_tasks"),
}

# ---------------------------------------------------------------------------
# Enumerate retained (scenario, rep) pairs under scenarios/<id>/<rep>/,
# mirroring verify-retention-root.sh's own layout walk.
# ---------------------------------------------------------------------------
scenarios_dir = os.path.join(retention_root, "scenarios")
pair_dirs = []
if os.path.isdir(scenarios_dir):
    for scenario_id in sorted(os.listdir(scenarios_dir)):
        scenario_dir = os.path.join(scenarios_dir, scenario_id)
        if not os.path.isdir(scenario_dir):
            continue
        for rep in sorted(os.listdir(scenario_dir)):
            rep_dir = os.path.join(scenario_dir, rep)
            if not os.path.isdir(rep_dir):
                continue
            pair_dirs.append((scenario_id, rep, rep_dir))

if not pair_dirs:
    fail(f"no retained (scenario, rep) pairs found under {scenarios_dir}")

# Per-cell accumulators for the rolled-up time/cost blocks.
stage_time_totals = {cat: 0.0 for cat in STAGE_CATEGORIES}
interval_time_totals = {cat: 0.0 for cat in INTERVAL_CATEGORIES}
share_time_totals = {cell: 0.0 for cell in SHARE_CELLS}
total_lifecycle_wall_seconds = 0.0

stage_cost_totals = {cat: 0.0 for cat in STAGE_CATEGORIES}
interval_cost_totals = {cat: 0.0 for cat in INTERVAL_CATEGORIES}
share_cost_totals = {cell: 0.0 for cell in SHARE_CELLS}
total_provider_cost_usd = 0.0
total_observed_ceiling_cost = 0.0

scenarios_list = []
manifest_digest_entries = []

for scenario_id, rep, rep_dir in pair_dirs:
    pair_label = f"{scenario_id}/{rep}"

    package_path = os.path.join(rep_dir, "package.yaml")
    lifecycle_path = os.path.join(rep_dir, "lifecycle.jsonl")
    evaluation_path = os.path.join(rep_dir, "evaluation.jsonl")
    manifest_path = os.path.join(rep_dir, "manifest.json")
    for name, path in (
        ("package.yaml", package_path),
        ("lifecycle.jsonl", lifecycle_path),
        ("evaluation.jsonl", evaluation_path),
        ("manifest.json", manifest_path),
    ):
        if not os.path.isfile(path):
            fail(f"{pair_label}: required retained artifact missing: {name}")

    try:
        with open(package_path, encoding="utf-8") as fh:
            package = yaml.safe_load(fh) or {}
    except (OSError, yaml.YAMLError) as exc:
        fail(f"{pair_label}: cannot read package.yaml: {exc}")
    if not isinstance(package, dict):
        fail(f"{pair_label}: package.yaml is not a YAML mapping")
    family = require(package, "entity_family", f"{pair_label} package.yaml")
    scenario_version = require(package, "scenario_version", f"{pair_label} package.yaml")

    lc = read_one_line_json(lifecycle_path, pair_label)
    ev = read_one_line_json(evaluation_path, pair_label)

    lc_identity = lc.get("identity") or {}
    if str(lc_identity.get("scenario_id")) != scenario_id:
        fail(
            f"{pair_label}: lifecycle.jsonl identity.scenario_id "
            f"{lc_identity.get('scenario_id')!r} disagrees with retention path"
        )

    outcome = require(lc, "outcome", f"{pair_label} lifecycle.jsonl")
    eligibility = require(ev, "eligibility", f"{pair_label} evaluation.jsonl")
    for key in ("aggregate_eligible", "publication_eligible", "invalidity_reasons"):
        if key not in eligibility:
            fail(f"{pair_label}: evaluation.jsonl eligibility.{key} missing")

    try:
        with open(manifest_path, encoding="utf-8") as fh:
            manifest = json.load(fh)
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"{pair_label}: manifest.json is not valid JSON: {exc}")
    artifacts_manifest = manifest.get("artifacts") or {}
    source_digests = {}
    for name, entry in sorted(artifacts_manifest.items()):
        if isinstance(entry, dict) and entry.get("sha256"):
            source_digests[name] = entry["sha256"]
    manifest_digest_entries.append(
        {"scenario_id": scenario_id, "rep": rep, "manifest_sha256": sha256_file(manifest_path)}
    )

    try:
        rep_int = int(rep)
    except ValueError:
        fail(f"{pair_label}: retention directory rep {rep!r} is not an integer")

    scenarios_list.append(
        {
            "scenario_id": scenario_id,
            "scenario_version": scenario_version,
            "family": family,
            "rep": rep_int,
            "retention_path": f"scenarios/{scenario_id}/{rep}",
            "eligibility": eligibility,
            "outcome": outcome,
            "source_digests": source_digests,
        }
    )

    # -----------------------------------------------------------------
    # /time, /cost -- verbatim I-08 totals (ADR-F10-04) reconciled
    # against per-cell sums computed here from retained I-07 stages.
    # -----------------------------------------------------------------
    metrics = ev.get("metrics") or {}

    def verbatim_metric(name):
        entry = metrics.get(name)
        if not isinstance(entry, dict) or not entry.get("available", False):
            fail(
                f"{pair_label}: evaluation.jsonl metrics.{name} unavailable -- "
                f"cannot build time/cost blocks (upstream contract gap, REQ-F-008)"
            )
        return entry["value"]

    pair_elapsed = verbatim_metric("elapsed_time")
    pair_cost = verbatim_metric("provider_cost")
    pair_rework_rollup = verbatim_metric("rework")
    if not is_number(pair_elapsed):
        fail(f"{pair_label}: metrics.elapsed_time.value is not numeric")
    if not is_number(pair_cost):
        fail(f"{pair_label}: metrics.provider_cost.value is not numeric")
    if not isinstance(pair_rework_rollup, int) or isinstance(pair_rework_rollup, bool):
        fail(f"{pair_label}: metrics.rework.value is not an integer")

    total_lifecycle_wall_seconds += pair_elapsed
    total_provider_cost_usd += pair_cost

    limits = lc.get("limits") or {}
    observed_cost = limits.get("observed_cost_usd")
    if not is_number(observed_cost):
        fail(f"{pair_label}: lifecycle.jsonl limits.observed_cost_usd is not numeric")
    total_observed_ceiling_cost += observed_cost

    stages = lc.get("stages")
    if not isinstance(stages, list):
        fail(f"{pair_label}: lifecycle.jsonl stages is not a list")

    pair_rework_stage_count = 0
    for idx, stage in enumerate(stages):
        if not isinstance(stage, dict):
            fail(f"{pair_label}: stages[{idx}] is not an object")
        category = stage.get("category")
        if category not in STAGE_CATEGORIES:
            fail(f"{pair_label}: stages[{idx}].category {category!r} not in closed stage_category vocabulary")
        rework = stage.get("rework")
        if not isinstance(rework, bool):
            # ADR-F10-08: rework MUST be read as retained, never inferred --
            # an absent/non-boolean flag is a schema violation, not a
            # silent default to False (TC-084 Negative Cases).
            fail(f"{pair_label}: stages[{idx}].rework must be a retained boolean, got {rework!r}")
        cost_usd = stage.get("cost_usd")
        if not is_number(cost_usd):
            fail(f"{pair_label}: stages[{idx}].cost_usd is not numeric")
        intervals = stage.get("intervals")
        if not isinstance(intervals, list):
            fail(f"{pair_label}: stages[{idx}].intervals is not a list")

        if category == "code" and rework:
            pair_rework_stage_count += 1

        for j, interval in enumerate(intervals):
            if not isinstance(interval, dict):
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] is not an object")
            icat = interval.get("category")
            if icat not in INTERVAL_CATEGORIES:
                fail(
                    f"{pair_label}: stages[{idx}].intervals[{j}].category {icat!r} "
                    f"not in closed interval_category vocabulary"
                )
            start = interval.get("start")
            end = interval.get("end")
            if not is_number(start) or not is_number(end):
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] start/end must be numeric")
            duration = end - start
            if duration < 0:
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] end precedes start")

            stage_time_totals[category] += duration
            interval_time_totals[icat] += duration
            share = share_for_cell(category, icat, rework)
            if share is not None:
                share_time_totals[share] += duration

        # Cost design decision 3 (see header): a stage's entire cost_usd
        # is attributed to (category, "provider_active"), never pro-rated
        # across its wait/tool/unclassified intervals.
        stage_cost_totals[category] += cost_usd
        interval_cost_totals["provider_active"] += cost_usd
        cost_share = share_for_cell(category, "provider_active", rework)
        if cost_share is not None:
            share_cost_totals[cost_share] += cost_usd

    # REQ-F-011 / REQ-F-008: the rework share's contributing-stage count
    # must reconcile against I-08's metrics.rework rollup for THIS pair.
    # A mismatch is an upstream contract defect -- reported loudly, never
    # silently resolved by trusting either side (test-plan.md TC-084
    # Negative Cases).
    if pair_rework_stage_count != pair_rework_rollup:
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-008): {pair_label}: "
            f"rework stage count computed from retained I-07 stages "
            f"({pair_rework_stage_count}) does not match I-08 metrics.rework rollup "
            f"({pair_rework_rollup}); reported, not resolved locally",
            file=sys.stderr,
        )

scenarios_list.sort(key=lambda s: (s["scenario_id"], s["rep"]))
manifest_digest_entries.sort(key=lambda e: (e["scenario_id"], e["rep"]))


def build_partition(totals, categories, total_value):
    block = {cat: round(totals.get(cat, 0.0), 6) for cat in categories}
    matched = sum(totals.get(cat, 0.0) for cat in categories)
    block[UNATTRIBUTED] = round(total_value - matched, 6)
    return block


time_block = {
    "lifecycle_wall_seconds": round(total_lifecycle_wall_seconds, 6),
    "stage_category": build_partition(stage_time_totals, STAGE_CATEGORIES, total_lifecycle_wall_seconds),
    "interval_category": build_partition(interval_time_totals, INTERVAL_CATEGORIES, total_lifecycle_wall_seconds),
    "share_partition": build_partition(share_time_totals, SHARE_CELLS, total_lifecycle_wall_seconds),
}

cost_block = {
    "stage_category": build_partition(stage_cost_totals, STAGE_CATEGORIES, total_provider_cost_usd),
    "interval_category": build_partition(interval_cost_totals, INTERVAL_CATEGORIES, total_provider_cost_usd),
    "share_partition": build_partition(share_cost_totals, SHARE_CELLS, total_provider_cost_usd),
    "ceiling_consumption": {
        "observed_cost_usd": round(total_observed_ceiling_cost, 6),
        "max_cost_usd": ceilings["max_cost_usd"],
    },
}

# /identity.retention_root_digest -- digest of digests (design decision 5).
retention_root_digest = sha256_bytes(
    json.dumps(
        {"batch_json_sha256": sha256_bytes(batch_bytes), "pairs": manifest_digest_entries},
        sort_keys=True,
        separators=(",", ":"),
    ).encode("utf-8")
)

identity = {
    "schema_version": SCHEMA_VERSION,
    "batch_id": batch_id,
    "phase": PHASE_LABEL,
    "retention_root_digest": retention_root_digest,
    "batch_policy_digest": batch_policy_digest,
    "ceilings": ceilings,
    "acknowledgement_ref": acknowledgement_ref,
    "min_reps": min_reps,
}

aggregate = {
    "identity": identity,
    "scenarios": scenarios_list,
    "time": time_block,
    "cost": cost_block,
}

print(json.dumps(aggregate, indent=2, sort_keys=True))
PYEOF
