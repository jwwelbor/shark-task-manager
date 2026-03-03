---
feature_key: E18-F02-bug-entity-core-model-repository-service
epic_key: E18
title: Bug Entity Core (Model, Repository, Service)
description: Implement the Bug model, BugRepository, and BugService following E15 service layer patterns, including triage logic, link validation, and workflow integration.
---

# Bug Entity Core (Model, Repository, Service)

**Feature Key**: E18-F02
**Execution Order**: 2 (depends on F01)

---

## Epic

- **Epic PRD**: [Bug and Change-Card Management System](../epic.md)
- **Requirements**: [Epic Requirements](../requirements.md)
- **Scope**: [Epic Scope Boundaries](../scope.md)
- **Tech Feasibility**: [Tech Feasibility Review](../E18-TECH-FEASIBILITY-REVIEW.md)

---

## Goal

### Problem

With the database schema and workflow engine in place (F01), there is no application-layer code to create, retrieve, update, or manage bug entities. The Bug model struct, repository for database access, and service for business logic must exist before CLI commands (F04), unified CLI integration (F06), or dashboard analytics (F07) can function. Without the service layer, there is no place to enforce link validation (verifying linked epics/features/tasks exist), no triage operation (setting severity + advancing status atomically), and no workflow-integrated status transitions for bugs.

### Solution

Implement the full Bug entity stack as three layers following E15 service layer patterns:

1. **Model** (`internal/models/bug.go`): Bug struct with typed fields, severity constants, key validation (`B###`), and structural `Validate()` method.
2. **Repository** (`internal/repository/bug_repository.go`): Pure CRUD operations against the `bugs` table, plus filtering queries by status, severity, and linked entity. Repository interface defined in the service package for testability.
3. **Service** (`internal/services/bug_service.go`): All business logic -- bug creation with auto-key generation and link validation, triage operation (set severity + assign + advance status atomically), workflow-integrated status transitions via `workflowSvc.ForLevel(LevelBug)`, and markdown file generation via the `fileops` package.
4. **Supporting files**: Service DTOs (`bug_dto.go`), markdown template (`templates/bug.md.tmpl`), service accessor (`GetBugService()` in `services_global.go`).

### Impact

- Enables the full bug lifecycle at the service layer, ready for CLI (F04) and HTTP API consumption
- Triage operation encapsulated as a single atomic service call -- avoids multi-step CLI commands
- Link validation prevents orphaned references to non-existent epics, features, or tasks
- All business logic centralized in the service layer, not scattered across future CLI commands

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want to create a bug entity programmatically (via service call) so that CLI and HTTP entry points can create bugs without duplicating creation logic.

**Acceptance Criteria**:
- [ ] `BugService.CreateBug(ctx, CreateBugInput{Title: "Login fails"})` returns a persisted `*models.Bug` with an auto-generated key (B001, B002, etc.)
- [ ] The returned bug has status `reported`, severity `medium` (default), and a valid `CreatedAt` timestamp
- [ ] A markdown file is created at `docs/bugs/B###.md` using the bug template with frontmatter (key, title, status, severity) and body sections
- [ ] If file creation fails after database insert, the database record is rolled back (atomic operation)
- [ ] The bug key is globally unique -- creating 3 bugs produces B001, B002, B003 regardless of deletions

**Story 2**: As a developer, I want to create a bug with a severity and a link to an existing entity so that the bug has full context from the moment of creation.

**Acceptance Criteria**:
- [ ] `BugService.CreateBug(ctx, CreateBugInput{Title: "Crash", Severity: "critical", LinkedEntityKey: "E07-F01"})` creates a bug with severity `critical` linked to feature E07-F01
- [ ] Link validation: if `LinkedEntityKey` is provided and the entity does not exist, `CreateBug` returns an error containing the invalid key (e.g., `"linked entity not found: E07-F99"`)
- [ ] Link validation detects entity type from key format: `E##` = epic, `E##-F##` = feature, `E##-F##-###` = task
- [ ] Severity validation: if severity is not one of `critical`, `high`, `medium`, `low`, `CreateBug` returns a validation error
- [ ] The linked entity key and type are stored in the bug record and appear in `BugService.GetBug` output

**Story 3**: As a QA engineer, I want to triage a bug with a single service call so that severity, assignment, and status advance happen atomically.

**Acceptance Criteria**:
- [ ] `BugService.TriageBug(ctx, TriageBugInput{Key: "B001", Severity: "high", AssignedTo: "developer"})` sets severity to `high`, sets assigned_to (via context field), and advances status from `reported` to `triaged`
- [ ] Triage is only valid when the bug status is `reported`; calling triage on a bug with status `triaged` returns an error: `"cannot triage bug B001: current status is 'triaged', must be 'reported'"`
- [ ] If the severity or assigned_to update succeeds but the status advance fails (workflow error), the entire operation is rolled back
- [ ] The triage operation records a status history entry

**Story 4**: As a developer, I want to advance a bug through its workflow so that bugs progress from reported to resolved with proper validation.

**Acceptance Criteria**:
- [ ] `BugService.AdvanceBugStatus(ctx, "B001")` advances the bug to the next status defined by the bug workflow (reported -> triaged -> in_fix -> in_verification -> resolved)
- [ ] `BugService.SetBugStatus(ctx, "B001", "wont_fix")` sets the bug to the `wont_fix` terminal status
- [ ] `BugService.SetBugStatus(ctx, "B001", "duplicate")` sets the bug to the `duplicate` terminal status
- [ ] Invalid transitions return a workflow error (e.g., advancing from `resolved` returns `"no valid next status for 'resolved'"`)
- [ ] Each status change records a history entry via the workflow service

**Story 5**: As a developer, I want to list and filter bugs so that I can find bugs by status, severity, or linked entity.

**Acceptance Criteria**:
- [ ] `BugService.ListBugs(ctx, BugFilters{})` returns all bugs sorted by creation date descending
- [ ] `BugService.ListBugs(ctx, BugFilters{Status: "reported"})` returns only bugs with status `reported`
- [ ] `BugService.ListBugs(ctx, BugFilters{Severity: "critical"})` returns only critical bugs
- [ ] `BugService.ListBugs(ctx, BugFilters{LinkedEntityKey: "E07-F01"})` returns only bugs linked to feature E07-F01
- [ ] Multiple filters combine with AND logic: `BugFilters{Status: "reported", Severity: "critical"}` returns reported critical bugs only
- [ ] An empty result set returns an empty slice (not nil), no error

**Story 6**: As a developer, I want to retrieve, update, and delete individual bugs so that I have full CRUD capabilities.

**Acceptance Criteria**:
- [ ] `BugService.GetBug(ctx, "B001")` returns the full bug record including all fields
- [ ] `BugService.GetBug(ctx, "B999")` returns a `NotFoundError` with entity "bug" and key "B999"
- [ ] `BugService.UpdateBug(ctx, "B001", BugUpdates{Title: ptr("New title")})` updates only the specified fields
- [ ] `BugService.UpdateBug` does not allow changing the bug key
- [ ] `BugService.DeleteBug(ctx, "B001")` removes the bug from the database
- [ ] Deleting a non-existent bug returns a `NotFoundError`

### Should-Have Stories

**Story 7**: As a developer, I want the bug entity to support notes and context fields so that investigation history and metadata are captured.

**Acceptance Criteria**:
- [ ] The `BugService` supports entity type `"bug"` when calling `NoteService.AddNote` and `ContextService.SetField`
- [ ] `shark bug context set B001 --field environment --value "Safari 17.2 on macOS 14.3"` succeeds (via F04 CLI, but service support is in F02)
- [ ] The service layer registers "bug" as a valid entity type for notes and context operations

**Story 8**: As a developer, I want the bug markdown template to include structured sections so that bug reports are consistent.

**Acceptance Criteria**:
- [ ] The generated markdown file contains sections: Reproduction Steps, Expected Behavior, Actual Behavior, Environment, Additional Context
- [ ] Frontmatter includes: key, title, status, severity, linked_entity (if present)
- [ ] The template is registered in the templates package and loadable via the standard template system

### Edge Case & Error Stories

**Error Story 1**: As a developer, when I create a bug with an invalid severity value, I want a clear validation error so that I can correct my input.

**Acceptance Criteria**:
- [ ] `CreateBug` with severity `"urgent"` returns error: `"invalid severity 'urgent': must be one of critical, high, medium, low"`
- [ ] The error is returned before any database or file operations occur

**Error Story 2**: As a developer, when I create a bug and the file system is unavailable, I want the database record to be rolled back so that I do not have orphaned records.

**Acceptance Criteria**:
- [ ] If `fileops.WriteEntityFile` fails, the database transaction is rolled back
- [ ] No bug record exists in the database after a failed creation
- [ ] The error message includes both the file operation failure reason and that the creation was rolled back

**Error Story 3**: As a developer, when I advance a bug in a terminal status, I want a clear error so that I know the bug lifecycle is complete.

**Acceptance Criteria**:
- [ ] Advancing a bug with status `resolved` returns: `"cannot advance bug B001: status 'resolved' is terminal"`
- [ ] Advancing a bug with status `wont_fix` returns the same pattern
- [ ] Advancing a bug with status `duplicate` returns the same pattern

---

## Requirements

### Functional Requirements

**Category: Bug Model**

1. **F02-REQ-001**: Bug Model Struct
   - **Description**: The Bug model struct must contain all fields needed to represent a bug entity: ID, Key, Title, Status, Severity, Slug, LinkedEntityType, LinkedEntityKey, ContextData (JSON), FilePath, CreatedAt, UpdatedAt.
   - **Traces to**: REQ-F-001, REQ-F-002, REQ-F-003
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] All fields have proper Go types and JSON/db struct tags
     - [ ] BugStatus is a typed string (not raw string) following the TaskStatus pattern
     - [ ] BugSeverity has constants: BugSeverityCritical, BugSeverityHigh, BugSeverityMedium, BugSeverityLow
     - [ ] LinkedEntityType and LinkedEntityKey are pointer types (nullable)

2. **F02-REQ-002**: Bug Model Validation
   - **Description**: The Bug model must have a `Validate()` method that performs structural validation: non-empty title, non-empty key matching `B###` format, non-empty status, valid severity value.
   - **Traces to**: REQ-F-001, REQ-NF-005
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `Validate()` returns error for empty title
     - [ ] `Validate()` returns error for key not matching `^B\d{3}$` pattern
     - [ ] `Validate()` returns error for empty status
     - [ ] `Validate()` returns error for invalid severity (not in the 4 valid values)
     - [ ] `Validate()` does NOT check workflow status validity (that is the service layer's job)

3. **F02-REQ-003**: Bug Key Validation Function
   - **Description**: A standalone `ValidateBugKey(key string) error` function for validating bug key format.
   - **Traces to**: REQ-F-001, REQ-NF-004
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Accepts: `B001`, `B042`, `B999`
     - [ ] Rejects: `B0001` (4 digits), `B01` (2 digits), `b001` (lowercase -- validation is case-sensitive on the stored key), `bug001`, empty string
     - [ ] Error messages include the invalid key and expected format

**Category: Bug Repository**

4. **F02-REQ-004**: Bug Repository Interface
   - **Description**: Define a `BugRepository` interface in the service package with all methods the BugService needs, following the consumer-side interface pattern.
   - **Traces to**: REQ-F-006, REQ-NF-005
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Interface includes: Create, GetByKey, GetByID, List, Update, Delete, UpdateStatus
     - [ ] Interface includes query methods: CountByStatus, CountBySeverity, ListBySeverity, ListByLinkedEntity
     - [ ] All methods accept `context.Context` as first parameter
     - [ ] The concrete `repository.BugRepository` implements this interface

5. **F02-REQ-005**: Bug Repository CRUD Implementation
   - **Description**: Implement the `BugRepository` struct with pure CRUD operations against the `bugs` table using parameterized queries.
   - **Traces to**: REQ-F-006, REQ-NF-001, REQ-NF-002
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `Create` inserts a bug record and sets the auto-generated ID on the model
     - [ ] `GetByKey` returns the bug or `NotFoundError` if not found
     - [ ] `GetByKey` is case-insensitive on the key lookup (matching existing entity behavior)
     - [ ] `List` returns all bugs, optionally filtered by provided parameters
     - [ ] `Update` updates title, severity, and other mutable fields (not key)
     - [ ] `Delete` removes the bug record; returns `NotFoundError` if not found
     - [ ] `UpdateStatus` updates only the status field and updated_at timestamp

6. **F02-REQ-006**: Bug Repository Filtering Queries
   - **Description**: Implement filtered list queries for bugs by status, severity, and linked entity.
   - **Traces to**: REQ-F-002, REQ-F-018, REQ-NF-002
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `ListBySeverity(ctx, "critical")` returns bugs with severity `critical`
     - [ ] `ListByLinkedEntity(ctx, "E07-F01")` returns bugs linked to feature E07-F01
     - [ ] `CountByStatus(ctx)` returns a map of status -> count
     - [ ] `CountBySeverity(ctx)` returns a map of severity -> count
     - [ ] All queries use indexed columns for performance (status, severity, linked_entity_key indexes from F01)

7. **F02-REQ-007**: Bug Key Auto-Generation
   - **Description**: The repository must support auto-generating the next bug key by finding the maximum existing B### key and incrementing.
   - **Traces to**: REQ-F-001, REQ-NF-004
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `GetNextKey(ctx)` returns `"B001"` when no bugs exist
     - [ ] `GetNextKey(ctx)` returns `"B003"` when B001 and B002 exist
     - [ ] Key generation is safe under concurrent access (uses database-level max query, not in-memory counter)
     - [ ] Deleted keys are not reused (B002 deleted, next key is still B003 not B002)

**Category: Bug Service**

8. **F02-REQ-008**: Bug Service Constructor
   - **Description**: `NewBugService` constructor with explicit dependency injection following the established pattern.
   - **Traces to**: REQ-NF-005
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Constructor accepts: BugRepository (required), *workflow.Service (required), link validator dependencies (EpicRepository, FeatureRepository, TaskRepository -- for link validation)
     - [ ] Workflow service is scoped to bug level: `workflowSvc.ForLevel(workflow.LevelBug)`
     - [ ] Optional dependencies (note repo) degrade gracefully when nil

9. **F02-REQ-009**: CreateBug Service Method
   - **Description**: Implement `CreateBug(ctx, CreateBugInput) (*models.Bug, error)` with key generation, link validation, model validation, file creation, and database persistence.
   - **Traces to**: REQ-F-001, REQ-F-002, REQ-F-003, REQ-NF-003
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Auto-generates the next bug key via repository
     - [ ] Auto-generates slug from title
     - [ ] Sets default severity to `medium` if not provided
     - [ ] Sets default status to `reported` (from workflow service default status)
     - [ ] Validates linked entity exists (if provided) by detecting entity type from key format and calling the appropriate repository
     - [ ] Creates markdown file via `fileops.WriteEntityFile` with `UseAtomicWrite: true`
     - [ ] Persists bug record in database
     - [ ] If file creation fails, database insert is rolled back
     - [ ] Returns the created bug with all fields populated (including ID and timestamps)

10. **F02-REQ-010**: TriageBug Service Method
    - **Description**: Implement `TriageBug(ctx, TriageBugInput) (*models.Bug, error)` that sets severity, sets assigned_to context, and advances status from `reported` to `triaged` atomically.
    - **Traces to**: REQ-F-005
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Validates bug exists and is in `reported` status
      - [ ] Updates severity if provided in input
      - [ ] Sets `assigned_to` as a context field if provided
      - [ ] Advances status from `reported` to `triaged` via workflow service
      - [ ] All three operations (severity, context, status) are atomic -- if any fails, all are rolled back
      - [ ] Returns the updated bug

11. **F02-REQ-011**: Status Transition Methods
    - **Description**: Implement `AdvanceBugStatus(ctx, key) (*models.Bug, error)` and `SetBugStatus(ctx, key, status) (*models.Bug, error)` using the workflow service.
    - **Traces to**: REQ-F-004
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] `AdvanceBugStatus` gets the next status from the workflow service and transitions to it
      - [ ] `SetBugStatus` validates the transition via the workflow service before applying
      - [ ] Both methods record status history
      - [ ] Both return the updated bug with new status
      - [ ] Invalid transitions return a descriptive error

12. **F02-REQ-012**: ListBugs Service Method with Filters
    - **Description**: Implement `ListBugs(ctx, BugFilters) ([]*models.Bug, error)` with filter support.
    - **Traces to**: REQ-F-006, REQ-F-018
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Supports filtering by Status, Severity, LinkedEntityKey
      - [ ] Multiple filters combine with AND logic
      - [ ] Returns empty slice (not nil) when no bugs match
      - [ ] Default sort: creation date descending

**Category: Service DTOs**

13. **F02-REQ-013**: Bug Service DTOs
    - **Description**: Define input DTOs for service methods: `CreateBugInput`, `BugFilters`, `TriageBugInput`, `BugUpdates`.
    - **Traces to**: REQ-NF-005
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] `CreateBugInput` has: Title (required), Severity (optional, default medium), LinkedEntityKey (optional), Description (optional)
      - [ ] `BugFilters` has: Status, Severity, LinkedEntityKey, ShowAll (bool)
      - [ ] `TriageBugInput` has: Key (required), Severity (optional), AssignedTo (optional)
      - [ ] `BugUpdates` has: Title, Severity -- all pointer types for partial updates
      - [ ] All DTOs have JSON tags for HTTP API compatibility

**Category: Markdown Template**

14. **F02-REQ-014**: Bug Markdown Template
    - **Description**: Create a bug report markdown template with structured sections.
    - **Traces to**: REQ-F-016
    - **Priority**: Should-Have
    - **Acceptance Criteria**:
      - [ ] Template file at `templates/bug.md.tmpl`
      - [ ] Frontmatter includes: key, title, status, severity, linked_entity (if present)
      - [ ] Body sections: Reproduction Steps, Expected Behavior, Actual Behavior, Environment, Additional Context
      - [ ] Template renders correctly via the templates package
      - [ ] Each section has placeholder instructional text (e.g., "Describe the steps to reproduce the issue...")

**Category: Service Accessor**

15. **F02-REQ-015**: GetBugService Global Accessor
    - **Description**: Add `GetBugService()` to `internal/cli/services_global.go` following the existing `GetTaskService()` pattern.
    - **Traces to**: REQ-NF-005
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Creates BugRepository, wires workflow service scoped to LevelBug, injects link validation dependencies
      - [ ] Creates new service instance per call (matching existing pattern)
      - [ ] Panics on DB failure (matching existing CLI entry point pattern)

### Non-Functional Requirements

**Performance**

1. **F02-NFR-001**: Bug Creation Speed
   - **Description**: `BugService.CreateBug` must complete in under 500ms on local SQLite including key generation, link validation, file creation, and database insert.
   - **Traces to**: REQ-NF-001
   - **Measurement**: Integration test timing
   - **Target**: < 500ms local

2. **F02-NFR-002**: Bug List Performance
   - **Description**: `BugService.ListBugs` must return results in under 1 second with up to 1000 bugs in the database.
   - **Traces to**: REQ-NF-002
   - **Measurement**: Repository test with seeded data
   - **Target**: < 1s for 1000 records

**Data Integrity**

3. **F02-NFR-003**: Atomic Bug Creation
   - **Description**: If `fileops.WriteEntityFile` fails during bug creation, the database insert must be rolled back. No orphaned records.
   - **Traces to**: REQ-NF-003
   - **Measurement**: Test case simulating file system failure
   - **Target**: Zero orphaned records

**Consistency**

4. **F02-NFR-004**: Pattern Consistency
   - **Description**: All Bug model, repository, and service code must follow the patterns established by the Idea entity (for standalone entities) and Task entity (for service layer architecture).
   - **Traces to**: REQ-NF-005
   - **Measurement**: Code review against existing patterns
   - **Target**: Zero pattern deviations without documented justification

**Testability**

5. **F02-NFR-005**: Test Coverage
   - **Description**: Model validation tests, repository tests (real DB with cleanup), and service tests (mocked repositories) must all be implemented.
   - **Measurement**: `go test` pass rate and coverage report
   - **Target**: Model tests: 100% validation paths. Repository tests: all CRUD + filtering. Service tests: all business logic paths including error cases.

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Bug Creation Happy Path**
- **Given** the bugs table exists (F01 complete) and no bugs have been created
- **When** `BugService.CreateBug(ctx, CreateBugInput{Title: "Button misaligned on mobile"})` is called
- **Then** a bug with key `B001`, status `reported`, severity `medium` is returned
- **And** a markdown file exists at `docs/bugs/B001.md` with the bug template content
- **And** `BugService.GetBug(ctx, "B001")` returns the same bug

**Scenario 2: Bug Creation with Link Validation**
- **Given** feature E07-F01 exists in the database
- **When** `BugService.CreateBug(ctx, CreateBugInput{Title: "Crash", LinkedEntityKey: "E07-F01"})` is called
- **Then** the bug is created with `LinkedEntityType: "feature"` and `LinkedEntityKey: "E07-F01"`

**Scenario 3: Bug Creation with Invalid Link**
- **Given** feature E07-F99 does not exist
- **When** `BugService.CreateBug(ctx, CreateBugInput{Title: "Crash", LinkedEntityKey: "E07-F99"})` is called
- **Then** an error is returned containing `"linked entity not found: E07-F99"`
- **And** no bug record exists in the database

**Scenario 4: Triage Operation**
- **Given** bug B001 exists with status `reported` and severity `medium`
- **When** `BugService.TriageBug(ctx, TriageBugInput{Key: "B001", Severity: "high", AssignedTo: "alice"})` is called
- **Then** the bug severity is updated to `high`
- **And** the bug status is advanced to `triaged`
- **And** `assigned_to` context field is set to `"alice"`

**Scenario 5: Triage on Non-Reported Bug (Error)**
- **Given** bug B001 exists with status `triaged`
- **When** `BugService.TriageBug(ctx, TriageBugInput{Key: "B001", Severity: "critical"})` is called
- **Then** an error is returned: `"cannot triage bug B001: current status is 'triaged', must be 'reported'"`
- **And** no fields are modified

**Scenario 6: Full Workflow Traversal**
- **Given** bug B001 exists with status `reported`
- **When** status is advanced through: reported -> triaged -> in_fix -> in_verification -> resolved
- **Then** each transition succeeds
- **And** attempting to advance past `resolved` returns a terminal status error

**Scenario 7: Terminal Status (wont_fix)**
- **Given** bug B001 exists with status `triaged`
- **When** `BugService.SetBugStatus(ctx, "B001", "wont_fix")` is called
- **Then** the bug status is set to `wont_fix`
- **And** attempting to advance returns a terminal status error

**Scenario 8: Filtered Bug List**
- **Given** 5 bugs exist: 2 critical+reported, 1 high+triaged, 1 medium+reported, 1 low+resolved
- **When** `BugService.ListBugs(ctx, BugFilters{Severity: "critical", Status: "reported"})` is called
- **Then** exactly 2 bugs are returned (the critical reported ones)

---

## Out of Scope

### Explicitly Excluded

1. **CLI Commands for Bugs**
   - **Why**: CLI commands are the responsibility of E18-F04. This feature delivers the service layer that F04 will call.
   - **Future**: F04 creates `shark bug create/get/list/update/delete/triage` commands as thin wrappers around BugService.

2. **Unified CLI Key Auto-Detection for B### Keys**
   - **Why**: Key detection and dispatch routing are the responsibility of E18-F06.
   - **Future**: F06 adds B### to the `DetectEntityType()` function and updates all dispatch switch statements.

3. **Dashboard and Analytics for Bugs**
   - **Why**: Dashboard and analytics integration is the responsibility of E18-F07.
   - **Future**: F07 adds bug count sections to the `shark status` dashboard and bug metrics to `shark analytics`.

4. **Bug-to-Task Promotion**
   - **Why**: Promotion is a Could Have requirement (REQ-F-019) deferred to a follow-on epic per epic scope decisions.
   - **Workaround**: Manually create a task and add a note referencing the bug key.

5. **Bug Duplicate Detection**
   - **Why**: Duplicate detection is a Could Have requirement (REQ-F-021) deferred to a follow-on epic.
   - **Workaround**: Use `shark bug list` (via F04) to manually search for similar bugs before creating.

6. **Bug Dependencies (Bug A blocks Bug B)**
   - **Why**: Explicitly excluded at epic level (see scope.md, Edge Cases section).
   - **Workaround**: Add notes referencing related bugs.

---

## Dependencies & Integrations

### Dependencies

- **E18-F01 (Database Schema and Workflow Engine Extension)**: Requires the `bugs` table, bug workflow level (`LevelBug`), default bug workflow definition, and entity_notes migration. F01 must be complete before F02 development begins.
- **E15 (Service Layer Architecture)**: Follows the service layer patterns established in E15. No direct code dependency, but architectural alignment is required.
- **E16 (Multi-Level Workflow System)**: Uses `workflow.Service.ForLevel(LevelBug)` for status transitions. The `ForLevel` infrastructure is confirmed working.

### Integration Points

- **`internal/keys/service.go`**: Bug key format (`B###`) must be recognizable by the key service for entity type detection. However, updating the key service is F06's scope -- F02 uses direct key validation via `ValidateBugKey()`.
- **`internal/fileops/writer.go`**: Bug file creation uses `WriteEntityFile` with `EntityType: "bug"` and `UseAtomicWrite: true`.
- **`internal/services/context_service.go`**: The context service must accept `"bug"` as a valid entity type. This is a 2-line switch case addition.
- **`internal/services/note_service.go`**: The note service must accept `"bug"` as a valid entity type. Same pattern as context service.
- **`internal/cli/services_global.go`**: The `GetBugService()` function wires all dependencies.

---

## Open Questions & Assumptions

No open questions -- all requirements are clear.

**Resolved Assumptions**:
1. Bug keys use the `B###` format (3 digits), supporting up to 999 bugs. This matches the epic design. If more than 999 bugs are needed, the format can be extended to `B####` in a future release without breaking existing keys.
2. The `assigned_to` field is stored as a context field (key-value pair in `context_data` JSON), not as a dedicated database column. This follows the pattern for optional metadata and avoids a schema change for an attribute that is only set during triage.
3. Slug generation for bugs follows the same algorithm as tasks/features/epics: lowercase title, replace spaces with hyphens, remove special characters.
4. The `docs/bugs/` directory is created automatically by `fileops.WriteEntityFile` if it does not exist (matching the existing behavior for `docs/plan/` directories).
5. Bug status history uses the same entity_notes or status_history infrastructure used by tasks, epics, and features. The specific mechanism depends on what F01 implements for the `bugs` table (triggers vs. explicit history inserts).

---

## Requirements Traceability

| Epic Requirement | F02 Coverage | How Verified |
|-----------------|-------------|-------------|
| REQ-F-001 (Bug Entity Creation) | CreateBug service with key gen, file creation | Service test: CreateBug returns persisted bug with B### key |
| REQ-F-002 (Bug Severity Tracking) | Severity field, default, validation, filtering | Model test: severity validation. Service test: default severity. Repo test: ListBySeverity |
| REQ-F-003 (Bug Entity Linking) | Link validation in CreateBug | Service test: valid link stored, invalid link rejected |
| REQ-F-004 (Bug Status Workflow) | AdvanceBugStatus, SetBugStatus via workflow service | Service test: full workflow traversal, terminal status error |
| REQ-F-005 (Bug Triage Command) | TriageBug service method | Service test: triage from reported, error from other statuses |
| REQ-F-015 (Bug Notes/Context) | Service supports entity_type "bug" for notes/context | Integration point: context_service, note_service switch cases |
| REQ-F-016 (Bug Markdown Template) | Bug markdown template | Template test: renders with all sections |
| REQ-F-018 (Filtering by Linked Entity) | ListByLinkedEntity repository method, BugFilters.LinkedEntityKey | Repo test: filter by linked entity |
| REQ-NF-001 (Creation Speed) | Efficient repository queries | Performance test: < 500ms |
| REQ-NF-002 (List Performance) | Indexed queries (from F01) | Performance test: < 1s for 1000 records |
| REQ-NF-003 (Atomic Operations) | fileops atomic write + DB transaction rollback | Test: simulate file failure, verify no DB record |
| REQ-NF-005 (CLI Pattern Consistency) | Service returns domain models, follows established patterns | Code review |

---

## Sprint Sizing

**Estimate**: M complexity (1-2 sprints)

| Component | Size | Notes |
|-----------|------|-------|
| Bug model + validation | S | Template from Idea model; add severity constants, key validation |
| Bug repository interface | S | Standard interface definition following consumer-side pattern |
| Bug repository implementation | M | CRUD + filtering queries + key generation + tests (real DB) |
| Bug service | M | CreateBug (key gen, link validation, file creation), TriageBug, status transitions |
| Service DTOs | S | 4 structs with JSON tags |
| Markdown template | S | Template file with sections |
| Service accessor | S | 1 function in services_global.go |
| Model tests | S | Validation paths, key format validation |
| Repository tests | M | CRUD + filtering + key generation with real DB and cleanup |
| Service tests | M | All business logic paths with mocked repos |

---

*Last Updated*: 2026-03-03
