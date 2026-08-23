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
	python3 - "$consumer" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text()
if "subprocess.run" not in text:
    raise SystemExit(f"{path}: no subprocess call to the shared digest helper")
if path.name in {"retain_pair", "verify_pair_retention"}:
    required = 'with_name("digest_path")'
else:
    required = 'os.environ["F10_DIGEST_HELPER"]'
if required not in text:
    raise SystemExit(f"{path}: digest helper is not the executable authority")
PY
done

echo "TC-093: shared digest authority and symlink refusal pass"
