#!/usr/bin/env bash
# replay-manifest.sh --record <path/to/record.jsonl>
#                     --band <aggregate.json>
#                     --out <replay_artifact_root>
#                     [--corpus <corpus.yaml>] [--skip-canary]
#
# T-E40-F03-006 (spec.md ADR-F03-02, REQ-F-025..028; test-plan.md TC-019a/
# b/c/h): the DISPATCH half of G7 replay. Checks REQ-F-027's three
# preconditions before any API spend, builds a synthetic single-item
# corpus.yaml pinning the stored manifest's fixture_base_sha/
# corpus_schema_version/p2p_set, records corpus_drift against the live
# corpus, and invokes run-one.sh (via RUN_ONE_BIN) with the manifest's
# stored --rep value against a distinct --out root.
#
# T-E40-F03-007 (spec.md ADR-F03-05, REQ-F-029/030; test-plan.md TC-019d/
# e/f/g): the POST-DISPATCH comparison half. After a successful dispatch,
# compares the fresh record's manifest.variant_bundle_sha256/.model_ids
# against the stored record's -- either mismatch (or REQ-F-028's own
# corpus_drift from the dispatch half above) yields the verdict `invalid`,
# never `fail`: the inputs were not reproduced, so a metric comparison
# would be meaningless (ADR-F03-05). Only when every identity field
# matches does this script compute a per-metric comparison: the freshly
# dispatched single record is aggregated on its own (via a $WORKDIR-scoped
# copy, NEVER the caller's --out root directly -- a reused --out root could
# hold OTHER items/dispatches whose divergent provenance would trip
# aggregate-runs.sh's own uniformity check and suppress tasks[] entirely)
# by invoking aggregate-runs.sh (sibling path, never stubbed -- the
# Caller-Path Contract stubs only the external run-one.sh call), and each
# metric published in the --band's task entry is compared against the
# freshly computed single-rep value. The three-valued verdict is
# `invalid` (identity mismatch), `fail` (every identity field matches, at
# least one metric outside its published interval), or `pass` (every
# identity field matches, every comparable metric inside its interval).
#
# REQ-F-027's three preconditions (checked in order, ALL collected before
# reporting -- a caller debugging a replay needs every failing check in
# one pass, not one-at-a-time):
#   (i)   bench/corpus/ledgers/<manifest.fixture_base_sha>/ exists --
#         resolved from THIS SCRIPT's own location ($BENCH_DIR/corpus/
#         ledgers/<sha>/), never from --corpus's directory (spec.md: the
#         base-SHA ledger paths run-one.sh reads are derived from its own
#         location, not the corpus file's -- a synthetic corpus in a temp
#         dir still resolves the real ledgers).
#   (ii)  the stored record's manifest.item_id resolves in --corpus's
#         items: list.
#   (iii) the --band aggregate has an entry for (item_id, variant_id): a
#         tasks[] entry with a matching item_id, AND the aggregate's own
#         baseline_id (the only place aggregate.json exposes its variant)
#         reconstructs to "<variant_id>-<fixture_base_sha[:12]>-r<reps>"
#         from the BAND's own provenance.fixture_base_sha/reps -- never
#         from the manifest's fixture_base_sha, so a drifted fixture_base_
#         sha (precondition i's failure) never also manufactures a false
#         precondition (iii) failure.
# Any failure here means ZERO run-one.sh invocations (REQ-F-027's whole
# point: a replay that cannot be valid costs no API spend).
#
# Synthetic replay corpus (ADR-F03-02): written to a temp directory,
# never bench/corpus/corpus.yaml. Carries the source corpus's fixture:/
# p2p_sets: blocks verbatim (fixture.base_sha overwritten from the
# manifest -- corpus.yaml's own INVARIANT requires it match the item's
# own fixture_base_sha byte-for-byte), one items: entry (a deep copy of
# the resolved source item with fixture_base_sha/p2p_set overwritten from
# the manifest and seed_path/f2p.paths absolutized against the SOURCE
# corpus's own directory, per run-one.sh's os.path.join(corpus_dir, path)
# resolution -- an already-absolute path is returned unchanged), and no
# negative_items: key at all.
#
# corpus_drift (REQ-F-028): any divergence between the LIVE corpus item's
# fixture_base_sha/p2p_set or the live corpus's top-level schema_version
# and the manifest's pinned values is named -- one reasons[] entry per
# divergent field, keyed by the manifest's own field name
# (corpus_schema_version, not schema_version) so T-E40-F03-007's
# variant_bundle_drift/model_version_drift entries share one vocabulary.
# The dispatched synthetic corpus always carries the MANIFEST's values
# regardless of any detected drift.
#
# RUN_ONE_BIN default resolution (TD-077's defect class, same binding
# decision as run-batch.sh/T-E40-F03-001): resolved as
# ${RUN_ONE_BIN:-$SCRIPT_DIR/run-one.sh}, the SIBLING path, never a bare
# PATH-resolved name.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RUN_ONE_BIN="${RUN_ONE_BIN:-$SCRIPT_DIR/run-one.sh}"

# aggregate-runs.sh: the T-E40-F03-007 per-metric comparison's own
# primitive (invoked, never re-derived). Always the sibling path -- unlike
# RUN_ONE_BIN, this is never an API-spend boundary, so no override seam is
# offered: TC-019's Caller-Path Contract stubs only the external
# run-one.sh call, never replay-manifest.sh's own comparison logic.
AGGREGATE_BIN="$SCRIPT_DIR/aggregate-runs.sh"

usage() {
	cat >&2 <<'EOF'
usage: replay-manifest.sh --record <path/to/record.jsonl>
                          --band <aggregate.json>
                          --out <replay_artifact_root>
                          [--corpus <corpus.yaml>] [--skip-canary]
EOF
	exit 2
}

record_path=""
band_path=""
out_root=""
corpus_yaml=""
skip_canary="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--record)
		[[ $# -ge 2 ]] || usage
		record_path="$2"
		shift 2
		;;
	--band)
		[[ $# -ge 2 ]] || usage
		band_path="$2"
		shift 2
		;;
	--out)
		[[ $# -ge 2 ]] || usage
		out_root="$2"
		shift 2
		;;
	--corpus)
		[[ $# -ge 2 ]] || usage
		corpus_yaml="$2"
		shift 2
		;;
	--skip-canary)
		skip_canary="true"
		shift
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$record_path" && -n "$band_path" && -n "$out_root" ]] || usage
corpus_yaml="${corpus_yaml:-$BENCH_DIR/corpus/corpus.yaml}"

command -v python3 >/dev/null 2>&1 || {
	echo "replay-manifest: python3 not found on PATH" >&2
	exit 1
}
command -v "$RUN_ONE_BIN" >/dev/null 2>&1 || {
	echo "replay-manifest: run-one.sh binary not found (resolved as '$RUN_ONE_BIN'); build it or set RUN_ONE_BIN" >&2
	exit 1
}
[[ -x "$AGGREGATE_BIN" ]] || {
	echo "replay-manifest: aggregate-runs.sh not found or not executable: $AGGREGATE_BIN" >&2
	exit 1
}
[[ -f "$record_path" ]] || {
	echo "replay-manifest: --record not found: $record_path" >&2
	exit 1
}
[[ -f "$band_path" ]] || {
	echo "replay-manifest: --band not found: $band_path" >&2
	exit 1
}
[[ -f "$corpus_yaml" ]] || {
	echo "replay-manifest: --corpus not found: $corpus_yaml" >&2
	exit 1
}

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

SYNTHETIC_CORPUS="$WORKDIR/replay-corpus.yaml"
DRIFT_STATE="$WORKDIR/corpus-drift.json"

# --- REQ-F-027 preconditions + REQ-F-028 corpus_drift + ADR-F03-02
# synthetic corpus construction, all in one pass over the same parsed
# documents. On ANY precondition failure: every failing check is printed
# to stderr (never just the first), nothing is written to $SYNTHETIC_
# CORPUS/$DRIFT_STATE, and this prints nothing to stdout -- the field-
# count check below is what tells the caller apart from a real failure,
# matching run-one.sh's own item_fields convention. ---
readarray -d '' -t precheck_fields < <(python3 - "$record_path" "$corpus_yaml" "$band_path" "$BENCH_DIR/corpus/ledgers" "$SYNTHETIC_CORPUS" "$DRIFT_STATE" <<'PYEOF'
import copy
import json
import os
import sys

import yaml

record_path, corpus_yaml, band_path, ledgers_root, synth_out, drift_out = sys.argv[1:7]


def load_one_line_json(path, label):
    try:
        with open(path) as f:
            content = f.read()
    except OSError as e:
        sys.stderr.write("replay-manifest: cannot read %s: %s\n" % (label, e))
        sys.exit(1)
    lines = [ln for ln in (raw.strip() for raw in content.splitlines()) if ln]
    if len(lines) != 1:
        sys.stderr.write("replay-manifest: %s: expected exactly one JSON record line, found %d\n" % (label, len(lines)))
        sys.exit(1)
    try:
        doc = json.loads(lines[0])
    except json.JSONDecodeError as e:
        sys.stderr.write("replay-manifest: %s: unparseable JSON: %s\n" % (label, e))
        sys.exit(1)
    if not isinstance(doc, dict):
        sys.stderr.write("replay-manifest: %s: not a JSON object\n" % label)
        sys.exit(1)
    return doc


stored = load_one_line_json(record_path, "--record")
manifest = stored.get("manifest")
if not isinstance(manifest, dict):
    sys.stderr.write("replay-manifest: --record: missing manifest block\n")
    sys.exit(1)

item_id = manifest.get("item_id")
variant_id = manifest.get("variant_id")
rep = manifest.get("rep")
fixture_base_sha = manifest.get("fixture_base_sha")
corpus_schema_version = manifest.get("corpus_schema_version")
p2p_set = manifest.get("p2p_set")
timeout_cap_s = manifest.get("timeout_cap_s")

required = {
    "manifest.item_id": item_id,
    "manifest.variant_id": variant_id,
    "manifest.rep": rep,
    "manifest.fixture_base_sha": fixture_base_sha,
    "manifest.corpus_schema_version": corpus_schema_version,
    "manifest.p2p_set": p2p_set,
}
failures = []
for name, value in required.items():
    if value is None:
        failures.append("--record's %s is missing; cannot replay" % name)
if failures:
    for msg in failures:
        sys.stderr.write("replay-manifest: precondition failed: %s\n" % msg)
    sys.exit(1)

# --- (i) ledger directory (REQ-F-027) ---
ledger_dir = os.path.join(ledgers_root, fixture_base_sha)
if not os.path.isdir(ledger_dir):
    failures.append(
        "bench/corpus/ledgers/%s/ not found (path: %s) -- REQ-F-027 (i)" % (fixture_base_sha, ledger_dir)
    )

# --- (ii) item id resolves in --corpus (REQ-F-027) ---
try:
    with open(corpus_yaml) as f:
        corpus_doc = yaml.safe_load(f) or {}
except OSError as e:
    sys.stderr.write("replay-manifest: cannot read --corpus %s: %s\n" % (corpus_yaml, e))
    sys.exit(1)

live_item = None
for it in corpus_doc.get("items") or []:
    if it.get("id") == item_id:
        live_item = it
        break
if live_item is None:
    failures.append(
        "item id %r does not resolve in --corpus %s -- REQ-F-027 (ii)" % (item_id, corpus_yaml)
    )

# --- (iii) the --band aggregate has an entry for (item_id, variant_id)
# (REQ-F-027). baseline_id is the only place aggregate.json exposes its
# variant (spec.md "Aggregate document"): reconstruct it from the BAND's
# OWN provenance.fixture_base_sha/reps, never from the manifest's, so a
# precondition-(i) failure never also manufactures a precondition-(iii)
# failure. ---
try:
    with open(band_path) as f:
        band = json.load(f)
except OSError as e:
    sys.stderr.write("replay-manifest: cannot read --band %s: %s\n" % (band_path, e))
    sys.exit(1)
except json.JSONDecodeError as e:
    sys.stderr.write("replay-manifest: --band %s: unparseable JSON: %s\n" % (band_path, e))
    sys.exit(1)

tasks = band.get("tasks") or []
has_task = any(isinstance(t, dict) and t.get("item_id") == item_id for t in tasks)
provenance = band.get("provenance") or {}
band_fixture_sha = provenance.get("fixture_base_sha")
band_reps = provenance.get("reps")
baseline_id = band.get("baseline_id")
expected_baseline_id = None
if band_fixture_sha and isinstance(band_reps, int):
    expected_baseline_id = "%s-%s-r%d" % (variant_id, band_fixture_sha[:12], band_reps)
band_ok = has_task and baseline_id is not None and baseline_id == expected_baseline_id
if not band_ok:
    failures.append(
        "no band entry for (item_id=%s, variant_id=%s) in --band %s -- REQ-F-027 (iii)"
        % (item_id, variant_id, band_path)
    )

if failures:
    for msg in failures:
        sys.stderr.write("replay-manifest: precondition failed: %s\n" % msg)
    sys.exit(1)

# --- REQ-F-028: corpus_drift -- compared AFTER every precondition holds,
# so drift is recorded (never a precondition failure on its own: "the
# item id resolves" is the only corpus-membership precondition). One
# entry per divergent field, keyed by the manifest's own field name. ---
drift = []
live_fixture_sha = live_item.get("fixture_base_sha")
if live_fixture_sha != fixture_base_sha:
    drift.append(
        {"reason": "corpus_drift", "field": "fixture_base_sha", "expected": fixture_base_sha, "actual": live_fixture_sha}
    )
live_p2p_set = live_item.get("p2p_set")
if live_p2p_set != p2p_set:
    drift.append({"reason": "corpus_drift", "field": "p2p_set", "expected": p2p_set, "actual": live_p2p_set})
live_schema_version = corpus_doc.get("schema_version")
if live_schema_version != corpus_schema_version:
    drift.append(
        {
            "reason": "corpus_drift",
            "field": "corpus_schema_version",
            "expected": corpus_schema_version,
            "actual": live_schema_version,
        }
    )
with open(drift_out, "w") as f:
    json.dump(drift, f)

# --- ADR-F03-02: synthetic single-item corpus.yaml, written to a temp
# path -- bench/corpus/corpus.yaml itself is never touched. ---
corpus_dir = os.path.dirname(os.path.abspath(corpus_yaml))


def absolutize(rel_or_abs):
    # os.path.join returns an already-absolute second argument unchanged
    # (spec.md "Synthetic replay corpus"), so this is safe to apply
    # whether the source corpus stored the path as relative or absolute.
    return os.path.join(corpus_dir, rel_or_abs)


synth_item = copy.deepcopy(live_item)
synth_item["fixture_base_sha"] = fixture_base_sha
synth_item["p2p_set"] = p2p_set
synth_item["seed_path"] = absolutize(live_item["seed_path"])
f2p = copy.deepcopy(live_item.get("f2p") or {})
f2p["paths"] = [absolutize(p) for p in (f2p.get("paths") or [])]
synth_item["f2p"] = f2p

synth_fixture = copy.deepcopy(corpus_doc.get("fixture") or {})
synth_fixture["base_sha"] = fixture_base_sha

synth_doc = {
    "schema_version": corpus_schema_version,
    "fixture": synth_fixture,
    "p2p_sets": copy.deepcopy(corpus_doc.get("p2p_sets") or {}),
    "items": [synth_item],
}
with open(synth_out, "w") as f:
    yaml.safe_dump(synth_doc, f, sort_keys=False)

record_abspath = os.path.abspath(record_path)
for value in (item_id, variant_id, str(rep), str(timeout_cap_s) if timeout_cap_s is not None else "", record_abspath):
    sys.stdout.write(value)
    sys.stdout.write("\0")
PYEOF
)
[[ "${#precheck_fields[@]}" -eq 5 ]] || exit 1

item_id="${precheck_fields[0]}"
variant_id="${precheck_fields[1]}"
rep="${precheck_fields[2]}"
timeout_cap_s="${precheck_fields[3]}"
stored_record_abspath="${precheck_fields[4]}"

run_key="${item_id}::${variant_id}::rep${rep}"

mkdir -p "$out_root"
out_root_abs="$(cd "$out_root" && pwd)"
fresh_record_path="$out_root_abs/$item_id/$variant_id/rep-$rep/record.jsonl"

# Cheap zero-spend guard for REQ-F-025's "distinct --out root": a caller
# passing an --out that resolves the fresh record onto the stored one
# would otherwise let run-one.sh's own overwrite-refusal guard (or worse,
# a silent clobber) discover this mid-flight, after dispatch.
if [[ "$fresh_record_path" == "$stored_record_abspath" ]]; then
	echo "replay-manifest: --out resolves the fresh record path to the same file as --record ($fresh_record_path); pass a distinct --out root (REQ-F-025)" >&2
	exit 1
fi

dispatch_args=(--item "$item_id" --variant "$variant_id" --rep "$rep" --out "$out_root_abs" --corpus "$SYNTHETIC_CORPUS")
if [[ -n "$timeout_cap_s" ]]; then
	dispatch_args+=(--timeout "$timeout_cap_s")
fi
if [[ "$skip_canary" == "true" ]]; then
	dispatch_args+=(--skip-canary)
fi

echo "replay-manifest: dispatching $run_key (replay)" >&2
set +e
"$RUN_ONE_BIN" "${dispatch_args[@]}" </dev/null
rc=$?
set -e
if [[ "$rc" -ne 0 ]]; then
	echo "replay-manifest: RUN_ONE_BIN failed (exit $rc) for $run_key" >&2
	exit 1
fi

# --- T-E40-F03-007: aggregate the freshly dispatched record ON ITS OWN,
# in a $WORKDIR-scoped copy -- NEVER against $out_root_abs directly. A
# caller's --out root may be reused across multiple replay dispatches over
# time (REQ-F-025 only requires it be distinct from the STORED record's
# root, not that it holds exactly one item); aggregating the whole root
# would then risk tripping aggregate-runs.sh's own provenance-uniformity
# check across unrelated dispatches and suppressing tasks[] entirely
# (REQ-F-011), which would make this comparison silently unable to run.
# A scoped single-record copy is always exactly one record. ---
metrics_root="$WORKDIR/metrics-root"
mkdir -p "$metrics_root/$item_id/$variant_id/rep-$rep"
cp "$fresh_record_path" "$metrics_root/$item_id/$variant_id/rep-$rep/record.jsonl"

fresh_aggregate_path="$WORKDIR/fresh-aggregate.json"
aggregate_err="$WORKDIR/aggregate-stderr.log"
set +e
"$AGGREGATE_BIN" --root "$metrics_root" >"$fresh_aggregate_path" 2>"$aggregate_err"
agg_rc=$?
set -e
# aggregate-runs.sh still prints its full document on an `anomaly` exit
# (non-zero, doc still written) -- only a structurally invalid/non-uniform
# input writes nothing to stdout (REQ-F-010/REQ-F-011). A single,
# already I-02-shaped record can be non-uniform or structurally invalid
# only if run-one.sh itself produced a malformed record, which is a real
# failure worth surfacing loudly (REQ-N-005), not silently swallowed.
if [[ ! -s "$fresh_aggregate_path" ]]; then
	echo "replay-manifest: aggregate-runs.sh produced no output for the replayed record $run_key (exit $agg_rc): $(cat "$aggregate_err")" >&2
	exit 1
fi

# --- Assemble the verification document (spec.md "Verification document").
# REQ-F-028's corpus_drift (from the dispatch half above) and
# REQ-F-029's variant_bundle_drift/model_version_drift share one
# reasons[] list and vocabulary. ANY reason present yields `invalid` and
# skips the metric comparison entirely -- ADR-F03-05: the inputs were not
# reproduced, so a metric comparison would be meaningless. Only with
# zero reasons does REQ-F-030's per-metric comparison run, yielding
# `pass` (every comparable metric inside its published interval) or
# `fail` (at least one outside, named with its replayed value and
# interval; every other metric still individually scored in metrics[]). ---
python3 - "$stored_record_abspath" "$fresh_record_path" "$band_path" "$DRIFT_STATE" "$fresh_aggregate_path" "$item_id" <<'PYEOF'
import json
import sys

stored_path, replayed_path, band_path, drift_path, fresh_aggregate_path, item_id = sys.argv[1:7]

IDENTITY_FIELDS = ("fixture_base_sha", "corpus_schema_version", "p2p_set", "variant_bundle_sha256", "model_ids")

# REQ-F-029: these two fields are compared post-dispatch, on top of
# REQ-F-028's corpus_drift (already recorded to drift_path by the dispatch
# half). Named here, not folded into IDENTITY_FIELDS above, because each
# gets its own reasons[] entry/vocabulary word (variant_bundle_drift,
# model_version_drift) rather than a generic "differs" bit.
POST_DISPATCH_DRIFT_FIELDS = (
    ("variant_bundle_sha256", "variant_bundle_drift"),
    ("model_ids", "model_version_drift"),
)


def load_manifest(path):
    with open(path) as f:
        content = f.read()
    lines = [ln for ln in (raw.strip() for raw in content.splitlines()) if ln]
    rec = json.loads(lines[0])
    return rec.get("manifest", {})


def identity(path, manifest):
    d = {"run_key": manifest.get("run_key"), "path": path}
    for field in IDENTITY_FIELDS:
        if field in manifest:
            d[field] = manifest[field]
    return d


with open(band_path) as f:
    band = json.load(f)
with open(drift_path) as f:
    reasons = json.load(f)

stored_manifest = load_manifest(stored_path)
replayed_manifest = load_manifest(replayed_path)

for field, reason_name in POST_DISPATCH_DRIFT_FIELDS:
    expected = stored_manifest.get(field)
    actual = replayed_manifest.get(field)
    if expected != actual:
        reasons.append({"reason": reason_name, "field": field, "expected": expected, "actual": actual})

doc = {
    "baseline_id": band.get("baseline_id"),
    "input_digest": band.get("input_digest"),
    "stored": identity(stored_path, stored_manifest),
    "replayed": identity(replayed_path, replayed_manifest),
}

if reasons:
    doc["reasons"] = reasons
    doc["verdict"] = "invalid"
else:
    # --- REQ-F-030: per-metric comparison. band_metrics is the PUBLISHED
    # band (this item's tasks[] entry in --band); fresh_metrics is the
    # single freshly-replayed record's own (n=1) statistics, computed by
    # invoking aggregate-runs.sh (never re-derived here). A metric is
    # comparable only when the band published an interval/accept_set for
    # it AND the fresh side actually carries a value -- an absent metric
    # is never read as zero (REQ-N-005), it is simply left out of
    # metrics[] rather than fabricated. ---
    band_task = next((t for t in band.get("tasks") or [] if t.get("item_id") == item_id), None)
    band_metrics = (band_task or {}).get("metrics") or {}

    with open(fresh_aggregate_path) as f:
        fresh_agg = json.load(f)
    fresh_task = next((t for t in fresh_agg.get("tasks") or [] if t.get("item_id") == item_id), None)
    fresh_metrics = (fresh_task or {}).get("metrics") or {}

    metrics_out = []
    for metric_id in sorted(band_metrics):
        band_block = band_metrics[metric_id]
        fresh_block = fresh_metrics.get(metric_id) or {}

        if "accept_set" in band_block:
            if "true_count" not in fresh_block:
                continue
            replayed_value = bool(fresh_block["true_count"])
            accept_set = band_block["accept_set"]
            verdict = "pass" if replayed_value in accept_set else "fail"
            metrics_out.append(
                {
                    "metric": metric_id,
                    "replayed_value": replayed_value,
                    "accept_set": accept_set,
                    "verdict": verdict,
                }
            )
        elif "accept_lo" in band_block:
            if "mean" not in fresh_block:
                continue
            replayed_value = fresh_block["mean"]
            accept_lo, accept_hi = band_block["accept_lo"], band_block["accept_hi"]
            verdict = "pass" if accept_lo <= replayed_value <= accept_hi else "fail"
            metrics_out.append(
                {
                    "metric": metric_id,
                    "replayed_value": replayed_value,
                    "accept_lo": accept_lo,
                    "accept_hi": accept_hi,
                    "verdict": verdict,
                }
            )
        # else: band published no interval for this metric (insufficient_reps
        # on the band side) -- nothing to compare against, skipped.

    doc["metrics"] = metrics_out
    doc["verdict"] = "fail" if any(m["verdict"] == "fail" for m in metrics_out) else "pass"

print(json.dumps(doc, indent=2, sort_keys=True))
# spec.md Interface contracts: verdict `pass` -> exit 0; `fail`/`invalid` -> non-zero.
sys.exit(0 if doc["verdict"] == "pass" else 1)
PYEOF
