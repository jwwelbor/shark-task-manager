#!/usr/bin/env bash
# TC-066: a worker question is minted and linked by the parent controller.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
fail() { echo "TC-066 FAIL: $1" >&2; exit 1; }
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch"

cat >"$WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
with open(os.environ["SHARK_EVENTS"], "a") as stream:
    stream.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "ROOT-001"]:
    response = {"entity_key":"TASK-001","entity_type":"task","status":"development","action":"spawn_agent","agent_type":"developer","provider":"fixture","model":"fixture","effort":"medium","prompt":"question prompt\n","prompt_sha256":"d5ae4f5f2c6d4f8d2a0d2f5e2aaf8f4c4f7e7c0b2e7b2b0b4a7f3d6e7d4f0a1b","prompt_bytes":16,"resolved_via":["ROOT-001"]}
    # The runner validates the digest; provide it from the actual prompt.
    import hashlib
    response["prompt_sha256"] = hashlib.sha256(response["prompt"].encode()).hexdigest()
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:2] == ["claim", "TASK-001"]:
    print('{"session_id":"SID-Q"}')
elif args[0] == "question" and args[1] == "create":
    print('{"key":"Q001"}')
elif args[:2] == ["question", "configure-workflow"] or args[0] == "link":
    print('{"ok":true}')
elif args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
cat >/dev/null
printf '%s\n' '{"worker_id":"worker-066","session_id":"SID-Q","kind":"question","category":"scope","question":"Which scope should be used?","why_blocking":"The scope is ambiguous.","recommendation":"Use the smallest scope.","evidence":{}}'
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

set +e
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" \
  "$RUNNER" --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
  --run-id tc066 --root ROOT-001 --scratch-root "$WORKDIR/scratch" --output "$WORKDIR/lifecycle.jsonl" >"$WORKDIR/runner.out" 2>"$WORKDIR/runner.err"
code=$?
set -e
[[ "$code" -eq 1 ]] || { cat "$WORKDIR/runner.err" >&2; fail "question pause exited $code, want 1"; }

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/lifecycle.jsonl" <<'PY'
import json, sys
events = [json.loads(line)["argv"] for line in open(sys.argv[1])]
record = json.loads(open(sys.argv[2]).readline())
assert [argv[0:2] for argv in events] == [
    ["next", "ROOT-001"], ["claim", "TASK-001"],
    ["question", "create"], ["question", "configure-workflow"], ["link", "Q001"], ["release", "TASK-001"],
], events
worker = record["dispatches"][0]["worker"]
assert worker["kind"] == "question"
assert worker["question_key"] == "Q001"
assert record["dispatches"][0]["outcome"] == "pause"
assert record["outcome"]["terminal"] == "pause"
assert record["outcome"]["publication_eligible"] is False
PY

echo "TC-066: pass (question worker result creates, configures, links, and pauses the parent-owned lifecycle)"
