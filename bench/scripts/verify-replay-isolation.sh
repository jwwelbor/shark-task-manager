#!/usr/bin/env bash
# verify-replay-isolation.sh <transcript_path>
#
# T-E40-F07-004 slice: REQ-F-005's OBSERVATIONAL transcript scan -- the
# BINDING half of the live-egress two-half proof (spec.md "Live-egress set
# and its two-half proof"; ADR-F07-03). Scans a retained scored-run
# transcript for tool-use records naming any bench/replay/live-egress-tools.yaml
# member; a single such record stops the scenario with the named terminal
# outcome `live_interaction_reached`, identifying the tool name and the
# stage. This holds regardless of whether run-prelude.sh's --disallowedTools
# denial (structural, belt-and-braces) actually worked -- REQ-F-004's
# argument-vector check is not treated as proof by itself (REQ-F-005).
#
# Called with exactly ONE positional argument (a transcript path), this
# script runs the transcript-scan half only. T-E40-F07-006 MODIFIES this
# same file to add a second invocation shape --
#   verify-replay-isolation.sh <bundle_path> <fixture_checkout> <scratch_project>
# -- the REQ-F-012 bulk-disclosure guard, dispatched on argument COUNT
# rather than a subcommand keyword (three positional args, not one). This
# task's Brownfield Context requires the transcript-scan portion below to be
# left untouched by that later change.
#
# Retained-transcript format (this task's own, first definition -- no prior
# convention in this repo fixes a shape for it): JSONL, one JSON object per
# non-blank line. A `{"type": "tool_use", "stage": "D0X", "tool_name": "..."}`
# record is a tool invocation; `stage` MUST be one of
# bench/replay/i06-schema.yaml's `stage:` vocabulary (REQ-F-018: read the
# single vocabulary owner, never a private copy). Any other `type` value is
# ignored by this scan. A tool_use record whose `tool_name` is not itself a
# live-egress-set member is not a violation -- ordinary tool use (Read,
# Write, Bash for replay-answer.sh, ...) is expected and permitted; only a
# live-egress-set member's name triggers `live_interaction_reached`.
#
# FAILS CLOSED, NEVER OPEN (ADR-F07-03): a transcript in which this scanner
# recognizes ZERO tool_use records anywhere -- e.g. one in a shape this
# scanner does not understand -- is refused (ScriptError, exit 2), never
# silently certified CLEAN. The binding gate must not rest on an assumption
# about transcript shape that can stop holding without this script noticing;
# a real D01-D05 session always uses at least one tool.
#
# Exit status: 0 = transcript clean ("CLEAN" on stdout). 1 =
# `live_interaction_reached` -- a normal, informative verdict; the message
# on stderr names the tool, the stage, the transcript line number, and the
# transcript path. 2 = a script/usage/authoring error (bad args, missing
# files, malformed live-egress-tools.yaml, malformed i06-schema.yaml,
# malformed transcript JSON, a tool_use record with a missing/unknown stage
# or tool_name, or a transcript with zero recognized tool_use records).
#
# Requires: python3 with PyYAML (`import yaml`) on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
LIVE_EGRESS_FILE="$BENCH_DIR/replay/live-egress-tools.yaml"
I06_SCHEMA="$BENCH_DIR/replay/i06-schema.yaml"

usage() {
	echo "usage: verify-replay-isolation.sh <transcript_path>" >&2
	exit 2
}

[[ $# -eq 1 ]] || usage
transcript_path="$1"

[[ -f "$transcript_path" ]] || {
	echo "verify-replay-isolation: transcript not found: $transcript_path" >&2
	exit 2
}
[[ -f "$LIVE_EGRESS_FILE" ]] || {
	echo "verify-replay-isolation: live-egress-tools.yaml not found: $LIVE_EGRESS_FILE" >&2
	exit 2
}
[[ -f "$I06_SCHEMA" ]] || {
	echo "verify-replay-isolation: i06-schema.yaml not found: $I06_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-replay-isolation: python3 not found on PATH" >&2
	exit 2
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "verify-replay-isolation: python3 module 'yaml' (PyYAML) not available" >&2
	exit 2
}

transcript_path_abs="$(cd "$(dirname "$transcript_path")" && pwd)/$(basename "$transcript_path")"

python3 - "$transcript_path_abs" "$LIVE_EGRESS_FILE" "$I06_SCHEMA" <<'PYEOF'
import json
import sys

import yaml

transcript_path, live_egress_path, i06_schema_path = sys.argv[1:4]


class ScriptError(RuntimeError):
    """A prerequisite this guard depends on could not be resolved at all --
    malformed live-egress-tools.yaml/i06-schema.yaml, or a malformed
    transcript record -- distinct from a normal live_interaction_reached
    verdict. Reported on stderr and mapped to exit 2."""


def load_live_egress_members(path):
    with open(path) as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        raise ScriptError(f"live-egress-tools.yaml is not a YAML mapping: {path}")
    tools = data.get("tools")
    if not isinstance(tools, list) or not tools:
        raise ScriptError(f"live-egress-tools.yaml declares no tools[] entries: {path}")
    members = set()
    for idx, entry in enumerate(tools):
        if not isinstance(entry, dict) or not entry.get("name"):
            raise ScriptError(f"live-egress-tools.yaml tools[{idx}] is missing a required name: {entry!r}")
        members.add(entry["name"])
    return members


def load_valid_stages(path):
    # REQ-F-018: stage vocabulary has exactly one owner (i06-schema.yaml);
    # this script reads it rather than embedding a private D01-D05 copy.
    with open(path) as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        raise ScriptError(f"i06-schema.yaml is not a YAML mapping: {path}")
    stages = data.get("stage")
    if not isinstance(stages, list) or not stages:
        raise ScriptError(f"i06-schema.yaml declares no stage: vocabulary: {path}")
    return set(stages)


def scan_transcript(path, live_egress_members, valid_stages):
    """Returns (violation, recognized_count). violation is (tool_name, stage,
    lineno) for the FIRST tool_use record naming a live-egress-set member, or
    None if no such record was found. recognized_count is the number of
    records this scanner recognized as a tool_use record (type == "tool_use"
    with a well-formed tool_name/stage) anywhere in the transcript --
    regardless of whether any of them named a live-egress-set member.

    recognized_count exists so the binding gate (ADR-F07-03) cannot fail
    OPEN on a transcript in a shape this scanner does not recognize at all
    (e.g. a real provider transcript's own native record shape, which this
    task does not define -- see the module docstring). A transcript this
    scanner cannot recognize as containing ANY tool_use record is refused,
    not silently certified clean; main() below is the single caller that
    enforces that. Parses real JSON per line -- REQ-NF-004/test-plan.md
    Caller-Path Contract: never a boolean "contains a violation" flag."""
    violation = None
    recognized_count = 0
    with open(path) as f:
        for lineno, raw_line in enumerate(f, start=1):
            line = raw_line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except ValueError as exc:
                raise ScriptError(f"transcript line {lineno} is not valid JSON: {path}: {exc}") from exc
            if not isinstance(record, dict):
                raise ScriptError(f"transcript line {lineno} is not a JSON object: {path}")
            if record.get("type") != "tool_use":
                continue
            tool_name = record.get("tool_name")
            if not isinstance(tool_name, str) or not tool_name:
                raise ScriptError(f"transcript line {lineno}: tool_use record has no tool_name: {path}")
            stage = record.get("stage")
            if not isinstance(stage, str) or stage not in valid_stages:
                raise ScriptError(
                    f"transcript line {lineno}: tool_use record has a missing/unknown stage {stage!r}: {path}"
                )
            recognized_count += 1
            if violation is None and tool_name in live_egress_members:
                violation = (tool_name, stage, lineno)
    return violation, recognized_count


def main():
    live_egress_members = load_live_egress_members(live_egress_path)
    valid_stages = load_valid_stages(i06_schema_path)
    violation, recognized_count = scan_transcript(transcript_path, live_egress_members, valid_stages)
    if violation is not None:
        tool_name, stage, lineno = violation
        sys.stderr.write(
            "verify-replay-isolation: live_interaction_reached: "
            f"tool={tool_name} stage={stage} line={lineno} transcript={transcript_path}\n"
        )
        sys.exit(1)
    if recognized_count == 0:
        # Fail closed, never open: a transcript this scanner recognized zero
        # tool_use records in is NOT proof of isolation -- it may simply be
        # in a shape this scanner does not understand (ADR-F07-03: the
        # binding gate must not rest on an assumption that can silently stop
        # holding). A real D01-D05 session always uses at least one tool
        # (Read/Write/Bash at minimum), so a genuinely clean transcript has
        # recognized_count > 0.
        raise ScriptError(
            "transcript contains zero records this scanner recognizes as a tool_use record "
            '(a JSON object with "type": "tool_use", a non-empty "tool_name", and a "stage" in '
            f"i06-schema.yaml's stage: vocabulary) -- refusing to certify CLEAN: {transcript_path}"
        )
    print("CLEAN")


try:
    main()
except ScriptError as exc:
    sys.stderr.write(f"verify-replay-isolation: {exc}\n")
    sys.exit(2)
PYEOF
