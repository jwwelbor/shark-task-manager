from pathlib import Path
import shutil
import uuid

import pytest

from bench.scripts.lib.e40_benchmark import write_yaml

REPO_ROOT = Path(__file__).resolve().parents[4]
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


def _sample_task_workflow(provider, model, effort):
    return {
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


def _sample_feature_workflow(provider, model, effort):
    return {
        "version": "1.0",
        "start": "discovery",
        "steps": {
            "discovery": {
                "phase": "planning",
                "responsibility": "agent",
                "action": "spawn_agent",
                "provider": provider,
                "model": f"{model}-light",
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
                "model": f"{model}-light",
                "effort": "low",
            },
        },
    }


def create_sample_workflow_bundle(
    root: Path, provider="copilot", model="synthetic-model", effort="high"
) -> Path:
    """Create a sample Shark 2.x workflow directory structure for testing."""
    workflow_dir = root / "shark-data" / "workflow"
    workflow_dir.mkdir(parents=True, exist_ok=True)
    write_yaml(workflow_dir / "task.yaml", _sample_task_workflow(provider, model, effort))
    write_yaml(workflow_dir / "feature.yaml", _sample_feature_workflow(provider, model, effort))
    return workflow_dir
