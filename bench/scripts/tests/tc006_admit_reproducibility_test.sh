#!/usr/bin/env bash
# TC-006 (test-plan.md AC test matrix; T-E40-F01-007 task spec Test Cases).
#
# Exercises AC-005 / AC-T1 (REQ-F-005, REQ-F-012): `admit.sh
# bench/corpus/corpus.yaml` over the full candidate set (the admitted set,
# the three committed negative candidates, and the two TC-005 transient
# candidates) invoked twice, each time against independently provisioned
# checkout-fixture.sh temp checkouts, produces byte-identical verdict
# output both times.
#
# "Independently provisioned" (Determinism boundary, test-plan.md) means:
# no state is shared between round 1 and round 2 -- each round rebuilds its
# own transient (c)/(e) candidate patches from a fresh scratch checkout, and
# admit.sh itself provisions a brand-new checkout-fixture.sh temp checkout
# per candidate it evaluates (see admit.sh's evaluate()). Reusing one
# checkout or one transient-patch file across rounds would let a
# memoizing/caching bug in admit.sh pass this test by accident.
#
# Caller-Path Contract (test-plan.md TC-006): drives the real admit.sh
# entrypoint, which itself shells out to real `git apply` and real
# `go test` inside fresh checkout-fixture.sh checkouts each round -- no
# subprocess result is stubbed, no verdict is memoized or re-read from a
# prior invocation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

REUSE_ITEM="inventory-reserve-boundary"

fail() {
	echo "TC-006 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
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

reuse_patch_abs="$(python3 - "$CORPUS_YAML" "$REUSE_ITEM" <<'PYEOF'
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
reuse_patch_abs="$BENCH_DIR/corpus/$reuse_patch_abs"
[[ -f "$reuse_patch_abs" ]] || fail "reference patch for $REUSE_ITEM not found: $reuse_patch_abs"

# build_transient_c <round_dir> -> prints path to a truncated (malformed)
# copy of $REUSE_ITEM's reference.patch -- rejection branch (c),
# patch-apply-failure. Rebuilt fresh per round; no filesystem state shared
# between rounds.
build_transient_c() {
	local round_dir="$1"
	local out="$round_dir/branch-c-truncated.patch"
	head -c 60 "$reuse_patch_abs" >"$out"
	echo "$out"
}

# build_transient_e <round_dir> -> provisions its own fresh
# checkout-fixture.sh checkout, applies $REUSE_ITEM's real reference.patch,
# then mutates pkg/pricing/pricing.go's TaxAmount body by one line so an
# unrelated P2P test flips to failing post-patch -- rejection branch (e),
# P2P-red-post-patch. Prints the combined patch path.
build_transient_e() {
	local round_dir="$1"
	local scratch_dir="$round_dir/branch-e-scratch"
	"$CHECKOUT_SCRIPT" "$BASE_SHA" "$scratch_dir" >/dev/null

	git -C "$scratch_dir" apply "$reuse_patch_abs" || fail "branch (e): could not apply $REUSE_ITEM's own reference.patch to a fresh checkout"

	local pricing_go="$scratch_dir/pkg/pricing/pricing.go"
	[[ -f "$pricing_go" ]] || fail "branch (e): pkg/pricing/pricing.go not found in scratch checkout"
	grep -qF 'return subtotalCents * rateBasisPoints / 10000' "$pricing_go" || fail "branch (e): TaxAmount body did not match the expected source line -- mutation target moved"
	sed -i 's#return subtotalCents \* rateBasisPoints / 10000#return subtotalCents * rateBasisPoints / 1000#' "$pricing_go"

	local combined="$round_dir/branch-e-combined.patch"
	git -C "$scratch_dir" diff >"$combined"
	[[ -s "$combined" ]] || fail "branch (e): combined patch is empty"
	echo "$combined"
}

# run_round <round_dir> -> runs the full candidate set (admitted-set full
# run + the three committed negatives + the two transient candidates) once,
# writing one file per candidate group into round_dir. Each call provisions
# entirely fresh checkouts -- admit.sh creates a new checkout-fixture.sh
# temp checkout per item internally (see admit.sh evaluate()), and the two
# transient candidates are rebuilt from a fresh scratch checkout by
# build_transient_e above.
run_round() {
	local round_dir="$1"
	mkdir -p "$round_dir"

	set +e
	"$ADMIT_SCRIPT" "$CORPUS_YAML" >"$round_dir/full.jsonl" 2>"$round_dir/full.err"
	echo $? >"$round_dir/full.exit"
	set -e

	local neg_id
	for neg_id in cart-new-cart-zero-subtotal cart-add-item-rejects-negative-quantity validate-sku-uppercase-only; do
		set +e
		"$ADMIT_SCRIPT" "$CORPUS_YAML" --item "$neg_id" >"$round_dir/neg-${neg_id}.json" 2>"$round_dir/neg-${neg_id}.err"
		echo $? >"$round_dir/neg-${neg_id}.exit"
		set -e
	done

	local patch_c
	patch_c="$(build_transient_c "$round_dir")"
	set +e
	"$ADMIT_SCRIPT" "$CORPUS_YAML" --item "$REUSE_ITEM" --patch "$patch_c" >"$round_dir/transient-c.json" 2>"$round_dir/transient-c.err"
	echo $? >"$round_dir/transient-c.exit"
	set -e

	local patch_e
	patch_e="$(build_transient_e "$round_dir")"
	set +e
	"$ADMIT_SCRIPT" "$CORPUS_YAML" --item "$REUSE_ITEM" --patch "$patch_e" >"$round_dir/transient-e.json" 2>"$round_dir/transient-e.err"
	echo $? >"$round_dir/transient-e.exit"
	set -e
}

echo "TC-006: round 1 - independently provisioned checkouts"
run_round "$WORKDIR/round1"

echo "TC-006: round 2 - independently provisioned checkouts"
run_round "$WORKDIR/round2"

echo "TC-006: comparing round 1 vs round 2 for byte-identical output"

compare_file() {
	local label="$1" rel="$2"
	local f1="$WORKDIR/round1/$rel" f2="$WORKDIR/round2/$rel"
	[[ -f "$f1" ]] || fail "$label: round 1 output missing: $f1"
	[[ -f "$f2" ]] || fail "$label: round 2 output missing: $f2"
	if ! cmp -s "$f1" "$f2"; then
		fail "$label: round 1 and round 2 output differ (not byte-identical): $(diff "$f1" "$f2" | head -20)"
	fi
	echo "TC-006: $label byte-identical across both rounds"
}

compare_file "full admitted-set run (stdout)" "full.jsonl"
compare_file "full admitted-set run (exit code)" "full.exit"
[[ "$(cat "$WORKDIR/round1/full.exit")" -eq 0 ]] || fail "full admitted-set run did not exit 0 in round 1"

for neg_id in cart-new-cart-zero-subtotal cart-add-item-rejects-negative-quantity validate-sku-uppercase-only; do
	compare_file "negative candidate $neg_id (stdout)" "neg-${neg_id}.json"
	compare_file "negative candidate $neg_id (exit code)" "neg-${neg_id}.exit"
	[[ "$(cat "$WORKDIR/round1/neg-${neg_id}.exit")" -eq 1 ]] || fail "negative candidate $neg_id did not exit 1 (rejected) in round 1"
done

compare_file "transient branch (c) (stdout)" "transient-c.json"
compare_file "transient branch (c) (exit code)" "transient-c.exit"
[[ "$(cat "$WORKDIR/round1/transient-c.exit")" -eq 1 ]] || fail "transient branch (c) did not exit 1 (rejected) in round 1"

compare_file "transient branch (e) (stdout)" "transient-e.json"
compare_file "transient branch (e) (exit code)" "transient-e.exit"
[[ "$(cat "$WORKDIR/round1/transient-e.exit")" -eq 1 ]] || fail "transient branch (e) did not exit 1 (rejected) in round 1"

echo "TC-006: PASS"
