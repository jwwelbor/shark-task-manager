#!/usr/bin/env bash
# TC-044 (test-plan.md AC test matrix row AC-004; Caller-Path Contracts row
# tc044; T-E40-F06-004 task spec Test Cases, AC-T1 through AC-T5).
#
# Drives bench/scripts/verify-stage-evidence.sh (the ledger-reconciliation
# portion T-E40-F06-004 implements) against real bundle fixtures under
# bench/scripts/testdata/evidence/ledger/ -- REQ-F-005, REQ-F-017,
# REQ-NF-007 (category-attribution branch), ADR-F06-06:
#   (i)   valid-disjoint: six disjoint half-open intervals whose union
#         reconciles exactly within epsilon -> accepted (AC-T1).
#   (ii)  overlap: two intervals from different categories (provider_active,
#         tool_and_test) overlapping by exactly one nanosecond -> rejected
#         naming both categories (AC-T2 case ii).
#   (iii) window-escape: one interval whose end exceeds stage_end by one
#         nanosecond -> rejected naming the interval and the stage window
#         (AC-T2 case iii).
#   (iv)  residual-over-epsilon: an unassigned residual (1000ns) larger than
#         reconciliation_epsilon_ns (100ns) -> rejected naming the residual
#         magnitude (AC-T2 case iv).
#   (v)   residual-within-epsilon: an unassigned residual (50ns) within
#         epsilon (100ns) -> accepted, and the guard's own verdict reports
#         the residual attributed to "unclassified" and absent from
#         provider_active's total (AC-T3).
#   AC-T5: unknown-category plants an interval category ("network_wait")
#         that does not exist in bench/evidence/i05-schema.yaml's
#         interval_category vocabulary -> rejected naming the category,
#         proving the vocabulary is read from that file rather than
#         hardcoded (REQ-F-017).
#   AC-T4/REQ-NF-007: dispatch-boundary-check is a companion fixture whose
#         tool_and_test category's first sub-interval [0,3000) represents
#         the dispatch-boundary check's (REQ-F-011) own elapsed time --
#         accepted, and the guard's per-category totals show that span
#         counted under tool_and_test and never under provider_active.
#
# Caller-Path Contract (test-plan.md tc044): the guard itself performs the
# interval arithmetic against real fixture files on disk; this test never
# reimplements reconciliation/overlap/window-containment math to
# independently "check the guard agrees" -- every expected value below is a
# literal constant computed once by hand while authoring the fixture JSON,
# not derived by a parallel implementation in this script.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
FIXTURES="$SCRIPTS_DIR/testdata/evidence/ledger"

fail() {
	echo "TC-044 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $GUARD"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Case (i): AC-T1 -- six disjoint intervals, exact reconciliation.
# ---------------------------------------------------------------------------
echo "TC-044: case (i) - six disjoint half-open intervals reconcile exactly -> accepted"

out="$WORKDIR/i.out"
err="$WORKDIR/i.err"
set +e
"$GUARD" "$FIXTURES/valid-disjoint" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (i): guard exited $code on a fully-reconciling disjoint ledger, want 0: $(cat "$err")"
grep -q '"result": "accepted"' "$out" || fail "case (i): stdout does not report result=accepted: $(cat "$out")"
grep -q '"residual_ns": 0' "$out" || fail "case (i): stdout does not report residual_ns=0: $(cat "$out")"
grep -q '"provider_active": 400000' "$out" || fail "case (i): stdout does not report provider_active total 400000: $(cat "$out")"

echo "TC-044(case i: disjoint six-category ledger reconciles exactly -> accepted) PASS"

# ---------------------------------------------------------------------------
# Case (ii): AC-T2 -- one-nanosecond overlap between two categories.
# ---------------------------------------------------------------------------
echo "TC-044: case (ii) - one-nanosecond overlap across two categories -> rejected naming both"

err="$WORKDIR/ii.err"
set +e
"$GUARD" "$FIXTURES/overlap" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (ii): guard exited $code on an overlapping ledger, want 1: $(cat "$err")"
grep -q "ledger_overlap" "$err" || fail "case (ii): stderr does not name ledger_overlap: $(cat "$err")"
grep -q "category_a=provider_active" "$err" || fail "case (ii): stderr does not name provider_active: $(cat "$err")"
grep -q "category_b=tool_and_test" "$err" || fail "case (ii): stderr does not name tool_and_test: $(cat "$err")"

echo "TC-044(case ii: 1ns overlap between provider_active and tool_and_test -> rejected naming both categories) PASS"

# ---------------------------------------------------------------------------
# Case (iii): AC-T2 -- an interval escaping the stage window.
# ---------------------------------------------------------------------------
echo "TC-044: case (iii) - interval end exceeds stage_end -> rejected naming interval/window"

err="$WORKDIR/iii.err"
set +e
"$GUARD" "$FIXTURES/window-escape" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (iii): guard exited $code on a window-escaping ledger, want 1: $(cat "$err")"
grep -q "ledger_window_escape" "$err" || fail "case (iii): stderr does not name ledger_window_escape: $(cat "$err")"
grep -q "interval=\[700000,1000001)" "$err" || fail "case (iii): stderr does not name the escaping interval: $(cat "$err")"
grep -q "stage_window=\[0,1000000)" "$err" || fail "case (iii): stderr does not name the stage window: $(cat "$err")"

echo "TC-044(case iii: interval end 1000001 > stage_end 1000000 -> rejected naming interval and window) PASS"

# ---------------------------------------------------------------------------
# Case (iv): AC-T2 -- a residual larger than epsilon left unassigned.
# ---------------------------------------------------------------------------
echo "TC-044: case (iv) - residual (1000ns) > epsilon (100ns) -> rejected naming the magnitude"

err="$WORKDIR/iv.err"
set +e
"$GUARD" "$FIXTURES/residual-over-epsilon" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (iv): guard exited $code on a non-reconciling ledger, want 1: $(cat "$err")"
grep -q "ledger_non_reconciling" "$err" || fail "case (iv): stderr does not name ledger_non_reconciling: $(cat "$err")"
grep -q "residual_ns=1000" "$err" || fail "case (iv): stderr does not name the residual magnitude (1000ns): $(cat "$err")"
grep -q "epsilon_ns=100" "$err" || fail "case (iv): stderr does not name the epsilon (100ns): $(cat "$err")"

echo "TC-044(case iv: 1000ns residual > 100ns epsilon -> rejected naming the residual magnitude) PASS"

# ---------------------------------------------------------------------------
# Case (v): AC-T3 -- a within-epsilon residual is accepted and attributed to
# unclassified, never provider_active.
# ---------------------------------------------------------------------------
echo "TC-044: case (v) - residual (50ns) within epsilon (100ns) -> accepted, attributed to unclassified"

out="$WORKDIR/v.out"
err="$WORKDIR/v.err"
set +e
"$GUARD" "$FIXTURES/residual-within-epsilon" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (v): guard exited $code on a within-epsilon-residual ledger, want 0: $(cat "$err")"
grep -q '"result": "accepted"' "$out" || fail "case (v): stdout does not report result=accepted: $(cat "$out")"
grep -q '"residual_ns": 50' "$out" || fail "case (v): stdout does not report residual_ns=50: $(cat "$out")"
grep -q '"residual_category": "unclassified"' "$out" || fail "case (v): stdout does not attribute the residual to unclassified: $(cat "$out")"
# The residual must never inflate provider_active's own total -- its
# category total must equal exactly the explicit [0,400000) interval this
# fixture declares for it, with no trace of the 50ns residual folded in.
grep -q '"provider_active": 400000' "$out" || fail "case (v): provider_active total was not the fixture's explicit 400000ns -- residual may have leaked into provider_active: $(cat "$out")"

echo "TC-044(case v: 50ns residual within 100ns epsilon -> accepted, unclassified, absent from provider_active) PASS"

# ---------------------------------------------------------------------------
# AC-T5: closed vocabulary read from bench/evidence/i05-schema.yaml -- a
# category outside that vocabulary is rejected naming it.
# ---------------------------------------------------------------------------
echo "TC-044: AC-T5 - interval category outside i05-schema.yaml's vocabulary -> rejected"

err="$WORKDIR/vocab.err"
set +e
"$GUARD" "$FIXTURES/unknown-category" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "AC-T5: guard exited $code on an unknown interval category, want 1: $(cat "$err")"
grep -q "unknown_interval_category" "$err" || fail "AC-T5: stderr does not name unknown_interval_category: $(cat "$err")"
grep -q "category=network_wait" "$err" || fail "AC-T5: stderr does not name the offending category (network_wait): $(cat "$err")"

# Code-discipline half of AC-T5 (mirrors verify-evidence-roots.sh's own
# REQ-F-017 proof for root names): the vocabulary comes from a load of
# i05-schema.yaml, never a hardcoded Python list literal of the six
# category names embedded in the script.
grep -q "i05-schema.yaml" "$GUARD" || fail "AC-T5: verify-stage-evidence.sh does not reference i05-schema.yaml at all"
grep -qE 'schema\.get\("interval_category"\)|schema\["interval_category"\]' "$GUARD" ||
	fail "AC-T5: verify-stage-evidence.sh does not read interval_category from the loaded schema object"
if grep -qE '\[\s*"provider_active"\s*,\s*"tool_and_test"' "$GUARD"; then
	fail "AC-T5: verify-stage-evidence.sh appears to embed a hardcoded interval-category list literal instead of reading i05-schema.yaml"
fi

echo "TC-044(AC-T5: unknown category rejected naming it; vocabulary sourced from i05-schema.yaml, not hardcoded) PASS"

# ---------------------------------------------------------------------------
# AC-T4 / REQ-NF-007 (category-attribution branch): the dispatch-boundary
# check's own elapsed time is categorized tool_and_test, never
# provider_active.
# ---------------------------------------------------------------------------
echo "TC-044: AC-T4/REQ-NF-007 - dispatch-boundary check's own elapsed time categorized tool_and_test"

out="$WORKDIR/boundary.out"
err="$WORKDIR/boundary.err"
set +e
"$GUARD" "$FIXTURES/dispatch-boundary-check" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "AC-T4: guard exited $code on the dispatch-boundary-check fixture, want 0: $(cat "$err")"
grep -q '"result": "accepted"' "$out" || fail "AC-T4: stdout does not report result=accepted: $(cat "$out")"
# Fixture: tool_and_test's first sub-interval [0,3000) IS the dispatch-
# boundary check's own synthetic elapsed time (3000ns); tool_and_test's
# second sub-interval [3000,450000) is the stage's other tool/test work.
# Their sum (450000) must appear entirely under tool_and_test's total.
grep -q '"tool_and_test": 450000' "$out" || fail "AC-T4: tool_and_test total does not include the boundary-check span (want 450000, i.e. 3000 + 447000): $(cat "$out")"
# provider_active in this fixture is declared only as [450000,499970) --
# 49970ns -- entirely disjoint from the [0,3000) boundary-check span; its
# reported total must match exactly, proving no part of that span landed
# there.
grep -q '"provider_active": 49970' "$out" || fail "AC-T4: provider_active total is not exactly the fixture's declared 49970ns span -- boundary-check time may have leaked into provider_active: $(cat "$out")"
# The fixture itself never declares any provider_active interval starting
# at 0 -- the only interval covering [0,3000) in this ledger is tool_and_test's.
python3 -c "
import json
with open('$FIXTURES/dispatch-boundary-check/stages/1-develop.json') as f:
    snap = json.load(f)
provider_intervals = snap['time_ledger']['intervals']['provider_active']
assert all(start != 0 for start, end in provider_intervals), (
    'fixture bug: provider_active declares an interval starting at 0, '
    'which would confound the AC-T4 assertion'
)
tool_intervals = snap['time_ledger']['intervals']['tool_and_test']
assert tool_intervals[0] == [0, 3000], (
    f'fixture bug: expected tool_and_test[0] == [0, 3000] (the boundary-check span), got {tool_intervals[0]!r}'
)
" || fail "AC-T4: fixture self-check failed (see message above)"

echo "TC-044(AC-T4/REQ-NF-007: dispatch-boundary check's own elapsed time attributed to tool_and_test, never provider_active) PASS"

echo "TC-044: PASS"
