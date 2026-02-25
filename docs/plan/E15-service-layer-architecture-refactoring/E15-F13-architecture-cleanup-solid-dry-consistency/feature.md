---
feature_key: E15-F13-architecture-cleanup-solid-dry-consistency
epic_key: E15
title: Architecture Cleanup - SOLID DRY Consistency
description: Complete the service layer migration by removing business logic from repositories, eliminating code duplication, and standardizing patterns across all entities.
---

# Architecture Cleanup - SOLID DRY Consistency

**Feature Key**: E15-F13-architecture-cleanup-solid-dry-consistency

---

## Goal

### Problem

An architecture review revealed that while the E15 service layer refactoring successfully moved business logic out of CLI commands into services, several issues remain:

1. **Smart Repositories**: `TaskRepository` still contains ~144 lines of orchestration logic (`updateStatusForcedInternal`), workflow validation, rejection note creation, and auto-unblock logic. `FeatureRepository` and `EpicRepository` contain cascade and status override business logic. This violates the stated architecture principle: "repositories must NOT contain business logic."

2. **DRY Violations**: Significant code duplication across entity services (document linking, constructor validation) and across repositories (GetByKey implementations, scan boilerplate, file path update methods).

3. **Inconsistent Patterns**: Constructor patterns vary between services (TaskService has 2 constructors + 4 setters, NoteService has 1 constructor + 0 setters). Repository structures are inconsistent (TaskRepository carries workflow config, others don't).

### Solution

Complete the repository-to-service migration for all remaining business logic, extract shared patterns to reduce duplication, and standardize constructor/dependency patterns across all entity services.

### Impact

- Repositories become truly "dumb" data access layers - testable in isolation
- Consistent patterns across all entities reduce cognitive load
- DRY improvements reduce maintenance surface area by ~1000+ lines
- Full SOLID compliance across the service and repository layers

---

## Architecture Review Findings

### Current State Assessment

#### Service Layer (internal/services/)

**Services Inventory:**
| Service | LOC | Constructors | Setters | Repo Interfaces |
|---------|-----|-------------|---------|-----------------|
| TaskService | ~418 | 2 | 4 | 8 |
| FeatureService | ~500 | 2 | 1 | 5 |
| EpicService | ~500 | 2 | 2 | 5 |
| NoteService | ~180 | 1 | 0 | 3 |
| ContextService | ~120 | 1 | 0 | 3 |
| ResumeService | ~300 | 1 | 0 | 5 |

**Strengths:**
- Excellent interface design - minimal, consumer-defined interfaces
- Proper separation of concerns - no formatting in services
- DTOs well-structured (25+ DTOs)
- Error wrapping with business context at each layer
- Context as first parameter consistently
- Services reusable across CLI and HTTP API

**Issues:**
- Inconsistent constructor variants (2 primary + setters vs 1 simple)
- Naming conventions vary across repo interfaces (_Repository, _Lookup, _Counter)
- Optional dependency handling differs between services

#### Repository Layer (internal/repository/)

**Repository Inventory:**
| Repository | Public Methods | Business Logic Methods |
|-----------|---------------|----------------------|
| TaskRepository | 28+ | 15+ (workflow validation, block/unblock, status orchestration) |
| FeatureRepository | 19 | 6 (cascade, status override) |
| EpicRepository | 19 | 6 (cascade, rollup) |

**Critical Finding - TaskRepository has 147% more methods than others** due to business logic that should be in TaskService.

**Specific Business Logic in Repositories:**

1. `TaskRepository.updateStatusForcedInternal()` (~lines 885-1029, 144 lines):
   - Workflow validation via `isValidTransition()`
   - Rejection reason validation
   - Timestamp management (started_at, completed_at, blocked_at)
   - History record creation
   - Rejection note creation
   - Auto-unblock logic

2. `TaskRepository.isValidStatusEnum()` / `isValidTransition()` (~lines 818-860):
   - Workflow config-driven validation belongs in service layer

3. `TaskRepository.BlockTask()` / `UnblockTask()` / `ReopenTask()` (~lines 1100-1241):
   - Transition orchestration with workflow validation

4. `FeatureRepository.CascadeStatusToTasks()` (~lines 958-986):
   - Business rule propagation

5. `FeatureRepository.SetStatusOverride()` (~lines 866-925):
   - Business logic for status override rules

6. `EpicRepository.CascadeStatusToFeaturesAndTasks()` (~lines 578-621):
   - Complex status propagation business rules

**Inconsistent Repository Structures:**
```go
TaskRepository struct {
    db       *DB
    workflow *config.WorkflowConfig  // Carries business config!
}
FeatureRepository struct { db *DB }  // No workflow config
EpicRepository struct { db *DB }     // No workflow config
```

#### CLI Layer (internal/cli/commands/)

**98% of commands properly use the service layer.** Only 3 exceptions, all justified infrastructure commands (cloud.go, init.go, migrate_backfill_slugs.go).

**Minor Issue:** `analytics.go` uses `os.Exit(3)` directly instead of returning errors.

#### DI Wiring (internal/cli/services_global.go)

- Thread-safe singletons via `sync.Once`
- Lazy initialization matching CLI usage patterns
- `buildTaskServiceDeps()` shared helper reduces duplication
- `GetTaskService()` calls `GetFeatureService()` creating tight coupling

### SOLID Compliance Summary

| Principle | Rating | Key Issue |
|-----------|--------|-----------|
| Single Responsibility | Good | TaskRepository violates - contains orchestration logic |
| Open/Closed | Good | Workflow profiles enable extension via config |
| Liskov Substitution | Excellent | All repo interfaces properly designed for mock injection |
| Interface Segregation | Excellent | Consumer-defined minimal interfaces |
| Dependency Inversion | Good | `*workflow.Service` used directly, not via interface |

### DRY Violations Identified

1. **Document Linking/Unlinking** (~15-20 lines x 3 entities): Identical `LinkDocument()`/`UnlinkDocument()` in TaskService, FeatureService, EpicService

2. **GetByKey Implementations** (3 different algorithms): Epic, Feature, Task repos each implement slug lookup differently

3. **Scan Boilerplate** (~1000+ lines): Identical `rows.Scan()` patterns across all repositories

4. **Constructor Nil-Check Validation**: Same panic pattern repeated in every service constructor

5. **File Path Update Methods**: Duplicated identically across epic/feature/task repos

6. **Setter Methods for Optional Repos**: 6+ identical setter methods across services

---

## Requirements

### Functional Requirements

**Category: Repository Cleanup (P1 - Critical)**

1. **REQ-F-001**: Extract TaskRepository Business Logic to TaskService
   - Move `updateStatusForcedInternal()` orchestration to TaskService
   - Move `isValidStatusEnum()` / `isValidTransition()` to service layer
   - Move `BlockTask()` / `UnblockTask()` / `ReopenTask()` orchestration to TaskService
   - Move `UpdateStatusWithAction()` / `getOrchestratorAction()` to service
   - Remove `workflow *config.WorkflowConfig` from TaskRepository struct
   - TaskRepository should only do: `UPDATE tasks SET status = ? WHERE id = ?`

2. **REQ-F-002**: Extract FeatureRepository Business Logic to FeatureService
   - Move `CascadeStatusToTasks()` / `CascadeStatusToTasksByKey()` to FeatureService
   - Move `SetStatusOverride()` / `SetStatusOverrideByKey()` to FeatureService
   - Move `UpdateStatusIfNotOverridden()` to FeatureService

3. **REQ-F-003**: Extract EpicRepository Business Logic to EpicService
   - Move `CascadeStatusToFeaturesAndTasks()` to EpicService

**Category: DRY Improvements (P2 - Important)**

4. **REQ-F-004**: Extract Shared Document Linking Pattern
   - Create shared helper or generic function for LinkDocument/UnlinkDocument
   - Used by TaskService, FeatureService, EpicService

5. **REQ-F-005**: Standardize GetByKey Across Repositories
   - Implement consistent slug lookup algorithm in all three entity repos
   - Extract shared lookup helper

6. **REQ-F-006**: Unify Service Constructor Patterns
   - Standardize on consistent approach for optional dependencies
   - Options: functional options pattern, builder pattern, or consistent setters

**Category: Minor Fixes (P3 - Cleanup)**

7. **REQ-F-007**: Fix analytics.go os.Exit() Usage
   - Replace direct os.Exit() calls with error returns

8. **REQ-F-008**: Extract Constructor Validation Helper
   - Create `requireNonNil()` helper for constructor panic patterns

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Repository Has No Business Logic**
- **Given** all tasks in this feature are complete
- **When** reviewing any repository file
- **Then** repositories contain ONLY: CRUD operations, prepared statements, scan patterns
- **And** no workflow validation, no cascade logic, no status orchestration

**Scenario 2: Consistent Patterns**
- **Given** all three entity services (Task, Feature, Epic)
- **When** comparing their structure
- **Then** constructor patterns follow the same convention
- **And** document linking uses shared code
- **And** GetByKey uses the same algorithm

**Scenario 3: All Tests Pass**
- **Given** refactoring is complete
- **When** running `make fmt && make lint && make test`
- **Then** all checks pass with no regressions

---

## Out of Scope

1. **Scan Boilerplate Reduction** - Code generation or reflection-based scanning would be a larger effort; defer to separate feature
2. **HTTP API Handler Implementation** - `WireServices()` is ready but handlers are a separate feature
3. **Service Factory / Wire Package** - Current manual DI is acceptable for project size
4. **Workflow Service Interface Extraction** - Using concrete `*workflow.Service` is acceptable

---

## Dependencies

- Existing E15 service layer (TaskService, FeatureService, EpicService must exist)
- All existing tests must continue to pass after refactoring

---

*Last Updated*: 2026-02-24
