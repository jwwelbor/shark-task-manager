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
#   (i)   bench/corpus/ledgers/<manifest.fixture_base_sha>/{tests,lint}.json
#         both EXIST as files -- resolved from THIS SCRIPT's own location
#         ($BENCH_DIR/corpus/ledgers/<sha>/), never from --corpus's
#         directory (spec.md: the base-SHA ledger paths run-one.sh reads
#         are derived from its own location, not the corpus file's -- a
#         synthetic corpus in a temp dir still resolves the real
#         ledgers). The directory's mere existence is not the check
#         (UAT F-10): a partially-pruned ledger dir missing either file
#         fails loudly, naming which file is missing.
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
import re
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

# --- R2-F-13 fix (code-review-20260808-235300-E40-F03.md /
# uat-20260808-231500-E40-F03.md): item_id/variant_id here come from the
# STORED RECORD -- never the operator -- and this script's bash half
# later interpolates both, unvalidated, into fresh_record_path
# (out_root_abs/item_id/variant_id/rep-N/record.jsonl). A forged or
# corrupted record must not be able to steer that path outside --out.
# Checked here, at parse time, and batched into the SAME failures[] list
# as (i)/(ii)/(iii) below -- consistent with REQ-F-027's "every failing
# check reported in one pass" convention -- before any of those checks
# construct or touch a path derived from either value. The grammar
# structurally forbids '/' (no traversal component) and a leading '.' (so
# neither '.' nor '..' can ever match). ---
SAFE_SLUG_RE = re.compile(r'^[A-Za-z0-9][A-Za-z0-9._-]*$')
for field_name, field_value in (("manifest.item_id", item_id), ("manifest.variant_id", variant_id)):
    if not isinstance(field_value, str) or not SAFE_SLUG_RE.match(field_value):
        failures.append(
            "%s = %r is not a safe path identifier (must match %s -- no '/', no leading '.') -- REQ-F-006"
            % (field_name, field_value, SAFE_SLUG_RE.pattern)
        )

# --- (i) ledger directory's retained artifacts (REQ-F-027, UAT F-10):
# the directory's mere existence is not the requirement -- REQ-F-027 (i)
# names bench/corpus/ledgers/<sha>/{tests,lint}.json specifically, so a
# partially-pruned ledger dir (container present, one or both artifacts
# missing) must fail loudly and name which file is missing, not pass on
# an os.path.isdir check alone. ---
ledger_dir = os.path.join(ledgers_root, fixture_base_sha)
for ledger_file in ("tests.json", "lint.json"):
    ledger_path = os.path.join(ledger_dir, ledger_file)
    if not os.path.isfile(ledger_path):
        failures.append(
            "bench/corpus/ledgers/%s/%s not found (path: %s) -- REQ-F-027 (i)"
            % (fixture_base_sha, ledger_file, ledger_path)
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
# NEW-1 (UAT round 3, uat-20260809-013000-E40-F03.md): out_root_abs MUST be
# canonicalized the same way run-batch.sh's own out_root_canon is (realpath
# -m, symlinks resolved) -- the prior `cd "$out_root" && pwd` form is
# LOGICAL (bash pwd's default -L, symlinks preserved), which disagreed with
# the fully-canonicalized comparison assert_within_out_root() performs
# below and wrongly rejected every legitimate --out that resolves through
# a symlink.
out_root_abs="$(realpath -m -- "$out_root")"
fresh_record_path="$out_root_abs/$item_id/$variant_id/rep-$rep/record.jsonl"

# R2-F-13 belt-and-braces containment check (in addition to the safe-slug
# grammar check on item_id/variant_id above): item_id/variant_id are
# already grammar-validated, so this should never fire in practice, but
# the constructed path is still canonicalized with `realpath -m` and
# asserted beneath out_root_abs before any mkdir/cp/dispatch below uses
# it. assert_within_out_root() is lifted into lib/path-safety.sh, shared
# with run-batch.sh's own R2-F-13 fix, so the two implementations cannot
# drift apart again.
out_root_canon="$out_root_abs"
# shellcheck source=lib/path-safety.sh
source "$SCRIPT_DIR/lib/path-safety.sh"
assert_within_out_root "$fresh_record_path" || exit 1

# Cheap zero-spend guard for REQ-F-025's "distinct --out root": a caller
# passing an --out that resolves the fresh record onto the stored one
# would otherwise let run-one.sh's own overwrite-refusal guard (or worse,
# a silent clobber) discover this mid-flight, after dispatch.
#
# Sweep (NEW-1's fix guidance): stored_record_abspath comes from Python's
# os.path.abspath(), which normalizes but never resolves symlinks --
# comparing it directly against the now-canonicalized fresh_record_path
# would reintroduce the same two-namespaces defect class this task exists
# to fix. Canonicalize both sides with the same realpath -m form before
# comparing.
stored_record_canon="$(realpath -m -- "$stored_record_abspath")"
if [[ "$(realpath -m -- "$fresh_record_path")" == "$stored_record_canon" ]]; then
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
python3 - "$stored_record_abspath" "$fresh_record_path" "$band_path" "$DRIFT_STATE" "$fresh_aggregate_path" "$item_id" "$agg_rc" <<'PYEOF'
import json
import sys

stored_path, replayed_path, band_path, drift_path, fresh_aggregate_path, item_id, agg_rc_str = sys.argv[1:8]
agg_rc = int(agg_rc_str)

IDENTITY_FIELDS = (
    "run_key",
    "fixture_base_sha",
    "corpus_schema_version",
    "p2p_set",
    "variant_bundle_sha256",
    "model_ids",
    "shark_version",
)

# UAT R2-F-6 (docs/review/.../uat-20260808-231500-E40-F03.md): round-1's F-6
# fix hand-picked only variant_bundle_sha256/model_ids (POST_DISPATCH_DRIFT_
# FIELDS) plus a specially-cased run_key, leaving fixture_base_sha,
# corpus_schema_version, and p2p_set displayed (via identity() below) but
# NEVER compared -- a fresh record could carry a completely different
# fixture_base_sha and still verdict pass. IDENTITY_FIELDS above is now the
# SINGLE set that drives both comparison and display: every field is
# compared here, and identity() below can only display what this loop
# already compared. variant_bundle_sha256/model_ids/run_key keep their own
# named reason (REQ-F-029, REQ-F-025); every other field gets a generic
# identity_mismatch reason naming the field.
IDENTITY_REASON_OVERRIDES = {
    "run_key": "run_key_mismatch",
    "variant_bundle_sha256": "variant_bundle_drift",
    "model_ids": "model_version_drift",
}


def load_manifest(path):
    with open(path) as f:
        content = f.read()
    lines = [ln for ln in (raw.strip() for raw in content.splitlines()) if ln]
    rec = json.loads(lines[0])
    return rec.get("manifest", {})


def identity(path, manifest):
    d = {"path": path}
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

# UAT F-6 / R2-F-6: loop over the FULL IDENTITY_FIELDS set (not two
# hand-picked fields plus a special-cased run_key) so every displayed
# identity field is also enforced. A mismatch on any of them -- a
# producer-side rep-renumbering bug (run_key), a diverged fixture
# (fixture_base_sha), a stale corpus schema, a different p2p_set, a bundle
# hash drift, a model-ID drift, or a shark binary version skew -- yields
# `invalid`, never a reproducibility pass with the differing values merely
# printed side by side.
for field in IDENTITY_FIELDS:
    expected = stored_manifest.get(field)
    actual = replayed_manifest.get(field)
    if expected != actual:
        reason_name = IDENTITY_REASON_OVERRIDES.get(field, "identity_mismatch")
        reasons.append({"reason": reason_name, "field": field, "expected": expected, "actual": actual})

# UAT F-1 (CRITICAL): a non-zero aggregate-runs.sh exit for the freshly
# replayed record means the aggregator itself refused to treat it as a
# comparable datum (its own `anomaly` classification, REQ-F-009) -- the
# `:452` empty-output guard above catches only a structurally invalid/
# non-uniform input, never this. Checked here, not skipped, so this
# reason lands in the SAME reasons[]/invalid path as every other identity
# mismatch above (never a metric comparison over data the aggregator
# itself would not certify).
if agg_rc != 0:
    reasons.append(
        {
            "reason": "aggregator_anomaly",
            "detail": "aggregate-runs.sh exited %d for the replayed run -- it could not be certified as a comparable datum" % agg_rc,
        }
    )

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

    # UAT F-1, fix guidance #2: a band-published metric the fresh side
    # cannot supply (dropped family, gate never executed, invalid value
    # type, ...) is NEVER silently skipped -- it gets an explicit
    # `not_comparable` entry naming the fresh block's own n/excluded[]
    # reasons, so metrics_out accounts for every metric the band
    # published. "Case (b) is the trap": an explained_absence record can
    # pass agg_rc == 0 while still dropping comparable band metrics this
    # way, so this check is independent of (and in addition to) the
    # agg_rc check above.
    metrics_out = []
    not_comparable_ids = []
    band_no_interval_ids = []
    for metric_id in sorted(band_metrics):
        band_block = band_metrics[metric_id]
        fresh_block = fresh_metrics.get(metric_id) or {}

        if "accept_set" in band_block:
            if "true_count" not in fresh_block:
                not_comparable_ids.append(metric_id)
                metrics_out.append(
                    {
                        "metric": metric_id,
                        "verdict": "not_comparable",
                        "n": fresh_block.get("n"),
                        "excluded": fresh_block.get("excluded") or [],
                    }
                )
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
                not_comparable_ids.append(metric_id)
                metrics_out.append(
                    {
                        "metric": metric_id,
                        "verdict": "not_comparable",
                        "n": fresh_block.get("n"),
                        "excluded": fresh_block.get("excluded") or [],
                    }
                )
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
        else:
            # UAT R2-F-1 (HIGH, same loop as round-1's CRITICAL F-1 fix --
            # that fix handled the fresh side unable to supply a metric;
            # this is the band-side equivalent, left unhandled): the band
            # published NO interval/accept_set for this metric at all (e.g.
            # insufficient_reps on the band side) -- previously silently
            # dropped from metrics_out with no trace. Never skipped: an
            # explicit not_comparable entry naming the BAND block's own
            # n/excluded[] (there is no fresh-side value to blame -- the
            # band itself had nothing to compare against).
            #
            # Kept in its OWN list (band_no_interval_ids), not
            # not_comparable_ids: unlike the two branches above -- where the
            # band published a claim (an interval) the fresh side failed to
            # answer, REQ-N-005's "an absent metric is never read as zero"
            # territory -- here the band never made a claim about this
            # metric at all (AC-13/REQ-F-016's own insufficient_reps case,
            # already surfaced at publish time by report-baseline.sh's
            # flags.insufficient_reps). There is nothing for a replay to
            # have violated, so this must not by itself force `invalid`
            # (AC-27's `pass` must stay reachable for a band with ANY
            # insufficient_reps metric) -- it is still traced in
            # reasons[]/metrics[], never silently dropped.
            band_no_interval_ids.append(metric_id)
            metrics_out.append(
                {
                    "metric": metric_id,
                    "verdict": "not_comparable",
                    "reason": "band_no_interval",
                    "n": band_block.get("n"),
                    "excluded": band_block.get("excluded") or [],
                }
            )

    # Structural invariant (UAT R2-F-1 fix guidance): metrics_out must
    # account for every band-published metric -- no branch above may skip
    # appending, so this can never trip once the else: above is in place.
    #
    # NEW-6 (uat-20260809-013000-E40-F03.md, round-3): this was a bare
    # `assert`, elided entirely under PYTHONOPTIMIZE=1 (Python strips
    # `assert` statements -- not just their messages -- whenever `-O`/
    # PYTHONOPTIMIZE is set), silently turning a runtime invariant into a
    # no-op in that mode. An explicit conditional writing to stderr and
    # calling sys.exit(1) -- this file's own non-elidable hard-failure
    # mechanism, the same one every load_one_line_json/precondition/--band
    # failure above already uses -- can never be compiled away.
    if len(metrics_out) != len(band_metrics):
        sys.stderr.write(
            "replay-manifest: metrics_out=%d entries, band_metrics=%d keys -- a band-published metric was dropped silently\n"
            % (len(metrics_out), len(band_metrics))
        )
        sys.exit(1)

    doc["metrics"] = metrics_out
    if band_no_interval_ids:
        doc.setdefault("reasons", []).append({"reason": "band_no_interval", "metrics": band_no_interval_ids})

    scored_metrics = [m for m in metrics_out if m["verdict"] in ("pass", "fail")]

    if not_comparable_ids:
        doc.setdefault("reasons", []).append({"reason": "metric_not_comparable", "metrics": not_comparable_ids})
        doc["verdict"] = "invalid"
    elif not scored_metrics:
        # No metric this item published actually got a pass/fail score --
        # either metrics_out is empty (band published nothing for this
        # item) or every entry is band_no_interval (the band published
        # metrics, but none carried an interval). Either way, never read
        # as a vacuous pass (REQ-N-005).
        doc.setdefault("reasons", []).append({"reason": "no_comparable_metrics"})
        doc["verdict"] = "invalid"
    else:
        doc["verdict"] = "fail" if any(m["verdict"] == "fail" for m in metrics_out) else "pass"

print(json.dumps(doc, indent=2, sort_keys=True))
# spec.md Interface contracts: verdict `pass` -> exit 0; `fail`/`invalid` -> non-zero.
sys.exit(0 if doc["verdict"] == "pass" else 1)
PYEOF
