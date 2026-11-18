#!/usr/bin/env python3
"""B083: production execution binds the manifest to an immutable routing profile."""

import argparse
import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import stat
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


def _setup_fixture(work):
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
    profile_i05 = operator_root / "b083-i05-profile"
    source_i05 = operator_root / "b083-i05-source"
    profile_i05.mkdir()
    source_i05.mkdir()
    workflow_root = operator_root / "scratch-template" / "shark-data" / "workflow"
    config = yaml.safe_load((operator_root / "e40-demo.yaml").read_text(encoding="utf-8"))

    stages = set()
    for workflow_path in sorted((*workflow_root.rglob("*.yaml"), *workflow_root.rglob("*.yml"))):
        workflow = yaml.safe_load(workflow_path.read_text(encoding="utf-8")) or {}
        changed = False
        for stage, step in (workflow.get("steps") or {}).items():
            if isinstance(step, dict) and step.get("action") == "spawn_agent":
                stages.add(stage)
                step.update(provider="anthropic", model="b083-source-model", effort="medium")
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
    config["runtime"]["lifecycle_adapter"] = str(work / "fake-adapter.py")
    config["scenario_roots"][SCENARIO_ID]["i05_bundle_dir"] = str(profile_i05)
    config_path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")

    prompt_bytes = PROMPT.encode("utf-8")
    response = {
        "entity_key": config["scenario_roots"][SCENARIO_ID]["root_key"],
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
    response_path = work / "shark-response.json"
    response_path.write_text(json.dumps(response), encoding="utf-8")
    return {
        "work": work,
        "operator_root": operator_root,
        "config": config,
        "config_path": config_path,
        "profile_path": profile_path,
        "workflow_root": workflow_root,
        "stages": stages,
        "source_i05": source_i05,
        "response_path": response_path,
        "capture_path": work / "adapter-request.json",
        "source_capture_path": work / "adapter-request-no-profile.json",
    }


def _write_fake_adapter(path):
    path.write_text(
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
    path.chmod(0o755)


def _write_fake_shark(path):
    path.write_text(
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
    print('{"session_id":"SID-B083"}')
elif args and args[0] == 'heartbeat':
    print('{"ok":true}')
elif args[:2] == ['status', 'advance']:
    print('{"advanced":true}')
elif args and args[0] == 'release':
    print('{"released":true}')
elif args and args[0] == 'history':
    print(json.dumps({'entity_key': args[1], 'history': []}))
else:
    raise SystemExit('unexpected shark argv: ' + repr(args))
""",
        encoding="utf-8",
    )
    path.chmod(0o755)


def _install_fake_tools(fixture):
    _write_fake_adapter(fixture["work"] / "fake-adapter.py")
    shark_path = fixture["operator_root"] / "scratch-template" / "shark"
    _write_fake_shark(shark_path)
    digest = subprocess.run(
        [str(DIGEST), str(shark_path)], text=True, capture_output=True, check=True
    ).stdout.strip()
    setup_result_path = fixture["operator_root"] / "setup-result.json"
    setup_result = json.loads(setup_result_path.read_text(encoding="utf-8"))
    setup_result["shark_binary"]["sha256"] = digest
    setup_result_path.write_text(json.dumps(setup_result), encoding="utf-8")


def _load_benchmark_module(fixture):
    module_path = SCRIPTS_DIR / "lib" / "e40_benchmark.py"
    spec = importlib.util.spec_from_file_location("e40_benchmark_b083_tc123", module_path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    # Aggregation is outside the route contract; the batch, runner, adapter,
    # manifest, and retention paths remain real.
    module.aggregate_and_report = lambda _root_fd, _run_root: {}
    os.environ.update(
        {
            "SHARK_RESPONSE": str(fixture["response_path"]),
            "SHARK_WORKFLOW_DIR": str(SCRIPTS_DIR / "testdata" / "lifecycle" / "workflow"),
        }
    )
    return module


def _execute(module, fixture, run_id, run_config, capture_path, before_batch=None):
    os.environ["B083_ADAPTER_CAPTURE"] = str(capture_path)
    original_materialize = module.materialize_batch_policy

    def materialize_after_hook(*args, **kwargs):
        if before_batch is not None:
            before_batch()
        return original_materialize(*args, **kwargs)

    module.materialize_batch_policy = materialize_after_hook
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
    output, errors = io.StringIO(), io.StringIO()
    try:
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
            exit_code = module.execute_profile(args, role="baseline", run_mode="pilot")
    finally:
        module.materialize_batch_policy = original_materialize
    require(
        exit_code == 0,
        f"execute_profile({run_id}) returned {exit_code}: {errors.getvalue()} {output.getvalue()[-3000:]}",
    )

    manifest_path = fixture["operator_root"] / "runs" / run_id / "pilot-benchmark-manifest.json"
    require(manifest_path.is_file(), f"{run_id} has no pilot-benchmark-manifest.json")
    require(capture_path.is_file(), f"real run-lifecycle did not invoke fake adapter for {run_id}")
    return (
        json.loads(manifest_path.read_text(encoding="utf-8")),
        json.loads(capture_path.read_text(encoding="utf-8")),
    )


def _assert_profiled_run(operator_root, manifest, request, stages, source_profile_bytes):
    profile = manifest["identity"]["routing_profile"]
    expected_routes = {
        stage: {"provider": "copilot", "model": MODEL, "effort": "high"}
        for stage in sorted(stages)
    }
    require(
        profile["effective_routes"] == expected_routes,
        "manifest does not contain the selected provider/model/effort tuple for every live stage",
    )
    require(
        profile["profile_source_digest"] == hashlib.sha256(source_profile_bytes).hexdigest(),
        "manifest source digest does not bind the pre-mutation selected profile",
    )
    snapshot_path = Path(profile["profile_path"])
    require(
        snapshot_path.name == "routing-profile.snapshot.yaml",
        "manifest identity does not use the immutable run-scoped profile snapshot",
    )
    require(
        snapshot_path.is_relative_to(operator_root / "runs"),
        "profile snapshot is outside the retained benchmark run root",
    )
    require(
        stat.S_IMODE(snapshot_path.stat().st_mode) & 0o222 == 0,
        "run-scoped profile snapshot is writable after identity capture",
    )
    batch_policy = yaml.safe_load(
        (snapshot_path.parent / "pilot-batch-policy.yaml").read_text(encoding="utf-8")
    )
    require(
        all(
            item["routing_profile"] == str(snapshot_path)
            for item in batch_policy["scenarios"].values()
        ),
        "batch dispatch does not use the same profile snapshot as the manifest",
    )
    require(
        {key: request[key] for key in ("provider", "model", "effort")} == expected_routes["development"],
        "profiled adapter route differs from the manifest development route",
    )
    bound = request.get("routing_identity") or {}
    for field in (
        "profile_id",
        "profile_schema_version",
        "profile_digest",
        "profile_source_digest",
    ):
        require(bound.get(field) == profile.get(field), f"adapter {field} differs from the manifest")
    require(
        bound.get("routing_digest") == profile["provider_routing"]["routing_digest"],
        "adapter routing digest differs from the manifest",
    )


def _run_no_profile_case(module, fixture):
    config = fixture["config"]
    source_config = dict(config)
    source_config["routing_profile"] = None
    source_config["profile"] = dict(config["profile"])
    source_config["profile"]["name"] = "b083-route-no-profile"
    source_config["scenario_roots"] = {
        key: dict(value) for key, value in config["scenario_roots"].items()
    }
    source_config["scenario_roots"][SCENARIO_ID]["i05_bundle_dir"] = str(fixture["source_i05"])
    source_config_path = fixture["operator_root"] / "b083-no-profile.yaml"
    source_config_path.write_text(yaml.safe_dump(source_config, sort_keys=False), encoding="utf-8")

    workflow = yaml.safe_load(
        (fixture["workflow_root"] / "bug.yaml").read_text(encoding="utf-8")
    )
    route = workflow["steps"]["development"]
    response = json.loads(fixture["response_path"].read_text(encoding="utf-8"))
    response.update(provider=route["provider"], model=route["model"], effort=route["effort"])
    fixture["response_path"].write_text(json.dumps(response), encoding="utf-8")
    return _execute(
        module,
        fixture,
        "b083-route-no-profile",
        source_config_path,
        fixture["source_capture_path"],
    )


def _assert_no_profile_route(manifest, request):
    identity = manifest["identity"]["provider_routing"]
    source_route = next(
        route
        for route in identity["routes"]
        if route["source"] == "bug.yaml" and route["pointer"] == "/steps/development"
    )
    require(manifest["identity"]["routing_profile"] is None, "no-profile manifest has a selected profile")
    expected = {key: source_route[key] for key in ("provider", "model", "effort")}
    actual = {key: request[key] for key in ("provider", "model", "effort")}
    require(actual == expected, "no-profile adapter route differs from source workflow identity")
    require(
        (request.get("routing_identity") or {}).get("routing_digest") == identity["routing_digest"],
        "no-profile adapter routing digest differs from source workflow identity",
    )


def main():
    with tempfile.TemporaryDirectory(prefix="tc123-b083-") as temporary:
        fixture = _setup_fixture(Path(temporary))
        _install_fake_tools(fixture)
        module = _load_benchmark_module(fixture)
        source_profile_bytes = fixture["profile_path"].read_bytes()

        def mutate_source_profile_after_manifest_capture():
            profile = yaml.safe_load(fixture["profile_path"].read_text(encoding="utf-8"))
            profile["routes"]["development"]["model"] = "b083-mutated-after-identity"
            fixture["profile_path"].write_text(yaml.safe_dump(profile, sort_keys=False), encoding="utf-8")

        manifest, request = _execute(
            module,
            fixture,
            "b083-route-profile",
            fixture["config_path"],
            fixture["capture_path"],
            before_batch=mutate_source_profile_after_manifest_capture,
        )
        require(
            fixture["profile_path"].read_bytes() != source_profile_bytes,
            "counterfactual did not change the source profile after manifest identity capture",
        )
        _assert_profiled_run(
            fixture["operator_root"],
            manifest,
            request,
            fixture["stages"],
            source_profile_bytes,
        )

        source_manifest, source_request = _run_no_profile_case(module, fixture)
        _assert_no_profile_route(source_manifest, source_request)
    print("TC-123 (B083/F3: immutable profile identity and all-stage routes reach the real batch and adapter path) PASS")


if __name__ == "__main__":
    main()
