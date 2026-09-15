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
assert record["structural"]["observed_result"] == "fail"
assert record["judge"]["observed_result"] == "not_applicable"
assert record["execution_oracle"]["observed_result"] == "not_run"
assert record["eligibility"]["aggregate_eligible"] is False
assert any(item["code"] == "missing_oracle" for item in record["eligibility"]["invalidity_reasons"])
assert any(item["code"] == "source_malformed" and item["path"] == "/stages/0/input_lineage" for item in record["eligibility"]["invalidity_reasons"])
assert any(item["code"] == "identity_missing" and item["path"] == "/identity/toolchain_identity" for item in record["eligibility"]["invalidity_reasons"])
PY
python3 - "$output.oracle.json" <<'PY'
import json, sys
oracle = json.load(open(sys.argv[1], encoding="utf-8"))
assert oracle["observed_result"] == "not_run", oracle
assert oracle["invalidity_reasons"], oracle
PY

# A later stage's prior-artifact lineage and the producer's consumer graph
# must describe the same observed edge.
python3 - "$tmp/contradictory-consumer-i07.jsonl" <<'PY'
import hashlib, json, sys
digest = lambda value: hashlib.sha256(value.encode()).hexdigest()
lineage = [
    {"source_kind": kind, "path": f"/{kind}", "digest": digest(kind)}
    for kind in (
        "scenario_package", "rendered_prompt", "fixture_checkout",
        "shark_content", "execution_adapter", "lifecycle_adapter",
        "agent_visible_input",
    )
]
record = {
    "identity": {"run_id": "tc067", "scenario_id": "py-bug-due-date-boundary"},
    "entity_graph": {"nodes": ["T"]},
    "dispatches": [{"transition": "development"}, {"transition": "code_review"}],
    "stages": [
        {"category": "code", "input_lineage": lineage, "artifacts": [
            {"path": "artifacts/0001.patch", "digest": digest("artifact-1"), "consumers": [
                {"consuming_stage": "review", "edge_kind": "read", "observed_at": "   "},
            ]},
        ]},
        {"stage": "review", "category": "review", "input_lineage": lineage + [
            {"source_kind": "prior_stage_artifact", "path": "artifacts/0001.patch", "digest": digest("artifact-1")},
        ], "artifacts": []},
    ],
    "outcome": {"terminal": "complete"},
}
with open(sys.argv[1], "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY
contradictory_consumer_output="$tmp/contradictory-consumer-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/contradictory-consumer-i07.jsonl" \
    --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --output "$contradictory_consumer_output" >/dev/null 2>/dev/null; then
  echo "TC-067: contradictory artifact-consumer graph unexpectedly eligible" >&2
  exit 1
fi
python3 - "$contradictory_consumer_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
reasons = record["eligibility"]["invalidity_reasons"]
assert any(
    item["code"] == "source_malformed" and item["path"] == "/stages/0/artifacts/0/consumers"
    for item in reasons
), reasons
PY

sed 's/"terminal":"complete"/"terminal":"resource_limit"/' "$tmp/i07.jsonl" >"$tmp/stopped-i07.jsonl"
stopped_output="$tmp/stopped-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/stopped-i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$stopped_output" >/dev/null 2>/dev/null; then
  echo "TC-067: resource-limit lifecycle unexpectedly eligible" >&2
  exit 1
fi
python3 - "$stopped_output" "$stopped_output.oracle.json" <<'PY'
import json, sys
evaluation = json.load(open(sys.argv[1], encoding="utf-8"))
oracle = json.load(open(sys.argv[2], encoding="utf-8"))
assert evaluation["execution_oracle"] == oracle, (evaluation, oracle)
assert oracle["observed_result"] == "not_run", oracle
assert any(item["code"] == "aggregate_ineligible" for item in oracle["invalidity_reasons"]), oracle
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

# QA round-2 minor observation (docs/review/.../qa-20260819T033417Z-E40-F09.md,
# same defect class as QA2-001): evaluate-lifecycle.sh:446-447 rejects any
# caller-supplied --review-findings with review_findings_bypass ("evaluator
# owns normalization"), but no committed test ever passed that flag, so the
# rejection branch was never invoked. The flag's value is never opened before
# the rejection check, so a nonexistent path is sufficient to prove it.
bypass_output="$tmp/review-findings-bypass-evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/i05" --i07 "$tmp/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$bypass_output" --review-findings "$tmp/does-not-exist.json" >/dev/null 2>/dev/null; then
  echo "TC-067: caller-supplied --review-findings unexpectedly accepted" >&2
  exit 1
fi
python3 - "$bypass_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
reasons = record["eligibility"]["invalidity_reasons"]
assert any(item["code"] == "review_findings_bypass" and item["path"] == "/review_findings" for item in reasons), reasons
assert record["eligibility"]["aggregate_eligible"] is False
PY
echo "TC-067: caller-supplied --review-findings is rejected as review_findings_bypass"

# A held-back oracle can create its output file and still fail JSON decoding.
# Execute evaluate-lifecycle.sh's own run_oracle() implementation directly
# with that controlled dependency, then prove it replaces corrupt bytes with
# the exact synthetic result returned to the I-08 builder.
python3 - "$EVALUATOR" "$tmp" <<'PY'
import argparse
import json
import os
import re
import subprocess
import sys

evaluator, temp_dir = sys.argv[1:]
source = open(evaluator, encoding="utf-8").read()
match = re.search(r"def run_oracle\(.*?\n\ndef build_source_artifacts", source, re.DOTALL)
assert match, "run_oracle() source not found"
implementation = match.group(0).rsplit("\n\ndef build_source_artifacts", 1)[0]
output = os.path.join(temp_dir, "decode-error-evaluation.jsonl")
oracle = os.path.join(temp_dir, "malformed-oracle.sh")
with open(oracle, "w", encoding="utf-8") as stream:
    stream.write("#!/usr/bin/env bash\nprintf '{malformed oracle output\\n' > \"${10}\"\nexit 1\n")
os.chmod(oracle, 0o755)

namespace = {
    "args": argparse.Namespace(output=output, scenario="scenario", i07="lifecycle.jsonl", i05="bundle"),
    "oracle": oracle,
    "STOP_OUTCOMES": set(),
    "json": json,
    "os": os,
    "subprocess": subprocess,
    "reason": lambda code, path, detail: {"code": code, "path": path, "detail": detail},
}
exec(compile(implementation, "evaluate-lifecycle.sh::run_oracle", "exec"), namespace)
result = namespace["run_oracle"](
    {"roots": {"agent_fixture_checkout": temp_dir}},
    {"outcome": {"terminal": "complete"}},
    [],
)
sidecar = json.load(open(output + ".oracle.json", encoding="utf-8"))
assert result == sidecar, (result, sidecar)
assert result["observed_result"] == "not_run", result
assert any(item["code"] == "missing_oracle" for item in result["invalidity_reasons"]), result
PY
echo "TC-067: malformed oracle sidecar is replaced with the recorded synthetic result"

echo "TC-067: truth blocks remain independent and missing oracle evidence is retained"
