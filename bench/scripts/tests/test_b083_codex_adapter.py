"""Codex adapter routing tests for B083."""

import argparse
import json
from pathlib import Path
import sys

import pytest

TEST_LIB = Path(__file__).resolve().parent / "lib"
sys.path.insert(0, str(TEST_LIB))
from lifecycle_adapter_test_helper import load_adapter_module

_adapter = load_adapter_module()
command_for = _adapter["command_for"]
AdapterError = _adapter["AdapterError"]
provider_environment = _adapter["provider_environment"]


def make_args(provider_command=None):
    return argparse.Namespace(provider_command=provider_command)


# ---------------------------------------------------------------------------
# B083: E40 pins the Codex model and reasoning effort at the adapter boundary.
# ---------------------------------------------------------------------------
@pytest.mark.parametrize("model", ["fixture-codex-model-a", "fixture-codex-model-b"])
def test_b083_codex_command_preserves_model_and_effort(model):
    request = {"provider": "openai", "model": model, "effort": "high"}
    cmd = command_for(make_args(), request)

    assert cmd[:3] == ["codex", "exec", "--json"]
    assert cmd[cmd.index("-m") + 1] == model
    assert cmd[cmd.index("-c") + 1] == 'model_reasoning_effort="high"'


def test_b083_codex_executable_override_preserves_model_and_effort():
    request = {"provider": "codex", "model": "fixture-codex-model-b", "effort": "xhigh"}
    cmd = command_for(make_args("/custom/codex"), request)

    assert cmd[:2] == ["/custom/codex", "exec"]
    assert cmd[cmd.index("-m") + 1] == "fixture-codex-model-b"
    assert cmd[cmd.index("-c") + 1] == 'model_reasoning_effort="xhigh"'


def test_b083_explicit_codex_executable_override_precedes_environment_command(monkeypatch):
    monkeypatch.setenv(
        "LIFECYCLE_PROVIDER_COMMAND",
        json.dumps(["/environment/codex", "exec", "--json", "-m", "fixture-codex-model-b"]),
    )
    request = {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"}

    command = command_for(make_args("/explicit/codex"), request)

    assert command[0] == "/explicit/codex"
    assert command[command.index("-m") + 1] == "fixture-codex-model-b"


def test_b083_codex_base_command_uses_catalog_bound_cli_and_exact_route(monkeypatch):
    selected_cli = ["/selected/codex", "--profile", "workbench"]
    monkeypatch.setenv("E40_CODEX_COMMAND", json.dumps(selected_cli))
    monkeypatch.delenv("LIFECYCLE_PROVIDER_COMMAND", raising=False)
    request = {
        "provider": "codex",
        "model": "fixture-codex-model-b",
        "effort": "xhigh",
    }

    command = command_for(make_args(), request)

    assert command == [
        "/selected/codex",
        "--profile",
        "workbench",
        "exec",
        "--json",
        "-m",
        "fixture-codex-model-b",
        "-c",
        'model_reasoning_effort="xhigh"',
    ]


@pytest.mark.parametrize(
    "route_flags",
    [
        ["--model", "fixture-codex-model-a"],
        ["-mfixture-codex-model-a"],
        ["-c", 'model="fixture-codex-model-a"'],
        ["--config=model_reasoning_effort=low"],
        ["--config", 'model_provider="other"'],
        ["--oss"],
    ],
)
def test_b083_codex_base_command_rejects_route_overrides(monkeypatch, route_flags):
    monkeypatch.setenv("E40_CODEX_COMMAND", json.dumps(["/selected/codex", *route_flags]))
    monkeypatch.delenv("LIFECYCLE_PROVIDER_COMMAND", raising=False)

    with pytest.raises(AdapterError, match="cannot preserve the pinned Codex route"):
        command_for(
            make_args(),
            {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"},
        )


def test_b083_codex_profile_value_exec_is_not_a_subcommand(monkeypatch):
    selected_cli = ["/selected/codex", "--profile", "exec"]
    monkeypatch.setenv("E40_CODEX_COMMAND", json.dumps(selected_cli))
    monkeypatch.delenv("LIFECYCLE_PROVIDER_COMMAND", raising=False)

    command = command_for(
        make_args(),
        {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"},
    )
    subcommand_index = _adapter["_codex_exec_subcommand_index"](
        [*selected_cli, "exec", "--json"]
    )

    assert subcommand_index == 3
    assert command[:4] == ["/selected/codex", "--profile", "exec", "exec"]


def test_b083_selected_codex_cli_wins_over_provider_neutral_command(monkeypatch):
    monkeypatch.setenv("E40_CODEX_COMMAND", json.dumps(["/selected/codex"]))
    monkeypatch.setenv("LIFECYCLE_PROVIDER_COMMAND", json.dumps(["/other/codex"]))
    request = {
        "provider": "codex",
        "model": "fixture-codex-model-b",
        "effort": "high",
    }

    command = command_for(make_args(), request)

    assert command[:2] == ["/selected/codex", "exec"]
    assert command[command.index("-m") + 1] == request["model"]


def test_b083_selected_codex_cli_ignores_provider_neutral_exec_override(monkeypatch):
    monkeypatch.setenv("E40_CODEX_COMMAND", json.dumps(["/selected/codex"]))
    monkeypatch.setenv(
        "LIFECYCLE_PROVIDER_COMMAND",
        json.dumps([
            "/other/codex", "exec", "--json", "-m", "fixture-codex-model-b",
            "-c", 'model_reasoning_effort="high"',
        ]),
    )

    command = command_for(
        make_args(),
        {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"},
    )

    assert command[:2] == ["/selected/codex", "exec"]
    assert command[command.index("-m") + 1] == "fixture-codex-model-b"


@pytest.mark.parametrize(
    ("model", "effort", "message"),
    [
        ("", "high", "requires a non-empty model"),
        ("fixture-codex-model-b", "", "non-empty reasoning effort"),
    ],
)
def test_b083_codex_rejects_invalid_model_or_effort(model, effort, message):
    with pytest.raises(AdapterError, match=message):
        command_for(make_args(), {"provider": "codex", "model": model, "effort": effort})


def test_b083_codex_passes_through_runtime_advertised_ultra_effort():
    command = command_for(
        make_args(),
        {"provider": "codex", "model": "fixture-codex-model-b", "effort": "ultra"},
    )

    assert command[command.index("-c") + 1] == 'model_reasoning_effort="ultra"'


def test_b083_codex_preserves_runtime_advertised_effort_named_unavailable():
    command = command_for(
        make_args(),
        {
            "provider": "codex",
            "model": "fixture-codex-model-b",
            "effort": "unavailable",
        },
    )

    assert command[command.index("-c") + 1] == 'model_reasoning_effort="unavailable"'


@pytest.mark.parametrize("subcommand", ["exec", "e"])
def test_b083_codex_validates_and_preserves_exec_alias_command_override(
    monkeypatch, subcommand
):
    route = [
        "codex",
        subcommand,
        "--json",
        "-m",
        "fixture-codex-model-b",
        "-c",
        'model_reasoning_effort="high"',
    ]
    monkeypatch.setenv("LIFECYCLE_PROVIDER_COMMAND", json.dumps(route))

    command = command_for(
        make_args(),
        {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"},
    )

    assert command == route


def test_b083_codex_opaque_command_override_rejects_unbound_route(monkeypatch):
    monkeypatch.setenv("LIFECYCLE_PROVIDER_COMMAND", '["/custom/wrapper"]')

    with pytest.raises(AdapterError, match="cannot preserve the pinned Codex route"):
        command_for(make_args(), {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"})


@pytest.mark.parametrize(
    "extra_route_flags",
    [
        ["--model=fixture-codex-model-a"],
        ["--model=fixture-codex-model-b"],
        ["--model", "fixture-codex-model-a"],
        ["-m", "fixture-codex-model-a"],
        ["-mfixture-codex-model-a"],
        ["-c", 'model="fixture-codex-model-a"'],
        ['-cmodel="fixture-codex-model-a"'],
        ["--config=model_reasoning_effort=low"],
        ["--config", 'model_reasoning_effort="low"'],
        ["--config", 'model_provider="other"'],
        ["--config=model_provider=other"],
        ['-cmodel_reasoning_effort="low"'],
        ["--oss"],
    ],
)
def test_b083_codex_command_override_rejects_alternate_route_syntax(monkeypatch, extra_route_flags):
    command = [
        "codex",
        "exec",
        "--json",
        "-m",
        "fixture-codex-model-b",
        "-c",
        'model_reasoning_effort="high"',
        *extra_route_flags,
    ]
    monkeypatch.setenv("LIFECYCLE_PROVIDER_COMMAND", json.dumps(command))

    with pytest.raises(AdapterError, match="cannot preserve the pinned Codex route"):
        command_for(
            make_args(),
            {"provider": "codex", "model": "fixture-codex-model-b", "effort": "high"},
        )


def test_b083_codex_route_flag_parser_collects_repeated_override_forms():
    parser = _adapter.get("_codex_route_flags")
    assert callable(parser), "B083 route override parsing should be isolated from command construction"

    models, configs = parser(
        [
            "codex",
            "exec",
            "-m",
            "fixture-codex-model-b",
            "--model=fixture-codex-model-a",
            "-c",
            'model_provider="other"',
            "--config=model_reasoning_effort=low",
        ]
    )

    assert models == ["fixture-codex-model-b", "fixture-codex-model-a"]
    assert configs == {
        "model": [],
        "model_provider": ['model_provider="other"'],
        "model_reasoning_effort": ["model_reasoning_effort=low"],
    }


def test_b083_route_identity_reaches_provider_environment():
    request = {
        "entity_key": "TASK-1",
        "entity_type": "task",
        "provider": "codex",
        "model": "fixture-codex-model-b",
        "effort": "high",
        "routing_identity": {
            "profile_id": "codex-test",
            "profile_schema_version": "1.0",
            "profile_digest": "a" * 64,
            "profile_source_digest": "b" * 64,
            "routing_digest": "c" * 64,
        },
    }

    environment = provider_environment(request, "session-1", "d" * 64, 1)

    assert environment["LIFECYCLE_ROUTING_DIGEST"] == "c" * 64
    assert environment["LIFECYCLE_ROUTING_PROFILE_ID"] == "codex-test"
    assert environment["LIFECYCLE_ROUTING_PROFILE_VERSION"] == "1.0"
    assert environment["LIFECYCLE_ROUTING_PROFILE_DIGEST"] == "a" * 64
    assert environment["LIFECYCLE_ROUTING_PROFILE_SOURCE_DIGEST"] == "b" * 64


@pytest.mark.parametrize("invalid_value", [["unexpected"], {"unexpected": True}, 7, True])
def test_b083_route_identity_rejects_non_string_environment_fields(invalid_value):
    request = {
        "entity_key": "TASK-1",
        "entity_type": "task",
        "provider": "codex",
        "model": "fixture-codex-model-b",
        "routing_identity": {"routing_digest": invalid_value},
    }

    with pytest.raises(AdapterError, match="routing_identity.routing_digest must be a string or null"):
        provider_environment(request, "session-1", "d" * 64, 1)
