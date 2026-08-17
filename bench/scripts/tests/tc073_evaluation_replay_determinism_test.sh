#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
fixture="$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl"
[[ -x "$VERIFIER" ]] || { echo "TC-073: verifier missing" >&2; exit 1; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
LC_ALL=C "$VERIFIER" "$fixture" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/one"
LC_ALL=C "$VERIFIER" "$fixture" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/two"
cmp "$tmp/one" "$tmp/two"
cp "$fixture" "$tmp/duplicate.jsonl"; cat "$fixture" >> "$tmp/duplicate.jsonl"
if "$VERIFIER" "$tmp/duplicate.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >/dev/null 2>"$tmp/error"; then exit 1; fi
grep -q 'duplicate' "$tmp/error"
echo "TC-073: deterministic offline verification and duplicate rejection pass"
