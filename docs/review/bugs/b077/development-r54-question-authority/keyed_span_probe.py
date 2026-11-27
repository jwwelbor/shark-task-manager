"""C06. Read the real post-claim span from the mapped Opus harness fixture."""
import json
import sys
from pathlib import Path
from unittest.mock import patch

evidence_dir = Path(__file__).resolve().parent
sys.path.insert(0, str(evidence_dir.parent / "development-6c6345cf"))
import harness_probe

fixture = harness_probe.fixture
original_setup = fixture.RiderCLI.setUp
observed = []


def enable_fixture_trace(case):
    original_setup(case)
    cfg_path = case.work / ".sharkconfig.json"
    cfg = json.loads(cfg_path.read_bytes())
    cfg["observability"] = dict(enabled=True, tracing_enabled=True, exporter="stdout")
    cfg_path.write_text(json.dumps(cfg))
    trace = case.directory / "keyed.stderr"
    wrapper = case.directory / "shark"
    original = wrapper.read_text()
    wrapper.write_text(original.replace('exec "$B077_SHARK" "$@"', '''
if [ "$1" = next ]; then exec "$B077_SHARK" "$@" 2> "$CASE/keyed.stderr"; fi
exec "$B077_SHARK" "$@"
'''))
    case.addCleanup(collect_span, trace)


def collect_span(trace):
    for line in trace.read_text().splitlines():
        if line.startswith("{"):
            event = json.loads(line)
            if event.get("Name") == "shark.next":
                observed.append({attr["Key"]: attr["Value"]["Value"] for attr in event["Attributes"]})


with patch.object(fixture.RiderCLI, "setUp", enable_fixture_trace):
    result = harness_probe.run("matching")
expected = dict(entity_key="B001", entity_type="bug", status="plan_review",
                action="spawn_agent", agent_type="tech-lead", model="opus",
                harness="codex", prompt_bytes=result["keyed_and_received_bytes"],
                unresolved_placeholders=0, exit_status="ok")
assert observed == [expected], observed
print(json.dumps(dict(span=observed[0], receiver_digest=result["keyed_and_received_sha256"],
                      producing_model=result["checked_model"])))
