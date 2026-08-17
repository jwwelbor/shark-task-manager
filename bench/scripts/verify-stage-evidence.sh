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
# T-E40-F06-007 evaluator-access ordering guard (REQ-F-012, ADR-F06-08):
#
#   The bare `<bundle_dir>` invocation above is a passive, after-the-fact
#   validator over an already-written snapshot -- it cannot be what REQ-F-012
#   describes as "invoked ... is rejected" for a placement that has not
#   happened yet. So this task adds a second, ACTIVE invocation shape,
#   `<bundle_dir> --grant-access <inject-tests|in-place-read> ...`, which
#   is a broker: it decides whether to allow one evaluator-only access,
#   performs it for real when allowed, and is the thing that appends the
#   resulting `evaluator_access` event to `<bundle_dir>/access.jsonl` --
#   never a caller-supplied event the broker merely echoes. The bare
#   `<bundle_dir>` (no `--grant-access`) invocation is completely untouched
#   by this addition; every check above still runs exactly as before.
#
#   `--grant-access inject-tests` (REQ-F-012(b)): gates on
#   `<bundle_dir>/bundle.json`'s `terminal_status.reached` (new bundle-level
#   field this task introduces; absent/false is "not yet terminal"). Before
#   terminal, the request is refused as `isolation_violation` and NEITHER
#   the adapter NOR the filesystem is touched -- AC-T1 depends on the
#   placement never happening, not merely on a non-zero exit. After
#   terminal, this invokes I-04's REAL `adapter.sh inject-tests --checkout
#   <dir> --files <path>...` as a subprocess (ADR-F06-08: reuse inject-tests
#   rather than a new copy path) and then reads the file the adapter placed
#   back off disk and digests it independently -- observing the real
#   adapter invocation's effect, never trusting the adapter's own stdout
#   claim as a boolean "it worked" flag (task Notes for Agent).
#
#   `--grant-access in-place-read` (REQ-F-012(c)): same terminal-status gate,
#   for a read of `evaluator_only.reference_solution` / `answer_keys[]`
#   performed IN PLACE from `--evaluator-root`, never copied. Before the
#   digest is read and the event appended, this walks `--checkout` (the
#   agent-visible root) for any file byte-identical to the requested
#   artifact whose mtime PREDATES `terminal_status.reached_at` -- a
#   pre-completion copy, the actual violation case (d) exists to catch
#   (task Notes for Agent: rejected for WHEN the copy happened, not merely
#   that a copy exists -- an accepted `inject-tests` placement of a
#   DIFFERENT, digest-distinct oracle-test file must never trip this check).
#
#   Both grant modes append at most one `evaluator_access` event
#   `{accessor, artifact_path, digest, phase, granted_at}` per successful
#   grant, always with `phase: "post_terminal"` -- i05-schema.yaml documents
#   `pre_terminal` as a negative-only value no valid bundle ever carries, so
#   a refused (pre-terminal) request raises `isolation_violation` and never
#   writes any event at all, matching REQ-F-012's "not a warning."
#
# T-E40-F06-008 stop-outcome eligibility check (REQ-F-014, spec.md "Bundle
# layout (I-05)" `stop_outcome`/`publication_eligible`/
# `ineligibility_reasons` rows):
#
#   Bundle-level (not per-stage): a bare `<bundle_dir>` invocation reads the
#   optional `bundle.json` `stop_outcome` field. Absent -> not applicable,
#   a clean terminal run, untouched by this check. Present -> it MUST be one
#   of the ten REQ-F-017 stop_outcome values sourced live from
#   `bench/evidence/i05-schema.yaml` (never a private copy here, mirroring
#   the interval_category and stage_category sourcing discipline above),
#   `publication_eligible` MUST be `false`, and `ineligibility_reasons[]`
#   MUST be a non-empty list -- else rejected. A bundle pairing a
#   stop_outcome with `publication_eligible: true` is rejected naming
#   `publication_eligible_conflict` (i05-schema.yaml `error_kind`
#   vocabulary) -- the literal contradiction AC-T2 exists to catch.
#
#   This check adds no gate to the per-stage loop above: whatever subset of
#   `stages[]` a stopped bundle indexes is read and validated exactly like
#   any other bundle's stages, so a bundle carrying fewer stage snapshots
#   than a complete run (the real shape of "partial evidence retained") is
#   reported stage-by-stage same as always -- AC-T1's "partial snapshots
#   stay present/readable" is a property of that unchanged per-stage read,
#   not of anything new this check adds.
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
	echo "       verify-stage-evidence.sh <bundle_dir> --grant-access inject-tests --accessor <name> --adapter <adapter.sh> --checkout <dir> --files <path>..." >&2
	echo "       verify-stage-evidence.sh <bundle_dir> --grant-access in-place-read --accessor <name> --evaluator-root <dir> --artifact <rel_path> --checkout <dir>" >&2
	exit 2
}

[[ $# -ge 1 ]] || usage
bundle_dir="$1"
shift

grant_mode=""
accessor=""
adapter_path=""
checkout=""
evaluator_root=""
artifact_rel=""
files=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--grant-access)
		grant_mode="${2:-}"
		shift 2
		;;
	--accessor)
		accessor="${2:-}"
		shift 2
		;;
	--adapter)
		adapter_path="${2:-}"
		shift 2
		;;
	--checkout)
		checkout="${2:-}"
		shift 2
		;;
	--evaluator-root)
		evaluator_root="${2:-}"
		shift 2
		;;
	--artifact)
		artifact_rel="${2:-}"
		shift 2
		;;
	--files)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			files+=("$1")
			shift
		done
		;;
	*)
		usage
		;;
	esac
done

[[ -d "$bundle_dir" ]] || {
	echo "verify-stage-evidence: bundle dir not found: $bundle_dir" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-stage-evidence: python3 not found on PATH" >&2
	exit 2
}

bundle_dir_abs="$(cd "$bundle_dir" && pwd)"

# T-E40-F06-007: the --grant-access broker path is a completely separate
# invocation shape from the bare-bundle validation below -- see the header
# comment. It requires neither PyYAML nor the full bundle.json stage index,
# so it is handled and exited before any of that is required or touched.
if [[ -n "$grant_mode" ]]; then
	[[ "$grant_mode" == "inject-tests" || "$grant_mode" == "in-place-read" ]] || usage
	[[ -n "$accessor" ]] || usage

	case "$grant_mode" in
	inject-tests)
		[[ -n "$adapter_path" ]] || usage
		[[ -x "$adapter_path" ]] || {
			echo "verify-stage-evidence: adapter not found or not executable: $adapter_path" >&2
			exit 2
		}
		[[ -n "$checkout" ]] || usage
		[[ -d "$checkout" ]] || {
			echo "verify-stage-evidence: checkout dir not found: $checkout" >&2
			exit 2
		}
		[[ ${#files[@]} -gt 0 ]] || usage
		;;
	in-place-read)
		[[ -n "$evaluator_root" ]] || usage
		[[ -d "$evaluator_root" ]] || {
			echo "verify-stage-evidence: evaluator root not found: $evaluator_root" >&2
			exit 2
		}
		[[ -n "$artifact_rel" ]] || usage
		[[ -n "$checkout" ]] || usage
		[[ -d "$checkout" ]] || {
			echo "verify-stage-evidence: checkout dir not found: $checkout" >&2
			exit 2
		}
		;;
	esac

	adapter_abs=""
	[[ -z "$adapter_path" ]] || adapter_abs="$(cd "$(dirname "$adapter_path")" && pwd)/$(basename "$adapter_path")"
	checkout_abs=""
	[[ -z "$checkout" ]] || checkout_abs="$(cd "$checkout" && pwd)"
	evaluator_root_abs=""
	[[ -z "$evaluator_root" ]] || evaluator_root_abs="$(cd "$evaluator_root" && pwd)"

	python3 - "$bundle_dir_abs" "$grant_mode" "$accessor" "$adapter_abs" "$checkout_abs" "$evaluator_root_abs" "$artifact_rel" "${files[@]}" <<'PYEOF'
import datetime
import hashlib
import json
import os
import subprocess
import sys

(
    bundle_dir,
    mode,
    accessor,
    adapter_path,
    checkout,
    evaluator_root,
    artifact_rel,
    *files,
) = sys.argv[1:]


class ScriptError(RuntimeError):
    """A prerequisite this broker depends on could not be resolved at all --
    distinct from a normal isolation-violation refusal. Reported on stderr
    and mapped to exit 2."""


class Violation(RuntimeError):
    """A real, informative REQ-F-012 rejection verdict -- mapped to exit 1,
    never conflated with a script/usage error."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


def now_rfc3339():
    return datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def parse_rfc3339(value):
    if value.endswith("Z"):
        value = value[:-1] + "+00:00"
    return datetime.datetime.fromisoformat(value)


def sha256_file(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def load_terminal_status(bundle_dir):
    bundle_json_path = os.path.join(bundle_dir, "bundle.json")
    if not os.path.isfile(bundle_json_path):
        raise ScriptError(f"bundle.json not found: {bundle_json_path}")
    with open(bundle_json_path) as f:
        try:
            bundle = json.load(f)
        except json.JSONDecodeError as exc:
            raise ScriptError(f"bundle.json is not valid JSON: {bundle_json_path}: {exc}") from exc
    terminal = bundle.get("terminal_status")
    if not isinstance(terminal, dict) or "reached" not in terminal:
        raise ScriptError(f"bundle.json missing terminal_status.reached: {bundle_json_path}")
    reached = bool(terminal["reached"])
    reached_at = terminal.get("reached_at")
    if reached and not reached_at:
        raise ScriptError(f"bundle.json terminal_status.reached=true but reached_at is missing: {bundle_json_path}")
    return reached, reached_at


def append_access_event(bundle_dir, event):
    # REQ-F-012: "Every read of evaluator-only material MUST append one
    # evaluator_access event ... to the bundle" -- append-only, one JSON
    # line per grant, matching the bundle layout's access.jsonl (spec.md
    # "Bundle layout (I-05)").
    access_path = os.path.join(bundle_dir, "access.jsonl")
    with open(access_path, "a") as f:
        f.write(json.dumps(event, sort_keys=True) + "\n")


def grant_inject_tests(bundle_dir, accessor, adapter_path, checkout, files):
    reached, reached_at = load_terminal_status(bundle_dir)
    if not reached:
        # AC-T1: refused BEFORE any provider/adapter call -- the adapter is
        # never invoked and the checkout is never touched.
        raise Violation(
            "isolation_violation",
            f"accessor={accessor!r} mode=inject-tests phase=pre_terminal: "
            f"terminal status not reached; inject-tests refused before invoking the adapter",
        )

    # REQ-F-012(b) / ADR-F06-08: placement goes through I-04's REAL
    # adapter.sh inject-tests capability -- a real subprocess, never a
    # direct file write bypassing the adapter.
    proc = subprocess.run(
        [adapter_path, "inject-tests", "--checkout", checkout, "--files", *files],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise ScriptError(f"adapter.sh inject-tests failed (exit {proc.returncode}): {proc.stderr.strip()}")
    try:
        result = json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise ScriptError(f"adapter.sh inject-tests produced non-JSON stdout: {proc.stdout!r} ({exc})") from exc
    injected = result.get("injected")
    if not isinstance(injected, list) or not injected:
        raise ScriptError(f"adapter.sh inject-tests reported no injected files: {proc.stdout!r}")

    events = []
    for entry in injected:
        source = entry.get("source")
        destination_rel = entry.get("destination")
        if not source or not destination_rel:
            raise ScriptError(f"adapter.sh inject-tests entry missing source/destination: {entry!r}")
        destination_abs = os.path.join(checkout, destination_rel)
        # Observe the REAL effect on disk -- digest the file the adapter
        # actually placed, never the adapter's own stdout claim alone (task
        # Notes for Agent: "observes the real adapter invocation's effect,
        # not a boolean flag").
        if not os.path.isfile(destination_abs):
            raise ScriptError(
                f"adapter.sh inject-tests reported destination {destination_rel!r} "
                f"but it does not exist on disk: {destination_abs}"
            )
        event = {
            "accessor": accessor,
            "artifact_path": source,
            "destination": destination_rel,
            "digest": f"sha256:{sha256_file(destination_abs)}",
            "phase": "post_terminal",
            "granted_at": now_rfc3339(),
        }
        append_access_event(bundle_dir, event)
        events.append(event)
    return events


def grant_in_place_read(bundle_dir, accessor, evaluator_root, artifact_rel, checkout):
    reached, reached_at = load_terminal_status(bundle_dir)
    if not reached:
        raise Violation(
            "isolation_violation",
            f"accessor={accessor!r} mode=in-place-read artifact={artifact_rel!r} phase=pre_terminal: "
            f"terminal status not reached; read refused",
        )

    evaluator_root_real = os.path.realpath(evaluator_root)
    source_abs = os.path.realpath(os.path.join(evaluator_root, artifact_rel))
    if source_abs != evaluator_root_real and not source_abs.startswith(evaluator_root_real + os.sep):
        raise ScriptError(f"--artifact escapes --evaluator-root: {artifact_rel!r}")
    if not os.path.isfile(source_abs):
        raise ScriptError(f"evaluator-only artifact not found: {source_abs}")
    source_digest = sha256_file(source_abs)

    # REQ-F-012(c) / task Notes for Agent: rejected for WHEN a copy of THIS
    # artifact landed in the agent-visible checkout, not merely that a copy
    # exists -- a byte-identical file already present, backdated before
    # terminal_status.reached_at, is the pre-completion-copy violation case
    # (d); an unrelated (digest-distinct) file, such as case (b)'s own
    # accepted inject-tests placement, must never trip this.
    reached_at_dt = parse_rfc3339(reached_at)
    for dirpath, dirnames, filenames in os.walk(checkout):
        dirnames[:] = sorted(d for d in dirnames if d != ".git")
        for name in sorted(filenames):
            candidate = os.path.join(dirpath, name)
            try:
                candidate_digest = sha256_file(candidate)
            except OSError:
                continue
            if candidate_digest != source_digest:
                continue
            candidate_mtime = datetime.datetime.fromtimestamp(
                os.path.getmtime(candidate), tz=datetime.timezone.utc
            )
            if candidate_mtime < reached_at_dt:
                raise Violation(
                    "isolation_violation",
                    f"accessor={accessor!r} artifact={artifact_rel!r}: pre-completion copy found at "
                    f"{candidate} (mtime={candidate_mtime.isoformat()}, "
                    f"terminal_status.reached_at={reached_at})",
                )

    event = {
        "accessor": accessor,
        "artifact_path": source_abs,
        "digest": f"sha256:{source_digest}",
        "phase": "post_terminal",
        "granted_at": now_rfc3339(),
    }
    append_access_event(bundle_dir, event)
    return [event]


def main():
    if mode == "inject-tests":
        events = grant_inject_tests(bundle_dir, accessor, adapter_path, checkout, files)
    elif mode == "in-place-read":
        events = grant_in_place_read(bundle_dir, accessor, evaluator_root, artifact_rel, checkout)
    else:
        raise ScriptError(f"unknown --grant-access mode: {mode!r}")
    print(json.dumps({"bundle_dir": bundle_dir, "granted": events}, sort_keys=True))


try:
    main()
except Violation as violation:
    sys.stderr.write(f"verify-stage-evidence: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"verify-stage-evidence: {exc}\n")
    sys.exit(2)
PYEOF
	exit 0
fi

[[ -f "$I05_SCHEMA" ]] || {
	echo "verify-stage-evidence: i05-schema.yaml not found: $I05_SCHEMA" >&2
	exit 2
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "verify-stage-evidence: python3 module 'yaml' (PyYAML) not available" >&2
	exit 2
}

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


class EligibilityViolation(Violation):
    """A real, informative REQ-F-014 rejection verdict -- a stop_outcome
    bundle that also declares publication_eligible: true."""


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


def validate_eligibility(bundle, known_stop_outcomes):
    """Implements REQ-F-014 for the bundle-level `stop_outcome` /
    `publication_eligible` / `ineligibility_reasons` triad (spec.md "Bundle
    layout (I-05)"). Returns None when the bundle carries no `stop_outcome`
    key at all -- REQ-F-014 does not apply to a clean terminal run, so
    every fixture used by tc043-tc048 (none of which declare stop_outcome)
    is untouched by this check, mirroring validate_candidate's /
    validate_artifacts's not-applicable return for a field that does not
    apply to every bundle.

    Raises EligibilityViolation naming `publication_eligible_conflict`
    (bench/evidence/i05-schema.yaml `error_kind` vocabulary) when a
    stop_outcome bundle also declares `publication_eligible: true` -- the
    literal contradiction AC-T2 exists to catch. This is the only
    REQ-F-014 failure mode this function treats as an informative
    rejection verdict rather than a malformed-input ScriptError, because it
    is the only REQ-F-014 failure mode with a named `error_kind` in
    i05-schema.yaml.

    Raises ScriptError for a `stop_outcome` value outside the closed
    REQ-F-017 vocabulary (sourced live from i05-schema.yaml, never a
    private copy here), or a stop_outcome bundle missing a non-empty
    `ineligibility_reasons[]` -- malformed authoring input this script
    cannot evaluate as a described contradiction, not one of the two
    states this task's ACs exercise.

    Does NOT itself confirm "partial snapshots stay present/readable"
    (AC-T1) -- that is a property of the unchanged per-stage loop in
    main(), which reads and validates whatever subset of `stages[]` the
    bundle actually indexes for every bundle, stop_outcome or not. This
    function's own job is strictly the bundle-level eligibility triad.
    """
    stop_outcome = bundle.get("stop_outcome")
    if stop_outcome is None:
        return None

    if stop_outcome not in known_stop_outcomes:
        raise ScriptError(
            f"bundle.json stop_outcome={stop_outcome!r} is not one of the closed REQ-F-017 "
            f"stop_outcome vocabulary in i05-schema.yaml: {sorted(known_stop_outcomes)}"
        )

    publication_eligible = bundle.get("publication_eligible")
    if publication_eligible is not False:
        raise EligibilityViolation(
            "publication_eligible_conflict",
            f"stop_outcome={stop_outcome!r} but publication_eligible={publication_eligible!r} "
            f"(REQ-F-014 requires publication_eligible: false whenever stop_outcome is present)",
        )

    ineligibility_reasons = bundle.get("ineligibility_reasons")
    if not isinstance(ineligibility_reasons, list) or not ineligibility_reasons:
        raise ScriptError(
            f"bundle.json stop_outcome={stop_outcome!r}: ineligibility_reasons[] missing or empty "
            f"(REQ-F-014 requires a non-empty list whenever publication_eligible is false)"
        )

    return {
        "stop_outcome": stop_outcome,
        "publication_eligible": publication_eligible,
        "ineligibility_reasons": ineligibility_reasons,
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
    known_stop_outcomes = schema.get("stop_outcome")
    if not isinstance(known_stop_outcomes, list) or not known_stop_outcomes:
        raise ScriptError(f"i05-schema.yaml declares no stop_outcome vocabulary: {i05_schema_path}")

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

    output = {"bundle_dir": bundle_dir, "stages": stage_results}
    # REQ-F-014: bundle-level, not per-stage -- see header comment.
    eligibility_result = validate_eligibility(bundle, known_stop_outcomes)
    if eligibility_result is not None:
        output["eligibility"] = eligibility_result

    print(json.dumps(output, sort_keys=True))


try:
    main()
except Violation as violation:
    sys.stderr.write(f"verify-stage-evidence: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"verify-stage-evidence: {exc}\n")
    sys.exit(2)
PYEOF
