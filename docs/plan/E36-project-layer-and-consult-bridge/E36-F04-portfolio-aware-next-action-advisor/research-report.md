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

# Research report — Portfolio-Aware Next-Action Advisor

## Scope

This report defines a read-only, no-argument mode for `shark next` that returns
the evidence and prompt needed to recommend one epic root from a single-project
portfolio. It covers explicit order, live eligibility and WIP signals,
deterministic graph construction, product-context guidance, and the handoff to
keyed workflow execution. It excludes implementation planning, cross-project
fan-out, sprint reordering, and a new project entity.

E36 has no parent research report. The upstream record is its epic PRD,
requirements, scope, and registered canonical design. All three sibling feature
specifications were inspected; none has a separate architecture or research
report.

## Capability map

| Capability | Established in | Decision | Application to E36-F04 |
| --- | --- | --- | --- |
| Read-only advisor boundary and graceful degradation | E36-F01 | **REUSE** | Keep advice non-mutating and explain unavailable signals. Do not reuse persona consultation as the execution mechanism. |
| Skill-layer project coordination | E36-F02 | **EXTEND** | Extend Rider's existing state-aware help procedure. Do not read `docs/product/progress.md` as authoritative order; F02 defines it as advisory. |
| Recurring state belongs in Shark entities | E36-F03 | **REUSE** | Represent durable epic relationships with existing entity links, not a second project ledger. |
| Stable lexicographic pull order and `selection_reason` | E19-F07 | **REUSE** | Reuse the deterministic precedence and explanation shape, but not sprint-scoped `sprint_order`. |
| Polymorphic `depends_on` and `follows` links | Entity relationship service | **EXTEND** | Use epic links for hard dependency and soft roadmap order. Detect `follows` cycles because the service prevents cycles only for `depends_on` and `blocks`. |
| State-aware next-action help | Rider `help.md` | **EXTEND** | Let state-aware help consume bare `shark next` after the command exists. Preserve `--fast`, `commands`, and verb-specific help as zero-state static modes. |
| Keyed dispatch resolution | `shark next <key>` | **REUSE** | Keep dispatch, cascade selection, prompt assembly, and normalization behavior unchanged. |
| Portfolio-advice envelope | bare `shark next` | **NEW** | Add a read-only no-argument path that returns epic state, relationships, ordering layers, warnings, and a product-context prompt. |
| Advisory `shark next --preview` flag | `next.go` | **CONTRADICTS** | The flag adds no behavior because `shark next` never spawns agents itself, while resolution may transition statuses. Remove the flag and keep dispatch outside the read-only advisor. |
| Full cross-entity implementation plan | Idea `I-2026-01-02-06` | **NEW** | Remains separate future work. E36-F04 selects one epic root and does not build a general execution-plan model. |

No sibling capability needs reimplementation. This feature will not recreate
the progress record, sprint queue, generic relationship storage, claim lease,
or `shark next` dispatch engine.

## Ubiquitous vocabulary

- **Portfolio**: all epics in the current Shark database; not multiple projects.
- **Root**: the epic key passed to `/shark-rider run <key>`.
- **Roadmap order**: explicit soft precedence expressed by an epic-to-epic
  `follows` relationship; it does not imply a hard dependency.
- **Hard dependency**: an outgoing epic `depends_on` relationship whose target
  must reach a configured terminal state before the source is eligible.
- **Candidate**: a non-terminal epic still under consideration.
- **Eligible**: a candidate whose hard dependencies are terminal and whose live
  workflow state does not expose a known blocker to starting the root.
- **Readiness**: the live workflow engine's ability to resolve a dispatch step;
  stored progress percentage alone is not readiness.
- **WIP**: work represented by a live, non-expired claim. A child claim counts
  as WIP for its ancestor epic.
- **Blocker**: a blocked-phase entity or unresolved hard dependency. A blocked
  child does not disqualify an epic if another child remains dispatchable.
- **Recommendation**: one eligible root plus the decisive ordering reason and
  runner-up comparison.
- **Portfolio-advice envelope**: the read-only response from bare `shark next`;
  it contains Shark evidence, deterministic graph output, warnings, and an
  advisory prompt.
- **Keyed dispatch**: `shark next <key>` resolution inside a chosen root. It may
  normalize workflow statuses while finding the next dispatch.
- **Dispatch handoff**: the point where advice ends and the operator explicitly
  invokes `/shark-rider run <key>`, allowing keyed dispatch to begin.

## Findings

1. **No explicit epic-order field exists.** `Epic` stores priority and business
   value but no execution order (`internal/models/epic.go`). `shark list --json`
   returns progress, priority, and business value through
   `internal/cli/commands/epic.go:195-241`; it cannot supply roadmap order.
2. **The product documents are not a roadmap queue.** `docs/product/progress.md`
   records decisions, and `cross-epic-integration-map.md` records integration
   ownership. Reading either as work order would create a new, implicit source
   of truth and conflict with E36-F02's advisory-progress rule.
3. **Existing links can express the missing semantics.** `follows` means
   "should be done after target" and `depends_on` means "cannot start until
   target completes" (`internal/models/entity_relationship.go:12-19`). Links
   work for epic keys through `shark link`/`shark links`. However, `follows`
   cycles are currently allowed; only `depends_on` and `blocks` receive service
   cycle detection.
4. **The CLI has no portfolio mode.** `nextCmd` currently requires exactly one
   argument and routes directly into keyed dispatch resolution. A no-argument
   branch can add portfolio advice without changing the keyed response or
   overloading entity-key semantics.
5. **The preview flag has no useful contract.** `nextPreview` was registered but
   never consumed. `shark next` already returns rather than spawning, so the
   caller needs no flag to decline execution. Meanwhile, cascade completion and
   agentless `advance_status` actions can call `TransitionStatus`; the dispatch
   API is intentionally not a read-only advisor query. Remove the flag rather
   than inventing simulation behavior for keyed dispatch.
6. **Bare mode needs non-mutating claim inspection.** `ClaimService.List`
   reclaims expired leases before returning them
   (`internal/services/claim_service.go`), so the portfolio path cannot use it
   while claiming zero writes. The specification must define a read-only claim
   query or omit claim rows that cannot be inspected without mutation.
7. **Ordered selection prior art favors explicit tiers, not weights.** Sprint
   selection uses a stable lexicographic sort and reports the first tier that
   separated the winner (`internal/services/sprint_service.go:1497-1635`). The
   portfolio advisor should copy that explainable shape, not invent a weighted
   readiness score.

## Decisions

1. Add bare `shark next` as the primary portfolio-advice command. Let Rider help
   consume this response after implementation; do not make Rider reconstruct
   the graph through a sequence of CLI calls.
2. Keep `shark next <key>` as the canonical keyed dispatch API with its current
   response, cascade behavior, and normalization transitions.
3. Use epic `follows` links as explicit roadmap order and `depends_on` links as
   hard eligibility gates. Do not add `execution_order` to epics or use the
   product progress file as an order store.
4. Have Go return deterministic dependency and ordering layers, stable epic
   facts, and warnings. Do not hard-code the final product recommendation in
   Go.
5. Return a prompt that directs the receiving agent to inspect relevant
   `docs/product/` artifacts, treat Shark as authoritative for live state, and
   recommend one root with a concise "why now" reason and runner-up comparison.
6. Keep bare mode strictly read-only, including claim inspection. Leave exact
   dispatchability and workflow normalization to the explicitly invoked Rider
   run loop.
7. Remove the inert `shark next --preview` flag and all active Rider guidance
   that recommends it. Preserve `shark next` status-normalization behavior and
   keep it confined to keyed dispatch workflows.
8. Update E36's parent docs during specification: they still describe only
   three features and "exactly one Go change," which no longer reflects F04 or
   the new portfolio command contract.

## Sources

- E36 upstream: `epic.md`, `requirements.md`, `scope.md`, and
  `dev-artifacts/2026-06-29-project-entity-design/plan.md`.
- Siblings: E36-F01, E36-F02, and E36-F03 `feature.md` files.
- Related work: E19-F07 `feature.md`; Shark idea `I-2026-01-02-06` via
  `shark get`; `docs/product/progress.md`; and
  `docs/product/cross-epic-integration-map.md`.
- Rider contracts: `skills/shark-rider/SKILL.md` and `verbs/help.md`.
- Backend evidence: `internal/cli/commands/{epic,link,next}.go`,
  `internal/models/{epic,entity_relationship}.go`, and
  `internal/services/{claim,entity_relationship,sprint_service}.go`.
