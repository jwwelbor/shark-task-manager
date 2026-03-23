# E07-F36 Specification: Split Large Repository Package into Entity-Specific Sub-Packages

**Feature**: E07-F36
**Date**: 2026-03-22
**Status**: In Specification
**Complexity**: COMPLEX (18/27 per assessment)

---

## 1. Requirements

### 1.1 Functional Requirements

**REQ-F-001: Extract shared utilities to a common sub-package**
- **Description**: Move `key_lookup.go`, `order_resequence.go`, and `tracing.go` from `internal/repository/` into `internal/repository/repoutil/` so that entity-specific sub-packages can import them without circular dependencies.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `internal/repository/repoutil/` contains `key_lookup.go`, `order_resequence.go`, `tracing.go`
  - [ ] Exported functions: `ContainsHyphen`, `IsNumeric`, `SplitAtFirstHyphen`, `SplitAtNthHyphen`, `ResequenceOrders`, `NewTracer`, `RecordSpanError`
  - [ ] All existing tests for these utilities pass from their new location
  - [ ] No code duplication between root package and `repoutil`

**REQ-F-002: Activate dbconn package as the canonical DB type**
- **Description**: Make `repository.DB` a type alias for `dbconn.DB`. The `dbconn` package already exists at `internal/repository/dbconn/db.go` but is currently unused. All external callers continue using `repository.DB` (backward-compatible via alias).
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `internal/repository/repository.go` defines `type DB = dbconn.DB` and `var NewDB = dbconn.NewDB`
  - [ ] `dbconn.DB` is the single source of truth for the DB wrapper type
  - [ ] All 74 files importing `repository.DB` continue to compile without changes
  - [ ] `make test` passes with zero changes to callers

**REQ-F-003: Extract standalone entity repositories into sub-packages**
- **Description**: Move the following repositories (which have no intra-package type dependencies) into dedicated sub-packages:
  - `IdeaRepository` -> `internal/repository/idea/`
  - `BugRepository` + `bug_aggregate.go` -> `internal/repository/bug/`
  - `ChangeCardRepository` + `change_card_aggregate.go` -> `internal/repository/changecard/`
  - `WorkSessionRepository` -> `internal/repository/worksession/`
  - `DocumentRepository` -> `internal/repository/document/`
  - `EntityDocumentRepository` -> `internal/repository/entitydoc/`
  - `EntityHistoryRepository` -> `internal/repository/entityhistory/`
  - `SearchRepository` -> `internal/repository/search/`
  - `TemplateEnrichmentRepository` -> `internal/repository/templateenrich/`
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Each listed repository resides in its own sub-package
  - [ ] Type aliases in `internal/repository/aliases.go` re-export all types for backward compatibility
  - [ ] Constructor aliases (`NewIdeaRepository`, `NewBugRepository`, etc.) re-export from root package
  - [ ] All 74 importing files compile without modification
  - [ ] All repository tests pass

**REQ-F-004: Resolve TaskRepository -> EntityNoteRepository coupling**
- **Description**: `TaskRepository.updateStatusForcedInternal()` and `StatusUpdateRaw()` directly instantiate `NewEntityNoteRepository(r.db)` at lines ~1016 and ~1110 of `task_repository.go`. This must be resolved before `TaskRepository` and `EntityNoteRepository` can live in separate packages.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `TaskRepository` accepts an optional `NoteCreator` interface via constructor injection
  - [ ] The `NoteCreator` interface is defined in the `repoutil` package (or `task` sub-package) to avoid import cycles
  - [ ] When `NoteCreator` is nil, rejection note creation is silently skipped (graceful degradation)
  - [ ] Existing behavior (rejection notes created on forced status updates) is preserved when `NoteCreator` is provided
  - [ ] No direct import from `task` -> `note` or `note` -> `task` packages

**REQ-F-005: Extract EntityNoteRepository into sub-package**
- **Description**: Move `EntityNoteRepository` to `internal/repository/note/` after REQ-F-004 resolves the coupling.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `internal/repository/note/repository.go` contains `EntityNoteRepository`
  - [ ] Type alias in root `repository` package preserves backward compatibility
  - [ ] `TaskRepository` uses the `NoteCreator` interface, not a direct import of `note` package
  - [ ] All tests pass

**REQ-F-006: Extract core entity repositories into sub-packages**
- **Description**: Move the three core entity repositories:
  - `EpicRepository` + `epic_relationship_repository.go` -> `internal/repository/epic/`
  - `FeatureRepository` + `feature_relationship_repository.go` -> `internal/repository/feature/`
  - `TaskRepository` + `task_dependency.go` + `task_relationship_repository.go` + `task_history_repository.go` -> `internal/repository/task/`
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Each core entity repository and its related files reside in its own sub-package
  - [ ] DTO types (`TaskDisplayDataRaw`, `EpicDisplayDataRaw`, `FeatureDisplayDataRaw`, `FeatureProgressData`, `HistoryFilters`) are accessible from their sub-packages
  - [ ] Type aliases in root package re-export all types and constructors
  - [ ] All 74 importing files compile without changes
  - [ ] All repository and service tests pass

**REQ-F-007: Extract EntityRelationshipRepository into sub-package**
- **Description**: Move `EntityRelationshipRepository` to `internal/repository/entityrel/`.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `internal/repository/entityrel/repository.go` contains `EntityRelationshipRepository`
  - [ ] Type alias and constructor alias in root package
  - [ ] All tests pass

**REQ-F-008: Clean up root repository package**
- **Description**: After all extractions, the root `internal/repository/` package should contain only: `aliases.go` (type aliases and constructor re-exports), `repository.go` (DB alias), and `polymorphic_doc_adapter.go` (adapter that depends on multiple sub-packages).
- **Priority**: Should-Have
- **Acceptance Criteria**:
  - [ ] Root package contains no more than 5 production Go files
  - [ ] No production code duplicated between root and sub-packages
  - [ ] All callers continue to work via type aliases

**REQ-F-009: Remove empty placeholder directories**
- **Description**: Delete or repurpose the 5 empty placeholder directories (`epic/`, `feature/`, `task/`, `note/`, `worksession/`) created during earlier prep work, replacing them with the actual sub-package files.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] No empty directories remain under `internal/repository/`
  - [ ] Each directory that exists contains at least one `.go` file

### 1.2 Non-Functional Requirements

**REQ-NF-001: Zero breaking changes for external callers**
- **Description**: All 74 files that currently import `internal/repository` must compile without any import path or type name changes.
- **Measurement**: `make test` passes with zero changes to files outside `internal/repository/`.
- **Target**: 100% backward compatibility.
- **Justification**: The codebase has active development across multiple features (E07-F26, E07-F31, E23-F01). Breaking imports would cause merge conflicts and block parallel work.

**REQ-NF-002: No performance regression**
- **Description**: The re-export shim layer (type aliases) must have zero runtime cost.
- **Measurement**: Go type aliases (`type X = Y`) are compile-time only; verify via `go test -bench`.
- **Target**: Zero measurable overhead (type aliases are resolved at compile time).

**REQ-NF-003: Test isolation preserved**
- **Description**: Repository tests that use real databases must continue to work with the shared test DB at `internal/repository/test-shark-tasks.db`. Sub-package tests must use `internal/test/testdb.go` for DB setup.
- **Measurement**: `make test` passes; no new test database files created.

**REQ-NF-004: OpenTelemetry tracing preserved**
- **Description**: Each sub-package must have its own `trace.Tracer` with a descriptive instrumentation name (e.g., `internal/repository/task`). The per-sub-package tracer pattern improves observability by attributing spans to specific repositories.
- **Measurement**: Existing tracing tests pass; new sub-packages produce correctly-named spans.

### 1.3 Out of Scope

1. **Updating callers to use sub-package import paths directly**
   - Why: The type alias shim provides full backward compatibility. Direct imports can be done incrementally in future work.
   - Future: Individual features can optionally switch to direct imports when convenient.

2. **Moving DTO types to a shared `dto` or `models` package**
   - Why: Types like `TaskDisplayDataRaw` are repository-layer concerns. Moving them to `models` would blur the layer boundary.
   - Workaround: DTOs stay with their entity sub-package and are re-exported via aliases.

3. **Splitting test files into sub-packages**
   - Why: The 42 test files use `package repository` (same-package access). Moving production code while keeping test files in root would break tests. Tests must move with their production code.
   - This IS in scope -- test files move alongside production files.

4. **Refactoring the `polymorphic_doc_adapter.go`**
   - Why: This adapter intentionally depends on `EntityDocumentRepository`. It stays in the root package as a cross-cutting adapter. Refactoring it is a separate concern.

---

## 2. Architecture

### 2.1 Target Directory Structure

```
internal/repository/
├── dbconn/                          # EXISTING (activate)
│   └── db.go                        # DB wrapper type (canonical)
├── repoutil/                        # NEW
│   ├── key_lookup.go                # Shared key parsing utilities
│   ├── key_lookup_test.go
│   ├── order_resequence.go          # Shared order resequencing
│   ├── order_resequence_test.go
│   └── tracing.go                   # NewTracer() + RecordSpanError()
├── epic/                            # REPURPOSE (was empty placeholder)
│   ├── repository.go                # EpicRepository + EpicDisplayDataRaw
│   ├── repository_test.go
│   ├── relationship.go              # from epic_relationship_repository.go
│   └── relationship_test.go
├── feature/                         # REPURPOSE (was empty placeholder)
│   ├── repository.go                # FeatureRepository + FeatureDisplayDataRaw + FeatureProgressData
│   ├── repository_test.go
│   ├── relationship.go              # from feature_relationship_repository.go
│   └── relationship_test.go
├── task/                            # REPURPOSE (was empty placeholder)
│   ├── repository.go                # TaskRepository + TaskDisplayDataRaw + HistoryFilters
│   ├── repository_test.go
│   ├── dependency.go                # from task_dependency.go
│   ├── dependency_test.go
│   ├── relationship.go              # from task_relationship_repository.go
│   ├── relationship_test.go
│   ├── history.go                   # from task_history_repository.go
│   └── history_test.go
├── note/                            # REPURPOSE (was empty placeholder)
│   ├── repository.go                # EntityNoteRepository
│   └── repository_test.go
├── worksession/                     # REPURPOSE (was empty placeholder)
│   ├── repository.go                # WorkSessionRepository
│   └── repository_test.go
├── idea/                            # NEW
│   ├── repository.go                # IdeaRepository
│   └── repository_test.go
├── bug/                             # NEW
│   ├── repository.go                # BugRepository + aggregate
│   └── repository_test.go
├── changecard/                      # NEW
│   ├── repository.go                # ChangeCardRepository + aggregate
│   └── repository_test.go
├── document/                        # NEW
│   ├── repository.go                # DocumentRepository
│   └── repository_test.go
├── entitydoc/                       # NEW
│   ├── repository.go                # EntityDocumentRepository
│   └── repository_test.go
├── entityhistory/                   # NEW
│   ├── repository.go                # EntityHistoryRepository
│   └── repository_test.go
├── entityrel/                       # NEW
│   ├── repository.go                # EntityRelationshipRepository
│   └── repository_test.go
├── search/                          # NEW
│   ├── repository.go                # SearchRepository
│   └── repository_test.go
├── templateenrich/                  # NEW
│   ├── repository.go                # TemplateEnrichmentRepository
│   └── repository_test.go
├── aliases.go                       # NEW: type aliases for backward compat
├── repository.go                    # MODIFIED: DB = dbconn.DB alias
└── polymorphic_doc_adapter.go       # UNCHANGED (stays in root)
```

### 2.2 Data Model Changes

None. This is a pure code organization refactoring. No database schema, migrations, or data changes.

### 2.3 Key Technical Decisions

**Decision 1: Type Alias Re-Export Shim (Approach A from research report)**
- **Choice**: Use Go type aliases (`type X = Y`) in `aliases.go` to re-export all types from sub-packages under the `repository` package name.
- **Rationale**: 74 files import `repository`. Updating them all in one commit is high-risk and blocks parallel development. Type aliases are zero-cost at runtime (resolved at compile time) and provide 100% source compatibility.
- **Alternative Rejected**: Full migration (Approach B) -- too high risk for parallel feature development; would touch 74 files across the codebase.
- **Reference**: Go specification on type aliases: `type T1 = T2` creates an identical type, not a distinct one. No runtime indirection.

**Decision 2: `repoutil` package for shared code (not `shared/`)**
- **Choice**: Name the shared utilities package `repoutil` rather than `shared`.
- **Rationale**: `shared` is a generic name that doesn't convey purpose. `repoutil` (repository utilities) is specific and follows Go naming conventions (lowercase, concise, descriptive). Follows pattern of `internal/fileops`, `internal/taskcreation`.
- **Contains**: Key parsing (`ContainsHyphen`, `IsNumeric`, `SplitAtFirstHyphen`, `SplitAtNthHyphen`), order resequencing (`ResequenceOrders`, `OrderedItem`), tracing helpers (`NewTracer`, `RecordSpanError`).

**Decision 3: Per-sub-package OTel tracers**
- **Choice**: Each sub-package creates its own `trace.Tracer` via `repoutil.NewTracer("internal/repository/<subpkg>")`.
- **Rationale**: Per-package tracers provide better observability than a single shared tracer. Spans are automatically attributed to the correct repository (task, epic, feature), improving debugging. This aligns with OTel best practices from E23-F01.
- **Alternative Rejected**: Shared singleton tracer in `repoutil` -- loses per-package attribution.

**Decision 4: NoteCreator interface for TaskRepository decoupling**
- **Choice**: Define a `NoteCreator` interface in the `task` sub-package that `TaskRepository` optionally accepts via constructor injection.
- **Rationale**: `TaskRepository` currently calls `NewEntityNoteRepository(r.db)` directly in two places. This creates a concrete dependency that would cause an import cycle (`task` -> `note` -> `task`). An interface in the `task` package breaks the cycle while preserving the behavior.
- **Interface definition**:
  ```go
  // In internal/repository/task/repository.go
  type NoteCreator interface {
      CreateRejectionNote(ctx context.Context, entityType models.EntityType,
          entityID int64, previousStatusOrder int, fromStatus, toStatus,
          reason, rejectedBy string, metadata *string) error
  }
  ```
- **Wiring**: The root `aliases.go` constructor wrapper wires `EntityNoteRepository` as the `NoteCreator` when constructing `TaskRepository`. This happens at the root package level which can import both sub-packages.

**Decision 5: Keep `polymorphic_doc_adapter.go` in root package**
- **Choice**: Leave `PolymorphicDocRepoAdapter` in the root `repository` package.
- **Rationale**: This adapter intentionally depends on `EntityDocumentRepository`. After the split, it imports `entitydoc` sub-package. Since the root package already imports all sub-packages for aliasing, this creates no new dependency.

**Decision 6: Test files move with production code**
- **Choice**: Each sub-package's test files move alongside the production code.
- **Rationale**: Test files currently use `package repository` (same-package testing). When code moves to `package task`, tests must use `package task` or `package task_test`. Since tests access unexported helpers, they should stay in the same package. The test database helper (`internal/test/testdb.go`) remains shared.

### 2.4 Integration with Existing Code

**Files Modified (in `internal/repository/`):**

| Current File | Action | Target Location |
|---|---|---|
| `repository.go` | MODIFY | Keep in root; change `DB` to type alias for `dbconn.DB` |
| `key_lookup.go` | MOVE | `repoutil/key_lookup.go` (export functions) |
| `key_lookup_test.go` | MOVE | `repoutil/key_lookup_test.go` |
| `order_resequence.go` | MOVE | `repoutil/order_resequence.go` (export `ResequenceOrders`, `OrderedItem`) |
| `order_resequence_test.go` | MOVE | `repoutil/order_resequence_test.go` |
| `tracing.go` | MOVE | `repoutil/tracing.go` (export `NewTracer`, `RecordSpanError`) |
| `tracing_test.go` | MOVE | `repoutil/tracing_test.go` |
| `epic_repository.go` | MOVE | `epic/repository.go` |
| `epic_repository_test.go` | MOVE | `epic/repository_test.go` |
| `epic_relationship_repository.go` | MOVE | `epic/relationship.go` |
| `epic_relationship_repository_test.go` | MOVE | `epic/relationship_test.go` |
| `feature_repository.go` | MOVE | `feature/repository.go` |
| `feature_repository_test.go` | MOVE | `feature/repository_test.go` |
| `feature_relationship_repository.go` | MOVE | `feature/relationship.go` |
| `feature_relationship_repository_test.go` | MOVE | `feature/relationship_test.go` |
| `task_repository.go` | MOVE + MODIFY | `task/repository.go` (add `NoteCreator` interface + injection) |
| `task_repository_test.go` | MOVE | `task/repository_test.go` |
| `task_dependency.go` | MOVE | `task/dependency.go` |
| `task_dependency_test.go` | MOVE | `task/dependency_test.go` |
| `task_relationship_repository.go` | MOVE | `task/relationship.go` |
| `task_relationship_repository_test.go` | MOVE | `task/relationship_test.go` |
| `task_history_repository.go` | MOVE | `task/history.go` |
| `task_history_repository_test.go` | MOVE | `task/history_test.go` |
| `entity_note_repository.go` | MOVE | `note/repository.go` |
| `entity_note_repository_test.go` | MOVE | `note/repository_test.go` |
| `work_session_repository.go` | MOVE | `worksession/repository.go` |
| `work_session_repository_test.go` | MOVE | `worksession/repository_test.go` |
| `idea_repository.go` | MOVE | `idea/repository.go` |
| `idea_repository_test.go` | MOVE | `idea/repository_test.go` |
| `bug_repository.go` | MOVE | `bug/repository.go` |
| `bug_repository_test.go` | MOVE | `bug/repository_test.go` |
| `bug_aggregate.go` | MOVE | `bug/aggregate.go` |
| `bug_aggregate_test.go` | MOVE | `bug/aggregate_test.go` |
| `change_card_repository.go` | MOVE | `changecard/repository.go` |
| `change_card_repository_test.go` | MOVE | `changecard/repository_test.go` |
| `change_card_aggregate.go` | MOVE | `changecard/aggregate.go` |
| `change_card_aggregate_test.go` | MOVE | `changecard/aggregate_test.go` |
| `document_repository.go` | MOVE | `document/repository.go` |
| `document_repository_test.go` | MOVE | `document/repository_test.go` |
| `entity_document_repository.go` | MOVE | `entitydoc/repository.go` |
| `entity_document_repository_test.go` | MOVE | `entitydoc/repository_test.go` |
| `entity_history_repository.go` | MOVE | `entityhistory/repository.go` |
| `entity_history_repository_test.go` | MOVE | `entityhistory/repository_test.go` |
| `entity_relationship_repository.go` | MOVE | `entityrel/repository.go` |
| `entity_relationship_repository_test.go` | MOVE | `entityrel/repository_test.go` |
| `search_repository.go` | MOVE | `search/repository.go` |
| `search_repository_test.go` | MOVE | `search/repository_test.go` |
| `template_enrichment_repository.go` | MOVE | `templateenrich/repository.go` |
| `template_enrichment_repository_test.go` | MOVE | `templateenrich/repository_test.go` |
| `polymorphic_doc_adapter.go` | MODIFY | Keep in root; update imports to use `entitydoc` sub-package |

**New File Created:**

| File | Purpose |
|---|---|
| `internal/repository/aliases.go` | Type aliases and constructor re-exports for all sub-packages |
| `internal/repository/repoutil/key_lookup.go` | Exported key parsing utilities |
| `internal/repository/repoutil/order_resequence.go` | Exported order resequencing utility |
| `internal/repository/repoutil/tracing.go` | Exported tracing helpers |

**Files Outside `internal/repository/` -- ZERO CHANGES REQUIRED:**
All 74 files that import `internal/repository` continue to work unchanged due to the type alias shim.

### 2.5 Alias File Design

The `internal/repository/aliases.go` file is the key to backward compatibility:

```go
package repository

import (
    "github.com/jwwelbor/shark-task-manager/internal/repository/bug"
    "github.com/jwwelbor/shark-task-manager/internal/repository/changecard"
    "github.com/jwwelbor/shark-task-manager/internal/repository/document"
    "github.com/jwwelbor/shark-task-manager/internal/repository/entitydoc"
    "github.com/jwwelbor/shark-task-manager/internal/repository/entityhistory"
    "github.com/jwwelbor/shark-task-manager/internal/repository/entityrel"
    "github.com/jwwelbor/shark-task-manager/internal/repository/epic"
    "github.com/jwwelbor/shark-task-manager/internal/repository/feature"
    "github.com/jwwelbor/shark-task-manager/internal/repository/idea"
    "github.com/jwwelbor/shark-task-manager/internal/repository/note"
    "github.com/jwwelbor/shark-task-manager/internal/repository/search"
    "github.com/jwwelbor/shark-task-manager/internal/repository/task"
    "github.com/jwwelbor/shark-task-manager/internal/repository/templateenrich"
    "github.com/jwwelbor/shark-task-manager/internal/repository/worksession"
)

// Type aliases — zero runtime cost, full source compatibility.

type TaskRepository = task.TaskRepository
type TaskDisplayDataRaw = task.TaskDisplayDataRaw
type HistoryFilters = task.HistoryFilters
type TaskHistoryRepository = task.HistoryRepository

type EpicRepository = epic.EpicRepository
type EpicDisplayDataRaw = epic.EpicDisplayDataRaw

type FeatureRepository = feature.FeatureRepository
type FeatureDisplayDataRaw = feature.FeatureDisplayDataRaw
type FeatureProgressData = feature.FeatureProgressData

type EntityNoteRepository = note.EntityNoteRepository
type WorkSessionRepository = worksession.WorkSessionRepository
type IdeaRepository = idea.IdeaRepository
type BugRepository = bug.BugRepository
type ChangeCardRepository = changecard.ChangeCardRepository
type DocumentRepository = document.DocumentRepository
type EntityDocumentRepository = entitydoc.EntityDocumentRepository
type EntityHistoryRepository = entityhistory.EntityHistoryRepository
type EntityRelationshipRepository = entityrel.EntityRelationshipRepository
type SearchRepository = search.SearchRepository
type TemplateEnrichmentRepository = templateenrich.TemplateEnrichmentRepository

// Constructor aliases — callers continue using repository.NewXxxRepository(db).

var NewTaskRepository = task.NewTaskRepository
var NewEpicRepository = epic.NewEpicRepository
var NewFeatureRepository = feature.NewFeatureRepository
var NewEntityNoteRepository = note.NewEntityNoteRepository
var NewWorkSessionRepository = worksession.NewWorkSessionRepository
var NewIdeaRepository = idea.NewIdeaRepository
var NewBugRepository = bug.NewBugRepository
var NewChangeCardRepository = changecard.NewChangeCardRepository
var NewDocumentRepository = document.NewDocumentRepository
var NewEntityDocumentRepository = entitydoc.NewEntityDocumentRepository
var NewEntityHistoryRepository = entityhistory.NewEntityHistoryRepository
var NewEntityRelationshipRepository = entityrel.NewEntityRelationshipRepository
var NewSearchRepository = search.NewSearchRepository
var NewTemplateEnrichmentRepository = templateenrich.NewTemplateEnrichmentRepository
var NewTaskHistoryRepository = task.NewHistoryRepository
```

Note: The `NewTaskRepository` alias will need to handle the `NoteCreator` injection. The root-level constructor wrapper will wire `note.EntityNoteRepository` as the `NoteCreator`:

```go
// NewTaskRepository creates a TaskRepository with rejection note support.
func NewTaskRepository(db *DB) *task.TaskRepository {
    noteRepo := note.NewEntityNoteRepository(db)
    return task.NewTaskRepository(db, noteRepo)
}
```

This ensures existing callers of `repository.NewTaskRepository(db)` get the same behavior (rejection notes created during forced status updates) without knowing about the internal coupling resolution.

### 2.6 Repoutil Tracing Design

```go
// internal/repository/repoutil/tracing.go
package repoutil

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// NewTracer creates an OTel tracer scoped to a repository sub-package.
// When observability is disabled (the default), otel.Tracer() returns the
// global no-op tracer, making every span call a sub-microsecond no-op.
func NewTracer(name string) trace.Tracer {
    return otel.Tracer(name)
}

// RecordSpanError records an error on the span and sets its status to Error.
// This is a no-op if err is nil or span is nil.
func RecordSpanError(span trace.Span, err error) {
    if err != nil && span != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
}
```

Each sub-package uses it like:
```go
// internal/repository/task/repository.go
var taskTracer = repoutil.NewTracer("internal/repository/task")
```

This follows the existing pattern in `internal/repository/tracing.go` but with per-package granularity, improving span attribution per E23-F01 observability goals.

### 2.7 Implementation Phases

The work should be done in phases to maintain a compilable codebase at each step:

**Phase 1: Foundation (REQ-F-001 + REQ-F-002)**
1. Create `repoutil/` with exported utilities
2. Update root package files to import from `repoutil` (internal change only)
3. Activate `dbconn.DB` via type alias in `repository.go`
4. Verify: `make fmt && make lint && make test`

**Phase 2: Standalone repositories (REQ-F-003 + REQ-F-007 + REQ-F-009)**
1. Move standalone repositories (idea, bug, changecard, worksession, document, entitydoc, entityhistory, entityrel, search, templateenrich) to sub-packages
2. Create `aliases.go` with type aliases and constructor re-exports
3. Update `polymorphic_doc_adapter.go` to import from `entitydoc`
4. Verify: `make fmt && make lint && make test`

**Phase 3: Resolve coupling and extract note (REQ-F-004 + REQ-F-005)**
1. Define `NoteCreator` interface in `task` package
2. Modify `TaskRepository` constructor to accept optional `NoteCreator`
3. Move `EntityNoteRepository` to `note/`
4. Wire `NoteCreator` in root `aliases.go` constructor wrapper
5. Verify: `make fmt && make lint && make test`

**Phase 4: Core entity repositories (REQ-F-006 + REQ-F-008)**
1. Move `EpicRepository` + relationship to `epic/`
2. Move `FeatureRepository` + relationship to `feature/`
3. Move `TaskRepository` + dependency + relationship + history to `task/`
4. Update `aliases.go` with all core type aliases
5. Clean up root package (REQ-F-008)
6. Verify: `make fmt && make lint && make test`

### 2.8 Risk Mitigations

| Risk | Mitigation |
|---|---|
| Merge conflicts with parallel features (E07-F26, E07-F31, E23-F01) | Type alias shim means zero changes outside `internal/repository/`. Merge conflicts limited to within the repository package itself. |
| Test failures from moved files | Tests move alongside production code. Each phase verified with `make test` before proceeding. |
| `TaskRepository`/`EntityNoteRepository` coupling | Resolved via `NoteCreator` interface injection in Phase 3 before any split. |
| Circular imports between sub-packages | `repoutil` has no intra-repository imports. Sub-packages import only `dbconn` and `repoutil`. Root package imports all sub-packages (one-directional). |
| `dbconn.DB` activation breaks callers | `type DB = dbconn.DB` is a type alias (not a new type); it is assignment-compatible. No caller changes needed. |

---

## 3. Traceability

| Requirement | Epic PRD | Test Strategy |
|---|---|---|
| REQ-F-001 | E07 Enhancements (code organization) | Existing `key_lookup_test.go`, `order_resequence_test.go`, `tracing_test.go` move to `repoutil/` |
| REQ-F-002 | E07 Enhancements | `make test` -- all 74 importing files compile unchanged |
| REQ-F-003 | E07 Enhancements | Each sub-package has its moved test files; `make test` validates |
| REQ-F-004 | E07 Enhancements | New unit test for `TaskRepository` with nil `NoteCreator` (graceful degradation) |
| REQ-F-005 | E07 Enhancements | Existing `entity_note_repository_test.go` moves to `note/` |
| REQ-F-006 | E07 Enhancements | Existing tests move to sub-packages; service interface checks (`interface_checks.go`) still pass |
| REQ-F-007 | E07 Enhancements | Existing `entity_relationship_repository_test.go` moves to `entityrel/` |
| REQ-F-008 | E07 Enhancements | Manual verification: count production `.go` files in root package |
| REQ-F-009 | E07 Enhancements | Verification: no empty directories under `internal/repository/` |
| REQ-NF-001 | E07 Enhancements | `make test` with zero changes outside `internal/repository/` |
| REQ-NF-002 | E07 Enhancements | `go test -bench` shows no regression |
| REQ-NF-003 | E07 Enhancements | `make test` passes; test DB shared via `internal/test/testdb.go` |
| REQ-NF-004 | E07 Enhancements | Tracing test verifies per-package tracer names |

---

*Specification authored: 2026-03-22*
