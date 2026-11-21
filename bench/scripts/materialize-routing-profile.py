#!/usr/bin/env python3
"""Validate an E40 route and apply an optional profile to an isolated workflow copy."""

import argparse
import json
import os
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from bench.scripts.lib.e40_benchmark import (
    OperatorError,
    apply_selected_routing_profile,
    live_workflow_routes,
    live_workflow_stages,
    load_routing_profile,
    path_digest,
    validate_routing_profile_schema,
    workflow_routing_identity,
    _update_route_mapping,
    load_yaml,
    write_yaml,
    workflow_yaml_paths,
)
from bench.scripts.lib.codex_model_catalog import (
    CatalogError,
    discover_models,
    resolve_cli_command,
    resolve_route,
    validate_catalog_snapshot,
)


def _codex_catalog():
    raw = os.environ.get("E40_CODEX_COMMAND")
    try:
        command = json.loads(raw) if raw else ["codex"]
    except json.JSONDecodeError as exc:
        raise OperatorError("E40_CODEX_COMMAND must be a JSON argv array") from exc
    if not isinstance(command, list) or not command or not all(isinstance(item, str) for item in command):
        raise OperatorError("E40_CODEX_COMMAND must be a non-empty JSON argv array")
    try:
        frozen = os.environ.get("E40_CODEX_CATALOG_JSON")
        if frozen:
            catalog = json.loads(frozen)
            return validate_catalog_snapshot(catalog, command)
        return discover_models(resolve_cli_command(command))
    except CatalogError as exc:
        raise OperatorError(f"route_defect: Codex model discovery failed: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise OperatorError("E40_CODEX_CATALOG_JSON must be valid JSON") from exc


def _resolve_route(stage, route, catalog):
    provider = str(route["provider"]).strip().lower()
    effort = route["effort"]
    if isinstance(effort, str):
        effort = effort.strip().lower()
    normalized = {**route, "provider": provider, "effort": effort}
    if provider not in {"codex", "openai"}:
        return normalized
    try:
        selected = resolve_route(
            route["model"],
            effort,
            catalog,
        )
    except CatalogError as exc:
        raise OperatorError(f"route_defect: stage {stage!r}: {exc}") from exc
    return {**normalized, **selected}


def _resolve_routes(routes, catalog):
    return {
        stage: _resolve_route(stage, route, catalog)
        for stage, route in routes.items()
    }


def _selected_codex_metadata(routes, catalog):
    if catalog is None:
        return None
    entries = {entry["model"]: entry for entry in catalog["models"]}
    selected = {}
    for stage, route in routes.items():
        if route["provider"] not in {"codex", "openai"}:
            continue
        entry = entries.get(route["model"])
        if entry is None:
            raise OperatorError(
                f"route_defect: selected Codex model metadata is missing for stage {stage!r}"
            )
        selected[stage] = dict(entry)
    return selected


def _write_routes(workflow_root: Path, routes, families=None):
    base_dir = workflow_root.parent if workflow_root.is_file() else workflow_root
    for path in workflow_yaml_paths(workflow_root, families):
        source = path.relative_to(base_dir).as_posix()
        document = load_yaml(path, "workflow file")
        file_routes = [
            route for route in routes.values() if route.get("source") == source
        ]
        profile_routes = {}
        effective_routes = {}
        for route in file_routes:
            stage = route["stage"]
            profile_routes[stage] = (
                {"effort": route["effort"]}
                if route["effort"] is not None
                else {}
            )
            effective_routes[stage] = route
        changed = False
        for section in (document.get("steps"), document.get("status_metadata")):
            changed = _update_route_mapping(
                section, profile_routes, effective_routes
            ) or changed
        if changed:
            write_yaml(path, document)


def _profile_request(profile_path: Path, workflow_root: Path, families=None):
    profile = load_routing_profile(profile_path)
    errors = validate_routing_profile_schema(profile)
    if errors:
        raise OperatorError(f"cannot select invalid routing profile: {'; '.join(errors)}")
    stages = live_workflow_stages(workflow_root, families)
    routes = profile["routes"]
    requested = [
        {
            "stage": stage,
            "provider": profile["provider"],
            "model": routes[stage].get("model"),
            "effort": routes[stage].get("effort"),
        }
        for stage in sorted(stages)
        if stage in routes
    ]
    return profile, requested


def _raise_inherited_route_error() -> None:
    route_error = os.environ.get("E40_ROUTE_MATERIALIZATION_ERROR")
    if route_error:
        raise OperatorError(route_error)


def _materialize_profile(
    profile_path: Path, workflow_root: Path, requested_routes: list, families=None
):
    profile, requested = _profile_request(profile_path, workflow_root, families)
    requested_routes.extend(requested)
    _raise_inherited_route_error()
    provider = str(profile.get("provider", "")).strip().lower()
    catalog = _codex_catalog() if provider in {"codex", "openai"} else None
    requested = {item["stage"]: {key: item[key] for key in ("provider", "model", "effort")} for item in requested}
    resolved = _resolve_routes(requested, catalog or {"models": []})
    effective_profile = dict(profile)
    effective_profile["provider"] = provider
    effective_profile["routes"] = _effective_profile_routes(profile, resolved)
    result = apply_selected_routing_profile(
        effective_profile,
        workflow_root,
        families=families,
        validate_catalogs=False,
    )
    effective_routes = _effective_routes_by_source(workflow_root, families)
    result["effective_routes"] = effective_routes
    result["requested_routes"] = requested_routes
    result["provider_routing"] = workflow_routing_identity(workflow_root)
    if catalog is not None:
        result["codex_catalog"] = catalog
        result["selected_codex_models"] = _selected_codex_metadata(
            effective_routes, catalog
        )
    result["profile_path"] = str(profile_path.resolve())
    result["profile_source_digest"] = path_digest(profile_path, "routing profile")
    return result


def _effective_profile_routes(profile, resolved):
    provider = str(profile.get("provider", "")).strip().lower()
    routes = {}
    for stage, route in resolved.items():
        effective = {"model": route["model"]}
        if provider in {"codex", "openai"} or "effort" in profile["routes"][stage]:
            effective["effort"] = route["effort"]
        routes[stage] = effective
    return routes


def _effective_routes_by_source(workflow_root, families):
    return {
        route_key: {key: route[key] for key in ("provider", "model", "effort")}
        for route_key, route in live_workflow_routes(workflow_root, families).items()
    }


def _materialize_explicit_routes(
    workflow_root: Path, requested_routes: list, families=None
):
    routes = live_workflow_routes(workflow_root, families)
    requested = [
        {"route_key": route_key, **route}
        for route_key, route in sorted(routes.items())
    ]
    requested_routes.extend(requested)
    _raise_inherited_route_error()
    catalog = _codex_catalog() if any(
        route["provider"].strip().lower() in {"codex", "openai"}
        for route in routes.values()
    ) else None
    routes = _resolve_routes(routes, catalog or {"models": []})
    _write_routes(workflow_root, routes, families)
    result = {
        "profile_id": None,
        "profile_path": None,
        "profile_source_digest": None,
        "profile_digest": None,
        "effective_routes": routes,
        "requested_routes": requested_routes,
        "provider_routing": workflow_routing_identity(workflow_root),
    }
    if catalog is not None:
        result["codex_catalog"] = catalog
        result["selected_codex_models"] = _selected_codex_metadata(routes, catalog)
    return result


def _materialize(
    workflow_root: Path,
    profile_path: Path | None,
    requested_routes: list,
    families=None,
):
    if profile_path is not None:
        return _materialize_profile(
            profile_path, workflow_root, requested_routes, families
        )
    return _materialize_explicit_routes(workflow_root, requested_routes, families)


def _failure_record(error: OperatorError, requested_routes: list[dict]):
    return {
        "resolution": "failed",
        "cause_class": "route_defect",
        "reason": str(error),
        "dispatch_count": 0,
        "requested_routes": requested_routes,
    }


def _validate_workflow_root(workflow_root: Path, allowed_root: Path | None) -> None:
    if allowed_root is None:
        return
    try:
        root = allowed_root.resolve(strict=True)
        target = workflow_root.resolve(strict=True)
    except OSError as exc:
        raise OperatorError(f"workflow root cannot be resolved: {exc}") from exc
    try:
        target.relative_to(root)
    except ValueError as exc:
        raise OperatorError(
            "workflow root resolves outside the allowed isolated workflow copy"
        ) from exc


def main(argv=None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--workflow-root", required=True)
    parser.add_argument("--allowed-root")
    parser.add_argument("--profile")
    parser.add_argument("--family", action="append", dest="families")
    args = parser.parse_args(argv)
    requested_routes = []
    try:
        workflow_root = Path(args.workflow_root)
        _validate_workflow_root(
            workflow_root, Path(args.allowed_root) if args.allowed_root else None
        )
        result = _materialize(
            workflow_root,
            Path(args.profile) if args.profile else None,
            requested_routes,
            args.families,
        )
    except OperatorError as exc:
        print(json.dumps(_failure_record(exc, requested_routes), sort_keys=True))
        return 1
    except OSError as exc:
        error = OperatorError(f"route_defect: cannot write materialized workflow route: {exc}")
        print(json.dumps(_failure_record(error, requested_routes), sort_keys=True))
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
