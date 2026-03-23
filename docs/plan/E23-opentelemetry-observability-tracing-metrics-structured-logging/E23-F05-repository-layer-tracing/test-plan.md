# E23-F05: Repository Layer Tracing — Test Plan

**Feature Key:** E23-F05-repository-layer-tracing
**Date:** 2026-03-22

---

## Testing Strategy

Repository tests use a real SQLite database (per `.claude/rules/testing/architecture.md`). The global OTel TracerProvider defaults to a no-op in tests — no test setup changes are needed to accommodate spans. The test plan verifies that:

1. All existing repository tests continue to pass without modification (no-op safety).
2. Instrumented methods correctly pass the enriched context to database calls (context propagation).
3. The package-level tracer variable is present (build-time verification).

**Test Location:** `internal/repository/` — existing test files, plus a new `tracing_test.go`.

---

## Test Categories

### Category 1: No-Op Safety (Regression Tests)

**Objective:** Verify all existing repository tests pass after instrumentation is added.

**Approach:** Run `make test` targeting `internal/repository/...`. No test code changes should be required.

**Tests:**
- All existing tests in `task_repository_test.go`
- All existing tests in `feature_repository_test.go`
- All existing tests in `epic_repository_test.go`
- All existing tests in `entity_note_repository_test.go`

**Pass Criteria:** `make test ./internal/repository/...` exits with code 0, no test failures.

---

### Category 2: Context Propagation

**Objective:** Verify that instrumented methods use the context returned by `span.Start` (the enriched context with span ID) for all database calls, not the original context.

**Approach:** Use a test tracer that captures spans, inject it via `otel.SetTracerProvider` in the test, and verify that the span is started and ended correctly.

**File:** `internal/repository/tracing_test.go`

**Tests:**

#### TEST-TR-001: TaskRepository.GetByKey Span Created

```go
func TestTaskRepository_GetByKey_SpanCreated(t *testing.T) {
    // Setup: use a recording tracer provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithSyncer(exporter),
    )
    otel.SetTracerProvider(tp)
    defer otel.SetTracerProvider(otel.GetTracerProvider()) // restore noop

    db := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Create a test task first
    // ...

    _, _ = repo.GetByKey(context.Background(), testKey)

    spans := exporter.GetSpans()
    require.NotEmpty(t, spans)

    span := spans[0]
    assert.Equal(t, "TaskRepository.GetByKey", span.Name())

    attrMap := spanAttrMap(span)
    assert.Equal(t, "SELECT", attrMap["db.operation"])
    assert.Equal(t, "tasks", attrMap["db.table"])
    assert.Equal(t, "sqlite", attrMap["db.system"])
}
```

#### TEST-TR-002: Error Recorded on Span When Task Not Found

```go
func TestTaskRepository_GetByKey_ErrorRecorded(t *testing.T) {
    // Setup recording tracer
    // ...

    db := test.GetTestDB()
    repo := NewTaskRepository(db)

    _, err := repo.GetByKey(context.Background(), "NONEXISTENT-KEY")
    require.Error(t, err)

    spans := exporter.GetSpans()
    require.NotEmpty(t, spans)

    span := spans[0]
    assert.Equal(t, codes.Error, span.Status().Code)
    assert.NotEmpty(t, span.Events()) // RecordError adds an event
}
```

#### TEST-TR-003: FeatureRepository.GetByKey Span Created

Same pattern as TEST-TR-001 but for `FeatureRepository.GetByKey` with `db.table=features`.

#### TEST-TR-004: EpicRepository.GetByKey Span Created

Same pattern as TEST-TR-001 but for `EpicRepository.GetByKey` with `db.table=epics`.

#### TEST-TR-005: EntityNoteRepository.Create Span Created

Same pattern verifying `db.operation=INSERT`, `db.table=entity_notes`.

#### TEST-TR-006: TaskRepository.Create Span Created

Verifies `db.operation=INSERT`, `db.table=tasks`.

#### TEST-TR-007: TaskRepository.UpdateStatus Span Created

Verifies `db.operation=UPDATE`, `db.table=tasks`.

#### TEST-TR-008: TaskRepository.Delete Span Created

Verifies `db.operation=DELETE`, `db.table=tasks`.

---

### Category 3: Build Verification

**Objective:** Verify the package-level tracer variables compile correctly and the OTel imports do not cause import cycles.

**Approach:** `make build` succeeds. `make lint` reports no new warnings.

**Pass Criteria:** Both commands exit 0.

---

### Category 4: No-Op Overhead Benchmark

**Objective:** Verify no-op tracer adds less than 1% overhead (REQ-NF-001).

**File:** `internal/repository/task_repository_bench_test.go` (new or existing)

```go
func BenchmarkTaskRepository_GetByKey_NoOp(b *testing.B) {
    // Ensure noop provider (default)
    db := test.GetTestDB()
    repo := NewTaskRepository(db)

    // Pre-create a task
    // ...

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = repo.GetByKey(context.Background(), testKey)
    }
}
```

**Pass Criteria:** Benchmark delta vs. pre-F05 baseline is within measurement noise (< 1% on repeated runs).

---

## Test Helpers

### Span Recording Setup

```go
// testSpanExporter captures spans for assertion in tests
type testSpanExporter struct {
    mu    sync.Mutex
    spans []sdktrace.ReadOnlySpan
}

func (e *testSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.spans = append(e.spans, spans...)
    return nil
}

func (e *testSpanExporter) Shutdown(_ context.Context) error { return nil }

func (e *testSpanExporter) GetSpans() []sdktrace.ReadOnlySpan {
    e.mu.Lock()
    defer e.mu.Unlock()
    result := make([]sdktrace.ReadOnlySpan, len(e.spans))
    copy(result, e.spans)
    return result
}

func (e *testSpanExporter) Reset() {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.spans = nil
}

// spanAttrMap extracts attributes into a string map for easy assertion
func spanAttrMap(span sdktrace.ReadOnlySpan) map[string]string {
    result := make(map[string]string)
    for _, attr := range span.Attributes() {
        result[string(attr.Key)] = attr.Value.AsString()
    }
    return result
}
```

---

## Test Execution

```bash
# Run all repository tests (regression safety)
go test -v ./internal/repository/...

# Run only tracing tests
go test -v ./internal/repository/ -run TestTaskRepository.*Span
go test -v ./internal/repository/ -run TestFeatureRepository.*Span
go test -v ./internal/repository/ -run TestEpicRepository.*Span

# Run benchmarks
go test -bench=BenchmarkTaskRepository_GetByKey -benchmem ./internal/repository/

# Full quality gate
make fmt && make lint && make test
```

---

## Pass/Fail Criteria Summary

| Category | Tests | Pass Criteria |
|----------|-------|---------------|
| No-Op Safety | All existing repo tests | `make test ./internal/repository/...` exits 0 |
| Span Creation | TEST-TR-001 through TEST-TR-008 | All assertions pass |
| Build Verification | `make build`, `make lint` | Both exit 0 |
| Overhead Benchmark | BenchmarkTaskRepository_GetByKey_NoOp | < 1% regression vs baseline |
