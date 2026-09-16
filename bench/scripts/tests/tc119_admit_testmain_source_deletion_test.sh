#!/usr/bin/env bash
# TC-119 / B060: P2P expected test names must be captured before any P2P
# package runs. A TestMain in the writable checkout can otherwise delete its
# own *_test.go files, exit 0 without m.Run, and make a post-execution
# enumeration falsely report that no terminal events were expected.
#
# Caller-Path Contract: run the real admit.sh entrypoint, its checkout
# provisioner, real go test, and testenum. Nothing in check_p2p_green() is
# stubbed; the transient patch behaves like an untrusted package under test.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-119 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# This item's F2P test is in pkg/inventory, leaving pkg/cart as a P2P-only
# package in the default ./... set. The item's genuine reference patch keeps
# F2P green; the added TestMain targets the independent P2P package.
ITEM_ID="inventory-reserve-boundary"
BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1], encoding="utf-8") as stream:
    print(yaml.safe_load(stream)["fixture"]["base_sha"])
PYEOF
)"
REFERENCE_PATCH="$BENCH_DIR/corpus/items/$ITEM_ID/reference.patch"
[[ -f "$REFERENCE_PATCH" ]] || fail "reference patch missing: $REFERENCE_PATCH"

SCRATCH="$WORKDIR/fixture"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$SCRATCH" >/dev/null
git -C "$SCRATCH" apply "$REFERENCE_PATCH" || fail "could not apply $ITEM_ID reference patch"

CART_TEST="$SCRATCH/pkg/cart/cart_test.go"
[[ -f "$CART_TEST" ]] || fail "pkg/cart/cart_test.go not found in fixture checkout"
grep -qF 'import "testing"' "$CART_TEST" || fail "pkg/cart/cart_test.go import shape changed"

python3 - "$CART_TEST" <<'PYEOF'
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as stream:
    content = stream.read()
content = content.replace(
    'import "testing"',
    'import (\n\t"os"\n\t"path/filepath"\n\t"testing"\n)',
    1,
)
content += '''
func TestMain(m *testing.M) {
	paths, err := filepath.Glob("*_test.go")
	if err != nil {
		os.Exit(2)
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			os.Exit(2)
		}
	}
	os.Exit(0)
}
'''
with open(path, "w", encoding="utf-8") as stream:
    stream.write(content)
PYEOF

PATCH="$WORKDIR/self-deleting-testmain.patch"
git -C "$SCRATCH" diff >"$PATCH"
[[ -s "$PATCH" ]] || fail "self-deleting TestMain patch is empty"

OUT_FILE="$WORKDIR/stdout.json"
ERR_FILE="$WORKDIR/stderr.log"
set +e
"$ADMIT_SCRIPT" "$CORPUS_YAML" --item "$ITEM_ID" --patch "$PATCH" >"$OUT_FILE" 2>"$ERR_FILE"
CODE=$?
set -e

[[ "$CODE" -eq 1 ]] || fail "expected B060 patch to be rejected (exit 1), got $CODE; stdout: $(cat "$OUT_FILE"); stderr: $(cat "$ERR_FILE")"

python3 - "$OUT_FILE" <<'PYEOF'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    verdict = json.loads(stream.read().strip())

if verdict.get("status") != "rejected":
    sys.exit(f"TC-119 FAIL: self-deleting TestMain was admitted: {verdict}")
if verdict.get("failing_check") != "P2P-red-post-patch":
    sys.exit(
        "TC-119 FAIL: expected P2P-red-post-patch, got "
        f"{verdict.get('failing_check')!r}: {verdict}"
    )
problems = verdict.get("unexplained_failed_packages") or []
if not any("/pkg/cart" in problem and "missing terminal event" in problem for problem in problems):
    sys.exit(
        "TC-119 FAIL: rejection did not preserve pre-execution expected tests "
        f"for pkg/cart: {problems}"
    )

print("TC-119: self-deleting TestMain rejected with missing P2P terminal evidence")
PYEOF
