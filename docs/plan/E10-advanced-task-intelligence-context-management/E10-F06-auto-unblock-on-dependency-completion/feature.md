---
feature_key: E10-F06-auto-unblock-on-dependency-completion
epic_key: E10
title: Auto-Unblock on Dependency Completion
description: When a task is completed, automatically unblock dependent tasks whose dependencies are all satisfied.
status: draft
execution_order: 6
---

# Auto-Unblock on Dependency Completion

**Feature Key**: E10-F06

---

## Goal

### Problem

When a task completes, dependent tasks remain blocked until someone manually runs `shark task unblock`. The system already auto-blocks dependents when a task is reopened (`ReopenTaskWithAutoBlock`), but the reverse does not exist. This asymmetry breaks automated agent pipelines and requires human bookkeeping.

### Solution

After any status transition to `completed` or `archived`, evaluate all tasks that depend on the completed task. For each that is currently `blocked` with a dependency-driven reason, check if ALL dependencies are now satisfied. If so, transition it to `todo` and record the event in `task_history`.

### Impact

- Eliminates manual `shark task unblock` for dependency chains
- Enables fully autonomous multi-agent task pipelines
- Symmetric with existing auto-block-on-reopen behavior

---

## User Stories

### Must-Have

**Story 1**: As an AI agent, I want dependent tasks to auto-unblock when I complete their prerequisite so that `shark task next` immediately returns new work.

**Acceptance Criteria**:
- [ ] Completing T-001 auto-unblocks T-002 if T-002 depends only on T-001
- [ ] T-002 transitions from `blocked` to `todo`
- [ ] `task_history` records the auto-unblock with provenance

**Story 2**: As a developer, I want tasks with multiple dependencies to only unblock when ALL dependencies are satisfied.

**Acceptance Criteria**:
- [ ] T-003 depends on T-001 and T-002. Completing T-001 does NOT unblock T-003.
- [ ] Completing T-002 (with T-001 already complete) DOES unblock T-003.

**Story 3**: As a developer, I want manually-blocked tasks to remain blocked even when dependencies complete.

**Acceptance Criteria**:
- [ ] Task blocked via `shark task block --reason="waiting on API key"` is NOT auto-unblocked
- [ ] Only tasks with dependency-pattern blocked_reason are candidates

### Should-Have

**Story 4**: As an orchestrator, I want CLI output to show which tasks were auto-unblocked after a completion.

**Acceptance Criteria**:
- [ ] Human output lists unblocked task keys and titles
- [ ] JSON output includes `auto_unblocked` array

---

## Design

### Key Files to Modify

| File | Change |
|------|--------|
| `internal/repository/task_dependency.go` | Add `CompleteTaskWithAutoUnblock()` method |
| `internal/repository/task_repository.go` | Call auto-unblock from `UpdateStatusForced` when new status is completed/archived |
| `internal/cli/commands/task.go` | Surface auto-unblocked tasks in output |

### Algorithm

```
CompleteTaskWithAutoUnblock(ctx, taskID, ...):
  1. Begin transaction
  2. Update task status to completed (existing logic)
  3. Get all dependents of this task (GetTaskDependents)
  4. For each dependent that is blocked:
     a. Check blocked_reason matches dependency pattern
     b. Parse ALL dependencies from depends_on + task_relationships
     c. Check if ALL dependencies are completed/archived
     d. If yes: set status=todo, clear blocked_at/blocked_reason
     e. Create task_history entry
  5. Commit transaction
  6. Return list of auto-unblocked task keys
```

### Dependency-Block Detection

A task is considered "dependency-blocked" (eligible for auto-unblock) if its `blocked_reason`:
- Matches `"Prerequisite task % was reopened"` (set by `ReopenTaskWithAutoBlock`)
- OR matches `"Auto-blocked:%"` (future-proofing)

Tasks blocked with other reasons (manual blocks) are never auto-unblocked.

---

## Out of Scope

- Auto-start: unblocked tasks go to `todo`, not `in_progress`
- Cross-epic dependencies
- Notifications/webhooks
- Configurable enable/disable toggle

---

*Last Updated*: 2026-02-05
