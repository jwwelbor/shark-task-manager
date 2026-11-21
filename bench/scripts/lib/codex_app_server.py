"""Bounded JSON-RPC transport for the selected Codex app-server."""

from __future__ import annotations

import json
import os
import selectors
import subprocess
import time
from typing import Any

try:
    from .codex_model_errors import CatalogError
except ImportError:
    from codex_model_errors import CatalogError


APP_SERVER_REQUEST_TIMEOUT_SECONDS = 20.0
MODEL_DISCOVERY_TIMEOUT_SECONDS = 120.0


def _pop_response_line(buffer: bytearray) -> bytes | None:
    newline = buffer.find(b"\n")
    if newline < 0:
        return None
    line = bytes(buffer[:newline])
    del buffer[: newline + 1]
    return line


def _read_response_line(output_fd: int, selector, buffer: bytearray, deadline: float) -> bytes | None:
    while True:
        line = _pop_response_line(buffer)
        if line is not None:
            return line
        remaining = deadline - time.monotonic()
        if remaining <= 0 or not selector.select(remaining):
            return None
        try:
            chunk = os.read(output_fd, 65536)
        except BlockingIOError:
            continue
        except OSError as exc:
            raise CatalogError(f"Codex app-server response read failed: {exc}") from exc
        if not chunk:
            return None
        buffer.extend(chunk)
        if len(buffer) > 16 * 1024 * 1024:
            raise CatalogError("Codex app-server response line is too large")


def matching_response(line: bytes, request_id: int) -> dict[str, Any] | None:
    if not line:
        return None
    try:
        message = json.loads(line)
    except UnicodeDecodeError as exc:
        raise CatalogError("Codex app-server returned a non-UTF-8 response") from exc
    except json.JSONDecodeError:
        return None
    if not isinstance(message, dict) or message.get("id") != request_id:
        return None
    if "error" in message:
        raise CatalogError(f"Codex app-server {message['error']}")
    result = message.get("result")
    if not isinstance(result, dict):
        raise CatalogError("Codex app-server returned a malformed response")
    return result


def read_response(
    process: subprocess.Popen[str],
    selector,
    request_id: int,
    timeout_seconds: float = APP_SERVER_REQUEST_TIMEOUT_SECONDS,
) -> dict[str, Any]:
    assert process.stdout is not None
    output_fd = process.stdout.fileno()
    os.set_blocking(output_fd, False)
    buffer = getattr(process, "_codex_output_buffer", bytearray())
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        line = _read_response_line(output_fd, selector, buffer, deadline)
        if line is None:
            break
        result = matching_response(line, request_id)
        if result is not None:
            process._codex_output_buffer = buffer
            return result
    process._codex_output_buffer = buffer
    raise CatalogError("Codex app-server did not complete model catalog discovery")


def start_app_server(argv: list[str], env: dict[str, str] | None):
    try:
        process = subprocess.Popen(
            [*argv, "app-server", "--stdio"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            encoding="utf-8",
            env=env,
            bufsize=1,
        )
    except OSError as exc:
        raise CatalogError(f"cannot start selected Codex app-server: {exc}") from exc
    assert process.stdin is not None and process.stdout is not None
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    return process, selector


def request(
    process: subprocess.Popen[str], selector, request_id: int,
    method: str, params: dict[str, Any],
    *, timeout_seconds: float = APP_SERVER_REQUEST_TIMEOUT_SECONDS,
) -> dict[str, Any]:
    assert process.stdin is not None
    process.stdin.write(json.dumps({
        "jsonrpc": "2.0", "id": request_id, "method": method, "params": params,
    }) + "\n")
    process.stdin.flush()
    return read_response(process, selector, request_id, timeout_seconds)


def send_notification(process, method: str, params: dict[str, Any]) -> None:
    assert process.stdin is not None
    process.stdin.write(json.dumps({"jsonrpc": "2.0", "method": method, "params": params}) + "\n")
    process.stdin.flush()


def stop_app_server(process, selector) -> None:
    selector.close()
    process.terminate()
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()
