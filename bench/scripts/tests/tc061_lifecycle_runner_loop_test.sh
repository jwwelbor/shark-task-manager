#!/usr/bin/env bash
# TC-061 / T-E40-F08-002: caller-path coverage for the real lifecycle runner.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
fail() { echo "TC-061 FAIL: $1" >&2; exit 1; }
[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch/shark-data/workflow" "$WORKDIR/scratch/shark-data/prompts/_shared"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/fixture"
cat >"$WORKDIR/scratch/shark-data/workflow/bug.yaml" <<'YAML'
version: "1.0"
start: development
steps:
  development:
    phase: development
    action: spawn_agent
    provider: anthropic
    model: fixture-model
    prompt: bug/development.md
    outcomes: {pass: code_review}
  code_review:
    phase: code_review
    action: spawn_agent
    provider: anthropic
    model: fixture-model
    prompt: _shared/code_review.md
    outcomes: {pass: qa, fail: development}
  qa:
    phase: qa
    action: spawn_agent
    provider: anthropic
    model: fixture-model
    prompt: _shared/qa.md
    outcomes: {pass: completed, fail: development}
  completed: {phase: done, action: archive, terminal: true}
YAML
printf '%s\n' 'review prompt' >"$WORKDIR/scratch/shark-data/prompts/_shared/code_review.md"
printf '%s\n' 'qa prompt' >"$WORKDIR/scratch/shark-data/prompts/_shared/qa.md"

cat >"$WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
with open(os.environ["SHARK_EVENTS"], "a") as f:
    f.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    response = json.load(open(os.environ["SHARK_RESPONSE"]))
    response["provider"] = "anthropic"
    state_path = os.environ["SHARK_STATE"]
    try:
        next_count = int(open(state_path).read())
    except (FileNotFoundError, ValueError):
        next_count = 0
    next_count += 1
    open(state_path, "w").write(str(next_count))
    if next_count == 2:
        response["status"] = "code_review"
    elif next_count > 2:
        response = {"action": "archive", "entity_key": "ROOT-001", "entity_type": "bug"}
    if response["action"] == "spawn_agent":
        path = args[args.index("--prompt-out") + 1]
        open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:2] == ["claim", "TASK-002"]:
    next_count = int(open(os.environ["SHARK_STATE"]).read())
    print(json.dumps({"session_id": f"SID-{next_count:03d}"}, separators=(",", ":")))
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
elif args[:2] == ["get", "TASK-002"]:
    print('{"key":"TASK-002","notes":[]}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "test" ]]; then
	printf '%s\n' '{"entries":[{"id":"tests.test_due_date::test_due_today_is_overdue","outcome":"pass"}]}'
	exit 0
fi
sleep 0.05
python3 /dev/fd/3 3<<'PY'
import json
import os
import sys

request = json.load(sys.stdin)
with open(os.environ["ADAPTER_REQUEST"], "a", encoding="utf-8") as stream:
    stream.write(json.dumps(request, separators=(",", ":")) + "\n")
if not os.path.exists(os.environ["ADAPTER_MUTATION_MARKER"]):
    with open(os.path.join(os.environ["E40_AGENT_FIXTURE_CHECKOUT"], "taskmanager", "due_date.py"), "a", encoding="utf-8") as stream:
        stream.write("\n# TC-061 stage-end candidate identity mutation\n")
    cache_dir = os.path.join(os.environ["E40_AGENT_FIXTURE_CHECKOUT"], "taskmanager", "__pycache__")
    os.makedirs(cache_dir, exist_ok=True)
    with open(os.path.join(cache_dir, "provider.cache"), "w", encoding="utf-8") as stream:
        stream.write("provider-created ignored bytes\n")
    open(os.environ["ADAPTER_MUTATION_MARKER"], "w").close()
usage = {
    "cost_usd": 0.0,
    "input_tokens": 10,
    "output_tokens": 5,
    "cache_read_input_tokens": 0,
    "cache_creation_input_tokens": 0,
    "model_ids": ["fixture-model"],
    "api_active_duration_ms": 20,
    "turn_count": 1,
    "provider_session_id": "provider-session-061",
}
if os.environ.get("PARTIAL_USAGE") == "1":
    usage = {"input_tokens": 10}
print(json.dumps({
    "worker_id": "worker-061",
    "session_id": request["session_id"],
    "kind": "final",
    "recommended_outcome": "pass",
    "evidence": {"summary": "fixture complete"},
    "cost_usd": 0.0,
    "usage": usage,
}, separators=(",", ":")))
PY
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/next-count" ADAPTER_REQUEST="$WORKDIR/requests.ndjson" \
	ADAPTER_MUTATION_MARKER="$WORKDIR/adapter-mutated" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
	--run-id tc061 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
	--fixture-root "$WORKDIR/fixture" \
    --evidence-root "$WORKDIR/evidence" --output "$WORKDIR/lifecycle.jsonl" >/dev/null

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/requests.ndjson" "$WORKDIR/lifecycle.jsonl" <<'PY'
import hashlib, json, os, sys
events = [json.loads(line) for line in open(sys.argv[1])]
requests = [json.loads(line) for line in open(sys.argv[2])]
record = json.loads(open(sys.argv[3]).readline())
names = [e["argv"][0] for e in events]
assert names[0:2] == ["next", "claim"] and names[-1] == "next", events
assert names.count("next") == 3, events
assert names.count("claim") == 2, events
assert names.count("status") == 2, events
assert names.count("release") == 2, events
assert names.count("heartbeat") >= 1, events
assert events[0]["argv"][:4] == ["next", "ROOT-001", "--json", "--prompt-out"]
assert events[1]["argv"][1] == "TASK-002"
heartbeats = [e for e in events if e["argv"][0] == "heartbeat"]
assert all(e["argv"][1] == "TASK-002" for e in heartbeats)
assert len(requests) == 2, requests
for index, request in enumerate(requests, start=1):
    assert request["session_id"] == f"SID-{index:03d}", request
    assert request["prompt"] == "run exact bytes\n"
    assert request["prompt_sha256"] == hashlib.sha256(request["prompt"].encode()).hexdigest()
    assert request["prompt_bytes"] == len(request["prompt"].encode())
    assert not any(k in request for k in ("claim", "heartbeat", "advance", "release"))
assert record["dispatches"][0]["response"]["resolved_via"] == ["ROOT-001"]
assert "prompt" not in record["dispatches"][0]["response"]
assert [item["response"]["status"] for item in record["dispatches"]] == ["development", "code_review"], record["dispatches"]
assert record["outcome"]["terminal"] == "complete"
assert record["outcome"]["publication_eligible"] is True
assert record["prelude"]["terminal_outcome"] == "not_applicable", record.get("prelude")
assert [item["stage"] for item in record["prelude"]["stages"]] == ["D01", "D02", "D03", "D04", "D05"]
assert record["workflow_policy"]["enabled_gates"] == ["code_review", "qa"], record["workflow_policy"]
assert record["workflow_policy"]["gate_order"] == ["code_review", "qa"], record["workflow_policy"]
assert len(record["workflow_policy"]["gate_policies"]) == 2, record["workflow_policy"]
assert len(record["review_gates"]) == 2, record["review_gates"]
assert record["review_gates"][0]["state"] == "zero_findings", record["review_gates"]
assert record["review_gates"][1]["gate_id"] == "qa" and record["review_gates"][1]["state"] == "not_reached", record["review_gates"]
bundle = json.load(open(os.path.join(os.path.dirname(sys.argv[3]), "evidence", "bundle.json")))
assert [stage["dispatch_ordinal"] for stage in bundle["stages"]] == [1, 2], bundle
assert bundle["dispatches"] == record["dispatches"]
assert bundle["prelude"] == record["prelude"]
for stage in record["stages"]:
    candidate = stage["candidate"]
    assert candidate["base_commit"] != "0" * 40
    assert all(candidate[field] != "0" * 64 for field in ("tree_digest", "binary_diff_digest", "changed_path_digest", "test_suite_digest", "identity_digest", "snapshot_digest"))
    assert "tests.test_due_date::test_is_overdue_true_for_past_date" in candidate["test_suite_ids"], candidate
    interval_seconds = sum(interval["end"] - interval["start"] for interval in stage["intervals"])
    assert abs(interval_seconds - stage["elapsed_seconds"]) < 1e-9, stage
    assert stage["elapsed_seconds"] < 10, stage
    assert any(interval["category"] == "queue_or_claim_wait" for interval in stage["intervals"]), stage
    assert any(interval["category"] == "unclassified" for interval in stage["intervals"]), stage
    assert not any(interval["category"] == "provider_active" for interval in stage["intervals"]), stage
    assert stage["errors"] == [], stage
    assert {item["source_kind"] for item in stage["input_lineage"]} >= {
        "scenario_package", "rendered_prompt", "fixture_checkout",
        "shark_content", "execution_adapter", "lifecycle_adapter",
    }, stage
first = record["stages"][0]
manifest = {entry["path"]: entry for entry in first["candidate"]["dirty_untracked_manifest"]}
assert manifest["taskmanager/due_date.py"]["tracked"] is True, first
assert manifest["taskmanager/due_date.py"]["digest"].startswith("sha256:"), first
assert manifest["taskmanager/__pycache__/provider.cache"]["tracked"] is False, first
assert first["candidate"]["binary_diff_digest"] == first["output_digests"][0], first
stage_snapshot = json.load(open(os.path.join(os.path.dirname(sys.argv[3]), "evidence", "stages", "0001-development.json")))
assert stage_snapshot["candidate"]["test_suite_ids"] == first["candidate"]["test_suite_ids"], stage_snapshot
assert stage_snapshot["candidate"]["test_suite_dir"] == first["candidate"]["test_suite_dir"], stage_snapshot
transcript_dir = os.path.join(os.path.dirname(sys.argv[3]), "evidence", "transcripts")
assert sorted(os.listdir(transcript_dir)) == ["0001-development.json", "0002-code_review.json"]
PY

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/evidence" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/lifecycle.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >/dev/null
"$SCRIPTS_DIR/replay-stage-evidence.sh" "$WORKDIR/evidence" \
	--checkout "$WORKDIR/fixture" --adapter "$SCRIPTS_DIR/../adapters/python/adapter.sh" >/dev/null

# A mechanically complete workflow with incomplete provider usage must fail
# closed instead of becoming publication eligible.
mkdir -p "$WORKDIR/partial-scratch"
cp -a "$WORKDIR/scratch/shark-data" "$WORKDIR/partial-scratch/shark-data"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/partial-fixture"
: >"$WORKDIR/partial-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/partial-events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/partial-next-count" ADAPTER_REQUEST="$WORKDIR/partial-requests.ndjson" \
	ADAPTER_MUTATION_MARKER="$WORKDIR/partial-adapter-mutated" PARTIAL_USAGE=1 \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
	--run-id tc061-partial --root ROOT-001 --scratch-root "$WORKDIR/partial-scratch" \
	--fixture-root "$WORKDIR/partial-fixture" \
    --evidence-root "$WORKDIR/partial-evidence" --output "$WORKDIR/partial-lifecycle.jsonl" >/dev/null
python3 - "$WORKDIR/partial-lifecycle.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["outcome"]["terminal"] == "error", record["outcome"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
assert any(
    error["kind"] == "usage_slot_unavailable"
    for stage in record["stages"] for error in stage["errors"]
), record["stages"]
PY

echo "TC-061: pass (canonical multi-stage claim/heartbeat/transition/release loop through archive)"
