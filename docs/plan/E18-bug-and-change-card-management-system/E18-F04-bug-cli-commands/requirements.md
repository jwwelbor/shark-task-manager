# Feature Requirements: Bug CLI Commands

**Feature Key**: E18-F04
**Epic**: [E18 - Bug and Change-Card Management System](../epic.md)
**Parent PRD**: [Feature PRD](./prd.md)

---

## Overview

This document contains feature-level acceptance criteria for all `shark bug` CLI commands. These are incremental over the epic-level requirements (REQ-F-001 through REQ-F-018) and specify exact CLI behavior: flag combinations, output formatting, error messages, and edge cases.

**Traceability**: Each requirement below maps to one or more epic requirements and specifies the CLI-specific acceptance criteria that the epic requirements do not cover.

---

## CRUD Commands

### F04-REQ-01: Bug Create Command

**Traces to**: REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-006

**Command**: `shark bug create "<title>" [--severity=S] [--link=KEY]`

**Acceptance Criteria**:

#### AC-01a: Basic creation with title only
Given the bug table is empty
When the user runs `shark bug create "Login page crashes on submit"`
Then a bug is created with key `B001`, status `reported`, severity `medium`
And the CLI outputs: `Created bug B001`
And a markdown file is created at `docs/bugs/B001.md`

#### AC-01b: Creation with severity flag
Given the bug table is empty
When the user runs `shark bug create "API timeout" --severity=critical`
Then a bug is created with severity `critical`
And the CLI outputs: `Created bug B001`

#### AC-01c: Creation with link to existing entity
Given epic E07 exists in the database
When the user runs `shark bug create "Button misaligned" --link=E07`
Then a bug is created with linked_entity_type `epic` and linked_entity_key `E07`
And the CLI outputs: `Created bug B001`

#### AC-01d: Creation with link to feature
Given feature E07-F01 exists in the database
When the user runs `shark bug create "Form validation broken" --link=E07-F01`
Then a bug is created with linked_entity_type `feature` and linked_entity_key `E07-F01`

#### AC-01e: Creation with link to task
Given task E07-F01-001 exists in the database
When the user runs `shark bug create "Edge case failure" --link=E07-F01-001`
Then a bug is created with linked_entity_type `task` and linked_entity_key `E07-F01-001`

#### AC-01f: Creation with invalid link
Given entity E99 does not exist in the database
When the user runs `shark bug create "Some bug" --link=E99`
Then the CLI returns an error: `Linked entity not found: E99`
And no bug is created
And the exit code is 1

#### AC-01g: Creation with invalid severity
When the user runs `shark bug create "Some bug" --severity=extreme`
Then the CLI returns an error containing `invalid severity`
And the error message lists valid values: `critical, high, medium, low`
And no bug is created
And the exit code is 3

#### AC-01h: Creation with empty title
When the user runs `shark bug create ""`
Then the CLI returns an error: `title cannot be empty`
And no bug is created

#### AC-01i: JSON output on creation
When the user runs `shark bug create "New bug" --json`
Then the output is valid JSON containing `key`, `title`, `status`, `severity`, `created_at`
And the JSON `status` field is `reported`
And the JSON `severity` field is `medium`

#### AC-01j: Auto-incrementing keys
Given bugs B001 and B002 already exist
When the user runs `shark bug create "Third bug"`
Then the new bug gets key `B003`

---

### F04-REQ-02: Bug Get Command

**Traces to**: REQ-F-006

**Command**: `shark bug get <key> [--json] [--field NAME]`

**Acceptance Criteria**:

#### AC-02a: Get bug details
Given bug B001 exists with title "Login crash" and severity "high"
When the user runs `shark bug get B001`
Then the CLI outputs the bug details including key, title, status, severity, linked entity (if any), created date, and updated date

#### AC-02b: Get bug with JSON output
Given bug B001 exists
When the user runs `shark bug get B001 --json`
Then the output is valid JSON matching the Bug model schema

#### AC-02c: Field extraction
Given bug B001 exists with severity "critical"
When the user runs `shark bug get B001 --field severity`
Then the output is exactly: `critical`

#### AC-02d: Bug not found
Given bug B999 does not exist
When the user runs `shark bug get B999`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

#### AC-02e: Case-insensitive key
Given bug B001 exists
When the user runs `shark bug get b001`
Then the bug is found and returned successfully

#### AC-02f: Invalid field name
Given bug B001 exists
When the user runs `shark bug get B001 --field nonexistent_field`
Then the CLI returns an error indicating the field does not exist
And the exit code is 4

---

### F04-REQ-03: Bug List Command

**Traces to**: REQ-F-006, REQ-F-018

**Command**: `shark bug list [--status=S] [--severity=S] [--link=KEY] [--json]`

**Acceptance Criteria**:

#### AC-03a: List all bugs
Given 3 bugs exist in the database
When the user runs `shark bug list`
Then a table is displayed with columns: KEY, TITLE, STATUS, SEVERITY, LINKED TO
And all 3 bugs appear in the table

#### AC-03b: Filter by status
Given bugs exist with statuses `reported`, `triaged`, and `resolved`
When the user runs `shark bug list --status=reported`
Then only bugs with status `reported` appear

#### AC-03c: Filter by severity
Given bugs exist with severities `critical`, `high`, `medium`
When the user runs `shark bug list --severity=critical`
Then only bugs with severity `critical` appear

#### AC-03d: Filter by linked entity
Given bug B001 is linked to E07-F01 and bug B002 is linked to E07-F02
When the user runs `shark bug list --link=E07-F01`
Then only B001 appears

#### AC-03e: Combined filters
Given multiple bugs exist
When the user runs `shark bug list --status=reported --severity=high`
Then only bugs matching both criteria appear

#### AC-03f: JSON list output
When the user runs `shark bug list --json`
Then the output is a valid JSON array of bug objects

#### AC-03g: Empty result
Given no bugs exist
When the user runs `shark bug list`
Then the CLI outputs a message indicating no bugs found (not an error)
And the exit code is 0

#### AC-03h: Link filter with epic scope
Given bugs are linked to E07-F01 and E07-F02
When the user runs `shark bug list --link=E07`
Then bugs linked to any entity under epic E07 appear

---

### F04-REQ-04: Bug Update Command

**Traces to**: REQ-F-006

**Command**: `shark bug update <key> [--title="..."] [--severity=S]`

**Acceptance Criteria**:

#### AC-04a: Update title
Given bug B001 exists with title "Old title"
When the user runs `shark bug update B001 --title="New title"`
Then the bug title is updated to "New title"
And the CLI outputs a success message

#### AC-04b: Update severity
Given bug B001 exists with severity "medium"
When the user runs `shark bug update B001 --severity=critical`
Then the bug severity is updated to "critical"

#### AC-04c: Update both title and severity
Given bug B001 exists
When the user runs `shark bug update B001 --title="Updated" --severity=low`
Then both fields are updated in a single operation

#### AC-04d: Bug not found on update
Given bug B999 does not exist
When the user runs `shark bug update B999 --title="New"`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

#### AC-04e: No update flags provided
When the user runs `shark bug update B001` (no flags)
Then the CLI outputs an error indicating at least one update flag is required

#### AC-04f: JSON output on update
Given bug B001 exists
When the user runs `shark bug update B001 --title="New" --json`
Then the output is valid JSON containing the updated bug

---

### F04-REQ-05: Bug Delete Command

**Traces to**: REQ-F-006

**Command**: `shark bug delete <key> [--force]`

**Acceptance Criteria**:

#### AC-05a: Delete with confirmation
Given bug B001 exists
When the user runs `shark bug delete B001`
Then the CLI prompts for confirmation
And if confirmed, the bug and its markdown file are deleted
And the CLI outputs a success message

#### AC-05b: Delete with force flag
Given bug B001 exists
When the user runs `shark bug delete B001 --force`
Then the bug is deleted without confirmation prompt

#### AC-05c: Delete non-existent bug
Given bug B999 does not exist
When the user runs `shark bug delete B999`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

#### AC-05d: JSON output on delete
Given bug B001 exists
When the user runs `shark bug delete B001 --force --json`
Then the output is valid JSON confirming the deletion

---

## Lifecycle Commands

### F04-REQ-06: Bug Triage Command

**Traces to**: REQ-F-005

**Command**: `shark bug triage <key> --severity=S [--assign=AGENT]`

**Acceptance Criteria**:

#### AC-06a: Triage with severity
Given bug B001 exists in status `reported`
When the user runs `shark bug triage B001 --severity=high`
Then the bug status advances to `triaged`
And the severity is set to `high`
And the CLI outputs a success message

#### AC-06b: Triage with severity and assignment
Given bug B001 exists in status `reported`
When the user runs `shark bug triage B001 --severity=medium --assign=developer`
Then the bug status advances to `triaged`
And the severity is set to `medium`
And the agent is set to `developer`

#### AC-06c: Triage already triaged bug
Given bug B001 exists in status `triaged`
When the user runs `shark bug triage B001 --severity=high`
Then the CLI outputs an error: `Cannot triage bug B001: current status is 'triaged', expected 'reported'`
And the exit code is 3

#### AC-06d: Triage without severity flag
When the user runs `shark bug triage B001`
Then the CLI outputs an error indicating `--severity` is required for triage
And the exit code is 3

#### AC-06e: JSON output on triage
Given bug B001 exists in status `reported`
When the user runs `shark bug triage B001 --severity=high --json`
Then the output is valid JSON with status `triaged` and severity `high`

#### AC-06f: Triage non-existent bug
Given bug B999 does not exist
When the user runs `shark bug triage B999 --severity=high`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

---

## Note and Context Commands

### F04-REQ-07: Bug Note Commands

**Traces to**: REQ-F-015

**Commands**: `shark bug note add <key> --type=TYPE "<content>"`, `shark bug notes <key>`

**Acceptance Criteria**:

#### AC-07a: Add note to bug
Given bug B001 exists
When the user runs `shark bug note add B001 --type=comment "Reproduced on Safari 17.2"`
Then a note is created with entity_type `bug`, entity_id matching B001, note_type `comment`
And the CLI outputs confirmation

#### AC-07b: List notes on bug
Given bug B001 has 2 notes
When the user runs `shark bug notes B001`
Then both notes are displayed with type, content, and timestamp

#### AC-07c: Add note with decision type
Given bug B001 exists
When the user runs `shark bug note add B001 --type=decision "Root cause is race condition in session handler"`
Then a note with type `decision` is created

#### AC-07d: Notes on non-existent bug
Given bug B999 does not exist
When the user runs `shark bug notes B999`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

---

### F04-REQ-08: Bug Context Commands

**Traces to**: REQ-F-015

**Commands**: `shark bug context set/get/clear <key>`

**Acceptance Criteria**:

#### AC-08a: Set context field
Given bug B001 exists
When the user runs `shark bug context set B001 --field environment --value "Safari 17.2 on macOS 14.3"`
Then the context field `environment` is stored with value `Safari 17.2 on macOS 14.3`

#### AC-08b: Get context
Given bug B001 has context fields `environment` and `component`
When the user runs `shark bug context get B001`
Then both context fields and their values are displayed

#### AC-08c: Get context with JSON
Given bug B001 has context fields
When the user runs `shark bug context get B001 --json`
Then the output is valid JSON containing all context key-value pairs

#### AC-08d: Clear context field
Given bug B001 has context field `environment`
When the user runs `shark bug context clear B001 --field environment`
Then the field is removed from context

#### AC-08e: Context on non-existent bug
Given bug B999 does not exist
When the user runs `shark bug context get B999`
Then the CLI outputs: `Bug not found: B999`
And the exit code is 1

---

## Command Registration

### F04-REQ-09: Command Group Registration

**Traces to**: REQ-NF-005

**Acceptance Criteria**:

#### AC-09a: Parent command registration
Given Shark CLI is built
When the user runs `shark bug --help`
Then the help text shows all available subcommands: create, get, list, update, delete, triage, note, notes, context

#### AC-09b: Command group placement
The `shark bug` command is registered under the `advanced` command group (same as `shark idea`)

#### AC-09c: Global flags inherited
All `shark bug` subcommands inherit global flags: `--json`, `--field`, `--no-color`, `--verbose`, `--db`, `--config`

#### AC-09d: Help text quality
Each subcommand has:
- A Short description (under 60 characters)
- A Long description with usage examples
- Args validation (e.g., `cobra.ExactArgs(1)` for key-based commands)

---

## Service Accessor

### F04-REQ-10: GetBugService Accessor

**Traces to**: Architecture (service accessor pattern)

**Acceptance Criteria**:

#### AC-10a: Accessor function exists
`internal/cli/services_global.go` contains a `GetBugService()` function that returns `*services.BugService`

#### AC-10b: Dependency wiring
`GetBugService()` creates a BugRepository from the global DB connection, obtains the workflow service, and passes both to `services.NewBugService()`

#### AC-10c: Panic on DB failure
If the database connection fails, `GetBugService()` panics with a descriptive message (matching the existing `GetTaskService()` pattern)

---

## Cross-Cutting Acceptance Criteria

### F04-REQ-11: CLI Pattern Consistency

**Traces to**: REQ-NF-005

**Acceptance Criteria**:

#### AC-11a: Output function usage
All commands use `cli.OutputJSON()` for JSON output, `cli.OutputTable()` for tables, `cli.Success()` / `cli.Error()` for messages

#### AC-11b: No business logic in commands
No command handler contains: workflow validation, status checks, repository calls, transaction management, filtering logic, or key generation. All business logic is delegated to BugService.

#### AC-11c: Error propagation
Command handlers return errors from service calls. They do not catch and swallow errors. Exit code mapping follows the standard pattern (1=not found, 2=DB error, 3=invalid state, 4=field not found).

#### AC-11d: Flag naming consistency
Flag names match existing entity commands:
- `--severity` (not `--sev` or `--priority`)
- `--link` (not `--linked-to` or `--parent`)
- `--force` (not `--yes` or `--confirm`)
- `--assign` (not `--agent` on triage -- note: triage uses `--assign` because it is the action of assigning to an agent, distinct from `--agent` which is a filter)
- `--type` on notes (matching existing note commands)
- `--field` and `--value` on context (matching existing context commands)

---

*Last Updated*: 2026-03-03
