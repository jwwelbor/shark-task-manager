# E19-F10 specification: Roadmap-gated sprint admission and goal acceptance

## Requirements

This feature extends the lifecycle, assignment, readiness, and planning
contracts in the E19 epic PRD. It does not redefine their existing capacity or
workflow semantics.

### Functional requirements

| ID | Requirement |
| --- | --- |
| REQ-F10-001 | The system must evaluate every candidate for individual admission, bulk admission, planning backlog, and dispatch against a single roadmap-admission decision. The decision must include the candidate key, selected portfolio epic, direct unmet ancestor dependency keys, state (`allowed`, `blocked`, or `overridden`), and a machine-readable reason code. |
| REQ-F10-002 | A candidate is blocked when its ancestor epic has an unmet `depends_on` prerequisite or when its ancestor is outside the current portfolio gate. A prerequisite is met only when its workflow status is terminal and successful. The evaluator must use configured workflow metadata, not literal status names. |
| REQ-F10-003 | `shark sprint add` and bulk-add must reject blocked work before writing an assignment. An exceptional prerequisite candidate may proceed only with `--override-reason <text>` of 20-500 non-whitespace characters; the system must record an immutable override record. |
| REQ-F10-004 | Sprint readiness must include a roadmap-admission factor. Any blocked assigned item forces this factor to zero and lists each blocking reason. The existing three-item factor must not offset this result. |
| REQ-F10-005 | `shark sprint plan`, `shark plan sprint`, and `shark sprint next` must exclude blocked candidates, retain existing claim/Question filters and ordering, and expose the selected portfolio gate plus excluded counts and reason codes in JSON planning output. |
| REQ-F10-006 | Closing a sprint requires a submitted Sprint Goal Review containing a declared executable goal, before-step result, after-step result, reviewer identity, and an `accepted` or `rejected` outcome. A rejected or absent review returns an attempted close to active and records no completion row; accepted review permits the existing carryover/completion transaction. |

### Non-functional requirements

| ID | Requirement |
| --- | --- |
| REQ-NF10-001 | Every consumer must call the shared admission evaluator; no CLI command may duplicate ancestor or roadmap queries. |
| REQ-NF10-002 | Read-only plan/selection commands must not create overrides, goal reviews, claims, sessions, status changes, or assignments. |
| REQ-NF10-003 | An override and goal review must be transactionally persisted with the action they authorize or decide. Failed persistence must fail closed. |
| REQ-NF10-004 | The evaluator must use bounded, batched repository reads and report errors rather than silently treating unavailable dependency data as eligible. |

### Acceptance criteria

1. Given a task under an epic with an incomplete ancestor dependency, `sprint add`, bulk add, planning, selection, and next reject or omit it with the same reason code.
2. Given the same candidate with a valid override reason, admission succeeds and stores one override record; selection/readiness show it as overridden, not ordinarily allowed.
3. Given a sprint with a blocked assigned item plus three otherwise healthy items, roadmap readiness is zero and the overall readiness result identifies the blocked key.
4. Given an active sprint whose first-ranked item is blocked and second-ranked item is allowed, both plan and next return the allowed second item without changing claims or statuses.
5. Given a planning sprint, plan output reports the selected portfolio gate, each excluded reason count, and no writes.
6. Given all sprint items completed but no accepted goal review, close returns the sprint to active and creates no `sprint_completions` row. Given an accepted review, close creates the normal completion row and retains the review evidence.

### Out of scope

- New roadmap authoring commands or changes to portfolio prioritization.
- Replacement of E19-F09’s common plan selection envelope or E34-F10’s prompt guard.
- Automatic execution or interpretation of a demo command; a reviewer submits the observed before/after evidence.
- Retroactive goal-review requirements for existing completed sprints.

## Architecture

### Components and file changes

| Component | Change |
| --- | --- |
| `internal/services/sprint_admission_service.go` | Add the typed evaluator, decision/result types, portfolio-gate resolution, ancestor traversal, and override validation. |
| `internal/repository/sprint/repository.go` | Add batched assignment/ancestor lookup methods and persistence methods for overrides and goal reviews. |
| `internal/models/sprint.go` | Add `SprintAdmissionDecision`, `SprintAdmissionOverride`, and `SprintGoalReview` models. |
| `internal/db/db.go` | Add idempotent migrations for `sprint_admission_overrides` and `sprint_goal_reviews`, keyed by sprint and candidate/review attempt. |
| `internal/services/sprint_service.go` | Inject the evaluator into admission, bulk admission, readiness, planning, selection, next, and close paths. |
| `internal/cli/commands/sprint.go` | Add `--override-reason`, goal-review submission, and structured rendering. |
| `internal/cli/commands/plan.go` | Add gate/exclusion fields to sprint plan JSON only; preserve E19-F09’s selection shape. |
| `internal/cli/services_global.go` | Construct and inject the evaluator from repository and portfolio services. |
| `internal/repository/sprint/repository_test.go`, `internal/services/sprint_service_test.go`, `internal/cli/commands/*test.go` | Cover repository, service, CLI, and persisted producer-consumer contracts. |
| `tests/contracts/e19_f10_interactions_test.go` | Verify one shared admission decision reaches every declared consumer. |

### Data model

`sprint_admission_overrides` stores: `id`, `sprint_id`, `entity_type`,
`entity_id`, `reason`, `requested_by`, `created_at`, and the evaluator reason
code that was overridden. It has a unique active record for
`(sprint_id, entity_type, entity_id)` and cascade deletion from `sprints`.

`sprint_goal_reviews` stores: `id`, `sprint_id`, `goal`, `before_result`,
`after_result`, `reviewer`, `outcome`, `reviewed_at`. `outcome` is
`accepted` or `rejected`. Each close attempt creates at most one review; only
the latest accepted review may authorize a completion transaction.

### Interface contracts

The admission evaluator accepts a sprint key and candidate identity. It returns
a decision containing `state`, `reason_code`, `portfolio_epic_key`,
`unmet_ancestor_keys`, and any applicable override reference. The goal-review
service accepts declared goal evidence and returns a persisted review. The
existing close service accepts its existing carryover input but may complete
only when the latest goal review is accepted.

The evaluator is read-only. The admission service persists an override only
after a blocked decision and valid reason. The close service reads the latest
review inside its existing transaction before it writes status, carryover, or
completion records.

### Key technical decisions

1. Use one typed evaluator rather than per-command predicates. This preserves
   a consistent reason code and makes contract testing possible.
2. Treat missing dependency/roadmap data as an evaluator error, not an allow.
   This prevents unsafe dispatch during repository failure.
3. Keep the override narrow and auditable. A non-empty Boolean bypass would
   erase the reason and make later planning/readiness misleading.
4. Treat goal acceptance as explicit review evidence. Completed task rows
   measure throughput; they do not prove the sprint goal was demonstrated.

### Integration with existing code

`AddEntityToSprint` and `BulkAddToSprint` call the evaluator before assignment
writes. `PlanSprint`, `SelectSprint`, and `GetNextTask` consume decisions to
filter output. `GetSprintReadiness` adds the admission factor after it loads
assignments. `CloseSprintWithCarryover` validates the latest goal review before
its status transition and completion write.

## Cross-epic integrations

### Consumes X-03

E19-F10 extends the existing E19 role-aware sprint pull contract consumed by
E38-F04. Its shape source remains E38 architecture §4.1 and §4.6; blocked
candidates are omitted before E38 self-pulls them. Contract coverage extends
`tests/contracts/e38_f04_interactions_test.go#TC-003` and adds the F10
admission-consumer contract test. The skill handoff remains read-only and
continues to call keyed `shark next` only after a plan candidate is selected.

No E19 interaction map exists, so this feature declares no invented I-##
records.
