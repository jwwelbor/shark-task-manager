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
