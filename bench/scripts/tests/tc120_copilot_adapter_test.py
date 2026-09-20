"""Unit tests for GitHub Copilot CLI command assembly and sandbox confinement in lifecycle-worker-adapter.sh.

Covers AC-F13-01, AC-F13-02, and AC-F13-12 (TC-F13-01, TC-F13-02, TC-F13-12).
"""

import argparse
import os
import re
from pathlib import Path
import pytest

# Load adapter functions by parsing the embedded python script inside lifecycle-worker-adapter.sh
ADAPTER_SCRIPT = Path(__file__).resolve().parent.parent / "lifecycle-worker-adapter.sh"


def _load_adapter_module():
    content = ADAPTER_SCRIPT.read_text(encoding="utf-8")
    match = re.search(r"<<'PYEOF'\n(.*)\nPYEOF", content, re.DOTALL)
    assert match is not None, "Could not find embedded Python in lifecycle-worker-adapter.sh"
    py_code = match.group(1)
    py_code_no_main = re.sub(r"\nmain\(\)\s*$", "\n", py_code)
    ns = {}
    exec(py_code_no_main, ns)
    return ns


_adapter = _load_adapter_module()
command_for = _adapter["command_for"]
decode_envelope = _adapter["decode_envelope"]
provider_measurements = _adapter["provider_measurements"]
AdapterError = _adapter["AdapterError"]


def make_args(provider_command=None):
    return argparse.Namespace(provider_command=provider_command)


# ---------------------------------------------------------------------------
# TC-F13-01: Provider alias detection and command construction (AC-F13-01)
# ---------------------------------------------------------------------------
@pytest.mark.parametrize("provider", ["copilot", "github-copilot", "github_copilot", "github"])
def test_copilot_command_construction(provider):
    request = {
        "provider": provider,
        "model": "claude-sonnet-5",
        "scratch_root": "/sandbox/scratch-001",
    }
    args = make_args()
    cmd = command_for(args, request)

    assert cmd[0] == "copilot"
    assert "--output-format" in cmd and cmd[cmd.index("--output-format") + 1] == "json"
    assert "--stream" in cmd and cmd[cmd.index("--stream") + 1] == "off"
    assert "--model" in cmd and cmd[cmd.index("--model") + 1] == "claude-sonnet-5"
    assert "--disallow-temp-dir" in cmd
    assert "--no-custom-instructions" in cmd
    assert "--no-ask-user" in cmd
    assert "--allow-all-tools" in cmd
    assert "--disable-builtin-mcps" in cmd
    assert "--no-color" in cmd
    assert "-C" in cmd and cmd[cmd.index("-C") + 1] == "/sandbox/scratch-001"


def test_copilot_command_provider_command_override():
    request = {
        "provider": "copilot",
        "model": "claude-sonnet-5",
    }
    args = make_args(provider_command="/custom/bin/copilot-wrapper")
    cmd = command_for(args, request)
    assert cmd == ["/custom/bin/copilot-wrapper"]


def test_copilot_missing_model_raises_adapter_error():
    args = make_args()
    with pytest.raises(AdapterError, match="copilot provider requires a non-empty model"):
        command_for(args, {"provider": "copilot", "model": ""})

    with pytest.raises(AdapterError, match="copilot provider requires a non-empty model"):
        command_for(args, {"provider": "copilot"})


# ---------------------------------------------------------------------------
# TC-F13-02: Model pinned & reasoning effort mapping (AC-F13-02)
# ---------------------------------------------------------------------------
@pytest.mark.parametrize(
    "model,effort,expected_effort_flag",
    [
        ("gemini-3.8-flash", None, None),
        ("gemini-3.8-flash", "unavailable", None),
        ("gemini-3.8-flash", "", None),
        ("gemini-3.8-flash", "minimal", "minimal"),
        ("claude-sonnet-5", "low", "low"),
        ("claude-sonnet-5", "medium", "medium"),
        ("claude-sonnet-5", "high", "high"),
        ("gpt-5.4", "xhigh", "xhigh"),
        ("gpt-5.4", "max", "max"),
        ("gpt-5.4", "none", "none"),
        ("claude-sonnet-5", "HIGH", "high"),  # Case insensitive normalization
    ],
)
def test_copilot_model_and_effort_flags(model, effort, expected_effort_flag):
    request = {
        "provider": "copilot",
        "model": model,
    }
    if effort is not None:
        request["effort"] = effort

    args = make_args()
    cmd = command_for(args, request)

    assert "--model" in cmd
    assert cmd[cmd.index("--model") + 1] == model

    if expected_effort_flag is not None:
        assert "--reasoning-effort" in cmd
        assert cmd[cmd.index("--reasoning-effort") + 1] == expected_effort_flag
    else:
        assert "--reasoning-effort" not in cmd


# ---------------------------------------------------------------------------
# TC-F13-12: Scratch root sandbox containment (AC-F13-12)
# ---------------------------------------------------------------------------
def test_copilot_scratch_confinement_from_request():
    request = {
        "provider": "copilot",
        "model": "claude-sonnet-5",
        "scratch_root": "/scenario/scratch/project-a",
    }
    args = make_args()
    cmd = command_for(args, request)

    assert "-C" in cmd
    assert cmd[cmd.index("-C") + 1] == "/scenario/scratch/project-a"
    assert "--disallow-temp-dir" in cmd


def test_copilot_scratch_confinement_from_env(monkeypatch):
    monkeypatch.setenv("LIFECYCLE_SCRATCH_ROOT", "/env/scratch/project-b")
    request = {
        "provider": "copilot",
        "model": "claude-sonnet-5",
    }
    args = make_args()
    cmd = command_for(args, request)

    assert "-C" in cmd
    assert cmd[cmd.index("-C") + 1] == "/env/scratch/project-b"
    assert "--disallow-temp-dir" in cmd


def test_copilot_scratch_confinement_env_precedence(monkeypatch):
    monkeypatch.setenv("LIFECYCLE_SCRATCH_ROOT", "/env/scratch/from-env")
    request = {
        "provider": "copilot",
        "model": "claude-sonnet-5",
        "scratch_root": "/request/scratch/from-request",
    }
    args = make_args()
    cmd = command_for(args, request)

    # LIFECYCLE_SCRATCH_ROOT takes precedence over request.scratch_root
    assert "-C" in cmd
    assert cmd[cmd.index("-C") + 1] == "/env/scratch/from-env"


def test_copilot_scratch_confinement_missing():
    request = {
        "provider": "copilot",
        "model": "claude-sonnet-5",
    }
    args = make_args()
    # When neither env nor request has scratch_root, -C is omitted
    cmd = command_for(args, request)
    assert "-C" not in cmd
    assert "--disallow-temp-dir" in cmd


# ---------------------------------------------------------------------------
# TC-F13-03: JSONL Control Envelope Decoding (AC-F13-03)
# ---------------------------------------------------------------------------
def test_copilot_jsonl_decode_envelope():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-101"}}
{"type":"model.call_start","data":{"model":"claude-sonnet-5"}}
{"type":"assistant.message","data":{"content":"{\\"kind\\":\\"final\\",\\"recommended_outcome\\":\\"pass\\",\\"evidence\\":[{\\"kind\\":\\"test\\",\\"summary\\":\\"all tests passing\\"}]}","model":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":1450}}}
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "pass"
    assert envelope["evidence"] == [{"kind": "test", "summary": "all tests passing"}]


def test_copilot_jsonl_wrapped_envelope():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-102"}}
{"type":"assistant.message","data":{"content":"{\\"structured_output\\":{\\"kind\\":\\"final\\",\\"recommended_outcome\\":\\"pass\\",\\"evidence\\":[]}}","model":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0}}
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "pass"


def test_copilot_jsonl_question_envelope():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-103"}}
{"type":"assistant.message","data":{"content":"{\\"kind\\":\\"question\\",\\"question\\":\\"Should we proceed?\\",\\"evidence\\":[]}","model":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0}}
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "question"
    assert envelope["question"] == "Should we proceed?"


# ---------------------------------------------------------------------------
# TC-F13-04: Prose Outcome Fallback Parsing (AC-F13-04)
# ---------------------------------------------------------------------------
def test_copilot_text_outcome_fallback():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-104"}}
{"type":"assistant.message","data":{"content":"Investigation complete. All criteria verified against the specification.\\n\\nRECOMMENDED OUTCOME: pass\\n","model":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0}}
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "pass"
    assert envelope["evidence"] == []


def test_copilot_text_outcome_fallback_raw_stream():
    stream = """Starting analysis...
Checking requirements...
RECOMMENDED OUTCOME: fail
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "fail"
    assert envelope["evidence"] == []


# ---------------------------------------------------------------------------
# TC-F13-14 & TC-F13-15: Fail-closed & bounded stream processing (AC-F13-03, AC-F13-04)
# ---------------------------------------------------------------------------
def test_copilot_fail_closed_no_envelope():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-105"}}
{"type":"assistant.message","data":{"content":"I did some work but forgot to specify outcome.","model":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0}}
"""
    with pytest.raises(AdapterError, match="provider output does not contain a control envelope"):
        decode_envelope(stream)


def test_copilot_fail_closed_empty_output():
    with pytest.raises(AdapterError, match="provider returned no control envelope"):
        decode_envelope("   \n\t  ")


def test_copilot_tolerates_malformed_intermediate_events():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-106"}}
{malformed json line here!
{"type":"assistant.message","data":{"content":"RECOMMENDED OUTCOME: pass"}}
not a valid json event either
{"type":"result","data":{"exitCode":0}}
"""
    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "pass"


def test_copilot_bounded_stream_ignoring_irrelevant_events():
    # Simulate a stream with thousands of irrelevant token events and one assistant.message
    lines = ['{"type":"token","data":{"diff":"fragment"}}'] * 1000
    lines.append('{"type":"assistant.message","data":{"content":"{\\"kind\\":\\"final\\",\\"recommended_outcome\\":\\"pass\\",\\"evidence\\":[]}"}}')
    lines.extend(['{"type":"token","data":{"diff":"fragment"}}'] * 1000)
    lines.append('{"type":"result","data":{"exitCode":0}}')
    stream = "\n".join(lines)

    envelope = decode_envelope(stream)
    assert envelope["kind"] == "final"
    assert envelope["recommended_outcome"] == "pass"


# ---------------------------------------------------------------------------
# TC-F13-05: Provider measurement extraction from Copilot JSONL stream (AC-F13-05)
# ---------------------------------------------------------------------------
def test_copilot_measurements_extraction():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-999"}}
{"type":"model.call_start","data":{"model":"claude-sonnet-5"}}
{"type":"assistant.message","data":{"content":"First turn response","model":"claude-sonnet-5"}}
{"type":"assistant.message","data":{"content":"Second turn response","model":"claude-sonnet-5"}}
{"type":"session.usage_checkpoint","data":{"promptCacheBreakState":{"models":{"claude-sonnet-5":{"prompt_tokens":1200,"output_tokens":450,"cache_read":800,"cache_write":200}}},"totalNanoAiu":55000000,"totalPremiumRequests":2,"lastActiveModel":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":3200,"sessionDurationMs":4500},"sessionId":"copilot-session-999"}}
"""
    measurements = provider_measurements(stream)
    assert "usage" in measurements
    usage = measurements["usage"]

    assert usage["input_tokens"] == 1200
    assert usage["output_tokens"] == 450
    assert usage["cache_read_input_tokens"] == 800
    assert usage["cache_creation_input_tokens"] == 200
    assert usage["model_ids"] == ["claude-sonnet-5"]
    assert usage["api_active_duration_ms"] == 3200
    assert usage["turn_count"] == 2
    assert usage["provider_session_id"] == "copilot-session-999"

    assert "provider_usage_envelope" in measurements
    env = measurements["provider_usage_envelope"]
    assert env["usage"]["input_tokens"] == 1200
    assert env["usage"]["output_tokens"] == 450
    assert env["usage"]["cache_read_input_tokens"] == 800
    assert env["usage"]["cache_creation_input_tokens"] == 200
    assert "claude-sonnet-5" in env["modelUsage"]
    assert env["duration_api_ms"] == 3200
    assert env["sessionDurationMs"] == 4500
    assert env["num_turns"] == 2
    assert env["session_id"] == "copilot-session-999"
    assert env["total_nano_aiu"] == 55000000
    assert env["premium_requests"] == 2


def test_copilot_measurements_multiple_models_sorted():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-multi"}}
{"type":"assistant.message","data":{"content":"Turn 1","model":"gpt-5.4"}}
{"type":"assistant.message","data":{"content":"Turn 2","model":"claude-sonnet-5"}}
{"type":"session.usage_checkpoint","data":{"promptCacheBreakState":{"models":{"claude-sonnet-5":{"prompt_tokens":500,"output_tokens":100,"cache_read":100,"cache_write":50},"gpt-5.4":{"prompt_tokens":700,"output_tokens":200,"cache_read":200,"cache_write":50}}},"lastActiveModel":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":1500},"sessionId":"copilot-session-multi"}}
"""
    measurements = provider_measurements(stream)
    usage = measurements["usage"]
    assert usage["model_ids"] == ["claude-sonnet-5", "gpt-5.4"]
    assert usage["input_tokens"] == 1200
    assert usage["output_tokens"] == 300
    assert usage["cache_read_input_tokens"] == 300
    assert usage["cache_creation_input_tokens"] == 100


def test_copilot_measurements_prompt_cache_break_state_variants():
    # Test alternative nesting: data.promptCacheBreakState.main.models
    stream = """{"type":"assistant.message","data":{"content":"Turn 1","model":"gemini-3.8-flash"}}
{"type":"session.usage_checkpoint","data":{"promptCacheBreakState":{"main":{"models":{"gemini-3.8-flash":{"prompt_tokens":300,"output_tokens":50,"cache_read":50,"cache_write":25}}}}}}
{"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":800},"sessionId":"session-variant"}}
"""
    measurements = provider_measurements(stream)
    assert measurements["usage"]["input_tokens"] == 300
    assert measurements["usage"]["output_tokens"] == 50
    assert measurements["usage"]["cache_read_input_tokens"] == 50
    assert measurements["usage"]["cache_creation_input_tokens"] == 25
    assert measurements["usage"]["model_ids"] == ["gemini-3.8-flash"]


# ---------------------------------------------------------------------------
# TC-F13-06: Zero USD Cost Fabrication (Cost Honesty) (AC-F13-06)
# ---------------------------------------------------------------------------
def test_copilot_cost_never_fabricated():
    stream = """{"type":"session.start","data":{"sessionId":"copilot-session-1001"}}
{"type":"assistant.message","data":{"content":"Finished work","model":"claude-sonnet-5"}}
{"type":"session.usage_checkpoint","data":{"promptCacheBreakState":{"models":{"claude-sonnet-5":{"prompt_tokens":1000,"output_tokens":200,"cache_read":0,"cache_write":0}}},"totalNanoAiu":125000000,"totalPremiumRequests":1,"lastActiveModel":"claude-sonnet-5"}}
{"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":2500},"sessionId":"copilot-session-1001"}}
"""
    measurements = provider_measurements(stream)

    # Cost USD must NOT be present in measurements root or usage
    assert "cost_usd" not in measurements
    assert "total_cost_usd" not in measurements
    assert "cost_usd" not in measurements["usage"]
    assert "total_cost_usd" not in measurements["usage"]
    assert "cost_usd" not in measurements["provider_usage_envelope"]
    assert "total_cost_usd" not in measurements["provider_usage_envelope"]

    # Credit metrics are preserved honestly in provider_usage_envelope
    env = measurements["provider_usage_envelope"]
    assert env["total_nano_aiu"] == 125000000
    assert env["premium_requests"] == 1


