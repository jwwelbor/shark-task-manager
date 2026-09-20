---
research_schema: 2
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Copilot-backed model routing for E40 lifecycle benchmarks

## Scope

E40-F13 enables E40 lifecycle benchmarks to execute the provider and model
selected by each Shark workflow route through GitHub Copilot CLI. The existing
lifecycle worker adapter (`bench/scripts/lifecycle-worker-adapter.sh`) supports
Claude/Codex-oriented command generation, while the global provider-command
override (`LIFECYCLE_PROVIDER_COMMAND`) forces a single harness-wide command
that cannot model per-route model selection (e.g. Sonnet for development and
Opus or GPT-5 for code review). This feature adds: (1) a provider-aware Copilot
execution path; (2) normalization of Copilot CLI's JSONL event stream into
E40's bounded control envelope and provider usage envelope; (3) honest recording
of usage without inventing synthetic USD cost when Copilot only exposes credits
or token metrics; (4) pinned, retained Copilot model identity; (5) strict tool
and filesystem isolation to the scenario scratch root; and (6) versioned,
profile-owned model routing that integrates with E40's immutable run manifest
and routing digest verification.

Out of scope: modifying Shark core workflow transitions, claims, or leases;
re-implementing the I-05 or I-07 schemas; creating alternate benchmark
evaluators; or modifying the admitted scenario packages themselves.

Vocabulary: **Copilot JSONL stream** (the multi-line event stream emitted by
`copilot --output-format json`), **control envelope** (the bounded `{kind,
recommended_outcome, evidence}` dictionary consumed by `run-lifecycle.sh`),
**usage envelope** (the raw provider measurement dictionary consumed by
`usage-mapping.yaml`), **Nano-AIU** (`totalNanoAiu`, Copilot's billionth-of-a-
credit AI billing unit), **route-aware dispatch** (selecting model and flags
per workflow step rather than through one global process string), and **routing
digest** (the canonical hash of per-step provider, model, and effort routes
bound into the benchmark manifest).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F13-copilot-backed-model-routing-for-e40-lifecycle-ben/feature.md` defines the Copilot CLI execution path, JSONL normalization, credit-versus-cost honesty, scratch-root sandboxing, and versioned routing profile vocabulary.
- [x] `affected_implementation_or_contract` — Evidence: `bench/scripts/lifecycle-worker-adapter.sh` (lines 144-175 `command_for()`, lines 201-236 `decode_envelope()`, lines 241-305 `provider_measurements()`), `bench/evidence/usage-mapping.yaml` (providers block and required slots), `bench/scripts/lib/e40_benchmark.py` (lines 129-175 `workflow_routing_identity()`, lines 2208-2235 `preflight_provider_command_finding()`, lines 4401-4406 `provider_routing` identity assertions), and `bench/runs/i07-schema.yaml` (`/dispatches[]/response/{provider,model}` and `/stages[]/usage`).
- [x] `related_work` — Evidence: parent epic research report `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (foundation boundaries); bug report `docs/plan/bugs/B064.md` (prior caller-path and usage integrity repair); feature `docs/plan/E38-shark-attack-team-orchestration/E38-F10-cross-provider-adapter-conformance/feature.md` (cross-provider adapter conformance history); and sibling reports `E40-F10-operator-workflow-and-retained-lifecycle-baseline/research-report.md`, `E40-F11-six-family-lifecycle-benchmark-readiness-and-cover/research-report.md`, and `E40-F12-live-i-05-evidence-bundle-producer-for-lifecycle-v/research-report.md`.
- [x] `pattern_contract` — Evidence: `bench/evidence/usage-mapping.yaml` (`anthropic_claude_cli` mapped slots and `openai_codex_cli` unmapped pattern); `bench/scripts/canary-usagemapping.sh` (canary verification against real transcripts); `tests/contracts/e40_i05_stage_evidence_contract_test.go` (TC-042 usage mapping self-check); and `bench/scripts/run-lifecycle.sh` (`mapped_provider()` and `resolve_usage()` envelope lookup).
- [x] `dependency_impact` — Evidence: `bench/scripts/lib/e40_benchmark.py` (`runtime_readiness`, `preflight_provider_command_finding`, and manifest execution inputs), `bench/scripts/run-lifecycle-batch.sh` (pair execution and adapter invocation), and `bench/scripts/tests/tc103_preflight_fail_closed_test.sh` (preflight P4 provider readiness checks).
- [x] `cross_boundary_risks` — Evidence: Copilot CLI v1.0.86 stdout format divergence (Copilot outputs a continuous multi-line JSONL event stream while `lifecycle-worker-adapter.sh` expects a single JSON document or single-line regex, causing `json.loads` to crash and reversing scans to hit `type: "result"` rather than `assistant.message`); lack of USD cost output in Copilot CLI (`totalNanoAiu` and `premiumRequests` only, where synthesizing USD or defaulting to zero would corrupt comparison identity); and tool permissions and filesystem escapes (Copilot requires explicit `--disallow-temp-dir`, `-C <scratch_root>`, and tool deny flags to preserve E40's three-root isolation).
- [x] `alternatives` — Evidence: `bench/scripts/lifecycle-worker-adapter.sh` lines 145-155 (`LIFECYCLE_PROVIDER_COMMAND` global JSON argv override); alternative 1 (keep single global override and shell-script wrapper) rejected because it hides per-route model identity from the run manifest and defeats variant comparison; alternative 2 (fabricate USD cost from AI credits via fixed multiplier) rejected because it violates evidence integrity rules established in B064; and alternative 3 (parse Copilot interactive text output) rejected because JSONL provides deterministic message and token checkpoint structures.

## Capability map

| Capability | Brownfield evidence | Decision | E40-F13 responsibility |
|---|---|---|---|
| Copilot CLI Execution Adapter | `bench/scripts/lifecycle-worker-adapter.sh:144-175` (`command_for` supports Claude and Codex; no Copilot branch) | NEW | Add a native `copilot` provider execution path in `lifecycle-worker-adapter.sh` supporting `-p` or piped prompt, `-C <scratch_dir>`, `--model <model>`, `--disallow-temp-dir`, `--no-custom-instructions`, and `--output-format json`. |
| Copilot JSONL Normalization & Usage Extraction | Live canary capture; `lifecycle-worker-adapter.sh:201-305` (`decode_envelope` and `provider_measurements` fail on JSONL) | NEW | Implement line-by-line JSONL parsing in the adapter: extract assistant outcome and evidence from `assistant.message.data.content`, tokens from `session.usage_checkpoint`, durations from `result.usage`, and model from `assistant.message.data.model`. |
| Copilot Usage Mapping | `bench/evidence/usage-mapping.yaml` (defines `anthropic_claude_cli` mapped and `openai_codex_cli` unmapped; no Copilot block) | EXTEND | Add `github_copilot_cli` provider block to `usage-mapping.yaml` binding real-captured envelope paths for tokens, cache read/write, durations, turn count, session ID, and model IDs, while leaving `total_cost` unmapped or unavailable. |
| Per-Route Model Routing in Benchmark Profiles | `bench/scripts/lib/e40_benchmark.py:129-175` (`workflow_routing_identity`), lines 2208-2235 (`preflight_provider_command_finding`) | EXTEND | Support per-route provider and model dispatch from Shark workflow steps (`response["provider"]` and `response["model"]`); allow profile-backed model configuration; update preflight readiness checks so Copilot routes report ready without requiring a global `provider_command` override. |
| Three-Root Scenario Isolation | `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (harness-owned isolation); `bench/scripts/run-lifecycle.sh` | REUSE | Restrict Copilot execution to the isolated scenario `scratch_root` using `-C`, `--disallow-temp-dir`, and appropriate tool restrictions, ensuring worker tools cannot mutate parent repository files. |
| Lifecycle Loop & State Authority | `bench/scripts/run-lifecycle.sh:2540-2642`; `B064.md` fix notes | REUSE | Retain Shark as the sole authority for claim, heartbeat, transition, and release. Provider session metadata is retained as observational evidence only. |
| Global `LIFECYCLE_PROVIDER_COMMAND` Override | `bench/scripts/lifecycle-worker-adapter.sh:147-155` | CONTRADICTS | A static global command override forces identical execution across all steps, contradicting per-route model selection; per-route dispatch must take precedence when provider and model are declared. |

## Findings

1. **GitHub Copilot CLI is installed, authenticated, and verified locally.**
   `copilot --version` confirms GitHub Copilot CLI 1.0.86 at
   `/home/jwwel/.nvm/versions/node/v20.20.0/bin/copilot`. Piped stdin execution
   via `copilot --output-format json` functions cleanly in non-interactive
   mode without requiring `-p -`.

2. **Copilot CLI outputs multi-line JSONL, breaking the existing adapter.**
   `lifecycle-worker-adapter.sh` assumes a single JSON document (Claude CLI
   format) and attempts `json.loads(raw)` in `provider_measurements()`. On
   Copilot's JSONL stream, `json.loads` raises `JSONDecodeError`, causing all
   token, duration, and session measurements to be dropped. Furthermore,
   `decode_envelope()` scans lines in reverse and encounters the final
   `{"type":"result", ...}` line, which lacks `kind` or `recommended_outcome`,
   failing to extract the assistant's actual response.

3. **The assistant's control envelope and outcome are inside `assistant.message`.**
   In Copilot's JSONL output, the worker's text response is carried in the
   event `{"type":"assistant.message", "data":{"content":"...", "model":"..."}}`.
   The adapter must extract `data.content` and decode the control envelope or
   `RECOMMENDED OUTCOME: <outcome>` from that payload.

4. **Token usage and duration are spread across specific event types.**
   - Token metrics are emitted in `session.usage_checkpoint` events under
     `data.promptCacheBreakState.main.models[<model>]` (`prompt_tokens`,
     `cache_read`, `cache_write`) and `data.totalNanoAiu` /
     `data.totalPremiumRequests`.
   - Duration metrics are emitted in `result` events under
     `data.usage.totalApiDurationMs` and `sessionDurationMs`.
   - Active model identity is emitted in `assistant.message.data.model`,
     `model.call_start.data.model`, and `session.tools_updated.data.model`.

5. **Copilot CLI does not output USD cost, and cost must not be fabricated.**
   Copilot reports `totalNanoAiu` (AI credit nano-units) and `premiumRequests`,
   never `total_cost_usd`. B064 and E40 evidence integrity principles mandate
   that unavailable fields must never be fabricated, guessed, or defaulted to
   zero. USD cost must be honestly omitted or marked unavailable. In
   `run-lifecycle.sh` and `usage-mapping.yaml`, an unavailable cost slot
   records a clean `usage_slot_unavailable` entry rather than a corrupted
   financial calculation.

6. **Account model catalog verified with real live canary invocations.**
   Live canary execution verified models including `claude-sonnet-5`,
   `gemini-3.8-flash`, and `gpt-5.4` on this host. Model identity must be
   explicitly pinned in benchmark profiles and captured in `model_ids` to
   prevent unrecorded model drift.

7. **Preflight currently blocks on unconfigured provider command and adapter.**
   In `bench/scripts/lib/e40_benchmark.py:2208-2235`, `preflight_provider_command_finding()`
   flags `runtime.provider_command is not configured` when an adapter is set
   without a global command. For Copilot routing, preflight must recognize
   a provider-aware adapter capable of resolving the workflow's configured
   routes.

8. **Filesystem and tool sandboxing is required.**
   Copilot CLI supports tools including bash execution and file editing. To
   preserve E40's three-root isolation, Copilot must be invoked with `-C
   <scratch_root>`, `--disallow-temp-dir`, `--no-custom-instructions`,
   `--no-ask-user`, and restricted tool parameters to prevent writes outside
   the scenario scratch tree.

## Decisions

1. **Implement native Copilot support directly in `lifecycle-worker-adapter.sh`.**
   Add a `copilot` branch in `command_for()` that constructs Copilot CLI
   arguments from `request["model"]` and `request["effort"]`, rather than
   forcing operators to author external wrapper scripts.
2. **Normalize Copilot JSONL via dedicated stream parsing.**
   Add JSONL decoding logic in `lifecycle-worker-adapter.sh` that iterates over
   all lines, extracts `assistant.message` for control envelopes, aggregates
   tokens from `session.usage_checkpoint`, and extracts durations and session
   ID from `result`.
3. **Record unavailable USD cost honestly without synthesis.**
   Omit `cost_usd` when running Copilot routes; do not multiply AI credits by a
   synthetic conversion rate or substitute zero. Allow downstream aggregation
   to report cost as unavailable while preserving token and credit metrics.
4. **Extend `bench/evidence/usage-mapping.yaml` with a `github_copilot_cli` block.**
   Map verified envelope slots (`input_tokens`, `cache_read_input_tokens`,
   `cache_creation_input_tokens`, `model_ids`, `api_active_duration_ms`,
   `turn_count`, `provider_session_id`) based on real captured Copilot
   envelopes, while leaving `total_cost` unmapped.
5. **Prioritize per-route workflow dispatch over global overrides.**
   When `request["provider"]` and `request["model"]` are specified by the
   dispatched Shark route, use them directly; treat `LIFECYCLE_PROVIDER_COMMAND`
   only as a fallback for unmapped custom providers.
6. **Enforce scratch-root confinement via Copilot CLI flags.**
   Always pass `-C <scratch_root>` and `--disallow-temp-dir` when launching
   Copilot workers to preserve harness-owned isolation.
7. **Adopt COMPLEX rigor.**
   Because this feature bridges external CLI streaming behavior, contract
   validation, usage mapping, and multi-model routing across Go and Python
   components, classify the research as complex and route to pass.

RECOMMENDED OUTCOME: pass

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F13-copilot-backed-model-routing-for-e40-lifecycle-ben/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`
- `docs/plan/bugs/B064.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F10-cross-provider-adapter-conformance/feature.md`
- `bench/scripts/lifecycle-worker-adapter.sh`
- `bench/scripts/run-lifecycle.sh`
- `bench/scripts/run-lifecycle-batch.sh`
- `bench/scripts/lib/e40_benchmark.py`
- `bench/evidence/usage-mapping.yaml`
- `bench/scripts/canary-usagemapping.sh`
- `bench/runs/i07-schema.yaml`
- `bench/scripts/tests/tc103_preflight_fail_closed_test.sh`
- `tests/contracts/e40_i05_stage_evidence_contract_test.go`
- Live `copilot --version` (v1.0.86) and `--output-format json` non-interactive test executions
- `internal/sharkdata/default_data/research/recipes.yaml`
