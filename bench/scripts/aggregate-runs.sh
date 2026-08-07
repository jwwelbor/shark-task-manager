#!/usr/bin/env bash
# aggregate-runs.sh --root <artifact_root> [--variant <id>]
#
# T-E40-F03-003 (spec.md ADR-F03-01, ADR-F03-03; test-plan.md TC-018).
# THIS TASK'S SLICE only: pinned-glob enumeration, per-record structural
# validation, classification (complete/explained_absence/anomaly), and
# five-field provenance uniformity (REQ-F-011) -- emitting the aggregate's
# `schema_version`, `provenance`, `inventory`, `outcomes`, and `anomalies[]`
# blocks. Statistics/bands (`tasks[]`, `corpus`, `flags`, `input_digest`,
# `baseline_id`) are T-E40-F03-004's extension of this same file -- not
# implemented here (task spec "Notes for Agent": "Classification only -- no
# statistics/bands here").
#
# A pure function of an artifact root (ADR-F03-01): prints one JSON
# document to stdout, diagnostics to stderr, never touches anything but the
# record.jsonl files the pinned glob below matches (REQ-F-007 -- no
# batch-log.jsonl, no scratch project, no database, no network).
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
# Provenance uniformity (REQ-F-011): the five manifest fields (model_ids,
# fixture_base_sha, variant_bundle_sha256, corpus_schema_version,
# shark_version) are compared across every contributing record's PRESENT
# value only -- a record where the field is legitimately absent (e.g.
# manifest.model_ids on a record whose timeout fired before any stage
# resolved modelUsage, bench/README.md "I-02 record schema field
# reference") never counts as a divergence by itself; two or more DISTINCT
# present values for the same field does. Non-uniform provenance still
# prints the full document (provenance.uniform=false, divergences[] naming
# both differing values, plus this task's own classification/inventory/
# outcomes/anomalies blocks, which are independent of provenance) and exits
# non-zero -- it is the STATISTICAL blocks (tasks[]/corpus, T-E40-F03-004's
# scope, not implemented by this script) that must never be "published as a
# result" over an unpinned batch (REQ-F-011, AC-11: "no other block
# (tasks[], corpus) is populated as if the batch were valid" -- inventory/
# outcomes/anomalies are not named by that sentence and remain published).
#
# Exit status (spec.md "Interface contracts", this task's slice):
#   0        every record complete/explained_absence, provenance uniform
#   non-zero any anomaly (full document still printed, anomalies named);
#            OR a structurally invalid record (unsupported schema_version,
#            unparseable JSON, missing manifest/outcome) -- prints NOTHING
#            to stdout, names the offending file on stderr (REQ-F-010);
#            OR non-uniform provenance (full document still printed, see
#            above)
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
import json
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


def is_explained(rec):
    """explained_absence rule (spec.md "Record classification"), verbatim:
    outcome == "timeout" (no post-run phase runs on that path); or
    quality.toolchain_guard != "pass"; or errors[] carries a
    postrun_check_aborted entry."""
    if rec.get("outcome") == "timeout":
        return True
    quality = rec.get("quality")
    if isinstance(quality, dict) and quality.get("toolchain_guard") not in (None, "pass"):
        return True
    for err in rec.get("errors") or []:
        if isinstance(err, dict) and err.get("kind") == ERROR_KIND_POSTRUN_ABORTED:
            return True
    return False


def classify(rec):
    missing = [f for f in EXPECTED_FAMILIES if f not in rec]
    if not missing:
        return "complete", missing
    if is_explained(rec):
        return "explained_absence", missing
    return "anomaly", missing


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

# --- classification + inventory + outcomes + anomalies ---
inventory = {}
outcome_counts = {}
anomalies = []
anomaly_count = 0
timeout_count = 0
seen_run_keys = {}

for path, rec in records:
    run_key = rec["manifest"]["run_key"]
    if run_key in seen_run_keys:
        fail("duplicate manifest.run_key %r across %s and %s" % (run_key, seen_run_keys[run_key], path))
    seen_run_keys[run_key] = path

    classification, missing = classify(rec)
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

aggregate = {
    "schema_version": AGGREGATE_SCHEMA_VERSION,
    "provenance": provenance,
    "inventory": inventory,
    "outcomes": outcomes_block,
    "anomalies": anomalies,
}

# The one and only stdout write (REQ-N-004: sorted keys, fixed list order,
# no timestamp -- json.dumps(sort_keys=True) sorts every nested dict's keys
# too, and anomalies[]/divergences[]/values[] were explicitly sorted above).
print(json.dumps(aggregate, indent=2, sort_keys=True))

if not uniform:
    sys.exit(1)
if anomaly_count > 0:
    sys.exit(1)
sys.exit(0)
PYEOF
