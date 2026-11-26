"""C04/R20-01. Run candidate Rider blocks against an isolated Shark research assignment."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shlex
import sqlite3
import sys
from unittest.mock import patch

root = Path(__file__).resolve().parents[5]
spec = importlib.util.spec_from_file_location("rider_fixture", root / "tests/contracts/testdata/b077_codex_route.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)


def probe(field, after, omit_comparison=False, effort="low", model="haiku"):
    case = fixture.RiderCLI()
    try:
        case.setUp()
        # Use identical persona bytes to distinguish a route change from a prompt change.
        agents = case.work / "shark-data/agents"
        (agents / "alternate.md").write_bytes((agents / "researcher.md").read_bytes())
        wrapper = case.directory / "shark"
        alter = case.directory / "after-claim.py"
        alter.write_text(
            "from pathlib import Path\nimport json, sys\n"
            "path = Path(sys.argv[1]); field = sys.argv[2]; after = json.loads(sys.argv[3])\n"
            "before, research = path.read_text().split('  research:\\n')\n"
            "rows = [line for line in research.splitlines() if not line.startswith('    ' + field + ':')]\n"
            "if after is not None: rows.insert(1, '    ' + field + ': ' + json.dumps(after))\n"
            "path.write_text(before + '  research:\\n' + '\\n'.join(rows) + '\\n')\n"
        )
        command = shlex.join([sys.executable, str(alter), str(case.workflow), field, json.dumps(after)])
        if field:
            wrapper.write_text(wrapper.read_text().replace('exec "$B077_SHARK" "$@"',
                'if [ "$1" = next ]; then\n    ' + command + '\nfi\nexec "$B077_SHARK" "$@"'))
        initial_mapped = model == "haiku"
        selected = "discovered-A" if initial_mapped else (model or "documented-default")
        settings = dict(model_provider="openai", model=selected, model_reasoning_effort=effort or "xhigh")
        commands = fixture.commands()
        wire = "agent_type" if field == "agent" else field
        if omit_comparison:
            token = ',"' + wire + '"'
            comparison = 'def assignment: with_entries(select(.key | IN('
            line = next(line for line in commands.splitlines() if comparison in line)
            assert line.count(token) == 1
            commands = commands.replace(line, line.replace(token, ""))
        with patch.object(fixture, "commands", return_value=commands):
            calls, receipt, rc = case.submit(model, settings, mapped=initial_mapped,
                                             default="documented-default", effort=effort)
        old, new = [json.loads((case.directory / f).read_text()) for f in ("request", "current")]
        fields = ["entity_key", "entity_type", "status", "action", "agent_type", "provider", "model", "effort"]
        changed = [key for key in fields if old.get(key) != new.get(key)]
        assert changed == ([wire] if field else []), changed
        prompt = lambda text: hashlib.sha256(text.encode()).hexdigest()
        assert old["prompt"] == new["prompt"], (field, "keyed prompt varied")
        sequence = (case.directory / "counts").read_text().splitlines()
        with sqlite3.connect(case.work / "shark-tasks.db") as db:
            claims = db.execute("select count(*) from entity_claims where entity_key='B001'").fetchone()[0]
            sessions = db.execute("select session_id,outcome,ended_at from work_sessions where entity_key='B001'").fetchall()
        expected = not field or omit_comparison
        executions = [call for call in calls if call["method"] == "exec"]
        stage = json.loads(case.shark("get", "B001", "--json").stdout)["status"]
        assert (rc == 0) == expected and len(executions) == int(expected), (field, rc, case.stderr)
        assert all(call["prompt"] == new["prompt"] for call in executions)
        assert claims == 0 and sequence.count("claim") == sequence.count("release") == 1
        assert len(sessions) == 1 and sessions[0][2] and sessions[0][1] == ("pass" if expected else "released")
        assert stage == ("repair_plan" if expected else "research")
        return dict(field=wire, before=old.get(wire), after=new.get(wire), omitted_comparison=omit_comparison,
                    keyed_prompt_sha256=prompt(new["prompt"]), executable_calls=[c["method"] for c in calls],
                    claim_command_sequence=sequence, exit_status=rc, resulting_status=stage,
                    remaining_claims=claims, journal_outcome=sessions[0][1])
    finally:
        case.doCleanups()


if __name__ == "__main__":
    cases = [("", None, "low", "haiku"), ("agent", "alternate", "low", "haiku"),
             ("provider", "openai", "low", "haiku"), ("model", "opus", "low", "haiku"),
             ("effort", "high", "low", "haiku"), ("effort", None, "low", "haiku"),
             ("effort", "low", "", "haiku"), ("model", "", "low", "explicit-pin"),
             ("model", "explicit-pin", "low", "")]
    for field, after, effort, model in cases:
        for mutant in ([False, True] if field else [False]):
            print(json.dumps(probe(field, after, mutant, effort, model)), flush=True)
