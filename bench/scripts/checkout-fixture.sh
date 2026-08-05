#!/usr/bin/env bash
# checkout-fixture.sh <base_sha> <dest_dir>
#
# Clones the bench/fixture-repo submodule locally and checks out <base_sha>
# into the caller-supplied <dest_dir>. Shared by admit.sh (T-E40-F01-006),
# build-ledgers.sh (T-E40-F01-008), and later E40-F02; the <base_sha>
# <dest_dir> interface must not change once those depend on it.
#
# Every checkout this script produces is verified clean before control
# returns to the caller: the call to verify-clean-checkout.sh below is this
# script's own exit path, not an optional extra step, so the held-back-test
# invisibility guarantee (REQ-F-004) reaches every current and future
# caller structurally rather than by convention.
#
# REQ-NF-003: this script never touches the live shark database,
# .sharkconfig.json, or the live repository working tree, and never invokes
# shark project-initialisation commands -- all work happens inside
# <dest_dir>.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
	echo "usage: checkout-fixture.sh <base_sha> <dest_dir>" >&2
	exit 2
}

[[ $# -eq 2 ]] || usage
base_sha="$1"
dest_dir="$2"

fixture_submodule="$BENCH_DIR/fixture-repo"
corpus_yaml="$BENCH_DIR/corpus/corpus.yaml"

[[ -e "$fixture_submodule/.git" ]] || {
	echo "checkout-fixture: bench/fixture-repo submodule not initialized; run 'git submodule update --init'" >&2
	exit 1
}
[[ -f "$corpus_yaml" ]] || {
	echo "checkout-fixture: corpus yaml not found: $corpus_yaml" >&2
	exit 1
}

if [[ -e "$dest_dir" ]]; then
	echo "checkout-fixture: dest_dir already exists: $dest_dir" >&2
	exit 1
fi
mkdir -p "$(dirname "$dest_dir")"

git -c advice.detachedHead=false clone --quiet -- "$fixture_submodule" "$dest_dir"
git -C "$dest_dir" -c advice.detachedHead=false checkout --quiet "$base_sha"

# Unconditional -- this is the script's exit path, no flag guards it.
"$SCRIPT_DIR/verify-clean-checkout.sh" "$dest_dir" "$corpus_yaml"
