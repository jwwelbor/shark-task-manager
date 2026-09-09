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
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch" "$WORKDIR/i05"

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
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:2] == ["claim", "TASK-002"]:
    print('{"session_id":"SID-002"}')
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], level + ".json")) as f:
        print(f.read())
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
sleep 0.05
python3 -c 'import json,os,sys; json.dump(json.load(sys.stdin), open(os.environ["ADAPTER_REQUEST"], "w"), separators=(",", ":"))'
printf '%s\n' '{"worker_id":"worker-061","session_id":"SID-002","kind":"final","recommended_outcome":"pass","evidence":{"summary":"fixture complete"}}'
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" ADAPTER_REQUEST="$WORKDIR/request.json" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc061 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
    --i05-bundle-dir "$WORKDIR/i05" \
    --output "$WORKDIR/lifecycle.jsonl" >/dev/null

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/request.json" "$WORKDIR/lifecycle.jsonl" <<'PY'
import hashlib, json, sys
events = [json.loads(line) for line in open(sys.argv[1])]
request = json.load(open(sys.argv[2]))
record = json.loads(open(sys.argv[3]).readline())
names = [e["argv"][0] for e in events]
# T-E40-F12-002: record_stage()'s own phase lookup (`admin workflow list`)
# runs after the dispatch's claim/heartbeat/status/release sequence
# completes, so it legitimately appends one trailing "admin" event here --
# excluded from this test's own claim/heartbeat/transition/release
# ordering assertion, which is what TC-061 actually covers.
non_i05_names = [name for name in names if name != "admin"]
assert non_i05_names[0:2] == ["next", "claim"] and non_i05_names[-2:] == ["status", "release"], events
assert names.count("heartbeat") >= 1, events
assert events[0]["argv"][:4] == ["next", "ROOT-001", "--json", "--prompt-out"]
assert events[1]["argv"][1] == "TASK-002"
heartbeats = [e for e in events if e["argv"][0] == "heartbeat"]
assert all(e["argv"][1] == "TASK-002" and "SID-002" in e["argv"] for e in heartbeats)
non_i05_events = [e for e in events if e["argv"][0] != "admin"]
status = non_i05_events[-2]
release = non_i05_events[-1]
assert "SID-002" in status["argv"] and "development" in status["argv"]
assert release["argv"][1] == "TASK-002" and "SID-002" in release["argv"]
assert record["dispatches"][0]["heartbeats"]
assert all(item["session_id"] == "SID-002" for item in record["dispatches"][0]["heartbeats"])
assert request["prompt"] == "run exact bytes\n"
assert request["prompt_sha256"] == hashlib.sha256(request["prompt"].encode()).hexdigest()
assert request["prompt_bytes"] == len(request["prompt"].encode())
assert not any(k in request for k in ("claim", "heartbeat", "advance", "release"))
assert record["dispatches"][0]["response"]["resolved_via"] == ["ROOT-001"]
assert "prompt" not in record["dispatches"][0]["response"]
assert record["outcome"]["terminal"] == "complete"
assert record["outcome"]["publication_eligible"] is True
candidate = record["stages"][0]["candidate"]
assert candidate["base_commit"] != "0" * 40
assert all(candidate[field] != "0" * 64 for field in ("tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest", "identity_digest", "snapshot_digest"))
PY

echo "TC-061: pass (canonical claim/heartbeat/transition/release and exact prompt handoff)"
