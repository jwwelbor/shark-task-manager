#!/usr/bin/env bash
# TC-068: the held-back oracle must refuse pre-terminal access before invoking
# an adapter and must use the existing I-05 access broker after terminality.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
! rg -q 'events\[-1\]' "$SCRIPT_DIR/../run-heldback-oracle.sh" || {
  echo "TC-068: oracle must not select provenance from an unrelated last event" >&2
  exit 1
}

REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ORACLE="$REPO_ROOT/bench/scripts/run-heldback-oracle.sh"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"

[[ -x "$ORACLE" ]] || { echo "TC-068: oracle missing or not executable" >&2; exit 1; }
grep -q 'adapter_calls' "$ORACLE" || { echo "TC-068: oracle must retain adapter call accounting" >&2; exit 1; }
grep -q 'restore_checkout' "$ORACLE" || { echo "TC-068: oracle must restore checkout on post-grant failure" >&2; exit 1; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bundle" "$tmp/checkout"
echo '{"terminal_status":{"reached":false}}' > "$tmp/bundle/bundle.json"
python3 - "$tmp/bundle" "$tmp/checkout" <<'PY'
import json, sys
path, checkout = sys.argv[1:]
with open(path + "/bundle.json", "w", encoding="utf-8") as stream:
    json.dump({"terminal_status": {"reached": False}, "roots": {"agent_fixture_checkout": checkout}}, stream)
    stream.write("\n")
PY
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

# Caller-path proof: the evaluator must own construction and invocation of the
# oracle, while retaining the same pre-terminal result and untouched roots.
eval_output="$tmp/evaluation.jsonl"
if "$EVALUATOR" --i05 "$tmp/bundle" --i07 "$tmp/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$eval_output" >/dev/null 2>"$tmp/evaluator.stderr"; then
  echo "TC-068: evaluator pre-terminal oracle unexpectedly passed" >&2
  exit 1
fi
python3 - "$eval_output" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["execution_oracle"]["observed_result"] == "not_run"
assert any(item["code"] == "pre_terminal" for item in record["eligibility"]["invalidity_reasons"])
PY
echo "TC-068: evaluator entrypoint retains pre-terminal oracle refusal"
