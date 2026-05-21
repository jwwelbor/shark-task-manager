---
feature_key: E07-F44
title: Test Plan — Cascade action auto-advance for parent workflows
status: in_test_planning
last_updated: 2026-05-19
references:
  - spec.md
  - feature.md
  - ../E07-F38-auto-reopen-parent-entities-on-child-regression/spec.md
  - ../E07-F38-auto-reopen-parent-entities-on-child-regression/uat-plan.md
---

# E07-F44 Test Plan: Cascade Action Auto-Advance for Parent Workflows

Every test in this plan traces to one or more acceptance criteria in `spec.md`. There is no parent epic `uat-plan.md` under `E07`; the reopen interaction scenarios instead trace to E07-F38's spec and UAT plan, which already define the backward-regression behaviors this feature must preserve.

## 1. AC Test Matrix

### AC-01 / AC-02 / AC-03 / AC-04 / AC-05 — Feature rollup auto-advance semantics

**Requirements:** REQ-F-001 through REQ-F-005, REQ-F-007, REQ-N-003

| Field | Value |
|-------|-------|
| Test names | `TestFeatureProgressService_RecalculateAndSetProgress_CascadeStatusDerivation`, `TestFeatureProgressService_RecalculateAndSetProgress_NonCascadeStatusPreserved` |
| File | `internal/services/feature_progress_service_test.go` |
| Type | Unit/service test with real `FeatureProgressService` and test workflow config |

**Setup:**
- Feature starts in a status whose metadata action is either `cascade` or non-`cascade`.
- Task breakdown is provided through the existing feature progress test harness.
- Variants cover all-terminal tasks, mixed terminal/non-terminal tasks, and `status_override=true`.

**Caller-path contract:**
- Production entrypoint: `FeatureService.RecalculateAndSetProgress(ctx, featureID)` leading into `FeatureProgressService.RecalculateAndSetProgress(...)`.
- Lowest allowed mock seam: feature/task repository interfaces already used by `feature_progress_service_test.go`.
- Forbidden mocks: direct testing of `deriveFeatureProgressStatus(...)` with status combinations that cannot be produced by the production progress recalculation path.
- Counter-factual: a buggy implementation would still return `completed` whenever weighted progress reaches 100%, even if the current status is non-`cascade` or the workflow's next step is non-terminal.

**Expected outcomes:**
- `cascade` status + incomplete tasks => parent stays in current status.
- `cascade` status + all tasks terminal => parent advances exactly one configured step.
- simple profile variant => the configured next step is terminal, so the parent becomes terminal.
- extended profile variant => the configured next step is a review/QA/approval stage, so the parent lands there instead of `completed`.
- non-`cascade` status + all tasks terminal => status unchanged.
- `status_override=true` => status unchanged regardless of progress.

**Edge cases:**
- Zero-task feature remains unchanged.
- Already terminal but non-`completed` manual terminal status stays preserved.
- Multiple valid transitions from the current status must be treated as configuration smell; the test should assert the implementation consumes the first configured next step consistently or explicitly guards against ambiguous multi-next-step states.

### AC-06 / AC-07 / AC-08 — Epic rollup auto-advance semantics

**Requirements:** REQ-F-001 through REQ-F-004, REQ-F-006, REQ-F-007

| Field | Value |
|-------|-------|
| Test names | `TestEpicService_RecalculateStatus_CascadeAutoAdvance`, `TestDeriveEpicStatusFromFeatures_CascadeVsNonCascade` |
| File | `internal/services/epic_service_test.go` and/or `internal/services/epic_analytics_service.go` companion tests |
| Type | Unit/service test |

**Setup:**
- Epic begins in either a `cascade` or non-`cascade` status.
- Child feature status breakdown covers all-terminal and mixed-terminal cases.
- Workflow variants cover both simple and extended parent flows.

**Caller-path contract:**
- Production entrypoint: `EpicService.RecalculateStatus(ctx, epicID)`.
- Lowest allowed mock seam: `EpicRepository.GetByID`, `GetFeatureStatusBreakdown`, and `UpdateStatus`, matching the existing `epic_service_test.go` pattern.
- Forbidden mocks: bypassing `RecalculateStatus(...)` and testing only `deriveEpicStatusFromFeatures(...)` with impossible current-status/workflow combinations.
- Counter-factual: a buggy implementation would still collapse "all child features terminal" directly to `completed` or `active`, ignoring the current status action and the workflow-configured next step.

**Expected outcomes:**
- `cascade` status + incomplete child features => epic stays in current status.
- `cascade` status + all child features terminal => epic advances one configured step.
- extended workflow => epic can move from `active` to a non-terminal post-execution status.
- non-`cascade` status => "all terminal children" does not force advancement.

**Edge cases:**
- Zero-feature epic remains unchanged.
- Archived/cancelled child features count as terminal only when the active workflow says they are terminal.
- Status override on epic, if surfaced in the recalculation path, is preserved.

### AC-09 / AC-10 — E07-F38 reopen compatibility after forward auto-advance

**Requirements:** REQ-F-008, REQ-N-004

| Field | Value |
|-------|-------|
| Test names | `TestCascade_ForwardAutoAdvanceThenTaskRegressionReopensParent`, `TestCascade_NewChildUnderForwardAdvancedParentStillReopens` |
| File | extend existing E07-F38 regression surfaces in `internal/services/cascade_reopen_test.go`, `internal/services/task_service_test.go`, or `internal/services/feature_service_test.go` |
| Type | Integration-style service tests |

**Setup:**
- Parent feature or epic first reaches a `cascade` status and is auto-advanced forward by the new E07-F44 logic.
- Then either a child regresses out of terminal or a new child is created beneath a terminal parent.

**Caller-path contract:**
- Production entrypoints:
  - regression path: `TaskService.TransitionStatus(...)` or `FeatureService.TransitionStatus(...)`
  - creation path: `TaskService.CreateTask(...)` or `FeatureService.CreateFeature(...)`
- Lowest allowed mock seam: the same cascade dependency seams already used in E07-F38 tests.
- Forbidden mocks: direct calls to `cascadeParentReopens(...)` without first establishing the forward-advanced status history that this feature changes.
- Counter-factual: a buggy implementation would auto-advance forward successfully but would either fail to reopen later, reopen to the wrong prior status, or leave the parent terminal after new child work appears.

**Expected outcomes:**
- Parent reopens correctly after a child regression even if its most recent pre-regression state came from E07-F44 forward auto-advance.
- Creation-based reopen remains functional and history-driven.

**Edge cases:**
- Feature leg already reopened, epic still terminal.
- Reopen after the parent advanced into a non-terminal review status rather than `completed`.

### AC-11 — Status override remains authoritative

**Requirements:** REQ-N-003

| Field | Value |
|-------|-------|
| Test name | `TestFeatureProgressService_RecalculateAndSetProgress_StatusOverrideBeatsCascadeAutoAdvance` |
| File | `internal/services/feature_progress_service_test.go` |
| Type | Unit/service test |

**Caller-path contract:**
- Production entrypoint: `FeatureService.RecalculateAndSetProgress(...)`.
- Lowest allowed mock seam: existing feature progress test harness.
- Forbidden mocks: direct field mutation on the output feature after recalculation.
- Counter-factual: a buggy implementation would respect `status_override` for reopen-on-drop but still auto-advance on 100% progress.

**Expected outcome:**
- Even with all child tasks terminal and current status action `cascade`, a feature with `status_override=true` keeps its existing status.

**Edge cases:**
- Overridden status itself is terminal.
- Overridden status is a custom non-terminal status outside the default profile.

### AC-12 — Full workflow/profile coverage

**Requirements:** REQ-F-007, REQ-N-004, REQ-N-005

| Field | Value |
|-------|-------|
| Test names | table-driven variants embedded in the feature/epic tests above |
| File | same as above |
| Type | Unit/service plus final repo-wide quality gate |

**Caller-path contract:**
- Production entrypoints: `FeatureService.RecalculateAndSetProgress(...)`, `EpicService.RecalculateStatus(...)`, `TaskService.TransitionStatus(...)`, `TaskService.CreateTask(...)`, `FeatureService.CreateFeature(...)`.
- Lowest allowed mock seam: existing service/repository seams already used by the surrounding test files.
- Forbidden mocks: helper-only test workflows that omit the actual status metadata fields (`orchestrator_action`, terminal status groups, valid transitions) the production code consumes.
- Counter-factual: a buggy implementation would pass on the default profile but fail under a richer profile because it implicitly assumes `active -> completed`.

**Expected outcome:**
- Tests cover both simple and extended workflow shapes.
- Final implementation passes `make fmt && make lint && make test`.

**Edge cases:**
- Custom workflow where the `cascade` status is not named `active`.
- Custom workflow where terminal states are not named `completed`.

## 2. Integration Scenarios

### Scenario A: Feature progress path -> workflow metadata

- Components: `FeatureProgressService`, task workflow metadata, feature workflow metadata, feature repository update.
- Verify:
  - terminality still comes from task workflow;
  - parent advancement decision comes from feature workflow status metadata;
  - only one step of parent advancement occurs.
- Epic/UAT linkage: supports E07-F44 AC-01..AC-05.

### Scenario B: Epic status path -> workflow metadata

- Components: `EpicService.RecalculateStatus`, `deriveEpicStatusFromFeatures`, epic workflow metadata, feature status breakdown repository.
- Verify:
  - epic does not hardcode `active`/`completed`;
  - epic honors `cascade` action semantics and next-step resolution.
- Epic/UAT linkage: supports E07-F44 AC-06..AC-08.

### Scenario C: Forward auto-advance -> E07-F38 reopen cascade

- Components: E07-F44 rollup logic plus E07-F38 `cascadeParentReopens` and creation-trigger reopen paths.
- Verify:
  - forward auto-advance changes the parent's recorded status history;
  - backward reopen still resolves correctly from history;
  - new child creation still reopens terminal parents.
- Epic/UAT linkage: maps to E07-F38 UAT-S1, UAT-S2, and UAT-S6.

## 3. Test Infrastructure

Existing patterns to follow:

- `internal/services/feature_progress_service_test.go` for progress-driven status derivation tests.
- `internal/services/epic_service_test.go` for epic recalculation and workflow-aware status tests.
- `internal/services/cascade_reopen_test.go` for E07-F38-compatible reopen behavior and workflow-profile-sensitive cascade tests.
- `CLAUDE.md` testing rule: repository tests use real DB; non-repository service tests use mocks or lightweight harnesses.

New helpers likely needed:

- A small workflow fixture/helper for feature and epic tests that can express:
  - current status metadata action = `cascade` vs non-`cascade`
  - next configured transition terminal vs non-terminal
  - custom status names for profile-independence checks
- If one already exists in workflow/service tests, reuse it rather than inventing a second DSL.

No new DB fixtures should be required unless an existing E07-F38 regression test is promoted from mock/service level to repository-backed integration coverage.

## 4. Exit Gate Checklist

- Every AC in `spec.md` has at least one mapped test case.
- Every mapped test case includes a caller-path contract.
- Feature rollup, epic rollup, and E07-F38 interaction coverage are all present.
- Existing test surfaces and file paths are identified.
- The final implementation gate remains `make fmt && make lint && make test`.
