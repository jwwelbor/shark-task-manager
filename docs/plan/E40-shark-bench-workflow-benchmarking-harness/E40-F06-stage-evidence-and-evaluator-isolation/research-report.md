---
research_schema: 2
entity_key: E40-F06
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Stage evidence and evaluator isolation

## Scope

E40-F06 defines **I-05**, the stage-evidence bundle and three-root isolation
contract that E40-F08, E40-F09, and E40-F10 consume read-only: the
agent-visible fixture checkout, the scratch Shark project, and an
evaluator-only root that never becomes visible during worker dispatch. Its
shape is already pinned by
`architecture.md#stage-evidence-and-isolation-contract` and the I-05 row in
`E40-interaction-map.md` — like sibling F05 for I-04, F06 implements a shape
the epic has already fixed, not an open design. F06 consumes **I-04**
(`bench/scenarios/scenarios.yaml` and its packages) read-only for stage
applicability, `evaluator_only` references, and `toolchain_identity` — per
the I-04 staged edge, F06's consumption slice is
`evaluator_only`/`toolchain_identity`/both stage-matrix halves
(`E40-interaction-map.md` lines 39-52). F06 also owns **X-09**: it must
verify the current E27-F15 provider-usage field mapping and fail closed on
missing comparison identity rather than inventing token, cost, model,
session, or timing fields (`E40-cross-epic-map.md` X-09 row; `architecture.md`
"Metric collection and artifact schema": "E40-F06 verifies the current
E27-F15 field mapping through X-09").

Per `architecture.md`'s component table, F06 is "New bench contract and
guards" — like F05, it does not touch `internal/`. Its output is new schema
and admission/dispatch-boundary tooling under `bench/`, plus a new Go-only
I-05 contract-test validator under `tests/contracts/` (the same pattern F05
established for I-04's `tests/contracts/e40_i04_scenario_contract_test.go`,
per `E40-interaction-map.md`'s I-05 staged edge: "F06's spec.md must name the
shared contract-test pointer at specification time, the same way F05's
spec.md named TC-030 for I-04"). I-05 is a `contract-only` predeclared
handoff to F08/F09/F10; F06's own task_review is where the shared
contract-test pointer must be named.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/feature.md` (Scope, Acceptance boundary, Contracts, Out of scope, and the 2026-08-13 value-attribution amendment) and `architecture.md#stage-evidence-and-isolation-contract` define I-05's three roots, the stage-snapshot field list, the non-overlapping time-ledger categories, the code-producing/review candidate-identity fields, the artifact producer/consumer record shape, and the admission/dispatch-boundary isolation checks.
- [x] `affected_implementation_or_contract` — Evidence: `internal/runner/controller.go:84-135` (`RunResult`/`StageLog` — the existing per-stage record F06 must extend without mutating: today it carries only status, action, agent_type, provider, `duration_ns`, exit_code, and a truncated output summary — no tokens, cost, candidate/tree digests, artifact records, or interval decomposition); `internal/runner/transcript.go:10-25` (the pinned `COMMAND:`/`EXIT:`/`DURATION:`/`---STDOUT---`/`---STDERR---` byte format that stage snapshots must reference, not reparse, for provider-envelope evidence); `internal/runner/dispatcher.go` (`DefaultDisallowedTools` — the existing worker self-advancement guard, the one local precedent for "block a category of worker action structurally" that the new evaluator-material dispatch-boundary check generalizes); `bench/scripts/canary-runsurface.sh` (the X-07 real-invocation preflight pattern: assert a pinned field/byte shape via one real `shark run`, never a re-derivation from memory — the concrete precedent I-05's own admission/dispatch-boundary checks should follow); `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml` (`evaluator_only.{reference_solution,oracle_tests,answer_keys}` and `input.agent_visible` — the exact I-04 fields F06's isolation checks must prove absent from/present in each root).
- [x] `related_work` — Evidence: `architecture.md` (Stage evidence and isolation contract, Lifecycle scenario package contract, Metric collection and artifact schema, Run lifecycle and isolation contract, Delivery boundaries); `E40-interaction-map.md` (I-04 row and staged edge — F06's consumption slice; I-05 row and staged edge — F06 as producer, F08/F09/F10 as `contract-only` consumers, and the TC-030-precedent instruction for F06's own spec.md); `E40-cross-epic-map.md` (X-09 row: producer E27-F15, owning feature E40-F06, status `assigned`, "E40-F06 must verify its current artifact and implementation state before depending on it"); parent epic `research-report.md` (Capability map "Claude JSON usage envelope" and "Provider usage and session metadata" rows, Decisions on E27-F15 field-name uncertainty); sibling `E40-F05-.../research-report.md` (Decision 4: I-04's evaluator/usage reference fields are kept opaque specifically because F06/X-09 had not yet verified the decoded shape — this feature is that verification); `shark get E40-F06/E27-F15/E22 --field status`; `git branch --all --contains` the E27-F15 commit; direct grep of `internal/runner/claude_dispatcher.go` on `main` for `modelUsage`/`num_turns`/`duration_api_ms`/`total_cost_usd`.
- [x] `pattern_contract` — Evidence: `internal/runner/dispatcher.go`'s `DefaultDisallowedTools` slice (`Bash(shark status advance*)` etc.) — the codebase's one existing structural-guard precedent for preventing a category of worker action, the shape the new "evaluator-only material must be structurally absent from both agent-visible roots" check should follow (a checkable allowlist/denylist, not a runtime convention); `bench/scripts/canary-runsurface.sh`'s real-invocation-over-re-derivation discipline (already cited above) as the pattern F06's own dispatch-boundary and admission checks must match: a real filesystem/process check against the actual roots, not a simulated assertion; `bench/corpus/corpus.yaml`'s `schema_version` field and I-04's own `schema_version: "1.0"` — the established versioned-schema precedent I-05 should follow in form.
- [x] `cross_boundary_risks` — Evidence: `architecture.md`'s Run lifecycle section ("one run spans two roots" — `--workdir` sets only the agent's cwd while Shark's own state, transcripts, and `run.log` live in the scratch project) versus I-05's three-root model, which adds a *third* root (evaluator-only) that must never be reachable from either of the first two during dispatch — the existing two-root split has no analog for this new boundary today, so F06's admission/dispatch-boundary checks are a genuinely new guard, not an extension of an existing one; the unmerged `E27-F15-cross-session-usage-tracking` branch's `claudeJSONResult` decode (`type/session_id/request_id/result/model/total_cost_usd/usage{...}`, no `modelUsage`/`num_turns`/`duration_api_ms`) versus `architecture.md`'s and the parent design's assumed field names — F06 inherits this exact unresolved gap directly, and `architecture.md` states F09 "rejects a record whose required usage or model identity is absent," making X-09's fail-closed posture load-bearing for downstream evaluation, not merely tidy; `RunResult`'s truncated `OutputSummary` (only for successful `spawn_agent` stages) versus I-05's requirement that every applicable stage, including failed and stopped ones, produce "exactly one addressable snapshot or a named missing-stage failure" — the current liveness/transcript surface does not guarantee a snapshot exists for every stage category today.
- [x] `alternatives` — Evidence: `architecture.md`'s explicit rejection of copying the Phase 1 provider-envelope parser "by assumption" (Metric collection section) in favor of F06 verifying E27-F15's field mapping directly — rejects reusing an unaudited decoder; `E40-F05` sibling report's Decision 4 (keep I-04's evaluator/usage reference fields opaque) confirms the alternative of typing decoded-usage fields directly into I-04 was already rejected upstream, so F06 cannot retrofit that shortcut; the existing two-root (`--workdir` checkout + scratch project) isolation model as the alternative of *not* adding a third evaluator-only root — rejected by the feature's explicit acceptance boundary ("A dispatch fails before provider spend if any evaluator-only file is visible to the worker"), which requires the isolation to be structural and checkable, not merely a convention that reference/oracle files live outside `input.agent_visible` in the I-04 package layout.

## Capability map

| Capability | Brownfield evidence | Decision | F06 responsibility |
|---|---|---|---|
| `RunResult`/`StageLog` per-stage record (X-07/X-08, `internal/runner/controller.go:84-135`) | `controller.go:84-135`; `bench/scripts/canary-runsurface.sh` (pins the same field set for its own preflight) | EXTEND (additively, without mutating the pinned shape) | I-05's stage snapshot must add tokens, cost, elapsed decomposition, candidate identity, and artifact records as new, separately-addressed evidence keyed to the same stage — it must not require changing `RunResult`/`StageLog`'s existing JSON keys, which X-07/X-08 and F02's canary already treat as frozen. |
| Transcript byte format (`internal/runner/transcript.go:10-25`) | `transcript.go` header comment; `canary-runsurface.sh`'s pinned-literal expected fields | REUSE (read-only source) | Stage snapshots reference transcript paths/digests as evidence of provider-active work; F06 does not redefine or reparse the transcript format, only points at it. |
| Worker structural guard (`DefaultDisallowedTools`, `internal/runner/dispatcher.go`) | `dispatcher.go` `DefaultDisallowedTools` slice | REUSE (pattern only, not code) | Model the new evaluator-material dispatch-boundary check as an analogous structural, checkable guard (a path allowlist/denylist verified before dispatch) rather than a documentation-only convention — the one existing local precedent for "block a category of worker capability by construction." |
| Real-invocation preflight discipline (`bench/scripts/canary-runsurface.sh`, X-07) | `canary-runsurface.sh` header comment and pinned literals | REUSE (pattern only) | F06's own admission and dispatch-boundary checks must assert against the real scratch project / real checkout / real evaluator root, matching this repo's established "assert the real shape, never re-derive from memory" discipline. |
| I-04 scenario package schema and versioning convention (`bench/scenarios/scenarios.yaml`, `package.yaml`'s `schema_version`) | `bench/scenarios/scenarios.yaml`; `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml`; `tests/contracts/e40_i04_scenario_contract_test.go` | EXTEND (consume read-only; mirror form for I-05) | Read `evaluator_only`, `toolchain_identity`, and both stage-matrix halves per the I-04 staged edge's named consumption slice; give I-05 its own `schema_version` field and its own Go contract-test validator under `tests/contracts/`, following I-04's precedent in form, not by importing I-04's package structs. |
| Provider-usage envelope decode (X-09, E27-F15) | `git branch --all --contains` the E27-F15 commit shows it only on branch `E27-F15-cross-session-usage-tracking`; `internal/runner/claude_dispatcher.go` on `main` has zero hits for `modelUsage`/`num_turns`/`duration_api_ms`/`total_cost_usd`; `shark get E27-F15 --field status` = `active`; `E40-cross-epic-map.md` X-09 row status `assigned`, owning feature E40-F06 | CONTRADICTS the design doc's assumed field names; EXTEND once verified | F06 is the owning feature for X-09 and must verify the *actual* decoded envelope shape on the E27-F15 branch (or its current state, whichever is authoritative at implementation time) before I-05 names any usage/cost/model field — architecture.md is explicit that F06, not F08, does this verification, and F09 fails a record closed when the identity is missing rather than F06 guessing it now. |
| I-04 staged edge and I-05 staged edge (contract-only handoffs) | `E40-interaction-map.md` I-04 and I-05 staged-edge sections | REUSE (process pattern) | F06's own spec.md must name the shared I-05 contract-test pointer at specification time, matching F05's spec.md naming TC-030 for I-04 — this is a process obligation the interaction map states explicitly, not an inference. |
| Two-root run isolation (`--workdir` checkout + scratch Shark project) | `architecture.md` "Run lifecycle and isolation contract" ("one run spans two roots") | EXTEND (add a third, evaluator-only root) | The existing split has no analog for evaluator-only material; F06 is introducing a genuinely new boundary, not relabeling or hardening an existing one, and must design the admission/dispatch-boundary check accordingly (no partial reuse of the two-root split's mechanics is available). |

## Findings

1. **`RunResult`/`StageLog` today captures none of I-05's required evidence beyond duration and identity.** `internal/runner/controller.go:84-135` shows the full pinned field set: `status`, `action`, `agent_type`, `provider`, `duration_ns`, `exit_code`, and a truncated `output_summary` populated only for successful `spawn_agent` stages. There is no token count, cost, candidate/tree digest, artifact record, or non-overlapping interval ledger anywhere in this struct. I-05's stage snapshot is therefore new evidence F06 must assemble from multiple sources (transcripts, scratch DB, post-run checks per `architecture.md`'s "Metric collection and artifact schema" table), not a restructuring of an existing record — confirming the feature scope's framing that this is new evidence capture, not a schema rename.

2. **X-09/E27-F15 remains genuinely unmerged and undecoded on `main` today**, unchanged from F05's earlier finding. `git branch --all --contains` the E27-F15 commit shows it only on `E27-F15-cross-session-usage-tracking`; `internal/runner/claude_dispatcher.go` on `main` has zero hits for `modelUsage`, `num_turns`, `duration_api_ms`, or `total_cost_usd`; `shark get E27-F15 --field status` returns `active`. `architecture.md`'s "Metric collection and artifact schema" section is explicit that F06 — not F08 or F09 — carries the obligation to verify E27-F15's *current* field mapping before any usage field is trusted. This is the single highest-leverage unresolved dependency for this feature: I-05's usage/cost fields cannot be finalized until this verification happens, and F09's fail-closed posture ("rejects a record whose required usage or model identity is absent") only works if F06 names the real fields rather than the design doc's assumed ones.

3. **The codebase already has one structural worker-restriction precedent, but it guards a different boundary than I-05 needs.** `internal/runner/dispatcher.go`'s `DefaultDisallowedTools` blocks a fixed list of `shark status advance*`-shaped commands from any dispatched agent, regardless of stage or scenario — a config-level denylist enforced by the dispatcher, not a filesystem-presence check. I-05's isolation requirement ("A dispatch fails before provider spend if any evaluator-only file is visible to the worker") is a *content/path* check against two live filesystem roots at dispatch time, not a command-string denylist. The pattern (structural, checkable, fails loud) is directly reusable; the mechanism (path presence in a root) is not — this is a new check, modeled on an existing discipline.

4. **`bench/scripts/canary-runsurface.sh` establishes the exact discipline I-05's own checks should follow: assert the real shape via a real invocation, never a re-derived assumption.** Its header comment states this explicitly for the X-07 `RunResult`/transcript shape and names the anti-pattern directly ("never a re-derivation of the shape from memory"). F06's admission and dispatch-boundary checks inherit the same obligation for the three-root isolation guarantee: they must inspect the actual scratch project and actual fixture checkout at dispatch time, not assert a policy document says the roots are separate.

5. **I-04's `evaluator_only` block gives F06 a concrete, already-populated example of what must never cross into the agent-visible root.** `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml` shows `evaluator_only: {reference_solution: evaluator/reference.patch, oracle_tests: [evaluator/test_recurring.py], answer_keys: []}` alongside `input.agent_visible: input/prompt.md` — the package layout already separates these paths structurally (an `evaluator/` subtree distinct from `input/`), which F05's own TC-030 (AC-004) already asserts holds for every package. F06's dispatch-boundary check can therefore build on an existing, tested path-separation convention in I-04 rather than inventing path semantics from nothing, though the *runtime* enforcement (proving the evaluator root is unreachable from the actual worker process, not just that the package YAML labels it correctly) remains new work.

6. **The interaction map assigns F06 a specific, named documentation obligation before implementation can proceed.** The I-05 staged edge states verbatim that "F06's spec.md must name the shared contract-test pointer at specification time, the same way F05's spec.md named TC-030 for I-04." This is a concrete, checkable requirement on F06's own downstream spec artifact, not general guidance — F06's spec.md needs an equivalent named Go contract test (e.g., under `tests/contracts/`) before its own task_review, mirroring the I-04 precedent exactly.

## Decisions

1. **Design I-05 as new, additively-composed evidence keyed to existing `RunResult`/`StageLog` stages, not as a replacement or mutation of that struct.** `controller.go`'s pinned field set is a X-07/X-08 contract F06 must not touch; the canary already asserts its exact shape.

2. **Treat X-09/E27-F15 verification as a blocking prerequisite for finalizing I-05's usage/cost field names, not a detail to fill in later.** `architecture.md` assigns this verification to F06 explicitly; guessing field names now would repeat the exact risk F05's Decision 4 already flagged and deliberately avoided by keeping I-04's references opaque.

3. **Model the evaluator-only dispatch-boundary check as a structural, checkable guard analogous to `DefaultDisallowedTools`, but implement it as a path-presence/absence check against the real roots, not a command denylist.** The pattern is reusable; the mechanism must be new because the boundary being enforced (filesystem content visibility) differs from the existing guard's boundary (command execution).

4. **Follow `canary-runsurface.sh`'s real-invocation discipline for I-05's own admission and dispatch-boundary checks.** Every isolation assertion must inspect the actual scratch project and actual fixture checkout, matching the one established local precedent for this class of contract test in `bench/`.

5. **Reuse I-04's existing `evaluator/` vs. `input/` path-separation convention as the starting structural signal for what the dispatch-boundary check must prove absent, but do not treat package-layout separation alone as sufficient enforcement.** TC-030 already verifies the *declared* separation; F06 must additionally verify *runtime* absence at dispatch time, which is new evidence, not a restatement of an existing check.

6. **Name I-05's shared contract-test pointer in F06's own spec.md at specification time, mirroring TC-030's precedent for I-04.** This is an explicit, stated obligation in `E40-interaction-map.md`'s I-05 staged edge, not an inferred convention.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (Scope and component design; Run lifecycle and isolation contract; Lifecycle scenario package contract; Stage evidence and isolation contract; Metric collection and artifact schema)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-04 row and staged edge; I-05 row and staged edge)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (X-09 row and Notes)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (parent epic report — Capability map "Claude JSON usage envelope" / "Provider usage and session metadata" rows)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/feature.md`, `research-report.md`, `spec.md` (TC-030 precedent)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md`
- `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F15-codex-and-claude-cross-session-usage-tracking/feature.md`, `implementation-plan.md`
- `internal/runner/controller.go:84-135` (`RunResult`, `StageLog`)
- `internal/runner/transcript.go:10-25` (transcript byte format)
- `internal/runner/dispatcher.go` (`DefaultDisallowedTools`)
- `internal/runner/claude_dispatcher.go` (grepped for `modelUsage`/`num_turns`/`duration_api_ms`/`total_cost_usd` on `main`: no hits)
- `bench/scripts/canary-runsurface.sh` (X-07 real-invocation preflight pattern)
- `bench/scenarios/scenarios.yaml`, `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml` (I-04 `evaluator_only`/`input.agent_visible` shape)
- `tests/contracts/e40_i04_scenario_contract_test.go` (TC-030, I-04 validator precedent)
- `shark get E40-F06/E27-F15/E22 --field status`
- `git branch --all --contains` the E27-F15 commit

RECOMMENDED OUTCOME: pass
