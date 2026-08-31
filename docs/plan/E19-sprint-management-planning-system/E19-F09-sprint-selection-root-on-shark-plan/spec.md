# Specification: E19-F09 — Sprint selection root on `shark plan`

This specification implements the incremental sprint-selection capability in
the E19 epic PRD. It follows the research report's capability map: extend
existing plan selection and sprint assignment behavior, and do not create a
second scheduler, claim store, dispatch path, or schema.

## Requirements

### Functional requirements

- **REQ-F09-001:** `shark plan sprint [--agent TYPE] --json` selects from the
  active sprint backlog and returns the established read-only plan response.
  Return a single selected candidate or `parallel_candidates`, capped by
  `max_parallel_items`.
- **REQ-F09-002:** `shark plan S### [--agent TYPE] --json` accepts active and
  planning sprint states. A planning-state result is explicitly a preview and
  contains no dispatch prompt, claim, heartbeat, release, or status mutation.
- **REQ-F09-003:** Rank eligible assignments by `sprint_order` ascending with
  nulls last, `execution_order` ascending with nulls last, priority ascending,
  then `assigned_at` ascending. Preserve stable ordering for complete ties.
- **REQ-F09-004:** Exclude terminal, actively claimed, and Question-gated
  items. Determine terminal state and `--agent` eligibility from each
  candidate's configured entity workflow, not hardcoded status strings or the
  persisted display agent.
- **REQ-F09-005:** Return task, bug, change-card, and tech-debt candidates as
  directly dispatchable. Return assigned feature and epic candidates with an
  explicit expansion marker and never auto-expand or silently skip them.
- **REQ-F09-006:** Keep `shark sprint next [--agent TYPE]` as a compatibility
  adapter: it returns the shared selector's first sequential candidate in its
  existing output shape. Its help text, but not JSON output, points users to
  `shark plan sprint`.

### Non-functional requirements

- **REQ-NF09-001:** All plan forms remain side-effect free. Selection performs
  no claim, heartbeat, release, workflow mutation, or prompt rendering.
- **REQ-NF09-002:** Put selection and eligibility rules in a service layer;
  commands only parse, call services, and render output.
- **REQ-NF09-003:** Reuse existing assignment tables and claim/Question query
  seams. Do not add a migration, table, or column.
- **REQ-NF09-004:** Preserve exact compatibility behavior for the legacy JSON
  fields while making its selected key equal the first sequential plan result.

### Acceptance criteria

1. An active sprint with six assignments, one claim, and one open Question
   returns four eligible candidates in four-tier order and respects the
   configured cap.
2. Claiming the first returned candidate without changing status causes the
   next plan invocation to omit it and expose the next ranked candidate.
3. `--agent` filters by the current workflow-step role for every supported
   assignment type.
4. `shark plan S###` on a planning sprint returns an identified preview and
   does not change database state.
5. Feature and epic assignments carry an expansion marker; direct assignment
   types remain candidates without implicit hierarchy traversal.
6. `shark sprint next` selects the same first eligible key as a sequential
   `shark plan sprint` result, and its JSON output has no deprecation text.

### Out of scope

- Changing sprint schema, reorder behavior, lifecycle YAML, or sprint
  capacity/analytics.
- Claiming or dispatching from `shark plan`.
- Replacing existing downstream skills in this feature; E38 consumes the new
  root after it ships.

## Architecture

### Component changes

| File | Change |
| --- | --- |
| `internal/services/sprint_service.go` | Extract or introduce the shared sprint-selection operation. Reuse the existing four-tier rank and workflow-role index; add Question-gate and preview handling through narrow read-only seams. |
| `internal/services/sprint_service_test.go` | Add mock-backed service tests for rank, terminal/claim/Question filtering, role filtering, planning preview, and all assignment types. |
| `internal/cli/commands/plan.go` | Parse `sprint` collection and `S###` root before generic entity parsing, call the shared selector, and render the standard plan response without writes. |
| `internal/cli/commands/plan*_test.go` | Add command-level response-shape and read-only regression tests using mocked services. |
| `internal/cli/commands/sprint.go` | Convert `runSprintNext` to the compatibility adapter and update help text; retain its current output formatting contract. |
| `internal/cli/commands/sprint*_test.go` | Prove compatibility selection equivalence and absence of deprecation text in JSON. |

### Data model

No schema change. The selector reads existing `sprints`, `sprint_assignments`,
assigned entity records, lease state, and Question-gate state. It exposes an
in-memory result with sprint key, candidate identity, rank fields, current
workflow role, eligibility result, and a direct-dispatch versus expansion
marker.

### Service and command contracts

The shared service accepts a selection input containing either the active
sprint selector or an explicit sprint key, optional workflow role, selection
mode, and configured parallel cap. It returns a read-only ordered candidate
set and response metadata that identifies execution selection versus planning
preview.

`plan.go` converts that result into the existing plan wire shapes. It does not
call repositories. `sprint.go` requests a sequential cap of one from the same
service and converts the first direct candidate to the existing
`BacklogItemView` output. An expansion-only top candidate is not silently
dispatched; the compatibility result must report the documented expansion
boundary consistently with the plan result.

### Decisions

1. One service owns rank and eligibility. This eliminates the current drift
   risk between legacy `GetNextTask` and the plan surface.
2. Eligibility uses configured workflow services per entity level. This keeps
   custom workflow names valid and honors Question and claim state.
3. Plan selection never reserves work. A consumer obtains the canonical prompt
   and lease later through `shark next <key>`.
4. Planning and execution modes share ranking but differ in response metadata;
   preview does not imply an execution action.

## Cross-epic integrations

### Produces: X-03 — priority/dependency order and workflow-role-aware pull/claim

- **Consumer:** E38 Shark Attack Team Orchestration.
- **Contract / shape source:** `E38 architecture §4.1 and §4.6; sprint
  pull/claim contract`.
- **Handoff:** the Scrum Master can inspect ordered eligible work while each
  specialist still acquires a real keyed Rider lease before execution.
- **Test coverage:** add the E19 producer contract scenario to this feature's
  test plan; E38 retains the consumer pointer
  `tests/contracts/e38_f04_interactions_test.go#TC-003`.

## Verification strategy

Use mock service/repository seams for CLI and service tests. Keep integration
tests limited to the real sprint/claim/Question persistence boundary. Run
`make fmt`, `make lint`, and `make test` after implementation.

RECOMMENDED OUTCOME: pass
