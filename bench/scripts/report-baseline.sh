#!/usr/bin/env bash
# report-baseline.sh --aggregate <aggregate.json>
#
# T-E40-F03-005 (spec.md ADR-F03-01, REQ-F-020..024, REQ-N-004; test-plan.md
# TC-018n/o/p/u). A pure function of one `aggregate.json` document (ADR-F03-01):
# reads exactly the file named by `--aggregate`, consults no clock, invokes
# no subprocess, and prints one markdown document to stdout, diagnostics to
# stderr (REQ-N-001). Every value in the report traces to a field already
# present in the aggregate (REQ-F-020) -- this script computes no new
# statistic; a missing caveat/provenance field in the aggregate is
# T-E40-F03-004's bug, never patched around here.
#
# Sections emitted, each keyed to its requirement:
#   Provenance             -- REQ-F-022: model_ids/fixture_base_sha/
#                              variant_bundle_sha256/corpus_schema_version/
#                              shark_version/reps/input_digest, reproduced
#                              verbatim from `aggregate.json`'s `provenance`
#                              block (plus `baseline_id`/`input_digest` from
#                              the document root).
#   Noise band per task    -- REQ-F-021: per metric, the observed min/
#                              median/max/spread AND the acceptance interval
#                              (or accept_set) AND a printed sentence naming
#                              which of ADR-F03-04's derivation rules applied
#                              -- never means alone.
#   Corpus feedback to     -- REQ-F-023: heading names "E40-F01" verbatim,
#     E40-F01                lists `flags.non_discriminative_tasks[]`.
#   Measurement caveats    -- REQ-F-024: the four caveats, always emitted
#                              (standing methodology notes, not derived from
#                              this particular aggregate's data): the
#                              timeout-exclusion rule (prose, no tracked
#                              item), TD-079 (anomaly bucket), TD-081 (fmt/
#                              vet attribution), T-004 (tests_pass is not a
#                              regression signal).
#
# Per-metric derivation-rule sentence and Class A/B/C selection: the metric
# id -> class mapping below is copied verbatim from spec.md "Metric registry
# and classes" (the shape source both aggregate-runs.sh and this script draw
# on) -- `aggregate.json` itself carries no `cls` field per metric, so this
# is the one piece of registry knowledge this script must also hold to pick
# the right rule sentence; it never re-derives any statistic from it. Class
# B's rule is a fixed formula printed unconditionally; Class C has three
# branches (ADR-F03-04) selected by reading the metric's own already-present
# `spread_abs`/`median` fields -- not by recomputing them.
#
# Exit status (spec.md "Interface contracts"):
#   0        report printed to stdout
#   non-zero the aggregate file could not be read, its JSON was unparseable,
#            or its `schema_version` is unsupported -- nothing is printed to
#            stdout in this case (the whole input is validated before any
#            output line is built, so a caller redirecting stdout to a file
#            is never left with a partial/truncated report, matching
#            REQ-F-020's "pure function" framing and aggregate-runs.sh's own
#            single-stdout-write convention).
set -euo pipefail

usage() {
	echo "usage: report-baseline.sh --aggregate <aggregate.json>" >&2
	exit 2
}

aggregate_path=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--aggregate)
		[[ $# -ge 2 ]] || usage
		aggregate_path="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$aggregate_path" ]] || usage
command -v python3 >/dev/null 2>&1 || {
	echo "report-baseline: python3 not found on PATH" >&2
	exit 1
}

python3 - "$aggregate_path" <<'PYEOF'
import json
import sys

AGGREGATE_SCHEMA_VERSIONS_SUPPORTED = {"1.0"}

# spec.md "Metric registry and classes" -- shape source, copied verbatim
# (classification only; no statistic here is re-derived from it).
STATIC_METRIC_CLASS = {
    "oracle_f2p_resolved": "A",
    "oracle_repro_confirmed": "A",
    "quality_fmt_clean": "A",
    "quality_vet_ok": "A",
    "quality_tests_pass": "A",
    "p2p_regressions_count": "B",
    "p2p_removed_count": "B",
    "lint_new_issues_count": "B",
    "rejections_rework_loops": "B",
    "loc_files_touched": "B",
    "loc_prod_added": "C",
    "loc_prod_deleted": "C",
    "loc_test_added": "C",
    "loc_test_deleted": "C",
    "wall_clock_ns": "C",
    "run_total_duration_ns": "C",
    "tokens_input_total": "C",
    "tokens_output_total": "C",
    "cache_read_total": "C",
    "cache_creation_total": "C",
    "cost_usd_total": "C",
    "api_duration_ms_total": "C",
}


def metric_class(metric_id):
    if metric_id in STATIC_METRIC_CLASS:
        return STATIC_METRIC_CLASS[metric_id]
    if metric_id.startswith("rejections_by_gate."):
        return "B"
    if metric_id.startswith("step."):
        return "C"
    return None


def fail(msg):
    """Hard-abort with NOTHING printed to stdout: only ever called before
    the report's line list is assembled and joined, so a caller piping
    stdout to report.md is left with a zero-byte/absent file, never a
    partial one (see header, TC-018u)."""
    print("report-baseline: " + msg, file=sys.stderr)
    sys.exit(1)


def fmt(value):
    """Deterministic scalar rendering -- the same input always renders the
    same text on a given Python/platform (REQ-N-004's byte-identity claim
    is about repeated invocations against the same input, not
    cross-platform float repr stability)."""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, float):
        if value == int(value) and abs(value) < 1e15:
            return str(int(value))
        s = ("%.6f" % value).rstrip("0").rstrip(".")
        return s if s not in ("", "-") else "0"
    return str(value)


path = sys.argv[1]

try:
    with open(path) as f:
        content = f.read()
except OSError as e:
    fail("%s: cannot read: %s" % (path, e))

try:
    agg = json.loads(content)
except json.JSONDecodeError as e:
    fail("%s: unparseable JSON: %s" % (path, e))

if not isinstance(agg, dict):
    fail("%s: aggregate is not a JSON object" % path)

schema_version = agg.get("schema_version")
if schema_version not in AGGREGATE_SCHEMA_VERSIONS_SUPPORTED:
    fail(
        "%s: unsupported aggregate schema_version %r (supported: %s)"
        % (path, schema_version, sorted(AGGREGATE_SCHEMA_VERSIONS_SUPPORTED))
    )

lines = []


def emit(text=""):
    lines.append(text)


# --- Provenance (REQ-F-022, AC-21) ------------------------------------
emit("# Shark Bench Baseline Report")
emit()
emit("## Provenance")
emit()

provenance = agg.get("provenance", {})
PROVENANCE_FIELDS = [
    ("Model IDs", "model_ids"),
    ("Fixture base SHA", "fixture_base_sha"),
    ("Variant bundle sha256", "variant_bundle_sha256"),
    ("Corpus schema version", "corpus_schema_version"),
    ("Shark version", "shark_version"),
    ("Reps", "reps"),
]
for label, key in PROVENANCE_FIELDS:
    if key not in provenance:
        continue
    value = provenance[key]
    if isinstance(value, list):
        value_str = ", ".join("`%s`" % v for v in value)
    else:
        value_str = "`%s`" % value
    emit("- %s: %s" % (label, value_str))

if "input_digest" in agg:
    emit("- Input digest: `%s`" % agg["input_digest"])
if "baseline_id" in agg:
    emit("- Baseline id: `%s`" % agg["baseline_id"])
emit()

# --- Noise band per task (REQ-F-021, AC-19/AC-20) ----------------------
emit("## Noise band per task")
emit()

tasks = agg.get("tasks", [])
if not tasks:
    emit("_No per-task noise band published in this aggregate._")
    emit()

for task in sorted(tasks, key=lambda t: t.get("item_id", "")):
    item_id = task.get("item_id", "")
    item_type = task.get("item_type", "task")
    emit("### `%s` (%s)" % (item_id, item_type))
    emit()
    if task.get("non_discriminative"):
        emit(
            "_Flagged non-discriminative -- see \"Corpus feedback to "
            "E40-F01\" below._"
        )
        emit()

    metrics = task.get("metrics", {})
    for metric_id in sorted(metrics):
        block = metrics[metric_id]
        cls = metric_class(metric_id)
        emit("#### `%s`" % metric_id)

        if cls == "A" or (cls is None and "true_count" in block):
            n = block.get("n", 0)
            if "true_count" in block:
                emit(
                    "- n=%d, true_count=%d, rate=%s"
                    % (n, block["true_count"], fmt(block["rate"]))
                )
            else:
                emit("- n=%d" % n)
            if block.get("insufficient_reps"):
                emit(
                    "- insufficient_reps: true -- fewer than two "
                    "contributing reps, no acceptance interval published "
                    "(REQ-F-016)."
                )
            else:
                accept_set = block.get("accept_set", [])
                emit(
                    "- Acceptance set: [%s]"
                    % ", ".join(fmt(v) for v in accept_set)
                )
                emit(
                    "- Derivation: the closed set of boolean values "
                    "observed across the %d contributing reps." % n
                )
            emit()
            continue

        # Class B/C (or an unregistered metric id -- rendered generically).
        n = block.get("n", 0)
        if n == 0:
            emit(
                "- n=0, insufficient_reps: true -- no contributing reps, "
                "no acceptance interval published (REQ-F-016)."
            )
            emit()
            continue

        spread_rel = block.get("spread_rel")
        emit(
            "- n=%d, min=%s, median=%s, max=%s, mean=%s, spread_abs=%s, "
            "spread_rel=%s"
            % (
                n,
                fmt(block["min"]),
                fmt(block["median"]),
                fmt(block["max"]),
                fmt(block["mean"]),
                fmt(block["spread_abs"]),
                fmt(spread_rel) if spread_rel is not None else "null (median is 0)",
            )
        )

        if block.get("insufficient_reps"):
            emit(
                "- insufficient_reps: true -- fewer than two contributing "
                "reps, no acceptance interval published (REQ-F-016)."
            )
            emit()
            continue

        accept_lo, accept_hi = block["accept_lo"], block["accept_hi"]
        emit("- Acceptance interval: [%s, %s]" % (fmt(accept_lo), fmt(accept_hi)))

        if cls == "B":
            emit(
                "- Derivation: fixed integer-count rule -- accept_lo = "
                "max(0, min-1) = %s, accept_hi = max+1 = %s."
                % (fmt(accept_lo), fmt(accept_hi))
            )
        elif cls == "C":
            r = block["spread_abs"]
            median = block["median"]
            if r > 0:
                emit(
                    "- Derivation: observed range r = max - min = %s "
                    "(> 0); interval = [min - r, max + r]." % fmt(r)
                )
            elif median != 0:
                r_eff = 0.10 * abs(median)
                emit(
                    "- Derivation: observed range is zero across every "
                    "rep; interval widened by a fixed 10%% of the median "
                    "(r_eff = 0.10 x |median| = %s); interval = "
                    "[min - r_eff, max + r_eff]." % fmt(r_eff)
                )
            else:
                emit(
                    "- Derivation: metric is identically zero across "
                    "every rep (median = 0); exact interval [0, 0] -- the "
                    "documented zero-spread rule, not the 10%-median-floor "
                    "formula."
                )
        else:
            emit(
                "- Derivation: unregistered metric id -- interval "
                "reproduced from the aggregate as-is."
            )

        if block.get("unusable"):
            emit(
                "- Flagged unusable at the current rep count: spread_abs "
                "exceeds mean (REQ-F-018)."
            )
        emit()

# --- Corpus feedback to E40-F01 (REQ-F-023, AC-14) ----------------------
emit("## Corpus feedback to E40-F01")
emit()
non_discriminative = sorted(agg.get("flags", {}).get("non_discriminative_tasks", []))
if non_discriminative:
    emit(
        "The following tasks reported an identical `oracle.f2p_resolved` "
        "result (all true or all false) across every contributing rep. "
        "They are surfaced here as corpus feedback to E40-F01 for "
        "potential re-screening; F03 flags them and acts on none of it."
    )
    emit()
    for item_id in non_discriminative:
        emit("- `%s`" % item_id)
else:
    emit("No tasks in this baseline were flagged non-discriminative.")
emit()

# --- Measurement caveats (REQ-F-024, AC-20) ------------------------------
# Standing methodology notes -- fixed content, not derived from this
# particular aggregate's data (spec.md "Notes for Agent": a caveat/
# provenance field missing from the aggregate is T-E40-F03-004's bug, not
# something to patch around here).
emit("## Measurement caveats")
emit()
emit(
    "- **Timeout exclusion.** A record whose `outcome` is `timeout` is "
    "excluded from every Class A/B/C band: its `timing.harness_wall_ns` "
    "reflects the harness's `--timeout` cap, not the workload, so a "
    "timed-out run is counted in the outcome distribution and timeout "
    "rate only, never averaged into a band."
)
emit(
    "- **Unexplained-absence anomaly bucket (TD-079).** A record missing "
    "a metric family (`oracle`/`quality`/`loc`) with no explanation on "
    "the record is classified `anomaly`, contributes to no band, and is "
    "named in the aggregate's `anomalies[]` list rather than silently "
    "averaged away."
)
emit(
    "- **fmt/vet attribution (TD-081).** `quality.fmt_clean` and "
    "`quality.vet_ok` are not provably agent-attributable at the current "
    "producer version; treat their observed rate as environment-level "
    "signal, not confirmed per-run agent behavior, until a null-with-"
    "reason value ships for these gates."
)
emit(
    "- **`quality.tests_pass` is not a regression signal (T-004).** The "
    "fixture corpus carries a deliberately, permanently failing "
    "regression probe, so `quality.tests_pass` reads false on nearly "
    "every run; the only regression signal this report publishes is "
    "`oracle.p2p_regressions_count`."
)

print("\n".join(lines))
PYEOF
