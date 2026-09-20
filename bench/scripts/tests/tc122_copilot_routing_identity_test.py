"""Unit and contract tests for E40 model routing profiles and workflow routing identity.

Covers:
- REQ-F-009: Versioned model routing profiles and workflow routing identity calculation.
- AC-F13-11 (TC-F13-11): Deterministic computation of routing_digest, provider_digest,
  model_digest, and effort_digest by workflow_routing_identity().
- AC-F13-13 (TC-F13-13): Schema conformance of bench/profiles/*.yaml against
  bench/profiles/schema.yaml.
"""

import copy
import re
import shutil
import sys
import uuid
from pathlib import Path
import pytest
import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent.parent.parent
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from bench.scripts.lib.e40_benchmark import (
    DEFAULT_PROFILES_DIR,
    DEFAULT_PROFILE_SCHEMA,
    OperatorError,
    apply_routing_profile,
    load_routing_profile,
    load_yaml,
    validate_routing_profile_schema,
    workflow_routing_identity,
    write_yaml,
)
HEX_64_PATTERN = re.compile(r"^[0-9a-f]{64}$")
SCRATCH_BASE = REPO_ROOT / "dev-artifacts"


@pytest.fixture
def scratch_dir():
    """Provides an isolated scratch directory within the repository (never in /tmp)."""
    SCRATCH_BASE.mkdir(parents=True, exist_ok=True)
    unique_id = uuid.uuid4().hex[:12]
    work_dir = SCRATCH_BASE / f"tc122-{unique_id}"
    work_dir.mkdir(parents=True, exist_ok=False)
    try:
        yield work_dir
    finally:
        if work_dir.exists():
            shutil.rmtree(work_dir, ignore_errors=True)


def create_sample_workflow_bundle(root: Path, provider="copilot", model="claude-sonnet-5", effort="high") -> Path:
    """Create a sample Shark 2.x workflow directory structure for testing."""
    workflow_dir = root / "shark-data" / "workflow"
    workflow_dir.mkdir(parents=True, exist_ok=True)

    task_workflow = {
        "version": "1.0",
        "start": "draft",
        "steps": {
            "draft": {
                "phase": "planning",
                "progress_weight": 0.0,
            },
            "development": {
                "phase": "development",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": model,
                "effort": effort,
                "prompt": "task/development.md",
            },
            "review": {
                "phase": "review",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": model,
                "effort": effort,
                "prompt": "task/review.md",
            },
            "completed": {
                "phase": "done",
                "progress_weight": 1.0,
                "terminal": True,
            },
        },
    }
    feature_workflow = {
        "version": "1.0",
        "start": "discovery",
        "steps": {
            "discovery": {
                "phase": "planning",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": "gemini-3.8-flash",
                "effort": "medium",
            },
            "specification": {
                "phase": "planning",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": model,
                "effort": "medium",
            },
            "qa": {
                "phase": "qa",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": "gemini-3.8-flash",
                "effort": "low",
            },
        },
    }

    write_yaml(workflow_dir / "task.yaml", task_workflow)
    write_yaml(workflow_dir / "feature.yaml", feature_workflow)
    return workflow_dir


# ---------------------------------------------------------------------------
# TC-F13-13: Schema Conformance of Routing Profiles (AC-F13-13)
# ---------------------------------------------------------------------------

def test_routing_profiles_schema():
    """Verify that bench/profiles/schema.yaml exists and that shipped profiles conform."""
    assert DEFAULT_PROFILE_SCHEMA.exists(), f"Schema file missing: {DEFAULT_PROFILE_SCHEMA}"
    schema = load_yaml(DEFAULT_PROFILE_SCHEMA, "routing profile schema")
    assert schema.get("schema_version") == "1.0"
    assert "required_fields" in schema
    assert "valid_efforts" in schema
    assert "valid_providers" in schema

    # Verify copilot-balanced.yaml
    balanced_path = DEFAULT_PROFILES_DIR / "copilot-balanced.yaml"
    assert balanced_path.exists(), f"copilot-balanced.yaml missing: {balanced_path}"
    balanced = load_routing_profile(balanced_path)
    errors = validate_routing_profile_schema(balanced, DEFAULT_PROFILE_SCHEMA)
    assert errors == [], f"copilot-balanced.yaml has schema errors: {errors}"
    assert balanced["profile_id"] == "copilot-balanced"
    assert balanced["provider"] == "copilot"

    # Route assignments for balanced:
    # discovery, planning, qa -> gemini-3.8-flash
    # specification, development, review -> claude-sonnet-5
    routes = balanced["routes"]
    assert routes["discovery"]["model"] == "gemini-3.8-flash"
    assert routes["discovery"]["effort"] == "medium"
    assert routes["planning"]["model"] == "gemini-3.8-flash"
    assert routes["planning"]["effort"] == "minimal"
    assert routes["qa"]["model"] == "gemini-3.8-flash"
    assert routes["qa"]["effort"] == "low"

    assert routes["specification"]["model"] == "claude-sonnet-5"
    assert routes["specification"]["effort"] == "medium"
    assert routes["development"]["model"] == "claude-sonnet-5"
    assert routes["development"]["effort"] == "high"
    assert routes["review"]["model"] == "claude-sonnet-5"
    assert routes["review"]["effort"] == "high"

    # Verify copilot-variant-review.yaml
    variant_path = DEFAULT_PROFILES_DIR / "copilot-variant-review.yaml"
    assert variant_path.exists(), f"copilot-variant-review.yaml missing: {variant_path}"
    variant = load_routing_profile(variant_path)
    errors = validate_routing_profile_schema(variant, DEFAULT_PROFILE_SCHEMA)
    assert errors == [], f"copilot-variant-review.yaml has schema errors: {errors}"
    assert variant["profile_id"] == "copilot-variant-review"
    assert variant["provider"] == "copilot"

    # Route assignments for variant:
    # review step routes to gpt-5.4 with effort xhigh; other steps match balanced
    variant_routes = variant["routes"]
    assert variant_routes["review"]["model"] == "gpt-5.4"
    assert variant_routes["review"]["effort"] == "xhigh"

    for step in ("discovery", "specification", "planning", "development", "qa"):
        assert variant_routes[step] == routes[step], f"Mismatch for step {step}"

    # Verify copilot-default.yaml (if present)
    default_path = DEFAULT_PROFILES_DIR / "copilot-default.yaml"
    if default_path.exists():
        default_prof = load_routing_profile(default_path)
        errors = validate_routing_profile_schema(default_prof, DEFAULT_PROFILE_SCHEMA)
        assert errors == [], f"copilot-default.yaml has schema errors: {errors}"


def test_routing_profile_schema_negative_cases():
    """Verify validate_routing_profile_schema fails closed on malformed profiles."""
    valid_profile = {
        "schema_version": "1.0",
        "profile_id": "test-valid",
        "description": "Test profile",
        "provider": "copilot",
        "routes": {
            "development": {"model": "claude-sonnet-5", "effort": "high"}
        },
    }
    assert validate_routing_profile_schema(valid_profile) == []

    # Non-dictionary profile
    assert validate_routing_profile_schema("invalid") != []

    # Missing schema_version
    bad = copy.deepcopy(valid_profile)
    del bad["schema_version"]
    assert any("schema_version" in e for e in validate_routing_profile_schema(bad))

    # Unsupported schema_version
    bad = copy.deepcopy(valid_profile)
    bad["schema_version"] = "2.0"
    assert any("schema_version" in e for e in validate_routing_profile_schema(bad))

    # Missing or empty profile_id
    bad = copy.deepcopy(valid_profile)
    bad["profile_id"] = "   "
    assert any("profile_id" in e for e in validate_routing_profile_schema(bad))

    # Missing description
    bad = copy.deepcopy(valid_profile)
    del bad["description"]
    assert any("description" in e for e in validate_routing_profile_schema(bad))

    # Missing or invalid provider
    bad = copy.deepcopy(valid_profile)
    bad["provider"] = "unsupported-vendor-xyz"
    assert any("provider" in e for e in validate_routing_profile_schema(bad))

    # Missing or empty routes
    bad = copy.deepcopy(valid_profile)
    bad["routes"] = {}
    assert any("routes" in e for e in validate_routing_profile_schema(bad))

    # Route missing model
    bad = copy.deepcopy(valid_profile)
    bad["routes"]["development"] = {"effort": "high"}
    assert any("model" in e for e in validate_routing_profile_schema(bad))

    # Route with invalid effort
    bad = copy.deepcopy(valid_profile)
    bad["routes"]["development"]["effort"] = "super-duper-high"
    assert any("effort" in e for e in validate_routing_profile_schema(bad))


# ---------------------------------------------------------------------------
# TC-F13-11: Workflow Routing Identity & Determinism (AC-F13-11)
# ---------------------------------------------------------------------------

def test_workflow_routing_identity_copilot(scratch_dir):
    """Verify workflow_routing_identity computes all 4 digests deterministically for Copilot routes."""
    workflow_dir = create_sample_workflow_bundle(scratch_dir, provider="copilot")

    identity = workflow_routing_identity(workflow_dir)

    # Verify top-level structure
    for key in ("routes", "provider_digest", "model_digest", "effort_digest", "routing_digest"):
        assert key in identity, f"Missing key in identity: {key}"

    # Verify SHA-256 digest formats
    assert HEX_64_PATTERN.match(identity["provider_digest"])
    assert HEX_64_PATTERN.match(identity["model_digest"])
    assert HEX_64_PATTERN.match(identity["effort_digest"])
    assert HEX_64_PATTERN.match(identity["routing_digest"])

    # Verify routes contain Copilot provider
    routes = identity["routes"]
    assert len(routes) >= 4
    for route in routes:
        assert route["provider"] == "copilot"
        assert HEX_64_PATTERN.match(route["source"]) or route["source"].endswith(".yaml")

    # Verify determinism: recomputing gives identical digests
    identity_again = workflow_routing_identity(workflow_dir)
    assert identity_again["provider_digest"] == identity["provider_digest"]
    assert identity_again["model_digest"] == identity["model_digest"]
    assert identity_again["effort_digest"] == identity["effort_digest"]
    assert identity_again["routing_digest"] == identity["routing_digest"]


def test_workflow_routing_identity_model_mutation(scratch_dir):
    """Verify mutating a route's model alters model_digest and routing_digest while preserving provider and effort digests."""
    workflow_dir = create_sample_workflow_bundle(scratch_dir, provider="copilot", model="claude-sonnet-5", effort="high")
    baseline = workflow_routing_identity(workflow_dir)

    # Mutate the model in task.yaml for development step
    task_yaml_path = workflow_dir / "task.yaml"
    task_doc = load_yaml(task_yaml_path, "task workflow")
    task_doc["steps"]["development"]["model"] = "gpt-5.4"
    write_yaml(task_yaml_path, task_doc)

    mutated = workflow_routing_identity(workflow_dir)

    # Model and overall routing digests MUST change
    assert mutated["model_digest"] != baseline["model_digest"]
    assert mutated["routing_digest"] != baseline["routing_digest"]

    # Provider and effort digests MUST be preserved
    assert mutated["provider_digest"] == baseline["provider_digest"]
    assert mutated["effort_digest"] == baseline["effort_digest"]


def test_workflow_routing_identity_effort_mutation(scratch_dir):
    """Verify mutating a route's effort alters effort_digest and routing_digest while preserving provider and model digests."""
    workflow_dir = create_sample_workflow_bundle(scratch_dir, provider="copilot", model="claude-sonnet-5", effort="high")
    baseline = workflow_routing_identity(workflow_dir)

    # Mutate effort in task.yaml for development step
    task_yaml_path = workflow_dir / "task.yaml"
    task_doc = load_yaml(task_yaml_path, "task workflow")
    task_doc["steps"]["development"]["effort"] = "xhigh"
    write_yaml(task_yaml_path, task_doc)

    mutated = workflow_routing_identity(workflow_dir)

    # Effort and overall routing digests MUST change
    assert mutated["effort_digest"] != baseline["effort_digest"]
    assert mutated["routing_digest"] != baseline["routing_digest"]

    # Provider and model digests MUST be preserved
    assert mutated["provider_digest"] == baseline["provider_digest"]
    assert mutated["model_digest"] == baseline["model_digest"]


def test_workflow_routing_identity_provider_mutation(scratch_dir):
    """Verify mutating provider alters provider_digest and routing_digest while preserving model and effort digests."""
    workflow_dir = create_sample_workflow_bundle(scratch_dir, provider="copilot", model="claude-sonnet-5", effort="high")
    baseline = workflow_routing_identity(workflow_dir)

    # Mutate provider in task.yaml for development step
    task_yaml_path = workflow_dir / "task.yaml"
    task_doc = load_yaml(task_yaml_path, "task workflow")
    task_doc["steps"]["development"]["provider"] = "anthropic"
    write_yaml(task_yaml_path, task_doc)

    mutated = workflow_routing_identity(workflow_dir)

    # Provider and overall routing digests MUST change
    assert mutated["provider_digest"] != baseline["provider_digest"]
    assert mutated["routing_digest"] != baseline["routing_digest"]

    # Model and effort digests MUST be preserved
    assert mutated["model_digest"] == baseline["model_digest"]
    assert mutated["effort_digest"] == baseline["effort_digest"]


def test_apply_routing_profile_to_workflow(scratch_dir):
    """Verify applying copilot-balanced and copilot-variant-review profiles produces distinct, expected identities."""
    balanced_profile = load_routing_profile(DEFAULT_PROFILES_DIR / "copilot-balanced.yaml")
    variant_profile = load_routing_profile(DEFAULT_PROFILES_DIR / "copilot-variant-review.yaml")

    # Create baseline workflow directory
    wf_balanced = create_sample_workflow_bundle(scratch_dir / "balanced")
    apply_routing_profile(balanced_profile, wf_balanced)
    id_balanced = workflow_routing_identity(wf_balanced)

    # Create variant workflow directory
    wf_variant = create_sample_workflow_bundle(scratch_dir / "variant")
    apply_routing_profile(variant_profile, wf_variant)
    id_variant = workflow_routing_identity(wf_variant)

    # Both profiles specify provider: copilot -> provider_digest must match
    assert id_balanced["provider_digest"] == id_variant["provider_digest"]

    # Variant modifies review step to gpt-5.4 with xhigh effort -> model, effort, routing digests must differ
    assert id_balanced["model_digest"] != id_variant["model_digest"]
    assert id_balanced["effort_digest"] != id_variant["effort_digest"]
    assert id_balanced["routing_digest"] != id_variant["routing_digest"]


def test_workflow_routing_identity_error_cases(scratch_dir):
    """Verify fail-closed error handling for empty or nonexistent directories."""
    # Nonexistent path
    with pytest.raises(OperatorError, match="does not exist"):
        workflow_routing_identity(scratch_dir / "nonexistent-dir")

    # Directory with no workflow routes
    empty_dir = scratch_dir / "empty-dir"
    empty_dir.mkdir()
    write_yaml(empty_dir / "other.yaml", {"key": "value"})
    with pytest.raises(OperatorError, match="contains no provider/model/effort routes"):
        workflow_routing_identity(empty_dir)
