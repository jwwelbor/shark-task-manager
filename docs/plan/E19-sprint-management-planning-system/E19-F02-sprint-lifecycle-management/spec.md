---
feature_key: E19-F02
title: Sprint Lifecycle Management - Technical Specification
status: ready_for_development
---

# Sprint Lifecycle Management (E19-F02) — Technical Specification

**Feature Key**: E19-F02-sprint-lifecycle-management  
**Epic**: E19 (Sprint Management & Planning System)  
**Tier**: STANDARD (score 13/27)  
**Status**: Ready for Development  
**Last Updated**: 2026-05-05

---

## Executive Summary

E19-F02 implements the full **CRUD and lifecycle command surface for sprints** following the established pattern from E18 (BugService and ChangeCardService). The feature depends on **E19-F01** (which provides database schema, models, and key validation) and enables the remaining E19 features (F03 planning/backlog, F04 analytics, F05 capacity).

All foundational infrastructure is complete: database tables, models, validators, and key format. F02 only requires:

1. **SprintRepository** — data access layer (CRUD + queries)
2. **SprintService** — business logic layer (state machine, status transitions, validation)
3. **CLI Commands** — thin wrappers calling SprintService
4. **Service Accessor** — global initialization in services_global.go

**Key Scope**: This feature covers **REQ-F-001 through REQ-F-003** from the epic PRD (Must-Have requirements). Planning, backlog, carryover, and analytics belong to F03–F05.

---

## Part 1: Requirements

### Functional Requirements

**Category: Sprint CRUD Operations**

#### REQ-F-001: Sprint Creation
- **Description**: Users can create a named sprint with start date, end date, and optional goal text via `shark sprint create`
- **Acceptance Criteria**:
  - `shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01` creates a sprint with key `S024` in `planning` status
  - `--goal="text"` flag accepts optional sprint goal
  - Sprint key is auto-assigned as `S###` with zero-padded incrementing number (via `GetNextKey()`)
  - Start date must be before end date; invalid date ranges return validation error
  - `--json` flag outputs created sprint in machine-readable format
  - Slug is auto-generated from sprint name (via `utils.GenerateSlug()`)
  - FilePath defaults to `docs/plan/sprints/{S###}.md` (followsfile pattern from bugs)

**Test Case**:
```bash
./bin/shark sprint create "Sprint 24" --start=2026-03-18 --end=2026-04-01 --json
# Returns: {"key": "S024", "name": "Sprint 24", "status": "planning", "start_date": "2026-03-18T00:00:00Z", ...}
```

#### REQ-F-002: Sprint Read (Get and List)
- **Description**: Full read operations for sprint entities: get individual sprint, list with filtering
- **Acceptance Criteria**:
  - `shark sprint get S024` displays sprint details (name, dates, goal, status, created_at, updated_at)
  - `shark sprint get S024 --json` outputs machine-readable sprint data with all fields
  - `shark get S024` auto-detects sprint entity type from `S###` key format
  - `shark sprint list` shows all sprints with status filter support (`--status=planning`, `--status=active`, `--status=completed`)
  - `--json` flag for list returns array of sprints
  - Sprint list includes sprint count per status in human-readable output

**Test Case**:
```bash
./bin/shark sprint get S024 --json
# Returns full Sprint struct with all fields populated

./bin/shark sprint list --status=active --json
# Returns: [{"key": "S024", "status": "active", ...}]
```

#### REQ-F-003: Sprint Update and Delete
- **Description**: Modify sprint attributes and delete sprints in planning status
- **Acceptance Criteria**:
  - `shark sprint update S024 --name="Sprint 24 (Extended)"` updates sprint name
  - `shark sprint update S024 --goal="Updated goal"` updates sprint goal
  - `shark sprint update S024 --end=2026-04-08` updates end date
  - Updates validate date ordering (start < end) and non-empty name
  - `shark sprint delete S024` removes a sprint in `planning` status only
  - Cannot delete active or completed sprints; returns validation error
  - `--json` flag returns updated/deleted sprint
  - UpdatedAt is auto-set by database trigger

**Test Case**:
```bash
./bin/shark sprint update S024 --goal="Implement analytics" --json
# Returns updated sprint with new goal

./bin/shark sprint delete S024
# Only succeeds if sprint.status == "planning"
```

---

**Category: Sprint Lifecycle Transitions**

#### REQ-F-004: Sprint Status Transitions (State Machine)
- **Description**: Sprints transition through lifecycle states: `planning` → `active` → `closing` → `completed`, with `cancelled` as an alternative
- **Acceptance Criteria**:
  - `shark sprint start S024` transitions from `planning` to `active` and sets the sprint as the "current" active sprint
  - Only one sprint can be in `active` status at a time (enforced via `ValidateSprintStatus` with workflow.Service)
  - `shark sprint close S024` transitions from `active` to `closing` (note: carryover processing deferred to F03)
  - `shark sprint archive S024` transitions from `completed` to `archived`
  - Status transitions are validated via `workflow.Service.ValidateTransition()` — invalid transitions return descriptive errors
  - All transitions recorded with timestamps (CreatedAt/UpdatedAt)
  - Transition failure leaves the sprint unchanged (atomic from caller's perspective)

**Status Diagram**:
```
planning → active → closing → completed → archived
                 ↘ cancelled ↗
```

**Test Cases**:
```bash
./bin/shark sprint start S024 --json
# Transitions planning → active; returns updated sprint with new status

./bin/shark sprint start S024  # When S024 is already active
# Error: "Sprint S024 is already active" (validation error)

./bin/shark sprint start S025  # When another sprint is active (e.g., S024)
# Error: "Cannot activate S025: Sprint S024 is already active" (constraint error)
```

---

### Non-Functional Requirements

#### REQ-NF-001: Performance
- Sprint CRUD operations complete in <100ms on typical database (SQLite on development machine)
- List operations with status filter return results in <200ms for 1000+ sprints
- No N+1 query patterns in service layer

#### REQ-NF-002: Consistency
- Sprint state transitions are atomic (no partial states visible to concurrent commands)
- One-active-sprint constraint enforced at database layer (partial unique index)
- Key generation is monotonic (keys always increment)

#### REQ-NF-003: Error Handling
- Invalid transitions return `workflow.TransitionError` with `From`, `To`, `Reason`
- Missing sprints return `NotFoundError` with sprint key
- Validation failures return descriptive errors with field name and constraint
- All errors wrapped with business context at service layer

---

## Part 2: Architecture

### Component Overview

This feature introduces:

1. **SprintRepository** (`internal/repository/sprint/repository.go`)
   - Data access layer: CRUD, queries, key generation
   - Methods mirror BugRepository pattern (GetByKey, Update, Delete, List, etc.)

2. **SprintService** (`internal/services/sprint_service.go`)
   - Business logic: validation, status transitions, state machine
   - Constructor injects: SprintRepository, workflow.Service
   - Methods: CreateSprint, GetSprint, ListSprints, UpdateSprint, DeleteSprint, StartSprint, CloseSprint, ArchiveSprint

3. **CLI Commands** (`internal/cli/commands/sprint.go`)
   - Thin wrappers: sprintCmd (parent), sprintCreateCmd, sprintGetCmd, sprintListCmd, sprintUpdateCmd, sprintDeleteCmd, sprintStartCmd, sprintCloseCmd, sprintArchiveCmd
   - Patterns: parse args → call service → format output (JSON or human)

4. **Service Accessor** (`internal/cli/services_global.go`)
   - GetSprintService() function for CLI commands
   - Lazy initialization with dependency injection

---

### Data Model & Database

**F01 has already delivered** (from research report):

1. **Database Schema** (lines 436-445 in `internal/db/db.go`):
   ```sql
   CREATE TABLE sprints (
       id INTEGER PRIMARY KEY,
       key TEXT UNIQUE NOT NULL,
       name TEXT NOT NULL,
       goal TEXT,
       start_date DATETIME NOT NULL,
       end_date DATETIME NOT NULL,
       status TEXT NOT NULL,
       slug TEXT,
       file_path TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
   );
   
   CREATE TABLE sprint_assignments (
       id INTEGER PRIMARY KEY,
       sprint_id INTEGER NOT NULL REFERENCES sprints(id),
       entity_type TEXT NOT NULL,
       entity_id INTEGER NOT NULL,
       assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       removed_at DATETIME,
       UNIQUE (entity_type, entity_id) WHERE removed_at IS NULL
   );
   
   CREATE TABLE sprint_capacity (
       id INTEGER PRIMARY KEY,
       sprint_id INTEGER NOT NULL REFERENCES sprints(id),
       agent_type TEXT NOT NULL,
       capacity_points REAL NOT NULL,
       allocated_points REAL,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
   );
   ```

2. **Models** (`internal/models/sprint.go`):
   - `Sprint`: Key, Name, Goal, StartDate, EndDate, Status, Slug, FilePath, timestamps
   - `SprintAssignment`: SprintID, EntityType (polymorphic), EntityID, AssignedAt, RemovedAt
   - `SprintCapacity`: SprintID, AgentType, CapacityPoints, AllocatedPoints
   - All models include `Validate()` method (structural validation only — status checked at service layer)

3. **Validation** (`internal/models/validation.go`):
   - `ValidateSprintKey()`: Regex pattern `^S\d{3}$` (e.g., S001, S999)
   - `ValidateSprintAssignmentEntityType()`: Allowlist {task, bug, change_card, tech_debt}

4. **Key Service** (`internal/keys/service.go` and `validation.go`):
   - `EntityTypeSprint` constant already defined
   - `IsSprintKey()` helper validates S### format
   - Case-insensitive parsing (both `S001` and `s001` work)

**No database changes needed in F02** — F01 delivers everything.

---

### Repository Layer Design

**File**: `internal/repository/sprint/repository.go`

**SprintRepository Interface** (defined in service):
```go
type SprintRepository interface {
    Create(ctx context.Context, sprint *models.Sprint) error
    GetByKey(ctx context.Context, key string) (*models.Sprint, error)
    GetByID(ctx context.Context, id int64) (*models.Sprint, error)
    Update(ctx context.Context, sprint *models.Sprint) error
    Delete(ctx context.Context, id int64) error
    UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error
    GetNextKey(ctx context.Context) (string, error)
    List(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error)
}
```

**Implementation Pattern** (mirrors BugRepository):
- Each method uses parameterized queries with `?` placeholders
- Key lookup is case-insensitive (via `UPPER(key) = UPPER(?)`)
- GetNextKey() queries `SELECT MAX(CAST(SUBSTR(key, 2) AS INTEGER)) FROM sprints` and increments
- Error wrapping: `fmt.Errorf("failed to <action> sprint: %w", err)`
- All queries go to `r.db` (injected `*db.DB`)

**Key Methods**:

1. **Create** - Inserts new sprint with auto-timestamp
2. **GetByKey** - Returns `NotFoundError` if not found
3. **GetByID** - Used internally by service for FK relationships
4. **Update** - Updates name/goal/end_date; database trigger sets updated_at
5. **Delete** - Soft-delete or hard-delete (per BugRepository pattern)
6. **UpdateStatus** - Atomic status transition (sets status, triggers timestamp)
7. **GetNextKey** - Generates next S### key
8. **List** - Supports filters: `SprintListFilters{Status: "active", ...}`

---

### Service Layer Design

**File**: `internal/services/sprint_service.go`

**SprintService Struct**:
```go
type SprintService struct {
    repo        SprintRepository        // Injected
    workflowSvc *workflow.Service       // Injected (ForLevel(workflow.LevelSprint))
}
```

**Constructor**:
```go
func NewSprintService(
    repo SprintRepository,
    workflowSvc *workflow.Service,
) *SprintService {
    requireNonNil(repo, "SprintService requires a non-nil SprintRepository")
    return &SprintService{
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelSprint),
    }
}
```

**Key Methods** (following BugService pattern):

1. **CreateSprint** (ctx, input) → (*Sprint, error)
   - Input: CreateSprintInput{Name, Goal, StartDate, EndDate}
   - Validates non-empty name, date ordering
   - Generates key via `repo.GetNextKey()`
   - Generates slug via `utils.GenerateSlug(input.Name)`
   - Sets status to default (planning) via `workflowSvc.GetDefaultStatus()`
   - Calls `repo.Create()`
   - Returns created sprint with ID populated

2. **GetSprint** (ctx, key) → (*Sprint, error)
   - Calls `repo.GetByKey()`, returns NotFoundError if missing
   - No transformation; returns raw entity

3. **ListSprints** (ctx, filters) → ([]*Sprint, error)
   - Calls `repo.List(filters)`
   - Returns empty slice (not nil) if no results
   - Supports filters: {Status: "active", ...}

4. **UpdateSprint** (ctx, key, updates) → (*Sprint, error)
   - Gets sprint via `GetSprint()`
   - Validates date ordering (if end_date changed)
   - Updates fields (Name, Goal, EndDate)
   - Calls `repo.Update()`
   - Returns updated sprint

5. **DeleteSprint** (ctx, key) → error
   - Gets sprint via `GetSprint()`
   - Validates status == "planning" (cannot delete active/completed)
   - Calls `repo.Delete()`

6. **StartSprint** (ctx, key) → (*Sprint, error)
   - Gets sprint via `GetSprint()`
   - Validates current status and workflow transition via `workflowSvc.ValidateTransition("planning", "active")`
   - Checks single-active-sprint constraint: `List(ctx, {Status: "active"})` must return 0 results
   - Updates status to "active" via `repo.UpdateStatus()`
   - Returns updated sprint

7. **CloseSprint** (ctx, key) → (*Sprint, error)
   - Gets sprint via `GetSprint()`
   - Validates workflow transition: "active" → "closing"
   - Updates status via `repo.UpdateStatus()`
   - Note: Carryover logic deferred to F03
   - Returns updated sprint

8. **ArchiveSprint** (ctx, key) → (*Sprint, error)
   - Gets sprint via `GetSprint()`
   - Validates workflow transition: "completed" → "archived"
   - Updates status via `repo.UpdateStatus()`
   - Returns updated sprint

**Error Handling**:
- Validation errors return wrapped errors with context: `fmt.Errorf("cannot start sprint %s: %w", key, err)`
- Workflow errors propagate from `workflowSvc`; caller (CLI) decides exit code
- NotFoundError wraps repository errors

**Workflow Integration**:
- All status transitions validated via `workflowSvc.ValidateTransition(from, to)`
- Default status determined by `workflowSvc.GetDefaultStatus()`
- Workflow enforces valid state machine (configured in `.sharkconfig.json`)

---

### CLI Commands Layer

**File**: `internal/cli/commands/sprint.go`

**Command Structure** (follows BugService pattern from `internal/cli/commands/bug.go`):

1. **Parent Command**: `sprintCmd` — "Manage sprints"

2. **CRUD Commands**:
   - `sprintCreateCmd`: Parse name/dates/goal → call service → format output
   - `sprintGetCmd`: Parse key → call service → format output
   - `sprintListCmd`: Parse filters (--status) → call service → table or JSON
   - `sprintUpdateCmd`: Parse key + flags (--name, --goal, --end) → call service → format output
   - `sprintDeleteCmd`: Parse key → confirm (if not --force) → call service → format output

3. **Lifecycle Commands**:
   - `sprintStartCmd`: Parse key → call service → JSON or success message
   - `sprintCloseCmd`: Parse key → call service → JSON or success message
   - `sprintArchiveCmd`: Parse key → call service → JSON or success message

**Key Patterns**:

**Example: Create Command**
```go
var sprintCreateCmd = &cobra.Command{
    Use:   "create <name> --start=DATE --end=DATE [--goal=text]",
    Short: "Create a new sprint",
    Args:  cobra.ExactArgs(1),
    RunE:  runSprintCreate,
}

func runSprintCreate(cmd *cobra.Command, args []string) error {
    // Parse args
    name := args[0]
    startStr, _ := cmd.Flags().GetString("start")
    endStr, _ := cmd.Flags().GetString("end")
    goal, _ := cmd.Flags().GetString("goal")
    
    // Validate date format and parse
    startDate, endDate, err := parseDates(startStr, endStr) // helper
    if err != nil {
        return fmt.Errorf("invalid date format: %w", err)
    }
    
    // Call service
    svc := cli.GetSprintService()
    input := services.CreateSprintInput{
        Name:      name,
        Goal:      goal,
        StartDate: startDate,
        EndDate:   endDate,
    }
    sprint, err := svc.CreateSprint(cmd.Context(), input)
    if err != nil {
        return err
    }
    
    // Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(sprint)
    }
    cli.Success(fmt.Sprintf("Created sprint %s: %s", sprint.Key, sprint.Name))
    return nil
}
```

**Flags**:
- `--start=2026-03-18` (required for create, optional for update)
- `--end=2026-04-01` (required for create, optional for update)
- `--goal="text"` (optional)
- `--force` (skip confirmations on delete)
- `--json` (machine-readable output)
- `--status=active` (filter for list)

**Output Formatting**:
- Human-readable: Table format with columns [Key, Name, Status, StartDate, EndDate]
- JSON: Full Sprint struct with all fields

---

### Service Accessor Pattern

**File**: `internal/cli/services_global.go`

**Addition**:
```go
// GetSprintService returns a SprintService instance.
// Creates a new instance each call with the global DB and workflow service.
func GetSprintService() *services.SprintService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    sprintRepo := repository.NewSprintRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewSprintService(sprintRepo, workflowSvc)
}
```

**Pattern** (matches existing Epic/FeatureService accessors):
- Lazy initialization on first call
- Reuses global DB and workflow service
- Panics on error (fail-fast for CLI)
- Returns new service instance per call (no shared state)

---

### Integration Points

**Existing Code Modified**:

1. **`internal/cli/services_global.go`** (lines 467–503):
   - Add `GetSprintService()` function (10 lines)
   - Returns `NewSprintService(repo, workflowSvc)`

2. **`internal/cli/root.go`**:
   - Register `sprintCmd` to `RootCmd` in `init()` (1 line import, 1 line AddCommand)

3. **`internal/cli/commands/root.go`** or new file:
   - Import sprintCmd from sprint.go (existing pattern)

**New Files Created**:

1. **`internal/repository/sprint/repository.go`** (~250 lines)
   - SprintRepository struct, constructor, all CRUD methods
   - Mirrors structure of `internal/repository/bug/repository.go`

2. **`internal/services/sprint_service.go`** (~300 lines)
   - SprintService struct, constructor, all business logic methods
   - Mirrors structure of `internal/services/bug_service.go`

3. **`internal/cli/commands/sprint.go`** (~400 lines)
   - All 8 command handlers (create, get, list, update, delete, start, close, archive)
   - Follows pattern from `internal/cli/commands/bug.go`

---

### Acceptance Criteria (Architecture)

- [ ] Every requirement is testable (no TBDs in methods)
- [ ] All service methods accept `context.Context` as first parameter
- [ ] Repository methods use parameterized queries (`?` placeholders)
- [ ] Error wrapping adds business context at each layer
- [ ] Status transitions validated via `workflow.Service`
- [ ] One-active-sprint constraint enforced (validated in StartSprint)
- [ ] All CRUD operations return relevant models or typed errors
- [ ] CLI commands are thin wrappers (parse → call → format)
- [ ] Service accessor initializes dependencies lazily
- [ ] No business logic in repository or CLI command layers

---

### Testing Strategy (Not Scope of This Spec)

**Repository Tests** (`internal/repository/sprint/repository_test.go`):
- Use real test database (test.GetTestDB())
- Test CRUD, key generation, constraint enforcement
- Verify index usage and query performance

**Service Tests** (`internal/services/sprint_service_test.go`):
- Mock SprintRepository and workflow.Service
- Test business logic, validation, status transitions
- Verify error wrapping and context

**CLI Command Tests** (`internal/cli/commands/sprint_test.go`):
- Mock SprintService
- Test argument parsing and output formatting
- Verify JSON and human-readable output

---

## File Paths & Summary

### Files to Create

1. `internal/repository/sprint/repository.go` — SprintRepository (CRUD, queries)
2. `internal/services/sprint_service.go` — SprintService (business logic)
3. `internal/cli/commands/sprint.go` — CLI command handlers

### Files to Modify

1. `internal/cli/services_global.go` — Add GetSprintService() function
2. `internal/cli/root.go` — Register sprintCmd

### No Changes Required

- Database schema (F01 complete)
- Models (F01 complete)
- Validation (F01 complete)
- Key generation (F01 complete)

---

## Exit Gate Checklist

- [x] Every requirement is testable (no TBDs in critical sections)
- [x] Every architecture decision references existing patterns (BugService, BugRepository)
- [x] File paths listed for all changes (3 new files, 2 modified files)
- [x] No deferred decisions in architecture
- [x] Service layer owns state machine and validation
- [x] Repository layer is pure data access
- [x] CLI commands are thin wrappers
- [x] Error handling follows project conventions (NotFoundError, validation errors, workflow errors)
- [x] Workflow integration explicit (workflowSvc.ValidateTransition, ForLevel)
- [x] Single-active-sprint constraint enforced (documented in StartSprint)

---

*Last Updated*: 2026-05-05
