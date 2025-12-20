# Task: E05-F01-T05 - Testing, Optimization & Performance Validation

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T05

## Description

Implement comprehensive testing, performance optimization, and validation to ensure the status dashboard meets all quality and performance requirements. This task validates that the implementation is correct, performs well (<500ms for 100 epics), and provides good test coverage.

Tasks include:
- Unit tests for service layer (models, queries, aggregation logic)
- Integration tests for CLI command
- Performance benchmarks for various project sizes
- Query optimization and index verification
- Memory profiling and optimization
- Code coverage measurement

**Why This Matters**: A well-tested, optimized feature is maintainable and reliable. Performance validation ensures the feature works at scale. Test coverage prevents regressions.

## What You'll Build

Complete test suite spanning multiple files:

```
internal/status/
├── status_test.go (Unit & Benchmark tests)
│   ├── TestStatusService_GetDashboard_EmptyProject
│   ├── TestStatusService_GetDashboard_FullProject
│   ├── TestEpicHealthCalculation
│   ├── BenchmarkStatusService_GetDashboard_LargeProject
│   └── 10+ additional tests
│
internal/cli/commands/
└── status_test.go (Integration tests)
    ├── TestStatusCommand_JSONOutput
    ├── TestStatusCommand_FilteredByEpic
    ├── TestStatusCommand_InvalidEpic
    └── TestStatusCommand_NoColor
```

Plus test helpers for creating test databases and validating output.

## Success Criteria

- [x] Unit tests cover service layer: data models, queries, calculations
- [x] Integration tests verify CLI command end-to-end
- [x] Benchmark: Empty project <50ms
- [x] Benchmark: Small project (127 tasks) <50ms
- [x] Benchmark: Large project (2000 tasks) <500ms
- [x] Benchmark: Filtered by epic <50ms
- [x] Performance: <500ms for 100 epics with 2000 tasks
- [x] Memory: <50MB peak allocation for large project
- [x] No N+1 query problems verified
- [x] Code coverage >80% for status package
- [x] All tests pass: `go test ./internal/status ./internal/cli/commands -v`
- [x] No race conditions: `go test -race ./internal/status`
- [x] Queries verified with `EXPLAIN QUERY PLAN`

## Implementation Notes

### Test Database Setup Helpers

Create reusable test database builders:

```go
// internal/status/status_test.go

func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test DB: %v", err)
    }

    // Initialize schema (simplified version)
    // In real implementation, use existing schema initialization
    initTestSchema(db)

    return db
}

func createTestEpics(t *testing.T, db *sql.DB, count int) {
    for i := 1; i <= count; i++ {
        _, err := db.Exec(
            "INSERT INTO epics (key, title, status, priority) VALUES (?, ?, ?, ?)",
            fmt.Sprintf("E%02d", i), fmt.Sprintf("Epic %d", i), "active", "high",
        )
        if err != nil {
            t.Fatalf("Failed to create test epic: %v", err)
        }
    }
}

func createTestTasks(t *testing.T, db *sql.DB, featureID int64, count int, statuses []string) {
    for i := 0; i < count; i++ {
        status := statuses[i%len(statuses)]
        _, err := db.Exec(
            "INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, ?, ?, ?, ?)",
            featureID, fmt.Sprintf("T-%03d", i), fmt.Sprintf("Task %d", i), status, 5,
        )
        if err != nil {
            t.Fatalf("Failed to create test task: %v", err)
        }
    }
}
```

### Unit Tests - Service Layer

```go
func TestStatusService_GetDashboard_EmptyProject(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    defer db.Close()

    service := NewStatusService(db, repos...)
    request := &StatusRequest{}

    // Act
    dashboard, err := service.GetDashboard(context.Background(), request)

    // Assert
    if err != nil {
        t.Fatalf("GetDashboard failed: %v", err)
    }
    if dashboard.Summary.Epics.Total != 0 {
        t.Errorf("Expected 0 epics, got %d", dashboard.Summary.Epics.Total)
    }
    if len(dashboard.Epics) != 0 {
        t.Errorf("Expected empty epics list, got %d", len(dashboard.Epics))
    }
}

func TestStatusService_GetDashboard_WithData(t *testing.T) {
    // Arrange: Create 3 epics, 12 features, 127 tasks
    db := setupTestDB(t)
    defer db.Close()

    createTestEpics(t, db, 3)
    createTestFeatures(t, db, 12, 3)
    createTestTasks(t, db, 127, []string{"completed", "in_progress", "todo", "blocked"})

    service := NewStatusService(db, repos...)
    request := &StatusRequest{}

    // Act
    dashboard, err := service.GetDashboard(context.Background(), request)

    // Assert
    if err != nil {
        t.Fatalf("GetDashboard failed: %v", err)
    }

    if dashboard.Summary.Epics.Total != 3 {
        t.Errorf("Expected 3 epics, got %d", dashboard.Summary.Epics.Total)
    }
    if dashboard.Summary.Tasks.Total != 127 {
        t.Errorf("Expected 127 tasks, got %d", dashboard.Summary.Tasks.Total)
    }
    if len(dashboard.Epics) != 3 {
        t.Errorf("Expected 3 epics in breakdown, got %d", len(dashboard.Epics))
    }

    // Verify progress calculation
    if dashboard.Summary.OverallProgress < 0 || dashboard.Summary.OverallProgress > 100 {
        t.Errorf("Invalid progress: %f", dashboard.Summary.OverallProgress)
    }
}

func TestStatusService_EpicHealthCalculation(t *testing.T) {
    tests := []struct {
        name    string
        progress float64
        blocked int
        expected string
    }{
        {"Healthy 100%", 100, 0, "healthy"},
        {"Healthy 75%", 75, 0, "healthy"},
        {"Healthy 75% 1 blocked", 75, 1, "warning"},
        {"Warning 74%", 74, 0, "warning"},
        {"Warning 25%", 25, 0, "warning"},
        {"Critical 24%", 24, 0, "critical"},
        {"Critical with blockers", 50, 4, "critical"},
        {"Warning 3 blockers", 60, 3, "warning"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := &statusService{}
            result := service.determineEpicHealth(tt.progress, tt.blocked)
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}

func TestStatusService_FilterByEpic(t *testing.T) {
    // Arrange: Create multiple epics with different task counts
    db := setupTestDB(t)
    defer db.Close()

    createTestEpics(t, db, 3)
    // ... create features and tasks

    service := NewStatusService(db, repos...)
    request := &StatusRequest{EpicKey: "E01"}

    // Act
    dashboard, err := service.GetDashboard(context.Background(), request)

    // Assert
    if err != nil {
        t.Fatalf("GetDashboard with filter failed: %v", err)
    }

    if len(dashboard.Epics) != 1 {
        t.Errorf("Expected 1 epic, got %d", len(dashboard.Epics))
    }

    if dashboard.Epics[0].Key != "E01" {
        t.Errorf("Expected E01, got %s", dashboard.Epics[0].Key)
    }
}

func TestStatusService_RelativeTimeFormatting(t *testing.T) {
    service := &statusService{}

    tests := []struct {
        name     string
        duration time.Duration
        pattern  string
    }{
        {"Seconds ago", 30 * time.Second, "seconds ago"},
        {"Minutes ago", 5 * time.Minute, "minutes ago"},
        {"Hours ago", 2 * time.Hour, "hours ago"},
        {"Days ago", 24 * time.Hour, "day ago"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            past := time.Now().Add(-tt.duration)
            result := service.calculateRelativeTime(past)
            if !strings.Contains(result, tt.pattern) {
                t.Errorf("Expected pattern %q in %q", tt.pattern, result)
            }
        })
    }
}
```

### Integration Tests - CLI Command

```go
func TestStatusCommand_JSONOutput(t *testing.T) {
    // Arrange: Create test database with known data
    db := setupTestDB(t)
    defer db.Close()
    createTestData(t, db)

    // Act: Execute status command with --json flag
    cmd := &cobra.Command{}
    // ... configure command

    // Assert: Verify JSON output is valid
    output := captureOutput(t, func() {
        runStatus(cmd, []string{})
    })

    var dashboard StatusDashboard
    if err := json.Unmarshal([]byte(output), &dashboard); err != nil {
        t.Fatalf("Output is not valid JSON: %v", err)
    }

    if dashboard.Summary == nil {
        t.Error("Summary is nil")
    }
}

func TestStatusCommand_FilterByEpic(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    defer db.Close()
    createTestData(t, db)

    // Act: Run with --epic=E01
    // ...

    // Assert: Verify only E01 data in output
}

func TestStatusCommand_InvalidEpic(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    defer db.Close()

    // Act: Run with --epic=E999
    // ...

    // Assert: Expect error message about epic not found
    if !strings.Contains(output, "Epic not found") {
        t.Error("Expected 'Epic not found' error message")
    }
}

func TestStatusCommand_NoColor(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    createTestData(t, db)

    // Act: Run with --no-color
    output := runStatusCommand(t, db, "--no-color")

    // Assert: No ANSI color codes in output
    if strings.Contains(output, "\033[") {
        t.Error("Output contains ANSI color codes despite --no-color")
    }
}
```

### Benchmark Tests

```go
func BenchmarkStatusService_GetDashboard_EmptyProject(b *testing.B) {
    db := setupTestDB(&testing.T{})
    defer db.Close()

    service := NewStatusService(db, repos...)
    ctx := context.Background()
    request := &StatusRequest{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetDashboard(ctx, request)
        if err != nil {
            b.Fatalf("GetDashboard failed: %v", err)
        }
    }

    // Target: <50ms
}

func BenchmarkStatusService_GetDashboard_LargeProject(b *testing.B) {
    // Setup: 100 epics, 500 features, 2000 tasks
    db := setupTestDB(&testing.T{})
    defer db.Close()
    createLargeTestDB(db, 100, 500, 2000)

    service := NewStatusService(db, repos...)
    ctx := context.Background()
    request := &StatusRequest{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetDashboard(ctx, request)
        if err != nil {
            b.Fatalf("GetDashboard failed: %v", err)
        }
    }

    b.StopTimer()

    // Verify performance target
    elapsed := b.Elapsed().Seconds() / float64(b.N)
    if elapsed > 0.5 {
        b.Fatalf("Performance goal not met: %.0f ms (target: <500ms)", elapsed*1000)
    }
}

func BenchmarkStatusService_FilteredByEpic(b *testing.B) {
    // Setup: Large project with single epic filter
    db := setupTestDB(&testing.T{})
    defer db.Close()
    createLargeTestDB(db, 100, 500, 2000)

    service := NewStatusService(db, repos...)
    ctx := context.Background()
    request := &StatusRequest{EpicKey: "E01"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetDashboard(ctx, request)
        if err != nil {
            b.Fatalf("GetDashboard failed: %v", err)
        }
    }

    // Target: <50ms (filtered queries should be much faster)
}
```

### Performance Test Output Interpretation

Running benchmark:
```bash
go test -bench=GetDashboard -benchmem ./internal/status
```

Output example:
```
BenchmarkStatusService_GetDashboard_EmptyProject-8        50000    24187 ns/op    10240 B/op    12 allocs/op
BenchmarkStatusService_GetDashboard_SmallProject-8        20000    48932 ns/op    45820 B/op   145 allocs/op
BenchmarkStatusService_GetDashboard_LargeProject-8          100   298104 ns/op   856234 B/op  1245 allocs/op
```

**Reading**:
- `298104 ns/op` = 298.1 µs per operation (< 500ms ✓)
- `856234 B/op` = 856 KB per operation (< 50MB for project ✓)
- `1245 allocs/op` = allocations (reasonable for 2000 items)

### Code Coverage

```bash
# Generate coverage report
go test ./internal/status -coverprofile=coverage.out

# View HTML report
go tool cover -html=coverage.out

# Check coverage percentage
go test ./internal/status -cover
```

**Target**: >80% coverage for status package

### Race Condition Detection

```bash
go test -race ./internal/status ./internal/cli/commands
```

Should complete without reporting any data races.

### Query Optimization Verification

```bash
sqlite3 shark-tasks.db
sqlite> EXPLAIN QUERY PLAN SELECT e.id, e.key, COUNT(DISTINCT t.id) FROM epics e LEFT JOIN features f ON e.id = f.epic_id LEFT JOIN tasks t ON f.id = t.feature_id WHERE e.archived = false GROUP BY e.id;
```

Expected output shows index usage (SEARCH), not full scans (SCAN):
```
0|0|0|SEARCH TABLE epics AS e
1|0|1|SEARCH TABLE features AS f USING idx_features_epic_id (epic_id=?)
2|0|2|SEARCH TABLE tasks AS t USING idx_tasks_feature_id (feature_id=?)
```

## Dependencies

- Go testing: testing, context
- Benchmarking tools: built-in
- Test helpers: internal/test package
- Database: SQLite with test schema

## Related Tasks

- **E05-F01-T01 through T04**: All testing validates these implementations

## Acceptance Criteria

**Unit Testing**:
- [ ] >15 unit tests covering service layer
- [ ] Tests for each query method
- [ ] Tests for health calculation
- [ ] Tests for filtering
- [ ] Tests for relative time formatting
- [ ] All unit tests pass

**Integration Testing**:
- [ ] >8 integration tests for CLI command
- [ ] Tests for JSON output validity
- [ ] Tests for filtering
- [ ] Tests for error cases
- [ ] Tests for --no-color flag
- [ ] All integration tests pass

**Performance Benchmarking**:
- [ ] Empty project: <50ms
- [ ] Small project: <50ms
- [ ] Large project (2000 tasks): <500ms
- [ ] Filtered queries: <50ms
- [ ] Memory usage: <50MB
- [ ] Linear scaling with data size

**Code Quality**:
- [ ] Code coverage >80% for status package
- [ ] No race conditions detected
- [ ] No N+1 query problems
- [ ] All linting issues resolved
- [ ] Proper error handling in tests

**Documentation**:
- [ ] Benchmark results documented
- [ ] Performance trade-offs documented
- [ ] Test coverage report generated
- [ ] Query optimization decisions documented

## Verification Steps

```bash
# Run all tests
go test ./internal/status ./internal/cli/commands -v

# Run with coverage
go test ./internal/status -cover -v

# View coverage in browser
go test ./internal/status -coverprofile=coverage.out && go tool cover -html=coverage.out

# Run benchmarks
go test -bench=GetDashboard -benchmem ./internal/status

# Detect race conditions
go test -race ./internal/status ./internal/cli/commands

# Profile performance
go test -cpuprofile=cpu.prof -bench=LargeProject ./internal/status
go tool pprof cpu.prof

# Check query plans
sqlite3 shark-tasks.db "EXPLAIN QUERY PLAN SELECT ..."
```

## Implementation Checklist

See Phase 5 in implementation-checklist.md:
- [ ] Task 5.1: Unit Tests
- [ ] Task 5.2: Integration Tests
- [ ] Task 5.3: Performance Testing
- [ ] Task 5.4: Query Performance Optimization
- [ ] Task 5.5: Memory Profiling
- [ ] Task 5.6: Code Coverage
- [ ] Task 5.7: Documentation
