#!/usr/bin/env bash
# TC-065 / T-E40-F08-004: pre-dispatch prelude, Question, and I-05 isolation gate.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PRELUDE="$SCRIPTS_DIR/lifecycle-prelude.sh"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
fail() { echo "TC-065 FAIL: $1" >&2; exit 1; }
[[ -x "$PRELUDE" ]] || fail "lifecycle-prelude.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch" "$WORKDIR/evaluator"
# verify-evidence-roots requires a real git checkout at package-feature.yaml's
# own frozen fixture.base_sha below -- a plain mkdir'd directory fails its
# fixture_checkout HEAD check.
git clone -q "$SCRIPTS_DIR/../fixture-py" "$WORKDIR/fixture"
git -C "$WORKDIR/fixture" checkout -q 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9
mkdir -p "$WORKDIR/evaluator/evaluator"
printf 'fixture reference\n' >"$WORKDIR/evaluator/evaluator/reference.patch"
printf 'fixture reference\n' >"$WORKDIR/evaluator/reference.patch"
mkdir -p "$WORKDIR/evaluator/replay"
cat >"$WORKDIR/evaluator/replay/bundle.json" <<'JSON'
{"bundle_version":"1.0.0","scenario_binding":{"scenario_id":"tc065-feature","scenario_version":1},"entries":[]}
JSON

cat >"$WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
with open(os.environ["SHARK_EVENTS"], "a", encoding="utf-8") as stream:
    stream.write(json.dumps({"argv": args}, separators=(",", ":")) + "\n")
if args[:2] == ["next", "Q-E40-F08-001"]:
    print(json.dumps({"question_block": {"question_key": "Q-E40-F08-001", "current_responder": "responder-a"}}))
elif args[:2] == ["next", "Q-E40-F08-002"]:
    print(json.dumps({"question_block": None}))
elif args[:2] == ["next", "ROOT-LINEAGE"]:
    prompt = "capture replay lineage\n"
    path = args[args.index("--prompt-out") + 1]
    open(path, "w", encoding="utf-8").write(prompt)
    import hashlib
    print(json.dumps({"entity_key": "FEATURE-001", "entity_type": "feature", "status": "research",
                      "action": "spawn_agent", "agent_type": "researcher", "provider": "fixture",
                      "model": "fixture-model", "effort": "low", "prompt": prompt,
                      "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(), "prompt_bytes": len(prompt),
                      "resolved_via": ["ROOT-LINEAGE"], "unresolved_placeholders": [], "error": ""}))
elif args[:2] == ["next", "FEATURE-001"]:
    print(json.dumps({"action": "archive", "entity_key": "FEATURE-001", "entity_type": "feature"}))
elif args[:2] == ["claim", "Q-E40-F08-001"]:
    print('{"session_id":"SID-Q"}')
elif args[:2] == ["claim", "FEATURE-001"]:
    print('{"session_id":"SID-LINEAGE"}')
elif args[:2] == ["question", "respond"] or args[:2] == ["question", "resolve"]:
    print('{"ok":true}')
elif args and args[0] in ("heartbeat", "release"):
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args[:3] == ["admin", "workflow", "list"]:
    with open(os.path.join(os.environ["SHARK_WORKFLOW_DIR"], args[3] + ".json"), encoding="utf-8") as stream:
        print(stream.read())
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

cat >"$WORKDIR/package-feature.yaml" <<'YAML'
schema_version: "1.0"
scenario_id: tc065-feature
scenario_version: 1
entity_family: feature
stage_matrix:
  prelude:
    D01: {applicable: true}
    D02: {applicable: true}
    D03: {applicable: true}
    D04: {applicable: true}
    D05: {applicable: true}
admission: {status: admitted}
replay_reference: evaluator/replay/bundle.json
evaluator_only:
  reference_solution: evaluator/reference.patch
  oracle_tests: []
  answer_keys: []
fixture:
  fixture_id: py
  submodule_path: bench/fixture-py
  base_sha: "964fa68e4c9e0c4e0f3756d9efd78b888c558fd9"
adapter: {name: python, version: "1.0.0"}
toolchain_identity: [{key: python_version, value: "3.12.3"}]
resource_policy: {max_cost_usd: 1, max_wall_clock_seconds: 60, max_generated_tasks: 10}
YAML

cat >"$WORKDIR/package-bug.yaml" <<'YAML'
schema_version: "1.0"
scenario_id: tc065-bug
scenario_version: 1
entity_family: bug
stage_matrix:
  prelude:
    D01: {applicable: false, reason: "bug scenarios bypass product design"}
    D02: {applicable: false, reason: "bug scenarios bypass product design"}
    D03: {applicable: false, reason: "bug scenarios bypass product design"}
    D04: {applicable: false, reason: "bug scenarios bypass product design"}
    D05: {applicable: false, reason: "bug scenarios bypass product design"}
admission: {status: admitted}
evaluator_only:
  reference_solution: evaluator/reference.patch
  oracle_tests: []
  answer_keys: []
YAML

python3 - "$WORKDIR/replay-complete.json" <<'PY'
import json
import sys

consumed_entries = [{"entry_digest": f"digest-{index:02d}"} for index in range(40)]
json.dump({
    "schema_version": "1.0",
    "scenario": {"scenario_id": "tc065-feature", "scenario_version": 1},
    "run_id": "tc065",
    "terminal_outcome": "complete",
    "replay_bundle": {
        "bundle_path": "$WORKDIR/evaluator/replay/bundle.json",
        "bundle_digest": "REPLACE",
        "bundle_version": "1.0.0",
    },
    "stages": [
        {"stage": "D01", "consumed_entries": consumed_entries, "artifacts": []},
        {"stage": "D02"}, {"stage": "D03"}, {"stage": "D04"}, {"stage": "D05"},
    ],
    "questions": [{
        "question_key": "Q-E40-F08-001", "current_responder": "responder-a",
        "owner": "owner-a", "summary": "approved",
        "evidence_pointer": "runs/tc065/answer.json", "resolution_kind": "accepted",
        "resolution_pointer": "runs/tc065/resolution.json",
    }],
}, open(sys.argv[1], "w", encoding="utf-8"))
PY

cat >"$WORKDIR/replay-blocked.json" <<'JSON'
{"schema_version":"1.0","scenario":{"scenario_id":"tc065-feature","scenario_version":1},"terminal_outcome":"unresolved_gate","stages":[]}
JSON

cat >"$WORKDIR/replay-missing-scenario-id.json" <<'JSON'
{"schema_version":"1.0","scenario":{},"run_id":"tc065","terminal_outcome":"complete","stages":[{"stage":"D01"},{"stage":"D02"},{"stage":"D03"},{"stage":"D04"},{"stage":"D05"}]}
JSON

cat >"$WORKDIR/replay-missing-question-block.json" <<'JSON'
{"schema_version":"1.0","scenario":{"scenario_id":"tc065-feature","scenario_version":1},"run_id":"tc065","terminal_outcome":"complete","stages":[{"stage":"D01"},{"stage":"D02"},{"stage":"D03"},{"stage":"D04"},{"stage":"D05"}],"questions":[{"question_key":"Q-E40-F08-002","current_responder":"responder-a","owner":"owner-a","summary":"approved","evidence_pointer":"runs/tc065/answer.json","resolution_kind":"accepted","resolution_pointer":"runs/tc065/resolution.json"}]}
JSON

python3 - "$WORKDIR" <<'PY'
import hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
bundle = root / "evaluator/replay/bundle.json"
provenance = {
    "bundle_path": str(bundle.resolve()),
    "bundle_digest": "sha256:" + hashlib.sha256(bundle.read_bytes()).hexdigest(),
    "bundle_version": "1.0.0",
}
for name in ("replay-complete.json", "replay-missing-question-block.json"):
    path = root / name
    doc = json.loads(path.read_text())
    doc["replay_bundle"] = provenance
    path.write_text(json.dumps(doc, separators=(",", ":")) + "\n")
PY

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$PRELUDE" --scenario "$WORKDIR/package-feature.yaml" --replay "$WORKDIR/replay-complete.json" \
  --run-id tc065 --output "$WORKDIR/feature.jsonl" --fixture-root "$WORKDIR/fixture" \
  --scratch-root "$WORKDIR/scratch" --evaluator-root "$WORKDIR/evaluator" >/dev/null

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$PRELUDE" --scenario "$WORKDIR/package-bug.yaml" --run-id tc065-bug \
  --output "$WORKDIR/bug.jsonl" >/dev/null

if PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$PRELUDE" --scenario "$WORKDIR/package-feature.yaml" --replay "$WORKDIR/replay-blocked.json" \
  --run-id tc065-blocked --output "$WORKDIR/blocked.jsonl" >/dev/null 2>"$WORKDIR/blocked.err"; then
    fail "blocked replay unexpectedly passed"
fi

if PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$PRELUDE" --scenario "$WORKDIR/package-feature.yaml" --replay "$WORKDIR/replay-missing-scenario-id.json" \
  --run-id tc065-missing-scenario --output "$WORKDIR/missing-scenario.jsonl" >/dev/null 2>"$WORKDIR/missing-scenario.err"; then
    fail "replay missing scenario_id unexpectedly passed"
fi
grep -q "missing scenario.scenario_id" "$WORKDIR/missing-scenario.err" || fail "missing replay scenario_id was not named"

events_before="$(wc -l <"$WORKDIR/events.ndjson")"
# run-lifecycle.sh (unlike lifecycle-prelude.sh above) has no --fixture-root
# override -- it resolves fixture_root from the scenario package's own
# fixture.submodule_path, so a private copy of the package pointed at this
# same clone is needed for isolation.
RUNNER_SCENARIO="$WORKDIR/package-feature-runner.yaml"
sed "s#^  submodule_path: .*#  submodule_path: $WORKDIR/fixture#" \
    "$WORKDIR/package-feature.yaml" >"$RUNNER_SCENARIO"
mkdir -p "$WORKDIR/input"
printf 'feature input\n' >"$WORKDIR/input/prompt.md"
printf '\ninput:\n  agent_visible: input/prompt.md\n' >>"$RUNNER_SCENARIO"
set +e
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$RUNNER" --scenario "$RUNNER_SCENARIO" --replay "$WORKDIR/replay-blocked.json" \
  --run-id tc065-runner-blocked --root ROOT-NOT-DISPATCHED --scratch-root "$WORKDIR/scratch" \
  --i05-bundle-dir "$WORKDIR/runner-i05" \
  --output "$WORKDIR/runner-blocked.jsonl" --mode contract >/dev/null
blocked_code=$?
set -e
# unresolved_gate is a named stop outcome with retained evidence, not
# "complete" -- exit 1.
[[ "$blocked_code" -eq 1 ]] || fail "runner-blocked run exited $blocked_code, want 1 (unresolved_gate)"
events_after="$(wc -l <"$WORKDIR/events.ndjson")"
[[ "$events_after" == "$events_before" ]] || fail "runner dispatched Shark work after an unresolved I-06 replay"
python3 - "$WORKDIR/runner-blocked.jsonl" <<'PY'
import json, sys
record = json.load(open(sys.argv[1], encoding="utf-8"))
assert record["outcome"]["terminal"] == "unresolved_gate", record["outcome"]
assert record["outcome"]["publication_eligible"] is False, record["outcome"]
assert record["dispatches"] == [], record["dispatches"]
assert record["prelude"]["terminal_outcome"] == "unresolved_gate", record["prelude"]
PY

# TD-227: I-05 snapshots must materialize the authoritative I-06 consumption
# ledger retained in the real feature prelude. This uses the production runner
# entrypoint and its normal --prelude caller signature; it would fail if
# record_stage() restored the empty replay_lineage placeholder.
PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
  "$RUNNER" --scenario "$RUNNER_SCENARIO" --prelude "$WORKDIR/feature.jsonl" \
  --run-id tc065-lineage --root ROOT-LINEAGE --scratch-root "$WORKDIR/scratch" \
  --i05-bundle-dir "$WORKDIR/lineage-i05" --output "$WORKDIR/lineage.jsonl" --mode dry-run \
  || fail "feature replay-lineage fixture run failed"

python3 - "$WORKDIR/lineage-i05" "$WORKDIR/evaluator/replay/bundle.json" "$WORKDIR/lineage.jsonl" <<'PY'
import json
import pathlib
import sys

i05_dir = pathlib.Path(sys.argv[1])
reference = str(pathlib.Path(sys.argv[2]).resolve())
bundle = json.loads((i05_dir / "bundle.json").read_text(encoding="utf-8"))
assert len(bundle["stages"]) == 1, bundle
snapshot = json.loads((i05_dir / bundle["stages"][0]["snapshot_path"]).read_text(encoding="utf-8"))
assert snapshot["replay_lineage"] == [
    {"replay_reference": reference, "entry_digest": f"digest-{index:02d}"}
    for index in range(40)
], snapshot
# The persisted lifecycle record remains diagnostic and bounded. Its replay
# copy retains only 32 nested consumed entries, so this proves the snapshot
# did not take its semantic lineage from that diagnostic copy.
record = json.loads(pathlib.Path(sys.argv[3]).read_text(encoding="utf-8"))
diagnostic_entries = record["prelude"]["replay"]["stages"][0]["consumed_entries"]
assert len(diagnostic_entries) == 32, diagnostic_entries
PY

# Artifact-scoped claims are optional and non-authoritative. They must never
# manufacture I-05 replay lineage when the resolver-owned stage ledger is
# empty. Keep all 40 claims here so this is also a counterfactual against a
# future bounded diagnostic source.
python3 - "$WORKDIR/feature.jsonl" "$WORKDIR/artifact-only.jsonl" <<'PY'
import json
import sys

prelude = json.loads(open(sys.argv[1], encoding="utf-8").readline())
stage = prelude["replay"]["stages"][0]
claims = stage.pop("consumed_entries")
stage["artifacts"] = [{"consumed_entries": claims}]
with open(sys.argv[2], "w", encoding="utf-8") as stream:
    json.dump(prelude, stream, separators=(",", ":"))
    stream.write("\n")
PY

PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
  "$RUNNER" --scenario "$RUNNER_SCENARIO" --prelude "$WORKDIR/artifact-only.jsonl" \
  --run-id tc065-artifact-only --root ROOT-LINEAGE --scratch-root "$WORKDIR/scratch" \
  --i05-bundle-dir "$WORKDIR/artifact-only-i05" --output "$WORKDIR/artifact-only.jsonl.out" --mode dry-run \
  || fail "artifact-only replay-lineage fixture run failed"

python3 - "$WORKDIR/artifact-only-i05" <<'PY'
import json
import pathlib
import sys

i05_dir = pathlib.Path(sys.argv[1])
bundle = json.loads((i05_dir / "bundle.json").read_text(encoding="utf-8"))
snapshot = json.loads((i05_dir / bundle["stages"][0]["snapshot_path"]).read_text(encoding="utf-8"))
assert snapshot["replay_lineage"] == [], snapshot
PY

# The runner rejects malformed externally supplied prelude JSON before it can
# dereference the new replay-lineage projection. Exercise each guarded shape
# through the production --prelude entrypoint.
python3 - "$WORKDIR/feature.jsonl" "$WORKDIR/malformed-prelude.jsonl" "$WORKDIR/malformed-replay.jsonl" "$WORKDIR/malformed-bundle.jsonl" <<'PY'
import json
import sys

prelude = json.loads(open(sys.argv[1], encoding="utf-8").readline())
open(sys.argv[2], "w", encoding="utf-8").write("[]\n")
for path, key, value in ((sys.argv[3], "replay", []), (sys.argv[4], "replay_bundle", [])):
    candidate = json.loads(json.dumps(prelude))
    if key == "replay_bundle":
        candidate["replay"][key] = value
    else:
        candidate[key] = value
    with open(path, "w", encoding="utf-8") as stream:
        json.dump(candidate, stream, separators=(",", ":"))
        stream.write("\n")
PY

for malformed_case in prelude replay bundle; do
  case "$malformed_case" in
    prelude) malformed_path="$WORKDIR/malformed-prelude.jsonl"; expected="lifecycle prelude must be a JSON object" ;;
    replay) malformed_path="$WORKDIR/malformed-replay.jsonl"; expected="lifecycle prelude replay must be a JSON object" ;;
    bundle) malformed_path="$WORKDIR/malformed-bundle.jsonl"; expected="lifecycle prelude replay_bundle must be a JSON object" ;;
  esac
  if PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" SHARK_WORKFLOW_DIR="$SCRIPTS_DIR/testdata/lifecycle/workflow" \
    "$RUNNER" --scenario "$RUNNER_SCENARIO" --prelude "$malformed_path" --run-id "tc065-malformed-$malformed_case" \
    --root ROOT-LINEAGE --scratch-root "$WORKDIR/scratch" --i05-bundle-dir "$WORKDIR/malformed-$malformed_case-i05" \
    --output "$WORKDIR/malformed-$malformed_case.out" --mode dry-run >/dev/null 2>"$WORKDIR/malformed-$malformed_case.err"; then
    fail "malformed $malformed_case prelude unexpectedly passed"
  fi
  grep -q "$expected" "$WORKDIR/malformed-$malformed_case.err" || fail "malformed $malformed_case prelude did not name its shape error"
done

if PATH="$WORKDIR/bin:$PATH" SHARK_EVENTS="$WORKDIR/events.ndjson" \
  "$PRELUDE" --scenario "$WORKDIR/package-feature.yaml" --replay "$WORKDIR/replay-missing-question-block.json" \
  --run-id tc065-missing-question --output "$WORKDIR/missing-question.jsonl" --fixture-root "$WORKDIR/fixture" \
  --scratch-root "$WORKDIR/scratch" --evaluator-root "$WORKDIR/evaluator" >/dev/null 2>"$WORKDIR/missing-question.err"; then
    fail "missing question_block unexpectedly passed"
fi
grep -q "omitted question_block" "$WORKDIR/missing-question.err" || fail "missing question_block was not named"

python3 - "$WORKDIR/feature.jsonl" "$WORKDIR/bug.jsonl" "$WORKDIR/blocked.jsonl" "$WORKDIR/events.ndjson" <<'PY'
import json, sys
feature = json.loads(open(sys.argv[1], encoding="utf-8").readline())
bug = json.loads(open(sys.argv[2], encoding="utf-8").readline())
blocked = json.loads(open(sys.argv[3], encoding="utf-8").readline())
events = [json.loads(line) for line in open(sys.argv[4], encoding="utf-8")]
assert feature["terminal_outcome"] == "complete"
assert [stage["stage"] for stage in feature["prelude"]] == ["D01", "D02", "D03", "D04", "D05"]
assert len(feature["replay"]["digest"]) == 64, feature["replay"]
assert feature["replay"]["stage_count"] == 5, feature["replay"]
assert [stage["stage"] for stage in feature["replay"]["stages"]] == ["D01", "D02", "D03", "D04", "D05"]
assert feature["replay"]["replay_bundle"]["bundle_version"] == "1.0.0", feature["replay"]
assert feature["questions"][0]["terminal_result"] == "accepted"
assert bug["terminal_outcome"] == "not_applicable"
assert all(stage["outcome"] == "not_applicable" for stage in bug["prelude"])
assert blocked["terminal_outcome"] == "unresolved_gate"
assert blocked["publication_eligible"] is False
question = [event["argv"] for event in events if event["argv"] and event["argv"][0] == "question"]
assert question == [
    ["question", "respond", "Q-E40-F08-001", "--session", "SID-Q", "--responder", "responder-a", "--summary", "approved", "--evidence-pointer", "runs/tc065/answer.json"],
    ["question", "resolve", "Q-E40-F08-001", "--owner", "owner-a", "--resolution-kind", "accepted", "--resolution-pointer", "runs/tc065/resolution.json"],
]
assert not any(event["argv"] and event["argv"][0] == "question" for event in events if event["argv"] and event["argv"][0] == "question" and "blocked" in str(event))
PY

echo "TC-065: pass (prelude ordering, explicit non-applicable records, Question routing, blocked replay, I-05 replay lineage, and isolation gate)"
