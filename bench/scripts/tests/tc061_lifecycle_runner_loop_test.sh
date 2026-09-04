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
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch/shark-data/workflow" "$WORKDIR/scratch/shark-data/prompts/_shared" "$WORKDIR/i05"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/fixture"
# py-bug-due-date-boundary's fixture.base_sha is frozen ahead of the
# submodule's current tip (T-E40-F05-014); the clone must be pinned there so
# verify-evidence-roots's fixture_checkout HEAD check passes.
git -C "$WORKDIR/fixture" checkout -q 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9

# main's run-lifecycle.sh resolves fixture_root from the scenario package's
# own fixture.submodule_path (no --fixture-root override), so isolating the
# candidate-identity mutation below to a private clone -- instead of the
# real, shared bench/fixture-py checkout -- means pointing a private copy of
# the scenario package at this clone (absolute path: submodule_path resolves
# against the caller's cwd, not this file's directory). The whole package
# directory is copied, not just package.yaml, so evaluator_only's own
# evaluator-root-relative paths (e.g. evaluator/reference.patch) still
# resolve.
cp -r "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary" "$WORKDIR/scenario"
SCENARIO="$WORKDIR/scenario/package.yaml"
sed -i "s#^  submodule_path: .*#  submodule_path: $WORKDIR/fixture#" "$SCENARIO"

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
    statuses = {
        1: "development",
        2: "code_review",
        3: "development",
        4: "code_review",
    }
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
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
elif args[:3] == ["admin", "workflow", "list"]:
    level = args[3]
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], level + ".json")) as f:
        print(f.read())
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
    "input_tokens": 10,
    "output_tokens": 5,
    "cache_read_input_tokens": 0,
    "cache_creation_input_tokens": 0,
}
outcome = "pass"
if request["status"] == "code_review" and not os.path.exists(os.environ["ADAPTER_REVIEW_MARKER"]):
    open(os.environ["ADAPTER_REVIEW_MARKER"], "w").close()
    outcome = "fail"
envelope = {
    "worker_id": "worker-061",
    "session_id": request["session_id"],
    "kind": "final",
    "recommended_outcome": outcome,
    "evidence": {"summary": "fixture complete"},
    "cost_usd": 0.0,
    "total_cost_usd": 0.0,
    "duration_api_ms": 20,
    "num_turns": 1,
    "modelUsage": {"fixture-model": {}},
    "usage": usage,
}
if os.environ.get("PARTIAL_USAGE") == "1":
    envelope["usage"] = {"input_tokens": 10}
    del envelope["total_cost_usd"]
    del envelope["duration_api_ms"]
    del envelope["num_turns"]
    del envelope["modelUsage"]
print(json.dumps(envelope, separators=(",", ":")))
PY
ADAPTER
chmod +x "$WORKDIR/adapter.sh"

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/next-count" ADAPTER_REQUEST="$WORKDIR/requests.ndjson" \
SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
ADAPTER_MUTATION_MARKER="$WORKDIR/adapter-mutated" \
ADAPTER_REVIEW_MARKER="$WORKDIR/adapter-reviewed" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$SCENARIO" \
    --run-id tc061 --root ROOT-001 --scratch-root "$WORKDIR/scratch" \
    --i05-bundle-dir "$WORKDIR/i05" \
    --output "$WORKDIR/lifecycle.jsonl" >/dev/null

python3 - "$WORKDIR/events.ndjson" "$WORKDIR/requests.ndjson" "$WORKDIR/lifecycle.jsonl" <<'PY'
import hashlib, json, os, sys
events = [json.loads(line) for line in open(sys.argv[1])]
requests = [json.loads(line) for line in open(sys.argv[2])]
record = json.loads(open(sys.argv[3]).readline())
names = [e["argv"][0] for e in events]
# T-E40-F12-002: record_stage()'s own phase lookup (`admin workflow list`)
# runs after the dispatch's claim/heartbeat/status/release sequence
# completes, and capture_review_gate()'s own note lookup (`get TASK-002`)
# runs after that for a code_review/qa dispatch -- both legitimately append
# trailing events here, excluded from this test's own claim/heartbeat/
# transition/release ordering assertion, which is what TC-061 actually
# covers.
non_i05_events = [e for e in events if e["argv"][0] not in ("admin", "get")]
non_i05_names = [e["argv"][0] for e in non_i05_events]
assert non_i05_names[0:2] == ["next", "claim"], events
# The loop's final "next" resolves the archive action (queue drains without
# a fifth real dispatch), so the fourth (code_review) dispatch's own
# status/release is followed by exactly this one trailing lookup, not the
# loop's own last real-dispatch event.
assert non_i05_names[-3:] == ["status", "release", "next"], events
assert names.count("next") == 5, events
assert names.count("claim") == 4, events
assert names.count("status") == 4, events
assert names.count("release") == 4, events
assert names.count("heartbeat") >= 1, events
assert events[0]["argv"][:4] == ["next", "ROOT-001", "--json", "--prompt-out"]
assert events[1]["argv"][1] == "TASK-002"

def session_of(argv):
    return argv[argv.index("--session") + 1]

heartbeats = [e for e in events if e["argv"][0] == "heartbeat"]
assert all(e["argv"][1] == "TASK-002" for e in heartbeats), heartbeats
heartbeat_sessions = {session_of(e["argv"]) for e in heartbeats}
assert heartbeat_sessions <= {f"SID-{index:03d}" for index in range(1, 5)}, heartbeats
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
i05_dir = os.path.join(os.path.dirname(sys.argv[3]), "i05")
bundle = json.load(open(os.path.join(i05_dir, "bundle.json")))
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
first = record["stages"][0]
manifest = {entry["path"]: entry for entry in first["candidate"]["dirty_untracked_manifest"]}
assert manifest["taskmanager/due_date.py"]["tracked"] is True, first
assert manifest["taskmanager/due_date.py"]["digest"].startswith("sha256:"), first
assert manifest["taskmanager/__pycache__/provider.cache"]["tracked"] is False, first
# I05BundleWriter.record_stage()'s own "candidate" field (code/review
# categories only) tracks Shark's OWN *_test.go corpus for the harness's
# own replay identity (REQ-F-006) -- a different test-suite concept from
# I-07's own stage["candidate"]["test_suite_ids"] above, which is the
# scenario's Python fixture suite via candidate_identity(). Not equal by
# design; only presence is checked here.
stage_snapshot = json.load(open(os.path.join(i05_dir, "stages", "1-development.json")))
assert stage_snapshot["candidate"]["test_suite_ids"], stage_snapshot
assert stage_snapshot["candidate"]["test_suite_dir"], stage_snapshot
transcript_dir = os.path.join(i05_dir, "transcripts")
assert sorted(os.listdir(transcript_dir)) == [
    "1-development.txt", "2-code_review.txt",
    "3-development.txt", "4-code_review.txt",
]
PY

"$SCRIPTS_DIR/verify-stage-evidence.sh" "$WORKDIR/i05" >/dev/null
"$SCRIPTS_DIR/verify-lifecycle-run.sh" "$WORKDIR/lifecycle.jsonl" \
    --schema "$SCRIPTS_DIR/../runs/i07-schema.yaml" >/dev/null
# Not replay-stage-evidence.sh here: its test-suite-drift check assumes
# candidate.test_suite_ids is the scenario's own adapter-enumerated test
# corpus (the now-deleted write_stage_evidence()'s source). I05BundleWriter.
# record_stage() deliberately populates that field from test_suite_reference()
# -- Shark's own *_test.go corpus (REQ-F-006) -- for a different purpose, so
# diffing it against `<python-adapter> test --checkout <fixture>` would
# always show total, spurious drift. tc049 already covers replay-stage-
# evidence.sh's real contract against a snapshot it actually applies to.
#
# Not the "missing prior-stage artifact lineage is rejected" negative test
# either: every stage's own `artifacts` stays the honest, always-empty
# placeholder until real artifact population lands (still E40-F07's own
# scope per record_stage()'s docstring -- see port-list), so there is no
# real prior_stage_artifact entry to strip from any stage's input_lineage
# yet; stripping nothing from an already-empty list is not a genuine
# negative case and would never actually exercise
# verify-lifecycle-run.sh's new check.

# A mechanically complete workflow with incomplete provider usage must fail
# closed instead of becoming publication eligible.
mkdir -p "$WORKDIR/partial-scratch" "$WORKDIR/partial-i05"
cp -a "$WORKDIR/scratch/shark-data" "$WORKDIR/partial-scratch/shark-data"
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/partial-fixture"
git -C "$WORKDIR/partial-fixture" checkout -q 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9
cp -r "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary" "$WORKDIR/partial-scenario"
PARTIAL_SCENARIO="$WORKDIR/partial-scenario/package.yaml"
sed -i "s#^  submodule_path: .*#  submodule_path: $WORKDIR/partial-fixture#" "$PARTIAL_SCENARIO"
: >"$WORKDIR/partial-events.ndjson"
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/partial-events.ndjson" \
SHARK_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json" SHARK_STATE="$WORKDIR/partial-next-count" ADAPTER_REQUEST="$WORKDIR/partial-requests.ndjson" \
	ADAPTER_MUTATION_MARKER="$WORKDIR/partial-adapter-mutated" PARTIAL_USAGE=1 \
	SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
	ADAPTER_REVIEW_MARKER="$WORKDIR/partial-adapter-reviewed" \
LIFECYCLE_ADAPTER="$WORKDIR/adapter.sh" LIFECYCLE_HEARTBEAT_INTERVAL_SECONDS=0.01 "$RUNNER" \
    --scenario "$PARTIAL_SCENARIO" \
	--run-id tc061-partial --root ROOT-001 --scratch-root "$WORKDIR/partial-scratch" \
	--i05-bundle-dir "$WORKDIR/partial-i05" --output "$WORKDIR/partial-lifecycle.jsonl" >/dev/null
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
