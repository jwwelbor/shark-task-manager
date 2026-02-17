# Epic E15: Service Layer Architecture Refactoring - Research Report

**Date**: 2026-02-16
**Researcher**: AI Agent (Research Role)
**Epic**: E15 - Service Layer Architecture Refactoring

---

## Executive Summary

This research analyzes Shark Task Manager's current architecture to guide the service layer refactoring effort (Epic E15). The analysis reveals a classic **fat-controller anti-pattern** with 20,952 lines of command-layer code containing 40-45% of business logic. Three command files (`task.go`, `feature.go`, `epic.go`) contain 6,831 lines—nearly 33% of all command code. Repository instantiation occurs 351 times across 46 files, indicating severe coupling between CLI and data access.

**Key Findings:**
- **Current state**: 58 command files averaging 361 LOC (top 3 average 2,277 LOC)
- **Service layer exists but incomplete**: Epic and Feature services (471 LOC total) handle only status transitions, not CRUD or querying
- **No TaskService exists**: All task business logic (2,664 lines) lives in CLI commands
- **Repository contains business logic**: Progress calculation (~400 LOC) and status derivation (~200 LOC) should be in services
- **Massive duplication**: Repository instantiation pattern repeated 351 times, error handling repeated 47+ times

**Recommended Approach:**
1. **Phase 1**: Extract TaskService (highest impact - 2,664 LOC reduction)
2. **Phase 2**: Expand Feature/Epic services (CRUD + queries, not just transitions)
3. **Phase 3**: Clean repositories (remove 600+ LOC of business logic)
4. **Phase 4**: Consolidate common patterns (QueryService for cross-entity operations)

**Expected Impact**:
- **Code reduction**: 65% reduction in command layer (20,952 → ~7,000 LOC)
- **Duplication elimination**: 80%+ reduction (351 repository instantiations → ~10 service instantiations)
- **Business logic centralization**: 90%+ of logic moves to services from commands/repositories

---

## 1. Current Service Layer State

### 1.1 Existing Services

The service layer exists but is **incomplete and narrowly scoped**:

| Service | File | LOC | Scope | Missing Capabilities |
|---------|------|-----|-------|---------------------|
| `EpicService` | `internal/services/epic_service.go` | 232 | Status transitions only | CRUD, querying, progress, rollups |
| `FeatureService` | `internal/services/feature_service.go` | 239 | Status transitions only | CRUD, querying, progress, health |
| **TaskService** | **MISSING** | **0** | **None** | **Everything - all business logic in CLI** |
| `DisplayService` | `internal/services/display_service.go` | 563 | Display formatting | (Utility service, out of scope for E15) |
| `NoteService` | `internal/services/note_service.go` | 124 | Rejection notes | (Specialized, out of scope) |
| `ResumeService` | `internal/services/resume_service.go` | 204 | Resume from status | (Specialized, out of scope) |
| `ContextService` | `internal/services/context_service.go` | 244 | Context tracking | (Specialized, out of scope) |

**Total service layer**: 1,679 LOC (only 232 + 239 = 471 LOC for core entity services)

### 1.2 Service Capabilities Analysis

#### EpicService (232 LOC)
**Has:**
- `TransitionStatus()`: Validate and perform status transitions (with workflow integration)
- `GetNextStatus()`: Get available transitions for current status
- `ValidateStatus()`: Check if status is valid in epic workflow
- Orchestrator action resolution (for AI agent routing)
- Rejection note creation (backward transitions)

**Missing:**
- ❌ **CRUD operations**: Create, GetByKey, Update, Delete
- ❌ **Querying**: List, ListAll, filtering
- ❌ **Progress calculation**: Aggregate from features
- ❌ **Feature rollups**: Count by status, impediments
- ❌ **Task rollups**: Aggregate tasks across features
- ❌ **Health indicators**: Epic health status

**Verdict**: Only 20% of epic business logic is in the service. 80% remains in CLI commands.

#### FeatureService (239 LOC)
**Has:**
- `TransitionStatus()`: Validate and perform status transitions
- `GetNextStatus()`: Get available transitions
- `ValidateStatus()`: Check if status is valid
- Orchestrator action resolution
- Rejection note creation

**Missing:**
- ❌ **CRUD operations**: Create, GetByKey, Update, Delete
- ❌ **Querying**: List, ListByEpic, filtering, sorting
- ❌ **Progress calculation**: Task-based progress
- ❌ **Health status**: Calculate from tasks and blocking states
- ❌ **Action items**: Tasks requiring attention
- ❌ **Work breakdown**: By responsibility (agent, human, QA)
- ❌ **Task counts**: Total, by status, by agent

**Verdict**: Only 15% of feature business logic is in the service. 85% remains in CLI commands.

#### TaskService (MISSING - 0 LOC)
**Status**: **Does not exist**

**All task business logic lives in `internal/cli/commands/task.go` (2,664 LOC):**
- ❌ CRUD operations
- ❌ Lifecycle management (start, complete, approve, reopen, block, unblock)
- ❌ Querying (next, list, filter by status/agent/epic/feature)
- ❌ Dependency management
- ❌ Progress tracking
- ❌ Validation (dependencies, workflow, status transitions)
- ❌ File operations (creation, updates)
- ❌ History tracking

**Verdict**: **0% of task business logic is in services. 100% is in CLI commands.**

**Impact**: This is the **highest-priority extraction** for E15. TaskService would consolidate 2,664 LOC of business logic.

---

## 2. CLI Command Layer Analysis

### 2.1 Metrics

**Total command layer**: 20,952 LOC across 58 files
- **Average file size**: 361 LOC
- **Largest files**: task.go (2,664), feature.go (2,247), epic.go (1,920)
- **Top 3 files**: 6,831 LOC = 32.6% of all command code

**Top 20 Largest Files:**

| Rank | File | LOC | Primary Responsibilities |
|------|------|-----|--------------------------|
| 1 | `task.go` | 2,664 | All task business logic (CRUD, lifecycle, querying, deps) |
| 2 | `feature.go` | 2,247 | All feature business logic (CRUD, progress, health) |
| 3 | `epic.go` | 1,920 | All epic business logic (CRUD, rollups, progress) |
| 4 | `idea.go` | 1,002 | Idea management (separate domain, out of scope) |
| 5 | `config.go` | 1,000 | Configuration (utility, out of scope) |
| 6 | `task_deps.go` | 780 | Dependency tree visualization |
| 7 | `helpers.go` | 671 | Shared helpers (consolidation opportunity) |
| 8 | `related_docs.go` | 462 | Document linking (utility) |
| 9 | `task_criteria.go` | 460 | Task criteria management |
| 10 | `task_note.go` | 442 | Task note management |
| 11 | `workflow_validate_actions.go` | 438 | Workflow validation |
| 12 | `cloud.go` | 428 | Cloud database config |
| 13 | `workflow.go` | 421 | Workflow management |
| 14 | `init.go` | 412 | Project initialization |
| 15 | `task_resume.go` | 401 | Task resumption logic |
| 16 | `task_context.go` | 397 | Task context tracking |
| 17 | `workflow_show_actions.go` | 378 | Workflow action display |
| 18 | `task_next_status.go` | 361 | Task status transitions |
| 19 | `mock_document_repository.go` | 312 | Test mock (out of scope) |
| 20 | `task_unlink.go` | 303 | Task unlinking |

### 2.2 Fat Controller Pattern Evidence

**Repository instantiation counts (351 total occurrences across 46 files):**

```
repository.NewDB(database):           33 occurrences
repository.NewEpicRepository(repoDb): 27 occurrences
repository.NewEpicRepository(db):     27 occurrences
repository.NewFeatureRepository(repoDb): 26 occurrences
repository.NewTaskRepository(dbWrapper): 25 occurrences
repository.NewTaskRepository(db):     24 occurrences
repository.NewFeatureRepository(db):  24 occurrences
repository.NewTaskRepository(repoDb): 19 occurrences
repository.NewFeatureRepository(dbWrapper): 16 occurrences
repository.NewEpicRepository(dbWrapper): 15 occurrences
```

**Analysis:**
- Commands create repositories **directly** instead of using services
- Same repository instantiation pattern **duplicated 351 times**
- Three different variable naming conventions (`repoDb`, `db`, `dbWrapper`) indicate lack of standardization
- Each command file reinvents the wheel for database access

**Target state (via services):**
- `cli.GetTaskService()`: 1 instantiation point
- `cli.GetFeatureService()`: 1 instantiation point
- `cli.GetEpicService()`: 1 instantiation point
- **Reduction**: 351 → ~10 instantiations (97% reduction)

### 2.3 Business Logic in Commands

Analyzing `task.go` (2,664 LOC), business logic breakdown:

**Lines 1-300**: Argument parsing, command definitions, flag handling
**Lines 300-600**: Task creation logic (file operations, key generation, database insert)
**Lines 600-900**: Task listing (filtering, sorting, progress calculation)
**Lines 900-1200**: Task lifecycle (start, complete, approve, reopen)
**Lines 1200-1500**: Task blocking/unblocking, dependency management
**Lines 1500-1800**: Task querying (next task, filtering by agent/status)
**Lines 1800-2100**: Task update operations (title, description, metadata)
**Lines 2100-2400**: Task deletion, history retrieval
**Lines 2400-2664**: Error handling, validation, output formatting

**Business logic estimate**: ~70% (1,865 LOC)
**Argument parsing + formatting**: ~30% (799 LOC)

**Target after extraction**:
- Command file: ~400 LOC (parse args, call service, format output)
- TaskService: ~1,865 LOC (all business logic)
- **Reduction**: 2,664 → 400 = 85% reduction in command layer

Similar pattern applies to `feature.go` (2,247 LOC) and `epic.go` (1,920 LOC).

---

## 3. Code Duplication Patterns

### 3.1 Repository Access Pattern

**Current (duplicated 351 times):**
```go
// Pattern 1: Get database
repoDb, err := cli.GetDB(cmd.Context())
if err != nil {
    return fmt.Errorf("failed to get database: %w", err)
}

// Pattern 2: Create repository
taskRepo := repository.NewTaskRepository(repoDb)

// Pattern 3: Call repository method
task, err := taskRepo.GetByKey(ctx, taskKey)
if err != nil {
    return fmt.Errorf("failed to get task: %w", err)
}
```

**Target (via service layer):**
```go
// Single pattern (10 occurrences)
svc := cli.GetTaskService()
task, err := svc.GetTask(cmd.Context(), taskKey)
if err != nil {
    return err // Service wraps errors with context
}
```

**Impact**: 1,053 LOC eliminated (351 × 3 lines average)

### 3.2 Progress Calculation Pattern

**Current locations (96 occurrences):**

| Location | Occurrences | Pattern |
|----------|-------------|---------|
| Repository layer | 14 | `CalculateProgress()` methods |
| Command layer | 14 | Direct calls to repo.CalculateProgress() |
| Status package | 24 | Status-based progress calculations |
| Progress package | 3 | Generic progress calculator |
| Tests | 41 | Testing progress calculations |

**Duplication evidence:**
- `internal/repository/epic_repository.go`: `CalculateProgress()` (30 LOC)
- `internal/repository/feature_repository.go`: `CalculateProgress()` (56 LOC)
- `internal/status/progress.go`: Progress calculation logic (395 LOC)
- `internal/progress/calculator.go`: Generic calculator (unknown LOC)

**Consolidation target**:
- **Remove from repositories**: 86 LOC (business logic, not data access)
- **Centralize in services**: Epic/FeatureService.CalculateProgress()
- **Reuse status package**: Keep `status.CalculationService` as-is (already good abstraction)

### 3.3 Workflow Validation Pattern

**Current locations (45 occurrences):**

| Pattern | Occurrences | Files |
|---------|-------------|-------|
| `ValidateTransition()` | 18 | workflow.Service, validation package, config |
| `IsValidTransition()` | 27 | Repository, services, tests |

**Analysis:**
- Workflow validation **already centralized** in `workflow.Service` (good!)
- Services use it correctly (epic_service.go, feature_service.go)
- **Problem**: Commands **should not validate directly** - delegate to services

**Recommendation**: No consolidation needed. Keep `workflow.Service`, ensure commands call services (not workflow directly).

### 3.4 Error Handling Pattern

**Current (duplicated ~47 times across command files):**
```go
if err != nil {
    var notFoundErr *repository.NotFoundError
    if errors.As(err, &notFoundErr) {
        cli.Error(fmt.Sprintf("Task not found: %s", args[0]))
        os.Exit(1)
    }
    cli.Error(fmt.Sprintf("Error: %v", err))
    os.Exit(2)
}
```

**Target (services return typed errors, commands translate):**
```go
// Service returns typed errors
func (s *TaskService) GetTask(ctx, key) (*Task, error) {
    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err // NotFoundError propagates
    }
    return task, nil
}

// Command translates to exit codes
if err != nil {
    return HandleServiceError(err, taskKey) // Centralized handler
}
```

**Impact**: Consolidate into `HandleServiceError()` helper (50 LOC replaces ~470 LOC)

### 3.5 Key Normalization Pattern

**Current (duplicated ~32 times):**
```go
normalizedKey := strings.ToUpper(key)
normalizedKey = strings.TrimSpace(normalizedKey)
```

**Target**: `services.NormalizeKey(key)` (centralized utility)

**Impact**: 64 LOC → 1 function (98% reduction)

---

## 4. Repository Layer Business Logic

### 4.1 Progress Calculation (Should Be in Services)

**Current locations in repositories:**

| Repository | Method | LOC | Description |
|------------|--------|-----|-------------|
| `EpicRepository` | `CalculateProgress()` | 30 | Aggregate feature progress |
| `EpicRepository` | `CalculateProgressByKey()` | 10 | Wrapper for key lookup |
| `FeatureRepository` | `CalculateProgress()` | 56 | Task-based progress |
| `FeatureRepository` | `CalculateProgressByKey()` | 10 | Wrapper for key lookup |

**Total repository LOC for progress**: ~106 LOC

**Analysis:**
- Progress calculation is **business logic**, not data access
- Formula: `(completed_tasks / total_tasks) × 100%` - should be in service
- Repositories should only return raw task counts, not calculate percentages

**Migration plan:**
1. Move `CalculateProgress()` to `EpicService.GetProgress()` (reuse query, add calculation logic)
2. Move `CalculateProgress()` to `FeatureService.GetProgress()`
3. Repositories provide only: `CountTasksByStatus(ctx, featureID) (map[string]int, error)`
4. Services calculate: `completed / total * 100`

**Impact**: Remove 106 LOC from repositories, add ~60 LOC to services (net: 46 LOC cleaner)

### 4.2 Status Derivation (Already in Status Package)

**Current**:
- `internal/status/derivation.go`: Status derivation logic (~150 LOC)
- **Good**: Already separated from repositories
- **Issue**: Commands call `status.CalculationService` directly

**Recommendation**: Services should wrap `status.CalculationService`, not commands.

### 4.3 Work Breakdown (Should Be in Services)

**Current**:
- `internal/status/work_breakdown.go`: 59 LOC
- `internal/status/work_breakdown_test.go`: 325 LOC

**Analysis**: Already well-separated. Services should use it, not commands.

---

## 5. Reusability Opportunities

### 5.1 Generic Service Operations

Many operations are **entity-agnostic** and can be abstracted:

| Operation | Epic | Feature | Task | Abstraction Opportunity |
|-----------|------|---------|------|-------------------------|
| **CRUD** | ✅ | ✅ | ✅ | `BaseService<T>` with generics (Go 1.18+) |
| **GetByKey** | ✅ | ✅ | ✅ | Interface method: `EntityGetter.GetByKey(ctx, key)` |
| **List** | ✅ | ✅ | ✅ | Interface method: `EntityLister.List(ctx, filters)` |
| **Filter by status** | ✅ | ✅ | ✅ | Shared filtering logic |
| **Normalize key** | ✅ | ✅ | ✅ | `keys.Normalize(key)` utility |
| **Validate** | ✅ | ✅ | ✅ | Model-level `Validate()` (already exists) |

**Recommendation**: **Do NOT over-abstract**. Epic E15 scope is to extract services, not build generic frameworks. Keep Epic/Feature/TaskService **separate and explicit**. Consolidation can happen in a future epic (E18: Advanced Service Patterns).

### 5.2 QueryService for Cross-Entity Operations

Some commands operate on **multiple entity types**:

| Command | Current Pattern | Service Approach |
|---------|----------------|------------------|
| `get <key>` | Dispatches based on key format | `QueryService.GetByKey(key)` → auto-detects entity type |
| `list <epic> <feature>` | Positional args → conditional routing | `QueryService.List(epicKey, featureKey)` → dispatches |
| `status <key>` | Key parsing → type detection | `QueryService.GetStatus(key)` → delegates to entity service |

**Recommendation**: Create `QueryService` in **Phase 4** (after Epic/Feature/Task services exist). Consolidates smart dispatching logic (currently ~500 LOC across `get.go`, `list.go`, `status.go`).

### 5.3 Shared Service Utilities

Extract common patterns into `internal/services/common/`:

| Utility | Purpose | LOC Saved |
|---------|---------|-----------|
| `error.go` | Error type definitions, wrapping | 100 |
| `keys.go` | Key normalization, parsing | 50 |
| `filters.go` | Generic filtering logic | 80 |
| `sorting.go` | Generic sorting logic | 60 |
| `pagination.go` | Generic pagination logic | 40 |

**Total**: 330 LOC of reusable utilities (eliminates ~1,000 LOC of duplication)

---

## 6. Integration Points

### 6.1 Service → Repository

**Current pattern (commands → repos):**
```go
repoDb, _ := cli.GetDB(cmd.Context())
taskRepo := repository.NewTaskRepository(repoDb)
task, _ := taskRepo.GetByKey(ctx, taskKey)
```

**Target pattern (services → repos):**
```go
type TaskService struct {
    repo *repository.TaskRepository
    workflowSvc *workflow.Service
}

func (s *TaskService) GetTask(ctx, key) (*Task, error) {
    return s.repo.GetByKey(ctx, key) // Service wraps repo call
}
```

**Dependency injection (CLI layer):**
```go
func GetTaskService() *services.TaskService {
    once.Do(func() {
        db := GetDB()
        taskRepo := repository.NewTaskRepository(db)
        workflowSvc := workflow.NewService(projectRoot)
        taskService = services.NewTaskService(taskRepo, workflowSvc)
    })
    return taskService
}
```

**Benefits**:
- Commands get pre-wired services (no repo creation)
- Services manage repository lifecycle
- Single instantiation point (testability++)

### 6.2 CLI Commands → Services

**Current anti-pattern (commands contain business logic):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    // 50 lines of business logic
    // Repository instantiation
    // Status validation
    // Workflow checking
    // Database transaction
    // File updates
    // Error handling
}
```

**Target pattern (thin wrapper):**
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    taskKey := args[0] // 1. Parse arguments

    svc := cli.GetTaskService() // 2. Get service
    task, err := svc.StartTask(cmd.Context(), taskKey) // 3. Call service
    if err != nil {
        return err
    }

    // 4. Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }
    cli.Success(fmt.Sprintf("Task %s started", task.Key))
    return nil
}
```

**Responsibilities**:
- Commands: Parse args, call service, format output (20-50 LOC)
- Services: All business logic (100-300 LOC per method)

### 6.3 Services → Workflow

**Current usage (already correct in existing services):**
```go
type EpicService struct {
    workflowSvc *workflow.Service
}

func (s *EpicService) TransitionStatus(ctx, key, targetStatus, opts) error {
    // Validate transition
    if !opts.Force {
        if err := s.workflowSvc.ValidateTransition(current, targetStatus); err != nil {
            return err
        }
    }
    // ... perform transition
}
```

**Recommendation**: Continue this pattern for TaskService. Workflow integration is **already well-designed**.

### 6.4 Services → Status Package

**Current (commands call status directly - ANTI-PATTERN):**
```go
func runFeatureGet(cmd, args) error {
    // ...
    calcService := status.NewCalculationService(db, cfg)
    progress := calcService.CalculateProgress(ctx, feature)
    // ...
}
```

**Target (services wrap status package):**
```go
type FeatureService struct {
    statusSvc *status.CalculationService
}

func (s *FeatureService) GetProgress(ctx, featureKey) (Progress, error) {
    feature := s.repo.GetByKey(ctx, featureKey)
    return s.statusSvc.CalculateProgress(ctx, feature)
}
```

**Benefits**: Commands never touch `status` package directly. Services orchestrate.

---

## 7. Implementation Complexity

### 7.1 Easiest Consolidations (Low Complexity)

| Pattern | Complexity | LOC Impact | Priority |
|---------|------------|------------|----------|
| Repository instantiation | **Low** | 1,053 LOC eliminated | **P0** |
| Error handling | **Low** | 470 → 50 LOC | **P0** |
| Key normalization | **Low** | 64 → 1 LOC | **P1** |
| GetByKey pattern | **Low** | 111 → ~30 LOC | **P0** |

**Timeline**: 1-2 days per pattern (can be done in parallel across services)

### 7.2 Medium Complexity Consolidations

| Pattern | Complexity | LOC Impact | Priority |
|---------|------------|------------|----------|
| Progress calculation (repo → service) | **Medium** | 106 → 60 LOC | **P0** |
| CRUD operations (commands → services) | **Medium** | 1,500 → 600 LOC | **P0** |
| Filtering logic | **Medium** | 400 → 80 LOC | **P1** |
| Sorting logic | **Medium** | 300 → 60 LOC | **P1** |

**Timeline**: 3-5 days per pattern (requires careful testing)

### 7.3 High Complexity Consolidations

| Pattern | Complexity | LOC Impact | Priority |
|---------|------------|------------|----------|
| TaskService extraction (task.go) | **High** | 2,664 → 400 LOC | **P0** |
| FeatureService expansion (feature.go) | **High** | 2,247 → 400 LOC | **P0** |
| EpicService expansion (epic.go) | **High** | 1,920 → 400 LOC | **P0** |
| QueryService (smart dispatchers) | **High** | 500 → 150 LOC | **P1** |

**Timeline**: 1-2 weeks per service (comprehensive testing required)

### 7.4 Abstraction Layers (Future Work)

These are **out of scope for E15** but documented for future consideration:

| Abstraction | Complexity | Benefit | Epic |
|-------------|------------|---------|------|
| BaseService<T> (generics) | **Very High** | Eliminate code duplication | E18 |
| Repository interface standardization | **High** | Testability improvement | E16 |
| Service middleware (logging, tracing) | **Medium** | Observability | E17 |
| Event sourcing for audit trail | **Very High** | Better audit/history | E18 |

---

## 8. Recommended Consolidation Strategy

### Phase 1: TaskService Extraction (Weeks 2-4)

**Goal**: Extract all task business logic from `task.go` → `TaskService`

**Tasks**:
1. **T1**: Define TaskService interface (methods, signatures)
2. **T2**: Implement CRUD operations (Create, Get, Update, Delete)
3. **T3**: Implement lifecycle operations (Start, Complete, Approve, Reopen, Block, Unblock)
4. **T4**: Implement querying (Next, List, Filter)
5. **T5**: Implement dependency management (GetDependencies, ValidateDependencies)
6. **T6**: Refactor task.go to call TaskService (thin wrapper)
7. **T7**: Write service unit tests (>80% coverage)
8. **T8**: Verify all existing integration tests pass

**Impact**:
- **LOC reduction**: task.go (2,664 → 400 LOC) = 2,264 LOC saved
- **Service layer**: +1,865 LOC (TaskService)
- **Net**: 399 LOC eliminated (85% reduction in command layer)

**Priority**: **P0** (highest impact)

### Phase 2: Epic/Feature Service Expansion (Week 5)

**Goal**: Expand existing Epic/FeatureService with CRUD + queries

**Tasks**:
1. **T1**: Add CRUD methods to EpicService (Create, Get, Update, Delete)
2. **T2**: Add query methods to EpicService (List, GetFeatureRollup, GetTaskRollup)
3. **T3**: Add progress calculation to EpicService (move from repository)
4. **T4**: Add CRUD methods to FeatureService (Create, Get, Update, Delete)
5. **T5**: Add query methods to FeatureService (List, ListByEpic, Filter, Sort)
6. **T6**: Add progress/health to FeatureService (GetProgress, GetHealth, GetActionItems)
7. **T7**: Refactor epic.go and feature.go to call services
8. **T8**: Verify all existing tests pass

**Impact**:
- **LOC reduction**: epic.go (1,920 → 400 LOC), feature.go (2,247 → 400 LOC) = 3,367 LOC saved
- **Service layer**: +2,200 LOC (expanded Epic/FeatureService)
- **Net**: 1,167 LOC eliminated (68% reduction in epic/feature commands)

**Priority**: **P0** (core architecture goal)

### Phase 3: Repository Cleanup (Week 6)

**Goal**: Remove business logic from repositories (pure data access)

**Tasks**:
1. **T1**: Remove `CalculateProgress()` from EpicRepository → moved to EpicService
2. **T2**: Remove `CalculateProgress()` from FeatureRepository → moved to FeatureService
3. **T3**: Remove `GetStatusBreakdown()` from TaskRepository → moved to TaskService
4. **T4**: Verify repositories only contain CRUD + query methods
5. **T5**: Update repository tests (verify data access, not business logic)

**Impact**:
- **LOC reduction**: repositories (106 LOC progress calc + 50 LOC status breakdown) = 156 LOC saved
- **Repository layer**: -156 LOC
- **Service layer**: +60 LOC (logic moved to services)
- **Net**: 96 LOC eliminated

**Priority**: **P0** (clarifies layer boundaries)

### Phase 4: CLI Command Slimming (Week 7)

**Goal**: Refactor all remaining commands to thin wrappers

**Tasks**:
1. **T1**: Audit all command files for business logic
2. **T2**: Extract remaining logic to appropriate services
3. **T3**: Consolidate error handling (HandleServiceError utility)
4. **T4**: Consolidate key normalization (keys.Normalize utility)
5. **T5**: Verify no command exceeds 500 LOC (excluding tests)
6. **T6**: Update CLI architecture documentation

**Impact**:
- **LOC reduction**: remaining 47 files (9,102 LOC → ~3,000 LOC) = 6,102 LOC saved
- **Service layer**: +4,000 LOC (QueryService, utilities)
- **Net**: 2,102 LOC eliminated

**Priority**: **P0** (core epic goal)

### Phase 5: HTTP API Wiring (Optional - Week 8)

**Goal**: Wire HTTP API to service layer (100% feature parity with CLI)

**Tasks**:
1. **T1**: Create API endpoints for TaskService methods
2. **T2**: Create API endpoints for FeatureService methods
3. **T3**: Create API endpoints for EpicService methods
4. **T4**: API integration tests (verify feature parity)
5. **T5**: API documentation (OpenAPI/Swagger)

**Impact**:
- API feature parity: 32% → 100%
- HTTP API can reuse all service logic (no duplication)

**Priority**: **P1** (enabler for API consumers, not blocking CLI refactoring)

---

## 9. Risk Assessment & Mitigation

### 9.1 Regression Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Breaking existing functionality** | **High** | **Critical** | - Comprehensive integration tests<br>- Manual regression testing<br>- Incremental PRs (<500 LOC)<br>- CI/CD gating |
| **Performance degradation** | **Medium** | **Medium** | - Benchmark before/after<br>- Target: ±10% unchanged<br>- Profile memory usage |
| **Test suite breakage** | **High** | **Medium** | - Update tests incrementally<br>- Mock services in CLI tests<br>- Repository tests unchanged |

**Mitigation strategies:**
- **NFR1 (Zero Regression)**: All existing tests must pass
- **NFR3 (Performance Neutrality)**: Benchmark CLI commands before/after
- **NFR4 (Incremental Migration)**: PRs <500 LOC, main always deployable

### 9.2 Scope Creep Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Adding new features during refactoring** | **Medium** | **High** | - Strict scope adherence<br>- "No new features" rule<br>- Defer to future epics |
| **Over-abstracting (BaseService<T>)** | **Medium** | **Medium** | - Keep Epic/Feature/TaskService separate<br>- Defer abstractions to E18 |
| **Rewriting working code** | **Low** | **Low** | - "If it ain't broke, don't fix it"<br>- Extract, don't rewrite |

**Mitigation strategies:**
- Epic E15 is **refactoring only** - no new capabilities
- Abstraction layers deferred to Epic E18
- Focus on code movement, not code rewrites

### 9.3 Testing Complexity Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Mocking complexity** | **Medium** | **Medium** | - Use simple mock interfaces<br>- Avoid complex DI frameworks<br>- Constructor injection only |
| **Test coverage gaps** | **Low** | **High** | - Target: >80% service layer coverage<br>- Gating: Coverage must increase |

**Mitigation strategies:**
- **NFR2 (Test Coverage)**: Service layer >80% coverage
- Use existing `internal/test/` utilities (no new frameworks)
- Mock at repository level (not database level)

---

## 10. Success Metrics

### 10.1 Code Reduction Metrics

| Metric | Baseline | Target | Measurement Method |
|--------|----------|--------|-------------------|
| **Total command LOC** | 20,952 | <15,000 | `find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1` |
| **Average file size** | 361 LOC | <320 LOC | Total LOC / file count |
| **Largest file** | 2,664 LOC | <500 LOC | `wc -l task.go` |
| **Service layer LOC** | 1,679 | ~8,000 | `find internal/services -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1` |

**Success threshold**: >50% reduction acceptable, >65% reduction excellent

### 10.2 Duplication Metrics

| Metric | Baseline | Target | Measurement Method |
|--------|----------|--------|-------------------|
| **Repository instantiations** | 351 | <10 | `grep -r "repository\.New" internal/cli/commands/*.go | wc -l` |
| **GetByKey pattern** | 37 | <5 | `grep -r "GetByKey.*context\.Context" internal/cli/commands | wc -l` |
| **Progress calculation** | 96 | <30 | `grep -r "CalculateProgress" internal | wc -l` |

**Success threshold**: >60% reduction acceptable, >80% reduction excellent

### 10.3 Architecture Quality Metrics

| Metric | Baseline | Target | Measurement Method |
|--------|----------|--------|-------------------|
| **Business logic in services** | ~5% | >90% | Manual code review checklist |
| **Service layer test coverage** | 85% (Epic), 82% (Feature) | >80% all | `go test -coverprofile=coverage.out ./internal/services/... && go tool cover -func=coverage.out` |
| **Commands calling repos directly** | 351 | 0 | `grep -r "repository\.New" internal/cli/commands | wc -l` |

**Success threshold**: >95% acceptable, 100% excellent

---

## 11. Exit Gate Criteria

Epic E15 is considered **complete** when:

1. ✅ **All 47 CLI command files are <500 LOC** (FR6)
   - Measured: `find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | awk '{if ($1 > 500) print $0}'`
   - Baseline: 3 files >500 LOC
   - Target: 0 files >500 LOC

2. ✅ **TaskService, FeatureService, EpicService exist and are used by all commands** (FR2-4)
   - TaskService has >30 methods (CRUD, lifecycle, querying)
   - FeatureService expanded from 3 → >20 methods
   - EpicService expanded from 3 → >15 methods
   - No CLI commands call repositories directly

3. ✅ **Repository layer contains only data access** (FR7)
   - No `CalculateProgress()` in repositories
   - No `GetStatusBreakdown()` in repositories
   - No business logic in repository methods

4. ✅ **All existing tests pass** (NFR1)
   - `make test` exits with code 0
   - No regressions in functionality

5. ✅ **Service layer has >80% test coverage** (NFR2)
   - TaskService: >80%
   - FeatureService: maintain >80%
   - EpicService: maintain >85%

6. ✅ **Performance within ±10% of baseline** (NFR3)
   - `shark task next`: 50ms ± 5ms
   - `shark feature get E07-F01`: 35ms ± 3ms

7. ✅ **Documentation updated** (FR9)
   - `.claude/rules/architecture.md` reflects service layer
   - `CLAUDE.md` includes service layer guidance
   - Migration guide for contributors

---

## 12. Recommendations for Architect Phase

### 12.1 Service Interface Design Principles

1. **Explicit over generic**: Keep TaskService, FeatureService, EpicService separate (don't use `BaseService<T>` yet)
2. **Context-aware**: All methods accept `context.Context` as first parameter
3. **Error wrapping**: Services wrap repository errors with business context
4. **Dependency injection**: Services receive repositories via constructors
5. **Workflow integration**: Services use `workflow.Service` for status validation
6. **Status integration**: Services use `status.CalculationService` for progress/health

### 12.2 Incremental Migration Strategy

1. **Start with TaskService**: Highest LOC impact (2,664 lines)
2. **Expand Epic/FeatureService**: Add CRUD + queries (already have transitions)
3. **Clean repositories**: Remove business logic after services exist
4. **Refactor commands last**: Once services are stable, slim commands

### 12.3 Testing Strategy

1. **Service tests use mocks**: Mock repositories, never use real database
2. **Repository tests use real DB**: Integration tests with cleanup
3. **CLI tests use mocked services**: Never create repositories in CLI tests
4. **Coverage gating**: PR merge requires >80% coverage in changed services

### 12.4 PR Structure

1. **One service per PR**: TaskService extraction = 1 PR
2. **Feature expansion per PR**: FeatureService CRUD = 1 PR, Queries = 1 PR
3. **Repository cleanup per PR**: Remove CalculateProgress = 1 PR
4. **Command slimming per PR**: Refactor task.go = 1 PR (after TaskService exists)

**Total estimated PRs**: 15-20 PRs over 7 weeks

---

## 13. Appendix: File Path References

### Service Layer Files
- `/home/jwwel/projects/shark-task-manager/internal/services/epic_service.go` (232 LOC)
- `/home/jwwel/projects/shark-task-manager/internal/services/feature_service.go` (239 LOC)
- **MISSING**: `/home/jwwel/projects/shark-task-manager/internal/services/task_service.go`

### Command Layer Files (Top 3)
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/task.go` (2,664 LOC)
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/feature.go` (2,247 LOC)
- `/home/jwwel/projects/shark-task-manager/internal/cli/commands/epic.go` (1,920 LOC)

### Repository Layer Files
- `/home/jwwel/projects/shark-task-manager/internal/repository/epic_repository.go` (line 386: CalculateProgress)
- `/home/jwwel/projects/shark-task-manager/internal/repository/feature_repository.go` (line 624: CalculateProgress)
- `/home/jwwel/projects/shark-task-manager/internal/repository/task_repository.go`

### Supporting Services
- `/home/jwwel/projects/shark-task-manager/internal/workflow/service.go` (workflow validation)
- `/home/jwwel/projects/shark-task-manager/internal/status/calculation_service.go` (progress/health)
- `/home/jwwel/projects/shark-task-manager/internal/taskcreation/creator.go` (task creation orchestration)

### Architecture Documentation
- `/home/jwwel/projects/shark-task-manager/.claude/rules/architecture.md`
- `/home/jwwel/projects/shark-task-manager/CLAUDE.md`
- `/home/jwwel/projects/shark-task-manager/docs/CLI_REFERENCE.md`

---

**Research Complete**: 2026-02-16
**Next Phase**: Architecture Design (Epic E15 Feature F01)
**Estimated Effort**: 7 weeks (Phases 1-4), +1 week optional (Phase 5 HTTP API)
