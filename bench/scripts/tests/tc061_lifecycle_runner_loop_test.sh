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
    outcomes: {deep_verify: qa, fail: development, pass: completed}
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
import hashlib, json, os, sys
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
    configured_statuses = os.environ.get("SHARK_STATUSES")
    statuses = (
        {index: status for index, status in enumerate(json.loads(configured_statuses), start=1)}
        if configured_statuses else
        {1: "development", 2: "code_review", 3: "development", 4: "code_review"}
    )
    if next_count in statuses:
        response["status"] = statuses[next_count]
        response["prompt"] = f"run {response['status']} round {next_count}\n"
        encoded = response["prompt"].encode()
        response["prompt_sha256"] = hashlib.sha256(encoded).hexdigest()
        response["prompt_bytes"] = len(encoded)
    else:
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
    rejection_mode = os.environ.get("SHARK_REJECT_ADVANCE", "")
    rejection_marker = os.environ.get("SHARK_REJECTION_MARKER", "")
    should_reject = rejection_mode == "always" or (
        rejection_mode == "first" and rejection_marker and not os.path.exists(rejection_marker)
    )
    if should_reject:
        if rejection_marker:
            open(rejection_marker, "w").close()
        print(json.dumps({"error": True, "code": "COMMAND_ERROR", "message": "artifact validation failed: missing pattern_contract"}), file=sys.stderr)
        raise SystemExit(1)
    print('{"advanced":true}')
elif args[:2] == ["notes", "add"]:
    print('{"note_added":true}')
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
outcome = "pass"
if request["status"] == "code_review" and not os.path.exists(os.environ["ADAPTER_REVIEW_MARKER"]):
    open(os.environ["ADAPTER_REVIEW_MARKER"], "w").close()
    outcome = "fail"
print(json.dumps({
    "worker_id": "worker-061",
    "session_id": request["session_id"],
    "kind": "final",
    "recommended_outcome": outcome,
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
	ADAPTER_REVIEW_MARKER="$WORKDIR/adapter-reviewed" \
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
assert names.count("next") == 5, events
assert names.count("claim") == 4, events
assert names.count("status") == 4, events
assert names.count("release") == 4, events
assert names.count("heartbeat") >= 1, events
assert events[0]["argv"][:4] == ["next", "ROOT-001", "--json", "--prompt-out"]
assert events[1]["argv"][1] == "TASK-002"
heartbeats = [e for e in events if e["argv"][0] == "heartbeat"]
assert all(e["argv"][1] == "TASK-002" for e in heartbeats)
assert len(requests) == 4, requests
for index, request in enumerate(requests, start=1):
    assert request["session_id"] == f"SID-{index:03d}", request
    assert request["prompt"] == f"run {request['status']} round {index}\n"
    assert request["prompt_sha256"] == hashlib.sha256(request["prompt"].encode()).hexdigest()
    assert request["prompt_bytes"] == len(request["prompt"].encode())
    assert not any(k in request for k in ("claim", "heartbeat", "advance", "release"))
assert record["dispatches"][0]["response"]["resolved_via"] == ["ROOT-001"]
assert "prompt" not in record["dispatches"][0]["response"]
assert [item["response"]["status"] for item in record["dispatches"]] == [
    "development", "code_review", "development", "code_review",
], record["dispatches"]
assert [item["outcome"] for item in record["dispatches"]] == ["pass", "fail", "pass", "pass"], record["dispatches"]
assert record["outcome"]["terminal"] == "complete"
assert record["outcome"]["publication_eligible"] is True
assert record["prelude"]["terminal_outcome"] == "not_applicable", record.get("prelude")
assert [item["stage"] for item in record["prelude"]["stages"]] == ["D01", "D02", "D03", "D04", "D05"]
assert record["workflow_policy"]["enabled_gates"] == ["code_review", "qa"], record["workflow_policy"]
assert record["workflow_policy"]["gate_order"] == ["code_review", "qa"], record["workflow_policy"]
policies = record["workflow_policy"]["gate_policies"]
assert len(policies) == 4, record["workflow_policy"]
assert [item["gate_id"] for item in policies] == ["code_review", "qa", "code_review", "code_review"], policies
assert len({item["policy_digest"] for item in policies}) == 4, policies
assert len(record["review_gates"]) == 3, record["review_gates"]
assert [gate["gate_id"] for gate in record["review_gates"]] == ["code_review", "code_review", "qa"], record["review_gates"]
assert [gate["round"] for gate in record["review_gates"][:2]] == [1, 2], record["review_gates"]
assert all(gate["state"] == "zero_findings" for gate in record["review_gates"][:2]), record["review_gates"]
assert record["review_gates"][2]["state"] == "not_reached", record["review_gates"]
policy_digests = {item["policy_digest"] for item in policies}
assert all(gate["policy_ref"]["policy_digest"] in policy_digests for gate in record["review_gates"]), record["review_gates"]
assert record["review_gates"][0]["policy_ref"] != record["review_gates"][1]["policy_ref"], record["review_gates"]
bundle = json.load(open(os.path.join(os.path.dirname(sys.argv[3]), "evidence", "bundle.json")))
assert [stage["dispatch_ordinal"] for stage in bundle["stages"]] == [1, 2, 3, 4], bundle
assert bundle["dispatches"] == record["dispatches"]
assert bundle["prelude"] == record["prelude"]
for stage_index, stage in enumerate(record["stages"]):
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
        "agent_visible_input",
    }, stage
    expected_prior = sorted(
        (artifact["path"], artifact["digest"])
        for prior_stage in record["stages"][:stage_index]
        for artifact in prior_stage["artifacts"]
    )
    observed_prior = sorted(
        (item["path"], item["digest"])
        for item in stage["input_lineage"]
        if item["source_kind"] == "prior_stage_artifact"
    )
    assert observed_prior == expected_prior, stage
    for artifact in stage["artifacts"]:
        expected_consumers = sorted(
            later["stage"] for later in record["stages"][stage_index + 1:]
        )
        observed_consumers = sorted(
            edge["consuming_stage"] for edge in artifact["consumers"]
        )
        assert observed_consumers == expected_consumers, artifact
        assert all(
            set(edge) == {"consuming_stage", "edge_kind", "observed_at"}
            and edge["edge_kind"] == "read" and edge["observed_at"].strip()
            for edge in artifact["consumers"]
        ), artifact
        snapshot_entry = next(
            item for item in bundle["stages"]
            if item["dispatch_ordinal"] == stage["dispatch_ordinal"]
        )
        snapshot = json.load(open(os.path.join(
            os.path.dirname(sys.argv[3]), "evidence", snapshot_entry["snapshot_path"],
        )))
        snapshot_artifact = next(
            item for item in snapshot["artifacts"]
            if (item["path"], item["digest"]) == (artifact["path"], artifact["digest"])
        )
        assert snapshot_artifact["consumers"] == artifact["consumers"], (snapshot_artifact, artifact)
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
assert sorted(os.listdir(transcript_dir)) == [
    "0001-development.json", "0002-code_review.json",
    "0003-development.json", "0004-code_review.json",
]
PY

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/evidence" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/lifecycle.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >/dev/null
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR/missing-prior-lineage.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
record["stages"][1]["input_lineage"] = [
    entry for entry in record["stages"][1]["input_lineage"]
    if entry["source_kind"] != "prior_stage_artifact"
]
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY
if "$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/missing-prior-lineage.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >"$WORKDIR/missing-prior.out" 2>"$WORKDIR/missing-prior.err"; then
	fail "verifier accepted a later stage with missing prior-stage artifact lineage"
fi
grep -q "prior-stage artifact lineage" "$WORKDIR/missing-prior.err" || \
	fail "verifier did not name missing prior-stage artifact lineage"
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR/missing-consumer.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
record["stages"][0]["artifacts"][0]["consumers"] = []
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY
if "$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/missing-consumer.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >"$WORKDIR/missing-consumer.out" 2>"$WORKDIR/missing-consumer.err"; then
	fail "verifier accepted lineage whose producer still marked the artifact orphaned"
fi
grep -q "consumer graph disagrees" "$WORKDIR/missing-consumer.err" || \
	fail "verifier did not name the artifact-consumer graph contradiction"
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR/whitespace-consumer.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
record["stages"][0]["artifacts"][0]["consumers"][0]["observed_at"] = "   "
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY
if "$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/whitespace-consumer.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >"$WORKDIR/whitespace-consumer.out" 2>"$WORKDIR/whitespace-consumer.err"; then
	fail "verifier accepted a typed consumer edge with whitespace-only observed_at"
fi
grep -q "consumer fields must be non-empty strings" "$WORKDIR/whitespace-consumer.err" || \
	fail "verifier did not name the whitespace-only typed consumer field"
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR" <<'PY'
import copy, json, os, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
mutations = {
    "missing-observed-at": lambda edge: edge.pop("observed_at"),
    "unknown-edge-kind": lambda edge: edge.__setitem__("edge_kind", "invented"),
    "unexpected-edge-field": lambda edge: edge.__setitem__("unexpected", True),
}
for name, mutate in mutations.items():
    variant = copy.deepcopy(record)
    mutate(variant["stages"][0]["artifacts"][0]["consumers"][0])
    with open(os.path.join(sys.argv[2], f"{name}.jsonl"), "w", encoding="utf-8") as stream:
        stream.write(json.dumps(variant, separators=(",", ":")) + "\n")
PY
for invalid_edge_case in missing-observed-at unknown-edge-kind unexpected-edge-field; do
	if "$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/$invalid_edge_case.jsonl" \
	    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >"$WORKDIR/$invalid_edge_case.out" 2>"$WORKDIR/$invalid_edge_case.err"; then
		fail "verifier accepted malformed typed consumer edge: $invalid_edge_case"
	fi
done
grep -q "must contain only consuming_stage, edge_kind, and observed_at" "$WORKDIR/missing-observed-at.err" || \
	fail "verifier did not name the missing typed consumer field"
grep -q "edge_kind is not in the I-05 vocabulary" "$WORKDIR/unknown-edge-kind.err" || \
	fail "verifier did not reject the unknown typed consumer edge_kind"
grep -q "must contain only consuming_stage, edge_kind, and observed_at" "$WORKDIR/unexpected-edge-field.err" || \
	fail "verifier did not reject the unexpected typed consumer field"
python3 - "$WORKDIR/lifecycle.jsonl" "$WORKDIR/deduplicated-consumers.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
consumers = record["stages"][0]["artifacts"][0]["consumers"]
seen = set()
record["stages"][0]["artifacts"][0]["consumers"] = [
    edge for edge in consumers
    if edge["consuming_stage"] not in seen and not seen.add(edge["consuming_stage"])
]
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    stream.write(json.dumps(record, separators=(",", ":")) + "\n")
PY
if "$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/deduplicated-consumers.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >"$WORKDIR/deduplicated-consumers.out" 2>"$WORKDIR/deduplicated-consumers.err"; then
	fail "verifier accepted consumer edges collapsed to unique stage names"
fi
grep -q "consumer graph disagrees" "$WORKDIR/deduplicated-consumers.err" || \
	fail "verifier did not detect lost repeated-dispatch consumer multiplicity"

# A worker can recommend pass while Shark correctly rejects the transition
# because the produced artifact fails its phase validator. The controller must
# retain that attempt, record the rejection as a note, and redispatch the same
# stage against the still-present scratch artifact instead of terminating the
# whole benchmark as worker_failure.
mkdir -p "$WORKDIR/retry-scratch/shark-data/workflow" "$WORKDIR/retry-scratch/shark-data/prompts/_shared"
cp "$WORKDIR/scratch/shark-data/workflow/bug.yaml" "$WORKDIR/retry-scratch/shark-data/workflow/bug.yaml"
cp "$WORKDIR/scratch/shark-data/prompts/_shared/"*.md "$WORKDIR/retry-scratch/shark-data/prompts/_shared/"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/retry-fixture"
: >"$WORKDIR/retry-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/retry-events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/retry-next-count" \
SHARK_STATUSES='["development","development"]' SHARK_REJECT_ADVANCE=first SHARK_REJECTION_MARKER="$WORKDIR/rejection-seen" \
ADAPTER_REQUEST="$WORKDIR/retry-requests.ndjson" ADAPTER_MUTATION_MARKER="$WORKDIR/retry-adapter-mutated" \
ADAPTER_REVIEW_MARKER="$WORKDIR/retry-adapter-reviewed" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc061-transition-retry --root ROOT-001 --scratch-root "$WORKDIR/retry-scratch" \
    --fixture-root "$WORKDIR/retry-fixture" \
    --evidence-root "$WORKDIR/retry-evidence" --output "$WORKDIR/retry-lifecycle.jsonl" >/dev/null
python3 - "$WORKDIR/retry-events.ndjson" "$WORKDIR/retry-lifecycle.jsonl" <<'PY'
import json, sys
events = [json.loads(line) for line in open(sys.argv[1], encoding="utf-8")]
record = json.load(open(sys.argv[2], encoding="utf-8"))
assert [stage["stage"] for stage in record["stages"]] == ["development", "development"], record
assert [stage["rework"] for stage in record["stages"]] == [False, True], record
assert [dispatch["outcome"] for dispatch in record["dispatches"]] == ["fail", "pass"], record
rejected = record["dispatches"][0]["transition"]
assert rejected["accepted"] is False and rejected["retry_scheduled"] is True, rejected
assert rejected["rejection_attempt"] == 1 and "missing pattern_contract" in rejected["rejection"], rejected
assert record["outcome"]["terminal"] == "complete" and record["outcome"]["publication_eligible"] is True, record
note_events = [event for event in events if event["argv"][:2] == ["notes", "add"]]
assert len(note_events) == 1 and "missing pattern_contract" in " ".join(note_events[0]["argv"]), note_events
assert sum(event["argv"][:2] == ["status", "advance"] for event in events) == 2, events
PY
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/retry-evidence" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/retry-lifecycle.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >/dev/null

# The recovery is bounded independently of provider cost/time ceilings. Three
# consecutive transition rejections (initial attempt plus two retries) stop as
# worker_failure and retain every attempt, rather than spending indefinitely.
mkdir -p "$WORKDIR/exhausted-scratch/shark-data/workflow" "$WORKDIR/exhausted-scratch/shark-data/prompts/_shared"
cp "$WORKDIR/scratch/shark-data/workflow/bug.yaml" "$WORKDIR/exhausted-scratch/shark-data/workflow/bug.yaml"
cp "$WORKDIR/scratch/shark-data/prompts/_shared/"*.md "$WORKDIR/exhausted-scratch/shark-data/prompts/_shared/"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/exhausted-fixture"
: >"$WORKDIR/exhausted-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/exhausted-events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/exhausted-next-count" \
SHARK_STATUSES='["development","development","development","development"]' SHARK_REJECT_ADVANCE=always \
SHARK_REJECTION_MARKER="$WORKDIR/exhausted-rejection-seen" ADAPTER_REQUEST="$WORKDIR/exhausted-requests.ndjson" \
ADAPTER_MUTATION_MARKER="$WORKDIR/exhausted-adapter-mutated" ADAPTER_REVIEW_MARKER="$WORKDIR/exhausted-adapter-reviewed" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
    --run-id tc061-transition-exhausted --root ROOT-001 --scratch-root "$WORKDIR/exhausted-scratch" \
    --fixture-root "$WORKDIR/exhausted-fixture" \
    --evidence-root "$WORKDIR/exhausted-evidence" --output "$WORKDIR/exhausted-lifecycle.jsonl" >/dev/null
python3 - "$WORKDIR/exhausted-events.ndjson" "$WORKDIR/exhausted-lifecycle.jsonl" <<'PY'
import json, sys
events = [json.loads(line) for line in open(sys.argv[1], encoding="utf-8")]
record = json.load(open(sys.argv[2], encoding="utf-8"))
assert len(record["dispatches"]) == 3 and len(record["stages"]) == 3, record
assert [item["outcome"] for item in record["dispatches"]] == ["fail", "fail", "worker_failure"], record
assert [item["transition"]["rejection_attempt"] for item in record["dispatches"]] == [1, 2, 3], record
assert record["dispatches"][-1]["transition"]["retry_scheduled"] is False, record
assert record["outcome"]["terminal"] == "worker_failure" and record["outcome"]["publication_eligible"] is False, record
assert "rejected 3 times" in record["outcome"]["reason"], record
assert sum(event["argv"][:2] == ["notes", "add"] for event in events) == 3, events
assert sum(event["argv"][:2] == ["status", "advance"] for event in events) == 3, events
PY
"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/exhausted-evidence" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/exhausted-lifecycle.jsonl" \
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
	ADAPTER_REVIEW_MARKER="$WORKDIR/partial-adapter-reviewed" \
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
