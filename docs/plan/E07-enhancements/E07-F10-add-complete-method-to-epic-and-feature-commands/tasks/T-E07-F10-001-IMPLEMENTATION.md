# T-E07-F10-001: Implement shark feature complete command

**Task Type**: Implementation
**Epic**: E07 - Enhancements
**Feature**: E07-F10 - Add complete method to epic and feature
**Agent Type**: backend / go-developer
**Complexity**: Medium
**Status**: ready_for_development

## Overview

Implement the `shark feature complete` command to bulk-complete all tasks within a feature. This command should provide safeguards to prevent accidental completion of incomplete tasks via a `--force` flag requirement.

## Acceptance Criteria

### Command Functionality
- [ ] Command exists: `shark feature complete <feature-key>`
- [ ] If all tasks are already completed or in review → completes immediately without warning
- [ ] If any tasks are not completed → shows warning with count breakdown and requires `--force`
- [ ] With `--force` → completes all tasks regardless of current status
- [ ] Shows summary of affected tasks (count by status)
- [ ] Highlights problematic statuses (blocked tasks)

### Status Transitions
- [ ] Tasks transition from current status → completed
- [ ] `completed_at` timestamp is updated
- [ ] Task history records are created for each task
- [ ] Feature progress is calculated correctly (should be 100%)

### Warning/Force Behavior
- [ ] Without `--force`, if incomplete tasks exist:
  - Show breakdown: "X todo, Y in_progress, Z blocked, W ready_for_review"
  - List specific task keys that will be affected (up to 10 tasks)
  - Suggest addressing incomplete tasks first
  - Exit with code 3 (invalid state)
- [ ] With `--force`:
  - Complete all tasks
  - Show completion summary with count
  - No error exit

### Transactional Safety
- [ ] All tasks complete or all rollback (all-or-nothing)
- [ ] Rollback on any error during the operation

### Output Formats
- [ ] Human-readable output with clear formatting
- [ ] `--json` flag support for machine-readable output
- [ ] JSON output includes:
  - feature_key
  - completed_count
  - total_count
  - status_breakdown (object with counts by status)
  - affected_tasks (list of task keys)

## Technical Requirements

### Implementation Details
1. Create command structure in `internal/cli/commands/feature.go`
   - Add `featureCompleteCmd` with proper documentation
   - Register in `init()` function with parent feature command
   - Add flags: `--force`, `--json`, standard global flags

2. Implement `runFeatureComplete` handler function
   - Parse feature key from args
   - Get feature from repository
   - Validate feature exists
   - List all tasks in feature via `TaskRepository.ListByFeature()`
   - Analyze status breakdown
   - Check if force is needed:
     - Incomplete tasks = any status NOT in {completed, ready_for_review}
     - If incomplete tasks exist AND no --force → show warning and exit 3
     - If --force → proceed
   - Complete all tasks:
     - For each task: call `TaskRepository.UpdateStatusForced(ctx, taskID, TaskStatusCompleted, agent, notes, true)`
     - Wrap in transaction for atomicity
   - Update feature progress (triggers automatically)
   - Output results in requested format (JSON or human)

3. Follow existing patterns from `runTaskComplete`:
   - Database initialization and cleanup
   - Context timeout management
   - Error handling with proper exit codes
   - Agent identifier extraction
   - JSON output handling via `cli.OutputJSON()`

### Code References
- Task complete command: `/internal/cli/commands/task.go` line 710
- Feature repository: `/internal/repository/feature_repository.go`
- Task repository: `/internal/repository/task_repository.go`
- Models: `/internal/models/` (Task, Feature, TaskStatus constants)

## Testing

### Manual Testing
1. Create a test feature with multiple tasks in different states
2. Run `shark feature complete <feature-key>` without --force
   - Should show warning if incomplete tasks exist
   - Should exit with code 3
3. Run `shark feature complete <feature-key> --force`
   - Should complete all tasks
   - Should show success message
4. Verify task history records created
5. Verify feature progress = 100%
6. Test with --json flag

### Expected Behavior Examples
```
# No incomplete tasks
$ shark feature complete E07-F08
Feature E07-F08 completed: 5/5 tasks completed

# Incomplete tasks, no force
$ shark feature complete E07-F08
Warning: Cannot complete feature with incomplete tasks
  Status breakdown: 2 todo, 1 in_progress, 0 blocked, 2 ready_for_review

Affected tasks:
  - T-E07-F08-001 (todo)
  - T-E07-F08-002 (in_progress)

Use --force to complete all tasks regardless of status

# With force
$ shark feature complete E07-F08 --force
Feature E07-F08 completed: Force-completed 5 tasks (2 todo, 1 in_progress, 2 ready_for_review)
```

## Definition of Done

- [ ] Feature complete command implemented and functional
- [ ] All acceptance criteria met
- [ ] Command follows existing code patterns and style
- [ ] Error handling proper with correct exit codes
- [ ] Manual testing completed successfully
- [ ] Code ready for integration tests
- [ ] No breaking changes to existing commands

## Related Documents
- Feature PRD: `/docs/plan/E07-enhancements/E07-F10-complete-commands/prd.md`
- Task complete reference: `T-E07-F10-001-REFERENCE.md`
