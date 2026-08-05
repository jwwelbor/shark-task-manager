#!/usr/bin/env bash
# TC-003 (test-plan.md AC test matrix; T-E40-F01-005 task spec Test Cases).
#
# Exercises AC-001 / AC-T1-T3:
#   - checkout-fixture.sh + verify-clean-checkout.sh together report CLEAN
#     on a real checkout of the fixture at the corpus base SHA.
#   - checkout-fixture.sh itself invokes verify-clean-checkout.sh on its own
#     exit path (not merely that the verifier script exists on disk).
#   - a deliberately-seeded copy of one held-back F2P file into a checkout
#     is detected and named on a direct re-invocation of
#     verify-clean-checkout.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
VERIFY_SCRIPT="$SCRIPTS_DIR/verify-clean-checkout.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-003 FAIL: $1" >&2
	exit 1
}

[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -x "$VERIFY_SCRIPT" ]] || fail "verify-clean-checkout.sh missing or not executable"
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

echo "TC-003: part 1 - real checkout, real verify, expect CLEAN"
dest_dir="$WORKDIR/checkout-clean"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$dest_dir" || fail "checkout-fixture.sh exited non-zero on a clean checkout"
[[ -d "$dest_dir" ]] || fail "checkout-fixture.sh did not create $dest_dir"
[[ -f "$dest_dir/go.mod" ]] || fail "checkout does not look like the fixture (no go.mod)"

verify_out="$("$VERIFY_SCRIPT" "$dest_dir" "$CORPUS_YAML")" || fail "direct verify-clean-checkout.sh invocation failed on a clean checkout: $verify_out"
[[ "$verify_out" == "CLEAN" ]] || fail "expected CLEAN, got: $verify_out"

echo "TC-003: part 2 - checkout-fixture.sh invokes the verifier itself"
inst_root="$WORKDIR/instrumented"
mkdir -p "$inst_root/bench"
ln -s "$BENCH_DIR/fixture-repo" "$inst_root/bench/fixture-repo"
ln -s "$BENCH_DIR/corpus" "$inst_root/bench/corpus"
cp -r "$SCRIPTS_DIR" "$inst_root/bench/scripts"

marker="$WORKDIR/verify-invoked-marker"
rm -f "$marker"
cat >"$inst_root/bench/scripts/verify-clean-checkout.sh" <<EOF
#!/usr/bin/env bash
touch "$marker"
echo "CLEAN"
exit 0
EOF
chmod +x "$inst_root/bench/scripts/verify-clean-checkout.sh"

dest_dir2="$WORKDIR/checkout-instrumented"
"$inst_root/bench/scripts/checkout-fixture.sh" "$BASE_SHA" "$dest_dir2" || fail "instrumented checkout-fixture.sh exited non-zero"
[[ -f "$marker" ]] || fail "checkout-fixture.sh did not invoke verify-clean-checkout.sh on its own exit path"

echo "TC-003: part 3 - seeded leak is detected and named on direct re-invocation"
dest_dir3="$WORKDIR/checkout-leak"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$dest_dir3" || fail "checkout-fixture.sh exited non-zero provisioning the leak checkout"

leak_rel_path="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["items"][0]["f2p"]["paths"][0])
PYEOF
)"
leak_src="$BENCH_DIR/corpus/$leak_rel_path"
[[ -f "$leak_src" ]] || fail "expected held-back source file missing: $leak_src"
leak_basename="$(basename "$leak_rel_path")"
cp "$leak_src" "$dest_dir3/$leak_basename"

if leak_out="$("$VERIFY_SCRIPT" "$dest_dir3" "$CORPUS_YAML" 2>&1)"; then
	fail "verify-clean-checkout.sh did not detect the seeded leak; output: $leak_out"
fi
echo "$leak_out" | grep -qF -- "$leak_basename" || fail "leak output did not name the leaked file ($leak_basename): $leak_out"

echo "TC-003: PASS"
