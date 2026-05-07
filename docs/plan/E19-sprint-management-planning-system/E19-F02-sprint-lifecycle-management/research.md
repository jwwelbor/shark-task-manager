# Sprint Lifecycle Management (E19-F02) — Research Report

**Feature Key**: E19-F02-sprint-lifecycle-management  
**Epic**: E19 (Sprint Management & Planning System)  
**Complexity**: COMPLEX  
**Date**: 2026-05-05

---

## 1. Executive Summary

E19-F02 implements the full CRUD and lifecycle command surface for sprints following the established pattern from E18 (bugs and change-cards). The feature depends on E19-F01 (already complete) which provides:
- Database schema (3 tables: sprints, sprint_assignments, sprint_capacity)
- Sprint, SprintAssignment, SprintCapacity models
- Validation layer for sprint keys (S### format)
- Schema version 18 with idempotent migrations

**Key Finding**: The entire foundational layer is built. F02 only needs to implement:
1. SprintRepository (data access CRUD)
2. SprintService (business logic + state machine)
3. CLI commands (thin wrappers calling SprintService)
4. Service accessor in services_global.go

**Extensibility**: All 4 assignable entity types (task, bug, change_card, tech_debt) are pre-configured in the polymorphic sprint_assignments table. Sprint commands work with all types by design.

---

## 2. Existing Implementations (What's Built)

### 2.1 Database Layer (COMPLETE - F01)

**File**: `/home/jwwel/projects/shark-task-manager/internal/db/db.go`

- **Schema Version**: 18 (lines 436-445)
- **Migration Function**: `migrateSprintTables()` (lines 3358+)
- **Tables**:
  - `sprints` (id, key, name, goal, start_date, end_date, status, slug, file_path, created_at, updated_at)
  - `sprint_assignments` (id, sprint_id, entity_type, entity_id, assigned_at, removed_at) — polymorphic
  - `sprint_capacity` (id, sprint_id, agent_type, capacity_points, allocated_points, created_at, updated_at)
- **Indexes**: On key, status, slug; partial unique index on (entity_type, entity_id) WHERE removed_at IS NULL
- **Constraints**: start_date < end_date; single-active-sprint-per-entity enforced via partial index
- **Triggers**: Auto-timestamp on sprints.updated_at

### 2.2 Models Layer (COMPLETE - F01)

**File**: `/home/jwwel/projects/shark-task-manager/internal/models/sprint.go`

- **Sprint struct** (lines 25-37):
  - ID, Key (S###), Name, Goal, StartDate, EndDate, Status, Slug, FilePath, CreatedAt, UpdatedAt
  - `Validate()` method: checks key format, non-empty name, date ordering, non-empty status
  - **Does NOT** validate status validity (deferred to service layer)

- **SprintAssignment struct** (lines 52-59):
  - SprintID, EntityType, EntityID, AssignedAt, RemovedAt
  - Polymorphic: entity_type ∈ {task, bug, change_card, tech_debt}
  - RemovedAt nullable: NULL = active assignment, non-NULL = historical

- **SprintCapacity struct** (lines 64-72):
  - SprintID, AgentType, CapacityPoints, AllocatedPoints (optional), CreatedAt, UpdatedAt
  - Represents capacity budget per agent type per sprint

### 2.3 Keys & Validation (COMPLETE - F01)

**Files**:
- `/home/jwwel/projects/shark-task-manager/internal/keys/service.go` (lines 28-31, 214)
- `/home/jwwel/projects/shark-task-manager/internal/keys/validation.go` (lines 215-221)

- **EntityTypeSprint**: Constant defined, recognized by KeyService.Parse()
- **IsSprintKey()**: Helper validates S### format (3-digit zero-padded)
- **Case-insensitive**: Both `S001` and `s001` parse correctly to EntityTypeSprint
- **Slug support**: Future sprint slugs (S001-sprint-name) will work via dual-key lookup pattern established by E07-F42

---

## 3. Architecture & Dependency Map

### 3.1 Service Layer Pattern (Reference: Bug & ChangeCard)

**Bug Service** (`internal/services/bug_service.go` — 576 lines):
- Constructor: NewBugService(repo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, projectRoot, tagSvc)
- Methods: CreateBug, GetBug, ListBugs, UpdateBug, DeleteBug, TriageBug, GetNextStatusForBug, GetOrchestratorAction
- Uses repositories: BugRepository (primary), EntityService, EpicRepository, FeatureRepository, TaskRepository
- Manages: Status transitions, workflow validation, lifecycle state machine

**Change Card Service** (`internal/services/change_card_service.go` — 525 lines):
- Similar constructor and method set to BugService
- Manages: Status transitions, approval workflows

**SprintService (to be built)**: Should follow this pattern exactly for consistency:
- Constructor injection of repositories
- Thin wrapper methods calling repository layer
- Workflow validation via shared WorkflowService
- Integration with EntityService for polymorphic assignment tracking

### 3.2 Repository Layer Pattern (Reference: Bug & ChangeCard)

**Bug Repository** (`internal/repository/bug/repository.go`):
- Constructor: NewBugRepository(db *DB)
- Methods: Create, GetByKey, Update, Delete, List, TriageBug, etc.
- Pure data access (CRUD + queries)
- No business logic, no status validation

**SprintRepository (to be built)**: Will provide:
- CRUD: Create, GetByKey, Update, Delete
- Queries: ListSprints, ListByStatus, GetAssignments, etc.
- Lifecycle: UpdateStatus
- Pure data access layer

### 3.3 CLI Commands Pattern (Reference: Bug Commands)

**File**: `/home/jwwel/projects/shark-task-manager/internal/cli/commands/bug.go`

Pattern:
1. Service accessor via `getBugService()` (test override support)
2. Each command is a thin wrapper: parse → call service → format output
3. Commands: bugCmd (parent), bugCreateCmd, bugGetCmd, bugListCmd, bugUpdateCmd, bugDeleteCmd, bugTriageCmd, etc.
4. Output: JSON support via `cli.GlobalConfig.JSON`, table formatting via `cli.OutputTable()`

**SprintCommands (to be built)**: Will follow identical pattern:
- sprintCmd (parent: "Manage sprints")
- sprintCreateCmd, sprintGetCmd, sprintListCmd, sprintUpdateCmd, sprintDeleteCmd
- sprintStartCmd, sprintCloseCmd, sprintArchiveCmd (lifecycle transitions)
- sprintAddCmd, sprintRemoveCmd (assignment management)

### 3.4 Service Accessor Pattern

**File**: `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` (lines 467-503)

**GetBugService()** example shows the pattern to follow. **GetSprintService()** (to be built) will be simpler:
- Inject: SprintRepository, TaskRepository, BugRepository, ChangeCardRepository, TechDebtRepository for assignment lookups
- No tag service needed (sprints don't have tags)
- No size enforcement (sprints aren't sized)
- Will likely use EntityService for polymorphic operations

---

## 4. Inter-Feature Dependency Map (E19 Scope)

### E19 Features:
1. **E19-F01**: Sprint Database Schema (COMPLETE)
2. **E19-F02**: Sprint Lifecycle Management (THIS FEATURE) — CRUD + lifecycle commands
3. **E19-F03**: Sprint Planning & Backlog (depends on F02) — bulk assignment, carryover logic
4. **E19-F04**: Sprint Analytics (depends on F02, F03) — velocity, burndown, summary reports
5. **E19-F05**: Agent Capacity Management (depends on F02, F04) — capacity allocation, overallocation detection

### Dependency Chain:
```
E19-F01 (DB schema, models)
    ↓
E19-F02 (CRUD commands, service layer) ← THIS FEATURE
    ↓
E19-F03 (bulk operations, carryover)
    ↓
E19-F04 (analytics)
    ↓
E19-F05 (capacity)
```

### Cross-Epic Dependencies:
- **E07-F42** (Entity `size` field): Velocity and capacity depend on task.size. F02 should handle sprints without size dependency.
- **E13** (Workflow & Status): Sprint status workflows are config-driven via `.sharkconfig.json`. Reuse workflow.Service pattern.
- **E28-F04** (Tags): Not in scope for F02. Tags can be added later via polymorphic entity tagging.

---

## 5. Extension vs New Analysis

### 5.1 What Can Be Extended (Reuse)

| Component | File Path | Reuse Strategy | Notes |
|-----------|-----------|-----------------|-------|
| **Workflow Service** | `internal/workflow/service.go` | Reuse directly | Sprint status workflow must be config-driven. Use WorkflowService.IsValidTransition() |
| **EntityService** | `internal/services/entity_service.go` | Reuse for history tracking | Track sprint status changes in entity_history table |
| **Repository Base Patterns** | `internal/repository/bug/`, `internal/repository/changecard/` | Copy structure | SprintRepository follows BugRepository exactly |
| **CLI Command Patterns** | `internal/cli/commands/bug.go`, `change.go` | Copy thin-wrapper pattern | All sprint commands use same parse→call→format skeleton |
| **Service Accessor Pattern** | `internal/cli/services_global.go` | Extend with GetSprintService() | Add function following BugService example |
| **Status Management** | `internal/status/` | Reuse CalculationService | For progress calculation if needed |
| **Key Parsing** | `internal/keys/service.go` | Already extended | EntityTypeSprint already recognized |
| **Slug Architecture** | `internal/keys/` | Already ready | Dual-key lookup (S001 + S001-my-sprint) works via existing pattern |
| **Validation Layer** | `internal/models/validation.go` | Extend (ValidateSprintKey, ValidateSprintStatus) | Add sprint-specific validators |

### 5.2 What Must Be Built New

| Component | Location | Scope | Lines (Est.) |
|-----------|----------|-------|--------------|
| **SprintRepository** | `internal/repository/sprint/repository.go` | CRUD for sprints, assignments, capacity | 300-400 |
| **SprintRepository Tests** | `internal/repository/sprint/repository_test.go` | Positive + negative test cases | 150-200 |
| **SprintService** | `internal/services/sprint_service.go` | Business logic, lifecycle state machine, assignment management | 250-350 |
| **SprintService Tests** | `internal/services/sprint_service_test.go` | Mock-based tests for all state transitions | 200-300 |
| **Sprint DTO** | `internal/services/sprint_dto.go` | Input DTOs | 50-100 |
| **Sprint CLI Commands** | `internal/cli/commands/sprint.go` | 8-10 command handlers | 400-600 |
| **Sprint CLI Tests** | `internal/cli/commands/sprint_test.go` | Test command parsing, JSON output, error handling | 150-250 |
| **Service Accessor** | `internal/cli/services_global.go` | Add GetSprintService() function | 20-30 |

**Total New Code**: ~1500-2500 lines (comparable to BugService + commands)

---

## 6. Implementation Approach (Recommended)

### Phase 1: Foundation
1. **SprintRepository**: Pure CRUD for sprint entity, assignments, capacity
2. **SprintService**: Business logic layer with state machine
3. **Status Workflow Configuration**: Add sprint status config to `.sharkconfig.json`

### Phase 2: CLI Layer
1. **Sprint Commands**: create, get, list, update, delete, start, close, archive
2. **Assignment Commands**: add, remove (may move to F03)
3. **Service Accessor**: GetSprintService() in services_global.go

### Phase 3: Testing & QA
1. **Repository Tests**: All CRUD + constraint validation
2. **Service Tests**: State machine transitions, validation rules
3. **Command Tests**: Argument parsing, JSON output
4. **Integration Tests**: End-to-end flow

---

## 7. Critical Implementation Details

### 7.1 State Machine (SprintService)

**Valid Transitions**:
```
planning → active → closing → completed
  ↓         ↓         ↓
  └─────────┴─────────→ cancelled

any → archived
```

**Invariants**:
- Only ONE sprint can have status='active' at a time
- On StartSprint, verify no other active sprint exists

### 7.2 Polymorphic Assignment (Sprint_Assignments)

**Supported EntityTypes**: task, bug, change_card, tech_debt

**Single-Active Rule**: A task can belong to at most one active sprint at a time.
- Enforced by: Partial unique index on (entity_type, entity_id) WHERE removed_at IS NULL
- When reassigning, soft-remove old assignment before creating new one

**Historical Tracking**: RemovedAt column preserves assignment history for analytics (F04).

### 7.3 Key Generation

**S### Format**: Auto-increment via SQLite AUTOINCREMENT on sprints.id
- sprintRepository.Create() validates S### format
- Example: S001, S002, …, S999

### 7.4 Status Workflow (Config-Driven)

Sprint status must be config-driven from `.sharkconfig.json`:
```json
{
  "status_metadata": {
    "planning": { "phase": "planning", "color": "gray" },
    "active": { "phase": "development", "color": "yellow" },
    "closing": { "phase": "review", "color": "cyan" },
    "completed": { "phase": "done", "color": "green" },
    "cancelled": { "phase": "any", "color": "red" },
    "archived": { "phase": "any", "color": "gray" }
  }
}
```

**Service Layer Validation**: Use WorkflowService.IsValidTransition(), don't hardcode.

---

## 8. Files to Create/Modify

### Create:
1. `/home/jwwel/projects/shark-task-manager/internal/repository/sprint/repository.go`
2. `/home/jwwel/projects/shark-task-manager/internal/repository/sprint/repository_test.go`
3. `/home/jwwel/projects/shark-task-manager/internal/services/sprint_service.go`
4. `/home/jwwel/projects/shark-task-manager/internal/services/sprint_service_test.go`
5. `/home/jwwel/projects/shark-task-manager/internal/services/sprint_dto.go`
6. `/home/jwwel/projects/shark-task-manager/internal/cli/commands/sprint.go`
7. `/home/jwwel/projects/shark-task-manager/internal/cli/commands/sprint_test.go`

### Modify:
1. `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` — Add GetSprintService()
2. `/home/jwwel/projects/shark-task-manager/internal/cli/root.go` — Register sprintCmd
3. `/home/jwwel/projects/shark-task-manager/.sharkconfig.json` — Add sprint status config

### No Changes Needed:
- Database layer (E19-F01 complete)
- Models (E19-F01 complete)
- Keys validation (already supports EntityTypeSprint)
- Workflow service (reuse existing)

---

## 9. Success Criteria

- [ ] SprintRepository implements full CRUD for sprints, assignments, capacity
- [ ] SprintService enforces single-active-sprint rule and state machine
- [ ] CLI commands (create, get, list, start, close, archive, add, remove) work end-to-end
- [ ] All commands support --json flag
- [ ] Service accessor GetSprintService() is registered in services_global.go
- [ ] Tests: Repository tests use real DB; Service/Command tests use mocks
- [ ] Integration tests verify: create → start → list with status filter → close → archive
- [ ] Config-driven status workflow is in place

---

## 10. Related Files (Reference Only)

| File | Purpose |
|------|---------|
| `internal/db/db.go` | Sprint schema migration |
| `internal/models/sprint.go` | Data structures |
| `internal/keys/service.go` | S### key parsing |
| `internal/services/bug_service.go` | Service template (576 lines) |
| `internal/cli/commands/bug.go` | Command template |
| `internal/cli/services_global.go` | Service wiring |
| `internal/repository/bug/repository.go` | Repository template |
| `.sharkconfig.json` | Workflow config |

---

**Report Status**: COMPLETE - All existing implementations identified with file paths, extension points documented, actionable for architect in specify step.
