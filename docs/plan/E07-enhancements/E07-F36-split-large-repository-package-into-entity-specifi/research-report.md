# E07-F36 Research Report: Split Large Repository Package into Entity-Specific Sub-Packages

**Feature**: E07-F36 — Split large repository package into entity-specific sub-packages
**Date**: 2026-03-22
**Status**: Research complete

---

## 1. Executive Summary

The `internal/repository` package has grown to **79 Go files** containing **30,867 total lines** (9,714 production lines + 21,153 test lines). One file alone (`task_repository.go`) is 1,967 lines. A sub-package split has been **partially started**: the `dbconn` sub-package (`internal/repository/dbconn/`) was extracted with the `DB` wrapper type, and five empty placeholder directories (`epic/`, `feature/`, `task/`, `note/`, `worksession/`) were created. No code has actually been moved into those directories yet.

The feature's core challenge is that 71 files across the codebase import `internal/repository` using the concrete `repository.*` type names. Any split must preserve backward compatibility for all callers.

---

## 2. Current State: What EXISTS

### 2.1 Repository Package Structure

**Production files** (`internal/repository/`):

| File | Lines | Entity Group |
|------|-------|--------------|
| `task_repository.go` | 1,967 | Task |
| `feature_repository.go` | 1,087 | Feature |
| `epic_repository.go` | 791 | Epic |
| `entity_note_repository.go` | 694 | Shared/Cross-entity |
| `work_session_repository.go` | 523 | WorkSession |
| `task_dependency.go` | 501 | Task |
| `task_relationship_repository.go` | 438 | Task |
| `search_repository.go` | 395 | Shared |
| `change_card_repository.go` | 393 | ChangeCard |
| `task_history_repository.go` | 369 | Task |
| `bug_repository.go` | 353 | Bug |
| `idea_repository.go` | 315 | Idea |
| `entity_relationship_repository.go` | 292 | Shared/Cross-entity |
| `template_enrichment_repository.go` | 209 | Shared |
| `feature_relationship_repository.go` | 199 | Feature |
| `epic_relationship_repository.go` | 199 | Epic |
| `bug_aggregate.go` | 192 | Bug |
| `document_repository.go` | 144 | Document |
| `entity_document_repository.go` | 125 | Shared/Cross-entity |
| `entity_history_repository.go` | 124 | Shared/Cross-entity |
| `change_card_aggregate.go` | 118 | ChangeCard |
| `order_resequence.go` | 107 | Shared utility |
| `key_lookup.go` | 91 | Shared utility |
| `repository.go` | 27 | Core (DB wrapper) |
| `tracing.go` | 21 | Shared infrastructure |
| `polymorphic_doc_adapter.go` | 41 | Adapter/Shared |

**Already-extracted sub-package:**

- `internal/repository/dbconn/db.go` — `DB` struct + transaction helpers
  - Package comment explicitly states intent: "extracted so entity-specific sub-packages can import the connection type without creating import cycles"
  - Currently **not imported by any file** (it exists but `repository.go` still defines its own `DB` type)

**Empty placeholder directories** (no Go files):
- `internal/repository/epic/`
- `internal/repository/feature/`
- `internal/repository/task/`
- `internal/repository/note/`
- `internal/repository/worksession/`

### 2.2 Concrete Types Referenced Outside the Package

The following `repository.*` types are directly referenced in non-repository code (71 importing files, 26 non-test):

```
repository.BugRepository
repository.ChangeCardRepository
repository.DB
repository.EntityNoteRepository
repository.EpicDisplayDataRaw
repository.EpicRepository
repository.FeatureDisplayDataRaw
repository.FeatureProgressData
repository.FeatureRepository
repository.HistoryFilters
repository.IdeaRepository
repository.SearchRepository
repository.TaskDisplayDataRaw
repository.TaskHistoryRepository
repository.TaskRepository
repository.WorkSessionRepository
```

Key callers of concrete types:
- `/internal/cli/services_global.go` — constructs all repository types for service wiring
- `/internal/cli/db_global.go`, `/internal/cli/db_init.go` — DB initialization
- `/internal/cli/service_accessors.go` — service accessor pattern
- `/internal/services/epic_service.go` — references `repository.FeatureProgressData`, `repository.EpicDisplayDataRaw`
- `/internal/services/task_service.go` — references `repository.TaskDisplayDataRaw`, `repository.HistoryFilters`
- `/cmd/server/services.go`, `/cmd/demo/main.go` — server wiring
- `/internal/status/`, `/internal/validation/`, `/internal/taskcreation/` — direct imports

### 2.3 Intra-Package Cross-Type Dependencies (Critical Finding)

Within the repository package, types depend on each other:

1. **`TaskRepository` → `EntityNoteRepository`**: `task_repository.go` directly instantiates `EntityNoteRepository` via `NewEntityNoteRepository(r.db)` in `updateStatusForcedInternal()` (line ~1016) and `StatusUpdateRaw()` (line ~1110). This creates a coupling that **prevents naively moving these to separate packages**.

2. **`task_dependency.go`** is part of `TaskRepository` (methods on `*TaskRepository`) — it cannot be split without moving `TaskRepository` itself.

3. **`tracing.go`** defines `repoTracer` (package-level var) and `recordSpanError()` — used by ALL repository types in the package. Moving any type to a sub-package requires that sub-package to have its own tracer or import from a shared location.

4. **`key_lookup.go`** defines shared key-parsing utilities (`ContainsHyphen`, `IsNumeric`, `SplitAtFirstHyphen`, `SplitAtNthHyphen`) — used by epic, feature, and task repositories.

5. **`order_resequence.go`** defines `resequenceOrders()` — a private function used by feature and task repositories.

---

## 3. Integration Points

### 3.1 Service Layer Integration (Primary Consumer)

Services use repository types via interfaces defined in the service package. The concrete `repository.*` types are only referenced at wiring points:

```
internal/cli/services_global.go    ← primary CLI wiring
cmd/server/services.go             ← HTTP server wiring
internal/services/interface_checks.go ← compile-time checks
```

### 3.2 Data Type (DTO) References

Several `repository.*` structs are used as return types in service interfaces:
- `repository.FeatureProgressData` — used in `epic_service.go`'s interface
- `repository.EpicDisplayDataRaw` — used in `epic_service.go`'s interface
- `repository.TaskDisplayDataRaw` — used in `task_service.go`'s interface
- `repository.HistoryFilters` — used in `task_service.go`'s interface

These would need package path updates if moved to sub-packages.

### 3.3 Test Infrastructure

The test database `internal/repository/test-shark-tasks.db` and test helper `internal/test/testdb.go` are shared across all 42 test files. Tests use `package repository` (same package), so they access private helpers directly.

### 3.4 OpenTelemetry Tracing

The `repoTracer` package-level variable in `tracing.go` is referenced by all repository files. Moving files to sub-packages would require each sub-package to define its own tracer (or import from a shared `tracing` helper).

---

## 4. Inter-Feature Dependency Map (E07)

Features that interact with E07-F36:

| Feature | Relationship |
|---------|-------------|
| **E07-F31** (Unified Entity Display Rendering) | Uses `repository.*DisplayDataRaw` types — a split must not break these type references |
| **E07-F26** (Centralized Workflow Service Authority) | Affects how `TaskRepository` validates workflows — status: active |
| **E23-F01** (Observability Foundation) | Added `tracing.go` and OTel instrumentation to ALL repository types — any split must carry tracing forward in sub-packages |
| **E07-F22** (Rejection Reason) | Added `EntityNoteRepository` coupling inside `TaskRepository.StatusUpdateRaw()` — the key intra-package coupling discovered in this research |

---

## 5. Extension-vs-New Analysis

### 5.1 What Can Be Extended (Extend)

| Component | Extension Strategy |
|-----------|-------------------|
| `dbconn/db.go` | Already extracted. Needs `repository.go` updated to re-export `DB` from dbconn OR callers updated to use `dbconn.DB`. Extension = wire it in. |
| Empty placeholder directories | Were clearly created for this purpose. Extend by adding `.go` files with the package declaration and moving types. |
| `tracing.go` | Can be extended to become a shared `internal/repository/shared/tracing.go` or each sub-package gets its own tracer registered under its own instrumentation name. |
| `key_lookup.go` | Already qualifies as a shared utility. Move to `internal/repository/shared/` sub-package. |

### 5.2 What Must Be Newly Created

| Component | What's Needed |
|-----------|--------------|
| Re-export shim layer | If full backward compat is required, `internal/repository/*.go` can re-export types from sub-packages using type aliases. Go supports `type TaskRepository = task.TaskRepository`. |
| Sub-package package declarations | Each of the 5 placeholder dirs needs at minimum a `doc.go` or the moved files. |
| Updated import paths in callers | 71 files need import path updates if no re-export shim is used. |

---

## 6. Recommended Implementation Approach

### Approach A: Re-Export Shim (Lower Risk, Recommended)

Move types to sub-packages but add Go type aliases in the root `repository` package so all 71 callers need zero changes.

```
internal/repository/
├── dbconn/              ← already done
│   └── db.go
├── shared/
│   ├── tracing.go       ← move from root, rename tracer
│   ├── key_lookup.go    ← move from root
│   └── order_resequence.go ← move from root
├── task/
│   ├── repository.go    ← TaskRepository + TaskDisplayDataRaw
│   ├── dependency.go    ← task_dependency.go methods
│   ├── relationship.go  ← task_relationship_repository.go
│   └── history.go       ← task_history_repository.go
├── epic/
│   └── repository.go    ← EpicRepository + related types
├── feature/
│   └── repository.go    ← FeatureRepository + related types
├── note/
│   └── repository.go    ← EntityNoteRepository
├── worksession/
│   └── repository.go    ← WorkSessionRepository
├── aliases.go           ← type aliases re-exporting all types for backward compat
└── repository.go        ← keep DB type (or import from dbconn)
```

The `aliases.go` file would contain:
```go
type TaskRepository = task.TaskRepository
type EpicRepository = epic.EpicRepository
// etc.
```

### Approach B: Full Migration (Higher Risk)

Move types and update all 71 importing files. No shim layer. This is the "clean" approach but requires touching ~71 files and may break during migration if tests are run mid-migration.

### Key Implementation Risks

1. **`TaskRepository` → `EntityNoteRepository` coupling**: Before splitting these into separate packages (`task/` and `note/`), the direct instantiation must be resolved. Options:
   - Pass `EntityNoteRepository` via dependency injection into `TaskRepository`
   - Create a `note.Creator` interface that `task.TaskRepository` accepts
   - Keep both in the same sub-package temporarily

2. **`tracing.go`'s package-level `repoTracer`**: Cannot be shared across packages without either (a) each sub-package defines its own tracer with a unique name, or (b) a `shared/tracing.go` provides a factory function.

3. **Test files**: 42 test files use `package repository` and access package-internal helpers. Moving production code while keeping test files in the root package would break those tests.

4. **`dbconn.DB` vs `repository.DB`**: Currently `dbconn.DB` exists but `repository.DB` is still the type used everywhere. The first concrete step is to either:
   - Make `repository.DB` an alias for `dbconn.DB`, OR
   - Keep `repository.DB` as the authoritative type and move `dbconn/db.go` back into root

---

## 7. Actionable Summary for Architect

### Current Position
- `dbconn/` sub-package is extracted but disconnected (nobody imports it)
- 5 empty placeholder dirs exist as targets
- The root `repository` package is a single Go package with 79 files and no intra-package imports
- Primary coupling risk: `TaskRepository` directly constructs `EntityNoteRepository` inline

### Recommended First Steps (Phased)

**Phase 1 — No-risk shared utilities**:
Move `key_lookup.go`, `order_resequence.go`, and `tracing.go` to `internal/repository/shared/`. These have no type dependencies on other repository types. Update all references within the root package (only internal).

**Phase 2 — Resolve `dbconn` situation**:
Either activate `dbconn` by making `repository.DB` an alias for `dbconn.DB`, or retire `dbconn/db.go` and keep `DB` in root. This decision gates all sub-package splits since every sub-package needs the `DB` type.

**Phase 3 — Split leaf repositories first** (no intra-package dependents):
- `WorkSessionRepository` → `worksession/`
- `EntityNoteRepository` → `note/` (after resolving TaskRepository coupling)
- `SearchRepository` — standalone, no intra-package type dependencies
- `IdeaRepository` — standalone
- `BugRepository`, `ChangeCardRepository` — standalone

**Phase 4 — Split core entity repositories** (high external reference count):
- `EpicRepository` → `epic/`
- `FeatureRepository` → `feature/`
- `TaskRepository` + `task_dependency.go` + `task_relationship_repository.go` → `task/`

Add type aliases in root `repository` package to preserve all 71 callers.

### Files That Must Change for Any Split

| File | Required Change |
|------|----------------|
| `/internal/repository/task_repository.go` | Must inject `EntityNoteRepository` instead of constructing it directly |
| `/internal/repository/tracing.go` | Must move to `shared/` or duplicate per sub-package |
| `/internal/repository/repository.go` | Must resolve `DB` type (alias to dbconn or keep in root) |
| All 71 importing files | May need import path updates (avoided with Approach A shim) |

---

*This report was produced via codebase analysis. All file paths are absolute to the project root at `/home/jwwel/projects/shark-task-manager`.*
