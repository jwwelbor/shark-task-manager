#!/usr/bin/env bash
# replay-answer.sh --bundle <path> --stage <D01|...|D05> \
#                  --kind <human_question|research_query> --topic <key>
#
# The single named owner of REQ-F-006/REQ-F-007's request matching,
# response supply, and consumption-recording arithmetic (spec.md "Resolver
# interface"; ADR-F07-04, ADR-F07-05) -- no other F07 script, test, or
# future consumer (run-prelude.sh's dispatched session, invoked indirectly
# through this script over Bash) may re-derive these semantics. Matches the
# F05/F06 "single named owner per piece of arithmetic" discipline
# (eval-predicate.sh, verify-stage-evidence.sh).
#
# Matching is ORDINAL-PRIMARY WITH A TOPIC ASSERTION (ADR-F07-04), never a
# match against the model's own literal request text: the resolver looks at
# the bundle's entries[] for the caller's --stage, finds the lowest ordinal
# not yet recorded as consumed in this bundle's own consumption ledger (a
# sibling file this script owns -- see LEDGER SIDE-FILE below), and only
# supplies its response when that entry's OWN request_kind and topic_key
# both equal the caller-supplied --kind/--topic. There is no nearest,
# partial, or fuzzy match: a caller topic that is a one-character near-miss
# of an unconsumed entry's real topic_key fails exactly like any other
# disagreement (replay_desync) -- it is never "close enough."
#
# Three, and only three, outcomes exist (spec.md "Resolver interface"):
#   - exit 0: the lowest-unconsumed-ordinal entry's request_kind/topic_key
#     both match the caller's --kind/--topic. The response bytes are
#     written verbatim to stdout (no added newline), and exactly one
#     consumption record is appended to the ledger before the response is
#     printed.
#   - exit 1, "replay_desync": an unconsumed entry exists for the stage,
#     but its request_kind or topic_key disagrees with the caller's
#     --kind/--topic. Stderr names both the expected (bundle-declared) and
#     supplied (caller-given) kind/topic. No response is printed and no
#     consumption record is appended.
#   - exit 1, "unresolved_gate": no unconsumed entry remains for the stage
#     (either every entry was already consumed, or the bundle never
#     declared one for this stage). Stderr names stage, kind, and topic
#     (REQ-F-008: the harness -- not this script -- is the one that must
#     stop the prelude and never invent an answer; this script's own job is
#     only to report the gate precisely enough for that caller to do so).
#   - exit 2, ScriptError: a malformed bundle (missing/invalid JSON,
#     missing required entry fields, duplicate ordinal within a stage) or
#     an unresolvable {path, digest} response (escapes the bundle
#     directory, file missing, or resolved bytes do not hash to the
#     declared digest) -- never a matching verdict, always a
#     script/authoring problem distinct from replay_desync/unresolved_gate.
#
# LEDGER SIDE-FILE: this script is the single writer of the consumption
# ledger (REQ-F-009's "single writer" rule, though full lineage
# reconciliation against it is T-E40-F07-007's job, not this task's). The
# ledger lives at "<bundle_path>.consumption.jsonl", one compact JSON
# object per successful supply, appended in call order:
#   {"entry_id", "entry_digest", "stage", "ordinal", "request_kind",
#    "topic_key", "response_digest", "supplied_at"}
# "supplied_at" is wall-clock, recorded for the ledger only -- never
# reported as a proxy field, never used to gate a decision, and never
# synthesized (REQ-NF-007: no F07 component invents delay). A fresh
# ledger (no side-file yet) means every entry in the bundle is unconsumed;
# deleting the side-file resets replay state for the bundle it sits next
# to, which is how two independent scored passes over the identical
# committed bundle each start from an empty ledger and therefore consume a
# byte-identical response sequence when driven by the same call sequence
# (REQ-F-007, AC-005) -- a fresh scratch project's own bundle copy simply
# has no side-file yet.
#
# No network call is made anywhere in this script -- every operation is a
# local file read/write.
#
# Exit status: 0 = supplied (response on stdout). 1 = replay_desync or
# unresolved_gate, named on stderr. 2 = ScriptError (malformed bundle,
# unresolvable response path, bad usage).
#
# Requires: python3 (stdlib only -- no PyYAML needed, the bundle is JSON).
set -euo pipefail

usage() {
	echo "usage: replay-answer.sh --bundle <path> --stage <D01|D02|D03|D04|D05> --kind <human_question|research_query> --topic <key>" >&2
	exit 2
}

bundle_path=""
stage=""
kind=""
topic=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--bundle)
		[[ $# -ge 2 ]] || usage
		bundle_path="$2"
		shift 2
		;;
	--stage)
		[[ $# -ge 2 ]] || usage
		stage="$2"
		shift 2
		;;
	--kind)
		[[ $# -ge 2 ]] || usage
		kind="$2"
		shift 2
		;;
	--topic)
		[[ $# -ge 2 ]] || usage
		topic="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$bundle_path" && -n "$stage" && -n "$kind" && -n "$topic" ]] || usage

[[ -f "$bundle_path" ]] || {
	echo "replay-answer: bundle not found: $bundle_path" >&2
	exit 2
}

command -v python3 >/dev/null 2>&1 || {
	echo "replay-answer: python3 not found on PATH" >&2
	exit 2
}

bundle_abs="$(cd "$(dirname "$bundle_path")" && pwd)/$(basename "$bundle_path")"

python3 - "$bundle_abs" "$stage" "$kind" "$topic" <<'PYEOF'
import hashlib
import json
import os
import sys
import time

bundle_path, stage_arg, kind_arg, topic_arg = sys.argv[1:5]
bundle_dir = os.path.dirname(bundle_path)
ledger_path = bundle_path + ".consumption.jsonl"

VALID_STAGES = {"D01", "D02", "D03", "D04", "D05"}
VALID_KINDS = {"human_question", "research_query"}


class ScriptError(RuntimeError):
    """A prerequisite this resolver depends on could not be resolved at all
    -- malformed bundle, malformed ledger, or an unresolvable response path
    -- distinct from a normal replay_desync/unresolved_gate verdict.
    Reported on stderr and mapped to exit 2."""


class Violation(RuntimeError):
    """A real, informative rejection verdict -- mapped to exit 1. Shared
    base for the two named outcomes REQ-F-007 fixes, mirroring
    verify-stage-evidence.sh's Violation/ScriptError split."""

    def __init__(self, kind, detail):
        super().__init__(detail)
        self.kind = kind
        self.detail = detail


def resolve_within(root, rel_path, label):
    """Resolves a bundle-relative response path and enforces containment,
    mirroring verify-stage-evidence.sh's resolve_within /
    verify-evidence-roots.sh's resolve_in_evaluator_root (REQ-NF-005
    posture applied to I-06's own response references)."""
    if os.path.isabs(rel_path):
        raise ScriptError(f"response path must not be absolute: {rel_path!r}")
    root_real = os.path.realpath(root)
    candidate = os.path.realpath(os.path.join(root, rel_path))
    if candidate != root_real and not candidate.startswith(root_real + os.sep):
        raise ScriptError(f"response path escapes bundle directory: {rel_path!r} -> {candidate!r}")
    return candidate


def load_bundle(path):
    try:
        with open(path, "rb") as f:
            raw = f.read()
    except OSError as exc:
        raise ScriptError(f"could not read bundle: {path}: {exc}") from exc
    try:
        doc = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ScriptError(f"bundle is not valid JSON: {path}: {exc}") from exc
    if not isinstance(doc, dict):
        raise ScriptError(f"bundle is not a JSON object: {path}")
    entries = doc.get("entries")
    if not isinstance(entries, list) or not entries:
        raise ScriptError(f"bundle has no non-empty entries[] array: {path}")

    required_fields = {
        "entry_id",
        "stage",
        "ordinal",
        "request_kind",
        "topic_key",
        "required",
        "response",
        "response_digest",
        "entry_digest",
    }
    seen_ordinals_by_stage = {}
    for idx, entry in enumerate(entries):
        if not isinstance(entry, dict):
            raise ScriptError(f"entries[{idx}] is not a JSON object")
        missing = sorted(required_fields - entry.keys())
        if missing:
            raise ScriptError(f"entries[{idx}] ({entry.get('entry_id', '?')}) missing required field(s): {missing}")
        if entry["request_kind"] not in VALID_KINDS:
            raise ScriptError(f"entries[{idx}] ({entry['entry_id']}) has unknown request_kind: {entry['request_kind']!r}")
        if entry["stage"] not in VALID_STAGES:
            raise ScriptError(f"entries[{idx}] ({entry['entry_id']}) has unknown stage: {entry['stage']!r}")
        stage_ordinals = seen_ordinals_by_stage.setdefault(entry["stage"], set())
        if entry["ordinal"] in stage_ordinals:
            raise ScriptError(f"duplicate ordinal {entry['ordinal']} within stage {entry['stage']!r}")
        stage_ordinals.add(entry["ordinal"])
    return entries


def load_consumed_ids(path, stage_arg):
    consumed = set()
    if not os.path.exists(path):
        return consumed
    with open(path) as f:
        for line_no, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ScriptError(f"ledger line {line_no} is not valid JSON: {path}: {exc}") from exc
            if record.get("stage") == stage_arg:
                consumed.add(record.get("entry_id"))
    return consumed


def resolve_response_bytes(entry):
    response = entry["response"]
    if isinstance(response, str):
        return response.encode("utf-8")
    if isinstance(response, dict):
        rel_path = response.get("path")
        declared_digest = response.get("digest")
        if not rel_path or not declared_digest:
            raise ScriptError(f"entry {entry['entry_id']} has a malformed {{path, digest}} response: {response!r}")
        resolved = resolve_within(bundle_dir, rel_path, f"entry {entry['entry_id']} response.path")
        if not os.path.isfile(resolved):
            raise ScriptError(f"entry {entry['entry_id']} response file not found: {resolved}")
        with open(resolved, "rb") as f:
            data = f.read()
        actual_digest = hashlib.sha256(data).hexdigest()
        if actual_digest != declared_digest:
            raise ScriptError(
                f"entry {entry['entry_id']} response file digest mismatch: declared={declared_digest} actual={actual_digest}"
            )
        return data
    raise ScriptError(f"entry {entry['entry_id']} has an unsupported response type: {type(response).__name__}")


def main():
    entries = load_bundle(bundle_path)
    stage_entries = sorted(
        (e for e in entries if e["stage"] == stage_arg),
        key=lambda e: e["ordinal"],
    )
    consumed_ids = load_consumed_ids(ledger_path, stage_arg)
    remaining = [e for e in stage_entries if e["entry_id"] not in consumed_ids]

    if not remaining:
        raise Violation(
            "unresolved_gate",
            f"stage={stage_arg} kind={kind_arg} topic={topic_arg}: no unconsumed entry remains for this stage",
        )

    candidate = remaining[0]  # lowest unconsumed ordinal -- ADR-F07-04
    if candidate["request_kind"] != kind_arg or candidate["topic_key"] != topic_arg:
        raise Violation(
            "replay_desync",
            (
                f"stage={stage_arg} ordinal={candidate['ordinal']}: "
                f"expected kind={candidate['request_kind']!r} topic={candidate['topic_key']!r}, "
                f"supplied kind={kind_arg!r} topic={topic_arg!r}"
            ),
        )

    response_bytes = resolve_response_bytes(candidate)
    computed_digest = hashlib.sha256(response_bytes).hexdigest()
    if computed_digest != candidate["response_digest"]:
        raise ScriptError(
            f"entry {candidate['entry_id']} response_digest mismatch: "
            f"declared={candidate['response_digest']} computed={computed_digest}"
        )

    record = {
        "entry_id": candidate["entry_id"],
        "entry_digest": candidate["entry_digest"],
        "stage": candidate["stage"],
        "ordinal": candidate["ordinal"],
        "request_kind": candidate["request_kind"],
        "topic_key": candidate["topic_key"],
        "response_digest": candidate["response_digest"],
        "supplied_at": time.time(),
    }
    # REQ-F-006: exactly one consumption record per successful supply,
    # appended before the response is printed so a crash after printing
    # can never leave a supply unrecorded.
    with open(ledger_path, "a") as f:
        f.write(json.dumps(record, sort_keys=True))
        f.write("\n")

    sys.stdout.buffer.write(response_bytes)
    sys.stdout.buffer.flush()


try:
    main()
except Violation as violation:
    sys.stderr.write(f"replay-answer: {violation.kind}: {violation.detail}\n")
    sys.exit(1)
except ScriptError as exc:
    sys.stderr.write(f"replay-answer: {exc}\n")
    sys.exit(2)
PYEOF
