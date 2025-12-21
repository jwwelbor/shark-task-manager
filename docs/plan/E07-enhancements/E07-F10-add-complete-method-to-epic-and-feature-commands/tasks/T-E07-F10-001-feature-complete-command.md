---
task_key: T-E07-F10-001
epic_key: E07
feature_key: E07-F10
title: Implement shark feature complete command
status: todo
priority: 5
agent_type: backend
depends_on: []
---

# Task: Implement shark feature complete command

## Objective

Add a new `shark feature complete <feature-key>` command that marks all tasks in a feature as completed, with safeguards to prevent accidental completion of incomplete tasks.

## Acceptance Criteria

- [ ] Command `shark feature complete E07-F08` exists and works
- [ ] Lists all tasks in feature and their current statuses
- [ ] If any tasks are not completed/reviewed → shows warning summary
- [ ] Warning shows count breakdown: todo, in_progress, blocked, ready_for_review
- [ ] Without `--force`, command fails with exit code 1
- [ ] With `--force`, completes all tasks regardless of status
- [ ] Each task gets a task_history record
- [ ] Feature progress updates to 100%
- [ ] Works with JSON output for programmatic use

## Implementation Notes

See `shark epic complete` (T-E07-F10-002) for parallel implementation pattern.

