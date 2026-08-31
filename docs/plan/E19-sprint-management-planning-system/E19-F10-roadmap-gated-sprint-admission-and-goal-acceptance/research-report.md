---
research_schema: 2
rigor: complex
categories: [backend, api, data, workflow_operations, documentation]
related_work: true
---

# E19-F10 Research and Complexity Triage

**Feature:** Roadmap-Gated Sprint Admission and Goal Acceptance  
**Assessment:** COMPLEXITY: COMPLEX (score: 22/27)

## Scope validation

This is a feature, not a task. It adds one coherent policy across sprint
admission, bulk admission, readiness, planning, selection, override audit,
and close/goal acceptance. The affected code already spans
`SprintService`, sprint CLI commands, plan CLI commands, readiness models,
portfolio/roadmap services, and completion persistence.

## Existing seams

- Individual admission: `SprintService.AddEntityToSprint`.
- Bulk admission and planning candidates: `SprintService.BulkAddToSprint` and
  `SprintService.PlanSprint`.
- Readiness: `SprintService.GetSprintReadiness` and its assignment query
  boundary.
- Read-only selection: `SelectSprint`, `SelectActiveSprint`, and legacy
  `GetNextTask`.
- Closure: `CloseSprintWithCarryover` and sprint completion records.
- Roadmap/portfolio selection: plan services and existing epic dependency
  relationships.

## Complexity score

| Dimension | Score | Rationale |
| --- | ---: | --- |
| File impact | 3 | Service, repository, CLI, workflow/plan, models, and tests change. |
| Pattern novelty | 3 | A reusable roadmap-admission policy and goal-review contract do not exist. |
| Data model | 2 | A reasoned override and goal-review evidence need durable semantics. |
| API surface | 3 | Add, bulk add, readiness, plan, next, and close all expose behavior. |
| Cross-feature dependencies | 3 | Depends on E19 lifecycle/assignment/readiness and E19-F09 selection; consults roadmap/epic dependencies. |
| UI complexity | 0 | CLI/JSON only. |
| Task estimation | 3 | At least six independently testable producer/consumer changes. |
| Regression risk | 3 | Existing sprint dispatch and close behavior becomes policy-gated. |
| Execution effort | 2 | Cross-cutting implementation plus adversarial lifecycle tests. |

**Total:** 22/27 — **COMPLEX**. Autonomous build is not appropriate until
the policy contract, override authority, and executable-goal evidence format
are specified and independently tested.

## Scope

The feature owns roadmap-gated sprint admission and dispatch plus acceptance
of a declared Sprint Goal at close. It does not replace E19-F09's common
selection root or E34-F10's reusable prompt guard; it consumes and extends
their contracts.

Vocabulary: *portfolio gate* is the selected executable roadmap/epic layer;
*admission decision* is the reasoned allow/deny/override result for a sprint
member; *Sprint Goal Review* is evidence that the declared demonstration ran
and was accepted, distinct from task terminal status.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E19-sprint-management-planning-system/E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance/feature.md`
- [x] `affected_implementation_or_contract` — Evidence: `internal/services/sprint_service.go`
- [x] `related_work` — Evidence: `docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/feature.md`
- [x] `pattern_contract` — Evidence: `docs/plan/E19-sprint-management-planning-system/E19-F09-sprint-selection-root-on-shark-plan/spec.md`
- [x] `dependency_impact` — Evidence: `internal/services/sprint_service.go`
- [x] `cross_boundary_risks` — Evidence: `internal/cli/commands/sprint.go`
- [x] `alternatives` — Evidence: `docs/plan/E19-sprint-management-planning-system/E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance/feature.md`

## Capability map

| Capability | Source | Decision |
| --- | --- | --- |
| Sprint member admission, backlog, and close | E19-F03 service/CLI contract | EXTEND with one shared policy decision. |
| Capacity/readiness and planning composite | E19-F05 | EXTEND; roadmap eligibility becomes a scored/blocking factor. |
| Read-only plan and next selection | E19-F09 | EXTEND through an eligibility seam; do not create a second selector. |
| Workflow phase routing | E19-F08 | REUSE for status semantics. |
| Prompt-side reusable guard | E34-F10 | REUSE as consumer context; no duplicate guard. |

## Findings and recommendations

1. Make one service-level admission evaluator the only authority. Every
   mutating or dispatching surface must consume its typed decision rather than
   reimplementing ancestor/roadmap checks.
2. Keep planning output observational: show the selected portfolio gate and
   excluded reasons without mutating claims, status, or roadmap state.
3. Model the override as an explicit, reasoned audit record with a narrowly
   defined actor/command path; never use a silent boolean bypass.
4. Split goal acceptance from task completion. Closing needs a declared
   executable demonstration and a recorded review result that can reject back
   to active without destroying completed-task evidence.
5. Build contract tests around each policy consumer and a real
   producer-consumer integration fixture for ancestor dependencies and
   roadmap state.

## Findings

- Existing sprint behavior distributes policy decisions across add, bulk add,
  readiness, planning, and selection. A per-command implementation would
  recreate the exact drift E19-F09 removed.
- The policy crosses persisted dependency data, service eligibility, CLI JSON,
  and close-time evidence. Its failure mode is unsafe dispatch or a falsely
  completed sprint, not merely a display discrepancy.
- The viable alternative—guidance-only output that leaves each caller to
  decide—cannot enforce admission or preserve an audit trail. The selected
  shared evaluator is therefore required.

## Decisions

- Specify a typed, reusable admission evaluation service before changing any
  consumer command.
- Make overrides explicit, reasoned, persisted, and visible to every consumer.
- Treat a failed/missing goal demonstration as a close rejection returning the
  sprint to active; do not undo completed work.

## Sources

- `docs/plan/E19-sprint-management-planning-system/E19-F10-roadmap-gated-sprint-admission-and-goal-acceptance/feature.md`
- `docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/feature.md`
- `docs/plan/E19-sprint-management-planning-system/E19-F09-sprint-selection-root-on-shark-plan/spec.md`
- `internal/services/sprint_service.go`
- `internal/cli/commands/sprint.go`

RECOMMENDED OUTCOME: pass
