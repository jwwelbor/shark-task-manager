---
research_schema: 2
rigor: complex
categories:
  - backend
  - api
  - workflow_operations
  - documentation
related_work: true
---

# Research report: E19-F09 — Sprint selection root on `shark plan`

## Scope

E19-F09 is a COMPLEX feature that makes the sprint a first-class, read-only
selection root. It adds `shark plan sprint` for the active sprint and `shark
plan S###` for an active or planning sprint. Both must use the established
selection contract: filter non-terminal, unclaimed, and Question-gated work;
apply the configured parallel cap; and expose ordering and any expansion
boundary without dispatching, claiming, or changing status.

`shark sprint next` remains during its deprecation window, but becomes a
compatibility projection of the same selection result. The feature does not
add a schema migration, a second scheduler, a second claim store, or an
automatic expansion of feature and epic assignments.

Vocabulary: **selection** is read-only candidate ranking; **keyed dispatch**
starts only with `shark next <key>`; **active sprint** is the execution
collection; **planning sprint** is a preview-only collection; **Question
gate** excludes an item while its Question remains open; and **four-tier
order** means `sprint_order`, `execution_order`, `priority`, then
`assigned_at`.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md` defines the selection,
  dispatch, preview, compatibility, and ordering vocabulary; the parent
  `../../epic.md` establishes the sprint-management scope.
- [x] `affected_implementation_or_contract` — Evidence:
  `internal/cli/commands/plan.go`, `internal/services/plan_hierarchy_service.go`,
  `internal/cli/commands/sprint.go`, and `internal/services/sprint_service.go`
  are the direct CLI and service boundaries for plan selection and the legacy
  sprint-next selector.
- [x] `related_work` — Evidence: parent `../../research-report.md`, sibling
  feature briefs `../E19-F07-sprint-relative-ordering-for-backlog-items/feature.md`
  and `../E19-F08-sprint-workflow-yaml-first-class-lifecycle-with-embedded-agent-routing/feature.md`,
  related `../../E38-shark-attack-team-orchestration/proposal-parallel-team-integration.md`,
  and downstream consumer
  `../../E38-shark-attack-team-orchestration/E38-F12-parallel-team-topology/research-report.md`
  were reviewed.
- [x] `pattern_contract` — Evidence:
  `internal/services/plan_hierarchy_service.go:30-37` defines a claimable,
  non-mutating direct-child selection snapshot; `internal/cli/commands/plan.go`
  builds the existing plan response and its candidate-edge evidence.
- [x] `dependency_impact` — Evidence:
  `internal/cli/commands/sprint.go:39-56` keeps assignment and planning
  services separate, while its `GetNextTask` dependency exposes the legacy
  consumer contract that must delegate rather than retain divergent ranking.
- [x] `cross_boundary_risks` — Evidence:
  `internal/services/plan_hierarchy_service.go` crosses repository snapshots,
  workflow terminal checks, claim policy, and relationship state; the E38
  proposal §2 and §5 consume selection before each candidate re-enters keyed
  dispatch through `shark next <key>`.
- [x] `alternatives` — Evidence: `feature.md` rejects the earlier
  claim-aware exclusion flag and client-side enumeration; the E38 proposal §2
  rejects a hand-rolled team scheduler and makes Shark's plan surface the
  selection authority.

## Capability map

| Capability | Evidence | Decision for E19-F09 |
| --- | --- | --- |
| Existing plan hierarchy selection | `internal/services/plan_hierarchy_service.go`; `internal/cli/commands/plan.go` | EXTEND — preserve response shape, read-only behavior, claim filtering, and parallel cap while adding a sprint collection root. |
| Sprint assignment and four-tier rank | E19-F07 feature brief; `internal/services/sprint_service.go` | EXTEND — reuse assignment data and ordering; do not add storage or alter reorder semantics. |
| Sprint workflow roles | E19-F08 feature brief | REUSE — determine `--agent` eligibility from the candidate's current workflow step, not an ad hoc sprint roster. |
| Legacy `shark sprint next` consumers | `internal/cli/commands/sprint.go`; E38 proposal §5 | EXTEND — retain the top-candidate output as a compatibility adapter over the shared selector. |
| E38 parallel-team sprint mode | E38 proposal §2 and §5; E38-F12 research report | EXTEND — replace the documented interim client-side enumeration when the root ships; retain keyed `next` for dispatch. |
| A second selector, scheduler, or claim store | Feature brief and E38 proposal §2 | CONTRADICTS — these would duplicate Shark-owned selection and lease authority. |

## Findings

1. `PlanHierarchyService` already establishes the central safety pattern:
   enumerate a bounded snapshot, use entity-level workflow terminal checks,
   and expose only currently claimable children. Sprint selection should reuse
   that pattern rather than let command code query repositories directly.
2. The current legacy sprint command exposes `GetNextTask` through the
   assignment service. A compatibility wrapper can preserve its one-candidate
   result only if both commands use one ranking and eligibility path.
3. The feature crosses four authority boundaries: sprint-assignment data,
   workflow role and terminal classification, lease/claim state, and the E39
   Question gate. A read-only result must report candidates without reserving
   them; the caller must still obtain a canonical prompt and claim through
   `shark next <key>`.
4. Feature and epic assignments are not directly craft-dispatchable. Return
   them with an explicit expansion marker so a caller can invoke `shark plan
   <key>`; silently expanding them would change the selection contract and
   blur collection selection with hierarchy selection.
5. Planning-state `S###` is not equivalent to an active sprint. It needs a
   preview response that does not imply dispatch, while active-sprint selection
   supports the same read-only candidate contract used by the E38 coordinator.
6. The main regression risk is semantic drift between `shark plan sprint` and
   `shark sprint next`: claims, Question gates, ordering, agent filtering, and
   empty-result behavior must come from one service-level implementation and
   be tested against the same fixtures.

## Decisions

1. Add a dedicated sprint planning/selection service seam or narrowly extend
   the existing plan service. Keep CLI commands thin and free of repository
   access.
2. Make the shared selector own active versus planning visibility, eligibility,
   four-tier ranking, workflow-role filtering, expansion markers, and the
   configured cap. Keep it side-effect free.
3. Implement `shark sprint next` as a compatibility adapter that selects one
   sequential top candidate from that shared result. Put its deprecation notice
   in help text only, never JSON output.
4. Test read-only behavior, claim and Question exclusion, ranking, cap, agent
   filter, planning preview, all assigned entity types, expansion markers, and
   compatibility equivalence before activation.
5. No open external dependency blocks specification. The next workflow stage
   should turn these decisions into the feature spec and test plan.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E19-sprint-management-planning-system/E19-F09-sprint-selection-root-on-shark-plan/feature.md`
- `docs/plan/E19-sprint-management-planning-system/{epic.md,research-report.md}`
- `docs/plan/E19-sprint-management-planning-system/E19-F{07,F08}-*/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/{proposal-parallel-team-integration.md,E38-F12-parallel-team-topology/research-report.md}`
- `internal/cli/commands/{plan.go,sprint.go}`
- `internal/services/{plan_hierarchy_service.go,sprint_service.go}`

RECOMMENDED OUTCOME: pass
