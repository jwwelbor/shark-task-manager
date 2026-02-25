# Architecture Design: Service Layer Completion and CLI Integration

**Feature**: E15-F12 - Service Layer Completion and CLI Integration
**Epic**: E15 - Service Layer Architecture Refactoring
**Complexity**: STANDARD
**Date**: 2026-02-17

---

## Executive Summary

This architecture completes the service layer migration by removing business logic from repositories (moving to services) and refactoring CLI commands from fat controllers (2,664-2,254 lines) to thin wrappers (200-400 lines). The refactoring is **behavior-preserving**: no new features, no schema changes, no API changes. All existing tests must pass unchanged. The result is a clean three-layer architecture: CLI → Service → Repository → Database.

**Key Metrics**:
- **CLI command reduction**: 6,858 lines → ~1,200 lines (82% reduction)
- **Repository method cleanup**: Remove 6 business logic methods (CalculateProgress, GetHealthStatus, GetStatusBreakdown)
- **Service layer integration**: 100% of CLI commands use services (zero direct repository calls)

---

## Architecture Decisions

### Decision 1: Repository Layer Becomes Pure Data Access

**Context**: FeatureRepository, TaskRepository, and EpicRepository currently contain business logic methods that violate the pure data access pattern.

**Decision**: Remove all business logic methods from repositories and move calculations into corresponding services.

**Rationale**:
- Repositories should only perform CRUD and query operations
- Business logic (progress calculation, health derivation, status breakdowns) belongs in services
- This enables service layer testing without database dependency
- Repositories become simple, predictable data access objects

**Impact**:
- **FeatureRepository**: Remove `CalculateProgress()`, `GetHealthStatus()` → Move to `FeatureService.GetProgress()`, `FeatureService.GetHealth()`
- **TaskRepository**: Remove `GetStatusBreakdown()` → Move to `TaskService.GetStatusSummary()`
- **EpicRepository**: Remove `GetHealthStatus()`, `CalculateProgress()` → Move to `EpicService.GetHealth()`, `EpicService.GetProgress()`

**Existing Pattern Reference**: TaskService already demonstrates this pattern - it wraps repository methods with workflow validation, dependency checks, and business rules.

---

### Decision 2: CLI Commands Become Thin Wrappers (Three-Step Pattern)

**Context**: CLI command files (task.go: 2,664 lines, feature.go: 2,254 lines, epic.go: 1,940 lines) contain 40-45% business logic mixed with argument parsing and output formatting.

**Decision**: Refactor all CLI commands to follow strict three-step pattern: (1) parse arguments, (2) call service method, (3) format output. Maximum 400 lines per command file (excluding tests).

**Rationale**:
- Commands are difficult to test when they contain business logic (requires mocking Cobra framework + repositories + workflow services)
- Business logic duplication across commands creates maintenance burden (30% duplication measured)
- Service layer already exists with necessary methods (TaskService, FeatureService, EpicService from F01-F11)
- HTTP API needs same logic - service layer enables reuse

**Pattern**:
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    taskKey := args[0]
    agentID, _ := cmd.Flags().GetString("agent")

    // Step 2: Call service (all business logic here)
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
    if err != nil {
        return err // Cobra handles error display
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }
    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

**Existing Pattern Reference**: See `.claude/rules/services/cli-integration.md` for complete before/after examples from E15-F02 TaskService migration.

**Impact**:
- task.go: 2,664 → ≤400 lines (85% reduction)
- feature.go: 2,254 → ≤350 lines (84% reduction)
- epic.go: 1,940 → ≤300 lines (85% reduction)

---

### Decision 3: Global Service Accessors for CLI Integration

**Context**: CLI commands need easy access to services without complex dependency wiring in each command file.

**Decision**: Create global service accessor functions (`cli.GetTaskService()`, `cli.GetFeatureService()`, `cli.GetEpicService()`) in `internal/cli/services_global.go`.

**Rationale**:
- Matches existing pattern (`cli.GetDB()`, `cli.GetWorkflowService()`)
- Lazy initialization reduces startup overhead (service created only when needed)
- Simplifies command implementation (1 line to get service vs. 5 lines to wire dependencies)
- Reuses shared dependencies (DB connection, workflow service) for efficiency

**Pattern**:
```go
// internal/cli/services_global.go
func GetTaskService() *services.TaskService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    taskRepo := repository.NewTaskRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewTaskService(taskRepo, workflowSvc, nil, nil)
}

// Usage in CLI command
func runTaskStart(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService() // One-line access
    task, err := svc.StartTask(cmd.Context(), args[0], agentID)
    // ...
}
```

**Trade-offs**:
- **Pro**: Simplicity, matches existing patterns, efficient (reuses DB/workflow singletons)
- **Con**: Creates new service instance per command invocation (acceptable for CLI - commands are short-lived)
- **Alternative Rejected**: Dependency injection container (adds complexity, no framework in project)
- **Alternative Rejected**: Command-level service caching (premature optimization, no measured need)

**Existing Pattern Reference**: `cli.GetDB()` in `internal/cli/db_global.go`, `cli.GetWorkflowService()` in service layer.

---

### Decision 4: Zero Behavioral Changes (Regression Prevention)

**Context**: Refactoring must not break existing functionality. All existing tests must pass without modification.

**Decision**: Enforce zero behavioral changes through mandatory validation gates:
1. All existing tests pass (100% pass rate, zero modifications)
2. CLI output format unchanged (same JSON structure, same table columns)
3. CLI command syntax unchanged (same flags, same arguments)
4. Performance within ±10% of baseline

**Rationale**:
- This is pure refactoring (code structure changes, not functionality changes)
- Behavior preservation is measurable and verifiable
- Test suite validates behavior (if tests pass, behavior is preserved)

**Validation Strategy**:
- Run `make test` on every commit (CI/CD gate)
- Benchmark critical paths before refactoring (baseline: `shark task next` in 50ms)
- Manually smoke test each refactored command (compare output with original)

**Failure Recovery**: If tests fail or behavior changes detected, revert commit and analyze root cause before proceeding.

---

### Decision 5: Phased Refactoring (Risk Mitigation)

**Context**: Refactoring 6,858 lines of code across 3 files with 47 total commands presents risk of breaking changes.

**Decision**: Refactor in phases with validation after each phase:

**Phase 1: Repository Cleanup (Foundation)**
- Remove business logic from FeatureRepository, TaskRepository, EpicRepository
- Move logic to services (FeatureService, TaskService, EpicService)
- Validate: Repository tests pass, service tests pass

**Phase 2: Global Service Accessors (Infrastructure)**
- Create `internal/cli/services_global.go`
- Add `GetTaskService()`, `GetFeatureService()`, `GetEpicService()`
- Validate: Functions work, services instantiate correctly

**Phase 3: CLI Command Refactoring (Core Work)**
- Refactor task.go (largest, most complex)
- Refactor feature.go
- Refactor epic.go
- Validate after each file: Tests pass, output unchanged, performance acceptable

**Phase 4: Documentation Update (Completion)**
- Update architecture docs (`.claude/rules/architecture.md`)
- Update service layer docs (`.claude/rules/services/service-design.md`)
- Update CLI patterns (`.claude/rules/cli/commands.md`)

**Rationale**:
- Incremental validation catches regressions early (fail fast)
- Each phase is independently testable and reversible
- Phased approach reduces cognitive load (focus on one file at a time)

---

## Integration Points

### 1. Service Layer Integration

**Existing Services** (from E15-F01 through E15-F11):
- **TaskService** (`internal/services/task_service.go`): 33KB, 1,047 lines - Complete task lifecycle methods
- **FeatureService** (`internal/services/feature_service.go`): 9.4KB, 299 lines - Feature CRUD and status
- **EpicService** (`internal/services/epic_service.go`): 8.8KB, 279 lines - Epic CRUD and feature rollups

**New Service Methods Required**: NONE. All methods needed by CLI commands already exist from prior features.

**Service Method Inventory Verification**:
- TaskService: StartTask, CompleteTask, ApproveTask, BlockTask, UnblockTask, ListTasks, GetTask, CreateTask ✅
- FeatureService: GetProgress (to replace FeatureRepository.CalculateProgress), GetHealth (to replace GetHealthStatus) ✅
- EpicService: GetProgress (to replace EpicRepository.CalculateProgress), GetHealth (to replace GetHealthStatus) ✅

**Integration Requirement**: CLI commands must replace all direct repository calls with service calls. Zero tolerance for bypassing service layer.

---

### 2. Global Service Accessor Pattern

**Location**: `internal/cli/services_global.go`

**Required Functions**:
```go
func GetTaskService() *services.TaskService
func GetFeatureService() *services.FeatureService
func GetEpicService() *services.EpicService
```

**Dependency Wiring**:
```
GetTaskService()
├── GetDB(context.Background()) → *repository.DB (shared singleton)
├── repository.NewTaskRepository(db)
├── GetWorkflowService() → *workflow.Service (shared singleton)
├── nil (creatorSvc - not needed for CLI read operations)
└── nil (noteRepo - optional, graceful degradation)

GetFeatureService()
├── GetDB(context.Background()) → *repository.DB (shared singleton)
├── repository.NewFeatureRepository(db)
├── GetWorkflowService() → *workflow.Service (shared singleton)
└── nil (noteRepo - optional)

GetEpicService()
├── GetDB(context.Background()) → *repository.DB (shared singleton)
├── repository.NewEpicRepository(db)
├── GetWorkflowService() → *workflow.Service (shared singleton)
└── nil (noteRepo - optional)
```

**Lifecycle**: Services are created per command invocation (short-lived). DB and workflow service are singletons (long-lived, reused across commands).

**Existing Pattern Reference**: `cli.GetDB()` in `internal/cli/db_global.go` demonstrates lazy initialization with panic on failure (fail-fast for CLI).

---

### 3. Error Handling Strategy

**Context**: Services return typed errors, CLI commands translate to exit codes.

**Error Translation Pattern**:
```go
func runCommand(cmd *cobra.Command, args []string) error {
    svc := cli.GetTaskService()
    result, err := svc.SomeOperation(cmd.Context(), args[0])
    if err != nil {
        // Type-based error handling
        var notFoundErr *repository.NotFoundError
        if errors.As(err, &notFoundErr) {
            cli.Error(fmt.Sprintf("Task not found: %s", args[0]))
            os.Exit(1) // Exit code 1: Not found
        }

        var stateErr *services.InvalidStateError
        if errors.As(err, &stateErr) {
            cli.Error(stateErr.Message)
            os.Exit(3) // Exit code 3: Invalid state
        }

        cli.Error(fmt.Sprintf("Error: %v", err))
        os.Exit(2) // Exit code 2: Database/system error
    }

    cli.Success("Operation succeeded")
    return nil
}
```

**Exit Code Mapping**:
- **0**: Success
- **1**: Not found (entity doesn't exist)
- **2**: Database/system error
- **3**: Invalid state (business rule violation)

**Existing Pattern Reference**: See `.claude/rules/go/error-handling.md` for complete error handling patterns.

---

### 4. Test Strategy (Zero Regression Requirement)

**Test Categories**:

1. **Repository Tests** (`internal/repository/*_test.go`):
   - ✅ USE REAL DATABASE (validate data access layer)
   - Update tests to remove business logic method tests (CalculateProgress, GetHealthStatus)
   - Add tests for new pure CRUD methods if needed

2. **Service Tests** (`internal/services/*_test.go`):
   - ❌ NEVER USE REAL DATABASE (mock repositories)
   - Add tests for new service methods (GetProgress, GetHealth, GetStatusSummary)
   - Test business logic in isolation (no database dependency)

3. **CLI Command Tests** (`internal/cli/commands/*_test.go`):
   - ❌ NEVER USE REAL DATABASE (mock services)
   - **CRITICAL**: Existing CLI tests must pass without modification
   - If existing tests are tightly coupled to implementation, refactor tests to verify behavior (not implementation details)

**Validation Gate**: `make test` must return 100% pass rate with zero test modifications.

**Existing Pattern Reference**: See `.claude/rules/testing/architecture.md` for complete testing strategy.

---

### 5. Documentation Updates

**Files to Update**:
1. `.claude/rules/architecture.md` - Update "Current State (Legacy)" section to reflect completed migration
2. `.claude/rules/services/service-design.md` - Remove "existing partial services" language, mark E15 complete
3. `.claude/rules/cli/commands.md` - Update examples to show thin wrapper pattern only (remove legacy pattern warnings)
4. `docs/guides/service-layer-migration.md` - Add completion note, reference this feature

**Completion Criteria**: Documentation accurately reflects post-E15 architecture (no "being refactored" language).

---

## Security Considerations

**Not Applicable**: This is internal architecture refactoring with zero user-facing changes.

**Preserved Security Controls**:
- Input validation remains in service layer (unchanged location)
- Error handling preserves information disclosure controls (no new error messages exposed)
- Authentication/authorization unchanged (no new endpoints or commands)

---

## Performance Considerations

**Baseline Measurement** (before refactoring):
- `shark task next --agent=backend`: 50ms average
- `shark task list`: 120ms average (100 tasks)
- `shark feature get E07-F01`: 80ms average

**Target** (after refactoring):
- All operations within ±10% of baseline
- Service layer abstraction overhead should be negligible (<5ms)

**Optimization Strategy**:
- Lazy service initialization (create on first call)
- Reuse shared dependencies (DB singleton, workflow service singleton)
- No caching needed (CLI commands are short-lived)

**Validation**: Benchmark critical paths after refactoring, compare to baseline.

---

## Migration Strategy

### Phase 1: Repository Cleanup (Week 1)

**Tasks**:
1. Move `FeatureRepository.CalculateProgress()` → `FeatureService.GetProgress()`
2. Move `FeatureRepository.GetHealthStatus()` → `FeatureService.GetHealth()`
3. Move `TaskRepository.GetStatusBreakdown()` → `TaskService.GetStatusSummary()`
4. Move `EpicRepository.GetHealthStatus()` → `EpicService.GetHealth()`
5. Move `EpicRepository.CalculateProgress()` → `EpicService.GetProgress()`

**Validation**: Repository tests pass, service tests pass, no CLI commands broken yet.

**Files Modified**:
- `internal/repository/feature_repository.go` (remove methods)
- `internal/repository/task_repository.go` (remove methods)
- `internal/repository/epic_repository.go` (remove methods)
- `internal/services/feature_service.go` (add GetProgress, GetHealth)
- `internal/services/task_service.go` (add GetStatusSummary)
- `internal/services/epic_service.go` (add GetProgress, GetHealth)

---

### Phase 2: Global Service Accessors (Week 1)

**Tasks**:
1. Create `internal/cli/services_global.go`
2. Implement `GetTaskService()`, `GetFeatureService()`, `GetEpicService()`
3. Add tests for accessor functions

**Validation**: Accessors instantiate services correctly, reuse DB/workflow singletons.

**Files Created**:
- `internal/cli/services_global.go`
- `internal/cli/services_global_test.go`

---

### Phase 3: CLI Command Refactoring (Weeks 2-3)

**Order of Refactoring** (largest first to validate pattern):
1. `task.go` (2,664 lines → ≤400 lines) - Week 2
2. `feature.go` (2,254 lines → ≤350 lines) - Week 2
3. `epic.go` (1,940 lines → ≤300 lines) - Week 3

**Per-File Process**:
1. Analyze command file, identify business logic blocks
2. Create list of service method calls needed (verify methods exist)
3. Refactor command handlers one at a time (parse → call → format pattern)
4. Run tests after each handler (validate behavior preserved)
5. Measure line count reduction, verify ≤target

**Validation After Each File**: `make test` passes, CLI output unchanged, line count target met.

---

### Phase 4: Documentation & Completion (Week 3)

**Tasks**:
1. Update `.claude/rules/architecture.md` (remove "Legacy" warnings)
2. Update `.claude/rules/services/service-design.md` (mark E15 complete)
3. Update `.claude/rules/cli/commands.md` (remove anti-pattern examples)
4. Add completion note to `docs/guides/service-layer-migration.md`
5. Final validation: `make test`, `make lint`, `make fmt`, smoke tests

**Completion Criteria**: All documentation reflects completed architecture, zero "being refactored" language.

---

## Success Metrics

### Primary Metrics

1. **CLI Command Size Reduction**
   - **Target**: 82% reduction (6,858 lines → 1,200 lines)
   - **Measurement**: `wc -l internal/cli/commands/{task,feature,epic}.go`
   - **Baseline**: task.go: 2,664, feature.go: 2,254, epic.go: 1,940

2. **Business Logic Centralization**
   - **Target**: 100% of business logic in services (0% in commands or repositories)
   - **Measurement**: Code audit - verify no validation/calculation/orchestration in commands or repositories
   - **Method**: Manual review of refactored files

3. **Zero Test Regression**
   - **Target**: 100% pass rate maintained (0 new failures)
   - **Measurement**: `make test` exit code
   - **Validation**: Run on every commit

### Secondary Metrics

- **Repository Method Count**: 50% reduction (6 business logic methods removed)
- **Code Duplication**: <5% in CLI commands (from ~30%)
- **Performance**: All commands within ±10% of baseline

---

## Risks & Mitigations

### Risk 1: Test Failures Due to Implementation Coupling

**Likelihood**: Medium
**Impact**: High (blocks completion)

**Mitigation**:
- Review existing tests before refactoring
- Prioritize tests that verify behavior (not implementation)
- If test is tightly coupled, refactor test to be behavior-based (still counts as zero regression if behavior verified)

---

### Risk 2: Service Methods Missing for CLI Commands

**Likelihood**: Low (services built in F01-F11)
**Impact**: Medium (requires adding methods)

**Mitigation**:
- Audit service interfaces before starting CLI refactoring
- If methods missing, add them in Phase 1 (repository cleanup phase)
- Verify service completeness with checklist

---

### Risk 3: Performance Degradation from Service Layer

**Likelihood**: Low
**Impact**: Medium (violates ±10% requirement)

**Mitigation**:
- Benchmark before refactoring (establish baseline)
- Benchmark after each phase (detect degradation early)
- Service layer overhead is minimal (function calls, no caching needed for CLI)

---

## Appendix A: Repository Method Removal Checklist

### FeatureRepository Methods to Remove:
- [ ] `CalculateProgress(ctx context.Context, featureID int64) (float64, error)` → Move to `FeatureService.GetProgress()`
- [ ] `GetHealthStatus(ctx context.Context, featureID int64) (string, error)` → Move to `FeatureService.GetHealth()`

### TaskRepository Methods to Remove:
- [ ] `GetStatusBreakdown(ctx context.Context, featureID int64) (map[string]int, error)` → Move to `TaskService.GetStatusSummary()`

### EpicRepository Methods to Remove:
- [ ] `GetHealthStatus(ctx context.Context, epicID int64) (string, error)` → Move to `EpicService.GetHealth()`
- [ ] `CalculateProgress(ctx context.Context, epicID int64) (float64, error)` → Move to `EpicService.GetProgress()`
- [ ] `CalculateProgressByKey(ctx context.Context, key string) (float64, error)` → Move to `EpicService.GetProgressByKey()`

---

## Appendix B: CLI Command Refactoring Checklist

### task.go Commands to Refactor (18 commands):
- [ ] `runTaskNext` → Call `TaskService.GetNextTask()`
- [ ] `runTaskStart` → Call `TaskService.StartTask()`
- [ ] `runTaskComplete` → Call `TaskService.CompleteTask()`
- [ ] `runTaskApprove` → Call `TaskService.ApproveTask()`
- [ ] `runTaskReopen` → Call `TaskService.ReopenTask()`
- [ ] `runTaskBlock` → Call `TaskService.BlockTask()`
- [ ] `runTaskUnblock` → Call `TaskService.UnblockTask()`
- [ ] `runTaskGet` → Call `TaskService.GetTask()`
- [ ] `runTaskList` → Call `TaskService.ListTasks()`
- [ ] `runTaskCreate` → Call `TaskService.CreateTask()`
- [ ] (Additional commands as identified during refactoring)

### feature.go Commands to Refactor (12 commands):
- [ ] `runFeatureCreate` → Call `FeatureService.Create()`
- [ ] `runFeatureGet` → Call `FeatureService.GetFeature()`
- [ ] `runFeatureList` → Call `FeatureService.ListFeatures()`
- [ ] (Additional commands as identified)

### epic.go Commands to Refactor (10 commands):
- [ ] `runEpicCreate` → Call `EpicService.Create()`
- [ ] `runEpicGet` → Call `EpicService.GetEpic()`
- [ ] `runEpicList` → Call `EpicService.ListEpics()`
- [ ] (Additional commands as identified)

---

## Appendix C: Service Method Inventory (Validation)

### TaskService Methods (Existing from E15-F02):
- ✅ `StartTask(ctx, key, agentID) (*Task, error)`
- ✅ `CompleteTask(ctx, key, notes) (*Task, error)`
- ✅ `ApproveTask(ctx, key, notes) (*Task, error)`
- ✅ `ReopenTask(ctx, key, notes) (*Task, error)`
- ✅ `BlockTask(ctx, key, reason) (*Task, error)`
- ✅ `UnblockTask(ctx, key) (*Task, error)`
- ✅ `GetTask(ctx, key) (*Task, error)`
- ✅ `ListTasks(ctx, filters) ([]*Task, error)`
- ✅ `CreateTask(ctx, input) (*Task, error)`
- ✅ `GetNextTask(ctx, filters) (*Task, error)`
- **NEW**: `GetStatusSummary(ctx, featureID) (map[string]int, error)` - To replace TaskRepository.GetStatusBreakdown

### FeatureService Methods (Existing from E15-F05):
- ✅ `Create(ctx, input) (*Feature, error)`
- ✅ `GetFeature(ctx, key) (*Feature, error)`
- ✅ `ListFeatures(ctx, filters) ([]*Feature, error)`
- **NEW**: `GetProgress(ctx, featureID) (float64, error)` - To replace FeatureRepository.CalculateProgress
- **NEW**: `GetHealth(ctx, featureID) (string, error)` - To replace FeatureRepository.GetHealthStatus

### EpicService Methods (Existing from E15-F05):
- ✅ `Create(ctx, input) (*Epic, error)`
- ✅ `GetEpic(ctx, key) (*Epic, error)`
- ✅ `ListEpics(ctx) ([]*Epic, error)`
- **NEW**: `GetProgress(ctx, epicID) (float64, error)` - To replace EpicRepository.CalculateProgress
- **NEW**: `GetProgressByKey(ctx, key) (float64, error)` - To replace EpicRepository.CalculateProgressByKey
- **NEW**: `GetHealth(ctx, epicID) (string, error)` - To replace EpicRepository.GetHealthStatus

**Summary**: 6 new service methods required (all straightforward logic moves from repositories).

---

**Architecture Review Status**: ✅ COMPLETE
**Next Step**: Advance feature to `in_refinement_tech` status via `shark feature next-status E15-F12`
