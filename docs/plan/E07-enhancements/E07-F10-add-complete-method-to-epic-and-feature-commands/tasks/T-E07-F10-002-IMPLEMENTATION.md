# T-E07-F10-002: Implement shark epic complete command

**Task Type**: Implementation
**Epic**: E07 - Enhancements
**Feature**: E07-F10 - Add complete method to epic and feature
**Agent Type**: backend / go-developer
**Complexity**: Medium
**Status**: ready_for_development

## Overview

Implement the `shark epic complete` command to bulk-complete all tasks across all features within an epic. This command should provide safeguards and detailed feedback about the impact.

## Acceptance Criteria

### Command Functionality
- [ ] Command exists: `shark epic complete <epic-key>`
- [ ] If all tasks are already completed or in review → completes immediately without warning
- [ ] If any tasks are incomplete → shows detailed breakdown with task counts and requires `--force`
- [ ] With `--force` → completes all tasks across all features regardless of status
- [ ] Shows comprehensive summary including:
  - Total task count
  - Status breakdown by count
  - Feature-by-feature status (optional, or just overall)
  - Highlighted warnings for blocked tasks

### Status Transitions
- [ ] Tasks across all features transition from current status → completed
- [ ] `completed_at` timestamp is updated for each task
- [ ] Task history records are created for each task
- [ ] Epic progress is calculated correctly (should be 100% with all features at 100%)

### Warning/Force Behavior
- [ ] Without `--force`, if incomplete tasks exist:
  - Show overall breakdown: "X todo, Y in_progress, Z blocked, W ready_for_review (across N features)"
  - Optionally show per-feature breakdown
  - List specific task keys of most problematic tasks (up to 15 tasks, prioritizing blocked)
  - Highlight blocked tasks as requiring explicit attention
  - Exit with code 3 (invalid state)
- [ ] With `--force`:
  - Complete all tasks in all features
  - Show completion summary with total count and breakdown
  - No error exit

### Transactional Safety
- [ ] All tasks complete or all rollback (all-or-nothing)
- [ ] Rollback on any error during the operation
- [ ] Feature and epic progress calculated correctly

### Output Formats
- [ ] Human-readable output with clear formatting
- [ ] `--json` flag support for machine-readable output
- [ ] JSON output includes:
  - epic_key
  - feature_count
  - total_task_count
  - completed_count
  - status_breakdown (object with counts by status)
  - affected_tasks (list of task keys)

## Technical Requirements

### Implementation Details
1. Create command structure in `internal/cli/commands/epic.go`
   - Add `epicCompleteCmd` with proper documentation
   - Register in `init()` function with parent epic command
   - Add flags: `--force`, `--json`, standard global flags

2. Implement `runEpicComplete` handler function
   - Parse epic key from args
   - Get epic from repository
   - Validate epic exists
   - List all tasks in epic via `TaskRepository.ListByEpic()`
   - Analyze status breakdown across all tasks
   - Check if force is needed:
     - Incomplete tasks = any status NOT in {completed, ready_for_review}
     - If incomplete tasks exist AND no --force → show detailed warning and exit 3
     - If --force → proceed
   - Complete all tasks:
     - For each task: call `TaskRepository.UpdateStatusForced(ctx, taskID, TaskStatusCompleted, agent, notes, true)`
     - Wrap in transaction for atomicity
   - Update feature and epic progress (triggers automatically)
   - Output results in requested format (JSON or human)

3. Follow existing patterns from `runTaskComplete` and `runEpicGet`:
   - Database initialization and cleanup
   - Context timeout management
   - Error handling with proper exit codes
   - Agent identifier extraction
   - JSON output handling via `cli.OutputJSON()`
   - Feature list traversal similar to epic get command

### Code References
- Task complete command: `/internal/cli/commands/task.go` line 710
- Epic get command (feature traversal pattern): `/internal/cli/commands/epic.go`
- Feature repository: `/internal/repository/feature_repository.go`
- Task repository: `/internal/repository/task_repository.go`
- Models: `/internal/models/` (Task, Epic, Feature, TaskStatus constants)

## Testing

### Manual Testing
1. Create a test epic with multiple features containing tasks in different states
2. Run `shark epic complete <epic-key>` without --force
   - Should show detailed warning with breakdown if incomplete tasks exist
   - Should show which features are affected
   - Should exit with code 3
3. Run `shark epic complete <epic-key> --force`
   - Should complete all tasks across all features
   - Should show success summary
4. Verify task history records created for all tasks
5. Verify epic progress = 100%
6. Verify all feature progress = 100%
7. Test with --json flag
8. Test with blocked tasks (should require explicit --force)

### Expected Behavior Examples
```
# No incomplete tasks
$ shark epic complete E07
Epic E07 completed: 15/15 tasks completed across 3 features

# Incomplete tasks, no force
$ shark epic complete E07
Warning: Cannot complete epic with incomplete tasks

Total tasks: 15
Status breakdown: 3 todo, 2 in_progress, 1 blocked, 9 ready_for_review

Feature breakdown:
  E07-F08: 5 tasks (1 todo, 1 in_progress)
  E07-F09: 4 tasks (2 todo, 1 blocked)
  E07-F10: 6 tasks (all ready_for_review)

Most problematic tasks:
  - T-E07-F09-001 (blocked) - Depends on external resource
  - T-E07-F09-002 (blocked) - Awaiting review
  - T-E07-F08-001 (todo)
  - T-E07-F08-002 (in_progress)

Use --force to complete all tasks regardless of status

# With force
$ shark epic complete E07 --force
Epic E07 completed: Force-completed 15 tasks (3 todo, 2 in_progress, 1 blocked, 9 ready_for_review)
```

## Definition of Done

- [ ] Epic complete command implemented and functional
- [ ] All acceptance criteria met
- [ ] Command follows existing code patterns and style
- [ ] Error handling proper with correct exit codes
- [ ] Shows detailed breakdown of affected tasks
- [ ] Manual testing completed successfully
- [ ] Code ready for integration tests
- [ ] No breaking changes to existing commands

## Related Documents
- Feature PRD: `/docs/plan/E07-enhancements/E07-F10-complete-commands/prd.md`
- Feature complete task: `T-E07-F10-001-IMPLEMENTATION.md`
