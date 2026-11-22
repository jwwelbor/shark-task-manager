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
import shutil
import stat
import sys
import tempfile

import yaml

REPO_ROOT = Path(__file__).resolve().parents[3]
SCRIPTS_DIR = REPO_ROOT / "bench" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR / "tests" / "lib"))
from b083_production_route_fixture import (
    MODEL,
    SCENARIO_ID,
    _install_fake_tools,
    _load_benchmark_module,
    _set_profile_route,
    _setup_fixture,
    _write_fake_codex,
    require,
)
from b083_production_route_failure_cases import (
    run_expected_digest_mismatch_case,
    run_missing_catalog_block_case,
    run_post_dispatch_access_denied_case,
)






def _execute_profile(module, args, before_batch):
    original_materialize = module.materialize_batch_policy

    def materialize_after_hook(*args, **kwargs):
        if before_batch is not None:
            before_batch()
        return original_materialize(*args, **kwargs)

    module.materialize_batch_policy = materialize_after_hook
    output, errors = io.StringIO(), io.StringIO()
    try:
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
            exit_code = module.execute_profile(args, role="baseline", run_mode="pilot")
    finally:
        module.materialize_batch_policy = original_materialize
    return exit_code, output.getvalue(), errors.getvalue()


def _execute(
    module, fixture, run_id, run_config, capture_path, before_batch=None,
    expected_exit=0, expect_adapter=True,
):
    os.environ["B083_ADAPTER_CAPTURE"] = str(capture_path)
    count_path = capture_path.with_suffix(capture_path.suffix + ".count")
    os.environ["B083_ADAPTER_CALL_COUNT"] = str(count_path)
    args = argparse.Namespace(
        config=str(run_config), run_id=run_id, reps=1, scenario=SCENARIO_ID,
        acknowledge_provider_spend=True, max_cost_usd=1.0,
        max_wall_clock_seconds=120, max_generated_tasks=1,
        max_provider_calls=2, out=None, retry_incomplete=False,
    )
    exit_code, output, errors = _execute_profile(module, args, before_batch)
    require(
        exit_code == expected_exit,
        f"execute_profile({run_id}) returned {exit_code}, expected {expected_exit}: "
        f"{errors} {output[-3000:]}",
    )

    manifest_path = fixture["operator_root"] / "runs" / run_id / "pilot-benchmark-manifest.json"
    require(manifest_path.is_file(), f"{run_id} has no pilot-benchmark-manifest.json")
    if expect_adapter:
        require(capture_path.is_file(), f"real run-lifecycle did not invoke fake adapter for {run_id}")
        require(count_path.read_text(encoding="utf-8") == "1", f"expected one adapter call for {run_id}")
    else:
        require(not capture_path.exists(), f"route identity failure unexpectedly invoked adapter for {run_id}")
        require(not count_path.exists(), f"route identity failure unexpectedly counted an adapter call for {run_id}")
    return manifest_path, output + errors


def _assert_profiled_manifest(operator_root, manifest, stages, provider, model, source_profile_bytes):
    profile = manifest["identity"]["routing_profile"]
    expected_routes = {
        f"bug.yaml::{stage}": {"provider": provider, "model": model, "effort": "high"}
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


def _assert_profiled_adapter(manifest, request, provider, model):
    profile = manifest["identity"]["routing_profile"]
    expected_route = {"provider": provider, "model": model, "effort": "high"}
    require(
        {key: request[key] for key in ("provider", "model", "effort")} == expected_route,
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


def _write_no_profile_config(fixture):
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
    response = json.loads(fixture["response_path"].read_text(encoding="utf-8"))
    response.update(
        provider="codex", model="b083-synthetic-first", effort="low"
    )
    fixture["response_path"].write_text(json.dumps(response), encoding="utf-8")
    return source_config_path


def _route_fixture_workflows_to_codex(fixture):
    def set_route(stage, step):
        if not isinstance(step, dict) or step.get("action") != "spawn_agent":
            return
        if stage == "development":
            step.update(provider="codex")
            step.pop("model", None)
            step.pop("effort", None)
            return
        model = "b083-synthetic-catalog-model"
        effort = None
        if stage != "research":
            effort = "high"
        step.update(provider="codex", model=model, effort=effort)

    for workflow_path in sorted(fixture["workflow_root"].rglob("*.yaml")):
        workflow = yaml.safe_load(workflow_path.read_text(encoding="utf-8")) or {}
        for section in (workflow.get("steps"), workflow.get("status_metadata")):
            if not isinstance(section, dict):
                continue
            for name, step in section.items():
                set_route(name, step)
        workflow_path.write_text(yaml.safe_dump(workflow, sort_keys=False), encoding="utf-8")


@contextlib.contextmanager
def _codex_without_claude(codex):
    original_codex_command = os.environ.get("E40_CODEX_COMMAND")
    original_path = os.environ.get("PATH", "")
    path_without_claude = os.pathsep.join(
        entry
        for entry in original_path.split(os.pathsep)
        if entry and not (Path(entry) / "claude").is_file()
    )
    require(
        shutil.which("claude", path=path_without_claude) is None,
        "test path still exposes a Claude executable",
    )
    os.environ["E40_CODEX_COMMAND"] = json.dumps([str(codex)])
    os.environ["PATH"] = path_without_claude
    try:
        yield
    finally:
        if original_codex_command is None:
            os.environ.pop("E40_CODEX_COMMAND", None)
        else:
            os.environ["E40_CODEX_COMMAND"] = original_codex_command
        os.environ["PATH"] = original_path


def _run_no_profile_case(module, fixture):
    source_config_path = _write_no_profile_config(fixture)
    _route_fixture_workflows_to_codex(fixture)
    codex = fixture["work"] / "fake-codex-no-profile"
    _write_fake_codex(codex)
    with _codex_without_claude(codex):
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
    catalog = manifest["identity"].get("codex_catalog") or {}
    require(
        catalog.get("source") == "codex-app-server-model-list",
        "no-profile manifest has no runtime Codex catalog provenance",
    )
    require(
        catalog["models"][0]["description"] == "first page",
        "no-profile manifest omitted the runtime Codex model description",
    )
    require(
        catalog["models"][0]["display_name"] == "First",
        "no-profile manifest omitted the runtime Codex model display name",
    )
    require(
        manifest["identity"]["selected_codex_models"]["bug.yaml::development"]["description"]
        == "first page",
        "no-profile identity omitted selected model metadata",
    )
    require(
        manifest["identity"]["selected_codex_models"]["bug.yaml::development"]["display_name"]
        == "First",
        "no-profile identity omitted the selected model display name",
    )
    require(
        any(
            route["model"] == "b083-synthetic-first" and route["effort"] == "low"
            for route in identity["routes"]
        ),
        "no-profile default route did not resolve from the runtime catalog",
    )








def _replace_catalog_description(codex, old, new):
    catalog_path = Path(str(codex) + ".catalog.json")
    catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
    changed = False
    for page in (catalog.get("first_page"), catalog.get("second_page")):
        if not isinstance(page, dict):
            continue
        entries = page.get("data")
        if not isinstance(entries, list):
            continue
        for entry in entries:
            if isinstance(entry, dict) and entry.get("description") == old:
                entry["description"] = new
                changed = True
    require(changed, f"fake catalog description {old!r} was not found")
    catalog_path.write_text(json.dumps(catalog), encoding="utf-8")


def _assert_codex_manifest(fixture, manifest_path, source_profile_bytes):
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    request = json.loads(fixture["codex_capture_path"].read_text(encoding="utf-8"))
    require(
        request.get("_global_provider_command") is None,
        "Codex CLI selection leaked into the provider-neutral command variable",
    )
    _assert_profiled_manifest(
        fixture["operator_root"], manifest, fixture["selected_stages"], "codex", "b083-synthetic-catalog-model", source_profile_bytes
    )
    _assert_profiled_adapter(manifest, request, "codex", "b083-synthetic-catalog-model")
    require(
        manifest["identity"]["routing_profile"]["codex_catalog"]["source"] == "codex-app-server-model-list",
        "manifest does not retain Codex catalog provenance",
    )
    require(
        manifest["identity"]["routing_profile"]["codex_catalog"]["models"][1]["description"] == "fixture route",
        "manifest omitted the selected Codex model description",
    )
    require(
        manifest["identity"]["routing_profile"]["codex_catalog"]["models"][1]["display_name"] == "Synthetic",
        "manifest omitted the selected Codex model display name",
    )
    require(
        manifest["identity"]["routing_profile"]["selected_codex_models"]["bug.yaml::development"]["description"]
        == "fixture route",
        "manifest identity omitted the selected Codex model metadata",
    )
    require(
        manifest["identity"]["routing_profile"]["selected_codex_models"]["bug.yaml::development"]["display_name"]
        == "Synthetic",
        "manifest identity omitted the selected Codex model display name",
    )
    return manifest


def _assert_catalog_change_rebinds_identity(manifest, changed_manifest_path):
    changed_manifest = json.loads(changed_manifest_path.read_text(encoding="utf-8"))
    original_profile = manifest["identity"]["routing_profile"]
    changed_profile = changed_manifest["identity"]["routing_profile"]
    require(
        original_profile["provider_routing"]["routing_digest"]
        == changed_profile["provider_routing"]["routing_digest"],
        "changing only Codex display metadata changed the route identity",
    )
    require(
        original_profile["codex_catalog"]["catalog_digest"]
        != changed_profile["codex_catalog"]["catalog_digest"],
        "changing Codex display metadata did not change catalog provenance",
    )
    require(
        manifest["identity"]["identity_digest"]
        != changed_manifest["identity"]["identity_digest"],
        "manifest identity did not bind changed catalog provenance",
    )


def _run_codex_propagation_case(module, fixture):
    _set_profile_route(fixture, "codex", "b083-synthetic-catalog-model")
    source_profile_bytes = fixture["profile_path"].read_bytes()
    codex = fixture["work"] / "fake-codex"
    _write_fake_codex(codex)
    with _codex_without_claude(codex):
        manifest_path, _output = _execute(
            module,
            fixture,
            "b083-route-codex-production",
            fixture["config_path"],
            fixture["codex_capture_path"],
            before_batch=lambda: _replace_catalog_description(
                codex, "fixture route", "changed after identity"
            ),
        )
        require(
            "changed after identity" in Path(str(codex) + ".catalog.json").read_text(encoding="utf-8"),
            "same-run catalog counterfactual did not mutate after identity capture",
        )
        changed_catalog = fixture["work"] / "fake-codex-changed-description"
        _write_fake_codex(changed_catalog)
        _replace_catalog_description(changed_catalog, "fixture route", "fixture route revised")
        os.environ["E40_CODEX_COMMAND"] = json.dumps([str(changed_catalog)])
        changed_manifest_path, _output = _execute(
            module,
            fixture,
            "b083-route-codex-metadata-change",
            fixture["config_path"],
            fixture["work"] / "adapter-request-codex-metadata.json",
        )
    manifest = _assert_codex_manifest(fixture, manifest_path, source_profile_bytes)
    _assert_catalog_change_rebinds_identity(manifest, changed_manifest_path)


def _run_profiled_copilot_case(module, fixture):
    source_profile_bytes = fixture["profile_path"].read_bytes()

    def mutate_source_profile_after_manifest_capture():
        profile = yaml.safe_load(fixture["profile_path"].read_text(encoding="utf-8"))
        profile["routes"]["development"]["model"] = "b083-mutated-after-identity"
        fixture["profile_path"].write_text(yaml.safe_dump(profile, sort_keys=False), encoding="utf-8")

    manifest_path, _output = _execute(
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
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    request = json.loads(fixture["capture_path"].read_text(encoding="utf-8"))
    _assert_profiled_manifest(
        fixture["operator_root"], manifest, fixture["selected_stages"], "copilot", MODEL,
        source_profile_bytes,
    )
    _assert_profiled_adapter(manifest, request, "copilot", MODEL)


def _run_secondary_route_cases(module, fixture):
    no_profile_path, _ = _run_no_profile_case(module, fixture)
    source_manifest = json.loads(no_profile_path.read_text(encoding="utf-8"))
    source_request = json.loads(fixture["source_capture_path"].read_text(encoding="utf-8"))
    _assert_no_profile_route(source_manifest, source_request)
    _run_codex_propagation_case(module, fixture)
    run_post_dispatch_access_denied_case(
        module,
        fixture,
        execute=_execute,
        codex_without_claude=_codex_without_claude,
        assert_profiled_manifest=_assert_profiled_manifest,
        assert_profiled_adapter=_assert_profiled_adapter,
    )
    for mismatch_kind in ("routing", "catalog"):
        run_expected_digest_mismatch_case(
            module,
            fixture,
            mismatch_kind,
            execute=_execute,
            codex_without_claude=_codex_without_claude,
        )
    run_missing_catalog_block_case(
        fixture,
        module=module,
        execute=_execute,
        codex_without_claude=_codex_without_claude,
    )


def main():
    with tempfile.TemporaryDirectory(prefix="tc123-b083-") as temporary:
        fixture = _setup_fixture(Path(temporary))
        _install_fake_tools(fixture)
        module = _load_benchmark_module(fixture)
        _run_profiled_copilot_case(module, fixture)
        _run_secondary_route_cases(module, fixture)
    print("TC-123 (B083/F3: immutable profile identity and all-stage routes reach the real batch and adapter path) PASS")


if __name__ == "__main__":
    main()
