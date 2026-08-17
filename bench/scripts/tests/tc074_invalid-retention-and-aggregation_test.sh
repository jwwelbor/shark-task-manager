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

# TC-074 counter-factual: every nested schema-required eligibility field must
# be enforced, not only aggregate_eligible.
python3 - "$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl" "$tmp/missing-publication.jsonl" <<'PY'
import json, sys
record = json.loads(open(sys.argv[1], encoding="utf-8").readline())
del record["eligibility"]["publication_eligible"]
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    json.dump(record, stream)
    stream.write("\n")
PY
if "$VERIFIER" "$tmp/missing-publication.jsonl" --schema "$REPO_ROOT/bench/evaluation/i08-schema.yaml" >"$tmp/missing-out" 2>"$tmp/missing-error"; then exit 1; fi
grep -q '/eligibility/publication_eligible' "$tmp/missing-error"

echo "TC-074: invalid record retained and excluded from eligibility"
