#!/usr/bin/env bash
# TC-060 (test-plan.md TC-002/TC-003/TC-010/TC-012; T-E40-F08-003).
#
# Caller-path contract: invoke the real provider-neutral adapter as a
# subprocess and stub only the external provider executable. The adapter is
# responsible for prompt verification, foreground waiting, bounded result
# projection, and authority redaction.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ADAPTER="$SCRIPTS_DIR/lifecycle-worker-adapter.sh"
TESTDATA="$SCRIPTS_DIR/testdata/lifecycle"

fail() {
	echo "TC-060 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADAPTER" ]] || fail "lifecycle-worker-adapter.sh missing or not executable"
[[ -x "$TESTDATA/bin/worker-canary" ]] || fail "worker canary missing or not executable"
[[ -f "$TESTDATA/adapter/request-complete.json" ]] || fail "complete request fixture missing"
[[ -f "$TESTDATA/adapter/worker-result-sensitive.json" ]] || fail "worker result fixture missing"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

REQUEST="$TESTDATA/adapter/request-complete.json"
PROMPT="$TESTDATA/adapter/prompt.txt"
RESULT_FIXTURE="$TESTDATA/adapter/worker-result-sensitive.json"
CANARY_LOG="$WORKDIR/adapter-argv.ndjson"
OUT="$WORKDIR/result.json"

echo "TC-060: TC-002 exact prompt handoff and bounded semantic result"
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$RESULT_FIXTURE" \
LIFECYCLE_WORKER_ID="worker-canary-001" \
"$ADAPTER" --request "$REQUEST" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT"

python3 - "$OUT" "$CANARY_LOG" "$PROMPT" <<'PYEOF'
import hashlib
import json
import pathlib
import sys

result_path, log_path, prompt_path = map(pathlib.Path, sys.argv[1:])
result = json.loads(result_path.read_text())
expected_prompt = prompt_path.read_bytes()

assert result == {
    "worker_id": "worker-canary-001",
    "session_id": "SID-002",
    "kind": "final",
    "recommended_outcome": "deep_verify",
    "evidence": [
        {"type": "artifact", "path": "runs/tc060/evidence.json", "size_bytes": 42},
        "[REDACTED]",
    ],
    "prompt_sha256": hashlib.sha256(expected_prompt).hexdigest(),
    "prompt_bytes": len(expected_prompt),
}, result

assert "credential-sentinel" not in result_path.read_text()
assert "provider-secret-sentinel" not in result_path.read_text()
assert "transcript-line-999" not in result_path.read_text()

entries = [json.loads(line) for line in log_path.read_text().splitlines()]
assert len(entries) == 1, entries
entry = entries[0]
assert entry["argv"] == [], entry
assert entry["stdin_sha256"] == hashlib.sha256(expected_prompt).hexdigest(), entry
assert entry["stdin_bytes"] == len(expected_prompt), entry
assert entry["session_id"] == "SID-002", entry
assert entry["entity_key"] == "E40-F08-CHILD-001", entry
serialized = log_path.read_text()
for secret in ("run exact bytes", "credential-sentinel", "provider-secret-sentinel", "transcript-line-999"):
    assert secret not in serialized, (secret, serialized)
PYEOF
echo "TC-060(TC-002: exact prompt bytes, identity/session, outcome, bounded evidence, and no authority/leak) PASS"

echo "TC-060: controller stdin request path reaches the real adapter"
rm -f "$CANARY_LOG" "$OUT"
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$RESULT_FIXTURE" \
LIFECYCLE_WORKER_ID="worker-canary-001" \
"$ADAPTER" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT" <"$REQUEST"
python3 - "$OUT" "$CANARY_LOG" <<'PYEOF'
import json
import pathlib
import sys

result_path, log_path = map(pathlib.Path, sys.argv[1:])
result = json.loads(result_path.read_text())
entries = [json.loads(line) for line in log_path.read_text().splitlines()]
assert result["recommended_outcome"] == "deep_verify", result
assert len(entries) == 1, entries
assert entries[0]["entity_key"] == "E40-F08-CHILD-001", entries[0]
PYEOF
echo "TC-060(TC-002: stdin request path matches run-lifecycle.sh caller contract) PASS"

echo "TC-060: structured provider envelope preserves bounded usage"
STRUCTURED_RESULT="$WORKDIR/structured-result.json"
python3 - "$STRUCTURED_RESULT" <<'PYEOF'
import json
import sys

result = {
    "type": "result",
    "subtype": "success",
    "is_error": False,
    "structured_output": {
        "kind": "final",
        "recommended_outcome": "pass",
        "evidence": [],
    },
    "total_cost_usd": 0.125,
    "duration_api_ms": 1234,
    "num_turns": 2,
    "usage": {
        "input_tokens": 10,
        "output_tokens": 20,
        "cache_read_input_tokens": 30,
        "cache_creation_input_tokens": 40,
    },
    "modelUsage": {"claude-haiku-fixture": {"costUSD": 0.125}},
}
with open(sys.argv[1], "w", encoding="utf-8") as stream:
    json.dump(result, stream)
PYEOF
rm -f "$CANARY_LOG" "$OUT"
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$STRUCTURED_RESULT" \
"$ADAPTER" --request "$REQUEST" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT" >/dev/null
python3 - "$OUT" <<'PYEOF'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
assert result["worker_id"] == "fixture-fixture-model", result
assert result["recommended_outcome"] == "pass", result
assert result["cost_usd"] == 0.125, result
assert result["usage"] == {
    "api_active_duration_ms": 1234,
    "cache_creation_input_tokens": 40,
    "cache_read_input_tokens": 30,
    "cost_usd": 0.125,
    "input_tokens": 10,
    "model_ids": ["claude-haiku-fixture"],
    "output_tokens": 20,
    "turn_count": 2,
}, result
PYEOF
echo "TC-060(TC-002: structured output and real usage envelope projection) PASS"

echo "TC-060: legacy advance recommendation maps to the workflow-owned pass outcome"
ALIAS_REQUEST="$WORKDIR/request-allowed-outcomes.json"
ALIAS_RESULT="$WORKDIR/alias-result.json"
python3 - "$REQUEST" "$ALIAS_REQUEST" "$ALIAS_RESULT" <<'PYEOF'
import json
import sys

request = json.load(open(sys.argv[1], encoding="utf-8"))
request["allowed_outcomes"] = ["blocked", "fail", "on_hold", "pass"]
json.dump(request, open(sys.argv[2], "w", encoding="utf-8"), sort_keys=True)
json.dump({"kind": "final", "recommended_outcome": "advance", "evidence": []}, open(sys.argv[3], "w", encoding="utf-8"))
PYEOF
rm -f "$CANARY_LOG" "$OUT"
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$ALIAS_RESULT" \
"$ADAPTER" --request "$ALIAS_REQUEST" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT" >/dev/null
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["recommended_outcome"] == "pass"' "$OUT"
echo "TC-060(TC-002: adapter projects legacy prose outcome onto the workflow-owned outcome vocabulary) PASS"

echo "TC-060: provider session metadata cannot replace parent claim authority"
SESSION_RESULT="$WORKDIR/session-result.json"
python3 - "$SESSION_RESULT" <<'PYEOF'
import json
import sys

json.dump(
    {"kind": "final", "recommended_outcome": "deep_verify", "session_id": "provider-invented", "evidence": []},
    open(sys.argv[1], "w", encoding="utf-8"),
)
PYEOF
rm -f "$CANARY_LOG" "$OUT"
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$SESSION_RESULT" \
"$ADAPTER" --request "$REQUEST" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT" >/dev/null
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["session_id"] == "SID-002"' "$OUT"
echo "TC-060(TC-002: parent claim session remains authoritative over provider metadata) PASS"

echo "TC-060: TC-002 invalid prompt provenance refuses provider dispatch"
BAD_REQUEST="$WORKDIR/request-bad-digest.json"
python3 - "$REQUEST" "$BAD_REQUEST" <<'PYEOF'
import json
import sys

source, target = sys.argv[1:]
document = json.load(open(source))
document["prompt_sha256"] = "0" * 64
json.dump(document, open(target, "w"), sort_keys=True)
PYEOF
rm -f "$CANARY_LOG" "$OUT"
set +e
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
LIFECYCLE_WORKER_RESULT="$RESULT_FIXTURE" \
"$ADAPTER" --request "$BAD_REQUEST" --provider-command "$TESTDATA/bin/worker-canary" --result-out "$OUT" >"$WORKDIR/bad.out" 2>"$WORKDIR/bad.err"
code=$?
set -e
[[ "$code" -eq 2 ]] || fail "bad prompt digest exited $code, want 2: $(cat "$WORKDIR/bad.err")"
[[ ! -e "$CANARY_LOG" || ! -s "$CANARY_LOG" ]] || fail "provider ran after prompt digest mismatch"
grep -q "prompt_sha256" "$WORKDIR/bad.err" || fail "digest failure did not name prompt_sha256: $(cat "$WORKDIR/bad.err")"
echo "TC-060(TC-002: prompt digest mismatch fails before provider invocation) PASS"

echo "TC-060: TC-003 missing semantic result refuses exit-code substitution"
FAIL_CANARY="$TESTDATA/bin/worker-canary-fails"
rm -f "$CANARY_LOG" "$OUT"
set +e
LIFECYCLE_CANARY_LOG="$CANARY_LOG" \
LIFECYCLE_EXPECTED_PROMPT="$PROMPT" \
"$ADAPTER" --request "$REQUEST" --provider-command "$FAIL_CANARY" --result-out "$OUT" >"$WORKDIR/fail.out" 2>"$WORKDIR/fail.err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "provider failure exited $code, want 1: $(cat "$WORKDIR/fail.err")"
[[ ! -e "$OUT" ]] || fail "adapter wrote a semantic result after provider failure"
grep -q "provider exited" "$WORKDIR/fail.err" || fail "provider failure was not reported: $(cat "$WORKDIR/fail.err")"
echo "TC-060(TC-003: awaited provider failure never becomes a completed outcome) PASS"

echo "TC-060: PASS"
