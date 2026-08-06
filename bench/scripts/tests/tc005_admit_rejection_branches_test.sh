#!/usr/bin/env bash
# TC-005 (test-plan.md AC test matrix; T-E40-F01-006 task spec Test Cases).
#
# Exercises AC-004 / AC-T2-T3: `admit.sh <corpus_yaml> --item <id>` against
# all five admission-gate rejection branches --
#   (a) F2P-green-at-base       -- committed negative cart-new-cart-zero-subtotal
#   (b) P2P-red-at-base         -- committed negative cart-add-item-rejects-negative-quantity
#   (c) patch-apply-failure     -- transient: an admitted item's reference.patch truncated mid-hunk
#   (d) F2P-still-red-post-patch -- committed negative validate-sku-uppercase-only
#   (e) P2P-red-post-patch      -- transient: an admitted item's reference.patch plus a one-line
#                                   mutation elsewhere that flips an unrelated P2P test to failing
#
# and AC-T3: none of the three committed negatives ever appears in a full
# (no `--item`) admit.sh run's admitted-item output.
#
# This script builds the two transient (c)/(e) candidates itself (task spec
# Scope) by copying bench/corpus/items/inventory-reserve-boundary/reference.patch
# and corrupting/extending the copy -- it never adds a committed corpus.yaml
# entry for them (REQ-F-007 excuses branches (c)/(e) from a committed
# fixture, not from test coverage).
#
# Caller-Path Contract (test-plan.md TC-005): same as TC-004 -- the real
# `admit.sh` entrypoint driving real `git apply` and real `go test`. No
# check's subprocess result is stubbed, no per-item boolean is hardcoded,
# and no negative/transient candidate is special-cased to force its
# expected rejection message -- the same `admit.sh` code path that admits
# the real corpus is what rejects these.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-005 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

assert_rejected() {
	local label="$1" verdict_file="$2" expected_check="$3"
	python3 - "$label" "$verdict_file" "$expected_check" <<'PYEOF'
import json
import sys

label, verdict_path, expected_check = sys.argv[1:4]

with open(verdict_path) as f:
    line = f.read().strip()

try:
    verdict = json.loads(line)
except json.JSONDecodeError as exc:
    sys.exit(f"TC-005 FAIL: {label}: admit.sh did not print a single JSON verdict line: {line!r} ({exc})")
if verdict["status"] != "rejected":
    sys.exit(f"TC-005 FAIL: {label}: expected status 'rejected', got {verdict['status']!r}: {line}")
if verdict["failing_check"] != expected_check:
    sys.exit(
        f"TC-005 FAIL: {label}: expected failing_check {expected_check!r}, "
        f"got {verdict['failing_check']!r}: {line}"
    )
print(f"TC-005: {label} rejected with {expected_check!r} as expected")
PYEOF
}

run_item() {
	# run_item <label> <item_id> [--patch <path>]
	local label="$1" item_id="$2"
	shift 2
	local out_file="$WORKDIR/${item_id}-$(echo "$label" | tr -c 'a-zA-Z0-9' '_').json"
	set +e
	"$ADMIT_SCRIPT" "$CORPUS_YAML" --item "$item_id" "$@" >"$out_file"
	local code=$?
	set -e
	[[ "$code" -eq 1 ]] || fail "$label: admit.sh exited $code (expected 1 for a rejected candidate); output: $(cat "$out_file")"
	echo "$out_file"
}

echo "TC-005: part 1 - three committed negative candidates (branches a, b, d)"

out_a="$(run_item "branch (a) cart-new-cart-zero-subtotal" cart-new-cart-zero-subtotal)"
assert_rejected "branch (a)" "$out_a" "F2P-green-at-base"

out_b="$(run_item "branch (b) cart-add-item-rejects-negative-quantity" cart-add-item-rejects-negative-quantity)"
assert_rejected "branch (b)" "$out_b" "P2P-red-at-base"

out_d="$(run_item "branch (d) validate-sku-uppercase-only" validate-sku-uppercase-only)"
assert_rejected "branch (d)" "$out_d" "F2P-still-red-post-patch"

echo "TC-005: part 2 - transient branch (c): truncated reference.patch -> git apply failure"

REUSE_ITEM="inventory-reserve-boundary"
reuse_patch="$(python3 - "$CORPUS_YAML" "$REUSE_ITEM" <<'PYEOF'
import sys
import yaml

corpus_yaml_path, item_id = sys.argv[1:3]
with open(corpus_yaml_path) as f:
    data = yaml.safe_load(f)
for item in data["items"]:
    if item["id"] == item_id:
        print(item["reference_patch_path"])
        break
else:
    sys.exit(f"item not found: {item_id}")
PYEOF
)"
reuse_patch_abs="$BENCH_DIR/corpus/$reuse_patch"
[[ -f "$reuse_patch_abs" ]] || fail "reference patch for $REUSE_ITEM not found: $reuse_patch_abs"

truncated_patch="$WORKDIR/branch-c-truncated.patch"
head -c 60 "$reuse_patch_abs" >"$truncated_patch"

out_c="$(run_item "branch (c) truncated $REUSE_ITEM patch" "$REUSE_ITEM" --patch "$truncated_patch")"
assert_rejected "branch (c)" "$out_c" "patch-apply-failure"

echo "TC-005: part 3 - transient branch (e): reference.patch + one-line P2P-flipping mutation"

BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$BASE_SHA" ]] || fail "could not derive base_sha from $CORPUS_YAML"

scratch_dir="$WORKDIR/branch-e-scratch"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$scratch_dir" >/dev/null

git -C "$scratch_dir" apply "$reuse_patch_abs" || fail "branch (e): could not apply $REUSE_ITEM's own reference.patch to a fresh checkout"

pricing_go="$scratch_dir/pkg/pricing/pricing.go"
[[ -f "$pricing_go" ]] || fail "branch (e): pkg/pricing/pricing.go not found in scratch checkout"
grep -qF 'return subtotalCents * rateBasisPoints / 10000' "$pricing_go" || fail "branch (e): TaxAmount body did not match the expected source line -- mutation target moved"
sed -i 's#return subtotalCents \* rateBasisPoints / 10000#return subtotalCents * rateBasisPoints / 1000#' "$pricing_go"

combined_patch="$WORKDIR/branch-e-combined.patch"
git -C "$scratch_dir" diff >"$combined_patch"
[[ -s "$combined_patch" ]] || fail "branch (e): combined patch is empty"

out_e="$(run_item "branch (e) $REUSE_ITEM patch + TaxAmount mutation" "$REUSE_ITEM" --patch "$combined_patch")"
assert_rejected "branch (e)" "$out_e" "P2P-red-post-patch"

echo "TC-005: part 3b - branch (e) variant: runtime package death (UAT round 2, UAT-002)"

# UAT-002 (T-E40-F01-006 rejection): a package that BUILDS fine and then
# dies at runtime -- TestMain calling os.Exit(1), a panic outside a test
# body, an init() crash -- emits only a package-level Action:fail summary
# from `go test -json`, with no "Test" field and no "FailedBuild" marker
# for any of its tests. The P2P check used to derive pass/fail from
# per-test events only, so a package that died wholesale contributed zero
# entries to the per-test map and read as "no failing tests -> green",
# letting a reference patch silently break an unrelated package and still
# admit the candidate. This reproduces that exact scenario: $REUSE_ITEM's
# own valid reference.patch, plus a runtime TestMain failure injected into
# pkg/cart -- a package $REUSE_ITEM's patch never touches -- must still
# reject with P2P-red-post-patch, and admit.sh's verdict must name
# pkg/cart in unexplained_failed_packages.
runtime_scratch_dir="$WORKDIR/branch-e2-scratch"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$runtime_scratch_dir" >/dev/null

git -C "$runtime_scratch_dir" apply "$reuse_patch_abs" || fail "branch (e2): could not apply $REUSE_ITEM's own reference.patch to a fresh checkout"

cart_test_go="$runtime_scratch_dir/pkg/cart/cart_test.go"
[[ -f "$cart_test_go" ]] || fail "branch (e2): pkg/cart/cart_test.go not found in scratch checkout"
grep -qF 'import "testing"' "$cart_test_go" || fail "branch (e2): pkg/cart/cart_test.go import line did not match the expected shape -- mutation target moved"
python3 - "$cart_test_go" <<'PYEOF'
import sys

path = sys.argv[1]
with open(path) as f:
    content = f.read()
content = content.replace('import "testing"', 'import (\n\t"os"\n\t"testing"\n)')
content += '\nfunc TestMain(m *testing.M) {\n\tos.Exit(1)\n}\n'
with open(path, "w") as f:
    f.write(content)
PYEOF

runtime_patch="$WORKDIR/branch-e2-combined.patch"
git -C "$runtime_scratch_dir" diff >"$runtime_patch"
[[ -s "$runtime_patch" ]] || fail "branch (e2): combined patch is empty"

out_e2="$(run_item "branch (e2) $REUSE_ITEM patch + pkg/cart runtime TestMain failure" "$REUSE_ITEM" --patch "$runtime_patch")"
assert_rejected "branch (e2)" "$out_e2" "P2P-red-post-patch"

echo "TC-005: part 3c - branch (e) variant: masked runtime failure behind the excluded probe (UAT round 3, UAT-005)"

# UAT-005 (T-E40-F01-006 rejection round 3): round 2's fix explained a
# package-level "fail" by checking whether ANY per-test "fail" was also
# recorded for that package -- but pkg/inventory's own intentional
# TestStock_PermanentlyFailingRegressionProbe (excluded from the default
# p2p_set) always supplies exactly such a per-test fail, so it can MASK
# an independent runtime failure riding on the same package's process
# exit. This is the UAT report's own reproduction: a real, valid
# reference patch for an item whose F2P lives OUTSIDE pkg/inventory
# (isolating this to the P2P check, not the F2P check -- see the
# f2p_packages scoping note in admit.sh), plus TestMain(m *testing.M) {
# m.Run(); os.Exit(1) } injected into pkg/inventory, the package that
# already contains the excluded probe. Must still reject with
# P2P-red-post-patch, naming pkg/inventory in
# unexplained_failed_packages -- the probe's own real failure must not
# provide cover for it.
MASK_ITEM="validate-sku-max-length"
mask_patch="$(python3 - "$CORPUS_YAML" "$MASK_ITEM" <<'PYEOF'
import sys
import yaml

corpus_yaml_path, item_id = sys.argv[1:3]
with open(corpus_yaml_path) as f:
    data = yaml.safe_load(f)
for item in data["items"]:
    if item["id"] == item_id:
        print(item["reference_patch_path"])
        break
else:
    sys.exit(f"item not found: {item_id}")
PYEOF
)"
mask_patch_abs="$BENCH_DIR/corpus/$mask_patch"
[[ -f "$mask_patch_abs" ]] || fail "reference patch for $MASK_ITEM not found: $mask_patch_abs"

mask_scratch_dir="$WORKDIR/branch-e3-scratch"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$mask_scratch_dir" >/dev/null

git -C "$mask_scratch_dir" apply "$mask_patch_abs" || fail "branch (e3): could not apply $MASK_ITEM's own reference.patch to a fresh checkout"

inventory_test_go="$mask_scratch_dir/pkg/inventory/inventory_test.go"
[[ -f "$inventory_test_go" ]] || fail "branch (e3): pkg/inventory/inventory_test.go not found in scratch checkout"
grep -qF 'import "testing"' "$inventory_test_go" || fail "branch (e3): pkg/inventory/inventory_test.go import line did not match the expected shape -- mutation target moved"
python3 - "$inventory_test_go" <<'PYEOF'
import sys

path = sys.argv[1]
with open(path) as f:
    content = f.read()
content = content.replace('import "testing"', 'import (\n\t"os"\n\t"testing"\n)')
content += '\nfunc TestMain(m *testing.M) {\n\tcode := m.Run()\n\t_ = code\n\tos.Exit(1)\n}\n'
with open(path, "w") as f:
    f.write(content)
PYEOF

mask_patch_combined="$WORKDIR/branch-e3-combined.patch"
git -C "$mask_scratch_dir" diff >"$mask_patch_combined"
[[ -s "$mask_patch_combined" ]] || fail "branch (e3): combined patch is empty"

out_e3="$(run_item "branch (e3) $MASK_ITEM patch + pkg/inventory TestMain masked behind the excluded probe" "$MASK_ITEM" --patch "$mask_patch_combined")"
assert_rejected "branch (e3)" "$out_e3" "P2P-red-post-patch"

python3 - "$out_e3" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    verdict = json.loads(f.read().strip())

unexplained = verdict.get("unexplained_failed_packages") or []
if not any(pkg.endswith("/pkg/inventory") for pkg in unexplained):
    sys.exit(
        "TC-005 FAIL: branch (e3): verdict's unexplained_failed_packages does not "
        f"name pkg/inventory (the probe's own package): {unexplained}"
    )
print(f"TC-005: branch (e3) unexplained_failed_packages names pkg/inventory despite the excluded probe's own failure: {unexplained}")
PYEOF

echo "TC-005: part 3d - branch variant: masked runtime failure in the SAME package as the F2P test (UAT-005's literal reproduction)"

# The UAT report's own adversarial run used inventory-reserve-boundary
# specifically (F2P test in pkg/inventory, the same package as the
# probe and the injected TestMain), and asserted only that admit.sh must
# not admit -- not a specific check name. Under this fix, the F2P check
# is scoped to exactly the F2P test's own package (see admit.sh's
# f2p_packages comment), so it observes the same poisoned process exit
# and correctly rejects as F2P-still-red-post-patch rather than
# P2P-red-post-patch; either is a correct, safe rejection, but the
# candidate must never be admitted.
same_pkg_scratch_dir="$WORKDIR/branch-e4-scratch"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$same_pkg_scratch_dir" >/dev/null

git -C "$same_pkg_scratch_dir" apply "$reuse_patch_abs" || fail "branch (e4): could not apply $REUSE_ITEM's own reference.patch to a fresh checkout"

same_pkg_inventory_test_go="$same_pkg_scratch_dir/pkg/inventory/inventory_test.go"
[[ -f "$same_pkg_inventory_test_go" ]] || fail "branch (e4): pkg/inventory/inventory_test.go not found in scratch checkout"
grep -qF 'import "testing"' "$same_pkg_inventory_test_go" || fail "branch (e4): pkg/inventory/inventory_test.go import line did not match the expected shape -- mutation target moved"
python3 - "$same_pkg_inventory_test_go" <<'PYEOF'
import sys

path = sys.argv[1]
with open(path) as f:
    content = f.read()
content = content.replace('import "testing"', 'import (\n\t"os"\n\t"testing"\n)')
content += '\nfunc TestMain(m *testing.M) {\n\tcode := m.Run()\n\t_ = code\n\tos.Exit(1)\n}\n'
with open(path, "w") as f:
    f.write(content)
PYEOF

same_pkg_patch_combined="$WORKDIR/branch-e4-combined.patch"
git -C "$same_pkg_scratch_dir" diff >"$same_pkg_patch_combined"
[[ -s "$same_pkg_patch_combined" ]] || fail "branch (e4): combined patch is empty"

out_e4="$(run_item "branch (e4) $REUSE_ITEM patch + pkg/inventory TestMain (same package as F2P and the probe)" "$REUSE_ITEM" --patch "$same_pkg_patch_combined")"
python3 - "$out_e4" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    verdict = json.loads(f.read().strip())

if verdict["status"] != "rejected":
    sys.exit(
        "TC-005 FAIL: branch (e4): expected status 'rejected' for a patch that adds "
        f"an unconditional TestMain os.Exit(1) to pkg/inventory, got {verdict['status']!r}: {verdict}"
    )
print(f"TC-005: branch (e4) rejected as {verdict['failing_check']!r} (F2P and P2P share pkg/inventory here; either is a correct, safe rejection)")
PYEOF

python3 - "$out_e2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    verdict = json.loads(f.read().strip())

unexplained = verdict.get("unexplained_failed_packages") or []
if not any(pkg.endswith("/pkg/cart") for pkg in unexplained):
    sys.exit(
        "TC-005 FAIL: branch (e2): verdict's unexplained_failed_packages does not "
        f"name pkg/cart: {unexplained}"
    )
print(f"TC-005: branch (e2) unexplained_failed_packages names the runtime-dead package: {unexplained}")
PYEOF

echo "TC-005: part 4 (AC-T3) - no committed negative appears in a full admit.sh run"

full_out="$WORKDIR/full-run.jsonl"
"$ADMIT_SCRIPT" "$CORPUS_YAML" >"$full_out"

# Asserted against the actual admitted-id set derived from corpus.yaml, not
# a substring grep -- a grep for the three known negative ids would stay
# green even if the admitted set were empty or if a future admit.sh change
# renamed the "item_id" key, since it never checks what IS present. Here,
# any mismatch between the run's id set and corpus.yaml's own items[]/
# negative_items[] id sets fails the test, and parsing each line's actual
# "item_id" field (not a raw string match) means a renamed key breaks the
# test loudly via a JSON KeyError rather than passing by coincidence.
python3 - "$CORPUS_YAML" "$full_out" <<'PYEOF'
import json
import sys

import yaml

corpus_yaml_path, full_out_path = sys.argv[1:3]

with open(corpus_yaml_path) as f:
    data = yaml.safe_load(f)

admitted_ids = {item["id"] for item in data["items"]}
negative_ids = {item["id"] for item in data.get("negative_items") or []}

with open(full_out_path) as f:
    lines = [line for line in f.read().splitlines() if line]

run_ids = {json.loads(line)["item_id"] for line in lines}

leaked = run_ids & negative_ids
if leaked:
    sys.exit(f"TC-005 FAIL: AC-T3 violated: negative candidate(s) appeared in a full admit.sh run: {sorted(leaked)}")

if run_ids != admitted_ids:
    sys.exit(
        "TC-005 FAIL: AC-T3: full admit.sh run's id set does not match corpus.yaml's "
        f"items[] id set exactly -- run emitted {sorted(run_ids)}, expected {sorted(admitted_ids)}"
    )

print(f"TC-005: full run emitted exactly the {len(admitted_ids)} items[] ids, none of the {len(negative_ids)} negative_items ids")
PYEOF

echo "TC-005: PASS"
