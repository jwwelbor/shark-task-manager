#!/usr/bin/env bash
# aggregate-runs.sh --root <artifact_root> [--variant <id>]
#
# T-E40-F03-003/004 (spec.md ADR-F03-01, ADR-F03-03, ADR-F03-04, ADR-F03-06;
# test-plan.md TC-018). A pure function of an artifact root (ADR-F03-01):
# prints one JSON document to stdout, diagnostics to stderr, never touches
# anything but the record.jsonl files the pinned glob below matches
# (REQ-F-007 -- no batch-log.jsonl, no scratch project, no database, no
# network).
#
# T-E40-F03-003's slice (unchanged by this task): pinned-glob enumeration,
# per-record structural validation, classification (complete/
# explained_absence/anomaly), and five-field provenance uniformity
# (REQ-F-011) -- `schema_version`, `provenance`, `inventory`, `outcomes`,
# `anomalies[]`.
#
# T-E40-F03-004's extension (this task): the metric registry (Class A/B/C),
# per-(item, metric) statistics and ADR-F03-04's acceptance-interval
# formula, per-step token/cost/duration keying (REQ-F-013), the locked
# regression signal (REQ-F-014, ADR-F03-06), `rejections.by_gate` zero-vs-
# excluded handling (REQ-F-015), the `insufficient_reps`/`unusable`/
# `non_discriminative` flags (REQ-F-016..018), the computed `input_digest`
# (REQ-F-019), `baseline_id`, and the corpus-level rollup -- the `tasks[]`/
# `corpus`/`flags`/`baseline_id` blocks.
#
# Record classification (spec.md "Record classification", verbatim):
#   complete           -- every one of oracle/quality/loc is present (the
#                          three post-run "observational" families -- the
#                          only ones whose presence is genuinely ambiguous;
#                          runresult/stages/rejections/timing are
#                          deterministically tied to `outcome` and never
#                          need an explanation).
#   explained_absence  -- one or more of oracle/quality/loc is absent AND
#                          the record itself explains it: outcome ==
#                          "timeout" (no post-run phase ever ran); or
#                          quality.toolchain_guard != "pass" (the guard
#                          aborted the whole post-run phase before
#                          oracle/loc ran, but quality itself IS present,
#                          carrying only toolchain_guard); or errors[]
#                          carries a postrun_check_aborted entry.
#   anomaly            -- one or more of oracle/quality/loc is absent with
#                          NO explanation on the record (the F-4 case,
#                          TD-079) -- any anomaly makes this script exit
#                          non-zero (REQ-F-009).
#
# Family presence for classification and for `inventory[].families_present`
# is read from the family BLOCK ITSELF, never from `sources` (REQ-F-007 2nd
# sentence, TD-076's consumer-side consequence, test-plan.md TC-018q) --
# `sources` is never inspected by this script at all.
#
# Per-metric exclusion reasons (closed vocabulary, documented in
# bench/README.md's "Baseline aggregation, noise band, and replay"
# section this task opens):
#   outcome_timeout       -- outcome == "timeout": excluded from EVERY
#                             registry metric applicable to the record's
#                             item_type, whether or not the record could
#                             structurally have carried a value (a
#                             timeout's timing.harness_wall_ns is the cap,
#                             not a measurement -- spec.md "Exclusion
#                             rules").
#   toolchain_guard_abort -- the record is explained_absence via
#                             quality.toolchain_guard != "pass", and this
#                             metric's own source field is individually
#                             absent (which may be true even for a metric
#                             inside the "quality" family itself, e.g.
#                             quality.fmt_clean, since toolchain_guard
#                             aborts every OTHER quality sub-field too).
#   postrun_aborted        -- same as above, but explained via an
#                             errors[].kind == "postrun_check_aborted"
#                             entry instead of the guard.
#   unexplained_absence    -- the record is classified `anomaly`: EVERY
#                             oracle/quality/loc metric is excluded with
#                             this reason, regardless of whether this
#                             particular metric's own field happens to be
#                             present on the record (spec.md "anomaly ...
#                             excluded from every post-run band").
#   gate_not_executed       -- this metric's own source field EXISTS on the
#                             record but holds JSON null -- bench/README.md
#                             documents quality.fmt_clean/.vet_ok/
#                             .tests_pass as bool|null, "null means the
#                             gate could not be executed -- never a silent
#                             pass". Reading a null as a real boolean/
#                             numeric measurement (e.g. Python's
#                             bool(None) == False) would be exactly the
#                             fabrication REQ-N-005 forbids, so a null
#                             leaf is EXCLUDED, never coerced.
#   partial_usage           -- a Class C sum over stages[].usage.* (or a
#                             step.<status>.tokens_input/.tokens_output/
#                             .cost_usd metric) where at least one
#                             contributing spawn_agent stage exists for
#                             this run but is missing `usage` or the
#                             specific sub-field -- never summed over a
#                             subset (spec.md "Data model changes").
#   step_not_reached        -- a step.<status>.* metric where this run's
#                             stages[] contains no entry at all with that
#                             status (the run never passed through this
#                             workflow step) -- distinct from "the step
#                             ran an auto-advance stage with no agent
#                             usage", which is a true zero, not an
#                             exclusion (no by_gate-style zero-fill is
#                             licensed for a MISSING step the way it is
#                             licensed for an omitted-but-present by_gate
#                             key, REQ-F-015's own text: "the producer
#                             defines rejections.rework_loops as the sum
#                             of by_gate's values" -- nothing licenses a
#                             step-zero the same way).
#   family_absent            -- catch-all: this metric's source field (or
#                             its whole containing family/block, e.g. a
#                             wholly-absent `rejections` block) is simply
#                             absent from the record, and none of the
#                             above more-specific reasons applies. Applies
#                             to families outside classification's own
#                             oracle/quality/loc set (rejections/
#                             runresult/timing/stages), and to the rare
#                             case of a `complete`-classified record whose
#                             family block is present but one specific
#                             leaf sub-field is missing anyway.
#
# rejections.by_gate's OWN zero-vs-excluded rule is distinct from the
# above and is NOT an exclusion at all: an omitted gate key within a
# PRESENT by_gate object counts as 0 (still `n`-counted, REQ-F-015); only
# a wholly ABSENT `rejections` block excludes the record from every
# rejections_* metric (reason `family_absent`, unless outcome_timeout
# already applies).
#
# Band and acceptance interval (spec.md "Band and acceptance interval",
# ADR-F03-04, verbatim formulas):
#   Class A (binary)     -- n, true_count, rate; accept_set = the set of
#                            booleans observed.
#   Class B (int count)  -- n, min, median, max, mean, spread_abs,
#                            spread_rel; accept interval
#                            [max(0, min-1), max+1], unconditionally (every
#                            registered Class B metric is non-negative).
#   Class C (continuous) -- same observed statistics; r = max-min,
#                            r_eff = r when r>0, else 0.10*abs(median), else
#                            (median==0 too) exactly 0; interval
#                            [min-r_eff, max+r_eff], lower-clamped at 0.
# spread_rel = spread_abs/median, or null when median==0. n<2 publishes no
# interval/accept_set and sets `insufficient_reps` (REQ-F-016). A metric
# whose spread_abs strictly exceeds its mean is flagged `unusable`
# (REQ-F-018, strict >).
#
# Provenance uniformity (REQ-F-011): the five manifest fields (model_ids,
# fixture_base_sha, variant_bundle_sha256, corpus_schema_version,
# shark_version) are compared across every contributing record's PRESENT
# value only -- a record where the field is legitimately absent (e.g.
# manifest.model_ids on a record whose timeout fired before any stage
# resolved modelUsage, bench/README.md "I-02 record schema field
# reference") never counts as a divergence by itself; two or more DISTINCT
# present values for the same field does. Non-uniform provenance still
# prints the full document (provenance.uniform=false, divergences[] naming
# both differing values, plus classification/inventory/outcomes/anomalies,
# which are independent of provenance) and exits non-zero -- but `tasks[]`,
# `corpus`, `flags`, and `baseline_id` are OMITTED entirely: this is the
# STATISTICAL half of the document (REQ-F-011, AC-11: "no other block
# (tasks[], corpus) is populated as if the batch were valid"). `input_digest`
# still publishes even when non-uniform -- it identifies the exact
# (possibly-invalid) input set, which is the one thing a non-uniform batch
# report still needs to name precisely.
#
# `manifest.variant_id` is NOT one of REQ-F-011's five uniformity fields
# (a batch is expected to be single-variant per REQ-F-001; --variant exists
# to select one out of a mixed root), but silently averaging two variants'
# runs together under one item_id would be a fabricated result (REQ-N-005),
# so a root containing more than one distinct variant_id with no --variant
# filter to disambiguate is a hard, nothing-printed failure, same class as
# a structurally invalid record.
#
# Exit status (spec.md "Interface contracts"):
#   0        every record complete/explained_absence, provenance uniform
#   non-zero any anomaly (full document still printed, anomalies named);
#            OR a structurally invalid record (unsupported schema_version,
#            unparseable JSON, missing manifest/outcome), OR more than one
#            distinct variant_id with no --variant filter -- prints
#            NOTHING to stdout, names the offending file/values on stderr
#            (REQ-F-010); OR non-uniform provenance (full document still
#            printed minus tasks/corpus/flags/baseline_id, see above)
#
# Record enumeration is PINNED to the bash glob below, never `find` -- a
# bare `*` component never matches a leading-dot path component, so the
# dot-prefixed `.incomplete/` quarantine root (run-batch.sh's own
# ADR-F03-07 quarantine path) can never contribute, by the glob's own
# semantics, not by a runtime skip-rule (test-plan.md TC-018s).
#
# REQ-N-001: bash entry point, embedded python3 for JSON work,
# machine-readable JSON to stdout, diagnostics to stderr.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	cat >&2 <<'EOF'
usage: aggregate-runs.sh --root <artifact_root> [--variant <id>]
EOF
	exit 2
}

root=""
variant_filter=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--root)
		[[ $# -ge 2 ]] || usage
		root="$2"
		shift 2
		;;
	--variant)
		[[ $# -ge 2 ]] || usage
		variant_filter="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$root" ]] || usage
[[ -d "$root" ]] || {
	echo "aggregate-runs: artifact root not found: $root" >&2
	exit 1
}
command -v python3 >/dev/null 2>&1 || {
	echo "aggregate-runs: python3 not found on PATH" >&2
	exit 1
}

# --- pinned enumeration (see header comment; REQ-F-007, TC-018s) ---
shopt -s nullglob
record_files=("$root"/*/*/rep-*/record.jsonl)
shopt -u nullglob

python3 - "$root" "$variant_filter" "${record_files[@]}" <<'PYEOF'
import hashlib
import json
import os
import statistics
import sys

# This aggregate document's OWN schema version -- distinct from I-02's
# (spec.md "Aggregate document").
AGGREGATE_SCHEMA_VERSION = "1.0"

I02_SUPPORTED_SCHEMA_VERSIONS = {"1.0"}

# The three post-run "observational" families whose presence/absence is
# what record classification is about (spec.md "Record classification").
EXPECTED_FAMILIES = ("oracle", "quality", "loc")

# Every family block `inventory[].families_present` may name. Deliberately
# excludes structural/meta top-level fields (manifest, schema_version,
# outcome, errors, sources) -- those are not "metric families".
ALL_FAMILY_KEYS = (
    "loc",
    "oracle",
    "quality",
    "rejections",
    "runresult",
    "stages",
    "timeout_detail",
    "timing",
)

# REQ-F-011's five uniformity fields, all under `manifest`.
UNIFORMITY_FIELDS = (
    "model_ids",
    "fixture_base_sha",
    "variant_bundle_sha256",
    "corpus_schema_version",
    "shark_version",
)

ERROR_KIND_POSTRUN_ABORTED = "postrun_check_aborted"

# Metric registry (spec.md "Metric registry and classes"), the STATIC
# entries -- rejections_by_gate.<gate> and step.<status>.<kind> are
# generated dynamically per item from the gate/status keys actually
# observed (spec.md: "The gate key universe is the union of keys observed
# across contributing records").
STATIC_METRICS = [
    # Class A (binary)
    {"id": "oracle_f2p_resolved", "cls": "A", "family": "oracle", "path": "oracle.f2p_resolved", "item_types": ("task", "bug")},
    {"id": "oracle_repro_confirmed", "cls": "A", "family": "oracle", "path": "oracle.repro_confirmed", "item_types": ("bug",)},
    {"id": "quality_fmt_clean", "cls": "A", "family": "quality", "path": "quality.fmt_clean", "item_types": ("task", "bug")},
    {"id": "quality_vet_ok", "cls": "A", "family": "quality", "path": "quality.vet_ok", "item_types": ("task", "bug")},
    {"id": "quality_tests_pass", "cls": "A", "family": "quality", "path": "quality.tests_pass", "item_types": ("task", "bug")},
    # Class B (integer count)
    {"id": "p2p_regressions_count", "cls": "B", "family": "oracle", "path": "oracle.p2p_regressions_count", "item_types": ("task", "bug")},
    {"id": "p2p_removed_count", "cls": "B", "family": "oracle", "path": "oracle.removed_count", "item_types": ("task", "bug")},
    {"id": "lint_new_issues_count", "cls": "B", "family": "quality", "path": "quality.lint_new_issues_count", "item_types": ("task", "bug")},
    {"id": "rejections_rework_loops", "cls": "B", "family": "rejections", "path": "rejections.rework_loops", "item_types": ("task", "bug")},
    {"id": "loc_files_touched", "cls": "B", "family": "loc", "path": "loc.files_touched", "item_types": ("task", "bug")},
    # Class C (continuous)
    {"id": "loc_prod_added", "cls": "C", "family": "loc", "path": "loc.prod_added", "item_types": ("task", "bug")},
    {"id": "loc_prod_deleted", "cls": "C", "family": "loc", "path": "loc.prod_deleted", "item_types": ("task", "bug")},
    {"id": "loc_test_added", "cls": "C", "family": "loc", "path": "loc.test_added", "item_types": ("task", "bug")},
    {"id": "loc_test_deleted", "cls": "C", "family": "loc", "path": "loc.test_deleted", "item_types": ("task", "bug")},
    {"id": "wall_clock_ns", "cls": "C", "family": "timing", "path": "timing.harness_wall_ns", "item_types": ("task", "bug")},
    {"id": "run_total_duration_ns", "cls": "C", "family": "runresult", "path": "runresult.total_duration_ns", "item_types": ("task", "bug")},
    {"id": "tokens_input_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "input_tokens", "item_types": ("task", "bug")},
    {"id": "tokens_output_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "output_tokens", "item_types": ("task", "bug")},
    {"id": "cache_read_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "cache_read_input_tokens", "item_types": ("task", "bug")},
    {"id": "cache_creation_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "cache_creation_input_tokens", "item_types": ("task", "bug")},
    {"id": "cost_usd_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "total_cost_usd", "item_types": ("task", "bug")},
    {"id": "api_duration_ms_total", "cls": "C", "family": "stages", "kind": "sum_usage", "subfield": "duration_api_ms", "item_types": ("task", "bug")},
]

STEP_USAGE_SUBFIELD = {
    "tokens_input": "input_tokens",
    "tokens_output": "output_tokens",
    "cost_usd": "total_cost_usd",
}


def fail(msg):
    """Hard-abort with NOTHING printed to stdout (REQ-F-010, REQ-N-005):
    only ever called before the final document is assembled, so a caller
    piping stdout to aggregate.json is left with a zero-byte/absent file,
    never a partial one."""
    print("aggregate-runs: " + msg, file=sys.stderr)
    sys.exit(1)


def load_and_validate(path):
    try:
        with open(path) as f:
            content = f.read()
    except OSError as e:
        fail("%s: cannot read: %s" % (path, e))

    lines = [ln for ln in (raw.strip() for raw in content.splitlines()) if ln]
    if len(lines) != 1:
        fail("%s: expected exactly one JSON record line, found %d" % (path, len(lines)))

    try:
        rec = json.loads(lines[0])
    except json.JSONDecodeError as e:
        fail("%s: unparseable JSON: %s" % (path, e))

    if not isinstance(rec, dict):
        fail("%s: record is not a JSON object" % path)

    schema_version = rec.get("schema_version")
    if schema_version not in I02_SUPPORTED_SCHEMA_VERSIONS:
        fail(
            "%s: unsupported schema_version %r (supported: %s)"
            % (path, schema_version, sorted(I02_SUPPORTED_SCHEMA_VERSIONS))
        )

    manifest = rec.get("manifest")
    if not isinstance(manifest, dict):
        fail("%s: missing or invalid 'manifest' block" % path)

    if "outcome" not in rec:
        fail("%s: missing 'outcome' field" % path)

    if not manifest.get("run_key"):
        fail("%s: manifest.run_key missing or empty" % path)

    return rec


def explain_kind(rec):
    """The single reason a record's oracle/quality/loc absence -- or a
    timeout's blanket exclusion -- is EXPLAINED rather than anomalous
    (spec.md "Record classification", verbatim three conditions), returned
    as the exact exclusion-reason string this script publishes. None means
    "no explanation on the record" (classify() then calls it `anomaly` if
    a family is actually missing)."""
    if rec.get("outcome") == "timeout":
        return "outcome_timeout"
    quality = rec.get("quality")
    if isinstance(quality, dict) and quality.get("toolchain_guard") not in (None, "pass"):
        return "toolchain_guard_abort"
    for err in rec.get("errors") or []:
        if isinstance(err, dict) and err.get("kind") == ERROR_KIND_POSTRUN_ABORTED:
            return "postrun_aborted"
    return None


def classify(rec):
    missing = [f for f in EXPECTED_FAMILIES if f not in rec]
    if not missing:
        return "complete", missing
    if explain_kind(rec) is not None:
        return "explained_absence", missing
    return "anomaly", missing


def get_leaf(rec, path):
    """Generic dotted-path getter. Returns (found, value). A JSON `null`
    counts as NOT found: bench/README.md documents `quality.fmt_clean`/
    `.vet_ok`/`.tests_pass` as `bool | null` where "null means the gate
    could not be executed -- never a silent pass" -- reading `null` as the
    Python `None` and then coercing it into a boolean/numeric statistic
    (e.g. `bool(None) == False`) would be exactly the fabrication
    REQ-N-005 forbids: a real measurement of "the gate failed" where the
    record says "no measurement exists". No registered metric is ever
    legitimately null, so this guard is generic, not quality-specific."""
    node = rec
    for part in path.split("."):
        if not isinstance(node, dict) or part not in node:
            return False, None
        node = node[part]
    if node is None:
        return False, None
    return True, node


def leaf_is_present_but_null(rec, path):
    """True when the dotted path resolves to a key that EXISTS and holds
    JSON null -- distinguishes "the gate ran and produced no measurement"
    (a real, documented producer state) from "the field was never written
    at all" (a different, less specific exclusion reason)."""
    node = rec
    for part in path.split("."):
        if not isinstance(node, dict) or part not in node:
            return False
        node = node[part]
    return node is None


def agent_stages(rec):
    """stages[] entries whose action is spawn_agent -- the only stages
    that ever carry `usage` (an advance_status stage structurally never
    does, README "I-02 record schema field reference"). Returns None when
    `stages` itself isn't a list (family absent)."""
    stages = rec.get("stages")
    if not isinstance(stages, list):
        return None
    return [s for s in stages if isinstance(s, dict) and s.get("action") == "spawn_agent"]


def sum_usage_field(rec, subfield):
    """Class C whole-run sum over stages[].usage.<subfield> (spec.md "Data
    model changes": only when every agent stage contributed that
    sub-field; never summed over a subset). A `null` sub-field value
    counts the same as a missing one (partial_usage) -- never summed as
    zero."""
    agents = agent_stages(rec)
    if agents is None:
        return ("excluded", "family_absent")
    if not agents:
        # No spawn_agent stage in this run at all -- a true, not a
        # fabricated, zero (there is nothing to sum).
        return ("ok", 0)
    total = 0
    for s in agents:
        usage = s.get("usage")
        if not isinstance(usage, dict) or usage.get(subfield) is None:
            return ("excluded", "partial_usage")
        total += usage[subfield]
    return ("ok", total)


def step_metric(rec, status, step_kind):
    """Per-step Class C metric, keyed on stages[].status with repeat
    occurrences summed (REQ-F-013). `duration_ns` sums stage-level
    duration_ns over EVERY stage at that status (present on every stage,
    agent or not); the usage-derived kinds sum only the spawn_agent
    stages at that status, and are a true zero (not excluded) when the
    step was reached only via a non-agent (e.g. advance_status) stage."""
    stages = rec.get("stages")
    if not isinstance(stages, list):
        return ("excluded", "family_absent")
    matching = [s for s in stages if isinstance(s, dict) and s.get("status") == status]
    if not matching:
        return ("excluded", "step_not_reached")

    if step_kind == "duration_ns":
        total = 0
        for s in matching:
            if s.get("duration_ns") is None:
                return ("excluded", "partial_usage")
            total += s["duration_ns"]
        return ("ok", total)

    subfield = STEP_USAGE_SUBFIELD[step_kind]
    agent_matching = [s for s in matching if s.get("action") == "spawn_agent"]
    if not agent_matching:
        return ("ok", 0)
    total = 0
    for s in agent_matching:
        usage = s.get("usage")
        if not isinstance(usage, dict) or usage.get(subfield) is None:
            return ("excluded", "partial_usage")
        total += usage[subfield]
    return ("ok", total)


def evaluate_metric(rec, m, classification):
    """Returns ("ok", value) or ("excluded", reason) for one (record,
    metric) pair. `classification` is the record's already-computed
    classify() result (reused, never recomputed per metric)."""
    ek = explain_kind(rec)
    if ek == "outcome_timeout":
        return ("excluded", "outcome_timeout")

    kind = m.get("kind")
    if kind == "by_gate":
        rj = rec.get("rejections")
        bg = rj.get("by_gate") if isinstance(rj, dict) else None
        if not isinstance(bg, dict):
            return ("excluded", "family_absent")
        value = bg.get(m["gate"], 0)
        if value is None:
            return ("excluded", "gate_not_executed")
        # REQ-F-015: an omitted gate key within a PRESENT by_gate object
        # counts as 0 -- included, not excluded.
        return ("ok", value)
    if kind == "sum_usage":
        return sum_usage_field(rec, m["subfield"])
    if kind == "step":
        return step_metric(rec, m["status"], m["step_kind"])

    found, val = get_leaf(rec, m["path"])
    family = m["family"]

    if family in ("oracle", "quality", "loc"):
        if classification == "anomaly":
            # spec.md: "anomaly -- excluded from every post-run band",
            # unconditionally -- even a family this specific record
            # happens to carry (TC-018c's partial-anomaly negative case).
            return ("excluded", "unexplained_absence")
        if found:
            # explained_absence "still contributes to every family it
            # does carry" (spec.md); complete always finds it.
            return ("ok", val)
        if leaf_is_present_but_null(rec, m["path"]):
            # README: quality.fmt_clean/.vet_ok/.tests_pass are bool|null,
            # "null means the gate could not be executed -- never a
            # silent pass". Preferred over the guard/postrun/family_absent
            # reasons below: a null value is a MORE specific signal than
            # "the field wasn't written at all".
            return ("excluded", "gate_not_executed")
        if ek in ("toolchain_guard_abort", "postrun_aborted"):
            return ("excluded", ek)
        return ("excluded", "family_absent")

    if found:
        return ("ok", val)
    if leaf_is_present_but_null(rec, m["path"]):
        return ("excluded", "gate_not_executed")
    return ("excluded", "family_absent")


def compute_stats(cls, values):
    """ADR-F03-04's band + acceptance-interval formula, per class. `values`
    is the list of "ok" contributions only (excluded reps never enter)."""
    n = len(values)

    if cls == "A":
        block = {"n": n}
        if n == 0:
            block["insufficient_reps"] = True
            return block
        true_count = sum(1 for v in values if v is True)
        block["true_count"] = true_count
        block["rate"] = true_count / n
        if n < 2:
            block["insufficient_reps"] = True
        else:
            block["accept_set"] = sorted({bool(v) for v in values})
        return block

    if n == 0:
        return {"n": 0, "insufficient_reps": True}

    mn, mx = min(values), max(values)
    median = statistics.median(values)
    mean = statistics.mean(values)
    spread_abs = mx - mn
    spread_rel = (spread_abs / median) if median != 0 else None

    block = {
        "n": n,
        "min": mn,
        "median": median,
        "max": mx,
        "mean": mean,
        "spread_abs": spread_abs,
        "spread_rel": spread_rel,
    }

    if n < 2:
        block["insufficient_reps"] = True
        return block

    if cls == "B":
        # Band table: [max(0, min-1), max+1], unconditionally -- every
        # registered Class B metric is a non-negative count.
        accept_lo = max(0, mn - 1)
        accept_hi = mx + 1
    else:
        r = spread_abs
        if r > 0:
            r_eff = r
        elif median != 0:
            r_eff = 0.10 * abs(median)
        else:
            # Identically zero across every rep -- the documented exact
            # rule (spec.md), not a coincidence of the 0.10*median formula.
            r_eff = 0
        accept_lo = max(0, mn - r_eff)
        accept_hi = mx + r_eff

    block["accept_lo"] = accept_lo
    block["accept_hi"] = accept_hi

    if spread_abs > mean:
        block["unusable"] = True

    return block


root, variant_filter = sys.argv[1], sys.argv[2]
# Explicit, locale-independent (Python's default str ordering is codepoint-
# based, never consults LC_COLLATE) re-sort of whatever order the bash glob
# handed us -- REQ-N-004/AC-06: output must not depend on the glob's own
# (potentially locale-influenced) enumeration order.
record_paths = sorted(sys.argv[3:])

if not record_paths:
    fail(
        "no record.jsonl files found under %r via the pinned glob "
        "<root>/*/*/rep-*/record.jsonl" % root
    )

records = []
for path in record_paths:
    rec = load_and_validate(path)
    if variant_filter and rec["manifest"].get("variant_id") != variant_filter:
        continue
    records.append((path, rec))

if not records:
    fail("no record.jsonl files matched --variant %r under %r" % (variant_filter, root))

# A root spanning more than one variant_id with no --variant filter to
# disambiguate would silently blend two variants' runs under one item_id
# -- a fabricated result (REQ-N-005), so this is a hard, nothing-printed
# failure, same class as a structurally invalid record.
observed_variant_ids = sorted({rec["manifest"].get("variant_id") for _, rec in records if rec["manifest"].get("variant_id")})
if len(observed_variant_ids) > 1:
    fail(
        "records span multiple variant_ids %r under one aggregation -- "
        "pass --variant to select one" % observed_variant_ids
    )
variant_id = observed_variant_ids[0] if observed_variant_ids else None

# --- classification + inventory + outcomes + anomalies ---
inventory = {}
outcome_counts = {}
anomalies = []
anomaly_count = 0
timeout_count = 0
seen_run_keys = {}
classification_by_run_key = {}

for path, rec in records:
    run_key = rec["manifest"]["run_key"]
    if run_key in seen_run_keys:
        fail("duplicate manifest.run_key %r across %s and %s" % (run_key, seen_run_keys[run_key], path))
    seen_run_keys[run_key] = path

    classification, missing = classify(rec)
    classification_by_run_key[run_key] = classification
    outcome = rec["outcome"]

    inventory[run_key] = {
        "classification": classification,
        "outcome": outcome,
        "families_present": sorted(f for f in ALL_FAMILY_KEYS if f in rec),
    }

    outcome_counts[outcome] = outcome_counts.get(outcome, 0) + 1
    if outcome == "timeout":
        timeout_count += 1

    if classification == "anomaly":
        anomaly_count += 1
        anomalies.append({"run_key": run_key, "missing_families": sorted(missing)})

anomalies.sort(key=lambda a: a["run_key"])

total = len(records)
outcomes_block = {
    "counts": outcome_counts,
    "timeout_rate": (timeout_count / total) if total else 0,
    "anomaly_count": anomaly_count,
}

# --- provenance uniformity (REQ-F-011) ---
# field -> {json_repr_of_value: (value, [run_key, ...])} -- built only from
# records where the field is PRESENT; a record where it's legitimately
# absent (e.g. model_ids on a timeout record) never participates, so it
# can never manufacture a spurious divergence.
field_values = {f: {} for f in UNIFORMITY_FIELDS}
for _, rec in records:
    manifest = rec["manifest"]
    run_key = manifest["run_key"]
    for field in UNIFORMITY_FIELDS:
        if field not in manifest:
            continue
        value = manifest[field]
        key = json.dumps(value, sort_keys=True)
        if key not in field_values[field]:
            field_values[field][key] = (value, [])
        field_values[field][key][1].append(run_key)

provenance = {}
divergences = []
for field in UNIFORMITY_FIELDS:
    distinct = field_values[field]
    if not distinct:
        continue  # never observed anywhere -- never fabricated (REQ-N-005)
    if len(distinct) == 1:
        value, _run_keys = next(iter(distinct.values()))
        provenance[field] = value
    else:
        divergences.append(
            {
                "field": field,
                "values": sorted(
                    ({"value": value, "run_keys": sorted(run_keys)} for value, run_keys in distinct.values()),
                    key=lambda entry: json.dumps(entry["value"], sort_keys=True),
                ),
            }
        )

uniform = not divergences
provenance["uniform"] = uniform
if divergences:
    provenance["divergences"] = divergences

distinct_reps = sorted({rec["manifest"]["rep"] for _, rec in records if "rep" in rec["manifest"]})
provenance["reps"] = len(distinct_reps)

# --- input_digest (REQ-F-019): sha256 over sorted "<sha256>  <relpath>"
# lines of EVERY contributing record.jsonl -- the whole read/validated
# set, independent of any metric's own excluded[]/band membership. Ties a
# published report to the exact artifact set it was computed from, so the
# digest must be order-invariant (sorted lines) and rooted at `--root`
# (relpath, not the absolute mktemp/checkout path) so the same artifact
# set aggregated from two different locations digests identically.
digest_lines = []
for path, _rec in records:
    with open(path, "rb") as fbin:
        file_sha = hashlib.sha256(fbin.read()).hexdigest()
    relpath = os.path.relpath(path, root)
    digest_lines.append("%s  %s" % (file_sha, relpath))
digest_lines.sort()
input_digest = hashlib.sha256(("\n".join(digest_lines) + "\n").encode("utf-8")).hexdigest()

aggregate = {
    "schema_version": AGGREGATE_SCHEMA_VERSION,
    "input_digest": input_digest,
    "provenance": provenance,
    "inventory": inventory,
    "outcomes": outcomes_block,
    "anomalies": anomalies,
}

# --- per-item metrics, corpus rollup, flags, baseline_id -----------------
# STATISTICAL blocks only -- computed, and published, when provenance is
# uniform (AC-11: "never published as a result" over an unpinned batch).
if uniform:
    item_ids = sorted({rec["manifest"]["item_id"] for _, rec in records})
    item_type_by_item = {}
    for _, rec in records:
        iid = rec["manifest"]["item_id"]
        item_type_by_item.setdefault(iid, rec["manifest"].get("item_type", "task"))

    tasks_out = []
    flags_unusable = []
    flags_insufficient = []
    flags_non_discriminative = []
    spread_rel_by_metric = {}
    f2p_total_true = 0
    f2p_total_n = 0
    f2p_by_rep = {}

    for item_id in item_ids:
        item_type = item_type_by_item[item_id]
        item_records = [(p, r) for p, r in records if r["manifest"]["item_id"] == item_id]

        applicable = [m for m in STATIC_METRICS if item_type in m["item_types"]]

        gates = set()
        steps = set()
        for _, rec in item_records:
            rj = rec.get("rejections")
            if isinstance(rj, dict) and isinstance(rj.get("by_gate"), dict):
                gates.update(rj["by_gate"].keys())
            st = rec.get("stages")
            if isinstance(st, list):
                for s in st:
                    if isinstance(s, dict) and "status" in s:
                        steps.add(s["status"])

        metric_defs = list(applicable)
        for gate in sorted(gates):
            metric_defs.append({"id": "rejections_by_gate.%s" % gate, "cls": "B", "family": "rejections", "kind": "by_gate", "gate": gate})
        for status in sorted(steps):
            for step_kind in ("duration_ns", "tokens_input", "tokens_output", "cost_usd"):
                metric_defs.append(
                    {
                        "id": "step.%s.%s" % (status, step_kind),
                        "cls": "C",
                        "family": "stages",
                        "kind": "step",
                        "status": status,
                        "step_kind": step_kind,
                    }
                )

        metrics_block = {}
        for m in metric_defs:
            values = []
            per_rep_ok = []
            excluded_entries = []
            for path, rec in item_records:
                run_key = rec["manifest"]["run_key"]
                classification = classification_by_run_key[run_key]
                status, result = evaluate_metric(rec, m, classification)
                if status == "ok":
                    values.append(result)
                    per_rep_ok.append((rec["manifest"].get("rep"), result))
                else:
                    excluded_entries.append({"run_key": run_key, "reason": result})

            stat_block = compute_stats(m["cls"], values)
            stat_block["excluded"] = sorted(excluded_entries, key=lambda e: e["run_key"])
            metrics_block[m["id"]] = stat_block

            if stat_block.get("insufficient_reps"):
                flags_insufficient.append({"item_id": item_id, "metric": m["id"]})
            if stat_block.get("unusable"):
                flags_unusable.append({"item_id": item_id, "metric": m["id"]})
            if m["cls"] in ("B", "C") and stat_block.get("spread_rel") is not None:
                spread_rel_by_metric.setdefault(m["id"], []).append(stat_block["spread_rel"])

            if m["id"] == "oracle_f2p_resolved":
                for rep, val in per_rep_ok:
                    bucket = f2p_by_rep.setdefault(rep, [0, 0])
                    bucket[1] += 1
                    if val is True:
                        bucket[0] += 1

        f2p_block = metrics_block.get("oracle_f2p_resolved")
        non_discriminative = False
        # >=2 contributing reps required -- a single-rep "identical" set
        # is vacuous, not evidence of non-discrimination (REQ-F-017 is
        # about a REPEATED identical result).
        if f2p_block and f2p_block.get("n", 0) >= 2 and len(f2p_block.get("accept_set", [])) == 1:
            non_discriminative = True
            flags_non_discriminative.append(item_id)
        if f2p_block:
            f2p_total_true += f2p_block.get("true_count", 0)
            f2p_total_n += f2p_block.get("n", 0)

        tasks_out.append(
            {
                "item_id": item_id,
                "item_type": item_type,
                "metrics": metrics_block,
                "non_discriminative": non_discriminative,
            }
        )

    corpus = {}
    for metric_id, rels in sorted(spread_rel_by_metric.items()):
        corpus[metric_id] = {
            "spread_rel": {"min": min(rels), "median": statistics.median(rels), "max": max(rels)},
        }
    if f2p_total_n > 0:
        f2p_corpus = {"pass_rate": f2p_total_true / f2p_total_n}
        rep_rates = []
        for rep in sorted(f2p_by_rep, key=lambda r: (r is None, r)):
            true_n, total_n = f2p_by_rep[rep]
            if total_n > 0:
                rep_rates.append(true_n / total_n)
        if rep_rates:
            f2p_corpus["rep_slice_band"] = {
                "min": min(rep_rates),
                "median": statistics.median(rep_rates),
                "max": max(rep_rates),
            }
        corpus["oracle_f2p_resolved"] = f2p_corpus

    aggregate["tasks"] = tasks_out
    aggregate["corpus"] = corpus
    aggregate["flags"] = {
        "unusable_metrics": sorted(flags_unusable, key=lambda e: (e["item_id"], e["metric"])),
        "insufficient_reps": sorted(flags_insufficient, key=lambda e: (e["item_id"], e["metric"])),
        "non_discriminative_tasks": sorted(flags_non_discriminative),
    }

    reps = provenance.get("reps", 0)
    fixture_base_sha = provenance.get("fixture_base_sha")
    if variant_id and fixture_base_sha:
        aggregate["baseline_id"] = "%s-%s-r%d" % (variant_id, fixture_base_sha[:12], reps)

# The one and only stdout write (REQ-N-004: sorted keys, fixed list order,
# no timestamp -- json.dumps(sort_keys=True) sorts every nested dict's keys
# too, and every list built above was explicitly sorted).
print(json.dumps(aggregate, indent=2, sort_keys=True))

if not uniform:
    sys.exit(1)
if anomaly_count > 0:
    sys.exit(1)
sys.exit(0)
PYEOF
