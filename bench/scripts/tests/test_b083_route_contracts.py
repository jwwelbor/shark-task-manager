"""Codex routing profile and route-contract tests for B083."""

import json
import subprocess
from pathlib import Path
import sys

import pytest

REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))
TEST_LIB = Path(__file__).resolve().parent / "lib"
sys.path.insert(0, str(TEST_LIB))
from tc122_workflow_fixtures import create_sample_workflow_bundle, scratch_dir

from bench.scripts.lib.e40_benchmark import (
    DEFAULT_PROFILES_DIR,
    OperatorError,
    apply_selected_routing_profile,
    apply_routing_profile,
    load_routing_profile,
    load_yaml,
    validate_routing_profile_schema,
    workflow_routing_identity,
    write_yaml,
)
import bench.scripts.lib.e40_benchmark as e40_benchmark


def test_b083_profile_rejects_missing_live_stage_before_mutation(scratch_dir):
    profile = load_routing_profile(DEFAULT_PROFILES_DIR / "copilot-balanced.yaml")
    del profile["routes"]["review"]
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    before = {p.name: p.read_bytes() for p in workflow_dir.glob("*.yaml")}

    with pytest.raises(OperatorError, match="omits live workflow stage.*review"):
        apply_routing_profile(profile, workflow_dir)

    assert {p.name: p.read_bytes() for p in workflow_dir.glob("*.yaml")} == before


def _remove_development_route(workflow_dir):
    task_path = workflow_dir / "task.yaml"
    task = load_yaml(task_path, "task workflow")
    task["steps"]["development"].pop("provider")
    task["steps"]["development"].pop("model")
    write_yaml(task_path, task)


def _complete_codex_profile():
    return {
        "schema_version": "1.0",
        "profile_id": "codex-complete-route",
        "description": "Complete route fixture for B083",
        "provider": "codex",
        "routes": {
            stage: {"model": "b083-synthetic-model", "effort": "high"}
            for stage in ("development", "review", "discovery", "specification", "qa")
        },
    }


def test_b083_codex_profile_accepts_effort_names_from_runtime_catalog():
    profile = _complete_codex_profile()
    profile["routes"]["development"]["effort"] = "ultra"

    assert validate_routing_profile_schema(profile) == []


@pytest.mark.parametrize("profile_id", ["profile\x1fother", "profile\nother"])
def test_b083_routing_profile_rejects_control_characters_in_profile_id(profile_id):
    profile = _complete_codex_profile()
    profile["profile_id"] = profile_id

    errors = validate_routing_profile_schema(profile)

    assert any("profile_id must not contain control characters" in error for error in errors)


def test_b083_no_profile_rejects_source_stage_missing_provider(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    _remove_development_route(workflow_dir)

    with pytest.raises(OperatorError, match="has no explicit provider"):
        e40_benchmark.live_workflow_routes(workflow_dir)


def test_b083_batch_workflow_root_rejects_symlink_escape(scratch_dir):
    trusted_scratch = scratch_dir / "trusted-scratch"
    trusted_scratch.mkdir()
    outside_workflow = scratch_dir / "outside-workflow"
    outside_workflow.mkdir()
    workflow_link = trusted_scratch / "workflow"
    workflow_link.symlink_to(outside_workflow, target_is_directory=True)

    with pytest.raises(OperatorError, match="resolve inside the trusted scratch template"):
        e40_benchmark._relative_batch_workflow_root(workflow_link, trusted_scratch)


def test_b083_no_profile_resolves_legacy_status_metadata_routes(scratch_dir):
    workflow_dir = scratch_dir / "workflow"
    workflow_dir.mkdir()
    write_yaml(workflow_dir / "task.yaml", {
        "status_metadata": {
            "development": {
                "phase": "development",
                "orchestrator_action": {
                    "action": "spawn_agent",
                    "provider": "codex",
                    "instruction_template": "Implement {task_id}: {title}",
                },
            }
        }
    })

    routes = e40_benchmark.live_workflow_routes(workflow_dir)

    assert routes == {
        "task.yaml::development": {
            "source": "task.yaml",
            "stage": "development",
            "provider": "codex",
            "model": None,
            "effort": None,
        }
    }


def test_b083_explicit_routes_keep_workflow_file_identity_and_ignore_other_families(scratch_dir):
    workflow_dir = scratch_dir / "workflow"
    workflow_dir.mkdir()
    routes = {
        "bug.yaml": ("anthropic", "bug-synthetic-route"),
        "feature.yaml": ("copilot", "feature-synthetic-route"),
    }
    for filename, (provider, model) in routes.items():
        write_yaml(workflow_dir / filename, {
            "steps": {
                "development": {
                    "action": "spawn_agent",
                    "provider": provider,
                    "model": model,
                    "effort": "high",
                }
            }
        })
    write_yaml(workflow_dir / "question.yaml", {
        "steps": {"open": {"action": "spawn_agent"}}
    })

    result = subprocess.run(
        [
            sys.executable,
            str(REPO_ROOT / "bench/scripts/materialize-routing-profile.py"),
            "--workflow-root", str(workflow_dir),
            "--family", "bug",
            "--family", "feature",
        ],
        text=True,
        capture_output=True,
        check=False,
    )

    assert result.returncode == 0, result.stdout + result.stderr
    identity = json.loads(result.stdout)
    effective = identity["effective_routes"]
    assert effective["bug.yaml::development"]["model"] == "bug-synthetic-route"
    assert effective["feature.yaml::development"]["model"] == "feature-synthetic-route"
    assert load_yaml(workflow_dir / "bug.yaml", "bug") ["steps"]["development"]["model"] == "bug-synthetic-route"
    assert load_yaml(workflow_dir / "feature.yaml", "feature")["steps"]["development"]["model"] == "feature-synthetic-route"
    assert load_yaml(workflow_dir / "question.yaml", "question")["steps"]["open"] == {
        "action": "spawn_agent"
    }


def test_b083_selected_profile_supplies_routes_missing_from_source_workflow(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    _remove_development_route(workflow_dir)
    profile = _complete_codex_profile()
    result = apply_selected_routing_profile(
        profile,
        workflow_dir,
        validate_catalogs=False,
    )

    expected_routes = {
        stage: {"provider": "codex", "model": "b083-synthetic-model", "effort": "high"}
        for stage in ("development", "review", "discovery", "specification", "qa")
    }
    assert result["effective_routes"] == expected_routes

    materialized_routes = {}
    for workflow_path in workflow_dir.glob("*.yaml"):
        workflow = load_yaml(workflow_path, "materialized workflow")
        for stage, step in workflow["steps"].items():
            if isinstance(step, dict) and step.get("action") == "spawn_agent":
                materialized_routes[stage] = {
                    key: step[key] for key in ("provider", "model", "effort")
                }
    assert materialized_routes == expected_routes


def test_b083_codex_profile_rejects_claude_model_before_materialization(scratch_dir, monkeypatch):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    profile = {
        "schema_version": "1.0",
        "profile_id": "codex-incompatible-model",
        "description": "Codex must reject a Claude model before route materialization.",
        "provider": "codex",
        "routes": {
            stage: {"model": "claude-sonnet-5", "effort": "high"}
            for stage in ("development", "review", "discovery", "specification", "qa")
        },
    }
    cli_catalog = {"models": ["fixture-codex-model-b"]}
    before = {path.name: path.read_bytes() for path in workflow_dir.glob("*.yaml")}
    materializer_calls = []

    def record_materialization(*args, **kwargs):
        materializer_calls.append((args, kwargs))

    monkeypatch.setattr(e40_benchmark, "apply_validated_routing_profile", record_materialization)
    with pytest.raises(OperatorError, match="CLI compatibility catalog does not contain pinned model 'claude-sonnet-5'"):
        apply_selected_routing_profile(
            profile,
            workflow_dir,
            cli_catalog=cli_catalog,
        )

    assert materializer_calls == []
    assert {path.name: path.read_bytes() for path in workflow_dir.glob("*.yaml")} == before


def test_b083_validated_profile_rejects_bad_route_map_before_any_workflow_write(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    profile = {
        "schema_version": "1.0",
        "profile_id": "partial-profile",
        "description": "A direct validated-helper call must not partially write files.",
        "provider": "copilot",
        "routes": {
            stage: {"model": "gemini-3.8-flash", "effort": "high"}
            for stage in ("discovery", "specification", "qa")
        },
    }
    effective_routes = {
        stage: {"provider": "copilot", "model": "gemini-3.8-flash", "effort": "high"}
        for stage in ("development", "review", "discovery", "specification", "qa")
    }
    before = {path.name: path.read_bytes() for path in workflow_dir.glob("*.yaml")}

    with pytest.raises(OperatorError, match="validated routes must be present in the selected profile"):
        e40_benchmark.apply_validated_routing_profile(
            profile, effective_routes, workflow_dir
        )

    assert {path.name: path.read_bytes() for path in workflow_dir.glob("*.yaml")} == before


def test_b083_selected_profile_validates_route_coverage_once(scratch_dir, monkeypatch):
    profile = load_routing_profile(DEFAULT_PROFILES_DIR / "copilot-balanced.yaml")
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    original = e40_benchmark.validate_profile_coverage
    calls = 0

    def count_coverage(*args, **kwargs):
        nonlocal calls
        calls += 1
        return original(*args, **kwargs)

    monkeypatch.setattr(e40_benchmark, "validate_profile_coverage", count_coverage)
    apply_selected_routing_profile(profile, workflow_dir)

    assert calls == 1


def test_b083_profile_identity_uses_effective_routes_and_preserves_source(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    source_bytes = {p.name: p.read_bytes() for p in workflow_dir.glob("*.yaml")}
    original = workflow_routing_identity(workflow_dir)
    profile_path = DEFAULT_PROFILES_DIR / "copilot-balanced.yaml"
    effective = e40_benchmark._materialize_route_identity(
        workflow_dir, profile_path, None, None
    )

    assert effective["provider_routing"]["routing_digest"] != original["routing_digest"]
    assert effective["provider_routing"]["model_digest"] != original["model_digest"]
    assert effective["profile_id"] == "copilot-balanced"
    assert {p.name: p.read_bytes() for p in workflow_dir.glob("*.yaml")} == source_bytes


def test_b083_cli_catalog_match_does_not_claim_runtime_access():
    validator = getattr(e40_benchmark, "validate_codex_catalogs", None)
    assert callable(validator), "B083 requires CLI catalog compatibility validation"
    cli_catalog = {"models": ["b083-synthetic-model"]}
    assert validator("b083-synthetic-model", cli_catalog, None) == {
        "cli_compatibility": "catalog_match",
        "runtime_access": "unverified",
    }
    with pytest.raises(OperatorError, match="does not contain pinned model"):
        validator("b083-synthetic-model", {"models": ["b083-other-model"]}, None)


def test_b083_materializer_rejects_workflow_root_outside_allowed_copy(tmp_path):
    outside = tmp_path / "outside-workflow"
    outside.mkdir()
    sentinel = outside / "bug.yaml"
    sentinel.write_text("steps: {}\n", encoding="utf-8")
    allowed = tmp_path / "isolated-copy"
    allowed.mkdir()
    materializer = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"

    result = subprocess.run(
        [
            sys.executable,
            str(materializer),
            "--workflow-root",
            str(outside),
            "--allowed-root",
            str(allowed),
        ],
        text=True,
        capture_output=True,
        check=False,
    )

    assert result.returncode == 1
    assert "outside the allowed isolated workflow copy" in result.stdout
    assert sentinel.read_text(encoding="utf-8") == "steps: {}\n"


def test_b083_account_catalog_is_not_a_pre_dispatch_requirement():
    validator = e40_benchmark.validate_codex_catalogs
    caller_asserted_catalog = {
        "schema_version": "1.0",
        "same_account": True,
        "retrieved_at": e40_benchmark.utc_now(),
        "models": ["b083-synthetic-model"],
    }

    assert validator("b083-synthetic-model", {"models": ["b083-synthetic-model"]}, caller_asserted_catalog)["runtime_access"] == "unverified"


@pytest.mark.parametrize(
    ("field", "value", "message"),
    [
        ("provider", "unregistered-provider", "unsupported provider"),
        ("effort", "unregistered-effort", "unsupported effort"),
        ("effort", 7, "effort must be a string"),
    ],
)
def test_b083_unprofiled_workflow_rejects_invalid_route_enums(
    scratch_dir, field, value, message
):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    path = workflow_dir / "task.yaml"
    document = load_yaml(path, "task workflow")
    document["steps"]["development"][field] = value
    write_yaml(path, document)

    with pytest.raises(OperatorError, match=message):
        e40_benchmark.live_workflow_routes(workflow_dir)


@pytest.mark.parametrize("model", [7, "", {}, []])
def test_b083_codex_workflow_rejects_malformed_explicit_model(scratch_dir, model):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="codex", model="synthetic-codex-pin"
    )
    path = workflow_dir / "task.yaml"
    document = load_yaml(path, "task workflow")
    document["steps"]["development"]["model"] = model
    write_yaml(path, document)

    with pytest.raises(OperatorError, match="model must be a non-empty string when supplied"):
        e40_benchmark.live_workflow_routes(workflow_dir)


def test_b083_codex_workflow_rejects_empty_explicit_effort(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="codex", model="synthetic-codex-pin"
    )
    path = workflow_dir / "task.yaml"
    document = load_yaml(path, "task workflow")
    document["steps"]["development"]["effort"] = ""
    write_yaml(path, document)

    with pytest.raises(OperatorError, match="effort must be a non-empty string"):
        e40_benchmark.live_workflow_routes(workflow_dir)


def test_b083_codex_workflow_preserves_structural_model_omission(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="codex", model="synthetic-codex-pin"
    )
    path = workflow_dir / "task.yaml"
    document = load_yaml(path, "task workflow")
    document["steps"]["development"].pop("model")
    write_yaml(path, document)

    routes = e40_benchmark.live_workflow_routes(workflow_dir)

    assert routes["task.yaml::development"]["model"] is None


def test_b083_profile_catalog_accepts_model_named_unavailable(scratch_dir):
    workflow_dir = create_sample_workflow_bundle(
        scratch_dir, provider="claude", model="claude-sonnet-4", effort="low"
    )
    stages = e40_benchmark.live_workflow_stages(workflow_dir)
    profile = {
        "schema_version": "1.0",
        "profile_id": "synthetic-codex-profile",
        "description": "Exercise an explicit catalog ID matching the old sentinel.",
        "provider": "codex",
        "routes": {
            stage: {"model": "unavailable", "effort": "low"} for stage in stages
        },
    }

    effective = e40_benchmark.validate_profile_coverage(
        profile, workflow_dir, cli_catalog={"models": ["unavailable"]}
    )

    assert {route["model"] for route in effective.values()} == {"unavailable"}


def test_b083_route_identity_distinguishes_literal_unavailable_from_omission(tmp_path):
    workflow_path = tmp_path / "workflow.yaml"
    workflow = {
        "steps": {
            "development": {"action": "spawn_agent", "provider": "codex"}
        }
    }
    write_yaml(workflow_path, workflow)
    omitted = workflow_routing_identity(workflow_path)

    workflow["steps"]["development"]["model"] = "unavailable"
    workflow["steps"]["development"]["effort"] = "unavailable"
    write_yaml(workflow_path, workflow)
    explicit = workflow_routing_identity(workflow_path)

    assert explicit["model_digest"] != omitted["model_digest"]
    assert explicit["effort_digest"] != omitted["effort_digest"]
    assert explicit["routes"][0]["model"] == "unavailable"


def test_b083_catalog_compatibility_is_independent_of_account_metadata():
    validator = getattr(e40_benchmark, "validate_codex_catalogs", None)
    assert callable(validator), "B083 requires a separately reported catalog validator"
    cli_catalog = {"models": ["b083-synthetic-model"]}
    assert validator("b083-synthetic-model", cli_catalog, {"malformed": True})["cli_compatibility"] == "catalog_match"


def test_b083_codex_catalog_does_not_require_account_timestamp():
    validator = getattr(e40_benchmark, "validate_codex_catalogs", None)
    assert callable(validator), "B083 requires a separately reported catalog validator"
    account_catalog = {
        "schema_version": "1.0",
        "same_account": True,
        "retrieved_at": "2026-11-18T09:00:00",
        "models": ["b083-synthetic-model"],
    }

    assert validator("b083-synthetic-model", {"models": ["b083-synthetic-model"]}, account_catalog)["cli_compatibility"] == "catalog_match"
