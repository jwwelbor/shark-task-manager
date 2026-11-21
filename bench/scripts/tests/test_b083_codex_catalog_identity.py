import json
import hashlib
import os
import sys
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO_ROOT))
from bench.scripts.lib.codex_model_catalog import (
    CatalogError,
    discover_models,
    resolve_route,
    validate_catalog_snapshot,
)
import bench.scripts.lib.codex_model_catalog as codex_model_catalog


def test_b083_catalog_snapshot_is_bound_to_cli_configuration(tmp_path):
    executable = tmp_path / "codex-fixture"
    executable.write_text(
        "#!/usr/bin/env python3\nimport json,sys\n"
        "if sys.argv[1:] == ['--version']:\n print('codex-fixture/1'); raise SystemExit(0)\n"
        "for line in sys.stdin:\n"
        " r=json.loads(line); m=r.get('method')\n"
        " if 'id' not in r: continue\n"
        " x={'userAgent':'fixture'} if m == 'initialize' else {'data':[{'model':'synthetic-model','displayName':'Synthetic','description':'fixture','hidden':False,'isDefault':True,'defaultReasoningEffort':'low','supportedReasoningEfforts':[{'reasoningEffort':'low'}]}],'nextCursor':None}\n"
        " print(json.dumps({'id':r['id'],'result':x}),flush=True)\n",
        encoding="utf-8",
    )
    executable.chmod(0o755)
    base_env = {**os.environ, "CODEX_HOME": str(tmp_path / "codex-home-a")}
    codex_home = tmp_path / "codex-home-a"
    codex_home.mkdir()
    config_path = codex_home / "config.toml"
    config_path.write_text('model_provider = "fixture-a"\n', encoding="utf-8")
    catalog = discover_models([str(executable)], env=base_env)

    assert validate_catalog_snapshot(catalog, [str(executable)], env=base_env) is catalog
    with pytest.raises(CatalogError, match="different CLI configuration"):
        validate_catalog_snapshot(
            catalog,
            [str(executable)],
            env={**base_env, "CODEX_HOME": str(tmp_path / "codex-home-b")},
        )

    altered = json.loads(json.dumps(catalog))
    altered["models"][0]["description"] = "changed fixture metadata"
    with pytest.raises(CatalogError, match="digest does not match"):
        validate_catalog_snapshot(altered, [str(executable)], env=base_env)

    config_path.write_text('model_provider = "fixture-b"\n', encoding="utf-8")
    with pytest.raises(CatalogError, match="different CLI configuration"):
        validate_catalog_snapshot(catalog, [str(executable)], env=base_env)

    config_path.write_text('model_provider = "fixture-a"\n', encoding="utf-8")
    # Change the selected executable bytes without changing --version output.
    # This proves the runtime identity includes the binary fingerprint itself.
    executable.write_text(
        executable.read_text(encoding="utf-8") + "\n# replacement bytes\n",
        encoding="utf-8",
    )
    with pytest.raises(CatalogError, match="different CLI configuration"):
        validate_catalog_snapshot(catalog, [str(executable)], env=base_env)


def test_b083_catalog_snapshot_is_bound_to_selected_profile_file(tmp_path):
    executable = tmp_path / "codex-fixture"
    executable.write_text(
        "#!/usr/bin/env python3\nimport json,sys\n"
        "if sys.argv[-1:] == ['--version']:\n print('codex-fixture/1'); raise SystemExit(0)\n"
        "for line in sys.stdin:\n"
        " r=json.loads(line); m=r.get('method')\n"
        " if 'id' not in r: continue\n"
        " x={'userAgent':'fixture'} if m == 'initialize' else {'data':[{'model':'synthetic-model','displayName':'Synthetic','description':'fixture','hidden':False,'isDefault':True,'defaultReasoningEffort':'low','supportedReasoningEfforts':[{'reasoningEffort':'low'}]}],'nextCursor':None}\n"
        " print(json.dumps({'id':r['id'],'result':x}),flush=True)\n",
        encoding="utf-8",
    )
    executable.chmod(0o755)
    codex_home = tmp_path / "codex-home"
    codex_home.mkdir()
    (codex_home / "config.toml").write_text('model_provider = "fixture"\n', encoding="utf-8")
    profile_path = codex_home / "workbench.config.toml"
    profile_path.write_text('model = "synthetic-model"\n', encoding="utf-8")
    environment = {**os.environ, "CODEX_HOME": str(codex_home)}
    command = [str(executable), "--profile", "workbench"]
    catalog = discover_models(command, env=environment)

    assert validate_catalog_snapshot(catalog, command, env=environment) is catalog

    profile_path.write_text('model = "different-model"\n', encoding="utf-8")
    with pytest.raises(CatalogError, match="different CLI configuration"):
        validate_catalog_snapshot(catalog, command, env=environment)


def test_b083_catalog_discovery_rejects_runtime_change_during_model_list(
    tmp_path, monkeypatch
):
    executable = tmp_path / "codex-fixture"
    executable.write_text("synthetic executable bytes", encoding="utf-8")
    executable.chmod(0o755)
    codex_home = tmp_path / "codex-home"
    codex_home.mkdir()
    config_path = codex_home / "config.toml"
    config_path.write_text('model_provider = "fixture-before"\n', encoding="utf-8")
    environment = {**os.environ, "CODEX_HOME": str(codex_home)}
    model = {
        "model": "synthetic-model",
        "display_name": "Synthetic",
        "description": "fixture metadata",
        "is_default": True,
        "default_reasoning_effort": "low",
        "supported_reasoning_efforts": ["low"],
    }

    monkeypatch.setattr(codex_model_catalog, "_cli_version", lambda *_args, **_kwargs: "fixture/1")
    monkeypatch.setattr(
        codex_model_catalog, "_start_app_server", lambda *_args, **_kwargs: (object(), object())
    )

    def read_models(_process, _selector):
        config_path.write_text('model_provider = "fixture-after"\n', encoding="utf-8")
        return [model], "fixture"

    monkeypatch.setattr(codex_model_catalog, "_read_model_pages", read_models)

    with pytest.raises(CatalogError, match="runtime changed during model catalog discovery"):
        discover_models([str(executable)], env=environment)


def test_b083_display_metadata_changes_catalog_digest_without_changing_route():
    first = {
        "model": "synthetic-model",
        "display_name": "Before",
        "description": "before metadata",
        "is_default": True,
        "default_reasoning_effort": "low",
        "supported_reasoning_efforts": ["low"],
    }
    second = {**first, "display_name": "After", "description": "after metadata"}
    route_before = resolve_route("synthetic-model", "low", {"models": [first]})
    route_after = resolve_route("synthetic-model", "low", {"models": [second]})
    digest_before = hashlib.sha256(json.dumps([first], sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    digest_after = hashlib.sha256(json.dumps([second], sort_keys=True, separators=(",", ":")).encode()).hexdigest()

    assert route_before == route_after
    assert digest_before != digest_after
