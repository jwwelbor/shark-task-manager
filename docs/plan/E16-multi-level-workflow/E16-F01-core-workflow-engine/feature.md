---
feature_key: E16-F01-core-workflow-engine
epic_key: E16
title: Core Workflow Engine
description: Parse level-specific configs, validate transitions for epic/feature, next-status commands
priority: P0
---

# Core Workflow Engine

**Feature Key**: E16-F01-core-workflow-engine
**Priority**: P0 (Foundation - all other features depend on this)

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

Shark's configurable workflow system (delivered in E11) applies only to tasks. Epics and features have hardcoded statuses (`draft`, `active`, `completed`, `archived`) with no configurable transitions and no workflow enforcement. This means planning work at the epic/feature level cannot be tracked through a structured workflow, and the `active` status is overloaded to mean both "being planned" and "being built."

### Solution

Extend the workflow engine to support three independent workflow definitions in `.sharkconfig.json`: `epic_workflow`, `feature_workflow`, and the existing task workflow (backward compatible). Each level gets its own status flow, metadata, and validation. Add `next-status` commands for epic and feature entities, mirroring the existing `shark task next-status`.

### Impact

- Enables configurable workflows at all entity levels
- Foundation for end-to-end PDLC orchestration
- All other E16 features depend on this core engine

---

## User Stories

### Must-Have Stories

**Story 1**: As a project manager, I want to configure custom status flows for epics and features in `.sharkconfig.json` so that I can track planning work through a structured workflow.

**Acceptance Criteria**:
- [ ] `epic_workflow` section is parsed from `.sharkconfig.json`
- [ ] `feature_workflow` section is parsed from `.sharkconfig.json`
- [ ] Each workflow is validated independently
- [ ] Invalid configurations produce actionable error messages

**Story 2**: As a project manager, I want `shark epic update --status` to validate transitions against the configured epic workflow so that invalid transitions are caught.

**Acceptance Criteria**:
- [ ] Valid transitions succeed with confirmation message
- [ ] Invalid transitions are rejected with error showing valid options
- [ ] `--force` flag overrides validation

**Story 3**: As a project manager, I want `shark feature update --status` to validate transitions against the configured feature workflow.

**Acceptance Criteria**:
- [ ] Same behavior as epic validation
- [ ] Uses `feature_workflow.status_flow` for validation

**Story 4**: As an orchestrator, I want `shark epic next-status <key>` and `shark feature next-status <key>` to advance entities to the next logical status.

**Acceptance Criteria**:
- [ ] Next status = first entry in `status_flow` array for current status
- [ ] Returns transition details in both human and JSON output
- [ ] Works with `--json` flag

---

### Edge Case & Error Stories

**Error Story 1**: As a user with no custom epic/feature workflow configured, I want the system to use default hardcoded statuses (`draft`, `active`, `completed`, `archived`) so that existing behavior is preserved.

**Acceptance Criteria**:
- [ ] Missing `epic_workflow` uses default statuses
- [ ] Missing `feature_workflow` uses default statuses
- [ ] Existing `shark epic update --status active` works identically to today

**Error Story 2**: As a user, when I attempt an invalid status transition, I want to see valid options so I can correct the command.

**Acceptance Criteria**:
- [ ] Error message lists all valid transitions from current status
- [ ] Suggests `--force` to override

---

## Requirements

### Functional Requirements

**Category: Configuration Parsing (FR-1)**

1. **REQ-F-001**: Parse `epic_workflow` section from `.sharkconfig.json`
   - **Description**: Parse `version`, `status_flow`, `status_metadata`, and `special_statuses` from `epic_workflow`
   - **Priority**: Must-Have

2. **REQ-F-002**: Parse `feature_workflow` section from `.sharkconfig.json`
   - **Description**: Same structure as epic_workflow but for features
   - **Priority**: Must-Have

3. **REQ-F-003**: Backward-compatible task workflow
   - **Description**: Existing top-level `status_flow` / `status_metadata` remains the task workflow, unchanged
   - **Priority**: Must-Have

4. **REQ-F-004**: Default fallback for missing workflows
   - **Description**: If `epic_workflow` is absent, use hardcoded defaults: `draft`, `active`, `completed`, `archived`. Same for `feature_workflow`.
   - **Priority**: Must-Have

5. **REQ-F-005**: Independent validation per level
   - **Description**: Each workflow is validated independently (no cross-references between levels)
   - **Priority**: Must-Have

**Category: Status Transitions (FR-2, FR-3)**

6. **REQ-F-006**: Epic status transition validation
   - **Description**: `shark epic update <key> --status <new>` validates against `epic_workflow.status_flow`
   - **Priority**: Must-Have

7. **REQ-F-007**: Feature status transition validation
   - **Description**: `shark feature update <key> --status <new>` validates against `feature_workflow.status_flow`
   - **Priority**: Must-Have

8. **REQ-F-008**: Force override for all levels
   - **Description**: `--force` flag overrides workflow validation for epic and feature, matching task behavior
   - **Priority**: Must-Have

**Category: Next-Status Commands (FR-4)**

9. **REQ-F-009**: `shark epic next-status <key>`
   - **Description**: Advances epic to the next logical status (first entry in `status_flow` array for current status)
   - **Priority**: Must-Have

10. **REQ-F-010**: `shark feature next-status <key>`
    - **Description**: Advances feature to the next logical status
    - **Priority**: Must-Have

11. **REQ-F-011**: JSON output for next-status
    - **Description**: `--json` returns structured response with entity_type, entity_id, transition (from/to/timestamp)
    - **Priority**: Must-Have

**Category: Workflow Validation (FR-7)**

12. **REQ-F-012**: Extend `shark workflow validate` to cover epic and feature workflows
    - **Description**: Check for unreachable statuses, missing terminal states, orphan statuses per level
    - **Priority**: Must-Have

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Workflow validation for all three levels completes in < 100ms
   - **Measurement**: Wall clock time for `shark workflow validate`

2. **REQ-NF-002**: `next-status` commands for epic/feature have same performance as task `next-status`

**Backward Compatibility**

3. **REQ-NF-003**: Existing `.sharkconfig.json` files without `epic_workflow` or `feature_workflow` work unchanged
4. **REQ-NF-004**: `draft -> active` shortcut is always valid at all levels

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Parse Custom Epic Workflow**
- **Given** a `.sharkconfig.json` with an `epic_workflow` section
- **When** any shark command initializes the workflow service
- **Then** the epic workflow is parsed and available for validation

**Scenario 2: Epic Status Transition**
- **Given** an epic at status `draft` with a configured workflow
- **When** `shark epic update E16 --status ready_for_research`
- **Then** the transition succeeds with confirmation: "Epic E16 updated: draft -> ready_for_research"

**Scenario 3: Invalid Transition Rejected**
- **Given** an epic at status `draft` with a configured workflow
- **When** `shark epic update E16 --status in_refinement`
- **Then** error: "Cannot transition from 'draft' to 'in_refinement'. Valid: ready_for_research, active, cancelled"

**Scenario 4: Next-Status Advances**
- **Given** an epic at status `draft`
- **When** `shark epic next-status E16`
- **Then** epic transitions to `ready_for_research` (first entry in flow)

**Scenario 5: Backward Compatibility**
- **Given** a `.sharkconfig.json` WITHOUT `epic_workflow`
- **When** `shark epic update E16 --status active`
- **Then** transition succeeds using default statuses

---

## Out of Scope

1. **Orchestrator action responses** - Handled by E16-F02
2. **Display/aggregation threshold behavior** - Handled by E16-F03
3. **Notes/context for epic/feature** - Handled by E16-F04
4. **Backward transition guards with --reason** - Handled by E16-F05
5. **Workflow visualization (workflow list)** - Handled by E16-F06

---

## Dependencies & Integrations

### Dependencies

- **E11 - Configurable Status Workflow System**: Existing task workflow engine that this extends
- **E13 - Workflow-Aware Task Command System**: `next-status` and `show-actions` patterns to replicate
- **workflow.Service** (`internal/services/workflow/`): Core service to extend with level awareness

### Downstream Dependents

- **E16-F02**: Orchestrator Actions (needs workflow engine)
- **E16-F03**: Display & Aggregation (needs `is_planning` metadata)
- **E16-F04**: Notes & Context (needs entity-level workflow awareness)
- **E16-F05**: Backward Transition (needs transition validation)
- **E16-F06**: Workflow Visualization (needs multi-level workflow data)

---

*Last Updated*: 2026-02-08
