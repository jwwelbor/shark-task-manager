---
feature_key: E07-F16
epic_key: E07
title: Workflow Config Integration for Status Display
description: Centralize workflow configuration access and integrate it into all status display commands for dynamic, workflow-aware task management
status: in_development
---

# Workflow Config Integration for Status Display

**Feature Key**: E07-F16
**Epic**: E07 - Enhancements

---

## Goal

### Problem

Currently, shark CLI has multiple pain points related to workflow status handling:

1. **Task creation uses hardcoded "todo" status** instead of respecting the workflow configuration's entry status (e.g., "draft" or "ready_for_development")
2. **No way to filter tasks by workflow status** - users cannot easily find tasks in specific workflow states like "ready_for_code_review"
3. **No workflow progression command** - users must manually update status through the complete workflow chain
4. **Status display doesn't leverage workflow metadata** - phase information, descriptions, and agent assignments aren't shown

### Solution

Create a centralized **WorkflowService** that:
- Loads and caches workflow configuration from `.sharkconfig.json`
- Provides a single source of truth for status ordering, metadata, and transitions
- Integrates into all status display commands (feature get, epic get, task list)
- Enables dynamic, workflow-aware status display with colors and metadata

Add new CLI capabilities:
- `shark task list status <status-name>` - Filter tasks by workflow status
- `shark task next-status <task-key>` - Interactive workflow progression

### Impact

- **Eliminate workflow violations**: Tasks created in correct entry status
- **Improve discoverability**: Users can filter and find tasks by workflow state
- **Reduce friction**: One-command workflow progression instead of manual status updates
- **Better UX**: Status displays show phase, description, and agent information

---

## User Personas

### Persona 1: AI Development Agent

**Profile**:
- **Role/Title**: Automated coding agent (Claude, GPT, etc.)
- **Experience Level**: Expert CLI user, needs machine-readable output
- **Key Characteristics**:
  - Queries task status frequently
  - Needs to find next available work
  - Updates status after completing work

**Goals Related to This Feature**:
1. Filter tasks by status to find work matching their specialty
2. Progress tasks through workflow without knowing all intermediate states
3. Get structured JSON output for parsing

**Pain Points This Feature Addresses**:
- Cannot filter by workflow-specific statuses (only legacy "todo", "in_progress")
- Must know exact workflow chain to update status
- No visibility into what transitions are valid

**Success Looks Like**:
Agent can run `shark task list status ready_for_development --agent=backend --json` and immediately find their next task, then run `shark task next-status T-E07-F16-003` to progress it.

### Persona 2: Human Developer

**Profile**:
- **Role/Title**: Software developer managing project tasks
- **Experience Level**: Moderate CLI proficiency
- **Key Characteristics**:
  - Prefers human-readable output
  - May not remember exact workflow status names
  - Needs clear feedback on what's happening

**Goals Related to This Feature**:
1. Quickly see what tasks are in review, blocked, or ready
2. Move tasks forward without memorizing workflow
3. Understand current task state at a glance

**Pain Points This Feature Addresses**:
- Feature/epic display doesn't show workflow-aware status breakdown
- Creating tasks puts them in wrong initial state
- No guidance on valid next steps for a task

**Success Looks Like**:
Developer sees clear status breakdown when running `shark feature get E07-F16`, with statuses ordered by workflow phase and showing counts. Running `shark task get T-E07-F16-003` shows available transitions.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want tasks I create to use the workflow entry status so that they appear in the correct workflow state.

**Acceptance Criteria**:
- [ ] `shark task create` uses `special_statuses._start_[0]` from workflow config
- [ ] Falls back to "todo" if workflow config is missing
- [ ] Existing tasks with "todo" status continue to work

**Story 2**: As a developer, I want to filter tasks by workflow status so that I can find tasks in specific states.

**Acceptance Criteria**:
- [ ] `shark task list status <status>` filters by exact status name
- [ ] Case-insensitive status matching
- [ ] Invalid status shows helpful error with available statuses
- [ ] Works with `--json` output

**Story 3**: As a developer, I want to progress tasks through the workflow so that I don't need to memorize status transitions.

**Acceptance Criteria**:
- [ ] `shark task next-status <task-key>` shows available transitions
- [ ] Interactive mode when multiple transitions available
- [ ] `--status=<name>` for explicit non-interactive transition
- [ ] `--preview` shows transitions without changing status

**Story 4**: As a developer, I want feature status display to be workflow-aware so that I understand task distribution by phase.

**Acceptance Criteria**:
- [ ] `shark feature get` shows status breakdown ordered by workflow
- [ ] Zero-count statuses are hidden
- [ ] Status names formatted with colors from workflow config

---

### Should-Have Stories

**Story 5**: As a developer, I want to see available transitions in task details so that I know what actions I can take.

**Acceptance Criteria**:
- [ ] `shark task get` shows "Available Transitions" section
- [ ] Each transition shows description and valid agents
- [ ] Suggests `shark task next-status` command

---

### Could-Have Stories

**Story 6**: As a developer, I want to filter tasks by workflow phase so that I can see all tasks in "development" or "review" phases.

**Acceptance Criteria**:
- [ ] `shark task list phase <phase>` filters by phase
- [ ] Phase mapped from workflow config status_metadata

---

## Requirements

### Functional Requirements

**Category: WorkflowService Core**

1. **REQ-F-001**: WorkflowService Implementation
   - **Description**: Create centralized service for workflow config access
   - **User Story**: Foundation for all stories
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `internal/workflow/service.go` created
     - [ ] `internal/workflow/types.go` created
     - [ ] Service loads and caches config on construction
     - [ ] Graceful fallback to defaults when config missing

2. **REQ-F-002**: Status Filtering
   - **Description**: Filter tasks by workflow status via CLI
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task list status <name>` command works
     - [ ] `--status=<name>` flag as alias
     - [ ] Case-insensitive matching
     - [ ] Helpful error on invalid status

3. **REQ-F-003**: Workflow Progression
   - **Description**: Progress tasks through workflow via CLI
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task next-status <key>` command works
     - [ ] Interactive selection when multiple options
     - [ ] `--status` for explicit transition
     - [ ] `--preview` for dry-run

**Category: Display Integration**

4. **REQ-F-004**: Feature Status Display
   - **Description**: Workflow-aware status breakdown in feature get
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Statuses ordered by workflow phase
     - [ ] Zero-count statuses hidden
     - [ ] Colors from workflow config applied

5. **REQ-F-005**: Task Transitions Display
   - **Description**: Show available transitions in task get
   - **User Story**: Story 5
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] "Available Transitions" section in task details
     - [ ] Description and agents shown for each
     - [ ] Command suggestion included

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Config Caching
   - **Description**: Workflow config loaded once per command execution
   - **Target**: Config load < 10ms
   - **Justification**: Avoid repeated file I/O

**Backward Compatibility**

1. **REQ-NF-010**: Legacy Status Support
   - **Description**: Existing "todo" status tasks continue to work
   - **Implementation**: Workflow validation allows legacy statuses with warning
   - **Risk Mitigation**: Gradual migration, not breaking change

---

## Success Metrics

### Primary Metrics

1. **Task Creation Compliance**
   - **What**: % of new tasks created in correct workflow entry status
   - **Target**: 100%
   - **Measurement**: Audit log of task creation statuses

2. **Workflow Command Usage**
   - **What**: Usage of `next-status` vs manual status updates
   - **Target**: 50% adoption within 1 week
   - **Measurement**: Command frequency in usage logs

---

## Dependencies & Integrations

### Dependencies

- **internal/config**: Existing workflow config loading
- **internal/repository**: Task status updates
- **internal/cli**: Command framework

### Internal Integration

- **feature.go**: Status breakdown display
- **epic.go**: Aggregated status display
- **task.go**: Filtering and progression commands

---

## Implementation Plan

See `IMPLEMENTATION_PLAN.md` for detailed task breakdown:

**Phase 1**: Core Infrastructure (T-E07-F16-003, 004, 005)
**Phase 2**: Repository Enhancement (T-E07-F16-006, 007, 008)
**Phase 3**: Feature Get Command (T-E07-F16-009, 010, 011)
**Phase 4**: Task Creation Refactor (T-E07-F16-012)
**Phase 5**: Testing & Documentation (T-E07-F16-013, 014, 015)

---

## CX Design Reference

See `/docs/E07-F16-CX-RECOMMENDATIONS.md` for detailed UX patterns:
- Positional argument syntax for status filtering
- Interactive mode for workflow progression
- Error message patterns with available statuses
- JSON output structure for machine consumption

---

*Last Updated*: 2026-01-09
