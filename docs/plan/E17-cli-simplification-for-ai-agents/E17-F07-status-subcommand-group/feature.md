---
feature_key: E17-F07
epic_key: E17
title: Status Subcommand Group
description: Create a `shark status` subcommand group that consolidates all status-related operations (set, advance, options, history) under one discoverable namespace, replacing scattered task-specific commands with a unified, entity-type-agnostic API.
execution_order: 4
phase: 1
complexity: M
status: draft
dependencies: []
depended_on_by:
  - E17-F06 (progress needs status namespace separation)
epic_requirements:
  - F01 (Status Subcommand Group)
  - NFR-1 (Backward Compatibility)
  - NFR-3 (Service Layer Integration)
  - NFR-4 (Testing)
prd_mapping_note: This feature corresponds to F01 in the epic PRD. It was assigned E17-F07 in shark because F02-F06 already existed when it was created.
---

# Status Subcommand Group

**Feature Key**: E17-F07
**Phase**: 1 (Must-Have)
**Complexity**: M
**Execution Order**: 4 (after infrastructure features F04, F05, F03)

---

## Scope

### Problem

AI agents face status transition confusion as the top usability pain point. Status operations are scattered across multiple commands: `shark task next-status`, `shark task update --status`, `shark task set-status`, `shark task start/complete/approve`, and `shark feature/epic` variants. Agents cannot discover all available status operations from a single help page. Furthermore, `shark status <id>` currently shows progress information (a smart dispatcher), creating a namespace collision with the intended status transition commands.

### Solution

Create a `shark status` subcommand group that consolidates all status-related operations under one namespace:
- `shark status set <id> <status>` -- set an entity to a specific status
- `shark status advance <id>` -- move to the next workflow status
- `shark status options <id>` -- show current status and valid transitions
- `shark status history <id>` -- show status change log

### What This Feature Does

- Creates `status` as a Cobra parent command with `set`, `advance`, `options`, and `history` subcommands
- Auto-detects entity type (epic, feature, task) from the ID format
- `status set` validates transitions via the workflow service, supports `--force`, `--notes`, `--agent`
- `status set` is idempotent: returns success (exit 0) with `"changed": false` if entity is already at target status
- `status advance` moves an entity to its next status in the workflow, with `--to` for disambiguation
- `status options` returns current status, valid transitions, phase, and agent type
- `status history` returns the status change log (replaces existing `shark history` which becomes a hidden alias)
- All subcommands support `--json` output
- Works with both short (`E18-F05-001`) and traditional (`T-E18-F05-001`) key formats

### What This Feature Does NOT Do

- Does not remove existing status transition commands (`task start`, `task complete`, etc.) -- they continue to work
- Does not change the behavior of any existing command
- Does not implement batch mode for status changes (that is F07 in the PRD, Phase 2)
- Does not resolve the `shark status <id>` smart dispatcher collision (that is handled by E17-F06 Progress Command)

---

## Acceptance Criteria

### `shark status set`
- [ ] `shark status set E18-F05-001 in_development` changes task status
- [ ] `shark status set E18-F05 active` changes feature status
- [ ] `shark status set E18 active` changes epic status
- [ ] Auto-detects entity type from ID format
- [ ] Validates transition via workflow service
- [ ] Returns updated entity as JSON when `--json` active
- [ ] Accepts `--force` to skip workflow validation
- [ ] Accepts `--notes` for transition notes
- [ ] Accepts `--agent` to set agent on task transitions
- [ ] Idempotent: returns success (exit 0) if entity already at target status
- [ ] Response includes `"changed": false` to indicate no-op transition
- [ ] No history record created for no-op transitions
- [ ] Works with both short and traditional key formats

### `shark status advance`
- [ ] `shark status advance E18-F05-001` moves task to next status in workflow
- [ ] When multiple next statuses are valid, uses the primary/default transition
- [ ] `--to <status>` to specify which next status (when ambiguous)
- [ ] `--force` to skip workflow validation
- [ ] `--notes` for transition notes
- [ ] `--agent` to set agent on transition
- [ ] Works for tasks, features, and epics (auto-detected from ID)
- [ ] Returns updated entity with new status

### `shark status options`
- [ ] `shark status options E18-F05-001` shows current status and valid transitions
- [ ] Output includes: `current_status`, `valid_transitions[]`, `phase`, `agent_type`
- [ ] Replaces the `next-status --preview` pattern
- [ ] Works for all entity types

### `shark status history`
- [ ] `shark status history E18-F05-001` shows status change log
- [ ] Existing `shark history` becomes a hidden alias for `shark status history`
- [ ] Each entry shows: from_status, to_status, timestamp, agent, notes

### General
- [ ] Human-readable errors still go to stderr when NOT in JSON mode (unchanged behavior)
- [ ] All existing tests pass without modification (`make test` green)

---

## Dependencies

### Depends On

None. This is standalone -- all Phase 1 features are independent.

### Depended On By

- **E17-F06 (Progress Command)**: The progress command needs the status namespace to be claimed by this subcommand group so that `shark status <id>` (progress) can be migrated to `shark progress <id>`.

---

## Implementation Notes

- Implement as `internal/cli/commands/status_group.go`
- `status` is a Cobra parent command with `set`, `advance`, `options`, `history` subcommands
- Must use the service layer (TaskService, FeatureService, EpicService) -- not direct repo calls
- `set` and `advance` share underlying status transition logic already in `workflow.Service`
- Entity type detection: reuse the existing key-format parsing used by smart dispatchers
- Idempotent `set`: check current status before transition, skip if already at target
- The existing `shark status <id>` smart dispatcher will need to coexist temporarily. Resolution is handled by F06 (Phase 2) which introduces `shark progress`.
- Existing lifecycle commands (`task start`, `task complete`, etc.) remain unchanged as separate commands

---

## Success Metrics

- **Primary**: Agents use `shark status set/advance` for all status transitions instead of entity-specific commands
- **Measured by**: Count of `shark status set` and `shark status advance` invocations vs legacy commands
- **Discovery**: Single `shark status --help` shows all status operations
- **Backward Compatibility**: 100% -- all existing commands unchanged

---

## UAT Scenarios

- J1-S03: Set task status with `shark status set`
- J1-S05: Advance task to next status with `shark status advance`
- J1-S06: Check valid transitions with `shark status options`
- J2-S01: Status transitions in batch workflow context
- BC-01 through BC-07: Existing commands still functional

---

*Last Updated*: 2026-02-25
