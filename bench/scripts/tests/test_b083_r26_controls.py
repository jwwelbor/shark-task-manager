"""Regression and counterfactual controls for B083 R26 obligations."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

import pytest
import yaml

SCRIPTS = Path(__file__).resolve().parents[1]
REPO_ROOT = SCRIPTS.parents[1]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))
if str(SCRIPTS / "tests" / "lib") not in sys.path:
    sys.path.insert(0, str(SCRIPTS / "tests" / "lib"))

from bench.scripts.lib.e40_benchmark import (
    _effective_profile_route,
    _identity_routing_fields,
    _live_stage_effort,
    _route_action_mapping,
    _validate_effective_routes,
    live_workflow_stages,
    matrix_workflow_families,
    package_workflow_families,
    resolve_row_workflow_families,
)
from lifecycle_adapter_test_helper import load_adapter_module

_adapter = load_adapter_module()
AdapterError = _adapter["AdapterError"]
decode_envelope = _adapter["decode_envelope"]
project_result = _adapter["project_result"]
_codex_route = _adapter["_codex_route"]


def test_route_action_mapping_matches_keyed_worker_semantics(tmp_path):
    wf = tmp_path / "workflow.yaml"
    wf.write_text(
        yaml.safe_dump(
            {
                "steps": {
                    "s_spawn": {"action": "spawn_agent", "agent": "dev", "provider": "codex", "model": "m"},
                    "s_resume": {"action": "check_or_resume", "agent": "dev", "provider": "codex", "model": "m"},
                    "s_adv_agent": {"action": "advance_status", "agent": "dev", "provider": "codex", "model": "m"},
                    "s_adv_no_agent": {"action": "advance_status", "provider": "codex", "model": "m"},
                    "s_empty_ok": {"action": "", "agent": "dev", "prompt": "do work", "provider": "codex", "model": "m"},
                    "s_empty_no_prompt": {"action": "", "agent": "dev", "provider": "codex", "model": "m"},
                    "s_empty_no_agent": {"action": "", "prompt": "do work", "provider": "codex", "model": "m"},
                    "s_pause": {"action": "pause", "agent": "dev"},
                    "s_archive": {"action": "archive"},
                    "s_term": {"action": "spawn_agent", "agent": "dev", "terminal": True},
                }
            }
        ),
        encoding="utf-8",
    )
    stages = live_workflow_stages(wf)
    assert stages == {"s_spawn", "s_resume", "s_adv_agent", "s_empty_ok"}
    assert _route_action_mapping({"orchestrator_action": {"action": "check_or_resume", "agent_type": "qa"}}) is not None
    assert _route_action_mapping({"orchestrator_action": {"action": "advance_status", "agent_type": ""}}) is None


@pytest.mark.parametrize("effort", ["HiGh", "FixtureBoost"])
def test_codex_effort_preserved_verbatim_in_adapter_and_materializer(tmp_path, effort):
    # 1. Adapter route helper preserves mixed-case effort verbatim.
    model, got_effort = _codex_route({"model": "synthetic-m", "effort": effort})
    assert model == "synthetic-m"
    assert got_effort == effort

    # 2. e40_benchmark helpers preserve mixed-case effort for codex/openai and fold for other providers.
    assert _live_stage_effort(effort, "codex", "development") == effort
    assert _live_stage_effort(effort, "openai", "development") == effort
    assert _live_stage_effort("HiGh", "anthropic", "development") == "high"

    profile_route = _effective_profile_route(
        "development",
        {"model": "synthetic-m", "effort": effort},
        "codex",
        None,
        None,
        False,
    )
    assert profile_route == {"provider": "codex", "model": "synthetic-m", "effort": effort}

    # Mismatched casing against profile is rejected even when case-insensitive would match.
    prof_obj = {
        "schema_version": "1.0",
        "profile_id": "synthetic-effort",
        "provider": "codex",
        "description": "Preserve mixed-case effort",
        "routes": {"development": {"model": "synthetic-m", "effort": effort}},
    }
    _validate_effective_routes(prof_obj, {"development": profile_route})
    if effort != effort.lower():
        with pytest.raises(Exception, match="does not match"):
            _validate_effective_routes(
                prof_obj,
                {"development": {"provider": "codex", "model": "synthetic-m", "effort": effort.lower()}},
            )

    # 3. materialize-routing-profile.py preserves exact effort in both workflow and profile modes.
    fake_codex = tmp_path / "fake_codex"
    fake_codex.write_text(
        f"""#!/usr/bin/env python3
import json, sys
if sys.argv[1:] == ["--version"]:
    print("codex-fixture/1")
    raise SystemExit(0)
for line in sys.stdin:
    req = json.loads(line)
    if "id" not in req:
        continue
    if req.get("method") == "initialize":
        res = {{"userAgent": "codex-fixture/1"}}
    elif req.get("method") == "model/list":
        res = {{"data": [{{"model": "synthetic-m", "displayName": "M", "description": "m", "hidden": False, "isDefault": True, "defaultReasoningEffort": {effort!r}, "supportedReasoningEfforts": [{{"reasoningEffort": {effort!r}}}]}}], "nextCursor": None}}
    else:
        continue
    print(json.dumps({{"id": req["id"], "result": res}}), flush=True)
""",
        encoding="utf-8",
    )
    fake_codex.chmod(0o755)
    env = {**os.environ, "E40_CODEX_COMMAND": json.dumps([str(fake_codex)])}

    wf = tmp_path / "bug.yaml"
    wf.write_text(
        yaml.safe_dump({"steps": {"development": {"action": "spawn_agent", "provider": "codex", "model": "synthetic-m", "effort": effort}}}),
        encoding="utf-8",
    )
    res_wf = subprocess.run(
        [sys.executable, str(SCRIPTS / "materialize-routing-profile.py"), "--workflow-root", str(wf)],
        cwd=tmp_path,
        env=env,
        capture_output=True,
        text=True,
        check=True,
    )
    wf_out = json.loads(res_wf.stdout)
    assert wf_out["selected_codex_models"]["bug.yaml::development"]["model"] == "synthetic-m"
    assert yaml.safe_load(wf.read_text(encoding="utf-8"))["steps"]["development"]["effort"] == effort

    wf.write_text(
        yaml.safe_dump({"steps": {"development": {"action": "spawn_agent", "provider": "anthropic", "model": "claude-sonnet", "effort": "high"}}}),
        encoding="utf-8",
    )
    profile_file = tmp_path / "profile.yaml"
    profile_file.write_text(
        yaml.safe_dump({
            "schema_version": "1.0",
            "profile_id": "synthetic-effort",
            "provider": "codex",
            "description": "Preserve mixed-case effort",
            "routes": {"development": {"model": "synthetic-m", "effort": effort}},
        }),
        encoding="utf-8",
    )
    res_prof = subprocess.run(
        [sys.executable, str(SCRIPTS / "materialize-routing-profile.py"), "--workflow-root", str(wf), "--profile", str(profile_file)],
        cwd=tmp_path,
        env=env,
        capture_output=True,
        text=True,
        check=True,
    )
    prof_out = json.loads(res_prof.stdout)
    assert prof_out["selected_codex_models"]["bug.yaml::development"]["model"] == "synthetic-m"
    assert yaml.safe_load(wf.read_text(encoding="utf-8"))["steps"]["development"]["effort"] == effort


def test_descendant_family_scope_includes_cascaded_families(tmp_path):
    epic_pkg = REPO_ROOT / "bench" / "scenarios" / "packages" / "py-epic-task-organization" / "package.yaml"
    feature_pkg = REPO_ROOT / "bench" / "scenarios" / "packages" / "py-feature-recurring-tasks" / "package.yaml"
    bug_pkg = REPO_ROOT / "bench" / "scenarios" / "packages" / "py-bug-due-date-boundary" / "package.yaml"

    assert matrix_workflow_families([{"family": "epic", "package_path": str(epic_pkg)}]) == ["epic", "feature", "task"]
    assert matrix_workflow_families([{"family": "feature", "package_path": str(feature_pkg)}]) == ["feature", "task"]
    assert matrix_workflow_families([{"family": "bug", "package_path": str(bug_pkg)}]) == ["bug"]
    assert package_workflow_families({"entity_family": "feature"}) == {"feature", "task"}


def test_generated_worker_id_bounds_long_model_deterministically():
    long_model = "m" * 251
    req = {
        "provider": "codex",
        "model": long_model,
        "prompt": "hi",
        "allowed_outcomes": ["pass"],
    }
    env = {"kind": "final", "recommended_outcome": "pass", "evidence": []}
    res = project_result(env, req, "sess-1", "a" * 64, 2)
    expected_digest = hashlib.sha256(
        json.dumps({"model": long_model, "provider": "codex"}, sort_keys=True, separators=(",", ":")).encode("utf-8")
    ).hexdigest()
    assert res["worker_id"] == f"e40.worker.v1.{expected_digest}"
    assert len(res["worker_id"]) <= 256

    # Invalid nonempty envelope worker_id fails even when LIFECYCLE_WORKER_ID is valid.
    with pytest.MonkeyPatch.context() as mp:
        mp.setenv("LIFECYCLE_WORKER_ID", "valid-host")
        with pytest.raises(AdapterError):
            project_result({**env, "worker_id": "bad whitespace"}, req, "sess-1", "a" * 64, 2)


def test_copilot_assistant_precedence_and_fallback_decision_5055():
    stream_both = "\n".join(
        [
            "RECOMMENDED OUTCOME: pass",
            json.dumps({"type": "assistant.message", "data": {"content": json.dumps({"kind": "final", "recommended_outcome": "fail", "evidence": []})}}),
            json.dumps({"type": "result", "data": {"exitCode": 0}}),
        ]
    )
    assert decode_envelope(stream_both)["recommended_outcome"] == "fail"

    stream_fallback = "\n".join(
        [
            "RECOMMENDED OUTCOME: pass",
            "RECOMMENDED OUTCOME: fail",
            json.dumps({"type": "assistant.message", "data": {"content": "Just prose without control."}}),
            json.dumps({"type": "result", "data": {"exitCode": 0}}),
        ]
    )
    assert decode_envelope(stream_fallback)["recommended_outcome"] == "pass"


def test_structured_step_rejects_text_fallback_and_preserves_raw_final():
    req = {
        "provider": "codex",
        "model": "m",
        "prompt": "hi",
        "result_contract": "gate_result_v1",
        "allowed_outcomes": ["pass", "fail"],
    }
    env = {"kind": "final", "recommended_outcome": "pass", "evidence": []}
    with pytest.raises(AdapterError, match="structured step requires a JSON control envelope"):
        project_result(env, req, "sess-1", "a" * 64, 2, raw_final=None)

    raw_bytes = '{"kind":"final","recommended_outcome":"pass","evidence":[],"gate_result":{"schema_version":1,"summary":"ok","findings":[],"kickbacks":[]}}'
    res = project_result(env, req, "sess-1", "a" * 64, 2, raw_final=raw_bytes)
    assert res["raw_final"] == raw_bytes


def _load_derive_metrics():
    content = (SCRIPTS / "evaluate-lifecycle.sh").read_text(encoding="utf-8")
    ns = {}
    body = content.split("<<'PYEOF'\n", 1)[1].rsplit("\nPYEOF", 1)[0]
    # Execute only up to content_digest to get derive_metrics without argparse.
    snippet = body.split("def content_digest(", 1)[0]
    snippet = "def metric(" + snippet.split("def metric(", 1)[1]
    exec(snippet, ns)
    return ns["derive_metrics"]


def test_evaluate_lifecycle_cost_availability_for_unmapped_and_measured_zero():
    derive_metrics = _load_derive_metrics()
    lifecycle = {
        "stages": [
            {
                "dispatch_ordinal": 1,
                "stage": "dev",
                "elapsed_seconds": 1.0,
                "cost_usd": 0.0,
                "errors": [{"kind": "unmapped_provider", "provider": "codex_cli"}],
            }
        ]
    }
    ev = derive_metrics(lifecycle, {}, {})
    assert ev["provider_cost"] == {
        "available": False,
        "value": None,
        "detail": "sum of retained stage and judge cost_usd",
    }

    # Missing total_cost slot on Copilot also yields unavailable.
    lifecycle["stages"][0]["errors"] = [{"kind": "usage_slot_unavailable", "slot": "total_cost"}]
    ev_copilot = derive_metrics(lifecycle, {}, {})
    assert ev_copilot["provider_cost"]["available"] is False
    assert ev_copilot["provider_cost"]["value"] is None

    # Genuinely measured 0.0 (even with an unrelated missing token slot) yields available=True, value=0.0.
    lifecycle["stages"][0]["errors"] = [{"kind": "usage_slot_unavailable", "slot": "cache_read_input_tokens"}]
    ev_zero = derive_metrics(lifecycle, {}, {})
    assert ev_zero["provider_cost"]["available"] is True
    assert ev_zero["provider_cost"]["value"] == 0.0


def test_resolve_row_workflow_families_handles_inferred_and_synthetic_fixtures(tmp_path):
    wf_dir = tmp_path / "workflow"
    wf_dir.mkdir()
    (wf_dir / "feature.yaml").write_text("steps: {}\n", encoding="utf-8")

    # Graphless feature fixture with only feature.yaml drops inferred task.
    assert resolve_row_workflow_families(wf_dir, ["feature", "task"], explicit_graph=False) == ["feature"]
    # Explicit entity graph keeps all mapped families so missing files fail closed.
    assert resolve_row_workflow_families(wf_dir, ["feature", "task"], explicit_graph=True) == ["feature", "task"]
    # Synthetic custom family with no mapping falls back to empty (scan *.yaml).
    assert resolve_row_workflow_families(wf_dir, ["family-e2e-comparison"], explicit_graph=False) == []


def test_identity_routing_fields_skips_provider_command_for_custom_adapter_without_profile(tmp_path, monkeypatch):
    wf_dir = tmp_path / "workflow"
    wf_dir.mkdir()
    (wf_dir / "bug.yaml").write_text(
        yaml.safe_dump({"steps": {"dev": {"action": "spawn_agent", "provider": "openai", "model": "gpt-5"}}}),
        encoding="utf-8",
    )
    spy_log = tmp_path / "provider.log"
    spy = tmp_path / "provider_spy.sh"
    spy.write_text(f"#!/usr/bin/env bash\necho called >> {str(spy_log)!r}\nexit 97\n", encoding="utf-8")
    spy.chmod(0o755)
    monkeypatch.delenv("E40_CODEX_COMMAND", raising=False)
    config = {
        "workflow_root": str(wf_dir),
        "runtime": {"lifecycle_adapter": str(spy), "provider_command": [str(spy)]},
    }
    fields = _identity_routing_fields(config, tmp_path, ["bug"])
    assert fields["codex_catalog"] is None
    assert not spy_log.exists()
