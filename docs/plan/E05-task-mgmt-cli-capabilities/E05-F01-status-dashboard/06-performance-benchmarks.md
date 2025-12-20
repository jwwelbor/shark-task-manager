# Performance Benchmarking Guide: Status Dashboard (E05-F01)

**Document Version**: 1.0
**Status**: Performance Testing & Optimization Reference
**Last Updated**: 2025-12-19

This guide provides comprehensive performance testing strategy and benchmarking approach for the Status Dashboard feature.

---

## Table of Contents

1. [Performance Goals](#performance-goals)
2. [Benchmark Scenarios](#benchmark-scenarios)
3. [Testing Infrastructure](#testing-infrastructure)
4. [Profiling Techniques](#profiling-techniques)
5. [Optimization Strategies](#optimization-strategies)
6. [Benchmarking Checklist](#benchmarking-checklist)

---

## Performance Goals

### 1.1 Target Metrics

| Component | Metric | Target | Acceptable | Threshold |
|-----------|--------|--------|------------|-----------|
| Database Queries | Execution time | <200ms | <250ms | >300ms ❌ |
| Output Formatting | Rendering time | <100ms | <150ms | >200ms ❌ |
| Total Dashboard | End-to-end | <500ms | <700ms | >1000ms ❌ |
| Memory Usage | Peak allocation | <50MB | <75MB | >100MB ❌ |
| JSON Marshaling | Serialization | <50ms | <100ms | >150ms ❌ |

### 1.2 Scale-Up Projections

For scaling beyond target benchmark size (100 epics):

```
Time grows approximately linearly with data size
(assuming index efficiency maintained)

10 epics, 50 features, 100 tasks:   ~50ms (baseline)
50 epics, 250 features, 500 tasks:  ~150ms
100 epics, 500 features, 1000 tasks: ~300ms (target)
200 epics, 1000 features, 2000 tasks: ~600ms (acceptable)
```

### 1.3 Success Criteria

Dashboard is considered **successful** if:

- ✓ Achieves <500ms end-to-end for 100 epics
- ✓ Query execution <200ms (no N+1 problems)
- ✓ Memory usage <50MB
- ✓ No regressions on repeated calls
- ✓ Scales linearly with data size

---

## Benchmark Scenarios

### 2.1 Scenario 1: Empty Project

**Setup**:
- Database with zero epics, features, tasks
- No historical data

**Expected Results**:
- Dashboard renders in <50ms
- All sections show "no data" messages
- Memory: <5MB

**Test Code**:
```go
func TestStatusDashboard_EmptyProject(b *testing.B) {
    db := setupEmptyTestDB()
    defer db.Close()

    service := createStatusService(db)
    ctx := context.Background()
    request := &StatusRequest{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetDashboard(ctx, request)
        if err != nil {
            b.Fatalf("GetDashboard failed: %v", err)
        }
    }
}
```

**Run**:
```bash
go test -bench=EmptyProject -benchmem ./internal/status
```

### 2.2 Scenario 2: Small Project

**Setup**:
- 3 epics
- 12 features (4 per epic)
- 127 tasks distributed across features
- Mix of task statuses: todo, in_progress, completed, blocked
- 50 tasks completed in last 24 hours

**Rationale**:
- Representative of typical small project
- Easy to verify correctness
- Reasonable baseline for performance

**Expected Results**:
- Dashboard renders in <50ms
- All sections populated with data
- Correct counts and grouping
- Memory: <10MB

**Test Code**:
```go
func TestStatusDashboard_SmallProject(b *testing.B) {
    db := setupSmallTestDB()  // 3 epics, 127 tasks
    defer db.Close()

    service := createStatusService(db)
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

    // Verify correctness
    if b.Elapsed().Seconds()/float64(b.N) > 0.05 {
        b.Fatalf("Too slow: %v per iteration", b.Elapsed()/time.Duration(b.N))
    }
}
```

**Run**:
```bash
go test -bench=SmallProject -benchmem ./internal/status
# Expected: <50ms per iteration
```

### 2.3 Scenario 3: Large Project

**Setup**:
- 100 epics
- 500 features (5 per epic)
- 2000 tasks (4 per feature)
- Realistic distribution of statuses:
  - 40% completed
  - 20% in_progress
  - 30% todo
  - 8% blocked
  - 2% ready_for_review
- 500 tasks completed in last 24 hours

**Rationale**:
- Tests scaling with large data
- Verifies no N+1 query problems
- Validates <500ms goal

**Expected Results**:
- Dashboard renders in <500ms
- Correct aggregation of 2000+ items
- All sections properly populated
- Memory: <50MB
- No visible slowdown vs. small project (linear scaling)

**Test Code**:
```go
func BenchmarkStatusDashboard_LargeProject(b *testing.B) {
    db := setupLargeTestDB()  // 100 epics, 2000 tasks
    defer db.Close()

    service := createStatusService(db)
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

    // Verify performance goal
    elapsed := b.Elapsed().Seconds() / float64(b.N)
    if elapsed > 0.5 {
        b.Fatalf("Performance goal not met: %.1f ms (target: <500ms)", elapsed*1000)
    }
}
```

**Run**:
```bash
go test -bench=LargeProject -benchmem ./internal/status
# Expected: <500ms per iteration
```

### 2.4 Scenario 4: Filtered by Epic

**Setup**:
- Same as large project (100 epics, 2000 tasks)
- Query filtered to single epic (E01)

**Expected Results**:
- Query filters to 50 tasks for single epic
- Time <50ms (much faster than full dashboard)
- Memory: <5MB

**Test Code**:
```go
func BenchmarkStatusDashboard_FilteredByEpic(b *testing.B) {
    db := setupLargeTestDB()
    defer db.Close()

    service := createStatusService(db)
    ctx := context.Background()
    request := &StatusRequest{EpicKey: "E01"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetDashboard(ctx, request)
        if err != nil {
            b.Fatalf("GetDashboard failed: %v", err)
        }
    }

    b.StopTimer()
    elapsed := b.Elapsed().Seconds() / float64(b.N)
    if elapsed > 0.05 {
        b.Fatalf("Filtered query too slow: %.1f ms", elapsed*1000)
    }
}
```

**Run**:
```bash
go test -bench=FilteredByEpic -benchmem ./internal/status
# Expected: <50ms per iteration
```

### 2.5 Scenario 5: Query Decomposition

**Purpose**: Identify which queries are fast vs. slow

**Test Code**:
```go
func BenchmarkStatusService_ProjectSummary(b *testing.B) {
    // Time just the project summary queries
    db := setupLargeTestDB()
    service := createStatusService(db)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.getProjectSummary(ctx, "")
    }
    // Target: <50ms
}

func BenchmarkStatusService_EpicsTable(b *testing.B) {
    // Time just the epic breakdown query
    db := setupLargeTestDB()
    service := createStatusService(db)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.getEpics(ctx, "")
    }
    // Target: <50ms
}

func BenchmarkStatusService_ActiveTasks(b *testing.B) {
    // Time just the active tasks query
    db := setupLargeTestDB()
    service := createStatusService(db)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.getActiveTasks(ctx, "")
    }
    // Target: <50ms
}

func BenchmarkStatusService_BlockedTasks(b *testing.B) {
    // Time just the blocked tasks query
    db := setupLargeTestDB()
    service := createStatusService(db)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.getBlockedTasks(ctx, "")
    }
    // Target: <30ms
}

func BenchmarkStatusService_RecentCompletions(b *testing.B) {
    // Time just the recent completions query
    db := setupLargeTestDB()
    service := createStatusService(db)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.getRecentCompletions(ctx, "", "24h")
    }
    // Target: <30ms
}
```

**Run all**:
```bash
go test -bench=StatusService -benchmem ./internal/status
# Shows which queries are bottlenecks
```

### 2.6 Scenario 6: Output Formatting

**Purpose**: Measure JSON/table rendering performance separately

**Test Code**:
```go
func BenchmarkJSONFormatting_LargeProject(b *testing.B) {
    // Time JSON marshaling only
    dashboard := createLargeDashboard()  // Pre-create dashboard

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        json.MarshalIndent(dashboard, "", "  ")
    }
    // Target: <50ms
}

func BenchmarkTableFormatting_LargeProject(b *testing.B) {
    // Time table rendering only
    dashboard := createLargeDashboard()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        outputRichTable(dashboard)
    }
    // Target: <100ms
}
```

**Run**:
```bash
go test -bench=Formatting -benchmem ./internal/status
```

---

## Testing Infrastructure

### 3.1 Test Database Setup

**Create test databases with known data**:

```go
func setupEmptyTestDB() *sql.DB {
    // Create in-memory or temp file database
    db, _ := sql.Open("sqlite3", ":memory:")
    // Initialize schema
    return db
}

func setupSmallTestDB() *sql.DB {
    db, _ := sql.Open("sqlite3", ":memory:")
    // Initialize schema
    // Create 3 epics
    for i := 1; i <= 3; i++ {
        createTestEpic(db, fmt.Sprintf("E%02d", i), "Epic Title")
    }
    // Create 12 features
    for i := 1; i <= 12; i++ {
        epicID := (i % 3) + 1
        createTestFeature(db, epicID, "Feature Title")
    }
    // Create 127 tasks
    for i := 1; i <= 127; i++ {
        featureID := (i % 12) + 1
        status := selectStatus(i)  // Distribute statuses
        createTestTask(db, featureID, status, "Task Title")
    }
    return db
}

func setupLargeTestDB() *sql.DB {
    db, _ := sql.Open("sqlite3", ":memory:")
    // Initialize schema
    // Create 100 epics
    for i := 1; i <= 100; i++ {
        createTestEpic(db, fmt.Sprintf("E%02d", i), "Epic Title")
    }
    // Create 500 features
    for i := 1; i <= 500; i++ {
        epicID := (i % 100) + 1
        createTestFeature(db, epicID, "Feature Title")
    }
    // Create 2000 tasks
    for i := 1; i <= 2000; i++ {
        featureID := (i % 500) + 1
        status := selectStatus(i)
        createTestTask(db, featureID, status, "Task Title")
    }
    return db
}

func selectStatus(index int) string {
    // Distribute: 40% completed, 20% in_progress, 30% todo, 8% blocked, 2% ready_for_review
    percent := index % 100
    switch {
    case percent < 40:
        return "completed"
    case percent < 60:
        return "in_progress"
    case percent < 90:
        return "todo"
    case percent < 98:
        return "blocked"
    default:
        return "ready_for_review"
    }
}
```

### 3.2 Running Benchmarks

**Basic run**:
```bash
go test -bench=. ./internal/status
```

**With memory allocation reporting**:
```bash
go test -bench=. -benchmem ./internal/status
# Shows: ns/op, B/op (bytes per op), allocs/op (allocations per op)
```

**Run specific benchmark**:
```bash
go test -bench=BenchmarkStatusService_GetDashboard_LargeProject ./internal/status
```

**Run for specific duration**:
```bash
go test -bench=. -benchtime=10s ./internal/status
# Runs each benchmark for 10 seconds instead of default
```

**Verbose output**:
```bash
go test -bench=. -v ./internal/status
```

### 3.3 Benchmark Output Interpretation

Example output:
```
BenchmarkStatusService_GetDashboard_EmptyProject-8        50000    24187 ns/op    10240 B/op    12 allocs/op
BenchmarkStatusService_GetDashboard_SmallProject-8        20000    48932 ns/op    45820 B/op    145 allocs/op
BenchmarkStatusService_GetDashboard_LargeProject-8          100    298104 ns/op   856234 B/op    1245 allocs/op
```

**Reading the output**:
- First number: iterations (50000, 20000, 100)
  - Higher is better (less variance)
  - Slow benchmarks run fewer iterations

- `ns/op`: Nanoseconds per operation
  - 24187 ns = 24.2 µs
  - 298104 ns = 298.1 µs
  - Check against target (e.g., 500ms = 500,000,000 ns)

- `B/op`: Bytes allocated per operation
  - 856234 B = 856 KB
  - Check against memory target (<50MB total)

- `allocs/op`: Number of heap allocations
  - 1245 allocations per operation
  - Look for unnecessary allocations (use object pooling if high)

---

## Profiling Techniques

### 4.1 CPU Profiling

**Capture CPU profile**:

```bash
go test -cpuprofile=cpu.prof -bench=LargeProject ./internal/status
```

**Analyze with pprof**:

```bash
go tool pprof cpu.prof
```

**Interactive commands in pprof**:
```
(pprof) top10          # Show top 10 functions by CPU time
(pprof) list QueryName # Show source code with line-by-line time
(pprof) web            # Generate graph (requires graphviz)
(pprof) quit
```

**Export to HTML**:
```bash
go tool pprof -http=:8080 cpu.prof
# Opens browser with interactive visualization
```

### 4.2 Memory Profiling

**Capture memory profile**:

```bash
go test -memprofile=mem.prof -bench=LargeProject ./internal/status
```

**Analyze peak memory**:

```bash
go tool pprof -alloc_space mem.prof  # Total allocations
go tool pprof -alloc_objects mem.prof # Number of allocations
go tool pprof -inuse_space mem.prof   # Peak memory in use
go tool pprof -inuse_objects mem.prof # Current allocations
```

### 4.3 Trace Analysis

**Capture execution trace**:

```bash
go test -trace=trace.out ./internal/status
```

**Analyze with trace viewer**:

```bash
go tool trace trace.out
# Opens browser with timeline view
```

### 4.4 Query Plan Analysis

**View SQLite query plan**:

```bash
sqlite3 shark-tasks.db
sqlite> EXPLAIN QUERY PLAN SELECT ... FROM ... WHERE ...;
```

**Example analysis**:
```
0|0|0|SCAN TABLE epics
1|0|0|SEARCH TABLE features USING idx_features_epic_id (epic_id=?)
2|0|0|SEARCH TABLE tasks USING idx_tasks_feature_id (feature_id=?)
```

**Good plan indicators**:
- Uses SEARCH with indexes (not SCAN)
- Accesses indexes in efficient order
- Low cost numbers

**Bad plan indicators**:
- SCAN TABLE (full table scan, slow)
- Multiple SCAN operations
- High cost numbers

---

## Optimization Strategies

### 5.1 If Query Performance is Slow (>200ms)

**Diagnosis**:
1. Run `EXPLAIN QUERY PLAN` on slow queries
2. Look for SCAN TABLE instead of SEARCH
3. Check if indexes are missing or not used

**Solutions**:

**Solution 1: Add Missing Indexes**

```sql
-- Verify these indexes exist
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_completed_at ON tasks(completed_at);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_type ON tasks(agent_type);
CREATE INDEX IF NOT EXISTS idx_features_epic_id ON features(epic_id);
CREATE INDEX IF NOT EXISTS idx_tasks_feature_id ON tasks(feature_id);

-- Composite index for common patterns
CREATE INDEX IF NOT EXISTS idx_tasks_status_completed_at
    ON tasks(status, completed_at DESC);
```

**Solution 2: Optimize Query Structure**

```go
// BEFORE: Queries status counts separately
var completed, total int
// Query 1: COUNT WHERE status='completed'
// Query 2: COUNT(*)
// Result: 2 queries, N+1 risk

// AFTER: Single query with aggregation
var completed, total int
// Query: SELECT COUNT(*), SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END)
// Result: 1 query, fast
```

**Solution 3: Use Prepared Statements**

If not already used, prepare statements to avoid parsing overhead:

```go
stmt, err := db.Prepare("SELECT ... WHERE status = ?")
defer stmt.Close()
// Reuse stmt for multiple executions
```

### 5.2 If Memory Usage is High (>50MB)

**Diagnosis**:
1. Run memory profile: `go test -memprofile=mem.prof`
2. Identify largest allocations: `go tool pprof -alloc_space`
3. Look for unnecessary slice growth

**Solutions**:

**Solution 1: Preallocate Slices**

```go
// BEFORE: Slice grows during append
tasks := []*TaskInfo{}
for rows.Next() {
    tasks = append(tasks, &task)  // May reallocate
}

// AFTER: Preallocate with known size
tasks := make([]*TaskInfo, 0, expectedCount)
for rows.Next() {
    tasks = append(tasks, &task)  // No reallocation
}
```

**Solution 2: Use Object Pooling**

For frequently created objects, reuse allocated memory:

```go
var taskInfoPool = sync.Pool{
    New: func() interface{} {
        return &TaskInfo{}
    },
}

// Get from pool
task := taskInfoPool.Get().(*TaskInfo)

// Use task...

// Return to pool
taskInfoPool.Put(task)
```

**Solution 3: Limit Result Sets**

Cap the number of results returned:

```go
// Recent completions: limit to 100
LIMIT 100

// Active tasks: limit to 1000
LIMIT 1000
```

### 5.3 If JSON Formatting is Slow (>100ms)

**Diagnosis**:
1. Profile JSON marshaling separately
2. Check for large struct hierarchies

**Solutions**:

**Solution 1: Stream JSON**

For very large dashboards, stream JSON incrementally:

```go
// Instead of Marshal (buffered)
encoder := json.NewEncoder(os.Stdout)
encoder.Encode(dashboard)  // Streams, less memory
```

**Solution 2: Reduce Data in Dashboard**

Don't include fields that aren't displayed:

```go
// Remove unnecessary fields from CompletionInfo
// Only include: key, title, completedAgo (omit other timestamps)
```

**Solution 3: Compress Output**

For very large JSON, consider gzip:

```bash
shark status --json | gzip > status.json.gz
```

### 5.4 If Table Rendering is Slow (>150ms)

**Diagnosis**:
1. Profile output formatting separately
2. Check pterm library usage

**Solutions**:

**Solution 1: Reduce Visual Complexity**

- Limit sections to top N items (100 active tasks → 50)
- Use simpler formatting without extra colors
- Cache terminal size detection

**Solution 2: Buffer Output**

Write to buffer instead of directly to stdout:

```go
var buf bytes.Buffer
// Format all output to buf
// Single write to stdout at end
fmt.Fprint(os.Stdout, buf.String())
```

---

## Benchmarking Checklist

### Pre-Benchmarking

- [ ] Database is properly indexed
- [ ] Queries use parameterized statements
- [ ] No debug logging is enabled (slow)
- [ ] Running on consistent hardware
- [ ] Close other applications (minimize system load)

### During Benchmarking

- [ ] Run benchmarks multiple times
- [ ] Note system load during runs
- [ ] Record CPU temperature (if overheating, results invalid)
- [ ] Use `-benchmem` to capture allocations

### Post-Benchmarking

- [ ] Compare against baseline (initial implementation)
- [ ] Verify all targets met
- [ ] If targets not met, run profiling
- [ ] Document any trade-offs made
- [ ] Commit benchmark code and results

### Benchmark Execution Steps

1. **Baseline Run** (before any optimization)
   ```bash
   go test -bench=LargeProject -benchmem ./internal/status > baseline.txt
   ```

2. **Apply Optimization**
   - Implement change (add index, optimize query, etc.)

3. **Measure Improvement**
   ```bash
   go test -bench=LargeProject -benchmem ./internal/status > after.txt
   ```

4. **Compare**
   ```bash
   benchstat baseline.txt after.txt
   ```
   (requires: `go install golang.org/x/perf/cmd/benchstat@latest`)

### Regression Testing

After meeting performance targets:

```bash
# Create regression test that fails if performance degrades
func TestStatusDashboard_PerformanceRegression(t *testing.T) {
    db := setupLargeTestDB()
    defer db.Close()

    service := createStatusService(db)
    ctx := context.Background()

    start := time.Now()
    _, err := service.GetDashboard(ctx, &StatusRequest{})
    elapsed := time.Since(start)

    if err != nil {
        t.Fatalf("GetDashboard failed: %v", err)
    }

    if elapsed > 500*time.Millisecond {
        t.Errorf("Performance regression: took %v, max allowed 500ms", elapsed)
    }

    // Track improvement over time
    t.Logf("Dashboard rendered in %v", elapsed)
}
```

---

## Benchmark Timeline

### Week 1: Baseline & Simple Optimization
- [ ] Run baseline benchmarks
- [ ] Add missing indexes
- [ ] Document baseline results
- [ ] Expected improvement: 20-30%

### Week 2: Query Optimization
- [ ] Analyze slow queries with EXPLAIN PLAN
- [ ] Refactor N+1 queries into JOINs
- [ ] Expected improvement: 30-50%

### Week 3: Fine Tuning
- [ ] Memory profiling
- [ ] Reduce allocations if needed
- [ ] Cache hot paths if applicable
- [ ] Expected improvement: 10-20%

### Week 4: Regression Testing
- [ ] Lock down final performance targets
- [ ] Add regression tests
- [ ] Document final results
- [ ] All targets should be met

---

## Performance Target Summary

| Component | Current | Target | Status |
|-----------|---------|--------|--------|
| Empty project | TBD | <50ms | TBD |
| Small project (127 tasks) | TBD | <50ms | TBD |
| Large project (2000 tasks) | TBD | <500ms | TBD |
| Filtered query (epic) | TBD | <50ms | TBD |
| Memory usage | TBD | <50MB | TBD |
| JSON serialization | TBD | <100ms | TBD |

**After first benchmark run, update this table with actual baseline numbers.**

---

## References

### Go Benchmarking Documentation
- https://pkg.go.dev/testing#hdr-Benchmarks
- https://golang.org/doc/diagnostics

### Profiling Tools
- https://pkg.go.dev/runtime/pprof
- https://golang.org/blog/pprof
- https://github.com/google/pprof

### SQLite Query Optimization
- https://www.sqlite.org/queryplanner.html
- https://www.sqlite.org/optoverview.html
- EXPLAIN QUERY PLAN documentation

### Benchmarking Best Practices
- https://golang.org/blog/benchmarking-go
- https://medium.com/google-cloud/optimization-patterns-in-go-613c5e5f14d8

---

## Appendix: Benchmark Commands Quick Reference

```bash
# Basic benchmarks
go test -bench=. ./internal/status

# With memory reporting
go test -bench=. -benchmem ./internal/status

# Specific benchmark
go test -bench=LargeProject -benchmem ./internal/status

# With CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/status

# With memory profile
go test -memprofile=mem.prof -bench=. ./internal/status

# With trace
go test -trace=trace.out ./internal/status

# Analyze profiles
go tool pprof cpu.prof
go tool pprof -http=:8080 mem.prof
go tool trace trace.out

# Compare benchmarks
benchstat baseline.txt after.txt

# View query plan
sqlite3 shark-tasks.db
sqlite> EXPLAIN QUERY PLAN SELECT ...;
```
