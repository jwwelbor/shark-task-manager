#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"
fixture="$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl"
[[ -x "$VERIFIER" ]] || { echo "TC-073: verifier missing" >&2; exit 1; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$fixture" >"$tmp/eligible.jsonl"
LC_ALL=C "$VERIFIER" "$tmp/eligible.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/one"
LC_ALL=C "$VERIFIER" "$tmp/eligible.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/two"
cmp "$tmp/one" "$tmp/two"
cp "$tmp/eligible.jsonl" "$tmp/duplicate.jsonl"; cat "$tmp/eligible.jsonl" >> "$tmp/duplicate.jsonl"
if "$VERIFIER" "$tmp/duplicate.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >/dev/null 2>"$tmp/error"; then exit 1; fi
grep -q 'duplicate' "$tmp/error"
# TC-073 caller-path proof: replay retained I-05/I-07 files through the real
# evaluator twice; no provider, scenario rerun, or live-tree input is needed.
mkdir -p "$tmp/i05"
echo '{"roots":{"agent_fixture_checkout":"/does/not/exist"},"stages":[]}' >"$tmp/i05/bundle.json"
echo '{"identity":{"run_id":"tc073","scenario_id":"py-bug-due-date-boundary"},"dispatches":[],"stages":[],"outcome":{"terminal":"complete"}}' >"$tmp/i07.jsonl"
scenario="$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml"
for n in one two; do "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/i07.jsonl" --scenario "$scenario" --output "$tmp/eval-$n.jsonl" >/dev/null 2>"$tmp/eval-$n.err" || true; done
cmp "$tmp/eval-one.jsonl" "$tmp/eval-two.jsonl"
grep -q 'evaluation_invalid' "$tmp/eval-one.err"
echo "TC-073: deterministic offline verification and duplicate rejection pass"
