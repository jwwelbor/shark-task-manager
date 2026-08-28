#!/usr/bin/env bash
# TC-097 (B053, second pass). filter_expected_by_run_selector()'s naive
# `run_selector.split("/", 1)[0]` does not correctly replicate Go's real
# `-run` semantics: Go's testing.splitRegexp splits a selector on top-level
# `|` INTO AN ALTERNATION FIRST, then splits EACH ALTERNATIVE on `/`
# independently. A selector like "^TestA$/^NoSuchSub$|^TestZ$" makes real
# `go test -run` select and run BOTH TestA and TestZ, but the naive
# split("/", 1)[0] approach yields only {TestA} -- silently dropping TestZ
# from `expected`, reopening the evidence-blindness class this admission
# gate exists to prevent (a test could fail/be hidden without triggering
# the missing-evidence check).
#
# Fix (code review, second pass): validate run_selector against a
# conservative grammar before using the naive split -- reject any selector
# containing `|`, `[`, `(`, or `\` (loudly, RuntimeError -> admit.sh exit
# 2) rather than silently producing a wrong/incomplete `expected` set.
# Within that restricted grammar (anchors ^ $, wildcards . *, and /
# for subtest separators) the naive split IS provably correct.
#
# This is a unit-level test of filter_expected_by_run_selector() in
# isolation, following tc095's pattern of extracting the real function
# definitions out of admit.sh's embedded python body (regex-matched, not
# reimplemented) rather than driving the full go-test/corpus pipeline --
# the grammar check is pure Python logic with no external dependency on a
# real Go package.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"

# shellcheck source=lib/transient-run-selector-corpus.sh
source "$SCRIPT_DIR/lib/transient-run-selector-corpus.sh"

fail() {
	echo "TC-097 FAIL: $1" >&2
	exit 1
}

[[ -f "$ADMIT_SCRIPT" ]] || fail "admit.sh missing: $ADMIT_SCRIPT"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

python3 - "$ADMIT_SCRIPT" <<'PY'
import re
import sys

admit_path = sys.argv[1]

# Extract just the two run_selector helper function definitions (pure
# functions, no dependency on the script's top-level argv/corpus-loading
# state) out of admit.sh's embedded python body, and exec them in an
# isolated namespace. Real code under test, not a reimplementation.
source = open(admit_path, encoding="utf-8").read()
match = re.search(
    r"(_UNSAFE_RUN_SELECTOR_CHARS = .*?\ndef enumerate_tests\()",
    source,
    re.DOTALL,
)
assert match, "could not locate run_selector helper functions in admit.sh"
body = match.group(1)
# Trim the trailing "def enumerate_tests(" anchor back off -- it was only
# needed to bound the DOTALL match at the right function boundary.
body = body[: body.rindex("\ndef enumerate_tests(")]

namespace = {"re": re, "__name__": "admit_under_test"}
exec(compile(body, admit_path, "exec"), namespace)
filter_expected_by_run_selector = namespace["filter_expected_by_run_selector"]

failures = []


def check(desc, fn):
    try:
        fn()
    except AssertionError as exc:
        failures.append(f"{desc}: {exc}")


# --- Case 1: the reviewer's exact repro selector must now be REJECTED
# loudly (RuntimeError), not silently mis-filtered. Confirms the blocker
# is closed: before this fix, this call returned {"TestA"} instead of
# raising -- silently dropping TestZ from `expected` even though real
# `go test -run` selects and runs it.
def case_repro_selector_rejected():
    expected = {"TestA", "TestZ", "TestUnrelated"}
    selector = "^TestA$/^NoSuchSub$|^TestZ$"
    try:
        result = filter_expected_by_run_selector(expected, selector)
    except RuntimeError as exc:
        assert "|" in str(exc) or "unsupported" in str(exc), (
            f"RuntimeError raised but message doesn't explain why: {exc!r}"
        )
        return
    raise AssertionError(
        f"expected RuntimeError for selector {selector!r} containing top-level "
        f"'|', but got a silent (wrong) result instead: {result!r}"
    )


check("case 1 (repro selector must be rejected)", case_repro_selector_rejected)

# --- Case 2: valid grammar preserved -- simple anchored selector, first-pass
# behavior unchanged.
def case_simple_anchor_preserved():
    expected = {"TestTaxAmount", "TestOther"}
    result = filter_expected_by_run_selector(expected, "^TestTaxAmount$")
    assert result == {"TestTaxAmount"}, f"got {result!r}"


check("case 2 (simple anchored selector)", case_simple_anchor_preserved)

# --- Case 3: valid grammar preserved -- "/" (subtest separator) still
# allowed and still only the top-level element before the first "/" is
# used to filter.
def case_subtest_slash_preserved():
    expected = {"TestFoo", "TestBar"}
    result = filter_expected_by_run_selector(expected, "^TestFoo$/^SubTest$")
    assert result == {"TestFoo"}, f"got {result!r}"


check("case 3 (subtest slash selector)", case_subtest_slash_preserved)

# --- Case 4: each banned construct is rejected individually.
def case_banned_constructs_rejected():
    expected = {"TestA", "TestB"}
    for bad_selector in (
        "^Test(A|B)$",       # group + alternation
        "^Test[AB]$",        # character class
        "^Test\\d+$",        # escape
        "^TestA$|^TestB$",   # top-level alternation
    ):
        try:
            result = filter_expected_by_run_selector(expected, bad_selector)
        except RuntimeError:
            continue
        raise AssertionError(
            f"selector {bad_selector!r} should have been rejected, "
            f"got {result!r} instead"
        )


check("case 4 (banned constructs)", case_banned_constructs_rejected)

if failures:
    for f in failures:
        print(f"TC-097 FAIL: {f}", file=sys.stderr)
    sys.exit(1)

print("TC-097: unit-level grammar checks PASS")
PY

# --- Case 5 (end-to-end): drive the real admit.sh process against a
# run_selector Go's real `-run` accepts and runs cleanly (both alternatives
# pass), but which the restricted grammar rejects (top-level `|`). Proves
# the observable process contract the task requires: admit.sh exits 2 (not
# 0 or 1 from a verdict line), with a stderr message naming the rejected
# selector -- not just that the underlying RuntimeError is raised in
# isolation. Also proves no intervening handler in evaluate()/main()
# swallows the RuntimeError into an ordinary rejected-verdict line.
CORPUS_YAML="$SCRIPTS_DIR/../corpus/corpus.yaml"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

TRANSIENT_ITEM="pricing-negative-subtotal"
TRANSIENT_SET="pricing_bad_grammar_selector"
TRANSIENT_CORPUS="$WORKDIR/corpus-bad-grammar.yaml"
# Both TestTaxAmount and TestApplyDiscount are real, passing tests in
# bench/fixture-repo/pkg/pricing/pricing_test.go -- go test -run would
# happily select and run both; only admit.sh's Python-side grammar check
# must reject this selector, on account of the top-level "|".
BAD_SELECTOR='^TestTaxAmount$|^TestApplyDiscount$'

build_transient_run_selector_corpus \
	"$CORPUS_YAML" "$TRANSIENT_CORPUS" "$TRANSIENT_ITEM" "$TRANSIENT_SET" \
	"$BAD_SELECTOR" "./pkg/pricing/..."

set +e
OUT_FILE="$WORKDIR/stdout.log"
ERR_FILE="$WORKDIR/stderr.log"
"$ADMIT_SCRIPT" "$TRANSIENT_CORPUS" --item "$TRANSIENT_ITEM" >"$OUT_FILE" 2>"$ERR_FILE"
CODE=$?
set -e

[[ "$CODE" -eq 2 ]] || fail "expected admit.sh to exit 2 for a rejected-grammar run_selector, got $CODE (stdout: $(cat "$OUT_FILE"), stderr: $(cat "$ERR_FILE"))"
[[ -s "$OUT_FILE" ]] && fail "admit.sh printed a verdict line for a rejected-grammar run_selector -- the RuntimeError must abort before any verdict is emitted: $(cat "$OUT_FILE")"
grep -q "^admit: " "$ERR_FILE" || fail "stderr does not carry the script's own error prefix ('admit: '): $(cat "$ERR_FILE")"
grep -qF "$BAD_SELECTOR" "$ERR_FILE" || fail "stderr does not name the rejected run_selector: $(cat "$ERR_FILE")"

echo "TC-097: end-to-end admit.sh process exits 2 with a clear stderr message for a rejected-grammar run_selector"
echo "TC-097: PASS"
