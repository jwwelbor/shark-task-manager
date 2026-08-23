#!/usr/bin/env bash
# TC-093: all retention digest consumers use the schema-owned helper.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
HELPER="$BENCH_DIR/lib/digest_path"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/tree/sub"
printf 'alpha' >"$tmp/tree/a.txt"
printf 'beta' >"$tmp/tree/sub/b.txt"
digest_one="$($HELPER "$tmp/tree")"
digest_two="$($HELPER "$tmp/tree")"
[[ -n "$digest_one" && "$digest_one" == "$digest_two" ]]

ln -s "$tmp/tree/a.txt" "$tmp/tree/link.txt"
if "$HELPER" "$tmp/tree" >/dev/null 2>&1; then
	echo "TC-093: symlinked tree unexpectedly produced a digest" >&2
	exit 1
fi

for consumer in "$BENCH_DIR/lib/retain_pair" "$BENCH_DIR/lib/verify_pair_retention" "$BENCH_DIR/pilot-ledger.sh" "$BENCH_DIR/verify-retention-root.sh"; do
	grep -q 'F10_DIGEST_HELPER\|digest_path' "$consumer"
done

echo "TC-093: shared digest authority and symlink refusal pass"
