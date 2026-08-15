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
# T-E40-F06-004 stood up this scaffold and implemented the REQ-F-005
# time_ledger reconciliation check (ADR-F06-06). T-E40-F06-005 adds the
# REQ-F-006 / ADR-009 candidate-identity check described below. Later tasks
# MODIFY this same script to add artifact records (T-E40-F06-006,
# REQ-F-008), evaluator_access ordering (T-E40-F06-007, REQ-F-012), and
# stop-outcome eligibility (T-E40-F06-008, REQ-F-014) -- the `<bundle_dir>`
# argument contract is deliberately the ONLY thing this task fixes for those
# tasks to build on; nothing else about this script's internal structure is
# a promise.
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
# T-E40-F06-005 candidate identity check (REQ-F-006, ADR-009, spec.md
# "Stage snapshot (I-05)" `candidate` row):
#
#   For every stage snapshot whose OWN `stage_category` field (read from
#   the snapshot itself, never from the bundle.json index -- mirroring the
#   Go contract validator's e40I05ValidateStageSnapshot,
#   tests/contracts/e40_i05_stage_evidence_contract_test.go, TC-042) is
#   `code` or `review`:
#
#   1. the snapshot MUST carry a `candidate` object with `base_commit`,
#      `tree_digest`, `binary_diff_digest`, `changed_path_digest`,
#      `dirty_untracked_manifest`, and `test_suite_digest` -- a snapshot
#      missing any one of them is rejected naming that exact field
#      (`candidate_field_missing`, sourced from
#      `bench/evidence/i05-schema.yaml`'s `error_kind` vocabulary).
#      `test_suite_digest` is validated only for presence, exactly like the
#      other digest fields -- never recomputed, never inspected for a
#      Python- or Go-shaped path, matching ADR-F06-05's opacity discipline
#      (REQ-F-007): the digest is already computed elsewhere from the
#      adapter's normalized test-id set, and this validator's only job is
#      to confirm it is present and fold it into the identity hash unread.
#   2. `base_commit` is one field of the candidate's identity, never the
#      identity by itself (ADR-009): this script never groups, compares, or
#      dedupes candidates by `base_commit` alone. It hashes ALL SIX
#      required fields -- `base_commit` plus the five digest/manifest
#      fields, `dirty_untracked_manifest` hashed in its declared order
#      (REQ-F-006 defines it as an ORDERED list; reordering it is itself an
#      identity change, so this hash must never re-sort it) -- into a
#      `candidate_identity_digest` reported alongside `base_commit` in the
#      success JSON. Two snapshots sharing `base_commit` but differing in
#      any one of the other five fields deterministically produce different
#      digests; this script performs no cross-bundle merging of its own,
#      leaving distinctness verification to the caller (tc045).
#
# T-E40-F06-006 artifact-record check (REQ-F-008, ADR-F06-07, spec.md
# "Stage snapshot (I-05)" `artifacts` row):
#
#   For every entry in a stage snapshot's OPTIONAL `artifacts` array, this
#   script reports a per-artifact consumer verdict derived from the RAW
#   JSON-key shape of that entry's `consumers` field -- never from a
#   `.get(key, default)`-style read that would silently coerce "key
#   absent" into "key present, empty" (the exact Go `omitempty`/zero-value
#   trap ADR-F06-07 forbids and this task's Notes for Agent name
#   explicitly):
#
#     - `consumers` key present and an empty list (`consumers: []`) ->
#       verdict `orphan` ("no consumer observed").
#     - `consumers` key entirely absent from the artifact object -> verdict
#       `consumption_evidence_missing` ("consumption evidence was not
#       collected").
#     - `consumers` key present and non-empty -> verdict `consumed`. Not
#       one of ADR-F06-07's two named states, but the third cell the same
#       field naturally partitions into: spec.md's ADR-F06-07 names this
#       distinction's whole point as being able to "distinguish a consumed
#       artifact from an orphan" (UAT-18), so a validator that could only
#       ever report `orphan` or `consumption_evidence_missing` -- with no
#       way to represent an artifact that WAS actually consumed -- would
#       leave that distinction unrepresentable in its own output.
#
#   A snapshot with no `artifacts` key at all is untouched by this check
#   (mirrors validate_candidate's not-applicable return), so ledger-only
#   fixtures under bench/scripts/testdata/evidence/ledger/ and
#   bench/scripts/testdata/evidence/candidate/ (neither of which declare an
#   `artifacts` key) are unaffected. This check reads only the `path` and
#   `consumers` shape needed to derive the verdict -- `path` is not itself
#   being asserted as a REQ-F-008 required field here (TC-042's Go
#   validator owns REQ-F-016's full field-inventory rejection, including a
#   record missing `producer_stage` or `digest`); it is simply the
#   identifier this function's per-artifact verdict is keyed to, so an
#   artifact entry with no usable `path` means this function cannot report
#   a verdict for it at all, not that REQ-F-008's field list is enforced
#   here.
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
import hashlib
import json
import os
import sys

import yaml

bundle_dir, i05_schema_path = sys.argv[1:3]


class ScriptError(RuntimeError):
    """A prerequisite this validator depends on could not be resolved at
    all -- malformed/missing input distinct from a normal rejection
    verdict. Reported on stderr and mapped to exit 2."""


class Violation(RuntimeError):
    """A real, informative rejection verdict -- mapped to exit 1, never
    silently swallowed, never conflated with a script/usage error. Shared
    base for every REQ-specific violation this validator raises so the
    single top-level handler below stays exhaustive as more checks
    (T-E40-F06-005 onward) are added to this script."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


class LedgerViolation(Violation):
    """A real, informative REQ-F-005 rejection verdict."""


class CandidateViolation(Violation):
    """A real, informative REQ-F-006 / ADR-009 candidate-identity rejection
    verdict."""


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


# REQ-F-006 / ADR-009: the five candidate fields the task spec's AC-T1
# dedicates an independent missing-field case to. `base_commit` is required
# too (REQ-F-006's field list; mirrored from the Go contract validator's
# e40I05ValidateCandidate) even though AC-T1 does not dedicate a case to it.
CANDIDATE_REQUIRED_STRING_FIELDS = (
    "base_commit",
    "tree_digest",
    "binary_diff_digest",
    "changed_path_digest",
    "test_suite_digest",
)


def validate_candidate(stage_key, dispatch_ordinal, stage_category, snapshot):
    """Implements REQ-F-006 / ADR-009 for one code/review stage snapshot's
    `candidate` block. Mirrors the Go contract validator's
    e40I05ValidateCandidate (tests/contracts/e40_i05_stage_evidence_contract_test.go,
    TC-042) field-for-field so the two validators agree on what "missing"
    means. Returns None for any stage_category outside {code, review} --
    REQ-F-006 does not apply, so a ledger-only fixture with no `candidate`
    key at all (e.g. bench/scripts/testdata/evidence/ledger/*, none of
    which declare a snapshot-level stage_category) is untouched by this
    check. Raises CandidateViolation naming the first missing field for a
    real rejection; never raises ScriptError, because a missing/incomplete
    candidate block on a code/review snapshot is exactly the informative
    verdict AC-T1 exists to produce, not a malformed input this script
    cannot evaluate at all.

    Identity discipline (REQ-F-006, ADR-009, AC-T2): `base_commit` is one
    field of a candidate's identity, never the identity by itself. This
    function never groups, compares, or dedupes candidates by `base_commit`
    alone -- it hashes ALL SIX required fields (`base_commit` plus the five
    digest/manifest fields, `dirty_untracked_manifest` hashed in its
    declared order -- REQ-F-006 defines it as an ORDERED list, so this hash
    must never re-sort it) into `candidate_identity_digest`, so two
    snapshots sharing `base_commit` but differing in any one of the other
    five fields deterministically produce different digests. Callers (and
    tc045) compare that digest, and the `base_commit` reported alongside
    it, across separate guard invocations to prove distinctness; this
    script performs no cross-bundle merging of its own.
    """
    if stage_category not in ("code", "review"):
        return None

    candidate = snapshot.get("candidate")
    if not isinstance(candidate, dict):
        raise CandidateViolation(
            "candidate_field_missing",
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} category={stage_category}: "
            f"candidate block missing or not an object",
        )

    for field in CANDIDATE_REQUIRED_STRING_FIELDS:
        value = candidate.get(field)
        if not isinstance(value, str) or not value.strip():
            raise CandidateViolation(
                "candidate_field_missing",
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} category={stage_category} field={field}",
            )

    if "dirty_untracked_manifest" not in candidate:
        raise CandidateViolation(
            "candidate_field_missing",
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} category={stage_category} field=dirty_untracked_manifest",
        )

    # Identity hash over all six required fields, in their own declared
    # shapes/order (never re-sorted) -- json.dumps(sort_keys=True) here
    # only sorts the OBJECT's key order for a stable serialization, not the
    # dirty_untracked_manifest list's element order.
    identity_payload = {
        "base_commit": candidate["base_commit"],
        "tree_digest": candidate["tree_digest"],
        "binary_diff_digest": candidate["binary_diff_digest"],
        "changed_path_digest": candidate["changed_path_digest"],
        "dirty_untracked_manifest": candidate["dirty_untracked_manifest"],
        "test_suite_digest": candidate["test_suite_digest"],
    }
    identity_digest = hashlib.sha256(
        json.dumps(identity_payload, sort_keys=True).encode("utf-8")
    ).hexdigest()

    return {
        "base_commit": candidate["base_commit"],
        "candidate_identity_digest": f"sha256:{identity_digest}",
        "result": "accepted",
    }


def validate_artifacts(stage_key, dispatch_ordinal, snapshot):
    """Implements REQ-F-008 / ADR-F06-07 per-artifact consumer verdicts for
    one stage snapshot's `artifacts` array. Returns None when the snapshot
    carries no `artifacts` key at all -- REQ-F-008 does not apply, so a
    ledger-only or candidate-only fixture with no `artifacts` key is
    untouched by this check (mirrors validate_candidate's not-applicable
    return for a non-code/review stage_category). Raises ScriptError for a
    malformed `artifacts` array/entry this script cannot evaluate at all
    (not a list, an entry that is not an object, a missing `path`, or a
    `consumers` value that is present but not a list) -- never for the
    empty-vs-absent `consumers` distinction itself, which is the normal,
    informative verdict this function exists to report, not a rejection.

    ADR-F06-07 non-coercion discipline: the empty-vs-absent distinction is
    read directly off the decoded artifact dict's own key membership
    (`"consumers" not in artifact`), never via a `.get("consumers", [])`
    default read, which would silently collapse "key absent" into "key
    present, empty" -- the exact trap this check exists to catch (tc046
    AC-T3).
    """
    if "artifacts" not in snapshot:
        return None

    artifacts = snapshot["artifacts"]
    if not isinstance(artifacts, list):
        raise ScriptError(
            f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: snapshot.artifacts must be a list"
        )

    results = []
    for idx, artifact in enumerate(artifacts):
        if not isinstance(artifact, dict):
            raise ScriptError(
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: artifacts[{idx}] is not an object"
            )
        path = artifact.get("path")
        if not isinstance(path, str) or not path.strip():
            raise ScriptError(
                f"stage={stage_key} dispatch_ordinal={dispatch_ordinal}: artifacts[{idx}] missing required field 'path'"
            )

        if "consumers" not in artifact:
            verdict = "consumption_evidence_missing"
        else:
            consumers = artifact["consumers"]
            if not isinstance(consumers, list):
                raise ScriptError(
                    f"stage={stage_key} dispatch_ordinal={dispatch_ordinal} path={path}: "
                    f"consumers must be a list when present"
                )
            verdict = "orphan" if len(consumers) == 0 else "consumed"

        results.append({"path": path, "verdict": verdict})

    return results


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

        stage_result = {
            "dispatch_ordinal": dispatch_ordinal,
            "stage_key": stage_key,
            "time_ledger": ledger_result,
        }
        # REQ-F-006: stage_category is read from the snapshot's OWN field,
        # never from the bundle.json index entry -- see header comment.
        candidate_result = validate_candidate(
            stage_key, dispatch_ordinal, snapshot.get("stage_category"), snapshot
        )
        if candidate_result is not None:
            stage_result["candidate"] = candidate_result
        artifact_results = validate_artifacts(stage_key, dispatch_ordinal, snapshot)
        if artifact_results is not None:
            stage_result["artifacts"] = artifact_results
        stage_results.append(stage_result)

    print(json.dumps({"bundle_dir": bundle_dir, "stages": stage_results}, sort_keys=True))


try:
    main()
except Violation as violation:
    sys.stderr.write(f"verify-stage-evidence: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"verify-stage-evidence: {exc}\n")
    sys.exit(2)
PYEOF
