---
feature_key: E16-F03-display-and-aggregation-threshold
epic_key: E16
title: Display and Aggregation Threshold
description: Update get/status commands to respect is_planning vs aggregation mode
priority: P1
---

# Display and Aggregation Threshold

**Feature Key**: E16-F03-display-and-aggregation-threshold
**Priority**: P1

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

Currently, `shark epic get` and `shark feature get` always show aggregated progress from children (tasks/features). But when an entity is in a planning state (e.g., `in_refinement`), it may have no children yet, and showing empty progress bars is misleading. There's no visual distinction between "entity is being planned" and "entity is being built."

### Solution

Implement the **aggregation threshold** concept: entities in planning states (`is_planning: true`) display their own workflow status and position; entities in aggregation states (`is_planning: false`, marked with `_aggregation_`) display aggregated child progress (current behavior). The `_aggregation_` special status is the bridge between planning and execution.

### Impact

- Clear visual distinction between planning and execution phases
- Entities in planning show workflow position instead of empty progress
- All existing aggregation behavior preserved for `active` entities

---

## User Stories

### Must-Have Stories

**Story 1**: As a project manager, when I view an epic in a planning state, I want to see its workflow position and phase instead of aggregated progress.

**Acceptance Criteria**:
- [ ] `shark epic get E16` shows status, phase, and workflow position when `is_planning: true`
- [ ] No progress bars or task counts shown during planning
- [ ] Message indicates "No features yet (epic is still being refined)" or similar

**Story 2**: As a project manager, when I view an epic in `active` state, I want to see aggregated progress from features/tasks (current behavior unchanged).

**Acceptance Criteria**:
- [ ] `shark epic get E16` shows feature list with progress bars when status is `active`
- [ ] Task counts and progress percentages displayed as today
- [ ] Features in planning states show `(planning)` instead of progress bar

**Story 3**: As a project manager, when I view a feature in a planning state, I want to see its workflow position.

**Acceptance Criteria**:
- [ ] `shark feature get E16-F01` shows status, phase, workflow position when `is_planning: true`
- [ ] No task progress shown during planning

**Story 4**: As a project manager, when I view a feature in `active` state, I want to see aggregated task progress (current behavior unchanged).

**Acceptance Criteria**:
- [ ] Current aggregation behavior preserved exactly

---

## Requirements

### Functional Requirements

**Category: Aggregation Threshold (FR-6)**

1. **REQ-F-001**: `_aggregation_` special status support
   - **Description**: New key in `special_statuses` identifies which statuses trigger aggregation mode
   - **Priority**: Must-Have

2. **REQ-F-002**: `is_planning` metadata field
   - **Description**: Status metadata includes `is_planning: true/false` to control display behavior
   - **Priority**: Must-Have

3. **REQ-F-003**: `aggregates_from` metadata field
   - **Description**: Status metadata can include `aggregates_from: "features"` or `aggregates_from: "tasks"` to indicate child type
   - **Priority**: Must-Have

4. **REQ-F-004**: Epic get - planning mode display
   - **Description**: When epic status has `is_planning: true`, show workflow position, phase, description. Do NOT show aggregated progress.
   - **Priority**: Must-Have

5. **REQ-F-005**: Epic get - aggregation mode display
   - **Description**: When epic status has `is_planning: false`, show aggregated progress (current behavior). Features in planning states show `(planning)` label.
   - **Priority**: Must-Have

6. **REQ-F-006**: Feature get - planning mode display
   - **Description**: Same as epic but for features
   - **Priority**: Must-Have

7. **REQ-F-007**: Feature get - aggregation mode display
   - **Description**: Current behavior preserved for `active` features
   - **Priority**: Must-Have

8. **REQ-F-008**: Status command - respect planning/aggregation
   - **Description**: `shark status E16` and `shark status E16-F01` use the same planning vs aggregation logic
   - **Priority**: Must-Have

9. **REQ-F-009**: JSON output includes planning/aggregation metadata
   - **Description**: JSON responses include `is_planning`, `phase`, and `workflow_position` fields
   - **Priority**: Must-Have

---

### Non-Functional Requirements

1. **REQ-NF-001**: Aggregation calculations (when in `active` state) are unchanged from current behavior
2. **REQ-NF-002**: Planning mode display adds no additional database queries

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Epic in Planning Phase**
- **Given** epic E16 at status `in_refinement`
- **When** `shark epic get E16`
- **Then** shows workflow position: `draft -> research -> [in_refinement] -> decomposition -> active -> completed`
- **And** shows "No features yet (epic is still being refined)"

**Scenario 2: Epic in Aggregation Phase**
- **Given** epic E16 at status `active` with 3 features
- **When** `shark epic get E16`
- **Then** shows feature list with progress bars (current behavior)
- **And** features in planning states show `(planning)` instead of progress

**Scenario 3: Feature in Planning Phase**
- **Given** feature E16-F01 at status `ready_for_refinement_tech`
- **When** `shark feature get E16-F01`
- **Then** shows workflow position and phase
- **And** no task progress shown

**Scenario 4: Feature in Aggregation Phase**
- **Given** feature E16-F01 at status `active` with tasks
- **When** `shark feature get E16-F01`
- **Then** shows aggregated task progress (current behavior, unchanged)

**Scenario 5: Default Behavior (No Custom Workflow)**
- **Given** no `epic_workflow` in config
- **When** `shark epic get E07` (existing epic at `active`)
- **Then** shows current behavior exactly (aggregated progress)

---

## Out of Scope

1. **Workflow position visualization** - Detailed visual is in E16-F06
2. **Auto-transition to active** - Manual transition required

---

## Dependencies & Integrations

### Dependencies

- **E16-F01**: Core Workflow Engine (provides `is_planning`, `_aggregation_`, and `aggregates_from` metadata)
- **internal/status/**: Existing status calculation package to extend

---

*Last Updated*: 2026-02-08
