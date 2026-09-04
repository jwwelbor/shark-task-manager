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
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch" "$WORKDIR/i05"

cat >"$WORKDIR/bin/python3" <<'PYTHON'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${HANG_TEST_DISCOVERY:-}" == "1" && "$PWD" == */e40-test-identity-*/checkout && "${1:-}" == "-m" && "${2:-}" == "pytest" ]]; then
	/usr/bin/python3 - "$CHILD_PID" "$CHILD_HEARTBEAT" <<'PY' &
import os, pathlib, sys, time
pathlib.Path(sys.argv[1]).write_text(str(os.getpid()))
while True:
    pathlib.Path(sys.argv[2]).touch()
    time.sleep(0.02)
PY
	wait
fi
exec /usr/bin/python3 "$@"
PYTHON
chmod +x "$WORKDIR/bin/python3"

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
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], args[3] + ".json")) as f:
        print(f.read())
else: raise SystemExit("unexpected shark argv: " + repr(args))
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
if [[ "${HANG_ADAPTER:-}" == "1" ]]; then
	python3 - "$CHILD_PID" "$CHILD_HEARTBEAT" <<'PY' &
import os, pathlib, sys, time
pathlib.Path(sys.argv[1]).write_text(str(os.getpid()))
while True:
    pathlib.Path(sys.argv[2]).touch()
    time.sleep(0.02)
PY
	wait
fi
python3 -c 'import json,sys; request=json.load(sys.stdin); cost=0.01 if request["entity_key"] == "TASK-001" else 0.0; print(json.dumps({"worker_id":"worker-062","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":cost,"evidence":{"summary":"fixture"}}))'
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

cp "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" "$WORKDIR/not-admitted.yaml"
python3 - "$WORKDIR/not-admitted.yaml" <<'PY'
import sys
import yaml

path = sys.argv[1]
with open(path) as stream:
    data = yaml.safe_load(stream)
data.pop("admission", None)
with open(path, "w") as stream:
    yaml.safe_dump(data, stream, sort_keys=False)
PY
if PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$WORKDIR/not-admitted.yaml" --run-id tc062-not-admitted --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
    --mode contract >/dev/null 2>"$WORKDIR/not-admitted.err"; then
    fail "scenario without admission.status unexpectedly ran"
fi
grep -q "scenario package is not admitted" "$WORKDIR/not-admitted.err" || fail "missing admission status was not rejected"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
    --i05-bundle-dir "$WORKDIR/i05" \
    --limits "$SCRIPTS_DIR/testdata/lifecycle/limits/first-exceed.yaml" --output "$WORKDIR/lifecycle.jsonl"

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/lifecycle.jsonl" "$WORKDIR/i05/bundle.json" <<'PY'
import json, sys
events = [json.loads(line) for line in open(sys.argv[1])]
record = json.loads(open(sys.argv[2]).readline())
bundle = json.load(open(sys.argv[3]))
assert [e["argv"][1] for e in events if e["argv"][0] == "next"] == ["ROOT-001", "TASK-001"]
assert record["limits"]["observed_generated_tasks"] == 1
assert record["limits"]["first_exceeded"] == "max_cost_usd"
assert record["outcome"]["terminal"] == "resource_limit"
assert record["outcome"]["partial_evidence"] is True
assert record["outcome"]["publication_eligible"] is False
assert record["outcome"]["reason"]
assert bundle["terminal_status"]["reached"] is True
assert bundle["stop_outcome"] == "resource_limit"
assert sum(e["argv"][0] == "release" for e in events) == 1
assert not any(e["argv"][0] == "next" and e["argv"][1] == "TASK-002" for e in events)
PY

echo "TC-062: pass (first exceeded ceiling stops scenario and retains partial evidence)"

echo "TC-062: in-flight wall deadline terminates the adapter process group and retains stop evidence"
cat >"$WORKDIR/deadline-limits.yaml" <<'YAML'
max_cost_usd: 10
max_wall_clock_seconds: 1.5
max_generated_tasks: 10
YAML
mkdir -p "$WORKDIR/deadline-scratch"
: >"$WORKDIR/deadline-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/deadline-events.ndjson" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
HANG_ADAPTER=1 CHILD_PID="$WORKDIR/child.pid" CHILD_HEARTBEAT="$WORKDIR/child.heartbeat" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062-deadline --root ROOT-001 --scratch-root "$WORKDIR/deadline-scratch" \
    --limits "$WORKDIR/deadline-limits.yaml" --i05-bundle-dir "$WORKDIR/deadline-i05" \
    --output "$WORKDIR/deadline.jsonl"

[[ -s "$WORKDIR/child.pid" && -e "$WORKDIR/child.heartbeat" ]] || fail "deadline adapter child never started"
child_pid="$(cat "$WORKDIR/child.pid")"
mtime_before="$(stat -c %Y.%y "$WORKDIR/child.heartbeat")"
sleep 0.1
mtime_after="$(stat -c %Y.%y "$WORKDIR/child.heartbeat")"
[[ "$mtime_before" == "$mtime_after" ]] || fail "adapter descendant remained active after deadline"
if kill -0 "$child_pid" 2>/dev/null; then
	fail "adapter descendant PID $child_pid survived process-group termination"
fi
python3 - "$WORKDIR/deadline.jsonl" "$WORKDIR/deadline-i05/bundle.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
bundle = json.load(open(sys.argv[2], encoding="utf-8"))
assert record["outcome"]["terminal"] == "resource_limit", record["outcome"]
assert record["limits"]["first_exceeded"] == "max_wall_clock_seconds", record["limits"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
assert record["dispatches"][0]["release"], record["dispatches"][0]
assert bundle["stop_outcome"] == "resource_limit", bundle
assert bundle["publication_eligible"] is False, bundle
PY
echo "TC-062: pass (deadline kills adapter descendants and emits retained resource_limit evidence)"

echo "TC-062: SIGTERM terminates the adapter process group and retains cancellation evidence"
cat >"$WORKDIR/signal-limits.yaml" <<'YAML'
max_cost_usd: 10
max_wall_clock_seconds: 30
max_generated_tasks: 10
YAML
mkdir -p "$WORKDIR/signal-scratch"
: >"$WORKDIR/signal-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/signal-events.ndjson" \
HANG_ADAPTER=1 CHILD_PID="$WORKDIR/signal-child.pid" CHILD_HEARTBEAT="$WORKDIR/signal-child.heartbeat" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062-signal --root ROOT-001 --scratch-root "$WORKDIR/signal-scratch" \
    --limits "$WORKDIR/signal-limits.yaml" --i05-bundle-dir "$WORKDIR/signal-i05" \
    --output "$WORKDIR/signal.jsonl" >/dev/null &
runner_pid=$!
for _ in $(seq 1 500); do
	[[ -s "$WORKDIR/signal-child.pid" && -e "$WORKDIR/signal-child.heartbeat" ]] && break
	sleep 0.02
done
[[ -s "$WORKDIR/signal-child.pid" && -e "$WORKDIR/signal-child.heartbeat" ]] || fail "signal adapter child never started"
signal_child_pid="$(cat "$WORKDIR/signal-child.pid")"
kill -TERM "$runner_pid"
set +e
wait "$runner_pid"
runner_status=$?
set -e
[[ "$runner_status" -eq 0 ]] || fail "signal stop exited $runner_status instead of retaining a named stop"
if kill -0 "$signal_child_pid" 2>/dev/null; then
	fail "adapter descendant PID $signal_child_pid survived SIGTERM handling"
fi
python3 - "$WORKDIR/signal.jsonl" "$WORKDIR/signal-i05/bundle.json" "$WORKDIR/signal-events.ndjson" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
bundle = json.load(open(sys.argv[2], encoding="utf-8"))
events = [json.loads(line) for line in open(sys.argv[3], encoding="utf-8")]
assert record["outcome"]["terminal"] == "cancellation", record["outcome"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
assert record["dispatches"][0]["release"], record["dispatches"][0]
assert bundle["stop_outcome"] == "cancellation", bundle
assert sum(event["argv"][0] == "release" for event in events) == 1, events
PY
echo "TC-062: pass (SIGTERM kills adapter descendants and emits retained cancellation evidence)"

echo "TC-062: SIGTERM also terminates post-dispatch test-discovery descendants"
mkdir -p "$WORKDIR/discovery-scratch"
: >"$WORKDIR/discovery-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/discovery-events.ndjson" \
HANG_TEST_DISCOVERY=1 CHILD_PID="$WORKDIR/discovery-child.pid" CHILD_HEARTBEAT="$WORKDIR/discovery-child.heartbeat" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062-discovery-signal --root ROOT-001 --scratch-root "$WORKDIR/discovery-scratch" \
    --limits "$WORKDIR/signal-limits.yaml" --i05-bundle-dir "$WORKDIR/discovery-i05" \
    --output "$WORKDIR/discovery.jsonl" >/dev/null &
discovery_runner_pid=$!
for _ in $(seq 1 500); do
	[[ -s "$WORKDIR/discovery-child.pid" && -e "$WORKDIR/discovery-child.heartbeat" ]] && break
	sleep 0.02
done
[[ -s "$WORKDIR/discovery-child.pid" && -e "$WORKDIR/discovery-child.heartbeat" ]] || fail "test-discovery child never started"
discovery_child_pid="$(cat "$WORKDIR/discovery-child.pid")"
kill -TERM "$discovery_runner_pid"
set +e
wait "$discovery_runner_pid"
discovery_runner_status=$?
set -e
[[ "$discovery_runner_status" -eq 0 ]] || fail "test-discovery signal stop exited $discovery_runner_status instead of retaining a named stop"
if kill -0 "$discovery_child_pid" 2>/dev/null; then
	fail "test-discovery descendant PID $discovery_child_pid survived SIGTERM handling"
fi
python3 - "$WORKDIR/discovery.jsonl" "$WORKDIR/discovery-i05/bundle.json" "$WORKDIR/discovery-events.ndjson" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
bundle = json.load(open(sys.argv[2], encoding="utf-8"))
events = [json.loads(line) for line in open(sys.argv[3], encoding="utf-8")]
assert record["outcome"]["terminal"] == "cancellation", record["outcome"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
# The candidate-identity snapshot taken at the top of each loop iteration
# (before `shark claim` is ever called) also runs real test discovery, so a
# hang there is cancelled before any dispatch/claim exists at all -- unlike
# the worker-adapter hang above, there is no dispatch or claim/release cycle
# to retain here.
assert record["dispatches"] == [], record["dispatches"]
assert record["stages"] == [], record["stages"]
assert bundle["stop_outcome"] == "cancellation", bundle
assert bundle["publication_eligible"] is False, bundle
assert sum(event["argv"][0] == "release" for event in events) == 0, events
PY
echo "TC-062: pass (SIGTERM kills test-discovery descendants and emits retained cancellation evidence)"

echo "TC-062: wall deadline also terminates post-dispatch test-discovery descendants"
mkdir -p "$WORKDIR/discovery-deadline-scratch"
: >"$WORKDIR/discovery-deadline-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/discovery-deadline-events.ndjson" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
HANG_TEST_DISCOVERY=1 CHILD_PID="$WORKDIR/discovery-deadline-child.pid" CHILD_HEARTBEAT="$WORKDIR/discovery-deadline-child.heartbeat" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc062-discovery-deadline --root ROOT-001 --scratch-root "$WORKDIR/discovery-deadline-scratch" \
    --limits "$WORKDIR/deadline-limits.yaml" --i05-bundle-dir "$WORKDIR/discovery-deadline-i05" \
    --output "$WORKDIR/discovery-deadline.jsonl" >/dev/null
[[ -s "$WORKDIR/discovery-deadline-child.pid" && -e "$WORKDIR/discovery-deadline-child.heartbeat" ]] || \
	fail "deadline test-discovery child never started"
discovery_deadline_child_pid="$(cat "$WORKDIR/discovery-deadline-child.pid")"
if kill -0 "$discovery_deadline_child_pid" 2>/dev/null; then
	fail "test-discovery descendant PID $discovery_deadline_child_pid survived wall deadline"
fi
python3 - "$WORKDIR/discovery-deadline.jsonl" "$WORKDIR/discovery-deadline-i05/bundle.json" "$WORKDIR/discovery-deadline-events.ndjson" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
bundle = json.load(open(sys.argv[2], encoding="utf-8"))
events = [json.loads(line) for line in open(sys.argv[3], encoding="utf-8")]
assert record["outcome"]["terminal"] == "resource_limit", record["outcome"]
assert record["limits"]["first_exceeded"] == "max_wall_clock_seconds", record["limits"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
assert record["dispatches"][0]["release"], record["dispatches"][0]
assert any(
    error["kind"] == "test_suite_unavailable"
    for stage in record["stages"] for error in stage["errors"]
), record["stages"]
assert bundle["stop_outcome"] == "resource_limit", bundle
assert bundle["publication_eligible"] is False, bundle
assert sum(event["argv"][0] == "release" for event in events) == 1, events
PY
echo "TC-062: pass (deadline kills test-discovery descendants and emits retained resource_limit evidence)"
