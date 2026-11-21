"""Normalize scratch workflow routes for tests unrelated to B083 routing."""

import argparse
from pathlib import Path

import yaml


def _route_updates(step):
    if not isinstance(step, dict) or step.get("action") != "spawn_agent":
        return {}
    provider = step.get("provider")
    if provider in {"codex", "openai"} or not provider:
        return {"provider": "anthropic", "model": "haiku"}
    if not step.get("model"):
        return {"model": "haiku"}
    return {}


def _validate_workflow_root(workflow_root: Path, allowed_root: Path) -> Path:
    if allowed_root.is_symlink() or not allowed_root.is_dir():
        raise ValueError(f"allowed root must be an existing non-symlink directory: {allowed_root}")
    if workflow_root.is_symlink() or not workflow_root.is_dir():
        raise ValueError(f"workflow root must be an existing non-symlink directory: {workflow_root}")
    allowed = allowed_root.resolve(strict=True)
    root = workflow_root.resolve(strict=True)
    try:
        root.relative_to(allowed)
    except ValueError as exc:
        raise ValueError(f"workflow root is outside the allowed test scratch root: {workflow_root}") from exc
    return root


def _validate_workflow_file(path: Path, allowed_root: Path) -> None:
    resolved = path.resolve(strict=True)
    if path.is_symlink() or not resolved.is_relative_to(allowed_root):
        raise ValueError(f"workflow YAML path escapes the allowed test scratch root: {path}")


def _normalize_steps(steps: dict) -> bool:
    changed = False
    for step in steps.values():
        updates = _route_updates(step)
        if updates:
            step.update(updates)
            changed = True
    return changed


def _normalize_workflow_file(path: Path, allowed_root: Path) -> None:
    _validate_workflow_file(path, allowed_root)
    document = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(document, dict):
        raise ValueError(f"workflow YAML document must be a mapping: {path}")
    steps = document.get("steps") or {}
    if not isinstance(steps, dict):
        raise ValueError(f"workflow steps must be a mapping: {path}")
    if _normalize_steps(steps):
        path.write_text(yaml.safe_dump(document, sort_keys=False), encoding="utf-8")


def normalize_workflow_routes(workflow_root: Path, allowed_root: Path) -> None:
    root = _validate_workflow_root(workflow_root, allowed_root)
    allowed = allowed_root.resolve(strict=True)
    paths = sorted((*root.rglob("*.yaml"), *root.rglob("*.yml")))
    for path in paths:
        _normalize_workflow_file(path, allowed)


def main(argv=None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("workflow_root", type=Path)
    parser.add_argument("--allowed-root", type=Path, required=True)
    args = parser.parse_args(argv)
    try:
        normalize_workflow_routes(args.workflow_root, args.allowed_root)
    except (OSError, ValueError, yaml.YAMLError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
