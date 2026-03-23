# Test Plan: E07-F36 — Split Large Repository Package into Entity-Specific Sub-Packages

**Feature**: E07-F36
**Date**: 2026-03-22
**Status**: Ready for development

---

## Overview

This test plan covers the refactoring of `internal/repository` from a single monolithic package (79 files, ~30k lines) into entity-specific sub-packages with a backward-compatible type alias shim. The plan traces every acceptance criterion in spec.md to at least one test case. Tests use real databases (repository tests) and follow existing patterns in `tracing_test.go`, `key_lookup_test.go`, and `internal/test/testdb.go`.

---

## 1. AC Test Matrix

### REQ-F-001: Extract shared utilities to `repoutil/`

**AC**: `internal/repository/repoutil/` contains `key_lookup.go`, `order_resequence.go`, `tracing.go`
**AC**: Exported functions: `ContainsHyphen`, `IsNumeric`, `SplitAtFirstHyphen`, `SplitAtNthHyphen`, `ResequenceOrders`, `NewTracer`, `RecordSpanError`
**AC**: All existing tests pass from new location
**AC**: No code duplication between root package and `repoutil`

---

**TC-F001-1: Key lookup functions are exported from repoutil**

- Setup: `repoutil` package created with `key_lookup.go`
- Input: Call `repoutil.ContainsHyphen("E07-F01")`, `repoutil.IsNumeric("123")`, `repoutil.SplitAtFirstHyphen("E07-F01-001")`, `repoutil.SplitAtNthHyphen("T-E07-F01-001", 2)`
- Expected: Functions compile and return correct results (true, true, ["E07", "F01-001"], ["T-E07", "F01-001"])
- Test file: `internal/repository/repoutil/key_lookup_test.go` (moved from `key_lookup_test.go`)
- Edge cases:
  - Empty string input: `ContainsHyphen("")` returns false, `IsNumeric("")` returns false
  - String with no hyphen: `SplitAtFirstHyphen("E07")` returns `["E07", ""]` or single-element result
  - `SplitAtNthHyphen` with N larger than available hyphens

**TC-F001-2: ResequenceOrders exported from repoutil**

- Setup: `repoutil` package created with `order_resequence.go`
- Input: Create a slice of items implementing `repoutil.OrderedItem`, call `repoutil.ResequenceOrders(items, 2, 5)` — insert at position 2 within a 5-item sequence
- Expected: Items returned with correct integer order values, no gaps
- Test file: `internal/repository/repoutil/order_resequence_test.go` (moved)
- Edge cases:
  - Empty item list
  - Single item list
  - Insert position at beginning (0) and end (len)
  - Re-sequence with duplicate order values in input

**TC-F001-3: NewTracer and RecordSpanError exported from repoutil**

- Setup: `repoutil` package created with `tracing.go`; configure in-memory OTel `TracerProvider` identical to pattern in `tracing_test.go`
- Input: `tracer := repoutil.NewTracer("test/package")` then create a span; call `repoutil.RecordSpanError(span, errors.New("boom"))`
- Expected: Span created under instrumentation name `"test/package"`; span status is `codes.Error`; span has one event from `RecordError`
- Test file: `internal/repository/repoutil/tracing_test.go` (moved)
- Edge cases:
  - `RecordSpanError(nil, err)` — must not panic
  - `RecordSpanError(span, nil)` — span status must remain unchanged (not set to Error)

**TC-F001-4: Root package no longer defines duplicate utilities**

- Setup: Run `go vet ./internal/repository/...` after migration
- Input: Inspect root package for functions named `ContainsHyphen`, `IsNumeric`, `SplitAtFirstHyphen`, `SplitAtNthHyphen`, `resequenceOrders`
- Expected: None of these identifiers exist in `package repository` root files
- Verification method: `grep -rn "func ContainsHyphen\|func IsNumeric\|func SplitAt\|func resequenceOrders" internal/repository/*.go` returns empty
- Edge cases: Check `_test.go` files separately — test helpers may duplicate names for test-local use (acceptable)

---

### REQ-F-002: Activate dbconn package as the canonical DB type

**AC**: `internal/repository/repository.go` defines `type DB = dbconn.DB` and `var NewDB = dbconn.NewDB`
**AC**: `dbconn.DB` is the single source of truth
**AC**: All 74 importing files compile without changes
**AC**: `make test` passes with zero changes to callers

---

**TC-F002-1: DB type alias satisfies existing call sites**

- Setup: `repository.go` modified to `type DB = dbconn.DB`
- Input: `make build` executed with no changes to any file outside `internal/repository/`
- Expected: Zero compilation errors; binary produced
- Test file: Verified by `make test` (all existing tests must pass)
- Edge cases:
  - `var db *repository.DB` used in external files — must remain valid (type alias is assignment-compatible)
  - `repository.NewDB(sqlDB)` call sites — `var NewDB = dbconn.NewDB` ensures identical signature

**TC-F002-2: dbconn.DB is not duplicated in root**

- Setup: After type alias activation
- Input: Inspect `internal/repository/repository.go` for struct definition of `DB`
- Expected: No `type DB struct` in root package; only `type DB = dbconn.DB`
- Verification method: `grep -n "type DB struct" internal/repository/repository.go` returns empty

**TC-F002-3: make test passes with no external changes**

- Setup: Complete REQ-F-002 changes in isolation (only `repository.go` modified)
- Input: Run `make fmt && make lint && make test` from project root
- Expected: All tests pass; lint clean; format unchanged
- Edge cases:
  - Test DB initialization uses `repository.NewDB(...)` — alias must preserve the constructor signature exactly

---

### REQ-F-003: Extract standalone repositories to sub-packages

**AC**: Each listed repository in its own sub-package (idea, bug, changecard, worksession, document, entitydoc, entityhistory, search, templateenrich)
**AC**: Type aliases in `aliases.go` re-export all types
**AC**: Constructor aliases re-export from root package
**AC**: All 74 importing files compile without modification
**AC**: All repository tests pass

---

**TC-F003-1: Each standalone sub-package compiles independently**

- Setup: After moving each repository file to its sub-package
- Input: `go build ./internal/repository/idea/...`, `go build ./internal/repository/bug/...` etc. for each sub-package
- Expected: Each sub-package builds without errors; no import of `internal/repository` root (no circular import)
- Test file: Per-sub-package `repository_test.go` files (moved alongside production code)
- Edge cases:
  - `bug` sub-package must include `aggregate.go` — verify `BugAggregate` type is accessible from `bug` package
  - `changecard` sub-package must include `aggregate.go` similarly

**TC-F003-2: Type aliases in aliases.go preserve all external-facing types**

- Setup: `aliases.go` created with full alias list from spec section 2.5
- Input: In a file outside `internal/repository/`, use `repository.IdeaRepository`, `repository.BugRepository`, `repository.ChangeCardRepository`, `repository.WorkSessionRepository`, `repository.DocumentRepository`, `repository.EntityDocumentRepository`, `repository.EntityHistoryRepository`, `repository.SearchRepository`, `repository.TemplateEnrichmentRepository`
- Expected: All type references compile without import path changes
- Verification: `make build` with no changes to files outside `internal/repository/`
- Edge cases:
  - If any type is defined only in `aggregate.go` (e.g., aggregate structs), verify those are also aliased
  - Check `BugAggregate` and `ChangeCardAggregate` types are accessible if referenced externally

**TC-F003-3: Constructor aliases produce identical behavior**

- Setup: `aliases.go` defines `var NewIdeaRepository = idea.NewIdeaRepository` etc.
- Input: Call `repository.NewIdeaRepository(db)` vs `idea.NewIdeaRepository(db)` on the same `*DB`
- Expected: Both return the same concrete type; methods on the result are identical
- Test file: Root-level package test asserting the alias is correct type (can use `reflect.TypeOf`)
- Edge cases:
  - Constructor that takes additional parameters (if any standalone repo added params) — alias signature must match exactly

**TC-F003-4: Moved tests pass in new sub-package context**

- Setup: Each `_test.go` file moved to the sub-package directory; package declaration changed from `package repository` to `package idea` (etc.)
- Input: `go test ./internal/repository/idea/...` (and each sub-package)
- Expected: All tests that previously passed in root continue to pass
- Test infrastructure: Each sub-package test uses `test.GetTestDB()` from `internal/test/testdb.go` — verify the relative path resolution works from sub-package location
- Edge cases:
  - `testdb.go` resolves the DB path using `os.Stat("../../internal/repository")` — validate this relative path resolves correctly from each sub-package location

---

### REQ-F-004: Resolve TaskRepository -> EntityNoteRepository coupling

**AC**: `TaskRepository` accepts optional `NoteCreator` interface via constructor injection
**AC**: `NoteCreator` interface defined in `task` sub-package (or `repoutil`) to avoid import cycles
**AC**: When `NoteCreator` is nil, rejection note creation is silently skipped
**AC**: Existing behavior preserved when `NoteCreator` is provided
**AC**: No direct import from `task` -> `note` or `note` -> `task`

---

**TC-F004-1: NoteCreator interface defined correctly**

- Setup: `NoteCreator` interface defined in `internal/repository/task/` or `repoutil`
- Input: Inspect interface definition for method signature
- Expected: Interface has exactly one method matching `EntityNoteRepository.CreateRejectionNote` signature: `CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64, previousStatusOrder int, fromStatus, toStatus, reason, rejectedBy string, metadata *string) error`
- Verification method: Compile-time check that `note.EntityNoteRepository` satisfies `task.NoteCreator` (use `var _ task.NoteCreator = (*note.EntityNoteRepository)(nil)`)

**TC-F004-2: NoteCreator nil — graceful degradation during forced status update**

- Setup: Create `TaskRepository` with `task.NewTaskRepository(db, nil)` (nil NoteCreator)
- Input: Call `repo.updateStatusForcedInternal(ctx, task, "blocked", "reason", "agent")` or equivalent public method that triggers rejection note creation
- Expected: No panic; status update succeeds; no rejection note created; `err == nil`
- Test file: `internal/repository/task/repository_test.go` — new test `TestTaskRepository_ForcedStatus_NilNoteCreator`
- Edge cases:
  - Call the method multiple times with nil NoteCreator — consistent behavior
  - Force-status with empty reason string and nil NoteCreator — still no panic

**TC-F004-3: NoteCreator provided — rejection note created**

- Setup: Create a mock `NoteCreator` that records calls; pass to `task.NewTaskRepository(db, mockNoteCreator)`
- Input: Trigger `updateStatusForcedInternal` with a reason string
- Expected: Mock's `CreateRejectionNote` called exactly once with correct parameters (entityID, fromStatus, toStatus, reason)
- Test file: `internal/repository/task/repository_test.go` — new test `TestTaskRepository_ForcedStatus_WithNoteCreator`
- Edge cases:
  - NoteCreator returns an error — task repository should log/ignore the error (spec says "silently skipped" on failure)
  - Empty reason string with non-nil NoteCreator — verify whether CreateRejectionNote is still called

**TC-F004-4: No import cycle between task and note packages**

- Setup: After splitting task and note to sub-packages
- Input: `go build ./internal/repository/task/...` and `go build ./internal/repository/note/...`
- Expected: Both build independently; `go list -json ./internal/repository/task/ | jq '.Imports'` does not contain `internal/repository/note`; and vice versa
- Verification method: `go vet ./...` catches import cycles

---

### REQ-F-005: Extract EntityNoteRepository into `note/` sub-package

**AC**: `internal/repository/note/repository.go` contains `EntityNoteRepository`
**AC**: Type alias in root `repository` package preserves backward compatibility
**AC**: `TaskRepository` uses `NoteCreator` interface, not direct `note` import
**AC**: All tests pass

---

**TC-F005-1: EntityNoteRepository accessible via both paths**

- Setup: After moving to `internal/repository/note/`; `aliases.go` defines `type EntityNoteRepository = note.EntityNoteRepository`
- Input: Both `repository.NewEntityNoteRepository(db)` (via alias) and `note.NewEntityNoteRepository(db)` return same type
- Expected: Both identifiers compile; the type is identical (alias means `repository.EntityNoteRepository == note.EntityNoteRepository`)
- Edge cases:
  - Code that stores `*repository.EntityNoteRepository` pointer — must still work after alias change

**TC-F005-2: Moved note tests pass**

- Setup: `entity_note_repository_test.go` moved to `internal/repository/note/repository_test.go`; package changed to `package note`
- Input: `go test ./internal/repository/note/...`
- Expected: All tests that previously passed in root continue to pass
- Edge cases:
  - Tests that access package-private helpers — may need minor refactoring if helpers were root-package-private

**TC-F005-3: NoteCreator interface satisfied by EntityNoteRepository**

- Setup: Compile-time check added to `note` or `task` package
- Input: `var _ task.NoteCreator = (*note.EntityNoteRepository)(nil)` in a `_test.go` or `interface_check.go`
- Expected: Compiles without error
- Edge cases: None — this is a purely compile-time check

---

### REQ-F-006: Extract core entity repositories into sub-packages

**AC**: EpicRepository + relationship → `epic/`; FeatureRepository + relationship → `feature/`; TaskRepository + dependency + relationship + history → `task/`
**AC**: DTO types accessible from sub-packages
**AC**: Type aliases in root re-export all types and constructors
**AC**: All 74 importing files compile without changes
**AC**: All repository and service tests pass

---

**TC-F006-1: EpicRepository sub-package compiles and passes tests**

- Setup: `epic_repository.go` and `epic_relationship_repository.go` moved to `internal/repository/epic/`
- Input: `go test ./internal/repository/epic/...`
- Expected: All epic repository tests pass
- Test file: `internal/repository/epic/repository_test.go` (moved from `epic_repository_test.go`) and `internal/repository/epic/relationship_test.go`
- Edge cases:
  - `EpicDisplayDataRaw` DTO type is in `epic` package and aliased in root as `type EpicDisplayDataRaw = epic.EpicDisplayDataRaw`
  - `epic_feature_integration_test.go` in root — must be updated or moved

**TC-F006-2: FeatureRepository sub-package compiles and passes tests**

- Setup: `feature_repository.go` and `feature_relationship_repository.go` moved to `internal/repository/feature/`
- Input: `go test ./internal/repository/feature/...`
- Expected: All feature repository tests pass
- Test file: `internal/repository/feature/repository_test.go` and `internal/repository/feature/relationship_test.go`
- Edge cases:
  - `FeatureDisplayDataRaw` and `FeatureProgressData` DTOs in `feature` package, aliased in root
  - `feature_query_test.go` and `feature_filepath_test.go` in root — must move to `feature/` or remain in root as integration tests

**TC-F006-3: TaskRepository sub-package compiles and passes tests**

- Setup: `task_repository.go`, `task_dependency.go`, `task_relationship_repository.go`, `task_history_repository.go` moved to `internal/repository/task/`
- Input: `go test ./internal/repository/task/...`
- Expected: All task repository tests pass (this is the largest test surface — many test files)
- Test files to move: `task_repository_test.go`, `task_dependency_test.go`, `task_relationship_repository_test.go`, `task_history_repository_test.go`, `task_auto_unblock_test.go`, `task_completion_metadata_test.go`, `task_dependency_auto_block_test.go`, `task_dependency_validation_test.go`, `task_dual_key_test.go`, `task_lifecycle_test.go`, `task_metadata_filter_test.go`, `task_note_rejection_test.go`, `task_query_test.go`, `task_rejection_counts_test.go`, `task_update_orchestrator_action_test.go`, `task_workflow_enum_bypass_test.go`, `task_workflow_force_test.go`, `task_workflow_integration_test.go`, `task_workflow_validation_test.go`
- Edge cases:
  - `TaskDisplayDataRaw` and `HistoryFilters` DTOs in `task` package, aliased in root
  - `task_workflow_integration_e2e_test.go` — verify it can run from sub-package with DB access
  - `task_note_rejection_test.go` — must be updated to use `NoteCreator` injection (TC-F004)

**TC-F006-4: Service interface checks still pass after type alias**

- Setup: After all core repositories moved; `aliases.go` complete
- Input: `go build ./internal/services/...`
- Expected: `internal/services/interface_checks.go` (compile-time interface satisfaction checks) compiles cleanly
- Edge cases:
  - Service interfaces that reference `repository.TaskDisplayDataRaw` as a return type — these must resolve through the alias

**TC-F006-5: HistoryFilters DTO accessible from service layer**

- Setup: `HistoryFilters` defined in `task` sub-package; `type HistoryFilters = task.HistoryFilters` in `aliases.go`
- Input: Code in `internal/services/task_service.go` uses `repository.HistoryFilters{...}` struct literal
- Expected: Compiles; struct fields are accessible via alias
- Edge cases:
  - Struct literal `repository.HistoryFilters{Field: value}` — must work through alias (type alias preserves struct field access)

---

### REQ-F-007: Extract EntityRelationshipRepository into `entityrel/`

**AC**: `internal/repository/entityrel/repository.go` contains `EntityRelationshipRepository`
**AC**: Type alias and constructor alias in root package
**AC**: All tests pass

---

**TC-F007-1: EntityRelationshipRepository in entityrel sub-package**

- Setup: `entity_relationship_repository.go` moved to `internal/repository/entityrel/repository.go`
- Input: `go test ./internal/repository/entityrel/...`
- Expected: All relationship repository tests pass
- Test file: `internal/repository/entityrel/repository_test.go` (moved from `entity_relationship_repository_test.go`)
- Edge cases:
  - Verify no other repository type imports `EntityRelationshipRepository` as a concrete type (would create dependency that needs aliasing)

**TC-F007-2: Root alias for EntityRelationshipRepository**

- Setup: `aliases.go` has `type EntityRelationshipRepository = entityrel.EntityRelationshipRepository` and `var NewEntityRelationshipRepository = entityrel.NewEntityRelationshipRepository`
- Input: Existing callers use `repository.NewEntityRelationshipRepository(db)` and `repository.EntityRelationshipRepository`
- Expected: No caller changes required; `make build` clean

---

### REQ-F-008: Clean up root repository package

**AC**: Root package contains no more than 5 production Go files
**AC**: No production code duplicated between root and sub-packages
**AC**: All callers continue to work via type aliases

---

**TC-F008-1: Root package file count**

- Setup: After all phases complete
- Input: `ls internal/repository/*.go | grep -v _test.go | wc -l`
- Expected: Count is 5 or fewer
- Acceptable root files: `aliases.go`, `repository.go`, `polymorphic_doc_adapter.go` (plus at most 2 others)
- Edge cases:
  - `task_repository.go.backup` — must be removed (not a production Go file by name, but should be cleaned up)

**TC-F008-2: No duplicate type definitions**

- Setup: After all phases complete
- Input: For each type exported in `aliases.go`, verify the type is defined exactly once (in the sub-package) and aliased in root
- Expected: `grep -rn "type TaskRepository struct\|type EpicRepository struct" internal/repository/*.go` returns empty (only sub-packages define structs)
- Edge cases:
  - Verify `polymorphic_doc_adapter.go` does not re-define any type — it is an adapter, not a type definition

---

### REQ-F-009: Remove empty placeholder directories

**AC**: No empty directories remain under `internal/repository/`
**AC**: Each directory that exists contains at least one `.go` file

---

**TC-F009-1: No empty directories under internal/repository/**

- Setup: After all phases complete
- Input: `find internal/repository -type d -empty`
- Expected: Empty output (no empty directories)
- Test method: Shell assertion in CI/verification script
- Edge cases:
  - `__pycache__` or `.git` subdirectories if present — filter to only check Go-relevant directories

**TC-F009-2: Each sub-package directory contains at least one .go file**

- Setup: After all phases complete
- Input: For each of `epic/`, `feature/`, `task/`, `note/`, `worksession/`, `idea/`, `bug/`, `changecard/`, `document/`, `entitydoc/`, `entityhistory/`, `entityrel/`, `search/`, `templateenrich/`, `repoutil/`, `dbconn/`
- Expected: `ls internal/repository/<dir>/*.go` returns at least one file
- Edge cases: Verify test files (ending in `_test.go`) alone do not count as "non-empty" — each dir needs at least one production `.go` file

---

### REQ-NF-001: Zero breaking changes for external callers

**AC**: All 74 files that import `internal/repository` compile without import path or type name changes
**AC**: `make test` passes with zero changes to files outside `internal/repository/`

---

**TC-NF001-1: Full project build after each phase**

- Setup: After each implementation phase (Phases 1-4 per spec section 2.7)
- Input: `make fmt && make lint && make test` from project root — with no modifications to any file outside `internal/repository/`
- Expected: All pass; zero compilation errors; all tests pass
- Test scope: This is the highest-priority integration check — run after EVERY phase
- Edge cases:
  - `internal/services/interface_checks.go` — explicitly check compile-time interface assertions
  - `internal/cli/services_global.go` — all constructor calls must compile
  - `cmd/server/services.go` and `cmd/demo/main.go` — both wiring files must compile

**TC-NF001-2: No external file modifications in git diff**

- Setup: After complete implementation
- Input: `git diff --name-only` filtered to files outside `internal/repository/`
- Expected: Only files within `internal/repository/` and `internal/repository/dbconn/` appear in diff (plus new sub-package files)
- Edge cases:
  - `internal/services/interface_checks.go` may need a minor update if it directly references concrete types from sub-packages — but per spec, this must be zero changes

---

### REQ-NF-002: No performance regression

**AC**: Type aliases have zero runtime cost
**AC**: `go test -bench` shows no regression

---

**TC-NF002-1: Benchmark comparison for repository operations**

- Setup: Run existing benchmarks before and after migration (if any exist); otherwise create a minimal benchmark
- Input: `go test -bench=. -benchmem ./internal/repository/task/...` (post-migration)
- Expected: No measurable overhead introduced by alias layer; type alias is compile-time only
- Verification method: Go spec guarantees `type T = U` is identical to `U` at runtime — no indirection. Document this guarantee in test notes rather than running a micro-benchmark.
- Edge cases:
  - `NewTaskRepository` wrapper function in `aliases.go` adds one function call per construction — this is one-time cost, not per-operation cost; acceptable

---

### REQ-NF-003: Test isolation preserved

**AC**: Repository tests continue using shared test DB at `internal/repository/test-shark-tasks.db`
**AC**: Sub-package tests use `internal/test/testdb.go` for DB setup
**AC**: No new test database files created

---

**TC-NF003-1: Sub-package tests use correct test DB path**

- Setup: Sub-package test files reference `test.GetTestDB()` from `internal/test/testdb.go`
- Input: `go test ./internal/repository/task/...` (run from project root)
- Expected: Tests find and use `internal/repository/test-shark-tasks.db`; no new `.db` file created in `internal/repository/task/`
- Verification: `find internal/repository -name "*.db" | grep -v "test-shark-tasks.db"` returns empty after test run
- Edge cases:
  - `testdb.go` uses path heuristics (`os.Stat("internal/repository")`, `os.Stat("../../internal/repository")`). When tests run from `internal/repository/task/`, the `../../internal/repository` path resolves correctly — verify this with a minimal sub-package test before moving all files

**TC-NF003-2: Test database not duplicated**

- Setup: After migration, run full test suite
- Input: `find internal/repository -name "*.db" | wc -l`
- Expected: Exactly 1 (the shared `test-shark-tasks.db`); `-shm` and `-wal` files may also appear, which is acceptable

---

### REQ-NF-004: OpenTelemetry tracing preserved

**AC**: Each sub-package has its own `trace.Tracer` with descriptive instrumentation name
**AC**: Existing tracing tests pass
**AC**: New sub-packages produce correctly-named spans

---

**TC-NF004-1: Per-sub-package tracer name in spans**

- Setup: Each sub-package initializes `var tracer = repoutil.NewTracer("internal/repository/<subpkg>")` following the pattern in spec section 2.6
- Input: For `task`, `epic`, `feature`, `note` sub-packages: configure in-memory OTel TracerProvider; execute a repository operation (e.g., `GetByKey`); inspect spans
- Expected: Span instrumentation scope name matches `"internal/repository/task"` (etc.) — NOT `"internal/repository"` (the old monolithic name)
- Test file: New test `TestTracerName_Task` in `internal/repository/task/repository_test.go`; similar tests for `epic`, `feature`, `note`
- Test pattern: Mirrors `TestTaskRepository_GetByKey_SpanCreated` in existing `tracing_test.go` — use `tracetest.NewInMemoryExporter()` and `sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))`; then call `repoutil.NewTracer("internal/repository/task")` after setting the provider
- Edge cases:
  - When OTel is not configured (no provider set), `repoutil.NewTracer(name)` returns the global no-op tracer — operations must not panic in this case (verify with a test that calls NewTracer with no provider set)

**TC-NF004-2: Existing tracing tests pass after migration**

- Setup: `tracing_test.go` moved to root package (or split into sub-package tests)
- Input: `go test ./internal/repository/... -run TestTaskRepository.*Span`
- Expected: All existing tracing tests pass; span names are unchanged for tests that don't test sub-package tracers specifically
- Edge cases:
  - `setupTracingTest` helper re-initializes `repoTracer` in root package — after migration this pattern must be replicated per sub-package (`taskTracer`, `epicTracer`, etc.)

---

## 2. Integration Scenarios

### IS-001: Service layer compiles and tests pass end-to-end

**Components**: `internal/services/` → `internal/repository/` (via aliases)
**What to verify**: Service files that reference repository DTOs (`repository.TaskDisplayDataRaw`, `repository.HistoryFilters`, `repository.EpicDisplayDataRaw`, `repository.FeatureProgressData`) still compile and function correctly after types are aliased from sub-packages.
**Verification**: `go test ./internal/services/...` passes with no changes to service files.
**Epic UAT contribution**: E07 enhancements are non-breaking infrastructure improvements — this scenario ensures service behavior is preserved across all other E07 features (E07-F26, E07-F31) that depend on repository types.

### IS-002: CLI wiring compiles and shark binary runs

**Components**: `internal/cli/services_global.go`, `internal/cli/service_accessors.go` → `internal/repository/`
**What to verify**: All `repository.New*Repository(db)` constructor calls in CLI wiring files compile; `./bin/shark task list` and similar commands execute without runtime panic.
**Verification**: `make build` then `./bin/shark list` returns expected output.
**Epic UAT contribution**: CLI functionality across all E07 features remains intact.

### IS-003: Server wiring compiles

**Components**: `cmd/server/services.go`, `cmd/demo/main.go` → `internal/repository/`
**What to verify**: Server entry points compile and the `WireServices()` function can construct all repositories via aliases.
**Verification**: `make build` succeeds for all binaries.

### IS-004: E23-F01 observability traces carry through sub-packages

**Components**: `repoutil.NewTracer` → each sub-package's `var tracer` → OTel spans
**What to verify**: Spans produced by task, epic, and feature repositories appear under their respective sub-package instrumentation names, improving observability attribution per E23-F01 goals.
**Verification**: TC-NF004-1 and TC-NF004-2 above. The span name `"internal/repository/task"` (versus the old `"internal/repository"`) is the key observable difference.

### IS-005: NoteCreator wiring in root aliases.go preserves rejection note behavior

**Components**: `internal/repository/aliases.go` → `task.NewTaskRepository(db, noteRepo)` wiring → `note.EntityNoteRepository`
**What to verify**: Callers of `repository.NewTaskRepository(db)` (the alias constructor) receive a `TaskRepository` with rejection note support wired in. Existing `task_note_rejection_test.go` behaviors are preserved.
**Verification**: Move `task_note_rejection_test.go` to `task/` sub-package; tests pass with `NoteCreator` wired. Additionally, a root-package integration test verifies that `repository.NewTaskRepository(db)` (the alias wrapper) creates rejection notes when status is force-updated.

### IS-006: Parallel feature development unblocked

**Components**: E07-F26 (Workflow Service Authority), E07-F31 (Unified Entity Display Rendering) → `internal/repository/`
**What to verify**: The type alias shim means zero import path changes for parallel features. After E07-F36 merge, E07-F26 and E07-F31 branches must apply cleanly.
**Verification**: After implementation, confirm that `git merge` or rebase of a simulated parallel branch (touching only service/CLI layers) produces no merge conflicts in files outside `internal/repository/`.

---

## 3. Test Infrastructure

### Existing test patterns to follow

**File**: `internal/repository/tracing_test.go`
- Pattern: `setupTracingTest(t)` configures in-memory `TracerProvider`, re-initializes package-level `repoTracer`, registers cleanup. Each sub-package test must replicate this pattern for its own package-level tracer variable.
- Example usage for sub-packages: Each sub-package defines a parallel `setupSubpkgTracingTest(t)` that re-initializes the sub-package's local `tracer` variable.

**File**: `internal/test/testdb.go` + `internal/repository/test-shark-tasks.db`
- Pattern: `test.GetTestDB()` returns a shared `*sql.DB`; `test.SeedTestData()` inserts canonical E99 epic/feature for tests to use. All moved test files continue using this pattern — no new DB files.
- Note: When running from sub-package directories, the path heuristic in `testdb.go` uses `os.Stat("../../internal/repository")` — this resolves correctly from `internal/repository/task/` as `../../internal/repository`. Verify this explicitly before bulk-moving test files.

**File**: `internal/repository/key_lookup_test.go`
- Pattern: Table-driven tests with `[]struct{input, expected}` — follow this for `repoutil` tests.

**File**: `internal/services/interface_checks.go`
- Pattern: Compile-time `var _ SomeInterface = (*ConcreteType)(nil)` assertions. Add `var _ task.NoteCreator = (*note.EntityNoteRepository)(nil)` to validate the interface satisfaction.

### New test helpers needed

**TC-F004-2 and TC-F004-3 require a mock `NoteCreator`:**

```go
// internal/repository/task/repository_test.go (or a test helper file)
type mockNoteCreator struct {
    calls []noteCreatorCall
}

type noteCreatorCall struct {
    entityType          models.EntityType
    entityID            int64
    fromStatus, toStatus string
    reason              string
}

func (m *mockNoteCreator) CreateRejectionNote(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    previousStatusOrder int,
    fromStatus, toStatus, reason, rejectedBy string,
    metadata *string,
) error {
    m.calls = append(m.calls, noteCreatorCall{
        entityType: entityType,
        entityID:   entityID,
        fromStatus: fromStatus,
        toStatus:   toStatus,
        reason:     reason,
    })
    return nil
}
```

This mock is self-contained in the `task` test package and does not import `note` — preserving the no-cycle requirement.

**TC-NF003-1 requires a path verification test before bulk migration:**

Create a minimal test `internal/repository/task/db_path_test.go` early in Phase 4 to verify `test.GetTestDB()` resolves the correct path. This catches the path heuristic issue before all test files are moved.

```go
package task_test

import (
    "testing"
    "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestGetTestDB_PathResolves(t *testing.T) {
    db := test.GetTestDB()
    if db == nil {
        t.Fatal("GetTestDB returned nil — path resolution failed from sub-package location")
    }
    if err := db.Ping(); err != nil {
        t.Fatalf("test DB ping failed: %v", err)
    }
}
```

---

## 4. Test Execution Order

Tests should be run in phase order, matching spec section 2.7:

1. **After Phase 1** (repoutil + dbconn alias): Run `go test ./internal/repository/repoutil/...` and `make test`
2. **After Phase 2** (standalone repos): Run `go test ./internal/repository/idea/ ./internal/repository/bug/ ...` and `make test`
3. **After Phase 3** (NoteCreator + note extraction): Run `go test ./internal/repository/task/ ./internal/repository/note/...` and `make test`
4. **After Phase 4** (core entity repos + cleanup): Run `go test ./internal/repository/...` and full `make fmt && make lint && make test`

Each phase gate: `make fmt && make lint && make test` must pass completely before proceeding.

---

## 5. Exit Gate Checklist

- [x] Every AC in spec.md has at least one test case (REQ-F-001 through REQ-NF-004)
- [x] Edge cases identified for each AC
- [x] Integration scenarios cover cross-component boundaries (service layer, CLI, server, OTel, parallel features)
- [x] Test patterns reference existing infrastructure (`tracing_test.go`, `testdb.go`, `interface_checks.go`)
- [x] New test helpers identified where needed (mock NoteCreator, path verification test)
- [x] All tests trace to feature acceptance criteria, which trace to E07 epic requirements (code organization, zero breaking changes)

---

*Test plan authored: 2026-03-22*
