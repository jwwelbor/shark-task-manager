#!/usr/bin/env bash
# eval-predicate.sh <package.yaml> <test-output.json> <lint-output.json>
#
# The single named owner of the REQ-F-010 final_predicate arithmetic for all
# four kinds (f2p_p2p, acceptance_tests, p2p_plus_rule_drop,
# child_oracles_union) -- research-report.md Decision 1's single-named-owner
# discipline for I-01's own arithmetic-owning diff tool, carried forward
# here. admit-scenario.sh and later E40-F09 invoke this script instead of
# re-deriving REQ-F-010's semantics themselves.
#
# Reads exactly three files -- its own three positional arguments -- and
# nothing else: <package.yaml>'s final_predicate block, and the `test`/
# `lint` adapter capability documents a caller already produced by invoking
# bench/adapters/<name>/adapter.sh directly. This script never shells out to
# an adapter, never resolves a checkout, and never opens any other path, so
# there is nothing here that could read a stored prior-run comparison file,
# even by accident (ADR-F05-10, REQ-F-017, AC-021; verified mechanically by
# tests/tc041_predicate_argument_trace_test.sh).
#
# Absolute, not relative (REQ-F-017, ADR-F05-10; see also spec.md's Final
# predicate vocabulary paragraph, which is authoritative over the vocabulary
# table's three "regression"-worded cells -- test-plan.md "Acceptance-
# criteria review"). Every kind is evaluated the same way:
#   1. every id this predicate's own operands name -- f2p_test_ids /
#      acceptance_test_ids / (integration_test_ids + child_oracles), none
#      for p2p_plus_rule_drop -- must be present in <test-output.json>'s
#      entries with outcome "pass". child_oracles is assumed to be a list of
#      test ids, same shape as integration_test_ids: nothing else is
#      evaluable from `test`/`lint` output alone per REQ-F-010, and this is
#      the operand shape T-E40-F05-013 (the child_oracles_union seed task)
#      must author against. A missing id is treated as unresolved, not
#      vacuously satisfied.
#   2. every entry <test-output.json> actually contains must be "pass" --
#      this is the p2p_selection clause every kind carries (REQ-F-017): the
#      selection was already resolved into that file by whoever invoked the
#      adapter's `test` capability with its include/exclude_test_ids, so
#      this script itself never touches a checkout or a raw path, and never
#      reads p2p_selection.include/exclude_test_ids at all.
#   3. p2p_plus_rule_drop additionally requires the count of
#      <lint-output.json> issues whose rule matches the predicate's own
#      `rule` operand to be <= `max_remaining` (AC-T2: "<=", not "<").
#
# Prints one JSON document to stdout: {"kind", "result" (the boolean),
# "named_test_ids" (per-id outcome, empty for p2p_plus_rule_drop),
# "p2p_entries" ({"total", "failing"} -- "failing" excludes ids already
# named in "named_test_ids" so it reports only the shared-clause failures,
# not a duplicate of the named-id diagnostics), "lint_rule_count" (present
# only for p2p_plus_rule_drop)} -- the boolean result plus every operand
# value this script actually read, never a comparison-file path, because
# there is none to read.
#
# Exit status reflects only whether the evaluation itself ran to completion
# (valid inputs, a recognized final_predicate.kind, every required operand
# present) -- 0 on a successful evaluation regardless of whether "result" is
# true or false, matching the I-01 diff tool's own exit-status convention:
# a computed answer is never itself encoded as failure. Non-zero is reserved
# for malformed input: missing/unreadable files, an unrecognized kind, or a
# missing operand. Interpreting "result" (e.g. REQ-F-012 checks (d)/(e):
# false at base, true after the reference) is the caller's decision.
set -euo pipefail

usage() {
	echo "usage: eval-predicate.sh <package.yaml> <test-output.json> <lint-output.json>" >&2
	exit 2
}

[[ $# -eq 3 ]] || usage

package_yaml="$1"
test_output_json="$2"
lint_output_json="$3"

[[ -f "$package_yaml" ]] || {
	echo "eval-predicate: package file not found: $package_yaml" >&2
	exit 1
}
[[ -f "$test_output_json" ]] || {
	echo "eval-predicate: test-output file not found: $test_output_json" >&2
	exit 1
}
[[ -f "$lint_output_json" ]] || {
	echo "eval-predicate: lint-output file not found: $lint_output_json" >&2
	exit 1
}

command -v python3 >/dev/null 2>&1 || {
	echo "eval-predicate: python3 not found on PATH" >&2
	exit 1
}
python3 -c 'import yaml' >/dev/null 2>&1 || {
	echo "eval-predicate: python3 module 'yaml' (PyYAML) not available (required to parse package.yaml)" >&2
	exit 1
}

python3 - "$package_yaml" "$test_output_json" "$lint_output_json" <<'PYEOF'
import json
import sys

import yaml

package_yaml_path, test_output_path, lint_output_path = sys.argv[1:4]

KNOWN_KINDS = {"f2p_p2p", "acceptance_tests", "p2p_plus_rule_drop", "child_oracles_union"}


def fail(msg):
    print(f"eval-predicate: {msg}", file=sys.stderr)
    sys.exit(1)


with open(package_yaml_path) as f:
    package = yaml.safe_load(f)

if not isinstance(package, dict):
    fail(f"{package_yaml_path} is not a YAML mapping")

predicate = package.get("final_predicate")
if not isinstance(predicate, dict):
    fail(f"{package_yaml_path} has no final_predicate object")

kind = predicate.get("kind")
if kind not in KNOWN_KINDS:
    fail(f"{package_yaml_path} final_predicate.kind is not one of {sorted(KNOWN_KINDS)}: {kind!r}")

with open(test_output_path) as f:
    test_doc = json.load(f)
entries = test_doc.get("entries")
if not isinstance(entries, list):
    fail(f"{test_output_path} is missing an 'entries' array")
outcome_by_id = {}
for i, entry in enumerate(entries):
    if not isinstance(entry, dict) or "id" not in entry or "outcome" not in entry:
        fail(f"{test_output_path} entries[{i}] is not a {{id, outcome}} object")
    outcome_by_id[entry["id"]] = entry["outcome"]

with open(lint_output_path) as f:
    lint_doc = json.load(f)
issues = lint_doc.get("issues")
if not isinstance(issues, list):
    fail(f"{lint_output_path} is missing an 'issues' array")


def named_ids_for(kind, predicate):
    """Returns the ordered list of operand test ids this kind must find
    'pass' in <test-output.json>, beyond the shared p2p_selection clause
    every kind carries -- empty for p2p_plus_rule_drop, which names no test
    ids of its own (REQ-F-010's Final predicate vocabulary table)."""
    if kind == "f2p_p2p":
        return list(predicate.get("f2p_test_ids") or [])
    if kind == "acceptance_tests":
        return list(predicate.get("acceptance_test_ids") or [])
    if kind == "child_oracles_union":
        return list(predicate.get("integration_test_ids") or []) + list(predicate.get("child_oracles") or [])
    return []  # p2p_plus_rule_drop


named_ids = named_ids_for(kind, predicate)
if kind != "p2p_plus_rule_drop" and not named_ids:
    fail(f"final_predicate.kind {kind!r} names no test ids to evaluate")
named_id_set = set(named_ids)

named_test_ids = {}
named_ok = True
for test_id in named_ids:
    outcome = outcome_by_id.get(test_id)
    named_test_ids[test_id] = outcome if outcome is not None else "missing"
    if outcome != "pass":
        named_ok = False

# The shared p2p_selection clause (REQ-F-017): absolute over every entry
# <test-output.json> actually holds, because the selection was already
# resolved into that file before this script ever ran -- this script itself
# never reads a p2p_selection.include/exclude_test_ids path or touches a
# checkout (AC-T3). Every entry counts toward the boolean, including any
# entry that also happens to be a named id above; the "failing" diagnostic
# list below excludes named ids purely for reporting so a named-id failure
# is not double-reported under both clauses.
all_failing = sorted(tid for tid, outcome in outcome_by_id.items() if outcome != "pass")
p2p_ok = len(outcome_by_id) > 0 and not all_failing
diagnostic_failing = [tid for tid in all_failing if tid not in named_id_set]

result_data = {
    "kind": kind,
    "named_test_ids": named_test_ids,
    "p2p_entries": {"total": len(outcome_by_id), "failing": diagnostic_failing},
}

lint_ok = True
if kind == "p2p_plus_rule_drop":
    rule = predicate.get("rule")
    max_remaining = predicate.get("max_remaining")
    if not rule:
        fail("final_predicate.kind p2p_plus_rule_drop is missing 'rule'")
    if not isinstance(max_remaining, int) or max_remaining < 0:
        fail("final_predicate.kind p2p_plus_rule_drop is missing a non-negative integer 'max_remaining'")
    count = sum(1 for issue in issues if issue.get("rule") == rule)
    lint_ok = count <= max_remaining  # AC-T2: "<=", not "<"
    result_data["lint_rule_count"] = {"rule": rule, "count": count, "max_remaining": max_remaining}

result_data["result"] = bool(named_ok and p2p_ok and lint_ok)

print(json.dumps(result_data, sort_keys=True))
PYEOF
