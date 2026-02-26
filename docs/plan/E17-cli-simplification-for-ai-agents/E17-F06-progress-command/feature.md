---
feature_key: E17-F06
epic_key: E17
title: Progress Command
description: Add `shark progress <id>` for viewing entity progress rollups, health indicators, and task breakdowns, replacing the overloaded `shark status <id>` smart dispatcher and resolving the status namespace collision.
execution_order: 6
phase: 2
complexity: M
status: draft
dependencies:
  - E17-F07 (status subcommand must exist first to free the status namespace)
  - E17-F02 (--field support for targeted extraction of progress metrics)
depended_on_by: []
epic_requirements:
  - F06 (Progress Command)
  - NFR-1 (Backward Compatibility)
  - NFR-3 (Service Layer Integration)
  - NFR-4 (Testing)
---

# Progress Command

**Feature Key**: E17-F06
**Phase**: 2 (Should-Have)
**Complexity**: M
**Execution Order**: 6 (after all Phase 1 features; depends on F07 and F02)

---

## Scope

### Problem

The `shark status <id>` smart dispatcher currently shows progress information (weighted %, task breakdowns, health indicators). However, E17-F07 introduces `shark status set/advance/options/history` as a subcommand group for status transitions. This creates a namespace collision: `shark status E18-F05` could mean either "show progress for feature" or be ambiguous with the status subcommand group. Agents and users need a clear, unambiguous command for viewing progress and a clear command for performing status transitions.

### Solution

Add `shark progress <id>` as a dedicated command for viewing entity progress rollups, health indicators, task breakdowns, and action items. The existing `shark status <id>` smart dispatcher becomes a hidden alias for `shark progress <id>`, preserving backward compatibility while resolving the namespace collision.

### What This Feature Does

- Creates `shark progress <id>` command for viewing progress information
- Auto-detects entity type from ID format (epic, feature, task)
- For features: shows weighted progress %, completion %, task breakdown by status, health indicators, action items
- For epics: shows feature rollup, impediments, aggregate progress
- For tasks: shows task progress context (current status, phase, blocking info)
- Supports `--field` flag for targeted extraction (e.g., `--field progress_pct`)
- Supports `--json` for structured output
- Makes existing `shark status <id>` (the progress smart dispatcher) a hidden alias for `shark progress`
- Reuses the existing `status.CalculationService` and `status.GetStatusContext` infrastructure

### What This Feature Does NOT Do

- Does not change status transition behavior (that is E17-F07)
- Does not add new progress metrics -- exposes existing calculations through a clearer command
- Does not remove the existing `shark status <id>` smart dispatcher -- it becomes a hidden alias
- Does not implement batch progress queries

---

## Acceptance Criteria

- [ ] `shark progress E18-F05` shows feature progress: weighted %, completion %, task breakdown by status
- [ ] `shark progress E18` shows epic progress: feature rollup, impediments
- [ ] `shark progress E18-F05-001` shows task progress context (status, phase, blocking info)
- [ ] `--field` flag works: `shark progress E18-F05 --field progress_pct` returns `78.5`
- [ ] `--json` returns structured progress data
- [ ] Existing `shark status <id>` becomes a hidden alias for `shark progress` (backward compat)
- [ ] Health indicators: healthy, warning, critical
- [ ] Action items: tasks requiring attention (grouped by actionable status)
- [ ] All existing tests pass without modification (`make test` green)

---

## Dependencies

### Depends On

- **E17-F07 (Status Subcommand Group)**: The status subcommand group must exist first so that the `status` namespace is claimed for transitions, making the migration of progress to `shark progress` logical and unambiguous.
- **E17-F02 (Field Flag)**: The `--field` flag infrastructure must exist for targeted extraction of progress metrics.

### Depended On By

None.

---

## Implementation Notes

- Implement as `internal/cli/commands/progress.go`
- Reuse the existing `status.CalculationService` for progress calculations (weighted progress, completion progress)
- Reuse `status.GetStatusContext` for health indicators and action items
- Reuse `status.GetActionItems` for tasks requiring attention
- The heavy lifting is already done by the status package -- this is primarily a CLI routing change
- Entity type detection: reuse the existing key-format parsing from smart dispatchers
- For the hidden alias: modify the existing `shark status` smart dispatcher to redirect to `shark progress` when called with an entity ID (not a subcommand like `set`, `advance`, etc.)
- Must use the service layer, not direct repository calls
- Consider creating a `ProgressService` or using the existing `ContextService` to aggregate progress data

---

## Success Metrics

- **Primary**: Clear separation between status transitions (`shark status set/advance`) and progress viewing (`shark progress`)
- **Measured by**: Zero confusion reports about `shark status` ambiguity in agent logs
- **Backward Compatibility**: 100% -- `shark status <id>` still works via hidden alias

---

## UAT Scenarios

- J2-S02: View feature progress during batch workflow
- J4-S01: Check feature progress for decision-making
- J4-S02: Extract specific progress metric with `--field`
- BC-05: `shark status E18-F05` still works (redirects to progress)

---

*Last Updated*: 2026-02-25
