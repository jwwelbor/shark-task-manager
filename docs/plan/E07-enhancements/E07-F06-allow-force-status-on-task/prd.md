---
feature_key: E07-F06-allow-force-status-on-task
epic_key: E07
title: allow force status on task
description: 
---

# allow force status on task

**Feature Key**: E07-F06-allow-force-status-on-task

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
The task management system enforces strict status flow transitions (todo → in_progress → ready_for_review → completed), which is generally good for workflow integrity. However, there are legitimate scenarios where users need to force a status change: bulk cleanup operations, fixing incorrect status, migrating existing tasks, or administrative corrections. Currently, there's no way to bypass status validation, forcing users to step through each intermediate status.

### Solution
Add a `--force` flag to task and feature status update commands that bypasses status flow validation. At the task level, allow forcing any status change. At the feature level with `--force`, update the feature status and also force update all child tasks to the specified status. No additional warnings needed since `--force` flag itself signals intentional override.

### Impact
- Enable administrative corrections and bulk operations
- Support task migration and cleanup scenarios
- Maintain workflow integrity while providing escape hatch when needed
- Reduce friction for legitimate override scenarios

---

## User Personas

### Persona 1: Project Administrator / Tech Lead

**Profile**:
- **Role/Title**: Technical Lead or Project Manager with administrative responsibilities
- **Experience Level**: High, manages project cleanup and data corrections
- **Key Characteristics**:
  - Performs bulk operations and cleanup
  - Fixes data inconsistencies from migrations or errors
  - Needs administrative override capabilities

**Goals Related to This Feature**:
1. Quickly mark completed work as done without stepping through statuses
2. Perform bulk status updates for feature completion
3. Fix incorrect statuses from previous errors

**Pain Points This Feature Addresses**:
- Cannot directly set task status without following strict flow
- Bulk marking tasks complete requires many individual commands
- No way to administratively correct status errors

**Success Looks Like**:
Can run `shark task complete T-E01-F01-001 --force` to directly mark task complete. Can run `shark feature complete E01-F01 --force` to mark feature and all its tasks complete in one command.

---

## User Stories

### Must-Have Stories

**Story 1**: As an administrator, I want to force task status changes so that I can correct errors and perform cleanup operations.

**Acceptance Criteria**:
- [ ] Can use `--force` flag on task status update commands
- [ ] `--force` bypasses status flow validation
- [ ] Task status changes to specified value regardless of current status

**Story 2**: As a project manager, I want to force feature status and all child tasks so that I can bulk complete features.

**Acceptance Criteria**:
- [ ] Can use `--force` flag on feature status commands
- [ ] Feature status updates to specified value
- [ ] All child tasks also update to specified status
- [ ] Single command updates feature + all tasks

---

### Should-Have Stories

None identified.

---

### Could-Have Stories

**Story 3**: As a user, I want confirmation output showing what was updated when using force.

**Acceptance Criteria**:
- [ ] Success message lists number of tasks updated
- [ ] Shows old and new status values

---

### Edge Case & Error Stories

None identified - force flag explicitly bypasses normal validation.

---

## Requirements

### Functional Requirements

**Category: Status Management**

1. **REQ-F-001**: Force Flag on Task Commands
   - **Description**: Add `--force` flag to task status update commands (start, complete, approve, etc.)
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `--force` flag added to all task status commands
     - [ ] When present, status flow validation is skipped
     - [ ] Status changes directly to target value
     - [ ] History still recorded with forced=true indicator

2. **REQ-F-002**: Force Flag on Feature Commands
   - **Description**: Add `--force` flag to feature status update commands
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `--force` flag added to feature status commands
     - [ ] Feature status updates to specified value
     - [ ] All child tasks also updated to same status
     - [ ] Batch operation is atomic (all or nothing)

3. **REQ-F-003**: New Feature Status Commands
   - **Description**: Create `shark feature complete` and similar status commands (if they don't exist)
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark feature complete <key> [--force]`
     - [ ] `shark feature start <key> [--force]`
     - [ ] Other status transitions as needed
     - [ ] Consistent with task command patterns

---

### Non-Functional Requirements

**Safety**

1. **REQ-NF-001**: Audit Trail for Forced Changes
   - **Description**: All forced status changes logged in history with forced flag
   - **Measurement**: History records include forced=true field
   - **Justification**: Enable auditing of administrative overrides

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Force Task Status Change**
- **Given** task T-E01-F01-001 is in status "todo"
- **When** admin runs `shark task complete T-E01-F01-001 --force`
- **Then** task status changes directly to "completed"
- **And** bypasses normal in_progress/ready_for_review steps
- **And** history records forced=true

**Scenario 2: Force Feature and Tasks Status**
- **Given** feature E01-F01 has 5 tasks in various statuses
- **When** manager runs `shark feature complete E01-F01 --force`
- **Then** feature status changes to "completed"
- **And** all 5 child tasks also change to "completed"
- **And** success message shows "Updated 1 feature and 5 tasks"

**Scenario 3: Force Without Flag Fails**
- **Given** task T-E01-F01-001 is in status "todo"
- **When** user runs `shark task complete T-E01-F01-001` (no --force)
- **Then** normal validation occurs
- **And** error about invalid status transition is shown

---

## Out of Scope

### Explicitly Excluded

1. **Additional confirmation prompts**
   - **Why**: User specified --force flag explicitly signals intent
   - **Future**: No - force flag is sufficient signal
   - **Workaround**: N/A

2. **Undo capability for forced changes**
   - **Why**: Out of scope; can be addressed separately if needed
   - **Future**: Could add general undo/rollback feature
   - **Workaround**: Manually set status back if needed

3. **Role-based permissions for force**
   - **Why**: No authentication/authorization system currently
   - **Future**: If auth added, could restrict force to admins
   - **Workaround**: Use with care, documented as admin feature

---

### Alternative Approaches Rejected

**Alternative 1: Separate "admin" Commands**
- **Description**: Create `shark admin task-set-status` commands
- **Why Rejected**: Unnecessary complexity; --force flag is clearer and simpler

**Alternative 2: Interactive Confirmation**
- **Description**: Prompt "Are you sure?" when using --force
- **Why Rejected**: Annoying for batch scripts; --force is intentional enough

---

## Success Metrics

### Primary Metrics

1. **Force Usage**
   - **What**: Number of forced status changes
   - **Target**: Used judiciously (<5% of all status changes)
   - **Measurement**: Query history for forced=true

---

## Dependencies & Integrations

### Dependencies

- **Task/Feature Status Commands**: Modify existing commands
- **Repository Layer**: Update status methods to accept force flag
- **History Tracking**: Add forced indicator to history records

### Integration Requirements

None

---

## Compliance & Security Considerations

**Audit Trail**: All forced changes must be logged with forced=true indicator for accountability

---

*Last Updated*: 2025-12-17
