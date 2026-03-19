---
feature_key: E20-F03-task-workflow-standardization
epic_key: E20
title: Task Workflow Standardization
description: Standardize task workflow to use the same {entity}_workflow block pattern as epic, feature, bug, and change workflows.
---

# Task Workflow Standardization

**Feature Key**: E20-F03-task-workflow-standardization

---

## Epic

- **Epic PRD**: [Shark Templates](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)

---

## Goal

### Problem

Task workflow configuration uses a legacy structure that differs from the four other entity types. Epic, feature, bug, and change workflows each use a dedicated `{entity}_workflow` block (introduced in E16) with consistent sub-keys: `version`, `status_flow`, `status_metadata`, `orchestrator_actions`, and `special_statuses`. Task workflow, however, still uses top-level keys (`status_flow`, `status_metadata`, `special_statuses`, `status_flow_version`) in `.sharkconfig.json`.

This inconsistency forces the config loading code in `internal/config/workflow_parser.go` to maintain two separate parsing paths: `parseWorkflowSection()` for the four consistent entity types, and `parseTopLevelTaskWorkflow()` for the task-specific legacy format. This bifurcation complicates generic tooling (E21 Entity Polymorphism), makes the config harder to understand, and is a source of subtle bugs when extending workflow features.

### Solution

Add `task_workflow` block parsing to `LoadMultiLevelWorkflow()` in `workflow_parser.go`, using the same `parseWorkflowSection()` function already used for the other four entity types. When a `task_workflow` block is present, it takes precedence over the legacy top-level keys. The legacy `parseTopLevelTaskWorkflow()` function is retained as a backward-compatible fallback for existing configurations that have not adopted the block format.

### Impact

- All five entity types use an identical config structure, eliminating the task-specific special case.
- Config loading code is simplified: one parsing function handles all entity types.
- E21 (Entity Polymorphism) can write generic workflow-loading code without task-specific branches.
- Existing configurations continue to work without changes (legacy fallback preserved).

---

## Epic Requirement Mapping

| Epic Requirement | Coverage |
|------------------|----------|
| **REQ-F-005** (Consistent Task Workflow Block) | Full -- this is the primary requirement for this feature |
| **REQ-NF-001** (Zero Breaking Changes) | Partial -- backward compatibility for task workflow specifically |
| **REQ-NF-003** (Single Config Loading Code Path) | Partial -- eliminates the task bifurcation in the parsing path |
| **REQ-NF-004** (Test Isolation) | Partial -- new tests use fixtures, not production config |

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer editing workflow configuration, I want task workflow to use the same `task_workflow` block structure as other entities so that I can apply the same mental model and tooling to all entity types.

**Acceptance Criteria**:
- [ ] A `task_workflow` block with `version`, `status_flow`, `status_metadata`, `orchestrator_actions`, and `special_statuses` sub-keys is parsed by `LoadMultiLevelWorkflow()`
- [ ] The structure is identical in shape to `epic_workflow`, `feature_workflow`, `bug_workflow`, `change_workflow`
- [ ] When `task_workflow` block is present, it populates `MultiLevelWorkflow.Task`

**Story 2**: As a developer with an existing `.sharkconfig.json` using legacy top-level keys, I want the system to continue working without changes so that I do not need to migrate my config immediately.

**Acceptance Criteria**:
- [ ] When no `task_workflow` block is present, the system falls back to `parseTopLevelTaskWorkflow()` for legacy top-level keys
- [ ] Existing commands, status transitions, and template rendering continue to work identically
- [ ] The legacy path produces the same `*WorkflowConfig` as it does today

**Story 3**: As a developer with both `task_workflow` block and legacy top-level keys in my config, I want the block to take precedence so that the migration path is clear.

**Acceptance Criteria**:
- [ ] When `task_workflow` block is present, legacy top-level `status_flow` and `status_metadata` keys are ignored for task workflow resolution
- [ ] The precedence is: `task_workflow` block > legacy top-level keys > built-in defaults

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Task workflow block parsed correctly**
- **Given** a config file containing a `task_workflow` block with all required sub-keys
- **When** `LoadMultiLevelWorkflow()` is called
- **Then** `MultiLevelWorkflow.Task` is populated with the parsed workflow
- **And** the parsed workflow matches the content of the `task_workflow` block

**Scenario 2: Legacy fallback works**
- **Given** a config file with legacy top-level `status_flow` and `status_metadata` keys but no `task_workflow` block
- **When** `LoadMultiLevelWorkflow()` is called
- **Then** `MultiLevelWorkflow.Task` is populated from the legacy keys
- **And** the result is identical to the pre-E20 behavior

**Scenario 3: Block precedence over legacy**
- **Given** a config file with both a `task_workflow` block and legacy top-level keys, where the block defines different statuses
- **When** `LoadMultiLevelWorkflow()` is called
- **Then** `MultiLevelWorkflow.Task` uses the `task_workflow` block data
- **And** legacy top-level keys are ignored

**Scenario 4: Full regression**
- **Given** the current production `.sharkconfig.json` (no `task_workflow` block)
- **When** `LoadMultiLevelWorkflow()` is called after E20-F03 code changes
- **Then** the resolved task workflow is identical to the pre-E20-F03 behavior
- **And** all existing tests pass without modification

---

## Out of Scope

1. **Workflow file externalization** -- Handled by E20-F04. This feature only adds `task_workflow` block parsing within the existing single-file loading path.
2. **Deprecation warnings for legacy keys** -- Handled by E20-F06. This feature silently falls back to legacy keys.
3. **Removing `parseTopLevelTaskWorkflow()`** -- The legacy function is retained for backward compatibility. Removal is a future cleanup task.

---

## Dependencies & Integrations

### Dependencies

- **E16 (Multi-Level Workflow)**: Completed. Provides `parseWorkflowSection()`, `MultiLevelWorkflow` struct, and per-entity workflow block pattern that this feature extends to tasks.

### Downstream Dependents

- **E20-F04**: Builds on this feature. The workflow file loading assumes all five entity types use the block pattern.
- **E20-F05**: Generates workflow files with `task_workflow` block format.
- **E21 (Entity Polymorphism)**: Benefits from consistent workflow structure across all entity types.

---

## Implementation Notes

### Key Files to Modify

- `internal/config/workflow_parser.go` -- Add `task_workflow` block check in `LoadMultiLevelWorkflow()` before the legacy `parseTopLevelTaskWorkflow()` call
- `internal/config/workflow_parser_test.go` / `workflow_multilevel_test.go` -- Add test cases for `task_workflow` block parsing, precedence, and fallback

### Estimated Scope

- ~15-20 lines of new code in `workflow_parser.go`
- ~100-150 lines of new tests
- Complexity: S (Small)

### UAT Scenarios

Maps to UAT acceptance plan scenarios: AS-D01, AS-D02, AS-D03 (Area D: Task Workflow Standardization)

---

*Last Updated*: 2026-03-18
