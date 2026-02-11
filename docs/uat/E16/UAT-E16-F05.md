# UAT Test Guide - Backward Transition and Escalation

**Feature:** E16-F05 - Backward Transition and Escalation
**Epic:** E16 - Multi-Level Workflow System
**Generated:** 2026-02-11
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management.

**This Feature's Role:** Provides explicit, audited backward transition tooling (`set-status` commands) with `--reason` guards for epics and features. Enables design-flaw discovery at task level to surface at feature/epic level without automatic child status changes.

**Related Features:**
- E16-F01 Core Workflow Engine (completed) - Provides workflow validation and phase metadata
- E16-F02 Orchestrator Actions (completed) - Action metadata in transition responses
- E16-F03 Display & Aggregation Threshold (completed) - Planning vs aggregation display
- E16-F04 Notes and Context (active) - Note storage for logging reasons (soft dependency)
- E16-F06 Workflow Visualization (draft) - Future visualization

**Integration Points:**
- Uses `workflow.Service.IsBackwardTransition()` from E16-F01's core engine
- Uses `WorkflowConfig.IsBackwardTransition()` from config package (phase ordering)
- Reuses `performEntityTransition()` from E16-F02's next-status commands
- New `set-status` commands share `TransitionOptions`/`TransitionResult` types with `next-status`

---

## Design Intent

**From Epic PRD (FR-9):**
> When moving a feature BACKWARD (from `active` to a planning status like `ready_for_refinement_tech`), require `--reason` flag. Log the reason in feature history/notes. Do NOT automatically change child task statuses.

**From Feature PRD:**
> Add `shark feature set-status` and `shark epic set-status` commands that allow any valid status transition (with workflow validation) but require a `--reason` flag when moving backward. Child entities remain unchanged (orchestrator decides what to do with them).

**Key Design Decisions:**
- Backward detection uses phase ordering from `validation.PhaseOrder` map
- `--force` also requires `--reason` to document overrides
- `TransitionOptions` struct replaces `force bool` for richer transition control
- Child count is reported in results but child statuses are NOT changed
- JSON output uses `omitempty` for backward-specific fields on forward transitions

---

## Test Scenarios

### Scenario 1: TransitionOptions and TransitionResult Types (T-001)

**Spec requirement:** TransitionOptions struct with Force, Reason, Agent fields. TransitionResult with IsBackward, IsForced, Reason, ChildCount fields. JSON tags with omitempty for backward-specific fields.

**Verification:** Unit tests for type definitions and JSON serialization.

### Scenario 2: Backward Transition Detection (T-002)

**Spec requirement:** `workflow.Service.IsBackwardTransition()` compares phase ordering to detect when a transition moves to an earlier phase.

**Verification:** Service method delegates to WorkflowConfig which uses PhaseOrder map.

### Scenario 3: EpicService Backward Transition Guards (T-003)

**Spec requirement:** Epic backward transitions require `--reason`. Forward transitions do not. `--force` requires `--reason`.

**Verification:** Service-level tests with mock repos and test workflow configs.

### Scenario 4: FeatureService Backward Transition Guards (T-004)

**Spec requirement:** Same backward transition guards for features as epics.

**Verification:** Service-level tests with mock repos and test workflow configs.

### Scenario 5: CLI next-status Commands with Reason Flag (T-005)

**Spec requirement:** `--reason` and `--force` flags on `shark epic next-status` and `shark feature next-status`.

**Verification:** CLI command flag registration and mock-based tests.

### Scenario 6: Epic set-status CLI Command (T-006)

**Spec requirement:** `shark epic set-status <epic-key> <status>` with `--reason`, `--force`, `--agent` flags.

**Verification:** Command registration, arg validation, flag presence.

### Scenario 7: Feature set-status CLI Command (T-007)

**Spec requirement:** `shark feature set-status <feature-key> <status>` with same flags as epic.

**Verification:** Command registration, arg validation, flag presence.

### Scenario 8: Integration Tests and Documentation (T-008)

**Spec requirement:** Cross-cutting integration tests covering backward detection, reason enforcement, child counting, JSON serialization. CLI reference docs updated.

**Verification:** 12 integration test functions all passing. Documentation updated.

---

## Epic Acceptance Validation

| Epic AC | Description | Feature Contribution | Status |
|---------|-------------|---------------------|--------|
| FR-9.1 | `shark feature set-status` allows valid status transitions | Feature set-status command created | [ ] |
| FR-9.2 | Backward transitions require `--reason` | Backward detection + reason enforcement | [ ] |
| FR-9.3 | Reason logged in history/notes | TransitionResult includes reason field | [ ] |
| FR-9.4 | Child task statuses unchanged | Child count reported but statuses not modified | [ ] |

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| Story 1 | Tech lead can move feature backward with documented reason | [ ] |
| Story 2 | Backward transitions require `--reason`, forward do not | [ ] |
| Story 3 | `shark epic set-status` has same backward guards | [ ] |
| Error 1 | Clear error when backward transition missing reason | [ ] |

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-11 |
| Result | ALL PASS |
| Results File | results/UAT-E16-F05-20260211-193500-results.md |

**Previous Sessions:**
- 2026-02-11: ALL PASS (8/8 scenarios)
