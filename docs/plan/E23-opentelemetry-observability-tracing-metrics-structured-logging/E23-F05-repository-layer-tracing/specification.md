# E23-F05: Repository Layer Tracing — Technical Specification

**Feature Key:** E23-F05-repository-layer-tracing
**Epic:** E23 — OpenTelemetry Observability (tracing, metrics, structured logging)
**Date:** 2026-03-22
**Status:** Approved for Development

---

## 1. Feature Overview and Purpose

### Problem Statement

See epic PRD (docs/plan/E23-opentelemetry-observability-tracing-metrics-structured-logging/epic.md) for overall observability motivation. Within the epic, repository methods are the innermost layer of data access — they execute every SQLite query — yet they are invisible to tracing. Without repository spans, a trace showing `TaskService.GetTask` gives no insight into whether slowness is in Go code or in the SQLite query itself.

### Solution

Add a single package-level tracer (`var repoTracer = otel.Tracer("internal/repository")`) to each repository file, and wrap the key CRUD and status methods with OpenTelemetry spans. Each span records `db.operation` (SELECT/INSERT/UPDATE/DELETE), `db.table` (tasks/features/epics/entity_notes), and `db.system="sqlite"`. Errors are recorded on the span via `span.RecordError(err)`. Additionally, `shark.db.query.duration` histogram metrics and `shark.db.query.errors` counters are emitted per operation and table.

No constructor changes. When observability is disabled (the default), `otel.Tracer()` returns the global no-op tracer, making every span call a sub-microsecond no-op.

### Purpose within the Epic

F05 is downstream of F01 (observability foundation) and F02 (CLI lifecycle integration). It should be implemented after F01 and F02 are complete so the global TracerProvider is set before any repository method is called. F04 (service layer tracing) is a peer; F05 can proceed in parallel with F04 since both depend only on F01/F02.

---

## 2. Detailed Requirements

### 2.1 Functional Requirements

#### Category: Package-Level Tracer Setup

**REQ-F-001: Package-Level Tracer Variable**
- Description: Each repository file (`task_repository.go`, `feature_repository.go`, `epic_repository.go`, `entity_note_repository.go`) must declare a package-level tracer variable at the top of the file, after the import block.
- Priority: Must-Have
- Acceptance Criteria:
  - `var repoTracer = otel.Tracer("internal/repository")` is present in each repository file.
  - The variable is file-scoped (package-level), not inside any function or struct.
  - No constructor signature changes in any repository type.

**REQ-F-002: No Constructor Changes**
- Description: `NewTaskRepository`, `NewFeatureRepository`, `NewEpicRepository`, and `NewEntityNoteRepository` constructors must remain unchanged. The tracer is obtained via the global OTel API, not injected.
- Priority: Must-Have
- Acceptance Criteria:
  - Constructor signatures match their current definitions exactly.
  - Existing callers in `internal/cli/services_global.go` and `cmd/server/` require no changes.

#### Category: Span Instrumentation

**REQ-F-003: Span Creation Pattern**
- Description: Each instrumented method must start a span at the top (before any other logic) and defer `span.End()`. The enriched context must be passed to all downstream database calls.
- Priority: Must-Have
- Acceptance Criteria:
  - `ctx, span := repoTracer.Start(ctx, "RepoType.MethodName", ...)` is the first statement in each instrumented method body.
  - `defer span.End()` immediately follows `Start`.
  - All `r.db.*Context(ctx, ...)` calls use the updated `ctx` (not the original).
  - Errors are recorded: `span.RecordError(err)` before returning any non-nil error.

**REQ-F-004: Required Span Attributes**
- Description: Every span must include `db.operation`, `db.table`, and `db.system` attributes at span creation time.
- Priority: Must-Have
- Attribute definitions:
  - `db.operation`: One of `"SELECT"`, `"INSERT"`, `"UPDATE"`, `"DELETE"` matching the primary SQL operation.
  - `db.table`: The primary table being accessed: `"tasks"`, `"features"`, `"epics"`, `"entity_notes"`.
  - `db.system`: Always `"sqlite"`.
- Acceptance Criteria:
  - All three attributes are set via `trace.WithAttributes(...)` in the `Start` call.
  - The `attribute` package from `go.opentelemetry.io/otel/attribute` is used (not string concatenation).

**REQ-F-005: Error Recording**
- Description: When a method returns a non-nil error, the error must be recorded on the span before the return statement.
- Priority: Must-Have
- Acceptance Criteria:
  - `span.RecordError(err)` appears in every error return path.
  - `span.SetStatus(codes.Error, err.Error())` is also set on error paths using `go.opentelemetry.io/otel/codes`.

#### Category: Methods to Instrument

**REQ-F-006: TaskRepository Instrumented Methods**
- Description: The following `TaskRepository` methods must be instrumented with spans.
- Priority: Must-Have
- Methods:
  - `Create` — db.operation=INSERT, db.table=tasks
  - `GetByID` — db.operation=SELECT, db.table=tasks
  - `GetByKey` — db.operation=SELECT, db.table=tasks
  - `Update` — db.operation=UPDATE, db.table=tasks
  - `Delete` — db.operation=DELETE, db.table=tasks
  - `List` — db.operation=SELECT, db.table=tasks
  - `ListByFeature` — db.operation=SELECT, db.table=tasks
  - `ListByEpic` — db.operation=SELECT, db.table=tasks
  - `UpdateStatus` (delegates to `updateStatusForcedInternal`) — db.operation=UPDATE, db.table=tasks
  - `UpdateStatusForced` — db.operation=UPDATE, db.table=tasks
  - `FilterCombined` — db.operation=SELECT, db.table=tasks
- Not instrumented (internal helpers, batch ops, or specialized):
  - `listByFeatureInTx` (private; span already present on caller)
  - `updateStatusForcedInternal` (private; span present on public callers)
  - `BulkCreate` (low-frequency admin operation; exclude from initial scope)
  - `GetStatusBreakdownMapBatch` (analytics query; exclude from initial scope)
  - `ValidateTaskDependencies` (validation helper; not a primary DB op)

**REQ-F-007: FeatureRepository Instrumented Methods**
- Description: The following `FeatureRepository` methods must be instrumented.
- Priority: Must-Have
- Methods:
  - `Create` — db.operation=INSERT, db.table=features
  - `GetByID` — db.operation=SELECT, db.table=features
  - `GetByKey` — db.operation=SELECT, db.table=features
  - `Update` — db.operation=UPDATE, db.table=features
  - `Delete` — db.operation=DELETE, db.table=features
  - `List` — db.operation=SELECT, db.table=features
  - `ListByEpic` — db.operation=SELECT, db.table=features
  - `UpdateStatus` — db.operation=UPDATE, db.table=features
- Not instrumented (private helpers, low-frequency, or analytics):
  - `getByExactKey`, `getByNumericKey`, `getBySluggedKey` (private lookup helpers)
  - `listByEpicInTx` (private transaction helper)
  - `CascadeStatusToTasks`, `GetFeatureDisplayDataRaw` (complex analytics; out of scope)

**REQ-F-008: EpicRepository Instrumented Methods**
- Description: The following `EpicRepository` methods must be instrumented.
- Priority: Must-Have
- Methods:
  - `Create` — db.operation=INSERT, db.table=epics
  - `GetByID` — db.operation=SELECT, db.table=epics
  - `GetByKey` — db.operation=SELECT, db.table=epics
  - `Update` — db.operation=UPDATE, db.table=epics
  - `Delete` — db.operation=DELETE, db.table=epics
  - `List` — db.operation=SELECT, db.table=epics
  - `UpdateStatus` — db.operation=UPDATE, db.table=epics
- Not instrumented (complex analytics or cascade operations):
  - `CascadeStatusToFeaturesAndTasks`, `GetEpicDisplayDataRaw`
  - `GetFeatureProgressDataByEpic`, `GetFeatureStatusBreakdown`, `GetFeatureStatusRollup`, `GetTaskStatusRollup`

**REQ-F-009: EntityNoteRepository Instrumented Methods**
- Description: The following `EntityNoteRepository` methods must be instrumented.
- Priority: Must-Have
- Methods:
  - `Create` — db.operation=INSERT, db.table=entity_notes
  - `GetByID` — db.operation=SELECT, db.table=entity_notes
  - `GetByEntity` — db.operation=SELECT, db.table=entity_notes
  - `Delete` — db.operation=DELETE, db.table=entity_notes
  - `CreateRejectionNote` — db.operation=INSERT, db.table=entity_notes
- Not instrumented (complex multi-join queries; exclude from initial scope):
  - `Search`, `SearchWithTimePeriod`, `GetByEntityAndType`, `GetRejectionHistory`

#### Category: Metrics

**REQ-F-010: DB Query Duration Histogram**
- Description: Each instrumented method must record a `shark.db.query.duration` histogram metric on completion, with labels for `operation` and `table`.
- Priority: Should-Have
- Acceptance Criteria:
  - Uses `DBMetrics.RecordQueryDuration(operation, table, duration, err)` from `internal/observability/metrics.go` (defined in F01).
  - Duration measured from span start to end using `time.Since(start)`.
  - Both success and error cases are recorded (error case uses the `err != nil` path).

**REQ-F-011: DB Query Error Counter**
- Description: When a repository method returns an error, a `shark.db.query.errors` counter must be incremented.
- Priority: Should-Have
- Acceptance Criteria:
  - Uses `DBMetrics.RecordQueryDuration` which internally distinguishes error vs success.
  - Counter labels include `operation` and `table`.

### 2.2 Non-Functional Requirements

**REQ-NF-001: No-Op Overhead**
- Description: When observability is disabled (default), span creation and attribute setting must add less than 1 microsecond of overhead per repository call.
- Measurement: Benchmark `BenchmarkTaskRepository_GetByKey` with and without observability enabled.
- Target: Less than 1% regression on `shark get E01-F01-001` end-to-end latency.
- Justification: Repositories are called on every CLI command; any meaningful overhead is unacceptable for the default (disabled) case.

**REQ-NF-002: No Constructor Changes**
- Description: All repository constructors (`NewTaskRepository`, etc.) must remain backward-compatible. This is a hard constraint from ADR-4.
- Measurement: Compiler check — any caller that compiles before must compile after.

**REQ-NF-003: No Test Changes Required**
- Description: Existing repository tests must pass without modification. The no-op global TracerProvider (default) ensures span calls are no-ops in tests.
- Measurement: `make test` passes after all instrumentation is added.

---

## 3. Architecture

### 3.1 Component Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/repository/task_repository.go` | MODIFY | Add `repoTracer` var; add spans to 11 methods |
| `internal/repository/feature_repository.go` | MODIFY | Add `repoTracer` var; add spans to 8 methods |
| `internal/repository/epic_repository.go` | MODIFY | Add `repoTracer` var; add spans to 7 methods |
| `internal/repository/entity_note_repository.go` | MODIFY | Add `repoTracer` var; add spans to 5 methods |

**No other files are modified.** This feature is entirely self-contained within `internal/repository/`.

### 3.2 Data Model Changes

None. Observability is a cross-cutting concern that does not modify any database schema, entity models, or API contracts (see epic architecture ADR-4, section 3).

### 3.3 Package-Level Tracer Pattern

Per ADR-4 in `docs/plan/E23-opentelemetry-observability-tracing-metrics-structured-logging/architecture.md`:

```go
// At package level in each repository file, after imports:
var repoTracer = otel.Tracer("internal/repository")
```

This is the standard OTel Go idiom for library/package code. When `observability.Init()` sets the global TracerProvider (in `PersistentPreRunE` via F02), `repoTracer` picks it up. When observability is disabled (the default), the global TracerProvider is the SDK default no-op provider, so `repoTracer` is a no-op tracer.

Required imports to add to each repository file:
```go
"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
"go.opentelemetry.io/otel/codes"
"go.opentelemetry.io/otel/trace"
```

### 3.4 Method Instrumentation Pattern

Standard 4-line addition at the top of each instrumented method:

```go
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := repoTracer.Start(ctx, "TaskRepository.GetByKey",
        trace.WithAttributes(
            attribute.String("db.operation", "SELECT"),
            attribute.String("db.table", "tasks"),
            attribute.String("db.system", "sqlite"),
            attribute.String("db.key", key), // optional entity key for filtering
        ))
    defer span.End()

    // ... existing implementation unchanged ...
    // On error return:
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    // ...
}
```

For write operations, add the entity key as an optional attribute to aid trace filtering:
- `Create`: `attribute.String("db.key", task.Key)`
- `Update`: `attribute.String("db.key", task.Key)`
- `GetByKey`: `attribute.String("db.key", key)`
- `Delete`: `attribute.Int64("db.id", id)` (no key available)

### 3.5 Span Naming Convention

Span names follow the pattern `RepositoryType.MethodName`:
- `TaskRepository.Create`
- `TaskRepository.GetByKey`
- `TaskRepository.Update`
- `TaskRepository.Delete`
- `TaskRepository.List`
- `TaskRepository.ListByFeature`
- `TaskRepository.ListByEpic`
- `TaskRepository.UpdateStatus`
- `TaskRepository.UpdateStatusForced`
- `TaskRepository.FilterCombined`
- `FeatureRepository.Create`
- `FeatureRepository.GetByKey`
- ... (same pattern for all)

This naming is consistent with the service layer tracing in F04 (`TaskService.GetTask`, etc.) and produces clean waterfall traces in Jaeger/Tempo.

### 3.6 Metrics Integration

Uses `DBMetrics` from `internal/observability/metrics.go` (defined in F01). Since metrics require an initialized `MeterProvider`, and F02 sets that up in `PersistentPreRunE`, the pattern is:

```go
import "github.com/jwwelbor/shark-task-manager/internal/observability"

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
    start := time.Now()
    ctx, span := repoTracer.Start(ctx, "TaskRepository.Create", ...)
    defer func() {
        span.End()
        // Metrics recorded after span ends (span.End() records duration internally via OTel SDK)
    }()

    // ... implementation ...
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        observability.RecordDBQuery(ctx, "INSERT", "tasks", time.Since(start), err)
        return err
    }

    observability.RecordDBQuery(ctx, "INSERT", "tasks", time.Since(start), nil)
    return nil
}
```

**Note:** If `internal/observability` package is not yet available (F01 not merged), the metrics REQ-F-010 and REQ-F-011 should be deferred and implemented in a follow-up. The span instrumentation (REQ-F-001 through REQ-F-009) is independent of metrics and can ship first.

### 3.7 Import Management

Each repository file will need these new imports:

```go
import (
    // Existing imports remain unchanged
    // ...

    // New OTel imports
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)
```

The `trace` import is only needed for `trace.WithAttributes(...)`. If the linter flags unused imports, verify `trace.WithAttributes` is used (it is, in the `Start` call).

### 3.8 Dependency Requirements

F05 depends on:
- **F01 (Observability Foundation)**: Provides `otel.Tracer()` global API and `DBMetrics` helper. The OTel API package (`go.opentelemetry.io/otel`) must be in `go.mod` before F05 can compile.
- **F02 (CLI Lifecycle Integration)**: Wires the real TracerProvider into the global. Without F02, the tracer is no-op even when observability is enabled. F05 compiles and runs correctly without F02; spans are just no-ops.

---

## 4. Acceptance Criteria

### AC-001: Compilation

**Given** all four repository files are modified per this spec,
**When** `make build` is run,
**Then** the build succeeds with no compilation errors or new linting warnings.

### AC-002: Test Suite

**Given** the instrumented repository files,
**When** `make test` is run,
**Then** all existing repository tests pass without modification (no-op tracer in test environment).

### AC-003: Span Visibility (Integration)

**Given** observability is enabled (`observability.enabled=true`, `exporter=stdout`) in `.sharkconfig.json`,
**When** `shark get E01-F01-001` is run,
**Then** span output on stderr includes at minimum:
- A span named `TaskRepository.GetByKey`
- Attributes `db.operation=SELECT`, `db.table=tasks`, `db.system=sqlite`
- The span is nested under a parent `TaskService.*` span (from F04)

### AC-004: No-Op Overhead

**Given** observability is disabled (default config),
**When** `shark get E01-F01-001` is run 1000 times,
**Then** median latency regression is less than 1% compared to pre-F05 baseline.

### AC-005: Error Recording

**Given** a repository operation fails (e.g., database returns `sql.ErrNoRows`),
**When** the method returns an error,
**Then** the span has `span.RecordError(err)` called and `span.SetStatus(codes.Error, ...)` set.

### AC-006: Attribute Correctness

**Given** a span from `FeatureRepository.Create`,
**When** inspected in an OTel backend,
**Then** `db.operation="INSERT"`, `db.table="features"`, `db.system="sqlite"` are all present.

---

## 5. Out of Scope

1. **Private/internal helper methods** (`getByExactKey`, `getByNumericKey`, `getBySluggedKey`, `listByFeatureInTx`, `updateStatusForcedInternal`) — callers are already instrumented; double-counting would skew metrics.
2. **Analytics and batch methods** (`GetStatusBreakdownMapBatch`, `BulkCreate`, `CascadeStatusToFeaturesAndTasks`, `GetEpicDisplayDataRaw`, `GetFeatureDisplayDataRaw`, `GetFeatureProgressDataByEpic`, `Search`, `SearchWithTimePeriod`) — these are admin/analytics paths; adding spans is a future enhancement.
3. **Constructor injection of tracer** — ruled out by ADR-4; package-level tracer only.
4. **`task_history_repository.go`** — History repository is an audit trail; tracing is lower priority. Deferred to a follow-up.
5. **Prometheus exporter or pull-based metrics** — ruled out by epic architecture (CLI tool lifecycle incompatible with pull model).
6. **SQL query text as span attribute** — Excluded for security (queries may contain user-supplied values). Only operation type and table are recorded.

---

*Last Updated: 2026-03-22*
