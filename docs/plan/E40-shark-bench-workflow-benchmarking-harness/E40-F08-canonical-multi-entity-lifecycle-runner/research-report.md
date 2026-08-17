---
research_schema: 2
entity_key: E40-F08
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

# Research report: Canonical multi-entity lifecycle runner

## Scope

E40-F08 is the host-side lifecycle adapter for lifecycle-v2 benchmark runs. It
starts with an admitted I-04 scenario, optionally consumes the I-06 feature
prelude, and drives the scenario root plus every eligible descendant through
the public keyed Shark lifecycle. Its durable output is I-07: an evidence
record of dispatch, deterministic fork choice, claim/heartbeat/release,
unchanged prompt handoff, semantic outcome, transition, Question decisions,
stage evidence, policy identity, artifact use, review findings, limits, and
the final named stop outcome.

The benchmark owns scheduling, replay, evidence capture, and safety ceilings.
Shark remains authoritative for routing, prompt rendering, workflow state,
claims, transitions, and Questions. F08 must extend these contracts through a
thin adapter; it must not fork `shark run`, reconstruct prompts, or introduce a
second workflow engine, claim store, or Question store.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/feature.md` (Scope, Acceptance boundary, Contracts, and Out of scope) and `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#lifecycle-run-record-contract` define the root/descendant, keyed dispatch, I-05 snapshot, I-06 prelude, I-07 run record, named stop outcome, workflow-policy, and structured review-finding vocabulary.
- [x] `affected_implementation_or_contract` — Evidence: `internal/cli/commands/next.go` (`NextResponse` and keyed dispatch contract), `internal/cli/commands/claim.go` (claim, heartbeat, and session-scoped release surfaces), `internal/services/claim_service.go` (`Claim`, `Heartbeat`, `Release`, TTL, and work-session journaling), `internal/services/transition_types.go` (`TransitionOptions`/outcome routing), `skills/shark-rider/context/host-adapter-contract.md` (verbatim prompt/result handoff), `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#lifecycle-run-record-contract` (I-07 shape), and `internal/models/validation.go` (typed `review-finding` note support).
- [x] `related_work` — Evidence: parent `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (E22/E27-F15/E38/E39 capability map and lifecycle-v2 boundary); sibling reports `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/research-report.md`, `E40-F06-stage-evidence-and-evaluator-isolation/research-report.md`, and `E40-F07-replayable-product-design-prelude/research-report.md` (I-04, I-05, and I-06 producer decisions); `E40-interaction-map.md` (I-04→I-07, I-05→I-07, I-06→I-07 staged edges); `E40-cross-epic-map.md` (X-11 Rider and X-13 Question contracts); and `docs/plan/E38-shark-attack-team-orchestration/` plus `docs/plan/E39-question-and-decision-workflow-management/` (owning contracts).
- [x] `pattern_contract` — Evidence: `internal/cli/commands/next.go` documents the stable keyed `shark next <entity-key>` JSON contract and explicitly assigns the loop to the harness; `skills/shark-rider/context/workflow-and-status.md` defines outcome-routed transitions and claim-as-lease semantics; `skills/shark-rider/context/host-adapter-contract.md` requires exact prompt bytes and bounded result envelopes; `internal/services/claim_service.go` provides the existing TTL/reclaim/session-journal pattern.
- [x] `dependency_impact` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#lifecycle-run-record-contract` and `E40-interaction-map.md` identify I-04/I-05/I-06/X-11/X-13 inputs and I-07 consumers E40-F09/F10; `internal/services/claim_service.go` writes `work_sessions`; `internal/db/db.go` provides generic `entity_type`/`entity_key` history/session persistence; `internal/cli/commands/notes_add_dispatch.go` and `internal/models/validation.go` provide the review-finding note path.
- [x] `cross_boundary_risks` — Evidence: `internal/cli/commands/next.go` separates Shark-rendered prompt provenance from host scheduling; `skills/shark-rider/context/host-adapter-contract.md` requires verbatim prompt delivery and parent-owned persistence; `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#stage-evidence-and-isolation-contract` requires evaluator-only material to be absent at dispatch; `docs/plan/E39-question-and-decision-workflow-management/architecture.md` and `internal/services/question_service.go` define durable Question state rather than transcript-only decisions; and parent/sibling reports document the still-active E22 surface and X-09 usage-identity uncertainty.
- [x] `alternatives` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` §1 and `architecture.md` reject opaque top-level `shark run` for lifecycle v2 because it cannot expose replay, per-dispatch snapshots, or deterministic descendant scheduling; the F05 report rejects globalizing the Go-only I-01 manifest; the F06 report rejects unaudited provider parsing and a two-root-only isolation convention; and the F07 report rejects using durable Questions for in-session D01-D05 elicitation.

## Capability map

| Capability | Evidence | Decision | F08 disposition |
|---|---|---|---|
| I-04 scenario package and adapter identity | F05 research report; `architecture.md#lifecycle-scenario-package-contract` | REUSE | Read-only input; adapter executes language-specific checks without leaking them into Shark workflow contracts. |
| I-05 immutable stage evidence and three-root isolation | F06 research report; `architecture.md#stage-evidence-and-isolation-contract` | EXTEND | F08 writes one stage/lifecycle reference per observed dispatch and preserves missing/failed evidence as named invalidity. |
| I-06 D01-D05 replay prelude | F07 research report; `architecture.md#product-design-replay-contract` | REUSE | Consume the authorized prelude for feature scenarios; do not duplicate product-design methodology or answer live questions. |
| Keyed dispatch and rendered prompt | `internal/cli/commands/next.go`; `skills/shark-rider/context/host-adapter-contract.md` | REUSE | Call `shark next <key> --json`, preserve the response and digest, and pass `prompt` unchanged. |
| Claim, heartbeat, work session, and release | `internal/cli/commands/claim.go`; `internal/services/claim_service.go` | REUSE | Parent controller owns the lease; use returned session IDs and safe session-scoped release on every exit path. |
| Semantic transition routing | `internal/services/transition_types.go`; `skills/shark-rider/context/workflow-and-status.md` | REUSE | Persist the worker outcome, route through the configured outcome map, and retain transition evidence. |
| Durable Question lifecycle | E39 architecture and research artifacts; `internal/services/question_service.go`; X-13 map row | REUSE | Route only authorized replay responses; missing authorization becomes `unresolved_gate`, never an invented decision. |
| I-07 lifecycle run record | `architecture.md#lifecycle-run-record-contract`; F08 feature contract | NEW, extending existing contracts | F08 creates the cross-entity evidence join and named-stop record consumed by F09/F10; it is not a replacement for Shark persistence. |
| Structured review-finding capture | `internal/models/validation.go`; `internal/sharkdata/default_data/prompts/feature/{code_review,qa,approval}.md` | EXTEND | Read raw typed notes, retain zero-finding and collector-failure states distinctly, and link findings to exact candidate/policy identity without adjudicating them. |

## Findings

1. `shark next <key> --json` is the canonical dispatch boundary. Its response
   already carries entity identity, action, agent type, provider, model,
   effort, the fully rendered prompt, prompt digest/size, hierarchy traversal,
   unresolved placeholders, and Question handoff fields. F08 should record the
   response before scheduling and should fail or mark the run ineligible when
   prompt provenance or required identity is absent.

2. The existing lease contract is sufficient for multi-entity execution but
   has important ownership rules. Claim opens a work session and reclaims
   expired leases; heartbeat requires the claim's session; release is safely
   session-scoped and closes the work session with an outcome. The host loop
   must therefore keep one session record per concrete entity, heartbeat while
   the worker runs, and release in success, failure, cancellation, lease-loss,
   and missing-outcome paths.

3. Workflow state and lease state are separate. F08 may schedule and record,
   but it must route the semantic outcome through Shark's configured
   transition service/API. It must not set a terminal status directly or infer
   completion from process exit, a green subprocess, or a worker's prose.

4. Hierarchy traversal is a real policy boundary. `shark next` can resolve
   cascade paths and expose a fork/selection; F08 must persist the returned
   traversal and deterministic fork decision, execute every eligible generated
   task, and stop the whole scenario with `resource_limit` when a positive
   ceiling is reached. A silently truncated descendant plan is invalid.

5. I-05 is the evidence producer and I-06 is the feature-prelude producer;
   F08 is their first live lifecycle consumer. It should reference their
   immutable artifacts and add observed runtime values, candidate snapshots,
   artifact access, workflow-policy identity, and non-overlapping time
   intervals rather than regenerate upstream artifacts.

6. Review notes already have a typed `review-finding` category and prompts
   define gate, round, severity, defect class, fingerprint, criterion, and
   disposition metadata. F08 must preserve raw notes and explicitly record
   three distinct gate states: zero findings, findings, and collection failure.
   Finding confirmation, deduplication, and truth comparison belong to F09.

7. Questions are a durable human-gate surface, not a transcript convention.
   An authorized replay response may be recorded and routed; missing or
   unauthorized input must produce `unresolved_gate` with partial evidence.
   F08 must not answer a Question from worker text or invent a default.

8. The main compatibility risk is upstream contract drift. E22 remains an
   active foundation, and the parent/sibling research identified unresolved
   provider usage/model field mapping under X-09. F08 should pin/canary the
   concrete `next`, claim, transition, release, evidence, and usage shapes and
   fail closed for missing required identity rather than emit incomparable
   I-07 records.

## Decisions

1. **Build a new host-side I-07 recorder/scheduler over public contracts.**
   Reuse Shark for dispatch, prompt rendering, claims, workflow transitions,
   Questions, and review notes; create only the benchmark-side lifecycle record,
   policy scheduler, evidence join, and safety-ceiling enforcement.

2. **Use keyed dispatch one entity at a time.** Preserve each response, record
   `resolved_via`/fork decisions, claim the returned key, hand off the exact
   prompt, heartbeat, persist the semantic result, transition by outcome, and
   session-release in a `finally`-equivalent path.

3. **Make stop outcomes explicit and publication-invalidating where required.**
   `pause`, `archive`, `error`, lease loss, missing outcome, `unresolved_gate`,
   cancellation, worker failure, and `resource_limit` retain partial evidence;
   a safety ceiling invalidates the whole scenario instead of publishing a
   partial baseline.

4. **Keep evaluator and review authority outside F08.** F08 records exact
   candidate/policy/artifact/review evidence; F09 owns structural evaluation,
   finding confirmation, identity validation, and aggregate eligibility.

5. **Add contract canaries before provider-backed execution.** Validate the
   live JSON shapes, prompt digest, lease/session behavior, transition outcome,
   Question handoff, and required usage/model identity against real invocation
   surfaces. Treat missing identity as a hard ineligibility reason.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml` (v2 universal catalog and complex module rules).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md` (parent goals G8–G19 and F08 boundary).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (upstream brownfield findings; cited rather than duplicated).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` (lifecycle-v2 host-loop design and alternatives).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (I-04, I-05, I-06, I-07, isolation, lifecycle, and ADR contracts).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` and `E40-cross-epic-map.md` (producer/consumer and X-11/X-13 boundaries).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/research-report.md` (I-04 decision).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/research-report.md` (I-05/X-09 decision).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/research-report.md` (I-06/X-10 decision).
- `internal/cli/commands/next.go`, `claim.go`, `notes_add_dispatch.go`, and `internal/services/claim_service.go`, `transition_types.go` (live public contract implementations).
- `skills/shark-rider/context/host-adapter-contract.md` and `workflow-and-status.md` (Rider handoff and lease/transition procedure).
- `docs/plan/E38-shark-attack-team-orchestration/` and `docs/plan/E39-question-and-decision-workflow-management/` (owning X-11/X-13 contracts).
- `internal/models/validation.go` and `internal/sharkdata/default_data/prompts/feature/{code_review,qa,approval}.md` (review-finding type and producer metadata).

RECOMMENDED OUTCOME: pass

