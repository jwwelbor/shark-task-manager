# E30 UAT Plan: Entity Mutation and Sprint Operations

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)  
**Date**: 2026-05-07

## Coverage Map

This UAT plan is derived from the epic success criteria:

- routine edits completed in the viewer
- faster time-to-save for small updates
- most Sprint planning actions completed in Sprint mode
- visible history for every successful write
- low viewer exit rate for routine maintenance

## Scenario 1: Edit An Entity Field In The Viewer

**Goal covered:** routine edits completed in the viewer; time-to-save under 60 seconds.

**What to verify**

- A maintainer can open an epic, feature, or task in the viewer.
- The user can change an editable field such as title, description, priority, assignment, or execution order when applicable.
- Saving the change updates the entity view without requiring a CLI or database edit.
- The updated value is visible immediately after refresh.
- The operation creates or preserves the expected history entry.

## Scenario 2: Add A Note To An Entity

**Goal covered:** viewer-based mutation adoption; history coverage.

**What to verify**

- A contributor can add a note from the entity context.
- The note is attached to the selected entity.
- The note is visible in the entity note list and in the audit/history surface where applicable.
- The user remains in the same entity context after the save.

## Scenario 3: Add And Remove A Dependency Or Relationship

**Goal covered:** viewer-based mutation adoption; visible validation; history coverage.

**What to verify**

- A user can create a dependency-style link between two supported entities.
- The relationship is visible in the entity relationship view after save.
- Removing the relationship succeeds only when the relationship exists.
- Invalid targets, duplicate links, and cycle-creating links are rejected with a clear validation message.
- No partial relationship write is left behind after a failed attempt.

## Scenario 4: Transition Status From The Viewer

**Goal covered:** viewer-based mutation adoption; history coverage; faster edits.

**What to verify**

- A user can move an entity to a valid next status from the viewer.
- The transition obeys workflow rules and backward-transition checks.
- The change is reflected in the entity view and in history.
- Any orchestrator action associated with the new status is surfaced to the user.
- A forced or invalid transition is rejected unless the workflow explicitly allows it.

## Scenario 5: Use Sprint Mode To Stage Or Remove Work

**Goal covered:** Sprint planning action completion; viewer exit rate.

**What to verify**

- A planner can open Sprint mode and use the Plan subview.
- The planner can stage an eligible entity into the sprint.
- The planner can remove an entity from the sprint.
- Capacity indicators remain visible during the action.
- The user can jump from the Sprint view back to the entity view without losing context.

## Scenario 6: Review Sprint Report And Return To The Entity

**Goal covered:** Sprint planning action completion; context preservation.

**What to verify**

- The Sprint Report view still loads after the new mutation actions are added.
- The user can inspect the report and then return to a specific entity.
- The report remains read-only.
- Mutation actions do not break report rendering or navigation.

## Scenario 7: Reject Invalid Mutations Cleanly

**Goal covered:** validation error rate and data integrity.

**What to verify**

- Missing required fields are rejected before any write occurs.
- Invalid note, relationship, or status data produces a clear error message.
- Path traversal or file writes outside the project root remain blocked by the existing file edit boundary.
- A failed mutation does not leave partial updates in the database or the viewer state.

## Performance And Security Considerations

- A common single-field edit should complete within the existing sub-60-second target from the epic success criteria.
- Mutation routes must remain limited to the local viewer posture and its existing CORS rules.
- Relationship writes must continue to enforce cycle detection and self-link prevention.
- Sprint assignment actions must continue to respect the single-active-assignment rule and Sprint status preconditions.
- File-edit protections must remain unchanged: absolute paths and root escapes stay blocked.

## Acceptance Notes

- These scenarios are pass/fail checks on business behavior, not implementation tests.
- The plan is complete only when each scenario can be verified from the viewer without falling back to CLI or direct database edits.
- Any new UI action should be considered incomplete until it preserves history and keeps the user in context.

