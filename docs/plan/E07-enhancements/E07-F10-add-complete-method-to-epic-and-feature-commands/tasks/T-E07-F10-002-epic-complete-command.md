---
task_key: T-E07-F10-002
epic_key: E07
feature_key: E07-F10
title: Implement shark epic complete command
status: todo
priority: 5
agent_type: backend
depends_on: []
---

# Task: Implement shark epic complete command

## Objective

Add a new `shark epic complete <epic-key>` command that marks all tasks across all features in an epic as completed, with the same safeguards as the feature complete command.

## Acceptance Criteria

- [ ] Command `shark epic complete E07` exists and works
- [ ] Lists all tasks across all features in epic
- [ ] Shows warning with total task count and status breakdown
- [ ] Without `--force`, fails if any incomplete tasks exist
- [ ] With `--force`, completes all tasks
- [ ] Creates task_history records for all updated tasks
- [ ] Updates feature progress for all affected features
- [ ] Updates epic progress to 100%
- [ ] Works with JSON output

## Implementation Pattern

Mirror the feature complete implementation but aggregate across all features.

