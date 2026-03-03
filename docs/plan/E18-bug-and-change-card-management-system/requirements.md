# Requirements

**Epic**: [Bug and Change-Card Management System](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for adding bug and change-card entity types to Shark Task Manager.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Bug Entity -- Data Model

**REQ-F-001**: Bug Entity Creation
- **Description**: The system must support creating bug entities with a unique key in the format `B###` (e.g., B001, B042), a title, a status, and an optional severity field.
- **User Story**: As a developer, I want to create a bug report with a single CLI command so that I can capture defects without interrupting my workflow.
- **Acceptance Criteria**:
  - [ ] `shark bug create "<title>"` creates a bug with an auto-generated key (B001, B002, etc.)
  - [ ] Bug is persisted in the database with status `reported`
  - [ ] Bug key is globally unique and auto-incremented
  - [ ] A markdown file is generated at `docs/bugs/B###.md` with frontmatter and template content
- **Related Journey**: Journey 1, Step 1

**REQ-F-002**: Bug Severity Tracking
- **Description**: Each bug must have a severity field with values: critical, high, medium, low. Severity defaults to medium if not specified at creation.
- **User Story**: As a QA engineer, I want to set and filter by severity so that I can prioritize critical bugs first.
- **Acceptance Criteria**:
  - [ ] `shark bug create "<title>" --severity=high` sets severity at creation
  - [ ] `shark bug list --severity=critical` filters bugs by severity
  - [ ] Severity is stored as a context field, queryable via `shark bug get B001 --field severity`
  - [ ] Default severity is `medium` when not specified
- **Related Journey**: Journey 1, Step 2

**REQ-F-003**: Bug Entity Linking
- **Description**: Bugs must support an optional link to an existing epic, feature, or task, indicating where the defect was found.
- **User Story**: As a developer, I want to link a bug to the feature where I found it so that the team can see which features have quality issues.
- **Acceptance Criteria**:
  - [ ] `shark bug create "<title>" --link=E07-F01` creates a bug linked to feature E07-F01
  - [ ] `shark bug create "<title>" --link=E07-F01-001` creates a bug linked to task E07-F01-001
  - [ ] `shark bug create "<title>" --link=E07` creates a bug linked to epic E07
  - [ ] Links are validated -- the system rejects links to non-existent entities
  - [ ] `shark bug get B001` displays the linked entity in its output
- **Related Journey**: Journey 1, Step 1; Journey 3, Step 1

#### Bug Entity -- Workflow

**REQ-F-004**: Bug Status Workflow
- **Description**: Bugs must follow a dedicated status workflow: `reported -> triaged -> in_fix -> in_verification -> resolved`. Additional terminal statuses: `wont_fix`, `duplicate`.
- **User Story**: As a QA engineer, I want a bug-specific workflow with triage and verification steps so that every bug goes through quality gates before resolution.
- **Acceptance Criteria**:
  - [ ] `shark status advance B001` advances the bug through the defined workflow
  - [ ] `shark status options B001` shows valid next statuses for the current state
  - [ ] Invalid transitions are rejected with a clear error message
  - [ ] `shark status set B001 wont_fix` and `shark status set B001 duplicate` are valid from any non-terminal status
  - [ ] Status history is recorded in the database for audit trail
- **Related Journey**: Journey 1, Steps 2-5

**REQ-F-005**: Bug Triage Command
- **Description**: A dedicated `shark bug triage` command that sets severity and optionally assigns the bug in a single operation, advancing the status to `triaged`.
- **User Story**: As a QA engineer, I want to triage a bug with one command so that I can process the bug queue efficiently.
- **Acceptance Criteria**:
  - [ ] `shark bug triage B001 --severity=high --assign=developer` sets severity, assigns, and advances to `triaged`
  - [ ] Triage is only valid when bug is in `reported` status
  - [ ] If bug is already triaged, the command returns an error with the current status
- **Related Journey**: Journey 1, Step 2

#### Bug Entity -- CLI

**REQ-F-006**: Bug CRUD Commands
- **Description**: Full CRUD operations for bugs: create, get, list, update, delete.
- **User Story**: As a developer, I want standard CRUD commands for bugs so that I can manage defects using the same patterns I use for tasks.
- **Acceptance Criteria**:
  - [ ] `shark bug create "<title>" [--severity=S] [--link=KEY]` creates a bug
  - [ ] `shark bug get B001 [--json] [--field NAME]` retrieves bug details
  - [ ] `shark bug list [--status=S] [--severity=S] [--link=KEY] [--json]` lists bugs with filters
  - [ ] `shark bug update B001 [--title="..."] [--severity=S]` updates bug fields
  - [ ] `shark bug delete B001 [--force]` deletes a bug
  - [ ] All commands support `--json` output for machine consumption
- **Related Journey**: Journey 1, all steps

#### Change-Card Entity -- Data Model

**REQ-F-007**: Change-Card Entity Creation
- **Description**: The system must support creating change-card entities with a unique key in the format `C###` (e.g., C001, C015), a title, and a status.
- **User Story**: As a developer, I want to propose small enhancements as change-cards so that improvement ideas are captured and tracked without full feature overhead.
- **Acceptance Criteria**:
  - [ ] `shark change create "<title>"` creates a change-card with auto-generated key (C001, C002, etc.)
  - [ ] Change-card is persisted in the database with status `proposed`
  - [ ] Change-card key is globally unique and auto-incremented
  - [ ] A markdown file is generated at `docs/changes/C###.md` with frontmatter and template content
- **Related Journey**: Journey 2, Step 1

**REQ-F-008**: Change-Card Entity Linking
- **Description**: Change-cards must support an optional link to an existing epic or feature, indicating what area the enhancement improves.
- **User Story**: As a developer, I want to link a change-card to the epic it enhances so that product owners can see proposed improvements in context.
- **Acceptance Criteria**:
  - [ ] `shark change create "<title>" --link=E07` creates a change-card linked to epic E07
  - [ ] `shark change create "<title>" --link=E07-F01` creates a change-card linked to feature E07-F01
  - [ ] Links are validated -- the system rejects links to non-existent entities
  - [ ] `shark change get C001` displays the linked entity
- **Related Journey**: Journey 2, Step 1

#### Change-Card Entity -- Workflow

**REQ-F-009**: Change-Card Status Workflow
- **Description**: Change-cards must follow a dedicated status workflow: `proposed -> approved -> in_progress -> completed`. Additional terminal status: `declined`.
- **User Story**: As a product owner, I want a simple approve/decline workflow for change-cards so that I can control which enhancements get implemented.
- **Acceptance Criteria**:
  - [ ] `shark status advance C001` advances the change-card through the defined workflow
  - [ ] `shark status options C001` shows valid next statuses
  - [ ] Invalid transitions are rejected with a clear error message
  - [ ] `shark status set C001 declined` is valid from `proposed` status
  - [ ] Status history is recorded for audit trail
- **Related Journey**: Journey 2, Steps 2-4

**REQ-F-010**: Change-Card Approval Command
- **Description**: A dedicated `shark change approve` command that advances a change-card from `proposed` to `approved`.
- **User Story**: As a product owner, I want a single command to approve change-cards so that the approval workflow is clear and auditable.
- **Acceptance Criteria**:
  - [ ] `shark change approve C001` advances C001 from `proposed` to `approved`
  - [ ] Approval is only valid when change-card is in `proposed` status
  - [ ] If already approved, returns error with current status
- **Related Journey**: Journey 2, Step 3

#### Change-Card Entity -- CLI

**REQ-F-011**: Change-Card CRUD Commands
- **Description**: Full CRUD operations for change-cards: create, get, list, update, delete.
- **User Story**: As a developer, I want standard CRUD commands for change-cards so that I can manage enhancement proposals consistently.
- **Acceptance Criteria**:
  - [ ] `shark change create "<title>" [--link=KEY]` creates a change-card
  - [ ] `shark change get C001 [--json] [--field NAME]` retrieves change-card details
  - [ ] `shark change list [--status=S] [--link=KEY] [--json]` lists change-cards with filters
  - [ ] `shark change update C001 [--title="..."]` updates change-card fields
  - [ ] `shark change delete C001 [--force]` deletes a change-card
  - [ ] All commands support `--json` output
- **Related Journey**: Journey 2, all steps

#### Unified CLI Integration

**REQ-F-012**: Core Command Auto-Detection for Bugs and Change-Cards
- **Description**: The unified `shark get`, `shark status`, and `shark search` commands must auto-detect bugs (B###) and change-cards (C###) by key prefix, following the same pattern used for epics (E##), features (F##), and tasks (T-E##-F##-###).
- **User Story**: As a developer, I want `shark get B001` and `shark status advance C001` to work without specifying entity type so that the unified CLI experience extends to the new entity types.
- **Acceptance Criteria**:
  - [ ] `shark get B001` returns bug details (auto-detected from B prefix)
  - [ ] `shark get C001` returns change-card details (auto-detected from C prefix)
  - [ ] `shark status advance B001` advances the bug workflow
  - [ ] `shark status advance C001` advances the change-card workflow
  - [ ] `shark status set B001 wont_fix` sets bug status directly
  - [ ] `shark search "login" --type=bug` filters search to bugs only
  - [ ] `shark search "dark mode" --type=change` filters to change-cards only
- **Related Journey**: Journey 3

#### Reporting and Analytics

**REQ-F-013**: Dashboard Integration
- **Description**: The `shark status` dashboard must include bug and change-card counts alongside existing feature/task progress.
- **User Story**: As a product owner, I want to see bug and change-card status on the project dashboard so that I have a complete view of all work items.
- **Acceptance Criteria**:
  - [ ] `shark status` (project dashboard) shows a "Bugs" section with counts by status
  - [ ] `shark status` shows a "Change Cards" section with counts by status
  - [ ] Bug severity breakdown is shown (critical: N, high: N, medium: N, low: N)
  - [ ] `shark status E07-F01` (feature status) shows linked bug count if any bugs are linked
- **Related Journey**: Journey 3, Step 2

**REQ-F-014**: Analytics Integration
- **Description**: The `shark analytics` command must include bug and change-card metrics.
- **User Story**: As a product owner, I want analytics on bug resolution time and change-card throughput so that I can measure team quality and responsiveness.
- **Acceptance Criteria**:
  - [ ] `shark analytics` includes: total bugs, bugs by status, bugs by severity, average resolution time
  - [ ] `shark analytics` includes: total change-cards, change-cards by status, approval rate, average time-to-completion
  - [ ] `shark analytics --type=bug` shows bug-specific analytics only
  - [ ] `shark analytics --type=change` shows change-card-specific analytics only
- **Related Journey**: Journey 3, Step 3

---

### Should Have Requirements

#### Bug Entity -- Enhanced Features

**REQ-F-015**: Bug Notes and Context
- **Description**: Bugs must support notes (same pattern as tasks) and context fields for structured metadata (severity, environment, component).
- **User Story**: As a QA engineer, I want to add notes during triage and verification so that the full investigation history is captured.
- **Acceptance Criteria**:
  - [ ] `shark bug note add B001 --type=comment "Reproduced on Safari 17.2"` adds a note
  - [ ] `shark bug notes B001` lists all notes on a bug
  - [ ] `shark bug context set B001 --field environment --value "Safari 17.2 on macOS 14.3"` sets context
  - [ ] `shark bug context get B001` returns all context fields
- **Related Journey**: Journey 1, Steps 2-5

**REQ-F-016**: Bug Markdown File Template
- **Description**: Bug markdown files must include structured sections for reproduction steps, expected behavior, actual behavior, environment, and stack traces.
- **User Story**: As a developer, I want bug reports to have consistent structure so that I can quickly understand and reproduce the issue.
- **Acceptance Criteria**:
  - [ ] Generated markdown file includes: Title, Severity, Status, Linked Entity, Reproduction Steps, Expected Behavior, Actual Behavior, Environment, Additional Context
  - [ ] The `shark view B001` command renders the bug markdown file

#### Change-Card Entity -- Enhanced Features

**REQ-F-017**: Change-Card Notes and Context
- **Description**: Change-cards must support notes and context fields, following the same pattern as bugs and tasks.
- **User Story**: As a product owner, I want to add approval notes to change-cards so that the decision rationale is documented.
- **Acceptance Criteria**:
  - [ ] `shark change note add C001 --type=decision "Approved: aligns with Q2 UX improvements"` adds a note
  - [ ] `shark change notes C001` lists all notes
  - [ ] `shark change context set C001 --field effort --value "small"` sets context
  - [ ] `shark change context get C001` returns all context fields

#### Cross-Entity Features

**REQ-F-018**: Bug List Filtering by Linked Entity
- **Description**: Bug list must support filtering by linked entity to show all bugs related to a specific epic, feature, or task.
- **User Story**: As a QA engineer, I want to see all bugs linked to a specific feature so that I can assess the quality of that feature.
- **Acceptance Criteria**:
  - [ ] `shark bug list --link=E07-F01` returns only bugs linked to feature E07-F01
  - [ ] `shark bug list --link=E07` returns bugs linked to any entity under epic E07
  - [ ] `shark change list --link=E07` returns change-cards linked to epic E07

---

### Could Have Requirements

**REQ-F-019**: Bug-to-Task Promotion
- **Description**: Support promoting a bug to a task when the fix requires formal feature-level tracking.
- **User Story**: As a product owner, I want to promote a complex bug to a task so that it receives proper planning and estimation.
- **Acceptance Criteria**:
  - [ ] `shark bug promote B001 --epic=E07 --feature=F01` creates a new task from the bug's details
  - [ ] The original bug is linked to the new task
  - [ ] The bug status is updated to reflect promotion

**REQ-F-020**: Change-Card-to-Feature Promotion
- **Description**: Support promoting a change-card to a feature when the scope grows beyond a single work item.
- **User Story**: As a product owner, I want to promote a change-card to a feature when it turns out to be larger than expected.
- **Acceptance Criteria**:
  - [ ] `shark change promote C001 --epic=E07` creates a new feature from the change-card's details
  - [ ] The original change-card is linked to the new feature
  - [ ] The change-card status is updated to reflect promotion

**REQ-F-021**: Bug Duplicate Detection Hint
- **Description**: When creating a bug, the system shows potential duplicates based on title similarity.
- **User Story**: As a developer, I want the system to warn me about potential duplicate bugs so that I do not create redundant reports.
- **Acceptance Criteria**:
  - [ ] `shark bug create "Login fails on Safari"` displays a warning if an open bug with similar title exists
  - [ ] The warning includes the existing bug key and status
  - [ ] User can proceed with creation or cancel

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Bug and Change-Card Creation Speed
- **Description**: Creating a bug or change-card must complete in under 500ms on local SQLite and under 2 seconds on Turso cloud.
- **Measurement**: Time from command invocation to CLI output, measured via shell timing
- **Target**: < 500ms local, < 2s cloud
- **Justification**: Bug reporting must not interrupt developer flow. Any noticeable delay discourages bug reporting.

**REQ-NF-002**: List Command Performance
- **Description**: `shark bug list` and `shark change list` must return results in under 1 second with up to 1000 bugs or change-cards in the database.
- **Measurement**: Time from command invocation to full output, measured via shell timing
- **Target**: < 1s for 1000 entities
- **Justification**: Triage and review workflows depend on fast list queries.

### Data Integrity

**REQ-NF-003**: Atomic Operations
- **Description**: Bug and change-card creation must be atomic -- if the database insert succeeds but the markdown file creation fails, the database record must be rolled back.
- **Measurement**: Verification via test cases that simulate file system failures
- **Target**: Zero orphaned records (no database entry without corresponding file)
- **Justification**: Consistency between database and file system is fundamental to Shark's architecture.

**REQ-NF-004**: Key Uniqueness
- **Description**: Bug keys (B###) and change-card keys (C###) must be globally unique and never reused, even after deletion.
- **Measurement**: Database constraint enforcement
- **Target**: Zero key collisions
- **Justification**: Keys are used as permanent identifiers in notes, links, and commit messages.

### Consistency

**REQ-NF-005**: CLI Pattern Consistency
- **Description**: All bug and change-card commands must follow the same CLI patterns established by the existing task, feature, and epic commands (flag naming, output formatting, error messages, JSON structure).
- **Measurement**: Manual review against existing CLI patterns documented in `.claude/rules/cli/commands.md`
- **Target**: Zero deviations from established patterns without documented justification
- **Justification**: Users (including AI agents) should not need to learn new patterns for the new entity types.

### Extensibility

**REQ-NF-006**: Workflow Profile Integration
- **Description**: Bug and change-card workflows must be configurable through `.sharkconfig.json` using the same workflow profile mechanism used for task, feature, and epic workflows.
- **Measurement**: Ability to customize status flows and metadata via config file
- **Target**: Full parity with existing workflow profile capabilities
- **Justification**: Teams must be able to customize workflows to match their processes.

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
