---
feature_key: E07-F44
title: Cascade action auto-advance for parent workflows — Combined Spec
status: in_specification
last_updated: 2026-05-19
references:
  - feature.md
  - ../epic.md
  - ../E07-F38-auto-reopen-parent-entities-on-child-regression/spec.md
  - ../E07-F38-auto-reopen-parent-entities-on-child-regression/architecture.md
---

# E07-F44: Cascade Action Auto-Advance for Parent Workflows — Specification

This document is the combined requirements and architecture spec for E07-F44. It is incremental over the placeholder E07 epic and over the existing workflow/config architecture already in the codebase. For the prior parent-reopen design that this feature must preserve, see E07-F38 `spec.md` and `architecture.md` in the sibling feature directory.

The parent epic PRD (`../epic.md`) does not contain additional business requirements beyond "enhancements"; the authoritative problem statement and scope for this feature are in `feature.md`.

## 1. Requirements

### 1.1 Scope

In scope:

- Feature-level parent rollup when task children become terminal or regress out of terminal.
- Epic-level parent rollup when feature children become terminal or regress out of terminal.
- Gating forward auto-advance on the parent status whose orchestrator action is `cascade`.
- Advancing exactly one workflow-configured step from the current parent status when all children are terminal.
- Preserving E07-F38 reopen-on-regression and reopen-on-new-child behavior.
- Regression coverage for simple workflows where the next step remains `completed` and richer workflows where the next step is post-`active`.

Out of scope:

- New workflow stages, workflow profile redesign, or new orchestrator actions.
- Any schema migration or new persistent fields.
- Bugs, change-cards, ideas, or other non parent-child rollups.
- Reworking task-level completion semantics.
- Changing non-`cascade` planning or review statuses to auto-advance from child state.

### 1.2 Functional Requirements

**REQ-F-001**
Parent entities MUST only auto-advance from child completion when the parent's current status metadata has orchestrator action `cascade`.

**REQ-F-002**
If any child of a parent in a `cascade` status is non-terminal, the parent MUST remain in its current status. It MUST NOT jump backward, forward, or to `completed` solely because progress changed.

**REQ-F-003**
When all children of a parent in a `cascade` status are terminal, the parent MUST advance exactly one configured workflow step from its current status. The target status MUST come from workflow configuration, not a hardcoded status name.

**REQ-F-004**
If the parent's current status is not a `cascade` status, "all children terminal" MUST NOT by itself force any status transition. Cached progress may still update, but status derivation must leave the parent in place.

**REQ-F-005**
Feature-level forward auto-advance MUST work through the task rollup path currently implemented in `internal/services/feature_progress_service.go`.

**REQ-F-006**
Epic-level forward auto-advance MUST work through the feature rollup path currently implemented in `internal/services/epic_service.go` and `internal/services/epic_analytics_service.go`.

**REQ-F-007**
The implementation MUST remain workflow-profile-driven. Terminal classification, `cascade`-action detection, and next-step resolution MUST all derive from `workflow.Service` and status metadata, not from hardcoded checks for `active` or `completed`.

**REQ-F-008**
E07-F38 behavior remains intact:
- a parent that was auto-advanced forward may still reopen correctly when a child regresses out of terminal;
- a terminal parent may still reopen when new child work is created beneath it;
- forward auto-advance must not suppress or bypass the existing reopen cascade hooks.

**REQ-F-009**
When the workflow-configured next step after a `cascade` status is itself terminal, user-visible behavior remains compatible with current simple workflows: all children terminal still results in the parent becoming terminal.

**REQ-F-010**
When the workflow-configured next step after a `cascade` status is non-terminal, the parent MUST transition to that non-terminal step and stop there. Any later advancement is handled by the normal workflow/orchestrator, not by rollup code.

### 1.3 Non-Functional Requirements

**REQ-N-001**
No schema changes, migrations, or new tables.

**REQ-N-002**
No new CLI or HTTP commands. Behavior changes are limited to status derivation and the resulting existing outputs.

**REQ-N-003**
The implementation must preserve existing status-override behavior. If a parent has `status_override=true`, rollup recalculation must not forcibly rewrite its status.

**REQ-N-004**
Regression coverage must exercise both forward auto-advance and backward reopen interaction across feature and epic levels.

**REQ-N-005**
`make fmt && make lint && make test` must pass after implementation.

### 1.4 Acceptance Criteria

- **AC-01**: A feature in a `cascade` status with at least one non-terminal task remains in that same status after progress recalculation.
- **AC-02**: A feature in a `cascade` status with all tasks terminal advances one workflow step from its current status.
- **AC-03**: Under a simple workflow where the next step after the `cascade` status is terminal, the feature still lands on `completed`.
- **AC-04**: Under an extended workflow where the next step after the `cascade` status is non-terminal, the feature lands on that configured next status rather than `completed`.
- **AC-05**: A feature in a non-`cascade` status with all tasks terminal does not auto-advance solely from child state.
- **AC-06**: An epic in a `cascade` status with at least one non-terminal feature remains in that same status after status recalculation.
- **AC-07**: An epic in a `cascade` status with all features terminal advances one workflow step from its current status.
- **AC-08**: Under an extended workflow, an epic can advance from `active` to a configured post-execution status without skipping directly to `completed`.
- **AC-09**: If a child regresses after the parent was auto-advanced forward, the existing E07-F38 reopen cascade still reopens the parent chain correctly.
- **AC-10**: Creating new child work under a terminal parent still triggers the existing creation-based reopen behavior from E07-F38.
- **AC-11**: Status-override parents are not auto-advanced by rollup logic.
- **AC-12**: All relevant unit/integration tests pass under the repository's current workflow configuration and the targeted custom-profile fixtures used by the existing workflow tests.

## 2. Architecture

### 2.1 Design Summary

The current codebase has two separate forward-rollup paths that hardcode terminal-child completion to `completed`:

- `FeatureProgressService.deriveFeatureProgressStatus(...)` in `internal/services/feature_progress_service.go`
- `deriveEpicStatusFromFeatures(...)` in `internal/services/epic_analytics_service.go`, called from `EpicService.RecalculateStatus(...)`

E07-F44 replaces the hardcoded "all children terminal => completed" rule with an action-aware workflow rule:

1. Determine whether the parent's current status metadata has orchestrator action `cascade`.
2. If not, preserve the current parent status.
3. If yes and any child is non-terminal, preserve the current parent status.
4. If yes and all children are terminal, resolve the next configured workflow step from the current parent status and advance to that one step only.

This keeps rollup behavior aligned with workflow configuration and leaves later review/QA/approval progression to the normal orchestrator flow.

### 2.2 Affected Files

Files expected to change:

- `internal/services/feature_progress_service.go`
- `internal/services/feature_progress_service_test.go`
- `internal/services/epic_analytics_service.go`
- `internal/services/epic_service.go`
- `internal/services/epic_service_test.go`
- `internal/workflow/service.go` or a nearby workflow helper file if a small convenience helper for "next configured status" or "orchestrator action for current status" is needed
- Existing E07-F38 regression tests if they need extension for the forward-then-reopen scenario

Files expected to remain unchanged:

- `internal/services/task_service.go`
- `internal/services/feature_service.go`
- `internal/services/cascade_reopen.go`
- repository packages and database schema

Rationale: E07-F38's reopen cascade is already centralized and workflow-driven. E07-F44 should integrate with it through status semantics, not by rewriting the reopen implementation.

### 2.3 Detailed Changes

#### 2.3.1 Feature rollup path

`FeatureProgressService.deriveFeatureProgressStatus(...)` currently:

- preserves status when there are no tasks or `status_override=true`;
- returns `completed` when weighted progress reaches 100%;
- reopens `completed` features back to the aggregation status when progress drops below 100%.

Required change:

- Replace the unconditional `completed` return on 100% progress with logic that first inspects the current feature status metadata.
- If the current feature status action is not `cascade`, leave the status unchanged.
- If the current feature status action is `cascade`, resolve the next workflow step from the current feature status and return that step.
- Keep the existing reopen-on-progress-drop behavior for already-completed features, because that supports the broader reopen story and must remain compatible with E07-F38.

Implementation note:

- Weighted progress is still the correct signal for "all tasks terminal" because terminal classification already comes from the task workflow.
- The status transition decision must become action-aware; progress calculation itself does not change.

#### 2.3.2 Epic rollup path

`deriveEpicStatusFromFeatures(...)` currently returns:

- current status when there are no features;
- `completed` when all child features are `completed`/`archived`;
- `active` when any child is `active`;
- otherwise `draft`.

Required change:

- Replace this status-name-specific derivation with a workflow-aware derivation keyed off the current epic status and child terminality.
- If the current epic status action is not `cascade`, leave the current epic status unchanged.
- If the current epic status action is `cascade` and all child features are terminal, resolve the next configured workflow step from the current epic status and return it.
- If the current epic status action is `cascade` and any child feature is non-terminal, keep the current epic status.

This removes the hardcoded `active`/`completed` assumptions and makes epic rollup match feature rollup semantics.

#### 2.3.3 Workflow helper usage

The implementation should reuse existing `workflow.Service` APIs as much as possible:

- `GetStatusMetadata(status)` to inspect the current status metadata
- `GetValidTransitions(currentStatus)` to resolve the next configured step
- `IsTerminalStatus(status)` for terminal classification
- `ForLevel(...)` to ensure feature and epic logic each use the correct workflow level

If a helper is added, it should be small and level-agnostic, for example:

- `HasOrchestratorAction(status, action string) bool`
- `GetSingleNextStatus(currentStatus string) (string, bool)`

No helper should hardcode feature/epic status names.

#### 2.3.4 E07-F38 interaction

Forward auto-advance can move a parent from `active` to a post-execution non-terminal status such as `ready_for_code_review`. After that:

- a child regression must still reopen the parent via the existing E07-F38 cascade;
- the reopen target should remain history-driven per E07-F38;
- this feature must not add logic that traps the parent in a terminal state or short-circuits the reopen hooks.

The main compatibility risk is that forward auto-advance changes the "last meaningful non-terminal status" sequence for parent entities. That is acceptable and desirable, as long as reopen continues to use history rather than hardcoded aggregation status.

### 2.4 Interface Contracts

No new external contracts are required.

Internal behavior contracts:

- Feature rollup status derivation becomes `(current feature status, task terminality, workflow metadata) -> next feature status`.
- Epic rollup status derivation becomes `(current epic status, feature terminality, workflow metadata) -> next epic status`.
- The derivation functions must be pure with respect to status computation so they remain easy to unit test.

### 2.5 Testing Strategy

Required test additions or updates:

- Feature progress tests covering:
  - `cascade` status + incomplete tasks => stay put
  - `cascade` status + all terminal tasks => advance one configured step
  - non-`cascade` status + all terminal tasks => no auto-advance
  - simple workflow compatibility where the next step is terminal
  - extended workflow compatibility where the next step is non-terminal
  - `status_override=true` remains untouched

- Epic status tests covering:
  - `cascade` status + incomplete child features => stay put
  - `cascade` status + all terminal child features => advance one configured step
  - non-`cascade` status + all terminal child features => no auto-advance
  - simple and extended workflow variants

- Regression tests covering E07-F38 interaction:
  - parent auto-advances forward, then child regresses, and reopen still works
  - terminal parent gets new child work and reopen still works

Preferred test locations:

- `internal/services/feature_progress_service_test.go`
- `internal/services/epic_service_test.go`
- existing E07-F38-related tests in `internal/services/task_service_test.go`, `internal/services/feature_service_test.go`, or `internal/services/cascade_reopen_test.go` if integration coverage is best placed there

### 2.6 Technical Decisions

**Decision 1: Keep forward auto-advance in existing rollup services.**
This feature changes rollup semantics, so the correct insertion point is the existing rollup logic rather than the transition services or repositories.

**Decision 2: Use workflow metadata, not status names.**
`cascade` is the semantic trigger. Status labels like `active` are implementation details of the current workflow profiles and must not stay embedded in derivation logic.

**Decision 3: Advance one step only.**
Rollup logic should not recursively walk multiple workflow stages. If a richer workflow wants post-execution review, the parent must land on the next configured status and let the orchestrator/human process continue from there.

**Decision 4: Preserve reopen logic as the source of truth for backward movement.**
E07-F38 already solved reopen semantics. E07-F44 should only correct forward movement and must not fork a second reopen implementation.

## 3. Implementation Notes

- Check for an existing `spec.md` before implementation; none existed when this spec was written.
- Register this document in Shark immediately after creation.
- Implementation should follow `CLAUDE.md` service-layer guidance and must finish with `make fmt && make lint && make test`.
