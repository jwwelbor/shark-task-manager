# PRD: Complete Method for Epic and Feature Commands

**Epic**: E07 - Enhancements  
**Feature**: E07-F10 - Add complete method to epic and feature  
**Status**: Draft  
**Priority**: Medium  

## Overview

Add `shark epic complete` and `shark feature complete` commands to bulk-complete all tasks within an epic or feature. Includes safeguards to prevent accidental completion of incomplete tasks via a `--force` flag requirement.

## User Stories

### Story 1: Complete Feature Tasks
**As a** project manager  
**I want to** mark all tasks in a feature as completed  
**So that** I can quickly finalize a completed feature without manually updating each task

**Acceptance Criteria**:
- Command: `shark feature complete E07-F08`
- If all tasks are already completed or in review → completes immediately
- If any tasks are not completed → shows warning and requires `--force`
- With `--force` → completes all tasks regardless of current status

### Story 2: Complete Epic Tasks
**As a** project manager  
**I want to** mark all tasks in an epic as completed  
**So that** I can finalize an entire epic at once

**Acceptance Criteria**:
- Command: `shark epic complete E07`
- If all tasks across all features are completed/reviewed → completes
- If any tasks are incomplete → shows warning with count/breakdown and requires `--force`
- With `--force` → completes all tasks across all features

### Story 3: Notification of Incomplete Tasks
**As a** user  
**I want to** see which tasks will be force-completed  
**So that** I can make an informed decision about using `--force`

**Acceptance Criteria**:
- Show task count breakdown by status (todo, in_progress, blocked, ready_for_review)
- Highlight especially problematic statuses (blocked tasks)
- List specific task keys that will be affected
- Suggest addressing them before force-completing

## Requirements

### Functional Requirements

1. **Feature Complete Command**
   - Input: Feature key (e.g., `E07-F08`)
   - Behavior:
     - Query all tasks in feature
     - Check if any are not in {completed, ready_for_review}
     - If incomplete tasks exist: Show summary, require `--force`
     - If `--force`: Transition all to completed
     - If all completed/reviewed: Proceed without warning

2. **Epic Complete Command**
   - Input: Epic key (e.g., `E07`)
   - Behavior:
     - Query all tasks across all features in epic
     - Check completion status
     - Same warning/force behavior as feature complete

3. **Status Transition**
   - Tasks should transition: `current_status` → `completed`
   - Update `completed_at` timestamp
   - Create task_history record
   - Update feature/epic progress calculations

4. **--force Flag**
   - Optional boolean flag
   - Bypasses incomplete task warning
   - Completes tasks regardless of current status
   - Blocked tasks require explicit `--force` to complete

### Non-Functional Requirements

1. **Transactional Safety**
   - All-or-nothing: Complete all or complete none
   - Rollback on any error

2. **Performance**
   - Should complete 100+ tasks in < 1 second

3. **Auditability**
   - Each task gets history record
   - Track which user/agent issued complete command
   - Include timestamp

## Success Criteria

- [ ] `shark feature complete E07-F08` without --force shows warning if incomplete
- [ ] `shark feature complete E07-F08 --force` completes all tasks
- [ ] `shark epic complete E07` shows task breakdown by status
- [ ] All task_history records created with proper audit trail
- [ ] Feature/epic progress updated to 100%
- [ ] All tests pass (unit + integration)

## Open Questions

1. Should `--mark-reviewed` flag be added to transition to ready_for_review first?
2. Should epic/feature status be updated to "completed"?
3. Should blocked tasks be allowed to complete, or require separate unblock?

