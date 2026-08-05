#!/usr/bin/env bash
# build-ledgers.sh <checkout_dir> <output_dir>
#
# Generates the base-SHA test and lint ledgers (REQ-F-008, REQ-F-009) by
# running the fixture's own test/lint-equivalent JSON-emitting commands
# against <checkout_dir> and writing tests.json / lint.json into
# <output_dir>. Mirrors the command sequence in Makefile:86-121 and the
# fixture's own Makefile (research-report.md Decision 2): `go test -json`
# for the test ledger, `golangci-lint run --output.json.path stdout` (the
# v2 flag form) for the lint ledger.
#
# AC-T3 / REQ-F-010: both ledgers RECORD the toolchain that produced them
# (Go version, golangci-lint version, GOOS/GOARCH, a content hash of
# <checkout_dir>/.golangci.yml) but this script never COMPARES that block
# against anything -- toolchain comparison is diff-ledgers.sh
# --toolchain-guard's sole responsibility (T-E40-F01-009).
#
# AC-T1: tests.json contains one entry per <package import path>::<test
# name>, subtests as distinct slash-named entries, with only the terminal
# action (pass/fail/skip) per entry; the raw `go test -json` Elapsed field
# is dropped before writing.
#
# AC-T2: lint.json's issue identity is the tuple (from_linter,
# fixture-relative file path, issue text), excluding line/column.
# Identical identities are kept as repeated entries (a multiset, not a
# set) and the full entry list is written in a fixed sort order so two
# ledgers over the same input are byte-identical files.
#
# AC-T4 / REQ-F-012: this script performs no caching or memoization --
# every invocation re-runs `go test` and `golangci-lint` for real against
# <checkout_dir>, so re-running it against a second, independently
# provisioned checkout of the same base SHA reproduces both ledgers
# byte-for-byte (tests/tc007_ledger_reproducibility_test.sh).
#
# `go test` and `golangci-lint run` both exit non-zero when they observe a
# failing test or a lint issue -- that is expected data for a ledger, not a
# script failure, so this script does not treat their exit codes as fatal.
#
# Defect-class guard (UAT round 1, T-E40-F01-008 rejection): a subprocess
# exiting non-zero is not distinguishable, by exit code alone, from that
# same subprocess crashing or being misconfigured -- both can exit 1. This
# script therefore never infers "zero results" from a subprocess's exit
# code or from empty/unparseable output; it requires a parseable result
# shape before accepting one as ledger data, and fails loudly, naming the
# diagnostic, when a producer cannot supply one:
#   - the lint producer requires the v2 JSON report's "Issues" key to be
#     present after parsing the first stdout line -- a linter that exits
#     non-zero with no report (crash, incompatible flags, config load
#     failure) is refused, not silently written as a zero-issue ledger;
#   - the test producer treats a `go test -json` build-failure event
#     (Action":"fail" with no "Test" field and a "FailedBuild" package) as
#     a hard error naming the package, instead of silently omitting that
#     package's tests from the ledger, which a set-based reader would
#     otherwise indistinguishably read as "package removed";
#   - both producers already require a parseable event before writing
#     anything -- see "no test entries observed" and the lint-report check
#     below.
set -euo pipefail

usage() {
	echo "usage: build-ledgers.sh <checkout_dir> <output_dir>" >&2
	exit 2
}

[[ $# -eq 2 ]] || usage
checkout_dir="$1"
output_dir="$2"

[[ -d "$checkout_dir" ]] || {
	echo "build-ledgers: checkout dir not found: $checkout_dir" >&2
	exit 1
}
[[ -f "$checkout_dir/go.mod" ]] || {
	echo "build-ledgers: not a Go module checkout (no go.mod): $checkout_dir" >&2
	exit 1
}
golangci_config="$checkout_dir/.golangci.yml"
[[ -f "$golangci_config" ]] || {
	echo "build-ledgers: fixture .golangci.yml not found: $golangci_config" >&2
	exit 1
}
command -v go >/dev/null 2>&1 || {
	echo "build-ledgers: go toolchain not found on PATH" >&2
	exit 1
}
command -v golangci-lint >/dev/null 2>&1 || {
	echo "build-ledgers: golangci-lint not found on PATH" >&2
	exit 1
}
command -v python3 >/dev/null 2>&1 || {
	echo "build-ledgers: python3 not found on PATH" >&2
	exit 1
}

checkout_dir="$(cd "$checkout_dir" && pwd)"
mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd)"

# --- Toolchain block (recorded, never compared here) ------------------
go_version="$(go env GOVERSION)"
goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
golangci_lint_raw="$(golangci-lint version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
[[ -n "$golangci_lint_raw" ]] || {
	echo "build-ledgers: could not parse golangci-lint version from 'golangci-lint version'" >&2
	exit 1
}
golangci_lint_version="v${golangci_lint_raw}"
golangci_config_sha256="$(sha256sum "$golangci_config" | awk '{print $1}')"

# --- Test ledger: real `go test -json ./...` ---------------------------
test_raw="$(mktemp)"
lint_raw="$(mktemp)"
lint_err="$(mktemp)"
trap 'rm -f "$test_raw" "$lint_raw" "$lint_err"' EXIT
(cd "$checkout_dir" && go test -json ./...) >"$test_raw" 2>&1 || true

# --- Lint ledger: real `golangci-lint run --output.json.path stdout` ---
# Exit code alone cannot distinguish "issues found" (expected, non-zero)
# from "crashed before producing a report" (also non-zero, sometimes even
# the same code) -- stdout and stderr are captured separately so the
# python parser below can require a parseable report and surface the real
# diagnostic on failure, rather than treating `|| true` as "no issues".
lint_exit=0
(cd "$checkout_dir" && golangci-lint run --output.json.path stdout) >"$lint_raw" 2>"$lint_err" || lint_exit=$?

python3 - "$test_raw" "$lint_raw" "$lint_err" "$lint_exit" "$output_dir" \
	"$go_version" "$golangci_lint_version" "$goos" "$goarch" "$golangci_config_sha256" <<'PYEOF'
import json
import sys

test_raw_path, lint_raw_path, lint_err_path, lint_exit, output_dir, go_version, golangci_lint_version, goos, goarch, golangci_config_sha256 = sys.argv[1:11]

toolchain = {
    "go_version": go_version,
    "golangci_lint_version": golangci_lint_version,
    "goos": goos,
    "goarch": goarch,
    "golangci_config_sha256": golangci_config_sha256,
}

# --- tests.json ---------------------------------------------------------
# `go test -json` emits one event per line. Terminal per-test events carry
# a non-null "Test" field and an Action of pass/fail/skip; package-level
# summary events (no "Test" field) are not test identities and are
# dropped. "Elapsed" is dropped by construction: it is never read below.
#
# A package that fails to *build* never emits per-test events at all -- Go
# reports it instead as a package-level {"Action":"fail","FailedBuild":
# "<pkg>"} event with no "Test" field. Left unguarded, that package's
# tests would simply be absent from tests.json -- indistinguishable, to
# any downstream reader, from "those tests were legitimately removed"
# (diff-ledgers.sh --kind=test's own "removed" classification). That is
# the same defect class as the lint gap below wearing a different face: a
# subprocess transport/build failure silently becomes an empty domain
# result. Any build-failed package is therefore a hard error here, named,
# before any ledger is written.
terminal_actions = {"pass", "fail", "skip"}
entries_by_identity = {}
build_failed_packages = set()
with open(test_raw_path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except ValueError:
            continue
        action = event.get("Action")
        test_name = event.get("Test")
        package = event.get("Package")
        if action == "fail" and not test_name and package and event.get("FailedBuild"):
            build_failed_packages.add(package)
            continue
        if not test_name or action not in terminal_actions:
            continue
        if not package:
            continue
        identity = f"{package}::{test_name}"
        entries_by_identity[identity] = action

if build_failed_packages:
    sys.exit(
        "build-ledgers: 'go test -json' reported a build failure for package(s) "
        f"{sorted(build_failed_packages)} -- refusing to write a test ledger that "
        "would silently omit that package's tests instead of naming the failure"
    )

if not entries_by_identity:
    sys.exit("build-ledgers: no test entries observed from 'go test -json' -- "
             "checkout produced no test output")

test_entries = [
    {"identity": identity, "action": entries_by_identity[identity]}
    for identity in sorted(entries_by_identity)
]

# --- lint.json -----------------------------------------------------------
# golangci-lint's JSON report is the first line of stdout; any trailing
# human-readable summary lines are ignored. Identity is
# (from_linter, fixture-relative path, text) -- line/column are recorded
# nowhere in the ledger (ADR-F01-03). Identical identities are kept as
# repeated entries (a multiset), not deduplicated.
#
# UAT round 1 finding (defect class: transport failure misclassified as an
# empty domain result): `golangci-lint run` exits non-zero both when it
# reports real issues (expected) and when it crashes or is misconfigured
# before emitting any report (a producer failure) -- the two are not
# distinguishable by exit code alone, so this script never infers "zero
# issues" from a non-zero exit or from empty/unparseable stdout. A
# parseable v2 report (JSON object with an "Issues" key -- present, as an
# empty list, even on a genuine zero-issue run) is REQUIRED before any
# lint.json is written; its absence is a hard, named error carrying the
# real exit code and captured stderr, not a silently-accepted empty
# ledger.
lint_report = None
with open(lint_raw_path) as f:
    first_line = f.readline().strip()
if first_line:
    try:
        candidate = json.loads(first_line)
    except ValueError:
        candidate = None
    if isinstance(candidate, dict) and "Issues" in candidate:
        lint_report = candidate

if lint_report is None:
    with open(lint_err_path) as f:
        stderr_text = f.read().strip()
    sys.exit(
        "build-ledgers: 'golangci-lint run --output.json.path stdout' did not "
        f"produce a parseable v2 JSON report (exit {lint_exit}); refusing to "
        "write an empty lint ledger. captured stderr: "
        f"{stderr_text or '(empty)'} | captured stdout (first 500 chars): "
        f"{first_line[:500]!r}"
    )

lint_entries = []
for issue in lint_report.get("Issues") or []:
    pos = issue.get("Pos") or {}
    lint_entries.append({
        "from_linter": issue.get("FromLinter", ""),
        "path": pos.get("Filename", ""),
        "text": issue.get("Text", ""),
    })

lint_entries.sort(key=lambda e: (e["from_linter"], e["path"], e["text"]))

# --- Write both ledgers together ----------------------------------------
# Both producers above are fully validated by this point (build failures
# and unparseable lint reports already exited non-zero). Writing both
# files only now, after both are known-good, means a run that fails
# partway through validation never leaves a lone tests.json or lint.json
# on disk that a caller could mistake for a complete, successful build.
with open(f"{output_dir}/tests.json", "w") as f:
    json.dump({"toolchain": toolchain, "entries": test_entries}, f, indent=2, sort_keys=True)
    f.write("\n")

with open(f"{output_dir}/lint.json", "w") as f:
    json.dump({"toolchain": toolchain, "entries": lint_entries}, f, indent=2, sort_keys=True)
    f.write("\n")

print(f"build-ledgers: wrote {len(test_entries)} test entries, {len(lint_entries)} lint entries")
print(f"build-ledgers: toolchain: {json.dumps(toolchain, sort_keys=True)}")
PYEOF
