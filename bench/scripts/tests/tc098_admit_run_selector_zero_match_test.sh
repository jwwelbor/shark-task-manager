#!/usr/bin/env bash
# TC-098 (B053, third pass -- code review finding 1). check_p2p_green()
# filters `expected` through the same run_selector it passes to
# `go test -run` (tc096's fix). But a run_selector that matches ZERO of a
# package's testenum-enumerated tests was, before this fix, never
# distinguished from a run_selector that legitimately narrows `expected`
# to a passing subset (tc096's case): filtering reduces `expected` to the
# empty set, `missing` and `failed` are therefore both trivially empty,
# and `go test -run <no-match>` exits 0 because there is nothing to run --
# so the package silently reads as P2P-green with literally 0 tests
# actually executed.
#
# Fix: check_p2p_green() now tracks, across every package in the p2p_set,
# how many testenum-enumerated tests existed before run_selector filtering
# vs. how many survived it. If tests existed but none survived filtering,
# it raises RuntimeError (admit.sh exit 2) instead of returning a silent
# clean verdict -- see docs/plan/bugs/B053.md.
#
# This test builds a transient corpus.yaml (task spec REQ-F-007 pattern,
# same technique as tc096/tc097): a copy of the real, committed
# corpus.yaml with one addition -- a new p2p_set
# "pricing_zero_match_selector" scoped to ./pkg/pricing/... with a
# run_selector that matches none of that package's real, enumerated test
# names -- and reassigns the already-admitted "pricing-negative-subtotal"
# item to that set. No committed corpus.yaml entry is touched (REQ-F-007).
#
# Caller-Path Contract: same as tc096/tc097 -- the real admit.sh
# entrypoint driving real `go test`, `git apply`, and testenum. Nothing
# about check_p2p_green's own control flow is stubbed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

# shellcheck source=lib/transient-run-selector-corpus.sh
source "$SCRIPT_DIR/lib/transient-run-selector-corpus.sh"

fail() {
	echo "TC-098 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

TRANSIENT_ITEM="pricing-negative-subtotal"
TRANSIENT_SET="pricing_zero_match_selector"
TRANSIENT_CORPUS="$WORKDIR/corpus-zero-match.yaml"
# No test in bench/fixture-repo/pkg/pricing/pricing_test.go is named
# anything like this -- the selector is well-formed (passes the grammar
# check) but matches zero enumerated tests.
ZERO_MATCH_SELECTOR="^TestNoSuchPricingTest$"

build_transient_run_selector_corpus \
	"$CORPUS_YAML" "$TRANSIENT_CORPUS" "$TRANSIENT_ITEM" "$TRANSIENT_SET" \
	"$ZERO_MATCH_SELECTOR" "./pkg/pricing/..."

echo "TC-098: running admit.sh against a p2p_set whose run_selector matches zero enumerated tests"
set +e
OUT_FILE="$WORKDIR/stdout.log"
ERR_FILE="$WORKDIR/stderr.log"
"$ADMIT_SCRIPT" "$TRANSIENT_CORPUS" --item "$TRANSIENT_ITEM" >"$OUT_FILE" 2>"$ERR_FILE"
CODE=$?
set -e

[[ "$CODE" -eq 2 ]] || fail "expected admit.sh to exit 2 for a zero-match run_selector, got $CODE (stdout: $(cat "$OUT_FILE"), stderr: $(cat "$ERR_FILE"))"
[[ -s "$OUT_FILE" ]] && fail "admit.sh printed a verdict line for a zero-match run_selector -- a selector matching nothing must abort loudly before any verdict is emitted (never a silent P2P-green with 0 tests run): $(cat "$OUT_FILE")"
grep -q "^admit: " "$ERR_FILE" || fail "stderr does not carry the script's own error prefix ('admit: '): $(cat "$ERR_FILE")"
grep -qF "$ZERO_MATCH_SELECTOR" "$ERR_FILE" || fail "stderr does not name the zero-match run_selector: $(cat "$ERR_FILE")"

echo "TC-098: end-to-end admit.sh process exits 2 with a clear stderr message for a zero-match run_selector"

# --- Multi-package case (code review, PR #203): the zero-match guard is
# deliberately SET-WIDE (summed across every package in the p2p_set), not
# per-package -- see check_p2p_green()'s docstring in admit.sh. A selector
# legitimately scoped to only one package's tests must NOT trip the guard
# just because it matches zero tests in every OTHER package of a
# multi-package set. Without this case, the set-wide accumulation logic is
# only ever exercised with exactly one package in the loop, which is
# behaviorally indistinguishable from a (wrong) per-package check.
TRANSIENT_SET_MULTIPKG="pricing_multipkg_partial_match"
TRANSIENT_CORPUS_MULTIPKG="$WORKDIR/corpus-multipkg-partial-match.yaml"
# "./pkg/..." enumerates every fixture-repo package (pkg/pricing, pkg/cart,
# pkg/inventory, pkg/validate) as ONE p2p_set. TestTaxAmount exists only in
# pkg/pricing -- the selector matches something there and nothing in the
# other three packages.
build_transient_run_selector_corpus \
	"$CORPUS_YAML" "$TRANSIENT_CORPUS_MULTIPKG" "$TRANSIENT_ITEM" "$TRANSIENT_SET_MULTIPKG" \
	"^TestTaxAmount$" "./pkg/..."

echo "TC-098: running admit.sh against a multi-package p2p_set whose run_selector matches some tests in one package and none in the others"
set +e
OUT_FILE_MULTIPKG="$WORKDIR/stdout-multipkg.log"
ERR_FILE_MULTIPKG="$WORKDIR/stderr-multipkg.log"
"$ADMIT_SCRIPT" "$TRANSIENT_CORPUS_MULTIPKG" --item "$TRANSIENT_ITEM" >"$OUT_FILE_MULTIPKG" 2>"$ERR_FILE_MULTIPKG"
CODE_MULTIPKG=$?
set -e

[[ "$CODE_MULTIPKG" -eq 0 ]] || fail "expected admit.sh to exit 0 (admitted) for a selector that matches something in one package of a multi-package set, got $CODE_MULTIPKG (stdout: $(cat "$OUT_FILE_MULTIPKG"), stderr: $(cat "$ERR_FILE_MULTIPKG"))"

python3 - "$OUT_FILE_MULTIPKG" <<'PYEOF'
import json
import sys

verdict_path = sys.argv[1]
with open(verdict_path) as f:
    line = f.read().strip()

try:
    verdict = json.loads(line)
except json.JSONDecodeError as exc:
    sys.exit(f"TC-098 FAIL: admit.sh did not print a single JSON verdict line for the multi-package case: {line!r} ({exc})")

if verdict["status"] != "admitted":
    sys.exit(
        "TC-098 FAIL: expected status 'admitted' for a multi-package p2p_set whose "
        f"run_selector matches something in one package, got {verdict['status']!r} "
        f"(failing_check={verdict.get('failing_check')!r}) -- the zero-match guard must be "
        f"set-wide, not per-package: {line}"
    )
PYEOF

echo "TC-098: multi-package case correctly admitted -- the zero-match guard is set-wide, not per-package"
echo "TC-098: PASS"
