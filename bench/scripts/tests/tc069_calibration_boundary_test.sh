#!/usr/bin/env bash
# TC-069: judge evidence is eligible only with complete, disjoint calibration
# provenance; no default score may be fabricated.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"

[[ -x "$EVALUATOR" ]] || { echo "TC-069: evaluator missing or not executable" >&2; exit 1; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/i05"
echo '{"roots":{"agent_fixture_checkout":"/does/not/exist"},"stages":[{"stage_path":"stages/code.json"}]}' > "$tmp/i05/bundle.json"
echo '{"identity":{"run_id":"tc069","scenario_id":"tc069"},"entity_graph":{"nodes":["T"]},"dispatches":[{"transition":"development"}],"stages":[{"category":"code","input_lineage":"input"}],"outcome":{"terminal":"complete"}}' > "$tmp/i07.jsonl"
scenario="$tmp/scenario.yaml"
cat > "$scenario" <<'YAML'
scenario_id: tc069
scenario_version: 1
stage_matrix:
  prelude:
    D01: {applicable: true}
fixture: {fixture_id: py}
adapter: {name: python, version: "1.0.0"}
YAML
output="$tmp/evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/i07.jsonl" --scenario "$scenario" --output "$output" >/dev/null 2>"$tmp/stderr"; then echo "TC-069: missing calibration unexpectedly passed" >&2; exit 1; fi
python3 - "$output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["judge"]["applicability"] == "applicable"
assert record["judge"]["observed_result"] == "fail"
assert any(item["code"] == "missing_judge" for item in record["eligibility"]["invalidity_reasons"])
assert record["eligibility"]["aggregate_eligible"] is False
PY
echo "TC-069: missing calibration cannot fabricate an eligible score"
