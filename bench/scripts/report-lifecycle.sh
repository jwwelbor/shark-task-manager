#!/usr/bin/env bash
# report-lifecycle.sh --aggregate <aggregate.json> --view headline|stage_diagnostic
#
# T-E40-F10-011 (spec.md REQ-F-006, REQ-F-009, REQ-F-012, REQ-F-013,
# REQ-F-016, ADR-F10-06, ADR-F10-10; test-plan.md TC-083, TC-085/TC-086/
# TC-088 report halves; AC-006, AC-008, AC-009, AC-011, AC-T1, AC-T2). A
# pure function of one `aggregate.json` document (ADR-F10-06, modelled on
# `report-baseline.sh`'s own purity contract): reads exactly the file named
# by `--aggregate`, consults no clock, invokes no subprocess, and prints one
# markdown document to stdout, diagnostics to stderr. Every value in the
# report traces to a field already present in the aggregate -- this script
# computes no new statistic; a gap in `aggregate-lifecycle.sh`'s own output
# is that script's bug, never patched around here.
#
# THIS TASK builds `--view headline` only (the primary product result,
# REQ-F-009). `--view stage_diagnostic` is T-E40-F10-012's explicitly
# subordinate view (ADR-F10-07) and refuses cleanly below.
#
# Rendered inline from this file's own heredoc/template strings -- no
# separate `bench/reports/templates/*` files -- so a later static scan
# (T-E40-F10-014, TC-086/TC-088 report halves) targets this one script
# directly for the two closed forbidden-term lists this feature's schema
# owns (REQ-F-013, REQ-F-016).
#
# Sections emitted (spec.md "Data model changes" blocks table, REQ-F-006/
# REQ-F-009/REQ-F-010/REQ-F-011/REQ-F-012/REQ-F-013/REQ-F-016):
#   Eligibility verdicts and     -- per scenario, the verbatim I-08
#     offline evidence              eligibility.publication_eligible verdict
#                                    (REQ-F-006, AC-006: never inferred from
#                                    I-07 outcome.terminal alone) plus every
#                                    evidence artifact's direct file path
#                                    under that scenario's own already-
#                                    present /scenarios[]/retention_path --
#                                    oracle.json, lifecycle.jsonl, evidence/,
#                                    transcripts/ -- resolved by string
#                                    concatenation of a retained field and a
#                                    fixed retention-layout artifact name
#                                    (bench/reports/lifecycle-baseline-
#                                    schema.yaml's retention_required_
#                                    artifacts), never by rerunning the
#                                    scenario or invoking another script.
#   Time and cost partitions     -- REQ-F-010/REQ-F-011: this script prints
#                                    exactly the keys already present in
#                                    `/time` and `/cost`'s own partition
#                                    dicts (which `aggregate-lifecycle.sh`
#                                    already populates with the closed
#                                    stage_category/interval_category/
#                                    share_partition vocabularies plus their
#                                    own `unattributed` residual key) --
#                                    never a restated copy of those upstream
#                                    vocabularies.
#   Review value                 -- REQ-F-012: the seven finding measures
#                                    per gate, broken out by severity and
#                                    defect class, plus elapsed/provider/
#                                    resolution cost and truth-set
#                                    precision/recall, verbatim from
#                                    `/review_value/gates[]`. A gate's state
#                                    (findings / zero_findings /
#                                    collection_failure / not_reached) picks
#                                    a distinct explanatory line so a clean
#                                    zero-finding pass never reads the same
#                                    as a broken collection run.
#   Artifact use and replayed    -- REQ-F-013: produced/consumed/reused/
#     interaction proxy             orphan counts and typed producer/
#                                    consumer edges from
#                                    `/artifact_use`, plus the
#                                    replayed_interaction_proxy sub-block --
#                                    every field there is a replayed
#                                    interaction count, never phrased as an
#                                    observed measurement a person made.
#   Dimension separation and     -- REQ-F-016: quality (`confirmed_findings`
#     paired deltas                 noise band), time (`elapsed_time`), and
#                                    provider cost (`provider_cost`) render
#                                    as three separate blocks per scenario,
#                                    each carrying its own already-published
#                                    `/noise_bands[]` band. The one paired-
#                                    delta value this aggregate carries
#                                    today, `/comparisons[].comparison.
#                                    quality_delta`, is classified against
#                                    that SAME scenario's confirmed_findings
#                                    band using the schema's own closed
#                                    `noise_band_boundary_inclusivity` rule
#                                    (closed interval, a boundary-exact
#                                    delta is INSIDE) -- a classification of
#                                    two already-published values, not a
#                                    derived statistic. There is no upstream
#                                    per-dimension delta field for time or
#                                    provider cost (`compare-lifecycle-
#                                    evaluations.sh`'s own comparison object
#                                    carries only `quality_delta`); this
#                                    script renders those two `unavailable`
#                                    rather than computing a delta itself by
#                                    subtracting two evaluations' own
#                                    elapsed_time/provider_cost -- an honest,
#                                    named upstream gap, matching AC-T2's
#                                    "no new statistic" rule.
#
# Exit status (mirrors report-baseline.sh's own contract):
#   0        report printed to stdout (including a not-correct verdict --
#            ineligibility is expected, reportable content, never a script
#            error)
#   non-zero the aggregate file could not be read, its JSON was unparseable,
#            its identity.schema_version is unsupported, or the requested
#            view is not yet implemented by this script -- nothing is
#            printed to stdout in this case, so a caller redirecting stdout
#            to a file is never left with a partial report.
set -euo pipefail

usage() {
	echo "usage: report-lifecycle.sh --aggregate <aggregate.json> --view headline|stage_diagnostic" >&2
	exit 2
}

aggregate_path=""
view=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--aggregate)
		[[ $# -ge 2 ]] || usage
		aggregate_path="$2"
		shift 2
		;;
	--view)
		[[ $# -ge 2 ]] || usage
		view="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$aggregate_path" ]] || usage
[[ -n "$view" ]] || usage

# Report view names -- shape source: bench/reports/lifecycle-baseline-
# schema.yaml `report_view` (REQ-F-009, ADR-F10-07). A closed CLI-flag
# allowlist checked here, hardcoded rather than read from the schema file,
# so this script never reads a second file beyond the one named by
# `--aggregate` (report-baseline.sh's STATIC_METRIC_CLASS precedent for the
# same purity-over-restatement tradeoff).
case "$view" in
headline | stage_diagnostic) ;;
*)
	usage
	;;
esac

command -v python3 >/dev/null 2>&1 || {
	echo "report-lifecycle: python3 not found on PATH" >&2
	exit 1
}

python3 - "$aggregate_path" "$view" <<'PYEOF'
import json
import sys

AGGREGATE_SCHEMA_VERSIONS_SUPPORTED = {"1.0"}

# The three REQ-F-016 dimensions, each keyed to the exact already-published
# noise_bands[] metric name (aggregate-lifecycle.sh design decision 11) --
# never a private rename.
DIMENSION_METRICS = (
    ("Quality", "confirmed_findings"),
    ("Time", "elapsed_time"),
    ("Cost", "provider_cost"),
)


def fail(msg):
    """Hard-abort with NOTHING printed to stdout: only ever called before
    the report's line list is assembled and joined (report-baseline.sh's
    own convention), so a caller piping stdout to a file is left with a
    zero-byte/absent file, never a partial one."""
    print("report-lifecycle: " + msg, file=sys.stderr)
    sys.exit(1)


def fmt(value):
    """Deterministic scalar rendering -- report-baseline.sh's fmt(), reused
    verbatim so the same value renders the same text in both reports."""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, float):
        if value == int(value) and abs(value) < 1e15:
            return str(int(value))
        return "%.6g" % value
    if value is None:
        return "null"
    return str(value)


path, view = sys.argv[1], sys.argv[2]

try:
    with open(path, encoding="utf-8") as f:
        content = f.read()
except OSError as e:
    fail("%s: cannot read: %s" % (path, e))

try:
    agg = json.loads(content)
except json.JSONDecodeError as e:
    fail("%s: unparseable JSON: %s" % (path, e))

if not isinstance(agg, dict):
    fail("%s: aggregate is not a JSON object" % path)

identity = agg.get("identity")
if not isinstance(identity, dict):
    fail("%s: aggregate missing /identity" % path)

schema_version = identity.get("schema_version")
if schema_version not in AGGREGATE_SCHEMA_VERSIONS_SUPPORTED:
    fail(
        "%s: unsupported aggregate identity.schema_version %r (supported: %s)"
        % (path, schema_version, sorted(AGGREGATE_SCHEMA_VERSIONS_SUPPORTED))
    )

if view == "stage_diagnostic":
    # ADR-F10-07: the stage view is explicitly subordinate and is
    # T-E40-F10-012's own slice -- refused cleanly here, never a traceback,
    # and worded so this refusal is never confused with the DIFFERENT
    # "no headline verdict available" refusal T-E40-F10-012 will add.
    fail(
        "--view stage_diagnostic is not implemented by this build of "
        "report-lifecycle.sh yet (T-E40-F10-012 adds it); only "
        "--view headline renders today"
    )

lines = []


def emit(text=""):
    lines.append(text)


# ---------------------------------------------------------------------------
# Header (REQ-F-017/AC-012: every F10 report carries its phase label).
# ---------------------------------------------------------------------------
emit("# Shark Bench Lifecycle Headline Report")
emit()
emit("- Phase: `%s`" % fmt(identity.get("phase")))
emit("- Batch id: `%s`" % fmt(identity.get("batch_id")))
emit("- Retention root digest: `%s`" % fmt(identity.get("retention_root_digest")))
emit("- Batch policy digest: `%s`" % fmt(identity.get("batch_policy_digest")))
emit("- Declared minimum reps: %s" % fmt(identity.get("min_reps")))
emit()

# ---------------------------------------------------------------------------
# Eligibility verdicts and offline evidence (REQ-F-006, REQ-F-009, AC-006).
# ---------------------------------------------------------------------------
emit("## Eligibility verdicts and offline evidence")
emit()

scenarios = agg.get("scenarios") or []
quality_by_scenario = {
    (q.get("scenario_id"), q.get("rep")): q
    for q in ((agg.get("quality") or {}).get("by_scenario") or [])
}

if not scenarios:
    emit("_No scenarios recorded in this aggregate._")
    emit()

for s in sorted(scenarios, key=lambda s: (s.get("scenario_id") or "", s.get("rep") or 0)):
    sid, rep = s.get("scenario_id"), s.get("rep")
    emit("### `%s` rep %s" % (sid, rep))
    emit()
    emit("- Family: `%s`, scenario version: `%s`" % (fmt(s.get("family")), fmt(s.get("scenario_version"))))
    retention_path = s.get("retention_path")
    emit("- Retention path: `%s`" % fmt(retention_path))

    elig = s.get("eligibility") or {}
    aggregate_eligible = elig.get("aggregate_eligible")
    publication_eligible = elig.get("publication_eligible")
    invalidity_reasons = elig.get("invalidity_reasons") or []
    emit(
        "- I-08 eligibility (verbatim): aggregate_eligible=%s, publication_eligible=%s"
        % (fmt(aggregate_eligible), fmt(publication_eligible))
    )
    # AC-006: the verdict below is read from I-08's own verbatim
    # eligibility.publication_eligible -- NEVER from I-07's own
    # outcome.terminal/outcome.publication_eligible fields -- so a
    # completed-but-failed-oracle workflow can never render as a false pass
    # inferred from terminal status alone.
    if publication_eligible is True:
        emit("- **VERDICT: correct** -- publication-eligible per its verbatim I-08 eligibility.")
    else:
        emit(
            "- **VERDICT: NOT CORRECT** -- publication-ineligible per its verbatim I-08 "
            "eligibility, never inferred from I-07 outcome.terminal alone."
        )
    if invalidity_reasons:
        emit("- invalidity_reasons (verbatim):")
        for reason in invalidity_reasons:
            emit("  - `%s`" % json.dumps(reason, sort_keys=True))
    else:
        emit("- invalidity_reasons: none")
    emit()

    # REQ-F-006: every evidence artifact below is reachable by direct file
    # path -- this scenario's own already-present retention_path plus the
    # fixed retention-layout artifact name (bench/reports/lifecycle-
    # baseline-schema.yaml retention_required_artifacts) -- never by
    # rerunning the scenario or invoking another script.
    quality_entry = quality_by_scenario.get((sid, rep))
    oracle = (quality_entry or {}).get("execution_oracle")
    if isinstance(oracle, dict) and "observed_result" in oracle:
        oracle_text = "observed_result=%s" % fmt(oracle["observed_result"])
    else:
        # TC-083 Negative Cases: a missing execution_oracle result must
        # render distinctly from one that ran and passed -- never collapsed
        # into the same rendering.
        oracle_text = "no observed_result present in this record (distinct from a passed oracle)"
    emit("- Offline evidence (reachable without rerunning the scenario or contacting a provider):")
    emit("  - Execution oracle result (%s): `%s/oracle.json`" % (oracle_text, retention_path))
    emit("  - Dispatch and gate lineage (I-07 record): `%s/lifecycle.jsonl`" % retention_path)
    emit("  - Per-stage evidence bundle: `%s/evidence/`" % retention_path)
    emit("  - Stage transcripts: `%s/transcripts/`" % retention_path)
    emit()

    # REQ-F-008: an apparent gap between I-08's own execution_oracle result
    # and its own eligibility verdict is reported as an upstream contract
    # defect IN the document -- never overridden, softened, or resolved
    # locally, and never silently trusted from either field alone.
    if isinstance(oracle, dict) and oracle.get("observed_result") == "fail" and publication_eligible is True:
        emit(
            "> **UPSTREAM CONTRACT DEFECT (REQ-F-008):** execution_oracle.observed_result "
            "is `fail` for this scenario, but I-08 eligibility.publication_eligible is "
            "`true`. This report neither overrides nor softens either verbatim fact -- "
            "both are printed above."
        )
        emit()

# ---------------------------------------------------------------------------
# Time and cost partitions (REQ-F-010, REQ-F-011). Every key printed below
# is a key already present in aggregate-lifecycle.sh's own partition dict
# (the closed stage_category/interval_category/share_partition vocabularies
# plus their own `unattributed` residual key) -- iterated directly, never a
# restated copy of an upstream vocabulary.
# ---------------------------------------------------------------------------
emit("## Time and cost partitions")
emit()

time_block = agg.get("time") or {}
cost_block = agg.get("cost") or {}

emit("- Lifecycle wall time (seconds, rolled up across every retained pair): %s" % fmt(time_block.get("lifecycle_wall_seconds")))
emit()


def emit_partition(title, block):
    emit("#### %s" % title)
    if not isinstance(block, dict) or not block:
        emit("_not present in this aggregate_")
        emit()
        return
    for key in sorted(block):
        emit("- `%s`: %s" % (key, fmt(block[key])))
    emit()


emit("### Time")
emit()
emit_partition("Stage-category partition", time_block.get("stage_category"))
emit_partition("Interval-category partition", time_block.get("interval_category"))
emit_partition("REQ-F-011 share partition", time_block.get("share_partition"))

emit("### Cost")
emit()
emit_partition("Stage-category partition", cost_block.get("stage_category"))
emit_partition("Interval-category partition", cost_block.get("interval_category"))
emit_partition("REQ-F-011 share partition", cost_block.get("share_partition"))

ceiling = cost_block.get("ceiling_consumption") or {}
emit("#### Ceiling consumption")
emit("- observed_cost_usd: %s" % fmt(ceiling.get("observed_cost_usd")))
emit("- max_cost_usd: %s" % fmt(ceiling.get("max_cost_usd")))
emit()

# ---------------------------------------------------------------------------
# Review value (REQ-F-012, REQ-F-015, ADR-F10-10). Every field below is
# read verbatim from /review_value/gates[] -- this script performs no
# arithmetic on a finding count that /review_value doesn't already carry.
# ---------------------------------------------------------------------------
emit("## Review value")
emit()

gates = (agg.get("review_value") or {}).get("gates") or []
if not gates:
    emit("_No review gates recorded in this aggregate._")
    emit()


def emit_measure_row(label, measures):
    emit(
        "  - `%s`: emitted=%s, normalized_unique=%s, duplicate=%s, recurrent=%s, "
        "confirmed=%s, unconfirmed=%s, downstream_escape=%s"
        % (
            label,
            fmt(measures.get("emitted")),
            fmt(measures.get("normalized_unique")),
            fmt(measures.get("duplicate")),
            fmt(measures.get("recurrent")),
            fmt(measures.get("confirmed")),
            fmt(measures.get("unconfirmed")),
            fmt(measures.get("downstream_escape")),
        )
    )


for gate in sorted(gates, key=lambda g: g.get("gate_id") or ""):
    emit("### Gate `%s`" % gate.get("gate_id"))
    state = gate.get("state")
    emit("- State: `%s`" % fmt(state))
    # TC-085 Negative Cases: a zero-finding gate and a collection-failure
    # gate must never render the same way -- each closed gate_state value
    # gets its own explanatory sentence, distinct from the others.
    if state == "zero_findings":
        emit("- This gate ran to completion and reported zero findings (a clean pass, not a missing measurement).")
    elif state == "collection_failure":
        emit("- **COLLECTION FAILURE** -- finding collection for this gate did not complete; this is not the same as a clean zero-finding pass.")
    elif state == "not_reached":
        emit("- This gate was not reached by the workflow.")

    counts = gate.get("counts") or {}
    emit("- Counts:")
    emit_measure_row("total", counts)

    by_severity = gate.get("counts_by_severity") or {}
    if by_severity:
        emit("- Counts by severity:")
        for severity in sorted(by_severity):
            emit_measure_row(severity, by_severity[severity])

    by_defect_class = gate.get("counts_by_defect_class") or {}
    if by_defect_class:
        emit("- Counts by defect class:")
        for defect_class in sorted(by_defect_class):
            emit_measure_row(defect_class, by_defect_class[defect_class])

    emit("- elapsed_seconds: %s" % fmt(gate.get("elapsed_seconds")))
    emit("- provider_cost_usd: %s" % fmt(gate.get("provider_cost_usd")))
    emit("- resolution_cost_usd: %s" % fmt(gate.get("resolution_cost_usd")))
    emit("- truth_set_available: %s" % fmt(gate.get("truth_set_available")))
    # AC-T2: rendered by fmt() unchanged when the aggregate already carries
    # the literal string "unavailable" -- never coerced to 0 or an empty
    # value here.
    emit("- precision: %s" % fmt(gate.get("precision")))
    emit("- recall: %s" % fmt(gate.get("recall")))
    emit()

# ---------------------------------------------------------------------------
# Artifact use and replayed interaction proxy (REQ-F-013). Every replay
# proxy field below is labelled as a replayed count -- never phrased as an
# observed measurement a person made.
# ---------------------------------------------------------------------------
emit("## Artifact use")
emit()

artifact_use = agg.get("artifact_use") or {}
emit("- Produced: %s" % fmt(artifact_use.get("produced_count")))
emit("- Consumed: %s" % fmt(artifact_use.get("consumed_count")))
emit("- Reused: %s" % fmt(artifact_use.get("reused_count")))
emit("- Orphan: %s" % fmt(artifact_use.get("orphan_count")))
emit()

edges = artifact_use.get("edges") or []
if edges:
    emit("### Producer/consumer edges")
    emit()
    emit("| Producer stage | Artifact path | Artifact type | Consuming stage | Edge kind |")
    emit("| --- | --- | --- | --- | --- |")
    for edge in sorted(
        edges,
        key=lambda e: (
            e.get("producer_stage") or "",
            e.get("artifact_path") or "",
            e.get("consuming_stage") or "",
            e.get("edge_kind") or "",
        ),
    ):
        emit(
            "| `%s` | `%s` | `%s` | `%s` | `%s` |"
            % (
                fmt(edge.get("producer_stage")),
                fmt(edge.get("artifact_path")),
                fmt(edge.get("artifact_type")),
                fmt(edge.get("consuming_stage")),
                fmt(edge.get("edge_kind")),
            )
        )
    emit()
else:
    emit("_No producer/consumer edges recorded in this aggregate._")
    emit()

proxy = artifact_use.get("replayed_interaction_proxy") or {}
emit("### Replayed interaction proxy")
emit()
emit("_%s_" % fmt(proxy.get("label")))
emit()
emit("- Replayed request count: %s" % fmt(proxy.get("request_count")))
emit("- Replayed response count: %s" % fmt(proxy.get("response_count")))
emit("- Replayed payload size (bytes): %s" % fmt(proxy.get("payload_size_bytes")))
emit("- Replayed revision count: %s" % fmt(proxy.get("revision_count")))
emit("- Replayed unresolved-gate count: %s" % fmt(proxy.get("unresolved_gate_count")))
emit()

# ---------------------------------------------------------------------------
# Dimension separation and paired deltas (REQ-F-016). Quality, time, and
# cost render as three separate blocks, each carrying its own already-
# published noise_bands[] entry -- never a single merged figure.
# ---------------------------------------------------------------------------
emit("## Dimension separation and paired deltas")
emit()

bands_by_scenario = {}
for band in agg.get("noise_bands") or []:
    bands_by_scenario.setdefault(band.get("scenario_id"), {})[band.get("metric")] = band

comparisons_by_scenario = {}
for comparison in agg.get("comparisons") or []:
    comparisons_by_scenario.setdefault(comparison.get("scenario_id"), []).append(comparison)

if not bands_by_scenario:
    emit("_No noise bands published in this aggregate._")
    emit()


def emit_band(band):
    if not isinstance(band, dict):
        emit("  - Noise band: not published for this scenario/metric.")
        return
    interval = band.get("acceptance_interval") or {}
    emit(
        "  - Noise band: min=%s, median=%s, max=%s, spread_abs=%s, "
        "acceptance_interval=[%s, %s], derivation_rule=`%s`, rep_count=%s, "
        "insufficient_reps=%s"
        % (
            fmt(band.get("min")),
            fmt(band.get("median")),
            fmt(band.get("max")),
            fmt(band.get("spread_abs")),
            fmt(interval.get("lower_bound")),
            fmt(interval.get("upper_bound")),
            fmt(band.get("derivation_rule")),
            fmt(band.get("rep_count")),
            fmt(band.get("insufficient_reps")),
        )
    )


for sid in sorted(bands_by_scenario):
    emit("### `%s`" % sid)
    emit()
    per_metric = bands_by_scenario[sid]

    # REQ-F-016: the one paired-delta value this aggregate carries today is
    # /comparisons[].comparison.quality_delta (compare-lifecycle-
    # evaluations.sh's own comparison object shape has no time_delta/
    # cost_delta field -- checked). This never derives a delta itself by
    # subtracting two evaluations' own elapsed_time/provider_cost, which
    # would be exactly the new statistic AC-T2 forbids.
    quality_delta = None
    for comparison in comparisons_by_scenario.get(sid, []):
        candidate = (comparison.get("comparison") or {}).get("quality_delta")
        if isinstance(candidate, (int, float)) and not isinstance(candidate, bool):
            quality_delta = candidate
            break

    for label, metric_name in DIMENSION_METRICS:
        emit("#### %s (`%s`)" % (label, metric_name))
        band = per_metric.get(metric_name)
        emit_band(band)
        if metric_name != "confirmed_findings":
            # No upstream per-dimension delta field exists for time or
            # cost -- an honest gap, never a value computed here.
            emit("  - Paired delta: unavailable")
            emit()
            continue
        if quality_delta is None:
            emit("  - Paired delta: unavailable")
            emit()
            continue
        interval = (band or {}).get("acceptance_interval") or {}
        lower, upper = interval.get("lower_bound"), interval.get("upper_bound")
        # bench/reports/lifecycle-baseline-schema.yaml
        # noise_band_boundary_inclusivity: the published interval is closed
        # on both bounds -- a delta landing exactly on a bound is INSIDE
        # the band. This classifies two already-published values; it does
        # not derive a new one.
        inside = (
            isinstance(lower, (int, float))
            and isinstance(upper, (int, float))
            and lower <= quality_delta <= upper
        )
        if inside:
            emit("  - Paired delta: %s -- no detectable effect" % fmt(quality_delta))
        else:
            emit(
                "  - Paired delta: %s -- outside this scenario's published "
                "confirmed_findings noise band" % fmt(quality_delta)
            )
        emit()

print("\n".join(lines))
sys.exit(0)
PYEOF
