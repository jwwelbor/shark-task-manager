---
task_key: T-E07-F10-003
epic_key: E07
feature_key: E07-F10
title: Integration tests for complete commands
status: todo
priority: 4
agent_type: backend
depends_on:
  - T-E07-F10-001
  - T-E07-F10-002
---

# Task: Integration tests for complete commands

## Objective

Write comprehensive integration tests for both feature and epic complete commands.

## Test Cases

### Feature Complete
- [ ] Complete feature with all tasks completed → succeeds without warning
- [ ] Complete feature with mixed statuses → shows warning, fails without --force
- [ ] Complete feature with --force → completes all tasks regardless
- [ ] Blocked tasks require --force to complete
- [ ] Task history records created with correct timestamps
- [ ] Feature progress updated to 100%
- [ ] JSON output format verified

### Epic Complete
- [ ] Complete epic with all tasks completed → succeeds
- [ ] Complete epic with incomplete tasks → shows summary, requires --force
- [ ] Complete epic with multiple features → all tasks completed
- [ ] Feature progress values updated correctly
- [ ] Epic progress updated to 100%
- [ ] Cross-feature task_history records created

### Error Cases
- [ ] Invalid feature key → clear error message
- [ ] Invalid epic key → clear error message
- [ ] No tasks in feature/epic → appropriate message
- [ ] Database errors → rolled back transaction

