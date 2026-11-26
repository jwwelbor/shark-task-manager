#!/usr/bin/env bash
# lifecycle-worker-adapter.sh [--request <json>] [--provider-command <path>]
#
# Provider-neutral boundary for the F08 controller. The controller supplies a
# complete shark next response plus its claim session. The selected provider
# executable is awaited in the foreground and receives only the exact prompt
# bytes on stdin; routing and workflow mutation stay with the controller.
#
# A provider command may be supplied as --provider-command or as a JSON argv
# array in LIFECYCLE_PROVIDER_COMMAND. It must read the prompt from stdin and
# write one control envelope (or a provider wrapper containing one) to stdout.
# Provider-specific flags belong to that command, not to this adapter.
set -euo pipefail

REQUEST_ON_STDIN=1
for argument in "$@"; do
	if [[ "$argument" == "--request" || "$argument" == --request=* ]]; then
		REQUEST_ON_STDIN=0
		break
	fi
done

if [[ "$REQUEST_ON_STDIN" -eq 1 ]]; then
	STDIN_REQUEST="$(mktemp)"
	trap 'rm -f "$STDIN_REQUEST"' EXIT
	cat >"$STDIN_REQUEST"
	set -- --request "$STDIN_REQUEST" "$@"
fi

ADAPTER_SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
E40_ADAPTER_LIB_PATH="$ADAPTER_SCRIPT_DIR/lib" python3 - "$@" <<'PYEOF'
import argparse
import codecs
import hashlib
import json
import os
import re
import signal
import subprocess
import sys
import tempfile
import threading
from pathlib import Path

adapter_lib_path = os.environ.pop("E40_ADAPTER_LIB_PATH", None)
if adapter_lib_path:
    sys.path.insert(0, adapter_lib_path)
from codex_cli import exec_subcommand_index


CONTROL_KINDS = {"final", "question", "needs_council", "blocked_external", "failed"}
DIGEST = re.compile(r"^[0-9a-f]{64}$")
SENSITIVE = re.compile(
    r"(?i)(credential[-_ ]?sentinel|provider[-_ ]?secret[-_ ]?sentinel|"
    r"transcript(?:[-_ ]|$)|authorization|api[-_ ]?key|access[-_ ]?token|"
    r"password|secret)"
)
SAFE_EVIDENCE_KEYS = {"type", "path", "digest", "size_bytes", "description"}
CONTROL_SCHEMA = {
    "type": "object",
    "properties": {
        "kind": {"type": "string", "enum": sorted(CONTROL_KINDS)},
        "recommended_outcome": {"type": "string"},
        "worker_id": {"type": "string"},
        "evidence": {"type": "array"},
        "entity_key": {"type": "string"},
        "category": {"type": "string"},
        "question": {"type": "string"},
        "why_blocking": {"type": "string"},
        "recommendation": {"type": "string"},
        "options": {"type": "array", "items": {"type": "string"}},
    },
    "required": ["kind", "evidence"],
    "additionalProperties": True,
}
class AdapterError(ValueError):
    """A malformed request or provider control envelope."""


def fail(message, code=2):
    print(f"lifecycle-worker-adapter: {message}", file=sys.stderr)
    raise SystemExit(code)


def require_string(document, field):
    value = document.get(field)
    if not isinstance(value, str) or not value:
        raise AdapterError(f"{field} must be a non-empty string")
    return value


def parse_args():
    parser = argparse.ArgumentParser(
        description="Await one provider worker and return a bounded control envelope."
    )
    parser.add_argument(
        "--request",
        help="JSON file containing the complete next response; defaults to stdin",
    )
    parser.add_argument("--provider-command", help="foreground executable that reads the prompt from stdin")
    parser.add_argument("--result-out", help="optional path for the bounded result JSON")
    return parser.parse_args()


def load_json(path, label):
    try:
        with open(path, encoding="utf-8") as stream:
            return json.load(stream)
    except FileNotFoundError:
        raise AdapterError(f"{label} not found: {path}")
    except (OSError, json.JSONDecodeError) as exc:
        raise AdapterError(f"cannot read {label} {path}: {exc}") from exc


def load_request(path):
    if path:
        request = load_json(path, "request")
    else:
        try:
            request = json.load(sys.stdin)
        except json.JSONDecodeError as exc:
            raise AdapterError(f"cannot read request from stdin: {exc}") from exc
    if not isinstance(request, dict):
        raise AdapterError("request must be a JSON object")

    for field in ("entity_key", "entity_type", "status", "action", "provider", "model"):
        require_string(request, field)
    if request["action"] != "spawn_agent":
        raise AdapterError(f"action must be spawn_agent, got {request['action']!r}")

    prompt = request.get("prompt")
    if not isinstance(prompt, str):
        raise AdapterError("prompt must be a string")
    prompt_bytes = prompt.encode("utf-8")
    prompt_digest = request.get("prompt_sha256")
    if not isinstance(prompt_digest, str) or not DIGEST.fullmatch(prompt_digest):
        raise AdapterError("prompt_sha256 must be a lowercase SHA-256 digest")
    if prompt_digest != hashlib.sha256(prompt_bytes).hexdigest():
        raise AdapterError("prompt_sha256 does not match prompt bytes")
    declared_bytes = request.get("prompt_bytes")
    if not isinstance(declared_bytes, int) or isinstance(declared_bytes, bool):
        raise AdapterError("prompt_bytes must be an integer")
    if declared_bytes != len(prompt_bytes):
        raise AdapterError("prompt_bytes does not match prompt bytes")

    session_id = request.get("session_id", request.get("claim_session_id"))
    if not isinstance(session_id, str) or not session_id:
        raise AdapterError("session_id (the parent claim session) is required")
    return request, prompt_bytes, prompt_digest, declared_bytes, session_id


def _parse_provider_command(raw):
    try:
        command = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise AdapterError(f"LIFECYCLE_PROVIDER_COMMAND is not a JSON argv array: {exc}") from exc
    if not isinstance(command, list) or not command or not all(
        isinstance(item, str) and item for item in command
    ):
        raise AdapterError("LIFECYCLE_PROVIDER_COMMAND must be a non-empty JSON string array")
    return command


def _option_values(command, long_name, short_name):
    values = []
    index = 0
    while index < len(command):
        item = command[index]
        if item in {short_name, long_name}:
            values.append(command[index + 1] if index + 1 < len(command) else "")
            index += 2
            continue
        if item.startswith(f"{long_name}="):
            values.append(item.split("=", 1)[1])
            index += 1
        elif item.startswith(short_name) and item != short_name:
            values.append(item[len(short_name):])
            index += 1
        else:
            index += 1
    return values


def _codex_route_flags(command):
    model_values = _option_values(command, "--model", "-m")
    config_values = {"model": [], "model_provider": [], "model_reasoning_effort": []}
    for config in _option_values(command, "--config", "-c"):
        key, separator, _ = config.partition("=")
        if separator and key.strip() in config_values:
            config_values[key.strip()].append(config)
    return model_values, config_values


def _validate_codex_command_route(command, model, normalized_effort):
    model_values, config_values = _codex_route_flags(command)
    expected_effort = (
        f'model_reasoning_effort="{normalized_effort}"'
        if normalized_effort
        else None
    )
    effort_values = config_values["model_reasoning_effort"]
    if (
        model_values != [model]
        or config_values["model"]
        or config_values["model_provider"]
        or effort_values != ([expected_effort] if expected_effort else [])
        or "--oss" in command
    ):
        raise AdapterError("configured command cannot preserve the pinned Codex route")


def _codex_route(request):
    model = request.get("model")
    if not isinstance(model, str) or not model.strip():
        raise AdapterError("codex provider requires a non-empty model")
    effort = request.get("effort")
    if effort is None:
        normalized_effort = None
    elif not isinstance(effort, str) or not effort.strip():
        raise AdapterError("codex provider requires a non-empty reasoning effort when supplied")
    else:
        normalized_effort = effort.strip()
    return model, normalized_effort


def _select_codex_command(args):
    selected_cli_raw = os.environ.get("E40_CODEX_COMMAND")
    selected_cli = _parse_provider_command(selected_cli_raw) if selected_cli_raw else None
    if args.provider_command:
        command = [args.provider_command]
        if selected_cli is not None and command != selected_cli:
            raise AdapterError("configured command cannot preserve the selected Codex CLI identity")
    elif selected_cli is not None:
        command = list(selected_cli)
    else:
        raw = os.environ.get("LIFECYCLE_PROVIDER_COMMAND")
        command = _parse_provider_command(raw) if raw else ["codex"]
    return command, selected_cli


def _validate_codex_command_identity(command, selected_cli, exec_index=None):
    prefix = command if exec_index is None else command[:exec_index]
    if selected_cli is not None and prefix != selected_cli:
        raise AdapterError("configured command cannot preserve the selected Codex CLI identity")
    if selected_cli is None and Path(command[0]).name != "codex":
        raise AdapterError("configured command cannot preserve the pinned Codex route")


def _codex_exec_subcommand_index(command):
    try:
        return exec_subcommand_index(command)
    except ValueError as exc:
        raise AdapterError(str(exc)) from exc


def _pin_codex_command(command, selected_cli, model, normalized_effort):
    if command and command[0] in {"exec", "e"}:
        raise AdapterError("configured Codex command has no executable before exec")
    exec_index = _codex_exec_subcommand_index(command)
    if exec_index is not None:
        _validate_codex_command_identity(command, selected_cli, exec_index)
        _validate_codex_command_route(command, model, normalized_effort)
        return command
    _validate_codex_command_identity(command, selected_cli)
    command.extend(["exec", "--json", "-m", model])
    if normalized_effort:
        command.extend(["-c", f'model_reasoning_effort="{normalized_effort}"'])
    _validate_codex_command_route(command, model, normalized_effort)
    return command


def _codex_command_for(args, request):
    model, normalized_effort = _codex_route(request)
    command, selected_cli = _select_codex_command(args)
    return _pin_codex_command(command, selected_cli, model, normalized_effort)


def _claude_command_for(request):
    control_schema = json.loads(json.dumps(CONTROL_SCHEMA))
    outcomes = request.get("allowed_outcomes")
    if isinstance(outcomes, list) and outcomes and all(
        isinstance(item, str) and item for item in outcomes
    ):
        control_schema["properties"]["recommended_outcome"]["enum"] = sorted(set(outcomes))
    command = [
        "claude",
        "--print",
        "--output-format",
        "json",
        "--model",
        request["model"],
        "--json-schema",
        json.dumps(control_schema, sort_keys=True, separators=(",", ":")),
    ]
    effort = request.get("effort")
    if isinstance(effort, str) and effort and effort != "unavailable":
        command.extend(["--effort", effort])
    return command


def _copilot_command_for(request):
    model = request.get("model")
    if not isinstance(model, str) or not model:
        raise AdapterError("copilot provider requires a non-empty model")
    command = [
        "copilot",
        "--output-format",
        "json",
        "--stream",
        "off",
        "--model",
        model,
        "--disallow-temp-dir",
        "--no-custom-instructions",
        "--no-ask-user",
        "--allow-all-tools",
        "--disable-builtin-mcps",
        "--no-color",
    ]
    scratch_root = os.environ.get("LIFECYCLE_SCRATCH_ROOT") or request.get("scratch_root")
    if scratch_root:
        command.extend(["-C", str(scratch_root)])
    effort = request.get("effort")
    supported_efforts = {"none", "minimal", "low", "medium", "high", "xhigh", "max"}
    if isinstance(effort, str) and effort and effort != "unavailable":
        normalized_effort = effort.lower()
        if normalized_effort in supported_efforts:
            command.extend(["--reasoning-effort", normalized_effort])
    return command


def _non_codex_command_for(args, request):
    if args.provider_command:
        return [args.provider_command]
    raw = os.environ.get("LIFECYCLE_PROVIDER_COMMAND")
    if raw:
        return _parse_provider_command(raw)
    if request["provider"] in {"anthropic", "claude", "claude-code"}:
        return _claude_command_for(request)
    if request["provider"] in {"copilot", "github-copilot", "github_copilot", "github"}:
        return _copilot_command_for(request)
    raise AdapterError(
        "no provider command configured; use --provider-command or LIFECYCLE_PROVIDER_COMMAND"
    )


def command_for(args, request):
    if request["provider"] in {"openai", "codex"}:
        return _codex_command_for(args, request)
    return _non_codex_command_for(args, request)


def provider_environment(request, session_id, prompt_digest, prompt_bytes):
    # Metadata is deliberately split into scalar environment values. The
    # rendered prompt and complete request never enter the provider argv or
    # environment, so a provider canary cannot accidentally log them.
    environment = os.environ.copy()
    environment.pop("E40_CODEX_COMMAND", None)
    environment.pop("E40_CODEX_CATALOG_JSON", None)
    environment.pop("E40_EXPECTED_ROUTING_DIGEST", None)
    environment.pop("E40_EXPECTED_CODEX_CATALOG_DIGEST", None)
    environment.pop("E40_ROUTE_MATERIALIZATION_ERROR", None)
    metadata = {
        "LIFECYCLE_ENTITY_KEY": request["entity_key"],
        "LIFECYCLE_ENTITY_TYPE": request["entity_type"],
        "LIFECYCLE_SESSION_ID": session_id,
        "LIFECYCLE_PROVIDER": request["provider"],
        "LIFECYCLE_MODEL": request["model"],
        "LIFECYCLE_AGENT_TYPE": request.get("agent_type", ""),
        "LIFECYCLE_EFFORT": request.get("effort", ""),
        "LIFECYCLE_PROMPT_SHA256": prompt_digest,
        "LIFECYCLE_PROMPT_BYTES": str(prompt_bytes),
    }
    routing_identity = request.get("routing_identity")
    if routing_identity is not None and not isinstance(routing_identity, dict):
        raise AdapterError("routing_identity must be an object or null")
    if isinstance(routing_identity, dict):
        for source, target in (
            ("routing_digest", "LIFECYCLE_ROUTING_DIGEST"),
            ("profile_id", "LIFECYCLE_ROUTING_PROFILE_ID"),
            ("profile_schema_version", "LIFECYCLE_ROUTING_PROFILE_VERSION"),
            ("profile_digest", "LIFECYCLE_ROUTING_PROFILE_DIGEST"),
            ("profile_source_digest", "LIFECYCLE_ROUTING_PROFILE_SOURCE_DIGEST"),
        ):
            value = routing_identity.get(source)
            if value is not None and not isinstance(value, str):
                raise AdapterError(
                    f"routing_identity.{source} must be a string or null"
                )
            metadata[target] = value or ""
    environment.update(metadata)
    return environment


MAX_RAW_FINAL_BYTES = 256 * 1024
MAX_STREAM_LINE_BYTES = 4 * 1024 * 1024
MAX_DIRECT_BUFFER_CHARS = 4 * 1024 * 1024
OUTCOME_LINE_RE = re.compile(r"^RECOMMENDED OUTCOME:\s*(\S+)\s*$")
CODEX_EVENT_TYPES = {
    "thread.started",
    "turn.started",
    "turn.completed",
    "turn.failed",
    "item.started",
    "item.updated",
    "item.completed",
    "error",
}
WRAPPER_KEYS = ("structured_output", "result", "output", "message", "text")


def _extract_object_member_span(raw_json_text, key):
    """Return the exact raw JSON substring for `key` in a top-level JSON object."""
    decoder = json.JSONDecoder()
    idx = 0
    n = len(raw_json_text)
    while idx < n and raw_json_text[idx].isspace():
        idx += 1
    if idx >= n or raw_json_text[idx] != "{":
        return None
    idx += 1
    while idx < n:
        while idx < n and raw_json_text[idx].isspace():
            idx += 1
        if idx >= n or raw_json_text[idx] == "}":
            return None
        if raw_json_text[idx] == ",":
            idx += 1
            continue
        member_key, next_idx = decoder.raw_decode(raw_json_text, idx)
        idx = next_idx
        while idx < n and raw_json_text[idx].isspace():
            idx += 1
        if idx >= n or raw_json_text[idx] != ":":
            return None
        idx += 1
        while idx < n and raw_json_text[idx].isspace():
            idx += 1
        val_start = idx
        _, val_end = decoder.raw_decode(raw_json_text, val_start)
        if member_key == key:
            return raw_json_text[val_start:val_end].strip()
        idx = val_end
    return None


def _decode_direct_or_wrapper(raw_text):
    """Decode a single JSON object (direct envelope or wrapper), returning (envelope, raw_final)."""
    stripped = raw_text.strip()
    decoded = json.loads(stripped)
    if not isinstance(decoded, dict):
        return None, None
    if isinstance(decoded.get("kind"), str):
        return decoded, stripped
    matches = []
    for wrapper_key in WRAPPER_KEYS:
        if wrapper_key not in decoded:
            continue
        wrapped = decoded[wrapper_key]
        if isinstance(wrapped, dict) and isinstance(wrapped.get("kind"), str):
            span = _extract_object_member_span(stripped, wrapper_key)
            matches.append((wrapped, span))
        elif isinstance(wrapped, str) and wrapped.strip():
            inner_env, inner_raw = _decode_content_control(wrapped)
            if inner_env is not None:
                matches.append((inner_env, inner_raw))
    if len(matches) > 1:
        raise AdapterError("provider wrapper contains an ambiguous choice of terminal payload")
    if len(matches) == 1:
        return matches[0]
    return None, None


def _decode_content_control(text):
    """Decode control from message/wrapper string, returning (envelope, raw_final_or_none)."""
    stripped = text.strip()
    if not stripped:
        return None, None
    try:
        env, raw_final = _decode_direct_or_wrapper(stripped)
        if env is not None:
            return env, raw_final
    except json.JSONDecodeError:
        pass
    for line in stripped.splitlines():
        m = OUTCOME_LINE_RE.match(line.strip())
        if m:
            return {"kind": "final", "recommended_outcome": m.group(1), "evidence": []}, None
    return None, None


class _CopilotMeasurementAccumulator:
    def __init__(self):
        self.saw_events = False
        self.session_id = None
        self.duration_api_ms = None
        self.session_duration_ms = None
        self.turn_count = 0
        self.model_set = set()
        self.input_tokens = None
        self.output_tokens = None
        self.cache_read = None
        self.cache_write = None
        self.total_nano_aiu = None
        self.premium_requests = None

    def feed_event(self, event):
        event_type = event.get("type")
        if not isinstance(event_type, str):
            return
        self.saw_events = True
        data = event.get("data")
        if not isinstance(data, dict):
            return

        if event_type == "assistant.message":
            self.turn_count += 1
            model = data.get("model")
            if isinstance(model, str) and model:
                self.model_set.add(model)
        elif event_type == "model.call_start":
            model = data.get("model")
            if isinstance(model, str) and model:
                self.model_set.add(model)
        elif event_type == "session.usage_checkpoint":
            last_model = data.get("lastActiveModel")
            if isinstance(last_model, str) and last_model:
                self.model_set.add(last_model)
            # Replace cumulative snapshot on every checkpoint (including absence).
            self.input_tokens = None
            self.output_tokens = None
            self.cache_read = None
            self.cache_write = None
            self.total_nano_aiu = None
            self.premium_requests = None

            nano = data.get("totalNanoAiu")
            if isinstance(nano, (int, float)) and not isinstance(nano, bool) and nano >= 0:
                self.total_nano_aiu = int(nano)
            prem = data.get("totalPremiumRequests")
            if isinstance(prem, (int, float)) and not isinstance(prem, bool) and prem >= 0:
                self.premium_requests = int(prem)

            cache_state = data.get("promptCacheBreakState")
            models_dict = None
            if isinstance(cache_state, dict):
                if isinstance(cache_state.get("models"), dict):
                    models_dict = cache_state["models"]
                elif isinstance(cache_state.get("main"), dict) and isinstance(
                    cache_state["main"].get("models"), dict
                ):
                    models_dict = cache_state["main"]["models"]
            elif isinstance(cache_state, list):
                for item in cache_state:
                    if isinstance(item, dict) and isinstance(item.get("models"), dict):
                        models_dict = item["models"]
                        break

            if isinstance(models_dict, dict):
                for model_name, m_usage in models_dict.items():
                    if isinstance(model_name, str) and model_name:
                        self.model_set.add(model_name)
                    if not isinstance(m_usage, dict):
                        continue
                    for k in ("prompt_tokens", "input_tokens"):
                        if k in m_usage and isinstance(m_usage[k], int) and not isinstance(m_usage[k], bool) and m_usage[k] >= 0:
                            self.input_tokens = (self.input_tokens or 0) + m_usage[k]
                            break
                    for k in ("output_tokens", "completion_tokens"):
                        if k in m_usage and isinstance(m_usage[k], int) and not isinstance(m_usage[k], bool) and m_usage[k] >= 0:
                            self.output_tokens = (self.output_tokens or 0) + m_usage[k]
                            break
                    for k in ("cache_read", "cache_read_input_tokens"):
                        if k in m_usage and isinstance(m_usage[k], int) and not isinstance(m_usage[k], bool) and m_usage[k] >= 0:
                            self.cache_read = (self.cache_read or 0) + m_usage[k]
                            break
                    for k in ("cache_write", "cache_creation_input_tokens"):
                        if k in m_usage and isinstance(m_usage[k], int) and not isinstance(m_usage[k], bool) and m_usage[k] >= 0:
                            self.cache_write = (self.cache_write or 0) + m_usage[k]
                            break
        elif event_type == "result":
            sid = event.get("sessionId") or data.get("sessionId")
            if isinstance(sid, str) and sid:
                self.session_id = sid
            usage_data = data.get("usage")
            if isinstance(usage_data, dict):
                api_ms = usage_data.get("totalApiDurationMs")
                if isinstance(api_ms, (int, float)) and not isinstance(api_ms, bool) and api_ms >= 0:
                    self.duration_api_ms = int(api_ms)
                sess_ms = usage_data.get("sessionDurationMs")
                if isinstance(sess_ms, (int, float)) and not isinstance(sess_ms, bool) and sess_ms >= 0:
                    self.session_duration_ms = int(sess_ms)

    def finalize(self):
        if not self.saw_events:
            return {}
        model_ids = sorted(self.model_set)
        bounded_usage = {}
        provider_envelope = {}
        usage_dict = {}

        if self.input_tokens is not None:
            bounded_usage["input_tokens"] = self.input_tokens
            usage_dict["input_tokens"] = self.input_tokens
        if self.output_tokens is not None:
            bounded_usage["output_tokens"] = self.output_tokens
            usage_dict["output_tokens"] = self.output_tokens
        if self.cache_read is not None:
            bounded_usage["cache_read_input_tokens"] = self.cache_read
            usage_dict["cache_read_input_tokens"] = self.cache_read
        if self.cache_write is not None:
            bounded_usage["cache_creation_input_tokens"] = self.cache_write
            usage_dict["cache_creation_input_tokens"] = self.cache_write
        if usage_dict:
            provider_envelope["usage"] = usage_dict

        if model_ids:
            bounded_usage["model_ids"] = model_ids
            provider_envelope["modelUsage"] = {m: {} for m in model_ids}

        if self.duration_api_ms is not None:
            bounded_usage["api_active_duration_ms"] = self.duration_api_ms
            provider_envelope["duration_api_ms"] = self.duration_api_ms
        if self.session_duration_ms is not None:
            provider_envelope["sessionDurationMs"] = self.session_duration_ms
        if self.turn_count > 0:
            bounded_usage["turn_count"] = self.turn_count
            provider_envelope["num_turns"] = self.turn_count
        if self.session_id:
            bounded_usage["provider_session_id"] = self.session_id
            provider_envelope["session_id"] = self.session_id
        if self.total_nano_aiu is not None:
            provider_envelope["total_nano_aiu"] = self.total_nano_aiu
        if self.premium_requests is not None:
            provider_envelope["premium_requests"] = self.premium_requests

        measurements = {}
        if bounded_usage:
            measurements["usage"] = bounded_usage
        if provider_envelope:
            measurements["provider_usage_envelope"] = provider_envelope
        return measurements


class _StreamProcessor:
    def __init__(self):
        self.non_empty = False
        self.direct_buf = []
        self.direct_chars = 0
        self.direct_overflow = False
        self.saw_codex_events = False
        self.codex_turn_completed = False
        self.codex_turn_failed = False
        self.codex_last_agent_message = None
        self.latest_assistant_control = None
        self.first_fallback_outcome = None
        self.copilot_acc = _CopilotMeasurementAccumulator()
        self.saw_jsonl_events = False

    def feed_line(self, line):
        line_str = line.strip()
        if not line_str:
            return
        self.non_empty = True
        if not self.direct_overflow:
            if self.direct_chars + len(line) <= MAX_DIRECT_BUFFER_CHARS:
                self.direct_buf.append(line)
                self.direct_chars += len(line)
            else:
                self.direct_buf.clear()
                self.direct_overflow = True

        if self.first_fallback_outcome is None:
            m = OUTCOME_LINE_RE.match(line_str)
            if m:
                self.first_fallback_outcome = m.group(1)

        try:
            event = json.loads(line_str)
        except json.JSONDecodeError:
            return
        if not isinstance(event, dict):
            return

        event_type = event.get("type")
        if not isinstance(event_type, str):
            return
        self.saw_jsonl_events = True

        if event_type in CODEX_EVENT_TYPES or event_type.startswith(("thread.", "turn.", "item.")):
            self.saw_codex_events = True
            if event_type == "turn.started":
                self.codex_turn_completed = False
                self.codex_turn_failed = False
                self.codex_last_agent_message = None
            elif event_type == "item.completed":
                item = event.get("item")
                if (
                    isinstance(item, dict)
                    and item.get("type") == "agent_message"
                    and isinstance(item.get("text"), str)
                ):
                    self.codex_last_agent_message = item["text"]
            elif event_type == "turn.completed":
                self.codex_turn_completed = True
            elif event_type in {"turn.failed", "error", "turn.cancelled"}:
                self.codex_turn_failed = True
            return

        self.copilot_acc.feed_event(event)
        if event_type == "assistant.message":
            data = event.get("data")
            if isinstance(data, dict):
                content = data.get("content")
                if isinstance(content, str) and content.strip():
                    env, raw_final = _decode_content_control(content)
                    if env is not None:
                        self.latest_assistant_control = (env, raw_final)

    def finalize_control(self):
        if not self.non_empty:
            raise AdapterError("provider returned no control envelope")

        if self.saw_codex_events:
            if (
                not self.codex_turn_completed
                or self.codex_turn_failed
                or not isinstance(self.codex_last_agent_message, str)
                or not self.codex_last_agent_message.strip()
            ):
                raise AdapterError("Codex stream did not complete with an authoritative agent_message")
            env, raw_final = _decode_content_control(self.codex_last_agent_message)
            if env is None:
                raise AdapterError("Codex terminal agent_message does not contain a valid control envelope")
            return env, raw_final

        if not self.direct_overflow and self.direct_buf:
            raw_doc = "".join(self.direct_buf)
            try:
                env, raw_final = _decode_direct_or_wrapper(raw_doc)
                if env is not None:
                    return env, raw_final
            except json.JSONDecodeError:
                pass

        if self.latest_assistant_control is not None:
            return self.latest_assistant_control
        if self.first_fallback_outcome is not None:
            return {"kind": "final", "recommended_outcome": self.first_fallback_outcome, "evidence": []}, None
        raise AdapterError("provider output does not contain a control envelope")

    def finalize_measurements(self):
        if not self.direct_overflow and self.direct_buf:
            raw_doc = "".join(self.direct_buf)
            try:
                document = json.loads(raw_doc)
                if isinstance(document, dict):
                    return _direct_document_measurements(document)
            except json.JSONDecodeError:
                pass
        return self.copilot_acc.finalize()


def decode_envelope(raw):
    proc = _StreamProcessor()
    for line in raw.splitlines(keepends=True):
        proc.feed_line(line)
    envelope, _raw_final = proc.finalize_control()
    return envelope


def _direct_document_measurements(document):
    if not isinstance(document, dict):
        return {}

    # Keep a bounded copy in the provider's documented envelope vocabulary.
    # The lifecycle runner resolves usage-mapping.yaml against those paths;
    # `usage` below is its separate, normalized worker-facing projection.
    # Do not retain raw provider output here: only the slots the mapping can
    # consume are carried across this trust boundary.
    provider_envelope = {}
    usage = document.get("usage")
    bounded_usage = {}
    if isinstance(usage, dict):
        provider_usage = {}
        for key in (
            "input_tokens",
            "output_tokens",
            "cache_read_input_tokens",
            "cache_creation_input_tokens",
        ):
            value = usage.get(key)
            if isinstance(value, int) and not isinstance(value, bool) and value >= 0:
                bounded_usage[key] = value
                provider_usage[key] = value
        if provider_usage:
            provider_envelope["usage"] = provider_usage
    model_usage = document.get("modelUsage")
    if isinstance(model_usage, dict):
        model_ids = sorted(
            key for key in model_usage if isinstance(key, str) and key
        )
        bounded_usage["model_ids"] = model_ids
        if model_ids:
            provider_envelope["modelUsage"] = {model_id: {} for model_id in model_ids}
    for source, target in (
        ("duration_api_ms", "api_active_duration_ms"),
        ("num_turns", "turn_count"),
    ):
        value = document.get(source)
        if isinstance(value, int) and not isinstance(value, bool) and value >= 0:
            bounded_usage[target] = value
            provider_envelope[source] = value
    provider_session = document.get("session_id")
    if isinstance(provider_session, str) and provider_session:
        # Provider metadata is evidence only. The parent claim session remains
        # the sole authority used for heartbeat, transition, and release.
        bounded_usage["provider_session_id"] = provider_session
        provider_envelope["session_id"] = provider_session

    measurements = {}
    cost = document.get("total_cost_usd")
    if isinstance(cost, (int, float)) and not isinstance(cost, bool) and cost >= 0:
        measurements["cost_usd"] = float(cost)
        bounded_usage["cost_usd"] = float(cost)
        provider_envelope["total_cost_usd"] = float(cost)
    if bounded_usage:
        measurements["usage"] = bounded_usage
    if provider_envelope:
        measurements["provider_usage_envelope"] = provider_envelope
    return measurements


def provider_measurements(raw):
    proc = _StreamProcessor()
    for line in raw.splitlines(keepends=True):
        proc.feed_line(line)
    return proc.finalize_measurements()


def redact_text(value, prompt):
    text = value.replace("\r", " ").replace("\n", " ")
    if prompt and prompt in text:
        return "[REDACTED]"
    if SENSITIVE.search(text):
        return SENSITIVE.sub("[REDACTED]", text)
    return text[:512]


def bounded_evidence(value, prompt):
    if value is None:
        return []
    if not isinstance(value, list):
        raise AdapterError("evidence must be an array")
    if len(value) > 32:
        raise AdapterError("evidence exceeds the 32-entry bound")
    bounded = []
    for item in value:
        if isinstance(item, str):
            bounded.append(redact_text(item, prompt))
            continue
        if not isinstance(item, dict):
            raise AdapterError("evidence entries must be strings or objects")
        projected = {}
        for key in SAFE_EVIDENCE_KEYS:
            if key not in item:
                continue
            raw = item[key]
            if key == "size_bytes":
                if not isinstance(raw, int) or isinstance(raw, bool) or raw < 0:
                    raise AdapterError("evidence size_bytes must be a non-negative integer")
                projected[key] = raw
            elif isinstance(raw, str):
                projected[key] = redact_text(raw, prompt)
            else:
                raise AdapterError(f"evidence {key} must be a string")
        bounded.append(projected)
    return bounded


def safe_identity(value, field):
    if not isinstance(value, str) or not value or len(value) > 256 or any(char.isspace() for char in value):
        raise AdapterError(f"{field} must be a bounded non-empty token")
    if SENSITIVE.search(value):
        raise AdapterError(f"{field} contains sensitive content")
    return value


def _resolve_worker_id(envelope, request):
    if "worker_id" in envelope and envelope["worker_id"] not in (None, ""):
        return safe_identity(envelope["worker_id"], "worker_id")
    host_id = os.environ.get("LIFECYCLE_WORKER_ID")
    if host_id:
        return safe_identity(host_id, "worker_id")
    provider = request.get("provider", "")
    model = request.get("model", "")
    raw_token = f"{provider}-{model}"
    if not raw_token or any(char.isspace() for char in raw_token):
        raise AdapterError("worker_id must be a bounded non-empty token")
    if SENSITIVE.search(raw_token):
        raise AdapterError("worker_id contains sensitive content")
    if len(raw_token) <= 256:
        return raw_token
    digest_payload = json.dumps(
        {"model": model, "provider": provider},
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    generated = f"e40.worker.v1.{hashlib.sha256(digest_payload).hexdigest()}"
    return safe_identity(generated, "worker_id")


def project_result(envelope, request, session_id, prompt_digest, prompt_bytes, raw_final=None):
    if not isinstance(envelope, dict):
        raise AdapterError("control envelope must be an object")
    kind = envelope.get("kind")
    if kind not in CONTROL_KINDS:
        raise AdapterError(f"unsupported control-envelope kind: {kind!r}")

    worker_id = _resolve_worker_id(envelope, request)
    # Session identity is parent-owned authority. Provider prose or structured
    # output may echo or hallucinate a session token, but it can neither set
    # nor invalidate the claim session attached by this adapter.
    result = {
        "worker_id": worker_id,
        "session_id": session_id,
        "kind": kind,
    }
    structured = request.get("result_contract") == "gate_result_v1"
    if kind == "final":
        outcome = envelope.get("recommended_outcome")
        if not isinstance(outcome, str) or not outcome:
            raise AdapterError("final control envelope requires recommended_outcome")
        allowed = request.get("allowed_outcomes")
        if structured:
            if raw_final is None:
                raise AdapterError("structured step requires a JSON control envelope")
            if len(raw_final.encode("utf-8")) > MAX_RAW_FINAL_BYTES:
                raise AdapterError("raw_final exceeds the 256 KiB byte limit")
            if isinstance(allowed, list) and allowed and outcome not in allowed:
                raise AdapterError(
                    f"recommended_outcome {outcome!r} is not valid for this structured workflow step; allowed: {sorted(allowed)!r}"
                )
            result["raw_final"] = raw_final
        else:
            if not (isinstance(allowed, list) and outcome in allowed):
                aliased = {
                    "advance": "pass", "complete": "pass", "completed": "pass", "success": "pass",
                    "reject": "fail", "rejected": "fail",
                }.get(outcome.lower(), outcome)
                if not isinstance(allowed, list) or not allowed or aliased in allowed:
                    outcome = aliased
            if isinstance(allowed, list) and allowed and outcome not in allowed:
                raise AdapterError(
                    f"recommended_outcome {outcome!r} is not valid for this workflow step; allowed: {sorted(allowed)!r}"
                )
        result["recommended_outcome"] = redact_text(outcome, request["prompt"])
    result["evidence"] = bounded_evidence(envelope.get("evidence"), request["prompt"])

    # Question fields are structured handoff data, not workflow authority. A
    # parent decides how to route them; unknown provider fields are discarded.
    if kind == "question":
        for field in ("entity_key", "category", "question", "why_blocking", "recommendation"):
            if field in envelope and isinstance(envelope[field], str):
                result[field] = redact_text(envelope[field], request["prompt"])
        if "options" in envelope:
            options = envelope["options"]
            if not isinstance(options, list) or len(options) > 16 or not all(isinstance(item, str) for item in options):
                raise AdapterError("question options must be a bounded string array")
            result["options"] = [redact_text(item, request["prompt"]) for item in options]

    result["prompt_sha256"] = prompt_digest
    result["prompt_bytes"] = prompt_bytes
    return result


def write_result(path, result):
    if not path:
        return
    destination = Path(path)
    try:
        destination.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.NamedTemporaryFile(
            mode="w", encoding="utf-8", dir=destination.parent, delete=False
        ) as stream:
            json.dump(result, stream, sort_keys=True, separators=(",", ":"))
            stream.write("\n")
            temporary = Path(stream.name)
        os.replace(temporary, destination)
    except OSError as exc:
        raise AdapterError(f"cannot write result {path}: {exc}") from exc


def _stream_provider_process(command, prompt, environment):
    try:
        proc = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            env=environment,
            start_new_session=True,
        )
    except OSError as exc:
        raise AdapterError(f"provider could not be started: {exc}") from exc

    stdin_error = []

    def _write_stdin():
        try:
            if proc.stdin is not None:
                data = prompt if isinstance(prompt, bytes) else prompt.encode("utf-8")
                proc.stdin.write(data)
        except OSError as exc:
            stdin_error.append(exc)
        finally:
            if proc.stdin is not None:
                try:
                    proc.stdin.close()
                except OSError:
                    pass

    writer = threading.Thread(target=_write_stdin, daemon=True)
    writer.start()

    stream_proc = _StreamProcessor()
    utf8_decoder = codecs.getincrementaldecoder("utf-8")("strict")
    line_buf = []
    line_bytes = 0
    try:
        assert proc.stdout is not None
        while True:
            chunk = proc.stdout.read(65536)
            if not chunk:
                break
            try:
                text_chunk = utf8_decoder.decode(chunk, final=False)
            except UnicodeDecodeError as exc:
                raise AdapterError(f"provider output is not UTF-8: {exc}") from exc
            for ch in text_chunk:
                line_buf.append(ch)
                line_bytes += len(ch.encode("utf-8"))
                if line_bytes > MAX_STREAM_LINE_BYTES:
                    raise AdapterError("provider output line exceeds maximum stream bound")
                if ch == "\n":
                    stream_proc.feed_line("".join(line_buf))
                    line_buf.clear()
                    line_bytes = 0
        try:
            tail = utf8_decoder.decode(b"", final=True)
        except UnicodeDecodeError as exc:
            raise AdapterError(f"provider output is not UTF-8: {exc}") from exc
        if tail:
            line_buf.append(tail)
        if line_buf:
            stream_proc.feed_line("".join(line_buf))
            line_buf.clear()
        writer.join(timeout=2)
        returncode = proc.wait()
        if returncode != 0:
            fail(f"provider exited with status {returncode}", 1)
    finally:
        if proc.poll() is None:
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            proc.wait(timeout=2)
    return stream_proc


def main():
    args = parse_args()
    try:
        request, prompt, prompt_digest, prompt_bytes, session_id = load_request(args.request)
        command = command_for(args, request)
        environment = provider_environment(request, session_id, prompt_digest, prompt_bytes)
        stream_proc = _stream_provider_process(command, prompt, environment)
        envelope, raw_final = stream_proc.finalize_control()
        result = project_result(
            envelope, request, session_id, prompt_digest, prompt_bytes, raw_final=raw_final
        )
        result.update(stream_proc.finalize_measurements())
        write_result(args.result_out, result)
        json.dump(result, sys.stdout, sort_keys=True, separators=(",", ":"))
        sys.stdout.write("\n")
    except AdapterError as exc:
        fail(str(exc))


main()
PYEOF
