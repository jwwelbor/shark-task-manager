---
feature_key: E13-F06-command-consolidation
epic_key: E13
title: Command Consolidation
description: Consolidate overlapping task commands into unified interfaces with subcommands and flags
---

# Command Consolidation

**Feature Key**: E13-F06-command-consolidation

---

## Epic

- **Epic PRD**: [Workflow-Aware Task Command System](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md) (REQ-F-008, REQ-F-009, REQ-F-010)

---

## Goal

### Problem

The current task command set has redundant commands that confuse users and bloat the help interface:
- **`blocks` and `blocked-by`**: Two separate commands for viewing dependency relationships (outgoing vs incoming)
- **`note` and `notes`**: Inconsistent - one adds notes, one lists them, using different base commands
- **`timeline` and `history`**: Two overlapping commands for viewing task history in different formats

This creates a 25-command interface where users must remember multiple command names for related operations. For example, to understand all dependencies, users must run both `blocks` and `blocked-by`, then mentally merge the results.

### Solution

Consolidate overlapping commands into unified interfaces:

1. **Dependency consolidation**: Merge `blocks` and `blocked-by` into `deps --type=<type>`
   - `shark task deps <task-id>` - Shows all relationships (default)
   - `shark task deps <task-id> --type=blocks` - Shows what task blocks
   - `shark task deps <task-id> --type=blocked-by` - Shows what blocks task

2. **Notes consolidation**: Merge `note` and `notes` into single `notes` command with subcommands
   - `shark task notes <task-id>` - Lists notes (default)
   - `shark task notes add <task-id> "message"` - Adds note
   - `shark task notes list <task-id>` - Explicitly lists notes

3. **History consolidation**: Merge `timeline` into `history --format=timeline`
   - `shark task history <task-id>` - Shows default history view
   - `shark task history <task-id> --format=timeline` - Shows timeline visualization
   - `shark task history <task-id> --format=json` - Shows JSON output

After consolidation, delete deprecated commands: `blocks`, `blocked-by`, `note`, `timeline` (4 commands removed).

### Impact

**For All Users:**
- Reduce from 25 to 21 task commands (16% reduction)
- Cleaner help interface with fewer commands to scan
- Consistent command patterns (subcommands for actions, flags for variations)

**Expected Outcomes:**
- 78% faster command discovery through reduced list
- Zero functionality loss - all operations remain available
- Better command naming consistency (sets precedent for future commands)

---

## User Personas

### Persona 1: AI Orchestrator Agent (Atlas)

**Profile**:
- **Role**: Autonomous task coordination system
- **Experience Level**: Programmatic CLI consumer
- **Key Characteristics**:
  - Needs consistent command patterns for reliable scripting
  - Uses JSON output exclusively
  - Must adapt to command interface changes

**Goals Related to This Feature**:
1. Query dependencies with single command instead of merging results from `blocks` + `blocked-by`
2. Add task notes programmatically with clear, consistent command structure
3. Retrieve task history in different formats without memorizing separate commands

**Pain Points This Feature Addresses**:
- Must execute multiple commands (`blocks`, `blocked-by`) and merge results to get complete dependency view
- Inconsistent note commands (`note` vs `notes`) require special-case handling
- Separate `timeline` command duplicates `history` functionality

**Success Looks Like**:
Atlas queries dependencies with `shark task deps --json`, gets complete relationship graph, and makes blocking decisions without running multiple commands.

### Persona 2: Software Developer (Dev)

**Profile**:
- **Role**: Full-stack developer using shark for task tracking
- **Experience Level**: Moderate CLI proficiency, daily shark user
- **Key Characteristics**:
  - Wants fast command discovery (doesn't want to read full help every time)
  - Prefers logical command grouping
  - Uses tab completion

**Goals Related to This Feature**:
1. Understand task dependencies quickly without remembering multiple command names
2. Add context notes to tasks with clear, intuitive commands
3. View task history in readable format when debugging workflow issues

**Pain Points This Feature Addresses**:
- Forgets whether to use `blocks` or `blocked-by` for specific dependency direction
- Confusion between `note` (add) and `notes` (list) - expects one command
- Doesn't remember `timeline` exists, uses `history` and gets less readable output

**Success Looks Like**:
Dev types `shark task deps <task>` and immediately sees all dependency relationships. When adding context, intuitively tries `shark task notes add` and it works.

---

## User Stories

### Must-Have Stories

**Story 1**: As Atlas (AI orchestrator), I want to query all task dependencies with a single command so that I can make blocking decisions without merging multiple API responses.

**Acceptance Criteria**:
- [ ] `shark task deps <task-id> --json` returns all dependencies (depends-on, blocks, blocked-by) in single response
- [ ] Response includes relationship types and direction for each dependency
- [ ] Command executes in < 500ms for tasks with 20+ dependencies

**Story 2**: As Dev (developer), I want to use one command for both adding and listing notes so that I don't have to remember `note` vs `notes`.

**Acceptance Criteria**:
- [ ] `shark task notes <task-id>` lists all notes (default behavior)
- [ ] `shark task notes add <task-id> "message"` adds new note
- [ ] Both operations work with `--json` flag

**Story 3**: As Dev, I want to view task history in different formats with the same base command so that I don't need to remember separate `timeline` command.

**Acceptance Criteria**:
- [ ] `shark task history <task-id>` shows default history view
- [ ] `shark task history <task-id> --format=timeline` shows timeline visualization
- [ ] Timeline format matches current `timeline` command output exactly

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I try to use deprecated commands, I want clear migration guidance so that I can update my scripts quickly.

**Acceptance Criteria**:
- [ ] Deprecated commands (`blocks`, `blocked-by`, `note`, `timeline`) are deleted
- [ ] Attempting to run them shows: "Command not found. Use 'shark task <replacement>' instead."
- [ ] Help text updated to show only consolidated commands

---

## Requirements

### Functional Requirements

**Category: Dependency Command Consolidation**

1. **REQ-F-008**: Consolidate Dependency Commands
   - **Description**: Merge `blocks`, `blocked-by` into `deps --type=<type>` subcommand (from Epic Requirements)
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task deps <task-id>` shows all relationships (default)
     - [ ] `shark task deps <task-id> --type=depends-on` shows dependencies
     - [ ] `shark task deps <task-id> --type=blocks` shows outgoing blockers
     - [ ] `shark task deps <task-id> --type=blocked-by` shows incoming dependencies
     - [ ] Output format matches existing dependency display
     - [ ] JSON output includes relationship types and direction
     - [ ] Commands `blocks` and `blocked-by` are deleted from codebase

**Category: Notes Command Consolidation**

2. **REQ-F-009**: Consolidate Note Commands
   - **Description**: Merge `note` (add) and `notes` (list) into single `notes` command with subcommands (from Epic Requirements)
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task notes <task-id>` lists all notes (default behavior)
     - [ ] `shark task notes add <task-id> "message"` adds new note
     - [ ] `shark task notes list <task-id>` explicitly lists notes (alias for default)
     - [ ] Supports `--type=<type>` flag for filtering/categorizing notes
     - [ ] JSON output for both list and add operations
     - [ ] Standalone `note` command deleted from codebase

**Category: History Command Consolidation**

3. **REQ-F-010**: Merge Timeline into History
   - **Description**: Remove separate `timeline` command, add as `--format=timeline` flag to `history` (from Epic Requirements)
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task history <task-id>` shows default history view
     - [ ] `shark task history <task-id> --format=timeline` shows timeline visualization
     - [ ] `shark task history <task-id> --format=json` shows JSON output
     - [ ] Timeline format matches existing `timeline` command output exactly
     - [ ] `timeline` command deleted from codebase

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Cutover Strategy
   - **Description**: Straight cutover with no backward compatibility - deprecated commands removed immediately
   - **Rationale**: Epic E13 is comprehensive redesign; better to make clean break than maintain deprecated commands
   - **Implementation**: Delete command files, update help text, update documentation
   - **Migration Path**: Users must update scripts to use new command syntax

**Performance**

2. **REQ-NF-002**: Consolidated Command Performance
   - **Description**: Consolidation must not degrade performance
   - **Measurement**: Compare execution time before/after
   - **Target**:
     - `deps` < 500ms (same as current `blocks` + `blocked-by` combined)
     - `notes` operations < 200ms
     - `history --format=timeline` < 300ms (same as current `timeline`)

**Documentation**

3. **REQ-NF-003**: Updated Documentation
   - **Description**: All docs must reflect consolidated commands
   - **Deliverables**:
     - [ ] CLI_REFERENCE.md updated
     - [ ] CLAUDE.md updated with new command examples
     - [ ] Help text for `shark task --help` updated
     - [ ] Migration guide section in epic documentation

---

## Design

### Architecture

**Command Structure Changes:**

```
BEFORE:
  shark task blocks <task-id>           # Show outgoing blockers
  shark task blocked-by <task-id>       # Show incoming dependencies
  shark task note <task-id> "message"   # Add note
  shark task notes <task-id>            # List notes
  shark task timeline <task-id>         # Show timeline
  shark task history <task-id>          # Show history

AFTER:
  shark task deps <task-id> [--type=blocks|blocked-by|depends-on]
  shark task notes <task-id>                    # List (default)
  shark task notes add <task-id> "message"      # Add
  shark task notes list <task-id>               # List (explicit)
  shark task history <task-id> [--format=default|timeline|json]
```

### Implementation Approach

**Phase 1: Extend Existing Commands**
1. Add `--type` flag to existing `deps` command
   - Modify `internal/cli/commands/task_deps.go`
   - Add type filtering logic for blocks, blocked-by, depends-on
   - Default (no flag) shows all relationship types

2. Convert `notes` to subcommand structure
   - Modify `internal/cli/commands/task_notes.go`
   - Add subcommands: `add`, `list`
   - Make default behavior (no subcommand) list notes
   - Support `--type` flag for note categorization

3. Add `--format` flag to `history`
   - Modify `internal/cli/commands/task_history.go`
   - Add format options: default, timeline, json
   - Migrate timeline formatting logic from `timeline` command

**Phase 2: Delete Deprecated Commands**
1. Delete command files:
   - `internal/cli/commands/task_blocks.go`
   - `internal/cli/commands/task_blocked_by.go`
   - `internal/cli/commands/task_note.go` (if separate from notes)
   - `internal/cli/commands/task_timeline.go`

2. Update command registration in `internal/cli/commands/task.go`

3. Update help text and categorization

**Phase 3: Update Tests**
1. Update tests to use new command syntax
2. Delete tests for removed commands
3. Add tests for new flags and subcommands

### Data Model Changes

**No database changes required** - all consolidations are CLI interface changes only.

### API Changes

**Command Line Interface:**
- Removed: `blocks`, `blocked-by`, `note`, `timeline`
- Enhanced: `deps` (added --type), `notes` (added subcommands), `history` (added --format)
- All JSON outputs remain structurally compatible

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Dependencies - Show All Relationships**
- **Given** task T-E13-F06-001 has dependencies: depends-on T-E13-F06-002, blocks T-E13-F06-003, blocked-by T-E13-F06-004
- **When** user runs `shark task deps T-E13-F06-001`
- **Then** output shows all three relationship types grouped by category
- **And** each relationship shows task key, title, and relationship direction

**Scenario 2: Dependencies - Filter by Type**
- **Given** task has multiple relationship types
- **When** user runs `shark task deps T-E13-F06-001 --type=blocks`
- **Then** output shows only outgoing blocker relationships
- **And** matches format of former `blocks` command exactly

**Scenario 3: Notes - List by Default**
- **Given** task has 3 notes
- **When** user runs `shark task notes T-E13-F06-001`
- **Then** output lists all 3 notes with timestamps and authors
- **And** matches format of former `notes` command exactly

**Scenario 4: Notes - Add with Subcommand**
- **Given** task exists
- **When** user runs `shark task notes add T-E13-F06-001 "Architecture review completed"`
- **Then** note is created successfully
- **And** confirmation message shows note ID and content

**Scenario 5: History - Timeline Format**
- **Given** task has status changes and notes in history
- **When** user runs `shark task history T-E13-F06-001 --format=timeline`
- **Then** output shows timeline visualization with ASCII timeline
- **And** matches format of former `timeline` command exactly

**Scenario 6: Deprecated Commands - Removed**
- **Given** user has script using `shark task blocks`
- **When** script executes
- **Then** command fails with "Unknown command" error
- **And** no migration warning (clean cutover)

---

## Out of Scope

### Explicitly Excluded

1. **Backward Compatibility for Deprecated Commands**
   - **Why**: Epic E13 is comprehensive redesign; maintaining deprecated commands creates technical debt
   - **Future**: No plan to support old commands - this is a breaking change
   - **Workaround**: Users must update scripts to new syntax

2. **Automatic Migration Tool**
   - **Why**: Command syntax changes are straightforward; automated migration adds complexity
   - **Future**: Not planned
   - **Workaround**: Users update scripts manually using migration guide

3. **Consolidating Other Command Groups**
   - **Why**: This feature focuses only on deps/notes/history; other consolidations (if needed) handled separately
   - **Future**: Epic E13-F04 may address other command improvements

---

## Test Plan

### Unit Tests

1. **Test `deps --type` flag**
   - Test default (no --type) shows all relationships
   - Test --type=blocks filters correctly
   - Test --type=blocked-by filters correctly
   - Test --type=depends-on filters correctly
   - Test invalid --type value returns error
   - Test JSON output includes relationship types

2. **Test `notes` subcommands**
   - Test `notes <task>` lists notes (default)
   - Test `notes list <task>` lists notes (explicit)
   - Test `notes add <task> "message"` creates note
   - Test `notes add` without message returns error
   - Test JSON output for both add and list

3. **Test `history --format` flag**
   - Test default format
   - Test --format=timeline matches old timeline output
   - Test --format=json returns valid JSON
   - Test invalid format value returns error

### Integration Tests

1. **Test consolidated commands match deprecated command outputs**
   - Create test task with dependencies
   - Compare `deps --type=blocks` output to former `blocks` command output
   - Compare `deps --type=blocked-by` output to former `blocked-by` command output
   - Compare `notes list` output to former `notes` command output
   - Compare `history --format=timeline` to former `timeline` command output

2. **Test command deletion**
   - Verify `shark task blocks` returns "Unknown command"
   - Verify `shark task blocked-by` returns "Unknown command"
   - Verify `shark task note` returns "Unknown command"
   - Verify `shark task timeline` returns "Unknown command"

### Manual Testing

1. **Verify help text updated**
   - `shark task --help` shows only consolidated commands
   - `shark task deps --help` documents --type flag
   - `shark task notes --help` documents subcommands
   - `shark task history --help` documents --format flag

2. **Verify documentation updated**
   - CLI_REFERENCE.md shows new commands
   - CLAUDE.md examples use new syntax
   - No references to deprecated commands remain

---

## Success Metrics

### Primary Metrics

1. **Command Count Reduction**
   - **What**: Number of task subcommands
   - **Baseline**: 25 commands
   - **Target**: 21 commands (4 removed)
   - **Timeline**: Immediately after deployment
   - **Measurement**: Count commands in `shark task --help`

2. **Functional Equivalence**
   - **What**: All operations available before consolidation remain available
   - **Target**: 100% functional coverage
   - **Timeline**: Pre-release testing
   - **Measurement**: Integration tests compare outputs

### Secondary Metrics

- **Help Interface Clarity**: User feedback on command discoverability (qualitative)
- **Migration Friction**: Number of user-reported issues post-deployment (target: < 5 in first week)

---

## Dependencies & Integrations

### Dependencies

- **Existing `deps` command**: Must be extended, not replaced
- **Existing `notes` command**: Structure changed to subcommand model
- **Existing `history` command**: Must absorb `timeline` functionality

### Integration Requirements

- **None**: This is purely CLI interface consolidation with no external system dependencies

---

## Compliance & Security Considerations

**Not applicable**: This feature is CLI interface consolidation with no security, compliance, or data protection implications.

---

## Commands to Delete

Post-consolidation, delete these command files:

1. `internal/cli/commands/task_blocks.go`
2. `internal/cli/commands/task_blocked_by.go`
3. `internal/cli/commands/task_note.go` (if separate file from notes)
4. `internal/cli/commands/task_timeline.go`

---

*Last Updated*: 2026-01-11
