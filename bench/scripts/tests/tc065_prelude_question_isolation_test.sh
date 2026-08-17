#!/usr/bin/env bash
# TC-065 / T-E40-F08-004: pre-dispatch prelude, Question, and I-05 isolation gate.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PRELUDE="$SCRIPTS_DIR/lifecycle-prelude.sh"
fail() { echo "TC-065 FAIL: $1" >&2; exit 1; }
[[ -x "$PRELUDE" ]] || fail "lifecycle-prelude.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/fixture" "$WORKDIR/scratch" "$WORKDIR/evaluator"
mkdir -p "$WORKDIR/evaluator/evaluator"
printf 'fixture reference\n' >"$WORKDIR/evaluator/evaluator/reference.patch"
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
elif args[:2] == ["claim", "Q-E40-F08-001"]:
    print('{"session_id":"SID-Q"}')
elif args[:2] == ["question", "respond"] or args[:2] == ["question", "resolve"]:
    print('{"ok":true}')
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

cat >"$WORKDIR/replay-complete.json" <<'JSON'
{"schema_version":"1.0","scenario":{"scenario_id":"tc065-feature","scenario_version":1},"run_id":"tc065","terminal_outcome":"complete","replay_bundle":{"bundle_path":"$WORKDIR/evaluator/replay/bundle.json","bundle_digest":"REPLACE","bundle_version":"1.0.0"},"stages":[{"stage":"D01"},{"stage":"D02"},{"stage":"D03"},{"stage":"D04"},{"stage":"D05"}],"questions":[{"question_key":"Q-E40-F08-001","current_responder":"responder-a","owner":"owner-a","summary":"approved","evidence_pointer":"runs/tc065/answer.json","resolution_kind":"accepted","resolution_pointer":"runs/tc065/resolution.json"}]}
JSON

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

echo "TC-065: pass (prelude ordering, explicit non-applicable records, Question routing, blocked replay, and isolation gate)"
