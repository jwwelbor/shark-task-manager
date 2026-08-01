---
research_schema: 2
entity_key: E39-F03
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - api
  - data
  - workflow_operations
  - documentation
related_work: true
---

# E39-F03 research report: Scoped Question blocking gate

## Scope

E39-F03 consumes I-01, the registered Question record, and I-02, its
validated serial state. It produces I-03: a direct-only `question_blocks`
predicate and a compact stop result at supported keyed dispatch and transition
boundaries. A qualifying Question is open, has `blocking=true`, and has a
direct `Question -> affected entity` `question_blocks` relationship.

The gate must not claim, transition, resolve, or otherwise mutate either the
Question or linked work. It does not change generic `blocks`, traverse a
relationship graph, block unrelated work, add a global queue, or implement
F04 read and disclosure surfaces. The parent architecture marks this
cross-boundary change as COMPLEX research.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md`, parent `requirements.md` REQ-F-005 through REQ-F-006 and REQ-F-009, `scope.md`, and `user-journeys.md` J-03 define the direct open/blocking predicate, compact stop result, and no-linked-work-mutation boundary.
- [x] `affected_implementation_or_contract` — Evidence: `internal/models/entity_relationship.go`, `internal/repository/entityrel/repository.go`, `internal/cli/commands/next.go`, `internal/cli/commands/run.go`, and `internal/cli/commands/status_group.go` establish the relationship vocabulary, keyed dispatch/run pipeline, lease ordering, and status-advance path that the gate must extend.
- [x] `related_work` — Evidence: parent `research-report.md`, `architecture.md`, `E39-interaction-map.md`, F01 and F02 research reports, and F04 `feature.md` identify I-01/I-02 as prerequisites, I-03 as F04's input, and E38-F09 as a future consumer rather than F03 scope.
- [x] `pattern_contract` — Evidence: `internal/services/entity_relationship_service.go` provides typed directed relationship validation and `internal/services/standalone_planning_service.go` reads generic `depends_on`/`blocks` separately from workflow mutation; the Question gate needs an equivalent read-only, Question-specific seam.
- [x] `dependency_impact` — Evidence: `internal/services/question_service.go` exposes persisted Question status and validated `QuestionState`; `internal/cli/commands/next.go` is the keyed dispatch path; `internal/cli/commands/run.go` performs a pre-lease action check; and `internal/cli/commands/status_group.go` dispatches supported advances.
- [x] `cross_boundary_risks` — Evidence: `next.go` renders and resolves action after status lookup, while `run.go` claims only dispatching actions and `QuestionService` keeps responder completion separate from claims. A gate placed after placeholder rendering, claim acquisition, or transition persistence can leak responder behavior or mutate the very work it must only stop.
- [x] `alternatives` — Evidence: `models.EntityRelBlocks` and `StandaloneHardDependencyService` already implement generic dependency semantics, while the parent architecture and REQ-F-005 explicitly reject reinterpreting `blocks`, global blocking, indirect traversal, and worker-owned linked-work actions.

## Capability map

| Capability | Evidence | Decision | F03 responsibility |
| --- | --- | --- | --- |
| Registered Question identity and generic lookup | F01 research report; `QuestionRepositoryAdapter`; entity registry | REUSE | Resolve the Question and candidate through existing typed repositories; do not repeat F01 registration. |
| Validated open/answering state and current responder | F02 research report; `QuestionService`; `QuestionState` | REUSE | Treat I-02 state as input to the open predicate and compact result; do not record responses or resolve Questions. |
| Directed polymorphic relationships | `EntityRelationship`; entity relationship repository/service | EXTEND | Add a distinct `question_blocks` type with only `Question -> eligible candidate` direction. |
| Generic `blocks` and hard dependency evaluation | `EntityRelBlocks`; `StandaloneHardDependencyService` | REUSE, but separate | Preserve its current behavior. It neither qualifies nor substitutes for `question_blocks`. |
| Keyed next, run, and cascade dispatch | `next.go`; `run.go`; F02 dispatch fixes | EXTEND | Check the predicate before prompt rendering or lease acquisition and return a compact pause for the affected candidate. |
| Status advance | `status_group.go`; feature/task/epic transition services | EXTEND | Check immediately before supported transition commit and reject or pause without a linked-work write. |
| Compact Question handoff | Parent architecture; I-03 interaction-map row | NEW | Define a small stable payload containing the Question key, summary, resolution owner, and current responder; F04 projects the consumer-facing query surfaces. |
| F04 focused reads and E38-F09 activation | F04 `feature.md`; interaction map | CONTRADICTS | Do not implement full disclosure, host queues, or the blocked consumer adapter. |
| Generic/global/indirect blocking | Parent scope; REQ-F-005; architecture ADR-003 | CONTRADICTS | Exclude legacy `blocks`, transitive edges, unlinked Questions, and all non-Question blockers from this gate. |

## Findings

1. **The existing relationship vocabulary cannot express the F03 predicate.**
   `EntityRelationshipType` currently allows `blocks` but not
   `question_blocks`. The generic dependency reader treats incoming `blocks`
   as an unresolved hard dependency without checking Question type, Question
   status, or `blocking`. Reusing it would broaden legacy behavior and violate
   REQ-F-005.

2. **F03 needs a read-only QuestionBlocker service.** The repository can load
   directed incoming relationships and F01 supplies generic entity resolution.
   A dedicated service should resolve only incoming `question_blocks`, reject
   non-Question sources defensively, load each Question, and qualify only
   non-terminal open/answering Questions with `blocking=true`. It should return
   a deterministic compact record or no match; no path should write a Question,
   relationship, claim, history row, or candidate status.

3. **The gate belongs before all irreversible dispatch or transition work.**
   `resolveEntity` reads status, builds placeholders, renders an action, and
   then resolves cascades. `run` separately checks whether the action needs a
   lease. The gate must run before Question placeholder generation, prompt
   assembly, or lease acquisition for the candidate; cascade recursion must
   apply the same check to the selected child. `status advance` must call the
   same service before its transition service persists the target status.

4. **Open must follow Question lifecycle semantics, not a hard-coded status
   list copied into callers.** F02 owns the serial state and uses `open`,
   `answering`, and `ready_for_resolution` plus terminal resolution outcomes.
   The blocker should centralize its qualifying-status policy, use the Question
   workflow classifier where available, and test all configured/terminal
   states. A completed responder or a released claim must not itself clear the
   blocking predicate.

5. **The compact handoff is a contract boundary.** I-03 specifies `Q###`,
   summary, resolution owner, and current responder. The gate returns only
   those values to the linked-work owner. It must not render Question response
   summaries, evidence pointers, or full context; F04 owns focused read and
   disclosure behavior.

6. **The implementation must cover every supported candidate path, not just
   `shark next`.** The architecture requires keyed dispatch and advancement.
   F02's run/cascade parity work shows that a fix in `next` alone can diverge
   from `run`. Tests must cover direct keyed `next`, direct `run` (including
   dry-run), cascade-selected child dispatch, and each supported status advance
   adapter, plus unlinked and generic-`blocks` controls.

## Decisions

1. **Add `question_blocks` as a new directed relationship type.** Permit only
   `question -> affected eligible entity`; use a service validation rule to
   reject reverse direction, Question targets, and unsupported candidate types.
   Keep it non-cyclic and separate from the generic relationship graph rules.

2. **Implement a read-only `QuestionBlocker` seam.** Give it relationship and
   Question read dependencies plus workflow classification. Its contract is
   `Check(candidate type, candidate key/id) -> no match | compact block`.
   It never receives a transitioner, claim service, or write repository.

3. **Apply one predicate at all supported boundaries.** Integrate it before
   direct keyed dispatch, every cascade child dispatch, top-level run lease
   acquisition, and before a supported status transition commits. Reuse the
   same result and error shape so all paths remain semantically aligned.

4. **Keep the compact payload stable.** Return the qualifying Question key,
   one-line summary, resolution owner, and derived current responder. Do not
   include `question_state`, response summaries, evidence pointers, or generic
   relationship details.

5. **Prove direct-only, non-mutating behavior.** Add integration coverage for
   open+blocking+direct positive cases; each negative predicate term; generic
   `blocks`; indirect chains; unrelated siblings; terminal Questions; and
   atomic no-write assertions for Question, candidate, claim, history, and
   relationship rows. Add parity tests for next, run, cascade, and advance.

## Sources

- `docs/plan/E39-question-and-decision-workflow-management/E39-F03-scoped-question-blocking-gate/feature.md`
- Parent contracts: `epic.md`, `requirements.md`, `scope.md`, `research-report.md`, `architecture.md`, `user-journeys.md`, and `E39-interaction-map.md`
- Upstream work: F01 and F02 `feature.md` and `research-report.md`; F04 `feature.md`
- Recipe: `internal/sharkdata/default_data/research/recipes.yaml`
- Relationship implementation: `internal/models/entity_relationship.go`, `internal/repository/entityrel/repository.go`, and `internal/services/entity_relationship_service.go`
- Question state/dispatch implementation: `internal/models/question.go`, `internal/services/question_service.go`, `internal/cli/commands/{next,run,status_group}.go`
- Existing generic dependency behavior: `internal/services/standalone_planning_service.go`

RECOMMENDED OUTCOME: pass
