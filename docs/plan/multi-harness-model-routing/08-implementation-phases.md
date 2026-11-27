# Implementation phases and Shark backlog

The proposed sequence moves effort propagation and basic diagnostics into adapter qualification. This exposes incorrect configuration at launch before introducing fallback or independent review. The estimate is **8–13 engineering weeks** for one engineer familiar with Shark, including review and offline testing time. External qualification, entitlements and approval lead times can extend elapsed delivery. Phase ranges assume a bounded initial provider and CLI mode matrix.

Shark tracks the epic as `E41`, all four features and the three Phase 1 tasks. In Phases 2–4, the task rows are proposed work items for later decomposition. Task acceptance IDs refer to the [test plan](09-test-plan.md#acceptance-matrix).

## Phase 1 · Route resolution and compatibility

**P0 · 1.5–2.5 weeks.** Deliver strict versioned route/profile parsing, execution intent and a pure resolver. Add an offline explain surface and additive keyed execution intent. Keep the legacy launch path and classify unsupported strict launch modes explicitly while Phase 2 qualifies adapters. This phase is independently useful for assessing every route without changing workflow execution.

| Shark reference | Work item and code areas | Dependencies and completion |
|---|---|---|
| `E41-F01` | **Resolve execution routes from workflow intent** | R01–R03. Existing workflow configs load and dry-run as before. |
| `T-E41-F01-001` | Add optional step complexity and named pins in `internal/config/workflow`, action compilation and `internal/config/action`. Reserve the review field for Phase 3. | Start here. Canonical YAML, legacy workflows and invalid fields meet R01. |
| `T-E41-F01-002` | Load versioned YAML and project/global profile references in `internal/config`. Implement precedence, legacy normalization and complete-binding checks in a consumer-independent routing package. | F01-001. R01–R02 pass with injected capability fixtures and stable diagnostics. |
| `T-E41-F01-003` | Wire dry-run/explain and keyed intent through `internal/cli/commands/{run,next}.go` and shared CLI service composition. | F01-002. R03 proves stable additive output and zero worker or claim effects. |

**Migration and risk.** Existing `provider`, `model`, `effort` and `--harness*` keep their current legacy semantics. A strict route must return unsupported until the executable path is wired. Exercise master-index, bundle overrides and canonical aliases before enabling any profile. Schema interpretations and conflicting pins are the primary rollout risk.

## Phase 2 · Dispatch through three harness adapters

**P0 for parity, P1 for first Copilot mode · 2–3.5 weeks.** Deliver typed registry, approved harness descriptors, effective invocation, effort mappings and bounded evidence. Start the durable single-attempt record here to avoid qualifying route-aware results without provenance. Use existing process supervision and fixtures, with synthetic CLI executables in automated tests.

| Shark reference | Work item and code areas | Dependencies and completion |
|---|---|---|
| `E41-F02` | **Execute routed work through Claude, Codex and Copilot** | F01. R04–R07, including restricted review execution and compatible output handling. |
| `F02-001` | Add typed invocation and harness registry around `internal/runner/dispatcher.go`. Wire core/child controllers, approved cwd and capability interfaces. | F01. R04 and the supervisor cases of R07 pass. |
| `F02-002` | Adapt Claude/Codex builders, normalized effort, provider binding and correlated execution metadata. Parse bounded transport using existing worker-control contracts. | F02-001. R04–R06 pass against legacy goldens and qualified CLI canaries. |
| `F02-003` | Persist versioned selected/observed route and basic single-attempt evidence. Extend `StageLog`, logging and keyed output without exposing secrets. | F02-001 and existing gate identity. R07 covers journal binding, permissions and redaction. |
| `F02-004` | Qualify Copilot non-interactive mode in `internal/runner`. Reuse E40 JSONL fixture cases. Enforce mode-specific tool policy and reject unsupported effort. | F02-001 through 003. R04–R07 pass for Copilot. First qualification may cover only read-only work. |

**Migration and risk.** Roll out pinned Claude/Codex routes first, then enable Copilot per qualified mode. Keep legacy command and parser paths under regression coverage. Protocol changes, hidden configuration, account claims and different permission controls can block a binding. Capability checks and narrow canaries gate production use.

## Phase 3 · Independent review and safe fallback

**P1 · 2.5–4 weeks.** Deliver strict candidate-based review, ordered alternate routes and lease-bound recovery. Ship prelaunch fallback first. Only enable timeout/process retry where replay safety is proven.

| Shark reference | Work item and code areas | Dependencies and completion |
|---|---|---|
| `E41-F03` | **Enforce independent review and recover safe execution failures** | F02. R08–R10 and relevant gate regressions pass. |
| `F03-001` | Declare review intent and candidate author sources in workflow schema. Extend gate/worker-control evidence with producer attempt references and policy snapshot. | F02-003. R08 proves author binding and reviewer eligibility, including aggregate candidates. |
| `F03-002` | Integrate fresh reviewer selection and observed admission checks in runner gate ingestion, keyed host contract and UAT/review result paths. | F03-001. R08 covers pass, mismatch, missing metadata and manual disposition. |
| `F03-003` | Add fallback policy/config validation, typed error classification, deterministic budget evaluation and eligibility checks reusable by Rider inline fallback. | F02. R09 covers safe prelaunch alternate, unavailable provider and exhausted budget. |
| `F03-004` | Implement durable attempt recovery, process retirement, lease/step fencing and gate-sidecar reconciliation in runner and gate persistence. | F03-003 and F02-003. R10 proves crash safety across launch and apply boundaries. |

**Migration and risk.** Existing gates opt in individually. Require new candidate evidence for in-flight legacy authors before enabling strict review on their candidates, or use the existing authorized manual workflow. Mutating retries are the highest-risk change. Keep external tool access within an enforceable replay-safety boundary.

## Phase 4 · E40 and operations

**P1 · 2–3 weeks.** Deliver profile-only E40 runs, unified diagnostics, migration tooling and documented release support. Finish telemetry integration and snapshot provenance across callers.

| Shark reference | Work item and code areas | Dependencies and completion |
|---|---|---|
| `E41-F04` | **Operate and benchmark reusable routing profiles** | F02 plus F03 for strict review/fallback comparison lanes. R11–R13 pass. |
| `F04-001` | Ship `shark routing check`, strict local/offline capability validation and documented exit/JSON contract. Consolidate E40 discovery snapshot format and use probes where available. | F02. R11 passes without inference calls. Basic validation can land during Phase 2. |
| `F04-002` | Convert E40 profiles, export one immutable plan and consume it in batch, worker, preflight and manifest verification. Update `bench/scripts/lib/e40_benchmark.py`, lifecycle scripts and retention comparisons. | F01, F02 and F04-001. R12 passes without altering workflow bytes or comparison gates. |
| `F04-003` | Adopt versioned decisions in Rider instructions/host adapters and E40. Add the final Rider inline attempt after external routes are exhausted. Preserve B077 and B083 validation and result admission. | F02 and F03 for full policy support. R09 and R12–R13 cover inline eligibility, requested/selected/observed evidence and mixed-version behavior. |
| `F04-004` | Add dry-run importer, migration guidance, compatibility matrices and release notes in `docs/guides` and CLI reference. Qualify rollback and operators' diagnostics. | F04-001 through 003. R13 and the epic definition of done pass. |

**Migration and risk.** Coordinate consumers before dropping materialization from the new benchmark path. Keep legacy execution and artifact readers available. Version skew and comparison identity drift can make a result unusable even when the worker succeeds.

## Migration and rollback

1. **Install compatibility release.** Upgrade the parser, next-response consumers and gate-sidecar readers first. Verify existing fixtures and unmodified installed workflow bundles. Existing empty provider, `anthropic`, `codex` and `openai` use their legacy dispatch map. Preserve original `--harness*` rendering and legacy ignored effort until an operator opts in.
2. **Inventory and qualify.** Use explain/check to list canonical stages, legacy overrides, implicit CLI defaults and available bindings. Export proposed profile assignments preserving models and converting effort only when supported. Flag ambiguity, invalid effort and unmapped auth. Compare final argv and output handling in isolated canaries before switching behavior.
3. **Opt in workflow by workflow.** Begin with pinned profile assignments matching qualified legacy behavior. Move ordinary assignments from workflow fields to the profile. Add complexity and remove those pins when the tier mapping matches the desired execution. Keep explicit exceptions as named pins. Enable independent review and fallback only after the relevant gates qualify.
4. **Migrate E40 profiles.** Import each `schema_version: 1.0` profile with its original provider and source digest, normalize the serving binding and rewrite stage intent into profile step assignments keyed by canonical family/workflow and step. Preserve effective effort and fallback policy explicitly. Keep the variant review assignment divergent. Compare the imported plan with the legacy materialized assignment for every affected family. Surface unresolved differences before capture.
5. **Switch benchmark capture.** Use explicit profile selection and freeze its plan before any run. Retain workflow/prompt hashes, profile digest, catalog, adapter/CLI versions, policy, scenario/repetition and requested/selected/observed route evidence. Assert that workers consumed the frozen assignments. Retained comparisons include both configuration and execution differences. Disable fallback for default controlled comparisons, or encode an explicit fallback policy and admit only the designated fallback comparison lane. Keep old artifacts interpretable and preserve evaluator isolation, spend ceilings and publication rules.
6. **Roll back at a run boundary.** Stop scheduling new routed work, reconcile or retire leased operations and park uncertain writes. Restore original workflow/config and prior reader/runner where supported, or use the compatibility lane. Resume only against a compatible persisted operation. Retain routed evidence and keep new signed-off artifacts accessible through versioned readers. Restore the legacy E40 materializer lane for new legacy captures if the profile migration fails validation, preserving already retained runs.
