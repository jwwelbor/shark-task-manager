---
feature_key: E40-F13-copilot-backed-model-routing-for-e40-lifecycle-ben
epic_key: E40
title: Copilot-backed model routing for E40 lifecycle benchmarks — combined requirements and architecture specification
doc_type: spec
complexity: COMPLEX
size: L
research_report: research-report.md
---

# E40-F13 specification: Copilot-backed model routing for E40 lifecycle benchmarks

## 0. Product critical-path guard

The following four product critical-path files were verified at conversation start:
- `docs/product/D01-vision-statement.md`: MISSING (advisory)
- `docs/product/D02-success-criteria.md`: MISSING (advisory)
- `docs/plan/product-delivery-roadmap.md`: MISSING (advisory)
- `docs/plan/product-critical-path.md`: MISSING (advisory)

All four files are currently absent from the repository. Per the product critical-path guard policy, their absence is recorded as an advisory note and development proceeds without blocking.

---

## Part 1 — Requirements

### 1.1 Problem statement and context

Shark Bench (Epic E40) provides reproducible, controlled workflow benchmarking across lifecycle scenarios. In lifecycle v2 (E40-F05 through E40-F12), Shark workflow routes declare per-step execution targets, including `provider`, `model`, `effort`, `skills`, and `prompt`. The benchmark execution harness uses `bench/scripts/lifecycle-worker-adapter.sh` as the worker adapter bridge between the lifecycle runner (`bench/scripts/run-lifecycle.sh`) and underlying AI providers.

Existing limitations identified in `research-report.md`:
1. **Lack of native Copilot CLI support**: `lifecycle-worker-adapter.sh` only implemented native invocation logic for `claude` (Anthropic) and `codex` (OpenAI). Any attempt to run a route specifying `provider: copilot` failed closed with `no provider command configured`.
2. **Global command override bottleneck**: Setting `LIFECYCLE_PROVIDER_COMMAND` or `--provider-command` forced a single, static CLI invocation string across all workflow stages in the run. This completely defeated per-route model routing (e.g. using Sonnet for planning/development and Opus or GPT-5 for code review) and prevented comparing multi-model routing variants.
3. **JSONL stream format incompatibility**: GitHub Copilot CLI (`copilot --output-format json`) outputs an event stream of multi-line JSONL. The existing adapter expected a single JSON object (the format emitted by Claude CLI) or a reverse regex match on raw stdout. Running Copilot through the adapter caused `json.loads` crashes in `provider_measurements()` and resulted in empty usage data, while `decode_envelope()` failed to locate the control envelope.
4. **USD cost fabrication risk**: Copilot CLI outputs AI credit metrics (`totalNanoAiu` and `premiumRequests`), not USD cost. Fabricating a synthetic USD cost via a hard-coded conversion rate or defaulting to zero violates E40 evidence integrity principles (B064, REQ-F-009, REQ-F-018) and corrupts cross-variant comparison identity.
5. **Sandbox isolation**: When executing tools (file editing, bash commands), Copilot CLI must be strictly bounded to the scenario `scratch_root` to preserve E40's three-root isolation model and prevent host or repository pollution.
6. **Preflight blocking**: `bench/scripts/lib/e40_benchmark.py`'s preflight check (`preflight_provider_command_finding()`) flagged an unconfigured `provider_command` even when a route-aware adapter was present, blocking unattended benchmark runs.

### 1.2 Functional requirements

- **REQ-F-001 (Native Copilot Execution)**: The lifecycle worker adapter (`bench/scripts/lifecycle-worker-adapter.sh`) MUST natively support `provider` values matching `copilot`, `github-copilot`, `github_copilot`, or `github`. When invoked for a Copilot route, the adapter MUST construct the command using the Copilot CLI binary, passing non-interactive flags, model selection, reasoning effort, output format, and scratch directory containment.
- **REQ-F-002 (Per-Route Model Selection)**: The adapter MUST extract `model` and `effort` from the step request dictionary (`request["model"]` and `request.get("effort")`) and route to the corresponding Copilot CLI flags (`--model <model>` and `--reasoning-effort <effort>`). The dispatched route MUST take precedence over global command defaults.
- **REQ-F-003 (JSONL Stream Parsing & Control Envelope Normalization)**: The adapter MUST parse the multi-line JSONL event stream emitted by `copilot --output-format json`. It MUST locate `assistant.message` events, extract `data.content`, and decode the worker control envelope (`kind: final` or `kind: question`). If the assistant output contains text with `RECOMMENDED OUTCOME: <outcome>`, it MUST normalize that into a valid control envelope.
- **REQ-F-004 (Provider Measurement Extraction)**: The adapter MUST extract provider metrics from the Copilot JSONL stream:
  - Token counts from `session.usage_checkpoint` (`prompt_tokens`, `cache_read`, `cache_write`).
  - Active model identifiers from `assistant.message.data.model` and `session.usage_checkpoint.data.lastActiveModel`.
  - Duration from `result.usage.totalApiDurationMs` and `sessionDurationMs`.
  - Session identity from `result.sessionId`.
  - Turn counts from `assistant.message` events.
  - Credit metrics from `totalNanoAiu` and `totalPremiumRequests`.
- **REQ-F-005 (USD Cost Honesty — Zero Fabrication)**: The adapter and runner MUST NOT fabricate, estimate, or synthesize a USD cost from Copilot credit metrics (`totalNanoAiu`), nor default it to 0.0. When running Copilot, `total_cost_usd` MUST be omitted from `measurements` and `provider_usage_envelope`. In `run-lifecycle.sh` and stage evidence, the missing cost slot MUST be recorded honestly as `usage_slot_unavailable`.
- **REQ-F-006 (Usage Mapping Extension for Copilot)**: `bench/evidence/usage-mapping.yaml` MUST be extended with a `github_copilot_cli` provider block with `status: mapped`. It MUST bind verified envelope paths for all real-captured slots (`input_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `model_ids`, `api_active_duration_ms`, `turn_count`, `provider_session_id`) and leave `total_cost` unmapped.
- **REQ-F-007 (Scratch Root Sandbox Confinement)**: When invoking Copilot CLI, the adapter MUST confine file access and tool execution to the scenario scratch directory. It MUST pass `-C <scratch_root>` and `--disallow-temp-dir`, disable custom instructions (`--no-custom-instructions`), disable user interaction (`--no-ask-user`), disable built-in MCP servers (`--disable-builtin-mcps`), and disable color (`--no-color`).
- **REQ-F-008 (Route-Aware Preflight Readiness)**: `bench/scripts/lib/e40_benchmark.py` MUST be updated so `preflight_provider_command_finding()` does not block on a missing `runtime.provider_command` when `runtime.lifecycle_adapter` is route-aware and capable of resolving the workflow routes.
- **REQ-F-009 (Model Routing Profiles & Digest Verification)**: The harness MUST support versioned model routing profiles that configure per-step model assignments for Shark workflow definitions. All routing assignments MUST be captured in the run manifest's `identity.provider_routing` block, including `routing_digest`, `provider_digest`, `model_digest`, and `effort_digest`, computed deterministically by `workflow_routing_identity()`.

### 1.3 Non-functional requirements

- **REQ-NF-001 (Zero External Data Leaks)**: Prompts, scenario files, and credentials MUST NEVER be leaked into log files, environment variables, or tool execution calls outside the designated scenario scratch root.
- **REQ-NF-002 (Deterministic Offline Replay)**: Transcripts, stage snapshots, and adapter results captured from Copilot runs MUST be deterministically replayable and validatable by offline tools without requiring live network access or active Copilot authentication.
- **REQ-NF-003 (Subprocess Fail-Closed Safety)**: If Copilot CLI fails, terminates with non-zero exit code, times out, or emits invalid JSONL, the adapter MUST fail closed, returning a structured error without hanging the harness.
- **REQ-NF-004 (Bounded Memory Stream Processing)**: The adapter MUST parse JSONL lines safely without loading unbounded memory structures, discarding irrelevant streaming events (such as raw diff tokens or reasoning text) while preserving required usage checkpoints and control envelopes.

### 1.4 Acceptance criteria

| ID | Category | Criterion | Test Method |
|---|---|---|---|
| **AC-F13-01** | Invocation | `command_for()` generates valid Copilot argv with `--output-format json`, `--stream off`, `--model`, `-C`, `--disallow-temp-dir`, `--no-custom-instructions`, `--no-ask-user`, `--allow-all-tools`, and `--disable-builtin-mcps` when `request["provider"]` is `copilot`. | Unit test: `test_copilot_command_construction` |
| **AC-F13-02** | Model Pinned | The `--model` flag in the generated command strictly matches `request["model"]`. If `request["effort"]` is specified (e.g., `medium`, `high`), `--reasoning-effort` is appended. | Unit test: `test_copilot_model_and_effort_flags` |
| **AC-F13-03** | JSONL Envelope | `decode_envelope()` correctly parses a real Copilot JSONL stream, extracts `data.content` from `assistant.message`, and decodes the inner control envelope (`kind: final`, `recommended_outcome: pass`, `evidence: [...]`). | Unit test: `test_copilot_jsonl_decode_envelope` |
| **AC-F13-04** | Fallback Regex | When `assistant.message.data.content` contains unstructured text ending in `RECOMMENDED OUTCOME: pass`, `decode_envelope()` normalizes it into a valid control envelope. | Unit test: `test_copilot_text_outcome_fallback` |
| **AC-F13-05** | Usage Extraction | `provider_measurements()` extracts `input_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `model_ids`, `turn_count`, `duration_api_ms`, and `session_id` from Copilot JSONL events. | Unit test: `test_copilot_measurements_extraction` |
| **AC-F13-06** | Cost Honesty | `provider_measurements()` DOES NOT output `total_cost_usd` or `cost_usd`. It retains `total_nano_aiu` and `premium_requests` as raw provider envelope data without converting them to USD. | Unit test: `test_copilot_cost_never_fabricated` |
| **AC-F13-07** | Usage Mapping | `bench/evidence/usage-mapping.yaml` includes `github_copilot_cli` with `status: mapped` and real-capture slot bindings for all 7 available slots. `tests/contracts/e40_i05_stage_evidence_contract_test.go` passes. | Go test: `go test -v ./tests/contracts -run TestTC042_I05StageEvidenceContract` |
| **AC-F13-08** | Mapping Canary | `bench/scripts/canary-usagemapping.sh` or a dedicated canary script verifies that a captured Copilot transcript resolves all declared `github_copilot_cli` slots cleanly without drift. | Shell test: `bench/scripts/canary-copilot-usagemapping.sh` |
| **AC-F13-09** | Runner Mapping | `mapped_provider(response)` in `bench/scripts/run-lifecycle.sh` maps `copilot` and `github` to `github_copilot_cli`. `resolve_usage()` populates the stage usage block and flags `total_cost` as `usage_slot_unavailable`. | Integration test: `test_lifecycle_copilot_usage_resolution` |
| **AC-F13-10** | Preflight Readiness | `preflight_provider_command_finding()` in `e40_benchmark.py` returns `None` (passes) when `runtime.lifecycle_adapter` is route-aware, even if `runtime.provider_command` is unset. | Integration test: `tc103_preflight_fail_closed_test.sh` |
| **AC-F13-11** | Routing Identity | `workflow_routing_identity()` accurately computes `routing_digest`, `provider_digest`, `model_digest`, and `effort_digest` for workflow bundles with Copilot routes. | Unit test: `test_workflow_routing_identity_copilot` |
| **AC-F13-12** | Scratch Sandbox | Copilot CLI subprocess execution verifies working directory is confined to `scratch_root` and `--disallow-temp-dir` prevents access to system temp directories. | Security test: `test_copilot_scratch_confinement` |
| **AC-F13-13** | Profile Variants | Versioned routing profiles (`bench/profiles/*.yaml`) configure reproducible baseline and variant bundles (e.g. Sonnet dev vs Opus review). | Contract test: `test_routing_profiles_schema` |

---

## Part 2 — Architecture

### 2.1 System architecture and component design

```
+--------------------------------------------------------------------------+
|                     E40 Lifecycle Benchmark Harness                      |
+--------------------------------------------------------------------------+
                                     |
                                     v
+------------------------+   dispatches   +--------------------------------+
|  run-lifecycle.sh /    | -------------> | lifecycle-worker-adapter.sh    |
|  run-lifecycle-batch.sh|                +--------------------------------+
+------------------------+                               |
            ^                                            | launches with:
            | writes result.json                         | -C <scratch_root>
            | (control envelope +                        | --model <model>
            | provider measurements)                     | --output-format json
            |                                            | stdin < prompt
            v                                            v
+------------------------+                +--------------------------------+
| Stage Evidence (I-05)  |                | GitHub Copilot CLI Subprocess  |
| - bundle.json          |                | (runs inside scenario scratch) |
| - stages/<n>.json      |                +--------------------------------+
| - transcripts/<n>.txt  |                               |
+------------------------+                               | emits JSONL:
            ^                                            | - assistant.message
            | resolves slots via                         | - session.usage_checkpoint
            v                                            | - result
+---------------------------------------+                v
| bench/evidence/usage-mapping.yaml     | <--------------------------------+
| [github_copilot_cli]                  | parses stream & normalizes
+---------------------------------------+
```

### 2.2 Component changes and file inventory

| Component | File Path | Nature of Change | Description |
|---|---|---|---|
| **Worker Adapter** | `bench/scripts/lifecycle-worker-adapter.sh` | Modify | Add `copilot` provider branch in `command_for()`; implement JSONL stream parser in `decode_envelope()` and `provider_measurements()`; extract tokens, durations, model IDs, and credit nano-units; honestly omit USD cost. |
| **Usage Mapping** | `bench/evidence/usage-mapping.yaml` | Modify | Add `github_copilot_cli` provider block with real-capture bindings for tokens, durations, model IDs, turns, and session ID; leave `total_cost` unmapped. |
| **Lifecycle Runner** | `bench/scripts/run-lifecycle.sh` | Modify | Update `mapped_provider(response)` to recognize `copilot` / `github` and return `github_copilot_cli`; ensure `resolve_usage` records `usage_slot_unavailable` for missing `total_cost`. |
| **Benchmark Library** | `bench/scripts/lib/e40_benchmark.py` | Modify | Update `preflight_provider_command_finding()` to accept route-aware adapters without requiring `runtime.provider_command`. |
| **Usage Canary** | `bench/scripts/canary-copilot-usagemapping.sh` | Create | Script asserting that real captured Copilot JSONL transcripts resolve all declared `github_copilot_cli` slots against `usage-mapping.yaml`. |
| **Routing Profiles** | `bench/profiles/copilot-default.yaml`<br>`bench/profiles/copilot-variant-review.yaml` | Create | Versioned model routing profile definitions establishing reproducible model assignment baselines. |
| **Adapter Tests** | `bench/scripts/tests/tc120_copilot_adapter_test.py` | Create | Unit test suite verifying Copilot command line construction, JSONL parsing, envelope decoding, and measurement extraction. |

### 2.3 Copilot execution route and command invocation

#### 2.3.1 Provider routing detection
In `bench/scripts/lifecycle-worker-adapter.sh`, `command_for(args, request)` detects the requested provider:

```python
COPILOT_PROVIDERS = {"copilot", "github-copilot", "github_copilot", "github"}

def is_copilot_provider(provider):
    return str(provider).lower() in COPILOT_PROVIDERS
```

#### 2.3.2 Command assembly
When `request["provider"]` matches a Copilot provider:

```python
elif is_copilot_provider(request.get("provider")):
    model = request.get("model")
    if not isinstance(model, str) or not model:
        raise AdapterError("copilot provider requires a non-empty model")
    
    # Base executable: command override or system PATH
    copilot_bin = args.provider_command or "copilot"
    command = [
        copilot_bin,
        "--output-format", "json",
        "--stream", "off",
        "--model", model,
        "--disallow-temp-dir",
        "--no-custom-instructions",
        "--no-ask-user",
        "--allow-all-tools",
        "--disable-builtin-mcps",
        "--no-color",
    ]
    
    # Scratch directory containment
    scratch_root = os.environ.get("LIFECYCLE_SCRATCH_ROOT") or request.get("scratch_root")
    if scratch_root:
        command.extend(["-C", str(scratch_root)])
        
    # Reasoning effort mapping
    effort = request.get("effort")
    if isinstance(effort, str) and effort and effort != "unavailable":
        effort_normalized = effort.lower()
        if effort_normalized in {"none", "minimal", "low", "medium", "high", "xhigh", "max"}:
            command.extend(["--reasoning-effort", effort_normalized])
```

#### 2.3.3 Prompt transmission
Prompt transmission uses piped standard input (`subprocess.run(command, input=prompt.encode("utf-8"), ...)`). This avoids operating system limits on command-line argument lengths and prevents shell expansion issues.

### 2.4 JSONL parsing and worker control envelope mapping

Copilot CLI with `--output-format json` emits a line-delimited stream of JSON events.

#### 2.4.1 Relevant event types

| Event Type | Key Fields | Purpose |
|---|---|---|
| `assistant.message` | `data.content`<br>`data.model`<br>`data.turnId` | Contains the assistant's response text / structured JSON. Used for control envelope and turn tracking. |
| `session.usage_checkpoint` | `data.promptCacheBreakState[].models[<model>].prompt_tokens`<br>`cache_read`<br>`cache_write`<br>`data.totalNanoAiu`<br>`data.totalPremiumRequests` | Detailed token metrics, caching metrics, and credit usage. |
| `result` | `sessionId`<br>`usage.totalApiDurationMs`<br>`usage.sessionDurationMs`<br>`exitCode` | Session ID, durations, and process completion status. |

#### 2.4.2 Control envelope decoding algorithm
`decode_envelope(raw)` is updated:
1. Attempt `json.loads(raw)`. If successful and contains `kind`, return it.
2. If `raw` contains multiple lines, scan lines looking for JSON events.
3. For each JSON event where `type == "assistant.message"`:
   - Extract `content = event.get("data", {}).get("content", "")`.
   - Attempt `json.loads(content)`. If successful and contains a dictionary with `kind` in `{"final", "question"}`: return it.
   - If `content` contains a wrapped object (e.g. `structured_output` or `result`), unwrap it.
   - If `content` is prose, apply regex `(?m)^RECOMMENDED OUTCOME:\s*(\S+)`. If matched, return `{"kind": "final", "recommended_outcome": match.group(1), "evidence": []}`.
4. If no `assistant.message` yields a control envelope, check for fallback regex on the entire raw output.
5. If still not found, raise `AdapterError("provider output does not contain a control envelope")`.

### 2.5 Model identity pinning and normalization

#### 2.5.1 Account model catalog
Live canary testing on the target host confirmed the availability of models across families:
- Anthropic: `claude-sonnet-5`, `claude-opus-4.7`, `claude-haiku-4.5`
- OpenAI: `gpt-5.4`, `gpt-5-mini`, `gpt-5.3-codex`
- Google: `gemini-3.8-flash`, `gemini-3.7-flash`

#### 2.5.2 Model pinning discipline
- Workflow definitions must specify concrete model identifiers (e.g., `claude-sonnet-5`), never generic aliases like `auto`.
- `provider_measurements()` extracts the actual model reported by Copilot in `assistant.message.data.model` and `session.usage_checkpoint`.
- The reported model ID is stored in `model_ids` as a sorted list.
- In `run-lifecycle.sh` and `e40_benchmark.py`, comparison identity verifies that baseline and variant runs use their declared models without silent substitution.

### 2.6 Usage and credit mapping (Never fabricate USD)

#### 2.6.1 Copilot credit reporting
Copilot CLI reports:
- `totalNanoAiu`: Integer representing billionths of an AI credit (1 credit = 1,000,000,000 nano-AIUs).
- `totalPremiumRequests`: Integer count of premium model requests.

Copilot CLI DOES NOT report USD currency amounts.

#### 2.6.2 Evidence integrity rule
Converting `totalNanoAiu` to USD using a static multiplier (e.g. assuming $0.05 per credit) or defaulting missing cost to 0.0 is strictly prohibited:
1. Synthetic conversion implies financial truth that Copilot does not guarantee.
2. Defaulting to 0.0 would corrupt cross-variant comparisons, making Copilot routes appear falsely "free" compared to direct Claude or OpenAI routes.
3. Therefore: `cost_usd` and `total_cost_usd` MUST be omitted. Downstream stage snapshots record `total_cost` as `usage_slot_unavailable`.

#### 2.6.3 `usage-mapping.yaml` definition

```yaml
providers:
  github_copilot_cli:
    status: mapped
    slots:
      input_tokens:
        envelope_path: usage.input_tokens
        verification_tier: real_capture
      cache_read_input_tokens:
        envelope_path: usage.cache_read_input_tokens
        verification_tier: real_capture
      cache_creation_input_tokens:
        envelope_path: usage.cache_creation_input_tokens
        verification_tier: real_capture
      model_ids:
        envelope_path: "sorted(modelUsage keys)"
        verification_tier: real_capture
      api_active_duration_ms:
        envelope_path: duration_api_ms
        verification_tier: real_capture
      turn_count:
        envelope_path: num_turns
        verification_tier: real_capture
      provider_session_id:
        envelope_path: session_id
        verification_tier: real_capture
```

`verified_from` provenance metadata will be populated from the real canary capture transcript.

### 2.7 Sandbox containment and scratch root isolation

To preserve E40's three-root isolation policy (`agent_fixture_checkout`, `scratch_shark_project`, `evaluator_only`), Copilot CLI worker execution must be strictly isolated:

1. **Working directory**: Always set to the scenario scratch directory using `-C <scratch_root>`.
2. **Temp directory restriction**: `--disallow-temp-dir` prevents tool execution from creating or accessing files under `/tmp` or system temporary directories.
3. **Instruction suppression**: `--no-custom-instructions` prevents loading instructions from parent repositories or home directory configurations.
4. **Interactive suppression**: `--no-ask-user` ensures the model runs fully autonomously without hanging waiting for terminal input.
5. **Tool containment**: All file edits and bash executions performed by Copilot tools are executed relative to `<scratch_root>`.

### 2.8 Model routing profiles schema and definitions

A model routing profile specifies the model and effort configuration for each workflow step in a benchmark run.

#### 2.8.1 Schema (`bench/profiles/schema.yaml`)

```yaml
schema_version: "1.0"
profile_id: string
description: string
provider: string
routes:
  <step_name>:
    model: string
    effort: string # none | minimal | low | medium | high | xhigh | max | unavailable
```

#### 2.8.2 Shipped profile definitions
1. `bench/profiles/copilot-balanced.yaml`:
   - `discovery`: `gemini-3.8-flash`, effort: `medium`
   - `specification`: `claude-sonnet-5`, effort: `medium`
   - `planning`: `gemini-3.8-flash`, effort: `minimal`
   - `development`: `claude-sonnet-5`, effort: `high`
   - `review`: `claude-sonnet-5`, effort: `high`
   - `qa`: `gemini-3.8-flash`, effort: `low`
2. `bench/profiles/copilot-variant-review.yaml`:
   - Identical to `copilot-balanced` except `review` is routed to `gpt-5.4` with effort `xhigh`.

These profiles enable comparing the impact of model choice on review yield and quality without modifying the test harness code.

---

## Part 3 — Cross-feature interactions

### 3.1 Declared interactions

| ID | Seam | Producer | Consumer | Contract | Impact of E40-F13 |
|---|---|---|---|---|---|
| **I-05** | Stage evidence and isolation | E40-F06 / E40-F12 | E40-F08 / E40-F09 / E40-F10 | `bench/evidence/i05-schema.yaml` | Stage snapshots record Copilot token usage and durations; `total_cost` records `usage_slot_unavailable`; scratch isolation verified. |
| **I-07** | Lifecycle run record | E40-F08 | E40-F09 / E40-F10 | `bench/runs/i07-schema.yaml` | Dispatches record `provider: "copilot"` and the pinned `model`; worker results capture bounded control envelopes. |
| **I-08** | Evaluation and comparison identity | E40-F09 | E40-F10 | `bench/evaluation/i08-schema.yaml` | `workflow_routing_identity()` generates matching `routing_digest` and `model_digest` ensuring valid pair comparisons. |

---

## Part 4 — Cross-epic integrations

| ID | Producing Epic | Consumer | Integration Purpose | E40-F13 Responsibility |
|---|---|---|---|---|
| **X-09** | E27 (Status Viewer) | E40 (Shark Bench) | Provider usage field mapping | Extend `bench/evidence/usage-mapping.yaml` with `github_copilot_cli` block using real-captured envelope paths. |
| **X-11** | E38 (Shark Attack) | E40 (Shark Bench) | Provider-neutral execution loop | Execute step dispatches specifying `provider: copilot` and retain lease/claim authority in Shark. |
| **X-12** | E32 (Single Artifact) | E40 (Shark Bench) | Installed content identity | Include workflow routing YAMLs in content digest calculation without corrupting existing bundle hashes. |

---

## Part 5 — Durable unresolved decisions

### Q012: Copilot token metric granularity across streaming turns
- **Context**: Copilot CLI emits token checkpoints per turn and per model call in `session.usage_checkpoint`. In multi-turn tool calling sessions, multiple checkpoints may be emitted.
- **Options**:
  - *(a) Cumulative checkpoint (Adopted)*: Read the final `session.usage_checkpoint` event, which reports cumulative `prompt_tokens`, `cache_read`, and `cache_write` across the session.
  - *(b) Incremental sum*: Sum incremental tokens reported across all intermediate checkpoints.
- **Decision**: Option (a). The final checkpoint represents the provider's authoritative cumulative accounting for the session.

---

## Part 6 — Test and verification strategy

### 6.1 Test matrix

1. **Unit Tests (`bench/scripts/tests/tc120_copilot_adapter_test.py`)**:
   - `test_copilot_command_construction`: Verify complete argv with flags, model, effort, scratch path, and safety flags.
   - `test_copilot_jsonl_decode_envelope`: Verify decoding of multi-line JSONL to extract control envelope from `assistant.message`.
   - `test_copilot_measurements_extraction`: Verify token counts, durations, session ID, and model IDs.
   - `test_copilot_cost_never_fabricated`: Verify that `total_cost_usd` is not in output and credits are stored as raw measurements.
2. **Go Contract Test (`tests/contracts/e40_i05_stage_evidence_contract_test.go`)**:
   - Re-run `TestTC042_I05StageEvidenceContract` to confirm `usage-mapping.yaml` extension satisfies schema and identity rules.
3. **Usage Mapping Canary (`bench/scripts/canary-copilot-usagemapping.sh`)**:
   - Verify that real captured Copilot JSONL logs resolve all mapped slots without drift.
4. **Preflight Readiness Test (`bench/scripts/tests/tc103_preflight_fail_closed_test.sh`)**:
   - Verify preflight passes when using the route-aware adapter with Copilot routes.
5. **Full Quality Gate**:
   - Run `make fmt && make lint && make test` to guarantee repository health.
