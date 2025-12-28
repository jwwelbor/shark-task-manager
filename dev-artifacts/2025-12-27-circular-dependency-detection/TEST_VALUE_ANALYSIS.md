# Test Value Analysis - What Are We Actually Testing?

## Question: Do these tests have real business logic to verify, or are they just testing SQLite?

---

## Tests That Test Real Business Logic (KEEP)

### 1. TestEpicProgress_WeightedAverage (`progress_calc_test.go:273`)
**What it tests**: Epic progress calculation uses weighted average based on task counts
**Business Logic**:
```
Epic with 2 features:
  - Feature 1: 50% complete with 10 tasks
  - Feature 2: 100% complete with 10 tasks
Epic progress = (50×10 + 100×10) / (10+10) = 75%
NOT simple average of (50+100)/2 = 75%
```
**Value**: ✅ **HIGH** - This is complex calculation logic that needs verification
**Verdict**: Keep this test, it validates important business logic

---

### 2. TestFeatureDetailsIntegration (`epic_feature_integration_test.go:254`)
**What it tests**: Feature progress calculation
**Business Logic**:
```
Feature with 10 tasks: 7 completed, 2 in_progress, 1 todo
Expected: 70% progress (only completed count)
```
**Value**: ✅ **HIGH** - Validates progress calculation formula
**Verdict**: Keep, but should be a unit test with mocked data, not integration test

---

### 3. TestProgressCalculationEdgeCases (`epic_feature_integration_test.go:382`)
**What it tests**: Edge cases in progress calculation
**Business Logic**:
- Feature with zero tasks = 0% progress
- Feature with all completed = 100% progress
**Value**: ✅ **MEDIUM** - Edge cases are important
**Verdict**: Keep, but convert to unit tests

---

### 4. TestBulkCreateValidationFailure (`bulk_operations_test.go:194`)
**What it tests**: BulkCreate rejects duplicate keys and rolls back transaction
**Business Logic**:
```go
tasks := []*models.Task{
    {Key: "T-E04-F05-200"},
    {Key: "T-E04-F05-200"}, // Duplicate!
}
err := taskRepo.BulkCreate(ctx, tasks)
// Should error and rollback (no tasks created)
```
**Value**: ✅ **HIGH** - Transaction rollback is critical business logic
**Verdict**: Keep this test - validates atomicity guarantee

---

### 5. TestGetByKeysPartial (`bulk_operations_test.go:255`)
**What it tests**: GetByKeys handles mix of valid and invalid keys gracefully
**Business Logic**:
```go
keys := []string{"T-E99-F99-001", "T-E99-F99-999", "T-E99-F99-002"}
//                 exists          doesn't exist   exists
tasks := repo.GetByKeys(ctx, keys)
// Should return 2 tasks, not error on missing key
```
**Value**: ✅ **MEDIUM** - Validates partial success behavior
**Verdict**: Keep - tests important error handling logic

---

### 6. TestUpdateMetadata (`bulk_operations_test.go:274`)
**What it tests**: UpdateMetadata ignores status/priority/agent changes (database-only fields)
**Business Logic**:
```go
task.Title = "Updated"
task.Status = TaskStatusCompleted  // Should be ignored
repo.UpdateMetadata(ctx, task)
// Title should update, Status should NOT
```
**Value**: ✅ **HIGH** - Validates field-level update protection
**Verdict**: Keep - this is important business rule

---

## Tests That Just Test SQLite CRUD (REMOVE OR SIMPLIFY)

### 7. TestBulkCreate (`bulk_operations_test.go:119`)
**What it tests**: BulkCreate can insert multiple tasks
**Business Logic**: None - just tests INSERT works
**Value**: ❌ **LOW** - SQLite works, we don't need to test this
**Verdict**: Remove or merge into TestBulkCreateValidationFailure

---

### 8. TestGetByKeys (`bulk_operations_test.go:233`)
**What it tests**: GetByKeys returns multiple tasks by their keys
**Business Logic**: None - just tests SELECT with IN clause
**Value**: ❌ **LOW** - Basic query, no logic to validate
**Verdict**: Remove - covered by other tests

---

### 9. TestUpdateCompletionMetadata (`task_completion_metadata_test.go:12`)
**What it tests**: Can store and retrieve completion metadata
**Business Logic**: None - CRUD operations
**Value**: ❌ **LOW** - Just testing JSON serialization + storage
**Verdict**: Remove or make it a unit test with mocks

---

### 10. TestLinkToTask/Feature/Epic (`document_repository_test.go`)
**What it tests**: Can link documents to tasks/features/epics
**Business Logic**: None - INSERT into junction table
**Value**: ❌ **LOW** - Basic relational database operation
**Verdict**: Remove - this is just testing SQLite foreign keys work

---

### 11. TestListForEpic (`document_repository_test.go`)
**What it tests**: Can query documents linked to an epic
**Business Logic**: None - SELECT with JOIN
**Value**: ❌ **LOW** - Standard SQL query
**Verdict**: Remove

---

### 12. TestEpicDetailsIntegration (`epic_feature_integration_test.go:118`)
**What it tests**: Can create epic and retrieve it with features
**Business Logic**: None - CRUD operations
**Value**: ❌ **LOW** - Just testing INSERT/SELECT work
**Verdict**: Remove - no business logic to validate

---

## Performance Tests (DEBATABLE)

### 13. TestBulkCreatePerformance (`bulk_operations_test.go:488`)
**What it tests**: BulkCreate of 100 tasks completes quickly
**Business Logic**: None - measures speed
**Value**: ⚠️ **DEBATABLE** - Performance regression detection
**Verdict**: Move to separate performance test suite, skip in normal runs

---

### 14. TestGetByKeysPerformance (`bulk_operations_test.go:561`)
**What it tests**: GetByKeys of 100 tasks completes quickly
**Business Logic**: None - measures speed
**Value**: ⚠️ **DEBATABLE** - Performance regression detection
**Verdict**: Move to separate performance test suite, skip in normal runs

---

## Dashboard Tests (SHOULD BE MOCKED)

### 15-17. TestGetDashboard_* (`status_test.go`)
**What it tests**: Dashboard aggregation queries
**Business Logic**: Yes - aggregates epics, features, tasks with counts
**Value**: ✅ **MEDIUM** - Tests aggregation logic
**Issue**: These are SERVICE layer tests using repository layer
**Verdict**: Keep the tests but **MOCK the repository** - don't use real database

---

## Summary

### Tests to KEEP (with fixes):
1. ✅ **TestEpicProgress_WeightedAverage** - Real calculation logic
2. ✅ **TestFeatureDetailsIntegration** - Progress calculation (convert to unit test)
3. ✅ **TestProgressCalculationEdgeCases** - Edge cases (convert to unit test)
4. ✅ **TestBulkCreateValidationFailure** - Transaction rollback logic
5. ✅ **TestGetByKeysPartial** - Partial success behavior
6. ✅ **TestUpdateMetadata** - Field-level update protection
7. ✅ **TestGetDashboard_*** (3 tests) - Aggregation logic (MOCK the repository)

**Total: 9 tests worth keeping**

### Tests to REMOVE (no business logic):
1. ❌ TestBulkCreate - Just tests INSERT
2. ❌ TestGetByKeys - Just tests SELECT
3. ❌ TestUpdateCompletionMetadata - Just tests JSON + CRUD
4. ❌ TestLinkToTask - Just tests INSERT into junction table
5. ❌ TestLinkToFeature - Just tests INSERT into junction table
6. ❌ TestListForEpic - Just tests SELECT with JOIN
7. ❌ TestEpicDetailsIntegration - Just tests CRUD

**Total: 7 tests to remove**

### Tests to MOVE to separate performance suite:
1. ⚠️ TestBulkCreatePerformance
2. ⚠️ TestGetByKeysPerformance

**Total: 2 tests to relocate**

---

## Recommendation

**Delete 7 tests** that provide no value (just testing SQLite works)
**Fix 9 tests** that have real business logic:
- 3 progress calculation tests: Convert to unit tests with mocked data
- 3 repository tests: Fix to use isolated test data
- 3 status tests: Convert to use mocked repositories

This reduces the flaky test count from 18 → 9, and ensures all remaining tests have actual business logic to validate.
