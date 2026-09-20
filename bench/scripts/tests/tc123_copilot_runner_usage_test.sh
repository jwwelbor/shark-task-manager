#!/usr/bin/env bash
# TC-F13-09 (test-plan.md AC-F13-09; T-E40-F13-006 task spec).
#
# Exercises AC-F13-09 (REQ-F-005, REQ-F-006, Part 2 §2.6.2, Part 3 I-05, I-07):
# 1. mapped_provider(response) in bench/scripts/run-lifecycle.sh returns
#    "github_copilot_cli" for "copilot", "github", and variants.
# 2. resolve_usage() extracts all 7 mapped slots from worker response
#    measurements via usage-mapping.yaml.
# 3. Stage evidence snapshots record an entry in errors indicating
#    "usage_slot_unavailable" for "total_cost".
# 4. Preserves raw credit metrics in stage snapshot telemetry without
#    fabricating USD costs.
# 5. Missing total_cost is recorded honestly without corrupting the run into
#    a fatal evidence error stop outcome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
RUNNER="$BENCH_DIR/scripts/run-lifecycle.sh"
USAGE_MAPPING="$BENCH_DIR/evidence/usage-mapping.yaml"

fail() {
	echo "TC-F13-09 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUNNER" ]] || fail "run-lifecycle.sh missing or not executable: $RUNNER"
[[ -f "$USAGE_MAPPING" ]] || fail "usage-mapping.yaml missing: $USAGE_MAPPING"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$REPO_ROOT/.test_work_tc123_$$"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT
mkdir -p "$WORKDIR"

# ---------------------------------------------------------------------------
# Python test driver loading run-lifecycle.sh logic
# ---------------------------------------------------------------------------
python3 - "$RUNNER" "$BENCH_DIR" "$REPO_ROOT" "$WORKDIR" <<'PYEOF'
import json
import os
import pathlib
import sys

runner_script = pathlib.Path(sys.argv[1])
bench_dir = pathlib.Path(sys.argv[2]).resolve()
repo_root = pathlib.Path(sys.argv[3]).resolve()
work_dir = pathlib.Path(sys.argv[4]).resolve()

os.environ["LIFECYCLE_BENCH_DIR"] = str(bench_dir)

# Load run-lifecycle.sh python code
with open(runner_script, encoding="utf-8") as f:
    content = f.read()

py_code = content.split("<<'PY'")[1].rsplit("PY", 1)[0]
scope = {"__name__": "imported"}
exec(compile(py_code, "run-lifecycle.py", "exec"), scope)

mapped_provider = scope["mapped_provider"]
resolve_usage = scope["resolve_usage"]
provider_usage_envelope = scope["provider_usage_envelope"]
I05BundleWriter = scope["I05BundleWriter"]
stage_snapshot_digest = scope["stage_snapshot_digest"]

# ---------------------------------------------------------------------------
# Test 1: mapped_provider(response) mapping
# ---------------------------------------------------------------------------
print("--> Test 1: mapped_provider mapping")
assert mapped_provider({"provider": "copilot"}) == "github_copilot_cli"
assert mapped_provider({"provider": "github"}) == "github_copilot_cli"
assert mapped_provider({"provider": "github-copilot"}) == "github_copilot_cli"
assert mapped_provider({"provider": "github_copilot"}) == "github_copilot_cli"
assert mapped_provider({"provider": "Copilot"}) == "github_copilot_cli"
assert mapped_provider({"provider": "GITHUB"}) == "github_copilot_cli"
assert mapped_provider("copilot") == "github_copilot_cli"
assert mapped_provider("github") == "github_copilot_cli"
assert mapped_provider({"provider": "anthropic"}) == "anthropic_claude_cli"
assert mapped_provider({"provider": "claude"}) == "anthropic_claude_cli"
assert mapped_provider({"provider": "openai_codex_cli"}) == "openai_codex_cli"
assert mapped_provider("custom_provider") == "custom_provider"
print("    mapped_provider: PASS")

# ---------------------------------------------------------------------------
# Test 2: resolve_usage() extraction of all 7 mapped slots + total_cost error
# ---------------------------------------------------------------------------
print("--> Test 2: resolve_usage all 7 slots + total_cost error")
mock_envelope = {
    "usage": {
        "input_tokens": 2500,
        "output_tokens": 480,
        "cache_read_input_tokens": 1200,
        "cache_creation_input_tokens": 300,
    },
    "modelUsage": {
        "claude-sonnet-5": {},
    },
    "duration_api_ms": 4150,
    "num_turns": 3,
    "session_id": "copilot-sess-777",
    "total_nano_aiu": 1850000000,
    "premium_requests": 2,
}

usage, errors = resolve_usage("github_copilot_cli", mock_envelope)

expected_slots = {
    "input_tokens": 2500,
    "cache_read_input_tokens": 1200,
    "cache_creation_input_tokens": 300,
    "model_ids": ["claude-sonnet-5"],
    "api_active_duration_ms": 4150,
    "turn_count": 3,
    "provider_session_id": "copilot-sess-777",
}

assert len(usage) == 7, f"expected 7 slots, got {len(usage)}: {usage}"
for slot, want_val in expected_slots.items():
    assert slot in usage, f"missing expected slot {slot}"
    assert usage[slot] == want_val, f"slot {slot} value mismatch: {usage[slot]} != {want_val}"

# Assert cost honesty: no fabricated USD cost
assert "total_cost" not in usage, "total_cost must not be in usage"
assert "total_cost_usd" not in usage, "total_cost_usd must not be in usage"
assert "cost_usd" not in usage, "cost_usd must not be in usage"

# Assert total_cost recorded as usage_slot_unavailable
cost_errors = [e for e in errors if e.get("kind") == "usage_slot_unavailable" and e.get("slot") == "total_cost"]
assert len(cost_errors) == 1, f"expected exactly 1 total_cost usage_slot_unavailable error, got: {errors}"
assert cost_errors[0]["kind"] == "usage_slot_unavailable"
assert cost_errors[0]["slot"] == "total_cost"

# Also verify resolve_usage with 'copilot' short name normalizes to github_copilot_cli
usage_norm, errors_norm = resolve_usage("copilot", mock_envelope)
assert usage_norm == usage
assert errors_norm == errors
print("    resolve_usage 7 slots + cost honesty: PASS")

# ---------------------------------------------------------------------------
# Test 3: resolve_usage() missing mapped slot handling
# ---------------------------------------------------------------------------
print("--> Test 3: resolve_usage missing mapped slot")
partial_envelope = dict(mock_envelope)
partial_envelope["usage"] = {
    "output_tokens": 480,
    # input_tokens omitted
    "cache_read_input_tokens": 1200,
    "cache_creation_input_tokens": 300,
}
usage_part, errors_part = resolve_usage("github_copilot_cli", partial_envelope)
assert "input_tokens" not in usage_part
assert len(usage_part) == 6
assert any(e.get("kind") == "usage_slot_unavailable" and e.get("slot") == "input_tokens" for e in errors_part)
assert any(e.get("kind") == "usage_slot_unavailable" and e.get("slot") == "total_cost" for e in errors_part)
print("    resolve_usage missing slot: PASS")

# ---------------------------------------------------------------------------
# Test 4: provider_usage_envelope unwrap
# ---------------------------------------------------------------------------
print("--> Test 4: provider_usage_envelope unwrap")
worker_result = {
    "measurements": {
        "usage": {"input_tokens": 2500},
        "provider_usage_envelope": mock_envelope,
    }
}
unwrapped = provider_usage_envelope(worker_result)
assert unwrapped == mock_envelope
usage_unwrapped, errors_unwrapped = resolve_usage("github_copilot_cli", unwrapped)
assert len(usage_unwrapped) == 7
assert any(e.get("kind") == "usage_slot_unavailable" and e.get("slot") == "total_cost" for e in errors_unwrapped)
print("    provider_usage_envelope: PASS")

# ---------------------------------------------------------------------------
# Test 5: Stage snapshot evidence & telemetry preservation (I-05 / REQ-F-005)
# ---------------------------------------------------------------------------
print("--> Test 5: Stage snapshot evidence & telemetry preservation")
bundle_dir = work_dir / "i05_test_bundle"
bundle_dir.mkdir(parents=True, exist_ok=True)

scenario_path = bench_dir / "scenarios/packages/py-bug-due-date-boundary/package.yaml"
adapter_path = bench_dir / "scripts/lifecycle-worker-adapter.sh"

scenario = {
    "id": "test-scenario",
    "version": "1.0",
    "entity_family": "bug",
}
identity = {
    "scenario_id": "test-scenario",
    "scenario_version": "1.0",
    "scenario_path": str(scenario_path),
    "fixture_digest": "sha256:fix123",
    "shark_content_digest": "sha256:content123",
    "roots": {
        "agent_fixture_checkout": str(repo_root),
        "scratch_shark_project": str(work_dir),
        "evaluator_only": str(repo_root),
    },
}
record = {"dispatches": [], "identity": identity}
writer = I05BundleWriter(bundle_dir, str(scenario_path), scenario, identity, "tc123-run", record)

writer._phase_cache["task"] = {"develop": "development"}
writer._phase_cache["bug"] = {"develop": "development"}

dispatch = {
    "ordinal": 1,
    "response": {
        "status": "develop",
        "entity_key": "B-001",
        "entity_type": "bug",
        "provider": "copilot",
        "prompt_sha256": "abc123prompt",
    },
    "evidence_refs": {"prompt_sha256": "abc123prompt"},
}
stage_candidate = {
    "base_commit": "abc1234",
    "tree_digest": "sha256:tree123",
    "binary_diff_digest": "sha256:diff123",
    "changed_path_digest": "sha256:path123",
    "dirty_untracked_manifest": [],
    "test_suite_digest": "sha256:tests123",
    "snapshot_digest": "sha256:cand123",
}
timing = {
    "stage_start": 1000000000,
    "stage_end": 2000000000,
    "claimed": [("provider_active", 1100000000, 1900000000)],
}

writer_errors = writer.record_stage(
    dispatch=dispatch,
    stage_candidate=stage_candidate,
    shark="true",
    cwd=str(repo_root),
    worker_envelope=worker_result,
    timing=timing,
    fixture_root=str(repo_root),
    fixture_input_digest="sha256:input123",
    execution_adapter=str(adapter_path),
    lifecycle_adapter=str(adapter_path),
)

snapshot_file = bundle_dir / "stages/1-develop.json"
assert snapshot_file.is_file(), f"missing stage snapshot {snapshot_file}"
snapshot = json.loads(snapshot_file.read_text(encoding="utf-8"))

# Assert snapshot usage has all 7 slots
assert len(snapshot["usage"]) == 7
for slot in expected_slots:
    assert slot in snapshot["usage"]
assert "total_cost" not in snapshot["usage"]

# Assert snapshot errors has total_cost usage_slot_unavailable
assert any(
    e.get("kind") == "usage_slot_unavailable" and e.get("slot") == "total_cost"
    for e in snapshot["errors"]
), f"expected total_cost in snapshot errors: {snapshot['errors']}"

# Assert snapshot telemetry preserves raw credit metrics without USD fabrication
assert "telemetry" in snapshot, "snapshot missing telemetry block"
assert snapshot["telemetry"].get("total_nano_aiu") == 1850000000
assert snapshot["telemetry"].get("premium_requests") == 2
assert "total_cost" not in snapshot["telemetry"]
assert "cost_usd" not in snapshot["telemetry"]

# Assert snapshot digest integrity
assert stage_snapshot_digest(snapshot) == snapshot["snapshot_digest"]

# Assert Go contract rule (e40I05ValidateUsage): when errors[] records usage_slot_unavailable for a slot,
# that slot MUST be absent from usage
for err in snapshot["errors"]:
    if err.get("kind") == "usage_slot_unavailable":
        slot = err.get("slot")
        assert slot not in snapshot["usage"], f"slot {slot} reported unavailable but present in usage"

print("    stage snapshot & telemetry: PASS")

# ---------------------------------------------------------------------------
# Test 6: Non-corruption of run outcome from missing total_cost
# ---------------------------------------------------------------------------
print("--> Test 6: Non-corruption of run outcome")
# Emulate evidence_errors filtering logic from record_stage_and_gate
evidence_errors = []
stage_errors = list(writer_errors)
response = dispatch["response"]

filtered = [
    {"dispatch_ordinal": dispatch["ordinal"], **item}
    for item in stage_errors
    if item.get("kind") != "unmapped_provider"
    and not (
        item.get("kind") == "usage_slot_unavailable"
        and item.get("slot") == "total_cost"
        and mapped_provider(response) == "github_copilot_cli"
    )
]
evidence_errors.extend(filtered)

# In a run where only total_cost was unavailable for Copilot, evidence_errors must be empty
assert len(evidence_errors) == 0, f"evidence_errors should be empty for Copilot total_cost omission, got: {evidence_errors}"

# If terminal is "complete", evidence_errors being empty allows the run to succeed with publication_eligible=True
terminal = "complete"
assert terminal == "complete" and not evidence_errors
stop_outcome_triad = scope["stop_outcome_triad"]
stop_outcome, publication_eligible, ineligibility_reasons = stop_outcome_triad(terminal, "")
assert stop_outcome is None
assert publication_eligible is True
assert ineligibility_reasons == []

print("    non-corruption of run outcome: PASS")

print("\nALL TC-F13-09 TESTS PASSED")
PYEOF

echo "TC-F13-09: pass"
