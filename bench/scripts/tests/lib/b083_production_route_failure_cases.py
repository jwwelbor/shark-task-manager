"""Codex production-route failure cases for B083."""

import argparse
import io
import json
import os
import subprocess
import sys
from contextlib import contextmanager, redirect_stderr, redirect_stdout
from pathlib import Path

from b083_production_route_fixture import (
    SCENARIO_ID,
    _set_profile_route,
    _write_fake_codex,
    require,
)

SCRIPTS_DIR = Path(__file__).resolve().parents[2]


@contextmanager
def _access_denied_mode():
    previous = os.environ.get("B083_ADAPTER_MODE")
    os.environ["B083_ADAPTER_MODE"] = "access-denied"
    try:
        yield
    finally:
        if previous is None:
            os.environ.pop("B083_ADAPTER_MODE", None)
        else:
            os.environ["B083_ADAPTER_MODE"] = previous


def run_post_dispatch_access_denied_case(
    module, fixture, *, execute, codex_without_claude,
    assert_profiled_manifest, assert_profiled_adapter,
):
    _set_profile_route(fixture, "codex", "b083-synthetic-catalog-model")
    source_profile_bytes = fixture["profile_path"].read_bytes()
    codex = fixture["work"] / "fake-codex-access-denied"
    _write_fake_codex(codex)
    capture_path = fixture["work"] / "adapter-request-codex-access-denied.json"
    with _access_denied_mode(), codex_without_claude(codex):
        manifest_path, _output = execute(
            module,
            fixture,
            "b083-route-codex-access-denied",
            fixture["config_path"],
            capture_path,
        )

    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    request = json.loads(capture_path.read_text(encoding="utf-8"))
    assert_profiled_manifest(
        fixture["operator_root"], manifest, fixture["selected_stages"],
        "codex", "b083-synthetic-catalog-model", source_profile_bytes,
    )
    assert_profiled_adapter(
        manifest, request, "codex", "b083-synthetic-catalog-model"
    )

    lifecycle_path = (
        fixture["operator_root"] / "runs" / "b083-route-codex-access-denied"
        / "scenarios" / SCENARIO_ID / "1" / "lifecycle.jsonl"
    )
    require(lifecycle_path.is_file(), "post-dispatch denial did not retain lifecycle evidence")
    lifecycle = json.loads(lifecycle_path.read_text(encoding="utf-8").splitlines()[0])
    require(len(lifecycle["dispatches"]) == 1, "post-dispatch denial triggered another dispatch")
    attempted = lifecycle["dispatches"][0]["response"]
    require(
        {key: attempted[key] for key in ("provider", "model", "effort")}
        == {"provider": "codex", "model": "b083-synthetic-catalog-model", "effort": "high"},
        "post-dispatch denial did not retain the exact attempted route",
    )
    require(
        "access denied" in lifecycle["outcome"]["reason"],
        "post-dispatch denial evidence omitted the provider failure",
    )


def run_expected_digest_mismatch_case(
    module, fixture, mismatch_kind, *, execute, codex_without_claude,
):
    _set_profile_route(fixture, "codex", "b083-synthetic-catalog-model")
    codex = fixture["work"] / f"fake-codex-{mismatch_kind}-digest-mismatch"
    _write_fake_codex(codex)
    run_id = f"b083-route-{mismatch_kind}-digest-mismatch"
    capture_path = fixture["work"] / f"adapter-request-{mismatch_kind}-digest-mismatch.json"
    variable = (
        "E40_EXPECTED_ROUTING_DIGEST"
        if mismatch_kind == "routing"
        else "E40_EXPECTED_CODEX_CATALOG_DIGEST"
    )
    original_environment_builder = module.execution_environment

    def mismatched_environment(*args, **kwargs):
        environment = original_environment_builder(*args, **kwargs)
        environment[variable] = "b083-forced-digest-mismatch"
        return environment

    module.execution_environment = mismatched_environment
    try:
        with codex_without_claude(codex):
            _manifest_path, output = execute(
                module,
                fixture,
                run_id,
                fixture["config_path"],
                capture_path,
                expected_exit=4,
                expect_adapter=False,
            )
    finally:
        module.execution_environment = original_environment_builder

    _assert_digest_mismatch_retained(fixture, run_id, mismatch_kind, output)


def _assert_digest_mismatch_retained(fixture, run_id, mismatch_kind, output):
    failed_root = fixture["operator_root"] / "runs" / run_id / ".failed-attempts" / SCENARIO_ID
    retained_pairs = list(failed_root.iterdir()) if failed_root.is_dir() else []
    require(len(retained_pairs) == 1, f"{mismatch_kind} digest mismatch did not retain one failed pair")
    routing_identity_path = retained_pairs[0] / "routing-identity.json"
    require(routing_identity_path.is_file(), f"{mismatch_kind} mismatch lost route identity evidence")
    identity = json.loads(routing_identity_path.read_text(encoding="utf-8"))
    require(
        identity.get("codex_catalog", {}).get("source") == "codex-app-server-model-list",
        f"{mismatch_kind} mismatch did not retain runtime catalog provenance",
    )
    require(
        "differs" in output.lower(),
        f"{mismatch_kind} mismatch output omitted its failure reason",
    )
    invalid_path = fixture["operator_root"] / "runs" / run_id / "invalid" / "index.jsonl"
    invalid = json.loads(invalid_path.read_text(encoding="utf-8").splitlines()[0])
    require(
        invalid["invalidity_reasons"] == ["route_defect"],
        f"{mismatch_kind} mismatch was not classified as a route defect",
    )


def run_missing_catalog_block_case(
    fixture, *, module, execute, codex_without_claude
):
    _set_profile_route(fixture, "codex", "b083-synthetic-catalog-model")
    codex = fixture["work"] / "fake-codex-malformed"
    _write_fake_codex(codex, malformed=True)
    result = subprocess.run(
        [sys.executable, str(SCRIPTS_DIR / "materialize-routing-profile.py"),
         "--workflow-root", str(fixture["workflow_root"]),
         "--profile", str(fixture["profile_path"])],
        text=True, capture_output=True, check=False,
        env={**os.environ, "E40_CODEX_COMMAND": json.dumps([str(codex)])},
    )
    require(result.returncode == 1, "malformed Codex catalog unexpectedly resolved")
    failure = json.loads(result.stdout)
    require(failure.get("cause_class") == "route_defect", "catalog failure was not a route defect")
    require(failure.get("dispatch_count") == 0, "catalog failure allowed a provider dispatch")
    require(
        "malformed model data" in failure.get("reason", ""),
        "blocked production route did not report malformed app-server data",
    )
    require(not fixture["blocked_capture_path"].exists(), "failed discovery dispatched the adapter")

    run_missing_catalog_execute_case(module, fixture, codex, execute, codex_without_claude)
    run_missing_catalog_preflight_case(
        module, fixture, codex, codex_without_claude
    )


def run_missing_catalog_execute_case(module, fixture, codex, execute, codex_without_claude):
    run_id = "b083-missing-catalog-route"
    with codex_without_claude(codex):
        _manifest_path, output = execute(
            module, fixture, run_id, fixture["config_path"],
            fixture["blocked_capture_path"], expected_exit=4, expect_adapter=False,
        )
    require("route_defect before adapter dispatch" in output, "catalog failure bypassed route ledger")
    manifest_path = fixture["operator_root"] / "runs" / run_id / "pilot-benchmark-manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    route_failure = manifest["identity"].get("routing_resolution")
    require(route_failure.get("cause_class") == "route_defect", "manifest lost catalog route failure")
    failed_root = fixture["operator_root"] / "runs" / run_id / ".failed-attempts" / SCENARIO_ID
    retained_pairs = list(failed_root.iterdir()) if failed_root.is_dir() else []
    require(len(retained_pairs) == 1, "catalog failure did not retain one failed pair")
    route_record = json.loads((retained_pairs[0] / "routing-identity.json").read_text(encoding="utf-8"))
    require(
        route_record.get("cause_class") == "route_defect"
        and route_record.get("dispatch_count") == 0,
        "catalog failure was not retained as a zero-dispatch route defect",
    )


def run_missing_catalog_preflight_case(module, fixture, codex, codex_without_claude):
    output_root = fixture["operator_root"] / "preflight" / "b083-catalog-failure"
    args = argparse.Namespace(
        config=str(fixture["config_path"]), out=str(output_root), reps=1,
        scenario=SCENARIO_ID,
    )
    with codex_without_claude(codex):
        with redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()):
            exit_code = module.cmd_preflight(args)

    result = json.loads((output_root / "preflight-result.json").read_text(encoding="utf-8"))
    records = [
        json.loads(line)
        for line in Path(result["resolution_ledger_path"]).read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]
    require(
        exit_code == module.PREFLIGHT_BLOCKED_EXIT,
        "preflight did not fail closed on Codex catalog discovery failure",
    )
    require(result["status"] == "blocked", "preflight did not retain its ledger blocker status")
    require(result["preview_exit_code"] == 0, "ledger-recorded route failure crashed the preview runner")
    require(
        len(records) == 1
        and records[0].get("cause_class") == "route_defect"
        and records[0].get("dispatch_count") == 0,
        "preflight did not retain a zero-dispatch route-defect ledger record",
    )
