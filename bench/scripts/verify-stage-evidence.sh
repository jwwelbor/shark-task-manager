#!/usr/bin/env bash
# verify-stage-evidence.sh <bundle_dir>
#
# The single named owner of I-05 bundle validation arithmetic
# (spec.md Component-changes row for verify-stage-evidence.sh) -- the
# discipline eval-predicate.sh established for I-01/I-04's final_predicate
# and diff-ledgers.sh established for I-02's own ledger comparisons, carried
# forward here so F08, F09, and F10 invoke this script rather than
# re-deriving I-05's semantics themselves.
#
# T-E40-F06-004 stands up this scaffold and implements ONLY the REQ-F-005
# time_ledger reconciliation check (ADR-F06-06). Later tasks MODIFY this
# same script to add candidate identity (T-E40-F06-005, REQ-F-006), artifact
# records (T-E40-F06-006, REQ-F-008), evaluator_access ordering
# (T-E40-F06-007, REQ-F-012), and stop-outcome eligibility
# (T-E40-F06-008, REQ-F-014) -- the `<bundle_dir>` argument contract is
# deliberately the ONLY thing this task fixes for those tasks to build on;
# nothing else about this script's internal structure is a promise.
#
# Reads the bundle's `bundle.json` stage index (`<bundle_dir>/bundle.json`
# `stages[]`, spec.md "Bundle layout (I-05)"), then for each indexed stage
# snapshot (`<bundle_dir>/<snapshot_path>`) validates its `time_ledger`
# object (spec.md "Stage snapshot (I-05)" `time_ledger` row):
#
#   1. every category key under `intervals` resolves against the closed
#      `interval_category` vocabulary in `bench/evidence/i05-schema.yaml`
#      (REQ-F-017, AC-T5) -- read from that file at call time, never
#      embedded here as a private list, mirroring the root-name-sourcing
#      discipline verify-evidence-roots.sh already established for REQ-F-017.
#   2. every interval is a genuine half-open `[start, end)` span (start <
#      end) fully contained in `[stage_start, stage_end)` -- an interval
#      that escapes the stage window is rejected naming the interval and
#      the window (REQ-F-005).
#   3. no two intervals overlap, across ANY two categories -- rejected
#      naming both offending categories and both intervals (REQ-F-005,
#      AC-T2 case (ii)). Checked by sorting the union of every category's
#      intervals by start and comparing only adjacent pairs: if intervals A
#      and C (non-adjacent in sorted order) overlapped while neither
#      overlapped its immediate neighbor B, B's start would have to fall
#      inside BOTH A's and C's spans simultaneously, which would already
#      make A and B (an adjacent pair) overlap -- so checking adjacent pairs
#      after sorting by start catches every pairwise overlap in the set,
#      not just adjacent ones.
#   4. the union of all intervals reconciles to the stage window within
#      `reconciliation_epsilon_ns`: `residual_ns = window_ns - covered_ns`
#      (guaranteed >= 0 once window-containment and non-overlap both hold)
#      must be <= epsilon, else rejected naming the residual magnitude
#      (REQ-F-005, AC-T2 case (iv)).
#
# A residual within epsilon is accepted WITHOUT ever adding it to any
# category's total -- in particular never to `provider_active` (REQ-F-005,
# REQ-NF-007, AC-T3/AC-T4): this script's arithmetic only ever sums
# intervals actually present in the ledger, so an unattributed residual
# structurally cannot inflate any category's reported total. The success
# JSON's `residual_category` field always reads `"unclassified"`, naming
# the architecture's binding rule explicitly rather than leaving it
# implicit.
#
# Prints one fixed-order JSON document to stdout on success (bundle path,
# then `stages[]` sorted by `dispatch_ordinal`, each stage's own keys in
# sorted order via `sort_keys=True` -- "Emit fixed-order JSON (sorted by
# dispatch_ordinal, then field name)", REQ-NF-004 byte-identical verdicts).
# On a validation violation, prints nothing to stdout and one line to
# stderr naming the offending stage, category pair, interval, or residual
# magnitude, then exits 1. Exit 2 is reserved for a script/usage/authoring
# error (bad args, missing files, a bundle.json/snapshot missing a required
# field this script depends on) -- distinct from a normal, informative
# rejection verdict, matching verify-evidence-roots.sh's exit-code
# convention.
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
I05_SCHEMA="$BENCH_DIR/evidence/i05-schema.yaml"

usage() {
	echo "usage: verify-stage-evidence.sh <bundle_dir>" >&2
	exit 2
}

[[ $# -eq 1 ]] || usage
bundle_dir="$1"

[[ -d "$bundle_dir" ]] || {
	echo "verify-stage-evidence: bundle dir not found: $bundle_dir" >&2
	exit 2
}
[[ -f "$I05_SCHEMA" ]] || {
	echo "verify-stage-evidence: i05-schema.yaml not found: $I05_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-stage-evidence: python3 not found on PATH" >&2
	exit 2
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "verify-stage-evidence: python3 module 'yaml' (PyYAML) not available" >&2
	exit 2
}

bundle_dir_abs="$(cd "$bundle_dir" && pwd)"

python3 - "$bundle_dir_abs" "$I05_SCHEMA" <<'PYEOF'
import json
import os
import sys

import yaml

bundle_dir, i05_schema_path = sys.argv[1:3]


class ScriptError(RuntimeError):
    """A prerequisite this validator depends on could not be resolved at
    all -- malformed/missing input distinct from a normal rejection
    verdict. Reported on stderr and mapped to exit 2."""


class LedgerViolation(RuntimeError):
    """A real, informative REQ-F-005 rejection verdict -- mapped to exit 1,
    never silently swallowed, never conflated with a script/usage error."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


def load_yaml(path, label):
    with open(path) as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        raise ScriptError(f"{label} is not a YAML mapping: {path}")
    return data


def load_json(path, label):
    try:
        with open(path) as f:
            return json.load(f)
    except (OSError, json.JSONDecodeError) as exc:
        raise ScriptError(f"{label}: could not read/parse {path}: {exc}") from exc


def validate_time_ledger(stage_key, dispatch_ordinal, ledger, known_categories):
    """Implements REQ-F-005 / ADR-F06-06 for one stage's time_ledger.
    Raises LedgerViolation for a real rejection, ScriptError for malformed
    input this script cannot evaluate at all. Returns the JSON-serializable
    verdict dict on acceptance."""
    required = {"stage_start", "stage_end", "reconciliation_epsilon_ns", "intervals"}
    missing = sorted(required - ledger.keys())
    if missing:
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: time_ledger missing required field(s): {missing}"
        )

    stage_start = ledger["stage_start"]
    stage_end = ledger["stage_end"]
    epsilon = ledger["reconciliation_epsilon_ns"]
    intervals_by_category = ledger["intervals"]

    if not all(isinstance(v, int) for v in (stage_start, stage_end, epsilon)):
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: stage_start/stage_end/reconciliation_epsilon_ns must be integers"
        )
    if stage_end <= stage_start:
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: stage_end ({stage_end}) must be > stage_start ({stage_start})"
        )
    if epsilon < 0:
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: reconciliation_epsilon_ns must be >= 0, got {epsilon}"
        )
    if not isinstance(intervals_by_category, dict):
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: time_ledger.intervals must be an object keyed by category"
        )

    # REQ-F-017 / AC-T5: category keys resolve against the closed vocabulary
    # read from i05-schema.yaml at call time -- never a private copy here.
    unknown = sorted(set(intervals_by_category) - set(known_categories))
    if unknown:
        raise LedgerViolation(
            "unknown_interval_category",
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} category={unknown[0]}",
        )

    flat = []  # [(category, start, end), ...]
    for category, spans in intervals_by_category.items():
        if not isinstance(spans, list):
            raise ScriptError(
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: intervals.{category} must be a list of [start, end) pairs"
            )
        for span in spans:
            if not (isinstance(span, list) and len(span) == 2 and all(isinstance(x, int) for x in span)):
                raise ScriptError(
                    f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: intervals.{category} has a malformed [start, end) pair: {span!r}"
                )
            start, end = span
            if start >= end:
                raise ScriptError(
                    f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: intervals.{category} interval [{start},{end}) is not a valid half-open span (start must be < end)"
                )
            flat.append((category, start, end))

    # Window containment -- an interval escaping [stage_start, stage_end)
    # is rejected naming the interval and the window (REQ-F-005, AC-T2 (iii)).
    for category, start, end in flat:
        if start < stage_start or end > stage_end:
            raise LedgerViolation(
                "ledger_window_escape",
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} category={category} "
                f"interval=[{start},{end}) stage_window=[{stage_start},{stage_end})",
            )

    # Pairwise overlap across ALL categories combined -- sort by start and
    # check only adjacent pairs (see header comment for why this suffices).
    flat_sorted = sorted(flat, key=lambda t: (t[1], t[2]))
    for i in range(len(flat_sorted) - 1):
        cat_a, start_a, end_a = flat_sorted[i]
        cat_b, start_b, end_b = flat_sorted[i + 1]
        if start_b < end_a:
            raise LedgerViolation(
                "ledger_overlap",
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} "
                f"category_a={cat_a} interval_a=[{start_a},{end_a}) "
                f"category_b={cat_b} interval_b=[{start_b},{end_b})",
            )

    # Reconciliation. Window containment + non-overlap together guarantee
    # covered_ns <= window_ns, so residual_ns is always >= 0 here.
    window_ns = stage_end - stage_start
    covered_ns = sum(end - start for _, start, end in flat)
    residual_ns = window_ns - covered_ns
    if residual_ns > epsilon:
        raise LedgerViolation(
            "ledger_non_reconciling",
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} residual_ns={residual_ns} epsilon_ns={epsilon}",
        )

    category_totals_ns = {category: 0 for category in known_categories}
    for category, start, end in flat:
        category_totals_ns[category] += end - start

    return {
        "epsilon_ns": epsilon,
        "category_totals_ns": category_totals_ns,
        # REQ-F-005 / REQ-NF-007: an accepted residual is never folded into
        # any category's own total (see category_totals_ns above, which
        # sums only intervals actually present in the ledger) -- naming it
        # "unclassified" here makes that binding rule explicit rather than
        # leaving it implicit in the arithmetic.
        "residual_category": "unclassified",
        "residual_ns": residual_ns,
        "result": "accepted",
    }


def resolve_within(root, rel_path, label):
    """Resolves a bundle-relative path and enforces containment, mirroring
    verify-evidence-roots.sh's resolve_in_evaluator_root (REQ-NF-005)."""
    if os.path.isabs(rel_path):
        raise ScriptError(f"{label}: absolute path not allowed: {rel_path!r}")
    root_real = os.path.realpath(root)
    candidate = os.path.realpath(os.path.join(root, rel_path))
    if candidate != root_real and not candidate.startswith(root_real + os.sep):
        raise ScriptError(f"{label}: resolved path escapes bundle_dir: {rel_path!r} -> {candidate!r}")
    return candidate


def main():
    schema = load_yaml(i05_schema_path, "i05-schema.yaml")
    known_categories = schema.get("interval_category")
    if not isinstance(known_categories, list) or not known_categories:
        raise ScriptError(f"i05-schema.yaml declares no interval_category vocabulary: {i05_schema_path}")

    bundle_json_path = os.path.join(bundle_dir, "bundle.json")
    if not os.path.isfile(bundle_json_path):
        raise ScriptError(f"bundle.json not found: {bundle_json_path}")
    bundle = load_json(bundle_json_path, "bundle.json")

    stages_index = bundle.get("stages")
    if not isinstance(stages_index, list) or not stages_index:
        raise ScriptError(f"bundle.json has no non-empty stages[] array: {bundle_json_path}")

    stages_sorted = sorted(stages_index, key=lambda entry: entry.get("dispatch_ordinal", 0))

    stage_results = []
    for entry in stages_sorted:
        if "dispatch_ordinal" not in entry or "stage_key" not in entry or "snapshot_path" not in entry:
            raise ScriptError(f"bundle.json stages[] entry missing dispatch_ordinal/stage_key/snapshot_path: {entry!r}")
        dispatch_ordinal = entry["dispatch_ordinal"]
        stage_key = entry["stage_key"]
        snapshot_path = resolve_within(bundle_dir, entry["snapshot_path"], f"stages[] entry {stage_key!r}")
        if not os.path.isfile(snapshot_path):
            raise ScriptError(f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: snapshot file not found: {snapshot_path}")

        snapshot = load_json(snapshot_path, f"stage snapshot ({stage_key})")
        ledger = snapshot.get("time_ledger")
        if not isinstance(ledger, dict):
            raise ScriptError(f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: snapshot has no time_ledger object")

        ledger_result = validate_time_ledger(stage_key, dispatch_ordinal, ledger, known_categories)
        stage_results.append(
            {
                "dispatch_ordinal": dispatch_ordinal,
                "stage_key": stage_key,
                "time_ledger": ledger_result,
            }
        )

    print(json.dumps({"bundle_dir": bundle_dir, "stages": stage_results}, sort_keys=True))


try:
    main()
except LedgerViolation as violation:
    sys.stderr.write(f"verify-stage-evidence: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"verify-stage-evidence: {exc}\n")
    sys.exit(2)
PYEOF
