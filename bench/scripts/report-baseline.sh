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
    cross-platform float repr stability).

    R2-F-9 (uat-20260808-231500-E40-F03.md): a fixed 6-decimal-PLACE format
    (the old `"%.6f" % value`) rounds any magnitude below ~1e-6 to the
    literal string "0" -- a real cost_usd_total band of
    [4e-7, 5e-7, 6e-7] printed as spread_abs=0 alongside a nonzero
    spread_rel, and a derivation sentence asserting "0 > 0". `%.6g` renders
    by SIGNIFICANT digits instead: it switches to scientific notation only
    once the fixed-decimal form would need more leading zeros than that
    (the "small-magnitude branch" the finding names), so 4e-07 prints as
    "4e-07" -- still non-zero -- rather than being rounded away. `%g` also
    strips its own trailing zeros, so no rstrip() post-processing is
    needed."""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, float):
        if value == int(value) and abs(value) < 1e15:
            return str(int(value))
        return "%.6g" % value
    return str(value)


def render_value(value):
    """Shared renderer for a provenance-field VALUE (never a whole row --
    the caller still owns the label and the "absent from the aggregate"
    row-level cases, which have no `value` to render at all). NEW-5
    (uat-20260809-013000-E40-F03.md): the Divergences table used to
    interpolate `entry["value"]` directly into a `%s` -- a bare Python
    `None` for the R2-F-11 "every contributor shares this absence" shape,
    or a raw list repr (`['claude-evil-9']`) for an ordinary diverging
    list -- while the PROVENANCE_FIELDS loop three lines below rendered
    the exact same value shapes correctly. Both call sites now share this
    one function so they can never again disagree.

    `None` here is semantically the same absence T-E40-F03-003's
    `unresolved_fields` already names for the single-agreed-value case
    (`provenance[key] is None`): a field no contributing record could
    supply a usable value for, whether that showed up in `provenance{}`
    directly or as a `values: [{"value": null, ...}]` divergence entry.
    One wording for one meaning."""
    if value is None:
        return "_not resolvable from the contributing records_"
    if isinstance(value, list):
        return ", ".join("`%s`" % v for v in value)
    return "`%s`" % value


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


provenance = agg.get("provenance", {})
uniform = provenance.get("uniform")

emit("# Shark Bench Baseline Report")
emit()

# --- INVALID AS A BASELINE banner (R2-F-2, REQ-F-011, AC-11) ------------
# `provenance.uniform` and `provenance.divergences` used to be computed by
# aggregate-runs.sh and then never read anywhere in this file -- a
# non-uniform batch rendered as a normal, confident, exit-0 report
# indistinguishable from a pinned one. REQ-F-011 requires a non-uniform
# batch to be "reported as invalid as a baseline ... never published as a
# result", so this must be the first thing a reader sees, and the report
# must exit non-zero (see the exit-status block at the bottom of this
# file), mirroring aggregate-runs.sh's own contract.
if uniform is False:
    emit(
        "> **INVALID AS A BASELINE** -- provenance is not uniform across "
        "contributing records; this document is not a publishable "
        "baseline."
    )
    emit()

# --- Provenance (REQ-F-022, AC-21) ------------------------------------
emit("## Provenance")
emit()

divergences = provenance.get("divergences", [])
if divergences:
    emit("### Divergences")
    emit()
    emit("| Field | Value | Run keys |")
    emit("| --- | --- | --- |")
    for div in divergences:
        for entry in div["values"]:
            emit(
                "| %s | %s | %s |"
                % (
                    div["field"],
                    render_value(entry["value"]),
                    ", ".join("`%s`" % rk for rk in entry["run_keys"]),
                )
            )
    emit()

divergent_fields = {div["field"] for div in divergences}
PROVENANCE_FIELDS = [
    ("Model IDs", "model_ids"),
    ("Fixture base SHA", "fixture_base_sha"),
    ("Variant bundle sha256", "variant_bundle_sha256"),
    ("Corpus schema version", "corpus_schema_version"),
    ("Shark version", "shark_version"),
    ("Reps", "reps"),
]
# root-level (not `provenance{}`-nested) fields that are still part of the
# Provenance section's REQ-F-022/AC-21 contract (see header comment). They
# have no divergence concept of their own (they are computed once by
# aggregate-runs.sh, never per-contributing-record) but share the exact
# same "a field this section reports on can be absent from the aggregate,
# and that absence must be its own sentence, never silence" discipline --
# CR6-1 (code review round 6): `baseline_id` is deliberately withheld by
# aggregate-runs.sh on at least one live, tested, exit-0/uniform=true
# aggregate (NEW-2(a): a declared item with zero records), and the old
# `if key in agg: emit(...)` pair printed nothing at all in that case,
# leaving a reader with no way to tell "withheld on purpose" from "this
# aggregate's schema doesn't carry the field" from "present but missed".
ROOT_PROVENANCE_FIELDS = [
    ("Input digest", "input_digest"),
    ("Baseline id", "baseline_id"),
]


def emit_provenance_row(label, present, render):
    """One Provenance-section row for a field that might legitimately be
    absent from the aggregate. Presence and absence are both explicit,
    unconditional outcomes -- never a silently skipped row (R2-F-2/R2-F-4/
    CR6-1, the third instance of this file rendering an affirmative claim,
    or nothing at all, from a possibly-absent aggregate key). Both
    PROVENANCE_FIELDS (provenance-block fields, which can also be
    "absent because they diverged") and ROOT_PROVENANCE_FIELDS
    (root-level computed fields, which can be withheld) route through
    this one function so a field can't be added to this section again
    without going through the vetted absence-naming path."""
    if present:
        emit("- %s: %s" % (label, render()))
    else:
        emit("- %s: _absent from the aggregate_" % label)


# R2-F-2/R2-F-4 (uat-20260808-231500-E40-F03.md): every row is printed
# unconditionally -- a field's absence is now a fact the reader is told,
# never a silently skipped list entry. Three distinct absent-ness shapes,
# each its own explicit sentence (never a bare Python `None`):
#   - the field diverged across contributors: aggregate-runs.sh omits it
#     from `provenance{}` entirely (it is named above in the Divergences
#     table instead) -- rendered "_absent from the aggregate_" (R2-F-2 fix
#     guidance #2).
#   - the field could not be resolved from any contributing record
#     (`provenance[field] is None`, T-E40-F03-003's `unresolved_fields`) --
#     rendered "_not resolvable from the contributing records_" (R2-F-4,
#     via render_value()).
#   - the field has a real, agreed value -- rendered verbatim as before.
for label, key in PROVENANCE_FIELDS:
    present = key not in divergent_fields and key in provenance
    emit_provenance_row(label, present, lambda key=key: render_value(provenance[key]))

# CR6-1: root-level computed fields, same absence-naming discipline --
# `render_value()` keeps the rendering (backtick-wrapped scalar) identical
# to what the old bare `emit("- %s: \`%s\`" % (...))` lines produced when
# the field was present.
for label, key in ROOT_PROVENANCE_FIELDS:
    emit_provenance_row(label, key in agg, lambda key=key: render_value(agg[key]))
emit()

# --- Data quality (R2-F-3, REQ-N-005/REQ-F-012/REQ-F-021) ---------------
# aggregate-runs.sh computes flags.reduced_reps/unexpected_reps/
# insufficient_reps/unusable_metrics/missing_items/unexpected_items,
# anomalies[], and outcomes.anomaly_count/timeout_rate, and until R2-F-3's
# fix none of them were read by this file -- a reduced-rep-set band and an
# anomalous-record aggregate both rendered a clean report with zero
# mention of the reduction/anomaly (round-1 F-3's exact complaint,
# reappearing one layer downstream). This section is the single place a
# reader learns of them.
#
# NEW-4 (uat-20260809-013000-E40-F03.md, round 3): `flags{}` itself is
# only ever computed when `provenance.uniform` is true (aggregate-runs.sh
# gates `aggregate["flags"] = {...}` behind `if uniform:`) -- over a
# non-uniform batch it is not "computed, empty", it is ABSENT, and the
# old `flags.get(key, []) -> falsy -> "none."` fallback could not tell the
# two apart: a non-uniform aggregate rendered five confident "none."
# statements about analyses that never ran. Every flags-derived category
# below now branches on `flags_computed` FIRST -- presence, not
# emptiness -- before ever reaching the empty-list "none." case.
# `anomalies`/`outcomes` are set unconditionally in aggregate-runs.sh
# regardless of `uniform` (they are computed from the raw records, not
# from the statistical `if uniform:` block), so they keep the plain
# presence-vs-emptiness `if anomalies:` shape -- they are never "not
# computed".
NOT_COMPUTED = (
    "_not computed (provenance is not uniform; the statistical blocks "
    "were suppressed)_"
)

emit("## Data quality")
emit()

flags_computed = "flags" in agg
flags = agg.get("flags", {})
anomalies = agg.get("anomalies", [])
outcomes = agg.get("outcomes", {})

emit(
    "- outcomes.anomaly_count: %d, outcomes.timeout_rate: %s"
    % (outcomes.get("anomaly_count", 0), fmt(outcomes.get("timeout_rate", 0)))
)
if anomalies:
    emit("- Anomalous records (excluded from every band, contributes to no statistic):")
    for a in sorted(anomalies, key=lambda e: e["run_key"]):
        emit(
            "  - `%s` -- missing families: %s"
            % (a["run_key"], ", ".join(a.get("missing_families", [])))
        )
else:
    emit("- Anomalous records: none.")


def emit_flags_category(label, description, items, render_row):
    """One flags[]-derived Data-quality line -- presence-before-emptiness
    (NEW-4): `flags` absent entirely means this category was never
    computed, and must say so ("- <label>: <NOT_COMPUTED>"), never
    "none.". `label` is the short name used for both the not-computed and
    the empty ("none.") one-liners; `description` is the longer
    parenthetical shown only on the header of a non-empty listing, same
    as this section's original wording."""
    if not flags_computed:
        emit("- %s: %s" % (label, NOT_COMPUTED))
        return
    if items:
        emit("- %s (%s):" % (label, description))
        for item in items:
            emit(render_row(item))
    else:
        emit("- %s: none." % label)


emit_flags_category(
    "Reduced rep sets",
    "fewer contributing reps than the published matrix",
    sorted(flags.get("reduced_reps", []), key=lambda e: e["item_id"]),
    lambda r: "  - `%s`: %d of %d reps contributed." % (r["item_id"], r["contributing_reps"], r["published_reps"]),
)

emit_flags_category(
    "Unexpected reps",
    "outside the published matrix, excluded from every band",
    sorted(flags.get("unexpected_reps", []), key=lambda e: e["item_id"]),
    lambda r: "  - `%s`: rep(s) %s excluded (published matrix: %d reps)."
    % (r["item_id"], ", ".join(str(v) for v in r["unexpected_reps"]), r["published_reps"]),
)

emit_flags_category(
    "Missing items",
    "declared via --items but no record at all",
    flags.get("missing_items", []),
    lambda item_id: "  - `%s`" % item_id,
)

emit_flags_category(
    "Unexpected items",
    "observed but outside the declared --items set",
    flags.get("unexpected_items", []),
    lambda item_id: "  - `%s`" % item_id,
)

emit_flags_category(
    "Insufficient reps",
    "fewer than two contributing reps, no acceptance interval published",
    sorted(flags.get("insufficient_reps", []), key=lambda e: (e["item_id"], e["metric"])),
    lambda e: "  - `%s` / `%s`" % (e["item_id"], e["metric"]),
)

emit_flags_category(
    "Unusable metrics",
    "spread_abs exceeds mean at the current rep count",
    sorted(flags.get("unusable_metrics", []), key=lambda e: (e["item_id"], e["metric"])),
    lambda e: "  - `%s` / `%s`" % (e["item_id"], e["metric"]),
)
emit()

# --- Noise band per task (REQ-F-021, AC-19/AC-20) ----------------------
emit("## Noise band per task")
emit()

tasks = agg.get("tasks", [])
if not tasks:
    # NEW-4: same presence-vs-emptiness split as the Data-quality section
    # above -- `tasks` is absent entirely on a non-uniform batch
    # (aggregate-runs.sh's own `if uniform:` gate), a fundamentally
    # different fact from a uniform batch that genuinely contributed zero
    # tasks. The old unconditional "No per-task noise band published"
    # sentence could not tell a reader which one happened.
    if "tasks" not in agg:
        emit(
            "_No per-task noise band published in this aggregate -- "
            "suppressed because provenance is not uniform; the "
            "statistical blocks were never computed._"
        )
    else:
        emit(
            "_No per-task noise band published in this aggregate -- the "
            "batch is uniform but contributed zero tasks._"
        )
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

        # R2-F-3 fix guidance #2: the connective tissue between the
        # header's published rep count and this band's own n -- "the
        # header says r3, the band says n=2, and nothing connects them"
        # (round 1's own words). Printed whenever a record was excluded,
        # whether or not n itself dropped below the published reps (a
        # metric can carry excluded[] entries from unrelated per-rep
        # reasons, e.g. gate_not_executed, even at full n).
        excluded_entries = block.get("excluded", [])
        excluded_line = None
        if excluded_entries:
            excluded_reasons = sorted({e["reason"] for e in excluded_entries})
            excluded_line = "- Excluded: %d record(s) -- reasons: %s" % (
                len(excluded_entries),
                ", ".join(excluded_reasons),
            )

        if cls == "A" or (cls is None and "true_count" in block):
            n = block.get("n", 0)
            if "true_count" in block:
                emit(
                    "- n=%d, true_count=%d, rate=%s"
                    % (n, block["true_count"], fmt(block["rate"]))
                )
            else:
                emit("- n=%d" % n)
            if excluded_line:
                emit(excluded_line)
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
            if excluded_line:
                emit(excluded_line)
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
        if excluded_line:
            emit(excluded_line)

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
                    "(> 0); interval = [max(0, min - r), max + r] = "
                    "[%s, %s]." % (fmt(r), fmt(accept_lo), fmt(accept_hi))
                )
            elif median != 0:
                r_eff = 0.10 * abs(median)
                emit(
                    "- Derivation: observed range is zero across every "
                    "rep; interval widened by a fixed 10%% of the median "
                    "(r_eff = 0.10 x |median| = %s); interval = "
                    "[max(0, min - r_eff), max + r_eff] = [%s, %s]."
                    % (fmt(r_eff), fmt(accept_lo), fmt(accept_hi))
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
# CR5-1 (code review round 5): same presence-vs-emptiness defect class as
# NEW-4 (see emit_flags_category() above) -- `flags` is only computed when
# provenance is uniform, so `agg.get("flags", {}).get(...)` could not tell
# "computed, empty" apart from "never ran" and rendered a confident "no
# tasks flagged" claim about an analysis that never happened. Branch on
# flags_computed first, same as every other flags-derived line.
emit("## Corpus feedback to E40-F01")
emit()
if not flags_computed:
    emit("Non-discriminative-task detection: %s" % NOT_COMPUTED)
else:
    non_discriminative = sorted(flags.get("non_discriminative_tasks", []))
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

# --- Exit status (R2-F-2, R2-F-3) ----------------------------------------
# The document is always printed in full before this check runs -- an
# invalid/anomalous batch is named IN the report (the banner, the
# Divergences table, the Data quality section above), never suppressed;
# only the exit code changes, mirroring aggregate-runs.sh's own contract
# (full document still printed on its own non-zero anomaly/non-uniform
# exit). A caller that reads only this script's exit code -- never the
# aggregator's -- can no longer publish an invalid or anomalous baseline
# as if it were clean.
if uniform is False:
    sys.exit(1)
if agg.get("anomalies"):
    sys.exit(1)
sys.exit(0)
PYEOF
