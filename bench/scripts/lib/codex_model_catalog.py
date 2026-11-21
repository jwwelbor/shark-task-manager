"""Read the selected Codex CLI's advertised model catalog."""

from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import time
from typing import Any

try:
    from .codex_app_server import (
        APP_SERVER_REQUEST_TIMEOUT_SECONDS,
        MODEL_DISCOVERY_TIMEOUT_SECONDS,
        matching_response as _matching_response,
        read_response as _read_response,
        request as _request,
        send_notification as _send_notification,
        start_app_server as _start_app_server,
        stop_app_server as _stop_app_server,
    )
    from .codex_cli import CODEX_EXEC_COMMANDS, exec_subcommand_index, selected_profile
    from .codex_model_errors import CatalogError
except ImportError:
    from codex_app_server import (
        APP_SERVER_REQUEST_TIMEOUT_SECONDS,
        MODEL_DISCOVERY_TIMEOUT_SECONDS,
        matching_response as _matching_response,
        read_response as _read_response,
        request as _request,
        send_notification as _send_notification,
        start_app_server as _start_app_server,
        stop_app_server as _stop_app_server,
    )
    from codex_cli import CODEX_EXEC_COMMANDS, exec_subcommand_index, selected_profile
    from codex_model_errors import CatalogError


MAX_MODEL_LIST_PAGES = 100
MAX_MODEL_LIST_ENTRIES = 10_000


def app_server_command(command: list[str]) -> list[str]:
    """Drop a route-specific ``exec``/``e`` suffix from a Codex command."""
    if (
        not isinstance(command, list)
        or not command
        or any(not isinstance(part, str) or not part for part in command)
    ):
        raise CatalogError("Codex CLI command must be a non-empty argv array")
    if command[0] in CODEX_EXEC_COMMANDS:
        raise CatalogError("Codex command override has no executable before exec")
    try:
        exec_index = exec_subcommand_index(command)
    except ValueError as exc:
        raise CatalogError(str(exc)) from exc
    if exec_index is None:
        return list(command)
    return list(command[:exec_index])


def _resolved_command(command: list[str], env: dict[str, str] | None) -> list[str]:
    prefix = app_server_command(command)
    executable = shutil.which(prefix[0], path=(env or os.environ).get("PATH"))
    if executable is None:
        raise CatalogError("selected Codex CLI executable is unavailable")
    return [executable, *prefix[1:]]


def resolve_cli_command(command: list[str], *, env: dict[str, str] | None = None) -> list[str]:
    """Return the selected Codex executable and non-route CLI arguments."""
    return _resolved_command(command, env)


def _file_digest(path: str, *, missing_ok: bool = False) -> str | None:
    digest = hashlib.sha256()
    try:
        with open(path, "rb") as source:
            for chunk in iter(lambda: source.read(1024 * 1024), b""):
                digest.update(chunk)
    except FileNotFoundError as exc:
        if missing_ok:
            return None
        raise CatalogError(f"selected Codex executable disappeared: {path}") from exc
    except OSError as exc:
        raise CatalogError(f"cannot fingerprint Codex runtime file {path}: {exc}") from exc
    return digest.hexdigest()


def _profile_config_digest(command: list[str], codex_home: str) -> str | None:
    try:
        profile = selected_profile(command)
    except ValueError as exc:
        raise CatalogError(str(exc)) from exc
    if profile is None:
        return None
    if (
        profile in {".", ".."}
        or "/" in profile
        or "\\" in profile
        or any(ord(character) < 0x20 or ord(character) == 0x7F for character in profile)
    ):
        raise CatalogError("Codex profile name is not a safe file name")
    profile_path = os.path.join(codex_home, f"{profile}.config.toml")
    return _file_digest(profile_path, missing_ok=True)


def runtime_identity(command: list[str], executable: str, env: dict[str, str] | None = None) -> str:
    """Fingerprint non-secret settings which select the Codex configuration."""
    environment = env or os.environ
    home = environment.get("CODEX_HOME")
    if home:
        codex_home = os.path.abspath(os.path.expanduser(home))
    else:
        codex_home = os.path.join(
            os.path.abspath(os.path.expanduser(environment.get("HOME", os.path.expanduser("~")))),
            ".codex",
        )
    xdg_config = environment.get("XDG_CONFIG_HOME")
    if xdg_config:
        xdg_config = os.path.abspath(os.path.expanduser(xdg_config))
    config_digest = _file_digest(os.path.join(codex_home, "config.toml"), missing_ok=True)
    resolved_command = _resolved_command(command, env)
    payload = {
        "command": resolved_command,
        "codex_home": codex_home,
        "xdg_config_home": xdg_config,
        "config_digest": config_digest,
        "profile_config_digest": _profile_config_digest(resolved_command, codex_home),
        "executable_digest": _file_digest(executable),
    }
    return hashlib.sha256(
        json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()


def validate_catalog_snapshot(
    catalog: dict[str, Any], command: list[str], *, env: dict[str, str] | None = None
) -> dict[str, Any]:
    """Validate a frozen catalog against the exact CLI/config identity in use."""
    if not isinstance(catalog, dict) or catalog.get("source") != "codex-app-server-model-list":
        raise CatalogError("frozen Codex model catalog has an invalid source")
    resolved = _resolved_command(command, env)
    if catalog.get("cli_path") != resolved[0]:
        raise CatalogError("frozen Codex model catalog belongs to a different CLI executable")
    if catalog.get("cli_version") != _cli_version(resolved, env):
        raise CatalogError("frozen Codex model catalog belongs to a different CLI version")
    expected_identity = runtime_identity(command, resolved[0], env)
    if catalog.get("runtime_identity") != expected_identity:
        raise CatalogError("frozen Codex model catalog belongs to a different CLI configuration")
    models = catalog.get("models")
    if not isinstance(models, list) or not models:
        raise CatalogError("frozen Codex model catalog has no model entries")
    _validate_model_entries(models)
    encoded = json.dumps(models, sort_keys=True, separators=(",", ":")).encode()
    if catalog.get("catalog_digest") != hashlib.sha256(encoded).hexdigest():
        raise CatalogError("frozen Codex model catalog digest does not match its entries")
    return catalog


def _validate_snapshot_efforts(entry: dict[str, Any]) -> None:
    efforts = entry.get("supported_reasoning_efforts")
    if not isinstance(efforts, list):
        raise CatalogError("Codex model catalog contains incomplete route metadata")
    if any(not isinstance(effort, str) or not effort for effort in efforts):
        raise CatalogError("Codex model catalog contains incomplete route metadata")
    default_effort = entry.get("default_reasoning_effort")
    if not isinstance(default_effort, str) or default_effort not in efforts:
        raise CatalogError("Codex model catalog contains incomplete route metadata")


def _validate_snapshot_model(entry: Any) -> dict[str, Any]:
    if not isinstance(entry, dict):
        raise CatalogError("Codex model catalog contains a malformed model entry")
    model = entry.get("model")
    if not isinstance(model, str) or not model:
        raise CatalogError("Codex model catalog contains incomplete route metadata")
    if not isinstance(entry.get("is_default"), bool):
        raise CatalogError("Codex model catalog contains incomplete route metadata")
    _validate_snapshot_efforts(entry)
    return entry


def _validate_model_entries(entries: list[Any]) -> list[dict[str, Any]]:
    if not isinstance(entries, list) or not entries:
        raise CatalogError("Codex model catalog contains a malformed model entry")
    models = [_validate_snapshot_model(entry) for entry in entries]
    model_ids = [entry["model"] for entry in models]
    if len(model_ids) != len(set(model_ids)):
        raise CatalogError("Codex model catalog contains duplicate model identifiers")
    return models


def _select_model_entry(model: str | None, entries: list[dict[str, Any]]) -> dict[str, Any]:
    if model is None:
        defaults = [entry for entry in entries if entry.get("is_default") is True]
        if len(defaults) != 1:
            raise CatalogError("Codex model catalog must contain one unique default model")
        return defaults[0]
    if not isinstance(model, str) or not model:
        raise CatalogError("explicit Codex model pin must be a non-empty string")
    selected = {entry["model"]: entry for entry in entries}.get(model)
    if selected is None:
        raise CatalogError(f"Codex model catalog does not contain pinned model {model!r}")
    return selected


def _resolve_effort(selected: dict[str, Any], effort: str | None) -> str:
    advertised = selected.get("supported_reasoning_efforts")
    default_effort = selected.get("default_reasoning_effort")
    if not isinstance(advertised, list) or not isinstance(default_effort, str):
        raise CatalogError("Codex selected model has incomplete reasoning effort metadata")
    resolved_effort = default_effort if effort is None else effort
    if not isinstance(resolved_effort, str) or not resolved_effort:
        raise CatalogError("Codex reasoning effort must be a non-empty string")
    if resolved_effort not in advertised:
        raise CatalogError(
            f"Codex model {selected['model']!r} does not advertise effort {resolved_effort!r}"
        )
    return resolved_effort


def resolve_route(model: str | None, effort: str | None, catalog: dict[str, Any]) -> dict[str, str]:
    if not isinstance(catalog, dict):
        raise CatalogError("Codex model catalog must be an object")
    entries = catalog.get("models")
    if not isinstance(entries, list):
        raise CatalogError("Codex model catalog has no model list")
    selected = _select_model_entry(model, _validate_model_entries(entries))
    return {"model": selected["model"], "effort": _resolve_effort(selected, effort)}


def _cli_version(argv: list[str], env: dict[str, str] | None) -> str:
    try:
        version = subprocess.run(
            [*argv, "--version"], env=env, text=True, capture_output=True,
            check=False, timeout=10,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise CatalogError(f"cannot identify selected Codex CLI: {exc}") from exc
    if version.returncode != 0:
        raise CatalogError("selected Codex CLI version is unavailable")
    cli_version = (version.stdout or version.stderr).strip().splitlines()
    if not cli_version:
        raise CatalogError("selected Codex CLI returned an empty version")
    return cli_version[0]


def _visible_model_entry(entry: Any) -> bool:
    if not isinstance(entry, dict):
        raise CatalogError("Codex model/list contains a malformed model entry")
    hidden = entry.get("hidden", False)
    if not isinstance(hidden, bool):
        raise CatalogError("Codex model/list contains malformed hidden metadata")
    return not hidden


def _optional_model_text(entry: dict[str, Any], field: str) -> str | None:
    value = entry.get(field)
    if value is not None and not isinstance(value, str):
        raise CatalogError("Codex model/list contains incomplete route metadata")
    return value


def _model_entry_fields(entry: Any) -> dict[str, Any] | None:
    if not _visible_model_entry(entry):
        return None
    model = entry.get("model")
    display_name = _optional_model_text(entry, "displayName")
    description = _optional_model_text(entry, "description")
    is_default = entry.get("isDefault", False)
    efforts = entry.get("supportedReasoningEfforts", [])
    default_effort = entry.get("defaultReasoningEffort")
    if not isinstance(model, str) or not model:
        raise CatalogError("Codex model/list contains incomplete route metadata")
    if not isinstance(is_default, bool) or not isinstance(efforts, list):
        raise CatalogError("Codex model/list contains incomplete route metadata")
    if not isinstance(default_effort, str) or not default_effort:
        raise CatalogError("Codex model/list contains incomplete route metadata")
    return {
        "model": model,
        "display_name": display_name,
        "description": description,
        "is_default": is_default,
        "efforts": efforts,
        "default_effort": default_effort,
    }


def _normalized_efforts(efforts: list[Any], default_effort: str) -> list[str]:
    supported = [
        item["reasoningEffort"] for item in efforts
        if isinstance(item, dict) and isinstance(item.get("reasoningEffort"), str)
    ]
    if len(supported) != len(efforts) or any(not effort for effort in supported):
        raise CatalogError("Codex model/list contains malformed effort metadata")
    if default_effort not in supported:
        raise CatalogError("Codex model/list default effort is not in its supported efforts")
    return supported


def _normalize_model_entry(entry: Any) -> dict[str, Any] | None:
    fields = _model_entry_fields(entry)
    if fields is None:
        return None
    supported = _normalized_efforts(fields["efforts"], fields["default_effort"])
    return {
        "model": fields["model"],
        "display_name": fields["display_name"],
        "description": fields["description"],
        "is_default": fields["is_default"],
        "default_reasoning_effort": fields["default_effort"],
        "supported_reasoning_efforts": supported,
    }


def _page_models(page: dict[str, Any]) -> list[dict[str, Any]]:
    entries = page.get("data")
    if not isinstance(entries, list):
        raise CatalogError("Codex model/list returned malformed model data")
    return [model for entry in entries if (model := _normalize_model_entry(entry)) is not None]


def _next_cursor(page: dict[str, Any], seen: set[str]) -> str | None:
    cursor = page.get("nextCursor")
    if cursor is not None:
        if not isinstance(cursor, str) or not cursor or cursor in seen:
            raise CatalogError("Codex model/list returned an invalid continuation cursor")
        seen.add(cursor)
    return cursor


def _collect_model_pages(request, *, deadline: float | None = None) -> list[dict[str, Any]]:
    models: list[dict[str, Any]] = []
    cursor = None
    seen_cursors: set[str] = set()
    page_count = 0
    entry_count = 0
    while True:
        if page_count >= MAX_MODEL_LIST_PAGES:
            raise CatalogError("Codex model/list exceeded the page limit")
        request_timeout = APP_SERVER_REQUEST_TIMEOUT_SECONDS
        if deadline is not None:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise CatalogError("Codex model catalog discovery exceeded its total time limit")
            request_timeout = min(request_timeout, remaining)
        params = {} if cursor is None else {"cursor": cursor}
        page = request("model/list", params, timeout_seconds=request_timeout)
        page_count += 1
        entries = page.get("data")
        if not isinstance(entries, list):
            raise CatalogError("Codex model/list returned malformed model data")
        entry_count += len(entries)
        if entry_count > MAX_MODEL_LIST_ENTRIES:
            raise CatalogError("Codex model/list exceeded the entry limit")
        models.extend(_page_models(page))
        cursor = _next_cursor(page, seen_cursors)
        if cursor is None:
            break
    if not models:
        raise CatalogError("Codex model/list returned no visible models")
    return _validate_model_entries(models)


def _read_model_pages(process, selector) -> tuple[list[dict[str, Any]], str | None]:
    assert process.stdin is not None
    request_id = 1
    deadline = time.monotonic() + MODEL_DISCOVERY_TIMEOUT_SECONDS

    def request(
        method: str,
        params: dict[str, Any],
        *,
        timeout_seconds: float = APP_SERVER_REQUEST_TIMEOUT_SECONDS,
    ) -> dict[str, Any]:
        nonlocal request_id
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise CatalogError("Codex model catalog discovery exceeded its total time limit")
        current_id = request_id
        request_id += 1
        return _request(
            process, selector, current_id, method, params,
            timeout_seconds=min(timeout_seconds, remaining),
        )

    try:
        initialized = request("initialize", {
            "clientInfo": {"name": "e40-route-preflight", "version": "1"},
            "capabilities": {},
        })
        _send_notification(process, "initialized", {})
        models = _collect_model_pages(request, deadline=deadline)
        return models, initialized.get("userAgent")
    except (BrokenPipeError, OSError) as exc:
        raise CatalogError(f"Codex app-server protocol failed: {exc}") from exc
    finally:
        _stop_app_server(process, selector)


def discover_models(command: list[str], *, env: dict[str, str] | None = None) -> dict[str, Any]:
    """Return all visible model/list entries and non-secret CLI provenance."""
    argv = _resolved_command(command, env)
    executable = argv[0]
    cli_version = _cli_version(argv, env)
    initial_identity = runtime_identity(command, executable, env)
    process, selector = _start_app_server(argv, env)
    models, server_info = _read_model_pages(process, selector)
    final_identity = runtime_identity(command, executable, env)
    if final_identity != initial_identity:
        raise CatalogError("selected Codex runtime changed during model catalog discovery")

    encoded = json.dumps(models, sort_keys=True, separators=(",", ":")).encode()
    return {
        "source": "codex-app-server-model-list",
        "cli_path": executable,
        "cli_version": cli_version,
        "runtime_identity": initial_identity,
        "server_user_agent": server_info if isinstance(server_info, str) else None,
        "models": models,
        "catalog_digest": hashlib.sha256(encoded).hexdigest(),
    }
