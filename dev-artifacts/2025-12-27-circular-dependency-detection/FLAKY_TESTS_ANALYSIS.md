# Flaky Test Analysis - Parallel Execution Failures

**Date**: 2025-12-27
**Issue**: Tests pass 100% when run sequentially (`go test -p=1 ./...`) but fail intermittently when run in parallel (`make test`)
**Root Cause**: Shared database state and key collisions in parallel test execution

---

## Categories of Flaky Tests

### Category 1: Tests Using Shared E99 Seed Data (9 tests)

These tests rely on `test.SeedTestData()` which creates E99 epic, E99-F99 feature, and T-E99-F99-* tasks. When tests run in parallel, some tests delete this shared data while others are trying to use it.

#### Tests Affected:

1. **TestBulkCreate** (`internal/repository/bulk_operations_test.go:119`)
   - Uses: `SeedTestData()` then queries `E04-F05` feature
   - Failure: `FOREIGN KEY constraint failed` when E04 epic deleted by another test
   - Pattern: Deletes nothing, expects E04-F05 to exist

2. **TestBulkCreateValidationFailure** (`internal/repository/bulk_operations_test.go:194`)
   - Uses: `SeedTestData()` then queries `E04-F05` feature
   - Failure: `FOREIGN KEY constraint failed` when E04 epic deleted
   - Pattern: Deletes nothing, expects E04-F05 to exist

3. **TestGetByKeys** (`internal/repository/bulk_operations_test.go:233`)
   - Uses: `SeedTestData()` then queries `T-E99-F99-001/002/003`
   - Failure: Returns fewer tasks than expected when E99 data deleted
   - Pattern: Deletes nothing, expects E99-F99 tasks to exist

4. **TestGetByKeysPartial** (`internal/repository/bulk_operations_test.go:255`)
   - Uses: Deletes E99 data, calls `SeedTestData()`, queries E99-F99 tasks
   - Failure: `FOREIGN KEY constraint failed` when another test deleted E99 between DELETE and SeedTestData
   - Pattern: Deletes E99 → Seeds → Uses

5. **TestUpdateMetadata** (`internal/repository/bulk_operations_test.go:274`)
   - Uses: Deletes E99 data, calls `SeedTestData()`, queries `T-E99-F99-001`
   - Failure: `FOREIGN KEY constraint failed` when E99 deleted by another test
   - Pattern: Deletes E99 → Seeds → Uses

6. **TestUpdateCompletionMetadata** (`internal/repository/task_completion_metadata_test.go:12`)
   - Uses: `SeedTestData()` then updates `T-E99-F99-001`
   - Failure: Task not found when E99 data deleted
   - Pattern: Seeds → Uses T-E99-F99-001

7. **TestLinkToTask** (`internal/repository/document_repository_test.go`)
   - Uses: `SeedTestData()` then queries E99 epic tasks
   - Failure: No tasks found when E99 deleted
   - Pattern: Seeds → Uses E99 tasks

8. **TestLinkToFeature** (`internal/repository/document_repository_test.go`)
   - Similar pattern to TestLinkToTask

9. **TestListForEpic** (`internal/repository/document_repository_test.go`)
   - Similar pattern to TestLinkToTask

**Common Pattern**: All these tests either:
- Call `SeedTestData()` and expect E04/E99 data to persist
- Delete E04/E99 data, call `SeedTestData()`, then use the data
- Race condition: Another test deletes the data between seeding and usage

---

### Category 2: Tests Using Shared E04 Seed Data (2 tests)

These tests specifically use E04 epic data created by `SeedTestData()`.

1. **TestBulkCreatePerformance** (`internal/repository/bulk_operations_test.go:488`)
   - Uses: Deletes E04/E99, calls `SeedTestData()`, creates 100 tasks in E04-F05
   - Failure: `FOREIGN KEY constraint failed` when E04 deleted by another test
   - Pattern: Deletes E04/E99 → Seeds → Bulk creates in E04-F05

2. **TestGetByKeysPerformance** (`internal/repository/bulk_operations_test.go:561`)
   - Uses: Deletes tasks in E04-F05, queries by keys
   - Failure: `FOREIGN KEY constraint failed` or query returns empty
   - Pattern: Deletes tasks → Creates → Queries

---

### Category 3: Tests Using Dynamic Epic Keys with Collisions (4 tests)

These tests generate "unique" epic keys using timestamp, but with only 50 possible values (E50-E99 or E90-E99), collisions occur in parallel execution.

1. **TestEpicDetailsIntegration** (`internal/repository/epic_feature_integration_test.go:118`)
   - Uses: `generateTestEpicKey()` → E50-E99 range (50 values)
   - Failure: Epic already exists or wrong data retrieved
   - Parallel tests: 200+ tests competing for 50 epic keys

2. **TestFeatureDetailsIntegration** (`internal/repository/epic_feature_integration_test.go:254`)
   - Uses: `generateTestEpicKey()` → E50-E99 range
   - Failure: Progress calculation wrong (70% expected, 0% actual)
   - Cause: Another test using same epic key modified the tasks

3. **TestProgressCalculationEdgeCases** (`internal/repository/epic_feature_integration_test.go:382`)
   - Uses: `generateTestEpicKey()` → E50-E99 range
   - Failure: Assertion failures on progress percentages
   - Cause: Test assumes exclusive access to epic

4. **TestEpicProgress_WeightedAverage** (`internal/repository/progress_calc_test.go:273`)
   - Uses: E97 epic (hardcoded) with two features E97-F01, E97-F02
   - Failure: Expected 75% progress, got 77.3%
   - Cause: Another test using E97 added/modified tasks

---

### Category 4: Status Dashboard Tests (3 tests)

These tests query the dashboard which aggregates ALL epics in the database. Parallel tests pollute the data.

1. **TestGetDashboard_EmptyDatabase** (`internal/status/status_test.go`)
   - Expects: Database to be empty
   - Failure: Returns epics created by other parallel tests
   - Issue: Cannot guarantee empty database in parallel execution

2. **TestGetDashboard_WithData** (`internal/status/status_test.go`)
   - Expects: Specific epics and counts
   - Failure: Counts include data from other tests
   - Issue: Dashboard aggregates ALL data

3. **TestGetDashboard_MultipleEpicsOrdering** (`internal/status/status_test.go`)
   - Expects: Specific epic order and counts
   - Failure: Different epics or ordering due to parallel test data
   - Issue: Dashboard sees all test data

---

## Root Cause Analysis

### Problem 1: Shared Test Data (E04, E99)
- `SeedTestData()` creates E04 and E99 epics used by multiple tests
- Tests delete this data as part of cleanup
- With parallel execution: Test A seeds → Test B deletes → Test A fails

### Problem 2: Limited Key Space
- Epic keys must match `^E\d{2}$` (E00-E99 = 100 possible values)
- Tests use ranges:
  - E10-E99: 90 values (general tests)
  - E50-E99: 50 values (integration tests)
  - E90-E99: 10 values (progress tests)
- 200+ tests competing for limited keys → inevitable collisions

### Problem 3: Global State
- Dashboard tests query entire database
- Cannot isolate test data in global queries
- Parallel tests pollute the global view

---

## Statistics

**Total Tests in Suite**: ~250 tests
**Flaky Tests Identified**: 18 tests (~7% failure rate)
**Failure Rate in Parallel**: ~10-20% (varies per run)
**Success Rate Sequential**: 100%

**Breakdown**:
- E99 shared data tests: 9 tests
- E04 shared data tests: 2 tests
- Epic key collision tests: 4 tests
- Dashboard global state tests: 3 tests

---

## Current Mitigation Attempts

1. ✓ Tests clean up data before running
2. ✓ `SeedTestData()` uses `INSERT OR IGNORE` for idempotency
3. ✓ `SeedTestData()` gracefully handles FK constraint errors
4. ✓ Timestamp-based key generation for uniqueness
5. ✓ `CREATE TABLE IF NOT EXISTS` for race-safe schema
6. ✗ Still failing due to fundamental architectural issues

---

## Recommendations for Strategy

### Option 1: Unique Database Per Test
- Create isolated SQLite database for each test
- Pros: Complete isolation, no collisions
- Cons: Slower tests, more complex setup

### Option 2: Namespace Test Data
- Use test-specific prefixes (test ID + random suffix)
- Example: Instead of E50, use E50-{test-name}-{random}
- Problem: Violates E\d{2} format constraint
- Would require: Relaxing epic key validation in test environment

### Option 3: Eliminate Shared Test Data
- Remove `SeedTestData()` entirely
- Each test creates its own isolated data
- Tests use unique epic keys (no E04/E99 sharing)
- Pros: True test isolation
- Cons: More verbose tests, duplicate setup code

### Option 4: Run Tests Sequentially
- Use `go test -p=1 ./...` in CI/CD
- Pros: Immediate fix, no code changes
- Cons: Slower CI, doesn't fix root cause

### Option 5: Test Package Isolation
- Run repository tests sequentially, others in parallel
- `go test -p=1 ./internal/repository && go test ./internal/...`
- Pros: Faster than full sequential
- Cons: Still slower, partial fix

---

## Decision Needed

Which strategy would you like to pursue? I recommend **Option 3** (eliminate shared test data) as the proper long-term solution, even though it requires refactoring tests to be more verbose.
