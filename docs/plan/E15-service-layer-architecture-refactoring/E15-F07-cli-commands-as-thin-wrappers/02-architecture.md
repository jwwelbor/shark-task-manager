# E15-F07: CLI Commands as Thin Wrappers - Architecture

**Feature**: E15-F07
**Complexity Tier**: STANDARD
**Date**: 2026-02-18
**Status**: In Technical Refinement

---

## 1. Architecture Summary

This feature is a **pure refactoring** with no new user-facing functionality, no schema changes, and no new external integration points. The architecture decision is straightforward: redirect all business logic from three fat CLI command files through the existing service layer, using global service accessors that already exist.

**Core decision**: Apply the three-step thin wrapper pattern (parse → call service → format output) uniformly to all handlers in `task.go`, `feature.go`, and `epic.go`.

The services (`TaskService`, `FeatureService`, `EpicService`) and their global accessors (`cli.GetTaskService()`, `cli.GetFeatureService()`, `cli.GetEpicService()`) already exist with sufficient methods to cover the command handlers being refactored.

---

## 2. Current State Analysis

### File Sizes (Before Refactoring)

| File | Current Lines | Target Lines | Business Logic % |
|------|--------------|--------------|-----------------|
| `internal/cli/commands/task.go` | 2,664 | 300-400 | ~70% |
| `internal/cli/commands/feature.go` | 2,252 | 250-350 | ~65% |
| `internal/cli/commands/epic.go` | 1,938 | 200-300 | ~60% |
| **Total** | **6,854** | **~1,050** | |

### Existing Service Coverage

All service methods required for command refactoring are already implemented:

**TaskService** (`internal/services/task_service.go`):
- `CreateTask`, `GetTask`, `UpdateTask`, `DeleteTask`
- `ListTasks`, `ListTasksWithPagination`, `GetNextTask`
- `StartTask`, `CompleteTask`, `ApproveTask`, `ReopenTask`
- `BlockTask`, `UnblockTask`, `TransitionStatus`
- `ValidateDependencies`, `GetDependencyTree`
- `GetTasksByStatus`, `GetTasksByAgent`, `GetBlockedTasks`

**FeatureService** (`internal/services/feature_service.go`):
- `GetFeature`, `ListFeatures`
- `GetProgress`, `GetHealth`, `GetWorkBreakdown`, `GetActionItems`
- `GetEnrichedTaskStatusBreakdown`, `GetTaskStatusBreakdown`
- `TransitionStatus`, `GetNextStatus`
- `RecalculateAndSetProgress`, `RecalculateAndSetProgressByKey`

**EpicService** (`internal/services/epic_service.go`):
- `GetEpic`, `ListEpics`
- `CalculateProgress`, `GetProgress`
- `GetFeatureRollup`, `GetTaskStatusRollup`
- `GetImpediments`, `GetHealth`
- `TransitionStatus`, `GetNextStatus`

### Known Issues (Prerequisite Fixes)

Two `TODO` comments in the accessor files indicate interface mismatches that must be resolved before the refactoring proceeds:

1. **`service_accessors.go` line 34**: `EpicNoteRepository` interface mismatch
   - EpicService expects: `CreateRejectionNote(ctx, entityType string, entityID int64, historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error`
   - Repository provides: `CreateRejectionNote(ctx, entityType models.EntityType, entityID int64, historyID int64, fromStatus, toStatus, reason, rejectedBy string, documentPath *string) (*models.EntityNote, error)`
   - **Fix**: Add an adapter type or align the interface in `EpicNoteRepository` to match the concrete `EntityNoteRepository`

2. **`service_accessors.go` line 60**: `FeatureNoteRepository` interface mismatch
   - Same root cause as #1 for `FeatureService`
   - **Fix**: Same approach as #1

3. **`services_global.go` line 116**: `taskcreation.Creator` not wired for `TaskService`
   - Impact: Task creation (`runTaskCreate`) will need the creator service wired or must continue to use the direct repository pattern for file creation until this is resolved
   - **Decision**: Wire `taskcreation.Creator` into `GetTaskService()` accessor as part of this feature (FR-F07-001 scope)

---

## 3. Key Architecture Decisions

### Decision 1: Thin Wrapper Pattern (Three Steps Only)

**Decision**: Every command handler is reduced to exactly three logical steps with no embedded business logic.

**Pattern**:
```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    taskKey := args[0]
    agentID, _ := cmd.Flags().GetString("agent")

    // Step 2: Call service
    svc := cli.GetTaskService()
    task, err := svc.StartTask(cmd.Context(), taskKey, agentID)
    if err != nil {
        return handleServiceError(err, taskKey)
    }

    // Step 3: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }
    cli.Success(fmt.Sprintf("Started task %s", task.Key))
    return nil
}
```

**Rationale**: This is the established pattern in the project's CLAUDE.md documentation and matches the service-layer migration guide. It is proven across existing thin-wrapper commands in the codebase.

**Prohibited in handlers**: `repository.New*` calls, `cli.GetDB()` followed by direct repo construction, `workflow.Service` access, `status.NewCalculationService()` calls, filtering loops, sorting logic.

### Decision 2: Centralized Error Handler

**Decision**: Extract a shared `handleServiceError(err error, key string) error` function into a shared helpers file within the `commands` package (or `internal/cli/`).

**Implementation**:
```go
// handleServiceError translates service-layer errors to CLI exit codes.
// Called from all thin-wrapper command handlers.
func handleServiceError(err error, entityKey string) error {
    if err == nil {
        return nil
    }

    var notFoundErr *repository.NotFoundError
    if errors.As(err, &notFoundErr) {
        cli.Error(fmt.Sprintf("%s not found: %s", notFoundErr.Entity, notFoundErr.Key))
        os.Exit(1)
    }

    var workflowErr *workflow.TransitionError
    if errors.As(err, &workflowErr) {
        cli.Error(fmt.Sprintf("Invalid operation: %v", err))
        os.Exit(3)
    }

    cli.Error(fmt.Sprintf("Error: %v", err))
    os.Exit(2)
    return nil // unreachable, satisfies compiler
}
```

**Rationale**: The research report identified 47+ variations of the same error-handling code duplicated across the three command files. A single helper eliminates this duplication and ensures consistent exit codes across all commands.

**Location**: Add to `internal/cli/commands/helpers.go` (new file, or append to existing helpers if available). The function must be package-private (lowercase) as it is an implementation detail of the commands package.

### Decision 3: DTO Helper Functions for Argument Parsing

**Decision**: Extract complex positional argument parsing from command handlers into named helper functions returning typed DTOs or structs.

**Pattern for commands with complex input**:
```go
// In task.go:
func runTaskCreate(cmd *cobra.Command, args []string) error {
    input := parseCreateTaskInput(cmd, args)
    svc := cli.GetTaskService()
    task, err := svc.CreateTask(cmd.Context(), input)
    if err != nil {
        return handleServiceError(err, input.EpicKey+"-"+input.FeatureKey)
    }
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(task)
    }
    cli.Success(fmt.Sprintf("Created task %s at %s", task.Key, task.FilePath))
    return nil
}

func parseCreateTaskInput(cmd *cobra.Command, args []string) services.CreateTaskInput {
    epicKey, featureKey, title := parseTaskCreateArgs(args, cmd)
    agentType, _ := cmd.Flags().GetString("agent")
    priority, _ := cmd.Flags().GetInt("priority")
    execOrder, _ := cmd.Flags().GetInt("order")
    filePath, _ := cmd.Flags().GetString("file")
    force, _ := cmd.Flags().GetBool("force")
    return services.CreateTaskInput{
        EpicKey:        epicKey,
        FeatureKey:     featureKey,
        Title:          title,
        AgentType:      agentType,
        Priority:       priority,
        ExecutionOrder: execOrder,
        FilePath:       filePath,
        Force:          force,
    }
}
```

**Rationale**: Keeps the command handler under 30 lines while making argument parsing independently testable. The parsing helper can be unit-tested without any service or database dependency.

### Decision 4: Service Accessor Note Repository Interface Fix

**Decision**: Fix the `EpicNoteRepository` and `FeatureNoteRepository` interface mismatches by introducing a thin adapter type in the accessor files, rather than changing the service interface or the repository implementation.

**Approach**: Create package-level adapters in `service_accessors.go`:
```go
// epicNoteAdapter wraps EntityNoteRepository to satisfy EpicNoteRepository interface
type epicNoteAdapter struct {
    inner *repository.EntityNoteRepository
}

func (a *epicNoteAdapter) CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
    historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error {
    et := models.EntityType(entityType)
    dp := &documentPath
    _, err := a.inner.CreateRejectionNote(ctx, et, entityID, historyID, fromStatus, toStatus, reason, rejectedBy, dp)
    return err
}
```

**Rationale**: The adapter is the least invasive fix. It does not require modifying the service interfaces (which would affect all callers) or the repository implementation (which has its own interface contract). The adapter is a pure translation layer with no business logic.

**Alternative considered**: Aligning the service interface to match the repository. Rejected because the service interface uses simpler types (string vs models.EntityType, string vs *string) that are appropriate for the service boundary.

### Decision 5: Incremental File-by-File Delivery

**Decision**: Refactor `task.go`, `feature.go`, and `epic.go` in separate commits/PRs, in that order. The intermediate state where some handlers call services and others still call repositories is explicitly valid.

**Order rationale**:
1. `task.go` first: Largest file, highest impact (2,664 lines), TaskService is most complete
2. `feature.go` second: Second largest, FeatureService has all needed methods
3. `epic.go` third: Third largest, EpicService has all needed methods

**Rationale**: Reduces PR size and review burden. Allows regression detection per file. Main branch remains deployable at each stage.

---

## 4. Integration Points

### Service Accessor Files (Prerequisite Changes)

Before any command handler can be refactored, the service accessors must be fully functional:

| File | Required Change | Blocker |
|------|----------------|---------|
| `internal/cli/service_accessors.go` | Add `epicNoteAdapter` and `featureNoteAdapter` types; wire them in `GetEpicService()` and `GetFeatureService()` | Yes - note repo mismatch prevents full service use |
| `internal/cli/services_global.go` | Wire `taskcreation.Creator` into `GetTaskService()` | Yes - needed for `runTaskCreate` |

### Command Handler Mapping to Service Methods

**task.go handlers and their target service calls**:

| Handler | Service Method | DTO/Input |
|---------|---------------|-----------|
| `runTaskCreate` | `svc.CreateTask(ctx, input)` | `services.CreateTaskInput` |
| `runTaskList` | `svc.ListTasks(ctx, filters)` | `services.TaskFilters` |
| `runTaskGet` | `svc.GetTask(ctx, key)` | string key |
| `runTaskStart` | `svc.StartTask(ctx, key, agentID)` | string key + string |
| `runTaskComplete` | `svc.CompleteTask(ctx, key, notes)` | string key + string |
| `runTaskApprove` | `svc.ApproveTask(ctx, key, notes)` | string key + string |
| `runTaskReopen` | `svc.ReopenTask(ctx, key, notes)` | string key + string |
| `runTaskBlock` | `svc.BlockTask(ctx, key, reason)` | string key + string |
| `runTaskUnblock` | `svc.UnblockTask(ctx, key)` | string key |
| `runTaskNext` | `svc.GetNextTask(ctx, filters)` | `services.NextTaskFilters` |
| `runTaskDelete` | `svc.DeleteTask(ctx, key)` | string key |
| `runTaskUpdate` | `svc.UpdateTask(ctx, key, updates)` | `services.TaskUpdates` |
| `runTaskSetStatus` | `svc.TransitionStatus(ctx, key, status, opts)` | string key + string |

**feature.go handlers and their target service calls**:

| Handler | Service Method | DTO/Input |
|---------|---------------|-----------|
| `runFeatureCreate` | requires `CreateFeature` (missing - see Gap Analysis) | `services.CreateFeatureInput` |
| `runFeatureList` | `svc.ListFeatures(ctx, filters)` | `services.FeatureFilters` |
| `runFeatureGet` | `svc.GetFeature(ctx, key)` | string key |
| `runFeatureUpdate` | requires `UpdateFeature` (missing - see Gap Analysis) | `services.FeatureUpdates` |
| `runFeatureComplete` | `svc.TransitionStatus(ctx, key, target, opts)` | transition options |
| `runFeatureDelete` | requires `DeleteFeature` (missing - see Gap Analysis) | string key |

**epic.go handlers and their target service calls**:

| Handler | Service Method | DTO/Input |
|---------|---------------|-----------|
| `runEpicCreate` | requires `CreateEpic` (missing - see Gap Analysis) | `services.CreateEpicInput` |
| `runEpicList` | `svc.ListEpics(ctx, filters)` | `services.EpicFilters` |
| `runEpicGet` | `svc.GetEpic(ctx, key)` | string key |
| `runEpicUpdate` | requires `UpdateEpic` (missing - see Gap Analysis) | `services.EpicUpdates` |
| `runEpicComplete` | `svc.TransitionStatus(ctx, key, target, opts)` | transition options |
| `runEpicDelete` | requires `DeleteEpic` (missing - see Gap Analysis) | string key |

---

## 5. Service Method Gap Analysis

Several command handlers require service methods that do not yet exist. These gaps must be filled as part of this feature or escalated to E15-F05 before those handlers can be refactored.

### Missing Service Methods

**FeatureService gaps**:
- `CreateFeature(ctx context.Context, input CreateFeatureInput) (*models.Feature, error)` - needed by `runFeatureCreate`
- `UpdateFeature(ctx context.Context, key string, updates FeatureUpdates) (*models.Feature, error)` - needed by `runFeatureUpdate`
- `DeleteFeature(ctx context.Context, key string) error` - needed by `runFeatureDelete`

**EpicService gaps**:
- `CreateEpic(ctx context.Context, input CreateEpicInput) (*models.Epic, error)` - needed by `runEpicCreate`
- `UpdateEpic(ctx context.Context, key string, updates EpicUpdates) (*models.Epic, error)` - needed by `runEpicUpdate`
- `DeleteEpic(ctx context.Context, key string) error` - needed by `runEpicDelete`

### Gap Resolution Strategy

**Option A (Preferred)**: Add the missing methods to the respective services as part of this feature's task scope. The CRUD methods are straightforward and follow the same pattern as existing `GetFeature`, `ListFeatures`, `GetEpic`, `ListEpics` methods.

**Option B**: For handlers where the service method is missing, escalate to E15-F05 (Epic and Feature Service Expansion) and defer refactoring that specific handler until F05 is merged.

**Recommendation**: Option A for CRUD methods (Create, Update, Delete) since these are low-risk additions that follow established patterns. The method signatures below serve as the contracts:

```go
// FeatureService additions
func (s *FeatureService) CreateFeature(ctx context.Context, input CreateFeatureInput) (*models.Feature, error)
func (s *FeatureService) UpdateFeature(ctx context.Context, key string, updates FeatureUpdates) (*models.Feature, error)
func (s *FeatureService) DeleteFeature(ctx context.Context, key string) error

// EpicService additions
func (s *EpicService) CreateEpic(ctx context.Context, input CreateEpicInput) (*models.Epic, error)
func (s *EpicService) UpdateEpic(ctx context.Context, key string, updates EpicUpdates) (*models.Epic, error)
func (s *EpicService) DeleteEpic(ctx context.Context, key string) error
```

### DTO Contracts for Missing Methods

```go
// CreateFeatureInput for FeatureService.CreateFeature
type CreateFeatureInput struct {
    EpicKey        string
    Title          string
    ExecutionOrder int
    FilePath       string
    Force          bool
}

// FeatureUpdates for FeatureService.UpdateFeature
type FeatureUpdates struct {
    Title          *string
    ExecutionOrder *int
    Description    *string
}

// CreateEpicInput for EpicService.CreateEpic
type CreateEpicInput struct {
    Title         string
    FilePath      string
    Priority      int
    BusinessValue int
    Force         bool
}

// EpicUpdates for EpicService.UpdateEpic
type EpicUpdates struct {
    Title         *string
    Priority      *int
    BusinessValue *int
    Description   *string
}
```

---

## 6. Output Format Preservation

The refactoring must produce byte-identical output. The key risk areas are:

### JSON Output Preservation

Service methods return domain models (`*models.Task`, `*models.Feature`, `*models.Epic`). The JSON serialization depends on the `json:` struct tags in `internal/models/`. These are unchanged by the refactoring, so JSON output is structurally preserved.

**Risk**: Commands that currently build custom response structs (not direct model serialization) must continue to build those same structs after refactoring. Inspect each handler before refactoring to identify custom response types.

**Mitigation**: For `shark task get --json`, `shark feature get --json`, and `shark epic get --json`, capture the output before and after refactoring and compare with `diff`. These are the highest-risk commands for format regression.

### Table Output Preservation

Table column headers, ordering, and content are presentation logic that stays in the command layer. When the underlying data comes from service calls instead of direct repository queries, the data structure is the same (same model types). Column formatting remains in the command handler.

### Success/Error Message Preservation

Success messages (e.g., "Created task E07-F01-001") are generated in the command handler, not the service. These are preserved as-is.

Error messages change from ad-hoc strings to the centralized `handleServiceError` format. The centralized handler must produce the same or equivalent messages for each error type to maintain backward compatibility.

---

## 7. Testing Strategy

### Before Refactoring: Capture Baseline

For each of the three files, before making changes:
1. Run representative commands and capture output to baseline files
2. Record exit codes for error scenarios

Key commands to baseline:
```bash
./bin/shark task next --json > baseline-task-next.json
./bin/shark task get E15-F07-001 --json > baseline-task-get.json
./bin/shark feature get E15-F07 --json > baseline-feature-get.json
./bin/shark epic get E15 --json > baseline-epic-get.json
```

### Post-Refactoring Verification

After refactoring each file:
```bash
# Verify identical JSON structure
./bin/shark task next --json | diff baseline-task-next.json -
./bin/shark task get E15-F07-001 --json | diff baseline-task-get.json -

# Verify line counts meet targets
wc -l internal/cli/commands/task.go     # Must be <= 400
wc -l internal/cli/commands/feature.go  # Must be <= 350
wc -l internal/cli/commands/epic.go     # Must be <= 300

# Verify zero repository calls
grep -c "repository\.New" internal/cli/commands/task.go     # Must be 0
grep -c "repository\.New" internal/cli/commands/feature.go  # Must be 0
grep -c "repository\.New" internal/cli/commands/epic.go     # Must be 0

# Full test suite
make fmt && make lint && make test
```

### New Unit Tests for Thin Wrappers

Post-refactoring, command handlers are testable with mocked services. Write tests for:
1. Argument parsing helpers (`parseCreateTaskInput`, `parseTaskFilters`, etc.) - pure function tests, no mocking needed
2. Handler error-to-exit-code mapping - mock service returning specific error types
3. JSON output format - mock service returning known data, verify JSON structure

Test file locations:
- `internal/cli/commands/task_test.go` - task command tests with mocked TaskService
- `internal/cli/commands/feature_test.go` - feature command tests with mocked FeatureService
- `internal/cli/commands/epic_test.go` - epic command tests with mocked EpicService

Mock pattern (function-field mocks, consistent with project testing patterns):
```go
type mockTaskService interface {
    GetTask(ctx context.Context, key string) (*models.Task, error)
    StartTask(ctx context.Context, key string, agentID string) (*models.Task, error)
    // ... other methods used by command handlers
}

// In test file:
type stubTaskService struct {
    getTaskFunc  func(ctx context.Context, key string) (*models.Task, error)
    startTaskFunc func(ctx context.Context, key string, agentID string) (*models.Task, error)
}

func (s *stubTaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    if s.getTaskFunc != nil {
        return s.getTaskFunc(ctx, key)
    }
    return nil, fmt.Errorf("GetTask not configured")
}
```

---

## 8. Implementation Phases

### Phase 0: Prerequisites (Must Complete First)

- Fix `epicNoteAdapter` interface mismatch in `service_accessors.go`
- Fix `featureNoteAdapter` interface mismatch in `service_accessors.go`
- Wire `taskcreation.Creator` into `GetTaskService()` in `services_global.go`
- Add missing CRUD methods to `FeatureService` and `EpicService` (or escalate to F05)
- Run `make test` to confirm baseline is green

### Phase 1: task.go Refactoring

1. Capture baseline JSON output for key task commands
2. Add `handleServiceError` helper to `internal/cli/commands/helpers.go`
3. Refactor each handler function top-to-bottom:
   - Extract argument parsing to named helper (if complex)
   - Replace body with: get service → call method → format output
   - Remove repository imports from handler
4. Run `make fmt && make lint && make test`
5. Verify `wc -l task.go` is under 400
6. Verify `grep -c "repository\.New" task.go` is 0
7. Compare JSON output with baseline

### Phase 2: feature.go Refactoring

1. Capture baseline JSON output for key feature commands
2. Refactor each handler using same approach as Phase 1
3. Run `make fmt && make lint && make test`
4. Verify line count and zero repository imports

### Phase 3: epic.go Refactoring

1. Capture baseline JSON output for key epic commands
2. Refactor each handler using same approach as Phase 1
3. Run `make fmt && make lint && make test`
4. Verify line count and zero repository imports

### Phase 4: Test Addition

1. Write unit tests for parsing helpers
2. Write unit tests for error-to-exit-code mapping using service mocks
3. Run `make test-coverage` to verify coverage improvement

---

## 9. Security Considerations

**No security impact**. This refactoring:
- Makes no changes to authentication or authorization logic
- Makes no changes to what data is read or written
- Makes no changes to database access patterns (same underlying repositories)
- Does not introduce new external dependencies
- Does not change CLI input validation requirements (validation moves from command to service, where it already resides for the service-refactored commands)

The security posture is unchanged. Business logic previously in commands is moved to services; the same logic enforces the same rules.

---

## 10. Constraints and Non-Negotiables

1. **Zero behavior regression**: All commands produce identical output (verified by test suite + baseline comparison)
2. **Zero repository calls from commands**: `grep -c "repository\.New"` must return 0 for all three files after completion
3. **Line count targets**: task.go ≤ 400, feature.go ≤ 350, epic.go ≤ 300
4. **Incremental delivery**: Each file refactorable and mergeable independently
5. **Service accessor completeness**: All TODO comments in accessor files resolved before command refactoring begins
6. **No new business logic in commands**: Even if a service method is missing, the solution is to add it to the service (not add temporary business logic to the command)

---

## 11. Exit Gate Checklist

- [x] Key decisions documented with rationale (thin wrapper pattern, error handler centralization, DTO helpers, incremental delivery)
- [x] Service method gaps identified and resolution strategy defined
- [x] Note repository interface mismatch documented with fix approach (adapter pattern)
- [x] Handler-to-service method mapping complete for all three files
- [x] Output format preservation risk identified and mitigation defined (baseline comparison)
- [x] Testing strategy documented (baseline capture, post-refactor verification, unit test targets)
- [x] Implementation phases sequenced with clear prerequisites
- [x] No ambiguity in approach - developers have sufficient guidance to implement each task

---

*Architecture designed for E15-F07: CLI Commands as Thin Wrappers*
*Feature complexity: STANDARD*
*Last updated: 2026-02-18*
