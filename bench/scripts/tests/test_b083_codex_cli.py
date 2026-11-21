from pathlib import Path
import sys

import pytest

REPO_ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO_ROOT))
from bench.scripts.lib import codex_model_catalog
from bench.scripts.lib.codex_model_catalog import (
    CatalogError,
    _matching_response,
    app_server_command,
)
from bench.scripts.lib.e40_benchmark import (
    OperatorError,
    _codex_command,
    clear_codex_route_environment,
)


def test_b083_stale_catalog_provenance_is_cleared_before_a_new_run():
    environment = {
        "E40_CODEX_CATALOG_JSON": "stale-catalog",
        "E40_EXPECTED_ROUTING_DIGEST": "stale-route",
        "E40_EXPECTED_CODEX_CATALOG_DIGEST": "stale-digest",
        "E40_CODEX_COMMAND": '["/selected/codex"]',
    }

    clear_codex_route_environment(environment)

    assert environment == {"E40_CODEX_COMMAND": '["/selected/codex"]'}


def test_b083_empty_explicit_provider_command_fails_closed():
    with pytest.raises(OperatorError, match="non-empty argv array"):
        _codex_command({"runtime": {"provider_command": []}}, environment={})


@pytest.mark.parametrize("command", [17, "codex", ["codex", None]])
def test_b083_malformed_provider_command_is_a_route_defect(command):
    with pytest.raises(OperatorError, match="route_defect: Codex CLI selection failed"):
        _codex_command({"runtime": {"provider_command": command}}, environment={})


def test_b083_app_server_command_skips_global_option_values_named_exec():
    base = ["codex", "--profile", "exec"]
    route = [*base, "exec", "--json", "-m", "synthetic-model"]

    assert app_server_command(base) == base
    assert app_server_command(route) == base


@pytest.mark.parametrize("subcommand", ["exec", "e"])
def test_b083_app_server_command_strips_exec_command_and_alias(subcommand):
    base = ["codex", "--profile", "workbench"]

    assert app_server_command([*base, subcommand, "--json"]) == base


@pytest.mark.parametrize("command", [None, 17, "codex", [], ["codex", None]])
def test_b083_codex_command_rejects_malformed_argv(command):
    with pytest.raises(CatalogError, match="non-empty argv array"):
        app_server_command(command)


def test_b083_app_server_rejects_non_utf8_response_bytes():
    with pytest.raises(CatalogError, match="non-UTF-8"):
        _matching_response(b"\xff", request_id=1)
