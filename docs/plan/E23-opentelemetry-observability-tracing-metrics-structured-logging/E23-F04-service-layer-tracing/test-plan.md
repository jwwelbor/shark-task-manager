# E23-F04 Service Layer Tracing — Test Plan

**Feature**: Service Layer Tracing
**Epic**: E23 — OpenTelemetry Observability
**Test file locations**: `internal/services/*_tracing_test.go`

---

## Approach

Service tracing tests use the OTel in-memory `sdktrace.NewSimpleSpanProcessor` + `tracetest.NewInMemoryExporter()` to capture spans synchronously without any network or file I/O. This follows the existing pattern of mocked-dependency tests in `internal/services/`.

The tracer under test is constructed via `sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(...))` and injected via `svc.SetTracer(tp.Tracer("test"))`.

---

## Test Files

| File | Coverage Target |
|------|----------------|
| `internal/services/task_service_tracing_test.go` | TaskService span emission (7 methods) |
| `internal/services/feature_service_tracing_test.go` | FeatureService span emission (5 methods) |
| `internal/services/epic_service_tracing_test.go` | EpicService span emission (4 methods) |

---

## Test Cases — TaskService

### TC-TASK-01: GetTask emits span with correct name and attribute
- Arrange: mock repo returns a task; inject in-memory tracer
- Act: call `svc.GetTask(ctx, "E07-F01-001")`
- Assert: exporter has one span; `span.Name() == "TaskService.GetTask"`; attribute `task.key == "E07-F01-001"`

### TC-TASK-02: GetTask error records error on span
- Arrange: mock repo returns `repository.NotFoundError`
- Act: call `svc.GetTask(ctx, "E99-F99-999")`
- Assert: span status code == `codes.Error`; span has recorded event with error message

### TC-TASK-03: CreateTask emits span with epic and feature key attributes
- Arrange: mock repo creates task successfully
- Act: call `svc.CreateTask(ctx, CreateTaskInput{EpicKey: "E07", FeatureKey: "F01", Title: "test"})`
- Assert: span name `"TaskService.CreateTask"`; attributes include `task.epic_key` and `task.feature_key`

### TC-TASK-04: TransitionStatus emits span with target_status attribute
- Arrange: mock repo; mock workflow service
- Act: call `svc.TransitionStatus(ctx, "E07-F01-001", "completed", TransitionOptions{})`
- Assert: span name `"TaskService.TransitionStatus"`; attribute `task.target_status == "completed"`

### TC-TASK-05: ListTasks emits span
- Assert: span name `"TaskService.ListTasks"` present

### TC-TASK-06: Nil tracer falls back to global noop (no panic)
- Arrange: do not call SetTracer — leave tracer as nil
- Act: call any instrumented method
- Assert: no panic; method returns expected result

### TC-TASK-07: Context carries span to downstream calls
- Arrange: capture the context passed to `repo.GetByKey` via mock
- Act: call `svc.GetTask(ctx, key)`
- Assert: the context passed to the mock repo has a non-empty span context (i.e., span was started before repo call)

---

## Test Cases — FeatureService

### TC-FEAT-01: GetFeature emits span with feature.key
### TC-FEAT-02: GetFeature error records error on span
### TC-FEAT-03: TransitionStatus emits span with target_status
### TC-FEAT-04: CompleteFeature emits span
### TC-FEAT-05: ListFeatures emits span
### TC-FEAT-06: Nil tracer fallback — no panic

---

## Test Cases — EpicService

### TC-EPIC-01: GetEpic emits span with epic.key
### TC-EPIC-02: GetEpic error records error on span
### TC-EPIC-03: TransitionStatus emits span
### TC-EPIC-04: ListEpics emits span
### TC-EPIC-05: Nil tracer fallback — no panic

---

## Regression Tests

All existing service tests in `internal/services/` must pass without modification. No new imports or initialization required by existing tests.

Run: `make test` — zero failures expected.

---

## Helper Pattern

```go
// shared test helper (in each _tracing_test.go or a shared tracing_test_helpers_test.go)
func newTestTracerWithExporter(t *testing.T) (trace.Tracer, *tracetest.InMemoryExporter) {
    t.Helper()
    exp := tracetest.NewInMemoryExporter()
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
    )
    t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
    return tp.Tracer("test"), exp
}
```

Import path: `go.opentelemetry.io/otel/sdk/trace/tracetest` — already in go.mod via F01.
