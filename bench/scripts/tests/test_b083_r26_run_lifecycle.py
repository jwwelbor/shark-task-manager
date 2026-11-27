"""Integration tests for B083 R26 run-lifecycle.sh controller controls."""

import json
import os
from pathlib import Path
import subprocess
import sys

import pytest
import yaml


SCRIPTS = Path(__file__).resolve().parents[1]
REPO_ROOT = SCRIPTS.parents[1]
RUN_LIFECYCLE = SCRIPTS / "run-lifecycle.sh"


def _init_git_repo(path: Path) -> str:
    path.mkdir(parents=True, exist_ok=True)
    subprocess.run(["git", "init", "-q"], cwd=path, check=True)
    subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=path, check=True)
    subprocess.run(["git", "config", "user.name", "Test"], cwd=path, check=True)
    (path / "README.md").write_text("fixture\n", encoding="utf-8")
    subprocess.run(["git", "add", "README.md"], cwd=path, check=True)
    subprocess.run(["git", "commit", "-q", "-m", "init"], cwd=path, check=True)
    return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=path, text=True).strip()


def _setup_workspace(tmp_path: Path, *, family: str = "bug", root_key: str = "B001", extra_workflows: dict | None = None):
    fixture_root = tmp_path / "fixture"
    base_sha = _init_git_repo(fixture_root)

    scratch = tmp_path / "scratch"
    (scratch / "prompts").mkdir(parents=True, exist_ok=True)
    wf_dir = scratch / "shark-data" / "workflow"
    wf_dir.mkdir(parents=True, exist_ok=True)
    prompt_dir = scratch / "shark-data" / "prompts"
    prompt_dir.mkdir(parents=True, exist_ok=True)
    (prompt_dir / "review.md").write_text("Review prompt\n", encoding="utf-8")

    default_wf = {
        "start": "code_review",
        "steps": {
            "code_review": {
                "action": "spawn_agent",
                "phase": "code_review",
                "agent": "reviewer",
                "provider": "codex",
                "model": "synthetic-paid-model",
                "effort": "high",
                "prompt": "review.md",
                "outcomes": {"pass": "completed", "fail": "development"},
            },
            "completed": {"terminal": True},
            "development": {"action": "spawn_agent", "phase": "development", "outcomes": {"pass": "code_review"}},
        },
    }
    if extra_workflows:
        for fname, content in extra_workflows.items():
            (wf_dir / f"{fname}.yaml").write_text(yaml.safe_dump(content), encoding="utf-8")
    else:
        (wf_dir / f"{family}.yaml").write_text(yaml.safe_dump(default_wf), encoding="utf-8")

    pkg_dir = tmp_path / "pkg"
    (pkg_dir / "evaluator").mkdir(parents=True, exist_ok=True)
    (pkg_dir / "evaluator" / "test_x.py").write_text("def test_ok(): pass\n", encoding="utf-8")
    (pkg_dir / "input").mkdir(parents=True, exist_ok=True)
    (pkg_dir / "input" / "prompt.md").write_text("Solve task\n", encoding="utf-8")
    scenario_path = pkg_dir / "package.yaml"
    scenario = {
        "schema_version": "1.1",
        "scenario_id": "test-r26-scenario",
        "scenario_version": 1,
        "entity_family": family,
        "fixture": {"fixture_id": "py", "submodule_path": str(fixture_root), "base_sha": base_sha},
        "adapter": {"name": "python", "version": "1.0.0"},
        "toolchain_identity": [],
        "input": {"agent_visible": "input/prompt.md"},
        "evaluator_only": {"oracle_tests": ["evaluator/test_x.py"]},
        "final_predicate": {"kind": "oracle_tests"},
        "resource_policy": {"max_cost_usd": 10.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5},
        "admission": {"status": "admitted"},
    }
    scenario_path.write_text(yaml.safe_dump(scenario), encoding="utf-8")

    log_path = tmp_path / "shark_calls.jsonl"
    apply_capture = tmp_path / "applied_payload.bin"
    output_path = tmp_path / "lifecycle.jsonl"
    i05_dir = tmp_path / "i05"

    return {
        "fixture_root": fixture_root,
        "scratch": scratch,
        "scenario_path": scenario_path,
        "log_path": log_path,
        "apply_capture": apply_capture,
        "output_path": output_path,
        "i05_dir": i05_dir,
        "root_key": root_key,
    }


def _write_fake_shark(tmp_path: Path, log_path: Path, apply_capture: Path, script_body: str) -> Path:
    shark_bin = tmp_path / "fake_shark"
    shark_bin.write_text(
        f"""#!/usr/bin/env python3
import hashlib, json, os, sys
from pathlib import Path

LOG_PATH = Path({str(log_path)!r})
APPLY_CAPTURE = Path({str(apply_capture)!r})
argv = sys.argv[1:]
with LOG_PATH.open("a", encoding="utf-8") as f:
    f.write(json.dumps(argv) + "\\n")

def emit_next(payload):
    if isinstance(payload, dict) and "prompt" in payload and not payload.get("terminal"):
        b = payload["prompt"].encode("utf-8")
        payload.setdefault("prompt_bytes", len(b))
        payload.setdefault("prompt_sha256", hashlib.sha256(b).hexdigest())
        if "--prompt-out" in argv:
            p = Path(argv[argv.index("--prompt-out") + 1])
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes(b)
    print(json.dumps(payload))

if argv[:3] == ["admin", "workflow", "list"]:
    level = argv[3] if len(argv) > 3 else "bug"
    print(json.dumps({{"levels": [{{"level": level, "statuses": [
        {{"name": "code_review", "phase": "code_review"}},
        {{"name": "in_code_review", "phase": "code_review"}},
    ]}}]}}))
    sys.exit(0)

{script_body}
""",
        encoding="utf-8",
    )
    shark_bin.chmod(0o755)
    return shark_bin


def _write_fake_adapter(tmp_path: Path, body: str) -> Path:
    adapter_bin = tmp_path / "fake_adapter"
    adapter_bin.write_text(f"#!/usr/bin/env python3\n{body}\n", encoding="utf-8")
    adapter_bin.chmod(0o755)
    return adapter_bin


def _run_lifecycle(ws: dict, shark_bin: Path, adapter_bin: Path, *, with_i05: bool = True, run_id: str = "test-run-r26") -> subprocess.CompletedProcess:
    cmd = [
        str(RUN_LIFECYCLE),
        "--scenario", str(ws["scenario_path"]),
        "--root", ws["root_key"],
        "--scratch-root", str(ws["scratch"]),
        "--fixture-root", str(ws["fixture_root"]),
        "--output", str(ws["output_path"]),
        "--run-id", run_id,
        "--i05-bundle-dir", str(ws["i05_dir"]),
        "--mode", "live",
    ]
    env = {
        **os.environ,
        "SHARK_BIN": str(shark_bin),
        "LIFECYCLE_ADAPTER": str(adapter_bin),
        "LIFECYCLE_BENCH_DIR": str(REPO_ROOT / "bench"),
    }
    return subprocess.run(cmd, env=env, capture_output=True, text=True, check=False)


def test_r26_final_heartbeat_before_apply_result_and_retired_skips_release(tmp_path):
    ws = _setup_workspace(tmp_path)
    shark_body = """
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists():
        print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "B001", "entity_type": "bug", "status": "code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    idx = argv.index("--apply-result") + 1
    APPLY_CAPTURE.write_bytes(Path(argv[idx]).read_bytes())
    STATE_FILE.write_text("done", encoding="utf-8")
    print(json.dumps({
        "run_id": "r1", "entity_key": "B001", "entity_type": "bug",
        "outcome_key": "pass", "role": "success", "to_status": "completed",
        "transitioned": True, "status": {"retirement_state": "retired"},
    }))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    canonical = {
        "kind": "final",
        "recommended_outcome": "pass",
        "evidence": [],
        "gate_result": {"schema_version": 1, "summary": "ok", "findings": [], "kickbacks": []},
    }
    adapter_bin = _write_fake_adapter(
        tmp_path,
        f"import json, sys\nreq = json.load(sys.stdin)\nprint(json.dumps({canonical!r}))\n",
    )
    proc = _run_lifecycle(ws, shark_bin, adapter_bin)
    assert ws["output_path"].is_file(), proc.stderr

    calls = [json.loads(line) for line in ws["log_path"].read_text(encoding="utf-8").splitlines()]
    hb_idx = next(i for i, c in enumerate(calls) if c[0] == "heartbeat" and "--progress" in c and "1.0" in c)
    apply_idx = next(i for i, c in enumerate(calls) if c[0] == "run" and "--apply-result" in c)
    assert hb_idx < apply_idx
    assert not any(c[0] == "release" for c in calls), "retired apply-result must skip shark release"

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["dispatches"][0]["transition"]["retired"] is True
    assert record["outcome"]["terminal"] == "complete"


def test_r26_final_heartbeat_failure_raises_lease_loss_before_apply_result(tmp_path):
    ws = _setup_workspace(tmp_path)
    shark_body = """
if argv[0] == "next":
    emit_next({
        "entity_key": "B001", "entity_type": "bug", "status": "code_review",
        "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
        "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
        "allowed_outcomes": ["pass", "fail"],
    })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print("lease lost", file=sys.stderr)
    sys.exit(1)
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    canonical = {
        "kind": "final",
        "recommended_outcome": "pass",
        "evidence": [],
        "gate_result": {"schema_version": 1, "summary": "ok", "findings": [], "kickbacks": []},
    }
    adapter_bin = _write_fake_adapter(
        tmp_path,
        f"import json, sys\nreq = json.load(sys.stdin)\nprint(json.dumps({canonical!r}))\n",
    )
    proc = _run_lifecycle(ws, shark_bin, adapter_bin)
    assert ws["output_path"].is_file(), proc.stderr

    calls = [json.loads(line) for line in ws["log_path"].read_text(encoding="utf-8").splitlines()]
    assert not any(c[0] == "run" and "--apply-result" in c for c in calls)
    release_calls = [c for c in calls if c[0] == "release"]
    assert len(release_calls) == 1
    assert "--outcome" in release_calls[0] and "lease_loss" in release_calls[0]

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["dispatches"][0]["outcome"] == "lease_loss"
    assert record["outcome"]["terminal"] == "lease_loss"


@pytest.mark.parametrize(
    "mode",
    [
        "bare",
        "enriched_raw_final",
        "enriched_missing_raw_final",
        "enriched_disagreeing_raw_final",
        "enriched_empty_raw_final",
        "enriched_invalid_json_raw_final",
    ],
)
def test_r26_raw_final_boundary_bare_and_enriched(tmp_path, mode):
    ws = _setup_workspace(tmp_path)
    shark_body = """
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists():
        print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "B001", "entity_type": "bug", "status": "code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    idx = argv.index("--apply-result") + 1
    APPLY_CAPTURE.write_bytes(Path(argv[idx]).read_bytes())
    STATE_FILE.write_text("done", encoding="utf-8")
    print(json.dumps({"run_id": "r1", "to_status": "completed", "transitioned": True, "status": {"retirement_state": "retired"}}))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"exact-raw","findings":[],"kickbacks":[]}}'
    disagreeing_raw_final_str = '{"kind":"final","recommended_outcome":"fail","evidence":[],"gate_result":{"schema_version":1,"summary":"disagree","findings":[],"kickbacks":[]}}'
    if mode == "bare":
        adapter_body = f"import sys\nsys.stdin.read()\nprint({raw_final_str!r})\n"
    elif mode == "enriched_raw_final":
        adapter_body = (
            "import json, sys\nreq = json.load(sys.stdin)\n"
            f"print(json.dumps({{'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', 'recommended_outcome': 'pass', 'raw_final': {raw_final_str!r}}}))\n"
        )
    elif mode == "enriched_disagreeing_raw_final":
        adapter_body = (
            "import json, sys\nreq = json.load(sys.stdin)\n"
            f"print(json.dumps({{'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', 'recommended_outcome': 'pass', 'raw_final': {disagreeing_raw_final_str!r}}}))\n"
        )
    elif mode == "enriched_empty_raw_final":
        adapter_body = (
            "import json, sys\nreq = json.load(sys.stdin)\n"
            "print(json.dumps({'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', 'recommended_outcome': 'pass', 'raw_final': '   '}))\n"
        )
    elif mode == "enriched_invalid_json_raw_final":
        adapter_body = (
            "import json, sys\nreq = json.load(sys.stdin)\n"
            "print(json.dumps({'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', 'recommended_outcome': 'pass', 'raw_final': '{not-json'}))\n"
        )
    else:
        adapter_body = (
            "import json, sys\nreq = json.load(sys.stdin)\n"
            "print(json.dumps({'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', 'recommended_outcome': 'pass'}))\n"
        )
    adapter_bin = _write_fake_adapter(tmp_path, adapter_body)
    proc = _run_lifecycle(ws, shark_bin, adapter_bin)
    assert ws["output_path"].is_file(), proc.stderr

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    expected_failure_reasons = {
        "enriched_missing_raw_final": "omitted raw_final",
        "enriched_disagreeing_raw_final": "disagrees with host outcome",
        "enriched_empty_raw_final": "empty or non-string raw_final",
        "enriched_invalid_json_raw_final": "raw_final is not valid JSON",
    }
    if mode in expected_failure_reasons:
        assert not ws["apply_capture"].exists()
        assert record["dispatches"][0]["outcome"] == "worker_failure"
        assert expected_failure_reasons[mode] in record["outcome"]["reason"]
        logged = [json.loads(line) for line in ws["log_path"].read_text(encoding="utf-8").splitlines()]
        assert not any(cmd[0] == "run" and "--apply-result" in cmd for cmd in logged)
    else:
        assert ws["apply_capture"].read_text(encoding="utf-8") == raw_final_str
        assert record["outcome"]["terminal"] == "complete"


@pytest.mark.parametrize("exceed_stdout", [False, True])
def test_r26_selector_drains_large_stdout_stderr_and_enforces_stdout_bound(tmp_path, exceed_stdout):
    ws = _setup_workspace(tmp_path)
    shark_body = """
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists():
        print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "B001", "entity_type": "bug", "status": "code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    STATE_FILE.write_text("done", encoding="utf-8")
    print(json.dumps({"run_id": "r1", "to_status": "completed", "transitioned": True, "status": {"retirement_state": "retired"}}))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    if exceed_stdout:
        adapter_body = (
            "import sys\nsys.stdin.read()\n"
            "sys.stdout.write(' ' * (2 * 1024 * 1024 + 64))\n"
            "sys.stdout.flush()\n"
        )
    else:
        # Exceed 64KB OS pipe buffer on stderr AND write >100KB padded JSON on stdout.
        raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
        adapter_body = (
            "import sys\nsys.stdin.read()\n"
            "sys.stderr.write('E' * (128 * 1024))\n"
            "sys.stderr.flush()\n"
            f"sys.stdout.write({raw_final_str!r} + ' ' * (100 * 1024))\n"
            "sys.stdout.flush()\n"
        )
    adapter_bin = _write_fake_adapter(tmp_path, adapter_body)
    proc = _run_lifecycle(ws, shark_bin, adapter_bin)
    assert ws["output_path"].is_file(), proc.stderr

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    if exceed_stdout:
        assert record["dispatches"][0]["outcome"] == "worker_failure"
        assert "exceeded maximum transport bound" in record["outcome"]["reason"]
    else:
        assert record["outcome"]["terminal"] == "complete"


def test_r26_paid_rejection_preserves_cost_and_model(tmp_path):
    ws = _setup_workspace(tmp_path)
    shark_body = """
if argv[0] == "next":
    emit_next({
        "entity_key": "B001", "entity_type": "bug", "status": "code_review",
        "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
        "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
        "allowed_outcomes": ["pass", "fail"],
    })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    print("apply-result rejected: invalid gate schema", file=sys.stderr)
    sys.exit(1)
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
    adapter_body = (
        "import json, sys\nreq = json.load(sys.stdin)\n"
        "print(json.dumps({"
        "'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', "
        "'recommended_outcome': 'pass', 'cost_usd': 0.125, "
        "'usage': {'provider': 'codex', 'model': 'synthetic-paid-model', 'cost_usd': 0.125, 'input_tokens': 10, 'output_tokens': 5}, "
        f"'raw_final': {raw_final_str!r}"
        "}))\n"
    )
    adapter_bin = _write_fake_adapter(tmp_path, adapter_body)
    proc = _run_lifecycle(ws, shark_bin, adapter_bin, with_i05=True)
    assert ws["output_path"].is_file(), proc.stderr

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["limits"]["observed_cost_usd"] == pytest.approx(0.125)
    assert record["dispatches"][0]["cost_usd"] == pytest.approx(0.125)
    assert record["dispatches"][0]["transition"]["accepted"] is False
    assert record["stages"][0]["cost_usd"] == pytest.approx(0.125)
    assert record["stages"][0]["usage"]["model"] == "synthetic-paid-model"


def test_r26_descendant_gate_policy_captures_child_gate(tmp_path):
    workflows = {
        "epic": {
            "start": "active",
            "steps": {
                "active": {"action": "spawn_agent", "phase": "execution", "agent": "lead", "provider": "codex", "model": "m", "outcomes": {"pass": "completed"}},
                "completed": {"terminal": True},
            },
        },
        "feature": {
            "start": "active",
            "steps": {
                "active": {"action": "spawn_agent", "phase": "execution", "outcomes": {"pass": "completed"}},
                "completed": {"terminal": True},
            },
        },
        "task": {
            "start": "in_code_review",
            "steps": {
                "in_code_review": {
                    "action": "spawn_agent",
                    "phase": "code_review",
                    "agent": "reviewer",
                    "provider": "codex",
                    "model": "synthetic-paid-model",
                    "effort": "high",
                    "prompt": "review.md",
                    "outcomes": {"pass": "completed", "fail": "in_development"},
                },
                "in_development": {"action": "spawn_agent", "phase": "development", "outcomes": {"pass": "in_code_review"}},
                "completed": {"terminal": True},
            },
        },
    }
    ws = _setup_workspace(tmp_path, family="epic", root_key="E01", extra_workflows=workflows)
    shark_body = """
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists():
        print(json.dumps({"entity_key": "E01", "entity_type": "epic", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "T-E01-F01-001", "entity_type": "task", "status": "in_code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "codex", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review task", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    STATE_FILE.write_text("done", encoding="utf-8")
    print(json.dumps({"run_id": "r1", "to_status": "completed", "transitioned": True, "status": {"retirement_state": "retired"}}))
elif argv[0] == "get":
    print(json.dumps({"entity_key": "T-E01-F01-001", "notes": []}))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
    adapter_bin = _write_fake_adapter(
        tmp_path,
        f"import sys\nsys.stdin.read()\nprint({raw_final_str!r})\n",
    )
    proc = _run_lifecycle(ws, shark_bin, adapter_bin, with_i05=True)
    assert ws["output_path"].is_file(), proc.stderr

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["outcome"]["terminal"] == "complete"
    assert "in_code_review" in record["workflow_policy"]["enabled_gates"]
    reached_gates = [g for g in record["review_gates"] if g.get("gate_id") == "in_code_review" and g.get("state") != "not_reached"]
    assert len(reached_gates) == 1
    assert reached_gates[0]["round"] == 1
    assert reached_gates[0]["state"] == "zero_findings"


def test_r26_pending_stage_recovery_via_resume_run(tmp_path):
    """R26 step 4: interrupted apply recovers via --resume-run STAGE_ID --session SESSION before next/claim."""
    ws = _setup_workspace(tmp_path)
    shark_body = """
import signal
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists() and STATE_FILE.read_text(encoding="utf-8") == "resumed":
        print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "B001", "entity_type": "bug", "status": "code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "anthropic", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "run" and "--apply-result" in argv:
    idx = argv.index("--apply-result") + 1
    APPLY_CAPTURE.write_bytes(Path(argv[idx]).read_bytes())
    STATE_FILE.write_text("durably_stored", encoding="utf-8")
    os.kill(os.getppid(), signal.SIGKILL)
elif argv[0] == "run" and "--resume-run" in argv:
    stage_id = argv[argv.index("--resume-run") + 1]
    session = argv[argv.index("--session") + 1]
    assert STATE_FILE.read_text(encoding="utf-8") == "durably_stored"
    STATE_FILE.write_text("resumed", encoding="utf-8")
    print(json.dumps({
        "run_id": stage_id,
        "entity_key": "B001",
        "entity_type": "bug",
        "resume_action": "already_transitioned",
        "ingested": True,
        "to_status": "completed",
        "transitioned": False,
        "lease_released": True,
        "status": {"retirement_state": "retired"},
    }))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
    adapter_body = (
        "import json, sys\nreq = json.load(sys.stdin)\n"
        "print(json.dumps({"
        "'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', "
        "'recommended_outcome': 'pass', 'cost_usd': 0.01, "
        "'usage': {'provider': 'anthropic', 'model': 'synthetic-paid-model', 'cost_usd': 0.01, 'input_tokens': 10, 'output_tokens': 5}, "
        "'provider_usage_envelope': {'usage': {'input_tokens': 10, 'output_tokens': 5, 'cache_creation_input_tokens': 0, 'cache_read_input_tokens': 0}, 'total_cost_usd': 0.01, 'modelUsage': {'synthetic-paid-model': {}}, 'duration_api_ms': 100, 'num_turns': 1, 'session_id': 'prov-sess-1'}, "
        f"'raw_final': {raw_final_str!r}"
        "}))\n"
    )
    adapter_bin = _write_fake_adapter(tmp_path, adapter_body)

    proc1 = _run_lifecycle(ws, shark_bin, adapter_bin, run_id="r26-resume-run")
    assert proc1.returncode != 0
    pending_path = ws["scratch"] / ".shark" / "e40-apply" / "pending.json"
    assert pending_path.is_file()
    pending_data = json.loads(pending_path.read_text(encoding="utf-8"))
    expected_stage_id = pending_data["stage_id"]

    ws["log_path"].unlink()
    proc2 = _run_lifecycle(ws, shark_bin, adapter_bin, run_id="r26-resume-run")
    assert proc2.returncode == 0, proc2.stderr
    assert not pending_path.exists()

    logged = [json.loads(line) for line in ws["log_path"].read_text(encoding="utf-8").splitlines()]
    assert logged[0][0] == "run"
    assert "--resume-run" in logged[0]
    assert logged[0][logged[0].index("--resume-run") + 1] == expected_stage_id
    assert logged[0][logged[0].index("--session") + 1] == "sess-1"
    assert not any(cmd[0] == "run" and "--apply-result" in cmd for cmd in logged)
    assert not any(cmd[0] == "claim" for cmd in logged)

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["outcome"]["terminal"] == "complete"
    assert record["dispatches"][0]["transition"]["run_id"] == expected_stage_id
    assert record["dispatches"][0]["transition"]["resumed"] is True
    assert record["dispatches"][0]["transition"]["to_status"] == "completed"
    _verify_lifecycle_output(ws["output_path"])


def _verify_lifecycle_output(output_path: Path):
    proc = subprocess.run(
        [
            str(SCRIPTS / "verify-lifecycle-run.sh"),
            str(output_path),
            "--schema",
            str(REPO_ROOT / "bench" / "runs" / "i07-schema.yaml"),
        ],
        capture_output=True,
        text=True,
    )
    assert proc.returncode == 0, f"verify-lifecycle-run.sh failed:\nstdout: {proc.stdout}\nstderr: {proc.stderr}"
    verdict = json.loads(proc.stdout)
    assert verdict["result"] == "accepted"


def test_r26_pending_stage_recovery_unsubmitted_via_apply_result(tmp_path):
    """R26 step 4: interrupted before durable apply recovers via unsubmitted pending.json -> --apply-result."""
    ws = _setup_workspace(tmp_path)
    shark_body = """
import signal
STATE_FILE = LOG_PATH.parent / "state.txt"
if argv[0] == "next":
    if STATE_FILE.exists() and STATE_FILE.read_text(encoding="utf-8") == "applied":
        print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "completed", "action": "archive", "terminal": True}))
    else:
        emit_next({
            "entity_key": "B001", "entity_type": "bug", "status": "code_review", "phase": "code_review",
            "action": "spawn_agent", "agent_type": "reviewer", "provider": "anthropic", "model": "synthetic-paid-model",
            "effort": "high", "prompt": "Review it", "result_contract": "gate_result_v1",
            "allowed_outcomes": ["pass", "fail"],
        })
elif argv[0] == "claim":
    print(json.dumps({"session_id": "sess-1"}))
elif argv[0] == "heartbeat":
    print(json.dumps({"ok": True}))
elif argv[0] == "get":
    print(json.dumps({"entity_key": "B001", "entity_type": "bug", "status": "code_review"}))
elif argv[0] == "run" and "--apply-result" in argv:
    if not STATE_FILE.exists():
        # Simulate crash before durable storage on first attempt
        STATE_FILE.write_text("crashed_unsubmitted", encoding="utf-8")
        os.kill(os.getppid(), signal.SIGKILL)
        sys.exit(0)
    idx = argv.index("--apply-result") + 1
    APPLY_CAPTURE.write_bytes(Path(argv[idx]).read_bytes())
    STATE_FILE.write_text("applied", encoding="utf-8")
    print(json.dumps({
        "run_id": argv[argv.index("--run-id") + 1],
        "entity_key": "B001",
        "entity_type": "bug",
        "to_status": "completed",
        "transitioned": True,
        "lease_released": True,
        "status": {"retirement_state": "retired"},
    }))
elif argv[0] == "run" and "--resume-run" in argv:
    if STATE_FILE.exists() and STATE_FILE.read_text(encoding="utf-8") == "crashed_unsubmitted":
        sys.stderr.write("run not found\\n")
        raise SystemExit(1)
    print(json.dumps({}))
elif argv[0] == "release":
    print(json.dumps({"released": True}))
else:
    print(json.dumps({}))
"""
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], shark_body)
    raw_final_str = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
    adapter_body = (
        "import json, sys\nreq = json.load(sys.stdin)\n"
        "print(json.dumps({"
        "'worker_id': 'w1', 'session_id': req['session_id'], 'kind': 'final', "
        "'recommended_outcome': 'pass', 'cost_usd': 0.01, "
        "'usage': {'provider': 'anthropic', 'model': 'synthetic-paid-model', 'cost_usd': 0.01, 'input_tokens': 10, 'output_tokens': 5}, "
        "'provider_usage_envelope': {'usage': {'input_tokens': 10, 'output_tokens': 5, 'cache_creation_input_tokens': 0, 'cache_read_input_tokens': 0}, 'total_cost_usd': 0.01, 'modelUsage': {'synthetic-paid-model': {}}, 'duration_api_ms': 100, 'num_turns': 1, 'session_id': 'prov-sess-1'}, "
        f"'raw_final': {raw_final_str!r}"
        "}))\n"
    )
    adapter_bin = _write_fake_adapter(tmp_path, adapter_body)

    proc1 = _run_lifecycle(ws, shark_bin, adapter_bin, run_id="r26-unsubmitted")
    assert proc1.returncode != 0
    pending_path = ws["scratch"] / ".shark" / "e40-apply" / "pending.json"
    log_txt = ws["log_path"].read_text(encoding="utf-8") if ws["log_path"].exists() else "<no log>"
    assert pending_path.is_file(), f"rc={proc1.returncode} stdout={proc1.stdout}\nstderr={proc1.stderr}\nlog={log_txt}"
    pending_data = json.loads(pending_path.read_text(encoding="utf-8"))
    assert pending_data.get("submitted") is False
    expected_stage_id = pending_data["stage_id"]

    ws["log_path"].unlink()
    proc2 = _run_lifecycle(ws, shark_bin, adapter_bin, run_id="r26-unsubmitted")
    assert proc2.returncode == 0, proc2.stderr
    assert not pending_path.exists()

    logged = [json.loads(line) for line in ws["log_path"].read_text(encoding="utf-8").splitlines()]
    assert logged[0][0] == "run" and "--resume-run" in logged[0]
    assert any(cmd[0] == "get" and cmd[1] == "B001" for cmd in logged)
    apply_cmds = [cmd for cmd in logged if cmd[0] == "run" and "--apply-result" in cmd]
    assert len(apply_cmds) == 1, f"logged={logged}"
    assert apply_cmds[0][apply_cmds[0].index("--run-id") + 1] == expected_stage_id
    assert not any(cmd[0] == "claim" for cmd in logged)

    record = json.loads(ws["output_path"].read_text(encoding="utf-8"))
    assert record["outcome"]["terminal"] == "complete"
    assert record["dispatches"][0]["transition"]["run_id"] == expected_stage_id
    assert record["dispatches"][0]["transition"]["resumed"] is True
    assert record["dispatches"][0]["transition"]["resume_action"] == "initial_apply_after_unsubmitted"
    assert record["dispatches"][0]["transition"]["to_status"] == "completed"
    _verify_lifecycle_output(ws["output_path"])


def test_r26_pending_stage_recovery_rejects_mismatched_run_id(tmp_path):
    """R26 step 4: a different run_id cannot resume or clear another run's pending stage."""
    ws = _setup_workspace(tmp_path)
    pending_path = ws["scratch"] / ".shark" / "e40-apply" / "pending.json"
    pending_path.parent.mkdir(parents=True, exist_ok=True)
    pending_path.write_text(
        json.dumps({
            "run_id": "other-run-id",
            "stage_id": "e40s-1234",
            "entity_key": "B001",
            "entity_type": "bug",
            "source_status": "code_review",
            "session_id": "sess-1",
            "ordinal": 1,
            "outcome": "pass",
        }),
        encoding="utf-8",
    )
    shark_bin = _write_fake_shark(tmp_path, ws["log_path"], ws["apply_capture"], "print(json.dumps({}))\n")
    adapter_bin = _write_fake_adapter(tmp_path, "import sys\nsys.stdin.read()\nprint('{}')\n")

    proc = _run_lifecycle(ws, shark_bin, adapter_bin, run_id="new-run-id")
    assert proc.returncode != 0
    assert pending_path.is_file()
    assert not ws["log_path"].exists()
