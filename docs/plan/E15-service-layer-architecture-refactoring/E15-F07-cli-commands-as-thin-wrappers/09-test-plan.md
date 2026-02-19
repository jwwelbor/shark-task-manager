# Test Plan: E15-F07 — CLI Commands as Thin Wrappers

**Feature**: E15-F07 - CLI Commands as Thin Wrappers
**Epic**: E15 - Service Layer Architecture Refactoring
**Complexity Tier**: STANDARD
**Test Plan Type**: Focused Test Plan (AC Test Matrix + API Contract Tests + Component Test Strategy + Integration Scenarios)
**Date**: 2026-02-18
**Author**: QA Agent

---

## Executive Summary

This test plan validates refactoring of three fat CLI command files (`task.go`, `feature.go`, `epic.go`) into thin wrappers that follow the parse → call service → format output pattern. The refactoring is behavior-preserving: no user-facing functionality changes.

**User Goal This Serves**: Shark developers and AI Developer Agents need to navigate command behavior, write tests, and extend functionality without reading thousands of lines of intermixed business logic and presentation code.

**Critical Success Criteria**:
- All three command files meet line count targets (task.go ≤400, feature.go ≤350, epic.go ≤300)
- Zero `repository.New*` calls remain in any of the three files
- All existing tests pass unchanged (`make test` exits 0)
- JSON and table output is byte-identical before and after refactoring
- Error exit codes are consistent across all command handlers (1=not found, 2=system, 3=invalid state)

**Risk Profile**: The main risk is regression — 6,854 lines of command code with 61 repository instantiations is being redirected through the service layer. Baseline capture before each file refactoring is mandatory.

---

## 1. Acceptance Criteria Test Matrix

### Story 1: Global Service Accessors Complete and Wired

**User Goal**: Developers can call `cli.GetTaskService()`, `cli.GetFeatureService()`, or `cli.GetEpicService()` and receive a fully wired service instance without managing dependency construction.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC1.1** | `cli.GetTaskService()` returns wired `*services.TaskService` | Call `GetTaskService()` and invoke `.GetTask(ctx, key)` on result | Valid task key | Returns `*models.Task`, no nil pointer panic | Verify `taskcreation.Creator` is wired (not nil); verify no TODO in accessor body |
| **AC1.2** | `cli.GetFeatureService()` returns wired `*services.FeatureService` | Call `GetFeatureService()` and invoke `.GetFeature(ctx, key)` on result | Valid feature key | Returns `*models.Feature`, no nil pointer panic | Verify note repo adapter resolves interface mismatch at `service_accessors.go:60` |
| **AC1.3** | `cli.GetEpicService()` returns wired `*services.EpicService` | Call `GetEpicService()` and invoke `.GetEpic(ctx, key)` on result | Valid epic key | Returns `*models.Epic`, no nil pointer panic | Verify note repo adapter resolves interface mismatch at `service_accessors.go:34` |
| **AC1.4** | No TODO comments indicating missing wiring remain in accessor files | `grep -n "TODO" internal/cli/service_accessors.go internal/cli/services_global.go` | Both accessor files | Zero matches (or only approved deferred TODOs with tracked issues) | TODOs at `service_accessors.go:34`, `:60`, and `services_global.go:116` must all be resolved |
| **AC1.5** | Note repository interface mismatch fixed via adapter | Code review of `service_accessors.go` | Inspect `GetEpicService()` and `GetFeatureService()` | Adapter types present; no nil passed for noteRepo | Adapters wrap `*repository.EntityNoteRepository`; verify `CreateRejectionNote` signature translates correctly |

---

### Story 2: task.go Refactored to Thin Wrapper

**User Goal**: A developer opening `task.go` finds a navigable file where each handler is 15-30 lines of parse → call → format, not 50-200 lines of intermixed business logic.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC2.1** | `task.go` is at most 400 lines | `wc -l internal/cli/commands/task.go` | File on disk | ≤400 | Baseline: 2,664 lines; count includes blank lines and comments |
| **AC2.2** | No `repository.New*` calls in `task.go` | `grep -c "repository\.New" internal/cli/commands/task.go` | `task.go` content | 0 | Currently 33 occurrences; all must be eliminated |
| **AC2.3** | Every handler follows parse → call service → format | Code review: inspect `runTaskCreate`, `runTaskStart`, `runTaskComplete`, `runTaskApprove`, `runTaskReopen`, `runTaskBlock`, `runTaskUnblock`, `runTaskGet`, `runTaskList`, `runTaskNext` | `task.go` source | Each handler: (1) parses args/flags to primitives or DTO, (2) calls one `svc.Method(ctx, ...)`, (3) formats output | No filtering loops, no sort.Slice, no workflow checks, no `status.NewCalculationService()` calls |
| **AC2.4** | `shark task start E07-F01-001` output unchanged | Capture baseline before refactoring, compare after | Same task key pre/post | Byte-identical success message | Also verify exit code is 0 on success |
| **AC2.5** | `shark task next --json` output structure unchanged | Capture baseline JSON before refactoring, diff after | Same environment pre/post | JSON field names and types are identical | Field ordering may differ; validate semantic equivalence |
| **AC2.6** | `make lint` passes on `task.go` with no warnings | `make fmt && make lint` | Post-refactoring `task.go` | Exit code 0, no "imported and not used" for `repository` package | Unused imports removed as part of refactoring |
| **AC2.7** | `runTaskCreate` uses `parseCreateTaskInput()` helper | Code review of `task.go` | `task.go` source | Helper function `parseCreateTaskInput(cmd, args)` exists and is called from handler | Helper returns `services.CreateTaskInput` struct |

---

### Story 3: feature.go Refactored to Thin Wrapper

**User Goal**: A developer navigating `feature.go` finds each handler is a clean 3-step pattern with progress calculation, health derivation, and action item aggregation delegated entirely to `FeatureService`.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC3.1** | `feature.go` is at most 350 lines | `wc -l internal/cli/commands/feature.go` | File on disk | ≤350 | Baseline: 2,252 lines |
| **AC3.2** | No `repository.New*` calls in `feature.go` | `grep -c "repository\.New" internal/cli/commands/feature.go` | `feature.go` content | 0 | Currently 15 occurrences |
| **AC3.3** | No progress calculation logic in `feature.go` | `grep -n "completed.*total\|/ total\|CalculateProgress" internal/cli/commands/feature.go` | `feature.go` content | 0 matches | Progress must be retrieved via `svc.GetProgress(ctx, key)` |
| **AC3.4** | `shark feature get E07-F01 --json` output structure unchanged | Capture baseline JSON before refactoring, diff after | Same feature key pre/post | Identical JSON structure (fields, types, values) | Custom response structs in current handler must be preserved or mapped |
| **AC3.5** | `shark feature list --json` output structure unchanged | Capture baseline JSON, diff after refactoring | Same database state pre/post | Identical JSON array structure | Verify health indicators and progress fields present in output |
| **AC3.6** | `make lint` passes on `feature.go` | `make fmt && make lint` | Post-refactoring `feature.go` | Exit code 0, no unused `repository` import | |

---

### Story 4: epic.go Refactored to Thin Wrapper

**User Goal**: A developer navigating `epic.go` finds each handler delegates all feature rollup, impediment analysis, and progress aggregation to `EpicService`.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC4.1** | `epic.go` is at most 300 lines | `wc -l internal/cli/commands/epic.go` | File on disk | ≤300 | Baseline: 1,938 lines |
| **AC4.2** | No `repository.New*` calls in `epic.go` | `grep -c "repository\.New" internal/cli/commands/epic.go` | `epic.go` content | 0 | Currently 13 occurrences |
| **AC4.3** | No feature rollup loops in `epic.go` | `grep -n "for.*feature\|range features" internal/cli/commands/epic.go` | `epic.go` content | 0 matches (or only in formatting output) | Rollup must be retrieved via `svc.GetFeatureRollup(ctx, key)` |
| **AC4.4** | `shark epic get E15 --json` output structure unchanged | Capture baseline JSON before refactoring, diff after | Same epic key pre/post | Identical JSON structure | Feature rollup arrays, task rollup counts must be preserved |
| **AC4.5** | `make lint` passes on `epic.go` | `make fmt && make lint` | Post-refactoring `epic.go` | Exit code 0, no unused `repository` import | |

---

### Story 5: Zero Behavior Regression

**User Goal**: As a Shark user (human or AI), all CLI commands I depend on in scripts and workflows produce identical results after the refactoring.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC5.1** | `make test` exits 0 after each file refactoring | `make test` post task.go refactor, post feature.go refactor, post epic.go refactor | Full test suite | Exit code 0, no test failures | Run each time; intermediate states must not break CI |
| **AC5.2** | `shark task start` success message identical | `./bin/shark task start <key>` before and after | Same task, same initial status | Byte-identical success message (e.g., "Started task E07-F01-001") | Message wording preserved exactly |
| **AC5.3** | `shark task complete` success message identical | `./bin/shark task complete <key> --notes="test"` before and after | Same task key | Byte-identical success message | |
| **AC5.4** | `shark feature get` JSON is identical | JSON diff of `shark feature get E15-F07 --json` before and after | Same feature state | Identical JSON (semantic equality) | Use `jq` comparison for field-order independence |
| **AC5.5** | `shark epic get` JSON is identical | JSON diff of `shark epic get E15 --json` before and after | Same epic state | Identical JSON (semantic equality) | |
| **AC5.6** | Exit codes for error scenarios unchanged | Run commands targeting non-existent keys before and after | Invalid keys (e.g., `shark task get FAKE-001`) | Exit code 1 (not found) before and after | Test all three error categories: 1, 2, 3 |

---

### Story 6: Centralized Error Handler

**User Goal**: Developers reading any command handler find consistent error translation to exit codes rather than 47 variations of ad-hoc error string formatting.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC6.1** | `handleServiceError` (or equivalent) exists in commands package | `grep -rn "handleServiceError\|HandleServiceError" internal/cli/commands/` | commands package | At least one file defines the function | Function may be in `helpers.go` or similar |
| **AC6.2** | `NotFoundError` produces exit code 1 | Unit test: pass `&repository.NotFoundError{}` to error handler | Mock error | `os.Exit(1)` called; `cli.Error` called with message containing key | Test via subprocess or exit code capture |
| **AC6.3** | Workflow/transition error produces exit code 3 | Unit test: pass transition error to error handler | Mock workflow error | `os.Exit(3)` called; `cli.Error` called with human-readable message | |
| **AC6.4** | All other errors produce exit code 2 | Unit test: pass `fmt.Errorf("db connection failed")` to error handler | Generic error | `os.Exit(2)` called; `cli.Error` called with error details | |
| **AC6.5** | `handleServiceError` is used in all three command files | `grep -c "handleServiceError" internal/cli/commands/task.go internal/cli/commands/feature.go internal/cli/commands/epic.go` | All three files | Each file has at least one call | |

---

### Story 7: Argument Parsing Helpers

**User Goal**: Developers and AI agents writing tests for command argument parsing can test parsing logic independently from service mocking.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC7.1** | `parseCreateTaskInput()` helper exists and returns typed DTO | Code review of `task.go` | `task.go` source | Function returns `services.CreateTaskInput` | Helper handles 3-arg, 2-arg, and flag formats |
| **AC7.2** | Parsing helpers are independently unit-testable | Unit test calling `parseCreateTaskInput(cmd, args)` directly | Cobra command with populated flags, various arg formats | Correct `CreateTaskInput` struct populated | No service or database call required to test |
| **AC7.3** | 3-arg task create format parses correctly | Unit test: `args = ["E07", "F01", "My Task"]` | 3-arg format | `EpicKey="E07"`, `FeatureKey="F01"`, `Title="My Task"` | Case insensitive keys handled |
| **AC7.4** | 2-arg task create format parses correctly | Unit test: `args = ["E07-F01", "My Task"]` | 2-arg format | `EpicKey="E07"`, `FeatureKey="F01"` (or full `E07-F01`), `Title="My Task"` | Slug formats like `E07-F01-my-feature` also handled |
| **AC7.5** | Feature create positional parsing correct | Unit test: `args = ["E07", "My Feature Title"]` | Feature create args | `EpicKey="E07"`, `Title="My Feature Title"` | |

---

### Story 8: Repository Import Absence

**User Goal**: Developers reading command file import blocks see only the actual dependencies of the command layer (services, cli utilities) not the repository layer.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC8.1** | `task.go` does not import `repository` package | `grep "repository" internal/cli/commands/task.go` | `task.go` imports | No repository import | May still import `repository` for `NotFoundError` type assertion; acceptable if for error handling only |
| **AC8.2** | `feature.go` does not import `repository` package for data access | `grep "repository\.New" internal/cli/commands/feature.go` | `feature.go` | 0 matches for `repository.New` calls | Error type imports acceptable |
| **AC8.3** | `epic.go` does not import `repository` package for data access | `grep "repository\.New" internal/cli/commands/epic.go` | `epic.go` | 0 matches for `repository.New` calls | Error type imports acceptable |
| **AC8.4** | `make fmt` produces no changes to import sections | `make fmt` and `git diff --stat` | All three command files after refactoring | No changes to import formatting | |

---

## 2. API Contract Test Cases

These contracts define the exact interface between the refactored CLI commands and the service layer. Each contract must be verifiable via compilation and unit tests.

### Service Accessor Contracts

**Contract**: Global accessors return fully-wired service instances that can execute real operations.

| Contract ID | Accessor | Required Wiring | Test | Pass Condition |
|-------------|----------|-----------------|------|----------------|
| **ACC-C1** | `cli.GetTaskService()` | TaskRepository, workflow.Service, taskcreation.Creator | Call `.StartTask(ctx, validKey, "agent")` | No nil pointer panic; returns `*models.Task` or domain error |
| **ACC-C2** | `cli.GetFeatureService()` | FeatureRepository, workflow.Service, noteRepo (adapter), TaskRepository | Call `.GetFeature(ctx, validKey)` | Returns `*models.Feature`, no nil pointer |
| **ACC-C3** | `cli.GetEpicService()` | EpicRepository, workflow.Service, noteRepo (adapter), FeatureRepository, TaskRepository | Call `.GetEpic(ctx, validKey)` | Returns `*models.Epic`, no nil pointer |
| **ACC-C4** | Note repo adapter (epic) | Wraps `*repository.EntityNoteRepository` | Call `CreateRejectionNote` on adapter | Translates `string` entityType to `models.EntityType`; translates `string` docPath to `*string` |
| **ACC-C5** | Note repo adapter (feature) | Wraps `*repository.EntityNoteRepository` | Call `CreateRejectionNote` on adapter | Same translation as ACC-C4 |

### Command-to-Service Method Contracts

**Contract**: Each refactored handler calls exactly one service method per operation and passes correctly-typed inputs.

**task.go handler contracts**:

| Contract ID | Handler | Service Method | Input Type | Output Type |
|-------------|---------|----------------|------------|-------------|
| **CMD-C1** | `runTaskCreate` | `TaskService.CreateTask(ctx, input)` | `services.CreateTaskInput` | `*models.Task` |
| **CMD-C2** | `runTaskList` | `TaskService.ListTasks(ctx, filters)` | `services.TaskFilters` | `[]*models.Task` |
| **CMD-C3** | `runTaskGet` | `TaskService.GetTask(ctx, key)` | `string` | `*models.Task` |
| **CMD-C4** | `runTaskStart` | `TaskService.StartTask(ctx, key, agentID)` | `string, string` | `*models.Task` |
| **CMD-C5** | `runTaskComplete` | `TaskService.CompleteTask(ctx, key, notes)` | `string, string` | `*models.Task` |
| **CMD-C6** | `runTaskApprove` | `TaskService.ApproveTask(ctx, key, notes)` | `string, string` | `*models.Task` |
| **CMD-C7** | `runTaskReopen` | `TaskService.ReopenTask(ctx, key, notes)` | `string, string` | `*models.Task` |
| **CMD-C8** | `runTaskBlock` | `TaskService.BlockTask(ctx, key, reason)` | `string, string` | `*models.Task` |
| **CMD-C9** | `runTaskUnblock` | `TaskService.UnblockTask(ctx, key)` | `string` | `*models.Task` |
| **CMD-C10** | `runTaskNext` | `TaskService.GetNextTask(ctx, filters)` | `services.NextTaskFilters` | `*models.Task` |

**feature.go handler contracts**:

| Contract ID | Handler | Service Method | Input Type | Output Type |
|-------------|---------|----------------|------------|-------------|
| **CMD-C11** | `runFeatureCreate` | `FeatureService.CreateFeature(ctx, input)` | `services.CreateFeatureInput` | `*models.Feature` |
| **CMD-C12** | `runFeatureList` | `FeatureService.ListFeatures(ctx, filters)` | `services.FeatureFilters` | `[]*models.Feature` |
| **CMD-C13** | `runFeatureGet` | `FeatureService.GetFeature(ctx, key)` | `string` | `*models.Feature` |
| **CMD-C14** | `runFeatureUpdate` | `FeatureService.UpdateFeature(ctx, key, updates)` | `string, services.FeatureUpdates` | `*models.Feature` |

**epic.go handler contracts**:

| Contract ID | Handler | Service Method | Input Type | Output Type |
|-------------|---------|----------------|------------|-------------|
| **CMD-C15** | `runEpicCreate` | `EpicService.CreateEpic(ctx, input)` | `services.CreateEpicInput` | `*models.Epic` |
| **CMD-C16** | `runEpicList` | `EpicService.ListEpics(ctx, filters)` | `services.EpicFilters` | `[]*models.Epic` |
| **CMD-C17** | `runEpicGet` | `EpicService.GetEpic(ctx, key)` | `string` | `*models.Epic` |
| **CMD-C18** | `runEpicUpdate` | `EpicService.UpdateEpic(ctx, key, updates)` | `string, services.EpicUpdates` | `*models.Epic` |

### Error Handling Contracts

**Contract**: `handleServiceError(err, key)` translates service errors to consistent CLI exit codes.

| Contract ID | Error Type | Expected Exit Code | Expected Output |
|-------------|-----------|-------------------|-----------------|
| **ERR-C1** | `*repository.NotFoundError` | 1 | `cli.Error("<entity> not found: <key>")` + `os.Exit(1)` |
| **ERR-C2** | `*workflow.TransitionError` (or equivalent) | 3 | `cli.Error("Invalid operation: <message>")` + `os.Exit(3)` |
| **ERR-C3** | Any other error (db error, system error) | 2 | `cli.Error("Error: <message>")` + `os.Exit(2)` |
| **ERR-C4** | `nil` error | 0 (no exit call) | Function returns nil, execution continues |

### Output Format Contracts

**Contract**: Refactored commands produce structurally identical JSON output to pre-refactoring.

| Contract ID | Command | JSON Fields Required | Verified By |
|-------------|---------|---------------------|-------------|
| **OUT-C1** | `shark task get <key> --json` | `key`, `title`, `status`, `agent_type`, `priority`, `created_at`, `updated_at` | Baseline diff |
| **OUT-C2** | `shark task list --json` | Array of tasks with same fields as OUT-C1 | Baseline diff |
| **OUT-C3** | `shark task next --json` | Same as OUT-C1 + `orchestrator_action` if present | Baseline diff |
| **OUT-C4** | `shark feature get <key> --json` | `key`, `title`, `status`, `progress_pct`, `epic_id`, health fields | Baseline diff |
| **OUT-C5** | `shark feature list --json` | Array of features with same fields as OUT-C4 | Baseline diff |
| **OUT-C6** | `shark epic get <key> --json` | `key`, `title`, `status`, feature rollup fields, task rollup fields | Baseline diff |

---

## 3. Component Test Strategy

### Component 1: Service Accessor Wiring (`internal/cli/service_accessors.go`, `internal/cli/services_global.go`)

**Purpose**: Verify all three accessors return correctly wired, usable service instances with the note repository interface mismatches resolved.

**Test Approach**: Integration tests using real database (test DB) to confirm the full wiring chain works end-to-end.

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| `GetTaskService()` does not panic | Call in test with test DB available | No panic, returns `*services.TaskService` |
| `GetTaskService().GetTask()` succeeds for known key | Integration test with seeded data | Returns `*models.Task` with correct key |
| `GetFeatureService()` does not panic | Call in test | No panic, returns `*services.FeatureService` |
| `GetEpicService()` does not panic | Call in test | No panic, returns `*services.EpicService` |
| Note adapter translates `CreateRejectionNote` correctly | Unit test of adapter directly | `string` entityType converted to `models.EntityType`; `string` docPath to `*string` |
| `taskcreation.Creator` wired in `GetTaskService()` | Inspect accessor code; call `svc.CreateTask()` in integration test | No nil panic from creator; task file created |
| Accessor panics on DB failure | Mock/override DB to return error | Panic message contains "failed to get database" |

**Coverage Target**: 100% (all accessors exercised, both success and failure paths)

**Test file location**: `internal/cli/services_global_test.go` or `internal/cli/db_global_test.go` (extend existing)

---

### Component 2: Argument Parsing Helpers

**Purpose**: Verify that argument parsing helper functions correctly extract CLI inputs into typed DTOs, independently from service or database.

**Test Approach**: Pure unit tests. No database, no service mocking needed. Parsing helpers are pure functions.

**Key Test Cases**:

| Test Case | Input | Expected DTO Field Values |
|-----------|-------|--------------------------|
| `parseCreateTaskInput` 3-arg format | `args=["E07", "F01", "Task Title"]`, no flags | `EpicKey="E07"`, `FeatureKey="F01"`, `Title="Task Title"` |
| `parseCreateTaskInput` 2-arg format | `args=["E07-F01", "Task Title"]`, no flags | `EpicKey="E07"`, `FeatureKey="F01"` (or full key), `Title="Task Title"` |
| `parseCreateTaskInput` flag format | `args=["Task Title"]`, `--epic=E07 --feature=F01` | `EpicKey="E07"`, `FeatureKey="F01"`, `Title="Task Title"` |
| `parseCreateTaskInput` with optional flags | `args=["E07", "F01", "Task"]`, `--agent=backend --priority=8 --order=2` | `AgentType="backend"`, `Priority=8`, `ExecutionOrder=2` |
| `parseCreateTaskInput` case insensitive | `args=["e07", "f01", "Task"]` | Keys preserved as input (normalization is service responsibility) |
| Feature create positional | `args=["E07", "My Feature"]` | `EpicKey="E07"`, `Title="My Feature"` |
| Epic create with flags | `--title="My Epic" --priority=5` | `Title="My Epic"`, `Priority=5` |
| Task filters with positional args | `args=["E07", "F01"]`, `--status=todo` | `EpicKey="E07"`, `FeatureKey="F01"`, `Status="todo"` |

**Coverage Target**: 90%+ for all parsing helper functions

**Test file location**: `internal/cli/commands/task_test.go`, `feature_test.go`, `epic_test.go` (new sections or new files)

---

### Component 3: Centralized Error Handler (`handleServiceError`)

**Purpose**: Verify that the shared error translation function maps error types to correct exit codes consistently.

**Test Approach**: Unit tests. Use `os.Exit` capture pattern (subprocess or redirect) to verify exit codes.

**Key Test Cases**:

| Test Case | Input Error | Expected Exit Code | Expected cli.Error Message |
|-----------|-------------|-------------------|---------------------------|
| NotFoundError for task | `&repository.NotFoundError{Entity:"task", Key:"E07-F01-001"}` | 1 | Contains "task" and "E07-F01-001" |
| NotFoundError for feature | `&repository.NotFoundError{Entity:"feature", Key:"E07-F01"}` | 1 | Contains "feature" and "E07-F01" |
| Workflow transition error | Any `errors.As`-detectable transition error | 3 | Contains "Invalid operation" or similar |
| Database connection error | `fmt.Errorf("connection refused")` | 2 | Contains error details |
| nil error | nil | No exit called | No output |

**Note on Testing `os.Exit`**: Since `os.Exit` terminates the test process, use subprocess execution or replace `os.Exit` with a testable exit function via dependency injection for these tests.

**Coverage Target**: 100% of error branches

**Test file location**: `internal/cli/commands/helpers_errors_test.go` (existing) or new `helpers_test.go` section

---

### Component 4: Refactored Command Handlers (Structural Validation)

**Purpose**: Validate that the command handler structure follows the three-step pattern and has no prohibited patterns.

**Test Approach**: Automated code inspection (grep-based) plus existing test suite regression. New unit tests use mocked service interfaces.

**Key Test Cases — Code Inspection**:

| Test Case | Grep Command | Expected Result |
|-----------|-------------|-----------------|
| Zero repository instantiations in task.go | `grep -c "repository\.New" internal/cli/commands/task.go` | 0 |
| Zero repository instantiations in feature.go | `grep -c "repository\.New" internal/cli/commands/feature.go` | 0 |
| Zero repository instantiations in epic.go | `grep -c "repository\.New" internal/cli/commands/epic.go` | 0 |
| Zero `cli.GetDB` calls followed by direct repo use in task.go | `grep -c "cli\.GetDB" internal/cli/commands/task.go` | 0 |
| Zero `status.NewCalculationService` in task.go | `grep -c "status\.NewCalculationService" internal/cli/commands/task.go` | 0 |
| Service accessor calls present in task.go | `grep -c "cli\.GetTaskService\(\)" internal/cli/commands/task.go` | ≥1 |
| File line counts meet targets | `wc -l task.go feature.go epic.go` | task≤400, feature≤350, epic≤300 |

**Key Test Cases — Unit Tests with Mocked Services**:

Use stub service interface to test handler behavior without database:

```
TestRunTaskStart_CallsServiceStartTask
  - Mock: stubTaskService.StartTask returns (*Task, nil)
  - Assert: handler calls StartTask with correct key and agentID
  - Assert: handler outputs success message on stdout

TestRunTaskStart_NotFoundError_ExitsCode1
  - Mock: stubTaskService.StartTask returns (nil, &repository.NotFoundError{})
  - Assert: exit code 1 (subprocess test)

TestRunTaskGet_JSONOutput
  - Mock: stubTaskService.GetTask returns known *Task
  - Assert: JSON output matches expected structure

TestRunTaskList_WithFilters_PassesFiltersToService
  - Mock: stubTaskService.ListTasks captures input filters
  - Assert: EpicKey, FeatureKey, Status filters correctly populated from args/flags
```

**Coverage Target**: Maintain or improve existing CLI test coverage; new tests cover argument passing and output format

---

## 4. Integration Scenarios

### Scenario 1: Full Task Lifecycle via Refactored Commands (CLI → Service → Repository → DB)

**User Journey**: An AI Developer Agent picks up a task, starts it, completes it, and gets it approved — all through refactored thin-wrapper commands.

**Integration Points**: CLI parsing → `GetTaskService()` accessor → `TaskService` → `TaskRepository` → SQLite

**Steps**:
1. `shark task next --agent=developer --json` → Returns next available task
2. `shark task start <key> --agent=developer-agent-01` → Status transitions to `in_development`
3. `shark task complete <key> --notes="Implementation done"` → Status transitions to `ready_for_code_review`
4. `shark task approve <key> --notes="LGTM"` → Status transitions to `completed`

**Expected Results**:
- Each command exits 0
- Each command's JSON output contains the updated task status
- `shark task get <key> --json` after completion shows `status: "completed"`
- `shark history <key>` shows correct status transition history

**Cross-Feature Touchpoints**:
- Depends on TaskService methods from E15-F01/F02/F03/F04 (must be complete before this test)
- Verifies E15-F07 integration with existing `task_history` trigger in database (E15-F06 dependency)

---

### Scenario 2: Feature Progress Display after Task Completion

**User Journey**: A developer completes several tasks in a feature and then checks feature progress — all via refactored commands.

**Integration Points**: `runFeatureGet` → `GetFeatureService()` → `FeatureService.GetProgress()` → `TaskRepository`

**Steps**:
1. `shark task complete E07-F01-001 --notes="done"` (first of 4 tasks)
2. `shark task complete E07-F01-002 --notes="done"` (second of 4 tasks)
3. `shark feature get E07-F01 --json` → Should show 50% progress

**Expected Results**:
- `progress_pct` field in feature JSON equals 50.0
- No direct repository calls in `runFeatureGet` handler (validated by code inspection)
- `health` field reflects correct status based on remaining tasks

**Cross-Feature Touchpoints**:
- Depends on FeatureService progress calculation methods from E15-F05
- Verifies service accessor for FeatureService is correctly wired

---

### Scenario 3: Epic Rollup Display after Feature Completions

**User Journey**: A tech lead views epic status with all feature rollup data — handled by refactored `runEpicGet`.

**Integration Points**: `runEpicGet` → `GetEpicService()` → `EpicService.GetFeatureRollup()` → `FeatureRepository`

**Steps**:
1. Seed: Epic E07 with 3 features (2 in_progress, 1 completed)
2. `shark epic get E07 --json`

**Expected Results**:
- Response contains `feature_rollup` with counts per status
- Response contains `task_rollup` aggregated across all features
- No `for _, feature := range features` loops in `runEpicGet` handler
- JSON structure identical to pre-refactoring baseline

**Cross-Feature Touchpoints**:
- Depends on EpicService rollup methods from E15-F05
- Integration validates E15-F07 eliminates epic rollup loops from `epic.go`

---

### Scenario 4: Error Handling Consistency Across All Three Command Files

**User Journey**: A developer mistypes a task key in each of the three entity-type commands and receives consistent error behavior.

**Steps**:
1. `shark task get FAKE-001` → Should exit 1, "task not found: FAKE-001"
2. `shark feature get FAKE-F01` → Should exit 1, "feature not found: FAKE-F01"
3. `shark epic get FAKE-E99` → Should exit 1, "epic not found: FAKE-E99"
4. `shark task start E07-F01-001` when task is already `completed` → Should exit 3, workflow error message

**Expected Results**:
- Exit code 1 for all not-found scenarios (before and after refactoring)
- Exit code 3 for invalid state transitions
- Error messages are human-readable and include the entity key
- Behavior is consistent: same `handleServiceError` function used across all handlers

---

### Scenario 5: Baseline Comparison (Pre/Post Refactoring)

**User Journey**: QA validates behavior preservation for the five highest-risk commands.

**Baseline Capture** (run before each file refactoring):
```bash
./bin/shark task next --json > /tmp/baseline-task-next.json
./bin/shark task get E15-F07-001 --json > /tmp/baseline-task-get.json
./bin/shark feature get E15-F07 --json > /tmp/baseline-feature-get.json
./bin/shark feature list E15 --json > /tmp/baseline-feature-list.json
./bin/shark epic get E15 --json > /tmp/baseline-epic-get.json
```

**Post-Refactoring Validation**:
```bash
./bin/shark task next --json | diff /tmp/baseline-task-next.json -
./bin/shark task get E15-F07-001 --json | diff /tmp/baseline-task-get.json -
./bin/shark feature get E15-F07 --json | diff /tmp/baseline-feature-get.json -
./bin/shark feature list E15 --json | diff /tmp/baseline-feature-list.json -
./bin/shark epic get E15 --json | diff /tmp/baseline-epic-get.json -
```

**Expected Results**: All diffs are empty (zero byte difference). If field ordering changes, use `jq` normalized comparison:
```bash
jq -S . /tmp/baseline-task-get.json > /tmp/sorted-baseline.json
./bin/shark task get E15-F07-001 --json | jq -S . | diff /tmp/sorted-baseline.json -
```

---

## 5. Performance Approach

**NFR**: CLI command execution time must not increase by more than 10% after refactoring.
**Baseline**: `shark task next --agent=backend` ~ 50ms. Acceptable range post-refactoring: 45-55ms.

**Why Minimal Risk**: The service accessor pattern creates a new service instance per call, but this adds only nanoseconds of overhead (struct initialization + pointer assignment). The underlying repository calls and database queries are identical.

**Measurement Method**:
```bash
# Measure baseline (before refactoring)
time ./bin/shark task next --agent=backend
time ./bin/shark task next --agent=backend
time ./bin/shark task next --agent=backend
# Record median of 5 runs

# Measure post-refactoring
time ./bin/shark task next --agent=backend
# Compare medians
```

**Acceptance Gate**: If median post-refactoring time exceeds 55ms (10% over 50ms baseline), investigate before merging. Likely causes: service accessor creating duplicate DB connections, excessive object allocation in DTO construction.

---

## Quality Gates (Exit Gate Checklist)

This feature may not advance to `ready_for_task_generation` until ALL of the following pass:

### Structural Gates (Code Inspection)
- [ ] `grep -c "repository\.New" internal/cli/commands/task.go` returns 0
- [ ] `grep -c "repository\.New" internal/cli/commands/feature.go` returns 0
- [ ] `grep -c "repository\.New" internal/cli/commands/epic.go` returns 0
- [ ] `wc -l internal/cli/commands/task.go` ≤ 400
- [ ] `wc -l internal/cli/commands/feature.go` ≤ 350
- [ ] `wc -l internal/cli/commands/epic.go` ≤ 300

### Prerequisite Gates (Service Accessor Readiness)
- [ ] No TODO comments indicating broken wiring in `service_accessors.go` or `services_global.go`
- [ ] `GetTaskService()` confirmed wired with `taskcreation.Creator`
- [ ] `GetEpicService()` confirmed wired with epic note adapter
- [ ] `GetFeatureService()` confirmed wired with feature note adapter

### Regression Gates
- [ ] `make test` exits 0 (run after each file refactoring: task.go, feature.go, epic.go)
- [ ] `make fmt && make lint` exits 0 for all three files
- [ ] Baseline JSON comparison passes for: `task next`, `task get`, `feature get`, `feature list`, `epic get`

### Test Coverage Gates
- [ ] Unit tests exist for all argument parsing helpers (`parseCreateTaskInput`, `parseTaskFilters`, etc.)
- [ ] Unit tests exist for `handleServiceError` covering exit codes 1, 2, and 3
- [ ] At least one mocked-service handler test exists per command file (task, feature, epic)

### Integration Gates
- [ ] End-to-end task lifecycle test passes (Scenario 1 above)
- [ ] Feature progress display test passes (Scenario 2 above)
- [ ] Error handling consistency test passes (Scenario 4 above)

---

## Traceability to Epic UAT

This feature contributes to the following epic-level acceptance scenarios (from E15 epic UAT):

| Epic Scenario | F07 Contribution |
|---------------|-----------------|
| "Developer opens task.go and finds handler in <2 minutes" | task.go ≤ 400 lines; each handler is 15-30 lines |
| "No CLI command file exceeds 500 lines" (Epic FR6) | task.go ≤400, feature.go ≤350, epic.go ≤300 |
| "HTTP API can call same logic as CLI without duplication" | Commands route through services reachable by HTTP API |
| "All existing tests pass after E15 refactoring" | Zero regression in `make test` at each phase |
| "Developer can write CLI test with mocked service" | Thin wrappers + parsing helpers enable mock-based testing |

---

## Test Execution Order

Tests must run in this sequence to match the incremental delivery plan:

**Phase 0 (Prerequisites)**:
1. AC1 tests (service accessor wiring) - MUST PASS before any command refactoring
2. Baseline capture for all five key commands

**Phase 1 (task.go)**:
3. AC2 structural tests after task.go refactoring
4. `make test` regression gate
5. Baseline JSON comparison for task commands
6. AC7 parsing helper unit tests

**Phase 2 (feature.go)**:
7. AC3 structural tests after feature.go refactoring
8. `make test` regression gate
9. Baseline JSON comparison for feature commands

**Phase 3 (epic.go)**:
10. AC4 structural tests after epic.go refactoring
11. `make test` regression gate
12. Baseline JSON comparison for epic commands

**Phase 4 (Cross-cutting)**:
13. AC5 full regression suite
14. AC6 error handler tests
15. Integration Scenarios 1-5
16. Performance measurement

---

*Test plan authored for E15-F07: CLI Commands as Thin Wrappers*
*Feature complexity: STANDARD*
*Last updated: 2026-02-18*
