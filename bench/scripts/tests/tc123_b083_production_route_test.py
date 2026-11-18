#!/usr/bin/env python3
"""B083/F3: production route identity reaches the live lifecycle adapter."""

import argparse
import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

import yaml

REPO_ROOT = Path(__file__).resolve().parents[3]
SCRIPTS_DIR = REPO_ROOT / "bench" / "scripts"
OPERATOR = SCRIPTS_DIR / "e40-benchmark.sh"
DIGEST = SCRIPTS_DIR / "lib" / "digest_path"
SCENARIO_ID = "py-bug-due-date-boundary"
MODEL = "b083-integration-model"
PROMPT = "B083 deterministic adapter integration prompt\n"


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def main():
    with tempfile.TemporaryDirectory(prefix="tc123-b083-") as temporary:
        work = Path(temporary)
        operator_root = work / "operator"
        setup = subprocess.run(
            [str(OPERATOR), "setup", "--out", str(operator_root)],
            cwd=REPO_ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        require(setup.returncode == 0, f"real setup failed: {setup.stderr}")

        config_path = operator_root / "b083-route-execution.yaml"
        profile_path = operator_root / "b083-copilot-profile.yaml"
        adapter_path = work / "fake-adapter.py"
        i05_path = operator_root / "b083-i05-profile"
        i05_path.mkdir()
        no_profile_i05_path = operator_root / "b083-i05-source"
        no_profile_i05_path.mkdir()
        capture_path = work / "adapter-request.json"
        no_profile_capture_path = work / "adapter-request-no-profile.json"
        response_path = work / "shark-response.json"
        workflow_root = operator_root / "scratch-template" / "shark-data" / "workflow"

        with (operator_root / "e40-demo.yaml").open(encoding="utf-8") as stream:
            config = yaml.safe_load(stream)
        stages = set()
        for workflow_path in sorted((*workflow_root.rglob("*.yaml"), *workflow_root.rglob("*.yml"))):
            workflow = yaml.safe_load(workflow_path.read_text(encoding="utf-8")) or {}
            changed = False
            for stage, step in (workflow.get("steps") or {}).items():
                if isinstance(step, dict) and step.get("action") == "spawn_agent":
                    stages.add(stage)
                    step["provider"] = "anthropic"
                    step["model"] = "b083-source-model"
                    step["effort"] = "medium"
                    changed = True
            if changed:
                workflow_path.write_text(yaml.safe_dump(workflow, sort_keys=False), encoding="utf-8")
        require("development" in stages, "fixture workflow has no development dispatch stage")
        profile = {
            "schema_version": "1.0",
            "profile_id": "b083-production-path",
            "description": "Deterministic route propagation fixture.",
            "provider": "copilot",
            "routes": {
                stage: {"model": MODEL, "effort": "high"}
                for stage in sorted(stages)
            },
        }
        profile_path.write_text(yaml.safe_dump(profile, sort_keys=False), encoding="utf-8")
        config["routing_profile"] = str(profile_path)
        config["profile"]["name"] = "b083-route-execution"
        config["runtime"]["lifecycle_adapter"] = str(adapter_path)
        config["scenario_roots"][SCENARIO_ID]["i05_bundle_dir"] = str(i05_path)
        config_path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")

        root_key = config["scenario_roots"][SCENARIO_ID]["root_key"]
        prompt_bytes = PROMPT.encode("utf-8")
        response = {
            "entity_key": root_key,
            "entity_type": "bug",
            "status": "development",
            "action": "spawn_agent",
            "agent_type": "developer",
            "provider": "copilot",
            "model": MODEL,
            "effort": "high",
            "prompt": PROMPT,
            "prompt_sha256": hashlib.sha256(prompt_bytes).hexdigest(),
            "prompt_bytes": len(prompt_bytes),
            "resolved_via": ["ROOT-001"],
            "unresolved_placeholders": [],
            "error": "",
            "question_block": None,
            "current_responder": "",
        }
        response_path.write_text(json.dumps(response), encoding="utf-8")

        adapter_path.write_text(
            """#!/usr/bin/env python3
import json, os, sys
request = json.load(sys.stdin)
with open(os.environ['B083_ADAPTER_CAPTURE'], 'w', encoding='utf-8') as stream:
    json.dump(request, stream, sort_keys=True)
print(json.dumps({'worker_id':'b083-fixture-worker','session_id':request['session_id'],
    'kind':'final','recommended_outcome':'pass','cost_usd':0.0,
    'evidence':{'summary':'deterministic B083 route integration'}}))
""",
            encoding="utf-8",
        )
        adapter_path.chmod(0o755)

        shark_path = operator_root / "scratch-template" / "shark"
        shark_path.write_text(
            """#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
if args and args[0] == 'next':
    response = json.load(open(os.environ['SHARK_RESPONSE'], encoding='utf-8'))
    Path(args[args.index('--prompt-out') + 1]).write_bytes(response['prompt'].encode('utf-8'))
    print(json.dumps(response, separators=(',', ':')))
elif args[:3] == ['admin', 'workflow', 'list']:
    sys.stdout.write(Path(os.environ['SHARK_WORKFLOW_DIR'], args[3] + '.json').read_text(encoding='utf-8'))
elif args and args[0] == 'claim':
    print('{\"session_id\":\"SID-B083\"}')
elif args and args[0] == 'heartbeat':
    print('{\"ok\":true}')
elif args[:2] == ['status', 'advance']:
    print('{\"advanced\":true}')
elif args and args[0] == 'release':
    print('{\"released\":true}')
elif args and args[0] == 'history':
    print(json.dumps({'entity_key': args[1], 'history': []}))
else:
    raise SystemExit('unexpected shark argv: ' + repr(args))
""",
            encoding="utf-8",
        )
        shark_path.chmod(0o755)
        digest = subprocess.run(
            [str(DIGEST), str(shark_path)], text=True, capture_output=True, check=True
        ).stdout.strip()
        setup_result_path = operator_root / "setup-result.json"
        setup_result = json.loads(setup_result_path.read_text(encoding="utf-8"))
        setup_result["shark_binary"]["sha256"] = digest
        setup_result_path.write_text(json.dumps(setup_result), encoding="utf-8")

        module_path = SCRIPTS_DIR / "lib" / "e40_benchmark.py"
        spec = importlib.util.spec_from_file_location("e40_benchmark_b083_tc123", module_path)
        module = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = module
        spec.loader.exec_module(module)
        # Aggregation is outside F3; keep execution, batch, route materializer,
        # runner, adapter request, and retention production paths real.
        module.aggregate_and_report = lambda _root_fd, _run_root: {}
        os.environ.update(
            {
                "SHARK_RESPONSE": str(response_path),
                "SHARK_WORKFLOW_DIR": str(SCRIPTS_DIR / "testdata" / "lifecycle" / "workflow"),
                "B083_ADAPTER_CAPTURE": str(capture_path),
            }
        )
        def execute(run_id, run_config, adapter_capture):
            os.environ["B083_ADAPTER_CAPTURE"] = str(adapter_capture)
            args = argparse.Namespace(
                config=str(run_config),
                run_id=run_id,
                reps=1,
                scenario=SCENARIO_ID,
                acknowledge_provider_spend=True,
                max_cost_usd=1.0,
                max_wall_clock_seconds=120,
                max_generated_tasks=1,
                max_provider_calls=2,
                out=None,
                retry_incomplete=False,
            )
            execution_stdout = io.StringIO()
            execution_stderr = io.StringIO()
            with contextlib.redirect_stdout(execution_stdout), contextlib.redirect_stderr(execution_stderr):
                exit_code = module.execute_profile(args, role="baseline", run_mode="pilot")
            require(
                exit_code == 0,
                f"execute_profile({run_id}) returned {exit_code}: {execution_stderr.getvalue()} {execution_stdout.getvalue()[-3000:]}",
            )
            manifest_path = operator_root / "runs" / run_id / "pilot-benchmark-manifest.json"
            require(manifest_path.is_file(), f"{run_id} has no pilot-benchmark-manifest.json")
            require(adapter_capture.is_file(), f"real run-lifecycle did not invoke fake adapter for {run_id}")
            return (
                json.loads(manifest_path.read_text(encoding="utf-8")),
                json.loads(adapter_capture.read_text(encoding="utf-8")),
            )

        profiled_manifest, profiled_request = execute(
            "b083-route-profile", config_path, capture_path
        )
        profile_routing = profiled_manifest["identity"]["routing_profile"]
        profile_route = profile_routing["effective_routes"]["development"]
        require(
            {key: profiled_request[key] for key in ("provider", "model", "effort")} == profile_route,
            "profiled adapter route differs from benchmark manifest",
        )
        profile_bound = profiled_request.get("routing_identity") or {}
        for request_key, manifest_key in (
            ("profile_id", "profile_id"),
            ("profile_schema_version", "profile_schema_version"),
            ("profile_digest", "profile_digest"),
            ("profile_source_digest", "profile_source_digest"),
        ):
            require(profile_bound.get(request_key) == profile_routing.get(manifest_key), f"profiled adapter {request_key} mismatch")
        require(
            profile_bound.get("routing_digest") == profile_routing["provider_routing"]["routing_digest"],
            "profiled adapter routing digest differs from benchmark manifest",
        )

        source_config = dict(config)
        source_config["routing_profile"] = None
        source_config["profile"] = dict(config["profile"])
        source_config["profile"]["name"] = "b083-route-no-profile"
        source_config["scenario_roots"] = {key: dict(value) for key, value in config["scenario_roots"].items()}
        source_config["scenario_roots"][SCENARIO_ID]["i05_bundle_dir"] = str(no_profile_i05_path)
        source_config_path = operator_root / "b083-no-profile.yaml"
        source_config_path.write_text(yaml.safe_dump(source_config, sort_keys=False), encoding="utf-8")
        source_workflow = yaml.safe_load((workflow_root / "bug.yaml").read_text(encoding="utf-8"))
        source_route = source_workflow["steps"]["development"]
        response.update(
            provider=source_route["provider"],
            model=source_route["model"],
            effort=source_route["effort"],
        )
        response_path.write_text(json.dumps(response), encoding="utf-8")
        source_manifest, source_request = execute(
            "b083-route-no-profile", source_config_path, no_profile_capture_path
        )
        source_identity = source_manifest["identity"]["provider_routing"]
        source_step_route = next(
            route
            for route in source_identity["routes"]
            if route["source"] == "bug.yaml" and route["pointer"] == "/steps/development"
        )
        require(source_manifest["identity"]["routing_profile"] is None, "no-profile manifest has a selected profile")
        require(
            {key: source_request[key] for key in ("provider", "model", "effort")} ==
            {key: source_step_route[key] for key in ("provider", "model", "effort")},
            "no-profile adapter route differs from source workflow identity",
        )
        source_bound = source_request.get("routing_identity") or {}
        require(
            source_bound.get("routing_digest") == source_identity["routing_digest"],
            "no-profile adapter routing digest differs from benchmark manifest",
        )
    print("TC-123 (B083/F3: profiled and no-profile execute_profile runs propagate manifest-bound routes through real batch and runner) PASS")


if __name__ == "__main__":
    main()
