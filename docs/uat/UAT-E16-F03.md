# UAT Test Guide - Display and Aggregation Threshold

**Feature:** E16-F03 - Display and Aggregation Threshold
**Epic:** E16 - Multi-Level Workflow System
**Generated:** 2026-02-09
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management.

**This Feature's Role:** Implements the aggregation threshold concept -- entities in planning states show their own workflow position; entities in aggregation states show aggregated child progress. This is the display/UI layer that makes the multi-level workflow visible to users and orchestrators.

**Related Features:**
- E16-F01: Core Workflow Engine (provides config parsing, `next-status` commands) - active
- E16-F02: Orchestrator Actions (returns `orchestrator_action` in transition responses) - active
- E16-F04: Notes and Context for Epic and Feature - draft
- E16-F05: Backward Transition and Escalation - draft
- E16-F06: Workflow Visualization - draft

**Integration Points:**
- Depends on E16-F01 for `is_planning`, `_aggregation_`, and `aggregates_from` config metadata
- Depends on E16-F02 for orchestrator action response format in JSON output
- Status dashboard (`shark status`) integrates with both planning mode display and existing aggregation display

---

## Design Intent

**From Epic PRD:**
> When an entity reaches an aggregation status (marked with `_aggregation_` or `aggregates_from`), its display behavior switches from showing its own workflow status to showing aggregated child progress.
> Before `active`: The entity has its own workflow. `shark epic get E16` shows status `in_refinement`, not progress bars.
> At/after `active`: The entity aggregates from children (current behavior, unchanged).

**From Feature PRD:**
> Implement the aggregation threshold concept: entities in planning states (`is_planning: true`) display their own workflow status and position; entities in aggregation states (`is_planning: false`, marked with `_aggregation_`) display aggregated child progress (current behavior).

**Key Design Decisions:**
- Planning mode shows workflow position, phase, and description -- no progress bars
- Aggregation mode is the current behavior, completely unchanged
- When no custom workflow is configured, default to aggregation (100% backward compatible)
- Features in planning states show `(planning)` label in parent epic's aggregation view
- `_aggregation_` special status key takes precedence over `is_planning` metadata field
- JSON output includes `display_mode`, `phase`, `workflow_position` fields with `omitempty`

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| Scenario 1 | Epic in planning phase shows workflow position, no progress | [ ] |
| Scenario 2 | Epic in aggregation phase shows feature progress (current behavior) | [ ] |
| Scenario 3 | Feature in planning phase shows workflow position, no tasks | [ ] |
| Scenario 4 | Feature in aggregation phase shows task progress (current behavior) | [ ] |
| Scenario 5 | Default behavior (no custom workflow) shows current behavior | [ ] |

---

## Test Scenarios

### Scenario 1: Backward Compatibility - No Custom Workflows
**Tasks covered:** T-E16-F03-001, T-E16-F03-004, T-E16-F03-007

**Purpose:** Verify that when no `epic_workflow` or `feature_workflow` is configured, all display behavior is identical to current behavior (aggregation mode).

**Steps:**
1. Verify DisplayService defaults to aggregation when no workflow configured
2. Run full test suite to confirm no regressions
3. Verify `shark epic get` for existing active epics shows aggregated progress

**Success Criteria:**
- [ ] `DetermineEpicDisplayMode` returns `aggregation` when epic_workflow is nil
- [ ] `DetermineFeatureDisplayMode` returns `aggregation` when feature_workflow is nil
- [ ] Existing `shark epic get` output unchanged for active epics
- [ ] Full test suite passes (0 failures)

### Scenario 2: Config Model - `is_planning` and `_aggregation_` Fields
**Tasks covered:** T-E16-F03-002, T-E16-F03-003

**Purpose:** Verify the config model correctly parses `is_planning`, `aggregates_from`, and `_aggregation_` special status fields.

**Steps:**
1. Check StatusMetadata struct has `IsPlanning` and `AggregatesFrom` fields
2. Verify `AggregationStatusKey` constant exists
3. Verify default epic and feature workflows use these fields correctly
4. Verify WorkflowConfig struct has StatusFlow, StartStatus, SpecialStatuses

**Success Criteria:**
- [ ] `IsPlanning bool` field on StatusMetadata with `json:"is_planning,omitempty"` tag
- [ ] `AggregatesFrom string` field on StatusMetadata with `json:"aggregates_from,omitempty"` tag
- [ ] `AggregationStatusKey = "_aggregation_"` constant defined
- [ ] Default epic workflow: `draft` has `is_planning: true`, `active` has `aggregates_from: "features"`
- [ ] Default feature workflow: `draft` has `is_planning: true`, `active` has `aggregates_from: "tasks"`
- [ ] WorkflowConfig has StatusFlow, StartStatus, SpecialStatuses fields

### Scenario 3: DisplayService Core Logic - Mode Determination
**Tasks covered:** T-E16-F03-004, T-E16-F03-007

**Purpose:** Verify the DisplayService correctly determines planning vs aggregation mode based on workflow configuration.

**Steps:**
1. Verify DisplayService constructor and methods exist
2. Test mode determination with planning statuses
3. Test mode determination with aggregation statuses
4. Verify `_aggregation_` key takes precedence over `is_planning`
5. Verify workflow position building with ordered statuses

**Success Criteria:**
- [ ] `DetermineEpicDisplayMode` returns `planning` for statuses with `is_planning: true`
- [ ] `DetermineEpicDisplayMode` returns `aggregation` for statuses in `_aggregation_` list
- [ ] `_aggregation_` key precedence: a status with `is_planning: true` AND in `_aggregation_` list returns `aggregation`
- [ ] `BuildWorkflowPosition` returns ordered status list with current index
- [ ] `GetEpicDisplayInfo` and `GetFeatureDisplayInfo` return complete display info

### Scenario 4: Epic Get - Planning Mode Rendering
**Tasks covered:** T-E16-F03-002

**Purpose:** Verify that `shark epic get` renders planning mode correctly (workflow position, no progress bars).

**Steps:**
1. Read the epic get command's planning mode renderer code
2. Verify it shows status with "(workflow)" label, phase, workflow position
3. Verify it shows "No features yet" message
4. Verify JSON output includes display_mode, phase, workflow_position

**Success Criteria:**
- [ ] Planning mode renderer exists in epic.go (`renderEpicPlanning`)
- [ ] Shows status, phase, and workflow position in human-readable output
- [ ] JSON output includes `display_mode: "planning"`, `phase`, `workflow_position` fields
- [ ] No progress bars or task counts in planning mode output

### Scenario 5: Feature Get - Planning Mode Rendering
**Tasks covered:** T-E16-F03-003

**Purpose:** Verify that `shark feature get` renders planning mode correctly.

**Steps:**
1. Read the feature get command's planning mode renderer code
2. Verify it shows status with "(workflow)" label, phase, workflow position
3. Verify no task progress shown in planning mode
4. Verify JSON output includes display_mode, phase, workflow_position

**Success Criteria:**
- [ ] Planning mode renderer exists in feature.go (`renderFeaturePlanning`)
- [ ] Shows status, phase, workflow position in human-readable output
- [ ] JSON output includes `display_mode: "planning"`, `phase`, `workflow_position` fields
- [ ] No task counts, progress bars, or work breakdown in planning mode

### Scenario 6: Calculation Service Planning Mode Bypass
**Tasks covered:** T-E16-F03-005

**Purpose:** Verify that status recalculation is skipped for entities in planning mode.

**Steps:**
1. Read calculation_service.go planning mode bypass logic
2. Verify `isFeatureInPlanningMode` and `isEpicInPlanningMode` helpers exist
3. Verify RecalculateFeatureStatus and RecalculateEpicStatus skip derivation for planning entities
4. Verify backward compat: nil workflow returns false (aggregation)

**Success Criteria:**
- [ ] `isFeatureInPlanningMode()` checks `is_planning` from workflow config
- [ ] `isEpicInPlanningMode()` checks `is_planning` from workflow config
- [ ] Planning entities return `StatusChangeResult{WasSkipped: true}`
- [ ] Nil workflow returns false (no planning mode bypass)

### Scenario 7: Status Dashboard Integration
**Tasks covered:** T-E16-F03-006

**Purpose:** Verify the status dashboard correctly enriches epic summaries with planning mode information.

**Steps:**
1. Read status.go enrichEpicSummaries function
2. Verify EpicSummary model has DisplayMode, IsPlanning, Phase fields
3. Read formatter.go formatEpicTable planning mode handling
4. Verify JSON output includes planning fields with omitempty

**Success Criteria:**
- [ ] `enrichEpicSummaries` calls DisplayService for each epic
- [ ] EpicSummary has `DisplayMode`, `IsPlanning`, `Phase` fields with `omitempty` tags
- [ ] Formatter shows `(planning)` label and phase for planning-mode epics
- [ ] Aggregation-mode epics show progress bars as before

### Scenario 8: Integration Test Coverage
**Tasks covered:** T-E16-F03-007

**Purpose:** Verify comprehensive test coverage for all planning/aggregation scenarios.

**Steps:**
1. Run full test suite
2. Verify integration test count and coverage areas
3. Verify edge case coverage (empty status, circular flow, unknown statuses)

**Success Criteria:**
- [ ] All tests pass (0 failures)
- [ ] 27+ integration tests in display_service_integration_test.go
- [ ] Backward compatibility tests (nil workflows)
- [ ] Mixed mode tests (epic aggregating, features varied)
- [ ] Threshold crossing tests (bidirectional)
- [ ] Edge case tests (empty status, case sensitivity, aggregation precedence)

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-09 |
| Result | PASS (8/8 scenarios) |
| Results File | docs/uat/results/UAT-E16-F03-20260209-session1.md |

**Previous Sessions:**
- 2026-02-09: PASS (8/8) - All scenarios passed, full feature acceptance validated
