#!/usr/bin/env bash
# TC-008 (test-plan.md AC test matrix; T-E40-F01-009 task spec Test Cases).
#
# Exercises AC-007 / AC-T4 (REQ-F-009): the lint ledger's identity (from
# build-ledgers.sh, T-E40-F01-008) excludes line AND column. Editing the
# fixture to shift the recorded ineffassign issue's line, then -- in a
# second, independent edit -- its column, and diffing each edited checkout's
# freshly built lint ledger against the committed base lint ledger via
# `diff-ledgers.sh --kind=lint`, must report zero new issues in both cases
# independently.
#
# Caller-Path Contract (test-plan.md TC-008): drives real
# build-ledgers.sh against a real, independently provisioned
# checkout-fixture.sh checkout (line case and column case each get their
# own fresh checkout), which itself shells out to a real golangci-lint
# subprocess -- no line/column movement is faked, it comes from a real
# source edit and a real lint run. diff-ledgers.sh --kind=lint then
# compares the resulting post ledger against the committed base ledger.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

DIFF_SCRIPT="$SCRIPTS_DIR/diff-ledgers.sh"
BUILD_LEDGERS_SCRIPT="$SCRIPTS_DIR/build-ledgers.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
BASE_LINT_LEDGER="$BENCH_DIR/corpus/ledgers/4c24986844b09122e2d516f9bc1ec470b155b441/lint.json"

fail() {
	echo "TC-008 FAIL: $1" >&2
	exit 1
}

[[ -x "$DIFF_SCRIPT" ]] || fail "diff-ledgers.sh missing or not executable"
[[ -x "$BUILD_LEDGERS_SCRIPT" ]] || fail "build-ledgers.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
[[ -f "$BASE_LINT_LEDGER" ]] || fail "committed base lint ledger missing: $BASE_LINT_LEDGER"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$BASE_SHA" ]] || fail "could not derive base_sha from $CORPUS_YAML"

TARGET_FILE="pkg/pricing/pricing.go"
TARGET_PATTERN="n := 1"

# assert_zero_new <label> <post_ledger>
assert_zero_new() {
	local label="$1" post_ledger="$2"
	local out_file="$WORKDIR/${label// /_}.diff.out"
	"$DIFF_SCRIPT" --kind=lint --base="$BASE_LINT_LEDGER" --post="$post_ledger" >"$out_file" ||
		fail "$label: diff-ledgers.sh exited non-zero"

	python3 - "$label" "$out_file" <<'PYEOF'
import json
import sys

label, out_file = sys.argv[1], sys.argv[2]
with open(out_file) as f:
    data = json.load(f)

count = data.get("new_issues_count")
if count != 0:
    sys.exit(f"TC-008 FAIL: {label}: new_issues_count={count}, expected 0 -- "
              f"entries: {data.get('new_issues')}")
print(f"TC-008: {label}: new_issues_count=0 OK")
PYEOF
}

# --- Case 1: line shift only -- insert two blank lines directly above the
# recorded issue's line. The line the issue sits on shifts down by two;
# indentation (and therefore the issue's column) is untouched. ---
line_checkout="$WORKDIR/line-checkout"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$line_checkout" >/dev/null

target_path="$line_checkout/$TARGET_FILE"
[[ -f "$target_path" ]] || fail "target file not found in checkout: $target_path"
grep -q "$TARGET_PATTERN" "$target_path" || fail "target pattern '$TARGET_PATTERN' not found in $target_path -- fixture drifted"

python3 - "$target_path" "$TARGET_PATTERN" <<'PYEOF'
import sys

path, pattern = sys.argv[1], sys.argv[2]
with open(path) as f:
    lines = f.readlines()

idx = next(i for i, line in enumerate(lines) if pattern in line)
lines.insert(idx, "\n")
lines.insert(idx, "\n")

with open(path, "w") as f:
    f.writelines(lines)
PYEOF

grep -n "$TARGET_PATTERN" "$target_path"

line_ledgers="$WORKDIR/line-ledgers"
"$BUILD_LEDGERS_SCRIPT" "$line_checkout" "$line_ledgers" >"$WORKDIR/line-build.log" 2>&1 ||
	fail "build-ledgers.sh failed on the line-shifted checkout: $(cat "$WORKDIR/line-build.log")"

assert_zero_new "line shift" "$line_ledgers/lint.json"

# --- Case 2: column shift only -- a fresh, independent checkout, editing
# the SAME line in place (no lines inserted or removed anywhere in the
# file) by adding leading whitespace before the statement, so the issue's
# line number is unchanged but its column moves. ---
col_checkout="$WORKDIR/col-checkout"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$col_checkout" >/dev/null

col_target_path="$col_checkout/$TARGET_FILE"
[[ -f "$col_target_path" ]] || fail "target file not found in checkout: $col_target_path"

python3 - "$col_target_path" "$TARGET_PATTERN" <<'PYEOF'
import sys

path, pattern = sys.argv[1], sys.argv[2]
with open(path) as f:
    lines = f.readlines()

idx = next(i for i, line in enumerate(lines) if pattern in line)
before = len(lines)
lines[idx] = "\t" + lines[idx]  # extra leading tab: same line, later column
after = len(lines)
assert before == after, "column-shift edit must not change the file's line count"

with open(path, "w") as f:
    f.writelines(lines)
PYEOF

grep -n "$TARGET_PATTERN" "$col_target_path"

col_ledgers="$WORKDIR/col-ledgers"
"$BUILD_LEDGERS_SCRIPT" "$col_checkout" "$col_ledgers" >"$WORKDIR/col-build.log" 2>&1 ||
	fail "build-ledgers.sh failed on the column-shifted checkout: $(cat "$WORKDIR/col-build.log")"

assert_zero_new "column shift" "$col_ledgers/lint.json"

echo "TC-008: PASS"
