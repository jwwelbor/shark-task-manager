---
entity_key: E36-F04
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - api
  - data
  - workflow_operations
  - documentation
source_set:
  - docs/plan/E36-project-layer-and-consult-bridge/E36-F04-portfolio-aware-next-action-advisor/feature.md
  - docs/plan/E36-project-layer-and-consult-bridge/epic.md
  - docs/plan/E36-project-layer-and-consult-bridge/requirements.md
  - docs/plan/E36-project-layer-and-consult-bridge/scope.md
  - dev-artifacts/2026-06-29-project-entity-design/plan.md
  - docs/plan/E36-project-layer-and-consult-bridge/E36-F01-consult-bridge/feature.md
  - docs/plan/E36-project-layer-and-consult-bridge/E36-F02-project-namespace-and-progress-record/feature.md
  - docs/plan/E36-project-layer-and-consult-bridge/E36-F03-ops-as-entities-convention/feature.md
  - docs/product/progress.md
  - docs/product/cross-epic-integration-map.md
  - docs/plan/E19-sprint-management-planning-system/E19-F07-sprint-relative-ordering-for-backlog-items/feature.md
  - skills/shark-rider/SKILL.md
  - skills/shark-rider/verbs/help.md
  - internal/cli/commands/next.go
  - internal/cli/commands/next_test.go
  - internal/cli/commands/epic.go
  - internal/cli/commands/link.go
  - internal/models/epic.go
  - internal/models/entity_relationship.go
  - internal/services/sprint_service.go
  - internal/services/entity_relationship_service.go
  - internal/services/claim_service.go
related_work: true
---

# Research plan — Portfolio-Aware Next-Action Advisor

## Scope

Determine how Shark Rider can recommend one epic root from a single project's
portfolio without creating a second workflow engine or mutating project state.
The research will define the ordering and eligibility contract, identify
reusable state and ordering mechanisms, design a read-only no-argument mode for
`shark next`, preserve keyed dispatch behavior, and establish failure and
explanation behavior.

E36 has no parent `research-report.md`. Use its epic PRD, requirements, scope,
and registered canonical design as the upstream record. Inspect all three
completed sibling feature specifications and related ordering work; reference
their decisions instead of copying their requirements.

## Recipe

Use the universal recipe at STANDARD rigor. Cover the selected categories as
follows:

- **Backend**: trace keyed `shark next`, read-only portfolio queries, claim
  inspection, relationships, blockers, and current-work signals.
- **API**: define distinct bare and keyed `shark next` response contracts.
- **Data**: identify the authoritative source for explicit portfolio order and
  the existing epic fields available as fallback signals.
- **Workflow operations**: define graph extraction, ordering, dispatch handoff,
  degradation, and the no-mutation boundary for bare `shark next`.
- **Documentation**: identify the minimum Rider procedure, router, contract
  tests, and product-record updates required to make the behavior durable.

Do not research frontend behavior or external market alternatives. This is an
internal repository workflow whose constraints and prior art are local.

## Source set

The front matter lists the bounded source set. It includes the feature and
upstream E36 records, all sibling feature specifications, product-level
artifacts, the Rider help/router contracts, and the backend services that own
dispatch, claims, relationships, and established ordered selection.

Related work also includes Shark idea `I-2026-01-02-06` (cross-entity
implementation planning). Treat it as problem evidence, not an implemented
contract.

## Steps

1. Define ubiquitous vocabulary for portfolio order, eligibility, readiness,
   WIP, claim, blocker, recommendation, root, portfolio-advice envelope, and
   keyed dispatch.
2. Complete the related-work inspection and record a non-empty Capability map
   with `REUSE`, `EXTEND`, `NEW`, or `CONTRADICTS` decisions.
3. Trace the selected backend and data sources from CLI output through services
   and models; record which signals are authoritative, advisory, or absent.
4. Trace every keyed `shark next` mutation path and isolate it from a new
   no-argument portfolio-advice path.
5. Define the bare `shark next` evidence envelope and the prompt that directs an
   agent to combine Shark state with relevant `docs/product/` artifacts.
6. Define deterministic graph layers, cycles, contradictory relationships,
   missing evidence, and dispatch-handoff behavior without designing
   implementation tasks.
7. Write `research-report.md`, validate both artifacts against the universal
   recipe, and register them as related documents on E36-F04.
