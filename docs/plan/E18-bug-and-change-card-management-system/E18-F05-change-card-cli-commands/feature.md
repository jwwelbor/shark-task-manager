---
feature_key: E18-F05-change-card-cli-commands
epic_key: E18
title: Change-Card CLI Commands
description: Implement the shark change command group with create, get, list, update, delete, and approve subcommands as thin wrappers over ChangeCardService.
---

# Change-Card CLI Commands

**Feature Key**: E18-F05
**Execution Order**: 5 (depends on F03; can be developed in parallel with F04)

---

## Epic

- **Epic PRD**: [Bug and Change-Card Management System](../epic.md)
- **Requirements**: [Requirements Catalog](../requirements.md)
- **Scope Boundaries**: [Scope](../scope.md)

---

## Goal

### Problem

The ChangeCardService (F03) provides all change-card business logic -- creation, approval, status transitions, link validation -- but there is no user-facing interface. Without CLI commands, developers cannot propose change-cards, product owners cannot approve them, and the entire change-card workflow defined in the epic is inaccessible.

### Solution

Implement `shark change` as a new Cobra command group in `internal/cli/commands/change.go` with subcommands that follow the established thin-wrapper pattern: parse arguments, call ChangeCardService, format output. Each subcommand is 15-25 lines. The command group includes CRUD operations, the domain-specific `approve` command, and delegates to existing NoteService and ContextService for notes and context.

### Impact

- Change-card creation completes in under 60 seconds of user interaction (measured: command invocation to CLI output)
- Product owners approve or decline change-cards with a single command
- All 10 subcommands follow established CLI patterns, requiring zero new learning for existing Shark users
- `--json` output on all commands enables AI agent and CI/CD pipeline consumption

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to create a change-card with a single CLI command so that I can propose small enhancements without interrupting my workflow.

**Acceptance Criteria**:
- [ ] `shark change create "Add dark mode toggle" --link=E07` creates a change-card with auto-generated key (C001, C002, etc.) in `proposed` status
- [ ] The command outputs the created change-card key and file path
- [ ] `--json` flag outputs the full change-card object as JSON
- [ ] `--link` flag is validated -- linking to a non-existent entity returns a clear error message naming the invalid key
- [ ] When `--link` is omitted, the change-card is created without a link (link fields are null)

**Story 2**: As a developer, I want to retrieve change-card details so that I can review a change-card's current state and linked context.

**Acceptance Criteria**:
- [ ] `shark change get C001` displays change-card details: key, title, status, linked entity, file path, timestamps
- [ ] `shark change get C001 --json` outputs the full change-card object as JSON
- [ ] `shark change get C001 --field status` outputs only the status value
- [ ] Getting a non-existent key returns exit code 1 with message "change-card not found: C999"

**Story 3**: As a product owner, I want to list change-cards with filters so that I can review pending proposals and track change-card throughput.

**Acceptance Criteria**:
- [ ] `shark change list` displays all non-completed change-cards in a table with columns: Key, Title, Status, Linked Entity, Created At
- [ ] `shark change list --status=proposed` filters to only change-cards in `proposed` status
- [ ] `shark change list --link=E07` filters to change-cards linked to epic E07 or any entity under E07
- [ ] `shark change list --json` outputs the full list as a JSON array
- [ ] When no change-cards match filters, an empty table is shown (no error)

**Story 4**: As a developer, I want to update change-card fields so that I can correct titles or add missing information.

**Acceptance Criteria**:
- [ ] `shark change update C001 --title="Revised title"` updates the change-card title
- [ ] The command outputs a success message with the updated change-card key
- [ ] `--json` flag outputs the updated change-card object as JSON
- [ ] Updating a non-existent key returns exit code 1

**Story 5**: As a developer, I want to delete a change-card so that I can remove proposals that are no longer relevant.

**Acceptance Criteria**:
- [ ] `shark change delete C001` prompts for confirmation before deleting
- [ ] `shark change delete C001 --force` deletes without confirmation
- [ ] Deleting removes the database record and the markdown file
- [ ] Deleting a non-existent key returns exit code 1
- [ ] After deletion, the key C001 is never reused for new change-cards

**Story 6**: As a product owner, I want to approve a change-card with a single command so that the approval workflow is clear, fast, and auditable.

**Acceptance Criteria**:
- [ ] `shark change approve C001` advances C001 from `proposed` to `approved` status
- [ ] The command outputs a success message confirming the approval
- [ ] If the change-card is not in `proposed` status, the command returns exit code 3 with message indicating the current status and that approval is only valid from `proposed`
- [ ] If the change-card does not exist, the command returns exit code 1
- [ ] Status history records the transition with timestamp

---

### Should-Have Stories

**Story 7**: As a product owner, I want to add notes to change-cards so that approval rationale and review comments are documented.

**Acceptance Criteria**:
- [ ] `shark change note add C001 --type=decision "Approved: aligns with Q2 UX goals"` adds a typed note to the change-card
- [ ] `shark change notes C001` lists all notes on the change-card in chronological order
- [ ] Note types follow the existing note type set: comment, decision, blocker, solution, reference, implementation, testing, future, question
- [ ] Notes are stored via the existing NoteService with `entity_type="change"`

**Story 8**: As a developer, I want to manage context fields on change-cards so that I can store structured metadata like effort estimate and target area.

**Acceptance Criteria**:
- [ ] `shark change context set C001 --field effort --value "small"` sets a context field
- [ ] `shark change context get C001` returns all context fields as a JSON object
- [ ] `shark change context clear C001 --field effort` removes a specific context field
- [ ] Context operations are delegated to the existing ContextService with `entity_type="change"`

**Story 9**: As a developer, I want to view a change-card's markdown file so that I can read the full description and justification.

**Acceptance Criteria**:
- [ ] `shark view C001` renders the change-card markdown file to the terminal
- [ ] If the markdown file does not exist, the command returns an error indicating the file is missing

---

### Edge Case and Error Stories

**Error Story 1**: As a developer, when I create a change-card with an invalid link key format, I want a clear error message so that I can fix the command and retry.

**Acceptance Criteria**:
- [ ] `shark change create "Title" --link=INVALID` returns an error message: "invalid link key format: INVALID. Expected E##, E##-F##, or E##-F##-###"
- [ ] The change-card is not created (no partial record in database or filesystem)

**Error Story 2**: As a product owner, when I approve a change-card that is already completed, I want an error message that tells me the current status so that I understand why the operation failed.

**Acceptance Criteria**:
- [ ] `shark change approve C001` when C001 is in `completed` status returns exit code 3 with message: "cannot approve change-card C001: current status is 'completed', approval requires status 'proposed'"
- [ ] No status change occurs

**Error Story 3**: As a developer, when I list change-cards with an invalid status filter, I want an error so that I know the valid status values.

**Acceptance Criteria**:
- [ ] `shark change list --status=invalid_status` returns an error listing valid statuses: proposed, approved, in_progress, completed, declined

---

## Requirements

### Functional Requirements

**Category: Command Registration**

1. **F05-REQ-001**: Change Command Group Registration
   - **Description**: A `shark change` parent Cobra command is registered via `init()` in `internal/cli/commands/change.go`. All change-card subcommands are nested under this parent.
   - **Traces to**: REQ-F-011 (Change-Card CRUD Commands)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark change --help` displays all available subcommands with descriptions
     - [ ] `shark change` with no subcommand displays help text (not an error)

**Category: CRUD Operations**

2. **F05-REQ-002**: Create Command
   - **Description**: `shark change create "<title>" [--link=KEY]` creates a change-card by calling `ChangeCardService.CreateChangeCard`. Title is a positional argument. `--link` is an optional flag.
   - **Traces to**: REQ-F-007 (Change-Card Entity Creation), REQ-F-008 (Change-Card Entity Linking)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command parses title from positional arg and --link from flag
     - [ ] Calls ChangeCardService.CreateChangeCard with parsed input
     - [ ] Outputs success message with key and file path, or JSON with --json flag
     - [ ] Returns exit code 0 on success, exit code 2 on service error

3. **F05-REQ-003**: Get Command
   - **Description**: `shark change get <key> [--json] [--field NAME]` retrieves change-card details by calling `ChangeCardService.GetChangeCard`.
   - **Traces to**: REQ-F-011 (Change-Card CRUD Commands)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command parses key from positional arg
     - [ ] Outputs formatted details or JSON based on --json flag
     - [ ] `--field` extracts a single field value from the JSON response
     - [ ] Returns exit code 1 when change-card not found

4. **F05-REQ-004**: List Command
   - **Description**: `shark change list [--status=S] [--link=KEY] [--json]` lists change-cards with optional filters by calling `ChangeCardService.ListChangeCards`.
   - **Traces to**: REQ-F-011 (Change-Card CRUD Commands), REQ-F-018 (Filtering by Linked Entity)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Parses --status, --link, --json flags
     - [ ] Outputs table with Key, Title, Status, Linked Entity, Created At columns
     - [ ] Empty results produce an empty table, not an error
     - [ ] JSON output is an array of change-card objects

5. **F05-REQ-005**: Update Command
   - **Description**: `shark change update <key> [--title="..."]` updates change-card fields by calling `ChangeCardService.UpdateChangeCard`.
   - **Traces to**: REQ-F-011 (Change-Card CRUD Commands)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Parses key from positional arg and --title from flag
     - [ ] Outputs success message or JSON with --json flag
     - [ ] Returns exit code 1 when change-card not found

6. **F05-REQ-006**: Delete Command
   - **Description**: `shark change delete <key> [--force]` deletes a change-card by calling `ChangeCardService.DeleteChangeCard`.
   - **Traces to**: REQ-F-011 (Change-Card CRUD Commands)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Without --force, prompts for confirmation
     - [ ] With --force, deletes immediately
     - [ ] Returns exit code 1 when change-card not found

**Category: Domain-Specific Operations**

7. **F05-REQ-007**: Approve Command
   - **Description**: `shark change approve <key>` is a domain-specific command that advances a change-card from `proposed` to `approved` by calling `ChangeCardService.ApproveChangeCard`.
   - **Traces to**: REQ-F-010 (Change-Card Approval Command)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Parses key from positional arg
     - [ ] Calls ChangeCardService.ApproveChangeCard
     - [ ] Returns exit code 3 with descriptive message when status is not `proposed`
     - [ ] Returns exit code 1 when change-card not found
     - [ ] Outputs success message confirming the approval

**Category: Notes and Context**

8. **F05-REQ-008**: Note Subcommands
   - **Description**: `shark change note add <key> --type=TYPE "content"` and `shark change notes <key>` delegate to existing NoteService with `entity_type="change"`.
   - **Traces to**: REQ-F-017 (Change-Card Notes and Context)
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `note add` delegates to NoteService.AddNote with entity_type="change"
     - [ ] `notes` delegates to NoteService.ListNotes with entity_type="change"
     - [ ] Supports all existing note types

9. **F05-REQ-009**: Context Subcommands
   - **Description**: `shark change context set/get/clear <key>` delegates to existing ContextService with `entity_type="change"`.
   - **Traces to**: REQ-F-017 (Change-Card Notes and Context)
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `context set` delegates to ContextService with entity_type="change"
     - [ ] `context get` returns all context fields
     - [ ] `context clear` removes a specific context field

**Category: View Integration**

10. **F05-REQ-010**: View Command Integration
    - **Description**: `shark view C001` renders the change-card markdown file to the terminal, extending the existing view command to recognize C### keys.
    - **Traces to**: REQ-F-011 (Change-Card CRUD Commands)
    - **Priority**: Should-Have
    - **Acceptance Criteria**:
      - [ ] `shark view C001` retrieves the file path from the change-card record and renders the markdown
      - [ ] Returns an error if the file does not exist on disk

---

### Non-Functional Requirements

**CLI Pattern Consistency**

1. **F05-NFR-001**: Established CLI Pattern Adherence
   - **Description**: All change-card commands follow the CLI patterns documented in `.claude/rules/cli/commands.md` and `.claude/rules/cli/patterns.md`. This includes flag naming conventions, output formatting, error message structure, JSON response structure, and exit codes.
   - **Traces to**: REQ-NF-005 (CLI Pattern Consistency)
   - **Measurement**: Manual review against CLI pattern documentation
   - **Target**: Zero deviations from established patterns without documented justification

**Performance**

2. **F05-NFR-002**: Command Overhead
   - **Description**: CLI command overhead (argument parsing + output formatting) adds no more than 50ms to the service call time. Total creation time under 500ms local, under 2s on Turso.
   - **Traces to**: REQ-NF-001 (Bug/Change Creation Speed)
   - **Measurement**: Shell timing of `time shark change create "test" --json`
   - **Target**: < 500ms local, < 2s cloud

**Architecture**

3. **F05-NFR-003**: Thin Wrapper Architecture
   - **Description**: Every command handler is a thin wrapper with exactly three responsibilities: (1) parse arguments and flags, (2) call a single ChangeCardService method, (3) format output. No business logic, no direct repository calls, no transaction management in commands.
   - **Traces to**: REQ-NF-005, architecture rules in `.claude/rules/cli/patterns.md`
   - **Measurement**: Code review verifying each handler follows the parse-call-format pattern
   - **Target**: Zero business logic lines in command handlers

**Testing**

4. **F05-NFR-004**: Mocked Service Tests
   - **Description**: All CLI tests use mocked ChangeCardService. No real database access in CLI tests, following the testing architecture rule that only repository tests use real database.
   - **Traces to**: Testing architecture in `.claude/rules/testing/architecture.md`
   - **Measurement**: Zero `test.GetTestDB()` calls in CLI test files
   - **Target**: 100% of CLI tests use mock services

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create and Retrieve a Change-Card**
- **Given** Shark is initialized and the change-card entity type is enabled
- **When** a developer runs `shark change create "Add keyboard shortcuts" --link=E17`
- **Then** a change-card is created with a unique key (C###), status `proposed`, and a link to E17
- **And** running `shark change get C###` returns the change-card with all fields populated

**Scenario 2: Approve a Change-Card**
- **Given** a change-card C001 exists in `proposed` status
- **When** a product owner runs `shark change approve C001`
- **Then** the change-card status changes to `approved`
- **And** the status history records the transition from `proposed` to `approved`

**Scenario 3: Filter Change-Cards by Status**
- **Given** change-cards exist in statuses: proposed, approved, completed
- **When** a product owner runs `shark change list --status=proposed`
- **Then** only change-cards in `proposed` status are returned

**Scenario 4: Invalid Approval Attempt**
- **Given** a change-card C001 exists in `approved` status
- **When** a product owner runs `shark change approve C001`
- **Then** the command returns exit code 3 with a message indicating approval requires `proposed` status
- **And** no status change occurs

**Scenario 5: JSON Output for Machine Consumption**
- **Given** change-cards exist in the database
- **When** an AI agent runs `shark change list --json`
- **Then** the output is a valid JSON array of change-card objects
- **And** each object contains: key, title, status, linked_entity_type, linked_entity_key, created_at, updated_at

**Scenario 6: Delete with Confirmation**
- **Given** a change-card C001 exists
- **When** a developer runs `shark change delete C001 --force`
- **Then** the database record and markdown file are both deleted
- **And** `shark change get C001` returns exit code 1

---

## Out of Scope

### Explicitly Excluded

1. **Unified Command Auto-Detection (`shark get C001`, `shark status advance C001`)**
   - **Why**: This is the responsibility of F06 (Unified CLI Integration and Key Auto-Detection). F05 creates the entity-specific `shark change` commands; F06 wires C### keys into the unified dispatch.
   - **Future**: F06 must be implemented after F05.
   - **Workaround**: Users use `shark change get C001` instead of `shark get C001` until F06 is complete.

2. **Dashboard and Analytics Display**
   - **Why**: This is the responsibility of F07 (Dashboard and Analytics Integration). F05 is limited to CRUD and lifecycle commands.
   - **Workaround**: Users run `shark change list --json` and process output manually.

3. **Change-Card Promotion to Feature**
   - **Why**: REQ-F-020 is a Could Have requirement, deferred to a follow-on epic per the scope boundaries document.
   - **Workaround**: Product owners manually create a feature and decline the change-card with a note referencing the new feature.

4. **Batch Operations (bulk create, bulk approve)**
   - **Why**: No epic requirement for batch operations. Shell scripting with `shark change create` in a loop provides equivalent functionality.

---

## Dependencies and Integrations

### Dependencies

- **F03 (Change-Card Entity Core)**: F05 depends on `ChangeCardService` and all its methods: `CreateChangeCard`, `GetChangeCard`, `ListChangeCards`, `UpdateChangeCard`, `DeleteChangeCard`, `ApproveChangeCard`. F03 must be complete before F05 can be implemented.
- **F01 (Database Schema)**: Indirectly depends on F01 through F03. The `change_cards` table must exist.
- **Existing NoteService** (`internal/services/note_service.go`): For `shark change note add` and `shark change notes` commands. Requires the entity_notes CHECK constraint migration (F01) to allow `entity_type="change"`.
- **Existing ContextService** (`internal/services/context_service.go`): For `shark change context set/get/clear` commands. Requires a `case "change":` branch in the entity type switch statement (F01 or F03).
- **Service Accessor** (`internal/cli/services_global.go`): Requires `GetChangeCardService()` function to be implemented, following the `GetTaskService()` / `GetIdeaService()` pattern.

### Integration Requirements

- **Cobra CLI Framework**: `shark change` parent command and all subcommands registered via `init()` function in `internal/cli/commands/change.go`.
- **Global Flags**: All commands inherit `--json`, `--field`, `--no-color`, `--verbose` from the root command via `cli.GlobalConfig`.
- **Output Functions**: All commands use `cli.OutputJSON()`, `cli.OutputTable()`, `cli.Success()`, `cli.Error()` for consistent formatting.

---

## Requirements Traceability

| Epic Requirement | Feature Requirement | Coverage |
|-----------------|---------------------|----------|
| REQ-F-007 (Change-Card Entity Creation) | F05-REQ-002 | `shark change create` command |
| REQ-F-008 (Change-Card Entity Linking) | F05-REQ-002 | `--link` flag on create |
| REQ-F-009 (Change-Card Status Workflow) | Via ChangeCardService | Status advances delegated to service |
| REQ-F-010 (Change-Card Approval Command) | F05-REQ-007 | `shark change approve` command |
| REQ-F-011 (Change-Card CRUD Commands) | F05-REQ-002 through F05-REQ-006 | create, get, list, update, delete |
| REQ-F-017 (Change-Card Notes and Context) | F05-REQ-008, F05-REQ-009 | note and context subcommands |
| REQ-F-018 (Filtering by Linked Entity) | F05-REQ-004 | `--link` flag on list |
| REQ-NF-001 (Creation Speed) | F05-NFR-002 | < 500ms local, < 2s cloud |
| REQ-NF-005 (CLI Pattern Consistency) | F05-NFR-001, F05-NFR-003 | Thin wrapper pattern, established conventions |

---

## Open Questions and Assumptions

No open questions -- all requirements are clear.

**Resolved Assumptions**:
1. ChangeCardService methods and DTOs are defined by F03. This PRD assumes the service interface includes: `CreateChangeCard`, `GetChangeCard`, `ListChangeCards`, `UpdateChangeCard`, `DeleteChangeCard`, `ApproveChangeCard`. If F03 defines different method names, the CLI commands adapt to match.
2. The `shark view` command extension to support C### keys is a small change (add a case to the key detection switch). This is included in F05 scope, not deferred to F06, because it operates on the change-card entity directly rather than through unified dispatch.
3. The `--link` flag on `shark change list` filters by linked entity key using a prefix match (e.g., `--link=E07` matches change-cards linked to E07, E07-F01, or E07-F01-001). This behavior matches the epic requirement REQ-F-018.

---

*Last Updated*: 2026-03-03
