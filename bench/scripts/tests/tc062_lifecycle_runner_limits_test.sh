#!/usr/bin/env bash
# TC-062 / T-E40-F08-002: first-exceeded resource ceiling coverage.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
fail() { echo "TC-062 FAIL: $1" >&2; exit 1; }
[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch"

cat >"$WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, os, sys
args = sys.argv[1:]
with open(os.environ["SHARK_EVENTS"], "a") as f:
    f.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    print(json.dumps({"mode":"hierarchy_selection","action":"parallel_candidates",
        "root_key":"ROOT-001","root_type":"feature","selection_reason":"fixture",
        "resolved_via":["ROOT-001"],"parallel_execution":"available",
        "entities":[{"entity_key":"TASK-003","entity_type":"task"},
                    {"entity_key":"TASK-001","entity_type":"task"},
                    {"entity_key":"TASK-002","entity_type":"task"}]}, separators=(",", ":")))
elif args[0] == "next":
    key = args[1]; prompt = "work " + key + "\n"
    response = {"entity_key":key,"entity_type":"task","status":"development",
        "action":"spawn_agent","agent_type":"developer","provider":"fixture",
        "model":"fixture-model","effort":"medium","prompt":prompt,
        "prompt_sha256":hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes":len(prompt.encode()),"resolved_via":["ROOT-001"],
        "unresolved_placeholders":[],"error":"","question_block":None,
        "current_responder":""}
    path = args[args.index("--prompt-out") + 1]; open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[0] == "claim": print(json.dumps({"session_id":"SID-" + args[1]}))
elif args[0] == "heartbeat": print('{"ok":true}')
elif args[:2] == ["status", "advance"]: print('{"advanced":true}')
elif args[0] == "release": print('{"released":true}')
else: raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
python3 -c 'import json,sys; request=json.load(sys.stdin); cost=0.02 if request["entity_key"] == "TASK-001" else 0.0; print(json.dumps({"worker_id":"worker-062","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":cost,"evidence":{"summary":"fixture"}}))'
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
    --limits "$SCRIPTS_DIR/testdata/lifecycle/limits/first-exceed.yaml" --output "$WORKDIR/lifecycle.jsonl"

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/lifecycle.jsonl" <<'PY'
import json, sys
events = [json.loads(line) for line in open(sys.argv[1])]
record = json.loads(open(sys.argv[2]).readline())
assert [e["argv"][1] for e in events if e["argv"][0] == "next"] == ["ROOT-001", "TASK-001"]
assert record["limits"]["observed_generated_tasks"] == 1
assert record["limits"]["first_exceeded"] == "max_cost_usd"
assert record["outcome"]["terminal"] == "resource_limit"
assert record["outcome"]["partial_evidence"] is True
assert record["outcome"]["publication_eligible"] is False
assert record["outcome"]["reason"]
assert sum(e["argv"][0] == "release" for e in events) == 1
assert not any(e["argv"][0] == "next" and e["argv"][1] == "TASK-002" for e in events)
PY

echo "TC-062: pass (first exceeded ceiling stops scenario and retains partial evidence)"
