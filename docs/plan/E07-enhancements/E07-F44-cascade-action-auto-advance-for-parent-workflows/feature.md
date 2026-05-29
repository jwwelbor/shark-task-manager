---
feature_key: E07-F44-cascade-action-auto-advance-for-parent-workflows
epic_key: E07
title: Cascade action auto-advance for parent workflows
description: Change parent aggregation behavior for features and epics so child-driven advancement is gated by the parent status having orchestrator action 'cascade'. While any child is non-terminal, keep the parent in the cascade status. When all children become terminal, auto-advance the parent to its next configured workflow status instead of forcing completed. This must remain workflow-driven so simple profiles can still land on completed while richer profiles can continue beyond active. Preserve and regression-test E07-F38 auto-reopen behavior so parents still reopen correctly when a child regresses out of terminal or new work is introduced under a terminal parent.
size: 3
---

# Cascade action auto-advance for parent workflows

**Feature Key**: E07-F44

---

## Problem

Feature and epic rollup logic currently treats "all children terminal" as equivalent to "parent should become completed". That works for simple workflows where the aggregation status is the last meaningful step, but it breaks richer workflows where `active` is only the start of a post-execution phase such as code review, QA, or approval.

The real semantic boundary is the parent status whose orchestrator action is `cascade`. That status represents "run children until they are all terminal". When rollup code jumps straight to `completed`, it bypasses the parent workflow and can skip valid states after `active`. This also needs to coexist safely with E07-F38, which reopens parents when child work regresses or new work appears under a terminal parent.

## Solution

Make parent auto-advancement action-aware and workflow-driven. A feature or epic should only auto-advance because of child completion when its current status has orchestrator action `cascade`.

While any child is non-terminal, the parent remains in the cascade status. Once all children are terminal, the parent advances one configured workflow step from its current status. This preserves simple workflows where the next step is `completed`, while enabling richer workflows where the next step is something like `ready_for_code_review`.

## Scope

**In scope:**
- Feature-level child rollup behavior when tasks become terminal
- Epic-level child rollup behavior when features become terminal
- Action-aware gating based on the parent status using orchestrator action `cascade`
- Workflow-driven next-step resolution rather than hardcoded `completed`
- Regression-safe interaction with E07-F38 auto-reopen behavior
- Test coverage for both simple and extended workflow profiles

**Out of scope:**
- New workflow stages or profile redesign by itself
- Changing non-cascade planning/manual statuses to auto-advance
- Bugs, change-cards, or other non-parent/child entities
- Retrofitting unrelated status derivation behavior outside feature/epic parent cascades

## Planned Approach

1. Identify the live feature and epic rollup paths that currently equate "all children terminal" with `completed`.
2. Add a shared helper or consistent service logic that:
   - checks whether the parent's current status action is `cascade`
   - keeps the parent in place while children remain non-terminal
   - advances one workflow step when all children become terminal
3. Preserve the existing reopen-on-regression and reopen-on-new-child behavior introduced by E07-F38.
4. Align older derivation/recalculation paths so they do not reintroduce hardcoded `completed` behavior through side channels.
5. Add profile-sensitive tests showing:
   - basic/simple workflows still land on `completed`
   - extended workflows can advance beyond `active`

## Compatibility Constraint

This feature must not break E07-F38, [Auto-reopen parent entities on child regression](../E07-F38-auto-reopen-parent-entities-on-child-regression/feature.md).

Specifically:
- If a child regresses from terminal to non-terminal, terminal parents must still reopen correctly.
- If new child work is created under a terminal parent, that parent chain must still reopen correctly.
- The new forward-direction cascade behavior must not suppress or override the reopen cascade.
- Regression coverage must exercise the interaction between forward auto-advance and backward auto-reopen.

## Acceptance Criteria

- **AC1**: A parent feature or epic only auto-advances due to child terminality when its current status has orchestrator action `cascade`.
- **AC2**: While any child is non-terminal, a parent in a `cascade` status remains in that same status.
- **AC3**: When all children of a parent in a `cascade` status become terminal, the parent advances exactly one workflow step from its current status.
- **AC4**: The forward auto-advance target is workflow-driven; the implementation does not hardcode `active` or `completed`.
- **AC5**: In simple workflows where the cascade status transitions to `completed`, behavior remains unchanged from the user's perspective.
- **AC6**: In richer workflows where the cascade status transitions to another non-terminal status, the parent advances to that next state instead of skipping to `completed`.
- **AC7**: Parents in non-cascade statuses do not auto-advance solely because all children happen to be terminal.
- **AC8**: E07-F38 reopen-on-regression and reopen-on-new-child behavior continues to work unchanged.
- **AC9**: Automated tests cover both forward auto-advance and backward auto-reopen interactions across feature and epic levels.

## Planned Tasks

1. Trace and update feature-level rollup/progress recalculation so `cascade` semantics govern terminal-child advancement.
2. Trace and update epic-level rollup/recalculation so `cascade` semantics govern terminal-child advancement.
3. Refactor or align any legacy derivation paths that still force `completed` independently of workflow/action metadata.
4. Add unit and integration tests for:
   - feature forward auto-advance
   - epic forward auto-advance
   - simple workflow compatibility
   - extended workflow post-`active` advancement
   - E07-F38 regression interactions

## Risks

- Multiple status derivation paths may exist in parallel, so partial fixes could leave inconsistent behavior.
- Workflow profiles in the repo are not fully aligned today; code changes may enable richer behavior without activating it until workflow definitions are updated.
- Forward auto-advance and backward auto-reopen operate on the same parent entities, so ordering and trigger conditions must be explicit to avoid oscillation or stale status writes.

## Reference

- [E07-F38 feature](../E07-F38-auto-reopen-parent-entities-on-child-regression/feature.md)
- [E07 epic](../../epic.md)
