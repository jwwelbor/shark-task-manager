#!/usr/bin/env bash
# TC-068: the held-back oracle must refuse pre-terminal access before invoking
# an adapter and must use the existing I-05 access broker after terminality.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ORACLE="$REPO_ROOT/bench/scripts/run-heldback-oracle.sh"

[[ -x "$ORACLE" ]] || { echo "TC-068: oracle missing or not executable" >&2; exit 1; }
grep -q 'adapter_calls' "$ORACLE" || { echo "TC-068: oracle must retain adapter call accounting" >&2; exit 1; }
grep -q 'restore_checkout' "$ORACLE" || { echo "TC-068: oracle must restore checkout on post-grant failure" >&2; exit 1; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bundle" "$tmp/checkout"
echo '{"terminal_status":{"reached":false}}' > "$tmp/bundle/bundle.json"
echo '{"identity":{"run_id":"tc068"},"outcome":{"terminal":"in_progress"}}' > "$tmp/i07.jsonl"
if "$ORACLE" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --i07 "$tmp/i07.jsonl" --stage-bundle "$tmp/bundle" --checkout "$tmp/checkout" --output "$tmp/oracle.json" >/dev/null 2>"$tmp/stderr"; then echo "TC-068: pre-terminal oracle unexpectedly passed" >&2; exit 1; fi
python3 - "$tmp/oracle.json" "$tmp/checkout" <<'PY'
import json, os, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["observed_result"] == "not_run"
assert record["adapter_calls"] == 0
assert record["invalidity_reasons"][0]["code"] == "pre_terminal"
assert os.listdir(sys.argv[2]) == []
PY
echo "TC-068: pre-terminal access is refused before adapter or filesystem use"
