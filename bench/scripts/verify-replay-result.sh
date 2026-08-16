#!/usr/bin/env bash
# verify-replay-result.sh <result_path> <bundle_path>
#
# The single named owner of I-06 result arithmetic (spec.md "Component
# changes" row for verify-replay-result.sh) -- so E40-F08 and E40-F10 invoke
# this script rather than re-deriving I-06's result semantics themselves,
# matching the F05/F06 "single named owner per piece of arithmetic"
# discipline (eval-predicate.sh, verify-stage-evidence.sh).
#
# T-E40-F07-007 stands up this scaffold and implements ONLY REQ-F-009's
# lineage-reconciliation check (ADR-F07-05), described below. This task's
# own Notes for Agent fix the `<result_path> <bundle_path>` argument
# contract as the ONLY thing later tasks may rely on -- proxy validation,
# non-applicable-record completeness (REQ-F-013), and stop-outcome
# eligibility (REQ-F-017) are explicitly NOT implemented here; the full
# I-06 field inventory (REQ-F-001/002/010/011/017/019) is TC-052's (the Go
# contract validator's) job over static committed fixtures
# (tests/contracts/testdata/e40_i06/**), not this script's, for the tasks
# this feature actually decomposed into.
#
# REQ-F-009 / ADR-F07-05: the resolver's own consumption ledger
# (`<bundle_path>.consumption.jsonl`, replay-answer.sh's single-writer
# side-file -- see that script's own header) is the single arbiter every
# artifact's claimed lineage reconciles AGAINST, never trusted alongside.
# Two failure modes are named and MUST stay distinguishable, because they
# have different causes and different downstream owners:
#
#   - `unattributed_artifact`: a stage's artifact claims (in its own
#     `consumed_entries[]`) an {entry_id, entry_digest} pair the ledger
#     never recorded for that stage -- fabrication or resolver bypass, the
#     session invented or paraphrased an answer instead of calling
#     replay-answer.sh. Named on the entry and the artifact's stage. OR: a
#     stage that DID produce at least one artifact whose combined
#     `consumed_entries[]` across all of that stage's artifacts intersects
#     NONE of the bundle's own `required: true` entries for that stage --
#     the artifact exists but nothing it was required to ground itself in
#     was ever attributed to it. Named on the stage.
#   - `unresolved_gate`: a stage that produced NO artifact at all is
#     untouched by either check above -- REQ-F-008 already requires
#     `unresolved_gate` to STOP the prelude before an artifact for that
#     stage is ever produced, so "zero artifacts, zero required entries
#     consumed" is exactly the shape a genuine gate takes, not a rejection
#     this script raises. This script's success JSON passes the result's
#     own top-level `terminal_outcome` straight through unexamined (never
#     asserting or deriving it -- that field's full closed-vocabulary
#     validation is TC-052's job, T-E40-F07-009), which is how a genuine
#     `unresolved_gate` run is REPORTED by this script's own output without
#     this script ever manufacturing an `unattributed_artifact` verdict for
#     it. A validator that collapsed "stopped for a missing entry" and
#     "produced an artifact with nothing attributed to it" into one generic
#     verdict would be indistinguishable from AC-006's own attack case, and
#     is exactly what this split exists to prevent.
#
# A missing `<bundle_path>.consumption.jsonl` side-file means the ledger is
# EMPTY (zero consumptions occurred anywhere in the bundle) -- a valid
# state, not a script error, mirroring replay-answer.sh's own
# load_consumed_ids treating a missing side-file as "every entry
# unconsumed".
#
# Prints one fixed-order JSON document to stdout on success: `stages[]`
# sorted by `stage`, each stage's own `artifacts[]` sorted by `path`, each
# artifact's own claimed entries sorted by the bundle's `ordinal` for that
# entry (REQ-NF-004 byte-identical verdicts; matches F06's
# verify-stage-evidence.sh discipline of "sorted by dispatch_ordinal, then
# field name" carried over here as "by stage, then by entry ordinal"). On a
# violation, prints nothing to stdout and one line to stderr naming the
# offending entry or stage, then exits 1. Exit 2 is reserved for a
# script/usage/authoring error (bad args, missing files, malformed JSON, a
# result or bundle missing a field this check depends on) -- never a normal,
# informative rejection verdict, matching replay-answer.sh's and
# verify-stage-evidence.sh's exit-code convention.
#
# Requires: python3 (stdlib only).
set -euo pipefail

usage() {
	echo "usage: verify-replay-result.sh <result_path> <bundle_path>" >&2
	exit 2
}

[[ $# -eq 2 ]] || usage
result_path="$1"
bundle_path="$2"

[[ -f "$result_path" ]] || {
	echo "verify-replay-result: result not found: $result_path" >&2
	exit 2
}
[[ -f "$bundle_path" ]] || {
	echo "verify-replay-result: bundle not found: $bundle_path" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-replay-result: python3 not found on PATH" >&2
	exit 2
}

result_abs="$(cd "$(dirname "$result_path")" && pwd)/$(basename "$result_path")"
bundle_abs="$(cd "$(dirname "$bundle_path")" && pwd)/$(basename "$bundle_path")"

python3 - "$result_abs" "$bundle_abs" <<'PYEOF'
import json
import os
import sys

result_path, bundle_path = sys.argv[1:3]
ledger_path = bundle_path + ".consumption.jsonl"


class ScriptError(RuntimeError):
    """A prerequisite this validator depends on could not be resolved at
    all -- malformed/missing input distinct from a normal rejection
    verdict. Reported on stderr and mapped to exit 2."""


class Violation(RuntimeError):
    """A real, informative REQ-F-009 rejection verdict -- mapped to exit 1,
    never conflated with a script/usage error. Always kind
    'unattributed_artifact': this task implements only that verdict's two
    triggers; `unresolved_gate` is never raised by this script (see header
    comment) -- it is reported by passing terminal_outcome through, not by
    this class."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


def load_json(path, label):
    try:
        with open(path) as f:
            return json.load(f)
    except OSError as exc:
        raise ScriptError(f"{label}: could not read {path}: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise ScriptError(f"{label}: not valid JSON: {path}: {exc}") from exc


def load_ledger(path):
    """Returns dict[(stage, entry_id)] -> entry_digest, the REAL resolver
    consumption ledger replay-answer.sh owns as the single writer. A
    missing side-file means zero consumptions occurred anywhere in the
    bundle -- a valid, not-yet-exercised state, never a script error."""
    ledger = {}
    if not os.path.exists(path):
        return ledger
    with open(path) as f:
        for lineno, raw_line in enumerate(f, start=1):
            line = raw_line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ScriptError(f"ledger line {lineno} is not valid JSON: {path}: {exc}") from exc
            stage = record.get("stage")
            entry_id = record.get("entry_id")
            entry_digest = record.get("entry_digest")
            if not stage or not entry_id or not entry_digest:
                raise ScriptError(f"ledger line {lineno} missing stage/entry_id/entry_digest: {path}")
            ledger[(stage, entry_id)] = entry_digest
    return ledger


def load_bundle_entries(bundle):
    """Returns (required_by_stage, ordinal_by_entry):
    required_by_stage: dict[stage] -> set of entry_id where required=true.
    ordinal_by_entry: dict[(stage, entry_id)] -> ordinal, used only to
    order this script's own success-JSON output (REQ-NF-004), never to
    gate a verdict."""
    entries = bundle.get("entries")
    if not isinstance(entries, list) or not entries:
        raise ScriptError("bundle has no non-empty entries[] array")

    required_by_stage = {}
    ordinal_by_entry = {}
    for idx, entry in enumerate(entries):
        if not isinstance(entry, dict):
            raise ScriptError(f"bundle entries[{idx}] is not an object")
        stage = entry.get("stage")
        entry_id = entry.get("entry_id")
        ordinal = entry.get("ordinal")
        if not stage or not entry_id or ordinal is None:
            raise ScriptError(f"bundle entries[{idx}] missing stage/entry_id/ordinal")
        ordinal_by_entry[(stage, entry_id)] = ordinal
        if entry.get("required") is True:
            required_by_stage.setdefault(stage, set()).add(entry_id)
    return required_by_stage, ordinal_by_entry


def reconcile_stage(stage_key, stage_record, ledger, required_by_stage, ordinal_by_entry):
    """Implements REQ-F-009 for one result stage record. Raises Violation
    for either named unattributed_artifact trigger. Raises nothing for a
    stage with zero artifacts -- see header comment: that is exactly the
    shape a genuine unresolved_gate stage takes, not a rejection this
    function reports. Returns the JSON-serializable per-stage verdict on
    acceptance."""
    artifacts = stage_record.get("artifacts", [])
    if not isinstance(artifacts, list):
        raise ScriptError(f"stage={stage_key}: artifacts must be a list")

    required_ids = required_by_stage.get(stage_key, set())
    stage_consumed_required_ids = set()
    artifact_verdicts = []

    for a_idx, artifact in enumerate(artifacts):
        if not isinstance(artifact, dict):
            raise ScriptError(f"stage={stage_key} artifacts[{a_idx}] is not an object")
        path = artifact.get("path")
        if not isinstance(path, str) or not path.strip():
            raise ScriptError(f"stage={stage_key} artifacts[{a_idx}] missing required field 'path'")
        claims = artifact.get("consumed_entries", [])
        if not isinstance(claims, list):
            raise ScriptError(f"stage={stage_key} path={path}: consumed_entries must be a list")

        claimed = []  # [(ordinal, entry_id), ...] for this artifact
        for c_idx, claim in enumerate(claims):
            if not isinstance(claim, dict):
                raise ScriptError(f"stage={stage_key} path={path} consumed_entries[{c_idx}] is not an object")
            entry_id = claim.get("entry_id")
            entry_digest = claim.get("entry_digest")
            if not entry_id or not entry_digest:
                raise ScriptError(
                    f"stage={stage_key} path={path} consumed_entries[{c_idx}] missing entry_id/entry_digest"
                )
            ledger_digest = ledger.get((stage_key, entry_id))
            if ledger_digest is None or ledger_digest != entry_digest:
                # REQ-F-009 trigger 1: a claim the resolver's own ledger
                # never recorded for this stage -- fabrication or resolver
                # bypass, regardless of whether entry_id/entry_digest are
                # individually valid against the bundle itself (a
                # digest-correct claim for a real, never-consumed bundle
                # entry is exactly the attack this trigger exists to catch,
                # not merely a malformed-input case).
                raise Violation(
                    "unattributed_artifact",
                    f"stage={stage_key} path={path} entry_id={entry_id}: "
                    f"claimed consumption absent from resolver ledger",
                )
            if entry_id in required_ids:
                stage_consumed_required_ids.add(entry_id)
            claimed.append((ordinal_by_entry.get((stage_key, entry_id), 0), entry_id))

        claimed.sort()
        artifact_verdicts.append(
            {
                "path": path,
                "consumed_entries": [{"entry_id": entry_id, "ordinal": ordinal} for ordinal, entry_id in claimed],
                "result": "accepted",
            }
        )

    if artifacts and required_ids and not stage_consumed_required_ids:
        # REQ-F-009 trigger 2: this stage produced at least one artifact,
        # the bundle declares at least one required=true entry for this
        # stage, and NONE of that stage's artifacts' consumed_entries[]
        # (already proven, above, to each individually reconcile against
        # the ledger) covers any of them -- the artifact exists but nothing
        # it was required to ground itself in was ever attributed to it.
        raise Violation(
            "unattributed_artifact",
            f"stage={stage_key}: stage produced {len(artifacts)} artifact(s) having consumed zero required entries",
        )

    return {
        "stage": stage_key,
        "artifacts_produced": len(artifacts),
        "required_entry_count": len(required_ids),
        "consumed_required_entry_ids": sorted(stage_consumed_required_ids),
        "artifacts": sorted(artifact_verdicts, key=lambda v: v["path"]),
        "result": "accepted",
    }


def main():
    result = load_json(result_path, "result")
    bundle = load_json(bundle_path, "bundle")
    ledger = load_ledger(ledger_path)
    required_by_stage, ordinal_by_entry = load_bundle_entries(bundle)

    stages = result.get("stages")
    if not isinstance(stages, list) or not stages:
        raise ScriptError("result has no non-empty stages[] array")

    stage_results = {}
    for idx, stage_record in enumerate(stages):
        if not isinstance(stage_record, dict):
            raise ScriptError(f"result.stages[{idx}] is not an object")
        stage_key = stage_record.get("stage")
        if not stage_key:
            raise ScriptError(f"result.stages[{idx}] missing 'stage'")
        if stage_key in stage_results:
            raise ScriptError(f"result.stages[] has duplicate stage entry: {stage_key}")
        stage_results[stage_key] = reconcile_stage(
            stage_key, stage_record, ledger, required_by_stage, ordinal_by_entry
        )

    output = {
        "result_path": result_path,
        "bundle_path": bundle_path,
        # Passed through verbatim, never derived or asserted by this
        # check -- see header comment: this is how a genuine
        # unresolved_gate run is REPORTED without this script ever
        # deriving or validating the full terminal_outcome vocabulary
        # (TC-052's job, T-E40-F07-009).
        "terminal_outcome": result.get("terminal_outcome"),
        "stages": [stage_results[stage] for stage in sorted(stage_results)],
    }
    print(json.dumps(output, sort_keys=True))


try:
    main()
except Violation as violation:
    sys.stderr.write(f"verify-replay-result: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"verify-replay-result: {exc}\n")
    sys.exit(2)
PYEOF
