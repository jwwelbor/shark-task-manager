#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
fixture="$REPO_ROOT/bench/scripts/testdata/evaluation/ineligible.jsonl"
[[ -x "$VERIFIER" ]] || { echo "TC-074: verifier missing" >&2; exit 1; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
if "$VERIFIER" "$fixture" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/verdict" 2>"$tmp/error"; then exit 1; fi
grep -q 'aggregate_eligible' "$tmp/error"
grep -q 'failed_oracle' "$tmp/error"
[[ -s "$fixture" ]]
echo "TC-074: invalid record retained and excluded from eligibility"
