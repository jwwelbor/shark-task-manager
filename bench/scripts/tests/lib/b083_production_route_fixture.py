"""Shared isolated fixtures for B083 production route integration tests."""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

import yaml

REPO_ROOT = Path(__file__).resolve().parents[4]
SCRIPTS_DIR = REPO_ROOT / "bench" / "scripts"
OPERATOR = SCRIPTS_DIR / "e40-benchmark.sh"
DIGEST = SCRIPTS_DIR / "lib" / "digest_path"
SCENARIO_ID = "py-bug-due-date-boundary"
MODEL = "b083-integration-model"
PROMPT = "B083 deterministic adapter integration prompt\n"

def require(condition, message):
    if not condition:
        raise AssertionError(message)

def _prepare_source_workflows(workflow_root):
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
    return stages

def _spawn_stages(path):
    workflow = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return {
        name for name, step in (workflow.get("steps") or {}).items()
        if isinstance(step, dict) and step.get("action") == "spawn_agent"
    }

def _write_profile_config(operator_root, work, profile_i05, stages, provider, model):
    config_path = operator_root / "b083-route-execution.yaml"
    profile_path = operator_root / f"b083-{provider}-profile.yaml"
    config = yaml.safe_load((operator_root / "e40-demo.yaml").read_text(encoding="utf-8"))

    profile = {
        "schema_version": "1.0",
        "profile_id": "b083-production-path",
        "description": "Deterministic route propagation fixture.",
        "provider": provider,
        "routes": {
            stage: {"model": model, "effort": "high"}
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
        "provider": provider,
        "model": model,
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
    return config, config_path, profile_path, response_path

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

    profile_i05 = operator_root / "b083-i05-profile"
    source_i05 = operator_root / "b083-i05-source"
    profile_i05.mkdir()
    source_i05.mkdir()
    workflow_root = operator_root / "scratch-template" / "shark-data" / "workflow"
    stages = _prepare_source_workflows(workflow_root)
    config, config_path, profile_path, response_path = _write_profile_config(
        operator_root, work, profile_i05, stages, "copilot", MODEL
    )
    return {
        "work": work,
        "operator_root": operator_root,
        "config": config,
        "config_path": config_path,
        "profile_path": profile_path,
        "workflow_root": workflow_root,
        "stages": stages,
        "selected_stages": _spawn_stages(workflow_root / "bug.yaml"),
        "source_i05": source_i05,
        "response_path": response_path,
        "capture_path": work / "adapter-request.json",
        "source_capture_path": work / "adapter-request-no-profile.json",
        "codex_capture_path": work / "adapter-request-codex.json",
        "blocked_capture_path": work / "adapter-request-codex-blocked.json",
    }

def _write_fake_adapter(path):
    path.write_text(
        """#!/usr/bin/env python3
import json, os, sys
request = json.load(sys.stdin)
request['_global_provider_command'] = os.environ.get('LIFECYCLE_PROVIDER_COMMAND')
count_path = os.environ['B083_ADAPTER_CALL_COUNT']
try:
    with open(count_path, encoding='utf-8') as stream:
        count = int(stream.read())
except FileNotFoundError:
    count = 0
with open(count_path, 'w', encoding='utf-8') as stream:
    stream.write(str(count + 1))
with open(os.environ['B083_ADAPTER_CAPTURE'], 'w', encoding='utf-8') as stream:
    json.dump(request, stream, sort_keys=True)
if os.environ.get('B083_ADAPTER_MODE') == 'access-denied':
    print('access denied for selected synthetic route', file=sys.stderr)
    raise SystemExit(77)
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

def _write_fake_codex(path, *, malformed=False):
    first_page = (
        {"data": "malformed", "nextCursor": None}
        if malformed
        else {
            "data": [{
                "model": "b083-synthetic-first", "displayName": "First",
                "description": "first page", "hidden": False, "isDefault": True,
                "defaultReasoningEffort": "low",
                "supportedReasoningEfforts": [{"reasoningEffort": "low"}],
            }],
            "nextCursor": "page-two",
        }
    )
    second_page = {
        "data": [{
            "model": "b083-synthetic-catalog-model", "displayName": "Synthetic",
            "description": "fixture route", "hidden": False, "isDefault": False,
            "defaultReasoningEffort": "high",
            "supportedReasoningEfforts": [{"reasoningEffort": "high"}],
        }],
        "nextCursor": None,
    }
    Path(str(path) + ".catalog.json").write_text(
        json.dumps({"first_page": first_page, "second_page": second_page}),
        encoding="utf-8",
    )
    path.write_text(
        "#!/usr/bin/env python3\n"
        "import json, sys\n"
        "if sys.argv[1:] == ['--version']:\n print('codex-fixture/1'); raise SystemExit(0)\n"
        "with open(__file__ + '.catalog.json', encoding='utf-8') as stream:\n catalog=json.load(stream)\n"
        "for line in sys.stdin:\n"
        " request=json.loads(line); method=request.get('method')\n"
        " result={'userAgent':'codex-fixture/1'} if method == 'initialize' else "
        "(catalog['first_page'] if not request['params'].get('cursor') else catalog['second_page'])\n"
        " if method in ('initialize','model/list'):\n  print(json.dumps({'id':request['id'],'result':result}),flush=True)\n",
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

def _link_test_bench_tree(fixture):
    test_bench = fixture["work"] / "bench"
    test_scripts = test_bench / "scripts"
    test_scripts.mkdir(parents=True)
    (fixture["work"] / "skills").symlink_to(REPO_ROOT / "skills", target_is_directory=True)
    for item in (REPO_ROOT / "bench").iterdir():
        if item.name != "scripts":
            (test_bench / item.name).symlink_to(item, target_is_directory=item.is_dir())
    source_scripts = REPO_ROOT / "bench" / "scripts"
    for item in source_scripts.iterdir():
        if item.name != "materialize-routing-profile.py":
            (test_scripts / item.name).symlink_to(item, target_is_directory=item.is_dir())
    return test_bench, test_scripts

def _install_test_materializer(fixture):
    """Use the production route materializer in the isolated bench tree."""
    test_bench, test_scripts = _link_test_bench_tree(fixture)
    (test_scripts / "materialize-routing-profile.py").symlink_to(
        SCRIPTS_DIR / "materialize-routing-profile.py"
    )
    fixture["test_bench"] = test_bench
    fixture["test_scripts"] = test_scripts

def _set_profile_route(fixture, provider, model):
    profile = yaml.safe_load(fixture["profile_path"].read_text(encoding="utf-8"))
    profile["provider"] = provider
    for route in profile["routes"].values():
        route.update(model=model, effort="high")
    fixture["profile_path"].write_text(yaml.safe_dump(profile, sort_keys=False), encoding="utf-8")
    response = json.loads(fixture["response_path"].read_text(encoding="utf-8"))
    response.update(provider=provider, model=model, effort="high")
    fixture["response_path"].write_text(json.dumps(response), encoding="utf-8")
