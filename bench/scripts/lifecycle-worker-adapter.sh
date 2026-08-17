#!/usr/bin/env bash
# lifecycle-worker-adapter.sh --request <json> [--provider-command <path>]
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

exec python3 - "$@" <<'PYEOF'
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path


CONTROL_KINDS = {"final", "question", "needs_council", "blocked_external", "failed"}
DIGEST = re.compile(r"^[0-9a-f]{64}$")
SENSITIVE = re.compile(
    r"(?i)(credential[-_ ]?sentinel|provider[-_ ]?secret[-_ ]?sentinel|"
    r"transcript(?:[-_ ]|$)|authorization|api[-_ ]?key|access[-_ ]?token|"
    r"password|secret)"
)
SAFE_EVIDENCE_KEYS = {"type", "path", "digest", "size_bytes", "description"}


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
    parser.add_argument("--request", required=True, help="JSON file containing the complete next response")
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
    request = load_json(path, "request")
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


def command_for(args, request):
    if args.provider_command:
        command = [args.provider_command]
    else:
        raw = os.environ.get("LIFECYCLE_PROVIDER_COMMAND")
        if raw:
            try:
                command = json.loads(raw)
            except json.JSONDecodeError as exc:
                raise AdapterError(f"LIFECYCLE_PROVIDER_COMMAND is not a JSON argv array: {exc}") from exc
            if not isinstance(command, list) or not command or not all(isinstance(item, str) and item for item in command):
                raise AdapterError("LIFECYCLE_PROVIDER_COMMAND must be a non-empty JSON string array")
        elif request["provider"] in {"anthropic", "claude", "claude-code"}:
            command = ["claude", "--print", "--output-format", "json"]
        elif request["provider"] in {"openai", "codex"}:
            command = ["codex", "exec", "--json"]
        else:
            raise AdapterError(
                "no provider command configured; use --provider-command or LIFECYCLE_PROVIDER_COMMAND"
            )
    return command


def provider_environment(request, session_id, prompt_digest, prompt_bytes):
    # Metadata is deliberately split into scalar environment values. The
    # rendered prompt and complete request never enter the provider argv or
    # environment, so a provider canary cannot accidentally log them.
    environment = os.environ.copy()
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
    environment.update(metadata)
    return environment


def decode_envelope(raw):
    if not raw.strip():
        raise AdapterError("provider returned no control envelope")
    try:
        decoded = json.loads(raw)
    except json.JSONDecodeError:
        decoded = None
        for line in reversed(raw.splitlines()):
            try:
                candidate = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(candidate, dict):
                decoded = candidate
                break
        if decoded is None:
            match = re.search(r"(?m)^RECOMMENDED OUTCOME:\s*(\S+)\s*$", raw)
            if match:
                return {"kind": "final", "recommended_outcome": match.group(1), "evidence": []}
            raise AdapterError("provider output is not a JSON control envelope")

    if isinstance(decoded, dict) and isinstance(decoded.get("kind"), str):
        return decoded
    if isinstance(decoded, dict):
        for wrapper_key in ("result", "output", "message", "text"):
            wrapped = decoded.get(wrapper_key)
            if isinstance(wrapped, dict):
                return wrapped
            if isinstance(wrapped, str):
                return decode_envelope(wrapped)
    raise AdapterError("provider output does not contain a control envelope")


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


def project_result(envelope, request, session_id, prompt_digest, prompt_bytes):
    if not isinstance(envelope, dict):
        raise AdapterError("control envelope must be an object")
    kind = envelope.get("kind")
    if kind not in CONTROL_KINDS:
        raise AdapterError(f"unsupported control-envelope kind: {kind!r}")

    worker_id = safe_identity(
        envelope.get("worker_id", os.environ.get("LIFECYCLE_WORKER_ID", "")), "worker_id"
    )
    returned_session = envelope.get("session_id", session_id)
    if returned_session != session_id:
        raise AdapterError("provider session_id does not match the parent claim session")

    result = {
        "worker_id": worker_id,
        "session_id": session_id,
        "kind": kind,
    }
    if kind == "final":
        outcome = envelope.get("recommended_outcome")
        if not isinstance(outcome, str) or not outcome:
            raise AdapterError("final control envelope requires recommended_outcome")
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


def main():
    args = parse_args()
    try:
        request, prompt, prompt_digest, prompt_bytes, session_id = load_request(args.request)
        command = command_for(args, request)
        environment = provider_environment(request, session_id, prompt_digest, prompt_bytes)
        try:
            completed = subprocess.run(
                command,
                input=prompt,
                capture_output=True,
                env=environment,
                check=False,
            )
        except OSError as exc:
            raise AdapterError(f"provider could not be started: {exc}") from exc
        if completed.returncode != 0:
            fail(f"provider exited with status {completed.returncode}", 1)
        try:
            provider_output = completed.stdout.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise AdapterError(f"provider output is not UTF-8: {exc}") from exc
        result = project_result(
            decode_envelope(provider_output), request, session_id, prompt_digest, prompt_bytes
        )
        write_result(args.result_out, result)
        json.dump(result, sys.stdout, sort_keys=True, separators=(",", ":"))
        sys.stdout.write("\n")
    except AdapterError as exc:
        fail(str(exc))


main()
PYEOF
