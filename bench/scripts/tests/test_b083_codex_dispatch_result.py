"""Exercise the return path of one advertised Codex dispatch, using a local CLI stub."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

import pytest
import yaml


SCRIPTS = Path(__file__).resolve().parents[1]
MODEL = "synthetic-second"


@pytest.fixture
def codex(tmp_path):
    # Use the same executable for paginated discovery and the recorded exec.
    catalog = SCRIPTS / "tests" / "testdata" / "b083_fake_codex_cli.py"
    executable = tmp_path / "codex"
    executable.write_text(
        """#!/usr/bin/env python3
import json
import os
from pathlib import Path
import runpy
import sys

if sys.argv[1:2] != ["exec"]:
    runpy.run_path(os.environ["FIXTURE_CATALOG_CLI"], run_name="__main__")
    sys.exit(0)

with Path("exec.jsonl").open("a") as capture:
    print(json.dumps({"argv": sys.argv[1:], "prompt": sys.stdin.read()}), file=capture)
envelope = {"kind": "final", "recommended_outcome": "pass", "evidence": []}
if os.environ["FIXTURE_OUTPUT"] == "envelope":
    print(json.dumps(envelope))
else:
    print(json.dumps({"type": "thread.started", "thread_id": "synthetic-thread"}))
    print(json.dumps({"type": "turn.started"}))
    print(json.dumps({"type": "item.completed", "item": {
        "id": "synthetic-item", "type": "agent_message", "text": json.dumps(envelope),
    }}))
    print(json.dumps({"type": "turn.completed", "usage": {
        "input_tokens": 2, "cached_input_tokens": 0, "output_tokens": 1,
    }}))
""",
        encoding="utf-8",
    )
    executable.chmod(0o755)
    return {
        "PATH": os.environ["PATH"],
        "HOME": str(tmp_path),
        "E40_CODEX_COMMAND": json.dumps([str(executable)]),
        "FIXTURE_CATALOG_CLI": str(catalog),
    }


@pytest.mark.parametrize("selected_profile", [False, True], ids=["workflow", "profile"])
@pytest.mark.parametrize("output", ["envelope", "codex-jsonl"])
def test_tc123_advertised_dispatch_returns_worker_outcome(tmp_path, codex, selected_profile, output):
    workflow = tmp_path / "bug.yaml"
    stage = {"action": "spawn_agent", "provider": "codex", "model": MODEL, "effort": "high"}
    command = [
        sys.executable, str(SCRIPTS / "materialize-routing-profile.py"),
        "--workflow-root", str(workflow),
    ]
    if selected_profile:
        stage.update(provider="anthropic", model="synthetic-source")
        profile = tmp_path / "profile.yaml"
        profile.write_text(yaml.safe_dump({
            "schema_version": "1.0", "profile_id": "synthetic-switch", "provider": "codex",
            "description": "Synthetic route", "routes": {"development": {"model": MODEL, "effort": "high"}},
        }))
        command.extend(["--profile", str(profile)])
    workflow.write_text(yaml.safe_dump({"steps": {"development": stage}}))
    env = {**codex, "FIXTURE_OUTPUT": output}
    result = subprocess.run(command, cwd=tmp_path, env=env, capture_output=True, text=True, check=True)
    assert json.loads(result.stdout)["selected_codex_models"]["bug.yaml::development"]["model"] == MODEL
    assert not (tmp_path / "exec.jsonl").exists()

    prompt = "Return the fixture worker outcome."
    request = {
        **yaml.safe_load(workflow.read_text())["steps"]["development"],
        "entity_key": "B001", "entity_type": "bug", "status": "development",
        "session_id": "synthetic-session", "allowed_outcomes": ["pass"],
        "prompt": prompt, "prompt_bytes": len(prompt.encode()),
        "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(),
    }
    result = subprocess.run(
        [str(SCRIPTS / "lifecycle-worker-adapter.sh")], cwd=tmp_path, env=env,
        input=json.dumps(request), capture_output=True, text=True,
    )
    starts = [json.loads(line) for line in (tmp_path / "exec.jsonl").read_text().splitlines()]
    expected_argv = ["exec", "--json", "-m", MODEL, "-c", 'model_reasoning_effort="high"']
    assert starts == [{"argv": expected_argv, "prompt": prompt}]
    assert result.returncode == 0, result.stderr
    outcome = json.loads(result.stdout)
    assert (outcome["kind"], outcome["recommended_outcome"], outcome["session_id"]) == (
        "final", "pass", request["session_id"],
    )
