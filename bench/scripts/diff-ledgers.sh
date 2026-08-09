#!/usr/bin/env bash
# diff-ledgers.sh --kind=<lint|test> --base=<base-ledger.json> --post=<post-ledger.json>
# diff-ledgers.sh --toolchain-guard --base=<ledger.json>
#
# The single tool E40-F02 invokes instead of re-deriving REQ-F-011's diff
# semantics or REQ-F-010's fail-closed toolchain comparison (T-E40-F01-009).
# Two invocation shapes, both operating on ledger-shaped JSON produced by
# build-ledgers.sh (T-E40-F01-008) -- neither needs a fixture checkout.
#
# --kind=lint --base=<f> --post=<f> (AC-007/AC-008, REQ-F-011):
#   Computes the multiset difference over the REQ-F-009 lint identity
#   (from_linter, path, text) -- line/column are excluded by construction,
#   they are not present in the ledger at all. For every identity, the
#   reported new-issue count is max(post_count - base_count, 0): floored at
#   zero rather than negative, and a duplicated identity in the base ledger
#   is tracked at its own depth (a multiset), not collapsed to a single
#   known/unknown flag -- a set-based diff would silently under-report a
#   genuinely re-duplicated issue. Prints
#   {"kind":"lint","new_issues":[...],"new_issues_count":N} to stdout.
#
# --kind=test --base=<f> --post=<f> (AC-009, REQ-F-011):
#   pass(base) -> fail(post) is a regression. An identity present at base
#   and absent at post is reported separately as removed -- never as a
#   regression, never silently dropped. Any base fail/skip entry that is
#   still/newly fail at post produces no entry at all (regression is
#   defined strictly as base-pass becoming post-fail). Prints
#   {"kind":"test","regressions":[...],"regressions_count":N,
#   "removed":[...],"removed_count":N} to stdout.
#
# --toolchain-guard --base=<f> (AC-010, REQ-F-010):
#   Reads <f>'s recorded toolchain block and the live environment (Go
#   version, golangci-lint version, GOOS/GOARCH -- the same `go env`/
#   `golangci-lint version` invocations build-ledgers.sh uses to record
#   them -- and a content hash of the fixture's .golangci.yml, resolved by
#   default from the bench/fixture-repo submodule alongside this script;
#   DIFF_LEDGERS_GOLANGCI_CONFIG overrides that path, for tests that must
#   not mutate the tracked submodule working tree). Exits non-zero, naming
#   every mismatched axis on stderr, before either diff mode above would
#   run. This script is the single named owner of this comparison --
#   build-ledgers.sh only records a toolchain block, it never compares one
#   (see build-ledgers.sh's own header).
#
# This script's own exit code for --kind=lint|test reflects only whether
# the computation itself succeeded (valid ledger JSON in, JSON diff out) --
# it never encodes "new issues found" or "regression found" as failure.
# Interpreting the counts (e.g. failing a run because new_issues_count > 0)
# is the consumer's decision (E40-F02), not this tool's.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_GOLANGCI_CONFIG="$BENCH_DIR/fixture-repo/.golangci.yml"
GOLANGCI_CONFIG="${DIFF_LEDGERS_GOLANGCI_CONFIG:-$DEFAULT_GOLANGCI_CONFIG}"

usage() {
	echo "usage: diff-ledgers.sh --kind=<lint|test> --base=<f> --post=<f>" >&2
	echo "       diff-ledgers.sh --toolchain-guard --base=<f>" >&2
	exit 2
}

kind=""
base=""
post=""
toolchain_guard=0

for arg in "$@"; do
	case "$arg" in
	--kind=*)
		kind="${arg#--kind=}"
		;;
	--base=*)
		base="${arg#--base=}"
		;;
	--post=*)
		post="${arg#--post=}"
		;;
	--toolchain-guard)
		toolchain_guard=1
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$base" ]] || usage
[[ -f "$base" ]] || {
	echo "diff-ledgers: base ledger not found: $base" >&2
	exit 1
}

if [[ "$toolchain_guard" -eq 1 ]]; then
	[[ -z "$kind" && -z "$post" ]] || usage

	command -v go >/dev/null 2>&1 || {
		echo "diff-ledgers: go toolchain not found on PATH" >&2
		exit 1
	}
	command -v golangci-lint >/dev/null 2>&1 || {
		echo "diff-ledgers: golangci-lint not found on PATH" >&2
		exit 1
	}
	[[ -f "$GOLANGCI_CONFIG" ]] || {
		echo "diff-ledgers: golangci config not found: $GOLANGCI_CONFIG" >&2
		exit 1
	}

	go_version="$(go env GOVERSION)"
	goos="$(go env GOOS)"
	goarch="$(go env GOARCH)"
	golangci_lint_raw="$(golangci-lint version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$golangci_lint_raw" ]] || {
		echo "diff-ledgers: could not parse golangci-lint version from 'golangci-lint version'" >&2
		exit 1
	}
	golangci_lint_version="v${golangci_lint_raw}"
	golangci_config_sha256="$(sha256sum "$GOLANGCI_CONFIG" | awk '{print $1}')"

	python3 - "$base" "$go_version" "$golangci_lint_version" "$goos" "$goarch" "$golangci_config_sha256" <<'PYEOF'
import json
import sys

base_path, go_version, golangci_lint_version, goos, goarch, golangci_config_sha256 = sys.argv[1:7]

with open(base_path) as f:
    ledger = json.load(f)

recorded = ledger.get("toolchain") or {}
observed = {
    "go_version": go_version,
    "golangci_lint_version": golangci_lint_version,
    "goos": goos,
    "goarch": goarch,
    "golangci_config_sha256": golangci_config_sha256,
}

mismatches = [
    (field, recorded.get(field), observed_value)
    for field, observed_value in observed.items()
    if recorded.get(field) != observed_value
]

if mismatches:
    for field, recorded_value, observed_value in mismatches:
        print(
            f"diff-ledgers: toolchain mismatch: {field} recorded={recorded_value!r} observed={observed_value!r}",
            file=sys.stderr,
        )
    sys.exit(1)

print("diff-ledgers: toolchain OK -- " + json.dumps(observed, sort_keys=True))
PYEOF
	exit 0
fi

[[ -n "$kind" ]] || usage
[[ "$kind" == "lint" || "$kind" == "test" ]] || usage
[[ -n "$post" ]] || usage
[[ -f "$post" ]] || {
	echo "diff-ledgers: post ledger not found: $post" >&2
	exit 1
}

python3 - "$kind" "$base" "$post" <<'PYEOF'
import json
import sys
from collections import Counter

kind, base_path, post_path = sys.argv[1:4]

# Ledger-schema validation (UAT round 2, UAT-004; defect class: execution
# results without complete terminal evidence accepted as passing or empty
# domain outcomes). A ledger-shaped JSON document missing its required
# "entries" array is structurally incomplete -- a truncated write, a
# producer that crashed before finishing, a hand-authored fixture with a
# typo'd key -- not a legitimate "ran and found nothing" result. Before
# 1007db59 and this fix, `base.get("entries", [])` silently substituted an
# empty list for an absent key, so a corrupt or partial ledger scored as
# clean on both --kind=lint and --kind=test: zero new issues, zero
# regressions, zero removed. A genuinely empty RESULT is still valid and
# must keep working ("entries": [] is accepted below) -- only a
# structurally ABSENT or malformed "entries"/"toolchain"/per-entry field is
# rejected, named, before either diff mode computes anything.
REQUIRED_ENTRY_FIELDS = {
    "lint": {"from_linter": str, "path": str, "text": str},
    "test": {"identity": str, "action": str},
}


def load_ledger(path, kind):
    with open(path) as f:
        doc = json.load(f)
    if not isinstance(doc, dict):
        sys.exit(f"diff-ledgers: {kind} ledger {path} is not a JSON object")
    if not isinstance(doc.get("toolchain"), dict):
        sys.exit(
            f"diff-ledgers: {kind} ledger {path} is missing a 'toolchain' object -- "
            "structurally incomplete ledger, not a valid empty result"
        )
    entries = doc.get("entries")
    if not isinstance(entries, list):
        found = "absent" if "entries" not in doc else type(entries).__name__
        sys.exit(
            f"diff-ledgers: {kind} ledger {path} is missing an 'entries' array (found {found}) "
            "-- structurally incomplete ledger, not a valid empty result"
        )
    required_fields = REQUIRED_ENTRY_FIELDS[kind]
    for i, entry in enumerate(entries):
        if not isinstance(entry, dict):
            sys.exit(f"diff-ledgers: {kind} ledger {path} entries[{i}] is not a JSON object")
        for field, field_type in required_fields.items():
            if field not in entry:
                sys.exit(
                    f"diff-ledgers: {kind} ledger {path} entries[{i}] is missing "
                    f"required field {field!r}"
                )
            if not isinstance(entry[field], field_type):
                sys.exit(
                    f"diff-ledgers: {kind} ledger {path} entries[{i}].{field} has wrong "
                    f"type: expected {field_type.__name__}, got {type(entry[field]).__name__}"
                )
    return entries


if kind not in ("lint", "test"):
    sys.exit(f"diff-ledgers: unknown --kind {kind!r}")

base_entries = load_ledger(base_path, kind)
post_entries = load_ledger(post_path, kind)

if kind == "lint":
    def identity(entry):
        return (entry.get("from_linter", ""), entry.get("path", ""), entry.get("text", ""))

    base_counts = Counter(identity(e) for e in base_entries)
    post_counts = Counter(identity(e) for e in post_entries)

    # Duplicate identities carry the same (from_linter, path, text) fields
    # by construction (REQ-F-009 excludes line/column) -- any post-ledger
    # occurrence is a valid representative to report back for that identity.
    representative = {}
    for e in post_entries:
        representative.setdefault(identity(e), e)

    new_entries = []
    for ident, post_count in post_counts.items():
        new_count = max(post_count - base_counts.get(ident, 0), 0)
        if new_count:
            rep = representative[ident]
            entry = {
                "from_linter": rep.get("from_linter", ""),
                "path": rep.get("path", ""),
                "text": rep.get("text", ""),
            }
            new_entries.extend([entry] * new_count)

    new_entries.sort(key=lambda e: (e["from_linter"], e["path"], e["text"]))

    result = {
        "kind": "lint",
        "new_issues": new_entries,
        "new_issues_count": len(new_entries),
    }

else:  # kind == "test", the only other value load_ledger() accepted above
    base_actions = {e["identity"]: e["action"] for e in base_entries}
    post_actions = {e["identity"]: e["action"] for e in post_entries}

    regressions = sorted(
        identity
        for identity, base_action in base_actions.items()
        if base_action == "pass" and post_actions.get(identity) == "fail"
    )
    removed = sorted(identity for identity in base_actions if identity not in post_actions)

    result = {
        "kind": "test",
        "regressions": [{"identity": i} for i in regressions],
        "regressions_count": len(regressions),
        "removed": [{"identity": i} for i in removed],
        "removed_count": len(removed),
    }

print(json.dumps(result, indent=2, sort_keys=True))
PYEOF
