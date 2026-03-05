---
feature_key: E18-F06-unified-cli-integration-and-key-auto-detection
epic_key: E18
title: Unified CLI Integration and Key Auto-Detection
description: Extend all unified CLI commands (get, status, search, delete, update, context, notes, view, resume) to auto-detect B### and C### key formats and dispatch to bug/change-card handlers.
---

# Unified CLI Integration and Key Auto-Detection

**Feature Key**: E18-F06
**Execution Order**: 6 (depends on F04 and F05)

---

## Epic

- **Epic PRD**: [Bug and Change-Card Management System](../epic.md)
- **Epic Requirements**: [Requirements](../requirements.md)
- **Epic Scope**: [Scope Boundaries](../scope.md)

---

## Goal

### Problem

After F04 (Bug CLI Commands) and F05 (Change-Card CLI Commands) are implemented, users can manage bugs via `shark bug <subcommand>` and change-cards via `shark change <subcommand>`. However, the unified commands established by E17 (`shark get`, `shark status`, `shark search`, `shark delete`, `shark update`, `shark context`, `shark notes`, `shark view`, `shark history`) do not recognize `B###` and `C###` key formats. A user who types `shark get B001` receives an "unknown entity type" error instead of the bug details. This breaks the principle of consistent, entity-agnostic interaction that E17 established. AI agents that rely on `shark get <key> --json` for all entity types cannot use bugs and change-cards through the unified interface without special-casing their key handling logic.

### Solution

Extend the key detection system (`internal/keys/service.go`) and all entity-type dispatch points across the CLI and service layers to recognize `B###` (bug) and `C###` (change-card) key formats. Each dispatch point (currently ~15 switch statements across ~12 files) adds a `case "bug"` and `case "change"` that delegates to the bug and change-card services and repositories created in F02/F03. The search repository is extended to include bugs and change-cards in full-text search results. The `--type` filter on `shark search` gains `bug` and `change` as valid values.

### Impact

- Every unified CLI command works identically for bugs and change-cards as it does for epics, features, and tasks
- AI agents use `shark get B001 --json`, `shark status advance C001`, and `shark search "login" --type=bug` without learning new patterns
- Bug and change-card entities become truly first-class citizens in the CLI

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want `shark get B001` to return bug details so that I can inspect any entity using the same command pattern regardless of entity type.

**Acceptance Criteria**:
- [ ] `shark get B001` returns bug details with title, status, severity, and linked entity
- [ ] `shark get B001 --json` returns machine-readable JSON output with the same fields as `shark bug get B001 --json`
- [ ] `shark get B001 --field status` returns just the bug status value
- [ ] `shark get C001` returns change-card details with title, status, and linked entity
- [ ] `shark get C001 --json` returns machine-readable JSON output
- [ ] `shark get C001 --field status` returns just the change-card status value

**Story 2**: As a developer, I want `shark status advance B001` to advance a bug through its workflow so that I can manage bug lifecycle using the same status commands I use for tasks.

**Acceptance Criteria**:
- [ ] `shark status advance B001` advances bug B001 to the next workflow status
- [ ] `shark status advance C001` advances change-card C001 to the next workflow status
- [ ] `shark status set B001 wont_fix` sets bug status directly
- [ ] `shark status set C001 declined` sets change-card status directly
- [ ] `shark status options B001` shows valid next statuses for the bug's current state
- [ ] `shark status options C001` shows valid next statuses for the change-card's current state
- [ ] `shark status history B001` shows the bug's status change history
- [ ] `shark status history C001` shows the change-card's status change history

**Story 3**: As a developer, I want `shark search "login" --type=bug` to find bugs matching my query so that I can discover bugs without knowing their keys.

**Acceptance Criteria**:
- [ ] `shark search "login"` returns results from all entity types including bugs and change-cards
- [ ] `shark search "login" --type=bug` returns only bug results
- [ ] `shark search "dark mode" --type=change` returns only change-card results
- [ ] Search results for bugs include the key (B###), title, status, and severity
- [ ] Search results for change-cards include the key (C###), title, and status
- [ ] The `--type` flag validation rejects invalid values with a clear error listing valid options including "bug" and "change"

**Story 4**: As a developer, I want `shark delete B001` and `shark update B001 --title="new title"` to work so that all CRUD operations are available through unified commands.

**Acceptance Criteria**:
- [ ] `shark delete B001` deletes bug B001 (with confirmation prompt unless `--force`)
- [ ] `shark delete C001` deletes change-card C001 (with confirmation prompt unless `--force`)
- [ ] `shark update B001 --title="Updated bug title"` updates the bug title
- [ ] `shark update C001 --title="Updated card title"` updates the change-card title
- [ ] Error messages for not-found entities use the correct entity type name ("bug" or "change card")

**Story 5**: As a developer, I want `shark context get B001` and `shark notes B001` to work so that context and notes for bugs and change-cards are accessible through unified commands.

**Acceptance Criteria**:
- [ ] `shark context get B001` returns the bug's context data
- [ ] `shark context set B001 --field environment --value "Safari 17.2"` sets a context field on the bug
- [ ] `shark context clear B001` clears all context data for the bug
- [ ] `shark context get C001`, `shark context set C001`, and `shark context clear C001` work for change-cards
- [ ] `shark notes B001` lists all notes on the bug
- [ ] `shark notes C001` lists all notes on the change-card

**Story 6**: As a developer, I want `shark view B001` to display the bug's markdown file so that I can read the full bug report through the unified view command.

**Acceptance Criteria**:
- [ ] `shark view B001` renders the markdown file at `docs/bugs/B001.md`
- [ ] `shark view C001` renders the markdown file at `docs/changes/C001.md`
- [ ] If the markdown file does not exist, the command returns a clear error message

**Story 7**: As a developer, I want `shark history B001` to show the bug's status change history so that I can audit bug lifecycle through the unified history command.

**Acceptance Criteria**:
- [ ] `shark history B001` shows the bug's status change history with timestamps and actor
- [ ] `shark history C001` shows the change-card's status change history
- [ ] `shark history B001 --json` returns machine-readable JSON output

---

### Should-Have Stories

**Story 8**: As a developer, I want `shark resume B001` to display the full context for a bug so that I can pick up where I left off on a bug fix.

**Acceptance Criteria**:
- [ ] `shark resume B001` displays the bug's current status, context data, recent notes, and linked entity
- [ ] `shark resume C001` displays the change-card's current status, context data, and recent notes
- [ ] Output includes severity for bugs

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when I pass an ambiguous key like `B` (no digits) to `shark get`, I want a clear error message telling me the key format is invalid.

**Acceptance Criteria**:
- [ ] `shark get B` returns an error message indicating invalid key format
- [ ] `shark get C` returns an error message indicating invalid key format
- [ ] The error message lists valid key format examples including B### and C###

**Error Story 2**: As a developer, when I pass a valid-format bug key like `B999` that does not exist in the database, I want a clear "not found" error that identifies the entity type.

**Acceptance Criteria**:
- [ ] `shark get B999` returns "bug not found: B999" (not "task not found" or "unknown entity")
- [ ] `shark get C999` returns "change card not found: C999"
- [ ] Exit code is 1 (not found), matching existing patterns

**Error Story 3**: As a developer, when I pass a `--type` filter value that does not exist to `shark search`, I want the error message to list all valid types including "bug" and "change".

**Acceptance Criteria**:
- [ ] `shark search "query" --type=invalid` returns an error listing valid types
- [ ] Valid types list includes: epic, feature, task, bug, change

---

## Requirements

### Functional Requirements

**Category: Key Detection**

1. **F06-REQ-001**: B### Key Pattern Recognition
   - **Description**: The key detection system must recognize keys matching the pattern `B` followed by one or more digits (e.g., B001, B1, B42, B1000) as bug entity keys. Keys are case-insensitive (`b001` is equivalent to `B001`).
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `keys.KeyService.DetectEntityType("B001")` returns `EntityTypeBug`
     - [ ] `keys.KeyService.DetectEntityType("b042")` returns `EntityTypeBug` (case-insensitive)
     - [ ] `keys.KeyService.DetectEntityType("B1")` returns `EntityTypeBug` (variable digit count)
     - [ ] `keys.KeyService.Parse("B001")` returns a `ParsedKey` with `EntityType == EntityTypeBug` and the numeric portion extracted

2. **F06-REQ-002**: C### Key Pattern Recognition
   - **Description**: The key detection system must recognize keys matching the pattern `C` followed by one or more digits (e.g., C001, C1, C15) as change-card entity keys. Keys are case-insensitive.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `keys.KeyService.DetectEntityType("C001")` returns `EntityTypeChange`
     - [ ] `keys.KeyService.DetectEntityType("c015")` returns `EntityTypeChange` (case-insensitive)
     - [ ] `keys.KeyService.Parse("C001")` returns a `ParsedKey` with `EntityType == EntityTypeChange`

3. **F06-REQ-003**: EntityType Constants
   - **Description**: The `keys` package must export `EntityTypeBug` and `EntityTypeChange` constants alongside the existing `EntityTypeEpic`, `EntityTypeFeature`, `EntityTypeTask`, and `EntityTypeUnknown` constants.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `keys.EntityTypeBug` constant has value `"bug"`
     - [ ] `keys.EntityTypeChange` constant has value `"change"`
     - [ ] Both constants are of type `keys.EntityType`

4. **F06-REQ-004**: CLI Helper Key Detection Parity
   - **Description**: The `DetectEntityType()` function in `internal/cli/commands/helpers.go` must return `"bug"` for B### keys and `"change"` for C### keys, consistent with the `keys.KeyService` behavior.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `commands.DetectEntityType("B001")` returns `"bug"`
     - [ ] `commands.DetectEntityType("C001")` returns `"change"`
     - [ ] Both functions agree on entity type for all key formats

**Category: Unified Command Dispatch**

5. **F06-REQ-005**: Get Command Dispatch
   - **Description**: The `shark get` command must dispatch to the bug service for B### keys and the change-card service for C### keys.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `shark get B001` calls `BugService.GetBug()` and renders the bug details
     - [ ] `shark get C001` calls `ChangeCardService.GetChangeCard()` and renders the change-card details
     - [ ] JSON output structure matches the entity-specific command output (`shark bug get B001 --json`)

6. **F06-REQ-006**: Status Commands Dispatch
   - **Description**: All `shark status` subcommands (advance, set, options, history) must dispatch to the bug or change-card workflow services for B### and C### keys.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-004, REQ-F-009, REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `shark status advance B001` uses the bug workflow level to determine and apply the next status
     - [ ] `shark status advance C001` uses the change-card workflow level
     - [ ] `shark status set B001 wont_fix` validates the transition against the bug workflow and applies it
     - [ ] `shark status options B001` queries the bug workflow for valid next statuses
     - [ ] `shark status history B001` queries status history for the bug entity

7. **F06-REQ-007**: Delete Command Dispatch
   - **Description**: The `shark delete` command must dispatch to the bug or change-card service for B### and C### keys.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `shark delete B001` prompts for confirmation and deletes the bug
     - [ ] `shark delete C001` prompts for confirmation and deletes the change-card
     - [ ] `shark delete B001 --force` skips confirmation

8. **F06-REQ-008**: Update Command Dispatch
   - **Description**: The `shark update` command must dispatch to the bug or change-card service for B### and C### keys. The `--status` flag is not accepted (use `shark status set` instead), matching existing behavior for epics/features/tasks.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `shark update B001 --title="new title"` updates the bug title
     - [ ] `shark update C001 --title="new title"` updates the change-card title

9. **F06-REQ-009**: Context Command Dispatch
   - **Description**: The `shark context` commands (get, set, clear) must dispatch to the context service for B### and C### keys.
   - **User Story**: Story 5
   - **Priority**: Must-Have
   - **Traces to**: REQ-F-012
   - **Acceptance Criteria**:
     - [ ] `shark context get B001` returns bug context data
     - [ ] `shark context set B001 --field environment --value "Safari 17.2"` sets context on the bug
     - [ ] `shark context clear C001` clears change-card context

10. **F06-REQ-010**: Notes Command Dispatch
    - **Description**: The `shark notes` command must dispatch to the note service for B### and C### keys.
    - **User Story**: Story 5
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `shark notes B001` lists bug notes
      - [ ] `shark notes C001` lists change-card notes

11. **F06-REQ-011**: View Command Dispatch
    - **Description**: The `shark view` command must dispatch to the bug or change-card file path for B### and C### keys.
    - **User Story**: Story 6
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `shark view B001` renders the bug's markdown file
      - [ ] `shark view C001` renders the change-card's markdown file

12. **F06-REQ-012**: History Command Dispatch
    - **Description**: The `shark history` command must dispatch to the status history service for B### and C### keys.
    - **User Story**: Story 7
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `shark history B001` shows bug status change history
      - [ ] `shark history C001` shows change-card status change history

**Category: Search Extension**

13. **F06-REQ-013**: Search Repository Extension
    - **Description**: The search repository must include bugs and change-cards in full-text search results. Bug records include title, description, and notes. Change-card records include title, description, and notes.
    - **User Story**: Story 3
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `shark search "login"` returns matching bugs alongside tasks, features, and epics
      - [ ] `shark search "login"` returns matching change-cards
      - [ ] Bug search results include: key, title, status, severity
      - [ ] Change-card search results include: key, title, status

14. **F06-REQ-014**: Search Type Filter Extension
    - **Description**: The `--type` flag on `shark search` must accept `bug` and `change` as valid values to filter results to a single entity type.
    - **User Story**: Story 3
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `shark search "query" --type=bug` returns only bug results
      - [ ] `shark search "query" --type=change` returns only change-card results
      - [ ] Invalid `--type` values produce an error listing all valid types (epic, feature, task, bug, change)

**Category: Service Layer Extension**

15. **F06-REQ-015**: Context Service Extension
    - **Description**: The `ContextService` must handle `EntityTypeBug` and `EntityTypeChange` in its `getContextJSON()` and `setContextJSON()` dispatch methods.
    - **User Story**: Story 5
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `ContextService.GetContext(ctx, models.EntityTypeBug, "B001")` returns bug context data
      - [ ] `ContextService.SetContextField(ctx, models.EntityTypeChange, "C001", "effort", "small")` sets context on a change-card
      - [ ] Unsupported entity types still return a clear error

16. **F06-REQ-016**: Resume Service Extension
    - **Description**: The `ResumeService` must handle bug and change-card entities, returning current status, context data, recent notes, and linked entity information.
    - **User Story**: Story 8
    - **Priority**: Should-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `ResumeService.Resume(ctx, "B001")` returns bug context with severity and linked entity
      - [ ] `ResumeService.Resume(ctx, "C001")` returns change-card context

17. **F06-REQ-017**: Note Service Validation Extension
    - **Description**: The note service (and the `ValidEntityTypes` map in `models`) must accept `"bug"` and `"change"` as valid entity types for note operations.
    - **User Story**: Story 5
    - **Priority**: Must-Have
    - **Traces to**: REQ-F-012
    - **Acceptance Criteria**:
      - [ ] `models.ValidEntityTypes` map includes `EntityTypeBug` and `EntityTypeChange`
      - [ ] Note search with `--entity-type=bug` filters to bug notes only
      - [ ] Note search with `--entity-type=change` filters to change-card notes only

**Category: Display and Rendering**

18. **F06-REQ-018**: Display Rendering for Bugs and Change-Cards
    - **Description**: The `render_common.go` display infrastructure must support rendering bug and change-card details in the terminal, including entity type headers and field layouts.
    - **User Story**: Story 1
    - **Priority**: Must-Have
    - **Traces to**: REQ-NF-005
    - **Acceptance Criteria**:
      - [ ] `shark get B001` renders with a "Bug: B001" section header (matching "Epic: E07", "Feature: E07-F01" pattern)
      - [ ] `shark get C001` renders with a "Change Card: C001" section header
      - [ ] Bug display includes: title, status, severity, linked entity, created/updated timestamps
      - [ ] Change-card display includes: title, status, linked entity, created/updated timestamps

**Category: Dispatch Verification**

19. **F06-REQ-019**: Dispatch Point Inventory
    - **Description**: Before any implementation begins, a grep-based inventory of all entity-type dispatch points must be created and stored as a verification checklist. After implementation, every dispatch point in the inventory must be verified to handle `"bug"` and `"change"` cases.
    - **User Story**: Risk mitigation (Research Risk 1)
    - **Priority**: Must-Have
    - **Traces to**: REQ-NF-005
    - **Acceptance Criteria**:
      - [ ] A checklist document lists every file and line number containing an entity-type switch/dispatch
      - [ ] Every dispatch point in the checklist has a `"bug"` case or is explicitly documented as not applicable (with reason)
      - [ ] Every dispatch point in the checklist has a `"change"` case or is explicitly documented as not applicable (with reason)

---

### Non-Functional Requirements

**Consistency**

1. **F06-REQ-NF-001**: CLI Output Consistency
   - **Description**: The JSON output structure for `shark get B001 --json` must be identical to `shark bug get B001 --json`. The same applies for change-cards. Users must not see different JSON schemas depending on whether they used the unified command or the entity-specific command.
   - **Measurement**: Diff comparison of JSON output from both command paths for the same entity
   - **Target**: Zero structural differences in JSON output between unified and entity-specific commands
   - **Traces to**: REQ-NF-005

2. **F06-REQ-NF-002**: Error Message Consistency
   - **Description**: Error messages for bugs and change-cards must follow the same patterns used for epics, features, and tasks. Entity type names in errors must be "bug" and "change card" (not "change-card" or "changecard").
   - **Measurement**: Manual review of all error paths
   - **Target**: Zero deviations from established error message patterns
   - **Traces to**: REQ-NF-005

**Performance**

3. **F06-REQ-NF-003**: Key Detection Performance
   - **Description**: Adding B### and C### patterns to `DetectEntityType()` must not measurably increase key detection time. The function is called on every unified command invocation.
   - **Measurement**: Benchmark test comparing key detection before and after changes
   - **Target**: Less than 5% increase in median key detection time
   - **Traces to**: REQ-NF-001

**Completeness**

4. **F06-REQ-NF-004**: Dispatch Exhaustiveness
   - **Description**: Every entity-type dispatch point in the codebase must handle bugs and change-cards. No dispatch point may silently fall through to a default case that produces incorrect behavior or confusing errors.
   - **Measurement**: Dispatch inventory checklist (F06-REQ-019) with 100% coverage
   - **Target**: Zero missed dispatch points
   - **Traces to**: REQ-NF-005

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Unified Get for Bug**
- **Given** a bug B001 exists in the database with title "Login fails on Safari" and status "reported"
- **When** the user runs `shark get B001`
- **Then** the command displays the bug details including title, status, severity, and linked entity
- **And** the output matches the format produced by `shark get E07` (section header, field layout)

**Scenario 2: Unified Get for Change-Card**
- **Given** a change-card C001 exists with title "Add dark mode toggle"
- **When** the user runs `shark get C001 --json`
- **Then** the command returns valid JSON with the same fields as `shark change get C001 --json`

**Scenario 3: Unified Status Advance for Bug**
- **Given** a bug B001 exists in status "reported"
- **When** the user runs `shark status advance B001`
- **Then** the bug status advances to "triaged" (per the bug workflow)
- **And** the status change is recorded in the status history

**Scenario 4: Unified Status Set for Change-Card**
- **Given** a change-card C001 exists in status "proposed"
- **When** the user runs `shark status set C001 declined`
- **Then** the change-card status changes to "declined"
- **And** the status change is recorded in the status history

**Scenario 5: Search with Type Filter**
- **Given** bugs B001 ("Login fails") and B002 ("Signup error") exist, and a task with title "Login API" exists
- **When** the user runs `shark search "login" --type=bug`
- **Then** only B001 appears in the results (B002 does not match "login", task is excluded by type filter)

**Scenario 6: Search Without Type Filter Includes Bugs**
- **Given** bug B001 ("Login fails") and task E07-F01-001 ("Login API") exist
- **When** the user runs `shark search "login"`
- **Then** both B001 and E07-F01-001 appear in the results

**Scenario 7: Delete Bug via Unified Command**
- **Given** a bug B001 exists
- **When** the user runs `shark delete B001 --force`
- **Then** the bug is deleted from the database and the markdown file is removed
- **And** `shark get B001` returns "bug not found: B001"

**Scenario 8: Context Operations for Bug**
- **Given** a bug B001 exists with no context data
- **When** the user runs `shark context set B001 --field environment --value "Safari 17.2"`
- **Then** `shark context get B001` returns `{"environment": "Safari 17.2"}`

**Scenario 9: Not-Found Error Identifies Entity Type**
- **Given** no bug B999 exists
- **When** the user runs `shark get B999`
- **Then** the error message contains "bug" and "B999" (not "task" or "unknown entity")
- **And** the exit code is 1

**Scenario 10: Invalid Key Format Error**
- **Given** the user provides a key that starts with B but has no digits
- **When** the user runs `shark get B`
- **Then** the command returns an error about invalid key format
- **And** the error message includes examples of valid key formats

---

## Out of Scope

### Explicitly Excluded

1. **Dashboard and Analytics Integration**
   - **Why**: Dashboard sections for bugs and change-cards are the responsibility of F07 (Dashboard and Analytics Integration). F06 handles only the unified command dispatch, not reporting or aggregate views.
   - **Future**: Handled by F07 in the same epic.

2. **Entity Type Registry Pattern**
   - **Why**: The research report recommends an entity type registry to reduce scattered switch statements, but this is a refactoring concern that extends beyond E18's scope. Implementing it would affect all existing entity types and requires its own design.
   - **Future**: Candidate for a follow-on epic focused on reducing dispatch maintenance burden.
   - **Workaround**: Each dispatch point is updated manually with a verification checklist to ensure completeness.

3. **List Command Extension for Bugs and Change-Cards**
   - **Why**: `shark list` currently lists epics, features within an epic, and tasks within a feature. Bugs and change-cards are standalone entities that do not fit the hierarchical list pattern. `shark bug list` and `shark change list` (from F04/F05) are the appropriate commands. The `shark search` command provides cross-entity discovery.
   - **Workaround**: Use `shark bug list` or `shark search --type=bug`.

4. **Bug/Change-Card Slugged Key Support**
   - **Why**: Slugged keys (e.g., B001-login-fails-on-safari) require slug generation and dual-key lookup infrastructure for bugs and change-cards. This is a separate enhancement that can be added after the core key detection works.
   - **Future**: Can be added as a follow-on task if slug support is needed for bugs/change-cards.

---

## Dependencies & Integrations

### Dependencies

- **F04 (Bug CLI Commands)**: Bug service, repository, and model must exist. The `shark bug` command handlers must be implemented so that the unified dispatch can call them.
- **F05 (Change-Card CLI Commands)**: Change-card service, repository, and model must exist. The `shark change` command handlers must be implemented.
- **E17 (CLI Simplification)**: The unified command infrastructure (`shark get`, `shark status`, `shark delete`, `shark update`, `shark search`, `shark context`, `shark notes`, `shark view`, `shark history`) must be in place. F06 extends these existing dispatch points.
- **F01 (Database Schema)**: Bug and change-card database tables must exist with the workflow engine supporting bug and change-card levels.
- **F02 (Bug Entity Core)**: BugService and BugRepository interfaces must be defined and implemented.
- **F03 (Change-Card Entity Core)**: ChangeCardService and ChangeCardRepository interfaces must be defined and implemented.

### Integration Requirements

- **Key Service (`internal/keys/service.go`)**: Add `EntityTypeBug` and `EntityTypeChange` constants. Extend `Parse()` regex patterns. Extend `DetectEntityType()`.
- **CLI Helpers (`internal/cli/commands/helpers.go`)**: Extend `DetectEntityType()` function.
- **Models (`internal/models/`)**: Add `EntityTypeBug` and `EntityTypeChange` to `EntityType` constants and `ValidEntityTypes` map.
- **Service Accessors (`internal/cli/services_global.go`)**: Add `GetBugService()` and `GetChangeCardService()` accessor functions if not already added by F04/F05.
- **Search Repository (`internal/repository/search_repository.go`)**: Extend full-text search to include bugs and change-cards tables.

---

## Risk Mitigation

This feature addresses **Research Risk 1 (Cross-Cutting Entity Type Changes)**, rated HIGH probability / MEDIUM impact in the research report.

**Mitigation Strategy**:

1. **Pre-implementation dispatch inventory**: Before writing any code, run a comprehensive grep/search across the codebase for all entity-type switch statements, `DetectEntityType` calls, `EntityType` constants, and `ValidEntityTypes` references. Store the inventory as a checklist with file path, line number, and current entity types handled.

2. **Inventory-driven implementation**: Work through the inventory checklist systematically, adding `"bug"` and `"change"` cases to each dispatch point. Mark each item as complete.

3. **Post-implementation verification**: After all dispatch points are updated, re-run the inventory grep and diff against the original to confirm no dispatch points were missed. Any new dispatch points added by other features during development are caught.

4. **Integration test coverage**: Write integration tests that exercise every unified command with B### and C### keys. The test list maps 1:1 to the dispatch inventory.

5. **Compile-time safety**: Where possible, structure entity-type switches with explicit cases for all types and a `default` case that returns an error (not silent fallthrough). This ensures future entity types produce errors rather than silent failures.

---

## Open Questions & Assumptions

No open questions -- all requirements are clear.

**Resolved Assumptions**:
1. The key format for bugs is `B` followed by digits (B001, B42, B1000). No slug suffix is supported in the initial implementation. This matches the epic-level decision for `B###` format.
2. The key format for change-cards is `C` followed by digits (C001, C15). No slug suffix in initial implementation.
3. The entity type string for change-cards in code and CLI output is `"change"`, not `"change-card"` or `"changecard"`. This is for consistency with the short entity type names used elsewhere ("epic", "feature", "task", "bug", "change").
4. The `shark list` command is not extended for bugs/change-cards because they are standalone entities without the hierarchical parent-child relationship that `shark list` navigates.
5. Search indexing for bugs and change-cards uses the same full-text search approach as tasks (SQLite FTS5). The search index is rebuilt to include bug and change-card records.

---

*Last Updated*: 2026-03-03
