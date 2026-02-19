# Test Plan: E15-F09 — Additional Entity Services

**Feature**: E15-F09 - Additional Entity Services
**Epic**: E15 - Service Layer Architecture Refactoring
**Complexity Tier**: STANDARD
**Test Plan Type**: Focused Test Plan (AC Test Matrix + API Contract Tests + Component Test Strategy + Integration Scenarios)
**Date**: 2026-02-18
**Author**: QA Agent

---

## Executive Summary

This test plan validates the implementation of `EpicService` and `FeatureService` — two fully-featured domain services that centralize all business logic for epic and feature entities — along with the CLI accessor functions (`GetEpicService`, `GetFeatureService`) that wire them for CLI commands.

**Important context**: As documented in `02-architecture.md`, all three tasks in E15-F09 were already implemented before this feature formally reached the development phase. Implementation occurred as part of E15-F05 and related work. This test plan targets the already-delivered code and verifies it meets the acceptance criteria that would have been defined at refinement time.

**User Goals This Serves**: Two core Shark personas benefit from this work:

- **AI Developer Agent**: CLI commands for epics and features now route through `EpicService` / `FeatureService`, making business logic testable with mocks rather than requiring a real database. The agent can write service tests, know where all epic/feature logic lives, and trust that CLI commands are thin wrappers.
- **Tech Lead / Developer**: Can understand the full surface area of epic and feature operations by reading one service file per entity, rather than hunting across CLI command files, repositories, and status calculation services.

**Critical Success Criteria**:
- `EpicService` exposes complete CRUD, rollup, health, impediment, and lifecycle operations with mocked-repository tests
- `FeatureService` exposes complete CRUD, progress, health, action items, and lifecycle operations with mocked-repository tests
- `GetEpicService()` and `GetFeatureService()` accessors are wired with all dependencies and usable by CLI commands
- Note repository interface mismatch resolved via adapter pattern (no `nil` passed for noteRepo)
- `make test` passes with no failures; `make lint` passes
- CLI commands using these services produce correct output for `shark epic get`, `shark epic list`, `shark feature get`, `shark feature list`

**Risk Profile**: Low-to-medium. Services are already implemented and in use. Risk is primarily in test coverage gaps (uncovered branches in CRUD and lifecycle methods) and wiring correctness of the note adapter pattern.

---

## 1. Acceptance Criteria Test Matrix

### Story 1: EpicService — Complete Epic Domain Service

**User Goal**: An AI Developer Agent navigating `epic_service.go` can find all epic business logic in one file: CRUD, rollup, health, impediment tracking, status lifecycle, and cascade operations.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC1.1** | `EpicService.GetEpic()` retrieves epic by key | Unit test with `MockEpicRepository.GetByKeyFunc` returning known epic | `key="E15"` | Returns `*models.Epic` with correct key | Key not found → `NotFoundError`; repo error → wrapped error |
| **AC1.2** | `EpicService.ListEpics()` returns filtered list | Unit test with mock repo returning multiple epics | `EpicFilters{Status: "active"}` | Returns only epics matching filter | Empty result → returns `[]*models.Epic{}` not nil; repo error propagates |
| **AC1.3** | `EpicService.CreateEpic()` validates and persists new epic | Unit test with mock repo capturing `CreateFunc` call | `CreateEpicInput{Title: "My Epic"}` | Returns `*models.Epic` with generated key | Empty title → validation error; duplicate key → repo returns error |
| **AC1.4** | `EpicService.UpdateEpic()` applies partial updates | Unit test: mock repo verifies only changed fields written | `EpicUpdates{Title: ptr("New Title")}` | Returns updated `*models.Epic`; repo `Update` called once | nil updates → no-op; key not found → `NotFoundError` |
| **AC1.5** | `EpicService.DeleteEpic()` removes epic | Unit test with mock repo verifying `Delete` called | `key="E99"` | No error returned; repo `Delete` called with correct ID | Key not found → `NotFoundError` propagated |
| **AC1.6** | `EpicService.GetFeatureRollup()` aggregates feature status counts | Unit test: mock `EpicFeatureCounter` returns features with various statuses | `epicKey="E15"` | Returns `*FeatureRollup` with correct per-status counts | Zero features → all counts 0; epic not found → error |
| **AC1.7** | `EpicService.GetTaskStatusRollup()` aggregates task counts across features | Unit test: mock returns tasks in multiple statuses | `epicKey="E15"` | Returns `map[string]int` with correct counts | All tasks same status → single map entry; no tasks → empty map |
| **AC1.8** | `EpicService.GetProgress()` calculates weighted epic progress | Unit test: mock returns features with known progress | `epicKey="E15"` | Returns `*EpicProgressInfo` with weighted and completion percentages | Zero features → 0% progress; all features complete → 100% |
| **AC1.9** | `EpicService.GetHealth()` derives health from blocked task counts | Unit test: mock `EpicTaskLister` returns blocked tasks of varying priority | `epicKey="E15"` with 0 blocked | Returns `*EpicHealthInfo{Health: "healthy"}` | 1 low-priority blocked → "warning"; 2+ blocked or high-priority blocked → "critical" |
| **AC1.10** | `EpicService.GetImpediments()` lists blocked tasks with age | Unit test: mock returns blocked tasks | `epicKey="E15"` | Returns `[]*Impediment` with task and days-blocked | `nil` taskRepo → returns empty `[]*Impediment` gracefully (not panic) |
| **AC1.11** | `EpicService.TransitionStatus()` validates and applies transition | Unit test: mock workflow validates transition; mock repo captures update | `epicKey="E15", targetStatus="active"` | Returns `*TransitionResult` with new status | Invalid transition → workflow error returned; forced transition bypasses validation |
| **AC1.12** | `EpicService.GetNextStatus()` returns the next workflow status | Unit test: mock workflow service returns next status info | `epicKey="E15"` with status "draft" | Returns `*NextStatusInfo` with target status | Terminal status → returns appropriate terminal info |
| **AC1.13** | `EpicService.CascadeStatusToFeaturesAndTasks()` bulk-updates child entities | Unit test: mock feature and task repos verify update calls | `epicID=32, featureStatus="active", taskStatus="in_progress"` | Feature and task repos each receive one update call | Zero features → no task updates needed |
| **AC1.14** | `EpicService.CompleteEpic()` validates all features complete before completing | Unit test: mock feature counter shows incomplete features | `epicKey="E15", force=false` | Returns error when features not all complete | `force=true` bypasses check; epic not found → `NotFoundError` |

---

### Story 2: FeatureService — Complete Feature Domain Service

**User Goal**: An AI Developer Agent navigating `feature_service.go` can find all feature business logic in one file: CRUD, progress tracking, health status, action items, work breakdown, enriched status display, and lifecycle operations.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC2.1** | `FeatureService.GetFeature()` retrieves feature by key | Unit test with mock repo | `key="E15-F09"` | Returns `*models.Feature` with correct key | Not found → `NotFoundError`; repo error → wrapped error |
| **AC2.2** | `FeatureService.GetFeatureByID()` retrieves feature by numeric ID | Unit test with mock repo | `id=99` | Returns `*models.Feature` with matching ID | Zero ID → handled; not found → error |
| **AC2.3** | `FeatureService.ListFeatures()` returns features with optional filter | Unit test with mock returning multiple features | `FeatureFilters{}` (no filter) | Returns all features | Status filter applied in-memory; repo error propagates |
| **AC2.4** | `FeatureService.ListFeaturesByEpicKey()` filters by epic | Unit test with mock `FeatureEpicLookup` | `epicKey="E15", statusFilter=""` | Returns only features belonging to E15 | Non-existent epic → empty list or error depending on epic lookup |
| **AC2.5** | `FeatureService.CreateFeature()` validates and persists new feature | Unit test: mock repo captures Create call | `CreateFeatureInput{EpicKey: "E15", Title: "New Feature"}` | Returns `*models.Feature` with generated key | Empty title → validation error; epic not found → error from epic lookup |
| **AC2.6** | `FeatureService.UpdateFeature()` applies partial updates | Unit test: mock repo verifies update payload | `key="E15-F09", updates={Title: ptr("Updated")}` | Returns updated `*models.Feature` | nil updates → no change; key not found → `NotFoundError` |
| **AC2.7** | `FeatureService.DeleteFeature()` removes feature | Unit test with mock repo | `key="E15-F09"` | No error; repo `Delete` called | Key not found → error propagated |
| **AC2.8** | `FeatureService.GetProgress()` returns weighted and completion progress | Unit test: mock `FeatureTaskCounter` returns task breakdown with known weights | `key="E15-F09"` with 4 tasks (2 complete, 2 todo) | `FeatureProgressInfo{WeightedPct: 50.0, CompletionPct: 50.0, Total: 4}` | Zero tasks → 0% / 0% / 0; all tasks complete → 100% / 100% |
| **AC2.9** | `FeatureService.RecalculateAndSetProgress()` persists recalculated progress to DB | Unit test: mock repo verifies `Update` called with new progress value | `featureID=99` with task breakdown available | Feature repo `Update` called once with updated progress; auto-complete fires when all tasks done | Breakdown error → returns error without updating |
| **AC2.10** | `FeatureService.GetHealth()` derives health from blocked task counts and stale approvals | Unit test: mock task counter returns specific task statuses | No blocked tasks | Returns `*FeatureHealthInfo{Health: "healthy"}` | 1 blocked → "warning"; 2+ blocked OR stale approval > 3 days → "critical" |
| **AC2.11** | `FeatureService.GetWorkBreakdown()` categorizes tasks by responsibility | Unit test: mock task counter returns tasks in agent/human/qa_team statuses | Feature with mixed-responsibility tasks | Returns `*WorkBreakdown` with correct counts per responsibility | Empty feature → all counts 0; `nil` taskRepo → returns empty breakdown |
| **AC2.12** | `FeatureService.GetActionItems()` returns tasks requiring human/approval action | Unit test: mock returns tasks in `ready_for_*` statuses | Feature with tasks in `ready_for_approval` | Returns `*FeatureActionItems` with tasks grouped by status | No actionable tasks → returns empty `FeatureActionItems`; `nil` taskRepo → returns empty gracefully |
| **AC2.13** | `FeatureService.GetEnrichedTaskStatusBreakdown()` adds color/phase metadata to counts | Unit test: mock workflow service provides status metadata | Feature with tasks in `todo`, `in_progress`, `completed` | Returns `[]workflow.StatusCount` with color and phase populated | Empty feature → empty slice; workflow config error → returns error |
| **AC2.14** | `FeatureService.GetTaskStatusBreakdown()` returns raw counts per status | Unit test with mock task counter | Feature with 2 todo, 1 in_progress, 3 completed | Returns `map[string]int{"todo": 2, "in_progress": 1, "completed": 3}` | Single status → map with one key; no tasks → empty map |
| **AC2.15** | `FeatureService.TransitionStatus()` validates and applies status transition | Unit test: mock workflow, mock repo captures update | `featureKey="E15-F09", targetStatus="active"` | Returns `*TransitionResult` with new status and optional action | Invalid transition → workflow error; `force=true` → bypasses validation |
| **AC2.16** | `FeatureService.GetNextStatus()` returns the next logical workflow status | Unit test with mock workflow | Feature in "draft" status | Returns `*NextStatusInfo` with next status | Terminal status → terminal indicator returned |
| **AC2.17** | `FeatureService.CompleteFeature()` validates all tasks complete before completing | Unit test: mock task counter shows incomplete tasks | `featureKey="E15-F09", force=false` | Returns error when tasks not all complete | `force=true` → bypasses task completion check |
| **AC2.18** | `FeatureService.CascadeFeatureStatusToTasks()` bulk-updates all tasks in feature | Unit test: mock task repo verifies bulk update | `featureKey="E15-F09", targetTaskStatus="completed"` | Task repo receives batch update call | Zero tasks → no error, no update |

---

### Story 3: CLI Accessor Wiring

**User Goal**: Any CLI command file can call `cli.GetEpicService()` or `cli.GetFeatureService()` and receive a fully-wired, immediately-usable service instance with all dependencies resolved.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC3.1** | `GetEpicService()` returns non-nil `*services.EpicService` | Call `cli.GetEpicService()` with test DB available | Valid test database | Non-nil pointer, no panic | Verify it's callable: `svc.GetEpic(ctx, "E15")` returns result or domain error (not nil pointer panic) |
| **AC3.2** | `GetFeatureService()` returns non-nil `*services.FeatureService` | Call `cli.GetFeatureService()` with test DB available | Valid test database | Non-nil pointer, no panic | Verify it's callable: `svc.GetFeature(ctx, "E15-F09")` returns result or domain error |
| **AC3.3** | Note repository adapter for epics translates `string` entityType to `models.EntityType` | Unit test of `epicNoteAdapter.CreateRejectionNote` directly | `entityType="epic", documentPath=""` | Underlying `EntityNoteRepository.CreateRejectionNote` called with `models.EntityType("epic")` and `nil` docPath | Non-empty `documentPath` → `*string` non-nil passed to underlying repo |
| **AC3.4** | Note repository adapter for features translates correctly | Unit test of `featureNoteAdapter.CreateRejectionNote` directly | `entityType="feature", documentPath="/path/to/doc.md"` | `models.EntityType("feature")` and non-nil `*string` `"/path/to/doc.md"` passed to repo | Empty `documentPath` → `nil` `*string` |
| **AC3.5** | `GetEpicService()` wires `featureRepo` and `taskRepo` for rollup operations | Integration test: call `svc.GetFeatureRollup(ctx, "E15")` via `GetEpicService()` | Known epic with features | Returns `*FeatureRollup` with correct counts (not nil pointer panic from missing featureRepo) | Depends on test database having E15 with features |
| **AC3.6** | `GetEpicService()` wires `docRepo` via `SetDocRepo()` | Code inspection: verify `svc.SetDocRepo(docRepo)` called after construction | `service_accessors.go` source | Line `svc.SetDocRepo(docRepo)` present and executes without error | `docRepo` nil would silently disable document display; verify it's non-nil |
| **AC3.7** | `GetFeatureService()` uses `WithRelationships` constructor (not basic constructor) | Code inspection: verify `NewFeatureServiceWithRelationships` called | `service_accessors.go` source | `NewFeatureServiceWithRelationships` present in `GetFeatureService()` body | This ensures `docRepo` and `epicLookupRepo` are wired |
| **AC3.8** | Both accessors panic on database failure | Unit test: override `GetDB` to return error in test | DB returns error | Panic message contains "failed to get database" | Panic is correct behavior for CLI entry points (fail-fast) |
| **AC3.9** | No TODO comments indicating broken wiring remain in `service_accessors.go` | `grep "TODO" internal/cli/service_accessors.go` | File content | Zero TODO matches | Any remaining TODOs must be approved deferred work with a tracked issue |

---

### Story 4: Architecture Compliance

**User Goal**: Epic E15 is a refactoring with a firm architectural rule — no business logic in repository files and no repository instantiation in CLI command files. E15-F09 must not add new violations.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC4.1** | `epic_service.go` contains no direct SQL queries or `r.db.Exec` calls | `grep -n "\.db\." internal/services/epic_service.go` | `epic_service.go` | Zero matches (all data access via `s.repo.*` interface calls) | `s.workflowSvc.*` and `s.noteRepo.*` calls are acceptable |
| **AC4.2** | `feature_service.go` contains no direct SQL queries | `grep -n "\.db\." internal/services/feature_service.go` | `feature_service.go` | Zero matches | Same exception as AC4.1 |
| **AC4.3** | Workflow validation is scoped to correct entity level in constructors | Code inspection: verify `workflowSvc.ForLevel(workflow.LevelEpic)` in `NewEpicService`; `workflowSvc.ForLevel(workflow.LevelFeature)` in `NewFeatureService` | Constructor bodies | Both constructors scope workflow to entity level | Using global config instead of scoped config would break status validation for epic/feature |
| **AC4.4** | `make test` passes after all E15-F09 changes | `make test` | Full test suite | Exit code 0, zero test failures | Run specifically against `./internal/services/...` for faster feedback during development |
| **AC4.5** | `make fmt && make lint` passes | `make fmt && make lint` | All Go source | Exit code 0, no lint errors | Unused imports, missing godoc, naming violations all block this gate |

---

## 2. API Contract Test Cases

These contracts define the exact interfaces between `EpicService` and `FeatureService` and their callers. Each contract is verifiable by compilation (interface satisfaction) and unit tests.

### EpicService Method Contracts

| Contract ID | Method | Required Inputs | Return Types | Error Conditions |
|-------------|--------|----------------|--------------|------------------|
| **E-C1** | `GetEpic(ctx, key string)` | Non-empty key | `*models.Epic, error` | `NotFoundError` if key not in DB; wrapped repo error otherwise |
| **E-C2** | `ListEpics(ctx, EpicFilters)` | `EpicFilters` (can be zero-value) | `[]*models.Epic, error` | Repo error propagates; nil slice ≠ error |
| **E-C3** | `CreateEpic(ctx, CreateEpicInput)` | `CreateEpicInput{Title: non-empty}` | `*models.Epic, error` | Validation error if title empty; repo error if key conflict |
| **E-C4** | `UpdateEpic(ctx, key, EpicUpdates)` | Non-empty key; at least one non-nil update field | `*models.Epic, error` | `NotFoundError` if key not found; validation error for invalid field values |
| **E-C5** | `DeleteEpic(ctx, key)` | Non-empty key | `error` | `NotFoundError` if key not found |
| **E-C6** | `GetFeatureRollup(ctx, key)` | Valid epic key | `*FeatureRollup, error` | `NotFoundError` if epic not found; graceful empty rollup if no features |
| **E-C7** | `GetTaskStatusRollup(ctx, key)` | Valid epic key | `map[string]int, error` | `NotFoundError` if epic not found; empty map (not nil) if no tasks |
| **E-C8** | `GetProgress(ctx, key)` | Valid epic key | `*EpicProgressInfo, error` | `NotFoundError` if key not found; 0.0 progress for epics with no features |
| **E-C9** | `GetHealth(ctx, key)` | Valid epic key | `*EpicHealthInfo, error` | `NotFoundError` if key not found; "healthy" baseline when taskRepo is nil |
| **E-C10** | `GetImpediments(ctx, key)` | Valid epic key | `[]*Impediment, error` | Returns empty slice (not nil) when no blocked tasks; nil taskRepo → empty slice |
| **E-C11** | `TransitionStatus(ctx, key, targetStatus, opts)` | Valid key, valid target status | `*TransitionResult, error` | Workflow transition error if invalid; `NotFoundError` if key not found |
| **E-C12** | `GetNextStatus(ctx, key)` | Valid epic key | `*NextStatusInfo, error` | `NotFoundError` if key not found; terminal info if already at final status |

### FeatureService Method Contracts

| Contract ID | Method | Required Inputs | Return Types | Error Conditions |
|-------------|--------|----------------|--------------|------------------|
| **F-C1** | `GetFeature(ctx, key)` | Non-empty feature key | `*models.Feature, error` | `NotFoundError` if not found |
| **F-C2** | `GetFeatureByID(ctx, id)` | Valid int64 ID | `*models.Feature, error` | `NotFoundError` if no feature with that ID |
| **F-C3** | `ListFeatures(ctx, FeatureFilters)` | `FeatureFilters` (zero-value allowed) | `[]*models.Feature, error` | Empty slice not error; repo error propagates |
| **F-C4** | `ListFeaturesByEpicKey(ctx, epicKey, statusFilter)` | Valid epic key; optional status filter | `[]*models.Feature, error` | Epic not found → error from epic lookup; empty filter → all features returned |
| **F-C5** | `CreateFeature(ctx, CreateFeatureInput)` | `CreateFeatureInput{EpicKey, Title: non-empty}` | `*models.Feature, error` | Validation error for empty title; epic not found → error |
| **F-C6** | `UpdateFeature(ctx, key, FeatureUpdates)` | Valid key; at least one update | `*models.Feature, error` | `NotFoundError` if not found; validation error for invalid priority |
| **F-C7** | `DeleteFeature(ctx, key)` | Valid key | `error` | `NotFoundError` propagated |
| **F-C8** | `GetProgress(ctx, key)` | Valid feature key | `*FeatureProgressInfo, error` | Returns `(0.0, 0.0, 0, nil)` for feature with no tasks |
| **F-C9** | `RecalculateAndSetProgress(ctx, featureID)` | Valid feature ID | `error` | Breakdown error → returns error without DB write |
| **F-C10** | `GetHealth(ctx, key)` | Valid feature key | `*FeatureHealthInfo, error` | `NotFoundError` if not found; "healthy" for features with no tasks |
| **F-C11** | `GetWorkBreakdown(ctx, key)` | Valid feature key | `*WorkBreakdown, error` | Empty breakdown for features with no tasks; nil taskRepo → empty breakdown |
| **F-C12** | `GetActionItems(ctx, key)` | Valid feature key | `*FeatureActionItems, error` | Empty items for features with no actionable tasks; nil taskRepo → empty |
| **F-C13** | `GetEnrichedTaskStatusBreakdown(ctx, key)` | Valid feature key | `[]workflow.StatusCount, error` | Workflow config error propagates; empty slice for no tasks |
| **F-C14** | `GetTaskStatusBreakdown(ctx, key)` | Valid feature key | `map[string]int, error` | Empty map (not nil) for no tasks; repo error propagates |
| **F-C15** | `TransitionStatus(ctx, key, targetStatus, opts)` | Valid key and target status | `*TransitionResult, error` | Workflow transition error if invalid; `NotFoundError` if key not found |
| **F-C16** | `GetNextStatus(ctx, key)` | Valid feature key | `*NextStatusInfo, error` | `NotFoundError` if not found; terminal info at final status |

### CLI Accessor Contracts

| Contract ID | Accessor | Required Wiring | Verification | Pass Condition |
|-------------|----------|-----------------|--------------|----------------|
| **ACC-E1** | `cli.GetEpicService()` | EpicRepository, workflowSvc (scoped to LevelEpic), epicNoteAdapter, FeatureRepository, TaskRepository, DocumentRepository (via SetDocRepo) | Call `svc.GetEpic(ctx, "E15")` on test DB | Returns `*models.Epic` or domain error; no nil pointer panic |
| **ACC-E2** | `epicNoteAdapter` | Wraps `*repository.EntityNoteRepository` | Call `CreateRejectionNote` with `entityType="epic"` | `models.EntityType("epic")` passed to underlying repo; `""` docPath → `nil *string` |
| **ACC-F1** | `cli.GetFeatureService()` | FeatureRepository, workflowSvc (scoped to LevelFeature), featureNoteAdapter, TaskRepository, DocumentRepository, EpicRepository (epicLookupRepo) | Call `svc.GetFeature(ctx, "E15-F09")` on test DB | Returns `*models.Feature` or domain error; no nil pointer panic |
| **ACC-F2** | `featureNoteAdapter` | Wraps `*repository.EntityNoteRepository` | Call `CreateRejectionNote` with non-empty docPath | Non-nil `*string` passed to underlying repo |

---

## 3. Component Test Strategy

### Component 1: EpicService Unit Tests (`internal/services/epic_service_test.go`)

**Purpose**: Verify EpicService business logic in complete isolation from the database. All repository and workflow dependencies mocked.

**Current state**: `epic_service_test.go` exists at 1,299 lines with coverage of status transitions, rollup, health, progress, and action resolution. **Known gaps to verify**: CRUD operations (Create, Update, Delete), `CompleteEpic`, `CascadeStatusToFeaturesAndTasks`, `RenameKey`, `RecalculateStatus`.

**Test Approach**: Mock all five interfaces — `EpicRepository`, `EpicNoteRepository`, `EpicFeatureCounter`, `EpicTaskLister`, and `config.DocumentRepository` — via function-field mock structs defined in `internal/services/mocks_test.go` or within `epic_service_test.go`.

**Key Test Cases to Add or Verify Exist**:

| Test Case Name | Mock Setup | Key Assertion |
|----------------|------------|---------------|
| `TestEpicService_CreateEpic_Success` | `CreateFunc` captures input epic; `GetByKeyFunc` returns nil (key not taken) | Returned epic has non-empty key; `Create` called once |
| `TestEpicService_CreateEpic_ValidationError` | No mock needed | Empty title → validation error before repo call |
| `TestEpicService_UpdateEpic_Success` | `GetByKeyFunc` returns existing epic; `UpdateFunc` captures update | `Update` called once; returned epic reflects changes |
| `TestEpicService_UpdateEpic_NotFound` | `GetByKeyFunc` returns `NotFoundError` | Error propagated; `Update` not called |
| `TestEpicService_DeleteEpic_Success` | `DeleteFunc` called; `GetByKeyFunc` returns existing epic | `Delete` called with correct ID; no error returned |
| `TestEpicService_CompleteEpic_AllFeaturesComplete` | `EpicFeatureCounter` returns features all in terminal status | Completion succeeds; repo `Update` called to set completed status |
| `TestEpicService_CompleteEpic_FeaturesIncomplete` | `EpicFeatureCounter` returns features with non-terminal status | Error returned; `Update` not called (when `force=false`) |
| `TestEpicService_CompleteEpic_Force` | Features incomplete but `force=true` | Completion succeeds despite incomplete features |
| `TestEpicService_CascadeStatusToFeaturesAndTasks` | Mock `EpicFeatureCounter` returns 2 features; mock task repo | Feature and task repos each called with new status; returns nil error |
| `TestEpicService_GetImpediments_NilTaskRepo` | Construct service with `taskRepo=nil` | Returns `[]*Impediment{}`, no panic |

**Coverage Target**: 80%+ on `epic_service.go` statements; 100% on error paths of all CRUD methods.

**Test file location**: `internal/services/epic_service_test.go` (extend existing).

---

### Component 2: FeatureService Unit Tests (`internal/services/feature_service_test.go`)

**Purpose**: Verify FeatureService business logic in isolation. All repository and workflow dependencies mocked.

**Current state**: `feature_service_test.go` exists at 1,530 lines with coverage of status transitions, progress, health, work breakdown, action items, and status breakdown. **Known gaps to verify**: CRUD operations (Create, Update, Delete), `CompleteFeature` with force/no-force, `CascadeFeatureStatusToTasks`, `SetFeatureStatusOverride`, `UpdateFeatureKey`.

**Test Approach**: Mock `FeatureRepository`, `FeatureNoteRepository`, `FeatureTaskCounter`, `FeatureEpicLookup`, `DocumentRepository`, and `FeatureRelationshipRepository` (nil for CLI context).

**Key Test Cases to Add or Verify Exist**:

| Test Case Name | Mock Setup | Key Assertion |
|----------------|------------|---------------|
| `TestFeatureService_CreateFeature_Success` | `EpicLookupFunc` returns epic; `CreateFunc` captures feature | Feature key generated; `Create` called once; returned feature has epic association |
| `TestFeatureService_CreateFeature_EpicNotFound` | `EpicLookupFunc` returns `NotFoundError` | Error propagated; `Create` not called |
| `TestFeatureService_CreateFeature_EmptyTitle` | No mock needed (validation before repo calls) | Validation error returned |
| `TestFeatureService_UpdateFeature_Success` | `GetByKeyFunc` returns feature; `UpdateFunc` captures update | `Update` called once; title change reflected |
| `TestFeatureService_UpdateFeature_NotFound` | `GetByKeyFunc` returns `NotFoundError` | `NotFoundError` propagated |
| `TestFeatureService_DeleteFeature_Success` | `GetByKeyFunc` returns feature; `DeleteFunc` captures call | `Delete` called once; no error |
| `TestFeatureService_CompleteFeature_AllTasksDone` | `TaskCounter` returns all completed tasks | Completion succeeds; repo `Update` called |
| `TestFeatureService_CompleteFeature_TasksIncomplete` | `TaskCounter` returns some incomplete tasks, `force=false` | Error returned; `Update` not called |
| `TestFeatureService_CascadeFeatureStatusToTasks` | Mock task repo verifies bulk update | Task repo receives update call with correct status |
| `TestFeatureService_RecalculateAndSetProgress_AutoComplete` | All tasks completed (100%) | `RecalculateAndSetProgress` triggers feature completion |
| `TestFeatureService_GetActionItems_NilTaskRepo` | Construct service with `taskRepo=nil` | Returns empty `*FeatureActionItems`, no panic |
| `TestFeatureService_GetWorkBreakdown_NilTaskRepo` | Construct service with `taskRepo=nil` | Returns empty `*WorkBreakdown`, no panic |

**Coverage Target**: 80%+ on `feature_service.go` statements; 100% on error paths of all CRUD methods.

**Test file location**: `internal/services/feature_service_test.go` (extend existing).

---

### Component 3: CLI Accessor Wiring (`internal/cli/service_accessors.go`)

**Purpose**: Verify that the accessor functions (`GetEpicService`, `GetFeatureService`) return correctly wired services and the note adapters translate the interface mismatch correctly.

**Test Approach**: Two sub-approaches:
1. **Unit tests** for the note adapter types directly (pure unit, no DB needed)
2. **Integration tests** using the test database to verify end-to-end accessor → service → repository chain

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| `epicNoteAdapter` translates empty documentPath to nil | Unit test adapter directly | `nil *string` passed to underlying `EntityNoteRepository.CreateRejectionNote` |
| `epicNoteAdapter` translates non-empty documentPath to `*string` | Unit test adapter | Non-nil `*string` with correct value passed |
| `featureNoteAdapter` same translation behavior | Unit test | Same as epic adapter |
| `GetEpicService()` returns callable service | Integration test with test DB | Call `svc.GetEpic(ctx, key)` → returns `*models.Epic` or `NotFoundError`, no nil pointer panic |
| `GetFeatureService()` returns callable service | Integration test with test DB | Call `svc.GetFeature(ctx, key)` → returns `*models.Feature` or `NotFoundError`, no nil pointer panic |
| `GetEpicService()` wires taskRepo (for `GetImpediments`) | Integration test | `svc.GetImpediments(ctx, "E15")` returns non-error result |
| `GetEpicService()` wires docRepo (for `GetRelatedDocuments`) | Code inspection + integration test | `svc.GetRelatedDocuments(ctx, epicID)` returns result without nil pointer panic |
| `GetFeatureService()` uses `WithRelationships` constructor | Code inspection | `NewFeatureServiceWithRelationships` in body of `GetFeatureService()` |

**Coverage Target**: 100% of adapter methods; `GetEpicService` and `GetFeatureService` each exercised at least once.

**Test file location**: `internal/cli/service_accessors_test.go` (create if not exists).

---

## 4. Integration Scenarios

### Scenario 1: Epic Get Command — End-to-End via EpicService

**User Journey**: A developer or AI agent runs `shark epic get E15 --json` and receives full epic details including feature rollup, task rollup, progress, and health.

**Integration Points**: `runEpicGet` (CLI handler) → `cli.GetEpicService()` → `EpicService.GetEpic()` + `EpicService.GetFeatureRollup()` + `EpicService.GetProgress()` → repository → SQLite

**Test Steps**:
```bash
./bin/shark epic get E15 --json
```

**Expected Results**:
- Exit code 0
- JSON response contains: `key`, `title`, `status`, feature rollup (counts by status), progress percentage
- No `repository.New*` calls present in `epic.go` handler (code inspection)
- Response is structurally identical to pre-E15-F09 baseline (if baseline was captured)

**Cross-Feature Touchpoints**:
- Depends on `GetEpicService()` from T-E15-F09-003 (accessor wiring)
- Validates E15-F09 integration with epic CLI commands from E15-F07

---

### Scenario 2: Feature Get Command — Progress, Health, and Action Items

**User Journey**: A developer runs `shark feature get E15-F09 --json` and receives progress percentage, health status, and any action items (tasks needing attention).

**Integration Points**: `runFeatureGet` → `cli.GetFeatureService()` → `FeatureService.GetFeature()` + `FeatureService.GetProgress()` + `FeatureService.GetHealth()` + `FeatureService.GetActionItems()` → repository → SQLite

**Test Steps**:
```bash
./bin/shark feature get E15-F09
./bin/shark feature get E15-F09 --json
```

**Expected Results**:
- Human-readable output shows progress bar, health indicator (healthy/warning/critical), and action items section
- JSON output includes `progress_pct`, `health`, and action items array
- No nil pointer panics from missing service dependencies
- `progress_pct` value reflects actual task completion ratio for E15-F09

**Cross-Feature Touchpoints**:
- Depends on `GetFeatureService()` from T-E15-F09-003
- Validates progress calculation migrated from repository to `FeatureService` (from E15-F06)

---

### Scenario 3: Feature List with Epic Filter

**User Journey**: An AI Developer Agent lists all features in epic E15 to understand feature breakdown and status.

**Integration Points**: `runFeatureList` → `cli.GetFeatureService()` → `FeatureService.ListFeaturesByEpicKey()` → `FeatureEpicLookup` → repository

**Test Steps**:
```bash
./bin/shark feature list E15 --json
./bin/shark feature list E15
```

**Expected Results**:
- Returns all features for epic E15 (at least F01 through F12)
- Each feature in JSON array contains: `key`, `title`, `status`, `progress_pct`
- Human-readable table shows feature keys, titles, statuses, and progress
- Filtering to non-existent epic returns empty list or domain error (not nil pointer panic)

**Cross-Feature Touchpoints**:
- Validates `FeatureEpicLookup` interface is correctly wired in `GetFeatureService()`

---

### Scenario 4: Epic Rollup — After Task Status Changes

**User Journey**: After completing several tasks, a developer checks epic progress to see if it has updated.

**Integration Points**: `shark task complete` (transitions task) → `shark epic get E15 --json` → `EpicService.GetProgress()` → calculates from feature progress → task counts

**Test Steps**:
1. Record baseline: `./bin/shark epic get E15 --json | jq '.feature_rollup'`
2. Complete a task: `./bin/shark task complete <task-key> --notes="done"`
3. Check rollup: `./bin/shark epic get E15 --json | jq '.feature_rollup'`

**Expected Results**:
- Feature rollup counts reflect updated task status after step 2
- Progress percentage increases
- Health may change if blocked tasks were resolved
- No stale data from caching (all calculations are on-demand)

**Cross-Feature Touchpoints**:
- Validates `EpicService.GetFeatureRollup()` and `GetTaskStatusRollup()` against live task data
- Integration between TaskService (step 2) and EpicService (step 3) via shared repository layer

---

### Scenario 5: Note Adapter — Rejection Note Creation

**User Journey**: A product owner rejects a feature in approval status. A rejection note must be recorded with the entity type, document path, and reason.

**Integration Points**: Feature status transition with rejection → `FeatureService.TransitionStatus()` → `FeatureNoteRepository.CreateRejectionNote()` → `featureNoteAdapter` → `repository.EntityNoteRepository`

**Test Steps** (integration test, not CLI):
```go
svc := cli.GetFeatureService()
result, err := svc.TransitionStatus(ctx, "E15-F09", "blocked", TransitionOptions{
    Reason: "Missing acceptance criteria",
    RejectedBy: "product-owner",
})
```

**Expected Results**:
- `result` non-nil, status updated to "blocked"
- Database has a new row in `entity_notes` table with `entity_type="feature"`, `reason="Missing acceptance criteria"`
- `featureNoteAdapter` correctly translated `string` entityType to `models.EntityType` (verified via DB inspection or mock capture)

**Cross-Feature Touchpoints**:
- Validates the adapter pattern from Decision 3 in `02-architecture.md`
- Confirms `featureNoteAdapter.repo` is properly initialized in `GetFeatureService()`

---

## 5. Performance Approach

**NFR**: Service constructor overhead from `GetEpicService()` / `GetFeatureService()` must not noticeably degrade CLI command response time vs. direct repository access.

**Why Low Risk**: Service constructors do only struct initialization and pointer assignment — nanoseconds of work. The DB connection (`GetDB`) and workflow service (`GetWorkflowService`) are global singletons already initialized. Repository objects (`NewEpicRepository(db)`, etc.) are lightweight wrappers around the shared DB connection.

**Measurement Method** (if regression suspected):
```bash
# Measure feature get command
time ./bin/shark feature get E15-F09 --json
time ./bin/shark feature get E15-F09 --json
time ./bin/shark feature get E15-F09 --json
# Record median of 5 runs
```

**Acceptance Gate**: If median response time for `shark feature get` or `shark epic get` exceeds 200ms, investigate before merging. Likely cause: `GetFeatureService()` creating a new `DocumentRepository` that performs filesystem I/O during construction. Target: under 100ms for both commands.

---

## Quality Gates (Exit Gate Checklist)

This feature may not advance to `ready_for_task_generation` until ALL of the following pass:

### Architecture Compliance Gates
- [ ] `grep -c "\.db\." internal/services/epic_service.go` returns 0 (no direct DB calls)
- [ ] `grep -c "\.db\." internal/services/feature_service.go` returns 0 (no direct DB calls)
- [ ] `grep -n "TODO" internal/cli/service_accessors.go` returns 0 (no broken wiring markers)
- [ ] `NewEpicService` or `NewEpicServiceWithRelationships` constructors call `workflowSvc.ForLevel(workflow.LevelEpic)`
- [ ] `NewFeatureService` or `NewFeatureServiceWithRelationships` constructors call `workflowSvc.ForLevel(workflow.LevelFeature)`

### Service Accessor Completeness Gates
- [ ] `GetEpicService()` wires: EpicRepository, FeatureRepository, TaskRepository, epicNoteAdapter, workflowSvc, docRepo (via `SetDocRepo`)
- [ ] `GetFeatureService()` wires: FeatureRepository, TaskRepository, docRepo, epicLookupRepo (EpicRepository), featureNoteAdapter, workflowSvc
- [ ] Neither accessor passes `nil` for noteRepo (both must use adapter)
- [ ] `epicNoteAdapter.CreateRejectionNote` translates `string` entityType → `models.EntityType`; empty docPath → nil `*string`
- [ ] `featureNoteAdapter.CreateRejectionNote` same translation behavior

### Test Coverage Gates
- [ ] Unit tests for `epicNoteAdapter` and `featureNoteAdapter` exist covering both `""` and non-empty documentPath
- [ ] `epic_service_test.go` has tests for: CreateEpic, UpdateEpic, DeleteEpic, CompleteEpic (both force=true and force=false), CascadeStatusToFeaturesAndTasks, GetImpediments with nil taskRepo
- [ ] `feature_service_test.go` has tests for: CreateFeature, UpdateFeature, DeleteFeature, CompleteFeature (both force), CascadeFeatureStatusToTasks, GetActionItems with nil taskRepo, GetWorkBreakdown with nil taskRepo

### Regression Gates
- [ ] `make test` exits 0 (run against full suite)
- [ ] `make fmt && make lint` exits 0
- [ ] `./bin/shark epic get E15 --json` returns valid JSON with feature rollup fields
- [ ] `./bin/shark feature get E15-F09 --json` returns valid JSON with progress and health fields
- [ ] `./bin/shark feature list E15 --json` returns array of features (not error)
- [ ] `./bin/shark epic list --json` returns array of epics (not error)

### Integration Gates
- [ ] Scenario 1 (Epic Get end-to-end) passes
- [ ] Scenario 2 (Feature Get with progress/health) passes
- [ ] Scenario 3 (Feature List with epic filter) passes

---

## Traceability to Epic UAT

This feature contributes to the following epic-level acceptance scenarios from E15:

| Epic Requirement | E15-F09 Contribution |
|-----------------|---------------------|
| **FR3: FeatureService Implementation** — `FeatureService` defines all feature operations (CRUD, progress, health, rollups, lifecycle) | `FeatureService` at 1,122 lines with full method set; all feature CLI commands route through it |
| **FR4: EpicService Implementation** — `EpicService` defines all epic operations (CRUD, rollups, impediments, progress, lifecycle) | `EpicService` at 1,068 lines with full method set; all epic CLI commands route through it |
| **FR1: Service Layer Architecture** — services receive repositories via DI; own all business logic | Two-constructor-variant pattern with explicit DI; no direct SQL in services |
| "HTTP API can call same logic as CLI without duplication" | Both services are entry-point agnostic; `cmd/server/services.go` wires them same way for HTTP |
| "Developer can write CLI tests with mocked services" | Services accept interface repos; mock-repo unit tests cover all critical business logic paths |
| "All existing tests pass after E15 refactoring" | `make test` gate verifies no regressions introduced |

---

## Test Execution Order

**Phase 1 (Pre-Development — verify existing tests pass)**:
1. `make test` — establish baseline pass state
2. `make fmt && make lint` — verify no pre-existing quality issues

**Phase 2 (Unit Tests — extend existing test files)**:
3. Add missing CRUD tests to `epic_service_test.go` (AC1.1-AC1.5, AC1.13-AC1.14)
4. Add missing CRUD tests to `feature_service_test.go` (AC2.1-AC2.7, AC2.17-AC2.18)
5. Add note adapter unit tests to `service_accessors_test.go` (AC3.3-AC3.4)
6. `make test ./internal/services/...` — verify new tests pass

**Phase 3 (Integration Tests)**:
7. Add accessor integration tests to `service_accessors_test.go` (AC3.1-AC3.2, AC3.5-AC3.7)
8. `make test ./internal/cli/...` — verify accessor tests pass

**Phase 4 (End-to-End Validation)**:
9. Execute Integration Scenario 1 (Epic Get)
10. Execute Integration Scenario 2 (Feature Get)
11. Execute Integration Scenario 3 (Feature List)
12. Execute Architecture Compliance gates (AC4.1-AC4.5)
13. `make test` — full suite final check

---

*Test plan authored for E15-F09: Additional Entity Services*
*Feature complexity: STANDARD*
*Last updated: 2026-02-18*
