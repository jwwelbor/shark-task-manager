"""C05/C13/C14. Vary only the recorder's producing-turn evidence for each route."""
import json
import sqlite3
import sys
from pathlib import Path

source_dir = Path(__file__).resolve().parent.parent / "development-6c6345cf"
sys.path.insert(0, str(source_dir))
from assignment_probe import fixture

# These lines operate only on the recording provider's independent rollout.
cases = (
    ("matched", "pass", "verified"),
    ("other-provider", 'records[0][1]["model_provider"] = "other"', "mismatch"),
    ("other-model", 'records[1][1]["model"] = "other"', "mismatch"),
    ("other-model-and-provider", 'records[0][1]["model_provider"] = records[1][1]["model"] = "other"', "mismatch"),
    ("other-effort", 'records[1][1]["effort"] = "xhigh"', "mismatch"),
    ("missing-effort", 'del records[1][1]["effort"]', "unverified"),
    ("missing-model", 'del records[1][1]["model"]', "unverified"),
    ("missing-provider", 'del records[0][1]["model_provider"]', "unverified"),
    ("wrong-session", 'records[0][1]["id"] = str(uuid4())', "unverified"),
    ("wrong-turn", 'records[2][1]["turn_id"] = str(uuid4())', "unverified"),
    ("conflicting-model", 'records.append(("turn_context", dict(records[1][1], model="other")))', "unverified"),
)


def run(mapped):
    case = fixture.RiderCLI()
    try:
        case.setUp()
        recorder = case.cli.read_text()
        marker = '        rollout = Path(os.environ["CODEX_HOME"])'
        assert recorder.count(marker) == 1
        selected = "matched-model"
        response = dict(model_provider="openai", model=selected, model_reasoning_effort="low")
        for name, change, expected in cases:
            case.cli.write_text(recorder.replace(marker, "        " + change + "\n" + marker))
            calls, _, rc = case.submit("haiku" if mapped else selected, response, mapped=mapped, effort="low")
            data = json.loads((case.directory / "observation").read_bytes())
            assert [c["method"] for c in calls] == ["read", "exec"]
            assert data["route_status"] == expected and data["worker_id"] == calls[1]["thread"]
            assert data["usage"] == {"input_tokens": 7} and "cost" not in data
            accepted = expected == "verified"
            assert (rc == 0) == accepted, (name, case.stderr)
            if expected == "unverified":
                assert not {"effective_model", "effective_provider", "effective_effort"} & data.keys()
            else:
                assert data["effective_provider"] == ("other" if "provider" in name else "openai")
                assert data["effective_model"] == ("other" if "model" in name else selected)
                assert data["effective_effort"] == ("xhigh" if name == "other-effort" else "low")
            counts = (case.directory / "counts").read_text().splitlines()
            assert counts.count("claim") == counts.count("release") == 1
            stage = json.loads(case.shark("get", "B001", "--json").stdout)["status"]
            assert stage == ("repair_plan" if accepted else "research")
            with sqlite3.connect(case.work / "shark-tasks.db") as db:
                assert db.execute("select count(*) from entity_claims").fetchone()[0] == 0
                transitions = db.execute("select count(*) from entity_history where changed_by=?",
                                         ("researcher@openai/" + selected,)).fetchone()[0]
            assert transitions == 1  # The sole matched result in this fixture's first run.
            print(json.dumps(dict(mapped=mapped, case=name, route_status=expected, stage=stage,
                                  worker=data["worker_id"], usage=data["usage"], cost=None,
                                  claims=1, submissions=1, releases=1, cumulative_worker_transitions=transitions)), flush=True)
            if accepted:
                case.shark("status", "set", "B001", "research", "--force", "--reason", "reset isolated stage")
    finally:
        case.doCleanups()


if __name__ == "__main__":
    run(mapped=True)
    run(mapped=False)
