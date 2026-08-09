#!/usr/bin/env bash
# TC-010 (test-plan.md AC test matrix; T-E40-F01-009 task spec Test Cases).
#
# Exercises AC-009 / AC-T2 (REQ-F-011): `diff-ledgers.sh --kind=test
# --base=<f> --post=<f>` classifies every reachable base/post terminal-action
# transition, using committed, hand-authored synthetic ledgers shaped like
# real build-ledgers.sh output (no fixture checkout, no subprocess).
#
# Caller-Path Contract (test-plan.md TC-010): drives the real
# diff-ledgers.sh --kind=test entrypoint against pure JSON fixtures under
# testdata/test/ -- nothing in the script under test is stubbed or
# special-cased for this fixture's shape.
#
# testdata/test/base.json: {A: pass, B: pass, C: pass, D: fail, E: skip}
# testdata/test/post.json:  {A: pass, B: fail, D: fail, E: fail}  (C absent)
#
# Expected classification:
#   A  pass -> pass    no entry
#   B  pass -> fail    regression (the only reportable regression)
#   C  pass -> absent  removed -- reported separately, never as a regression,
#                       never silently dropped
#   D  fail -> fail    no entry (base was already failing, not a regression)
#   E  skip -> fail    no entry (regression is strictly base-pass -> post-fail)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

DIFF_SCRIPT="$SCRIPTS_DIR/diff-ledgers.sh"
TESTDATA_DIR="$SCRIPTS_DIR/testdata/test"
BASE="$TESTDATA_DIR/base.json"
POST="$TESTDATA_DIR/post.json"

fail() {
	echo "TC-010 FAIL: $1" >&2
	exit 1
}

[[ -x "$DIFF_SCRIPT" ]] || fail "diff-ledgers.sh missing or not executable"
[[ -f "$BASE" ]] || fail "base fixture missing: $BASE"
[[ -f "$POST" ]] || fail "post fixture missing: $POST"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

OUT_FILE="$WORKDIR/diff.out"
"$DIFF_SCRIPT" --kind=test --base="$BASE" --post="$POST" >"$OUT_FILE" ||
	fail "diff-ledgers.sh exited non-zero"

python3 - "$OUT_FILE" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    data = json.load(f)

if data.get("kind") != "test":
    sys.exit(f"TC-010 FAIL: kind field is {data.get('kind')!r}, expected 'test'")

regressions = {e["identity"] for e in data.get("regressions", [])}
removed = {e["identity"] for e in data.get("removed", [])}

# B: pass -> fail is the only regression.
if regressions != {"pkg::B"}:
    sys.exit(f"TC-010 FAIL: regressions={sorted(regressions)}, expected exactly {{'pkg::B'}}")
if data.get("regressions_count") != 1:
    sys.exit(f"TC-010 FAIL: regressions_count={data.get('regressions_count')}, expected 1")

# C: pass -> absent is removed, never a regression, never dropped.
if removed != {"pkg::C"}:
    sys.exit(f"TC-010 FAIL: removed={sorted(removed)}, expected exactly {{'pkg::C'}}")
if data.get("removed_count") != 1:
    sys.exit(f"TC-010 FAIL: removed_count={data.get('removed_count')}, expected 1")

# Removed and regression must be disjoint output classes -- C must never
# also appear as a regression, and B must never also appear as removed.
if regressions & removed:
    sys.exit(f"TC-010 FAIL: regressions and removed overlap: {regressions & removed}")

# D (fail->fail) and E (skip->fail) must produce no entry anywhere: not a
# regression (base was not pass) and not removed (still present at post).
for stale_fail_identity in ("pkg::D", "pkg::E"):
    if stale_fail_identity in regressions:
        sys.exit(f"TC-010 FAIL: {stale_fail_identity} (base fail/skip -> post fail) "
                  "was misreported as a regression")
    if stale_fail_identity in removed:
        sys.exit(f"TC-010 FAIL: {stale_fail_identity} is present at post but was reported as removed")

# A (pass->pass) must produce no entry in either field.
if "pkg::A" in regressions or "pkg::A" in removed:
    sys.exit("TC-010 FAIL: pkg::A (pass -> pass) was reported as a regression or removed")

print("TC-010: regressions={'pkg::B'}, removed={'pkg::C'}, "
      "D (fail->fail) and E (skip->fail) correctly produce no entry")
PYEOF

# --- Regression: structurally incomplete ledger must fail, not "0 result" ---
# UAT round 2 (UAT-004, T-E40-F01-009 rejection): diff-ledgers.sh used to
# read `base.get("entries", [])`, so a ledger-shaped document with no
# "entries" array at all silently scored as zero regressions/zero removed
# instead of failing loudly -- hiding real test regressions behind a
# truncated or malformed post-run ledger. testdata/test/
# malformed_missing_entries.json is a committed, hand-authored ledger
# carrying a valid "toolchain" block but no "entries" key.
check_rejected() {
	local ledger_file="$1" label="$2"
	local ledger="$TESTDATA_DIR/$ledger_file"
	[[ -f "$ledger" ]] || fail "$label: fixture missing: $ledger"

	set +e
	local out
	out="$("$DIFF_SCRIPT" --kind=test --base="$ledger" --post="$BASE" 2>&1)"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "$label: diff-ledgers.sh exited 0 against a structurally invalid ledger -- must fail loudly instead of scoring it as empty/clean: $out"
	echo "$out" | grep -qi "entries" || fail "$label: failure message does not name the missing/malformed 'entries' field: $out"
	echo "TC-010: $label correctly rejected (exit $code): $out"
}

check_rejected "malformed_missing_entries.json" "regression: ledger missing 'entries' array"
check_rejected "malformed_bad_entry.json" "regression: ledger entry missing required field"

echo "TC-010: PASS"
