"""Production route materialization boundary tests for B083."""

import json
import os
import shutil
import subprocess
import sys
from contextlib import redirect_stdout
from importlib.util import module_from_spec, spec_from_file_location
from io import StringIO
from pathlib import Path

import pytest
import yaml

REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))
TEST_LIB = Path(__file__).resolve().parent / "lib"
sys.path.insert(0, str(TEST_LIB))
from tc122_workflow_fixtures import create_sample_workflow_bundle, scratch_dir
from normalize_workflow_routes import normalize_workflow_routes

from bench.scripts.lib.e40_benchmark import (
    DEFAULT_PROFILES_DIR,
    OperatorError,
    load_routing_profile,
    load_yaml,
    workflow_routing_identity,
    write_yaml,
)
import bench.scripts.lib.e40_benchmark as e40_benchmark


def _invoke_materializer(script, workflow_root, profile_path, *, environment=None):
    return subprocess.run(
        [
            sys.executable,
            str(script),
            "--workflow-root",
            str(workflow_root),
            "--profile",
            str(profile_path),
        ],
        text=True,
        capture_output=True,
        check=False,
        env=environment,
    )


def _load_materializer_module():
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    spec = spec_from_file_location("materialize_routing_profile_test", script)
    assert spec is not None and spec.loader is not None
    module = module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


CODEX_FAILURE_STAGES = ("development", "review", "discovery", "specification", "qa")


def _codex_failure_profile():
    return {
        "schema_version": "1.0",
        "profile_id": "codex-test",
        "description": "offline catalog fail-closed fixture",
        "provider": "codex",
        "routes": {
            stage: {"model": "b083-synthetic-model", "effort": "high"}
            for stage in CODEX_FAILURE_STAGES
        },
    }


def _assert_codex_catalog_failure(result):
    assert result["resolution"] == "failed"
    assert result["cause_class"] == "route_defect"
    assert "Codex model discovery failed" in result["reason"]
    assert "selected Codex CLI executable is unavailable" in result["reason"]
    assert result["dispatch_count"] == 0
    assert result["requested_routes"] == [
        {
            "stage": stage,
            "provider": "codex",
            "model": "b083-synthetic-model",
            "effort": "high",
        }
        for stage in sorted(CODEX_FAILURE_STAGES)
    ]


def test_b083_route_materializer_cli_applies_profile_to_run_copy(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    assert script.is_file(), "B083 requires a production route materialization entrypoint"
    source = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    source_prompt = load_yaml(source / "task.yaml", "source task workflow")["steps"][
        "development"
    ]["prompt"]
    before = {p.name: p.read_bytes() for p in source.glob("*.yaml")}
    run_copy = scratch_dir / "run-scoped-workflow"
    shutil.copytree(source, run_copy)
    completed = _invoke_materializer(
        script, run_copy, DEFAULT_PROFILES_DIR / "copilot-balanced.yaml"
    )

    assert completed.returncode == 0, completed.stdout + completed.stderr
    result = json.loads(completed.stdout)
    assert result["profile_id"] == "copilot-balanced"
    assert result["profile_schema_version"] == "1.0"
    assert result["effective_routes"]["task.yaml::development"] == {
        "provider": "copilot",
        "model": "claude-sonnet-5",
        "effort": "high",
    }
    materialized_task = load_yaml(run_copy / "task.yaml", "materialized task workflow")
    assert materialized_task["steps"]["development"]["prompt"] == source_prompt
    assert result["provider_routing"] == workflow_routing_identity(run_copy)
    assert {p.name: p.read_bytes() for p in source.glob("*.yaml")} == before


def test_b083_profile_omitted_non_codex_effort_preserves_workflow_value(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    source = create_sample_workflow_bundle(
        scratch_dir, provider="anthropic", model="source-model", effort="xhigh"
    )
    run_copy = scratch_dir / "run-scoped-effort-workflow"
    shutil.copytree(source, run_copy)
    task_path = run_copy / "task.yaml"
    task_before = load_yaml(task_path, "task workflow")
    task_before["status_metadata"] = {
        "development": {"phase": "legacy display metadata"}
    }
    write_yaml(task_path, task_before)
    profile = load_routing_profile(DEFAULT_PROFILES_DIR / "copilot-balanced.yaml")
    profile["routes"]["development"].pop("effort")
    profile_path = scratch_dir / "copilot-profile-with-omitted-effort.yaml"
    write_yaml(profile_path, profile)

    completed = _invoke_materializer(script, run_copy, profile_path)

    assert completed.returncode == 0, completed.stdout + completed.stderr
    result = json.loads(completed.stdout)
    task = load_yaml(run_copy / "task.yaml", "materialized task workflow")
    assert task["steps"]["development"]["provider"] == "copilot"
    assert task["steps"]["development"]["model"] == "claude-sonnet-5"
    assert task["steps"]["development"]["effort"] == "xhigh"
    assert task["status_metadata"]["development"] == {
        "phase": "legacy display metadata"
    }
    assert result["effective_routes"]["task.yaml::development"]["effort"] == "xhigh"


def test_b083_materializer_updates_nested_legacy_route_and_preserves_prompt(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    workflow_dir = scratch_dir / "legacy-workflow"
    workflow_dir.mkdir()
    task_path = workflow_dir / "task.yaml"
    write_yaml(task_path, {
        "status_metadata": {
            "development": {
                "phase": "development",
                "orchestrator_action": {
                    "action": "spawn_agent",
                    "provider": "anthropic",
                    "model": "source-model",
                    "effort": "low",
                    "instruction_template": "Implement {task_id}: {title}",
                },
            }
        }
    })
    profile_path = scratch_dir / "legacy-profile.yaml"
    write_yaml(profile_path, {
        "schema_version": "1.0",
        "profile_id": "legacy-copilot-route",
        "description": "Copilot route for legacy workflow fixture",
        "provider": "copilot",
        "routes": {
            "development": {"model": "synthetic-model", "effort": "high"}
        },
    })

    completed = _invoke_materializer(script, workflow_dir, profile_path)

    assert completed.returncode == 0, completed.stdout + completed.stderr
    result = json.loads(completed.stdout)
    action = load_yaml(task_path, "materialized legacy workflow")["status_metadata"][
        "development"
    ]["orchestrator_action"]
    assert action == {
        "action": "spawn_agent",
        "provider": "copilot",
        "model": "synthetic-model",
        "effort": "high",
        "instruction_template": "Implement {task_id}: {title}",
    }
    assert result["effective_routes"]["task.yaml::development"] == {
        "provider": "copilot",
        "model": "synthetic-model",
        "effort": "high",
    }
    assert result["provider_routing"] == workflow_routing_identity(workflow_dir)


def test_b083_materializer_records_invalid_utf8_workflow_as_route_defect(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    workflow_dir = scratch_dir / "invalid-utf8-workflow"
    workflow_dir.mkdir()
    (workflow_dir / "task.yaml").write_bytes(b"steps:\n  development: \xff\n")

    completed = subprocess.run(
        [sys.executable, str(script), "--workflow-root", str(workflow_dir)],
        text=True,
        capture_output=True,
        check=False,
    )

    assert completed.returncode == 1
    result = json.loads(completed.stdout)
    assert result["resolution"] == "failed"
    assert result["cause_class"] == "route_defect"
    assert "cannot read workflow file" in result["reason"]


def test_b083_route_materializer_records_codex_catalog_failure_before_apply(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    assert script.is_file(), "B083 requires a production route materialization entrypoint"
    source = create_sample_workflow_bundle(scratch_dir)
    run_copy = scratch_dir / "codex-run-workflow"
    shutil.copytree(source, run_copy)
    profile_path = scratch_dir / "codex-profile.yaml"
    write_yaml(profile_path, _codex_failure_profile())
    before = {p.name: p.read_bytes() for p in run_copy.glob("*.yaml")}

    completed = _invoke_materializer(
        script,
        run_copy,
        profile_path,
        environment={
            **os.environ,
            "E40_CODEX_COMMAND": json.dumps([str(scratch_dir / "missing-codex")]),
        },
    )

    assert completed.returncode == 1
    result = json.loads(completed.stdout)
    _assert_codex_catalog_failure(result)
    assert {p.name: p.read_bytes() for p in run_copy.glob("*.yaml")} == before


def test_b083_materializer_reports_route_write_oserror_as_structured_failure(
    scratch_dir, monkeypatch
):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="anthropic", model="source-model", effort="low"
    )
    materializer = _load_materializer_module()

    def fail_route_write(*_args, **_kwargs):
        raise OSError("simulated workflow write failure")

    monkeypatch.setattr(materializer, "write_yaml", fail_route_write)
    output = StringIO()
    with redirect_stdout(output):
        return_code = materializer.main(
            ["--workflow-root", str(workflow_dir), "--allowed-root", str(scratch_dir)]
        )

    result = json.loads(output.getvalue())
    assert return_code == 1
    assert result["resolution"] == "failed"
    assert result["cause_class"] == "route_defect"
    assert "simulated workflow write failure" in result["reason"]
    assert result["dispatch_count"] == 0
    assert result["requested_routes"]


def test_b083_inherited_route_failure_keeps_requested_tuple_in_record(scratch_dir):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    workflow_dir = create_sample_workflow_bundle(scratch_dir)
    profile_path = scratch_dir / "codex-profile.yaml"
    write_yaml(
        profile_path,
        {
            "schema_version": "1.0",
            "profile_id": "codex-discovery-failed",
            "description": "Catalog discovery failure fixture.",
            "provider": "codex",
            "routes": {"development": {"model": "synthetic-model", "effort": "high"}},
        },
    )
    error = "route_defect: Codex model discovery failed: fixture unavailable"

    completed = _invoke_materializer(
        script,
        workflow_dir,
        profile_path,
        environment={**os.environ, "E40_ROUTE_MATERIALIZATION_ERROR": error},
    )

    assert completed.returncode == 1
    failure = json.loads(completed.stdout)
    assert failure["reason"] == error
    assert failure["cause_class"] == "route_defect"
    assert failure["dispatch_count"] == 0
    assert failure["requested_routes"] == [
        {"stage": "development", "provider": "codex", "model": "synthetic-model", "effort": "high"}
    ]


@pytest.mark.parametrize(
    "routes",
    ["not-a-mapping", {"development": 7}],
    ids=["routes-not-a-mapping", "route-entry-not-a-mapping"],
)
def test_b083_route_materializer_reports_malformed_profile_shape(scratch_dir, routes):
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    workflow_dir = create_sample_workflow_bundle(scratch_dir)
    profile_path = scratch_dir / "malformed-profile.yaml"
    write_yaml(
        profile_path,
        {
            "schema_version": "1.0",
            "profile_id": "malformed-profile",
            "description": "Malformed profile fixture",
            "provider": "copilot",
            "routes": routes,
        },
    )

    completed = _invoke_materializer(script, workflow_dir, profile_path)

    assert completed.returncode == 1
    result = json.loads(completed.stdout)
    assert result["resolution"] == "failed"
    assert result["cause_class"] == "route_defect"
    assert "cannot select invalid routing profile" in result["reason"]
    assert result["dispatch_count"] == 0


def test_test_route_normalizer_rejects_paths_outside_allowed_root(tmp_path):
    allowed = tmp_path / "allowed"
    outside = tmp_path / "outside"
    allowed.mkdir()
    outside.mkdir()
    workflow_path = outside / "workflow.yaml"
    workflow_path.write_text("steps: {}\n", encoding="utf-8")

    with pytest.raises(ValueError, match="outside the allowed test scratch root"):
        normalize_workflow_routes(outside, allowed)

    assert workflow_path.read_text(encoding="utf-8") == "steps: {}\n"


def test_test_route_normalizer_rejects_non_mapping_yaml(tmp_path):
    workflow = tmp_path / "workflow"
    workflow.mkdir()
    (workflow / "workflow.yaml").write_text("- not a mapping\n", encoding="utf-8")

    with pytest.raises(ValueError, match="document must be a mapping"):
        normalize_workflow_routes(workflow, tmp_path)
