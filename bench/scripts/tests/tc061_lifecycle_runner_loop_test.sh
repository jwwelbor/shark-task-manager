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
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch"

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
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
sleep 0.05
python3 /dev/fd/3 3<<'PY'
import json
import os
import sys

request = json.load(sys.stdin)
with open(os.environ["ADAPTER_REQUEST"], "a", encoding="utf-8") as stream:
    stream.write(json.dumps(request, separators=(",", ":")) + "\n")
print(json.dumps({
    "worker_id": "worker-061",
    "session_id": request["session_id"],
    "kind": "final",
    "recommended_outcome": "pass",
    "evidence": {"summary": "fixture complete"},
}, separators=(",", ":")))
PY
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/next-count" ADAPTER_REQUEST="$WORKDIR/requests.ndjson" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc061 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
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
bundle = json.load(open(os.path.join(os.path.dirname(sys.argv[3]), "evidence", "bundle.json")))
assert [stage["dispatch_ordinal"] for stage in bundle["stages"]] == [1, 2], bundle
assert bundle["dispatches"] == record["dispatches"]
for stage in record["stages"]:
    candidate = stage["candidate"]
    assert candidate["base_commit"] != "0" * 40
    assert all(candidate[field] != "0" * 64 for field in ("tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest", "identity_digest", "snapshot_digest"))
    interval_seconds = sum(interval["end"] - interval["start"] for interval in stage["intervals"])
    assert abs(interval_seconds - stage["elapsed_seconds"]) < 1e-9, stage
    assert stage["elapsed_seconds"] < 10, stage
transcript_dir = os.path.join(os.path.dirname(sys.argv[3]), "evidence", "transcripts")
assert sorted(os.listdir(transcript_dir)) == ["0001-development.json", "0002-code_review.json"]
PY

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/evidence" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/lifecycle.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >/dev/null

echo "TC-061: pass (canonical multi-stage claim/heartbeat/transition/release loop through archive)"
