#!/usr/bin/env bash
# canary-copilot-usagemapping.sh [--transcript <path>]
#
# Real capture canary verification for Copilot usage mapping (T-E40-F13-005,
# AC-F13-08, TC-F13-08).
#
# Verifies that captured Copilot CLI JSONL transcripts resolve all declared
# github_copilot_cli slots in bench/evidence/usage-mapping.yaml without drift.
#
# Default (no --transcript): checks committed envelope fixtures under
# bench/scripts/testdata/usagemapping/copilot/ (or defaults to the clean capture).
# --transcript <path> lets an operator or test point the canary at any other
# transcript path.
#
# Detects:
#   (a) envelope-field drift -- mapped envelope path missing from extracted measurements.
#       Fails: "FAIL: usage_slot_unavailable slot=<slot> envelope_path=<path>"
#   (b) envelope-source unavailable -- missing or unparseable STDOUT block.
#       Fails: "FAIL: envelope_source_unavailable transcript=<path>"
#
# Output contract:
# All diagnostics go to STDERR.
# Exits 0 with last line "PASS" on stderr on success.
# Exits 1 with last line "FAIL: <detail>" on failure.
# Exits 2 on invalid usage or missing files.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

MAPPING_FILE="$BENCH_DIR/evidence/usage-mapping.yaml"
ADAPTER_SCRIPT="$SCRIPT_DIR/lifecycle-worker-adapter.sh"
DEFAULT_FIXTURE_DIR="$SCRIPT_DIR/testdata/usagemapping/copilot"

usage() {
	echo "usage: canary-copilot-usagemapping.sh [--transcript <path>]" >&2
	exit 2
}

transcript_override=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--transcript)
		[[ $# -ge 2 ]] || usage
		transcript_override="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

command -v python3 >/dev/null 2>&1 || {
	echo "canary-copilot-usagemapping: python3 not found on PATH" >&2
	exit 2
}

[[ -f "$MAPPING_FILE" ]] || {
	echo "canary-copilot-usagemapping: usage-mapping.yaml not found: $MAPPING_FILE" >&2
	exit 2
}

[[ -f "$ADAPTER_SCRIPT" ]] || {
	echo "canary-copilot-usagemapping: lifecycle-worker-adapter.sh not found: $ADAPTER_SCRIPT" >&2
	exit 2
}

transcripts=()
if [[ -n "$transcript_override" ]]; then
	[[ -f "$transcript_override" ]] || {
		echo "canary-copilot-usagemapping: --transcript file not found: $transcript_override" >&2
		exit 2
	}
	transcripts=("$transcript_override")
else
	shopt -s nullglob
	transcripts=("$DEFAULT_FIXTURE_DIR"/clean-*.log)
	if [[ "${#transcripts[@]}" -eq 0 ]]; then
		transcripts=("$DEFAULT_FIXTURE_DIR"/*.log)
	fi
	shopt -u nullglob
	[[ "${#transcripts[@]}" -gt 0 ]] || {
		echo "canary-copilot-usagemapping: no committed envelope fixtures found under $DEFAULT_FIXTURE_DIR" >&2
		exit 2
	}
fi

python3 - "$MAPPING_FILE" "$ADAPTER_SCRIPT" "${transcripts[@]}" <<'PYEOF'
import json
import re
import sys
from pathlib import Path

import yaml


def _load_adapter_module(adapter_path):
    content = Path(adapter_path).read_text(encoding="utf-8")
    match = re.search(r"<<'PYEOF'\n(.*)\nPYEOF", content, re.DOTALL)
    if not match:
        raise RuntimeError("could not find embedded Python in lifecycle-worker-adapter.sh")
    py_code = match.group(1)
    py_code_no_main = re.sub(r"\nmain\(\)\s*$", "\n", py_code)
    ns = {}
    exec(py_code_no_main, ns)
    return ns


def extract_stdout_block(transcript_text):
    start_marker = "---STDOUT---\n"
    start = transcript_text.find(start_marker)
    if start == -1:
        # Fallback: if transcript is directly JSONL
        if transcript_text.strip().startswith("{") and "type" in transcript_text:
            return transcript_text.strip()
        return None
    rest = transcript_text[start + len(start_marker):]
    end_marker = "\n---STDERR---"
    end = rest.find(end_marker)
    if end == -1:
        return rest
    return rest[:end]


def resolve_slot(envelope, envelope_path):
    if envelope_path == "sorted(modelUsage keys)":
        model_usage = envelope.get("modelUsage")
        if not isinstance(model_usage, dict):
            return None, False
        return sorted(model_usage.keys()), True

    cur = envelope
    for part in envelope_path.split("."):
        if not isinstance(cur, dict) or part not in cur:
            return None, False
        cur = cur[part]
    return cur, True


def fail(field):
    sys.stderr.write("FAIL: %s\n" % field)
    sys.exit(1)


mapping_path = sys.argv[1]
adapter_path = sys.argv[2]
transcript_paths = sys.argv[3:]

with open(mapping_path) as f:
    mapping = yaml.safe_load(f)

if not isinstance(mapping, dict):
    sys.stderr.write("canary-copilot-usagemapping: usage-mapping.yaml is not a YAML mapping\n")
    sys.exit(2)

provider = mapping.get("providers", {}).get("github_copilot_cli", {})
if provider.get("status") != "mapped":
    sys.stderr.write(
        "canary-copilot-usagemapping: github_copilot_cli is not declared 'mapped' in %s\n" % mapping_path
    )
    sys.exit(2)

slots = provider.get("slots", {})
if not slots:
    sys.stderr.write(
        "canary-copilot-usagemapping: github_copilot_cli has no slots in %s\n" % mapping_path
    )
    sys.exit(2)

# Ensure total_cost is recognized as unmapped
if "total_cost" in slots:
    sys.stderr.write(
        "canary-copilot-usagemapping: total_cost is declared in slots but must be unmapped\n"
    )
    fail("total_cost_must_be_unmapped")

adapter = _load_adapter_module(adapter_path)
provider_measurements = adapter["provider_measurements"]

for transcript_path in transcript_paths:
    with open(transcript_path) as f:
        content = f.read()

    stdout_block = extract_stdout_block(content)
    if stdout_block is None:
        sys.stderr.write(
            "canary-copilot-usagemapping: envelope_source_unavailable: %s\n" % transcript_path
        )
        fail("envelope_source_unavailable transcript=%s" % transcript_path)

    # Decode JSON or parse JSONL events via provider_measurements
    # Check if raw stdout is direct JSON or JSONL stream
    measurements = {}
    is_direct_json = False
    try:
        doc = json.loads(stdout_block)
        if isinstance(doc, dict):
            is_direct_json = True
            envelope = doc
    except (ValueError, TypeError):
        pass

    if not is_direct_json:
        # JSONL stream: first check that the stream has valid JSONL objects
        lines = [l.strip() for l in stdout_block.splitlines() if l.strip()]
        valid_events = []
        for line in lines:
            try:
                ev = json.loads(line)
                if isinstance(ev, dict):
                    valid_events.append(ev)
            except (ValueError, TypeError):
                pass

        if not valid_events:
            sys.stderr.write(
                "canary-copilot-usagemapping: envelope_source_unavailable: %s\n" % transcript_path
            )
            fail("envelope_source_unavailable transcript=%s" % transcript_path)

        # Inspect raw events for slot availability before adapter fallback
        checkpoint_models = {}
        result_usage = {}
        has_checkpoint = False
        has_result = False
        session_id_val = None

        for ev in valid_events:
            ev_type = ev.get("type")
            data = ev.get("data")
            if not isinstance(data, dict):
                continue
            if ev_type == "session.usage_checkpoint":
                has_checkpoint = True
                cache_state = data.get("promptCacheBreakState")
                if isinstance(cache_state, dict):
                    if isinstance(cache_state.get("models"), dict):
                        checkpoint_models = cache_state["models"]
                    elif isinstance(cache_state.get("main"), dict) and isinstance(cache_state["main"].get("models"), dict):
                        checkpoint_models = cache_state["main"]["models"]
                elif isinstance(cache_state, list):
                    for item in cache_state:
                        if isinstance(item, dict) and isinstance(item.get("models"), dict):
                            checkpoint_models = item["models"]
                            break
            elif ev_type == "result":
                has_result = True
                if isinstance(data.get("usage"), dict):
                    result_usage = data["usage"]
                sid = ev.get("sessionId") or data.get("sessionId")
                if sid:
                    session_id_val = sid
            elif ev_type == "session.start":
                sid = data.get("sessionId")
                if sid and not session_id_val:
                    session_id_val = sid

        if not has_checkpoint and not has_result:
            sys.stderr.write(
                "canary-copilot-usagemapping: envelope_source_unavailable: missing usage events in %s\n"
                % transcript_path
            )
            fail("envelope_source_unavailable transcript=%s" % transcript_path)

        # Verify raw availability for slots that could be defaulted by adapter
        # cache_read_input_tokens: cache_read or cache_read_input_tokens in checkpoint model entry
        if "cache_read_input_tokens" in slots:
            slot_info = slots["cache_read_input_tokens"]
            envelope_path = slot_info.get("envelope_path", "")
            found = False
            for m_name, m_data in checkpoint_models.items():
                if isinstance(m_data, dict) and ("cache_read" in m_data or "cache_read_input_tokens" in m_data):
                    found = True
                    break
            if not found:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=cache_read_input_tokens envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=cache_read_input_tokens envelope_path=%s" % envelope_path)

        # cache_creation_input_tokens: cache_write or cache_creation_input_tokens in checkpoint model entry
        if "cache_creation_input_tokens" in slots:
            slot_info = slots["cache_creation_input_tokens"]
            envelope_path = slot_info.get("envelope_path", "")
            found = False
            for m_name, m_data in checkpoint_models.items():
                if isinstance(m_data, dict) and ("cache_write" in m_data or "cache_creation_input_tokens" in m_data):
                    found = True
                    break
            if not found:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=cache_creation_input_tokens envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=cache_creation_input_tokens envelope_path=%s" % envelope_path)

        # input_tokens: prompt_tokens or input_tokens in checkpoint model entry
        if "input_tokens" in slots:
            slot_info = slots["input_tokens"]
            envelope_path = slot_info.get("envelope_path", "")
            found = False
            for m_name, m_data in checkpoint_models.items():
                if isinstance(m_data, dict) and ("prompt_tokens" in m_data or "input_tokens" in m_data):
                    found = True
                    break
            if not found:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=input_tokens envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=input_tokens envelope_path=%s" % envelope_path)

        # output_tokens: completion_tokens or output_tokens in checkpoint model entry or result usage
        if "output_tokens" in slots:
            slot_info = slots["output_tokens"]
            envelope_path = slot_info.get("envelope_path", "")
            found = False
            if "output_tokens" in result_usage or "completion_tokens" in result_usage:
                found = True
            if not found:
                for m_name, m_data in checkpoint_models.items():
                    if isinstance(m_data, dict) and ("completion_tokens" in m_data or "output_tokens" in m_data):
                        found = True
                        break
            if not found:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=output_tokens envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=output_tokens envelope_path=%s" % envelope_path)

        # api_active_duration_ms: totalApiDurationMs in result usage
        if "api_active_duration_ms" in slots:
            slot_info = slots["api_active_duration_ms"]
            envelope_path = slot_info.get("envelope_path", "")
            if "totalApiDurationMs" not in result_usage:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=api_active_duration_ms envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=api_active_duration_ms envelope_path=%s" % envelope_path)

        # provider_session_id: sessionId in result or session.start
        if "provider_session_id" in slots:
            slot_info = slots["provider_session_id"]
            envelope_path = slot_info.get("envelope_path", "")
            if not session_id_val:
                sys.stderr.write(
                    "canary-copilot-usagemapping: usage_slot_unavailable: slot=provider_session_id envelope_path=%s\n"
                    % envelope_path
                )
                fail("usage_slot_unavailable slot=provider_session_id envelope_path=%s" % envelope_path)

        # JSONL stream: use provider_measurements to extract provider_usage_envelope
        measurements = provider_measurements(stdout_block)
        if not measurements:
            sys.stderr.write(
                "canary-copilot-usagemapping: envelope_source_unavailable: %s\n" % transcript_path
            )
            fail("envelope_source_unavailable transcript=%s" % transcript_path)
        envelope = measurements.get("provider_usage_envelope", {})
        if not envelope or not isinstance(envelope, dict):
            sys.stderr.write(
                "canary-copilot-usagemapping: envelope_source_unavailable: no provider_usage_envelope in %s\n"
                % transcript_path
            )
            fail("envelope_source_unavailable transcript=%s" % transcript_path)

    # Validate each declared slot
    for slot in sorted(slots.keys()):
        envelope_path = slots[slot].get("envelope_path", "")
        _, found = resolve_slot(envelope, envelope_path)
        if not found:
            sys.stderr.write(
                "canary-copilot-usagemapping: usage_slot_unavailable: slot=%s envelope_path=%s transcript=%s\n"
                % (slot, envelope_path, transcript_path)
            )
            fail("usage_slot_unavailable slot=%s envelope_path=%s" % (slot, envelope_path))

    sys.stderr.write(
        "canary-copilot-usagemapping: all %d github_copilot_cli slots resolved against %s\n"
        % (len(slots), transcript_path)
    )

sys.stderr.write("PASS\n")
PYEOF
