# E23-F04 Service Layer Tracing — Specification

**Feature**: Service Layer Tracing
**Epic**: E23 — OpenTelemetry Observability (Tracing, Metrics, Structured Logging)
**Complexity**: STANDARD (score: 9/27)
**Status**: in_specification

---

## Context

See epic PRD and epic architecture.md for system-level OTel decisions, provider initialization, and the `GetTracer(name string) trace.Tracer` accessor already provided by `internal/cli/observability_global.go` (F01/F02 work).

This feature instruments the service layer methods so that every significant operation in `TaskService`, `FeatureService`, and `EpicService` emits an OpenTelemetry span. Spans nest automatically because the enriched `context.Context` flows through to repository calls.

---

## Requirements

### Functional Requirements

**REQ-F-001: TaskService tracer field**
- **Description**: `TaskService` struct gains an optional `trace.Tracer` field. When nil it falls back to `otel.Tracer("shark/services/task")` (no-op by default until provider is wired).
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - `TaskService` has a `tracer trace.Tracer` field.
  - `SetTracer(t trace.Tracer)` setter is available on `*TaskService`.
  - When tracer is nil at call time, `otel.Tracer("shark/services/task")` is used (never panics).

**REQ-F-002: FeatureService tracer field**
- **Description**: Same pattern as REQ-F-001 for `FeatureService`.
- **Priority**: Must-Have
- **Acceptance Criteria**: `FeatureService` has `tracer` field and `SetTracer` setter.

**REQ-F-003: EpicService tracer field**
- **Description**: Same pattern as REQ-F-001 for `EpicService`.
- **Priority**: Must-Have
- **Acceptance Criteria**: `EpicService` has `tracer` field and `SetTracer` setter.

**REQ-F-004: Instrumented TaskService methods**
- **Description**: The following `TaskService` methods wrap their body in a span:
  `CreateTask`, `GetTask`, `UpdateTask`, `DeleteTask`, `ListTasks`, `TransitionStatus`, `GetNextStatus`.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - Each method starts a child span with `s.getTracer().Start(ctx, "TaskService.<method>")`.
  - `defer span.End()` is the next statement.
  - On error, `span.RecordError(err)` and `span.SetStatus(codes.Error, err.Error())` are called before returning.
  - Span attributes include `task.key` where applicable.
  - The enriched `ctx` is passed to all downstream repository calls.

**REQ-F-005: Instrumented FeatureService methods**
- **Description**: `GetFeature`, `ListFeatures`, `TransitionStatus`, `CompleteFeature`, `GetProgress`.
- **Priority**: Must-Have
- **Acceptance Criteria**: Same pattern as REQ-F-004 with `feature.key` attribute.

**REQ-F-006: Instrumented EpicService methods**
- **Description**: `GetEpic`, `ListEpics`, `TransitionStatus`, `CompleteEpic` (and any equivalent in epic_service.go).
- **Priority**: Must-Have
- **Acceptance Criteria**: Same pattern as REQ-F-004 with `epic.key` attribute.

**REQ-F-007: Wire tracers in services_global.go**
- **Description**: `GetTaskService()`, `GetFeatureService()`, and `GetEpicService()` (and their `WithXxx` variants) call `SetTracer(cli.GetTracer("shark/services/<entity>"))` after constructing each service.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - All three global accessors wire a named tracer.
  - `GetTracer` is the existing function from `internal/cli/observability_global.go`.
  - Build compiles with no new imports beyond `go.opentelemetry.io/otel/trace` in service files and `internal/cli` referencing `observability_global.go`.

**REQ-F-008: No-op when OTel is disabled**
- **Description**: When `ObservabilityConfig.Enabled = false` (or provider not initialized), all spans are no-ops — no performance cost, no panics.
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - Existing tests continue to pass without any OTel initialization.
  - CLI commands that don't call `InitObservability` still function normally.

### Non-Functional Requirements

**REQ-NF-001: Overhead — noop path**
- Noop tracer overhead per method call must be negligible (OTel SDK guarantee via `trace.NewNoopTracerProvider()`).

**REQ-NF-002: Context propagation**
- Span context must be propagated via the `context.Context` parameter, not global state, so concurrent requests remain isolated.

### Acceptance Criteria (Feature-Level)

**Scenario 1: Span appears in stdout when tracing is enabled**
- Given: `SHARK_OTEL_ENABLED=true OTEL_EXPORTER=stdout`
- When: `shark task get E07-F01-001` is executed
- Then: stderr contains a JSON span with `name: "TaskService.GetTask"` and attribute `task.key = "E07-F01-001"`

**Scenario 2: No span output when tracing is disabled**
- Given: default config (OTel disabled)
- When: any shark command is executed
- Then: no OTel output on stderr; command exits normally

**Scenario 3: Nested span chain**
- Given: tracing enabled
- When: `shark task create E07 F01 "My task"` is executed
- Then: span `TaskService.CreateTask` appears, and its TraceID matches any repository-layer spans (F05 concern, but context must flow through)

### Out of Scope

- Repository-layer spans (F05)
- CLI command-level spans (F06/F07)
- BugService and ChangeCardService (not yet migrated to service layer; can be added as follow-on)
- Metrics instrumentation (F06)

---

## Architecture

### Component Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/services/task_service.go` | Modify | Add `tracer trace.Tracer` field; add `SetTracer`; add `getTracer` helper; instrument 7 methods |
| `internal/services/feature_service.go` | Modify | Same pattern; instrument 5 methods |
| `internal/services/epic_service.go` | Modify | Same pattern; instrument 4 methods |
| `internal/cli/services_global.go` | Modify | Call `svc.SetTracer(GetTracer("shark/services/<entity>"))` in each accessor |
| `internal/services/task_service_tracing_test.go` | Create | Unit tests for tracing behaviour using mock tracer |
| `internal/services/feature_service_tracing_test.go` | Create | Same |
| `internal/services/epic_service_tracing_test.go` | Create | Same |

No data model changes. No database migrations.

### Tracer Field Pattern (applied identically to all three services)

```go
// In struct definition (e.g. TaskService)
tracer trace.Tracer // optional; defaults to otel.Tracer("shark/services/task") if nil

// Setter — wired by services_global.go
func (s *TaskService) SetTracer(t trace.Tracer) {
    s.tracer = t
}

// Private helper — avoids nil check repetition in every method
func (s *TaskService) getTracer() trace.Tracer {
    if s.tracer != nil {
        return s.tracer
    }
    return otel.Tracer("shark/services/task")
}
```

Follows the optional-dependency / graceful-degradation pattern already used by `SetDocRepo`, `SetHistoryRepo`, etc. in `task_service.go`.

### Span Instrumentation Pattern

```go
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
    ctx, span := s.getTracer().Start(ctx, "TaskService.GetTask",
        trace.WithAttributes(attribute.String("task.key", key)),
    )
    defer span.End()

    task, err := s.repo.GetByKey(ctx, key)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("failed to get task %s: %w", key, err)
    }
    return task, nil
}
```

Required imports (per service file):
```go
"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
"go.opentelemetry.io/otel/codes"
"go.opentelemetry.io/otel/trace"
```

### Wiring in services_global.go

```go
func GetTaskService() *services.TaskService {
    // ... existing construction ...
    svc.SetTracer(GetTracer("shark/services/task"))
    return svc
}
```

`GetTracer` is already defined in `internal/cli/observability_global.go` as:
```go
func GetTracer(name string) trace.Tracer { return otel.Tracer(name) }
```

### Methods to Instrument

**TaskService** (7 methods):
- `CreateTask` — attribute: `task.epic_key`, `task.feature_key`, `task.title`
- `GetTask` — attribute: `task.key`
- `UpdateTask` — attribute: `task.key`
- `DeleteTask` — attribute: `task.key`
- `ListTasks` — attribute: `task.filter` (stringified filter summary)
- `TransitionStatus` — attribute: `task.key`, `task.target_status`
- `GetNextStatus` — attribute: `task.key`

**FeatureService** (5 methods):
- `GetFeature` — attribute: `feature.key`
- `ListFeatures` — attribute: none (or `feature.epic_key` if filter has one)
- `TransitionStatus` — attribute: `feature.key`, `feature.target_status`
- `CompleteFeature` — attribute: `feature.key`
- `GetProgress` — attribute: `feature.key`

**EpicService** (4 methods — verify against actual epic_service.go public methods):
- `GetEpic` — attribute: `epic.key`
- `ListEpics` — no span attribute needed
- `TransitionStatus` — attribute: `epic.key`, `epic.target_status`
- `CompleteEpic` (or equivalent) — attribute: `epic.key`

### Key Technical Decisions

1. **Setter over constructor injection**: The three services use many optional dependencies added via setters after construction (`SetDocRepo`, `SetHistoryRepo`, etc.). A `SetTracer` setter follows the established pattern and avoids changing constructor signatures (which would require updating all test construction call sites).

2. **`getTracer()` helper**: Avoids repeating the nil-check in every instrumented method. Returns OTel global tracer (which is noop until InitObservability wires an SDK provider).

3. **Span name format `"<ServiceType>.<MethodName>"`**: Consistent with OpenTelemetry semantic convention for `rpc.method`-style naming; enables filtering by service in Jaeger/Tempo.

4. **`codes` and `attribute` packages**: Standard OTel Go packages. Import path `go.opentelemetry.io/otel/codes` and `go.opentelemetry.io/otel/attribute` — both already present in go.mod via F01.

5. **Context threading**: `ctx, span := s.getTracer().Start(ctx, ...)` overwrites the local `ctx` variable so all downstream calls (repo, sub-service) receive the child context automatically.
