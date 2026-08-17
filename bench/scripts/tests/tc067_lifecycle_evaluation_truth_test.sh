#!/usr/bin/env bash
# TC-067: the evaluator keeps structural, calibrated-judge, and held-back
# oracle truth independent; terminal status and worker exit cannot substitute.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"

[[ -x "$EVALUATOR" ]] || { echo "TC-067: evaluator missing or not executable" >&2; exit 1; }
! rg -q 'workflow_policy\.setdefault' "$EVALUATOR" || {
  echo "TC-067: evaluator must not fabricate workflow-policy identity" >&2
  exit 1
}
! rg -q 'identity\["shark_content_digest"\]\s*=' "$EVALUATOR" || {
  echo "TC-067: evaluator must not overwrite producer content identity" >&2
  exit 1
}
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/i05"
echo '{"roots":{"agent_fixture_checkout":"/does/not/exist"},"stages":[{"stage_path":"stages/code.json"}]}' > "$tmp/i05/bundle.json"
echo '{"identity":{"run_id":"tc067","scenario_id":"py-bug-due-date-boundary"},"entity_graph":{"nodes":["T"]},"dispatches":[{"transition":"development"}],"stages":[{"category":"code","input_lineage":"input"}],"outcome":{"terminal":"complete"}}' > "$tmp/i07.jsonl"
output="$tmp/evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$output" >/dev/null 2>"$tmp/stderr"; then status=0; else status=$?; fi
[[ "$status" -ne 0 ]] || { echo "TC-067: incomplete oracle evidence must be ineligible" >&2; exit 1; }
python3 - "$output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["structural"]["observed_result"] == "pass"
assert record["judge"]["observed_result"] == "not_applicable"
assert record["execution_oracle"]["observed_result"] == "not_run"
assert record["eligibility"]["aggregate_eligible"] is False
assert any(item["code"] == "missing_oracle" for item in record["eligibility"]["invalidity_reasons"])
assert any(item["code"] == "identity_missing" and item["path"] == "/identity/toolchain_identity" for item in record["eligibility"]["invalidity_reasons"])
PY

# Producer identity is not completed from the scenario package at the join.
cat > "$tmp/missing-producer-identity-i07.jsonl" <<'JSON'
{"identity":{"run_id":"missing-identity-run","scenario_id":"py-bug-due-date-boundary"},"dispatches":[],"stages":[],"outcome":{"terminal":"complete"}}
JSON
missing_identity_output="$tmp/missing-producer-identity-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/missing-producer-identity-i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$missing_identity_output" >/dev/null 2>/dev/null; then
  echo "TC-067: missing producer identity unexpectedly passed" >&2
  exit 1
fi
python3 - "$missing_identity_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
reasons = record["eligibility"]["invalidity_reasons"]
assert any(item["code"] == "missing_join" and item["path"] == "/identity/fixture_id" for item in reasons), reasons
assert record["identity"].get("fixture_id") is None
PY

# Cross-artifact join counter-factual: matching-looking scenario labels must
# not hide a contradictory I-05/I-07 run reference.
mkdir -p "$tmp/join-i05"
cat > "$tmp/join-i05/bundle.json" <<'JSON'
{"identity":{"run_id":"i05-run","scenario_id":"py-bug-due-date-boundary","scenario_version":1},"dispatches":[{"ordinal":1,"requested_key":"T-1"}],"roots":{"agent_fixture_checkout":"/does/not/exist"},"stages":[{"stage_path":"stages/code.json"}]}
JSON
cat > "$tmp/join-i07.jsonl" <<'JSON'
{"identity":{"run_id":"i07-run","scenario_id":"py-bug-due-date-boundary","scenario_version":1},"dispatches":[{"ordinal":1,"requested_key":"T-1"}],"stages":[],"outcome":{"terminal":"complete"}}
JSON
join_output="$tmp/join-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/join-i05" --i07 "$tmp/join-i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$join_output" >/dev/null 2>/dev/null; then
  echo "TC-067: contradictory cross-artifact join unexpectedly eligible" >&2
  exit 1
fi
python3 - "$join_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert any(item["code"] == "contradictory_join" and item["path"] == "/join/run_id" for item in record["eligibility"]["invalidity_reasons"]), record
PY

# Missing workflow-policy identity is a counter-factual invalid record, not a
# digest the evaluator may reconstruct from the remaining fields.
mkdir -p "$tmp/policy-i05"
cp "$tmp/i05/bundle.json" "$tmp/policy-i05/bundle.json"
cat > "$tmp/policy-i07.jsonl" <<'JSON'
{"identity":{"run_id":"policy-run","scenario_id":"py-bug-due-date-boundary","scenario_version":1},"dispatches":[],"workflow_policy":{"enabled_gates":["qa"],"gate_order":["qa"],"reviewer":{"provider":"fixture","model":"reviewer","effort":"low"},"prompt_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","rendered_prompt_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","deep_review_bundle_digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","fixes_allowed_between_gates":false},"stages":[],"outcome":{"terminal":"complete"}}
JSON
policy_output="$tmp/policy-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/policy-i05" --i07 "$tmp/policy-i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$policy_output" >/dev/null 2>/dev/null; then
  echo "TC-067: missing workflow-policy identity unexpectedly eligible" >&2
  exit 1
fi
python3 - "$policy_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
reasons = record["eligibility"]["invalidity_reasons"]
assert any(item["code"] == "identity_missing" and item["path"] == "/workflow_policy/workflow_policy_identity_digest" for item in reasons), reasons
PY

# Parseable but malformed nested shape must remain a bounded invalid record.
cat > "$tmp/malformed-nested-i07.jsonl" <<'JSON'
{"identity":[],"stages":[],"outcome":{"terminal":"complete"}}
JSON
malformed_output="$tmp/malformed-nested-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/malformed-nested-i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$malformed_output" >/dev/null 2>/dev/null; then
  echo "TC-067: malformed nested identity unexpectedly eligible" >&2
  exit 1
fi
python3 - "$malformed_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert any(item["code"] == "source_malformed" for item in record["eligibility"]["invalidity_reasons"])
PY

echo "TC-067: truth blocks remain independent and missing oracle evidence is retained"
