---
feature_key: E40-F13-copilot-backed-model-routing-for-e40-lifecycle-ben
epic_key: E40
doc_type: test-plan
spec: spec.md
feature_spec: feature.md
research_report: research-report.md
status: APPROVED
complexity: COMPLEX
size: L
---

# Test Plan: E40-F13 — Copilot-backed model routing for E40 lifecycle benchmarks

**Created:** 2026-09-19
**Feature spec:** `spec.md` (Parts 1-6)
**Feature PRD / Triage:** `feature.md`
**Research report:** `research-report.md`
**Status:** APPROVED
**Quality Gate / Framework:** Shift-Left QA (Contract Verification, Unit & Integration Coverage, BVA, Isolation Guarantees)

---

## 0. Product critical-path guard

The product critical-path guard checks four foundational roadmap documents at test planning time:
- `docs/product/D01-vision-statement.md`: MISSING (advisory)
- `docs/product/D02-success-criteria.md`: MISSING (advisory)
- `docs/plan/product-delivery-roadmap.md`: MISSING (advisory)
- `docs/plan/product-critical-path.md`: MISSING (advisory)

As recorded in `spec.md` §0, these files are absent from the repository. Per the product critical-path guard policy, their absence is recorded as an advisory finding and test planning proceeds without blocking.

---

## 1. Executive Summary & Quality Strategy

### 1.1 Objective
Feature E40-F13 enables E40 lifecycle benchmarks to execute the provider and model selected by each Shark workflow route through GitHub Copilot CLI. The existing lifecycle worker adapter (`bench/scripts/lifecycle-worker-adapter.sh`) implemented native paths for Claude and Codex but lacked native Copilot execution, falling back to a global provider-command override that flattened per-route model selections. Additionally, Copilot CLI outputs multi-line streaming JSONL, breaking the adapter's single-object parser and failing envelope/token extraction. Furthermore, Copilot reports AI credit metrics (`totalNanoAiu` and `premiumRequests`) rather than USD costs; synthesizing or guessing USD costs violates evidence integrity (B064).

This test plan delivers rigorous shift-left verification across six core areas:
1. **Native Copilot Command Construction**: CLI arguments, non-interactive flags, model and effort parameter binding, tool permissions, and scratch root confinement.
2. **JSONL Stream Normalization & Control Envelope Extraction**: Parsing multi-line JSONL streams without memory exhaustion, extracting `assistant.message` payloads, and resolving both structured control envelopes and prose fallback outcomes (`RECOMMENDED OUTCOME: <outcome>`).
3. **Usage Extraction & USD Cost Honesty**: Extracting token counts, cache read/write, model identity, durations, and credit nano-units from `session.usage_checkpoint` and `result` events while strictly omitting synthetic USD cost and asserting `usage_slot_unavailable` in stage snapshots.
4. **Usage Mapping Integration (X-09 & I-05)**: Extending `bench/evidence/usage-mapping.yaml` with the `github_copilot_cli` block, validating schema conformance via Go contract tests (`TestTC042_I05StageEvidenceContract`), and asserting zero-drift mapping via canary execution (`canary-copilot-usagemapping.sh`).
5. **Route-Aware Preflight Readiness**: Ensuring `e40_benchmark.py` preflight verification allows route-aware adapters without requiring global `runtime.provider_command` overrides.
6. **Scratch Root Sandbox Confinement**: Enforcing strict containment within `scratch_root` via `-C` and `--disallow-temp-dir`, preventing repository mutations or credential leakage.

---

## 2. Spec Drift Analysis

A comprehensive comparison between `feature.md`, `research-report.md`, and `spec.md` was conducted:

1. **Prerequisite & Environment Alignment**: `research-report.md` confirmed GitHub Copilot CLI v1.0.86 is installed at `/home/jwwel/.nvm/versions/node/v20.20.0/bin/copilot`. Piped stdin execution via `copilot --output-format json` operates non-interactively without requiring `-p -`. `spec.md` §2.3.3 correctly reflects piped stdin delivery.
2. **JSONL Stream Parsing vs. Single Object Assumption**: `feature.md` and `research-report.md` identified that Copilot CLI outputs a streaming JSONL event format, while `lifecycle-worker-adapter.sh` assumed a single JSON document (Claude format). `spec.md` Part 2 §2.4 specifies line-delimited scanning for `assistant.message` and `session.usage_checkpoint`. No drift.
3. **Cost Honesty & Absence of Synthetic Multipliers**: `feature.md`, `research-report.md`, and `spec.md` §1.2 (REQ-F-005) uniformly reject converting `totalNanoAiu` to USD cost via hardcoded multipliers. Stage snapshots must record `usage_slot_unavailable` for missing `total_cost`. No drift.
4. **Scope Boundaries**: Shark workflow state authority (claims, leases, heartbeats, transitions, releases) remains exclusively in Shark core (`run-lifecycle.sh`). Provider session metadata is observational only. No drift.

---

## 3. Traceability Matrix: Acceptance Criteria to Test Cases

Every Functional and Non-Functional Requirement from `spec.md` maps directly to concrete, executable test cases.

| Acceptance Criterion | Category | Target Component / Seam | Test Case ID | Test Script / Suite |
|---|---|---|---|---|
| **AC-F13-01** | Invocation & Flags | `lifecycle-worker-adapter.sh:command_for()` | **TC-F13-01** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_command_construction` |
| **AC-F13-02** | Model & Effort Pinned | `lifecycle-worker-adapter.sh:command_for()` | **TC-F13-02** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_model_and_effort_flags` |
| **AC-F13-03** | JSONL Control Envelope | `lifecycle-worker-adapter.sh:decode_envelope()` | **TC-F13-03** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_jsonl_decode_envelope` |
| **AC-F13-04** | Text Outcome Fallback | `lifecycle-worker-adapter.sh:decode_envelope()` | **TC-F13-04** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_text_outcome_fallback` |
| **AC-F13-05** | Usage Measurement Extraction | `lifecycle-worker-adapter.sh:provider_measurements()` | **TC-F13-05** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_measurements_extraction` |
| **AC-F13-06** | Zero USD Cost Fabrication | `lifecycle-worker-adapter.sh:provider_measurements()` | **TC-F13-06** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_cost_never_fabricated` |
| **AC-F13-07** | Usage Mapping Schema & Go Contract | `bench/evidence/usage-mapping.yaml` | **TC-F13-07** | `tests/contracts/e40_i05_stage_evidence_contract_test.go` |
| **AC-F13-08** | Real Capture Canary Verification | `bench/scripts/canary-copilot-usagemapping.sh` | **TC-F13-08** | `bench/scripts/canary-copilot-usagemapping.sh` |
| **AC-F13-09** | Runner Provider & Usage Resolution | `bench/scripts/run-lifecycle.sh:mapped_provider()` | **TC-F13-09** | `bench/scripts/tests/tc121_copilot_runner_usage_test.sh` |
| **AC-F13-10** | Route-Aware Preflight Readiness | `bench/scripts/lib/e40_benchmark.py` | **TC-F13-10** | `bench/scripts/tests/tc103_preflight_fail_closed_test.sh` |
| **AC-F13-11** | Routing Identity & Digests | `bench/scripts/lib/e40_benchmark.py:workflow_routing_identity()` | **TC-F13-11** | `bench/scripts/tests/tc122_copilot_routing_identity_test.py` |
| **AC-F13-12** | Sandbox & Scratch Confinement | Subprocess execution environment & flags | **TC-F13-12** | `bench/scripts/tests/tc120_copilot_adapter_test.py::test_copilot_scratch_confinement` |
| **AC-F13-13** | Routing Profiles Conformance | `bench/profiles/*.yaml` | **TC-F13-13** | `bench/scripts/tests/tc122_copilot_routing_identity_test.py::test_profiles_schema_conformance` |

---

## 4. Test Case Specifications & Caller-Path Contracts

### 4.1 Unit Test Suite: `tc120_copilot_adapter_test.py`

#### TC-F13-01: Copilot Command Construction
- **Objective**: Verify that `command_for()` generates the exact Copilot CLI argv when `request["provider"]` is `copilot` or alias (`github-copilot`, `github_copilot`, `github`).
- **Caller-Path Contract**: Subprocess execution of `lifecycle-worker-adapter.sh` or direct module import of adapter logic with controlled `request` payloads.
- **Inputs**:
  ```json
  {
    "provider": "copilot",
    "model": "claude-sonnet-5",
    "scratch_root": "/sandbox/scratch-001"
  }
  ```
- **Expected Outputs**:
  - Base binary defaults to `copilot` (or `args.provider_command` if passed).
  - Flags present: `--output-format json`, `--stream off`, `--model claude-sonnet-5`, `-C /sandbox/scratch-001`, `--disallow-temp-dir`, `--no-custom-instructions`, `--no-ask-user`, `--allow-all-tools`, `--disable-builtin-mcps`, `--no-color`.
- **Failure Modes Checked**: Missing model raises `AdapterError`; missing scratch path does not crash but omits `-C`; missing temp-dir disallow flag fails closed.

#### TC-F13-02: Model and Reasoning Effort Pinned Flags
- **Objective**: Assert that `--model` matches `request["model"]` exactly and `--reasoning-effort` is appended when `request["effort"]` is specified.
- **Inputs & Test Matrix**:
  | Test Case Variant | Model in Request | Effort in Request | Expected Flags |
  |---|---|---|---|
  | Standard Model | `gemini-3.8-flash` | `None` | `--model gemini-3.8-flash` (no effort flag) |
  | Minimal Effort | `gemini-3.8-flash` | `minimal` | `--model gemini-3.8-flash --reasoning-effort minimal` |
  | High Effort | `claude-sonnet-5` | `high` | `--model claude-sonnet-5 --reasoning-effort high` |
  | XHigh Effort | `gpt-5.4` | `xhigh` | `--model gpt-5.4 --reasoning-effort xhigh` |
  | Unavailable Effort | `claude-sonnet-5` | `unavailable` | `--model claude-sonnet-5` (effort flag suppressed) |
- **Expected Outputs**: All flags strictly formatted, lowercase normalized, and free of whitespace pollution.

#### TC-F13-03: JSONL Control Envelope Decoding
- **Objective**: Verify that `decode_envelope()` correctly parses real Copilot multi-line JSONL streams, finds `assistant.message`, and decodes structured JSON control envelopes.
- **Fixture Input**: Captured Copilot JSONL stream:
  ```json
  {"type":"session.start","data":{"sessionId":"copilot-session-101"}}
  {"type":"model.call_start","data":{"model":"claude-sonnet-5"}}
  {"type":"assistant.message","data":{"content":"{\"kind\":\"final\",\"recommended_outcome\":\"pass\",\"evidence\":[{\"kind\":\"test\",\"summary\":\"all tests passing\"}]}","model":"claude-sonnet-5"}}
  {"type":"result","data":{"exitCode":0,"usage":{"totalApiDurationMs":1450}}}
  ```
- **Expected Outputs**:
  ```json
  {
    "kind": "final",
    "recommended_outcome": "pass",
    "evidence": [{"kind": "test", "summary": "all tests passing"}]
  }
  ```
- **Boundary Verification**: Verify trailing `result` event is not mistakenly treated as the envelope.

#### TC-F13-04: Prose Outcome Fallback Parsing
- **Objective**: Verify that unstructured assistant responses containing `RECOMMENDED OUTCOME: <outcome>` normalize to a valid control envelope.
- **Fixture Input**: Copilot `assistant.message` containing text:
  ```
  Investigation complete. All criteria verified against the specification.
  RECOMMENDED OUTCOME: pass
  ```
- **Expected Outputs**:
  ```json
  {
    "kind": "final",
    "recommended_outcome": "pass",
    "evidence": []
  }
  ```
- **Negative Variant**: If neither structured JSON nor `RECOMMENDED OUTCOME:` is present, `decode_envelope()` must raise `AdapterError("provider output does not contain a control envelope")`.

#### TC-F13-05: Provider Measurements Extraction
- **Objective**: Assert accurate extraction of token metrics, caching metrics, active models, durations, turn count, and session IDs from JSONL event types.
- **Fixture Input**: JSONL stream containing `session.usage_checkpoint`, `assistant.message`, and `result` events.
- **Expected Outputs**:
  - `measurements["usage"]["input_tokens"]` matches `prompt_tokens`.
  - `measurements["usage"]["cache_read_input_tokens"]` matches `cache_read`.
  - `measurements["usage"]["cache_creation_input_tokens"]` matches `cache_write`.
  - `measurements["usage"]["model_ids"]` contains `["claude-sonnet-5"]` (sorted list).
  - `measurements["usage"]["api_active_duration_ms"]` matches `totalApiDurationMs`.
  - `measurements["usage"]["turn_count"]` equals total `assistant.message` events.
  - `measurements["provider_usage_envelope"]` mirrors raw provider slot structures for downstream mapping.

#### TC-F13-06: Zero USD Cost Fabrication (Cost Honesty)
- **Objective**: Assert that `total_cost_usd` or `cost_usd` is NEVER synthesized from `totalNanoAiu` or defaulted to `0.0`.
- **Inputs**: Copilot JSONL stream reporting `totalNanoAiu: 125000000` and `totalPremiumRequests: 1`.
- **Expected Outputs**:
  - `measurements.get("cost_usd")` is `None` (omitted).
  - `measurements.get("total_cost_usd")` is `None` (omitted).
  - `measurements["usage"].get("cost_usd")` is absent.
  - Credit metrics (`total_nano_aiu`, `premium_requests`) are preserved as raw provider envelope properties without financial claims.

#### TC-F13-12: Scratch Root Sandbox Confinement
- **Objective**: Ensure child worker processes are restricted to the scenario scratch directory.
- **Inputs**: Environment variable `LIFECYCLE_SCRATCH_ROOT` or `request["scratch_root"]`.
- **Expected Outputs**:
  - Generated command includes `-C <scratch_root>`.
  - Flag `--disallow-temp-dir` is unconditionally included.
  - Assert that tool executions cannot write to `/tmp`, parent git checkouts, or evaluator directories.

---

### 4.2 Contract & Integration Tests

#### TC-F13-07: Usage Mapping Go Contract Test
- **Objective**: Verify that adding `github_copilot_cli` to `bench/evidence/usage-mapping.yaml` satisfies all Go contract assertions in `tests/contracts/e40_i05_stage_evidence_contract_test.go`.
- **Caller-Path Contract**: `go test -v ./tests/contracts -run TestTC042_I05StageEvidenceContract`
- **Validation Criteria**:
  - Schema version is `1.0`.
  - `providers.github_copilot_cli.status` is `mapped`.
  - Declared slots include: `input_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `model_ids`, `api_active_duration_ms`, `turn_count`, `provider_session_id`.
  - `total_cost` is NOT in `github_copilot_cli.slots` (unmapped).
  - All declared slots have `verification_tier: real_capture` (except unverified session ID if applicable).
  - `required_identity_slots` remains consistent with cross-variant comparison requirements.

#### TC-F13-08: Usage Mapping Canary Verification
- **Objective**: Execute `bench/scripts/canary-copilot-usagemapping.sh` against real-captured Copilot transcript files.
- **Caller-Path Contract**: `./bench/scripts/canary-copilot-usagemapping.sh --transcript <transcript_path>`
- **Expected Outputs**:
  - Canary parses transcript's `---STDOUT---` JSONL block.
  - Every mapped slot in `usage-mapping.yaml` resolves successfully against the captured envelope.
  - Last line of stderr is `PASS`. Exit code is `0`.

#### TC-F13-09: Lifecycle Runner Provider & Usage Resolution (`tc121_copilot_runner_usage_test.sh`)
- **Objective**: Verify `mapped_provider()` and `resolve_usage()` in `bench/scripts/run-lifecycle.sh`.
- **Inputs**: Dispatched response with `provider: "copilot"` and mock worker envelope.
- **Expected Outputs**:
  - `mapped_provider(response)` returns `"github_copilot_cli"`.
  - `resolve_usage("github_copilot_cli", envelope)` resolves all 7 available slots.
  - `errors` list contains `{"kind": "usage_slot_unavailable", "slot": "total_cost", ...}`, honestly recording missing cost without corrupting the run.

#### TC-F13-10: Route-Aware Preflight Readiness (`tc103_preflight_fail_closed_test.sh`)
- **Objective**: Verify `preflight_provider_command_finding()` in `bench/scripts/lib/e40_benchmark.py` passes when `runtime.lifecycle_adapter` is set, even if `runtime.provider_command` is unset.
- **Test Scenarios**:
  1. `lifecycle_adapter` unset -> Fails P4 (`lifecycle_adapter is not configured`).
  2. `lifecycle_adapter` set to custom unknown script without `provider_command` -> Fails P4 (`runtime.provider_command is not configured`).
  3. `lifecycle_adapter` set to route-aware `lifecycle-worker-adapter.sh` with Copilot workflow routes -> Passes P4 (`None` finding).

#### TC-F13-11: Model Routing Identity & Profiles Conformance (`tc122_copilot_routing_identity_test.py`)
- **Objective**: Verify that `workflow_routing_identity()` correctly computes deterministic digests (`routing_digest`, `provider_digest`, `model_digest`, `effort_digest`) for workflow bundles containing Copilot steps.
- **Validation Criteria**:
  - Profiles `bench/profiles/copilot-default.yaml` and `bench/profiles/copilot-variant-review.yaml` conform to `bench/profiles/schema.yaml`.
  - Mutating a model in a workflow step (e.g. `claude-sonnet-5` -> `gpt-5.4`) changes `model_digest` and `routing_digest` while preserving `provider_digest`.
  - Mutating reasoning effort changes `effort_digest` and `routing_digest`.

---

## 5. Non-Functional & Boundary Test Scenarios

### 5.1 ISO 25010 Quality Characteristics Matrix

| Characteristic | Sub-characteristic | Requirement | Test Case ID | Verification Method |
|---|---|---|---|---|
| **Functional Suitability** | Completeness & Correctness | REQ-F-001..004 | TC-F13-01..05 | Unit & integration tests on command assembly, parsing, and measurement extraction. |
| **Reliability** | Fault Tolerance / Fail-Closed | REQ-NF-003 | TC-F13-14 (below) | Malformed JSONL, non-zero subprocess exit, timeout handling. |
| **Security** | Isolation & Data Protection | REQ-NF-001, REQ-F-007 | TC-F13-12 | Scratch root confinement, temp-dir disallow, credential redaction. |
| **Maintainability** | Analysability & Testability | REQ-F-005, REQ-F-006 | TC-F13-06, 07, 08 | Strict usage mapping, honest slot omission, canary verification. |
| **Performance Efficiency** | Time & Memory Boundedness | REQ-NF-004 | TC-F13-15 (below) | Streaming line-by-line processing, discarding unneeded raw event data. |

### 5.2 Negative & Boundary Test Cases

#### TC-F13-14: Fail-Closed Subprocess & Error Recovery
- **Malformed Event Stream**: Input stream contains corrupted JSON on line 3. Adapter parses valid earlier/later lines or gracefully fails closed with `AdapterError("provider output is not a JSON control envelope")`.
- **Non-Zero Exit Code**: Copilot CLI exits with code `1` or `127`. Adapter captures stderr, creates structured error result, and does not hang the runner.
- **Execution Timeout**: Worker times out after configured threshold. Subprocess tree is cleanly terminated; status recorded as `timeout`.

#### TC-F13-15: Memory Bounded Stream Processing
- **Large Context / Diff Streaming**: Copilot emits 10,000 JSONL events containing token-by-token diff fragments. Adapter buffers only target events (`assistant.message`, `session.usage_checkpoint`, `result`), keeping memory usage strictly bounded (<50MB RSS).

---

## 6. Test Environment & Execution Commands

### 6.1 Prerequisites
- Python 3.10+ with `pyyaml`
- Go 1.22+
- GitHub Copilot CLI v1.0.86+ installed on PATH
- Local environment scratch directory (`scripts/shark-scratch-env.sh`)

### 6.2 Execution Runbook

```bash
# 1. Run Adapter Unit Tests
pytest -v bench/scripts/tests/tc120_copilot_adapter_test.py

# 2. Run Go Usage Mapping Contract Test
go test -v ./tests/contracts -run TestTC042_I05StageEvidenceContract

# 3. Run Usage Mapping Canary Against Captured Transcripts
./bench/scripts/canary-copilot-usagemapping.sh

# 4. Run Preflight Readiness Test
./bench/scripts/tests/tc103_preflight_fail_closed_test.sh

# 5. Run Routing Identity & Profiles Conformance Tests
pytest -v bench/scripts/tests/tc122_copilot_routing_identity_test.py

# 6. Run Full Repository Quality Gate
make fmt && make lint && make test
```

---

## 7. Exit Gate Criteria Checklist

To advance feature E40-F13 from test planning to implementation:
- [x] All 13 Acceptance Criteria from `spec.md` (§1.4) are mapped 1:1 to concrete test cases.
- [x] Caller-path contracts and production entrypoints are explicitly documented.
- [x] Negative, error, boundary, and isolation scenarios are fully covered.
- [x] Zero USD cost fabrication constraint (REQ-F-005, AC-F13-06) is enforced with dedicated negative tests.
- [x] Sandboxing and scratch root confinement flags (`-C`, `--disallow-temp-dir`) are validated.
- [x] File paths and execution commands match existing E40 benchmark test conventions.
