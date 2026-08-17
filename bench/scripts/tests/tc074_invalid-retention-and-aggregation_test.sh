#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
AGGREGATOR="$REPO_ROOT/bench/scripts/aggregate-runs.sh"
fixture="$REPO_ROOT/bench/scripts/testdata/evaluation/ineligible.jsonl"
[[ -x "$VERIFIER" ]] || { echo "TC-074: verifier missing" >&2; exit 1; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$fixture" >"$tmp/ineligible.jsonl"
if "$VERIFIER" "$tmp/ineligible.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/verdict" 2>"$tmp/error"; then exit 1; fi
grep -q 'aggregate_eligible' "$tmp/error"
grep -q 'failed_oracle' "$tmp/error"
[[ -s "$fixture" ]]

# TC-074 caller-path proof: invoke the real aggregator on a retained invalid
# record. It must retain inventory and reject the unusable aggregate rather
# than silently dropping the record.
mkdir -p "$tmp/aggregate/item/variant/rep-1"
cp "$tmp/ineligible.jsonl" "$tmp/aggregate/item/variant/rep-1/record.jsonl"
if "$AGGREGATOR" --root "$tmp/aggregate" >"$tmp/aggregate.json" 2>"$tmp/aggregate.err"; then
  echo "TC-074: invalid retained record unexpectedly aggregated" >&2
  exit 1
fi
grep -Eq 'anomaly|unusable|invalid|record' "$tmp/aggregate.err" "$tmp/aggregate.json"

# TC-074 counter-factual: every nested schema-required eligibility field must
# be enforced, not only aggregate_eligible.
python3 - "$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl" "$tmp/missing-publication.jsonl" <<'PY'
import json, sys
record = json.loads(open(sys.argv[1], encoding="utf-8").readline())
record["metrics"] = {"quality": {}, "elapsed_time": {}, "provider_cost": {}, "rework": {}, "artifact_use": {}}
del record["eligibility"]["publication_eligible"]
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    json.dump(record, stream)
    stream.write("\n")
PY
if "$VERIFIER" "$tmp/missing-publication.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/missing-out" 2>"$tmp/missing-error"; then exit 1; fi
grep -q '/eligibility/publication_eligible' "$tmp/missing-error"

echo "TC-074: invalid record retained and excluded from eligibility"
