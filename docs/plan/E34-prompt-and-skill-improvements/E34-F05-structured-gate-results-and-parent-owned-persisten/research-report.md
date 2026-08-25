---
research_schema: 2
entity_key: E34-F05
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - workflow_operations
  - documentation
related_work: true
---

# E34-F05 Research Report: Structured Gate Results and Parent-Owned Persistence

## Scope

E34-F05 closes the result-handoff gap between quality-gate workers and the
parent orchestrator. It covers the Go core runner, the host-side Shark Rider
loop, shared prompt output policy, typed note persistence, task kickbacks, and
replay behavior. It does not replace Shark workflow configuration, add a new
database entity, or standardize every non-gate worker response.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `E34-F05-structured-gate-results-and-parent-owned-persisten/feature.md`; `skills/shark-rider/verbs/run.md`; `skills/shark-rider/context/task-execution-pattern.md`; `skills/shark-rider/context/host-adapter-contract.md`. These sources distinguish worker evidence from parent-owned persistence, kickback, transition, and release.
- [x] `affected_implementation_or_contract` — Evidence: `internal/runner/controller.go`; `internal/runner/dispatcher.go`; `internal/sharkdata/default_data/prompts/feature/{code_review.md,qa.md,approval.md}`. The core runner currently extracts only an outcome line, while gate prompts produce richer directives.
- [x] `related_work` — Evidence: `docs/plan/E34-prompt-and-skill-improvements/{epic.md,requirements.md,scope.md,E34-interaction-map.md}`; E34-F03 and E34-F04 feature/research packets; WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/{PROPOSAL.md,INVENTORY.md}`.
- [x] `pattern_contract` — Evidence: `internal/runner/controller.go` `parseQuestionResponseHandoff`; `internal/models/question.go` bounded-text validation; `internal/services/question_workflow_service.go` exact-response replay; existing `review-finding` prompt metadata.
- [x] `dependency_impact` — Evidence: `internal/runner/controller.go` dispatch-to-transition path; `skills/shark-rider/verbs/run.md` directive application order; note and task workflow command surfaces used by gate prompts.
- [x] `cross_boundary_risks` — Evidence: the worker-to-parent JSON boundary crosses untrusted model output, persistence services, task workflow mutation, and a final lifecycle transition. A parse or partial-write error could otherwise advance work or duplicate mutations.
- [x] `alternatives` — Evidence: the proposal considered worker-authored notes and free-form parent directive parsing. Worker writes violate the established parent authority boundary; extending ad hoc line grammars would preserve Rider/core divergence.

## Capability map

| Capability | Evidence | Decision | E34-F05 responsibility |
|---|---|---|---|
| Opaque workflow outcome routing | `recommendedOutcome` and `targetStatusForDispatch` in `internal/runner/controller.go` | EXTEND | Keep configured outcomes opaque while sourcing them from a validated gate envelope. |
| Parent-owned Rider mutations | `skills/shark-rider/verbs/run.md` | REUSE | Preserve claim, persistence, kickback, transition, and release authority in the parent. |
| Bounded worker handoff | Question response parser/model/service | REUSE | Generalize the marker, validation, safe errors, and exact-replay pattern for gates. |
| Typed notes with finding metadata | Gate prompts and Shark note commands | REUSE | Persist existing note types and metadata instead of adding a table. |
| GateResult v1 | Canonical worker-control final envelope exists; no nested gate payload or persistence coordinator exists | EXTEND | Add one nested model and shared persistence order without a second envelope. |
| Core runner directive persistence | `handleSpawnAgent` transitions directly after output parsing | NEW | Insert validated persistence before transition and fail closed on any error. |
| Rider/core behavioral parity | Separate current contracts | NEW | Add common fixtures and documentation assertions so the two paths cannot drift. |

## Findings

1. The current ownership rule is sound, but its return channel is incomplete.
   Letting workers write notes would weaken lease and replay guarantees; the
   parent should instead receive a complete bounded result.

2. The Question workflow already demonstrates the right architectural shape:
   accept one line of compact JSON, bind it to the real entity and parent
   session, validate bounded fields, persist before release, and recognize an
   exact replay. A generic gate result should reuse this pattern without using
   Question-specific storage.

3. The core runner and Rider are two real consumers. Updating only prompts or
   only `run.md` would leave `shark run` capable of advancing with lost
   findings. Contract fixtures must cover both paths.

4. Existing note storage can represent findings and sweeps. The result needs
   durable identity fields and an idempotent coordinator, but the research
   found no need for a schema migration.

5. Model output is untrusted. Top-level shape, version, workflow outcome,
   collection cardinality, field bounds, duplicate identities, and forbidden
   content must be validated before any object access or write.

## Decisions

1. Extend exactly one canonical worker-control `kind: final` envelope with a
   `gate_result` v1 payload for configured gates; add no second marker.
2. Keep the outcome opaque and validate it against the live step configuration.
3. Persist notes and kickbacks before transition under the stable run identity,
   with the authorized parent session retained as associated provenance.
4. Reuse typed notes and exact-replay patterns; do not add a finding table.
5. Keep an explicit migration period for non-gate and legacy prompt output,
   but never silently downgrade a gate configured for structured results.

## Sources

- `skills/shark-rider/verbs/run.md`
- `skills/shark-rider/context/{task-execution-pattern.md,host-adapter-contract.md}`
- `internal/runner/{controller.go,dispatcher.go,controller_test.go}`
- `internal/models/question.go`
- `internal/services/question_workflow_service.go`
- `internal/sharkdata/default_data/prompts/feature/{code_review.md,qa.md,approval.md}`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E34-prompt-and-skill-improvements/E34-F0{3,4}-*/{feature.md,research-report.md}`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/{PROPOSAL.md,INVENTORY.md}`
