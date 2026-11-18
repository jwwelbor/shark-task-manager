#!/usr/bin/env python3
"""Validate an E40 route and apply an optional profile to an isolated workflow copy."""

import argparse
import json
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from bench.scripts.lib.e40_benchmark import (
    OperatorError,
    apply_selected_routing_profile,
    live_workflow_routes,
    load_routing_profile,
    path_digest,
    validate_codex_catalogs,
    workflow_routing_identity,
)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--workflow-root", required=True)
    parser.add_argument("--profile")
    args = parser.parse_args()
    workflow_root = Path(args.workflow_root)
    requested_routes = []
    try:
        if args.profile:
            profile_path = Path(args.profile)
            profile = load_routing_profile(profile_path)
            current_routes = live_workflow_routes(workflow_root)
            profile_routes = profile.get("routes") or {}
            requested_routes = [
                {
                    "stage": stage,
                    "provider": str(profile.get("provider") or current_routes[stage]["provider"]),
                    "model": str((profile_routes.get(stage) or {}).get("model") or current_routes[stage]["model"]),
                    "effort": str((profile_routes.get(stage) or {}).get("effort") or current_routes[stage]["effort"]),
                }
                for stage in sorted(current_routes)
            ]
            result = apply_selected_routing_profile(profile, workflow_root)
            result["profile_path"] = str(profile_path.resolve())
            result["profile_source_digest"] = path_digest(profile_path, "routing profile")
        else:
            routes = live_workflow_routes(workflow_root)
            requested_routes = [
                {"stage": stage, **route} for stage, route in sorted(routes.items())
            ]
            for route in routes.values():
                if route["provider"].lower() in {"codex", "openai"}:
                    validate_codex_catalogs(route["model"], None, None)
            result = {
                "profile_id": None,
                "profile_path": None,
                "profile_source_digest": None,
                "profile_digest": None,
                "effective_routes": routes,
                "provider_routing": workflow_routing_identity(workflow_root),
            }
    except OperatorError as exc:
        print(
            json.dumps(
                {
                    "resolution": "failed",
                    "cause_class": "route_defect",
                    "reason": str(exc),
                    "dispatch_count": 0,
                    "requested_routes": requested_routes,
                },
                sort_keys=True,
            )
        )
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
