#!/usr/bin/env bash
# TC-007 (test-plan.md AC test matrix; T-E40-F01-008 task spec Test Cases).
#
# Exercises AC-006 / AC-T1 / AC-T4 (REQ-F-008, REQ-F-012): `build-ledgers.sh
# <checkout_dir> <output_dir>`, invoked once per independently provisioned
# checkout of the same base SHA, produces:
#   - one tests.json entry per <package>::<test>, subtests as distinct
#     slash-named entries, covering all three terminal actions
#     (pass/fail/skip);
#   - a byte-identical entry set for BOTH tests.json and lint.json across
#     the two independently provisioned checkouts.
#
# "Independently provisioned" (Determinism boundary, test-plan.md) means
# two separate checkout-fixture.sh invocations into two different temp
# directories -- not one checkout read twice -- so a build-ledgers.sh that
# memoized or re-read its own prior output instead of re-running `go test`
# and `golangci-lint` could not pass this test by accident.
#
# Caller-Path Contract (test-plan.md TC-007): drives the real
# build-ledgers.sh entrypoint, which itself shells out to real
# `go test -json` and a real `golangci-lint` subprocess inside fresh
# checkout-fixture.sh checkouts each round -- no subprocess output is
# stubbed, no ledger is read back instead of regenerated.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

BUILD_LEDGERS_SCRIPT="$SCRIPTS_DIR/build-ledgers.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-007 FAIL: $1" >&2
	exit 1
}

[[ -x "$BUILD_LEDGERS_SCRIPT" ]] || fail "build-ledgers.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

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

# run_round <round_dir> -> provisions its own fresh checkout-fixture.sh
# checkout of $BASE_SHA and runs build-ledgers.sh against it. No state is
# shared between rounds: each gets its own checkout directory and its own
# output directory.
run_round() {
	local round_dir="$1"
	local checkout_dir="$round_dir/checkout"
	local output_dir="$round_dir/ledgers"

	"$CHECKOUT_SCRIPT" "$BASE_SHA" "$checkout_dir" >/dev/null

	set +e
	"$BUILD_LEDGERS_SCRIPT" "$checkout_dir" "$output_dir" >"$round_dir/build.log" 2>&1
	echo $? >"$round_dir/build.exit"
	set -e
}

echo "TC-007: round 1 - independently provisioned checkout"
run_round "$WORKDIR/round1"
[[ "$(cat "$WORKDIR/round1/build.exit")" -eq 0 ]] || fail "build-ledgers.sh did not exit 0 in round 1: $(cat "$WORKDIR/round1/build.log")"

echo "TC-007: round 2 - independently provisioned checkout"
run_round "$WORKDIR/round2"
[[ "$(cat "$WORKDIR/round2/build.exit")" -eq 0 ]] || fail "build-ledgers.sh did not exit 0 in round 2: $(cat "$WORKDIR/round2/build.log")"

TESTS_1="$WORKDIR/round1/ledgers/tests.json"
TESTS_2="$WORKDIR/round2/ledgers/tests.json"
LINT_1="$WORKDIR/round1/ledgers/lint.json"
LINT_2="$WORKDIR/round2/ledgers/lint.json"

for f in "$TESTS_1" "$TESTS_2" "$LINT_1" "$LINT_2"; do
	[[ -f "$f" ]] || fail "expected ledger file missing: $f"
done

echo "TC-007: comparing round 1 vs round 2 for byte-identical ledgers"

if ! cmp -s "$TESTS_1" "$TESTS_2"; then
	fail "tests.json round 1 and round 2 are not byte-identical: $(diff "$TESTS_1" "$TESTS_2" | head -20)"
fi
echo "TC-007: tests.json byte-identical across both rounds"

if ! cmp -s "$LINT_1" "$LINT_2"; then
	fail "lint.json round 1 and round 2 are not byte-identical: $(diff "$LINT_1" "$LINT_2" | head -20)"
fi
echo "TC-007: lint.json byte-identical across both rounds"

# Elapsed must never appear in the written test ledger (AC-T1).
if grep -q '"Elapsed"' "$TESTS_1"; then
	fail "tests.json contains an 'Elapsed' field -- must be dropped before writing"
fi

python3 - "$TESTS_1" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    data = json.load(f)

entries = data["entries"]
if not entries:
    sys.exit("TC-007 FAIL: tests.json has zero entries")

identities = [e["identity"] for e in entries]
if len(identities) != len(set(identities)):
    sys.exit("TC-007 FAIL: tests.json has duplicate identities -- one entry per <package>::<test> required")

actions = {e["action"] for e in entries}
missing = {"pass", "fail", "skip"} - actions
if missing:
    sys.exit(f"TC-007 FAIL: tests.json is missing terminal action(s) {sorted(missing)} -- "
              "fixture suite must exercise pass/fail/skip and the ledger must record all three")

for e in entries:
    if set(e.keys()) != {"identity", "action"}:
        sys.exit(f"TC-007 FAIL: tests.json entry has unexpected fields (Elapsed must be dropped): {e}")

# Subtests must appear as distinct slash-named entries under their parent,
# not collapsed into the parent's identity (AC-006 negative case).
subtest_identities = [i for i in identities if "::TestApplyDiscount/" in i]
if len(subtest_identities) < 2:
    sys.exit("TC-007 FAIL: expected TestApplyDiscount subtests as distinct slash-named "
              f"entries, found {len(subtest_identities)}: {subtest_identities}")
parent_identity = [i for i in identities if i.endswith("::TestApplyDiscount")]
if not parent_identity:
    sys.exit("TC-007 FAIL: expected a distinct entry for the TestApplyDiscount parent test itself")

toolchain = data["toolchain"]
required_toolchain_fields = {"go_version", "golangci_lint_version", "goos", "goarch", "golangci_config_sha256"}
if set(toolchain.keys()) != required_toolchain_fields:
    sys.exit(f"TC-007 FAIL: tests.json toolchain block has unexpected field set: {sorted(toolchain.keys())}")
for field in required_toolchain_fields:
    if not toolchain[field]:
        sys.exit(f"TC-007 FAIL: tests.json toolchain.{field} is empty")

print(f"TC-007: tests.json has {len(entries)} entries covering actions {sorted(actions)}")
PYEOF

python3 - "$LINT_1" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    data = json.load(f)

toolchain = data["toolchain"]
required_toolchain_fields = {"go_version", "golangci_lint_version", "goos", "goarch", "golangci_config_sha256"}
if set(toolchain.keys()) != required_toolchain_fields:
    sys.exit(f"TC-007 FAIL: lint.json toolchain block has unexpected field set: {sorted(toolchain.keys())}")
for field in required_toolchain_fields:
    if not toolchain[field]:
        sys.exit(f"TC-007 FAIL: lint.json toolchain.{field} is empty")

for e in data["entries"]:
    if set(e.keys()) != {"from_linter", "path", "text"}:
        sys.exit(f"TC-007 FAIL: lint.json entry excludes only line/column but has unexpected field set: {e}")
    if "line" in e or "column" in e or "Line" in e or "Column" in e:
        sys.exit(f"TC-007 FAIL: lint.json entry retains line/column, which must be excluded from identity: {e}")

print(f"TC-007: lint.json has {len(data['entries'])} entries, toolchain block present")
PYEOF

# --- Regression: version-valid golangci-lint that crashes with no report ---
# UAT round 1 (T-E40-F01-008 rejection): a temporary golangci-lint
# executable that correctly reports version 2.9.0 for `version` but exits
# non-zero with no JSON report for `run` made build-ledgers.sh exit 0 and
# silently write a zero-entry lint.json -- a linter crash or
# misconfiguration became indistinguishable from "genuinely zero lint
# issues", which would corrupt every downstream diff-ledgers.sh comparison.
# This section pins that regression: the stub is version-valid on purpose,
# so a check that only compares the reported version (instead of requiring
# a parseable report) could not catch it.
echo "TC-007: regression - version-valid golangci-lint that crashes with no report"

STUB_BIN_DIR="$WORKDIR/stub-bin"
mkdir -p "$STUB_BIN_DIR"
cat >"$STUB_BIN_DIR/golangci-lint" <<'STUBEOF'
#!/usr/bin/env bash
if [[ "$1" == "version" ]]; then
	echo "golangci-lint has version 2.9.0 built with go1.26.0 from deadbeef on 2026-01-01T00:00:00Z"
	exit 0
fi
# Version-valid but crashes on `run` -- no JSON report on stdout, matching
# the UAT reproduction exactly (exit 1, empty stdout, stderr diagnostic).
echo "stub golangci-lint: simulated config load failure" >&2
exit 1
STUBEOF
chmod +x "$STUB_BIN_DIR/golangci-lint"

regression_dir="$WORKDIR/regression"
regression_checkout="$regression_dir/checkout"
regression_output="$regression_dir/ledgers"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$regression_checkout" >/dev/null

set +e
PATH="$STUB_BIN_DIR:$PATH" "$BUILD_LEDGERS_SCRIPT" "$regression_checkout" "$regression_output" >"$regression_dir/build.log" 2>&1
regression_exit=$?
set -e

[[ "$regression_exit" -ne 0 ]] || fail "build-ledgers.sh exited 0 against a crashing golangci-lint stub -- must fail loudly instead of accepting an empty lint ledger"

if [[ -e "$regression_output/lint.json" ]]; then
	fail "build-ledgers.sh wrote lint.json despite golangci-lint producing no parseable report: $(cat "$regression_output/lint.json")"
fi
if [[ -e "$regression_output/tests.json" ]]; then
	fail "build-ledgers.sh wrote tests.json despite failing lint validation -- ledgers must be written all-or-nothing"
fi

grep -qi "golangci-lint" "$regression_dir/build.log" || fail "failure diagnostic does not name golangci-lint: $(cat "$regression_dir/build.log")"
grep -qi "report" "$regression_dir/build.log" || fail "failure diagnostic does not mention the missing/unparseable report: $(cat "$regression_dir/build.log")"

echo "TC-007: crashing-linter regression correctly rejected (exit $regression_exit, no ledger written)"

# --- Regression: package that fails to BUILD (TD-075) ----------------------
# Code-review round-3 non-blocker (TD-075): commit 1007db59's build-failure
# guard -- a `go test -json` package-level {"Action":"fail","FailedBuild":
# "<pkg>"} event must hard-fail the builder, naming the package, instead of
# silently omitting that package's tests from tests.json -- shipped with no
# dedicated committed regression test. This section adds it: a real
# checkout with one package's source deliberately made unbuildable.
echo "TC-007: regression - go test -json build failure (TD-075)"

buildfail_dir="$WORKDIR/buildfail"
buildfail_checkout="$buildfail_dir/checkout"
buildfail_output="$buildfail_dir/ledgers"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$buildfail_checkout" >/dev/null

buildfail_cart_go="$buildfail_checkout/pkg/cart/cart.go"
[[ -f "$buildfail_cart_go" ]] || fail "build-failure regression: pkg/cart/cart.go not found in checkout"
printf '\nthis is not valid go syntax {{{\n' >>"$buildfail_cart_go"

set +e
"$BUILD_LEDGERS_SCRIPT" "$buildfail_checkout" "$buildfail_output" >"$buildfail_dir/build.log" 2>&1
buildfail_exit=$?
set -e

[[ "$buildfail_exit" -ne 0 ]] || fail "build-ledgers.sh exited 0 against an unbuildable package -- must fail loudly instead of silently omitting its tests"

if [[ -e "$buildfail_output/tests.json" || -e "$buildfail_output/lint.json" ]]; then
	fail "build-ledgers.sh wrote a ledger despite a build failure: $(ls "$buildfail_output" 2>&1)"
fi

grep -qi "build failure" "$buildfail_dir/build.log" || fail "failure diagnostic does not name the build failure: $(cat "$buildfail_dir/build.log")"
grep -qi "pkg/cart" "$buildfail_dir/build.log" || fail "failure diagnostic does not name the unbuildable package: $(cat "$buildfail_dir/build.log")"

echo "TC-007: build-failure regression correctly rejected (exit $buildfail_exit, no ledger written) -- closes TD-075"

# --- Regression: package that BUILDS but dies at RUNTIME (UAT round 2) -----
# UAT-003 (T-E40-F01-008 rejection): a package that builds fine and then
# dies at runtime -- TestMain calling os.Exit(1), a panic outside a test
# body, an init() crash -- emits only a package-level Action:fail summary
# from `go test -json`, with no "Test" field and no "FailedBuild" marker.
# The pre-fix builder silently omitted that package's tests from tests.json
# and exited 0; a later diff-ledgers.sh comparison could not distinguish
# that omission from "tests legitimately removed". This section reproduces
# it against a real checkout: pkg/cart's own tests are replaced with a
# TestMain that exits 1 before any test in the package can run.
echo "TC-007: regression - go test -json runtime package failure (UAT-003)"

runtime_dir="$WORKDIR/runtimefail"
runtime_checkout="$runtime_dir/checkout"
runtime_output="$runtime_dir/ledgers"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$runtime_checkout" >/dev/null

runtime_cart_test_go="$runtime_checkout/pkg/cart/cart_test.go"
[[ -f "$runtime_cart_test_go" ]] || fail "runtime-failure regression: pkg/cart/cart_test.go not found in checkout"
grep -qF 'import "testing"' "$runtime_cart_test_go" || fail "runtime-failure regression: pkg/cart/cart_test.go import line did not match the expected shape -- mutation target moved"
python3 - "$runtime_cart_test_go" <<'PYEOF'
import sys

path = sys.argv[1]
with open(path) as f:
    content = f.read()
content = content.replace('import "testing"', 'import (\n\t"os"\n\t"testing"\n)')
content += '\nfunc TestMain(m *testing.M) {\n\tos.Exit(1)\n}\n'
with open(path, "w") as f:
    f.write(content)
PYEOF

set +e
"$BUILD_LEDGERS_SCRIPT" "$runtime_checkout" "$runtime_output" >"$runtime_dir/build.log" 2>&1
runtime_exit=$?
set -e

[[ "$runtime_exit" -ne 0 ]] || fail "build-ledgers.sh exited 0 against a package that died at runtime -- must fail loudly instead of silently omitting its tests"

if [[ -e "$runtime_output/tests.json" || -e "$runtime_output/lint.json" ]]; then
	fail "build-ledgers.sh wrote a ledger despite an unexplained package-level runtime failure: $(ls "$runtime_output" 2>&1)"
fi

grep -qi "pkg/cart" "$runtime_dir/build.log" || fail "failure diagnostic does not name the runtime-failed package: $(cat "$runtime_dir/build.log")"
grep -qi "no per-test failure" "$runtime_dir/build.log" || fail "failure diagnostic does not distinguish this from a build failure: $(cat "$runtime_dir/build.log")"

echo "TC-007: runtime-failure regression correctly rejected (exit $runtime_exit, no ledger written)"

echo "TC-007: PASS"
