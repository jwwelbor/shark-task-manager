---
feature_key: E18-F01-database-schema-and-workflow-engine-extension
epic_key: E18
title: Database Schema and Workflow Engine Extension
description: Create bugs and change_cards database tables, extend the multi-level workflow engine with bug and change-card workflow levels, and update workflow profiles.
---

# Database Schema and Workflow Engine Extension

**Feature Key**: E18-F01
**Execution Order**: 1 (Foundation -- all other E18 features depend on this)

---

## Epic

- **Epic PRD**: [Bug and Change-Card Management System](../epic.md)
- **Requirements**: [Requirements](../requirements.md)
- **Scope Boundaries**: [Scope](../scope.md)
- **Tech Feasibility Review**: [E18-TECH-FEASIBILITY-REVIEW.md](../E18-TECH-FEASIBILITY-REVIEW.md)

---

## Goal

### Problem

Shark has no database tables for bugs or change-cards, and the multi-level workflow engine (`internal/config/workflow_multilevel.go`) only supports epic, feature, and task levels. The `MultiLevelWorkflow` struct has three fields (`Epic`, `Feature`, `Task`), and the `GetWorkflowForLevel()` switch statement handles only three cases. The `entity_notes` table has a CHECK constraint restricting `entity_type` to `('epic', 'feature', 'task')`, which blocks notes from being attached to bugs or change-cards. Without the data layer and workflow definitions, no bug or change-card entities can be created, stored, or advanced through their lifecycles, blocking all downstream features (F02-F07).

### Solution

Create the `bugs` and `change_cards` database tables following the proven schema patterns from the existing `ideas` table (standalone entity with auto-increment key, optional linking, timestamps). Extend the `MultiLevelWorkflow` struct with `Bug` and `Change` fields. Add `"bug"` and `"change"` cases to `GetWorkflowForLevel()`. Implement `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` functions. Update the workflow profile system to include bug and change-card defaults. Migrate the `entity_notes` table to remove the CHECK constraint on `entity_type`.

### Impact

- Unblocks all downstream E18 features (F02-F07) -- without this, no bug or change-card can exist in the system
- Bug and change-card workflows become configurable through `.sharkconfig.json`, matching existing entity workflow customizability
- The notes and context systems support bugs and change-cards from day one, avoiding a second migration later

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer building the bug management feature (F02), I want a `bugs` database table with the correct schema so that I can implement the BugRepository with standard CRUD operations.

**Acceptance Criteria**:
- [ ] A `bugs` table exists in the database with columns: id (INTEGER PRIMARY KEY AUTOINCREMENT), key (TEXT NOT NULL UNIQUE), title (TEXT NOT NULL), status (TEXT NOT NULL DEFAULT 'reported'), severity (TEXT NOT NULL DEFAULT 'medium' with CHECK constraint for critical/high/medium/low), slug (TEXT), linked_entity_type (TEXT), linked_entity_key (TEXT), context_data (TEXT), file_path (TEXT), created_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP), updated_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)
- [ ] Indexes exist on: key, status, severity, linked_entity_key, slug
- [ ] The table is created via the auto-migration system in `internal/db/db.go` so existing databases upgrade seamlessly
- [ ] Inserting a row with severity value outside ('critical', 'high', 'medium', 'low') is rejected by the database CHECK constraint

**Story 2**: As a developer building the change-card management feature (F03), I want a `change_cards` database table with the correct schema so that I can implement the ChangeCardRepository with standard CRUD operations.

**Acceptance Criteria**:
- [ ] A `change_cards` table exists in the database with columns: id (INTEGER PRIMARY KEY AUTOINCREMENT), key (TEXT NOT NULL UNIQUE), title (TEXT NOT NULL), status (TEXT NOT NULL DEFAULT 'proposed'), slug (TEXT), linked_entity_type (TEXT), linked_entity_key (TEXT), context_data (TEXT), file_path (TEXT), created_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP), updated_at (TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)
- [ ] Indexes exist on: key, status, linked_entity_key, slug
- [ ] The table is created via the auto-migration system

**Story 3**: As a developer building BugService, I want the workflow engine to support a `bug` level with a defined status flow so that `workflowSvc.ForLevel("bug")` returns a workflow with bug-specific statuses and transitions.

**Acceptance Criteria**:
- [ ] `MultiLevelWorkflow` struct has a `Bug *WorkflowConfig` field
- [ ] `GetWorkflowForLevel("bug")` returns the bug workflow config, falling back to `DefaultBugWorkflow()` if nil
- [ ] `DefaultBugWorkflow()` defines the status flow: reported -> triaged -> in_fix -> in_verification -> resolved
- [ ] Terminal statuses `wont_fix` and `duplicate` are reachable from any non-terminal status
- [ ] Status metadata (colors, phases, progress weights, responsibility) is defined for all bug statuses

**Story 4**: As a developer building ChangeCardService, I want the workflow engine to support a `change` level with a defined status flow so that `workflowSvc.ForLevel("change")` returns a workflow with change-card-specific statuses and transitions.

**Acceptance Criteria**:
- [ ] `MultiLevelWorkflow` struct has a `Change *WorkflowConfig` field
- [ ] `GetWorkflowForLevel("change")` returns the change-card workflow config, falling back to `DefaultChangeCardWorkflow()` if nil
- [ ] `DefaultChangeCardWorkflow()` defines the status flow: proposed -> approved -> in_progress -> completed
- [ ] Terminal status `declined` is reachable from `proposed` status
- [ ] Status metadata (colors, phases, progress weights, responsibility) is defined for all change-card statuses

**Story 5**: As a developer building note and context commands for bugs and change-cards, I want the `entity_notes` table to accept `'bug'` and `'change'` as valid `entity_type` values so that notes can be attached to these new entity types.

**Acceptance Criteria**:
- [ ] The CHECK constraint `entity_type IN ('epic', 'feature', 'task')` is removed from the `entity_notes` table
- [ ] Inserting a note with `entity_type = 'bug'` succeeds
- [ ] Inserting a note with `entity_type = 'change'` succeeds
- [ ] Existing notes with entity_type 'epic', 'feature', 'task' are preserved after migration
- [ ] The migration runs automatically via the auto-migration system with no manual steps required

**Story 6**: As a team lead configuring project workflows, I want `shark init update --workflow=advanced` to include bug and change-card workflow definitions so that the workflow profile system is complete for all entity types.

**Acceptance Criteria**:
- [ ] Running `shark init update --workflow=advanced` writes bug workflow and change-card workflow sections to `.sharkconfig.json`
- [ ] Running `shark init update --workflow=basic` writes simplified bug and change-card workflow sections
- [ ] Bug and change-card workflow sections follow the same JSON structure as existing epic/feature/task workflow sections
- [ ] Existing epic, feature, and task workflow configuration is preserved when updating

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when the `entity_notes` migration encounters a database with a large number of existing notes, I want the migration to complete without data loss so that production databases upgrade safely.

**Acceptance Criteria**:
- [ ] Migration uses CREATE TABLE ... SELECT pattern (or equivalent) to preserve all existing rows
- [ ] Migration is idempotent -- running it multiple times does not fail or duplicate data
- [ ] If migration fails partway, the original table remains intact (transaction-protected)

**Error Story 2**: As a developer, when a bug is inserted with an invalid severity value via direct SQL, I want the CHECK constraint to reject it so that data integrity is enforced at the database level.

**Acceptance Criteria**:
- [ ] `INSERT INTO bugs (..., severity, ...) VALUES (..., 'invalid', ...)` returns a CHECK constraint violation error
- [ ] Valid values ('critical', 'high', 'medium', 'low') are accepted
- [ ] The default value ('medium') is applied when severity is not specified

---

## Requirements

### Functional Requirements

**Category: Database Schema**

1. **F01-REQ-001**: Bugs Table Creation
   - **Description**: Create a `bugs` table with the schema defined in the tech feasibility review, including auto-increment primary key, unique key constraint, severity CHECK constraint, linking columns, and timestamp defaults.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-001, REQ-F-002, REQ-F-003, REQ-NF-004
   - **Acceptance Criteria**:
     - [ ] Table schema matches the specification in the tech feasibility review section 1.1
     - [ ] `UNIQUE` constraint on `key` column prevents duplicate bug keys
     - [ ] `CHECK` constraint on `severity` enforces valid values
     - [ ] `DEFAULT` values applied for `status` ('reported'), `severity` ('medium'), and timestamps

2. **F01-REQ-002**: Change-Cards Table Creation
   - **Description**: Create a `change_cards` table with auto-increment primary key, unique key constraint, linking columns, and timestamp defaults. No severity column (change-cards do not have severity).
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-007, REQ-F-008, REQ-NF-004
   - **Acceptance Criteria**:
     - [ ] Table schema matches specification in tech feasibility review section 1.2
     - [ ] `UNIQUE` constraint on `key` column prevents duplicate change-card keys
     - [ ] `DEFAULT` values applied for `status` ('proposed') and timestamps

3. **F01-REQ-003**: Database Indexes
   - **Description**: Create indexes on frequently queried columns for both tables to ensure list and filter operations meet performance targets (REQ-NF-002: under 1 second for 1000 entities).
   - **User Story**: Stories 1, 2
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-NF-002
   - **Acceptance Criteria**:
     - [ ] `idx_bugs_key` on `bugs(key)`
     - [ ] `idx_bugs_status` on `bugs(status)`
     - [ ] `idx_bugs_severity` on `bugs(severity)`
     - [ ] `idx_bugs_linked_entity_key` on `bugs(linked_entity_key)`
     - [ ] `idx_bugs_slug` on `bugs(slug)`
     - [ ] `idx_change_cards_key` on `change_cards(key)`
     - [ ] `idx_change_cards_status` on `change_cards(status)`
     - [ ] `idx_change_cards_linked_entity_key` on `change_cards(linked_entity_key)`
     - [ ] `idx_change_cards_slug` on `change_cards(slug)`

4. **F01-REQ-004**: Auto-Migration for New Tables
   - **Description**: All table creation and index creation statements must be added to the auto-migration system in `internal/db/db.go` using `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` patterns so existing databases upgrade without manual intervention.
   - **User Story**: Stories 1, 2
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-NF-003 (data integrity)
   - **Acceptance Criteria**:
     - [ ] Migration runs automatically on `InitDB()` call
     - [ ] Migration is idempotent -- safe to run on databases that already have the tables
     - [ ] Migration preserves all existing data in other tables

**Category: Workflow Engine**

5. **F01-REQ-005**: Workflow Level Constants
   - **Description**: Add `LevelBug` and `LevelChange` string constants to the workflow level system, following the pattern of existing level constants.
   - **User Story**: Stories 3, 4
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-004, REQ-F-009
   - **Acceptance Criteria**:
     - [ ] Constants `LevelBug = "bug"` and `LevelChange = "change"` are defined
     - [ ] Constants are usable by downstream services via `workflowSvc.ForLevel(workflow.LevelBug)`

6. **F01-REQ-006**: MultiLevelWorkflow Extension
   - **Description**: Add `Bug *WorkflowConfig` and `Change *WorkflowConfig` fields to the `MultiLevelWorkflow` struct. Add `"bug"` and `"change"` cases to `GetWorkflowForLevel()` with fallback to default workflows.
   - **User Story**: Stories 3, 4
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-004, REQ-F-009
   - **Acceptance Criteria**:
     - [ ] `MultiLevelWorkflow` struct has 5 fields: Epic, Feature, Task, Bug, Change
     - [ ] `GetWorkflowForLevel("bug")` returns `m.Bug` if non-nil, else `DefaultBugWorkflow()`
     - [ ] `GetWorkflowForLevel("change")` returns `m.Change` if non-nil, else `DefaultChangeCardWorkflow()`
     - [ ] Existing behavior for "epic", "feature", "task" is unchanged

7. **F01-REQ-007**: Default Bug Workflow
   - **Description**: Implement `DefaultBugWorkflow()` returning a `*WorkflowConfig` with the bug status flow, status metadata, and special status groups.
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-004
   - **Acceptance Criteria**:
     - [ ] Status flow defines: reported -> triaged -> in_fix -> in_verification -> resolved
     - [ ] Terminal statuses: resolved, wont_fix, duplicate
     - [ ] wont_fix is reachable from: reported, triaged, in_fix, in_verification
     - [ ] duplicate is reachable from: reported, triaged
     - [ ] Status metadata includes color, phase, progress_weight, and responsibility for each status
     - [ ] Progress weights: reported=0, triaged=20, in_fix=50, in_verification=80, resolved=100, wont_fix=100, duplicate=100

8. **F01-REQ-008**: Default Change-Card Workflow
   - **Description**: Implement `DefaultChangeCardWorkflow()` returning a `*WorkflowConfig` with the change-card status flow, status metadata, and special status groups.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-009
   - **Acceptance Criteria**:
     - [ ] Status flow defines: proposed -> approved -> in_progress -> completed
     - [ ] Terminal statuses: completed, declined
     - [ ] declined is reachable from: proposed
     - [ ] Status metadata includes color, phase, progress_weight, and responsibility for each status
     - [ ] Progress weights: proposed=0, approved=25, in_progress=50, completed=100, declined=100

**Category: entity_notes Migration**

9. **F01-REQ-009**: entity_notes CHECK Constraint Removal
   - **Description**: Migrate the `entity_notes` table to remove the CHECK constraint on `entity_type`. The recommended approach is table recreation: create new table without constraint, copy data, drop old table, rename new table. Application-layer validation in the service layer is sufficient.
   - **User Story**: Story 5
   - **Priority**: Must-Have
   - **Traces To**: Epic REQ-F-015, REQ-F-017
   - **Acceptance Criteria**:
     - [ ] After migration, `INSERT INTO entity_notes (..., entity_type, ...) VALUES (..., 'bug', ...)` succeeds
     - [ ] After migration, `INSERT INTO entity_notes (..., entity_type, ...) VALUES (..., 'change', ...)` succeeds
     - [ ] All existing notes with entity_type 'epic', 'feature', 'task' are preserved
     - [ ] All indexes on entity_notes are recreated after migration
     - [ ] Cascade delete triggers for entity_notes are preserved or recreated
     - [ ] Migration is idempotent -- running on a database already migrated does not fail

**Category: Workflow Profile Integration**

10. **F01-REQ-010**: Workflow Profile Updates
    - **Description**: Update the workflow profile system (`shark init update --workflow=advanced` and `--workflow=basic`) to include bug and change-card workflow definitions in the generated `.sharkconfig.json`.
    - **User Story**: Story 6
    - **Priority**: Must-Have
    - **Traces To**: Epic REQ-NF-006
    - **Acceptance Criteria**:
      - [ ] `--workflow=advanced` generates `bug_workflow` section with full bug status flow, metadata, and agent assignments
      - [ ] `--workflow=advanced` generates `change_workflow` section with full change-card status flow, metadata, and agent assignments
      - [ ] `--workflow=basic` generates simplified bug and change-card workflows
      - [ ] Existing epic, feature, and task workflow sections are unmodified by the profile update
      - [ ] Config deserialization correctly parses bug and change-card workflow sections

---

### Non-Functional Requirements

**Data Integrity**

1. **F01-REQ-NF-001**: Migration Safety
   - **Description**: The entity_notes migration must be transaction-protected. If any step fails, all changes are rolled back and the original table remains intact.
   - **Measurement**: Run migration against a database with 10,000+ notes; verify zero data loss
   - **Target**: Zero data loss, zero orphaned records
   - **Justification**: Production databases may have significant note data that cannot be recreated

**Backward Compatibility**

2. **F01-REQ-NF-002**: Existing Database Upgrade
   - **Description**: All schema changes must use auto-migration patterns (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`) that are safe to run on both fresh and existing databases.
   - **Measurement**: Run `InitDB()` on fresh database and on pre-existing database; verify both succeed
   - **Target**: Zero errors on upgrade of any existing Shark database
   - **Justification**: Users should not need to run manual migration scripts

**Configuration Compatibility**

3. **F01-REQ-NF-003**: Config File Forward Compatibility
   - **Description**: `.sharkconfig.json` files without bug/change-card workflow sections must still load correctly, with the system falling back to default workflows.
   - **Measurement**: Load a pre-E18 `.sharkconfig.json` and verify `GetWorkflowForLevel("bug")` returns `DefaultBugWorkflow()`
   - **Target**: Zero errors when loading old config files
   - **Justification**: Users upgrading from pre-E18 versions must not experience config parsing failures

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Fresh Database Initialization**
- **Given** a project with no existing database
- **When** `shark init --non-interactive` is run
- **Then** the `bugs`, `change_cards` tables are created with correct schemas
- **And** all indexes are created
- **And** the `entity_notes` table has no CHECK constraint on `entity_type`

**Scenario 2: Existing Database Upgrade**
- **Given** a project with an existing pre-E18 database containing tasks, features, epics, and notes
- **When** `InitDB()` is called (triggered by any shark command)
- **Then** the `bugs` and `change_cards` tables are added
- **And** the `entity_notes` CHECK constraint is removed
- **And** all existing data in all tables is preserved
- **And** all existing notes retain their entity_type, entity_id, and content

**Scenario 3: Bug Workflow Resolution**
- **Given** a default bug workflow configuration (no custom config)
- **When** `GetWorkflowForLevel("bug")` is called
- **Then** the returned workflow has statuses: reported, triaged, in_fix, in_verification, resolved, wont_fix, duplicate
- **And** the default initial status is `reported`
- **And** terminal statuses are: resolved, wont_fix, duplicate

**Scenario 4: Change-Card Workflow Resolution**
- **Given** a default change-card workflow configuration (no custom config)
- **When** `GetWorkflowForLevel("change")` is called
- **Then** the returned workflow has statuses: proposed, approved, in_progress, completed, declined
- **And** the default initial status is `proposed`
- **And** terminal statuses are: completed, declined

**Scenario 5: Workflow Profile Application**
- **Given** a project with an existing `.sharkconfig.json`
- **When** `shark init update --workflow=advanced` is run
- **Then** the config file contains `bug_workflow` and `change_workflow` sections
- **And** existing epic, feature, and task workflow sections are preserved
- **And** the config file is valid JSON that parses without error

**Scenario 6: Old Config File Compatibility**
- **Given** a `.sharkconfig.json` from a pre-E18 installation (no bug/change workflow sections)
- **When** any shark command runs and loads the config
- **Then** `GetWorkflowForLevel("bug")` returns `DefaultBugWorkflow()` without error
- **And** `GetWorkflowForLevel("change")` returns `DefaultChangeCardWorkflow()` without error

---

## Scope

### In Scope

1. **`bugs` table creation** -- Schema with all columns, constraints, defaults, and indexes as specified in F01-REQ-001 and F01-REQ-003
2. **`change_cards` table creation** -- Schema with all columns, constraints, defaults, and indexes as specified in F01-REQ-002 and F01-REQ-003
3. **Workflow engine extension** -- `LevelBug` and `LevelChange` constants, `MultiLevelWorkflow` struct fields, `GetWorkflowForLevel()` cases as specified in F01-REQ-005 and F01-REQ-006
4. **Default workflow definitions** -- `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` as specified in F01-REQ-007 and F01-REQ-008
5. **entity_notes migration** -- CHECK constraint removal as specified in F01-REQ-009
6. **Workflow profile integration** -- Bug and change-card sections in profile generation as specified in F01-REQ-010
7. **Auto-migration integration** -- All changes added to `InitDB()` migration system as specified in F01-REQ-004

### Out of Scope

1. **Bug and change-card Go model structs** (`models/bug.go`, `models/change_card.go`)
   - **Why**: Model structs are part of F02 (Bug Entity Core) and F03 (Change-Card Entity Core)
   - **Future**: Implemented in the immediately following features

2. **Repository implementations** (`repository/bug_repository.go`, `repository/change_card_repository.go`)
   - **Why**: Repositories depend on model structs and are part of F02/F03
   - **Future**: Implemented in F02/F03

3. **Service layer** (`services/bug_service.go`, `services/change_card_service.go`)
   - **Why**: Services depend on repositories and models, and are part of F02/F03
   - **Future**: Implemented in F02/F03

4. **CLI commands** (`cli/commands/bug.go`, `cli/commands/change.go`)
   - **Why**: CLI commands are part of F04/F05
   - **Future**: Implemented in F04/F05

5. **Key auto-detection dispatch** (B###/C### patterns in `keys/service.go`, `helpers.go`)
   - **Why**: Key detection is part of F06 (Unified CLI Integration)
   - **Future**: Implemented in F06

6. **Dashboard and analytics** (bug/change-card sections in dashboard output)
   - **Why**: Dashboard is part of F07
   - **Future**: Implemented in F07

7. **Cascade delete triggers for bugs/change-cards** -- triggers to delete notes when a bug/change-card is deleted
   - **Why**: Cascade deletes depend on the repository layer (F02/F03) which implements the delete operation. The triggers should be added when the delete functionality is built.
   - **Future**: Added as part of F02/F03 implementation

---

## Dependencies & Integrations

### Dependencies

- **E16 (Multi-Level Workflow System)**: The `ForLevel()` infrastructure and `WorkflowConfig` struct must exist. **Confirmed**: Inspected `internal/config/workflow_multilevel.go` -- the struct and method are implemented and working for epic/feature/task levels.
- **E11 (Configurable Workflow)**: The `.sharkconfig.json` workflow configuration pattern must be established. **Confirmed**: The pattern exists and is used by `shark init update --workflow=advanced`.
- **Auto-Migration System**: The `InitDB()` auto-migration infrastructure in `internal/db/db.go` must support adding new tables and running idempotent migrations. **Confirmed**: The existing system handles `CREATE TABLE IF NOT EXISTS` and has precedent for column-addition migrations.

### Integration Points

- **`internal/db/db.go`**: Add `bugs` and `change_cards` table creation SQL, index creation SQL, and `entity_notes` migration logic to the auto-migration sequence
- **`internal/config/workflow_multilevel.go`**: Add `Bug` and `Change` fields to struct, extend `GetWorkflowForLevel()` switch
- **`internal/config/` (default workflows)**: Add `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` functions, following `DefaultEpicWorkflow()` pattern
- **`internal/init/` (workflow profiles)**: Update profile generation to include bug and change-card workflow sections
- **`.sharkconfig.json`**: Add `bug_workflow` and `change_workflow` JSON sections

### No E18 Internal Dependencies

This is the foundation feature. F02-F07 all depend on F01. F01 depends on no other E18 feature.

---

## Open Questions & Assumptions

No open questions -- all decisions are resolved.

**Resolved Decisions:**

1. **entity_notes migration approach**: Remove the CHECK constraint entirely rather than updating it to include 'bug' and 'change'. This eliminates future migration needs when adding entity types. Application-layer validation is sufficient. (Per Tech Feasibility Review, Recommendation 3.)

2. **Severity as column vs. context field**: Severity is a dedicated database column on the `bugs` table (not stored in `context_data` JSON). This enables indexed queries for `shark bug list --severity=critical`. (Per Tech Feasibility Review, section 1.1.)

3. **Workflow level naming**: Bug level is `"bug"`, change-card level is `"change"` (not `"change_card"`). This matches the key prefix pattern (B for bug, C for change) and keeps level names short.

4. **Default status names**: Bug uses `reported` (not `open` or `new`). Change-card uses `proposed` (not `draft` or `submitted`). These names match the workflows defined in the epic PRD.

5. **Cascade delete triggers**: Not included in F01. Cascade delete triggers for bugs and change-cards (to clean up entity_notes when a bug/change-card is deleted) will be added in F02/F03 when the delete operations are implemented.

---

## Requirements Traceability

| Epic Requirement | F01 Coverage | Remaining Coverage |
|-----------------|-------------|-------------------|
| REQ-F-001 (Bug Entity Creation) | Database table and key generation infrastructure | F02 (model, repo, service), F04 (CLI) |
| REQ-F-002 (Bug Severity Tracking) | Severity column with CHECK constraint and index | F02 (filtering logic), F04 (--severity flag) |
| REQ-F-003 (Bug Entity Linking) | linked_entity_type and linked_entity_key columns | F02 (link validation), F04 (--link flag) |
| REQ-F-004 (Bug Status Workflow) | Bug workflow level, default status flow, metadata | F02 (service transitions), F06 (unified dispatch) |
| REQ-F-007 (Change-Card Entity Creation) | Database table and key generation infrastructure | F03 (model, repo, service), F05 (CLI) |
| REQ-F-008 (Change-Card Entity Linking) | linked_entity_type and linked_entity_key columns | F03 (link validation), F05 (--link flag) |
| REQ-F-009 (Change-Card Status Workflow) | Change-card workflow level, default status flow, metadata | F03 (service transitions), F06 (unified dispatch) |
| REQ-F-015 (Bug Notes and Context) | entity_notes CHECK constraint migration | F02 (service), F04 (CLI) |
| REQ-F-017 (Change-Card Notes and Context) | entity_notes CHECK constraint migration | F03 (service), F05 (CLI) |
| REQ-NF-004 (Key Uniqueness) | UNIQUE constraints on key columns | Fully covered by F01 |
| REQ-NF-006 (Workflow Profile Integration) | Workflow profile system updates | Fully covered by F01 |

---

## Sprint Sizing

**Estimate**: 1 sprint (S-M complexity)

| Component | Size | Rationale |
|-----------|------|-----------|
| bugs table creation + indexes | S | Direct pattern reuse from ideas table |
| change_cards table creation + indexes | S | Same pattern as bugs table, fewer columns |
| Workflow engine extension (struct + switch) | S | 2 new fields, 2 new switch cases |
| DefaultBugWorkflow() | S | Follow DefaultEpicWorkflow() pattern |
| DefaultChangeCardWorkflow() | S | Simpler than bug workflow |
| entity_notes migration | M | Table recreation required; must preserve data and triggers |
| Workflow profile integration | S | Add sections to existing profile generation |

---

*Last Updated*: 2026-03-03
