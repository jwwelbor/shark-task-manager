# E38-F06 role-aware sprint pull test plan

## Scope

This plan covers `T-E38-F06-004`, the live correction for
`shark sprint next --agent=<workflow-role>`. The selection authority is the
role declared by the current workflow step for each candidate entity; the
legacy persisted `tasks.agent_type` value is display/planning data only.

Claim conflicts remain outside this selection operation and are owned by
`ClaimService`.

## Test cases

### TC-F06-004 — CLI forwards a non-default role exactly

- Entrypoint: `runSprintNext` with `--agent=qa`.
- Mock seam: `sprintAssignmentServicer.GetNextTask`.
- Assertion: the CLI calls `GetNextTask(ctx, "qa")` unchanged.
- Counter-factual: a CLI that substitutes `developer` or drops the supplied
  value fails the captured-argument assertion.

### TC-F06-008 — workflow role controls eligibility despite a legacy mismatch

- Entrypoint: `SprintService.GetNextTask(ctx, "qa")`.
- Mock seam: `SprintRepository.List` and `SprintRepository.ListBacklog`; the
  real `SprintService` grouping, workflow metadata lookup, filtering, and
  comparator execute.
- Fixture: a `ready_for_development` task persisted as `qa`, followed by a
  `ready_for_qa` task persisted as `developer`.
- Assertion: the `ready_for_qa` task is selected. The persisted values must not
  authorize either candidate.
- Counter-factual: filtering `BacklogItemView.AgentType` returns the legacy
  `qa` task (or no task), so this test fails.

### TC-F06-009 — a non-matching workflow role returns no work

- Entrypoint: `SprintService.GetNextTask(ctx, "architect")`.
- Mock seam: `SprintRepository.List` and `SprintRepository.ListBacklog`.
- Assertion: no candidate is returned when no candidate step includes
  `architect`.

### TC-F06-010 — omitted agent remains unfiltered and deterministic

- Entrypoint: `SprintService.GetNextTask(ctx, "")`.
- Mock seam: `SprintRepository.List` and `SprintRepository.ListBacklog`.
- Assertion: all non-terminal items remain eligible and the existing
  sprint-order, execution-order, priority, then assigned-at comparator still
  selects the same first item.

### TC-F06-011 — selection remains separate from claim conflict handling

- Entrypoint: `ClaimService.Claim` after a selector result.
- Existing coverage: `TestClaimService_Claim_BlockedWhenLive`.
- Assertion: one ordinary claim succeeds and a competing non-force claim
  returns the existing conflict; `GetNextTask` does not claim, heartbeat,
  release, or change status.

## Required verification

- Focused service and CLI tests for TC-F06-004 and TC-F06-008 through
  TC-F06-010.
- Existing non-force ClaimService conflict regression for TC-F06-011.
- Full Go quality gate: `make fmt && make lint && make test`.
