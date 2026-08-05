#!/usr/bin/env bash
# TC-009 (test-plan.md AC test matrix; T-E40-F01-009 task spec Test Cases).
#
# Exercises AC-008 / AC-T1 (REQ-F-011): `diff-ledgers.sh --kind=lint
# --base=<f> --post=<f>` computes the multiset difference over the
# (from_linter, path, text) identity, floored at zero, against the full
# boundary set -- issue count 0/1/N, duplicate identity, and net-negative --
# using committed, hand-authored synthetic ledgers shaped like real
# build-ledgers.sh output (no fixture checkout, no golangci-lint
# subprocess).
#
# Caller-Path Contract (test-plan.md TC-009): drives the real
# diff-ledgers.sh --kind=lint entrypoint against pure JSON fixtures under
# testdata/lint/ -- nothing in the script under test is stubbed or
# special-cased for this fixture's shape.
#
# testdata/lint/base.json holds 3 issues: identity X (ineffassign,
# pkg/pricing/pricing.go, "ineffectual assignment to n") appearing TWICE,
# and identity Y (staticcheck, pkg/inventory/inventory.go, "unused
# variable foo") once.
#
#   post_identical.json      (a) byte-identical to base           -> 0 new
#   post_plus1_duplicate.json (b) one MORE occurrence of X (X x3)  -> 1 new
#                                 -- a set-based (not multiset) diff would
#                                 see X as "already known" and wrongly
#                                 report 0; this case exists to catch that.
#   post_plus3.json           (c) base + 3 wholly new identities   -> 3 new
#   post_shift_plus1.json     (d) base's X/Y unchanged (identity excludes
#                                 line/column, so a "shifted" issue is
#                                 represented as the same identity) + 1
#                                 genuinely new identity            -> 1 new
#                                 (not 4 -- a line-inclusive identity bug
#                                 would double-count the unchanged entries)
#   post_minus2.json          (e) 2 of the 3 base occurrences removed,
#                                 0 added                           -> 0 new,
#                                 floored at zero rather than negative
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

DIFF_SCRIPT="$SCRIPTS_DIR/diff-ledgers.sh"
TESTDATA_DIR="$SCRIPTS_DIR/testdata/lint"
BASE="$TESTDATA_DIR/base.json"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

fail() {
	echo "TC-009 FAIL: $1" >&2
	exit 1
}

[[ -x "$DIFF_SCRIPT" ]] || fail "diff-ledgers.sh missing or not executable"
[[ -f "$BASE" ]] || fail "base fixture missing: $BASE"

# check_case <post_file> <expected_new_count> <case_label>
check_case() {
	local post_file="$1" expected="$2" label="$3"
	local post="$TESTDATA_DIR/$post_file"
	[[ -f "$post" ]] || fail "$label: post fixture missing: $post"

	local out_file="$WORKDIR/${post_file}.out"
	"$DIFF_SCRIPT" --kind=lint --base="$BASE" --post="$post" >"$out_file" ||
		fail "$label: diff-ledgers.sh exited non-zero"

	python3 - "$label" "$expected" "$out_file" <<'PYEOF'
import json
import sys

label, expected, out_file = sys.argv[1], int(sys.argv[2]), sys.argv[3]
with open(out_file) as f:
    data = json.load(f)

if data.get("kind") != "lint":
    sys.exit(f"TC-009 FAIL: {label}: kind field is {data.get('kind')!r}, expected 'lint'")

count = data.get("new_issues_count")
if count != expected:
    sys.exit(f"TC-009 FAIL: {label}: new_issues_count={count}, expected {expected}")

entries = data.get("new_issues", [])
if len(entries) != expected:
    sys.exit(f"TC-009 FAIL: {label}: len(new_issues)={len(entries)}, expected {expected}")

for e in entries:
    if set(e.keys()) != {"from_linter", "path", "text"}:
        sys.exit(f"TC-009 FAIL: {label}: new_issues entry has unexpected field set: {e}")

sorted_entries = sorted(entries, key=lambda e: (e["from_linter"], e["path"], e["text"]))
if entries != sorted_entries:
    sys.exit(f"TC-009 FAIL: {label}: new_issues is not in fixed sort order")

print(f"TC-009: {label}: new_issues_count={count} OK")
PYEOF
}

check_case "post_identical.json" 0 "case (a) identical"
check_case "post_plus1_duplicate.json" 1 "case (b) duplicate re-occurrence"
check_case "post_plus3.json" 3 "case (c) three new distinct issues"
check_case "post_shift_plus1.json" 1 "case (d) unchanged identities + 1 new"
check_case "post_minus2.json" 0 "case (e) net-negative floored at zero"

# Multiset depth check: the base ledger's duplicated identity (X, appearing
# twice) must not have been silently collapsed into 1 occurrence anywhere
# above -- re-assert directly against the base fixture's own shape.
python3 - "$BASE" <<'PYEOF'
import json
import sys
from collections import Counter

with open(sys.argv[1]) as f:
    base = json.load(f)

def identity(e):
    return (e["from_linter"], e["path"], e["text"])

counts = Counter(identity(e) for e in base["entries"])
duplicated = [ident for ident, n in counts.items() if n > 1]
if not duplicated:
    sys.exit("TC-009 FAIL: base fixture does not contain a duplicated identity -- fixture setup invalid")
if counts[duplicated[0]] != 2:
    sys.exit(f"TC-009 FAIL: expected the duplicated identity at depth 2, got {counts[duplicated[0]]}")

print("TC-009: base fixture duplicate-identity depth confirmed at 2")
PYEOF

echo "TC-009: PASS"
